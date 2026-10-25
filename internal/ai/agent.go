package ai

import (
	"context"
	"errors"
	"strings"
)

// Agent answers questions about a chat through a Provider. It only ever
// sends text and captions: the caller hands it recent message texts,
// never media bytes, never view-once content, never another account's
// messages.
type Agent struct {
	Provider Provider
	Config   AIConfig
}

// NewAgent returns an agent for p under cfg.
func NewAgent(p Provider, cfg AIConfig) *Agent {
	return &Agent{Provider: p, Config: cfg}
}

// Enabled reports whether the agent can answer: turned on, with a
// provider behind it.
func (a *Agent) Enabled() bool {
	return a != nil && a.Config.Enabled && a.Provider != nil
}

// SystemPrompt is the prompt for chatName: the account's own, or the
// default, with the chat named.
func (a *Agent) SystemPrompt(chatName string) string {
	p := a.Config.EffectiveSystemPrompt()
	if chatName == "" {
		return p
	}
	return p + "\nThis chat is with " + chatName + "."
}

// BuildContext turns recent message texts into the messages the AI
// sees: the system prompt, then the last maxN, oldest first. Each line
// is truncated; empty ones are dropped.
func BuildContext(systemPrompt string, recent []string, maxN int) []Message {
	if maxN <= 0 {
		maxN = 20
	}
	if len(recent) > maxN {
		recent = recent[len(recent)-maxN:]
	}
	msgs := []Message{{Role: "system", Content: systemPrompt}}
	for _, r := range recent {
		if t := truncate(strings.TrimSpace(r), 1000); t != "" {
			msgs = append(msgs, Message{Role: "user", Content: t})
		}
	}
	return msgs
}

// truncate cuts s to n runes, marking the cut.
func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

// complete runs one request and collects its stream into one answer.
func (a *Agent) complete(ctx context.Context, msgs []Message) (string, error) {
	if a.Provider == nil {
		return "", errors.New("no AI provider is set up")
	}
	ch, err := a.Provider.Chat(ctx, ChatRequest{
		Model: a.Config.Model, Messages: msgs, Temperature: a.Config.Temperature,
	})
	if err != nil {
		return "", err
	}
	var b strings.Builder
	for chunk := range ch {
		if chunk.Done {
			break
		}
		b.WriteString(chunk.Text)
	}
	return strings.TrimSpace(b.String()), nil
}

// contextOf builds the messages for a question about chatName.
func (a *Agent) contextOf(chatName string, recent []string) []Message {
	return BuildContext(a.SystemPrompt(chatName), recent, a.Config.EffectiveMaxContext())
}

// Ask answers question about the chat's recent messages.
func (a *Agent) Ask(ctx context.Context, question, chatName string, recent []string) (string, error) {
	msgs := a.contextOf(chatName, recent)
	msgs = append(msgs, Message{Role: "user", Content: question})
	return a.complete(ctx, msgs)
}

// Summarize sums up the chat's recent messages in a few lines.
func (a *Agent) Summarize(ctx context.Context, chatName string, recent []string) (string, error) {
	msgs := a.contextOf(chatName, recent)
	msgs = append(msgs, Message{Role: "user", Content: "Summarise this conversation briefly, in a few short lines."})
	return a.complete(ctx, msgs)
}

// DraftReplies drafts three replies to the recent messages: a short
// one, a polite one and a casual one, in that order.
func (a *Agent) DraftReplies(ctx context.Context, chatName string, recent []string) ([3]string, error) {
	var out [3]string
	msgs := a.contextOf(chatName, recent)
	msgs = append(msgs, Message{Role: "user", Content: "Draft three short replies to the last message: " +
		"one short, one polite, one casual. Put each on its own line, numbered 1. 2. 3., nothing else."})
	ans, err := a.complete(ctx, msgs)
	if err != nil {
		return out, err
	}
	return parseDrafts(ans), nil
}

// parseDrafts pulls the three numbered lines out of an answer.
func parseDrafts(ans string) [3]string {
	var out [3]string
	i := 0
	for _, line := range strings.Split(ans, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		line = strings.TrimPrefix(line, "- ")
		for _, p := range []string{"1.", "2.", "3.", "1)", "2)", "3)"} {
			if rest, ok := strings.CutPrefix(line, p); ok {
				line = strings.TrimSpace(rest)
				break
			}
		}
		if i < 3 {
			out[i] = line
			i++
		}
	}
	return out
}

// Rewrite rewrites text in style ("shorter", "more polite", "more
// formal", ...), keeping its meaning and language.
func (a *Agent) Rewrite(ctx context.Context, text, style string) (string, error) {
	if strings.TrimSpace(style) == "" {
		style = "clearer"
	}
	msgs := []Message{
		{Role: "system", Content: a.Config.EffectiveSystemPrompt()},
		{Role: "user", Content: "Rewrite this text to be " + style + ", keeping its meaning:\n\n" + text},
	}
	return a.complete(ctx, msgs)
}

// Translate translates text into targetLang ("id" or "en", or a language
// name), returning only the translation.
func (a *Agent) Translate(ctx context.Context, text, targetLang string) (string, error) {
	lang := map[string]string{"id": "Indonesian", "en": "English"}[targetLang]
	if lang == "" {
		lang = targetLang
	}
	msgs := []Message{
		{Role: "system", Content: a.Config.EffectiveSystemPrompt()},
		{Role: "user", Content: "Translate this text into " + lang + ". Reply with only the translation:\n\n" + text},
	}
	return a.complete(ctx, msgs)
}

// AutoReply drafts the auto-reply to the chat's recent messages: one
// short reply, in the chat's language.
func (a *Agent) AutoReply(ctx context.Context, chatName string, recent []string) (string, error) {
	msgs := a.contextOf(chatName, recent)
	msgs = append(msgs, Message{Role: "user", Content: "Write one short, friendly reply to the last message, " +
		"in the language of the conversation. Reply with only that reply, nothing else."})
	return a.complete(ctx, msgs)
}
