# Sprint Plan: M-AILANG-FMT-PHASE2 — Lossless Comment Preservation for `ailang fmt`

**Design doc**: [m-ailang-fmt-phase2.md](m-ailang-fmt-phase2.md)
**Status**: UNPARKED by human directive (Mark, 2026-07-19, commit `c624b456d`). Quorum-complete-by-decision — NOT re-quorumed.
**Target**: v0.30.0
**Sprint ID**: `M-AILANG-FMT-PHASE2`
**Estimated duration**: **3–3.5 days**
**Risk level**: Medium (lexer/AST-adjacent, but presentation-only; correctness-gated by property + corpus tests)

---

## Human directive constraints (BINDING — folded into milestones)

Mark resolved the two Rev-3 architecture objections BY DECISION (option "b" + recommendations). These are non-negotiable and appear below as concrete milestones/acceptance criteria:

1. **M0 = PRINTER CHILD-LIST CODE AUDIT is the FIRST milestone.** It is a deliverable with its own acceptance criterion (a *written, proven inventory* of every ordered child-list emission site in the printer), NOT a warm-up. The audit's findings BIND M1–M3: every site it enumerates must be covered by boundary resolution + a totality fixture. Any site found beyond the doc's partial list EXTENDS the inventory — it is never skipped.
2. **Interpolation = FAIL-CLOSED CARVE-OUT, not a general interpolation attacher.** The scope is a preflight that REFUSES (Phase-1-style: fail-closed, byte-identical file) any file containing a comment INSIDE a `${...}` hole. Silent deletion of an interior comment must be structurally impossible. Full interpolation-aware *attachment* is DEFERRED and evidence-gated on the measured refusal rate (expected ≈0, measured in the M3 corpus gate).

---

## Goal

`ailang fmt` formats commented AILANG files losslessly — every input comment appears in the output exactly once, deterministically placed — unlocking the formatter for the 94.7% of the corpus Phase 1 refuses (372/393 files). Ships via a formatter-owned **token-anchored envelope** (AST spans were proven unusable at design time; see doc V15–V17). Removes the Phase-1 blanket comment refusal, replacing it with a fail-closed envelope-error posture gated to zero occurrences over the parse-valid corpus, plus a fail-closed interpolation-comment carve-out.

## Velocity derivation

CHANGELOG LOC-scrape found no metrics for the last 14 days (mission has been in a design/quorum-heavy stretch, not implementation). Velocity is therefore anchored on the two closest *comparable completed sprints* plus the Phase-1 actual for the SAME subsystem:

| Reference | Domain | LOC | Days | LOC/day |
|---|---|---|---|---|
| Phase 1 `ailang fmt` (implemented, same subsystem) | printer + CLI + corpus round-trip harness, correctness-gated | (test-heavy) | 4 | — (gated by node-audit + idempotence, not typing) |
| `M-SYNTAX-AI-FORGIVING` | parser/format, test-heavy, fuzz-gated | 315 | 3 | ~105 |
| `M-ARITY-STYLE-DIAGNOSTIC` | diagnostic, small-surface | 100 | 2 | ~100 |

**Conclusion:** correctness-gated `internal/format`/lexer work runs at a conservative **~100–120 LOC/day**, and — as Phase 1 showed — the budget is set by *edge-case correctness gates* (idempotence, totality, corpus zero-error), not by typing volume. The doc's ~1500 LOC (300 envelope + 250 attach + ~700 tests + 150 lexer + 150 emission + ~40 corpus/docs) is heavily test/fixture LOC, so it plans out as a correctness-gated 2.5–3 day skeleton in the doc. Adding M0 (printer audit, ~0.5 day) yields **3–3.5 days**. This tracks Phase-1's own "Phase 2 = 2–3 days" estimate, +0.5 day for the human-directed audit.

---

## Milestones

### M0 — Printer Child-List Code Audit (proven inventory) — 0.5 day

**FIRST milestone. Deliverable = a written, source-verified inventory folded into the design doc BEFORE any attachment code is written.**

- [ ] Read every printer child-list emission site across `internal/format/*.go` (`format.go`, `decl.go`, `expr.go`, `pattern.go`, `types.go`, `literal.go`, `precedence.go`) and enumerate EVERY ordered child-list boundary the printer emits: params, type args, constructor args, record fields, annotations, effect rows, list/tuple/record elements, match arms, import lists, top-level decls, block children — **and any others discovered in the source**.
- [ ] For each site, record: the AST node kind, the printer function/line, the delimiter pair (`()`/`[]`/`{}`/effect-row braces/none), and whether it can carry a comment boundary.
- [ ] Cross-check the inventory against the doc's partial list (doc §"Child-boundary resolution" names: top-level decls, block children, list/tuple/record elements, match arms, import lists). Every site the doc omits is FLAGGED and ADDED — the audit's completeness claim is verified against printer source, not asserted.
- [ ] Fold the proven inventory into `m-ailang-fmt-phase2.md` as a new "Printer Child-List Inventory (M0, verified)" subsection under Solution Design.

**Acceptance criteria:**
- Written inventory exists in the design doc, enumerating every printer child-list emission site with node kind + source location + delimiter + comment-boundary applicability.
- Every site the doc's prior partial list omitted is explicitly listed and marked "added by M0 audit".
- M1's boundary-resolution coverage and M2's totality fixtures are keyed to THIS inventory (the audit BINDS the later milestones).

---

### M1 — Lossless Collector + Token-Anchored Envelope + Interpolation Carve-Out — 1 day

- [ ] **Premise-sweep test first** (`internal/format/premise_sweep_test.go`): re-implement the design-time V18 sweep as a permanent test — every parse-path token's rune-walk-converted offset lands on its source text over all corpus files, literal interiors exempted via the region map. Green before any attachment code exists.
- [ ] Extend `internal/lexer/comment_scan.go` from boolean `ScanForComment` into a comment collector + literal-region map: `[]Comment{Kind, Text, Start, End}` with byte-exact spans; literal/quasiquote disambiguation preserved; parser token stream provably unchanged (`go test ./internal/lexer ./internal/parser`).
- [ ] **Interpolation FAIL-CLOSED carve-out (BINDING CONSTRAINT 2):** the collector detects any comment token that falls INSIDE a `${...}` interpolation hole and surfaces it as a refusal signal. Wire a preflight so `ailang fmt` on such a file is REFUSED (fail-closed, exit 2, byte-identical file, clear message) — silent deletion of an interior comment is structurally impossible. NO general interpolation-aware attachment; `skipString` only needs enough interpolation-nesting awareness to correctly *classify* a comment as interior (fixing the naive first-`"` termination measured on `directory_ops.ail`, V19) so the carve-out cannot miss one.
- [ ] Envelope (`internal/format/envelope.go`): line-start table + rune-walk anchor conversion + literal-region clamping; byte-level bracket matching over code bytes; child-boundary resolution (min-anchor + closed-class left widening over brackets/modifiers) with the **hard left wall** (left-widening stops at the nearest enclosing-list open-delimiter; comments before the first child attach to the parent's boundary 0); envelope-error taxonomy; fail-closed wiring.
- [ ] Unit tests (`envelope_test.go`): collector spans (incl. unicode + nested-interpolation classification), boundary resolution over `export`/`pure` and paren-wrapped heads, the parent-open-delimiter hard left wall (`[ /* C */ x ]`, `[ /* C */ [ y ] ]`), envelope errors on constructed inconsistencies, **and the interpolation-comment refusal** (a `${ /* c */ … }` fixture → exit-2 refusal, file untouched).

**Acceptance criteria:**
- Premise-sweep test green over all corpus files.
- Parser/lexer token stream byte-for-byte unchanged (existing lexer tests incl. `TestComments` pass unmodified).
- Boundary resolution covers every site from the M0 inventory.
- A comment inside `${...}` is refused fail-closed (exit 2, byte-identical) with a clear message; verified by a dedicated fixture.

---

### M2 — Deterministic Attachment + Emission — 1 day

- [ ] Implement attachment rules 1–5 (`internal/format/attach.go`) + `Attachment` model; totality check — every comment attached to exactly one boundary, or the file errors fail-closed.
- [ ] **Totality fixtures keyed to the M0 inventory:** at minimum the hard-left-wall cases (`[ /* C */ x ]` → attaches to the list's boundary 0, NOT `x`; `[ /* C */ [ y ] ]` → outer list's boundary 0, not widened into the inner list) PLUS a comment-boundary fixture for each additional printer child-list site the M0 audit enumerated (params, type args, ctor args, record fields, annotations, effect rows, …).
- [ ] Emission interleaving in the document builder (`format.go`/`decl.go`/`expr.go`): fixed order leading / node / same-line trailing / boundary-floating; rule-5 blank-line grouping. `internal/ast/print.go` untouched.
- [ ] Fixed-point tests per rule: each attachment class re-derives identically on its own output (rule-level idempotence — e.g. a trailing comment re-emits on the same line as its owner so a second pass cannot reclassify it as leading).

**Acceptance criteria:**
- Every rule 1–5 has a passing unit test + a paired fixed-point (idempotence) test.
- Totality: every comment gets exactly one attachment or the file fails closed — proven over the hard-left-wall fixtures AND one fixture per M0-enumerated site.
- Comment-free output stays byte-identical to Phase 1 goldens (zero-comment regression gate).

---

### M3 — Property Gate, Refusal Removal + Exit Split, Corpus + Docs — 0.5–1 day

- [ ] Marker property test (`marker_property_test.go`): unique marker per input comment; formatted output contains each marker exactly once; `fmt(fmt(x)) == fmt(x)` byte-for-byte.
- [ ] **Corpus gate (hard precondition):** sweep over the **386 parse-valid** `examples/**/*.ail` files (of 393; the 7 non-parsing error-demo/experimental/bug fixtures enumerated in V21 exit 3 and are OUT of the gate) — every parse-valid file formats with **0 comment-refusals AND 0 envelope/attachment errors**; structural round-trip `Parse(fmt(x)) ≡ Parse(x)` green. **Measure and record the interpolation-comment refusal rate** (BINDING CONSTRAINT 2 evidence gate; expected ≈0 — any refusal is enumerated so the deferred full-interpolation follow-up is evidence-gated).
- [ ] Remove the `HasComments` refusal path in `cmd/ailang/fmt.go` (currently `fmt.go:103-108`) — **only after the marker property test + corpus round-trip are green.** Split exit codes: **3 = input parse error** (`parseForFmt` on original source), **2 = operational error** (usage / read / print / round-trip / envelope / write). Update `fmt --help` (remove the Phase-1 limitation paragraph at `fmt.go:280`; add the 2/3 exit distinction) and `docs/docs/reference/formatter.md` (exit table + NFC-normalization note).
- [ ] `make test`, `make verify-examples`, `make check-file-sizes` green.

**Acceptance criteria:**
- Marker property test + idempotence green on generated commented ASTs AND the corpus.
- 386/386 parse-valid corpus files format with zero comment-refusals and zero envelope/attachment errors (CI-gated); refusal path removed only after this is green.
- Interpolation-comment refusal rate measured and recorded in the doc's Verification Log.
- Exit-code split live (3 = parse error, 2 = operational); help + `formatter.md` updated.
- `make check-file-sizes` green (watch `decl.go`/`expr.go` — see note below).

**Total: 3–3.5 days.**

---

## CI-gated regression fixtures (per human directive)

Every milestone is independently testable and CI-gated:
- **Round-trip byte-identity on commented corpus files** — M3 corpus sweep + marker property test.
- **Refusal on `${...}`-comment files** — M1 interpolation carve-out fixture (`${ /* c */ … }` → exit 2, byte-identical).
- **Zero-comment byte-identity vs Phase 1 goldens** — M2 (highest-value regression gate).
- **Per-M0-site totality fixtures** — M2, keyed to the audit inventory.
- **Permanent premise-sweep** — M1, corpus-wide, guards parser/lexer drift.

## Files (from doc §"Files to Modify/Create")

**New:** `internal/format/envelope.go` (~300), `internal/format/attach.go` (~250), `internal/format/{attach_test,envelope_test,marker_property_test,premise_sweep_test}.go` (~700 total).
**Modified:** `internal/lexer/comment_scan.go` (+150/−30, incl. interpolation-classification for the carve-out), `internal/format/comments.go` (+20/−10), `internal/format/{format,decl,expr}.go` (+150), `cmd/ailang/fmt.go` (+25/−25), `internal/format/corpus_test.go` (+40), `docs/docs/reference/formatter.md` (+35/−10).

## Notes for the controller (live-checks against HEAD)

All doc premises live-checked accurate against `dev` HEAD (v0.30.0):
- Corpus is exactly 393 `.ail` files (matches V4/V21).
- AST `Span` field present only in `ast.go` + `ast_decl.go` (the 3 node kinds — V16 accurate).
- `HasComments` refusal at `fmt.go:103-108` with the exact refusal message; all exits are `2` (no `3` exists yet — confirms the exit-split is net-new).
- `ScanForComment` is boolean with the interpolation-naive `skipString` at `comment_scan.go:102` — exactly the machinery M1 extends.
- Phase-1 limitation help text present at `fmt.go:280` (M3 removes it).
- **File-size watch (not a blocker):** `internal/format/decl.go` is 486 LOC and `expr.go` is 437; the +150 LOC emission interleaving lands across them. Under the 800 hard limit, but if concentrated in `decl.go` it could reach ~560-600 — executor should split emission helpers if `make check-file-sizes` approaches the limit.
- **No STALE premises found.** (Nothing to fix; flagged for record only.)
