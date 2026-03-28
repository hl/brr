# File Safety

## Purpose

File safety provides TOCTOU-safe file operations used throughout brr. Multiple components need to read files while rejecting symlinks, directories, and other non-regular file types. This module centralizes those checks into reusable operations that are resistant to race conditions where a file could be swapped between the check and the use.

## Requirements

1. Opening a file for reading performs a three-step TOCTOU-safe check: `lstat()` to verify the path is a regular file, `open()` to get a file descriptor, then `fstat()` on the descriptor to verify it still matches the original `lstat()` result.
2. If the file is a symlink, directory, FIFO, or any non-regular file type, the operation returns a specific "not a regular file" error.
3. If the file at the path changes between `lstat()` and `open()` (TOCTOU race), the `fstat()` comparison detects the mismatch and returns an error.
4. A convenience function reads the entire contents of a regular file, combining the safe-open and read operations.
5. A non-error boolean check reports whether a path points to a regular file (returns false for missing files and non-regular files).
6. A symlink guard checks whether a path is a symlink and returns an error if so. This is used by write-side callers (such as project initialization) to reject symlinks before writing without needing the full open-read sequence.

## Constraints

- All checks must use `lstat()` (not `stat()`) to avoid following symlinks.
- Must work on Linux, macOS, and Windows.
- Must not introduce external dependencies.

## Dependencies

- No dependencies on other specs.

## Acceptance Criteria

- [ ] Regular files are opened and read successfully.
- [ ] Symlinks are rejected with a "not a regular file" error.
- [ ] Directories are rejected.
- [ ] TOCTOU race between lstat and open is detected via fstat comparison.
- [ ] The boolean check returns false for missing files, symlinks, and directories.
- [ ] The symlink guard rejects symlinks with an error and passes for regular files and missing paths.
- [ ] All requirements have corresponding tests that pass.
- [ ] Existing tests continue to pass.
