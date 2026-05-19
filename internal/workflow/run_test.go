package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hl/brr/internal/engine"
)

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
