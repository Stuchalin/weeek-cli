package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

// BoardCreate contains the fields accepted when creating a board.
type BoardCreate struct {
	Name      string `json:"name"`
	ProjectID int64  `json:"projectId"`
}

// BoardUpdate contains the fields accepted when updating a board.
type BoardUpdate struct {
	Name string `json:"name"`
}

// BoardMove describes the board after which another board is placed.
// A nil UpperBoardID moves the board to the top.
type BoardMove struct {
	UpperBoardID *int64 `json:"upperBoardId"`
}

// BoardColumnCreate contains the fields accepted when creating a board column.
type BoardColumnCreate struct {
	Name    string `json:"name"`
	BoardID int64  `json:"boardId"`
}

// BoardColumnUpdate contains the fields accepted when updating a board column.
type BoardColumnUpdate struct {
	Name string `json:"name"`
}

// BoardColumnMove describes the column after which another column is placed.
// A nil UpperBoardColumnID moves the column to the top.
type BoardColumnMove struct {
	UpperBoardColumnID *int64 `json:"upperBoardColumnId"`
}

// ListBoards returns boards belonging to a project.
func (c *Client) ListBoards(ctx context.Context, projectID int64) (json.RawMessage, error) {
	query := url.Values{"projectId": {strconv.FormatInt(projectID, 10)}}
	return c.do(ctx, http.MethodGet, pathWithQuery("/tm/boards", query), nil)
}

// GetBoard returns one board from the documented project-scoped board list.
func (c *Client) GetBoard(
	ctx context.Context,
	projectID int64,
	id string,
) (json.RawMessage, error) {
	response, err := c.ListBoards(ctx, projectID)
	if err != nil {
		return nil, err
	}

	var payload struct {
		Success bool              `json:"success"`
		Boards  []json.RawMessage `json:"boards"`
	}
	if err := json.Unmarshal(response, &payload); err != nil {
		return nil, fmt.Errorf("decoding board list: %w", err)
	}

	for _, board := range payload.Boards {
		var identity struct {
			ID json.Number `json:"id"`
		}
		if err := json.Unmarshal(board, &identity); err != nil {
			return nil, fmt.Errorf("decoding board: %w", err)
		}
		if identity.ID.String() != id {
			continue
		}

		result, err := json.Marshal(struct {
			Success bool            `json:"success"`
			Board   json.RawMessage `json:"board"`
		}{
			Success: payload.Success,
			Board:   board,
		})
		if err != nil {
			return nil, fmt.Errorf("encoding board response: %w", err)
		}
		return json.RawMessage(result), nil
	}

	return nil, errors.New("board not found")
}

// CreateBoard creates a board.
func (c *Client) CreateBoard(ctx context.Context, input BoardCreate) (json.RawMessage, error) {
	return c.doJSON(ctx, http.MethodPost, "/tm/boards", input)
}

// UpdateBoard updates a board.
func (c *Client) UpdateBoard(
	ctx context.Context,
	id string,
	input BoardUpdate,
) (json.RawMessage, error) {
	return c.doJSON(ctx, http.MethodPut, boardPath(id), input)
}

// DeleteBoard deletes a board.
func (c *Client) DeleteBoard(ctx context.Context, id string) (json.RawMessage, error) {
	return c.do(ctx, http.MethodDelete, boardPath(id), nil)
}

// MoveBoard changes a board's position.
func (c *Client) MoveBoard(
	ctx context.Context,
	id string,
	input BoardMove,
) (json.RawMessage, error) {
	return c.doJSON(ctx, http.MethodPost, boardPath(id)+"/move", input)
}

// ListBoardColumns returns board columns, optionally restricted to a board.
func (c *Client) ListBoardColumns(
	ctx context.Context,
	boardID *int64,
) (json.RawMessage, error) {
	query := url.Values{}
	if boardID != nil {
		query.Set("boardId", strconv.FormatInt(*boardID, 10))
	}
	return c.do(ctx, http.MethodGet, pathWithQuery("/tm/board-columns", query), nil)
}

// CreateBoardColumn creates a board column.
func (c *Client) CreateBoardColumn(
	ctx context.Context,
	input BoardColumnCreate,
) (json.RawMessage, error) {
	return c.doJSON(ctx, http.MethodPost, "/tm/board-columns", input)
}

// UpdateBoardColumn updates a board column.
func (c *Client) UpdateBoardColumn(
	ctx context.Context,
	id string,
	input BoardColumnUpdate,
) (json.RawMessage, error) {
	return c.doJSON(ctx, http.MethodPut, boardColumnPath(id), input)
}

// DeleteBoardColumn deletes a board column.
func (c *Client) DeleteBoardColumn(ctx context.Context, id string) (json.RawMessage, error) {
	return c.do(ctx, http.MethodDelete, boardColumnPath(id), nil)
}

// MoveBoardColumn changes a board column's position.
func (c *Client) MoveBoardColumn(
	ctx context.Context,
	id string,
	input BoardColumnMove,
) (json.RawMessage, error) {
	return c.doJSON(ctx, http.MethodPost, boardColumnPath(id)+"/move", input)
}

func boardPath(id string) string {
	return "/tm/boards/" + url.PathEscape(id)
}

func boardColumnPath(id string) string {
	return "/tm/board-columns/" + url.PathEscape(id)
}
