# Loop Engine

## Purpose

The loop engine is the core orchestrator of brr. It repeatedly spawns a configured command with the prompt piped to stdin, monitors for completion signals, enforces iteration limits, and manages failure streaks. It is the main runtime that ties together locking, signal handling, process management, and signal file detection.

## Requirements

1. The engine spawns the configured command once per iteration, piping the resolved prompt text to the subprocess's stdin.
2. The engine runs iterations until one of these exit conditions is met: a signal file is detected, the maximum iteration count is reached, the process is interrupted by an OS signal, or three consecutive failures occur.
3. When `max` is set to a positive integer, the engine runs at most that many iterations. When `max` is zero, iterations are unlimited.
4. A command that exits with a non-zero status or fails to start counts as a failure — unless the git working tree is dirty after the exit, which indicates the agent made progress before crashing (e.g. context exhaustion, timeout). A dirty-tree failure resets the consecutive failure counter to zero instead of incrementing it. Three consecutive clean-tree failures stop the engine with an error describing the failure streak. If the fail-streak limit and max-iterations limit are reached on the same iteration, fail-streak takes precedence.
5. A successful iteration (exit code 0) resets the consecutive failure counter to zero. A failed iteration with a dirty working tree also resets the counter (see requirement 4).
6. If the maximum iteration count is reached and any iteration failed during the run, the engine returns an error wrapping the most recent failure. If all iterations succeeded, the engine returns nil.
7. Signal files are checked before spawning each iteration and after each iteration completes. Pre-existing signal files from a previous run are cleaned up and cause the engine to exit before the first iteration. The engine returns the corresponding stop reason for whichever signal file is present (signal-file-complete, signal-file-failed, or signal-file-approval). If multiple signal files are present, precedence follows `signal-files.md`: complete, failed, then approval.
8. Each iteration prints a header showing the iteration number and timestamp.
9. The engine acquires an exclusive lock before starting and releases it on exit.
10. When the engine stops, it communicates the stop reason to its caller. The stop reason distinguishes six cases: signal-file-complete, signal-file-failed, signal-file-approval, max-iterations-reached, fail-streak, and interrupted. The "interrupted" reason covers both SIGINT (Ctrl+C) and SIGTERM exits. The stop reason is not just an error -- successful exits (complete, approval, failed signal, max-with-final-success) must also be distinguishable. For signal-file-failed and signal-file-approval, the stop reason includes the file contents read at detection time (before cleanup). If the file cannot be read (per `signal-files.md` requirement 6), the content is an empty string. This allows callers (such as notifications) to take reason-specific action without re-reading cleaned-up files.
11. All engine output (iteration headers, status messages, stop-reason messages) is written to stderr. Subprocess stdout passes through to the parent's stdout unmodified. This allows callers to pipe or capture agent output without brr's own diagnostics mixed in.

## Constraints

- The engine must not leak child processes. All spawned processes must be cleaned up on exit regardless of how the engine stops. Between iterations, the engine reaps any zombie child processes to prevent process table exhaustion during long runs.
- Signal file cleanup must only remove regular files (not directories or symlinks).
- The engine must work on Linux, macOS, and Windows.

## Dependencies

- Depends on `docs/specs/concurrent-run-prevention.md` for exclusive locking.
- Depends on `docs/specs/signal-files.md` for the agent-to-brr communication protocol.
- Depends on `docs/specs/signal-handling.md` for OS signal response.
- Depends on `docs/specs/process-management.md` for cross-platform subprocess spawning.

## Acceptance Criteria

- [ ] Engine runs the correct number of iterations when max is set.
- [ ] Engine runs indefinitely when max is zero (until another exit condition).
- [ ] Three consecutive clean-tree failures stop the engine with an error.
- [ ] A success after failures resets the failure counter.
- [ ] A dirty-tree failure resets the failure counter (crash with progress).
- [ ] Signal files created during an iteration stop the engine after that iteration.
- [ ] Pre-existing signal files cause immediate exit before the first iteration.
- [ ] `.brr-failed` returns a failed stop reason and preserves the failure content.
- [ ] Max reached with any iteration having failed returns an error.
- [ ] Max reached with all iterations succeeded returns nil.
- [x] Stop reason is reported for each exit condition (complete, approval, max, fail-streak, interrupted).
- [ ] All engine output goes to stderr; agent stdout is not contaminated.
- [ ] All requirements have corresponding tests that pass.
- [ ] Existing tests continue to pass.
