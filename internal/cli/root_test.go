package cli

import (
	"os"
	"path/filepath"
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

	_, err := resolvePrompt("../../etc/passwd")
	if err == nil {
		t.Error("expected error for path traversal attempt")
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

func TestLooksLikeFilePathWithSeparator(t *testing.T) {
	if !looksLikeFilePath("path/to/file") {
		t.Error("expected true for path with separator")
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
