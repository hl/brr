package workflow

import (
	"fmt"
	"os"
	"strings"

	"github.com/hl/brr/internal/engine"
	"github.com/hl/brr/internal/ui"
)

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

func stageStatusFromResult(result *engine.Result, err error) string {
	if result != nil && result.Reason == engine.ReasonInterrupted {
		return "interrupted"
	}
	if err != nil {
		return "error"
	}
	if result == nil {
		return "completed"
	}
	switch result.Reason {
	case engine.ReasonFailed:
		return "failed"
	case engine.ReasonApproval:
		return "approval"
	case engine.ReasonCycle:
		return "cycle"
	default:
		return "completed"
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
