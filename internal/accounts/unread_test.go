package accounts

import (
	"os"
	"path/filepath"
	"testing"
)

func TestUnreadCounts(t *testing.T) {
	root := t.TempDir()
	c := LoadUnread(root)
	if c.Total() != 0 || c.Get("accounts/2") != 0 {
		t.Errorf("fresh counts: %+v", c.Snapshot())
	}
	if c.Set("accounts/2", 3) != true || c.Set("accounts/2", 3) != false {
		t.Error("Set didn't report the change")
	}
	c.Set("", 1)
	if c.Total() != 4 {
		t.Errorf("total = %d, want 4", c.Total())
	}
	if err := c.Save(root); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "unread.json")); err != nil {
		t.Fatalf("no unread.json: %v", err)
	}
	c = LoadUnread(root)
	if c.Get("accounts/2") != 3 || c.Get("") != 1 {
		t.Errorf("reloaded: %+v", c.Snapshot())
	}
	if !c.Remove("accounts/2") || c.Remove("accounts/2") {
		t.Error("Remove didn't report right")
	}
	if c.Prune([]string{""}) || c.Prune([]string{"accounts/3"}) != true {
		t.Errorf("Prune: %+v", c.Snapshot())
	}
	if c.Total() != 0 {
		t.Errorf("total after prune = %d", c.Total())
	}
}
