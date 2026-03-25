package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
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

	// AGENTS.md
	if _, err := os.Stat("AGENTS.md"); err != nil {
		if err := writeAgentsMD(); err != nil {
			return err
		}
		created = append(created, "AGENTS.md")
	}

	fmt.Printf("  Created:\n")
	for _, f := range created {
		fmt.Printf("    %s\n", f)
	}
	fmt.Println()
	fmt.Println("  Get example prompts: https://github.com/hl/brr/tree/main/prompts")
	fmt.Println("  Put your prompts in .brr/prompts/, then: brr plan")

	return nil
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

func writeAgentsMD() error {
	content := `# Agents

## Validation

` + "```bash\n" +
		"# Add your validation commands here\n" +
		"```" + `

## Conventions

- Commit messages: type(scope): description
- Add project-specific conventions here

## Decision Authority

- Routine implementation: proceed without approval
- Changes requiring human review: use [APPROVAL] marker in implementation plan
`
	return os.WriteFile("AGENTS.md", []byte(content), 0o644)
}
