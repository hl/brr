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

	// Check AGENTS.md was created
	data, err = os.ReadFile("AGENTS.md")
	if err != nil {
		t.Fatal("expected AGENTS.md to exist")
	}
	if !strings.Contains(string(data), "Validation") {
		t.Error("expected Validation section in AGENTS.md")
	}
}

func TestInitAlreadyExists(t *testing.T) {
	t.Chdir(t.TempDir())

	os.WriteFile(".brr.yaml", []byte("existing"), 0o644)

	err := Init(false)
	if err == nil {
		t.Error("expected error when .brr.yaml already exists")
	}
}

func TestInitForce(t *testing.T) {
	t.Chdir(t.TempDir())

	os.WriteFile(".brr.yaml", []byte("existing"), 0o644)

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
