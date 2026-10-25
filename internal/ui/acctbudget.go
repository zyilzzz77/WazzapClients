// The memory budget for always-connected accounts: at most
// accounts.MaxAlways of them (configurable), so background accounts
// can't grow the idle footprint without bound.
package ui

import "github.com/chomosuke9/wazzapclients/internal/accounts"

// alwaysBudgetOK reports whether dir may run Always without passing
// the cap. dir's own Always doesn't count against it.
func (h *host) alwaysBudgetOK(dir string) bool {
	if l := h.o.Accounts; l != nil {
		return l.CanSetAlways(dir)
	}
	return true
}

// alwaysLimit is the cap on Always accounts, for the refusal toast.
func (h *host) alwaysLimit() int {
	if l := h.o.Accounts; l != nil {
		return l.AlwaysLimit()
	}
	return accounts.MaxAlways
}

// setAlwaysLimit changes the cap on Always accounts. Accounts already
// Always keep running until they are switched or their mode changes;
// the cap bites on the next mode change and at startup.
func (h *host) setAlwaysLimit(n int) {
	l := h.o.Accounts
	if l == nil {
		return
	}
	l.SetAlwaysLimit(n)
	h.saveAccounts()
	h.refreshAccounts()
}
