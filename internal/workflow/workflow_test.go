package workflow

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/hl/brr/internal/config"
	"github.com/hl/brr/internal/engine"
)

func TestLoadValidWorkflow(t *testing.T) {
	data := []byte(`
stages:
  - prompt: spec
    max: 3
  - prompt: build
    max: 100
    cycle: true
  - prompt: review
    max: 1
max_cycles: 5
`)
	wf, err := Load(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(wf.Stages) != 3 {
		t.Errorf("expected 3 stages, got %d", len(wf.Stages))
	}
	if wf.MaxCycles != 5 {
		t.Errorf("expected max_cycles 5, got %d", wf.MaxCycles)
	}
	if !wf.Stages[1].Cycle {
		t.Error("expected stage 2 to have cycle: true")
	}
}

func TestLoadDefaultMaxCycles(t *testing.T) {
	data := []byte(`
stages:
  - prompt: build
    max: 10
`)
	wf, err := Load(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if wf.MaxCycles != 3 {
		t.Errorf("expected default max_cycles 3, got %d", wf.MaxCycles)
	}
}

func TestLoadNoStages(t *testing.T) {
	data := []byte(`max_cycles: 3`)
	_, err := Load(data)
	if err == nil {
		t.Fatal("expected error for empty stages")
	}
	if !strings.Contains(err.Error(), "no stages") {
		t.Errorf("expected 'no stages' error, got: %v", err)
	}
}

func TestLoadEmptyPrompt(t *testing.T) {
	data := []byte(`
stages:
  - prompt: ""
    max: 3
`)
	_, err := Load(data)
	if err == nil {
		t.Fatal("expected error for empty prompt")
	}
	if !strings.Contains(err.Error(), "prompt is required") {
		t.Errorf("expected 'prompt is required' error, got: %v", err)
	}
}

func TestLoadZeroMax(t *testing.T) {
	data := []byte(`
stages:
  - prompt: build
    max: 0
`)
	_, err := Load(data)
	if err == nil {
		t.Fatal("expected error for zero max")
	}
	if !strings.Contains(err.Error(), "max must be >= 1") {
		t.Errorf("expected 'max must be >= 1' error, got: %v", err)
	}
}

func TestLoadMultipleCycleStages(t *testing.T) {
	data := []byte(`
stages:
  - prompt: build
    max: 10
    cycle: true
  - prompt: verify
    max: 3
    cycle: true
`)
	_, err := Load(data)
	if err == nil {
		t.Fatal("expected error for multiple cycle stages")
	}
	if !strings.Contains(err.Error(), "at most one stage") {
		t.Errorf("expected 'at most one stage' error, got: %v", err)
	}
}

func TestLoadInvalidYAML(t *testing.T) {
	data := []byte(`{{{not yaml`)
	_, err := Load(data)
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestResolveProjectPath(t *testing.T) {
	t.Chdir(t.TempDir())

	dir := filepath.Join(".brr", "workflows")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := "stages:\n  - prompt: build\n    max: 1\n"
	if err := os.WriteFile(filepath.Join(dir, "ship.yaml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	data, err := Resolve("ship")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(string(data), "build") {
		t.Error("expected workflow content")
	}
}

func TestResolveNotFound(t *testing.T) {
	t.Chdir(t.TempDir())

	_, err := Resolve("nonexistent")
	if err == nil {
		t.Fatal("expected error for missing workflow")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

func TestResolvePathTraversal(t *testing.T) {
	_, err := Resolve("../evil")
	if err == nil {
		t.Fatal("expected error for path traversal")
	}
	if !strings.Contains(err.Error(), "invalid workflow name") {
		t.Errorf("expected 'invalid workflow name' error, got: %v", err)
	}
}

func TestResolveProjectWorkflowSymlinkRejected(t *testing.T) {
	t.Chdir(t.TempDir())

	dir := filepath.Join(".brr", "workflows")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(t.TempDir(), "ship.yaml")
	if err := os.WriteFile(target, []byte("stages:\n  - prompt: build\n    max: 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(dir, "ship.yaml")); err != nil {
		t.Skip("symlinks not supported")
	}

	_, err := Resolve("ship")
	if err == nil {
		t.Fatal("expected symlinked workflow to be rejected")
	}
	if !strings.Contains(err.Error(), filepath.Join(".brr", "workflows", "ship.yaml")) {
		t.Errorf("expected error to mention workflow path, got: %v", err)
	}
}

func TestHasUnfinishedTasks(t *testing.T) {
	t.Chdir(t.TempDir())

	// No file — no tasks
	if hasUnfinishedTasks() {
		t.Error("expected no tasks when file doesn't exist")
	}

	// File with unchecked tasks
	if err := os.WriteFile(planFile, []byte("- [ ] **1.1 — Task** — files: foo.go\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !hasUnfinishedTasks() {
		t.Error("expected unfinished tasks")
	}

	// File with all tasks done (no unchecked marker)
	if err := os.WriteFile(planFile, []byte("All done!\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if hasUnfinishedTasks() {
		t.Error("expected no unfinished tasks")
	}
}

func TestRunSequentialStages(t *testing.T) {
	t.Chdir(t.TempDir())

	// Each stage command appends its prompt to a log file so we can verify order
	logFile := filepath.Join(".", "stage-log")
	var cmd []string
	if runtime.GOOS == "windows" {
		// Windows: read from stdin via "more" and append
		cmd = []string{"cmd", "/c", "set /p x= & echo %x% >> " + logFile}
	} else {
		cmd = []string{"sh", "-c", "cat >> " + logFile}
	}

	wf := Workflow{
		Stages: []Stage{
			{Prompt: "first", Max: 1},
			{Prompt: "second", Max: 1},
			{Prompt: "third", Max: 1},
		},
		MaxCycles: 1,
	}

	cfg := config.Config{
		Default:  "test",
		Profiles: map[string]config.Profile{"test": {Command: cmd[0], Args: cmd[1:]}},
	}

	result, err := Run(Options{
		Name:     "test",
		Workflow: wf,
		Config:   cfg,
		ResolvePrompt: func(name string) (string, error) {
			return name + "\n", nil
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	data, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("log file not created: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "first") || !strings.Contains(content, "second") || !strings.Contains(content, "third") {
		t.Errorf("expected all three prompts in log, got: %s", content)
	}

	// Verify order: first should appear before second, second before third
	i1 := strings.Index(content, "first")
	i2 := strings.Index(content, "second")
	i3 := strings.Index(content, "third")
	if i1 >= i2 || i2 >= i3 {
		t.Errorf("stages ran out of order: first@%d second@%d third@%d", i1, i2, i3)
	}
}

func TestRunCycleOnUnfinishedTasks(t *testing.T) {
	t.Chdir(t.TempDir())

	// Stage 1 (cycle point): creates a counter file
	// Stage 2: creates IMPLEMENTATION_PLAN.md with a task on the first pass,
	//          removes it on the second pass
	counter := filepath.Join(".", "cycle-counter")
	var buildCmd, reviewCmd []string
	if runtime.GOOS == "windows" {
		buildCmd = []string{"cmd", "/c", "echo x >> " + counter}
		reviewCmd = []string{"cmd", "/c",
			"if exist " + counter + " (findstr /c:\"xx\" " + counter + " >nul 2>&1 && (echo done > " + planFile + ") || (echo - [ ] fix > " + planFile + "))"}
	} else {
		buildCmd = []string{"sh", "-c", "echo x >> " + counter}
		// First call: creates plan with task. Second call: lines >= 2 → clears the plan.
		reviewCmd = []string{"sh", "-c",
			`lines=$(wc -l < ` + counter + ` | tr -d ' '); if [ "$lines" -ge 2 ]; then echo "done" > ` + planFile + `; else echo "- [ ] fix" > ` + planFile + `; fi`}
	}

	wf := Workflow{
		Stages: []Stage{
			{Prompt: "build", Max: 1, Cycle: true},
			{Prompt: "review", Max: 1},
		},
		MaxCycles: 3,
	}

	cfg := config.Config{
		Default: "build",
		Profiles: map[string]config.Profile{
			"build":  {Command: buildCmd[0], Args: buildCmd[1:]},
			"review": {Command: reviewCmd[0], Args: reviewCmd[1:]},
		},
	}

	wf.Stages[1].Profile = "review"

	_, err := Run(Options{
		Name:     "test",
		Workflow: wf,
		Config:   cfg,
		ResolvePrompt: func(name string) (string, error) {
			return "go", nil
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have run build twice (initial + one cycle)
	data, err := os.ReadFile(counter)
	if err != nil {
		t.Fatal("counter not created")
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 {
		t.Errorf("expected 2 build iterations (initial + cycle), got %d", len(lines))
	}
}

func TestRunStageProfileOverride(t *testing.T) {
	t.Chdir(t.TempDir())

	// Track which profile was used via different output
	logFile := filepath.Join(".", "profile-log")

	var defaultCmd, overrideCmd []string
	if runtime.GOOS == "windows" {
		defaultCmd = []string{"cmd", "/c", "echo default >> " + logFile}
		overrideCmd = []string{"cmd", "/c", "echo override >> " + logFile}
	} else {
		defaultCmd = []string{"sh", "-c", "echo default >> " + logFile}
		overrideCmd = []string{"sh", "-c", "echo override >> " + logFile}
	}

	wf := Workflow{
		Stages: []Stage{
			{Prompt: "first", Max: 1},
			{Prompt: "second", Max: 1, Profile: "special"},
		},
		MaxCycles: 1,
	}

	cfg := config.Config{
		Default: "normal",
		Profiles: map[string]config.Profile{
			"normal":  {Command: defaultCmd[0], Args: defaultCmd[1:]},
			"special": {Command: overrideCmd[0], Args: overrideCmd[1:]},
		},
	}

	_, err := Run(Options{
		Name:     "test",
		Workflow: wf,
		Config:   cfg,
		ResolvePrompt: func(name string) (string, error) {
			return "go", nil
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatal("log not created")
	}
	content := string(data)
	if !strings.Contains(content, "default") {
		t.Error("expected default profile for first stage")
	}
	if !strings.Contains(content, "override") {
		t.Error("expected override profile for second stage")
	}
}

func TestRunSignalComplete(t *testing.T) {
	t.Chdir(t.TempDir())

	// Stage creates .brr-complete — workflow should continue to next stage
	var cmd1, cmd2 []string
	marker := filepath.Join(".", "second-ran")
	if runtime.GOOS == "windows" {
		cmd1 = []string{"cmd", "/c", "echo done > .brr-complete"}
		cmd2 = []string{"cmd", "/c", "echo ran > " + marker}
	} else {
		cmd1 = []string{"sh", "-c", "touch .brr-complete"}
		cmd2 = []string{"sh", "-c", "touch " + marker}
	}

	wf := Workflow{
		Stages: []Stage{
			{Prompt: "first", Max: 5, Profile: "cmd1"},
			{Prompt: "second", Max: 1, Profile: "cmd2"},
		},
		MaxCycles: 1,
	}

	cfg := config.Config{
		Default: "cmd1",
		Profiles: map[string]config.Profile{
			"cmd1": {Command: cmd1[0], Args: cmd1[1:]},
			"cmd2": {Command: cmd2[0], Args: cmd2[1:]},
		},
	}

	_, err := Run(Options{
		Name:     "test",
		Workflow: wf,
		Config:   cfg,
		ResolvePrompt: func(name string) (string, error) {
			return "go", nil
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Second stage should have run
	if _, err := os.Stat(marker); err != nil {
		t.Error("second stage did not run after first stage completed via .brr-complete")
	}
}

func TestRunSignalFailed(t *testing.T) {
	t.Chdir(t.TempDir())

	// First stage creates .brr-failed — workflow should stop and preserve state
	var cmd1, cmd2 []string
	marker := filepath.Join(".", "second-ran")
	if runtime.GOOS == "windows" {
		cmd1 = []string{"cmd", "/c", "echo error > .brr-failed"}
		cmd2 = []string{"cmd", "/c", "echo ran > " + marker}
	} else {
		cmd1 = []string{"sh", "-c", "echo 'error details' > .brr-failed"}
		cmd2 = []string{"sh", "-c", "touch " + marker}
	}

	wf := Workflow{
		Stages: []Stage{
			{Prompt: "first", Max: 5, Profile: "cmd1"},
			{Prompt: "second", Max: 1, Profile: "cmd2"},
		},
		MaxCycles: 1,
	}

	cfg := config.Config{
		Default: "cmd1",
		Profiles: map[string]config.Profile{
			"cmd1": {Command: cmd1[0], Args: cmd1[1:]},
			"cmd2": {Command: cmd2[0], Args: cmd2[1:]},
		},
	}

	result, err := Run(Options{
		Name:     "test",
		Workflow: wf,
		Config:   cfg,
		ResolvePrompt: func(name string) (string, error) {
			return "go", nil
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Reason != engine.ReasonFailed {
		t.Errorf("expected ReasonFailed, got %d", result.Reason)
	}

	// Second stage should NOT have run
	if _, err := os.Stat(marker); err == nil {
		t.Error("second stage ran despite first stage failing via .brr-failed")
	}

	// State file should be preserved for resume
	if _, err := os.Stat(StateFile); err != nil {
		t.Error("expected state file to be preserved on failure")
	}
}

func TestRunFailStreakStopsWorkflow(t *testing.T) {
	t.Chdir(t.TempDir())

	var cmd []string
	if runtime.GOOS == "windows" {
		cmd = []string{"cmd", "/c", "exit 1"}
	} else {
		cmd = []string{"sh", "-c", "exit 1"}
	}

	wf := Workflow{
		Stages: []Stage{
			{Prompt: "failing", Max: 10},
		},
		MaxCycles: 1,
	}

	cfg := config.Config{
		Default:  "test",
		Profiles: map[string]config.Profile{"test": {Command: cmd[0], Args: cmd[1:]}},
	}

	_, err := Run(Options{
		Name:     "test",
		Workflow: wf,
		Config:   cfg,
		ResolvePrompt: func(name string) (string, error) {
			return "go", nil
		},
	})
	if err == nil {
		t.Fatal("expected error from fail streak")
	}
	if !strings.Contains(err.Error(), "failing") {
		t.Errorf("error should mention the failing stage, got: %v", err)
	}
}

func TestRunMaxCyclesLimit(t *testing.T) {
	t.Chdir(t.TempDir())

	// Stage always creates a plan with tasks — should stop after max_cycles
	counter := filepath.Join(".", "cycle-counter")
	var cmd []string
	if runtime.GOOS == "windows" {
		cmd = []string{"cmd", "/c", "echo x >> " + counter + " & echo - [ ] task > " + planFile}
	} else {
		cmd = []string{"sh", "-c", "echo x >> " + counter + " && echo '- [ ] task' > " + planFile}
	}

	wf := Workflow{
		Stages: []Stage{
			{Prompt: "build", Max: 1, Cycle: true},
		},
		MaxCycles: 2,
	}

	cfg := config.Config{
		Default:  "test",
		Profiles: map[string]config.Profile{"test": {Command: cmd[0], Args: cmd[1:]}},
	}

	_, err := Run(Options{
		Name:     "test",
		Workflow: wf,
		Config:   cfg,
		ResolvePrompt: func(name string) (string, error) {
			return "go", nil
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have run 1 (initial) + 2 (cycles) = 3 times
	data, err := os.ReadFile(counter)
	if err != nil {
		t.Fatal("counter not created")
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 3 {
		t.Errorf("expected 3 iterations (1 initial + 2 cycles), got %d", len(lines))
	}
}

func TestLoadPerStageProfile(t *testing.T) {
	data := []byte(`
stages:
  - prompt: build
    max: 10
    profile: opus
  - prompt: review
    max: 1
`)
	wf, err := Load(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if wf.Stages[0].Profile != "opus" {
		t.Errorf("expected profile 'opus', got %q", wf.Stages[0].Profile)
	}
	if wf.Stages[1].Profile != "" {
		t.Errorf("expected empty profile, got %q", wf.Stages[1].Profile)
	}
}

// --- Resume tests ---

func makeTestCfg(cmd []string) config.Config {
	return config.Config{
		Default:  "test",
		Profiles: map[string]config.Profile{"test": {Command: cmd[0], Args: cmd[1:]}},
	}
}

func echoCmd() []string {
	if runtime.GOOS == "windows" {
		return []string{"cmd", "/c", "echo ok"}
	}
	return []string{"true"}
}

func TestResumeSkipsCompletedStages(t *testing.T) {
	t.Chdir(t.TempDir())

	// Each stage appends its prompt to a log
	logFile := filepath.Join(".", "stage-log")
	var cmd []string
	if runtime.GOOS == "windows" {
		cmd = []string{"cmd", "/c", "set /p x= & echo %x% >> " + logFile}
	} else {
		cmd = []string{"sh", "-c", "cat >> " + logFile}
	}

	wf := Workflow{
		Stages: []Stage{
			{Prompt: "first", Max: 1},
			{Prompt: "second", Max: 1},
			{Prompt: "third", Max: 1},
		},
		MaxCycles: 1,
	}
	cfg := makeTestCfg(cmd)

	// Pre-seed state: skip to stage index 2 (third stage)
	trySaveState(&State{Workflow: "test", Stage: 2, Cycle: 0, StartSHA: "abc123"})

	_, err := Run(Options{
		Name:     "test",
		Workflow: wf,
		Config:   cfg,
		ResolvePrompt: func(name string) (string, error) {
			return name + "\n", nil
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatal("log not created")
	}
	content := string(data)

	// Only the third stage should have run
	if strings.Contains(content, "first") || strings.Contains(content, "second") {
		t.Errorf("expected only third stage to run, got: %s", content)
	}
	if !strings.Contains(content, "third") {
		t.Error("expected third stage to run")
	}
}

func TestResumeMismatchedWorkflowStartsFresh(t *testing.T) {
	t.Chdir(t.TempDir())

	logFile := filepath.Join(".", "stage-log")
	var cmd []string
	if runtime.GOOS == "windows" {
		cmd = []string{"cmd", "/c", "set /p x= & echo %x% >> " + logFile}
	} else {
		cmd = []string{"sh", "-c", "cat >> " + logFile}
	}

	wf := Workflow{
		Stages: []Stage{
			{Prompt: "alpha", Max: 1},
			{Prompt: "beta", Max: 1},
		},
		MaxCycles: 1,
	}
	cfg := makeTestCfg(cmd)

	// State from a different workflow — should be ignored
	trySaveState(&State{Workflow: "other-workflow", Stage: 1, Cycle: 0, StartSHA: "abc"})

	_, err := Run(Options{
		Name:     "test",
		Workflow: wf,
		Config:   cfg,
		ResolvePrompt: func(name string) (string, error) {
			return name + "\n", nil
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatal("log not created")
	}
	content := string(data)

	// Both stages should have run (started fresh)
	if !strings.Contains(content, "alpha") || !strings.Contains(content, "beta") {
		t.Errorf("expected both stages to run on mismatched workflow, got: %s", content)
	}
}

func TestResumeOutOfBoundsStartsFresh(t *testing.T) {
	t.Chdir(t.TempDir())

	logFile := filepath.Join(".", "stage-log")
	var cmd []string
	if runtime.GOOS == "windows" {
		cmd = []string{"cmd", "/c", "set /p x= & echo %x% >> " + logFile}
	} else {
		cmd = []string{"sh", "-c", "cat >> " + logFile}
	}

	wf := Workflow{
		Stages:    []Stage{{Prompt: "only", Max: 1}},
		MaxCycles: 1,
	}
	cfg := makeTestCfg(cmd)

	// Stage index 5 is way out of bounds
	trySaveState(&State{Workflow: "test", Stage: 5, Cycle: 0, StartSHA: "abc"})

	_, err := Run(Options{
		Name:     "test",
		Workflow: wf,
		Config:   cfg,
		ResolvePrompt: func(name string) (string, error) {
			return name + "\n", nil
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatal("log not created")
	}
	if !strings.Contains(string(data), "only") {
		t.Error("expected stage to run after out-of-bounds reset")
	}
}

func TestResetFlag(t *testing.T) {
	t.Chdir(t.TempDir())

	logFile := filepath.Join(".", "stage-log")
	var cmd []string
	if runtime.GOOS == "windows" {
		cmd = []string{"cmd", "/c", "set /p x= & echo %x% >> " + logFile}
	} else {
		cmd = []string{"sh", "-c", "cat >> " + logFile}
	}

	wf := Workflow{
		Stages: []Stage{
			{Prompt: "first", Max: 1},
			{Prompt: "second", Max: 1},
		},
		MaxCycles: 1,
	}
	cfg := makeTestCfg(cmd)

	// Pre-seed state to skip to second stage
	trySaveState(&State{Workflow: "test", Stage: 1, Cycle: 0, StartSHA: "abc"})

	// --reset should ignore saved state
	_, err := Run(Options{
		Name:     "test",
		Workflow: wf,
		Config:   cfg,
		Reset:    true,
		ResolvePrompt: func(name string) (string, error) {
			return name + "\n", nil
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatal("log not created")
	}
	content := string(data)

	// Both stages should have run
	if !strings.Contains(content, "first") || !strings.Contains(content, "second") {
		t.Errorf("expected both stages with --reset, got: %s", content)
	}
}

func TestStateDeletedOnCompletion(t *testing.T) {
	t.Chdir(t.TempDir())

	cmd := echoCmd()
	wf := Workflow{
		Stages:    []Stage{{Prompt: "build", Max: 1}},
		MaxCycles: 1,
	}
	cfg := makeTestCfg(cmd)

	_, err := Run(Options{
		Name:     "test",
		Workflow: wf,
		Config:   cfg,
		ResolvePrompt: func(name string) (string, error) {
			return "go", nil
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := os.Stat(StateFile); !os.IsNotExist(err) {
		t.Error("expected state file to be deleted on completion")
	}
}

func TestStatePreservedOnFailStreak(t *testing.T) {
	t.Chdir(t.TempDir())

	var cmd []string
	if runtime.GOOS == "windows" {
		cmd = []string{"cmd", "/c", "exit 1"}
	} else {
		cmd = []string{"sh", "-c", "exit 1"}
	}

	wf := Workflow{
		Stages:    []Stage{{Prompt: "failing", Max: 10}},
		MaxCycles: 1,
	}
	cfg := makeTestCfg(cmd)

	_, _ = Run(Options{
		Name:     "test",
		Workflow: wf,
		Config:   cfg,
		ResolvePrompt: func(name string) (string, error) {
			return "go", nil
		},
	})

	// State file should still exist for resume
	if _, err := os.Stat(StateFile); err != nil {
		t.Error("expected state file to be preserved on failure")
	}
}

func TestStateContents(t *testing.T) {
	t.Chdir(t.TempDir())

	// Write state, read it back, verify fields
	want := &State{Workflow: "ship", Stage: 2, Cycle: 1, StartSHA: "abc123"}
	trySaveState(want)

	got, err := loadState()
	if err != nil {
		t.Fatalf("loadState error: %v", err)
	}
	if got.Workflow != want.Workflow {
		t.Errorf("workflow: got %q, want %q", got.Workflow, want.Workflow)
	}
	if got.Stage != want.Stage {
		t.Errorf("stage: got %d, want %d", got.Stage, want.Stage)
	}
	if got.Cycle != want.Cycle {
		t.Errorf("cycle: got %d, want %d", got.Cycle, want.Cycle)
	}
	if got.StartSHA != want.StartSHA {
		t.Errorf("start_sha: got %q, want %q", got.StartSHA, want.StartSHA)
	}
}

func TestStateFileIsValidJSON(t *testing.T) {
	t.Chdir(t.TempDir())

	trySaveState(&State{Workflow: "test", Stage: 0, Cycle: 0, StartSHA: "def456"})

	data, err := os.ReadFile(StateFile)
	if err != nil {
		t.Fatal(err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("state file is not valid JSON: %v", err)
	}
	if raw["start_sha"] != "def456" {
		t.Errorf("expected start_sha 'def456', got %v", raw["start_sha"])
	}
}

func TestTrySaveStateRejectsSymlink(t *testing.T) {
	t.Chdir(t.TempDir())

	target := filepath.Join(t.TempDir(), "target-state")
	if err := os.WriteFile(target, []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, StateFile); err != nil {
		t.Skip("symlinks not supported")
	}

	trySaveState(&State{Workflow: "test", Stage: 1, Cycle: 0, StartSHA: "abc"})

	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "keep" {
		t.Fatalf("state write followed symlink and modified target: %q", data)
	}
}

func TestDeleteStateIgnoresDirectory(t *testing.T) {
	t.Chdir(t.TempDir())

	if err := os.Mkdir(StateFile, 0o755); err != nil {
		t.Fatal(err)
	}

	deleteState()

	fi, err := os.Stat(StateFile)
	if err != nil {
		t.Fatalf("state directory was removed: %v", err)
	}
	if !fi.IsDir() {
		t.Fatal("expected state path to remain a directory")
	}
}

func TestResumeWithCycleState(t *testing.T) {
	t.Chdir(t.TempDir())

	counter := filepath.Join(".", "counter")
	var cmd []string
	if runtime.GOOS == "windows" {
		cmd = []string{"cmd", "/c", "echo x >> " + counter}
	} else {
		cmd = []string{"sh", "-c", "echo x >> " + counter}
	}

	wf := Workflow{
		Stages: []Stage{
			{Prompt: "build", Max: 1, Cycle: true},
			{Prompt: "verify", Max: 1},
		},
		MaxCycles: 3,
	}
	cfg := makeTestCfg(cmd)

	// Resume at stage 1 (verify), cycle 2 — should run verify then be done
	// (no tasks remain since we didn't create IMPLEMENTATION_PLAN.md)
	trySaveState(&State{Workflow: "test", Stage: 1, Cycle: 2, StartSHA: "abc"})

	_, err := Run(Options{
		Name:     "test",
		Workflow: wf,
		Config:   cfg,
		ResolvePrompt: func(name string) (string, error) {
			return "go", nil
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Only verify (1 stage) should have run
	data, err := os.ReadFile(counter)
	if err != nil {
		t.Fatal("counter not created")
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 1 {
		t.Errorf("expected 1 iteration (just verify), got %d", len(lines))
	}
}
