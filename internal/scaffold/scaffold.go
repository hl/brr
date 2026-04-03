package scaffold

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Init scaffolds a project for brr.
func Init(force bool) error {
	// Pre-flight: reject symlinks to prevent writes outside the repo
	for _, path := range []string{".brr.yaml", ".gitignore", ".brr"} {
		if err := rejectSymlink(path); err != nil {
			return err
		}
	}

	// Check if .brr.yaml exists using Lstat (works even for write-only files)
	yamlInfo, lstatErr := os.Lstat(".brr.yaml")
	yamlExists := lstatErr == nil

	if yamlExists && !force {
		return fmt.Errorf(".brr.yaml already exists (use --force to overwrite)")
	}

	// Backup for rollback — abort if file exists but can't be read (can't guarantee restore)
	var existingYAML []byte
	var existingMode os.FileMode
	if yamlExists {
		data, err := os.ReadFile(".brr.yaml")
		if err != nil {
			return fmt.Errorf("cannot back up .brr.yaml for rollback: %w", err)
		}
		existingYAML = data
		existingMode = yamlInfo.Mode().Perm()
	}

	// Track whether prompts dir existed before we started (for rollback)
	promptDir := filepath.Join(".brr", "prompts")
	_, promptDirStatErr := os.Lstat(promptDir)
	promptDirIsNew := os.IsNotExist(promptDirStatErr)

	// Stage 1: write .brr.yaml (re-verify no symlink swap before writing)
	if err := rejectSymlink(".brr.yaml"); err != nil {
		return err
	}
	if err := writeBrrYAML(); err != nil {
		return err
	}

	// Stage 2: create .brr/prompts/
	if err := os.MkdirAll(promptDir, 0o755); err != nil {
		if rErr := restoreFile(".brr.yaml", existingYAML, existingMode, yamlExists); rErr != nil {
			fmt.Fprintf(os.Stderr, "warning: rollback of .brr.yaml failed: %v\n", rErr)
		}
		return err
	}

	// Stage 3: update .gitignore (re-verify no symlink swap)
	if err := rejectSymlink(".gitignore"); err != nil {
		rollbackInit(existingYAML, existingMode, yamlExists, promptDirIsNew, promptDir)
		return err
	}
	gitignoreUpdated, err := updateGitignore()
	if err != nil {
		rollbackInit(existingYAML, existingMode, yamlExists, promptDirIsNew, promptDir)
		return fmt.Errorf("updating .gitignore: %w", err)
	}

	created := []string{".brr.yaml", ".brr/prompts/"}
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

// rollbackInit undoes partial Init work, logging any rollback failures.
func rollbackInit(existingYAML []byte, existingMode os.FileMode, yamlExists, promptDirIsNew bool, promptDir string) {
	if rErr := restoreFile(".brr.yaml", existingYAML, existingMode, yamlExists); rErr != nil {
		fmt.Fprintf(os.Stderr, "warning: rollback of .brr.yaml failed: %v\n", rErr)
	}
	if promptDirIsNew {
		_ = os.Remove(promptDir)               // .brr/prompts/
		_ = os.Remove(filepath.Dir(promptDir)) // .brr/ (only succeeds if empty)
	}
}

// rejectSymlink returns an error if path exists and is a symlink.
func rejectSymlink(path string) error {
	fi, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("checking %s: %w", path, err)
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("%s is a symlink — refusing to overwrite (security risk)", path)
	}
	return nil
}

// restoreFile restores a file to its previous state, preserving the original permissions.
// Returns an error if the rollback itself fails.
func restoreFile(path string, data []byte, mode os.FileMode, existed bool) error {
	if existed {
		return os.WriteFile(path, data, mode)
	}
	err := os.Remove(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// gitignoreEntries are lines brr needs in .gitignore.
var gitignoreEntries = []string{
	".brr-complete",
	".brr-needs-approval",
	".brr.lock",
}

// updateGitignore appends missing brr entries to .gitignore.
// Returns true if any entries were added.
func updateGitignore() (bool, error) {
	existing, err := os.ReadFile(".gitignore")
	if err != nil && !os.IsNotExist(err) {
		return false, err
	}
	content := string(existing)

	// Parse existing lines to find exact matches (ignoring comments and whitespace)
	existingLines := make(map[string]bool)
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && !strings.HasPrefix(trimmed, "#") {
			existingLines[trimmed] = true
		}
	}

	var missing []string
	for _, entry := range gitignoreEntries {
		if !existingLines[entry] {
			missing = append(missing, entry)
		}
	}

	if len(missing) == 0 {
		return false, nil
	}

	var buf strings.Builder
	// Add a blank line separator if file doesn't end with newline
	if len(content) > 0 && !strings.HasSuffix(content, "\n") {
		buf.WriteString("\n")
	}
	buf.WriteString("\n# brr\n")
	for _, entry := range missing {
		buf.WriteString(entry + "\n")
	}

	f, err := os.OpenFile(".gitignore", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return false, err
	}
	_, writeErr := f.WriteString(buf.String())
	closeErr := f.Close()
	if err := errors.Join(writeErr, closeErr); err != nil {
		return false, err
	}

	return true, nil
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

  compile:
    command: claude
    args: [-p, --dangerously-skip-permissions, --model, opus, --max-turns, "500"]

  codex:
    command: codex
    args: [exec, --ephemeral, --dangerously-bypass-approvals-and-sandbox, --model, gpt-5.4, -]
`
	return atomicWriteFile(".brr.yaml", []byte(content), 0o644)
}

// atomicWriteFile writes data to a temp file then renames it to path,
// preventing TOCTOU races where a symlink is swapped in between check and write.
func atomicWriteFile(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".brr-tmp-*")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpName := tmp.Name()
	defer func() {
		// Clean up temp file on any failure path
		_ = os.Remove(tmpName)
	}()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(perm); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}
