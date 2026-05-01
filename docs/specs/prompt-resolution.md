# Prompt Resolution

## Purpose

Prompt resolution determines how a user-provided prompt argument is interpreted and converted into text that gets piped to the subprocess. It supports three modes -- direct file path, named prompt lookup, and inline text -- with a defined priority order and security boundaries around file access.

## Requirements

1. Resolution follows a strict priority order: existing regular file > named prompt > inline text. The first match wins.
2. The argument is first checked against the filesystem via `lstat()`. If it matches a regular file on disk, its contents are read and used as the prompt text -- regardless of whether it looks like a file path syntactically.
3. File path resolution rejects symlinks, FIFOs, and other non-regular files. Directories fall through to named prompt or inline text resolution.
4. Prompt files larger than 10 MiB are rejected with an error.
5. If the argument does not match an existing file, but looks like a file path (contains a path separator or has a recognized extension: `.md`, `.txt`, `.prompt`; when the argument contains spaces, both a path separator and a recognized extension are required), the system reports a "file not found" error rather than falling through.
6. When the argument has no spaces and did not match an existing file or trigger a file-not-found error, it is treated as a named prompt. Named prompts are looked up in `.brr/prompts/<name>.md` (project-local) then the user-global prompt directory: `~/.config/brr/prompts/<name>.md` on Linux, `~/Library/Application Support/brr/prompts/<name>.md` on macOS, `%AppData%\brr\prompts\<name>.md` on Windows (matching the OS conventions in `docs/specs/configuration.md` requirement 2). The first regular file found wins.
7. Named prompt lookup rejects path traversal attempts (arguments containing `..`).
8. Named prompt lookup rejects symlinks at both search paths.
9. If no existing file, file-not-found error, or named prompt matches, the argument is used as inline prompt text verbatim.
10. The final resolved prompt must be non-empty after trimming whitespace. An empty prompt produces an error.

## Constraints

- File reads must use TOCTOU-safe operations (lstat before open, verify after open).
- Must not follow symlinks at any stage of resolution.
- Must work on Linux, macOS, and Windows.

## Dependencies

- Depends on `docs/specs/file-safety.md` for symlink-safe file reads.

## Acceptance Criteria

- [ ] Inline text passes through unchanged.
- [ ] A valid file path returns the file's contents.
- [ ] Direct symlinks and FIFOs are rejected without being read.
- [ ] Direct directories fall through to the next resolution stage without being read.
- [ ] Named prompts resolve from project-local before user-global.
- [ ] Symlinks and other non-regular files at named prompt paths are rejected.
- [ ] Path traversal (`..`) in named prompts is rejected.
- [ ] Files over 10 MiB are rejected.
- [ ] Empty/whitespace-only prompts produce an error.
- [ ] All requirements have corresponding tests that pass.
- [ ] Existing tests continue to pass.
