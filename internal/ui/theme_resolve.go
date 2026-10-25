package ui

// Themes beyond Light and Dark: Dark OLED, Navy, Forest, Sepia and
// WhatsApp Green. Each is a full Palette (see palettes_dark.go and
// palettes_light.go), so every widget keeps reading its colors from
// u.pal and a switch applies at once, without a restart.

// prefThemeAcct is the per-account theme override in Backend.Pref. It is
// not in appPrefs, so it stays with its account; "" means the global
// default (prefTheme) instead.
const prefThemeAcct = "theme_acct"

// themeNames names every value prefTheme and prefThemeAcct take.
var themeNames = map[string]string{
	"":        "System default",
	"light":   "Light",
	"dark":    "Dark",
	"oled":    "Dark OLED",
	"navy":    "Navy",
	"forest":  "Forest",
	"sepia":   "Sepia",
	"wagreen": "WhatsApp Green",
}

// themeNamesList is the themes in the order the settings page lists them.
var themeNamesList = []string{"", "light", "dark", "oled", "navy", "forest", "sepia", "wagreen"}

// themePalette returns the palette and dark flag for a theme name; ""
// follows the system.
func themePalette(name string, sysDark bool) (*Palette, bool) {
	switch name {
	case "light":
		return &lightPalette, false
	case "dark":
		return &darkPalette, true
	case "oled":
		return &oledPalette, true
	case "navy":
		return &navyPalette, true
	case "forest":
		return &forestPalette, true
	case "sepia":
		return &sepiaPalette, false
	case "wagreen":
		return &waGreenPalette, false
	}
	if sysDark {
		return &darkPalette, true
	}
	return &lightPalette, false
}

// resolveTheme picks the effective theme: the account's override, else
// the global default, else the system's. A system that doesn't say gets
// the dark theme, which was the app's own default.
func resolveTheme(global, acct string) (*Palette, bool) {
	name := acct
	if name == "" {
		name = global
	}
	dark, ok := systemDark()
	if !ok {
		dark = true
	}
	return themePalette(name, dark)
}

// themePage lists the themes twice: the default for all accounts, and
// this account's own override. A pick applies at once.
func (u *UI) themePage() []settingsSection {
	rows := func(cur string, set func(string), def string) []settingRow {
		out := make([]settingRow, 0, len(themeNamesList))
		for _, t := range themeNamesList {
			t := t
			title := themeNames[t]
			if t == "" {
				title = def
			}
			out = append(out, settingRow{key: "theme:" + t, kind: setRadio,
				title: title, on: cur == t, run: func() { set(t) }})
		}
		return out
	}
	return []settingsSection{
		{title: "Default for all accounts", rows: rows(u.backend.Pref(prefTheme), u.setTheme, "System default")},
		{title: "This account", rows: rows(u.backend.Pref(prefThemeAcct), u.setThemeAcct, "Use global default"),
			note: "Use the global default, or a theme of this account's own."},
	}
}

// themeName is the theme in effect: the account's override, else the
// global default, else the system's.
func (u *UI) themeName() string {
	if t := u.backend.Pref(prefThemeAcct); t != "" {
		return themeNames[t]
	}
	return themeNames[u.backend.Pref(prefTheme)]
}

// setPalette switches the palette, like SetDark but for a named theme.
func (u *UI) setPalette(p *Palette, dark bool) {
	u.dark = dark
	u.pal = p
	u.th.Palette.Fg = p.Text
	u.th.Palette.Bg = p.Panel
	richBlocks.m = nil // spans carry palette colors (mentions, links)
	u.th.Palette.ContrastBg = p.Green
}
