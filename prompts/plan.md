You are one iteration of a planning loop. Process one spec, update the plan, commit, exit.

## Phase 0: Precondition

1. **Recover dirty state.** Run `git status --porcelain`. If there are uncommitted changes to `IMPLEMENTATION_PLAN.md` (a previous iteration crashed between writing and committing), stage and commit them with `docs(plan): recover uncommitted plan changes` before proceeding. If there are other uncommitted changes, `git stash --include-untracked -m "recovered: dirty state from crashed plan iteration"`.

## Phase 1: Route

Read `IMPLEMENTATION_PLAN.md` (if it exists) and determine which phase applies:

**A) File doesn't exist, or has no `## Spec Queue` section and no task sections** → Phase 2 (Build Queue)
**A2) File exists with task sections but no `## Spec Queue` section** → Phase 2 (Build Queue), but the existing tasks MUST be preserved below the new queue
**B) `## Spec Queue` section exists and has entries** → Phase 3 (Process Spec)
**C) `## Spec Queue` section exists but is empty** → Phase 4 (Finalize)

## Phase 2: Build Queue

1. Read the project's `AGENTS.md` and codebase to understand the project scope and goals
2. Identify the inputs to plan against. If `docs/specs/` exists with spec files, use those. Otherwise, identify logical components or features from the codebase and AGENTS.md.
3. Build a dependency graph: if component A depends on component B, A should appear after B in the queue. For spec files, read the `## Dependencies` section of each spec.
4. Topologically sort so dependencies come before dependents. If the graph contains cycles, break them by placing the component with fewer incoming edges first and log which cycle was broken as a comment in the queue (e.g. `<!-- cycle broken: A ↔ B, A placed first -->`). Within the same dependency depth, group components that share the most dependencies adjacently. Leaf components (no dependencies) go first; high-level features (CLI, web UI, dashboards) go last.
5. Create or update `IMPLEMENTATION_PLAN.md` — add a `## Spec Queue` section at the top with one line per component in the sorted order:
   ```
   ## Spec Queue
   - component-a (or docs/specs/component-a.md)
   - component-b
   - component-c
   ...
   ```
6. Preserve any existing task sections below the queue
7. `git add IMPLEMENTATION_PLAN.md && git commit -m "docs(plan): build spec queue"`
8. Exit

## Phase 3: Process Spec

1. Take the first entry from `## Spec Queue` — this is the component to process
2. If the entry references a file that doesn't exist (renamed or deleted), remove it from the queue, commit, and exit. The next iteration will pick up the next entry.
3. Read any associated spec or documentation, `AGENTS.md`, and the rest of `IMPLEMENTATION_PLAN.md`
4. Search all source and test files for existing implementations related to this component. Don't assume functionality is missing — confirm with code search first.
5. Compare the implementation against the component's requirements. Identify:
   - Wrong approach: spec prescribes specific tools, frameworks, or patterns and the code uses something different — this is an implementation gap, not a docs issue
   - Missing functionality
   - Partial implementations
   - Requirements not covered by tests
   - TODOs, placeholders, stubs
   - Skipped/flaky tests
   - Known bugs
6. **Specs are immutable.** If the project has specs in `docs/specs/`, NEVER propose tasks that modify, update, or reconcile them. If code diverges from a spec, the task must change code to match the spec — never the reverse. If a spec appears wrong, flag the task as `[APPROVAL]` so a human can decide.
7. Update `IMPLEMENTATION_PLAN.md`:
   - Add, update, or remove tasks for this component in the task sections below the queue
   - Remove the processed entry from `## Spec Queue`
   - If every requirement and acceptance criterion is satisfied (confirm each one explicitly), don't add tasks — just remove it from the queue
8. `git add IMPLEMENTATION_PLAN.md && git commit -m "docs(plan): check <component-name> against implementation"`. Skip if no changes.
9. Exit

## Phase 4: Finalize

1. Read `IMPLEMENTATION_PLAN.md`
2. Final cleanup pass:
   - Prioritize: blockers/dependencies first, then core functionality, then refinements
   - Deduplicate tasks across components
   - Remove any tasks where the implementation verifiably satisfies the requirements (confirm with code search)
   - Delete the empty `## Spec Queue` section
   - Delete empty phase headings
3. `git add IMPLEMENTATION_PLAN.md && git commit -m "docs(plan): finalize implementation plan"`. Skip if no changes.
4. Create `.brr-complete`
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
IMPORTANT: If the project has specs, they are immutable. NEVER generate tasks that propose modifying, updating, or reconciling spec files. Code conforms to specs — not the other way around.
