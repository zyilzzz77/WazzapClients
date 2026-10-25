package ui

import (
	"image/color"

	"gioui.org/font"
	"gioui.org/text"
	"gioui.org/widget/material"
)

// Palette holds every color the UI uses. The dark values were sampled from
// WhatsApp Desktop (2025 design); the light ones follow the same structure.
type Palette struct {
	Frame, FrameText                             color.NRGBA // title bar and rail
	PanelBorder, Divider                         color.NRGBA
	Panel, ChatBg                                color.NRGBA
	Hover, Selected, Search                      color.NRGBA
	Chip, ChipBorder, ChipText                   color.NRGBA
	ChipActive, ChipActiveBorder, ChipActiveText color.NRGBA

	Text, TextSecondary, Icon, IconActive color.NRGBA
	IconStrong                            color.NRGBA // header, composer and menu buttons
	Green, OnGreen                        color.NRGBA
	RailActive, RailSeparator             color.NRGBA

	Doodle                       color.NRGBA
	BubbleIn, BubbleOut          color.NRGBA
	MetaIn, MetaOut, TickRead    color.NRGBA
	Link                         color.NRGBA // web links in messages
	TextOut, SecondaryOut        color.NRGBA
	QuoteIn, QuoteOut            color.NRGBA
	BubbleLine, BubbleButton     color.NRGBA // dividers and text of message buttons
	CodeBg                       color.NRGBA // behind `inline code`
	MentionPill                  color.NRGBA // behind mentions of you
	Selection                    color.NRGBA // behind selected text
	DateChip, DateChipText       color.NRGBA
	UnreadBand                   color.NRGBA // behind "N unread messages" (unread.go)
	Encryption, EncryptionText   color.NRGBA
	Composer, ComposerHint       color.NRGBA
	GroupAvatar, GroupAvatarIcon color.NRGBA
	UserAvatar, UserAvatarIcon   color.NRGBA
	// AvatarBgs and AvatarFgs color the initial of someone without a
	// picture, chosen by their ID; GeneralAvatar is behind a community's
	// General chat's speech bubble.
	AvatarBgs, AvatarFgs     []color.NRGBA
	GeneralAvatar            color.NRGBA
	Banner, BannerText, QRFg color.NRGBA
	Menu, MenuHover, Shadow  color.NRGBA
	CloseHover               color.NRGBA
	Senders                  []color.NRGBA

	EmptyIcon                        color.NRGBA // big glyphs of empty panes
	RowHover                         color.NRGBA // settings and info list rows
	Danger, DangerSoft               color.NRGBA // "Log out"; "Exit group" and friends
	RingViewed                       color.NRGBA // status ring once seen
	Verified                         color.NRGBA
	AnnounceBg, AnnounceIcon         color.NRGBA // community announcements tile
	ChannelAvatar, ChannelAvatarIcon color.NRGBA // channel without a picture
	StatusBg                         color.NRGBA // status viewer backdrop

	Popup, PopupBorder, PopupHover color.NRGBA // context menus and the reaction bar
	PopupDivider, PopupSub         color.NRGBA
	Picker, PickerTab              color.NRGBA // emoji picker panel; its active tab
	PickerTabBorder                color.NRGBA
	Viewer, ViewerDivider          color.NRGBA // media viewer
	ViewerThumb, ViewerArrow       color.NRGBA
	Scrim                          color.NRGBA // behind dialogs
	Dialog                         color.NRGBA
	Toast, ToastText               color.NRGBA
}

// markColors are what the photo editor draws with, the same in both
// themes since they end up in the photo.
var markColors = rgbs(0xffffff, 0x000000, 0xff3b30, 0xff9500, 0xffcc00, 0x34c759, 0x007aff, 0xaf52de)

func rgb(c uint32) color.NRGBA {
	return color.NRGBA{R: uint8(c >> 16), G: uint8(c >> 8), B: uint8(c), A: 0xff}
}

func argb(c uint32, a uint8) color.NRGBA {
	col := rgb(c)
	col.A = a
	return col
}

func rgbs(cs ...uint32) []color.NRGBA {
	out := make([]color.NRGBA, len(cs))
	for i, c := range cs {
		out[i] = rgb(c)
	}
	return out
}

var darkPalette = Palette{
	Frame: rgb(0x111b21), FrameText: rgb(0xfafafa),
	PanelBorder: rgb(0x222d34), Divider: rgb(0x222d34),
	Panel: rgb(0x111b21), ChatBg: rgb(0x0b141a),
	Hover: rgb(0x2a3942), Selected: rgb(0x2a3942), Search: rgb(0x202c33),
	Chip: rgb(0x111b21), ChipBorder: rgb(0x2a3942), ChipText: rgb(0xa5a7a9),
	ChipActive: rgb(0x1a342b), ChipActiveBorder: rgb(0x2c4a3d), ChipActiveText: rgb(0xd9fdd3),

	Text: rgb(0xfafafa), TextSecondary: rgb(0x8696a0), Icon: rgb(0x8696a0), IconActive: rgb(0xe9edef),
	IconStrong: rgb(0xffffff),
	Green:      rgb(0x00a884), OnGreen: rgb(0xffffff),
	RailActive: rgb(0x2a3942), RailSeparator: rgb(0x1c2b33),

	Doodle:   rgb(0x1c2b33),
	BubbleIn: rgb(0x202c33), BubbleOut: rgb(0x005c4b),
	MetaIn: rgb(0x8696a0), MetaOut: rgb(0xa8c5b6), TickRead: rgb(0x53bdeb),
	Link:    rgb(0x00a884),
	TextOut: rgb(0xfafafa), SecondaryOut: rgb(0xa8c5b6),
	QuoteIn: rgb(0x111b21), QuoteOut: rgb(0x00473c),
	DateChip: rgb(0x182229), DateChipText: rgb(0x8696a0),
	UnreadBand: argb(0x000000, 0x4c),
	Encryption: rgb(0x182229), EncryptionText: rgb(0xe3c77b),
	Composer: rgb(0x202c33), ComposerHint: rgb(0x8696a0),
	GroupAvatar: rgb(0x102540), GroupAvatarIcon: rgb(0x70adff),
	UserAvatar: rgb(0x202c33), UserAvatarIcon: rgb(0x8696a0),
	AvatarBgs:     rgbs(0x242346, 0x102540, 0x0f3330, 0x16321d, 0x3a2e10, 0x3d2414, 0x3d1a2b),
	AvatarFgs:     rgbs(0xa598f3, 0x70adff, 0x5fcfbf, 0x6fd38a, 0xefc152, 0xf5a172, 0xf18db5),
	GeneralAvatar: rgb(0xd1d8da),
	Banner:        rgb(0x202c33), BannerText: rgb(0xd0d2d2), QRFg: rgb(0x122e31),
	Menu: rgb(0x1f2c34), MenuHover: rgb(0x2a3942), Shadow: argb(0x000000, 0x60),
	CloseHover: rgb(0xc42b1c),
	Senders:    rgbs(0xcca48f, 0x8fb8e8, 0xe6a1b8, 0x86c9a8, 0xdcc27a, 0xb1a3e6, 0xe89b7f, 0x7fc3d6, 0xc7b7a0),

	EmptyIcon: rgb(0x3b4a54), RowHover: rgb(0x1f2c34),
	Danger: rgb(0xe85d65), DangerSoft: rgb(0xed9ea5),
	RingViewed: rgb(0x2a3942), Verified: rgb(0x3c79e6),
	AnnounceBg: rgb(0x342c21), AnnounceIcon: rgb(0xf6d78b),
	ChannelAvatar: rgb(0x32281e), ChannelAvatarIcon: rgb(0xd3a887),
	StatusBg: rgb(0x0b141a),

	Popup: rgb(0x111b21), PopupBorder: rgb(0x222d34), PopupHover: rgb(0x202c33),
	PopupDivider: rgb(0x222d34), PopupSub: rgb(0x8696a0),
	Picker: rgb(0x111b21), PickerTab: rgb(0x202c33), PickerTabBorder: rgb(0x2a3942),
	Viewer: rgb(0x0b141a), ViewerDivider: rgb(0x1f2c34),
	ViewerThumb: rgb(0x202c33), ViewerArrow: rgb(0x111b21),
	Scrim: argb(0x000000, 0x99), Dialog: rgb(0x202c33),
	Toast: rgb(0x2a3942), ToastText: rgb(0xfafafa),
	BubbleLine: argb(0xffffff, 0x14), BubbleButton: rgb(0x53bdeb),
	CodeBg:      argb(0xffffff, 0x12),
	MentionPill: argb(0x00a884, 0x24),
	Selection:   rgb(0x1e3a9e),
}

var lightPalette = Palette{
	Frame: rgb(0xf0f2f5), FrameText: rgb(0x0a0a0a),
	PanelBorder: rgb(0xe3e0dc), Divider: rgb(0xe9edef),
	Panel: rgb(0xffffff), ChatBg: rgb(0xefeae2),
	Hover: rgb(0xf5f6f6), Selected: rgb(0xf0f2f5), Search: rgb(0xf6f5f4),
	Chip: rgb(0xffffff), ChipBorder: rgb(0xe3e0dc), ChipText: rgb(0x5e6468),
	ChipActive: rgb(0xd9fdd3), ChipActiveBorder: rgb(0xc3eebc), ChipActiveText: rgb(0x15603e),

	Text: rgb(0x0a0a0a), TextSecondary: rgb(0x667781), Icon: rgb(0x54656f), IconActive: rgb(0x0a0a0a),
	IconStrong: rgb(0x0a0a0a),
	Green:      rgb(0x00a884), OnGreen: rgb(0xffffff),
	RailActive: rgb(0xe8e6e3), RailSeparator: rgb(0xe3e0dc),

	Doodle:   rgb(0xe8e1d8),
	BubbleIn: rgb(0xffffff), BubbleOut: rgb(0xd9fdd3),
	MetaIn: rgb(0x667781), MetaOut: rgb(0x5b7a66), TickRead: rgb(0x53bdeb),
	Link:    rgb(0x027eb5),
	TextOut: rgb(0x0a0a0a), SecondaryOut: rgb(0x587a64),
	QuoteIn: rgb(0xf5f6f6), QuoteOut: rgb(0xd1f4cc),
	DateChip: rgb(0xffffff), DateChipText: rgb(0x54656f),
	UnreadBand: argb(0xffffff, 0x40),
	Encryption: rgb(0xffeecd), EncryptionText: rgb(0x54656f),
	Composer: rgb(0xffffff), ComposerHint: rgb(0x667781),
	GroupAvatar: rgb(0xdbe7fb), GroupAvatarIcon: rgb(0x3778e5),
	UserAvatar: rgb(0xe8ebed), UserAvatarIcon: rgb(0x8a9499),
	AvatarBgs:     rgbs(0xe8e5fc, 0xdbe7fb, 0xd5f2ee, 0xd9f2df, 0xf8edcf, 0xfbe3d4, 0xfadbe8),
	AvatarFgs:     rgbs(0x5e4fc8, 0x3778e5, 0x1f8a7d, 0x2b8a45, 0x9a6c00, 0xb3561f, 0xb43d72),
	GeneralAvatar: rgb(0xdfe5e7),
	Banner:        rgb(0xfff4c5), BannerText: rgb(0x54656f), QRFg: rgb(0x122e31),
	Menu: rgb(0xffffff), MenuHover: rgb(0xf5f6f6), Shadow: argb(0x0b141a, 0x30),
	CloseHover: rgb(0xc42b1c),
	Senders:    rgbs(0x1f7aec, 0xe542a3, 0x02a698, 0xc85a00, 0x7f66ff, 0xd62f45, 0x029d00, 0x0e8a94, 0xa4661f),

	EmptyIcon: rgb(0xc4c9cc), RowHover: rgb(0xf5f6f6),
	Danger: rgb(0xea0038), DangerSoft: rgb(0xd4314a),
	RingViewed: rgb(0xc3c9cc), Verified: rgb(0x2a7de1),
	AnnounceBg: rgb(0xfdf1d8), AnnounceIcon: rgb(0xc58a13),
	ChannelAvatar: rgb(0xf6e7da), ChannelAvatarIcon: rgb(0xa86f45),
	StatusBg: rgb(0x0f0f0f),

	Popup: rgb(0xffffff), PopupBorder: rgb(0xe9edef), PopupHover: rgb(0xf5f6f6),
	PopupDivider: rgb(0xe9edef), PopupSub: rgb(0x667781),
	Picker: rgb(0xffffff), PickerTab: rgb(0xf0f2f5), PickerTabBorder: rgb(0xe3e0dc),
	Viewer: rgb(0xffffff), ViewerDivider: rgb(0xe9edef),
	ViewerThumb: rgb(0xf0f2f5), ViewerArrow: rgb(0xf0f2f5),
	Scrim: argb(0xffffff, 0xb0), Dialog: rgb(0xffffff),
	Toast: rgb(0x3b4a54), ToastText: rgb(0xffffff),
	BubbleLine: argb(0x000000, 0x14), BubbleButton: rgb(0x027eb5),
	CodeBg:      argb(0x000000, 0x0f),
	MentionPill: argb(0x00a884, 0x22),
	Selection:   rgb(0xb3d4fc),
}

func hashIndex(s string, n int) int {
	var h uint32 = 2166136261
	for i := 0; i < len(s); i++ {
		h ^= uint32(s[i])
		h *= 16777619
	}
	return int(h % uint32(n))
}

// Typeface prefers Segoe UI (what WhatsApp Desktop uses on Windows), color
// emoji from Noto, then common system UI fonts and the bundled Go fonts.
//
// MS Gothic covers Japanese and most Chinese text in a single file for
// every weight. Without it the fallback picks Yu Gothic, whose weights are
// separate files that each cost ~40 MB of heap once loaded (the whole
// glyph table is parsed), so a bold name plus a regular message would load
// two of them.
const typeface font.Typeface = "Segoe UI, " + emojiTypeface + ", Segoe UI Symbol, MS Gothic, Cambria Math, Helvetica Neue, Roboto, Noto Sans, sans-serif, Go"

// newShaper returns a shaper of the bundled fonts and the system's.
func newShaper() *text.Shaper { return text.NewShaper(text.WithCollection(bundledFonts())) }

func newTheme() *material.Theme {
	th := material.NewTheme()
	// System fonts are enabled by default and cover other scripts.
	th.Shaper = newShaper()
	th.Face = typeface
	th.TextSize = 14
	return th
}
