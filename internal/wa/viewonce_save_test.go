package wa

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/chomosuke9/wazzapclients/internal/model"
)

// Saving view once media is gated on the opt-in, and lands in the
// account's media folder.
func TestSaveViewOnce(t *testing.T) {
	b := testBackend(t)
	m := &model.Message{ID: "VO1", ChatID: "555@s.whatsapp.net", Media: model.MediaImage, Time: time.Unix(1_700_000_000, 0)}
	want := filepath.Join(b.dataDir, "media", "viewonce")

	// Off by default: nothing is written, only a notice.
	b.SaveViewOnce(m)
	if got := pollNotice(t, b, "Saving view once media is off"); got == "" {
		t.Fatal("no notice that saving is off")
	}
	if _, err := os.Stat(want); !os.IsNotExist(err) {
		t.Errorf("view once media was saved with the opt-in off")
	}

	// On, with the media already cached locally: it is written, 0600.
	b.SetPref(model.PrefViewOnceSave, "on")
	if err := os.MkdirAll(filepath.Dir(b.mediaPath(m.ChatID, m.ID)), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(b.mediaPath(m.ChatID, m.ID), []byte("secret photo"), 0o600); err != nil {
		t.Fatal(err)
	}
	path, err := b.saveViewOnce(m)
	if err != nil {
		t.Fatalf("saving: %v", err)
	}
	if !strings.HasPrefix(path, want) {
		t.Errorf("saved to %q, want under %q", path, want)
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "secret photo" {
		t.Errorf("saved %q (%v), want the photo", data, err)
	}
	if st, err := os.Stat(path); err != nil || st.Mode().Perm() != 0o600 {
		t.Errorf("saved file mode %v, want 0600", st.Mode())
	}
	// Saving again doesn't overwrite: it numbers the file.
	path2, err := b.saveViewOnce(m)
	if err != nil || path2 == path {
		t.Errorf("second save gave %q (%v), want a new file", path2, err)
	}
}

// pollNotice waits up to 2s for a NoticeEvent whose text starts with
// want, and returns it.
func pollNotice(t *testing.T, b *Backend, want string) string {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		for _, e := range b.Poll() {
			if n, ok := e.(model.NoticeEvent); ok && strings.HasPrefix(n.Text, want) {
				return n.Text
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	return ""
}
