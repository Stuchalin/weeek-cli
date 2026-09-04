package commands

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Stuchalin/weeek-cli/internal/api"
	"github.com/Stuchalin/weeek-cli/internal/auth"
)

func TestAuthLoginWithFlag(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("WEEEK_TOKEN", "")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	registry := NewRegistry("test-version")
	code := registry.Run(
		[]string{"auth", "login", "--token", "saved-token"},
		&Ctx{Stdout: &stdout, Stderr: &stderr},
	)
	if code != 0 {
		t.Fatalf("auth login code = %d, want 0; stderr = %q", code, stderr.String())
	}

	path := filepath.Join(home, ".config", "weeek", "config.json")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat token config: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Errorf("token config mode = %04o, want 0600", got)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read token config: %v", err)
	}
	if got := string(data); got != `{"token":"saved-token"}` {
		t.Errorf("token config = %q, want saved token", got)
	}

	normalized := strings.ReplaceAll(stdout.String(), path, "<config>")
	assertGolden(t, "auth_login.golden", []byte(normalized))
}

func TestAuthLoginFromPrompt(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("WEEEK_TOKEN", "")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	registry := NewRegistry("test-version")
	code := registry.Run(
		[]string{"auth", "login"},
		&Ctx{
			Stdin:  strings.NewReader("prompt-token\n"),
			Stdout: &stdout,
			Stderr: &stderr,
		},
	)
	if code != 0 {
		t.Fatalf("auth login code = %d, want 0; stderr = %q", code, stderr.String())
	}
	if got := stderr.String(); got != "Token: " {
		t.Errorf("auth login prompt = %q, want %q", got, "Token: ")
	}

	t.Setenv("WEEEK_TOKEN", "")
	token, err := auth.Token()
	if err != nil {
		t.Fatalf("Token() error = %v", err)
	}
	if token != "prompt-token" {
		t.Errorf("Token() = %q, want %q", token, "prompt-token")
	}
}

func TestAuthStatus(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("WEEEK_TOKEN", "status-token")

	const responseBody = `{"id":42,"name":"Test User"}`
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/user/me" {
			t.Errorf("request path = %q, want %q", request.URL.Path, "/user/me")
		}
		if got := request.Header.Get("Authorization"); got != "Bearer status-token" {
			t.Errorf("Authorization header = %q, want %q", got, "Bearer status-token")
		}
		_, _ = response.Write([]byte(responseBody))
	}))
	defer server.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	registry := NewRegistry("test-version")
	code := registry.Run(
		[]string{"auth", "status"},
		&Ctx{
			NewAPIClient: func(token string) *api.Client {
				return api.NewClient(server.URL, token, server.Client())
			},
			Stdout: &stdout,
			Stderr: &stderr,
		},
	)
	if code != 0 {
		t.Fatalf("auth status code = %d, want 0; stderr = %q", code, stderr.String())
	}
	assertGolden(t, "auth_status.golden", stdout.Bytes())
}

func TestAuthStatusUnauthorized(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("WEEEK_TOKEN", "invalid-token")

	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.WriteHeader(http.StatusUnauthorized)
		_, _ = response.Write([]byte(`{"message":"unauthorized"}`))
	}))
	defer server.Close()

	var stderr bytes.Buffer
	registry := NewRegistry("test-version")
	code := registry.Run(
		[]string{"auth", "status"},
		&Ctx{
			NewAPIClient: func(token string) *api.Client {
				return api.NewClient(server.URL, token, server.Client())
			},
			Stderr: &stderr,
		},
	)
	if code != 1 {
		t.Errorf("auth status code = %d, want 1", code)
	}
	if got := stderr.String(); got != "{\"error\":\"token is invalid or expired\",\"status\":0}\n" {
		t.Errorf("auth status stderr = %q, want invalid-token error", got)
	}
}

func TestAuthStatusWithoutToken(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("WEEEK_TOKEN", "")

	var stderr bytes.Buffer
	registry := NewRegistry("test-version")
	code := registry.Run([]string{"auth", "status"}, &Ctx{Stderr: &stderr})
	if code != 1 {
		t.Errorf("auth status code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), auth.ErrNoToken.Error()) {
		t.Errorf("auth status stderr = %q, want ErrNoToken", stderr.String())
	}
}

func TestAuthLogout(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("WEEEK_TOKEN", "")
	if err := auth.Save("saved-token"); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	path, err := auth.ConfigPath()
	if err != nil {
		t.Fatalf("ConfigPath() error = %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	registry := NewRegistry("test-version")
	code := registry.Run(
		[]string{"auth", "logout"},
		&Ctx{Stdout: &stdout, Stderr: &stderr},
	)
	if code != 0 {
		t.Fatalf("auth logout code = %d, want 0; stderr = %q", code, stderr.String())
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("auth logout stat error = %v, want os.ErrNotExist", err)
	}
	assertGolden(t, "auth_logout.golden", stdout.Bytes())
}
