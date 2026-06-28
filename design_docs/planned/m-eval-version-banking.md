# M-EVAL-VERSION-BANKING: per-release eval banking so the loop can see its own progress

**Status:** In Progress (2026-06-28). Lane: **AILANG-side harness change** (per [PROGRAM.md](../PROGRAM.md) — the loop must be able to measure movement across releases).

## Problem

The nightly rotation runs with `--skip-existing`, whose existing-check globs
`{benchmark}_{lang}_{model}_*.json` (eval_suite.go:565) — **no version anywhere**. So once a
`(model, benchmark, lang)` combo is banked, it is skipped *forever*, regardless of the AILANG version.
A new release (v0.26.0, Jun 26) does **not** trigger a re-eval → the banked pass-rates are frozen.
The banked motoko number (~69%) is pre-Jun-19, never refreshed (memory: os-rolling data is stale). The
flywheel cannot see its own progress.

## Fix

Namespace banked output by the **AILANG build version**. A `--bank-by-version` flag appends
`internal/version.Version` (e.g. `v0.26.0-22-g9ebbda9ed` — stable between `make install`s) to the
output dir **once, right after flag parse**, so the metrics writer, the `--skip-existing` glob, and the
rotation summarizer all operate on `eval_results/<out>/<version>/` consistently (they all read the same
`*outputDir`).

Result:
- A new build (new code — a release, or a `make install` of a fix) → empty version dir → **re-evals
  from scratch → fresh data**.
- History **accumulates per build/release**: `os-rolling/v0.25.0/`, `os-rolling/v0.26.0-22-g…/`, … One
  banked set per build. Cross-release trend = read each `<version>/summary.json`.
- `--skip-existing` still accumulates the chunked rotation **within** a version (the version is stable
  between installs, so a full sweep completes before it changes).

## Granularity choice

Namespace by the **full build version** (git-describe), not just the release tag — so a `make install`
of an intra-release fix also gets fresh data (we re-eval when the code actually changes, which is the
point). Clean releases are clean tags (`v0.26.0`); dev builds carry the `-N-g<sha>` suffix. A
release-only view groups by the tag prefix. `--skip-existing` without `--bank-by-version` is unchanged.

## Changes
- `cmd/ailang/eval_suite.go`: add `--bank-by-version`; append the version to `*outputDir` after parse.
- `tools/launchd/os-rotation-filler.sh`: pass `--bank-by-version`.
- *Follow-up:* a cross-version trend view (`<out>/*/summary.json`) — the per-release movement chart.

## Verify
- `make test` green. `--dry-run --bank-by-version` prints the version-namespaced output dir. The next
  rotation cycle banks under the current version; a version bump produces a fresh sweep.
