package ui

import (
	"testing"

	"github.com/chomosuke9/wazzapclients/internal/mock"
	"github.com/chomosuke9/wazzapclients/internal/model"
)

// The Save button shows only with the opt-in on, and only for media this
// device has.
func TestViewOnceSaveShown(t *testing.T) {
	b := mock.New()
	u := New(b)
	m := &model.Message{Media: model.MediaImage}
	if u.viewOnceSaveShown(m) {
		t.Error("Save shows with the opt-in off")
	}
	b.SetPref(model.PrefViewOnceSave, "on")
	u.loadExtras()
	if !u.viewOnceSaveShown(m) {
		t.Error("Save doesn't show with the opt-in on")
	}
	m.OnPhone = true
	if u.viewOnceSaveShown(m) {
		t.Error("Save shows for a message only the phone has")
	}
	m.OnPhone, m.Media = false, model.MediaNone
	if u.viewOnceSaveShown(m) {
		t.Error("Save shows for a message with no media")
	}
	m.Media = model.MediaVideo
	if !u.viewOnceSaveShown(m) {
		t.Error("Save doesn't show for a view once video")
	}
	// Turning the setting off hides it again, live.
	b.SetPref(model.PrefViewOnceSave, "off")
	u.loadExtras()
	if u.viewOnceSaveShown(m) {
		t.Error("Save still shows after the opt-in was turned off")
	}
}
