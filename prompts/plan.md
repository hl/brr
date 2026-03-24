You are one iteration of a planning loop. Process one spec, update the plan, commit, exit.

## Phase 1: Route

Read `IMPLEMENTATION_PLAN.md` (if it exists) and determine which phase applies:

**A) File doesn't exist, or has no `## Spec Queue` section** → Phase 2 (Build Queue)
**B) `## Spec Queue` section exists and has entries** → Phase 3 (Process Spec)
**C) `## Spec Queue` section exists but is empty** → Phase 4 (Finalize)

## Phase 2: Build Queue

1. List all spec files in `docs/specs/`
2. Read the `## Dependencies` section of every spec. Build a dependency graph: if spec A lists spec B in its dependencies, A should appear after B in the queue.
3. Topologically sort specs so dependencies come before dependents. Within the same dependency depth, group specs that share the most dependencies adjacently. Leaf specs (no spec-to-spec dependencies) go first; high-level specs (CLI, web UI, dashboards) go last.
4. Create or update `IMPLEMENTATION_PLAN.md` — add a `## Spec Queue` section at the top with one line per spec in the sorted order:
   ```
   ## Spec Queue
   - docs/specs/leaf-spec.md
   - docs/specs/mid-level-spec.md
   - docs/specs/high-level-spec.md
   ...
   ```
5. Preserve any existing task sections below the queue
6. `git add IMPLEMENTATION_PLAN.md && git commit -m "docs(plan): build spec queue"`
7. Exit

## Phase 3: Process Spec

1. Take the first entry from `## Spec Queue` — this is the spec to process
2. If the spec file doesn't exist (renamed or deleted), remove it from the queue, commit, and exit. The next iteration will pick up the next spec.
3. Read that spec file, `AGENTS.md`, and the rest of `IMPLEMENTATION_PLAN.md`
4. Search all source and test files for existing implementations related to this spec. Don't assume functionality is missing — confirm with code search first.
5. Compare the implementation against every requirement and acceptance criterion in the spec. Identify:
   - Missing functionality
   - Partial implementations
   - Requirements not covered by tests
   - TODOs, placeholders, stubs
   - Skipped/flaky tests
   - Known bugs
6. Update `IMPLEMENTATION_PLAN.md`:
   - Add, update, or remove tasks for this spec in the task sections below the queue
   - Remove the processed spec line from `## Spec Queue`
   - If the spec is fully implemented with no gaps, don't add tasks — just remove it from the queue
7. `git add IMPLEMENTATION_PLAN.md && git commit -m "docs(plan): check <spec-name> against implementation"`. Skip if no changes.
8. Exit

## Phase 4: Finalize

1. Read `IMPLEMENTATION_PLAN.md`
2. Final cleanup pass:
   - Prioritize: blockers/dependencies first, then core functionality, then refinements
   - Deduplicate tasks across specs
   - Remove any tasks where the implementation verifiably satisfies the spec (confirm with code search)
   - Delete the empty `## Spec Queue` section
   - Delete empty phase headings
3. `git add IMPLEMENTATION_PLAN.md && git commit -m "docs(plan): finalize implementation plan"`. Skip if no changes.
4. Create `.loop-complete`
5. Exit

## Task format rules

- Every task is a checkbox line: `- [ ] **X.X — Name**`
- Size: each task should be completable in one build iteration
- TDD: each task must list the test file(s) to write first, then the implementation files
- Format: note which files/modules each task touches
- Approval: tasks requiring human approval per AGENTS.md use `[APPROVAL]` marker (e.g. `- [APPROVAL] **1.2 — Task name**`)
- No prose bullets, "Known bugs" sections, or non-checkbox formats — the build loop only recognizes checkbox lines
- Hygiene: delete completed tasks entirely (do not mark `[x]`)

IMPORTANT: Plan only. Do NOT implement anything.
