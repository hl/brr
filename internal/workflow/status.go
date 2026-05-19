package workflow

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func Status(name string, w io.Writer) error {
	if name != "" {
		return printStatus(name, w)
	}
	entries, err := os.ReadDir(StateDir)
	if err != nil {
		if os.IsNotExist(err) {
			_, err := fmt.Fprintln(w, "No workflow state found.")
			return err
		}
		return fmt.Errorf("reading %s: %w", StateDir, err)
	}
	found := false
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		found = true
		name := strings.TrimSuffix(entry.Name(), ".json")
		if err := printStatus(name, w); err != nil {
			return err
		}
	}
	if !found {
		if _, err := fmt.Fprintln(w, "No workflow state found."); err != nil {
			return err
		}
	}
	return nil
}

func printStatus(name string, w io.Writer) error {
	if err := validateName(name); err != nil {
		return err
	}
	state, err := (store{name: name}).load()
	if err != nil {
		if os.IsNotExist(err) {
			_, err := fmt.Fprintf(w, "No state found for workflow %q.\n", name)
			return err
		}
		return fmt.Errorf("reading workflow state: %w", err)
	}
	return writeStatus(w, state)
}

func writeStatus(w io.Writer, state *State) error {
	lines := []string{
		fmt.Sprintf("workflow: %s", state.Workflow),
		fmt.Sprintf("run_id: %s", state.RunID),
		fmt.Sprintf("next_stage: %s", state.NextStageID),
		fmt.Sprintf("cycle_count: %d", state.CycleCount),
		fmt.Sprintf("started_at: %s", state.StartedAt.Format(time.RFC3339)),
		fmt.Sprintf("updated_at: %s", state.UpdatedAt.Format(time.RFC3339)),
	}
	for _, line := range lines {
		if _, err := fmt.Fprintln(w, line); err != nil {
			return err
		}
	}
	for _, stage := range state.Stages {
		if _, err := fmt.Fprintf(w, "- %s: %s", stage.ID, stage.Status); err != nil {
			return err
		}
		if stage.Reason != "" {
			if _, err := fmt.Fprintf(w, " (%s)", stage.Reason); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintln(w); err != nil {
			return err
		}
	}
	return nil
}
