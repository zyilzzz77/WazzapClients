package ai

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// fakeServer is an OpenAI-compatible endpoint for tests: it streams
// chunks, or answers with a status.
func fakeServer(t *testing.T, chunks []string, status int) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Errorf("path %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Errorf("authorization %q", got)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("content type %q", r.Header.Get("Content-Type"))
		}
		if status != http.StatusOK {
			w.Header().Set("Retry-After", "30")
			w.WriteHeader(status)
			fmt.Fprint(w, `{"error":{"message":"nope"}}`)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		for _, c := range chunks {
			fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"content\":%q}}]}\n\n", c)
		}
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
}

func collect(t *testing.T, ch <-chan StreamChunk) string {
	t.Helper()
	var b strings.Builder
	timeout := time.After(5 * time.Second)
	for {
		select {
		case c, ok := <-ch:
			if !ok {
				t.Fatal("channel closed without Done")
			}
			if c.Done {
				return b.String()
			}
			b.WriteString(c.Text)
		case <-timeout:
			t.Fatal("stream timed out")
		}
	}
}

func TestClientStream(t *testing.T) {
	srv := fakeServer(t, []string{"Hello", " ", "world"}, http.StatusOK)
	defer srv.Close()
	c := &Client{BaseURL: srv.URL, APIKey: "test-key", Model: "test-model"}
	ch, err := c.Chat(context.Background(), ChatRequest{
		Model:    "test-model",
		Messages: []Message{{Role: "user", Content: "hi"}},
	})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if got := collect(t, ch); got != "Hello world" {
		t.Fatalf("stream %q", got)
	}
	// The key goes in a header, never in the URL.
	if strings.Contains(srv.URL, "test-key") {
		t.Fatal("key in URL")
	}
}

func TestClientErrors(t *testing.T) {
	cases := []struct {
		status int
		want   error
	}{
		{http.StatusUnauthorized, ErrInvalidKey},
		{http.StatusForbidden, ErrInvalidKey},
		{http.StatusNotFound, ErrModelNotFound},
		{http.StatusTooManyRequests, ErrRateLimited},
		{http.StatusInternalServerError, ErrBadResponse},
	}
	for _, tc := range cases {
		srv := fakeServer(t, nil, tc.status)
		c := &Client{BaseURL: srv.URL, APIKey: "test-key", Model: "m"}
		_, err := c.Chat(context.Background(), ChatRequest{Messages: []Message{{Role: "user", Content: "x"}}})
		srv.Close()
		if !errors.Is(err, tc.want) {
			t.Errorf("status %d: err %v, want %v", tc.status, err, tc.want)
		}
		if tc.status == http.StatusTooManyRequests {
			var rl *RateLimitedError
			if !errors.As(err, &rl) || rl.RetryAfter != "30" {
				t.Errorf("429: Retry-After not kept: %v", err)
			}
			if !strings.Contains(UserError(err), "30") {
				t.Errorf("429 user text drops Retry-After: %q", UserError(err))
			}
		}
	}
	// Unreachable endpoint.
	c := &Client{BaseURL: "http://127.0.0.1:1/v1", APIKey: "k", Model: "m"}
	if _, err := c.Chat(context.Background(), ChatRequest{Messages: []Message{{Role: "user", Content: "x"}}}); !errors.Is(err, ErrUnreachable) {
		t.Errorf("unreachable: %v", err)
	}
	if err := c.TestConnection(context.Background()); !errors.Is(err, ErrUnreachable) {
		t.Errorf("unreachable test: %v", err)
	}
}

func TestClientTestConnection(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"choices":[{"message":{"content":"pong"}}]}`)
	}))
	defer srv.Close()
	c := &Client{BaseURL: srv.URL, APIKey: "test-key", Model: "m"}
	if err := c.TestConnection(context.Background()); err != nil {
		t.Fatalf("test connection: %v", err)
	}
	bad := fakeServer(t, nil, http.StatusUnauthorized)
	defer bad.Close()
	c2 := &Client{BaseURL: bad.URL, APIKey: "test-key", Model: "m"}
	if err := c2.TestConnection(context.Background()); !errors.Is(err, ErrInvalidKey) {
		t.Fatalf("bad key test: %v", err)
	}
}

func TestSSEParsing(t *testing.T) {
	// A heartbeat line, a comment and a chunk with no content are
	// skipped; [DONE] ends the stream.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, ": heartbeat\n\ndata: {\"choices\":[{\"delta\":{}}]}\n\n")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"yo\"}}]}\n\n")
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()
	c := &Client{BaseURL: srv.URL, APIKey: "test-key", Model: "m"}
	ch, err := c.Chat(context.Background(), ChatRequest{Messages: []Message{{Role: "user", Content: "x"}}})
	if err != nil {
		t.Fatal(err)
	}
	if got := collect(t, ch); got != "yo" {
		t.Fatalf("stream %q", got)
	}
}

func TestUserError(t *testing.T) {
	if got := UserError(ErrInvalidKey); !strings.Contains(got, "Settings > AI") {
		t.Errorf("401 text: %q", got)
	}
	if got := UserError(ErrModelNotFound); !strings.Contains(got, "model") {
		t.Errorf("404 text: %q", got)
	}
	if got := UserError(nil); got != "" {
		t.Errorf("nil error: %q", got)
	}
}
