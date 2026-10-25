// Per-account unread counts, kept beside accounts.json so the
// switcher's badges and the tray tooltip survive a restart.
//
// Ownership: only the UI goroutine reads and writes these counts
// (host.updateTooltip syncs them from the notifiers). The mutex only
// guards against a stray call from another goroutine, e.g. in tests;
// it is not a licence to share the store across goroutines.
package accounts

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

const unreadFile = "unread.json"

// Counts is the unread-chat count of each account, by Dir.
type Counts struct {
	mu     sync.Mutex
	counts map[string]int
}

// LoadUnread reads the counts of root, or empty ones when there is no
// file yet.
func LoadUnread(root string) *Counts {
	c := &Counts{counts: map[string]int{}}
	if data, err := os.ReadFile(filepath.Join(root, unreadFile)); err == nil {
		_ = json.Unmarshal(data, &c.counts)
	}
	if c.counts == nil {
		c.counts = map[string]int{}
	}
	return c
}

// Save writes the counts next to accounts.json (0600, tmp+rename).
func (c *Counts) Save(root string) error {
	c.mu.Lock()
	data, err := json.MarshalIndent(c.counts, "", "\t")
	c.mu.Unlock()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(root, 0o700); err != nil {
		return err
	}
	tmp := filepath.Join(root, unreadFile+".tmp")
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(root, unreadFile))
}

// Set stores dir's count. It reports whether the count changed.
func (c *Counts) Set(dir string, n int) bool {
	if n < 0 {
		n = 0
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.counts[dir] == n {
		return false
	}
	c.counts[dir] = n
	return true
}

// Get returns dir's count, or 0.
func (c *Counts) Get(dir string) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.counts[dir]
}

// Total adds every account's count.
func (c *Counts) Total() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	total := 0
	for _, n := range c.counts {
		total += n
	}
	return total
}

// Remove forgets dir (it left the list), reporting whether it had one.
func (c *Counts) Remove(dir string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.counts[dir]; !ok {
		return false
	}
	delete(c.counts, dir)
	return true
}

// Prune forgets every dir not in dirs, reporting whether any went.
func (c *Counts) Prune(dirs []string) bool {
	keep := make(map[string]bool, len(dirs))
	for _, d := range dirs {
		keep[d] = true
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	changed := false
	for d := range c.counts {
		if !keep[d] {
			delete(c.counts, d)
			changed = true
		}
	}
	return changed
}

// Snapshot copies the counts, for tests and the tooltip.
func (c *Counts) Snapshot() map[string]int {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make(map[string]int, len(c.counts))
	for d, n := range c.counts {
		out[d] = n
	}
	return out
}
