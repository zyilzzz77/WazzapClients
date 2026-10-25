package ui

import (
	"strconv"
	"strings"

	"gioui.org/layout"
	"gioui.org/widget"

	"github.com/chomosuke9/wazzapclients/internal/ai"
)

// aiSettings is the Settings > AI page: the switch, the mode, the
// provider, the endpoint and key, and Test connection. It follows the
// KLIPY key editor's pattern (gifs.go): editors save live.
func (u *UI) aiSettings() []settingsSection {
	a := &u.ai
	on := a.cfg.Enabled
	sec := settingsSection{title: "AI", rows: []settingRow{
		{key: "ai:enable", kind: setToggle, title: "AI features",
			sub: "Ask the AI about chats, draft replies, and auto-reply",
			on:  on, run: func() { u.aiToggle(!on) }},
	}}
	if !on {
		sec.note = "AI is off. Nothing is sent to any AI endpoint while it is off."
		return []settingsSection{sec}
	}
	sec.rows = append(sec.rows,
		settingRow{key: "ai:mode:suggest", kind: setRadio, title: "Suggest only",
			sub: "Drafts and answers only when you ask", on: a.cfg.Mode != ai.ModeAuto,
			run: func() { u.aiSetMode(ai.ModeSuggest) }},
		settingRow{key: "ai:mode:auto", kind: setRadio, title: "Auto-reply",
			sub: "Replies for you in chats, with a cooldown and limits",
			on:  a.cfg.Mode == ai.ModeAuto, run: func() { u.aiSetMode(ai.ModeAuto) }},
	)
	for _, p := range ai.Presets() {
		p := p
		sec.rows = append(sec.rows, settingRow{key: "ai:provider:" + string(p.Name),
			kind: setRadio, title: p.Title, sub: p.BaseURL,
			on: a.cfg.Provider == p.Name, run: func() { u.aiSetProvider(p.Name) }})
	}
	sec.rows = append(sec.rows,
		settingRow{key: "ai:url", kind: setCustom, w: u.aiField("Base URL", &a.urlEd, "https://api.openai.com/v1", func(t string) {
			if t != a.cfg.BaseURL {
				a.cfg.BaseURL, a.testedOK = t, false
				u.aiSave()
				u.rebuildAgent()
			}
		})},
		settingRow{key: "ai:key", kind: setCustom, w: u.aiKeyField()},
		settingRow{key: "ai:model", kind: setCustom, w: u.aiField("Model", &a.modelEd, "gpt-4o-mini", func(t string) {
			if t != a.cfg.Model {
				a.cfg.Model, a.testedOK = t, false
				u.aiSave()
				u.rebuildAgent()
			}
		})},
		settingRow{key: "ai:temp", kind: setCustom, w: u.aiField("Temperature (0–2)", &a.tempEd, "0.7", func(t string) {
			if f, err := strconv.ParseFloat(t, 64); err == nil && f >= 0 && f <= 2 && f != a.cfg.Temperature {
				a.cfg.Temperature = f
				u.aiSave()
				u.rebuildAgent()
			}
		})},
		settingRow{key: "ai:prompt", kind: setCustom, w: u.aiField("System prompt (empty for the default)", &a.promptEd, "You are a helpful assistant…", func(t string) {
			if t != a.cfg.SystemPrompt {
				a.cfg.SystemPrompt = t
				u.aiSave()
				u.rebuildAgent()
			}
		})},
	)
	sub := "Check the endpoint, key and model"
	if a.testResult != "" {
		sub = a.testResult
	}
	sec.rows = append(sec.rows, settingRow{key: "ai:test", title: "Test connection", sub: sub,
		run: func() { u.aiTestConnection() }})
	sec.note = "Your key is stored in this account's data folder only (ai.json), never in the chat database, " +
		"and is sent only to the endpoint above. Auto-reply also needs the account's AI switch in Settings > Account."
	return []settingsSection{sec}
}

// aiSetProvider switches preset, filling in its endpoint and model.
func (u *UI) aiSetProvider(p ai.ProviderPreset) {
	a := &u.ai
	pr, ok := ai.PresetOf(p)
	if !ok {
		return
	}
	a.cfg.Provider = p
	if pr.BaseURL != "" {
		a.cfg.BaseURL = pr.BaseURL
		a.urlEd.SetText(pr.BaseURL)
	}
	if pr.Model != "" {
		a.cfg.Model = pr.Model
		a.modelEd.SetText(pr.Model)
	}
	a.testedOK = false
	u.aiSave()
	u.rebuildAgent()
	u.settings.stale = true
}

// aiField is a settings row with a title and a one-line editor, which
// saves live through set, like the KLIPY key editor.
func (u *UI) aiField(title string, ed *widget.Editor, hint string, set func(string)) layout.Widget {
	return func(gtx C) D {
		return layout.Inset{Left: 24, Right: 24, Bottom: 4}.Layout(gtx, func(gtx C) D {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(u.label(15, title, u.pal.Text).Layout),
				layout.Rigid(func(gtx C) D {
					d := u.pollLine(gtx, ed, hint, false)
					if set != nil {
						set(strings.TrimSpace(ed.Text()))
					}
					return d
				}),
			)
		})
	}
}

// aiKeyField is the API key row: masked, with a Show/Hide button. The
// editor always holds the real key; only the display is masked.
func (u *UI) aiKeyField() layout.Widget {
	return func(gtx C) D {
		a := &u.ai
		if a.showKeyBtn.Clicked(gtx) {
			a.showKey = !a.showKey
			a.keyEd.Mask = 0 // shown
			if !a.showKey {
				a.keyEd.Mask = '•'
			}
		}
		return layout.Inset{Left: 24, Right: 24, Bottom: 4}.Layout(gtx, func(gtx C) D {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(u.label(15, "API key", u.pal.Text).Layout),
				layout.Rigid(func(gtx C) D {
					return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
						layout.Flexed(1, func(gtx C) D {
							d := u.pollLine(gtx, &a.keyEd, "sk-…", false)
							if k := strings.TrimSpace(a.keyEd.Text()); k != a.cfg.APIKey {
								a.cfg.APIKey, a.testedOK = k, false
								u.aiSave()
								u.rebuildAgent()
							}
							return d
						}),
						layout.Rigid(func(gtx C) D {
							label := "Show"
							if a.showKey {
								label = "Hide"
							}
							return clickable(gtx, &a.showKeyBtn, u.label(15, label, u.pal.Green).Layout)
						}),
					)
				}),
			)
		})
	}
}
