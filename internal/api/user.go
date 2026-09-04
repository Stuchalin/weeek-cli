package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// Me contains the fields from the current user response used by the CLI.
type Me struct {
	ID string `json:"id"`
}

// UnmarshalJSON accepts both the current string ID and legacy numeric fixtures.
func (m *Me) UnmarshalJSON(data []byte) error {
	var payload struct {
		ID json.RawMessage `json:"id"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return err
	}
	if len(payload.ID) == 0 || string(payload.ID) == "null" {
		m.ID = ""
		return nil
	}
	if err := json.Unmarshal(payload.ID, &m.ID); err == nil {
		return nil
	}

	var number json.Number
	if err := json.Unmarshal(payload.ID, &number); err != nil {
		return fmt.Errorf("decoding current user id: %w", err)
	}
	m.ID = number.String()
	return nil
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
