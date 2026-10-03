# check-file-sizes and check-boundaries are blind outside internal/ and cmd/

- **Date**: 2026-10-03
- **Class**: bug
- **Recommend**: design-doc
- **Searched**: `find internal cmd`, `go list` in `make/code-health.mk`; `apiserver`, `serveapi`, `CORE_PKGS|DASHBOARD_PKGS|CORE_SURFACE_PKGS` in `scripts/check_boundaries.sh`; `check-file-sizes`, `check-boundaries`, `m-arch-boundaries` across `design_docs/`
- **Source**: sunholo-data/ailang#584 (mission iteration 138)

## Status at origin/dev `790169359` — unchanged since filing

- `make/code-health.mk:167` still enumerates `find internal cmd -name "*.go"` (and the report targets at :288–318 do the same).
- `scripts/check_boundaries.sh:43–55` still iterates three fixed arrays; `grep -c 'apiserver\|serveapi'` → **0**.

## Why design-doc rather than direct fix

Widening either gate turns it red on day one, so each needs a scope ruling, not just a code change:

1. **File sizes.** Non-test Go files over 800 lines outside `internal/`+`cmd/` today: `scripts/verify_examples.go` (968), `tools/build-snapshot/main.go` (1198), and eight vendored files under `third_party/goopus/` (821–1642). Decide: exclude `third_party/` (vendored, not ours to split) — almost certainly yes; split or explicitly exempt the two first-party files; enumerate from `go list ./...` minus excludes, with an anti-vacuity floor (enumerated count must exceed a known minimum) and a self-test fixture outside `internal/`+`cmd/`.
2. **Boundaries.** `internal/apiserver/server.go` and `internal/apiserver/load_project.go` import `internal/pipeline` directly. If `apiserver` joins `DASHBOARD_PKGS`, Rule 2 ("dashboard must reach the compiler only via `internal/embed`") fails immediately. Decide which layer `apiserver` and the public `serveapi/` facade belong to — a dashboard app (then route through `embed`), or a sanctioned bridge like `embed` (then say so in ARCHITECTURE.md and the script) — and derive package sets from `go list` with a check that every first-party package is classified (an unclassified package fails the gate, closing the vacuous-pass class the script's own header warns about).

Both are small once ruled (~1 day), and the ruling is the point: the issue's harm is that both gates are quoted as acceptance evidence for code they never scan.
