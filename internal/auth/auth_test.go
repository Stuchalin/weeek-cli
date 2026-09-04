package auth_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/Stuchalin/weeek-cli/internal/auth"
)

func TestToken(t *testing.T) {
	tests := []struct {
		name       string
		envToken   string
		configData string
		wantToken  string
		wantErr    error
	}{
		{
			name:       "environment takes priority over config",
			envToken:   "environment-token",
			configData: `{"token":"config-token"}`,
			wantToken:  "environment-token",
		},
		{
			name:       "reads token from config",
			configData: `{"token":"config-token"}`,
			wantToken:  "config-token",
		},
		{
			name:    "token is not configured",
			wantErr: auth.ErrNoToken,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)
			t.Setenv("WEEEK_TOKEN", tt.envToken)

			if tt.configData != "" {
				writeConfig(t, home, tt.configData, 0o600)
			}

			got, err := auth.Token()
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Token() error = %v, want %v", err, tt.wantErr)
			}
			if got != tt.wantToken {
				t.Errorf("Token() = %q, want %q", got, tt.wantToken)
			}
		})
	}
}

func TestToken_ErrNoTokenMessage(t *testing.T) {
	const want = "token not found: set WEEEK_TOKEN or run weeek auth login"
	if got := auth.ErrNoToken.Error(); got != want {
		t.Errorf("ErrNoToken.Error() = %q, want %q", got, want)
	}
}

func TestSave(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	path := configPath(home)
	writeConfig(t, home, `{"token":"old-token"}`, 0o644)

	if err := auth.Save("new-token"); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat saved config: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Errorf("saved config permissions = %04o, want 0600", got)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read saved config: %v", err)
	}
	const want = `{"token":"new-token"}`
	if got := string(data); got != want {
		t.Errorf("saved config = %q, want %q", got, want)
	}
}

func TestSave_TokenRoundTrip(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("WEEEK_TOKEN", "")

	if err := auth.Save("roundtrip-token"); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	got, err := auth.Token()
	if err != nil {
		t.Fatalf("Token() error = %v", err)
	}
	if got != "roundtrip-token" {
		t.Errorf("Token() = %q, want %q", got, "roundtrip-token")
	}
}

func TestDelete(t *testing.T) {
	tests := []struct {
		name         string
		configExists bool
	}{
		{
			name:         "removes existing config",
			configExists: true,
		},
		{
			name: "missing config succeeds",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)

			if tt.configExists {
				writeConfig(t, home, `{"token":"config-token"}`, 0o600)
			}

			if err := auth.Delete(); err != nil {
				t.Fatalf("Delete() error = %v", err)
			}
			if _, err := os.Stat(configPath(home)); !errors.Is(err, os.ErrNotExist) {
				t.Errorf("config after Delete() stat error = %v, want os.ErrNotExist", err)
			}
		})
	}
}

func writeConfig(t *testing.T, home, data string, mode os.FileMode) {
	t.Helper()

	path := configPath(home)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("create config directory: %v", err)
	}
	if err := os.WriteFile(path, []byte(data), mode); err != nil {
		t.Fatalf("write config: %v", err)
	}
}

func configPath(home string) string {
	return filepath.Join(home, ".config", "weeek", "config.json")
}
