package ai

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// PromptHash hashes a prompt for the auto-reply log: the log keeps the
// hash, never the prompt itself.
func PromptHash(prompt string) string {
	sum := sha256.Sum256([]byte(prompt))
	return hex.EncodeToString(sum[:])[:16]
}

// ReplyLogName is the auto-reply log's file in an account's data dir.
const ReplyLogName = "ai-replies.log"

// AppendReplyLog records one auto-reply: when, the prompt's hash and the
// model. It never records the prompt or the reply text.
func AppendReplyLog(dir string, at time.Time, promptHash, model string) error {
	if dir == "" {
		return nil
	}
	f, err := os.OpenFile(filepath.Join(dir, ReplyLogName), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = fmt.Fprintf(f, "%s %s %s\n", at.Format(time.RFC3339), promptHash, model)
	return err
}
