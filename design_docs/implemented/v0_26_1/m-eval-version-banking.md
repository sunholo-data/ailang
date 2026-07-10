# M-EVAL-VERSION-BANKING: per-release eval banking so the loop can see its own progress

**Status:** ✅ Implemented (v0.26.1) — `--bank-by-version` shipped. Lane: AILANG-side harness change.

## Problem

The nightly rotation runs with `--skip-existing`, whose existing-check globs
`{benchmark}_{lang}_{model}_*.json` (eval_suite.go:565) — **no version anywhere**. So once a
`(model, benchmark, lang)` combo is banked, it is skipped *forever*, regardless of the AILANG version.
A new release (v0.26.0, Jun 26) does **not** trigger a re-eval → the banked pass-rates are frozen.
The banked motoko number (~69%) is pre-Jun-19, never refreshed (memory: os-rolling data is stale). The
flywheel cannot see its own progress.

## Fix

Namespace banked output by the **AILANG release tag**. A `--bank-by-version` flag appends the release
tag derived from `internal/version.Version` (e.g. `v0.26.0`) to the
output dir **once, right after flag parse**, so the metrics writer, the `--skip-existing` glob, and the
rotation summarizer all operate on `eval_results/<out>/<version>/` consistently (they all read the same
`*outputDir`).

Result:
- A new build (new code — a release, or a `make install` of a fix) → empty version dir → **re-evals
  from scratch → fresh data**.
- History **accumulates per build/release**: `os-rolling/v0.25.0/`, `os-rolling/v0.26.0/`, … One
  banked set per build. Cross-release trend = read each `<version>/summary.json`.
- `--skip-existing` still accumulates the chunked rotation **within** a version (the version is stable
  between installs, so a full sweep completes before it changes).

## Granularity choice — RELEASE only

Namespace by the **release tag**: `releaseTag()` strips the git-describe `-<N>-g<sha>` (and `-dirty`)
metadata, so `v0.26.0-26-g9249a66bf` → `v0.26.0` (pre-release tags like `v0.26.0-rc1` are preserved).
The rotation therefore re-evals **only on a new release tag** — a dev `make install` under the same tag
reuses that tag's banked set (`--skip-existing` skips it). This is the intended cadence: **per-release
movement, no churn on every dev commit**. An intra-release fix we specifically want to measure gets a
**manual forced re-eval** (clear that tag's bucket, or a one-off `--output`), not an automatic sweep.
`--skip-existing` without `--bank-by-version` is unchanged.

## Changes
- `cmd/ailang/eval_suite.go`: add `--bank-by-version`; append the version to `*outputDir` after parse.
- `tools/launchd/os-rotation-filler.sh`: pass `--bank-by-version`.
- *Follow-up:* a cross-version trend view (`<out>/*/summary.json`) — the per-release movement chart.

## Verify
- `make test` green. `--dry-run --bank-by-version` prints the version-namespaced output dir. The next
  rotation cycle banks under the current version; a version bump produces a fresh sweep.
