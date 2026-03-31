package notify

import (
	"fmt"
	"strings"
)

// escapeAppleScript escapes a string for safe embedding in an AppleScript
// double-quoted literal. Backslashes and double-quotes are the only characters
// that need escaping in AppleScript string literals.
func escapeAppleScript(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return s
}

func send(title, body string) error {
	script := fmt.Sprintf(`display notification "%s" with title "%s"`,
		escapeAppleScript(body), escapeAppleScript(title))
	return run("osascript", "-e", script)
}
