package commands

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Stuchalin/weeek-cli/internal/api"
)

func TestProjectAndPortfolioCommands(t *testing.T) {
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
			name:         "project list",
			args:         []string{"project", "list"},
			method:       http.MethodGet,
			path:         "/tm/projects",
			responseBody: `{"success":true,"projects":[{"id":8,"name":"Roadmap","isPrivate":true}]}`,
			golden:       "project_list.golden",
		},
		{
			name:         "project get",
			args:         []string{"project", "get", "8"},
			method:       http.MethodGet,
			path:         "/tm/projects/8",
			responseBody: `{"success":true,"project":{"id":8,"name":"Roadmap","color":"#00AAFF"}}`,
			golden:       "project_get.golden",
		},
		{
			name: "project create",
			args: []string{
				"project", "create", "--name", "Roadmap", "--private", "1",
				"--logo", "logo.png", "--description", "Launch", "--portfolio", "4",
			},
			method:      http.MethodPost,
			path:        "/tm/projects",
			requestBody: `{"name":"Roadmap","isPrivate":1,"logo":"logo.png","description":"Launch","portfolioId":4}`,
			responseBody: `{"success":true,"project":{"id":8,"name":"Roadmap",` +
				`"description":"Launch","isPrivate":true,"portfolioId":4}}`,
			golden: "project_create.golden",
		},
		{
			name:           "project update",
			args:           []string{"project", "update", "8", "--name", "Launch", "--private", "0", "--color", "red"},
			method:         http.MethodPut,
			path:           "/tm/projects/8",
			requestBody:    `{"name":"Launch","isPrivate":0,"color":"red"}`,
			responseBody:   `{"success":true}`,
			expectedOutput: `{"success":true}` + "\n",
		},
		{
			name:           "project delete",
			args:           []string{"project", "delete", "8"},
			method:         http.MethodDelete,
			path:           "/tm/projects/8",
			responseBody:   `{"success":true}`,
			expectedOutput: `{"success":true}` + "\n",
		},
		{
			name:           "project archive",
			args:           []string{"project", "archive", "8"},
			method:         http.MethodPost,
			path:           "/tm/projects/8/archive",
			responseBody:   `{"success":true}`,
			expectedOutput: `{"success":true}` + "\n",
		},
		{
			name:           "project unarchive",
			args:           []string{"project", "unarchive", "8"},
			method:         http.MethodPost,
			path:           "/tm/projects/8/un-archive",
			responseBody:   `{"success":true}`,
			expectedOutput: `{"success":true}` + "\n",
		},
		{
			name: "portfolio list",
			args: []string{
				"portfolio", "list", "--search", "roadmap", "--parent", "3",
				"--limit", "20", "--offset", "5",
			},
			method:       http.MethodGet,
			path:         "/tm/portfolios",
			query:        "limit=20&offset=5&parentId=3&search=roadmap",
			responseBody: `{"success":true,"data":[{"id":4,"name":"Delivery","parentId":3}],"hasMore":false}`,
			golden:       "portfolio_list.golden",
		},
		{
			name:         "portfolio get",
			args:         []string{"portfolio", "get", "4"},
			method:       http.MethodGet,
			path:         "/tm/portfolios/4",
			responseBody: `{"success":true,"data":{"id":4,"name":"Delivery","isPrivate":false}}`,
			golden:       "portfolio_get.golden",
		},
		{
			name:         "portfolio create",
			args:         []string{"portfolio", "create", "--name", "Delivery", "--parent", "3"},
			method:       http.MethodPost,
			path:         "/tm/portfolios",
			requestBody:  `{"name":"Delivery","parentId":3}`,
			responseBody: `{"success":true,"data":{"id":4,"name":"Delivery","parentId":3}}`,
			golden:       "portfolio_create.golden",
		},
		{
			name:           "portfolio update",
			args:           []string{"portfolio", "update", "4", "--name", "Delivery v2"},
			method:         http.MethodPut,
			path:           "/tm/portfolios/4",
			requestBody:    `{"name":"Delivery v2"}`,
			responseBody:   `{"success":true}`,
			expectedOutput: `{"success":true}` + "\n",
		},
		{
			name:           "portfolio delete",
			args:           []string{"portfolio", "delete", "4"},
			method:         http.MethodDelete,
			path:           "/tm/portfolios/4",
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

func TestProjectAndPortfolioAPIErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		args       []string
		status     int
		message    string
		wantStatus string
	}{
		{
			name:       "project not found",
			args:       []string{"project", "get", "404"},
			status:     http.StatusNotFound,
			message:    "project not found",
			wantStatus: "404",
		},
		{
			name:       "portfolio unauthorized",
			args:       []string{"portfolio", "list"},
			status:     http.StatusUnauthorized,
			message:    "unauthorized",
			wantStatus: "401",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
				response.WriteHeader(tt.status)
				_, _ = response.Write([]byte(`{"message":"` + tt.message + `"}`))
			}))
			defer server.Close()

			var stderr bytes.Buffer
			code := NewRegistry("test-version").Run(tt.args, &Ctx{
				Client: api.NewClient(server.URL, "test-token", server.Client()),
				Stderr: &stderr,
			})
			if code != 1 {
				t.Errorf("command code = %d, want 1", code)
			}
			want := `{"error":"api request failed with status ` + tt.wantStatus + `: ` + tt.message +
				`","status":` + tt.wantStatus + `}` + "\n"
			if got := stderr.String(); got != want {
				t.Errorf("command stderr = %q, want %q", got, want)
			}
		})
	}
}

func TestProjectAndPortfolioCommandFlags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
	}{
		{name: "project list rejects arguments", args: []string{"project", "list", "extra"}},
		{name: "project get requires id", args: []string{"project", "get"}},
		{name: "project create requires name", args: []string{"project", "create", "--private", "1"}},
		{name: "project create requires private", args: []string{"project", "create", "--name", "Roadmap"}},
		{
			name: "project create rejects invalid private",
			args: []string{"project", "create", "--name", "Roadmap", "--private", "2"},
		},
		{
			name: "project create rejects invalid portfolio",
			args: []string{
				"project", "create", "--name", "Roadmap", "--private", "0", "--portfolio", "0",
			},
		},
		{name: "project update requires id", args: []string{"project", "update", "--name", "Roadmap", "--private", "0"}},
		{name: "project archive rejects extra id", args: []string{"project", "archive", "8", "9"}},
		{name: "portfolio list rejects invalid parent", args: []string{"portfolio", "list", "--parent", "0"}},
		{name: "portfolio list rejects excessive limit", args: []string{"portfolio", "list", "--limit", "101"}},
		{name: "portfolio create requires name", args: []string{"portfolio", "create"}},
		{name: "portfolio update requires name", args: []string{"portfolio", "update", "4"}},
		{name: "portfolio delete rejects bad flag", args: []string{"portfolio", "delete", "4", "--unknown"}},
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
