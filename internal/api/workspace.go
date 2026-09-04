package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// TagCreate contains the fields accepted when creating a workspace tag.
type TagCreate struct {
	Title string `json:"title"`
}

// TagUpdate contains the fields accepted when updating a workspace tag.
type TagUpdate struct {
	Title string `json:"title"`
	Color string `json:"color"`
}

// GetWorkspace returns information about the current workspace.
func (c *Client) GetWorkspace(ctx context.Context) (json.RawMessage, error) {
	return c.do(ctx, http.MethodGet, "/ws", nil)
}

// GetMembers returns members of the current workspace.
func (c *Client) GetMembers(ctx context.Context) (json.RawMessage, error) {
	return c.do(ctx, http.MethodGet, "/ws/members", nil)
}

// ListTags returns tags in the current workspace.
func (c *Client) ListTags(ctx context.Context) (json.RawMessage, error) {
	return c.do(ctx, http.MethodGet, "/ws/tags", nil)
}

// CreateTag creates a workspace tag.
func (c *Client) CreateTag(ctx context.Context, input TagCreate) (json.RawMessage, error) {
	return c.doJSON(ctx, http.MethodPost, "/ws/tags", input)
}

// UpdateTag updates a workspace tag.
func (c *Client) UpdateTag(
	ctx context.Context,
	id string,
	input TagUpdate,
) (json.RawMessage, error) {
	return c.doJSON(ctx, http.MethodPut, tagPath(id), input)
}

// DeleteTag deletes a workspace tag.
func (c *Client) DeleteTag(ctx context.Context, id string) (json.RawMessage, error) {
	return c.do(ctx, http.MethodDelete, tagPath(id), nil)
}

// GetAttachment returns metadata for a workspace attachment.
func (c *Client) GetAttachment(ctx context.Context, fileID string) (json.RawMessage, error) {
	path := "/ws/attachments/" + url.PathEscape(fileID)
	return c.do(ctx, http.MethodGet, path, nil)
}

func (c *Client) doJSON(
	ctx context.Context,
	method string,
	path string,
	input any,
) (json.RawMessage, error) {
	body, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("encoding api request: %w", err)
	}

	return c.do(ctx, method, path, body)
}

func tagPath(id string) string {
	return "/ws/tags/" + url.PathEscape(id)
}
