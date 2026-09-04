package commands

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Stuchalin/weeek-cli/internal/api"
)

func TestBoardCommands(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		args           []string
		method         string
		path           string
		query          string
		requestBody    string
		responseBody   string
		expectedOutput string
		golden         string
	}{
		{
			name:         "board list",
			args:         []string{"board", "list", "--project", "7"},
			method:       http.MethodGet,
			path:         "/tm/boards",
			query:        "projectId=7",
			responseBody: `{"success":true,"boards":[{"id":9,"name":"Delivery","projectId":7,"isPrivate":false}]}`,
			golden:       "board_list.golden",
		},
		{
			name:         "board get",
			args:         []string{"board", "get", "9", "--project", "7"},
			method:       http.MethodGet,
			path:         "/tm/boards",
			query:        "projectId=7",
			responseBody: `{"success":true,"boards":[{"id":8,"name":"Planning"},{"id":9,"name":"Delivery","projectId":7}]}`,
			golden:       "board_get.golden",
		},
		{
			name:         "board create",
			args:         []string{"board", "create", "--name", "Delivery", "--project", "7"},
			method:       http.MethodPost,
			path:         "/tm/boards",
			requestBody:  `{"name":"Delivery","projectId":7}`,
			responseBody: `{"success":true,"board":{"id":9,"name":"Delivery","projectId":7,"isPrivate":false}}`,
			golden:       "board_create.golden",
		},
		{
			name:           "board update",
			args:           []string{"board", "update", "9", "--name", "Shipping"},
			method:         http.MethodPut,
			path:           "/tm/boards/9",
			requestBody:    `{"name":"Shipping"}`,
			responseBody:   `{"success":true}`,
			expectedOutput: `{"success":true}` + "\n",
		},
		{
			name:           "board delete",
			args:           []string{"board", "delete", "9"},
			method:         http.MethodDelete,
			path:           "/tm/boards/9",
			responseBody:   `{"success":true}`,
			expectedOutput: `{"success":true}` + "\n",
		},
		{
			name:           "board move",
			args:           []string{"board", "move", "9", "--after", "8"},
			method:         http.MethodPost,
			path:           "/tm/boards/9/move",
			requestBody:    `{"upperBoardId":8}`,
			responseBody:   `{}`,
			expectedOutput: `{}` + "\n",
		},
		{
			name:         "column list",
			args:         []string{"column", "list", "--board", "9"},
			method:       http.MethodGet,
			path:         "/tm/board-columns",
			query:        "boardId=9",
			responseBody: `{"success":true,"boardColumns":[{"id":19,"name":"In progress","boardId":9}]}`,
			golden:       "column_list.golden",
		},
		{
			name:         "column create",
			args:         []string{"column", "create", "--name", "In progress", "--board", "9"},
			method:       http.MethodPost,
			path:         "/tm/board-columns",
			requestBody:  `{"name":"In progress","boardId":9}`,
			responseBody: `{"success":true,"boardColumn":{"id":19,"name":"In progress","boardId":9}}`,
			golden:       "column_create.golden",
		},
		{
			name:           "column update",
			args:           []string{"column", "update", "19", "--name", "Review"},
			method:         http.MethodPut,
			path:           "/tm/board-columns/19",
			requestBody:    `{"name":"Review"}`,
			responseBody:   `{"success":true}`,
			expectedOutput: `{"success":true}` + "\n",
		},
		{
			name:           "column delete",
			args:           []string{"column", "delete", "19"},
			method:         http.MethodDelete,
			path:           "/tm/board-columns/19",
			responseBody:   `{"success":true}`,
			expectedOutput: `{"success":true}` + "\n",
		},
		{
			name:           "column move to top",
			args:           []string{"column", "move", "19"},
			method:         http.MethodPost,
			path:           "/tm/board-columns/19/move",
			requestBody:    `{"upperBoardColumnId":null}`,
			responseBody:   `{"success":true}`,
			expectedOutput: `{"success":true}` + "\n",
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
				if request.URL.RawQuery != tt.query {
					t.Errorf("request query = %q, want %q", request.URL.RawQuery, tt.query)
				}
				body, err := io.ReadAll(request.Body)
				if err != nil {
					t.Errorf("read request body: %v", err)
				}
				if string(body) != tt.requestBody {
					t.Errorf("request body = %q, want %q", body, tt.requestBody)
				}
				_, _ = response.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			var stdout bytes.Buffer
			var stderr bytes.Buffer
			code := NewRegistry("test-version").Run(tt.args, &Ctx{
				Client: api.NewClient(server.URL, "test-token", server.Client()),
				Stdout: &stdout,
				Stderr: &stderr,
			})
			if code != 0 {
				t.Fatalf("command code = %d, want 0; stderr = %q", code, stderr.String())
			}
			if tt.golden != "" {
				assertGolden(t, tt.golden, stdout.Bytes())
				return
			}
			if got := stdout.String(); got != tt.expectedOutput {
				t.Errorf("command stdout = %q, want %q", got, tt.expectedOutput)
			}
		})
	}
}

func TestBoardColumnNotFound(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodDelete || request.URL.Path != "/tm/board-columns/404" {
			t.Errorf("request = %s %s, want DELETE /tm/board-columns/404", request.Method, request.URL.Path)
		}
		response.WriteHeader(http.StatusNotFound)
		_, _ = response.Write([]byte(`{"message":"board column not found"}`))
	}))
	defer server.Close()

	var stderr bytes.Buffer
	code := NewRegistry("test-version").Run(
		[]string{"column", "delete", "404"},
		&Ctx{
			Client: api.NewClient(server.URL, "test-token", server.Client()),
			Stderr: &stderr,
		},
	)
	if code != 1 {
		t.Errorf("column delete code = %d, want 1", code)
	}
	const want = `{"error":"api request failed with status 404: board column not found","status":404}` + "\n"
	if got := stderr.String(); got != want {
		t.Errorf("column delete stderr = %q, want %q", got, want)
	}
}

func TestBoardCommandFlags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
	}{
		{name: "board list requires project", args: []string{"board", "list"}},
		{name: "board get requires id", args: []string{"board", "get", "--project", "7"}},
		{name: "board create requires name", args: []string{"board", "create", "--project", "7"}},
		{name: "board update requires name", args: []string{"board", "update", "9"}},
		{name: "board move rejects nonpositive after", args: []string{"board", "move", "9", "--after", "0"}},
		{name: "column list rejects nonpositive board", args: []string{"column", "list", "--board", "0"}},
		{name: "column create requires board", args: []string{"column", "create", "--name", "Review"}},
		{name: "column update requires id", args: []string{"column", "update", "--name", "Review"}},
		{name: "column delete rejects extra id", args: []string{"column", "delete", "19", "20"}},
		{name: "column move rejects bad flag", args: []string{"column", "move", "19", "--unknown"}},
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
