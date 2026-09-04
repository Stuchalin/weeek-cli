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

func TestClient_ProjectAndPortfolioMethods(t *testing.T) {
	t.Parallel()

	logo := "logo.png"
	description := "Launch roadmap"
	emptyLogo := ""
	color := "#00AAFF"
	portfolioID := int64(4)
	parentID := int64(3)
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
			name:   "list projects",
			method: http.MethodGet,
			path:   "/tm/projects",
			call: func(ctx context.Context, client *Client) (json.RawMessage, error) {
				return client.ListProjects(ctx)
			},
		},
		{
			name:        "get project",
			method:      http.MethodGet,
			path:        "/tm/projects/project/one",
			escapedPath: "/tm/projects/project%2Fone",
			call: func(ctx context.Context, client *Client) (json.RawMessage, error) {
				return client.GetProject(ctx, "project/one")
			},
		},
		{
			name:   "create project",
			method: http.MethodPost,
			path:   "/tm/projects",
			body: `{"name":"Roadmap","isPrivate":1,"logo":"logo.png",` +
				`"description":"Launch roadmap","portfolioId":4}`,
			call: func(ctx context.Context, client *Client) (json.RawMessage, error) {
				return client.CreateProject(ctx, ProjectCreate{
					Name:        "Roadmap",
					IsPrivate:   1,
					Logo:        &logo,
					Description: &description,
					PortfolioID: &portfolioID,
				})
			},
		},
		{
			name:   "update project",
			method: http.MethodPut,
			path:   "/tm/projects/8",
			body:   `{"name":"Roadmap v2","isPrivate":0,"logo":"","color":"#00AAFF"}`,
			call: func(ctx context.Context, client *Client) (json.RawMessage, error) {
				return client.UpdateProject(ctx, "8", ProjectUpdate{
					Name:      "Roadmap v2",
					IsPrivate: 0,
					Logo:      &emptyLogo,
					Color:     &color,
				})
			},
		},
		{
			name:   "delete project",
			method: http.MethodDelete,
			path:   "/tm/projects/8",
			call: func(ctx context.Context, client *Client) (json.RawMessage, error) {
				return client.DeleteProject(ctx, "8")
			},
		},
		{
			name:   "archive project",
			method: http.MethodPost,
			path:   "/tm/projects/8/archive",
			call: func(ctx context.Context, client *Client) (json.RawMessage, error) {
				return client.ArchiveProject(ctx, "8")
			},
		},
		{
			name:   "unarchive project",
			method: http.MethodPost,
			path:   "/tm/projects/8/un-archive",
			call: func(ctx context.Context, client *Client) (json.RawMessage, error) {
				return client.UnarchiveProject(ctx, "8")
			},
		},
		{
			name:   "list portfolios",
			method: http.MethodGet,
			path:   "/tm/portfolios",
			query: url.Values{
				"search":   {"roadmap"},
				"parentId": {"3"},
				"limit":    {"20"},
				"offset":   {"5"},
			},
			call: func(ctx context.Context, client *Client) (json.RawMessage, error) {
				return client.ListPortfolios(ctx, PortfolioFilters{
					Search:   "roadmap",
					ParentID: &parentID,
					Limit:    20,
					Offset:   5,
				})
			},
		},
		{
			name:   "get portfolio",
			method: http.MethodGet,
			path:   "/tm/portfolios/4",
			call: func(ctx context.Context, client *Client) (json.RawMessage, error) {
				return client.GetPortfolio(ctx, "4")
			},
		},
		{
			name:   "create portfolio",
			method: http.MethodPost,
			path:   "/tm/portfolios",
			body:   `{"name":"Delivery","parentId":3}`,
			call: func(ctx context.Context, client *Client) (json.RawMessage, error) {
				return client.CreatePortfolio(ctx, PortfolioCreate{
					Name:     "Delivery",
					ParentID: &parentID,
				})
			},
		},
		{
			name:   "update portfolio",
			method: http.MethodPut,
			path:   "/tm/portfolios/4",
			body:   `{"name":"Delivery v2"}`,
			call: func(ctx context.Context, client *Client) (json.RawMessage, error) {
				return client.UpdatePortfolio(ctx, "4", PortfolioUpdate{Name: "Delivery v2"})
			},
		},
		{
			name:   "delete portfolio",
			method: http.MethodDelete,
			path:   "/tm/portfolios/4",
			call: func(ctx context.Context, client *Client) (json.RawMessage, error) {
				return client.DeletePortfolio(ctx, "4")
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
				t.Fatalf("project or portfolio method error = %v", err)
			}
			if string(got) != `{"success":true}` {
				t.Errorf("project or portfolio method response = %q", got)
			}
		})
	}
}

func TestClient_ListPortfoliosWithoutFilters(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.RawQuery != "" {
			t.Errorf("request query = %q, want empty", request.URL.RawQuery)
		}
		_, _ = response.Write([]byte(`{"success":true,"data":[],"hasMore":false}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token", server.Client())
	if _, err := client.ListPortfolios(t.Context(), PortfolioFilters{}); err != nil {
		t.Fatalf("ListPortfolios() error = %v", err)
	}
}
