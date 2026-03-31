package ui

import (
	"os"

	"golang.org/x/term"
)

// ANSI color codes used across the CLI.
// Empty when stdout is not a terminal.
var (
	Bold    = "\033[1m"
	Dim     = "\033[2m"
	Blue    = "\033[34m"
	Cyan    = "\033[36m"
	Magenta = "\033[35m"
	Green   = "\033[32m"
	Yellow  = "\033[33m"
	Red     = "\033[31m"
	Reset   = "\033[0m"
)

func init() {
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		Bold = ""
		Dim = ""
		Blue = ""
		Cyan = ""
		Magenta = ""
		Green = ""
		Yellow = ""
		Red = ""
		Reset = ""
	}
}
