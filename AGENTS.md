# Agents

## Workflow

1. Read relevant specs/docs before implementing
2. Implement — one commit per logical unit of working, testable code
3. Run the project's full CI/test suite: `make check`
4. Fix any failures at the root cause
5. Commit: `type(scope): description` (feat, fix, refactor, perf, test, docs, chore)

## Quality Gates

Run all quality gates before committing:

```bash
make check
```

This runs, in order: `fmt` → `vet` → `lint` → `test` → `build`. All must pass.

If any gate fails, fix the root cause. Never suppress, skip, or work around a failure.

Individual gates:

```bash
make build          # compile
make test           # unit tests
make vet            # go vet
make lint           # golangci-lint (errcheck, staticcheck, unused, ineffassign)
make fmt            # check formatting (fix with: make fmt-fix)
```

## Code Standards

- No `// TODO`, `// FIXME`, stubs, or `panic("not implemented")` in committed code
- Commit format: `type(scope): description`
- CHANGELOG: update for all user-visible changes; group by Added/Changed/Fixed/Removed
- Keep files small enough for agent workflows. Prefer files under 400 lines; split by responsibility before a file reaches 500 lines. Tests may exceed this only when a cohesive fixture/table would become harder to follow if split.

## Conventions

- Go 1.26.1 (managed via mise)
- Use `cobra` for CLI, `viper` for config
- All Go commands via `mise exec --` or the Makefile
- Shared constants (colors, signal files) live in `internal/ui`
- Config uses named profiles with `command` + `args`
- Prompt resolution: existing file > `.brr/prompts/<name>.md` > `~/.config/brr/prompts/<name>.md` > inline text
- brr is agent-agnostic — profiles in `.brr.yaml` determine what runs

## Creating brr Assets

- Use `brr instructions` when creating prompts, workflows, or project-local brr config from an installed binary.
- Use `docs/agent-instructions.md` for the same guidance while working in this repository.
- Put reusable prompts in `.brr/prompts/<name>.md`.
- Put Workflow V2 files in `.brr/workflows/<name>.yaml`.
- Run `brr workflow validate <name>` after creating or changing a workflow.
- Do not commit runtime state from `.brr/state/` or signal files.

## Decision Authority

- Routine implementation choices: proceed without approval
- New external dependencies beyond cobra/viper: document rationale
- Changes to the spec: require `[APPROVAL]`
