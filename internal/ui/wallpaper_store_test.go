package ui

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/chomosuke9/wazzapclients/internal/mock"
)

// Wallpaper choices round-trip through their stored form.
func TestWallpaperChoiceRoundTrip(t *testing.T) {
	for _, v := range []string{"doodle", "solid:075e54", "image:wallpaper-abc123.jpg"} {
		c, ok := parseWallChoice(v)
		if !ok || c.String() != v {
			t.Errorf("%q parsed to %+v, stored as %q", v, c, c.String())
		}
	}
	for _, v := range []string{"", "bogus", "solid:xyz", "solid:12345", "image:", "image:../escape.jpg", "image:a/b.jpg"} {
		if _, ok := parseWallChoice(v); ok {
			t.Errorf("%q parsed, want unset", v)
		}
	}
}

// A chat's wallpaper wins over the global default, which wins over the
// doodle; dimming defaults to 30 for a photo and 0 otherwise.
func TestWallpaperFor(t *testing.T) {
	b := mock.New()
	b.SetDataDir(t.TempDir())
	u := New(b)
	if got := u.wallpaperFor("rina"); got.kind != wallDoodle || got.dim != 0 {
		t.Errorf("no wallpaper: %+v, want the doodle undimmed", got)
	}
	u.setWallpaperDefault(wallChoice{kind: wallSolid, color: rgb(0x075e54)})
	if got := u.wallpaperFor("rina"); got.kind != wallSolid || got.dim != 0 {
		t.Errorf("global solid: %+v", got)
	}
	u.setWallpaper("rina", wallChoice{kind: wallImage, file: "wallpaper-x.jpg"})
	got := u.wallpaperFor("rina")
	if got.kind != wallImage || got.dim != 30 || got.path == "" {
		t.Errorf("chat photo: %+v, want dim 30 and a path", got)
	}
	if got := u.wallpaperFor("work"); got.kind != wallSolid {
		t.Errorf("another chat: %+v, want the global solid", got)
	}
	// An explicit dimming wins over the default, and clamps.
	u.setWallpaperDim("rina", 120)
	if got := u.wallpaperFor("rina"); got.dim != 100 {
		t.Errorf("dim is %d, want 100", got.dim)
	}
	u.setWallpaperDim("rina", 0)
	if got := u.wallpaperFor("rina"); got.dim != 0 {
		t.Errorf("dim is %d, want 0", got.dim)
	}
	// The shown wallpaper is cached per chat until a setter drops it.
	if got := u.wallpaperShown("rina"); got.kind != wallImage {
		t.Fatalf("shown wallpaper: %+v, want the photo", got)
	}
	u.backend.SetPref(prefWallpaperChat+"rina", "doodle")
	if got := u.wallpaperShown("rina"); got.kind != wallImage {
		t.Errorf("cached wallpaper: %+v, want the photo still", got)
	}
	u.dropWallpaperCache()
	if got := u.wallpaperShown("rina"); got.kind != wallDoodle {
		t.Errorf("after dropping the cache: %+v, want the doodle", got)
	}
}

// A picked photo is stored downsized, never full-res.
func TestWallpaperImageIngest(t *testing.T) {
	b := mock.New()
	dir := t.TempDir()
	b.SetDataDir(dir)
	u := New(b)

	// A 4000x3000 photo becomes at most 1920 px on its long side.
	img := image.NewNRGBA(image.Rect(0, 0, 4000, 3000))
	img.SetNRGBA(10, 10, color.NRGBA{R: 0xff, A: 0xff})
	src := filepath.Join(dir, "big.png")
	f, err := os.Create(src)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
	f.Close()
	name, err := u.saveWallpaperImage(src)
	if err != nil {
		t.Fatalf("saving the wallpaper: %v", err)
	}
	stored := filepath.Join(u.wallpaperDir(), name)
	st, err := os.Stat(stored)
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode().Perm() != 0o600 {
		t.Errorf("stored wallpaper mode %o, want 0600", st.Mode().Perm())
	}
	cfg, _, err := image.DecodeConfig(mustOpen(t, stored))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Width > maxWallpaperSide || cfg.Height > maxWallpaperSide {
		t.Errorf("stored wallpaper is %dx%d, want at most %d on a side", cfg.Width, cfg.Height, maxWallpaperSide)
	}
	if cfg.Width != maxWallpaperSide {
		t.Errorf("stored wallpaper is %d px wide, want %d", cfg.Width, maxWallpaperSide)
	}
	// The same picture again reuses the file.
	if name2, err := u.saveWallpaperImage(src); err != nil || name2 != name {
		t.Errorf("re-saving gave %q (%v), want %q", name2, err, name)
	}
}

// An oversize file is refused before it is read, and a file that isn't
// a picture is refused too.
func TestWallpaperImageRejected(t *testing.T) {
	b := mock.New()
	dir := t.TempDir()
	b.SetDataDir(dir)
	u := New(b)

	huge := filepath.Join(dir, "huge.jpg")
	hf, err := os.Create(huge)
	if err != nil {
		t.Fatal(err)
	}
	hf.Close()
	if err := os.Truncate(huge, maxWallpaperBytes+1); err != nil {
		t.Fatal(err)
	}
	if _, err := u.saveWallpaperImage(huge); err == nil {
		t.Error("an oversize picture was accepted")
	}
	txt := filepath.Join(dir, "notes.txt")
	os.WriteFile(txt, []byte("hello"), 0o600)
	if _, err := u.saveWallpaperImage(txt); err == nil {
		t.Error("a text file was accepted as a wallpaper")
	}
}

func mustOpen(t *testing.T, path string) *os.File {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { f.Close() })
	return f
}
