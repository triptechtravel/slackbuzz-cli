package text

import "testing"

// Captured output (go test's stdout is a pipe, as an agent's or a script's
// is) gets plain text even in a terminal that renders OSC 8 links.
func TestHyperlink_PlainWhenStdoutIsNotATerminal(t *testing.T) {
	t.Setenv("TERM_PROGRAM", "iTerm.app")
	if got := Hyperlink("slack://channel?id=C1", "1795154860.267309"); got != "1795154860.267309" {
		t.Errorf("Hyperlink = %q, want the plain text", got)
	}
}
