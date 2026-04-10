# CLI Interface

## Purpose

The CLI interface is the user-facing entry point to brr. It parses commands and flags, wires together configuration and prompt resolution, invokes the engine, and translates engine results into appropriate exit codes and output. It also provides the `init` subcommand for project setup and the `workflow` subcommand for multi-stage pipelines.

## Requirements

1. The primary command has four forms: `brr <prompt> [flags]` (run the loop), `brr init [--force]` (scaffold a project), `brr workflow <name> [flags]` (run a multi-stage workflow), and `brr --version` (print version). The prompt is required only for the run form.
2. The `--max` flag sets the maximum number of iterations. Defaults to zero (unlimited). Negative values are rejected with an error.
3. The `--profile` flag selects a named profile from the config. Defaults to the config's default profile when omitted.
4. The `--version` flag prints the version string and exits. It does not require a prompt or a valid config.
5. The `--notify` / `-n` flag enables desktop notifications on loop termination. Off by default.
6. The `init` subcommand scaffolds a new brr project. It accepts a `--force` flag to overwrite existing files.
7. The `workflow` subcommand runs multi-stage pipelines. It accepts `--profile` (default profile for all stages), `--notify` (notification on completion), and `--reset` (discard saved progress) flags.
8. On startup, brr prints an ASCII banner followed by a summary of the resolved configuration (profile name, command, max iterations). All CLI output (banner, config summary, status messages) is written to stderr so that stdout is reserved for agent output.
9. When the engine returns an interrupted stop reason (from Ctrl+C or SIGTERM), brr exits with code 130.
10. When the engine returns any other error, brr exits with code 1.
11. On success, brr exits with code 0.

## Constraints

- Version and commit hash are injected at build time via linker flags; they must not be hardcoded.
- The CLI must not import or depend on engine internals beyond the `engine.Run()` entry point and lock functions for workflow orchestration.
- Terminal colors are only emitted when stderr is a terminal.

## Dependencies

- Depends on `docs/specs/configuration.md` for profile resolution.
- Depends on `docs/specs/prompt-resolution.md` for prompt interpretation.
- Depends on `docs/specs/loop-engine.md` for iteration execution.
- Depends on `docs/specs/notifications.md` for the `--notify` flag behavior.
- Depends on `docs/specs/project-initialization.md` for the `init` subcommand.
- Depends on `docs/specs/workflow.md` for the `workflow` subcommand.

## Acceptance Criteria

- [ ] `brr <prompt>` runs the loop with the default profile.
- [ ] `--max N` limits iterations to N.
- [ ] `--profile P` selects profile P from the config.
- [ ] `--version` prints the version and exits without requiring a prompt.
- [ ] `--notify` enables desktop notifications on loop termination.
- [ ] Negative `--max` values are rejected with an error.
- [ ] `brr init` delegates to project initialization.
- [ ] `brr workflow <name>` delegates to workflow execution.
- [ ] Exit code is 130 on Ctrl+C interruption.
- [ ] Exit code is 1 on engine error.
- [ ] Exit code is 0 on success.
- [ ] Banner and config summary are printed to stderr on startup.
- [ ] All CLI output goes to stderr; stdout is reserved for agent output.
- [ ] Colors are suppressed when stderr is not a terminal.
- [ ] All requirements have corresponding tests that pass.
- [ ] Existing tests continue to pass.
