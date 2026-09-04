package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
)

// ProjectCreate contains the fields accepted when creating a project.
type ProjectCreate struct {
	Name        string  `json:"name"`
	IsPrivate   int     `json:"isPrivate"`
	Logo        *string `json:"logo,omitempty"`
	Description *string `json:"description,omitempty"`
	PortfolioID *int64  `json:"portfolioId,omitempty"`
}

// ProjectUpdate contains the fields accepted when updating a project.
type ProjectUpdate struct {
	Name      string  `json:"name"`
	IsPrivate int     `json:"isPrivate"`
	Logo      *string `json:"logo,omitempty"`
	Color     *string `json:"color,omitempty"`
}

// PortfolioFilters contains the supported portfolio list filters.
type PortfolioFilters struct {
	Search   string
	ParentID *int64
	Limit    int
	Offset   int
}

// PortfolioCreate contains the fields accepted when creating a portfolio.
type PortfolioCreate struct {
	Name     string `json:"name"`
	ParentID *int64 `json:"parentId,omitempty"`
}

// PortfolioUpdate contains the fields accepted when updating a portfolio.
type PortfolioUpdate struct {
	Name string `json:"name"`
}

// ListProjects returns all projects visible to the current user.
func (c *Client) ListProjects(ctx context.Context) (json.RawMessage, error) {
	return c.do(ctx, http.MethodGet, "/tm/projects", nil)
}

// GetProject returns one project.
func (c *Client) GetProject(ctx context.Context, id string) (json.RawMessage, error) {
	return c.do(ctx, http.MethodGet, projectPath(id), nil)
}

// CreateProject creates a project.
func (c *Client) CreateProject(ctx context.Context, input ProjectCreate) (json.RawMessage, error) {
	return c.doJSON(ctx, http.MethodPost, "/tm/projects", input)
}

// UpdateProject updates a project.
func (c *Client) UpdateProject(
	ctx context.Context,
	id string,
	input ProjectUpdate,
) (json.RawMessage, error) {
	return c.doJSON(ctx, http.MethodPut, projectPath(id), input)
}

// DeleteProject deletes a project.
func (c *Client) DeleteProject(ctx context.Context, id string) (json.RawMessage, error) {
	return c.do(ctx, http.MethodDelete, projectPath(id), nil)
}

// ArchiveProject archives a project.
func (c *Client) ArchiveProject(ctx context.Context, id string) (json.RawMessage, error) {
	return c.do(ctx, http.MethodPost, projectPath(id)+"/archive", nil)
}

// UnarchiveProject restores an archived project.
func (c *Client) UnarchiveProject(ctx context.Context, id string) (json.RawMessage, error) {
	return c.do(ctx, http.MethodPost, projectPath(id)+"/un-archive", nil)
}

// ListPortfolios returns portfolios matching the supplied filters.
func (c *Client) ListPortfolios(
	ctx context.Context,
	filters PortfolioFilters,
) (json.RawMessage, error) {
	query := url.Values{}
	if filters.Search != "" {
		query.Set("search", filters.Search)
	}
	if filters.ParentID != nil {
		query.Set("parentId", strconv.FormatInt(*filters.ParentID, 10))
	}
	if filters.Limit != 0 {
		query.Set("limit", strconv.Itoa(filters.Limit))
	}
	if filters.Offset != 0 {
		query.Set("offset", strconv.Itoa(filters.Offset))
	}

	return c.do(ctx, http.MethodGet, pathWithQuery("/tm/portfolios", query), nil)
}

// GetPortfolio returns one portfolio.
func (c *Client) GetPortfolio(ctx context.Context, id string) (json.RawMessage, error) {
	return c.do(ctx, http.MethodGet, portfolioPath(id), nil)
}

// CreatePortfolio creates a portfolio.
func (c *Client) CreatePortfolio(ctx context.Context, input PortfolioCreate) (json.RawMessage, error) {
	return c.doJSON(ctx, http.MethodPost, "/tm/portfolios", input)
}

// UpdatePortfolio updates a portfolio.
func (c *Client) UpdatePortfolio(
	ctx context.Context,
	id string,
	input PortfolioUpdate,
) (json.RawMessage, error) {
	return c.doJSON(ctx, http.MethodPut, portfolioPath(id), input)
}

// DeletePortfolio deletes a portfolio.
func (c *Client) DeletePortfolio(ctx context.Context, id string) (json.RawMessage, error) {
	return c.do(ctx, http.MethodDelete, portfolioPath(id), nil)
}

func projectPath(id string) string {
	return "/tm/projects/" + url.PathEscape(id)
}

func portfolioPath(id string) string {
	return "/tm/portfolios/" + url.PathEscape(id)
}
