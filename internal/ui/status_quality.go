package ui

import (
	"os"
	"path/filepath"

	"github.com/chomosuke9/wazzapclients/internal/model"
)

// HD status: the photo quality menu (attach.go) is the one chats use,
// and its choice flows into PostStatus through queueSend and
// flushOutbox — Standard fits 1600 px, HD 4096 px. Videos have no
// quality setting: WhatsApp sends them as they are, already HD, so the
// menu says so with their size instead.

// videoSizeNote is the quality menu's note for the send view's videos:
// how many, their total size, sent as they are. "" when there are none.
func (a *attachState) videoSizeNote() string {
	var total int64
	n := 0
	for _, f := range a.files {
		if f.Media != model.MediaVideo {
			continue
		}
		if st, err := os.Stat(f.Path); err == nil {
			n, total = n+1, total+st.Size()
		}
	}
	switch n {
	case 0:
		return ""
	case 1:
		return "1 video (" + formatSize(total) + ") is sent as it is."
	}
	return itoa(n) + " videos (" + formatSize(total) + ") are sent as they are."
}

// fileSizeLabel is a file's name with its size, for the send view's top
// bar: "video.mp4 · 12.4 MB".
func fileSizeLabel(path string) string {
	name := filepath.Base(path)
	if st, err := os.Stat(path); err == nil {
		return name + " · " + formatSize(st.Size())
	}
	return name
}
