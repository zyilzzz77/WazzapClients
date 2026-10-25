package ui

import (
	"context"
	"strconv"
	"time"

	"gioui.org/widget"

	"github.com/chomosuke9/wazzapclients/internal/ai"
)

// aiUI is the UI's AI state: the account's config, the agent built from
// it, and the settings page's editors. The key lives in the config file
// only (ai.json, 0600); it is never written to the chat database.
type aiUI struct {
	cfg   ai.AIConfig
	agent *ai.Agent
	dir   string // the account data dir the config came from

	urlEd, keyEd, modelEd, tempEd, promptEd widget.Editor
	showKey                                 bool
	showKeyBtn                              widget.Clickable
	testResult                              string
	testing, testedOK                       bool

	suggest aiSuggestState
}

// prefAIConsent is the pref that remembers the one-time AI consent.
const prefAIConsent = "ai_consent"

// loadAI reads the account's AI config and builds the agent. In demo
// mode there is no data dir and no config, so AI stays off: no key, no
// network.
func (u *UI) loadAI() {
	b := u.backend
	if b == nil {
		return
	}
	u.ai.dir = b.DataDir()
	cfg, err := ai.Load(u.ai.dir)
	if err != nil {
		cfg = ai.DefaultConfig()
	}
	u.ai.cfg = cfg
	u.ai.testedOK = false
	u.ai.urlEd.SingleLine = true
	u.ai.keyEd.SingleLine = true
	u.ai.modelEd.SingleLine = true
	u.ai.tempEd.SingleLine = true
	u.ai.keyEd.Mask = '•' // masked until Show
	u.ai.urlEd.SetText(cfg.BaseURL)
	u.ai.keyEd.SetText(cfg.APIKey)
	u.ai.modelEd.SetText(cfg.Model)
	u.ai.tempEd.SetText(strconv.FormatFloat(cfg.Temperature, 'f', -1, 64))
	u.ai.promptEd.SetText(cfg.SystemPrompt)
	u.rebuildAgent()
}

// accountAIEnabled is the per-account AI switch (Settings > Account).
// Without an account list it doesn't block: the config's own switch
// still gates everything.
func (u *UI) accountAIEnabled() bool {
	if u.host == nil || u.host.o.Accounts == nil || u.backend == nil {
		return true
	}
	dir := u.backend.DataDir()
	for _, a := range u.host.o.Accounts.Accounts {
		if u.host.o.Accounts.Path(a.Dir) == dir {
			return a.AIEnabled
		}
	}
	return true
}

// aiAccountName is the open account's name, for the AI's prompt.
func (u *UI) aiAccountName() string {
	if u.host == nil || u.host.o.Accounts == nil || u.backend == nil {
		return ""
	}
	dir := u.backend.DataDir()
	for _, a := range u.host.o.Accounts.Accounts {
		if u.host.o.Accounts.Path(a.Dir) == dir {
			return a.Label()
		}
	}
	return ""
}

// rebuildAgent builds the agent from the config, or drops it when AI
// is off at any level, and (un)hooks auto-reply.
func (u *UI) rebuildAgent() {
	a := &u.ai
	if a.cfg.Enabled && u.accountAIEnabled() {
		a.agent = ai.NewAgent(&ai.Client{
			BaseURL: a.cfg.BaseURL, APIKey: a.cfg.APIKey, Model: a.cfg.Model,
		}, a.cfg)
	} else {
		a.agent = nil
	}
	if u.auto == nil {
		return
	}
	if a.agent != nil && a.cfg.Mode == ai.ModeAuto {
		rules := ai.NewAutoReply(u.now)
		rules.Enabled, rules.AccountEnabled = true, true
		u.auto.SetAIHook(a.agent, rules, u.aiAccountName())
		return
	}
	u.auto.SetAIHook(nil, nil, "")
}

// aiReady reports whether the AI can answer right now.
func (u *UI) aiReady() bool { return u.ai.agent != nil && u.ai.agent.Enabled() }

// aiSave validates the config and writes it to the account's file.
func (u *UI) aiSave() {
	a := &u.ai
	if err := a.cfg.Validate(); err != nil {
		a.testResult = err.Error()
		return
	}
	if err := a.cfg.Save(a.dir); err != nil {
		a.testResult = "Couldn't save the AI settings: " + err.Error()
		return
	}
	a.testResult = ""
}

// aiToggle turns AI on or off. The first turn-on asks once whether the
// user accepts that chat text goes to their AI endpoint.
func (u *UI) aiToggle(on bool) {
	a := &u.ai
	if on && u.backend.Pref(prefAIConsent) != "on" {
		u.confirm("Turn on AI?",
			"AI features send the text of the chats you ask about to the AI endpoint in Settings > AI, "+
				"using your own API key. Nothing is sent unless you ask, or you turn on auto-reply for a chat.\n\n"+
				"Media, view-once messages and other accounts' chats are never sent.",
			dialogButton{label: "Turn on AI", primary: true, run: func() {
				u.backend.SetPref(prefAIConsent, "on")
				a.cfg.Enabled = true
				u.aiSave()
				u.rebuildAgent()
				u.settings.stale = true
			}})
		return
	}
	a.cfg.Enabled = on
	u.aiSave()
	u.rebuildAgent()
	u.settings.stale = true
}

// aiSetMode picks what the AI may do on its own. Auto-reply needs a
// connection test to have succeeded for the current config.
func (u *UI) aiSetMode(mode string) {
	a := &u.ai
	if mode == ai.ModeAuto && !a.testedOK {
		a.testResult = "Test the connection first, then turn on auto-reply."
		return
	}
	a.cfg.Mode = mode
	u.aiSave()
	u.rebuildAgent()
	u.settings.stale = true
}

// aiTestConnection checks the endpoint, key and model, off the UI
// goroutine, and reports the result under the button.
func (u *UI) aiTestConnection() {
	a := &u.ai
	if a.testing {
		return
	}
	cfg := a.cfg // a copy for the goroutine
	a.testing, a.testResult = true, "Testing…"
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		err := (&ai.Client{BaseURL: cfg.BaseURL, APIKey: cfg.APIKey, Model: cfg.Model}).TestConnection(ctx)
		a.testing = false
		if err != nil {
			a.testedOK = false
			a.testResult = ai.UserError(err)
		} else {
			a.testedOK = true
			a.testResult = "Connected. The endpoint, key and model work."
		}
		if u.window != nil {
			u.window.Invalidate()
		}
	}()
}
