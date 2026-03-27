package cli

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// newTestRootCmd creates a fresh cobra command with the same flags as rootCmd.
// Using a fresh command per test avoids global state leakage.
func newTestRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "brr <prompt> [flags]",
		Args:         cobra.ExactArgs(1),
		RunE:         run,
		SilenceUsage: true,
	}
	cmd.Flags().IntP("max", "m", 0, "max iterations")
	cmd.Flags().StringP("profile", "p", "", "profile name")
	return cmd
}

func writeTestConfig(t *testing.T) {
	t.Helper()
	var yaml string
	if runtime.GOOS == "windows" {
		yaml = `default: test
profiles:
  test:
    command: cmd
    args: ["/c", "exit 0"]
  failing:
    command: cmd
    args: ["/c", "exit 1"]
`
	} else {
		yaml = `default: test
profiles:
  test:
    command: "true"
    args: []
  failing:
    command: "false"
    args: []
`
	}
	if err := os.WriteFile(".brr.yaml", []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestRunIntegrationSuccess(t *testing.T) {
	t.Chdir(t.TempDir())
	writeTestConfig(t)

	cmd := newTestRootCmd()
	cmd.SetArgs([]string{"hello world", "-m", "1"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunIntegrationMultipleIterations(t *testing.T) {
	t.Chdir(t.TempDir())
	writeTestConfig(t)

	cmd := newTestRootCmd()
	cmd.SetArgs([]string{"hello world", "-m", "3"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunIntegrationExplicitProfile(t *testing.T) {
	t.Chdir(t.TempDir())
	writeTestConfig(t)

	cmd := newTestRootCmd()
	cmd.SetArgs([]string{"hello world", "-m", "1", "-p", "test"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunIntegrationBadProfile(t *testing.T) {
	t.Chdir(t.TempDir())
	writeTestConfig(t)

	cmd := newTestRootCmd()
	cmd.SetArgs([]string{"hello world", "-m", "1", "-p", "nonexistent"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for nonexistent profile")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' in error, got: %v", err)
	}
}

func TestRunIntegrationMissingConfig(t *testing.T) {
	t.Chdir(t.TempDir())

	cmd := newTestRootCmd()
	cmd.SetArgs([]string{"hello world", "-m", "1"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when no config exists")
	}
	if !strings.Contains(err.Error(), "no config found") {
		t.Errorf("expected 'no config found' error, got: %v", err)
	}
}

func TestRunIntegrationEmptyPrompt(t *testing.T) {
	t.Chdir(t.TempDir())
	writeTestConfig(t)

	cmd := newTestRootCmd()
	cmd.SetArgs([]string{"   ", "-m", "1"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for whitespace-only prompt")
	}
	if !strings.Contains(err.Error(), "prompt is empty") {
		t.Errorf("expected 'prompt is empty' error, got: %v", err)
	}
}

func TestRunIntegrationNegativeMax(t *testing.T) {
	t.Chdir(t.TempDir())
	writeTestConfig(t)

	cmd := newTestRootCmd()
	cmd.SetArgs([]string{"hello", "-m", "-1"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for negative max")
	}
	if !strings.Contains(err.Error(), "--max must be >= 0") {
		t.Errorf("expected '--max must be >= 0' error, got: %v", err)
	}
}

func TestRunIntegrationFailingCommand(t *testing.T) {
	t.Chdir(t.TempDir())
	writeTestConfig(t)

	cmd := newTestRootCmd()
	cmd.SetArgs([]string{"hello", "-m", "10", "-p", "failing"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for failing command")
	}
	if !strings.Contains(err.Error(), "consecutive failures") {
		t.Errorf("expected consecutive failures error, got: %v", err)
	}
}

func TestRunIntegrationPromptFromFile(t *testing.T) {
	t.Chdir(t.TempDir())
	writeTestConfig(t)

	if err := os.WriteFile("task.md", []byte("do the thing"), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := newTestRootCmd()
	cmd.SetArgs([]string{"task.md", "-m", "1"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunIntegrationNoArgs(t *testing.T) {
	t.Chdir(t.TempDir())
	writeTestConfig(t)

	cmd := newTestRootCmd()
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing args")
	}
}

func TestSetVersionFormat(t *testing.T) {
	SetVersion("1.2.3", "abc123")
	expected := "1.2.3 (abc123)"
	if rootCmd.Version != expected {
		t.Errorf("expected version %q, got %q", expected, rootCmd.Version)
	}
}

func TestPrintConfigFormatting(t *testing.T) {
	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printConfig("plan", "claude", []string{"claude", "-p"}, 0)

	w.Close()
	os.Stdout = old

	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	if !strings.Contains(output, "plan") {
		t.Error("expected prompt name in output")
	}
	if !strings.Contains(output, "claude") {
		t.Error("expected profile name in output")
	}
	if !strings.Contains(output, "unlimited") {
		t.Error("expected 'unlimited' for max=0")
	}
}

func TestPrintConfigWithMax(t *testing.T) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printConfig("plan", "claude", []string{"claude"}, 5)

	w.Close()
	os.Stdout = old

	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	if !strings.Contains(output, fmt.Sprintf("%d", 5)) {
		t.Error("expected max count in output")
	}
}
