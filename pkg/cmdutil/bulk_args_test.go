package cmdutil

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpandArgs_PassThroughSingleIDs(t *testing.T) {
	got := ExpandArgs([]string{"1706000000.000001", "1706000000.000002"})
	assert.Equal(t, []string{"1706000000.000001", "1706000000.000002"}, got)
}

func TestExpandArgs_SplitsWhitespaceWithinArgs(t *testing.T) {
	got := ExpandArgs([]string{"1706000000.000001 1706000000.000002"})
	assert.Equal(t, []string{"1706000000.000001", "1706000000.000002"}, got)
}

func TestExpandArgs_DropsEmpty(t *testing.T) {
	got := ExpandArgs([]string{"   ", "ts1", "", "ts2"})
	assert.Equal(t, []string{"ts1", "ts2"}, got)
}

func TestExpandArgs_HandlesMixedNewlinesAndSpaces(t *testing.T) {
	got := ExpandArgs([]string{"ts1\nts2  ts3\tts4"})
	assert.Equal(t, []string{"ts1", "ts2", "ts3", "ts4"}, got)
}

func TestReadLines_TrimsAndSkipsBlanks(t *testing.T) {
	in := strings.NewReader("ts1\n\n  ts2  \n\nts3\n")
	got, err := ReadLines(in)
	require.NoError(t, err)
	assert.Equal(t, []string{"ts1", "ts2", "ts3"}, got)
}

func TestReadLines_SkipsCommentLines(t *testing.T) {
	in := strings.NewReader("# notes go here\nts1\n# more notes\nts2\n")
	got, err := ReadLines(in)
	require.NoError(t, err)
	assert.Equal(t, []string{"ts1", "ts2"}, got)
}

func TestReadLines_NilReader(t *testing.T) {
	got, err := ReadLines(nil)
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestValidateSlackTS_AcceptsWellFormed(t *testing.T) {
	require.NoError(t, ValidateSlackTS([]string{
		"1706000000.000001",
		"1234567890.123456",
	}))
}

func TestValidateSlackTS_RejectsEmpty(t *testing.T) {
	err := ValidateSlackTS([]string{"1706000000.000001", ""})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty")
}

func TestValidateSlackTS_RejectsURLUnsafe(t *testing.T) {
	cases := []string{"1706000000 .000001", "ts/1", "ts?bad", "ts#frag"}
	for _, ts := range cases {
		t.Run(ts, func(t *testing.T) {
			err := ValidateSlackTS([]string{ts})
			require.Error(t, err)
		})
	}
}
