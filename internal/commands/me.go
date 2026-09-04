package commands

import (
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
			if ctx.Client == nil {
				return errors.New("api client is not configured")
			}

			user, _, err := ctx.Client.GetMe(context.Background())
			if err != nil {
				return err
			}
			workspace, err := ctx.Client.GetWorkspace(context.Background())
			if err != nil {
				return err
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
