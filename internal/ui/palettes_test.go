package ui

import (
	"image/color"
	"math"
	"reflect"
	"testing"

	"github.com/chomosuke9/wazzapclients/internal/mock"
)

// Every theme's palette is complete: no colour left at its zero value,
// and the OLED theme really is black.
func TestPalettesComplete(t *testing.T) {
	pals := map[string]Palette{
		"light": lightPalette, "dark": darkPalette, "oled": oledPalette,
		"navy": navyPalette, "forest": forestPalette,
		"sepia": sepiaPalette, "wagreen": waGreenPalette,
	}
	for name, p := range pals {
		v := reflect.ValueOf(p)
		for i := 0; i < v.NumField(); i++ {
			f := v.Field(i)
			switch f.Kind() {
			case reflect.Struct: // color.NRGBA
				if f.Interface().(color.NRGBA).A == 0 {
					t.Errorf("%s: %s is transparent", name, v.Type().Field(i).Name)
				}
			case reflect.Slice:
				if f.Len() == 0 {
					t.Errorf("%s: %s is empty", name, v.Type().Field(i).Name)
				}
			}
		}
	}
	if got := oledPalette.Frame; got != rgb(0x000000) {
		t.Errorf("OLED Frame is %#x, want #000000", got)
	}
	for _, f := range []color.NRGBA{oledPalette.Panel, oledPalette.ChatBg, oledPalette.Viewer,
		oledPalette.Dialog, oledPalette.Popup, oledPalette.Picker, oledPalette.StatusBg} {
		if f != rgb(0x000000) {
			t.Errorf("OLED background is %#x, want #000000", f)
		}
	}
	// ...while a bubble stays visible on it.
	if oledPalette.BubbleIn == rgb(0x000000) {
		t.Error("OLED BubbleIn is black: rows would vanish")
	}
}

// Text stays readable on every theme: WCAG AA-ish contrast for the text
// on its panel and bubbles.
func TestPaletteContrast(t *testing.T) {
	pals := map[string]Palette{
		"light": lightPalette, "dark": darkPalette, "oled": oledPalette,
		"navy": navyPalette, "forest": forestPalette,
		"sepia": sepiaPalette, "wagreen": waGreenPalette,
	}
	for name, p := range pals {
		for _, pair := range []struct {
			fg, bg color.NRGBA
			want   float64
			what   string
		}{
			{p.Text, p.Panel, 4.5, "Text/Panel"},
			{p.Text, p.BubbleIn, 4.5, "Text/BubbleIn"},
			{p.TextOut, p.BubbleOut, 4.5, "TextOut/BubbleOut"},
			{p.TextSecondary, p.Panel, 3, "TextSecondary/Panel"},
		} {
			if got := contrastRatio(pair.fg, pair.bg); got < pair.want {
				t.Errorf("%s: %s contrast %.2f, want %.1f", name, pair.what, got, pair.want)
			}
		}
	}
}

// luminance is a colour's relative luminance, per WCAG.
func luminance(c color.NRGBA) float64 {
	lin := func(v uint8) float64 {
		f := float64(v) / 255
		if f <= 0.04045 {
			return f / 12.92
		}
		return math.Pow((f+0.055)/1.055, 2.4)
	}
	return 0.2126*lin(c.R) + 0.7152*lin(c.G) + 0.0722*lin(c.B)
}

// contrastRatio is the WCAG contrast of fg on bg.
func contrastRatio(fg, bg color.NRGBA) float64 {
	la, lb := luminance(fg), luminance(bg)
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05)
}

// The resolver picks the named theme, follows the system for "", and the
// account's override wins over the global default.
func TestThemeResolve(t *testing.T) {
	if p, dark := themePalette("oled", false); p != &oledPalette || !dark {
		t.Errorf("oled resolved to dark=%v", dark)
	}
	if p, dark := themePalette("sepia", true); p != &sepiaPalette || dark {
		t.Errorf("sepia resolved to dark=%v", dark)
	}
	if p, dark := themePalette("", true); p != &darkPalette || !dark {
		t.Error("system dark didn't resolve to dark")
	}
	if p, dark := themePalette("", false); p != &lightPalette || dark {
		t.Error("system light didn't resolve to light")
	}
	if p, dark := themePalette("bogus", true); p != &darkPalette || !dark {
		t.Error("an unknown theme didn't fall back to dark")
	}
}

// A theme applies at once, and the account's own theme survives a new UI
// on the same backend while the global default stays another theme.
func TestThemePerAccount(t *testing.T) {
	b := mock.New()
	u := New(b)
	u.setTheme("oled")
	if u.pal != &oledPalette || !u.dark {
		t.Fatal("the global theme didn't apply")
	}
	u.setThemeAcct("sepia")
	if u.pal != &sepiaPalette || u.dark {
		t.Fatal("the account theme didn't apply")
	}
	if got := b.Pref(prefTheme); got != "oled" {
		t.Errorf("global theme pref is %q, want oled", got)
	}
	u2 := New(b)
	if u2.pal != &sepiaPalette {
		t.Error("a new UI didn't keep the account's theme")
	}
	u2.setThemeAcct("")
	if u2.pal != &oledPalette {
		t.Error("clearing the override didn't go back to the global theme")
	}
	if got := u2.themeName(); got != "Dark OLED" {
		t.Errorf("theme name is %q, want Dark OLED", got)
	}
}
