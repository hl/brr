package engine

import (
	"os"
	"path/filepath"
	"runtime"
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

	// "true" always exits 0 on Unix; "cmd /c exit 0" on Windows
	var cmd []string
	if runtime.GOOS == "windows" {
		cmd = []string{"cmd", "/c", "exit", "0"}
	} else {
		cmd = []string{"true"}
	}

	err := Run(Options{
		Prompt:  "test",
		Max:     3,
		Command: cmd,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
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

	var cmd []string
	if runtime.GOOS == "windows" {
		cmd = []string{"cmd", "/c", "echo", "hello"}
	} else {
		cmd = []string{"echo", "hello"}
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

func TestRunSignalFileNeedsApproval(t *testing.T) {
	t.Chdir(t.TempDir())

	if err := os.WriteFile(ui.SignalNeedsApproval, []byte("please review"), 0o644); err != nil {
		t.Fatal(err)
	}

	var cmd []string
	if runtime.GOOS == "windows" {
		cmd = []string{"cmd", "/c", "echo", "hello"}
	} else {
		cmd = []string{"echo", "hello"}
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
