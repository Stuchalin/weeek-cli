package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/Stuchalin/weeek-cli/internal/commands"
)

func TestRunCommandsWithoutToken(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("WEEEK_TOKEN", "")

	tests := []struct {
		name     string
		args     []string
		wantCode int
	}{
		{
			name: "bare invocation",
		},
		{
			name: "help",
			args: []string{"help"},
		},
		{
			name: "version",
			args: []string{"version"},
		},
		{
			name:     "auth login",
			args:     []string{"auth", "login"},
			wantCode: 2,
		},
		{
			name:     "unknown command",
			args:     []string{"unknown"},
			wantCode: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer

			gotCode := runWithInput(tt.args, strings.NewReader(""), &stdout, &stderr)
			if gotCode != tt.wantCode {
				t.Errorf("run() code = %d, want %d", gotCode, tt.wantCode)
			}
			if strings.Contains(stderr.String(), "token not found") {
				t.Errorf("run() stderr = %q, must not resolve token", stderr.String())
			}
		})
	}
}

func TestNeedsToken(t *testing.T) {
	t.Parallel()

	registry := commands.NewRegistry("test-version")

	tests := []struct {
		name string
		args []string
		want bool
	}{
		{name: "bare invocation"},
		{name: "help", args: []string{"help"}},
		{name: "version", args: []string{"version"}},
		{name: "auth", args: []string{"auth", "status"}},
		{name: "registered api command", args: []string{"task", "list"}, want: true},
		{name: "unknown command", args: []string{"unknown"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := needsToken(tt.args, registry); got != tt.want {
				t.Errorf("needsToken(%v) = %t, want %t", tt.args, got, tt.want)
			}
		})
	}
}
