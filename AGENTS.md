# Fork WazzapClients — Master Prompt untuk AI Agent

> Cara pakai: fork dulu `Chomosuke9/WazzapClients` ke GitHub lo, clone fork lo, jalankan `sh patches/apply.sh`, lalu paste **Master Prompt** di bawah ini ke AI agent lo (Claude Code / Codex / Cursor / dll). Jangan skip Phase 0.

---

## MASTER PROMPT (copy-paste dari sini)

You are a senior Go desktop engineer. Fork and upgrade this repo, do not start from scratch.

### 0. Repo truth — read this first, do not assume

- Repo: fork of `Chomosuke9/WazzapClients`, module `github.com/chomosuke9/wazzapclients`, Go `1.27.0`.
- Stack: Go + Gio (`gioui.org v0.10.3`) + `github.com/polymorfa/hypermeow` (performance fork of `tulir/whatsmeow`, branch `main`). **Do NOT use upstream `go.mau.fi/whatsmeow`.**
- Storage: `modernc.org/sqlite`, messages in local SQLite (`wazzap.db` per account data dir).
- UI: Gio immediate-mode GPU UI, no WebView, no Electron, no Chromium, no Node, no HTML/CSS/JS. Single Go binary.
- **First read `AGENTS.md` (58KB, source of truth) in full before writing any code.** Gio is pre-1.0, its API changes between minors. Check `pkg.go.dev/gioui.org` for the pinned version in `go.mod`, and read the actual Gio / hypermeow source when unsure. Do not rely on training-data examples.
- Gio and go-text are patched: run `sh patches/apply.sh` after cloning. It builds `third_party/gio` and `third_party/typesetting`. **Never edit `third_party/` directly** — change them via `patches/*.patch` only.
- Entry: `cmd/wazzap/main.go`. Core packages:
  - `internal/wa/` — `Backend` wrapping hypermeow, `Send(chatID string, d model.Draft)`, media, groups, polls, status, etc.
  - `internal/model/` — `model.Message`, `model.Draft`, `model.Backend` interface, prefs.
  - `internal/ui/` — all Gio UI: `app.go`, `conversation.go`, `settings.go`, `theme.go` (`Palette`), `accounts.go`, `slash.go`, `composertext.go`, `scrollbar.go` (`u.scrollList`), `login.go`, `notifications.go`.
  - `internal/accounts/` — `accounts.go`: `Account{Dir, ID, Name, Phone, Background Mode}`, `List{Active string, Accounts []Account}`, `Load/Save/Add/Remove/Find/Current/Others/Path`.
  - `internal/auto/` — `auto.go`: `Backend` wraps `model.Backend` for `/schedule` (`prefJobs="auto_scheduled"`) and `/afk` (`prefAway="auto_away"`). It sees every event/message, persists in backend prefs, works with window closed / after restart. Respect ghost mode (`model.PrefGhost`): it refuses user-like sends.
  - `internal/klipy/` + `internal/ui/gifs.go` — **the pattern to copy for API keys**: key entered in Settings > Extra features, validated against the provider, error shown in UI, never bundled.
  - `internal/memtrim/`, `internal/offheap/`, `internal/photo/` — memory discipline. Read them before touching images/caches.

### 1. Hard constraints (non-negotiable)

1. No WebView2, no Electron, no Chromium, no bundled browser, no web frontend. Keep it native Gio.
2. Keep idle RAM low. Do not regress upstream idle memory. Keep caches capped, keep `memtrim` behaviour (hand memory back after ~10s idle), keep big pixel buffers off-heap, do not keep all chats/media in memory, paginate / virtualize lists.
3. No new heavy dependencies. Prefer stdlib `net/http`. No new CGo deps unless justified and optional.
4. No breaking change to existing single-account data: an install with one account must keep working, same data dir, same `accounts.json`, same `wazzap.db`.
5. No secrets in git, logs, or screenshots. API keys are never logged, never printed, never sent anywhere except the user-configured AI endpoint.
6. No telemetry, no proxy server, no third-party server of ours. The app talks directly: WhatsApp via hypermeow, AI via the user's endpoint only.
7. Keep existing tests passing: `go vet ./...` and `go test ./...` must pass. Add tests for every new package / behaviour. Follow existing test style in `internal/ui/*_test.go`, `internal/wa/*_test.go`, `internal/auto/*_test.go`.
8. Work in small, reviewable commits, one phase at a time. Do not rewrite the whole UI in one commit. After each phase, report: what changed, files touched, how to test manually, measured idle RAM / startup time before vs after.

### 2. Phase 0 — Audit first (do this before any feature code)

Audit and report in `docs/audit.md` (create it):
- How account switching works today: `internal/ui/accounts.go` (`host.switchAccount`, `reqSwitch`, `reqAddAccount`, `setAccountMode`, `layoutAccountMenu`), `internal/accounts` modes `Off=0`, `Always=-1`, `Checks=[5,15,30,60]` minutes. What happens to the open `wa.Backend`, drafts, notifications, and background accounts on switch?
- How `internal/wa.Backend` connects, reconnects, and emits events. Where is the single place to intercept incoming messages for AI / auto-reply?
- How settings are stored (`model.Pref`), how Settings > Extra features is structured, and how KLIPY key is stored/validated — reuse that exact pattern for AI keys.
- How slash commands work in `internal/ui/slash.go` and how `/schedule` / `/afk` are wired through `internal/auto`.
- List the 10 biggest UI/UX gaps vs official WhatsApp Desktop (layout, density, empty states, loading, search, keyboard, accessibility) with file locations.
- Do not change behaviour in Phase 0. Stop and show the audit for approval before Phase 1.

### 3. Phase 1 — UI/UX upgrade (Gio only)

Goal: look and feel closer to official WhatsApp Desktop, more polished, faster to use — without adding RAM.

- Keep the 3-pane layout (rail / chat list / conversation). Polish, do not redesign from zero:
  - Chat list: clearer unread badges, mute/archive/pin states, avatar initials + photo, last-message preview with sender, timestamp, ticks, typing indicator. Keep virtualized scrolling via `u.scrollList`, never render all rows.
  - Conversation: cleaner bubbles, better spacing/typography, quote/reply rendering, reactions, link previews, date separators, day jumps. Improve empty state, loading skeleton, and error/failed-send states.
  - Composer: keep formatting toolbar, @mentions, emoji/sticker/GIF picker. Improve focus, Enter to send / Shift+Enter newline, draft preserved per chat and per account.
  - Add density setting Compact / Comfortable, font scale (keep existing 80–200%), Light / Dark / System theme via `internal/ui/theme.go` `Palette`. Add design tokens there, do not hardcode colours in widgets.
  - Add keyboard shortcuts and show them in menus: `Ctrl+K` search, `Ctrl+N` new chat, `Ctrl+1..9` switch account, `Ctrl+Shift+M` mute, `Esc` close dialog / clear search. Add a searchable Settings (reuse existing settings search).
  - Improve notifications + tray: notification shows account name + sender, Reply / Mark as read on Windows, tray icon shows total unread across accounts.
  - Performance in UI: follow `AGENTS.md` gotchas — use `internal/ui/icon` (not `widget.Icon`), `fillRRect` / `paintRRect` (not big `clip.RRect`), `memo.go` `roundShape`, avoid whole-window opacity layers, avoid `paint.ImageOp` per animation frame, use `photo.Shrink` for downscaling, keep emoji bitmap handling as in `emojiimg.go` / `styledtext`.
- Acceptance: `go run ./cmd/wazzap -demo` runs, all existing UI tests pass, no new WebView dep, manual QA checklist in `docs/qa-ui.md`.

### 4. Phase 2 — True multi-account

Current state is partial: several accounts can be linked, but only one is open at a time; others are `Off`, `Always`, or periodic check. Upgrade it to true multi-account:

- Account manager: add / remove / rename / reorder accounts, each with avatar, name, phone, unread count. Stored in `internal/accounts` (`accounts.json`, per-account dir `""` for first account, `accounts/<n>` for later ones — keep this scheme).
- Fast switching: switch account without full app restart, preserve drafts per chat per account, preserve scroll position where cheap. Switching must be <1s for a linked account.
- Simultaneous mode (opt-in per account): `Background = Always` keeps that account connected in background; `Off` disconnects when not open; periodic `Checks` connects, syncs, then disconnects. Enforce a memory budget: max 2 `Always` accounts by default (configurable), background accounts must not keep chat history / media decoded in memory, run `memtrim` after switch.
- Unified signals: rail/sidebar account switcher with per-account unread badge, total unread in tray + taskbar, notifications labelled with account name, notification click opens the correct account + chat.
- Isolation: each account has its own `wazzap.db`, session, prefs, AI config, and media cache. Never leak messages, contacts, or drafts across accounts. Logout of one account must not touch others. `Remove` deletes only that account's dir (keep root-dir rule in `accounts.go`).
- Per-account settings: background mode, notifications on/off, AI on/off + AI config (see Phase 3) are per-account, with a global default.
- Acceptance: 2 accounts linked, switch A→B→A, send from each, background `Always` account receives a message while B is open and shows badge + notification with correct account label, idle RAM with 2 accounts measured and reported, `go test ./internal/accounts/... ./internal/ui/...` passes.

### 5. Phase 3 — AI agent integration (custom provider, custom key, custom endpoint)

Goal: provider-agnostic, OpenAI-compatible AI. User brings their own key + endpoint + model. No hardcoded provider, no hardcoded key.

**5.1 New package `internal/ai/` (create it, do not scatter AI code in `ui/` or `wa/`)**

```
internal/ai/
  config.go    // AIConfig, ProviderPreset, validation, storage (0600 file, see 5.2)
  provider.go  // Provider interface, OpenAI-compatible client
  client.go    // net/http, timeouts, SSE streaming, retries, error mapping
  agent.go     // Agent: system prompt, context builder, tools/actions
  agent_test.go
  provider_test.go // httptest fake OpenAI-compatible server, no network
```

- `Provider` interface: `Chat(ctx, req ChatRequest) (<-chan StreamChunk, error)` + `TestConnection(ctx) error`.
- Default implementation: OpenAI-compatible Chat Completions (`POST {BaseURL}/chat/completions`, `model`, `messages`, `temperature`, `stream:true`, SSE `data:` parsing). Must work with:
  - Presets: OpenAI, Groq, OpenRouter, Google Gemini (OpenAI-compatible endpoint), Ollama (`http://localhost:11434/v1`), LM Studio (`http://localhost:1234/v1`), Custom.
  - Custom: user sets `BaseURL` (e.g. `https://api.openai.com/v1`, `https://openrouter.ai/api/v1`, `http://localhost:11434/v1`), `APIKey`, `Model` (free text, e.g. `gpt-4o-mini`, `llama3.1`, `qwen2.5`).
- No SDK dependency. Only stdlib HTTP. Timeouts: connect 10s, request 120s (streaming), cancel via context. Map errors to user text: invalid key (401), model not found (404), rate limit (429, show retry-after), endpoint unreachable, invalid JSON.
- Streaming in Gio: stream tokens into composer / AI panel, with Cancel button, retry, and a visible “AI is generating…” state. Never block the UI goroutine — do network on a worker goroutine, post results to UI like `desktop` / `host` does (never `SendMessage` to window from UI goroutine, see AGENTS.md).

**5.2 Config & secrets**

- Settings > AI (new section, follow KLIPY pattern in `internal/ui/gifs.go`): Enable, Provider preset dropdown, Base URL, API Key (password field, masked, Show/Hide), Model, Temperature, System Prompt (editable, per-account default), Test Connection button.
- Storage: per-account AI config, plus global default. Store in the account data dir, file mode `0600`, JSON. Key is stored as-is (user-owned device, like KLIPY today) — do NOT commit it, do NOT log it, do NOT include it in bug reports / `-debug` output. Add a `Redact()` helper for logs.
- Validation on save: Test Connection must succeed before enabling auto-reply. Show model list hint, but do not require a model-list endpoint (many local servers don't have it).

**5.3 Features (implement in this order)**

1. **Ask AI (`/ai`)**: slash command + AI panel. `/ai <question>` answers using optional context: current chat last N messages (default 20, configurable, only when user explicitly runs it). Show answer in-chat as a local AI message (clearly labelled, not sent to WhatsApp unless user taps Send).
2. **Chat tools (context menu on a message / chat)**: Summarize chat, Summarize unread, Draft reply (3 variants: short / polite / casual), Rewrite selected text, Translate to Indonesian / English. All are suggest-only by default.
3. **Smart replies**: in composer, “✨ Suggest replies” generates 3 short replies from last messages. Tap to insert into composer, user still sends.
4. **AI auto-reply agent (opt-in, off by default)**: hook into `internal/auto.Backend` — reuse its event loop, prefs, and timer, do not create a second auto-reply system. Per-chat toggle + per-account toggle + global toggle (all three must be on).
   - Modes: `Off` / `Suggest only` (default) / `Auto-reply` (allowlist chats only).
   - Rules: never reply to own messages, status, channels, or groups unless @mentioned (configurable); cooldown per chat (default 60s), max replies per hour per chat (default 10), quiet hours, keyword to hand over to human (e.g. `human`, `admin` stops auto-reply for that chat), stop on user sending a message in that chat.
   - System prompt per account + per chat override, include: account name, chat name, recent context (max N messages, no media bytes, only captions), language (follow chat language, default Indonesian).
   - Every auto-reply is logged locally with timestamp, prompt hash (not full prompt), and model. Show “🤖 Auto-reply on” indicator in that chat.
5. **AI in search**: optional “Ask about this chat” from chat search, same context rules as (1).

**5.4 Privacy & safety**

- AI is opt-in per chat. No chat history is sent to any AI endpoint unless the user triggers (1)–(3) or explicitly enables (4) for that chat. Show a one-time consent dialog the first time AI is enabled per account.
- Never send: phone numbers in bulk, media files, view-once content, or messages from other accounts. Only text + captions, truncated to `MaxContextMessages`.
- Ghost mode: if ghost is on, AI auto-reply must not send (same rule as `internal/auto`).

### 6. Performance & QA

- Before/after measurements: startup time, idle RAM after 10s idle (use `cmd/memprobe` if present), RAM with 2 accounts, RAM during AI streaming. Put numbers in the PR description.
- `go vet ./...`, `go test ./...`, `go build ./...` must pass on Windows + Linux (at least `GOOS=windows go build`, `GOOS=linux go build` if cross-compile works).
- Manual QA: `go run ./cmd/wazzap -demo` for UI, real link with a secondary number (never the user's main number) for multi-account + AI. Document the QA steps in `docs/qa-ai.md` and `docs/qa-multiaccount.md`.
- Keep the demo mode working: AI features must be disabled / mocked in `-demo` (no real key, no network).

### 7. Deliverables

1. `docs/audit.md` (Phase 0)
2. Phase 1 UI/UX, Phase 2 multi-account, Phase 3 AI — each as separate commits / PRs, each with tests.
3. `docs/ai-setup.md`: how to set up OpenAI / Groq / OpenRouter / Ollama / LM Studio / Custom endpoint, with example BaseURL + Model for each, and troubleshooting (401 / 404 / 429 / CORS is N/A — desktop app, no browser).
4. `docs/qa-ui.md`, `docs/qa-multiaccount.md`, `docs/qa-ai.md`.
5. Updated `README.md`: new features, screenshots in demo mode, AI setup link, multi-account note, and the existing disclaimer that this is unofficial and may violate WhatsApp ToS / risk suspension.

### 8. Out of scope — do not do

- Do not add WebView / Electron / Chromium, do not add a web frontend.
- Do not add calls / video calls unless already in upstream and trivial.
- Do not add a server, account system, or cloud sync. Everything stays local.
- Do not change the WhatsApp protocol code in `hypermeow` — if a protocol bug blocks you, report it, do not fork hypermeow inside this repo.
- Do not enable AI auto-reply by default. Do not send any message automatically without the explicit per-chat opt-in in 5.3(4).

Work now. Start with Phase 0, show the audit, wait for approval before coding Phase 1.

---

## Checklist buat lo sebelum paste

- [ ] Fork `Chomosuke9/WazzapClients` di GitHub
- [ ] `git clone <fork-lo>`, `cd <repo>`, `sh patches/apply.sh`
- [ ] `go run ./cmd/wazzap -demo` jalan dulu (pastikan base-nya sehat)
- [ ] Paste Master Prompt di atas ke AI agent lo
- [ ] Minta agent berhenti setelah Phase 0 audit, review dulu sebelum lanjut
