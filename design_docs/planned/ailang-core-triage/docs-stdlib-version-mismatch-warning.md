# `ailang docs` reads a stale stdlib root silently — no version-mismatch warning like `run`

- **Date**: 2026-09-28
- **Class**: bug
- **Recommend**: design-doc
- **Searched**: `numerics-quick`, `STDLIB_PATH`, `stdlib version mismatch`, `docs stdlib`, triage dir listing
- **Related**: `design_docs/implemented/v0_47_0/m-numerics-vec-array-ingest.md` (the landed M-NUMERICS-QUICK work this report verifies — closed, not duplicated by this triage); `design_docs/planned/v0_44_0/m-stdlib-root-resolution.md` (one stdlib root per command — root *resolution* is done, the docs *version warning* is not in scope there: it only discusses `checkStdlibVersion` applying to on-disk roots for run/check)

<One paragraph> The reporter's environment had `AILANG_STDLIB_PATH` pointing at a
checkout 186 commits behind the v0.47.0 binary. `ailang run` correctly warned
("stdlib version mismatch: expected v0.47.0, found v0.41.0 at …") but
`ailang docs std/list` printed the stale module list with no warning, which
nearly produced a false "the release does not contain the fixes" bug report.
Mechanism confirmed in code: the run/check path version-checks via
`internal/loader/stdlib_resolver.go` (`StdlibResolver.checkStdlibVersion`,
compares base semver of the root's `VERSION` file against `loader.BinaryVersion`,
warns once per process unless `--strict`/`AILANG_NO_VERSION_WARNINGS`), but
`cmd/ailang/docs.go` resolves its root directly via `stdlibroot.Resolve("")`
(docs.go:94) and never reads `VERSION`. The check lives unexported on
`StdlibResolver`, and the expected version is a package var set by `main.go`
(`loader.BinaryVersion`), so `docs` cannot reuse it as-is. Fix options differ in
a way someone could disagree with: (a) move/duplicate the check into
`internal/stdlibroot` so every root consumer benefits vs (b) export a helper
from `internal/loader` and call it in `docs.go` only — plus warn-vs-error and
whether `--list`/`--all-functions` are covered. CLAUDE.md principle 3 (systemic
fixes — audit the pattern before patching one command) points at (a), but that
is a design decision, not a mechanical edit.

## What this report contains (three notes, one actionable)

1. **Withdrawn by the reporter** — the original report's `std/array.set` O(n)
   copy-on-write item no longer matters once native `axpy` exists (per-element
   set left the hot path). Confirmed still absent on v0.47.0 and no longer
   wanted. No action; noted so the backlog doesn't resurrect it.
2. **The actionable kernel** — the docs-command version-mismatch warning
   classified above. Failure mode: a developer concludes a shipped feature does
   not exist. This is the bug row of the rubric; the fix is multi-file with
   competing designs, hence design-doc.
3. **Not actionable for AILANG** — logistic regression (1,000 rows × 30 epochs,
   11.2 s to test) overfits at 64.0% test without regularisation; k-NN (70.3%)
   stays the production second opinion in DuckDB. A finding about the model,
   enabled by the 55× speedup; no language or stdlib change requested.

## Verification value (for the record)

Same code, same machine: v0.41.0 hand-rolled helpers 3m36s → v0.47.0 same code
2m27s (1.5×) → v0.47.0 native primitives (`std/list.range`, `std/embedding.dot`
/`.axpy`, `std/list.nth_or`/`.maximumFloat`) 3.9s (55×). Per SGD step (768×5)
84 ms → 0.34 ms (247×); JSON load+eval of 19 MB / 1,252×768 14.4 s → 3.08 s
(4.7×). Accuracy identical to 3 s.f. across all runs. This closes
`inbox_1790498369833_8034dce8` and its correction
`inbox_1790530530325_58bd9fdb`; nothing further owed on M-NUMERICS-QUICK
beyond item 2 above.