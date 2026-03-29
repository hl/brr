# Audit & Plan

You are an autonomous audit pipeline. You will audit the entire codebase, consolidate findings, and produce a prioritized implementation plan.

## Severity Gate

Only include a finding if it meets **all three criteria**:
1. It causes incorrect behavior, data loss, security compromise, crash, a meaningful performance problem (e.g. unnecessary blocking, redundant I/O, O(n²) where O(n) is trivial), **or** the implementation contradicts a spec/requirement that callers depend on
2. It can be triggered through a realistic code path (not theoretical)
3. You can point to the specific line(s) where the bug lives

Everything else — dead code, missing types, doc drift, style, missing test coverage for non-critical paths — is out of scope. The audit's job is to find things that calcify, not things that are obvious when you're in the file.

## Phase 0: Precondition

1. **Recover dirty state.** Run `git status --porcelain`. If there are uncommitted changes to `IMPLEMENTATION_PLAN.md`, stage and commit them with `docs(plan): recover uncommitted audit plan changes` before proceeding. If there are other uncommitted changes, `git stash --include-untracked -m "recovered: dirty state from crashed audit"`.

## Phase 1: Audit (Parallel)

1. Read `AGENTS.md` and any specs or docs in the project to understand the architecture and conventions.
2. Identify the major subsystems in the codebase (e.g. by directory structure, package boundaries, or logical grouping).
3. Spawn a team of parallel agents (subagent_type: general-purpose, model: opus) — one per subsystem. Each agent reads `AGENTS.md` and relevant specs before starting.

Each agent audits its subsystem for:
- **Spec divergence:** behavior that contradicts documented specs or requirements
- **State bugs:** transitions or flows that can leave the system in an unrecoverable or inconsistent state
- **Resource leaks:** file descriptors, connections, processes, or handles not cleaned up on failure paths
- **Injection vectors:** untrusted input reaching shell calls, queries, or command construction
- **Crash paths:** missing or broken error handling that crashes the process
- **Concurrency bugs:** races, deadlocks, or unsafe shared state under concurrent use
- **Auth/security gaps:** bypasses, missing validation, or key material mishandling

Each agent returns a structured report: `{file, line, description, suggested_fix}`.

Wait for all agents to complete before proceeding.

## Phase 2: Consolidate

1. Collect all findings from Phase 1.
2. Deduplicate: merge findings that describe the same issue. Two agents independently finding the same bug is a strong signal — note it.
3. Apply the Severity Gate: drop any finding that does not meet all three criteria.
4. Create `IMPLEMENTATION_PLAN.md` as a prioritized task list:
   - Format: `- [ ] **X.X — Description** — files: list of files`
   - Order by impact: correctness/security bugs first, then resource leaks, then crash-path issues
   - Each task is one atomic, committable unit of work
5. Commit: `docs(plan): audit findings`

## Rules

- Read `AGENTS.md` and `CLAUDE.md` before starting — they define all project constraints.
- If confidence in a finding is below 95%, investigate further. If it still doesn't meet the severity gate, drop it.
- The output of this prompt is `IMPLEMENTATION_PLAN.md` — do not begin fixing anything.
