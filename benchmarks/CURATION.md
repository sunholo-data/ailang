# Benchmark Curation Guide

This document defines how AILANG benchmarks are organized, tagged, rotated, and
promoted/demoted between tiers. It is the source of truth for new benchmark
authors and for anyone deciding whether a benchmark still earns its place in
the suite.

The tiering and tagging schema was introduced in **v0.14.0** (sprint
`M-EVAL-SUITE-PREP`) and replaces the earlier flat suite + ad-hoc vision list.

---

## 1. Tier definitions

Every benchmark YAML declares exactly one tier. Tiers express *what signal the
benchmark produces* — not difficulty alone. A difficult benchmark that every
frontier model trivially passes is not a useful `stretch` benchmark.

| Tier      | Purpose                                        | Expected pass rate (Core ≈ frontier avg) | Run cost  |
|-----------|------------------------------------------------|-----------------------------------------|-----------|
| `smoke`   | Regression gate. Fast, near-100% everywhere.   | ≥ 95% AILANG, ≥ 90% Python              | ~seconds  |
| `core`    | **The headline number.** Representative corpus.| 70–95%. This is what releases are judged on. | ~minutes |
| `stretch` | Headroom / differentiation benchmarks.         | 30–70%. Non-trivial for every model.    | ~minutes  |
| `vision`  | Aspirational. May not compile on AILANG today. | 0–50%. Measures *potential*, not parity.| variable  |

**Default tier**: `core`. `LoadSpec` (`internal/eval_harness/spec.go`) fills in
`tier: core` when the field is absent, so unannotated benchmarks automatically
land in the headline set.

**CI gate**: the dashboard headline metric is `tiers.core.ailang_success_rate`
from `docs/static/benchmarks/latest.json`. If that number regresses between
baselines we investigate before releasing.

---

## 2. Tag taxonomy

Every benchmark declares **1–3 tags** describing the AILANG feature surface it
exercises. Tags are orthogonal to tiers: a `smoke` benchmark and a `vision`
benchmark can share `[adt_pattern_match]` — they test the same feature at
different difficulty levels.

The canonical tag set (do **not** invent new tags without updating this list):

| Tag                  | Exercises                                                          |
|----------------------|--------------------------------------------------------------------|
| `adt_pattern_match`  | Sum types, `match`, exhaustiveness, nested patterns                |
| `algorithmic`        | Numeric / classical algorithms (sorting, gcd, dp, graph search)    |
| `contracts`          | `requires` / `ensures`, invariants, property-style checks          |
| `data_transform`     | List/record shaping, map/filter/reduce pipelines, JSON shuffling   |
| `effects_io`         | `! {IO}`, `! {FS}`, stdin/stdout plumbing, effect handlers         |
| `error_handling`     | `Result` / `Option` threading, typed failures, refusal patterns    |
| `functional`         | Higher-order functions, currying, composition, closures            |
| `records`            | Record literals, field access, `{ r with ... }` update syntax      |
| `recursion`          | Structural recursion, accumulator patterns, mutual recursion       |
| `state_machine`      | Explicit state ADTs + transition functions                         |
| `string_algo`        | String parsing, encoding, formatting, tokenization                 |
| `type_safety`        | Type-level guarantees: phantom types, GADT-like refinement, row polymorphism |

Multi-feature benchmarks (e.g. `tree_transformation_pipeline` touches records,
recursion, and pattern matching) should list all three. If a benchmark needs a
fourth tag, it is probably doing two things — consider splitting it.

Tags power:
- `ailang eval-matrix --by-tags` — per-tag AILANG vs Python delta table.
- Dashboard benchmark tag chips (from `benchmarks.<id>.tags` in `latest.json`).
- `jq` queries in [`.claude/skills/eval-analyzer/resources/jq_queries.md`](../.claude/skills/eval-analyzer/resources/jq_queries.md).

---

## 3. Writing a new benchmark YAML

Required fields (beyond the existing `id` / `description` / `languages` / `entrypoint`):

```yaml
tier: core                          # smoke | core | stretch | vision
tags: [recursion, adt_pattern_match]  # 1-3 from the canonical list
```

Full example (abridged from `contract_bst_validate.yml`):

```yaml
id: contract_bst_validate
description: "BST insert, search, validation — ADT recursion with invariants"
difficulty: hard
category: data_structure
languages: ["ailang", "python"]
entrypoint: "main"
caps: ["IO"]
tier: core
tags: [contracts, recursion, adt_pattern_match]
```

**Validation** (`internal/eval_harness/spec.go`): `LoadSpec` rejects benchmarks
whose `tier` is outside the enum or whose `tags` are empty, >3, or unknown. CI
runs this validation on every benchmark at startup, so a malformed YAML fails
the eval-suite command immediately.

### Source constraints (constrained-construction benchmarks)

`source_constraints` grades the program TEXT itself, before execution —
unlocking benchmarks whose difficulty cannot be delegated to the program
(byte-precise self-accounting, lexically-constrained construction):

```yaml
source_constraints:
  exact_bytes: 256              # normalized source must be exactly N bytes
  # max_bytes: 400              # or: at most N bytes (mutually exclusive)
  # banned_chars: "0123456789"  # none of these characters anywhere
  # banned_substrings: ["**"]   # none of these substrings anywhere
```

Normalization: CRLF/CR → LF, then ALL trailing newlines stripped
(`NormalizeSource` in `internal/eval_harness/source_constraints.go`). A
violation fails the run with `error_category: constraint_violation`; the code
is never executed. The violation message (with exact byte deltas / offending
lines) is fed to the one-shot self-repair attempt. The constraint MUST be
stated verbatim in `task_prompt` — models are graded only against rules they
were told. Standard (0-shot) mode only for now; agent mode ignores them.

### Sizing rule for scale-sensitive benchmarks

The harness kills a run at **30s** (`eval-suite --timeout` default), and the
AILANG tree-walking interpreter is 1–2 orders of magnitude slower than CPython
on hot loops. A benchmark must fail on **logic, not runtime** — a correct but
unoptimized AILANG solution that times out contaminates the AILANG-vs-Python
comparison. Therefore, for any benchmark whose input size is a free parameter
(generated streams, large lists, deep iteration):

1. **One language-neutral N** — never per-language input sizes; the
   `expected_stdout` must be identical across languages.
2. **Reference headroom gate**: a hand-written AILANG reference solution
   (checked into `benchmarks/frontier_refs/` or a sibling refs dir) must run in
   **≤20% of the timeout** (≈6s). Model solutions are routinely 2–5× slower
   than a tuned reference; 20% keeps them inside the budget.
   (Empirical: `stream_lcg_topk` at N=5000 ran 13s — 43% of budget, too hot;
   N=2000 runs ~2s and keeps all three output lines scale-sensitive.)
3. **Prefer precision traps over asymptotic traps** — off-by-one, exact modular
   arithmetic, 64-bit overflow discipline discriminate at small N. Punishing
   O(n²)-vs-O(n log n) needs an N too large for the current interpreter;
   defer true asymptotic-complexity benchmarks until the bytecode VM/perf work
   lands, then revisit N (and this rule's 20% figure) at that baseline.

---

## 4. Rotation rules

The eval suite is not immutable. Benchmarks are rotated on every release cycle
using the primitives in `internal/eval_analysis/`:

- **Saturation detector** (`IsSaturated`): flags benchmarks at 100% pass across
  *every* model × language pair. Surfaced via `eval-matrix --show-saturated`.
- **AILANG-only-wins detector** (`AILANGOnlyWins`): flags `(benchmark, model)`
  cells where AILANG passes and Python fails. Surfaced via `eval-matrix
  --ailang-wins`.
- **Refusal detector** (`DetectRefusal`): flags runs where a model refused the
  task rather than attempting code. Surfaced in the failure taxonomy.

Rotation cadence: review before each minor release (every ~2 weeks). The
`eval-analyzer` skill's `benchmark_health.sh` script produces the rotation
candidate list.

---

## 5. Promotion and demotion criteria

Tier membership is not fixed. Apply these rules at release review:

### Promotion (move **up** the tier ladder)

| Current → New      | Trigger                                                                   |
|--------------------|---------------------------------------------------------------------------|
| `stretch` → `core` | ≥ 80% AILANG + ≥ 80% Python pass for **two consecutive baselines**        |
| `core` → `smoke`   | ≥ 95% AILANG + ≥ 95% Python pass for **three consecutive baselines**, AND benchmark is ≤ 50 LOC |
| `vision` → `stretch` | AILANG reaches ≥ 30% pass (i.e. the benchmark is no longer aspirational) |

Promotion moves a benchmark *out of* the headline `core` metric when it
saturates upward into `smoke`, and *into* the headline when `stretch` work
matures. This keeps the headline metric honest — it always tracks benchmarks
that are genuinely discriminating.

### Demotion (move **down** the tier ladder)

| Current → New        | Trigger                                                                     |
|----------------------|-----------------------------------------------------------------------------|
| `smoke` → `core`     | Any regression to < 90% AILANG pass. Investigate before demoting.           |
| `core` → `stretch`   | < 30% AILANG pass for **two baselines** *and* the benchmark is accurate. If it's just a prompt / stdlib gap, fix the language instead. |
| `core` → retired     | Known-broken in AILANG for reasons unlikely to change (e.g. requires an unimplemented language feature). Move YAML to `benchmarks/retired/` with a note. |

### Retirement (remove from suite entirely)

Retire a benchmark when:
- It is **saturated** (100% across all models × all languages) *and* no longer
  differentiates models. A retired saturated benchmark can be brought back if
  a regression is suspected.
- The scenario is **no longer representative** of real AILANG usage (e.g. a
  benchmark that tests a syntax we removed).
- It has been **superseded** by a more focused benchmark covering the same
  tag.

Retired YAMLs move to `benchmarks/retired/` (not deleted) so historical
baselines remain reproducible.

---

## 6. Sanity checks before merging a new benchmark

1. **YAML validates**: `ailang eval-suite --tier <new_tier> --benchmarks <new_id> --models claude-haiku-4-5` runs without error.
2. **Python reference works**: the `python.source_path` file (if cross-language) runs stand-alone and produces the expected stdout.
3. **AILANG reference works**: `ailang run examples/<related>.ail` produces the expected stdout if a reference exists.
4. **Tag coverage**: if your new benchmark introduces a tag not in §2, update this document *and* every existing benchmark that also exercises the new feature.
5. **Dashboard round-trip**: regenerate `latest.json` via `ailang eval-report
   <baseline_dir> <version> --format=json` and verify the new benchmark
   appears under its declared tier in `tiers.<tier>.benchmark_count`.

---

## 7. Where to look when the numbers move

- **`tiers.core.*` drops** → triage against `ailang eval-matrix --by-tags` and
  the dashboard's per-benchmark delta; probably a language regression.
- **`tiers.smoke.*` drops** → treat as a release blocker; smoke tier is
  supposed to be flat near 100%.
- **`tiers.stretch.*` climbs above 70%** → promotion candidates (see §5).
- **`tiers.vision.*` climbs above 30%** → promotion candidates; also update
  `benchmarks/VISION_BENCHMARKS.md`.
- **`--show-saturated` list grows** → rotation candidates (see §4).
- **`--ailang-wins` list shrinks** → AILANG's differentiation is eroding;
  investigate whether Python baselines got better or AILANG regressed.

See [docs/docs/guides/evaluation/README.mdx](../docs/docs/guides/evaluation/README.mdx)
for the human-facing evaluation guide and the tier structure section.

---

## 8. Suite-change events (`events.yml`)

Every time the benchmark set changes in a way that moves the dashboard
time-series, record the event in [`events.yml`](events.yml). These events
render as dashed `ReferenceLine` annotations on the per-model trend,
per-model delta, and overall success charts so readers can see *why* a
number jumped between two releases.

**What counts as an event:**

| Kind               | When to record                                                      |
|--------------------|---------------------------------------------------------------------|
| `benchmark_add`    | One or more new benchmarks landed in a release                      |
| `benchmark_remove` | A benchmark was retired or moved to `benchmarks/retired/`           |
| `taxonomy`         | Tier mapping shifted, tag list changed, or tier thresholds moved    |
| `prompt`           | The system prompt or `ailang prompt` output materially changed      |

Everything else — parser/stdlib changes that affect success rate, model
additions, infra upgrades — is **not** an event. Those show up in the
numbers themselves; events only exist for *suite* changes that the reader
would otherwise mistake for a language-level regression.

**Schema** (see [`internal/eval_analysis/types.go`](../internal/eval_analysis/types.go) `SuiteEvent`):

```yaml
- version: v0.14.0           # required — when this change shipped
  label: "Tier + tag taxonomy" # required — short string drawn on the chart
  kind: taxonomy             # required — one of the kinds above
  color: "#E67E22"           # optional — overrides annotationColor() default
  affects_tiers: [stretch]   # optional — when set, only render when tier selected
```

**When `affects_tiers` is set**, the event is hidden unless the dashboard's
TierToggle matches. Use this for changes that only move *one* tier's numbers
(e.g. a +2 stretch-tier addition should not decorate the Core chart).

**Authoring workflow** (part of the release checklist):

1. While preparing the release, open `benchmarks/events.yml`.
2. Append any events that describe suite changes in this release — check the
   `git diff` of `benchmarks/*.yml` to catch adds/removes.
3. Use the earliest version the change was visible in (usually the release
   version being prepared).
4. Run `ailang eval-report <baseline_dir> <version> --format=json` and
   verify the new entries appear under `.events[]` of `latest.json`.
5. Commit the YAML change as part of the release; the `post-release`
   skill does not auto-populate this file.

**Example workflow** — adding two stretch benchmarks in v0.14.0:

```yaml
- version: v0.14.0
  label: "+2 stretch/vision benchmarks"
  kind: benchmark_add
  affects_tiers: [stretch, vision]   # hidden when Core is selected
```

The dashboard will now show a dashed line at v0.14.0 on every time-series
chart when Stretch or Vision is the active tier, and no line when Core is
active — avoiding a visual distraction that doesn't match the data.

**Rule of thumb**: if a reader asks "why did the number jump here?" and the
answer is "we changed the benchmark set," an event entry belongs there.

