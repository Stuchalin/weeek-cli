package commands

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"strings"

	"github.com/Stuchalin/weeek-cli/internal/api"
)

type taskMemberRequest func(
	context.Context,
	*api.Client,
	string,
	[]string,
) (json.RawMessage, error)

func registerTaskSubcommands(registry Registry) {
	registry.Register(taskCommentListCommand())
	registry.Register(taskCommentAddCommand())
	registry.Register(taskCommentDeleteCommand())
	registry.Register(taskMemberCommand(
		"task assignee add",
		"Add task assignees",
		func(ctx context.Context, client *api.Client, taskID string, users []string) (json.RawMessage, error) {
			return client.AddTaskAssignees(ctx, taskID, users)
		},
	))
	registry.Register(taskMemberCommand(
		"task assignee remove",
		"Remove task assignees",
		func(ctx context.Context, client *api.Client, taskID string, users []string) (json.RawMessage, error) {
			return client.RemoveTaskAssignees(ctx, taskID, users)
		},
	))
	registry.Register(taskMemberCommand(
		"task watcher add",
		"Add task watchers",
		func(ctx context.Context, client *api.Client, taskID string, users []string) (json.RawMessage, error) {
			return client.AddTaskWatchers(ctx, taskID, users)
		},
	))
	registry.Register(taskMemberCommand(
		"task watcher remove",
		"Remove task watchers",
		func(ctx context.Context, client *api.Client, taskID string, users []string) (json.RawMessage, error) {
			return client.RemoveTaskWatchers(ctx, taskID, users)
		},
	))
	registry.Register(taskIDCommand(
		"task timer start",
		"Start a task timer",
		func(ctx context.Context, client *api.Client, taskID string) (json.RawMessage, error) {
			return client.StartTaskTimer(ctx, taskID)
		},
	))
	registry.Register(taskIDCommand(
		"task timer stop",
		"Stop a task timer",
		func(ctx context.Context, client *api.Client, taskID string) (json.RawMessage, error) {
			return client.StopTaskTimer(ctx, taskID)
		},
	))
	registry.Register(taskTimeEntryCommand(false))
	registry.Register(taskTimeEntryCommand(true))
	registry.Register(taskTimeEntryDeleteCommand())
	registry.Register(taskAttachmentUploadCommand())
	registry.Register(taskLocationAddCommand())
	registry.Register(taskLocationRemoveCommand())
}

func taskCommentListCommand() Command {
	return Command{
		Name:  "task comment list",
		Usage: "task comment list TASK_ID [--limit N] [--offset N]",
		Short: "List task comments",
		Run: func(ctx *Ctx) error {
			fs := flag.NewFlagSet("task comment list", flag.ContinueOnError)
			limit := fs.Int("limit", 0, "Maximum comments to return")
			offset := fs.Int("offset", 0, "Comments to skip")
			taskID, err := parseID(fs, ctx.Args, "task comment list")
			if err != nil {
				return err
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
			response, err := client.ListTaskComments(
				context.Background(),
				taskID,
				api.CommentFilters{Limit: *limit, Offset: *offset},
			)
			if err != nil {
				return err
			}
			return writeRawJSON(ctx.Stdout, response)
		},
	}
}

func taskCommentAddCommand() Command {
	return Command{
		Name:  "task comment add",
		Usage: "task comment add TASK_ID --markdown TEXT [--parent COMMENT_ID]",
		Short: "Add a task comment",
		Run: func(ctx *Ctx) error {
			fs := flag.NewFlagSet("task comment add", flag.ContinueOnError)
			markdown := fs.String("markdown", "", "Comment text in Markdown")
			parentID := fs.Int64("parent", 0, "Parent comment ID")
			taskID, err := parseID(fs, ctx.Args, "task comment add")
			if err != nil {
				return err
			}
			if strings.TrimSpace(*markdown) == "" {
				return &usageError{err: errors.New("--markdown is required")}
			}
			if flagWasSet(fs, "parent") && *parentID <= 0 {
				return &usageError{err: errors.New("--parent must be positive")}
			}

			client, err := requireClient(ctx)
			if err != nil {
				return err
			}
			response, err := client.AddTaskComment(
				context.Background(),
				taskID,
				api.CommentCreate{
					Markdown: *markdown,
					ParentID: int64PointerIfSet(fs, "parent", parentID),
				},
			)
			if err != nil {
				return err
			}
			return writeRawJSON(ctx.Stdout, response)
		},
	}
}

func taskCommentDeleteCommand() Command {
	return taskTwoIDCommand(
		"task comment delete",
		"Delete a task comment",
		"TASK_ID COMMENT_ID",
		func(ctx context.Context, client *api.Client, taskID string, commentID string) (json.RawMessage, error) {
			return client.DeleteTaskComment(ctx, taskID, commentID)
		},
	)
}

func taskMemberCommand(name string, short string, request taskMemberRequest) Command {
	return Command{
		Name:  name,
		Usage: name + " TASK_ID --user USER_ID[,USER_ID...]",
		Short: short,
		Run: func(ctx *Ctx) error {
			fs := flag.NewFlagSet(name, flag.ContinueOnError)
			users := stringListFlag{}
			fs.Var(&users, "user", "User IDs, comma-separated or repeated")
			taskID, err := parseID(fs, ctx.Args, name)
			if err != nil {
				return err
			}
			if len(users) == 0 {
				return &usageError{err: errors.New("--user is required")}
			}

			client, err := requireClient(ctx)
			if err != nil {
				return err
			}
			response, err := request(context.Background(), client, taskID, users)
			if err != nil {
				return err
			}
			return writeRawJSON(ctx.Stdout, response)
		},
	}
}

func taskTimeEntryCommand(isUpdate bool) Command {
	name := "task time-entry create"
	short := "Create a task time entry"
	positionals := "TASK_ID"
	if isUpdate {
		name = "task time-entry update"
		short = "Update a task time entry"
		positionals = "TASK_ID ENTRY_ID"
	}

	return Command{
		Name: name,
		Usage: name + " " + positionals +
			" --user USER_ID --date DATE --duration MINUTES [--overtime]",
		Short: short,
		Run: func(ctx *Ctx) error {
			fs := flag.NewFlagSet(name, flag.ContinueOnError)
			userID := fs.String("user", "", "User ID")
			date := fs.String("date", "", "Entry date in YYYY-MM-DD format")
			duration := fs.Int("duration", 0, "Duration in minutes")
			isOvertime := fs.Bool("overtime", false, "Mark the entry as overtime")
			idCount := 1
			if isUpdate {
				idCount = 2
			}
			ids, err := parseIDs(fs, ctx.Args, name, idCount)
			if err != nil {
				return err
			}
			if strings.TrimSpace(*userID) == "" {
				return &usageError{err: errors.New("--user is required")}
			}
			if strings.TrimSpace(*date) == "" {
				return &usageError{err: errors.New("--date is required")}
			}
			if *duration <= 0 {
				return &usageError{err: errors.New("--duration must be positive")}
			}

			client, err := requireClient(ctx)
			if err != nil {
				return err
			}
			input := api.TimeEntryInput{
				UserID:     *userID,
				IsOvertime: *isOvertime,
				Date:       *date,
				Duration:   *duration,
			}
			var response json.RawMessage
			if isUpdate {
				response, err = client.UpdateTaskTimeEntry(
					context.Background(),
					ids[0],
					ids[1],
					input,
				)
			} else {
				response, err = client.CreateTaskTimeEntry(
					context.Background(),
					ids[0],
					input,
				)
			}
			if err != nil {
				return err
			}
			return writeRawJSON(ctx.Stdout, response)
		},
	}
}

func taskTimeEntryDeleteCommand() Command {
	return taskTwoIDCommand(
		"task time-entry delete",
		"Delete a task time entry",
		"TASK_ID ENTRY_ID",
		func(ctx context.Context, client *api.Client, taskID string, entryID string) (json.RawMessage, error) {
			return client.DeleteTaskTimeEntry(ctx, taskID, entryID)
		},
	)
}

func taskAttachmentUploadCommand() Command {
	return Command{
		Name:  "task attachment upload",
		Usage: "task attachment upload TASK_ID --file PATH",
		Short: "Upload a task attachment",
		Run: func(ctx *Ctx) error {
			fs := flag.NewFlagSet("task attachment upload", flag.ContinueOnError)
			filePath := fs.String("file", "", "Path to the file to upload")
			taskID, err := parseID(fs, ctx.Args, "task attachment upload")
			if err != nil {
				return err
			}
			if strings.TrimSpace(*filePath) == "" {
				return &usageError{err: errors.New("--file is required")}
			}

			client, err := requireClient(ctx)
			if err != nil {
				return err
			}
			response, err := client.UploadTaskAttachment(
				context.Background(),
				taskID,
				*filePath,
			)
			if err != nil {
				return err
			}
			return writeRawJSON(ctx.Stdout, response)
		},
	}
}

func taskLocationAddCommand() Command {
	return Command{
		Name: "task location add",
		Usage: "task location add TASK_ID --project PROJECT_ID [--column COLUMN_ID] " +
			"[--after TASK_ID] [--before TASK_ID]",
		Short: "Add a task to a project",
		Run: func(ctx *Ctx) error {
			fs := flag.NewFlagSet("task location add", flag.ContinueOnError)
			projectID := fs.Int64("project", 0, "Project ID")
			columnID := fs.Int64("column", 0, "Board column ID")
			afterID := fs.Int64("after", 0, "Place after task ID")
			beforeID := fs.Int64("before", 0, "Place before task ID")
			taskID, err := parseID(fs, ctx.Args, "task location add")
			if err != nil {
				return err
			}
			if !flagWasSet(fs, "project") || *projectID <= 0 {
				return &usageError{err: errors.New("--project must be positive")}
			}
			if flagWasSet(fs, "column") && *columnID <= 0 {
				return &usageError{err: errors.New("--column must be positive")}
			}
			if flagWasSet(fs, "after") && *afterID <= 0 {
				return &usageError{err: errors.New("--after must be positive")}
			}
			if flagWasSet(fs, "before") && *beforeID <= 0 {
				return &usageError{err: errors.New("--before must be positive")}
			}

			client, err := requireClient(ctx)
			if err != nil {
				return err
			}
			response, err := client.AddTaskLocation(
				context.Background(),
				taskID,
				api.LocationCreate{
					ProjectID:     *projectID,
					BoardColumnID: int64PointerIfSet(fs, "column", columnID),
					After:         int64PointerIfSet(fs, "after", afterID),
					Before:        int64PointerIfSet(fs, "before", beforeID),
				},
			)
			if err != nil {
				return err
			}
			return writeRawJSON(ctx.Stdout, response)
		},
	}
}

func taskLocationRemoveCommand() Command {
	return Command{
		Name:  "task location remove",
		Usage: "task location remove TASK_ID --project PROJECT_ID",
		Short: "Remove a task from a project",
		Run: func(ctx *Ctx) error {
			fs := flag.NewFlagSet("task location remove", flag.ContinueOnError)
			projectID := fs.Int64("project", 0, "Project ID")
			taskID, err := parseID(fs, ctx.Args, "task location remove")
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
			response, err := client.RemoveTaskLocation(
				context.Background(),
				taskID,
				*projectID,
			)
			if err != nil {
				return err
			}
			return writeRawJSON(ctx.Stdout, response)
		},
	}
}

func taskTwoIDCommand(
	name string,
	short string,
	positionals string,
	request func(context.Context, *api.Client, string, string) (json.RawMessage, error),
) Command {
	return Command{
		Name:  name,
		Usage: name + " " + positionals,
		Short: short,
		Run: func(ctx *Ctx) error {
			fs := flag.NewFlagSet(name, flag.ContinueOnError)
			ids, err := parseIDs(fs, ctx.Args, name, 2)
			if err != nil {
				return err
			}
			client, err := requireClient(ctx)
			if err != nil {
				return err
			}
			response, err := request(
				context.Background(),
				client,
				ids[0],
				ids[1],
			)
			if err != nil {
				return err
			}
			return writeRawJSON(ctx.Stdout, response)
		},
	}
}

func parseIDs(
	fs *flag.FlagSet,
	args []string,
	name string,
	count int,
) ([]string, error) {
	positionals, err := parseAll(fs, args)
	if err != nil {
		return nil, err
	}
	if len(positionals) != count {
		return nil, &usageError{err: errors.New(name + " requires the documented IDs")}
	}
	for _, id := range positionals {
		if strings.TrimSpace(id) == "" {
			return nil, &usageError{err: errors.New(name + " requires non-empty IDs")}
		}
	}
	return positionals, nil
}
