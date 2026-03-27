package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolvePromptInlineText(t *testing.T) {
	t.Chdir(t.TempDir())

	text, err := resolvePrompt("Fix all the bugs")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text != "Fix all the bugs" {
		t.Errorf("expected inline text, got %q", text)
	}
}

func TestResolvePromptFromFile(t *testing.T) {
	t.Chdir(t.TempDir())

	if err := os.WriteFile("my-prompt.md", []byte("do the thing"), 0o644); err != nil {
		t.Fatal(err)
	}

	text, err := resolvePrompt("my-prompt.md")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text != "do the thing" {
		t.Errorf("expected file content, got %q", text)
	}
}

func TestResolvePromptDirectoryFallsThrough(t *testing.T) {
	t.Chdir(t.TempDir())

	// Create a directory named "plan" AND a named prompt .brr/prompts/plan.md
	if err := os.Mkdir("plan", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(".brr", "prompts"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(".brr", "prompts", "plan.md"), []byte("named prompt"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Should skip the directory and resolve the named prompt
	text, err := resolvePrompt("plan")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text != "named prompt" {
		t.Errorf("expected named prompt, got %q", text)
	}
}

func TestResolvePromptNamedFromProject(t *testing.T) {
	t.Chdir(t.TempDir())

	if err := os.MkdirAll(filepath.Join(".brr", "prompts"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(".brr", "prompts", "plan.md"), []byte("planning prompt"), 0o644); err != nil {
		t.Fatal(err)
	}

	text, err := resolvePrompt("plan")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text != "planning prompt" {
		t.Errorf("expected named prompt content, got %q", text)
	}
}

func TestResolvePromptMissingFile(t *testing.T) {
	t.Chdir(t.TempDir())

	_, err := resolvePrompt("nonexistent.md")
	if err == nil {
		t.Error("expected error for missing file with .md extension")
	}
}

func TestResolvePromptPathTraversal(t *testing.T) {
	t.Chdir(t.TempDir())

	// This exercises the looksLikeFilePath path-separator check (file not found)
	_, err := resolvePrompt("../../etc/passwd")
	if err == nil {
		t.Error("expected error for path traversal attempt")
	}
}

func TestResolvePromptNamedPathTraversalGuard(t *testing.T) {
	t.Chdir(t.TempDir())

	// This exercises the explicit ".." rejection in the named prompt branch.
	// Use a bare name without spaces/separators so it enters named prompt resolution.
	_, err := resolvePrompt("..secret")
	if err == nil {
		t.Error("expected error for named prompt with '..' in name")
	}
	if err != nil && !strings.Contains(err.Error(), "invalid prompt name") {
		t.Errorf("expected 'invalid prompt name' error, got: %v", err)
	}
}

func TestResolvePromptDottedInlineText(t *testing.T) {
	t.Chdir(t.TempDir())

	// A bare name with dots but no recognized extension should be tried as a named prompt
	// and then fall through to inline text (not error as "file not found")
	text, err := resolvePrompt("v1.2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text != "v1.2" {
		t.Errorf("expected inline text, got %q", text)
	}
}

func TestResolvePromptInlineTextWithSlash(t *testing.T) {
	t.Chdir(t.TempDir())

	// Inline prompts containing slashes should work (README example)
	text, err := resolvePrompt("Fix all TODO comments in src/")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text != "Fix all TODO comments in src/" {
		t.Errorf("expected inline text, got %q", text)
	}
}

func TestResolvePromptInlineTextWithPathInMiddle(t *testing.T) {
	t.Chdir(t.TempDir())

	text, err := resolvePrompt("Refactor pkg/config/loader.go to use generics")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text != "Refactor pkg/config/loader.go to use generics" {
		t.Errorf("expected inline text, got %q", text)
	}
}

func TestLooksLikeFilePathWithSeparatorNoSpaces(t *testing.T) {
	// A path-like string with separators is treated as a file path
	if !looksLikeFilePath("path/to/file") {
		t.Error("expected true for path with separators")
	}
}

func TestLooksLikeFilePathWithSeparatorAndExtension(t *testing.T) {
	// path/to/file.md without spaces IS a file path
	if !looksLikeFilePath("path/to/file.md") {
		t.Error("expected true for path with .md extension")
	}
}

func TestLooksLikeFilePathWithSpaces(t *testing.T) {
	// Text with spaces and slashes but no prompt extension → inline text
	if looksLikeFilePath("Fix stuff in src/") {
		t.Error("expected false for text with spaces and no prompt extension")
	}
}

func TestLooksLikeFilePathWithSpacesAndExtension(t *testing.T) {
	// Path with spaces AND separator AND prompt extension → file path
	if !looksLikeFilePath("docs/Build Plan.md") {
		t.Error("expected true for path with spaces, separator, and .md extension")
	}
	// Extension without separator → inline text (e.g. "fix login.md module")
	if looksLikeFilePath("fix login.md") {
		t.Error("expected false for text with spaces and extension but no separator")
	}
}

func TestLooksLikeFilePathWithExtension(t *testing.T) {
	if !looksLikeFilePath("prompt.md") {
		t.Error("expected true for .md extension")
	}
	if !looksLikeFilePath("notes.txt") {
		t.Error("expected true for .txt extension")
	}
}

func TestLooksLikeFilePathPlainName(t *testing.T) {
	if looksLikeFilePath("plan") {
		t.Error("expected false for plain name without extension")
	}
}

func TestLooksLikeFilePathDottedName(t *testing.T) {
	// Dotted names like "v1.2" should NOT look like file paths
	if looksLikeFilePath("v1.2") {
		t.Error("expected false for dotted name without recognized extension")
	}
}

func TestResolvePromptNamedSymlinkRejected(t *testing.T) {
	t.Chdir(t.TempDir())

	// Create a symlink at .brr/prompts/evil.md -> /etc/hosts
	if err := os.MkdirAll(filepath.Join(".brr", "prompts"), 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(t.TempDir(), "secret.txt")
	if err := os.WriteFile(target, []byte("secret data"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(".brr", "prompts", "evil.md")); err != nil {
		t.Skip("symlinks not supported")
	}

	// Should NOT read through the symlink — should fall through to inline text
	text, err := resolvePrompt("evil")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text == "secret data" {
		t.Error("symlink in .brr/prompts/ should not be followed")
	}
	if text != "evil" {
		t.Errorf("expected fallthrough to inline text 'evil', got %q", text)
	}
}

func TestResolvePromptPathWithSeparator(t *testing.T) {
	t.Chdir(t.TempDir())

	// A path-like string with separators but no recognized extension
	// should error, not become inline text
	_, err := resolvePrompt("prompts/build")
	if err == nil {
		t.Error("expected error for path-like argument without extension")
	}
}
