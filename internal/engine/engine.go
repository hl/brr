package engine

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"github.com/hl/brr/internal/fsutil"
	"github.com/hl/brr/internal/ui"
)

const maxApprovalFileSize = 4096

const maxFailStreak = 3

// ErrInterrupted is returned when the engine is stopped by a user signal (Ctrl+C).
var ErrInterrupted = fmt.Errorf("interrupted")

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

	// If signal files exist from a previous run, respect them immediately
	if stop := checkSignalFiles(); stop {
		// Clean up the signal files so they don't block subsequent runs
		os.Remove(ui.SignalComplete)
		os.Remove(ui.SignalNeedsApproval)
		return nil
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

	// Signal handling: three levels
	// 1st Ctrl+C: finish current iteration, then stop
	// 2nd Ctrl+C: send SIGINT to child (graceful shutdown)
	// 3rd Ctrl+C: force kill child
	var stopping atomic.Bool
	var sigCount atomic.Int32
	sigCh := make(chan os.Signal, 3)
	done := make(chan struct{})
	notifySignals(sigCh)
	defer signal.Stop(sigCh)

	go func() {
		for {
			select {
			case <-done:
				return
			case sig, ok := <-sigCh:
				if !ok {
					return
				}
				// SIGTERM: forward to child immediately for graceful shutdown
				if sig == sigTERM {
					stopping.Store(true)
					mu.Lock()
					cmd := currentCmd
					mu.Unlock()
					if cmd != nil && cmd.Process != nil {
						_ = killGroup(cmd, sigTERM)
					}
					fmt.Printf("\n  %s%s⏳ SIGTERM received, forwarding to child...%s\n",
						ui.Bold, ui.Yellow, ui.Reset)
					continue
				}
				// SIGINT (Ctrl+C): three escalation levels
				n := sigCount.Add(1)
				switch n {
				case 1:
					stopping.Store(true)
					fmt.Printf("\n  %s%s⏳ Finishing current iteration...%s (Ctrl+C again to interrupt now)\n",
						ui.Bold, ui.Yellow, ui.Reset)
				case 2:
					mu.Lock()
					cmd := currentCmd
					mu.Unlock()
					if cmd != nil && cmd.Process != nil {
						_ = killGroup(cmd, sigINT)
					}
				default:
					mu.Lock()
					cmd := currentCmd
					mu.Unlock()
					if cmd != nil && cmd.Process != nil {
						_ = killGroup(cmd, sigKILL)
					}
				}
			}
		}
	}()
	defer close(done)

	failStreak := 0
	i := 0

	for opts.Max == 0 || i < opts.Max {
		// Check if user requested stop (first Ctrl+C) between iterations
		if stopping.Load() {
			fmt.Printf("\n  %s%sStopped%s.\n", ui.Bold, ui.Yellow, ui.Reset)
			return ErrInterrupted
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
		cmd := exec.Command(opts.Command[0], opts.Command[1:]...)
		cmd.Stdin = strings.NewReader(opts.Prompt)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		setProcAttr(cmd)

		// Start then publish: ensures cmd.Process is populated before the signal
		// handler can see currentCmd, preventing races where signals arrive between
		// setting currentCmd and the process actually existing.
		if err := cmd.Start(); err != nil {
			mu.Lock()
			currentCmd = nil
			mu.Unlock()
			// Start failure counts as iteration failure
			failStreak++
			fmt.Printf("  %s%s✗ Iteration %d failed to start%s: %v. Consecutive failures: %d/%d\n",
				ui.Bold, ui.Red, iterNum, ui.Reset, err, failStreak, maxFailStreak,
			)
			if failStreak >= maxFailStreak {
				fmt.Printf("  %s%s✗ Too many consecutive failures. Stopping.%s\n", ui.Bold, ui.Red, ui.Reset)
				return fmt.Errorf("stopped after %d consecutive failures", maxFailStreak)
			}
			i++
			continue
		}

		mu.Lock()
		currentCmd = cmd
		mu.Unlock()

		err := cmd.Wait()

		mu.Lock()
		currentCmd = nil
		mu.Unlock()

		// Check for signal files immediately after subprocess exits
		if stop := checkSignalFiles(); stop {
			return nil
		}

		// If user requested stop (first Ctrl+C), exit gracefully now that the iteration is done
		if stopping.Load() {
			fmt.Printf("\n  %s%sStopped after iteration %d%s.\n", ui.Bold, ui.Yellow, iterNum, ui.Reset)
			return ErrInterrupted
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
// Only regular files are treated as signals (symlinks and directories are ignored).
// Returns true if the engine should stop.
func checkSignalFiles() bool {
	if fsutil.IsRegularFile(ui.SignalComplete) {
		fmt.Printf("\n  %s%s✓ All tasks complete%s (%s found). Stopping.\n", ui.Bold, ui.Green, ui.Reset, ui.SignalComplete)
		return true
	}
	// Try to open and read in one pass; fall back to existence check for unreadable files
	if f, err := fsutil.OpenRegularFile(ui.SignalNeedsApproval); err == nil {
		fmt.Printf("\n  %s%s⏸ Task needs human approval%s (%s found):\n", ui.Bold, ui.Yellow, ui.Reset, ui.SignalNeedsApproval)
		content, readErr := readCappedFromFile(f, maxApprovalFileSize)
		_ = f.Close()
		if readErr == nil {
			trimmed := strings.TrimSpace(content)
			if trimmed != "" {
				fmt.Println(trimmed)
			} else {
				fmt.Println("  (no details provided)")
			}
		} else {
			fmt.Printf("  (could not read details: %v)\n", readErr)
		}
		return true
	} else if fsutil.IsRegularFile(ui.SignalNeedsApproval) {
		// File exists but can't be opened (e.g. permissions) — still honor the signal
		fmt.Printf("\n  %s%s⏸ Task needs human approval%s (%s found):\n", ui.Bold, ui.Yellow, ui.Reset, ui.SignalNeedsApproval)
		fmt.Printf("  (could not read details: %v)\n", err)
		return true
	}
	return false
}

// readCappedFromFile reads up to maxBytes from an already-open file, appending a truncation notice if needed.
// Truncation is aligned to a valid UTF-8 boundary to avoid garbled output.
func readCappedFromFile(f *os.File, maxBytes int64) (string, error) {
	data, err := io.ReadAll(io.LimitReader(f, maxBytes+1))
	if err != nil {
		return "", err
	}
	if int64(len(data)) > maxBytes {
		// Walk backwards to a UTF-8 rune boundary (at most 3 bytes back)
		cut := int(maxBytes)
		for cut > 0 && !utf8.RuneStart(data[cut]) {
			cut--
		}
		return string(data[:cut]) + "\n  ... (truncated)", nil
	}
	return string(data), nil
}
