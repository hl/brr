package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Init scaffolds a project for brr.
func Init(force bool) error {
	created := []string{}

	// .brr.yaml
	if _, err := os.Stat(".brr.yaml"); err == nil && !force {
		return fmt.Errorf(".brr.yaml already exists (use --force to overwrite)")
	}
	if err := writeBrrYAML(); err != nil {
		return err
	}
	created = append(created, ".brr.yaml")

	// .brr/prompts/ — create directory for user prompts
	promptDir := filepath.Join(".brr", "prompts")
	if err := os.MkdirAll(promptDir, 0o755); err != nil {
		return err
	}

	// .gitignore — append brr entries if missing
	gitignoreUpdated := updateGitignore()
	if gitignoreUpdated {
		created = append(created, ".gitignore (updated)")
	}

	fmt.Printf("  Created:\n")
	for _, f := range created {
		fmt.Printf("    %s\n", f)
	}
	fmt.Println()
	fmt.Println("  Next steps:")
	fmt.Println("    1. Add prompts to .brr/prompts/ (e.g. plan.md, build.md)")
	fmt.Println("    2. Run them: brr plan  →  resolves to .brr/prompts/plan.md")
	fmt.Println()
	fmt.Println("  Examples:  https://github.com/hl/brr/tree/main/prompts")
	fmt.Println("  AGENTS.md: https://github.com/hl/brr/blob/main/AGENTS.md")

	return nil
}

// gitignoreEntries are lines brr needs in .gitignore.
var gitignoreEntries = []string{
	".brr-complete",
	".brr-needs-approval",
}

// updateGitignore appends missing brr entries to .gitignore.
// Returns true if any entries were added.
func updateGitignore() bool {
	existing, _ := os.ReadFile(".gitignore")
	content := string(existing)

	var missing []string
	for _, entry := range gitignoreEntries {
		if !strings.Contains(content, entry) {
			missing = append(missing, entry)
		}
	}

	if len(missing) == 0 {
		return false
	}

	f, err := os.OpenFile(".gitignore", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return false
	}
	defer f.Close()

	// Add a blank line separator if file doesn't end with newline
	if len(content) > 0 && !strings.HasSuffix(content, "\n") {
		f.WriteString("\n")
	}

	f.WriteString("\n# brr\n")
	for _, entry := range missing {
		f.WriteString(entry + "\n")
	}

	return true
}

func writeBrrYAML() error {
	content := `# brr configuration — see https://github.com/hl/brr

# Default profile to use when --profile is not specified.
default: claude

# Agent profiles. Each profile defines a command and its arguments.
# The prompt is piped to stdin. Switch profiles with: brr <prompt> -p <name>
profiles:
  claude:
    command: claude
    args: [-p, --dangerously-skip-permissions, --model, sonnet, --max-turns, "200"]

  opus:
    command: claude
    args: [-p, --dangerously-skip-permissions, --model, opus, --max-turns, "200"]

  codex:
    command: codex
    args: [exec, --ephemeral, --dangerously-bypass-approvals-and-sandbox, --model, gpt-5.4, -]
`
	return os.WriteFile(".brr.yaml", []byte(content), 0o644)
}
