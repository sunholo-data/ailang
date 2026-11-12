# M-BYTECODE-STRAIGHTLINE-REG-REUSE — strict-VM register exhaustion in long straight-line functions: root-frame bindings (sequential lets and every elaborated `$tmp` of a long string interpolation) are pinned for the function's entire lifetime; there is no release after last use

**Status**: Planned
**Target**: v0.60.4
**Priority**: P0 (the strict VM — the determinism gate — silently degrades to `BC001` on exactly the flat, generated shapes AI code emits: dozens of sequential lets plus printlns that interpolate them; reported from sunholo/worlds `_smoke.ail` in the field)
**Estimated**: 2–3 days (~250 LOC, pure `internal/bytecode/compiler/` + tests; no VM, no parser, no typechecker, no language semantics change)
**Dependencies**: None. Builds on the already-landed Phase A of [m-bytecode-strict-256-reg-ceiling.md](../v0_53_1/m-bytecode-strict-256-reg-ceiling.md) (`BC001`, `check --strict-bytecode`, strict pre-flight) — this doc does not change that surface, it removes the pressure that triggers it.

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Removes a class of engine divergence: today the evaluator runs a 40-let function the strict VM refuses (one engine answers, the other traps `BC001`); after the fix both report the same output on the same program. The release decision is a pure function of the flat statement list — same input, same registers, same instruction bytes — so no new nondeterminism is introduced. |
| A2: Replayability | 0 | No trace/replay surface change. |
| A3: Effect Legibility | 0 | No effect annotations change; all affected code is ordinary `let` + `println` bodies. |
| A4: Explicit Authority | 0 | No capability surface change. |
| A5: Bounded Verification | +1 | Register pressure becomes liveness-bounded and per-statement checkable (unit-assertable `NumRegs` on generated functions of any length), instead of a whole-function ceiling that only failure reveals. |
| A6: Safe Concurrency | 0 | No concurrency change. |
| A7: Machines First | +1 | The failing shapes are exactly what AI generators write: a data-prep `main` with a dozen lets plus two or three diagnostic printlns interpolating them (field report), and 40-plus-value straight-line bodies. The human workaround (move the printlns into a separate function, as the reporter did) is manual register-pressure management — precisely the machine-facing friction this removes. |
| A8: Minimal Syntax | +1 | No new syntax; fixes codegen of already-taught, already-type-checking constructs. |
| A9: Cost Visibility | 0 | No resource semantics change; register reuse reduces frames, it does not hide cost. |
| A10: Composability | 0 | Fix is internal to one compiler package. |
| A11: Structured Failure | +1 | A pre-pass unsoundness cannot produce silently wrong bytecode: a read of a released binding is a loud compile error (`unbound variable`) that tags the function `EvalOnly` and surfaces as the existing coded, positioned `BC001` — the failure mode degrades to today's behavior, never below it. |
| A12: System Boundary | 0 | No boundary change. |

**Net Score: +5** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): no implicit nondeterminism introduced (release schedule is a pure function of the statement list)
- [x] A3 (Effects): no hidden side effects
- [x] A4 (Authority): no ambient access granted
- [x] A7 (Machines First): fixes a machine-facing gap; optimizes nothing for human convenience

## Problem Statement

Reported from stapledons-godot (AILANG v0.56.1, 05c860a), field hit in `sunholo/worlds/_smoke.ail`: a `main` with about a dozen float lets plus three printlns that interpolate them cannot run under `--strict-bytecode`:

```
bytecode execution failed: entry "main" is evaluator-only
(register allocator: contiguous block of 3 would exceed 256);
strict mode disables bridge dispatch
```

The reporter's generator reproduces it cleanly (n float lets + one println interpolating all n): 30 values run; 40, 80 and 150 fail with the `contiguous block of 3` message; 300 fails with `function exceeds 256-register limit`. The interpreter runs all of them. Workaround in the field: move the new printlns into their own function.

**Reproduced and measured on the installed binary (v0.60.2, 33f3009 — Verification Log V1; repo HEAD is v0.60.3, same `regalloc.go` code path).** The reporter's thresholds sharpen to exact ones:

| Program shape | Result today |
|---|---|
| n lets + one println interpolating all n | n ≤ 36 passes (`reg36` compiles at `regs=250`); n ≥ 37 fails `BC001 … contiguous block of 3 would exceed 256` |
| One let + a single println interpolating the same variable n times | n = 40 passes at `regs=239`; n ≥ 45 fails (`regs` grows ~6 per interpolated hole: 179 at n=30, 209 at n=35, 239 at n=40) |
| 15 lets + 3 printlns each interpolating all 15 (the `_smoke.ail` shape) | fails `contiguous block of 3`; 14 lets + 3×14 passes |
| 100 lets + a trivial println | passes |
| 300 lets + a trivial println | fails `function exceeds 256-register limit` |

The second row is the important one: **a single string interpolation with ~45 interpolated holes exhausts the register file on its own**, with one live user variable. Register pressure is proportional to the *total* number of bindings a function ever creates, not to the number simultaneously live — nothing is ever released.

### Why nothing is ever released (root cause, code-verified end to end)

1. **Parser** desugars an interpolated string into a left-associative chain of `concat_String` calls, one `show(...)` per hole (`internal/parser/parser_literals.go:137-227`, `parseInterpolatedString`; the fold at `:222-227`). n holes with separators ⇒ ~2n−1 call nodes.
2. **Elaboration** normalizes to ANF: `normalizeToAtomic` (`internal/elaborate/core.go:349-369`) binds every non-atomic argument of every call to a fresh `$tmpN`, and `wrapWithBindings` (`core.go:381-392`) wraps them as nested `core.Let`s. The scope extent — "$tmpN lives until its one consumer" — is *present* in Core IR.
3. **Lowering** destroys it: `FlattenBlock` (`internal/gen/lower/block.go:36-47`) flattens the nested let-chain into a **flat sequence of root-level `stmt.VarDecl`s**. This is the heart of the imperative Statement IR and is correct for semantics — but the per-binding extent is gone.
4. **The bytecode compiler** then pins every one of those flat `VarDecl`s for the function's whole lifetime: `compileVarDecl` (`internal/bytecode/compiler/stmt.go:70-86`) binds via `bindScoped`, and root-scope frames are never popped — by design (`internal/bytecode/compiler/regalloc.go:160-176`: "Root-scope bindings (function-level lets) are never popped, so they stay pinned for the function's lifetime").
5. **Per-hole cost:** each hole permanently pins ~2 registers (the `show` result `$tmp` and the `concat_String` result `$tmp`), and each `concat_String` frame (`compileBuiltinCall`, `builtins.go:88-120`: `allocContig(n+1)` = the "contiguous block of 3") bumps fresh registers because the freed arg-slot runs are returned only *after* the next frame's `allocContig` has already bumped. Net: high-water mark ≈ 6 registers per interpolated hole, plus one pin per user let. 256 − overhead is reached at ~36–44 holes.

Nested-scope lets are *not* affected — `compileIfStmt`/`compileSwitch` push and pop scope frames around branch bodies (`control_flow.go:59-88`, `switch.go:78-113`), so bindings created inside control flow already recycle (M-BYTECODE-REGALLOC-FIX, v0.11). The pathology is specific to the **flat root-frame statement list** — which is what every straight-line function body is.

### Impact

- **AI code generators** (primary users) emit long straight-line bodies and multi-hole interpolations naturally — logging, state dumps, "print all the fields" diagnostics. The language type-checks these programs (`ailang check reg40.ail` → `✓ No errors found!`, V6) and the evaluator runs them; the strict VM refuses them at dispatch with `BC001`.
- **Field cost:** the reporter manually split `_smoke.ail`'s printlns into a separate function to regain strict coverage — hand-managed register pressure, invisible to the next agent that adds a println.
- The strict VM is the determinism gate the program leans on; every function that silently degrades to the bridge (non-strict `--bytecode` runs it with a warning, V7) is coverage lost from that gate for a *non-semantic* reason.

## Goals

**Primary Goal:** make register pressure a function of *simultaneously live* bindings instead of *total* bindings, so long straight-line functions (many sequential lets, one or many interpolations, many sequential printlns) compile to bytecode and run identically on both engines; keep the existing `BC001` diagnostic as the loud residual for functions that are genuinely over the live ceiling.

**Success Metrics:**
- 40 lets + one 40-hole interpolation (the reporter's repro) compiles and runs under `--strict-bytecode` with evaluator-identical output, at `NumRegs` ≤ ~50 (measured today: never compiles).
- An 80-hole and 300-hole single interpolation compile and match the evaluator (today: fail at 45).
- 300 lets consumed progressively (each used once, then dead) compiles (today: fails).
- 15 lets + 3×15-hole printlns (the `_smoke.ail` shape) compiles (today: fails).
- A function with >256 genuinely simultaneous live bindings (e.g. a 300-field record literal) still fails loudly with `BC001` naming the function, file:line — the existing fixture (`cmd/ailang/check_strict_bytecode_test.go:103`) keeps passing.
- All existing bytecode compiler tests and engine-parity corpus green; `make check-boundaries` clean.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Fix lane is **compiler-only**: last-use release of root-frame bindings inside `internal/bytecode/compiler/` | The binding-lifetime information exists nowhere else after `FlattenBlock` flattens it; reconstructing it in the compiler needs no IR, evaluator, or VM change (blast radius = 1 package) | agent | design | med |
| Release applies to **all eligible root-frame `VarDecl`s** (user lets and `$tmp`s alike), not just `$tmp*` | User lets are half the reported pressure (300-let case); restricting to `$tmp` would fix interpolations but leave long sequential bodies degraded | agent | design | low |
| Eligibility excludes RecGroup members and any name ever on an `AssignStmt` LHS | letrec members are closure-rebuilt from capture registers (`letrec.go`), and assign-targets (`$valTmp`, match/if flags) are mutated after binding — releasing either would be wrong | agent | design | low |
| Statement-granular release, not instruction-granular | References inside nested if/switch branches compile inline within one top-level statement; releasing after that statement is exactly sound. Instruction-granularity buys nothing measurable and costs a real interval analysis | agent | design | low |
| A read of a released binding is a **loud compile error** (→ `EvalOnly` → `BC001`), never a reused register | Guarantees the worst case of a pre-pass bug is today's behavior (function degrades loudly), never silently wrong bytecode | agent | design | low |
| No spilling, no ceiling raise, no instruction-encoding change | All three re-open frozen VM surface (prior doc ruled: 8-bit ABC operands are core-floor); liveness release makes them unnecessary for the reported class | human (inherited from v0_53_1 rulings) | design | high (avoided) |

### Design Freeze

- [x] Fix lane: `internal/bytecode/compiler/` only. No `internal/vm`, `internal/eval`, `internal/gen`, `internal/elaborate`, parser, or CLI surface changes (a census re-run and CHANGELOG/doc note are the only non-compiler files touched).
- [x] `BC001` remains the sole user-facing diagnostic; no new error code is allocated (the loud guard is an internal compile-error string carried in the existing `EvalReason` detail).
- [x] Release schedule is computed per function from the flat statement list *before* codegen (deterministic pre-pass), not lazily during compile.
- [ ] Sprint-planner confirms the acceptance thresholds (the four repro shapes above) become the first test file, before any allocator change.

## Conflict Surface

This design touches `internal/bytecode/compiler/` (regalloc/scoping/codegen). Required enumeration:

1. **Positions extended:** the lifetime of root-scope bindings in `scopeStack`'s root frame (`regalloc.go:155-231`). Today a root-frame binding is unconditionally pinned until function end; after the change it is pinned until one top-level statement past its last read. Nothing else in the allocator changes: `allocPinned`, `allocTemp`, `allocContig`, free-list behavior, `highWater` semantics are untouched.
2. **Other valid constructs already living in those positions (audited — every producer of root-frame bindings):**
   - Parameters (`compiler.go:315-321`) — must stay pinned for the function lifetime. **Excluded**: the pre-pass only releases `VarDecl` bindings.
   - `letrec` group members (`letrec.go:103-161`, bound via `bindScoped` after `compileRecGroup`) — excluded by the RecGroup rule.
   - `elseIfInline` alias bindings (`control_flow.go:74-88`) — bound with plain `bind` in a *pushed* frame, never the root frame; unaffected.
   - Lambda/letrec **captures** (`lambda.go:43-96`, `resolveCaptures`) — captures are read from outer registers *at `OpClosure` execution time and copied by value* into the closure (`internal/vm/vm.go:407-430`, V10); the enclosing top-level statement containing the `OpClosure` is itself a use, so last-use release after it is sound. The pre-pass walker must count `stmt.Lambda` bodies' reads at the enclosing statement's index (the existing `freeVarVisitor` already descends into lambdas, `lambda.go:236-252`).
   - Switch case bindings (`switch.go:78-113`) — scoped frames, already recycled on pop; the pre-pass does not touch pushed frames.
   - `$valTmp` / match-flag bindings created by `flattenValue`/`rewriteReturnsToAssign` (`block.go:213-320`, `match_ifchain.go`) — always `AssignStmt` targets, excluded by the assign rule (they are few per function; they stay pinned as today).
3. **How the pre-pass disambiguates:** binding *instances*, not names. A re-binding (`let x = x + 1` shadowing) creates a new instance; each read of a root-frame name is attributed to the currently-live instance at that point in the flat walk. Name-keyed accounting would be conservative but under-deliver (a shadowed name's earlier register would only free at the name's global last use).
4. **Programs that MUST still work post-change (regression fixtures — all exist, verified):**
   - `cmd/ailang/check_strict_bytecode_test.go` — asserts the 300-field record literal still reports `BC001 … contiguous block of 300 would exceed 256 … at wide.ail:3` (genuine liveness; unaffected by release).
   - `internal/bytecode/compiler/regalloc_test.go` — allocator unit tests (API untouched).
   - `internal/bytecode/compiler/control_flow_test.go`, `collections_test.go`, `call_test.go`, `lambda_test.go`, `letrec_test.go` — codegen shape tests (chunking, `elseIfInline`, captures, letrec).
   - `internal/bytecode/compiler/determinism_test.go` — byte-identical repeated compiles (release schedule is a pure function; must keep passing).
   - `cmd/ailang/run_bytecode_match_test.go` — engine-parity corpus (strict VM output must keep matching the evaluator).
   - `internal/bytecode/compiler/ctor_temps_test.go` — `inlineCtorTemps` behavior (pre-pass runs *after* `inlineBodyCtorTemps` rewriting, so the two rewrites compose deterministically; ordering fixed in the implementation plan).
5. **What deliberately changes:**
   - `NumRegs` and (for functions whose bodies contain `$tmp`s or dead lets) disassembly change — freed slots are reused. No golden in `goldens/` pins a prototype's `NumRegs` today (sprint task: `grep goldens/` to confirm); `disasm` output is not a stable contract.
   - Fewer functions are tagged `EvalOnly`, so the non-strict `--bytecode` warning list and the `check --strict-bytecode` `BC001` report shrink — the intended effect. Scripts that count `BC001` entries (the v0_53_1 census) will see the delta; the census doc is re-run, not silently invalidated.
   - A released-then-read binding now errors at compile time where it previously compiled (this can only arise from a lower-pass bug; today's equivalent guard class is `isLowerPassDictFallback`, `builtins.go:52-70`).

## Solution Design

### Overview

One mechanism: a **deterministic last-use release schedule** for root-frame bindings, computed by a pre-pass over the flat statement list and applied by `compileStmts` after each top-level statement. Register pressure becomes `max simultaneous live bindings + per-expression temps` instead of `total bindings`.

Pressure arithmetic after the fix, for the interpolation chain `$tmp1 = show(a0); $tmp2 = " "; $tmp3 = concat($tmp1, $tmp2); $tmp4 = show(a1); …` — each `$tmp`'s last read is the initializer of the statement that consumes it, so each is released one statement later and its slot is reused by the next temp; a chain of any length compiles at ~constant pressure. The user lets `a0..a(n-1)`, each read once by `show(a_i)`, recycle the same way. Peak for the 40-let repro is a handful of registers plus the largest single call frame — not 250+.

### Architecture

**1. Liveness pre-pass (new file `internal/bytecode/compiler/liveness.go`, ~120 LOC).**

A pure function `releaseSchedule(body []stmt.Stmt, ret stmt.Expr) []releaseEvent`:

- Walks `inlineBodyCtorTemps(body, ret)`'s output — the exact list `compile()` will compile — once, in order, treating the trailing return expression as one final statement (the same convention `inlineBodyCtorTemps` already uses, `ctor_temps.go:70-76`).
- Maintains `name → binding instance` (a new instance on each root-level `stmt.VarDecl`; shadowing handled per Conflict Surface §3).
- For each root-level statement index, records reads (extend the existing complete `freeVarVisitor`, `lambda.go:122-274`, which already visits every statement kind — including nested if/switch bodies and lambda bodies — and every expression kind that can read a local; `GlobalRef` is correctly absent since it resolves through `funcIdx`, never `locals`, V11) and marks:
  - `lastUse[instance] = max(statement index containing a read)`,
  - `ineligible[instance] = true` if the instance is a `RecGroup` member, or its name appears as an `AssignStmt` LHS anywhere in the function (`AssignStmt` names are a def-site, which today's visitor omits — add it),
  - **dead-on-arrival is fine**: an instance never read has `lastUse` = its own declaration index and releases immediately (a side-effecting-but-unused let still executes its initializer; only its register goes back).
- Emits, per statement index, the ordered list of instances to release after that statement (declaration order, deterministic).

**2. Application in `compileStmts` (`letrec.go:31-61`).**

`funcCompiler` gains the schedule; after each successful top-level `compileStmt`, it releases the scheduled instances: remove the name from the scopeStack root frame and `freeTemp` its register. Release only ever touches the root frame (pushed frames keep their existing pop semantics, and `compileRecGroup` runs are compiled as a unit before the schedule's cursor advances past them). New helper on `scopeStack` (~15 LOC): `releaseRoot(name)` — delete from `frames[0].names` and return the register.

**3. Loud guard (in `compileVarRef`, `expr.go:170-205`).**

Nothing new to build: once a binding is removed from the root frame, a later read falls through to `recMember`/`hasFunc`/`preludeFunc` and errors `compiler: unbound variable %q` — the function is tagged `EvalOnly` and surfaces as `BC001` with the reason string. The design leans on that existing path *deliberately* (A11): the failure mode of a pre-pass bug is the current behavior. The guard's error string should name the situation ("binding %q released after last use; lower pass emitted a read past the schedule") so a future lower-pass bug is diagnosable as such, the same way `isLowerPassDictFallback` labels its class.

**4. Census + docs (Phase B of v0_53_1, repeated).**

Re-run the `BC001` census over `std/`, `examples/`, `goldens/` and append the delta to [m-bytecode-strict-256-reg-ceiling-census.md](../v0_53_1/m-bytecode-strict-256-reg-ceiling-census.md); CHANGELOG entry; note in the strict-bytecode docs page that register pressure is liveness-based for straight-line functions and what the remaining genuine ceiling is.

### Implementation Plan

**Phase 1: pre-pass + release (the fix, ~1.5 days)**
- [ ] `liveness.go`: `freeVarVisitor` extension (assign-LHS tracking, instance tracking, per-index last-use) + `releaseSchedule` (~120 LOC)
- [ ] `letrec.go`/`stmt.go`: schedule storage on `funcCompiler`, release hook after each top-level statement, `scopeStack.releaseRoot` (~30 LOC)
- [ ] Ordering pinned: schedule computed from `inlineBodyCtorTemps` output; `compile()` must not re-run `inlineCtorTemps` on an already-processed list (it currently calls it once, `compiler.go:328` — assert idempotence in a unit test)
- [ ] Unit tests: shadowing (`let x = 1; let x = x + 1`), assign-excluded `$valTmp`, letrec-excluded members, lambda-captured binding released only after the `OpClosure` statement, dead lets, per-index release order (~150 LOC)

**Phase 2: acceptance + parity (~1 day)**
- [ ] Generated-repro tests (the four success-metric shapes) under `cmd/ailang/` mirroring `run_bytecode_match_test.go`: strict VM === evaluator output; `NumRegs` assert via `Compile` unit test (e.g. 40-let repro ⇒ `NumRegs` ≤ 50; 300-hole interpolation ⇒ `NumRegs` ≤ 32)
- [ ] `make test` + `make check-boundaries`; grep `goldens/` for `NumRegs` pins
- [ ] Mutation check: break the schedule (release one statement early) and confirm the loud guard fires, not wrong output

**Phase 3: census + docs (~0.5 day)**
- [ ] Re-run BC001 census, append delta; CHANGELOG; strict-bytecode docs note

### Files to Modify/Create

**New files:**
- `internal/bytecode/compiler/liveness.go` (~120 LOC) + `liveness_test.go` (~150 LOC)

**Modified files:**
- `internal/bytecode/compiler/letrec.go` — schedule on `funcCompiler`, release hook in `compileStmts` (~25 LOC)
- `internal/bytecode/compiler/stmt.go` — nothing in `compileVarDecl` itself; only a comment update pinning the new contract (~5 LOC)
- `internal/bytecode/compiler/regalloc.go` — `scopeStack.releaseRoot` (~15 LOC)
- `internal/bytecode/compiler/compiler.go` — comment update (allocator description, ~5 LOC)
- Tests: `cmd/ailang/straightline_reuse_test.go` (generated repros, ~120 LOC)
- Docs: census doc append + CHANGELOG

## Examples

### Example 1: the reporter's repro

```ailang
module reg40
export func main() -> () ! {IO} {
    let a0 = 1.5*2.0
    -- … 39 more lets …
    println("${a0} ${a1} … ${a39}")
}
```

**Before:**
```
$ ailang run --strict-bytecode --caps IO --entry main reg40.ail
Error: bytecode execution failed: 1 evaluator-only function(s) reachable
from main; --strict-bytecode wires no evaluator bridge:
  BC001: function reg40.main is evaluator-only under --strict-bytecode
  (register allocator: contiguous block of 3 would exceed 256) at reg40.ail:2
```
**After:** prints the 40 values, identical to the evaluator; `ailang disasm` shows `main` at `NumRegs` ≈ 10–50 and the `$tmp` chain reusing a handful of slots.

### Example 2: the `_smoke.ail` shape (15 lets, 3 printlns × 15 holes)

**Before:** `BC001 … contiguous block of 3 would exceed 256` (measured; 14 lets passes, 15 fails).
**After:** compiles; each println's `$tmp` chain releases after its own statement; the 15 user lets release after their last read in the final println.

### Example 3: what still fails (correctly)

A 300-field record literal — 300 values simultaneously live at one `MAKE_RECORD` — keeps today's behavior and diagnostic:
```
BC001: function m.wide is evaluator-only under --strict-bytecode
  (register allocator: contiguous block of 300 would exceed 256) at wide.ail:3
```
The fixture asserting this (`cmd/ailang/check_strict_bytecode_test.go:103`) must keep passing: release-after-last-use cannot reduce a peak of genuinely simultaneous live values.

## Success Criteria

- [ ] The reporter's repro (40 lets + 40-hole interpolation) and the generalizations (80, 300 lets with progressive consumption) run under `--strict-bytecode --caps IO` with evaluator-identical output.
- [ ] A single interpolation of 300 holes compiles (measured today: fails at 45).
- [ ] The `_smoke.ail` shape (15 lets + 3×15-hole printlns) compiles.
- [ ] `NumRegs` is liveness-bounded: unit test asserts 40-let repro ≤ 50, 300-hole interpolation ≤ 32.
- [ ] `BC001` remains for genuine >256-live functions; `check_strict_bytecode_test.go` (300-field record) unchanged and passing.
- [ ] All tests passing: `make test`, `make check-boundaries` (no layer violations — change is inside `internal/bytecode/compiler` only), `determinism_test.go` (byte-identical repeated compiles), engine-parity corpus.
- [ ] Census delta documented; CHANGELOG updated.

## Testing Strategy

**Unit tests (`internal/bytecode/compiler/liveness_test.go`):**
- Schedule correctness: per-index release lists for shadowing, dead lets, assign-excluded, letrec-excluded, lambda-captured, branch-read (read inside an if counts at the enclosing statement).
- `releaseSchedule` idempotence under the exact `inlineBodyCtorTemps` composition `compile()` performs.

**Integration (`cmd/ailang/straightline_reuse_test.go`):**
- The four success-metric shapes, generated in-test (the reporter's generator is ~10 lines of Go): strict VM output === evaluator output; exit codes; `NumRegs` asserts via direct `Compile` calls.
- Non-strict `--bytecode` runs of the same files: no evaluator-only warning remains (the stderr warning asserted by Phase A tests shrinks to zero for these files).

**Manual/mutation:**
- `ailang disasm` before/after on the repros: confirm slot reuse in the `$tmp` chain.
- Deliberately mis-schedule (release early) in a scratch build: confirm `unbound variable … released after last use` compile error → `EvalOnly` → `BC001`, never silent wrong output.

## Deferred Decisions

- **Instruction-granular release** (release inside a long initializer before the statement ends): buys peak reduction only for single statements with hundreds of simultaneous temps — the 300-field record class, which stays `BC001` regardless. Agent may skip entirely.
- **Releasing `$valTmp`/match-flag temps** via re-materialization analysis: they are few per function (one per hoisted if/match value position); skip unless the post-fix census shows a real program pinned by them.
- **Widening `inlineCtorTemps` from constructor positions to call-arg positions**: subsumed by last-use release for the reported shapes; the evaluation-order guard that keeps `inlineCtorTemps` conservative stays untouched.
- **Extending the walker to emit a richer live-interval table** (for a future linear-scan allocator): the schedule data structure may be shaped to allow it, but nothing in this sprint depends on it.

## Non-Goals

**Not attempted in this feature:**
- **Register spilling (`OpSpill`/`OpRestore`)** — ~500 LOC of new VM ABI per the v0.11 assessment (M-BYTECODE-REGALLOC-FIX), carried as a non-goal in the v0_53_1 sibling; revisit only if the post-fix census shows real programs with >256 genuinely simultaneous live values (each will be a named `BC001`).
- **Raising the 256-register ceiling or widening instruction operands** — frozen 8-bit ABC encoding; core-floor lane (v0_53_1 ruling, inherited).
- **>255-element single-instruction collection construction** (chunked list lowering already landed; tuple chunking deferred there, still deferred here).
- **Changes to elaboration/lowering scope structure** (scope markers in Statement IR) — the stmt IR is shared with the evaluator and Go codegen; a compiler-local schedule achieves the same result with a fraction of the blast radius.
- **Any evaluator, VM, parser, typechecker, std, or CLI behavior change** (the strict/non-strict CLI surface is frozen as-is).

## Timeline

**Week 1 (~2.5 days, estimates are 2× initial guess):**
- Phase 1: pre-pass + release + unit tests (1.5 d)
- Phase 2: generated-repro acceptance + parity + mutation (1 d — partially overlaps Phase 1)
- Phase 3: census + docs (0.5 d)

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Pre-pass unsoundness (a binding released while still read) | Med | Loud guard: read-after-release is a compile error → `EvalOnly` → `BC001`; mutation test proves the guard fires; worst case = today's behavior, never wrong bytecode (A11) |
| Pre-pass incompleteness (a read path not walked → false release → loud errors on valid code) | Med | Reuse the existing complete `freeVarVisitor` (it already powers capture analysis — the same completeness requirement); engine-parity corpus + `make test` as the net |
| Register reuse changes disassembly of existing corpus functions (`NumRegs` shrinks) | Low | No golden pins `NumRegs` (sprint greps `goldens/` to confirm); determinism is preserved (byte-identical across compiles of the same input); `determinism_test.go` is the regression net |
| Shadowing subtleties (instance vs name accounting) | Med | Instance-keyed schedule by design; dedicated unit test (`let x = 1; let x = x + 1; println(x)`) asserts each instance's release index |
| Fewer `EvalOnly` tags changes census/warning baselines | Low (it's the point) | Census re-run and delta doc'd in Phase 3; `strict_report_test.go`/Phase A tests assert report *format*, not counts |

## Related Documents

**Sibling (checked for overlap — duplicate gate):**
- [m-bytecode-strict-256-reg-ceiling.md](../v0_53_1/m-bytecode-strict-256-reg-ceiling.md) — the parent effort: Phase A (`BC001`, check/run gates, census) and Phases B1/B2 (list-literal chunking, else-if chain result reuse) have **landed**; its census showed 0 register-ceiling entries across the in-repo corpus, so the remaining pressure comes from field/AI-generated shapes, not std. **This doc is distinct**: different root cause (root-frame binding lifetime in straight-line bodies vs. single-construct oversized lowering), different fix (liveness release vs. chunking/re-association). Its rulings are inherited: no spilling, no ceiling raise, `BC001` as the diagnostic. Its Deferred Decision "a smarter general-purpose allocator — revisit only if the post-fix census shows real programs still failing" is exactly the trigger this doc responds to.
- [m-bytecode-strict-256-reg-ceiling-census.md](../v0_53_1/m-bytecode-strict-256-reg-ceiling-census.md) — the census this doc's Phase 3 extends.

**Implemented (informs the design):**
- [m-bytecode-regalloc-fix.md](../implemented/v0_11_0/m-bytecode-regalloc-fix.md) — v0.11: scope recycling (`scopeStack`) and contiguous-from-free-list; established that nested-scope bindings recycle and that spilling is ~500 LOC and overkill; this doc is the root-frame half of the same problem, unaddressed then because "the corpus is small enough that the 256-register ceiling is not a constraint" (`regalloc.go:11-14`).
- [m-bytecode-2d-parity.md](../implemented/v0_11_0/m-bytecode-2d-parity.md) — the `EvalOnly` stub + bridge contract this design preserves.
- #1865 / `ctor_temps.go` — single-use `$tmp` inlining for constructor literals; this doc generalizes the underlying insight (compiler temps should not be permanent) to call chains and user lets, by lifetime rather than by syntactic position.

**Planned (checked for overlap):** none — the create script's SimHash/neural searches returned no matches on "bytecode straightline reg reuse" (auto-query too narrow), so a manual sweep of `design_docs/` was performed: the only register-pressure docs are the sibling above and the v0.11 regalloc-fix (shipped, different scope).

## References

- [Design Axioms](/docs/references/axioms) — the 12 non-negotiable principles
- Field report: sunholo/worlds `_smoke.ail` (AILANG v0.56.1, 05c860a); reporter's generator + thresholds quoted in the Problem Statement
- [M-BYTECODE-2D M3 contract](../implemented/v0_11_0/m-bytecode-2d-parity.md) — per-function compile failure tags `EvalOnly`; this design keeps that contract and its strict-mode gate

## Verification Log

Every load-bearing claim, with how it was checked. Live checks on the installed binary `ailang` v0.60.2 (built from 33f3009, reporter's version v0.56.1 has the same `regalloc.go:66` code path; this workspace has no Go toolchain, so the installed binary is the verification instrument — same method as the v0_53_1 sibling). Code claims read at repo HEAD 14ad75c8 (v0.60.3).

| # | Claim | Method | Result |
|---|-------|--------|--------|
| V1 | The reporter's repro fails under `--strict-bytecode` at n=40/80/150 with `contiguous block of 3` and at n=300 with `function exceeds 256-register limit`; n=30 passes | Live: generated `reg{30,40,80,150,300}.ail` per the reporter's generator; `ailang run --strict-bytecode --caps IO --entry main` | Confirmed — exact messages reproduced (transcript A) |
| V2 | Exact threshold for the lets+interpolation repro: 36 passes, 37 fails | Live: generated n=31..39 variants, ran each | n ≤ 36 pass; n = 37, 38, 39 fail `BC001` |
| V3 | A single interpolation alone (1 let, same var repeated) fails from 45 holes; `regs` grows ~6 per hole (179@30, 209@35, 239@40) | Live: generated `varB{n}.ail` (n=30..60), ran + `ailang disasm` `regs=` of `varB.main` | Confirmed: 40 passes at regs=239; 45/48/50/60 fail (transcript B) |
| V4 | The `_smoke.ail` shape fails at 15 lets + 3 printlns × 15 holes; 14 lets passes | Live: generated `smoke{n}.ail` for n=12..20, ran each | 12–14 pass; 15, 16, 20 fail `contiguous block of 3` |
| V5 | 300 lets + trivial println fails `function exceeds 256-register limit`; 100 lets passes | Live: `varA300.ail`, `varD100.ail` | Confirmed (reporter's n=300 message class) |
| V6 | The failing repro type-checks clean: it is legal AILANG | Live: `ailang check reg40.ail` | `✓ No errors found!`, exit 0 |
| V7 | The evaluator runs it; non-strict `--bytecode` bridges it with the Phase A warning (no silent fallback remains in the current version — the v0.56.1 reporter's "strict mode disables bridge dispatch" message is now the `BC001` pre-flight) | Live: `ailang run --caps IO --entry main reg40.ail` (prints 40 values), `ailang run --bytecode --caps IO --entry main reg40.ail` (warning + correct output) | Confirmed (transcript A) |
| V8 | `ailang check --strict-bytecode` already reports the failure as a compile-time `BC001` naming function, reason, file:line — the diagnostic half of the reporter's ask exists | Live: `ailang check --strict-bytecode reg40.ail` | `BC001: function reg40.main … at reg40.ail:2` |
| V9 | `reg36` compiles at `regs=250` (one under the ceiling) | Live: `ailang disasm reg36.ail` → `Prototype 0: reg36.main … regs=250` | Confirmed |
| V10 | Lambda captures are copied **by value at `OpClosure` execution**, so releasing a captured binding after the enclosing statement is sound | Code: `internal/vm/vm.go:407-430` (`caps[i] = frame.Regs[...]`; `bytecode.NewClosure(innerProto, caps)`), `lambda.go:29-42` (capture model comment) | Confirmed |
| V11 | The only expression path that reads root-frame bindings is `VarRef` → `locals.lookup`; `GlobalRef` resolves via `funcIdx`, never locals — so a walker without a `GlobalRef` case cannot miss a local read | Code: `expr.go:170-205` (`compileVarRef`), `expr.go:96-127` (`GlobalRef` → `compileTopLevelRef`); walker coverage read in full (`lambda.go:122-274`, covers VarRef, BinOp, UnOp, Call, FieldAccess, RecordLit, RecordUpdate, ListLit, ArrayLit, TupleLit, Cons, ADTConstructor, Lambda, TypeAssert, IfExpr, BuiltinCall, ADTTagEq + all stmt kinds incl. MatchFail, IfStmt, SwitchStmt) | Confirmed |
| V12 | No liveness/last-use analysis exists in the bytecode compiler today; the allocator is documented as intentionally pin-all | Code: `grep -rn "liveness\|last-use\|lastUse" internal/bytecode/` → only the `compiler.go:10` comment "No SSA, no graph coloring, no liveness"; `regalloc.go:1-14` header; `stmt.go:70-86` `compileVarDecl` + its comment "Root-scope bindings … stay pinned for the function's lifetime" | Confirmed (negative-existence, grepped all of `internal/bytecode/`) |
| V13 | Root-frame bindings are only popped for nested scopes; the top-level statement list has no scope push/pop | Code: `grep -rn "locals.push()\|locals.pop()" internal/bytecode/compiler/` → only `control_flow.go` (if branches) and `switch.go` (case bodies); `compileStmts` (`letrec.go:31-61`) has none | Confirmed |
| V14 | Interpolation desugars to a left-associative `concat_String` chain with one `show(...)` per hole, at parse time | Code: `parser_literals.go:137-227` (read in full; fold at `:222-227`, `show` wrap at `:172-204`) | Confirmed |
| V15 | Elaboration binds every non-atomic call argument to a `$tmpN` and nests the Lets; lowering then flattens them to root-level `VarDecl`s, destroying the extent | Code: `internal/elaborate/core.go:349-369` (`normalizeToAtomic`), `core.go:381-392` (`wrapWithBindings` — nested Lets), `internal/gen/lower/block.go:36-47` (`FlattenBlock` flattens; the file header states "This is the heart of the Statement IR lowering") | Confirmed |
| V16 | `compileBuiltinCall` frees arg slots but the result `$tmp` stays pinned; `allocContig(n+1)` is the "contiguous block of 3" in the error | Code: `builtins.go:88-120` (`allocContig(n+1)`, `freeContig(dst+1, n)`), `regalloc.go:60-68` (error site), disasm of `reg3.ail` showing the pinned `$tmp` chain (r5, r7, r8, r10, r11, r13 …) | Confirmed — mechanism matches the measured 6/hole growth of V3 |
| V17 | `inlineCtorTemps` covers constructor positions only, never call/operator positions, and would not by itself fix a long interpolation (the left-nested chain keeps O(n) result registers live within one statement) | Code: `ctor_temps.go:19-25` ("never through a call or operator"), `:76-99` (`absorbTemps` shape); read in full | Confirmed — motivates last-use release over widening it |
| V18 | The only in-repo test asserting a register-allocator failure is the 300-field record literal `BC001` fixture, which must keep failing post-change (genuine liveness) | `grep -rn "register allocator\|contiguous block" internal/bytecode/compiler/*_test.go cmd/ailang/*_test.go internal/runner/*_test.go` → `cmd/ailang/check_strict_bytecode_test.go:103` (300-field record), `internal/runner/strict_report_test.go:14` (report format only) | Confirmed |
| V19 | `AssignStmt` is produced only by the lower pass for compiler temps (`$valTmp`, match/if flags); there is no user-facing assignment statement | `grep -rn "AssignStmt" internal/gen/lower/*.go` (all producers are hoist/rewrite sites, `block.go:237-320`, `match_ifchain.go:74`); `grep -rn "\"var\"\|AssignStmt" internal/parser/*.go` → no parser producer | Confirmed |
| V20 | `elseIfInline` aliases bind in pushed frames with plain `bind`, so root-frame release cannot interfere with chain result registers | Code: `control_flow.go:74-88` (`fc.locals.push()` … `fc.locals.bind(alias.name, outer)` … `fc.locals.pop()`) | Confirmed |
| V21 | All regression fixtures cited exist | `ls` of `internal/bytecode/compiler/` and `cmd/ailang/` — `regalloc_test.go`, `control_flow_test.go`, `collections_test.go`, `call_test.go`, `lambda_test.go`, `letrec_test.go`, `determinism_test.go`, `ctor_temps_test.go`, `evalonly_test.go`, `cmd/ailang/check_strict_bytecode_test.go`, `cmd/ailang/run_bytecode_match_test.go` | All exist |
| V22 | The elaborated `$tmp`/`$valTmp` naming conventions the eligibility rules depend on are the actual fresh-var generators | Code: `internal/elaborate/core.go:348` (`$tmp%d`), `internal/gen/lower/block.go:224` (`$valTmp%d`); eligibility is enforced structurally (RecGroup non-empty, AssignStmt LHS) rather than by name-prefix, so a renamed generator cannot silently break it | Confirmed |

**V-transcript A** (v0.60.2 installed binary; `reg40.ail` from the reporter's generator):
```
$ ailang check reg40.ail
✓ No errors found!
$ ailang check --strict-bytecode reg40.ail
Error: BC001: function reg40.main is evaluator-only under --strict-bytecode
  (register allocator: contiguous block of 3 would exceed 256) at reg40.ail:2
$ ailang run --caps IO --entry main reg40.ail        # evaluator
3.0 3.0 … 3.0   (40 values)                          # exit 0
$ ailang run --strict-bytecode --caps IO --entry main reg40.ail
Error: bytecode execution failed: 1 evaluator-only function(s) reachable
from main; --strict-bytecode wires no evaluator bridge:
  BC001: function reg40.main is evaluator-only … at reg40.ail:2   # exit 1
```

**V-transcript B** (interpolation alone, one live variable):
```
$ ailang disasm varB40.ail | grep -A2 'Prototype 0: varB40.main'
  params=1  regs=239  captures=0  insns=359
$ ailang run --strict-bytecode --caps IO --entry main varB45.ail
BC001 … (register allocator: contiguous block of 3 would exceed 256) …
```

## Future Work

- **Spilling** — if the post-fix census names real programs with >256 genuinely simultaneous live values (the 300-field record class), the v0.11 ~500-LOC assessment is the starting point; `BC001` now names each offender with position, so the evidence will be concrete.
- **Wider-operand instruction format** — core-floor lane; out of scope, unchanged from v0_53_1.
- **Linear-scan allocator on the schedule's interval table** — the pre-pass data structure can evolve into one if register pressure ever needs to be optimal rather than bounded; nothing here forecloses it.
