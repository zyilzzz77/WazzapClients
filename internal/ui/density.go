package ui

import "gioui.org/unit"

// Density is how tall the chat list's rows are: Compact fits about a
// sixth more chats on screen, Comfortable is the default. Unlike zoom,
// which scales everything, density only changes row heights and their
// spacing.

// prefDensity is the density in Backend.Pref: "compact", or ""
// (comfortable, the default).
const prefDensity = "density"

// densityNames names the values of prefDensity.
var densityNames = map[string]string{"": "Comfortable", "compact": "Compact"}

// densityScale is a density's row height multiplier.
func densityScale(name string) float32 {
	if name == "compact" {
		return 0.85
	}
	return 1
}

// densityRowH is a row's height in px at its base dp height.
func (u *UI) densityRowH(gtx C, base float32) int {
	return gtx.Dp(unit.Dp(base * densityScale(u.densityName)))
}

// setDensity remembers the density; rows resize at once.
func (u *UI) setDensity(name string) {
	u.densityName = name
	u.backend.SetPref(prefDensity, name)
}

// densityPage is the Density settings sub-page.
func (u *UI) densityPage() []settingsSection {
	cur := u.backend.Pref(prefDensity)
	row := func(name string) settingRow {
		return settingRow{key: "density:" + name, kind: setRadio, title: densityNames[name],
			on: cur == name, run: func() { u.setDensity(name) }}
	}
	return []settingsSection{{title: "Chat list density", rows: []settingRow{row(""), row("compact")},
		note: "Compact fits more chats in the list; Comfortable is roomier."}}
}
