# Sprint Plan: M-SMT-CROSS-MODULE-FUNCTIONS

**Sprint ID**: M-SMT-CROSS-MODULE-FUNCTIONS
**Design Doc**: [m-smt-cross-module-functions.md](m-smt-cross-module-functions.md)
**Status**: Planned
**Target**: v0.14.3 (patch on current v0.14.2)
**Estimated**: 5–6 days (~700 LOC including tests)
**Risk**: Medium — extends existing callee_resolver.go machinery; Z3 formula size risk for large callees

## Context

M-SMT-XFUNC (completed) added same-module cross-function call resolution via
`callee_resolver.go`. When the function body has a `VarGlobal` ref to a function
in the same `core.Program`, it resolves and inlines it as a `define-fun`.

The gap: when the `VarGlobal` module is an **imported** module (not a builtin, not
the current module), the resolver has no `core.Program` for it. The call silently
falls through to ADT constructor handling — wrong encoding, silent misverification.

The inbox_v2 cross-module IFC demo worked around this using cross-module TYPES
instead of function calls. This sprint removes that workaround by giving the
encoder imported programs to look up.

## Current Implementation State

```
internal/smt/callee_resolver.go      ← resolves same-module callees only
internal/smt/codegen_apps.go:96–103  ← VarGlobal fallthrough to constructor if not in activeResolvedCallees
internal/smt/codegen.go:63–94        ← EncodeFunctionOpts — has Program, NO ImportedPrograms
cmd/ailang/verify.go                 ← builds Program for current module; doesn't pass imports
```

## Milestones

### M1: Plumbing — ImportedPrograms field (~30 LOC + tests)

**Goal**: `EncodeFunctionOpts` can carry imported modules' `core.Program`s; the
verify command passes them in.

**Files**:
- `internal/smt/codegen.go`: add `ImportedPrograms map[string]*core.Program` to `EncodeFunctionOpts`
- `cmd/ailang/verify.go`: build the map from `result.Modules` and pass into `EncodeFunctionOpts`
- `internal/smt/codegen_test.go` or new `codegen_xmod_test.go`: fixture test — 2-module pair,
  confirm the field round-trips and the encoder receives the imported program

**Acceptance**:
- Field compiles, existing callers pass nil (backwards compat)
- A test fixture with 2 modules passes the imported program through and asserts it's visible to the encoder

**Estimated**: 0.5 day

---

### M2: Inline imported pure-function bodies (~200 LOC + tests)

**Goal**: When encoding a `VarGlobal` call to an imported module, look up the
function in `ImportedPrograms`, extend `callee_resolver.go` to resolve across
module boundaries, and inline the body as a `define-fun`.

**Files**:
- `internal/smt/callee_resolver.go`: extend `ResolveCallees` / `collectCalleeCalls`
  to walk `VarGlobal` refs with `Module != currentModule`, look them up in
  `importedPrograms`, and queue for inlining. Add depth counter
  `crossModuleInlineDepth` (default 3) to prevent inliner explosion.
- `internal/smt/codegen_apps.go`: when `VarGlobal` module isn't builtin/stdlib and
  isn't in same-module resolved callees, check `importedPrograms` lookup before
  falling through to constructor
- `internal/smt/codegen_xmod_inline_test.go` (NEW): two-module fixture where
  `math_lib.double(n)` has `ensures { result == n + n }` and `main.quadruple`
  calls `double(double(n))` with `ensures { result == n * 4 }`. Verifier should
  accept.

**Acceptance**:
- `ailang verify inbox_v2_app.ail` (original design with `fetchMailBody()` imported,
  not the record-type workaround) produces: 3 verified, 2 violations
- `cross_module_types.ail` outcomes unchanged (no regressions)
- New `quadruple/double` fixture passes

**Estimated**: 2 days

---

### M3: Contract-based fallback for large / recursive callees (~300 LOC + tests)

**Goal**: When the importer crosses `crossModuleInlineDepth` or the callee is
recursive, fall back to emitting `(assert (=> requires(args) ensures(result, args)))`
with a fresh `declare-const` for the result — the standard SMT inter-procedural
approach.

**Files**:
- `internal/smt/codegen_xmod_contract.go` (NEW): `EncodeCalleeByContract(callee,
  args, callSite)` — emits `declare-const $result_<site>`, asserts `requires`
  (if present), asserts `ensures` (if present). Returns the result constant name.
- `internal/smt/codegen_xmod_contract_test.go` (NEW): fixture where callee only
  has `ensures { result >= 0 }` (no inlinable body). Caller's `ensures` depends
  on that bound — verifier accepts.
- `internal/smt/codegen_apps.go`: wire contract fallback path when inline is refused
- Second test: mutually-recursive cross-module pair (`f` calls `g`, `g` calls `f`).
  Both have `ensures { result >= 0 }`. Verifier should accept both.

**Acceptance**:
- Recursive cross-module calls (depth-exceeded) fall to contract path, no panic
- A test with an intentionally weak callee contract shows the caller's stronger
  `ensures` produces a violation (proves the encoder is using the contract, not
  silently inlining or skipping)
- `make verify-examples` passes (no regressions across all 20+ contract examples)

**Estimated**: 2 days

---

### M4 (optional stretch): Inline-vs-contract heuristic + CLI flag (~100 LOC + tests)

**Goal**: Automated selection between inline and contract modes; user-facing CLI
override for debugging.

**Files**:
- `internal/smt/codegen_xmod_strategy.go` (NEW): `ShouldInlineImportedCallee(callee)
  bool` — inline iff `nodeCount < 50 && !recursive`. Counting via a lightweight
  AST walk.
- `cmd/ailang/verify.go`: add `--xmod-strategy=auto|inline|contract` flag (default
  `auto`). Pass into `EncodeFunctionOpts`.

**Acceptance**:
- `--xmod-strategy=inline` forces inlining even for large callees
- `--xmod-strategy=contract` forces contract reasoning even for small callees
- `auto` uses heuristic; existing tests pass without `--xmod-strategy` flag

**Estimated**: 1 day (only if M1–M3 are complete with time remaining)

---

## Day-by-Day Plan

| Day | Work |
|-----|------|
| 1 (AM) | M1: `ImportedPrograms` field, verify.go wiring, field round-trip test |
| 1 (PM) | M2 start: extend `callee_resolver.go` to walk cross-module VarGlobals |
| 2 | M2: codegen_apps.go wiring, `quadruple/double` fixture, inbox_v2 redesign |
| 3 | M2: run `make verify-examples`, fix any regressions. Checkpoint. |
| 4 | M3: `codegen_xmod_contract.go`, fallback path, contract-only fixture |
| 5 | M3: mutual recursion test, weak-contract violation test, verify-examples, CHANGELOG |
| 6 (stretch) | M4: heuristic + CLI flag |

## Success Metrics

- [ ] `ailang verify inbox_v2_app.ail` with `fetchMailBody()` as imported call: 3 verified / 2 violations
- [ ] `cross_module_types.ail`: outcomes unchanged (4 verified / 1 SKIP)
- [ ] New `examples/runnable/contracts/cross_module_functions.ail` created and verified
- [ ] `make verify-examples` clean (all 20+ contract examples pass)
- [ ] `make test` and `make lint` green
- [ ] CHANGELOG entry under [Unreleased]
- [ ] Both docs moved to `design_docs/implemented/v0_14_x/` on completion

## Example Files

- `examples/runnable/contracts/cross_module_functions.ail` (NEW) — `double`/`quadruple` demo
- `examples/runnable/contracts/cross_module_functions_lib.ail` (NEW) — `math_lib` module
- `examples/runnable/contracts/inbox_v2_app.ail` (UPDATE) — restore original imported-call design

## Risks

| Risk | Mitigation |
|------|-----------|
| Inliner explosion (large callee) | `crossModuleInlineDepth` cap; M4 heuristic |
| Contract drift (stale iface) | Existing iface digest invalidation handles this |
| Recursive cross-module contracts cause Z3 timeout | Depth cap + telemetry log when fallback activates |
| `verify-examples` regressions | Run after M2 and M3 before merging |

## Dependencies

- [M-SMT-CROSS-MODULE-TYPES](../../implemented/v0_13_0/m-smt-cross-module-types.md) ✅ completed v0.14.3
- [M-SMT-XFUNC](../../../.ailang/state/sprints/sprint_M-SMT-XFUNC.json) ✅ completed (callee_resolver.go plumbing)
- [M-TAINT-TYPES](../../implemented/v0_16_0/m-taint-types.md) ✅ completed v0.16.0 (surfaced the gap)
