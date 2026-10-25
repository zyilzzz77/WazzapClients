package auto

import (
	"testing"
	"time"

	"github.com/chomosuke9/wazzapclients/internal/ai"
	"github.com/chomosuke9/wazzapclients/internal/model"
)

// pollForReply polls until the backend sends a message, or gives up.
func (at *autoTest) pollForReply(t *testing.T) *model.Message {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if ms, _ := at.poll(); len(ms) > 0 {
			return ms[0]
		}
		time.Sleep(10 * time.Millisecond)
	}
	return nil
}

func newAITest(t *testing.T) (*autoTest, *ai.MockProvider) {
	t.Helper()
	at := newAutoTest(t)
	mock := &ai.MockProvider{Text: "hello back"}
	cfg := ai.DefaultConfig()
	cfg.Enabled = true
	rules := ai.NewAutoReply(func() time.Time { return at.now })
	rules.Enabled, rules.AccountEnabled = true, true
	at.a.SetAIHook(ai.NewAgent(mock, cfg), rules, "Me")
	return at, mock
}

func TestAIAutoReply(t *testing.T) {
	at, mock := newAITest(t)
	at.f.in = append(at.f.in, model.MessageEvent{
		Msg: &model.Message{ID: "m1", ChatID: "rina", Text: "hi there", Time: at.now}, New: true})
	m := at.pollForReply(t)
	if m == nil || m.Text != "hello back" || m.ChatID != "rina" {
		t.Fatalf("reply %+v", m)
	}
	if mock.Calls != 1 {
		t.Fatalf("agent called %d times", mock.Calls)
	}
}

func TestAIAutoReplyGhost(t *testing.T) {
	at, _ := newAITest(t)
	at.a.SetPref(model.PrefGhost, "on") // ghost mode refuses sends
	at.f.in = append(at.f.in, model.MessageEvent{
		Msg: &model.Message{ID: "m1", ChatID: "rina", Text: "hi there", Time: at.now}, New: true})
	if m := at.pollForReply(t); m != nil {
		t.Fatalf("ghost mode sent %+v", m)
	}
}

func TestAIAutoReplyDetached(t *testing.T) {
	at, mock := newAITest(t)
	at.a.SetAIHook(nil, nil, "")
	at.f.in = append(at.f.in, model.MessageEvent{
		Msg: &model.Message{ID: "m1", ChatID: "rina", Text: "hi there", Time: at.now}, New: true})
	at.poll() // the message passes through
	if mock.Calls != 0 {
		t.Fatal("detached hook still called the agent")
	}
}
