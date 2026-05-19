package cli

import (
	_ "embed"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

//go:embed agent_instructions.md
var agentInstructions string

var instructionsCmd = &cobra.Command{
	Use:   "instructions",
	Short: "Print agent setup instructions",
	Long:  "Print Markdown instructions for creating brr config, prompts, and workflows from an installed binary.",
	Args:  cobra.NoArgs,
	RunE:  printInstructions,
}

func init() {
	rootCmd.AddCommand(instructionsCmd)
}

func printInstructions(cmd *cobra.Command, _ []string) error {
	text := agentInstructions
	if !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	_, err := fmt.Fprint(cmd.OutOrStdout(), text)
	return err
}
