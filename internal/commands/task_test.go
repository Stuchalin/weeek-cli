package commands

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/Stuchalin/weeek-cli/internal/api"
)

func TestTaskCommandsGolden(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		args         []string
		method       string
		path         string
		query        url.Values
		requestBody  string
		responseBody string
		golden       string
	}{
		{
			name: "task list",
			args: []string{
				"task", "list", "--responsible", "user-42", "--search", "release",
				"--priority", "0", "--tags", "7,9", "--type", "action",
				"--completed=false", "--day", "04.09.2026", "--all",
				"--sort", "-created", "--limit", "25", "--offset", "5",
			},
			method: http.MethodGet,
			path:   "/tm/tasks",
			query: url.Values{
				"all":       {"1"},
				"completed": {"0"},
				"day":       {"04.09.2026"},
				"offset":    {"5"},
				"perPage":   {"25"},
				"priority":  {"0"},
				"search":    {"release"},
				"sortBy":    {"-created"},
				"tags":      {"7", "9"},
				"type":      {"action"},
				"userId":    {"user-42"},
			},
			responseBody: `{"success":true,"tasks":[{"id":42,"title":"Release"}]}`,
			golden:       "task_list.golden",
		},
		{
			name:         "task get",
			args:         []string{"task", "get", "42"},
			method:       http.MethodGet,
			path:         "/tm/tasks/42",
			responseBody: `{"success":true,"task":{"id":42,"title":"Release"}}`,
			golden:       "task_get.golden",
		},
		{
			name: "task create",
			args: []string{
				"task", "create", "--title", "Release", "--description", "Ship it",
				"--project", "11", "--board", "12", "--column", "13",
				"--responsible", "user-42", "--type", "action", "--priority", "0",
				"--due", "2026-09-05", "--tags", "7,9", "--parent", "4",
			},
			method: http.MethodPost,
			path:   "/tm/tasks",
			requestBody: `{"title":"Release","description":"Ship it","projectId":11,` +
				`"boardId":12,"boardColumnId":13,"userId":"user-42","type":"action",` +
				`"priority":0,"dueDate":"2026-09-05","tags":[7,9],"parentId":4}`,
			responseBody: `{"success":true,"task":{"id":43,"title":"Release"}}`,
			golden:       "task_create.golden",
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
				if !queryValuesMatch(request.URL.Query(), tt.query) {
					t.Errorf("request query = %v, want %v", request.URL.Query(), tt.query)
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
			code := NewRegistry("test-version").Run(
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

func TestTaskListResolvesMeOnce(t *testing.T) {
	t.Parallel()

	var meCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/user/me":
			meCalls.Add(1)
			_, _ = response.Write([]byte(`{"success":true,"user":{"id":"user-me"}}`))
		case "/tm/tasks":
			if got := request.URL.Query().Get("userId"); got != "user-me" {
				t.Errorf("userId = %q, want %q", got, "user-me")
			}
			_, _ = response.Write([]byte(`{"success":true,"tasks":[]}`))
		default:
			t.Errorf("unexpected request path %q", request.URL.Path)
			response.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := NewRegistry("test-version").Run(
		[]string{"task", "list", "--responsible", "me"},
		&Ctx{
			Client: api.NewClient(server.URL, "test-token", server.Client()),
			Stdout: &stdout,
			Stderr: &stderr,
		},
	)
	if code != 0 {
		t.Fatalf("task list code = %d, want 0; stderr = %q", code, stderr.String())
	}
	if got := meCalls.Load(); got != 1 {
		t.Errorf("/user/me calls = %d, want 1", got)
	}
}

func TestTaskMutationCommands(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		args   []string
		method string
		path   string
		body   string
	}{
		{
			name:   "update",
			args:   []string{"task", "update", "42", "--title", "Changed", "--priority", "0", "--tags", "7,9"},
			method: http.MethodPut,
			path:   "/tm/tasks/42",
			body:   `{"title":"Changed","priority":0,"tags":[7,9]}`,
		},
		{
			name:   "delete",
			args:   []string{"task", "delete", "42"},
			method: http.MethodDelete,
			path:   "/tm/tasks/42",
		},
		{
			name:   "complete",
			args:   []string{"task", "complete", "42"},
			method: http.MethodPost,
			path:   "/tm/tasks/42/complete",
		},
		{
			name:   "uncomplete",
			args:   []string{"task", "uncomplete", "42"},
			method: http.MethodPost,
			path:   "/tm/tasks/42/un-complete",
		},
		{
			name:   "move to board",
			args:   []string{"task", "move", "42", "--board", "12"},
			method: http.MethodPost,
			path:   "/tm/tasks/42/board",
			body:   `{"boardId":12}`,
		},
		{
			name:   "move to column",
			args:   []string{"task", "move", "42", "--column", "13"},
			method: http.MethodPost,
			path:   "/tm/tasks/42/board-column",
			body:   `{"boardColumnId":13}`,
		},
		{
			name:   "move to parent",
			args:   []string{"task", "move", "42", "--parent", "4"},
			method: http.MethodPost,
			path:   "/tm/tasks/42/parent",
			body:   `{"parentId":4}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
				body, err := io.ReadAll(request.Body)
				if err != nil {
					t.Errorf("read request body: %v", err)
				}
				if request.Method != tt.method || request.URL.Path != tt.path || string(body) != tt.body {
					t.Errorf(
						"request = %s %s %q, want %s %s %q",
						request.Method,
						request.URL.Path,
						body,
						tt.method,
						tt.path,
						tt.body,
					)
				}
				_, _ = response.Write([]byte(`{"success":true}`))
			}))
			defer server.Close()

			var stdout bytes.Buffer
			var stderr bytes.Buffer
			code := NewRegistry("test-version").Run(
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
		})
	}
}

func TestTaskCommandErrors(t *testing.T) {
	t.Parallel()

	t.Run("bad flag returns usage exit", func(t *testing.T) {
		t.Parallel()

		var stderr bytes.Buffer
		code := NewRegistry("test-version").Run(
			[]string{"task", "list", "--unknown"},
			&Ctx{Stderr: &stderr},
		)
		if code != 2 {
			t.Errorf("task list code = %d, want 2", code)
		}
		if !strings.Contains(stderr.String(), "Usage: weeek task list") {
			t.Errorf("task list stderr = %q, want usage", stderr.String())
		}
	})

	t.Run("not found returns api exit", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
			response.WriteHeader(http.StatusNotFound)
			_, _ = response.Write([]byte(`{"message":"task not found"}`))
		}))
		defer server.Close()

		var stderr bytes.Buffer
		code := NewRegistry("test-version").Run(
			[]string{"task", "get", "404"},
			&Ctx{
				Client: api.NewClient(server.URL, "test-token", server.Client()),
				Stderr: &stderr,
			},
		)
		if code != 1 {
			t.Errorf("task get code = %d, want 1", code)
		}
		const want = `{"error":"api request failed with status 404: task not found","status":404}` + "\n"
		if got := stderr.String(); got != want {
			t.Errorf("task get stderr = %q, want %q", got, want)
		}
	})
}

func TestTaskMoveRequiresOneDestination(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
	}{
		{name: "missing destination", args: []string{"task", "move", "42"}},
		{name: "multiple destinations", args: []string{"task", "move", "42", "--board", "1", "--column", "2"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var stderr bytes.Buffer
			code := NewRegistry("test-version").Run(tt.args, &Ctx{Stderr: &stderr})
			if code != 2 {
				t.Errorf("task move code = %d, want 2", code)
			}
		})
	}
}

func queryValuesMatch(left, right url.Values) bool {
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
