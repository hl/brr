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
	_, err := Run(Options{Prompt: "hello", Command: nil})
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

	result, err := Run(Options{
		Prompt:  "test",
		Max:     3,
		Command: cmd,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Reason != ReasonMaxIterations {
		t.Errorf("expected ReasonMaxIterations, got %d", result.Reason)
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
	counter := filepath.Join(".", "fail-counter")
	var cmd []string
	if runtime.GOOS == "windows" {
		cmd = []string{"cmd", "/c", "echo x >> " + counter + " & exit 1"}
	} else {
		cmd = []string{"sh", "-c", "echo x >> " + counter + " && exit 1"}
	}

	result, err := Run(Options{
		Prompt:  "test",
		Max:     10,
		Command: cmd,
	})
	if err == nil {
		t.Fatal("expected error after consecutive failures")
	}
	if result.Reason != ReasonFailStreak {
		t.Errorf("expected ReasonFailStreak, got %d", result.Reason)
	}
	if !strings.Contains(err.Error(), "stopped after 3 consecutive failures") {
		t.Errorf("expected fail-streak error message, got: %v", err)
	}

	// Verify exactly maxFailStreak iterations ran
	data, readErr := os.ReadFile(counter)
	if readErr != nil {
		t.Fatal("counter file not created")
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != maxFailStreak {
		t.Errorf("expected %d iterations, got %d", maxFailStreak, len(lines))
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

	result, err := Run(Options{
		Prompt:  "test",
		Max:     5,
		Command: cmd,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Reason != ReasonComplete {
		t.Errorf("expected ReasonComplete, got %d", result.Reason)
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

	result, err := Run(Options{
		Prompt:  "test",
		Max:     5,
		Command: cmd,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Reason != ReasonApproval {
		t.Errorf("expected ReasonApproval, got %d", result.Reason)
	}
	if result.ApprovalContent != "please review" {
		t.Errorf("expected approval content 'please review', got %q", result.ApprovalContent)
	}

	// The child should never have run because the signal file pre-existed
	if _, err := os.Stat(marker); err == nil {
		t.Error("child process ran despite pre-existing .brr-needs-approval signal file")
	}
}

func TestRunSignalFileCreatedByChild(t *testing.T) {
	t.Chdir(t.TempDir())

	// Child creates the signal file AND a counter — engine should stop after one iteration
	signalPath := filepath.Join(".", ui.SignalComplete)
	counter := filepath.Join(".", "iter-counter")

	var cmd []string
	if runtime.GOOS == "windows" {
		cmd = []string{"cmd", "/c", "echo x >> " + counter + " & echo done > " + signalPath}
	} else {
		cmd = []string{"sh", "-c", "echo x >> " + counter + " && touch " + signalPath}
	}

	result, err := Run(Options{
		Prompt:  "test",
		Max:     5,
		Command: cmd,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Reason != ReasonComplete {
		t.Errorf("expected ReasonComplete, got %d", result.Reason)
	}

	// Verify only one iteration ran (engine stopped after detecting signal file)
	data, readErr := os.ReadFile(counter)
	if readErr != nil {
		t.Fatal("counter file not created")
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 1 {
		t.Errorf("expected 1 iteration before signal file stopped the loop, got %d", len(lines))
	}
}

func TestRunMaxIterationsWithFailure(t *testing.T) {
	t.Chdir(t.TempDir())

	// Command that always fails — but max=2 is less than maxFailStreak(3)
	var cmd []string
	if runtime.GOOS == "windows" {
		cmd = []string{"cmd", "/c", "exit 1"}
	} else {
		cmd = []string{"sh", "-c", "exit 1"}
	}

	result, err := Run(Options{
		Prompt:  "test",
		Max:     2,
		Command: cmd,
	})
	if err == nil {
		t.Fatal("expected error when max iterations reached with failures")
	}
	if result.Reason != ReasonMaxIterations {
		t.Errorf("expected ReasonMaxIterations, got %d", result.Reason)
	}
	if !strings.Contains(err.Error(), "last iteration failed") {
		t.Errorf("expected 'last iteration failed' error, got: %v", err)
	}
}

func TestRunMaxIterationsLastSucceeds(t *testing.T) {
	t.Chdir(t.TempDir())

	// Command that fails on first call, succeeds on second (uses counter file)
	counter := filepath.Join(".", "attempt")
	var cmd []string
	if runtime.GOOS == "windows" {
		cmd = []string{"cmd", "/c", "echo x >> " + counter + " & exit 0"}
	} else {
		cmd = []string{"sh", "-c", "echo x >> " + counter}
	}

	result, err := Run(Options{
		Prompt:  "test",
		Max:     2,
		Command: cmd,
	})
	if err != nil {
		t.Fatalf("expected nil when last iteration succeeds, got: %v", err)
	}
	if result.Reason != ReasonMaxIterations {
		t.Errorf("expected ReasonMaxIterations, got %d", result.Reason)
	}
}

func TestSignalFileDirNotDeletedOnCleanup(t *testing.T) {
	t.Chdir(t.TempDir())

	// Create a directory named .brr-complete — engine should NOT delete it
	if err := os.Mkdir(ui.SignalComplete, 0o755); err != nil {
		t.Fatal(err)
	}

	var cmd []string
	if runtime.GOOS == "windows" {
		cmd = []string{"cmd", "/c", "exit 0"}
	} else {
		cmd = []string{"true"}
	}

	_, err := Run(Options{
		Prompt:  "test",
		Max:     1,
		Command: cmd,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Directory should still exist — removeIfRegular should have skipped it
	fi, statErr := os.Stat(ui.SignalComplete)
	if statErr != nil {
		t.Fatalf("directory %s was deleted by cleanup", ui.SignalComplete)
	}
	if !fi.IsDir() {
		t.Error("expected .brr-complete to still be a directory")
	}
}

func TestCheckSignalFilesNone(t *testing.T) {
	t.Chdir(t.TempDir())

	if checkSignalFiles() != nil {
		t.Error("expected nil when no signal files exist")
	}
}

func TestCheckSignalFilesComplete(t *testing.T) {
	t.Chdir(t.TempDir())

	if err := os.WriteFile(ui.SignalComplete, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	sig := checkSignalFiles()
	if sig == nil {
		t.Fatal("expected non-nil when .brr-complete exists")
	}
	if sig.reason != ReasonComplete {
		t.Errorf("expected ReasonComplete, got %d", sig.reason)
	}
}

func TestCheckSignalFilesNeedsApproval(t *testing.T) {
	t.Chdir(t.TempDir())

	if err := os.WriteFile(ui.SignalNeedsApproval, []byte("review this"), 0o644); err != nil {
		t.Fatal(err)
	}

	sig := checkSignalFiles()
	if sig == nil {
		t.Fatal("expected non-nil when .brr-needs-approval exists")
	}
	if sig.reason != ReasonApproval {
		t.Errorf("expected ReasonApproval, got %d", sig.reason)
	}
	if sig.approvalContent != "review this" {
		t.Errorf("expected approval content 'review this', got %q", sig.approvalContent)
	}
}

func TestCheckSignalFilesDirectoryIgnored(t *testing.T) {
	t.Chdir(t.TempDir())

	// A directory named .brr-complete should NOT be treated as a signal file
	if err := os.Mkdir(ui.SignalComplete, 0o755); err != nil {
		t.Fatal(err)
	}

	if checkSignalFiles() != nil {
		t.Error("expected nil when .brr-complete is a directory, not a regular file")
	}
}

func TestCheckSignalFilesSymlinkIgnored(t *testing.T) {
	t.Chdir(t.TempDir())

	// A symlink named .brr-complete should NOT be treated as a signal file
	target := filepath.Join(t.TempDir(), "target")
	if err := os.WriteFile(target, []byte("done"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, ui.SignalComplete); err != nil {
		t.Skip("symlinks not supported")
	}

	if checkSignalFiles() != nil {
		t.Error("expected nil when .brr-complete is a symlink, not a regular file")
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

	sig := checkSignalFiles()
	if sig == nil {
		t.Fatal("expected non-nil when .brr-needs-approval exists but is unreadable")
	}
	if sig.reason != ReasonApproval {
		t.Errorf("expected ReasonApproval, got %d", sig.reason)
	}
	if sig.approvalContent != "" {
		t.Errorf("expected empty approval content for unreadable file, got %q", sig.approvalContent)
	}
}

func TestReadCappedSmallFile(t *testing.T) {
	t.Chdir(t.TempDir())

	if err := os.WriteFile("small.txt", []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open("small.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	content, err := readCappedFromFile(f, 4096)
	if err != nil {
		t.Fatal(err)
	}
	if content != "hello" {
		t.Errorf("expected 'hello', got %q", content)
	}
}

func TestReadCappedLargeFile(t *testing.T) {
	t.Chdir(t.TempDir())

	// Create a file larger than the cap
	big := strings.Repeat("x", 5000)
	if err := os.WriteFile("big.txt", []byte(big), 0o644); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open("big.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	content, err := readCappedFromFile(f, 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(content) > 200 { // 100 + truncation notice
		t.Errorf("expected capped content, got %d bytes", len(content))
	}
	if !strings.Contains(content, "truncated") {
		t.Error("expected truncation notice")
	}
}

func TestReadCappedUTF8Boundary(t *testing.T) {
	t.Chdir(t.TempDir())

	// Write a file with multi-byte UTF-8 characters (each emoji is 4 bytes)
	// Cut at a byte boundary that falls mid-codepoint
	emoji := "🔥🔥🔥🔥🔥" // 5 emojis × 4 bytes = 20 bytes
	if err := os.WriteFile("utf8.txt", []byte(emoji), 0o644); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open("utf8.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	// Cap at 6 bytes — falls mid-codepoint (second emoji starts at byte 4, ends at 8)
	content, err := readCappedFromFile(f, 6)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(content, "truncated") {
		t.Error("expected truncation notice")
	}
	// The truncated content (before the notice) must be valid UTF-8
	idx := strings.Index(content, "\n  ... (truncated)")
	if idx < 0 {
		t.Fatal("truncation notice not found")
	}
	prefix := content[:idx]
	// Should have backed up to byte 4 (one full emoji)
	if prefix != "🔥" {
		t.Errorf("expected one full emoji before truncation, got %q", prefix)
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

	_, err := Run(Options{
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
