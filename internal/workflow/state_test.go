package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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

func TestInitTemplateRejectsSymlinkedWorkflowDir(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.Mkdir(".brr", 0o755); err != nil {
		t.Fatal(err)
	}
	targetDir := filepath.Join(t.TempDir(), "outside-workflows")
	if err := os.Mkdir(targetDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(targetDir, filepath.Join(".brr", "workflows")); err != nil {
		t.Skip("symlinks not supported")
	}

	err := InitTemplate("ship", "ship")
	if err == nil {
		t.Fatal("expected symlinked workflow dir to be rejected")
	}
	if !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("expected symlink error, got: %v", err)
	}
	if _, err := os.Stat(filepath.Join(targetDir, "ship.yaml")); !os.IsNotExist(err) {
		t.Fatalf("template write followed symlink, stat err: %v", err)
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
