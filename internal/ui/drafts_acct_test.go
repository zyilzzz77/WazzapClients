package ui

import (
	"slices"
	"testing"

	"github.com/chomosuke9/wazzapclients/internal/mock"
	"github.com/chomosuke9/wazzapclients/internal/model"
)

// TestDraftsPerAccount: a draft typed in one account waits for that
// account while another is open, and comes back on switching back —
// the other account's draft is kept apart from it.
func TestDraftsPerAccount(t *testing.T) {
	h := newAccountsHost(t)
	l := h.o.Accounts
	h.drafts["rina"] = &chatDraft{text: "for the first account"}

	h.addAccount()
	h.settle()
	if l.Active != "accounts/2" {
		t.Fatalf("active %q", l.Active)
	}
	if h.drafts["rina"] != nil {
		t.Fatalf("the first account's draft followed: %+v", h.drafts["rina"])
	}
	if d := h.draftsByAcct[""]; d == nil || d["rina"] == nil || d["rina"].text != "for the first account" {
		t.Fatalf("shelved drafts: %+v", h.draftsByAcct[""])
	}
	h.drafts["rina"] = &chatDraft{text: "for the second account"}

	h.switchAccount("")
	h.settle()
	if l.Active != "" {
		t.Fatalf("active %q", l.Active)
	}
	if d := h.drafts["rina"]; d == nil || d.text != "for the first account" {
		t.Fatalf("the first account's draft didn't come back: %+v", d)
	}
	h.switchAccount("accounts/2")
	h.settle()
	if d := h.drafts["rina"]; d == nil || d.text != "for the second account" {
		t.Fatalf("the second account's draft didn't come back: %+v", d)
	}
}

// TestScrollPerAccount: a chat reopened in the same account returns
// to the message it was left at, not to the newest.
func TestScrollPerAccount(t *testing.T) {
	u := New(mock.New())
	u.Start(func() {})
	u.host = newHost(mock.New(), Options{})
	u.host.scrollByAcct = map[string]map[string]string{}
	u.SelectID("rina")
	if len(u.msgs) < 3 {
		t.Fatalf("rina has %d messages", len(u.msgs))
	}
	mid := u.msgs[len(u.msgs)/2].ID
	u.host.scrollByAcct[""] = map[string]string{"rina": mid}

	u.SelectID("budi")
	u.SelectID("rina")
	if !slices.ContainsFunc(u.msgs, func(m *model.Message) bool { return m.ID == mid }) {
		t.Fatalf("the anchor message %s isn't loaded: %+v", mid, u.msgs)
	}
	// loadLatest leaves scrollTo at the zero position (the newest);
	// loadAround puts it at the anchor's row.
	if u.conv.scrollTo == nil || u.conv.scrollTo.First == 0 {
		t.Errorf("the chat came back at the newest message instead of the anchor: %+v", u.conv.scrollTo)
	}
}
