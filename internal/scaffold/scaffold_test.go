package scaffold

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDetectElixir(t *testing.T) {
	t.Chdir(t.TempDir())

	os.WriteFile("mix.exs", nil, 0o644)
	r := Detect()
	if r.Name != "elixir" {
		t.Errorf("expected 'elixir', got %q", r.Name)
	}
}

func TestDetectNode(t *testing.T) {
	t.Chdir(t.TempDir())

	os.WriteFile("package.json", nil, 0o644)
	r := Detect()
	if r.Name != "node" {
		t.Errorf("expected 'node', got %q", r.Name)
	}
}

func TestDetectGenericFallback(t *testing.T) {
	t.Chdir(t.TempDir())

	r := Detect()
	if r.Name != "generic" {
		t.Errorf("expected 'generic', got %q", r.Name)
	}
}

func TestGetRecipe(t *testing.T) {
	r, err := GetRecipe("rust")
	if err != nil {
		t.Fatalf("GetRecipe('rust') error: %v", err)
	}
	if r.Name != "rust" {
		t.Errorf("expected 'rust', got %q", r.Name)
	}

	_, err = GetRecipe("unknown")
	if err == nil {
		t.Error("expected error for unknown recipe")
	}
}

func TestInit(t *testing.T) {
	t.Chdir(t.TempDir())

	r, _ := GetRecipe("elixir")
	if err := Init(r, false); err != nil {
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

	// Check AGENTS.md was created with validation
	data, err = os.ReadFile("AGENTS.md")
	if err != nil {
		t.Fatal("expected AGENTS.md to exist")
	}
	if !strings.Contains(string(data), "mix compile") {
		t.Error("expected elixir validation in AGENTS.md")
	}

	// Check docs/specs/ was created
	if _, err := os.Stat(filepath.Join("docs", "specs", ".gitkeep")); err != nil {
		t.Error("expected docs/specs/.gitkeep to exist")
	}
}

func TestInitAlreadyExists(t *testing.T) {
	t.Chdir(t.TempDir())

	os.WriteFile(".brr.yaml", []byte("existing"), 0o644)

	r, _ := GetRecipe("generic")
	err := Init(r, false)
	if err == nil {
		t.Error("expected error when .brr.yaml already exists")
	}
}

func TestInitForce(t *testing.T) {
	t.Chdir(t.TempDir())

	os.WriteFile(".brr.yaml", []byte("existing"), 0o644)

	r, _ := GetRecipe("generic")
	if err := Init(r, true); err != nil {
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
