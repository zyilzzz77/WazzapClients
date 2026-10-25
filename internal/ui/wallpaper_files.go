package ui

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strconv"

	"github.com/chomosuke9/wazzapclients/internal/photo"
)

// Where a wallpaper photo lives, and how a choice is stored.

// wallpaperDir is where this account's wallpaper photos live.
func (u *UI) wallpaperDir() string {
	return filepath.Join(u.backend.DataDir(), "wallpaper")
}

// wallpaperPath is the stored photo file's full path.
func (u *UI) wallpaperPath(file string) string {
	return filepath.Join(u.wallpaperDir(), file)
}

// setWallpaper stores a chat's wallpaper (wallDefault: the global one).
// A replaced photo file is removed.
func (u *UI) setWallpaper(id string, c wallChoice) {
	old, _ := parseWallChoice(u.backend.Pref(prefWallpaperChat + id))
	u.backend.SetPref(prefWallpaperChat+id, c.String())
	u.removeWallpaperFile(old, c)
	u.dropWallpaperCache()
}

// setWallpaperDefault stores the global default wallpaper.
func (u *UI) setWallpaperDefault(c wallChoice) {
	old, _ := parseWallChoice(u.backend.Pref(prefWallpaper))
	u.backend.SetPref(prefWallpaper, c.String())
	u.removeWallpaperFile(old, c)
	u.dropWallpaperCache()
}

// removeWallpaperFile deletes a replaced photo file.
func (u *UI) removeWallpaperFile(old, new wallChoice) {
	if old.kind == wallImage && old.file != "" && old.file != new.file {
		os.Remove(u.wallpaperPath(old.file))
	}
}

// setWallpaperDim stores a chat's dimming; "" is the global one.
func (u *UI) setWallpaperDim(id string, dim int) {
	key := prefWallpaperDimChat + id
	if id == "" {
		key = prefWallpaperDim
	}
	u.backend.SetPref(key, strconv.Itoa(clampDim(dim)))
	u.dropWallpaperCache()
}

// saveWallpaperImage downsizes the picked file to at most
// maxWallpaperSide px on the long side and stores it as a JPEG in the
// account's wallpaper dir. It returns the stored filename.
func (u *UI) saveWallpaperImage(path string) (string, error) {
	st, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if st.Size() > maxWallpaperBytes {
		return "", fmt.Errorf("that picture is too big (over %s)", formatSize(maxWallpaperBytes))
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	release := acquireDecode(data)
	img, _, err := image.Decode(bytes.NewReader(data))
	release()
	if err != nil {
		return "", err
	}
	jpg, _, _, err := photo.Fit(img, maxWallpaperSide)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	name := "wallpaper-" + hex.EncodeToString(sum[:8]) + ".jpg"
	if err := os.MkdirAll(u.wallpaperDir(), 0o700); err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(u.wallpaperDir(), name), jpg, 0o600); err != nil {
		return "", err
	}
	return name, nil
}
