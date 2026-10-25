package ui

import (
	"testing"

	"gioui.org/layout"
	"gioui.org/unit"

	"github.com/chomosuke9/wazzapclients/internal/mock"
)

// Compact rows are shorter than Comfortable ones, and an unknown density
// stays Comfortable.
func TestDensity(t *testing.T) {
	if got := densityScale("compact"); got != 0.85 {
		t.Errorf("compact scale %.2f, want 0.85", got)
	}
	if got := densityScale(""); got != 1 {
		t.Errorf("comfortable scale %.2f, want 1", got)
	}
	if got := densityScale("bogus"); got != 1 {
		t.Errorf("unknown density scale %.2f, want 1", got)
	}
	gtx := layout.Context{Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}}
	u := New(mock.New())
	full := u.densityRowH(gtx, 76.3)
	u.densityName = "compact"
	if got := u.densityRowH(gtx, 76.3); got >= full {
		t.Errorf("compact row %d px, want under %d", got, full)
	}
	if got := u.densityRowH(gtx, 76.3); got != gtx.Dp(unit.Dp(76.3*0.85)) {
		t.Errorf("compact row %d px, want 76.3*0.85", got)
	}
}

// The density setting round-trips through prefs and the settings page
// names it.
func TestDensityPref(t *testing.T) {
	b := mock.New()
	u := New(b)
	u.setDensity("compact")
	if got := b.Pref(prefDensity); got != "compact" {
		t.Errorf("density pref is %q, want compact", got)
	}
	if got := densityNames[u.backend.Pref(prefDensity)]; got != "Compact" {
		t.Errorf("density name is %q, want Compact", got)
	}
	if got := New(b).densityName; got != "compact" {
		t.Errorf("a new UI reads density %q, want compact", got)
	}
}
