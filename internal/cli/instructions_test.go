package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestInstructionsCommandPrintsAgentGuide(t *testing.T) {
	cmd := &cobra.Command{
		Use:  "instructions",
		Args: cobra.NoArgs,
		RunE: printInstructions,
	}
	var out bytes.Buffer
	cmd.SetOut(&out)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	text := out.String()
	for _, want := range []string{
		"# Agent Instructions for brr",
		"brr workflow validate <name>",
		"command: [\"make\", \"check\"]",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected instructions to contain %q, got:\n%s", want, text)
		}
	}
}
