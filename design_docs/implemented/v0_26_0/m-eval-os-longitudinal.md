# M-EVAL-OS-LONGITUDINAL: Per-Release Local-Model Eval Trend

**Status**: ✅ Implemented (v0.26.0) — per-release local-model eval trend
**Target**: v0.25.x
**Priority**: P1 — answers the core question "does each AILANG release move the needle for local models?"
**Estimated**: ~250 LOC across 3 milestones; ~1 day
**Owner**: sunholo (eval rig)

## Problem

We run a continuous local-rig rotation (opencode + pi + motoko on local qwen, 4
langs, 3 trials) that publishes `docs/static/benchmarks/os/latest.json` and shows
a model × harness × language table on the website. But it **cannot show
release-over-release evolution**, which is the whole point — AILANG is designed
for AI code synthesis, and each release should make local models *better at
AILANG*. Today:

1. **Not tied to the AILANG version.** `os/latest.json` is labeled by date
   (`version: "rolling-20260616"`), never `std/VERSION`. No number knows which
   AILANG release it measured.
2. **No per-version history.** Only `latest.json` exists — no archive of past
   releases, so no trend.
3. **Rotation decoupled from releases.** The rotation uses `--skip-existing`, so
   after a full pass the data **freezes** and never re-runs — even across an
   AILANG release. `post-release` re-runs *cloud* baselines but never touches the
   local rig.

Net: if we ship v0.26.0, the rig keeps showing v0.25.0-era local numbers with no
way to see "v0.25 → v0.26 moved local AILANG 65% → 72%."

## Goal

Track local-model eval performance **keyed by AILANG version**, re-run on each
release, archived so the website can chart **AILANG version (x) vs local-model
pass-rate (y)** per harness/model. Older/retired models freeze at their last
version's numbers; active models re-run each release.

## Design — three pieces

### M1 — Version-tag the OS JSON (~50 LOC)
`eval-publish` (the command that writes `os/latest.json`) stamps the AILANG
version from `std/VERSION` into the JSON as a first-class field:
```json
{ "ailang_version": "v0.25.0", "generated": "2026-06-16", "trials": 3,
  "languages": [...], "rows": [...] }
```
Keep the date (`generated`) for "when measured". The rotation already passes a
label; have eval-publish read `std/VERSION` itself (canonical source) so every
publisher is consistent. Unit-test the version is read + embedded.

### M2 — Release snapshot + reset (~70 LOC)
A script `tools/os-release-snapshot.sh <version>` (invoked by the `post-release`
skill, new step) that on each release:
1. **Snapshot**: append the current `os/latest.json` (now version-tagged) as an
   entry in a new history file `docs/static/benchmarks/os/history.json` (an array
   of `{ailang_version, generated, rows}`), deduped by `ailang_version` (re-runs
   of the same version replace).
2. **Reset active models**: delete the **active**-model result files from
   `eval_results/rotation/os-rolling/` (the models in the rotation's MODELS list)
   so the rotation re-runs them against the new release. **Retired** models
   (e.g. qwen3.5) are left untouched → frozen at their last version.
3. Re-publish `os/latest.json` (now reflecting the new in-progress version).

Add the step to `.claude/skills/post-release/SKILL.md` (after the dashboard step)
so it fires every release. Idempotent + safe to re-run.

### M3 — Website trend view (~130 LOC)
Extend `docs/src/components/BenchmarkExplorer/index.jsx` (or a sibling component)
to fetch `os/history.json` and render a **version-trend chart**: x = AILANG
version, y = pass-rate, one line per (model, harness, language=ailang by default,
language-switchable). Keep the existing current-snapshot table from
`os/latest.json` as the "live" view; add the trend as a second view/tab. Empty or
single-version history degrades gracefully (table only).

## Data model summary

| file | role | updated |
|---|---|---|
| `os/latest.json` | current/live snapshot (table) | every rotation cycle |
| `os/history.json` | array of per-version snapshots (trend) | once per release (M2) |

## Acceptance criteria

- [ ] `os/latest.json` contains `ailang_version` matching `std/VERSION` (M1)
- [ ] `eval-publish` unit test asserts the version field is embedded (M1)
- [ ] `os-release-snapshot.sh v0.25.0` appends a v0.25.0 entry to `history.json`,
      clears active-model rolling files, leaves retired-model (qwen3.5) files,
      and re-publishes latest.json (M2)
- [ ] post-release SKILL.md documents the new step (M2)
- [ ] Explorer renders a version-trend chart from `history.json`; graceful with
      0/1 versions; `npm run build` (or component) succeeds (M3)
- [ ] First trend data point = v0.25.0 (the snapshot we take when wiring this)

## Axiom compliance

| Axiom | Score | Justification |
|---|---|---|
| A7 Machines First | +2 | Directly measures whether AILANG releases improve AI (local) code synthesis |
| A2 Replayability | +1 | Version-keyed, archived snapshots are reproducible baselines |
| A9 Cost Visibility | 0 | Reuses the $0 local rotation |
| others | 0 | — |

**Net +3 → Proceed.**

## Out of scope / follow-ups
- Cloud-model version longitudinal (the cloud leaderboard already preserves
  history separately).
- Statistical confidence bands on the trend (N=3 trials; add later if noisy).
- Auto-detecting a version bump mid-rotation (we drive it from the release step,
  not by polling std/VERSION).

## References
- Current publisher: `ailang eval-publish` → `os/latest.json`
- Rotation: `tools/launchd/os-rotation-filler.sh` (MODELS = active models)
- Predecessor context: [[motoko-strategic-goal]] (the motoko-vs-pi-vs-opencode KPI this trends)
