# Local dependency override (Go-style `replace`) for development vs registry resolution

- **Date**: 2026-09-15
- **Class**: feature
- **Recommend**: design-doc
- **Searched**: `override`, `replace`, `ailang.local`, `[replace]`, `path dep`, `dev-dependenc`, `rev` across `design_docs/`; confirmed `rev` is the only manifest override today (`internal/pkg/gitcache.go`, `Resolve(gitURL, tag, rev, subdir)`); no doc covers local overrides.
- **Estimate**: n/a (design-doc)

The report proposes a new, untracked manifest surface — either a gitignored
`ailang.local.toml` or a `[replace]` table gated on `AILANG_DEV=1` — plus
resolve-time override semantics and lock/visibility behaviour. That is row 3
head-on: the reporter themselves offers two acceptable mechanisms (local file
vs env-gated table) with different trade-offs (leakage into teammates, silent
override risk, lock recording rules), so a ruling is needed rather than a
mechanical edit. It also changes a public file format (`ailang.toml`, the
lock) and spans resolver, lock writer, and CLI reporting
(`internal/pkg/resolver.go`, lock tooling) — multi-file, semantics-sensitive.
Mark's rule (registry at runtime, local paths only for the package developer)
gives the design doc a clear target but not a single implementation.

Note: the related bug — lock files carrying absolute paths for `path` deps —
is tracked separately in `design_docs/planned/ailang-core-backlog.md` (2026-09-15
row, `direct-fix`, pointing at `filepath.Abs()` at resolver.go:127 per the
`m-pkg-lock-portability.md` doc, now relocated). Do not fold it into this
design; this feature would make the override path moot for the Daneel use case,
but the lock-portability fix is independent.

A prior triage already appended this item to the shared backlog
(`design_docs/planned/ailang-core-backlog.md:16`) as `design-doc`; this file
is the per-report artifact for the same report.
