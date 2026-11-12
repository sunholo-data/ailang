### Fixed — Go dependency graph was stale; Dependabot Go alerts read the wrong module list (2026-11-12)

The CI `dependency-submission` job used `actions/go-dependency-submission`, which failed on every
push with `assertion failed: expected no more than one package in cache with namespace+name`
(upstream issue #74, still in v2.0.3). `continue-on-error: true` hid it, so GitHub's dependency
graph kept an old Go snapshot (it still listed `github.com/ollama/ollama v0.13.4`, removed in
#2206). The new `tools/ci/go_dependency_snapshot.sh` builds the snapshot from `go list -deps all`
and `go mod graph`, keyed by full module path, and POSTs it to the dependency-graph snapshots API
under the action's detector name and correlator so it supersedes the stale snapshot. A failed or
`INVALID` submission now fails the job instead of passing silently.

### Removed — `headless-runner` skill and the `UI build gate` stub workflow (2026-11-12)

Owner decisions (Mark, 2026-11-12). The `headless-runner` skill (2025-10, unlisted in the session
skill listing) and its `.agents` mirror are deleted; its resources taught the retired
`ailang agent` command and accounted for almost all of the stale command citations
`make simplicity-audit` reported. `.github/workflows/dashboard-ui-build.yml` (job `ui-gate`, check
`UI build gate`) is deleted: the dashboard UI is gone and Mark removed the check from `dev`'s
required set, which is now `test`, `lint`, `build`, `docs-gate`, `checks`.

### Changed — platform split timing decided: post-1.0 (v1.1) (2026-11-12)

`design_docs/planned/v1_1_0/m-platform-split.md` records D4 as decided by Mark on 2026-11-12: every
milestone, M0 included, lands after v1.0.0. D1–D3 stay open until v1.1 starts.
