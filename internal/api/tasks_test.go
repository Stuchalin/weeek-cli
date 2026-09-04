package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestClient_ListTasks(t *testing.T) {
	t.Parallel()

	priority := 0
	completed := false
	all := true
	wantQuery := url.Values{
		"all":       {"1"},
		"completed": {"0"},
		"day":       {"04.09.2026"},
		"offset":    {"5"},
		"perPage":   {"25"},
		"priority":  {"0"},
		"search":    {"release notes"},
		"sortBy":    {"-created"},
		"tags":      {"7", "9"},
		"type":      {"action"},
		"userId":    {"user-42"},
	}
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet {
			t.Errorf("request method = %q, want %q", request.Method, http.MethodGet)
		}
		if request.URL.Path != "/tm/tasks" {
			t.Errorf("request path = %q, want %q", request.URL.Path, "/tm/tasks")
		}
		if got := request.URL.Query(); !queryValuesEqual(got, wantQuery) {
			t.Errorf("request query = %v, want %v", got, wantQuery)
		}
		_, _ = response.Write([]byte(`{"success":true,"tasks":[]}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token", server.Client())
	got, err := client.ListTasks(t.Context(), TaskFilters{
		Responsible: "user-42",
		Search:      "release notes",
		Priority:    &priority,
		Tags:        []string{"7", "9"},
		Type:        "action",
		Completed:   &completed,
		Day:         "04.09.2026",
		All:         &all,
		Sort:        "-created",
		Limit:       25,
		Offset:      5,
	})
	if err != nil {
		t.Fatalf("ListTasks() error = %v", err)
	}
	if string(got) != `{"success":true,"tasks":[]}` {
		t.Errorf("ListTasks() response = %q", got)
	}
}

func TestClient_ListTasksOmitsEmptyFilters(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.RawQuery != "" {
			t.Errorf("request query = %q, want empty", request.URL.RawQuery)
		}
		_, _ = response.Write([]byte(`{"success":true,"tasks":[]}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token", server.Client())
	if _, err := client.ListTasks(t.Context(), TaskFilters{}); err != nil {
		t.Fatalf("ListTasks() error = %v", err)
	}
}

func TestClient_TaskMethods(t *testing.T) {
	t.Parallel()

	description := "Detailed task"
	projectID := int64(11)
	columnID := int64(13)
	responsibleID := "user-42"
	taskType := "action"
	priority := 0
	day := "2026-09-05"
	dueDate := "2026-09-06"
	parentID := int64(9)
	tags := []int64{4, 5}
	title := "Updated task"

	tests := []struct {
		name        string
		method      string
		path        string
		escapedPath string
		body        string
		call        func(context.Context, *Client) (json.RawMessage, error)
	}{
		{
			name:        "get task",
			method:      http.MethodGet,
			path:        "/tm/tasks/task/one",
			escapedPath: "/tm/tasks/task%2Fone",
			call: func(ctx context.Context, client *Client) (json.RawMessage, error) {
				return client.GetTask(ctx, "task/one")
			},
		},
		{
			name:   "create task",
			method: http.MethodPost,
			path:   "/tm/tasks",
			body: `{"title":"New task","description":"Detailed task","day":"2026-09-05",` +
				`"locations":[{"projectId":11,"boardColumnId":13}],"userId":"user-42",` +
				`"type":"action","priority":0,"parentId":9}`,
			call: func(ctx context.Context, client *Client) (json.RawMessage, error) {
				return client.CreateTask(ctx, TaskCreate{
					Title:       "New task",
					Description: &description,
					Day:         &day,
					Locations: []TaskCreateLocation{{
						ProjectID:     projectID,
						BoardColumnID: &columnID,
					}},
					ResponsibleID: &responsibleID,
					Type:          &taskType,
					Priority:      &priority,
					ParentID:      &parentID,
				})
			},
		},
		{
			name:   "update task",
			method: http.MethodPut,
			path:   "/tm/tasks/42",
			body: `{"title":"Updated task","type":"action","priority":0,` +
				`"dueDate":"2026-09-06","tags":[4,5]}`,
			call: func(ctx context.Context, client *Client) (json.RawMessage, error) {
				return client.UpdateTask(ctx, "42", TaskUpdate{
					Title:    &title,
					Type:     &taskType,
					Priority: &priority,
					DueDate:  &dueDate,
					Tags:     &tags,
				})
			},
		},
		{
			name:   "delete task",
			method: http.MethodDelete,
			path:   "/tm/tasks/42",
			call: func(ctx context.Context, client *Client) (json.RawMessage, error) {
				return client.DeleteTask(ctx, "42")
			},
		},
		{
			name:   "complete task",
			method: http.MethodPost,
			path:   "/tm/tasks/42/complete",
			call: func(ctx context.Context, client *Client) (json.RawMessage, error) {
				return client.Complete(ctx, "42")
			},
		},
		{
			name:   "uncomplete task",
			method: http.MethodPost,
			path:   "/tm/tasks/42/un-complete",
			call: func(ctx context.Context, client *Client) (json.RawMessage, error) {
				return client.Uncomplete(ctx, "42")
			},
		},
		{
			name:   "set parent",
			method: http.MethodPost,
			path:   "/tm/tasks/42/parent",
			body:   `{"parentId":9}`,
			call: func(ctx context.Context, client *Client) (json.RawMessage, error) {
				return client.SetParent(ctx, "42", parentID)
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
				t.Fatalf("task method error = %v", err)
			}
			if string(got) != `{"success":true}` {
				t.Errorf("task method response = %q", got)
			}
		})
	}
}

func TestClient_CreateTaskAlwaysSendsLocationsArray(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		body, err := io.ReadAll(request.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		if got := string(body); got != `{"title":"Unplaced task","locations":[]}` {
			t.Errorf("request body = %q, want required empty locations array", got)
		}
		_, _ = response.Write([]byte(`{"success":true}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token", server.Client())
	if _, err := client.CreateTask(t.Context(), TaskCreate{Title: "Unplaced task"}); err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
}

func TestClient_GetTaskError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.WriteHeader(http.StatusNotFound)
		_, _ = response.Write([]byte(`{"message":"task not found"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token", server.Client())
	_, err := client.GetTask(t.Context(), "404")
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("GetTask() error = %v, want APIError", err)
	}
	if apiErr.Status != http.StatusNotFound {
		t.Errorf("GetTask() status = %d, want %d", apiErr.Status, http.StatusNotFound)
	}
}

func queryValuesEqual(left, right url.Values) bool {
	if len(left) != len(right) {
		return false
	}
	for key, leftValues := range left {
		rightValues, ok := right[key]
		if !ok || len(leftValues) != len(rightValues) {
			return false
		}
		for index := range leftValues {
			if leftValues[index] != rightValues[index] {
				return false
			}
		}
	}
	return true
}
