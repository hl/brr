# Signal Files

## Purpose

Signal files are the communication protocol between an agent running inside brr and the brr engine itself. Agents cannot communicate with brr directly, so they write sentinel files to the working directory to signal that work is complete, failed, needs human approval, or needs a workflow cycle. This spec defines the file names, semantics, and safety rules for this protocol.

## Requirements

1. Four signal files are recognized: `.brr-complete`, `.brr-failed`, `.brr-needs-approval`, and `.brr-cycle`.
2. `.brr-complete` indicates the agent has finished all work. When detected, the engine prints a success message and stops.
3. `.brr-failed` indicates the agent hit a blocker or failed after making the allowed recovery attempts. When detected, the engine reads up to 4 KiB of the file's contents and displays them to the user, then stops.
4. `.brr-needs-approval` indicates the agent needs human review before continuing. When detected, the engine reads up to 4 KiB of the file's contents and displays them to the user, then stops.
5. `.brr-cycle` indicates a workflow stage needs another pass from the configured cycle stage. When detected, the engine returns a cycle stop reason; workflow orchestration decides whether cycling is available.
6. Content read from `.brr-failed` and `.brr-needs-approval` is truncated at a UTF-8 character boundary to prevent garbled output.
7. If `.brr-failed` or `.brr-needs-approval` exists but cannot be read, the engine still honors the signal and stops -- the file's existence is the signal, not its contents.
8. Only regular files are honored as signal files. Directories, symlinks, and other non-regular file types are ignored.
9. Signal files are cleaned up (deleted) when the engine exits. Cleanup only removes regular files.
10. If signal files exist before the engine's first iteration (left over from a previous run), they are cleaned up and the engine exits immediately without running any iterations.
11. If multiple signal files exist at once, precedence is `.brr-complete`, then `.brr-failed`, then `.brr-needs-approval`, then `.brr-cycle`.

## Constraints

- Signal file names must be stable across versions -- agents in prompt templates depend on these exact names.
- Signal file detection must not follow symlinks.
- Signal file paths are always relative to the working directory.

## Dependencies

- Depends on `docs/specs/file-safety.md` for symlink-safe file detection and reads.

## Acceptance Criteria

- [ ] `.brr-complete` causes the engine to stop with a success message.
- [ ] `.brr-failed` causes the engine to stop and display file contents.
- [ ] `.brr-needs-approval` causes the engine to stop and display file contents.
- [ ] `.brr-cycle` causes the engine to stop with a cycle reason.
- [ ] `.brr-failed` and `.brr-needs-approval` content is truncated at UTF-8 boundaries.
- [ ] Unreadable `.brr-failed` and `.brr-needs-approval` still trigger a stop.
- [ ] Signal precedence is complete, failed, approval, then cycle.
- [ ] Symlinks and directories named as signal files are ignored.
- [ ] Pre-existing signal files are cleaned up and prevent iteration.
- [ ] Signal file cleanup only removes regular files.
- [ ] All requirements have corresponding tests that pass.
- [ ] Existing tests continue to pass.
