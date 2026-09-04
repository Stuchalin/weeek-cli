package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClient_GetMe(t *testing.T) {
	t.Parallel()

	const responseBody = `{"id":42,"name":"Test User"}`
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet {
			t.Errorf("request method = %q, want %q", request.Method, http.MethodGet)
		}
		if request.URL.Path != "/user/me" {
			t.Errorf("request path = %q, want %q", request.URL.Path, "/user/me")
		}
		if got := request.Header.Get("Authorization"); got != "Bearer secret" {
			t.Errorf("Authorization header = %q, want %q", got, "Bearer secret")
		}

		response.Header().Set("Content-Type", "application/json")
		_, _ = response.Write([]byte(responseBody))
	}))
	defer server.Close()

	client := NewClient(server.URL, "secret", server.Client())
	raw, me, err := client.GetMe(t.Context())
	if err != nil {
		t.Fatalf("GetMe() error = %v", err)
	}
	if string(raw) != responseBody {
		t.Errorf("GetMe() response = %q, want %q", raw, responseBody)
	}
	if me.ID != 42 {
		t.Errorf("GetMe() ID = %d, want 42", me.ID)
	}
}

func TestClient_GetMeRejectsInvalidJSON(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		_, _ = response.Write([]byte(`not-json`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "secret", server.Client())
	_, _, err := client.GetMe(t.Context())
	if err == nil {
		t.Fatal("GetMe() error = nil, want decoding error")
	}
}
