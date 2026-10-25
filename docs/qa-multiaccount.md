# Manual QA — true multi-account (Phase 2)

Use a **secondary number only, never the main number**. Link two
accounts (A = the first/root account, B = a second account in
`accounts/2`) before starting. Build with `go run ./cmd/wazzap` (or
the installed build); `-demo` shows the switcher with demo accounts
but no real switching.

## 1. Two accounts linked, switch A → B → A

1. Link A, then ⋮ menu → Switch account → Add account, link B.
2. In A, type half a message in a chat (do not send). Switch to B.
3. In B, type half a message in a chat. Switch back to A.
4. Expect: A's half-written message is still in its composer, B's is
   still in B's — neither leaked into the other account, and neither
   was lost. (Automated: `TestDraftsPerAccount`.)
5. Switching to a linked account that was already connected in the
   background should feel instant (no QR, no reconnect); switching to
   an Off account reconnects and shows its stored chats at once.

## 2. Send from each

1. Send a message from A to a contact, then switch to B and send one
   from B.
2. Expect: each message goes from the right number, and each account's
   chat list shows only its own chats.

## 3. Background Always account receives while B is open

1. Set A to "Always connected" (switcher → A's mode button, or
   Settings > Account > A), keep B open.
2. From a third phone, send a message to A.
3. Expect:
   - A's row in the account switcher shows an unread badge with the
     count (automated: `TestMultiAccountUnread`).
   - The tray tooltip totals unread across accounts (hover the tray
     icon: "N unread chats", counting A and B together).
   - A notification appears titled "<chat> · <A's name>" even though
     the window shows B, with Reply / Mark as read.
   - Clicking the notification switches to A and opens the chat.
     (Automated: `TestBackgroundNotifies`.)
4. Set A to "Check every 15 minutes" instead: the message arrives at
   the next check, the switcher says "Checked HH:MM · every 15 min",
   and the notification still names A.

## 4. Memory budget (max 2 Always, default)

1. Link a third account C.
2. Set A and B to "Always connected", then try C → "Always connected".
3. Expect: a toast "Only 2 accounts can stay always connected. Set
   another account to check every few minutes instead." and C stays
   "Only while open". (Automated: `TestAlwaysBudgetRefused`.)
4. The cap is `always_limit` in `accounts.json` (0 = default 2), or
   Settings > Account > an account > "Always connected, at most".

## 5. Per-account settings

1. Settings > Account > B: turn "Notifications" off, send B a message
   from another phone while A is open.
2. Expect: no notification for B; A's notifications still work.
   (Automated: `TestAccountNotifyOff`.)
3. The "AI features" switch on the same page is a Phase 3 placeholder:
   it is stored per account and nothing runs on it yet.

## 6. Scroll position per chat per account

1. In A, open a long chat, scroll up to an older message, switch to B,
   switch back to A, reopen the same chat.
2. Expect: the chat opens at the message it was left at, not at the
   newest. (Automated: `TestScrollPerAccount`.)

## 7. Rename / reorder / remove isolation

Rename and reorder are API-level in this phase (`List.Rename`,
`List.Move` in `internal/accounts/manage.go`, host methods
`renameAccount`/`moveAccount`); there is no rename/reorder UI yet.
Removal is UI-level and tested:

1. Settings > Account > B → Log out (or switcher → B → Log out).
2. Expect: B leaves the list, its `accounts/2` directory is deleted,
   and A's data (`wazzap.db`, chats, prefs) is untouched.
   (Automated: `TestRemoveIsolation`, `TestBackgroundLogout`.)

## 8. Idle RAM with 2 accounts

1. With A open and B "Always connected", wait 10 s idle (memtrim
   runs), then read the private working set in Task Manager.
2. Compare with one account open and B off. Expect: the background
   account costs only a few MB (upstream: ~2 MB idle, +~10 MB for a
   big database) — background accounts keep no chat history or media
   decoded in memory, only the notifier's per-chat counts, and a
   memtrim runs on every switch (`useBackend`).
3. Record the numbers here when measured on Windows:

   | Setup | Idle private working set |
   |---|---|
   | 1 account | _ MB |
   | 2 accounts, B Always | _ MB |
   | 2 accounts, B Off | _ MB |

## Automated coverage

```sh
go vet ./internal/accounts/... ./internal/ui/...
go test ./internal/accounts/... ./internal/ui/ -run 'TestAccount|TestUnread|TestDraft|TestSwitch|TestMulti|TestAlways'
GOOS=windows go build ./...
```

New tests: `TestAccountRenameMove`, `TestAlwaysBudget`,
`TestAccountSettings`, `TestRemoveIsolation`, `TestUnreadCounts`
(`internal/accounts`); `TestDraftsPerAccount`,
`TestScrollPerAccount`, `TestMultiAccountUnread`,
`TestAccountNotifyOff`, `TestAlwaysBudgetRefused`,
`TestSwitchKeepsBackgroundUnread` (`internal/ui`).
