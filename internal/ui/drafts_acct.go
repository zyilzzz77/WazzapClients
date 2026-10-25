// Per-account drafts and scroll positions.
//
// The UI keeps one chat-keyed draft map (host.drafts, draft.go); on a
// switch it used to be thrown away (clearDrafts). Instead the host now
// shelves each account's drafts under its Dir and hands them back when
// the account opens again, so a half-written message waits for its own
// account. Scroll positions are the same idea, cheaper: only the first
// visible message's ID per chat, restored with loadAround when the chat
// opens again (and only when the user had scrolled up from the newest).
package ui

// stashAccountDrafts shelves the open account's drafts under dir and
// leaves a fresh map for the next account. The drafts' temp files move
// with them; nothing is deleted.
func (h *host) stashAccountDrafts(dir string) {
	if h.draftsByAcct == nil {
		h.draftsByAcct = map[string]map[string]*chatDraft{}
	}
	h.draftsByAcct[dir] = h.drafts
	h.drafts = make(map[string]*chatDraft)
}

// restoreAccountDrafts hands dir's shelved drafts back, or a fresh map
// when it has none.
func (h *host) restoreAccountDrafts(dir string) {
	if m, ok := h.draftsByAcct[dir]; ok {
		h.drafts = m
		delete(h.draftsByAcct, dir)
		return
	}
	h.drafts = make(map[string]*chatDraft)
}

// dropAccountDrafts forgets dir's shelved drafts (it left the list),
// temp files included.
func (h *host) dropAccountDrafts(dir string) {
	if m := h.draftsByAcct[dir]; m != nil {
		clearDrafts(m)
		delete(h.draftsByAcct, dir)
	}
}

// currentDir is the open account's Dir, or "" without accounts.
func (h *host) currentDir() string {
	if h.o.Accounts == nil {
		return ""
	}
	return h.o.Accounts.Active
}

// stashScroll remembers the open chat's scroll position for the open
// account: the first visible message's ID. Nothing is recorded before
// the list has laid out (Position.Count is 0 in tests and headless
// windows), and being at the newest message clears the record, since
// opening the chat scrolls there anyway.
func (u *UI) stashScroll() {
	h := u.host
	if h == nil || u.selected == nil || u.conv.list.Position.Count == 0 {
		return
	}
	if h.scrollByAcct == nil {
		h.scrollByAcct = map[string]map[string]string{}
	}
	dir := h.currentDir()
	m := h.scrollByAcct[dir]
	if m == nil {
		m = map[string]string{}
		h.scrollByAcct[dir] = m
	}
	m[u.selected.ID] = u.scrollAnchorID()
}

// scrollAnchorID is the first visible message's ID, or "" at the
// newest message (or when no message row is visible).
func (u *UI) scrollAnchorID() string {
	if u.conv.list.Position.BeforeEnd {
		return ""
	}
	rows := u.rows(u.selected)
	i := u.conv.list.Position.First
	if i < 0 || i >= len(rows) {
		return ""
	}
	for _, r := range rows[i:] {
		if r.msg != nil {
			return r.msg.ID
		}
	}
	return ""
}

// restoreScroll returns the open chat to where it was left in this
// account, silently: no flash, no toast when the message is gone.
func (u *UI) restoreScroll() {
	h := u.host
	if h == nil || u.selected == nil {
		return
	}
	id := h.scrollByAcct[h.currentDir()][u.selected.ID]
	if id == "" {
		return
	}
	if len(u.msgs) > 0 && u.msgs[len(u.msgs)-1].ID == id {
		return // already at the newest message
	}
	u.loadAround(id)
}
