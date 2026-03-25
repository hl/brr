package cli

import (
	"fmt"

	"github.com/hl/brr/internal/scaffold"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Scaffold a project for brr",
	Long:  "Detect project type and generate .brr.yaml, AGENTS.md, and docs/specs/.",
	Args:  cobra.NoArgs,
	RunE:  runInit,
}

func init() {
	initCmd.Flags().String("recipe", "", "force a specific recipe (elixir, node, go, rust, python, generic)")
	initCmd.Flags().Bool("force", false, "overwrite existing .brr.yaml")
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	force, _ := cmd.Flags().GetBool("force")

	recipeName, _ := cmd.Flags().GetString("recipe")
	var recipe scaffold.Recipe
	if recipeName != "" {
		var err error
		recipe, err = scaffold.GetRecipe(recipeName)
		if err != nil {
			return err
		}
	} else {
		recipe = scaffold.Detect()
	}

	fmt.Println()
	return scaffold.Init(recipe, force)
}
