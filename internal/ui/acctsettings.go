// Per-account settings on an account's settings page: its
// notifications, its AI switch (Phase 3 placeholder — stored, nothing
// runs on it yet), and the always-connected budget.
package ui

// setAccountNotify turns dir's notifications on or off.
func (h *host) setAccountNotify(dir string, on bool) {
	l := h.o.Accounts
	if l == nil {
		return
	}
	if err := l.SetNotifications(dir, on); err != nil {
		return
	}
	h.saveAccounts()
	h.applyAcctNotify()
	h.refreshAccounts()
}

// setAccountAI stores dir's AI switch (Phase 3 placeholder).
func (h *host) setAccountAI(dir string, on bool) {
	l := h.o.Accounts
	if l == nil {
		return
	}
	if err := l.SetAI(dir, on); err != nil {
		return
	}
	h.saveAccounts()
	h.refreshAccounts()
}

// renameAccount renames dir, and moveAccount reorders it to position
// to (0 is first). Both save and refresh the switcher.
func (h *host) renameAccount(dir, name string) error {
	l := h.o.Accounts
	if l == nil {
		return nil
	}
	if err := l.Rename(dir, name); err != nil {
		return err
	}
	h.saveAccounts()
	h.refreshAccounts()
	return nil
}

func (h *host) moveAccount(dir string, to int) error {
	l := h.o.Accounts
	if l == nil {
		return nil
	}
	if err := l.Move(dir, to); err != nil {
		return err
	}
	h.saveAccounts()
	h.refreshAccounts()
	return nil
}

// acctPrefSections are the per-account settings rows of the account
// with dir: notifications, AI (placeholder), and the Always budget.
func (u *UI) acctPrefSections(dir string) []settingsSection {
	a := u.accountByDir(dir)
	if a == nil {
		return nil
	}
	toggle := func(key, title, sub string, on bool, set func(bool)) settingRow {
		return settingRow{key: key, kind: setToggle, title: title, sub: sub, on: on,
			run: func() { set(!on); u.settings.stale = true }}
	}
	notify := toggle("acctnotify:"+dir, "Notifications",
		"Show notifications from this account while another one is open",
		!a.NotifyOff, func(on bool) {
			if u.host != nil {
				u.host.setAccountNotify(dir, on)
				return
			}
			a.NotifyOff = !on // demo data
		})
	ai := toggle("acctai:"+dir, "AI features",
		"Off until Phase 3: this switch is stored per account, nothing runs on it yet",
		a.AIEnabled, func(on bool) {
			if u.host != nil {
				u.host.setAccountAI(dir, on)
				return
			}
			a.AIEnabled = on // demo data
		})
	secs := []settingsSection{{title: "This account", rows: []settingRow{notify, ai}}}
	if u.host != nil && u.host.o.Accounts != nil {
		limit := u.host.alwaysLimit()
		rows := []settingRow{}
		for _, n := range []int{1, 2, 3} {
			n := n
			rows = append(rows, settingRow{key: "alwayslimit:" + itoa(n), kind: setRadio,
				title: itoa(n) + " account" + pluralS(n), on: limit == n,
				run: func() { u.host.setAlwaysLimit(n); u.settings.stale = true }})
		}
		secs = append(secs, settingsSection{title: "Always connected, at most",
			note: "How many accounts may stay always connected at once. " +
				"Accounts over the budget check every few minutes instead.",
			rows: rows})
	}
	return secs
}

func pluralS(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}
