---
description: Create a new release with version bump, changelog update, and git tag
allowed-tools:
  - Bash(git:*)
  - Bash(make:*)
  - Bash(gh:*)
  - Edit
  - Read
---

# Release Command

Create a new AILANG release with the specified version number.

**Usage:** `/release <version>`

**Example:** `/release 0.0.10`

## Steps to perform:

1. **Read current version** from README.md to understand the current state

2. **Update version in documentation:**
   - README.md: Update "Current Version: vX.X.X"
   - docs/reference/implementation-status.md: Update "Current Version: vX.X.X"
   - CHANGELOG.md: Change `## [Unreleased]` to `## [v$1] - $(date +%Y-%m-%d)`

3. **Verify tests pass:**
   - Run `make test` to ensure all tests pass
   - Run `make lint` to ensure code quality

4. **Commit changes:**
   - Stage README.md, CHANGELOG.md, and docs/reference/implementation-status.md
   - Commit with message: "Release v$1"

5. **Create and push git tag:**
   - Create annotated tag: `git tag -a v$1 -m "Release v$1"`
   - Push tag: `git push origin v$1`

6. **Push commit:**
   - Push to remote: `git push`

7. **Monitor CI/CD:**
   - Run `gh run list --limit 3` to check CI status
   - Verify builds pass on all platforms (Linux, macOS, Windows)

8. **Summary:**
   - Confirm version v$1 released
   - Show git tag details
   - Show CI workflow URLs

## Version Format

Version should be in semantic versioning format: `MAJOR.MINOR.PATCH`
- Examples: `0.0.9`, `0.1.0`, `1.0.0`

## Prerequisites

- Working directory must be clean (no uncommitted changes)
- Current branch should be `dev` or `main`
- All tests must pass before release
