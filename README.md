# brr

Your AI agent, but unhinged.

brr runs a prompt in a loop, spinning up a fresh session for each iteration. Each run gets a clean context, so tasks stay focused and agents stay sharp. Point it at a list, a directory, an API, or anything that produces work — brr just keeps going.

Works with any AI coding agent that accepts prompts on stdin — [Claude Code](https://docs.anthropic.com/en/docs/claude-code), [Codex](https://openai.com/index/codex/), or whatever comes next. Based on [The Ralph Playbook](https://github.com/ClaytonFarr/ralph-playbook).

## Install

```bash
go install github.com/hl/brr/cmd/brr@latest

# or
git clone https://github.com/hl/brr && cd brr && make build
```

## Usage

```bash
# Run a prompt file in a loop
brr prompts/build.md --max 20

# Named prompt (resolves to .brr/prompts/plan.md)
brr plan --max 3

# Inline prompt
brr "Fix all TODO comments in src/" --max 5

# Use a different profile
brr plan --max 10 -p opus

# Scaffold a project
brr init
```

## Configuration

`brr init` generates a `.brr.yaml` with agent profiles:

```yaml
default: claude

profiles:
  claude:
    command: claude
    args: [-p, --dangerously-skip-permissions, --model, sonnet, --max-turns, "200"]

  opus:
    command: claude
    args: [-p, --dangerously-skip-permissions, --model, opus, --max-turns, "200"]

  codex:
    command: codex
    args: [exec, --ephemeral, --dangerously-bypass-approvals-and-sandbox, --model, gpt-5.4, -]
```

Each profile defines a `command` and its `args`. The prompt is piped to stdin. Switch profiles with `-p`:

```bash
brr plan --max 10            # uses default (claude)
brr plan --max 10 -p opus    # uses opus
brr plan --max 10 -p codex   # uses codex
```

Add your own profiles for any agent or configuration you want.

**Priority:** `.brr.yaml` > `<os-config-dir>/brr/config.yaml` (e.g. `~/.config/brr/` on Linux, `~/Library/Application Support/brr/` on macOS, `%AppData%\brr\` on Windows).

## Writing prompts

brr doesn't care what your prompt says — it just runs it over and over. Each iteration gets a fresh context window, so the prompt needs to be self-contained: orient, pick work, do it, commit.

The repo includes [`plan`](prompts/plan.md) and [`build`](prompts/build.md) example prompts. Copy them to `.brr/prompts/` and adapt, or write your own from scratch for anything:

- Triage a backlog of issues
- Run a migration across microservices
- Review and fix lint violations one file at a time
- Process items from a queue or API

A good loop prompt follows this shape:

```markdown
You are one iteration of a loop. Do one unit of work, then exit.

1. Read the state (a file, an API, a queue)
2. Pick one item
3. Do the work
4. Update the state (mark it done, commit, etc.)
5. If nothing left, create `.brr-complete` and exit
```

Put prompts in `.brr/prompts/` (per-project) or `<os-config-dir>/brr/prompts/` (global). Then `brr plan` resolves to `.brr/prompts/plan.md`. Or point at any file: `brr ./my-prompt.md`.

## How it works

brr pipes the same prompt to the configured command, once per iteration. Each run gets a fresh process with a clean context window.

Prompts control the loop by creating signal files:
- `.brr-complete` — all done, stop
- `.brr-needs-approval` — needs a human, stop and print contents

Three consecutive failures stop the loop automatically.

## Safety

Most agent CLIs have a "skip permissions" flag for a reason — brr is designed to use it. But:

- Run in isolated environments (Docker, VM, devcontainer)
- Minimum viable access — only the API keys you need
- Set `--max` to bound iterations
- `Ctrl+C` stops gracefully (1st: finish current iteration; 2nd: interrupt child; 3rd: force kill)

## References

- [The Ralph Playbook](https://github.com/ClaytonFarr/ralph-playbook)
- [Original Ralph Post](https://ghuntley.com/ralph/) by Geoff Huntley
