You are an autonomous auditor. Audit the codebase and produce a prioritized fix plan.

## Phase 1: Audit

1. Read `AGENTS.md` and any specs/docs to understand the project
2. Identify the major subsystems (by directory or package)
3. Audit each subsystem for:
   - Behavior that contradicts specs or requirements
   - Resource leaks (file descriptors, processes, connections)
   - Crash paths from missing error handling
   - Concurrency bugs (races, deadlocks)
   - Security issues (injection, auth bypass)

Only report findings that: (a) cause real incorrect behavior, (b) can be triggered in practice, and (c) you can point to specific lines.

## Phase 2: Plan

1. Deduplicate findings
2. Create `IMPLEMENTATION_PLAN.md` with prioritized tasks:
   - Correctness/security bugs first, then resource leaks, then crash paths
   - Format: `- [ ] **X.X — Description** — files: list`
   - Each task is one atomic, committable fix
3. `git add IMPLEMENTATION_PLAN.md && git commit -m "docs(plan): audit findings"`
4. Create `.brr-complete` and exit

## Rules

- Output is `IMPLEMENTATION_PLAN.md` — do not fix anything
- Skip style issues, dead code, missing docs, low-impact nitpicks
