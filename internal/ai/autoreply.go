package ai

import (
	"strings"
	"time"
)

// ReplyEvent is an incoming message the auto-reply rules look at.
type ReplyEvent struct {
	ChatID string
	// IsGroup is a group chat; IsChannel is a status, channel or
	// broadcast, which never gets an auto-reply.
	IsGroup, IsChannel bool
	FromMe             bool
	// MentionedMe is set when a group message @mentions you.
	MentionedMe bool
	Text        string
	// New marks a message that just arrived, not history or an edit.
	New bool
}

// AutoReply is the pure rules for AI auto-reply: which incoming message
// gets a reply, and how often. It keeps no provider and sends nothing;
// the caller generates and sends when ShouldReply says yes.
type AutoReply struct {
	// Enabled is the global switch; AccountEnabled the account's.
	Enabled, AccountEnabled bool
	// AllowChats is the per-chat allowlist; nil allows every chat.
	AllowChats map[string]bool
	// GroupsRequireMention replies in groups only when @mentioned.
	GroupsRequireMention bool
	// Cooldown is the quiet time between replies in one chat.
	Cooldown time.Duration
	// MaxPerHour caps the replies in one chat per hour.
	MaxPerHour int
	// QuietStart and QuietEnd are hours (0-23) of the quiet hours;
	// equal means no quiet hours.
	QuietStart, QuietEnd int
	// HandoffWords stop auto-reply in a chat, e.g. "human", "admin".
	HandoffWords []string
	// Ghost is ghost mode: on, nothing is ever sent.
	Ghost bool

	now     func() time.Time
	last    map[string]time.Time   // last reply per chat
	sent    map[string][]time.Time // reply times per chat
	stopped map[string]bool        // handed over to a human
}

// NewAutoReply returns the rules with the defaults: groups only on
// @mention, a 60s cooldown per chat, at most 10 replies an hour per
// chat, "human" and "admin" hand over to a human.
func NewAutoReply(now func() time.Time) *AutoReply {
	if now == nil {
		now = time.Now
	}
	return &AutoReply{
		GroupsRequireMention: true,
		Cooldown:             60 * time.Second,
		MaxPerHour:           10,
		HandoffWords:         []string{"human", "admin"},
		now:                  now,
		last:                 map[string]time.Time{},
		sent:                 map[string][]time.Time{},
		stopped:              map[string]bool{},
	}
}

// ShouldReply reports whether ev gets an auto-reply. A handoff word in
// the message stops auto-reply in that chat, for good.
func (r *AutoReply) ShouldReply(ev ReplyEvent) bool {
	if r.Ghost || !r.Enabled || !r.AccountEnabled {
		return false
	}
	if !ev.New || ev.FromMe || ev.IsChannel {
		return false
	}
	if ev.IsGroup && r.GroupsRequireMention && !ev.MentionedMe {
		return false
	}
	if r.AllowChats != nil && !r.AllowChats[ev.ChatID] {
		return false
	}
	if r.stopped[ev.ChatID] {
		return false
	}
	if r.handoff(ev.Text) {
		r.stopped[ev.ChatID] = true
		return false
	}
	if r.quiet() {
		return false
	}
	now := r.now()
	if last, ok := r.last[ev.ChatID]; ok && now.Sub(last) < r.Cooldown {
		return false
	}
	if len(r.recent(ev.ChatID, now)) >= r.MaxPerHour {
		return false
	}
	return true
}

// handoff reports whether text asks for a human.
func (r *AutoReply) handoff(text string) bool {
	low := strings.ToLower(text)
	for _, w := range r.HandoffWords {
		if w != "" && strings.Contains(low, strings.ToLower(w)) {
			return true
		}
	}
	return false
}

// quiet reports whether now is in the quiet hours.
func (r *AutoReply) quiet() bool {
	if r.QuietStart == r.QuietEnd {
		return false
	}
	h := r.now().Hour()
	if r.QuietStart < r.QuietEnd {
		return h >= r.QuietStart && h < r.QuietEnd
	}
	return h >= r.QuietStart || h < r.QuietEnd
}

// recent lists the chat's replies in the last hour.
func (r *AutoReply) recent(chatID string, now time.Time) []time.Time {
	var out []time.Time
	for _, t := range r.sent[chatID] {
		if now.Sub(t) < time.Hour {
			out = append(out, t)
		}
	}
	return out
}

// NoteReply records a reply sent in chatID, for cooldown and cap.
func (r *AutoReply) NoteReply(chatID string) {
	now := r.now()
	r.last[chatID] = now
	r.sent[chatID] = append(r.recent(chatID, now), now)
}

// UserSent stops auto-reply in chatID: you took over the chat.
func (r *AutoReply) UserSent(chatID string) { r.stopped[chatID] = true }

// Stop hands chatID to a human; Resume turns auto-reply back on there.
func (r *AutoReply) Stop(chatID string)   { r.stopped[chatID] = true }
func (r *AutoReply) Resume(chatID string) { delete(r.stopped, chatID) }

// Stopped reports whether chatID is handed over to a human.
func (r *AutoReply) Stopped(chatID string) bool { return r.stopped[chatID] }
