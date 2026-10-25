package auto

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/chomosuke9/wazzapclients/internal/ai"
	"github.com/chomosuke9/wazzapclients/internal/model"
)

// aiPending is an auto-reply the agent finished, waiting for Poll to
// send it on the UI goroutine.
type aiPending struct {
	chatID  string
	replyTo *model.Message
	text    string
}

// aiState is the AI auto-reply agent attached to a Backend: its rules,
// its agent, and the replies waiting to go out.
type aiState struct {
	mu         sync.Mutex
	agent      *ai.Agent
	rules      *ai.AutoReply
	account    string // the account's name, for the prompt
	pending    []aiPending
	generating map[string]bool
}

// SetAIHook attaches the AI auto-reply agent (or, with a nil agent,
// detaches it). The rules decide what gets a reply; the agent writes it.
func (a *Backend) SetAIHook(agent *ai.Agent, rules *ai.AutoReply, account string) {
	if agent == nil || rules == nil {
		a.ai = nil
		return
	}
	a.ai = &aiState{agent: agent, rules: rules, account: account, generating: map[string]bool{}}
}

// aiSeen looks at a message for the AI auto-reply agent: your own sends
// hand the chat to you, and a message the rules allow starts a reply
// being generated, off the UI goroutine.
func (a *Backend) aiSeen(e model.MessageEvent) {
	st := a.ai
	if st == nil || e.Msg == nil {
		return
	}
	m := e.Msg
	if m.FromMe {
		st.rules.UserSent(m.ChatID)
		return
	}
	ev := ai.ReplyEvent{
		ChatID: m.ChatID, IsGroup: a.isGroup(m.ChatID),
		IsChannel: strings.HasSuffix(m.ChatID, "@newsletter") ||
			strings.HasSuffix(m.ChatID, "@broadcast"),
		MentionedMe: mentionsMe(m.Text), Text: m.Text, New: e.New,
	}
	st.mu.Lock()
	if st.rules.Ghost != a.ghost {
		st.rules.Ghost = a.ghost
	}
	if !st.rules.ShouldReply(ev) || st.generating[m.ChatID] {
		st.mu.Unlock()
		return
	}
	st.generating[m.ChatID] = true
	st.mu.Unlock()
	recent := aiRecent(a.Backend, m.ChatID, st.agent.Config.EffectiveMaxContext())
	name := chatName(a.Backend, m.ChatID)
	go st.generate(a, m, name, recent)
}

// generate asks the agent for a reply, then queues it for Poll to send.
func (st *aiState) generate(a *Backend, m *model.Message, name string, recent []string) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	text, err := st.agent.AutoReply(ctx, name, recent)
	st.mu.Lock()
	delete(st.generating, m.ChatID)
	if err == nil && text != "" {
		st.pending = append(st.pending, aiPending{chatID: m.ChatID, replyTo: m, text: text})
	}
	st.mu.Unlock()
	if a.notify != nil {
		a.notify() // Poll drains the reply
	}
}

// aiDrain sends the auto-replies the agent finished. Ghost mode drops
// them instead: it never sends.
func (a *Backend) aiDrain() {
	st := a.ai
	if st == nil {
		return
	}
	st.mu.Lock()
	pending := st.pending
	st.pending = nil
	st.mu.Unlock()
	for _, p := range pending {
		if a.ghost {
			continue
		}
		if m := a.Backend.Send(p.chatID, model.Draft{Text: p.text, Reply: p.replyTo}); m != nil {
			a.ours[m.ID] = true
			st.rules.NoteReply(p.chatID)
			a.out = append(a.out, model.MessageEvent{Msg: m})
		}
	}
}

// aiRecent is the chat's recent texts for the agent: text and captions
// only, never view-once content.
func aiRecent(b model.Backend, chatID string, n int) []string {
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
			out = append(out, t)
		}
	}
	return out
}

// chatName is the chat's name, or "" when it isn't known.
func chatName(b model.Backend, chatID string) string {
	for _, c := range b.Chats() {
		if c.ID == chatID {
			return c.Name
		}
	}
	return ""
}
