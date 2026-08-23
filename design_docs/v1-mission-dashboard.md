# Mission Dashboard — V1

> Snapshot only; history in `v1-mission.md` + `v1-mission-log.md`. Written: **iter 258, 2026-08-23**.

## Where we are
- **v0.33.1**. **BAR-FIRST** (`D-28`) until all 5 clauses close. **dev CI GREEN** at `9417c5ff7`
  (16 checks, zero not-green; control on parent = 20).
- **`D-29` RESOLVED by Mark this morning.** Ledger 31 rows, **TWO OPEN: `D-30`, `D-31`**.

## In flight / next
1. **`m-verify-bounded-unrolling-false-counterexample`** *(new, P0)* — the verifier reports its own
   incompleteness as a `counterexample`, so correct recursive code is graded a verification FAILURE.
2. **`m-cohort-manifest-build-provenance`** — re-scoped to a split at iter-257; `[NEXT]`, in-loop.
3. **`m-module-cache-identity-not-compiler-bytes`** — the consumer split out of (2).
4. **`m-benchmark-ensures-coverage`** *(new)* — Mark's second `D-29` clause; blocked on (1) for 4 of 5.
5. **`m-contract-verification-coverage`** — parked on `D-30` only (predicate re-read, unchanged).

## New this iteration
- **`D-29` ruled BOTH**: publish a **strict** arm (`$0.7778`, denom 3) *beside* an **effective** arm
  (`$0.2121`, denom 11). Nothing is restated, the 2026-07-27 ratification stands, and both arms need
  the `not_applicable` split — so the publishing milestone lands inside the `D-30`-parked doc.
- **Scoping Mark's "ensures where it makes sense" uncovered a P0.** Same file, same clause, only
  recursion differs: `encFlat` **verified**, `encRec` **counterexample** — on a postcondition that
  holds at runtime for every input tried. **Structural, not depth tuning**: the witness grows with
  the bound (depth 2 → `"AAA"`, 4 → `"ABDCA"`, 8 → `"ABCDEFGHAB"`), so no finite `k` ever verifies.
- **`contract_sorted_merge.yml:23-24` ships that exact shape on `sLength` today** — live for the
  frozen cohort. `isVerifiedSuccess` fails on `VerifyCounterex > 0`, so this corrupts **both** arms
  `D-29` just ratified, in a direction no exemption repairs.
- **Enumerated, not guessed**: 7 of 92 benchmark YAMLs carry a `contract_spec`; functions partition
  16 `ensures` / 10 no-clauses / **5 `requires`-only** — exactly `D-29`'s five, of which only
  `minor3` is non-recursive. (Charter self-correction: that row wrongly said `minor3` had no clauses.)

## Routing · Cost
Controller opus only. **No designer/planner/executor/evaluator spawned** — the deliverable is a
ruling recorded and a scope measured. Metered **$0.00** of $5. Rotation pointer not advanced.

## PARKED ON MARK — two one-word calls
- **`D-30`** — enforce the harness↔`ai-check` version coupling how? **(a) schema-version JSON** ·
  **(b) bind child to `os.Executable()`** · **(c) accept + spot-check**. Blocks the KPI split.
- **`D-31`** — the designer rotation has ONE usable authoring lane (codex *is* a quorum reviewer;
  gemini is read-only). **Widen it**, **split authoring from review**, or **accept the Fable cost**?
