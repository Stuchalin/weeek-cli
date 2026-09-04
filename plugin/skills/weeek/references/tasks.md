# Tasks

Use the list → ID → action pattern. All examples return JSON on stdout.

## List and inspect

```text
weeek task list [flags]
weeek task get ID
```

`task list` flags:

- `--responsible ID|me`: filter by responsible user; `me` resolves the current user.
- `--search TEXT`: search titles and descriptions.
- `--priority N`: priority from 0 through 3.
- `--tags ID[,ID...]`: tag IDs, comma-separated or repeated.
- `--type TYPE`: task type.
- `--completed=BOOL`: filter by completion state.
- `--day DATE`: task day value passed to the API.
- `--all=BOOL`: include completed and deleted tasks according to the API.
- `--sort FIELD`: API sort field.
- `--limit N`: maximum number of results; must not be negative.
- `--offset N`: number of results to skip; must not be negative.

Examples:

```sh
weeek task list --responsible me --completed=false --limit 50
weeek task list --search "incident" --priority 3 --tags 10,11
weeek task get 123
```

## Create and update

```text
weeek task create --title TITLE [--description TEXT] [--project ID [--column ID]] [--responsible ID|me] [--type TYPE] [--priority N] [--day DATE] [--parent ID]
weeek task update ID [--title TITLE] [--type TYPE] [--priority N] [--due DATE] [--tags ID[,ID...]]
```

`--title` is required for create. Create sends the API's required `locations`
array; `--column` therefore requires `--project`. Project, column, and parent IDs
must be positive. Update requires at least one field flag. Priority is 0 through
3. Update tags may be comma-separated or supplied with repeated `--tags` flags.
Change assignees with `task assignee add|remove`; the API does not expose task
description updates.

```sh
weeek task create --title "Ship CLI" --project 42 --responsible me --priority 2
weeek task update 123 --priority 2 --tags 10,11
```

## Lifecycle and placement

```text
weeek task complete ID
weeek task uncomplete ID
weeek task delete ID
weeek task move ID --parent ID
```

Use `task location add` below to place a task in a project or board column.

```sh
weeek task get 123
weeek task move 123 --parent 100
weeek task complete 123
```

## Comments

```text
weeek task comment list TASK_ID [--limit N] [--offset N]
weeek task comment add TASK_ID --markdown TEXT [--parent COMMENT_ID]
weeek task comment delete TASK_ID COMMENT_ID
```

Comment list limit is 0 through 100 and offset must not be negative. `--parent`
creates a reply and must be a positive comment ID. The API has no comment-update
command.

```sh
weeek task comment list 123 --limit 20
weeek task comment add 123 --markdown "Blocked by task #456"
```

## Assignees and watchers

```text
weeek task assignee add TASK_ID --user USER_ID[,USER_ID...]
weeek task assignee remove TASK_ID --user USER_ID[,USER_ID...]
weeek task watcher add TASK_ID --user USER_ID[,USER_ID...]
weeek task watcher remove TASK_ID --user USER_ID[,USER_ID...]
```

Resolve user IDs with `weeek ws members`. `--user` accepts comma-separated IDs or
repeated flags.

## Timers and time entries

```text
weeek task timer start ID
weeek task timer stop ID
weeek task time-entry create TASK_ID --user USER_ID --date DATE --duration MINUTES [--overtime]
weeek task time-entry update TASK_ID ENTRY_ID --user USER_ID --date DATE --duration MINUTES [--overtime]
weeek task time-entry delete TASK_ID ENTRY_ID
```

Dates use `YYYY-MM-DD`; duration is a positive number of minutes. `--overtime` is
a boolean switch. The API has no time-entry list command.

## Attachments and locations

```text
weeek task attachment upload TASK_ID --file PATH
weeek task location add TASK_ID --project PROJECT_ID [--column COLUMN_ID] [--after TASK_ID] [--before TASK_ID]
weeek task location remove TASK_ID --project PROJECT_ID
```

The file path and project ID are required. Optional column, after, and before IDs
must be positive. Attachment upload is the only task attachment operation; use
`weeek ws attachment get ID` for attachment metadata returned elsewhere.

## Agent workflows

Find and complete one matching task:

```sh
weeek task list --responsible me --search "release" --completed=false
weeek task get 123
weeek task complete 123
```

Add a task to a known project column:

```sh
weeek project list
weeek board list --project 42
weeek column list --board 7
weeek task location add 123 --project 42 --column 9
```
