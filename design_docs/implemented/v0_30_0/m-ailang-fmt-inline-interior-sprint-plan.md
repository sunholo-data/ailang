# Sprint Plan — M-AILANG-FMT-INLINE-INTERIOR

**Design doc**: [m-ailang-fmt-inline-interior.md](m-ailang-fmt-inline-interior.md)
**Sprint JSON**: `.ailang/state/sprints/sprint_M-AILANG-FMT-INLINE-INTERIOR.json`
**Target**: v0.30.0 (final item of the "finish off ailang fmt" polish pair)
**Risk level**: LOW (printer-local; READ-ONLY for `internal/parser` + `internal/ast`)
**Estimated**: ~1.75 days (~13h) · ~450 LOC (impl + tests)
**Status**: UNPARKED — Mark 2026-07-20 ("yes lets finish off ailang fmt — proceed on the data, no
re-quorum"). Quorum was consumed at mission iteration 67; the R2 objection is DATA-REFUTED for all
28 target cases. Routed straight to executor after this plan. Do NOT re-run quorum.
**Related**: sibling of [m-fmt-properties-printer-roundtrip](m-fmt-properties-printer-roundtrip-sprint-plan.md) (landed #424).

## Summary

`ailang fmt` Phase 2 fails-closed (exit 2, no output) when a comment lands strictly inside a
multi-line child span that no registered child list decomposes. The current corpus gate measures
**59 refusals / 386 parse-valid files (15.28%)**, zero comment loss, zero Phase-2 round-trip
regressions (re-confirmed live at HEAD `37ef22321`; see Premise Verification below).

The largest coherent refusal class is **multi-line `let … in` chains (28/59, 47.46%)**. The parser
collapses a source sequence of sequential lets into nested `*ast.Let` (via the `Body` field); the
Phase-1 printer's `letIn` emitter writes them inline (`let x = v in let y = w in tail`); so a
comment between two source bindings has no stable printer boundary → `attachOne`'s strict-interior
guard refuses (`comment-unattached`).

The fix (design Decision 1, option (a)) is a **printer-local conditional multi-line let-chain
emitter**: the attacher discovers the maximal nested-let chain and registers non-overlapping
binding/tail child spans; the printer emits the chain multi-line ONLY when a comment attaches to one
of those boundaries. Comment-free and non-attaching input keep the exact existing inline bytes.

**Acceptance headline metric**: the corpus `comment-unattached` refusal count must fall from **59 to
≤ 31** (≤ 8.03%) on the unchanged corpus, with the 28 let-chain files newly accepted, **zero comment
loss, zero Phase-2 round-trip regression, and `preexisting-Phase1-rt-bug` still 0** (the sibling
sprint hardened that counter into a fatal gate — see below).

## Premise verification vs repo reality (HEAD `37ef22321`, `v0.30.0-43-g37ef22321`)

Every load-bearing premise re-confirmed against the current tree (the doc's Verification Log ran at
`c9ae4ce55`, +16 commits ago). **Line numbers below are HEAD-accurate and supersede the doc where
it drifts.**

| Claim | Doc reference | Verified at HEAD |
|-------|---------------|------------------|
| `ast.Let` = `{ Name; Type; Value; Body; Pos }` — single `Pos`, NO `in`-token position | V13, decl at `ast_expr.go:240` | ✅ `internal/ast/ast_expr.go:240-246` |
| Inline `letIn` emitter (the principal emission site) | `expr.go` | ✅ `internal/format/expr.go:228` (`func (p *printer) letIn`) |
| `funcBody` continuation entry site | `decl.go` | ✅ `internal/format/decl.go:162` |
| `topLevelLet` continuation entry site | `decl.go` | ✅ `internal/format/decl.go:545` |
| `comment-unattached` raised by strict-interior guard | `attach.go` | ✅ `internal/format/attach.go:298` (raise) + `:358-371` (guard: `ch.startLine < cLine && cLine < ch.endLine`) |
| Smallest-enclosing-list selection precedes the guard | Decision 2 | ✅ `attach.go:335-356` picks tightest `best` list first; guard at `:367-371` only fires when none decomposes |
| Corpus counter `letInRefusal` counts every `comment-unattached` | §Corpus classification gate | ✅ `corpus_comment_test.go:28,53` |
| Current refusal baseline = 59/386 (15.28%) | V3 | ✅ **live re-run at HEAD**: `parse-valid=386 formatted=327 let-in-refusal=59 (15.28%)`, `interp-refusal=0 other-refusal=0 PHASE2-rt-regression=0 marker-fail=0` |
| All 28 target files present on disk | §Reproduced enumeration | ✅ 28/28 exist |

### Premise CORRECTIONS made vs the design doc

1. **Problem-Statement bare-`;` sentence is inaccurate (the R2 defect).** The doc's Problem
   Statement (§Problem Statement, first bullet) claims `{ let x = v; let y = w; tail }` "collapses
   into nested `*ast.Let` expressions". **Verified false at HEAD**: `parseBlockOrExpression`
   (`internal/parser/parser_expr.go:388`) accumulates `;`- *and* newline-separated statements into
   `exprs` and returns `&ast.Block{Exprs: exprs}` whenever `len(exprs) > 1` (lines 425-468). Only a
   single-expression block (`len==1`) is returned directly (line 461-463) — and the 28 targets reach
   nested `*ast.Let.Body` because they chain via `let … in`, not bare `;`. **M0 must correct this
   sentence in the design doc** (state that the bare-`;`/newline block form is `Block.Exprs`, and
   the 28 targets are the `let … in` continuation form). This correction is bookkeeping — it does
   NOT change scope (no target file uses the bare-`;` form; controller sampled 11/28, M0 verifies
   all 28).

2. **`ailang debug ast` is NOT usable for the M0 shape probe.** It runs through elaboration and
   prints the CORE AST, not the surface `internal/ast` shape. The reliable M0 tool is a **Go test in
   `internal/format`** that parses each of the 28 files with `parser.New(lexer.New(...))` and
   type-switches on the surface `*ast.Let` / `*ast.Block` nodes (the same nodes the attacher walks).
   The plan mandates this Go-test dump rather than a CLI dump.

3. **`preexisting-Phase1-rt-bug` is now a HARD gate (0), not tolerated.** The doc's Testing Strategy
   (§Existing hard gates) predates the sibling sprint. As of `37ef22321`, `corpus_comment_test.go:125`
   makes `preExistingRT != 0` a `t.Fatalf`. This sprint MUST keep it at 0 — a printer round-trip
   regression introduced by the new multi-line path would regrow that class and fail the gate.

**No other premise corrections.** Every cited behavior, node shape, and refusal count holds at HEAD.

## Milestones

Three milestones, matching the design doc's M1/M2/M3, with a mandatory **M0 verification gate
prepended** per the human-fork RECOMMENDED option 1. Total ~1.75d.

---

### M0 — Surface-AST shape verification + doc correction (~0.25 day, ~120 LOC) — MANDATORY, BEFORE ANY PRINTER CHANGE

**Purpose**: prove — not assume — that all 28 target files chain via nested `*ast.Let.Body` (not
`*ast.Block.Exprs`), so the design's flattening traversal actually applies. This is the gate the
human fork requires; the sampled 11/28 all used `let … in`, but M0 must verify **all 28**.

**Files**: `internal/format/inline_interior_shape_test.go` (new, ~100 LOC),
`design_docs/planned/v0_30_0/m-ailang-fmt-inline-interior.md` (~1 sentence correction).

1. Add `TestInlineInterior_LetChainSurfaceShape` (skip/read-only, no printer change): embed the
   list of 28 target paths as a table. For each, `parser.New(lexer.New(data, path)).Parse()`, walk
   the surface `*ast.File`, and locate the refused comment's enclosing expression.
2. Assert each of the 28 reaches a **root `*ast.Let` with non-nil `Body`** whose body is (or
   flattens through) another `*ast.Let` — i.e. a nested `Let.Body` chain — at the position the CLI
   currently refuses. Classify any file whose refusal site is a `*ast.Block{Exprs}` (bare-`;`/newline
   form) or a non-let shape into an explicit `EXCLUDED` bucket with the reason logged.
3. **Decision rule** (records the M0 outcome for M1 scoping):
   - If all 28 are nested-`Let.Body` → proceed with the 28-file target unchanged.
   - If any file is `Block.Exprs` or non-let → **exclude it from the M1 target set and downgrade the
     acceptance target accordingly** (e.g. 26/28 nested → refusal target becomes ≤ 33 not ≤ 31).
     Adding `Block.Exprs` handling is explicitly OUT of scope for this sprint (design Deferred
     Decision; human fork: "add `Block.Exprs` handling only if any file actually needs it").
4. Correct the Problem-Statement bare-`;` sentence in the design doc (premise correction #1 above):
   state that the bare-`;`/newline block form parses to `*ast.Block.Exprs`, and the 28 targets are
   the `let … in` continuation form that collapses to nested `*ast.Let.Body`.

**Acceptance criteria** (M0 = the shape-proof gate):
- `go test ./internal/format/ -run TestInlineInterior_LetChainSurfaceShape -v` passes and LOGS the
  per-file classification for all 28.
- The count of files classified nested-`Let.Body` is recorded; any `EXCLUDED` file is named with a
  reason, and the M1 target set + refusal-count target are adjusted to match (documented in the
  sprint JSON `notes`).
- The design doc's Problem-Statement bare-`;` sentence is corrected.
- **Metric**: this milestone changes NO refusal count (read-only) — its output is the *verified
  target set N* (expected 28) that fixes M1/M3's acceptance denominator.

---

### M1 — Let-chain discovery + attachment model (~0.5 day, ~150 LOC)

**Files**: `internal/format/attach.go` (~90 LOC), `internal/format/anchors.go` (consumed, maybe
tiny helper), `internal/format/attach_test.go` or new `inline_interior_test.go` (~60 LOC).

1. Add a maximal-chain flattening helper (shared by attachment and later emission): given a root
   `*ast.Let`, walk `Body` links while they are `*ast.Let`, producing the logical child sequence
   `[binding(x,vx), binding(y,vy), …, tail]`. Register **maximal chains only** (skip a `*ast.Let`
   that is being consumed as the `Body` link of an enclosing chain) — no overlapping suffix lists.
2. In `attach.go`, register one `childList` owned by the root `*ast.Let` with **explicit
   non-overlapping child spans** (the generic `addList` cannot be used because
   `subtreeEnd(bindingLet)` includes the whole nested body):
   - binding child end = `subtreeEnd(bindingLet.Value)` (the `in` has no AST position — V13 — so it
     is part of the separator emitted after the value);
   - tail child = `MinAnchor(tail)` / `subtreeEnd(tail)`;
   - list open boundary never claims comments before the root `let`.
3. Recurse into each binding value and the tail to discover nested blocks/matches/independent
   chains.
4. Add focused attachment tests: comment before 2nd binding (leading), blank-separated
   (floating), same-line after `in` (trailing), consecutive comments (source order), comment before
   tail, nested independent chain in a binding value, and a comment inside an UNMODELLED
   binding-value interior (must STILL return `comment-unattached` — the guard is unchanged).

**Acceptance criteria**:
- The M0-verified N (≈28) let-chain paths no longer return `comment-unattached` at the attachment
  layer (assert via a test that runs the attacher over each and requires attachment success).
- The unmodelled-binding-interior negative test still refuses (strict-interior guard intact —
  Decision 2).
- `go test ./internal/format/ -run 'TestInlineInterior|TestAttach' -v` green.
- **Metric**: attachment success for N/N target files (was 0/N); refusals not yet reduced at the
  CLI/corpus layer until M2 emits the multi-line layout.

---

### M2 — Conditional multi-line emission + acceptance-gated round-trip (~0.75 day, ~120 LOC)

**Files**: `internal/format/format.go` (~15 LOC, `hasAnyAttachment(owner)`),
`internal/format/expr.go` (~40 LOC, `letIn` conditional path),
`internal/format/decl.go` (~25 LOC, `funcBody`/`topLevelLet` continuation indent),
`internal/format/inline_interior_test.go` (~40 LOC acceptance test).

1. Add `hasAnyAttachment(owner)` in `format.go`: true iff any leading/floating/trailing entry has
   `key.owner == owner`. With `att:nil` (`Source`) or an empty index (zero-comment
   `SourceWithComments`) it is false.
2. `letIn` (`expr.go:228`) gains two paths:
   - **no attachment owned by root let** → the EXISTING inline code path, byte-for-byte;
   - **≥1 attachment owned by root let** → flatten the maximal chain and emit each binding on its
     own line then the tail, interleaving `emitLeading`/`emitFloating`/`emitTrailing` at the
     attachment indexes. Write `in` BEFORE a trailing comment so it cannot swallow syntax.
3. `funcBody` (`decl.go:162`) and `topLevelLet` (`decl.go:545`): when the entered chain is attached,
   write ` =`, hardline, and emit the chain one indent level deeper; chain siblings hold a CONSTANT
   indent (do not increment per nested AST link).
4. Add `TestInlineInterior_LetChainPreservedAndIdempotent` (design §Testing Strategy) over the
   Example-2 source: parse OK → `SourceWithComments` OK → `KEEP_INTERIOR` present exactly once →
   exact canonical multi-line golden → reparse structurally equal to original (ignore Pos/Span) →
   `SourceWithComments` on the output → pass-two bytes == pass-one bytes. The test must FAIL if the
   comment is floated to a declaration/block boundary instead of the chain boundary.

**Acceptance criteria**:
- `TestInlineInterior_LetChainPreservedAndIdempotent` passes all 7 assertions (idempotence +
  never-lossy proven for the canonical case).
- `TestAttach_ZeroCommentByteIdentity` still green (comment-free `SourceWithComments` == `Source`).
- `TestIdempotenceAndRoundTrip_GeneratedCases` still green (incl. `explicit_let_in`).
- `go test ./internal/format/` fully green.
- **Metric**: a targeted probe (format each of the N target files, confirm exit 0 + comment count
  preserved) shows N/N now format losslessly — the CLI refusals for the target set drop to 0.

---

### M3 — Corpus gate split + refusal-count reduction + docs (~0.25 day, ~60 LOC)

**Files**: `internal/format/corpus_comment_test.go` (~40 LOC classifier + assertion),
`internal/format/marker_property_test.go` (verify auto-inclusion, likely 0 LOC),
`changelogs/v0.18-current.md` (CHANGELOG), design doc status flip to implemented.

1. Split the `letInRefusal` counter into measured construct classes (let-chain interior, non-let
   equation body, inline tests list, no-enclosing-list, other) so residual refusals are labeled
   truthfully (§Corpus classification gate). Classification is reporting only — it must not alter
   attachment behavior.
2. Assert the corpus gate: **total `comment-unattached` refusals ≤ 31** (or ≤ M0-adjusted target),
   **let-chain-interior class == 0**, `preexisting-Phase1-rt-bug == 0` (unchanged hard gate),
   `PHASE2-rt-regression == 0`, `marker-fail == 0`.
3. Confirm `TestMarkerProperty_CorpusIdempotence` auto-includes the newly accepted files and stays
   byte-identical on pass two.
4. CLI fail-closed probe: a still-deferred refusal file (e.g. `examples/docs/records_person.ail`,
   a non-let equation body) still exits 2 with unchanged SHA-256 under `fmt --write`.
5. CHANGELOG entry; flip design-doc status to implemented (move to `implemented/v0_30_0/` per the
   post-sprint convention, or leave the executor to do so).

**Acceptance criteria** (M3 = the headline metric gate):
- `go test ./internal/format/ -run TestCorpusCommentGate -v`: `let-in-refusal` (total
  `comment-unattached`) drops from **59 to ≤ 31** (≤ 8.03%); let-chain-interior sub-class == 0;
  `preexisting-Phase1-rt-bug == 0`; `PHASE2-rt-regression == 0`; `marker-fail == 0`.
- `TestMarkerProperty_CorpusIdempotence` green with the newly accepted files in the denominator.
- `TestIdempotenceAndRoundTrip_GeneratedCases` green.
- A deferred non-let refusal file still exits 2 with identical SHA-256 under `fmt --write`.
- `make test` green; `make verify-examples` green (manifest-drift is the known-benign red mode, not
  a type regression).
- **Metric**: refusal-count delta **59 → ≤ 31** is the release gate; a residual of exactly the
  non-let/deferred classes (≈31) is the expected end state.

## Invariants preserved (design Goals 3–6, non-negotiable)

1. **Idempotence** `fmt(fmt(x)) == fmt(x)`: layout is selected from reconstructed ATTACHMENT
   ownership (structural, repeatable), never from "was the source multi-line?". Proven by the M2
   two-pass byte-identity test + the corpus idempotence gate.
2. **Never-lossy**: no comment deleted/duplicated. Proven by marker-count preservation (corpus gate)
   + `KEEP_INTERIOR`-exactly-once (M2 test).
3. **Comment-free byte identity**: `att:nil`/empty-index take the exact old inline branch. Proven by
   `TestAttach_ZeroCommentByteIdentity`.
4. **Strict-interior guard UNCHANGED** (Decision 2): unsupported interiors still fail-closed. Proven
   by the M1 negative test + the M3 deferred-file CLI probe.

## Non-Goals (respect the doc — do NOT expand scope)

- No non-let equation-body attachment (3 files stay fail-closed).
- No inline `tests [...]`, footer/no-enclosing-list, or other-interior attachment (deferred classes,
  ~31 files stay fail-closed and byte-identical).
- No `letrec` chain support (measure first).
- No `Block.Exprs` (bare-`;`) handling unless M0 proves a target file needs it (none sampled does).
- No edit-grade parser spans (option (b), deferred multi-sprint).
- **READ-ONLY for `internal/parser` and `internal/ast`** — no grammar, token, AST node, or `Span`
  change. `cmd/ailang/fmt.go` needs no implementation change (it just sees fewer refusals).

## Total estimate

- **Milestones**: 4 (M0 gate + M1/M2/M3 matching the doc).
- **Effort**: ~13h → ~1.75 days (M0 2h · M1 4h · M2 5h · M3 2h). Within the doc's ~1.5–2d.
- **LOC**: ~450 (M0 ~120 · M1 ~150 · M2 ~120 · M3 ~60).
- **Risk**: LOW — printer-local, parser/ast read-only, fail-closed verifier + idempotence gate are
  the safety net.

---

**SPRINT_PLAN_PATH**: `design_docs/planned/v0_30_0/m-ailang-fmt-inline-interior-sprint-plan.md`
**SPRINT_JSON_PATH**: `.ailang/state/sprints/sprint_M-AILANG-FMT-INLINE-INTERIOR.json`
