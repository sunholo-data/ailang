# M-EVAL-ELO-PRIORITY-ROTATION: ELO-Driven Benchmark Prioritization for the OS Rotation Filler

**Status**: Planned
**Target**: v0.30.x (tooling-only; no compiler/runtime change)
**Priority**: P1
**Estimated**: 0.5 day
**Dependencies**: M-EVAL-OS-FRONTIER-COVERAGE (AILANG-first pass, shipped), M-EVAL-DASHBOARD-REDESIGN (ratings block in latest.json, shipped)

## PROGRAM.md Routing

**Lane: extension** (rig tooling). The change lives entirely in `tools/launchd/os-rotation-filler.sh`
plus one new small helper; no core compiler package is touched, and the ELO fit itself
(`internal/eval_harness/ratings.go`, `internal/eval_analysis/ratings_export.go`) is reused as-is.

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

This is rig-tooling (shell + published JSON), not a language feature; axioms are scored for the
harness behavior it shapes.

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Benchmark order becomes a pure function of (tier files, ratings JSON, cursor) instead of `ls` order; a given ratings snapshot always yields the same order |
| A2: Replayability | 0 | No trace impact |
| A3: Effect Legibility | 0 | No effect-system impact |
| A4: Explicit Authority | 0 | No new access; reads a repo-local JSON already produced by the rig |
| A5: Bounded Verification | 0 | No impact |
| A6: Safe Concurrency | 0 | No impact |
| A7: Machines First | +1 | Rig time is redirected to benchmarks that carry discriminative signal for the data-led loop — more information per GPU-hour |
| A8: Minimal Syntax | +1 | No new syntax; env-var knobs only, default-on with a kill switch |
| A9: Cost Visibility | +1 | Saturated re-confirmation cost is made explicit (1 trial vs 3) and logged per cycle |
| A10: Composability | +1 | Composes with the existing cursor, `--skip-existing`, per-version banking, and lap-marker hand-off without changing their semantics |
| A11: Structured Failure | 0 | Missing/unparseable ratings degrades to today's behavior, loudly logged |
| A12: System Boundary | 0 | No boundary change |

**Net Score: +5** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Not optimizing for human convenience over machine analysis

## Problem Statement

The local-rig rotation filler (`tools/launchd/os-rotation-filler.sh`, launchd every 2700s)
round-robins its benchmark picks in **filesystem `ls` order** — both the AILANG-first full-tier
pass (`BENCHES_FULL`, cursor `.ailang-full-cursor`, `FULL_CHUNK=3`/cycle) and the cross-language
pass. Every benchmark gets equal rig time regardless of whether it can still tell models apart.

Meanwhile the published dashboard already knows which benchmarks carry signal:
`docs/static/benchmarks/latest.json` → `ratings.agent.benchmarks[]` has a per-benchmark
`saturated` flag (ELO band "Trivial", i.e. fitted rating < 1300 in
`internal/eval_analysis/ratings_export.go:126` / `internal/eval_harness/ratings.go:88`).
**As of 2026-07-20, 30 of 56 agent-mode benchmarks are saturated** — verified against the
checked-in `latest.json` (top-level `ratings.agent` and `ratings.agent.byLang.ailang` currently
agree: 30/56, because agent-mode runs are AILANG-only).

**Current State:**
- With ~56+ full-tier benchmarks at 3/cycle and a ~45-min launchd interval (minus blackout,
  nightly yields, and eval wall time), a full AILANG lap takes days. Under `ls` order, roughly
  half of every lap's early cycles are spent re-confirming benchmarks every model already
  passes ("Trivial" band) while discriminating benchmarks wait their alphabetical turn.
- Mark (2026-07-20): "we need the evals to run across the benchmarks that count and have
  signal — that's what the ELO is supposed to help with."

**Impact:**
- Per-release signal coverage (the numbers that move the dashboard, ELO fits, and A/B
  verdicts) completes days later than it needs to.
- Saturated benchmarks burn 3 trials × 3 harnesses each per version for near-zero information.

## Goals

**Primary Goal:** The rotation filler orders each version's lap so non-saturated ("signal")
benchmarks are banked first, and saturated benchmarks get a cheap once-per-version
re-confirmation run afterwards (which is exactly what lets them un-saturate after a regression).

**Success Metrics:**
- First N cycles of a fresh version lap pick only non-saturated (or unrated) benchmarks until
  the signal set is exhausted.
- Saturated benchmarks still run once per version (re-confirmation), at reduced trial count.
- Ratings JSON missing/corrupt ⇒ behavior identical to today (ls-order round-robin), with a
  logged warning — no silent behavior change, no crash.

## Key Facts (verified 2026-07-20)

These are the load-bearing premises, each checked against the working tree:

1. **Where the saturated set lives:** `docs/static/benchmarks/latest.json` →
   `ratings.<mode>.benchmarks[] = {id, elo, band, saturated, passRate}`, produced by
   `buildRatingsBlock` → `fitLeaderboard` (`internal/eval_analysis/ratings_export.go`), where
   `saturated := band == "Trivial"` and `Band(r) == "Trivial" ⇔ r < 1300`
   (`internal/eval_harness/ratings.go:85-98`). A per-language sub-map
   `ratings.<mode>.byLang.<lang>` exists with the same shape.
2. **Local trials DO feed the fit — no separate local fit is needed.** The filler's own step 8
   runs `tools/publish-unified-dashboard.sh`, which calls
   `ailang eval-report <cloud baseline> <ver> --merge <rotation dir>`; merged results flow into
   the same `dashboard.Ratings = buildRatingsBlock(...)` (`internal/eval_analysis/export_json.go:499`).
   So `latest.json` ratings are cloud+local blended and refresh every filler cycle once a cloud
   baseline exists for the version. (The standalone `tools/eval-elo` CLI is a separate
   render/persist path into observatory.db; it is NOT the source the filler should read.)
3. **Closed loop:** filler banks trials → step 8 re-fits ratings into `latest.json` → next
   cycle's priority order reflects them. A saturated benchmark that fails its re-confirmation
   run drops out of the Trivial band at the next fit and re-enters the signal set — for the
   *next* version's lap (within a version, `--skip-existing` means it already banked).
4. **`jq` is available in the launchd rig context** — `tools/launchd/nightly-lang-eval.sh`
   already depends on it.
5. **Per-version banking:** both passes bank under `eval_results/rotation/os-rolling/<ver>/`
   with `--skip-existing`, and the lap marker (`.ailang-full-lapped`) hands rig time to the
   cross-language pass after one full lap. This design keeps all of that intact.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Reorder-only, not skip: saturated benchmarks stay in the lap (tail position) | Preserves per-version full coverage + the un-saturation path; "occasional re-confirmation" falls out of per-version banking for free | human (Mark's directive interpreted) | design | med |
| Read saturation from `latest.json` (`ratings.agent`, preferring `byLang.ailang` for the AILANG pass) | It's the authoritative, self-refreshing, cloud+local-blended fit; avoids a second fitting path (observatory.db / eval-elo CLI) | human | design | low |
| Saturated re-confirmation runs at `--trials 1` (vs 3 for signal) | Cuts ~2/3 of saturated rig cost; 1 failing trial is enough to move the fit and un-saturate next version | agent (knob `OS_FILLER_SAT_TRIALS`, default 1) | design | low |
| Unrated benchmarks (no entry in ratings) count as SIGNAL | New benchmarks have unknown difficulty — they are exactly what needs trials first | agent | design | low |
| Fail-open to current behavior when ratings are unreadable | The rig must never wedge on a missing/renamed JSON key (see [[project_mission_unbounded_wait_wedges_loop]] class of failures) | agent | design | low |
| Cursor semantics unchanged (index into the *reordered* list) | `--skip-existing` makes mid-lap reorder repeats free (skipped), so no new cursor state is needed | agent | design | low |

### Design Freeze

- [x] Reorder-only (no skipping) — resolved by Mark's "saturated ones get occasional re-confirmation runs"
- [x] Source of truth = `docs/static/benchmarks/latest.json` ratings block

## Solution Design

### Overview

Add a **partition step** between benchmark-list construction and the round-robin cursor in
both passes of `os-rotation-filler.sh`:

```
BENCHES (tier/ls order)  →  [signal... (non-saturated ∪ unrated), saturated...]  →  cursor/chunk as today
```

Each sub-list keeps stable (sorted) order internally so the cursor walk stays deterministic.
When a cycle's chunk contains both signal and saturated picks (the boundary chunk), the
eval-suite invocation is split into two calls — signal picks at `--trials 3`, saturated picks
at `--trials $OS_FILLER_SAT_TRIALS` (default 1) — so trial count follows the benchmark, not
the chunk. Everything else (cursor files, wrap/lap detection, `--skip-existing`,
`--bank-by-version`, blackout, lock, publish steps) is untouched.

### Architecture

**Components:**

1. **`tools/eval-signal-set.sh` (new helper, ~40 LOC)** — prints the saturated benchmark ids
   for a mode/lang, one per line:
   ```bash
   tools/eval-signal-set.sh --json docs/static/benchmarks/latest.json --mode agent [--lang ailang]
   ```
   - jq query: `.ratings.agent.byLang.ailang.benchmarks[]? | select(.saturated == true) | .id`,
     falling back to top-level `.ratings.agent.benchmarks[]` when the byLang key is absent.
   - Exits 0 with empty output (and a stderr note) when the file/keys are missing or jq is
     unavailable → callers degrade to "everything is signal" = today's behavior. Fail-open by
     design (a stale order is acceptable; a wedged rig is not).
   - Standalone (not inlined in the filler) so the nightly scripts or a future cloud-side
     scheduler can read the same set the same way — this is "wherever the saturated set is
     read from" made canonical.

2. **Filler partition function (in `os-rotation-filler.sh`, ~30 LOC, bash-3.2-safe)** —
   `prioritize_benches` takes the constructed array + the saturated id list and rewrites the
   array as signal-first, saturated-tail. Applied to `BENCHES_FULL` (with `--lang ailang`) and
   to `BENCHES` (top-level agent set) when `OS_FILLER_ELO_PRIORITY=1` (default **1**; set 0 to
   restore pure ls-order).

3. **Split-chunk trial policy (~25 LOC)** — after picking the chunk, split picks into
   signal/saturated groups against the same saturated set; run one `ailang eval-suite` per
   non-empty group. The existing `*reimplement*` special-case (trials=1, 5400s timeout) is
   evaluated per group and wins over the saturated policy (both say trials=1 for reimplement).

**What changes for lap/coverage semantics: nothing.** The lap is complete when the cursor
wraps the full (reordered) list — same total work per version, front-loaded by signal. The
`MIN_COV`-based coverage check counts banked files and is order-independent.

### Data/Refresh Considerations

- **Staleness across versions:** at the start of a new version's lap, `latest.json` still
  carries the previous version's fit. That is the correct prior — saturation is a property of
  benchmark-vs-model-population, not of a release — and Key Fact 3 refreshes it as soon as the
  new version has a cloud baseline + banked local results.
- **Worktree/launchd caveat:** launchd executes the filler from the MAIN checkout's working
  tree. This change lands via normal commit → push to `dev` → rig pulls; do not hand-edit the
  main checkout from a worktree session.

### Implementation Plan

**Phase 1: Helper + partition (~2h)**
- [ ] `tools/eval-signal-set.sh` with `--json/--mode/--lang`, byLang→top-level fallback, fail-open
- [ ] `prioritize_benches` in the filler; wire into `BENCHES_FULL` and `BENCHES` behind `OS_FILLER_ELO_PRIORITY` (default 1)
- [ ] Cycle log line: `elo-priority: N signal / M saturated (source: latest.json agent[/ailang])`

**Phase 2: Split-chunk trials (~1.5h)**
- [ ] Group chunk picks by saturation; per-group `eval-suite` calls with `--trials 3` vs `--trials $OS_FILLER_SAT_TRIALS` (default 1)
- [ ] Preserve reimplement special-case per group

**Phase 3: Verification + docs (~1h)**
- [ ] Dry-run harness: `bash -n` + a shellcheck pass + an offline simulation of the partition/pick/split logic (no eval-suite execution)
- [ ] Verify fail-open: point at a bogus JSON path → order identical to today, warning logged
- [ ] CHANGELOG entry + this doc moved to implemented on release

### Files to Modify/Create

**New files:**
- `tools/eval-signal-set.sh` — saturated-set reader (~40 LOC)

**Modified files:**
- `tools/launchd/os-rotation-filler.sh` — partition + split-chunk trials (~+60 LOC)
- `CHANGELOG.md` — entry under current version

## Examples

### Example 1: Fresh version lap (v0.31.0, 56 full-tier benchmarks, 26 signal / 30 saturated)

**Before:** cycle 1 picks `adt_option, ai_agent_context, api_url_builder` (ls order — 2 of 3
already Trivial); the signal set finishes banking only when the whole alphabet has passed,
days in.

**After:** cycles 1–9 pick only the 26 signal benchmarks (`docx_reimplement`,
`float_eq`, … in sorted order) at 3 trials; cycles 10–19 sweep the 30 saturated benchmarks at
1 trial each; lap marker set on wrap exactly as today.

### Example 2: Regression un-saturates a benchmark

`records_person` is saturated (ELO 1240). During the v0.31.0 re-confirmation sweep its single
trial FAILS on qwen3.6. Step 8's next `eval-report --merge` re-fit lifts it out of the Trivial
band → `saturated: false` in `latest.json` → in the v0.31.1 lap it is back in the signal set at
full 3 trials. No manual intervention.

### Example 3: Ratings unavailable

`latest.json` is mid-rewrite/corrupt when the cycle fires. `eval-signal-set.sh` prints nothing,
logs `elo-priority: ratings unreadable — falling back to ls-order`, and the cycle behaves
byte-for-byte like today's filler.

## Success Criteria

- [ ] Offline simulation shows: signal benchmarks occupy list head, saturated tail, both sorted; boundary chunk splits into two eval-suite calls with correct `--trials`
- [ ] `OS_FILLER_ELO_PRIORITY=0` reproduces current ls-order behavior exactly
- [ ] Missing/corrupt ratings JSON ⇒ current behavior + logged warning (fail-open proven)
- [ ] One live rig cycle on the main checkout logs the priority line and banks a signal-first chunk
- [ ] shellcheck clean on both scripts; CHANGELOG updated

## Testing Strategy

**Unit-ish (offline, no GPU):**
- Fixture `latest.json` snippets (byLang present / absent / missing ratings / corrupt) →
  assert `eval-signal-set.sh` output and exit codes.
- Source the filler's partition + pick logic with a stubbed bench list and cursor →
  assert order, chunk splits, wrap detection at the partition boundary.

**Integration (rig, post-merge):**
- Watch `/tmp/ailang-os-filler.log` for the `elo-priority:` line on the next cycles; confirm
  picks come from the known signal set and saturated runs use `--trials 1`.

**Manual:**
- `OS_FILLER_ELO_PRIORITY=0 bash tools/launchd/os-rotation-filler.sh` smoke (behavior parity).

## Deferred Decisions

- Exact placement of the helper (`tools/` vs `tools/launchd/`) — agent may choose; `tools/` preferred for reuse.
- Whether the cross-language pass should use per-language saturated sets (python/js/go byLang
  keys, once they exist in ratings) instead of the top-level agent set — agent may start with
  top-level; refine when byLang carries those languages.
- Sorting *within* the signal head by descending ELO (hardest-first) vs plain sort — agent may
  choose; plain sort is sufficient for the stated goal and keeps diffs minimal.

## Non-Goals

- **No skipping of saturated benchmarks** — per-version full coverage and the un-saturation
  path require they still run.
- **No new ELO fitting path** — the filler consumes the published fit; it does not run
  `tools/eval-elo` or fit locally (Key Fact 2 shows the published fit already includes local trials).
- **No change to nightly-eval / nightly-lang-eval scheduling** — they have their own fixed benchmark sets.
- **No core (`internal/`) changes** — PROGRAM.md extension lane.
- **No cursor-state migration** — existing cursor files remain valid (an index is an index).

## Timeline

Single sitting: ~4.5 hours total (Phases 1–3). No multi-week plan needed for a ~100-LOC
shell change.

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Mid-lap ratings refresh reorders the list under a live cursor → some benchmarks repeat/skip within the lap | Low | Repeats are free (`--skip-existing`); skipped ones are caught by the `MIN_COV` coverage check, which blocks the lap marker until every benchmark is banked |
| jq missing on some future host | Low | Fail-open to ls-order + warning; helper exits 0 |
| `latest.json` schema drift (ratings block renamed) | Med | Helper's empty-output fail-open + warning makes drift visible in the cycle log instead of wedging; success criterion covers the corrupt-JSON path |
| Saturated 1-trial re-confirmation flakes (false regression) un-saturates a healthy benchmark | Low | Cost is bounded: it returns to the signal set for one lap at 3 trials, then re-saturates on the next fit |

## Related Documents

**Implemented (inform design):**
- [design_docs/implemented/v0_15_0/m-ollama-local-eval.md](../implemented/v0_15_0/m-ollama-local-eval.md) — the rig this schedules
- M-EVAL-OS-FRONTIER-COVERAGE / M-EVAL-VERSION-BANKING — the AILANG-first pass + per-version banking this builds on (see filler script comments)
- M-EVAL-DASHBOARD-REDESIGN — introduced the ratings block consumed here

**Planned (checked for overlap — distinct):**
- [design_docs/planned/v0_29_0/m-eval-openrouter-baseline-rotation.md](v0_29_0/m-eval-openrouter-baseline-rotation.md) (0.39) — cloud baseline rotation cadence; does not touch local pick ordering
- [design_docs/planned/m-eval-validity-discipline.md](m-eval-validity-discipline.md) — coverage-gating for ELO *display*; complementary (this doc changes *acquisition* order)

## References

- `tools/launchd/os-rotation-filler.sh` — current cursor/chunk logic
- `internal/eval_analysis/ratings_export.go` — `saturated` derivation
- `internal/eval_harness/ratings.go` — `FitFromTrials`, `Band`
- `tools/publish-unified-dashboard.sh` + `internal/eval_analysis/export_json.go:499` — cloud+local merged fit
- Mark directive, 2026-07-20: "run across the benchmarks that count and have signal"

## Future Work

- Descending-ELO ordering within the signal head (hardest-first) once byLang ratings stabilize.
- Cloud-side analogue: use the same signal set to order OpenRouter baseline top-ups.

---

**Document created**: 2026-07-20
**Last updated**: 2026-07-20
