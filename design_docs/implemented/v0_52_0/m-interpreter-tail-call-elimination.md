# M-INTERPRETER-TAIL-CALL-ELIMINATION: The Tree-Walking Interpreter Diverges From the VM on Tail Calls (RT_REC_003 at 10k Where the VM Runs in Constant Stack)

**Status**: Superseded — the same problem (#1486, #1317) was solved by [m-eval-tail-calls.md](m-eval-tail-calls.md) (M-EVAL-TAIL-CALLS, commits 37ba2153a..3c77de8e4). Kept for the record; not implemented as written.
**Target**: v0.51.1
**Priority**: P1 (High) — blocks VM↔interpreter parity sessions for exactly the loop shape AI-generated code writes most (stateful read/process loops); forces unsafe `--max-recursion-depth` workarounds
**Estimated**: 3 days (1.5d implementation + 1.5d tests/verification/docs, 2x'd)
**Dependencies**: None
**Bug report**: v0.51.0 (b99dd25), darwin arm64 — coordinator task `task-40abb254`; downstream consumer `stapledons-godot` (`sim/ship.ail`, #1317 ceiling)
**Historical note**: [M-R4_recursion](../../implemented/v0_3_0/M-R4_recursion.md) (v0.3.0) listed "Tail-call optimization (deferred to v0.4.0)" as a secondary goal for the evaluator. It never landed there; the bytecode VM got it instead (`OpTailCall`), and the evaluator-side half was forgotten. This doc closes that 48-version-old deferral.

## Problem Statement

The bytecode compiler detects calls in tail position and lowers them to `OpTailCall`
(`internal/bytecode/compiler/call.go:192-262`, `compileReturnExpr`/`compileTailReturn`), which
reuses the current VM frame (`internal/vm/vm.go:296-349`) — "Frame stack depth does NOT grow
— that's the whole point" (`vm.go:348`). The tree-walking interpreter — the default
`ailang run` path, also used by `ailang repl`, `ailang test`, and the EvalOnly interop bridge
the VM itself calls — has no equivalent: every AILANG call nests a Go frame via
`evalCoreApp` → `e.evalCore(coreBody)` (`internal/eval/eval_operations.go:16`, body eval at
~line 200), and the `recursionDepth` counter increments once per application
(`eval_operations.go:56-61`), tail position or not.

**Consequence**: the same program succeeds on `--bytecode` and dies on the interpreter, or
vice versa needs `--max-recursion-depth` raised into the memory-hungry regime #1317 measured
(12.8 GB RSS for 2M plain-recursive calls; `changelogs/v0.32-current.md`, "#1317" entry,
2026-09-26).

**Reproduced against v0.51.0 (b99dd25) — see Verification Log V1-V4.** The reporter's
stdin-loop repro (`loop.ail`: `let line = readLine(()); if line == "" then println(show(n))
else loop(n + 1)`):

```console
$ seq 1 20000 > in.txt
$ ailang run --quiet --bytecode --caps IO --entry main loop.ail < in.txt   # prints 20000
$ ailang run --quiet --caps IO --entry main loop.ail < in.txt
Error: execution failed: RT_REC_003: max recursion depth 10000 exceeded. Try a smaller input,
an iterative std/list helper such as foldl or map instead of hand-rolled recursion, or raise
the ceiling with --max-recursion-depth
```

The divergence is **not** limited to the effectful `let`+`if` shape. Verified live, both
engines, 20,000 iterations of each tail shape:

| Tail-call shape | `--bytecode` (default) | interpreter | Verified |
|---|---|---|---|
| direct `if ... then a else f(...)` | 200010000 ✅ | RT_REC_003 at 10,000 ❌ | V2 |
| `let x = ...;` then tail `if` | 200010000 ✅ | RT_REC_003 at 10,000 ❌ | V3 |
| `match` arm body is the tail call | 200010000 ✅ | RT_REC_003 at 10,000 ❌ | V4 |

(VM coverage of all three shapes matches its compiler: `compileReturnExpr` recurses through
`IfExpr`, and surface `let`/`match` arms lower to statements whose `ReturnStmt` re-enters
`compileReturnExpr` — `compiler/call.go:207-231`, `compiler/stmt.go:96-98`. Read, not
inferred: V5.)

**Impact:**
- **stapledons-godot's sim shell** (`sim/ship.ail`) "is exactly this shape": its 10,000-tick
  replay session must run the interpreter with `--max-recursion-depth 100000` purely to
  check VM == interpreter parity. Per #1317 that ceiling is not safe much beyond ~84k
  list-pattern frames, and raising it costs gigabytes — so interpreter parity for longer
  sessions is capped by this missing optimization, not by the language.
- **AI-generated code** writes stateful read/process loops as tail recursion because AILANG
  teaches it to (the RT_REC_003 message itself says "use an iterative std/list helper such as
  foldl or map instead of hand-rolled recursion" — advice that is wrong for a stdin loop, and
  impossible for an interactive shell).
- **`ailang test --max-recursion-depth`** (added in `1c8bebee`) exists to paper over exactly
  this class of failure in recursive inline tests; tail-recursive test bodies will stop
  needing it.

## Goals

**Primary Goal:** Eliminate calls in tail position in the tree-walking interpreter with
constant stack, so the reporter's repro runs identically under `ailang run` and
`ailang run --bytecode` (20,000-line stdin loop passes both; no flag changes).

**Success Metrics:**
- `loop.ail` repro: interpreter prints `20000` at the default depth limit (today: RT_REC_003).
- Interpreter survives ≥300,000 tail-recursive iterations at default settings (reporter's
  VM figure; memory bounded, no `--max-recursion-depth`).
- Zero behavior change for non-tail recursion: `sum(5000)`, fib, factorial fixtures and the
  #1317 e2e depth tests (`cmd/ailang/deep_recursion_test.go`) pass unchanged, and
  RT_REC_003 still fires for non-tail recursion at the configured limit.
- All three verified tail shapes (if / let+if / match arm) run flat in the interpreter.
- Every existing test passes except the two that pin the OLD behavior (listed under
  "What deliberately changes"), each updated with a replacement that pins the NEW behavior
  more strongly than before.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| TCO via a trampoline at the single FunctionValue application site (`evalCoreApp`), driven by a tail-aware body evaluator — not via AST annotation or a rewrite pass | Touches the hottest path of the production evaluator; an annotation/elaboration pass would touch `internal/core`/`internal/elaborate` and every core producer | human | design | high |
| Tail-position marker is an unexported sentinel value returned by the tail evaluator and consumed only by the trampoline; it can never escape as an AILANG `Value` | A leaked marker corrupts user-visible values (show, equality, traces) in ways tests may not catch | agent | design | med |
| Tail-transparent core node set: `App`, `If` (branches), `Let` (body), `LetRec` (body), `Match` (arm body, linear path only) — mirroring the VM's verified coverage | Defines the parity contract; too small leaves divergences, too large risks mis-marking non-tail positions | human | design | high |
| Functions carrying per-invocation obligations — `Preconditions`/`Postconditions`, `EffectBudgets`/`EffectMinBudgets`, non-empty `EffectRandMode` — are EXCLUDED from TCO in M1 (fall back to the existing recursive path) | The flat trampoline cannot honor `ensures` at a tail-call boundary without unwind-time machinery; getting it wrong silently breaks contract/budget/replay semantics | human | design | high |
| Trace events stay bit-identical: one `RecordFunctionEnter` per AILANG call (per trampoline iteration); `RecordFunctionExit` for every pending call is emitted at trampoline unwind carrying the final result — which is exactly the value each call returned in the pre-TCO call tree | Traces are observable, deterministic behavior (typed trace ring); the chosen scheme makes trace output for any program that runs today identical before/after | agent | design | med |
| Eliminated calls do NOT increment `recursionDepth`; an infinite tail-recursive loop now hangs (exactly like the VM) instead of raising RT_REC_003 | This is the one deliberate semantic change: RT_REC_003's contract shrinks from "deep recursion" to "deep NON-TAIL recursion"; `TestStackOverflow` must be redesigned | human | design | high |
| RT_REC_003 message gains a remedy that now exists ("rewrite so the recursive call is in tail position"), and the test banning that phrase flips to REQUIRE the remedy to work | The ban exists because the advice "enable tail recursion" previously named a feature the evaluator lacked (rt_rec_003_message_test.go:118-139); after this doc it no longer does | agent | compile | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] Trampoline at the application site, marker-based (Decision 1-2) — no core-AST changes.
- [x] Tail-transparent node set fixed to `App`/`If`/`Let`/`LetRec`/`Match` (Decision 3).
- [x] Contract/budget/rand-annotated functions excluded in M1 (Decision 4).
- [x] Trace semantics: per-iteration Enter + unwind-time Exit with final result (Decision 5).
- [x] Depth counter semantics: eliminated calls uncounted; infinite tail loop hangs like the VM (Decision 6).
- [ ] Exact RT_REC_003 message wording (agent may propose; human approves at review).

## Deferred Decisions

The following are intentionally left open for the implementer:

- Marker type name and file placement (`internal/eval/eval_tail.go` is suggested, not required) — agent may choose.
- Whether `evalCoreTail`'s `LetRec` handling shares a body-evaluator parameter with `evalCoreLetRec` or duplicates ~40 lines — agent may choose (prefer sharing; the function is comment-dense for good reason).
- The exact shape of the exclusion predicate helper (`hasPerInvocationObligations(fn *FunctionValue) bool`) — agent may choose, but it must read exactly the five `FunctionValue` fields named in Decision 4.
- How to make the redesigned `TestStackOverflow` fast (non-tail infinite loop at a low limit) and what new flat-loop test to add beside it — agent may choose.
- DTree (`AILANG_DTREE=1`) match path: excluded from tail evaluation in M1 (falls back to `evalCore`); whether to extend later is Future Work.
- REPL/bare-expression entry (`SimpleEvaluator`, `applyFunctionAST`) gets no TCO in M1 — the `ailang run`/`ailang test`/REPL file path is `CoreEvaluator`; extending the simple evaluator is Future Work.

## Solution Design

### Overview

Give the `CoreEvaluator` a second, tail-aware evaluation mode used **only** for function
bodies. When a `FunctionValue` is applied, the evaluator evaluates its body with
`evalCoreTail` instead of `evalCore`. `evalCoreTail` walks the tail-transparent nodes
(`If`/`Let`/`LetRec`/`Match`) structurally, and when it reaches the body's final expression
being an `App` whose callee is a plain `*FunctionValue` of matching arity, it does NOT
apply it — it returns an unexported `tailCall{fn, args, name}` sentinel. The application
site (the `*FunctionValue` case of `evalCoreApp`) runs as a trampoline: it loops, consuming
one sentinel per iteration, rebinding the callee's parameters into a fresh environment, and
re-running its body — so N tail calls cost O(1) Go stack instead of N nested
`evalCoreApp` frames.

The recursion-depth guard, per-call traces, budget frames, rand-mode pushes, and contract
checks run per AILANG call exactly as today — but per **trampoline iteration** for the
eliminated calls, with trace exits emitted at unwind. Functions carrying per-invocation
obligations (contracts, budgets, rand mode) skip the trampoline entirely and take the
existing recursive path, byte-for-byte.

### Architecture

**Components** (all inside `internal/eval`; no changes to `internal/core`, `internal/vm`,
`internal/bytecode`, or `cmd/`):

1. **`tailCall` sentinel** (~15 LOC, new): unexported struct implementing `Value`
   (`Type() == "tail-call"`, `String() == "<tail-call>"`) so it can flow through the
   evaluator's `(Value, error)` plumbing. Fields: `fn *FunctionValue`, `args []Value`,
   `name string` (for the next iteration's `RecordFunctionEnter`). Constructed only by
   `evalCoreTail`; consumed only by the trampoline in `evalCoreApp`. If it ever reaches any
   other code path, that's a bug — a debug assertion in `show`/`structuralEqual` is NOT
   added (silent marker paths must be impossible by construction: the trampoline is the only
   caller of `evalCoreTail`).

2. **`evalCoreTail(expr core.CoreExpr) (Value, error)`** (~120 LOC, new file
   `internal/eval/eval_tail.go`): tail-position evaluator.
   - `*core.App`: evaluate `Func` and args with the existing non-tail machinery
     (`evalCore`, IndirectValue forcing exactly as `evalCoreApp` does at
     `eval_operations.go:23-50`). If the callee is a `*FunctionValue` AND
     `len(args) == len(fn.Params)` AND `!hasPerInvocationObligations(fn)` → return
     `&tailCall{...}`. Otherwise (builtin, constructor, curried, arity mismatch, obligated
     function, non-function) → fall through to the existing `evalCoreApp` path verbatim.
     Auto-curry and over-application keep today's semantics; only exact-arity plain calls
     are eliminable.
   - `*core.If`: evaluate the condition via `evalCore`, then `return e.evalCoreTail(branch)`
     for the taken branch — structurally identical to `evalCoreIf`
     (`eval_expressions.go:546-565`) with the final call swapped.
   - `*core.Let`: evaluate the binding via `evalCore`, bind, then `evalCoreTail(let.Body)` —
     mirrors `evalCoreLet` (`eval_expressions.go:343-360`).
   - `*core.LetRec`: reuse the binding logic of `evalCoreLetRec`
     (`eval_expressions.go:365-427`), but evaluate the body with `evalCoreTail`. Preferred
     shape: refactor `evalCoreLetRec` to take the body evaluator (or export a shared
     helper) so the indirection-cell phases stay single-sourced.
   - `*core.Match`: mirror the linear path of `evalCoreMatch` (`eval_patterns.go:17-100`):
     scrutinee and guards via `evalCore`, matched arm's bindings into a child env, then
     `evalCoreTail(arm.Body)`. When the experimental DTree path is enabled
     (`config.DTree()`), skip tail evaluation for the whole node (fall back to
     `evalCore`) — DTree is experimental and disabled by default.
   - **everything else** → `return e.evalCore(expr)`. This is the disambiguation rule: a
     call wrapped in anything other than the tail-transparent nodes (`n + f(n-1)`, a
     lambda body, a record field, a guard, a `DictApp` dispatch) is simply NOT in tail
     position and evaluates exactly as today.

3. **The trampoline** (modify the `*FunctionValue` case of `evalCoreApp`,
   `eval_operations.go:52-231`, ~+80 LOC net): restructure the existing per-call body into
   a per-iteration helper (suggested: unexported `applyFunctionValueOnce(fn, args)`) so all
   current `defer`s (depth guard decrement, budget boundary, budget frame pop, rand-mode
   pop, env/resolver restore) remain unwind-safe per iteration because the helper returns
   once per AILANG call. The `*FunctionValue` case becomes:
     1. `recursionDepth++`, guard against `maxRecursionDepth`, `defer` decrement —
        unchanged, ONCE for the whole trampoline (eliminated calls don't nest, so they
        don't count; matches the VM, whose frame stack doesn't grow on `OpTailCall`).
     2. Loop: call `applyFunctionValueOnce`. On `*tailCall` result, append the current
        call's name to a pending-name list, set `fn/args` from the sentinel, continue.
     3. On a real `Value` result, unwind: for each pending name (reverse order), emit
        `RecordFunctionExit(name, finalResult)` — the value each of those calls returned in
        the pre-TCO call tree — then return the result.
     4. Any error propagates immediately (the per-iteration defers already restored
        env/resolver/budget/rand state; no exit events on error, matching today's behavior
        where an error skips the exit event).
   The pending-name list is O(iterations) small strings; trace emission itself is
   tier-gated exactly as today (`RecordsFunctionCalls()`), so untraced runs pay one
   type-assert per iteration.

4. **Exclusion predicate** `hasPerInvocationObligations(fn *FunctionValue) bool` (~15 LOC):
   true iff `len(fn.Preconditions) > 0 || len(fn.Postconditions) > 0 ||
   len(fn.EffectBudgets) > 0 || len(fn.EffectMinBudgets) > 0 || fn.EffectRandMode != ""`
   (fields at `value.go:396-411`). True → the tail evaluator falls back to the ordinary
   application. Rationale: `requires` can still be checked per iteration, but `ensures`
   cannot (the result isn't known at a tail-call boundary), and an `@min` frame's exit check
   at a tail-call boundary is semantically ambiguous — M1 declines to answer those
   questions rather than answer them wrong. Unwind-time postcondition checking is Future
   Work.

5. **Entrypoint bodies**: no change needed — `main`/exports are applied through the same
   `evalCoreApp` path (`internal/runner/entrypoint.go` → `runtime.CallEntrypoint` →
   evaluator `CallFunction`), so the trampoline and its depth semantics apply uniformly.

### What does NOT change

- `evalCore`, the dispatch table, and every non-body evaluation: untouched.
- `applyFunction` (`eval_operations.go:652-722`, the auto-curry continuation path) and
  `SimpleEvaluator.applyFunctionAST`: untouched in M1.
- The VM, bytecode compiler, `internal/core`, lexing/parsing/typechecking: untouched.
- `--max-recursion-depth` default (10000), the stack-hop segment machinery
  (`evalSegmentLevels`, #1317): untouched — non-tail recursion still needs both.
- Effect rows, capability gates, replay contracts: untouched.

### Implementation Plan

**Phase 1: Core trampoline (M1, ~1 day)**
- [ ] Add `tailCall` sentinel + `evalCoreTail` in `internal/eval/eval_tail.go`, with the
      exclusion predicate.
- [ ] Restructure the `*FunctionValue` case of `evalCoreApp` into
      `applyFunctionValueOnce` + trampoline; preserve every existing defer's placement
      relative to its AILANG call.
- [ ] Unit tests (new `internal/eval/eval_tail_test.go`): table-driven over the three
      verified shapes at `SetMaxRecursionDepth(100)` with 1,000+ iterations (flat run
      proves elimination); non-tail recursion still trips RT_REC_003 at the low limit;
      curry fallback (over-applied tail call), builtin callee, constructor callee,
      obligated function (each takes the legacy path); mutual tail recursion
      (`isEven`/`isOdd` at 20k); `IndirectValue` self-reference forcing.

**Phase 2: Behavior pins and message (M2, ~0.5 day)**
- [ ] Redesign `TestStackOverflow` (`recursion_test.go:262-303`): today it feeds an
      infinite *tail* loop and expects RT_REC_003 — after this change that program hangs.
      Replace with (a) non-tail infinite loop (`1 + loop(n+1)` shape) still raising
      RT_REC_003, and (b) the new pin: finite tail loop 10,000+ iterations at depth limit
      100 succeeds.
- [ ] Flip `TestRTREC003DoesNotAdvertiseTailRecursion`
      (`rt_rec_003_message_test.go:121-139`): the ban exists because the advice named a
      feature the evaluator lacked. After Phase 1 it has it: replace the ban with a
      rule-3k-style execution test — the message's tail-position remedy is *followed*
      (sum rewritten tail-recursively with an accumulator at depth limit 100 succeeds).
- [ ] Update the RT_REC_003 message (`recursion_limit_error.go:18`) to name the remedy
      that now exists; keep `--max-recursion-depth` wording (its flag test still executes
      it, unchanged).
- [ ] Update the stale rationale comments in `rt_rec_003_message_test.go:14-17`
      ("tail-call machinery lives in internal/vm ... which `ailang run` does not use").

**Phase 3: End-to-end parity and docs (~0.5 day)**
- [ ] Add the reporter's `loop.ail` (stdin tail loop) as a fixture/test; run 20k lines
      through both engines, assert identical output; interpreter at 300k lines at default
      settings with a memory ceiling check.
- [ ] Run `cmd/ailang/deep_recursion_test.go` (#1317 e2e) unchanged — it must pass as-is.
- [ ] Regression sweep over the Conflict Surface fixtures (below) in both engines.
- [ ] CHANGELOG entry + `docs/` note on tail calls (interpreter and VM now agree);
      mention in RT_REC_003's docs page if one exists.

### Files to Modify/Create

**New files:**
- `internal/eval/eval_tail.go` (~200 LOC) — `tailCall` sentinel, `evalCoreTail`,
  `hasPerInvocationObligations`
- `internal/eval/eval_tail_test.go` (~250 LOC) — unit tests per Phase 1

**Modified files:**
- `internal/eval/eval_operations.go` (+~90/-~60 LOC) — `evalCoreApp` FunctionValue case →
  `applyFunctionValueOnce` + trampoline
- `internal/eval/eval_expressions.go` (+~10 LOC) — `evalCoreLetRec` shares binding phases
  with the tail variant
- `internal/eval/recursion_test.go` (+~40/-~10 LOC) — `TestStackOverflow` redesign
- `internal/eval/rt_rec_003_message_test.go` (+~35/-~15 LOC) — banned-phrase test flips to
  remedy-execution test; stale comments
- `internal/eval/recursion_limit_error.go` (~1 LOC) — message gains the tail-position remedy

**Untouched (explicitly):** `internal/vm/**`, `internal/bytecode/**`, `internal/core/**`,
`cmd/ailang/**`, `std/**`.

## Examples

**Before (verified, V1):** identical program, two answers:
```console
$ ailang run --quiet --bytecode --caps IO --entry main loop.ail < in.txt
20000
$ ailang run --quiet --caps IO --entry main loop.ail < in.txt
Error: execution failed: RT_REC_003: max recursion depth 10000 exceeded. ...
```

**After (goal state):**
```console
$ ailang run --quiet --caps IO --entry main loop.ail < in.txt
20000
$ seq 1 300000 | ailang run --quiet --caps IO --entry main loop.ail   # constant stack
300000
```

**Consumer impact (stapledons-godot):** `sim/ship.ail`'s tick loop runs 10,000+ ticks under
the interpreter at default settings for VM↔interpreter parity sessions; the
`--max-recursion-depth 100000` workaround and its #1317 memory exposure are dropped for
tail-shaped loops. Non-tail recursion in the same program keeps needing the flag — unchanged.

## Success Criteria

- [ ] Reporter's repro passes under both engines with identical stdout (20k lines)
- [ ] Interpreter handles ≥300,000 tail iterations at default settings without RT_REC_003
- [ ] All three tail shapes (if / let+if / match) run flat in the interpreter (unit-pinned)
- [ ] Non-tail recursion: RT_REC_003 still fires at the limit; #1317 e2e tests pass unchanged
- [ ] Trace output for programs that run today is bit-identical (Enter/Exit counts and values)
- [ ] Contract/budget/rand-annotated functions: legacy path, zero behavior change
- [ ] All tests passing (`make test`), including the two redesigned behavior-pin tests
- [ ] CHANGELOG + docs updated; VM/interpreter tail-call parity documented

## Conflict Surface

### Syntactic positions touched

No grammar, token, or core-AST positions are created or extended — this change is entirely
inside the `*FunctionValue` application path of `evalCoreApp`
(`internal/eval/eval_operations.go:52-231`) and a new evaluator mode for function *bodies*.
The semantic position touched: **the result position of an AILANG function body**, for the
five core node shapes listed in Decision 3.

### What else lives there

Every one of these runs per application today and must keep running per AILANG call:

| Mechanism in the application path | Where | Tail-mode handling |
|---|---|---|
| `recursionDepth` guard (RT_REC_003) | `eval_operations.go:56-61` | checked once per trampoline entry; eliminated calls uncounted (Decision 6) |
| IndirectValue forcing (LetRec self-ref) | `eval_operations.go:23-28, 364-410` | forced in `evalCoreTail` before the marker is built |
| Auto-curry (over-application) | `eval_operations.go:63-112` | marker requires exact arity; else legacy path |
| Zero-arg-export unit injection | `value.go:417-427` | unaffected (outer `CallFunction` layer) |
| Trace Enter/Exit (tier-gated) | `eval_operations.go:127-141, 218-225` | Enter per iteration; Exit at unwind with final result (Decision 5) |
| Budget boundary + `@limit`/`@min` frames | `eval_operations.go:143-155` | annotated functions excluded from TCO (Decision 4) |
| Rand-mode push/pop | `eval_operations.go:156-159` | ditto |
| Preconditions / postconditions | `eval_operations.go:176-215` | ditto |
| Env save/restore, resolver fallback | `eval_operations.go:135-174` | per iteration in `applyFunctionValueOnce` |
| Budget-frame Rand interplay under `map`/`foldl` callbacks | `stack_hop_test.go`, `#1317` callback wrappers | unchanged: callbacks are non-tail applications |

And every one of these occupies the *body result position* today without being a tail call —
each must keep its exact current semantics:

| Construct in body-result position | Why it is NOT a tail call | Handling |
|---|---|---|
| `n * f(n-1)`, `f(n-1) + g(n-2)` (BinOp-wrapped calls) | call is an operand | default case → `evalCore` (RT_REC_003 still applies) |
| lambda / curried body `\y. ...` | body is a returned *value*; its own body becomes tail only when the closure is later applied | default case → `evalCore` |
| record/array/tuple fields, `DictApp` dispatch | calls inside value construction | default case |
| builtin / constructor callee in final position | not a `*FunctionValue`; returns immediately | legacy apply path |
| match guards, scrutinee | not body-result positions | `evalCore` in `evalCoreTail`'s Match case |
| DTree match path (`AILANG_DTREE=1`) | experimental, disabled by default | whole Match falls back to `evalCore` |

### Disambiguation strategy

Purely structural, no context flags: `evalCoreTail` recurses only through
`If`/`Let`/`LetRec`/`Match` and marks only a final `App` whose callee resolves (after
IndirectValue forcing) to an exact-arity, unobligated `*FunctionValue`. Any other shape
falls to `evalCore`, which is the unchanged code path. There is no lookahead, no
parser/typechecker interaction, and no way for a non-tail position to be marked — the
marker cannot even be constructed outside the final-expression position of a function body.

### Programs that MUST still work

Regression fixtures for Phase 3 (all exist — Verification Log V10):

1. `examples/inline_tests_recursive.ail` — factorial (non-tail `n * factorial(n-1)`), fib;
   inline tests run through `ailang test` (the evaluator).
2. `examples/bounded_xml_fold.ail` — effectful iterative-style processing with match/loops.
3. `examples/url_route_dispatch.ail` — match-heavy dispatch (Match tail case).
4. `examples/first_non_repeat.ail` — string loop with lets and ifs.
5. `std/list.ail` + `tests/stdlib/bounded_take_parity_test.ail` — stdlib recursion +
   delegated iterative builtins whose RT_REC_003 exemptions (#817 iteration) must not
   shift.
6. `cmd/ailang/deep_recursion_test.go` — #1317 plain-recursive depths through `map`
   callbacks (stack-hop + counter parity must hold exactly).
7. `internal/eval/stack_hop_test.go` — counter balance (`recursionDepth`/`evalDepth`/
   `segmentBase` return to zero) after error paths, through the new trampoline.

### What deliberately changes

- **Infinite tail-recursive loops no longer raise RT_REC_003 in the interpreter — they hang,
  exactly as the VM does today** (verified: the same program runs unbounded under
  `--bytecode`). Migration: none possible at the language level (a step budget would be a
  new feature — Future Work); users who relied on the depth guard to kill runaway tail
  loops lose that safety net in interpreter mode, same as they already lost it in VM mode.
- Tail-recursive programs that previously died at the depth limit now run — by design.
- `TestStackOverflow` and `TestRTREC003DoesNotAdvertiseTailRecursion` are redesigned as
  specified in Phase 2; anything else that breaks is a regression, not intentional change.
- The RT_REC_003 message text changes to add the (now-real) tail-position remedy.

## Testing Strategy

**Unit tests** (`internal/eval/eval_tail_test.go`): per Phase 1 — shape table (if / let+if /
match / mutual recursion) at `SetMaxRecursionDepth(100)` with ≥1,000 iterations; fallbacks
(curry, builtin, constructor, obligated fn); marker never escapes (run a tail program, then
`show`/compare the result — result must be the ordinary value).

**Behavior-pin tests** (Phase 2): redesigned `TestStackOverflow` (non-tail still guards;
finite tail loop runs flat at depth 100), remedy-execution test for the RT_REC_003 message.

**Integration tests** (Phase 3): `loop.ail` fixture through both engines, byte-identical
stdout; 300k-iteration memory-bounded run; Conflict Surface fixture sweep in both engines;
`ailang test` on `examples/inline_tests_recursive.ail`.

**Regression-surface tests**: one per "Programs that MUST still work" entry, run in BOTH
engines, output-compared.

**Mutation checks** (sprint-evaluator gate): (a) delete the marker-construction branch →
flat-run tests fail with RT_REC_003; (b) make the exclusion predicate always-false →
contract/budget fixtures' expectations change; (c) count `recursionDepth` per iteration →
the flat-run-at-depth-100 test fails. Each mutant is a sole killer.

**Manual**: the reporter's stapledons-godot parity session shape (sim shell tick loop) —
10k ticks on the interpreter at default settings.

## Non-Goals

- **No TCO for contract/budget/rand-annotated functions** — M1 excludes them (Future Work:
  unwind-time `ensures`).
- **No SimpleEvaluator / REPL bare-expression TCO** — file-run and inline-test paths use
  `CoreEvaluator`; the simple evaluator is out of scope.
- **No VM or bytecode changes** — the VM already does this; it is the reference behavior.
- **No new CLI flags, no default-limit changes** — `--max-recursion-depth` keeps its
  meaning for non-tail recursion.
- **No step budget / fuel for infinite loops** — hangs become possible in interpreter mode
  exactly as in VM mode; a global step budget is a separate feature if ever demanded.
- **No DTree tail support** — experimental path, disabled by default.
- **No stack-hop removal** — #1317 machinery stays; non-tail recursion (list patterns,
  nested matches) still needs it.

## Timeline

**Day 1** (6h): Phase 1 — sentinel, `evalCoreTail`, trampoline restructure, unit tests green.
**Day 2** (6h): Phase 2 — behavior-pin redesigns, message, trace-parity verification
(`typed_trace_ring_test`, enter/exit counts on a 3-call chain vs pre-change).
**Day 3** (6h): Phase 3 — e2e parity fixtures, mutation checks, docs/CHANGELOG, sweep.

**Total: ~18 hours** (initial estimate 9h, doubled per house rules).

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Per-iteration defers subtly reorder budget/rand/env unwind vs today | High | `applyFunctionValueOnce` keeps defers inside a function that returns once per AILANG call — the defer lifetimes are structurally identical to today's; obligated functions never enter the trampoline at all |
| Trace Enter/Exit imbalance corrupts the typed trace ring | Med | Unwind-time exits are emitted before the trampoline returns; add a ring-invariant test comparing pre/post-change traces for a fixed multi-call chain (bit-identical requirement) |
| A tail-marked call changes a program that used to die — and dying was load-bearing (a timeout relied on RT_REC_003) | Med | Deliberate, documented change, identical to VM behavior; surfaced in CHANGELOG and the redesigned tests |
| `evalCoreLetRec` refactor breaks module-level binding propagation (Phase 2.5 comment block) | Med | Prefer parameterizing the existing function over duplicating it; keep `letrec` unit tests and module propagation tests in the sweep |
| Latency regression on shallow code from the extra mode dispatch | Low | `evalCoreTail` adds one type switch per body; measure fib(27) before/after (the #1317 benchmark shape) — within-noise gate |
| Mutual-recursion marker carries stale env/resolver | Med | Marker stores only `fn`+`args`+`name`; env/resolver are re-derived per iteration inside `applyFunctionValueOnce`, exactly as a fresh application would |

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Removes a nondeterministic-feeling engine divergence: same program, same answer, either engine |
| A2: Replayability | 0 | Trace events preserved bit-identically; no replay-contract changes (rand-mode functions excluded) |
| A3: Effect Legibility | 0 | No effect-row changes; the effectful read-loop case now simply *works* in both engines |
| A4: Explicit Authority | 0 | No capability changes |
| A5: Bounded Verification | +1 | Interpreter stack use for tail loops becomes input-independent (constant), instead of proportional to input length |
| A6: Safe Concurrency | 0 | Evaluator remains single-threaded; trampoline adds no goroutines |
| A7: Machines First | +1 | AI-generated tail loops are the canonical machine pattern; today they fail in the default engine with advice ("use foldl") that is wrong for interactive loops |
| A8: Minimal Syntax | +1 | No syntax; removes the need for a workaround flag (`--max-recursion-depth 100000`) in consumer configs |
| A9: Cost Visibility | +1 | Removes the gigabyte-scale memory cost #1317 measured for deep-recursion workarounds |
| A10: Composability | 0 | No new composition surface |
| A11: Structured Failure | 0 | RT_REC_003 stays for non-tail recursion; message gains a working remedy |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +4** ✅ Proceed to implementation

### Hard Violation Check

- [x] A1 (Determinism): no implicit nondeterminism introduced — trace order remains call-tree order
- [x] A3 (Effects): no hidden side effects — effectful builtins run in let-bound (non-tail) positions per iteration
- [x] A4 (Authority): no ambient access granted
- [x] A7 (Machines First): parity and stack safety for the machine-written loop idiom, not human convenience

## Verification Log

Every load-bearing claim in this doc, with the command or read that proves it. Environment:
repo at `1c8bebee` (dev), shipped binary v0.51.0 `b99dd25` (the report's build) — behavioral
claims were verified against the shipped binary; structural claims against the repo.

| # | Claim | Verification |
|---|---|---|
| V1 | Reporter's repro diverges: bytecode prints 20000, interpreter dies RT_REC_003 at 10,000 | Ran verbatim against `b99dd25`: `ailang run --quiet --bytecode --caps IO --entry main loop.ail < in.txt` → `20000`; same minus `--bytecode` → `RT_REC_003: max recursion depth 10000 exceeded` (full transcript in Problem Statement) |
| V2 | Direct-if tail shape: VM flat, interpreter RT_REC_003 | `tcolet`-style minus let: `if x == 0 then acc else go(x-1, acc+x)` over 20,000 iterations → bytecode `200010000`, interpreter RT_REC_003 |
| V3 | `let` + tail `if` shape (reporter's exact shape, pure) | `let x = n; if x == 0 then acc else go(x - 1, acc + x)` → bytecode `200010000`, interpreter RT_REC_003 |
| V4 | `match`-arm tail shape | `match if n == 0 then None else Some(n) { None => acc, Some(k) => classify(k-1, acc+k) }` → bytecode `200010000`, interpreter RT_REC_003 |
| V5 | VM TCO is compile-time tail detection + frame reuse | Read `internal/bytecode/compiler/call.go:192-262` (`compileReturnExpr` recurses IfExpr; ReturnStmt path at `stmt.go:96-98`), `internal/vm/vm.go:296-349` (`OpTailCall` reuses frame; "Frame stack depth does NOT grow — that's the whole point") |
| V6 | The interpreter has no tail-call machinery (negative existence) | `grep -rn -i "tail" internal/eval/*.go` → only list-tail/pattern-tail mentions plus comments stating the machinery lives in `internal/vm`/`internal/bytecode` (`rt_rec_003_message_test.go:14-17, 118-119`) |
| V7 | Single application site + per-call machinery inventory | Read `internal/eval/eval_operations.go:16-231` (`evalCoreApp`: depth guard 56-61, trace enter 127-141, budget/rand 143-159, contracts 176-215, body eval ~200) and `:652-722` (`applyFunction`, curry continuation only) |
| V8 | FunctionValue carries the obligation fields for the exclusion predicate | Read `internal/eval/value.go:396-411`: `EffectBudgets`, `EffectMinBudgets`, `EffectRandMode`, `Preconditions`, `Postconditions` |
| V9 | Two tests pin the OLD behavior and must change | Read bodies: `TestStackOverflow` (`recursion_test.go:262-303`, infinite *tail* loop expects RT_REC_003), `TestRTREC003DoesNotAdvertiseTailRecursion` (`rt_rec_003_message_test.go:121-139`, bans "tail call"/"tail recursion" in the message with rationale "this evaluator has no tail-call elimination") |
| V10 | Regression fixtures exist | `ls`: `examples/inline_tests_recursive.ail`, `examples/bounded_xml_fold.ail`, `examples/url_route_dispatch.ail`, `examples/first_non_repeat.ail`, `std/list.ail`, `tests/stdlib/bounded_take_parity_test.ail`, `cmd/ailang/deep_recursion_test.go` — all present |
| V11 | #1317 stack-hop facts (12.8 GB @ 2M calls, ~84k list-pattern crash, 439-751 B/level) | Read `internal/eval/eval_expressions.go:11-27` and `changelogs/v0.32-current.md` "#1317" entry (2026-09-26) |
| V12 | VM has a frame-stack guard distinct from the interpreter's counter | Read `internal/vm/vm.go:11-13` (`DefaultMaxStack = 1000`), `:41-42`, `:112-113`, `:279-280` — note: this repo HEAD may differ from the shipped `b99dd25` binary; behavior-relevant claims in this doc rest on V1-V4, not on the VM's exact constant |
| V13 | `ailang run --bytecode` can silently fall back to the evaluator in non-strict mode (pre-existing, out of scope) | Read `internal/runner/entrypoint.go:144-158` ("falling back to evaluator") and `runner/vm.go:133-141`; observed `--strict-bytecode` on the repro failing with the *evaluator's* RT_REC_003, confirming the two-layer dispatch |
| V14 | `ailang test` runs inline tests through the evaluator with a `--max-recursion-depth` flag | Read `cmd/ailang/commands_language.go:240`; `git log -1 1c8bebee` = "feat(test): ailang test --max-recursion-depth, matching ailang run" |
| V15 | M-R4 deferred evaluator TCO in v0.3.0 | Read `design_docs/implemented/v0_3_0/M-R4_recursion.md` ("Tail-call optimization (deferred to v0.4.0)") |
| V16 | No new error code is proposed (namespace check n/a) | This doc introduces no `MOD/PAR/TC/EFF` code; the only message change is RT_REC_003 prose |

## References

- **Bug context**: coordinator task `task-40abb254` (v0.51.0 `b99dd25`, darwin arm64); consumer `stapledons-godot` `sim/ship.ail`
- **Issue**: #1317 (`--max-recursion-depth` vs Go's stack; e2e tests in `cmd/ailang/deep_recursion_test.go`)
- **Prior art in this repo**: [M-R4_recursion](../../implemented/v0_3_0/M-R4_recursion.md) (v0.3.0, deferred this work); `OpTailCall` design in [v0.10-v0.17 bytecode-vm changelog](../../../changelogs/v0.10-v0.17-bytecode-vm.md)
- **Adjacent machinery**: #1317 stack-hop (`eval_expressions.go:11-60`), [m-iterative-list-builtins](../../implemented/v0_9_2/m-iterative-list-builtins.md) (the foldl/map advice RT_REC_003 gives)
- **Axiom reference**: [Design Axioms](/docs/references/axioms)

## Future Work

- **Unwind-time `ensures`**: check postconditions of tail-calling functions against the
  trampoline's final result at unwind, extending TCO to contract-annotated functions.
- **`@min`/`@limit` budget frames across tail calls**: define charge-boundary semantics for
  eliminated calls, then admit budgeted functions.
- **SimpleEvaluator/REPL bare-expression parity** if a REPL-session tail loop ever bites.
- **Step budget / fuel** (opt-in), if runaway tail loops in interpreter mode ever become an
  operational problem — would apply to both engines for parity.
- **DTree match path tail support** if DTree ever leaves experimental.
