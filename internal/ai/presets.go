package ai

// ProviderPreset names an AI provider the app knows how to reach.
type ProviderPreset string

// The presets: an OpenAI-compatible endpoint each, or a local server.
const (
	PresetOpenAI     ProviderPreset = "openai"
	PresetGroq       ProviderPreset = "groq"
	PresetOpenRouter ProviderPreset = "openrouter"
	PresetGemini     ProviderPreset = "gemini"
	PresetOllama     ProviderPreset = "ollama"
	PresetLMStudio   ProviderPreset = "lmstudio"
	PresetCustom     ProviderPreset = "custom"
)

// preset is what a preset stands for: its default endpoint, and whether
// it runs on this computer (so no API key is needed).
type preset struct {
	Title   string
	BaseURL string
	Local   bool
	Model   string // a model that works there, for a first run
}

// Presets lists the presets, in the order the settings page shows them.
func Presets() []struct {
	Name ProviderPreset
	preset
} {
	return []struct {
		Name ProviderPreset
		preset
	}{
		{PresetOpenAI, preset{"OpenAI", "https://api.openai.com/v1", false, "gpt-4o-mini"}},
		{PresetGroq, preset{"Groq", "https://api.groq.com/openai/v1", false, "llama-3.3-70b-versatile"}},
		{PresetOpenRouter, preset{"OpenRouter", "https://openrouter.ai/api/v1", false, "openai/gpt-4o-mini"}},
		{PresetGemini, preset{"Google Gemini", "https://generativelanguage.googleapis.com/v1beta/openai", false, "gemini-2.0-flash"}},
		{PresetOllama, preset{"Ollama (local)", "http://localhost:11434/v1", true, "llama3.1"}},
		{PresetLMStudio, preset{"LM Studio (local)", "http://localhost:1234/v1", true, "local-model"}},
		{PresetCustom, preset{"Custom", "", false, ""}},
	}
}

// PresetOf returns p's preset, or false when p isn't one.
func PresetOf(p ProviderPreset) (preset, bool) {
	for _, pr := range Presets() {
		if pr.Name == p {
			return pr.preset, true
		}
	}
	return preset{}, false
}

// Mode names what the AI may do on its own.
const (
	ModeOff     = "off"     // nothing automatic
	ModeSuggest = "suggest" // drafts and answers only when asked
	ModeAuto    = "auto"    // auto-reply in the chats allowed, see autoreply.go
)
