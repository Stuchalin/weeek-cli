package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
)

// TaskFilters contains the supported task-list filters.
type TaskFilters struct {
	Responsible string
	Search      string
	Priority    *int
	Tags        []string
	Type        string
	Completed   *bool
	Day         string
	All         *bool
	Sort        string
	Limit       int
	Offset      int
}

// TaskCreate contains the fields accepted when creating a task.
type TaskCreate struct {
	Title         string               `json:"title"`
	Description   *string              `json:"description,omitempty"`
	Day           *string              `json:"day,omitempty"`
	Locations     []TaskCreateLocation `json:"locations"`
	ResponsibleID *string              `json:"userId,omitempty"`
	Type          *string              `json:"type,omitempty"`
	Priority      *int                 `json:"priority,omitempty"`
	ParentID      *int64               `json:"parentId,omitempty"`
}

// TaskCreateLocation places a newly created task in a project and optional column.
type TaskCreateLocation struct {
	ProjectID     int64  `json:"projectId"`
	BoardColumnID *int64 `json:"boardColumnId"`
}

// TaskUpdate contains task fields that can be changed independently.
type TaskUpdate struct {
	Title    *string  `json:"title,omitempty"`
	Type     *string  `json:"type,omitempty"`
	Priority *int     `json:"priority,omitempty"`
	DueDate  *string  `json:"dueDate,omitempty"`
	Tags     *[]int64 `json:"tags,omitempty"`
}

type taskParentChange struct {
	ParentID int64 `json:"parentId"`
}

// ListTasks returns tasks matching filters.
func (c *Client) ListTasks(ctx context.Context, filters TaskFilters) (json.RawMessage, error) {
	query := url.Values{}
	setQueryValue(query, "userId", filters.Responsible)
	setQueryValue(query, "search", filters.Search)
	setQueryValue(query, "type", filters.Type)
	setQueryValue(query, "day", filters.Day)
	setQueryValue(query, "sortBy", filters.Sort)
	if filters.Priority != nil {
		query.Set("priority", strconv.Itoa(*filters.Priority))
	}
	for _, tag := range filters.Tags {
		query.Add("tags", tag)
	}
	if filters.Completed != nil {
		query.Set("completed", booleanQueryValue(*filters.Completed))
	}
	if filters.All != nil {
		query.Set("all", booleanQueryValue(*filters.All))
	}
	if filters.Limit != 0 {
		query.Set("perPage", strconv.Itoa(filters.Limit))
	}
	if filters.Offset != 0 {
		query.Set("offset", strconv.Itoa(filters.Offset))
	}

	return c.do(ctx, http.MethodGet, pathWithQuery("/tm/tasks", query), nil)
}

// GetTask returns one task.
func (c *Client) GetTask(ctx context.Context, id string) (json.RawMessage, error) {
	return c.do(ctx, http.MethodGet, taskPath(id), nil)
}

// CreateTask creates a task.
func (c *Client) CreateTask(ctx context.Context, input TaskCreate) (json.RawMessage, error) {
	if input.Locations == nil {
		input.Locations = []TaskCreateLocation{}
	}
	return c.doJSON(ctx, http.MethodPost, "/tm/tasks", input)
}

// UpdateTask updates a task.
func (c *Client) UpdateTask(
	ctx context.Context,
	id string,
	input TaskUpdate,
) (json.RawMessage, error) {
	return c.doJSON(ctx, http.MethodPut, taskPath(id), input)
}

// DeleteTask deletes a task.
func (c *Client) DeleteTask(ctx context.Context, id string) (json.RawMessage, error) {
	return c.do(ctx, http.MethodDelete, taskPath(id), nil)
}

// Complete marks a task completed.
func (c *Client) Complete(ctx context.Context, id string) (json.RawMessage, error) {
	return c.do(ctx, http.MethodPost, taskPath(id)+"/complete", nil)
}

// Uncomplete marks a task incomplete.
func (c *Client) Uncomplete(ctx context.Context, id string) (json.RawMessage, error) {
	return c.do(ctx, http.MethodPost, taskPath(id)+"/un-complete", nil)
}

// SetParent changes a task's parent.
func (c *Client) SetParent(ctx context.Context, id string, parentID int64) (json.RawMessage, error) {
	return c.doJSON(
		ctx,
		http.MethodPost,
		taskPath(id)+"/parent",
		taskParentChange{ParentID: parentID},
	)
}

func taskPath(id string) string {
	return "/tm/tasks/" + url.PathEscape(id)
}

func pathWithQuery(path string, query url.Values) string {
	encoded := query.Encode()
	if encoded == "" {
		return path
	}
	return path + "?" + encoded
}

func setQueryValue(query url.Values, name string, value string) {
	if value != "" {
		query.Set(name, value)
	}
}

func booleanQueryValue(value bool) string {
	if value {
		return "1"
	}
	return "0"
}
