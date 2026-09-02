# M-PARSER-RESERVED-KEYWORD-DIAGNOSTICS: catch `as`-as-identifier before it cascades

**Status**: Planned
**Target**: v0.33.1
**Priority**: P2 — real, recurring, but narrow (one keyword, confirmed across 5 model×benchmark pairs)
**Estimated**: 2-3 days (diagnostic-only); +2-3 days if the contextual-keyword option is chosen
**Dependencies**: None

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Parser behavior on invalid input becomes more specific, not nondeterministic |
| A2: Replayability | 0 | Not touched |
| A3: Effect Legibility | 0 | Not touched |
| A4: Explicit Authority | 0 | Not touched |
| A5: Bounded Verification | +1 | A model (or human) can now diagnose and fix this specific class of parse failure from the FIRST error alone, instead of a multi-line cascade |
| A6: Safe Concurrency | 0 | Not touched |
| A7: Machines First | +1 | Directly reduces token cost / retry count for the exact failure mode observed in real eval data (5 confirmed instances) |
| A8: Minimal Syntax | 0 (diagnostic-only option) / +1 (contextual-keyword option, since it makes `as` behave like an ordinary identifier in more positions, reducing special-cased vocabulary a model must memorize) | See High-Impact Decisions — score depends on which option is chosen |
| A9: Cost Visibility | 0 | Not touched |
| A10: Composability | 0 | Not touched |
| A11: Structured Failure | +1 | The new diagnostic is a specific, typed parser error code, not prose bolted onto an existing one |
| A12: System Boundary | 0 | Not touched |

**Net Score: +3 (diagnostic-only) or +4 (contextual-keyword)** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Explicitly reduces AI token/retry cost, the opposite of optimizing for human convenience alone

### Decision Thresholds

| Net Score | Decision |
|-----------|----------|
| ≥ +2 | ✅ Proceed to implementation |

## Verification Log

Claims in this doc verified against the live codebase (2026-08-04, dev @ v0.33.0):

| Claim | Method | Result |
|-------|--------|--------|
| `as` is a hard, globally-reserved lexer keyword (not contextual) | Read [internal/lexer/token.go:186](../../../internal/lexer/token.go) (`AS: "as"` in the `TokenType` stringer) and [token.go:297](../../../internal/lexer/token.go) (`"as": AS` in the `keywords` map) | Confirmed — the lexer maps the literal text `as` to token type `AS` unconditionally, before the parser ever sees it; there is no contextual re-classification |
| `AS` is consumed in exactly one grammar area | `grep -rn "lexer\.AS\b" internal/parser/*.go` (excluding `_test.go`) | 3 hits, all in [internal/parser/parser_file.go:301,330,360](../../../internal/parser/parser_file.go) — import-alias parsing (`import M as L`, `import M (f as g)`) is the ONLY grammar position that consumes `AS` |
| No other construct (pattern-binding, casts, etc.) uses `as` | Same grep — zero hits outside `parser_file.go` | Confirmed by exhaustive search, not inference — `as` genuinely has no other role in the grammar today |
| The failure mode is real and recurring, not a single model's mistake | `jq` scan of `eval_results/baselines/v0.32.0/standard/*_ailang_*.json` for `stderr` containing `"unexpected token in expression: as"` | 5 distinct (model, benchmark) pairs, 4 different model families: `or-kimi-k2-7-code`/`contract_sorted_merge`, `gemini-3-flash`/`dep_resolver_backtrack`, `claude-sonnet-4-6`/`docx_reimplement`, `claude-sonnet-4-6`/`markdown_reimplement`, `or-minimax-m3`/`ssa_constant_fold` |
| Concrete example of the collision and its cascade | Read generated code + stderr for `dep_resolver_backtrack_ailang_gemini-3-flash_*.json` | `pure func findAssigned(as: [Assignment], name: string) -> Option[int] = ...` at line 13; parser desyncs at `13:26` ("expected next token to be ), got : instead") and produces 25+ downstream `PAR_UNEXPECTED_TOKEN`/`PAR_NO_PREFIX_PARSE`/`PAR015` errors scattered through the rest of the file, none naming `as` or the real cause |
| Duplicate design docs | `create_planned_doc.sh` auto-search (SimHash + neural) | Top matches (`m-codegen-blank-identifier.md`, `m-codegen-typed-slices.md`, `m-serveapi-unify.md`, `global-collaboration-hub.md`, `m-agent-loop-architecture.md`, `m-forall-properties-direct-core-eval.md`) are unrelated keyword-coincidence hits, none about reserved-keyword diagnostics or `as`. Not a duplicate. |

## Problem Statement

`as` is one of ~40 hard-reserved AILANG keywords, used only for import aliasing
(`import std/array (length as lengthAlt)`). It is also one of the shortest, most natural English
words a model reaches for as a variable/parameter name — "assignments" abbreviates to `as`
instinctively, the same way a human might.

When a model uses `as` as an identifier, the parser doesn't recognize "reserved keyword in
identifier position" as a distinct failure. Instead it desyncs at that token and free-associates
through the rest of the file, emitting a wall of `PAR_UNEXPECTED_TOKEN` / `PAR_NO_PREFIX_PARSE` /
`PAR015` errors at unrelated-looking line numbers, none of which name `as` or point back to the
actual cause.

**Current State:**
- Confirmed across 5 distinct (model, benchmark) pairs in v0.32.0 standard-mode results, spanning
  4 model families (Gemini, Claude, Kimi, MiniMax) — not a single model's idiosyncrasy.
- A representative case (`dep_resolver_backtrack`, `gemini-3-flash`) produces over 25 cascading
  parser errors from one root cause, none of which say "as is reserved."
- In standard (0-shot, no-retry) mode this is an outright benchmark failure. In agent mode it costs
  extra turns for the model to guess the real problem from a noisy error dump.

**Impact:**
- Affects any model, on any benchmark, that happens to reach for `as` as a short name — this is a
  content-independent failure mode, not tied to benchmark difficulty.
- Standard-mode failures from this cause are miscategorized as generic `compile_error` /
  `PAR_*` noise in eval data, making them invisible to per-benchmark difficulty analysis unless
  someone reads the raw stderr (as this investigation did).

## Goals

**Primary Goal:** When a model writes a reserved keyword where an identifier is expected, the
FIRST parser error names the real cause ("`as` is a reserved keyword, not a valid identifier")
instead of a downstream cascade of unrelated-looking token errors.

**Success Metrics:**
- All 5 known failing (model, benchmark) result files, re-run through the fixed parser, produce a
  single clear diagnostic instead of a cascade (verified via `ailang check` on extracted
  minimal repros, not just the original noisy multi-error dumps).
- A new regression fixture (`examples/`) using `as` as a parameter name produces exactly one
  diagnostic naming the keyword and the fix.
- (If the contextual-keyword option is chosen) the same fixture instead **compiles successfully**,
  and the 3 existing `import ... as ...` regression tests still pass unchanged.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| **Option A (diagnostic-only)**: detect "reserved keyword in identifier position" and emit one clear `PAR_RESERVED_KEYWORD`-style error, still rejecting the program, vs. **Option B (contextual keyword)**: make `as` a valid identifier everywhere EXCEPT immediately after `import <path> (` / `import <path>`, so `func findAssigned(as: ...)` simply compiles | Option A is strictly additive and safe (~2-3 days); Option B removes the footgun entirely but changes what's a valid program (~4-6 days total, needs the Conflict Surface below signed off) | human | design | high |
| If Option B is chosen: whether contextual re-classification happens in the lexer (context-sensitive lexing) or the parser (lexer always emits `AS`, parser's identifier-expecting call sites accept `AS` as a synonym for `IDENT` outside import position) | Affects blast radius — parser-level re-classification is more localized and lower-risk than teaching the lexer about parse context | agent | compile | med |
| Whether to extend this diagnostic to OTHER reserved keywords beyond `as` (e.g. `select`, `send`, `recv`, `timeout`, `test` — all short, plausible identifier names not observed failing in this dataset) | Broader coverage front-loads more risk/scope for zero confirmed evidence beyond `as`; the data only supports `as` | human | design | low |

### Design Freeze

- [ ] Choose Option A vs. Option B before implementation starts (this doc's Solution Design covers both; Non-Goals scopes Option B's implementation as optional/deferred pending this choice)
- [ ] Confirm whether to extend beyond `as` to other reserved keywords (default: no, per Non-Goals — data only supports `as`)

## Conflict Surface

*(Required — this design touches `internal/lexer/` and `internal/parser/`.)*

### Syntactic positions touched

- **Option A** touches only the PARSER's error-recovery/reporting path when an identifier is
  expected but an `AS` token is found — e.g. wherever `parseIdentifier`/`expectIdent`-style
  helpers currently produce a generic "expected IDENT" error. No grammar production changes.
- **Option B** touches the lexer/parser boundary for the `AS` token specifically: it would make
  `AS` acceptable as an identifier in every position that currently accepts `IDENT`, EXCEPT the 3
  confirmed import-alias positions (`parser_file.go:301,330,360`).

### What else lives here

| Position | Existing valid form | Shape |
|----------|--------------------|-------|
| After `import <path> (`, before `)` | import-list alias | `import std/array (length as lengthAlt)` — `as` here MUST remain the alias keyword |
| After `import <path>`, before end-of-statement | whole-module alias | `import std/array as Arr` — `as` here MUST remain the alias keyword |
| Function parameter name position | any `IDENT` | `func f(x: int) -> ...` — Option B would add `func f(as: int) -> ...` as a NEW valid form here |
| `let`-binding name position | any `IDENT` | `let x = ...` — Option B would add `let as = ...` as a NEW valid form here |
| Record field name position | any `IDENT` | `{name: string}` — Option B would add `{as: string}` as a NEW valid form here (needs explicit confirmation this is desired; a record field literally named `as` is unusual but not harmful) |

Because the grep in the Verification Log found `AS` consumed in EXACTLY 3 call sites, all inside
`parser_file.go`'s import-declaration parsing, there is no ambiguous THIRD claimant on any of
these positions today — the only disambiguation needed is "am I directly inside an import
declaration's alias slot, or not," which the parser already tracks structurally (it's mid-way
through `parseImportDecl` when it reaches those 3 call sites).

### Disambiguation strategy

**Option A** needs no disambiguation — it only changes the error message, not what parses.

**Option B**: the 3 call sites in `parser_file.go` that consume `AS` today would keep consuming it
as the alias keyword unconditionally (they are only reached while already inside
`parseImportDecl`). Every OTHER place the parser currently calls its identifier-expecting helper
(function params, let-bindings, record fields, etc.) would additionally accept an `AS` token and
treat its literal text as a plain identifier. This is sound because those two code paths never
overlap — `parseImportDecl` is a distinct top-level parse function from the general
expression/declaration parser, not a shared entry point with ambiguous lookahead.

### Programs that MUST still work

- `import std/array (length as lengthAlt)` — [std/list.ail or similar] still parses as an aliased
  import, not a parameter named `as`.
- `import std/array as Arr` — whole-module alias still works.
- Every existing example/test using `as` for import aliasing (search: `grep -rln " as " examples/*.ail std/*.ail | xargs grep -l "^import"`) continues to parse identically.
- (Option B only) A program using `as` as a plain identifier outside import position — the new
  regression fixture this doc adds — must now compile where it previously produced `PAR_*` errors.

### What deliberately changes

- (Option A only) The TEXT of the error message/code for this specific failure mode changes; the
  program is still rejected. No previously-accepted program is affected.
- (Option B only) Programs using `as` as a non-import identifier — previously always a parse
  error — now compile. This is an intentional, backward-compatible relaxation (strictly grows the
  set of accepted programs; nothing previously valid becomes invalid).

## Solution Design

### Overview

Implement Option A (diagnostic) unconditionally — it's low-risk and directly closes the
"cascading noise" problem. Implement Option B (contextual keyword) only if the Design Freeze
decision selects it; the Conflict Surface above shows it's low-risk given `AS`'s exhaustively
verified single-purpose grammar footprint, but it's a larger, separable change.

### Architecture

**Components:**
1. **Reserved-keyword identifier check (Option A)**: wherever the parser calls its
   identifier-expecting helper and receives a token whose type is a keyword (not `IDENT`), emit a
   specific error naming the literal keyword text and its one legitimate use, instead of falling
   through to the generic "expected IDENT, got X" path that triggers the cascade.
2. **Contextual `AS` acceptance (Option B, conditional)**: a small set of parser call sites
   (function params, let-bindings, record fields — enumerated in Conflict Surface) accept `AS` as
   a synonym for `IDENT`, using the token's literal text as the identifier name.

### Implementation Plan

**Phase 1: Reserved-keyword diagnostic (Option A)** (~1.5 days)
- [ ] Identify the shared identifier-parsing helper(s) in `internal/parser/` (likely one or two
      central functions given AILANG's existing centralized error-code approach elsewhere, e.g.
      `IMP010`'s "did you mean" pattern in the import resolver)
- [ ] Add a check: if the unexpected token is a keyword token (lookup its literal via the reverse
      of the `keywords` map in token.go), emit a new error code (e.g. `PAR_RESERVED_KEYWORD`) with
      the keyword's name and a suggestion ("`as` is reserved for import aliasing; choose a
      different name")
- [ ] Verify the new code is unallocated: `grep -rn "PAR_RESERVED_KEYWORD" internal/` before use
- [ ] Regression test: the `dep_resolver_backtrack`/`gemini-3-flash` minimal repro (extracted from
      the real failing code) now produces exactly this one error, not a cascade

**Phase 2: Contextual `AS` acceptance (Option B — only if Design Freeze selects it)** (~2-3 days)
- [ ] Implement the parser-level acceptance at each Conflict-Surface-enumerated call site
- [ ] Regression tests: every "Programs that MUST still work" entry from the Conflict Surface gets
      a pinned test
- [ ] New fixture: `examples/reserved_word_as_identifier.ail` using `as` as a parameter name,
      compiles and runs

**Phase 3: Verify against the real failure set** (~0.5 day)
- [ ] Re-extract the 5 confirmed failing code snippets from v0.32.0 results and confirm the fix
      resolves them (Option A: clear single diagnostic; Option B: compiles)

### Files to Modify/Create

**Modified files:**
- `internal/parser/parser_*.go` (exact file depends on where the shared identifier-parsing helper
  lives — confirm during Phase 1) - Option A diagnostic, ~20-40 LOC
- `internal/parser/parser_file.go`, plus wherever function params / let-bindings / record fields
  are parsed - Option B contextual acceptance, ~30-60 LOC (only if selected)
- `internal/lexer/token.go` - no change expected (keyword map stays as-is; only the PARSER's
  reaction to an `AS` token changes)

**New files:**
- `examples/reserved_word_as_identifier.ail` - regression fixture (Option B only), ~10 LOC

## Examples

### Example 1: Option A — before/after error output

**Before:**
```
$ ailang check dep_resolver_backtrack.ail
PAR_UNEXPECTED_TOKEN at ...:13:26: expected next token to be ), got : instead
PAR_UNEXPECTED_TOKEN at ...:13:26: expected next token to be {, got : instead
PAR_NO_PREFIX_PARSE at ...:13:26: unexpected token in expression: :
... (25+ more lines, none mentioning "as")
```

**After:**
```
$ ailang check dep_resolver_backtrack.ail
PAR_RESERVED_KEYWORD at ...:13:26: 'as' is a reserved keyword (used for import aliasing) and
cannot be used as an identifier here.

Suggestion: choose a different parameter name, e.g. 'assignments' or 'asgn'.
```

### Example 2: Option B — before/after acceptance

**Before:** `func findAssigned(as: [Assignment], name: string) -> Option[int] = ...` fails to parse.

**After:** the same line compiles; `as` is bound as an ordinary parameter name inside the function
body, exactly like any other identifier.

## Success Criteria

- [ ] Option A: the 5 known failing snippets each produce exactly one `PAR_RESERVED_KEYWORD`-class
      error, not a cascade (acceptance test per snippet)
- [ ] Option A: `PAR_RESERVED_KEYWORD` (or chosen code name) verified unallocated before use
- [ ] (Option B) All "Programs that MUST still work" fixtures pass unchanged
- [ ] (Option B) New fixture compiles and runs correctly
- [ ] All tests passing
- [ ] `ailang prompt` / teaching-prompt reserved-word documentation (if any) stays accurate

## Testing Strategy

**Unit tests:**
- Parser-level: feeding `AS` where `IDENT` is expected produces the new diagnostic (Option A) or
  successfully parses (Option B), for each Conflict-Surface position

**Integration tests:**
- Full `ailang check` on each of the 5 real failing snippets (redacted/minimized to just the
  triggering construct)
- Full `ailang check` on every existing `import ... as ...` example/test (regression)

**Manual testing:**
- Run one of the 5 originally-failing model outputs through `ailang check` and visually confirm
  the improved diagnostic (Option A) or successful compile (Option B)

## Deferred Decisions

- Exact parser call site(s) for the Option A check — agent may choose, once the shared
  identifier-parsing helper(s) are located in Phase 1
- Exact new error code name/number (must be verified unallocated at implementation time, not
  assumed from this doc)

## Non-Goals

- **Extending this diagnostic/contextual treatment to reserved keywords other than `as`** - no
  failure evidence exists for any other keyword in the current dataset; extending speculatively
  risks Conflict Surface gaps for words with genuinely ambiguous grammar positions (unlike `as`,
  which was confirmed to have exactly one). If future eval data surfaces another keyword collision,
  it should get its own doc with its own Conflict Surface analysis.
- **Implementing Option B unless the Design Freeze selects it** - Option A alone fully addresses
  the "confusing cascade" problem; Option B is a larger, separable improvement this doc scopes but
  doesn't mandate.

## Timeline

**Days 1-2**: Phase 1 (Option A)
**Days 3-5** (if selected): Phase 2 (Option B)
**Final half-day**: Phase 3 verification

**Total: 2-3 days (Option A only) or 4.5-5.5 days (both options)**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| (Option B) A parser call site consuming `AS` for import aliasing is missed during the Conflict Surface enumeration, silently breaking an alias | Medium | The grep in the Verification Log is exhaustive (3 hits, all read and confirmed) — re-run the same grep as a CI-time regression check so a future `AS` consumer can't silently appear without updating this analysis |
| (Option B) A record field or binding genuinely named `as` reads as confusing to a human, even if the parser handles it fine | Low | Axiom 7 (Machines First) accepts this trade-off; document `as` as a discouraged-but-legal identifier in the language reference if Option B ships |
| New error code collides with an existing one | Low | Verification step in Phase 1 explicitly greps for the chosen code name before use (hard gate) |

## Related Documents

**Implemented (informs this design):**
- IMP010 "not exported by X, did you mean Y" auto-import hints (referenced in
  `design_docs/PROGRAM.md` §5c AILANG fix backlog as a shipped, similarly-scoped diagnostics
  improvement) — same spirit: turn a confusing raw failure into an actionable, specific message

**Planned (no overlap found):**
- None of the top SimHash/neural matches (see Verification Log) discuss reserved keywords or
  parser diagnostics for identifier collisions.

## References

- [Design Axioms](/docs/references/axioms)
- `internal/lexer/token.go`, `internal/parser/parser_file.go`

## Future Work

- If eval data later surfaces collisions with other short reserved keywords (`select`, `send`,
  `recv`, `timeout`, `test` are plausible candidates given their brevity), repeat this doc's
  Conflict Surface methodology per-keyword rather than batching speculatively.

---

**Document created**: 2026-08-04
**Last updated**: 2026-08-04
