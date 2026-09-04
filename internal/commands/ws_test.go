package commands

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Stuchalin/weeek-cli/internal/api"
)

func TestWorkspaceCommandsGolden(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		args         []string
		method       string
		path         string
		requestBody  string
		responseBody string
		golden       string
	}{
		{
			name:         "workspace info",
			args:         []string{"ws", "info"},
			method:       http.MethodGet,
			path:         "/ws",
			responseBody: `{"success":true,"workspace":{"id":7,"title":"Test Workspace"}}`,
			golden:       "ws_info.golden",
		},
		{
			name:         "workspace members",
			args:         []string{"ws", "members"},
			method:       http.MethodGet,
			path:         "/ws/members",
			responseBody: `{"success":true,"members":[{"id":"member-1","email":"agent@example.com"}]}`,
			golden:       "ws_members.golden",
		},
		{
			name:         "tag list",
			args:         []string{"ws", "tag", "list"},
			method:       http.MethodGet,
			path:         "/ws/tags",
			responseBody: `{"success":true,"tags":[{"id":7,"title":"Backend","color":"#112233"}]}`,
			golden:       "ws_tag_list.golden",
		},
		{
			name:         "tag create",
			args:         []string{"ws", "tag", "create", "--name", "Backend"},
			method:       http.MethodPost,
			path:         "/ws/tags",
			requestBody:  `{"title":"Backend"}`,
			responseBody: `{"success":true,"tag":{"id":7,"title":"Backend","color":"#112233"}}`,
			golden:       "ws_tag_create.golden",
		},
		{
			name:         "tag update",
			args:         []string{"ws", "tag", "update", "7", "--name", "Platform", "--color", "#445566"},
			method:       http.MethodPut,
			path:         "/ws/tags/7",
			requestBody:  `{"title":"Platform","color":"#445566"}`,
			responseBody: `{"success":true}`,
			golden:       "ws_tag_update.golden",
		},
		{
			name:         "tag delete",
			args:         []string{"ws", "tag", "delete", "7"},
			method:       http.MethodDelete,
			path:         "/ws/tags/7",
			responseBody: `{"success":true}`,
			golden:       "ws_tag_delete.golden",
		},
		{
			name:         "attachment get",
			args:         []string{"ws", "attachment", "get", "file-1"},
			method:       http.MethodGet,
			path:         "/ws/attachments/file-1",
			responseBody: `{"success":true,"data":{"id":"file-1","name":"notes.txt","url":"https://example.test/notes.txt"}}`,
			golden:       "ws_attachment_get.golden",
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
				body, err := io.ReadAll(request.Body)
				if err != nil {
					t.Errorf("read request body: %v", err)
				}
				if string(body) != tt.requestBody {
					t.Errorf("request body = %q, want %q", body, tt.requestBody)
				}
				if _, err := response.Write([]byte(tt.responseBody)); err != nil {
					t.Errorf("write response body: %v", err)
				}
			}))
			defer server.Close()

			var stdout bytes.Buffer
			var stderr bytes.Buffer
			registry := NewRegistry("test-version")
			code := registry.Run(
				tt.args,
				&Ctx{
					Client: api.NewClient(server.URL, "test-token", server.Client()),
					Stdout: &stdout,
					Stderr: &stderr,
				},
			)
			if code != 0 {
				t.Fatalf("command code = %d, want 0; stderr = %q", code, stderr.String())
			}
			assertGolden(t, tt.golden, stdout.Bytes())
		})
	}
}

func TestWorkspaceTagNotFound(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodDelete || request.URL.Path != "/ws/tags/404" {
			t.Errorf("request = %s %s, want DELETE /ws/tags/404", request.Method, request.URL.Path)
		}
		response.WriteHeader(http.StatusNotFound)
		if _, err := response.Write([]byte(`{"message":"tag not found"}`)); err != nil {
			t.Errorf("write response body: %v", err)
		}
	}))
	defer server.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	registry := NewRegistry("test-version")
	code := registry.Run(
		[]string{"ws", "tag", "delete", "404"},
		&Ctx{
			Client: api.NewClient(server.URL, "test-token", server.Client()),
			Stdout: &stdout,
			Stderr: &stderr,
		},
	)
	if code != 1 {
		t.Errorf("ws tag delete code = %d, want 1", code)
	}
	const want = `{"error":"api request failed with status 404: tag not found","status":404}` + "\n"
	if got := stderr.String(); got != want {
		t.Errorf("ws tag delete stderr = %q, want %q", got, want)
	}
}

func TestWorkspaceTagFlags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
	}{
		{
			name: "create requires name",
			args: []string{"ws", "tag", "create"},
		},
		{
			name: "update requires id",
			args: []string{"ws", "tag", "update", "--name", "Backend", "--color", "#112233"},
		},
		{
			name: "update requires color",
			args: []string{"ws", "tag", "update", "7", "--name", "Backend"},
		},
		{
			name: "attachment requires id",
			args: []string{"ws", "attachment", "get"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var stderr bytes.Buffer
			code := NewRegistry("test-version").Run(tt.args, &Ctx{Stderr: &stderr})
			if code != 2 {
				t.Errorf("command code = %d, want 2", code)
			}
			if stderr.Len() == 0 {
				t.Error("command stderr is empty, want usage")
			}
		})
	}
}
