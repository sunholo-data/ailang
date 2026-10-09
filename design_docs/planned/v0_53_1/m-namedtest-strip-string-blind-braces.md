# Named-test source strip: parse-time block ends replace the string-blind brace scan

**Status**: Planned
**Target**: v0.53.1
**Priority**: P1 (Medium) — a whole test file fails on valid source, but a workaround exists (brace-balanced string literals, or hoisting bodies into named `pure` funcs)
**Estimated**: 1 day (implementation ~3h, tests ~3h, docs/verification ~2h)
**Dependencies**: None. Builds on the strip that `m-test-runner-compile-once` (planned v0_53_0, shipped in changelog under v0.52.2) batch-compiles on top of.

## Problem Statement

`ailang test` re-derives, from raw source text, the line range of every
`test "…" { … }` and `property "…" { … }` block before compiling test bodies.
`testAndPropertySkipRanges` (`internal/testing/source_strip.go`) scans the
characters of each line from the decl's start line, counting `{` and `}` with
**no awareness of string literals, comments, or interpolation**, and takes the
first return to depth 0 as the block's end. Any unbalanced brace inside a
string literal, a `--` comment, or the test's own name string ends the scan at
the wrong line. The rest of the block then leaks into the "stripped" module
that every named-test compile is built on, and the generated file does not
parse. Because **one** strip is shared by every test of the file — the
v0.52.2+ shared batch compile *and* each per-body fallback compile — one bad
literal in one test fails **every test in the file** with a parse error named
against a synthesized temp file the user never wrote.

Reported against v0.52.0 (`stapledons-godot sim/protocol_ism_test.ail`, every
test in the file failing with `pipeline error: module loading error: failed to
load .../ailang-namedtest-NNN/protocol_ism_test.ail: parse errors` while
`ailang check` of the same file passes), and already triaged as
`design_docs/planned/ailang-core-triage/source-strip-string-brace-skip-ranges.md`
(2026-09-15, **Recommend: design-doc**).

**Reproduced on the current tree** (v0.53.0-dev; installed binary v0.52.5,
commit `7200786`, 2026-10-07). With a well-formed module plus:

```ailang
test "json close brace" {
    contains(reply(newSession(), helloLine(7)), "}")
}

test "still fine" {
    contains(reply(newSession(), helloLine(7)), "minor")
}
```

`ailang check` passes and `ailang test` fails **both** tests:

```
→ named tests in unbalanced_test.ail: could not share one compile (pipeline
  error: module loading error: failed to load /tmp/ailang-namedtest-45499571/
  unbalanced_test.ail: parse errors …); compiled each test separately
✗ json close brace   pipeline error: … PAR_NO_PREFIX_PARSE at …/unbalanced_test.ail:9:1:
                      unexpected token in expression: }
✗ still fine         (same error — this test's own body never ran)
```

Note the double failure mode: the batch fails, the per-body fallback **also**
fails (it is built from the same corrupted strip), so the D1 fallback cannot
isolate the bad test — and `finishNamedTests` then correctly reports "this is
an ailang test harness bug; please report it" (every body fails alone), which
is exactly what the reporter did.

Two more triggers of the same scan verified live (same symptom, all tests in
the file fail): a `}` inside a `--` comment within a test body, and a `}`
inside the **test's name string** (`test "close } brace in name" { … }` — the
scan starts on the decl line, whose name literal precedes the block's `{`).
Balanced braces inside strings are *not* a trigger (the depth count returns
through them); the reporter's `"\"minor\":7"` snippet alone passes on HEAD —
protocol/JSON test files are the natural habitat of the unbalanced forms
(`"}"` JSON fragments, malformed-payload fixtures).

**Impact:** any test module whose bodies check brace-bearing text — JSON,
protocol frames, error messages — silently loses its entire test suite to a
parse error pointing at a nonexistent file. The reporter's workaround (hoisting
every body into a named `pure` func) costs real test-suite ergonomics and
misattributes the failure to escaped quotes (see Verification Log V1).

## Goals

**Primary Goal:** A test block containing any valid AILANG expression —
including unbalanced braces inside string literals, comments, or its own name
— is stripped correctly, so `ailang test` reports each test's own outcome.

**Success Metrics:**
- The three live reproductions above (string, comment, name) go from
  "every test in the file fails with a parse error" to "each test passes/fails
  on its own body" — pinned by a new end-to-end regression test.
- `testAndPropertySkipRanges` no longer reads source characters; the strip's
  block ranges come from parser-recorded positions only (the character scan is
  deleted, not extended).
- No behavior change for any currently-working file: the recorded range equals
  the scanned range on all well-formed inputs (both are `[decl line … closing-
  brace line]`), pinned by the existing `internal/testing/testdata/strip/`
  and `testdata/named_batch/` fixtures.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Fix at the parser (record the block's end `Pos` on `TestDecl`/`PropertyDecl`) rather than making the strip string/comment-aware or re-lexing in the stripper | Decides whether lexical knowledge is duplicated in `internal/testing` forever; the alternative leaves a second, drift-prone lexer half (escapes, interpolation, comments) next to the real one | human (this doc; triage row listed all three as plausible) | design | med |
| The old character scan is **deleted**, not kept as a fallback path | A retained fallback re-arms the bug for any AST path that skips the new field; dead code with a known defect is the incremental-special-casing anti-pattern | human | design | low |
| Field shape on the AST nodes: `End Pos` vs a full `Span` mirroring `FuncDecl.Span` | Additive AST change consumed by `format`, JSON `Compact`, LSP-ish tooling; must not break `String()`/`Position()` contracts | agent | implementation | low |
| Guard for unset/inverted `End` (hand-built ASTs) mirrors `functionSkipRanges`' existing `endLine == 0 \|\| endLine < startLine` guard | Keeps the strip total over ASTs that did not come from the parser without reintroducing textual scanning | agent | implementation | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] Block ends come from the parser, not from any textual scan (decision 1 above).
- [x] The character scan in `testAndPropertySkipRanges` is removed outright (decision 2).
- [ ] Exact AST field shape (`End Pos` vs `Span`) — agent decides at implementation,
      both are additive; record the choice in the sprint notes.

## Solution Design

### Overview

The parser is the only component that already knows where a test block ends —
it consumes the closing `}` — and it knows it *authoritatively* (strings,
comments, and interpolation are already lexed away). Today `parseTestDecl` and
`parsePropertyDecl` (`internal/parser/parser_test_decl.go`) deliberately leave
the cursor **at** the closing `RBRACE` and then discard that position,
returning only the start `Pos`. The fix is to record it: add an end position
to `ast.TestDecl` and `ast.PropertyDecl`, set it at the closing brace exactly
as `parseFunctionDeclaration` already does for `FuncDecl.Span`
(`endPos := p.curPos()` at the closing brace, `fn.Span` set at
  `parser_func.go:231,324`), and
make `testAndPropertySkipRanges` use `[Pos.Line, End.Line]` instead of
counting braces.

This is triage option (c) ("fall back to the AST decl spans instead of
re-scanning"). Options (a) (string-aware scanning in the stripper) and (b)
(reuse the lexer token stream in the stripper) both keep a second copy of
lexical knowledge in `internal/testing`; (c) deletes the copy. It also fixes
all three trigger classes at once — body strings, comments, and the name
string — which a body-string-only scanner (option a) would not.

### Architecture

**Components:**
1. **`internal/ast/ast_decl.go`** — `TestDecl` and `PropertyDecl` gain an end
   position (e.g. `End Pos`, or `Span Span` if mirroring `FuncDecl` is
   preferred). Additive: only the parser constructs these nodes in non-test
   code (verified — see Verification Log V6), and `String()`/`Position()` are
   unchanged.
2. **`internal/parser/parser_test_decl.go`** — in `parseTestDecl`, after the
   body loop the current token is the closing `RBRACE` (the "leave parser AT
   the RBRACE" convention); capture `p.curPos()` as the end before returning.
   Same for `parsePropertyDecl` after its closing-brace check. This is the
   `FuncDecl` pattern verbatim.
3. **`internal/testing/source_strip.go`** — `testAndPropertySkipRanges` stops
   taking `sourceLines` and emits `sourceLineRange{start: d.Pos.Line,
   end: <End.Line>}` per test/property decl, with the same invalid-span guard
   `functionSkipRanges` uses (`end == 0 || end < start` → collapse to the
   decl's own line). The brace-count loop, `depth`, and the `goto foundEnd`
   are deleted. `stripWithLineMap`'s signature drops the now-unused
   `sourceLines` plumbing.

For every well-formed file the produced range is identical to today's — the
scan's `endLine` was the closing brace's line, and `End` is that same
position — so no currently-passing strip changes. The `lineMap` (batched
line → user line, `named_batch.go` D5) is built from the same per-line
filter and is unaffected.

### Implementation Plan

**Phase 1: Parser records the block end** (~2 hours)
- [ ] Add the end-position field to `TestDecl` and `PropertyDecl` in
      `internal/ast/ast_decl.go`.
- [ ] Set it in `parseTestDecl`/`parsePropertyDecl` at the closing `RBRACE`.
- [ ] Parser unit tests: end line equals the closing brace's line for a
      single-line block, a multi-line body, and a body containing `}` inside
      a string literal and inside a comment.

**Phase 2: Strip uses the recorded end** (~3 hours)
- [ ] Rewrite `testAndPropertySkipRanges` to span-based ranges with the
      invalid-span guard; delete the character scan.
- [ ] Strip unit tests (extend `internal/testing/source_strip_test.go`):
      new `testdata/strip/` fixtures whose test blocks contain `}` in a
      string, in a comment, and in the test name — each must be removed
      exactly `[start, end]` (assert on the stripped text and on the
      `lineMap`); a hand-built `TestDecl` with unset `End` exercises the
      guard; the existing `TestStripNonPureFunctions_*` table keeps passing
      unchanged (function stripping is untouched).

**Phase 3: End-to-end regression + docs** (~3 hours)
- [ ] New end-to-end test (alongside `cmd/ailang/test_bytecode_test.go`):
      a module with the three trigger tests plus one clean test — `ailang
      test` reports each test's own outcome; `ailang test --bytecode` and
      `--strict-bytecode` agree (the strip feeds `evalNamedTestBodyOnVM`'s
      `baseSource` too); no `named tests in …: could not share one compile`
      notice, and no harness-bug notice.
- [ ] Run the whole existing suite: `make test-core`, plus
      `internal/testing` (named_batch, engine_parity), `internal/parser`,
      `internal/format` (the printer re-prints `TestDecl` via
      `decl.go:528` — must be unaffected).
- [ ] CHANGELOG entry under the next version.

### Files to Modify/Create

**Modified files:**
- `internal/ast/ast_decl.go` (+2 LOC) — end-position fields on `TestDecl`, `PropertyDecl`.
- `internal/parser/parser_test_decl.go` (+4 LOC) — record the end at the closing `RBRACE`.
- `internal/testing/source_strip.go` (−30/+12 LOC) — span-based test/property ranges; scan deleted.
- `internal/testing/source_strip_test.go` (+~80 LOC) — trigger fixtures + guard test.
- `internal/parser/parser_test_decl_test.go` (or the parser's existing test file) (+~40 LOC).
- `cmd/ailang/test_strip_braces_test.go` (new, ~60 LOC) — end-to-end regression.
- `internal/testing/testdata/strip/brace_literals.ail` (new fixture, ~15 LOC).

**Explicitly NOT modified:** `internal/lexer` (no lexical change),
`internal/testing/executor_helpers.go` (`PrintAILANGSource` — its
`StringLit` re-escaping is correct; verified V1),
`internal/testing/named_batch.go` / `property_batch.go` (they consume
`stripWithLineMap` unchanged).

## Examples

### Example 1: The reporter's class of file

**Before** (`ailang test`, current HEAD — every test in the file fails):
```
named tests in protocol_ism_test.ail: could not share one compile (pipeline
error: module loading error: failed to load …/ailang-namedtest-45499571/
protocol_ism_test.ail: parse errors …); compiled each test separately —
every body compiles on its own, so this is an ailang test harness bug; please report it
✗ hello contains minor    PAR_NO_PREFIX_PARSE at …/protocol_ism_test.ail:9:1: unexpected token in expression: }
✗ (…every other test, same error)
```

**After** (each test runs its own body):
```
✓ hello contains minor
✓ frame round-trips JSON fragments like "}"
```

### Example 2: Why the workaround worked

Hoisting bodies into `pure func check1() -> bool ! {}` worked not because
named funcs are special to the runner, but because the hoisted bodies are no
longer inside a `test "…" { … }` block: the strip keeps `pure` functions
verbatim (span-based `functionSkipRanges`, already parser-driven), so there
is no brace scan left to misread. After this fix the hoist is unnecessary.

## Success Criteria

- [ ] A test body containing `}` inside a string literal, a `--` comment, or
      the test name no longer corrupts the strip: every test in the file
      reports its own outcome (end-to-end test, both engines).
- [ ] `testAndPropertySkipRanges` contains no character-level brace counting
      (grep: no `case '{'` / `case '}'` in `internal/testing/`).
- [ ] All existing strip/batch/parity tests pass unchanged:
      `TestStripNonPureFunctions_*`, `TestNamedBatch_*`,
      `TestEngineParity_*`, `TestTestCommandBytecodeFlags`.
- [ ] All tests passing (`make test-core` + `go test ./internal/testing/
      ./internal/parser/ ./internal/format/ ./cmd/ailang/`).
- [ ] CHANGELOG entry added; this doc moved to implemented on ship.

## Conflict Surface

This change touches `internal/parser/` and `internal/ast/` (the mandatory
sections apply), but it extends **no grammar production and no token
position**: it records a position the parser already stops at.

### Syntactic positions touched

- `parseTestDecl`'s return point (parser_test_decl.go:65-73): the closing
  `RBRACE` of `test "name" { … }`. The parser is already AT that token by
  convention (comment at line 65); we read `p.curPos()` and return it in a
  new field.
- `parsePropertyDecl`'s return point (parser_test_decl.go:111-120): the
  closing `RBRACE` of `property "name" { forall(…) => … }` (same convention
  at line 111).
- AST shape of `*ast.TestDecl` / `*ast.PropertyDecl`: one additive field.

### What else lives here

| Position | Existing valid form | Shape |
|----------|--------------------|-------|
| Body of a test block | any expression list | `test "…" { <expr> (; <expr>)* }`, where `<expr>` may contain braces in string literals (`"}"`), in comments (`-- }`), in interpolation (`"${…}"`), and in `match`/record/block expressions (real braces) |
| Test name | any string literal | `test "<string>" {`, where `<string>` may itself contain `{`/`}` (verified trigger V4) |
| After the body | closing `}` then EOF or the next top-level decl | the recorded `End` is exactly this token's line |

Real braces from `match`/records/blocks inside bodies balance under both the
old scan and the new span — they were never broken. Only brace-like
characters *inside strings/comments/names* diverged, and only the parser can
tell those apart.

### Disambiguation strategy

None needed: no syntax or parse decision changes. The only semantic change is
which **lines** the test-harness strip removes, and that range becomes a
strict function of parser-recorded positions (deterministic by A1: same
source, same range, no scan state).

### Programs that MUST still work (regression fixtures — all exist, verified)

- `internal/testing/testdata/strip/named_test_multiline.ail` — multi-line
  test block + effectful func stripped by span (read; used by
  `TestStripNonPureFunctions_DeclarationRanges`).
- `internal/testing/testdata/strip/named_test_contract.ail`,
  `named_test_annotated.ail`, `named_test_effectful.ail`,
  `malformed_control.ail` — the existing strip table (read).
- `internal/testing/testdata/named_batch/mixed.ail` (7 test blocks) and
  `internal/testing/testdata/engine_parity/mixed.ail`, `sweep.ail` — batch
  and engine-parity fixtures whose test blocks must strip identically.
- `cmd/ailang/test_bytecode_test.go::TestTestCommandBytecodeFlags` — the
  `--bytecode`/`--strict-bytecode` end-to-end pin (read).
- External: the `stapledons-godot` test suite (29-test `protocol_test.ail`
  benchmarked in m-test-runner-compile-once) and the motoko_agent fork.

### What deliberately changes

Nothing that works today. For well-formed inputs the recorded range equals
the scanned range byte-for-byte (both end at the closing brace's line).
The only observable difference: previously-corrupted strips (unbalanced
braces in strings/comments/names) now strip correctly. No syntax is added,
removed, or reinterpreted; `ailang check` output is untouched — mechanism
verified from the import graph, not just the output (V2b): `cmd/ailang/`
imports `internal/testing` from exactly two non-test files (`test.go`,
`commands_language.go`); `cmd/ailang/check.go` imports neither the strip
nor the package, so the check path cannot read stripped source at all
(V2 additionally observed check passing on every trigger file).

## Testing Strategy

**Unit tests:**
- Parser: `End.Line` equals the closing-brace line for single-line and
  multi-line blocks, and for bodies with braces in strings/comments.
- Strip (`internal/testing/source_strip_test.go`): new fixture
  `testdata/strip/brace_literals.ail` with the three trigger classes; assert
  exact removal `[Pos.Line, End.Line]` and the unchanged `lineMap` for the
  lines that remain; hand-built `TestDecl`/`PropertyDecl` with `End` unset
  exercises the guard (mirroring
  `TestStripNonPureFunctions_InvalidSpanFallsBackToPosition`, whose
  both-disjuncts table is the precedent); existing
  `TestStripNonPureFunctions_*` table passes unchanged.

**Integration tests:**
- End-to-end `ailang test` (new `cmd/ailang/test_strip_braces_test.go`):
  trigger file + one clean sibling test → all four report their own outcomes;
  `--bytecode` and `--strict-bytecode` runs agree (the strip feeds
  `evalNamedTestBodyOnVM` via `baseSource`); assert the run prints **no**
  `could not share one compile` notice and no harness-bug notice.

**Regression-surface tests:** one per "Programs that MUST still work" fixture
above — they are already run by
`TestStripNonPureFunctions_*`/`TestNamedBatch_*`/`TestEngineParity_*`/
`TestTestCommandBytecodeFlags`; the sprint must confirm each still passes
without text changes (per the m-arity-style-diagnostic lesson: do not plan
test-text edits without reading the assertions — read, and they assert
behavior, not strings).

**Manual testing:**
- Reproduce the reporter's scenario: the three trigger files from this doc's
  Problem Statement against a built binary; `ailang check` passes before and
  after; `ailang test` goes from all-fail to per-test outcomes.

## Verification Log

Live verification on the current tree (binary `ailang v0.52.5`, commit
`7200786`, built 2026-10-07; workspace `v0.53.0-dev @ f974e8a9`):

- **V1 — the reporter's exact snippet passes on HEAD.**
  `test "x" { contains(reply(newSession(), helloLine(7)), "\"minor\":7") }`
  in a minimal module: `ailang check` exit 0, `ailang test` — 1 passed. The
  escaped-quote printing (`PrintAILANGSource`'s `StringLit` branch,
  `executor_helpers.go`) round-trips `\"` correctly. The report's mechanism
  hypothesis ("mangles escaped quotes") is therefore refuted for HEAD; the
  live defect matching every reported symptom is the brace scan below. What
  v0.52.0 did with this snippet in isolation cannot be re-verified (that
  commit is not in this workspace's history), but in the reporter's *file*
  context the symptom is fully explained by any sibling body/name/comment
  with an unbalanced brace — "every test in the file" is the strip's
  blast radius, not the snippet's.
- **V2 — `}` inside a string literal in a test body ⇒ every test in the file
  fails with parse errors against `ailang-namedtest-NNN`, `ailang check`
  passes** (transcript in Problem Statement; both the batch and every
  per-body fallback compile fail, so the harness-bug notice fires).
- **V2b — `ailang check` never touches the strip**: `grep -rln
  "internal/testing" cmd/ailang/*.go` → `commands_language.go`, `test.go`
  (plus test files); `cmd/ailang/check.go` has no such import and no strip
  reference — the check path reads the user's file through the parser only.
  Mechanism read from the import graph, not inferred from V2's output.
- **V3 — `}` inside a `--` comment in a test body ⇒ same failure** (verified).
- **V4 — `}` inside the test's name string ⇒ same failure** (verified).
- **V5 — balanced braces in strings are NOT a trigger** (`"{\"minor\":7}"`
  body passes) — scoping the fix to unbalanced brace-like characters inside
  strings/comments/names.
- **V6 — only the parser constructs `TestDecl`/`PropertyDecl` in non-test
  code**: `grep -rn "ast.TestDecl{\|ast.PropertyDecl{" internal/ cmd/` → no
  non-test hits; strip consumers are exactly `stripWithLineMap`'s two callers
  (`stripNonPureFunctions`, named/property batch). `TestDecl` carries no
  end/Span today (read `internal/ast/ast_decl.go:144-164`) — the negative
  this design relies on.
- **V7 — the parser already stops at the closing `RBRACE` and discards the
  position** (read `parser_test_decl.go`: both parse functions end with
  "Leave parser AT the RBRACE"); `FuncDecl.Span` shows the exact capture
  pattern to copy (`endPos := p.curPos()` at the closing brace,
  `parser_func.go:230-231,323-324`), and `functionSkipRanges`' invalid-span guard is
  the precedent for hand-built ASTs.
- **V8 — the strip feeds both engines**: `evalNamedTestBodyOnVM`
  (`bytecode_engine.go:103-115`) compiles `baseSource` (from
  `stripNonPureFunctions`, `executor.go:198`) through the same
  `runNamedTestPipeline`; the default evaluator path does the same at
  `executor.go:260`. One strip, one fix.
- **V9 — no other character-level brace scanner exists in the test harness**
  (grep `case '{'`/`case '}'` over `internal/testing/`, `cmd/ailang/`:
  `source_strip.go` only) and **`internal/testing` does not import
  `internal/lexer` in non-test code** — triage option (b) would be new
  coupling, option (c) adds none.
- **V10 — related-doc search**: the script's SimHash/neural search found no
  implemented or planned design doc on this topic; the two relevant
  artifacts are triage rows (Related Documents below), and
  `ailang-core-triage/ai-deadline-and-named-test-roundtrip.md` already
  classified a 2026-10 report of this same mechanism as
  duplicate-of the source-strip triage row. Not a duplicate of
  `m-test-runner-compile-once` (that doc batches compiles on top of the
  strip; it neither caused nor covers this defect — though its "one strip
  for the whole file" design is why the blast radius is every test).
- **V11 — error-code claim**: none proposed; no new `PAR*`/`MOD*` code.
- **V12 — fixtures cited exist** (all `ls`-verified; test assertions read, not
  assumed): `testdata/strip/*` (7 files), `testdata/named_batch/mixed.ail`,
  `testdata/engine_parity/{mixed,sweep}.ail`, and the end-to-end test
  functions named in Testing Strategy.

## Deferred Decisions

The following are intentionally left open for the implementer:

- AST field shape — `End Pos` vs `Span Span` (mirroring `FuncDecl`) — agent
  may choose; both additive.
- Whether the parser test goes in a new `parser_test_decl_test.go` or the
  existing parser test files — agent may choose.
- Exact fixture naming/content for `testdata/strip/brace_literals.ail` —
  agent may choose, provided it covers all three trigger classes.
- Whether `stripWithLineMap`'s now-unused `sourceLines` parameter is dropped
  from the unexported helper chain in the same change — agent may choose
  (preferred: yes, it is unexported).

## Non-Goals

**Not attempted in this feature:**
- Teaching `PrintAILANGSource` anything new — its string re-escaping is
  correct (V1); no printer change.
- Inline `tests [...]` tables — they are not stripped textually (no
  `TestDecl` block); nothing to fix.
- Any lexer or grammar change — the parser already ends at the right token.
- Making the batch compile resilient to a corrupted strip (e.g. diffing
  stripped output) — the right fix is to stop corrupting it; belt-and-suspenders
  parsing of the stripped base would be a silent-fallback anti-pattern.
- The v0.52.0-only possibility that the reporter's isolated snippet also
  failed for a second, since-fixed printer reason — unverifiable from this
  workspace (commit `bf2436a` not in history); if it existed, it is already
  fixed (V1) and needs no design.

## Timeline

**Day 1** (~8 hours):
- Phase 1: AST field + parser capture + parser tests (~2h)
- Phase 2: strip rewrite + guard + strip tests (~3h)
- Phase 3: end-to-end regression, full suite, CHANGELOG (~3h)

**Total: ~1 day** (2× the naive ~4h estimate per project convention).

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| A hand-built `TestDecl` (Go API consumer, future test harness code) reaches the strip with `End` unset | Med | Guard mirrors `functionSkipRanges`' existing `endLine == 0 \|\| endLine < startLine` fallback; unit test pins both disjuncts (precedent: `TestStripNonPureFunctions_InvalidSpanFallsBackToPosition`) |
| `End` is wrong on some parser recovery path, mis-stripping valid lines | Med | `parseTestDecl`/`parsePropertyDecl` return `nil` on every error path today (read) — `End` is only set on full success; end-to-end + existing fixtures pin well-formed ranges |
| AST field addition breaks a JSON/golden snapshot | Low | `ast.Compact` marshals via `simplify`; additive fields flow through; `internal/format` re-prints from `Name`/`Body` (read `decl.go:528`); goldens re-baselined if any snapshot the field |
| Scope creep into lexer/comment policy | Low | Non-Goals fence it; no lexical code is touched at all |

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | The strip range becomes a pure function of parser-recorded positions; the textual scan's outcome depended on brace-like characters in string contents — same source, same answer either way, but the new one cannot diverge from what the parser actually parsed |
| A2: Replayability | 0 | No trace/replay impact |
| A3: Effect Legibility | 0 | No effect changes |
| A4: Explicit Authority | 0 | No capability changes |
| A5: Bounded Verification | +1 | `ailang check` and `ailang test` now agree on every valid file (they share the parser); a whole-suite verification path stops being silently unusable |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | AI-generated test suites (JSON/protocol fixtures are exactly what models write) stop failing with `PAR_NO_PREFIX_PARSE` at a temp file the model never wrote and cannot map back to its code |
| A8: Minimal Syntax | +1 | Deletes a hand-rolled lexical approximation (~30 LOC) instead of adding syntax or parallel lexical rules |
| A9: Cost Visibility | 0 | No resource changes |
| A10: Composability | +1 | The batch compile, per-body fallback, evaluator and VM paths all consume the same corrected strip; the fix composes with m-test-runner-compile-once rather than patching around it |
| A11: Structured Failure | +1 | Removes a misleading failure class (parse errors blaming synthesized files) and the spurious "harness bug" report it triggers; failures return to being per-test and pointing at the user's real source lines |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +6** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced — the new range is a pure function of the parse
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Fixes a machine-facing failure mode (misattributed parse errors on generated test files); optimizes for machine analysis, not convenience

## Related Documents

<!-- Auto-search (SimHash + neural) found no implemented/planned design docs on
     this topic; the artifacts below were located by reading the triage lane. -->

**Triage (the row that recommended this doc):**
- `design_docs/planned/ailang-core-triage/source-strip-string-brace-skip-ranges.md` — 2026-09-15 triage of this exact defect; lists the three fix options and the ~20-40 LOC estimate this design superseded with the parser-recording option
- `design_docs/planned/ailang-core-triage/ai-deadline-and-named-test-roundtrip.md` — independently reproduced the same mechanism (a later sibling's `"a{b"` fails the first test) and classified it duplicate-of the row above

**Planned (adjacent, not overlapping):**
- `design_docs/planned/v0_53_0/m-test-runner-compile-once.md` — the batch compile built on this strip; its D1 fallback and harness-bug notice are how this bug now announces itself

**Implemented (context):**
- `design_docs/implemented/v0_29_0/m-named-test-blocks.md` — the test-block lowering pipeline this strip feeds
- `design_docs/implemented/v0_51_0/m-vm-pure-builtin-coverage.md` — `PrintAILANGSource`'s literal handling (the printer this bug was first blamed on)

## References

- [Design Axioms](/docs/references/axioms) — the 12 non-negotiable principles
- Reporter: AILANG v0.52.0, `stapledons-godot sim/protocol_ism_test.ail` (branch `sprint/ism-dust`); every test in the file failing with `pipeline error: … ailang-namedtest-NNN … parse errors`, `ailang check` passing
- Pattern precedent: `FuncDecl.Span` capture at the closing brace (`internal/parser/parser_func.go:231,324`); invalid-span guard (`internal/testing/source_strip.go:functionSkipRanges`; `TestStripNonPureFunctions_InvalidSpanFallsBackToPosition`)

## Future Work

- Consider recording `End` positions for every declaration node as a sweep
  (only `FuncDecl` has one today) if another consumer needs block extents —
  not while the strip is the only consumer.
- The harness-bug detector (`finishNamedTests`) worked exactly as designed
  here; no change needed, but its notice could link to the docs page for
  known-fixed classes once this ships.

---

**Document created**: 2026-10-09
**Last updated**: 2026-10-09
