# M-EVAL-RATING-EFFICIENCY: ELO-Style Benchmark Difficulty + Targeted Reruns + Tier Saturation

**Status**: Planned
**Target**: v0.24.0 (rating + selective rerun) → v0.25.0 (tier graduation logic)
**Priority**: P1 — moves the rig from "run everything every time" to "run what's worth running, score what you learn"
**Estimated**: 3-4 days
**Dependencies**: M-EVAL-METRICS-TAXONOMY (planned, provides schema for `cell_metrics` we'll annotate with ratings)

## Problem statement

The rig currently treats every benchmark equally: full 17-benchmark smoke tier × N trials per rotation, regardless of which benchmarks are saturated (every model passes) or which are discriminating (only strong models pass). Three concrete inefficiencies result:

1. **Vague difficulty labels.** We hand-wave "fizzbuzz is easy" vs "dense_operator_program is hard". These are guesses derived from a handful of runs. With ELO-style ratings updated by every PASS/FAIL across every model, **the rig itself becomes the authority on what "hard" means** — and the labels stay current as compiler improvements move the goalposts.

2. **All-or-nothing reruns.** Re-running the full smoke tier to confirm one benchmark is 2-3 hours wall clock on this rig. Most of those trials produce no new information (we already know `gcd_lcm` passes; running it again is data we don't need). Targeted reruns of just the *interesting* benchmarks (saturated → flipped, flaky → need confirmation) would cut rotation time 5-10x.

3. **No tier graduation signal.** We've been running `smoke` only. As smoke saturates (current weekend run: 14/17 benchmarks at 100% pass rate for gemma4:26b-ailang), the marginal information from another smoke rotation drops to near-zero. We need an automated "smoke is saturated → graduate to core" signal, plus the inverse for new models that bomb on smoke (don't waste compute on stretch).

The Friday-Sunday weekend run actually proved this: 14/17 smoke benchmarks now pass on a 26B local model. Each of those 14 has near-zero remaining information value for *this* model+config combo. The 3 hard ones (balanced_parens, binary_tree_sum, dense_operator_program — currently being re-run at N=3 in Iter 6) are where every additional trial actually moves our belief.

## Why this matters NOW

The technical eval-harness work is **done**. The weekend run:
- Established the rig as stable (no hangs, no thrashing, all error modes catalogued)
- Validated the strategic claim (compiler error quality unlocks small-model benchmarks)
- Produced 6 rotation datasets with 102 trials of evidence across 5 iteration variants

The rig has graduated from "tool we're building" to "tool we're using." The next bottleneck is **information-per-compute-hour**, not "does the rig work." The work in this design doc is what makes that bottleneck loose.

Parallelism is not the answer (M-series single-GPU is bandwidth-bound at 27B+; we confirmed empirically Friday that NUM_PARALLEL>1 causes thrashing without throughput gain). **The answer is being smarter about which benchmarks to run, not running more in parallel.**

## Goals

1. **ELO-style ratings** for every (benchmark, model) pair, updated after every trial. Ratings let us:
   - Rank models against models (leaderboard)
   - Rank benchmarks by difficulty (within a tier)
   - Detect saturation (benchmark rating drops as everyone passes)
   - Detect promotion candidates (benchmark rating spikes as a new model class arrives)

2. **Selective reruns** — `ailang eval-suite --benchmarks-by-confidence` mode that, instead of running everything, runs only benchmarks where additional trials would meaningfully update our posterior:
   - Currently flaky (pass rate between 20-80% on the model under test)
   - Recently flipped (last 2 rotations disagree on outcome)
   - Adjacent to current rating boundary (model's strength is within ±100 ELO of benchmark's difficulty)

3. **Tier saturation report** at each release. The rig auto-recommends: "smoke is saturated for gemma4:26b-ailang at 14/17 reliable, time to run a core-tier rotation."

4. **Multi-model leaderboard via ratings**, not just pass rate. Today we can only compare two models on the SAME benchmark set; with ratings, we can compare models that have evaluated on DIFFERENT subsets (each new model only runs the benchmarks at its rating ± window, but the ELO update lets us still rank it against models tested previously).

## ELO-style rating system

### The model

Each `(benchmark, model)` cell starts at rating 1500 (chess convention). After a trial:

```
expected_outcome = 1 / (1 + 10^((benchmark_rating - model_rating) / 400))
actual_outcome   = 1.0 if PASS else 0.0
K                = 32  // standard chess K-factor for early ratings; drops to 16-24 as confidence grows

model_rating     += K * (actual_outcome - expected_outcome)
benchmark_rating += K * (expected_outcome - actual_outcome)   // mirror update
```

Intuition:
- Strong model wins easy benchmark = small rating change (expected to win)
- Strong model loses easy benchmark = large rating drop for model, large rating rise for benchmark
- Weak model wins hard benchmark = large rating rise for model, large drop for benchmark
- Equal-rated pair flipping a coin = small symmetric updates

### Per-benchmark difficulty tiers (derived, not hand-labeled)

Once each benchmark has accumulated enough trials (N=30 across all models), the rating clusters into bands:

| Band | ELO range | Interpretation |
|---|---|---|
| **Trivial** | < 1300 | Every model passes; saturated; consider tier-promoting to drop from rotations |
| **Easy** | 1300-1500 | Most models pass; useful for smoke tier |
| **Moderate** | 1500-1700 | Discriminates dev models from cheap models |
| **Hard** | 1700-1900 | Discriminates frontier from sub-frontier |
| **Very hard** | > 1900 | Currently passing for <20% of tested models |

The bands are descriptive, not normative — they emerge from data. Current smoke tier examples (predicted from weekend run):

- gcd_lcm, recursion_fibonacci, numeric_modulo → likely Trivial (100% reliable across all configs)
- adt_option, fizzbuzz, record_update → likely Easy (mostly pass, occasional flip)
- balanced_parens, canonical_normalization → likely Moderate (flaky in the 30-70% band)
- dense_operator_program (pre-PAR016) → was Very hard (0% on a 26B model); post-PAR016 → likely Moderate

### Cross-model rating updates

This is the powerful piece: when **Model B** runs a benchmark Model A has already rated, Model A's rating doesn't change (already settled), but Model B's rating updates against the *current* benchmark rating. This means new models can be slotted into the leaderboard with as few as 5-10 benchmark trials, not the full 17-benchmark smoke suite.

Concrete protocol for adding a new model:
1. Start the new model at rating 1500 with high uncertainty (K=32)
2. Run the 5 benchmarks closest to rating 1500 (mid-difficulty for the suite)
3. After 5 trials, rating settles into a band; K drops to 24
4. Run 3 more benchmarks at the inferred rating band ± 200 ELO
5. Total: 8 trials, ~30-45 min wall clock, gives confidence interval ± 80 ELO

Vs current approach: 17 benchmarks × 3 trials = 51 trials, 3-4 hours wall, mostly redundant.

## Selective rerun: `--benchmarks-by-confidence`

A new mode for `ailang eval-suite` that, instead of `--benchmarks <comma-list>` or `-tier smoke`, takes a rating database and a confidence target:

```bash
# Re-run only benchmarks where one more trial would meaningfully update belief
ailang eval-suite -agent \
  -models opencode-gemma4-26b-ailang \
  --benchmarks-by-confidence ~/.ailang/state/ratings.db \
  --confidence-target 0.9 \
  --max-benchmarks 5
```

Selection algorithm:

1. Load (benchmark, model) ratings + trial-count history from `ratings.db`
2. For each benchmark, compute information gain from one more trial (Bayesian: how much would the posterior shift?)
3. Sort by information gain descending
4. Take top-N (capped at `--max-benchmarks`)
5. Skip benchmarks that:
   - Have ≥ N=5 trials at the current code version with pass rate ≥95% or ≤5% (saturated either way)
   - Haven't changed compiler/prompt version since last run (no new signal possible)

Concrete payoff: a "did this morning's compiler fix help?" check, which today is a 3-4 hour full re-rotation, becomes a 20-30 minute targeted re-run of just the 3-5 benchmarks the change was likely to affect.

### Rating database (`ratings.db`)

New SQLite table alongside `observatory.db`:

```sql
CREATE TABLE benchmark_ratings (
  benchmark_id TEXT NOT NULL,
  rating REAL NOT NULL DEFAULT 1500.0,
  n_trials INTEGER NOT NULL DEFAULT 0,
  last_updated TIMESTAMP NOT NULL,
  PRIMARY KEY (benchmark_id)
);

CREATE TABLE model_ratings (
  model_id TEXT NOT NULL,
  rating REAL NOT NULL DEFAULT 1500.0,
  n_trials INTEGER NOT NULL DEFAULT 0,
  k_factor INTEGER NOT NULL DEFAULT 32,
  last_updated TIMESTAMP NOT NULL,
  PRIMARY KEY (model_id)
);

CREATE TABLE trial_history (
  trial_id TEXT PRIMARY KEY,
  benchmark_id TEXT NOT NULL,
  model_id TEXT NOT NULL,
  outcome INTEGER NOT NULL,  -- 0 or 1
  prompt_version TEXT,
  sampling_config TEXT,
  compiler_version TEXT,     -- git SHA
  benchmark_rating_before REAL,
  model_rating_before REAL,
  benchmark_rating_after REAL,
  model_rating_after REAL,
  recorded_at TIMESTAMP NOT NULL
);
```

Bootstrapping: the weekend's 102 trials become the seed. Replay them through the ELO update to get the first set of ratings before going live.

## Tier saturation analysis

At each release (or any time via `ailang eval-trend tier-saturation`), generate:

```
Tier saturation report — gemma4:26b-ailang, v0.23.0:
  smoke (17 benchmarks):
    14 saturated (100% pass rate over last 3 rotations)
      → gcd_lcm, recursion_fibonacci, numeric_modulo, ... (12 more)
    3 discriminating
      → balanced_parens (66%), canonical_normalization (66%), binary_tree_sum (50%)
    0 unsaturated (0% pass)

  Recommendation: smoke is 82% saturated. Consider:
    - Promoting trivial benchmarks (gcd_lcm, etc.) to "warmup" tier
    - Running core tier next rotation to find new discriminators
    - Re-running only the 3 discriminating benchmarks for routine drift detection

  core (20 benchmarks): not yet evaluated for this model
  stretch (8 benchmarks): not yet evaluated
```

This output becomes the recommendation for the next rotation, dropping into the design-doc-creator skill if a tier-promotion candidate is identified.

## What stays the same

- The eval rig itself (verify_setup.sh, ollama config, opencode wiring, sampling Modelfile, MCP, slim prompt) — no changes needed
- Existing `eval-suite -tier <name>` mode — still works, runs the full tier
- `eval-trend candidates` — surfaces persistent failures; works alongside ratings as a different lens
- The result JSON schema — additive only (rating fields appended)

## Implementation plan (3-4 day sprint)

**Day 1**: ELO update logic + `ratings.db` schema. Replay weekend's 102 trials to seed. Unit tests against synthetic trial sequences.

**Day 2**: `ailang eval-suite --benchmarks-by-confidence` mode. Bayesian-ish information-gain selection. Test by running on the weekend's data and confirming the suggested benchmarks match intuition (balanced_parens, binary_tree_sum, dense_operator_program).

**Day 3**: `ailang eval-trend tier-saturation` command. Renders the saturation report.  Integrate ratings into `eval-publish` leaderboard tables (per-benchmark difficulty column).

**Day 4**: Cross-model leaderboard view + multi-model rating computation. Validate the "add a new model in 8 trials" workflow end-to-end on whatever second model gets pulled (qwen3-coder:30b candidate).

## Conflict surface

Touches:
- `internal/eval_harness/ratings.go` (new) — ELO update logic
- `internal/observatory/ratings.go` (new) — SQLite store
- `cmd/ailang/eval_suite.go` — new flag wiring
- `cmd/ailang/eval_trend.go` — new `tier-saturation` action
- `cmd/ailang/eval_publish.go` — rendering integration
- `internal/eval_harness/rotation_summary.go` — appends rating updates

Does NOT touch parser/typechecker/codegen/runtime — pure harness extension.

## Cross-references

- M-EVAL-METRICS-TAXONOMY (planned) — provides the `cell_metrics` schema this layers on top of
- M-AILANG-ERROR-QUALITY (planned) — error-quality work creates rating-changing events (a compiler fix shifts benchmark ratings down for models that newly pass)
- M-EVAL-FINETUNING-DATA-PIPELINE (planned) — the cross-model leaderboard makes "which model do we fine-tune?" a data-driven decision instead of a guess

## What this gives us strategically

Today:
- "Run smoke tier" = 17 benchmarks × N trials, mostly redundant
- "Is this model better than that one?" = manual comparison, only on identical benchmark sets
- "Did this compiler fix help?" = full re-rotation (3-4 hours)

With ratings + selective reruns:
- "Run smoke tier" = run benchmarks at the model's rating band ± window (5-8 trials, ~30 min)
- "Is this model better than that one?" = single ELO comparison, valid even across different benchmark sets
- "Did this compiler fix help?" = re-run the 3-5 benchmarks the change was likely to affect (~20 min)

5-10x compute efficiency. Same model fidelity. Plus the rating itself becomes a publishable artifact (per-release leaderboard ranks every model the rig has tested by ELO, regardless of which subset they ran).

This is the rig graduating from "throughput infrastructure" to "decision infrastructure."

## Parallelism caveat (re-confirmed 2026-05-24)

We've explored whether parallelism can speed any of this. **It cannot on a single M4 Max:**

- 128 GB unified memory: ample headroom (we use ~21 GB at NUM_PARALLEL=1)
- BUT single integrated 40-core GPU is **memory-bandwidth bound at 27B+** ([Starmorph: Apple Silicon LLM Inference Optimization](https://blog.starmorph.com/blog/apple-silicon-llm-inference-optimization-guide))
- Two concurrent ollama requests SHARE the same bandwidth → ~1x throughput with 2x latency = no net gain
- We verified this empirically Friday: NUM_PARALLEL=4 caused 15-min TTFT timeouts; NUM_PARALLEL=1 is the only stable mode

**Conclusion**: Rig is bottlenecked by serial GPU compute, not memory. Want more throughput? Need (a) selective reruns (this design doc), (b) bigger machine (2x M4 Ultra or Apple equivalent for 2 concurrent rigs), or (c) horizontal scaling (multiple Mac Studios as a small cluster). All three are valid future directions.

The selective rerun approach is the highest-leverage of the three because it requires no new hardware.
