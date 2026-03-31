package ui

import "testing"

func TestColorsEmptyInNonTerminal(t *testing.T) {
	// Test runner stdout is piped (not a terminal), so init() should have
	// set all color codes to empty strings.
	codes := map[string]string{
		"Bold":    Bold,
		"Dim":     Dim,
		"Blue":    Blue,
		"Cyan":    Cyan,
		"Magenta": Magenta,
		"Green":   Green,
		"Yellow":  Yellow,
		"Red":     Red,
		"Reset":   Reset,
	}
	for name, val := range codes {
		if val != "" {
			t.Errorf("expected %s to be empty in non-terminal context, got %q", name, val)
		}
	}
}
