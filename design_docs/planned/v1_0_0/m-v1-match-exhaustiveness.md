# M-V1-MATCH-EXHAUSTIVENESS: Non-exhaustive `match` is a compile error, with a witness

**Status**: Planned
**Target**: v0.58.x (the last breaking window before the 1.x Stability Promise)
**Priority**: P0. Review §3.1(b); raises [#2146](https://github.com/sunholo-data/ailang/issues/2146) from P2
**Estimated**: 6–8 working days across 5 milestones. This is an estimate, not a plan (§12)
**Dependencies**: [m-v1-nominal-type-identity.md](m-v1-nominal-type-identity.md) M1 must land before M2 here (§8).
[m-v1-downstream-compat-gate.md](m-v1-downstream-compat-gate.md) supplies the downstream measurement for M4.
**Planner-Lane**: opus-required for M1–M2 (soundness core); codex-ok for M0, M3 and M4

**Related**: [v1-release-readiness-review-2026-11-09.md](../v1-release-readiness-review-2026-11-09.md) §3.1(b), §12 Q2 ·
[m_r3_pattern_matching.md](../../implemented/v0_2_0/m_r3_pattern_matching.md) (the original choice to warn) ·
[m-typecheck-no-auto-unwrap-result-sprint-plan.md](../../implemented/v0_20_0/m-typecheck-no-auto-unwrap-result-sprint-plan.md) (the post-inference verify pass this copies) ·
[m-bytecode-vm-parity-bugs.md](m-bytecode-vm-parity-bugs.md) (VM fallback replay) · `docs/LIMITATIONS.md` · `docs/docs/reference/stability.md`

## 1. Problem

A `match` that leaves out a case passes `ailang check` and fails only at run time. Repro, using the
HEAD binary `51f6636`:

```ailang
type Color = Red | Green | Blue
pure func name(c: Color) -> string = match c { Red => "r", Green => "g" }
```

`ailang check` prints `No errors found!`. Then `ailang run` fails with
`no pattern matched in match expression`, which carries no position. The rest of this section traces
the code that produces this.

**Where the only check lives.** `normalizeMatch` (`internal/elaborate/patterns.go:51-67`) runs the
checker during elaboration, before type inference. At that point it has no type information. So
`inferScrutineeType` (`patterns.go:246-266`) guesses a type from **literal** patterns only. Every
match on an ADT, list, tuple, record, Option or Result gets `nil` and is skipped. The checker itself,
`internal/elaborate/exhaustiveness.go`, is a stub with these problems:
- Lists, tuples and type variables all get a universe of "wildcard" (`:76-92`).
- `subtract` never removes anything from an infinite universe (`:145-148`).
- It names types `"Bool"`/`"Int"` (`:62-68`). The real constants are `bool`/`int`
  (`internal/types/types.go:453-457`), so the checker only works on the types the guess invents.

The typechecker has `// TODO: Implement full exhaustiveness checking` and copies the elaborator's flag
through unchanged (`internal/types/typechecker_patterns.go:18-20`).

**The warnings that do exist are invisible to agents.** For a literal match, `ailang check` prints
`warning: non-exhaustive match at lit1:2:33`, and the location is the module name, not the file path.
Both machine formats drop the warning:
- `check --json` returns `{"passed": true, "errors": []}`.
- `check --format agent` returns `ok: lit1.ail`.

Both were measured on `51f6636`. An AI author on either channel never sees the warning. That is
stronger evidence for the review's "AI authors do not read warnings": they *cannot*.

**The run-time failure is worse on the VM than the review says.**
- **Evaluator:** both failure sites raise an untyped `fmt.Errorf`: the linear arm loop
  (`internal/eval/eval_patterns.go:101`) and the `AILANG_DTREE` path (`internal/eval/decision_tree.go:114`).
  The code `RT002 "Pattern match failure"` is registered (`internal/errors/codes.go:370`) and unused.
- **Bytecode VM: no error at all.** `LowerMatchStmt` (`internal/gen/lower/match.go:23-31`) emits
  either a `SwitchStmt` with an empty `Default` or an if-chain with no final `else`
  (`match_ifchain.go:35+`). Control falls off the end and the match yields **Unit**. Measured:
  - `show(code(Blue))` dies later, inside `__string_intToStr: expected int`, at the *caller's* line.
  - `code(Blue) + 10` dies with `arith ADD: type mismatch Unit vs Int`. The run then prints
    `bytecode path unavailable … falling back to evaluator` and re-runs the program.

  Both are silent wrong values that are caught only by luck downstream. The VM is not the default
  engine, but `run --bytecode` is a supported path.

**Impact.** The language's pitch is that the checker makes AI-written code trustworthy. This is the
most common way a well-typed AILANG program crashes. The teaching prompt admits it:
`ailang prompt` v0.16.12 says *"`ailang check` does NOT catch it; it fails at run time"*.

## 2. Goals

**Primary goal:** every user-written `match` whose unguarded arms do not cover the scrutinee's type
is rejected by `ailang check`/`run`. The diagnostic gives a concrete missing pattern, file:line:col,
a stable code, and a fix-it. It is delivered through the human, `--json` and `--format agent` outputs.

**Success metrics:**
- The §1 repro gives `TC_MATCH_002 … missing: Blue` with `"line": 3` in `check --json`.
- Zero false positives on a differential test against the run-time matcher (§9 M1).
- `std/`, `examples/`, `tests/`, `benchmarks/` and the pinned downstream repos are green after M4.
- A run-time match failure on either engine is `RT002` with a source position, never Unit.
- The check adds ≤2% to `ailang check` wall time on the ailang-parse package (§7).

## 3. High-impact decisions

| Decision | Why high impact | Chosen by | Deadline | Change cost |
|---|---|---|---|---|
| Non-exhaustive match is an **error**, not a warning | Rejects programs that pass today. After 1.0, tightening needs 2.0 unless a soundness clause exists (review §3.1) | human (Mark, §11 Q1) | design | high |
| Guarded arms never count toward coverage | Fixes the semantics authors must learn. This is the OCaml/Rust/Swift rule | compiler | design | high |
| The check runs **after inference**, in `internal/types`, keyed on nominal ADT identity | Determines correctness under name collisions (§8) | agent, within this doc | design | med |
| Pure, bottom-typed `unreachable(msg)` ships alongside | Without it, an "impossible" arm gets a dummy value, and a loud crash becomes a silent wrong answer | human (§11 Q2) | compile | med |
| Redundant arm = **warning** `TC_MATCH_003` | A second breaking rule. It does not crash programs | human (§11 Q3) | compile | low |
| Literal-only matches (int/string/float) follow the same error rule | One rule, not two. Removes the elaborator's heuristic | human (§11 Q4) | design | low |

### Design freeze
- [ ] Error severity, with or without a 0.58.x escape hatch (§10, §11 Q1)
- [ ] `unreachable` accepted, plus its name and module (§11 Q2)
- [ ] Error-code numbers `TC_MATCH_002/003/004` and the message shape (§5)
- [ ] Ordering against nominal identity (§8)

### Deferred decisions (implementer's latitude)
- The internal representation of the pattern matrix, and whether column selection uses a heuristic.
- How many witnesses to list. The default is all missing top-level constructors, capped at 5,
  plus one nested witness.
- Whether multiple `TC_MATCH_002` errors in one module are returned joined or as the first only.
  The preference is all, so an agent can fix them in one pass. Follow whatever the Report plumbing
  supports.

## 4. The algorithm

Use **Maranget's usefulness algorithm** (*Warnings for pattern matching*, JFP 2007): `U(P, q)`
over a pattern matrix, with the standard *specialise* and *default* operations. It is the algorithm
OCaml and Rust use, and it gives both exhaustiveness and redundancy from one primitive. The two
uses:
- **Exhaustiveness:** the match is exhaustive iff `U(P_unguarded, [_])` is false. The
  witness-producing variant (Maranget §3.1, "computing missing patterns") returns the missing
  value as a pattern vector.
- **Redundancy:** arm *i* is unreachable iff `U(P_<i unguarded, [p_i])` is false.

The repo already has a matrix compiler in `internal/dtree` (`decision_tree.go:72-140`), but it is
disabled by default. It also has no list, tuple or record support and no guard backtracking
(`eval_patterns.go:31-36`). It is a decision-tree compiler, not a usefulness checker. **Do not build
on it.** A self-contained ~400-line package is smaller than fixing it.

### 4.1 Pattern forms, enumerated from the parser

The source is `internal/parser/parser_pattern.go:10-108`, `internal/ast/ast_patterns.go` and
Core `internal/core/core.go:346-420`. **There are no or-patterns and no as-patterns.** The parser
rejects `1 | 2 =>`. Separately, its own hint text suggests `match x { 1 | 2 => ... }`, which is a
small misleading-hint bug to file.

| Surface form | Core | Constructor signature used by `U` |
|---|---|---|
| `_`, `x` (lowercase identifier) | `WildcardPattern`, `VarPattern` | wildcard |
| `C`, `C(p1..pn)`, aliased import (canonicalised, `patterns.go:180-188`) | `ConstructorPattern{Name, TypeName}` | ADT's full constructor list with arities, from the **defining** module (§8) — never "constructors in scope" |
| `true`, `false` | `LitPattern{bool}` | finite: `{true, false}` |
| int / float / string literal | `LitPattern` | infinite: never complete, so a wildcard (or variable) is required |
| `[]`, `[p1..pk]`, `[p1..pk, ...r]`, `h :: t`, `::(h, t)` | `ListPattern{Elements, Tail}` | `Nil/0 \| Cons/2`. `[a, b]` = `Cons(a, Cons(b, Nil))`; `[a, ...r]` = `Cons(a, r)` |
| `(p1, …, pn)` | `TuplePattern` | a single constructor of arity n |
| `()` | unit | a single nullary constructor |
| `{f: p, g}`, `{f: p, ...}` | `RecordPattern{Fields, Rest}` | a single constructor whose fields are the union of the fields named in that column. An unnamed field counts as a wildcard. Open versus closed changes typing, not coverage |
| `p if cond` | `MatchArm.Guard` | **the row is dropped** for exhaustiveness. It is still checked for its own reachability |

A column's signature comes from its non-wildcard heads. The program is already well typed, so
every head in a column belongs to one type. The scrutinee's resolved type from `CoreTI` is used for
two things: to cross-check the root column, and to name the type in the message. A mismatch between
`patternADT(head)` and the `CoreTI` type is an **internal error**, not a silent skip.

### 4.2 Where it runs

Add `(tc *CoreTypeChecker) VerifyMatchExhaustiveness(prog *core.Program) error` and call it right
after `FinalizeSubstitutions()`, next to the existing post-inference gate
`VerifyTaggedUnionFieldAccesses()` (`internal/pipeline/pipeline_module_compile.go:210-218`). It needs
the equivalent call in the single-file/REPL path (`pipeline_single.go`, after its type check).

The check walks Core, not TypedAST. `applySubstitutionToTyped` does not descend into `TypedMatch`
(`typechecker_substitution.go:259-301`, `default: return node`), so a typed match's scrutinee type
can be a stale type variable. `CoreTI` is finalised. ANF guarantees an atomic scrutinee
(`patterns.go:16`), so `tc.CoreTI.Get(m.Scrutinee.ID())` is always defined.

The walk runs before dictionary elaboration, specialisation and op-lowering. Those passes only
*copy* matches (`grep '&core.Match{'`: `specialize_expr.go:262`, `op_lowering.go:187`,
`dictionaries.go:285`, `show_normalize.go:246`). So only user-written matches are seen. The parser
is the only producer of `ast.Match`.

**Pipeline pass coverage:**
- [ ] `prog.Decls`
- [ ] `prog.Meta[*].Contracts[*].Expr` (type-checked later, `pipeline_module_compile.go:238-280`; run the walk after that block)
- [ ] `test` / `tests [...]` bodies (confirm they are in `Decls`; the M2 test must prove coverage)
- [ ] the single-file/REPL pipeline

**Delete** `internal/elaborate/exhaustiveness.go`, its test, `inferScrutineeType`, and the
elaborator's `warnings []*ExhaustivenessWarning` plumbing. Keep the `elaborate.Warning` interface
in `warnings.go`, which other passes use. Then set `core.Match.Exhaustive` authoritatively from the
new pass. That unlocks a later SMT simplification: `smt/codegen_control.go:76-83` skips constructor
matches today *"because exhaustiveness is only a warning"*. That simplification is a non-goal here.

## 5. Diagnostics

The codes follow the existing `TC_MATCH_001` (`typechecker_ctor_scope.go:15`).

| Code | Severity | When |
|---|---|---|
| `TC_MATCH_002` | **error** | a match is not exhaustive |
| `TC_MATCH_003` | warning | an arm can never be reached (redundant) |
| `TC_MATCH_004` | error | the per-match step budget was exceeded (§7). It never fired in the census |
| `RT002` | runtime | no arm matched (defence in depth, §6) |

Human output:

```
Error: p1.ail:3:38: TC_MATCH_002: non-exhaustive match on Color — no arm matches: Blue
  Suggestion: add the missing arm:  Blue => ...
  If this case cannot happen, say so:  Blue => unreachable("<why it cannot happen>")
  Note: arms with `if` guards do not count toward coverage.
```

Witnesses are rendered in surface syntax that `ailang check` accepts:
- constructors: `Blue`, `Some(Err(_))`
- lists: `[]`, `[_, ..._]` (verified: a spread must bind a name, `PAT_SPREAD_NEEDS_IDENT`; `..._` parses)
- tuples: `(false, false)`
- records: `{a: false, ...}`

For an infinite literal domain the witness is `_`, followed by *"(any int other than 0, 1)"*,
listing at most five literals.

`check --json` builds each entry with `errorToCheckJSONError` (`cmd/ailang/check_output.go:111-131`).
It fills `code/line/column/suggestion` only for an `ailerrors.Report`. Even `TC_MATCH_001` falls
through to `"code": "ERROR"` today, with no line (measured). `TC_MATCH_002` **must** be a `Report`
with `Span` and `Fix.Suggestion`:

```json
{"code": "TC_MATCH_002", "file": "p1.ail", "line": 3, "column": 38,
 "message": "non-exhaustive match on Color — no arm matches: Blue",
 "suggestion": "add the missing arm: Blue => ..."}
```

`--format agent` then prints
`p1.ail:3:38 TC_MATCH_002: non-exhaustive match on Color — no arm matches: Blue → add the missing arm: Blue => ...`.

`TC_MATCH_003` is only useful if warnings reach agents. M3 adds a `warnings` array to `check --json`
and a `warn:` line to `--format agent`. Both are additive to a Stable command (§11 Q3).

## 6. Run-time defence in depth

Once guarded arms stop counting, **a checked program cannot reach a no-match at run time.** That
corrects the framing that it stays "reachable via guards". The run-time error then exists to catch
three things:
- checker bugs;
- the cross-module ADT unsoundness (§3.1a) until it is fixed;
- any path that bypasses the check.

The design follows the RT001 precedent: `errors.DivByZeroError{Pos}` (`internal/errors/arith.go`),
with position attached by `internal/eval/div_zero_pos.go`.

- **Evaluator:** add `errors.MatchFailureError{Pos, Tag}` with code `RT002`. Raise it at
  `eval_patterns.go:101` and `decision_tree.go:114`, with `Pos` taken from `match.OriginalSpan()`
  (file:line:col). Message: `RT002: no arm matched in match at p1.ail:3:38 (scrutinee constructor: Blue)`.
  Show the constructor tag or list shape (`empty list`) only, **never the payload**. Scrutinees can
  be IFC-labelled secrets, and trace redaction exists for a reason.
- **VM:** `LowerMatchStmt` appends a trap wherever the arms are not closed by an unguarded
  catch-all: the `SwitchStmt.Default`, and the final `else` of the if-chain. The trap is a new
  `stmt.MatchFail{Line, Col}` (stmt already carries `Line`, `stmt.go:142`). It compiles to either
  - an `OpMatchFail` opcode, or
  - an `OpBuiltinTrap` into a `_match_fail` builtin,

  and either raises the same `RT002` with file:line (the VM's precision, as for RT001). It
  **must not** yield Unit.
- **Go codegen** already emits `panic("non-exhaustive match")` (`gen/golang/codegen_match*.go`).
  Add the position to that string only.

M0 below is non-breaking and can ship first, independently of everything else.

## 7. Performance

Usefulness is exponential in the worst case. In practice it is near-linear.
- **Arms per match (census, §10):** median 2, p99 8, maximum 97 (`email-parse attachments.ail:393`, string literals).
- **Speed:** the Python census prototype parses and checks 60,156 matches in about 10 s.

Budgets, each with a benchmark test:
- ≤2% added to `ailang check` wall time on the ailang-parse package.
- <10 ms for a synthetic 1,000-arm string-literal match.
- <100 ms for a 12-column bool-tuple match with 2^10 rows.
- A hard cap of 10^6 specialisation steps per match. Exceeding it gives `TC_MATCH_004` with the
  fix-it *"add a final `_ =>` arm"*. That is fail-closed. A silent pass would break the repo's
  no-silent-fallback rule.

## 8. Interactions and ordering

- **Nominal type identity ([sibling doc](m-v1-nominal-type-identity.md)).** The check needs, for
  an ADT *identity*, the full ordered constructor list with arities. Today every registry is keyed
  by **bare** names:
  - `constructorTypes` holds only *in-scope* constructors (`pipeline_module_compile.go:92-111`).
    Using it would wrongly call `match e { Message(_) => … }` exhaustive when the user imported one
    constructor.
  - `AllCtorTypes` is transitive, but collapses colliding names to `""`
    (`pipeline_module_imports.go:86-99`).

  Both are the §3.1a defect. **Ordering:** M1 (the pure algorithm) can start now. M2 (wiring plus
  the universe registry `adtUniverse[identity] → []CtorSig`, built from local declarations and every
  loaded iface's `Constructors`) waits for nominal-identity M1, and uses its identity key. If Mark
  overrides that order, M2 keys on the bare name and **fails closed** on an ambiguous name, giving
  `TC_MATCH_002` with "ambiguous type name; add `_ =>`". It must not skip, and a test must go red
  when nominal identity lands.

  Constructor order for witnesses is declaration order. `iface.ConstructorScheme` has no ordinal
  today, so add one, or sort by name, but never by map order.
- **`Json` gaining a constructor (`JInt`, review §3.3).** Only **1** match in the corpus covers
  `Json` explicitly with no catch-all (motoko). std's three have catch-alls. Either order works:
  if `JInt` lands after this, that match fails with the precise witness `JInt(_)`, which is the
  intended behaviour. Both must land inside 0.58.x.
- **`pure` implies empty row (#2184) and `unreachable`.** A pure `unreachable[a](msg: string) -> a`
  diverges like `0 / 0` already does. `std/result.unwrap` uses `0 / 0` as its abort today
  (`std/result.ail:49-53`), and `unwrap` should switch to it. No `panic`/`abort`/`unreachable`
  exists: `ailang builtins list` has 407 entries and only `_io_exit` (`! {IO}`) comes close.

## 9. Milestones

Sizes are estimates for sprint-planner.

**M0: run-time position and VM trap (0.5–1 d, non-breaking).** `RT002` with position on both
evaluator sites. The VM trap from §6.

Tests:
- One `match` per engine (`run` and `run --bytecode`) that asserts `RT002` and `file:line`.
- One VM test that asserts no Unit value escapes the match.

Mutation check: remove the trap and the VM test must observe the old `type mismatch Unit` failure.

**M1: algorithm package (2 d).** `internal/types/exhaust/`, pure, with no pipeline imports. It
contains the matrix, `U`, the witness, redundancy, and the step budget, behind a `Universe`
interface (`CtorsOf(head) []CtorSig`).

Tests:
- A table test per row of §4.1, covering the negative and the positive case.
- Nested `Some(Err(_))`.
- Guarded-only arms.
- Fixed-length lists.
- Records with disjoint field subsets.

Then a **differential property test**:
- Generate random small ADTs (depth ≤3, ≤4 constructors, including lists, tuples and bools).
- Generate random arm sets.
- Enumerate every value up to depth k.
- Assert that the checker says exhaustive ⇔ every value is matched by the evaluator's own
  `matchPattern` (`eval_patterns.go`).
- Assert that every witness, instantiated, is matched by no arm.

Mutation checks:
- Count guarded rows as covering: the guard test must fail.
- Make literals finite: the int test must fail.

**M2: wiring and error (1.5–2 d, needs nominal M1).** The universe registry,
`VerifyMatchExhaustiveness` in both pipelines with contract and test coverage, the `TC_MATCH_002`
`Report`, and deleting the elaborator checker and its tests.

Tests:
- The §1 repro in `check`, `check --json` (code, line, column) and `--format agent`.
- The same through `run`.
- A cross-module case: the match is in module B, the ADT is imported from A, and B imports a
  **single** constructor (this guards against the in-scope registry trap).
- A contract-body match.
- A test-body match.
- `make test`, `make verify-examples`, `make test-imports`.

Mutation check: make the pass `return nil`, and the repro test must fail.

**M3: redundancy warning and warnings in machine output (1 d).** `TC_MATCH_003`, plus the `warnings`
array in `check --json` and agent output, if §11 Q3 is yes.

Test: `Committed(n) => …, other => …, Idle => …`, from `tests/golden/bytecode/nested_patterns.ail:301`,
warns on `Idle`.

**M4: migration (1–1.5 d).**
- Fix every repo site.
- Add `unreachable`, if approved.
- Update the teaching prompt through prompt-manager, plus the docs, LIMITATIONS and the changelog
  fragment.
- Run the downstream compat gate against the candidate. Each downstream fix ships as a one-line PR
  in that repo.

Teaching prompt: replace the line at `ailang prompt` v0.16.12 *"`ailang check` does NOT catch it…"*
with *"`ailang check` rejects a `match` that misses a case (`TC_MATCH_002`) and names the missing
pattern. Add that arm, or `_ => …`, or `C => unreachable("why")` when the case cannot happen. Arms
with `if` guards do not count."*

## 10. Blast radius, measured

**Instrument.** A static census prototype of §4 at
`/private/tmp/claude-501/-Users-voightkampff-dev-sunholo-data-ailang/795df2bf-8bbb-44ec-86b9-f8166df96f9c/scratchpad/dd-exhaust/census.py`
(scratch, not in the repo). It does the following:
1. Lexes `.ail` files.
2. Extracts each `match` and splits its arms on `,` **or newline**, both of which the parser
   accepts.
3. Drops guarded arms.
4. Runs usefulness, with constructor sets taken from `type` declarations across the whole corpus,
   preferring the local file.

It does **not** infer types. It resolves an ADT by constructor name and flags ambiguity. It was
validated on 11 hand cases, all correct. The repos were read only, with no git operations; motoko
is `~/dev/mk-main`.

| Corpus | files with a match | matches | undecided* | **non-exhaustive** | (of which literal-only, warned today) | redundant arm |
|---|--:|--:|--:|--:|--:|--:|
| ailang `std/` | 18 | 139 | 0 | **0** | 0 | 0 |
| ailang `examples/` | 245 | 759 | 12 | **1** | 1 | 1 |
| ailang `benchmarks/` | 22 | 317 | 0 | **0** | 0 | 0 |
| ailang `tests/` | 29 | 159 | 19 | **0** | 0 | 1 |
| ailang-parse | 73 | 1,058 | 88 | **0** | 0 | 0 |
| email-parse | 58 | 527 | 12 | **0** | 0 | 0 |
| daneel | 66 | 741 | 5 | **1** | 0 | 0 |
| ailang-packages | 123 | 1,358 | 5 | **0** | 0 | 0 |
| world | 57 | 762 | 0 | **0** | 0 | 0 |
| motoko (`mk-main`) | 370 | 5,327 | 254 | **4** | 0 | 5 |
| **AI-written, passing eval runs** (`eval_results/**`, `lang=ailang`, `stdout_ok`, deduplicated) | 10,166 | 49,009 | 125 | **359** | 31 | 96 |

\*Undecided means an unparsed match, or a constructor whose declaration the census could not
resolve. It is about 3% of the non-eval matches.

**Hand classification, non-eval (all 6 read).**
- **Real latent crashes:**
  - `daneel tools/video/daneel_video.ail:231`: the `StreamEvent` match omits `SSEData`, `SourceText`
    and `SourceBytes` (`std/stream.ail:53`).
  - `mk-main src/core/test/scripted_ports.ail:31`: covers 3 of `StepProvider`'s 7 constructors.
- **Invariant-protected:** `mk-main scripts/dst/stream_parity_dst.ail:123`, where an inner
  `b :: _` is safe only because an outer arm handled `[]`.
- **Test code:** two `progress_contract_guard.ail` matches (motoko, lines 233 and 237).
- **Intentional:** `examples/tests/test_exhaustive_bool_incomplete.ail`, which becomes a negative
  fixture.

Downstream cost: **6 one-line edits across 2 repos**.

**AI-written code is where this bites.** 150 of 10,166 passing programs (1.5%) contain a silently
non-exhaustive match. They are concentrated in 31 of 85 benchmarks: `binary_strings_1e18` 28/70,
`decision_block_capture` 20/192, `glob_match_spec` 16/166. The witness kinds are:

| Witness kind | Matches |
|---|--:|
| list | 282 |
| `_` | 31 |
| `None` | 14 |
| tuple | 11 |

In the 18 cases read by hand:
- The census matched a real gap every time, so there were no false positives from the census itself.
- Most gaps rely on an invariant: a list used as a fixed 9-element matrix, a stack machine that
  assumes depth, or a constant scrutinee like `match Leaf { Leaf => … }`.
- Some are real crashes on untested input, such as `maximum` over `[]`.

So under an error, about 1.5% of passing single-shot AILANG runs would fail to compile, up to 40% on
one benchmark. Agent-mode runs get the witness and fix-it in the repair loop. **Estimated, not
measured:** the net pass-rate effect. Measuring it is a paid paired eval (§11 Q5).

The teaching prompt's 64 `ailang` code blocks contain **0** non-exhaustive matches. One `Tree`
example is undecided for lack of a type declaration.

**Not verified:** behaviour on files the census could not resolve; docparse (`ailang-parse` was
used as the current repo); the other `mk-*` checkouts; packages outside these trees.

## 11. Decisions for Mark

1. **Error or warning?** Recommended: **error, from the first 0.58.x that ships this, with no escape
   hatch.**
   - A warning phase is worthless to agents, because warnings are dropped from `--json` and agent
     output (§1).
   - The downstream cost is 6 edits.
   - An opt-out flag would be one more surface to retire before 1.0. Precedent if you want one
     anyway: the `allowUnsafeFieldAccess` migration flag (`tagged_union_predicate.go:133`), plus a
     config-registry entry (the forbidigo rule), removed at 1.0.
2. **Add `unreachable(msg: string) -> a`?** Recommended: **yes**. Put it in `std/prelude` or the
   builtin module, mark it Stable, and make `std/result.unwrap` use it. Without it, the fix-it
   steers authors toward dummy values.
3. **Redundant arms:** recommended: **warning `TC_MATCH_003`**, and expose warnings in `check --json`
   and `--format agent` (additive).
4. **Literal-only int/string matches become errors too?** Recommended: **yes**. One rule; 31 eval
   programs and 1 example are affected.
5. **A paired eval on the 31 affected benchmarks**, standard mode, error on versus off, to measure
   the pass-rate cost. It is real spend (about 5× Python per run). Optional; the change is right
   either way.
6. **Soundness clause** in `stability.md` (review §3.1). This doc assumes it is adopted, so a later
   checker hole can be closed in 1.x.

## 12. Non-goals, risks, effort

**Non-goals:**
- or-patterns and as-patterns (none exist);
- range or refinement reasoning on ints ("`n < 0`, `n >= 0` covers int") — guards never count;
- flow-sensitive exhaustiveness (the outer-arm invariant case);
- SMT encoding of constructor matches;
- the VM's evaluator-fallback replay ([m-bytecode-vm-parity-bugs.md](m-bytecode-vm-parity-bugs.md) B4);
- the misleading or-pattern parser hint (file separately).

**Risks:**
- **The universe comes from the wrong registry.** A false "exhaustive" when only some constructors
  are in scope. Mitigation: the M2 cross-module single-import test.
- **Witness rendering drifts from parseable syntax.** Mitigation: an M1 test runs each rendered
  witness through the parser.
- **AI eval pass rates dip about 1.5%** (§10). Mitigation: the fix-it, the prompt line and
  `unreachable`.
- **M2 slips behind nominal identity**, which is itself in the breaking window. M0, M1 and M3 do not
  wait.

**Effort, as an estimate:** M0 0.5–1 d · M1 2 d · M2 1.5–2 d · M3 1 d · M4 1–1.5 d, so **6–8 working
days**, plus 0.5 d for `unreachable` if approved.

## 13. Migration

**What downstream authors change:**
- Add the arm the error names.
- Or end with `_ => …`.
- Or, for a case that truly cannot happen, `C => unreachable("why")`.
- A guarded arm needs an unguarded fallback after it.

**Docs:**
- Remove the "check does not catch it" row from the teaching prompt, through prompt-manager (§9 M4).
- Add `TC_MATCH_002`, `TC_MATCH_003`, `TC_MATCH_004` and `RT002` to `docs/docs/reference/errors/index.md`.
- Add a "Pattern matching: exhaustiveness" paragraph to the language reference.
- `docs/LIMITATIONS.md` currently has no row for this: add none, and note in the changelog that the
  prompt caveat is gone.
- Add `examples/match_exhaustiveness.ail`, covering the positive case and `unreachable`, and turn
  `test_exhaustive_bool_incomplete.ail` into an expected-failure fixture.

**Changelog fragment draft** (`changelogs/unreleased/2026-11-NN-match-exhaustiveness.md`):

```markdown
### Changed — BREAKING: a `match` that misses a case is now a compile error (TC_MATCH_002)

`ailang check` and `ailang run` reject a `match` whose unguarded arms do not cover every value
of the scrutinee's type — ADTs, Option/Result, lists, tuples, records, bool, and int/string
literals without a catch-all. The error names a concrete missing pattern (`Blue`,
`Some(Err(_))`, `[]`) with file:line:col in human, `--json` and `--format agent` output. Arms with
`if` guards do not count toward coverage. Fix: add the named arm, end with `_ => …`, or write
`C => unreachable("why")` for a case that cannot happen. Previously these programs passed `check`
and failed at run time with an unlocated "no pattern matched" — or, on `--bytecode`, produced a
Unit value and failed later elsewhere. Run-time match failure is now `RT002` with a source
position on both engines. Design: design_docs/planned/v1_0_0/m-v1-match-exhaustiveness.md (#2146).
```
