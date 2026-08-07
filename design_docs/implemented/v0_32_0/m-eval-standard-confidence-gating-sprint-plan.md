# Sprint Plan: M-EVAL-STANDARD-CONFIDENCE-GATING

## Summary

Wire the already-implemented ELO confidence-select mechanism (`--benchmarks-by-confidence`,
M-EVAL-RATING-EFFICIENCY v0.26.0) into the cloud/standard release baseline, add the two pieces
that don't exist yet (a real `--dry-run` cost estimate, an aggregate `--budget-usd` cap), and
replace the post-release skill's admittedly-stale cost table with computed numbers.

**Duration:** 4 days
**Dependencies:** None — the ELO fit, `Band()`, and `selectBenchmarksByConfidence` are already
shipped and reused unmodified. Design Freeze fully resolved in the design doc (opt-in default,
quarterly-or-roster-change full-audit cadence, graceful-stop budget behavior, $150 default ceiling).
**Risk Level:** Low — every phase is additive (new flag, new file, new script logic) with an
explicit fail-open path; nothing in the existing full-tier behavior changes unless `--confidence-gate`
is passed.

## Current Status Analysis

### Completed Recently (context, not part of this sprint)
- ✅ M-EVAL-RATING-EFFICIENCY (v0.26.0): ELO fit + `--benchmarks-by-confidence`/`--max-benchmarks`
- ✅ M-EVAL-ELO-PRIORITY-ROTATION (v0.30.0): proved the ratings-driven pattern on the local rig
  (reorder-only there — this sprint makes the skip call for the cloud path instead)

### Velocity
- No directly comparable recent eval-harness sprint in the last 7 days to benchmark LOC/day
  against; repo velocity this week has been mission/skill-doc heavy, not Go-implementation heavy.
- Falling back to the design doc's own phase-by-phase estimate (~4 days, ~450 LOC across 4 files),
  which is in line with the two precedent M-EVAL sprints' scale (3-4 days and ~4.5 hours
  respectively, for comparable-sized harness changes).

### Remaining from Design Doc
- ⏳ Phase 1: gating wiring in `run_eval_baseline.sh` (~90 LOC shell)
- ⏳ Phase 2: cost estimator (~100 LOC + wiring)
- ⏳ Phase 3: aggregate budget cap (~60 LOC + ~80 LOC tests)
- ⏳ Phase 4: SKILL.md rewrite + CHANGELOG + first live comparison

## Proposed Milestones

### M1: Gating wiring in run_eval_baseline.sh (~90 LOC)

**Goal:** Standard-mode confidence gating is available behind a new opt-in flag, with a proven
fail-open path and a new-model safety rule, wired to the script's existing tier-resolution helpers.

**Estimated:** ~90 LOC shell (no Go changes — the selection mechanism itself, `--benchmarks-by-confidence
auto --max-benchmarks 0`, already exists and needs no modification)
**Duration:** 1 day
**Dependencies:** None

**Tasks:**
- Day 1: Add `--confidence-gate` flag to `run_eval_baseline.sh`'s arg parser (default off)
- Day 1: Add the DB-state check (`sqlite3 $DB "SELECT COUNT(*) FROM benchmark_ratings WHERE mode='standard'"`,
  fail open to today's `--tier core,stretch,frontier` behavior with a logged message if zero/missing/corrupt)
- Day 1: Add the new-model/rated-model split — diff `extended_suite`'s model list (from `models.yml`)
  against `model_ratings WHERE mode='standard'` coverage
- Day 1: Wire the confidence-gated call for rated models (`--benchmarks-by-confidence auto
  --max-benchmarks 0` on `core,stretch`, bash-intersected with `resolve_benchmarks_in_tiers
  "core,stretch"` to exclude any frontier ID the global ranking surfaces) + an always-full
  `--tier frontier` call for every model regardless of rating state
- Day 1: Add the self-seeding step after Step 1 banks (`go run ./tools/eval-elo --mode standard
  --persist $DB "$RESULTS_DIR"`)
- Day 1: One-time DB bootstrap: seed `observatory.db` from the most recent existing full baseline
  (v0.30.0) so M1's own manual verification has real data to gate against

**Acceptance Criteria:**
- [ ] `--confidence-gate` off (default): script behavior is byte-identical to today, confirmed by diffing dry-run output before/after this milestone on an unrelated benchmark set
- [ ] `--confidence-gate` on, DB seeded: gated `core,stretch` benchmark list is a strict, sorted subset of the full tier's list; `frontier`'s resolved list is byte-identical between gated and non-gated runs
- [ ] `--confidence-gate` on, DB has zero standard rows (today's actual live state, a real fixture): script falls open to full `core,stretch,frontier`, logs `confidence-gate: standard ratings unavailable — falling back to full tier`, does not error
- [ ] A synthetic "new model" (present in `extended_suite`, absent from `model_ratings`) always resolves to the full tier regardless of gating state
- [ ] shellcheck clean

**Risks:**
- Bash-side set intersection has an off-by-one/duplicate bug — Mitigation: acceptance criterion above explicitly diffs the resolved list against `resolve_benchmarks_in_tiers` output, not just eyeballed

### M2: Cost estimator + --dry-run wiring (~170 LOC)

**Goal:** `ailang eval-suite --dry-run` prints a real, computed `$` estimate instead of nothing,
so a release run's cost is known before it's spent — replacing the skill's hand-guessed prose.

**Estimated:** ~100 LOC implementation + ~70 LOC tests = ~170 LOC
**Duration:** 1 day
**Dependencies:** None (independent of M1 — could run in parallel, sequenced here for review bandwidth)

**Tasks:**
- Day 2: `internal/eval_harness/cost_estimate.go` — load historical mean input/output tokens per
  benchmark from the most recent baseline dir's banked results; conservative flat default (clearly
  labelled, not silently substituted) for benchmarks with no history
- Day 2: Projection function: `Σ over (model, benchmark, lang) of (mean_input_tokens × model.pricing.input_per_1k + mean_output_tokens × model.pricing.output_per_1k) / 1000`
- Day 2: Wire into `cmd/ailang/eval_suite.go`'s existing `--dry-run` block (`eval_suite.go:381-404`) as an additional printed line, alongside the existing planned-run-count output
- Day 2: Unit tests against a fixture of known historical token counts + `models.yml` pricing → exact match to a hand calculation

**Acceptance Criteria:**
- [ ] `--dry-run` output includes a `$` total line
- [ ] Benchmarks with no historical data are labelled "no history — rough estimate" in the breakdown, not silently folded into the total without a marker
- [ ] Unit test: fixture-derived estimate matches hand-computed expected value exactly
- [ ] `go build ./...` clean, existing `--dry-run` tests (`eval_suite_cohort_test.go`) still pass unmodified

**Risks:**
- No historical data exists yet for a brand-new benchmark — Mitigation: explicit fallback label, not a silent zero or omission (NO SILENT FALLBACKS)

### M3: Aggregate --budget-usd cap (~140 LOC)

**Goal:** A release run can be capped at a real dollar ceiling and will stop gracefully — not
mid-benchmark, not silently — when it's reached, leaving a clearly-labelled partial baseline.

**Estimated:** ~60 LOC implementation + ~80 LOC tests = ~140 LOC
**Duration:** 1 day
**Dependencies:** None functionally, but reviewed after M2 since both touch `eval_suite.go`'s
flag set and this milestone's design doc default ($150) references M2's cost data for calibration

**Tasks:**
- Day 3: `--budget-usd` flag on `eval-suite` (unset = no cap, preserving today's unconstrained behavior for dev/non-release use)
- Day 3: Shared running-cost counter across the worker pool, checked after each completed trial's `cost_usd` is banked
- Day 3: On breach: stop scheduling new work, let in-flight trials finish, print a loud warning
- Day 3: Write `budget_stopped: true` into `baseline.json` (same augmentation pattern `run_eval_baseline.sh` already uses for the agent-stage metadata block)
- Day 3: `cmd/ailang/eval_suite_budget_test.go` — stub a cost stream crossing the cap mid-run, assert graceful stop (not silent truncation, not a hard kill of in-flight work)
- Day 3: Wire `run_eval_baseline.sh --full` to pass `--budget-usd 150` by default (the Design-Freeze-resolved default), overridable

**Acceptance Criteria:**
- [ ] Unset `--budget-usd`: behavior byte-identical to today (no regression for non-release use)
- [ ] Set and breached: scheduling of new trials stops before exceeding the cap by more than one in-flight batch; already-banked results are retained, not discarded
- [ ] `baseline.json` carries `budget_stopped: true` only when the cap actually fired
- [ ] Unit test passes with a stubbed cost stream
- [ ] `run_eval_baseline.sh --full` passes `--budget-usd 150` by default; overridable via a new script flag

**Risks:**
- A race between the shared counter and concurrent trial completion lets spend overshoot slightly — Mitigation: this is accepted and documented (graceful stop, not a hard real-time kill); the "more than one in-flight batch" tolerance in the acceptance criterion reflects that explicitly rather than promising unrealistic precision

### M4: SKILL.md rewrite + CHANGELOG + live comparison (docs + verification, no new LOC budget)

**Goal:** The post-release skill documents the new workflow accurately (no more self-admitted-stale
numbers), and the first real gated run is measured and reported before anyone defaults it on.

**Estimated:** ~60 lines of `SKILL.md` rewritten, 1 CHANGELOG entry, 1 comparison report (no code)
**Duration:** 1 day
**Dependencies:** M1, M2, M3 (needs the real flags/output to document accurately)

**Tasks:**
- Day 4: Replace `SKILL.md`'s "Cost & time" table and its "STALE and UNDERSTATE" warning with
  the computed-estimate workflow (`--dry-run` now prints the real number) and the `--confidence-gate`
  opt-in flag's existence
- Day 4: Document the resolved cadence (quarterly-or-roster-change full audit) and the $150 default budget in SKILL.md
- Day 4: CHANGELOG.md entry under the active changelog file
- Day 4: Run `run_eval_baseline.sh --full --dry-run` twice — once with `--confidence-gate`, once
  without — and record the projected `$` delta as the first real evidence of savings, reported to
  Mark before `--confidence-gate` is proposed as default-on for a future sprint

**Acceptance Criteria:**
- [ ] `SKILL.md`'s cost table reflects the real, computed workflow — no hand-guessed numbers remain
- [ ] CHANGELOG entry present
- [ ] Side-by-side `--dry-run` comparison (gated vs full) recorded and reported
- [ ] Documentation updated
- [ ] Examples added (N/A — this is tooling, not a language feature; skip per design doc's Non-Goals)

**Risks:**
- Live `--dry-run` comparison depends on the DB actually being seeded (M1's bootstrap step) — Mitigation: M1 already includes the bootstrap task; M4 verifies it landed before relying on it

## Success Metrics
- All 4 milestones' acceptance criteria checked
- `go build ./... && go vet ./...` clean
- `make lint` clean on all new/modified Go and shell files (shellcheck for `.sh`)
- New unit tests passing (M2, M3) alongside the full existing `eval_suite`/`eval_confidence` test suite
- `SKILL.md` cost table replaced with a real, computed workflow
- Real evidence (not projection) of the gated-vs-full cost delta, reported to Mark

## Dependencies
- None external. Internal ordering: M1 and M2 can run independently; M3 is sequenced after M2 for
  calibration context (not a hard technical dependency); M4 depends on M1-M3 being in place to
  document accurately and produce the live comparison.

## Open Questions
- None blocking — Design Freeze in the design doc resolved all four open decisions before this
  plan was written. The only thing this sprint explicitly defers is the decision to flip
  `--confidence-gate` to default-on, which is Future Work per the design doc, pending M4's real
  evidence.

## Notes
- This sprint touches `.claude/skills/post-release/scripts/run_eval_baseline.sh` and
  `.claude/skills/post-release/SKILL.md`, both of which currently carry small unrelated uncommitted
  edits in the working tree (a stale-cost-comment touch-up). M1/M4 build on top of that existing
  diff rather than reverting it; it's superseded by M4's full rewrite of the same block.
- Working directly in the main checkout (not a worktree) — this is an additive, flag-gated change
  with no impact on default behavior, and the precedent M-EVAL sprints (rating-efficiency,
  priority-rotation) were done the same way at similar scale. Commits land at each milestone
  boundary, not as one end-of-sprint commit, to keep the shared-tree window small.
