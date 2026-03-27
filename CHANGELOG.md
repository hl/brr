# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/),
and this project adheres to [Semantic Versioning](https://semver.org/).

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
