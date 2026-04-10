package cli

import (
	"fmt"
	"os"

	"github.com/hl/brr/internal/scaffold"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Scaffold a project for brr",
	Long:  "Generate .brr.yaml and .brr/prompts/ to get started.",
	Args:  cobra.NoArgs,
	RunE:  runInit,
}

func init() {
	initCmd.Flags().Bool("force", false, "overwrite existing .brr.yaml")
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	force, err := cmd.Flags().GetBool("force")
	if err != nil {
		return fmt.Errorf("reading --force flag: %w", err)
	}

	fmt.Fprintln(os.Stderr)
	return scaffold.Init(force)
}
