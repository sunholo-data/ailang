# Inline-Test Expected Values: lists and ADTs evaluate; `check` rejects what `test` cannot run

**Refs #495** (Finding 2; Finding 1 fixed by a9e26ffd6 / PR #549. Finding 3 re-verification: §F3 below.)

**Status**: Planned
**Target**: v0.53.0
**Priority**: P1 (Medium) — ADT- and list-returning functions cannot be table-tested at all, and the check leg reads green on rows the test leg cannot run.
**Estimated**: 2 days
**Dependencies**: None hard. Coordinates with [m-inline-test-multiarg-tuple-rows.md](../m-inline-test-multiarg-tuple-rows.md) (input-side row grammar; still Planned, target v0.39.0 — stale) — see §Relationship.

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

Every feature must align with AILANG's 12 Design Axioms. Score each axiom and verify no hard violations.

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Expected-value evaluation is a pure AST→Core conversion of a literal-shaped expression plus one evaluator pass — same input, same value, every run. No random generation involved (unlike property legs). |
| A2: Replayability | 0 | Seeds, replay flags, and report formats unchanged. |
| A3: Effect Legibility | 0 | No effect change. Effectful functions under test keep today's `--caps` gating (verified live, V6); the grammar admits no new effectful surface (arbitrary calls in rows were already possible on the input side). |
| A4: Explicit Authority | 0 | No capability change. |
| A5: Bounded Verification | +1 | The check leg gains a bounded, per-row static gate; today its silence on test rows is an unbounded "trust me" surface. The test leg fast-fails on rejected rows instead of dying mid-evaluation (or crashing the process, V5). |
| A6: Safe Concurrency | 0 | No concurrency change. |
| A7: Machines First | +1 | The two legs stop contradicting each other on identical source. An agent running only `ailang check` (the common inner loop) no longer reads green on a table that cannot execute — the exact false-green shape that already cost this project a CI run once (#495's own framing). |
| A8: Minimal Syntax | 0 | No syntax change; `tests [...]` rows are unchanged textually. |
| A9: Cost Visibility | 0 | No compiles added to either leg (the expected-value evaluation is an in-process evaluator pass, not a pipeline run; the check gate is an AST walk). |
| A10: Composability | +1 | Expected evaluation reuses the input side's existing converter (`astExprToCore`), evaluator factory (`newHarnessEvaluator`), and comparator (`equalValues`), retiring the row surface's fourth, scalar-only mechanism instead of adding a fifth; the grammar predicate is one function consumed by both legs. |
| A11: Structured Failure | +1 | Rejections get named `TST` codes with row locations at check time; the `panic` path that crashes the whole `ailang test` process today (V5) becomes a per-row typed error. |
| A12: System Boundary | 0 | No boundary crossings. The gate lives in `internal/pipeline` (already in the check binary's closure) rather than importing the runner into check. |

**Net Score: +5** → **Decision: Move forward.**

### Hard Violation Check

- [x] A1 (Determinism): no implicit nondeterminism introduced
- [x] A3 (Effects): no hidden side effects
- [x] A4 (Authority): no ambient access granted
- [x] A7 (Machines First): the change exists precisely to make machine-readable gates agree

## Problem Statement

An inline `tests [...]` row whose **expected** value is a list, an ADT constructor application
(possibly nullary), a tuple, or a record — i.e. every composite value the language has —
always fails at **test** time with `failed to evaluate expected: expected literal expression,
got *ast.List` / `*ast.Identifier` / `*ast.FuncCall`, while `ailang check` exits **0 clean**
on the same file. ADT- and list-returning functions therefore cannot be table-tested at all.

**Current State** (re-verified live 2026-10-08, installed binary v0.52.5 @ `7200786`, and
source-confirmed at origin/dev `1dfd5615`; triage verified the same at `658ff76a3`):

- The ADT case (the issue's own `u2b`) fails verbatim — **both** the nullary (`Allow`,
  `*ast.Identifier`) and applied (`Deny("nope")`, `*ast.FuncCall`) constructors — and
  `ailang check` prints `✓ No errors found!` (rc=0) on the same file (V1).
- The list case (folded in from #1574) fails with `got *ast.List`; check rc=0 (V2).
- The root cause is one function: `EvaluateLiteral`
  (`internal/testing/executor.go:512-565`) accepts only scalar literals
  (int/float/bool/string/unit, plus unary minus) and is the **only** production caller-side
  expected evaluator (`internal/testing/runner.go:176`; grep confirms no other non-test
  caller, V3).

**Impact:** Every function returning `list`, a tuple, a record, or any ADT — the majority of
interesting pure functions — is forced out of table tests into hand-written `test "name" { ... }`
blocks, or into canonical string projections, just to have something testable. Worse, the
**gate disagreement** is invisible: a CI or agent loop that runs only `ailang check` reads
green on a test table that cannot execute. #495's own framing: "the same shape as the
ai-check-without-z3 silent skip that already cost this project a false-green CI run."

### The systemic audit (the gap is bigger than the expected arm)

Following the anti-incremental-patching rule, all sibling paths on the same surface were
audited and measured. The inline-test row surface is evaluated by **four** disjoint
mechanisms today, and **none of them is visible to the check leg**:

| Mechanism | Where | Grammar | Failure mode |
|---|---|---|---|
| `EvaluateLiteral` (expected arm) | `executor.go:512` | scalars only | per-row error; check silent |
| `astExprToCore` (input arm) | `harness.go:152` | literals, tuples, lists, records, unary `-`, identifiers, calls; **panics** on other node kinds; `BinOp` lowers but the evaluator rejects arithmetic (`BinOp reached evaluator; dictionaries not elaborated`) | **process crash** (panic) for unsupported nodes; per-row error for arithmetic; check silent |
| `EvaluateNamedTestBodyExprs` (named bodies) | `executor.go:171` | full pipeline re-elaboration (OpLowering) | per-test error; check silent |
| `valueToLiteral` (shrinker splice-back) | `value_splice.go` | all value shapes incl. ADTs | n/a (inverse direction) |

Measured sibling defects, all with `ailang check` rc=0 on the same file:

1. **Arithmetic in a row input** `tests [ ((1+2, 3), 6) ]` fails at test time:
   `harness evaluation failed: internal: BinOp reached evaluator; dictionaries not elaborated (op='+')` (V4).
2. **An unsupported node in a row input** (`L.map(\x. x, [1,2])`) does not merely fail the
   test — it **panics in `astExprToCore` and kills the entire `ailang test` process**
   (rc=2, stack trace through `BuildInlineTestHarness` → `buildFunctionCall` →
   `astExprToCore`, V5).
3. **Named test bodies are also invisible to check**: a file whose `test "bad" { 1 + "x" }`
   body has a type error passes `ailang check` rc=0 and fails only at test time (V10).
4. The row grammar for multi-arg and tuple-valued inputs is itself broken in a fourth way,
   already designed in a separate planned doc (see §Relationship).

This doc fixes the **expected arm** (the F2 defect), the **panic**, and the **check-leg
silence for row expressions**. The named-body and row-shape/arity gaps are explicitly
scoped (§Non-Goals) with their owning docs.

## Goals

**Primary Goal:** A `tests [...]` row whose expected value is any composite literal the
language has — list, tuple, record, ADT constructor (nullary or applied), or a nested
combination — passes `ailang test`, and `ailang check` exits 1 on any row the test runner
cannot run, so the two legs can no longer disagree on identical source.

**Success Metrics:**
- Expected-value grammar: 6 scalar kinds → all `astExprToCore`-buildable shapes (measured by
  the locked-step premise test, both directions)
- Check-leg false greens on inline rows: every live repro in this doc (V1, V2, V4, V5) →
  rc=1 with a named TST code and row location
- Process crashes from bad rows (`ailang test` rc=2 panic): 1 known repro → 0
- Expression-evaluation mechanisms on the row surface: 4 (incl. the scalar-only
  `EvaluateLiteral`) → 3, all shared with the input arm
- New compiles added to either leg: 0

## Root-Cause Analysis

Three causes stack:

1. **The expected arm has its own hand-rolled evaluator.** `EvaluateLiteral` duplicates (a
   scalar subset of) what `astExprToCore` + the harness evaluator already do for the input
   arm. Every composite value kind had to be re-implemented there, and wasn't. The input
   arm, by contrast, already handles lists, tuples, records, identifiers and constructor
   applications — and the **comparator** `equalValues` (`executor_helpers.go:123-205`)
   already deep-compares `ListValue`, `TupleValue`, `RecordValue` and `TaggedValue`. The
   round-trip test `TestB12_RoundTrip` (`value_splice_roundtrip_test.go:61`) proves the full
   cycle value→AST→Core→evaluated-value for ADT-bearing values today (V8). Only the
   expected arm's evaluator is missing.
2. **`astExprToCore` panics on unknown node kinds** (`harness.go:247`,
   `panic(fmt.Sprintf("unsupported AST expression type in test harness: %T", expr))`),
   and the inline-test call path has no `recover()`. A malformed row input is therefore a
   process crash, not a test failure.
3. **The check pipeline never elaborates any test surface.** `FuncDecl.Tests` is parsed,
   copied through elaboration (`internal/elaborate/file.go:421`, `scc.go:19`), and never
   visited again — no typechecker or pipeline stage reads it (grep over `internal/types`
   finds no `Tests` visitor, V9). Whatever the test runner can or cannot run is therefore
   structurally invisible to `ailang check`, `ai-check`, and the LSP. Precedent for the fix
   exists in-repo: `internal/pipeline/effect_ceiling.go:100` is a pipeline stage that
   validates the **surface AST** (`result.Artifacts.AST`, V11) — exactly the shape needed
   here.

## The Decision: evaluate the expected expression (through the input arm's existing machinery), do not grow a literal set

The task posed two options. Both are designed here; the recommendation is the first.

### Option A (RECOMMENDED): evaluate the expected expression via `astExprToCore` + the harness evaluator

Replace the `EvaluateLiteral` call in `runner.go` with an evaluation of the expected
expression through **the exact mechanism the input arm already uses**: `astExprToCore`
(hardened to return errors) → a one-expression Core program → `newHarnessEvaluator()` →
`EvalCoreProgram`. Compare with the existing `equalValues`.

- Fixes F2 completely: lists, ADTs (nullary and applied), tuples, records, nested
  combinations — all already lowered by `astExprToCore` and already compared by
  `equalValues`. No new value machinery.
- **One grammar for both row arms** (everything `astExprToCore` can build, minus `BinaryOp`),
  so the check gate is a single, cheap, deterministic AST walk — no elaboration, no extra
  compile, no closure growth.
- Arithmetic stays rejected **symmetrically** on both arms (it is broken on the input side
  today, V4) — an honest, documented limit rather than a new asymmetry.
- No new compiles: the conversion is in-process; the evaluator factory and module/ADT
  injections are already wired on the executor at this point in the runner (the pipeline
  result is memoized per file since #1328's fix, `runInlineCompile`).

### Option B (REJECTED as primary, kept as future work): extend the literal set in Go

Grow `EvaluateLiteral` with arms for `List`/`Tuple`/`Record`/`Identifier` (nullary ctor)/
`FuncCall` (applied ctor), constructing `eval.Value`s directly in Go and resolving
constructor names against the module's ADT declarations.

- Fixes F2 too, but adds a **fifth** expression mechanism that must forever track the
  evaluator's semantics (arity handling, `TaggedValue` layout, record field maps) in
  lockstep, with no shared test to enforce it. This is precisely the anti-pattern the
  design-doc audit exists to prevent: the next composite kind (or the VM parity test,
  `engine_parity_test.go`) re-opens the same bug.

### Option C (REJECTED as primary, kept as future work): full-pipeline elaboration of rows (the named-body mechanism)

Route both row arms through the source-append + re-elaboration path named tests use
(`EvaluateNamedTestBodyExprs`), gaining arithmetic via OpLowering.

- Rejected for now because it makes the **input** arm more capable than the expected arm
  fix requires, drags in the unresolved effectful-helper design
  ([m-named-test-effectful-helper.md](m-named-test-effectful-helper.md): entries are
  `pure func`, effectful callees are refused — while inline tests on effectful functions
  work today under `--caps`, V6, and must not regress), and costs a batched compile per
  file on both legs. When input-side elaboration is ever designed (with effect honesty),
  **both arms move together** and the grammar gate widens accordingly (§Future Work).

## Solution Design

### Overview

1. **M1 — expected arm evaluation**: new `Executor.EvaluateExpectedExpr(expr ast.Expr) (eval.Value, error)`
   = `astExprToCore` (error-returning) as the sole expression of a one-expression Core
   program, evaluated with `newHarnessEvaluator()`. `runner.go:176` switches to it. `EvaluateLiteral` is retired
   (its only production caller is this line, V3); its unit tests move to the new function.
2. **M2 — harden `astExprToCore`**: the `default: panic(...)` arm returns an error instead
   (signature change `astExprToCore(expr) (core.CoreExpr, error)`), and the two harness
   builders propagate it. The input-side process crash (V5) becomes a per-row typed
   failure. Property harnesses keep their behavior: the production ensures path is
   pre-lowered Core (`BuildEnsuresPropertyHarnessFromCore`, `harness.go:316`), so only the
   unit-test-only AST wrapper (`BuildEnsuresPropertyHarness`, `harness.go:303`) touches the
   changed signature.
3. **M3 — one grammar predicate, two consumers**: `RowExprSupported(expr ast.Expr) (ok bool, reason string)`
   in `internal/ast` (leaf; in the check closure). Accepts exactly what M1/M2 can run:
   `Literal`, `Tuple`, `List`, `Record`, `UnaryOp` (`-` only), `Identifier`, `FuncCall`;
   rejects `BinaryOp` and everything else with a reason string.
   - **Check leg**: a new pipeline stage (mirroring `effect_ceiling.go`) walks
     `result.Artifacts.AST` `FuncDecl.Tests` rows and appends errors with new codes
     **TST001** (row contains an expression form the test runner cannot evaluate) and
     **TST002** (row contains an arithmetic/comparison operator — rows are not elaborated;
     write the value, or move the case to a named test). Codes registered in
     `internal/errors/codes.go` (namespace verified unallocated, V12). Flows to
     `ailang check`, `ai-check`'s `check` JSON section, and the LSP with no per-command wiring.
   - **Test leg**: `runTest`'s inline branch calls the same predicate before evaluation and
     reports the same code/reason for rejected rows (fast-fail; today such rows fail later
     with a less explanatory message, or crash).
4. The legs **cannot disagree**: both read the one predicate. A locked-step premise test
   (table-driven) asserts that every expression form the predicate accepts actually
   evaluates under M1, and every rejected form produces TST001/TST002 on both legs.

### Architecture

**Components:**
1. `internal/testing/executor.go` — `EvaluateExpectedExpr` (~30 LOC); delete `EvaluateLiteral` (~55 LOC removed).
2. `internal/testing/harness.go` — `astExprToCore` error-returning (~20 LOC churn), builders propagate.
3. `internal/ast/` — `RowExprSupported` grammar predicate (~40 LOC, no imports beyond ast).
4. `internal/pipeline/validate_test_rows.go` — the check stage (~80 LOC, patterned on `effect_ceiling.go`), plus `internal/errors/codes.go` registry entries.
5. `internal/testing/runner.go` — switch line 176; add pre-eval predicate check (~15 LOC).

### Implementation Plan

**Phase 1: expected arm + panic hardening** (~1 day)
- [ ] M2: `astExprToCore` returns errors; fix the two builders + unit-test callers.
- [ ] M1: `EvaluateExpectedExpr`; switch `runner.go`; port `EvaluateLiteral` unit tests.
- [ ] Premise/acceptance tests: `u2b` (ADT nullary + applied), list, nested list-of-records, tuple, record rows all pass `ailang test` (V1/V2 repros become fixtures).

**Phase 2: the shared gate** (~1 day)
- [ ] M3 predicate in `internal/ast`; TST001/TST002 in `internal/errors/codes.go` + registry.
- [ ] Pipeline stage `validate_test_rows`; check/ai-check/LSP surface them.
- [ ] Runner fast-fail with the same codes.
- [ ] Locked-step table test: predicate-accepts ⇒ evaluates; predicate-rejects ⇒ same code on both legs.
- [ ] Regression sweep: `examples/inline_tests_*.ail` (5 files), `make test`, `make check-boundaries`.

### Files to Modify/Create

**New files:**
- `internal/pipeline/validate_test_rows.go` — check-leg stage (~80 LOC)
- `internal/ast/test_row_grammar.go` (or similar) — shared predicate (~40 LOC)
- `internal/testing/testdata/` fixtures: `u2b` (ADT), `listexp` (list), `row_arith_negative` (arithmetic), `row_lambda_negative` (panic repro)

**Modified files:**
- `internal/testing/executor.go` — add `EvaluateExpectedExpr`, delete `EvaluateLiteral`
- `internal/testing/harness.go` — error-returning `astExprToCore`
- `internal/testing/runner.go` — expected-arm switch + fast-fail
- `internal/errors/codes.go` — TST001, TST002 + registry rows
- `internal/pipeline` stage registration (where `effect_ceiling` is wired)

## Conflict Surface

This touches `internal/testing`, `internal/ast`, and `internal/pipeline` (a surface-AST
validation stage, same class as `effect_ceiling.go`). No parser/lexer/typechecker change.

1. **Positions extended**: the *expected* arm of `tests [...]` rows gains evaluation of
   composite expressions (list/tuple/record/ADT/identifier/call). The *input* arm's grammar
   is unchanged — only its failure mode (panic → typed error).
2. **Other valid constructs already living in those positions**: the parser accepts **any**
   expression inside a row (lambdas, matches, string interpolation, arithmetic, HOFs all
   parse). The predicate is what decides runnability, post-parse; nothing syntactic is
   disambiguated. Rows with `BinaryOp` (arithmetic/comparison), lambdas, matches etc. keep
   parsing and are now *rejected by name* at check time instead of failing (or crashing) at
   test time.
3. **Parser/typechecker disambiguation**: none needed — no grammar change. The check stage
   runs on the parsed surface AST after the type check, so it never competes with
   elaboration.
4. **Programs that MUST still work** (regression fixtures, all exist):
   - `examples/inline_tests_types.ail`, `examples/inline_tests_nullary.ail`,
     `examples/inline_tests_arithmetic.ail`, `examples/inline_tests_recursive.ail`,
     `examples/inline_tests_best_practices.ail` (the entire passing scalar corpus).
   - Effectful functions under test with `--caps IO` (V6 live repro): the input-arm
     harness and its capability gating are untouched.
   - Property/ensures legs: production uses pre-lowered Core predicates
     (`BuildEnsuresPropertyHarnessFromCore`); only unit tests use the AST wrapper.
   - Named test blocks: untouched by this doc.
   - Multi-arg rows `((a, b), expected)` on N-ary functions (the working form today).
5. **Deliberate changes (intentional incompatibilities)**:
   - Rows containing `BinaryOp` or unsupported node kinds change status from
     check-rc-0 + test-failure/crash → check-rc-1 (TST001/TST002) + test fast-fail. No
     currently-passing corpus can contain these rows — argument evaluation is eager, so
     even an ignored `1+2` input fails today (V4, confirmed live with an input-ignoring
     function).
   - `EvaluateLiteral` is deleted (no production caller remains after the switch, V3).

## Verification Log

Binary used for live checks: installed `ailang` v0.52.5, commit `7200786` (2026-10-07);
source claims read at origin/dev `1dfd5615` (shallow checkout, both post-triage `658ff76a3`).

| # | Claim | Evidence |
|---|---|---|
| V1 | F2 ADT repro: both nullary (`*ast.Identifier`) and applied (`*ast.FuncCall`) expected ctors fail at test time; `check` rc=0 `✓ No errors found!` | Live: `/tmp` repro of the issue's `u2b`; transcripts captured 2026-10-08. Matches the 2026-08-04 issue comment verbatim. |
| V2 | F2 list repro: `got *ast.List` at test time; `check` rc=0 | Live: `tl` with `tests [ (([1, 2, 3]), [2, 3]) ]`. Matches the 2026-10-08 triage comment verbatim. Input-side list literals already evaluate (only the expected arm fails). |
| V3 | `EvaluateLiteral` accepts scalars only; `runner.go:176` is its only production caller | Code read `executor.go:512-565`; `grep -rn "EvaluateLiteral" internal/ cmd/` → definition, its self-recursion, and `runner.go:176` only (plus `_test.go`). |
| V4 | Arithmetic in a row input fails: `BinOp reached evaluator; dictionaries not elaborated (op='+')`; check rc=0 | Live: `add` with `tests [ ((1+2, 3), 6) ]`; row 2 without arithmetic passes, proving the mechanism not the function. **Confirmed eager**: a function that ignores its input (`konst(x: int) -> int { 7 }`) still fails on the `1+2` row — so no currently-passing corpus can contain a `BinaryOp` row. |
| V5 | Unsupported input node **panics the process** (rc=2) through `astExprToCore`'s default arm; check rc=0 | Live: `L.map(\x. x, [1, 2])` as row input → stack trace `harness.go:247` via `buildFunctionCall`/`BuildInlineTestHarness`; `ailang test` exits 2, no Test Results section. |
| V6 | Inline tests on effectful functions run today under `--caps` (capability gate in the harness evaluator) | Live: `shout(n: int) -> string ! {IO}` with rows fails with `effect 'IO' requires capability, but none provided` — the gate exists and is the behavior to preserve. |
| V7 | `equalValues` already deep-compares `ListValue`, `TupleValue`, `RecordValue`, `TaggedValue` | Code read `executor_helpers.go:123-205`. (Known quirk, out of scope: `TaggedValue` compares `CtorName` only, not `TypeName`.) |
| V8 | AST→Core→evaluator already round-trips ADT/list/record values | `TestB12_RoundTrip` (`value_splice_roundtrip_test.go:61`) asserts structural equality across the exact cycle M1 needs; `valueToLiteral` (`value_splice.go`) emits the same AST node kinds the predicate will accept. |
| V9 | No check-time validation of test rows exists (negative-existence) | `grep -rn "Tests" internal/types/` → only a wasm comment; `internal/elaborate/file.go:421` copies `f.Tests` into SCC metadata and nothing else reads it pre-test. |
| V10 | `check` ignores named test bodies too | Live: `test "bad body" { 1 + "x" }` → check rc=0; test fails with the type error. |
| V11 | The pipeline exposes the surface AST; a surface-AST validation stage is established practice | `internal/pipeline/compile_unit.go:24` (`Surface *ast.File`), `canonical_json.go:109` (`result.Artifacts.AST`), `effect_ceiling.go:100` (`validateEffectCeiling(surfaceAST, ...)`). |
| V12 | `TST` error-code namespace is unallocated | `grep -rn '"TST' internal/ cmd/ --include='*.go'` → empty; `internal/errors/codes.go` has no TST family. (The multiarg doc *proposes* TST codes but defines none.) |
| V13 | `internal/testing` is not in the check binary's language closure | `grep -n "internal/testing" ARCHITECTURE.md` → no hit; the 49-package closure list does not include it — hence the gate lives in `internal/pipeline`/`internal/ast`, not by making `check` import the runner. |
| V14 | Inline-test compiles are memoized per file (M1 adds no pipeline runs) | `inline_memo_test.go` + `ailang-core-triage/inline-test-per-case-recompile-1328.md` (FIXED 2026-10-06, `Executor.runInlineCompile`). |
| V15 | `injectADTConstructors` binds only the source file's own ADT constructors (imported-module ctors like `Some` are not bound) | Code read `executor_helpers.go:456-492` — a pre-existing, symmetric (both arms) limitation; documented, not widened here. |

## F3 Re-verification (composed contract body fails Z3 encoding at rc=0) — 2026-10-08

Re-verified live per the task directive; **not reproducible in the shapes tried, and the
rc=0 leg is fixed**:

1. **Composed bodies verify.** Same-module composed `ensures` calling two other predicates
   (bool), composed `ensures` over ADT matches (`match e { Log(m) => ..., Cap(n,c) => ... }`),
   and **cross-module** composed contracts (`import world/preds as P`; `ensures { result == (match e { ... P.nameOk(m) ... }) }`) all report `✓ VERIFIED` (z3 4.8.12, rc=0).
2. **The "rc=0 despite Z3 errors" behavior is fixed.** A genuine live Z3 encoding error —
   `ensures { result <= a }` over `a / 2.0` (float) → `(error "invalid constant definition, sort mismatch")` + `(error "unknown constant result")` — now exits **1** on both legs:
   `ailang verify` prints `! ERROR half` (rc=1) and `ai-check` reports `"errors": 1` (rc=1).
   Mechanism (code read): `aiCheckExitCode` returns 1 when `verify.Errors > 0`
   (`cmd/ailang/ai_check.go:203-208`, added by M-Z3-ADT-RECORD-SORT), and `a7470ce4d`
   (#1561, 2026-10-03) made the `verify` leg exit 1 on solver errors.
3. **The machinery that produced the original errors has been reworked since filing.**
   The original sketch's `unknown constant effectNameMatches` matches the documented
   ai-check drift (no `ImportedPrograms` → every cross-module callee an "unknown constant"),
   folded into ONE verification loop by M-V1-SIMPLIFY-S4 M3A (`6a03975d1`); callee resolution
   now inlines up to 3 hops or falls back to contract-as-spec
   (`internal/smt/callee_resolver.go:104-140`), nullary callees encode as constants
   (`9b6584612`), unencodable callee sorts skip honestly with a named reason
   (`d949db2f2`, live-verified: `SKIPPED total — calls "foldl" whose signature uses an
   unencodable type "b"`), and literal/guarded matches encode (`a7470ce4d`, `a5b3b1440` #757).
4. **Residual, out of F3's scope:** the float division contract above still *errors* (an
   encoding gap for float `/`), but it is loud, exit-coded, and a different defect class
   from F3's composition failure.

**Recommendation:** close F3 as fixed / cannot-reproduce. The original 213-line sketch was
never attached to the issue; if the reporter re-supplies it and a composed predicate still
fails Z3, that is a new report against the current encoder. The rc contract it relied on
(process status disagreeing with the JSON) no longer exists.

## Examples

### Example 1: ADT expected values (the issue's own repro) — Before/After

**Before** (live, V1): `check` → `✓ No errors found!` (rc=0); `test` → both rows fail
(`expected literal expression, got *ast.Identifier` / `*ast.FuncCall`).

```ailang
type Decision = Allow | Deny(string)
export func decide(ok: bool) -> Decision ! {}
tests [ ((true), Allow), ((false), Deny("nope")) ]
{ if ok then Allow else Deny("nope") }
```

**After:** both rows pass; `check` stays rc=0 (grammar satisfied).

### Example 2: the false green becomes a loud red

**Before** (live, V4/V5): check rc=0; `test` → row fails with
`BinOp reached evaluator; dictionaries not elaborated` — or, for a lambda in the row, the
whole `ailang test` process crashes (rc=2).

```ailang
export pure func add(a: int, b: int) -> int ! {}
tests [ ((1+2, 3), 6) ]
{ a + b }
```

**After:** `ailang check` fails with
`TST002 (row 1): arithmetic operators in a tests row are not evaluated; write the value (6), or move this case to a named test block` at the row's location; `ailang test` reports the same code for that row instead of a mid-harness failure. (Named test blocks remain the escape hatch for anything needing elaboration — their bodies run the full pipeline.)

## Success Criteria

- [ ] The `u2b` fixture (ADT nullary + applied) passes `ailang test` (acceptance: both rows green).
- [ ] A list expected `[2, 3]`, a nested list-of-records, a tuple and a record expected all pass.
- [ ] `ailang check` on a row with arithmetic (V4 fixture) exits 1 with TST002 and the row's location; `ai-check` JSON carries it in `check.errors`.
- [ ] `ailang check` on a row with an unsupported node (V5 fixture) exits 1 with TST001.
- [ ] `ailang test` on the same fixtures fails the row with the same code — never a panic, never a process crash (rc=1 with a report).
- [ ] Locked-step premise test: for every AST node kind, `RowExprSupported` acceptance matches `EvaluateExpectedExpr` success (table-driven, both directions).
- [ ] All 5 `examples/inline_tests_*.ail` unchanged in outcomes; `make test`, `make check-boundaries` clean.
- [ ] Effectful inline tests still run under `--caps` (V6 fixture unchanged in message and rc).

## Testing Strategy

**Unit tests:** predicate table (every node kind × both verdicts); `EvaluateExpectedExpr` per value shape; `astExprToCore` error path (no panic); codes registry presence.

**Integration tests:** the V1/V2/V4/V5/V6 fixtures as `internal/testing/testdata/` + `cmd/ailang` end-to-end (check rc, test rc, JSON fields); `ai-check` JSON includes TST rows.

**Manual testing:** LSP diagnostics surface TST001/TST002 on an open file with a bad row (the stage rides the same pipeline); `--bytecode` path unaffected for inline rows (expected comparison is Go-side).

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Evaluate expected via `astExprToCore`+evaluator (Option A) vs extend `EvaluateLiteral` (B) vs elaborate rows (C) | Sets the row grammar for both legs and whether a 5th expression mechanism exists | agent (this doc recommends A; user approves doc) | design | med |
| Grammar excludes ALL `BinaryOp` (incl. comparisons) symmetrically | Defines what check rejects; could over-reject rows that happen to work today (none exist — eager arg evaluation, V4) | agent | design | low |
| Gate lives in `internal/pipeline` stage (not cmd wiring, not runner import) | Keeps `internal/testing` out of the check closure (V13); one call site feeds check, ai-check, LSP | agent | design | med |
| TST001/TST002 codes (new family) | Shared namespace; multiarg doc also proposes TST — this defines the family first | agent | design | low |
| Retire `EvaluateLiteral` | Public-ish API of internal package; grep-verified single caller | agent | compile | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [ ] Option A vs B/C confirmed by doc approval (user approval of this doc resolves it).

## Deferred Decisions

The following are intentionally left open for the implementer:

- Predicate placement (`internal/ast` vs `internal/testing` + thin re-export) — agent may choose, provided the check closure stays clean (V13).
- Whether the runner evaluates all expected exprs of a file in one batched Core program — agent may choose (per-row is the baseline; both are in-process).
- Exact TST message wording/hints — agent may choose; must name the row and the offending construct.
- UnaryOp widening beyond `-` (e.g. `not`) — agent may choose IF a premise test proves the evaluator path.

## Non-Goals

**Not attempted in this feature:**
- **Row shape and arity rulings** (flat rows, tuple-input arity disambiguation, mis-arity
  refusal) — owned by [m-inline-test-multiarg-tuple-rows.md](../m-inline-test-multiarg-tuple-rows.md).
  Until it lands, `check` remains silent on mis-arity rows; this is documented, not fixed here.
- **Named-test-body visibility in `check`** (V10) and **properties[] row validation** — same
  disease, different surface; folding them in would triple the blast radius. Flagged as the
  natural follow-up once the TST stage exists (the stage can grow a named-body pass later).
- **Arithmetic/stdlib-call expressions in rows** (Option C) — deferred until input-side
  elaboration is designed with effect honesty (see m-named-test-effectful-helper).
- **Imported-module ADT constructors in the harness env** (`Some(1)` expected, V15) and the
  `TaggedValue` TypeName-less comparison (V7 quirk) — pre-existing, symmetric, unchanged.
- **F1 (requires vs derived ensures)** — fixed by a9e26ffd6 (PR #549).
- **F3** — re-verified fixed/cannot-reproduce; see §F3. No work item.

## Timeline

**Week 1** (~2 days):
- Phase 1 (expected arm + hardening): day 1
- Phase 2 (gate + codes + locked-step tests): day 2, with the regression sweep

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| `astExprToCore` signature change ripples into property-harness unit tests | Med | Mechanical; production ensures path uses pre-lowered Core (`harness.go:316`); sweep is one package. |
| Grammar over-rejects a row shape some corpus uses today | Med | No such corpus can exist: rows rejected by the predicate fail evaluation today (V4) or crash (V5); the locked-step test plus the examples sweep guards the boundary. |
| Two TST-cataloguing docs (this + multiarg) drift apart | Med | §Relationship names the sequencing; sprint-planner should sequence multiarg's collection gates to migrate into this pipeline stage rather than add a second one. |
| Eval of identifier/call expected values resolves names the checker cannot see (typos surface only at test time) | Low | Documented residual; the grammar is honest about what it guarantees (grammar, not name resolution). |

## Relationship to `m-inline-test-multiarg-tuple-rows.md`

That planned doc (issue #715; target v0.39.0 — stale) owns the **input-side row grammar**:
row parse shape, arity-based desugaring, and refusal of unconstructible rows (its Ruling 3
also proposes "TST family codes" without defining any). This doc owns the **expected arm,
the panic hardening, and the check-leg stage**, and **defines the TST family** (TST001/002).
Overlap management, for the sprint planner:

- This doc's `astExprToCore` error-return refactor is the shared substrate for the
  multiarg doc's "refuse, don't panic" ruling — implement once, here.
- The multiarg doc's collection-time shape/arity gates should migrate into this doc's
  pipeline stage when that doc is implemented, so `check` sees them too (its own gates at
  collection time would leave the check leg silent — the exact disease this doc removes).
- Its target (v0.39.0) predates five releases; re-triage before sequencing.

## Related Documents

**Planned (check for overlap):**
- [m-inline-test-multiarg-tuple-rows.md](../m-inline-test-multiarg-tuple-rows.md) — input-side row grammar (see §Relationship)
- [m-test-runner-compile-once.md](m-test-runner-compile-once.md) — named-test one-compile batch; the named-body mechanism Option C defers to
- [m-named-test-effectful-helper.md](m-named-test-effectful-helper.md) — effect honesty for batched entries (why Option C is deferred)
- [m-package-test-discovery.md](../m-package-test-discovery.md) — inline tests under `ailang test --package` (gate must not depend on run mode)
- [inline-test-per-case-recompile-1328.md](../ailang-core-triage/inline-test-per-case-recompile-1328.md) — inline compile memoization (V14)

**Implemented (may inform design):**
- [m-test-harness-module-scoped-envs.md](../../implemented/v0_52_0/m-test-harness-module-scoped-envs.md) — the module-scoped harness evaluator M1 reuses (fixed #1574's collision leg)
- [m-property-generator-coverage.md](../../implemented/v0_31_0/m-property-generator-coverage.md) — the value-splice refusal pattern (`valueToLiteral`) M1 mirrors in the opposite direction
- [m-dx26-ensures-result-binding-sprint-plan.md](../../implemented/v0_20_0/m-dx26-ensures-result-binding-sprint-plan.md) — `result` binding in the ensures harness, which M1's one-expression evaluation parallels

**Issues:**
- [#495](https://github.com/sunholo-data/ailang/issues/495) — this doc (F2; F1 fixed; F3 re-verified)
- [#1574](https://github.com/sunholo-data/ailang/issues/1574) — closed; its residual (list expected) is F2's list case
- [#1328](https://github.com/sunholo-data/ailang/issues/1328) — inline compile memoization (context, V14)

## References

- [Design Axioms](/docs/references/axioms) — the 12 non-negotiable principles
- [ARCHITECTURE.md](/ARCHITECTURE.md) — layering; the language-closure constraint (V13) behind the gate's placement
- #495 triage comments (2026-08-04, 2026-10-08) — live verification history

## Future Work

- **Elaborated rows (Option C):** move both row arms onto the named-body re-elaboration
  path (OpLowering, arithmetic, stdlib calls) once effect honesty for batched entries is
  designed; widen the shared grammar predicate in the same change so the legs keep
  agreeing.
- **Named-body and properties[] check visibility:** grow the TST pipeline stage to
  elaborate named test bodies and properties rows, closing the whole-surface version of
  V10.
- **Imported-module constructors in the harness env** (V15) and TypeName-qualified
  TaggedValue comparison (V7 quirk).

---

**Document created**: 2026-10-08
**Last updated**: 2026-10-08
