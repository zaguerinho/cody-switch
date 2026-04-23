// Package notify sends macOS desktop notifications.
// It tries terminal-notifier first, then falls back to osascript.
// Errors and non-macOS platforms are silently ignored.
package notify

import (
	"os/exec"
	"runtime"
)

// Send delivers a desktop notification on macOS.
// title is the notification title, message is the body text, and openPath
// (if non-empty) is opened when the notification is clicked.
// On non-macOS platforms or if both notification methods fail, the call is a no-op.
func Send(title, message, openPath string) {
	if runtime.GOOS != "darwin" {
		return
	}

	// Try terminal-notifier first (richer UX, click-to-open support).
	if path, err := exec.LookPath("terminal-notifier"); err == nil {
		args := []string{"-title", title, "-message", message, "-sound", "default"}
		if openPath != "" {
			args = append(args, "-open", "file://"+openPath)
		}
		_ = exec.Command(path, args...).Run()
		return
	}

	// Fall back to osascript.
	script := `display notification "` + escapeAppleScript(message) + `" with title "` + escapeAppleScript(title) + `"`
	_ = exec.Command("osascript", "-e", script).Run()
}

// escapeAppleScript escapes double quotes and backslashes for AppleScript string literals.
func escapeAppleScript(s string) string {
	var out []byte
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '"':
			out = append(out, '\\', '"')
		case '\\':
			out = append(out, '\\', '\\')
		default:
			out = append(out, s[i])
		}
	}
	return string(out)
}
