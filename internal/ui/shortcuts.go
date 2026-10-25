package ui

import (
	"strconv"

	"gioui.org/io/key"
)

// Keyboard shortcuts (see Settings > Keyboard shortcuts): Ctrl+K
// focuses the chat list's search, Ctrl+N opens New chat, Ctrl+1..9
// switches account, Ctrl+Shift+M mutes the open chat. Esc already closes
// the topmost overlay (update's escape) and clears the search.
// This runs after layout, in layoutWindow, so the composer's own
// shortcuts read their keys first — Ctrl+Shift+M is monospace while
// typing, and only mutes when the composer isn't focused.

// shortcutKeys handles the window's shortcuts.
func (u *UI) shortcutKeys(gtx C) {
	if u.dialog.isOpen() || u.viewer.open || u.picker.open || u.gallery.open ||
		u.status.viewer.isOpen() || u.status.text.isOpen() {
		return
	}
	for {
		ev, ok := gtx.Event(
			key.Filter{Name: "K", Required: key.ModShortcut},
			key.Filter{Name: "N", Required: key.ModShortcut},
			key.Filter{Name: "M", Required: key.ModShortcut | key.ModShift},
			key.Filter{Name: "1", Required: key.ModShortcut},
			key.Filter{Name: "2", Required: key.ModShortcut},
			key.Filter{Name: "3", Required: key.ModShortcut},
			key.Filter{Name: "4", Required: key.ModShortcut},
			key.Filter{Name: "5", Required: key.ModShortcut},
			key.Filter{Name: "6", Required: key.ModShortcut},
			key.Filter{Name: "7", Required: key.ModShortcut},
			key.Filter{Name: "8", Required: key.ModShortcut},
			key.Filter{Name: "9", Required: key.ModShortcut},
		)
		if !ok {
			break
		}
		e, ok := ev.(key.Event)
		if !ok || e.State != key.Press {
			continue
		}
		switch e.Name {
		case "K":
			u.focusSearch()
		case "N":
			u.openNewChat()
		case "M":
			u.toggleMuteCurrent()
		default:
			u.switchAccountKey(string(e.Name))
		}
	}
}

// focusSearch opens the chat list and focuses its search.
func (u *UI) focusSearch() {
	u.setPage(pageChats)
	u.setListHidden(false)
	u.requestFocus(&u.sidebar.search)
}

// switchAccountKey switches to the nth account (Ctrl+1..9), if there is
// one and it isn't open.
func (u *UI) switchAccountKey(name string) {
	n, err := strconv.Atoi(name)
	if err != nil || n < 1 || n > len(u.accounts) || u.host == nil {
		return
	}
	if a := u.accounts[n-1]; !a.active {
		u.host.request(request{kind: reqSwitch, dir: a.Dir})
	}
}

// toggleMuteCurrent mutes the open chat for good, or unmutes it.
func (u *UI) toggleMuteCurrent() {
	c := u.selected
	if c == nil {
		return
	}
	u.backend.SetMuted(c.ID, !c.Muted, 0)
}
