# Signal Handling

## Purpose

Signal handling defines how brr responds to OS interrupts (Ctrl+C / SIGINT) and termination signals (SIGTERM). It implements a three-level escalation for SIGINT that gives the user progressively more forceful control over shutdown, and immediate forwarding for SIGTERM. This ensures child processes are always cleaned up and the user is never stuck.

## Requirements

1. The first SIGINT (Ctrl+C) sets a stopping flag. The current iteration is allowed to finish, then the engine exits gracefully with an interrupted status.
2. The second SIGINT sends SIGINT to the entire child process group, requesting a graceful shutdown of the subprocess tree.
3. The third SIGINT sends SIGKILL to the entire child process group, forcefully terminating the subprocess tree.
4. SIGTERM is forwarded immediately to the child process group as SIGTERM. The engine waits for the current child process to exit (it should terminate in response to the forwarded signal), then exits the loop.
5. All signals target the child process group (not just the direct child process), ensuring the entire subprocess tree receives the signal.
6. On Windows, where SIGTERM is not deliverable, the equivalent behavior uses `taskkill` for process tree management: graceful kill without `/F` for interrupt-level signals, force kill with `/F` for SIGKILL-level signals.
7. If the process-group signal delivery fails on Windows, the system falls back to killing the direct child process.

## Constraints

- Signal handling must not interfere with the child process's own signal handling during the first Ctrl+C (the child should complete naturally).
- Must work correctly on Linux, macOS, and Windows.
- Must handle the case where no child process is running when a signal arrives.

## Dependencies

- Depends on `docs/specs/process-management.md` for process group setup.

## Acceptance Criteria

- [ ] First Ctrl+C allows the current iteration to finish, then stops the engine.
- [ ] Second Ctrl+C interrupts the child process group.
- [ ] Third Ctrl+C force-kills the child process group.
- [ ] SIGTERM is forwarded immediately to the child process group.
- [ ] Signals work correctly on Unix (Linux/macOS).
- [ ] Signals work correctly on Windows with taskkill fallback.
- [ ] No child process leak after any signal sequence.
- [ ] All requirements have corresponding tests that pass.
- [ ] Existing tests continue to pass.
