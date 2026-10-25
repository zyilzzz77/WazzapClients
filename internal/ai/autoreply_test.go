package ai

import (
	"testing"
	"time"
)

var base = time.Date(2026, time.October, 25, 15, 0, 0, 0, time.UTC)

func testTime() time.Time { return base }

func newRules(now time.Time) (*AutoReply, func(time.Time)) {
	cur := now
	r := NewAutoReply(func() time.Time { return cur })
	r.Enabled, r.AccountEnabled = true, true
	return r, func(t time.Time) { cur = t }
}

func TestShouldReplyBasics(t *testing.T) {
	r, _ := newRules(base)
	ev := ReplyEvent{ChatID: "rina", Text: "hi", New: true}
	if !r.ShouldReply(ev) {
		t.Fatal("a plain 1:1 message should get a reply")
	}
	cases := []struct {
		name string
		ev   ReplyEvent
		mod  func(*AutoReply)
	}{
		{"old message", ReplyEvent{ChatID: "rina", Text: "hi"}, nil},
		{"own message", ReplyEvent{ChatID: "rina", Text: "hi", New: true, FromMe: true}, nil},
		{"channel", ReplyEvent{ChatID: "news@newsletter", Text: "hi", New: true, IsChannel: true}, nil},
		{"group without mention", ReplyEvent{ChatID: "g@g.us", Text: "hi", New: true, IsGroup: true}, nil},
		{"global off", ev, func(r *AutoReply) { r.Enabled = false }},
		{"account off", ev, func(r *AutoReply) { r.AccountEnabled = false }},
		{"ghost on", ev, func(r *AutoReply) { r.Ghost = true }},
		{"chat not allowed", ev, func(r *AutoReply) { r.AllowChats = map[string]bool{"other": true} }},
	}
	for _, tc := range cases {
		rr, _ := newRules(base)
		if tc.mod != nil {
			tc.mod(rr)
		}
		if rr.ShouldReply(tc.ev) {
			t.Errorf("%s: replied", tc.name)
		}
	}
	// A group @mention does get one.
	if r2, _ := newRules(base); !r2.ShouldReply(ReplyEvent{ChatID: "g@g.us", Text: "hi", New: true, IsGroup: true, MentionedMe: true}) {
		t.Fatal("group @mention should get a reply")
	}
}

func TestShouldReplyCooldownAndCap(t *testing.T) {
	r, setNow := newRules(base)
	ev := ReplyEvent{ChatID: "rina", Text: "hi", New: true}
	if !r.ShouldReply(ev) {
		t.Fatal("first message")
	}
	r.NoteReply("rina")
	if r.ShouldReply(ev) {
		t.Fatal("replied inside the 60s cooldown")
	}
	setNow(base.Add(61 * time.Second))
	if !r.ShouldReply(ev) {
		t.Fatal("no reply after the cooldown")
	}
	// Ten replies in an hour: the eleventh is refused.
	r2, setNow2 := newRules(base)
	for i := 0; i < 10; i++ {
		if !r2.ShouldReply(ev) {
			t.Fatalf("reply %d refused", i+1)
		}
		r2.NoteReply("rina")
		setNow2(base.Add(time.Duration(i+1) * 61 * time.Second))
	}
	// Still inside the hour (10 x 61s = 610s): the cap is hit.
	if r2.ShouldReply(ev) {
		t.Fatal("replied past the hourly cap")
	}
	setNow2(base.Add(2 * time.Hour))
	if !r2.ShouldReply(ev) {
		t.Fatal("no reply once the hour passed")
	}
	// Another chat has its own budget.
	if r2.ShouldReply(ReplyEvent{ChatID: "budi", Text: "hi", New: true}) == false {
		t.Fatal("another chat shares the cap")
	}
}

func TestShouldReplyQuietHours(t *testing.T) {
	r, setNow := newRules(base)
	r.QuietStart, r.QuietEnd = 23, 7
	ev := ReplyEvent{ChatID: "rina", Text: "hi", New: true}
	setNow(time.Date(2026, time.October, 25, 2, 0, 0, 0, time.UTC))
	if r.ShouldReply(ev) {
		t.Fatal("replied in the quiet hours")
	}
	setNow(time.Date(2026, time.October, 25, 12, 0, 0, 0, time.UTC))
	if !r.ShouldReply(ev) {
		t.Fatal("no reply outside the quiet hours")
	}
	// A daytime window works too.
	r.QuietStart, r.QuietEnd = 13, 14
	setNow(time.Date(2026, time.October, 25, 13, 30, 0, 0, time.UTC))
	if r.ShouldReply(ev) {
		t.Fatal("replied in the daytime quiet hours")
	}
}

func TestShouldReplyHandoff(t *testing.T) {
	r, _ := newRules(base)
	ev := ReplyEvent{ChatID: "rina", Text: "I want to talk to a human", New: true}
	if r.ShouldReply(ev) {
		t.Fatal("replied to a handoff word")
	}
	if !r.Stopped("rina") {
		t.Fatal("chat not stopped after handoff")
	}
	if r.ShouldReply(ReplyEvent{ChatID: "rina", Text: "hello?", New: true}) {
		t.Fatal("replied after handoff")
	}
	r.Resume("rina")
	if !r.ShouldReply(ReplyEvent{ChatID: "rina", Text: "hello?", New: true}) {
		t.Fatal("no reply after resume")
	}
	// Your own send stops it too, and only for that chat.
	r.UserSent("budi")
	if !r.Stopped("budi") || r.Stopped("rina") {
		t.Fatal("UserSent stopped the wrong chat")
	}
}
