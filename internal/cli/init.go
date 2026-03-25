package cli

import (
	"fmt"

	"github.com/hl/brr/internal/scaffold"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Scaffold a project for brr",
	Long:  "Generate .brr.yaml and AGENTS.md to get started.",
	Args:  cobra.NoArgs,
	RunE:  runInit,
}

func init() {
	initCmd.Flags().Bool("force", false, "overwrite existing .brr.yaml")
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	force, _ := cmd.Flags().GetBool("force")

	fmt.Println()
	return scaffold.Init(force)
}
