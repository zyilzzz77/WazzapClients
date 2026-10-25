package ai

import (
	"context"
	"strings"
	"testing"
)

func TestBuildContext(t *testing.T) {
	recent := []string{"hello", "how are you?", "", "see you at 5"}
	msgs := BuildContext("sys", recent, 2)
	// The last two are "" and "see you at 5"; the empty one is dropped.
	if len(msgs) != 2 || msgs[0].Role != "system" || msgs[0].Content != "sys" {
		t.Fatalf("system message: %+v", msgs)
	}
	if msgs[1].Content != "see you at 5" {
		t.Fatalf("truncation kept the wrong messages: %+v", msgs)
	}
	// Empty lines are dropped, long ones truncated.
	long := strings.Repeat("x", 1500)
	msgs = BuildContext("sys", []string{long, "  "}, 20)
	if len(msgs) != 2 || !strings.HasSuffix(msgs[1].Content, "…") || len([]rune(msgs[1].Content)) != 1001 {
		t.Fatalf("long line: len %d", len(msgs))
	}
	// Zero max means the default.
	if msgs := BuildContext("sys", recent, 0); len(msgs) != 4 {
		t.Fatalf("default max: %d messages", len(msgs))
	}
}

func TestAgentAsk(t *testing.T) {
	mock := &MockProvider{Text: "the answer"}
	a := NewAgent(mock, DefaultConfig())
	if a.Enabled() {
		t.Fatal("disabled config reports enabled")
	}
	cfg := DefaultConfig()
	cfg.Enabled = true
	a = NewAgent(mock, cfg)
	ans, err := a.Ask(context.Background(), "what did we say?", "Rina", []string{"hi", "let's meet at 5"})
	if err != nil || ans != "the answer" {
		t.Fatalf("ask: %q, %v", ans, err)
	}
	if mock.Calls != 1 {
		t.Fatalf("calls %d", mock.Calls)
	}
	if _, err := NewAgent(nil, cfg).Ask(context.Background(), "x", "", nil); err == nil {
		t.Fatal("asked without a provider")
	}
}

func TestAgentDraftReplies(t *testing.T) {
	mock := &MockProvider{Text: "1. Sure, see you then\n2. Of course, I'd love to\n3. yup, sounds good"}
	a := NewAgent(mock, DefaultConfig())
	drafts, err := a.DraftReplies(context.Background(), "Rina", []string{"coffee tomorrow?"})
	if err != nil {
		t.Fatal(err)
	}
	want := [3]string{"Sure, see you then", "Of course, I'd love to", "yup, sounds good"}
	if drafts != want {
		t.Fatalf("drafts %+v, want %+v", drafts, want)
	}
	// A plain answer without numbers still fills what it can.
	mock.Text = "Yes\nMaybe later"
	drafts, _ = a.DraftReplies(context.Background(), "Rina", []string{"x"})
	if drafts[0] != "Yes" || drafts[1] != "Maybe later" || drafts[2] != "" {
		t.Fatalf("unnumbered drafts %+v", drafts)
	}
}

func TestAgentSystemPrompt(t *testing.T) {
	a := NewAgent(&MockProvider{}, DefaultConfig())
	if got := a.SystemPrompt("Rina"); !strings.Contains(got, "Rina") || !strings.Contains(got, "Indonesian") {
		t.Fatalf("system prompt: %q", got)
	}
	a.Config.SystemPrompt = "Be brief."
	if got := a.SystemPrompt(""); got != "Be brief." {
		t.Fatalf("custom prompt: %q", got)
	}
	if got := a.Config.EffectiveSystemPrompt(); got != "Be brief." {
		t.Fatalf("effective prompt: %q", got)
	}
}

func TestPromptHashAndLog(t *testing.T) {
	h1, h2 := PromptHash("hello"), PromptHash("hello")
	if h1 != h2 || len(h1) != 16 {
		t.Fatalf("hash %q", h1)
	}
	if PromptHash("other") == h1 {
		t.Fatal("hash collision on different prompts")
	}
	dir := t.TempDir()
	if err := AppendReplyLog(dir, testTime(), h1, "gpt-4o-mini"); err != nil {
		t.Fatal(err)
	}
	if err := AppendReplyLog("", testTime(), h1, "m"); err != nil {
		t.Fatalf("no dir: %v", err)
	}
}
