---
name: AILANG Release Manager
description: Create new AILANG releases with version bumps, changelog updates, git tags, and CI/CD verification. Use when user says "ready to release", "create release", mentions version numbers, or wants to publish a new version.
---

# AILANG Release Manager

Create a complete AILANG release with version bump, changelog update, git tag, and CI/CD verification.

## Quick Start

**Most common usage:**
```bash
# User says: "Ready to release v0.3.14"
# This skill will:
# 1. Run pre-release checks (tests, lint, file sizes)
# 2. Update version in docs
# 3. Create git tag
# 4. Push to trigger CI/CD
# 5. Verify release artifacts
```

## When to Use This Skill

Invoke this skill when:
- User says "ready to release", "create release", "publish release"
- User mentions a specific version number (e.g., "v0.3.14")
- User asks about release process or workflow
- After completing a sprint and code is ready to ship

## Available Scripts

### `scripts/pre_release_checks.sh`
Run all pre-release verification checks before making any changes.

**Usage:**
```bash
.claude/skills/release-manager/scripts/pre_release_checks.sh
```

**What it checks:**
1. Test suite passes (`make test`)
2. Linting passes (`make lint`)
3. No files exceed 800 lines (`make check-file-sizes`)

**Output:**
```
Running pre-release checks...

1/3 Running test suite...
  ✓ Tests passed

2/3 Running linter...
  ✓ Linting passed

3/3 Checking file sizes...
  ✓ File sizes OK (all files ≤800 lines)

✓ All pre-release checks passed!
Ready to proceed with release.
```

**Exit codes:**
- `0` - All checks passed
- `1` - One or more checks failed (see logs in /tmp/pre_release_*.log)

### `scripts/post_release_checks.sh <version>`
Verify release was created successfully on GitHub.

**Usage:**
```bash
.claude/skills/release-manager/scripts/post_release_checks.sh 0.3.14
```

**What it checks:**
1. Git tag exists (`git tag -l v0.3.14`)
2. GitHub release exists (`gh release view v0.3.14`)
3. All platform binaries present (Darwin x64/ARM64, Linux, Windows)
4. Latest CI run passed

**Output:**
```
Verifying release v0.3.14...

1/4 Checking git tag...
  ✓ Tag v0.3.14 exists

2/4 Checking GitHub release...
  ✓ GitHub release v0.3.14 exists

3/4 Checking release binaries...
  ✓ ailang-darwin-amd64.tar.gz
  ✓ ailang-darwin-arm64.tar.gz
  ✓ ailang-linux-amd64.tar.gz
  ✓ ailang-windows-amd64.zip
  ✓ All platform binaries present

4/4 Checking CI status...
  ✓ Latest CI run passed

✓ Release v0.3.14 verified successfully!
URL: https://github.com/sunholo-data/ailang/releases/tag/v0.3.14
```

## Release Workflow

### 1. Pre-Release Verification (CRITICAL)

**Run checks BEFORE making any changes:**
```bash
.claude/skills/release-manager/scripts/pre_release_checks.sh
```

**If checks fail:**
- Tests failing → Fix tests first
- Linting failing → Run `make fmt` or fix issues
- File sizes failing → Use `codebase-organizer` agent to split large files
- **DO NOT proceed until all checks pass**

### 2. Update Version in Documentation

Update these files:
- **README.md**: Change "Current Version: vX.X.X"
- **docs/reference/implementation-status.md**: Change "Current Version: vX.X.X"
- **CHANGELOG.md**: Change `## [Unreleased]` to `## [vX.X.X] - YYYY-MM-DD`

### 3. Post-Update Verification (CRITICAL)

**Run checks AGAIN after documentation changes:**
```bash
make test
make lint
```

If either fails, fix before committing.

### 4. Commit Changes

```bash
git add README.md CHANGELOG.md docs/reference/implementation-status.md
git commit -m "Release vX.X.X"
```

### 5. Create and Push Git Tag

```bash
git tag -a vX.X.X -m "Release vX.X.X"
git push origin vX.X.X
```

### 6. Push Commit

```bash
git push
```

### 7. Monitor CI/CD

```bash
# Check recent runs
gh run list --limit 3

# Watch for completion (typically 2-3 minutes)
gh run watch
```

### 8. Verify Release

**Use the verification script:**
```bash
.claude/skills/release-manager/scripts/post_release_checks.sh X.X.X
```

Or manually:
```bash
gh release view vX.X.X
```

Expected binaries:
- ailang-darwin-amd64.tar.gz (macOS Intel)
- ailang-darwin-arm64.tar.gz (macOS Apple Silicon)
- ailang-linux-amd64.tar.gz (Linux)
- ailang-windows-amd64.zip (Windows)

### 9. Handle CI Failures

If CI fails after push:
```bash
# Check logs
gh run list --workflow=CI --limit 3
gh run view <run-id> --log-failed

# Fix issues
# Commit fixes
git commit -m "Fix CI: <issue>"
git push
```

### 10. Summary

Show user:
- ✓ Version vX.X.X released
- ✓ Git tag created
- ✓ Release URL: https://github.com/sunholo-data/ailang/releases/tag/vX.X.X
- ✓ CI workflow status
- **Next step**: Run `post-release` skill to update benchmarks and dashboard

## Resources

### Release Checklist
See [`resources/release_checklist.md`](resources/release_checklist.md) for complete step-by-step checklist.

## Prerequisites

- Working directory must be clean (no uncommitted changes)
- Current branch should be `dev` or `main`
- All tests must pass (`make test`)
- All linting must pass (`make lint`)
- No files exceed 800 lines (`make check-file-sizes`)

## Version Format

Semantic versioning: `MAJOR.MINOR.PATCH`
- Examples: `0.0.9`, `0.1.0`, `1.0.0`

## Common Issues

### Tests Fail Before Release
**Solution**: Fix tests first, don't skip this step.

### Linting Fails
**Solution**: Run `make fmt` to auto-format, or fix manually.

### File Size Check Fails
**Solution**: Use `codebase-organizer` agent to split large files before releasing.

### CI Fails After Push
**Solution**: Check logs with `gh run view <run-id> --log-failed`, fix, commit, push again.

### Release Missing Binaries
**Solution**: CI workflow may still be running. Wait 2-3 minutes and check again.

## Progressive Disclosure

This skill loads information progressively:

1. **Always loaded**: This SKILL.md file (YAML frontmatter + workflow overview)
2. **Execute as needed**: Scripts in `scripts/` directory (validation, verification)
3. **Load on demand**: `resources/release_checklist.md` (detailed checklist)

Scripts execute without loading into context window, saving tokens while providing automation.

## Notes

- This skill follows Anthropic's Agent Skills specification (Oct 2025)
- Scripts handle verification automatically
- Always run pre-release checks BEFORE making changes
- Always run post-update checks AFTER documentation changes
- Use post-release skill after successful release for benchmarks and dashboard
