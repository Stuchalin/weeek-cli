package commands

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"strings"

	"github.com/Stuchalin/weeek-cli/internal/api"
)

func registerWorkspaceCommands(registry Registry) {
	registry.Register(workspaceInfoCommand())
	registry.Register(workspaceMembersCommand())
	registry.Register(workspaceTagListCommand())
	registry.Register(workspaceTagCreateCommand())
	registry.Register(workspaceTagUpdateCommand())
	registry.Register(workspaceTagDeleteCommand())
	registry.Register(workspaceAttachmentGetCommand())
}

func workspaceInfoCommand() Command {
	return readCommand(
		"ws info",
		"Show workspace information",
		func(ctx context.Context, client *api.Client) (json.RawMessage, error) {
			return client.GetWorkspace(ctx)
		},
	)
}

func workspaceMembersCommand() Command {
	return readCommand(
		"ws members",
		"List workspace members",
		func(ctx context.Context, client *api.Client) (json.RawMessage, error) {
			return client.GetMembers(ctx)
		},
	)
}

func workspaceTagListCommand() Command {
	return readCommand(
		"ws tag list",
		"List workspace tags",
		func(ctx context.Context, client *api.Client) (json.RawMessage, error) {
			return client.ListTags(ctx)
		},
	)
}

func workspaceTagCreateCommand() Command {
	return Command{
		Name:  "ws tag create",
		Usage: "ws tag create --name NAME",
		Short: "Create a workspace tag",
		Run: func(ctx *Ctx) error {
			fs := flag.NewFlagSet("ws tag create", flag.ContinueOnError)
			name := fs.String("name", "", "Tag name")
			if err := parseNoPositionals(fs, ctx.Args, "ws tag create"); err != nil {
				return err
			}
			if strings.TrimSpace(*name) == "" {
				return &usageError{err: errors.New("--name is required")}
			}
			client, err := requireClient(ctx)
			if err != nil {
				return err
			}

			response, err := client.CreateTag(
				context.Background(),
				api.TagCreate{Title: *name},
			)
			if err != nil {
				return err
			}

			return writeRawJSON(ctx.Stdout, response)
		},
	}
}

func workspaceTagUpdateCommand() Command {
	return Command{
		Name:  "ws tag update",
		Usage: "ws tag update ID --name NAME --color COLOR",
		Short: "Update a workspace tag",
		Run: func(ctx *Ctx) error {
			fs := flag.NewFlagSet("ws tag update", flag.ContinueOnError)
			name := fs.String("name", "", "Tag name")
			color := fs.String("color", "", "Tag color in #RRGGBB format")
			id, err := parseID(fs, ctx.Args, "ws tag update")
			if err != nil {
				return err
			}
			if strings.TrimSpace(*name) == "" {
				return &usageError{err: errors.New("--name is required")}
			}
			if strings.TrimSpace(*color) == "" {
				return &usageError{err: errors.New("--color is required")}
			}
			client, err := requireClient(ctx)
			if err != nil {
				return err
			}

			response, err := client.UpdateTag(
				context.Background(),
				id,
				api.TagUpdate{
					Title: *name,
					Color: *color,
				},
			)
			if err != nil {
				return err
			}

			return writeRawJSON(ctx.Stdout, response)
		},
	}
}

func workspaceTagDeleteCommand() Command {
	return Command{
		Name:  "ws tag delete",
		Usage: "ws tag delete ID",
		Short: "Delete a workspace tag",
		Run: func(ctx *Ctx) error {
			fs := flag.NewFlagSet("ws tag delete", flag.ContinueOnError)
			id, err := parseID(fs, ctx.Args, "ws tag delete")
			if err != nil {
				return err
			}
			client, err := requireClient(ctx)
			if err != nil {
				return err
			}

			response, err := client.DeleteTag(context.Background(), id)
			if err != nil {
				return err
			}

			return writeRawJSON(ctx.Stdout, response)
		},
	}
}

func workspaceAttachmentGetCommand() Command {
	return Command{
		Name:  "ws attachment get",
		Usage: "ws attachment get ID",
		Short: "Get workspace attachment metadata",
		Run: func(ctx *Ctx) error {
			fs := flag.NewFlagSet("ws attachment get", flag.ContinueOnError)
			id, err := parseID(fs, ctx.Args, "ws attachment get")
			if err != nil {
				return err
			}
			client, err := requireClient(ctx)
			if err != nil {
				return err
			}

			response, err := client.GetAttachment(context.Background(), id)
			if err != nil {
				return err
			}

			return writeRawJSON(ctx.Stdout, response)
		},
	}
}

func requireClient(ctx *Ctx) (*api.Client, error) {
	if ctx.Client != nil {
		return ctx.Client, nil
	}
	if ctx.ResolveToken == nil || ctx.NewAPIClient == nil {
		return nil, errors.New("api client is not configured")
	}
	token, err := ctx.ResolveToken()
	if err != nil {
		return nil, err
	}
	ctx.Client = ctx.NewAPIClient(token)

	return ctx.Client, nil
}

func parseNoPositionals(fs *flag.FlagSet, args []string, name string) error {
	positionals, err := parseAll(fs, args)
	if err != nil {
		return err
	}
	if len(positionals) != 0 {
		return &usageError{err: errors.New(name + " does not accept arguments")}
	}
	return nil
}

func parseID(fs *flag.FlagSet, args []string, name string) (string, error) {
	positionals, err := parseAll(fs, args)
	if err != nil {
		return "", err
	}
	if len(positionals) != 1 || strings.TrimSpace(positionals[0]) == "" {
		return "", &usageError{err: errors.New(name + " requires one ID")}
	}
	return positionals[0], nil
}
