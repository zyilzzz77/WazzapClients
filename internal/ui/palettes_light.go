package ui

// Light theme variants. Each starts from lightPalette and overrides the
// colors that make it its own, so every Palette field stays set.

// sepiaPalette is the light theme on warm paper.
var sepiaPalette = func() Palette {
	p := lightPalette
	p.Frame, p.Panel, p.ChatBg = rgb(0xf5eeda), rgb(0xfdf6e3), rgb(0xf3ead3)
	p.Composer, p.BubbleIn = rgb(0xfdf6e3), rgb(0xfffdf5)
	p.Doodle = rgb(0xe6dcc3)
	p.Search, p.Hover, p.Selected = rgb(0xf5eeda), rgb(0xf5eeda), rgb(0xf0e8d2)
	p.PanelBorder, p.Divider = rgb(0xe3d9bd), rgb(0xe9dfc5)
	p.Chip, p.ChipBorder, p.ChipText = rgb(0xfdf6e3), rgb(0xe3d9bd), rgb(0x6b5f45)
	p.Text, p.TextSecondary = rgb(0x2e2413), rgb(0x7a6c4f)
	p.Icon, p.IconActive, p.IconStrong = rgb(0x6b5f45), rgb(0x2e2413), rgb(0x2e2413)
	p.FrameText, p.TextOut = rgb(0x2e2413), rgb(0x2e2413)
	p.MetaIn, p.DateChipText = rgb(0x7a6c4f), rgb(0x6b5f45)
	p.DateChip, p.Encryption = rgb(0xfdf6e3), rgb(0xffeecd)
	p.EncryptionText = rgb(0x6b5f45)
	p.QuoteIn = rgb(0xf5eeda)
	p.GroupAvatar, p.GroupAvatarIcon = rgb(0xe9dfc0), rgb(0x8a7a4d)
	p.UserAvatar, p.UserAvatarIcon = rgb(0xefe7cf), rgb(0x8a7a4d)
	p.BannerText = rgb(0x6b5f45)
	p.Menu, p.MenuHover = rgb(0xfdf6e3), rgb(0xf5eeda)
	p.EmptyIcon, p.RowHover = rgb(0xc9bfa0), rgb(0xf5eeda)
	p.RingViewed = rgb(0xd8cdb2)
	p.Popup, p.PopupBorder, p.PopupHover = rgb(0xfdf6e3), rgb(0xe9dfc5), rgb(0xf5eeda)
	p.PopupDivider, p.PopupSub = rgb(0xe9dfc5), rgb(0x7a6c4f)
	p.Picker, p.PickerTab, p.PickerTabBorder = rgb(0xfdf6e3), rgb(0xf0e8d2), rgb(0xe3d9bd)
	p.Viewer, p.ViewerDivider = rgb(0xfdf6e3), rgb(0xe9dfc5)
	p.ViewerThumb, p.ViewerArrow = rgb(0xf0e8d2), rgb(0xf0e8d2)
	p.Dialog = rgb(0xfdf6e3)
	p.RailActive, p.RailSeparator = rgb(0xefe7cf), rgb(0xe3d9bd)
	return p
}()

// waGreenPalette is the saturated light variant: WhatsApp's green accent
// and bubbles on a pale green frame.
var waGreenPalette = func() Palette {
	p := lightPalette
	p.Frame, p.ChatBg = rgb(0xd8f0e2), rgb(0xe6f4ea)
	p.Green, p.Link, p.BubbleButton = rgb(0x00a884), rgb(0x008069), rgb(0x008069)
	p.Doodle = rgb(0xcfe8d8)
	p.Hover, p.Selected, p.Search = rgb(0xe6f4ea), rgb(0xdcf3e4), rgb(0xe6f4ea)
	p.PanelBorder, p.Divider = rgb(0xcfe0d4), rgb(0xd8e8dc)
	p.ChipBorder = rgb(0xcfe0d4)
	p.ChipActive, p.ChipActiveText = rgb(0xd9fdd3), rgb(0x075e54)
	p.MetaOut, p.SecondaryOut = rgb(0x4a7a5e), rgb(0x4a7a5e)
	p.QuoteOut = rgb(0xc8efd9)
	p.GroupAvatar, p.GroupAvatarIcon = rgb(0xd5f2e4), rgb(0x008069)
	p.UserAvatar, p.UserAvatarIcon = rgb(0xdcf3e4), rgb(0x5e8a74)
	p.MenuHover = rgb(0xe6f4ea)
	p.EmptyIcon, p.RowHover = rgb(0xa8cbb4), rgb(0xe6f4ea)
	p.RingViewed = rgb(0xb8d8c4)
	p.AnnounceBg, p.AnnounceIcon = rgb(0xe6f4ea), rgb(0x008069)
	p.ChannelAvatar, p.ChannelAvatarIcon = rgb(0xdcf3e4), rgb(0x008069)
	p.PopupBorder, p.PopupHover = rgb(0xd8e8dc), rgb(0xe6f4ea)
	p.PopupDivider = rgb(0xd8e8dc)
	p.PickerTab, p.PickerTabBorder = rgb(0xe6f4ea), rgb(0xcfe0d4)
	p.ViewerDivider = rgb(0xd8e8dc)
	p.ViewerThumb, p.ViewerArrow = rgb(0xe6f4ea), rgb(0xe6f4ea)
	p.MentionPill = argb(0x00a884, 0x22)
	p.RailActive, p.RailSeparator = rgb(0xdcf3e4), rgb(0xcfe0d4)
	return p
}()
