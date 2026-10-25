package command

import (
	"context"
	"errors"
	"strings"

	"github.com/chomosuke9/wazzapclients/internal/model"
)

// AIAnswerer answers a question about a chat, using its recent messages.
// The UI implements it with the AI agent; nil means AI isn't set up.
type AIAnswerer interface {
	Answer(ctx context.Context, question string, chat *model.Chat, recent []string) (string, error)
	Summarize(ctx context.Context, chat *model.Chat, recent []string) (string, error)
}

func init() {
	All = append(All,
		&Command{Name: "ai", Description: "Asks the AI about this chat's recent messages",
			Options: []Option{{Name: "question", Description: "What to ask", Kind: Text, Required: true}},
			Run:     runAI},
		&Command{Name: "aisummary", Description: "Asks the AI to summarise this chat's recent messages",
			Run: runAISummary},
	)
}

// errNoAI is what /ai says where AI isn't set up.
var errNoAI = errors.New("AI isn't set up yet. Turn it on in Settings > AI.")

// recentTexts is the chat's recent texts for the AI: text and captions
// only, never view-once content.
func recentTexts(b model.Backend, chatID string, n int) []string {
	msgs := b.Messages(chatID, n)
	out := make([]string, 0, len(msgs))
	for _, m := range msgs {
		if m == nil || m.Kind == model.KindViewOnce {
			continue
		}
		if t := strings.TrimSpace(m.Text); t != "" {
			if m.Sender != "" && !m.FromMe {
				t = m.Sender + ": " + t
			}
			out = append(out, truncateRunes(t, 500))
		}
	}
	return out
}

func truncateRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

// runAI answers a question about the chat, as a local note: nothing is
// sent to WhatsApp unless the user taps Send.
func runAI(c *Context) error {
	if c.AI == nil {
		return errNoAI
	}
	q := strings.TrimSpace(c.Text("question"))
	recent := recentTexts(c.Backend, c.Chat.ID, 20)
	n := busy(c, "Asking the AI…")
	chat := c.Chat
	c.Do(func() func() {
		ans, err := c.AI.Answer(context.Background(), q, chat, recent)
		return func() {
			if err != nil {
				fail(n, err.Error())
				return
			}
			n.Busy = false
			n.Text = "🤖 " + ans
			n.Buttons = aiSendButtons(c, n, ans)
		}
	})
	return nil
}

// runAISummary summarises the chat's recent messages, as a local note.
func runAISummary(c *Context) error {
	if c.AI == nil {
		return errNoAI
	}
	recent := recentTexts(c.Backend, c.Chat.ID, 20)
	n := busy(c, "Summarising…")
	chat := c.Chat
	c.Do(func() func() {
		ans, err := c.AI.Summarize(context.Background(), chat, recent)
		return func() {
			if err != nil {
				fail(n, err.Error())
				return
			}
			n.Busy = false
			n.Text = "🤖 " + ans
			n.Buttons = aiSendButtons(c, n, ans)
		}
	})
	return nil
}

// aiSendButtons offers to send an AI answer to the chat, or dismiss it.
func aiSendButtons(c *Context, n *Note, ans string) []Button {
	return []Button{
		{Label: "Send", Run: func() {
			m := c.Backend.Send(c.Chat.ID, model.Draft{Text: ans})
			if m != nil {
				c.Sent(m)
			}
			n.Text, n.Buttons = "Sent:\n"+ans, nil
		}},
		{Label: "Dismiss", Run: func() { c.Dismiss(n) }},
	}
}
