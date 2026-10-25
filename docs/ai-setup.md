# AI setup

WazzapClients has a provider-agnostic AI assistant. You bring your own
key, endpoint and model; the app talks straight to that endpoint (no
proxy, no server of ours) and only with the text you ask about. There is
no bundled provider and no bundled key.

## Turn it on

1. Open **Settings > AI**.
2. Pick a provider preset, or **Custom** for anything else that speaks
   the OpenAI Chat Completions API.
3. Enter the **Base URL**, **API Key** (masked; Show/Hide beside it) and
   **Model**.
4. Press **Test connection**. It must succeed before auto-reply can be
   turned on.
5. Flip **AI features** on. The first time, the app asks once whether you
   accept that chat text goes to your endpoint.

The config is stored per account, as `<account data dir>/ai.json`
(file mode 0600). The key is never written to the chat database, never
logged, and never leaves the endpoint you configured. Auto-reply also
needs the account's AI switch in **Settings > Account** (per-account,
off by default) — and the per-chat allowlist, which is all chats unless
narrowed in the config.

## Providers

| Provider | Base URL | Example model | Key |
|---|---|---|---|
| OpenAI | `https://api.openai.com/v1` | `gpt-4o-mini` | OpenAI API key |
| Groq | `https://api.groq.com/openai/v1` | `llama-3.3-70b-versatile` | Groq API key |
| OpenRouter | `https://openrouter.ai/api/v1` | `openai/gpt-4o-mini` | OpenRouter API key |
| Google Gemini | `https://generativelanguage.googleapis.com/v1beta/openai` | `gemini-2.0-flash` | Google AI Studio key |
| Ollama (local) | `http://localhost:11434/v1` | `llama3.1` | none needed |
| LM Studio (local) | `http://localhost:1234/v1` | `local-model` | none needed |
| Custom | your endpoint's base URL | your model | as the endpoint asks |

For Ollama, first pull a model (`ollama pull llama3.1`) and make sure it
serves on port 11434. For LM Studio, load a model and start its local
server (port 1234). Local providers need no key and send nothing to the
internet.

Temperature (0–2, default 0.7) steers how free the answers are; the
system prompt (empty for the default) steers tone. Answers follow the
chat's language, defaulting to Indonesian. At most 20 recent messages
go to the AI, text and captions only — never media, never view-once
content, never another account's chats.

## What it does

- **`/ai <question>`** — answers about the open chat's last 20 messages.
  The answer shows as a local AI note, clearly labelled; it is sent to
  WhatsApp only if you tap **Send**.
- **`/aisummary`** — same, but a summary of the recent messages.
- **✨ Suggest replies** — the button beside the composer's emoji button
  drafts three replies (short, polite, casual) from the recent messages.
  Tap one to put it in the composer; you still send it.
- **Auto-reply** (Settings > AI > Auto-reply, opt-in) — replies for you
  in chats. Rules: never your own messages, status or channels; groups
  only when @mentioned; 60s cooldown per chat; at most 10 replies an
  hour per chat; quiet hours; `human` or `admin` in a message hands that
  chat to you; sending a message yourself stops it in that chat; ghost
  mode on means it never sends. Every auto-reply is logged locally in
  `<account data dir>/ai-replies.log` with a timestamp, a hash of the
  prompt (never the prompt) and the model.

## Troubleshooting

- **401 / "API key was refused"** — the key is wrong, revoked, or for
  another provider. Check the key and the preset; a Groq key does not
  work at OpenAI's endpoint, and vice versa.
- **404 / "model was not found"** — the model name is wrong for that
  endpoint, or the model isn't loaded (Ollama/LM Studio). Check the
  model name, or pull/load it first.
- **429 / "rate limiting"** — the provider says slow down. Wait for the
  Retry-After time shown, or use a smaller/cheaper model. Auto-reply
  already caps itself at 10 replies an hour per chat.
- **"endpoint couldn't be reached"** — the base URL is wrong, the local
  server isn't running, or there's no connection. For Ollama/LM Studio,
  check the server is up on its port; for a custom endpoint, check the
  URL ends in `/v1` (or whatever the provider documents).
- **"unexpected response"** — the endpoint answered, but not as a chat
  completions endpoint. Check the base URL: it should be the API root
  (e.g. `https://api.openai.com/v1`), not a chat or docs page.
- **Auto-reply won't turn on** — press **Test connection** first; it
  must succeed for the current config. Also check the account's AI
  switch in Settings > Account.
- **AI is off in demo mode** — `-demo` has no config and no key, so AI
  features are disabled there on purpose. Set them up in a real account.
