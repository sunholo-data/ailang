# M-VM-COVERAGE: Close the four gaps that keep model-written AILANG off the bytecode VM

**Status**: Planned
**Target**: v0.55.0
**Priority**: P1 overall. Phase 1b (silent wrong answer under `--bytecode`) is P0.
**Estimated**: 6–8 days (Phases 1–3); Phase 4 is a human decision, not an estimate
**Dependencies**: None. Interacts with [#1956](https://github.com/sunholo-data/ailang/issues/1956) (`--bytecode --verify-contracts` silently checks nothing), which gates Phase 4.
**Planner-Lane**: opus-required (VM compiler name resolution, lowering, the interop bridge)

## Routing Lane (per design_docs/PROGRAM.md §4)

**Lane: AILANG fix.** PROGRAM.md §4 row 1 routes "bad/unfixable error, missing builtin, dialect
trap, type-system gap" to an AILANG fix with a design doc plus a regression test. This is a runtime
gap in the AILANG toolchain, not motoko agent-loop behaviour, so an extension cannot fix it.
Nothing here touches the frozen motoko core.

## Quorum Triggers

- **Trigger 1 (design-freeze items): FIRES.** F1 and F2 (make the VM the default for `ailang run`
  and for the eval harness) are human decisions.
- **Trigger 3 (cost/KPI semantics or banked-data schema): FIRES, for Phase 4 only.** Changing the
  harness engine shifts time-limited pass rates, and banked rows need an `engine` field so that
  baselines before and after the change can be told apart.
- Trigger 2 (overrides shared machinery): does not fire. Every Conflict Surface row below *reuses*
  an existing mechanism (lexical locals, the loader's synthetic prelude import, the existing ADT
  type table).
- Trigger 4 (external systems): does not fire. Every premise can be re-checked in-repo.

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Removes a case where the same program gives different output on the evaluator and the VM (Phase 1b: 10 vs 1000) |
| A2: Replayability | 0 | Trace semantics unchanged |
| A3: Effect Legibility | +1 | Stops a run from failing *after* effects have committed: the bridge-ADT and module-level-value cases crash after printing, and non-strict mode then refuses replay (#1545) |
| A4: Explicit Authority | 0 | No capability change |
| A5: Bounded Verification | 0 | Type checking unchanged |
| A6: Safe Concurrency | 0 | No concurrency change |
| A7: Machines First | +1 | Model-written programs get the engine that measured 21× faster without rewriting their code. The coverage metric is machine-checkable |
| A8: Minimal Syntax | 0 | No syntax change |
| A9: Cost Visibility | +1 | The parity script reports, per file, whether it ran natively or silently fell back, which is invisible today |
| A10: Composability | 0 | No module-system change |
| A11: Structured Failure | +1 | Any construct the VM cannot handle either compiles to a clean, pre-effect fallback or fails at compile time, never as a mid-run type confusion such as `ADD: type mismatch Int vs Closure` |
| A12: System Boundary | 0 | No boundary change |

**Net Score: +5** → **Decision: Move forward.** No −1 on A1/A3/A4/A7.

## Problem Statement

A competitive-programming spike (2026-10-28; 20 ICPC/CCPC 2024 problems, Haiku 4.5 agent mode)
found AILANG 10–1000× slower than Python on correct solutions. It also found that **every eval
speed number we have is evaluator speed**: the harness runs `ailang run --entry main --quiet
--relax-modules …` with no `--bytecode` (`internal/eval_harness/runner.go:277`). Re-timing the
same banked solutions on the VM:

| Solution (same code) | Input | Evaluator | `--bytecode` | Python | Why |
|---|---|---|---|---|---|
| cp_60212 | 504KB | 23.8s | **1.14s** | 118s, wrong output | Runs natively: 21× faster |
| cp_60212 with only `println` dropped from `import std/io` | 504KB | 24.1s | 23.8s | — | Gap G2: `main` falls back to the evaluator |
| cp_60198 | 141KB | 19.05s | 19.08s | 0.04s | Gap G1: a local `letrec` falls back |
| cp_60159 | 17KB | >60s | **crash** | 0.05s | Gaps G1 + G4: falls back, then hits a bridge ADT error after printing |
| cp_60164 | 15KB | >60s | >60s | 0.11s (difference array) | Runs natively. The algorithm is O(n³); the VM cannot fix that |
| cp_60181 | 32KB | >120s | >120s | 0.1s | Runs natively, same as above |

The tracing tier makes no difference here. `AILANG_TRACE=standard` and `off` are within 2% on both
engines for cp_60212 and cp_60198 (V3).

**Do not oversell this.** The VM is a constant-factor lever. It does not rescue an O(n³)
brute force (cp_60164, cp_60181). It is also about 1.8× *slower* than the evaluator on copy-bound
cons building (docs/LIMITATIONS.md, "`--bytecode` is ~1.8× slower…"). The algorithmic gaps
(no O(1)/O(log n) update structure, no BigInt) are separate roadmap items.

### What keeps programs off the VM: measured over two corpora

**Corpus A: 41 banked model AILANG solutions** (20 problems, Haiku 4.5, 1–3 trials each), each run
on its smallest hidden test under evaluator, `--bytecode` and `--strict-bytecode` (V6):

| Outcome | Count |
|---|---|
| NATIVE (strict VM passes, same stdout) | 31 |
| SILENT_FALLBACK (strict fails, `--bytecode` falls back and passes) | 6 |
| BYTECODE_CRASH (`--bytecode` fails although the evaluator passes) | 4 |

**All 10 non-native solutions use a local `letrec`, and no native solution does** (V7). The
teaching prompt shows `letrec` 12 times (V8).

**Corpus B: `examples/runnable/`, 260 files** (V9): 205 pass on the evaluator. Of those, 142 run
natively, 61 silently fall back, 1 crashes and 1 diverges. The divergence is `xml_walk_perf.ail`,
which prints timing, so it is benign. Reasons for falling back:

| Reason (`--strict-bytecode` EvalReason) | Files |
|---|---|
| `call to unbound name "println"`: prelude `println` without an import (G2) | 47 |
| `effectful builtin … not yet wired (Phase 2E)` (FS, clock sleep, tar, game) | 7 |
| `unknown ADT type "…"` (contract-module ADTs) | 3 |
| Local `letrec` (G1), `stmt.ArrayLit`, opaque `__html_parse` | 1 each |
| VM runtime type confusion on a module-level value `let` (G3) | 3 (`array_basic`, `array_grid`, `module_let_helpers`) |

The banked eval corpus is exposed the same way. Of 17,411 unique AILANG `code` bodies in the 40
most recent `eval_results/*` directories, **6,756 (39%) call `println` with no `import std/io`**,
and 301 (1.7%) use `letrec` (V10). Those counts come from text matching and are an upper bound
on exposure, not a measured fallback rate.

### The four gaps

**G1. A local `letrec` cannot reference itself in the VM, and the self-reference can bind to the
wrong function.** `letrec loop = \… . … loop(…)` lowers to a plain `stmt.VarDecl`
(`internal/gen/lower/block.go:182`). The lambda value is compiled before `loop` is a local, so
free-variable analysis does not capture it (`internal/bytecode/compiler/lambda.go:23–37`).
`classifyCallee` then falls back to *top-level* names (`internal/bytecode/compiler/call.go:85–105`).
That gives two outcomes:
- **G1a**: with no top-level function of that name, compilation fails with `call to unbound name`.
  The enclosing function becomes evaluator-only, so the program loses the VM speedup and, under
  `--bytecode`, falls back without saying so.
- **G1b (P0)**: with a top-level function of that name, **the VM calls the top-level function**.
  `benchmark/shadow.ail` below prints `10` on the evaluator and **`1000` under both `--bytecode`
  and `--strict-bytecode`**: a silently wrong answer that the strict gate does not catch (V11).
  This has already happened in the wild. cp_60170 trial 3 has a top-level `func go(t: int)` and a
  local `letrec go = \n vs. …` in `getAt`. The VM bound the local name to the top-level function
  and failed only because the arities differed (`TAIL_CALL: benchmark/solution.go expects 1 args,
  got 2`) (V12).

**G2. Prelude `println` is invisible to the VM compiler.** The prelude injects `println`'s *type*
(`internal/pipeline/prelude.go`, `InjectPrelude`) and, separately, its evaluator *value*
(`InjectPreludeValues`). It is not an import, so the VM compiler sees an unbound name and makes
`main` evaluator-only. When `main` is evaluator-only, the runner hands the **whole program** to the
evaluator (`internal/runner/vm.go`, `proto.EvalOnly` branch). The teaching prompt itself says
"println is in prelude, no import needed!" (V8), so a model that follows the prompt opts out of the
VM. The loader already has a mechanism that reaches both compile and runtime: the synthetic
entry-module import used for `Option`/`Result` (`internal/loader/prelude_imports.go`,
M-PRELUDE-OPTION-RESULT). `println` is the one prelude binding that does not use it.

**G3. A module-level value `let` (a constant applicative form, CAF) is miscompiled as a closure.**
In `let base = 40 + 1` followed by `base + 1` inside `main`, the VM reads `base` as a closure, and
`--strict-bytecode` fails at run time with `arith ADD: type mismatch Closure vs Int`. Compilation
*succeeds*, so the strict gate reports the program as native. Under `--bytecode` the failure
arrives after `println("start")` has run, so the runner cannot replay it (#1545) and the run
**fails with exit 1 on a program that is correct on the evaluator** (V13).
`examples/runnable/array_basic.ail` fails this way today.

**G4. The interop bridge cannot carry a user ADT from the VM to the evaluator.** A VM ADT value
carries only a constructor ordinal and name. `bytecodeToEval` returns `bridge: ADT values not yet
supported (M-BYTECODE-2E scope)` (`internal/vm/convert.go:105–107`). The reverse direction handles
only std ADTs through `stdADTTag` (`convert.go:186–192`). When VM code passes a user ADT into an
evaluator-only function, which in practice is often one made evaluator-only by G1, the run fails
*after* earlier effects, so it cannot fall back (V14). The no-replay rule is correct, because
consumed stdin cannot be re-read. The defect is that the incompatibility is discoverable at
*compile* time (argument types are static) but is found only at run time.

## Goals

**Primary goal:** every model-written program that uses only pure code, `std/io`, local `letrec`,
module-level values and user ADTs runs natively on the VM, and no program gives different output
on the two engines.

**Success metrics** (measured by the Phase 3 coverage report):
- Corpus A NATIVE: 31/41 → **≥ 40/41**.
- `examples/runnable` NATIVE among evaluator-passing files: 142/205 → **≥ 195/205**. The
  remainder is limited to Phase-2E effectful builtins plus documented exclusions.
- **Zero** DIVERGE and zero BYTECODE_CRASH in both corpora, excluding timing-printing files on an
  explicit, reviewed allowlist.
- cp_60198 on its 141KB test under `--bytecode`: 19s → measured and reported. No target number is
  claimed, because it has never run natively.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| G1 strategy: lambda-lift local `letrec` vs patch self-captures in the compiler | Determines TCO, mutual recursion and closure-escape behaviour | agent (constraints below) | design | med |
| Lexical-scope precedence rule in `classifyCallee` | A name bound in *any* enclosing lexical scope must never resolve to a top-level function. This is the systemic fix for G1b | compiler semantics | design | med |
| G2: conditional loader injection of `println` vs resolving it in the VM compiler (a naive synthetic import breaks user `func println`, V22) | Must preserve user shadowing (V21) | agent; either, never both | design | low |
| G4: carry ADT type identity in VM values vs propagate evaluator-only status statically to callers | The first is a value-layout change (40-byte VM values, see LIMITATIONS); the second fixes the symptom only | agent; both recommended (identity is the fix, propagation is the safety net) | compile | med |
| **F1: make the VM the default for `ailang run`** | Changes runtime behaviour for every user; a VM-only bug becomes a default-path bug | **human** | after Phase 3 | high |
| **F2: make the VM the default engine in the eval harness** | Moves time-limited pass rates and creates a baseline boundary | **human** | after Phase 3 | high |

### Design Freeze

Phases 1–3 need no human decision. Phase 4 **must pause** for:

- [ ] **F1**: VM default for `ailang run`, with `--evaluator` as the opt-out. Preconditions in Phase 4.
- [ ] **F2**: VM default in `internal/eval_harness/runner.go`. Banked rows get
      `engine: evaluator|vm`, and the first VM baseline is recorded as a boundary in the eval-analyzer
      boundary list.

## Solution Design

### Phase 1: G1, local `letrec` and lexical precedence (~2.5 days)

**1a. Compile local `letrec` natively.** The agent may pick either strategy; the constraints are
what matter.
- *Option L (lambda-lift, recommended):* in `internal/gen/lower`, hoist each local `letrec`
  binding to a module-private `FuncDecl` with a mangled, collision-free name
  (`<enclosing>$letrec$<name>$<n>`). Its free variables become leading parameters, and call sites
  are rewritten to pass them. A use that escapes as a *value* becomes a lambda that captures the
  free variables and calls the lifted function. Self tail calls become top-level `TAIL_CALL`.
  **Measured (V24):** a top-level self tail call with extra leading parameters runs 1,000,000
  iterations in 0.21s under `--strict-bytecode` (evaluator 7.77s) without overflowing, so the
  lifted form inherits depth-safe TCO. Mutual recursion falls out naturally, and mangling removes
  G1b's collision by construction.
- *Option C (self-capture):* pre-bind the `letrec` names as locals before compiling the values,
  emit `OpClosure` for each, then patch capture slots once all the closures exist (a new
  fixup op, which handles mutual recursion). **Unmeasured:** whether a tail call *through a
  closure value* runs in constant stack on the VM. V24 covers only top-level self tail calls.
  Choosing C requires measuring that first; this is why L is recommended.
- **Constraints, whichever option is chosen:** TCO is preserved (a 1,000,000-iteration local
  `letrec` loop must not overflow, matching the evaluator after
  m-interpreter-tail-call-elimination). Mutual local `letrec` works. Output matches the evaluator
  on every Conflict Surface fixture.

**1b (P0). Lexical precedence.** `classifyCallee` falls back to `funcIdx` only when the name is
bound by **no** enclosing lexical scope. Audit every other binder that the VM compiler does not
register as a local before the body that uses it: `letrec`, `let`-bound lambdas referenced in their
own value, match-arm bindings holding functions, and lambda parameters (already correct according
to the comment at `call.go:89`; verify with a test). If a binder is in scope but has no register, the
compiler must return an **error**, never fall back to a top-level name. That turns any remaining
case into a clean fallback instead of a wrong answer.

### Phase 2: G2, G3 and G4 (~2.5 days)

**2a. G2.** *(Revised after quorum round 0; see V22–V23.)* A plain synthetic
`import std/io (println)` is **not** safe. An import of a name the module also defines is a hard
`MOD015` error (V22), and the existing synthetic Option/Result import does *not* yield to a user
definition (a user `func None` is a type error, V23). Injecting naively would break entry
modules that define `func println`, which work today: user definitions shadow the type-only
prelude (V21). So the injection must be **conditional on the module's own bindings**:
- The loader adds `import std/io (println)` to an entry module **only if** the module has no
  top-level declaration named `println` and no explicit `std/io` import, selective or whole-module.
  The loader already holds the parsed AST at the injection site, so this is a lookup, not a new pass.
- **Fallback option, if the loader route proves awkward:** the VM compiler resolves an unbound
  `println` in an *entry* module to the `std/io.println` GlobalRef, mirroring exactly the
  type-only prelude's rule. The agent may choose between the two, but must not keep both, and
  either way `PreludeSurface()` stays the single source of truth.
- Required tests: (1) prelude `println`, no import, under `--strict-bytecode`; (2) user
  `func println` with no import still shadows and still compiles (V21 shape); (3) explicit import
  plus prelude use (no MOD015); (4) non-entry module with bare `println` still errors.
- **Out of scope, but surfaced:** V23 shows the existing Option/Result synthetic import has the
  same shadowing hole for `func None`/`Some`/`Ok`/`Err`. File it separately; do not widen this doc.
- Retire `InjectPreludeValues` for `println` only if nothing else depends on it (grep first; the
  REPL may).

**2b. G3.** Compile a module-level non-lambda `let` as a global evaluated once, in declaration
order, at image initialisation, using the value semantics the evaluator already gives it. Until
that lands, the compiler must **reject** a reference to a module-level value as a compile error,
so the program falls back before any effect instead of crashing mid-run.

**2c. G4.** Give VM ADT values their type identity (module + type name). The compiler has
`adtTypes`/`adtOrder` (`lambda.go:52–54`) and the LIMITATIONS entry on cross-ADT tag collisions
shows why name lookup by constructor is unsafe. With that identity, `bytecodeToEval` and
`EvalToBytecode` can round-trip user ADTs. As a safety net, if a VM→evaluator call edge would pass
a closure (still unsupported) or an ADT the bridge cannot name, the compiler marks the *caller*
evaluator-only, transitively, so the fallback happens before `main` runs.

### Phase 3: Make VM coverage a gated metric (~1 day)

- Extend `scripts/verify_bytecode_parity.go` with a strict pass. Each file gets
  `NATIVE | SILENT_FALLBACK | BYTECODE_CRASH | DIVERGE`, plus a histogram of fallback reasons.
  Today a silent fallback counts as `MATCH`, which is why these gaps went unseen.
- Corpus: `examples/runnable/` plus a frozen fixture set of model-written solutions with
  inputs and expected outputs. Seed it from the 41 Corpus A solutions; adding them under
  `tests/vm_corpus/` is planned work, not done.
- A `make vm-coverage` target that exits non-zero when the NATIVE count drops or any DIVERGE or
  BYTECODE_CRASH appears, with the snapshot banked next to the perf-sweep snapshots.

### Phase 4: VM as the default (human decision, F1/F2)

Evidence for the human: the timing table and coverage numbers above, re-measured after Phases 1–3.
The decision should not be taken until these preconditions hold:
1. Phase 3 gate green: zero DIVERGE and zero BYTECODE_CRASH on both corpora.
2. **#1956 resolved**, or the VM default refuses `--verify-contracts` rather than silently checking
   nothing. Audit *every* `ailang run` flag for VM-path no-ops (`--verify-contracts`,
   `--emit-trace`, recording and trace-value flags), so that a default switch drops no feature
   without saying so.
3. Memory: VM values are 40 bytes against the evaluator's 16 (LIMITATIONS), which means about 2.5×
   the allocation on cons-heavy code. Re-check `AILANG_EVAL_MAX_RSS` headroom on the
   memory-shaped perf-sweep programs.
4. Eval KPI (F2): bank `engine` on every row, record the first VM baseline as a boundary, and run
   `ailang eval-paired` evaluator-vs-VM on the same model outputs, which needs no new model calls,
   before any cross-version comparison is published.

### Files to Modify/Create

- `internal/gen/lower/block.go`: local `LetRec` lowering (lift or mark recursive), ~120 LOC
- `internal/gen/lower/program.go`: emit lifted local-`letrec` FuncDecls, ~60 LOC
- `internal/bytecode/compiler/call.go`: lexical-precedence rule, error instead of fallback, ~40 LOC
- `internal/bytecode/compiler/lambda.go`: Option C only, self-capture fixup, ~80 LOC
- `internal/bytecode/compiler/compiler.go`: module-level value globals (G3), transitive EvalOnly propagation (G4 safety net), ~120 LOC
- `internal/loader/prelude_imports.go`: `println` via synthetic `std/io` import, ~30 LOC
- `internal/pipeline/prelude.go`: keep `PreludeSurface()` authoritative; retire the value injection if it is unused, ~20 LOC
- `internal/vm/convert.go`: user-ADT round-trip using type identity, ~80 LOC
- `internal/bytecode/value.go`: ADT type identity on values, ~30 LOC
- `scripts/verify_bytecode_parity.go`: strict pass, four-way classification, reason histogram, ~120 LOC
- `Makefile`: `vm-coverage` target, ~10 LOC
- `examples/runnable/letrec_local_shadow.ail` (new): the G1b fixture, with expected output pinned, ~20 LOC
- `docs/LIMITATIONS.md`: remove the gaps as they close, ~10 LOC

## Conflict Surface

1. **Local name resolution in the VM compiler (`classifyCallee`).** Other constructs that live in
   this position: lambda parameters shadowing a top-level name (the `apply(f, x)` comment at
   `call.go:89`); a non-recursive `let f = \x. …` called later in the same body; top-level
   recursion and mutual recursion (`lowerTopLevelDecl` LetRec → per-binding FuncDecls); calls to
   imported functions that share a bare name with a local; module-qualified `GlobalRef` calls.
   Disambiguation: the innermost lexical binding wins, and a top-level name is used only if no
   lexical binding exists. *Deliberate change:* programs that print a wrong answer under
   `--bytecode` today (G1b) will print the evaluator's answer.
2. **Prelude `println`.** Other constructs: a user-defined `func println`, which must still shadow
   the prelude ("User definitions shadow prelude", `prelude.go`); explicit
   `import std/io (println)` (no duplicate-import error); `import std/io as IO` with
   `IO.println`; non-entry modules, which have no prelude and must still error; the REPL.
3. **Module-level value `let`.** Other constructs: module-level lambda lets (`let quad = \y. …` in
   `module_let_helpers.ail`, which must stay functions); top-level `func`; exported lets read from
   another module; lets whose initialisers have effects, which the type checker should already
   reject at module level (verify, do not assume).
4. **Bridge ADT conversion.** Other constructs: std ADTs via `stdADTTag` (the existing path must
   be unchanged); display conversion (`BytecodeToEvalForDisplay`); same-named constructors across
   two ADTs (the cross-ADT tag-collision LIMITATIONS entry, which is why identity must be carried
   and not looked up by name).

**Programs that must still work, with byte-identical output on both engines:**
`examples/runnable/adt_option.ail`, `adt_simple.ail`, `adt_list_fields.ail`, `block_recursion.ail`
and `effectful_list_t3_foldlE_acc.ail` (all NATIVE today, V15). Plus `letrec_recursion.ail`,
`module_let_helpers.ail`, `array_basic.ail` and `array_grid.ail`, which are evaluator-correct
today and must become NATIVE (V15).

## Examples

### G1b: silent wrong answer (P0)

```ailang
module benchmark/shadow
import std/io (println)

func go(a: int, b: int) -> int = 1000

func sumTo(n: int) -> int {
  letrec go = \i acc. if i > n then acc else go(i + 1, acc + i);
  go(1, 0)
}

export func main() -> () ! {IO} {
  println(show(sumTo(4)))
}
```
Evaluator: `10`. `--bytecode`: `1000`. `--strict-bytecode`: `1000`. After the fix, all three
print `10`.

### G3: miscompiled module-level value

```ailang
module benchmark/caf
import std/io (println)

let base = 40 + 1

export func main() -> () ! {IO} = {
  println("start");
  println(show(base + 1))
}
```
Evaluator: `start`, `42`. `--bytecode`: `start`, then `arith ADD: type mismatch Closure vs Int`,
exit 1. After the fix, both engines print `start` and `42`.

## Success Criteria

- [ ] `shadow.ail` prints `10` under evaluator, `--bytecode` and `--strict-bytecode` (pinned regression test)
- [ ] A local `letrec` 1,000,000-iteration tail loop runs natively without stack overflow; mutual local `letrec` passes
- [ ] `examples/runnable/letrec_recursion.ail`, `module_let_helpers.ail`, `array_basic.ail` and `array_grid.ail` pass `--strict-bytecode` with evaluator-identical stdout
- [ ] An entry module that uses prelude `println` with no import passes `--strict-bytecode`; the user-`func println` shadowing test still passes
- [ ] A user ADT passed VM→evaluator→VM round-trips; any unbridgeable edge falls back *before* `main` runs (test asserts no stdout before the fallback)
- [ ] `make vm-coverage` reports Corpus A ≥ 40/41 NATIVE and `examples/runnable` ≥ 195/205 NATIVE, with zero DIVERGE or BYTECODE_CRASH
- [ ] cp_60212, cp_60198 and cp_60159 re-timed on both engines, with numbers recorded in the implementation report
- [ ] `make test`, `make verify-examples` and `make parity` pass; changelog fragment; LIMITATIONS updated

## Testing Strategy

**Unit tests:** compiler tests for lexical precedence (each binder kind from the Phase 1b audit),
local `letrec` (self, mutual, escaping-as-value, tail), module-level value globals, and bridge ADT
round-trip.
**Mutation-test the new tests:** revert each fix in a scratchpad copy and confirm that the test
fails, per the repo's mutation-testing practice.
**Integration:** the Phase 3 strict parity run over both corpora.
**Manual:** re-time the spike solutions; confirm there is no change for the timing-printing allowlist.

## Deferred Decisions

- G1 Option L vs Option C: the agent may choose, within the constraints.
- Whether G4 type identity is a pointer to a shared ADT descriptor or an interned id: the agent may choose.
- Whether `InjectPreludeValues` survives for the REPL: the agent may choose after grepping its callers.
- Corpus fixture storage location and size cap: the agent may choose (keep it small; no model-output dumps).

## Non-Goals

- Phase-2E effectful builtins (FS, clock sleep, tar, game): already scoped elsewhere, and they fall back cleanly today.
- VM speed on cons-heavy code (the 1.8× LIMITATIONS entry): the cons representation fix (D-19 / m-list-cons-quadratic).
- Algorithmic complexity gaps (O(n) `Array.set`/`Map.insert`, no BigInt): separate roadmap docs.
- Closure values across the bridge: covered only by the static fallback safety net.

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Lambda-lifting changes closure identity or effect rows | Med | Lifted functions keep the binding's effect row; escaping uses go through a capturing lambda; byte-identity fixtures |
| Synthetic `println` import collides with user code (MOD015, V22) | High | Inject only when the module defines or imports no `println`; four required tests in Phase 2a |
| Module-level value initialisation order differs from the evaluator | Med | Declaration order, evaluated once; a test with dependent constants |
| A coverage gate that is flaky on timing-printing files | Low | A reviewed allowlist, never a pattern |
| Phase 4 taken before #1956 is closed | High | Listed as a hard precondition on F1 |

## Verification Log

Binary: `ailang v0.54.19-1-g379a9e20e-dirty` (commit 379a9e2), 2026-10-28. Scratch artefacts live in
the session scratchpad; the repro sources are inlined in Examples or quoted below.

| # | Claim | Command / evidence | Result |
|---|-------|--------------------|--------|
| V1 | The harness runs the evaluator, with no `--bytecode` | read `internal/eval_harness/runner.go:277` | `args := []string{"run", "--entry", "main", "--quiet"}` plus `--relax-modules`, stdlib, caps; no bytecode flag. Confirmed |
| V2 | `ailang run` has `--bytecode` and `--strict-bytecode` | `ailang run --help` | "Run via the bytecode VM instead of the evaluator", "Fail instead of falling back…". Confirmed |
| V3 | cp_60212 21×; trace tier irrelevant | sequential re-time, `AILANG_TRACE=standard|off` × eval/VM | 23.77/1.14s (standard), 23.96/1.15s (off); cp_60198 19.05/19.08 vs 19.35/19.08. Confirmed |
| V4 | Dropping the `println` import removes the speedup | cp_60212 copy with `import std/io (readLine)` only; `ailang check` passes | eval 24.10s, VM 23.77s. Confirmed |
| V5 | cp_60164/cp_60181 time out on both engines | `--bytecode` and `--strict-bytecode` on the 15KB and 32KB tests | TLE >60s / >120s on all three modes. Confirmed. Appends patched to cons (`::`): still >60s |
| V6 | Corpus A classification | per-solution eval / strict / bytecode run on the smallest hidden test | NATIVE 31, SILENT_FALLBACK 6, BYTECODE_CRASH 4. Confirmed |
| V7 | The non-native set equals the `letrec` set | `grep -l letrec` vs the non-native list | Identical 10 = 10. Confirmed |
| V8 | Teaching prompt: 12 `letrec` mentions; println "no import needed" | `ailang prompt \| grep -c letrec`; grep prelude | 12; line 13 "println is in prelude, no import needed!". Confirmed |
| V9 | `examples/runnable` classification and reasons | 260 files, caps IO,FS,Clock, empty stdin, 20s; reasons from strict EvalReason | 205 eval-pass: 142 NATIVE / 61 FALLBACK / 1 CRASH / 1 DIVERGE (timing output); 47 × unbound `println`. Confirmed |
| V10 | Banked-corpus exposure | 40 newest `eval_results/*`, `lang==ailang`, dedup by sha1 of `code` | 17,411 unique; 6,756 prelude `println` without `std/io` import; 301 `letrec`. Text heuristic, upper bound |
| V11 | G1b silent wrong answer | `shadow.ail` (Examples) under the three modes | 10 / 1000 / 1000. Confirmed |
| V12 | G1b in a model solution | cp_60170 t3: `grep -n "\bgo\b"`; `--bytecode` run | top-level `func go(t: int)` at line 202, local `letrec go = \n vs.` at line 21; `TAIL_CALL: benchmark/solution.go expects 1 args, got 2`. Confirmed |
| V13 | G3 miscompile | `caf.ail` (Examples) | evaluator `start 42`; `--bytecode` `start` then `arith ADD: type mismatch Closure vs Int`, rc=1; `array_basic.ail`: `__string_intToStr: expected int` after output, rc=1. Confirmed |
| V14 | G4 crash after effects | `adtbridge.ail`: `type Dir = U \| D`; `step(d: Dir, n)` with local `letrec go`; `main` prints "start", then `step(U, 5)` | evaluator `start 5`; `--bytecode` `start` then `CALL via eval interop: arg 0: bridge: ADT values not yet supported`. The same program with no earlier effect (`adtarg.ail`) falls back silently and prints `5`. Confirmed: the crash depends on an effect running first, per the #1545 no-replay rule in `internal/runner/vm.go` |
| V15 | Regression fixtures exist; current state | `ls` + strict run | `adt_option`, `adt_simple`, `adt_list_fields`, `block_recursion`, `effectful_list_t3_foldlE_acc`: strict rc 0. `letrec_recursion`, `module_let_helpers`, `array_basic`, `array_grid`: evaluator rc 0, strict rc 1. Confirmed |
| V16 | Negative: no existing doc covers these gaps | duplicate gate (neural top 0.38); `grep -rniE "shadow\|letrec\|prelude"` on LIMITATIONS for VM rows; `gh issue list --search "bytecode letrec"` | No VM-letrec/prelude/CAF row or issue. Related: #1956 only. Confirmed |
| V17 | Negative: the parity script cannot see silent fallback | read `scripts/verify_bytecode_parity.go` header | Categories MATCH/EVAL_SKIP/VM_BRIDGE/VM_COMPILE/VM_RUNTIME/DIVERGE; a non-strict fallback that matches is `MATCH`. Confirmed |
| V18 | Negative: the VM compiler has no LetRec handling | `grep -rni letrec internal/bytecode` (non-test) | No hits. LetRec is lowered to `stmt.VarDecl` in `internal/gen/lower/block.go:182`. Confirmed |
| V19 | Mechanism: `classifyCallee` falls back to top-level names | read `internal/bytecode/compiler/call.go:85–105` | locals → `canonicalFuncName` in `funcIdx` → bare name in `funcIdx` → `call to unbound name`. Confirmed |
| V20 | Mechanism: the prelude is not an import | read `internal/pipeline/prelude.go` and `internal/loader/prelude_imports.go` | `println` is a type binding plus an evaluator value; only `std/option` and `std/result` are synthetic loader imports. Confirmed |
| V21 | A user `func println` shadows the type-only prelude today | entry module with `func println(s: string) -> () ! {IO} = ()` and no import, calling `println("…")` | Compiles and prints nothing. Confirmed: the behaviour Phase 2a must preserve |
| V22 | An import plus a same-name definition is an error | `import std/io (println)` plus `func println` | `MOD015: 'println' is both imported … and defined in this module as a func`. Confirmed. A naive synthetic import would break V21 programs (quorum objection, glm) |
| V23 | The synthetic Option import does not yield to a user definition | entry module with `func None() -> int = 7` | Type error: `cannot unify type application Option[α2] with () -> α3`. Confirmed: an existing hole, out of scope here |
| V24 | VM TCO on top-level self tail calls with extra params (quorum objection, kimi) | `func loop(k, acc, n) = if n == 0 then acc else loop(k, acc + k, n - 1)` with n=1,000,000, plus a higher-order variant | `--strict-bytecode`: correct, 0.21s; evaluator 7.77s. Confirmed for top-level; closure-value tail calls unmeasured (Option C caveat) |

## Quorum Log

**Round 0 (2026-10-28T09:30:23Z): BLOCKED**, $0.2203. Artifact:
`.ailang/state/mission-quorum/m-vm-coverage-letrec-prelude-bridge-2026-10-28T09-30-23Z.json`.
- `oc-kimi-k3`: **reject**. Option L's "VM already optimises self tail calls" was unmeasured.
  **Resolved by measurement (V24):** true for top-level self tail calls. Option C's closure-value
  tail calls are now explicitly marked unmeasured.
- `oc-glm-5-3`: **reject**. Phase 2a assumed a user `func println` would shadow a synthetic import.
  **Correct, and the design changed:** V22 shows MOD015 and V23 shows the existing synthetic import
  does not yield either. Phase 2a now injects conditionally on the module's own bindings, or
  resolves in the compiler; the separate Option/Result hole is surfaced for its own report.
- `gemini-4-argon`: **pass**, noting that the REPL dependence on `InjectPreludeValues` and the
  ADT-identity representation are left open. Both are already listed as Deferred Decisions,
  resolvable by the agent.
- `gpt6-1-sol`: **absent** (unreachable). The quorum degraded to N−1; this is not a silent pass.
- Controller (in-session): pass.

Round 1 has **not** been run. Both blocking objections were answered by first-party measurement
and a design change. Per the re-quorum-once guardrail, the next quorum is the human's or the
planner's call.

## Related Documents

- [m-bytecode-vm-parity-bugs.md](../v1_0_0/m-bytecode-vm-parity-bugs.md): output-divergence soundness bugs; distinct scope (effect replay classification, closures)
- [m-perf4-bytecode-interpreter.md](../v1_1_0/m-perf4-bytecode-interpreter.md): pre-VM planning doc; its open question "opt-in flag or default" is F1 here
- [m-vm-determinism.md](../v0_47_2/m-vm-determinism.md): compile-time determinism; orthogonal
- [m-interpreter-tail-call-elimination.md](../../implemented/v0_52_0/m-interpreter-tail-call-elimination.md): TCO parity contract that Phase 1 must preserve
- [m-vm-stack-limit-parity.md](../../implemented/v1_0_0/m-vm-stack-limit-parity.md): stack-limit parity
- [m-foldl-cons-cost-model.md](../../implemented/v0_52_0/m-foldl-cons-cost-model.md): why the VM is slower on cons building
- [m-list-cons-quadratic.md](../m-list-cons-quadratic.md): PARKED cons representation fix

## References

- [Design Axioms](/docs/references/axioms)
- [#1956](https://github.com/sunholo-data/ailang/issues/1956): `--bytecode --verify-contracts` silently checks no contracts
- [#1545](https://github.com/sunholo-data/ailang/issues/1545): no replay after effects in non-strict VM mode

## Future Work

- Closure values across the bridge (completes the M-BYTECODE-2E scope).
- Phase-2E effectful builtins, so FS-heavy benchmarks also run natively.
- Once F2 lands: a per-benchmark VM-vs-evaluator speed column in the eval dashboard.

---

**Document created**: 2026-10-28
**Last updated**: 2026-10-28
