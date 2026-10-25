package ui

import (
	"context"
	"image"
	"sync"
	"time"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/widget"
)

// aiSuggestState is the composer's "Suggest replies": three drafts from
// the chat's recent messages. Tap one to put it in the composer; the
// user still sends it.
type aiSuggestState struct {
	mu    sync.Mutex
	btn   widget.Clickable
	busy  bool
	gen   int
	items []string
	// clicks are fixed, one per draft slot, so the goroutine that
	// replaces items never touches them.
	clicks  [3]widget.Clickable
	dismiss widget.Clickable
}

// requestAISuggest asks the agent for three reply drafts, off the UI
// goroutine.
func (u *UI) requestAISuggest() {
	s := &u.ai.suggest
	ag := u.ai.agent
	if ag == nil || u.selected == nil {
		return
	}
	s.mu.Lock()
	if s.busy {
		s.mu.Unlock()
		return
	}
	s.busy = true
	s.gen++
	gen := s.gen
	s.mu.Unlock()
	chat := u.selected
	recent := u.aiRecent(chat.ID, u.ai.cfg.EffectiveMaxContext())
	name := chat.Name
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()
		drafts, err := ag.DraftReplies(ctx, name, recent)
		s.mu.Lock()
		defer s.mu.Unlock()
		if gen != s.gen {
			return // a newer request is running
		}
		s.busy = false
		if err == nil {
			s.items = drafts[:]
		}
		if u.window != nil {
			u.window.Invalidate()
		}
	}()
}

// layoutAISuggestButton is the ✨ button beside the composer's emoji
// button. It shows only while AI is ready.
func (u *UI) layoutAISuggestButton(gtx C) D {
	s := &u.ai.suggest
	if s.btn.Clicked(gtx) {
		u.requestAISuggest()
	}
	if !u.aiReady() {
		return D{}
	}
	return clickable(gtx, &s.btn, func(gtx C) D {
		sz := gtx.Dp(40)
		if h := u.hover(gtx, &s.btn); h > 0 {
			fillCircle(gtx, image.Pt(sz/2, sz/2), sz/2, faded(u.pal.Hover, h))
		}
		return centerIn(gtx, sz, u.label(20, "✨", u.pal.IconStrong).Layout)
	})
}

// layoutAISuggestBar is the row of suggested replies above the
// composer, and the "thinking" line while they generate.
func (u *UI) layoutAISuggestBar(gtx C) D {
	s := &u.ai.suggest
	if s.dismiss.Clicked(gtx) {
		s.mu.Lock()
		s.items = nil
		s.mu.Unlock()
	}
	s.mu.Lock()
	items, busy := append([]string(nil), s.items...), s.busy
	s.mu.Unlock()
	if !u.aiReady() || (!busy && len(items) == 0) {
		return D{}
	}
	if busy {
		return layout.Inset{Left: 8, Bottom: 4}.Layout(gtx,
			u.label(14, "✨ Thinking…", u.pal.TextSecondary).Layout)
	}
	return layout.Inset{Left: 4, Right: 4, Bottom: 6}.Layout(gtx, func(gtx C) D {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
			u.suggestChips(gtx, items)...,
		)
	})
}

// suggestChips lays out one chip per draft, then a ✕ to dismiss them.
func (u *UI) suggestChips(gtx C, items []string) []layout.FlexChild {
	s := &u.ai.suggest
	var out []layout.FlexChild
	for i, text := range items {
		if text == "" {
			continue
		}
		i, text := i, text
		c := &s.clicks[i]
		if c.Clicked(gtx) {
			u.insertSuggestion(text)
			s.mu.Lock()
			s.items = nil
			s.mu.Unlock()
		}
		out = append(out, layout.Rigid(func(gtx C) D {
			return clickable(gtx, c, func(gtx C) D {
				m := op.Record(gtx.Ops)
				dims := layout.Inset{Top: 6, Bottom: 6, Left: 10, Right: 10}.Layout(gtx,
					u.label(14, text, u.pal.Text, labelOpts{maxLines: 1}).Layout)
				call := m.Stop()
				fillRRect(gtx, image.Rectangle{Max: dims.Size}, gtx.Dp(12), u.pal.Hover)
				call.Add(gtx.Ops)
				return dims
			})
		}), layout.Rigid(layout.Spacer{Width: 6}.Layout))
	}
	out = append(out, layout.Rigid(func(gtx C) D {
		return clickable(gtx, &s.dismiss, u.label(14, "✕", u.pal.TextSecondary).Layout)
	}))
	return out
}

// insertSuggestion puts text in the composer, after what is there.
func (u *UI) insertSuggestion(text string) {
	ed := &u.conv.composer
	rs := []rune(ed.Text())
	ed.SetCaret(len(rs), len(rs))
	if len(rs) > 0 {
		text = " " + text
	}
	ed.Insert(text)
	u.requestFocus(ed)
}
