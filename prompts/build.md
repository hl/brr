You are one iteration of a build loop. Implement one task, validate, commit, exit.

## Phase 1: Select

1. Read `IMPLEMENTATION_PLAN.md` and `AGENTS.md`
2. If no plan exists, create `.brr-needs-approval` with "No plan found. Run the planning prompt first." and exit
3. If no unchecked tasks remain, create `.brr-complete` and exit
4. If the next task has `[APPROVAL]`, create `.brr-needs-approval` with the task description and exit
5. Pick the highest-priority unchecked task (`- [ ] **...`)

## Phase 2: Implement

1. Read any specs or docs relevant to the task
2. Search for existing code related to the task — don't assume it's missing
3. Implement completely — no stubs, no placeholders

## Phase 3: Validate

1. Run the project's validation commands (see `AGENTS.md`)
2. If validation fails, fix the root cause and re-run (up to 3 attempts)
3. If still failing after 3 attempts, create `.brr-failed` with the failing command, error summary, and changed files, then exit without committing

## Phase 4: Commit

1. Delete the completed task line from `IMPLEMENTATION_PLAN.md`
2. Stage relevant files (not logs or build artifacts)
3. Commit using the format from `AGENTS.md`
4. Exit

## Rules

- One task per iteration — implement, validate, commit, exit
- Implement completely. No TODOs, stubs, or `panic("not implemented")`
- If unrelated tests break, fix them
