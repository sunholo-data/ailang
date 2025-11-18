---
description: Manage AILANG releases with pre-flight checks and verification.
---

# AILANG Release Workflow

This workflow manages the release process for AILANG, ensuring all quality checks pass before tagging and publishing.

## Prerequisites
- Clean working directory (no uncommitted changes).
- Current branch should be `dev` or `main`.
- `gh` CLI authenticated with correct user (`MarkEdmondson1234`).

## Steps

1.  **Verify GitHub Authentication**
    Ensure you are authenticated as the correct user for the `sunholo-data` organization.
    ```bash
    gh auth status
    # If wrong user: gh auth switch --user MarkEdmondson1234
    ```

2.  **Pre-Release Checks**
    Run the automated pre-release verification script. This checks tests, linting, and file sizes.
    ```bash
    .claude/skills/release-manager/scripts/pre_release_checks.sh
    ```
    **Stop if this fails.** Fix issues before proceeding.

3.  **Update Version**
    If checks pass, update the version number in the following files:
    - `README.md`
    - `docs/reference/implementation-status.md`
    - `CHANGELOG.md` (Move `[Unreleased]` to new version)
    - `std/VERSION`

4.  **Post-Update Verification**
    Run tests again to ensure documentation updates didn't break anything.
    ```bash
    make test
    make lint
    ```

5.  **Commit and Tag**
    ```bash
    git add README.md CHANGELOG.md docs/reference/implementation-status.md std/VERSION
    git commit -m "Release v[VERSION]"
    git tag -a v[VERSION] -m "Release v[VERSION]"
    git push origin v[VERSION]
    git push
    ```

6.  **Monitor CI/CD**
    Watch the GitHub Actions run to ensure the release build succeeds.
    ```bash
    gh run watch
    ```

7.  **Verify Release**
    Once CI completes, verify the release artifacts exist.
    ```bash
    .claude/skills/release-manager/scripts/post_release_checks.sh [VERSION]
    ```

## Rollback
If CI fails after tagging:
1.  Fix the issue.
2.  Delete the local and remote tag: `git tag -d v[VERSION] && git push --delete origin v[VERSION]`
3.  Commit the fix.
4.  Re-tag and push.
