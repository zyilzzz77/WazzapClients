package ai

import (
	"context"
	"errors"
	"fmt"
)

// Message is one turn of a chat with the AI.
type Message struct {
	Role    string `json:"role"` // "system", "user" or "assistant"
	Content string `json:"content"`
}

// ChatRequest is a chat completion request.
type ChatRequest struct {
	Model       string
	Messages    []Message
	Temperature float64
}

// StreamChunk is one piece of a streaming answer.
type StreamChunk struct {
	Text string
	// Done marks the end of the answer.
	Done bool
}

// Provider is an AI that answers chat completions, streaming.
type Provider interface {
	// Chat asks req and streams the answer's chunks, ending with a
	// Done chunk. The channel closes on error too; check Done.
	Chat(ctx context.Context, req ChatRequest) (<-chan StreamChunk, error)
	// TestConnection checks the endpoint, key and model with a tiny
	// request.
	TestConnection(ctx context.Context) error
}

// Errors a provider can return, mapped to words the user can act on.
var (
	// ErrInvalidKey is a 401: the key is wrong or revoked.
	ErrInvalidKey = errors.New("the AI API key was refused")
	// ErrModelNotFound is a 404: no such model at this endpoint.
	ErrModelNotFound = errors.New("the AI model was not found at this endpoint")
	// ErrRateLimited is a 429: the provider says slow down.
	ErrRateLimited = errors.New("the AI provider is rate limiting requests")
	// ErrUnreachable is a network failure reaching the endpoint.
	ErrUnreachable = errors.New("the AI endpoint couldn't be reached")
	// ErrBadResponse is a response the client couldn't read.
	ErrBadResponse = errors.New("the AI endpoint gave an unexpected response")
)

// RateLimitedError is a 429 with the provider's Retry-After, when it
// gave one.
type RateLimitedError struct {
	RetryAfter string
}

// Is matches ErrRateLimited, so errors.Is works on it.
func (e *RateLimitedError) Is(target error) bool { return target == ErrRateLimited }

func (e *RateLimitedError) Error() string {
	if e.RetryAfter == "" {
		return ErrRateLimited.Error()
	}
	return ErrRateLimited.Error() + " (retry after " + e.RetryAfter + ")"
}

// UserError turns err into words for the user. It never includes a key,
// a URL with credentials, or a full prompt.
func UserError(err error) string {
	if err == nil {
		return ""
	}
	var rl *RateLimitedError
	switch {
	case errors.Is(err, ErrInvalidKey):
		return "The AI API key was refused. Check it in Settings > AI."
	case errors.Is(err, ErrModelNotFound):
		return "The AI model wasn't found at this endpoint. Check the model name in Settings > AI."
	case errors.As(err, &rl):
		return rl.Error() + ". Try again in a moment."
	case errors.Is(err, ErrRateLimited):
		return "The AI provider is rate limiting requests. Try again in a moment."
	case errors.Is(err, ErrUnreachable):
		return "The AI endpoint couldn't be reached. Check the base URL and your connection."
	case errors.Is(err, ErrBadResponse):
		return "The AI endpoint gave an unexpected response. Check the base URL in Settings > AI."
	case errors.Is(err, context.Canceled):
		return "The AI request was cancelled."
	case errors.Is(err, context.DeadlineExceeded):
		return "The AI request timed out. Try again."
	}
	return fmt.Sprintf("The AI request failed: %v", err)
}
