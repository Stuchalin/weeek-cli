package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClient_GetMe(t *testing.T) {
	t.Parallel()

	const responseBody = `{"success":true,"user":{"id":"user-42","name":"Test User"}}`
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
	if me.ID != "user-42" {
		t.Errorf("GetMe() ID = %q, want %q", me.ID, "user-42")
	}
}

func TestClient_GetMeIDValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		responseBody string
		wantID       string
		wantErr      bool
	}{
		{name: "missing id", responseBody: `{"user":{}}`},
		{name: "null id", responseBody: `{"user":{"id":null}}`},
		{name: "numeric id", responseBody: `{"user":{"id":42}}`, wantErr: true},
		{name: "object id", responseBody: `{"user":{"id":{}}}`, wantErr: true},
		{name: "boolean id", responseBody: `{"user":{"id":true}}`, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
				_, _ = response.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := NewClient(server.URL, "secret", server.Client())
			_, me, err := client.GetMe(t.Context())
			if (err != nil) != tt.wantErr {
				t.Fatalf("GetMe() error = %v, wantErr %t", err, tt.wantErr)
			}
			if me.ID != tt.wantID {
				t.Errorf("GetMe() ID = %q, want %q", me.ID, tt.wantID)
			}
		})
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
