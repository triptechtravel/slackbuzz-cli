package browser

import (
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// IsWSL reports whether we're running inside Windows Subsystem for Linux,
// where xdg-open usually has no browser to hand off to.
func IsWSL() bool {
	if runtime.GOOS != "linux" {
		return false
	}
	if os.Getenv("WSL_DISTRO_NAME") != "" {
		return true
	}
	data, err := os.ReadFile("/proc/sys/kernel/osrelease")
	return err == nil && strings.Contains(strings.ToLower(string(data)), "microsoft")
}

// Open opens a URL in the user's default browser.
func Open(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = linuxCommand(url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}

func linuxCommand(url string) *exec.Cmd {
	if IsWSL() {
		// Hand the URL to Windows directly via interop; rundll32 involves
		// no cmd.exe, so `&` and `%` survive. wslview (wslu) as a fallback —
		// older releases mangle `&` in query strings.
		if _, err := exec.LookPath("rundll32.exe"); err == nil {
			return exec.Command("rundll32.exe", "url.dll,FileProtocolHandler", url)
		}
		if _, err := exec.LookPath("wslview"); err == nil {
			return exec.Command("wslview", url)
		}
	}
	return exec.Command("xdg-open", url)
}
