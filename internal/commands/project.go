package commands

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"strings"

	"github.com/Stuchalin/weeek-cli/internal/api"
)

func registerProjectCommands(registry Registry) {
	registry.Register(projectListCommand())
	registry.Register(projectGetCommand())
	registry.Register(projectCreateCommand())
	registry.Register(projectUpdateCommand())
	registry.Register(projectDeleteCommand())
	registry.Register(projectArchiveCommand())
	registry.Register(projectUnarchiveCommand())
	registry.Register(portfolioListCommand())
	registry.Register(portfolioGetCommand())
	registry.Register(portfolioCreateCommand())
	registry.Register(portfolioUpdateCommand())
	registry.Register(portfolioDeleteCommand())
}

func projectListCommand() Command {
	return readCommand(
		"project list",
		"List projects",
		func(ctx context.Context, client *api.Client) (json.RawMessage, error) {
			return client.ListProjects(ctx)
		},
	)
}

func projectGetCommand() Command {
	return entityIDCommand(
		"project get",
		"Get a project",
		func(ctx context.Context, client *api.Client, id string) (json.RawMessage, error) {
			return client.GetProject(ctx, id)
		},
	)
}

func projectCreateCommand() Command {
	return Command{
		Name: "project create",
		Usage: "project create --name NAME --private 0|1 [--logo URL] [--description TEXT] " +
			"[--portfolio ID]",
		Short: "Create a project",
		Run: func(ctx *Ctx) error {
			fs := flag.NewFlagSet("project create", flag.ContinueOnError)
			name := fs.String("name", "", "Project name")
			isPrivate := fs.Int("private", -1, "Privacy flag: 0 or 1")
			logo := fs.String("logo", "", "Project logo")
			description := fs.String("description", "", "Project description")
			portfolioID := fs.Int64("portfolio", 0, "Portfolio ID")
			if err := parseNoPositionals(fs, ctx.Args, "project create"); err != nil {
				return err
			}
			if err := validateProjectFields(fs, *name, *isPrivate, portfolioID); err != nil {
				return err
			}
			client, err := requireClient(ctx)
			if err != nil {
				return err
			}

			response, err := client.CreateProject(context.Background(), api.ProjectCreate{
				Name:        *name,
				IsPrivate:   *isPrivate,
				Logo:        stringPointerIfSet(fs, "logo", logo),
				Description: stringPointerIfSet(fs, "description", description),
				PortfolioID: int64PointerIfSet(fs, "portfolio", portfolioID),
			})
			if err != nil {
				return err
			}
			return writeRawJSON(ctx.Stdout, response)
		},
	}
}

func projectUpdateCommand() Command {
	return Command{
		Name:  "project update",
		Usage: "project update ID --name NAME --private 0|1 [--logo URL] [--color COLOR]",
		Short: "Update a project",
		Run: func(ctx *Ctx) error {
			fs := flag.NewFlagSet("project update", flag.ContinueOnError)
			name := fs.String("name", "", "Project name")
			isPrivate := fs.Int("private", -1, "Privacy flag: 0 or 1")
			logo := fs.String("logo", "", "Project logo")
			color := fs.String("color", "", "Project color")
			id, err := parseID(fs, ctx.Args, "project update")
			if err != nil {
				return err
			}
			if err := validateProjectFields(fs, *name, *isPrivate, nil); err != nil {
				return err
			}
			client, err := requireClient(ctx)
			if err != nil {
				return err
			}

			response, err := client.UpdateProject(context.Background(), id, api.ProjectUpdate{
				Name:      *name,
				IsPrivate: *isPrivate,
				Logo:      stringPointerIfSet(fs, "logo", logo),
				Color:     stringPointerIfSet(fs, "color", color),
			})
			if err != nil {
				return err
			}
			return writeRawJSON(ctx.Stdout, response)
		},
	}
}

func projectDeleteCommand() Command {
	return entityIDCommand(
		"project delete",
		"Delete a project",
		func(ctx context.Context, client *api.Client, id string) (json.RawMessage, error) {
			return client.DeleteProject(ctx, id)
		},
	)
}

func projectArchiveCommand() Command {
	return entityIDCommand(
		"project archive",
		"Archive a project",
		func(ctx context.Context, client *api.Client, id string) (json.RawMessage, error) {
			return client.ArchiveProject(ctx, id)
		},
	)
}

func projectUnarchiveCommand() Command {
	return entityIDCommand(
		"project unarchive",
		"Restore an archived project",
		func(ctx context.Context, client *api.Client, id string) (json.RawMessage, error) {
			return client.UnarchiveProject(ctx, id)
		},
	)
}

func portfolioListCommand() Command {
	return Command{
		Name:  "portfolio list",
		Usage: "portfolio list [--search TEXT] [--parent ID] [--limit N] [--offset N]",
		Short: "List portfolios",
		Run: func(ctx *Ctx) error {
			fs := flag.NewFlagSet("portfolio list", flag.ContinueOnError)
			search := fs.String("search", "", "Search portfolio names")
			parentID := fs.Int64("parent", 0, "Parent portfolio ID")
			limit := fs.Int("limit", 0, "Maximum portfolios to return")
			offset := fs.Int("offset", 0, "Portfolios to skip")
			if err := parseNoPositionals(fs, ctx.Args, "portfolio list"); err != nil {
				return err
			}
			if flagWasSet(fs, "parent") && *parentID <= 0 {
				return &usageError{err: errors.New("--parent must be positive")}
			}
			if *limit < 0 || *limit > 100 {
				return &usageError{err: errors.New("--limit must be between 0 and 100")}
			}
			if *offset < 0 {
				return &usageError{err: errors.New("--offset must not be negative")}
			}
			client, err := requireClient(ctx)
			if err != nil {
				return err
			}

			response, err := client.ListPortfolios(context.Background(), api.PortfolioFilters{
				Search:   *search,
				ParentID: int64PointerIfSet(fs, "parent", parentID),
				Limit:    *limit,
				Offset:   *offset,
			})
			if err != nil {
				return err
			}
			return writeRawJSON(ctx.Stdout, response)
		},
	}
}

func portfolioGetCommand() Command {
	return entityIDCommand(
		"portfolio get",
		"Get a portfolio",
		func(ctx context.Context, client *api.Client, id string) (json.RawMessage, error) {
			return client.GetPortfolio(ctx, id)
		},
	)
}

func portfolioCreateCommand() Command {
	return Command{
		Name:  "portfolio create",
		Usage: "portfolio create --name NAME [--parent ID]",
		Short: "Create a portfolio",
		Run: func(ctx *Ctx) error {
			fs := flag.NewFlagSet("portfolio create", flag.ContinueOnError)
			name := fs.String("name", "", "Portfolio name")
			parentID := fs.Int64("parent", 0, "Parent portfolio ID")
			if err := parseNoPositionals(fs, ctx.Args, "portfolio create"); err != nil {
				return err
			}
			if strings.TrimSpace(*name) == "" {
				return &usageError{err: errors.New("--name is required")}
			}
			if flagWasSet(fs, "parent") && *parentID <= 0 {
				return &usageError{err: errors.New("--parent must be positive")}
			}
			client, err := requireClient(ctx)
			if err != nil {
				return err
			}

			response, err := client.CreatePortfolio(context.Background(), api.PortfolioCreate{
				Name:     *name,
				ParentID: int64PointerIfSet(fs, "parent", parentID),
			})
			if err != nil {
				return err
			}
			return writeRawJSON(ctx.Stdout, response)
		},
	}
}

func portfolioUpdateCommand() Command {
	return namedEntityUpdateCommand(
		"portfolio update",
		"Update a portfolio",
		func(ctx context.Context, client *api.Client, id, name string) (json.RawMessage, error) {
			return client.UpdatePortfolio(ctx, id, api.PortfolioUpdate{Name: name})
		},
	)
}

func portfolioDeleteCommand() Command {
	return entityIDCommand(
		"portfolio delete",
		"Delete a portfolio",
		func(ctx context.Context, client *api.Client, id string) (json.RawMessage, error) {
			return client.DeletePortfolio(ctx, id)
		},
	)
}

func validateProjectFields(fs *flag.FlagSet, name string, isPrivate int, portfolioID *int64) error {
	if strings.TrimSpace(name) == "" {
		return &usageError{err: errors.New("--name is required")}
	}
	if !flagWasSet(fs, "private") || (isPrivate != 0 && isPrivate != 1) {
		return &usageError{err: errors.New("--private must be 0 or 1")}
	}
	if portfolioID != nil && flagWasSet(fs, "portfolio") && *portfolioID <= 0 {
		return &usageError{err: errors.New("--portfolio must be positive")}
	}
	return nil
}
