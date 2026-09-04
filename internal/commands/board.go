package commands

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"strings"

	"github.com/Stuchalin/weeek-cli/internal/api"
)

func registerBoardCommands(registry Registry) {
	registry.Register(boardListCommand())
	registry.Register(boardGetCommand())
	registry.Register(boardCreateCommand())
	registry.Register(boardUpdateCommand())
	registry.Register(boardDeleteCommand())
	registry.Register(boardMoveCommand())
	registry.Register(columnListCommand())
	registry.Register(columnCreateCommand())
	registry.Register(columnUpdateCommand())
	registry.Register(columnDeleteCommand())
	registry.Register(columnMoveCommand())
}

func boardListCommand() Command {
	return Command{
		Name:  "board list",
		Usage: "board list --project ID",
		Short: "List project boards",
		Run: func(ctx *Ctx) error {
			fs := flag.NewFlagSet("board list", flag.ContinueOnError)
			projectID := fs.Int64("project", 0, "Project ID")
			if err := parseNoPositionals(fs, ctx.Args, "board list"); err != nil {
				return err
			}
			if !flagWasSet(fs, "project") || *projectID <= 0 {
				return &usageError{err: errors.New("--project must be positive")}
			}
			client, err := requireClient(ctx)
			if err != nil {
				return err
			}

			response, err := client.ListBoards(context.Background(), *projectID)
			if err != nil {
				return err
			}
			return writeRawJSON(ctx.Stdout, response)
		},
	}
}

func boardGetCommand() Command {
	return Command{
		Name:  "board get",
		Usage: "board get ID --project ID",
		Short: "Get a project board",
		Run: func(ctx *Ctx) error {
			fs := flag.NewFlagSet("board get", flag.ContinueOnError)
			projectID := fs.Int64("project", 0, "Project ID")
			id, err := parseID(fs, ctx.Args, "board get")
			if err != nil {
				return err
			}
			if !flagWasSet(fs, "project") || *projectID <= 0 {
				return &usageError{err: errors.New("--project must be positive")}
			}
			client, err := requireClient(ctx)
			if err != nil {
				return err
			}

			response, err := client.GetBoard(context.Background(), *projectID, id)
			if err != nil {
				return err
			}
			return writeRawJSON(ctx.Stdout, response)
		},
	}
}

func boardCreateCommand() Command {
	return Command{
		Name:  "board create",
		Usage: "board create --name NAME --project ID",
		Short: "Create a board",
		Run: func(ctx *Ctx) error {
			fs := flag.NewFlagSet("board create", flag.ContinueOnError)
			name := fs.String("name", "", "Board name")
			projectID := fs.Int64("project", 0, "Project ID")
			if err := parseNoPositionals(fs, ctx.Args, "board create"); err != nil {
				return err
			}
			if strings.TrimSpace(*name) == "" {
				return &usageError{err: errors.New("--name is required")}
			}
			if !flagWasSet(fs, "project") || *projectID <= 0 {
				return &usageError{err: errors.New("--project must be positive")}
			}
			client, err := requireClient(ctx)
			if err != nil {
				return err
			}

			response, err := client.CreateBoard(context.Background(), api.BoardCreate{
				Name:      *name,
				ProjectID: *projectID,
			})
			if err != nil {
				return err
			}
			return writeRawJSON(ctx.Stdout, response)
		},
	}
}

func boardUpdateCommand() Command {
	return namedEntityUpdateCommand(
		"board update",
		"Update a board",
		func(ctx context.Context, client *api.Client, id, name string) (json.RawMessage, error) {
			return client.UpdateBoard(ctx, id, api.BoardUpdate{Name: name})
		},
	)
}

func boardDeleteCommand() Command {
	return boardEntityIDCommand(
		"board delete",
		"Delete a board",
		func(ctx context.Context, client *api.Client, id string) (json.RawMessage, error) {
			return client.DeleteBoard(ctx, id)
		},
	)
}

func boardMoveCommand() Command {
	return moveEntityCommand(
		"board move",
		"Move a board",
		func(ctx context.Context, client *api.Client, id string, after *int64) (json.RawMessage, error) {
			return client.MoveBoard(ctx, id, api.BoardMove{UpperBoardID: after})
		},
	)
}

func columnListCommand() Command {
	return Command{
		Name:  "column list",
		Usage: "column list [--board ID]",
		Short: "List board columns",
		Run: func(ctx *Ctx) error {
			fs := flag.NewFlagSet("column list", flag.ContinueOnError)
			boardID := fs.Int64("board", 0, "Board ID")
			if err := parseNoPositionals(fs, ctx.Args, "column list"); err != nil {
				return err
			}
			if flagWasSet(fs, "board") && *boardID <= 0 {
				return &usageError{err: errors.New("--board must be positive")}
			}
			client, err := requireClient(ctx)
			if err != nil {
				return err
			}

			response, err := client.ListBoardColumns(
				context.Background(),
				int64PointerIfSet(fs, "board", boardID),
			)
			if err != nil {
				return err
			}
			return writeRawJSON(ctx.Stdout, response)
		},
	}
}

func columnCreateCommand() Command {
	return Command{
		Name:  "column create",
		Usage: "column create --name NAME --board ID",
		Short: "Create a board column",
		Run: func(ctx *Ctx) error {
			fs := flag.NewFlagSet("column create", flag.ContinueOnError)
			name := fs.String("name", "", "Column name")
			boardID := fs.Int64("board", 0, "Board ID")
			if err := parseNoPositionals(fs, ctx.Args, "column create"); err != nil {
				return err
			}
			if strings.TrimSpace(*name) == "" {
				return &usageError{err: errors.New("--name is required")}
			}
			if !flagWasSet(fs, "board") || *boardID <= 0 {
				return &usageError{err: errors.New("--board must be positive")}
			}
			client, err := requireClient(ctx)
			if err != nil {
				return err
			}

			response, err := client.CreateBoardColumn(context.Background(), api.BoardColumnCreate{
				Name:    *name,
				BoardID: *boardID,
			})
			if err != nil {
				return err
			}
			return writeRawJSON(ctx.Stdout, response)
		},
	}
}

func columnUpdateCommand() Command {
	return namedEntityUpdateCommand(
		"column update",
		"Update a board column",
		func(ctx context.Context, client *api.Client, id, name string) (json.RawMessage, error) {
			return client.UpdateBoardColumn(ctx, id, api.BoardColumnUpdate{Name: name})
		},
	)
}

func columnDeleteCommand() Command {
	return boardEntityIDCommand(
		"column delete",
		"Delete a board column",
		func(ctx context.Context, client *api.Client, id string) (json.RawMessage, error) {
			return client.DeleteBoardColumn(ctx, id)
		},
	)
}

func columnMoveCommand() Command {
	return moveEntityCommand(
		"column move",
		"Move a board column",
		func(ctx context.Context, client *api.Client, id string, after *int64) (json.RawMessage, error) {
			return client.MoveBoardColumn(ctx, id, api.BoardColumnMove{UpperBoardColumnID: after})
		},
	)
}

func namedEntityUpdateCommand(
	name string,
	short string,
	request func(context.Context, *api.Client, string, string) (json.RawMessage, error),
) Command {
	return Command{
		Name:  name,
		Usage: name + " ID --name NAME",
		Short: short,
		Run: func(ctx *Ctx) error {
			fs := flag.NewFlagSet(name, flag.ContinueOnError)
			entityName := fs.String("name", "", "New name")
			id, err := parseID(fs, ctx.Args, name)
			if err != nil {
				return err
			}
			if strings.TrimSpace(*entityName) == "" {
				return &usageError{err: errors.New("--name is required")}
			}
			client, err := requireClient(ctx)
			if err != nil {
				return err
			}

			response, err := request(context.Background(), client, id, *entityName)
			if err != nil {
				return err
			}
			return writeRawJSON(ctx.Stdout, response)
		},
	}
}

func moveEntityCommand(
	name string,
	short string,
	request func(context.Context, *api.Client, string, *int64) (json.RawMessage, error),
) Command {
	return Command{
		Name:  name,
		Usage: name + " ID [--after ID]",
		Short: short,
		Run: func(ctx *Ctx) error {
			fs := flag.NewFlagSet(name, flag.ContinueOnError)
			afterID := fs.Int64("after", 0, "ID after which to place the item")
			id, err := parseID(fs, ctx.Args, name)
			if err != nil {
				return err
			}
			if flagWasSet(fs, "after") && *afterID <= 0 {
				return &usageError{err: errors.New("--after must be positive")}
			}
			client, err := requireClient(ctx)
			if err != nil {
				return err
			}

			response, err := request(
				context.Background(),
				client,
				id,
				int64PointerIfSet(fs, "after", afterID),
			)
			if err != nil {
				return err
			}
			return writeRawJSON(ctx.Stdout, response)
		},
	}
}

func boardEntityIDCommand(
	name string,
	short string,
	request func(context.Context, *api.Client, string) (json.RawMessage, error),
) Command {
	return Command{
		Name:  name,
		Usage: name + " ID",
		Short: short,
		Run: func(ctx *Ctx) error {
			fs := flag.NewFlagSet(name, flag.ContinueOnError)
			id, err := parseID(fs, ctx.Args, name)
			if err != nil {
				return err
			}
			client, err := requireClient(ctx)
			if err != nil {
				return err
			}

			response, err := request(context.Background(), client, id)
			if err != nil {
				return err
			}
			return writeRawJSON(ctx.Stdout, response)
		},
	}
}
