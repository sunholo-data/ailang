# M-EFFECT-LATENT-FUNCTION-VALUES execution validation

Date: 2026-10-08. Target: v0.53.0. Refs #1326, Refs #573; lands before #616.

## Provenance and scope

Fresh baseline built from repository commit `48f4b5ef929532cc6bf89d32034435ec616f80e7`.
Fixed binary built from the reviewed implementation working tree over that commit;
all probes used `AILANG_NO_CACHE=1`. The implementation commit is recorded in git.
No installed version stamp was used as evidence. Binary SHA-256 values:

- base: `108056e4ddb8c3cd5bfb2b0c0534bd9632559af6d0ed6bc441933a83002d22f2`
- fixed: `77d693cff183961ced2d4c9daca296d8be4e6ce04a3ffb4ebec565afbf9c87f1`

Implements Phases 0–2 and 4, with L1-open. Phase 3 closed annotation upper bounds
remain deferred. No opt-out, runtime change or std signature change. Tail union,
difference and tail extraction semantics are unchanged.

The independent evaluator found storage regressions through conditional, field and
annotated-alias callees in round 1. These were fixed by preserving source concrete
callback contracts on `TFunc2.ConcreteEffectContract`, rather than relying on tail
spelling or scheme-only metadata. Substitution and JSON/cache round trips preserve
this flag; imported function aliases expose their latent row via `EffectValueType`.
Per-App `LatentParamMask` remains explicit and fails loudly if missing for a typed
application. #616 extends this publication next, rather than adding another authority.

## Acceptance evidence

- AC1–AC4: local HOF, wrong FS row, `sortBy`, `zipWith`, `flatMap`, recursive `anyR`,
  local forwarding and std `flatMap` forwarding reject undeclared IO at the caller.
  Current-base `any`, `findIndex`, `foldr` already reject IO callbacks via closed
  iterative builtin contracts (#1518); both pure/IO-declared caller rejection
  results are deliberately preserved. No stdlib signatures changed.
- AC5–AC6: named/inline record fields, callback parameters, function returns, let
  annotations, function aliases (local/imported), ADT payloads, record updates
  reject with `Missing effects: IO` at the offending caller. Annotation tests retain
  written tails, budgets and parameterised effect modes.
- AC7–AC8: pure record construction, direct and aliased storage-only contracts,
  computed/record/annotated-alias storage callees, effect ceilings, genuine-effect
  import rejection and inferred combinator controls remain accepted/rejected as
  specified. Runnable mapE and DOM examples retain their baseline check results.
- AC9: parameter/let/pattern shadowing accepts; direct IO call still rejects.
- AC10: DOM replay checks without effect-label payload collisions. Unit-payload
  tests and record-row negative controls isolate the canonicalizer and unifier.
- AC11: rebuilt binary tree-walk and bytecode `run` reject both HOF and record-field
  repros before execution. No `EFFECT PERFORMED` marker appears. The runnable
  example checks and prints `HOF: IO declared` then `field: IO declared` with IO.
- AC12: fresh existing corpus is **440 files: 356 pass / 84 fail** on both base and
  final fixed binary, with **zero status flips**. New example adds one passing file.
  All **49 std import probes pass** on base and fixed, with zero status flips.
  No example/std migration is needed. New rejection paths are tested independently
  with fresh modules. AC13 does not apply (Phase 3 deferred).

## Mutation evidence

Each component was disabled temporarily, its targeted tests failed, then original
source was restored before the final green package/core checks:

| Mutation | Red evidence | Controls |
|---|---|---|
| L2 removed | local/wrong-row HOF, sortBy/zipWith/flatMap, all three forwarding shapes | annotation and storage arms unchanged |
| L1 removed | fields, callback parameter, aliases, ADT payload | local/wrong-row HOF remain rejected |
| Open-parameter gate removed | direct, aliased, computed, record and annotated-alias storage | effect-performing arms remain rejected |
| D4 removed | shadowed parameter and let | direct call remains rejected |
| L0 canonicalizer removed | Unit payload assertion | record payload remains structurally checked |
| L0 unifier protection removed | effect-row payload mismatch | record payload mismatch remains rejected |
| Both L0 components removed | DOM fails `Cog vs ()` effect-label payload unification | restored DOM checks green |

## Gates and coverage

| Gate | Result |
|---|---|
| Focused types/elaborate/pipeline/iface package suite | PASS |
| `make test-core` (CGO enabled) | PASS |
| `make check-file-sizes` | PASS (all files <=800 lines) |
| `make check-boundaries` | PASS |
| `make fmt-check` | PASS |
| `make lint` | PASS (0 issues) |
| `make verify-examples` | PASS (rebuilt binary reused) |
| Independent sprint-evaluator round 2 | PASS 96/100; independent report recorded separately |

Changed-package statement coverage: types **53.1%**, elaborate **64.2%**, pipeline
**75.8%**, iface **42.4%** (types/traverse **92.1%**). Aggregate selected-package
coverage **61.8%**. These are current measurements, not an invented repository-wide
baseline.

The environment initially lacked PATH entries for Go, jq/make and a C compiler.
Tools were provisioned under /tmp without repository scripts. A no-CGO core run
failed SQLite-backed tests; the required CGO-enabled run passed using a local C
compiler. An uncapped linter and a concurrent redundant binary build were killed
by the environment memory limit; final stable checks use bounded concurrency and
memory. The constrained linter initially reached its default timeout with zero
issues; its stable retry with a longer timeout and warmed caches passed. The example retry reuses the already-rebuilt exact implementation binary.

Full `make test` is intentionally deferred to implementation PR CI per the approved
plan (RAM-backed /tmp SIGBUS history). No push or merge was performed. Use `Closes
#1326` and `Closes #573` only in the implementation PR; do not close #616 here.

The deferred V13 width case was freshly checked (exit 0) and documented in both
limitations pages with the existing issue references. The multi-phase design and
its plan stay together in the planned area pending Phase 3, using the evaluator
skill's explicit exception for multi-phase designs; all approved current-sprint
milestones are complete.

Per-file base/fixed exit codes, binary hashes, mutation failures and gate exit codes
are banked in `.ailang/state/sprints/validation_M-EFFECT-LATENT-FUNCTION-VALUES.json`.
