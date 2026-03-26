package main

import "github.com/hl/brr/internal/cli"

// Set by goreleaser ldflags.
var (
	version = "dev"
	commit  = "unknown"
)

func main() {
	cli.SetVersion(version, commit)
	cli.Execute()
}
