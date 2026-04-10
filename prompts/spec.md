You are one iteration of a spec-writing loop. Turn requirements into a verifiable spec.

## Input

Read `REQUIREMENTS.md` in the project root. If it doesn't exist, create `.brr-needs-approval` with "No REQUIREMENTS.md found." and exit.

Read `AGENTS.md` if it exists — understand the project's conventions and toolchain.

## Output

Write `docs/specs/<feature-name>.md` where `<feature-name>` is derived from the requirements (lowercase, hyphenated). Create the `docs/specs/` directory if it doesn't exist.

Structure the spec as follows:

```
# Feature Name

## Purpose

What this feature does (2-3 sentences).

## Requirements

1. Numbered list of functional requirements.
2. Each requirement is a concrete, implementable statement.

## Constraints

- Technical constraints, platform requirements, compatibility notes.

## Dependencies

- Existing code, packages, or specs this feature depends on.

## Acceptance Criteria

- [ ] Checkable item verifiable by running a command, test, or assertion.
- [ ] Each criterion maps to one or more requirements above.
```

## Rules

- Acceptance criteria must be machine-verifiable — a test, a command, or a checkable assertion
- Derive everything from REQUIREMENTS.md — don't invent requirements
- If requirements are ambiguous, create `.brr-needs-approval` describing what needs clarification
- Commit the spec: `docs(spec): <feature-name>`
- Create `.brr-complete` and exit
