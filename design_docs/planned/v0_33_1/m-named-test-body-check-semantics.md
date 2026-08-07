# m-named-test-body-check-semantics — every expression in a pure named test body is a check (#604)

**Status**: Planned
**Target**: v0.33.1
**Priority**: P0 (High) — vacuous-pass class: a named test containing a failing check reports `All tests passed!` with exit 0
**Estimated**: 1–1.5 days (3 milestones; the third is optional and severable)
**Dependencies**: #590 assert lowering — implemented and live at the measurement base (the sentinel
machinery this design extends: `FoldTestBody`, `CheckInfo`, `decodeCheckSentinel`)
**GitHub issues**: [#604](https://github.com/sunholo-data/ailang/issues/604) (closes)
**Author**: design-doc-creator role, V1 mission iteration 157
**Created**: 2026-08-07
**Revised**: 2026-08-07 — revision pass 1 after multi-provider quorum (both objections upheld by
controller measurement; decisions recorded in **Quorum Revision Decisions** below)
**Measurement base**: `74dd06bb6` (`v0.33.0-60-g74dd06bb6`) — see Verification Log header

---

## Problem Statement

`test "name" { a; b }` evaluates `a`, binds its value to a dead `_seq` name, and grades ONLY `b`.
A failing check in non-final position is silently swallowed [V1]:

```ailang
module arm_a
export pure func add_one(n: int) -> int { n + 1 }
test "non-final discarded" {
  add_one(1) == 99;
  add_one(1) == 2
}
```

```
✓ All tests passed!
1 tests: 1 passed, 0 failed, 0 skipped
rc=0
```

The discriminating control — the same file shape with the order swapped, so the false check is
final — correctly fails with `expected true, got false`, rc=1 [V1]. Same binary, same shape; the
only variable is position. The instrument is live and the failure is positional, not luck.

This is the mission's **vacuous-pass class**: a check reporting success for work it never
performed, failing in the unsafe direction (reports green). Its sibling #590 (`assert` in named
bodies) at least failed loudly; #604 fails silently. Note #590's fix landed AFTER the issue was
filed and touches this exact code path — it deliberately did NOT fix #604 (see the verbatim scope
note in [V3]), so the bug is confirmed live at the measurement base [V1].

### Mechanism (line numbers re-derived at base — the issue's `runner.go:156-158` is stale)

- `internal/testing/test_body_lowering.go:30` `FoldBodyExprs` — legacy fold; non-final bare
  expressions are wrapped as `let _seq = <expr> in <rest>` (line 61). They are **evaluated but
  their value is discarded** — not dropped, dead. [V2]
- `internal/testing/test_body_lowering.go:24-26` — the doc comment claims stripping non-final
  expressions "is safe" because "pure bodies shouldn't have real side effects anyway, so semantics
  are preserved". **That sentence is the false premise this item exists to correct**: in a pure
  body, an expression whose value is thrown away is either a swallowed failing check or dead code.
  Neither is "safe" — one lies, the other is a user error worth surfacing. [V2]
- `internal/testing/test_body_lowering.go:132` `FoldTestBody` — the #590 sentinel path; line 212
  is the `_seq` wrap that non-final NON-assert expressions still take even on this path. [V2]
- `internal/testing/executor.go:184` `EvaluateNamedTestBodyExprs` — prints the folded AST back to
  AILANG source, appends it to the stripped module source, re-runs the FULL pipeline
  (`pipeline.Run` at executor.go:299), and evaluates; assert-bearing bodies return an int
  sentinel decoded by `decodeCheckSentinel` (executor.go:353). [V2] [V4]
- `internal/testing/runner.go:198-211` — "Pass contract: last expression must evaluate to bool
  true"; non-bool → `"expected bool result, got %T"` (runner.go:202). The comment at
  runner.go:152-163 documents the discard as intentional. [V2]

### The crux constraint the issue does not contain (V-D)

**The fold is purely syntactic — no type information exists at fold time.** `FoldTestBody` runs
BEFORE elaboration and type-checking: the folded AST is printed to source text and only then does
the pipeline (parse → elaborate → type-check → evaluate) run over it (executor.go:212 → :230 →
:299, in that order within one function) [V4]. Therefore the issue's suggested direction
"**conjoin all bool-valued expressions**" is **unimplementable as stated**: at fold time there is
no way to know which expressions are bool-valued, so no fold can selectively conjoin them and
pass the rest through. Any viable design must impose a **uniform syntactic rule** on non-final
bare expressions and let the type checker enforce the resulting obligation downstream. Both
options below satisfy that; the issue's literal "conjoin the bool-valued ones" does not.

---

## Goals

1. A failing check in ANY **top-level position** of a named test body fails the test — this
   closes the top-level instance of the vacuous-pass class, which is the shape #604 filed. It
   does NOT close the nested-block instance (see **Known Residual Vector** — measured live
   [V19], explicitly scoped out, follow-up filed as part of this sprint's definition of done).
2. Failing non-final checks report ordinal, source text, and the user's ORIGINAL position (reusing
   #590's `CheckInfo` machinery, which already does exactly this).
3. Every named test that passes today for a non-vacuous reason continues to pass, byte-identically
   where possible.
4. The rule is one sentence a model can hold: **every top-level bare expression in a named test
   body is a check; `let` binds; there is nothing else at top level.** Inside a check, ordinary
   expression semantics apply — including `*ast.Block`'s value-discard semantics, which is
   exactly why the residual vector exists and is named rather than papered over.

## Non-Goals

- No change to inline tests (`tests [(input, expected)]`) — structurally immune, see Scope.
- No change to property tests (`property "..."`) — separate runner path (`runProperty`).
- No new syntax, no new lexer/parser/AST/elaborator/type-checker code.
- No fix or scope-touch for #590 (already landed), #612, or the proxy-boundary item.
- No general temp-file→user-source position mapping for pipeline errors (optional M3 delivers a
  bounded hint instead; full mapping is future work).
- **No recursion into nested `*ast.Block` expressions inside checks.** The nested vacuous-pass
  vector is real, measured [V19], and deliberately NOT closed by this sprint — it is documented
  in **Known Residual Vector** and routed to a filed follow-up issue (a hard definition-of-done
  item, the same pattern this mission used for #612). This doc claims to close the top-level
  vector only.

---

## Quorum Revision Decisions (pass 1)

Both quorum objections were factually correct and controller-verified at base. The design
direction (option 2, sentinel-path lowering) was not disputed; the claims were wider than the
mechanism. Decisions:

**Objection 1 (trigger vs. uniformity claims) → option (a): narrow the claims, keep the
trigger.** The trigger `containsAssertStmt(exprs) || hasBareNonFinal(exprs)` intentionally
leaves single-final and lets+final bodies on the legacy path. On that path the final expression
has NO `CheckInfo` ordinal, and its bool obligation is enforced by the runner's pass contract —
non-bool → `expected bool result, got %T` (runner.go:198-211), measured [V18] — not by a lowered
`if` condition. Every uniformity sentence in this doc is now scoped to the sentinel path, and
the legacy path's enforcement mechanism is stated explicitly: **one enforcement mechanism per
path, both loud, no path leaves the bool obligation unenforced.** Why not option (b) (widen the
trigger so all bodies take the sentinel path): it buys zero safety — the legacy path already
fails loudly on both non-bool results [V18] and false results [V1 arm B] — while destroying the
byte-identical-legacy property for every currently-passing body (all 14 parseable tracked
fixtures [V5]), regenerating source for bodies the fix does not touch, and destabilizing #590's
pinned fixtures. Claims-accuracy problems are fixed in the claims, not by expanding the blast
radius.

**Objection 2 (nested-block vector) → option (ii): scope out honestly, file the follow-up as
part of this sprint's definition of done.** Full argument, repros, why-not-(i), why-not-(iii),
and the interim-diagnostic answer in **Known Residual Vector** below. The phrase "structurally
impossible" has been removed from this document; A5/A7 axiom scores were re-graded accordingly.

---

## The Decision

### What does a multi-expression PURE named test body MEAN?

**Chosen: Option (2) — every non-final bare (non-`let`, non-`assert`) expression IS a check**,
lowered through the #590 sentinel path exactly as if the user had written `assert` in front of it.
A non-bool non-final expression becomes a type error (the lowered `if <expr> then … else k`
imposes `bool` on `<expr>` through the existing pipeline, no new type rule).

The two options in the issue point in opposite directions — (1) says "bare bools are a mistake,
use `assert`"; (2) says "bare bools are checks". The decision is made on evidence, not taste:

**E1 — #590 already committed to "bare bool expression = check".** On the sentinel path, the
FINAL bare expression of an assert-bearing body is counted as a check with `IsAssert: false`, and
`decodeCheckSentinel` already carries a dedicated non-assert failure message for it:
``check %d failed: `%s` (at %s)`` (executor.go:370-371) [V10]. The blessed semantics for a bare
bool expression in a named test body is therefore already "this is a check" — option (2) extends
that to the remaining position; option (1) would contradict it (a bare bool would be a check in
final position and a type error one line up).

**E2 — the machinery's author scoped #604 as exactly this extension.** The comment at
`test_body_lowering.go:126-131` says verbatim that making non-final non-assert expressions
short-circuit "is a one-case extension of the switch below", deliberately deferred to #604
because it changes a type obligation [V3]. This design is that one-case extension plus the
trigger-condition change it implies.

**E3 — the measured cost of option (2) is zero in this repository.** The only shape option (2)
breaks that passes today is a *pure non-bool* non-final expression (e.g. `add_one(1); true` —
passes today, rc=0 [V7]). Zero git-tracked, parseable `.ail` files contain ANY bare non-final
expression in a named test body (V-E breakdown below, [V5]). Effectful non-finals
(`println("hi"); …`) **already fail today** with an effect-checking error on the synthetic `_seq`
function — measured rc=1 at base [V6] — so option (2)'s often-cited cost ("a `println` in an
effectful body becomes an error") does not exist: that body is already an error, and
`internal/testing/testdata/strip/named_test_effectful.ail` does NOT rely on it (its test body is
the pure single expression `big(20) == true`; the file exercises stripping of effectful *module
functions*, not effectful *test bodies*) [V5].

> **Stated assumption A-1** (quorum-requested): the "already an error today, so zero migration
> cost" claim DEPENDS on effect inference rejecting the synthetic `_seq` binding (it inherits no
> permissive effect row) — measured true at base [V6], but a property of current effect
> inference for synthetic bindings, not a permanent one. If that inference ever changes, this
> cost silently becomes nonzero. Guard: M2 fixture `effectful_nonfinal_stays_error.ail` pins the
> invariant "an effectful non-final expression in a named test body is a loud error" (pre-change
> via `_seq`, post-change via the lowered `if`-condition context), so drift fails a pin instead
> of going unnoticed.

**E4 — no taught idiom breaks.** The shipped canonical prompt teaches inline
`tests [(input, expected)]` as the recommended form and never shows a multi-expression named test
body [V11]. The shipped testing guide teaches only the single-expression `test "name" =
expression` form [V12]. Nothing users are taught to write changes meaning.

**E5 — option (2) repairs the natural shape; option (1) punishes it.** For
`test "x" { a == 1; b == 2 }` — the shape users naturally write, and the exact shape of the bug —
option (2) makes both checks count (the test starts doing what its author meant); option (1)
turns it into a type error demanding a rewrite to `assert`. Same migration surface, strictly
better destination.

### Rejected options

**(1) Non-final expressions must be `()`-typed; a discarded `bool` is a type error.**
Mechanically implementable despite V-D — the fold can emit `let _seq: () = <expr> in <rest>`, and
annotated unit lets are valid AILANG at base (verified, [V9]), so the type checker enforces the
obligation downstream just as in option (2). Rejected on E1 (contradicts the shipped
final-bare-expression-is-a-check semantics), E5 (turns today's vacuous pass into an error instead
of into a working test), and ceremony: it makes `assert` mandatory for multi-check bodies without
adding any safety over (2).

**(3a) Status quo + warning on discarded non-final expressions.** Rejected: leaves the vacuous
pass in place; a warning on a machine-consumed lane (exit codes, eval harnesses) is a no-op (A7).

**(3b) The issue's literal "conjoin all bool-valued expressions, pass others through".**
Rejected as unimplementable: requires type information at fold time, which does not exist [V4].

---

## Solution Design

### Overview

One-case extension of `FoldTestBody`'s switch, plus its trigger condition, all inside
`internal/testing/test_body_lowering.go`. No change outside `internal/testing/` except docs.

**New body semantics** (the one-sentence rule): a named test body is a `;`-separated sequence of
`let`/`let rec` bindings, `assert` statements, and bare expressions. Every `assert` condition and
every **top-level** bare expression — in any position of that top-level sequence — is a
**check**. The body passes iff every check evaluates `true`; evaluation short-circuits at the
first failing check (later checks are NOT evaluated, matching #590's existing contract).

**Two enforcement paths, both loud** (quorum decision, objection 1, option (a)):

- **Sentinel path** (bodies containing an `assert` OR a bare non-final expression — the new
  trigger): checks are numbered left-to-right (1-based, `CheckInfo.Ordinal`, unchanged
  encoding), and each check's `bool` obligation is enforced by the existing type checker
  through the lowered `if` conditions — a non-bool check is a type error [V8], not a silent
  discard. Ordinals and `check k failed` messages are **sentinel-path properties**.
- **Legacy path** (single-final and lets+final bodies — every currently-passing tracked fixture
  [V5]): the fold is byte-identical to today. The final expression is the body's one check; it
  has NO `CheckInfo` ordinal, and its `bool` obligation is enforced by the runner's pass
  contract — a non-bool result fails with `expected bool result, got %T` (runner.go:198-211),
  measured at base [V18]; a false result fails with `expected true, got false` [V1 arm B].

Neither path can silently discard a top-level check. What this rule does NOT govern: expressions
**inside** a check evaluate under ordinary language semantics — see **Known Residual Vector**.

### Changes to `FoldTestBody` (test_body_lowering.go)

1. **Trigger**: the sentinel path currently activates only when `containsAssertStmt(exprs)`
   (line 136). New trigger: `containsAssertStmt(exprs) || hasBareNonFinal(exprs)`, where
   `hasBareNonFinal` reports any non-final expression that is neither `*ast.Let` nor
   `*ast.LetRec` (nor `*ast.AssertStmt`, already covered). Bodies consisting of lets + a final
   expression — every currently-passing tracked fixture [V5] — keep the legacy `FoldBodyExprs`
   path **byte-identically**, preserving #590's "no currently passing test changes code path"
   property for them. Consequence, stated plainly (objection 1): those bodies' final expression
   is never numbered and never lowered — its bool obligation is the runner's `expected bool
   result` contract [V18], not a lowered `if`. That is a deliberate two-mechanism design, not an
   oversight; see Quorum Revision Decisions.
2. **`isCheck`** (line 241): a non-final bare expression now returns `true` (currently only the
   final one does). Asserts and the trailing-`let` degenerate case are unchanged.
3. **The switch's `default` case** (line 209-216): instead of `let _seq = e in rest`, emit
   `if e then rest else intLit(ordinal)` — the same shape the `*ast.AssertStmt` case emits one
   arm up (line 187-192), with `IsAssert: false` in its `CheckInfo` so the failure message quotes
   what the user actually wrote (no fabricated `assert` prefix — the mechanism #590 already built
   for the trailing bare expression, executor.go:363-371 [V10]).
4. **Rewrite the false-premise comments**: `FoldBodyExprs`'s lines 24-26 ("stripping them is
   safe") and `FoldTestBody`'s lines 126-131 (the #604 deferral) describe the old semantics and
   must be replaced, along with `runner.go:152-163` ("Earlier expressions are evaluated for
   side-effects only"). Per this repo's standards, stale comments about check semantics are
   themselves a vacuous-pass vector for the next reader.

**Not changed**: `decodeCheckSentinel` (executor.go:353-378) — already decodes non-assert checks;
`EvaluateNamedTestBodyExprs` — already routes any non-empty `checks` slice through the sentinel
decode (executor.go:337-339); the parser — the body grammar (`parseStatement` loop,
parser_test_decl.go:37-59 [V17]) is untouched; `FoldBodyExprs` itself — still used by the legacy
path and by `executor_helpers.go:367`'s consumer.

### Ordinals and the #590 interaction

Checks are numbered left-to-right in source order over the UNION of asserts and bare expressions.
Consequences:

- A body with asserts only, or asserts + lets (+ optional trailing bare expression): **ordinals
  identical to today** — the union adds no members. All seven `testdata/assert/` fixtures are in
  this class [V5]; their reported ordinals do not change.
- A body mixing asserts AND non-final bare expressions: bare expressions now consume ordinals, so
  an assert after a bare expression gets a higher ordinal than it would today. In such a body
  today the bare expression is silently discarded — i.e. **ordinal numbering changes only inside
  bodies that are currently exhibiting the #604 bug**. Zero tracked files contain this mix [V5].
- The sentinel encoding (0 = all passed, k ≥ 1 = check k first to fail, later checks not
  evaluated) and the `CheckInfo` struct are unchanged; `IsAssert` already distinguishes the two
  failure-message forms.
- **Legacy-path bodies have no ordinals at all.** A single-final or lets+final body never enters
  the numbering pass; its failure messages are the runner's (`expected true, got false` /
  `expected bool result, got %T` [V18]), not `check k failed`. Ordinal numbering is a
  sentinel-path property and this doc claims it for nothing else.

### Diagnostics

No new error-code namespace is introduced. `internal/testing`'s runtime errors are uncoded
strings today (grep of non-test files finds no `TST`/`TEST_` codes; positive control: the parser's
coded `p.report("PAR_…")` calls in the same sweep) [V13]. The failure surfaces below are the
**sentinel path's**; on the legacy path the surfaces are the runner's existing
`expected true, got false` and `expected bool result, got %T` [V18], unchanged by this design:

1. **A failing check** (the common case): the existing `decodeCheckSentinel` message, with the
   check's ordinal, the user's source text, and the user's ORIGINAL position —
   ``check 1 failed: `add_one(1) == 99` (at arm_a.ail:4:3)`` — because `CheckInfo.Pos` is
   captured from the user's AST at fold time, before the round-trip [V2] [V10]. This is precise
   user-source pointing with zero new code.
2. **A non-bool check** (the new type obligation): the pipeline's existing unification error from
   the regenerated source. Measured today for the identical case of a non-bool `assert`
   condition [V8]: the test FAILS (rc=1) with
   `pipeline error: type error in _namedtest_body_… : type unification failed at [if condition at
   _namedtest_body_….ail:4:3]: cannot unify type constructors: int vs bool`, and the runner
   anchors the failure at the test block's user-source location (`at assertnonbool.ail:3:1`). So
   the user sees WHICH test failed at a real user position, plus a correct-but-temp-file-pointing
   type message. Option (2) inherits this behavior at exact parity with #590's shipped assert
   path — it does not regress it, and importantly the failure direction is safe (loud error, not
   silent pass).
3. **An effectful check**: unchanged failure class — effect checking already rejects effectful
   expressions in these bodies (measured [V6], subject to stated assumption A-1, pinned by
   fixture in M2); the error message's synthetic function name changes from `_seq` to the
   check's lowered `if`-condition context, still an error either way.

**Optional M3 (severable, +0.5 day)**: when `pipeline.Run` fails for a body whose `checks` slice
is non-empty, wrap the error with the check table — one line per check: ordinal, user source,
user position — so a temp-file-pointing type error is accompanied by the user-source candidates.
~20 LOC in `EvaluateNamedTestBodyExprs`. Full line-level temp→user position mapping is explicitly
future work, not this sprint.

### Scope: which harnesses need the change

**Only the named-test path** (`runNamedTest` → `EvaluateNamedTestBodyExprs`). The inline-test
harness (v0.4.7 module-scope path) is structurally immune: inline tests are
`tests [(input1, expected1), …]` — input/expected PAIRS parsed by `parseTestsBlock`
(internal/parser/parser_testing.go:8), with no expression-sequence body for a check to hide
in [V11] [V14]. Property tests route through `runProperty` (runner.go:219) — separate path, out
of scope. The legacy `test "name" = expr` single-expression form (shipped docs [V12], archived
examples [V5]) has exactly one expression, so it has no non-final position.

---

## Known Residual Vector: nested blocks (quorum objection 2 — scoped out, follow-up filed)

### The vector, measured

`*ast.Block` (internal/ast/ast_expr.go:271-276) is a general expression node whose documented
semantics are "the last expression is the return value, others are evaluated for effects" — i.e.
value discard, the same premise this doc corrects at the body's top level. `FoldTestBody`
operates ONLY on the top-level `[]ast.Expr` slice: its switch dispatches over `exprs[i]` as
`*ast.AssertStmt` / `*ast.Let` / `*ast.LetRec` / default, and no case descends into an
expression's children — a nested `Block` is one opaque expression to the fold [V20]. So the
vacuous-pass class survives ONE LEVEL DOWN, in two measured shapes, each with a discriminating
control [V19, controller, iteration 157]:

```ailang
-- Shape A: block inside an if-arm (single-final body → legacy path even post-fix)
test "..." { if true then { add_one(1) == 99; add_one(1) == 2 } else false }
--> rc=0, "All tests passed!"          FALSE CHECK SWALLOWED
-- Control A' (false check moved last): rc=1, "expected true, got false"

-- Shape B: bare block as the body's final expression
test "..." { { add_one(1) == 99; add_one(1) == 2 } }
--> rc=0, "All tests passed!"          FALSE CHECK SWALLOWED
-- Control B' (false check moved last): rc=1, "expected true, got false"
```

Position determines the verdict inside a nested block exactly as it does at top level today —
this is #604 one level down, and it is live syntax, not hypothetical. It persists after this
fix on BOTH paths: shape A/B bodies are single-final (legacy path, untouched), and even a
sentinel-path body like `{ {a; b}; c }` lowers the nested block as ONE check whose value is `b`
— `a` is still discarded inside it by ordinary `Block` semantics.

**Why this matters more after the fix than before it**: once #604 closes, users are told
multi-check bodies work, and will write nested ones. A doc that closed the top level while
claiming the class "structurally impossible" would convert a known bug into a trusted one. This
doc therefore claims top-level closure only, and makes the follow-up part of its definition of
done — the pattern this mission used for #612.

### Decision: option (ii) — scope out honestly, with a filed follow-up

**Why not (i), recurse the fold**: the sentinel is an int VALUE returned as the folded body's
result — `if c then rest else k` only works where the enclosing context expects the sentinel,
i.e. at the top of the body. Inside a nested position the enclosing expression imposes its own
type (the then-arm of shape A must be `bool` because the else-arm is `false`); an int-sentinel
arm there is a type error. Threading ordinals through nesting therefore means rewriting
ENCLOSING expressions (a CPS-like transform) or a new failure channel (exception/effect-like) —
neither exists in this machinery [V20]. That is a redesign of the failure channel, not the
"one-case extension of the switch" [V3] this sprint is scoped to; it does not fit 1–1.5 days.
If chosen later, it decomposes as: (1) exhaustive AST-walker infrastructure, (2) failure-channel
design, (3) ordinal scheme across nesting levels — the follow-up's problem statement.

**Why not (iii), static error on nested multi-expression blocks**: rejecting the shape requires
an exhaustive recursive walk over every AST node that carries expression children, and
`internal/ast` has no generic `Walk`/`Inspect` to lean on [V21]. A hand-written walk that
misses one node type silently re-opens the gap — the exact failure mode of this bug class, now
hidden behind a claim of closure. An exhaustiveness-guaranteed walker is real infrastructure,
belongs to the follow-up, and would also serve option (i). (iii) also over-rejects: the walker
would have to reach into lambda bodies and match arms where a multi-expression block may be
deliberate. Cheapest to REASON about, not cheapest to BUILD correctly.

**What this sprint does about it** (all in M2, definition of done):

1. **The follow-up issue is ALREADY FILED — it is [#614](https://github.com/sunholo-data/ailang/issues/614)**,
   opened by the controller during this iteration's quorum round (2026-08-07), carrying both
   measured shapes and both discriminating controls from [V19] verbatim. It exists BEFORE #604
   closes, which is the #612 pattern this DoD item was written for. What remains in M2 is
   therefore linking, not filing: reference `#614` from the CHANGELOG entry, from the rewritten
   comment in `test_body_lowering.go`, and from the testing-guide section. The follow-up decides
   between (i)-redesigned, (iii)-with-walker, or a
   language-level lint on value-discarding pure `Block`s — the vector is the general semantics
   of `Block` in pure code, not something the test lowering can fully own.
2. **Pin the residual**: fixture `nested_block_residual.ail` (shape A) checked in with a loud
   comment, asserting the CURRENT vacuous behavior, so the follow-up flips a failing pin
   instead of rediscovering the shape.
3. **Document the boundary**: the testing-guide update states the top-level rule AND the nested
   residual explicitly, so no reader can honestly generalize the fix to nested blocks.

**Interim WARN diagnostic** (the quorum asked): technically feasible — a best-effort syntactic
walk covering at least the two measured shapes (`If` arms and a final bare `Block`) is ~30 LOC,
and partial coverage is acceptable for a warning where it would be disqualifying for an error.
But by this doc's own A7 argument (the rejection of option 3a), a warning on a machine-consumed
lane is a no-op — so it is offered as an OPTIONAL M3 rider and counted as mitigation for
nothing. The follow-up issue, not the warning, is the accountable artifact.

---

## Migration / Compatibility

Quantified from the V-E breakdown [V5], base `74dd06bb6`:

| Corpus | Count | Post-change behavior |
|---|---:|---|
| Git-tracked `.ail` files (control: instrument sees corpus) | 621 | — |
| …containing a named test block (`test "…"`) | 19 | — |
| …using only the legacy `test "…" = expr` form (archived, `examples/archive/broken/`) | 2 | unchanged (single expression, no non-final position) |
| …in `examples/experimental/`, **fail to parse at base** (measured per-file [V5]) | 3 | unreachable by `ailang test` before AND after; unchanged |
| …parseable, bodies = single expr, lets+final expr, or assert-bearing (#590 fixtures) | 14 | **byte-identical fold or identical ordinals — zero behavior change** |
| …parseable, with a bare non-final expression in a named test body (**the shape this design changes**) | **0** | — |

Positive control for the zero: the same inspection instrument (the block-dump command in [V5])
applied to the repro file `arm_a.ail` displays the vulnerable shape; the instrument can see a hit.

**Out-of-repo bodies that change behavior**, exhaustively by shape:

| Shape today | Today | After | Direction |
|---|---|---|---|
| `{ <bool that is true>; <final> }` | passes vacuously | passes genuinely | none visible |
| `{ <bool that is false>; <final true> }` | **passes (the bug)** | fails with check ordinal + user position | the fix |
| `{ <pure non-bool>; <final> }` | passes (value discarded) | type error, test fails loudly [V7→V8 class] | **the one breaking change** |
| `{ <effectful expr>; <final> }` | already fails (effect check [V6]) | still fails (error class may shift effect→type) | none material — subject to stated assumption A-1 below |
| checks nested INSIDE another expression (e.g. `{ if c then { a; b } else d }`) | non-final `a` discarded inside the block [V19] | **unchanged — the residual vector**, pinned by fixture + filed follow-up | none in this sprint (Known Residual Vector) |

The one breaking change is the intended semantics correction: a discarded pure non-bool value in
a pure body is dead code, and this design chooses (with #590's precedent) to make it a loud error
rather than silence. Migration for an affected user is mechanical: bind it (`let _ignored = e;`)
or delete it. This goes in the CHANGELOG under breaking changes for v0.33.1.

---

## Implementation Plan and Milestones

Fits the 0.5–2 day gate: **M1+M2 ≈ 1 day; optional M3 +0.5 day.**

#### M1 — Lowering extension (~0.5 day)

- `internal/testing/test_body_lowering.go`: `hasBareNonFinal` helper; trigger change in
  `FoldTestBody`; `isCheck` extension; `default:` switch case emits the `if`-chain arm with
  `IsAssert: false`; rewrite the three stale comments (test_body_lowering.go:24-26, :126-131,
  runner.go:152-163). ~40 LOC net.
- Update `named_test_assert_test.go`'s byte-identical-legacy pin (line 178 region): its premise
  narrows from "assert-free bodies fold byte-identically to legacy" to "assert-free bodies with
  no bare non-final expressions fold byte-identically to legacy". Per Testing Policy, the old
  test is rewritten, not preserved.

#### M2 — Fixtures and end-to-end tests (~0.5 day)

- New fixtures under `internal/testing/testdata/assert/` (same convention as #590's):
  `bare_first_fails.ail` (arm A shape — the regression test for #604), `bare_second_passes.ail`
  (arm B control), `bare_mixed_assert.ail` (bare + assert ordinal interaction),
  `bare_nonbool_nonfinal.ail` (type-error surface), `lets_only_legacy.ail` (byte-identical legacy
  guard), `legacy_nonbool_final.ail` (pins the legacy-path runner contract [V18]),
  `nested_block_residual.ail` (pins the residual vector, Known Residual Vector item 2),
  `effectful_nonfinal_stays_error.ail` (pins assumption A-1). Each validated with `ailang test`
  before commit (skill hard gate).
- **Link the nested-block follow-up issue `#614`** (Known Residual Vector item 1 — ALREADY FILED 2026-08-07) from the
  CHANGELOG entry, the rewritten `test_body_lowering.go` comment, and the testing guide. #604
  does not close without this link chain.
- Unit tests pinning: sentinel shape for a two-bare-check body; ordinal assignment in mixed
  bodies; trigger condition (lets+final stays legacy); failure messages (non-assert form, no
  fabricated `assert` prefix).
- `make test` + `make verify-examples` green (the 3 experimental files already fail to parse at
  base [V5]; verify-examples baseline unchanged).

#### M3 (OPTIONAL, severable) — check-table hint on pipeline errors (~0.5 day)

- Wrap `pipeline.Run` errors in `EvaluateNamedTestBodyExprs` with the `CheckInfo` table when
  `len(checks) > 0`. If the sprint runs long, cut M3 without loss of the fix.
- Optional rider: best-effort nested-block WARN (~30 LOC, Known Residual Vector) — advisory
  only, mitigation for nothing; cut with M3.

#### Documentation (inside M2)

- `CHANGELOG.md`: breaking-change entry (the pure-non-bool shape) + the fix.
- `docs/docs/guides/testing.md`: document multi-check named bodies (currently teaches only the
  `=` form [V12]).
- Close #604 via commit message.

### Files to Modify/Create

- `internal/testing/test_body_lowering.go` — trigger, `isCheck`, switch case, comments (~40 LOC)
- `internal/testing/runner.go` — comment block 152-163 only (no code change)
- `internal/testing/named_test_assert_test.go` — rewrite legacy-pin premise, add ordinal/shape tests (~120 LOC)
- `internal/testing/testdata/assert/*.ail` — 8 new fixtures (~60 LOC)
- GitHub: follow-up issue `#614` ALREADY FILED (nested-block residual); M2 DoD is to LINK it, no code
- `internal/testing/executor.go` — M3 only: error-wrap hint (~20 LOC)
- `CHANGELOG.md`, `docs/docs/guides/testing.md` — docs (~40 LOC)

---

## Examples

All AILANG below was executed at base with the freshly built binary [V0]; commands and outputs in
the Verification Log.

**The bug, and what this design makes of it** (source verified via `ailang test` [V1]):

```ailang
module arm_a
export pure func add_one(n: int) -> int { n + 1 }
test "non-final discarded" {
  add_one(1) == 99;
  add_one(1) == 2
}
```

Today: `✓ All tests passed!`, rc=0 [V1]. After: fails with
``check 1 failed: `add_one(1) == 99` (at arm_a.ail:4:3)``, rc=1 — the message format and
position mechanism already shipped in #590 [V10]; only the lowering of the non-final position
changes.

**The one breaking shape** (source verified via `ailang test`, passes today rc=0 [V7]):

```ailang
module purenonbool
export pure func add_one(n: int) -> int { n + 1 }
test "pure non-bool non-final" {
  add_one(1);
  add_one(1) == 2
}
```

Today: passes (int value discarded). After: type error `cannot unify type constructors: int vs
bool` surfaced as a loud test failure, anchored at the test's user-source location — the exact
diagnostic class the shipped assert path produces for a non-bool condition today [V8].

---

## Success Criteria

- [ ] Arm A (`bare_first_fails.ail`) fails with rc=1, reporting check 1 with the user's source
      text and original position; arm B control still fails on its final check.
- [ ] All seven existing `testdata/assert/` fixtures: identical pass/fail AND identical ordinals.
- [ ] Lets+final-expression bodies produce byte-identical folds to the legacy path (pinned test),
      and `legacy_nonbool_final.ail` pins the runner's `expected bool result` contract on that
      path [V18].
- [ ] `{ <pure non-bool>; <expr> }` fails loudly with the unification error (no silent pass).
- [ ] Nested-block follow-up issue `#614` (filed 2026-08-07) LINKED from CHANGELOG, code comment, and testing
      guide; `nested_block_residual.ail` pin in place. #604 does not close without this.
- [ ] `effectful_nonfinal_stays_error.ail` pins assumption A-1.
- [ ] `make test`, `make verify-examples`, `make ci` green.
- [ ] CHANGELOG breaking-change entry + testing-guide update (top-level rule AND nested residual)
      landed.

## Timeline

Day 1: M1 + M2 (fix, tests, fixtures, docs). Day 1.5 (optional): M3. **Revision pass 1 scope
check**: the quorum-driven additions are 3 fixtures (~20 LOC), one issue filing, and doc text —
≈1h inside M2, because the chosen answers were claims-narrowing (objection 1a) and scope-out
(objection 2ii), not mechanism growth; the 1–1.5 day estimate survives. The rejected
alternatives that would NOT have fit are recorded in Quorum Revision Decisions and Known
Residual Vector, with a decomposition sketch for recursion. If implementation reveals scope
beyond this — e.g. the trigger change destabilizes the #590 fixtures in a way the pinned tests
can't absorb — STOP and split M3 + diagnostics into a follow-up rather than growing the sprint.

## Risks & Mitigations

- **Risk**: printed `if`-chains for bare checks hit a printer gap (`PrintAILANGSource`) for some
  expression form. *Mitigation*: the identical chain shape is already exercised by #590's assert
  fixtures through the same printer; new fixtures extend coverage to bare-expression conditions.
  Any check whose condition prints wrongly fails LOUDLY (parse/type error in the round-trip),
  never silently.
- **Risk**: an out-of-repo user relies on `{ e; final }` discarding `e`. *Mitigation*: breaking
  change is documented, the migration is a one-line `let _ignored =` binding, and the error is
  loud with the test's location.
- **Risk**: ordinal renumbering in mixed bodies confuses tooling parsing failure messages.
  *Mitigation*: only bodies currently exhibiting the #604 bug renumber [V5]; message FORMAT is
  unchanged.
- **Risk**: users generalize the fix — told multi-check bodies work, they write nested blocks
  and trust the green [V19]. This is the residual vector's trust-transfer failure mode.
  *Mitigation*: the testing guide states the top-level boundary explicitly; the residual is
  pinned by fixture and carried by a filed follow-up (Known Residual Vector). NOT mitigated by
  the optional WARN — a warning on a machine lane counts for nothing (A7).

---

## Conflict Surface

This change alters a **type obligation** (non-final bare expressions: unconstrained → `bool`) but
implements it entirely inside `internal/testing`'s lowering; the obligation is enforced by the
unmodified type checker through the regenerated source's `if` conditions. Enumeration of
everything that reads named test bodies (grep-derived [V14] [V15]):

1. **`internal/parser/parser_test_decl.go`** (parseTestDecl, :37-59) — parses the body as a
   `parseStatement` loop into `TestDecl.Body`. **No grammar change**; the set of parseable bodies
   is identical [V17].
2. **`internal/testing/collector.go:58,71`** — copies `TestDecl.Body` verbatim into
   `TestCase.Body`. Unaffected (no interpretation).
3. **`internal/testing/test_body_lowering.go`** — the change site (above).
4. **`internal/testing/executor.go`** — consumes the fold; `decodeCheckSentinel` already handles
   `IsAssert: false` checks [V10]. No change (M3 optional wrap aside).
5. **`internal/testing/runner.go:164-216`** — consumes bool-or-error; unchanged code, corrected
   comment.
6. **`internal/testing/source_strip.go:71`** — strips `TestDecl`s when regenerating module
   source; reads only the decl boundary, never the body semantics. Unaffected.
7. **`internal/format/decl.go:28,516` + `internal/format/expr.go:77,562`** — `ailang fmt`
   formats test bodies as written (including `AssertStmt` inside bodies); it never evaluates or
   lowers. Unaffected — a formatted body round-trips to the same statement sequence.
8. **Elaborator / type checker** — `internal/elaborate/` has NO `TestDecl` case (grep empty;
   positive control: `ast.FuncDecl` cases present in the same package) [V15]. Test bodies reach
   the type checker only as regenerated ordinary source via the executor round-trip, so no
   elaborator/type-checker code path changes.
9. **`internal/testing/named_test_assert_test.go:178`** — pins "assert-free fold ==
   `FoldBodyExprs` byte-identically". Its premise narrows under this design (intentional
   incompatibility; rewritten in M1).
10. **`internal/ast/ast_expr.go:271-276` (`*ast.Block`)** — NOT modified, and that is a
    decision, not an omission: `Block`'s value-discard semantics inside a check is the residual
    vector [V19] [V20], scoped out per Known Residual Vector and carried by the filed
    follow-up. Any future change to `Block` semantics is the follow-up's conflict surface, not
    this sprint's.

**Programs that MUST still work** (all verified present at base [V5]): the seven
`internal/testing/testdata/assert/` fixtures; `internal/testing/testdata/strip/named_test_{annotated,contract,effectful,multiline}.ail`
and `malformed_control.ail`; `tests/record_update_regression_test.ail` (three lets+final bodies —
legacy path, byte-identical); `internal/pkg/testdata/multi_module/src/core_test.ail`.

**Deliberately changes**: `{ <bare non-final>; … }` bodies — swallowed-check → enforced-check
(the fix), pure-non-bool-discard → type error (the breaking change), and the
`named_test_assert_test.go:178` pin's premise.

The honest answer is not "no conflicts": the conflicts are (a) the narrowed
byte-identical-legacy invariant (item 9), (b) the type obligation on shapes that today pass
vacuously — both intentional, both quantified at zero occurrences in-repo [V5] — and (c) the
DELIBERATE non-conflict of item 10: this sprint leaves `Block` semantics untouched, so the
nested vacuous-pass vector remains open, named, pinned, and filed rather than claimed closed.

---

## Verification Log

**Measurement base**: `74dd06bb6` (`git describe` → `v0.33.0-60-g74dd06bb6`). Binary:
freshly built at base; `ailang --version` → `AILANG v0.33.0-60-g74dd06bb6-dirty` — the `-dirty`
is the pre-existing rig-synced working-tree files (`docs/static/benchmarks/os/*.json`,
`.claude/fmt_hook_events.jsonl` per `git status`), no Go or `.ail` source modified. Rows marked
**[controller V-x]** were measured first-party by the mission controller this iteration at the
same base and are cited with their commands; rows marked **[controller, iteration 157]** (V18,
V19, and the V6 re-measurement) were measured by the controller during quorum-objection
verification at the same base with a freshly built binary (`ailang --version` == `git describe`
== `v0.33.0-60-g74dd06bb6`) and are recorded per controller direction, not re-run by this
author; all other rows were run by this author in-session.

| # | Claim | Command | Observed |
|---|---|---|---|
| V0 | Binary under test is the base commit | `ailang --version; git describe --tags` | `AILANG v0.33.0-60-g74dd06bb6-dirty / Commit: 74dd06b` and `v0.33.0-60-g74dd06bb6` — match |
| V1 | Bug live at base, positional (discriminating control) — re-run of [controller V-A] | `ailang test arm_a.ail; echo rc=$?` then `ailang test arm_b.ail; echo rc=$?` (sources in Examples; arm_b = order swapped) | Arm A: `✓ All tests passed! … 1 passed, 0 failed`, `rc=0`. Arm B: `✗ … expected true, got false … 0 passed, 1 failed`, `rc=1`. Same shape, same binary, only position varies |
| V2 | Mechanism sites at base (issue's `runner.go:156-158` is stale) — [controller V-B], confirmed by reading each file this session | `Read internal/testing/test_body_lowering.go`, `Read internal/testing/executor.go:140-378`, `Read internal/testing/runner.go:150-216` | `FoldBodyExprs` at test_body_lowering.go:30, `_seq` wraps at :61 and :212, "is safe" premise at :24-26, `FoldTestBody` at :132; `EvaluateNamedTestBodyExprs` at executor.go:184; pass contract at runner.go:198, `"expected bool result, got %T"` at runner.go:202 |
| V3 | #604 scoped by #590's author as a one-case extension of the sentinel switch — [controller V-C], confirmed by reading | `Read internal/testing/test_body_lowering.go:126-131` | Verbatim: non-final non-assert exprs keep legacy `_seq`; short-circuiting them "would turn a non-bool non-final expression into a type error in bodies that pass today — … tracked separately as #604. Adding it later is a one-case extension of the switch below." |
| V4 | Fold runs BEFORE elaboration/type-checking; no type info at fold time — [controller V-D], confirmed by reading the call order | `Read internal/testing/executor.go:184-341` | Within one function: `FoldTestBody` (:212) → `PrintAILANGSource` into combined source (:230) → `pipeline.Run` (:299) → `EvalCoreProgram` (:330) → `decodeCheckSentinel` (:337). The fold's output is source TEXT re-entering the full pipeline; "conjoin bool-valued exprs" is unimplementable at fold time |
| V5 | Blast radius: 19 tracked `.ail` files have named test blocks (corpus control 621); **0 parseable files contain a bare non-final expression in a named test body** | `git ls-files '*.ail' \| xargs grep -l -E '^\s*test\s+"'` → 19 files (list in doc body); corpus control `git ls-files '*.ail' \| wc -l` → 621 [controller V-E]. Breakdown (this author): `for f in <19 files>; do grep -n -A 12 -E '^[[:space:]]*test[[:space:]]+"' "$f"; done` + `cat` of the 4 truncated fixtures — every block classified: 2 files legacy `=` form; 3 experimental; 14 parseable with bodies = single-expr / lets+final / assert-bearing only. Experimental reachability: `ailang check examples/experimental/{concurrent_pipeline,web_api,factorial}.ail` → `PAR_NO_PREFIX_PARSE:14:17`, `IMP012_UNSUPPORTED_NAMESPACE:10:12`, `PAR_UNEXPECTED_TOKEN:18:20` — none parse. **Positive control** for the zero: same dump command on repro dir shows `arm_a.ail:3 test "non-final discarded" { add_one(1) == 99; add_one(1) == 2 }` — instrument sees the shape | Count of the vulnerable shape among tracked, parseable files: **0** |
| V6 | Effectful non-final expressions ALREADY fail at base (option 2's alleged cost is nonexistent — SUBJECT TO stated assumption A-1) | `ailang test effectful.ail` with body `{ println("hi"); add_one(1) == 2 }` (+ `import std/io (println)`); rc via separate clean run. Independently re-measured by controller, iteration 157 (same command shape, same output; controller notes omitting the import instead yields `undefined variable: println` — an instrument error, not this row) | `✗ … pipeline error: effect checking failed … Effect checking failed for function '_seq' … Missing effects: IO`, `rc=1` |
| V7 | Pure non-bool non-final passes today — the ONLY shape this design breaks | `ailang test purenonbool.ail` with body `{ add_one(1); add_one(1) == 2 }` | `✓ All tests passed!`, `rc=0` |
| V8 | Non-bool check → loud type error anchored at the test's user location; temp-file-pointing detail (existing #590 parity target) | `ailang test assertnonbool.ail` with body `{ assert add_one(1); add_one(1) == 2 }` | `✗ … pipeline error: type error in _namedtest_body_… : … [if condition at _namedtest_body_….ail:4:3]: cannot unify type constructors: int vs bool` + runner anchor `at assertnonbool.ail:3:1`, `rc=1` |
| V9 | Option (1) is mechanically implementable (annotated `()` lets valid) — honesty row for the rejected option | `ailang test annlet.ail` with body `{ let u: () = (); f() }` and `ailang check annlet.ail` | `rc=0`; `✓ No errors found!` |
| V10 | Non-assert checks already have a shipped failure form with original user positions | `Read internal/testing/executor.go:353-378` | ``check %d failed: `%s` (at %s)`` at executor.go:370-371, keyed on `CheckInfo.IsAssert == false`; `CheckInfo.Pos` captured pre-round-trip (test_body_lowering.go:77,162) |
| V11 | Shipped prompt teaches inline `tests [...]` pairs, never multi-expression named bodies | `ailang prompt \| grep -n -i 'assert\|test'` (18 `test` hits inspected; positive control: same pipe counts 200 `func` hits) | Line 2227-2232: "Inline tests on functions (recommended): `pure func square(x: int) -> int tests [(0, 0), (5, 25)] { x * x }`"; `test`/`assert` otherwise appear only in the keyword table (line 2269). No `test "…" { a; b }` shape anywhere |
| V12 | Shipped docs teach only single-expression `test "name" = expression`; no doc teaches bare multi-check bodies | `grep -rn -A6 'test "' docs/docs/ --include='*.md'` (11 occurrences, all inspected; the count itself is the instrument-sees-corpus control) | `docs/docs/guides/testing.md:78` `test "name" = expression` + examples; `docs/docs/reference/effects.md:588` shows an effect-mock form using function-call `assert(...)`; zero occurrences of the `{ a; b }` bare multi-check shape |
| V13 | No error-code namespace exists in `internal/testing` runtime code (new codes not required) | `grep -rn 'TST\|"TEST_' internal/testing/*.go \| grep -v _test.go` → empty; positive control: `p.report("PAR_UNEXPECTED_TOKEN", …)` sites read directly in parser_test_decl.go:14,21,29 | Empty on runtime files; control shows coded reporting exists and is greppable in the same tree |
| V14 | Exhaustive consumer set of `TestDecl`/`FoldBodyExprs`/`FoldTestBody`/`EvaluateNamedTestBodyExprs` (Conflict Surface basis) | `grep -rn "FoldBodyExprs\|FoldTestBody\|EvaluateNamedTestBodyExprs" --include='*.go'` and `grep -rln "TestDecl" internal/ cmd/ --include='*.go'` | Consumers: parser (parser_decl.go:400,406; parser_test_decl.go), testing (collector.go:58, source_strip.go:71, test_body_lowering.go, executor.go:212, executor_helpers.go:367, runner.go:189), format (decl.go:28,516; expr.go:77,562), plus tests. No others outside `_test.go` name-collisions (`TestDeclare…` funcs in smt/types — inspected, unrelated) |
| V15 | Negative claim: the elaborator has NO `TestDecl` case (type obligation needs no elaborator change) | `grep -rn "TestDecl" internal/elaborate/*.go` → exit 1 (no match); positive control in same package: `grep -rn "ast.FuncDecl" internal/elaborate/*.go` → 3 hits (file.go:234,389,400) | Confirmed absent with a control proving the instrument sees decl-type cases in that package |
| V16 | Duplicate/coverage gate: no existing doc covers #604; nearest sibling deliberately scopes it out | `ailang docs search "named test body check semantics multi-expression" --neural` (SimHash results returned; no neural scores were produced, so none are asserted) + `ls design_docs/planned/v0_33_1/` + `head -40 …/m-named-test-assert-lowering-sprint-plan.md` | Top SimHash hits unrelated (CI build speed 1.00 is a keyword artifact). `v0_33_1/` contains #590's sprint plan (`m-named-test-assert-lowering-sprint-plan.md`, "closes #590", design-doc-NONE bug-fix lane) — which, with [V3], defers #604 to a separate item: this document |
| V17 | Parser body grammar is a statement loop; no grammar change needed for this design | `Read internal/parser/parser_test_decl.go:8-72` | Body = loop of `parseStatement` + optional semicolons into `TestDecl.Body []ast.Expr` (:37-59); asserts parse only at statement start (containsAssertStmt comment, test_body_lowering.go:223-228) |
| V18 | Legacy-path bool obligation is the RUNNER's contract, not a lowered `if` — the basis of the two-path claims (objection 1) — [controller, iteration 157] | `ailang test` on `test "non-bool final legacy path" { 42 }` | `rc=1`, error text `expected bool result, got *eval.IntValue` — rejected at runner.go:198/202, no `CheckInfo` ordinal involved. Loud failure, wrong-mechanism-for-the-old-claim, hence claims narrowed not trigger widened |
| V19 | Nested-block vacuous pass is LIVE at base in two shapes, each with a discriminating control (objection 2) — [controller, iteration 157] | Shape A: `ailang test` on `test "…" { if true then { add_one(1) == 99; add_one(1) == 2 } else false }`; control A′ = same body, false check moved last. Shape B: body `{ { add_one(1) == 99; add_one(1) == 2 } }`; control B′ = false check moved last. All four files parse and run at base | A: `rc=0`, `All tests passed!`, 1 passed 0 failed — false check swallowed. A′: `rc=1`, `expected true, got false`, 0 passed 1 failed. B: `rc=0`, `All tests passed!`. B′: `rc=1`, `expected true, got false`. Position determines verdict inside nested blocks exactly as at top level |
| V20 | `FoldTestBody` never descends into expression children; a nested `Block` is opaque to it; `Block` semantics discard non-final values | `Read internal/testing/test_body_lowering.go` (full file) + `Read internal/ast/ast_expr.go:271-286` | Fold switch dispatches over top-level `exprs[i]` as `*ast.AssertStmt`/`*ast.Let`/`*ast.LetRec`/default (:182-217); no case recurses into children; sentinel is an int VALUE built at body top (:169-180). `type Block struct { Exprs []Expr; … }` at ast_expr.go:273 with doc comment "The last expression is the return value, others are evaluated for effects" (:271-272) |
| V21 | Negative claim: no generic AST walker exists in `internal/ast` (cost basis for rejecting (iii)/full-WARN in this sprint) | `grep -rn "func Walk\|func Inspect\|func.*Visit" internal/ast/*.go` → exit 1 (no match); positive control in same sweep: `grep -c "func.*Position" internal/ast/ast_expr.go` → 23 | Confirmed absent with a control proving the instrument sees function decls in that package |

**Not verified / limits of this log**: (a) post-fix outputs in Examples are the design's intended
behavior, not measured — the fix does not exist yet; every PRE-fix claim they rest on is measured
above. (b) The V5 breakdown covers git-tracked files only; out-of-repo user code is addressed by
the shape table in Migration, not by measurement. (c) The neural doc-search backend returned no
neural scores in-session (SimHash only); the duplicate gate rests on the SimHash sweep plus
direct inspection of `design_docs/planned/v0_33_1/`. (d) V18/V19 are controller-measured and
recorded per controller direction, not independently re-run by this author (this author verified
the underlying mechanism by code-reading instead — V20). (e) RESOLVED during the quorum round: the nested-block
follow-up issue was filed by the controller as `#614` on 2026-08-07, so it IS citable and M2's
DoD item is now "link it", not "file it"
here. (f) The claim that recursing the fold requires a failure-channel redesign (Known Residual
Vector, why-not-(i)) is design analysis grounded in V20's read of the sentinel construction, not
a measurement — no prototype of a recursive fold was attempted.

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

| Axiom | Score | Justification |
|---|---:|---|
| A1: Determinism | 0 | No evaluation-order or determinism change; short-circuit order matches #590's shipped contract |
| A2: Replayability | 0 | No change |
| A3: Effect Legibility | 0 | Effectful bodies were already rejected [V6]; unchanged |
| A4: Explicit Authority | 0 | No capability change |
| A5: Bounded Verification | +1 | Every TOP-LEVEL check the user writes is now enforced (was +2; re-graded in revision pass 1 — the nested-block residual [V19] means "a written-but-unenforced check ceases to exist as a category" was false, and this doc no longer claims it) |
| A6: Safe Concurrency | 0 | No change |
| A7: Machines First | +2 | Closes the top-level instance of a vacuous-pass class that lies to exit-code consumers (CI, eval harnesses) in the unsafe direction; the nested instance is named, pinned, and filed rather than silently left as a trusted green |
| A8: Minimal Syntax | +1 | No new syntax; the natural shape `{ a == 1; b == 2 }` gains its intuitive meaning |
| A9: Cost Visibility | 0 | No change |
| A10: Composability | 0 | No change |
| A11: Structured Failure | +1 | Non-final failures gain ordinal + source + original position via existing `CheckInfo` |
| A12: System Boundary | 0 | No change |

**Net Score: +5** (was +6 before revision pass 1; A5 re-graded +2→+1) → Proceed to sprint
planning.

### Hard Violation Check

- [x] A1: no hidden nondeterminism introduced; lowering is a pure syntactic transform.
- [x] A3: no hidden effects; effect discipline on bodies unchanged.
- [x] A4: no implicit authority.
- [x] A7: the fix is machine-checkable (fixtures + exit codes); failure direction moves from silent-pass to loud-fail.

## Related Documents

- [m-named-test-assert-lowering-sprint-plan](m-named-test-assert-lowering-sprint-plan.md) — #590:
  built the sentinel/`CheckInfo` machinery this design extends by one switch case; deliberately
  scoped #604 out [V3]. Distinction: #590 made `assert` execute at all; this item defines and
  enforces what bare expressions in the body MEAN.
- [M-NAMED-TEST-BLOCKS](../../implemented/v0_29_0/m-named-test-blocks.md) — v0.29.0 origin of the
  named-test execution path (and of the last-expression contract this design replaces).
- [docs/docs/guides/testing.md](../../../docs/docs/guides/testing.md) — to be updated in M2 [V12].

## Future Work

- **The nested-block follow-up (filed in M2, definition of done)**: close the residual vector —
  candidates are a recursive fold with a redesigned failure channel, a static error backed by an
  exhaustiveness-guaranteed AST walker, or a language-level lint on value-discarding pure
  `Block`s. Repros and decomposition sketch in Known Residual Vector.
- Line-level temp-file → user-source position mapping for pipeline errors in regenerated test
  bodies (generalizes M3's hint; benefits assert bodies too).
- Teach the canonical prompt (`ailang prompt`) multi-check named bodies once this ships, so
  models write `{ check; check }` bodies that actually mean what they say — the prompt update
  must state the top-level boundary, for the same trust-transfer reason as the testing guide.
