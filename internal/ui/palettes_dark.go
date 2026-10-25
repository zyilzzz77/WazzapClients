package ui

// Dark theme variants. Each starts from darkPalette and overrides the
// colors that make it its own, so every Palette field stays set.

// oledPalette is the dark theme on true black: every background is
// #000000, and bubbles and rows are lifted just enough to stay visible
// on it.
var oledPalette = func() Palette {
	p := darkPalette
	black := rgb(0x000000)
	p.Frame, p.Panel, p.ChatBg = black, black, black
	p.Viewer, p.Dialog, p.Popup, p.Picker = black, black, black, black
	p.Menu, p.Search, p.Chip = black, rgb(0x161616), black
	p.Hover, p.Selected = rgb(0x1c1c1c), rgb(0x1c1c1c)
	p.RowHover, p.PopupHover, p.MenuHover = rgb(0x141414), rgb(0x1c1c1c), rgb(0x1c1c1c)
	p.BubbleIn, p.Composer = rgb(0x161616), rgb(0x161616)
	p.Doodle, p.DateChip, p.Encryption = rgb(0x1a1a1a), rgb(0x161616), rgb(0x161616)
	p.PanelBorder, p.Divider = rgb(0x1f1f1f), rgb(0x1f1f1f)
	p.ChipBorder, p.PopupBorder = rgb(0x262626), rgb(0x262626)
	p.ViewerDivider, p.PickerTabBorder = rgb(0x1f1f1f), rgb(0x262626)
	p.PickerTab, p.ViewerThumb = rgb(0x161616), rgb(0x161616)
	p.ViewerArrow, p.StatusBg = black, black
	p.Toast, p.Banner = rgb(0x1c1c1c), rgb(0x1c1c1c)
	p.RingViewed, p.EmptyIcon = rgb(0x2a2a2a), rgb(0x454545)
	p.QuoteIn, p.QuoteOut = rgb(0x161616), rgb(0x00473c)
	p.UserAvatar = rgb(0x161616)
	p.RailActive, p.RailSeparator = rgb(0x1c1c1c), rgb(0x1f1f1f)
	return p
}()

// navyPalette is the dark theme in deep blue.
var navyPalette = func() Palette {
	p := darkPalette
	p.Frame, p.Panel, p.ChatBg = rgb(0x0a1622), rgb(0x0e1a2b), rgb(0x0a1622)
	p.Composer, p.BubbleIn = rgb(0x16283c), rgb(0x16283c)
	p.BubbleOut = rgb(0x14506e)
	p.QuoteOut, p.MetaOut, p.SecondaryOut = rgb(0x0f3d55), rgb(0x9fc3d8), rgb(0x9fc3d8)
	p.Text, p.TextSecondary = rgb(0xeaf2f8), rgb(0x9fb4c8)
	p.Icon, p.IconActive = rgb(0x9fb4c8), rgb(0xeaf2f8)
	p.Link = rgb(0x6fc3e8)
	p.TextOut = rgb(0xeaf2f8)
	p.Doodle, p.DateChip, p.Encryption = rgb(0x14273a), rgb(0x0e1a2b), rgb(0x0e1a2b)
	p.DateChipText = rgb(0x9fb4c8)
	p.Hover, p.Selected, p.Search = rgb(0x1a2c40), rgb(0x1a2c40), rgb(0x16283c)
	p.PanelBorder, p.Divider = rgb(0x1c3049), rgb(0x1a2c40)
	p.Chip, p.ChipBorder, p.ChipText = rgb(0x0e1a2b), rgb(0x1c3049), rgb(0x9fb4c8)
	p.Menu, p.MenuHover = rgb(0x16283c), rgb(0x1d3349)
	p.Popup, p.PopupBorder, p.PopupHover = rgb(0x0e1a2b), rgb(0x1c3049), rgb(0x16283c)
	p.PopupDivider = rgb(0x1a2c40)
	p.Picker, p.PickerTab, p.PickerTabBorder = rgb(0x0e1a2b), rgb(0x16283c), rgb(0x1c3049)
	p.Viewer, p.ViewerDivider = rgb(0x0a1622), rgb(0x16283c)
	p.ViewerThumb, p.ViewerArrow = rgb(0x16283c), rgb(0x0a1622)
	p.Dialog, p.Toast = rgb(0x0e1a2b), rgb(0x1d3349)
	p.ToastText = rgb(0xeaf2f8)
	p.Banner, p.BannerText = rgb(0x1a2c40), rgb(0xd0d2d2)
	p.RowHover, p.EmptyIcon = rgb(0x101f30), rgb(0x33506b)
	p.RingViewed = rgb(0x33506b)
	p.AnnounceBg, p.ChannelAvatar = rgb(0x1a2c40), rgb(0x16283c)
	p.StatusBg = rgb(0x0a1622)
	p.UserAvatar, p.UserAvatarIcon = rgb(0x16283c), rgb(0x9fb4c8)
	p.RailActive, p.RailSeparator = rgb(0x1a2c40), rgb(0x1c3049)
	p.FrameText = rgb(0xeaf2f8)
	return p
}()

// forestPalette is the dark theme in deep green.
var forestPalette = func() Palette {
	p := darkPalette
	p.Frame, p.Panel, p.ChatBg = rgb(0x101a12), rgb(0x14231a), rgb(0x0e1c13)
	p.Composer, p.BubbleIn = rgb(0x1c2f23), rgb(0x1c2f23)
	p.BubbleOut = rgb(0x205c33)
	p.QuoteOut, p.MetaOut, p.SecondaryOut = rgb(0x1a4230), rgb(0xa9c8b2), rgb(0xa9c8b2)
	p.Text, p.TextSecondary = rgb(0xf2f7f2), rgb(0xa8bfa9)
	p.Icon, p.IconActive = rgb(0xa8bfa9), rgb(0xf2f7f2)
	p.Link = rgb(0x6fd38a)
	p.TextOut = rgb(0xf2f7f2)
	p.Doodle, p.DateChip, p.Encryption = rgb(0x182a1f), rgb(0x14231a), rgb(0x14231a)
	p.DateChipText = rgb(0xa8bfa9)
	p.Hover, p.Selected, p.Search = rgb(0x223626), rgb(0x223626), rgb(0x1c2f23)
	p.PanelBorder, p.Divider = rgb(0x24382b), rgb(0x1f2f24)
	p.Chip, p.ChipBorder, p.ChipText = rgb(0x14231a), rgb(0x24382b), rgb(0xa8bfa9)
	p.Menu, p.MenuHover = rgb(0x1c2f23), rgb(0x223626)
	p.Popup, p.PopupBorder, p.PopupHover = rgb(0x14231a), rgb(0x24382b), rgb(0x1c2f23)
	p.PopupDivider = rgb(0x1f2f24)
	p.Picker, p.PickerTab, p.PickerTabBorder = rgb(0x101a12), rgb(0x1c2f23), rgb(0x24382b)
	p.Viewer, p.ViewerDivider = rgb(0x101a12), rgb(0x1c2f23)
	p.ViewerThumb, p.ViewerArrow = rgb(0x1c2f23), rgb(0x0e1c13)
	p.Dialog, p.Toast = rgb(0x14231a), rgb(0x223626)
	p.ToastText = rgb(0xf2f7f2)
	p.Banner, p.BannerText = rgb(0x223626), rgb(0xd0d2d2)
	p.RowHover, p.EmptyIcon = rgb(0x16241b), rgb(0x3a4a3d)
	p.RingViewed = rgb(0x3a4a3d)
	p.AnnounceBg, p.ChannelAvatar = rgb(0x24382b), rgb(0x1c2f23)
	p.StatusBg = rgb(0x0e1c13)
	p.UserAvatar, p.UserAvatarIcon = rgb(0x1c2f23), rgb(0xa8bfa9)
	p.RailActive, p.RailSeparator = rgb(0x223626), rgb(0x24382b)
	p.FrameText = rgb(0xf2f7f2)
	p.QuoteIn = rgb(0x101a12)
	return p
}()
