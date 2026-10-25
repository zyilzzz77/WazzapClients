package ui

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chomosuke9/wazzapclients/internal/mock"
	"github.com/chomosuke9/wazzapclients/internal/model"
)

// A status photo posts at the quality the send view chose: HD in, HD out.
func TestStatusHDQuality(t *testing.T) {
	b := mock.New()
	u := New(b)
	u.Start(func() {})
	u.SelectID("rina")
	u.setPage(pageStatus)

	dir := t.TempDir()
	photo := filepath.Join(dir, "photo.png")
	f, err := os.Create(photo)
	if err != nil {
		t.Fatal(err)
	}
	img := image.NewNRGBA(image.Rect(0, 0, 40, 30))
	img.SetNRGBA(1, 1, color.NRGBA{R: 0xff, A: 0xff})
	png.Encode(f, img)
	f.Close()

	u.attachPaths([]string{photo})
	if !u.postingStatus() {
		t.Fatal("the photo didn't become a status")
	}
	// The HD toggle, as the quality menu offers it.
	u.attach.quality = model.QualityHD
	u.conv.composer.SetText("caption")
	u.sendComposer()
	u.flushOutbox()
	u.applyEvents()
	if got := b.LastStatusPost(); got.File == nil || got.File.Quality != model.QualityHD {
		t.Fatalf("posted status quality %+v, want HD", got.File)
	}
}

// A status starts at Standard quality, whatever a chat used last.
func TestStatusDefaultQuality(t *testing.T) {
	b := mock.New()
	u := New(b)
	u.Start(func() {})
	u.SelectID("rina")
	u.attach.quality = model.QualityHD
	u.setPage(pageStatus)

	dir := t.TempDir()
	photo := filepath.Join(dir, "photo.png")
	f, err := os.Create(photo)
	if err != nil {
		t.Fatal(err)
	}
	img := image.NewNRGBA(image.Rect(0, 0, 40, 30))
	png.Encode(f, img)
	f.Close()

	u.attachPaths([]string{photo})
	if !u.postingStatus() {
		t.Fatal("the photo didn't become a status")
	}
	if u.attach.quality != model.QualityStandard {
		t.Errorf("status quality is %v, want Standard", u.attach.quality)
	}
}

// The quality menu says what happens to videos: sent as they are, with
// their size.
func TestStatusVideoNote(t *testing.T) {
	u := New(mock.New())
	u.Start(func() {})
	u.SelectID("rina")
	u.setPage(pageStatus)

	dir := t.TempDir()
	video := filepath.Join(dir, "clip.mp4")
	os.WriteFile(video, make([]byte, 123456), 0o600)
	u.attachPaths([]string{video})
	if !u.postingStatus() {
		t.Fatal("the video didn't become a status")
	}
	note := u.attach.videoSizeNote()
	if !strings.Contains(note, "1 video") || !strings.Contains(note, "sent as it is") {
		t.Errorf("video note is %q", note)
	}
	if got := fileSizeLabel(video); !strings.Contains(got, "clip.mp4") || !strings.Contains(got, "·") {
		t.Errorf("file size label is %q", got)
	}
}
