package api

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	t.Parallel()

	original := &http.Client{Timeout: time.Second}
	client := NewClient("https://example.test/", "secret", original)

	if client.baseURL != "https://example.test" {
		t.Errorf("NewClient() base URL = %q, want %q", client.baseURL, "https://example.test")
	}
	if client.httpClient.Timeout != requestTimeout {
		t.Errorf("NewClient() timeout = %s, want %s", client.httpClient.Timeout, requestTimeout)
	}
	if original.Timeout != time.Second {
		t.Errorf("NewClient() changed caller's client timeout to %s", original.Timeout)
	}
}

func TestAPIError_Error(t *testing.T) {
	t.Parallel()

	err := &APIError{
		Status:  http.StatusUnauthorized,
		Message: "token\n is invalid",
	}

	const want = "api request failed with status 401: token is invalid"
	if got := err.Error(); got != want {
		t.Errorf("APIError.Error() = %q, want %q", got, want)
	}
}

func TestClient_DoSuccess(t *testing.T) {
	t.Parallel()

	const responseBody = "{\n  \"success\": true,\n  \"items\": [1, 2]\n}"
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if got := request.Header.Get("Authorization"); got != "Bearer secret" {
			t.Errorf("Authorization header = %q, want %q", got, "Bearer secret")
		}
		if request.URL.Path != "/items" {
			t.Errorf("request path = %q, want %q", request.URL.Path, "/items")
		}
		response.WriteHeader(http.StatusOK)
		if _, err := io.WriteString(response, responseBody); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "secret", server.Client())
	got, err := client.do(t.Context(), http.MethodGet, "/items", nil)
	if err != nil {
		t.Fatalf("do() error = %v", err)
	}
	if string(got) != responseBody {
		t.Errorf("do() body = %q, want exact pass-through %q", got, responseBody)
	}
}

func TestClient_DoAPIError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.WriteHeader(http.StatusUnauthorized)
		if _, err := io.WriteString(response, `{"message":"token is invalid"}`); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "secret", server.Client())
	_, err := client.do(t.Context(), http.MethodPost, "/items", []byte(`{}`))
	if err == nil {
		t.Fatal("do() error = nil, want APIError")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("do() error type = %T, want *APIError", err)
	}
	if apiErr.Status != http.StatusUnauthorized {
		t.Errorf("APIError.Status = %d, want %d", apiErr.Status, http.StatusUnauthorized)
	}
	if apiErr.Message != "token is invalid" {
		t.Errorf("APIError.Message = %q, want %q", apiErr.Message, "token is invalid")
	}
}

func TestClient_DoRetriesGET(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		statuses     []int
		wantBody     string
		wantErr      bool
		wantRequests int
		wantDelays   []time.Duration
	}{
		{
			name:         "429 then success",
			statuses:     []int{http.StatusTooManyRequests, http.StatusOK},
			wantBody:     `{"ok":true}`,
			wantRequests: 2,
			wantDelays:   []time.Duration{500 * time.Millisecond},
		},
		{
			name:         "5xx exhausts retries",
			statuses:     []int{http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable},
			wantErr:      true,
			wantRequests: 3,
			wantDelays:   []time.Duration{500 * time.Millisecond, 2 * time.Second},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var requests int
			server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
				status := tt.statuses[requests]
				requests++
				response.WriteHeader(status)
				if status == http.StatusOK {
					if _, err := io.WriteString(response, tt.wantBody); err != nil {
						t.Errorf("write response: %v", err)
					}
					return
				}
				if _, err := io.WriteString(response, `{"message":"try again"}`); err != nil {
					t.Errorf("write response: %v", err)
				}
			}))
			defer server.Close()

			client := NewClient(server.URL, "secret", server.Client())
			gotDelays := []time.Duration{}
			client.sleep = func(_ context.Context, delay time.Duration) error {
				gotDelays = append(gotDelays, delay)
				return nil
			}

			got, err := client.do(t.Context(), http.MethodGet, "/items", nil)
			if (err != nil) != tt.wantErr {
				t.Fatalf("do() error = %v, wantErr %t", err, tt.wantErr)
			}
			if string(got) != tt.wantBody {
				t.Errorf("do() body = %q, want %q", got, tt.wantBody)
			}
			if requests != tt.wantRequests {
				t.Errorf("request count = %d, want %d", requests, tt.wantRequests)
			}
			if !equalDurations(gotDelays, tt.wantDelays) {
				t.Errorf("retry delays = %v, want %v", gotDelays, tt.wantDelays)
			}
		})
	}
}

func TestClient_DoNetworkErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		transport roundTripFunc
		wantCause error
	}{
		{
			name: "timeout",
			transport: func(request *http.Request) (*http.Response, error) {
				return nil, context.DeadlineExceeded
			},
			wantCause: context.DeadlineExceeded,
		},
		{
			name: "connection reset",
			transport: func(request *http.Request) (*http.Response, error) {
				return nil, io.ErrUnexpectedEOF
			},
			wantCause: io.ErrUnexpectedEOF,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client := NewClient("https://example.test", "secret", &http.Client{Transport: tt.transport})
			_, err := client.do(t.Context(), http.MethodGet, "/items", nil)
			if err == nil {
				t.Fatal("do() error = nil, want network error")
			}
			if !errors.Is(err, tt.wantCause) {
				t.Errorf("do() error = %v, want wrapped cause %v", err, tt.wantCause)
			}
			if !strings.Contains(err.Error(), "sending api request") {
				t.Errorf("do() error = %q, want understandable request context", err)
			}
		})
	}
}

func TestClient_DoResponseReadError(t *testing.T) {
	t.Parallel()

	body := &failingReadCloser{readErr: io.ErrUnexpectedEOF}
	transport := roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       body,
		}, nil
	})
	client := NewClient("https://example.test", "secret", &http.Client{Transport: transport})

	response, err := client.do(t.Context(), http.MethodGet, "/items", nil)
	if err == nil {
		t.Fatal("do() error = nil, want response read error")
	}
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Errorf("do() error = %v, want wrapped cause %v", err, io.ErrUnexpectedEOF)
	}
	if response != nil {
		t.Errorf("do() response = %q, want nil", response)
	}
	if !body.closed {
		t.Error("do() did not close response body after read error")
	}
}

func TestClient_DoStopsWhenRetryWaitFails(t *testing.T) {
	t.Parallel()

	requests := 0
	transport := roundTripFunc(func(*http.Request) (*http.Response, error) {
		requests++
		return &http.Response{
			StatusCode: http.StatusTooManyRequests,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"message":"retry later"}`)),
		}, nil
	})
	client := NewClient("https://example.test", "secret", &http.Client{Transport: transport})
	client.sleep = func(context.Context, time.Duration) error {
		return context.Canceled
	}

	_, err := client.do(t.Context(), http.MethodGet, "/items", nil)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("do() error = %v, want wrapped cause %v", err, context.Canceled)
	}
	if requests != 1 {
		t.Errorf("request count = %d, want 1", requests)
	}
}

func TestSleepContextCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	if err := sleepContext(ctx, time.Hour); !errors.Is(err, context.Canceled) {
		t.Errorf("sleepContext() error = %v, want %v", err, context.Canceled)
	}
}

func TestClient_DoDoesNotRetryNonGET(t *testing.T) {
	t.Parallel()

	var requests int
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		requests++
		if got := request.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type header = %q, want %q", got, "application/json")
		}
		response.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient(server.URL, "secret", server.Client())
	client.sleep = func(_ context.Context, _ time.Duration) error {
		t.Fatal("sleep called for non-GET request")
		return nil
	}

	_, err := client.do(t.Context(), http.MethodPost, "/items", []byte(`{"title":"item"}`))
	if err == nil {
		t.Fatal("do() error = nil, want API error")
	}
	if requests != 1 {
		t.Errorf("request count = %d, want 1", requests)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}

type failingReadCloser struct {
	readErr error
	closed  bool
}

func (r *failingReadCloser) Read([]byte) (int, error) {
	return 0, r.readErr
}

func (r *failingReadCloser) Close() error {
	r.closed = true
	return nil
}

func equalDurations(left, right []time.Duration) bool {
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
