# M-SMT-CALLEE-SORT-GATE: Reject Unencodable Callee Signature Sorts (No Z3 Crash)

**Status**: Planned
**Target**: v0.30.0
**Priority**: P1 (Medium) — correctness/robustness of the verifier's "graceful skip" contract
**Estimated**: ~0.5 day (4 hours)
**Dependencies**: Builds on M-SMT-FRAGMENT-EXPANSION (v0.8.0, cross-function inlining)

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No runtime behavior change; verifier-only |
| A2: Replayability | 0 | No impact on traces |
| A3: Effect Legibility | 0 | No effect changes |
| A4: Explicit Authority | 0 | No authority changes |
| A5: Bounded Verification | +1 | Makes the fragment boundary honest — an unencodable callee signature is now a *named, bounded* skip instead of an unbounded Z3 crash |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | Turns an opaque Z3 solver error into a structured `UNENCODABLE_TYPE` rejection an agent can read and act on |
| A8: Minimal Syntax | 0 | No new syntax |
| A9: Cost Visibility | 0 | No cost model change |
| A10: Composability | +1 | The skip-with-reason path already exists; this makes it compose over cross-function call graphs |
| A11: Structured Failure | +1 | Replaces a raw solver crash with a typed `SMTRejectionReason` |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +4** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No nondeterminism introduced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Improves machine-readable output (structured rejection vs solver crash)

## Problem Statement

`ailang verify` promises a **graceful skip with a named reason** whenever a function falls
outside the SMT-encodable fragment. That promise holds for direct constructs (arrays, unmapped
builtins, unit types → `UNENCODABLE_TYPE`). It **breaks for cross-function calls**: when a
contracted function calls a callee whose *signature* contains an unencodable sort — most
commonly an ADT-over-primitive like `Option[float]` — the verifier neither inlines nor cleanly
rejects it. Instead the callee leaks into the SMT script as an undeclared sort / uninterpreted
symbol, and **Z3 hard-errors (exit status 1)**.

**Current State (reproduced live, v0.29.2):**

```ailang
module xfn
import std/option (Option, Some, None)

export func convertTo(x: float, target: string) -> Option[float] ! {}
{ Some(x * 2.0) }

export func gradeNumeric(x: float) -> float ! {}
requires { x >= 0.0 }
ensures { result >= 0.0 }
{
  match convertTo(x, "F") {
    Some(v) => v,
    None => 0.0
  }
}
```

`ailang verify` output:

```
! ERROR gradeNumeric
  Z3 error (exit exit status 1): (error "... Invalid function definition: unknown sort 'Option'")
  (error "... unknown constant convertTo (Real String) ")
  (error "... unknown constant result")
```

**Root cause.** The fragment/encodability gate only inspects **`$builtin`/stdlib call
*names*** — it never inspects the **signature sorts** of a user/stdlib cross-function callee.
Specifically:

- `firstUnencodableBuiltin` (`internal/smt/encodable.go:569`) and `hasUnencodableTypes`
  (`internal/smt/encodable.go:432`) walk the body for `$builtin`/`std/string`/`std/list` names
  with no SMT mapping. The `*core.VarGlobal` case (`encodable.go:477-491`) returns `false`
  (i.e. "encodable") for any other user/ADT reference — so an `Option`-returning callee slips
  through.
- In `ResolveCallees` (`internal/smt/callee_resolver.go:43`), the callee either (a) is judged
  encodable by `IsSMTEncodableForCallee` (`callee_resolver.go:456`) and gets a
  `(define-fun convertTo (...) Option ...)` with the **undeclared `Option` sort**, or (b) is
  skipped (`continue`, `callee_resolver.go:128`) and the call site in `encodeApp`
  (`internal/smt/codegen_apps.go:107-108`) falls through to `encodeConstructorApp`, emitting a
  **raw `(convertTo …)` symbol**.
- The last line of defense, `validateDeclarations` (`internal/smt/codegen.go:642`), **only scans
  `(declare-datatype …)` lines** (early-continue at `codegen.go:662`) and never validates
  `(define-fun …)` bodies — so the malformed script reaches Z3.

**Impact:**
- **Who:** any agent (or human) using contracts on realistic code. Grader/scoring logic that
  routes through helper functions returning `Option`/`Result`/enums over `float`/`string` is a
  common, natural shape — this is exactly the reported `gradeNumeric` case.
- **How significant:** it violates a core verifier invariant. `verify` is sold as "prove it or
  skip it with a reason." A hard Z3 crash is neither — it looks like a verifier bug, erodes
  trust in the feature, and gives the agent an unactionable solver-internals error instead of a
  fragment boundary it can code around.

## Goals

**Primary Goal:** No contracted function should ever produce a raw Z3 solver crash because a
*callee's signature* is unencodable — it must skip with a structured `UNENCODABLE_TYPE` reason
naming the offending callee and sort.

**Success Metrics:**
- The `gradeNumeric`/`convertTo` repro (and its `Result`/enum variants) reports
  `Status: skipped` with a clear reason, **not** `Status: error` / Z3 exit 1.
- Zero `(define-fun …)` referencing an undeclared sort ever reaches the solver (defense-in-depth
  validation catches any future leak).
- No regression: all functions in `examples/runnable/contracts/*.ail` that verify today still
  verify (`cross_function.ail` int-chain, `temperature.ail`, `record_verify.ail`, etc.).

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Gate-and-skip vs. actually support `Option[float]` | Determines scope: a robustness fix (hours) vs. a parametric-ADT monomorphization feature (days) | human | design | med |
| Where to gate: caller-level (`IsSMTEncodable`) vs. callee-resolver (`ResolveCallees`) | Affects which function is reported skipped and the reason string granularity | agent | design | low |
| Add `validateDeclarations` scan of `define-fun` sorts as defense-in-depth | Prevents *any* future undeclared-sort leak, not just this class | agent | design | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] **Scope = gate-and-skip (option a), NOT parametric-ADT support (option b).** Supporting
  `Option[float]` requires registering stdlib ADT variants into `adtTypes`, monomorphizing the
  type argument into the field sort, and encoding `Some`/`None`/`match` — a separate feature
  (see Future Work). This doc closes the *crash*; it does not extend the fragment.
- [x] **Gate at the caller level** so the *contracted* function (`gradeNumeric`) is the unit
  reported skipped, with a reason naming the callee + sort. (Callee-resolver-level skip is the
  fallback mechanism, already present; the caller gate is what turns a leak into a named skip.)
- [x] **Add the `define-fun` sort-validation as defense-in-depth** — cheap and closes the whole
  class.

## Solution Design

### Overview

Two layers, both small:

1. **Primary — caller-level sort gate.** Extend the encodability check so that, when walking a
   contracted function's body, any cross-function callee whose **signature (return type or any
   parameter type) maps to a non-primitive, undeclared sort** causes the *caller* to be rejected
   with `UNENCODABLE_TYPE`, naming the callee and the offending sort. This routes the function to
   the existing skip-with-reason path (`verify.go:294-307`) before `EncodeFunction` is ever
   called.

2. **Defense-in-depth — declaration validation.** Extend `validateDeclarations`
   (`codegen.go:642`) to also scan emitted `(define-fun …)` signatures for sorts that are neither
   primitive (`Int`/`Bool`/`String`/`Real`) nor declared datatypes/records/`(Seq …)`. If found,
   return `ErrUnresolvableTypes` (which `verify.go:358-371` already treats as a graceful skip),
   guaranteeing no undeclared sort can reach Z3 even if a future code path forgets the gate.

The "primitive + declared" sort predicate is shared logic: primitives come from
`primitiveSorts` (`codegen.go:634`), declared datatypes/records are tracked during
`EncodeFunction` (`codegen.go:245-336`).

### Architecture

**Components:**

1. **`calleeSignatureSort` collection.** The verify driver already computes
   `allSurfaceReturnSorts[funcName]` and `surfaceParams` via `astTypeToSMTSort`
   (`verify.go:170-226`). We reuse these: for each cross-function callee referenced in the body,
   look up its return sort + param sorts.

2. **`firstUnencodableCalleeSort(body, sigLookup) (calleeName, sort string)`** — a new walker in
   `internal/smt/encodable.go`, structurally mirroring `firstUnencodableBuiltin`. For each
   `*core.App` whose head is a user `*core.VarGlobal` (non-`$builtin`, non-stdlib-mapped), it
   looks up the callee's signature sorts and returns the first that is **not** encodable
   (not a primitive, not a declared/declarable ADT/record/Seq). Returns `("","")` if all clean.

3. **Wire into `IsSMTEncodable`** (`encodable.go:43`): add check 7 (after the existing builtin/
   type checks). On a hit, emit
   `SMTRejectionReason{Code: RejectUnencodable, Message: "callee '<name>' has unencodable sort '<sort>' in its signature", Hint: "..."}`.
   Because `IsSMTEncodable` is called from both `verify.go:281` and `ai_check.go:273`, one change
   covers both drivers.

4. **`validateDeclarations` extension** (`codegen.go:642`): parse `(define-fun <name> (<params>) <ret> …)`
   headers, extract param + return sort tokens, and assert each is primitive-or-declared.

**Why a signature lookup and not "declared sorts at gate time":** the gate runs *before*
`EncodeFunction`, so the set of declared datatypes isn't built yet. The predicate at gate time is
therefore "is this sort a **primitive, or a user ADT/record that WOULD be declared** (i.e. it
appears in `adtTypes`/record set)". A stdlib parametric ADT like `Option` is neither → reject.
This is exactly why `Option` fails but a user's own `type Grade = A | B | C` return type passes
(it gets declared).

### Implementation Plan

**Phase 1: Caller-level sort gate** (~2 hours)
- [ ] Add `firstUnencodableCalleeSort` walker in `internal/smt/encodable.go` (mirror
  `firstUnencodableBuiltin` structure; take a signature-sort lookup + a "declarable sort"
  predicate).
- [ ] Thread the callee signature-sort lookup + declarable-ADT set into `IsSMTEncodable`
  (extend its signature or a small context struct; update both call sites `verify.go:281`,
  `ai_check.go:273`).
- [ ] Emit `RejectUnencodable` reason naming callee + sort.

**Phase 2: Defense-in-depth declaration validation** (~1 hour)
- [ ] Extend `validateDeclarations` (`codegen.go:642`) to scan `(define-fun …)` signature sorts
  against primitive-or-declared; return `ErrUnresolvableTypes` on violation.

**Phase 3: Tests + example + docs** (~1 hour)
- [ ] Regression example `examples/runnable/contracts/unencodable_callee_skip.ail` (the
  `gradeNumeric`/`convertTo` repro).
- [ ] Go tests: `internal/smt` unit test for `firstUnencodableCalleeSort` +
  `validateDeclarations`; `cmd/ailang` (or existing verify test harness) asserting
  `Status: skipped` with `UNENCODABLE_TYPE` for the repro, and asserting `cross_function.ail`
  still verifies.
- [ ] CHANGELOG entry; note in verifier limitations doc.

### Files to Modify/Create

**New files:**
- `examples/runnable/contracts/unencodable_callee_skip.ail` — repro that must skip cleanly, ~20 LOC

**Modified files:**
- `internal/smt/encodable.go` — add `firstUnencodableCalleeSort` + wire into `IsSMTEncodable`, ~60 LOC
- `internal/smt/codegen.go` — extend `validateDeclarations` to scan `define-fun` sorts, ~30 LOC
- `cmd/ailang/verify.go` — pass callee signature-sort lookup + declarable set into `IsSMTEncodable`, ~15 LOC
- `cmd/ailang/ai_check.go` — mirror the same wiring, ~15 LOC
- `internal/smt/encodable_test.go` (+ codegen test file) — unit tests, ~80 LOC
- `CHANGELOG.md` — entry

## Examples

### Example 1: Cross-function callee returning `Option[float]`

**Before:**
```
$ ailang verify grader.ail
! ERROR gradeNumeric
  Z3 error (exit exit status 1): (error "... unknown sort 'Option'")
  (error "... unknown constant convertTo (Real String) ")
  1 functions: 1 errors
```

**After:**
```
$ ailang verify grader.ail
  ⊘ SKIP gradeNumeric
    UNENCODABLE_TYPE: callee 'convertTo' has unencodable sort 'Option' in its signature
    Hint: cross-function verification requires callee signatures over int/float/bool/string,
          records, or user enum ADTs. Option/Result over primitives are not yet encodable.
  0 errors, 1 skipped
```

### Example 2: No regression — user enum callee still verifies

`cross_function.ail`'s `shippingCost` calls `baseCost`/`priorityMultiplier` returning `int`;
these map to `Int` (primitive) and continue to inline + verify. A callee returning a user
`type Region = DOMESTIC | INTERNATIONAL` is *declared* in `adtTypes` and therefore passes the
gate.

## Success Criteria

- [x] `unencodable_callee_skip.ail` reports `Status: skipped` + `UNENCODABLE_TYPE` naming the callee/type (acceptance test)
- [x] No Z3 exit-status-1 / "unknown sort" / "unknown constant" on the repro (acceptance test)
- [x] `cross_function.ail`, `temperature.ail`, `record_verify.ail` still verify unchanged (12 examples byte-identical pre/post + regression test)
- [x] `validateDeclarations` rejects a hand-crafted `define-fun` over an undeclared sort (unit test)
- [x] Tests passing (`cmd/ailang`, `internal/smt`); `go build ./...` clean; `make verify-examples` green
- [x] Documentation updated (CHANGELOG)
- [x] Example added
- [x] Mirrored to `ailang ai-check` twin

**Implementation note:** the encodability gate is AST-based (`firstUnencodableCalleeType` /
`astTypeEncodable` in `cmd/ailang/verify.go`), not sort-string based as originally sketched.
Discovered during implementation: an imported parametric ADT like `Option` gets registered in
`adtTypes` (so a "declarable sort name" check wrongly passes it), and `astTypeToSMTSort` flattens
`Option[float]` to `"Option"`, dropping the argument. The AST distinguishes a monomorphic enum
(`SimpleType`) from a parametric application (`TypeApp` with args) — the true signal.

## Testing Strategy

**Unit tests:**
- `firstUnencodableCalleeSort`: callee returning `Option[float]` → hit; callee returning `int`/
  user-enum → no hit; callee with unencodable *param* sort → hit.
- `validateDeclarations`: `(define-fun f ((x Real)) Option …)` → error; `(define-fun f ((x Real)) Real …)` → ok.

**Integration tests:**
- `ailang verify` on the repro → skipped-with-reason (assert on JSON output `--json`).
- `ailang verify` on `examples/runnable/contracts/cross_function.ail` → still `verified`.

**Manual testing:**
- Run `ailang verify` on the original reporter shape (`Result`-returning callee, string params)
  to confirm the gate generalizes beyond `Option`.

## Deferred Decisions

- Exact wording of the `Hint:` string — agent may choose, must name the encodable sort set.
- Whether to fold the param-sort and return-sort checks into one walk or two — agent's choice.

## Non-Goals

- **Actually verifying `Option[float]`/`Result`/enum-over-primitive callees** — that is a
  fragment *expansion* (parametric-ADT monomorphization), tracked as Future Work, not this doc.
- **Changing runtime contract checking** — runtime `requires`/`ensures` already work for these
  functions; this is static-proof-path only.
- **IEEE-754 float semantics** — orthogonal; `float` remains `Real` in the fragment.

## Timeline

**Day 1** (4 hours):
- Phase 1 (caller-level gate) + Phase 2 (define-fun validation)
- Phase 3 (tests, example, docs)
- Release in v0.30.0

**Total: ~4 hours**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Gate is too aggressive → rejects callees that DO encode (e.g. user records/enums) | Med | The declarable-sort predicate explicitly admits primitives + `adtTypes` + record sorts + `(Seq …)`; regression tests on `cross_function.ail`/`record_verify.ail` guard this |
| Signature-sort lookup misses a callee (indirect/transitive) | Med | Defense-in-depth `validateDeclarations` catches any leak that slips the gate → `ErrUnresolvableTypes` → graceful skip, never a Z3 crash |
| Changing `IsSMTEncodable` signature ripples to other callers | Low | Only two call sites (`verify.go`, `ai_check.go`); grep-confirmed |

## Related Documents

**Implemented (informs design):**
- [design_docs/implemented/v0_8_0/m-smt-fragment-expansion.md](design_docs/implemented/v0_8_0/m-smt-fragment-expansion.md) — introduced cross-function inlining (Phase A); this doc closes a gap it left in the encodability gate
- [design_docs/implemented/v0_8_0/m-smt-record-discovery.md](design_docs/implemented/v0_8_0/m-smt-record-discovery.md) — record datatype declaration machinery (the "declarable sort" precedent)

**Planned:**
- None overlapping (neural search top match 0.36, well below duplicate threshold)

## References

- [Design Axioms](/docs/references/axioms)
- `internal/smt/encodable.go` — fragment gate (the change site)
- `internal/smt/callee_resolver.go` — cross-function inlining
- `internal/smt/codegen.go:642` — `validateDeclarations`
- Reproduced live on v0.29.2 (see Problem Statement transcript)

## Follow-up: M3 leak-site guard (round 2)

A reporter follow-up showed the M1 signature gate was necessary but **not sufficient**. Two more
shapes still hard-ERRORed with `unknown constant`:
- a callee returning a **record** (`canon(s) -> Rec`) — the callee is *dropped* by
  `ResolveCallees` (record body not inlinable / return sort not declared), so no define-fun is
  emitted and the call site leaks a raw symbol. M1's signature check treats `RecordType` as
  encodable, so it doesn't fire; M2 has no define-fun to inspect.
- a user function called inside a **contract predicate** (`ensures { legal(result) }`) —
  `collectCalleeCalls` only walks the body, so `legal` is never a resolution candidate; when the
  ensures clause is encoded it leaks.

**Root cause (systemic):** *any* referenced user function (body OR contract) that isn't resolved
into a define-fun/contract-substitution falls through `encodeApp` → `encodeConstructorApp` → raw
symbol → Z3 `unknown constant`. The right fix is at the **leak site**, not a per-shape gate:
`EncodeFunction` now records every user function name (`activeUserFunctions`), and `encodeApp`
returns `ErrUnresolvableTypes` (→ graceful skip) when it meets an unresolved call to one, instead
of emitting a raw symbol. This one guard subsumes M1's cases and covers every drop path + both
positions; real ADT constructors (not user functions) are unaffected. M1 is retained only for its
nicer, type-specific message on the common `Option`/`Result` case.

Verified: `useCanon` (record callee) and `pick` (`legal` in `ensures`) now SKIP; `useDbl` (int
callee) and `classify` (enum-in-ensures, no user func) still VERIFY. The misleading M1 hint that
listed "records / monomorphic enum ADTs" as supported callee signatures was corrected.

## Future Work

- **M-SMT-PARAMETRIC-ADT (option b):** actually verify `Option`/`Result`/enum-over-primitive
  callees by (i) registering stdlib ADT variants into `adtTypes`, (ii) monomorphizing the type
  argument into the field sort (`Option[float]` → `(declare-datatype Option ((None) (Some (some_0 Real))))`),
  and (iii) encoding constructor application + `match`. The datatype-declaration machinery
  (`DeclareDatatype`, `types.go:113`) already exists; the missing piece is monomorphization +
  stdlib-ADT registration.

---

**Document created**: 2026-07-14
**Last updated**: 2026-07-14
