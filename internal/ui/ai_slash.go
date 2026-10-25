package ui

import (
	"context"
	"errors"
	"strings"

	"github.com/chomosuke9/wazzapclients/internal/command"
	"github.com/chomosuke9/wazzapclients/internal/model"
)

// aiAnswerer is the command package's AI, backed by this UI's agent.
// /ai and /aisummary run through it; nil agent means AI isn't set up.
type aiAnswerer struct{ u *UI }

var errAINotReady = errors.New("AI isn't set up yet. Turn it on in Settings > AI.")

func (a aiAnswerer) Answer(ctx context.Context, question string, chat *model.Chat, recent []string) (string, error) {
	ag := a.u.ai.agent
	if ag == nil {
		return "", errAINotReady
	}
	return ag.Ask(ctx, question, chat.Name, recent)
}

func (a aiAnswerer) Summarize(ctx context.Context, chat *model.Chat, recent []string) (string, error) {
	ag := a.u.ai.agent
	if ag == nil {
		return "", errAINotReady
	}
	return ag.Summarize(ctx, chat.Name, recent)
}

// aiRecent is the chat's recent texts for the agent: text and captions
// only, never view-once content, never another chat's messages.
func (u *UI) aiRecent(chatID string, n int) []string {
	if u.backend == nil {
		return nil
	}
	msgs := u.backend.Messages(chatID, n)
	out := make([]string, 0, len(msgs))
	for _, m := range msgs {
		if m == nil || m.Kind == model.KindViewOnce {
			continue
		}
		if t := strings.TrimSpace(m.Text); t != "" {
			if m.Sender != "" && !m.FromMe {
				t = m.Sender + ": " + t
			}
			out = append(out, t)
		}
	}
	return out
}

var _ command.AIAnswerer = aiAnswerer{}
