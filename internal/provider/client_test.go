package provider

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// fakeClock lets limiter tests advance time without sleeping.
type fakeClock struct {
	now   time.Time
	slept []time.Duration
}

func (c *fakeClock) Now() time.Time { return c.now }

func (c *fakeClock) Sleep(_ context.Context, d time.Duration) error {
	c.slept = append(c.slept, d)
	c.now = c.now.Add(d)
	return nil
}

func TestRateLimiterEnforcesWindows(t *testing.T) {
	clock := &fakeClock{now: time.Unix(1_700_000_000, 0)}
	l := newRateLimiter(5, 15)
	l.now, l.sleep = clock.Now, clock.Sleep

	start := clock.now
	for i := 0; i < 16; i++ {
		if err := l.wait(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	// 15 requests fit in the first minute (three per-second bursts of 5); the 16th waits for the minute window.
	if elapsed := clock.now.Sub(start); elapsed < time.Minute {
		t.Fatalf("16 requests finished after %s, want >= 1m", elapsed)
	}
	if len(clock.slept) == 0 {
		t.Fatal("expected the limiter to sleep")
	}
}

func TestRateLimiterPerSecond(t *testing.T) {
	clock := &fakeClock{now: time.Unix(1_700_000_000, 0)}
	l := newRateLimiter(2, 0)
	l.now, l.sleep = clock.Now, clock.Sleep

	for i := 0; i < 3; i++ {
		if err := l.wait(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	if len(clock.slept) != 1 || clock.slept[0] != time.Second {
		t.Fatalf("slept %v, want a single 1s sleep", clock.slept)
	}
}

func TestRetryDelay(t *testing.T) {
	if got := retryDelay("7", 0); got != 7*time.Second {
		t.Errorf("Retry-After: got %s, want 7s", got)
	}
	if got := retryDelay("", 0); got != 5*time.Second {
		t.Errorf("attempt 0: got %s, want 5s", got)
	}
	if got := retryDelay("", 1); got != 10*time.Second {
		t.Errorf("attempt 1: got %s, want 10s", got)
	}
	if got := retryDelay("", 10); got != maxRetryBackoff {
		t.Errorf("attempt 10: got %s, want %s", got, maxRetryBackoff)
	}
}

func newTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	c := NewClientWithOptions("test-key", srv.URL, 0, 0)
	c.sleep = func(context.Context, time.Duration) error { return nil }
	return c
}

func TestClientRetriesOn429(t *testing.T) {
	var calls int32
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("api-key") != "test-key" {
			t.Errorf("missing api-key header")
		}
		if atomic.AddInt32(&calls, 1) <= 2 {
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte("API calls quota exceeded! maximum admitted 15 per Minute"))
			return
		}
		_, _ = w.Write([]byte(`{"ok":true}`))
	})

	body, err := c.Get(context.Background(), "/topics")
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != `{"ok":true}` || atomic.LoadInt32(&calls) != 3 {
		t.Fatalf("body=%s calls=%d", body, calls)
	}
}

func TestClientGivesUpAfterMaxRetries(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	})
	c.MaxRetries = 2

	_, err := c.Get(context.Background(), "/topics")
	apiErr, ok := err.(*APIError)
	if !ok || apiErr.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("got %v, want a 429 APIError", err)
	}
}

func TestClientRetriesServerErrorsForIdempotentMethods(t *testing.T) {
	var calls int32
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&calls, 1) == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	if _, err := c.Delete(context.Background(), "/integrations/inbound/1"); err != nil {
		t.Fatalf("DELETE should succeed after a transient 500: %v", err)
	}
	if atomic.LoadInt32(&calls) != 2 {
		t.Fatalf("calls = %d, want 2", calls)
	}
}

func TestClientDoesNotRetryPostOnServerError(t *testing.T) {
	var calls int32
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusInternalServerError)
	})

	_, err := c.Post(context.Background(), "/topics", map[string]string{"topic_name": "x"})
	if apiErr, ok := err.(*APIError); !ok || apiErr.StatusCode != http.StatusInternalServerError {
		t.Fatalf("got %v, want a 500 APIError", err)
	}
	if atomic.LoadInt32(&calls) != 1 {
		t.Fatalf("POST was sent %d times, want exactly once", calls)
	}
}

func TestClientGivesUpAfterServerErrorRetries(t *testing.T) {
	var calls int32
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusBadGateway)
	})

	if _, err := c.Get(context.Background(), "/topics/1"); err == nil {
		t.Fatal("expected an error")
	}
	if got := atomic.LoadInt32(&calls); got != maxServerErrorRetries+1 {
		t.Fatalf("calls = %d, want %d", got, maxServerErrorRetries+1)
	}
}

func TestClientNotFoundAndNoContent(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/topics/1" {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"errors":[{"message":"Topic: 1 not found"}]}`))
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	if _, err := c.Get(context.Background(), "/topics/1"); !IsNotFound(err) {
		t.Fatalf("got %v, want not found", err)
	}
	body, err := c.Put(context.Background(), "/topics/2", map[string]string{"topic_name": "x"})
	if err != nil || string(body) != "{}" {
		t.Fatalf("204: body=%s err=%v", body, err)
	}
	if IsNotFound(nil) {
		t.Fatal("nil error reported as not found")
	}
}

func TestClientDeleteWithBody(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		if r.Method != http.MethodDelete || string(b) != `[{"workflow_id":5}]` {
			t.Errorf("got %s %s", r.Method, b)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	if _, err := c.DeleteWithBody(context.Background(), "/response_plays/1/workflows", []map[string]int{{"workflow_id": 5}}); err != nil {
		t.Fatal(err)
	}
}
