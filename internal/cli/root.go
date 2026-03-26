package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hl/brr/internal/config"
	"github.com/hl/brr/internal/engine"
	"github.com/hl/brr/internal/ui"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:          "brr <prompt> [flags]",
	Short:        "Your AI agent, but unhinged",
	Long:         "brr runs a prompt in a loop, spinning up a fresh session for each iteration.",
	Args:         cobra.ExactArgs(1),
	RunE:         run,
	SilenceUsage: true,
}

func init() {
	rootCmd.Flags().IntP("max", "m", 0, "max iterations (0 = unlimited)")
	rootCmd.Flags().StringP("profile", "p", "", "agent profile from .brr.yaml (uses 'default' if omitted)")
}

// SetVersion configures the version string shown by --version.
func SetVersion(version, commit string) {
	rootCmd.Version = fmt.Sprintf("%s (%s)", version, commit)
}

func run(cmd *cobra.Command, args []string) error {
	max, _ := cmd.Flags().GetInt("max")
	if max < 0 {
		return fmt.Errorf("--max must be >= 0, got %d", max)
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	profileName, _ := cmd.Flags().GetString("profile")
	command, resolvedName, err := cfg.ResolveProfile(profileName)
	if err != nil {
		return err
	}

	promptText, err := resolvePrompt(args[0])
	if err != nil {
		return err
	}

	if strings.TrimSpace(promptText) == "" {
		return fmt.Errorf("prompt is empty")
	}

	printBanner()
	printConfig(args[0], resolvedName, command, max)

	return engine.Run(engine.Options{
		Prompt:  promptText,
		Max:     max,
		Command: command,
	})
}

// resolvePrompt reads a prompt from a file path, .brr/prompts/<name>.md, or returns it as inline text.
func resolvePrompt(nameOrPath string) (string, error) {
	// If it's an existing file on disk, read it directly
	if _, statErr := os.Stat(nameOrPath); statErr == nil {
		data, err := os.ReadFile(nameOrPath)
		if err != nil {
			return "", fmt.Errorf("reading prompt file: %w", err)
		}
		return string(data), nil
	} else if looksLikeFilePath(nameOrPath) {
		// It looks like a file path but doesn't exist — that's an error, not inline text
		return "", fmt.Errorf("prompt file not found: %s", nameOrPath)
	}

	// For bare names (no spaces), try named prompt resolution
	if !strings.Contains(nameOrPath, " ") {
		name := strings.TrimSuffix(nameOrPath, ".md")

		// Reject path traversal attempts
		if strings.Contains(name, "..") {
			return "", fmt.Errorf("invalid prompt name: %q", name)
		}

		// Try .brr/prompts/<name>.md
		projectPath := filepath.Join(".brr", "prompts", name+".md")
		if data, err := os.ReadFile(projectPath); !errors.Is(err, os.ErrNotExist) && err != nil {
			return "", fmt.Errorf("reading %s: %w", projectPath, err)
		} else if err == nil {
			return string(data), nil
		}

		// Try user config dir prompts/<name>.md
		if configDir, err := os.UserConfigDir(); err == nil {
			userPath := filepath.Join(configDir, "brr", "prompts", name+".md")
			if data, err := os.ReadFile(userPath); !errors.Is(err, os.ErrNotExist) && err != nil {
				return "", fmt.Errorf("reading %s: %w", userPath, err)
			} else if err == nil {
				return string(data), nil
			}
		}
	}

	// Treat as inline prompt text
	return nameOrPath, nil
}

// looksLikeFilePath returns true if s looks like a file path (has path separator
// or a recognized prompt file extension).
func looksLikeFilePath(s string) bool {
	if strings.ContainsRune(s, os.PathSeparator) {
		return true
	}
	ext := filepath.Ext(s)
	switch ext {
	case ".md", ".txt", ".prompt":
		return true
	}
	return false
}

func printBanner() {
	fmt.Println()
	fmt.Printf("  %s%s██████╗ ██████╗ ██████╗%s\n", ui.Bold, ui.Cyan, ui.Reset)
	fmt.Printf("  %s%s██╔══██╗██╔══██╗██╔══██╗%s\n", ui.Bold, ui.Blue, ui.Reset)
	fmt.Printf("  %s%s██████╔╝██████╔╝██████╔╝%s\n", ui.Bold, ui.Magenta, ui.Reset)
	fmt.Printf("  %s%s██╔══██╗██╔══██╗██╔══██╗%s\n", ui.Bold, ui.Red, ui.Reset)
	fmt.Printf("  %s%s██████╔╝██║  ██║██║  ██║%s\n", ui.Bold, ui.Yellow, ui.Reset)
	fmt.Printf("  %s%s╚═════╝ ╚═╝  ╚═╝╚═╝  ╚═╝%s\n", ui.Bold, ui.Green, ui.Reset)
	fmt.Printf("  %syour AI agent, but unhinged%s\n", ui.Dim, ui.Reset)
	fmt.Println()
}

func printConfig(promptName string, profileName string, command []string, max int) {
	maxLabel := "unlimited"
	if max > 0 {
		maxLabel = fmt.Sprintf("%d", max)
	}
	fmt.Printf("  %sprompt:%s  %s\n", ui.Dim, ui.Reset, promptName)
	fmt.Printf("  %sprofile:%s %s %s(%s)%s\n", ui.Dim, ui.Reset, profileName, ui.Dim, command[0], ui.Reset)
	fmt.Printf("  %smax:%s     %s\n", ui.Dim, ui.Reset, maxLabel)
	fmt.Println()
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
