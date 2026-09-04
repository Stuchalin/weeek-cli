package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Stuchalin/weeek-cli/internal/auth"
)

func TestRunCommandsWithoutToken(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("WEEEK_TOKEN", "")

	tests := []struct {
		name           string
		args           []string
		wantCode       int
		wantTokenError bool
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
		{
			name:     "invalid api command usage",
			args:     []string{"task", "list", "--definitely-unknown"},
			wantCode: 2,
		},
		{
			name:           "valid api command",
			args:           []string{"task", "list"},
			wantCode:       1,
			wantTokenError: true,
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
			hasTokenError := strings.Contains(stderr.String(), "token not found")
			if hasTokenError != tt.wantTokenError {
				t.Errorf("run() stderr = %q, token error = %t, want %t", stderr.String(), hasTokenError, tt.wantTokenError)
			}
		})
	}
}

func TestRunAPICommandTokenSources(t *testing.T) {
	tests := []struct {
		name  string
		value string
		save  bool
	}{
		{name: "environment", value: "test-env-value"},
		{name: "config", value: "test-config-value", save: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("HOME", t.TempDir())
			if tt.save {
				t.Setenv("WEEEK_TOKEN", "")
				if err := auth.Save(tt.value); err != nil {
					t.Fatalf("Save() error = %v", err)
				}
			} else {
				t.Setenv("WEEEK_TOKEN", tt.value)
			}

			server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
				if got := request.Header.Get("Authorization"); got != "Bearer "+tt.value {
					t.Errorf("Authorization header = %q, want %q", got, "Bearer "+tt.value)
				}
				if request.URL.Path != "/ws" {
					t.Errorf("request path = %q, want %q", request.URL.Path, "/ws")
				}
				_, _ = response.Write([]byte(`{"success":true}`))
			}))
			defer server.Close()

			originalBaseURL := apiBaseURL
			apiBaseURL = server.URL
			defer func() { apiBaseURL = originalBaseURL }()

			var stdout bytes.Buffer
			var stderr bytes.Buffer
			code := runWithInput([]string{"ws", "info"}, strings.NewReader(""), &stdout, &stderr)
			if code != 0 {
				t.Fatalf("runWithInput() code = %d, want 0; stderr = %q", code, stderr.String())
			}
			if got := stdout.String(); got != "{\"success\":true}\n" {
				t.Errorf("runWithInput() stdout = %q, want API response", got)
			}
		})
	}
}
