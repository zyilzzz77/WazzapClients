# QA — AI (Phase 3)

Manual checks for the AI assistant. Use a **secondary number only**,
never your main number, for anything that sends to WhatsApp. Most of
this needs no WhatsApp at all: the unit tests cover the client, the
agent and the auto-reply rules against a fake server.

## Automated

```sh
go vet ./internal/ai/... ./internal/ui/... ./internal/auto/... ./internal/command/...
go test ./internal/ai/... ./internal/auto/... ./internal/command/... ./internal/ui/...
GOOS=windows go build ./...
```

`internal/ai` tests use an `httptest` fake OpenAI-compatible server (no
network): SSE streaming incl. `[DONE]`, 401/404/429 (with Retry-After)
and unreachable-endpoint error mapping, presets, validation, masking,
config save/load (0600), context truncation, and every auto-reply rule.
`internal/auto` tests attach a mock agent and check a reply is sent,
ghost mode sends nothing, and detaching stops it.

## Settings > AI

1. `go run ./cmd/wazzap -demo` — open Settings > AI. The page shows
   the switch off and a note that nothing is sent. AI features are
   disabled in demo mode (no config, no key, no network).
2. With a real account, pick the Ollama preset: base URL fills in as
   `http://localhost:11434/v1`, model `llama3.1`, no key asked for.
   With Ollama running, Test connection succeeds and reports so.
3. Pick OpenAI with a wrong key: Test connection reports the key was
   refused. With a wrong model name: reports the model wasn't found.
4. Turn AI on the first time: a consent dialog appears once, explaining
   what is sent. Accept, then check `<account data dir>/ai.json` exists
   with mode 0600 and holds the key (don't share the file).
5. Switch provider to Groq and back: the base URL and model follow the
   preset. Edit the base URL by hand: it saves live and survives a
   restart.
6. API key field: masked by default (bullets), Show reveals it, Hide
   masks it again.
7. Auto-reply mode before any successful test: picking it shows "Test
   the connection first". After a successful test, it sticks.
8. Turn the account's AI switch off in Settings > Account: the agent
   stops (no ✨ button, /ai reports AI isn't set up). Turn it back on.

## /ai and /aisummary

1. In a chat with some history, type `/ai what did we talk about?`
   and press Enter. A local note appears with a 🤖 answer and **Send**
   and **Dismiss** buttons. Nothing is sent to WhatsApp yet — check on
   the other side (or in the chat) that no message arrived.
2. Tap **Send**: the answer goes as your message. Tap **Dismiss** on a
   second answer: the note goes away, nothing sent.
3. `/aisummary`: a short summary appears as a local note, same buttons.
4. With AI off, `/ai hi` reports "AI isn't set up yet. Turn it on in
   Settings > AI."
5. Ask about a chat that has a view-once message: the answer must not
   quote its content (view-once is never sent to the AI).

## Suggest replies

1. With AI on, open a chat and tap ✨ beside the emoji button. "✨
   Thinking…" shows, then three short draft chips appear above the
   composer.
2. Tap a chip: its text lands in the composer (appended after what is
   there). Nothing is sent until you press send.
3. Tap ✕: the chips go away. With AI off, the ✨ button isn't there.

## Auto-reply (secondary number only)

1. On the secondary account, set up a local provider (Ollama) so no
   key or internet is involved. Turn on AI, test the connection, pick
   Auto-reply mode.
2. From another number, send the secondary account a 1:1 message. An
   AI reply arrives within a couple of seconds.
3. Send two messages within a minute: only the first gets a reply
   (60s cooldown). Send 11 messages spaced a minute apart: the 11th
   gets none (10/hour cap).
4. In a group, send a message without @mentioning the account: no
   reply. @mention it: a reply comes.
5. Send "can I talk to a human please": no reply, and no reply to the
   next message either (handoff). Send a message yourself from the
   secondary account in that chat: auto-reply stops there too.
6. Turn ghost mode on (`/ghost`): no auto-reply is sent at all.
7. Check `<account data dir>/ai-replies.log`: one line per reply with
   a timestamp, a 16-hex-char prompt hash and the model — no prompt
   text, no message text.
8. Close the window and repeat step 2: auto-reply still works (it runs
   in `internal/auto`, not the UI).
