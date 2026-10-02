# M-EVAL-TAIL-CALLS: tail-call elimination in the tree-walking evaluator

**Status**: IMPLEMENTED
**Target**: v0.51.1
**Priority**: P1
**Estimated**: 3 days
**Dependencies**: None
**Issues**: [#1486](https://github.com/sunholo-data/ailang/issues/1486) (primary). Related: #1487 (`ailang test --bytecode`), #1317 (unsafe depth ceilings)
**Reporter**: `stapledons_godot`. Its sim shell (`sim/ship.ail`) is a tail-recursive stdin loop, and its 10,000-tick replay needs `--max-recursion-depth 100000` on the interpreter.
**Quorum**: trigger 2 fired (D3 changes the shared apply path). Round 1 was BLOCKED 3/3 (env/resolver
restore on the exit path; D3 contradicted D6; trace schema unverified). Round 2 was BLOCKED 3/3 (per-chain
trace option; resolver chain semantics; budget boundary premise). Every objection was verified against the
code and folded in: D3 per-frame contracts, "Frame state", V15–V19. A third round would exceed the re-quorum
guardrail, so the revised doc goes to the operator for ratification. Artifacts are in
`.ailang/state/mission-quorum/m-eval-tail-calls-*`.
**History**: tail-call optimization was deferred to "v0.4.0" in `implemented/v0_3_0/M-R4_recursion.md` (§Future Work), offered again as Option A in `implemented/v0_4_8/m-bug-recursion-depth.md`, and never built.

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Removes a VM↔interpreter divergence: the same program no longer succeeds on one engine and fails RT_REC_003 on the other. Evaluation order is unchanged |
| A2: Replayability | 0 | Trace events stay byte-identical (D6) |
| A3: Effect Legibility | 0 | Effects run in the same order; frames with effect annotations are never replaced (D4) |
| A4: Explicit Authority | 0 | Budget (`@limit`/`@min`) and rand-mode frames keep their exact scoping (D4) |
| A5: Bounded Verification | 0 | `ensures` keeps running on every call that has one (D4) |
| A6: Safe Concurrency | 0 | Per-evaluator state only |
| A7: Machines First | +1 | Agents write tail-recursive loops, the idiomatic AILANG iteration. They stop failing at an arbitrary 10,000 |
| A8: Minimal Syntax | 0 | No syntax |
| A9: Cost Visibility | +1 | Memory for a tail loop goes from O(n) live frames (~439–751 B/level, V6) to O(1) |
| A10: Composability | +1 | `CallFunction` (entry + builtin callbacks) and `evalCoreApp` share one apply path (D3) |
| A11: Structured Failure | 0 | RT_REC_003 still bounds non-tail recursion |
| A12: System Boundary | 0 | — |

**Net Score: +4** → Proceed. No −1 on A1/A3/A4/A7.

## Problem Statement

Every AILANG function call in the evaluator is Go recursion. `evalCore` → `evalCoreApp` → `evalCore(body)`
(V1), and each call increments `recursionDepth`, which RT_REC_003 caps at 10,000 (V2). A loop in tail position
therefore dies at 10,000 iterations in the interpreter, while the bytecode VM, which emits `TAIL_CALL` and
reuses the frame (V5), runs it in constant space. Measured on v0.51.0 (V4), at n = 50,000:

| Shape | interpreter | `--bytecode --strict-bytecode` |
|---|---|---|
| tail call in `if` branch | RT_REC_003 | 50000 |
| tail call in `match` arm | RT_REC_003 | 50000 |
| `let` then tail call | RT_REC_003 | 50000 |
| mutual recursion (`isEven`/`isOdd`) | RT_REC_003 | true |

Raising `--max-recursion-depth` is only a stopgap: it is unsafe much beyond ~84k frames (#1317). Every pending
frame also pins its bindings (V7), so memory grows linearly.

## Goals

- A call in tail position of a function body runs in constant evaluator depth and constant live memory, for
  any callee, matching the VM on all four shapes above.
- No observable change otherwise: same results, effect order, contract checks, budget/rand scoping and trace
  events. RT_REC_003 still bounds non-tail recursion at the configured depth.

## High-Impact Decisions

| # | Decision | Choice | Why |
|---|---|---|---|
| D1 | Scope | **General tail calls**: any `*FunctionValue` callee in tail position, including mutual recursion | VM parity (V4). Self-only TCE would leave mutual recursion divergent |
| D2 | How tail position is tracked | **An explicit `tail bool` argument** threaded through the tail-capable evaluators (If, Let, LetRec, Match, decision-tree match, DictAbs), not a mutable evaluator field | A field would leak into builtin callbacks (`CallValue` re-enters the evaluator mid-expression, V3) and turn a non-tail call into a tail call. An argument is scoped by construction |
| D3 | Where the trampoline lives, and whose contract each frame follows | **One shared `applyFunctionValue`** used by `evalCoreApp` and `CallFunction`. Contracts are **per frame**: the *first* frame follows its caller's current contract (`evalCoreApp`: depth, trace, resolver wrap, budget name = function name; `CallFunction`: zero-arg unit injection, no depth, no trace, no resolver wrap, budget name `""`, V17). **Every tail-called frame follows the `evalCoreApp` contract**, because today it is reached through `evalCoreApp` | Quorum round 2: a per-chain option would drop the trace events of helpers that an untraced entry tail-calls. Per-frame contracts make each frame emit exactly what the nested call emits today |
| D4 | When a frame may be replaced | Only when the **current** function has none of: `ensures` with contract checking enabled, `@limit`/`@min` budgets, a declared rand mode. Otherwise the call is an ordinary nested call | Each needs work after the body: postcondition evaluation (V8), `@min` check on pop, rand-mode pop (V9). Replacing the frame would change their meaning |
| D5 | `recursionDepth` on a tail call | **Unchanged** (the frame is reused, as in the VM) | RT_REC_003 then measures what it was meant to, the depth of pending work |
| D6 | Trace events | **The same `RecordFunctionEnter`/`RecordFunctionExit` call sequence as today**, so the collector's output is identical except for timestamps and durations, which already differ run to run. Enters fire as the chain advances. Exits are replayed innermost-first when the chain returns, reproducing today's per-frame rules: a body error still emits the exit (nil result); a failed `requires` or `ensures` emits none for that frame; a Go panic emits none (today's exits are not deferred either) | `Depth`, span IDs and durations are computed **inside the collector** from the order of calls (V15). The evaluator passes only name and args/result, so an identical call sequence gives identical events. The name stack (O(n)) exists only while per-call tracing is on, which already stores O(n) events |

### Design Freeze

- [x] D1–D6 above. All are agent-resolvable; no human ratification needed.

## Solution Design

### Overview

```
evalCoreApp(app)                     CallFunction(fn, args)
  opts{countDepth, trace}               opts{}            ← each keeps today's contract (D3)
        \                                 /
         applyFunctionValue(fn, args, name, opts)
           baseEnv, baseResolver := e.env, e.resolver
           defer restore(baseEnv, baseResolver)          ← every return path, incl. panic
           if first frame counts depth: depth++ (once) ; defer depth--
           (per frame: contract = caller's opts for the first frame, evalCoreApp's for tail frames)
           loop:
             enter budget charge boundary; keep restore only if not the no-op (V9, V19)
             if frame traces: RecordFunctionEnter(name); names.push(name)
             e.env      = fn.Env child + params
             e.resolver = wrapIfNotCovered(e.resolver, fn.Resolver)   ← as the nested call does (V18);
                                                                    skipped for a CallFunction first frame
             if !replaceable(fn): push budget frame / rand mode (deferred pops, as today)
             requires(fn)  → on failure: names.pop() (no exit for this frame), replay rest, return err
             r, err := evalTail(body, tail = replaceable(fn))    (D4)
             if err == nil && r is *tailCall:
                 fn, args, name = r.fn, r.args, r.name           ← depth unchanged (D5)
                 continue
             if err == nil: ensures(fn, r) → on failure: names.pop(), replay rest, return err
             replay exits innermost-first for names (D6)
             return r, err
```
**Frame state (quorum rounds 1–2).** Before the loop, capture the caller's `e.env` and `e.resolver` as the
*base*; restore both on **every** return path (success, body error, requires/ensures failure, Go panic; a
deferred restore, where today's code restores by hand on each path, V16).
- *Env:* each iteration sets `e.env` to a child of `fn.Env` holding the params, as today. The child's parent is
  the closure's env, never the caller's (V7), so nothing chains.
- *Resolver:* each tail frame wraps the **current** `e.resolver` with `fn.Resolver` unless `resolverCovers`
  says it is already present, which is exactly what the nested call does today, where the callee wraps the
  caller's current chain. Frame *i* therefore sees the same chain as today, and the chain stops growing once
  each distinct module resolver is in it (`resolverCovers` dedupe, V18). It is bounded by the number of modules,
  not iterations. The first frame of a `CallFunction` chain does not wrap (V17).
- *Budget charge boundary:* entered **per iteration**, as each nested call does today. The restore closure is
  kept only when it is not the no-op. `enterBudgetChargeBoundary` returns `func(){}` when the saved depth is 0
  (V9), and the depth is 0 at a tail call because it is reset at frame entry and raised only inside a builtin's
  paired `Begin`/`EndBudgetChargeScope` (V19). So the retained list is O(1) in practice, and exact if that ever
  changes.
- *Budget frames / rand mode:* pushed only by frames D4 keeps nested, so a replaced frame never owns one.

`evalTail(expr, tail)`, the tail-aware dispatch:
- `core.App` with `tail == true`: evaluate the callee and arguments with `evalCore`, force an `IndirectValue`.
  If the callee is a `*FunctionValue` with exactly `len(Params)` args, return `&tailCall{fn, args, name}`
  instead of applying. Over-application (auto-curry), builtins and constructors apply normally.
- `core.If`: the condition is non-tail, both branches use `tail`.
- `core.Let` / `core.LetRec`: binding values are non-tail, the body uses `tail`. The env restore runs as today;
  the trampoline re-establishes the env on each iteration.
- `core.Match` and the `AILANG_DTREE` decision-tree path: scrutinee and guards are non-tail, the arm body uses
  `tail`.
- `core.DictAbs`: the body uses `tail`.
- Anything else: `evalCore(expr)`, never a tail call.

Blocks need no case. `normalizeBlock` lowers `{a; b; c}` to nested `core.Let` (V10), so the last expression
is a Let body.

`tailCall` is an unexported type implementing `Value`. Only `evalTail` with `tail == true` produces it, and
only `applyFunctionValue` consumes it. An invariant test asserts it never escapes a public evaluator entry point.

### Files to Modify/Create

- `internal/eval/eval_apply.go` (NEW): `applyFunctionValue`, `tailCall`, `replaceable` (~200)
- `internal/eval/eval_operations.go`: `evalCoreApp` delegates the `*FunctionValue` case (−120)
- `internal/eval/eval_evaluator.go`: `CallFunction` delegates (−80)
- `internal/eval/eval_expressions.go`: `evalCoreIf`, `evalCoreLet`, `evalCoreLetRec` take `tail bool` (~+30)
- `internal/eval/eval_patterns.go`: `evalCoreMatch`, `evalDictAbs` take `tail bool` (~+20)
- `internal/eval/decision_tree.go`: arm body takes `tail` (~+10)
- `internal/eval/tail_call_test.go` (NEW): shapes, non-regressions, invariants
- `cmd/ailang/tail_call_parity_test.go` (NEW): interpreter vs strict VM, #1486 repro
- `examples/runnable/tail_recursion_loop.ail` (NEW)
- `changelogs/unreleased/<date>-eval-tail-calls.md`

## Conflict Surface

1. **Positions extended:** the `*FunctionValue` branch of `evalCoreApp`, `CallFunction`, and the
   body/branch positions of If, Let, LetRec, Match, DTree match and DictAbs.
2. **Other constructs living there:**
   - Auto-curry over-application (`len(args) > len(params)`, V3): stays a nested call.
   - `BuiltinFunction` and `ConstructorClosure` callees: never tail calls. They push no frame anyway.
   - Builtin callbacks (`CallValue`/`CallValueN` → `CallFunction`): they get the trampoline for the callback's
     own body. A callback's result returns to the builtin, so no tail position crosses that boundary.
   - Contracts: `requires` runs per iteration at frame setup, as today. A function with active `ensures` is
     never replaced (D4).
   - Budget charge boundary (`enterBudgetChargeBoundary`, V9): save/reset on entry and restore on exit.
     Running it once around the loop is equivalent, because every iteration enters at depth 0.
   - Goroutine stack hops (`evalSegmentLevels`, V11): untouched. TCE makes them rarer.
   - Trace enter/exit pairing: D6.
3. **Disambiguation:** static. Tail position comes from the evaluator's own recursion structure, and
   replaceability from fields on `*FunctionValue`.
4. **Programs that must still work:** `examples/` under `make verify-examples`; the contract suites
   (`internal/eval` contract tests, `ailang test` with `ensures`); `TestBudgetFrame_*` (cmd/ailang); the
   `stack_hop_test.go` suite; deep-trace golden tests; `tests/golden/`.
5. **Deliberate changes:** a tail-recursive program that used to fail RT_REC_003 now succeeds. A program
   relying on RT_REC_003 to stop a runaway tail loop now loops until killed, the same behaviour it already
   has under `--bytecode`.

## Examples

```ailang
module tail_recursion_loop
pure func count(i: int, n: int) -> int = if i >= n then i else count(i + 1, n)
export pure func main(n: int) -> int = count(0, n)
```
`ailang run --entry main --args-json 1000000` prints `1000000`. Today: RT_REC_003 at 10,000.

## Success Criteria

- [ ] The four shapes in the Problem Statement run at n = 1,000,000 at the default depth, with output identical to `--strict-bytecode`
- [ ] #1486 repro: a 20,000-line stdin loop prints `20000` without `--max-recursion-depth`
- [ ] Non-tail recursion (`f(n) = 1 + f(n-1)`) still fails RT_REC_003 at the configured depth
- [ ] A function with `ensures` that tail-calls itself still checks the postcondition on every call (a violation in an inner call is reported)
- [ ] `@limit`/`@min` budget and rand-mode functions behave exactly as before (`TestBudgetFrame_*` green)
- [ ] With `AILANG_TRACE=deep`, a small tail loop's trace is identical before and after, except timestamps and durations: same events, names, args, results, depths and span parentage
- [ ] Trace replay matches today on a body error inside a tail chain (exits emitted) and on a `requires` failure at iteration k (frames 1..k-1 emit exits; frame k does not)
- [ ] `CallFunction` (entry, builtin callbacks): recursion depth and trace events unchanged for its own frame; helpers it tail-calls are traced exactly as today
- [ ] Three-module resolver test: identical lookups to today; chain length bounded after 10^5 iterations
- [ ] Invariant: no public entry point ever returns a `*tailCall`
- [ ] Memory: live heap during a 10^6 tail loop stays flat. Assert allocation, not RSS
- [ ] `make test`, `make lint`, `make verify-examples`, `make check-file-sizes` green

## Testing Strategy

Table tests per shape (self, mutual, through If/Let/LetRec/Match/DTree/DictAbs) at depth 10^6. Negative tests
for D4 (each disqualifier keeps a nested call; depth counter observed via a too-small `--max-recursion-depth`).
Trace byte-identity. CLI parity against `--strict-bytecode`. Mutation check: disable `replaceable` and confirm
the contract and budget tests fail.

## Risks & Mitigations

| Risk | Mitigation |
|---|---|
| A tail flag leaks into a non-tail context and changes semantics | D2: an argument, not state. Invariant test that `*tailCall` never escapes |
| Env/resolver restored wrongly between iterations | Teardown runs before each jump, exactly as the nested return path does; tests read globals and closures inside tail loops |
| Trace consumers see different nesting | D6 keeps the events byte-identical |
| Shared routine changes `CallFunction`'s behaviour | D3 `opts` keeps its contract (no depth increment, no trace). A test asserts the entry call's depth and trace events are unchanged |
| Resolver chain differs from today's | Each frame wraps the current chain exactly as the nested call does (no base re-wrap). A test runs a **three-module** tail chain A→B→C→B→C… where only module B supplies a fallback binding that C's body reaches through the chain, and asserts the same lookup results as today, plus a bounded chain length after 10^5 iterations |

## Verification Log

| # | Claim | Evidence |
|---|---|---|
| V1 | Calls are direct Go recursion | `internal/eval/eval_expressions.go:105` `case *core.App` → `evalCoreApp` (`eval_operations.go:16`) → `e.evalCore(coreBody)` (`eval_operations.go:~190`) |
| V2 | Depth guard per call, default 10,000 | `eval_operations.go:56-61` (`recursionDepth++` / defer); `eval_evaluator.go:159,171` |
| V3 | `CallFunction` duplicates setup without the depth guard or trace; builtins re-enter via `CallValue` | `eval_evaluator.go:296-380, 449, 462` |
| V4 | Four shapes: interpreter fails, VM passes | v0.51.0 `bin/ailang`, `tc.ail` at n = 50,000 (this doc's table) |
| V5 | VM reuses the frame on `TAIL_CALL` | `internal/vm/vm.go:296-345` (`frame.reuseFor`), `internal/bytecode/compiler/call.go:189-260` |
| V6 | ~439–751 B per evaluator level | `internal/eval/eval_expressions.go:11-26` (stack-segment rationale) |
| V7 | Pending frames pin bindings | `evalCoreApp` keeps `oldEnv` live in each Go frame (`eval_operations.go:157`); `implemented/v1_0_0/m-v1-memory-footprint.md:234` (V23) |
| V8 | Postconditions run after the body | `eval_operations.go:200-209`; `checkPostconditions` `eval_evaluator.go:627` (no-op when `len(fn.Postconditions)==0` or checking disabled) |
| V9 | Budget frame / rand mode only for annotated functions; charge boundary is save/reset/restore | `pushBudgetFrameIfAnnotated` (needs `EffectBudgets`/`EffectMinBudgets`), `pushRandModeIfDeclared` (needs `EffectRandMode`), `enterBudgetChargeBoundary` (`eval_evaluator.go:95`) |
| V10 | Blocks lower to nested Let | `internal/elaborate/expr_control.go:239` `normalizeBlock` |
| V11 | Stack hopping exists, no `SetMaxStack` | `eval_expressions.go:27,50-79` `evalSegmentLevels`, `evalCoreOnFreshStack` |
| V12 | No existing tail-call mechanism in eval | `grep -rn "tail\|trampoline\|TailCall" internal/eval` → only `_list_tail` and list-pattern tails |
| V13 | No call-stack structure to keep consistent | `grep -rn "CallStack\|callStack" internal/eval` → empty |
| V14 | `ailang run` and `ailang test` both use `CoreEvaluator` | `internal/runtime/runtime.go:61`, `internal/testing/executor.go:35` |
| V15 | Trace depth, spans and durations are computed by the collector from call order | `internal/trace/collector.go:203-255`: `RecordFunctionEnter` does `c.depth++`, `pushSpan`, `funcEntryTimes[c.depth]`; `RecordFunctionExit` uses `c.depth`, `popSpan`, then `c.depth--`. The evaluator passes only name and args/result (`eval_operations.go:117-122, 219-221`) |
| V17 | `CallFunction` contract | `eval_evaluator.go:296-345`: zero-arg unit injection, child env, budget boundary + budget frame (name `""`) + rand mode, requires/ensures; **no** `recursionDepth`, **no** trace, **no** resolver wrap |
| V18 | Resolver wrap dedupe | `eval_evaluator.go:39` `resolverCovers` walks the `FallbackResolver` tree; `eval_operations.go:165-172` wraps only when not covered |
| V19 | Charge-scope depth is raised only inside builtin effect ops and reset per frame | `internal/effects/context.go:100-130` (`Begin`/`EndBudgetChargeScope` paired; `SaveAndResetBudgetChargeScope` zeroes on function entry) |
| V16 | Today's per-frame exit rules | `eval_operations.go:178-185` (precondition failure returns before the exit), `:200-209` (postcondition failure returns before the exit), `:211-221` (a body error falls through to restore and `RecordFunctionExit`) |

## Related Documents

- `design_docs/implemented/v0_3_0/M-R4_recursion.md`, `design_docs/implemented/v0_4_8/m-bug-recursion-depth.md` (deferred TCO)
- `design_docs/implemented/v1_0_0/m-v1-memory-footprint.md` (frame retention)
- #1487 `ailang test --bytecode` (complementary: VM speed for tests)
