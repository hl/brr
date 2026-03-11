# Loop

Autonomous AI coding loop. Runs Claude repeatedly with a prompt — each iteration gets a fresh context window.

Based on [The Ralph Playbook](https://github.com/ClaytonFarr/ralph-playbook).

## Usage

```bash
./loop.sh <prompt> [--max N] [--model NAME] [--turns N] [--effort LEVEL]
```

The prompt is a file path or an inline string. Flags can go in any order.

| Flag | Default | Description |
| ---- | ------- | ----------- |
| `--max` | `0` (unlimited) | Max loop iterations |
| `--model` | `sonnet` | Claude model |
| `--turns` | `200` | Tool-use rounds per iteration |
| `--effort` | _(none)_ | Reasoning effort: `low`, `medium`, or `high` |

## Examples

### Plan then build

```bash
# Plan: generate an implementation plan from specs
./loop.sh prompts/plan.md --max 3

# Build: implement one task per iteration
./loop.sh prompts/build.md --max 20

# Build with a stronger model
./loop.sh prompts/build.md --max 20 --model opus
```

### Quick one-off

```bash
./loop.sh "Fix all TODO comments in src/" --max 5
./loop.sh "Run the tests, fix anything broken" --max 10
```

## How It Works

The script loops, piping the same prompt to `claude -p --dangerously-skip-permissions` each iteration. If an iteration fails, the loop continues — but 3 consecutive failures stop the loop to avoid burning iterations on a persistent error.

Each iteration gets a fresh context window — no degradation over long sessions. All behavior (what to implement, when to commit, when to stop) lives in the prompt, not the script.

Prompts can signal the loop to stop by creating files:
- `.loop-complete` — all tasks done, stop the loop
- `.loop-needs-approval` — a task needs human review, stop and print the contents

These signal files are cleaned up automatically on exit.

## Safety

`--dangerously-skip-permissions` bypasses Claude's permission system.

- Run in isolated environments (Docker, VM)
- Minimum viable access (only required API keys)
- Set a max iteration count to avoid runaway loops
- `Ctrl+C` stops the loop

## References

- [The Ralph Playbook](https://github.com/ClaytonFarr/ralph-playbook)
- [Original Ralph Post](https://ghuntley.com/ralph/) by Geoff Huntley
