package command

import (
	"context"
	"strings"
	"testing"

	"github.com/chomosuke9/wazzapclients/internal/mock"
	"github.com/chomosuke9/wazzapclients/internal/model"
)

type fakeAI struct {
	answer    string
	question  string
	chat      *model.Chat
	recent    []string
	summaries int
}

func (f *fakeAI) Answer(ctx context.Context, question string, chat *model.Chat, recent []string) (string, error) {
	f.question, f.chat, f.recent = question, chat, recent
	return f.answer, nil
}

func (f *fakeAI) Summarize(ctx context.Context, chat *model.Chat, recent []string) (string, error) {
	f.summaries++
	f.chat, f.recent = chat, recent
	return f.answer, nil
}

// aiTestHost is the whisper test host, plus Sent.
type aiTestHost struct {
	*whisperTestHost
	sent *model.Message
}

func (h *aiTestHost) Sent(m *model.Message) { h.sent = m }

func aiContext(name, input string, values [][]Value, b model.Backend, h Host, ai AIAnswerer) *Context {
	return &Context{Cmd: Lookup(name), Input: input, Values: values,
		Chat: &model.Chat{ID: "rina", Name: "Rina"}, Backend: b, Host: h, AI: ai}
}

func TestAICommand(t *testing.T) {
	if Lookup("ai") == nil || Lookup("aisummary") == nil {
		t.Fatal("/ai or /aisummary not registered")
	}
	b, h := mock.New(), &aiTestHost{whisperTestHost: &whisperTestHost{}}
	f := &fakeAI{answer: "five o'clock"}
	Execute(aiContext("ai", "/ai what time?", [][]Value{{{Text: "what time?"}}}, b, h, f))
	if h.note == nil || h.note.Text != "🤖 five o'clock" {
		t.Fatalf("note %+v", h.note)
	}
	if f.question != "what time?" || f.chat.ID != "rina" {
		t.Fatalf("ai got %+v", f)
	}
	if len(h.note.Buttons) != 2 || h.note.Buttons[0].Label != "Send" {
		t.Fatalf("buttons %+v", h.note.Buttons)
	}
	// Send really sends the answer to the chat.
	h.note.Buttons[0].Run()
	if h.sent == nil || h.sent.Text != "five o'clock" {
		t.Fatalf("sent %+v", h.sent)
	}
	sent := b.Messages("rina", 1)
	if len(sent) == 0 || sent[0].Text != "five o'clock" || !sent[0].FromMe {
		t.Fatalf("sent %+v", sent)
	}
}

func TestAICommandSummary(t *testing.T) {
	b, h := mock.New(), &aiTestHost{whisperTestHost: &whisperTestHost{}}
	f := &fakeAI{answer: "you talked about coffee"}
	Execute(aiContext("aisummary", "/aisummary", [][]Value{}, b, h, f))
	if h.note == nil || !strings.Contains(h.note.Text, "coffee") || f.summaries != 1 {
		t.Fatalf("summary note %+v", h.note)
	}
}

func TestAICommandNoAI(t *testing.T) {
	b, h := mock.New(), &aiTestHost{whisperTestHost: &whisperTestHost{}}
	Execute(aiContext("ai", "/ai hi", [][]Value{{{Text: "hi"}}}, b, h, nil))
	if h.note == nil || !h.note.Failed || h.note.Text != errNoAI.Error() {
		t.Fatalf("note %+v", h.note)
	}
}

// stubBackend returns canned messages for recentTexts.
type stubBackend struct {
	model.Backend
	msgs []*model.Message
}

func (b *stubBackend) Messages(chatID string, limit int) []*model.Message { return b.msgs }

func TestRecentTextsSkipsViewOnce(t *testing.T) {
	b := &stubBackend{msgs: []*model.Message{
		{ID: "1", ChatID: "rina", Text: "hello", Sender: "Rina"},
		{ID: "2", ChatID: "rina", Kind: model.KindViewOnce, Text: "secret photo"},
		{ID: "3", ChatID: "rina", Text: "  "},
		{ID: "4", ChatID: "rina", Text: "bye", FromMe: true},
	}}
	recent := recentTexts(b, "rina", 20)
	if len(recent) != 2 || recent[0] != "Rina: hello" || recent[1] != "bye" {
		t.Fatalf("recent %q", recent)
	}
}
