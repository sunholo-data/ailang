# M-SYNTAX-AI-FORGIVING: Forgiving statement syntax — accept the AI's newline-separator prior

**Status**: Planned
**Target**: v0.24.0
**Priority**: P1 — eliminates the single largest small-model failure *class* (~32%) with a localized parser change; helps every model, not just a fine-tuned one
**Estimated**: ~3 days (R1: 1 day; R2: 1.5 days; `ailang fmt` canonicalization + eval round-trip: 0.5 day)
**Dependencies**: None. Complements M-AILANG-ERROR-QUALITY (which made these errors *actionable*; this removes the error class entirely). A strictly cheaper, broader alternative to the first iteration of M-EVAL-FINETUNING-DATA-PIPELINE for the dialect-adherence portion of the gap.

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Surface-only relaxation. Same AST → same result; no execution or trace nondeterminism. `ailang fmt` normalizes presentation (see Risks). |
| A2: Replayability | 0 | No impact on traces. |
| A3: Effect Legibility | 0 | No change to effect rows or declarations. |
| A4: Explicit Authority | 0 | No change to capabilities. |
| A5: Bounded Verification | 0 | Type checker and elaborator are untouched; only the parse of an already-unambiguous form changes. |
| A6: Safe Concurrency | 0 | No concurrency surface. |
| A7: Machines First | **+1** | This *is* the axiom in action — it removes a parse-failure class that costs ~32% of small-model synthesis attempts, optimizing AILANG for autonomous code generation rather than forcing the generator to learn an idiosyncratic separator rule. |
| A8: Minimal Syntax | **+1** | Counter-intuitively positive: today's rule is *non-minimal* — `;` is mandatory in `{ }` blocks, forbidden in `=` bodies, and newlines are ignored, so the same intent has different validity depending on body form. Unifying to "statements are separated (by `;` or newline), uniformly" is a *simpler* mental model with fewer special cases. (Debatable: it adds surface redundancy; the `ailang fmt` canonical form is the mitigation — see Risks.) |
| A9: Cost Visibility | 0 | No resource-cost surface. |
| A10: Composability | 0 | No effect/module composition change. |
| A11: Structured Failure | 0 | Errors that remain (genuine syntax errors) keep their structure; this just stops emitting them for a form that has a single unambiguous meaning. |
| A12: System Boundary | 0 | No boundary crossing. |

**Net Score: +2** → **Decision: Move forward.**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism — same AST, same trace; `ailang fmt` keeps presentation deterministic.
- [x] A3 (Effects): No hidden side effects.
- [x] A4 (Authority): No ambient access.
- [x] A7 (Machines First): **+1** — the feature is defined by this axiom.

## Problem Statement

AILANG's lexer **consumes newlines** (`skipWhitespace`; there is no `NEWLINE` token — see `.claude/rules/parser.md`) and the parser requires `;` between statements. *Every* language LLMs are trained on at scale — Python, Go, JavaScript, Swift, Kotlin, Rust — treats a **newline as a statement separator**. So a model's natural output is one delimiter away from valid AILANG, and that single design choice trips it in **both directions**.

**Current state — measured.** Across a 334-trial corpus of `opencode-qwen3-5-35b-a3b-mxfp8` agent runs (6 nightly runs + the first core baseline, 2026-06-07…06-12), the `;`/block-separator confusion is **~32% of all failures**, the single largest class:

| Signature | % of failures |
|-----------|--------------:|
| `;` in expression-body (`PAR017`) — `func f() = let x = e; rest` | **20.5%** |
| block vs expr-body `{}` confusion (incl. *missing* `;`, `PAR020`) | **11.4%** |
| (next: logic errors 15.9% — capability, not addressable here) | — |

**Near-miss proof (verified with `ailang check`, 2026-06-12):** the model has the right *intent*; it just omits the delimiter AILANG demands.

```
func g() -> int = let x = 5; x + 1        → ❌ PAR017 (rejected)
func g() -> int = { let x = 5; x + 1 }    → ✅ No errors found!   (+ braces)
func g() -> int { let x = 5; x + 1 }      → ✅ No errors found!   (− the =)
```
```
func g() -> int {        → ❌ PAR020 ("missing ;") — newline-separated, no ;
  let x = 5
  x + 1
}
```

**Impact:** AI models are AILANG's *primary* consumers. We have already spent the M-AILANG-ERROR-QUALITY iteration making `PAR017`/`PAR020` maximally actionable and sharpening the dialect-traps card — yet the model **still** makes these mistakes (4× `PAR017` despite the card in the 2026-06-12 run). That is the signal that the lever is no longer *information*: the model is told the rule and breaks it anyway, because the rule fights a near-universal prior. This doc proposes removing the rule that fights it.

## Goals

**Primary Goal:** Make AILANG *accept the statement-separator forms the model naturally writes*, eliminating the `;`/block-confusion parse-failure class without changing the model.

**Success Metrics:**
- The `;`-family benchmarks (`config_file_parser`, `csv_to_json_converter`, and the near-miss fixtures) **parse** on the natural form (no `PAR017`/`PAR020`).
- A controlled A/B on the rig (old vs new parser, the `;`-family benchmarks, 5 trials each, 1M-token cap) shows a **measurable reduction in `compile_error` rate** on those benchmarks — the cluster is ~32% of failures, so even partial elimination is a real pass-rate lift.
- **Zero regression** on `make verify-examples` (181/5/2 baseline) and the full Go suites.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Accept newline as a statement/decl separator (R2) | Touches the core "newlines are insignificant" invariant in `.claude/rules/parser.md`; ripples to every block-parsing site | human | design | high |
| Accept `;`-sequences in `=` function bodies (R1) | Extends the `=`-body position, which collides with anonymous-function `func(...)` expressions | human | design | med |
| Canonical form via `ailang fmt` to preserve presentation determinism | Whether "two forms accepted" is acceptable depends on having one *canonical* normalizer | human | design | med |
| `func IDENT` vs `func (` disambiguation for the R1 decl-boundary | Language-semantics call; must be proven sufficient (cf. M-TAINT 2-token-lookahead regression) | compiler | compile | med |

### Design Freeze

Before implementation begins, these must be resolved:

- [ ] **Approve the newline-as-soft-separator direction** (R2) — it relaxes the long-standing "newlines insignificant" invariant. This is the load-bearing decision.
- [ ] **Approve accepting both `;` and newline** vs choosing one canonical separator — and confirm `ailang fmt` will normalize to a single canonical form so the corpus/examples stay uniform.
- [ ] **Confirm the R1 decl-boundary disambiguation** (`func IDENT` = declaration, `func (` = anonymous-function expression; `export`/`type`/`import`/EOF = hard boundaries) is sufficient — see Conflict Surface.

## Solution Design

### Overview

Two **independent, backward-compatible** relaxations. Existing `;`-delimited code is byte-for-byte unaffected; both only *accept additional* inputs that today error. They can ship and be measured separately.

- **R1 — `;`-sequences in `=` bodies.** Parse `func f(...) -> T = s1; s2; e` as if it were `func f(...) -> T { s1; s2; e }`. Kills `PAR017` (20.5%).
- **R2 — newline as a soft separator in blocks.** Inside `{ ... }`, treat "the next token begins on a later source line" as an implicit `;` between statements, *in addition to* explicit `;`. Kills `PAR020` / block-confusion (11.4%).

### Architecture

**R1 (parser only, localized).** In `parseFunctionDeclaration` (`internal/parser/parser_func.go`), the `=` branch currently calls `parseExpression(LOWEST)` once. Replace with a statement-sequence parse that consumes `expr (; expr)*` and **stops at a declaration boundary**: the peek token is `export`, `type`, `import`, EOF, or `func` *followed by an identifier* (a named declaration). It does **not** stop at `func (` — that is an anonymous-function expression continuing the body. The collected statements are wrapped in the existing `ast.Block` (the same node `{ }` produces), so elaboration/typing are unchanged.

**R2 (parser, line-aware).** The lexer already records `Line`/`Column` on every token. In the block-statement loops (`parseBlockOrExpression` in `parser_expr.go` and `parseFunctionBody`/the `{ }` body parse in `parser_func.go`), after a complete statement, treat the loop as continuing if `peek` is `;` **or** `peek.Line > cur.Line` **and** `peek` starts a statement (reuse `peekStartsBlockStatement`). An explicit `;` still works. The `PAR020` actionable error (M-AILANG-ERROR-QUALITY) becomes a fallback for the genuinely-malformed case (e.g., two statements jammed on one line with no `;`).

### Implementation Plan

**Phase 1: R1 — `=`-body statement sequences** (~1 day)
- [ ] `parseFunctionDeclaration` `=` branch: parse `expr (; expr)*` until decl-boundary; wrap in `ast.Block`.
- [ ] Decl-boundary helper: `peekIsDeclBoundary()` = `export|type|import|EOF | (func && peek2 is IDENT)`.
- [ ] Tests: the near-miss fixture compiles; multi-statement `=` body; `=` body immediately followed by `func g`, `export func`, `type`, EOF; anonymous-function in `=` body still parses (`func f() = map(func(x) -> int { x+1 }, xs)`).

**Phase 2: R2 — newline-as-soft-separator in blocks** (~1.5 days)
- [ ] `parseBlockOrExpression` + function-body block loop: continue on `;` OR (`peek.Line > cur.Line` AND `peekStartsBlockStatement()`).
- [ ] Keep `PAR020` for the same-line no-`;` case (genuine error).
- [ ] Tests: newline-separated block statements; mixed `;` + newline; record literals `{a: 1\n b: 2}` still parse as records (Conflict Surface); single-expression blocks unaffected.

**Phase 3: canonicalization + measurement** (~0.5 day)
- [ ] `ailang fmt`: normalize to the canonical form (decide: `;`-on-one-line vs newline-per-statement). Add a golden test that `fmt` is idempotent and maps all three accepted forms to one output.
- [ ] A/B on the rig (old vs new parser, `;`-family benchmarks, 5 trials, 1M cap) — record `compile_error` Δ.
- [ ] `make verify-examples` unchanged; full Go suites green.

### Files to Modify/Create

**Modified:**
- `internal/parser/parser_func.go` — R1 `=`-body sequence parse + decl-boundary helper (~40 LOC)
- `internal/parser/parser_expr.go` — R2 newline-soft-separator in `parseBlockOrExpression` (~15 LOC)
- `internal/parser/par020_block_semicolon_test.go` + new `syntax_ai_forgiving_test.go` — fixtures for both relaxations + Conflict Surface fixtures (~120 LOC)
- `cmd/ailang/fmt*.go` (or the formatter) — canonicalization (~40 LOC)
- `prompts/agent/dialect-traps.md` — once shipped, *remove* trap #2/the `;` guidance (the rule it warns about no longer exists), shrinking the card
- `changelogs/v0.10-current.md`, `docs/LIMITATIONS.md`

## Conflict Surface

**(Required: this touches `internal/parser`, `internal/ast`.)**

1. **What positions does this extend?**
   - R1 extends the `=` function-body position from "exactly one expression" to "a `;`-separated statement sequence."
   - R2 extends the in-block separator set from `{;}` to `{; , newline}`.

2. **What else lives in those positions?**
   - **`=` body (R1): anonymous functions.** Verified: `let f = func(x: int) -> int { x + 1 }` runs — `func(...)` is a valid *expression*. So a `func` token after the body is **ambiguous**: new declaration vs anonymous-function continuation. **Resolution:** `func IDENT` (name follows) = declaration boundary; `func (` = anonymous-function expression (stays in the body). This is the existing decl-vs-funclit distinction; R1 must reuse it, not invent a new one.
   - **`=` body (R1): `export`/`type`/`import`.** Verified: `let x = export …` → `PAR_` (cannot start an expression). These are **safe hard boundaries**.
   - **Block separator (R2): record literals.** `{ a: 1, b: 2 }` and `{ base | f: v }` are detected up-front in `parseBlockOrExpression` (IDENT/STRING `:` or IDENT `|`) *before* the statement loop, so the newline rule never reaches record bodies. Fixture required: `{ name: "a"\n  age: 2 }` must still parse as a **record**, not two statements.
   - **Block separator (R2): match arms.** Match arms are comma-separated *inside* `match x { ... }`, parsed by the match parser, not the block-statement loop — out of scope. Fixture: a multi-line `match` inside a block must still parse.

3. **How does the parser disambiguate?** R1: peek-1 = `export|type|import|EOF`, or peek-1 = `func` AND peek-2 = `IDENT` → declaration boundary (end body); otherwise continue. R2: record/record-update detection stays *ahead* of the statement loop; the newline rule applies only once we are in statement-sequence mode.

4. **Which existing programs MUST still work (fixtures)?**
   - `func f() -> int = x * 2` (single-expr `=` body) — unchanged.
   - `func f() -> int { let a = 1; let b = 2; a + b }` (explicit `;` block) — unchanged.
   - `let f = func(x: int) -> int { x + 1 } in f(5)` (funclit expression) — unchanged.
   - `func f() -> int = g(func(x) -> int { x + 1 }, xs)` (funclit *inside* an `=` body) — must NOT be cut at `func(`.
   - `{ name: "a", age: 2 }` and a multi-line record — must parse as a record.
   - Two declarations back-to-back with no blank line: `func f() = 1` newline `func g() = 2` — `f`'s body must end at `func g`.

5. **What deliberately changes (intentional incompatibilities)?** Forms that previously errored (`PAR017`, `PAR020` for newline-separated blocks) now parse. No previously-*valid* program changes meaning. `ailang fmt` output for affected files changes to the new canonical form (a formatting-only diff).

### Verified empirically (2026-06-12, clean dev binary v0.24.2-47-g66cc6629)

The load-bearing safety assumptions were checked with live `ailang run`/`check`, not asserted:

- **No juxtaposition application.** `let g = \x. x+1; g 5` → `PAR_UNEXPECTED_TOKEN: expected }, got INT`. Application is `f(args)` via an **infix `LPAREN`** (`parser.go:118 registerInfix(LPAREN, parseCallExpression)`), never `f x`. Therefore `expr⏎IDENT` can **never** be a continuation — it is unambiguously two statements. This is the single assumption R2's IDENT-trigger depends on. *(Honesty note: an earlier run against a stale A/B binary falsely showed `g 5` compiling; a clean rebuild settled it — always verify against a clean build.)*
- **Decl keywords are not expression-starts.** `let x = export` / `type` / `import` all → `PAR_`. Safe R1 hard boundaries.
- **Funclits survive R1.** `map(func(x) -> int { x*2 }, xs)` and `map(\x. x*2, xs)` inside an `=` body both compile; `func (` ≠ `func IDENT`.
- **Operator line-continuation is preserved.** `= 1⏎+ 2` and `let x = 1 +⏎2` both compile — operators are not in `peekStartsBlockStatement`, so R2 never splits them.
- **Precedent — the parser is *already* newline-aware.** `internal/parser/gap2_regression_test.go` (`TestGAP2_NewlineLPAREN`) shows `expr⏎(next)` is deliberately **not** parsed as a call — the infix loop already breaks on a newline-before-`LPAREN`. R2 is therefore an *extension* of an established mechanism, not a new invariant.

**Net of verification:** R1 is provably narrow — it activates only on the `;`-in-`=`-body form that hard-errors today, so it cannot change the meaning of any currently-compiling program. R2 is safe on every collision axis tested **and** extends an existing rule, but newline-significance is the class of change where a missed edge case hides (the GAP2 bug itself was one). **Phasing decision: ship R1 first; ship R2 only after its merge gate — the corpus AST-diff fuzz pass below — passes, not on the strength of these hand-picked tests.**

> **Cautionary precedent:** M-TAINT-TYPES (v0.14.3) added `T{not LABEL}` with a 2-token lookahead that proved insufficient vs `func … -> bool { not f(x) }`, causing ~14 mis-parses in the motoko fork. R1's `func IDENT` vs `func (` disambiguation is exactly such a lookahead; Phase 1 must include the funclit-in-`=`-body fixture and a fuzz pass over `benchmarks/*.ail` + `examples/runnable/*.ail` to confirm no existing program re-parses differently.

## Examples

### R1 — what the model writes now just works

**Before** (the model's output, rejected):
```
pure func validateVersion(version: string) -> bool =
  let parts = split(version, ".");
  length(parts) == 3
```
**After**: parses, identically to the braced form it meant.

### R2 — newline-separated block

**Before** (rejected, `PAR020`):
```
pure func countDots(s: string) -> int {
  let n = length(s)
  countOccurrences(s, ".")
}
```
**After**: parses, identically to the `;`-separated form.

## Success Criteria

- [ ] R1: the `=`-body near-miss fixture and `config_file_parser`'s `validateVersion` shape compile.
- [ ] R2: newline-separated block fixtures compile; same-line-no-`;` still errors (`PAR020`).
- [ ] All Conflict Surface fixtures (funclit-in-`=`-body, records, back-to-back decls) parse correctly — fuzz pass over `benchmarks/` + `examples/` shows **zero** re-parse diffs in meaning.
- [ ] `ailang fmt` is idempotent and maps all accepted forms → one canonical output.
- [ ] A/B on the rig: `compile_error` rate on the `;`-family benchmarks drops measurably vs the old parser.
- [ ] `make verify-examples` at baseline; full Go suites green.
- [ ] Docs updated; dialect-traps card trap #2 *removed* (rule no longer exists).

## Testing Strategy

**Unit (parser):** the fixtures enumerated in Conflict Surface §4, plus negative cases (same-line two statements without `;` → `PAR020`).
**Integration:** `make verify-examples` (no regression); a fuzz/round-trip pass parsing every `benchmarks/*.ail` and `examples/runnable/*.ail` under old vs new parser and asserting identical ASTs for currently-valid files.
**Eval (the real metric):** rig A/B on the `;`-family benchmarks.

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| R1 `func` lookahead insufficient (M-TAINT precedent) | High | funclit-in-`=`-body fixture; fuzz over all `.ail`; ship R1 behind no flag only after zero re-parse diffs |
| Canonical-form erosion (two ways to write the same thing) | Med | `ailang fmt` normalizes to one canonical form; idempotence golden test; corpus/examples re-formatted on adoption |
| R2 newline rule misfires on a multi-line *expression* (e.g. a call split across lines) | Med | newline only acts as a separator when `peekStartsBlockStatement()` (let/if/match/ident-at-stmt-start) is true after a *complete* statement; a mid-expression continuation does not start a statement |
| "Two forms" weakens determinism perception | Low | A1 is about execution/trace determinism (unaffected); `ailang fmt` gives presentation determinism |

## Deferred Decisions

- **Canonical separator choice** (`;`-on-line vs newline-per-statement) for `ailang fmt` — agent may choose, pending a quick readability poll; does not block R1/R2 parsing.
- **Whether to also relax tuple-type `(a; b)` → `(a, b)`** (a smaller, related dialect slip) — agent may fold in if trivial, else its own follow-up.

## Non-Goals

- **Significant indentation** (Python-style) — out of scope; newline-as-separator is the minimal change, not block-by-indentation.
- **`match … with` (ML) acceptance** — genuinely rare (2.3%, the card already works); not worth the `with`-keyword ambiguity here.
- **Auto-importing stdlib on use** — a different lever (the undefined-variable class); explicit imports are an AILANG value (A3/A4) and out of scope.
- **Logic errors** (15.9%) — capability, not syntax; addressed only by a stronger/fine-tuned model.

## Related Documents

> The auto-search surfaced only generically-scored, topically-unrelated docs (record-width subtyping, lambda arity, effect-row polymorphism, dashboard optimization). The genuinely related work is:

- [m-ailang-error-quality-for-llm-iteration.md](m-ailang-error-quality-for-llm-iteration.md) — **complement.** That doc made `PAR017`/`PAR020` *actionable so the agent can recover*; this doc removes the error *class* so there is nothing to recover from. The fresh-data section there (PAR017 #1 at 20.5%) is the evidence base for this proposal.
- [m-eval-finetuning-data-pipeline.md](m-eval-finetuning-data-pipeline.md) — **cheaper, broader alternative for the dialect portion.** Fine-tuning bakes the dialect into one model's weights (3-day pipeline + training + per-model cost); this fixes the language once for *every* model. Fine-tuning remains the lever for the *capability* (logic-error) portion this doc explicitly does not address.
- `prompts/agent/dialect-traps.md` — trap #2 (the `;` rule) becomes obsolete on adoption; the card shrinks.

## References

- [Design Axioms](/docs/references/axioms) — A7 (Machines First) is the governing axiom here.
- `.claude/rules/parser.md` — "NEWLINE Tokens Don't Exist" (the invariant R2 relaxes).
- M-AILANG-ERROR-QUALITY local-qwen frequency corpus (334 trials, 2026-06-07…06-12) — the evidence base.

## Future Work

- Extend `ailang fmt` into a pre-commit/CI gate so the canonical form is enforced repo-wide.
- Revisit `match … with` and tuple-type `;` slips if post-adoption eval data shows them rising in the remaining failure mix.

---

**Document created**: 2026-06-12
**Last updated**: 2026-06-12
