# Concurrent Run Prevention

## Purpose

Concurrent run prevention ensures that only one brr instance runs in a given working directory at a time. Multiple concurrent instances would race on signal files, produce interleaved output, and confuse the subprocess. This is enforced through an exclusive file lock.

## Requirements

1. Before starting its first iteration, the engine acquires an exclusive, non-blocking lock on a `.brr.lock` file in the working directory.
2. If the lock cannot be acquired because another process holds it, the engine exits immediately with an error message: "another brr instance is already running in this directory".
3. The lock is released when the engine exits, regardless of the exit path (success, error, or signal).
4. The `.brr.lock` file is intentionally not deleted on exit. Deleting the file would create a race condition where a new process could create and lock a different inode while the old process still holds a lock on the original inode.
5. On Unix, locking uses `flock()` with `LOCK_EX | LOCK_NB`.
6. On Windows, locking uses `LockFileEx()` with exclusive and fail-immediately flags.

## Constraints

- Lock acquisition must be non-blocking -- brr must never wait for another instance to finish.
- The lock file must survive process exit (no deletion) to prevent inode-reuse races.
- Must work on Linux, macOS, and Windows with platform-appropriate locking primitives.

## Dependencies

- No dependencies on other specs.

## Acceptance Criteria

- [ ] A single brr instance acquires the lock and runs normally.
- [ ] A second brr instance in the same directory fails immediately with a clear error.
- [ ] The lock is released on normal exit, error exit, and signal-triggered exit.
- [ ] The `.brr.lock` file remains on disk after the engine exits.
- [ ] All requirements have corresponding tests that pass.
- [ ] Existing tests continue to pass.
