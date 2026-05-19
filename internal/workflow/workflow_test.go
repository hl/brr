package workflow

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/hl/brr/internal/config"
	"github.com/hl/brr/internal/engine"
)

func TestLoadValidWorkflowV2(t *testing.T) {
	wf, err := Load([]byte(`
version: 2
description: ship it
defaults:
  profile: claude
  max: 3
cycle:
  target: build
  max: 2
stages:
  - id: spec
    type: agent
    prompt: spec
  - id: build
    type: agent
    prompt: build
    max: 10
  - id: check
    type: command
    command: ["true"]
`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if wf.Version != 2 {
		t.Fatalf("expected version 2, got %d", wf.Version)
	}
	if wf.Cycle == nil || wf.Cycle.Target != "build" || wf.Cycle.Max != 2 {
		t.Fatalf("unexpected cycle config: %#v", wf.Cycle)
	}
	if got := effectiveMax(wf, wf.Stages[0]); got != 3 {
		t.Fatalf("expected default max 3, got %d", got)
	}
}

func TestLoadRejectsLegacyWorkflow(t *testing.T) {
	_, err := Load([]byte(`
stages:
  - prompt: build
    max: 10
    cycle: true
max_cycles: 3
`))
	if err == nil {
		t.Fatal("expected legacy schema error")
	}
	if !strings.Contains(err.Error(), "legacy unversioned workflows are no longer supported") {
		t.Fatalf("expected migration-style error, got: %v", err)
	}
}

func TestLoadValidationErrors(t *testing.T) {
	tests := []struct {
		name string
		yaml string
		want string
	}{
		{
			name: "duplicate id",
			yaml: `
version: 2
defaults: {max: 1}
stages:
  - id: build
    type: agent
    prompt: build
  - id: build
    type: command
    command: ["true"]
`,
			want: "duplicated",
		},
		{
			name: "unknown type",
			yaml: `
version: 2
stages:
  - id: build
    type: magic
`,
			want: "type must be",
		},
		{
			name: "missing prompt",
			yaml: `
version: 2
defaults: {max: 1}
stages:
  - id: build
    type: agent
`,
			want: "prompt is required",
		},
		{
			name: "command is shell string",
			yaml: `
version: 2
stages:
  - id: check
    type: command
    command: []
`,
			want: "argv array",
		},
		{
			name: "bad cycle target",
			yaml: `
version: 2
defaults: {max: 1}
cycle:
  target: missing
  max: 1
stages:
  - id: build
    type: agent
    prompt: build
`,
			want: "cycle.target",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Load([]byte(tt.yaml))
			if err == nil {
				t.Fatal("expected validation error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q in error, got: %v", tt.want, err)
			}
		})
	}
}

func TestResolveProjectWorkflowSymlinkRejected(t *testing.T) {
	t.Chdir(t.TempDir())
	dir := filepath.Join(".brr", "workflows")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(t.TempDir(), "example.yaml")
	if err := os.WriteFile(target, []byte("version: 2\nstages: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(dir, "example.yaml")); err != nil {
		t.Skip("symlinks not supported")
	}
	_, err := Resolve("example")
	if err == nil {
		t.Fatal("expected symlinked workflow to be rejected")
	}
}

func TestValidateRuntimeProfileAndPrompt(t *testing.T) {
	wf := testWorkflow([]Stage{{ID: "build", Type: StageTypeAgent, Prompt: "build", Profile: "missing", Max: 1}}, nil)
	err := ValidateRuntime(wf, testConfig(echoCmd()), "", func(string) (string, error) {
		return "go", nil
	})
	if err == nil {
		t.Fatal("expected missing profile error")
	}
	if !strings.Contains(err.Error(), "profile") {
		t.Fatalf("expected profile error, got: %v", err)
	}
}

func TestRunSequentialAgentAndCommandStages(t *testing.T) {
	t.Chdir(t.TempDir())
	logFile := filepath.Join(".", "stage-log")
	agentCmd := catToFileCmd(logFile)
	commandCmd := appendTextCmd(logFile, "check\n")

	wf := testWorkflow([]Stage{
		{ID: "first", Type: StageTypeAgent, Prompt: "first", Max: 1},
		{ID: "check", Type: StageTypeCommand, Command: commandCmd},
		{ID: "second", Type: StageTypeAgent, Prompt: "second", Max: 1},
	}, nil)

	result, err := Run(Options{
		Name:     "ship",
		Workflow: wf,
		Config:   testConfig(agentCmd),
		ResolvePrompt: func(name string) (string, error) {
			return name + "\n", nil
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil || result.Reason != engine.ReasonComplete {
		t.Fatalf("expected complete result, got %#v", result)
	}
	data, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	i1 := strings.Index(content, "first")
	i2 := strings.Index(content, "check")
	i3 := strings.Index(content, "second")
	if i1 < 0 || i2 < 0 || i3 < 0 || i1 >= i2 || i2 >= i3 {
		t.Fatalf("stages did not run in order: %q", content)
	}
	if _, err := os.Stat(filepath.Join(StateDir, "ship.json")); !os.IsNotExist(err) {
		t.Fatalf("expected state file to be deleted on completion, got: %v", err)
	}
	if _, err := os.Stat(filepath.Join(StateDir, "ship.events.jsonl")); err != nil {
		t.Fatalf("expected event log to remain: %v", err)
	}
}

func TestRunCommandStageFailurePreservesState(t *testing.T) {
	t.Chdir(t.TempDir())
	wf := testWorkflow([]Stage{{ID: "check", Type: StageTypeCommand, Command: failCmd()}}, nil)

	result, err := Run(Options{
		Name:     "ship",
		Workflow: wf,
		Config:   testConfig(echoCmd()),
		ResolvePrompt: func(name string) (string, error) {
			return name, nil
		},
	})
	if err == nil {
		t.Fatal("expected command stage failure")
	}
	if result == nil || result.Reason != engine.ReasonFailStreak {
		t.Fatalf("expected fail streak result, got %#v", result)
	}
	state := readState(t, "ship")
	if state.NextStageID != "check" {
		t.Fatalf("expected resume at failed stage, got %q", state.NextStageID)
	}
	if state.Stages[0].Status != "error" {
		t.Fatalf("expected stage status error, got %q", state.Stages[0].Status)
	}
}

func TestRunCycleFromCommandStage(t *testing.T) {
	t.Chdir(t.TempDir())
	counter := filepath.Join(".", "counter")
	wf := testWorkflow([]Stage{
		{ID: "build", Type: StageTypeCommand, Command: appendTextCmd(counter, "x\n")},
		{ID: "review", Type: StageTypeCommand, Command: cycleOnceCmd(counter)},
	}, &Cycle{Target: "build", Max: 2})

	_, err := Run(Options{
		Name:     "ship",
		Workflow: wf,
		Config:   testConfig(echoCmd()),
		ResolvePrompt: func(name string) (string, error) {
			return name, nil
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, err := os.ReadFile(counter)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected build to run twice, got %d (%q)", len(lines), data)
	}
}

func TestRunCycleLimit(t *testing.T) {
	t.Chdir(t.TempDir())
	wf := testWorkflow([]Stage{{ID: "build", Type: StageTypeCommand, Command: alwaysCycleCmd()}}, &Cycle{Target: "build", Max: 1})

	_, err := Run(Options{
		Name:     "ship",
		Workflow: wf,
		Config:   testConfig(echoCmd()),
		ResolvePrompt: func(name string) (string, error) {
			return name, nil
		},
	})
	if err == nil {
		t.Fatal("expected cycle max error")
	}
	if !strings.Contains(err.Error(), "cycle.max 1") {
		t.Fatalf("expected cycle max in error, got: %v", err)
	}
	state := readState(t, "ship")
	if state.CycleCount != 1 {
		t.Fatalf("expected cycle count 1, got %d", state.CycleCount)
	}
}

func TestResumeUsesPerWorkflowState(t *testing.T) {
	t.Chdir(t.TempDir())
	logFile := filepath.Join(".", "stage-log")
	cmd := catToFileCmd(logFile)
	wf := testWorkflow([]Stage{
		{ID: "first", Type: StageTypeAgent, Prompt: "first", Max: 1},
		{ID: "second", Type: StageTypeAgent, Prompt: "second", Max: 1},
	}, nil)
	state := &State{
		SchemaVersion: SchemaVersion,
		Workflow:      "ship",
		RunID:         "abc",
		StartedAt:     testTime(),
		UpdatedAt:     testTime(),
		NextStageID:   "second",
		Stages:        initialStageStatus(wf),
	}
	(store{name: "ship"}).save(state)

	_, err := Run(Options{
		Name:     "ship",
		Workflow: wf,
		Config:   testConfig(cmd),
		ResolvePrompt: func(name string) (string, error) {
			return name + "\n", nil
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "first") || !strings.Contains(string(data), "second") {
		t.Fatalf("expected only second stage to run, got %q", data)
	}
}

func TestStatusPrintsSavedState(t *testing.T) {
	t.Chdir(t.TempDir())
	wf := testWorkflow([]Stage{{ID: "build", Type: StageTypeCommand, Command: echoCmd()}}, nil)
	(store{name: "ship"}).save(&State{
		SchemaVersion: SchemaVersion,
		Workflow:      "ship",
		RunID:         "abc",
		StartedAt:     testTime(),
		UpdatedAt:     testTime(),
		NextStageID:   "build",
		Stages:        initialStageStatus(wf),
	})
	var out strings.Builder
	if err := Status("ship", &out); err != nil {
		t.Fatalf("status error: %v", err)
	}
	if !strings.Contains(out.String(), "workflow: ship") || !strings.Contains(out.String(), "next_stage: build") {
		t.Fatalf("unexpected status output: %s", out.String())
	}
}

func TestInitTemplateWritesShipWorkflow(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := InitTemplate("ship", "ship"); err != nil {
		t.Fatalf("init template: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(".brr", "workflows", "ship.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	wf, err := Load(data)
	if err != nil {
		t.Fatalf("template should load: %v", err)
	}
	if len(wf.Stages) == 0 || wf.Cycle == nil || wf.Cycle.Target != "build" {
		t.Fatalf("unexpected template workflow: %#v", wf)
	}
}

func TestStateWriteRejectsSymlink(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.MkdirAll(StateDir, 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(t.TempDir(), "target-state")
	if err := os.WriteFile(target, []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(StateDir, "ship.json")); err != nil {
		t.Skip("symlinks not supported")
	}
	state := &State{SchemaVersion: SchemaVersion, Workflow: "ship", RunID: "abc", NextStageID: "build"}
	(store{name: "ship"}).save(state)
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "keep" {
		t.Fatalf("state write followed symlink and modified target: %q", data)
	}
}

func TestEventWriteRejectsSymlink(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.MkdirAll(StateDir, 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(t.TempDir(), "target-events")
	if err := os.WriteFile(target, []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(StateDir, "ship.events.jsonl")); err != nil {
		t.Skip("symlinks not supported")
	}
	(store{name: "ship"}).appendEvent(Event{RunID: "abc", Workflow: "ship", Type: "test", Time: testTime()})
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "keep" {
		t.Fatalf("event write followed symlink and modified target: %q", data)
	}
}

func readState(t *testing.T, name string) *State {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(StateDir, name+".json"))
	if err != nil {
		t.Fatal(err)
	}
	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		t.Fatal(err)
	}
	return &state
}

func testWorkflow(stages []Stage, cycle *Cycle) Workflow {
	return Workflow{
		Version:  SchemaVersion,
		Defaults: Defaults{Max: 1},
		Cycle:    cycle,
		Stages:   stages,
	}
}

func testConfig(cmd []string) config.Config {
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

func failCmd() []string {
	if runtime.GOOS == "windows" {
		return []string{"cmd", "/c", "exit 1"}
	}
	return []string{"false"}
}

func catToFileCmd(path string) []string {
	if runtime.GOOS == "windows" {
		return []string{"cmd", "/c", "set /p x= & echo %x% >> " + path}
	}
	return []string{"sh", "-c", "cat >> " + path}
}

func appendTextCmd(path, text string) []string {
	if runtime.GOOS == "windows" {
		return []string{"cmd", "/c", "echo " + strings.TrimSpace(text) + " >> " + path}
	}
	return []string{"sh", "-c", "printf " + shellQuote(text) + " >> " + shellQuote(path)}
}

func cycleOnceCmd(counter string) []string {
	if runtime.GOOS == "windows" {
		return []string{"cmd", "/c", "findstr /c:\"x\" " + counter + " >nul && findstr /c:\"x\" " + counter + " | find /c /v \"\" > count.tmp && set /p n=<count.tmp && if %n% LSS 2 echo again > .brr-cycle"}
	}
	return []string{"sh", "-c", `lines=$(wc -l < ` + shellQuote(counter) + ` | tr -d ' '); if [ "$lines" -lt 2 ]; then touch .brr-cycle; fi`}
}

func alwaysCycleCmd() []string {
	if runtime.GOOS == "windows" {
		return []string{"cmd", "/c", "echo again > .brr-cycle"}
	}
	return []string{"sh", "-c", "touch .brr-cycle"}
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

func testTime() time.Time {
	return time.Unix(1700000000, 0).UTC()
}
