# M-AILANG-FMT-INLINE-INTERIOR: Stable Multi-Line Let-Chain Comment Boundaries

**Status**: **UNPARKED — Mark 2026-07-20 ("yes lets finish off ailang fmt")**: the R2 objection is
DATA-REFUTED for all 28 target cases (controller data-check, ⛔ Quorum Record) — proceed on the
data. Route to sprint-planner; no re-quorum. Part of the finish-fmt set with
m-fmt-properties-printer-roundtrip.

## ⛔ Quorum Record (mission iteration 67, 2026-07-20)

**Designer**: `codex:gpt-5.6-sol` (rotation slot after `claude:claude-fable-5`). **Reviewers**: `gemini-3-1-pro` (API, distinct provider) + controller (opus, in-session). **`gpt5-6-sol` EXCLUDED** from the reviewer set: it is the same model as the designer (`gpt-5.6-sol`) → generator≠judge; including it would be self-review. N-1 degrade recorded (never a silent pass). Metered: gemini R1 $0.0251 + R2 $0.0266 = **$0.0517** (well under the $0.10/reviewer cap and the $5 iteration ceiling).

**R1 (2026-07-20T03:58Z) — BLOCKED.** gemini objection: the design leans on the `*ast.Let` struct shape (`Body`/`Value` fields; no `in`-token position) but the Verification Log never inspected `internal/ast`. **Resolved (bounded revision):** controller independently ran `rg 'type Let struct' internal/ast` → `type Let struct { Name string; Type Type; Value Expr; Body Expr; Pos Pos }` (single `Pos`, no `in`-token position). Folded verbatim as **Verification Log V13** and cross-referenced in Stable Child Spans. Premise TRUE → additive evidence, not a design change.

**R2 (2026-07-20T03:59Z) — BLOCKED (bounded gate now CONSUMED).** gemini raised a NEW premise: the Problem Statement claims block-form `{ let x = v; let y = w; tail }` collapses into nested `*ast.Let` via `Body`, unproven; if such sequences are instead a `Block` statement-list, the `Body`-traversal flattening would "silently fail on 15 of the 28 targeted cases."

**Controller in-session DATA-CHECK of the R2 objection (opus, $0 metered) → LARGELY DATA-REFUTED for the 28 targets:**
- `parseBlockOrExpression` (`internal/parser/parser_expr.go:388–469`) builds a `Block{Exprs:[…]}` statement-list for **bare** `;`-separated multi-expression blocks, returning the single expr directly only when `len==1`. So gemini is CORRECT that a *bare*-`;` `{ let x=v; let y=w; tail }` would be `Block.Exprs`, NOT nested `Let.Body`.
- **BUT no target file uses that bare-`;` form.** Sampled 11 of the 28 (incl. all sub-classes): every one chains lets via **`let … in`** → nested `*ast.Let.Body`. Confirmed incl. the brace-*body* outlier `examples/reference/neural_semantic_search.ail`, which uses leading-`in` continuation (`let _ = … ⏎ in let _ = … ⏎ in …`) inside `{ … }` — still one nested `Let` chain (block `len==1` → returned directly). `examples/integer_literals.ail` (a "Block1(*ast.Let)" case): `let red = 0xFF in ⏎ let flags = … in ⏎ …` → nested `Let.Body`. The design's flattening traversal therefore HOLDS for the 28.
- **Real, minor doc defect:** the Problem-Statement sentence asserting the bare-`;` brace form *also* collapses into nested `*ast.Let` is inaccurate (it would be `Block.Exprs`) — immaterial to the 28 (none use it), but it should be corrected, and the sprint's M0 should DUMP the surface AST of all 28 to prove the `Let.Body` shape rather than assert it.

**Bounded gate CONSUMED (1 revision + 1 re-quorum) → `needs-human-review`.** Human fork (comment on the bookkeeping issue):
1. **[RECOMMENDED]** Route to **sprint-planner** — the R2 objection is data-refuted for all 28 target cases; the design is sound. Add an M0 mandate: (a) `ailang debug ast`/Go-test dump verifying the surface `*ast.Let.Body` shape for each of the 28, and (b) correct the Problem-Statement bare-`;` sentence (and add `Block.Exprs` handling only if any file actually needs it — none sampled does).
2. Authorize ONE more bounded quorum round to fold the per-file AST-shape verification into the doc, then re-quorum.
3. Keep parked.

Est ~1.5–2d impl, LOW risk/conflict (printer-local, read-only for parser/ast). Log entry 72.

---

**Original Status**: Planned — date 2026-07-20 — target version v0.30.0 — est ~1.5–2d

## Problem Statement

`ailang fmt` Phase 2 is lossless and honest: when a comment cannot be attached to a stable
printer boundary, formatting refuses fail-closed with exit 2, emits no output, and leaves
`--write` inputs byte-identical. The current corpus gate measures **59 refusals among 386
parse-valid files (15.28%)**, with zero comment loss and zero Phase-2 structural round-trip
regressions (V3, V8).

The refusal is raised in `internal/format/attach.go` when `attachOne` finds a comment whose source
line is strictly inside a multi-line child span but no tighter registered child list decomposes
that span. Moving such a comment to an outer boundary would make its location depend on the
previous formatting pass, violating `fmt(fmt(x)) == fmt(x)`, so the current implementation
correctly refuses rather than guesses.

The largest coherent refusal class is a parser/printer shape mismatch around `let ... in` chains:

- The parser collapses a `let ... in` continuation sequence — an equation body or a single-expression
  brace body containing sequential `let x = v in let y = w in tail` bindings — into nested `*ast.Let`
  expressions (each binding's `Body` is the next `*ast.Let`, terminating in the tail expression).
  (CORRECTION, M0 2026-07-20: the bare-`;`/newline block form `{ let x = v; let y = w; tail }` does
  NOT collapse into nested `*ast.Let` — `parseBlockOrExpression` returns `*ast.Block{Exprs:[…]}` when
  `len(exprs) > 1` (`internal/parser/parser_expr.go:461-468`). The 28 measured targets are exclusively
  the `let … in` continuation form: M0's `TestInlineInterior_LetChainSurfaceShape` proved all 28 reach
  a root `*ast.Let` with non-nil `Body`, 0 are `*ast.Block.Exprs`. `Block.Exprs` handling is therefore
  out of scope — no target needs it.)
- The Phase-1 printer's `letIn` emitter writes every binding and body inline as
  `let x = v in let y = w in tail`.
- The Phase-2 attacher registers only child lists that the printer emits multi-line. A nested let
  chain therefore has no registered boundary between bindings.
- A comment between two source bindings is inside the enclosing declaration/block child span, but
  is not attached to either binding. `attachOne` reaches its strict-interior guard and returns
  `comment-unattached`.

The live 59-file enumeration refines the original coarse `let-in-refusal` counter (V4–V5):

| Refusal class | Files | Share of 59 | Notes |
|---|---:|---:|---|
| Let-chain interior | **28** | **47.46%** | Largest construct class: 15 equation-form `Block1(*ast.Let)`, 6 bare function-body `*ast.Let`, 5 let-body chains, 2 top-level let values rooted at `*ast.Let` |
| Non-let single-expression equation body | **3** | **5.08%** | Minor secondary class: record update / nested block-shaped expression comments |
| Inline `tests [...]` list | 9 | 15.25% | String-built inline child list; separate follow-up |
| Trailing/footer with no enclosing registered list | 7 | 11.86% | Comments after the last anchored expression/declaration; separate follow-up |
| Other function-expression interior | 8 | 13.56% | Match, record, binary/call, or contract-adjacent interiors |
| Other AST child interior | 4 | 6.78% | Type declaration, match-arm `if`, nested block, top-level binary expression |
| **Total** | **59** | **100%** | Current fail-closed set |

This design deliberately targets the **28 measured let-chain refusals**, reducing the known
fail-closed rate from 59/386 (15.28%) to at most 31/386 (8.03%) if the corpus is otherwise
unchanged: a **47.46% relative reduction** in the refusal set. It does not claim that all 59 share
one implementation fix.

## Goals

1. Format every one of the 28 measured let-chain cases without deleting, duplicating, or
   non-idempotently relocating a comment.
2. Add stable, formatter-owned boundaries between nested `let ... in` bindings without changing
   the parser or AST.
3. Preserve Phase-1 output byte-for-byte for comment-free input.
4. Preserve structural round-trip: `Parse(fmt(x)) ≡ Parse(x)`, ignoring only positions/spans.
5. Preserve and extend idempotence: every newly accepted let-chain file must satisfy
   `fmt(fmt(x)) == fmt(x)`; `TestMarkerProperty_CorpusIdempotence` must remain green.
6. Keep the existing fail-closed behavior for every unsupported inline-interior shape.
7. Deliver within ~1.5–2 days through a printer-local, acceptance-gated change.

## High-Impact Decisions

### Decision 1: Recommend option (a), conditional multi-line let-chain emission

**Recommendation:** register nested let chains as conditional ordered child lists and emit a
canonical multi-line chain only when a comment attaches to one of those boundaries.

This is the smallest design that addresses the measured dominant construct class while preserving
the current safety posture. It changes formatting layout only for commented let chains that refuse
today. Comment-free programs and commented programs without a let-chain attachment retain the
existing Phase-1 layout.

### Decision 2: Do not weaken the strict-interior refusal guard

The `attachOne` guard is not a defect; it enforces the load-bearing idempotence invariant. The
implementation must add a tighter structural list before the guard runs. It must not special-case
the guard to float a comment to the file, block, or declaration boundary.

### Decision 3: The conditional list owner is the root `*ast.Let`

For a maximal nested chain:

```text
root = let x = vx in let y = vy in tail
```

the formatter flattens the chain to the logical sequence:

```text
[binding(x, vx), binding(y, vy), tail]
```

Attachments are keyed by the root `*ast.Let` and an index into this sequence. This uses existing
`Attachment`, `attKey`, leading/floating/trailing maps, and comment emission helpers. No AST node,
synthetic parser node, or public API is added.

### Decision 4: Multi-line layout is attachment-triggered, not source-layout-preserving generally

The formatter remains canonical rather than becoming a general source-layout preserver. A let
chain uses the existing inline Phase-1 spelling unless its root owns at least one comment
attachment. When it does, the entire maximal chain uses one canonical multi-line spelling. This
binary rule is deterministic and survives reparsing.

### Decision 5: Option (b), edit-grade parser spans, remains deferred

Byte offsets and complete declaration/expression boundaries would benefit `astedit`, LSP, future
refactors, and formatter attachment. They are nevertheless a cross-cutting parser/AST contract,
not a ~2-day formatter fix. The current AST has partial `Span` population, `astedit` converts
line/column positions and scans for missing delimiters, and LSP explicitly notes that many nodes
have only starts, not reliable ends (V12). Landing trustworthy byte spans requires a separate
design, migration, and corpus-wide premise gate.

## Deferred Decisions

1. **Non-let single-expression equation bodies (3 files):** retain fail-closed behavior. A later
   printer-local design may add a conditional one-child equation-body list, but it is not coupled
   to let-chain flattening.
2. **Inline `tests [...]` comments (9 files):** retain fail-closed behavior. These require explicit
   test-case child boundaries and a multi-line test-list printer policy.
3. **Footer/no-enclosing-list comments (7 files):** retain fail-closed behavior pending an anchor
   end-range design for top-level expressions and declarations.
4. **Other expression interiors (12 files):** retain fail-closed behavior and split by measured
   construct before designing additional conditional lists.
5. **Edit-grade parser spans:** option (b) is deferred as a larger parser/AST/LSP/`astedit` project.
6. **Whether `letrec ... in` should share the same conditional list:** defer until measured. The
   current 28-file target is ordinary `*ast.Let`; no corpus evidence justifies widening scope.

## Solution Design

### Overview

The change adds a formatter-owned **conditional let-chain child list**. It is conditional only in
emission: the attacher always discovers the logical chain so a comment can resolve to it, while the
printer emits it multi-line only if the attachment index contains a comment owned by that chain.

The data flow remains:

```text
source bytes
  -> token-anchored Envelope
  -> AttachComments (now includes let-chain lists)
  -> attachIndex
  -> precedence-aware printer
  -> parse + structural AST comparison in cmd/ailang
```

No parser, lexer token stream, type checker, evaluator, or runtime behavior changes.

### Let-Chain Discovery

`attacher.walkExpr` recognizes a `*ast.Let` with non-nil `Body` as a chain root unless it is already
being consumed as the `Body` link of an enclosing chain. It flattens consecutive body links:

1. Append each `*ast.Let` binding in source order.
2. Stop when `Body` is not another `*ast.Let`.
3. Append that terminal expression as the final logical child.
4. Register one `childList` owned by the root let.
5. Recurse into every binding value and the terminal expression to discover nested blocks,
   matches, and independent let chains.

Only maximal chains are registered. Registering every suffix would create overlapping lists with
equal semantic boundaries and make “smallest enclosing list” selection dependent on traversal
order.

### Stable Child Spans

The existing generic `addList` cannot be used unchanged because `subtreeEnd(bindingLet)` includes
the entire nested body chain. The let-chain collector therefore builds explicit non-overlapping
`childSpan` values:

- Binding child start: `MinAnchor(bindingLet)`, widened using the enclosing hard-left wall.
- Binding child end: `subtreeEnd(bindingLet.Value)`. The `in` token is not represented in the AST
  (verified V13: `*ast.Let` has only `Name`, `Type`, `Value`, `Body`, `Pos` — no `in`-token
  position), so it is treated as part of the binding separator emitted immediately after the value.
- Terminal child start/end: `MinAnchor(tail)` / `subtreeEnd(tail)`.
- List open boundary: inherited from the nearest structural owner when available, otherwise the
  root let's minimum anchor minus one. The list must never claim comments before the root `let`.

This creates a stable boundary before each subsequent binding and before the tail. The measured
comments occur at these between-binding positions. A comment in an unmodelled location inside a
binding value still reaches the strict-interior guard and refuses.

### Attachment Semantics

Existing rules remain unchanged:

1. A same-line trailing comment attaches to the preceding logical binding child.
2. A contiguous comment immediately before the next binding/tail is leading at that boundary.
3. A separated comment between bindings is floating at that boundary.
4. Boundary 0 and the tail boundary are supported without crossing the let-chain hard wall.
5. Consecutive comments retain source order through the existing stable sort.

The only semantic extension is that a let chain now supplies a tighter list, so comments that
previously failed at the enclosing file/block child are resolved before the strict-interior guard.

### Conditional Printer Layout

Add an attachment-index query equivalent to:

```text
hasAnyAttachment(owner) =
  any leading/floating/trailing entry whose key.owner == owner
```

`letIn` follows two paths:

- **No attachment owned by the root let:** execute the existing inline code path byte-for-byte.
- **At least one attachment owned by the root let:** flatten the maximal chain and emit each
  binding on its own line, followed by the terminal expression on its own line.

Canonical commented layout:

```text
let x = vx in
-- comment at boundary 1
let y = vy in
tail
```

The emitter interleaves `emitLeading`, `emitFloating`, and `emitTrailing` at the same indexes used
by attachment. It writes `in` before a trailing comment so the comment cannot swallow syntax.

### Continuation Indentation

The entry site that introduces the chain owns continuation indentation:

- `funcBody` equation form writes ` =`, hardlines, and emits an attached let chain one indent level
  deeper.
- `topLevelLet` writes ` =`, hardlines, and emits an attached let-valued chain one indent level
  deeper.
- A chain already inside a braced block uses that block's current indentation.
- A nested chain used as a binding value hardlines after `=` and indents one additional level.

The chain emitter itself preserves a constant indentation for sibling bindings and the tail. It
must not increment indentation per nested AST link, because those links represent a flat source
sequence semantically.

### Comment-Free Compatibility

This is a hard design constraint, not an expectation:

- `Source` constructs `printer{att:nil}`. `hasAnyAttachment` is therefore false and the old inline
  `letIn` bytes are emitted.
- `SourceWithComments` on a zero-comment file constructs an empty attachment index. The same false
  branch is taken.
- `TestAttach_ZeroCommentByteIdentity` permanently requires `SourceWithComments == Source` for
  comment-free input and is green today (V10).
- Existing Phase-1 structural round-trip cases, including `explicit_let_in`, remain on the exact
  old path and are green today (V9).

### Idempotence Argument

Idempotence depends on the output recreating the same attachment decision:

1. First formatting attaches the comment to `(rootLet, boundary k)` and selects multi-line mode.
2. The emitted program reparses to the same nested `*ast.Let` chain; only positions change.
3. The emitted comment remains immediately at canonical boundary `k` between the same logical
   children.
4. The second attachment pass chooses the same maximal root chain and boundary.
5. The second printer pass selects the same multi-line mode and emits byte-identical text.

The design never uses “was the original source multi-line?” as a printer condition. Source layout
would be unstable after pass one; attachment ownership is structural and repeatable.

### Option Evaluation

| Criterion | Option (a): conditional multi-line let chain | Option (b): edit-grade parser spans |
|---|---|---|
| Measured impact | Directly addresses 28/59 refusals; projected total 31/386 | Could support all expression boundaries eventually, but spans alone do not define canonical comment emission policy |
| Scope | `internal/format` plus tests; `cmd/ailang` behavior consumed unchanged | Lexer token offsets, AST position/span contracts, parser construction sites, diagnostics, AST equality fixtures, `astedit`, LSP, formatter envelope |
| Estimate | ~1.5–2 days | Multi-sprint; requires its own design and migration |
| Comment-free compatibility | Exact old branch when no attachment | Requires proving every new span is correct without changing parser-visible semantics |
| Idempotence | Canonical boundary-driven multi-line spelling | Still requires printer policies for each interior construct |
| Reuse | Formatter-specific but small | High future reuse for editing/LSP |
| Risk | Localized layout/attachment indexing | Broad positional-contract and downstream-consumer risk |
| Decision | **Recommend now** | **Defer** |

Option (b) is strategically valuable, but it is not a substitute for an emission policy: knowing
the exact byte range of a let expression does not decide where the canonical printer should place
a comment after collapsing the source layout. Option (a) supplies that canonical policy now and
does not block replacing envelope-derived ranges with parser byte spans later.

## Examples

### Example 1: Current let-chain refusal

The following parse-valid program was checked with the built binary (V6):

```ailang
module demo
export func main() -> int =
  let x = 1 in
  -- KEEP_INTERIOR
  let y = 2 in
  x + y
```

Current `ailang fmt` returns exit 2, identifies the comment at byte 57, and emits zero stdout bytes
(V6). Removing the comment produces the current one-line canonical form (V6):

```ailang
module demo

export func main() -> int = let x = 1 in let y = 2 in x + y
```

### Example 2: Required canonical output after this design

The proposed output is parse/type-check valid today (V7):

```ailang
module demo

export func main() -> int =
  let x = 1 in
  -- KEEP_INTERIOR
  let y = 2 in
  x + y
```

After implementation, the first format must produce exactly this stable multi-line shape, and the
second format must be byte-identical.

### Example 3: Non-let equation body remains fail-closed

This parse-valid single-expression equation body is the measured secondary class (V6):

```ailang
module demo
pure func inc(x: int) -> int =
  -- KEEP_EQUATION
  x + 1
```

Current formatting refuses with exit 2 and zero stdout bytes. This sprint does not add a generic
equation-body list, so the case remains an explicit deferred boundary rather than being floated to
the function declaration.

### Representative Corpus Cases

| Shape | Representative files | Current first refused comment |
|---|---|---|
| Equation-form let chain | `examples/integer_literals.ail`, `examples/runnable/array_grid.ail`, `examples/runnable/lambdas_advanced.ail` | Between sequential lets in a one-expression equation body |
| Bare/block-form function let chain | `examples/reference/neural_semantic_search.ail`, `examples/runnable/string_split.ail` | Between bindings in a function body parsed as root `*ast.Let` |
| Top-level let value/body chain | `examples/deriving_eq.ail`, `examples/runnable/records.ail`, `examples/snippets/type_classes_working_reference.ail` | Inside a top-level `*ast.Let` value or body chain |
| Non-let equation body | `examples/docs/records_person.ail`, `examples/runnable/jwt_decode.ail`, `examples/runnable/structured_ai_schema.ail` | Inside a one-expression equation body without a root let chain |
| Inline test list (deferred) | `examples/inline_tests_arithmetic.ail`, `examples/tests/inline_tests_recursive.ail` | Within inline `tests [...]` cases |
| Footer (deferred) | `examples/runnable/cli_args_demo.ail`, `examples/tests/sugar_call0.ail` | After the final anchored top-level child |

### Reproduced 59-File Refusal Enumeration

The V4 CLI enumeration and V5 structural classification produced the following complete set.
Categories are mutually exclusive; the four let-chain subcategories total 28.

**Let chains — equation-form `Block1(*ast.Let)` (15):**

- `examples/integer_literals.ail`
- `examples/runnable/ai_image_generation.ail`
- `examples/runnable/array_adt.ail`
- `examples/runnable/array_grid.ail`
- `examples/runnable/func_expressions.ail`
- `examples/runnable/lambdas_advanced.ail`
- `examples/runnable/lambdas_closures.ail`
- `examples/runnable/lambdas_curried.ail`
- `examples/runnable/lambdas_higher_order.ail`
- `examples/runnable/std_deflate_pdf_objstm.ail`
- `examples/runnable/string_repeat.ail`
- `examples/runnable/xml_zip_roundtrip.ail`
- `examples/snippets/showcase/lambdas_basic.ail`
- `examples/snippets/showcase/lambdas_records.ail`
- `examples/tests/test_m_r7_comprehensive.ail`

**Let chains — bare function-body `*ast.Let` (6):**

- `examples/reference/neural_semantic_search.ail`
- `examples/reference/ollama_embed_test.ail`
- `examples/reference/semantic_retrieval.ail`
- `examples/runnable/string_interp_nested.ail`
- `examples/runnable/string_interpolation.ail`
- `examples/runnable/string_split.ail`

**Let chains — let body continues as another let (5):**

- `examples/reference/sharedmem_cache.ail`
- `examples/runnable/polymorphic_comparison_simple.ail`
- `examples/runnable/polymorphic_lambdas_phase1.ail`
- `examples/runnable/records.ail`
- `examples/runnable/tar_gzip_reader.ail`

**Let chains — top-level let value rooted at another let (2):**

- `examples/deriving_eq.ail`
- `examples/snippets/type_classes_working_reference.ail`

**Non-let single-expression equation bodies (3):**

- `examples/docs/records_person.ail`
- `examples/runnable/jwt_decode.ail`
- `examples/short_circuit_and.ail`

**Inline `tests [...]` lists (9):**

- `examples/inline_tests_arithmetic.ail`
- `examples/inline_tests_best_practices.ail`
- `examples/inline_tests_recursive.ail`
- `examples/inline_tests_types.ail`
- `examples/snippets/v3_3/math/gcd.ail`
- `examples/tests/inline_tests_edge_cases.ail`
- `examples/tests/inline_tests_multiarg.ail`
- `examples/tests/inline_tests_recursive.ail`
- `examples/tests/test_conditionals.ail`

**No enclosing registered list / footer (7):**

- `examples/bugs/parser_infinite_loop_on_test_syntax.ail`
- `examples/inline_tests_nullary.ail`
- `examples/runnable/cli_args_demo.ail`
- `examples/runnable/guards_basic.ail`
- `examples/runnable/patterns.ail`
- `examples/runnable/record_patterns.ail`
- `examples/tests/sugar_call0.ail`

**Other function-expression interiors (8):**

- `examples/option_pattern_import_free.ail`
- `examples/runnable/ai_tool_loop.ail`
- `examples/runnable/contracts/inbox_injection_v2.ail`
- `examples/runnable/contracts/inbox_v2_app.ail`
- `examples/runnable/module_let_helpers.ail`
- `examples/runnable/structured_ai_schema.ail`
- `examples/tests/bug_modulo_operator.ail`
- `examples/tests/test_effect_fs.ail`

**Other AST child interiors (4):**

- `examples/effect_budget_demo.ail` — `*ast.TypeDecl`
- `examples/runnable/ai_call.ail` — match-arm `*ast.If`
- `examples/runnable/json_array_extraction.ail` — match-arm `*ast.Block`
- `examples/runnable/typeclasses.ail` — top-level `*ast.BinaryOp`

## Milestones

### M1: Let-Chain Inventory and Attachment Model — 0.5 day

1. Add a maximal-chain flattening helper shared by attachment and emission.
2. Register non-overlapping binding/tail child spans in `attach.go`.
3. Add focused attachment tests for leading, floating, trailing, consecutive, and unmodelled
   binding-value comments.
4. Prove the original 28-path let-chain set no longer returns `comment-unattached`.

### M2: Conditional Multi-Line Emission — 0.75–1 day

1. Add owner-level attachment detection in `format.go`.
2. Add the multi-line let-chain path in `expr.go`.
3. Add continuation-aware entry handling in `decl.go` and block/value emission sites.
4. Preserve the exact existing inline branch when the root owns no attachments.
5. Add the required survival + second-pass byte-identity acceptance test.

### M3: Corpus Gates and Documentation — 0.25–0.5 day

1. Split the corpus refusal counter into measured construct classes rather than labeling every
   `comment-unattached` error as let-in.
2. Require all 28 baseline let-chain paths to format losslessly.
3. Require total unattached refusals to fall from 59 to at most 31 on the unchanged corpus.
4. Re-run marker preservation, idempotence, structural round-trip, full formatter tests, and CLI
   fail-closed probes for still-deferred shapes.

## Conflict Surface

This feature is presentation-only but touches the load-bearing boundary between attachment and
printer layout. The semantic invariant remains `Parse(fmt(x)) ≡ Parse(x)`.

### Syntactic positions touched

No grammar production, token kind, parser entry point, AST node, or AST `Span` field changes under
the recommended option. Comments remain absent from the parser-visible token stream.

Option (b) would change positional contracts throughout `internal/lexer`, `internal/parser`, and
`internal/ast`; that is the primary reason it is deferred.

### Areas touched

| Area | Relationship and constraint |
|---|---|
| `internal/format/expr.go` | `letIn` is the principal emission site. It gains a conditional multi-line path while retaining the current inline path exactly when the root owns no attachments. Precedence wrapping remains unchanged. |
| `internal/format/decl.go` | `funcBody` and `topLevelLet` introduce continuation indentation for an attached root/value let chain. Equation `Block` vs bare body identity rules must remain unchanged. |
| `internal/format/format.go` | Adds owner-level attachment detection and may centralize continuation emission. Existing leading/floating/trailing helpers and comment-free `att:nil` behavior remain authoritative. |
| `internal/format/attach.go` | Adds maximal let-chain discovery and non-overlapping child spans. The strict-interior refusal guard stays intact for unsupported interiors. |
| `internal/format/envelope.go` | Consumed unchanged for byte anchors, comments, line tables, and literal clamping. No new source-position premise is introduced. |
| `internal/format/anchors.go` | Consumed unchanged to find minimum anchors for binding and tail nodes. Any helper extension must remain formatter-private and token-anchored. |
| `internal/format/corpus_comment_test.go` | Refusal reporting is split into actual construct classes. The corpus remains the measured acceptance gate. |
| `internal/format/marker_property_test.go` | Existing corpus idempotence gate must include newly accepted files automatically and remain fully green. |
| `internal/format/roundtrip_test.go` | Existing comment-free structural round-trip and idempotence cases remain green; add a comment-aware structural comparison in the focused acceptance test. |
| `internal/format/attach_test.go` / new focused test | Adds exact survival, marker-count, canonical layout, structural round-trip, and second-pass byte-identity assertions. |
| `cmd/ailang/fmt.go` | No implementation change required. It consumes fewer `comment-unattached` errors; exit codes, all-or-nothing `--write`, reparsing, AST comparison, and atomic write behavior remain unchanged. Help text still truthfully states that unsupported inline interiors refuse. |
| `internal/parser`, `internal/ast` | Read-only under option (a). Their nested-let representation is consumed as-is. |
| `internal/astedit`, `internal/lsp` | No change under option (a). They are beneficiaries, not dependencies, of deferred option (b). |
| types / effects / evaluator / codegen / runtime | No dependency and no modification. |

### Programs that MUST still work

1. Every comment-free formatter fixture and generated round-trip case, including explicit let-in
   expressions, with byte-identical Phase-1 output.
2. All 327 files currently accepted by the Phase-2 corpus gate, with zero new refusals, zero marker
   failures, and zero Phase-2 round-trip regressions.
3. All 299 files currently exercised by `TestMarkerProperty_CorpusIdempotence`, still byte-identical
   on the second pass; newly accepted eligible files join the denominator.
4. The 28 identified let-chain files, now accepted with every comment present exactly once.
5. The 31 deferred refusal files, still fail-closed and byte-identical rather than being relocated.
6. `ailang fmt`, `--check`, and `--write` exit-code and all-or-nothing contracts.

### Proof that comment-free round-trip cannot regress

The layout branch is gated solely by attachments owned by the root let. `Source` has `att:nil`, and
zero-comment `SourceWithComments` has no attachment entries, so both execute the unchanged inline
branch. The parser sees the exact existing bytes, not merely equivalent new whitespace. Existing
`TestAttach_ZeroCommentByteIdentity` enforces this identity directly; generated structural
round-trip tests enforce the AST property (V9–V10).

### Proof that idempotence cannot regress by construction

The multi-line branch is selected by a structural attachment to a maximal nested-let chain, not by
source line count. Its output places the comment at the same indexed logical boundary that the next
parse reconstructs. The new focused test and the corpus idempotence gate verify the argument. If a
comment does not resolve to that structural list, the old refusal guard remains and no output is
produced.

### Option (b) conflict surface if pursued later

A parser-span retrofit must, at minimum:

1. Add byte offsets to lexer tokens and `ast.Pos` or introduce a separate edit-range type.
2. Define inclusive/exclusive end semantics for every expression, pattern, type, declaration,
   modifier, annotation, delimiter, and comment-adjacent boundary.
3. Populate full ranges at every parser construction site, including desugared nodes and
   interpolation-queued tokens.
4. Decide whether synthesized AST nodes have absent, inherited, or multi-origin ranges.
5. Update AST equality/golden tests, diagnostics, `astedit`, LSP indexing/ranges, formatter
   envelope conversion, and any serialized error/report contracts.
6. Prove byte accuracy across Unicode, CRLF normalization, strings/interpolations, tagged
   quasiquotes, modifiers, annotations, and malformed input.

This scope is useful but materially larger than the measured formatter-local intervention.

## Testing Strategy

### New acceptance-gated test

Add `TestInlineInterior_LetChainPreservedAndIdempotent` using the Example 2 source. It must:

1. Parse the original source without errors.
2. Call `SourceWithComments` and require success.
3. Require `KEEP_INTERIOR` exactly once in output.
4. Require the exact canonical multi-line let-chain golden.
5. Reparse output and require structural AST equality with the original, ignoring only
   `ast.Pos`/`ast.Span`.
6. Call `SourceWithComments` on the reparsed output.
7. Require pass-two bytes to equal pass-one bytes exactly.

The test fails if the implementation merely stops refusing but moves the comment to a declaration
or block boundary.

### Focused unit matrix

| Case | Expected result |
|---|---|
| Comment before second binding | Leading attachment at boundary 1; stable multi-line output |
| Blank-line-separated comment between bindings | Floating attachment at boundary 1; stable blank-line policy |
| Same-line comment after `in` | Trailing attachment emitted after `in`, never swallowing syntax |
| Consecutive comments between bindings | Source order retained, each marker exactly once |
| Comment before terminal tail | Leading attachment at tail boundary |
| Nested independent let chain inside a binding value | Separate maximal-chain owner; deterministic indentation |
| Comment-free let chain | Byte-identical to current Phase-1 inline output |
| Comment inside an unmodelled binding-value interior | Still returns `comment-unattached`; no partial output |
| `letrec` chain | Existing behavior unchanged unless separately measured and designed |

### Existing hard gates

1. `TestAttach_ZeroCommentByteIdentity` — comment-free `SourceWithComments` equals `Source`.
2. `TestIdempotenceAndRoundTrip_GeneratedCases` — all Phase-1 generated cases remain green.
3. `TestMarkerProperty_CorpusIdempotence` — currently 299/299; no accepted file may fail pass-two
   byte identity, and newly accepted files are included automatically.
4. `TestCorpusCommentGate` — zero marker loss, zero Phase-2 round-trip regressions, zero unexpected
   refusal class; let-chain baseline set accepted; total unattached refusal count ≤31.
5. Full `go test ./internal/format/`.
6. CLI `--write` probe on a still-deferred refusal — exit 2 and unchanged SHA-256.

### Corpus classification gate

The current variable name `letInRefusal` counts every `EnvelopeError{Kind:"comment-unattached"}` and
therefore obscures the live distribution. M3 must classify the failed comment's tightest owner and
child shape, producing at least these stable counters:

- let-chain interior;
- non-let equation body;
- inline tests list;
- no enclosing registered list;
- other expression/declaration interior.

Classification is reporting and acceptance evidence; it must not alter attachment behavior.

## Non-Goals

1. **No interpolation-interior attachment.** It remains separately deferred and is measured at
   0/386 parse-valid files. The interpolation fail-closed carve-out stays unchanged.
2. **No full parser-span retrofit.** Option (b) is not selected for this sprint.
3. No generic preservation of arbitrary original line breaks.
4. No weakening or removal of fail-closed refusal for unsupported interiors.
5. No changes to grammar, AST shape, parser-visible tokens, precedence, evaluation, types, effects,
   code generation, or runtime.
6. No attempt to fix the 28 pre-existing Phase-1 `properties[...]` structural round-trip bugs.
7. No inline `tests [...]`, record-field, type-field, match-expression, or footer comment solution.
8. No directory recursion, stdin mode, editor integration, or adoption-hook changes.

## Risks & Mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| Overlapping suffix-chain lists make attachment traversal-order-dependent | Non-deterministic or unstable boundary selection | Register maximal chains only; unit-test nested independent chains |
| Binding spans overlap because `subtreeEnd(*ast.Let)` includes its body | Comment still refuses or attaches to wrong binding | Build explicit binding spans ending at `subtreeEnd(Value)`; append tail separately |
| `--` comment swallows `in` or a separator | Formatted output fails to parse | Emit `in` before trailing comments; structural round-trip gate |
| Continuation lines return to column 1 | Poor canonical layout and possible contextual parse ambiguity | Entry site owns one continuation indent; chain siblings do not accumulate nesting indent |
| Comment-free output changes | Phase-1 compatibility regression | Attachment-triggered branch; `att:nil`/empty fast path; byte-identity test |
| First pass multi-lines, second pass collapses | Direct idempotence violation | Select layout from reconstructed attachment ownership, not original line count; exact two-pass test |
| Change accidentally accepts unsupported interiors by floating outward | Silent semantic comment relocation | Keep strict-interior guard unchanged; negative test for binding-value interior |
| Corpus count remains 59 because let list loses to outer same-line logic | No measured impact | Baseline 28-path acceptance list and construct-class counters are release gates |
| Option (a) becomes dead-end work before parser spans | Duplicated effort | Keep list/attachment API unchanged; future spans replace range derivation, not emission policy |
| Coarse existing gate mislabels residual failures | Misleading success claims | Split refusal taxonomy before declaring completion |

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|---|---:|---|
| A1: Determinism | +1 | Maximal-chain discovery, indexed boundaries, and attachment-triggered canonical layout are pure and rule-driven |
| A2: Replayability | 0 | Formatting does not enter execution traces |
| A3: Effect Legibility | 0 | No effect changes |
| A4: Explicit Authority | 0 | No capability or ambient authority changes |
| A5: Bounded Verification | +1 | Scope and acceptance are bounded by the reproduced 28-path let-chain set and corpus counters |
| A6: Safe Concurrency | 0 | No concurrency impact |
| A7: Machines First | +1 | Cuts the measured formatter refusal set by a projected 47.46% while retaining machine-checkable losslessness and idempotence |
| A8: Minimal Syntax | 0 | No syntax or grammar change |
| A9: Cost Visibility | 0 | No runtime resource model changes |
| A10: Composability | +1 | Extends the existing envelope/attachment/printer APIs and remains compatible with future parser byte spans |
| A11: Structured Failure | +1 | Narrows refusal only where a stable structural boundary is added; unsupported cases remain explicit fail-closed errors |
| A12: System Boundary | 0 | No external boundary changes |

**Net Score: +5** → **Decision: Proceed with option (a)**

### Hard Violation Check

- [x] A1 (Determinism): no source-layout heuristic or traversal-order-dependent suffix list
- [x] A3 (Effects): no hidden effects
- [x] A4 (Authority): no ambient access
- [x] A7 (Machines First): exact marker, structural, idempotence, and corpus gates remain authoritative

## Verification Log

All commands were run in the worktree on 2026-07-20. Temporary probe files and Go overlay files
were written under `/tmp`; no repository file was modified during verification.

| ID | Exact command | Observed result | What it proves |
|---|---|---|---|
| V1 | `make build` | Built `bin/ailang` with version ldflags; command exited 0. | Required live binary was rebuilt from this worktree before formatter probes. |
| V2 | `./bin/ailang --version` | `AILANG v0.30.0-27-gc9ae4ce55`; commit `c9ae4ce55b810965e7aceaacddf3acd3dd2e214c`; built `2026-07-20_03:47:34`. | Records the exact formatter binary used for all CLI observations. |
| V3 | `go test ./internal/format/ -run TestCorpusCommentGate -v` | PASS. `parse-valid=386 formatted=327`, `interp-refusal=0`, `let-in-refusal=59`, `other-refusal=0`, `PHASE2-rt-regression=0`, `preexisting-Phase1-rt-bug=28`, `marker-fail=0`; rates 0.00% interpolation and 15.28% unattached/let-in. | Reproduces the shipped corpus measurement and confirms current safety: no marker loss and no Phase-2 structural regression. |
| V4 | `find examples -name '*.ail' -type f \| sort \| while IFS= read -r f; do out=$(./bin/ailang fmt "$f" 2>&1 >/dev/null); rc=$?; if [ "$rc" -eq 2 ] && printf '%s' "$out" \| rg -q 'could not be attached\|stable boundary\|inline-emitted\|attach'; then printf '%s\t%s\n' "$f" "$out"; fi; done` | Produced exactly 59 path/error rows. The set starts with `examples/bugs/parser_infinite_loop_on_test_syntax.ail`, `examples/deriving_eq.ail`, `examples/docs/records_person.ail` and ends with `examples/tests/test_effect_fs.ail`, `examples/tests/test_m_r7_comprehensive.ail`. | Independently enumerates the 59 CLI refusals rather than relying only on the aggregate test counter. |
| V5 | `go test -overlay=/tmp/inline_overlay.json ./internal/format -run TestInlineClassifyOverlay -v` | PASS diagnostic overlay; `COUNTS=map[equation-block1-let-chain:15 equation-single-expression:3 function-let-chain:6 inline-tests-list:9 no-enclosing-list:7 other-*ast.BinaryOp:1 other-*ast.Block:1 other-*ast.If:1 other-*ast.TypeDecl:1 other-function-interior:8 top-level-let-interior:5 top-level-let-value-chain:2]`. Overlay mapped a temporary test into `internal/format` without changing the worktree. | Gives the fine-grained 59-file distribution. Let-chain classes total 28; non-let single-expression equation bodies total 3; all remaining classes total 28. |
| V6 | `./bin/ailang check /tmp/fmt-let-interior.ail; ./bin/ailang fmt /tmp/fmt-let-interior.ail; ./bin/ailang fmt /tmp/fmt-let-commentfree.ail; ./bin/ailang check /tmp/fmt-equation-interior.ail; ./bin/ailang fmt /tmp/fmt-equation-interior.ail` | Both commented probes checked with “No errors found”. Let-chain fmt: exit 2, `comment at byte 57 ("-- KEEP_INTERIOR") could not be attached`, zero stdout. Comment-free fmt: exit 0 and `export func main() -> int = let x = 1 in let y = 2 in x + y`. Equation fmt: exit 2, byte 45 `KEEP_EQUATION`, zero stdout. | Verifies current parser validity, inline collapse, let-chain refusal, and the minor non-let equation-body refusal on concrete programs used in Examples. |
| V7 | `./bin/ailang check /tmp/fmt-let-proposed.ail` | Exit 0, “No errors found” (only the expected temporary-path module warning). | Confirms the proposed canonical multi-line output is valid AILANG today. |
| V8 | `cp /tmp/fmt-let-interior.ail /tmp/fmt-let-write.ail; before=$(shasum -a 256 /tmp/fmt-let-write.ail \| cut -d' ' -f1); ./bin/ailang fmt --write /tmp/fmt-let-write.ail; rc=$?; after=$(shasum -a 256 /tmp/fmt-let-write.ail \| cut -d' ' -f1)` | Exit 2. Before and after SHA-256 both `cfbb65242a0084ef0877a74eb060b661a58c5b4c12b237036ee9b154b4d57adf`; `identical=yes`. | Confirms current behavior is safe fail-closed and `--write` does not modify a refused file. |
| V9 | `go test ./internal/format/ -run 'TestMarkerProperty_CorpusIdempotence\|TestIdempotenceAndRoundTrip_GeneratedCases' -v` | PASS. Corpus marker/idempotence `299/299`; all 22 generated structural round-trip/idempotence cases passed, including `explicit_let_in`, `eq_body_sequence`, and `top_level_let`. | Establishes the load-bearing current baseline that the design must preserve and extend. |
| V10 | `go test ./internal/format/ -run TestAttach_ZeroCommentByteIdentity -v` | PASS. | Confirms the existing permanent gate that comment-free `SourceWithComments` is byte-identical to Phase-1 `Source`. |
| V11 | `rg -n 'func \(p \*printer\) letIn\|func \(p \*printer\) funcBody\|func \(p \*printer\) topLevelLet\|comment-unattached' internal/format cmd/ailang/fmt.go` | Located inline `letIn` in `expr.go`, equation/bare-body selection and top-level let emission in `decl.go`, and the unattached error raised from `attach.go`; `cmd/ailang/fmt.go` propagates formatter errors before any write. | Verifies the recommended option's concrete printer/attachment/CLI conflict surface. |
| V12 | `rg -n '\.Span\b\|Span\{' internal/parser internal/ast internal/astedit internal/lsp` | Parser span population is partial; `internal/astedit/astedit.go` documents line/column-to-byte conversion and delimiter scanning; `internal/lsp/index.go` states many nodes expose only `Pos` starts, not reliable end spans. | Grounds the honest larger scope of option (b) and its `astedit`/LSP benefits. |
| V13 | `rg -n 'type Let struct' internal/ast && sed -n '240,246p' internal/ast/ast_expr.go` (controller-added at re-quorum, 2026-07-20) | `internal/ast/ast_expr.go:240` — `type Let struct { Name string; Type Type; Value Expr; Body Expr; Pos Pos }`. Single `Pos` start position; NO `in`-token position field. | Confirms the load-bearing premise of let-chain flattening: `*ast.Let` exposes `Value` and `Body` for maximal-chain traversal (Decision 3, Let-Chain Discovery), and the `in` keyword has no AST representation, so it must be emitted as part of the binding separator (Stable Child Spans). Closes the sole re-quorum objection (unverified `*ast.Let` structure premise). |

## References

- [M-AILANG-FMT-PHASE2](../../implemented/v0_30_0/m-ailang-fmt-phase2.md) — parent design,
  implemented envelope/attachment architecture, calibrated fail-closed boundary, conflict surface,
  axiom rubric, and Future Work entries refined here.
- `internal/format/attach.go` — ordered child-list inventory, deterministic attachment rules, and
  strict-interior fail-closed guard.
- `internal/format/expr.go` — precedence-aware expression printer and current inline `letIn` path.
- `internal/format/decl.go` — function equation/block identity rules and top-level let emission.
- `internal/format/envelope.go` — token-anchored byte/line envelope and interpolation carve-out.
- `internal/format/anchors.go` — AST token-anchor traversal used by attachment ranges.
- `internal/format/corpus_comment_test.go` — 386-file comment gate and current coarse refusal count.
- `internal/format/marker_property_test.go` — corpus idempotence property.
- `internal/format/roundtrip_test.go` — comment-free idempotence and structural AST round-trip.
- `cmd/ailang/fmt.go` — fail-closed CLI, exit codes, reparsing, structural comparison, and atomic
  write contract.
- `internal/astedit/astedit.go` and `internal/lsp/index.go` — current consumers motivating the
  deferred edit-grade parser-span path.
- [Design Axioms](/docs/references/axioms).

## Future Work

- **Non-let equation-body attachment** (3/386 measured): add a conditional equation-body boundary
  only after defining canonical indentation and proving two-pass identity.
- **Inline `tests [...]` attachment** (9/386 measured): model test cases as an ordered child list and
  choose a canonical multi-line tests printer.
- **Footer and full-declaration boundaries** (7/386 measured no-enclosing-list cases): likely best
  addressed by trustworthy declaration ends or a formatter-owned closing-token index.
- **Other expression-interior lists** (12/386 measured): split match, record, type-field, contract,
  and binary/call shapes; add only evidence-backed conditional lists.
- **Edit-grade parser spans** (deferred option (b)): byte offsets plus full declaration/expression
  boundaries including modifiers, annotations, and delimiters; migrate `astedit`, LSP, and the
  formatter envelope after a dedicated design and premise sweep.
- **Interpolation-interior comment attachment**: remains separately deferred and evidence-gated at
  0/386; do not combine it with this work.
- **Letrec chain support**: measure before extending the ordinary-let chain abstraction.
