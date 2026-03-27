package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hl/brr/internal/config"
	"github.com/hl/brr/internal/engine"
	"github.com/hl/brr/internal/fsutil"
	"github.com/hl/brr/internal/ui"
	"github.com/spf13/cobra"
)

const exitCodeSIGINT = 130 // 128 + SIGINT(2)

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

	defaultHelp := rootCmd.HelpFunc()
	rootCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		if !cmd.HasParent() {
			printBanner()
			fmt.Printf("  %shttps://github.com/hl/brr%s\n\n", ui.Dim, ui.Reset)
		}
		defaultHelp(cmd, args)
	})
}

// SetVersion configures the version string shown by --version.
func SetVersion(version, commit string) {
	rootCmd.Version = fmt.Sprintf("%s (%s)", version, commit)
}

func run(cmd *cobra.Command, args []string) error {
	max, err := cmd.Flags().GetInt("max")
	if err != nil {
		return fmt.Errorf("reading --max flag: %w", err)
	}
	if max < 0 {
		return fmt.Errorf("--max must be >= 0, got %d", max)
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	profileName, err := cmd.Flags().GetString("profile")
	if err != nil {
		return fmt.Errorf("reading --profile flag: %w", err)
	}
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

	err = engine.Run(engine.Options{
		Prompt:  promptText,
		Max:     max,
		Command: command,
	})
	if errors.Is(err, engine.ErrInterrupted) {
		cmd.SilenceErrors = true
	}
	return err
}

// resolvePrompt reads a prompt from a file path, .brr/prompts/<name>.md, or returns it as inline text.
func resolvePrompt(nameOrPath string) (string, error) {
	// If it's an existing regular file, read it directly
	if fi, statErr := os.Stat(nameOrPath); statErr == nil {
		if fi.IsDir() {
			// Don't treat directories as prompt files — fall through to named prompt lookup
		} else if data, err := os.ReadFile(nameOrPath); err == nil {
			return string(data), nil
		} else {
			return "", fmt.Errorf("reading prompt file: %w", err)
		}
	} else if looksLikeFilePath(nameOrPath) {
		// It looks like a file path — distinguish "not found" from other stat errors
		if os.IsNotExist(statErr) {
			return "", fmt.Errorf("prompt file not found: %s", nameOrPath)
		}
		return "", fmt.Errorf("accessing prompt file %s: %w", nameOrPath, statErr)
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
		if data, err := fsutil.ReadRegularFile(projectPath); err == nil {
			return string(data), nil
		} else if !errors.Is(err, os.ErrNotExist) && !errors.Is(err, fsutil.ErrNotRegularFile) {
			return "", fmt.Errorf("reading %s: %w", projectPath, err)
		}

		// Try user config dir prompts/<name>.md
		if configDir, err := os.UserConfigDir(); err == nil {
			userPath := filepath.Join(configDir, "brr", "prompts", name+".md")
			if data, err := fsutil.ReadRegularFile(userPath); err == nil {
				return string(data), nil
			} else if !errors.Is(err, os.ErrNotExist) && !errors.Is(err, fsutil.ErrNotRegularFile) {
				return "", fmt.Errorf("reading %s: %w", userPath, err)
			}
		}
	}

	// Treat as inline prompt text
	return nameOrPath, nil
}

// looksLikeFilePath returns true if s looks like a file path rather than inline prompt text.
func looksLikeFilePath(s string) bool {
	hasSep := strings.ContainsRune(s, filepath.Separator) || strings.ContainsRune(s, '/')
	hasPromptExt := isPromptExtension(filepath.Ext(s))

	if strings.Contains(s, " ") {
		// With spaces: only a file path if it has BOTH a separator and a recognized extension
		// e.g. "docs/Build Plan.md" → file path; "Fix stuff in src/" → inline text
		return hasSep && hasPromptExt
	}
	// Without spaces: separator alone or recognized extension alone → file path
	return hasSep || hasPromptExt
}

func isPromptExtension(ext string) bool {
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
		if errors.Is(err, engine.ErrInterrupted) {
			os.Exit(exitCodeSIGINT)
		}
		os.Exit(1)
	}
}
