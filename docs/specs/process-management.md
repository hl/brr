# Process Management

## Purpose

Process management handles cross-platform subprocess spawning for the loop engine. Each iteration spawns a command as a child process with the prompt piped to stdin. The subprocess must be created in its own process group so that signals can target the entire child tree rather than just the direct child.

## Requirements

1. Each iteration spawns the configured command with its arguments as a subprocess.
2. The resolved prompt text is piped to the subprocess's stdin.
3. The subprocess's stdout and stderr are connected to the parent's stdout and stderr.
4. On Unix, the subprocess is created in a new process group (`Setpgid=true`) so that signals can be sent to the group via negative PID.
5. On Windows, the subprocess is created with `CREATE_NEW_PROCESS_GROUP` so that console control events and `taskkill /T` target the child tree.
6. The engine tracks the currently running subprocess so that signal handlers can deliver signals to it.
7. When the subprocess exits, the engine captures its exit code to determine success (0) or failure (non-zero).

## Constraints

- Must not leak file descriptors or process handles between iterations.
- Must work on Linux, macOS, and Windows with platform-specific process attributes.
- Subprocess setup must not modify the parent process's own process group or signal disposition.

## Dependencies

- No dependencies on other specs.

## Acceptance Criteria

- [ ] Subprocess receives the prompt text on stdin.
- [ ] Subprocess stdout/stderr appear in the parent's terminal.
- [ ] On Unix, the subprocess runs in its own process group.
- [ ] On Windows, the subprocess runs with CREATE_NEW_PROCESS_GROUP.
- [ ] Exit code is correctly captured and reported.
- [ ] All requirements have corresponding tests that pass.
- [ ] Existing tests continue to pass.
