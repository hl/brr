You are one iteration of a verification loop. Check the implementation against spec acceptance criteria.

## Phase 1: Gather

1. Read `AGENTS.md` for the project's validation commands
2. Read every `docs/specs/*.md` — collect all unchecked (`- [ ]`) acceptance criteria
3. If no specs exist or all criteria are already checked, create `.brr-complete` and exit

## Phase 2: Validate

1. Run the project's full validation suite (e.g. `make check`) — if this fails, stop here and report
2. For each unchecked acceptance criterion:
   - Run the relevant test, command, or assertion
   - Check the result against the expected behavior
   - Record pass or fail with evidence

## Phase 3: Report

- **All pass:** Create `.brr-complete`
- **Any fail:** Write failing criteria as tasks in `IMPLEMENTATION_PLAN.md`:
  ```
  - [ ] **V.1 — Criterion description** — files: relevant/path.go
  ```
  If `IMPLEMENTATION_PLAN.md` already has tasks, append — don't overwrite.
  Commit: `docs(plan): verification findings`
  Create `.brr-cycle`

## Rules

- Only report genuine failures — not style, naming, or docs
- Each task must describe the expected behavior and what actually happens
- Run `make check` (or equivalent) before anything else — if that fails, everything fails
- One iteration: gather, validate, report, exit
