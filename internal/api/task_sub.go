package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

const (
	maxAttachmentSize = 100 * 1024 * 1024
	uploadTimeout     = 30 * time.Minute
)

// CommentFilters contains task-comment list pagination.
type CommentFilters struct {
	Limit  int
	Offset int
}

// CommentCreate contains the fields accepted when creating a task comment.
type CommentCreate struct {
	Markdown string `json:"markdown"`
	ParentID *int64 `json:"parentId,omitempty"`
}

// TimeEntryInput contains the fields required to create or update a time entry.
type TimeEntryInput struct {
	UserID     string `json:"userId"`
	IsOvertime int    `json:"isOvertime"`
	Date       string `json:"date"`
	Duration   int    `json:"duration"`
}

// LocationCreate contains the fields accepted when adding a task to a project.
type LocationCreate struct {
	ProjectID     int64  `json:"projectId"`
	BoardColumnID *int64 `json:"boardColumnId,omitempty"`
	After         *int64 `json:"after,omitempty"`
	Before        *int64 `json:"before,omitempty"`
}

type assigneeChange struct {
	Assignees []string `json:"assignees"`
}

type watcherChange struct {
	Watchers []string `json:"watchers"`
}

type locationDelete struct {
	ProjectID int64 `json:"projectId"`
}

// ListTaskComments returns comments for a task.
func (c *Client) ListTaskComments(
	ctx context.Context,
	taskID string,
	filters CommentFilters,
) (json.RawMessage, error) {
	query := url.Values{}
	if filters.Limit != 0 {
		query.Set("limit", strconv.Itoa(filters.Limit))
	}
	if filters.Offset != 0 {
		query.Set("offset", strconv.Itoa(filters.Offset))
	}

	return c.do(
		ctx,
		http.MethodGet,
		pathWithQuery(taskPath(taskID)+"/comments", query),
		nil,
	)
}

// AddTaskComment creates a comment or reply on a task.
func (c *Client) AddTaskComment(
	ctx context.Context,
	taskID string,
	input CommentCreate,
) (json.RawMessage, error) {
	return c.doJSON(ctx, http.MethodPost, taskPath(taskID)+"/comments", input)
}

// DeleteTaskComment deletes a comment from a task.
func (c *Client) DeleteTaskComment(
	ctx context.Context,
	taskID string,
	commentID string,
) (json.RawMessage, error) {
	path := taskPath(taskID) + "/comments/" + url.PathEscape(commentID)
	return c.do(ctx, http.MethodDelete, path, nil)
}

// AddTaskAssignees assigns users to a task.
func (c *Client) AddTaskAssignees(
	ctx context.Context,
	taskID string,
	assignees []string,
) (json.RawMessage, error) {
	return c.doJSON(
		ctx,
		http.MethodPost,
		taskPath(taskID)+"/assignees",
		assigneeChange{Assignees: assignees},
	)
}

// RemoveTaskAssignees removes assigned users from a task.
func (c *Client) RemoveTaskAssignees(
	ctx context.Context,
	taskID string,
	assignees []string,
) (json.RawMessage, error) {
	return c.doJSON(
		ctx,
		http.MethodDelete,
		taskPath(taskID)+"/assignees",
		assigneeChange{Assignees: assignees},
	)
}

// AddTaskWatchers subscribes users to a task.
func (c *Client) AddTaskWatchers(
	ctx context.Context,
	taskID string,
	watchers []string,
) (json.RawMessage, error) {
	return c.doJSON(
		ctx,
		http.MethodPost,
		taskPath(taskID)+"/watchers",
		watcherChange{Watchers: watchers},
	)
}

// RemoveTaskWatchers unsubscribes users from a task.
func (c *Client) RemoveTaskWatchers(
	ctx context.Context,
	taskID string,
	watchers []string,
) (json.RawMessage, error) {
	return c.doJSON(
		ctx,
		http.MethodDelete,
		taskPath(taskID)+"/watchers",
		watcherChange{Watchers: watchers},
	)
}

// StartTaskTimer starts the current user's timer for a task.
func (c *Client) StartTaskTimer(ctx context.Context, taskID string) (json.RawMessage, error) {
	return c.do(ctx, http.MethodPost, taskPath(taskID)+"/start-timer", nil)
}

// StopTaskTimer stops the current user's timer for a task.
func (c *Client) StopTaskTimer(ctx context.Context, taskID string) (json.RawMessage, error) {
	return c.do(ctx, http.MethodPost, taskPath(taskID)+"/stop-timer", nil)
}

// CreateTaskTimeEntry creates a time entry for a task.
func (c *Client) CreateTaskTimeEntry(
	ctx context.Context,
	taskID string,
	input TimeEntryInput,
) (json.RawMessage, error) {
	return c.doJSON(ctx, http.MethodPost, taskPath(taskID)+"/time-entries", input)
}

// UpdateTaskTimeEntry updates a time entry for a task.
func (c *Client) UpdateTaskTimeEntry(
	ctx context.Context,
	taskID string,
	timeEntryID string,
	input TimeEntryInput,
) (json.RawMessage, error) {
	path := taskPath(taskID) + "/time-entries/" + url.PathEscape(timeEntryID)
	return c.doJSON(ctx, http.MethodPut, path, input)
}

// DeleteTaskTimeEntry deletes a time entry from a task.
func (c *Client) DeleteTaskTimeEntry(
	ctx context.Context,
	taskID string,
	timeEntryID string,
) (json.RawMessage, error) {
	path := taskPath(taskID) + "/time-entries/" + url.PathEscape(timeEntryID)
	return c.do(ctx, http.MethodDelete, path, nil)
}

// UploadTaskAttachment uploads one file to a task with an extended request timeout.
func (c *Client) UploadTaskAttachment(
	ctx context.Context,
	taskID string,
	filePath string,
) (responseBody json.RawMessage, err error) {
	body, contentType, contentLength, err := createAttachmentBody(filePath)
	if err != nil {
		return nil, err
	}
	defer func() {
		cleanupErr := cleanupAttachmentBody(body)
		if cleanupErr == nil {
			return
		}

		cleanupErr = fmt.Errorf("cleaning attachment form: %w", cleanupErr)
		if err == nil {
			responseBody = nil
			err = cleanupErr
			return
		}
		err = errors.Join(err, cleanupErr)
	}()

	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+taskPath(taskID)+"/attachments",
		io.NopCloser(body),
	)
	if err != nil {
		return nil, fmt.Errorf("creating api request: %w", err)
	}
	request.ContentLength = contentLength
	request.Header.Set("Authorization", "Bearer "+c.token)
	request.Header.Set("Content-Type", contentType)

	uploadClient := *c.httpClient
	uploadClient.Timeout = uploadTimeout
	response, err := uploadClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("sending api request: %w", err)
	}

	return readResponse(response)
}

func createAttachmentBody(filePath string) (_ *os.File, _ string, _ int64, err error) {
	body, err := os.CreateTemp("", "weeek-attachment-*")
	if err != nil {
		return nil, "", 0, fmt.Errorf("creating attachment form: %w", err)
	}
	removeBody := true
	defer func() {
		if !removeBody {
			return
		}
		if cleanupErr := cleanupAttachmentBody(body); cleanupErr != nil {
			err = errors.Join(err, fmt.Errorf("cleaning attachment form: %w", cleanupErr))
		}
	}()

	writer := multipart.NewWriter(body)
	contentType := writer.FormDataContentType()
	if err := writeAttachmentPart(writer, filePath); err != nil {
		return nil, "", 0, err
	}
	if err := writer.Close(); err != nil {
		return nil, "", 0, fmt.Errorf("closing attachment form: %w", err)
	}

	contentLength, err := body.Seek(0, io.SeekEnd)
	if err != nil {
		return nil, "", 0, fmt.Errorf("measuring attachment form: %w", err)
	}
	if _, err := body.Seek(0, io.SeekStart); err != nil {
		return nil, "", 0, fmt.Errorf("rewinding attachment form: %w", err)
	}

	removeBody = false
	return body, contentType, contentLength, nil
}

func writeAttachmentPart(writer *multipart.Writer, filePath string) (err error) {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("reading attachment: %w", err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("closing attachment: %w", closeErr))
		}
	}()

	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("reading attachment metadata: %w", err)
	}
	if !info.Mode().IsRegular() {
		return errors.New("attachment must be a regular file")
	}
	if info.Size() > maxAttachmentSize {
		return fmt.Errorf("attachment exceeds %d byte limit", maxAttachmentSize)
	}

	part, err := writer.CreateFormFile("files[]", filepath.Base(filePath))
	if err != nil {
		return fmt.Errorf("creating attachment form: %w", err)
	}
	written, err := io.Copy(part, io.LimitReader(file, maxAttachmentSize+1))
	if err != nil {
		return fmt.Errorf("writing attachment form: %w", err)
	}
	if written > maxAttachmentSize {
		return fmt.Errorf("attachment exceeds %d byte limit", maxAttachmentSize)
	}

	return nil
}

func cleanupAttachmentBody(body *os.File) error {
	name := body.Name()
	closeErr := body.Close()
	removeErr := os.Remove(name)
	return errors.Join(closeErr, removeErr)
}

// AddTaskLocation adds a task to a project and optional board column.
func (c *Client) AddTaskLocation(
	ctx context.Context,
	taskID string,
	input LocationCreate,
) (json.RawMessage, error) {
	return c.doJSON(ctx, http.MethodPost, taskPath(taskID)+"/locations", input)
}

// RemoveTaskLocation removes a task from a project.
func (c *Client) RemoveTaskLocation(
	ctx context.Context,
	taskID string,
	projectID int64,
) (json.RawMessage, error) {
	return c.doJSON(
		ctx,
		http.MethodDelete,
		taskPath(taskID)+"/locations",
		locationDelete{ProjectID: projectID},
	)
}
