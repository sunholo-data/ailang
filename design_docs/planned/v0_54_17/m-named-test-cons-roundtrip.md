# M-NAMED-TEST-CONS-ROUNDTRIP: `::` in named-test and property bodies must survive the harness re-parse

**Status**: Planned
**Target**: v0.54.17
**Priority**: P1 (Medium) — a whole test file fails on valid source, but a workaround exists (`[x] ++ xs`, or hoisting bodies into named `pure` funcs) and is already deployed downstream
**Estimated**: 1 day (implementation ~2h, tests ~3h, docs/verification ~1h, ~2h buffer)
**Dependencies**: None. Builds on the named-test batch compile (`m-test-runner-compile-once`, planned v0_53_0, Phases 1–2 implemented 2026-10-06) and is the third documented bug in the same round-trip class (#590 assert, #1821 strip, this).

**Issue**: Reported by the stapledon mission loop, iteration 42 (R2-SIMILAR-S0 S0.1), 2026-10-20, against v0.54.14 (e1ef33f) and v0.54.16-dirty (128371f) — identical failure. Downstream: stapledons-godot `sim/similar_test.ail` writes the cons-free equivalent (`[x] ++ xs`) in test bodies with a comment citing this report.

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No evaluation-semantics change; the re-parse desugars back to the byte-identical `FuncCall{"::"}` node the first parse produced (V2, V3). |
| A2: Replayability | 0 | `--seed`, report format, and run order unchanged. |
| A3: Effect Legibility | +1 | Restores the intended diagnostic contract: an effectful named-test body is *effect-checked and refused* (named_test_effectful diagnostic), not pre-empted by a bogus parse error naming a temp file the user never wrote. |
| A4: Explicit Authority | 0 | No capability change. |
| A5: Bounded Verification | +1 | `ailang test` becomes a sound local check for the documented test form: today a cons-bearing body that `ailang check` accepts fails `ailang test` with rc 1 (V1). |
| A6: Safe Concurrency | 0 | No concurrency change. |
| A7: Machines First | +1 | Removes a trap that punishes AI-generated code for using the language's own cons operator in the language's own test form; the current failure mode (parse cascade + "could not share one compile") misroutes both machines and humans. |
| A8: Minimal Syntax | 0 | No new syntax; the fix re-uses the existing `::` infix sugar inside generated source only. |
| A9: Cost Visibility | 0 | No resource-cost change. |
| A10: Composability | +1 | One printer fix covers all six call sites (named-test batch, per-body fallback, VM path, property batch, property single-compile, check-failure message source). |
| A11: Structured Failure | 0 | No new error codes; no error semantics change. |
| A12: System Boundary | 0 | No boundary change. |

**Net Score: +4** → **Decision: proceed to implementation.**

### Hard Violation Check

- [x] A1 (Determinism): no implicit nondeterminism introduced — printed text is a pure function of the AST.
- [x] A3 (Effects): no hidden side effects — the harness printer is pure; re-parse desugar is the parser's existing behavior.
- [x] A4 (Authority): no ambient access granted.
- [x] A7 (Machines First): not optimizing for human convenience; this restores machine-decidable test parity with `ailang check`.

## Problem Statement

`ailang test` round-trips every named-test and forall-property body through
**print → re-parse**: the folded AST is printed back to AILANG source
(`PrintAILANGSource`, `internal/testing/executor_helpers.go:273`), appended to the stripped module as
a `pure func __namedtest_N() { <body> }` entry, and compiled through the general
pipeline (`runNamedTestPipeline`, `internal/testing/executor.go:292`).

The cons operator `x :: xs` never reaches that printer as an operator node: the
parser desugars it **at parse time** to a constructor call
(`internal/parser/parser_expr.go:475 parseConsExpression`, :498):

```go
&ast.FuncCall{Func: &ast.Identifier{Name: "::"}, Args: []ast.Expr{left, right}}
```

`PrintAILANGSource`'s generic `*ast.FuncCall` case
(`internal/testing/executor_helpers.go:318`) prints any call as
`Func(args)` — so cons prints as **`::(x, xs)`**. But `::` is the lexer's
`DCOLON` token, which has an **infix parselet only**
(`internal/parser/parser.go:128`); there is no prefix parselet for it in the
expression grammar (verified: `grep registerPrefix(lexer.DCOLON` over
`internal/parser/` is empty — V6). The re-parse of the generated entry therefore
fails:

```
PAR_NO_PREFIX_PARSE at r.ail:5 (test body): unexpected token in expression: ::
PAR_UNEXPECTED_TOKEN ... (a four-error cascade)
```

The canonical formatter already hit and solved this exact problem —
`internal/format/expr.go:154`:

> "The cons operator `a :: b` desugars to FuncCall{Func: `::`, Args: [a, b]}.
> `::` is an infix-only operator: the parser has no prefix/call form for it,
> so `::(a, b)` does NOT re-parse. It must be re-emitted in infix form to
> round-trip."

The test-harness printer was never given the same case.

**Current state (reproduced on this tree, binary v0.54.16, commit 4cc0f6a-dirty,
2026-10-19):**

```ailang
module r
import std/list (length)
pure func identity(xs: [int]) -> [int] = xs
export pure func consRun(n: int) -> int { let x = 1; let xs = [2]; length(identity(x :: xs)) }
test "cons argument" { let x = 1; let xs = [2]; length(identity(x :: xs)) == 2 }
```

- `ailang check r.ail` → rc 0, "No errors found!" (V1)
- `ailang run --quiet --entry consRun --args-json 0 r.ail` → rc 0, prints `2` (V1)
- `ailang test r.ail` → rc 1, `PAR_NO_PREFIX_PARSE at r.ail:5 (test body):
  unexpected token in expression: ::` + cascade, after the stderr notice
  `named tests in r.ail: could not share one compile (…); compiled each test separately` (V1)
- forall properties fail identically: `property "cons" { forall(xs: [int]) =>
  length(1 :: xs) == length(xs) + 1 }` → `PAR_NO_PREFIX_PARSE … (property)` (V4)

The reporter's controls all hold on this tree (V1, V4): parenthesised
`identity((x :: xs))` fails the same way; cons in a plain `let` in the body
fails the same way; cons in a **module-level function called from the test**
passes — because the stripped module retains the user's original function text
verbatim and only the **entry body** is printed by the harness. `ailang check`
accepts all of it, because the first parse desugars correctly everywhere.

**Impact:**

- Any AI- or human-authored test that uses the language's own list constructor
  in its own documented test form fails at rc 1 with a misleading parse error
  naming a temp file (`…/ailang-namedtest-NNN/r.ail`) the user never wrote.
- Because one batch is shared per file, one cons anywhere in one test poisons
  the shared compile (`BatchFailure`, `internal/testing/named_batch.go:82`) —
  the file then recompiles per body and fails again.
- The failure *silently misroutes*: it looks like a user syntax error (`ailang
  check` disagrees), it appears under a synthesized filename, and the
  workarounds pushed downstream (stapledons-godot's `[x] ++ xs`) leave the
  wrong canonical form in a consumer repo.

## Goals

**Primary Goal:** `ailang test` and `ailang check` agree on every named-test
and property body: any body `ailang check` accepts, `ailang test` compiles —
`::` included — with the desugared node surviving the round-trip unchanged.

**Success Metrics:**

- The reported repro (`r.ail`) goes rc 1 → rc 0, `1 tests: 1 passed`.
- A forall property whose predicate uses `::` compiles and runs (rc 0).
- `--bytecode` and `--strict-bytecode` paths agree (the VM entry source is
  printed by the same function, `internal/testing/bytecode_engine.go:112`).
- Every `PrintAILANGSource` caller is covered by one fix (no per-call-site
  patching): 6 call sites, 1 changed function.
- All existing named-test/assert/property fixtures keep their exact pass/fail
  patterns (`testdata/named_batch/mixed.ail` unchanged).

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Fix the harness printer (re-emit infix), not the parser (add a `::` prefix parselet) | The parser option changes the *language surface* (new accepted syntax) and needs its own design/docs/prompt update; the printer option is the already-ratified formatter precedent (format/expr.go:154) | human (this doc ratifies it) | design | low |
| Unconditional parentheses around the re-emitted cons `(a :: b)` | The harness printer is string-based with no precedence tracking; a bare `a :: b` spliced into `(… == 2)` would re-associate (`CONS` prec 10 > comparison prec) and silently change semantics | agent | implementation | low |
| Shape check `Identifier{Name:"::"} && len(Args)==2`, fall through otherwise | A `::` call with ≠2 args only exists on already-broken ASTs (strict-mode placeholder); those keep today's generic print (and today's failure), never a new silent path | compiler (AST shape is parser-defined) | compile | low |
| No new error code, no diagnostics change | The fix removes a wrong diagnostic; it does not introduce one | agent | implementation | low |
| Scope is `internal/testing` only; the strict-mode suggestion inconsistency (`::(x, xs)` is recommended by `reportSugarError` but does not parse) is routed out as its own follow-up | Touching the expression grammar is a language change with a different blast radius than this harness bug | human | design | med |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] Fix location: `PrintAILANGSource` (harness printer) mirrors `internal/format/expr.go:154`; the parser grammar is NOT extended in this sprint.
- [x] Emitted form: parenthesised infix `(<left> :: <right>)` in all contexts (no precedence-dependent parenthesisation).
- [x] Fallback for non-2-arity `::` calls: generic `::(...)` print, unchanged from today (broken-AST stays loud, not silently reshaped).

## Deferred Decisions

The following are intentionally left open for the implementer:

- Exact test file names for the new unit tests — agent may choose (style precedent: `named_test_assert_test.go`, `source_strip_brace_test.go`).
- Whether the cons fixture lives in `testdata/named_batch/cons.ail` (new file) or extends `mixed.ail` — agent may choose; a new file avoids re-basing the expected pass/fail table asserted in `named_batch_test.go`.
- Comment wording in the printer case — agent may choose (mirror format/expr.go:154's comment or shorten).
- Whether to also assert the printed output for `CheckInfo.Source` display (assert-failure messages) in a test — agent may choose.

## Solution Design

### Overview

Give `PrintAILANGSource` the same cons case the canonical formatter has had
since the desugar was introduced: when a `*ast.FuncCall` has
`Func == Identifier{Name: "::"}` and exactly 2 args, print `(a :: b)` instead
of the generic `::(a, b)`. The re-parse desugars it back to the byte-identical
`FuncCall{"::", [a, b]}` the first parse produced (`parser_expr.go:498`),
so downstream stages (elaboration `internal/elaborate/expr_calls.go:18`,
builtin registry `internal/builtins/registry.go:236` `"::"` `NumArgs: 2`,
`IsPure: true`) see exactly what a direct compile of the original file sees.

This is a **one-function fix** in `internal/testing`; all six printer call
sites are repaired at once:

| Call site | File:line | What it prints |
|---|---|---|
| per-body evaluator entry | `internal/testing/executor.go:245` | folded named-test body |
| shared batch (test entry) | `internal/testing/named_batch.go:122` | folded named-test body |
| shared batch (property entry) | `internal/testing/named_batch.go:133` | forall predicate |
| single property compile | `internal/testing/property_batch.go:68` | forall predicate |
| VM entry (`--bytecode`) | `internal/testing/bytecode_engine.go:112` | folded body, `-> bool/int` annotated |
| assert-failure message | `internal/testing/test_body_lowering.go:161` | check condition source (display) |

### Architecture

**Components:**

1. **`PrintAILANGSource` cons case** (`internal/testing/executor_helpers.go`,
   inside the `*ast.FuncCall` arm at :318): shape-check the call, re-emit
   infix with unconditional parens. Falls through to the existing generic call
   print for every other shape (including `::` with ≠2 args).
2. **No parser change.** `parseConsExpression` (infix desugar), the pattern
   prefix form (`parser_pattern.go:83`, already accepts canonical `::(h, t)`),
   and the builtin all stay as-is.
3. **Regression fixtures + tripwire test** (see Testing Strategy) pin the
   round-trip so the next desugar added to the parser without a printer case
   fails a named test, not a downstream consumer.

### Implementation Plan

**Phase 1: printer case** (~2h)
- [ ] Add the cons case to the `*ast.FuncCall` arm of `PrintAILANGSource`, with a comment citing format/expr.go:154 and this doc.
- [ ] Unit tripwire `TestPrintAILANGSource_ConsFuncCallReEmitsInfix` (directional, in the style of `TestPrintAILANGSource_AssertStmtIsTripwire`, `named_test_assert_test.go:317`): parse a cons body → `FoldTestBody` → `PrintAILANGSource` → (a) output contains ` :: ` and not `::(`, (b) re-parsing the output yields no parse errors, (c) the re-parse desugars back to `FuncCall{Identifier{"::"}, [x, xs]}`.

**Phase 2: fixtures + end-to-end** (~3h)
- [ ] Fixture `internal/testing/testdata/named_batch/cons.ail`: cons in a passing test body; cons inside a call argument; cons in a plain `let` in the body (the reporter's control cases); cons in a match *pattern* (pattern round-trip, already working — pins it); `[x] ++ xs` (the downstream workaround); an interpolated string (concat_String desugar chain round-trip).
- [ ] Fixture-driven test in the `named_batch_test.go` style (`engineFixtureFrom(t, filepath.Join("testdata", "named_batch", "cons.ail"))`): all cases pass, batch shared (no `BatchFailure` notice on stderr).
- [ ] Property case: forall predicate with `::` compiles and passes (`property_batch.go` path).
- [ ] cmd/ailang e2e in the `test_bytecode_test.go` pattern (`buildAilang(t)` + `runAilangBin`): the reporter's exact `r.ail` repro → rc 0, 1 passed; same file under `--bytecode` and `--strict-bytecode` → identical verdict.

**Phase 3: docs + downstream** (~1h)
- [ ] CHANGELOG entry (v0.54.17, `changelogs/v0.32-current.md` Unreleased).
- [ ] Notify stapledons-godot: revert `sim/similar_test.ail` to `x :: xs` (their comment cites this report).
- [ ] Route the strict-mode inconsistency as its own issue: `reportSugarError("CONS", "x :: xs", "::(x, xs)")` (`parser_expr.go:480`) recommends a canonical form that does not parse — a language-surface fix, deliberately out of scope here (see Non-Goals).

### Files to Modify/Create

**New files:**
- `internal/testing/executor_cons_test.go` (~80 LOC) — tripwire + fixture tests (or fold into an existing printer test file; agent's choice)
- `internal/testing/testdata/named_batch/cons.ail` (~15 LOC) — fixture
- `cmd/ailang/test_cons_test.go` (~60 LOC) — e2e repro (evaluator + bytecode paths)

**Modified files:**
- `internal/testing/executor_helpers.go` (+10/-0 LOC) — the cons case in `PrintAILANGSource`
- `changelogs/v0.32-current.md` (+1 section) — Unreleased entry

## Conflict Surface

### Syntactic positions touched

The change extends exactly one print position: the `*ast.FuncCall` arm of
`PrintAILANGSource` (`internal/testing/executor_helpers.go:318`), whose output
is spliced into generated `pure func __namedtest_N() { <body> }` entry bodies
(and the check-source string of assert-failure messages). It does **not**
touch the lexer, the parser, the pattern grammar, types, elaboration, eval,
or the VM.

### What else lives in that position

Every other `*ast.FuncCall` the folded body can contain — each keeps today's
output byte-for-byte (only the `::`-2-arg shape changes):

| Shape | Existing valid form | Printed as | Round-trips |
|---|---|---|---|
| ADT constructor call | `Some(1)` | `Some(1)` | yes (prefix call is real syntax) |
| builtin/imports call | `length(xs)` | `length(xs)` | yes |
| zero-arg call | `f()` | `f()` (zero-arg/unit special case at :323–330) | yes (S-CALL0 re-desugars) |
| interpolated string desugar | `"a${x}b"` → `concat_String("a", show(x), "b")` | same, as calls | yes (verified V5: an interp body passes `ailang test` today) |
| cons desugar **(this fix)** | `x :: xs` → `FuncCall{Identifier{"::"}, [x, xs]}` | was `::(x, xs)` (**not syntax**, V2/V6) → now `(x :: xs)` | yes, desugars back to the identical node |
| cons **pattern** in `match` | `x :: rest` → `ConstructorPattern{"::"}` → `String()` → `::(x, rest)` | unchanged | yes — the *pattern* grammar accepts the prefix form (`parser_pattern.go:83`); verified V7 (a match-on-cons body passes today and must keep passing) |
| strict-mode placeholder | `::` call with 1 arg (`parser_expr.go:479–488`) | falls through to generic print | n/a (broken-AST path, unchanged) |

### Disambiguation strategy

A single exact shape check in the printer, mirroring `format/expr.go:154`:
`Func` is `*ast.Identifier` with `Name == "::"` **and** `len(Args) == 2`.
Nothing else is affected: any other identifier named `::` cannot exist (a
user-written `::(x, xs)` never parses — V6 — so the AST shape is
parser-internal), and wrong-arity `::` calls only arise from the strict-mode
placeholder, which keeps today's generic print. No lookahead, no context
flags: the printer is a pure function of the AST node.

### Programs that MUST still work

Post-change regression surface (each exercised by Phase 2 fixtures/tests):

- `internal/testing/testdata/named_batch/mixed.ail` — the shared-batch fixture
  (assert sentinel bodies, runtime-error body, polymorphic `let`, lambda
  bodies): its exact pass/fail pattern is asserted by `named_batch_test.go` /
  `engine_parity_test.go` and must not shift.
- The workaround form `[x] ++ xs` inside a test body (BinaryOp `++` path —
  what stapledons-godot ships today): must keep passing.
- A test body with a `match` on a cons **pattern** (`x :: rest => …`) —
  pattern round-trip via `ConstructorPattern.String()`: must keep passing
  (verified V7 it passes today).
- A test body with an interpolated string: the concat_String desugar chain
  must keep re-parsing (verified V5 it passes today).
- `cmd/ailang/test_bytecode_test.go`'s `engine_test.ail` (evaluator vs
  `--bytecode` parity) — unchanged verdicts.

External consumer: stapledons-godot `sim/similar_test.ail` (and
`sim/protocol_ism_test.ail`, the #1821 reporter) — the restore-to-cons step
is the downstream acceptance test for this fix.

### What deliberately changes

Nothing. No previously-valid program changes meaning: the only text that
changes is the *generated* entry body of named tests and properties
(harness-internal), and the only programs that stop failing were failing
today with rc 1. The `[x] ++ xs` workaround remains valid (it is a different,
real operator). Anything not listed here that breaks is a regression, not an
intentional change.

## Testing Strategy

**Unit tests:**
- `TestPrintAILANGSource_ConsFuncCallReEmitsInfix` — directional tripwire (Phase 1): printed output contains ` :: `, not `::(`; re-parses clean; re-desugars to the identical `FuncCall` node.
- Keep `TestPrintAILANGSource_AssertStmtIsTripwire` green (the printer's other directional guard).

**Integration tests:**
- `testdata/named_batch/cons.ail` through the real `ailang test` runner: all bodies pass; stderr shows no `could not share one compile` notice (batch shared).
- Property fixture: forall predicate with cons, `property_batch` path.
- cmd/ailang e2e: reporter's exact repro on evaluator, `--bytecode`, and `--strict-bytecode` — identical verdicts.

**Regression-surface tests** (one per "Programs that MUST still work" above):
- `mixed.ail` expected pass/fail table unchanged (existing tests already assert this).
- cons-pattern body, interp body, `++` body — added to `cons.ail` as passing cases.

**Manual testing:**
- `ailang check` then `ailang test` on the reporter's `r.ail`: rc 0, `1 passed`.

## Verification Log

Premises checked before this doc was written (binary v0.54.16-dirty, commit
4cc0f6a-dirty, 2026-10-19; no Go toolchain in the reporting environment —
claims verified with the installed `ailang` binary and by reading the cited
code):

| # | Claim | How verified | Result |
|---|---|---|---|
| V1 | The reporter's exact repro fails: `check`/`run` rc 0, `test` rc 1 with `PAR_NO_PREFIX_PARSE … (test body): unexpected token in expression: ::` + cascade + `could not share one compile` notice | ran `ailang check` / `ailang run --quiet` / `ailang test` on the repro file | Confirmed, byte-similar to the report |
| V2 | `PrintAILANGSource` prints cons as `::(x, xs)` (generic FuncCall case, :318) | read `executor_helpers.go:273–446`; built the exact synthesized entry text by hand and `ailang check`ed it | Confirmed: hand-built entry with `::(x, xs)` at body position fails with the **identical** error signature |
| V3 | `::` desugars at parse time to `FuncCall{Identifier{"::"}, [l, r]}` | read `parser_expr.go:473–501`; `run` of `consRun` prints 2 (the desugared node elaborates via `expr_calls.go:18`, builtin `"::"` `registry.go:236`) | Confirmed |
| V4 | forall **properties** fail identically (`(property)` label) | `ailang test` on a file with `property "…" { forall(xs: [int]) => length(1 :: xs) == length(xs) + 1 }` | Confirmed: `PAR_NO_PREFIX_PARSE … (property): unexpected token in expression: ::` |
| V5 | Interpolated-string bodies round-trip today (concat_String desugar re-parses) | `ailang test` on a body with `"a${1}b"` | Confirmed: passes |
| V6 | **Negative-existence:** no prefix parselet for `DCOLON` in the expression grammar | `grep registerPrefix(lexer.DCOLON` over `internal/parser/` → empty; `parser.go:82–110` prefix table has none; `parser.go:128` registers DCOLON as **infix** only | Confirmed |
| V7 | Cons **patterns** in `match` inside test bodies pass today (pattern grammar accepts canonical `::(h, t)` at `parser_pattern.go:83`) | `ailang test` on a body with `match xs { x :: rest => length(rest) == 1 }` | Confirmed: passes |
| V8 | **Negative-existence:** `::(x, xs)` written directly in source does not parse (so the printed form is not user syntax) | `ailang check` on a file using `identity(::(x, xs))` | Confirmed: `PAR_NO_PREFIX_PARSE at …: unexpected token in expression: ::` |
| V9 | Strict-syntax mode is NOT enabled on the round-trip re-parse (so emitted sugar is legal there) | `pipeline.Config.StrictSyntaxMode` zero-value false; `runNamedTestPipeline` (`executor.go:318–325`) does not set it; `SugarUsed()` consumed only by the REPL | Confirmed |
| V10 | The canonical formatter already special-cases cons, with a comment naming this exact failure mode | `format/expr.go:154–165`, `format/format.go:496`, `format/pattern.go:127–150` | Confirmed (quoted in Problem Statement) |
| V11 | All cons round-trip failures flow through the **one** function `PrintAILANGSource`; no second hand-rolled expression printer exists in the harness | `grep PrintAILANGSource internal/` → 6 call sites, all in `internal/testing` (executor.go:245, named_batch.go:122+133, property_batch.go:68, bytecode_engine.go:112, test_body_lowering.go:161); `value_splice.go` splices *values*, not expressions | Confirmed |
| V12 | Cited fixtures exist | `ls internal/testing/testdata/named_batch/mixed.ail` (present, and its `ailang test` output pattern re-checked); `cmd/ailang/test_bytecode_test.go` read for the e2e pattern | Confirmed |
| V13 | No new error code is proposed (PAR_NO_PREFIX_PARSE is cited, not created) | — | n/a |
| V14 | **Negative-existence:** no in-repo example/testdata today puts `::` inside a `test "…" { }` body or forall property — the corpus gap that let this ship | `grep '::' examples/ std/ internal/testing/testdata/` → cons appears in function bodies and match patterns only (`archive/broken/testing_basic.ail:63` uses legacy `test "…" =` syntax, not run) | Confirmed |

## Non-Goals

**Not attempted in this feature:**

- **Parser prefix parselet for `::(a, b)` in expression position** — makes the
  desugared canonical form real syntax, would fix the round-trip generically,
  and would align the expression grammar with the pattern grammar (which
  accepts `::(h, t)`). But it *adds language surface* (docs, prompt, strict
  mode, fmt) — a language change, not a harness bug fix. Deferred to its own
  design if ever wanted.
- **The strict-mode inconsistency** — `reportSugarError("CONS", "x :: xs",
  "::(x, xs)")` (`parser_expr.go:480`) recommends a canonical form that does
  not parse (V8). Routed as a separate follow-up issue in Phase 3; touching
  it here would couple a language-surface decision to a harness fix.
- **Re-sugaring interpolated strings in the harness printer** (the formatter
  does; the harness prints the desugared calls, which re-parse fine — V5).
  No bug, no change.
- **Non-final non-assert expression short-circuiting** (#604, already tracked
  in `test_body_lowering.go`).
- **Sharing code with `internal/format`** — the formatter is a full-file
  pretty-printer with precedence handling; extracting an expression-level API
  from it for 10 LOC here is an architectural change out of proportion.

## Timeline

**Day 1** (~6h + 2h buffer):
- Phase 1: printer case + tripwire test (2h)
- Phase 2: fixtures, property case, e2e incl. bytecode parity (3h)
- Phase 3: CHANGELOG, downstream notice, strict-mode issue routing (1h)

**Total: ~1 day.** (Estimate is 1.5× the phase sums, per the 2× realism rule
applied to a well-scoped single-function change.)

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Printed infix `(a :: b)` changes re-parse precedence vs the original desugared node | High | Unconditional parentheses (Design Freeze); tripwire asserts the re-parse desugars back to the *identical* node (V3 path), not merely "parses" |
| A `::` call with ≠2 args gets silently reshaped | Med | Shape check is exact (`len(Args)==2`); other shapes fall through to today's generic print (Design Freeze) |
| Fix masks a future parser desugar regression (next desugar without a printer case) | Med | Tripwire pattern established (assert did this in `named_test_assert_test.go:317`); Phase 2 fixture exercises every desugar family in one file |
| Property path fix forgotten (different call site) | Med | Property fixture is an explicit Phase 2 deliverable; V4 pins the current failure |
| Downstream restore (stapledons-godot) conflicts with other sim changes | Low | Their comment cites this report; restore is a one-line revert per site, after this ships in a tagged release |

## Related Documents

**Implemented (may inform design):**
- [m-namedtest-strip-string-blind-braces](../implemented/v0_54_13/m-namedtest-strip-string-blind-braces.md) — #1821, the same round-trip class (strip step), fixed v0.54.13
- [m-test-runner-compile-once](../planned/v0_53_0/m-test-runner-compile-once.md) — the shared batch compile this bug's failure notice comes from (Phases 1–2 implemented 2026-10-06)

**Planned (check for overlap):**
- None found — related-doc search (SimHash + neural) on "named test cons roundtrip" returned no matches in either direction.

**Code-level prior art (same class):**
- #590 assert round-trip — fixed by lowering asserts in `FoldTestBody` (`internal/testing/test_body_lowering.go`), tripwire at `executor_helpers.go:430` (`AssertStmt` printer case) and `named_test_assert_test.go:317`
- Formatter cons case: `internal/format/expr.go:154` — the exact precedent this design mirrors, including its "must be re-emitted in infix form to round-trip" comment

## References

- [Design Axioms](/docs/references/axioms) — the 12 non-negotiable principles
- Report: stapledon mission loop iteration 42 (R2-SIMILAR-S0 S0.1), 2026-10-20; versions v0.54.14 (e1ef33f), v0.54.16-dirty (128371f)
- Class precedents: #590 (assert round-trip), #1821 (strip string-blind braces)

## Future Work

- Strict-mode `::(x, xs)` canonical-form inconsistency (routed in Phase 3).
- A corpus guard: a fixture matrix asserting every documented desugar (cons,
  CALL0, interpolation, assert) round-trips through the harness printer —
  the #1821/#590/this sequence suggests the next one will otherwise be found
  by a downstream consumer again.

---

**Document created**: 2026-10-20
**Last updated**: 2026-10-20