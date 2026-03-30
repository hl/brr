You are one iteration of a planning loop. Read the project, update the plan, commit, exit.

## Route

Read `IMPLEMENTATION_PLAN.md` (if it exists) and `AGENTS.md`. Determine which phase applies:

**A) No plan exists** → Phase 1 (Build Plan)
**B) Plan exists with unchecked tasks** → Phase 2 (Refine)
**C) All tasks are done** → Phase 3 (Finalize)

## Phase 1: Build Plan

1. Read the codebase, `AGENTS.md`, and any specs/docs to understand the project
2. Identify what needs to be built and in what order (dependencies first)
3. Create `IMPLEMENTATION_PLAN.md` with prioritized tasks:
   ```
   - [ ] **1.1 — Task name** — files: path/to/file.go
   - [ ] **1.2 — Next task** — files: path/to/other.go
   ```
4. Each task should be one committable unit of work
5. `git add IMPLEMENTATION_PLAN.md && git commit -m "docs(plan): create implementation plan"`
6. Exit

## Phase 2: Refine

1. Check the remaining tasks against the current codebase
2. Remove tasks that are already implemented (search the code to confirm)
3. Add tasks for anything that was missed
4. Commit changes to `IMPLEMENTATION_PLAN.md` if any, then exit

## Phase 3: Finalize

1. Clean up `IMPLEMENTATION_PLAN.md` — remove empty sections
2. Create `.brr-complete` and exit

## Rules

- Plan only — do NOT implement anything
- Delete completed tasks entirely (do not mark `[x]`)
- Tasks requiring human approval: `- [APPROVAL] **1.2 — Task name**`
