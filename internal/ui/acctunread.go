// Per-account unread counts and notification switches.
//
// The notifiers already know each account's unread chats (their chats'
// Unread, for the tooltip). This file syncs those numbers into the
// persisted accounts.Counts, so the switcher's badges and the tray
// tooltip show every account — including one whose backend is closed
// between checks — and survive a restart.
package ui

import "log"

// unreadChats is a notifier's unread-chat count, for the tooltip and
// the switcher's badge. -1 (not loaded yet) counts as none.
func (n *notifier) unreadChats() int { return max(n.unread, 0) }

// unreadFor is dir's unread-chat count: the live notifier's when it
// has one, else the last persisted count.
func (h *host) unreadFor(dir string) int {
	if h.o.Accounts != nil && dir == h.o.Accounts.Active && h.notes != nil {
		return h.notes.unreadChats()
	}
	if bg := h.bg[dir]; bg != nil && bg.notes != nil {
		return bg.notes.unreadChats()
	}
	if h.unread != nil {
		return h.unread.Get(dir)
	}
	return 0
}

// syncUnreadCounts copies every account's live unread count into the
// persisted store, drops accounts that left the list, and saves when
// anything changed. It also refreshes the switcher, whose rows carry
// the counts.
func (h *host) syncUnreadCounts() {
	l := h.o.Accounts
	if l == nil || h.unread == nil {
		return
	}
	changed := false
	dirs := make([]string, 0, len(l.Accounts))
	for _, a := range l.Accounts {
		dirs = append(dirs, a.Dir)
		if h.unread.Set(a.Dir, h.unreadFor(a.Dir)) {
			changed = true
		}
	}
	if h.unread.Prune(dirs) {
		changed = true
	}
	if changed {
		if err := h.unread.Save(l.Root()); err != nil {
			log.Printf("save unread counts: %v", err)
		}
		h.refreshAccounts()
	}
}

// acctNotifyOff reports whether dir's notifications are silenced in
// accounts.json.
func (h *host) acctNotifyOff(dir string) bool {
	if l := h.o.Accounts; l != nil {
		return !l.NotificationsOn(dir)
	}
	return false
}

// applyAcctNotify applies the per-account notification switches to the
// notifiers that exist now (open and background).
func (h *host) applyAcctNotify() {
	l := h.o.Accounts
	if l == nil {
		return
	}
	if h.notes != nil {
		h.notes.notifyOff = !l.NotificationsOn(l.Active)
	}
	for dir, bg := range h.bg {
		if bg.notes != nil {
			bg.notes.notifyOff = !l.NotificationsOn(dir)
		}
	}
}
