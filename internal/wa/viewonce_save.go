package wa

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/chomosuke9/wazzapclients/internal/model"
)

// Saving view once media, only with the opt-in on (model.PrefViewOnceSave).
// The sender expected the media to disappear, so this never runs on its
// own: the viewer asks first, and the setting is off by default.

// SaveViewOnce implements model.Backend: it saves m's media to the
// account's media folder, in the background; a NoticeEvent reports the
// result. With the opt-in off it does nothing but say so.
func (b *Backend) SaveViewOnce(m *model.Message) {
	if b.Pref(model.PrefViewOnceSave) != "on" {
		b.emit(model.NoticeEvent{Text: "Saving view once media is off. Turn it on in Settings > Privacy."})
		return
	}
	go func() {
		path, err := b.saveViewOnce(m)
		if err != nil {
			b.emit(model.NoticeEvent{Text: "Couldn't save the media: " + err.Error()})
			return
		}
		b.emit(model.NoticeEvent{Text: "Saved to " + path})
	}()
}

// saveViewOnce writes m's decrypted media into media/viewonce/ of the
// account's data dir and returns its path.
func (b *Backend) saveViewOnce(m *model.Message) (string, error) {
	data, media, blob, err := b.viewOnceData(m)
	if err != nil {
		return "", err
	}
	dir := filepath.Join(b.dataDir, "media", "viewonce")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	path, err := uniquePath(dir, fileName(m, media, blob))
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return "", err
	}
	return path, nil
}

// viewOnceData is m's media bytes: the local copy when there is one, else
// a download with the stored media key. An OnPhone message has neither.
func (b *Backend) viewOnceData(m *model.Message) ([]byte, model.Media, []byte, error) {
	if data, err := os.ReadFile(b.mediaPath(m.ChatID, m.ID)); err == nil {
		media, blob, _ := b.store.mediaBlob(b.ctx, m.ChatID, m.ID)
		return data, media, blob, nil
	}
	if data, err := os.ReadFile(b.mediaFilePath(m)); err == nil {
		media, blob, _ := b.store.mediaBlob(b.ctx, m.ChatID, m.ID)
		return data, media, blob, nil
	}
	media, blob, _ := b.store.mediaBlob(b.ctx, m.ChatID, m.ID)
	dl := mediaMessage(media, blob)
	cli := b.client()
	if dl == nil || cli == nil || !cli.IsConnected() {
		return nil, media, blob, fmt.Errorf("this media isn't available")
	}
	ctx, cancel := context.WithTimeout(b.ctx, 5*time.Minute)
	defer cancel()
	data, err := cli.DownloadAny(ctx, dl)
	if err != nil {
		return nil, media, blob, fmt.Errorf("couldn't download the media")
	}
	return data, media, blob, nil
}

// uniquePath is dir/name, numbered ("name (1).jpg") when it exists.
func uniquePath(dir, name string) (string, error) {
	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)
	for i := 0; ; i++ {
		try := name
		if i > 0 {
			try = base + " (" + strconv.Itoa(i) + ")" + ext
		}
		path := filepath.Join(dir, try)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return path, nil
		} else if err != nil {
			return "", err
		}
	}
}
