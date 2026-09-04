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
