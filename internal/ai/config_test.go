package ai

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPresets(t *testing.T) {
	seen := map[ProviderPreset]bool{}
	for _, p := range Presets() {
		if seen[p.Name] {
			t.Fatalf("preset %q twice", p.Name)
		}
		seen[p.Name] = true
		if p.Name != PresetCustom && p.BaseURL == "" {
			t.Fatalf("preset %q has no base URL", p.Name)
		}
		if _, ok := PresetOf(p.Name); !ok {
			t.Fatalf("PresetOf(%q) failed", p.Name)
		}
	}
	ollama, _ := PresetOf(PresetOllama)
	if !ollama.Local || ollama.BaseURL != "http://localhost:11434/v1" {
		t.Fatalf("ollama preset %+v", ollama)
	}
	lm, _ := PresetOf(PresetLMStudio)
	if !lm.Local || lm.BaseURL != "http://localhost:1234/v1" {
		t.Fatalf("lm studio preset %+v", lm)
	}
	openai, _ := PresetOf(PresetOpenAI)
	if openai.Local {
		t.Fatal("OpenAI must not be local")
	}
	if _, ok := PresetOf("nope"); ok {
		t.Fatal("unknown preset accepted")
	}
}

func TestValidate(t *testing.T) {
	good := DefaultConfig()
	good.APIKey = "sk-test"
	if err := good.Validate(); err != nil {
		t.Fatalf("good config: %v", err)
	}
	cases := []struct {
		name string
		mod  func(*AIConfig)
	}{
		{"unknown provider", func(c *AIConfig) { c.Provider = "nope" }},
		{"no base URL", func(c *AIConfig) { c.Provider = PresetCustom; c.BaseURL = "" }},
		{"bad scheme", func(c *AIConfig) { c.BaseURL = "ftp://x/v1" }},
		{"no host", func(c *AIConfig) { c.BaseURL = "https://" }},
		{"no model", func(c *AIConfig) { c.Model = "  " }},
		{"no key", func(c *AIConfig) { c.APIKey = "" }},
		{"bad temperature", func(c *AIConfig) { c.Temperature = 3 }},
		{"bad mode", func(c *AIConfig) { c.Mode = "loud" }},
	}
	for _, tc := range cases {
		c := DefaultConfig()
		c.APIKey = "sk-test"
		tc.mod(&c)
		if err := c.Validate(); err == nil {
			t.Errorf("%s: accepted", tc.name)
		}
	}
	// Local presets need no key.
	for _, p := range []ProviderPreset{PresetOllama, PresetLMStudio} {
		c := DefaultConfig()
		c.Provider = p
		pr, _ := PresetOf(p)
		c.BaseURL, c.APIKey = pr.BaseURL, ""
		if err := c.Validate(); err != nil {
			t.Errorf("%s without key: %v", p, err)
		}
	}
}

func TestMaskAndRedact(t *testing.T) {
	if got := MaskKey(""); got != "Not set" {
		t.Errorf("empty key: %q", got)
	}
	if got := MaskKey("ab"); got == "ab" || strings.Contains(got, "ab") {
		t.Errorf("short key leaked: %q", got)
	}
	if got := MaskKey("sk-1234567890"); got != "••••7890" {
		t.Errorf("masked: %q", got)
	}
	c := DefaultConfig()
	c.APIKey = "sk-secret-key"
	redacted := c.Redact("using sk-secret-key now")
	if strings.Contains(redacted, "sk-secret-key") {
		t.Errorf("key in redacted text: %q", redacted)
	}
}

func TestSaveLoad(t *testing.T) {
	dir := t.TempDir()
	cfg := DefaultConfig()
	cfg.Enabled = true
	cfg.Provider = PresetGroq
	cfg.BaseURL = "https://api.groq.com/openai/v1"
	cfg.APIKey = "gsk-secret"
	cfg.Model = "llama-3.3-70b-versatile"
	cfg.Mode = ModeAuto
	if err := cfg.Save(dir); err != nil {
		t.Fatalf("save: %v", err)
	}
	info, err := os.Stat(filepath.Join(dir, FileName))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("ai.json mode %o, want 0600", info.Mode().Perm())
	}
	got, err := Load(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got != cfg {
		t.Fatalf("loaded %+v, want %+v", got, cfg)
	}
	// Missing file loads the default; a corrupt one errors.
	empty := t.TempDir()
	def, err := Load(empty)
	if err != nil || def != DefaultConfig() {
		t.Fatalf("missing file: %+v, %v", def, err)
	}
	if err := os.WriteFile(filepath.Join(empty, FileName), []byte("{nope"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(empty); err == nil {
		t.Fatal("corrupt file loaded")
	}
	if err := DefaultConfig().Save(""); err == nil {
		t.Fatal("saved without a directory")
	}
}
