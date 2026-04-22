---
name: release-manager
description: Create new AILANG releases with version bumps, changelog updates, git tags, and CI/CD verification. Use when user says "ready to release", "create release", mentions version numbers, or wants to publish a new version.
---

# AILANG Release Manager

Create a complete AILANG release with version bump, changelog update, git tag, and CI/CD verification.

## Current State

- **Current version**: !'cat std/VERSION'
- **Branch**: !'git branch --show-current'
- **Latest tag**: !'git describe --tags --abbrev=0 2>/dev/null || echo "no tags"'
- **Working directory**: !'git status --short | head -5'
- **Recent commits since last tag**: !'git log $(git describe --tags --abbrev=0 2>/dev/null)..HEAD --oneline 2>/dev/null | head -10'
- **GitHub auth**: !'gh auth status 2>&1 | grep "Logged in" | head -1'
- **Active changelog**: !'ls changelogs/ | grep current 2>/dev/null'

> **Use the data above first.** Only re-run these commands manually if the injected context is empty or you need to refresh after making changes.

## Changelog Architecture

**The changelog is split into themed files in `changelogs/`:**
- Root `CHANGELOG.md` is an index file with links to archives
- Active changelog entries go in the file matching the current major.minor version (e.g., `changelogs/v0.10-current.md`)
- When releasing a new major version, create a new `changelogs/vX.Y-<theme>.md` file
- All changelog files are indexed by `ailang docs search`

**To find the active changelog file:**
```bash
ls changelogs/ | grep current  # Currently: v0.10-current.md
```

**When writing changelog entries**, write to the active `changelogs/v*.*.current.md` file, NOT root `CHANGELOG.md`.

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
# 6. Broadcast release notification with changelog
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

### `scripts/check_implemented_docs.sh <version>`
Verify all implemented design docs are documented in CHANGELOG.

**Usage:**
```bash
.claude/skills/release-manager/scripts/check_implemented_docs.sh 0.5.10
```

**What it checks:**
1. Finds all design docs in `design_docs/implemented/vX_Y_Z/`
2. Verifies each feature doc is referenced in CHANGELOG.md
3. Skips sprint plans and analysis docs (implementation artifacts)

**Exit codes:**
- `0` - All feature docs are in CHANGELOG
- `1` - Some docs are missing from CHANGELOG

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

### `scripts/update_version_constants.sh <version>`
Update website version constants to the new release version.

**Usage:**
```bash
.claude/skills/release-manager/scripts/update_version_constants.sh 0.5.7
```

**What it updates:**
- `docs/src/constants/version.js` - STABLE_RELEASE and ACTIVE_PROMPT

**Output:**
```
Updating docs/src/constants/version.js...
  STABLE_RELEASE: v0.5.6 → v0.5.7
  ACTIVE_PROMPT: v0.5.2 → v0.5.2
✓ Updated docs/src/constants/version.js
```

### `scripts/collect_closable_issues.sh <version> [--close] [--json]`
Find GitHub issues that can be closed with this release.

**Uses `ailang messages` GitHub integration** for efficient issue discovery.

**Usage:**
```bash
.claude/skills/release-manager/scripts/collect_closable_issues.sh 0.5.9
.claude/skills/release-manager/scripts/collect_closable_issues.sh 0.5.9 --close
.claude/skills/release-manager/scripts/collect_closable_issues.sh 0.5.9 --json
```

**What it scans:**
1. **GitHub sync** - Runs `ailang messages import-github` to sync latest issues
2. **ailang messages** - Queries local message database for GitHub-linked issues
3. Commits since last tag for issue references (Fixes #123, Closes #456, etc.)
4. CHANGELOG.md entry for the version
5. Design docs in `design_docs/implemented/vX_Y_Z/` and `design_docs/planned/vX_Y_Z/`
6. Keyword matching between message content and CHANGELOG

**Options:**
- `--close` - Actually close the issues via `gh issue close`
- `--json` - Output JSON format for including in release notes

For output examples, see [`resources/script_examples.md`](resources/script_examples.md).

**Note:** When closing issues (`--close`), the script also marks the corresponding `ailang messages` as read via `ailang messages ack`.

**Deduplication:** `ailang messages import-github` checks existing issues by number before importing - issues are never duplicated.

**Alternative: Auto-close via commits** - Instead of using `--close`, include `Fixes #123` in the release commit message. GitHub will auto-close issues when the commit is merged to the default branch.

### `scripts/close_issues_with_references.sh <version> <issue> [section]`
Close a GitHub issue with proper release references (URLs, commits, design docs).

**Usage:**
```bash
.claude/skills/release-manager/scripts/close_issues_with_references.sh 0.5.10 29
.claude/skills/release-manager/scripts/close_issues_with_references.sh 0.5.10 29 'M-STRING-CONVERT'
```

**What it does:**
1. Gets release URL and commit hash for the version
2. Extracts relevant CHANGELOG section (auto-detects or uses provided section name)
3. Finds related design doc if exists
4. Generates a comprehensive closing comment with all references
5. Prompts for confirmation, then closes the issue

**For more details:** See [`resources/issue_closure_guide.md`](resources/issue_closure_guide.md)

### `scripts/broadcast_release.sh <version> [--include-issues]`
Broadcast release notification with changelog to all projects.

**Usage:**
```bash
.claude/skills/release-manager/scripts/broadcast_release.sh 0.4.5
.claude/skills/release-manager/scripts/broadcast_release.sh 0.4.5 --include-issues
```

**Options:**
- `--include-issues` - Include list of closed/closable GitHub issues in the notification

**What it does:**
1. Extracts the changelog entry for the given version
2. Optionally collects related GitHub issues (using `collect_closable_issues.sh`)
3. Creates a structured release notification message with changelog and closed issues
4. Broadcasts to the user inbox (global notification point)
5. Projects receive notification when they check their inbox

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

### 2. Verify Implemented Design Docs (CRITICAL)

**Check that all implemented features are documented:**
```bash
.claude/skills/release-manager/scripts/check_implemented_docs.sh X.X.X
```

**This checks `design_docs/implemented/vX_Y_Z/` against CHANGELOG:**
- Every feature doc should have a CHANGELOG entry
- Sprint plans and analysis docs are skipped (implementation artifacts)
- If docs are missing from CHANGELOG, add entries before proceeding

**If docs are missing:**
- Read each missing doc to understand the feature
- Add appropriate entry to `changelogs/v0.10-current.md` under the version header
- Include: problem, solution, files changed, design doc link

### 3. Update Version in Documentation

**Run the version update script:**
```bash
.claude/skills/release-manager/scripts/update_version_constants.sh X.X.X
```

**Also update these files manually:**
- **changelogs/v0.10-current.md**: Change `## [Unreleased]` to `## [vX.X.X] - YYYY-MM-DD`
- **Note:** Root `CHANGELOG.md` is now an index file linking to themed archives in `changelogs/`
- **std/VERSION**: Change to `vX.X.X` (used by stdlib resolver for version checking)

**The script automatically updates:**
- **docs/src/constants/version.js** - Website STABLE_RELEASE and ACTIVE_PROMPT

### 4. Post-Update Verification (CRITICAL)

**Run checks AGAIN after documentation changes:**
```bash
make test
make lint
```

If either fails, fix before committing.

### 4. Commit Changes

```bash
git add changelogs/ std/VERSION docs/src/constants/version.js
git commit -m "Release vX.X.X"
```

### 5. Create and Push Git Tag

**CRITICAL: Tag MUST be on the dev branch at HEAD. Never tag a divergent commit.**

```bash
# 1. Create the tag on current HEAD
git tag -a vX.X.X -m "Release vX.X.X"

# 2. VERIFY tag is reachable from HEAD (prevents v0.9.0-style divergence)
git describe --tags --exact-match HEAD  # Must output "vX.X.X"

# 3. VERIFY binary version matches BEFORE pushing
go install -ldflags "-X main.Version=$(git describe --tags --always) -X main.Commit=$(git rev-parse --short HEAD) -X main.BuildTime=$(date -u '+%Y-%m-%d_%H:%M:%S')" ./cmd/ailang/
ailang --version  # Must show "AILANG vX.X.X"

# 4. Only push AFTER verification passes
git push origin dev
git push origin vX.X.X
```

**If `git describe` doesn't show the expected version:**
- The tag is on a different commit than HEAD — DELETE the tag and re-tag on HEAD
- `git tag -d vX.X.X` then start step 5 again
- **NEVER push a tag that diverges from the dev branch**

**What went wrong with v0.9.0:** The tag was placed on a commit not on `dev` (a WASM fix commit on a side branch). `dev` kept advancing, so `git describe --tags` couldn't find the tag as an ancestor — the binary reported `v0.8.1.1` instead of `v0.9.0`.

### 7. Monitor CI/CD

```bash
# Check recent runs
gh run list --limit 3

# Watch for completion (typically 2-3 minutes)
gh run watch
```

### 7.5. Reindex μRAG Corpora (REQUIRED)

After the tag pushes and CI publishes the release, refresh the brain corpora
that back micro-rag (μRAG). The indexer always resolves the **active prompt
version** at runtime — never pin a version anywhere — and `--reset` ensures
the new release replaces stale chunks rather than layering on top of them.

```bash
make brain-index-syntax-reset
```

**Acceptance check:**
- Output reports indexed chunk counts for `ailang-syntax`, `ailang-builtins`,
  `ailang-examples`.
- `ailang cache stats` shows non-zero counts for all three namespaces.
- Sample probe: `ailang cache search --namespace ailang-syntax --limit 1 "string interpolation"`
  returns a chunk whose `[version:vX.Y.Z]` tag matches the new release.

If the indexer fails (e.g. "could not resolve active prompt version"),
**do not skip this step** — investigate and fix before the release is
considered complete. The post-release skill will re-verify.

### 8. Collect and Close Related Issues

**Find issues that can be closed with this release:**
```bash
.claude/skills/release-manager/scripts/collect_closable_issues.sh X.X.X
```

**Review the suggested issues, then close them:**
```bash
.claude/skills/release-manager/scripts/collect_closable_issues.sh X.X.X --close
```

The script scans commits, CHANGELOG, and design docs for issue references, and matches open issues against CHANGELOG keywords.

### 9. Verify Release

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

### 10. Broadcast Release Notification

**Notify all projects about the new release:**
```bash
.claude/skills/release-manager/scripts/broadcast_release.sh X.X.X
```

This extracts the changelog entry for the version and broadcasts it to the user inbox.
External projects will see the notification when they check their inbox.

**What gets broadcast:**
- Version number and release date
- Full changelog section (Added, Changed, Fixed, etc.)
- Link to GitHub release page

### 11. Handle CI Failures

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

### 12. Summary

Show user:
- ✓ Version vX.X.X released
- ✓ Git tag created
- ✓ Release URL: https://github.com/sunholo-data/ailang/releases/tag/vX.X.X
- ✓ CI workflow status
- ✓ Related GitHub issues closed (if any)
- ✓ Release notification broadcast to projects
- **Next step**: Run `post-release` skill to update benchmarks and dashboard

## Resources

### Release Checklist
See [`resources/release_checklist.md`](resources/release_checklist.md) for complete step-by-step checklist.

### Issue Closure Guide
See [`resources/issue_closure_guide.md`](resources/issue_closure_guide.md) for how to properly close GitHub issues with:
- Release URLs and commit references
- CHANGELOG excerpts
- Design doc links
- Best practices for closing comments

## Prerequisites

- Working directory must be clean (no uncommitted changes)
- Current branch should be `dev` or `main`
- All tests must pass (`make test`)
- All linting must pass (`make lint`)
- No files exceed 800 lines (`make check-file-sizes`)

## Version Format

Semantic versioning: `MAJOR.MINOR.PATCH`
- Examples: `0.0.9`, `0.1.0`, `1.0.0`

## Prompt Version Consistency

After release, verify both prompt registries are consistent:
- `prompts/versions.json` — syntax teaching prompts (`ailang prompt`)
- `prompts/devtools/versions.json` — dev tools reference (`ailang devtools-prompt`)

Both are embedded in the binary via `//go:embed all:prompts`. New toolchain features should be reflected in the devtools prompt.

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
