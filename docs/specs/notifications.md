# Notifications

## Purpose

Send desktop notifications when the brr loop terminates so users don't have to
watch the terminal. Brr runs are long-lived and hands-off — users switch context
and need a nudge when the loop ends and attention is required.

## Requirements

1. A notification is sent when the loop stops for any of these reasons:
   - All tasks complete (`.brr-complete` signal file found)
   - Human approval needed (`.brr-needs-approval` signal file found)
   - Too many consecutive failures (fail streak reaches the maximum)
   - Max iterations reached
2. Each notification includes a title and body that identify which terminal
   event occurred. The approval notification includes the approval file
   contents carried in the engine's stop reason (read before signal file
   cleanup per `loop-engine.md` requirement 10), truncated to fit platform
   limits. If the engine could not read the file at detection time (per
   `signal-files.md` requirement 5), the notification body states that
   approval is needed without file contents.
3. Notifications are enabled with `--notify` / `-n` flag. Off by default.
4. When the terminal is not interactive (stderr is not a TTY), `--notify`
   is accepted but may silently no-op if the platform has no notification
   mechanism.
5. Notification delivery is best-effort. A failure to send a notification
   must not affect the exit code or behaviour of the loop.
6. Interrupted stops (Ctrl+C or SIGTERM) do not send a notification — the
   user is already at the terminal or the process is being managed externally.

## Constraints

- No new external dependencies. Use only OS-provided notification mechanisms
  and the Go standard library: `osascript` on macOS, `notify-send` on Linux.
  On Windows, notifications silently no-op (no OS mechanism available without
  external dependencies).
- Notification dispatch must not delay engine shutdown. Dispatch occurs
  after the engine has stopped, so it does not block the engine itself.
- The feature is a single concern: converting terminal events into OS
  notifications. It does not add sound, retry logic, webhook support, or
  custom notification commands. Those are separate specs if needed later.

## Dependencies

- Depends on `docs/specs/loop-engine.md` for structured stop reasons (requirement 10) that distinguish the four terminal events.
- Depends on `docs/specs/signal-files.md` for signal file names and the unreadable-file flow.

## Acceptance Criteria

- [x] `brr <prompt> --notify` sends a desktop notification on each of the
      four terminal events listed in requirement 1.
- [x] `brr <prompt>` (without `--notify`) sends no notifications.
- [x] Ctrl+C stop does not trigger a notification.
- [x] A notification failure (e.g. `notify-send` missing on Linux) is logged
      to stderr as a warning and does not change the exit code.
- [x] All existing tests continue to pass.
- [x] `make check` passes.

## Decisions

- **StopReason refactor prerequisite.** The loop-engine spec (requirement 10) required structured stop reasons but they were not yet implemented. Added `StopReason` type and changed `Run()` to return `(*Result, error)` as a prerequisite for this feature. This also fulfils the loop-engine requirement.
- **Synchronous dispatch.** Notification dispatch is synchronous after `Run()` returns, not in a goroutine. `osascript` and `notify-send` both complete in ~50ms, and a goroutine would race with process exit. The spec constraint "must not delay engine shutdown" is satisfied because the engine has already stopped.
- **Approval content truncated to 256 bytes.** Platform notification bodies have varying limits (macOS ~256 chars via osascript, Linux varies). Truncation breaks at word boundaries with an ellipsis marker.
- **No new dependencies.** Uses only `os/exec` and build tags (`darwin`, `linux`, `!darwin && !linux`) to select the platform mechanism. No third-party notification libraries.
