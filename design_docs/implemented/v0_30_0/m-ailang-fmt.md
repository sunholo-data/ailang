# M-AILANG-FMT: `ailang fmt` — Canonical AILANG Source Formatter

**Status**: **IMPLEMENTED (Phase 1) 2026-07-18** — M1–M4 landed on branch `sprint/m-ailang-fmt`
(commits ab98f4b98, 2fd01036c, 3021c6f23, d5fa52606). All acceptance criteria met; `make test`,
`make verify-examples`, and `make check-file-sizes` green; `internal/ast/print.go` untouched. Phase 2
comment preservation remains separately scheduled. Was: **GREENLIT by Mark 2026-07-18** — routed
sprint-planner (opus) → executor (opus) → evaluator (sonnet). Authored by `codex:gpt-5.6-sol`;
quorum `gemini-3-1-pro` + Opus controller.
**Target**: v0.30.0
**Priority**: P1 (canonical presentation for an intentionally forgiving syntax)
**Estimated effort**: Phase 1: 4 days; Phase 2 comment preservation: separately scheduled, estimated 2–3 days
**Dependencies**: Implemented [M-SYNTAX-AI-FORGIVING](../../implemented/v0_30_0/m-syntax-ai-forgiving.md)
**Split from**: [v0.29.0 formatter stub](../v0_29_0/m-ailang-fmt.md), itself split from M-SYNTAX-AI-FORGIVING discrepancy D1

## Problem Statement

AILANG intentionally accepts several spellings for statement sequencing. Live checks on 2026-07-18
confirm that an equation body with semicolons, a braced semicolon sequence, a braced newline sequence,
and a `let ... in` chain all check successfully. The existing parser structural tests confirm that
the first three separator variants produce equivalent ASTs after positions are ignored. A direct
`astdump` probe found an important correction to the supplied premise: explicit `let ... in` is a
nested `ast.Let` with non-nil `Body`, while statement-sequence lets are sibling block expressions
with nil `Body`; it is valid but structurally distinct and must remain explicit. There is no `fmt`
subcommand, so repositories and generated programs can drift among equivalent separator spellings.

The formatter is a developer and agent convenience, **not a correctness gate**. The landed corpus
AST-diff fuzz gate already protects parser compatibility and presentation-independent execution.
`fmt` adds one stable source representation; it does not make alternate accepted spellings invalid.

The apparent shortcut, `internal/ast.Print`/`PrintProgram`, is not usable for this feature. That code
emits normalized JSON for golden snapshots, deliberately removes positions, and has a stable testing
contract. AILANG needs a new AST-to-source printer.

Comments define the phase boundary. The lexer currently skips comments and the AST has no comment
or trivia fields. A naive AST reprint would silently delete user text. Phase 1 therefore formats
**comment-free files only and refuses commented input before parsing or writing**. Lossless comment
attachment is designed below as Phase 2 rather than compressed into this sprint.

## Goals

1. Add deterministic `ailang fmt [--write] [--check] <files...>` formatting for comment-free AILANG.
2. Emit one canonical statement-sequence form: newline-per-expression inside `{ }`, with the final
   expression bare and no trailing semicolon.
3. Guarantee idempotence: `fmt(fmt(x)) == fmt(x)` byte-for-byte.
4. Guarantee structural round-trip: parsing formatted output yields the same AST as the input after
   source positions and formatting-only metadata are ignored.
5. Provide a non-writing `--check` mode suitable for CI.
6. Fail closed on comments, parse errors, unsupported AST nodes, and write failures.

## Non-Goals

- No change to the grammar or to any spelling accepted by the parser.
- No `--strict-syntax` change and no requirement that checked source already be canonical.
- No semantic rewrites: no constant folding, import sorting, declaration sorting, dead-code removal,
  alpha-renaming, inferred annotation insertion, or desugaring based on type information.
- No formatter participation in compilation, evaluation, code generation, or trace generation.
- No reuse or modification of `internal/ast/print.go`'s deterministic JSON contract.
- No silent comment deletion. Commented files are a Phase 1 hard error and remain byte-identical.
- No stdin, directory recursion, configuration file, line-range formatting, or editor/LSP integration
  in Phase 1.

## Solution Design

### Canonical Form

The formatter emits source according to these rules:

1. **Sequences use braces and newlines.** An AST block with multiple expressions prints as:

   ```ailang
   {
     let x = 1
     let y = 2
     x + y
   }
   ```

   Each non-final expression occupies its own line. The final value expression is bare. No line has
   a sequence semicolon. This matches the canonical-form guidance in
   `prompts/agent/dialect-traps.md`, is easy for generated code to extend, and avoids the visual
   ambiguity of long same-line semicolon chains.

2. **Single-expression function bodies use `= expr`.** Multi-expression `ast.Block` bodies use a
   braced newline block. Nested blocks follow the same rule. A one-expression block wrapper created
   by parser normalization does not force braces unless the surrounding grammar requires them.

3. **`let ... in` remains explicit.** The AST retains the distinction needed here: statement lets
   inside `ast.Block.Exprs` have nil `Body`, while explicit let-in expressions have non-nil `Body`.
   The systemic rule is to print nil-body lets as newline statements in their enclosing sequence and
   non-nil-body lets as `let name = value in body`. No source-spelling side table is required.

4. **Layout is fixed.** Two-space indentation; LF output; exactly one final newline; one blank line
   between top-level declarations; module first, then imports in original AST order, then declarations
   in original AST order. Imports, symbols, cases, fields, and declarations are not reordered.

5. **Parentheses are precedence-driven.** Every expression printer receives the parent precedence
   and syntactic position. It adds parentheses whenever omission could change the parsed AST. It may
   retain redundant parentheses only if the AST represents them; because the current AST has no
   parenthesis node, Phase 1 removes source-only redundant parentheses.

6. **All literals are escaped canonically.** String and character emission must use one escaping
   routine and must never reinterpret quasiquote template contents. Unsupported or malformed literal
   payloads fail rather than falling back to a debug `String()` method.

### Printer Architecture

Add a new `internal/format` package with a deliberately small API:

```go
type Options struct {
    Indent string
}

func Source(program *ast.Program, options Options) ([]byte, error)
func HasComments(source []byte) (bool, error)
```

`Source` is an exhaustive visitor over the source AST: file/module/import nodes; declarations;
expressions; types; patterns; effects; annotations; tests; and properties. The implementation uses a
token/document builder with indentation, hard-line, soft-line, and grouping primitives rather than
concatenating ad hoc strings. Each node printer returns an error for nil-required children,
`ast.Error`, or an unknown concrete interface implementation. It must not call existing AST
`String()` methods as a fallback because those methods are debugging renderings and are neither
complete nor precedence-safe source.

The printer consumes only parsed AST plus fixed options. It does not consult the type checker,
elaborator, filesystem, environment, or wall clock. Output therefore depends only on the AST.

`HasComments` is a lossless lexical preflight, not a substring search: it distinguishes comment
introducers from string, character, regex, and quasiquote contents. Phase 1 invokes it before parse
and rejects a file if any source comment is present. This preflight may be implemented by adding an
optional trivia-emitting scan mode to `internal/lexer`; it must not change `Lexer.NextToken()` output
used by the parser.

### Idempotence and Round-Trip

For every formatter-eligible source `x`:

```text
bytes(Source(Parse(Source(Parse(x))))) == bytes(Source(Parse(x)))
AST(Parse(Source(Parse(x)))) == AST(Parse(x))
```

AST equality ignores positions, spans, file paths, and later trivia attachment metadata. Tests use
an independent structural comparison (`go-cmp` with explicit ignored metadata), not textual equality
of debug `String()` methods. `internal/ast.PrintProgram` may be used as an additional regression
oracle in tests, but the source printer must not depend on or modify it.

Round-trip failures are formatter defects. There is no fallback to original source in stdout mode.
`--write` validates the full input set before writing and replaces each individual file atomically.

### Comment and Trivia Model

#### Phase 1: Safe Deferral

Phase 1 is intentionally comment-non-preserving but **not comment-destructive**:

- Detect any source comment before parsing.
- Print `path: comments are not yet supported by ailang fmt` to stderr.
- Return exit 2.
- Emit no formatted stdout for that file and perform no write.
- In multi-file `--write`, preflight and format every input into memory before replacing any file, so
  one commented or invalid file leaves the entire invocation unchanged.

This keeps the formatter useful for generated/comment-free code while preserving user text and the
four-day sprint cap.

#### Phase 2: Lossless Attachment

Phase 2 adds trivia without changing grammar:

```go
type Comment struct {
    Kind       CommentKind
    Text       string
    Start, End ast.Pos
}

type Attachment struct {
    Comment Comment
    Owner   ast.Node
    Place   Leading | Trailing | Floating
    Index   int // boundary within an owner's ordered child list
}
```

The lexer exposes comments and exact byte spans through a separate lossless token stream. Parser AST
nodes gain complete source spans, including modifiers and delimiters, or a parallel syntax-envelope
index supplies those ranges without changing semantic nodes. Attachment is deterministic:

1. A comment after a node on the same source line attaches **trailing** to the nearest preceding node.
2. A comment before the next node with no blank line attaches **leading** to that next node.
3. A comment separated by a blank line, between sibling nodes, or immediately before a closing
   delimiter attaches **floating** to the smallest enclosing ordered list at the boundary before the
   next child (or at `len(children)` before the close).
4. Comments before the module attach to the file at boundary zero; comments after the final top-level
   node attach to the file at the final boundary.
5. Consecutive comments preserve source order and relative blank-line grouping.

Emission order is leading comments, node, same-line trailing comments, then boundary-floating
comments. A Phase 2 property test labels every input comment with a unique marker and proves the
formatted output contains every marker exactly once and remains idempotent. Until that phase lands,
the Phase 1 refusal is mandatory and cannot be weakened to a warning.

### CLI Contract

```text
ailang fmt <file.ail>
ailang fmt --write <files...>
ailang fmt --check <files...>
```

- Default mode accepts exactly one file and writes canonical source to stdout.
- `--write` accepts one or more files and atomically replaces each file only after all inputs have
  passed comment preflight, parsing, formatting, and round-trip verification in memory.
- `--check` accepts one or more files, writes each non-canonical path to stdout, and never modifies
  files. Canonical means byte-equal to formatter output, including final newline.
- `--write` and `--check` are mutually exclusive. Zero files, multiple files in stdout mode, or both
  flags is a usage error.
- Inputs are processed in argument order; diagnostics are deterministic and include the path.
- No shared safe-write helper exists. Phase 1 implements a small unexported helper in `cmd/ailang`
  that matches the repository's ad-hoc convention: create a temporary file in the target directory,
  preserve the original file mode, then atomically replace the path with `os.Rename`.

Exit codes:

| Code | Meaning |
|---:|---|
| 0 | Formatting succeeded; or every file is canonical in `--check` mode |
| 1 | `--check` found at least one non-canonical file and no operational error occurred |
| 2 | Usage, read, comment-preflight, parse, print, round-trip, or write error |

No new diagnostic code is introduced. Human-readable path-qualified errors are sufficient for this
CLI-only presentation feature; structured compiler diagnostics remain unchanged.

## Milestones

### M1: Exhaustive AST-to-Source Printer — 2 days

- Add `internal/format` document builder and precedence-aware printers.
- Cover every concrete declaration, expression, type, pattern, effect, test, and property node in
  `internal/ast`; unknown nodes fail loudly.
- Lock the canonical sequence, whitespace, parentheses, literal, and final-newline rules with goldens.

### M2: CLI and Transactional File Handling — 0.5 day

- Add `fmt` dispatch and help text in `cmd/ailang`.
- Implement stdout, `--write`, `--check`, mutual exclusion, argument validation, and exit codes.
- Buffer and verify every file before any multi-file write; implement the `cmd/ailang` unexported
  same-directory temporary-file + `os.Rename` helper, preserving each original file mode.

### M3: Idempotence and Corpus Round-Trip — 1 day

- Add property tests for byte idempotence and structural AST round-trip.
- Run over every comment-free `examples/**/*.ail` file and parser formatter fixture.
- Partition commented examples explicitly: detect them, assert exit 2, and assert unchanged bytes.
- Add focused fixtures for the three equivalent separator spellings and require identical formatted
  text; separately prove explicit `let ... in` remains explicit and structurally round-trips.

### M4: Comment Safety Gate and Documentation — 0.5 day

- Add lossless comment detection without changing parser token behavior.
- Add leading, trailing, floating, string-contained marker, and quasiquote-contained marker tests.
- Document Phase 1 refusal and Phase 2 attachment model in CLI help/guides.

**Phase 1 total: 4 days.** Phase 2 comment preservation is a separate 2–3 day sprint because exact
attachment requires lossless trivia plus reliable full-node ranges; it is not hidden inside M4.

## Conflict Surface

This feature is parser/AST-adjacent but presentation-only.

| Area | Relationship and constraint |
|---|---|
| `internal/ast` | **Read only** for Phase 1 source printing. The formatter walks current node shapes. `print.go` is untouched. Phase 2 may add non-semantic span/trivia envelopes under a separate design/implementation review. |
| `internal/format` | **New package.** Owns canonical source emission, precedence, layout, comment preflight API, and formatter tests. |
| `internal/lexer` | Phase 1 may add an opt-in lossless comment scan. Default `NextToken()` behavior and parser-visible tokens must remain byte-for-byte behaviorally unchanged. |
| `internal/parser` | Used to parse input and reparse output for verification. Accepted grammar, separator rules, AST construction, and diagnostics do not change. Existing R1/R2 equivalence and corpus AST-diff tests remain green. |
| `cmd/ailang` | Adds the `fmt` subcommand, local flags, help, deterministic diagnostics, exit codes, safe file orchestration, and the owned unexported atomic-write helper that stages a same-directory temporary, preserves the original mode, and calls `os.Rename`. No shared helper is reused because none exists. |
| `internal/ast/print.go` | **Forbidden modification.** Its normalized JSON output is a golden-test contract, not source formatting infrastructure. |
| evaluator/type/effects/codegen/runtime | No dependency and no modification. Formatting never enters semantic compilation paths. |

The semantic invariant is `Parse(fmt(x)) ≡ Parse(x)` for eligible input. Evaluation and trace output
therefore remain determined by the same AST. Axiom A1 is untouched: `fmt` changes presentation only
and does not introduce execution or trace nondeterminism.

## Acceptance Criteria

- [x] `ailang fmt file.ail` emits canonical source to stdout and leaves the file unchanged.
- [x] `ailang fmt --write a.ail b.ail` changes no file if preflight, parse, print, or round-trip
      validation fails for any input; each subsequent file replacement is individually atomic and
      preserves file mode. Cross-file crash atomicity is not claimed.
- [x] `ailang fmt --check` exits 0 when all inputs are canonical, exits 1 and lists drifted paths when
      formatting differs, and exits 2 on operational/usage errors; it never writes.
- [x] An idempotence property test proves `fmt(fmt(x)) == fmt(x)` over generated AST cases and the
      formatter-eligible example corpus.
- [x] A structural round-trip test proves `Parse(fmt(x)) == Parse(x)` over every comment-free example
      and dedicated fixture, ignoring only positions/spans/file paths.
- [x] Equation-semicolon, braced-semicolon, and braced-newline fixtures format to the same
      newline-per-statement braced form; explicit `let ... in` formats as `let ... in`.
- [x] Every commented example is detected before formatting, exits 2, and remains byte-identical;
      marker tests distinguish actual comments from comment-like text in literals/quasiquotes.
- [x] No parser-accepted grammar is removed or restricted; existing syntax-forgiving and corpus
      AST-diff tests pass unchanged.
- [x] `internal/ast/print.go` and its JSON golden output are byte-for-byte untouched.
- [x] Every concrete AST node kind has either a source printer test or an explicit unsupported-error
      test; there is no debug `String()` fallback.
- [x] `make test`, `make fmt`, and the repository's example verification target pass.

## Risks and Mitigations

| Risk | Mitigation |
|---|---|
| Missing parentheses changes the AST | Parent-precedence API; structural reparse property; operator-focused generated tests |
| An AST node silently prints invalid/debug syntax | Exhaustive type switches, unknown-node error, no `String()` fallback, node coverage table in tests |
| Multi-file `--write` encounters a late filesystem failure | Validate all files first; stage same-directory temporaries; replace each file atomically; report that cross-file crash atomicity is not provided |
| Comment detection mistakes `--` inside a literal for trivia | Lossless lexer state machine plus literal/quasiquote fixtures; never use substring matching |
| Phase 1 surprises users by refusing normal commented code | Explicit help/error text; no data loss; Phase 2 is separately estimated and specified |
| Existing AST spans are insufficient for Phase 2 floating comments | Do not guess attachment from start positions; require full ranges or a parallel syntax-envelope index |

The most important open risk is printer completeness: the AST surface is broad, and one precedence,
literal, pattern, or declaration omission can break round-trip even though simple sequence examples
look correct. M1 coverage plus M3 corpus/property tests are the release gate.

## Implementation Plan

- **Day 1–2:** M1 printer, precedence model, canonical-layout goldens, exhaustive node audit.
- **Day 3 morning:** M2 CLI and transactional writes.
- **Day 3 afternoon–Day 4 morning:** M3 idempotence and corpus round-trip harness.
- **Day 4 afternoon:** M4 comment refusal, literal-disambiguation tests, help/docs, full validation.

Phase 2 begins only under a follow-up sprint that implements and tests the attachment model above.
Phase 1 must not be described as comment-preserving.

## Related Documents

- [M-AILANG-FMT v0.29.0 stub](../v0_29_0/m-ailang-fmt.md) — **split from / superseded by this
  document.** Carries the original scope, non-goals, separator recommendation, and comment question.
- [M-SYNTAX-AI-FORGIVING](../../implemented/v0_30_0/m-syntax-ai-forgiving.md) — parser change that
  intentionally accepts the equivalent separator spellings; its corpus AST-diff gate remains the
  correctness protection and is not replaced by formatting.
- [`prompts/agent/dialect-traps.md`](../../../prompts/agent/dialect-traps.md) — current guidance for
  accepted statement separators and the bare final expression rule.

## Verification Log

All commands were run on 2026-07-18 in worktree HEAD `3b77bc036`. The required installed binary
reported `AILANG v0.29.2-362-g6896bcf5e` (commit `6896bcf`) and printed its stale-source warning on
each invocation; results below are recorded exactly rather than treating the warning as absent.

| # | Command / inspection | Observed result |
|---|---|---|
| V1 | `~/go/bin/ailang fmt --help` | Exit 1; stderr ends `Error: unknown command 'fmt'`. No formatter command exists. |
| V2 | `ailang check` on `export func main() -> int ! {} = let x = 1; let y = 2; x + y` | Exit 0; `✓ No errors found!` (plus expected temporary-path MOD010 warning). |
| V3 | `ailang check` on `export func main() -> int ! {} { let x = 1; let y = 2; x + y }` | Exit 0; `✓ No errors found!` (plus temporary-path warning). |
| V4 | `ailang check` on a braced body with the three expressions on separate lines and no semicolons | Exit 0; `✓ No errors found!` (plus temporary-path warning). |
| V5 | `ailang check` on `export func main() -> int ! {} = let x = 1 in let y = 2 in x + y` | Exit 0; `✓ No errors found!` (plus temporary-path warning). |
| V6 | `go test ./internal/parser -run 'TestR1_EqBodyMatchesBracedBody\|TestR2_NewlineMatchesSemicolon' -count=1` | `ok github.com/sunholo-data/ailang/internal/parser`; existing tests compare ASTs with `go-cmp` while ignoring `ast.Pos`/`ast.Span`. |
| V7 | Read `internal/parser/syntax_ai_forgiving_test.go` | `TestR1_EqBodyMatchesBracedBody` proves equation and braced semicolon bodies are structurally equal; `TestR2_NewlineMatchesSemicolon` proves newline and semicolon blocks are structurally equal across block parser paths. This grounds equivalence of the three separator spellings. |
| V8 | `go run ./cmd/astdump` on the semicolon chain and explicit `let ... in` chain | **Premise correction:** dumps differ beyond positions. Semicolon form is a 3-expression `ast.Block` with nil-body lets; let-in form is a 1-expression block containing nested non-nil-body `ast.Let` nodes. The design therefore preserves explicit `let ... in`. |
| V9 | Read `internal/ast/print.go` | `Print`/`PrintProgram` JSON-marshal `simplify(...)`, normalize paths, omit positional metadata, and document golden snapshot use. It is not source output. |
| V10 | Read `internal/ast/ast.go`, `ast_decl.go`, `ast_expr.go`, `ast_type.go`, `ast_patterns.go` | AST contains source constructs but no comment/trivia field. The printer must cover declarations, expressions, types, patterns, effects, tests, and properties. |
| V11 | Read `internal/lexer/lexer.go` and `lexer_test.go` | `NextToken` calls `skipComment()` and recursively returns the next token; `TestComments` explicitly expects comments to be skipped. `COMMENT` is allocated in `token.go` but is not emitted on the parser path. |
| V12 | `ailang check` on a file with a leading `--` comment and trailing `--` comment | Exit 0; `✓ No errors found!`; confirms comments are valid input while V10–V11 confirm they are not retained in AST/parser tokens. |
| V13 | `grep -rn -- '--write' internal/ cmd/ailang/` | Only existing `--write-env-snapshot` references; exact `--write` formatter flag is unallocated. |
| V14 | `grep -rn -- '--check' internal/ cmd/ailang/` | One JavaScript template reference to `node --check`; no AILANG CLI `--check` flag allocation. |
| V15 | `grep -rn 'case "fmt"' cmd/ailang/` | No output; command name is unallocated in dispatch. |
| V16 | Read `cmd/astdump/main.go` and `internal/parser/corpus_astdiff_test.go` | `astdump` is a deep deterministic AST comparison tool for parser corpus regression, not a source printer; formatter may reuse test strategy, not rendering output. |
| V17 | Read `internal/astedit/astedit.go` | Existing code explicitly identifies a faithful AST-to-source formatter as backlog and notes current parser spans lack byte offsets/full declaration boundaries, supporting the Phase 2 span risk. |
| V18 | Conflict-surface code read: `cmd/ailang/main.go`, parser entry path, lexer skip path, AST JSON printer | Confirmed routing: new CLI orchestration can parse into `internal/ast` and call a new `internal/format` package without changing parser grammar, semantic phases, or `print.go`. Axiom A1 remains unaffected because no formatted source is used during execution. |
| V19 | `grep -rn "os.Rename" internal/ cmd/ailang/` | No shared safe-write/atomic-write helper exists; the temporary-file + `os.Rename` pattern is inlined ad hoc at `internal/coordinator/heartbeat_file.go:114`, `internal/eval_analysis/dashboard_io.go:196`, `cmd/ailang/ext_registry_gen.go:240`, and `cmd/ailang/editor.go:246`. |
| V20 | `grep -rin "paren" internal/ast/*.go` (controller-added 2026-07-18, re-quorum round) | No `ast.ParenExpr`/parenthesis wrapper node exists; the only matches are two `// move to LPAREN` comments in `ast_decl.go:35` and `ast_expr.go:220`. Confirms Rule 5's premise: the AST does not retain source parentheses, so Phase 1 must reconstruct them precedence-driven and the exhaustive visitor needs no `ParenExpr` case. |

No new error/diagnostic code is proposed, so there is no diagnostic namespace allocation to reserve.

## Controller PARK-NOTE (2026-07-18, mission iteration 49)

**Status: GREENLIT — Mark 2026-07-18 (interactive session; supersedes this park banner).** The
one-line ratification the park asked for has been given; route to the sprint-planner.

**Provenance.** Authored by the rotation designer `codex:gpt-5.6-sol`; reviewed by a quorum of
`gemini-3-1-pro` (reject-by-default, provider-independent of the OpenAI author) + the Opus controller.
The controller passed BOTH rounds; gemini rejected BOTH rounds.

**Both gemini objections were verification-completeness nits, not design defects** — each a TRUE
negative-existence claim that lacked a Verification Log row, surfaced one-per-round:
- R1 (blocked): the atomic-write path was left conditional ("safe-write helper *if one exists*").
  → **Resolved by revision R1**: hedge removed; V19 records the `os.Rename` grep (no shared helper;
  ad-hoc at 4 sites); atomic-write assigned to an owned unexported `cmd/ailang` helper.
- R2 (blocked): Rule 5's "the AST has no parenthesis node" was asserted but unproven.
  → **Controller-verified TRUE and recorded as V20 above** (grep finds no `ast.ParenExpr`; only two
  `// move to LPAREN` comments). This is exactly the fix gemini proposed.

With V19 + V20 both recorded, every claim the design relies on is now verified. The re-quorum-ONCE
guardrail (Standing rule 2 — never force it through) is why this parks rather than proceeds.

**Recommended unblock (one-line greenlight):** the design is sound and fully verified; ratify it and
route to sprint-planner. Alternatively, authorize one more revision+re-quorum round (low value — no
open design question remains, only gemini's reject-by-default finding the next unverified negative
claim). No architectural decision is required from the human; this is a ratification, not a fork.

**Systemic note (routed to Gate-5 retro):** the quorum's reject-by-default keeps finding negative-
existence claims ("no X node", "no shared helper") that are TRUE but lack a log row — one per round,
so a sound design ping-pongs. The design-doc-creator gate should require a Verification Log row for
EVERY negative-existence claim about internal structures (AST nodes, helpers, error codes) the design
depends on, not only for language-level "AILANG (un)supports X" claims.
