package ui

import (
	"testing"

	"github.com/chomosuke9/wazzapclients/internal/ai"
	"github.com/chomosuke9/wazzapclients/internal/mock"
)

// TestAILoadDisabled checks that AI starts off with no config: no
// agent, no network, and the settings page still builds.
func TestAILoadDisabled(t *testing.T) {
	st := newSlashTest(t, "rina")
	if st.u.aiReady() {
		t.Fatal("AI is ready with no config")
	}
	if st.u.ai.agent != nil {
		t.Fatal("an agent was built with no config")
	}
	rows := st.u.aiSettings()
	if len(rows) != 1 || len(rows[0].rows) != 1 || rows[0].rows[0].key != "ai:enable" {
		t.Fatalf("AI page without config: %+v", rows)
	}
	// /ai is offered by the slash picker, like any other command.
	st.typeText("/ai ")
	if sp := st.u.slashQuery(); sp == nil {
		t.Fatal("/ai not offered")
	}
}

// TestAILoadEnabled saves a local-preset config (no key needed) and
// checks the agent comes up, and the settings page shows the rest.
func TestAILoadEnabled(t *testing.T) {
	b := mock.New()
	dir := t.TempDir()
	b.SetDataDir(dir)
	u := New(b)
	u.Start(func() {})
	cfg := ai.DefaultConfig()
	cfg.Enabled = true
	cfg.Provider = ai.PresetOllama
	cfg.BaseURL = "http://localhost:11434/v1"
	cfg.Model = "llama3.1"
	if err := cfg.Save(dir); err != nil {
		t.Fatal(err)
	}
	u.loadAI()
	if !u.aiReady() {
		t.Fatal("AI not ready with an enabled local config")
	}
	rows := u.aiSettings()
	if len(rows) != 1 || len(rows[0].rows) < 10 {
		t.Fatalf("AI page with config has %d rows", len(rows[0].rows))
	}
	// Turning it off drops the agent again.
	u.aiToggle(false)
	if u.aiReady() || u.ai.agent != nil {
		t.Fatal("AI stayed ready after turning it off")
	}
}

// TestAISuggestOff checks that Suggest replies does nothing while AI
// is off: no drafts, no panic.
func TestAISuggestOff(t *testing.T) {
	st := newSlashTest(t, "rina")
	st.u.requestAISuggest()
	st.u.ai.suggest.mu.Lock()
	defer st.u.ai.suggest.mu.Unlock()
	if st.u.ai.suggest.busy || len(st.u.ai.suggest.items) != 0 {
		t.Fatal("suggestions appeared with AI off")
	}
}
