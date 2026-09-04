// Package commands dispatches weeek CLI commands and formats their output.
package commands

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/Stuchalin/weeek-cli/internal/api"
)

// Ctx contains the dependencies and arguments available to a command.
type Ctx struct {
	Client       *api.Client
	NewAPIClient func(string) *api.Client
	Stdin        io.Reader
	Stdout       io.Writer
	Stderr       io.Writer
	Args         []string
}

// Command describes one command in the CLI registry.
type Command struct {
	Name  string
	Usage string
	Short string
	Run   func(*Ctx) error
}

// Registry maps full command names to their implementations.
type Registry map[string]Command

type usageError struct {
	err error
}

func (e *usageError) Error() string {
	return e.err.Error()
}

func (e *usageError) Unwrap() error {
	return e.err
}

// NewRegistry creates the built-in command registry.
func NewRegistry(version string) Registry {
	registry := Registry{}
	registerAuthCommands(registry)
	registerMeCommand(registry)
	registerWorkspaceCommands(registry)
	registerTaskCommands(registry)
	registerBoardCommands(registry)
	registry.Register(versionCommand(version))
	return registry
}

// Register adds a command to the registry.
func (r Registry) Register(command Command) {
	if command.Name == "" {
		panic("commands: command name must not be empty")
	}
	if command.Run == nil {
		panic("commands: command run function must not be nil")
	}
	if _, exists := r[command.Name]; exists {
		panic(fmt.Sprintf("commands: command %q is already registered", command.Name))
	}

	r[command.Name] = command
}

// HasCommand reports whether args start with an exact registered command name.
func (r Registry) HasCommand(args []string) bool {
	_, _, found := r.resolve(args)
	return found
}

// Run dispatches args and returns the process exit code.
func (r Registry) Run(args []string, ctx *Ctx) int {
	ctx = normalizeContext(ctx)

	if len(args) == 0 {
		r.writeCommandList(ctx.Stdout)
		return 0
	}
	if args[0] == "help" {
		return r.runHelp(args[1:], ctx)
	}

	command, commandArgs, found := r.resolve(args)
	if !found {
		r.writeUnknownCommand(ctx.Stderr, args)
		return 2
	}

	commandCtx := *ctx
	commandCtx.Args = commandArgs
	err := runCommand(command, &commandCtx)
	if err == nil {
		return 0
	}

	var usageErr *usageError
	if errors.As(err, &usageErr) {
		writeCommandUsage(ctx.Stderr, command, usageErr)
		return 2
	}

	WriteError(ctx.Stderr, err)
	return 1
}

func (r Registry) runHelp(args []string, ctx *Ctx) int {
	if len(args) == 0 {
		r.writeCommandList(ctx.Stdout)
		return 0
	}

	name := strings.Join(args, " ")
	command, found := r[name]
	if !found {
		r.writeUnknownCommand(ctx.Stderr, args)
		return 2
	}

	writeCommandUsage(ctx.Stdout, command, nil)
	return 0
}

func (r Registry) resolve(args []string) (Command, []string, bool) {
	names := r.names()
	sort.SliceStable(names, func(i, j int) bool {
		return len(strings.Fields(names[i])) > len(strings.Fields(names[j]))
	})

	for _, name := range names {
		parts := strings.Fields(name)
		if len(parts) > len(args) || !equalStrings(parts, args[:len(parts)]) {
			continue
		}

		return r[name], args[len(parts):], true
	}

	return Command{}, nil, false
}

func (r Registry) writeCommandList(writer io.Writer) {
	fmt.Fprintln(writer, "Usage: weeek <command> [arguments]")
	fmt.Fprintln(writer)
	fmt.Fprintln(writer, "Commands:")
	fmt.Fprintln(writer, "  help [command]\tShow help for commands")
	for _, name := range r.names() {
		command := r[name]
		fmt.Fprintf(writer, "  %-16s\t%s\n", command.Name, command.Short)
	}
}

func (r Registry) writeUnknownCommand(writer io.Writer, args []string) {
	query := strings.Join(args, " ")
	fmt.Fprintf(writer, "unknown command %q\n", query)

	similar := make([]string, 0)
	for _, name := range r.names() {
		if strings.HasPrefix(name, query) || strings.HasPrefix(query, name+" ") {
			similar = append(similar, name)
		}
	}
	if len(similar) > 0 {
		fmt.Fprintln(writer)
		fmt.Fprintln(writer, "Similar commands:")
		for _, name := range similar {
			fmt.Fprintf(writer, "  %s\n", name)
		}
	}

	fmt.Fprintln(writer)
	r.writeCommandList(writer)
}

func (r Registry) names() []string {
	names := make([]string, 0, len(r))
	for name := range r {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func parseAll(fs *flag.FlagSet, args []string) ([]string, error) {
	if fs == nil {
		return nil, &usageError{err: errors.New("flag set must not be nil")}
	}
	fs.SetOutput(io.Discard)

	flagArgs := make([]string, 0, len(args))
	positionals := make([]string, 0, len(args))
	hasTerminator := false
	for index := 0; index < len(args); index++ {
		arg := args[index]
		if arg == "--" {
			hasTerminator = true
			positionals = append(positionals, args[index+1:]...)
			break
		}
		if arg == "-" || !strings.HasPrefix(arg, "-") {
			positionals = append(positionals, arg)
			continue
		}

		flagArgs = append(flagArgs, arg)
		name, hasValue := flagName(arg)
		definition := fs.Lookup(name)
		if definition == nil || hasValue || isBooleanFlag(definition) {
			continue
		}
		if index+1 >= len(args) {
			continue
		}

		index++
		flagArgs = append(flagArgs, args[index])
	}

	if hasTerminator {
		flagArgs = append(flagArgs, "--")
	}
	reordered := append(flagArgs, positionals...)
	if err := fs.Parse(reordered); err != nil {
		return nil, &usageError{err: err}
	}

	return fs.Args(), nil
}

func flagName(arg string) (string, bool) {
	trimmed := strings.TrimLeft(arg, "-")
	name, _, hasValue := strings.Cut(trimmed, "=")
	return name, hasValue
}

func isBooleanFlag(definition *flag.Flag) bool {
	boolean, ok := definition.Value.(interface{ IsBoolFlag() bool })
	return ok && boolean.IsBoolFlag()
}

func versionCommand(version string) Command {
	return Command{
		Name:  "version",
		Usage: "version",
		Short: "Print the CLI version",
		Run: func(ctx *Ctx) error {
			if len(ctx.Args) != 0 {
				return &usageError{err: errors.New("version does not accept arguments")}
			}
			if err := json.NewEncoder(ctx.Stdout).Encode(map[string]string{"version": version}); err != nil {
				return fmt.Errorf("writing version output: %w", err)
			}

			return nil
		},
	}
}

func normalizeContext(ctx *Ctx) *Ctx {
	if ctx == nil {
		ctx = &Ctx{}
	}
	if ctx.Stdout == nil {
		ctx.Stdout = io.Discard
	}
	if ctx.Stderr == nil {
		ctx.Stderr = io.Discard
	}
	return ctx
}

func runCommand(command Command, ctx *Ctx) (runErr error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			runErr = &usageError{
				err: fmt.Errorf("command setup failed: %v", recovered),
			}
		}
	}()

	return command.Run(ctx)
}

func writeCommandUsage(writer io.Writer, command Command, err error) {
	if err != nil {
		fmt.Fprintf(writer, "error: %v\n", err)
	}
	fmt.Fprintf(writer, "Usage: weeek %s\n", command.Usage)
	if command.Short != "" {
		fmt.Fprintf(writer, "\n%s\n", command.Short)
	}
}

// WriteError writes the stable one-line JSON error contract.
func WriteError(writer io.Writer, err error) {
	status := 0
	if apiErr, ok := errors.AsType[*api.APIError](err); ok {
		status = apiErr.Status
	}
	payload := struct {
		Error  string `json:"error"`
		Status int    `json:"status"`
	}{
		Error:  err.Error(),
		Status: status,
	}
	if encodeErr := json.NewEncoder(writer).Encode(payload); encodeErr != nil {
		fmt.Fprintf(writer, "{\"error\":%q}\n", "writing error output: "+encodeErr.Error())
	}
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
