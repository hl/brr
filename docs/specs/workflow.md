# Workflow

## Purpose

The workflow command orchestrates multi-stage brr pipelines. Each stage runs the loop engine with a specific prompt and iteration limit. Stages execute sequentially and coordinate through shared state files (primarily `IMPLEMENTATION_PLAN.md`). After the final stage, the workflow can cycle back to a designated stage if unfinished tasks remain, enabling autonomous build-verify-review loops.

## Requirements

1. The command is `brr workflow <name> [flags]`. The name argument is required.
2. Workflow files are YAML. Resolution follows the same pattern as prompts: `.brr/workflows/<name>.yaml` first, then `<user-config-dir>/brr/workflows/<name>.yaml`. The first match wins. If no file is found, the command returns an error listing both searched paths.
3. The YAML schema has two top-level keys: `stages` (required, list) and `max_cycles` (optional, integer, default 3). Each stage has `prompt` (required, string), `max` (required, positive integer), `profile` (optional, string), and `cycle` (optional, boolean, default false).
4. Validation rules: at least one stage must be defined; each stage must have a non-empty `prompt` and a `max` greater than zero; at most one stage may set `cycle: true`; `max_cycles` must be greater than zero.
5. Profile resolution per stage: if the stage specifies `profile`, use it; otherwise use the config's default profile. Profile resolution uses the same `config.ResolveProfile` function as the main command.
6. Prompt resolution per stage uses the same `resolvePrompt` function as the main command.
7. The workflow acquires the exclusive lock (`.brr.lock`) once before the first stage and holds it until the workflow exits. Individual engine runs skip lock acquisition.
8. Stages execute sequentially. For each stage, the workflow resolves the prompt and profile, prints a stage header, then calls the engine. The engine's `SkipLock` option must be true.
9. Stage header output includes: the stage number (1-indexed), total stage count, prompt name, profile name, max iterations, and cycle number if cycling.
10. After each stage, the workflow inspects the engine result:
    - `ReasonComplete`: continue to the next stage.
    - `ReasonMaxIterations` with nil error: continue to the next stage.
    - `ReasonMaxIterations` with error (last iteration failed): stop the workflow with an error.
    - `ReasonApproval`: stop the workflow, print the approval content, exit with code 0.
    - `ReasonFailStreak`: stop the workflow with an error.
    - `ReasonInterrupted`: stop the workflow, exit with code 130.
11. After the final stage completes successfully, the workflow checks whether `IMPLEMENTATION_PLAN.md` exists and contains at least one unchecked task (a line matching `- [ ] `). If tasks remain and a stage with `cycle: true` exists and the cycle count has not reached `max_cycles`, the workflow increments the cycle counter and restarts from the cycle stage. Otherwise, the workflow exits successfully.
12. The cycle check uses a simple string search for `- [ ] ` (with the trailing space) in `IMPLEMENTATION_PLAN.md`. If the file does not exist or cannot be read, no tasks remain.
13. On startup, the workflow prints the banner, then a workflow summary: name, number of stages, max cycles, and the list of stages with their prompts and limits. All workflow output goes to stderr.
14. The `--profile` flag on the workflow command sets a default profile for all stages. Per-stage `profile` in the YAML overrides this. If neither is set, the config default applies.
15. The `--notify` flag sends a desktop notification when the workflow completes (not per-stage).
16. Exit codes: 0 for success or approval pause, 1 for errors, 130 for interrupt.
17. The workflow persists progress to `.brr-workflow-state.json` after each stage completes. The state file contains the workflow name, the index of the next stage to run, the current cycle count, and the git HEAD SHA at workflow start.
18. On startup, if `.brr-workflow-state.json` exists and its `workflow` field matches the current workflow name, the workflow resumes from the saved stage and cycle. If the saved stage index is out of bounds (e.g., the workflow YAML was modified), the workflow starts fresh.
19. When the workflow completes successfully (all stages done, no more cycles), the state file is deleted.
20. On error, approval pause, or interrupt, the state file is preserved so the next run resumes from the last checkpoint.
21. The `--reset` flag deletes the state file before starting, forcing a fresh run from stage 1.
22. The state file's `start_sha` field records `git rev-parse HEAD` at the time the workflow first starts (not on resume). This allows prompts (e.g., the review prompt) to determine the diff base.

## Constraints

- No new external dependencies. YAML parsing uses `go.yaml.in/yaml/v3` already available via viper.
- The workflow must work on Linux, macOS, and Windows.
- Workflow resolution must reject path traversal (`..`) in the name argument, consistent with prompt resolution.
- The workflow must not modify `IMPLEMENTATION_PLAN.md` itself. Only the agent (via prompts) modifies shared state files.

## Dependencies

- Depends on `docs/specs/loop-engine.md` for stage execution (engine.Run).
- Depends on `docs/specs/concurrent-run-prevention.md` for exclusive locking.
- Depends on `docs/specs/prompt-resolution.md` for per-stage prompt resolution.
- Depends on `docs/specs/configuration.md` for profile resolution.
- Depends on `docs/specs/notifications.md` for the `--notify` flag behavior.
- Depends on `docs/specs/cli-interface.md` for exit code conventions.

## Acceptance Criteria

- [ ] `brr workflow ship` loads and executes `.brr/workflows/ship.yaml`.
- [ ] Workflow file resolution searches project then user config directory.
- [ ] Missing workflow file returns an error listing both searched paths.
- [ ] Invalid YAML (missing stages, zero max, multiple cycle stages) is rejected with a clear error.
- [ ] Stages execute sequentially — each stage's engine run completes before the next starts.
- [ ] Per-stage profile overrides the default; absent profile falls back to config default.
- [ ] The lock is held for the entire workflow duration, not per-stage.
- [ ] Engine runs use `SkipLock: true`.
- [ ] `ReasonApproval` stops the workflow and exits 0.
- [ ] `ReasonFailStreak` stops the workflow and exits 1.
- [ ] `ReasonInterrupted` stops the workflow and exits 130.
- [ ] After the last stage, the workflow cycles back if tasks remain and cycles are available.
- [ ] Cycling stops when `max_cycles` is reached, even if tasks remain.
- [ ] Cycling stops when no unchecked tasks remain in `IMPLEMENTATION_PLAN.md`.
- [ ] Stage headers and workflow summary are printed to stderr.
- [ ] `--notify` sends a notification on workflow completion.
- [ ] Path traversal (`..`) in the workflow name is rejected.
- [ ] State file is written after each stage completes.
- [ ] Resuming from state file skips already-completed stages.
- [ ] State file with mismatched workflow name is ignored (starts fresh).
- [ ] State file with out-of-bounds stage index causes a fresh start.
- [ ] `--reset` deletes the state file and starts from stage 1.
- [ ] State file is deleted on successful workflow completion.
- [ ] State file is preserved on approval pause and interrupt.
- [ ] `start_sha` is set on first run and preserved across resumes.
- [ ] All requirements have corresponding tests that pass.
- [ ] Existing tests continue to pass.
