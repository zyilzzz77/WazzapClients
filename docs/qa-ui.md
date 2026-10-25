# Manual QA — Phase 1 / 1b / 1c UI upgrade

Run the app with a **secondary number** (a test account), never your main
number. Build it first:

```
GOOS=windows go build ./...
go run ./cmd/wazzap -demo   # UI-only smoke, no real account
```

Automated coverage: `go test ./internal/ui/ ./internal/wa/` — palette
completeness and contrast, theme resolution, density, wallpaper store,
view-once save gating, HD status quality, shortcuts.

## 1. Themes

1. Settings > Chats > Theme. The page has two sections: "Default for all
   accounts" and "This account".
2. Pick each theme in turn: System default, Light, Dark, Dark OLED, Navy,
   Forest, Sepia, WhatsApp Green. The whole window must recolour instantly,
   no restart, no flicker of the old palette.
3. Dark OLED: the chat background, rail, list and dialogs must be true
   black (#000000) — on an OLED screen the pixels are off. Bubbles stay
   readable (incoming #161616, outgoing #005c4b).
4. Navy: dark blue panes (#0a1622 / #0e1a2b), outgoing bubbles #14506e.
5. Per-account override: set the global default to Dark, then set "This
   account" to Sepia on one account. Switch to another account: it shows
   Dark. Switch back: Sepia again. Restart the app: both stick.
6. "Use global default" in the per-account section clears the override.
7. Check every theme in a chat with text, a link, a quote, a date chip, the
   composer, the emoji picker and a dialog — no unreadable text anywhere.

## 2. WhatsApp Web look and density

1. Compare with web.whatsapp.com: dark panes #111b21 / #202c33, chat
   background #0b141a, outgoing bubble #005c4b, green #00a884. Light:
   frame #f0f2f5, chat background #efeae2, outgoing bubble #d9fdd3.
2. Settings > Chats > Density: Comfortable, then Compact. The chat list
   rows get shorter (about 15% less tall) and tighter; avatars and badges
   stay aligned. Comfortable restores the old spacing.
3. The 3-pane layout (rail / chat list / conversation) is unchanged.
4. Density persists after restart.

## 3. Custom wallpaper

1. Open a chat > ⋮ menu > Wallpaper. The dialog offers: Use default
   wallpaper, Doodle wallpaper, Solid colour (6 swatches), Photo, and a
   Dimming slider when a photo is chosen.
2. Pick a solid colour: the chat background changes at once; other chats
   keep theirs. Pick "Use default wallpaper": the chat follows the global
   default again.
3. Settings > Chats > Wallpaper sets the global default: pick a solid
   colour there and open a chat with no own wallpaper — it shows the
   global one.
4. Photo: pick a 12 MP photo from the phone. It is stored downsized
   (long side at most 1920 px) in the account's `wallpaper/` folder as
   `wallpaper-<hash>.jpg`, mode 0600. The chat shows it cover-cropped.
5. Dimming: drag the slider to 0% and to 100%. At 0% a busy photo may
   fight the bubbles; at 100% the background is flat. Around the default
   30% the bubbles, date chip and day separator stay readable in every
   theme (Light, Dark, Dark OLED, Navy, Forest, Sepia, WhatsApp Green).
6. A file over 32 MB is refused with a toast; a non-picture file is
   refused too.
7. Restart the app: per-chat and global wallpapers are still there.
8. Replace a chat's photo wallpaper with another photo: the old file is
   deleted from `wallpaper/`.

## 4. View-once save (opt-in)

1. Settings > Privacy > View once: "Allow saving view-once media" is OFF
   by default. Open a view-once photo or video from the secondary number:
   the viewer shows only the close button, exactly as before.
2. Turn the setting ON. Open a view-once photo: the viewer now shows a
   Save button (download icon) next to close.
3. Tap Save: a dialog asks "Save this view-once photo/video?" and warns
   that the sender expected it to disappear. Confirm: the file lands in
   the account's `media/viewonce/` folder. Cancel: nothing is written.
4. Turn the setting OFF again: the Save button is gone, behaviour is
   exactly upstream.
5. Never auto-saves: with the setting ON, merely opening a view-once
   message writes no file.
6. On Windows, screenshots stay blocked while a view-once item is shown
   (upstream behaviour, unchanged).

## 5. HD status

1. Open Status, attach a photo. The send view's HD button (top bar) shows
   the quality menu: Standard (1600 px, ~116 kB estimate) and HD (4096 px,
   larger estimate). The estimate updates with the photo.
2. Post at Standard, then post the same photo at HD from the secondary
   number; view both on the phone. The HD one is visibly sharper on a
   detailed photo.
3. Status posts always start at Standard, even if the last chat photo was
   sent at HD.
4. Attach a video as a status: the top bar shows "clip.mp4 · 1.2 MB" and
   the quality menu notes "1 video (1.2 MB) is sent as it is." — videos
   have no quality setting and are sent unchanged.
5. The HD toggle for photos in chat is unchanged (reuse, not a new path).

## 6. Keyboard shortcuts

1. Ctrl+K: focuses the chat-list search from anywhere (even from the
   Status page, which jumps back to Chats first).
2. Ctrl+N: opens New chat. Esc closes it.
3. Ctrl+1 … Ctrl+9: switches to that account (with two accounts linked,
   Ctrl+1 and Ctrl+2 switch between them; with one account, nothing
   happens).
4. Ctrl+Shift+M: mutes, then unmutes, the open chat. While typing in the
   composer, Ctrl+Shift+M still inserts monospace formatting (the
   composer's shortcut wins while it has focus).
5. Esc: closes an open dialog / viewer / picker first; with none open and
   a list search typed, clears the search.
6. Settings > Shortcuts > Everywhere lists all five with their keys.

## 7. Notifications and tray

1. Link a second account, set it to background mode, and have someone
   message it while the first account is open. The notification reads
   "Sender · Account name".
2. With one account linked, notifications show no account label (as
   before).
3. The tray tooltip shows the total unread count across all accounts.
