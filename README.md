# weeek-cli

`weeek` is a zero-dependency Go CLI for the public Weeek API. It exposes tasks,
boards, projects, portfolios, and workspace data as predictable JSON so AI agents
and shell scripts can use Weeek without building raw HTTP requests.

```sh
weeek task list --responsible me --completed=false \
  | jq '.tasks[] | {id, title}'
```

## Installation

### Pre-built binaries

Download the binary for your platform and its `.sha256` file from
[GitHub Releases](https://github.com/Stuchalin/weeek-cli/releases/latest):

| Platform | Architecture | Binary |
| --- | --- | --- |
| macOS | Apple Silicon | `weeek_darwin_arm64` |
| macOS | Intel | `weeek_darwin_amd64` |
| Linux | arm64 | `weeek_linux_arm64` |
| Linux | amd64 | `weeek_linux_amd64` |

Rename the downloaded binary to `weeek`, make it executable, and put it on your
`PATH`. For example, on macOS with Apple Silicon:

```sh
mv weeek_darwin_arm64 weeek
chmod +x weeek
sudo mv weeek /usr/local/bin/weeek
weeek version
```

### Install with Go

Go 1.27 or later is required to install from source:

```sh
go install github.com/Stuchalin/weeek-cli/cmd/weeek@latest
```

Make sure the Go binary directory (usually `$HOME/go/bin`) is on your `PATH`.

## Agent plugin

The repository is a plugin marketplace for both Claude Code and Codex. The
plugin teaches an agent how to discover Weeek IDs, select commands, and avoid
inventing task or project identifiers. Install the CLI first, then add the
marketplace and plugin.

### Claude Code

```sh
claude plugin marketplace add Stuchalin/weeek-cli
claude plugin install weeek@weeek-cli
```

### Codex

```sh
codex plugin marketplace add Stuchalin/weeek-cli
codex plugin add weeek@weeek-cli
```

## Authentication

Create an access token in the API section of your Weeek workspace settings, as
described in the [Weeek Public API documentation](https://developers.weeek.net/).
Requests run on behalf of the user who created the token.

For agents and CI, set the environment variable:

```sh
export WEEEK_TOKEN="your-token"
weeek me
```

For local interactive use, save the token in the CLI config:

```sh
weeek auth login
weeek auth status
```

`auth login` stores the token in `~/.config/weeek/config.json` with permissions
`0600`. `WEEEK_TOKEN` takes precedence over the saved config. To remove the saved
token, run `weeek auth logout`.

## Examples

List open tasks assigned to the current user:

```sh
weeek task list --responsible me --completed=false
```

Select task IDs and titles with `jq`:

```sh
weeek task list --search "release" --limit 20 \
  | jq '.tasks[] | {id, title}'
```

Create a task and extract the created task:

```sh
weeek task create \
  --title "Prepare release" \
  --responsible me \
  --priority 2 \
  | jq '.task'
```

Use `weeek help` to list commands and `weeek help task create` to inspect one
command. Flags may appear before or after positional arguments.

## Output contract

- API commands write the Weeek API JSON response to stdout unchanged. Commands
  that combine responses or report local state, such as `me`, `auth`, and
  `version`, write their own JSON objects.
- API, network, and authentication errors write one JSON object to stderr, with
  `error` and `status` fields. Usage errors write a short message and command
  usage to stderr.
- Exit code `0` means success, `1` means an API, network, or authentication
  failure, and `2` means invalid command usage.

This contract is designed for agent and shell composition. Pipe stdout to `jq`
without parsing human-oriented log output.

## Development

The CLI intentionally uses only the Go standard library. This keeps the binary
and dependency surface small; command output is protected by golden tests because
it is an interface consumed by agents.

```sh
make build
make test
make vet
make check-plugin
make release-build
```

`make release-build` writes macOS and Linux binaries for amd64 and arm64 to
`dist/`.

## Release process

Before tagging a release, update the version in both plugin manifests and the
CLI compatibility version in `plugin/skills/weeek/SKILL.md`, then run
`make check-plugin`. Keep these three values identical.
