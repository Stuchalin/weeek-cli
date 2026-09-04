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
			client, err := requireClient(ctx)
			if err != nil {
				return err
			}

			user, _, err := client.GetMe(context.Background())
			if err != nil {
				return err
			}
			workspace, err := client.GetWorkspace(context.Background())
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
