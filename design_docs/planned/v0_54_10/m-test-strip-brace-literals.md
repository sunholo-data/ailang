# m-test-strip-brace-literals — Stop brace-counting raw text to find test-block ends

**Status**: Planned
**Target**: v0.54.10
**Priority**: P1 (breaks real user test suites; found in sunholo/crew_core 0.4.0 `population_test.ail`)
**Estimated**: 0.5–1 day
**Dependencies**: None

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

Every feature must align with AILANG's 12 Design Axioms. Score each axiom and verify no hard violations.

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No semantic change; the strip already runs deterministically, just wrongly. |
| A2: Replayability | 0 | No impact on traces. |
| A3: Effect Legibility | 0 | No effect-row changes. |
| A4: Explicit Authority | 0 | No capability changes. |
| A5: Bounded Verification | 0 | No verification surface changes. |
| A6: Safe Concurrency | 0 | No concurrency changes. |
| A7: Machines First | +1 | AI-generated code routinely embeds JSON fragments in string literals. Today `ailang check` accepts a module that `ailang test` rejects with a parse error pointing at valid code — a divergence that actively misleads machine agents. |
| A8: Minimal Syntax | 0 | No new syntax; no parser grammar change. |
| A9: Cost Visibility | 0 | No resource-cost changes. |
| A10: Composability | 0 | No compositional changes. |
| A11: Structured Failure | +1 | Removes a spurious `PAR_NO_PREFIX_PARSE` diagnostic on syntactically valid code, and makes the batch harness stop falling back per-body with a misleading "parse errors in x.ail:" reason. |
| A12: System Boundary | 0 | No boundary changes. |

**Net Score: +2** → **Decision: Move forward**

### Hard Violation Check

**These axioms cannot have −1 scores (automatic rejection):**

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Not optimizing for human convenience over machine analysis

### Decision Thresholds

| Net Score | Decision |
|-----------|----------|
| ≥ +2 | ✅ Proceed to implementation |
| 0 to +1 | ⚠️ Needs stronger justification |
| < 0 | ❌ Reject or redesign |
| Any −1 on A1/A3/A4/A7 | ❌ Automatic rejection |

## Problem Statement

`ailang test` fails to run named tests whose bodies contain string literals with
**unbalanced** braces, while `ailang check` accepts the same file. Minimal repro
(brace.ail):

```ailang
module brace
import std/string (contains)

test "brace in a string" {
	contains("{a", "{")
}
```

- `ailang check brace.ail` → `✓ No errors found!`
- `ailang test brace.ail` → `PAR_NO_PREFIX_PARSE at brace.ail:6:1: unexpected token in expression: }`

**Current State — verified root cause.** The named-test harness does not compile
the user's file directly. It first *strips* the test/property blocks out of the
source (`internal/testing/source_strip.go`) and compiles the remaining module
plus the folded test body (named_batch.go, property_batch.go, executor.go).
`testAndPropertySkipRanges` (source_strip.go:88) decides which lines to remove by
**counting `{`/`}` characters in the raw source text, skipping nothing** — not
string literals, not comments:

```go
for _, ch := range sourceLines[i] {
	switch ch {
	case '{':
		depth++
	case '}':
		depth--
		if depth == 0 { ... goto foundEnd }
	}
}
```

Two symmetric failure modes (both reproduced, see Verification Log):

- **String with an unmatched `{`** (e.g. `"{a"`, or the reported JSON fragment
  `"\"promoted\":{\"kind\""`): depth never returns to 0, `foundEnd` is never
  reached, and `endLine` stays at its initial value `startLine`. The strip range
  covers **only the `test "…" {` line**. The body and its closing `}` remain in
  the stripped source as stray top-level statements → parse error.
- **String with an unmatched `}`** (e.g. `"a}"`): depth hits 0 prematurely, the
  strip range ends **mid-body**, and the tail of the body plus the closing `}`
  are stranded → same parse error.

Both corrupt the module that every named test in the file shares. The batch
compile fails, the harness falls back to per-body compiles (which use the same
strip via `stripNonPureFunctions` → `stripWithLineMap`), and **every test in the
module fails** with the parse error. The interpreter path names a line; the
batch-failure notice (`BatchFailure.Notice`, named_batch.go:82) shows only the
first line of the reason, so users see an apparently empty
`"parse errors in ./x.ail:"` message.

The divergence between `check` (which parses with the real lexer/parser) and
`test` (which re-derives the block extent by brace counting) is the systemic
problem: **the strip path reimplements parsing, badly.** The parser already
knows exactly where each test block ends — the parser convention is to leave the
parser positioned **at the closing RBRACE** when `parseTestDecl`/
`parsePropertyDecl` return (parser_test_decl.go) — but `ast.TestDecl` and
`ast.PropertyDecl` carry no end position, so the strip must guess.

**Impact:** Any AI-generated or human test suite that embeds JSON, template
skeletons, or single-brace fragments in strings inside a named test. This is
routine for the language's target audience (AI code synthesis). Real-world hit:
sunholo/crew_core 0.4.0 `population_test.ail` (JSON fragment with open brace).

## Goals

**Primary Goal:** `ailang test` and `ailang check` agree on the parse validity of
every module — the test-strip path must derive block extents from the parser's
token-accurate positions, not from brace counting over raw text.

**Success Metrics:**
1. The two repro files (unmatched `{` and unmatched `}` in a string inside a
   named test) go green under `ailang test` (and under `--bytecode` /
   `--strict-bytecode`).
2. New corpus cases: strings with balanced braces, `${…}` interpolation whose
   interior contains braces, char literals, and comments containing braces in
   test bodies all strip correctly.
3. All existing named-test/property fixtures keep passing (list in Conflict
   Surface).
4. No new error codes; no parser grammar change; `make test` green.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| D1: Record `End Pos` on `ast.TestDecl`/`ast.PropertyDecl` in the parser and use it in the strip, rather than making the brace scanner string-aware | Chooses "AST is the single source of truth" over "third hand-rolled lexer". Fixes ALL lexical contexts (strings, char literals, comments, interpolation) at once instead of the reported one | human (this doc) | design | med (ast + parser + testing) |
| D2: Fallback behavior when `End.Line == 0` (e.g. an AST constructed without the field) | Decides between "keep the old brace scan as fallback" (matches `functionSkipRanges` convention for FuncDecls with unset Span) vs "error loudly" | agent | design | low |
| D3: Scope = TestDecl/PropertyDecl strip ranges only; do not refactor the line-granularity strip machinery | Boundaries the change; the strip's line-based nature is unchanged (identical granularity to today) | human (this doc) | design | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] D1 resolved: parser-recorded `End Pos` (see Solution Design — rejected
  alternative documented).
- [x] D2 resolved: when `End.Line == 0`, fall back to the existing brace scan,
  mirroring `functionSkipRanges`' `endLine == 0` fallback (source_strip.go:73).
  The fallback is unreachable from `ailang test` after this change (the parser
  sets `End` on every successfully parsed decl) and exists only for ASTs built
  by other constructors; it matches an existing, documented convention in the
  same function family. This is not a data-integrity fallback: worst case it
  reproduces today's behavior exactly.
- [x] D3 resolved: line-granularity strip unchanged.

## Solution Design

### Overview

Make the parser record the source position of the closing brace of each
top-level `test`/`property` block on the AST node (it already parks on that
token), and make the test-strip path consume that position instead of brace
counting raw text. ~25 LOC of product change.

This is the same pattern the codebase already uses for functions:
`functionSkipRanges` trusts `f.Span.End.Line` (with a zero-fallback), and
`parseModuleDecl` captures `endPos := p.curPos()` for `ModuleDecl.Span`
(parser_file.go:152). Test/property decls are the outlier that re-derives the
extent by hand.

### Rejected alternative (documented for reviewers)

**String-aware brace scanner** — teach `testAndPropertySkipRanges` to skip
string/char literals, comments, and `${…}` interpolation while counting, e.g. by
reusing the lexer's `LiteralRegion` machinery (`internal/lexer/comment_scan.go`,
`CollectComments`) or the format package's region-map envelope. Rejected because
it keeps a second, partial parser alive in the test path: any future lexical
addition (new literal form, new comment syntax) must be patched in two places,
which is exactly how this bug arose. The parser already answers the question
authoritatively; option A deletes the guesswork instead of improving it.

### Architecture

**Components:**
1. **AST** (`internal/ast/ast_decl.go`): add an `End Pos` field to `TestDecl`
   and `PropertyDecl`. Additive; zero value (`Pos{}`) means "unset".
2. **Parser** (`internal/parser/parser_test_decl.go`): in `parseTestDecl` and
   `parsePropertyDecl`, both functions already validate `p.curTokenIs(lexer.RBRACE)`
   immediately before constructing the node. Capture `endPos := p.curPos()` there
   and set `End: endPos`. The token position of the closing brace is line/column
   of its first character, which is what the strip needs.
3. **Strip** (`internal/testing/source_strip.go`): in
   `testAndPropertySkipRanges`, use `d.End.Line` as `endLine` when
   `d.End.Line > 0`; otherwise fall back to the existing brace scan (D2).

### Implementation Plan

**Phase 1: AST + parser record the end position** (~1–2 hours)
- [ ] Add `End Pos` to `ast.TestDecl` and `ast.PropertyDecl`
      (`internal/ast/ast_decl.go`).
- [ ] Set `End` from `p.curPos()` at the RBRACE in `parseTestDecl` and
      `parsePropertyDecl` (`internal/parser/parser_test_decl.go`). Cover both
      entry spellings (TEST/PROPERTY token and contextual IDENT "test"/"property")
      — they funnel into the same two functions, so no extra work.
- [ ] Unit-test: `internal/parser/parser_topleveltests_test.go` — assert
      `End.Line`/`End.Column` for a single-line and multi-line test block, and
      for a property block.

**Phase 2: Strip consumes the end position** (~1 hour)
- [ ] `testAndPropertySkipRanges`: prefer `d.End.Line`; keep the brace scan
      behind `d.End.Line == 0` (D2). Update the function's doc comment to say
      the AST end position is authoritative.
- [ ] Unit tests in `internal/testing/source_strip_test.go` with new fixtures
      under `internal/testing/testdata/strip/`: unmatched `{` in string,
      unmatched `}` in string, balanced `{}` pair in string, `${` interpolation
      containing braces, `--`/`//` comments containing braces, a property with
      braces in strings. Assert the stripped source no longer contains the body
      OR the closing `}`.

**Phase 3: End-to-end + regression surface** (~2 hours)
- [ ] Add `examples/tests/brace_string.ail` (the user's repro, both failure
      directions, plus an interpolation case) and run `ailang test` on it —
      interpreter, `--bytecode`, and `--strict-bytecode`.
- [ ] Re-run the Conflict-Surface fixtures and `make test`.
- [ ] Changelog entry (per skill convention, at ship time via
      move_to_implemented).

### Files to Modify/Create

**New files:**
- `examples/tests/brace_string.ail` — user-facing repro/e2e fixture (~20 LOC)
- `internal/testing/testdata/strip/named_test_brace_string.ail` — strip fixture,
  unmatched `{` (~10 LOC)
- `internal/testing/testdata/strip/named_test_closebrace_string.ail` — strip
  fixture, unmatched `}` (~10 LOC)
- `internal/testing/testdata/strip/named_test_interp_brace.ail` — strip fixture,
  `${…{…}…}` interpolation + comments with braces (~12 LOC)

**Modified files:**
- `internal/ast/ast_decl.go` — two `End Pos` fields (~4 LOC)
- `internal/parser/parser_test_decl.go` — capture and set `End` in both parse
  functions (~6 LOC)
- `internal/parser/parser_topleveltests_test.go` — end-position assertions
  (~30 LOC)
- `internal/testing/source_strip.go` — use `End.Line`, fallback doc (~15 LOC)
- `internal/testing/source_strip_test.go` — new fixture-driven cases (~80 LOC)

## Conflict Surface

*(Required: this design touches `internal/parser/`, `internal/ast/`, and a
consumer of both.)*

### Syntactic positions touched

- **No grammar production changes.** The parser's token decisions are untouched:
  `parseTestDecl`/`parsePropertyDecl` already validate the closing RBRACE before
  node construction; we only record `p.curPos()` at a token the parser is
  already parked on (its documented convention; the top-level loop at
  parser_file.go:86-89 consumes that RBRACE via `nextToken()`).
- **AST shape change:** `*ast.TestDecl` and `*ast.PropertyDecl` gain one
  additive field each (`End Pos`). This is the extent of the surface.

### What else lives here

| Position | Existing consumer | Effect of the new field |
|----------|-------------------|-------------------------|
| TestDecl construction | `internal/parser/parser_test_decl.go` (sole constructor — grep verified, Verification Log #8) | Sets one more field |
| TestDecl consumers | `internal/testing/collector.go` (reads Name/Body), `internal/testing/entry_effect_error.go` (compares Body pointers), `internal/format/decl.go` (prints from AST) | None read `End`; additive field is invisible to all |
| Parser tests | `internal/parser/parser_topleveltests_test.go` asserts **field-by-field** (Name, Body length), not `reflect.DeepEqual`/struct-literal equality (read, Verification Log #13) | Unaffected; new assertions added |
| Strip path | `testAndPropertySkipRanges` — sole consumer of the new field | Switches from brace count to `End.Line` |

### Disambiguation strategy

N/A — no parsing decision changes. There is no new token position, no new
lookahead, and no ambiguity introduced; the field is recorded where the parser
already unambiguously stands (at the block's closing RBRACE). This is the same
pattern as `ModuleDecl.Span` (parser_file.go:152) and `FuncDecl.Span`.

### Programs that MUST still work

Regression fixtures (all verified passing today, Verification Log #9):

1. `examples/tests/named_helper_purity.ail` — 1 named test + 1 `property` block
   (covers both decl kinds on the strip path). Currently: 2/2 pass.
2. `tests/record_update_regression_test.ail` — 3 multi-line named tests. 3/3.
3. `tests/stdlib/list_drop_test.ail` — 3 named tests. 3/3.
4. `tests/yaml_bridge_test.ail` — 1 named test. 1/1.
5. `internal/testing/testdata/strip/named_test_multiline.ail` (and the other
   strip fixtures) — exercised by
   `TestStripNonPureFunctions_DeclarationRanges` and friends.

Also the whole existing suite: `make test`.

### What deliberately changes

- **Nothing for valid programs.** No syntax, no semantics, no diagnostics change
  for modules that work today.
- Programs that fail today *because of this bug* start working. That is the
  point; it is not an incompatibility.
- **Line granularity is unchanged** (deliberate, D3): the strip still removes
  whole lines `start..end`. A test whose closing `}` shares a line with the next
  declaration strips that line exactly as the brace scanner does today — same
  granularity, strictly better end-line computation. No regression surface.

## Examples

### Example 1: The reported repro goes green

**Before:**
```
$ ailang check brace.ail && ailang test brace.ail
✓ No errors found!
…
✗ brace in a string — PAR_NO_PREFIX_PARSE at brace.ail:6:1: unexpected token in expression: }
1 tests: 0 passed, 1 failed, 0 skipped
```

**After:**
```
$ ailang test brace.ail
✓ brace in a string
1 tests: 1 passed, 0 failed, 0 skipped
```

### Example 2: JSON fragments in tests (the crew_core case)

```ailang
test "promotion payload" {
	let payload = decode("{\"promoted\":{\"kind\":\"crew\"}")
	-- … assertions on the decoded value …
}
```

Today: whole-module test failure with a misleading parse error. After: the
strip removes exactly the `test` block's lines, the module compiles, the test
runs.

## Success Criteria

- [ ] `ailang test examples/tests/brace_string.ail` passes 100% (interpreter,
      `--bytecode`, `--strict-bytecode`) — the two repro directions plus
      interpolation and comment cases.
- [ ] `ailang check` and `ailang test` agree on parse validity for every file
      under `examples/`, `tests/`, and `std/` (no check-passes-test-fails
      divergence attributable to stripping).
- [ ] New unit tests: strip fixtures for unmatched-`{`, unmatched-`}`,
      balanced-brace, interpolation, comment cases — stripped source contains
      neither body text nor the closing `}`.
- [ ] Parser unit tests assert `TestDecl.End`/`PropertyDecl.End` for single-line
      and multi-line blocks.
- [ ] All Conflict-Surface fixtures still pass (same counts as today).
- [ ] All tests passing (`make test`).
- [ ] Documentation updated: changelog entry at ship time.
- [ ] Examples added: `examples/tests/brace_string.ail`.

## Testing Strategy

**Unit tests:**
- Parser: `End` position correctness for single/multi-line test and property
  blocks, including the contextual-keyword spelling (`test` as IDENT).
- Strip (`internal/testing/source_strip_test.go`, fixture-driven as today):
  the six new lexical cases; assert exact stripped source per fixture style
  already used by `TestStripNonPureFunctions_DeclarationRanges`.

**Integration tests:**
- `ailang test` on `examples/tests/brace_string.ail` across all three engines.
- Property batch path (`property_batch.go` also calls `stripTestBlocks`): a
  property whose body strings contain unbalanced braces.

**Regression-surface tests** (Conflict Surface filled in above):
- One check per "Programs that MUST still work" entry, run via the existing
  `ailang test` fixture runs and `make test`. Any count change (e.g.
  named_helper_purity 2/2) is a regression, not test churn.

**Manual testing:**
- Copy the crew_core `population_test.ail` pattern (JSON fragment with
  `"\"promoted\":{\"kind\""`), verify green.
- `ailang test --bytecode` and `--strict-bytecode` on the new fixture.

## Verification Log

Every load-bearing claim above, with its evidence. (Note: no Go toolchain in
this design session — claims about behavior were verified against the prebuilt
binary `/usr/local/bin/ailang` v0.54.9-91e52cc and by reading the code in this
checkout; the repo is at 9b41b01b. The reproductions match the user report
exactly.)

| # | Claim | Evidence |
|---|-------|----------|
| 1 | `ailang check` accepts the repro while `ailang test` fails with `PAR_NO_PREFIX_PARSE at 6:1` | Live run, transcript in Problem Statement. |
| 2 | Both failure directions break (unmatched `{` AND unmatched `}`) | Live run of `/tmp/close_only.ail` (`contains("a}", "a")`) → identical `PAR_NO_PREFIX_PARSE at 6:1` failure. |
| 3 | Strip mechanism: `testAndPropertySkipRanges` counts braces over raw text, skipping nothing; `endLine` initialized to `startLine` and only set at `foundEnd` | Read `internal/testing/source_strip.go:86-125`. |
| 4 | Parser convention: `parseTestDecl`/`parsePropertyDecl` return with parser AT the RBRACE, which the top-level loop then consumes | Read `internal/parser/parser_test_decl.go` (comment "Leave parser AT the RBRACE (parser convention)") and `internal/parser/parser_file.go:83-90`. |
| 5 | Precedent: `parseModuleDecl` captures `endPos := p.curPos()` for `ModuleDecl.Span.End` at parser_file.go:152 | Read. |
| 6 | `ast.TestDecl`/`ast.PropertyDecl` carry no end position today (fields: Name, Body/Property, Pos only) — **negative-existence claim** | Read `internal/ast/ast_decl.go:144-164`. |
| 7 | `testAndPropertySkipRanges` is the ONLY raw brace scanner on the test path (other brace-depth scanners — `internal/lexer/comment_scan.go`, `internal/lexer/lexer_strings.go`, `internal/format/envelope.go` — are literal-region-aware) — **negative-existence claim** | `grep -rn "'{'"` across `internal/` filtered for depth/brace logic; only source_strip.go:104 counts raw characters. |
| 8 | The parser is the sole constructor of `ast.TestDecl{`/`ast.PropertyDecl{` literals outside tests — **negative-existence claim** | `grep -rn "ast.TestDecl{\|ast.PropertyDecl{\|&TestDecl{\|&PropertyDecl{"` over the repo excluding `parser_test_decl.go` and `*_test.go` → empty. |
| 9 | Conflict-Surface fixtures pass today | Live `ailang test` runs: named_helper_purity 2/2, record_update_regression 3/3, list_drop 3/3, yaml_bridge 1/1. (concurrent_pipeline.ail fails at parse today, pre-existing and unrelated — NOT used as a fixture.) |
| 10 | All strip consumers route through `testAndPropertySkipRanges` (executor.go:193 inline path, executor.go:439 per-body path, named_batch.go:110, property_batch.go:66) — one fix covers all | Grep of `stripTestBlocks`/`stripWithLineMap`/`stripNonPureFunctions` callers. |
| 11 | The "empty parse errors" impression is `BatchFailure.Notice`'s `firstLine(f.Reason)` truncation (named_batch.go:82-90); the full reason, including the PAR error, appears in the per-test failure output | Traced in live repro: per-test output shows both `parse errors in /tmp/brace.ail:` AND `PAR_NO_PREFIX_PARSE at /tmp/brace.ail:6:1` lines. |
| 12 | A string-aware scanner alternative exists (`CollectComments` → `LiteralRegion`; format envelope region map) — feasibility of rejected Option B | Read `internal/lexer/comment_scan.go:76-286`, `internal/format/envelope.go`. |
| 13 | Parser tests assert field-by-field, not struct-literal DeepEqual — additive AST field is safe — **negative-existence claim** | Read `internal/parser/parser_topleveltests_test.go` (Name/Body assertions only; no DeepEqual/require.Equal on TestDecl). |
| 14 | `functionSkipRanges` already trusts AST spans with a `endLine == 0` fallback — convention for D2 | Read `internal/testing/source_strip.go:57-83`. |

## Deferred Decisions

The following are intentionally left open for the implementer:

- Exact naming of the new AST field (`End` vs `Span` struct) — agent may choose;
  `End Pos` is suggested for symmetry with the existing `Pos` field and to avoid
  implying SID semantics that `FuncDecl.Span` carries. If `Span` is chosen, only
  `.End.Line` is consumed.
- Whether the fallback brace scan (D2) gets a dedicated comment citing
  `functionSkipRanges` — agent may choose.
- Test fixture placement (single multi-case `.ail` vs several files) under
  `testdata/strip/` — agent may choose, following the existing one-fixture-per-
  concern style.

## Non-Goals

**Not in this feature:**
- Refactoring the line-based strip into offset-based edits — same granularity
  kept deliberately (D3); no observable behavior change for working programs.
- Improving the `BatchFailure.Notice` first-line truncation (the "empty parse
  errors" impression, Verification Log #11) — presentation polish, distinct
  failure, and it disappears for this bug class once the strip is fixed.
  Future work.
- Teaching any other consumer to use the new `End` field (e.g. LSP, formatter) —
  they already work from AST bodies; no consumer asked for it.
- Changes to `internal/lexer/` — the lexer is already correct; the bug is in the
  reimplementation, which this design deletes.
- New error codes (none needed; no diagnostic surface changes).

## Timeline

**Week 1** (5 hours):
- Phase 1: AST + parser end positions (2h)
- Phase 2: strip consumes End (1h)
- Phase 3: fixtures, e2e, regression run (2h)

**Total: ~5 hours across 1 week** (estimate ×2 initial guess, per convention).

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| A code path constructs TestDecl without `End` and regresses to the old behavior silently | Low | Sole constructor is the parser (Log #8); fallback matches today's behavior exactly (Log #14 convention); unit tests pin `End` for all entry spellings |
| `End.Line` granularity differs from brace-scan end line in some edge (e.g. `}` not at line start) | Low | Granularity is line-based either way; strip removes whole lines `start..end` — same set of lines for every program that works today. New fixtures pin the exact stripped output |
| Bytecode/property batch paths behave differently from the interpreter path | Med | All three engines covered in Success Criteria; property_batch also routed through the same strip (Log #10) |
| Hidden consumer of `file.Decls` doing struct-literal equality breaks on the new field | Low | Verified: parser tests assert field-by-field (Log #13); no other TestDecl constructors (Log #8) |

## Related Documents

**Implemented (may inform design):**
- (none found by auto-search — search ran on "test strip brace literals",
  SimHash and neural, zero matches in both `planned/` and `implemented/`)

**Planned (check for overlap):**
- `design_docs/planned/m-parser-error-cascade-suppression.md` — adjacent
  (parser diagnostics) but distinct: this design changes no diagnostic; it
  removes a spurious one by fixing the strip. No overlap in mechanism.
- `design_docs/planned/m-package-test-discovery.md` — adjacent (test tooling)
  but about discovery, not block extents. No overlap.

## References

- [Design Axioms](/docs/references/axioms) - The 12 non-negotiable principles
- `internal/testing/source_strip.go` — the buggy scanner (line 88-125)
- `internal/parser/parser_test_decl.go` — the fix site (both decl kinds)
- `internal/ast/ast_decl.go:144-164` — TestDecl/PropertyDecl (no end position today)
- `internal/parser/parser_file.go:152` — ModuleDecl Span-end precedent
- Reported by user (v0.54.8, darwin arm64): sunholo/crew_core 0.4.0
  `population_test.ail`

## Future Work

- Surface `BatchFailure` reasons beyond the first line (or include the first
  parse-error line) so batch failures never look "empty" (Log #11).
- Migrate `functionSkipRanges`'s `endLine == 0` fallback and any remaining
  text-scanning strip helpers to a single AST-position-authoritative helper,
  if more such cases appear.

---

**Document created**: 2026-10-13
**Last updated**: 2026-10-13
