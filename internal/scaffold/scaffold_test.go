package scaffold

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInit(t *testing.T) {
	t.Chdir(t.TempDir())

	if err := Init(false); err != nil {
		t.Fatalf("Init error: %v", err)
	}

	// Check .brr.yaml was created with profiles
	data, err := os.ReadFile(".brr.yaml")
	if err != nil {
		t.Fatal("expected .brr.yaml to exist")
	}
	if !strings.Contains(string(data), "profiles:") {
		t.Error("expected 'profiles:' in .brr.yaml")
	}
	if !strings.Contains(string(data), "default:") {
		t.Error("expected 'default:' in .brr.yaml")
	}

	// Check .brr/prompts/ directory was created
	if _, err := os.Stat(filepath.Join(".brr", "prompts")); err != nil {
		t.Error("expected .brr/prompts/ to exist")
	}

	// Check .brr/workflows/ directory was created
	if _, err := os.Stat(filepath.Join(".brr", "workflows")); err != nil {
		t.Error("expected .brr/workflows/ to exist")
	}

	// AGENTS.md should not be created
	if _, err := os.Stat("AGENTS.md"); err == nil {
		t.Error("expected AGENTS.md to not be created by init")
	}

	// Check .gitignore was created with brr entries
	gitignore, err := os.ReadFile(".gitignore")
	if err != nil {
		t.Fatal("expected .gitignore to exist")
	}
	for _, entry := range gitignoreEntries {
		if !strings.Contains(string(gitignore), entry) {
			t.Errorf("expected %q in .gitignore", entry)
		}
	}
}

func TestInitGitignoreAppendsToExisting(t *testing.T) {
	t.Chdir(t.TempDir())

	if err := os.WriteFile(".gitignore", []byte("node_modules/\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := Init(false); err != nil {
		t.Fatalf("Init error: %v", err)
	}

	data, err := os.ReadFile(".gitignore")
	if err != nil {
		t.Fatal("expected .gitignore to exist")
	}
	content := string(data)

	// Existing content preserved
	if !strings.Contains(content, "node_modules/") {
		t.Error("expected existing .gitignore content to be preserved")
	}
	// brr entries added
	if !strings.Contains(content, ".brr-complete") {
		t.Error("expected .brr-complete in .gitignore")
	}
}

func TestInitGitignoreSkipsExistingEntries(t *testing.T) {
	t.Chdir(t.TempDir())

	if err := os.WriteFile(".gitignore", []byte(".brr-complete\n.brr-failed\n.brr-needs-approval\n.brr-cycle\n.brr.lock\n.brr-workflow-state.json\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := Init(false); err != nil {
		t.Fatalf("Init error: %v", err)
	}

	data, err := os.ReadFile(".gitignore")
	if err != nil {
		t.Fatal(err)
	}
	// Should not have duplicate "# brr" section
	if strings.Contains(string(data), "# brr") {
		t.Error("expected no brr section when all entries already present")
	}
}

func TestInitAlreadyExists(t *testing.T) {
	t.Chdir(t.TempDir())

	if err := os.WriteFile(".brr.yaml", []byte("existing"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := Init(false)
	if err == nil {
		t.Error("expected error when .brr.yaml already exists")
	}
}

func TestInitForce(t *testing.T) {
	t.Chdir(t.TempDir())

	if err := os.WriteFile(".brr.yaml", []byte("existing"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := Init(true); err != nil {
		t.Fatalf("Init with force error: %v", err)
	}

	data, err := os.ReadFile(".brr.yaml")
	if err != nil {
		t.Fatal("expected .brr.yaml to exist")
	}
	if string(data) == "existing" {
		t.Error("expected .brr.yaml to be overwritten")
	}
}

func TestInitSymlinkYAMLRejected(t *testing.T) {
	t.Chdir(t.TempDir())

	// Create a symlink .brr.yaml -> /tmp/target
	target := filepath.Join(t.TempDir(), "target.yaml")
	if err := os.WriteFile(target, []byte("target"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, ".brr.yaml"); err != nil {
		t.Skip("symlinks not supported")
	}

	err := Init(true)
	if err == nil {
		t.Error("expected error when .brr.yaml is a symlink")
	}
	if err != nil && !strings.Contains(err.Error(), "symlink") {
		t.Errorf("expected symlink error, got: %v", err)
	}
}

func TestInitSymlinkGitignoreRejected(t *testing.T) {
	t.Chdir(t.TempDir())

	target := filepath.Join(t.TempDir(), "target-gitignore")
	if err := os.WriteFile(target, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, ".gitignore"); err != nil {
		t.Skip("symlinks not supported")
	}

	err := Init(false)
	if err == nil {
		t.Error("expected error when .gitignore is a symlink")
	}
	if err != nil && !strings.Contains(err.Error(), "symlink") {
		t.Errorf("expected symlink error, got: %v", err)
	}
}

func TestInitSymlinkBrrDirRejected(t *testing.T) {
	t.Chdir(t.TempDir())

	target := filepath.Join(t.TempDir(), "target-brr-dir")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, ".brr"); err != nil {
		t.Skip("symlinks not supported")
	}

	err := Init(false)
	if err == nil {
		t.Error("expected error when .brr is a symlink")
	}
	if err != nil && !strings.Contains(err.Error(), "symlink") {
		t.Errorf("expected symlink error, got: %v", err)
	}
}

func TestInitSymlinkPromptDirRejected(t *testing.T) {
	t.Chdir(t.TempDir())

	if err := os.Mkdir(".brr", 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(t.TempDir(), "target-prompts")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(".brr", "prompts")); err != nil {
		t.Skip("symlinks not supported")
	}

	err := Init(false)
	if err == nil {
		t.Fatal("expected error when .brr/prompts is a symlink")
	}
	if !strings.Contains(err.Error(), "symlink") {
		t.Errorf("expected symlink error, got: %v", err)
	}
}

func TestInitWriteOnlyYAMLRequiresForce(t *testing.T) {
	t.Chdir(t.TempDir())

	// Create a write-only .brr.yaml — Lstat should detect existence
	if err := os.WriteFile(".brr.yaml", []byte("existing"), 0o200); err != nil {
		t.Fatal(err)
	}

	err := Init(false)
	if err == nil {
		t.Error("expected error when write-only .brr.yaml exists without --force")
	}
}

func TestInitForceWriteOnlyAborts(t *testing.T) {
	t.Chdir(t.TempDir())

	// Write-only file can't be backed up — Init should abort even with --force
	if err := os.WriteFile(".brr.yaml", []byte("existing"), 0o200); err != nil {
		t.Fatal(err)
	}

	err := Init(true)
	if err == nil {
		t.Error("expected error when write-only .brr.yaml can't be backed up")
	}
	if err != nil && !strings.Contains(err.Error(), "cannot back up") {
		t.Errorf("expected backup error, got: %v", err)
	}
}

func TestInitGitignoreCommentedEntriesNotMatched(t *testing.T) {
	t.Chdir(t.TempDir())

	// Commented-out entries should not prevent real entries from being added
	if err := os.WriteFile(".gitignore", []byte("# .brr-complete\n# .brr-needs-approval\n# .brr-cycle\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := Init(false); err != nil {
		t.Fatalf("Init error: %v", err)
	}

	data, err := os.ReadFile(".gitignore")
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)

	// Should have the brr section with actual entries
	if !strings.Contains(content, "# brr\n") {
		t.Error("expected brr section to be added when only commented entries exist")
	}

	// Count actual (non-commented) occurrences
	lines := strings.Split(content, "\n")
	realEntries := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		for _, entry := range gitignoreEntries {
			if trimmed == entry {
				realEntries++
			}
		}
	}
	if realEntries != len(gitignoreEntries) {
		t.Errorf("expected %d real gitignore entries, got %d", len(gitignoreEntries), realEntries)
	}
}
