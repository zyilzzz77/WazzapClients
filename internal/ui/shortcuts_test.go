package ui

import (
	"testing"

	"gioui.org/io/key"
)

// Ctrl+N opens New chat, Ctrl+K goes to the chat list, Ctrl+Shift+M
// mutes the open chat, and Ctrl+1..9 with no accounts does nothing.
func TestShortcuts(t *testing.T) {
	st := newSlashTest(t, "rina")
	if st.u.selected == nil || st.u.selected.Muted {
		t.Fatal("the test chat isn't open and unmuted")
	}
	press := func(name string, mods key.Modifiers) {
		st.r.Queue(key.Event{Name: key.Name(name), State: key.Press, Modifiers: mods})
		st.frame()
	}

	press("N", key.ModShortcut)
	if !st.u.newChat.open() {
		t.Error("Ctrl+N didn't open New chat")
	}
	st.u.newChatBack()
	st.frame()

	st.u.setPage(pageStatus)
	st.frame()
	press("K", key.ModShortcut)
	if st.u.page != pageChats {
		t.Error("Ctrl+K didn't go to the chat list")
	}

	press("M", key.ModShortcut|key.ModShift)
	st.u.applyEvents()
	if !st.u.selected.Muted {
		t.Error("Ctrl+Shift+M didn't mute the open chat")
	}
	press("M", key.ModShortcut|key.ModShift)
	st.u.applyEvents()
	if st.u.selected.Muted {
		t.Error("Ctrl+Shift+M didn't unmute the open chat")
	}

	// No accounts in the test harness: nothing to switch to, no crash.
	press("1", key.ModShortcut)
	press("9", key.ModShortcut)
}

// Ctrl+Shift+M while the composer is focused is monospace, not mute:
// the composer's formatting shortcut reads the key first.
func TestShortcutMuteYieldsToComposer(t *testing.T) {
	st := newSlashTest(t, "rina")
	st.u.requestFocus(&st.u.conv.composer)
	st.frame() // the composer takes focus
	st.frame()
	st.u.conv.composer.SetText("hi")
	st.r.Queue(key.Event{Name: key.Name("M"), State: key.Press, Modifiers: key.ModShortcut | key.ModShift})
	st.frame()
	st.u.applyEvents()
	if st.u.selected.Muted {
		t.Error("Ctrl+Shift+M muted the chat while typing")
	}
}
