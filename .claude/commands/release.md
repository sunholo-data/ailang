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

2. **Verify tests and linting BEFORE making any changes:**
   - Run `make test` to ensure all tests pass
   - Run `make lint` to ensure code quality
   - **CRITICAL**: If either fails, STOP and fix issues before proceeding
   - Do NOT proceed with version updates if tests or linting fail

3. **Update version in documentation:**
   - README.md: Update "Current Version: vX.X.X"
   - docs/reference/implementation-status.md: Update "Current Version: vX.X.X"
   - CHANGELOG.md: Change `## [Unreleased]` to `## [v$1] - $(date +%Y-%m-%d)`

4. **Verify tests and linting AGAIN after documentation changes:**
   - Run `make test` to ensure documentation changes didn't break anything
   - Run `make lint` to ensure all files pass linting
   - **CRITICAL**: If either fails, fix issues before committing

5. **Commit changes:**
   - Stage README.md, CHANGELOG.md, and docs/reference/implementation-status.md
   - Commit with message: "Release v$1"

6. **Create and push git tag:**
   - Create annotated tag: `git tag -a v$1 -m "Release v$1"`
   - Push tag: `git push origin v$1`

7. **Push commit:**
   - Push to remote: `git push`

8. **Monitor CI/CD:**
   - Run `gh run list --limit 3` to check CI status
   - Verify builds pass on all platforms (Linux, macOS, Windows)
   - Wait for release workflow to complete (typically 2-3 minutes)

9. **Verify Release:**
   - Run `gh release view v$1` to verify release was created successfully
   - Check that release includes all platform binaries:
     - ailang-darwin-amd64.tar.gz (macOS Intel)
     - ailang-darwin-arm64.tar.gz (macOS Apple Silicon)
     - ailang-linux-amd64.tar.gz (Linux)
     - ailang-windows-amd64.zip (Windows)
   - Verify release is published (not draft)
   - Check release notes are present

10. **Monitor for CI Failures:**
    - Run `gh run list --workflow=CI --limit 3` to check for any failures
    - If CI fails after push:
      - Check logs: `gh run view <run-id> --log-failed`
      - Fix issues (likely formatting or linting)
      - Commit fixes with clear message
      - Push again
    - Verify all checks pass on the release commit

11. **Summary:**
    - Confirm version v$1 released
    - Show git tag details
    - Show release URL: https://github.com/sunholo-data/ailang/releases/tag/v$1
    - Show CI workflow status

12. **Update eval benchmarks** (NEW: M-EVAL-LOOP documentation)
    - Run eval baseline to capture current performance: `make eval-baseline MODEL=claude-sonnet-4-5 LANGS=ailang`
    - Compare to previous version: `ailang eval-compare eval_results/baselines/v<prev> eval_results/baselines/v$1`
    - Update CHANGELOG.md with benchmark results:
      - Add section "### Benchmark Results (M-EVAL)"
      - Include success rate: "X/10 benchmarks passing (X%)"
      - List improvements: "✓ Fixed: <benchmark_ids>"
      - List regressions: "✗ Regressed: <benchmark_ids>" (if any)
      - Show comparison: "+X% improvement" or note if regression
    - Store baseline: Already saved to `eval_results/baselines/v$1/` by make target
    - Commit baseline results: `git add eval_results/baselines/v$1/ && git commit -m "Add eval baseline for v$1"`

13. **Update design docs**
    - Move design docs used into design_docs/implemented/
    - Update design docs used with what was implemented
    - If any features were missed or pushed to a future release, ensure they have new design_docs ready in design_docs/planned/

14. **Update public docs**
    - Ensure prompt in prompts/ reflects latest changes for AILANG syntax instruction
    - Ensure website docs in docs/ includes latest changes and is updated to remove any old references
    - Make sure latest examples are reflected on the website
    - Update docs/guides/evaluation/ with new benchmark results if significant improvements


## Version Format

Version should be in semantic versioning format: `MAJOR.MINOR.PATCH`
- Examples: `0.0.9`, `0.1.0`, `1.0.0`

## Prerequisites

- Working directory must be clean (no uncommitted changes)
- Current branch should be `dev` or `main`
- **CRITICAL**: All tests must pass before release
- **CRITICAL**: All linting must pass before release
- **IMPORTANT**: Run tests and linting TWICE:
  1. Before making version changes (to ensure current state is clean)
  2. After making version changes (to ensure changes didn't break anything)
