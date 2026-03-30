# Project Initialization

## Purpose

Project initialization (`brr init`) scaffolds the files and directories needed to use brr in a project. It creates a config file with starter profiles, a directory for named prompts, and updates `.gitignore` to exclude runtime artifacts. The operation is atomic -- partial failures are rolled back.

## Requirements

1. `brr init` creates a `.brr.yaml` file in the current directory with starter profiles for common agents.
2. `brr init` creates a `.brr/prompts/` directory for named prompt files.
3. `brr init` appends brr's runtime artifacts (`.brr-complete`, `.brr-needs-approval`, `.brr.lock`) to `.gitignore`. If `.gitignore` does not exist, it is created.
4. Entries already present in `.gitignore` are not duplicated. Matching is exact against non-comment, non-empty lines.
5. If `.brr.yaml` already exists and `--force` is not set, the command fails with an error.
6. If `.brr.yaml` already exists and `--force` is set, the file is overwritten.
7. If any stage fails, all previously completed stages are rolled back: created files are removed and overwritten files are restored to their original contents.
8. All writable paths (`.brr.yaml`, `.gitignore`, `.brr/`) are checked for symlinks before writing. Symlinks are rejected with an error.
9. File writes use an atomic temp-file-then-rename pattern to prevent partial writes.
10. The command prints a summary of what was created and next-step guidance.

## Constraints

- Must not follow or overwrite symlinks at any writable path.
- Rollback must be best-effort: errors during rollback are logged but do not suppress the original error.
- Must work on Linux, macOS, and Windows.

## Dependencies

- Depends on `docs/specs/signal-files.md` for signal file names (`.brr-complete`, `.brr-needs-approval`) added to `.gitignore`.
- Depends on `docs/specs/concurrent-run-prevention.md` for the lock file name (`.brr.lock`) added to `.gitignore`.
- Depends on `docs/specs/file-safety.md` for symlink rejection.

## Acceptance Criteria

- [ ] `brr init` in a clean directory creates `.brr.yaml`, `.brr/prompts/`, and `.gitignore` entries.
- [ ] Running `brr init` twice without `--force` fails.
- [ ] Running `brr init` twice with `--force` overwrites `.brr.yaml`.
- [ ] Existing `.gitignore` entries are not duplicated.
- [ ] Symlinks at any writable path are rejected.
- [ ] Partial failure triggers rollback of completed stages.
- [ ] File writes are atomic (no partial content on crash).
- [ ] All requirements have corresponding tests that pass.
- [ ] Existing tests continue to pass.
