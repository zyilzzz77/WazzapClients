package ui

import (
	"fmt"
	"image/color"
	"strconv"
	"strings"
)

// Custom wallpaper, like WhatsApp on the phone: a chat's own wallpaper,
// or the global default. A choice is stored in Backend.Pref, following
// the chat-theme precedent ("theme:<chatID>"); a photo file lives in the
// account's data dir, downsized at ingest, never full-res in memory.

// prefWallpaper is the global default wallpaper (in appPrefs, so it
// carries across accounts); prefWallpaperChat keys a chat's own.
const prefWallpaper = "wallpaper"
const prefWallpaperChat = "wallpaper:"

// prefWallpaperDim is the global wallpaper dimming, 0-100;
// prefWallpaperDimChat keys a chat's own.
const prefWallpaperDim = "wallpaper_dim"
const prefWallpaperDimChat = "wallpaper_dim:"

// maxWallpaperSide is the longest side a stored wallpaper photo keeps;
// maxWallpaperBytes is the largest picked file accepted.
const maxWallpaperSide = 1920
const maxWallpaperBytes = 32 << 20

type wallKind int

const (
	wallDefault wallKind = iota // use the global default (a chat's choice only)
	wallDoodle                  // the built-in doodle pattern
	wallSolid                   // one colour
	wallImage                   // an uploaded photo
)

// wallChoice is one wallpaper: its kind, a solid colour, or a photo file.
type wallChoice struct {
	kind  wallKind
	color color.NRGBA // solid
	file  string      // stored photo filename
	path  string      // the photo's full path, resolved (not persisted)
	dim   int         // dimming, 0-100
}

// parseWallChoice parses a stored wallpaper value: "doodle",
// "solid:#RRGGBB", "image:<filename>". "" (and anything unknown) means
// unset.
func parseWallChoice(v string) (wallChoice, bool) {
	switch {
	case v == "":
		return wallChoice{}, false
	case v == "doodle":
		return wallChoice{kind: wallDoodle}, true
	case strings.HasPrefix(v, "solid:"):
		c, err := parseHexColor(strings.TrimPrefix(v, "solid:"))
		if err != nil {
			return wallChoice{}, false
		}
		return wallChoice{kind: wallSolid, color: c}, true
	case strings.HasPrefix(v, "image:"):
		f := strings.TrimPrefix(v, "image:")
		if f == "" || strings.ContainsAny(f, `/\`) {
			return wallChoice{}, false
		}
		return wallChoice{kind: wallImage, file: f}, true
	}
	return wallChoice{}, false
}

// String stores a choice: "" for the default.
func (c wallChoice) String() string {
	switch c.kind {
	case wallSolid:
		return "solid:" + hexColor(c.color)
	case wallImage:
		return "image:" + c.file
	case wallDoodle:
		return "doodle"
	}
	return ""
}

// hexColor is a colour as RRGGBB.
func hexColor(c color.NRGBA) string {
	return fmt.Sprintf("%02x%02x%02x", c.R, c.G, c.B)
}

// parseHexColor reads RRGGBB.
func parseHexColor(s string) (color.NRGBA, error) {
	if len(s) != 6 {
		return color.NRGBA{}, fmt.Errorf("bad colour %q", s)
	}
	v, err := strconv.ParseUint(s, 16, 32)
	if err != nil {
		return color.NRGBA{}, err
	}
	return rgb(uint32(v)), nil
}

// wallpaperFor returns the wallpaper of chat id: its own, else the global
// default, else the doodle.
func (u *UI) wallpaperFor(id string) wallChoice {
	if c, ok := parseWallChoice(u.backend.Pref(prefWallpaperChat + id)); ok {
		return u.withDim(id, c)
	}
	if c, ok := parseWallChoice(u.backend.Pref(prefWallpaper)); ok {
		return u.withDim(id, c)
	}
	return wallChoice{kind: wallDoodle}
}

// wallpaperShown is the wallpaper the open chat draws with, cached per
// chat so a frame doesn't read prefs. Setters drop the cache.
func (u *UI) wallpaperShown(id string) wallChoice {
	wp := &u.conv.wallpaper
	if wp.shownFor != id {
		wp.shown, wp.shownFor = u.wallpaperFor(id), id
	}
	return wp.shown
}

// dropWallpaperCache forgets the shown wallpaper, so the next frame
// re-reads it.
func (u *UI) dropWallpaperCache() { u.conv.wallpaper.shownFor = "" }

// withDim fills in a choice's dimming and its photo's path.
func (u *UI) withDim(id string, c wallChoice) wallChoice {
	c.dim = u.wallpaperDim(id, c.kind)
	if c.kind == wallImage {
		c.path = u.wallpaperPath(c.file)
	}
	return c
}

// wallpaperDim resolves the dimming of a wallpaper: the chat's, else the
// global, else 30 for a photo (so the date chip stays readable) and 0
// otherwise.
func (u *UI) wallpaperDim(id string, kind wallKind) int {
	for _, key := range []string{prefWallpaperDimChat + id, prefWallpaperDim} {
		if v := u.backend.Pref(key); v != "" {
			if d, err := strconv.Atoi(v); err == nil {
				return clampDim(d)
			}
		}
	}
	if kind == wallImage {
		return 30
	}
	return 0
}

// clampDim keeps a dimming in 0-100.
func clampDim(d int) int { return min(100, max(0, d)) }

// wallpaperName describes the global default wallpaper, for Settings.
func (u *UI) wallpaperName() string {
	if c, ok := parseWallChoice(u.backend.Pref(prefWallpaper)); ok {
		switch c.kind {
		case wallSolid:
			return "Solid colour"
		case wallImage:
			return "Photo"
		}
	}
	return "Doodle"
}
