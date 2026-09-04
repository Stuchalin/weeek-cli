package commands

import (
	"bytes"
	"errors"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRegistry_Run(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		args       []string
		command    Command
		wantCode   int
		wantStdout string
		wantStderr string
	}{
		{
			name: "dispatches exact multiword command",
			args: []string{"task", "get", "42"},
			command: Command{
				Name:  "task get",
				Usage: "task get ID",
				Run: func(ctx *Ctx) error {
					_, err := strings.NewReader(strings.Join(ctx.Args, ",")).WriteTo(ctx.Stdout)
					return err
				},
			},
			wantStdout: "42",
		},
		{
			name: "execution error is json",
			args: []string{"task", "get"},
			command: Command{
				Name:  "task get",
				Usage: "task get ID",
				Run: func(*Ctx) error {
					return errors.New("task not found")
				},
			},
			wantCode:   1,
			wantStderr: "{\"error\":\"task not found\",\"status\":0}\n",
		},
		{
			name: "flag error prints command usage",
			args: []string{"task", "get", "--unknown"},
			command: Command{
				Name:  "task get",
				Usage: "task get ID",
				Short: "Get a task",
				Run: func(ctx *Ctx) error {
					fs := flag.NewFlagSet("task get", flag.ContinueOnError)
					fs.SetOutput(&bytes.Buffer{})
					_, err := parseAll(fs, ctx.Args)
					return err
				},
			},
			wantCode:   2,
			wantStderr: "error: flag provided but not defined: -unknown\nUsage: weeek task get ID\n\nGet a task\n",
		},
		{
			name: "panic prints command usage",
			args: []string{"task", "get"},
			command: Command{
				Name:  "task get",
				Usage: "task get ID",
				Run: func(*Ctx) error {
					panic("duplicate flag")
				},
			},
			wantCode:   2,
			wantStderr: "error: command setup failed: duplicate flag\nUsage: weeek task get ID\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			registry := Registry{}
			registry.Register(tt.command)
			var stdout bytes.Buffer
			var stderr bytes.Buffer

			gotCode := registry.Run(
				tt.args,
				&Ctx{Stdout: &stdout, Stderr: &stderr},
			)
			if gotCode != tt.wantCode {
				t.Errorf("Run() code = %d, want %d", gotCode, tt.wantCode)
			}
			if got := stdout.String(); got != tt.wantStdout {
				t.Errorf("Run() stdout = %q, want %q", got, tt.wantStdout)
			}
			if got := stderr.String(); got != tt.wantStderr {
				t.Errorf("Run() stderr = %q, want %q", got, tt.wantStderr)
			}
		})
	}
}

func TestRegistry_RunUnknownCommand(t *testing.T) {
	t.Parallel()

	registry := Registry{}
	registry.Register(Command{
		Name:  "task list",
		Usage: "task list [flags]",
		Short: "List tasks",
		Run:   func(*Ctx) error { return nil },
	})
	registry.Register(Command{
		Name:  "task get",
		Usage: "task get ID",
		Short: "Get a task",
		Run:   func(*Ctx) error { return nil },
	})
	var stderr bytes.Buffer

	gotCode := registry.Run([]string{"task"}, &Ctx{Stderr: &stderr})
	if gotCode != 2 {
		t.Errorf("Run() code = %d, want 2", gotCode)
	}
	for _, want := range []string{
		`unknown command "task"`,
		"Similar commands:",
		"task get",
		"task list",
		"Usage: weeek <command> [arguments]",
	} {
		if !strings.Contains(stderr.String(), want) {
			t.Errorf("Run() stderr = %q, want it to contain %q", stderr.String(), want)
		}
	}
}

func TestRegistry_RunHelp(t *testing.T) {
	t.Parallel()

	registry := NewRegistry("test-version")
	registry.Register(Command{
		Name:  "task list",
		Usage: "task list [flags]",
		Short: "List tasks",
		Run:   func(*Ctx) error { return nil },
	})

	tests := []struct {
		name     string
		args     []string
		contains []string
	}{
		{
			name: "bare invocation",
			contains: []string{
				"Usage: weeek <command> [arguments]",
				"help [command]",
				"task list",
				"version",
			},
		},
		{
			name: "help command listing",
			args: []string{"help"},
			contains: []string{
				"task list",
				"version",
			},
		},
		{
			name:     "help for command",
			args:     []string{"help", "task", "list"},
			contains: []string{"Usage: weeek task list [flags]", "List tasks"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var stdout bytes.Buffer
			var stderr bytes.Buffer
			if gotCode := registry.Run(tt.args, &Ctx{Stdout: &stdout, Stderr: &stderr}); gotCode != 0 {
				t.Fatalf("Run() code = %d, want 0; stderr = %q", gotCode, stderr.String())
			}
			for _, want := range tt.contains {
				if !strings.Contains(stdout.String(), want) {
					t.Errorf("Run() stdout = %q, want it to contain %q", stdout.String(), want)
				}
			}
		})
	}
}

func TestRegistry_RunVersionGolden(t *testing.T) {
	t.Parallel()

	registry := NewRegistry("test-version")
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if gotCode := registry.Run([]string{"version"}, &Ctx{Stdout: &stdout, Stderr: &stderr}); gotCode != 0 {
		t.Fatalf("Run() code = %d, want 0; stderr = %q", gotCode, stderr.String())
	}
	assertGolden(t, "version.golden", stdout.Bytes())
}

func assertGolden(t *testing.T, name string, got []byte) {
	t.Helper()

	path := filepath.Join("testdata", name)
	if os.Getenv("GOLDEN_UPDATE") == "1" {
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatalf("update golden file: %v", err)
		}
	}

	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden file: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("Run() stdout = %q, want golden %q", got, want)
	}
}

func TestRegistry_RunVersionWithArguments(t *testing.T) {
	t.Parallel()

	registry := NewRegistry("test-version")
	var stderr bytes.Buffer

	gotCode := registry.Run([]string{"version", "extra"}, &Ctx{Stderr: &stderr})
	if gotCode != 2 {
		t.Errorf("Run() code = %d, want 2", gotCode)
	}
	if got := stderr.String(); !strings.Contains(got, "Usage: weeek version") {
		t.Errorf("Run() stderr = %q, want version usage", got)
	}
}

func TestWriteError_ProducesOneLineJSON(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	WriteError(&output, errors.New("first line\nsecond line"))

	const want = "{\"error\":\"first line\\nsecond line\",\"status\":0}\n"
	if got := output.String(); got != want {
		t.Errorf("WriteError() = %q, want %q", got, want)
	}
}

func TestRegistry_HasCommand(t *testing.T) {
	t.Parallel()

	registry := NewRegistry("test-version")
	if !registry.HasCommand([]string{"version"}) {
		t.Error("HasCommand(version) = false, want true")
	}
	if registry.HasCommand([]string{"unknown"}) {
		t.Error("HasCommand(unknown) = true, want false")
	}
}

func TestParseAll(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		args            []string
		wantPositionals []string
		wantValue       string
		wantVerbose     bool
	}{
		{
			name:            "positional before flag",
			args:            []string{"42", "--name", "example"},
			wantPositionals: []string{"42"},
			wantValue:       "example",
		},
		{
			name:            "flag before positional",
			args:            []string{"--name", "example", "42"},
			wantPositionals: []string{"42"},
			wantValue:       "example",
		},
		{
			name:            "equals and boolean flags",
			args:            []string{"first", "--verbose", "--name=example", "second"},
			wantPositionals: []string{"first", "second"},
			wantValue:       "example",
			wantVerbose:     true,
		},
		{
			name:            "double dash stops flag parsing",
			args:            []string{"--name", "example", "--", "--verbose", "42"},
			wantPositionals: []string{"--verbose", "42"},
			wantValue:       "example",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fs := flag.NewFlagSet("test", flag.ContinueOnError)
			fs.SetOutput(&bytes.Buffer{})
			value := fs.String("name", "", "name")
			verbose := fs.Bool("verbose", false, "verbose output")

			gotPositionals, err := parseAll(fs, tt.args)
			if err != nil {
				t.Fatalf("parseAll() error = %v", err)
			}
			if strings.Join(gotPositionals, ",") != strings.Join(tt.wantPositionals, ",") {
				t.Errorf("parseAll() positionals = %v, want %v", gotPositionals, tt.wantPositionals)
			}
			if *value != tt.wantValue {
				t.Errorf("parseAll() name = %q, want %q", *value, tt.wantValue)
			}
			if *verbose != tt.wantVerbose {
				t.Errorf("parseAll() verbose = %t, want %t", *verbose, tt.wantVerbose)
			}
		})
	}
}
