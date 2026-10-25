package accounts

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAccountRenameMove(t *testing.T) {
	root := t.TempDir()
	l := Load(root)
	l.Current().ID = "1@s.whatsapp.net"
	dir2, _ := l.Add()
	l.Find(dir2).ID, l.Find(dir2).Name = "2@s.whatsapp.net", "Work"
	dir3, _ := l.Add()
	l.Find(dir3).ID, l.Find(dir3).Name = "3@s.whatsapp.net", "Shop"

	if err := l.Rename(dir2, "  Work 2 "); err != nil || l.Find(dir2).Name != "Work 2" {
		t.Errorf("Rename = %v, name %q", err, l.Find(dir2).Name)
	}
	if err := l.Rename("accounts/9", "Nope"); err == nil {
		t.Error("renaming a missing account worked")
	}
	if err := l.Rename(dir3, ""); err != nil || l.Find(dir3).Name != "" {
		t.Errorf("clearing a name: %v, %q", err, l.Find(dir3).Name)
	}
	if err := l.Move(dir3, 0); err != nil || l.Accounts[0].Dir != dir3 {
		t.Errorf("Move to front: %v, %+v", err, l.Accounts)
	}
	if err := l.Move(dir3, 99); err != nil || l.Accounts[len(l.Accounts)-1].Dir != dir3 {
		t.Errorf("Move past the end: %v, %+v", err, l.Accounts)
	}
	if err := l.Move("accounts/9", 0); err == nil {
		t.Error("moving a missing account worked")
	}
	if err := l.Save(); err != nil {
		t.Fatal(err)
	}
	l = Load(root)
	if l.Find(dir2).Name != "Work 2" || l.Accounts[len(l.Accounts)-1].Dir != dir3 {
		t.Errorf("reloaded: %+v", l.Accounts)
	}
}

func TestAlwaysBudget(t *testing.T) {
	if MaxAlways != 2 {
		t.Fatalf("MaxAlways = %d, want 2", MaxAlways)
	}
	root := t.TempDir()
	l := Load(root)
	if l.AlwaysLimit() != MaxAlways {
		t.Errorf("default limit = %d", l.AlwaysLimit())
	}
	l.Current().ID = "1@s.whatsapp.net"
	l.Current().Background = Always
	dir2, _ := l.Add()
	l.Find(dir2).ID, l.Find(dir2).Background = "2@s.whatsapp.net", Always
	dir3, _ := l.Add()
	l.Find(dir3).ID = "3@s.whatsapp.net"
	if l.AlwaysCount() != 2 || l.CanSetAlways(dir3) {
		t.Errorf("count %d, third can join %v", l.AlwaysCount(), l.CanSetAlways(dir3))
	}
	if l.CanSetAlways(dir2) != true {
		t.Error("an account that is already Always may stay Always")
	}
	l.SetAlwaysLimit(3)
	if !l.CanSetAlways(dir3) {
		t.Error("a limit of 3 should admit a third Always account")
	}
	l.SetAlwaysLimit(0)
	if l.AlwaysLimit() != MaxAlways {
		t.Errorf("a limit of 0 fell back to %d", l.AlwaysLimit())
	}
	if err := l.Save(); err != nil {
		t.Fatal(err)
	}
	if Load(root).AlwaysLimit() != MaxAlways {
		t.Error("the limit didn't survive a reload")
	}
}

func TestAccountSettings(t *testing.T) {
	root := t.TempDir()
	l := Load(root)
	l.Current().ID = "1@s.whatsapp.net"
	dir, _ := l.Add()
	l.Find(dir).ID = "2@s.whatsapp.net"
	if !l.NotificationsOn(dir) || !l.NotificationsOn("accounts/9") {
		t.Error("notifications start off")
	}
	if err := l.SetNotifications(dir, false); err != nil || l.NotificationsOn(dir) {
		t.Errorf("turning notifications off: %v", err)
	}
	if err := l.SetAI(dir, true); err != nil || !l.Find(dir).AIEnabled {
		t.Errorf("AI switch: %v", err)
	}
	if err := l.SetNotifications("accounts/9", false); err == nil {
		t.Error("setting a missing account worked")
	}
	if err := l.Save(); err != nil {
		t.Fatal(err)
	}
	l = Load(root)
	if l.NotificationsOn(dir) || !l.Find(dir).AIEnabled {
		t.Errorf("settings didn't survive a reload: %+v", l.Find(dir))
	}
}

func TestRemoveIsolation(t *testing.T) {
	root := t.TempDir()
	l := Load(root)
	l.Current().ID = "1@s.whatsapp.net"
	dir2, _ := l.Add()
	l.Find(dir2).ID = "2@s.whatsapp.net"
	dir3, _ := l.Add()
	l.Find(dir3).ID = "3@s.whatsapp.net"
	for _, d := range []string{"", dir2, dir3} {
		p := filepath.Join(l.Path(d), "wazzap.db")
		if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("db of "+d), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := l.Save(); err != nil {
		t.Fatal(err)
	}
	if err := l.Remove(dir2); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(l.Path(dir2)); !os.IsNotExist(err) {
		t.Errorf("removed account's dir kept: %v", err)
	}
	for _, d := range []string{"", dir3} {
		data, err := os.ReadFile(filepath.Join(l.Path(d), "wazzap.db"))
		if err != nil || string(data) != "db of "+d {
			t.Errorf("account %q touched: %q, %v", d, data, err)
		}
	}
	if l.Find(dir2) != nil || len(l.Accounts) != 2 {
		t.Errorf("list after remove: %+v", l.Accounts)
	}
}
