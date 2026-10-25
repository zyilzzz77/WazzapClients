package ui

import "github.com/chomosuke9/wazzapclients/internal/model"

// Saving view once media, only with the opt-in on (Settings > Privacy,
// off by default). The viewer offers a Save button, which asks again
// before writing, and the save itself goes to the account's media
// folder. It never runs on its own, and never uploads anything.

// viewOnceSaveShown reports whether the Save button shows for a view
// once message: only with the opt-in on, and only for media this device
// actually has (an OnPhone message has none to save).
func (u *UI) viewOnceSaveShown(m *model.Message) bool {
	return u.viewOnceSave && !m.OnPhone && m.Media != model.MediaNone
}

// saveViewOnce asks once more, then saves the shown view once media.
func (u *UI) saveViewOnce(m *model.Message) {
	what := "photo"
	if m.Media == model.MediaVideo {
		what = "video"
	}
	u.confirm("Save this view-once "+what+"?",
		"The sender expected this "+what+" to disappear after you viewed it. Saving keeps a copy in "+
			"this account's media folder.",
		dialogButton{label: "Save", primary: true, run: func() { u.backend.SaveViewOnce(m) }})
}
