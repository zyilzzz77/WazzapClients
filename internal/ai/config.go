// Package ai is the provider-agnostic AI agent: an OpenAI-compatible
// chat client (stdlib net/http only), the agent that builds its context
// and prompts, and the pure rules for auto-reply. It never sees media
// bytes, view-once content or messages from another account: only the
// text and captions the caller hands it.
//
// The config lives per account, as <account data dir>/ai.json (0600),
// so one account's key never leaks into another's.
package ai

import (
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// AIConfig is one account's AI setup.
type AIConfig struct {
	Enabled            bool           `json:"enabled"`
	Provider           ProviderPreset `json:"provider"`
	BaseURL            string         `json:"base_url"`
	APIKey             string         `json:"api_key,omitempty"`
	Model              string         `json:"model"`
	Temperature        float64        `json:"temperature"`
	SystemPrompt       string         `json:"system_prompt,omitempty"`
	MaxContextMessages int            `json:"max_context_messages"`
	// Mode is ModeOff, ModeSuggest or ModeAuto.
	Mode string `json:"mode"`
}

// DefaultConfig is the config a fresh account starts with: off, OpenAI,
// and no key.
func DefaultConfig() AIConfig {
	return AIConfig{
		Enabled:            false,
		Provider:           PresetOpenAI,
		BaseURL:            "https://api.openai.com/v1",
		Model:              "gpt-4o-mini",
		Temperature:        0.7,
		MaxContextMessages: 20,
		Mode:               ModeSuggest,
	}
}

// DefaultSystemPrompt is the prompt used when the account has none of
// its own. The language follows the chat; Indonesian is the default.
func DefaultSystemPrompt() string {
	return "You are a helpful assistant inside a WhatsApp chat. " +
		"Answer briefly and plainly. Reply in the language of the " +
		"conversation; when in doubt, use Indonesian."
}

// EffectiveSystemPrompt is the account's prompt, or the default.
func (c AIConfig) EffectiveSystemPrompt() string {
	if strings.TrimSpace(c.SystemPrompt) != "" {
		return c.SystemPrompt
	}
	return DefaultSystemPrompt()
}

// EffectiveMaxContext is how many recent messages go to the AI.
func (c AIConfig) EffectiveMaxContext() int {
	if c.MaxContextMessages <= 0 {
		return 20
	}
	return c.MaxContextMessages
}

// Validate checks the config: a reachable-looking endpoint, a model,
// and a key unless the provider runs locally.
func (c AIConfig) Validate() error {
	p, ok := PresetOf(c.Provider)
	if !ok {
		return errors.New("unknown AI provider")
	}
	if c.BaseURL == "" && p.BaseURL == "" {
		return errors.New("enter the AI endpoint's base URL")
	}
	u, err := url.Parse(c.BaseURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return errors.New("the base URL must be an http(s) address, like https://api.openai.com/v1")
	}
	if strings.TrimSpace(c.Model) == "" {
		return errors.New("enter a model, like gpt-4o-mini")
	}
	if !p.Local && strings.TrimSpace(c.APIKey) == "" {
		return errors.New("enter an API key for " + p.Title)
	}
	if c.Temperature < 0 || c.Temperature > 2 {
		return errors.New("temperature must be between 0 and 2")
	}
	switch c.Mode {
	case "", ModeOff, ModeSuggest, ModeAuto:
	default:
		return errors.New("unknown AI mode")
	}
	return nil
}

// MaskedKey shows the key the way the settings page does: never more
// than its last 4 characters.
func (c AIConfig) MaskedKey() string {
	return MaskKey(c.APIKey)
}

// MaskKey masks a key, showing at most its last 4 characters.
func MaskKey(key string) string {
	if key == "" {
		return "Not set"
	}
	if len(key) <= 4 {
		return "••••"
	}
	return "••••" + key[len(key)-4:]
}

// Redact replaces the key in s with a mask, so logs never carry it.
func (c AIConfig) Redact(s string) string {
	if c.APIKey == "" {
		return s
	}
	return strings.ReplaceAll(s, c.APIKey, "••••")
}

// FileName is the config file's name inside an account's data dir.
const FileName = "ai.json"

// Load reads dir's config, or the default when there is none yet.
func Load(dir string) (AIConfig, error) {
	cfg := DefaultConfig()
	if dir == "" {
		return cfg, nil
	}
	b, err := os.ReadFile(filepath.Join(dir, FileName))
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, err
	}
	var saved AIConfig
	if err := json.Unmarshal(b, &saved); err != nil {
		return cfg, err
	}
	if saved.Provider == "" {
		saved.Provider = cfg.Provider
	}
	if saved.BaseURL == "" {
		if p, ok := PresetOf(saved.Provider); ok {
			saved.BaseURL = p.BaseURL
		}
	}
	if saved.MaxContextMessages <= 0 {
		saved.MaxContextMessages = cfg.MaxContextMessages
	}
	if saved.Mode == "" {
		saved.Mode = cfg.Mode
	}
	return saved, nil
}

// Save writes c to dir's config file (0600), atomically.
func (c AIConfig) Save(dir string) error {
	if dir == "" {
		return errors.New("no account data directory")
	}
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	path := filepath.Join(dir, FileName)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
