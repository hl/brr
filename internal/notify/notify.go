package notify

import (
	"fmt"
	"os/exec"
	"strings"
	"unicode/utf8"

	"github.com/hl/brr/internal/engine"
)

// Send dispatches a best-effort desktop notification for the given engine result.
// It returns an error if the notification could not be sent, but callers should
// treat failures as non-fatal.
func Send(result *engine.Result) error {
	title, body := format(result)
	return send(title, body)
}

func format(result *engine.Result) (title, body string) {
	switch result.Reason {
	case engine.ReasonComplete:
		return "brr — complete", "All tasks complete."
	case engine.ReasonFailed:
		title = "brr — failed"
		if result.FailedContent != "" {
			body = truncate(result.FailedContent, 256)
		} else {
			body = "The agent reported a failure."
		}
		return title, body
	case engine.ReasonApproval:
		title = "brr — approval needed"
		if result.ApprovalContent != "" {
			body = truncate(result.ApprovalContent, 256)
		} else {
			body = "A task needs human approval."
		}
		return title, body
	case engine.ReasonMaxIterations:
		return "brr — max iterations", "Maximum iteration count reached."
	case engine.ReasonFailStreak:
		return "brr — stopped", "Too many consecutive failures."
	default:
		return "brr — stopped", "The loop has stopped."
	}
}

// truncate shortens s to at most maxLen bytes, breaking at the last space
// boundary to avoid cutting words mid-way. Appends "…" when truncated.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	cut := strings.LastIndex(s[:maxLen], " ")
	if cut <= 0 {
		cut = maxLen
	}
	for cut > 0 && !utf8.ValidString(s[:cut]) {
		cut--
	}
	return s[:cut] + "…"
}

// run executes a command and returns any error, swallowing stderr output.
func run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %w (%s)", name, err, strings.TrimSpace(string(out)))
	}
	return nil
}
