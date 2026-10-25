package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

// Timeouts: 10s to connect, 120s for a whole (streaming) request.
const (
	connectTimeout = 10 * time.Second
	requestTimeout = 120 * time.Second
	testTimeout    = 30 * time.Second
)

// Client is an OpenAI-compatible Chat Completions client, on stdlib
// net/http only. It works with OpenAI, Groq, OpenRouter, Gemini's
// OpenAI-compatible endpoint, Ollama, LM Studio and any custom endpoint
// that speaks the same shape.
type Client struct {
	BaseURL string
	APIKey  string
	Model   string
	// HTTP is the transport; nil uses one with a 10s connect timeout.
	HTTP *http.Client
}

func (c *Client) httpClient() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return &http.Client{
		Transport: &http.Transport{
			DialContext: (&net.Dialer{Timeout: connectTimeout}).DialContext,
		},
	}
}

// endpoint is BaseURL joined with path, without a trailing slash.
func (c *Client) endpoint(path string) string {
	return strings.TrimSuffix(c.BaseURL, "/") + path
}

// chatBody is the JSON of a chat completions request.
type chatBody struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature,omitempty"`
	Stream      bool      `json:"stream"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
}

// newRequest builds a chat completions request with the key as a Bearer
// header (never in the URL).
func (c *Client) newRequest(ctx context.Context, body chatBody) (*http.Request, error) {
	b, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint("/chat/completions"), bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}
	return req, nil
}

// Chat posts req with streaming on, and streams the answer's text.
func (c *Client) Chat(ctx context.Context, req ChatRequest) (<-chan StreamChunk, error) {
	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	model := req.Model
	if model == "" {
		model = c.Model
	}
	httpReq, err := c.newRequest(ctx, chatBody{
		Model: model, Messages: req.Messages, Temperature: req.Temperature, Stream: true,
	})
	if err != nil {
		cancel()
		return nil, err
	}
	resp, err := c.httpClient().Do(httpReq)
	if err != nil {
		cancel()
		return nil, mapNetError(err)
	}
	if resp.StatusCode != http.StatusOK {
		err := mapStatus(resp)
		cancel()
		return nil, err
	}
	out := make(chan StreamChunk)
	go c.stream(ctx, cancel, resp, out)
	return out, nil
}

// TestConnection sends a tiny non-streaming request: it proves the
// endpoint, key and model all work, without asking for a real answer.
func (c *Client) TestConnection(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, testTimeout)
	defer cancel()
	httpReq, err := c.newRequest(ctx, chatBody{
		Model: c.Model, Messages: []Message{{Role: "user", Content: "ping"}},
		Stream: false, MaxTokens: 1,
	})
	if err != nil {
		return err
	}
	resp, err := c.httpClient().Do(httpReq)
	if err != nil {
		return mapNetError(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return mapStatus(resp)
	}
	// A 2xx with a body the client can't read is still a failure: the
	// endpoint isn't speaking chat completions.
	if _, err := io.ReadAll(io.LimitReader(resp.Body, maxErrorBody)); err != nil {
		return ErrBadResponse
	}
	return nil
}
