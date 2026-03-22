#!/bin/bash
# loop.sh — Run Claude in a loop with the same prompt. Each iteration gets a fresh context window.
#
# Usage: ./loop.sh <prompt> [--max N] [--model NAME] [--turns N] [--effort LEVEL] [--quiet]
#
#   prompt     File path or inline string (required). If a file exists at the path, its contents
#              are piped to Claude. Otherwise the string itself is used as the prompt.
#   --max      Max iterations (default: 0 = unlimited). Use to bound execution.
#   --model    Claude model name (default: sonnet).
#   --turns    Max tool-use turns per iteration (default: 200).
#   --effort   Reasoning effort: low, medium, or high (optional).
#   --quiet    Suppress real-time progress output (only show iteration banners).
#
# Each iteration runs: claude -p --dangerously-skip-permissions --model MODEL --max-turns TURNS
# By default, streams real-time progress (tool calls, text output, cost summary).
# On failure, the loop continues to the next iteration. Ctrl+C to stop.
#
# Examples:
#   ./loop.sh prompts/build.md --max 20
#   ./loop.sh prompts/build.md --max 20 --model opus
#   ./loop.sh "Fix all TODO comments in src/" --max 5
#   ./loop.sh prompts/build.md --max 20 --quiet
set -euo pipefail
trap 'rm -f .loop-complete .loop-needs-approval' EXIT

PROMPT="" MAX=0 MODEL=sonnet MAX_TURNS=200 EFFORT="" FAIL_STREAK=0 MAX_FAIL_STREAK=3 QUIET=0

while [ $# -gt 0 ]; do
    case "$1" in
        --max)       MAX="$2"; shift 2 ;;
        --model)     MODEL="$2"; shift 2 ;;
        --turns)     MAX_TURNS="$2"; shift 2 ;;
        --effort)    EFFORT="$2"; shift 2 ;;
        --quiet)     QUIET=1; shift ;;
        -*)          echo "Unknown flag: $1"; exit 1 ;;
        *)           PROMPT="$1"; shift ;;
    esac
done

[ -z "$PROMPT" ] && { echo "Usage: ./loop.sh <prompt> [--max N] [--model NAME] [--turns N] [--effort LEVEL] [--quiet]"; exit 1; }

# Parse stream-json events into concise progress lines
progress_filter() {
    python3 -u -c '
import sys, json
tool_count = 0
for line in sys.stdin:
    line = line.strip()
    if not line:
        continue
    try:
        event = json.loads(line)
    except (json.JSONDecodeError, ValueError):
        continue
    t = event.get("type", "")
    if t == "assistant":
        msg = event.get("message", event)
        content = msg.get("content", [])
        if not isinstance(content, list):
            continue
        for block in content:
            if not isinstance(block, dict):
                continue
            bt = block.get("type", "")
            if bt == "tool_use":
                name = block.get("name", "?")
                inp = block.get("input", {})
                tool_count += 1
                if name == "Agent":
                    desc = inp.get("description", inp.get("prompt", "")[:60])
                    print(f"\r\033[K  \u25b8 Agent: {desc}", flush=True)
                elif name in ("Bash", "Write"):
                    detail = inp.get("command", inp.get("file_path", ""))[:80]
                    print(f"\r\033[K  \u25b8 {name}: {detail}", flush=True)
                elif name == "Skill":
                    skill_name = inp.get("skill", "?")
                    print(f"\r\033[K  \u25b8 Skill: {skill_name}", flush=True)
                else:
                    print(f"\r\033[K  \u25cb {tool_count} tool calls\u2026", end="", flush=True)
            elif bt == "text":
                text = block.get("text", "").strip()
                if len(text) > 10:
                    first = text.split("\n")[0].strip()[:120]
                    if first:
                        print(f"\r\033[K  \u2502 {first}", flush=True)
    elif t == "result":
        print("\r\033[K", end="", flush=True)
        cost = event.get("total_cost_usd")
        turns = event.get("num_turns")
        duration = event.get("duration_ms")
        parts = []
        if turns is not None:
            parts.append(f"{turns} turns")
        if duration is not None:
            parts.append(f"{duration / 60000:.1f}m")
        if cost is not None:
            parts.append(f"${cost:.2f}")
        if parts:
            summary = ", ".join(parts)
            print(f"  \u2713 Done: {summary}", flush=True)
' 2>/dev/null || true
}

I=0
while [ "$MAX" -eq 0 ] || [ "$I" -lt "$MAX" ]; do
    if [ -f .loop-complete ]; then
        echo "All tasks complete (.loop-complete found). Stopping."
        rm -f .loop-complete
        break
    fi
    if [ -f .loop-needs-approval ]; then
        echo "Task needs human approval (.loop-needs-approval found):"
        cat .loop-needs-approval
        rm -f .loop-needs-approval
        break
    fi
    EFFORT_FLAG=()
    [ -n "$EFFORT" ] && EFFORT_FLAG=(--effort "$EFFORT")
    ITER_NUM=$((I + 1))
    MAX_LABEL=""
    [ "$MAX" -gt 0 ] && MAX_LABEL="/$MAX"
    echo ""
    echo "━━━ Iteration ${ITER_NUM}${MAX_LABEL} ▸ $(date '+%H:%M:%S') ━━━"
    CLAUDE_FLAGS=(--dangerously-skip-permissions --model "$MODEL" --max-turns "$MAX_TURNS" ${EFFORT_FLAG[@]+"${EFFORT_FLAG[@]}"})
    if [ "$QUIET" -eq 0 ]; then
        CLAUDE_FLAGS+=(--verbose --output-format stream-json)
    fi
    RC=0
    set +o pipefail
    if [ -f "$PROMPT" ]; then
        if [ "$QUIET" -eq 0 ]; then
            claude -p "${CLAUDE_FLAGS[@]}" < "$PROMPT" 2>/dev/null | progress_filter
            RC=${PIPESTATUS[0]}
        else
            claude -p "${CLAUDE_FLAGS[@]}" < "$PROMPT" || RC=$?
        fi
    else
        if [ "$QUIET" -eq 0 ]; then
            printf '%s' "$PROMPT" | claude -p "${CLAUDE_FLAGS[@]}" 2>/dev/null | progress_filter
            RC=${PIPESTATUS[1]}
        else
            printf '%s' "$PROMPT" | claude -p "${CLAUDE_FLAGS[@]}" || RC=$?
        fi
    fi
    set -o pipefail
    if [ "$RC" -ne 0 ]; then
        FAIL_STREAK=$((FAIL_STREAK + 1))
        echo "Iteration $ITER_NUM failed (exit $RC). Consecutive failures: $FAIL_STREAK/$MAX_FAIL_STREAK"
        if [ "$FAIL_STREAK" -ge "$MAX_FAIL_STREAK" ]; then
            echo "Too many consecutive failures. Stopping."
            exit 1
        fi
    else
        FAIL_STREAK=0
    fi
    I=$((I + 1))
done
