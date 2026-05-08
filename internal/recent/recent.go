// Package recent persists short-lived "last used" context per command,
// mirroring the clickup-cli pattern (`pkg/cmdutil/recent.go`).
//
// Slack-specific shape:
//
//   - `Targets["dm"]`       last DM target (@user or U-id)
//   - `Targets["send"]`     last channel/DM sent to
//   - `Targets["channel"]`  last channel inspected
//
// Used to default no-arg invocations to the obvious last target without
// dragging Slack's whole "recent" surface into the local config. Bounded
// to a few KB of JSON in the user's config dir; auto-pruned to the last
// 16 entries per slot.
package recent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/triptechtravel/slackbuzz-cli/internal/config"
)

// Entry is a single (target, when) tuple. Newest first.
type Entry struct {
	Target string    `json:"target"`
	When   time.Time `json:"when"`
}

// Recent is the on-disk shape — a per-slot list of recent targets.
type Recent struct {
	Targets map[string][]Entry `json:"targets"`
}

const maxPerSlot = 16

var (
	mu     sync.Mutex
	loaded *Recent
)

func file() string {
	return filepath.Join(config.ConfigDir(), "recent.json")
}

// load reads the recent file from disk into the package-level cache.
// Unexported: external callers go through Push/Last/List, which return
// values rather than the mutable cached pointer.
//
// Caller must hold `mu`.
func load() (*Recent, error) {
	if loaded != nil {
		return loaded, nil
	}

	r := &Recent{Targets: map[string][]Entry{}}
	data, err := os.ReadFile(file())
	if err != nil {
		if os.IsNotExist(err) {
			loaded = r
			return r, nil
		}
		return nil, err
	}
	if err := json.Unmarshal(data, r); err != nil {
		return nil, fmt.Errorf("recent: parse %s: %w", file(), err)
	}
	if r.Targets == nil {
		r.Targets = map[string][]Entry{}
	}
	loaded = r
	return r, nil
}

// save writes the recent file. Caller must hold `mu`.
func (r *Recent) save() error {
	dir := config.ConfigDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(file(), data, 0o600)
}

// Push promotes target to the front of slot. Existing duplicates are
// removed. The slot is capped at maxPerSlot.
func Push(slot, target string) error {
	if target == "" {
		return nil
	}
	mu.Lock()
	defer mu.Unlock()

	r, err := load()
	if err != nil {
		return err
	}
	entries := r.Targets[slot]
	out := make([]Entry, 0, len(entries)+1)
	out = append(out, Entry{Target: target, When: time.Now()})
	for _, e := range entries {
		if e.Target == target {
			continue
		}
		out = append(out, e)
		if len(out) >= maxPerSlot {
			break
		}
	}
	r.Targets[slot] = out
	return r.save()
}

// Last returns the most recent target in slot, or "" if none.
func Last(slot string) string {
	mu.Lock()
	defer mu.Unlock()
	r, err := load()
	if err != nil {
		return ""
	}
	if entries, ok := r.Targets[slot]; ok && len(entries) > 0 {
		return entries[0].Target
	}
	return ""
}

// List returns up to n recent targets in slot, newest first. The result
// is a defensive copy — callers can mutate it freely without affecting
// the on-disk cache.
func List(slot string, n int) []Entry {
	mu.Lock()
	defer mu.Unlock()
	r, err := load()
	if err != nil {
		return nil
	}
	entries := r.Targets[slot]
	if n > 0 && len(entries) > n {
		entries = entries[:n]
	}
	out := make([]Entry, len(entries))
	copy(out, entries)
	sort.Slice(out, func(i, j int) bool { return out[i].When.After(out[j].When) })
	return out
}
