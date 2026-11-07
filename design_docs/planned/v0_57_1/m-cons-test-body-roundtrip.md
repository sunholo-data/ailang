# m-cons-test-body-roundtrip — the desugared cons form `::(x, xs)` is documented but unparseable, so cons (`::`) breaks every named-test and property body round-trip

**Status**: Planned
**Target**: v0.57.1
**Priority**: P0 (High)
**Estimated**: 1–2 days
**Dependencies**: None

**Source**: bug report via `ailang messages` from stapledons_godot, stapledon mission loop iteration 42 (R2-SIMILAR-S0 S0.1), 2026-10-20. Reproduced on v0.54.14 (pinned release, e1ef33f), v0.54.16-dirty (128371f), and — by this session — on the current tree v0.57.0 (b23cadc). Class-sibling of #590 (assert had no prefix parselet and broke the same round-trip) and the float-literal / brace-stripper round-trip defects triaged in `design_docs/planned/ailang-core-triage/ai-deadline-and-named-test-roundtrip.md`.

## Problem Statement

A named test body or property body that uses the cons operator `::` fails to compile with `PAR_NO_PREFIX_PARSE … unexpected token in expression: ::`, even though the identical expression type-checks (`ailang check` rc 0) and runs in function bodies (`ailang run` rc 0). Downstream, the stapledon mission loop's `sim/similar_test.ail` works around it by writing the cons-free, asymptotically worse `[x] ++ xs` in test bodies, citing this report.

Minimal repro (`r.ail`, from the report):

```ailang
module r
import std/list (length)
pure func identity(xs: [int]) -> [int] = xs
export pure func consRun(n: int) -> int { let x = 1; let xs = [2]; length(identity(x :: xs)) }
test "cons argument" { let x = 1; let xs = [2]; length(identity(x :: xs)) == 2 }
```

- `ailang run --quiet --entry consRun --args-json 0 r.ail` → rc 0, prints `2`.
- `ailang check r.ail` → rc 0, "✓ No errors found!".
- `ailang test r.ail` → rc 1: `named tests … could not share one compile (… parse errors …); compiled each test separately`, then the test itself fails with `PAR_NO_PREFIX_PARSE at r.ail:5 (test body): unexpected token in expression: ::` plus the classic delimiter cascade. **All three engine paths fail identically** — evaluator batch, per-body VM fallback, and `--strict-bytecode` (V1, V2).

**Current state (root-cause chain, each link verified in code):**

1. `parseConsExpression` (`internal/parser/parser_expr.go:473–501`) desugars the infix sugar `x :: xs` into `&ast.FuncCall{Func: &ast.Identifier{Name: "::"}, Args: [x, xs]}`.
2. The named-test/property harness does not evaluate the parsed AST directly. It folds each body (`FoldTestBody`, `internal/testing/test_body_lowering.go`), prints it back to AILANG source with `PrintAILANGSource`, wraps it as `pure func __namedtest_N() { … }`, and re-compiles that generated module through the general pipeline. Consumer sites: batched entries (`internal/testing/named_batch.go:123`, `:134`), the per-body path (`internal/testing/executor.go:245`), the per-body VM path (`internal/testing/bytecode_engine.go:112`), forall properties (`internal/testing/property_batch.go:68`), and assert-lowering check messages (`internal/testing/test_body_lowering.go:161`).
3. `PrintAILANGSource`'s generic `*ast.FuncCall` case (`internal/testing/executor_helpers.go:320–335`) prints any call as `f(args)` — so cons prints as **`::(x, xs)`**.
4. `::` has an **infix** parselet only (`p.registerInfix(lexer.DCOLON, p.parseConsExpression)`, `internal/parser/parser.go:128`). There is **no prefix parselet** for `lexer.DCOLON` (V6). So `::(x, xs)` is a guaranteed `PAR_NO_PREFIX_PARSE`: **the desugared form of S-CONS is not valid AILANG source**, and the harness round-trip cannot parse its own output.

**Two further facets of the same asymmetry (found by this session's audit, both verified):**

- **Strict mode promises the unparseable form.** `ailang check --strict-syntax` on `x :: xs` reports `SUGAR_CONS … Suggestion: Use ::(x, xs) (canonical syntax) instead of x :: xs` (`internal/parser/parser_expr.go:480`) — but `::(x, xs)` does not parse (V3), and there is no alternative: no `cons` builtin exists (V7), and `[x] ++ xs` is different semantics (O(n) copy vs O(1) prepend). Strict mode therefore has **no working cons at all**.
- **The teaching prompt documents it.** `docs/docs/prompts/current.md:440` lists List cons as `x :: xs` or `::(x, xs)` — every prompt-derived agent is entitled to write the second form and get a parse error (V4).

**Precedent:** `ailang fmt` already solved exactly this: `internal/format/expr.go:154–163` recognizes `FuncCall{Func: Identifier "::", Args: 2}` and re-emits infix, with a comment stating "`::` is an infix-only operator: the parser has no prefix/call form for it, so `::(a, b)` does NOT re-parse. It must be re-emitted in infix form to round-trip." The formatter prints cons correctly today; only the harness printer and the grammar's missing prefix form are broken.

**Impact:** every AI-generated or human test that conses in a test or property body fails with a misleading parser error far from the cause (the same "100% of the time, far from the actual cause" failure mode as #590). The stapledon mission loop already carries a workaround with a comment. Strict-mode users are advised to write syntax that cannot parse.

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No evaluation-semantics change; the prefix form desugars to the identical AST the infix form produces |
| A2: Replayability | 0 | No trace/replay changes |
| A3: Effect Legibility | 0 | Cons is pure; no effect surface touched |
| A4: Explicit Authority | 0 | No capability changes |
| A5: Bounded Verification | +1 | Documented-but-unparseable syntax is a false contract; making it parse restores local checkability of what `check --strict-syntax` and the prompt already promise |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | The broken path is the AI test-authoring path (named tests, properties, mission loop); fixing it removes a misleading error that AI consumers hit deterministically |
| A8: Minimal Syntax | +1 | Adds no new syntax surface — implements a form already documented in the teaching prompt and demanded by strict mode; today's surface *includes* the false promise, which is worse |
| A9: Cost Visibility | 0 | No cost semantics change (the `[x] ++ xs` workaround's hidden O(n) cost stops being *forced* on test authors) |
| A10: Composability | +1 | Round-trip (source → AST → source) becomes closed for the desugared cons node, like it already is for every other node the harness prints |
| A11: Structured Failure | +1 | Replaces a cascading, position-mapped `PAR_NO_PREFIX_PARSE` far from the cause with correct parses, and makes strict-mode's advice true |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +5** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): no implicit nondeterminism introduced
- [x] A3 (Effects): no hidden side effects
- [x] A4 (Authority): no ambient access granted
- [x] A7 (Machines First): fixes the machine test-authoring path; no human-convenience sugar added

## Goals

**Primary Goal:** make the desugared cons form `::(x, xs)` valid AILANG source (as documented and as strict mode demands), so that every harness round-trip path — batched named tests, per-body named tests, VM bodies, forall properties — compiles a body containing `::`, and `x :: xs` survives the round-trip verbatim in failure messages.

**Success Metrics:**

1. The report's `r.ail` passes `ailang test` (rc 0, 1 passed) — with **no** "could not share one compile" notice, i.e. the batched path succeeds — on the evaluator, on `--bytecode` (VM runs the body, zero fallbacks), and on `--strict-bytecode` (engine parity).
2. A forall property whose body contains `::` passes (`ailang test` rc 0).
3. `ailang check --strict-syntax` accepts a module that conses via `::(x, xs)` — the SUGAR_CONS suggestion becomes true.
4. A failing cons check reports the user's own spelling (`x :: xs`), not `::(x, xs)`.
5. Downstream: stapledons-godot `sim/similar_test.ail` can restore `x :: xs` in test bodies and drop the workaround comment.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| D1: Fix at the grammar root — add a prefix parselet for `::(head, tail)` (making the documented/strict-mode-promised form real), rather than only re-sugaring in the harness printer | Decides whether the language's documented surface is true or the docs/strict-mode message are edited back | human (approves this doc) | design | med (parser, but zero existing-valid-program impact — see Conflict Surface) |
| D2: ALSO re-sugar cons in `PrintAILANGSource` (mirror `internal/format/expr.go`) so harness-printed bodies and check-failure messages show the user's own infix spelling | Fidelity of failure messages + parity with the `ailang fmt` printer's established convention; keeps the harness robust even if other printers appear | human (approves this doc) | design | low (one printer case) |
| D3: The prefix form is canonical, not sugar — it must not set `sugarUsed` and must not be rejected in strict mode | Determines strict-mode semantics and REPL sugar feedback; wrong choice re-creates the bug in strict mode | compiler (semantics follow strict mode's own message) | design | low |
| D4: Arity of the prefix form is exactly 2 (`::(head, tail)`); `::(x)` / `::(x, y, z)` are clear `PAR_UNEXPECTED_TOKEN` errors, no new error code | Error surface and code-namespace hygiene (PAR codes are a shared namespace) | agent | design | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] D1 — prefix parselet included (this doc is the approval artifact; human ratifies by approving the sprint)
- [x] D2 — printer re-sugaring included
- [x] D3 — prefix form is canonical (no sugarUsed, strict-legal): dictated by the existing SUGAR_CONS message and the teaching prompt, both of which name `::(x, xs)` as canonical syntax
- [x] D4 — no new PAR code; reuse `PAR_UNEXPECTED_TOKEN` (namespace verified free of a suitable existing arity code — V8)

## Solution Design

### Overview

Close the round-trip asymmetry at both ends, mirroring the fix `ailang fmt` already chose on its side:

1. **Parser (root cause):** register a prefix parselet for `lexer.DCOLON` that parses the constructor-call form `::(head, tail)` and desugars it to the *identical* `FuncCall{Identifier "::", [head, tail]}` the infix form produces. This makes the harness printer's output re-parseable, honors the strict-mode SUGAR_CONS suggestion as written, and makes `docs/docs/prompts/current.md`'s "List cons" row true.
2. **Harness printer (fidelity + defense-in-depth):** add a cons case to `PrintAILANGSource`, ahead of the generic `*ast.FuncCall` case, that re-sugars `FuncCall{::, [a, b]}` to fully parenthesized `(a :: b)` — exactly the convention `PrintAILANGSource` already uses for `*ast.BinaryOp`, and exactly what `internal/format/expr.go:154–163` already does for `ailang fmt`. Failure messages and generated test sources then show the user's own spelling.
3. **No message or docs change needed:** with (1), the strict-mode suggestion and the teaching prompt become true as written. `ailang fmt` continues to normalize `::(x, xs)` to infix (its existing cons case already prints the desugared node infix; the AST is unchanged, so fmt behavior is unchanged).

### Architecture

**Components:**

1. **`parseConsPrefix` (new, `internal/parser/parser_expr.go`, next to `parseConsExpression`)**: fired when `lexer.DCOLON` appears in expression-prefix position. Expects `(`, parses `head` at `LOWEST`, expects `,`, parses `tail` at `LOWEST`, expects `)`, returns the same desugared `*ast.FuncCall` as `parseConsExpression`. Does **not** touch `p.sugarUsed` (D3: canonical form). Arity errors report `PAR_UNEXPECTED_TOKEN` with a message naming the `::(head, tail)` shape (D4).
2. **Registration**: one line in `internal/parser/parser.go` next to line 128: `p.registerPrefix(lexer.DCOLON, p.parseConsPrefix)`.
3. **`PrintAILANGSource` cons case (`internal/testing/executor_helpers.go`)**: inside the `*ast.FuncCall` case, before generic printing — if `Func` is `&ast.Identifier{Name: "::"}` and `len(Args) == 2`, return `"(" + PrintAILANGSource(Args[0]) + " :: " + PrintAILANGSource(Args[1]) + ")"`. Fully parenthesized, so precedence/right-associativity need no reasoning at this site (consistent with the printer's BinaryOp style). A `::` call with any other arity (the strict-mode error-recovery placeholder at `parser_expr.go:483` builds a 1-arg node) falls through to the generic case, which emits `::(...)` — acceptable because the strict-mode compile already failed at parse time; alternatively the implementer may emit a `<printer-error: …>` string for non-2 arity, their choice (see Deferred Decisions).
4. **No elaborator / typechecker / VM changes**: both spellings produce the same AST node, so everything downstream of the parser is untouched. This is why engine parity comes for free.

### Implementation Plan

**Phase 1: parser prefix form** (~3 hours)
- [ ] Add `parseConsPrefix` next to `parseConsExpression` in `internal/parser/parser_expr.go` with the S-CONS-style doc comment
- [ ] Register it: `p.registerPrefix(lexer.DCOLON, p.parseConsPrefix)` in `internal/parser/parser.go`
- [ ] Unit tests (extend `internal/parser/sugar_test.go` or a new `cons_prefix_test.go`):
  - `::(1, [2])` parses to `FuncCall{Func: Identifier "::", Args: 2}` — same shape asserts as the existing S-CONS tests at `sugar_test.go:170–193`
  - `::(1, [2])` under strict mode: parses, **no** SUGAR_CONS, no `sugarUsed`
  - `x :: xs` still sets `sugarUsed` and still errors under strict mode (unchanged)
  - right-associativity of infix cons unchanged (`1 :: 2 :: []`)
  - `::(1)` and `::(1, 2, 3)` → `PAR_UNEXPECTED_TOKEN` naming the shape
  - `::` alone / `:: x` → `PAR_UNEXPECTED_TOKEN`, not a panic
- [ ] `ailang check` the report's `r.ail` plus `::(x, xs)` spellings of every cons example

**Phase 2: harness printer re-sugaring** (~3 hours)
- [ ] Add the cons case to `PrintAILANGSource` (`internal/testing/executor_helpers.go`; file is 717 lines against the 800 `make check-file-sizes` cap — the ~8-line case fits)
- [ ] Unit tests in `internal/testing` (pattern of `named_test_assert_test.go`, which already pins this printer):
  - `PrintAILANGSource` of the desugared cons node yields `(a :: b)`, which re-parses to the same AST
  - the report's `r.ail` body folds+prints to source containing `1 :: xs`, not `::(1, xs)`
- [ ] Property path: forall body containing `::` prints re-parseably

**Phase 3: end-to-end and regression** (~3 hours)
- [ ] Integration test in `internal/testing` (or `cmd/ailang` test): the report's `r.ail` file, asserting rc 0 and **no** "could not share one compile" notice, on the evaluator; the same fixture under `--bytecode` (VM runs the body, 0 fallbacks) and `--strict-bytecode`
- [ ] Property fixture with `::` passes
- [ ] Failing-check message test: a cons check that fails reports the user's `x :: xs` spelling
- [ ] Full sweep: `make test-core`, `make fmt`, `make lint`, `make check-boundaries`, `make check-file-sizes`
- [ ] Regression sweep over cons fixtures (Conflict Surface §4) with `ailang check` + `ailang test` before/after: zero behavior change
- [ ] Update `docs/docs/reference/` only if it currently claims `::(x, xs)` is invalid anywhere (it does not — the prompt table is the claim, and it becomes true)

### Files to Modify/Create

**New files:**
- `internal/parser/cons_prefix_test.go` (~70 LOC) — or extend `sugar_test.go`; implementer's choice

**Modified files:**
- `internal/parser/parser_expr.go` — `parseConsPrefix` + doc comment (~30 LOC)
- `internal/parser/parser.go` — 1 registration line (~1 LOC)
- `internal/testing/executor_helpers.go` — cons case in `PrintAILANGSource` (~10 LOC)
- `internal/testing/named_test_assert_test.go` or a new `internal/testing/cons_roundtrip_test.go` (~60 LOC) — printer + end-to-end pins

## Verification Log

All claims verified live on v0.57.0 (b23cadc, current tree) unless noted. Repro transcripts preserved in this session.

| # | Claim | How verified | Result |
|---|-------|---------------|--------|
| V1 | The report's repro fails `ailang test` on the current tree, evaluator batch path | `ailang test r.ail` | rc 1; "could not share one compile … parse errors"; `PAR_NO_PREFIX_PARSE at r.ail:5 (test body): unexpected token in expression: ::` + cascade |
| V2 | All three engine paths fail | `ailang test r.ail`, `--bytecode`, `--strict-bytecode` | evaluator: 1 failed; `--bytecode`: "0 named-test bodies ran on the VM, 1 fell back"; `--strict-bytecode`: "1 failed under --strict-bytecode" |
| V3 | `::(x, xs)` — the desugared/canonical form — does not parse anywhere | `ailang check` on a module with `length(::(x, xs))` in a function body | `PAR_NO_PREFIX_PARSE … unexpected token in expression: ::` + the same cascade as V1 |
| V4 | The teaching prompt documents `::(x, xs)` as valid | `grep -n "List cons" docs/docs/prompts/current.md` | line 440: "`x :: xs` or `::(x, xs)`" |
| V5 | Strict mode's SUGAR_CONS suggestion names the unparseable form | `ailang check --strict-syntax` on `length(1 :: [2])` | `SUGAR_CONS … Suggestion: Use ::(x, xs) (canonical syntax) instead of x :: xs` |
| V6 | No prefix parselet for `::` exists; only infix | `grep -n "registerPrefix(lexer.DCOLON" internal/parser/*.go` → empty; `internal/parser/parser.go:128` registers `lexer.DCOLON` as infix only | Confirmed (negative-existence) |
| V7 | No `cons` builtin exists (no strict-mode alternative) | `grep -rn 'Name == "cons"\|"cons":\|Name: *"cons"' internal/builtins/ internal/runtime/ internal/eval/` → empty; `internal/builtins/list.go` registers the operator-named `::` builtin, tags only mention "cons" | Confirmed (negative-existence) |
| V8 | Parser error codes are a fixed 3-code set; no arity code to reuse, so D4 adds no new code | `grep PAR_ internal/parser/parser_error.go` | `PAR_NO_PREFIX_PARSE`, `PAR_RESERVED_KEYWORD`, `PAR_UNEXPECTED_TOKEN` only; the prefix-form arity guard reuses `PAR_UNEXPECTED_TOKEN` — no allocation needed |
| V9 | The parser desugars `x :: xs` to `FuncCall{Identifier "::", [x, xs]}` | `internal/parser/parser_expr.go:473–501`; `internal/parser/sugar_test.go:170–193` asserts exactly this | Confirmed (code + existing test) |
| V10 | The harness round-trips bodies through `PrintAILANGSource` → re-parse at all five consumer sites | `grep -rn "PrintAILANGSource" internal/ cmd/` (non-test) | `named_batch.go:123,:134`, `property_batch.go:68`, `executor.go:245`, `bytecode_engine.go:112`, `test_body_lowering.go:161` |
| V11 | `PrintAILANGSource`'s generic FuncCall case emits `::(x, xs)` | `internal/testing/executor_helpers.go:320–335` | Confirmed (code) |
| V12 | `ailang fmt` already re-sugars cons with an explicit "does NOT re-parse" comment | `internal/format/expr.go:154–163` | Confirmed (code) |
| V13 | Properties with cons fail identically (same class, not just named tests) | `ailang test` on `property "cons prop" { forall(n: int) => length([1] :: [2]) == 2 }` | rc 1; `PAR_NO_PREFIX_PARSE at …:3 (property): unexpected token in expression: ::` |
| V14 | Cons-adjacent forms already round-trip: cons PATTER in match, `++`, interpolation, unary minus | `ailang test` on four probe fixtures | match with `x :: xs` arm: pass; `xs ++ ys`: pass; `"v=${x}"`: pass; `(- x)`: pass — cons is the isolated gap |
| V15 | `BinaryOp{Op: "::"}` never occurs (infix DCOLON routes to `parseConsExpression`, not `parseInfixExpression`) | `internal/parser/parser.go:128`; grep for BinaryOp with `::` op | Confirmed (negative-existence) — the printer's cons case cannot collide with a BinaryOp spelling |
| V16 | Fixtures named in Conflict Surface §4 exist | `ls` on each | all present (see section) |

## Conflict Surface

Required: this design touches `internal/parser/` and `internal/testing/`.

**1. What syntactic positions does this change extend?**

Expression-prefix position only, for token `lexer.DCOLON`: today a guaranteed `PAR_NO_PREFIX_PARSE` (V3, V6). The new parselet additionally requires the exact shape `:: ( expr , expr )`.

**2. What other valid constructs already live in those positions?**

None. `::` today appears only as (a) infix operator (routes to `parseConsExpression`, unchanged), (b) cons pattern (`internal/parser/parser_pattern.go:46,:87` — `ConstructorPattern`, a different AST node and a different parse path, unchanged), (c) never as a prefix. `::` is an operator token, not `IDENT`, so no user-defined function named `::` can exist (V7); the only 2-arg `FuncCall{::}` producer is `parseConsExpression` (V9, V15), and the only 1-arg producer is strict mode's error-recovery placeholder (`parser_expr.go:483`), which surfaces only in compiles that already failed.

**3. How does the parser disambiguate?**

Prefix parselets fire only from `parseExpression`'s prefix dispatch — a position `::` can reach today only by error. The parselet then demands `LPAREN … COMMA … RPAREN`; any other next token is a `PAR_UNEXPECTED_TOKEN` naming the `::(head, tail)` shape. There is no lookahead conflict with types (`[int]` uses COLON only singly), patterns (separate parser), or the infix registration (Pratt parsing dispatches prefix/infix by position, not token, so the two `::` parselets cannot shadow each other). Zero currently-valid programs can change meaning: the change strictly converts hard errors into parses of a node that already exists.

**4. Which existing programs MUST still work? (fixtures — all verified present, V16)**

- `examples/first_non_repeat.ail`, `examples/record_cons_pattern.ail`, `examples/record_list_extraction.ail`, `examples/inline_tests_best_practice.ail` — cons-bearing example programs (`ailang check` + `ailang test` where applicable)
- `internal/parser/sugar_test.go` — S-CONS desugar assertions (`AssertIdentifier(t, funcCall.Func, "::")`)
- `internal/parser/list_cons_pattern_test.go` — cons pattern round-trip
- `internal/format/format_test.go:293–294` — the cons expression/pattern fmt contract ("The expression half was fixed…")
- The report's `r.ail` under `ailang run`, `ailang check`, and (post-fix) `ailang test` on all three engines
- `ailang fmt` idempotence over cons fixtures (AST unchanged ⇒ fmt output unchanged; `::(x, xs)`-spelled sources will normalize to infix, which is fmt's documented sugar-normalizing behavior, cf. `m-fmt-dialect-alignment`)

**5. What deliberately changes?**

- Source that previously failed with `PAR_NO_PREFIX_PARSE` (`::(x, xs)` in any expression position) now parses to the cons node. This is the intended fix, aligned with the documented surface (V4) and strict mode's own suggestion (V5).
- Strict mode accepts `::(x, xs)` (it is the canonical form strict mode demands); `x :: xs` remains SUGAR_CONS-rejected in strict mode, unchanged.
- Harness-generated entry sources and check-failure messages change spelling from `::(x, xs)` to `(x :: xs)`.

The honest conflict-surface answer is "effectively empty" — and unlike most parser changes, that is defensible here because the extended position is provably error-only today (V3, V6); every case where `::` is meaningful routes through a different parselet that this design does not touch.

## Examples

### Example 1: the report's repro

**Before:**
```
$ ailang test r.ail
→ named tests in r.ail: could not share one compile (… parse errors …); compiled each test separately
  ✗ cons argument
      PAR_NO_PREFIX_PARSE at r.ail:5 (test body): unexpected token in expression: ::
      …cascade…
```

**After:**
```
$ ailang test r.ail
  ✓ cons argument
1 tests: 1 passed, 0 failed, 0 skipped
```
…and identically under `--bytecode` (body runs on the VM) and `--strict-bytecode`.

### Example 2: strict mode gets a working cons

**Before:** `ailang check --strict-syntax` on `x :: xs` advises `::(x, xs)`; writing the advice yields `PAR_NO_PREFIX_PARSE`. No working cons exists in strict mode.

**After:** `pure func two() -> [int] = ::(1, [2])` type-checks under `--strict-syntax`; `1 :: [2]` still reports SUGAR_CONS there (unchanged).

### Example 3: failure messages show the user's spelling

**Before:** a failing cons check quotes back `::(x, xs)` — source the user never wrote (same complaint class as #590's assert quoting).

**After:** the check message quotes `(x :: xs)`.

## Success Criteria

- [ ] The report's `r.ail` passes `ailang test` on the evaluator, `--bytecode`, and `--strict-bytecode`, with no "could not share one compile" notice (acceptance: new integration fixture)
- [ ] A forall property body containing `::` passes (acceptance: fixture, was rc 1 before — V13)
- [ ] `ailang check --strict-syntax` accepts `::(x, xs)`; `x :: xs` still rejected there (acceptance: parser unit tests)
- [ ] `::(x)` / `::(1, 2, 3)` / bare `::` produce `PAR_UNEXPECTED_TOKEN` naming the shape, no panics (acceptance: parser unit tests)
- [ ] `PrintAILANGSource` round-trip: desugared cons prints as `(a :: b)` and re-parses to the identical AST (acceptance: printer unit test)
- [ ] Engine parity: evaluator and strict VM both pass every new fixture (per the structure's Success Criteria rule, generated Go is not required)
- [ ] All fixtures in Conflict Surface §4 unchanged in behavior; `make test-core`, `make fmt`, `make lint`, `make check-boundaries`, `make check-file-sizes` all pass
- [ ] All tests passing
- [ ] Docs: no reference doc claims `::(x, xs)` invalid; prompt table (current.md:440) becomes true — no edit needed there; changelog entry on release

## Testing Strategy

**Unit tests:**
- Parser: prefix-form desugar equality with infix (AST shape asserts identical to `sugar_test.go`), strict-mode acceptance, arity/malformed errors, sugar-used asymmetry (infix sets it, prefix doesn't)
- Printer: cons re-sugaring + re-parse round-trip; non-2 arity `::` nodes don't produce unparseable output silently

**Integration tests:**
- The report's `r.ail` as a fixture, across all three engines
- Property fixture with cons (V13's file)
- A deliberately failing cons check asserts the message shows `x :: xs`

**Manual testing:**
- `ailang test` on a copy of stapledons-godot `sim/similar_test.ail` with the workaround reverted (downstream confirmation), if the repo is reachable; else the r.ail fixture stands in

## Deferred Decisions

- Whether `PrintAILANGSource` should emit a self-describing `<printer-error: …>` string for a `::` call with arity ≠ 2 (the strict-mode placeholder) instead of the generic `::(...)` output — **agent may choose**; both are unreachable in passing compiles (V6 context), so this is a tripwire-style nicety in the spirit of the existing `*ast.AssertStmt` printer tripwire.
- Whether the new parser tests live in `sugar_test.go` or a new `cons_prefix_test.go` — **agent may choose**.
- Whether to also run the optional `ailang design-quorum` pass — **human may request**; the doc is self-contained without it.

## Non-Goals

- **No change to cons semantics, evaluation, or cost** — `[x] ++ xs` vs `x :: xs` performance characteristics are untouched (see `docs/docs/reference/limitations.md` §quadratic construction; that stays parked).
- **No new strict-mode ban or un-ban of any other sugar** — S-CALL0's `f (())` suggestion is parseable today (its canonical form works, unlike CONS's); auditing every S-* sugar's canonical form beyond cons is not needed for this fix and is not attempted (the audit that matters here — the harness printer's round-trip surface — was done: V14).
- **No unification of `PrintAILANGSource` with `internal/format`** — the two printers are deliberately separate (the executor's printer must not render `assert` verbatim; see the tripwire at `executor_helpers.go` and `named_test_assert_test.go:314–315`). Replacing one with the other is a larger refactor with its own failure modes; this fix keeps them aligned on cons via the same convention.
- **No property-runner rework** — properties fail only through the same printer; fixing the printer fixes them (V13).
- **No changelog/VERSION bump in this doc** — release mechanics belong to the sprint/release process.

## Timeline

**Week 1** (~9 hours):
- Phase 1: parser prefix form + unit tests (3h)
- Phase 2: printer re-sugaring + unit tests (3h)
- Phase 3: end-to-end fixtures, regression sweep, lint/boundaries/file-sizes (3h)

**Total: ~9 hours across 1 week** (2× the naive estimate of ~4.5h; the audit and verification above are already done, which is where such estimates usually blow up).

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-------------|
| A prefix parselet on DCOLON shadows or perturbs an existing `::` position (pattern, infix, type) | Med | Pratt dispatch is positional: prefix parselets only fire where today is a guaranteed error (V3, V6); Conflict Surface §3 enumerates the other `::` paths, all untouched; parser fuzz corpus (`fuzz_test.go`) and full example sweep gate the merge |
| The harness printer's `(a :: b)` output interacts badly with some downstream construct (e.g. a `startsWithStatementStarter`-style concern like format's) | Low | Full parenthesization makes the emitted form precedence-proof — the same guarantee the printer's BinaryOp case relies on; end-to-end fixtures cover evaluator and VM |
| Hidden consumers of `::`-shaped FuncCall other than the ones enumerated (V10) appear | Low | `grep PrintAILANGSource` is exhaustive over non-test consumers (V10); the parser-side fix additionally makes ANY printer's `::(a, b)` output valid, so the class is closed at the root, not per-printer |
| Someone reads the prefix form as new sugar and starts rejecting it in strict mode | Med | D3 is frozen in Design Freeze with the strict-mode acceptance test as its guard |
| The fmt dialect alignment tests consider `::(x, xs)` → `x :: xs` normalization a change | Low | `internal/format` already prints the desugared node infix (V12); only source text written in prefix spelling changes under fmt, which is fmt's documented normalizing role |

## Related Documents

**Implemented (informs the design):**
- `design_docs/implemented/v0_33_1/m-named-test-assert-lowering-sprint-plan.md` + `m-named-test-body-check-semantics.md` (#604) — the #590/#604 line of work that established `FoldTestBody`, the printer tripwire convention, and the check-sentinel lowering this fix builds on
- `design_docs/implemented/v0_54_13/m-namedtest-strip-string-blind-braces.md` — the strip-side member of the named-test round-trip bug family
- `design_docs/implemented/v0_32_0/m-fmt-dialect-alignment.md` — fmt's sugar-normalizing contract; the cons re-sugaring at `internal/format/expr.go:154` is the direct precedent for D2

**Planned (check for overlap):**
- `design_docs/planned/v0_53_0/m-test-runner-compile-once.md` — the batched-compile path whose failure notice is the visible symptom here; no overlap in scope (this fix un-breaks the batch, that doc built it)
- `design_docs/planned/ailang-core-triage/ai-deadline-and-named-test-roundtrip.md` — triage of the same round-trip family (float literals via the printer; string braces via the stripper); distinct defects, shared mechanism — this doc's D2 follows the same printer-fidelity direction that triage recommended for floats
- `design_docs/planned/ailang-core-triage/named-test-float-dict-resolution.md` — superseded-by-triage refinement of the float half; not cons

**Distinction note (duplicate gate):** the auto-search (SimHash + neural) found no ≥0.75 overlap; the related docs above cover adjacent members of the round-trip family, none covers cons or the missing `::` prefix parselet.

## References

- [Design Axioms](/docs/references/axioms)
- Bug report: stapledons_godot via `ailang messages`, 2026-10-20 (reproduced V1–V2); class-siblings #1821 / #590 as cited in that report
- `docs/docs/prompts/current.md` "List cons" row (line 440)
- `docs/docs/reference/limitations.md` §1 — quadratic `::`/`++` construction (cost context, explicitly untouched)

## Future Work

- **Quadratic list construction** (limitations.md §1) stays parked; once cons round-trips in tests, property-based regression suites for list cost could be written in AILANG itself.
- **A shared round-trip conformance corpus** for every parser desugar (S-CONS, S-CALL0, interpolation) asserting `parse(print(ast)) == ast` across both printers — this fix closes the cons member; a corpus test would prevent the next member of this family (floats are the known straggler per the triage doc).
- **Downstream:** revert the `[x] ++ xs` workaround + comment in stapledons-godot `sim/similar_test.ail` once v0.57.1 ships.

---

**Document created**: 2026-11-03
**Last updated**: 2026-11-03
