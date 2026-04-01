# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/),
and this project adheres to [Semantic Versioning](https://semver.org/).

## [Unreleased]

### Changed

- Crash with dirty working tree no longer counts toward the consecutive failure streak — agents that make progress before dying (context exhaustion, timeout) are retried instead of stopped

## [0.2.1] "Read the Signs" - 2026-03-29

### Changed

- Expanded signal file documentation (`.brr-complete`, `.brr-needs-approval`, `.brr.lock`) across CLI `--help`, landing page, and README with clearer descriptions and usage guidance

## [0.2.0] "Ding When Done" - 2026-03-29

### Added

- Desktop notifications on loop termination via `--notify` / `-n` flag — sends OS-native notifications for completion, approval needed, max iterations, and fail streak events (macOS via osascript, Linux via notify-send)
- Structured `StopReason` in engine results, distinguishing all five exit conditions (complete, approval, max-iterations, fail-streak, interrupted)
- Hardened default prompts: dirty state recovery, repeated failure detection with auto-escalation, retry limits, and combined review phase with fallbacks
- New `audit` prompt for autonomous codebase auditing with parallel agents and severity gating

## [0.1.3] "Locks Changed" - 2026-03-28

### Added

- GitHub Pages landing page styled after `brr --help`, including the ASCII banner, CLI color palette, install and usage docs, and a workflow to deploy `docs/` from `main`

### Fixed

- Lock file no longer deleted on release, preventing a race where two processes could both acquire the lock
- Lock errors now show the real cause (e.g. permission denied) instead of always blaming a concurrent instance
- `Run()` now returns an error when max iterations reached but the last iteration failed (previously exited 0)
- Signal file cleanup no longer deletes directories or symlinks that happen to share signal file names
- Prompt resolution rejects symlinks and FIFOs instead of following them
- Prompt files larger than 10 MiB are rejected with a clear error
- `rejectSymlink` no longer swallows non-ENOENT errors from `Lstat`
- Scaffold rollback reports errors instead of silently discarding them, and cleans up empty `.brr/` directories

## [0.1.2] "Thoroughly Frisked" - 2026-03-27

### Added

- Lockfile (`.brr.lock`) to prevent concurrent runs from racing on signal files

### Fixed

- `resolvePrompt` now distinguishes permission errors from "file not found" instead of misreporting all stat failures
- Fail-streak error now includes the underlying cause (last child error) instead of a generic message
- Scaffold `Init` correctly identifies permission errors vs missing directories during rollback
- Config validation errors no longer hardcode `.brr.yaml` when the config may come from the global path
- Signal handling now logs errors when killing child processes instead of silently discarding them

### Changed

- `brr init` now adds `.brr.lock` to `.gitignore`

## [0.1.1] "Helpful Graffiti" - 2026-03-27

### Added

- ASCII art banner, tagline, and GitHub link in `--help` output
- `/release` slash command for cutting releases from Claude Code

## [0.1.0] - 2026-03-27

### Added

- Loop engine: run any AI agent prompt in a loop with fresh sessions
- Agent-agnostic profiles via `.brr.yaml` (Claude Code, Codex, or custom)
- Prompt resolution: file path, named prompt (`.brr/prompts/`), or inline text
- `brr init` scaffolding for project config and example prompts
- Signal file control: `.brr-complete` and `.brr-needs-approval`
- Graceful Ctrl+C handling (finish current / interrupt child / force kill)
- Auto-stop after three consecutive failures
- `--max` flag to bound iterations
- `--version` flag with embedded build info
- Cross-platform support: Linux, macOS, Windows (amd64, arm64)
