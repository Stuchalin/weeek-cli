package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

func registerMeCommand(registry Registry) {
	registry.Register(meCommand())
}

func meCommand() Command {
	return Command{
		Name:  "me",
		Usage: "me",
		Short: "Show the current user and workspace",
		Run: func(ctx *Ctx) error {
			if len(ctx.Args) != 0 {
				return &usageError{err: errors.New("me does not accept arguments")}
			}
			client, err := requireClient(ctx)
			if err != nil {
				return err
			}

			userEnvelope, _, err := client.GetMe(context.Background())
			if err != nil {
				return err
			}
			user, err := envelopeField(userEnvelope, "user")
			if err != nil {
				return fmt.Errorf("decoding current user response: %w", err)
			}
			workspaceEnvelope, err := client.GetWorkspace(context.Background())
			if err != nil {
				return err
			}
			workspace, err := envelopeField(workspaceEnvelope, "workspace")
			if err != nil {
				return fmt.Errorf("decoding workspace response: %w", err)
			}

			output := struct {
				User      json.RawMessage `json:"user"`
				Workspace json.RawMessage `json:"workspace"`
			}{
				User:      user,
				Workspace: workspace,
			}
			if err := json.NewEncoder(ctx.Stdout).Encode(output); err != nil {
				return fmt.Errorf("writing me output: %w", err)
			}

			return nil
		},
	}
}

func envelopeField(response json.RawMessage, name string) (json.RawMessage, error) {
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(response, &envelope); err != nil {
		return nil, err
	}
	value, ok := envelope[name]
	if !ok || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
		return nil, fmt.Errorf("response does not contain %s", name)
	}
	return value, nil
}
