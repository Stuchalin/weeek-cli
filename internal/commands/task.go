package commands

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"strconv"
	"strings"

	"github.com/Stuchalin/weeek-cli/internal/api"
)

const (
	taskCreateUsage = "task create --title TITLE [--description TEXT] [--project ID [--column ID]] " +
		"[--responsible ID|me] [--type TYPE] [--priority N] [--day DATE] [--parent ID]"
	taskUpdateUsage = "task update ID [--title TITLE] [--type TYPE] [--priority N] " +
		"[--due DATE] [--tags ID[,ID...]]"
)

type stringListFlag []string

func (v *stringListFlag) String() string {
	return strings.Join(*v, ",")
}

func (v *stringListFlag) Set(value string) error {
	for _, item := range strings.Split(value, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			return errors.New("list values must not be empty")
		}
		*v = append(*v, item)
	}
	return nil
}

type optionalBool struct {
	value bool
	isSet bool
}

func (v *optionalBool) String() string {
	return strconv.FormatBool(v.value)
}

func (v *optionalBool) Set(value string) error {
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fmt.Errorf("invalid boolean value %q: %w", value, err)
	}
	v.value = parsed
	v.isSet = true
	return nil
}

func (v *optionalBool) IsBoolFlag() bool {
	return true
}

func registerTaskCommands(registry Registry) {
	registry.Register(taskListCommand())
	registry.Register(taskGetCommand())
	registry.Register(taskCreateCommand())
	registry.Register(taskUpdateCommand())
	registry.Register(taskDeleteCommand())
	registry.Register(taskCompleteCommand())
	registry.Register(taskUncompleteCommand())
	registry.Register(taskMoveCommand())
	registerTaskSubcommands(registry)
}

func taskListCommand() Command {
	return Command{
		Name:  "task list",
		Usage: "task list [flags]",
		Short: "List tasks",
		Run: func(ctx *Ctx) error {
			fs := flag.NewFlagSet("task list", flag.ContinueOnError)
			responsible := fs.String("responsible", "", "Responsible user ID or me")
			search := fs.String("search", "", "Search task titles and descriptions")
			priority := fs.Int("priority", 0, "Priority from 0 to 3")
			var tags stringListFlag
			fs.Var(&tags, "tags", "Tag IDs, comma-separated or repeated")
			taskType := fs.String("type", "", "Task type")
			var completed optionalBool
			fs.Var(&completed, "completed", "Filter by completion state")
			day := fs.String("day", "", "Task day")
			var all optionalBool
			fs.Var(&all, "all", "Include completed and deleted tasks")
			sortBy := fs.String("sort", "", "Sort field")
			limit := fs.Int("limit", 0, "Maximum tasks to return")
			offset := fs.Int("offset", 0, "Tasks to skip")
			if err := parseNoPositionals(fs, ctx.Args, "task list"); err != nil {
				return err
			}
			if *limit < 0 {
				return &usageError{err: errors.New("--limit must not be negative")}
			}
			if *offset < 0 {
				return &usageError{err: errors.New("--offset must not be negative")}
			}
			if flagWasSet(fs, "priority") && (*priority < 0 || *priority > 3) {
				return &usageError{err: errors.New("--priority must be between 0 and 3")}
			}

			client, err := requireClient(ctx)
			if err != nil {
				return err
			}
			resolvedResponsible, err := resolveResponsible(
				context.Background(),
				client,
				*responsible,
			)
			if err != nil {
				return err
			}

			filters := api.TaskFilters{
				Responsible: resolvedResponsible,
				Search:      *search,
				Tags:        tags,
				Type:        *taskType,
				Day:         *day,
				Sort:        *sortBy,
				Limit:       *limit,
				Offset:      *offset,
			}
			if flagWasSet(fs, "priority") {
				filters.Priority = priority
			}
			if completed.isSet {
				filters.Completed = &completed.value
			}
			if all.isSet {
				filters.All = &all.value
			}

			response, err := client.ListTasks(context.Background(), filters)
			if err != nil {
				return err
			}
			return writeRawJSON(ctx.Stdout, response)
		},
	}
}

func taskGetCommand() Command {
	return entityIDCommand(
		"task get",
		"Get a task",
		func(ctx context.Context, client *api.Client, id string) (json.RawMessage, error) {
			return client.GetTask(ctx, id)
		},
	)
}

func taskCreateCommand() Command {
	return Command{
		Name:  "task create",
		Usage: taskCreateUsage,
		Short: "Create a task",
		Run: func(ctx *Ctx) error {
			fs := flag.NewFlagSet("task create", flag.ContinueOnError)
			title := fs.String("title", "", "Task title")
			description := fs.String("description", "", "Task description")
			projectID := fs.Int64("project", 0, "Project ID")
			columnID := fs.Int64("column", 0, "Board column ID")
			responsible := fs.String("responsible", "", "Responsible user ID or me")
			taskType := fs.String("type", "", "Task type")
			priority := fs.Int("priority", 0, "Priority from 0 to 3")
			day := fs.String("day", "", "Task day")
			parentID := fs.Int64("parent", 0, "Parent task ID")
			if err := parseNoPositionals(fs, ctx.Args, "task create"); err != nil {
				return err
			}
			if strings.TrimSpace(*title) == "" {
				return &usageError{err: errors.New("--title is required")}
			}
			if err := validateTaskFields(
				fs,
				priority,
				projectID,
				columnID,
				parentID,
			); err != nil {
				return err
			}
			if flagWasSet(fs, "column") && !flagWasSet(fs, "project") {
				return &usageError{err: errors.New("--column requires --project")}
			}

			client, err := requireClient(ctx)
			if err != nil {
				return err
			}
			resolvedResponsible, err := resolveResponsible(
				context.Background(),
				client,
				*responsible,
			)
			if err != nil {
				return err
			}
			input := api.TaskCreate{Title: *title, Locations: []api.TaskCreateLocation{}}
			input.Description = stringPointerIfSet(fs, "description", description)
			input.Day = stringPointerIfSet(fs, "day", day)
			if flagWasSet(fs, "project") {
				input.Locations = append(input.Locations, api.TaskCreateLocation{
					ProjectID:     *projectID,
					BoardColumnID: int64PointerIfSet(fs, "column", columnID),
				})
			}
			input.ResponsibleID = valuePointerIfSet(fs, "responsible", resolvedResponsible)
			input.Type = stringPointerIfSet(fs, "type", taskType)
			input.Priority = intPointerIfSet(fs, "priority", priority)
			input.ParentID = int64PointerIfSet(fs, "parent", parentID)

			response, err := client.CreateTask(context.Background(), input)
			if err != nil {
				return err
			}
			return writeRawJSON(ctx.Stdout, response)
		},
	}
}

func taskUpdateCommand() Command {
	return Command{
		Name:  "task update",
		Usage: taskUpdateUsage,
		Short: "Update a task",
		Run: func(ctx *Ctx) error {
			fs := flag.NewFlagSet("task update", flag.ContinueOnError)
			title := fs.String("title", "", "Task title")
			taskType := fs.String("type", "", "Task type")
			priority := fs.Int("priority", 0, "Priority from 0 to 3")
			dueDate := fs.String("due", "", "Due date")
			var tags stringListFlag
			fs.Var(&tags, "tags", "Tag IDs, comma-separated or repeated")
			id, err := parseID(fs, ctx.Args, "task update")
			if err != nil {
				return err
			}
			if !anyFlagWasSet(fs) {
				return &usageError{err: errors.New("task update requires at least one field flag")}
			}
			if flagWasSet(fs, "priority") && (*priority < 0 || *priority > 3) {
				return &usageError{err: errors.New("--priority must be between 0 and 3")}
			}
			parsedTags, err := parseNumericTags(tags)
			if err != nil {
				return &usageError{err: err}
			}

			client, err := requireClient(ctx)
			if err != nil {
				return err
			}

			input := api.TaskUpdate{
				Title:    stringPointerIfSet(fs, "title", title),
				Type:     stringPointerIfSet(fs, "type", taskType),
				Priority: intPointerIfSet(fs, "priority", priority),
				DueDate:  stringPointerIfSet(fs, "due", dueDate),
			}
			if flagWasSet(fs, "tags") {
				input.Tags = &parsedTags
			}

			response, err := client.UpdateTask(context.Background(), id, input)
			if err != nil {
				return err
			}
			return writeRawJSON(ctx.Stdout, response)
		},
	}
}

func taskDeleteCommand() Command {
	return entityIDCommand(
		"task delete",
		"Delete a task",
		func(ctx context.Context, client *api.Client, id string) (json.RawMessage, error) {
			return client.DeleteTask(ctx, id)
		},
	)
}

func taskCompleteCommand() Command {
	return entityIDCommand(
		"task complete",
		"Complete a task",
		func(ctx context.Context, client *api.Client, id string) (json.RawMessage, error) {
			return client.Complete(ctx, id)
		},
	)
}

func taskUncompleteCommand() Command {
	return entityIDCommand(
		"task uncomplete",
		"Mark a task incomplete",
		func(ctx context.Context, client *api.Client, id string) (json.RawMessage, error) {
			return client.Uncomplete(ctx, id)
		},
	)
}

func taskMoveCommand() Command {
	return Command{
		Name:  "task move",
		Usage: "task move ID --parent ID",
		Short: "Move a task under a parent task",
		Run: func(ctx *Ctx) error {
			fs := flag.NewFlagSet("task move", flag.ContinueOnError)
			parentID := fs.Int64("parent", 0, "Parent task ID")
			id, err := parseID(fs, ctx.Args, "task move")
			if err != nil {
				return err
			}
			if !flagWasSet(fs, "parent") {
				return &usageError{err: errors.New("--parent is required")}
			}
			if *parentID <= 0 {
				return &usageError{err: errors.New("--parent must be positive")}
			}

			client, err := requireClient(ctx)
			if err != nil {
				return err
			}

			response, err := client.SetParent(context.Background(), id, *parentID)
			if err != nil {
				return err
			}
			return writeRawJSON(ctx.Stdout, response)
		},
	}
}

func resolveResponsible(ctx context.Context, client *api.Client, value string) (string, error) {
	if value != "me" {
		return value, nil
	}
	_, me, err := client.GetMe(ctx)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(me.ID) == "" {
		return "", errors.New("current user response does not contain id")
	}
	return me.ID, nil
}

func validateTaskFields(
	fs *flag.FlagSet,
	priority *int,
	ids ...*int64,
) error {
	if flagWasSet(fs, "priority") && (*priority < 0 || *priority > 3) {
		return &usageError{err: errors.New("--priority must be between 0 and 3")}
	}
	names := []string{"project", "column", "parent"}
	for index, value := range ids {
		if flagWasSet(fs, names[index]) && *value <= 0 {
			return &usageError{err: fmt.Errorf("--%s must be positive", names[index])}
		}
	}
	return nil
}

func parseNumericTags(values []string) ([]int64, error) {
	tags := make([]int64, 0, len(values))
	for _, value := range values {
		tag, err := strconv.ParseInt(value, 10, 64)
		if err != nil || tag <= 0 {
			return nil, fmt.Errorf("invalid tag id %q", value)
		}
		tags = append(tags, tag)
	}
	return tags, nil
}

func flagWasSet(fs *flag.FlagSet, name string) bool {
	isSet := false
	fs.Visit(func(flag *flag.Flag) {
		if flag.Name == name {
			isSet = true
		}
	})
	return isSet
}

func anyFlagWasSet(fs *flag.FlagSet) bool {
	isSet := false
	fs.Visit(func(*flag.Flag) {
		isSet = true
	})
	return isSet
}

func stringPointerIfSet(fs *flag.FlagSet, name string, value *string) *string {
	if !flagWasSet(fs, name) {
		return nil
	}
	return value
}

func valuePointerIfSet(fs *flag.FlagSet, name string, value string) *string {
	if !flagWasSet(fs, name) {
		return nil
	}
	return &value
}

func intPointerIfSet(fs *flag.FlagSet, name string, value *int) *int {
	if !flagWasSet(fs, name) {
		return nil
	}
	return value
}

func int64PointerIfSet(fs *flag.FlagSet, name string, value *int64) *int64 {
	if !flagWasSet(fs, name) {
		return nil
	}
	return value
}
