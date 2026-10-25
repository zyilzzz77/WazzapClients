package ai

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"strings"
)

// maxSSELine is one SSE line at most.
const maxSSELine = 1 << 20

// stream reads the SSE answer into out, then closes it.
func (c *Client) stream(ctx context.Context, cancel context.CancelFunc, resp *http.Response, out chan<- StreamChunk) {
	defer cancel()
	defer resp.Body.Close()
	defer close(out)
	sc := bufio.NewScanner(resp.Body)
	sc.Buffer(make([]byte, 0, 64<<10), maxSSELine)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, ":") {
			continue // a heartbeat or comment
		}
		data, ok := strings.CutPrefix(line, "data:")
		if !ok {
			continue
		}
		data = strings.TrimSpace(data)
		if data == "[DONE]" {
			out <- StreamChunk{Done: true}
			return
		}
		text, err := sseText(data)
		if err != nil {
			continue // one bad chunk doesn't end the answer
		}
		if text != "" {
			select {
			case out <- StreamChunk{Text: text}:
			case <-ctx.Done():
				return
			}
		}
	}
	out <- StreamChunk{Done: true}
}

// sseText pulls the delta content out of one SSE data JSON object.
func sseText(data string) (string, error) {
	var ev struct {
		Choices []struct {
			Delta struct {
				Content string `json:"content"`
			} `json:"delta"`
		} `json:"choices"`
	}
	if err := json.Unmarshal([]byte(data), &ev); err != nil {
		return "", err
	}
	if len(ev.Choices) == 0 {
		return "", nil
	}
	return ev.Choices[0].Delta.Content, nil
}
