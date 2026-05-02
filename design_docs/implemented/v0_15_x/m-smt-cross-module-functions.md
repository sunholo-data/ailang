# M-SMT-CROSS-MODULE-FUNCTIONS: Z3 verification of imported function calls

**Status**: Implemented
**Target**: v0.15.x (implemented 2026-05-02)
**Priority**: P2 — Medium (unblocks several follow-up demos; not on the
critical path for any current sprint)
**Estimated**: ~30–50 hours (1–2 sprints)
**Dependencies**:
  - [M-SMT-CROSS-MODULE-TYPES](../../implemented/v0_13_0/m-smt-cross-module-types.md) — shipped v0.14.3, provides cross-module type resolution
  - [M-TAINT-TYPES](../../implemented/v0_16_0/m-taint-types.md) — shipped v0.16.0, surfaced this gap during M7 cross-module demo

## Problem Statement

`ailang verify` cannot encode imported function calls into the SMT context.
When a function in module B calls a function defined in module A, the Z3
encoder needs the body of A's function to inline it — but A's body lives
in a separate `core.Program` that the verifier no longer has in hand by
the time it walks B's body.

### Concrete failure mode

The M7 cross-module demo for IFC labels originally tried this:

```ailang
-- examples/runnable/contracts/inbox_v2_lib.ail
module examples/runnable/contracts/inbox_v2_lib

export pure func fetchMailBody() -> string<email> ! {}
ensures { result == "raw-email-body" }
{ "raw-email-body" }
```

```ailang
-- examples/runnable/contracts/inbox_v2_app.ail
module examples/runnable/contracts/inbox_v2_app
import examples/runnable/contracts/inbox_v2_lib (fetchMailBody)

export pure func injectedForward(recipient: string) -> SendAction ! {}
ensures  { result.body == "[sanitized]" }
{
  let rawBody = fetchMailBody() in
  { to: recipient, body: rawBody }
}
```

`ailang verify inbox_v2_app.ail` produces:

```
! ERROR injectedForward
  encoding error: cannot encode function body:
  let value: constructor arg 0: unit literals cannot be encoded in SMT-LIB
```

The error path: `let rawBody = fetchMailBody() in …` becomes a Core
`App(fetchMailBody, ())` (zero-arg call → unit literal as the argument).
The encoder reaches the unit literal at `internal/smt/codegen_expr.go:142`
and bails out because SMT-LIB has no `Unit` sort.

The deeper issue is that even **non-zero-arg** cross-module calls fail —
the encoder has no mechanism to look up the imported function's body or
contract from another module's `core.Program`. M7 worked around this by
moving the cross-module flow into a record TYPE (the lib exports a
`Mail` record with labelled fields) instead of a function call.

### Who is affected

- Any contract module that wants to call helpers from another module
  for `requires` / `ensures` / body verification
- The IFC label demo (worked around)
- `ai-coding-lang-bench` benchmarks that span multiple modules
- Future Phase 2+ M-TAINT-TYPES work that wants to track label flow
  through real cross-module function calls

## Goals

**Primary goal**: `ailang verify` accepts cross-module calls to pure
functions and reasons about them using the callee's contract or body.

**Success metrics**:

1. The original M7 design (importing `fetchMailBody`) works end-to-end:
   `ailang verify inbox_v2_app.ail` produces 5 functions: 3 verified,
   2 violations.
2. `cross_module_types.ail` (currently 4 verified / 1 SKIP) gains the
   ability to invoke a helper from `cross_module_types_lib.ail` —
   without regressing any of the 4 currently-verified functions.
3. A new test in `internal/smt/` covers the round-trip:
   "module A defines a pure function with `ensures { result > 0 }`,
   module B calls it, B's `ensures` uses the property — verifier accepts".

**Non-goals** (deferred):

- Cross-module recursion (mutually recursive across module boundaries).
- Effectful imported function calls (Net, FS, IO) — those continue to
  be black-box havoc in the verifier; only pure imports are in scope.
- Cross-package (Go-style import path) verification — confined to
  AILANG modules within the same project.

## Solution Design

### Overview

Two complementary mechanisms, each addressing a different shape of
cross-module call:

1. **Inline the imported body when available.** During verification of
   module B, the loader has already resolved B's imports — it has each
   imported module's `core.Program`. Pass that map down to the SMT
   encoder; when it hits a `core.VarGlobal{Module: "A", Name: "fetchMailBody"}`,
   look up the function in A's program and inline the body using the
   same machinery that handles same-module calls today.
2. **Use the callee's contract as a black-box specification.** When
   the body is unavailable, too large, or the call is recursive, fall
   back to using the callee's `requires` / `ensures` as a specification:
   emit `(assert (=> requires(args) ensures(result, args)))` and a
   fresh constant for the result. This is the standard SMT
   inter-procedural verification approach (Dafny, Why3, etc.).

The first mechanism is mostly plumbing. The second mechanism is the
substantive new work and is the primary deliverable of this sprint.

### Why both

Inlining gives precise reasoning at the cost of program size — large
called functions blow up the SMT formula. Contract-only is bounded but
loses precision for callers that need more than the contract guarantees.
Real verification systems offer both knobs. Default behaviour: inline if
the callee is "small" (< some line/instruction budget) AND non-recursive,
otherwise use the contract.

### Architecture

```
┌─────────────────────────────────────────────────────────────┐
│ verifyCommand (cmd/ailang/verify.go)                        │
│   ├─ pipeline.Run() returns result.Modules (map[path]*Unit) │
│   ├─ NEW: build moduleCorePrograms map[modulePath]*core.Program │
│   └─ pass to smt.EncodeFunctionOpts.ImportedPrograms        │
└────────────────┬────────────────────────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────────────────────────┐
│ smt.EncodeFunctionOpts (internal/smt/codegen.go)            │
│   ├─ existing: Program, SurfaceParams, SurfaceReturnSorts   │
│   ├─ NEW: ImportedPrograms map[string]*core.Program          │
│   └─ NEW: ImportedContracts map[GlobalRef]ContractSpec       │
└────────────────┬────────────────────────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────────────────────────┐
│ codegenExpr (internal/smt/codegen_expr.go)                  │
│   case *core.App:                                           │
│     callee = lookupCallee(...)                              │
│     if callee.IsImported:                                   │
│        if shouldInline(callee):                             │
│           return inlineBody(callee.Program, callee.Body, args)│
│        else:                                                │
│           return assertContract(callee.Contract, args)      │
└─────────────────────────────────────────────────────────────┘
```

### Implementation Plan

#### M1: Plumbing — pass imported programs into the encoder (~10 LOC + tests)

**Files**:
- `cmd/ailang/verify.go`: build a `map[modulePath]*core.Program` from
  `result.Modules` and pass it through `EncodeFunctionOpts`.
- `internal/smt/codegen.go`: add `ImportedPrograms map[string]*core.Program`
  field to `EncodeFunctionOpts`. Wire to the encoder.
- `internal/smt/codegen_test.go`: golden-style test confirming the field
  round-trips and the encoder can look up an imported function by ref.

**Acceptance**:
- New field exists on `EncodeFunctionOpts`. Existing callers still compile
  (backwards compat).
- A new test passes a 2-module fixture and confirms the encoder sees the
  imported program.

**Pause**: none — fold into M2.

#### M2: Inline imported pure-function bodies (~150 LOC + tests)

**Files**:
- `internal/smt/codegen_expr.go`: in the `*core.App` case, when the
  callee is a `*core.VarGlobal` referencing an imported module, look up
  the body and inline it by recursing the encoder with the args
  substituted. Apply the existing same-module inlining strategy.
- `internal/smt/codegen_xmod_inline_test.go` (NEW): cross-module call
  with `result == arg + 1` ensures clause; verifier must catch a violation
  in the caller that asserts `result < arg`.

**Cycle protection**: each imported function has a depth counter; refuse
to inline beyond `crossModuleInlineDepth` (default 3). At the limit, fall
back to contract-based reasoning (M3).

**Acceptance**:
- Cross-module `helper(n)` call (with helper's body inlined) lets the
  verifier reason about `helper` invocations the same way it does for
  same-module calls today.
- The `inbox_v2_lib + inbox_v2_app` redesign — putting `fetchMailBody`
  back as an imported call — verifies cleanly.
- Existing `cross_module_types.ail` outcomes unchanged.

**Pause**: none — fold into M3.

#### M3: Use callee's contract as black-box spec (~250 LOC + tests)

**Files**:
- `internal/smt/codegen_xmod_contract.go` (NEW): given a callee's
  contract (its `requires` / `ensures` from `core.Meta.Contracts`),
  emit `(declare-const $result_<callerSite> SORT)` followed by
  assertions: `(assert <requires-translated>)` and
  `(assert <ensures-translated-with-result-bound>)`.
- `internal/smt/codegen_expr.go`: when inlining is refused (too deep,
  too large, recursive), fall back to contract-based reasoning. Track
  fallback decisions in encoder telemetry so we can tune the threshold.
- `internal/smt/codegen_xmod_contract_test.go` (NEW): tests that confirm
  the encoder produces a valid SMT-LIB script when the callee has only
  a contract (no body inlining), and that the resulting verification
  outcome matches the contract's promise.

**Acceptance**:
- Recursive cross-module calls (callee calls back into caller's module)
  no longer cause the inliner to spin. They fall through to the contract
  path.
- A test where the callee's `ensures` is intentionally weak shows the
  caller's stronger `ensures` becoming a violation — verifying that the
  encoder is using the contract, not silently inlining.
- `make verify-examples` passes (no regressions).

**Pause**: STOP after M3. Cross-module function-call verification is
feature-complete. Report verification outcomes for the M7 redesign and
any other multi-module examples before moving on.

#### M4 (optional): Heuristic for inline vs contract (~80 LOC + tests)

**Files**:
- `internal/smt/codegen_xmod_strategy.go` (NEW): function-size heuristic
  (line count, AST node count, recursive depth) selects inline vs
  contract automatically. Initially: inline iff `nodeCount < 50 && !recursive`.
- CLI flag `--xmod-strategy=auto|inline|contract` (default auto) lets
  users override the heuristic for debugging.

**Acceptance**:
- `--xmod-strategy=inline` forces inlining even for large callees;
  `--xmod-strategy=contract` forces contract-based reasoning even when
  the body is small. `auto` uses the heuristic.

**Pause**: none — final cleanup.

### Test Strategy

- New test fixtures in `internal/smt/testdata/xmod_calls/` — minimal
  2-module pairs covering each verification path.
- Re-enable the original M7 design (with `fetchMailBody` as an imported
  call) and confirm the same 3-verified / 2-violation outcome as the
  current type-based workaround.
- Property-style tests: for any pure helper with a non-trivial contract,
  inlining and contract-based reasoning must agree on the verification
  outcome (modulo precision differences that should manifest as
  more-counterexamples-found, not fewer).

### Risks

- **Inliner explosion.** A very large imported function inlined into many
  call sites blows up the SMT formula. Mitigated by the size heuristic
  (M4) and the `crossModuleInlineDepth` cap (M2).
- **Contract drift.** A callee's contract that the caller depends on
  changes; cached ifaces become stale. Mitigated by the existing iface
  digest mechanism — contract changes invalidate the iface, which
  invalidates downstream modules.
- **Recursive contracts.** Mutually-recursive cross-module functions
  cannot be inlined and must use the contract. This works correctly
  per design, but the user-facing diagnostic must explain why the
  encoder chose the contract path.

## Examples

### Example 1: Imported helper with a contract

```ailang
-- math_lib.ail
module myapp/math_lib

export pure func double(n: int) -> int ! {}
ensures { result == n + n }
{ n + n }
```

```ailang
-- main.ail
module myapp/main
import myapp/math_lib (double)

export pure func quadruple(n: int) -> int ! {}
ensures { result == n * 4 }
{ double(double(n)) }
```

`ailang verify main.ail` should accept `quadruple` as verified by
inlining `double` twice (small, non-recursive, two call sites).

### Example 2: Recursive cross-module fallback

```ailang
-- a.ail
module myapp/a
import myapp/b (g)
export pure func f(n: int) -> int ! {}
ensures { result >= 0 }
{ if n <= 0 then 0 else g(n - 1) }
```

```ailang
-- b.ail
module myapp/b
import myapp/a (f)
export pure func g(n: int) -> int ! {}
ensures { result >= 0 }
{ if n <= 0 then 0 else f(n - 1) }
```

The mutual recursion across module boundaries forces the verifier to
fall back to contract-based reasoning. With `ensures { result >= 0 }`
on both, the cross-call assertion `(assert (>= $result_g_at_f 0))`
discharges `f`'s `ensures`.

## Out of Scope

- Effectful imported calls. `! {Net}` / `! {FS}` / `! {IO}` calls remain
  havoc'd at verification time.
- Cross-package verification. Within a single AILANG project only.
- The standard library. `std/string.endsWith` and friends already work
  via the named-builtin path; they are not affected by this sprint.

## References

- M-SMT-CROSS-MODULE-TYPES (`design_docs/implemented/v0_13_0/`) — sister
  sprint that delivered cross-module *type* resolution.
- M-TAINT-TYPES M7 (`design_docs/implemented/v0_16_0/m-taint-types.md`)
  — surfaced this gap.
- Dafny inter-procedural verification:
  https://dafny.org/dafny/DafnyRef/DafnyRef#sec-method-specification
- Why3 module system: https://www.why3.org/doc/syntax.html#sec-modules
