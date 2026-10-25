package ai

import (
	"context"
	"strings"
)

// MockProvider answers with canned text, for tests and demo mode: it
// never touches the network.
type MockProvider struct {
	// Text is the answer, sent as one chunk.
	Text string
	// Err is returned by Chat and TestConnection, when set.
	Err error
	// Calls counts the Chat calls.
	Calls int
}

// Chat returns Text as one chunk, then Done.
func (m *MockProvider) Chat(ctx context.Context, req ChatRequest) (<-chan StreamChunk, error) {
	m.Calls++
	if m.Err != nil {
		return nil, m.Err
	}
	out := make(chan StreamChunk, 2)
	out <- StreamChunk{Text: strings.TrimSpace(m.Text)}
	out <- StreamChunk{Done: true}
	close(out)
	return out, nil
}

// TestConnection returns Err, or nil.
func (m *MockProvider) TestConnection(ctx context.Context) error { return m.Err }
