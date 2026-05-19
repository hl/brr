package workflow

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/hl/brr/internal/engine"
	"github.com/hl/brr/internal/fsutil"
	"github.com/hl/brr/internal/ui"
)

func detectSignalFiles() *engine.Result {
	if fsutil.IsRegularFile(engine.SignalComplete) {
		fmt.Fprintf(os.Stderr, "\n  %s%s✓ All tasks complete%s (%s found). Stopping.\n", ui.Bold, ui.Green, ui.Reset, engine.SignalComplete)
		return &engine.Result{Reason: engine.ReasonComplete}
	}
	if content, ok := readSignalContent(engine.SignalFailed); ok {
		fmt.Fprintf(os.Stderr, "\n  %s%s✗ Agent failed%s (%s found):\n", ui.Bold, ui.Red, ui.Reset, engine.SignalFailed)
		printSignalContent(content)
		return &engine.Result{Reason: engine.ReasonFailed, FailedContent: content}
	}
	if content, ok := readSignalContent(engine.SignalNeedsApproval); ok {
		fmt.Fprintf(os.Stderr, "\n  %s%s⏸ Task needs human approval%s (%s found):\n", ui.Bold, ui.Yellow, ui.Reset, engine.SignalNeedsApproval)
		printSignalContent(content)
		return &engine.Result{Reason: engine.ReasonApproval, ApprovalContent: content}
	}
	if fsutil.IsRegularFile(engine.SignalCycle) {
		fmt.Fprintf(os.Stderr, "\n  %s%s↻ Cycle requested%s (%s found). Stopping this stage.\n", ui.Bold, ui.Magenta, ui.Reset, engine.SignalCycle)
		return &engine.Result{Reason: engine.ReasonCycle}
	}
	return nil
}

func readSignalContent(path string) (string, bool) {
	f, err := fsutil.OpenRegularFile(path)
	if err != nil {
		return "", fsutil.IsRegularFile(path)
	}
	defer func() { _ = f.Close() }()
	data, err := io.ReadAll(io.LimitReader(f, 4097))
	if err != nil {
		return "", true
	}
	if len(data) > 4096 {
		data = data[:4096]
	}
	return strings.TrimSpace(string(data)), true
}

func printSignalContent(content string) {
	if content != "" {
		fmt.Fprintln(os.Stderr, content)
		return
	}
	fmt.Fprintln(os.Stderr, "  (no details provided)")
}

func cleanupSignalFiles() {
	for _, path := range []string{engine.SignalComplete, engine.SignalFailed, engine.SignalNeedsApproval, engine.SignalCycle} {
		if fsutil.IsRegularFile(path) {
			_ = os.Remove(path)
		}
	}
}
