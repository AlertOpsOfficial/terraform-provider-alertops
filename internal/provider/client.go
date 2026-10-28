package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const (
	defaultBaseURL = "https://api.alertops.com/api/v2"

	// The AlertOps v2 API allows 5 calls/second and 15 calls/minute per API key.
	defaultRequestsPerSecond = 5
	defaultRequestsPerMinute = 15

	defaultMaxRetries   = 6
	maxRetryBackoff     = 60 * time.Second
	initialRetryBackoff = 5 * time.Second

	// Transient 5xx responses (seen e.g. when deleting several integrations at once) are
	// retried for idempotent methods only, so a POST is never sent twice.
	maxServerErrorRetries = 3
	serverErrorBackoff    = 2 * time.Second
)

// Client manages communication with the AlertOps v2 API.
type Client struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
	MaxRetries int

	limiter *rateLimiter
	sleep   func(ctx context.Context, d time.Duration) error
}

// APIError is returned when the AlertOps API responds with a non-2xx status.
type APIError struct {
	Method     string
	Path       string
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API request %s %s failed with status %d: %s", e.Method, e.Path, e.StatusCode, e.Body)
}

// IsNotFound reports whether err is an API 404 response.
func IsNotFound(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound
}

// NewClient creates a new AlertOps API client using the default endpoint and rate limits.
func NewClient(apiKey string) *Client {
	return NewClientWithOptions(apiKey, defaultBaseURL, defaultRequestsPerSecond, defaultRequestsPerMinute)
}

// NewClientWithOptions creates a new AlertOps API client. A rate limit of 0 disables that window.
func NewClientWithOptions(apiKey, baseURL string, requestsPerSecond, requestsPerMinute int) *Client {
	return &Client{
		BaseURL: baseURL,
		APIKey:  apiKey,
		HTTPClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		MaxRetries: defaultMaxRetries,
		limiter:    newRateLimiter(requestsPerSecond, requestsPerMinute),
		sleep:      sleepContext,
	}
}

// doRequest performs an HTTP request with proper headers, client-side rate limiting
// and retries on HTTP 429.
func (c *Client) doRequest(ctx context.Context, method, path string, body interface{}) ([]byte, error) {
	url := c.BaseURL + path

	var jsonData []byte
	if body != nil {
		var err error
		jsonData, err = json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("error marshaling request body: %w", err)
		}
	}

	serverErrors := 0
	for attempt := 0; ; attempt++ {
		if err := c.limiter.wait(ctx); err != nil {
			return nil, err
		}

		if jsonData != nil {
			tflog.Debug(ctx, "API Request", map[string]interface{}{
				"method": method,
				"url":    url,
				"body":   string(jsonData),
			})
		} else {
			tflog.Debug(ctx, "API Request", map[string]interface{}{
				"method": method,
				"url":    url,
			})
		}

		var reqBody io.Reader
		if jsonData != nil {
			reqBody = bytes.NewReader(jsonData)
		}
		req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
		if err != nil {
			return nil, fmt.Errorf("error creating request: %w", err)
		}

		req.Header.Set("api-key", c.APIKey)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")

		resp, err := c.HTTPClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("error performing request: %w", err)
		}
		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("error reading response body: %w", err)
		}

		tflog.Debug(ctx, "API Response", map[string]interface{}{
			"status": resp.StatusCode,
			"body":   string(respBody),
		})

		if resp.StatusCode == http.StatusTooManyRequests && attempt < c.MaxRetries {
			delay := retryDelay(resp.Header.Get("Retry-After"), attempt)
			tflog.Warn(ctx, "AlertOps API rate limit exceeded, retrying", map[string]interface{}{
				"method":  method,
				"url":     url,
				"attempt": attempt + 1,
				"delay":   delay.String(),
			})
			if err := c.sleep(ctx, delay); err != nil {
				return nil, err
			}
			continue
		}

		if resp.StatusCode >= 500 && method != http.MethodPost && serverErrors < maxServerErrorRetries {
			serverErrors++
			delay := serverErrorBackoff << (serverErrors - 1)
			tflog.Warn(ctx, "AlertOps API server error, retrying", map[string]interface{}{
				"method":  method,
				"url":     url,
				"status":  resp.StatusCode,
				"attempt": serverErrors,
				"delay":   delay.String(),
			})
			if err := c.sleep(ctx, delay); err != nil {
				return nil, err
			}
			continue
		}

		if resp.StatusCode == http.StatusNoContent {
			return []byte("{}"), nil
		}

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, &APIError{Method: method, Path: path, StatusCode: resp.StatusCode, Body: string(respBody)}
		}

		return respBody, nil
	}
}

// retryDelay honours a Retry-After header (in seconds) and otherwise backs off exponentially.
func retryDelay(retryAfter string, attempt int) time.Duration {
	if secs, err := strconv.Atoi(retryAfter); err == nil && secs > 0 {
		return time.Duration(secs) * time.Second
	}
	if attempt > 4 {
		return maxRetryBackoff
	}
	delay := initialRetryBackoff << attempt
	if delay > maxRetryBackoff {
		delay = maxRetryBackoff
	}
	return delay
}

// Get performs a GET request.
func (c *Client) Get(ctx context.Context, path string) ([]byte, error) {
	return c.doRequest(ctx, http.MethodGet, path, nil)
}

// Post performs a POST request.
func (c *Client) Post(ctx context.Context, path string, body interface{}) ([]byte, error) {
	return c.doRequest(ctx, http.MethodPost, path, body)
}

// Put performs a PUT request.
func (c *Client) Put(ctx context.Context, path string, body interface{}) ([]byte, error) {
	return c.doRequest(ctx, http.MethodPut, path, body)
}

// Delete performs a DELETE request.
func (c *Client) Delete(ctx context.Context, path string) ([]byte, error) {
	return c.doRequest(ctx, http.MethodDelete, path, nil)
}

// DeleteWithBody performs a DELETE request with a JSON body. Several AlertOps endpoints
// (e.g. detaching workflows) expect the items to remove in the request body.
func (c *Client) DeleteWithBody(ctx context.Context, path string, body interface{}) ([]byte, error) {
	return c.doRequest(ctx, http.MethodDelete, path, body)
}

// rateLimiter enforces several sliding-window limits shared by all requests of a client.
type rateLimiter struct {
	mu      sync.Mutex
	windows []*rateWindow
	now     func() time.Time
	sleep   func(ctx context.Context, d time.Duration) error
}

type rateWindow struct {
	limit  int
	period time.Duration
	sent   []time.Time
}

func newRateLimiter(requestsPerSecond, requestsPerMinute int) *rateLimiter {
	l := &rateLimiter{now: time.Now, sleep: sleepContext}
	if requestsPerSecond > 0 {
		l.windows = append(l.windows, &rateWindow{limit: requestsPerSecond, period: time.Second})
	}
	if requestsPerMinute > 0 {
		l.windows = append(l.windows, &rateWindow{limit: requestsPerMinute, period: time.Minute})
	}
	return l
}

// wait blocks until a request may be sent without exceeding any window, then records it.
func (l *rateLimiter) wait(ctx context.Context) error {
	for {
		l.mu.Lock()
		now := l.now()
		var delay time.Duration
		for _, w := range l.windows {
			expired := 0
			for expired < len(w.sent) && now.Sub(w.sent[expired]) >= w.period {
				expired++
			}
			w.sent = w.sent[expired:]
			if len(w.sent) >= w.limit {
				if d := w.period - now.Sub(w.sent[0]); d > delay {
					delay = d
				}
			}
		}
		if delay <= 0 {
			for _, w := range l.windows {
				w.sent = append(w.sent, now)
			}
			l.mu.Unlock()
			return nil
		}
		l.mu.Unlock()

		if err := l.sleep(ctx, delay); err != nil {
			return err
		}
	}
}

func sleepContext(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
