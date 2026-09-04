package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const requestTimeout = 30 * time.Second

var retryDelays = [...]time.Duration{
	500 * time.Millisecond,
	2 * time.Second,
}

type sleepFunc func(context.Context, time.Duration) error

// APIError describes a non-success response returned by the Weeek API.
type APIError struct {
	Status  int
	Message string
}

func (e *APIError) Error() string {
	message := strings.Join(strings.Fields(e.Message), " ")
	if message == "" {
		return fmt.Sprintf("api request failed with status %d", e.Status)
	}

	return fmt.Sprintf("api request failed with status %d: %s", e.Status, message)
}

// Client sends requests to the Weeek API.
type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
	sleep      sleepFunc
}

// NewClient creates a Weeek API client.
func NewClient(baseURL, token string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{}
	}

	clientCopy := *httpClient
	clientCopy.Timeout = requestTimeout

	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		token:      token,
		httpClient: &clientCopy,
		sleep:      sleepContext,
	}
}

func (c *Client) do(
	ctx context.Context,
	method string,
	path string,
	body []byte,
) (json.RawMessage, error) {
	for attempt := 0; ; attempt++ {
		response, err := c.send(ctx, method, path, body)
		if err != nil {
			return nil, err
		}

		if shouldRetry(method, response.StatusCode) && attempt < len(retryDelays) {
			if err := discardAndClose(response.Body); err != nil {
				return nil, fmt.Errorf("discarding response before retry: %w", err)
			}
			if err := c.sleep(ctx, retryDelays[attempt]); err != nil {
				return nil, fmt.Errorf("waiting to retry request: %w", err)
			}

			continue
		}

		return readResponse(response)
	}
}

func (c *Client) send(
	ctx context.Context,
	method string,
	path string,
	body []byte,
) (*http.Response, error) {
	request, err := http.NewRequestWithContext(
		ctx,
		method,
		c.baseURL+"/"+strings.TrimLeft(path, "/"),
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, fmt.Errorf("creating api request: %w", err)
	}

	request.Header.Set("Authorization", "Bearer "+c.token)
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}

	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("sending api request: %w", err)
	}

	return response, nil
}

func shouldRetry(method string, status int) bool {
	isServerError := status >= 500 && status < 600
	return method == http.MethodGet && (status == http.StatusTooManyRequests || isServerError)
}

func readResponse(response *http.Response) (json.RawMessage, error) {
	body, readErr := io.ReadAll(response.Body)
	closeErr := response.Body.Close()
	if readErr != nil {
		return nil, fmt.Errorf("reading api response: %w", errors.Join(readErr, closeErr))
	}
	if closeErr != nil {
		return nil, fmt.Errorf("closing api response: %w", closeErr)
	}

	if response.StatusCode >= 200 && response.StatusCode < 300 {
		if response.StatusCode == http.StatusNoContent && len(body) == 0 {
			return nil, nil
		}
		if !json.Valid(body) {
			return nil, fmt.Errorf(
				"decoding api response: status %d returned invalid JSON",
				response.StatusCode,
			)
		}
		return json.RawMessage(body), nil
	}

	return nil, newAPIError(response.StatusCode, body)
}

func newAPIError(status int, body []byte) *APIError {
	message := strings.TrimSpace(string(body))

	var payload struct {
		Message string `json:"message"`
		Error   string `json:"error"`
	}
	if err := json.Unmarshal(body, &payload); err == nil {
		switch {
		case payload.Message != "":
			message = payload.Message
		case payload.Error != "":
			message = payload.Error
		}
	}

	if message == "" {
		message = http.StatusText(status)
	}

	return &APIError{
		Status:  status,
		Message: message,
	}
}

func discardAndClose(body io.ReadCloser) error {
	_, copyErr := io.Copy(io.Discard, body)
	closeErr := body.Close()
	return errors.Join(copyErr, closeErr)
}

func sleepContext(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
