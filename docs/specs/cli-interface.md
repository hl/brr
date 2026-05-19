# CLI Interface

## Purpose

The CLI interface is the user-facing entry point to brr. It parses commands and flags, wires together configuration and prompt resolution, invokes the engine, and translates engine results into appropriate exit codes and output. It also provides the `init` subcommand for project setup, the `workflow` subcommand for multi-stage pipelines, and the `instructions` subcommand for agent-facing setup guidance.

## Requirements

1. The primary command has these forms: `brr <prompt> [flags]` (run the loop), `brr init [--force]` (scaffold a project), `brr instructions` (print agent setup guidance), `brr workflow run <name> [flags]` (run a multi-stage workflow), `brr workflow validate <name> [flags]`, `brr workflow status [name]`, `brr workflow init <name> --template ship`, and `brr --version` (print version). The prompt is required only for the run form.
2. The `--max` flag sets the maximum number of iterations. Defaults to zero (unlimited). Negative values are rejected with an error.
3. The `--profile` flag selects a named profile from the config. Defaults to the config's default profile when omitted.
4. The `--version` flag prints the version string and exits. It does not require a prompt or a valid config.
5. The `--notify` / `-n` flag enables desktop notifications on loop termination. Off by default.
6. The `init` subcommand scaffolds a new brr project. It accepts a `--force` flag to overwrite existing files.
7. The `workflow` command manages multi-stage pipelines through explicit subcommands. `workflow run` accepts `--profile` (default profile for agent stages), `--notify` (notification on completion), and `--reset` (discard saved progress). `workflow validate` accepts `--profile`. `workflow init` accepts `--template`.
8. The `instructions` subcommand prints Markdown guidance to stdout so users and agents can pipe it into project docs. It does not require project config.
9. On startup, brr prints an ASCII banner followed by a summary of the resolved configuration (profile name, command, max iterations). Runtime CLI output (banner, config summary, status messages) is written to stderr so that stdout is reserved for agent output. The `instructions` subcommand is the explicit exception because its primary output is pipeable documentation.
10. When the engine returns an interrupted stop reason (from Ctrl+C or SIGTERM), brr exits with code 130.
11. When the engine returns any other error, brr exits with code 1.
12. On success, brr exits with code 0.

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
- Depends on `docs/specs/workflow.md` for workflow subcommands.

## Acceptance Criteria

- [ ] `brr <prompt>` runs the loop with the default profile.
- [ ] `--max N` limits iterations to N.
- [ ] `--profile P` selects profile P from the config.
- [ ] `--version` prints the version and exits without requiring a prompt.
- [ ] `--notify` enables desktop notifications on loop termination.
- [ ] Negative `--max` values are rejected with an error.
- [ ] `brr init` delegates to project initialization.
- [ ] `brr instructions` prints agent-facing Markdown setup guidance to stdout without requiring project config.
- [ ] `brr workflow run <name>` delegates to workflow execution.
- [ ] `brr workflow validate <name>` validates without running stages.
- [ ] `brr workflow status [name]` prints saved workflow state.
- [ ] `brr workflow init <name> --template ship` creates a workflow file.
- [ ] Exit code is 130 on Ctrl+C interruption.
- [ ] Exit code is 1 on engine error.
- [ ] Exit code is 0 on success.
- [ ] Banner and config summary are printed to stderr on startup.
- [ ] Runtime CLI output goes to stderr; stdout is reserved for agent output except for `brr instructions`.
- [ ] Colors are suppressed when stderr is not a terminal.
- [ ] All requirements have corresponding tests that pass.
- [ ] Existing tests continue to pass.
