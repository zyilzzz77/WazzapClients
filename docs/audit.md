# Phase 0 audit — WazzapClients fork (`feat/ui-ai-multiaccount`)

- Base: `cde5a34` ("Bring the Status page closer to WhatsApp and add status audiences"), branch `feat/ui-ai-multiaccount`.
- Stack verified in `go.mod`: Go `1.27.0`, `gioui.org v0.10.3` (replaced by `./third_party/gio`, patched via `patches/apply.sh`),
`github.com/polymorfa/hypermeow v0.0.0-20261010085506-b0e8e85a4075` (NOT upstream whatsmeow).
- Sources: upstream `AGENTS.md` (741 lines, read in full), master spec `~/workspace/AGENTS.md` §§0–8, 3b, 3c, and the code itself.
Every function/type name below was grepped from the tree. `go vet` on the audited packages passes (see §9). No `.go` file was
changed. New requirements (2026-10-25, §§3b–3c): WhatsApp Web look, themes incl. Dark OLED + Navy, custom wallpaper, view-once
saving (opt-in), HD status.

## 1. Account switching today

### 1.1 The registry — `internal/accounts/accounts.go` (208 lines)

- `Account{Dir, ID, Name, Phone, Background Mode, Checked time.Time}`. `Dir` is slash-separated, relative to the root data dir;
  `""` = the root itself (the first/single account, so a pre-accounts install keeps its data). Later accounts live in
  `accounts/<n>` (`List.Add` picks the first free `n`, starting at 2, and `os.RemoveAll`s leftovers in that dir first). `Linked()`
  = `ID != ""`; `Label()` = name or phone.
- `Mode int`: `Off = 0`, `Always = -1`, periodic checks are positive minutes; `Checks = []Mode{5, 15, 30, 60}`. `Mode.Every()`
  returns 0 for `<= 0`, else minutes. `Mode.String()`: "Only while open" / "Always connected" / "Check every N minutes" / "Check
  every hour". `Account.InBackground()` = linked && mode != Off.
- `List{root string (unexported), Active string, Accounts []Account}` persisted as `accounts.json` (0600, atomic tmp+rename in
  `Save()`; `Load()` synthesises one `Account{Dir: ""}` when the file is missing). `Find(dir)`, `Current()` (= `Find(Active)`),
  `Others()` (linked, not active), `Path(dir)` (root-joined), `PicturePath(dir)` (`<dir>/account.jpg`, the switcher avatar of a
  non-open account).
- `Remove(dir)`: deletes the list entry and `os.RemoveAll(Path(dir))` — **except** `dir == ""`, where only the picture file is
  removed ("it holds the other accounts too, and logging out already wiped its session and messages"). If the list empties it is
  re-seeded with `{Dir: ""}`; `Active` falls back to `Accounts[0]`. There is **no rename and no reorder** API — Phase 2 needs both
  added here.

### 1.2 The switch path — `internal/ui/accounts.go` (852 lines) + `internal/ui/host.go`

Entry points: `host.handle` (`host.go:408`) maps `reqSwitch` → `switchAccount(r.dir)` and `reqAddAccount` → `addAccount()`;
requests are queued non-blockingly by `host.request` (a flood drops extras) and served on the UI goroutine. The switcher UI calls
them via `updateAccountMenu` (`accounts.go:421`): row click → `request{kind: reqSwitch, dir}`, "Add account" click →
`reqAddAccount`; each row's mode button opens the `ctxAcctMode` menu (`acctModeItems`, `setAccountMode` →
`host.setBackgroundMode`).

- `host.addAccount()` (`accounts.go:151`): `List.Add()` then `switchAccount(newDir)`; the new account opens unlinked, so the
  window shows the QR/login screen for it.
- `host.switchAccount(dir)` (`accounts.go:167`): no-op (refresh only) if `dir == Active`; `takeBackground(dir)` first — if the
  target runs in the background its live backend is taken over (waiting on `bgOpen` while it is still `opening`), so it is **not**
  opened twice. Otherwise `awaitClosed(dir)` then `h.o.Open(l.Path(dir))` (failure toasts "Couldn't open the account"; a
  just-added unlinked account is removed again; a taken background backend is put back and `armBg()` resumes it). Then
  `closeAccount(b)`, `l.Active = dir`, `saveAccounts()`, `useBackend(b, bg)`.
- `host.closeAccount(next)` (`accounts.go:232`): (a) copies **app prefs only** (`appPrefs`, `accounts.go:30-34`: theme, doodles,
  enter-send, background, list width/hidden, zoom, privacy, volume, voice rate, emoji style, klipy, auto-update, app lock,
  `perfPrefs…`) onto `next` via `SetPref` — account prefs (incl. KLIPY key, slash toggles, gray-command switches) stay in each
  account's own `wazzap.db`, so the KLIPY key is **per-account today** (see §4); (b) saves the open account's avatar JPEG to
  `PicturePath(cur.Dir)` (0600); (c) `u.stopOutgoingTyping()` + `u.shutdown()` (stops voice, discards recording, hides
  viewer/status viewer, stops players); (d) if linked and `Background == Always` → `toBackground(cur.Dir)` and returns **without
  closing**; else `h.notes.clear()`, captures `auto.Backend.Next()`, `h.b.Close()`; unlinked/logged-out accounts are `l.Remove`d;
  a `Checks` account becomes a `bgAccount{dir, nextJob}` with `Checked = now` and `scheduleCheck(bg, now, mode)`. Also the logout
  path: `logout()`/`leave(dir)` set `h.leaving, h.leaveTo`, call `b.Logout()`; `accountState` (on the logged-out `ConnEvent`)
  queues `reqSwitch{leaveTo}`.
- `host.toBackground(dir)` (`accounts.go:290`): moves the live backend + notifier + `conn` into `h.bg[dir]` as `bgAccount{b,
  notes, conn, sending: map[string]bool{}}`, calls `SetBackground(true)` (presence "unavailable", §2) and
  `backgroundNotes`/`armBg()`.
- `host.useBackend(b, bg)` (`accounts.go:303`): wraps with `withAuto(b)` (`host.go:708`); `loadPerf(b)`; resets
  `conn/syncPct/queue/later`; a background backend keeps `bg.conn`, else shows `StateConnecting` with the stored name/ID so chats
  appear at once; **drafts: `h.u.dropAttachments()` then `clearDrafts(h.drafts)` — all drafts are discarded on switch** (see
  below); notifier is the background account's own if present, else a fresh `newNotifier(b, Active)`; `SetBackground(false)`;
  `dropCaches()` + `h.newUI()` (a brand-new `UI`); `b.Start(h.poke)` only if not already running; `updateTooltip()`;
  `memtrim.Trim()`.

### 1.3 What happens to the open backend, drafts, notifications, background accounts

| Thing | On switch |
|---|---|
| Open `wa.Backend` | `Close()` (disconnect + `db.Close()` + `logf.Close()`, waits for `work` WaitGroup) — **unless** mode is `Always`, in which case it is moved whole to `h.bg` via `toBackground` and keeps running. A `Checks` account is closed and re-opened later by `openBg` on a goroutine. |
| Drafts | **Lost.** `internal/ui/draft.go:122-129` `clearDrafts` "forgets every draft (another account opens)"; `host.drafts` is a single `map[string]*chatDraft` shared by all accounts (host struct, `host.go:219ff`), keyed by chat ID only — no account dimension, no persistence across restarts either ("They live as long as the app… not across restarts"). Send-view files are dropped via `u.dropAttachments()`. Phase 2 must key drafts by account+chat and persist them (or at least keep them in memory per account). |
| Scroll position / open chat | **Lost.** `newUI()` (`host.go:582`) builds a fresh `UI` from stored chats + latest `ConnEvent`; only `h.openChat` (notification target) survives via `openFromNotification`. No per-account scroll/chat memory exists. |
| Notifications | Open account's notifier is `clear()`ed on close (all shown toasts removed, `accounts.go:357`); a background account's notifier **outlives** its backend between checks (`background.go:50ff`, `bgAccount.notes`) and is re-attached in `useBackend`. IDs are `<account Dir>\|<chat>` (`notifier.noteID`/`splitNoteID`, `notifications.go:158-164`); titles append `" · " + account` for background accounts (`notifications.go:264`); `host.route` (`host.go:449`) switches to the owning account first, and Reply/Mark-as-read wait in `h.later` until that account is online (`actLater`). Tray tooltip counts **all** accounts' unread (`background.go:611-612`). |
| Background accounts | Unaffected by the switch itself except the account switched *away from* (becomes `Always`-running, `Checks`-scheduled, or closed per its mode). They are polled on the UI goroutine by `pollBackground()` (`background.go:393`) from the `Run` loop's `wake` case; their backends open/close on worker goroutines and return via `bgOpen`/`bgOpened`. Timings (`background.go:30-48`): `bgStartDelay` 5s, `checkGrace` 3s after `CaughtUpEvent`, `checkTimeout` 90s, `bgRetry` 5min, `logoutTimeout` 1min, `bgMaxWait` 10min. `bgEvents` tracks in-flight sends (`bg.sending`) so a check does not close mid-send, and a check is also armed at the next scheduled job (`auto.Next()`, `scheduleCheck`). |
| Theme / app prefs | Carry over (`appPrefs` copy in `closeAccount`); per-account prefs do not. App lock (`h.lock`) outlives window and account (`newUI`). |

### 1.4 What is missing for true multi-account (Phase 2 gaps)

1. **No simultaneous open accounts**: exactly one `UI`/window; background accounts have no UI state (no chats in memory, no
   scroll, no drafts). "Switch" is close+reopen (or backend hand-over), not a tab switch; latency is a full `wa.Open` (SQLite open
   + device-store upgrade + chat load) unless the target was `Always`.
2. **No per-account unread in the switcher**: `accountRow` (`accounts.go:37`) has `bg bgView` (running/checking/ state/failed) and
   `status()` text, but no unread count; the rail badge counts only the open account's chats (`rail.go`, `layoutRail`). Background
   notifiers know unread counts internally but do not expose them to the switcher.
3. **No memory budget**: any number of accounts can be `Always`; AGENTS.md says an idle connected background account costs ~2 MB
   (+~10 MB for a big DB) but nothing enforces a cap (master spec: max 2 `Always` by default).
4. **No rename/reorder** in `internal/accounts` (see §1.1); `List.Add` numbering is append-only.
5. **Drafts/scroll not preserved per account** (see §1.3); `clearDrafts` is unconditional.
6. **AI config has no home yet** — per-account AI settings must be stored per account dir (see §4/Phase 3) and must not leak
   across accounts; today only prefs (per-DB) and `accounts.json` exist as per-account stores.
7. **Switch is not <1s for a `Checks`/`Off` account**: it pays `Open` + `Start` + connect + history/app-state sync on the UI
   goroutine path (`switchAccount` calls `h.o.Open` synchronously; only background opens are goroutine-based). No pre-warming of
   the next account.
8. **Logout of one account is safe** (`bgLoggedOut`, `logoutAccount`, `background.go:471-535`) but a background logout while its
   backend is mid-check races `takeBackground` only via the `loggingOut` flag — worth a test.

## 2. `internal/wa` Backend — connect, reconnect, events, send

### 2.1 Structure

`Backend` (`backend.go:39-111`): `db *sql.DB` (modernc SQLite, WAL, `busy_timeout(10000)`), `container *sqlstore.Container`
(hypermeow device store, same file), `store msgStore` (`wz_*` tables), `cli *whatsmeow.Client`, `mu` + `events []model.Event`
queue, `notify func()`, `work sync.WaitGroup` (+ `workMu`/`closed`) for post-connect work, `background atomic.Bool`, `serverSkew`,
avatar/download `*fetcher`s, `names nameCache`, `statusPriv`, search/gallery/summary cancellers. `Open(dataDir, debug)`
(`backend.go:135`): `MkdirAll(dataDir, 0700)`, truncates `wazzap.log` over 4 MB, opens `wazzap.db`, `container.Upgrade`,
`store.init`, `useDevice(GetFirstDevice)` — the client is created **without connecting** so stored chats render first. `useDevice`
(`backend.go:264`): `whatsmeow.NewClient`, `EnableAutoReconnect = true`, `EmitAppStateEventsOnFullSync = true`,
`cli.AddEventHandler(b.handle)`, `cli.RawNodeHandler = b.groupedReceipts` (grouped receipts, `receipts.go`).

### 2.2 Connect / reconnect / lifecycle

- `Start(notify)` (`backend.go:216`): stores notify, `go b.run()`, `go b.typing.run`, avatar/download fetchers. `run()`
  (`backend.go:228`): if `Store.ID != nil` → `connect()`; else if background → emit `StateQRExpired` (background accounts never
  show QR); else `pair()` (QR loop, `startPair`/`pair`, `backend.go:293-367`).
- `connect()` (`backend.go:283`): emits `StateConnecting`, `cli.Connect()`; on error emits `StateOffline` (hypermeow
  auto-reconnects — `EnableAutoReconnect`).
- `handle(evt any)` (`backend.go:684`) is the single hypermeow event entry point. `*events.Connected` → `StateOnline` +
  `goWork(sendPresence, refreshGroupNames, resyncAppStateOnce, refreshChannels)`; `*events.OfflineSyncCompleted` → `CaughtUpEvent`
  (this is what ends a background check, §1.3); `Disconnected`/`KeepAliveTimeout` → `StateOffline`; `KeepAliveRestored` →
  `StateOnline`; `LoggedOut` → `go b.resetSession()` (wipes store via `store.wipe`, new device, re-pair or `StateQRExpired` if
  background); `StreamReplaced`/`TemporaryBan`/`ClientOutdated`/`ConnectFailure` → `fail()` = `StateError`; `HistorySync` →
  `onHistory`; `Message` → `onMessage`; `UndecryptableMessage` → `onUndecryptable`; `Receipt` → `onReceipt`;
  `ChatPresence`/`Presence`/`Pin`/`Mute`/… → typed model events. `Close()` (`backend.go:422`): marks closed, `cancel()`,
  `cli.Disconnect()`, `work.Wait()`, `db.Close()`, `logf.Close()`. `Retry()` (`backend.go:395`): re-pair if unlinked else
  disconnect+connect. `SetBackground(on)` (`backend.go:240`): flips the flag and re-sends presence (`sendPresence`,
  `backend.go:640`: "available" when open, "unavailable" when background — so the phone keeps notifying).
- Events are buffered under `mu` by `emit(e)` (`backend.go:198`) and drained by `Poll()` (`backend.go:208`). The host drains them
  in `host.poll` (`host.go:333`), the window via `hostBackend.Poll()` → `uiEvents()`; `host.poll` also feeds every event to
  `notifier.event` and handles `ConnEvent`/`AccountEvent`/`SyncEvent`. **UI code never touches hypermeow types** — only
  `internal/model` events. `MessageEvent.New` marks just-arrived messages (not history/edits/reactions) — the notifier only
  notifies on those.

### 2.3 Incoming-message interception point (for AI / auto-reply)

There are two candidate layers; the correct one for AI is the **outer** one:

1. **`wa.Backend.handle` → `onMessage` (`backend.go:1045`)** — the lowest point: status messages divert to `onStatus`, the rest go
   through `parse` → `apply` (`backend.go:1125`), which stores the message (`putMessage`, payload in `raw_payload`) and emits
   `model.MessageEvent` (+ `emitChat`). `onUndecryptable` (`backend.go:1086`) stores `KindWaiting` / `KindViewOnce` placeholders.
   Intercepting here would see undecryptable/view-once/status traffic and would have to duplicate `auto`'s `New`-filtering; do not
   add a second hook here — `auto` already wraps it.
2. **`internal/auto.Backend.Poll()` (`auto/auto.go:111-138`) — the single
   interception point to use.** It wraps any `model.Backend`, sees every `model.MessageEvent` via `seen(e)` (`auto.go:336`), plus
   `ConnEvent` (online/meID) and `ChatsEvent`/`ChatEvent` (group cache). It already implements the exact shape AI auto-reply
   needs: per-event filtering (`answers(m)`, `auto.go:371`: no newsletters/broadcasts, groups only on @-mention via `mentionsMe`),
   once-per-sender dedup (`Away.Told`), own-send detection (`ours` map + `FromMe`), and sending through the wrapped backend. AI
   auto-reply (§5.3(4) of the master spec) must be added **inside `auto.Backend`** (new fields/methods + prefs), reusing
   `Poll`/`seen`/`arm`, not as a second wrapper or a UI hook — otherwise scheduled/AFK/AI replies diverge and ghost mode
   (`refused()`, `auto.go:484`) would not cover AI sends. The UI must never be the interception point: with the window closed
   there is no UI, but `auto` keeps running (`withAuto` in `host.go:708` wraps the host's backend, and background accounts are
   wrapped the same way in `background.go`).

### 2.4 `Send(chatID, d model.Draft)` — `internal/wa/actions.go:179-235`

Builds `model.Message{ID: cli.GenerateMessageID(), FromMe: true, Receipt: Pending, Time: sendTime()}` (server-skew-adjusted),
builds the `waE2E.Message` (plain `Conversation`, or `ExtendedTextMessage` with `ContextInfo` from `draftContext` for
reply/mentions/link-preview, `GroupMentionedMessage` wrapper for @admin/@chat), then `storeAndSend`/`storeAndSendAfter`
(`actions.go:257-293`): stores the message **first** as pending with its raw payload (`putMessage`), `emitChat`, then
`sendAsyncPrep` on a goroutine — upload prep, `setRawPayload`, `sendMessage` via hypermeow (with fallbacks for button answers),
`setReceipt(Sent)`, `ReceiptEvent`, `serverTime` (server timestamp + skew learning). Failure → `sendFailed` (`actions.go:351`):
marks failed in the DB + `ReceiptEvent{Failed}` + `NoticeEvent`. Sibling send paths through `storeAndSend*`: `SendFile`,
`SendSticker`, `SendContacts`, `SendPoll`, `SendEvent`, `Forward`/`sendCopy`, `SendNewSticker` (`actions.go` / `files.go`); `auto`
calls `Backend.Send` for scheduled/AFK sends. `Send` returns nil when the JID does not parse or there is no client — callers must
nil-check.

## 3. `internal/auto/auto.go` (630 lines) — schedule, AFK, ghost

- `Backend` (`auto.go:58`) embeds `model.Backend`; fields: `jobs []Job` (sorted by `At`), `away *Away`, `meID`, `online`, `ours
  map[string]bool` (IDs it sent), `groups map[string]bool` cache, `out []model.Event` (events it injects on the next `Poll`),
  `notify func()`, `timer *time.Timer`, `seq`, `ver`.
- State is persisted **in the wrapped backend's prefs** (`Backend.Pref`/`SetPref` → `wz_meta` `pref:*`, §4): `prefJobs =
  "auto_scheduled"` (JSON `[]Job`), `prefAway = "auto_away"` (JSON `Away`); loaded in `Wrap` (`auto.go:79`). So jobs/AFK survive
  restarts and run with the window closed. `Job{ID, Chat, At, Text, Shown, Mentions, MentionAll, MentionAdmins, Reply}`;
  `Away{Reason, Since, Told []string}` (`"chat|sender"`, each once).
- `Poll()` (`auto.go:111`): drains wrapped events; on `MessageEvent` calls `seen(e)`; then `sendDue()`; appends `a.out` (its own
  sent messages as `MessageEvent`s + `NoticeEvent`s). `seen` (`auto.go:336`): ignores pre-AFK messages and its own sends; a
  `FromMe` message from another device ends AFK (`cameBack`); otherwise, if `e.New && answers(m)`, records `Told` and sends
  `AwayText(w, now)` ("💤 *AFK*…") as a reply via the wrapped backend, marking the ID in `ours`. `answers` (`auto.go:371`): false
  for `@newsletter` / `@broadcast`; groups only when `mentionsMe` (a `MentionNotifies` mark that is not `@all`); 1:1 always.
- Scheduling: `Schedule(j)` (`auto.go:139`) assigns `ID = <unixnano base36>-<seq>`, inserts sorted, `saveJobs()`, `arm()`;
  `Jobs(chat)`, `Next()` (first job's `At` — background checks are armed at this time), `Cancel(id)`, `SendNow(id)`. `arm()`
  (`auto.go:275`): a single `time.AfterFunc` for the next job, delay clamped to `[0, 1h]`, firing `notify` (the host then `Poll`s
  — **the timer only pokes; all work happens in `Poll` on the UI goroutine**). `sendDue()` (`auto.go:215`): only when `online`;
  sends due jobs, drops jobs more than `lateBy = 5min` late with `MissedText` notice. `send(j)` rebuilds the `model.Draft`
  (mentions, reply looked up via `MessagesFrom`).
- AFK: `SetAway(reason)`, `Back() int` (returns count told), `Away()`, `AwayText`/`AwayPlain`/`BackText`. Any user send ends AFK:
  `sent()` (`auto.go:459`) is called by every user-like send wrapper.
- Ghost mode: `ghost bool` read from `model.PrefGhost` (`"ghost"`, `"on"`) in `Wrap` and kept in sync by `SetPref`
  (`auto.go:472`). `refused()` (`auto.go:484`): when ghost is on, `Send`, `SendFile`, `SendSticker`, `SendNewSticker`,
  `SendContacts`, `SendPoll`, `SendEvent`, `Forward`, `PressButton`, `React`, `VotePoll`, `Edit`, `Delete` (for-everyone),
  `PinMessage`, `KeepInChat`, `PostStatus` all return nil/no-op and append `NoticeEvent{GhostText}` ("Ghost mode is on. Turn it
  off to send messages."). **Auto's own sends bypass `refused()`** — they call `a.Backend.Send` directly (`seen`, `send`). AI
  auto-reply must follow the same split, and must additionally check ghost itself (master spec §5.4: ghost on ⇒ AI auto-reply must
  not send).
- `SetBackground(on)` (`auto.go:175`) forwards to the wrapped `model.Backgrounder`. `Version()` bumps on every job change (UI
  re-reads jobs). `whisper.go`: `SendWhisper` (group whispers) is the only other `auto` send path. Tests:
  `internal/auto/*_test.go` use a fake backend + fake clock — the pattern to copy for AI tests.

## 4. Settings + KLIPY key pattern (reuse for AI)

### 4.1 Pref storage

- `model.Backend.Pref(key)/SetPref(key, value)` (`model.go:1500-1502`) — string key/value, per account. `wa` implements them in
  `actions.go:1017-1033`: `Pref` = `store.meta(ctx, "pref:"+key)`, `SetPref` = `store.setMetaValue(ctx, "pref:"+key, value)`, i.e.
  one row in `wz_meta(key TEXT PRIMARY KEY, value TEXT NOT NULL)` (`store.go:92`) inside that account's `wazzap.db`. No typing, no
  namespacing beyond the `pref:` prefix, no encryption, no file-mode protection beyond the DB file itself (0600, data-dir 0700).
  `SetPref("ghost")` additionally stops typing and re-sends presence. **Consequence for AI**: a per-account AI config can live in
  prefs (small JSON blob, like `auto_scheduled`), but the API key itself should follow §5.2 of the master spec — a separate `0600`
  JSON file in the account data dir, not a pref row — because prefs are plain text in the DB.
- App-vs-account split: `appPrefs` (`ui/accounts.go:30-34`) are copied to the next account on switch; everything else is
  per-account by construction. `prefOn(b, key)` (`notifications.go:25`) = `Pref != "off"`; `extraOn`/`setExtra`
  (`extras.go:38-48`) = `"on"`/`"off"` for extra features.

### 4.2 Settings structure — `internal/ui/settings.go` + `settingsdetail.go`

- `settingsItems` (`settings.go:50-63`, 13 categories mirroring WhatsApp Desktop): General, Profile, Account, Privacy, Chats,
  Video & voice, Notifications, Storage, Keyboard shortcuts, Performance, Extra features, Snippets, Help, Log out. `settingsState`
  holds `detail` (category+1), `sub` (page inside), `page []settingsSection`, `stale bool`, search `hits []settingsHit`
  (`settingsSearchHits`, `settings.go:234` — settings search already exists and indexes every page's rows, so a new "AI" section
  is searchable for free).
- Pages are built as `[]settingsSection{title, rows []settingRow, note}` by `settingsRows()` / `categoryPage()`
  (`settingsdetail.go:144-325`); `settingRow{key, kind (setToggle/setRadio/setCustom/setKeys), title, sub, on, run, trailing, …}`
  (`settingsdetail.go:61`). `extrasSettings()` (`extras.go:138`) builds the Extra features page: sections "Slash commands",
  "Mentions", "Photos", then the "Ethically gray features" sub-page (`grayToggle` asks for acknowledgement before enabling, every
  time; `grayCmdPref(name) = "cmd_"+name`).

### 4.3 KLIPY — the key pattern to copy (`internal/klipy/klipy.go`, `internal/ui/gifs.go`)

- `klipy.Client{Key, CustomerID, UserAgent}` (`klipy.go:36`); `baseURL = "https://api.klipy.com/api/v1/"`, package
  `http.Client{Timeout: 20s}`; `ErrKey` on 401/403, "too many requests" on 429, other non-2xx as `"klipy: <status>"`; responses
  capped (`maxJSON` 2 MB, `maxMedia` 16 MB); `do()` puts the key in the **URL path** (`baseURL + PathEscape(key) + "/" + path`) —
  KLIPY's documented integration, key is explicitly "not a secret" (`klipy.go` doc comment). `Search`/`Trending`/
  `Recent`/`Items`/`Share`/`Fetch` (Fetch requires `https://`). No proxy/cache of ours; results in server order; UI shows "Search
  KLIPY" + "Powered by KLIPY".
- UI side (`gifs.go:30-36`): prefs `prefKlipy = "klipy"` (on unless `"off"`), `prefKlipyKey = "klipy_key"` (user's own key, `""` =
  build's), `prefKlipyID = "klipy_customer"` (random customer ID, generated in `klipyClient`, `gifs.go:152-160`), `prefKlipyFavs =
  "klipy_favs"` (slugs, space-joined). `loadGIFPrefs` (`gifs.go:136-140`) reads them once (`Pref` reads the DB); the key editor is
  a `widget.Editor` (`keyEd`, single-line) on the Extra features page (`gifs.go:678-706`): on change it `SetPref(prefKlipyKey, k)`
  live, `reset()`s results, and shows the note "Get a free key on partner.klipy.com… keep its ads off". The build's key comes from
  `-X main.klipyKey` (release secret); `key()` = user's key else build's key; `key() == ""` ⇒ tab shows "Enter a KLIPY API key in
  Settings > Extra features". **Validation is lazy, not on save**: there is no "Test" button — a refused key surfaces as
  `klipy.ErrKey` → "KLIPY refused the API key. Check it in Settings > Extra features." (`gifs.go:530`, `layoutGIFTab`). AI must
  improve on this one point: master spec §5.2 requires Test Connection **before** enabling auto-reply.
- **What to reuse for AI, verbatim**: per-account pref keys (`ai_*`, plus a separate 0600 file for the secret); a
  `widget.Editor`-based settings row on a settings page with live `SetPref` + hint/note text; a `Client`-style struct with `Key` +
  timeouts + capped reads + typed sentinel errors (`ErrKey`-style) mapped to user-facing strings; never logging the key; a
  global-default-vs-per-account split mirroring `appPrefs` (KLIPY key is in `appPrefs`, so it *carries over* on switch — for AI
  the key must **not** be in `appPrefs`, or it would leak across accounts; only a global *default* endpoint/model may carry over).
  **What not to copy**: key-in-URL (AI keys go in an `Authorization: Bearer` header), no test-on-save, and KLIPY's "key is not a
  secret" stance (an AI key *is* a secret: mask it, `Redact()` in logs).

## 5. Slash commands

### 5.1 Registry — `internal/command/` (no Gio here)

`command.go`: `Command{Name, Description, Options []Option, Group, Admin bool, Gray bool, Run func(*Context) error, Preview
func(*Input) (string, bool)}`; `Option{Name, Description, Kind, Required, Multiple, Choices, Min/Max, Until, Filter, Mentions}`;
`Kind`: `Text, Member, Contact, Choice, Number, When, Snippet`. `Lookup(name)`, `Matching(prefix, group)`, `Usage()`, `Parse(text,
caret, mentions, members)` → `Input` (with `Text(name)`, `Int(name, def)`, `IDs(name)`, `Missing()`, `Problem()`), tokeniser
handling @mentions (`tokenize`, `mentionAt`). `context.go`: `Host` interface (Note/Dismiss, Group, Do — work off the UI goroutine
then a func back on it, Copy, Confirm, PickImage, Sent, Draft, SetGhost, SnippetVars, SnippetsChanged, EditSnippet, ShowPayload)
and `Context{Cmd, Input, Values, Chat, Info, Reply, Backend, Now, Auto *auto.Backend, Host}`. `Execute(c)` (`context.go:137`):
rejects non-group use of `Group` commands and non-admin use of `Admin` commands (when `Info` is known), then `Run`; errors become
`Fail` → a `Note` in chat. `Note{Title, Text, Failed, Buttons []Button}` is the commands' only UI output channel (local notes,
§5.2).

### 5.2 All commands — `internal/command/commands.go` (`All`)

| Command | File | Notes |
|---|---|---|
| `/add`, `/kick`, `/promote`, `/demote` | `commands.go:17–37` | Group+Admin; `Member`/`Contact` (multiple) → `changeMembers` |
| `/link`, `/lockdown`, `/description` | `commands.go:40–52` | Group+Admin; invite link (+Copy/Reset), lockdown Choice, description Text |
| `/sticker` | `:55` | `Text top (Until "\n")` + `bottom`; `runSticker` |
| `/purge` | `:63` | `Number count`; deletes last N / up to reply, for everyone |
| `/raffle` | `:69` | Group, `Number winners` 1–50; `raffle.go` |
| `/calc` | `:74` | `Text sum`, live `Preview` (`previewCalc`); `calc.go` |
| `/schedule`, `/scheduled` | `:81`, `:92` | `When when` + `Text message` (Mentions) → `c.Auto.Schedule(auto.Job{…})`; list jobs with send-now/cancel |
| `/afk` | `:96` | `Text reason` (optional); `runAFK` (`schedule.go:111`) → `c.Auto.SetAway(reason)` |
| `/ghost` | `:101` | **Gray**; `ghost.go:10 runGhost` → `Host.SetGhost` (toggles `model.PrefGhost`) |
| `/whisper` | `:105` | Group, **Gray**; `Member` multiple (Until newline) + `Text` (Mentions); `whisper.go` |
| `/snippet`, `/catch` | `:115`, `:125` | `Choice` send/save + `Snippet` name; `/catch` shows replied message's raw payload via `Host.ShowPayload` |


Gray commands need their own switch (`cmd_<name>`, `extras.go:32`, `commandOn` in `extras.go:36` gates the picker; `grayToggle`
asks for acknowledgement). `Auto` is nil without `auto` wrapping — `runSchedule`/ `runAFK` fail with `errNoAuto` ("This can't be
done here", `schedule.go:14`) in that case.

### 5.3 UI side — `internal/ui/slash.go` (947 lines)

- `slashState`/`slashPick`: `readSlash(txt, caret, c, info)` (`slash.go:149`) calls `command.Parse` + `command.Matching`; `rows()`
  counts picker rows; `slashShown`/`slashEnterPicks`; `slashKeys` (`slash.go:313`) reads Up/Down/Tab/Enter **before** the composer
  (AGENTS.md gotcha) and only while rows are offered; `slashTakesMentions` hands Enter to the mention picker for `Option.Mentions`
  text (e.g. `/schedule`'s message); `pickSlash`/`submitSlash` (`slash.go:361/391`) build `command.Context{…, Backend: u.backend,
  Auto: u.auto, Host: slashHost{…}}` and call `command.Execute(ctx)`; `slashHost` (`slash.go:436`) implements `command.Host`
  (`Note` → `localNote` rendered by `layoutNote` at the chat end, `Do` → `slashDo`/`drainSlashDone` for off-UI-goroutine work,
  `Draft` → `draftWith`, `SetGhost` → `u.setGhost`, etc.). The picker is drawn by `layoutSlashPicker` (`slash.go:722`); hints by
  `slashHint`/`paintSlashHint`; `//` escapes to a literal slash (`compose.go:191`). Slash commands are an extra feature:
  `prefSlash = "slash_commands"` (`extras.go`), `u.slash.on`; the composer only treats leading `/` as a command when on
  (`composertext.go:46`).
- **Adding `/ai`**: (1) append a `Command{Name: "ai", Description: …, Options: []Option{{Name: "question", Kind: Text, Required:
  true}}, Run: runAI}` entry to `All` in `commands.go` (file is already 566 lines — put `runAI` in a new `internal/command/ai.go`
  per the 200-line/file rule, and add an `AI` hook to `command.Context` or reuse `Host` — the `Host` interface is the extension
  point, but it has no AI method today, so add one, e.g. `AskAI(question, chatID string)` returning a channel/callback, or pass an
  `*ai.Agent` on `Context` like `Auto`); (2) add `"ai": ic…` to `commandIcons` (`extras.go`) or it falls back to `icTerminal`; (3)
  no `Gray` unless auto-reply is bundled into it — `/ai` as ask-only needs no gray switch; (4) the answer must come back as a
  `Note` (local, labelled, not sent) via `Host.Note`, with a "Send" button (`Note.Buttons`) that calls `Backend.Send` — matching
  `runLink`/`runScheduled` patterns; (5) network must run via `Host.Do` (off the UI goroutine), exactly like `runSticker`'s
  background work; (6) `Auto` on `Context` is the precedent for passing a backend-adjacent service into commands. Also update
  `slash.go`'s demo overlay list (`screenshot` `-overlay slash…`) if the picker gains rows.

## 6. Theme / UI today

### 6.1 `internal/ui/theme.go` — `Palette` (229 lines)

One struct, every colour the UI uses; no hard-coded colours in widgets (AGENTS.md convention). Two instances: `darkPalette` and
`lightPalette` (dark values sampled from WhatsApp Desktop 2025 design). Fields (see §11 for the full list): frame/panel/chat
backgrounds, hover/selected, chips, text/icon, green accent, rail, doodle, bubbles in/out + meta/ticks/link/quote,
code/mention/selection, date chip, unread band, encryption, composer, avatars (incl. `AvatarBgs`/`AvatarFgs` slices +
`GeneralAvatar`), banner, menu/shadow, close-hover, `Senders` (group sender colours), empty/row-hover, danger, status ring,
verified, announce/channel avatars, status bg, popup/picker/viewer/scrim/dialog/toast. `newTheme()`: `material.NewTheme()` with
`th.Face = typeface` ("Segoe UI, <emoji>, Segoe UI Symbol, MS Gothic, …", `theme.go`), `TextSize 14`. `SetDark(dark)`
(`app.go:605`): `u.dark = dark; u.pal = &darkPalette | &lightPalette`. Per-chat tinting exists already: `chatThemes`
(Default/Teal/Blue/Purple/Pink/Red/Orange/Yellow/Slate, `chatmenu.go:157-166`) + `themed(p, dark, t)` (`chatmenu.go:205-232`,
copies the `Palette` value and mixes `BubbleOut`/`QuoteOut`/`MetaOut`/`ChatBg`/`Doodle` toward the hue) + `chatPalette(c)`; the
choice is stored per chat as pref `theme:<chatID>` (`prefChatTheme`, `chatmenu.go:169-200`). `layoutConversation`
(`conversation.go:145-150`) temporarily swaps `u.pal` to the chat palette and restores it with `defer`.

### 6.2 Selection / persistence / application

Pref `theme` (`settingsdetail.go:25`): `"light"`, `"dark"`, or `""` = system default. `setTheme(t)` (`settingsdetail.go:770`):
`SetPref` + `applyTheme()` — **instant, no restart**. `applyTheme` (`settingsdetail.go:777-795`): light/dark forced, else
`systemDark()` (`internal/desktop/theme*.go`; re-read on window focus, `host.go:635`, so an OS change while running is followed; a
system that doesn't answer defaults to dark, "which was the app's own default"). `themeNames` maps values for the Chats settings
page (`settingsdetail.go:253-257, 299` — radio rows System default / Light / Dark). The theme pref is in `appPrefs`, so it
**carries over on account switch** (global, not per-account — §11 fixes that).

### 6.3 Layout — `internal/ui/app.go` (2134 lines), `rail.go`, `sidebar.go`, `conversation.go`

- `layoutWindow` (`app.go:736`): title bar (`layoutTitleBar`), then if not logged in → login screen (`layoutLogin` +
  `layoutLoginSwitch` + account menu); if app-locked → lock screen; else `layoutMain`.
- `layoutMain` (`app.go:809`): `railWidth = 68dp` (`rail.go:15`); list width from `listWidth` (`split.go:80`): 37.2% of pane width
  clamped to `[280dp, 430dp]` (draggable edge, `split.go`, `prefListWidth`/`prefListHidden`, Ctrl+Shift+L toggles); right pane =
  rest. Panels sit in a rounded bordered card (`fillRect` borders + `roundCorner`, deliberately **no** whole-window background
  fill / no big `clip.RRect` — GPU cost, AGENTS.md). `layoutPageSidebar` (`app.go:864`) switches on `page`
  (Chats/Status/Channels/Communities/Settings/Calls); `layoutRightPane` (`app.go:900`): open conversation (+ info/search/msg-info
  panel at `max(340dp, 30% width)`, `layoutWithInfo`, `app.go:949`) or `layoutPlaceholder`/`layoutEmpty` per page.
- `rail.go` (`layoutRail`, 198 lines): Chats (badge = unread non-archived count; archived-unread counted separately), Archive,
  Status (dot for unseen), Channels (dot), Communities, Settings at the bottom; active-item circle glides (`switcher` anim). No
  account avatar/unread in the rail — the account switcher lives in the ⋮ menu / login screen (`layoutAccountMenu`,
  `accounts.go:451`, 300/340dp wide).
- `sidebar.go` (798 lines): header + search (`layoutSearch`), filter chips (`chips.go`, custom lists), `layoutChatList` via
  `u.scrollList` (virtualised, `chatOrder` measures row heights, animated reorder); `layoutChatRow`/`chatRow`
  (`sidebar.go:295/375`): avatar (photo or initial), name + verified badge, time, preview (sender, media icon/label, "Draft:"),
  ticks, mute/pin/archive indicators, unread badge (`nameWithBadge`, `sidebar.go:437`; an empty green badge for mark-unread,
  `:724`).
- `conversation.go` (1846 lines): `layoutConversation` = header (`layoutConvHeader`: avatar, name, presence/ typing, ⋮ menu) +
  pinned banner + messages (`layoutMessages`, paged — see §8) over `u.conv.wallpaper.layout(gtx, pal.ChatBg, pal.Doodle,
  u.doodles)` + composer (`layoutComposer`, `compose.go:642`, floating over the wallpaper; recorded first so the list ends above
  it). Message grouping: same-sender runs within a day, 10-min sticker runs, albums, system rows, unread divider (`unread.go`),
  scheduled bubbles appended after newest (`scheduled.go`). Composer: `widget.Editor` with `paintComposerText` overlay for
  formatting/@mentions, formatting toolbar (`formatbar.go`), @mention picker, emoji/sticker/GIF picker, attach menu, voice
  recording (`voicerec.go`), slash picker (§5.3).
- Density/font scale: **no density setting exists**. Font scale = whole-window zoom, `scale.go`: `prefZoom = "zoom"`, `zoomLevels
  = [80, 90, 100, 110, 125, 150, 175, 200]`, `loadZoom`/`setZoom`/`stepZoom`/`zoomKeys` (Ctrl + / - / 0, with a 2s bubble);
  `applyZoom` wraps `layoutWindow` (`app.go:730`). Zoom scales everything, not just text — a Compact/Comfortable density must be a
  separate setting (row heights, paddings) or it will fight zoom.

## 7. Top 10 UI/UX gaps vs official WhatsApp Desktop

1. **No `Ctrl+K` quick search, no `Ctrl+N` new chat, no `Ctrl+1..9` account switch, no `Ctrl+Shift+M` mute.** Shortcuts that exist
   (`settingsdetail.go:813`): Enter/Shift+Enter, Ctrl+B/I/Shift+X/Shift+M formatting, Ctrl+C/A, viewer arrows/space, photo-editor
   Ctrl+Z/Delete, Esc, Ctrl+Shift+L, Ctrl+Shift+P, Ctrl + / - / 0, Ctrl+Shift+F (`app.go:1047`). Missing ones need `key.Filter`
   handlers in `update()` (`app.go:989`) + rows in `shortcutSettings`.
2. **No density setting.** Chat rows are fixed geometry
   (`settingsGeom`/`listGeom`, `rowHeight`/`chatOrder`, `sidebar.go:192-276`); WhatsApp's compact list fits ~30% more rows. Add a
   `density` pref + a row-height/ padding multiplier threaded through `sidebar.go`/`settings.go` geometry, not a second zoom.
3. **Chat list shows no per-account context.** The rail badge is open-account-only (`rail.go`); background accounts' unread is
   invisible until a notification arrives (see §1.4(2)). Fix: expose unread per `bgAccount` (notifier already counts it,
   `notifications.go:343-358`) into `accountRow`, draw a badge in `accountItem` (`accounts.go:541`) + a rail account avatar with
   badge.
4. **Loading states are thin.** Opening a chat shows stored messages at once (good), but history sync / app-state resync shows
   only a `SyncEvent` percent (`host.go:333`, `syncPct`) with no skeleton in the list; a fresh link shows the login screen with no
   progress beyond QR. WhatsApp shows skeleton rows + a sync progress bar. Fix: skeleton rows in `layoutChatList` while `syncPct
   >= 0`.
5. **Empty states are static text.** `layoutEmpty` (`conversation.go:1815`) / `emptyPane` (`app.go:923`) are icon + 2 lines, no
   action. Fix: add primary actions to `emptyPane` call sites, starting with Chats ("Start a chat" → `layoutNewChat`).
6. **Search is split and shallow.** Chat-list search (`listsearch.go`), in-chat search (`chatsearch.go`, panel replacing info),
   and settings search (`settings.go:234`) are three separate implementations; list search has no keyboard entry (no Ctrl+K) and
   no recent searches. Fix: Ctrl+K focuses `layoutSearch` (`sidebar.go:80`); share the `SearchMessages` backend path.
7. **No accessibility layer.** Gio exposes no screen-reader semantics; buttons are icon-only with no tooltips in most places
   (rail, composer, viewer toolbar `viewer.go:213-249`). Fix: hover labels on rail/composer/viewer controls at minimum, plus
   visible focus rings for keyboard navigation.
8. **Error/failed-send states are quiet.** A failed send marks the message `Failed` + a `NoticeEvent` toast (`actions.go:351`);
   there is no inline retry affordance on the bubble and no outbox/queue UI (`queueSend`/`outbox`, `editrender.go:136`, are
   invisible). Fix: failed bubble gets a retry button (needs a `wa` retry entry point — none exists today).
9. **Calls page is a dead end.** `layoutPlaceholder` for `pageCalls` (`app.go:939-941`): "Calling from this app isn't supported
   yet. Use your phone…" — correct per scope (no calls), but the rail still shows the Calls entry. Keep the honest placeholder; do
   not fake calling.
10. **Status/Channels/Communities list parity is partial.** Status list (`status.go`), channel list, community list each
   re-implement list chrome instead of sharing `sidebar.go`'s; status viewer (`status.go:341-516`) is separate from the media
   viewer (`viewer.go`). Fix: unify empty-state/action patterns and toolbar vocabulary, without merging the viewers (different
   input models).

## 8. Memory discipline — must not regress

- `internal/memtrim` (`memtrim.go:12-17`): `Trim()` = `debug.FreeOSMemory()` + Windows `SetProcessWorkingSetSize(-1,-1)`
  (`memtrim_windows.go`); called after account switch (`useBackend`), after window close (`closeWindow` → `actLater` path trims
  too), and after ~10s without a frame (`idleTrim` in `host.go`/`ui.Run`; `perf.trimAfter`, default `idleTrim`, `"never"` disables
  via `prefTrimAfter`). `quietTrim` (`host.go:384`) does the same with no window. **Keep every call site.**
- `internal/offheap` (`offheap.go`): `NewRGBA` allocates pixels outside the Go heap (mmap/VirtualAlloc per OS file) with a
  `runtime.AddCleanup` free; `InUse()` counts bytes. Rule from AGENTS.md: big pixel buffers (animated emoji sheets ≤64 MB, decoded
  emoji sheets on disk ≤256 MB in `<data>/emoji/sheets`) go off-heap; **never keep `Pix` (or a `SubImage`) without the picture**.
  New wallpaper images (§12) and AI-generated images must follow this.
- `internal/photo` (`photo.go`, `shrink.go`): `Prepare(data, q)` (Standard 1600px/q80, HD 4096px/q90, Raw passthrough, `settings`
  map), `Size(w,h,q)`, `Estimates` (decodes once, estimates all three); `Shrink(src,w,h)` is the only sanctioned downscaler —
  area-weighted box filter, a few rows of memory, vs `x/image/draw.CatmullRom`'s 157 MB for a 12 MP photo. JPEG decode is
  serialised (`acquireDecode`, progressive JPEG coefficients ~7.5 B/px). Reuse `Shrink` + `Prepare` for wallpaper (§12) and HD
  status (§14).
- Paging/virtualisation: `u.msgs` is a window, not the chat (`paging.go` — pages of 100, ≤400 loaded); `u.scrollList`
  (`scrollbar.go`) for every list; `imageCache` bounded by **bytes**; emoji bitmap caches bounded (`emojiimg.go`,
  `emojiAnims.endFrame` trims); `dropCaches()` (`host.go:755`) empties glyph/preview/ rich-block/read-more/icon/shape caches on
  window close and account switch. New per-account state must be dropped the same way (a second open account's caches would double
  idle RAM).
- GC: **do not set a memory limit** (`SetMemoryLimit`/`GOMEMLIMIT`) — AGENTS.md documents back-to-back GC and 20× scroll CPU once
  the live heap nears it. `prefGCPercent` (default 200) is user-tunable via Performance > Advanced (`perf.go:24-40`, `perfPrefs` —
  all in `appPrefs`, so they carry across accounts).
- Verified costs to keep in mind: idle connected background account ~2 MB (+~10 MB big DB); animated emoji sheets ≤64 MB; emoji
  sheet disk cache ≤256 MB, LRU by last read.

## 9. Risks / blockers for Phases 1–3, and baseline verification

### Risks

1. **Gio pre-1.0 API drift** — every UI change must be checked against the pinned `v0.10.3` source in `third_party/gio` +
   `pkg.go.dev/gioui.org`; training-data Gio is wrong (AGENTS.md Rule #1). A missing `sh patches/apply.sh` breaks the build
   (`replace gioui.org => ./third_party/gio`).
2. **hypermeow is a moving fork** (`@main`, tagged versions retracted): `go get github.com/polymorfa/hypermeow@main` can move
   behaviour (LID-first identities, `DangerousInternals`, secret-encrypted edits). Pin by the pseudo-version in `go.mod` and
   re-read its source on bump.
3. **200-line file / 50-line function hard limit** (master spec §1.9, allowed over-limit count is zero): nearly every file touched
   in Phase 1 is already over 200 lines (`app.go` 2134, `conversation.go` 1846, `accounts.go` 852, `slash.go` 947, `theme.go` 229,
   `settingsdetail.go` 1091, `background.go` 678, `auto.go` 630, `backend.go` 1665, `actions.go` 1165, `store.go` 1016). New code
   must go in **new files** (`internal/ai/*.go`); palette literals must move to new files (e.g. `palettes.go`) so `theme.go` stays
   under the limit.
4. **WhatsApp ToS / suspension risk** (unofficial client, libsignal-linked GPL-3.0-as-a-whole binary): README must keep the
   disclaimer; never test multi-account/AI against the user's main number (secondary only).
5. **View-once saving (§13) is ethically gray by design** — it must stay opt-in/off-by-default with an in-UI warning, or it breaks
   the trust model the "Ethically gray features" page exists to protect.
6. **AI auto-reply can loop / spam**: `auto`'s once-per-sender + cooldown pattern must be extended with per-chat cooldown (60s),
   per-hour cap (10), quiet hours, handover keyword, and stop-on-user-send (master spec §5.3(4)) — none exists for AI yet;
   `seen()` has no rate limiting at all today.
7. **Secrets**: AI keys must never enter `wazzap.log` (which `-debug` fills with raw protocol nodes), screenshots, or
   `accounts.json`. `Redact()` helper required (§4.1).
8. **Windows-only paths**: notifications (WinRT toasts, `internal/notify`), tray, `BlockCapture` (`SetWindowDisplayAffinity`),
   Media Foundation video, `dpi_windows.go` — cross-check every Phase 1/2 change on Linux (`GOOS=linux go build`) and Windows;
   `cmd/memprobe` is Windows-only for real numbers.

### Baseline verification (run from the repo root)

```sh
sh patches/apply.sh            # once after cloning; builds third_party/gio + third_party/typesetting
go vet ./... && go build ./... # full check (go is /usr/local/go/bin/go in this VM)
go test ./...                  # full suite; UI tests use a fixed clock (internal/ui/main_test.go)
go run ./cmd/wazzap -demo      # fake chats, no network — manual UI QA
go run ./cmd/screenshot -readme -w 1280 -h 800 -scale 1.25 -out docs/screenshots
go run ./cmd/memprobe -demo -scroll 2000 -size 1706x1040 -chat   # Windows memory/frame benchmark
```

Verified this audit: `go vet ./internal/accounts/... ./internal/auto/... ./internal/command/... ./internal/klipy/...
./internal/memtrim/... ./internal/offheap/... ./internal/photo/...` exits 0 (80s, module cache warm). Full `go test ./...` / `go
build ./...` were **not** run in this audit (time); run them before Phase 1 lands.

## 10. WhatsApp Web look — concrete deltas (Phase 1b)

Measured against web.whatsapp.com (2025 redesign; values below are the web client's documented palette — verify each against a
screenshot pair with `go run ./cmd/screenshot -compare shot.webp -view chats` before changing a literal; the repo's own values are
quoted from `theme.go` and are exact).

| Area | WhatsApp Web | This app today | File(s) to change |
|---|---|---|---|
| App frame | Web has **no native title bar / no left icon rail**: a single header row over the chat list; nav is a slim icon strip only in the 2025 redesign. App keeps a 68dp icon rail + custom title bar | `app.go:736 layoutWindow`, `rail.go:15 railWidth=68dp`, `layoutRail` | Keep the rail (it is the app's established navigation, and the master spec says keep the 3-pane layout) but align its metrics: rail icons 24dp already match Web's 24px; the rail background is `pal.Frame` while Web's side panel is the same colour as the list (`#111b21` dark / `#ffffff` light) — consider `Panel` for the rail. |
| Dark panel / header | Web dark: app bg `#111b21`, header/panels `#202c33`, chat bg `#0b141a`, input `#202c33` | Dark: `Frame #1d1f1e`, `Panel #171717`, `ChatBg #171717`, `Composer #242626`, `Search #2e2f2f` (`theme.go`) | `theme.go` dark literals: panels are ~2 steps darker and greener on Web (`#111b21` is blue-black, `#1d1f1e` is neutral). Shifting `Panel`/`Frame`/`ChatBg` toward `#111b21`/`#0b141a` is the single biggest "looks like Web" change; it also affects `Doodle #292c2b` and `BubbleIn #242626` (Web in-bubble `#202c33`). |
| Light panel / header | Web light: app bg `#f0f2f5`, panels `#ffffff`, chat bg `#efeae2` (doodle tint), input `#ffffff` | Light: `Frame #f7f5f3`, `Panel #ffffff`, `ChatBg #f5f1eb`, `Composer #ffffff` (`theme.go`) | Light is already close; `Frame #f7f5f3` → `#f0f2f5` and `ChatBg #f5f1eb` → `#efeae2` would match Web exactly. |
| Outgoing bubble | Web: light `#d9fdd3`, dark `#005c4b` (solid, saturated) | Light `#d9fdd3` ✓; dark `#254d39` — much darker/desaturated than Web | `theme.go` dark `BubbleOut`: `#254d39` → `#005c4b`, with `MetaOut`/`SecondaryOut`/`QuoteOut` re-mixed (`chatmenu.go:themed` mixes from `BubbleOut`, so a change propagates to chat themes). |
| Accent green | Web `#00a884` (buttons, ticks, active states) | Dark `Green #5dbf6e`, light `#1daa61` | `theme.go` `Green`/`OnGreen`: dark `#5dbf6e` is lighter than Web's `#00a884`; unifying on `#00a884` (already the Teal chat-theme hue, `chatmenu.go:160`) aligns ticks, send button, rail active dot. |
| Chat-list width | Web: list is ~30–40% of window, min ~300px, with a 1px divider `#e9edef`/`#222d34` | `listWidth`: 37.2% clamped `[280dp, 430dp]` (`split.go:80`); divider `PanelBorder` | Already in range; Web's list rows are 72px tall with 12px avatar gaps — compare `settingsGeom` (72dp rows) and `layoutChatRow` (`sidebar.go:295`) against a Web screenshot at 1.22 px/dp (`cmd/screenshot -compare`). |
| Conversation header | Web: 60px header, avatar 40px, name 16px semibold, subtext 13px, icons 24px | Header built in `layoutConvHeader` (`conversation.go`, via `layoutConversation`); avatar 40dp-class, name via `u.label(15…)`-class sizes | Verify header height/icon spacing against Web; the header currently has no bottom divider distinct from `PanelBorder` — Web's is `#e9edef` light / `#222d34` dark. |
| Composer | Web: rounded input (`#ffffff`/`#202c33`, radius ~8–24px), attach/emoji/mic icons outside the field, send button appears on text | Composer is a floating box over the wallpaper (`layoutComposer`, `compose.go:642`; `Composer` colour, radius via `fillRRect` in composer code) | Web's composer is a single rounded bar; ours floats a box + separate buttons. Align radius/padding to Web's 24px pill for the text field and keep the attach/emoji buttons outside it — a `compose.go` layout change, no new state. |
| Bubbles | Web: radius 7.5px, tail on first-in-run only, max-width ~65% of pane, meta inside bottom-right | Bubbles drawn in `conversation.go`/`bubbles` code with `fillRRect`/`paintRRect` (see AGENTS.md gotchas); runs grouped (`layoutConversation` rows) | Measure radius/max-width against Web screenshots; the grouping logic (`first` flag, `conversation.go:100-140`) already matches Web's tail rule — only radii/spacing need tuning. |
| Empty pane | Web: large illustration + "WhatsApp Web" heading + encryption note + (in 2025) a download-app CTA | `layoutEmpty` (`conversation.go:1815`) / `emptyPane` (`app.go:923`): icon + title + 2 lines, no CTA | See §7(5). |
| Date chip / unread | Web: date chip is a centred pill (`#ffffff`/`#182229`, shadow); unread divider is a full-width band | `DateChip`/`DateChipText`, `UnreadBand` in `Palette`; `unread.go` divider | Colours already tokenised; only the chip's shadow/elevation differs (Gio has no elevation — fake with `Shadow` colour). |

Net: the Web look is **mostly a `theme.go` literal change** (dark panels, `BubbleOut`, `Green`, light frame) plus small geometry
tuning in `rail.go`/`split.go`/`compose.go`/`conversation.go` — no redesign, no new dependencies; `cmd/screenshot -compare` is the
verification tool. Do not copy WhatsApp's logo or doodle artwork.

## 11. Themes — full `Palette` audit + new variants

### 11.1 Every `Palette` field (`internal/ui/theme.go`, 229 lines)

Grouped as declared (69 fields; slices noted): frame/chrome `Frame, FrameText, PanelBorder, Divider, Panel, ChatBg, Hover,
Selected, Search`; chips `Chip, ChipBorder, ChipText, ChipActive, ChipActiveBorder, ChipActiveText`; text/icons `Text,
TextSecondary, Icon, IconActive, IconStrong, Green, OnGreen, RailActive, RailSeparator`; chat `Doodle, BubbleIn, BubbleOut,
MetaIn, MetaOut, TickRead, Link, TextOut, SecondaryOut, QuoteIn, QuoteOut, BubbleLine, BubbleButton, CodeBg, MentionPill,
Selection, DateChip, DateChipText, UnreadBand, Encryption, EncryptionText, Composer, ComposerHint`; avatars `GroupAvatar,
GroupAvatarIcon, UserAvatar, UserAvatarIcon, AvatarBgs []color.NRGBA, AvatarFgs []color.NRGBA, GeneralAvatar`; bars/menus `Banner,
BannerText, QRFg, Menu, MenuHover, Shadow, CloseHover, Senders []color.NRGBA`; misc `EmptyIcon, RowHover, Danger, DangerSoft,
RingViewed, Verified, AnnounceBg, AnnounceIcon, ChannelAvatar, ChannelAvatarIcon, StatusBg`; overlays `Popup, PopupBorder,
PopupHover, PopupDivider, PopupSub, Picker, PickerTab, PickerTabBorder, Viewer, ViewerDivider, ViewerThumb, ViewerArrow, Scrim,
Dialog, Toast, ToastText`.

### 11.2 Current variants (key literals; full values in `theme.go`)

| Token | Light | Dark |
|---|---|---|
| `Frame` / `Panel` / `ChatBg` | `#f7f5f3` / `#ffffff` / `#f5f1eb` | `#1d1f1e` / `#171717` / `#171717` |
| `BubbleIn` / `BubbleOut` | `#ffffff` / `#d9fdd3` | `#242626` / `#254d39` |
| `Text` / `TextSecondary` | `#0a0a0a` / `#667781` | `#fafafa` / `#a5a7a8` |
| `Green` / `OnGreen` | `#1daa61` / `#ffffff` | `#5dbf6e` / `#0a1a0e` |
| `TickRead` | `#53bdeb` (both) | `#53bdeb` |
| `Composer` | `#ffffff` | `#242626` |
| `Doodle` | `#e8e1d8` | `#292c2b` |
| `Viewer` / `StatusBg` | `#ffffff` / `#0f0f0f` | `#171717` / `#0f0f0f` |

Selection/persistence/application: pref `theme` (`"light"`/`"dark"`/`""`=system), `setTheme`/`applyTheme`
(`settingsdetail.go:770-795`), `SetDark` (`app.go:605-611`) swaps `u.pal` between the two package-level `Palette` values —
**instant, no restart, no window rebuild** (only `u.pal`/`u.dark` change; a frame redraw picks it up). System theme is re-read on
focus (`host.go:635`). Per-chat tinting overlays via `themed()`/`chatPalette` (§6.1) without touching the base palette.

### 11.3 What adding themes needs

1. **New `Palette` values, not new fields.** Dark OLED, Navy, Forest, Sepia, WhatsApp Green are all expressible with the existing
   69 fields — add e.g. `oledPalette`, `navyPalette`, `forestPalette`, `sepiaPalette`, `waGreenPalette` literals. OLED:
   `Frame/Panel/ChatBg/Viewer/Dialog/Popup/Picker = #000000` (true black; `BubbleIn` must stay visibly above black, e.g.
   `#111111`–`#161616`, `Hover/Selected` lifted e.g. `#1c1c1c`, or rows vanish; `Scrim`/`Shadow` need opacity, not colour, for
   separation). Navy: dark-blue family (`Panel #0e1a2b`-class, `ChatBg #0a1622`-class, deep teal-blue `BubbleOut`). Forest: dark
   green-grey; Sepia: warm paper light theme (`Panel #fdf6e3`-class, brown-tinted text); WhatsApp Green: the *saturated* light
   variant (`#00a884` accents, `#d9fdd3` bubbles).
2. **Selection model.** Today `prefTheme` holds one of three strings and `applyTheme` branches on it. Extend the value set
   (`"oled"`, `"navy"`, `"forest"`, `"sepia"`, `"wagreen"`) and replace the bool `dark` branch with a resolver `func
   paletteFor(name string, systemDark bool) *Palette` + a `dark bool` derived per palette (OLED/Navy/Forest dark; Sepia/WA Green
   light) — `u.dark` is read in ~6 places (`viewonce.go:118`, `conversation.go:783,1427`, `memberlabel.go:91`,
   `sidebuttons.go:168`, `chatmenu.go:231`), so it must stay correct per palette. `themeNames` (`settingsdetail.go:764`) and the
   Chats settings radio rows (`:253-257`) gain the new entries; the page should move to Settings > Appearance per master spec §3b
   (today theme lives under Settings > Chats, `settingsdetail.go:299`).
3. **Per-account + global default.** `prefTheme` is currently in `appPrefs` (global, carried on switch). Split it: a global
   default pref (stays in `appPrefs`, e.g. keep `theme` as the default) + a per-account override pref (e.g. `theme_acct`, **not**
   in `appPrefs`, stored in that account's `wazzap.db` like any other account pref). Resolution: account override if set, else
   global default, else system. `applyTheme` must run on `useBackend` (it already runs via `newUI` → `loadPages`/`applyTheme` at
   `app.go:369` — verify the per-account value is read after the new backend is installed, not before).
4. **Instant switch without restart** already works (`setTheme` → `applyTheme` → next frame); new palettes inherit it for free
   *provided* no code caches a colour outside `u.pal` across frames. Audit points: `memo.go` `roundShape` (shapes recorded per
   size, colour applied at draw — safe), `icon` cache (per size+colour — safe), `wallpaper` tile (keyed by `bg, col, cell` —
   re-rasterizes on change — safe), `glyphs` cache in `dropCaches` (colour-keyed — safe). `themed()` copies `*p` per frame in
   `chatPalette`, so a palette swap propagates automatically.
5. **File-size rule**: `theme.go` is already 229 lines (>200). New palette literals must go in new files (e.g.
   `internal/ui/palettes_dark.go`, `palettes_light.go`) with `theme.go` keeping only the struct + resolver; each new file under
   200 lines.
6. **Contrast check**: every new palette must keep `Text`/`TextSecondary` vs `Panel`/`BubbleIn`/`BubbleOut` readable (WCAG AA-ish
   for 14sp text) — add a test asserting contrast ratios for each palette (new `theme_test.go`; no test exists for palettes
   today).

## 12. Custom wallpaper (Phase 1b)

### 12.1 How wallpaper works today

- **Only a doodle toggle, no choice.** `internal/ui/wallpaper.go` (164 lines): `wallpaper{key wallKey, tile paint.ImageOp, size
  int}`; `layout(gtx, bg, doodle color.NRGBA, doodles bool)` fills with `bg`, or tiles a pre-rendered doodle tile (`renderTile`,
  62dp cells, 7×7 grid, seeded `rand.NewSource(11)` so the pattern is stable; glyphs wrap at tile edges for seamless repeat;
  `flatten` pre-mixes the tile opaque in linear light because Gio blends in linear space). The tile is rasterized **once per (bg,
  doodle, cell)** and reused — a frame costs a handful of image draws. `u.doodles` (pref `doodles`, on unless `"off"`,
  `settingsdetail.go:26,301`, in `appPrefs`) is the only control; no image, colour choice, per-chat setting, or opacity.
- **Call site**: `layoutConversation` (`conversation.go:172`): `u.conv.wallpaper.layout(gtx, u.pal.ChatBg, u.pal.Doodle,
  u.doodles)` — drawn per conversation frame, under the message list, composer floating over it. The per-chat theme (`themed()`,
  `chatmenu.go:205-232`) already tints `ChatBg`/`Doodle` per chat — the existing per-chat wallpaper mechanism (colour-only).
- **No storage**: nothing about wallpaper is persisted except the global `doodles` pref. No file, no cache, no loader.
  `internal/ui/background.go` is **background accounts, not wallpaper** — do not confuse the two; the wallpaper-relevant files are
  `wallpaper.go`, `conversation.go:145-175`, `chatmenu.go:157-232` (per-chat theme), `settingsdetail.go:299-302` (doodles toggle),
  `theme.go` (`ChatBg`, `Doodle`).
- **Image loading/caching patterns to reuse**: `imageCache` (byte-bounded, `perf.go` `prefImageCache`), `u.images`/`u.emojiImgs`
  (`endFrame` trimming in `layoutWindow`), `offheap.NewRGBA` for big pixels, `photo.Shrink` for downscaling,
  `mediaPath`/`mediaFilePath` (`wa/media.go:275`, `wa/files.go:92`) for per-account media files under the account data dir (0600
  files, 0700 dirs — the mode wallpaper files should use).

### 12.2 What per-chat + global wallpaper needs

1. **Storage**: per-chat choice in prefs, following the `prefChatTheme = "theme:"` precedent (`chatmenu.go:169`): e.g.
   `wallpaper:<chatID>` = `doodle|solid:#RRGGBB|image:<filename>|default`, plus a global default pref (in `appPrefs`, e.g.
   `wallpaper`) resolved when the per-chat value is `default`/empty. Image file in the account data dir (e.g.
   `<data>/wallpaper/<chatID>.jpg` or a hash name), **never** in `accounts.json`. Per-account by construction (each account has
   its own `wazzap.db` + data dir) — no cross-account leak if the filename is namespaced by chat ID and the pref lives in the
   account DB.
2. **Ingest**: file picker (`internal/filepick`, already used by `pickStatusMedia`/`PickImage`) → `photo.Shrink` downscale to a
   capped size (suggest ≤1920px long side, JPEG q80 via `photo.Prepare` with `QualityStandard` — or a dedicated wallpaper size;
   **never keep full-res in memory**), write 0600, store only the filename in the pref. Cap total wallpaper bytes per account
   (suggest a few MB; delete the file when the choice changes or the chat is deleted — there is no cleanup hook today).
3. **Rendering**: extend `wallpaper` to hold an optional image tile (decoded once, off-heap via `offheap.NewRGBA`, keyed by
   filename+size like `wallKey`); draw it `cover`-cropped to the pane, then the doodle tile on top at reduced alpha, then an
   optional dimming veil. Dimming/opacity slider: a per-wallpaper opacity pref (e.g. `wallpaper_dim:<chatID>` 0–100, or one global
   slider) applied as a `ChatBg`-coloured veil with alpha over the image *before* bubbles draw — reuse the `veil()` helper rather
   than a full-window `PushOpacity` layer (AGENTS.md: opacity layers pin a screen-sized texture). Solid colour = `layout` with `bg
   = chosen colour`, doodles off; doodle = today's path.
4. **Readability**: bubble colours come from `u.pal` and are opaque, so a busy image cannot wash out text inside bubbles — but the
   date chip, unread band, and system rows sit directly on the wallpaper, so the dimming default must keep their contrast (suggest
   default dim ≥30% for uploaded images, and verify `DateChip`/`UnreadBand` against the dimmed image in `cmd/screenshot
   -compare`). The chat-theme tint (`themed()`) and wallpaper must compose: theme tints `ChatBg`/`Doodle`; an uploaded image
   replaces `ChatBg` as the base — decide that image wins over theme tint for the background, while theme tint still applies to
   `BubbleOut`.
5. **UI**: a "Wallpaper" row in the chat's ⋮ menu / info panel reusing the theme dialog pattern (`openChatTheme`/`themePanel`,
   `chatmenu.go:231-290`, dialog `dialogTheme`) plus a global default picker in Settings > Chats next to the doodles toggle;
   preview swatches via `themeSwatch`-style mini-wallpapers. Settings search must index it (`settingsSearchHits`).
6. **Memory**: one decoded wallpaper per open chat (not per chat in the list); drop it in `dropCaches()`/ on conversation switch;
   `memtrim.Trim()` after changing wallpaper. Acceptance: wallpaper persists after restart (pref + file), RAM does not regress vs
   upstream (`cmd/memprobe` before/after).

## 13. View-once (Phase 1c)

### 13.1 How view-once works today

- **Model/backend**: `model.KindViewOnce`, `Message.Opened`, `Message.OnPhone`. Linked devices get no ciphertext for view-once
  media — hypermeow delivers `events.UndecryptableMessage` with `IsUnavailable && UnavailableType == ViewOnce`;
  `wa.Backend.onUndecryptable` (`backend.go:1086-1124`) stores a `KindViewOnce` placeholder (media kind from `Info.MediaType`). A
  reply quoting it often carries its media key: `fillViewOnce` (`store.go:468-471`) fills the placeholder's media from the quote
  (`backend.go:1196`, during `apply`). Verified by `internal/wa/viewonce_test.go` (`TestViewOnce`): unavailable → `KindViewOnce` +
  `OnPhone`; keyless envelope → same; quoting reply → media filled (`mediaBlob` non-empty) + `OnPhone` cleared; `OpenedViewOnce`
  persists `opened`. Other undecryptables become `KindWaiting` (`TestWaiting`, same file).
- **Opening**: `UI.openViewOnce` (`viewonce.go:48-77`): voice → `toggleVoice` in place; photo/video → `openViewer(m)` with
  `viewer.items = [m]`, `viewer.viewOnce = true`; own messages never open; `OnPhone`/ no-media → toast directing to the phone;
  already-opened → toast (unless replay). On open, `backend.OpenedViewOnce(m)` (`actions.go:585-590`) sets `opened` in
  `wz_messages` — **unless** `viewOnceReplay` is on. `canOpenViewOnce` (`viewonce.go:40-46`): `!OnPhone && Media != None &&
  (viewOnceReplay || (!Opened && !FromMe))`.
- **Rendering**: bubble card `layoutViewOnceCard` (`viewonce.go:124-171`, ring icon `viewOnceRing`, label from `viewOnceLabel`:
  Photo/Video/Voice message/"Opened"). Viewer `layoutViewer` (`viewer.go:171-375`): when `v.viewOnce` the toolbar is reduced to
  **only the close button** (`tools = tools[len(tools)-1:]`, `viewer.go:228-230`, "nothing to keep it by") — no zoom, go-to,
  reply, star/pin/react, forward, or **save** (the `"save"` tool, `icDownload` → `b.SaveMedia(m)`, exists for normal media,
  `viewer.go:213`), no thumbnail strip (`viewer.go:322-328`), no prev/next arrows (`viewer.go:370`).
- **Screenshot blocking**: `layoutViewer` line 173: `u.blockCapture(u.privacy.noCapture || v.viewOnce && (v.open || v.anim.v > 0)
  && !u.viewOnceReplay)`. `UI.blockCapture` (`viewonce.go:184-198`) calls `desktop.BlockCapture(hwnd, on)` once `hwnd` is known
  (skips while unknown; next frames retry). `desktop.BlockCapture` (`desktop/capture_windows.go:23`, `SetWindowDisplayAffinity`;
  `capture_other.go` is a no-op) — **Windows-only**; on other OSes there is no blocking, and the opt-in Save must not assume
  blocking exists elsewhere. The same `blockCapture` also serves privacy mode (`prefNoCapture`, `privacy.go:34`).
- **Replay (gray)**: pref `model.PrefViewOnceReplay = "view_once_replay"` (`model.go:1277`), switch on the Ethically gray features
  page (`extras.go`); when on, a view-once message opens repeatedly **and screenshots are allowed** (`!u.viewOnceReplay` in the
  `blockCapture` condition). This is the existing precedent for a gray view-once switch — the new Save opt-in should sit beside
  it, not inside it.

### 13.2 Where Save would go (audit only — do not implement)

1. **Setting**: Settings > Privacy, OFF by default. The Privacy page is `privacySettings()` (`settingsdetail.go:613-682`):
   sections "Who can see my personal info", "Disappearing messages", "Groups", "Blocked contacts", "Advanced" (currently only
   "Disable link previews"). Add a row there (or a new "View once" section) backed by a new pref, e.g. `view_once_save` (suggest
   the `extraOn`/`setExtra` `"on"`/`"off"` pattern from `extras.go:38-48`, **not** a gray-command pref — it is a privacy consent,
   and it must default off). It must **not** be in `appPrefs` (per-account consent) and must be read live in `layoutViewer`, not
   cached at startup. In-UI warning text belongs on the same row ("The sender expected this to disappear…").
2. **Save button**: `layoutViewer`'s `tools` slice (`viewer.go:213-249`). Today `if v.viewOnce { tools = tools[len(tools)-1:] }`
   keeps only close. Change to: if `v.viewOnce && saveAllowed`, keep `{"save", icDownload, …, run: saveViewOnce}` + close; else
   close only. The button must be labelled unambiguously (Save, not the generic download icon alone) and, on tap, show the warning
   again (dialog via `u.dialog`, cf. `confirmLogout`) before writing.
3. **Save implementation point**: reuse `wa.Backend.SaveMedia(m)` (`actions.go:666-695`) as the template — it reads the local
   media cache (`mediaPath`/`mediaFilePath`), else downloads via `cli.DownloadAny(ctx, mediaMessage(media, blob))` from the stored
   `mediaBlob`, then `saveDownload(fileName(m, media, blob), data)` into Downloads and toasts "Saved to …". For view-once the
   media bytes are the same `mediaBlob` (`fillViewOnce` guarantees it is present when the viewer could open it; `OnPhone` items
   have no blob and must not offer Save). Differences from `SaveMedia`: destination should be the **account's media folder** per
   master spec §3c (decide: `mediaPath` dir vs Downloads), the call must be gated on the pref (check in UI *and* in `wa`, defence
   in depth), and it must never run automatically — no call from `onUndecryptable`/`fillViewOnce`/`apply`. Video view-once uses
   the same viewer (`isVideo(m)` → `layoutVideoControls`, `videoview.go`) — Save must work for video too.
4. **What must not change when OFF**: exactly upstream — no Save tool, `blockCapture` unchanged, `OpenedViewOnce` still marks
   opened on first view. Also note: saving does **not** un-mark opened; the sender is not notified by this app (WhatsApp has no
   such signal for linked devices).
5. **Tests to add in Phase 1c**: extend `viewonce_test.go` (pref-gated save path needs a `wa`-level test with a fake `mediaBlob` +
   a UI test that the tool list contains/omits Save); screenshot `-overlay viewer` with a view-once message in both states.

## 14. HD status (Phase 1c)

### 14.1 How status posting works today

- **UI entry**: Status page ⊕ / "My status" → `openStatusAdd` → `statusAddItems` (`statuspost.go:52-67`): "Photos & videos"
  (`pickStatusMedia` — file dialog, photos+videos), "Text" (`openStatusText` — full-window composer, 700 chars max, `statusColors`
  backgrounds), plus "Audio" for group status. `statusFiles` (`statuspost.go:31-50`) keeps only photo/video (or audio for group
  status) and toasts otherwise. Photos/videos go through the **send view like files for a chat**, with `statusChatID` as
  destination (`postingStatus()`, `statuspost.go:24-28`); text posts via `postStatusText` (`statuspost.go:124-132`) →
  `backend.PostStatus(StatusPost{Text, Background})`.
- **Send path (shared with chat)**: `sendAttachments` (`attach.go:672-700`): for status files it calls `queueSend(statusChatID, f,
  Draft{caption})` **without** opening an album; `queueSend` (`editrender.go:136-145`) sets `it.att.Quality = a.quality` (the send
  view's quality, see §14.2) and appends to `a.outbox`; the outbox worker sends via `Backend.SendFile` for chats, and for status
  the `Attachment` (with its `Quality`, `model.go:521-522`) becomes `StatusPost{File: *Attachment}` (`model.go:736-741`). So **the
  HD toggle in the send view already flows into status posts** — verify the outbox worker actually passes `Quality` through to
  `PostStatus` (it does via the `Attachment`; confirm in Phase 1c with a test — this is the crux of "reuse, do not duplicate").
- **Backend**: `wa.Backend.PostStatus(p)` (`statuspost.go:22-131`): text → `ExtendedTextMessage` with `BackgroundArgb`/`Font:
  SYSTEM`, stored in `wz_status` at once (`storeStatus`) so it shows in your own thread while uploading, sent by `sendStatus` in a
  goroutine. Media: `model.Message` (`KindImage`, `ChatID: statusChat`), `upload{path}`; **image → `up.prepareImage(a.Quality)`**
  (`files.go:383-391` → `photo.Prepare(data, q)`); video/audio → `os.Stat` only (no transcode, no quality concept — duration via
  `mp4Seconds`); then `fileMessage` + `up.push` (30-min timeout), `statusFailed` on error, media blob kept in
  `wz_status.media_blob` "so a reply to it can quote it". Pending statuses are marked in `wz_meta` (`pendingStatus`) and dropped
  on next start if unsent (`dropPendingStatuses`, from `Start`). Viewing: `wa/status.go`, UI `statusViewer` (`status.go:341-516`,
  picture 6s `statusDuration`).
- **Limits today**: image — whatever `photo.Prepare` does for the chosen `Quality` (see §14.2); video — no duration/size cap in
  `PostStatus` itself (no 60s/90s check found in `wa/statuspost.go` or `ui/statuspost.go` — WhatsApp's server enforces its own
  limits; a too-long video fails at `up.push`/ `sendStatus` with a generic notice). No estimated-size display anywhere in the
  status flow today.

### 14.2 The photo HD path to reuse (already exists for chat)

- `model.Quality` (`model.go:539-549`): `QualityStandard` (fit 1600px, JPEG q80), `QualityHD` (fit 4096px, JPEG q90), `QualityRaw`
  (JPEG/PNG as-is). `photo.settings` (`photo.go:21-25`) is the single source of truth; `photo.Prepare`/`Estimates` decode once and
  estimate all three qualities.
- **UI**: send-view tray HD button (`sendview.go:68-88`, `icon.Hd`/`icon.Raw` by quality) → `openQualityMenu`
   (`attach.go:299-305`) → `qualityMenuItems` (`attach.go:340-361`): "Standard quality" / "HD quality" (+ "Raw quality" when the
   `raw_photos` extra is on), each with `qualitySub(q)` (`attach.go:364-384`) showing **"116 kB · 1600 x 900"**-style estimated
   size — this is the estimated-size UI the HD status toggle needs, already built. `a.quality` lives on `attachState`
   (`attach.go:48-51`); estimates are computed per photo in the background (`a.est`, `openQualityMenu` starts estimation).
- **HD status work (audit direction)**: (1) confirm the status send view shows the same HD tray button (it uses the same send
  view, `postingStatus()` — check `sendview.go` does not hide the quality button for status destinations; if it does, that is the
  one-line gap); (2) default status quality: today it inherits whatever `a.quality` was last set to in chat (shared `attachState`)
  — decide an explicit status default (suggest Standard, with the toggle remembered per status post, not per chat); (3)
  `PostStatus` already honours `a.Quality` for images via `prepareImage` — no `wa` change needed for photos; (4) **video has no HD
  concept**: videos are sent as-is (`os.Stat` + `push`), so video status is already effectively HD; the work is to *show* that
  (estimated size = file size) and to not add a transcode step; (5) surface the estimate: reuse `qualitySub` text in the status
  send view's tray; (6) HD = `QualityHD`'s 4096px/q90 cap, **not** Raw — Raw stays behind the `raw_photos` extra and must not
  become the status default (a 12 MP phone photo as Raw would be multi-MB per status).
- **Media editor**: `internal/ui/mediaedit.go` (941 lines) is the photo editor (crop/filter/draw/text); it works on the picked
  file before `queueSend` and does not itself choose quality — quality is chosen in the send-view tray afterwards. No mediaedit
  change is needed for HD status; if the editor ever downscales on load, that would silently defeat HD — verify `mediaedit.go`
  loads full-res (it renders edits via `editrender.go`, which re-renders from the original path at send time; confirm in Phase
  1c).

## 15. QA notes (add to `docs/qa-ui.md` in Phase 1; checklist)

Baseline (master spec §6 + §9): `go vet ./...`, `go test ./...`, `go build ./...` (Windows + Linux cross-compile), `go run
./cmd/wazzap -demo` for UI, real link with a **secondary number only** for multi-account + AI. Record startup time, idle RAM after
10s idle (`cmd/memprobe`), RAM with 2 accounts, RAM during AI streaming, in the PR description.
1. **Account switching**: link 2 accounts (secondary numbers); switch A→B→A; send from each; set B to `Always`, send to B while A
   is open, confirm badge + notification labelled with B's name; switch to B via notification click; confirm drafts are discarded
   today — and after Phase 2, preserved per account.
2. **WhatsApp Web look**: `go run ./cmd/screenshot -compare shot.webp -crop 0,0,2000,1250 -scale 1.22 -view chats` against a
   web.whatsapp.com screenshot (dark + light); check panel colours, `BubbleOut`, accent green, rail, list width, composer pill,
   header height. Repeat after every `theme.go` change.
3. **Themes**: switch Light → Dark → Dark OLED → Navy → Forest → Sepia → WhatsApp Green from Settings > Appearance; each must
   apply instantly with no restart and no visual glitch (watch wallpaper tile, viewer, dialogs, status viewer); set a per-account
   theme on B, switch to A, confirm A keeps its own theme; restart, confirm both persist; check `Text` contrast on
   `BubbleIn`/`BubbleOut` in OLED at 100%/200% zoom.
4. **Custom wallpaper**: set a global default wallpaper (doodle, then solid colour, then an uploaded photo); open a chat, confirm
   it shows; set a different per-chat wallpaper on one chat, confirm other chats keep the default; move the dimming slider,
   confirm date chip/unread band stay readable in every theme; restart, confirm both persist; upload a 12 MP photo, confirm RAM
   does not regress (`cmd/memprobe`) and the stored file is downscaled, not full-res.
5. **View-once**: opt-in OFF (default) — open a view-once photo and video, confirm no Save button, screenshot blocked on Windows
   (Print Screen / Snipping Tool must come out black), message shows "Opened" afterwards. Opt-in ON in Settings > Privacy — reopen
   a fresh view-once message, confirm Save appears with the warning, save it, confirm the file lands in the account's media
   folder; confirm replay (`view_once_replay`) still allows re-opening and screenshots. Secondary number only.
6. **HD status**: post a photo status in Standard and in HD, confirm the tray shows the estimated size for each and the posted
   status visibly differs in sharpness (compare on the phone); post a video status, confirm it is sent as-is with its file size
   shown; confirm a text status is unaffected; verify the same HD toggle works for sending the same photo in chat (reuse, not
   duplicate).
7. **AI (Phase 3, for `docs/qa-ai.md`)**: `/ai` with a fake OpenAI-compatible server (`httptest`) and with Ollama
   `http://localhost:11434/v1`; wrong key → 401 message; unknown model → 404; rate limit → 429 with retry-after; auto-reply only
   in allowlisted chats, cooldown/cap respected, ghost mode blocks AI sends.
