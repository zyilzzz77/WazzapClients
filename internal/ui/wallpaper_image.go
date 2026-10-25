package ui

import (
	"bytes"
	"image"
	"image/draw"
	"os"

	"gioui.org/op/paint"

	"github.com/chomosuke9/wazzapclients/internal/offheap"
)

// Rendering an uploaded wallpaper photo: decoded once per file, off-heap
// (ingest already downsized it to at most maxWallpaperSide px), drawn
// cover-cropped over the pane and dimmed toward the chat background, so
// the date chip and system rows stay readable. Bubbles are opaque, so
// they read on any wallpaper.

// layoutChoice draws the conversation's wallpaper: the doodle, a solid
// colour, or an uploaded photo.
func (wp *wallpaper) layoutChoice(gtx C, p *Palette, c wallChoice, doodles bool) D {
	sz := gtx.Constraints.Max
	switch c.kind {
	case wallSolid:
		fillRect(gtx, image.Rectangle{Max: sz}, c.color)
		return D{Size: sz}
	case wallImage:
		if wp.layoutWallImage(gtx, c, p, sz) {
			return D{Size: sz}
		}
		// A photo that won't load falls back to the doodle.
	}
	return wp.layout(gtx, p.ChatBg, p.Doodle, doodles)
}

// layoutWallImage draws the photo cover-cropped over the pane, dimmed
// toward the chat background. It reports whether the photo drew.
func (wp *wallpaper) layoutWallImage(gtx C, c wallChoice, p *Palette, sz image.Point) bool {
	if c.path == "" || sz.X <= 0 || sz.Y <= 0 {
		return false
	}
	if wp.imgFile != c.path {
		wp.imgFile = c.path
		wp.img, wp.imgSize = loadWallImage(c.path)
	}
	if wp.imgSize.X <= 0 {
		return false
	}
	paintCover(gtx, wp.img, wp.imgSize, image.Rectangle{Max: sz})
	if c.dim > 0 {
		veil := p.ChatBg
		veil.A = uint8(c.dim * 255 / 100)
		fillRect(gtx, image.Rectangle{Max: sz}, veil)
	}
	return true
}

// loadWallImage decodes the wallpaper file into an off-heap picture and
// its size, or nothing when it won't decode.
func loadWallImage(path string) (paint.ImageOp, image.Point) {
	data, err := os.ReadFile(path)
	if err != nil {
		return paint.ImageOp{}, image.Point{}
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return paint.ImageOp{}, image.Point{}
	}
	b := img.Bounds()
	dst := offheap.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(dst, dst.Bounds(), img, b.Min, draw.Src)
	return paint.NewImageOp(dst), image.Pt(b.Dx(), b.Dy())
}
