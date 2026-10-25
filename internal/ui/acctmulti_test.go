package ui

import (
	"testing"

	"github.com/chomosuke9/wazzapclients/internal/accounts"
)

// TestMultiAccountUnread: the tray tooltip totals every account's
// unread chats, the counts persist across a reload, and the switcher
// rows carry them.
func TestMultiAccountUnread(t *testing.T) {
	bt := newBgTest(t, accounts.Always)
	bt.tick()
	h := bt.h
	if h.unread == nil {
		t.Fatal("the host has no unread store")
	}
	work := h.unreadFor(workDir)
	if work == 0 {
		t.Fatalf("Work's unread count is 0; its notifier counts %+v", h.bg[workDir].notes)
	}
	if got := h.unread.Get(workDir); got != work {
		t.Errorf("persisted count %d, live count %d", got, work)
	}
	reloaded := accounts.LoadUnread(bt.l.Root())
	if reloaded.Get(workDir) != work || reloaded.Total() != h.unread.Total() {
		t.Errorf("reloaded %+v, want %+v", reloaded.Snapshot(), h.unread.Snapshot())
	}
	var tip string
	old := setTooltip
	setTooltip = func(s string) { tip = s }
	t.Cleanup(func() { setTooltip = old })
	h.updateTooltip()
	if tip != tooltipText(h.unread.Total()) {
		t.Errorf("tooltip %q, want %q", tip, tooltipText(h.unread.Total()))
	}
	rows := h.accountRows()
	for _, r := range rows {
		if r.Dir == workDir && r.unread != work {
			t.Errorf("Work's switcher row carries %d unread, want %d", r.unread, work)
		}
	}
}

// TestAccountNotifyOff: silencing one account stops its notifications
// without touching the open account's.
func TestAccountNotifyOff(t *testing.T) {
	bt := newBgTest(t, accounts.Always)
	h := bt.h
	bt.tick()
	bt.flush() // clear anything pending
	bt.h.setAccountNotify(workDir, false)
	if bt.l.NotificationsOn(workDir) {
		t.Fatal("the switch didn't stick")
	}
	bt.work.Receive("rina", "Rina", "Are you coming?")
	h.settle()
	if got := bt.flush(); len(got) != 0 {
		t.Errorf("a silenced account notified: %+v", got)
	}
	bt.h.setAccountNotify(workDir, true)
	bt.work.Receive("rina", "Rina", "Are you coming?")
	h.settle()
	if got := bt.flush(); len(got) != 1 {
		t.Errorf("after turning notifications back on: %+v", got)
	}
}

// TestAlwaysBudgetRefused: a third account can't be set to Always
// while two already are, and raising the limit admits it.
func TestAlwaysBudgetRefused(t *testing.T) {
	h := newAccountsHost(t)
	l := h.o.Accounts
	h.setBackgroundMode("", accounts.Always)
	h.addAccount() // accounts/2, now open
	h.settle()
	h.setBackgroundMode("accounts/2", accounts.Always)
	h.switchAccount("")
	h.settle()
	if l.AlwaysCount() != 2 {
		t.Fatalf("Always count %d, want 2", l.AlwaysCount())
	}
	h.addAccount() // accounts/3, now open
	h.settle()
	h.setBackgroundMode("accounts/3", accounts.Always)
	if l.Find("accounts/3").Background != accounts.Off {
		t.Errorf("a third Always account was admitted: %+v", l.Find("accounts/3"))
	}
	if l.AlwaysCount() != 2 {
		t.Errorf("Always count %d after the refusal", l.AlwaysCount())
	}
	h.setAlwaysLimit(3)
	h.setBackgroundMode("accounts/3", accounts.Always)
	if l.Find("accounts/3").Background != accounts.Always {
		t.Error("raising the limit didn't admit the third account")
	}
}

// TestSwitchKeepsBackgroundUnread: switching away from an Always
// account keeps it connected, and its unread count stays visible.
func TestSwitchKeepsBackgroundUnread(t *testing.T) {
	bt := newBgTest(t, accounts.Always)
	h := bt.h
	bt.tick()
	h.setBackgroundMode("", accounts.Always) // the open account, too
	// Run feeds the open notifier its chats at startup; the harness
	// doesn't, so do it here before switching away.
	h.notes.setChats(bt.me.Chats())
	h.switchAccount(workDir)
	h.settle()
	if bt.l.Active != workDir {
		t.Fatalf("active %q", bt.l.Active)
	}
	if bg := h.bg[""]; bg == nil || bg.b == nil {
		t.Fatalf("the first account didn't stay connected in the background: %+v", h.bg[""])
	}
	if h.unreadFor("") == 0 {
		t.Error("the background account's unread count vanished on switch")
	}
}
