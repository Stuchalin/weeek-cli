# Authentication and workspace

## Authentication

```text
weeek auth login [--token TOKEN]
weeek auth status
weeek auth logout
```

Prefer the `WEEEK_TOKEN` environment variable for automation. `auth login` reads
the token from `--token` or prompts on stdin and stores it in
`~/.config/weeek/config.json`. Avoid exposing a token in logs or command history.

```sh
export WEEEK_TOKEN="..."
weeek auth status
weeek me
```

On HTTP 401, ask the user to replace the invalid or expired token. Never print,
guess, or commit a token.

## Current user and workspace

```text
weeek me
weeek ws info
weeek ws members
```

`weeek me` combines the current user and workspace in one JSON response. Use
`ws members` to resolve user IDs before assignee, watcher, and time-entry commands.

```sh
weeek me | jq '.user, .workspace'
weeek ws members | jq '.'
```

## Tags

```text
weeek ws tag list
weeek ws tag create --name NAME
weeek ws tag update ID --name NAME --color COLOR
weeek ws tag delete ID
```

Create requires a name. Update requires both a name and a color. Resolve a tag ID
with `ws tag list` before update, delete, or use in `task --tags`.

```sh
weeek ws tag list
weeek ws tag create --name "release"
weeek ws tag update 10 --name "release-ready" --color '#22AA66'
```

## Attachments

```text
weeek ws attachment get ID
```

This command returns the attachment response as JSON. Obtain the ID from an API
response; there is no workspace attachment list command. Upload a task attachment
with `weeek task attachment upload TASK_ID --file PATH`.

## Utility commands

```text
weeek version
weeek help
weeek help COMMAND...
```

Use the help command when the installed binary may differ from this skill's
documented compatibility version.
