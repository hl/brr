#!/bin/bash
# loop.sh — Run Claude in a loop with the same prompt. Each iteration gets a fresh context window.
#
# Usage: ./loop.sh <prompt> [--max N] [--model NAME] [--turns N] [--effort LEVEL]
#
#   prompt     File path or inline string (required). If a file exists at the path, its contents
#              are piped to Claude. Otherwise the string itself is used as the prompt.
#   --max      Max iterations (default: 0 = unlimited). Use to bound execution.
#   --model    Claude model name (default: sonnet).
#   --turns    Max tool-use turns per iteration (default: 200).
#   --effort   Reasoning effort: low, medium, or high (optional).
#
# Each iteration runs: claude -p --dangerously-skip-permissions --model MODEL --max-turns TURNS
# On failure, the loop continues to the next iteration. Ctrl+C to stop.
#
# Examples:
#   ./loop.sh prompts/build.md --max 20
#   ./loop.sh prompts/build.md --max 20 --model opus
#   ./loop.sh "Fix all TODO comments in src/" --max 5
set -euo pipefail
trap 'rm -f .loop-complete .loop-needs-approval' EXIT

PROMPT="" MAX=0 MODEL=sonnet MAX_TURNS=200 EFFORT="" FAIL_STREAK=0 MAX_FAIL_STREAK=3

while [ $# -gt 0 ]; do
    case "$1" in
        --max)       MAX="$2"; shift 2 ;;
        --model)     MODEL="$2"; shift 2 ;;
        --turns)     MAX_TURNS="$2"; shift 2 ;;
        --effort)    EFFORT="$2"; shift 2 ;;
        -*)          echo "Unknown flag: $1"; exit 1 ;;
        *)           PROMPT="$1"; shift ;;
    esac
done

[ -z "$PROMPT" ] && { echo "Usage: ./loop.sh <prompt> [--max N] [--model NAME] [--turns N] [--effort LEVEL]"; exit 1; }

cat <<'BANNER'

  ╦  ╔═╗╔═╗╔═╗ ┌─┐┬ ┬
  ║  ║ ║║ ║╠═╝ └─┐├─┤
  ╩═╝╚═╝╚═╝╩  o└─┘┴ ┴

BANNER
echo "  prompt: ${PROMPT}"
echo "  model:  ${MODEL} | turns: ${MAX_TURNS} | max: ${MAX:-unlimited}"
echo ""

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
    RC=0
    if [ -f "$PROMPT" ]; then
        claude -p --dangerously-skip-permissions --model "$MODEL" --max-turns "$MAX_TURNS" ${EFFORT_FLAG[@]+"${EFFORT_FLAG[@]}"} < "$PROMPT" || RC=$?
    else
        printf '%s' "$PROMPT" | claude -p --dangerously-skip-permissions --model "$MODEL" --max-turns "$MAX_TURNS" ${EFFORT_FLAG[@]+"${EFFORT_FLAG[@]}"} || RC=$?
    fi
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
