// Manage accounts: rename, reorder, per-account settings, and the
// memory budget for accounts that stay always connected.
package accounts

import (
	"errors"
	"slices"
	"strings"
)

// MaxAlways is the default cap on accounts that stay Always connected
// at once: each idle connected account costs a few MB, and the cap
// keeps the background budget bounded. A List's AlwaysLimit overrides
// it (see AlwaysLimit).
const MaxAlways = 2

// Root returns the data directory the list lives in.
func (l *List) Root() string { return l.root }

// AlwaysLimit returns the cap on Always accounts: the list's own, or
// MaxAlways when it has none (or a nonsense one).
func (l *List) AlwaysLimit() int {
	if l == nil || l.AlwaysCap <= 0 {
		return MaxAlways
	}
	return l.AlwaysCap
}

// SetAlwaysLimit sets the cap on Always accounts; 0 or less resets it
// to the default (MaxAlways).
func (l *List) SetAlwaysLimit(n int) {
	if n <= 0 {
		n = 0
	}
	l.AlwaysCap = n
}

// AlwaysCount counts the linked accounts set to Always.
func (l *List) AlwaysCount() int {
	n := 0
	for _, a := range l.Accounts {
		if a.Linked() && a.Background == Always {
			n++
		}
	}
	return n
}

// CanSetAlways reports whether dir may become Always without passing
// the cap. dir itself doesn't count: switching it on again is fine.
func (l *List) CanSetAlways(dir string) bool {
	n := 0
	for _, a := range l.Accounts {
		if a.Dir != dir && a.Linked() && a.Background == Always {
			n++
		}
	}
	return n < l.AlwaysLimit()
}

// Rename changes an account's display name. An empty name clears it,
// so the account shows its phone number again.
func (l *List) Rename(dir, name string) error {
	a := l.Find(dir)
	if a == nil {
		return errors.New("accounts: no such account")
	}
	a.Name = strings.TrimSpace(name)
	return nil
}

// Move reorders dir to position to (0 is first), clamping to the list.
func (l *List) Move(dir string, to int) error {
	i := slices.IndexFunc(l.Accounts, func(a Account) bool { return a.Dir == dir })
	if i < 0 {
		return errors.New("accounts: no such account")
	}
	a := l.Accounts[i]
	l.Accounts = slices.Delete(l.Accounts, i, i+1)
	to = max(0, min(to, len(l.Accounts)))
	l.Accounts = slices.Insert(l.Accounts, to, a)
	return nil
}

// NotificationsOn reports whether dir's notifications are on. The
// global notification prefs are the default; NotifyOff silences one
// account without touching the others.
func (l *List) NotificationsOn(dir string) bool {
	if a := l.Find(dir); a != nil {
		return !a.NotifyOff
	}
	return true
}

// SetNotifications turns dir's notifications on or off.
func (l *List) SetNotifications(dir string, on bool) error {
	a := l.Find(dir)
	if a == nil {
		return errors.New("accounts: no such account")
	}
	a.NotifyOff = !on
	return nil
}

// SetAI stores dir's AI switch (Phase 3 placeholder: no AI runs yet).
func (l *List) SetAI(dir string, on bool) error {
	a := l.Find(dir)
	if a == nil {
		return errors.New("accounts: no such account")
	}
	a.AIEnabled = on
	return nil
}
