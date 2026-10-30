// Package clipboard reads text from the system clipboard by shelling out to
// the platform's paste tool. It exists so token prompts can work where no
// TTY is available (e.g. a `!` command inside an AI agent's terminal).
package clipboard

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/triptechtravel/slackbuzz-cli/internal/browser"
)

// psGetClipboard retries because Windows fails clipboard reads outright
// ("Requested Clipboard operation did not succeed") while another app
// briefly holds the clipboard open.
const psGetClipboard = `$i=0; while ($true) { try { Get-Clipboard -Raw -ErrorAction Stop; break } catch { if (++$i -ge 5) { throw }; Start-Sleep -Milliseconds 200 } }`

// wslPowerShell is used when Windows paths aren't on PATH inside WSL
// (appendWindowsPath=false).
const wslPowerShell = "/mnt/c/Windows/System32/WindowsPowerShell/v1.0/powershell.exe"

// candidates returns paste commands to try, in order, for this platform.
func candidates() [][]string {
	ps := []string{"powershell.exe", "-NoProfile", "-Command", psGetClipboard}
	var linux [][]string
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		linux = append(linux, []string{"wl-paste", "--no-newline"})
	}
	linux = append(linux,
		[]string{"xclip", "-selection", "clipboard", "-o"},
		[]string{"xsel", "--clipboard", "--output"},
	)

	switch {
	case runtime.GOOS == "darwin":
		return [][]string{{"pbpaste"}}
	case runtime.GOOS == "windows":
		return [][]string{ps}
	case browser.IsWSL():
		// Windows clipboard first, then WSLg's Linux clipboard tools.
		abs := append([]string{wslPowerShell}, ps[1:]...)
		return append([][]string{ps, abs}, linux...)
	default:
		return linux
	}
}

// Read returns the clipboard's text with surrounding whitespace (including
// Windows CRLF) trimmed. It tries each available tool until one succeeds.
func Read() (string, error) {
	var failures []string
	for _, c := range candidates() {
		if _, err := exec.LookPath(c[0]); err != nil {
			continue
		}
		out, err := exec.Command(c[0], c[1:]...).Output()
		if err != nil {
			failures = append(failures, describe(c[0], err))
			continue
		}
		if text := strings.TrimSpace(string(out)); text != "" {
			return text, nil
		}
		failures = append(failures, c[0]+": clipboard is empty")
	}
	if len(failures) == 0 {
		return "", fmt.Errorf("no clipboard tool found — pipe the token with --with-token instead")
	}
	return "", fmt.Errorf("%s — pipe the token with --with-token instead", strings.Join(failures, "; "))
}

// describe summarises a failed paste command, keeping the tool's own error
// text (but only its first line — PowerShell errors run long).
func describe(tool string, err error) string {
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		if msg := strings.TrimSpace(strings.SplitN(string(ee.Stderr), "\n", 2)[0]); msg != "" {
			return tool + ": " + msg
		}
	}
	return fmt.Sprintf("%s: %v", tool, err)
}
