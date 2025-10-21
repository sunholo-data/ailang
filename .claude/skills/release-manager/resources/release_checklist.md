# Release Checklist

## Pre-Release (REQUIRED)

- [ ] All tests pass (`make test`)
- [ ] All linting passes (`make lint`)
- [ ] No files exceed 800 lines (`make check-file-sizes`)
- [ ] Working directory is clean (`git status --short` shows nothing)
- [ ] Current branch is `dev` or `main`

## Version Updates

- [ ] README.md: Update "Current Version: vX.X.X"
- [ ] docs/reference/implementation-status.md: Update "Current Version: vX.X.X"
- [ ] CHANGELOG.md: Change `## [Unreleased]` to `## [vX.X.X] - YYYY-MM-DD`

## Post-Update Verification (REQUIRED)

- [ ] Tests still pass after documentation changes (`make test`)
- [ ] Linting still passes after documentation changes (`make lint`)

## Git Operations

- [ ] Stage changes: `git add README.md CHANGELOG.md docs/reference/implementation-status.md`
- [ ] Commit: `git commit -m "Release vX.X.X"`
- [ ] Create annotated tag: `git tag -a vX.X.X -m "Release vX.X.X"`
- [ ] Push tag: `git push origin vX.X.X`
- [ ] Push commit: `git push`

## CI/CD Monitoring

- [ ] Check CI status: `gh run list --limit 3`
- [ ] Verify builds pass on all platforms (Linux, macOS, Windows)
- [ ] Wait for release workflow to complete (~2-3 minutes)

## Release Verification

- [ ] Release created: `gh release view vX.X.X`
- [ ] All platform binaries present:
  - [ ] ailang-darwin-amd64.tar.gz (macOS Intel)
  - [ ] ailang-darwin-arm64.tar.gz (macOS Apple Silicon)
  - [ ] ailang-linux-amd64.tar.gz (Linux)
  - [ ] ailang-windows-amd64.zip (Windows)
- [ ] Release is published (not draft)
- [ ] Release notes are present

## If CI Fails

- [ ] Check logs: `gh run view <run-id> --log-failed`
- [ ] Fix issues (likely formatting or linting)
- [ ] Commit fixes with clear message
- [ ] Push again
- [ ] Verify all checks pass

## Final Steps

- [ ] Confirm release URL: https://github.com/sunholo-data/ailang/releases/tag/vX.X.X
- [ ] Note next step: Run post-release skill for benchmarks and dashboard

## Version Format

Semantic versioning: `MAJOR.MINOR.PATCH`
- Examples: `0.0.9`, `0.1.0`, `1.0.0`
