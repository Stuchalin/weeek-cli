package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestClient_TaskSubentityMethods(t *testing.T) {
	t.Parallel()

	parentID := int64(7)
	columnID := int64(12)
	afterID := int64(41)
	beforeID := int64(43)
	timeEntry := TimeEntryInput{
		UserID:     "user-1",
		IsOvertime: 1,
		Date:       "2026-09-04",
		Duration:   45,
	}

	tests := []struct {
		name   string
		method string
		path   string
		query  string
		body   string
		status int
		want   string
		call   func(context.Context, *Client) (json.RawMessage, error)
	}{
		{
			name:   "list comments",
			method: http.MethodGet,
			path:   "/tm/tasks/42/comments",
			query:  "limit=25&offset=5",
			want:   `{"success":true}`,
			call: func(ctx context.Context, client *Client) (json.RawMessage, error) {
				return client.ListTaskComments(ctx, "42", CommentFilters{Limit: 25, Offset: 5})
			},
		},
		{
			name:   "add comment",
			method: http.MethodPost,
			path:   "/tm/tasks/42/comments",
			body:   `{"markdown":"Ship it","parentId":7}`,
			want:   `{"success":true}`,
			call: func(ctx context.Context, client *Client) (json.RawMessage, error) {
				return client.AddTaskComment(
					ctx,
					"42",
					CommentCreate{Markdown: "Ship it", ParentID: &parentID},
				)
			},
		},
		{
			name:   "delete comment",
			method: http.MethodDelete,
			path:   "/tm/tasks/42/comments/9",
			status: http.StatusNoContent,
			call: func(ctx context.Context, client *Client) (json.RawMessage, error) {
				return client.DeleteTaskComment(ctx, "42", "9")
			},
		},
		{
			name:   "add assignees",
			method: http.MethodPost,
			path:   "/tm/tasks/42/assignees",
			body:   `{"assignees":["user-1","user-2"]}`,
			want:   `{"success":true}`,
			call: func(ctx context.Context, client *Client) (json.RawMessage, error) {
				return client.AddTaskAssignees(ctx, "42", []string{"user-1", "user-2"})
			},
		},
		{
			name:   "remove assignees",
			method: http.MethodDelete,
			path:   "/tm/tasks/42/assignees",
			body:   `{"assignees":["user-1"]}`,
			want:   `{"success":true}`,
			call: func(ctx context.Context, client *Client) (json.RawMessage, error) {
				return client.RemoveTaskAssignees(ctx, "42", []string{"user-1"})
			},
		},
		{
			name:   "add watchers",
			method: http.MethodPost,
			path:   "/tm/tasks/42/watchers",
			body:   `{"watchers":["user-1","user-2"]}`,
			want:   `{"success":true}`,
			call: func(ctx context.Context, client *Client) (json.RawMessage, error) {
				return client.AddTaskWatchers(ctx, "42", []string{"user-1", "user-2"})
			},
		},
		{
			name:   "remove watchers",
			method: http.MethodDelete,
			path:   "/tm/tasks/42/watchers",
			body:   `{"watchers":["user-1"]}`,
			want:   `{"success":true}`,
			call: func(ctx context.Context, client *Client) (json.RawMessage, error) {
				return client.RemoveTaskWatchers(ctx, "42", []string{"user-1"})
			},
		},
		{
			name:   "start timer",
			method: http.MethodPost,
			path:   "/tm/tasks/42/start-timer",
			want:   `{"success":true}`,
			call: func(ctx context.Context, client *Client) (json.RawMessage, error) {
				return client.StartTaskTimer(ctx, "42")
			},
		},
		{
			name:   "stop timer",
			method: http.MethodPost,
			path:   "/tm/tasks/42/stop-timer",
			want:   `{"success":true}`,
			call: func(ctx context.Context, client *Client) (json.RawMessage, error) {
				return client.StopTaskTimer(ctx, "42")
			},
		},
		{
			name:   "create time entry",
			method: http.MethodPost,
			path:   "/tm/tasks/42/time-entries",
			body:   `{"userId":"user-1","isOvertime":1,"date":"2026-09-04","duration":45}`,
			want:   `{"success":true}`,
			call: func(ctx context.Context, client *Client) (json.RawMessage, error) {
				return client.CreateTaskTimeEntry(ctx, "42", timeEntry)
			},
		},
		{
			name:   "update time entry",
			method: http.MethodPut,
			path:   "/tm/tasks/42/time-entries/entry-1",
			body:   `{"userId":"user-1","isOvertime":1,"date":"2026-09-04","duration":45}`,
			want:   `{"success":true}`,
			call: func(ctx context.Context, client *Client) (json.RawMessage, error) {
				return client.UpdateTaskTimeEntry(ctx, "42", "entry-1", timeEntry)
			},
		},
		{
			name:   "delete time entry",
			method: http.MethodDelete,
			path:   "/tm/tasks/42/time-entries/entry-1",
			want:   `{"success":true}`,
			call: func(ctx context.Context, client *Client) (json.RawMessage, error) {
				return client.DeleteTaskTimeEntry(ctx, "42", "entry-1")
			},
		},
		{
			name:   "add location",
			method: http.MethodPost,
			path:   "/tm/tasks/42/locations",
			body:   `{"projectId":10,"boardColumnId":12,"after":41,"before":43}`,
			want:   `{"success":true}`,
			call: func(ctx context.Context, client *Client) (json.RawMessage, error) {
				return client.AddTaskLocation(
					ctx,
					"42",
					LocationCreate{
						ProjectID:     10,
						BoardColumnID: &columnID,
						After:         &afterID,
						Before:        &beforeID,
					},
				)
			},
		},
		{
			name:   "remove location",
			method: http.MethodDelete,
			path:   "/tm/tasks/42/locations",
			body:   `{"projectId":10}`,
			status: http.StatusNoContent,
			call: func(ctx context.Context, client *Client) (json.RawMessage, error) {
				return client.RemoveTaskLocation(ctx, "42", 10)
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
				if request.URL.RawQuery != tt.query {
					t.Errorf("request query = %q, want %q", request.URL.RawQuery, tt.query)
				}
				body, err := io.ReadAll(request.Body)
				if err != nil {
					t.Errorf("read request body: %v", err)
				}
				if string(body) != tt.body {
					t.Errorf("request body = %q, want %q", body, tt.body)
				}
				status := tt.status
				if status == 0 {
					status = http.StatusOK
				}
				response.WriteHeader(status)
				if status != http.StatusNoContent {
					_, _ = response.Write([]byte(`{"success":true}`))
				}
			}))
			defer server.Close()

			client := NewClient(server.URL, "test-token", server.Client())
			got, err := tt.call(t.Context(), client)
			if err != nil {
				t.Fatalf("task subentity method error = %v", err)
			}
			if string(got) != tt.want {
				t.Errorf("task subentity response = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestClient_UploadTaskAttachment(t *testing.T) {
	t.Parallel()

	filePath := filepath.Join(t.TempDir(), "notes.txt")
	if err := os.WriteFile(filePath, []byte("release notes"), 0o600); err != nil {
		t.Fatalf("write attachment: %v", err)
	}

	transport := roundTripFunc(func(request *http.Request) (*http.Response, error) {
		deadline, hasDeadline := request.Context().Deadline()
		if !hasDeadline {
			t.Fatal("upload request context has no deadline")
		}
		remaining := time.Until(deadline)
		if remaining <= requestTimeout || remaining > uploadTimeout {
			t.Errorf("upload request deadline is %s away, want more than %s and at most %s", remaining, requestTimeout, uploadTimeout)
		}
		if request.Method != http.MethodPost || request.URL.Path != "/tm/tasks/42/attachments" {
			t.Errorf("request = %s %s, want POST /tm/tasks/42/attachments", request.Method, request.URL.Path)
		}
		if got := request.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("Authorization = %q, want bearer token", got)
		}
		reader, err := request.MultipartReader()
		if err != nil {
			t.Fatalf("create multipart reader: %v", err)
		}
		part, err := reader.NextPart()
		if err != nil {
			t.Fatalf("read multipart part: %v", err)
		}
		contents, err := io.ReadAll(part)
		if err != nil {
			t.Fatalf("read multipart contents: %v", err)
		}
		if err := part.Close(); err != nil {
			t.Fatalf("close multipart part: %v", err)
		}
		if part.FormName() != "files[]" || part.FileName() != "notes.txt" {
			t.Errorf("multipart part = %q %q, want files[] notes.txt", part.FormName(), part.FileName())
		}
		if string(contents) != "release notes" {
			t.Errorf("multipart contents = %q, want release notes", contents)
		}

		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{},
			Body:       io.NopCloser(strings.NewReader(`{"success":true}`)),
			Request:    request,
		}, nil
	})
	client := NewClient(
		"https://example.test",
		"test-token",
		&http.Client{Transport: transport},
	)

	got, err := client.UploadTaskAttachment(context.Background(), "42", filePath)
	if err != nil {
		t.Fatalf("UploadTaskAttachment() error = %v", err)
	}
	if string(got) != `{"success":true}` {
		t.Errorf("UploadTaskAttachment() response = %q", got)
	}
}

func TestClient_UploadTaskAttachmentRejectsOversizedFile(t *testing.T) {
	t.Parallel()

	filePath := filepath.Join(t.TempDir(), "large.bin")
	file, err := os.Create(filePath)
	if err != nil {
		t.Fatalf("create attachment: %v", err)
	}
	if err := file.Truncate(maxAttachmentSize + 1); err != nil {
		t.Fatalf("truncate attachment: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("close attachment: %v", err)
	}

	client := NewClient("https://example.test", "test-token", nil)
	_, err = client.UploadTaskAttachment(t.Context(), "42", filePath)
	if err == nil || !strings.Contains(err.Error(), "byte limit") {
		t.Fatalf("UploadTaskAttachment() error = %v, want size limit error", err)
	}
}
