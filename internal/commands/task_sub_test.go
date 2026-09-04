package commands

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Stuchalin/weeek-cli/internal/api"
)

func TestTaskSubcommandsGolden(t *testing.T) {
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
			name:         "comment list",
			args:         []string{"task", "comment", "list", "42", "--limit", "25", "--offset", "5"},
			method:       http.MethodGet,
			path:         "/tm/tasks/42/comments",
			query:        url.Values{"limit": {"25"}, "offset": {"5"}},
			responseBody: `{"comments":[{"id":7,"markdown":"Ship it"}],"hasMore":false}`,
			golden:       "task_comment_list.golden",
		},
		{
			name: "comment add",
			args: []string{
				"task", "comment", "add", "42", "--markdown", "Ship it", "--parent", "7",
			},
			method:       http.MethodPost,
			path:         "/tm/tasks/42/comments",
			requestBody:  `{"markdown":"Ship it","parentId":7}`,
			responseBody: `{"comment":{"id":8,"parentId":7,"markdown":"Ship it"}}`,
			golden:       "task_comment_add.golden",
		},
		{
			name:         "timer start",
			args:         []string{"task", "timer", "start", "42"},
			method:       http.MethodPost,
			path:         "/tm/tasks/42/start-timer",
			responseBody: `{"success":true}`,
			golden:       "task_timer_start.golden",
		},
		{
			name: "time entry create",
			args: []string{
				"task", "time-entry", "create", "42", "--user", "user-1",
				"--date", "2026-09-04", "--duration", "45", "--overtime",
			},
			method: http.MethodPost,
			path:   "/tm/tasks/42/time-entries",
			requestBody: `{"userId":"user-1","isOvertime":1,` +
				`"date":"2026-09-04","duration":45}`,
			responseBody: `{"success":true,"data":{"id":"entry-1","duration":45}}`,
			golden:       "task_time_entry_create.golden",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
				if request.Method != tt.method || request.URL.Path != tt.path {
					t.Errorf(
						"request = %s %s, want %s %s",
						request.Method,
						request.URL.Path,
						tt.method,
						tt.path,
					)
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

			stdout, stderr, code := runTaskSubcommand(
				tt.args,
				api.NewClient(server.URL, "test-token", server.Client()),
			)
			if code != 0 {
				t.Fatalf("command code = %d, want 0; stderr = %q", code, stderr)
			}
			assertGolden(t, tt.golden, stdout)
		})
	}
}

func TestTaskSubentityMutationCommands(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		args   []string
		method string
		path   string
		body   string
		status int
	}{
		{
			name:   "assignee add",
			args:   []string{"task", "assignee", "add", "42", "--user", "user-1,user-2"},
			method: http.MethodPost,
			path:   "/tm/tasks/42/assignees",
			body:   `{"assignees":["user-1","user-2"]}`,
		},
		{
			name:   "assignee remove",
			args:   []string{"task", "assignee", "remove", "42", "--user", "user-1"},
			method: http.MethodDelete,
			path:   "/tm/tasks/42/assignees",
			body:   `{"assignees":["user-1"]}`,
		},
		{
			name:   "watcher add",
			args:   []string{"task", "watcher", "add", "42", "--user", "user-1,user-2"},
			method: http.MethodPost,
			path:   "/tm/tasks/42/watchers",
			body:   `{"watchers":["user-1","user-2"]}`,
		},
		{
			name:   "watcher remove",
			args:   []string{"task", "watcher", "remove", "42", "--user", "user-1"},
			method: http.MethodDelete,
			path:   "/tm/tasks/42/watchers",
			body:   `{"watchers":["user-1"]}`,
		},
		{
			name:   "timer stop",
			args:   []string{"task", "timer", "stop", "42"},
			method: http.MethodPost,
			path:   "/tm/tasks/42/stop-timer",
		},
		{
			name: "time entry update",
			args: []string{
				"task", "time-entry", "update", "42", "entry-1", "--user", "user-1",
				"--date", "2026-09-04", "--duration", "30",
			},
			method: http.MethodPut,
			path:   "/tm/tasks/42/time-entries/entry-1",
			body: `{"userId":"user-1","isOvertime":0,` +
				`"date":"2026-09-04","duration":30}`,
		},
		{
			name:   "time entry delete",
			args:   []string{"task", "time-entry", "delete", "42", "entry-1"},
			method: http.MethodDelete,
			path:   "/tm/tasks/42/time-entries/entry-1",
		},
		{
			name: "location add",
			args: []string{
				"task", "location", "add", "42", "--project", "10", "--column", "12",
				"--after", "41", "--before", "43",
			},
			method: http.MethodPost,
			path:   "/tm/tasks/42/locations",
			body:   `{"projectId":10,"boardColumnId":12,"after":41,"before":43}`,
			status: http.StatusNoContent,
		},
		{
			name:   "location remove",
			args:   []string{"task", "location", "remove", "42", "--project", "10"},
			method: http.MethodDelete,
			path:   "/tm/tasks/42/locations",
			body:   `{"projectId":10}`,
			status: http.StatusNoContent,
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
				if tt.status != 0 {
					response.WriteHeader(tt.status)
					return
				}
				_, _ = response.Write([]byte(`{"success":true}`))
			}))
			defer server.Close()

			stdout, stderr, code := runTaskSubcommand(
				tt.args,
				api.NewClient(server.URL, "test-token", server.Client()),
			)
			if code != 0 {
				t.Fatalf("command code = %d, want 0; stderr = %q", code, stderr)
			}
			if !jsonLine(stdout) {
				t.Errorf("command stdout = %q, want one JSON line", stdout)
			}
		})
	}
}

func TestTaskAttachmentUploadCommand(t *testing.T) {
	t.Parallel()

	filePath := filepath.Join(t.TempDir(), "notes.txt")
	if err := os.WriteFile(filePath, []byte("release notes"), 0o600); err != nil {
		t.Fatalf("write attachment: %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost || request.URL.Path != "/tm/tasks/42/attachments" {
			t.Errorf("request = %s %s, want upload endpoint", request.Method, request.URL.Path)
		}
		file, _, err := request.FormFile("files[]")
		if err != nil {
			t.Fatalf("read uploaded file: %v", err)
		}
		contents, err := io.ReadAll(file)
		if err != nil {
			t.Fatalf("read uploaded contents: %v", err)
		}
		if err := file.Close(); err != nil {
			t.Fatalf("close uploaded file: %v", err)
		}
		if string(contents) != "release notes" {
			t.Errorf("uploaded contents = %q, want release notes", contents)
		}
		_, _ = response.Write([]byte(`{"success":true}`))
	}))
	defer server.Close()

	stdout, stderr, code := runTaskSubcommand(
		[]string{"task", "attachment", "upload", "42", "--file", filePath},
		api.NewClient(server.URL, "test-token", server.Client()),
	)
	if code != 0 {
		t.Fatalf("command code = %d, want 0; stderr = %q", code, stderr)
	}
	if string(stdout) != "{\"success\":true}\n" {
		t.Errorf("command stdout = %q, want upload response", stdout)
	}
}

func TestTaskCommentDeleteForeignCommentError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodDelete || request.URL.Path != "/tm/tasks/42/comments/7" {
			t.Errorf("request = %s %s, want comment delete", request.Method, request.URL.Path)
		}
		response.WriteHeader(http.StatusForbidden)
		_, _ = response.Write([]byte(`{"message":"comment belongs to another user"}`))
	}))
	defer server.Close()

	_, stderr, code := runTaskSubcommand(
		[]string{"task", "comment", "delete", "42", "7"},
		api.NewClient(server.URL, "test-token", server.Client()),
	)
	if code != 1 {
		t.Errorf("command code = %d, want 1", code)
	}
	const want = `{"error":"api request failed with status 403: comment belongs to another user","status":403}` + "\n"
	if string(stderr) != want {
		t.Errorf("command stderr = %q, want %q", stderr, want)
	}
}

func TestTaskSubcommandUsageErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
	}{
		{name: "comment text required", args: []string{"task", "comment", "add", "42"}},
		{name: "member required", args: []string{"task", "assignee", "add", "42"}},
		{name: "time entry fields required", args: []string{"task", "time-entry", "create", "42"}},
		{name: "attachment path required", args: []string{"task", "attachment", "upload", "42"}},
		{name: "location project required", args: []string{"task", "location", "add", "42"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, stderr, code := runTaskSubcommand(tt.args, nil)
			if code != 2 {
				t.Errorf("command code = %d, want 2", code)
			}
			if !strings.Contains(string(stderr), "Usage: weeek") {
				t.Errorf("command stderr = %q, want usage", stderr)
			}
		})
	}
}

func runTaskSubcommand(args []string, client *api.Client) ([]byte, []byte, int) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := NewRegistry("test-version").Run(
		args,
		&Ctx{Client: client, Stdout: &stdout, Stderr: &stderr},
	)
	return stdout.Bytes(), stderr.Bytes(), code
}

func jsonLine(value []byte) bool {
	return len(value) > 1 && value[len(value)-1] == '\n' && json.Valid(value[:len(value)-1])
}
