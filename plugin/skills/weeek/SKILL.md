---
name: weeek
description: "Управляйте Weeek: задачи, таски, доски и спринты. Use for Weeek tasks, boards, sprints, projects, workspace data, and other weeek CLI operations."
---

# Weeek CLI

Use the `weeek` CLI to read and modify Weeek task-manager and workspace data.
The CLI compatibility version documented by this skill is `0.1.0`.

## Check the environment

1. Confirm the binary is available with `command -v weeek`.
2. Run `weeek me` before the first operation that needs the API.
3. If authentication is missing, tell the user to set `WEEEK_TOKEN` or run
   `weeek auth login`.
4. If the command returns HTTP 401, explain that the token is invalid or expired
   and ask the user to refresh it. Do not retry with guessed credentials.

All successful output is JSON on stdout. Pipe it to `jq` when selecting fields.
Errors are one-line JSON on stderr. Exit code 1 means an API, network, or token
failure; exit code 2 means invalid CLI usage.

## Work with IDs safely

Never invent a task, project, board, column, comment, user, or attachment ID.
For an operation on an existing entity:

1. Run the relevant `list` command with the narrowest useful filters.
2. Inspect the returned JSON and select the matching entity.
3. Use its actual ID in the requested action.
4. If multiple entities match, ask the user which one they mean.

Use `--responsible me` when the current user's tasks are requested. Prefer
read-only commands before create, update, move, complete, archive, or delete
commands. Confirm the target from fresh list/get output before a destructive
operation unless the user supplied an unambiguous ID.

## Command form

Write arguments in canonical order even though the parser accepts interspersed
flags:

```text
weeek <noun> <verb> [POSITIONAL_ID ...] [--flag VALUE ...]
```

Quote user-provided text. Comma-separated list flags may also be repeated. Use
`weeek help` for discovery and `weeek help <command>` for exact usage.

## Common commands

```sh
weeek me
weeek task list --responsible me --completed=false
weeek task list --search "release" --limit 20
weeek task get 123
weeek task create --title "Prepare release" --responsible me
weeek task update 123 --priority 2
weeek task comment add 123 --markdown "Ready for review"
weeek task complete 123
weeek project list
weeek board list --project 42
weeek column list --board 7
weeek ws members
weeek ws tag list
```

## Detailed references

- Read [tasks.md](references/tasks.md) for task CRUD, filters, comments,
  assignees, watchers, timers, time entries, attachments, and locations.
- Read [boards.md](references/boards.md) for boards and board columns.
- Read [projects.md](references/projects.md) for projects and portfolios.
- Read [workspace.md](references/workspace.md) for authentication, current-user,
  workspace, member, tag, and attachment commands.

Load only the reference needed for the current request.
