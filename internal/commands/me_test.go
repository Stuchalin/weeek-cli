package commands

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Stuchalin/weeek-cli/internal/api"
)

func TestMeGolden(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/user/me":
			if _, err := response.Write([]byte(`{"id":42,"name":"Test User"}`)); err != nil {
				t.Errorf("write user response: %v", err)
			}
		case "/ws":
			if _, err := response.Write(
				[]byte(`{"success":true,"workspace":{"id":7,"title":"Test Workspace"}}`),
			); err != nil {
				t.Errorf("write workspace response: %v", err)
			}
		default:
			t.Errorf("unexpected request path %q", request.URL.Path)
			response.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	registry := NewRegistry("test-version")
	code := registry.Run(
		[]string{"me"},
		&Ctx{
			Client: api.NewClient(server.URL, "test-token", server.Client()),
			Stdout: &stdout,
			Stderr: &stderr,
		},
	)
	if code != 0 {
		t.Fatalf("me code = %d, want 0; stderr = %q", code, stderr.String())
	}
	assertGolden(t, "me.golden", stdout.Bytes())
}
