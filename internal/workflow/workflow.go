package workflow

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"

	"github.com/hl/brr/internal/config"
	"github.com/hl/brr/internal/engine"
	"github.com/hl/brr/internal/fsutil"
	"github.com/hl/brr/internal/ui"
)

const (
	SchemaVersion = 2

	StageTypeAgent   = "agent"
	StageTypeCommand = "command"

	StateDir = ".brr/state/workflows"
)

type Defaults struct {
	Profile string `yaml:"profile,omitempty"`
	Max     int    `yaml:"max,omitempty"`
}

type Cycle struct {
	Target string `yaml:"target"`
	Max    int    `yaml:"max"`
}

type Stage struct {
	ID      string   `yaml:"id"`
	Type    string   `yaml:"type"`
	Prompt  string   `yaml:"prompt,omitempty"`
	Max     int      `yaml:"max,omitempty"`
	Profile string   `yaml:"profile,omitempty"`
	Command []string `yaml:"command,omitempty"`
}

type Workflow struct {
	Version     int      `yaml:"version"`
	Description string   `yaml:"description,omitempty"`
	Defaults    Defaults `yaml:"defaults,omitempty"`
	Cycle       *Cycle   `yaml:"cycle,omitempty"`
	Stages      []Stage  `yaml:"stages"`
}

type Options struct {
	Name          string
	Workflow      Workflow
	Config        config.Config
	ProfileFlag   string
	ResolvePrompt func(string) (string, error)
	Notify        func()
	Reset         bool
}

type State struct {
	SchemaVersion int           `json:"schema_version"`
	Workflow      string        `json:"workflow"`
	RunID         string        `json:"run_id"`
	StartedAt     time.Time     `json:"started_at"`
	UpdatedAt     time.Time     `json:"updated_at"`
	StartSHA      string        `json:"start_sha"`
	NextStageID   string        `json:"next_stage_id"`
	CycleCount    int           `json:"cycle_count"`
	Stages        []StageStatus `json:"stages"`
}

type StageStatus struct {
	ID       string        `json:"id"`
	Type     string        `json:"type"`
	Status   string        `json:"status"`
	Reason   string        `json:"reason,omitempty"`
	Duration time.Duration `json:"duration,omitempty"`
	Profile  string        `json:"profile,omitempty"`
	Prompt   string        `json:"prompt,omitempty"`
	Command  []string      `json:"command,omitempty"`
}

type Event struct {
	RunID    string    `json:"run_id"`
	Workflow string    `json:"workflow"`
	Time     time.Time `json:"time"`
	Type     string    `json:"type"`
	StageID  string    `json:"stage_id,omitempty"`
	Reason   string    `json:"reason,omitempty"`
	Message  string    `json:"message,omitempty"`
}

func Load(data []byte) (Workflow, error) {
	var raw map[string]interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return Workflow{}, fmt.Errorf("parsing workflow: %w", err)
	}
	if _, ok := raw["version"]; !ok {
		return Workflow{}, fmt.Errorf("workflow schema version is required; legacy unversioned workflows are no longer supported (add version: 2 and use stages with id/type plus cycle.target)")
	}

	var wf Workflow
	dec := yaml.NewDecoder(strings.NewReader(string(data)))
	dec.KnownFields(true)
	if err := dec.Decode(&wf); err != nil {
		return wf, fmt.Errorf("parsing workflow: %w", err)
	}
	return wf, validate(wf)
}

func Resolve(name string) ([]byte, error) {
	if err := validateName(name); err != nil {
		return nil, err
	}

	projectPath := filepath.Join(".brr", "workflows", name+".yaml")
	if data, err := fsutil.ReadRegularFile(projectPath); err == nil {
		return data, nil
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("reading %s: %w", projectPath, err)
	}

	configHint := "<config-dir>/brr/workflows/" + name + ".yaml"
	if configDir, err := os.UserConfigDir(); err == nil {
		userPath := filepath.Join(configDir, "brr", "workflows", name+".yaml")
		configHint = userPath
		if data, err := fsutil.ReadRegularFile(userPath); err == nil {
			return data, nil
		} else if !os.IsNotExist(err) {
			return nil, fmt.Errorf("reading %s: %w", userPath, err)
		}
	}

	return nil, fmt.Errorf("workflow %q not found (looked in %s and %s)", name, projectPath, configHint)
}

func ValidateRuntime(wf Workflow, cfg config.Config, profileFlag string, resolvePrompt func(string) (string, error)) error {
	if _, _, err := resolveWorkflowProfile(wf.Defaults.Profile, profileFlag, cfg); err != nil {
		return fmt.Errorf("defaults.profile: %w", err)
	}
	for _, stage := range wf.Stages {
		switch stage.Type {
		case StageTypeAgent:
			if _, _, err := resolveWorkflowProfile(stage.Profile, profileFlag, cfg); err != nil {
				return fmt.Errorf("stages.%s.profile: %w", stage.ID, err)
			}
			if resolvePrompt != nil {
				if _, err := resolvePrompt(stage.Prompt); err != nil {
					return fmt.Errorf("stages.%s.prompt: %w", stage.ID, err)
				}
			}
		case StageTypeCommand:
			if _, err := exec.LookPath(stage.Command[0]); err != nil {
				return fmt.Errorf("stages.%s.command: %w", stage.ID, err)
			}
		}
	}
	return nil
}

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

func InitTemplate(name, template string) error {
	if err := validateName(name); err != nil {
		return err
	}
	if template != "ship" {
		return fmt.Errorf("unknown workflow template %q (available: ship)", template)
	}
	target := filepath.Join(".brr", "workflows", name+".yaml")
	if err := rejectUnsafeExisting(target); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return fmt.Errorf("creating .brr/workflows: %w", err)
	}
	if _, err := os.Lstat(target); err == nil {
		return fmt.Errorf("%s already exists", target)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("checking %s: %w", target, err)
	}
	return atomicWriteRegularFile(target, []byte(ShipTemplate), 0o644)
}

// Run executes the workflow. The caller must hold the exclusive lock.
func Run(opts Options) (*engine.Result, error) {
	if err := validateName(opts.Name); err != nil {
		return nil, err
	}
	if err := ValidateRuntime(opts.Workflow, opts.Config, opts.ProfileFlag, nil); err != nil {
		return nil, err
	}

	store := store{name: opts.Name}
	if opts.Reset {
		store.delete()
	}

	stageIdx := 0
	now := time.Now().UTC()
	state := &State{
		SchemaVersion: SchemaVersion,
		Workflow:      opts.Name,
		RunID:         newRunID(),
		StartedAt:     now,
		UpdatedAt:     now,
		StartSHA:      gitHEAD(),
		NextStageID:   opts.Workflow.Stages[0].ID,
		CycleCount:    0,
		Stages:        initialStageStatus(opts.Workflow),
	}

	if !opts.Reset {
		if saved, err := store.load(); err == nil && validResumeState(saved, opts.Workflow) {
			state = saved
			stageIdx = stageIndexByID(opts.Workflow, saved.NextStageID)
			fmt.Fprintf(os.Stderr, "  %sresuming:%s stage %s", ui.Dim, ui.Reset, saved.NextStageID)
			if saved.CycleCount > 0 {
				fmt.Fprintf(os.Stderr, " (cycle %d)", saved.CycleCount)
			}
			fmt.Fprintln(os.Stderr)
		}
	}

	printWorkflowSummary(opts.Workflow, opts.Name)
	store.appendEvent(Event{RunID: state.RunID, Workflow: opts.Name, Time: time.Now().UTC(), Type: "workflow_started"})
	store.save(state)

	for stageIdx < len(opts.Workflow.Stages) {
		stage := opts.Workflow.Stages[stageIdx]
		result, err := runStage(opts, state, store, stage, stageIdx)
		if err != nil {
			if result != nil && result.Reason == engine.ReasonInterrupted {
				return result, err
			}
			store.appendEvent(Event{RunID: state.RunID, Workflow: opts.Name, Time: time.Now().UTC(), Type: "workflow_error", StageID: stage.ID, Reason: stopReason(result), Message: err.Error()})
			return result, err
		}

		switch result.Reason {
		case engine.ReasonFailed:
			fmt.Fprintf(os.Stderr, "\n  %s%sWorkflow stopped — stage %q failed%s\n", ui.Bold, ui.Red, stage.ID, ui.Reset)
			fmt.Fprintf(os.Stderr, "  %sDelete .brr-failed and re-run to resume.%s\n", ui.Dim, ui.Reset)
			return result, nil
		case engine.ReasonApproval:
			fmt.Fprintf(os.Stderr, "\n  %s%sWorkflow paused — stage %q needs approval%s\n", ui.Bold, ui.Yellow, stage.ID, ui.Reset)
			fmt.Fprintf(os.Stderr, "  %sDelete .brr-needs-approval and re-run to resume.%s\n", ui.Dim, ui.Reset)
			return result, nil
		case engine.ReasonCycle:
			nextIdx, err := nextCycleIndex(opts.Workflow, state.CycleCount)
			if err != nil {
				store.appendEvent(Event{RunID: state.RunID, Workflow: opts.Name, Time: time.Now().UTC(), Type: "workflow_error", StageID: stage.ID, Reason: "cycle", Message: err.Error()})
				return result, fmt.Errorf("stage %s: %w", stage.ID, err)
			}
			state.CycleCount++
			stageIdx = nextIdx
			state.NextStageID = opts.Workflow.Stages[stageIdx].ID
			state.UpdatedAt = time.Now().UTC()
			store.save(state)
			store.appendEvent(Event{RunID: state.RunID, Workflow: opts.Name, Time: state.UpdatedAt, Type: "cycle", StageID: stage.ID, Message: "restarting from " + state.NextStageID})
			fmt.Fprintf(os.Stderr, "\n%s━━━%s %s%sCycle %d/%d%s %s▸ restarting from %s ━━━%s\n",
				ui.Dim, ui.Reset,
				ui.Bold, ui.Magenta, state.CycleCount, opts.Workflow.Cycle.Max, ui.Reset,
				ui.Dim, state.NextStageID, ui.Reset,
			)
			continue
		}

		stageIdx++
		if stageIdx < len(opts.Workflow.Stages) {
			state.NextStageID = opts.Workflow.Stages[stageIdx].ID
		} else {
			state.NextStageID = ""
		}
		state.UpdatedAt = time.Now().UTC()
		store.save(state)
	}

	store.delete()
	store.appendEvent(Event{RunID: state.RunID, Workflow: opts.Name, Time: time.Now().UTC(), Type: "workflow_complete"})
	fmt.Fprintf(os.Stderr, "\n  %s%sWorkflow complete%s\n", ui.Bold, ui.Green, ui.Reset)
	if opts.Notify != nil {
		opts.Notify()
	}
	return &engine.Result{Reason: engine.ReasonComplete}, nil
}

func validate(wf Workflow) error {
	if wf.Version != SchemaVersion {
		return fmt.Errorf("workflow schema version must be %d, got %d", SchemaVersion, wf.Version)
	}
	if len(wf.Stages) == 0 {
		return fmt.Errorf("stages: at least one stage is required")
	}
	if wf.Defaults.Max < 0 {
		return fmt.Errorf("defaults.max must be >= 0, got %d", wf.Defaults.Max)
	}
	seen := make(map[string]bool)
	for i, stage := range wf.Stages {
		path := fmt.Sprintf("stages[%d]", i)
		if strings.TrimSpace(stage.ID) == "" {
			return fmt.Errorf("%s.id is required", path)
		}
		if strings.ContainsAny(stage.ID, `/\`) || strings.Contains(stage.ID, "..") {
			return fmt.Errorf("%s.id %q is invalid", path, stage.ID)
		}
		if seen[stage.ID] {
			return fmt.Errorf("%s.id %q is duplicated", path, stage.ID)
		}
		seen[stage.ID] = true
		if stage.Type != StageTypeAgent && stage.Type != StageTypeCommand {
			return fmt.Errorf("%s.type must be %q or %q", path, StageTypeAgent, StageTypeCommand)
		}
		switch stage.Type {
		case StageTypeAgent:
			if strings.TrimSpace(stage.Prompt) == "" {
				return fmt.Errorf("%s.prompt is required for agent stages", path)
			}
			if len(stage.Command) > 0 {
				return fmt.Errorf("%s.command is only valid for command stages", path)
			}
			if effectiveMax(wf, stage) < 1 {
				return fmt.Errorf("%s.max must be >= 1 for agent stages", path)
			}
		case StageTypeCommand:
			if len(stage.Command) == 0 || strings.TrimSpace(stage.Command[0]) == "" {
				return fmt.Errorf("%s.command must be a non-empty argv array", path)
			}
			if strings.TrimSpace(stage.Prompt) != "" {
				return fmt.Errorf("%s.prompt is only valid for agent stages", path)
			}
			if stage.Max != 0 {
				return fmt.Errorf("%s.max is only valid for agent stages", path)
			}
			if strings.TrimSpace(stage.Profile) != "" {
				return fmt.Errorf("%s.profile is only valid for agent stages", path)
			}
		}
	}
	if wf.Cycle != nil {
		if strings.TrimSpace(wf.Cycle.Target) == "" {
			return fmt.Errorf("cycle.target is required")
		}
		if !seen[wf.Cycle.Target] {
			return fmt.Errorf("cycle.target %q does not match any stage id", wf.Cycle.Target)
		}
		if wf.Cycle.Max < 1 {
			return fmt.Errorf("cycle.max must be >= 1, got %d", wf.Cycle.Max)
		}
	}
	return nil
}

func runStage(opts Options, state *State, store store, stage Stage, idx int) (*engine.Result, error) {
	printStageHeader(idx+1, len(opts.Workflow.Stages), stage, state.CycleCount, opts.Workflow)
	start := time.Now()
	updateStageStatus(state, stage, "running", "", 0)
	state.NextStageID = stage.ID
	state.UpdatedAt = time.Now().UTC()
	store.save(state)
	store.appendEvent(Event{RunID: state.RunID, Workflow: opts.Name, Time: state.UpdatedAt, Type: "stage_started", StageID: stage.ID})

	var result *engine.Result
	var err error
	switch stage.Type {
	case StageTypeAgent:
		result, err = runAgentStage(opts, stage)
	case StageTypeCommand:
		result, err = runCommandStage(stage)
	}

	duration := time.Since(start)
	reason := stopReason(result)
	status := "completed"
	if err != nil {
		status = "error"
	} else if result != nil {
		switch result.Reason {
		case engine.ReasonFailed:
			status = "failed"
		case engine.ReasonApproval:
			status = "approval"
		case engine.ReasonCycle:
			status = "cycle"
		}
	}
	updateStageStatus(state, stage, status, reason, duration)
	state.UpdatedAt = time.Now().UTC()
	store.save(state)
	store.appendEvent(Event{RunID: state.RunID, Workflow: opts.Name, Time: state.UpdatedAt, Type: "stage_finished", StageID: stage.ID, Reason: reason})

	if err != nil {
		return result, fmt.Errorf("stage %s: %w", stage.ID, err)
	}
	return result, nil
}

func runAgentStage(opts Options, stage Stage) (*engine.Result, error) {
	promptText, err := opts.ResolvePrompt(stage.Prompt)
	if err != nil {
		return nil, err
	}
	command, _, err := resolveWorkflowProfile(stage.Profile, opts.ProfileFlag, opts.Config)
	if err != nil {
		return nil, err
	}
	return engine.Run(engine.Options{
		Prompt:   promptText,
		Max:      effectiveMax(opts.Workflow, stage),
		Command:  command,
		SkipLock: true,
	})
}

func runCommandStage(stage Stage) (*engine.Result, error) {
	cmd := exec.Command(stage.Command[0], stage.Command[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	err := cmd.Run()
	if sig := detectSignalFiles(); sig != nil {
		cleanupSignalFiles()
		return sig, nil
	}
	if err != nil {
		return &engine.Result{Reason: engine.ReasonFailStreak}, err
	}
	return &engine.Result{Reason: engine.ReasonComplete}, nil
}

func detectSignalFiles() *engine.Result {
	if fsutil.IsRegularFile(engine.SignalComplete) {
		fmt.Fprintf(os.Stderr, "\n  %s%s✓ All tasks complete%s (%s found). Stopping.\n", ui.Bold, ui.Green, ui.Reset, engine.SignalComplete)
		return &engine.Result{Reason: engine.ReasonComplete}
	}
	if content, ok := readSignalContent(engine.SignalFailed); ok {
		fmt.Fprintf(os.Stderr, "\n  %s%s✗ Agent failed%s (%s found):\n", ui.Bold, ui.Red, ui.Reset, engine.SignalFailed)
		if content != "" {
			fmt.Fprintln(os.Stderr, content)
		} else {
			fmt.Fprintln(os.Stderr, "  (no details provided)")
		}
		return &engine.Result{Reason: engine.ReasonFailed, FailedContent: content}
	}
	if content, ok := readSignalContent(engine.SignalNeedsApproval); ok {
		fmt.Fprintf(os.Stderr, "\n  %s%s⏸ Task needs human approval%s (%s found):\n", ui.Bold, ui.Yellow, ui.Reset, engine.SignalNeedsApproval)
		if content != "" {
			fmt.Fprintln(os.Stderr, content)
		} else {
			fmt.Fprintln(os.Stderr, "  (no details provided)")
		}
		return &engine.Result{Reason: engine.ReasonApproval, ApprovalContent: content}
	}
	if fsutil.IsRegularFile(engine.SignalCycle) {
		fmt.Fprintf(os.Stderr, "\n  %s%s↻ Cycle requested%s (%s found). Stopping this stage.\n", ui.Bold, ui.Magenta, ui.Reset, engine.SignalCycle)
		return &engine.Result{Reason: engine.ReasonCycle}
	}
	return nil
}

func readSignalContent(path string) (string, bool) {
	f, err := fsutil.OpenRegularFile(path)
	if err != nil {
		return "", fsutil.IsRegularFile(path)
	}
	defer func() { _ = f.Close() }()
	data, err := io.ReadAll(io.LimitReader(f, 4097))
	if err != nil {
		return "", true
	}
	if len(data) > 4096 {
		data = data[:4096]
	}
	return strings.TrimSpace(string(data)), true
}

func cleanupSignalFiles() {
	for _, path := range []string{engine.SignalComplete, engine.SignalFailed, engine.SignalNeedsApproval, engine.SignalCycle} {
		if fsutil.IsRegularFile(path) {
			_ = os.Remove(path)
		}
	}
}

func nextCycleIndex(wf Workflow, currentCycles int) (int, error) {
	if wf.Cycle == nil {
		return -1, errors.New("requested a workflow cycle, but no cycle target is configured")
	}
	if currentCycles >= wf.Cycle.Max {
		return -1, fmt.Errorf("requested another workflow cycle after cycle.max %d", wf.Cycle.Max)
	}
	return stageIndexByID(wf, wf.Cycle.Target), nil
}

func initialStageStatus(wf Workflow) []StageStatus {
	statuses := make([]StageStatus, 0, len(wf.Stages))
	for _, stage := range wf.Stages {
		statuses = append(statuses, stageStatus(stage, "pending", "", 0))
	}
	return statuses
}

func updateStageStatus(state *State, stage Stage, status, reason string, duration time.Duration) {
	for i := range state.Stages {
		if state.Stages[i].ID == stage.ID {
			state.Stages[i] = stageStatus(stage, status, reason, duration)
			return
		}
	}
	state.Stages = append(state.Stages, stageStatus(stage, status, reason, duration))
}

func stageStatus(stage Stage, status, reason string, duration time.Duration) StageStatus {
	return StageStatus{
		ID:       stage.ID,
		Type:     stage.Type,
		Status:   status,
		Reason:   reason,
		Duration: duration,
		Profile:  stage.Profile,
		Prompt:   stage.Prompt,
		Command:  append([]string(nil), stage.Command...),
	}
}

func validResumeState(s *State, wf Workflow) bool {
	return s.SchemaVersion == SchemaVersion &&
		s.Workflow != "" &&
		s.RunID != "" &&
		s.CycleCount >= 0 &&
		(wf.Cycle != nil || s.CycleCount == 0) &&
		(wf.Cycle == nil || s.CycleCount <= wf.Cycle.Max) &&
		s.NextStageID != "" &&
		stageIndexByID(wf, s.NextStageID) >= 0
}

func stageIndexByID(wf Workflow, id string) int {
	for i, stage := range wf.Stages {
		if stage.ID == id {
			return i
		}
	}
	return -1
}

func effectiveMax(wf Workflow, stage Stage) int {
	if stage.Max > 0 {
		return stage.Max
	}
	return wf.Defaults.Max
}

func resolveWorkflowProfile(stageProfile, flagProfile string, cfg config.Config) ([]string, string, error) {
	name := stageProfile
	if name == "" {
		name = flagProfile
	}
	return cfg.ResolveProfile(name)
}

type store struct {
	name string
}

func (s store) statePath() string {
	return filepath.Join(StateDir, s.name+".json")
}

func (s store) eventsPath() string {
	return filepath.Join(StateDir, s.name+".events.jsonl")
}

func (s store) load() (*State, error) {
	data, err := fsutil.ReadRegularFile(s.statePath())
	if err != nil {
		return nil, err
	}
	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

func (s store) save(state *State) {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not save workflow state: %v\n", err)
		return
	}
	if err := ensureStateDir(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not create workflow state dir: %v\n", err)
		return
	}
	if err := atomicWriteRegularFile(s.statePath(), append(data, '\n'), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not save workflow state: %v\n", err)
	}
}

func (s store) delete() {
	removeStatePath(s.statePath())
}

func (s store) appendEvent(event Event) {
	if err := ensureStateDir(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not create workflow event dir: %v\n", err)
		return
	}
	data, err := json.Marshal(event)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not encode workflow event: %v\n", err)
		return
	}
	if err := appendRegularFile(s.eventsPath(), append(data, '\n'), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not write workflow event: %v\n", err)
	}
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
	if _, err := fmt.Fprintf(w, "workflow: %s\n", state.Workflow); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "run_id: %s\n", state.RunID); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "next_stage: %s\n", state.NextStageID); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "cycle_count: %d\n", state.CycleCount); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "started_at: %s\n", state.StartedAt.Format(time.RFC3339)); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "updated_at: %s\n", state.UpdatedAt.Format(time.RFC3339)); err != nil {
		return err
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

func removeStatePath(path string) {
	fi, err := os.Lstat(path)
	if err != nil {
		return
	}
	if fi.Mode().IsRegular() || fi.Mode()&os.ModeSymlink != 0 {
		_ = os.Remove(path)
	}
}

func atomicWriteRegularFile(path string, data []byte, perm os.FileMode) error {
	if err := rejectNonRegularPath(path); err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating directory %s: %w", dir, err)
	}
	tmp, err := os.CreateTemp(dir, ".brr-state-*")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(perm); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := rejectNonRegularPath(path); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

func appendRegularFile(path string, data []byte, perm os.FileMode) error {
	fi, err := os.Lstat(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if os.IsNotExist(err) {
		f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, perm)
		if err != nil {
			return err
		}
		defer func() { _ = f.Close() }()
		_, err = f.Write(data)
		return err
	}
	if !fi.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", path)
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, perm)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	fi2, err := f.Stat()
	if err != nil {
		return err
	}
	if !os.SameFile(fi, fi2) {
		return fmt.Errorf("%s: file changed between stat and open", path)
	}
	_, err = f.Write(data)
	return err
}

func ensureStateDir() error {
	for _, path := range []string{".brr", filepath.Join(".brr", "state"), StateDir} {
		fi, err := os.Lstat(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return fmt.Errorf("checking %s: %w", path, err)
		}
		if fi.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("%s is a symlink", path)
		}
		if !fi.IsDir() {
			return fmt.Errorf("%s is not a directory", path)
		}
	}
	return os.MkdirAll(StateDir, 0o755)
}

func rejectUnsafeExisting(path string) error {
	fi, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("checking %s: %w", path, err)
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("%s is a symlink", path)
	}
	return nil
}

func rejectNonRegularPath(path string) error {
	fi, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("checking %s: %w", path, err)
	}
	if !fi.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", path)
	}
	return nil
}

func validateName(name string) error {
	if strings.TrimSpace(name) == "" || strings.Contains(name, "..") || strings.ContainsAny(name, `/\`) {
		return fmt.Errorf("invalid workflow name: %q", name)
	}
	return nil
}

func gitHEAD() string {
	out, err := exec.Command("git", "rev-parse", "HEAD").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func newRunID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b[:])
}

func stopReason(result *engine.Result) string {
	if result == nil {
		return ""
	}
	switch result.Reason {
	case engine.ReasonComplete:
		return "complete"
	case engine.ReasonFailed:
		return "failed"
	case engine.ReasonApproval:
		return "approval"
	case engine.ReasonCycle:
		return "cycle"
	case engine.ReasonMaxIterations:
		return "max_iterations"
	case engine.ReasonFailStreak:
		return "fail_streak"
	case engine.ReasonInterrupted:
		return "interrupted"
	default:
		return "unknown"
	}
}

func printWorkflowSummary(wf Workflow, name string) {
	fmt.Fprintf(os.Stderr, "  %sworkflow:%s %s\n", ui.Dim, ui.Reset, name)
	if wf.Description != "" {
		fmt.Fprintf(os.Stderr, "  %sdescription:%s %s\n", ui.Dim, ui.Reset, wf.Description)
	}
	fmt.Fprintf(os.Stderr, "  %sstages:%s  %d\n", ui.Dim, ui.Reset, len(wf.Stages))
	if wf.Cycle != nil {
		fmt.Fprintf(os.Stderr, "  %scycle:%s   %s (max %d)\n", ui.Dim, ui.Reset, wf.Cycle.Target, wf.Cycle.Max)
	}
	fmt.Fprintln(os.Stderr)
	for i, stage := range wf.Stages {
		marker := " "
		if wf.Cycle != nil && wf.Cycle.Target == stage.ID {
			marker = "↻"
		}
		fmt.Fprintf(os.Stderr, "  %s %s%d.%s %s %s(%s)%s\n",
			marker, ui.Bold, i+1, ui.Reset, stage.ID, ui.Dim, stage.Type, ui.Reset,
		)
	}
	fmt.Fprintln(os.Stderr)
}

func printStageHeader(num, total int, stage Stage, cycle int, wf Workflow) {
	cycleLabel := ""
	if cycle > 0 {
		cycleLabel = fmt.Sprintf(" %s[cycle %d]%s", ui.Magenta, cycle, ui.Reset)
	}
	detail := strings.Join(stage.Command, " ")
	if stage.Type == StageTypeAgent {
		detail = fmt.Sprintf("%s (max %d)", stage.Prompt, effectiveMax(wf, stage))
	}
	fmt.Fprintf(os.Stderr, "\n%s━━━%s %s%sStage %d/%d — %s%s %s▸ %s%s ━━━%s\n",
		ui.Dim, ui.Reset,
		ui.Bold, ui.Cyan, num, total, stage.ID, ui.Reset,
		ui.Dim, detail, cycleLabel, ui.Reset,
	)
}

const ShipTemplate = `version: 2
description: "requirements -> verified, reviewed code"

defaults:
  profile: claude
  max: 3

cycle:
  target: build
  max: 3

stages:
  - id: spec
    type: agent
    prompt: spec
    max: 3

  - id: plan
    type: agent
    prompt: plan
    max: 5

  - id: build
    type: agent
    prompt: build
    max: 100

  - id: check
    type: command
    command: ["make", "check"]

  - id: verify
    type: agent
    prompt: verify
    max: 3

  - id: review
    type: agent
    prompt: review
    max: 1
`
