---
name: "release"
description: "Use when the user asks to release, cut, prepare, tag, or publish a new version of brr. Handles local SemVer release prep for this GoReleaser project, including CHANGELOG promotion, required checks, release commits, and local tags. Requires explicit approval before externally visible publishing such as pushing commits/tags or running a live GoReleaser release."
---

# brr Release

## Guardrails

- Treat `git push`, pushing tags, creating GitHub Releases, GoReleaser publishing, and Homebrew tap updates as externally visible. Stop and ask before doing them unless the user explicitly authorized that exact action.
- Local release commits and local annotated tags are acceptable when the user asked to release and the worktree has no unrelated changes.
- Do not skip `make check`. If it fails, fix the root cause before continuing. Yes, the compiler gets a vote.
- Do not rewrite unrelated worktree changes.

## Project Release Shape

- Version comes from Git tags via GoReleaser ldflags in `.goreleaser.yaml`; do not hardcode it in Go.
- Changelog follows Keep a Changelog with SemVer and project codenames:
  - `## [x.y.z] "Codename" - YYYY-MM-DD`
- Previous release prep commits use:
  - `chore(release): prepare vx.y.z`
- Release tags are annotated:
  - `git tag -a vx.y.z -m "vx.y.z"`
- Local validation targets:
  - `make check`
  - `make release-check`
  - `make release-snapshot`

## Workflow

1. Inspect state:
   - `git status --short --branch`
   - `git tag --sort=-v:refname | head -20`
   - `git log --oneline <latest-tag>..HEAD`
2. Pick the next SemVer from `CHANGELOG.md` and commits since the latest tag:
   - patch for fixes only
   - minor for added user-visible behavior or breaking `0.x` changes
   - major only for stable `v1+` breaking changes
3. If detached at `origin/main`, create or switch to a local release branch before committing.
4. Promote `CHANGELOG.md`:
   - Insert a fresh empty `## [Unreleased]` section.
   - Move current Unreleased entries under `## [x.y.z] "Codename" - YYYY-MM-DD`.
   - Preserve existing Added/Changed/Fixed/Removed grouping.
5. Run:
   - `make check`
   - `make release-check`
   - `make release-snapshot`
6. Commit:
   - `git add CHANGELOG.md`
   - `git commit -m "chore(release): prepare vx.y.z"`
7. Create the local annotated tag:
   - `git tag -a vx.y.z -m "vx.y.z"`

## Publish Stop

Before publishing, summarize:

- target version and tag
- release commit
- checks run and results
- exact commands needed to publish

Then ask for approval before running commands such as:

```shell
git push origin <branch>
git push origin vx.y.z
mise exec -- goreleaser release --clean
```
