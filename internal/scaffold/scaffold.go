package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
)

// Recipe defines project-type-specific defaults.
type Recipe struct {
	Name       string
	Validation []string
}

var recipes = map[string]Recipe{
	"elixir": {
		Name: "elixir",
		Validation: []string{
			"mix compile --warnings-as-errors",
			"mix test",
			"mix format --check-formatted",
		},
	},
	"node": {
		Name:       "node",
		Validation: []string{"npm test"},
	},
	"go": {
		Name: "go",
		Validation: []string{
			"go build ./...",
			"go test ./...",
			"go vet ./...",
		},
	},
	"rust": {
		Name: "rust",
		Validation: []string{
			"cargo build",
			"cargo test",
			"cargo clippy",
		},
	},
	"python": {
		Name:       "python",
		Validation: []string{"pytest"},
	},
	"generic": {
		Name:       "generic",
		Validation: nil,
	},
}

// Detect identifies the project type from marker files in the working directory.
func Detect() Recipe {
	markers := []struct {
		file   string
		recipe string
	}{
		{"mix.exs", "elixir"},
		{"package.json", "node"},
		{"go.mod", "go"},
		{"Cargo.toml", "rust"},
		{"pyproject.toml", "python"},
	}

	for _, m := range markers {
		if _, err := os.Stat(m.file); err == nil {
			return recipes[m.recipe]
		}
	}

	return recipes["generic"]
}

// GetRecipe returns a recipe by name. Returns an error if the recipe doesn't exist.
func GetRecipe(name string) (Recipe, error) {
	if r, ok := recipes[name]; ok {
		return r, nil
	}
	available := make([]string, 0, len(recipes))
	for k := range recipes {
		available = append(available, k)
	}
	return Recipe{}, fmt.Errorf("unknown recipe %q (available: %v)", name, available)
}

// Init scaffolds a project for brr.
func Init(recipe Recipe, force bool) error {
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
		if err := writeAgentsMD(recipe); err != nil {
			return err
		}
		created = append(created, "AGENTS.md")
	}

	// docs/specs/
	specDir := "docs/specs"
	if err := os.MkdirAll(specDir, 0o755); err != nil {
		return err
	}
	gitkeep := filepath.Join(specDir, ".gitkeep")
	if _, err := os.Stat(gitkeep); err != nil {
		if err := os.WriteFile(gitkeep, nil, 0o644); err != nil {
			return fmt.Errorf("writing %s: %w", gitkeep, err)
		}
		created = append(created, specDir+"/")
	}

	fmt.Printf("  Detected: %s\n", recipe.Name)
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

func writeAgentsMD(r Recipe) error {
	content := `# Agents

## Validation

` + "```bash\n"

	if len(r.Validation) > 0 {
		for _, v := range r.Validation {
			content += v + "\n"
		}
	} else {
		content += "# Add your validation commands here\n"
	}

	content += "```" + `

## Conventions

- Commit messages: type(scope): description
- Add project-specific conventions here

## Decision Authority

- Routine implementation: proceed without approval
- Changes requiring human review: use [APPROVAL] marker in implementation plan
`
	return os.WriteFile("AGENTS.md", []byte(content), 0o644)
}
