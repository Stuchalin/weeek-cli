package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// Me contains the fields from the current user response used by the CLI.
type Me struct {
	ID int64 `json:"id"`
}

// GetMe returns the current user response and the fields used by the CLI.
func (c *Client) GetMe(ctx context.Context) (json.RawMessage, Me, error) {
	response, err := c.do(ctx, http.MethodGet, "/user/me", nil)
	if err != nil {
		return nil, Me{}, err
	}

	var me Me
	if err := json.Unmarshal(response, &me); err != nil {
		return nil, Me{}, fmt.Errorf("decoding current user response: %w", err)
	}

	return response, me, nil
}
