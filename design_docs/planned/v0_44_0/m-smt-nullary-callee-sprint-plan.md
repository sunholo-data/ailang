# M-SMT-NULLARY-CALLEE — sprint plan (no design doc)

**Status**: Planned → executed in the same PR
**Report**: Daneel, 2026-09-26 — `ailang verify` skips a contract whose body calls a zero-arg pure
function (`Function "bound" calls "cap" whose signature uses an unencodable type "()"`).
**Scope**: `internal/smt` only. Encoder change; the contract language and its runtime semantics are
unchanged, so no design doc (design-doc-creator criteria: local to the SMT encoder).

## Root cause

`func cap() -> int` desugars to `cap(_: ())` and every call site passes a unit literal. Three places
in the SMT path did not know this, each blocking the next:

1. **Callee-sort gate** (`FirstUnencodableCalleeType`) checked every parameter type, so `_: ()`
   rejected the caller with `UNENCODABLE_TYPE "()"`.
2. **Callee define-fun signature** (`allSurfaceParams` in `verify.go`) kept the unit param, and
   `buildDefineFun` silently mapped the unencodable `()` sort to `Int`: `(define-fun cap ((_ Int)) ...)`.
3. **Call-site encoding** (`encodeUserFunctionCall`) encoded the unit argument and failed with
   `unit literals cannot be encoded in SMT-LIB`.

`UnwrapLambdaParams` already dropped the unit param for the *contracted* function itself
(`isUnitParam`); the callee paths were the second implementation of the same concept.

## Audit (principle 3)

| Shape | Before | After |
|---|---|---|
| Nullary `-> int` called in body | skipped (gate) | verified / counterexample |
| Nullary `-> bool`, `-> string` | skipped | verified |
| Chained nullary (`doubled() = cap() * 2`) | skipped | verified — callee bodies now encode calls to already-defined callees as define-fun applications |
| Nullary called in `requires`/`ensures` | skipped (leak guard) | verified — contract predicates are roots for callee resolution and for the sort gate |
| Cross-module nullary | skipped | verified |
| Recursive nullary (`spin() = spin()`) | skipped | skipped (cycle; sound) |
| Unit-returning helper (`-> ()`) | skipped | skipped — `()` return is genuinely unencodable |

## Milestones

- **M1** Gate + signatures: skip unit params in the gate and via one `SurfaceFunctionParams` helper
  for both current-module and imported surface params (~40 LOC).
- **M2** Call sites: drop unit args, reference a nullary define-fun by bare name; register callees
  as they are defined so chained callee bodies resolve (~40 LOC).
- **M3** Contract roots: resolve callees and run the sort gate over requires/ensures too (~50 LOC).
- **M4** Tests: unit tests in `internal/smt`, e2e `ailang verify` on
  `examples/runnable/contracts/nullary_constant.ail` (verified + a counterexample that only exists if
  the constant is really encoded + chained + in-contract). Mutation-test each milestone.

## Acceptance

- Exact Daneel repro prints `✓ VERIFIED bound`.
- A contract false *because of* the constant's value reports a counterexample with `cap: Int = 1000`.
- No status change on any contract-bearing `.ail` in the repo except the updated
  `contract_predicate_callee` e2e case (now verified rather than skipped).
