package engine

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/hl/brr/internal/ui"
)

func TestRunEmptyCommand(t *testing.T) {
	err := Run(Options{Prompt: "hello", Command: nil})
	if err == nil {
		t.Error("expected error for empty command")
	}
}

func TestRunMaxIterations(t *testing.T) {
	t.Chdir(t.TempDir())

	// Child appends to a counter file so we can verify iteration count
	counter := filepath.Join(".", "counter")
	var cmd []string
	if runtime.GOOS == "windows" {
		cmd = []string{"cmd", "/c", "echo", "x", ">>", counter}
	} else {
		cmd = []string{"sh", "-c", "echo x >> " + counter}
	}

	err := Run(Options{
		Prompt:  "test",
		Max:     3,
		Command: cmd,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(counter)
	if err != nil {
		t.Fatal("counter file not created")
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 3 {
		t.Errorf("expected 3 iterations, got %d", len(lines))
	}
}

func TestRunFailStreak(t *testing.T) {
	t.Chdir(t.TempDir())

	// "false" always exits 1 on Unix
	var cmd []string
	if runtime.GOOS == "windows" {
		cmd = []string{"cmd", "/c", "exit", "1"}
	} else {
		cmd = []string{"false"}
	}

	err := Run(Options{
		Prompt:  "test",
		Max:     10,
		Command: cmd,
	})
	if err == nil {
		t.Error("expected error after consecutive failures")
	}
}

func TestRunSignalFileComplete(t *testing.T) {
	t.Chdir(t.TempDir())

	// Create the signal file before running — engine should detect it before first iteration
	if err := os.WriteFile(ui.SignalComplete, []byte("done"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Use a command that creates a marker file so we can verify it never ran
	marker := filepath.Join(".", "child-ran")
	var cmd []string
	if runtime.GOOS == "windows" {
		cmd = []string{"cmd", "/c", "echo", "ran", ">", marker}
	} else {
		cmd = []string{"sh", "-c", "touch " + marker}
	}

	err := Run(Options{
		Prompt:  "test",
		Max:     5,
		Command: cmd,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The child should never have run because the signal file pre-existed
	if _, err := os.Stat(marker); err == nil {
		t.Error("child process ran despite pre-existing .brr-complete signal file")
	}
}

func TestRunSignalFileNeedsApproval(t *testing.T) {
	t.Chdir(t.TempDir())

	if err := os.WriteFile(ui.SignalNeedsApproval, []byte("please review"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Use a command that creates a marker file so we can verify it never ran
	marker := filepath.Join(".", "child-ran")
	var cmd []string
	if runtime.GOOS == "windows" {
		cmd = []string{"cmd", "/c", "echo", "ran", ">", marker}
	} else {
		cmd = []string{"sh", "-c", "touch " + marker}
	}

	err := Run(Options{
		Prompt:  "test",
		Max:     5,
		Command: cmd,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The child should never have run because the signal file pre-existed
	if _, err := os.Stat(marker); err == nil {
		t.Error("child process ran despite pre-existing .brr-needs-approval signal file")
	}
}

func TestRunSignalFileCreatedByChild(t *testing.T) {
	t.Chdir(t.TempDir())

	// Child creates the signal file — engine should detect after child exits
	signalPath := filepath.Join(".", ui.SignalComplete)

	var cmd []string
	if runtime.GOOS == "windows" {
		cmd = []string{"cmd", "/c", "echo", "done", ">", signalPath}
	} else {
		cmd = []string{"sh", "-c", "touch " + signalPath}
	}

	err := Run(Options{
		Prompt:  "test",
		Max:     5,
		Command: cmd,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckSignalFilesNone(t *testing.T) {
	t.Chdir(t.TempDir())

	if checkSignalFiles() {
		t.Error("expected false when no signal files exist")
	}
}

func TestCheckSignalFilesComplete(t *testing.T) {
	t.Chdir(t.TempDir())

	if err := os.WriteFile(ui.SignalComplete, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	if !checkSignalFiles() {
		t.Error("expected true when .brr-complete exists")
	}
}

func TestCheckSignalFilesNeedsApproval(t *testing.T) {
	t.Chdir(t.TempDir())

	if err := os.WriteFile(ui.SignalNeedsApproval, []byte("review this"), 0o644); err != nil {
		t.Fatal(err)
	}

	if !checkSignalFiles() {
		t.Error("expected true when .brr-needs-approval exists")
	}
}

func TestCheckSignalFilesNeedsApprovalUnreadable(t *testing.T) {
	t.Chdir(t.TempDir())

	// Write-only file: Stat succeeds but ReadFile fails
	if err := os.WriteFile(ui.SignalNeedsApproval, []byte("secret"), 0o200); err != nil {
		t.Fatal(err)
	}
	// Make it truly unreadable (not owner-writable-only, but no read)
	if err := os.Chmod(ui.SignalNeedsApproval, 0o000); err != nil {
		t.Skip("cannot remove read permission")
	}
	defer os.Chmod(ui.SignalNeedsApproval, 0o644)

	if !checkSignalFiles() {
		t.Error("expected true when .brr-needs-approval exists but is unreadable")
	}
}

func TestRunSignalFileCleanedAfterEarlyStop(t *testing.T) {
	t.Chdir(t.TempDir())

	// Pre-existing signal file should be cleaned up after the engine returns
	if err := os.WriteFile(ui.SignalComplete, []byte("done"), 0o644); err != nil {
		t.Fatal(err)
	}

	marker := filepath.Join(".", "child-ran")
	var cmd []string
	if runtime.GOOS == "windows" {
		cmd = []string{"cmd", "/c", "echo", "ran", ">", marker}
	} else {
		cmd = []string{"sh", "-c", "touch " + marker}
	}

	err := Run(Options{
		Prompt:  "test",
		Max:     5,
		Command: cmd,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Signal file should have been cleaned up
	if _, err := os.Stat(ui.SignalComplete); err == nil {
		t.Error("expected .brr-complete to be cleaned up after early stop")
	}
}
