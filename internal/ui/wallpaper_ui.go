package ui

import (
	"image"

	"gioui.org/font"
	"gioui.org/layout"
)

// The wallpaper dialog: a chat's wallpaper, or the global default when
// id is "". It follows the chat theme dialog's shape (chatmenu.go).

// wallpaperDialog is the open wallpaper dialog's state.
type wallpaperDialog struct {
	chatID string
	// dimVal is the dimming the slider shows; dimKey is the choice it was
	// read for, so picking another wallpaper re-reads it.
	dimVal int
	dimKey string
	// dragging is a slider drag in progress; w is the slider track's
	// width last frame.
	dragging bool
	w        int
	picking  bool
	picked   chan string
}

// wallSolids are the solid colours the dialog offers.
var wallSolids = []uint32{0x075e54, 0x0b141a, 0xefeae2, 0xd9fdd3, 0x000000, 0xffffff}

// openWallpaper opens the wallpaper dialog for a chat, or the global
// default when id is "".
func (u *UI) openWallpaper(id string) {
	u.dialog = dialogState{kind: dialogWallpaper, title: id}
	d := &u.wallpaperDlg
	d.chatID, d.dimKey, d.dragging, d.picking = id, "", false, false
}

// setWallpaperChoice stores a wallpaper for a chat, or the global
// default when id is "".
func (u *UI) setWallpaperChoice(id string, c wallChoice) {
	if id == "" {
		u.setWallpaperDefault(c)
	} else {
		u.setWallpaper(id, c)
	}
}

// wallpaperChoiceOf is the stored choice the dialog edits.
func (u *UI) wallpaperChoiceOf(id string) (wallChoice, bool) {
	if id == "" {
		return parseWallChoice(u.backend.Pref(prefWallpaper))
	}
	return parseWallChoice(u.backend.Pref(prefWallpaperChat + id))
}

// wallpaperPanel draws the wallpaper dialog.
func (u *UI) wallpaperPanel(gtx C) D {
	d := &u.dialog
	dlg := &u.wallpaperDlg
	id := d.title
	p := u.pal
	cur, set := u.wallpaperChoiceOf(id)
	if !set {
		cur = wallChoice{kind: wallDoodle}
		if id != "" {
			cur = wallChoice{kind: wallDefault}
		}
	}
	// A picked photo, when the dialog asked for one.
	u.wallpaperPicked(id)
	// The slider follows the choice it belongs to.
	key := cur.String() + "|" + id
	if key != dlg.dimKey {
		dlg.dimKey = key
		dlg.dimVal = u.wallpaperDim(id, cur.kind)
	}

	u.wallpaperClicks(gtx, id)
	title := "Wallpaper"
	if id == "" {
		title = "Default wallpaper"
	}
	w := min(gtx.Dp(440), gtx.Constraints.Max.X-gtx.Dp(32))
	gtx.Constraints = layout.Constraints{Min: image.Pt(w, 0), Max: image.Pt(w, gtx.Constraints.Max.Y)}
	return layout.UniformInset(24).Layout(gtx, func(gtx C) D {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(u.label(19.5, title, p.Text, labelOpts{weight: font.Medium}).Layout),
			layout.Rigid(layout.Spacer{Height: 6}.Layout),
			layout.Rigid(u.label(14.5, "Only you see the wallpaper you choose.", p.TextSecondary).Layout),
			layout.Rigid(layout.Spacer{Height: 18}.Layout),
			layout.Rigid(func(gtx C) D {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					u.wallpaperRows(gtx, id, cur)...,
				)
			}),
			layout.Rigid(layout.Spacer{Height: 16}.Layout),
			layout.Rigid(func(gtx C) D {
				return layout.E.Layout(gtx, func(gtx C) D {
					return u.dialogButton(gtx, "wall:done", dialogButton{label: "Done", primary: true})
				})
			}),
		)
	})
}

// wallpaperPicked stores a photo the dialog asked for, when it arrives.
func (u *UI) wallpaperPicked(id string) {
	dlg := &u.wallpaperDlg
	select {
	case path := <-dlg.picked:
		dlg.picking = false
		if path != "" {
			if name, err := u.saveWallpaperImage(path); err != nil {
				u.toast("Couldn't use that picture: " + err.Error())
			} else {
				u.setWallpaperChoice(id, wallChoice{kind: wallImage, file: name})
			}
		}
	default:
	}
}

// wallpaperClicks reads the dialog's buttons for this frame.
func (u *UI) wallpaperClicks(gtx C, id string) {
	if id != "" && u.btn("wall:default").Clicked(gtx) {
		u.setWallpaperChoice(id, wallChoice{})
	}
	if u.btn("wall:doodle").Clicked(gtx) {
		u.setWallpaperChoice(id, wallChoice{kind: wallDoodle})
	}
	for _, hex := range wallSolids {
		if u.btn("wall:solid:" + hexColor(rgb(hex))).Clicked(gtx) {
			u.setWallpaperChoice(id, wallChoice{kind: wallSolid, color: rgb(hex)})
		}
	}
	if u.btn("wall:photo").Clicked(gtx) {
		u.pickWallpaperPhoto()
	}
	if u.btn("wall:done").Clicked(gtx) {
		u.closeDialog()
	}
}

// wallpaperRows are the dialog's choices: the default, the doodle, solid
// colours, a photo, and the dimming slider for a photo.
func (u *UI) wallpaperRows(gtx C, id string, cur wallChoice) []layout.FlexChild {
	var rows []layout.FlexChild
	opt := func(key, label string, on bool) layout.FlexChild {
		return layout.Rigid(func(gtx C) D {
			cl := u.btn(key)
			return clickable(gtx, cl, func(gtx C) D {
				return u.wallpaperOption(gtx, label, on)
			})
		})
	}
	if id != "" {
		rows = append(rows, opt("wall:default", "Use default wallpaper", cur.kind == wallDefault))
	}
	rows = append(rows, opt("wall:doodle", "Doodle wallpaper", cur.kind == wallDoodle))
	rows = append(rows, layout.Rigid(func(gtx C) D {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx C) D {
				return u.wallpaperOption(gtx, "Solid colour", cur.kind == wallSolid)
			}),
			layout.Rigid(layout.Spacer{Height: 8}.Layout),
			layout.Rigid(u.wallpaperSwatches),
			layout.Rigid(layout.Spacer{Height: 8}.Layout),
		)
	}))
	rows = append(rows, opt("wall:photo", "Photo", cur.kind == wallImage))
	if cur.kind == wallImage {
		rows = append(rows, layout.Rigid(layout.Spacer{Height: 12}.Layout))
		rows = append(rows, layout.Rigid(func(gtx C) D {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx C) D {
					return u.label(14.5, "Dimming · "+itoa(u.wallpaperDlg.dimVal)+"%", u.pal.TextSecondary).Layout(gtx)
				}),
				layout.Rigid(layout.Spacer{Height: 10}.Layout),
				layout.Rigid(func(gtx C) D {
					u.wallpaperDimSlider(gtx, gtx.Constraints.Max.X)
					return D{Size: image.Pt(gtx.Constraints.Max.X, gtx.Dp(24))}
				}),
			)
		}))
	}
	return rows
}
