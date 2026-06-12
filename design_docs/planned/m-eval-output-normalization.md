# M-EVAL-OUTPUT-NORMALIZE: Boolean-case + numeric-format grading parity (sibling of M-EVAL-JSON-COMPARE)

**Status**: Planned
**Target**: v0.26.0 (rule) + re-grade of the v0.25.0 baseline
**Priority**: P1 (High — a grader artifact depresses every model's Python score and distorts the public AILANG-vs-Python story)
**Estimated**: 0.5–1 day (mirror the M-EVAL-JSON-COMPARE pattern + adversarial verification)
**Dependencies**: Extends `CompareOutput` in `internal/eval_harness/runner.go` (the same function [M-EVAL-JSON-COMPARE](../implemented/v0_24_0/m-eval-network-mock-fixture.md) made JSON-aware). Re-grade mechanism already proven (commit `926c1e56` re-graded v0.24.1 after the JSON-compare fix).

> **📊 EMPIRICALLY VERIFIED (v0.25.0 standard baseline, 2026-06-12):** Of **83** Python
> stdout-only failures across the 11-model `extended_suite`, **32 (39%)** are pure
> `True`/`False` vs expected `true`/`false` casing — Python prints native `True`/`False`,
> the benchmarks expect lowercase `true`/`false` (AILANG/JSON convention). It hits **every
> model** (1–4 false-failures each), concentrated in 4 contract benchmarks
> (`contract_bst_validate` 11/11 models fail, `contract_matrix_determinant` 9,
> `contract_roman_numeral` 8, `contract_rle_roundtrip` 4). Correcting only bool-casing moves
> the **field Python pass-rate 77.4% → 85.3%** and `claude-fable-5` **78.4% → 89.2%**. The
> apparent "AILANG beats Python" inversion is largely this artifact: corrected, the field gap
> is AILANG 82.6% vs Python 85.3% (−2.7pp, the expected direction). A second tier of artifacts
> (float `7.50` vs `7.5`; Python container repr `{1,2,3}` vs `[1,2,3]`) accounts for several
> more — e.g. `contract_sorted_merge` Python 0% and `json_transform` Python 27% are almost
> entirely formatting, not capability.

---

## Problem statement

`CompareOutput` grades a run by comparing program stdout to a single canonical
`expected_stdout`. [M-EVAL-JSON-COMPARE](../implemented/v0_24_0/) already fixed one class of
false-negative — JSON whitespace/key-order/int-vs-float — when **both sides are valid JSON**
(`ast_patch_roundtrip` 38% → 95%). But benchmark expected outputs are written in AILANG's
surface conventions (lowercase `true`/`false`, list-style `[1,2,3]`, minimal float formatting),
and a correct Python solution emits the language's **native** reprs (`True`/`False`,
`{1, 2, 3}`, `7.50`). The grader scores these idiomatically-correct programs as wrong.

This is not a Python-only nuisance — it is a **measurement bug**. It (a) depresses every model's
Python pass-rate by a few points, (b) inflates AILANG's relative standing on the public Model
Leaderboard, and (c) makes per-benchmark "AILANG wins" untrustworthy (the win may just be the
Python side mis-graded). A capability eval that penalizes correct output formatting teaches the
wrong lesson — directly against Axiom A7 (Machines First).

## Scope — what is and is NOT a grader bug

This distinction is the heart of the doc; over-normalizing would *reward* models for ignoring
the output spec, which is as bad as the current false negatives.

**IN (grader bug → normalize, safe-by-construction):**
1. **Boolean case** — `True`↔`true`, `False`↔`false` as standalone output tokens. The dominant
   class (32/83 failures). Unambiguous: no benchmark distinguishes a capital-B boolean as a
   *correct* answer distinct from lowercase.
2. **Numeric formatting** — trailing-zero / integer-float equivalence (`7.50`==`7.5`,
   `16.0`==`16`) when the surrounding tokens are otherwise identical. Mirrors JSON-compare's
   existing int-vs-float normalization, extended to bare (non-JSON) numeric output.

**DISCUSS (borderline — decide with data):**
3. **Container repr** — Python `{1, 2, 3}` (set) / `(1, 2)` (tuple) vs expected `[1,2,3]`.
   A set vs a list can be a *genuine* wrong data structure, OR pure repr. Proposal: normalize
   only list/tuple bracket+spacing (`[1, 2, 3]`==`[1,2,3]`), and **do not** silently equate a
   set `{}` with a list `[]` — instead surface it as a benchmark-prompt fix (ask for a list
   explicitly) so the signal stays honest.

**OUT (genuine model behavior → keep failing):**
4. **Extra / labelled output** — e.g. Fable printing `treeToList(tree): [1,2,3]` or prepending
   `Numbers: …\nEvens: …` before the required line. The values are correct but the model
   ignored "print exactly X". This is a real instruction-following miss and MUST keep failing;
   normalizing it away would reward verbosity and destroy the metric.

## Design

Extend `CompareOutput(expected, actual)` after the existing exact-match fast path and the
JSON-aware branch, with a final **normalized-equality** branch:

```
exact match            → pass   (unchanged, fast path)
both valid JSON & equal → pass   (M-EVAL-JSON-COMPARE, unchanged)
normalize(expected) == normalize(actual) → pass   (NEW)
otherwise              → fail
```

where `normalize` applies, line-by-line: standalone-boolean case-fold, numeric trailing-zero
canonicalization, and list/tuple bracket-spacing canonicalization. **Safe by construction**,
exactly like JSON-compare: the branch can only ever turn a *fail into a pass* when the
normalized forms are identical; it can never turn a pass into a fail, and a genuinely different
output (wrong value, missing/extra line, wrong structure) still fails because normalization is
content-preserving. Guard each transform so it only fires when it produces an exact match — never
a fuzzy/partial one.

### Re-grade rollout (no model re-runs)

Result JSONs already store `stdout` + `expected_stdout`, so the fix is applied to existing
baselines by **re-running the grader over stored outputs and flipping `stdout_ok`/pass** — the
same path used to regenerate v0.24.1 after JSON-compare (commit `926c1e56`). Re-grade the
v0.25.0 baseline immediately after the rule lands, then regenerate the dashboard
(history-preserving). No API spend.

## Non-goals

- No change to AILANG grading (it already emits lowercase `true`/`false`; normalization is a
  no-op there — verified: applying it to the AILANG arm changes zero verdicts).
- Not a fuzzy/semantic matcher — strict normalized equality only.
- Does not touch the verbosity class (§Scope OUT) — that stays a real failure.

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1 Determinism | +1 | Removes formatting-luck from the verdict; same capability → same score across languages. |
| A2 Replayability | +1 | Re-grade is a pure function over stored outputs; reproducible offline. |
| A7 Machines First | +2 | A capability eval that penalizes correct output formatting trains the wrong signal; this is the core fix. |
| A9 Cost Visibility | +1 | Re-grade reclaims accuracy with zero API spend (no re-runs). |
| A11 Structured Failure | +1 | Separates "wrong answer" from "right answer, native formatting" — they were conflated. |
| A12 System Boundary | +1 | The grader owns output-convention normalization instead of leaking it into every benchmark's expected file. |

**Hard violation check:** none. Risk is over-normalization (A7 inverted); the §Scope OUT
boundary + exact-normalized-match gate contain it.

## Acceptance

- [ ] `CompareOutput` bool-case + numeric branch added behind the existing exact/JSON fast paths.
- [ ] Unit tests incl. **safety cases**: `True`→pass vs `true`; `7.50`→pass vs `7.5`;
      and NEGATIVES that must still fail — wrong value (`5` vs `6`), extra line (verbosity),
      missing line, set-vs-list when treated as structural.
- [ ] **Adversarial verification** (mirror JSON-compare): re-grade the full v0.25.0 standard
      set and hand-audit a sample — zero passes created for genuinely-wrong outputs.
- [ ] Re-grade v0.25.0 baseline: field Python ≈ 85%+, AILANG unchanged, dashboard regenerated.
- [ ] Document the leaderboard delta in the v0.26.0 changelog (M-EVAL).

## Appendix — residual genuine gaps (after bool-case correction, v0.25.0 field)

Real AILANG weaknesses to target (Python ahead, corrected): `decision_block_capture`
(AILANG **0%** vs Py 82% — total failure, investigate first), `explicit_dataflow_ssa` (55/100),
`csv_to_json_converter` (73/100), `run_length_encode` (55/82). Real AILANG strengths (survive
correction, not formatting-inflated): `effect_composition`, `higher_order_functions`,
`expression_evaluator`, `polymorphic_ord_defaulting` (all AILANG 100%, Py ≤91%) — consistent
with AILANG's effect system + type-directed defaulting. NB: `contract_sorted_merge`,
`json_transform`, `list_comprehension` "AILANG wins" are partly Python-side artifacts (§Scope
2/3/4) and should be re-checked after this fix lands.
