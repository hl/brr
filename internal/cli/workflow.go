package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/hl/brr/internal/config"
	"github.com/hl/brr/internal/engine"
	"github.com/hl/brr/internal/notify"
	"github.com/hl/brr/internal/workflow"
	"github.com/spf13/cobra"
)

var workflowCmd = &cobra.Command{
	Use:   "workflow <name>",
	Short: "Run a multi-stage workflow",
	Long: `Run a multi-stage workflow defined in .brr/workflows/<name>.yaml.

Each stage runs the loop engine with its own prompt and iteration limit.
The workflow cycles back to a designated stage when any stage creates
.brr-cycle.

Progress is saved to .brr-workflow-state.json after each stage. If the
workflow is interrupted or paused for approval, re-running the same command
resumes from where it left off. Use --reset to start from scratch.

Example workflow:
  stages:
    - prompt: prepare
      max: 2
    - prompt: work
      max: 100
      cycle: true
    - prompt: check
      max: 2
  max_cycles: 3`,
	Args:         cobra.ExactArgs(1),
	RunE:         runWorkflow,
	SilenceUsage: true,
}

func init() {
	workflowCmd.Flags().StringP("profile", "p", "", "default profile for all stages (overridden by per-stage profile)")
	workflowCmd.Flags().BoolP("notify", "n", false, "send a desktop notification when the workflow completes")
	workflowCmd.Flags().Bool("reset", false, "discard saved progress and start from the first stage")
	rootCmd.AddCommand(workflowCmd)
}

func runWorkflow(cmd *cobra.Command, args []string) error {
	name := args[0]

	data, err := workflow.Resolve(name)
	if err != nil {
		return err
	}

	wf, err := workflow.Load(data)
	if err != nil {
		return err
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	profileFlag, err := cmd.Flags().GetString("profile")
	if err != nil {
		return fmt.Errorf("reading --profile flag: %w", err)
	}

	doNotify, err := cmd.Flags().GetBool("notify")
	if err != nil {
		return fmt.Errorf("reading --notify flag: %w", err)
	}

	reset, err := cmd.Flags().GetBool("reset")
	if err != nil {
		return fmt.Errorf("reading --reset flag: %w", err)
	}

	printBanner()

	// Acquire lock for the entire workflow
	lf, err := engine.AcquireLock()
	if err != nil {
		return err
	}
	defer engine.ReleaseLock(lf)

	var notifyFn func()
	if doNotify {
		notifyFn = func() {
			result := &engine.Result{Reason: engine.ReasonComplete}
			if nErr := notify.Send(result); nErr != nil {
				fmt.Fprintf(os.Stderr, "warning: notification failed: %v\n", nErr)
			}
		}
	}

	result, runErr := workflow.Run(workflow.Options{
		Name:          name,
		Workflow:      wf,
		Config:        cfg,
		ProfileFlag:   profileFlag,
		ResolvePrompt: resolvePrompt,
		Notify:        notifyFn,
		Reset:         reset,
	})

	if runErr != nil {
		if result != nil && result.Reason == engine.ReasonInterrupted {
			cmd.SilenceErrors = true
		}
		if doNotify && result != nil && result.Reason != engine.ReasonInterrupted {
			if nErr := notify.Send(result); nErr != nil {
				fmt.Fprintf(os.Stderr, "warning: notification failed: %v\n", nErr)
			}
		}
	} else if doNotify && result != nil && result.Reason == engine.ReasonFailed {
		if nErr := notify.Send(result); nErr != nil {
			fmt.Fprintf(os.Stderr, "warning: notification failed: %v\n", nErr)
		}
	}

	if errors.Is(runErr, engine.ErrInterrupted) {
		return engine.ErrInterrupted
	}

	return runErr
}
