# Workflow

## Purpose

The workflow command orchestrates versioned YAML pipelines. A workflow is a sequence of named stages. Agent stages run the loop engine with a prompt, while command stages run deterministic checks or gates. Workflow state and event history are persisted under `.brr/state/workflows/` so runs can be resumed, inspected, and debugged.

## Requirements

1. Workflow execution is invoked with `brr workflow run <name> [flags]`.
2. Workflow validation is invoked with `brr workflow validate <name> [flags]`.
3. Workflow status is invoked with `brr workflow status [name]`; `--watch` redraws a named workflow's saved state until the state file is cleared.
4. Workflow template creation is invoked with `brr workflow init <name> --template ship`.
5. Workflow files are YAML. Resolution searches `.brr/workflows/<name>.yaml`, then `<user-config-dir>/brr/workflows/<name>.yaml`. The first match wins.
6. Workflow names must be non-empty and must not include path separators or `..`.
7. Workflow files must use schema `version: 2`. Unversioned or older workflow files are rejected with a clear migration-style error.
8. The V2 schema has top-level keys: `version` (required, integer `2`), `description` (optional string), `defaults` (optional map), `cycle` (optional map), and `stages` (required list).
9. `defaults.profile` sets the default profile for agent stages. `defaults.max` sets the default max iteration count for agent stages and must be non-negative.
10. `cycle.target` names the stage to restart from when any stage produces `.brr-cycle`. `cycle.max` is required when `cycle` is set and must be positive.
11. Each stage must have a unique `id` and a `type` of `agent` or `command`.
12. Agent stages require `prompt`, may set `profile`, and may set positive `max`. Effective profile resolution is stage profile, then `--profile`, then workflow default profile, then config default. Effective max is stage max, then `defaults.max`.
13. Command stages require `command`, a non-empty argv array. Command stages do not accept `prompt`, `profile`, or `max`.
14. `brr workflow validate` resolves the workflow YAML, validates schema, validates profile references, resolves prompts, and checks command executables without running any stage.
15. `brr workflow run` acquires the exclusive lock once before the first stage and holds it until the workflow exits. Agent stage engine runs use `SkipLock: true`.
16. Stages execute sequentially. Each stage must complete before the next stage starts.
17. Command stages run the argv array directly without shell expansion and inherit stdin, stdout, and stderr.
18. `brr workflow run` prints a compact visual flow of stages and their current states as the workflow starts, advances, cycles, and finishes stages.
19. After every stage, the workflow records state in `.brr/state/workflows/<name>.json` and appends an event to `.brr/state/workflows/<name>.events.jsonl`.
20. The state file contains `schema_version`, `workflow`, `run_id`, `started_at`, `updated_at`, `start_sha`, `next_stage_id`, `cycle_count`, and per-stage status entries with status, reason, duration, prompt/profile/command metadata.
21. On startup, if the state file is valid for the current workflow, the run resumes from `next_stage_id`. Invalid state is ignored and a fresh run starts.
22. `--reset` deletes the workflow state file before starting. Event history is preserved.
23. On successful completion, the workflow state file is deleted. Event history remains.
24. On error, approval pause, failure signal, or interrupt, the workflow state file is preserved so the next run can resume.
25. If `.brr-cycle` is detected and `cycle` is configured with remaining cycles, the workflow increments `cycle_count` and restarts from `cycle.target`.
26. If `.brr-cycle` is detected without a cycle config, or after `cycle.max` is reached, the workflow exits with an error.
27. Agent-reported `.brr-failed` and `.brr-needs-approval` stop the workflow with exit code 0 and preserve state.
28. Command stage non-zero exits stop the workflow with exit code 1 and preserve state.
29. Interrupts stop the workflow with exit code 130 and preserve state.
30. `--notify` sends a desktop notification when the workflow completes successfully, stops due to `.brr-failed`, or stops due to error. Approval pauses and interrupts do not trigger notifications.
31. `brr init` creates `.brr/prompts/`, `.brr/workflows/`, `.brr/state/`, and gitignore entries for workflow runtime state.

## Constraints

- No legacy workflow execution path is retained.
- No shell string command stages. Command stages always use argv arrays.
- Workflow resolution and state persistence must not follow symlinks.
- The workflow must not inspect or modify prompt-owned task state such as `IMPLEMENTATION_PLAN.md`.
- Workflow execution must work on Linux, macOS, and Windows.

## Dependencies

- Depends on `docs/specs/loop-engine.md` for agent stage execution.
- Depends on `docs/specs/concurrent-run-prevention.md` for exclusive locking.
- Depends on `docs/specs/prompt-resolution.md` for per-stage prompt resolution.
- Depends on `docs/specs/configuration.md` for profile resolution.
- Depends on `docs/specs/notifications.md` for notification behavior.
- Depends on `docs/specs/cli-interface.md` for exit code conventions.

## Acceptance Criteria

- [ ] V2 workflows with agent and command stages load and validate.
- [ ] Legacy unversioned workflows are rejected with a migration-style error.
- [ ] Duplicate stage IDs, unknown stage types, invalid cycle targets, missing prompts, and empty command arrays are rejected.
- [ ] `brr workflow validate <name>` checks schema, profiles, prompts, and command executables without executing stages.
- [ ] `brr workflow run <name>` executes stages sequentially.
- [ ] `brr workflow run <name>` prints a visual stage flow with current status markers during execution.
- [ ] Command stages run argv directly, inherit stdio, and stop the workflow on non-zero exit.
- [ ] Agent stages use prompt resolution, profile resolution, effective max, and `SkipLock: true`.
- [ ] `.brr-cycle` restarts from `cycle.target` until `cycle.max` is reached.
- [ ] `.brr-cycle` errors when no cycle config exists or cycle max is reached.
- [ ] State is written to `.brr/state/workflows/<name>.json` and event logs to `.brr/state/workflows/<name>.events.jsonl`.
- [ ] Successful completion deletes only the state file and preserves event history.
- [ ] Failure, approval, error, and interrupt preserve state.
- [ ] Resume starts from the saved `next_stage_id`.
- [ ] `--reset` discards saved state and starts from the first stage.
- [ ] `brr workflow status [name]` prints saved state as a readable stage-by-stage view or a clear no-state message.
- [ ] `brr workflow status <name> --watch` redraws saved state with a running-stage animation.
- [ ] `brr workflow init <name> --template ship` creates a valid V2 workflow file.
- [ ] `brr init` creates `.brr/state/` and gitignores `.brr/state/`.
- [ ] State and event writes reject symlinks and other non-regular files.
- [ ] All requirements have corresponding tests that pass.
- [ ] Existing tests continue to pass.
