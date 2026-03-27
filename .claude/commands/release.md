---
description: Cut a new release — updates CHANGELOG, tags, and pushes to trigger CI
allowed-tools: Bash, Read, Edit, Write, Glob, Grep
user-input: Version bump type or explicit version (e.g. "patch", "minor", "major", "0.3.0")
---

You are releasing a new version of brr. Follow these steps exactly.

## 1. Determine the version

Run `git tag --sort=-v:refname | head -1` to get the latest tag.

The user provided: $ARGUMENTS

- If the user said "patch", "minor", or "major": bump that component of the latest tag (e.g. v0.1.0 + minor = v0.2.0)
- If the user gave an explicit version like "0.3.0": use that (add v prefix for the tag)
- If $ARGUMENTS is empty: look at the commits since the last tag and suggest a version based on conventional commits (feat = minor, fix = patch)

Confirm the version with the user before proceeding.

## 2. Gather changes

Run `git log --oneline <last-tag>..HEAD` to see all commits since the last release.

Categorize them into Added/Changed/Fixed/Removed sections per Keep a Changelog format. Only include user-visible changes — skip docs, test, chore, and ci commits unless they affect the user experience.

## 3. Update CHANGELOG.md

Read the existing CHANGELOG.md. Add a new version section below the header and above the previous version entry. Use this format:

```
## [X.Y.Z] - YYYY-MM-DD

### Added
- ...

### Fixed
- ...
```

Use today's date. Only include sections that have entries.

## 4. Commit and tag

```bash
git add CHANGELOG.md
git commit -m "chore(release): prepare vX.Y.Z"
git tag -a vX.Y.Z -m "vX.Y.Z"
```

## 5. Push

Push both the commit and the tag:

```bash
git push origin main
git push origin vX.Y.Z
```

Tell the user the release workflow is running and link to https://github.com/hl/brr/actions.

## 6. Verify

Run `gh run list --repo hl/brr --limit 1` to show the workflow status.
