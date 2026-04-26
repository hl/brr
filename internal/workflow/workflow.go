package workflow

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/hl/brr/internal/config"
	"github.com/hl/brr/internal/engine"
	"github.com/hl/brr/internal/fsutil"
	"github.com/hl/brr/internal/ui"
)

type Stage struct {
	Prompt  string `yaml:"prompt"`
	Max     int    `yaml:"max"`
	Profile string `yaml:"profile,omitempty"`
	Cycle   bool   `yaml:"cycle,omitempty"`
}

type Workflow struct {
	Stages    []Stage `yaml:"stages"`
	MaxCycles int     `yaml:"max_cycles"`
}

type Options struct {
	Name          string                       // workflow name (for display)
	Workflow      Workflow                     // parsed workflow definition
	Config        config.Config                // brr config (for profile resolution)
	ProfileFlag   string                       // --profile flag override (empty = use stage/config default)
	ResolvePrompt func(string) (string, error) // prompt resolution function
	Notify        func()                       // called on workflow completion (nil = no notification)
	Reset         bool                         // discard saved state and start fresh
}

const planFile = "IMPLEMENTATION_PLAN.md"

const StateFile = ".brr-workflow-state.json"

type State struct {
	Workflow string `json:"workflow"`
	Stage    int    `json:"stage"`
	Cycle    int    `json:"cycle"`
	StartSHA string `json:"start_sha"`
}

func Load(data []byte) (Workflow, error) {
	var wf Workflow
	if err := yaml.Unmarshal(data, &wf); err != nil {
		return wf, fmt.Errorf("parsing workflow: %w", err)
	}
	if wf.MaxCycles == 0 {
		wf.MaxCycles = 3
	}
	return wf, validate(wf)
}

func Resolve(name string) ([]byte, error) {
	if strings.Contains(name, "..") {
		return nil, fmt.Errorf("invalid workflow name: %q", name)
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

func validate(wf Workflow) error {
	if len(wf.Stages) == 0 {
		return fmt.Errorf("workflow has no stages")
	}
	if wf.MaxCycles < 1 {
		return fmt.Errorf("max_cycles must be >= 1, got %d", wf.MaxCycles)
	}

	cycleCount := 0
	for i, s := range wf.Stages {
		if strings.TrimSpace(s.Prompt) == "" {
			return fmt.Errorf("stage %d: prompt is required", i+1)
		}
		if s.Max < 1 {
			return fmt.Errorf("stage %d (%s): max must be >= 1, got %d", i+1, s.Prompt, s.Max)
		}
		if s.Cycle {
			cycleCount++
		}
	}
	if cycleCount > 1 {
		return fmt.Errorf("at most one stage may set cycle: true, found %d", cycleCount)
	}
	return nil
}

// Run executes the workflow. The caller must hold the exclusive lock.
func Run(opts Options) (*engine.Result, error) {
	stages := opts.Workflow.Stages

	cycleStart := -1
	for i, s := range stages {
		if s.Cycle {
			cycleStart = i
			break
		}
	}

	// Determine starting position: resume from state or start fresh
	stageIdx := 0
	cycle := 0
	startSHA := gitHEAD()

	if opts.Reset {
		deleteState()
	} else if saved, err := loadState(); err == nil && saved.Workflow == opts.Name {
		if saved.Stage < len(stages) {
			stageIdx = saved.Stage
			cycle = saved.Cycle
			startSHA = saved.StartSHA
			fmt.Fprintf(os.Stderr, "  %sresuming:%s stage %d/%d", ui.Dim, ui.Reset, stageIdx+1, len(stages))
			if cycle > 0 {
				fmt.Fprintf(os.Stderr, " (cycle %d)", cycle)
			}
			fmt.Fprintf(os.Stderr, "\n")
		}
	}

	printWorkflowSummary(opts)

	// Persist initial state so an immediate interrupt can resume
	trySaveState(&State{Workflow: opts.Name, Stage: stageIdx, Cycle: cycle, StartSHA: startSHA})

	for stageIdx < len(stages) {
		stage := stages[stageIdx]

		promptText, err := opts.ResolvePrompt(stage.Prompt)
		if err != nil {
			return nil, fmt.Errorf("stage %d (%s): %w", stageIdx+1, stage.Prompt, err)
		}

		profileName := stage.Profile
		if profileName == "" {
			profileName = opts.ProfileFlag
		}
		command, resolvedProfile, err := opts.Config.ResolveProfile(profileName)
		if err != nil {
			return nil, fmt.Errorf("stage %d (%s): %w", stageIdx+1, stage.Prompt, err)
		}

		printStageHeader(stageIdx+1, len(stages), stage.Prompt, resolvedProfile, stage.Max, cycle)

		result, err := engine.Run(engine.Options{
			Prompt:   promptText,
			Max:      stage.Max,
			Command:  command,
			SkipLock: true,
		})

		if err != nil {
			// State file stays at current stage — resume will retry this stage
			if result != nil {
				switch result.Reason {
				case engine.ReasonInterrupted:
					return result, err
				case engine.ReasonFailStreak:
					fmt.Fprintf(os.Stderr, "\n  %s%sWorkflow stopped — stage %q failed%s\n", ui.Bold, ui.Red, stage.Prompt, ui.Reset)
					return result, fmt.Errorf("stage %d (%s): %w", stageIdx+1, stage.Prompt, err)
				}
			}
			fmt.Fprintf(os.Stderr, "\n  %s%sWorkflow stopped — stage %q: %v%s\n", ui.Bold, ui.Red, stage.Prompt, err, ui.Reset)
			return result, fmt.Errorf("stage %d (%s): %w", stageIdx+1, stage.Prompt, err)
		}

		if result != nil && result.Reason == engine.ReasonFailed {
			// State file stays at current stage — resume will retry after failure is investigated
			fmt.Fprintf(os.Stderr, "\n  %s%sWorkflow stopped — stage %q failed%s\n", ui.Bold, ui.Red, stage.Prompt, ui.Reset)
			fmt.Fprintf(os.Stderr, "  %sDelete .brr-failed and re-run to resume.%s\n", ui.Dim, ui.Reset)
			return result, nil
		}

		if result != nil && result.Reason == engine.ReasonApproval {
			// State file stays at current stage — resume will retry after approval is resolved
			fmt.Fprintf(os.Stderr, "\n  %s%sWorkflow paused — stage %q needs approval%s\n", ui.Bold, ui.Yellow, stage.Prompt, ui.Reset)
			fmt.Fprintf(os.Stderr, "  %sDelete .brr-needs-approval and re-run to resume.%s\n", ui.Dim, ui.Reset)
			return result, nil
		}

		stageIdx++

		// After the last stage: check for cycle
		if stageIdx >= len(stages) && cycleStart >= 0 {
			if hasUnfinishedTasks() && cycle < opts.Workflow.MaxCycles {
				cycle++
				stageIdx = cycleStart
				fmt.Fprintf(os.Stderr, "\n%s━━━%s %s%sCycle %d/%d%s %s▸ tasks remain, restarting from %s ━━━%s\n",
					ui.Dim, ui.Reset,
					ui.Bold, ui.Magenta, cycle, opts.Workflow.MaxCycles, ui.Reset,
					ui.Dim, stages[cycleStart].Prompt, ui.Reset,
				)
			}
		}

		// Persist progress: next stage to run
		trySaveState(&State{Workflow: opts.Name, Stage: stageIdx, Cycle: cycle, StartSHA: startSHA})
	}

	// Workflow done — clean up state file
	deleteState()

	if hasUnfinishedTasks() {
		fmt.Fprintf(os.Stderr, "\n  %s%sWorkflow complete — unresolved tasks remain in %s%s\n", ui.Bold, ui.Yellow, planFile, ui.Reset)
	} else {
		fmt.Fprintf(os.Stderr, "\n  %s%sWorkflow complete%s\n", ui.Bold, ui.Green, ui.Reset)
	}

	if opts.Notify != nil {
		opts.Notify()
	}

	return &engine.Result{Reason: engine.ReasonComplete}, nil
}

func loadState() (*State, error) {
	data, err := fsutil.ReadRegularFile(StateFile)
	if err != nil {
		return nil, err
	}
	var s State
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

func trySaveState(s *State) {
	data, err := json.Marshal(s)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not save workflow state: %v\n", err)
		return
	}
	if err := atomicWriteRegularFile(StateFile, data, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not save workflow state: %v\n", err)
	}
}

func deleteState() {
	if fsutil.IsRegularFile(StateFile) {
		_ = os.Remove(StateFile)
	}
}

func atomicWriteRegularFile(path string, data []byte, perm os.FileMode) error {
	if err := rejectNonRegularPath(path); err != nil {
		return err
	}

	dir := filepath.Dir(path)
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

func gitHEAD() string {
	out, err := exec.Command("git", "rev-parse", "HEAD").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func hasUnfinishedTasks() bool {
	data, err := os.ReadFile(planFile)
	if err != nil {
		return false
	}
	return strings.Contains(string(data), "- [ ] ")
}

func printWorkflowSummary(opts Options) {
	fmt.Fprintf(os.Stderr, "  %sworkflow:%s %s\n", ui.Dim, ui.Reset, opts.Name)
	fmt.Fprintf(os.Stderr, "  %sstages:%s  %d\n", ui.Dim, ui.Reset, len(opts.Workflow.Stages))

	hasCycle := false
	for _, s := range opts.Workflow.Stages {
		if s.Cycle {
			hasCycle = true
			break
		}
	}
	if hasCycle {
		fmt.Fprintf(os.Stderr, "  %scycles:%s  %d max\n", ui.Dim, ui.Reset, opts.Workflow.MaxCycles)
	}

	fmt.Fprintf(os.Stderr, "\n")
	for i, s := range opts.Workflow.Stages {
		marker := " "
		if s.Cycle {
			marker = "↻"
		}
		fmt.Fprintf(os.Stderr, "  %s %s%d.%s %s %s(max %d)%s\n",
			marker, ui.Bold, i+1, ui.Reset, s.Prompt, ui.Dim, s.Max, ui.Reset,
		)
	}
	fmt.Fprintf(os.Stderr, "\n")
}

func printStageHeader(num, total int, prompt, profile string, max, cycle int) {
	cycleLabel := ""
	if cycle > 0 {
		cycleLabel = fmt.Sprintf(" %s[cycle %d]%s", ui.Magenta, cycle, ui.Reset)
	}
	fmt.Fprintf(os.Stderr, "\n%s━━━%s %s%sStage %d/%d — %s%s %s▸ %s (max %d)%s ━━━%s\n",
		ui.Dim, ui.Reset,
		ui.Bold, ui.Cyan, num, total, prompt, ui.Reset,
		ui.Dim, profile, max, cycleLabel, ui.Reset,
	)
}
