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
	rb := rollbackState{
		promptDir:   filepath.Join(".brr", "prompts"),
		workflowDir: filepath.Join(".brr", "workflows"),
	}
	if yamlExists {
		data, err := os.ReadFile(".brr.yaml")
		if err != nil {
			return fmt.Errorf("cannot back up .brr.yaml for rollback: %w", err)
		}
		rb.yamlData = data
		rb.yamlMode = yamlInfo.Mode().Perm()
		rb.yamlExisted = true
	}

	// Track whether dirs existed before we started (for rollback)
	_, promptDirStatErr := os.Lstat(rb.promptDir)
	rb.promptDirIsNew = os.IsNotExist(promptDirStatErr)
	_, workflowDirStatErr := os.Lstat(rb.workflowDir)
	rb.workflowDirIsNew = os.IsNotExist(workflowDirStatErr)

	// Stage 1: write .brr.yaml (re-verify no symlink swap before writing)
	if err := rejectSymlink(".brr.yaml"); err != nil {
		return err
	}
	if err := writeBrrYAML(); err != nil {
		return err
	}

	// Stage 2: create .brr/prompts/ and .brr/workflows/
	if err := os.MkdirAll(rb.promptDir, 0o755); err != nil {
		if rErr := restoreFile(".brr.yaml", rb.yamlData, rb.yamlMode, rb.yamlExisted); rErr != nil {
			fmt.Fprintf(os.Stderr, "warning: rollback of .brr.yaml failed: %v\n", rErr)
		}
		return err
	}
	if err := os.MkdirAll(rb.workflowDir, 0o755); err != nil {
		rb.rollback()
		return err
	}

	// Stage 3: update .gitignore (re-verify no symlink swap)
	if err := rejectSymlink(".gitignore"); err != nil {
		rb.rollback()
		return err
	}
	gitignoreUpdated, err := updateGitignore()
	if err != nil {
		rb.rollback()
		return fmt.Errorf("updating .gitignore: %w", err)
	}

	created := []string{".brr.yaml", ".brr/prompts/", ".brr/workflows/"}
	if gitignoreUpdated {
		created = append(created, ".gitignore (updated)")
	}

	fmt.Fprintf(os.Stderr, "  Created:\n")
	for _, f := range created {
		fmt.Fprintf(os.Stderr, "    %s\n", f)
	}
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "  Next steps:")
	fmt.Fprintln(os.Stderr, "    1. Copy prompts to .brr/prompts/ (examples: https://github.com/hl/brr/tree/main/prompts)")
	fmt.Fprintln(os.Stderr, "    2. Copy workflows to .brr/workflows/ (example: prompts/workflows/ship.yaml)")
	fmt.Fprintln(os.Stderr, "    3. Run them: brr plan  or  brr workflow ship")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "  Docs: https://github.com/hl/brr")

	return nil
}

type rollbackState struct {
	yamlData         []byte
	yamlMode         os.FileMode
	yamlExisted      bool
	promptDir        string
	promptDirIsNew   bool
	workflowDir      string
	workflowDirIsNew bool
}

func (rb *rollbackState) rollback() {
	if rErr := restoreFile(".brr.yaml", rb.yamlData, rb.yamlMode, rb.yamlExisted); rErr != nil {
		fmt.Fprintf(os.Stderr, "warning: rollback of .brr.yaml failed: %v\n", rErr)
	}
	if rb.workflowDirIsNew {
		_ = os.Remove(rb.workflowDir)
	}
	if rb.promptDirIsNew {
		_ = os.Remove(rb.promptDir)
		_ = os.Remove(filepath.Dir(rb.promptDir)) // .brr/ (only succeeds if empty)
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
	".brr-workflow-state.json",
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
