# Agent Instructions for brr

Use this guide when you are an agent setting up brr in a project. Prefer creating project-local files so the workflow is versioned with the codebase and repeatable for other agents.

## Setup Checklist

1. Run `brr init` if `.brr.yaml` or `.brr/` does not exist.
2. Add or update `.brr/prompts/<name>.md` for each reusable agent task.
3. Add or update `.brr/workflows/<name>.yaml` for multi-stage orchestration.
4. Run `brr workflow validate <name>` after creating or editing a workflow.
5. Keep runtime files out of commits: `.brr/state/`, `.brr-complete`, `.brr-failed`, `.brr-needs-approval`, `.brr-cycle`, and `.brr.lock`.

## Config

Project config lives in `.brr.yaml`. Use named profiles so workflows can choose the right agent without changing commands inline.

```yaml
default: claude

profiles:
  claude:
    command: claude
    args: [-p, --dangerously-skip-permissions, --model, sonnet, --max-turns, "200"]

  codex:
    command: codex
    args: [exec, --ephemeral, --dangerously-bypass-approvals-and-sandbox, --model, gpt-5.4, -]
```

Rules:

- Keep profile names stable; workflows refer to them by name.
- Put command flags in `args`, not in prompts.
- Use project-local `.brr.yaml` when the workflow depends on project-specific tooling.

## Prompts

Prompts live in `.brr/prompts/<name>.md`. A brr prompt should be self-contained because each iteration starts a fresh agent session.

```markdown
You are one iteration of a brr loop. Do exactly one unit of work, then exit.

1. Read the project instructions and relevant specs.
2. Select one concrete task.
3. Implement it completely.
4. Run the strongest practical validation.
5. Commit the completed work if this project expects commits.
6. If no work remains, create `.brr-complete`.
7. If blocked by a human decision, create `.brr-needs-approval` with the question.
8. If blocked by an unrecoverable failure, create `.brr-failed` with the command, error, and changed files.
```

Prompt guidelines:

- Make the unit of work explicit: one issue, one checklist item, one failing test, or one file group.
- Tell the agent where state lives: an issue tracker, `IMPLEMENTATION_PLAN.md`, a queue file, or a directory.
- Define done conditions and validation commands.
- Use `.brr-cycle` only inside workflows when a later stage discovers more work and should restart from `cycle.target`.

## Workflows

Workflows live in `.brr/workflows/<name>.yaml`. Use Workflow V2 and explicit stage IDs.

```yaml
version: 2
description: "plan -> build -> check -> review"

defaults:
  profile: claude
  max: 3

cycle:
  target: build
  max: 3

stages:
  - id: plan
    type: agent
    prompt: plan
    max: 3

  - id: build
    type: agent
    prompt: build
    max: 100

  - id: check
    type: command
    command: ["make", "check"]

  - id: review
    type: agent
    prompt: review
    max: 1
```

Workflow guidelines:

- Use `agent` stages for judgment, code changes, reviews, and planning.
- Use `command` stages for deterministic gates like `make check`, test commands, or build commands.
- Set `cycle.target` to the first stage that should rerun when verification or review finds more work.
- Keep workflow stage IDs short and stable because state files store `next_stage_id`.
- Run `brr workflow validate <name>` before `brr workflow run <name>`.

## Common Commands

```bash
brr init
brr instructions
brr workflow init ship
brr workflow validate ship
brr workflow run ship --notify
brr workflow status ship
brr workflow run ship --reset
```

## When to Ask for Approval

Create `.brr-needs-approval` instead of guessing when the workflow requires:

- destructive operations,
- production or shared infrastructure changes,
- security-sensitive changes outside the stated goal,
- a product decision that materially changes scope,
- a spec change that project instructions mark as approval-only.
