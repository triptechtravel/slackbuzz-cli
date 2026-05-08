package recent

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// withTempConfig redirects the recent-store to a tempdir for isolation.
func withTempConfig(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, ".config"))
	mu.Lock()
	loaded = nil
	mu.Unlock()
	t.Cleanup(func() {
		mu.Lock()
		loaded = nil
		mu.Unlock()
	})
}

func TestPushAndLast(t *testing.T) {
	withTempConfig(t)

	require.NoError(t, Push("dm", "@alice"))
	assert.Equal(t, "@alice", Last("dm"))

	require.NoError(t, Push("dm", "@bob"))
	assert.Equal(t, "@bob", Last("dm"), "newest entry wins")
}

func TestPushDeduplicates(t *testing.T) {
	withTempConfig(t)

	require.NoError(t, Push("dm", "@alice"))
	require.NoError(t, Push("dm", "@bob"))
	require.NoError(t, Push("dm", "@alice")) // promote to front

	got := List("dm", 10)
	require.Len(t, got, 2, "duplicate target should be deduped, not appended")
	assert.Equal(t, "@alice", got[0].Target)
	assert.Equal(t, "@bob", got[1].Target)
}

func TestPushCaps(t *testing.T) {
	withTempConfig(t)

	for i := 0; i < maxPerSlot+5; i++ {
		require.NoError(t, Push("dm", string(rune('a'+i))))
	}
	got := List("dm", 100)
	assert.LessOrEqual(t, len(got), maxPerSlot, "slot should be capped at maxPerSlot")
}

func TestSeparateSlots(t *testing.T) {
	withTempConfig(t)

	require.NoError(t, Push("dm", "@alice"))
	require.NoError(t, Push("channel", "#general"))

	assert.Equal(t, "@alice", Last("dm"))
	assert.Equal(t, "#general", Last("channel"))
}

func TestLast_EmptySlot(t *testing.T) {
	withTempConfig(t)
	assert.Equal(t, "", Last("nonexistent"))
}

func TestList_NewestFirst(t *testing.T) {
	withTempConfig(t)

	require.NoError(t, Push("dm", "@alice"))
	time.Sleep(2 * time.Millisecond) // ensure distinct timestamps
	require.NoError(t, Push("dm", "@bob"))
	time.Sleep(2 * time.Millisecond)
	require.NoError(t, Push("dm", "@carol"))

	got := List("dm", 3)
	require.Len(t, got, 3)
	assert.Equal(t, "@carol", got[0].Target)
	assert.Equal(t, "@bob", got[1].Target)
	assert.Equal(t, "@alice", got[2].Target)
}

func TestPersistsAcrossLoads(t *testing.T) {
	withTempConfig(t)

	require.NoError(t, Push("dm", "@alice"))

	// Force re-read from disk by clearing the in-memory cache.
	mu.Lock()
	loaded = nil
	mu.Unlock()

	assert.Equal(t, "@alice", Last("dm"))
}

func TestEmptyTargetIgnored(t *testing.T) {
	withTempConfig(t)

	require.NoError(t, Push("dm", ""))
	assert.Equal(t, "", Last("dm"))
	_, err := os.Stat(file())
	assert.True(t, os.IsNotExist(err), "no file should be created for empty pushes")
}
