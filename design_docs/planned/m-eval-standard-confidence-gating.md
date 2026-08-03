# M-EVAL-STANDARD-CONFIDENCE-GATING: ELO Confidence Gating for the Cloud Standard Release Baseline

**Status**: Planned
**Target**: v0.31.x (tooling-only; no compiler/runtime change; first gated release TBD by Mark)
**Priority**: P1 — every release currently spends ~$100-135 re-confirming benchmarks the roster already passes
**Estimated**: ~4 days
**Dependencies**: M-EVAL-RATING-EFFICIENCY (implemented v0.26.0 — provides the ELO fit + `--benchmarks-by-confidence` mechanism this reuses as-is), M-EVAL-ELO-PRIORITY-ROTATION (implemented v0.30.0 — proves the pattern on the local rig; this doc deliberately makes a different call on skip-vs-reorder, see below)

## PROGRAM.md Routing

**Lane: extension** (eval harness tooling). Nothing in `internal/{parser,types,eval,core,elaborate,effects,builtins,lexer,ast,pipeline,runtime,link,iface}` is touched. The change lives in `.claude/skills/post-release/scripts/run_eval_baseline.sh`, a small addition to `cmd/ailang/eval_suite.go`, and a new cost-projection helper in `internal/eval_harness/`. The ELO fit itself (`internal/eval_harness/ratings.go`, `cmd/ailang/eval_confidence.go`) is reused unmodified — this doc *points* it at a path it was never wired to, it does not change how it works.

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Today's full-tier run is a fixed constant, trivially reproducible. A gated run depends on the ratings DB's state at invocation time — *less* predictable ahead of time, but the resolved benchmark list is recorded in `baseline.json` (same pattern the agent step already uses), so the historical record stays fully auditable. Net neutral, not a regression. |
| A2: Replayability | 0 | No trace impact. |
| A3: Effect Legibility | 0 | No AILANG effect-system impact. |
| A4: Explicit Authority | 0 | Reads a repo-local/user-local SQLite DB our own tooling already produces; no new access granted. |
| A5: Bounded Verification | 0 | Not applicable — no type-checking surface. |
| A6: Safe Concurrency | 0 | No change to eval-suite's worker pool concurrency model. |
| A7: Machines First | +1 | Release $ is redirected to benchmarks that still carry discriminative signal; the hand-guessed cost prose in SKILL.md is replaced by a computed estimate. |
| A8: Minimal Syntax | +1 | No new AILANG syntax. The core selection mechanism (`--benchmarks-by-confidence`, `--max-benchmarks 0` = "all non-saturated") already exists — this is a wiring change, not new surface. |
| A9: Cost Visibility | +1 | The direct point of the doc: a real `--dry-run` cost estimate and an aggregate `--budget-usd` stop-gate replace a cost table the skill file itself already flags as stale. |
| A10: Composability | +1 | Composes with existing `--tier`, `--skip-existing`, `--bank-by-version`, and the already-shipped `--benchmarks-by-confidence`/`--max-benchmarks` flags without changing their semantics. |
| A11: Structured Failure | +1 | Missing/unseeded ratings DB fails open to today's full-tier behavior with a loud, actionable message (mirrors `eval-signal-set.sh`'s fail-open design in M-EVAL-ELO-PRIORITY-ROTATION) — never a silent truncated baseline. |
| A12: System Boundary | 0 | No boundary change. |

**Net Score: +5** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced — resolved list is always recorded
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Not optimizing for human convenience over machine analysis

## Key Facts (verified 2026-08-03)

These are the load-bearing premises, each checked against the working tree this session — cited directly rather than re-derived:

1. **The selection mechanism already exists and needs no new Go code to do the core job.** `cmd/ailang/eval_confidence.go::selectBenchmarksByConfidence` reads `benchmark_ratings`/`model_ratings` from a SQLite DB (`~/.ailang/state/observatory.db` via `--benchmarks-by-confidence auto`), drops any benchmark in the "Trivial" band (`< 1300` ELO, `internal/eval_harness/ratings.go:85-98`), and ranks survivors by proximity to the field's median model rating. Critically: **`--max-benchmarks 0` (the default) applies no cap** — it returns every non-saturated benchmark, which is exactly "run everything that isn't known-passing," not an arbitrary top-N truncation. (cmd/ailang/eval_suite.go:101-102, 271-289)
2. **It is currently standard-mode-blind.** `SELECT mode, COUNT(*) FROM benchmark_ratings GROUP BY mode` on the live `observatory.db` returns `agent|71` only — zero standard rows. `SELECT mode, COUNT(*) FROM model_ratings GROUP BY mode` returns `agent|3` — three local-rig models, none of the 18 `extended_suite` cloud models.
3. **It is currently cloud-blind.** The only invoker of `--benchmarks-by-confidence` anywhere in the repo is `tools/launchd/nightly-eval.sh`, which runs it against exactly one local Ollama model. `.claude/skills/post-release/scripts/run_eval_baseline.sh` (the script the `post-release` skill runs for every release) always resolves `--tier core,stretch,frontier` — the full 56-benchmark set (`smoke:23 core:19 stretch:21 frontier:16 vision:9` per `benchmarks/*.yml`, confirmed live) — against every model in `extended_suite`.
4. **Frontier tier's curation contract requires full coverage every release, already, independent of this doc.** Per `.claude/skills/post-release/SKILL.md`: "a frontier benchmark's defining property is that at least one frontier model FAILS it in standard mode; if every frontier model passes it, it demotes back to stretch... Release baselines are the ONLY routine source of frontier-failure data." Confidence-gating frontier would silently break that contract. This is a design constraint, not an open decision.
5. **Real cost, from banked `cost_usd` fields on the last two actual baselines (not the skill's stale hand-maintained estimate, which itself is flagged stale in the file):**
   - v0.30.0: standard `$98.33` (1810 files), agent `$15.07` (280 files)
   - v0.29.2: standard `$102.89` (1853 files), agent `$32.33` (386 files)
   - Per-model concentration, v0.30.0 standard: top 5 of 18 models = **71%** of spend — `claude-fable-5` $31.34 (32% alone), `claude-opus-4-8` $12.22, `claude-sonnet-5` $10.27, `gpt5-6-sol` $9.25, `claude-sonnet-4-6` $6.70. `or-deepseek-v4-flash` was the *cheapest* model in the whole suite at $0.25/109 runs.
6. **DeepSeek is out of scope for this doc.** Live pricing check (2026-08-03): official DeepSeek V4 Flash API is still $0.14/$0.28 per 1M, matching `models.yml` exactly. It is already in both `extended_suite` (`or-deepseek-v4-flash`/`-pro`, standard) and `agent_suite` (`opencode-or-deepseek-v4-pro` — OS agent champion, 97%). It is not the cost lever this doc addresses; no model-roster change is proposed here.
7. **No aggregate/pre-flight `$` budget exists anywhere in `eval-suite`.** Only a per-(benchmark,model) abort ceiling (`ModelConfig.ResolvedMaxCostUSD`, `internal/eval_harness/models.go`). `--dry-run` (`cmd/ailang/eval_suite.go:381-404`) prints planned run counts, models, benchmarks, and harness routing — no cost figure anywhere in that block, confirmed by reading it directly.
8. **M-EVAL-ELO-PRIORITY-ROTATION (v0.30.0, implemented) made a deliberate "reorder-only, not skip" call for the local rig** ("Non-Goal: No skipping of saturated benchmarks... per-version full coverage and the un-saturation path require they still run"). That decision was scoped to a *single local model iterating continuously* on one GPU, where full coverage is cheap (it's free/local) and the goal was ordering, not spend reduction. This doc is explicitly the opposite case — a *paid, multi-model, periodic* cloud run — and makes a different, equally deliberate call: skip (not just reorder) Trivial-band benchmarks for core/stretch, because the cost this doc is solving for doesn't exist on the local rig.

## Problem Statement

Every AILANG release runs the full `core,stretch,frontier` tier (56 benchmarks) against all 18 `extended_suite` models for standard eval, regardless of whether a given benchmark has been saturated (every model passing it) for the last several releases. This costs ~$100-135 per release in re-confirmation spend that produces near-zero new information for the majority of that roster, while `SKILL.md`'s own cost table admits it doesn't even know the real number ("these figures are STALE and UNDERSTATE the real spend").

The exact machinery to fix this — ELO-style benchmark difficulty ratings, a "drop saturated, rank the rest by information value" selector, and a fail-open design — was already built and proven in production (M-EVAL-RATING-EFFICIENCY, v0.26.0; M-EVAL-ELO-PRIORITY-ROTATION, v0.30.0). It has simply never been pointed at the expensive path: the ratings DB has zero standard-mode data, and the release baseline script never calls the selector.

**Current State:**
- `run_eval_baseline.sh --full` = fixed 56-benchmark × 18-model standard sweep, every release, no skip logic
- `observatory.db`: `agent|71` benchmark ratings / `agent|3` model ratings; zero `standard` rows
- `--dry-run` shows run counts, never a `$` estimate; no aggregate budget cap exists at all

**Impact:**
- ~$100-135/release spent partly re-confirming benchmarks that have not failed in months, concentrated in the 4-5 most expensive flagship models
- The only tool that could fix this is idle for this use case — not a "build something new" problem, a "wire it up" problem
- No one can answer "how much will this release baseline cost" before running it; the number in the skill file is admittedly wrong

## Goals

**Primary Goal:** The standard release baseline skips re-confirming Trivial-band (saturated) `core`/`stretch` benchmarks for models with an established rating, while `frontier` stays fully covered every release and new/changed models always get a full pass — with a real, computed cost estimate and an enforceable budget ceiling.

**Success Metrics:**
- A gated `core,stretch` standard run on a stable release (no roster change) resolves to a benchmark set that is a strict subset of today's full set, and `frontier` resolves to the identical full set as today
- `--dry-run` prints a `$` total that lands within a documented tolerance of the run's actual banked cost
- A model with zero standard-mode rating coverage always gets the full `core,stretch,frontier` set, regardless of gating state
- Missing/unseeded/corrupt ratings DB reproduces today's exact full-tier behavior, with a loud log line — never a silently truncated baseline
- `--budget-usd` stops new work before exceeding the cap, retains banked results, and marks the baseline as partial rather than silently presenting it as complete

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Frontier tier is never confidence-gated — always full, every release | Its entire curation definition depends on routine full-coverage failure data (Key Fact 4); gating it would silently break tier demotion logic | human (pre-decided by the existing curation contract; listed here for visibility, not re-litigation) | design | low |
| Confidence-gating starts **opt-in** (new `--confidence-gate` flag on `run_eval_baseline.sh`, off by default) vs on-by-default immediately | First real gated release is the only way to validate the cost-estimator's accuracy and confirm nothing regresses silently before this governs every future release's spend | human | design | med |
| Periodic forced full-audit cadence even once gating is trusted (e.g. every release regardless / every Nth release / quarterly / on minor-version bumps only) | Confidence-gating is benchmark-centric, not model-centric — a stable model *could* regress on a benchmark it used to pass without a new model triggering the full-pass rule; a cadence bounds how long that blind spot can persist | human | design | med |
| `--budget-usd` breach behavior: hard-stop new work / warn-and-continue to completion | A mid-run stop protects spend but risks banking a partial baseline that downstream dashboard/report tooling could misread as complete if not explicitly labelled | human | design | med |
| Default `--budget-usd` ceiling for release runs | Needs a real number grounded in Key Fact 5's actual spend data, informed by how aggressive the gating default above is | human | design | low |
| New/changed model → forced full pass, detected by diffing `extended_suite` model list against `model_ratings WHERE mode='standard'` coverage | Mechanical, matches the existing "unrated counts as signal" rule from M-EVAL-ELO-PRIORITY-ROTATION | agent | compile | low |
| Where the shell script scopes confidence-gating to `core∪stretch` only (bash-side intersection with `resolve_benchmarks_in_tiers`, no Go change) vs a new Go-side tier filter on the selector | Bash intersection reuses code already in `run_eval_baseline.sh`; a Go-side filter would be a second, redundant way to do the same thing | agent | compile | low |

### Design Freeze

Resolved 2026-08-03 (Mark: "continue — sprint plan and execute"; taking the proposed defaults below since they're all conservative/reversible — a flag default and a dollar constant, not architecture):

- [x] Confidence-gating default: **opt-in initially, flipped to ON 2026-08-03** — Phase 4's live comparison (~29% projected reduction, see Verification section below) was reviewed with Mark same-day and he asked for default-on directly (`ailang eval-suite --dry-run` cost line shipped, live comparison run, Mark: "I want this cost check on by default"), so the originally-planned separate review gate was satisfied by that exchange rather than a follow-up session. `--full` runs now confidence-gate by default; `--no-confidence-gate` forces the old full-tier behavior (commit `7dc207a58`).
- [x] Full-audit cadence once gating is trusted: **every release still forces a full run once per quarter, or immediately on any `extended_suite` roster change** (a roster change already forces the affected model's own full pass per the new-model rule; the quarterly clause additionally re-audits the *whole* population even when the roster is stable, bounding the "stable model quietly regresses on a Trivial benchmark" blind spot from Risks & Mitigations).
- [x] `--budget-usd` breach behavior: **graceful stop** — finish in-flight trials, stop scheduling new ones, loud warning, `budget_stopped: true` in `baseline.json` (this was already the Solution Design's Component 4 behavior; formalizing it here rather than leaving it only implicit).
- [x] Default `--budget-usd` ceiling: **$150** (~15% headroom over the highest real combined baseline seen, v0.29.2's $135) for `--full` release runs; unset (no cap) otherwise, since the flag is new and shouldn't silently constrain non-release/dev usage.

## Solution Design

### Overview

Reuse the existing `--benchmarks-by-confidence`/`--max-benchmarks 0` selector unmodified for `core`+`stretch`; keep `frontier` exactly as it runs today; close the loop by seeding `observatory.db` with standard-mode ratings from each release's own banked results (same self-refreshing pattern the local rig already uses for agent mode); add the two genuinely missing pieces — a cost estimator and an aggregate budget cap — to `eval-suite` itself.

### Architecture

**Components:**

1. **`run_eval_baseline.sh` gating step (Step 1 rewrite, shell-only, no Go change for selection itself).** Before running standard eval:
   - Check the DB has standard-mode ratings (`sqlite3 $DB "SELECT COUNT(*) FROM benchmark_ratings WHERE mode='standard'"`, or attempt the confidence call and catch its existing error). Zero/missing/corrupt → **fail open**: run `--tier core,stretch,frontier` exactly as today, log `confidence-gate: standard ratings unavailable — falling back to full tier` (mirrors `eval-signal-set.sh`'s fail-open design).
   - Otherwise, split `extended_suite`'s model list against `model_ratings WHERE mode='standard'` coverage: models with no row = **new**, get the full `core,stretch,frontier` set; models with coverage = **rated**, get confidence-gated `core,stretch` (via `ailang eval-suite --benchmarks-by-confidence auto --max-benchmarks 0`, then bash-intersected with `resolve_benchmarks_in_tiers "core,stretch"` to exclude any frontier benchmark the selector's global ranking happens to surface) plus a **separate, always-full** `--tier frontier` call.
   - This is the same "split-chunk" shape M-EVAL-ELO-PRIORITY-ROTATION already used for signal/saturated groups on the local rig — same pattern, different axis (model coverage instead of trial count).
2. **Self-seeding loop.** Immediately after Step 1 banks results, run `go run ./tools/eval-elo --mode standard --persist ~/.ailang/state/observatory.db "$RESULTS_DIR"` — refreshes the DB with this release's data (including the frontier full-run, which is harmless to include and simply won't rank Trivial). Next release's gating reflects the most recent state, closing the loop exactly like the rig's Key Fact 3.
3. **Cost estimator (`internal/eval_harness/`, new, ~100 LOC).** For a planned (model, benchmark, lang) set, look up each benchmark's historical mean input/output tokens (from the most recent baseline's banked results; benchmarks with no history get a conservative flat default, clearly labelled as such — not silently omitted), multiply by each model's `pricing.input_per_1k`/`output_per_1k`, sum. Wired into `--dry-run` as an additional printed line.
4. **Aggregate budget cap (`cmd/ailang/eval_suite.go`, new flag `--budget-usd`, ~60 LOC).** A shared running total of banked `cost_usd` checked after each completed trial; once it crosses the cap, stop scheduling new work (in-flight trials finish), print a loud warning, and write `budget_stopped: true` into `baseline.json` (same augmentation pattern `run_eval_baseline.sh` already uses for the agent-stage metadata) so downstream dashboard/report code can detect and label a partial baseline instead of silently treating it as complete.

### Implementation Plan

**Phase 1: Gating wiring in `run_eval_baseline.sh`** (~1 day)
- [ ] Fail-open check for unseeded/missing/corrupt standard ratings
- [ ] New-model vs rated-model split against `model_ratings WHERE mode='standard'`
- [ ] Confidence-gated `core,stretch` call + bash intersection with tier files + always-full `frontier` call
- [ ] Self-seeding `eval-elo --mode standard --persist` step after Step 1 banks
- [ ] `--confidence-gate` opt-in flag (off by default pending Design Freeze)
- [ ] Bootstrap: one-time seed of the DB from the most recent existing baseline (v0.30.0 or later) before the first gated run

**Phase 2: Cost estimator** (~1 day)
- [ ] Historical token-average lookup from the most recent baseline's banked results
- [ ] `$` projection combining lookup × `models.yml` pricing, with an explicit "no history — rough estimate" label for uncovered benchmarks
- [ ] Wire into `--dry-run` output

**Phase 3: Aggregate `--budget-usd` cap** (~1 day)
- [ ] Shared cost counter across the worker pool, checked post-trial
- [ ] Graceful stop (finish in-flight, stop scheduling new) + loud warning + `budget_stopped: true` in `baseline.json`
- [ ] Unit tests: stub cost stream crossing the cap mid-run

**Phase 4: Docs + first live comparison** (~1 day)
- [ ] Replace `SKILL.md`'s stale hand-maintained cost table with the computed-estimate workflow; document the gating cadence decided in Design Freeze
- [ ] CHANGELOG entry
- [ ] First real `--confidence-gate` dry-run on the next release, compared side-by-side against the equivalent full-tier dry-run cost estimate, reported to Mark before the flag is defaulted on

### Files to Modify/Create

**New files:**
- `internal/eval_harness/cost_estimate.go` — historical token lookup + `$` projection (~100 LOC)
- `cmd/ailang/eval_suite_budget_test.go` — budget-cap unit tests (~80 LOC)

**Modified files:**
- `.claude/skills/post-release/scripts/run_eval_baseline.sh` — gating split, fail-open check, self-seeding step, `--confidence-gate` flag (~+90 LOC)
- `cmd/ailang/eval_suite.go` — `--budget-usd` flag, cost-estimate line in `--dry-run` output (~+70 LOC)
- `.claude/skills/post-release/SKILL.md` — replace stale cost table, document new cadence (~60 lines rewritten)
- `CHANGELOG.md`

## Examples

### Example 1: Routine patch release, stable roster (e.g. v0.31.1)

**Before:** Step 1 runs `core,stretch,frontier` (56 benchmarks) × 18 models = ~$100, identical spend whether or not anything changed since v0.31.0.

**After:** DB has standard ratings from v0.31.0's self-seed. All 18 models are already rated. `frontier` (16 benchmarks) runs full as always. `core,stretch` (40 benchmarks) confidence-gates to, say, 22 non-Trivial benchmarks. Total ≈ 16+22=38 of 56 benchmarks × 18 models — a real reduction, magnitude to be confirmed by Phase 4's first live comparison, not asserted here.

### Example 2: Release with a flagship model swap (e.g. `claude-opus-4-8` → `claude-opus-5`)

**Before:** Same fixed 56×18 run — the new model gets exactly the same coverage as every established one, which is correct today because there's no gating to differ from.

**After:** The new-model split detects `claude-opus-5` has zero `model_ratings` rows for `mode='standard'`. It gets the full `core,stretch,frontier` set exactly like today. Every other already-rated model gets the gated `core,stretch` + full `frontier`. `claude-opus-5` earns its own rating from this run and joins the gated population starting next release.

### Example 3: Ratings DB missing or corrupt

`observatory.db` is deleted, mid-migration, or the `standard` rows are somehow wiped. `run_eval_baseline.sh`'s fail-open check finds zero standard rows, logs `confidence-gate: standard ratings unavailable — falling back to full tier`, and runs `--tier core,stretch,frontier` exactly as it does today. No manual intervention, no crash, no truncated baseline.

## Success Criteria

- [ ] A gated `core,stretch` run's resolved benchmark list is a strict, sorted subset of the full tier's list (verified against `resolve_benchmarks_in_tiers`)
- [ ] `frontier`'s resolved list is byte-identical between gated and full-tier runs
- [ ] `--dry-run` prints a `$` estimate; on the first live comparison (Phase 4) it lands within a documented tolerance of the actual banked cost (tolerance itself is a Phase 4 finding, not pre-committed here — first-run estimates are inherently rough)
- [ ] A model with zero standard-mode DB coverage always resolves to the full tier, confirmed with a synthetic "new model" fixture
- [ ] Deleting/corrupting the DB reproduces today's exact full-tier `run_eval_baseline.sh` behavior byte-for-byte, plus the fail-open log line
- [ ] `--budget-usd` stops scheduling new trials before exceeding the cap by more than one in-flight batch; `baseline.json` carries `budget_stopped: true` when it fires
- [ ] `SKILL.md`'s cost table is replaced with the computed workflow; the "STALE and UNDERSTATE" warning is removed because the number is now real
- [ ] One full gated release cycle reviewed with Mark, showing actual $ saved vs the equivalent full-tier estimate, before `--confidence-gate` is proposed as default-on

## Testing Strategy

**Unit tests:**
- Cost estimator against a fixture of known historical token counts + `models.yml` pricing → exact projected `$` matches a hand calculation
- Budget-cap stop logic against a stubbed cost stream crossing the cap mid-run — verifies graceful stop, not silent truncation
- New-model detection against a fixture `model_ratings` table with a partial roster

**Integration tests:**
- Run the gating logic against the *actual current* (unseeded) `observatory.db` — this is a real regression fixture already sitting in the repo's environment: it must trigger fail-open, not error out
- After a synthetic seed (`eval-elo --mode standard --persist` against an existing baseline dir), verify the gated selection excludes exactly the benchmarks that fit the Trivial band in that baseline's data

**Manual testing:**
- First live `--confidence-gate` release: diff the gated subset's pass/fail against a full run on a small sample of the "dropped" benchmarks to confirm nothing silently regressed
- Verify `ailang eval-elo <new baseline dir>` still renders sane leaderboards after the self-seeding step

## Deferred Decisions

The following are intentionally left open for the implementer:

- Exact token-history source for the cost estimator (most recent single baseline vs a rolling average across the last N) — agent may choose; start with most-recent-baseline, the simplest correct answer, and note if it proves too noisy
- Whether frontier's full-run results also get persisted into the standard ratings DB (harmless either way, since frontier is always forced-full regardless of its own rating) — agent may choose; leaning toward yes, for a fuller historical record feeding future tier-curation analysis
- Placement/naming of the cost-estimate helper (new file vs extending `internal/eval_harness/models.go`) — agent may choose

## Non-Goals

**Not attempted in this feature:**
- **Confidence-gating the `frontier` tier** — its curation contract requires full coverage every release (Key Fact 4); out of scope by design, not an oversight
- **Confidence-gating the agent-mode release step (`agent_suite`)** — that step already solves its own cost problem by construction (weak/cheap models only); a follow-up could extend gating there but it's a separate, lower-priority story (see Future Work)
- **Any DeepSeek pricing or model-roster change** — confirmed already optimal (Key Fact 6); not a lever this doc pulls
- **A general-purpose real-time cost dashboard or alerting system** — `--budget-usd` is a simple stop-gate, not observability tooling
- **Changing M-EVAL-ELO-PRIORITY-ROTATION's reorder-only behavior on the local rig** — different, already-settled scope; this doc does not revisit that decision

## Timeline

**Day 1:** Phase 1 — gating wiring, fail-open, self-seed, new-model split. Validate fail-open against the real current (unseeded) DB.

**Day 2:** Phase 2 — cost estimator, wired into `--dry-run`.

**Day 3:** Phase 3 — `--budget-usd` aggregate cap + unit tests.

**Day 4:** Phase 4 — `SKILL.md` rewrite, CHANGELOG, first live gated dry-run compared against today's full-tier estimate, reported to Mark.

**Total: ~4 days, single sprint.**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Gating masks a genuine regression in an already-rated model on a benchmark that used to be Trivial for the old roster | Med | New/changed models always forced full (Key Fact rule); periodic full-audit cadence (Design Freeze item) re-confirms the whole population on a bounded schedule |
| Cost estimator is inaccurate for benchmarks with no history (new benchmarks, roster changes) | Low | Explicit "no history — rough estimate" label rather than silent omission (NO SILENT FALLBACKS) |
| `--budget-usd` fires mid-run and the post-release skill doesn't notice, publishing a partial baseline as if complete | Med | `budget_stopped: true` in `baseline.json`, same visible-metadata pattern already used for the agent-stage augmentation; Success Criteria requires this be checked |
| Bash-side tier intersection has a set-logic bug, silently dropping or duplicating benchmarks | Low | Success Criteria explicitly diffs the gated resolved list against `resolve_benchmarks_in_tiers` output |
| First gated release under-delivers savings because the population is less saturated in standard mode than agent mode (no data yet either way) | Low | Phase 4 is a measurement step before any default-on decision — the doc doesn't assume savings, it measures them |

## Related Documents

**Implemented (this doc's foundation — do not duplicate, reuse):**
- [design_docs/implemented/v0_26_0/m-eval-rating-efficiency.md](../implemented/v0_26_0/m-eval-rating-efficiency.md) — the ELO fit + `--benchmarks-by-confidence` selector this doc wires into a new path, unmodified
- [design_docs/implemented/v0_30_0/m-eval-elo-priority-rotation.md](../implemented/v0_30_0/m-eval-elo-priority-rotation.md) — proves the pattern on the local rig; explicitly reorder-only there (different scope, different cost structure — see Key Fact 8) — this doc makes the opposite (skip) call deliberately, not by oversight

**Planned (checked for overlap — distinct):**
- [design_docs/planned/m-eval-validity-discipline.md](m-eval-validity-discipline.md) (0.41 similarity) — governs cross-cohort *display/ranking* validity (coverage-gated leaderboards); this doc governs *acquisition* (what gets run at all). Complementary, not overlapping — W5 of that doc even notes rig-side acquisition ordering as a separate concern.
- [design_docs/planned/v1_1_0/m-oracle-adequacy.md](v1_1_0/m-oracle-adequacy.md) (0.45) — convergence oracles for correctness evidence; unrelated axis (evidence quality, not run-selection cost)
- [design_docs/planned/v1_1_0/m-eval-trust-signals.md](v1_1_0/m-eval-trust-signals.md) (0.42) — external credibility signals (HumanEval port, receipts); unrelated axis

## References

- [Design Axioms](/docs/references/axioms) — The 12 non-negotiable principles
- `cmd/ailang/eval_confidence.go` — `selectBenchmarksByConfidence`, the reused selector
- `cmd/ailang/eval_suite.go:101-102,271-289` — existing `--benchmarks-by-confidence`/`--max-benchmarks` flag wiring
- `internal/eval_harness/ratings.go:85-98` — `Band()`, the Trivial-band threshold
- `internal/eval_harness/models.go` — `ResolvedMaxCostUSD`, the existing per-benchmark budget mechanism this doc extends to run-level
- `.claude/skills/post-release/scripts/run_eval_baseline.sh` — the script this doc modifies
- `.claude/skills/post-release/SKILL.md` — the skill doc whose cost table this doc replaces
- `tools/launchd/nightly-eval.sh` — the only current caller of `--benchmarks-by-confidence`, the precedent this doc generalizes
- `eval_results/baselines/v0.30.0/`, `eval_results/baselines/v0.29.2/` — source of the real cost figures in Key Fact 5

## Verification — first live comparison (M4, 2026-08-03)

Implemented and verified in one sitting (M1-M4 all landed same-day; see commits
`2d2e45e05`, `49582b33c`, `ecbb0d7f2`). `run_eval_baseline.sh` has no `--dry-run` mode of
its own (only `ailang eval-suite` does), so the sprint plan's planned "run
`run_eval_baseline.sh --full --dry-run` twice" step was executed as its precise equivalent:
`ailang eval-suite --dry-run` invoked directly with the exact model/benchmark splits
`run_confidence_gated_standard()` computes (verified live against the real `extended_suite`
roster and an `observatory.db` bootstrapped from the v0.30.0 baseline, per Key Fact set above).

| Call | Models | Benchmarks | Estimated cost |
|---|---|---|---|
| Full (no gating) | 18 | 56 (core+stretch+frontier) | $49.57 |
| Gated: new models | 3 (no rating history) | 56 (full tier) | $8.27 |
| Gated: rated models, gated core+stretch | 15 | 21 of 40 (Trivial-band dropped) | $15.71 |
| Gated: rated models, frontier | 15 | 16 (always full) | $11.30 |
| **Gated total** | | | **$35.28** |

**Projected savings: $14.29, ~29%** of the confidence-gated portion of standard-eval cost
(does not include the agent step or `--lang-harness`, which this design doesn't touch).

**Caveat, stated plainly:** this is a pre-flight *projection*, not a live spend comparison —
`estimateRunCostUSDWithMeans` fell back to the flat default for roughly half the priced pairs
in both calls (504/1008 full, ~140/315+168+240 gated), because `observatory.db`'s token-mean
history comes from v0.30.0, which predates 3 of the current 18 `extended_suite` models and
some tier reshuffling since. The projected 29% is directionally real (fewer benchmarks run for
the rated-model majority, priced at the same rates either way) but the exact dollar figures
will sharpen automatically after the first `--confidence-gate` release run self-seeds
`observatory.db` with current data (Solution Design point 2). The Success Criteria's "actual $
saved" review — a real spend comparison across a live release cycle, not this projection — is
still the gate before defaulting `--confidence-gate` on, per the Design Freeze resolution.

## Future Work

- Extend confidence-gating (or a lighter cadence-based rotation) to the `agent_suite` release step, once this pattern is proven safe on standard mode
- Descending-ELO ordering within the gated set (hardest-first), mirroring M-EVAL-ELO-PRIORITY-ROTATION's own noted future work
- A cloud-side analogue of the local rig's per-cycle `elo-priority:` log line, surfaced on the dashboard, so gating decisions are visible per release without reading `run_eval_baseline.sh` output

---

**Document created**: 2026-08-03
**Last updated**: 2026-08-03 (Design Freeze resolved)
