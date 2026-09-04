package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestClient_BoardMethods(t *testing.T) {
	t.Parallel()

	upperBoardID := int64(8)
	upperColumnID := int64(18)
	boardID := int64(12)
	tests := []struct {
		name        string
		method      string
		path        string
		escapedPath string
		query       url.Values
		body        string
		call        func(context.Context, *Client) (json.RawMessage, error)
	}{
		{
			name:   "list boards",
			method: http.MethodGet,
			path:   "/tm/boards",
			query:  url.Values{"projectId": {"7"}},
			call: func(ctx context.Context, client *Client) (json.RawMessage, error) {
				return client.ListBoards(ctx, 7)
			},
		},
		{
			name:   "create board",
			method: http.MethodPost,
			path:   "/tm/boards",
			body:   `{"name":"Delivery","projectId":7}`,
			call: func(ctx context.Context, client *Client) (json.RawMessage, error) {
				return client.CreateBoard(ctx, BoardCreate{Name: "Delivery", ProjectID: 7})
			},
		},
		{
			name:        "update board",
			method:      http.MethodPut,
			path:        "/tm/boards/board/one",
			escapedPath: "/tm/boards/board%2Fone",
			body:        `{"name":"Shipping"}`,
			call: func(ctx context.Context, client *Client) (json.RawMessage, error) {
				return client.UpdateBoard(ctx, "board/one", BoardUpdate{Name: "Shipping"})
			},
		},
		{
			name:   "delete board",
			method: http.MethodDelete,
			path:   "/tm/boards/9",
			call: func(ctx context.Context, client *Client) (json.RawMessage, error) {
				return client.DeleteBoard(ctx, "9")
			},
		},
		{
			name:   "move board after another board",
			method: http.MethodPost,
			path:   "/tm/boards/9/move",
			body:   `{"upperBoardId":8}`,
			call: func(ctx context.Context, client *Client) (json.RawMessage, error) {
				return client.MoveBoard(ctx, "9", BoardMove{UpperBoardID: &upperBoardID})
			},
		},
		{
			name:   "move board to top",
			method: http.MethodPost,
			path:   "/tm/boards/9/move",
			body:   `{"upperBoardId":null}`,
			call: func(ctx context.Context, client *Client) (json.RawMessage, error) {
				return client.MoveBoard(ctx, "9", BoardMove{})
			},
		},
		{
			name:   "list board columns",
			method: http.MethodGet,
			path:   "/tm/board-columns",
			query:  url.Values{"boardId": {"12"}},
			call: func(ctx context.Context, client *Client) (json.RawMessage, error) {
				return client.ListBoardColumns(ctx, &boardID)
			},
		},
		{
			name:   "list all board columns",
			method: http.MethodGet,
			path:   "/tm/board-columns",
			call: func(ctx context.Context, client *Client) (json.RawMessage, error) {
				return client.ListBoardColumns(ctx, nil)
			},
		},
		{
			name:   "create board column",
			method: http.MethodPost,
			path:   "/tm/board-columns",
			body:   `{"name":"In progress","boardId":12}`,
			call: func(ctx context.Context, client *Client) (json.RawMessage, error) {
				return client.CreateBoardColumn(ctx, BoardColumnCreate{
					Name:    "In progress",
					BoardID: 12,
				})
			},
		},
		{
			name:   "update board column",
			method: http.MethodPut,
			path:   "/tm/board-columns/19",
			body:   `{"name":"Review"}`,
			call: func(ctx context.Context, client *Client) (json.RawMessage, error) {
				return client.UpdateBoardColumn(ctx, "19", BoardColumnUpdate{Name: "Review"})
			},
		},
		{
			name:   "delete board column",
			method: http.MethodDelete,
			path:   "/tm/board-columns/19",
			call: func(ctx context.Context, client *Client) (json.RawMessage, error) {
				return client.DeleteBoardColumn(ctx, "19")
			},
		},
		{
			name:   "move board column after another column",
			method: http.MethodPost,
			path:   "/tm/board-columns/19/move",
			body:   `{"upperBoardColumnId":18}`,
			call: func(ctx context.Context, client *Client) (json.RawMessage, error) {
				return client.MoveBoardColumn(
					ctx,
					"19",
					BoardColumnMove{UpperBoardColumnID: &upperColumnID},
				)
			},
		},
		{
			name:   "move board column to top",
			method: http.MethodPost,
			path:   "/tm/board-columns/19/move",
			body:   `{"upperBoardColumnId":null}`,
			call: func(ctx context.Context, client *Client) (json.RawMessage, error) {
				return client.MoveBoardColumn(ctx, "19", BoardColumnMove{})
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
				if got := request.URL.Query(); !queryValuesEqual(got, tt.query) {
					t.Errorf("request query = %v, want %v", got, tt.query)
				}
				body, err := io.ReadAll(request.Body)
				if err != nil {
					t.Errorf("read request body: %v", err)
				}
				if string(body) != tt.body {
					t.Errorf("request body = %q, want %q", body, tt.body)
				}
				_, _ = response.Write([]byte(`{"success":true}`))
			}))
			defer server.Close()

			client := NewClient(server.URL, "test-token", server.Client())
			got, err := tt.call(t.Context(), client)
			if err != nil {
				t.Fatalf("board method error = %v", err)
			}
			if string(got) != `{"success":true}` {
				t.Errorf("board method response = %q", got)
			}
		})
	}
}

func TestClient_GetBoard(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet || request.URL.Path != "/tm/boards" {
			t.Errorf("request = %s %s, want GET /tm/boards", request.Method, request.URL.Path)
		}
		if got := request.URL.Query().Get("projectId"); got != "7" {
			t.Errorf("projectId = %q, want 7", got)
		}
		_, _ = response.Write([]byte(
			`{"success":true,"boards":[{"id":8,"name":"Planning"},` +
				`{"id":9,"name":"Delivery","extra":{"kept":true}}]}`,
		))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token", server.Client())
	got, err := client.GetBoard(t.Context(), 7, "9")
	if err != nil {
		t.Fatalf("GetBoard() error = %v", err)
	}
	const want = `{"success":true,"board":{"id":9,"name":"Delivery","extra":{"kept":true}}}`
	if string(got) != want {
		t.Errorf("GetBoard() response = %q, want %q", got, want)
	}
}

func TestClient_GetBoardNotFound(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		_, _ = response.Write([]byte(`{"success":true,"boards":[]}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token", server.Client())
	_, err := client.GetBoard(t.Context(), 7, "404")
	if err == nil || err.Error() != "board not found" {
		t.Errorf("GetBoard() error = %v, want board not found", err)
	}
}
