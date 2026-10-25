package ui

import (
	"image"
	"math"

	"gioui.org/f32"
	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"

	"github.com/chomosuke9/wazzapclients/internal/filepick"
)

// The wallpaper dialog's rows: the radio option, the colour swatches,
// the dimming slider, and the photo picker.

// wallpaperOption draws one radio row: a circle and a label.
func (u *UI) wallpaperOption(gtx C, label string, on bool) D {
	p := u.pal
	h := gtx.Dp(40)
	return vcenter(gtx, h, func(gtx C) D {
		return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx C) D {
				s := gtx.Dp(20)
				mid := image.Pt(s/2, s/2)
				ring := p.TextSecondary
				if on {
					ring = p.Green
				}
				strokeArc(gtx, f32.Pt(float32(mid.X), float32(mid.Y)), float32(s/2-1),
					0, 2*math.Pi, float32(max(1, gtx.Dp(2))), ring)
				if on {
					fillCircle(gtx, mid, s/4, p.Green)
				}
				return D{Size: image.Pt(s, s)}
			}),
			layout.Rigid(layout.Spacer{Width: 12}.Layout),
			layout.Rigid(u.label(15.5, label, p.Text).Layout),
		)
	})
}

// wallpaperSwatches draws the solid colour swatches in a row.
func (u *UI) wallpaperSwatches(gtx C) D {
	p := u.pal
	cur, _ := u.wallpaperChoiceOf(u.wallpaperDlg.chatID)
	s := gtx.Dp(40)
	gap := gtx.Dp(8)
	w := len(wallSolids)*(s+gap) - gap
	x := 0
	for _, hex := range wallSolids {
		hex := hex
		func() {
			defer op.Offset(image.Pt(x, 0)).Push(gtx.Ops).Pop()
			cl := u.btn("wall:solid:" + hexColor(rgb(hex)))
			clickable(gtx, cl, func(gtx C) D {
				fillRRect(gtx, image.Rect(0, 0, s, s), gtx.Dp(8), rgb(hex))
				if cur.kind == wallSolid && cur.color == rgb(hex) {
					strokeRRect(gtx, image.Rect(0, 0, s, s), gtx.Dp(8), float32(max(1, gtx.Dp(2))), p.Green)
				}
				return D{Size: image.Pt(s, s)}
			})
		}()
		x += s + gap
	}
	return D{Size: image.Pt(w, s)}
}

// wallpaperDimSlider is the dimming slider: drag it, or scroll over it.
// It saves when the drag ends, like the volume slider.
func (u *UI) wallpaperDimSlider(gtx C, w int) {
	d := &u.wallpaperDlg
	u.wallpaperDimEvents(gtx)
	d.w = w
	if w <= 0 {
		return
	}
	u.wallpaperDimDraw(gtx, w)
}

// wallpaperDimEvents reads the slider's pointer events for this frame.
func (u *UI) wallpaperDimEvents(gtx C) {
	d := &u.wallpaperDlg
	for {
		ev, ok := gtx.Event(pointer.Filter{Target: d, Kinds: pointer.Press | pointer.Drag | pointer.Release | pointer.Cancel | pointer.Scroll,
			ScrollY: pointer.ScrollRange{Min: -1e6, Max: 1e6}})
		if !ok {
			break
		}
		e, ok := ev.(pointer.Event)
		if !ok || d.w <= 0 {
			continue
		}
		t := min(1, max(0, e.Position.X/float32(d.w)))
		switch e.Kind {
		case pointer.Press:
			d.dragging = true
			d.dimVal = int(t*100 + 0.5)
		case pointer.Drag:
			if d.dragging {
				d.dimVal = int(t*100 + 0.5)
			}
		case pointer.Release, pointer.Cancel:
			if d.dragging {
				u.setWallpaperDim(d.chatID, d.dimVal)
			}
			d.dragging = false
		case pointer.Scroll:
			if e.Scroll.Y == 0 {
				continue
			}
			step := 5
			if e.Scroll.Y > 0 {
				step = -step
			}
			d.dimVal = clampDim(d.dimVal + step)
			u.setWallpaperDim(d.chatID, d.dimVal)
		}
	}
}

// wallpaperDimDraw draws the slider track, fill and knob.
func (u *UI) wallpaperDimDraw(gtx C, w int) {
	d := &u.wallpaperDlg
	p := u.pal
	hit := gtx.Dp(12)
	func() {
		defer op.Offset(image.Pt(0, -hit)).Push(gtx.Ops).Pop()
		defer clip.Rect{Max: image.Pt(w, 2*hit)}.Push(gtx.Ops).Pop()
		pointer.CursorPointer.Add(gtx.Ops)
		event.Op(gtx.Ops, d)
	}()
	th := gtx.Dp(4)
	fillRRect(gtx, image.Rect(0, -th/2, w, th-th/2), th/2, p.ChipBorder)
	kx := int(float32(d.dimVal)/100*float32(w) + 0.5)
	fillRRect(gtx, image.Rect(0, -th/2, kx, th-th/2), th/2, p.Green)
	kr := gtx.Dp(6)
	if d.dragging {
		kr = gtx.Dp(7)
	}
	fillCircle(gtx, image.Pt(kx, 0), kr, p.Green)
}

// pickWallpaperPhoto opens the system's file dialog for a photo, on its
// own goroutine; the path comes back on the dialog's channel.
func (u *UI) pickWallpaperPhoto() {
	d := &u.wallpaperDlg
	if d.picking {
		return
	}
	if d.picked == nil {
		d.picked = make(chan string, 1)
	}
	d.picking = true
	go func() {
		paths, err := filepick.Open("Choose a wallpaper photo", false,
			filepick.Filter{Name: "Pictures", Exts: photoExts}, filepick.Filter{Name: "All files"})
		if err != nil || len(paths) == 0 {
			d.picked <- ""
			return
		}
		d.picked <- paths[0]
	}()
}
