package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClient_WorkspaceMethods(t *testing.T) {
	t.Parallel()

	const responseBody = `{"success":true}`
	tests := []struct {
		name        string
		method      string
		path        string
		escapedPath string
		body        string
		call        func(context.Context, *Client) (json.RawMessage, error)
	}{
		{
			name:   "get workspace",
			method: http.MethodGet,
			path:   "/ws",
			call: func(ctx context.Context, client *Client) (json.RawMessage, error) {
				return client.GetWorkspace(ctx)
			},
		},
		{
			name:   "get members",
			method: http.MethodGet,
			path:   "/ws/members",
			call: func(ctx context.Context, client *Client) (json.RawMessage, error) {
				return client.GetMembers(ctx)
			},
		},
		{
			name:   "list tags",
			method: http.MethodGet,
			path:   "/ws/tags",
			call: func(ctx context.Context, client *Client) (json.RawMessage, error) {
				return client.ListTags(ctx)
			},
		},
		{
			name:   "create tag",
			method: http.MethodPost,
			path:   "/ws/tags",
			body:   `{"title":"Backend"}`,
			call: func(ctx context.Context, client *Client) (json.RawMessage, error) {
				return client.CreateTag(ctx, TagCreate{Title: "Backend"})
			},
		},
		{
			name:   "update tag",
			method: http.MethodPut,
			path:   "/ws/tags/7",
			body:   `{"title":"Backend","color":"#112233"}`,
			call: func(ctx context.Context, client *Client) (json.RawMessage, error) {
				return client.UpdateTag(
					ctx,
					"7",
					TagUpdate{Title: "Backend", Color: "#112233"},
				)
			},
		},
		{
			name:   "delete tag",
			method: http.MethodDelete,
			path:   "/ws/tags/7",
			call: func(ctx context.Context, client *Client) (json.RawMessage, error) {
				return client.DeleteTag(ctx, "7")
			},
		},
		{
			name:        "get attachment",
			method:      http.MethodGet,
			path:        "/ws/attachments/file/one",
			escapedPath: "/ws/attachments/file%2Fone",
			call: func(ctx context.Context, client *Client) (json.RawMessage, error) {
				return client.GetAttachment(ctx, "file/one")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
				if request.Method != tt.method {
					t.Errorf("request method = %q, want %q", request.Method, tt.method)
				}
				if request.URL.Path != tt.path {
					t.Errorf("request path = %q, want %q", request.URL.Path, tt.path)
				}
				if tt.escapedPath != "" && request.URL.EscapedPath() != tt.escapedPath {
					t.Errorf("escaped request path = %q, want %q", request.URL.EscapedPath(), tt.escapedPath)
				}
				if request.URL.RawQuery != "" {
					t.Errorf("request query = %q, want empty", request.URL.RawQuery)
				}
				body, err := io.ReadAll(request.Body)
				if err != nil {
					t.Errorf("read request body: %v", err)
				}
				if string(body) != tt.body {
					t.Errorf("request body = %q, want %q", body, tt.body)
				}
				if tt.body != "" && request.Header.Get("Content-Type") != "application/json" {
					t.Errorf("Content-Type = %q, want application/json", request.Header.Get("Content-Type"))
				}
				if _, err := response.Write([]byte(responseBody)); err != nil {
					t.Errorf("write response body: %v", err)
				}
			}))
			defer server.Close()

			client := NewClient(server.URL, "test-token", server.Client())
			got, err := tt.call(t.Context(), client)
			if err != nil {
				t.Fatalf("workspace method error = %v", err)
			}
			if string(got) != responseBody {
				t.Errorf("workspace method response = %q, want %q", got, responseBody)
			}
		})
	}
}
