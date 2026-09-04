# Development rules

- Keep the CLI dependency-free: use the Go standard library unless a dependency
  is explicitly approved.
- Treat stdout, stderr, and exit codes as public contracts. User-facing CLI
  output and errors must be in English.
- Preserve API responses with `json.RawMessage` unless a command intentionally
  combines responses or reports local state.
- Add or update tests with every behavior change. Command output changes require
  an intentional golden-file update with `GOLDEN_UPDATE=1 go test ./...`.
- Run `go test ./...`, `go vet ./...`, and `make check-plugin` before committing.

## Architecture

- `cmd/weeek` wires process I/O, token resolution, and API client construction.
- `internal/commands` owns command registration, argument parsing, usage errors,
  and the stdout/stderr/exit-code contract. Register new commands in
  `NewRegistry`.
- `internal/api` maps CLI operations to Weeek HTTP endpoints and preserves API
  response bodies unless a command intentionally combines them.
- `internal/auth` owns token precedence and the local token configuration.
- `plugin/` contains the shared Weeek skill and the Claude Code and Codex plugin
  manifests.

## Plugin release workflow

Keep the versions in both plugin manifests, `.claude-plugin/marketplace.json`,
and `plugin/skills/weeek/SKILL.md` synchronized. Run `make check-plugin` after
changing any plugin metadata or skill reference.
