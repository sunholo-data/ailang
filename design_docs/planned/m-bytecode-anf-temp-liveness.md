# M-BYTECODE-ANF-TEMP-LIVENESS — root-scope ANF temps consumed by calls stay pinned for the function's lifetime: a 60-element list literal of nested calls (and a 240-deep call chain with no list at all) goes evaluator-only under `--strict-bytecode` (third incident in the #1736 register-pressure family)

**Status**: Planned
**Target**: v0.60.4
**Priority**: P0 (strict-VM parity silently excludes exactly the generated test-fixture shapes `--strict-bytecode` exists to cover; field-reported from stapledons-godot R3-UNIVERSE-LAB UL3a)
**Estimated**: 2–3 days (~250 LOC, pure bytecode-compiler change; no VM, no parser, no typechecker, no language semantics, no new opcodes, no new error codes)
**Dependencies**: None. Lands on top of the two shipped fixes in this family (#1736 chunking, #1865 `inlineCtorTemps`) and removes the remaining pinning class they do not cover. The 256-register ceiling itself stays frozen (8-bit ABC encoding; re-freezing is the core-floor lane, per the human decision already recorded in [m-bytecode-strict-256-reg-ceiling.md](v0_53_1/m-bytecode-strict-256-reg-ceiling.md)).

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Today the evaluator runs a function the strict VM refuses (`ailang run` prints `ok`, strict prints BC001) — an engine divergence on identical input. After the fix both engines run the reported shapes and must agree (acceptance is strict-VM == evaluator output). |
| A2: Replayability | 0 | No trace or replay surface change. |
| A3: Effect Legibility | 0 | No effect annotations change. The fix is careful *not* to move evaluation: values are computed at the same statement they always were; only the register holding a dead value is recycled after its final read. |
| A4: Explicit Authority | 0 | No capability surface change. |
| A5: Bounded Verification | 0 | `check --strict-bytecode` (BC001) already reports the failure early; this fix removes the failure itself for the covered shapes. No new gate. |
| A6: Safe Concurrency | 0 | No concurrency change. |
| A7: Machines First | +1 | The failure shape is machine-generated code: test-suite builders that assert many nested-JSON fixtures in one `all([...])` literal, and unrolled `let aN = f(aN-1) in` pipelines. The human workaround (split one test function into several, or drop checks from the parity entry) is exactly the machine-facing friction this removes. |
| A8: Minimal Syntax | +1 | No new syntax; fixes compilation of already-taught, already-type-checking constructs. |
| A9: Cost Visibility | 0 | No resource semantics change. (Register reuse is a compile-time allocation; runtime cost strictly decreases with smaller frames.) |
| A10: Composability | 0 | Internal to one compiler module. |
| A11: Structured Failure | 0 | BC001 stays for the documented residual; no diagnostic surface change. |
| A12: System Boundary | 0 | No boundary change. |

**Net Score: +3** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): removes an engine divergence; the freeing rule is a deterministic function of the statement IR (same input → same allocation)
- [x] A3 (Effects): no evaluation reordering — a temp's value is computed exactly where it always was; only the register is recycled after the last instruction that reads it
- [x] A4 (Authority): no ambient access
- [x] A7 (Machines First): fixes a machine-facing gap

## Problem Statement

**Reported** (coordinator task task-2029e6ee, same class as #1736): a pure function holding a 60-element list literal whose elements are nested calls is evaluator-only under `--strict-bytecode`:

```
ailang run --quiet --bytecode --strict-bytecode --entry big --args-json 0 big60.ail
→ BC001: function big60.big is evaluator-only under --strict-bytecode
  (register allocator: contiguous block of 60 would exceed 256) at big60.ail:9
```

Fails on v0.56.1 (the game's pin) and dev (`v0.60.0-46-g6d28e1ad5a`). Reproduced verbatim on this session's dev build (v0.60.2, commit 33f3009 — Verification Log V1). With 80 or 120 elements the message degrades to `contiguous block of 3 would exceed 256` (V3): the allocator dies on a *call frame* long before it reaches the list.

The repro element shape (each element a small call tree, 60 of them in one `all([...])`):

```ailang
ok(wrap(pair("kN", [one("a", N.0), one("b", N.0), one("c", N.0)])), "kN"),
```

**Where it bites:** `sim/lab_scenario_test.ail` in stapledons-godot (R3-UNIVERSE-LAB UL3a, reported not re-verified here): two test functions holding 19 and 41 checks — each check a nested JSON fixture — cannot join the strict-VM parity entry. They currently run under `ailang test` only, and the parity entry names this report.

### Why the two shipped fixes in this family don't cover it

The #1736 family fixed register pressure twice, each for one consumer kind:

1. **Chunked list literals** (v0.54.13, `listLitChunk = 64`, `internal/bytecode/compiler/collections.go:127`): bounds the *contiguous block one `MAKE_LIST` needs*. It says nothing about how many registers are pinned by the elements' own temporaries.
2. **`inlineCtorTemps`** (v0.56.x, `internal/bytecode/compiler/ctor_temps.go`): absorbs a single-use `$tmpN` into the constructor literal that consumes it — but **only** when (a) the temp's one read sits in a *constructor position* (list/tuple/array/record/ADT literal), and (b) the temp decls form a *contiguous suffix run* of the pending statements immediately before the consumer (`ctor_temps.go:35-46`). A temp read by a **call** marks the substitution bad (`ctor_temps.go:172-176`); any not-yet-substituted run temp read inside an opaque expression also marks it bad.

A nested call element defeats both conditions. Elaboration's ANF (`internal/elaborate/expr_calls.go:96-121`, `normalizeToAtomic` at `core.go:351-374`) binds **every non-atomic subexpression** to a `$tmpN` let — and `IsAtomic` (`internal/core/core.go:598-605`) excludes calls and compound literals. So one element of the repro elaborates to seven temps:

```
$t1..$t3 = one(...)          ← read by the inner ListLit      → constructor position → absorbed ✓
$t4       = [$t1,$t2,$t3]     ← read as an ARG of pair(...)   → call position       → stays pinned ✗
$t5       = pair("kN", $t4)  ← read by wrap(...)             → call position       → stays pinned ✗
$t6       = wrap($t5)         ← read by ok(...)               → call position       → stays pinned ✗
$t7       = ok($t6, "kN")     ← read by the outer ListLit     → constructor position,
                                but the run scan fails: the 60 ok-temps are interleaved
                                with $t4/$t5/$t6 decls, so the only pure suffix run is the
                                last element's ok-temp alone                   → stays pinned ✗
```

Each element therefore leaves **4 permanently pinned registers** (inner list, `pair` result, `wrap` result, `ok` result). Measured on the disassembly of the N=40 variant (V2): `big40.big` uses **regs=204**, with the k0 element pinning exactly r1–r4, k1 pinning r5–r8, …, and the final outer list built as `MOVE r164←r4, MOVE r165←r8, MOVE r166←r12, …` — one MOVE per element from its pinned ok-register, and the *last* element's `ok` call compiled inline because it is the only pure-suffix temp `inlineCtorTemps` could absorb.

Arithmetic: 60 elements × 4 pins + 1 param + call-frame transients ≈ 245 live when the outer list's `allocContig(60)` fires → 245 + 60 > 256 → the reported `contiguous block of 60`. At N ≥ 80 the pin count alone (~320) crosses the ceiling mid-body, and the allocator dies on the next arity-2 call frame → `contiguous block of 3` (V3).

**Why the run-based absorb can't simply be extended instead:** substituting a temp into a *call argument* position moves its evaluation from the temp-declaration statement into the consumer's argument materialization — changing when effects run relative to the callee materialization and other arguments. The constructor-position restriction in `ctor_temps.go` exists precisely because constructor slots are compiled in element order (evaluation order provably unchanged); proving the same for every call shape re-derives an argument-ordering proof per consumer, and the contiguous-suffix-run algorithm remains structurally unable to handle interleaved sibling temps no matter how far the substitution is widened. The systemic fix is not another absorb case — it is making the register itself stop being pinned once the value is dead.

### The class is bigger than list literals (systemic audit, measured)

The pinning defect is in `compileVarDecl` (`internal/bytecode/compiler/stmt.go:44-66`): every root-scope `let` binds its result as a pinned local for the **function's lifetime** (`bindScoped`; the root scope frame is never popped — `regalloc.go` `scopeStack.pop()` panics at the root, and push/pop is only called for if/switch arm scopes, `control_flow.go:59-88`, `switch.go:78-113`). ANF temps and user lets alike. So any function whose *sequential* single-use bindings exceed ~250 registers fails — no list required:

- **chain240**: a plain pipeline of 240 `let aN = f(aN-1) in` bindings, no literal anywhere → `BC001: … contiguous block of 2 would exceed 256` (V7). Every `aN` is read exactly once, by the very next statement — yet all 240 stay pinned.
- **shallow60**: a 60-element list of *single* (non-nested) calls → runs strict today (V6), because the whole element-temp run is a pure suffix absorbed by `inlineCtorTemps`. This is why #1865's own 300-element test passes while the nested-call shape fails: **nesting** is what interleaves the temps and breaks the run.

One fix covers all of these: free the register at the binding's **last read**.

## Goals

**Primary Goal:** Make register lifetime track value liveness — a root-scope binding's register is freed after the last statement that reads it — so sequential single-use lets and nested-call list elements cost (number of simultaneously-live values + expression depth) registers, not (bindings × anything).

**Success Metrics:**
- big60 (the reported repro) compiles and runs under `--bytecode --strict-bytecode`, printing the evaluator's exact output (`ok`); N=80 and N=120 likewise.
- chain240 (the no-literal pipeline) compiles and runs strict with the evaluator's exact output.
- `ailang disasm` register count for the N=40 variant drops from the measured 204 to ≤ 80 (prediction ≈ 50: 40 live ok-temps + depth + transients — see Solution Design).
- All existing strict fixtures that must still fail keep failing with today's reason (e.g. the 300-field record literal in `cmd/ailang/check_strict_bytecode_test.go` — constant fields have no temps, and the 300-contiguous block is untouched).
- stapledons-godot `sim/lab_scenario_test.ail`'s two UL3a test functions (19 and 41 checks) can rejoin the strict parity entry (external; named in the CHANGELOG note, verified upstream).

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Fix the **register allocator/compiler** (last-read freeing at statement granularity); do not widen `inlineCtorTemps` substitution into call positions | Substituting into call-arg positions reorders effect timing relative to argument materialization; the run algorithm also cannot handle interleaved temps. Freeing a dead register changes no evaluation at all | agent | design | med |
| Liveness applies to **all root-scope single-assignment bindings** (ANF `$tmpN` *and* user `let`), not `$tmp`-only | chain240 proves user lets carry the same defect; a `let` is immutable (no user assignment surface exists — AssignStmt targets are compiler-hoisted `$valTmpN` only, V18) | agent | design | low |
| **Statement granularity**, not instruction/read-position granularity | Statement-level frees are trivially sound (all reads of a top-level statement's values are emitted within it) and deterministic; read-position granularity (free per element-slot MOVE, needed only to make >~240-computed-element literals chunk-bounded) is deferred until a real case exists | agent | design | low |
| Exclusions: AssignStmt targets (`$valTmpN`), letrec group members, params; reads inside a Lambda body attribute to the statement containing the lambda's creation | All three have later or non-local consumers; captures are copied by value at `OpClosure` (V13) so freeing after the creating statement is exact, and the conservative attribution needs no extra proof | agent | design | low |
| No VM, encoding, or ceiling change; no new diagnostic codes | The 8-bit encoding is frozen (family doc's human decision); BC001 already reports the residual early | human (prior, still binding) | design | high (avoided) |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] Fix lane: bytecode compiler only (`internal/bytecode/compiler/`); no VM, parser, typechecker, stdlib, or CLI surface change.
- [x] Free at statement granularity; read-position granularity is Future Work with a named trigger.
- [x] Scope: all root-scope bindings in the enclosing funcCompiler (functions, lambdas, letrec-bound lambda bodies — the three sites that already run `inlineBodyCtorTemps`), with the exclusion set above.
- [x] Determinism: free order is binding-declaration order (a recorded slice), never map iteration order.

## Conflict Surface

This design touches `internal/bytecode/compiler/` (codegen). Required enumeration:

1. **Syntactic/semantic positions extended:** none. No parser or typechecker change; all affected constructs already compile and type-check.
2. **Codegen positions changed:** exactly one — the *lifetime* of registers returned by `compileVarDecl` (`stmt.go:44-66`). Today every root-scope binding's register is pinned until function end; after, it is freed (and unbound) after the top-level statement containing its last read. `compileStmts` (`letrec.go:31`) gains a post-statement free step fed by a pre-computed last-read map. No new instructions: freeing is allocation bookkeeping only.
3. **Other consumers of the same machinery (audited):**
   - `scopeStack.pop()` (`regalloc.go`) frees *nested-scope* pinned regs at arm exit — untouched; the new pass only frees root-frame bindings, so no double-free (pop only touches its own frame's `pinnedRegs`, and the pass also unbinds the name it frees).
   - `isPinned` (`expr.go:101-110`) walks all frames' bindings — a freed-and-unbound name no longer matches, which is the desired fail-loud behavior for any analysis bug (stale read → `compiler: unbound variable`).
   - `inlineCtorTemps` runs **before** compilation (`compiler.go:331`, `lambda.go:64`, `letrec.go:174`); the liveness pre-pass runs on its *output*, so absorbed temps simply never appear.
   - `elseIfInline` (`control_flow.go`) aliases internal `$tmp` cond temps — these live in nested frames, not root scope, and are untouched.
   - Call frames (`call.go:36`, `allocContig(arity+1)`), list chunking (`collections.go:127`), record/ADT/tuple blocks: unchanged; they merely find a *less-fragmented* free list (freed temp regs are LIFO-reused exactly like existing `freeTemp` calls).
   - `highWater()`/`NumRegs`: strictly smaller or equal; the VM frame shrinks. `Image.ValidatePrototype` bounds are unaffected.
4. **Parser/typechecker disambiguation:** N/A.
5. **Programs that MUST still work post-change (fixtures, all verified to exist):**
   - `internal/bytecode/compiler/collections_test.go`, `control_flow_test.go`, `regalloc_test.go`, `lambda_test.go`, `letrec_test.go`, `determinism_test.go` — compiler tests assert *values* (e.g. `TestCompile_ListLit_ThreeInts` runs the compiled function), not register numbers (V16).
   - `cmd/ailang/bytecode_long_list_literal_test.go` — `TestLongListLiteralRunsOnStrictVM`, `TestLongElseIfChainRunsOnStrictVM`, `TestLongComputedListLiteralRunsOnStrictVM` (300 shallow computed elements — keeps passing; nesting is the new case).
   - `cmd/ailang/check_strict_bytecode_test.go` — the 300-field record fixture must **keep failing** with `contiguous block of 300 would exceed 256` (constant fields → no temps → liveness is a no-op there).
   - `cmd/ailang/run_bytecode_match_test.go`, `internal/testing/engine_parity_test.go` — evaluator/VM parity corpus.
   - `internal/bytecode/compiler/determinism_test.go` — same input must still produce byte-identical images across runs (free order is a recorded slice, not map iteration).
6. **What deliberately changes:**
   - Register assignment in nearly all compiled functions shifts (dead temps' registers get reused by later statements). No goldens pin compiler disassembly (`goldens/` holds 4 linker/import JSON fixtures only; `disasm_test.go` constructs prototypes by hand — V16). `ailang disasm` output for existing modules changes — accepted, fail-loud direction, same class as the #1736/#18690 chunking change.
   - `NumRegs` shrinks for functions with sequential lets; strict runs that previously trapped mid-body now run. No message-text consumers exist in-repo for the *changed* cases (BC001 remains for the untouched classes, asserted by `check_strict_bytecode_test.go`).

## Solution Design

### Overview

One new compiler pass plus one hook:

1. **Liveness pre-pass** (`tempLiveness`): over a function's post-`inlineCtorTemps` top-level statement list (plus the trailing return expression), compute for every root-scope binding the index of the top-level statement containing its **last read**, or a sentinel if it must stay pinned.
2. **Free hook** in `compileStmts`: after compiling top-level statement `i`, release every binding whose last-read index is `i`: `regs.freeTemp(reg)` + unbind the name from the root scope frame.

Values are computed exactly where they always were; evaluation order, effect order, and instruction semantics are untouched. Only the register holding a now-dead value returns to the free list — the same `freeTemp` every expression compiler already calls for its own transient results.

### Architecture

**Components:**

1. **`tempLiveness` analysis** (new file `internal/bytecode/compiler/liveness.go`, ~90 LOC):
   - Input: `body []stmt.Stmt` (top-level, post-inline), `ret stmt.Expr`.
   - A visitor in the `freeVarVisitor` style (`lambda.go:121-267` already counts reads per name and descends into `IfStmt`/`SwitchStmt` arms and Lambda bodies — reuse the pattern, or extend the visitor with a mode that records `map[name]int` last-statement-index instead of `counts`).
   - For each top-level statement index `i`, walk the statement (including nested arm statements and expressions, and Lambda bodies) recording `lastRead[name] = max(lastRead[name], i)`. The trailing return expression is index `len(body)`.
   - **Exclusions** (sentinel = never free):
     - any name that is an `AssignStmt` target anywhere (`$valTmpN` hoisting writes these from branch arms — `lower/block.go:225-250`, `match_ifchain.go:74`);
     - any `VarDecl` with a non-empty `RecGroup`;
     - parameters (bound before analysis, not by `VarDecl`);
     - any name whose *declaration* statement index is ≥ its last-read index (read-before-decl can't happen, but the pass asserts rather than assumes);
     - **shadowing hygiene**: only bindings declared in the root statement list participate. A same-named binding inside an arm is a different scope-frame slot (arm scopes pop today) — the analysis keys on root-declared names and ignores arm-local decls of the same name by tracking the root declaration order.
   - Output: an ordered slice `(name, lastReadIndex)` in declaration order (determinism), plus the set of never-free names.
   - Lambda capture attribution: a read of a root binding inside a Lambda body attributes to the index of the top-level statement containing the lambda expression. Sound because `OpClosure` **copies** capture values at creation (`internal/vm/vm.go:407-431`: `caps[i] = frame.Regs[...]` → `NewClosure(proto, caps)` — V13), so the register is dead once the creating statement completes.

2. **Free hook in `compileStmts`** (`letrec.go:31`, ~25 LOC):
   - `compile()` (`compiler.go:331`), `compileLambda` (`lambda.go:64`) and the letrec path (`letrec.go:174`) compute `live := tempLiveness(body, ret)` right where they already call `inlineBodyCtorTemps`.
   - After each top-level statement compiles, free that statement's dead bindings: `fc.regs.freeTemp(reg)` and remove the name from the root scope frame (defense-in-depth: a stale read then fails as `compiler: unbound variable` instead of silently reading a reused register — no-silent-fallbacks).
   - Bindings whose last-read index is `len(body)` (the return expression) are freed after `compileReturnExpr`; that free is a no-op in effect but keeps the accounting uniform.
   - Freeing happens in declaration order for reproducibility of the free list (the LIFO free list order feeds later `allocTemp` reuse).

**Why this yields the expected budget:** in big60, each element's `$t4/$t5/$t6` (inner list, pair, wrap) are read by the *next* statement — freed immediately; each element's `$t7` (ok result) is read by the outer-list statement — 60 live at the peak, plus ~6 transient/depth. Predicted `NumRegs` ≈ 70 (vs failure today); N=40 measured 204 → predicted ≈ 50. In chain240, each `aN` is freed after the next statement: constant ~4 registers. The pressure becomes **(simultaneously live values) + (expression depth)**, exactly the budget the report asks for — the product case is gone because nothing accumulates across siblings.

### Implementation Plan

**Phase 1: analysis + hook** (~1 day)
- [ ] `liveness.go`: `tempLiveness` visitor (last-read index per root binding; exclusions; declaration-ordered output) + unit tests on hand-built stmt IR: single-use chain, use-in-both-branches, use-in-lambda, assign-target, letrec member, shadowed arm-local name.
- [ ] `compileStmts` free hook; wire at `compiler.go:331`, `lambda.go:64`, `letrec.go:174`.
- [ ] Unit test: register high-water of a K-deep sequential let chain is constant in K (compile, assert `NumRegs` plateaus — e.g. K=8 vs K=200 within a small bound).

**Phase 2: acceptance fixtures (the report's shapes)** (~0.5 day)
- [ ] `cmd/ailang/bytecode_long_list_literal_test.go`: add `TestNestedCallListLiteralRunsOnStrictVM` — the big60 repro generator (N=40/60/80/120; strict VM output == evaluator output, exit 0); add `TestSequentialLetChainRunsOnStrictVM` (chain240 shape, K=240/300).
- [ ] Confirm `ailang check --strict-bytecode` exits 0 on both new fixtures.
- [ ] Confirm the 300-field record fixture in `check_strict_bytecode_test.go` still fails with `contiguous block of 300` (liveness is a no-op without temps).

**Phase 3: corpus + docs** (~0.5 day)
- [ ] Re-run the Phase A census style sweep (`check --strict-bytecode` over `std/`, `examples/`, `goldens/`): expect **zero new** BC001s and strictly-fewer-or-equal NumRegs everywhere; record before/after `disasm` NumRegs for a sample (big40: 204 → bank the number).
- [ ] `make test`, `make fmt`, `make lint`, `make check-boundaries` (no boundary changes expected).
- [ ] Docs: `docs/docs/reference/limitations.md` — update the strict/bytecode section with the new capability and the **residual limit** (see Non-Goals); CHANGELOG entry noting the UL3a parity-entry functions can rejoin.

### Files to Modify/Create

**New files:**
- `internal/bytecode/compiler/liveness.go` — the `tempLiveness` analysis (~90 LOC)
- `internal/bytecode/compiler/liveness_test.go` — IR-level unit tests (~150 LOC)

**Modified files:**
- `internal/bytecode/compiler/letrec.go` — `compileStmts` free hook (~20 LOC)
- `internal/bytecode/compiler/compiler.go` — compute liveness in `compile()` (~8 LOC)
- `internal/bytecode/compiler/lambda.go`, `letrec.go` — same wiring for lambda/letrec bodies (~8 LOC each)
- `cmd/ailang/bytecode_long_list_literal_test.go` — the two new strict fixtures (~80 LOC)
- `docs/docs/reference/limitations.md` — capability + residual limit (~10 LOC)
- `CHANGELOG.md` / `changelogs/v0.32-current.md` — entry (~10 LOC)

## Examples

### Example 1: the reported repro

**Before** (v0.56.1 pin and dev, measured V1):
```
$ ailang run --quiet --bytecode --strict-bytecode --entry big --args-json 0 big60.ail
Error: … BC001: function big60.big is evaluator-only under --strict-bytecode
  (register allocator: contiguous block of 60 would exceed 256) at big60.ail:9
```

**After** (acceptance):
```
$ ailang run --quiet --bytecode --strict-bytecode --entry big --args-json 0 big60.ail
ok
```

### Example 2: register budget, before/after (N=40, measured via `ailang disasm`)

**Before**: `big40.big params=1 regs=204` — four pinned registers per element, all live until function end; the outer list assembles via 40 `MOVE`s from the pinned ok-registers (V2).

**After** (predicted, banked as acceptance bound): `regs ≤ 80` — per-element temps recycled as consumed; only the 40 element results destined for the list stay live until the list statement.

## Success Criteria

- [ ] The reported repro (60 nested-call elements) runs under `--bytecode --strict-bytecode` with the evaluator's exact output; 80- and 120-element variants too (new fixture `TestNestedCallListLiteralRunsOnStrictVM`).
- [ ] A 240-deep sequential `let` pipeline (no literal) runs strict with the evaluator's exact output (`TestSequentialLetChainRunsOnStrictVM`).
- [ ] `ailang check --strict-bytecode` exits 0 on both new fixtures; the 300-field record fixture in `check_strict_bytecode_test.go` still fails with today's `contiguous block of 300` reason.
- [ ] `NumRegs(big40.big) ≤ 80` (from 204) — assert via disasm in the fixture.
- [ ] Determinism: two compiles of the same file produce byte-identical images (existing determinism tests + the free-order rule).
- [ ] All existing tests pass (`make test`); `make check-boundaries` clean; no new opcodes, error codes, flags, or env vars.
- [ ] Documentation updated (limitations.md residual-limit note; CHANGELOG names the UL3a parity-entry rejoin).

## Testing Strategy

**Unit tests (`liveness_test.go`):** last-read index correctness for: single-use chain; read in then-arm only; read in both arms (attributed to the enclosing statement); read inside a lambda (attributed to the creating statement, plus a later direct read taking max); `$valTmpN` assign-target excluded; RecGroup excluded; shadowing (root binding freed while an arm-local same-named binding stays scoped); determinism of free order.

**Integration tests:** the two new strict fixtures (Phase 2), each asserting interpreter == strict-VM output; existing parity corpus (`run_bytecode_match_test.go`, `engine_parity_test.go`) must stay green — it is the guard that early freeing never changes observable behavior.

**Manual/verification:** `ailang disasm` before/after on big40 and one std module (e.g. a `std/list` function with sequential lets) to bank NumRegs deltas; re-run the census sweep.

## Deferred Decisions

The following are intentionally left open for the implementer:

- **Visitor reuse vs. new walk** — extend `freeVarVisitor` with a last-index mode, or write a separate small visitor. Either is fine; the decision is the implementer's, but the output must be declaration-ordered.
- **Exact sentinel encoding** for never-free names (skip-list, index −1, or separate set) — implementer's choice.
- **Freeing at `len(body)` (return-expression reads) before or after `compileReturnExpr`** — observably equivalent; pick the simpler wiring.
- **Whether the analysis also covers `$valTmpN`-style assignables via a write-aware extension** — explicitly *not* required for this fix (no reported pressure); only revisit with a real case.

## Non-Goals

**Not attempted in this feature:**
- **Raising the 256-register ceiling or widening the 8-bit encoding** — frozen; core-floor lane (standing human decision from the family doc).
- **Chunked tuple/record literals** — the 300-field record stays evaluator-only with a BC001 (family doc already scoped this out).
- **Read-position (per-instruction) freeing** — would make a single literal with >~240 *computed* elements chunk-bounded instead of element-bounded. No reported case needs it (UL3a is 41); Future Work with a trigger.
- **Widening `inlineCtorTemps` into call-argument substitution** — rejected as the primary fix (effect-order proof burden, and the run algorithm cannot handle interleaved temps); `inlineCtorTemps` stays exactly as shipped.
- **Residual limit documentation:** after this fix, a function still goes evaluator-only when its *simultaneously live* set exceeds ~250 — e.g. one list literal with ~245+ computed elements (all element results live until the literal's statement), or a record/tuple literal needing a >256 contiguous block. That limit is inherent to the frozen encoding, already reported early by BC001, and is documented in limitations.md rather than fixed here.

## Timeline

**Week 1** (~2.5 days total, single sprint-sized change):
- Phase 1: analysis + hook + unit tests (day 1)
- Phase 2: acceptance fixtures, strict/parity verification (day 2)
- Phase 3: census sweep, docs, CHANGELOG (day 3, half day buffer)

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Analysis bug frees a register still read later → wrong value or unbound-variable compile error | High | Freeing also **unbinds** the name: a stale read fails loudly at compile time (`compiler: unbound variable`) instead of silently reading a reused register. Exclusion set is conservative (assign-targets, letrec, params, captures-at-creation). The whole parity corpus is the behavioral guard. |
| Read inside a not-taken arm frees "too early" | None (by construction) | Freeing is compile-time bookkeeping; no *emitted* instruction after the free reads the register (liveness over emission order). Non-executed arms never read it at runtime. |
| Register shifts break hidden disasm consumers | Med | Verified: no goldens or tests pin compiler register assignment (V16); disasm text changes were already accepted twice in this family (#1736, #18690-era chunking). |
| Double-free (pass frees a reg the scope pop also frees) | Med | The pass frees only root-frame bindings and unbinds them; `pop()` releases only its own frame's `pinnedRegs` (code-verified, V9). |
| Nondeterministic free order → image instability | High | Free order is the recorded declaration-order slice, never map iteration; determinism tests re-run. |
| Lambda capturing a temp freed before creation | High | Captures copy values at `OpClosure` (V13); a capture read attributes to the creating statement, so the free is strictly after the copy. Conservative fallback if the sprint finds a counter-case: sentinel any lambda-read name to never-free (costs nothing for the reported shapes). |

## Related Documents

**Same family (prior incidents, both shipped):**
- [m-bytecode-strict-256-reg-ceiling.md](v0_53_1/m-bytecode-strict-256-reg-ceiling.md) — the #1736/#1865 design doc: Phase A BC001 reporting, Phase B1 list chunking (K=64), Phase B2 else-if chain result reuse. This doc is its third incident: the *residual* class those phases explicitly did not cover (call-consumed and interleaved temps). No overlap: the family doc's phases are landed; nothing there covers register lifetime.
- [m-bytecode-strict-256-reg-ceiling-census.md](v0_53_1/m-bytecode-strict-256-reg-ceiling-census.md) — the EvalOnly census (0 register-ceiling entries across 538 checked files as of 2026-11-09). The new shapes are AI/user-code shapes, not std shapes; Phase 3 re-runs the census to confirm no regression.

**Distinct-but-adjacent:** M-VM-COVERAGE (fallback causes; this fix removes one more), `docs/docs/reference/limitations.md` (fallback section updated by Phase 3).

## Verification Log

Every load-bearing claim, verified this session on dev build v0.60.2 (commit 33f3009, `/usr/local/bin/ailang`; repo at `d5580243`). Repro scripts generated with the report's exact element shape.

| # | Claim | Verification |
|---|-------|--------------|
| V1 | The reported repro fails on current dev with the reported message | `ailang run --quiet --bytecode --strict-bytecode --entry big --args-json 0 big60.ail` → `BC001: function big60.big is evaluator-only under --strict-bytecode (register allocator: contiguous block of 60 would exceed 256) at big60.ail:9`. Exit 1. |
| V2 | Each element pins 4 registers for the function's lifetime; N=40 fits today | `ailang disasm big40.ail` → `big40.big params=1 regs=204 insns=1449`; k0 pins r1 (MAKE_LIST), r2 (pair), r3 (wrap), r4 (ok); k1 pins r5–r8; …; the outer list ends with `MOVE r164←r4, MOVE r165←r8, MOVE r166←r12, …` — one MOVE per pinned ok-register, and the last element (k39) has its `ok` call compiled inline (the only pure-suffix temp `inlineCtorTemps` absorbs). |
| V3 | N=80/120 die on a call frame instead | Both → `BC001 … (register allocator: contiguous block of 3 would exceed 256) at bigN.ail:9`. |
| V4 | The check-time gate reports it early | `ailang check --strict-bytecode big60.ail` → `Error: BC001: function big60.big is evaluator-only … at big60.ail:9`, exit ≠ 0 (Phase A of the family doc is live; no new gate needed). |
| V5 | The constructs are legal AILANG; the evaluator runs them | `ailang check bigN.ail` → `✓ No errors found!` for N=40/60/80/120; evaluator `ailang run --quiet --entry big --args-json 0 big60.ail` and big120 → `ok`. |
| V6 | Non-nested call elements already work (ctor_temps covers them) — nesting is the trigger | `shallow60.ail` (60 elements, each a single `key(one(...))` call) → strict run prints `ok`. Matches `TestLongComputedListLiteralRunsOnStrictVM` (300 shallow computed elements, `cmd/ailang/bytecode_long_list_literal_test.go:99`). |
| V7 | The class extends beyond list literals; user `let`s are affected too | `chain240.ail` (240 sequential `let aN = f(aN-1) in`, no literal) → strict run: `BC001: function chain240.big is evaluator-only under --strict-bytecode (register allocator: contiguous block of 2 would exceed 256) at chain240.ail:3`. |
| V8 | Elaboration binds every non-atomic subexpression to a `$tmpN` (calls and compound literals are non-atomic) | `internal/elaborate/expr_calls.go:96-121` (every call arg `normalizeToAtomic`); `internal/elaborate/core.go:351-374` (`normalizeToAtomic` binds non-atomic results to `freshVar()` = `$tmpN`, `core.go:347-352`); `internal/core/core.go:598-605` (`IsAtomic` = Var/Lit/Lambda/DictRef/VarGlobal only). |
| V9 | Root-scope bindings are pinned for the function's lifetime; nested scopes pop, root never | `internal/bytecode/compiler/stmt.go:44-66` (`compileVarDecl` → `compileExprToFresh` + `bindScoped`; comment: "Root-scope bindings (function-level lets) are never popped"); `regalloc.go` `scopeStack.pop()` panics at `len(frames) <= 1`; `push()/pop()` called only at `control_flow.go:59,63,74,88` and `switch.go:78-113` (arm scopes). |
| V10 | Why ctor_temps can't absorb these temps | `ctor_temps.go:35-46` (absorb requires a contiguous *suffix* run of pending `$tmp` decls); `ctor_temps.go:137-176` (`ctorSubst` descends only the constructor spine — ListLit/ArrayLit/TupleLit/ADTConstructor/RecordLit; any other expr reading a not-yet-used run temp sets `bad`). Consequence measured in V2: only the last element's ok-temp is absorbed. |
| V11 | List chunking bounds only the MAKE_LIST block, at 64 | `internal/bytecode/compiler/collections.go:90-120` (`compileListLit` → `allocContig(n)`), `:127-133` (`listLitChunk = 64`), `:136-165` (chunked path joins via `_concat_List`; element order preserved). |
| V12 | **Negative existence:** the bytecode compiler has no liveness analysis today | `internal/bytecode/compiler/compiler.go:10` — package comment: "No SSA, no graph coloring, no liveness". `grep -rn "liveness\|lastRead\|last_read\|deadTemp" internal/bytecode/` → only that comment. |
| V13 | **Capture semantics:** closure captures copy values at creation, so freeing after the creating statement is sound | `internal/bytecode/compiler/lambda.go:11-22` (capture model); `internal/vm/vm.go:407-431` (`OpClosure`: `caps[i] = frame.Regs[...]` → `NewClosure(innerProto, caps)`). |
| V14 | A read-counting visitor with Lambda/arm descent already exists (reuse pattern) | `internal/bytecode/compiler/lambda.go:121-139` (`freeVarVisitor`, `counts map[string]int`), `:158-267` (full `visitStmt`/`visitExpr` incl. `IfStmt`, `SwitchStmt`, Lambda-descent with free-var recording). |
| V15 | No new error code needed; BC001 exists and stays for the residual | `internal/errors/codes.go:123-128` (`BC001`); no code changes in this design. |
| V16 | **Negative existence:** nothing pins compiler register assignment | `goldens/` = 4 JSON linker/import fixtures (`import_conflict.json`, `imports_basic_success.json`, `lnk_unresolved_module.json`, `lnk_unresolved_symbol.json`); `internal/bytecode/disasm_test.go` builds prototypes by hand; compiler tests (`collections_test.go:12-40` etc.) assert *values* of run functions; `grep -rn "regs=\|NumRegs" --include="*_test.go"` → only hand-built disasm/image tests. |
| V17 | Fixtures that must keep failing, keep failing | `cmd/ailang/check_strict_bytecode_test.go:100-105` asserts `contiguous block of 300 would exceed 256` for a **300-field record literal** (`strictBytecodeFixtures`, fields are constants — no temps; the 300-contiguous MAKE_RECORD block is untouched by liveness). |
| V18 | AssignStmt targets are compiler-hoisted `$valTmpN` — liveness must exclude them | `internal/gen/lower/block.go:225-250` (`$valTmpN` hoisting writes `AssignStmt` from branch arms); `internal/gen/lower/match_ifchain.go:74`. No user assignment surface exists (no `var`/assignment production in `internal/parser/` — grep: only comments). |
| V19 | The 256 ceiling is frozen and was decided not to be raised | `internal/bytecode/compiler/regalloc.go:66-67` (ceiling error), `:145-152` (`bump` ceiling); family doc High-Impact Decisions: "do not raise the 256-register ceiling or widen instruction operands … a VM re-freeze (core-floor lane)" — chosen by: human. |
| V20 | Where it bites (UL3a) | Reported by the task: `sim/lab_scenario_test.ail` in stapledons-godot, two functions (19 and 41 checks), currently excluded from the strict parity entry. **Report-not-reverified**: the repo is external; acceptance for this doc is the in-repro fixtures (V1/V7 shapes), and the upstream rejoin is named in the CHANGELOG note. |

## References

- [Design Axioms](/docs/references/axioms) - The 12 non-negotiable principles
- [m-bytecode-strict-256-reg-ceiling.md](v0_53_1/m-bytecode-strict-256-reg-ceiling.md) — the #1736/#1865 family doc (Phase A BC001, Phase B1 chunking, Phase B2 chain reuse)
- [m-bytecode-strict-256-reg-ceiling-census.md](v0_53_1/m-bytecode-strict-256-reg-ceiling-census.md) — EvalOnly census baseline
- `changelogs/v0.32-current.md` — #1736 (v0.54.13) and #1865 (ctor_temps) entries
- Issue #1736 (closed), #1865 (closed); this report: coordinator task task-2029e6ee

## Future Work

- **Read-position freeing** (per-instruction granularity): free each element temp at its slot `MOVE` inside a chunked literal compile, making a single literal of *any* length of computed elements chunk-bounded (≤ 64 + depth) instead of element-bounded (~245). Trigger: a real function with >240 computed elements in one literal that strict mode must run. Reuses the same `tempLiveness` data with finer attribution.
- **Write-aware liveness for `$valTmpN`-style hoisted mutables** — only if a reported pressure case appears.
- Tuple/record literal chunking — separate decision, unchanged from the family doc's Non-Goals.

---

**Document created**: 2026-11-12
**Last updated**: 2026-11-12
