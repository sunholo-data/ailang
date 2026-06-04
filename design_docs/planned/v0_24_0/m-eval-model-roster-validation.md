# Eval Model Roster Validation

**Status**: Planned
**Target**: v0.24.0
**Priority**: P1 (Medium)
**Estimated**: 3 days (10h implementation + 6h testing + 2h docs + buffer)
**Dependencies**: None (operates on existing `internal/eval_harness/models.yml`)

---

## Problem Statement

The canonical eval model roster lives in `internal/eval_harness/models.yml` (**83 models**,
**44 `model_family` groupings**). Every time a vendor ships a new model, a human or agent edits
this file by hand: adding an entry, picking pricing, and judging whether the new model is
*actually newer* than what we already run. **None of that is validated by a machine.** The result
is recurring, silent roster drift that only surfaces when someone notices a benchmark is quoting
the wrong model.

**Current State:**

- **No machine-readable release date.** Release dates exist only as free-text comments
  (`# GLM-5.1 (Apr 2026)`) above **25 of 83** entries — the other **58 have no date at all**.
  The `ModelConfig` struct (`internal/eval_harness/models.go:16`) has **zero** date fields.
  "Which model is newest" is decided by eyeballing version numbers in YAML comments.
- **Version numbers are not monotonic across vendors.** The roster itself documents this:
  `internal/eval_harness/models.yml:894` literally says
  `z-ai versioning is non-linear: 5.1 > 5 > 4.7 > 4.6 > 4.5 by date`. Nothing enforces it.
- **This already caused a reactive fix.** Commit `b32dc9a2`
  ("eval: fix model versioning — remove 4 regressions, add actual new models") had to *remove*
  four entries that had been added as "new" but were actually **older** than models already in the
  roster (`glm-4.5` Jul 2025 < `glm-5` Feb 2026; `kimi-k2.5` Jan 2026 < `kimi-k2.6` Apr 2026;
  `deepseek-v3.2` Dec 2025 < `v4-flash` Apr 2026). That is the second hand-correction of this same
  class of error.
- **No pricing/identity guard.** A zero or missing `pricing` block silently produces `$0.00` cost
  estimates — a direct violation of CLAUDE.md Principle #2 ("NO SILENT FALLBACKS" explicitly names
  pricing and model configs). Nothing fails when pricing is absent.
- **No suite-integrity check.** The suite lists (`benchmark_suite`, `agent_suite`, `ollama_suite`,
  `dev_models`, …) reference model keys by string. A typo or a deleted model leaves a dangling
  reference that is only discovered when an eval run blows up mid-flight.

**Impact:**

- **Who:** eval maintainers (human + the model-manager agent), and anyone reading a published
  baseline. AILANG's headline numbers ("model X scores Y on AILANG") are only as trustworthy as the
  roster they were run against.
- **How significant:** recurring. The failure is *silent* — a wrong-but-plausible roster produces a
  clean run with wrong labels. Each occurrence costs real API spend on the wrong model plus the
  human time to notice and hand-correct (twice so far). This is the textbook "incremental
  special-casing" anti-pattern called out in CLAUDE.md Principle #3: we keep patching individual
  bad entries instead of making the bad entry *impossible to commit*.

---

## Goals

**Primary Goal:** Make every class of roster error that has actually occurred (stale "new" model,
missing pricing, dangling suite reference) **fail loudly at validation time** — in CI and as a
pre-eval gate — instead of being caught by a human after the fact.

**Success Metrics:**

- Structured `released` (ISO `YYYY-MM-DD`) field present on **100%** of models (83/83), up from
  **0/83** machine-readable today.
- `ailang eval-validate-models` (and `make validate-models` in CI) catches **all four** regression
  classes from `b32dc9a2`: stale-vs-existing, intra-family date/version inversion, missing pricing,
  dangling suite reference — verified by a regression fixture that reproduces the `b32dc9a2` state
  and must fail.
- **Zero** roster entries with `pricing.input_per_1k == 0 && output_per_1k == 0` unless explicitly
  marked `pricing_exempt: true` (e.g. local Ollama models).
- Hand-correction commits for "wrong model is newer" drop to **0** after adoption (tracked over the
  release following ship).

---

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Add a structured `released: YYYY-MM-DD` field to `ModelConfig` (vs. continue parsing comments) | Becomes the machine-readable source of truth for "newest"; every downstream check depends on its shape | human | design | high |
| `released` is **required** for all non-exempt models (validation fails if absent) | Determines whether this is a real guard or an advisory lint; "optional" reduces it to noise | human | design | high |
| Validation is a **hard gate** in CI + pre-eval, not a warning | A warning gets ignored; the whole point is to make bad rosters un-shippable | human | design | high |
| "Newest in family" = **max `released` date**, ties broken by explicit `supersedes` link, never by version-number string compare | This is the exact bug from `b32dc9a2`; version strings are non-monotonic across vendors | human | design | high |
| Zero-pricing requires explicit `pricing_exempt: true` opt-out | Ties to CLAUDE.md #2 (no silent fallback on pricing); opt-out keeps local models working | human | design | med |
| Validator is a new pure-Go function in `internal/eval_harness/` reusing `LoadModelsConfig`, surfaced via a CLI subcommand | Keeps one loader, one schema; avoids a second parallel parser | agent | implementation | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [ ] `released` field is added to `ModelConfig` as `yaml:"released"`, ISO `YYYY-MM-DD` string.
- [ ] `released` is **required** for every model unless `pricing_exempt`/`local` (final exemption
      predicate to be named in review).
- [ ] Validation runs as a **hard gate** (non-zero exit) in CI (`make validate-models`) and at the
      start of every eval run.
- [ ] "Newest" is computed from `released` dates only; version-number comparison is **never** used
      for recency.
- [ ] Backfill policy for the 58 dateless models (block-on-missing vs. one-time backfill PR first —
      see Deferred).

### Deferred Decisions

The following are intentionally left open for the implementer:

- Exact CLI subcommand name (`eval-validate-models` vs. `eval-models validate`) — agent may choose,
  following existing `ailang eval-*` naming.
- Internal validator function/file naming and how rules are organized (one big function vs. a slice
  of rule funcs) — agent may choose.
- Output format of the validation report (table vs. one-line-per-violation) — agent may choose;
  must be greppable and list model key + rule + reason.
- Whether the 58-model date backfill ships as a **prerequisite PR** before the gate turns on, or in
  the same PR behind a temporary allowlist — human at review.
- Whether `supersedes` (explicit predecessor link) ships now or is deferred to Future Work; the
  date-based check works without it — human at review.

---

## Solution Design

### Overview

Promote release date and pricing from "comment a human reads" to "field a machine checks." Add a
structured `released` date (and a small `pricing_exempt` escape hatch) to the existing
`ModelConfig`, then add one validator that loads the roster through the *existing* `LoadModelsConfig`
path and asserts a fixed set of invariants. Wire that validator into CI (so a bad roster can't
merge) and into the eval entry point (so a bad roster can't run). No new model loader, no second
schema — one source of truth, validated.

The design deliberately encodes the *actual* failures that have happened, not hypothetical ones.
Each rule maps to a real incident or a stated project principle.

### Architecture

**Components:**

1. **Schema extension** (`internal/eval_harness/models.go`): add
   `Released string \`yaml:"released"\`` and `PricingExempt bool \`yaml:"pricing_exempt"\`` to
   `ModelConfig`. Parse `Released` into a `time.Time` once at load for comparison; keep the raw
   string for round-tripping.
2. **Validator** (`internal/eval_harness/models_validate.go`): a pure function
   `ValidateRoster(cfg *ModelsConfig) []RosterViolation` returning a structured list. Each rule is
   independent and testable. No I/O — takes the already-loaded config.
3. **Rule set** — the invariants (see below).
4. **CLI surface** (`cmd/ailang/`): `ailang eval-validate-models` → loads `models.yml`, runs
   `ValidateRoster`, prints violations, exits non-zero if any.
5. **CI + pre-eval wiring**: `make validate-models` target in CI; a `ValidateRoster` call at the
   top of the eval run path that aborts before spending API budget.

**The rules** (each is one independent check):

| Rule | Invariant | Maps to incident / principle |
|------|-----------|------------------------------|
| R1 `released-present` | Every non-exempt model has a parseable `released` date | Root cause of `b32dc9a2` |
| R2 `family-recency-monotonic` | Within a `model_family`, no model has a higher version label but an *earlier* `released` date than another in the same family | The non-linear-versioning trap (`models.yml:894`) |
| R3 `pricing-present` | Non-exempt models have `pricing.input_per_1k > 0 && output_per_1k > 0` | CLAUDE.md #2 (no silent pricing fallback) |
| R4 `suite-refs-resolve` | Every key in `benchmark_suite`/`agent_suite`/`ollama_suite`/`dev_models`/etc. exists in `models` | Dangling-reference run failures |
| R5 `api-name-unique` | No two model keys share an identical `api_name` + `provider` unless intentionally aliased | Accidental duplicate roster entries |
| R6 `default-resolves` | `default` and each suite's first entry resolve to a real model | Boot-time roster sanity |

R2 is the heart of it: it does **not** rank versions itself. It only flags the *contradiction*
"this entry claims a newer label but an older date" so a human resolves it explicitly — exactly the
judgment that was wrong in `b32dc9a2`.

### Implementation Plan

**Phase 1: Schema + loader (~3h)**
- [ ] Add `Released` and `PricingExempt` to `ModelConfig` (`models.go:16`).
- [ ] Parse `Released` to `time.Time` at load; surface a clear error on malformed dates.
- [ ] Unit-test load with valid/invalid/missing `released`.

**Phase 2: Validator + rules (~4h)**
- [ ] Create `models_validate.go` with `RosterViolation` and `ValidateRoster`.
- [ ] Implement R1–R6 as independent rule functions.
- [ ] Table-driven unit tests per rule (one passing + one failing fixture each).
- [ ] **Regression fixture**: reproduce the pre-`b32dc9a2` roster state; assert R1/R2 flag all four
      removed entries.

**Phase 3: CLI + CI + pre-eval gate (~3h)**
- [ ] Add `ailang eval-validate-models` subcommand.
- [ ] Add `make validate-models` and wire into CI.
- [ ] Call `ValidateRoster` at eval-run entry; abort with the violation report before any API spend.
- [ ] Backfill `released` for the 58 dateless models from existing comments (mechanical, reviewed).

### Files to Modify/Create

**New files:**
- `internal/eval_harness/models_validate.go` (~220 LOC) — `RosterViolation`, `ValidateRoster`, R1–R6.
- `internal/eval_harness/models_validate_test.go` (~300 LOC) — per-rule + `b32dc9a2` regression fixture.
- `internal/eval_harness/testdata/roster_b32dc9a2_regression.yml` (~60 LOC) — frozen bad-roster fixture.

**Modified files:**
- `internal/eval_harness/models.go` (+25/-2 LOC) — `Released`/`PricingExempt` fields + parse.
- `internal/eval_harness/models.yml` (+~110/-0 LOC) — backfill `released:` on all 83 models.
- `cmd/ailang/` eval command file (+40 LOC) — `eval-validate-models` subcommand.
- `Makefile` (+4 LOC) — `validate-models` target + CI hook.

---

## Examples

### Example 1: The `b32dc9a2` regression, caught at validation time

**Before (today — silent, merges clean):**
```yaml
# someone adds, believing it's newer:
glm-4-5:
  api_name: "z-ai/glm-4.5"
  model_family: "glm-5"
  # (no released field; reviewer eyeballs "4.5" and assumes it's recent)
```
Run proceeds. Baseline quotes `glm-4.5` as a current model. Nobody notices until a human audits.

**After (`ailang eval-validate-models` fails the commit):**
```
✗ roster validation: 2 violations

  [R1 released-present]   glm-4-5: missing required `released` date
  [R2 family-recency]     glm-4-5 (released 2025-07) labeled '4.5' but family 'glm-5'
                          contains motoko-glm-5 (released 2026-02, label '5') —
                          higher version label must not predate a lower one.
                          Resolve: confirm date or remove entry.

make: *** [validate-models] Error 1
```

### Example 2: Missing pricing no longer silently bills $0

**Before:**
```yaml
new-model:
  api_name: "vendor/new-model"
  # pricing block forgotten → cost estimator reports $0.00, budgets never trip
```

**After:**
```
✗ [R3 pricing-present] new-model: pricing.input_per_1k and output_per_1k are both 0
                       (set real pricing, or mark `pricing_exempt: true` for local models)
```

### Example 3: New `released` field on a real entry

```yaml
motoko-glm-5-1:
  api_name: "z-ai/glm-5.1"
  model_family: "glm-5"
  released: "2026-04-07"        # NEW: machine-readable, replaces the comment
  pricing:
    input_per_1k: 0.0006
    output_per_1k: 0.0022
```

---

## Success Criteria

- [ ] `Released` + `PricingExempt` fields added; `LoadModelsConfig` parses and validates date format.
- [ ] All 83 models carry a `released` date (or `pricing_exempt`/local exemption).
- [ ] `ValidateRoster` implements R1–R6, each independently unit-tested.
- [ ] Regression fixture reproducing the pre-`b32dc9a2` roster **fails** validation on R1+R2.
- [ ] `ailang eval-validate-models` exits non-zero on any violation, zero on a clean roster.
- [ ] `make validate-models` runs in CI and blocks merge on violations.
- [ ] Eval run path calls `ValidateRoster` and aborts before API spend on a bad roster.
- [ ] All tests passing (90%+ coverage on new code).
- [ ] Documentation updated (CHANGELOG.md, model-manager skill, CLAUDE.md eval section if needed).

## Conflict Surface

**Not applicable.** This change touches only `internal/eval_harness/` (config schema + validation)
and `cmd/ailang/` (a new read-only subcommand). It does **not** modify
`internal/parser/`, `internal/lexer/`, `internal/ast/`, `internal/types/`, `internal/elaborate/`,
`internal/iface/`, `internal/codegen/`, `internal/eval/`, `internal/vm/`, `internal/effects/`, or
any compilation entry point, so it cannot affect parsing or type-checking of AILANG programs. The
only backward-compatibility surface is `models.yml`: adding fields is additive (existing readers
ignore unknown keys is *not* assumed — `LoadModelsConfig` uses typed structs, and the two new fields are
additive to the struct), and turning the gate on requires the backfill PR to land first (tracked in
Design Freeze).

> Note: per the design-doc-creator hard gate, no "AILANG supports / does not support X" language
> claim is made anywhere in this doc — it is pure Go tooling — so no `ailang check` transcript is
> required.

## Testing Strategy

**Unit tests:**
- Per-rule (R1–R6): one passing roster and one violating roster each (table-driven).
- Date parsing: valid ISO, malformed, missing, empty.
- `pricing_exempt` correctly suppresses R3.

**Regression-surface tests:**
- `roster_b32dc9a2_regression.yml`: a frozen copy of the bad pre-fix roster. Test asserts R1+R2
  flag all four entries removed in `b32dc9a2`. This pins the exact failure the feature exists to
  prevent — a failure here is a real regression, not test churn.

**Integration tests:**
- `ailang eval-validate-models` against the live `models.yml` returns clean (exit 0) — guards the
  shipped roster permanently.
- Eval-run entry aborts (no provider call) when handed a bad roster.

**Manual testing:**
- Add a deliberately-stale model, confirm `make validate-models` fails; remove it, confirm pass.

## Non-Goals

- **Auto-fetching release dates / pricing from vendor APIs** — out of scope; dates are entered by
  the editor and *checked* here, not sourced. [Future Work]
- **Ranking models by quality/benchmark score** — this validates roster *integrity*, not model
  *performance*. [Out of scope]
- **A general YAML schema/linter framework** — six concrete rules tied to real incidents, not a
  generic validation engine. [Avoids scope creep]
- **Changing how evals select or run models** — selection logic is untouched; only a pre-flight gate
  is added. [Out of scope]

## Timeline

**Days 1–2 (10h):**
- Phase 1: schema + loader (3h)
- Phase 2: validator + R1–R6 + regression fixture (4h)
- Phase 3: CLI + CI + pre-eval gate (3h)

**Day 3 (6h):**
- Backfill `released` on all 83 models from existing comments (2h)
- Tests to 90% coverage + integration (3h)
- Docs (CHANGELOG, model-manager skill) (1h)

**Total: ~3 days (16h work + buffer)**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Backfilling 58 dateless models introduces a wrong date | Medium | Source dates from existing comments; review diff; R2 cross-checks intra-family consistency and will flag contradictions |
| Hard gate blocks an urgent legitimate roster change | Medium | `pricing_exempt`/explicit per-rule allowance for documented exceptions; gate reports the exact rule + fix |
| R2's version-label parsing misreads an exotic name | Low | R2 only flags *contradictions* for human resolution, never auto-rejects on label alone; unparseable labels skip R2 (still covered by R1 date presence) |
| Pre-eval gate adds latency to every run | Low | Validation is in-memory over 83 entries (<1ms); runs once at startup |

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Roster recency becomes a deterministic function of dates, not human judgment of version strings |
| A2: Replayability | +1 | A validated, dated roster makes "which model produced this baseline" reproducible |
| A3: Effect Legibility | 0 | No effect-system impact |
| A4: Explicit Authority | 0 | No capability/authority changes |
| A5: Bounded Verification | +1 | Roster correctness is now a local, bounded check (CI + pre-eval) instead of post-hoc human audit |
| A6: Safe Concurrency | 0 | No concurrency impact |
| A7: Machines First | +1 | Replaces human-read comments with machine-checkable fields; the whole feature optimizes for machine verification |
| A8: Minimal Syntax | 0 | Two additive YAML fields, no language syntax |
| A9: Cost Visibility | +1 | R3 guarantees pricing is present, so cost/budget estimates can't silently read $0 |
| A10: Composability | +1 | Reuses existing `LoadModelsConfig`; validator composes as one rule list, extensible per-rule |
| A11: Structured Failure | +1 | Violations are structured `RosterViolation` values with rule + reason, fail loudly (CLAUDE.md #2) |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +8** ✅ Proceed to implementation

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced — removes a source of human variance.
- [x] A3 (Effects): No hidden side effects; validator is pure, CLI is read-only.
- [x] A4 (Authority): No ambient access granted.
- [x] A7 (Machines First): Directly advances machine-first by replacing comments with checked fields.

## References

- **Motivation**: commit `b32dc9a2` ("eval: fix model versioning — remove 4 regressions, add actual
  new models") — the second reactive hand-correction of stale roster entries.
- **Roster source of truth**: `internal/eval_harness/models.yml`, `internal/eval_harness/models.go:16`.
- **Project principles**: CLAUDE.md #2 (NO SILENT FALLBACKS — names pricing/model configs),
  #3 (SYSTEMIC FIXES — audit before patching).
- **Related tooling**: `model-manager` skill (test/validate/add models) — this feature gives that
  skill a machine check to run.
- **Axiom reference**: [Design Axioms](/docs/references/axioms)

## Future Work

- **`supersedes` links**: explicit predecessor pointer per model for exact lineage and auto-derived
  "current per family" lists.
- **Vendor-API date/pricing fetch**: optionally pull release dates and pricing from provider
  metadata to suggest (not silently set) values for the editor to confirm.
- **`ailang eval-models doctor`**: extend the validator into a richer roster health report (stale
  models not run in N releases, families with no "current" pick).
