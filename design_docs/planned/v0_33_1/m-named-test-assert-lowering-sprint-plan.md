# Sprint Plan — `m-named-test-assert-lowering` (make `test "…" { assert … }` execute)

**Sprint ID**: `M-NAMED-TEST-ASSERT-LOWERING`
**Mission**: V1 outer loop, iteration 152
**Planner**: Opus (sprint-planner role)
**Planner-Lane**: opus
**Date**: 2026-08-06
**Base**: `dev` @ `64970c118` (`v0.33.0-39-g64970c118`), main checkout — executor works in a worktree
**GitHub issues**: `#590` (closes)
**Design doc**: **NONE — bug-fix lane** (judgement in §1)
**Risk**: low
**Estimated**: ~480 LOC (impl ~180, tests ~240, testdata/docs ~60), **~1.5–2 executor-days**

---

## 0. Verification posture

Every claim in §2–§5 was measured first-party in this checkout with
`/Users/voightkampff/go/bin/ailang`, whose `--version` (`v0.33.0-39-g64970c118-dirty`) matches
`git describe` at `64970c118`. The `-dirty` is the two rig-synced benchmark JSON files
(`docs/static/benchmarks/os/*.json`) — no Go source is modified.

**Every lowering shape in §4 was validated as real AILANG source with `ailang test` before being
written down.** They are not sketches; they parse, typecheck, elaborate and evaluate through the
exact round-trip pipeline this sprint touches. Commands are recorded inline.

**Commit posture**: the main checkout is shared and carries uncommitted benchmark JSON. Do not
commit, branch, or stash in it. Work in a worktree; the controller finalizes.

---

## 1. JUDGEMENT: no design doc, no quorum — bug-fix lane

**Recommendation: proceed straight to execution.**

1. **No new syntax, no new flag, no new user-visible semantics.** `assert` is already a reserved
   lexer keyword (`internal/lexer/token.go`), already has a parser production
   (`parseAssertStmt`, `internal/parser/parser_test_decl.go:139`), already has an AST node that
   implements `exprNode()` (`internal/ast/ast_decl.go:167-176`), and is already handled by the
   formatter (`internal/format/expr.go:562`). Every layer exists **except the one that turns it
   into something executable.** This sprint supplies that layer and nothing else.
2. **Precedent is direct**: `#548` / `m-strip-decl-aware` ran doc-less as a bug fix inside
   `internal/testing`; so did `#524` and `#541`. This is the same package and the same shape —
   in fact it is the *sibling* of `#548`: both are defects in the AST→source→pipeline round-trip
   that `EvaluateNamedTestBodyExprs` performs.
3. **The chosen fix (§3) touches exactly one package, `internal/testing`.** No parser change, no
   lexer change, no elaborator change, no Conflict Surface obligation. The two options that
   *would* have triggered one are rejected on measured grounds in §3.

---

## 2. Premises: 6 confirmed, 1 partially refuted, 3 new findings

The controller supplied 7 measured premises. Six reproduce exactly. One is literally true but
its stated *inference* overstates the evidence. Three findings are new.

### Confirmed

| # | Premise | My re-measurement |
|---|---------|-------------------|
| 1 | `test "n" { assert … }` → rc=1 `PAR_NO_PREFIX_PARSE … assert`; true and false assertions fail identically; assert-free control rc=0 | **Confirmed.** `ailang test` on 3 fresh modules: assert-true rc=1, assert-false rc=1, no-assert control rc=0 `1 tests: 1 passed`. Two-assert body also rc=1, failing at col 14 not col 3 — because `FoldBodyExprs` wraps the first assert in `let _seq = … in …`, so `{ let _seq = ` (13 chars) precedes it. The column shift is itself proof the fold, not the parser, chooses the shape. |
| 2 | `registerPrefix(lexer.ASSERT` = 0 of 26; `parseAssertStmt` reachable from one gated site | **Confirmed.** `rg -n "lexer.ASSERT" internal/` → 2 hits, both in `parser_test_decl.go` (:124 dispatch, :142 self-check). 26 `registerPrefix` calls total. |
| 3 | Mechanism is `EvaluateNamedTestBodyExprs` → `FoldBodyExprs` → `PrintAILANGSource` → temp file → general pipeline; printer emits `assert %s` at `executor_helpers.go:487-488` | **Confirmed by reading all four sites.** |
| 4 | `assert` in a plain function body also fails `PAR_NO_PREFIX_PARSE`; the keyword has no working call site in the language | **Confirmed.** `ailang check` on `export func main() -> () { assert 1 == 1 }` → rc=1, `PAR_NO_PREFIX_PARSE at …:2:28`. |
| 6 | `verify-examples` gates `examples/runnable/` only; `examples/experimental` appears nowhere in make's database | **Confirmed with the same control.** `make -pn \| grep -c examples/experimental` = **0**; `examples/runnable` = **9**. `make/examples.mk:18-28` is the gate. |
| 7 | `internal/testing` and `internal/format` both know `ast.AssertStmt`; it implements `exprNode()` | **Confirmed.** |

### Partially refuted

**P5 — literally true, inference overstated.** `ailang prompt --source=embedded` line 2269 does
list `assert`. But the surrounding block (lines 2258-2260) is headed:

> `## Reserved Keywords` — *"AILANG reserves 43 keywords. You **cannot** use these as variable or
> function names."*

So the prompt is **warning models off using `assert` as an identifier**, not advertising it as a
usable construct. `assert` appears exactly twice in the whole embedded prompt (line 2269, and
line 2323 in an unrelated package blurb) and is **never demonstrated in a worked example**. The
demand evidence is therefore *weaker* than "every eval model is told this keyword exists [and can
be used]" — it is "models are told the word is part of the language and then never shown what it
does." That is still real demand (models reach for keywords they are told exist), but it is
second-hand, and it changes a scope decision: see §6, M3.

Corroborating: the issue says `assert` is "the form shown in the language's own test
examples/docs". I could not confirm that — `rg` over `docs/docs/` found no `test "…" { assert … }`
worked example. The only in-repo uses are the two ungated `examples/experimental` files and
`internal/testing`'s own Go-string fixtures.

### New findings (mine, iteration 152)

**F1 — The elaborator has ZERO `AssertStmt` handling.** `rg "AssertStmt" -g '*.go'` over the whole
repo returns 8 files: `ast`, `parser`, `format`, `testing`, and their tests. **`internal/elaborate`
is not among them**, and `internal/elaborate/expressions.go:116-121` ends its `normalize` switch
with `return nil, fmt.Errorf("normalization not implemented for %T", expr)`.

This is decisive for §3: it means options (a) and (b) as the reporter phrased them **do not fix the
bug on their own** — they relocate the failure from the parser to the elaborator.

**F2 — Non-final expressions in a named test body are silently discarded, so a false one passes.**
This is a *second, unfiled* vacuous-green bug in the same function:

```ailang
test "first expr false, last true" {
  add_one(1) == 99;    -- FALSE
  add_one(1) == 2      -- true
}
```
→ **`✓ ... All tests passed!` rc=0.**

Root cause: `FoldBodyExprs` (`executor_helpers.go:300-311`) wraps every non-final, non-`let`
expression as `let _seq = expr in rest` — the value is bound to a dead name and thrown away.
`runner.go:156-158` documents this as intentional ("evaluated for side-effects only … named test
bodies are pure by contract"), which is precisely why it is wrong: a *pure* body's non-final
expressions have no side effects, so discarding a `bool` can only ever hide a failure.

**This directly constrains the fix.** The obvious lowering — print `assert c` as plain `c` — would
make `test { assert false; assert true }` **PASS**. Any accepted design must short-circuit. §4 does.

**F3 — Numeric literals in a comparison mis-infer on the named-test path only.**
```
test "x" { (if true then 0 else 2) == 0 }   →  type error: No instance for Num[bool]
export pure func f() -> bool { (if true then 0 else 2) == 0 }   →  ailang check: ✓ No errors found
```
Byte-identical expression; typechecks in a function body, fails as a named-test body. This is a
third bug, unrelated to `assert`, surfaced while validating §4's shapes. **Not in scope** — it does
not block this sprint, because the chosen design never emits an in-AILANG comparison of the
sentinel (the executor reads the `IntValue` in Go). Recommend the controller file it separately.

---

## 3. THE DESIGN DECISION — option (c), as an assert-aware *fold*, not a printer hack

**Chosen: (c) — lower `AssertStmt` to general AILANG syntax before printing. Confined to
`internal/testing`. No parser, lexer, or elaborator change.**

### Why (b) is rejected

Option (b) — "register `assert` as a general-expression prefix parselet" — **does not fix the bug.**
By F1, the elaborator has no `AssertStmt` case. Registering the parselet moves the failure from
`PAR_NO_PREFIX_PARSE` to `normalization not implemented for *ast.AssertStmt`. To actually work it
needs: parser (parselet + precedence), elaborate (new Core form or a desugaring), types (what type
does `assert e` have?), and eval (what does it *do* when false — there is no pure abort primitive;
`_debug_check` is an **effect** builtin requiring the `Debug` capability, `internal/builtins/debug.go:62-107`,
so it is unusable in a pure test body). That is **4 core packages plus a new pure-abort semantics**.

It is also a **language change, not a bug fix**: making `assert` legal in ordinary expression
position decides that AILANG has an assertion expression everywhere, with a defined value, a
defined type, and a defined failure behaviour — in a language whose north star is "everything is an
expression" and "all non-determinism explicit". Blast radius: every expression position in the
grammar; it needs a Conflict Surface, a design doc, and a quorum. **If the mission wants that, it is
a separate FEATURE item, not iteration 152's bug fix.**

### Why (a) is rejected

Option (a) — "evaluate the parsed AST directly instead of print-and-reparse" — also needs F1's
missing elaborator support, *plus* it must reproduce what the round-trip buys. The comment at
`executor.go:169-182` claims the round-trip exists so OpLowering and module-scope bindings work.
I measured what that means concretely: **`TestDecl` is never elaborated by the pipeline at all** —
`rg "TestDecl" internal/elaborate/ internal/pipeline/ internal/link/ internal/iface/` returns
**zero non-coincidental hits** (the one hit is a Go test named `TestDeclaredRows…`). Contrast the
contract path, which *does* get Core for free from `result.Artifacts.Core.Meta[fn].Contracts[i].Expr`
(`EvaluateEnsuresHarnessFromCore`, `executor.go:128`) precisely because contracts live on function
declarations the pipeline compiles.

So option (a) is not "delete the round-trip"; it is "build a new AST→lowered-Core-in-module-scope
entry point that does not exist today". That is a real piece of compiler plumbing, not a bug fix,
and it would still need the elaborator to learn `AssertStmt`. **Rejected on size, not on principle**
— it may well be the right long-term shape, and §9 records it as future work.

### Why (c), and specifically why in the *fold*

The reporter framed (c) as "have `PrintAILANGSource` lower `AssertStmt` when printing outside a
test-decl body". Doing it in the printer is **wrong**, for a measured reason: the printer sees one
node at a time and cannot see F2's sequencing problem. `assert c` printed as `c` in a non-final
position is *swallowed by the surrounding* `let _seq = … in …` and the test goes vacuously green.
Short-circuiting is a property of the **sequence**, so the lowering belongs where the sequence is
built: `FoldBodyExprs`.

Three further properties make this the small option:

- **`AssertStmt` can only occur at the top level of a test body.** Verified: `parseStatement`
  (`parser_test_decl.go:122-136`) dispatches on `ASSERT` only at statement start, and
  `test "x" { let y = assert true; y }` is a parse error (rc=1, reporter shows `✗ parse`). So the
  lowering is a **flat walk over `[]ast.Expr`**, not a recursive AST rewrite.
- **`FoldBodyExprs` has exactly one caller** (`executor.go:206`). `rg "FoldBodyExprs" -g '*.go'`
  → 4 hits: the definition, its doc comment, one mention in another comment, and the single call.
- **The printer already emits everything the lowering needs.** `case *ast.If`
  (`executor_helpers.go:411-415`) prints `if c then a else b`; literals and `let … in …` are
  already covered. **No printer change is required for the lowering itself.**

---

## 4. The lowering, validated as source

### Semantics (this is the part the controller asked to be pinned down)

Define the body's **checks**, left to right, 1-based:
- every top-level `*ast.AssertStmt` is a check;
- the **final** expression is a check (it already is one under today's bool contract — if it is an
  `AssertStmt` it is the same check, counted once).

The body lowers to an **`int` sentinel**:
- **`0`** — every check passed.
- **`k ≥ 1`** — check `k` was the **first** to evaluate false; checks after `k` are **not evaluated**.

**All sentinels are positive or zero** — deliberately, so the lowering never needs a unary-minus
literal in `else` position.

**Only bodies that contain at least one `AssertStmt` take the sentinel path.** An assert-free body
keeps today's exact bool path, byte for byte. This is what makes the regression risk ~zero: every
currently-passing named test is assert-free (they must be — assert bodies are 100% broken), so
none of them change code path.

`EvaluateNamedTestBodyExprs` decodes in Go and preserves the runner's contract:

| sentinel | returned to `runner.go:191-211` |
|---|---|
| `0` | `*eval.BoolValue{true}` → `StatusPass` |
| `k ≥ 1` | `error`: `` assertion 1 failed: `assert add_one(1) == 2` (at a.ail:6:3) `` → `StatusFail` with that text |

So a failing assert reports **which** assertion failed, with its source text and its original
position — not a bare `false`. (`ast.AssertStmt.Pos` is populated at
`parser_test_decl.go:159`; the condition's source text comes from `PrintAILANGSource(e.Condition)`.)

### Transformation

Walk `exprs` right-to-left, exactly as today, with two changed cases:

| element | today | after |
|---|---|---|
| `*ast.Let` / `*ast.LetRec` (non-final) | thread `Body: result` | **unchanged** |
| `*ast.AssertStmt` (non-final) | `let _seq = <AssertStmt> in result` | `&ast.If{Cond: e.Condition, Then: result, Else: IntLit(k)}` |
| `*ast.AssertStmt` (final) | `<AssertStmt>` | `&ast.If{Cond: e.Condition, Then: IntLit(0), Else: IntLit(k)}` |
| other (non-final) | `let _seq = expr in result` | **unchanged** (see F2 disposition, §6) |
| other (final), assert-bearing body | itself | `&ast.If{Cond: expr, Then: IntLit(0), Else: IntLit(k)}` |

`IntLit` is `int64` — `internal/ast` `IntLit` takes `int64`, not `int` (`.claude/rules/parser.md`:
"IntLit is `int64`, not `int` (will panic!)").

### Validated shapes — every one of these was run

| shape | source fed to `ailang test` | result |
|---|---|---|
| single assert | `{ if add_one(1) == 2 then 0 else 1 }` | evaluates, `*eval.IntValue` |
| two asserts | `{ if add_one(1) == 2 then (if add_one(3) == 4 then 0 else 2) else 1 }` | evaluates, `*eval.IntValue` |
| let then assert | `{ let x = add_one(1) in if x == 2 then 0 else 1 }` | evaluates, `*eval.IntValue` |
| assert, let, assert | `{ if add_one(1) == 2 then (let x = 1 in if x == 1 then 0 else 2) else 1 }` | evaluates, `*eval.IntValue` |
| plain non-final + sentinel final | `{ let _seq = (1 == 1) in if add_one(1) == 2 then 0 else 1 }` | evaluates, `*eval.IntValue` |

**Sentinel values confirmed exactly**, by wrapping in module-level `isZero`/`isOne`/`isTwo` helpers
so the result becomes a bool the runner can pass on:
- all checks pass → `isZero(…)` → **`✓ 1 passed`** (sentinel is `0`)
- first check false → `isOne(…)` → **`✓ passed`** (sentinel is `1`)
- second check false → `isTwo(…)` → **`✓ passed`** (sentinel is `2`)
  (`2 passed, 0 failed` for the latter two in one module.)

**Dangling-else checked, and it is safe.** `PrintAILANGSource` emits `if` **without parentheses**.
I ran the unparenthesized nested form — `isZero(if c1 then if c2 then 0 else 2 else 1)` and both
failing variants — and got **`3 passed, 0 failed`**. Nested `if` in `then` position associates
correctly. **No printer paren change is needed.** (Recorded because this is exactly the class of
thing that mis-associates silently.)

### File-size constraint (measured, and it dictates the file layout)

`internal/testing/executor_helpers.go` is **717 lines**; `make check-file-sizes` fails at **>800**
and is **green at base**. Adding ~180 LOC of lowering there would breach the gate. **M1 therefore
creates a new file** `internal/testing/test_body_lowering.go` and moves `FoldBodyExprs` into it —
the same move `#548`/`M-STRIP-DECL-AWARE` made when it created `internal/testing/source_strip.go`.

---

## 5. Baselines — every acceptance command, run on the pristine tree first

Mission rule 3e. `RED` = fails today (so passing it proves the fix); `GREEN` = passes today (so it
is a *stays-green* guard and is only non-vacuous alongside a RED criterion).

| # | Command | Base result |
|---|---------|-------------|
| B1 | `ailang test <single-assert module>` | **rc=1** `PAR_NO_PREFIX_PARSE … unexpected token in expression: assert` — **RED** |
| B2 | `ailang test <two-assert module>` | **rc=1**, same error at col 14 — **RED** |
| B3 | `ailang test <assert-free control module>` | **rc=0**, `1 tests: 1 passed` — **GREEN (must stay)** |
| B4 | `ailang test <non-final false, no assert>` | **rc=0, PASSES** — the F2 vacuous green |
| B5 | `go test ./internal/testing/...` | **rc=0** `ok … 1.089s` — **GREEN (must stay)** |
| B6 | `go test ./internal/parser/... ./internal/format/... ./internal/ast/...` | **rc=0** — **GREEN (must stay; proves the parser was not touched)** |
| B7 | `go build ./internal/... ./cmd/ailang` | **rc=0** — usable gate |
| B8 | `go build ./...` | **rc=1** at base (`cmd/wasm`, `gen/main` have no native `main`) — **NOT A GATE, do not use** |
| B9 | `go vet ./internal/testing/...` | **rc=0** — **GREEN (must stay)** |
| B10 | `make check-file-sizes` | **rc=0**, `✓ All files within 800 line limit` — **GREEN (must stay)** |
| B11 | `go test ./...` | **rc=1** at base (`internal/smt TestSolve_HardTimeout_FakeSolverIgnoringT` under full-suite load, `#602`) — **NOT A GATE, do not use** |

**Note on `go test -run`:** a `-run` pattern that matches nothing **exits 0**. So "run the new test"
is vacuous until the test exists. AC-1 below is therefore worded as a TDD obligation with a
recorded red output, not as an exit code.

---

## 6. Milestones

### M1 — Assert-aware test-body lowering (the fix)
**Size**: impl ~130 LOC, tests ~130 LOC · **~0.75 day** · depends on: nothing

- Create `internal/testing/test_body_lowering.go`; **move** `FoldBodyExprs` there from
  `executor_helpers.go` (file-size constraint, §4) with its doc comment.
- Add `FoldTestBody(exprs []ast.Expr) (ast.Expr, []CheckInfo)` implementing §4. `CheckInfo` carries
  `{Ordinal int, Source string, Pos ast.Pos}` for each check, in order.
- When `exprs` contains **no** `*ast.AssertStmt`, `FoldTestBody` returns exactly what
  `FoldBodyExprs` returns today and an empty `[]CheckInfo` — the byte-identical legacy path.
- `EvaluateNamedTestBodyExprs` (`executor.go:184`) calls `FoldTestBody`, and when
  `len(checks) > 0` decodes the resulting `*eval.IntValue` per the §4 table.

**Acceptance (each a command; RED/GREEN per §5):**
1. **AC-1 (TDD, records the red)**: a new `internal/testing/named_test_assert_test.go` exists with
   `TestNamedTest_Assert_Passes` built on the existing `runInlineTestsOnSource` helper
   (`executor_regression_test.go:14`). The executor **must** commit/record the run of
   `go test ./internal/testing/ -run TestNamedTest_Assert` **before** the fix and paste its
   failing output into the sprint JSON `notes`. Vacuous otherwise (see §5 note).
2. **AC-2**: `ailang test` on a single-assert module → **rc=0**, output contains `1 passed`.
   *(B1: rc=1 at base.)*
3. **AC-3**: `ailang test` on a module whose **only** assert is false → **rc=1**, and the output
   does **not** contain `PAR_NO_PREFIX_PARSE`. *(Both halves red at base: base rc=1 is for the
   wrong reason and the string IS present.)*
4. **AC-4 (short-circuit / anti-F2)**: `test "x" { assert <false>; assert <true> }` → **rc=1**.
   Paired with AC-3's no-`PAR_NO_PREFIX_PARSE` clause so the rc is not inherited from the parse
   error. *(B2 red at base.)*
5. **AC-5**: `go test ./internal/testing/...` → rc=0. *(B5 green; stays green.)*
6. **AC-6**: `make check-file-sizes` → rc=0. *(B10 green; this is why the new file exists.)*

### M2 — Which check failed, and a tripwire that goes red on reintroduction
**Size**: impl ~50 LOC, tests ~110 LOC · **~0.5 day** · depends on: M1

- Executor maps sentinel `k` to `fmt.Errorf("assertion %d failed: `assert %s` (at %s)", …)` using
  `CheckInfo`. Runner needs **no change** — `runner.go:191-196` already surfaces
  `err.Error()` as `result.Error`.
- **Tripwire**: change `PrintAILANGSource`'s `case *ast.AssertStmt`
  (`executor_helpers.go:487-488`) from emitting `assert %s` to returning
  `<printer-error: AssertStmt reached the printer; FoldTestBody must lower it first>`. This matches
  the file's existing convention for unrepresentable nodes (`*ast.Error` at :486 and `default:` at
  :501 already return `<printer-error: …>` strings). Reaching the printer with an `AssertStmt` is
  now a *self-describing* failure instead of a `PAR_NO_PREFIX_PARSE` 200 lines away.

**Acceptance:**
7. **AC-7**: `go test ./internal/testing/ -run TestNamedTest_Assert_ReportsWhichAssertion` → rc=0,
   asserting `result.Error` matches `assertion 2 failed` for a body whose **second** assert is
   false. Red before M2 (the message would be `expected true, got false`).
8. **AC-8 (regression guard, both directions)**: a unit test asserting
   `PrintAILANGSource(&ast.AssertStmt{Condition: …})` contains `printer-error`. Goes **red** if
   someone restores the verbatim `assert %s` printing. Combined with AC-2/AC-4, which go red if
   someone reverts the fold, **#590 cannot be reintroduced without a red test.**
9. **AC-9**: `go test ./internal/parser/... ./internal/format/... ./internal/ast/...` → rc=0.
   *(B6 green; this is the "we did not touch the parser" proof. `internal/format`'s own
   `node_coverage_test.go:100` still expects `"assert x"` from the **formatter's** printer — a
   different printer — and must stay green.)*
10. **AC-10**: `go vet ./internal/testing/...` → rc=0 and `go build ./internal/... ./cmd/ailang`
    → rc=0. *(B9/B7 green. **Do not use `go build ./...`** — B8.)*

### M3 — Fixtures, CHANGELOG, and explicit dispositions
**Size**: ~40 LOC testdata + ~20 lines docs · **~0.25 day** · depends on: M1, M2

- Add `.ail` fixtures under `internal/testing/testdata/assert/` (alongside the existing
  `testdata/strip/` corpus) covering: single passing assert · single failing assert · two asserts
  with the second false · `let` + assert · assert + `let` + assert. Drive them from the Go tests.
- `CHANGELOG.md` entry under the unreleased heading, `Fixes #590`.

**Acceptance:**
11. **AC-11**: `ailang test` over each fixture produces the expected rc, driven by a table test;
    `go test ./internal/testing/...` rc=0.
12. **AC-12**: `rg -n "590" CHANGELOG.md` returns a hit.

**Explicit disposition — the two ungated `examples/experimental` files: LEAVE THEM. Do not convert
them to fixtures and do not gate them.** Grounds, measured:
- All three experimental files I checked fail `ailang check` **rc=1 with parse errors** *today, for
  reasons that have nothing to do with `assert`* — `web_api.ail` and `concurrent_pipeline.ail` use
  `spawn { … }`, `channel[T]()`, `ch <- v`, `let x <- ch` and `withMockDB { db => … }`; none of
  that parses. Gating them would mean implementing unrelated language features.
- `examples/experimental/README.md` **already** opens with *"**These examples DO NOT currently
  work!** They require features that are: on the roadmap / partially implemented / planned for
  future releases."* The directory is correctly and explicitly labelled as aspirational design
  documentation. Nothing is being hidden; adding a second warning would be noise.
- **And a gated example would not guard this bug anyway**: `verify-examples` runs
  `ailang run` (`scripts/verify_examples.go:108-115`) and **never** `ailang test`. Nothing in the
  repo gates `ailang test` on `.ail` files — `tests/record_update_regression_test.ail` is
  referenced by no Makefile target and no workflow (`rg` over `.github/workflows/`, `make/`,
  `scripts/`, `tools/` → zero hits). **The only instrument that can guard this is a Go test in
  `internal/testing`**, which is what M1–M3 build.

---

## 7. Non-goals (each with its reason)

- **Making `assert` legal in general expression position.** That is option (b) — a language change
  across 4 core packages needing a Conflict Surface, a design doc, and a quorum. §3.
- **Removing the print-and-reparse round-trip** (option (a)). Needs a new
  AST→lowered-Core-in-module-scope entry point that does not exist; `TestDecl` is never elaborated
  by the pipeline. §3, F1. Recorded as future work, §9.
- **Fixing F2 for non-assert expressions** (`test { false_expr; true_expr }` passes). Same function,
  genuinely the same family, and Critical Principle 3 says to look — I did, and I am **deliberately
  scoping it out**: making non-final plain expressions short-circuit changes the *type* obligation
  on every existing multi-expression named test body (a non-bool non-final expression would become
  a type error instead of being discarded), which is a semantics change to *currently-passing*
  tests. That is a behaviour decision, not a bug fix, and it deserves its own item. Asserts are
  safe to short-circuit precisely because they are bool by construction and 100% broken today, so
  no passing test can regress. **Recommend the controller file F2 as a separate issue** and note
  that this sprint's design is chosen so that fixing F2 later is a two-line extension of
  `FoldTestBody` (add the `default:` case to the sentinel walk).
- **Filing/fixing F3** (`Num[bool]` on the named-test path). Separate defect; does not block.
- **Touching the canonical prompt or `docs/`.** Per the P5 refutation the prompt never demonstrates
  `assert`, so there *is* a real teaching gap — but the canonical prompt is the eval harness's
  system prompt, and editing it mid-mission perturbs benchmark comparability across the very
  baselines this mission reads. **A worked `assert` example belongs in a prompt-manager item with
  its own A/B**, not smuggled into a bug fix. Flagged, not done.

---

## 8. Risks

| Risk | Impact | Mitigation |
|---|---|---|
| Sentinel `int` collides with a body that legitimately returns `int` | Low | Impossible: the sentinel path is entered **only** when the body contains an `AssertStmt`, and every such body is 100% broken today. Assert-free bodies keep the byte-identical bool path (AC-5, B3). |
| Nested `if` mis-associates when printed without parens | Medium if real | **Measured and refuted**: unparenthesized nested form ran `3 passed, 0 failed` across all-pass/first-fail/second-fail. §4. |
| `executor_helpers.go` breaches the 800-line gate | Medium | New file `test_body_lowering.go`; AC-6 gates it. |
| `AssertStmt` appears somewhere the flat walk misses | Low | **Measured**: `parseStatement` dispatches on `ASSERT` only at statement start, and a nested `assert` is a parse error. A flat walk is exhaustive. |
| A failing assert's message loses which one | — | AC-7 is exactly this criterion, and it is red before M2. |
| Executor "fixes" F2 opportunistically and regresses a passing test | Medium | §7 makes it an explicit non-goal with the reason. AC-5 catches it. |

---

## 9. Future work

- **Elaborate `ast.AssertStmt` directly** (F1) and give named-test bodies a real
  AST→Core-in-module-scope path, retiring the print-and-reparse round-trip for **all** node kinds,
  not just `assert`. This is the principled version of option (a) and would also retire the
  `<printer-error: …>` family in `PrintAILANGSource`.
- **F2** — non-final expression results discarded in named test bodies.
- **F3** — `Num[bool]` mis-inference on the named-test top-level-bare-expression path.
- A gate that runs `ailang test` over `.ail` fixtures (today **nothing** does).

---

**Document created**: 2026-08-06
**Base**: `dev` @ `64970c118`
</content>
</invoke>
