package config

import (
	"os"
	"testing"
)

func TestLoadNoConfig(t *testing.T) {
	t.Chdir(t.TempDir())

	_, err := Load()
	if err == nil {
		t.Error("expected error when no config exists")
	}
}

func TestLoadWithProfiles(t *testing.T) {
	t.Chdir(t.TempDir())

	yaml := `default: myagent
profiles:
  myagent:
    command: myagent
    args: [--fast, --no-confirm]
  other:
    command: other
    args: [--verbose]
`
	os.WriteFile(".brr.yaml", []byte(yaml), 0o644)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Default != "myagent" {
		t.Errorf("expected default 'myagent', got %q", cfg.Default)
	}
	if len(cfg.Profiles) != 2 {
		t.Errorf("expected 2 profiles, got %d", len(cfg.Profiles))
	}

	p := cfg.Profiles["myagent"]
	if p.Command != "myagent" {
		t.Errorf("expected command 'myagent', got %q", p.Command)
	}
	if len(p.Args) != 2 || p.Args[0] != "--fast" {
		t.Errorf("expected args [--fast, --no-confirm], got %v", p.Args)
	}
}

func TestLoadNoProfiles(t *testing.T) {
	t.Chdir(t.TempDir())

	os.WriteFile(".brr.yaml", []byte("default: x\n"), 0o644)

	_, err := Load()
	if err == nil {
		t.Error("expected error when no profiles defined")
	}
}

func TestLoadNoDefault(t *testing.T) {
	t.Chdir(t.TempDir())

	yaml := `profiles:
  claude:
    command: claude
    args: [-p]
`
	os.WriteFile(".brr.yaml", []byte(yaml), 0o644)

	_, err := Load()
	if err == nil {
		t.Error("expected error when no default set")
	}
}

func TestResolveProfileDefault(t *testing.T) {
	cfg := Config{
		Default: "claude",
		Profiles: map[string]Profile{
			"claude": {Command: "claude", Args: []string{"-p", "--model", "sonnet"}},
		},
	}

	cmd, name, err := cfg.ResolveProfile("")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if name != "claude" {
		t.Errorf("expected name 'claude', got %q", name)
	}
	if cmd[0] != "claude" || len(cmd) != 4 {
		t.Errorf("unexpected command: %v", cmd)
	}
}

func TestResolveProfileExplicit(t *testing.T) {
	cfg := Config{
		Default: "claude",
		Profiles: map[string]Profile{
			"claude": {Command: "claude", Args: []string{"-p"}},
			"codex":  {Command: "codex", Args: []string{"exec"}},
		},
	}

	cmd, name, err := cfg.ResolveProfile("codex")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if name != "codex" {
		t.Errorf("expected name 'codex', got %q", name)
	}
	if cmd[0] != "codex" {
		t.Errorf("expected command 'codex', got %q", cmd[0])
	}
}

func TestResolveProfileNotFound(t *testing.T) {
	cfg := Config{
		Default:  "claude",
		Profiles: map[string]Profile{"claude": {Command: "claude"}},
	}

	_, _, err := cfg.ResolveProfile("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent profile")
	}
}

func TestResolveProfileEmptyCommand(t *testing.T) {
	cfg := Config{
		Default:  "broken",
		Profiles: map[string]Profile{"broken": {Command: "", Args: []string{"-p"}}},
	}

	_, _, err := cfg.ResolveProfile("")
	if err == nil {
		t.Error("expected error for empty command")
	}
}
