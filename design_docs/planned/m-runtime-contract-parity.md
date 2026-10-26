# M-RUNTIME-CONTRACT-PARITY: one runtime contract semantics, checked against the verifier

**Status**: Planned
**Target**: v0.55.x (M1 can ship in the next patch release on its own)
**Priority**: P1. A safety flag silently does nothing on the VM (#1956). A contract that `ailang verify` proves crashes every runtime check of it (#1954, #1955). A shipped example does both at once: `examples/runnable/contracts/quantifier_verify.ail`.
**Estimated**: ~10 working days across 7 milestones. M1 takes half a day and stands alone. M2 and M3 are about a day each and fix the two reported crashes.
**Dependencies**: None blocking. It must be sequenced with [m-adt-equality-position-independence.md](m-adt-equality-position-independence.md), which also proposes type-checking contract predicates (see Design Freeze F2).
**Planner-Lane**: opus-required. It touches `internal/types`, `internal/pipeline`, `internal/eval`, `internal/testing`, `internal/gen/lower` and `internal/bytecode/compiler`, and it changes what `ailang check` accepts.
**Tracks**: [#1954](https://github.com/sunholo-data/ailang/issues/1954), [#1955](https://github.com/sunholo-data/ailang/issues/1955), [#1956](https://github.com/sunholo-data/ailang/issues/1956)
**Verified against**: origin/dev `7d29031ea` (2026-10-26, `std/VERSION` v0.54.18). Binary: a build of origin/dev. Every repro below was run on it. Transcripts are in the [Verification Log](#verification-log).
**Lane** (PROGRAM.md): **AILANG fix (core-floor)**. Contract semantics are language semantics. No extension can repair a VM that never reads contracts.
**Quorum**: trigger 1 (design-freeze items F1–F4) and trigger 2 (it replaces the evaluator's and the test harness's contract-evaluation paths, which are shared machinery) both fire, so the quorum is **required before sprint planning**. It was skipped for this draft at the requester's instruction.

---

## Problem Statement

Three open issues report three symptoms:

| Issue | Symptom | Where it breaks |
|---|---|---|
| #1954 (P1) | `ensures { result == best }` on a `deriving (Eq)` record. `ailang verify` proves it. `run --verify-contracts` and `ailang test` fail: `derived Eq for Run: expected TaggedValue, got *eval.RecordValue` | `internal/eval/eval_equality.go:23-28` |
| #1955 (P2) | `ensures { forall i: 0..length(result) => ... }`. `ailang verify` proves it. `ailang test` and `run --verify-contracts` fail: `core evaluation not implemented for *core.Forall` | `internal/eval/eval_expressions.go:164-165` |
| #1956 (P1) | `run --bytecode --verify-contracts` prints `-5` and exits 0 on a violated `ensures`. The evaluator reports the violation. | `internal/runner/run.go:533-538` installs `effCtx.Contracts`, and nothing on the VM path reads it |

### Root cause: one contract language, two semantics, and no seam check

There is one contract **language**: a predicate is any expression (`internal/parser/parser_contracts.go:126-148`, `predicate ::= expression`). It has two **semantics**:

1. **The verifier** (`internal/smt`). It gives the meaning by encoding a predicate into SMT-LIB. It has its own case for `*core.Forall` (`internal/smt/codegen_expr.go:110`), its own gates (`encodable.go:44-108`), and it reports VERIFIED, SKIPPED, ERROR or VIOLATION.
2. **Runtime evaluation**. It is split across four consumers, and each one rebuilds the meaning on its own:
   - **The evaluator under `run --verify-contracts`.** `runtime.attachContracts` (`internal/runtime/runtime.go:572-590`) copies the raw `Contract.Expr` onto the `FunctionValue`. `checkPreconditions`/`checkPostconditions` (`internal/eval/eval_evaluator.go:568-656`) evaluate it in the callee's environment, after `e.env.Set("result", …)` (`:620`).
   - **The test harness.** It wraps the same raw expression in hand-built `Let` nodes (`internal/testing/harness.go:215-311`) and evaluates it with a fresh evaluator (`executor.go:155-166`).
   - **The bytecode VM.** It has **no consumer at all**. `grep -rn Contract internal/vm internal/bytecode internal/gen/lower` finds nothing. The lowering pass erases a `core.Forall` to the literal `true` (`internal/gen/lower/expr.go:120-122`).
   - **Go codegen** (`compile --emit-go --verify-contracts`). It has its own predicate printer (`internal/gen/golang/contracts.go:334-440`).

**No contract predicate gets a static semantics.** That is where the two sides fall apart:

- **Type inference over a predicate is best-effort, and its error is thrown away.** See `internal/pipeline/pipeline_module_compile.go:269-276`: *"Errors are non-fatal: contract type-checking is best-effort for CoreTI population."* So `ensures { result == "x" }` on an `int` function passes `ailang check` (V7).
- **The type checker has no rule for `core.Forall`** (`internal/types/typechecker_core.go:771-772`, default case). Inference therefore fails for **every** forall contract, and that failure is discarded too.
- **OpLowering and dictionary elaboration pass `core.Forall` through untouched**: `internal/pipeline/op_lowering.go:296-298` and `internal/elaborate/dictionaries.go:315-318` both use `default: return expr`. A forall body keeps raw `Intrinsic` operators. The evaluator only rescues comparisons on those (`eval_operations.go:251-272`). Arithmetic still fails (`:274`).
- **No pass checks a predicate's effects.** A `pure` function's `ensures` can call `println`. `ailang check` accepts it, and `run --verify-contracts` **performs the IO** (V8).
- **No pass rejects a construct the runtime cannot evaluate.** Each gap therefore shows up late and differently on each consumer: a crash in the evaluator, a silent pass on the VM, a Go compile error in codegen.
- **No test runs the same contract through both semantics.** The verifier can report VERIFIED for a contract that no runtime can evaluate. `examples/runnable/contracts/quantifier_verify.ail` has shipped in exactly that state: `verify` → 3 VERIFIED, `run --verify-contracts` → crash, `run --bytecode --verify-contracts` → `41`, exit 0 (V4).

This is CLAUDE.md's "verify the seam" case. The bug lives where one concept, the meaning of a contract, has two implementations and nothing compares them. The history shows the seam being patched one symptom at a time: M-CONTRACTS-OPLOWERING (v0.8.0), M-CONTRACT-OPLOWERING-FIX (`f9f480c74`), M-DX26 Phases 5, 5.1 and 5.2 (`4a21d1959`, `43c85649f`, `3ebf60b1b`), #547 (`a9e26ffd6`). Each fixed one consumer for one construct. None added a guard.

### #1954 is not contract-specific; contracts just always hit it

The triage comment on #1954 says only the contract path fails, and that ordinary `==` works. That is true only for the control it used (a `let`-annotated binding). **The same comparison in an ordinary function body fails when both operands are typed by a parameter annotation** (V3):

```ailang
export type Run = { size: int, end: int } deriving (Eq)
export pure func same(a: Run, b: Run) -> bool = a == b      -- evaluator: same crash
```

The evaluator crashes on this body. The VM prints `true`. One operand being a record literal works (`a == { size: 1, end: 2 }`). So does a `let a: Run = …` binding.

**Mechanism (from reading the code; matches all four probes):**
- `registerDerivedEqInstances` (`internal/pipeline/derived_eq.go:57-76`) registers every `deriving (Eq)` type, **record aliases included**, as a nominal `Eq[TCon T]` instance with the `DerivedADTEquality` marker.
- `InstanceEnv.lookup` (`internal/types/instances.go:55-60`) tries a direct key match first. An operand typed `TCon "Run"` therefore hits that nominal instance, and its dictionary resolves to `makeADTEqualityFn`, which accepts only TaggedValues.
- An operand typed `TRecord{TypeName: "Run"}` misses the direct key. It falls through to `synthesizeEq` (`instances_eq_synth.go:74-81`), which returns a `Structural` instance and so the `StructuralEq` dictionary (`typechecker_substitution.go:493-494`).
- The VM never sees the difference: lowering turns every `Eq.eq` dictionary call into a structural `OpEq` (`internal/gen/lower/expr.go:656-662`).

Contracts always hit the bad path because the contract environment binds every parameter and `result` from the function's **declared signature** (`pipeline_module_compile.go:247-264`), and those are nominal `TCon`s. So #1954 is an evaluator-vs-VM Eq seam that the contract position makes certain to trigger. M2 fixes it for bodies and contracts alike.

**Impact:**
- **Users who write contracts.** The issue reporter hit all three while contracting `sunholo/explainer`. The workaround was to rewrite contracts so the runtime could evaluate them: compare fields one by one, avoid `forall`. That makes the contracts weaker than what the verifier can prove.
- **`bestof`/`select-best`.** It ranks candidates on `ailang run --verify-contracts` exit status (`internal/bestof/bestof.go:138-148`). A predicate that crashes exits 1, the same as a real violation. So a **correct** candidate with a record-equality or forall contract is ranked as contract-failing.
- **Contracts the verifier skips**: Array/Map parameters, HOFs, recursion, tuples. They can only be checked at runtime, and per-sample code needs the VM to be fast enough. The issue measured the VM at ~50×; this doc measured 35× (2.10 s vs 0.06 s on `perfc.ail`, V10). But `--bytecode --verify-contracts`, the one combination that would check them, checks nothing.

## Capability matrix

**Method.** 24 one-construct probe files (`c01`–`c24`, sources in the Verification Log), plus the three issue repros. Each probe was run through:
- `ailang check`
- `ailang verify` (SMT)
- `ailang test` (the property harness, which uses evaluator semantics)
- `ailang run --verify-contracts` (the evaluator)
- `ailang run --bytecode --verify-contracts` (the VM)
- `ailang compile --emit-go --verify-contracts` (Go codegen; shown for information, out of scope here)

Every probe's `main` calls the function with an input that satisfies the contract, except `bcviol`, which violates it on purpose.

**Legend:**
- ✓ = evaluated correctly. **V** = SMT VERIFIED. **S** = SMT SKIPPED. **E** = SMT ERROR.
- **CRASH** = runtime evaluation error, exit 1, reported like a violation.
- **NOT CHECKED** = the run succeeds and the contract is never evaluated.
- **SILENT-ACCEPT** = `ailang check` accepts an invalid contract.

| # | Construct in `requires`/`ensures` | `check` | SMT | Harness (`ailang test`) | Evaluator (`run --verify-contracts`) | VM (`--bytecode --verify-contracts`) | Go codegen (info) |
|---|---|---|---|---|---|---|---|
| c01 | int arithmetic + comparison | ✓ | V | ✓ | ✓ | NOT CHECKED | ✗ `undefined: result` |
| bcviol | violated `ensures { result > 0 }` | ✓ | VIOLATION | ✓ reports violation | ✓ reports violation | **NOT CHECKED: prints `-5`, exit 0** (#1956) | ✓ emits check |
| c02 | `not`, `\|\|`, `==` on bool | ✓ | V | ✓ | ✓ | NOT CHECKED | ✗ |
| c03 | `if … then … else` | ✓ | V | ✓ | ✓ | NOT CHECKED | ✗ |
| c04 | `let … in` | ✓ | V | ✓ | ✓ | NOT CHECKED | ✗ |
| c05 | `match` on ADT | ✓ | V | ✓ | ✓ | NOT CHECKED | ✗ |
| c06 | record field access | ✓ | V | ✓ | ✓ | NOT CHECKED | ✗ |
| c07 / receq | **derived Eq, nominal record** | ✓ | V | **CRASH** (#1954) | **CRASH** (#1954) | NOT CHECKED | ✓ |
| c19 | derived Eq vs record literal | ✓ | V | ✓ | ✓ | NOT CHECKED | ✗ |
| c08 | derived Eq, enum ADT | ✓ | V | ✓ | ✓ | NOT CHECKED | ✓ |
| c09 | list `==` | ✓ | V | ✓ | ✓ | NOT CHECKED | ✗ |
| c10 | tuple `==` | ✓ | S (tuple sort) | ✓ | ✓ | NOT CHECKED | ✗ |
| c11 | `Option` `==` | ✓ | V | skipped (no generator) | ✓ | NOT CHECKED | ✓ |
| c12 | call to a user pure function | ✓ | V | ✓ | ✓ | NOT CHECKED | ✗ |
| c13 | stdlib builtin (`length`) | ✓ | V | ✓ | ✓ | NOT CHECKED | ✗ |
| c14 | string interpolation | ✓ | E (`show` unencodable) | ✓ | ✓ | NOT CHECKED | ✗ |
| c15 | HOF + lambda (`any(\x. …)`) | ✓ | S | ✓ | ✓ | NOT CHECKED | ✗ |
| c16 / forallrt | **bounded `forall`, builtin body** | ✓ | V | **CRASH** (#1955) | **CRASH** (#1955) | NOT CHECKED (and lowering erases `Forall` to `true`) | ✗ unsupported `*core.Forall` |
| c17 | **bounded `forall`, arithmetic body** | ✓ | V | **CRASH** | **CRASH** | NOT CHECKED | ✗ |
| c24 | `not (forall …)` (the only way to say "exists") | ✓ | V | **CRASH** | **CRASH** | NOT CHECKED | — |
| c18 | float comparison | ✓ | V | ✓ | ✓ | NOT CHECKED | ✓ |
| c20 | **effectful call** (`println`) in a `pure` function's `ensures` | **SILENT-ACCEPT** | S | CRASH (`IO requires capability`) | **performs the IO**, then passes | NOT CHECKED | ✗ |
| c22 | **ill-typed** (`result == "x"`, `result: int`) | **SILENT-ACCEPT** | E (Z3 sort error) | CRASH (`eq_Int: expected IntValue`) | CRASH | NOT CHECKED (passes) | ✗ |
| noeq | `==` on an ADT without `Eq` | **SILENT-ACCEPT** (a body is rejected) | VIOLATION (correct) | — | ✓ (structural) | NOT CHECKED | — |
| c21 | `exists i: lo..hi => …` | parse error: `exists` is reserved, not in the grammar | — | — | — | — | — |
| c23 | `deriving (Ord)` comparison | parse error: `PAR_DERIVING_UNSUPPORTED` | — | — | — | — | — |

**Gap count: 22 probe constructs parse and pass `ailang check` as contracts** (c01–c20, c22, c24). That `ailang check` accepts two of them is itself a gap (G3).

- **SMT on its own:** 17 V, 3 S (c10, c15, c20), 2 E (c14, c22).
- **G1. VM checks none of the 22 constructs (0/22).** The VM consumes no contracts. It also lowers `Forall` to `true`, so a naive fix would still pass silently. This is #1956.
- **G2. The verifier proves 4 probes that every runtime crashes on.** c07 (record Eq), and c16, c17, c24 (bounded forall). These are #1954 and #1955. They are the dangerous direction: a VERIFIED badge on a contract nothing can check.
- **G3. `ailang check` silently accepts 3 invalid contracts.** c20 is effectful, c22 is ill-typed, and noeq has no `Eq` instance (the same expression is rejected in a body). The cause is that predicate inference errors are discarded, and no pass checks a predicate's effects.
- **G4. The evaluator and the harness disagree on c20.** The evaluator runs the IO. The harness crashes on capability.
- **G5. Neither runtime can tell a predicate that crashes from a predicate that is false.** Both exit 1 (A11). This is the `bestof` misranking above.
- **G6. Latent silent skips.** `checkPreconditions` and `checkPostconditions` silently `continue` past a predicate that is not Core (`eval_evaluator.go:583`, `:630`). `gen/lower` turns `Forall` into `true`.
- **Out of scope here: Go codegen fails to compile 17 of the 21 probes swept (c01–c20, c22) under `--verify-contracts`.** The failure is loud (it never builds), so it does not break the never-silent rule. It is a fifth semantics, listed for completeness. It belongs in a follow-up issue (see [Non-Goals](#non-goals)).

Totals: **5 gap classes (G1–G5) plus 2 latent silent paths (G6)**. In cells, over the 22 constructs × 3 runtime columns (harness, evaluator, VM) = 66 cells: **34 are wrong**: 22 VM NOT CHECKED, 8 harness/evaluator CRASH on contracts the verifier proved (c07, c16, c17, c24), and 4 harness/evaluator cells on c20 and c22, which `ailang check` should have rejected (one performs IO, three crash). Separately, the `check` column silently accepts 3 invalid contracts (c20, c22, noeq). Against that, the verifier is honest about what it skips.

## Goals

**Primary goal:** every contract has one runtime meaning. It is computed by one code path shared by `run`, `ailang test` and the VM. It is checked statically before it can run. And a test fails the build whenever the verifier and the runtime disagree about which constructs a contract may contain.

**Success metrics:**
- **Engine coverage.** Today the VM checks 0/22 contract constructs. Target: all valid ones (20 after M4 rejects c20 and c22), with identical outcomes on evaluator and VM for every probe and every corpus contract (M7).
- **"Verified but not runtime-evaluable."** 4 probes and 1 shipped example today; 0 after, enforced by the parity test (M5).
- **Invalid contracts accepted by `ailang check`.** 3 classes today (ill-typed, effectful, Eq-less); 0 after, each with a typed `CTR` diagnostic (M4; timing per F2).
- **Silent paths.** 3 today (VM NOT CHECKED, the non-Core `continue`, `Forall`→`true`); 0 after. `--bytecode --verify-contracts` refuses after M1 and enforces after M7.
- **VM speed with contracts on.** Within 2× of the VM with contracts off, and at least 15× faster than the evaluator with contracts on. Measured on `perfc.ail` (see [Risks](#risks--mitigations)).

## High-Impact Decisions

| Decision | Why high impact | Chosen by | Deadline | Change cost |
|---|---|---|---|---|
| D1. **VM enforcement mechanism**: compile contracts into bytecode as synthesized checker functions | Decides VM perf under contracts and whether VM and evaluator share one semantics | human (F1) | design | high |
| D2. **One runtime path**: every contract becomes a typed, lowered **checker declaration**, run identically by evaluator, harness and VM. The raw-expression evaluation paths are deleted | Replaces shared machinery in `eval` and `testing` | human (F1) | design | high |
| D3. **Contract predicates get full static semantics**: fatal type inference, purity check, runtime-evaluability check, all as `CTR` diagnostics | Changes what `ailang check` accepts. Can break published packages | human (F2) | design | high |
| D4. **`forall` stays contract-only**: a type rule only in contract context; a body-position `forall` gets a real diagnostic | Conflict surface: `forall` parses in any expression position today | agent (recommended below) | compile | low |
| D5. **Record Eq fix lives in the evaluator**: `makeADTEqualityFn` falls back to structural comparison for non-TaggedValue operands. The instance table is unchanged | Fixes ordinary bodies too. The VM and Go codegen already compare structurally | agent | compile | low |
| D6. **M1 refuses `--bytecode`/`--strict-bytecode` together with `--verify-contracts`** (including `--batch`) before anything executes | Behaviour change for anyone passing both flags today (who gets no checking anyway) | human (F3) | design | low |
| D7. **Parity rule**: SMT-encodes ⇒ runtime-evaluable is a **hard failure**. Runtime-evaluable ∧ ¬SMT-encodes is **ledgered** in a golden file, not failed | Decides what the parity test can block | human (F4) | design | med |

### Design Freeze

- [ ] **F1. VM enforcement = compiled checker functions (D1, D2).** Recommended. See [Alternatives](#alternatives-considered) for routing to the evaluator and for VM→evaluator callbacks.
- [ ] **F2. When contract type errors become fatal, and in what order relative to [m-adt-equality-position-independence](m-adt-equality-position-independence.md).**
  - That doc rejects "type-check contracts only, keep requiring `deriving (Eq)`" as friction-increasing. This doc's M4 would do exactly that for `noeq`-shaped contracts unless the compiler-provided ADT Eq lands first.
  - Recommended:
    1. M4 lands CTR001–CTR003 as **warnings** for one release, then measures the blast radius (see [Risks](#risks--mitigations)).
    2. They become errors in the following release.
    3. If m-adt-equality has not landed by then, CTR001 for a missing `Eq` instance alone stays a warning until it does.
  - Ownership: this doc owns contract-predicate type-checking (the "missing half" item in m-adt-equality §Goal 2). That doc keeps the compiler-provided ADT Eq ruling and the `std/prelude` hint fix.
- [ ] **F3. M1's refusal is an error, not a warning plus fallback to the evaluator.** Recommended: error (exit 2, `CTR004`). The user asked for the VM. Silently running 35× slower is a surprise of its own, and an automatic fallback would also hide the VM gap that M7 must close.
- [ ] **F4. Parity-test policy (D7).** Recommended:
  - **Hard failure:** any construct the verifier encodes but the runtime cannot evaluate.
  - **Hard failure:** any evaluator-vs-VM outcome difference.
  - **Golden ledger, reviewed in diff:** runtime-only constructs, and known model divergences (unbounded `Int` in Z3 vs `int64` at runtime).

## Solution Design

### Overview

The contract pipeline becomes:

```
surface predicate
  → elaborate (unchanged; Forall stays one Core node, also used by SMT)
  → STATIC CONTRACT CHECK (new, M4)
      type inference in contract env (fatal), purity, runtime-evaluability whitelist
      → CTR001 / CTR002 / CTR003
  → dictionary elaboration + OpLowering, now recursing into Forall (M3)
  → two consumers of the SAME lowered Core:
      ├─ SMT encoder (unchanged input)
      └─ CHECKER DECLARATION (new, M6): __contract_<fn>_<k>(p1..pn [, result]) -> bool
           the standard post-typecheck phases → an ordinary Core function
           ├─ evaluator:  applyFunction calls it (replaces raw-expr eval + env.Set("result"))
           ├─ harness:    calls it with generated args (replaces Let-wrapping)
           └─ VM:         compiled like any function; contracted functions get a
                          prologue/epilogue only in images built with contracts on (M7)
```

The predicate's meaning is fixed once, as typed and lowered Core. Each runtime only has to know how to call a function returning `bool`, which all of them already do, and which the existing evaluator↔VM parity machinery already covers (`internal/testing/bytecode_engine.go`, `engine_parity_test.go`). There is precedent for synthesizing a function and pushing it through the normal pipeline so the VM can compile it: `__namedtest_entry` (`bytecode_engine.go:10-22`). That file also explains why property harnesses are evaluator-only today: they are *"Core built in Go with no surface source or type info to lower"*. M6 removes that reason.

### Runtime contract semantics (frozen with F1)

These are the evaluator's observable rules today, pinned so the VM must match them:

- **Every call is checked, including recursive and tail calls.** `requires` runs on the arguments before the body. `ensures` runs on arguments plus `result` after the body returns. The fixture `tailens.ail` (an `ensures` on a function whose body is a tail call) is reported as a violation by the evaluator today (V11). The VM must not let `OpTailCall` skip the epilogue.
- **`result` shadows a parameter named `result` inside `ensures`** (V12, current behaviour, kept). The checker binds `result` as its last parameter.
- **Panic mode** (`effects.ContractModePanic`): the first failure aborts with the existing `contract violation: …` error.
- **New: a predicate that raises** (for example `_list_nth` out of range in a forall body) aborts with **CTR005 "contract predicate raised"**, not a violation. Exit code stays 1. The message and the JSON error code differ, so `bestof` can tell the two apart (G5).
- **New: the violation message names the function.** Today it is the empty `ensures failed in  at …`; the checker knows its owner.
- **Bounded `forall i: lo..hi => p`** means: `lo` and `hi` are evaluated once, as `int`; `p` is evaluated for `i = lo … hi-1` in ascending order and short-circuits on the first `false`. An empty range is `true`. This matches the SMT encoding (`internal/smt/codegen_quantifier.go`) over the same half-open interval. Runtime cost is O(hi−lo) evaluations of the body. That cost is the contract author's to own and is documented, not capped. A cap is a deferred decision.

### Milestones

**M1: refuse `--bytecode` + `--verify-contracts` (#1956, ~0.5 d).** In `runner.Run` (`internal/runner/run.go`, before `:533`) and the batch path (`batch.go:77`):
- If `opts.VerifyContracts && opts.BytecodeMode` (`BytecodeMode` already folds in `--strict-bytecode`, see `cmd/ailang/main_run_exec.go:128`), exit 2 **before compiling or executing** with:

```
Error: CTR004: --verify-contracts is not enforced on the bytecode VM yet (runtime contracts
run only on the evaluator). Drop --bytecode to check contracts, or drop --verify-contracts
to run on the VM. Tracking: #1956.
```

- Register `CTR004` in `internal/errors/codes.go`.
- Unit test both entry points.
- Changelog fragment.
- This stays true until M7 lifts it.

**M2: derived-Eq parity for nominal records (#1954, ~0.5 d).**
- In `makeADTEqualityFn` (`internal/eval/eval_equality.go:16-43`), when either operand is not a `*TaggedValue`, compare with `valuesStructurallyEqual` (`:89`), the function every structural instance already uses.
- Keep the `rejectFunctionEquality` guard.
- `taggedValuesEqual` is a special case of structural equality, so results for ADTs do not change.
- Do **not** touch the instance table (D5). Changing `registerDerivedEqInstances` to mark record aliases `Structural` would rename the Go codegen dictionary (`dict_Eq_Run`), which c07 shows working today.
- Fixtures (each on both engines):
  - `receq` (contract)
  - `receq5` (body, two `Run` parameters)
  - `receq3` (a `Run` value through a function, then compared)
  - controls `receq6` and `receq7`
- Also audit the other synthesized shapes in contract position. c08 (ADT), c09 (list), c10 (tuple), c11 (Option) and `Result` (V13) already pass on both engines; pin them as regression rows.

**M3: bounded `forall` at runtime (#1955, ~1.5 d).**
1. Add a type rule for `*core.Forall` in `internal/types/typechecker_core.go`: `lo : int`, `hi : int`, `Var : int` in scope for `Body : bool`, result `bool`. It is active **only** when the checker is inferring a contract predicate. A `forall` in a body keeps failing, but with `CTR006: forall is only valid in requires/ensures` instead of `type inference not implemented for *core.Forall` (D4).
2. Make `OpLowerer.lowerExpr` (`internal/pipeline/op_lowering.go`) and `DictElaborator.transformExpr` (`internal/elaborate/dictionaries.go`) recurse into `Lo`, `Hi` and `Body`. Today both fall to `default: return expr`.
3. Add a `*core.Forall` case to `CoreEvaluator.evalCore` (`internal/eval/eval_expressions.go`) with the semantics above.
4. Replace the `Forall → LitBool{true}` arm in `internal/gen/lower/expr.go:120-122` with a **panic** naming the node. The VM never sees a `Forall` until M7, and `true` is the one wrong answer that must never ship silently.

After M3, c16, c17, c24, `forallrt` and `quantifier_verify.ail` pass on `ailang test` and `run --verify-contracts`. They still refuse under `--bytecode` (M1).

**M4: static contract check (~2 d).** A new `internal/pipeline/contract_check.go` runs after the module type check (`pipeline_module_compile.go:233-280` today). For each predicate:
- **CTR001: type error.** Make the inference at `:273` report instead of discarding, with the existing type-error formatting. This includes `Eq`/`Ord`/`Num` instance resolution, so `noeq` and `c22` are caught.
- **CTR002: effectful predicate.** The inferred effect row of a predicate must be empty. A contract is a specification, so it may not exercise authority, even inside an effectful function.
- **CTR003: construct with no runtime semantics.** A whitelist walk over the lowered Core. It is an exhaustive switch over the 22 `core.CoreExpr` node types (`internal/core/core.go`), so adding a node type without classifying it fails to compile (a `default` that panics in tests). After M3, every node the verifier encodes is on the list. CTR003 exists so the next verifier-only construct (for example a future `exists`) fails at `ailang check`, not at runtime.
- **CTR005 and CTR006** are registered here too.

Severity follows F2: warnings for one release, then errors. `ailang check --json` and `ai-check` carry the codes. The memory note about `ai-check` stands: today an unimported name in a contract surfaces only as a Z3 "unknown constant" verify error. After M4 it is a CTR001 at check time.

**M5: parity test (~1.5 d).** Three layers.

- **Static, exhaustive** (`internal/contracts/parity_test.go`, or under `internal/smt` if a new package is unwanted; agent's call). For each of the 22 Core node types, assert that `smt.EncodesNode` and `pipeline.RuntimeEvaluable` give a classification. Fail if SMT encodes a node and the runtime does not.
- **Corpus** (`make test-contract-parity`, added to `make ci`). For every contract in `examples/` and `std/`:
  - Today that is 46 files and roughly 216 one-line `requires`/`ensures` blocks, by grep (V14).
  - Plus the probe corpus, checked in as `examples/runnable/contracts/parity/c*.ail`.
  - For each, record:
    - the SMT outcome per function;
    - the static check outcome per predicate;
    - for every function that has a property harness, the harness outcome on 50 seeded inputs.
  - **Hard failures:**
    - (a) SMT VERIFIED, but the static check reports CTR003 or the runtime raises CTR005 on in-contract inputs;
    - (b) SMT VERIFIED, and the runtime finds a violation on a seeded input, unless that function is listed in the ledger with a reason (Z3 `Int` vs runtime `int64` overflow is the only class expected).
  - **Ledgered** in `testdata/contract_parity.golden`, so changes show up as a reviewed diff: SMT SKIPPED or ERROR while the runtime evaluates (c10, c14, c15 today).
- **Engines.** After M7, the corpus layer runs every harness case on the evaluator and on the VM and fails on any outcome difference. It reuses the `TestEngineParity_*` pattern from `internal/testing/engine_parity_test.go`.

The parity test lands **before** M6 and M7, so the refactor is measured against it.

**M6: one runtime path, the checker declarations (~2 d).**
- Synthesize, per contracted function `f` and clause `k`, a private top-level decl named `__contract_<f>_<k>`:
  - `requires` clause: `\p1 … pn. <pred>`
  - `ensures` clause: `\p1 … pn result. <pred>`
- The decls take the parameter types from the signature and return `bool`. They go through the standard post-typecheck phases: dictionary elaboration, monomorphization, OpLowering. Polymorphic contracted functions (`polyc.ail`, V12) get a `DictAbs` like any constrained function. The leading `__` keeps them out of `iface`, `docs` and exports, the same treatment as `__namedtest_entry`.
- `DeclMeta.Contracts[i]` keeps `Expr` (for SMT) and gains `Checker string`.
- **Evaluator.** `attachContracts` stores checker references. `checkPreconditions` and `checkPostconditions` apply them to the call's arguments, with no environment mutation; `e.env.Set("result", …)` at `eval_evaluator.go:620` and the silent non-Core `continue` go away.
- **Harness.** `EvaluateEnsuresHarnessFromCore` and `EvaluateRequiresHarnessFromCore` call the checker with the generated values. `BuildEnsuresPropertyHarnessFromCore` and `BuildRequiresPropertyHarnessFromCore` are deleted, together with their AST-based wrapper `BuildEnsuresPropertyHarness`, which only tests call. Per the testing policy, the tests that exercised the deleted builders go too.
- After M6, harness and evaluator differ only in who calls the checker. G4 is gone by construction, and c20 is already a CTR002 from M4.

**M7: VM enforcement; lift M1 (~2.5 d).**
- `runner.CompileBytecodeFromResult` gains a contracts option. Images built with it compile, for each contracted function:
  - a prologue that calls each `requires` checker on the arguments;
  - an epilogue that calls each `ensures` checker on the arguments and the return value;
  - each failing check calls into `effCtx.Contracts.CheckRequires`/`CheckEnsures` through the existing effect bridge, so panic and report modes and trace recording stay shared with the evaluator.
- A tail call out of a contracted function compiles as a normal call followed by the epilogue (the semantics above).
- Images built without the option are **byte-identical to today's**. A test diffs the disassembly of `examples/runnable/contracts/*.ail` with the option off against `main`.
- Whether to add one opcode (`OpContractCheck`) or a pure builtin call is left to the agent.
- Remove the M1 refusal.
- Extend the parity test with the VM engine.
- Run `perf-sweep` (see Risks).

### Files to Modify/Create

**New:**
- `internal/pipeline/contract_check.go` (~250 LOC). CTR001–CTR003 and CTR006, plus the runtime-evaluable whitelist.
- `internal/pipeline/contract_checkers.go` (~200 LOC). Synthesizes the `__contract_*` decls.
- `internal/pipeline/contract_check_test.go` (~250 LOC).
- `internal/contracts/parity_test.go` (~300 LOC), plus `testdata/contract_parity.golden`. The package location is the agent's call.
- `examples/runnable/contracts/parity/` holds the probe corpus (24 small files) and the issue repros.

**Modified:**
- `internal/runner/run.go` (+15). M1 refusal, removed in M7.
- `internal/runner/batch.go` (+10). The same for `--batch`.
- `internal/errors/codes.go` (+20). Registers `CTR001`–`CTR006`.
- `internal/eval/eval_equality.go` (+8/−4). M2.
- `internal/types/typechecker_core.go` (+40). The contract-context `Forall` rule (M3).
- `internal/pipeline/op_lowering.go` (+10). Recursion into `Forall` (M3).
- `internal/elaborate/dictionaries.go` (+10). The same (M3).
- `internal/eval/eval_expressions.go` (+40). The `Forall` case (M3).
- `internal/gen/lower/expr.go` (+5/−2). `Forall` panics in M3. In M7 contract prologue and epilogue emission land here or in the compiler.
- `internal/pipeline/pipeline_module_compile.go` (+10/−45). Calls `contract_check` and drops the best-effort block (M4).
- `internal/core/core.go` (+3). `Contract.Checker`, plus a gob registration check (M6).
- `internal/runtime/runtime.go` (±20). `attachContracts` stores checkers (M6).
- `internal/eval/eval_evaluator.go` (+20/−60). Checker calls (M6).
- `internal/testing/harness.go`, `executor.go`, `contract_domain.go`, `runner.go` (+40/−150). Checker calls; the deleted builders (M6).
- `internal/runner/vm.go` and `internal/bytecode/compiler/*` (+200). The contracts image option, prologue/epilogue, tail-call handling (M7).
- `Makefile` (+5). `test-contract-parity`, wired into `ci`.
- `docs/docs/` contracts guide (+30). Explains CTR codes and runtime `forall` semantics.
- `changelogs/unreleased/` gets one fragment per milestone.

### Pipeline Pass Coverage

- [ ] `prog.Decls` gets the synthesized `__contract_*` decls (M6). Every post-typecheck pass sees them as ordinary decls.
- [ ] `prog.Meta[*].Contracts[*].Expr` is still lowered for SMT. OpLowering and dictionary elaboration now recurse into `Forall` (M3).
- [ ] `TestContractExpressionsFullyLowered` still passes, extended with a forall predicate that has arithmetic in its body.

## Examples

### Example 1: #1956 after M1, and after M7

```
$ ailang run --bytecode --verify-contracts --caps IO --entry main bcviol.ail     # after M1
Error: CTR004: --verify-contracts is not enforced on the bytecode VM yet ... Tracking: #1956.
exit 2

$ ailang run --bytecode --verify-contracts --caps IO --entry main bcviol.ail     # after M7
Error: execution failed: contract violation: ensures failed in neg at bcviol.ail:6:13: (result > 0)
exit 1
```

### Example 2: an invalid contract caught by `ailang check` (M4, error phase)

```ailang
export pure func f(x: int) -> int
  ensures { result == "x" }
{ x }
```

```
$ ailang check c22illtyped.ail
Error: CTR001: contract predicate does not type-check at c22illtyped.ail:4:13:
  cannot unify int with string in (result == "x")
```

Today: `✓ No errors found!`. The verifier then reports a Z3 sort ERROR, and the evaluator crashes.

## Alternatives Considered

1. **Route contracted functions to the evaluator when `--verify-contracts` is on.** The VM already has EvalOnly fallback through the bridge.
   - For: about a day of work; correct immediately.
   - Rejected as the end state: the functions under contract are exactly the per-sample hot code the reporter needs on the VM, and every call pays bridge value conversion (`vm.BytecodeToEval`).
   - Kept as the fallback if M7 slips past one release. It would replace M1's refusal and still be honest, because contracts would be checked.
2. **The VM calls back into the evaluator to evaluate each predicate.**
   - Rejected. Every contracted call converts all arguments and the result between value representations. That is O(size of argument) per call, so a recursive list function becomes quadratic. It also keeps two contract evaluators alive.
3. **Teach the VM a mini-interpreter for predicate Core.** Rejected: a fifth semantics, the opposite of the goal.
4. **Make SMT the only semantics:** forbid runtime contracts for constructs the verifier cannot encode.
   - Rejected. Runtime contracts exist for what the verifier skips (Array/Map parameters, HOFs, recursion, tuples), as the #1955 body says.
5. **Patch each issue where it breaks:** a structural fallback in `makeADTEqualityFn`, a `Forall` case in the evaluator, refuse the flag pair.
   - These are M1–M3. On their own they are the sixth patch to this seam and leave nothing to stop the seventh.
   - M4 (static semantics), M5 (parity test) and M6 (one path) are what turn them into a fix.
6. **Fix #1954 by marking record-alias Eq instances `Structural` in the type layer.**
   - Rejected for now (D5). It renames the Go codegen dictionary that works today (c07) and changes instance resolution for every module. The evaluator change is one function with no type-level blast radius.
7. **Add `exists` alongside `forall`.**
   - Not in this doc. `exists` is a reserved keyword with no grammar production (`internal/parser/parser_error.go:256-257`; c21 is a parse error).
   - `not (forall i: lo..hi => not p)` already expresses it, verifies (c24), and runs after M3.
   - Adding syntax is a language change (A8) with its own SMT encoding. CTR003 makes sure that if someone adds it to the verifier only, `ailang check` refuses it.

## Success Criteria / Acceptance

**Exact repros** (sources in the issues and in the [Appendix](#appendix-probe-sources); M1 checks them into the repo, location per Deferred Decisions):

| ID | Command | Today | After |
|---|---|---|---|
| A1 (#1956) | `ailang run --bytecode --verify-contracts --caps IO --entry main bcviol.ail` | `-5`, exit 0 | M1: CTR004, exit 2, **no `-5` on stdout**. M7: `contract violation: ensures failed in neg …`, exit 1 |
| A1b | the same with `--strict-bytecode`, and with `--batch` | silent | the same as A1 |
| A2 (#1954) | `ailang run --verify-contracts --caps IO --entry main receq.ail` | CRASH `expected TaggedValue` | prints `1`, exit 0 (M2) |
| A2b | `ailang test receq.ail` | `pick_property_2` ✗ harness evaluation failed | no harness evaluation failure; `pick_property_2` ✓ (M2) |
| A2c | `ailang run --caps IO --entry main receq5.ail` (body `a == b`, two `Run` parameters) | CRASH | `true` on evaluator and VM (M2) |
| A3 (#1955) | `ailang test forallrt.ail` | ✗ `core evaluation not implemented for *core.Forall` | ✓ (M3) |
| A3b | `ailang run --verify-contracts --entry main examples/runnable/contracts/quantifier_verify.ail` | CRASH | prints `41`, exit 0 (M3) |
| A3c | a `forall` with an arithmetic body (c17) and `not (forall …)` (c24) under `ailang test` | CRASH | ✓ (M3) |
| A4 | `ailang check c22illtyped.ail` and `ailang check c20effect.ail` | `✓ No errors` | CTR001 / CTR002 (warning, then error per F2) (M4) |
| A5 | `make test-contract-parity` | n/a | green. Reverting M3's evaluator case turns it red on c16, c17, c24 and `quantifier_verify` (mutation check) (M5) |
| A6 | every parseable probe (c01–c20, c24; c20 and c22 are CTR errors after M4) under `run --bytecode --verify-contracts` | NOT CHECKED | the same stdout, exit code and violation text as `run --verify-contracts` (M7) |

**Negative controls** (must not change):
- N1. `ailang run --bytecode --caps IO --entry main bcviol.ail` (no `--verify-contracts`) still prints `-5`, exit 0, after M1 and M7.
- N2. `ailang run --verify-contracts … bcviol.ail` still reports `contract violation: ensures failed … (result > 0)`, exit 1, on the evaluator at every milestone.
- N3. `receq2.ail` and `receq7.ail` (record `==` that works today) still print `true` on both engines.
- N4. c08 (ADT Eq), c09 (list), c10 (tuple), c11 (Option) and the `Result` row keep their results on both engines.
- N5. A `forall` in a function body (`forallbody.ail`) is still rejected by `ailang check`, now with CTR006, and is never lowered to `true`.
- N6. `ailang verify` output for every file in `examples/runnable/contracts/` is unchanged by M1–M7. The SMT input is the same lowered `Contract.Expr`. Diff the before and after transcripts.
- N7. With contracts off, VM disassembly of `examples/runnable/contracts/*.ail` is byte-identical before and after M7.
- N8. `tailens.ail` (`ensures` on a body that is a tail call) reports a violation on both engines after M7.
- N9. `polyc.ail` (a polymorphic contracted function) runs on both engines with contracts on.

**General:**
- [ ] `make test`, `make test-core`, `make verify-examples` and `make check-file-sizes` are green.
- [ ] `ailang verify` results on `examples/` are unchanged (N6).
- [ ] The docs contracts guide documents CTR001–CTR006 and runtime `forall` semantics.
- [ ] One changelog fragment per milestone.

## Testing Strategy

**Unit tests:**
- CTR004 refusal: `run.go` and `batch.go`, flag matrix.
- `makeADTEqualityFn` on TaggedValue, RecordValue and mixed operands.
- The contract-context `Forall` type rule, positive and negative (a non-int bound, a non-bool body).
- Evaluator `Forall`: empty range, short-circuit, a raising body gives CTR005.
- `contract_check` classification table over all 22 Core node types.

**Integration:**
- The probe corpus (c01–c24 plus the issue repros) through `ailang test`, `run --verify-contracts` and (M7) `run --bytecode --verify-contracts`, as golden transcripts.
- `TestEngineParity_Contracts` for the harness on both engines.

**Regression surface** (Conflict Surface fixtures):
- `examples/runnable/contracts/basic.ail`, `record_verify.ail`, `list_verify.ail`, `recursive_verify.ail`, `cross_module_functions.ail`
- `examples/deriving_eq.ail` and `examples/eq_containers.ail`
- `quantifier_verify.ail`, which deliberately changes from crash to pass

**Mutation check** (per the feedback memory): after M3, revert the evaluator `Forall` case in a scratchpad copy and confirm the parity test fails; after M2, revert the structural fallback and confirm A2 and A2c fail.

## Conflict Surface

### Positions touched

1. **Type inference over contract predicates** (`pipeline_module_compile.go:233-280`) goes from discard-errors to report-errors (M4).
2. **A `core.Forall` type rule** (`typechecker_core.go`). `forall` is registered as a **general** prefix parser (`internal/parser/parser.go:106`), so it parses in any expression position today. Only the type checker's missing rule rejects it in bodies (V9).
3. **Derived Eq at runtime** (`eval_equality.go`) affects every `==` on a nominal `deriving (Eq)` record, in bodies too.
4. **New synthesized top-level decls** (`__contract_*`) in `prog.Decls`, and so in every pass over them: monomorphization, effect validation, iface generation, Go codegen, the bytecode compiler.
5. **VM function prologue and epilogue** for contracted functions in contract-mode images, and tail-call compilation in those functions.
6. **The runner flag matrix** (`--bytecode`, `--strict-bytecode`, `--batch` with `--verify-contracts`).

### What else lives there

| Position | Existing occupant | Resolution |
|---|---|---|
| 1 | Contracts that are ill-typed or effectful, or use `==` without an instance, all accepted today (c20, c22, noeq) | Intentional rejection, phased per F2. Blast radius measured first (Risks) |
| 1 | Contracts that reference an unimported builtin (today a Z3 "unknown constant" verify error) | Becomes CTR001 at check time. Strictly earlier, same verdict |
| 2 | `forall` in a body: a check error today | Stays an error, now CTR006. The rule fires only when the inference context says "contract" |
| 3 | ADT `==` (TaggedValue) and the structural instances | TaggedValue pairs still go to `taggedValuesEqual`. Only the non-TaggedValue branch changes, from error to structural |
| 4 | User functions whose names start with `__contract_` | `__`-prefixed names are already reserved for synthesized entries (`__namedtest_entry`). M6 adds a check that rejects a user decl with that prefix |
| 4 | Go codegen, which would now see extra decls | They are ordinary pure functions. Go codegen compiles them or fails as it does for any decl. Its contract printer is untouched (Non-Goal) |
| 5 | VM tail-call optimization for contracted functions | Lost **only** in contract-mode images (N8). Deep recursion risk is in Risks |
| 6 | `run --bytecode` without contracts | Untouched (N1, N7) |

### Programs that must still work

`examples/runnable/contracts/basic.ail`, `record_verify.ail`, `list_verify.ail`, `recursive_verify.ail`, `cross_module_functions.ail`, `examples/deriving_eq.ail`, `examples/eq_containers.ail`, and `polyc.ail` (a polymorphic contract).

### What deliberately changes

- `ailang check` rejects ill-typed, effectful and Eq-less contract predicates, phased per F2.
- `--bytecode --verify-contracts` refuses (M1), then enforces (M7).
- A crashing predicate reports CTR005 instead of masquerading as a violation.
- `quantifier_verify.ail` and any forall contract go from runtime crash to evaluated.
- Nominal-record `==` in bodies stops crashing on the evaluator.

## Axiom Compliance

| Axiom | Score | Justification |
|---|---|---|
| A1 Determinism | 0 | Contract checks are deterministic today and stay so. `forall` iterates in a fixed order |
| A2 Replayability | 0 | The contract trace events (`RecordContractCheck`) are unchanged and now emitted on the VM too |
| A3 Effect legibility | +1 | A contract can no longer perform effects (c20 performs IO today inside a `pure` function's `ensures`) |
| A4 Explicit authority | +1 | The same: capabilities granted to the program can no longer be exercised from a specification |
| A5 Bounded verification | +1 | The runtime checks exactly what the verifier proves, and a test enforces it |
| A6 Safe concurrency | 0 | No change |
| A7 Machines first | +1 | Typed CTR codes replace `expected TaggedValue, got *eval.RecordValue`. `bestof` can tell a crash from a violation |
| A8 Minimal syntax | 0 | No new syntax. `exists` is deliberately not added |
| A9 Cost visibility | 0 | The cost of runtime `forall` is documented. The perf gate is measured |
| A10 Composability | +1 | One contract semantics across evaluator, harness and VM |
| A11 Structured failure | +1 | CTR001–CTR006. A failing evaluation is distinct from a violation. No silent pass |
| A12 System boundary | 0 | No change |

**Net: +6. Proceed.** No −1 on A1, A3, A4 or A7.

## Risks & Mitigations

| Risk | How to measure | Mitigation |
|---|---|---|
| **M4 rejects published code** whose contracts are ill-typed or effectful | Run M4 in warning mode over `examples/`, `std/`, the `benchmarks/contract_*.yml` solutions, the motoko_agent and world forks, and every package in the registry cache. Count CTR001 and CTR002 by package | Warning phase (F2). The count goes in the release note. The flip to errors happens only after the count is reviewed |
| **Eq-less ADT `==` in contracts** (noeq) becomes an error before m-adt-equality lands | The same sweep, filtered on missing-`Eq` CTR001 | F2: that sub-case stays a warning until the compiler-provided ADT Eq lands |
| **VM perf with contracts on** | `perfc.ail` (a contracted `step` called 300k times). Today: evaluator 2.10 s, evaluator+contracts 2.75 s, VM 0.06 s (V10). Re-run at N = 3,000,000 with 5 repetitions per mode. Also `perf-sweep` | Gate: VM+contracts ≤ 2× VM, and ≥ 15× faster than evaluator+contracts. Contract-off images are byte-identical (N7), so the off path costs nothing |
| **Lost TCO in contract-mode images** overflows the VM stack on deep recursion | `recursive_verify.ail` and `list_recursive_verify.ail` at increasing depth, with and without contracts; compare against the evaluator's limit (`defaultVMMaxStack = 10000` in `bytecode_engine.go`) | The two engines fail at the same depth, or the difference is ledgered. A recursion overflow under contracts reports the existing stack-limit error, never a silent pass |
| **Evaluator perf** from calling a checker function instead of evaluating the expression inline | `perfc.ail` evaluator+contracts before and after M6 | Gate: ≤ 10% slower |
| **Compile time** from synthesized decls | `ailang check` wall time over `examples/runnable/contracts/` (46 files), before and after | Gate: ≤ 5%. Checker decls are the size of the predicates |
| **Partial functions in predicates** (`_list_nth` out of range) behave differently in Z3 (unspecified) and at runtime (raise) | The parity test's seeded runs record CTR005 per function | A raise is CTR005 (loud). If it happens on an in-contract input of a VERIFIED function, that is a parity failure, either fixed or ledgered with a reason |
| **Z3 `Int` vs runtime `int64`** makes a VERIFIED contract fail at runtime on overflow | Parity layer (b) | A ledgered known divergence. `verify` already prints "runtime equivalence not_established" |

## Deferred Decisions

- Opcode (`OpContractCheck`) vs a builtin call for the VM check. Agent's call.
- Package location of the parity test (`internal/contracts` vs `internal/smt`). Agent's call.
- One checker per clause vs one per kind returning a failing-clause index. Agent's call, as long as messages stay per-clause.
- A cost cap for runtime `forall` (for example `--max-forall-range`). Decided at review if the parity sweep finds a slow contract.
- Whether the warning phase of F2 gets a `--strict-contracts` opt-in to errors. Agent's call.
- Where the probe corpus and issue repros live. The deliberately violating ones (`bcviol`, `tailens`) must sit where the `verify-examples` manifest expects failure (for example `examples/expected_fail/`), because manifest drift is the usual cause of a red `verify-examples`. Agent's call.

## Non-Goals

- **Go codegen contract checks** (`compile --emit-go --verify-contracts`). They fail to compile for 17 of 21 swept probes. That failure is loud, so it is a separate defect. File a follow-up issue linked from M7. Once M6 exists, the natural fix is for Go codegen to emit the `__contract_*` decls as ordinary functions and call them, retiring `contracts.go`'s predicate printer.
- **`properties [forall(x: T) => …]` blocks.** [m-forall-properties-direct-core-eval.md](../v1_1_0/m-forall-properties-direct-core-eval.md) (P3) is **not subsumed**. Its construct is a test-only, generator-bound quantifier stored outside `Meta.Contracts`. This doc's bounded `forall i: lo..hi` is a contract-predicate quantifier over a finite integer range. Shared ground: after M6 a properties predicate could compile through the same checker-decl synthesis, with the binders as parameters. That doc's "elaborator path" option should then reuse `contract_checkers.go` instead of building its own.
- **SMT coverage changes** (tuples, `show` in contracts, callee gating). These are [m-contract-verification-coverage.md](m-contract-verification-coverage.md) and the triage notes `show-in-ensures-clause-unencodable.md` and `verify-ensures-verified-without-encodable-refutation.md`. This doc does not change the verifier's input or verdicts (N6).
- **The general VM parity defects** in [m-bytecode-vm-parity-bugs.md](../v1_0_0/m-bytecode-vm-parity-bugs.md): pattern arity, unsafe replay, stack limits. That doc never mentions contracts. This doc adds the contract column to evaluator↔VM parity and reuses the `internal/testing` engine-parity harness, not that doc's `scripts/verify_bytecode_parity.go` classification work.
- **`exists` syntax, `deriving (Ord)`**: c21 and c23 are parse errors today and stay so.

## Timeline

| Milestone | Effort | Independently shippable |
|---|---|---|
| M1 refuse (CTR004) | 0.5 d | yes. Closes the silent half of #1956 |
| M2 record Eq | 0.5 d | yes. Closes #1954 |
| M3 runtime `forall` | 1.5 d | yes. Closes #1955 |
| M4 static check (warning phase) | 2 d | yes |
| M5 parity test | 1.5 d | yes. Lands before M6 |
| M6 one runtime path | 2 d | yes. Behaviour-preserving, measured by M5 |
| M7 VM enforcement, lift M1 | 2.5 d | yes. Closes #1956 fully |
| **Total** | **~10.5 d** | |

## Verification Log

All runs used the origin/dev build (`scratchpad/a/ailang-od`), from directories where module path = file path. Scratch probes are in `scratchpad/dd5/` and `scratchpad/dd5/mx/`.

| # | Claim | Evidence |
|---|---|---|
| V1 | #1956 reproduces | `run --verify-contracts … bcviol.ail` → `contract violation: ensures failed in  at bcviol.ail:6:13: (result > 0)`, rc=1. `run --bytecode --verify-contracts …` → `-5`, rc=0 |
| V2 | #1954 reproduces; `verify` proves it | `verify receq.ail` → `✓ VERIFIED pick`. `run --verify-contracts` → `postcondition evaluation error … derived Eq for Run: expected TaggedValue, got *eval.RecordValue and *eval.RecordValue`. `test` → `pick_property_2 ✗ … harness evaluation failed` |
| V3 | #1954 is not contract-only | `receq5.ail` (`same(a: Run, b: Run) = a == b`): evaluator → the same crash; `--bytecode` → `true`. `receq3` (a value through `pick` then `same`): evaluator crash, VM `true`. Controls: `receq6` (parameter vs record literal) → `true`; `receq7` (`let a: Run`, `let b: Run`) → `true`; `receq2` → `true` |
| V4 | A shipped example has all three symptoms | `examples/runnable/contracts/quantifier_verify.ail`: `verify` → 3 VERIFIED. `run --verify-contracts --entry main` → `core evaluation not implemented for *core.Forall`, rc=1. `run --bytecode --verify-contracts` → `41`, rc=0. `test` → `positiveRange_property_2 ✗` |
| V5 | #1955 reproduces | `verify forallrt.ail` → `✓ VERIFIED two`. `test` → `ensures harness evaluation failed: core evaluation not implemented for *core.Forall` |
| V6 | The VM path reads no contracts | `grep -rn "Contract" internal/vm internal/bytecode internal/gen/lower` → no matches outside tests. `effCtx.Contracts` is set only at `run.go:533-538` and `batch.go:77-79` |
| V7 | Predicate type errors are discarded | `c22illtyped.ail` (`ensures { result == "x" }`, `-> int`): `check` → `✓ No errors found!`. `verify` → `Z3 error … Sorts Int and String are incompatible`. Evaluator → `eq_Int: expected IntValue for arg 1, got *eval.StringValue`. Code: `pipeline_module_compile.go:272-276` |
| V8 | Effects in predicates are unchecked, and the evaluator performs them | `c20effect.ail` (a `pure` function, `ensures { println("side effect in contract") == () }`): `check` ✓. `run --verify-contracts` prints `side effect in contract` then `1`. `test` → `effect 'IO' requires capability` |
| V9 | `forall` parses anywhere but has no type rule | `forallbody.ail` (`forall` in a body): `check` → `type inference not implemented for *core.Forall` (`typechecker_core.go:771-772`). `parser.go:106` registers `FORALL` as a general prefix |
| V10 | Perf baseline | `perfc.ail` (`loop(300000)` calling a contracted `step`): evaluator 2.10 s, evaluator+contracts 2.75 s, VM 0.06 s, VM+`--verify-contracts` 0.06 s (unchecked). `/usr/bin/time -p`; evaluator modes 1 run each, VM modes 3 runs each (stable at 0.06 s) |
| V11 | The evaluator checks `ensures` on tail-call bodies | `tailens.ail`: `f(x) ensures {result > 0} { g(x) }` → violation reported, rc=1. Negative claim refuted, so the doc relies on it as a fixture, not as a gap |
| V12 | `result` shadows a parameter; polymorphic contracts run | `resparam.ail` (`f(result: int) ensures {result > 100} { result + 1 }`) → violation. `polyc.ail` (`ident[a]`, `ensures { result == x }`) → `5`, rc=0; verify SKIPPED |
| V13 | Other Eq shapes in contracts work on the evaluator | `adteq.ail`: ADT, list, tuple, Option and Result `ensures { result == … }` all pass under `run --verify-contracts`. `verify`: tuple SKIPPED, the others VERIFIED |
| V14 | Corpus size, and today the only harness crash is `quantifier_verify` | `grep -rlE "^\s*(requires\|ensures)\s*\{" examples std --include='*.ail'` → 46 files; `grep -rhE` with the same pattern → 216 lines. The sweep used the unanchored pattern (47 files). Sweep of `ailang test` over the 47 files (`scratchpad/dd5/sweep.txt`; `per_function_depth_verify.ail` was killed after 14 CPU-minutes and has no result): 1 file with `harness evaluation failed` (`quantifier_verify.ail`). The sweep surfaced no nominal-record `==` crash in any corpus contract, which is consistent with #1954 going unseen (a property whose every case is discarded by `requires` would also hide it) |
| V15 | Proposed `CTR` codes are unallocated | `grep -rn '"CTR[0-9]' internal cmd docs` → empty. `grep -rn "E_CONTRACT\|E_FORALL" .` → empty. The prefixes in `internal/errors/codes.go` are DSG, ELB, EVA, IMP, LDR, LNK, MOD, PAR, RT, TC, TST |
| V16 | `exists` is not a contract construct | `c21exists.ail` → `PAR_NO_PREFIX_PARSE`. `parser_error.go:256-257`: "'exists' is reserved for existential types (future feature)" |
| V17 | Go codegen contract printing is separate and mostly broken | `compile --emit-go --verify-contracts` over c01–c20 and c22: 17 of 21 produce Go that does not compile (`undefined: result`, `not an interface`, slice compare). c16 and c17: `contract predicate: unsupported expression type: *core.Forall`. Builds: c07, c08, c11, c18, bcviol |
| V18 | The harness uses lowered Core from `Meta.Contracts` in production | Callers: `contract_domain.go:134,197` and `runner.go:481` use `Evaluate{Ensures,Requires}HarnessFromCore`. The AST variant `BuildEnsuresPropertyHarness` is called only from `executor.go:126` (`EvaluateEnsuresHarness`), which has no production caller |
| V19 | The Eq instance path for nominal types | `derived_eq.go:57-76` registers `Eq[TCon T]` with `DerivedADTEquality` for every derived type. `instances.go:55-60` does a direct-key lookup before `synthesizeEq`. `instances_eq_synth.go:74-81` makes `TRecord` structural. Code-read, consistent with V3. M2's unit test pins it |

Incidental, not in scope:
- `ailang test examples/runnable/contracts/per_function_depth_verify.ail` spun more than 14 CPU-minutes during the V14 sweep and was killed. Worth a separate issue.
- The violation message has an empty function name: `ensures failed in  at`. M6 fixes it as a side effect.

## Appendix: probe sources

Each `cNN` probe file is `module cNN<name>`, then `import std/io (println)` (plus the imports shown), the declarations below, and `export func main() -> () ! {IO} = println(show(<call f with an in-contract input>))`. The issue repros (`receq`, `forallrt`, `bcviol`, `receq2`) are verbatim in #1954, #1955 and #1956. Every file was checked from its own directory, so module path = file path.

```ailang
-- c01arith.ail
export pure func f(x: int) -> int
  requires { x >= 0 }
  ensures { result > x * 2 - 1 }
{ x * 2 + 1 }
-- c02bool.ail
export pure func f(x: int) -> bool
  ensures { not result || x > 0, result == (x > 0) }
{ x > 0 }
-- c03if.ail
export pure func f(x: int) -> int
  ensures { if x > 0 then result == x else result == 0 - x }
{ if x > 0 then x else 0 - x }
-- c04let.ail
export pure func f(x: int) -> int
  ensures { let y = x + 1 in result == y }
{ x + 1 }
-- c05match.ail
export type Sh = Circle(int) | Sq(int)
export pure func f(s: Sh) -> int
  ensures { match s { Circle(r) => result == r, Sq(w) => result == w } }
{ match s { Circle(r) => r, Sq(w) => w } }
-- c06field.ail
export type P = { a: int, b: int }
export pure func f(p: P) -> int
  requires { p.a >= 0 }
  ensures { result >= p.a }
{ p.a + 0 }
-- c07receq.ail
export type P = { a: int, b: int } deriving (Eq)
export pure func f(p: P) -> P
  ensures { result == p }
{ p }
-- c08adteq.ail
export type C = Red | Green deriving (Eq)
export pure func f(c: C) -> C
  ensures { result == c }
{ c }
-- c09listeq.ail
export pure func f(xs: [int]) -> [int]
  ensures { result == xs }
{ xs }
-- c10tupeq.ail
export pure func f(p: (int, int)) -> (int, int)
  ensures { result == p }
{ p }
-- c11opteq.ail
export pure func f(o: Option[int]) -> Option[int]
  ensures { result == o }
{ o }
-- c12call.ail
export pure func sq(x: int) -> int = x * x
export pure func f(x: int) -> int
  ensures { result == sq(x) + 1 }
{ x * x + 1 }
-- c13builtin.ail
import std/list (length)
export pure func f(xs: [int]) -> [int]
  ensures { length(result) == length(xs) + 1 }
{ 0 :: xs }
-- c14string.ail
export pure func f(s: string) -> string
  ensures { result == "${s}!" }
{ "${s}!" }
-- c15hof.ail
import std/list (any)
export pure func f(xs: [int]) -> [int]
  ensures { not any(\x. x < 0, result) }
{ [1, 2] }
-- c16forall.ail
import std/list (length)
export pure func f(xs: [int]) -> [int]
  ensures { forall i: 0..length(result) => _list_nth(result, i) >= 0 }
{ [1, 2] }
-- c17forallarith.ail
export pure func f(n: int) -> int
  requires { n >= 0 }
  ensures { forall i: 0..n => i + 1 > 0 }
{ n }
-- c18float.ail
export pure func f(x: float) -> float
  requires { x >= 0.0 }
  ensures { result >= x }
{ x + 1.0 }
-- c19reclit.ail
export type P = { a: int, b: int } deriving (Eq)
export pure func f(x: int) -> P
  ensures { result == { a: x, b: 0 } }
{ { a: x, b: 0 } }
-- c20effect.ail
export pure func f(x: int) -> int
  ensures { println("side effect in contract") == () }
{ x }
-- c21exists.ail
export pure func f(n: int) -> int
  ensures { exists i: 0..n => i == 0 }
{ n }
-- c22illtyped.ail
export pure func f(x: int) -> int
  ensures { result == "x" }
{ x }
-- c23ord.ail
export type C = Lo | Hi deriving (Eq, Ord)
export pure func f(c: C) -> C
  ensures { result >= c }
{ Hi }
-- c24notforall.ail
export pure func f(n: int) -> int
  requires { n > 0 }
  ensures { not (forall i: 0..n => i != 0) }
{ n }
-- receq5.ail
export type Run = { size: int, end: int } deriving (Eq)
export pure func same(a: Run, b: Run) -> bool = a == b
export func main() -> () ! {IO} = println(show(same({ size: 1, end: 2 }, { size: 1, end: 2 })))
-- receq3.ail
export type Run = { size: int, end: int } deriving (Eq)
export pure func same(a: Run, b: Run) -> bool = a == b
export pure func pick(best: Run) -> Run = best
export func main() -> () ! {IO} = println(show(same(pick({ size: 1, end: 2 }), { size: 1, end: 2 })))
-- receq6.ail
export type Run = { size: int, end: int } deriving (Eq)
export pure func same(a: Run) -> bool = a == { size: 1, end: 2 }
export func main() -> () ! {IO} = println(show(same({ size: 1, end: 2 })))
-- receq7.ail
export type Run = { size: int, end: int } deriving (Eq)
export func main() -> () ! {IO} { let a: Run = { size: 1, end: 2 }; let b: Run = { size: 1, end: 2 }; println(show(a == b)) }
-- tailens.ail
export pure func g(x: int) -> int = 0 - x
export pure func f(x: int) -> int
  ensures { result > 0 }
{ g(x) }
export pure func h(x: int) -> int
  ensures { result > 0 }
{ g(x) + 0 }
export func main() -> () ! {IO} { println(show(f(5))); println(show(h(5))) }
-- perfc.ail
export pure func step(x: int) -> int
  requires { x >= 0 }
  ensures { result >= 0 }
{ if x < 3 then x else x - 3 }
export pure func loop(i: int, acc: int) -> int = if i == 0 then acc else loop(i - 1, acc + step(i % 7))
export func main() -> () ! {IO} = println(show(loop(300000, 0)))
-- forallbody.ail
export pure func allpos(xs: [int]) -> bool = forall i: 0..2 => _list_nth(xs, i) > 0
export func main() -> () ! {IO} = println(show(allpos([0 - 1, 0 - 2])))
-- noeq.ail
export type Color = Red | Green
export pure func g(c: Color) -> Color
  ensures { result == Red }
{ c }
export func main() -> () ! {IO} = println(show(g(Red)))
-- polyc.ail
export pure func ident[a](x: a) -> a
  ensures { result == x }
{ x }
export func main() -> () ! {IO} = println(show(ident(5)))
-- resparam.ail
export pure func f(result: int) -> int
  ensures { result > 100 }
{ result + 1 }
export func main() -> () ! {IO} = println(show(f(5)))
```

## Relationship to existing docs

- [m-adt-equality-position-independence.md](m-adt-equality-position-independence.md) (P1, planned). It identified that contract predicates are not type-checked. Correction: they are inferred at `pipeline_module_compile.go:273`, but the errors are discarded, which has the same effect. **This doc takes over that doc's "contract predicates must be type-checked" half** (M4). That doc keeps the compiler-provided ADT Eq ruling and the hint fix. Ordering per F2.
- [m-forall-properties-direct-core-eval.md](../v1_1_0/m-forall-properties-direct-core-eval.md) (P3). Not subsumed; see Non-Goals. It can reuse M6's checker synthesis.
- [m-contract-verification-coverage.md](m-contract-verification-coverage.md) (P0, parked under D-29/D-30). It covers the verifier's skipped vs not-applicable reporting and package evidence. Its line "a static report must not imply runtime contracts ran" is consistent with this doc. M5's ledger is a runtime-side counterpart, not a change to its counters.
- [m-bytecode-vm-parity-bugs.md](../v1_0_0/m-bytecode-vm-parity-bugs.md). General VM soundness. It has no contract scope, so there is no overlap. This doc reuses the `internal/testing` engine-parity harness, not its parity script.
- Implemented: [m-verify-runtime-contracts.md](../../implemented/v0_6_1/m-verify-runtime-contracts.md) (v0.6.2, the original evaluator and Go runtime checks), [m-verify-contracts.md](../../implemented/v0_7_1/m-verify-contracts.md) (SMT), [m-contracts-oplowering.md](../../implemented/v0_8_0/m-contracts-oplowering.md), [m-eq-derive-containers.md](../../implemented/v0_42_0/m-eq-derive-containers.md) (`6d096fe98`; its record synthesis triggers only for `TRecord`-typed operands, see V19).
- [m-smt-well-sorted-preflight.md](m-smt-well-sorted-preflight.md) (P1, landed alongside this doc as `399630274`). It is the **verifier-side** sibling. Each doc stays on its own side of the seam:
  - Its M1 resolves SMT operators from `CoreTI`, **including at contract sites**. Today `CoreTI` is incomplete there: inference errors are discarded, and every `forall` predicate fails inference. This doc's M3 (the `Forall` type rule) and M4 (fatal inference) give it complete `CoreTI` for contracts. If this doc's M4 lands first, that doc's M1 needs fewer `encoder_gap` fallbacks at contract sites.
  - Its M3 turns an ill-sorted script into SKIPPED `encoder_gap`. After this doc's M4, an ill-typed contract (c22) never reaches the encoder: it is a CTR001 at `ailang check`.
  - Its S1 (Int `/` and `%` are Euclidean in SMT but truncating at runtime) is exactly the "VERIFIED, yet violated at runtime" divergence that M5 parity layer (b) is built to catch. Add its S1 fixture to the M5 corpus as a regression row once that doc's M0 lands.

## References

- Issues #1954, #1955, #1956, and their triage comments (2026-10-26).
- Closed #963 (lists; `6d096fe98`).
- CLAUDE.md principle 2 (no silent fallbacks), "verify the seam" (memory), PROGRAM.md lanes.

**Document created**: 2026-10-26
**Last updated**: 2026-10-26
