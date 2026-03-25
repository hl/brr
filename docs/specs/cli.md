# brr CLI Specification

## Overview

`brr` is a Go CLI tool that runs AI coding agents (Claude Code, Codex) in autonomous loops. Each iteration gets a fresh context window. The tool ships with built-in prompt templates (plan, build) and supports user-defined prompts with project-aware templating. The agent backend is configurable per-project.

## Dependencies

None (root spec).

## Commands

### `brr run <prompt> [flags]`

The core command. Runs Claude Code repeatedly with the given prompt.

**Arguments:**
- `prompt` (required) — file path, prompt name, or inline string. Resolution order:
  1. If a file exists at the path, use it
  2. If it matches a known prompt name (built-in or custom), use that
  3. Otherwise treat as inline prompt text

**Flags:**
- `--agent NAME` / `-a` — agent backend: `claude`, `codex` (default: from config, fallback `claude`)
- `--max N` / `-n` — max iterations (default: 0 = unlimited)
- `--model NAME` / `-m` — model name, agent-specific (default: from config; `sonnet` for claude, `gpt-5.4` for codex)
- `--turns N` / `-t` — max tool-use turns per iteration (default: from config, fallback `200`)
- `--effort LEVEL` / `-e` — reasoning effort: `low`, `medium`, `high` (optional)
- `--search` — enable Codex web search (codex only)
- `--profile NAME` — Codex config profile (codex only)

**Behavior:**
1. Load config (see Config section)
2. Resolve agent provider from config/flags
3. Resolve prompt (see Prompt Resolution)
4. If prompt is a template, render it with config values
5. Loop:
   - Check for signal files (`.brr-complete`, `.brr-needs-approval`)
   - Print iteration banner with number, timestamp, and optional max
   - Execute via provider:
     - **Claude:** `claude -p --dangerously-skip-permissions --model MODEL --max-turns TURNS [--effort EFFORT] < prompt`
     - **Codex:** `codex exec --ephemeral --dangerously-bypass-approvals-and-sandbox --model MODEL -c "max_agent_turns=TURNS" [--search] [--profile PROFILE] - < prompt`
   - On failure: increment fail streak. After 3 consecutive failures, stop.
   - On success: reset fail streak.
6. Clean up signal files on exit

### `brr plan [flags]`

Shorthand for `brr run plan [flags]`. Uses the built-in planning prompt.

### `brr build [flags]`

Shorthand for `brr run build [flags]`. Uses the built-in build prompt.

### `brr init [--recipe NAME]`

Scaffolds a project for use with brr.

**Behavior:**
1. If `.brr.yaml` already exists, warn and exit (unless `--force`)
2. Auto-detect project type from marker files (see Recipes)
3. Generate:
   - `.brr.yaml` with recipe defaults
   - `.brr/prompts/plan.md` and `.brr/prompts/build.md` (copies of built-in prompts for customization; preserved on re-init unless `--force`)
   - `AGENTS.md` template (if not exists)
   - `docs/specs/` directory (if not exists)
4. Print what was created

**Recipes:**

| Marker file | Recipe | source_dirs | validation |
|---|---|---|---|
| `mix.exs` | elixir | `lib, test, config, priv` | `mix compile --warnings-as-errors`, `mix test`, `mix format --check-formatted` |
| `package.json` | node | `src, test, tests, lib` | `npm test` |
| `go.mod` | go | `.` | `go build ./...`, `go test ./...`, `go vet ./...` |
| `Cargo.toml` | rust | `src, tests` | `cargo build`, `cargo test`, `cargo clippy` |
| `pyproject.toml` | python | `src, tests, lib` | `pytest` |
| fallback | generic | `src, lib, test, tests` | _(empty)_ |

### `brr prompts list`

Lists available prompts. Shows name, source (built-in/user/project), and one-line description.

### `brr prompts show <name>`

Prints the raw contents of a prompt (before template rendering).

### `brr prompts edit <name>`

Copies a built-in prompt to `~/.config/brr/prompts/<name>.md` for customization. Opens `$EDITOR` if set.

## Config

### File locations

1. Built-in defaults (compiled into binary)
2. `~/.config/brr/config.yaml` — user global defaults
3. `.brr.yaml` — project config (in working directory)
4. CLI flags — highest priority

Each level overrides the previous. Merging is shallow (per top-level key), not deep.

### Schema

```yaml
# Agent backend: claude or codex
agent: claude

# Agent settings (model name is agent-specific)
model: sonnet          # sonnet/opus for claude, gpt-5.4 for codex
max_turns: 200         # Max tool-use turns per iteration
effort: ""             # Reasoning effort (low/medium/high or empty)

# Codex-specific settings (ignored when agent: claude)
codex:
  search: false        # Enable web search
  profile: ""          # Config profile name

# Project structure (used in prompt templates)
spec_dir: docs/specs
plan_file: IMPLEMENTATION_PLAN.md
agents_file: AGENTS.md

# Source directories to search (used in prompt templates)
source_dirs:
  - lib
  - src
  - test

# Validation commands (used in prompt templates)
validation:
  - make test
```

## Prompt System

### Built-in prompts

Embedded in the binary via Go `embed.FS`. Ship with:
- `plan` — planning loop (generates implementation plan from specs)
- `build` — build loop (implements one task per iteration)

### Prompt resolution order

1. If a file exists at the given path, use it directly
2. `.brr/prompts/<name>.md` (project-local override)
3. `~/.config/brr/prompts/<name>.md` (user override)
4. Built-in `prompts/<name>.md`
5. If none match, treat the input as inline prompt text (e.g., `brr run "Fix all TODOs"`)

### Template rendering

Prompts are rendered as Go `text/template` with the following data:

| Variable | Source | Example |
|---|---|---|
| `.Model` | config | `sonnet` |
| `.MaxTurns` | config | `200` |
| `.SpecDir` | config | `docs/specs` |
| `.PlanFile` | config | `IMPLEMENTATION_PLAN.md` |
| `.AgentsFile` | config | `AGENTS.md` |
| `.SourceDirs` | config | `[lib, test, config]` |
| `.Validation` | config | `[mix test, mix compile]` |

Template delimiters: `{{` and `}}` (Go defaults).

If a prompt contains no template directives, it is used as-is (plain prompts work without config).

## Signal Files

The engine checks for these before each iteration:

- `.brr-complete` — all tasks done. Print success message, remove file, stop.
- `.brr-needs-approval` — task needs human review. Print file contents, remove file, stop.

Both files are cleaned up on process exit (trap).

## Output

### Banner

On startup, print:
- ASCII art logo
- Config summary: prompt, model, turns, max

### Iteration header

Before each iteration:
```
━━━ Iteration 1/20 ▸ 14:32:01 ━━━
```

### Failure output
```
✗ Iteration 3 failed (exit 1). Consecutive failures: 2/3
```

### Signal output
```
✓ All tasks complete (.brr-complete found). Stopping.
⏸ Task needs human approval (.brr-needs-approval found):
<file contents>
```

Use ANSI colors consistent with current styling.

## Acceptance Criteria

1. `brr run <file>` executes Claude in a loop identically to the original shell script behavior
2. `brr run <prompt-name>` resolves and renders built-in prompts with config
3. `brr plan` and `brr build` work as shorthands
4. `brr init` detects project type and generates correct scaffolding
5. `brr prompts list` shows all available prompts with source
6. `brr prompts edit <name>` creates user override and opens editor
7. Config cascade works: flags > .brr.yaml > ~/.config > defaults
8. Template rendering substitutes all config variables correctly
9. Signal files (.brr-complete, .brr-needs-approval) work identically to the original shell script
10. Fail streak logic (3 consecutive failures) works identically to the original shell script
11. ANSI colors render correctly; no color when stdout is not a terminal
12. Binary builds for linux/darwin, amd64/arm64
