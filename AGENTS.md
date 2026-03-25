# Agents

## Validation

Run all quality gates before committing:

```bash
make check
```

This runs, in order: `fmt` → `vet` → `lint` → `test` → `build`. All must pass.

Individual gates:

```bash
make build          # compile
make test           # unit tests
make vet            # go vet
make lint           # golangci-lint (errcheck, staticcheck, unused, ineffassign)
make fmt            # check formatting (fix with: make fmt-fix)
```

## Conventions

- Go 1.26.1 (managed via mise)
- Use `cobra` for CLI, `viper` for config
- All Go commands via `mise exec --` or the Makefile
- Shared constants (colors, signal files) live in `internal/ui`
- Config uses named profiles with `command` + `args`
- Prompt resolution: existing file > `.brr/prompts/<name>.md` > `~/.config/brr/prompts/<name>.md` > inline text
- Commit messages: `type(scope): description` (conventional commits)
- brr is agent-agnostic — profiles in `.brr.yaml` determine what runs

## Decision Authority

- Routine implementation choices: proceed without approval
- New external dependencies beyond cobra/viper: document rationale
- Changes to the spec: require `[APPROVAL]`
