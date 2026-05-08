package cmdutil

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// ExpandArgs normalises positional bulk arguments. Shells differ on
// whether unquoted variables word-split (bash splits on IFS by default;
// zsh does not), so a user running e.g. `slackbuzz react #ch $TIMESTAMPS`
// may end up passing "ts1 ts2 ts3" as a single argument with embedded
// whitespace. Splitting each arg on any ASCII whitespace handles that
// case, plus the common pattern of feeding a stdout/file with
// newline-separated IDs. Empty fields are dropped.
//
// Mirrors clickup-cli's `cmdutil.ExpandIDArgs` — same idiom, different
// domain (Slack timestamps + channel IDs instead of task IDs).
func ExpandArgs(args []string) []string {
	out := make([]string, 0, len(args))
	for _, a := range args {
		for _, part := range strings.Fields(a) {
			out = append(out, part)
		}
	}
	return out
}

// ReadLines reads newline-separated, whitespace-trimmed entries from r.
// Empty lines and lines starting with `#` are dropped, mirroring how
// other CLIs treat config-style input. Used by `--from-stdin` flags.
func ReadLines(r io.Reader) ([]string, error) {
	if r == nil {
		return nil, nil
	}
	var out []string
	s := bufio.NewScanner(r)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		out = append(out, line)
	}
	if err := s.Err(); err != nil {
		return nil, fmt.Errorf("reading bulk args: %w", err)
	}
	return out, nil
}

// ValidateSlackTS rejects timestamps that contain characters which would
// be silently URL-encoded or otherwise mangled. A clear "invalid timestamp"
// error is more useful than Slack's generic message_not_found reaction
// to a malformed `ts` value.
//
// Slack timestamps look like "1706000000.123456" — digits + a single dot.
// We're forgiving on the exact shape (Slack itself isn't fully strict),
// just rejecting URL-special characters.
func ValidateSlackTS(timestamps []string) error {
	for _, ts := range timestamps {
		if ts == "" {
			return fmt.Errorf("empty timestamp")
		}
		if strings.ContainsAny(ts, " \t\n\r/?#&") {
			return fmt.Errorf("invalid timestamp %q: contains whitespace or URL-special characters", ts)
		}
	}
	return nil
}
