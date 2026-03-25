package engine

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/hl/brr/internal/ui"
)

const maxFailStreak = 3

// Options configures a loop run.
type Options struct {
	Prompt  string   // resolved prompt text
	Max     int      // max iterations (0 = unlimited)
	Command []string // command + args to run (prompt piped to stdin)
}

// Run executes the loop until completion, max iterations, or interrupt.
func Run(opts Options) error {
	if len(opts.Command) == 0 {
		return fmt.Errorf("no command configured — set 'command' in .brr.yaml")
	}

	// Clean up stale signal files from previous runs
	os.Remove(ui.SignalComplete)
	os.Remove(ui.SignalNeedsApproval)

	// Clean up signal files on exit
	defer func() {
		if err := os.Remove(ui.SignalComplete); err != nil && !os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "warning: could not clean up %s: %v\n", ui.SignalComplete, err)
		}
	}()
	defer func() {
		if err := os.Remove(ui.SignalNeedsApproval); err != nil && !os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "warning: could not clean up %s: %v\n", ui.SignalNeedsApproval, err)
		}
	}()

	// Track the currently running subprocess so we can forward signals
	var mu sync.Mutex
	var currentCmd *exec.Cmd

	var interrupted atomic.Bool
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	go func() {
		first := true
		for range sigCh {
			interrupted.Store(true)
			mu.Lock()
			cmd := currentCmd
			mu.Unlock()
			if cmd != nil && cmd.Process != nil {
				if first {
					// Signal the entire process group (child + its descendants)
					_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGINT)
					first = false
				} else {
					// Second signal: force kill the process group
					_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
				}
			}
		}
	}()

	failStreak := 0
	i := 0

	for opts.Max == 0 || i < opts.Max {
		if interrupted.Load() {
			fmt.Printf("\n  %s%sInterrupted%s. Stopping.\n", ui.Bold, ui.Yellow, ui.Reset)
			return nil
		}

		if stop := checkSignalFiles(); stop {
			return nil
		}

		// Print iteration header
		iterNum := i + 1
		maxLabel := ""
		if opts.Max > 0 {
			maxLabel = fmt.Sprintf("/%d", opts.Max)
		}
		fmt.Printf("\n%s━━━%s %s%sIteration %d%s%s %s▸ %s ━━━%s\n",
			ui.Dim, ui.Reset,
			ui.Bold, ui.Cyan, iterNum, maxLabel, ui.Reset,
			ui.Dim, time.Now().Format("15:04:05"), ui.Reset,
		)

		// Run the command with prompt piped to stdin.
		// Use a process group so Ctrl+C kills the entire child tree.
		cmd := exec.Command(opts.Command[0], opts.Command[1:]...)
		cmd.Stdin = strings.NewReader(opts.Prompt)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

		mu.Lock()
		currentCmd = cmd
		mu.Unlock()

		err := cmd.Run()

		mu.Lock()
		currentCmd = nil
		mu.Unlock()

		// Check for interrupt or signal files immediately after subprocess exits
		if interrupted.Load() {
			fmt.Printf("\n  %s%sInterrupted%s. Stopping.\n", ui.Bold, ui.Yellow, ui.Reset)
			return nil
		}
		if stop := checkSignalFiles(); stop {
			return nil
		}

		if err != nil {
			failStreak++
			rc := 1
			if exitErr, ok := err.(*exec.ExitError); ok {
				rc = exitErr.ExitCode()
			}
			fmt.Printf("  %s%s✗ Iteration %d failed%s (exit %d). Consecutive failures: %d/%d\n",
				ui.Bold, ui.Red, iterNum, ui.Reset, rc, failStreak, maxFailStreak,
			)
			if failStreak >= maxFailStreak {
				fmt.Printf("  %s%s✗ Too many consecutive failures. Stopping.%s\n", ui.Bold, ui.Red, ui.Reset)
				return fmt.Errorf("stopped after %d consecutive failures", maxFailStreak)
			}
		} else {
			failStreak = 0
		}

		// i counts total attempts, including failures
		i++
	}

	return nil
}

// checkSignalFiles checks for .brr-complete and .brr-needs-approval.
// Returns true if the engine should stop.
func checkSignalFiles() bool {
	if _, err := os.Stat(ui.SignalComplete); err == nil {
		fmt.Printf("\n  %s%s✓ All tasks complete%s (%s found). Stopping.\n", ui.Bold, ui.Green, ui.Reset, ui.SignalComplete)
		return true
	}
	if data, err := os.ReadFile(ui.SignalNeedsApproval); err == nil {
		fmt.Printf("\n  %s%s⏸ Task needs human approval%s (%s found):\n", ui.Bold, ui.Yellow, ui.Reset, ui.SignalNeedsApproval)
		content := strings.TrimSpace(string(data))
		if content != "" {
			fmt.Println(content)
		} else {
			fmt.Println("  (no details provided)")
		}
		return true
	}
	return false
}
