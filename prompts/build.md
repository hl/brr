You are one iteration of an autonomous build loop. Implement one task, validate, commit, exit.

## Phase 0: Precondition

1. Delete any stale signal files from previous iterations: `.loop-complete`, `.loop-needs-approval`
2. If `IMPLEMENTATION_PLAN.md` does not exist, create `.loop-needs-approval` containing "No IMPLEMENTATION_PLAN.md found. Run the planning prompt first." and exit immediately.

## Phase 1: Orient

Spawn a Task agent (subagent_type: Explore, model: sonnet) to read:
- `docs/specs/*` — all spec files
- `IMPLEMENTATION_PLAN.md` — task list and status
- `AGENTS.md` — validation commands, conventions, and decision authority

Wait for the agent to return before proceeding.

## Phase 2: Select

If all tasks in IMPLEMENTATION_PLAN.md are complete, commit any final plan updates first, then create a file named `.loop-complete` on disk (do NOT stage or commit this file — it is a signal to the loop script, not part of the repo) and exit. Do not continue.

Pick the highest-priority incomplete task. If the task uses the `[APPROVAL]` checkbox marker (literally `- [APPROVAL] **...`), create a file named `.loop-needs-approval` containing the task description, then exit. Do not implement it. Note: only the literal `[APPROVAL]` marker triggers this — prose mentioning "approval" or "requiring approval" in a task description does NOT count.

Spawn a Task agent (subagent_type: Explore, model: sonnet) to search all source and test files for existing code related to this task — don't assume not implemented.

Assess: **can this task be split into independent sub-parts that touch different files?**
- Yes, clearly independent parts → Phase 3B
- No, or parts share files → Phase 3A (default)

Most tasks are 3A. Only use 3B when sub-parts are clearly file-disjoint.

## Phase 3A: Single worker (default)

Spawn one Task agent (subagent_type: general-purpose, model: opus) with:
- The task to implement with relevant context from Phases 1 and 2
- Instruction: implement completely, no stubs. Do NOT commit or push.

## Phase 3B: Parallel workers

Spawn multiple Task agents in a single message (subagent_type: general-purpose, model: opus) — one per independent sub-part, launched in parallel:
- Each agent gets its sub-part with explicit file boundaries and relevant context
- Each agent: implement completely, no stubs. Do NOT commit or push.
- Agents MUST NOT modify the same files. If you can't guarantee file-disjoint work, use 3A.
- After all agents return, verify no file was modified by more than one agent (`git diff --name-only` per agent's scope). If overlap is detected, discard and redo with 3A.

## Phase 4: Validate

Spawn a Task agent (subagent_type: general-purpose, model: opus) with:
- The validation commands from AGENTS.md
- What was just implemented (task description from Phase 2)
- Which files were created or modified (list them explicitly)
- Instruction: run all validation commands. If validation fails, diagnose and fix, then re-run. If still failing after a reasonable attempt, return the failure details.

If validation could not be fixed, discard the broken implementation with `git stash --include-untracked -m "failed: <task description>"`, then document the blocker in IMPLEMENTATION_PLAN.md, commit only the blocker update, and exit. The stash preserves the failed attempt for debugging. The next iteration will get a fresh attempt.

## Phase 5: Simplify (only if validation passed)

Run `/simplify` on the current changes.

## Phase 6: Review (only if validation passed)

Run `/pr-review-toolkit:review-pr` on the current changes (working diff, no PR required).

Only block for: failing tests, uncaught exceptions, security vulnerabilities (injection, auth bypass), data loss scenarios, or logic that contradicts the spec. Everything else is a suggestion — proceed to Phase 7.

If the review finds blocking issues, fix them and re-run validation (Phase 4). Do NOT re-review after fixing — proceed directly to Phase 7. If a blocking issue cannot be fixed in a reasonable attempt, discard the implementation (same as Phase 4 failure: stash, document blocker, commit blocker update, exit). Do NOT commit code with known blocking issues.

## Phase 7: Finalize (only if validation passed)

1. Remove the completed task from IMPLEMENTATION_PLAN.md. Add any learnings relevant to remaining tasks.
2. `git reset HEAD` to clear any pre-existing staged files.
3. `git add` relevant files (not logs or build artifacts). Stage files explicitly by path.
4. `git commit` using the commit format from AGENTS.md.

## Rules (priority order)

- Implement completely. No placeholders or stubs.
- Parallel agents must not write to the same files — if in doubt, use 3A.
- Keep AGENTS.md operational only — status updates belong in IMPLEMENTATION_PLAN.md.
- If unrelated tests fail, resolve them to keep validation passing.
- Keep IMPLEMENTATION_PLAN.md lean — remove completed tasks, git log is the history.
- When you learn something about running the project, update AGENTS.md (keep it brief).
- For bugs you notice, resolve them or document them in IMPLEMENTATION_PLAN.md.
