# M-FMT-PRINTER-LINE-WIDTH-LIMIT: A Width Predicate for the Printer's Existing Multi-Line Layouts, Then the Second Corpus Reformat

**Status**: Implemented (all milestones landed 2026-08-27, iteration 290)
**Milestones**: M0+M1+M1b `0c7f58351` (iter 287) · M2 `3ee848bb0` PR #928 (iter 289) · M3 corpus reformat (iter 290) — see "M3 results" at the end of this document
**Target**: v0.35.0
**Priority**: P0 (queue head by ruling — `D-39` places the width limit ahead of everything else in the fmt lane, and the fmt gate freeze is sequenced BEHIND it)
**Estimated**: 3 days (M0–M2 = 2 days, M3 corpus reformat = 1 day)
**Dependencies**: None (the multi-line emitters this design widens shipped in M-AILANG-FMT-INLINE-INTERIOR, v0.30.0)
**Queue row**: `m-fmt-printer-no-line-width-limit` (filed iteration 282; this doc uses the resolution-shaped name `m-fmt-printer-line-width-limit` — same item)
**Quorum**: authored unattended (mission loop, iteration 284 DESIGNER) → design-quorum REQUIRED before planning. The ruling itself (`D-39`) is already made; the concrete N and the construct scope were explicitly delegated to the loop ("part of your job is to propose the concrete N and the concrete scope"), so no freeze items remain — every decision below is agent-resolvable under that delegation. **Round 1 BLOCKED** (both reviewers present) on the under-specified width measurement; revised by the designer — see "Quorum Round 1 — Revision Record". **Round 2 BLOCKED** (both reviewers present) on two narrow width-arithmetic defects, neither disputing the design direction and both carrying concrete reviewer-authored fixes; the CONTROLLER applied them as a bounded refinement under the mission skill's narrow-refinement carve-out — see "Quorum Round 2 — Controller Refinement Record". One objection was CONFIRMED first-party and applied; the other was REFUTED by measurement and its underlying spec gap pinned as an invariant + AC anyway.

---

## Problem Statement

`ailang fmt` has **no line-width limit** — no width concept exists anywhere in the printer
(V3: a two-armed grep over `internal/format/*.go` non-test files finds **0** files with any
width token, while the same-scope control finds 14). Executing `D-38`(a) (reformat the
corpus to the emitter's output) made the consequence measurable: the printer collapses any
comment-free body onto ONE line **regardless of length**, so the corpus now contains a
**1,315-character line** (`examples/runnable/list_extremes.ail:24` — the entire `main()`
body), and **159 lines exceed 120 characters** across all 450 corpus files (V2). The corpus
is the teaching corpus: rendered on the website and shown to eval models.

Mark ruled in **`D-39`** (attended, 2026-08-26; full row at
[v1-mission.md](../../v1-mission.md), `grep '^| D-39 |'`):

- **YES, the printer needs a line-width limit** — option (a) of (a) add a gofmt-style wrap
  at N columns / (b) accept one-lining / (c) limit only the `let … in` chain collapse.
- **The corpus is reformatted a SECOND time once it lands** — that reformat is in scope
  here (M3), not a separate ask.
- **`D-38`(a) ratified the DIALECT, not the LINE LAYOUT** — so this is not a proposal
  against an affirmed form, and `m-fmt-typedecl-printer-needs-multiline-emit` is unblocked
  by the same ruling.
- **Sequencing**: do NOT wire or freeze the `fmt` gate until this lands and the second
  `fmt --write` pass has run — otherwise the collapsed form gets frozen as canonical by the
  gate rather than by a ruling. This design adds no gate wiring (Non-Goals, AC9).

## The Measurement That Decides the Scope

The controller's brief demanded this measurement rather than the assumption "the 159
over-long lines are all let-chains." **They are not.** Every one of the 159 lines >120
chars was extracted and classified by construct (V11), and segmented by whether its file is
`fmt --check`-canonical — i.e. whether the line is the **printer's own output** (V12:
**150 of 159 lines are**, across 84 of 88 distinct files; the other 9 lines sit in 4
fail-closed non-canonical files the printer never wrote).

| Construct | n | max | median | What fixes it |
|---|---|---|---|---|
| One-line `func` decl equation body (non-chain: if-chains, string builds, long calls, builtin wrappers) | 62 | 308 | 145 | **M2** (43 of the 72 M2 candidates), signature-heavy rest residual |
| Single `let x = <long value>` (mostly binding lines already inside multi-line chains) | 23 | 270 | 154 | intra-value reflow — **deferred** |
| `let … in` chain, 2+ bindings, collapsed to one line | 20 | **1315** | 173 | **M1** (18 fully; 2 leave one long binding line each) |
| `import` selector list | 10 | 223 | 173 | new canonical form + new attach list — **deferred** (comment-loss hazard class) |
| String-literal-dominated (one literal >110 chars on the line) | 10 | 263 | 157 | **nothing can** — an atom exceeds N; splitting a literal changes semantics |
| Record / list literal | 9 | 316 | 159 | intra-expression reflow — **deferred** |
| Other (nested calls, interpolations mid-expression) | 8 | 251 | 181 | intra-expression reflow — **deferred** |
| `type` declaration body | 6 | 227 | 185 | queued `m-fmt-typedecl-printer-needs-multiline-emit` — **out of scope here** |
| `match` arm | 5 | 187 | 140 | intra-expression reflow — **deferred** |
| `if`/`else if` chain (bare, non-decl lines) | 4 | 337 | 223 | intra-expression reflow — **deferred** |
| Comment | 2 | 153 | 153 | never — `fmt` does not rewrap prose |

Three consequences, each load-bearing:

1. **The catastrophic tail is entirely let-chains.** The top three lines (1315, 496, 485)
   are all collapsed chains; nothing else exceeds 340. Option (c) — chain-only — kills the
   tail. So **(c) is the correct first increment inside (a)**: M1 below IS (c), implemented
   as a width predicate rather than an unconditional expansion.
2. **(c) alone is nowhere near (a).** Chains are only 20 of 159 lines (13%). A second
   mechanism with the same reuse property — M2, moving a long equation body onto its own
   continuation line, a layout the printer ALREADY emits for attached chains — fixes a
   predicted 43 more (V14). M1+M2 ≈ **61 of 159 fixed, max line 1315 → ~316**.
3. **A hard "no line exceeds N" guarantee is unattainable and must not be an AC.** 10 lines
   carry a single string literal >110 chars (V14); 16 func-decl lines have the `=` at
   column ≥100 (the signature alone nearly fills the width — wrapping signatures is a far
   larger change). Like gofmt, the limit is best-effort at syntactic boundaries. The ACs
   below are therefore *measured-reduction* criteria with margins, not absolutes.

**Honest scope statement for the quorum:** this design executes (a) *partially* — it widens
the printer's two existing, idempotence-proven multi-line layouts with a width predicate,
reformats the corpus, and leaves a predicted residual of ~98 over-long lines (median ~150s,
max ~316) whose elimination needs a general intra-expression reflow engine (groups /
soft-lines) that does not exist yet (V9 — `doc.go`'s package comment *claims* "soft-line and
group primitives are provided"; the file is 67 lines and provides neither, nor column
tracking). Building that engine is a follow-on design, to be judged against the measured
residual after M3 — not smuggled into this sprint.

## Decision: N = 120, constant, not user-configurable

- **N = 120.** It is the damage metric `D-38`/`D-39` were measured in (57 → 147 → 159 lines
  >120); it is comfortable for website code blocks and side-by-side diffs. N=100 would put
  282 lines in scope (V2: >100 = 282) — +123 mostly-readable lines of extra churn in the
  second reformat, with no ruling-backed need. Revisit only with new evidence.
- **Not a CLI flag.** `ailang fmt` defines *canonical* AILANG; a user-settable width would
  make canonicity ambiguous per-machine. `Options.MaxWidth` exists so tests can probe both
  arms; the zero value resolves to 120 inside the package. This is a layout default — the
  fallback-policy carve-out for UI/layout defaults applies (CLAUDE.md §2), and it is the
  delivery path to the CLI, which passes zero-value `format.Options{}`
  ([cmd/ailang/fmt.go:130](../../../cmd/ailang/fmt.go#L130), V16).
- **The predicate is a function of (AST, current column) only** — never of the input's
  layout. That is the idempotence invariant (below).

## Solution Design

### The load-bearing existing mechanism (verified, not assumed)

The multi-line emitters already exist and are gated on ONE predicate at THREE sites, all of
the shape `owner has ≥1 comment attachment` (V4–V8):

- [internal/format/expr.go:265-270](../../../internal/format/expr.go#L265) — `letIn`:
  attached → `letChainMultiline` (one binding per line, constant indent, via the
  `flattenLetChain` helper SHARED with the comment attacher — the documented load-bearing
  requirement for idempotence); else `letInInline` (one line, any length).
- [internal/format/decl.go:174](../../../internal/format/decl.go#L174) — `funcBody`
  equation form: attached chain body → ` =` + hardline + indented chain (continuation
  layout); else ` = <body>` on one line.
- [internal/format/decl.go:572](../../../internal/format/decl.go#L572) — `topLevelLet`:
  same continuation layout for an attached value chain.

A two-armed repro (V10) proves the predicate is the only gate: the identical comment-free
program collapses to one line (88 chars) while adding one interior comment produces the
multi-line layout (max 33 chars) — and both the multi-line comment-free chain and the
continuation-body form **parse today** (`ailang check` rc=0 on both, fresh
`v0.34.0-2-gf45d4f0fe` binary), with `fmt` currently collapsing them back. So the canonical
multi-line layouts this design promotes are ALREADY the emitter's own output for commented
code: **no new layout form is invented anywhere in this sprint.**

### M0 — Width plumbing (0.5 day)

- `Options.MaxWidth int` in [internal/format/format.go](../../../internal/format/format.go)
  (zero value → 120, resolved once at printer construction).
- Column tracking in the `writer`
  ([internal/format/doc.go:14-19](../../../internal/format/doc.go#L14)): add a `col` field
  maintained by `write` (indent-aware on first write of a line) and reset by `hardline`.
  The struct currently tracks only `depth`/`atBOL`; the package comment's claim of column
  tracking and soft-line/group primitives is stale (V9) — fix the comment in the same
  commit so the next reader is not misled.
- The width measurement `inlineWidth(node) int` — specified in full in the next
  subsection, because round 1 of the quorum correctly rejected the earlier one-line
  description as ambiguous (printer-based vs AST-only) and therefore unimplementable.
- Unit tests for column tracking and the measurement, including a fixture where the same
  chain fits at column 0 but not at column 40, plus the bounded-depth and
  no-wrap-in-measurement tests specified under AC11/AC12.

### The width measurement — a mode-locked measurement printer, bounded and correct by construction

**Mechanism.** `inlineWidth(node)` constructs a *measurement printer* and renders `node`
with the SAME emitter code the real printer uses, then returns the rune count of the
rendering's FIRST line (up to the first `\n`, or the whole output when none). The
predicate at every M1/M2 site is
`p.w.col + pendingPrefix(site) + inlineWidth(node) > maxWidth`.

**`pendingPrefix` — the site-specific envelope. Added by the CONTROLLER at quorum round 2
under the narrow-refinement carve-out, applying `gpt5-6-sol`'s objection verbatim in
substance; see "Quorum Round 2 — Controller Refinement Record".** The round-1 formula
omitted the runes a site still has to write between the current column and the node, so a
line could exceed `MaxWidth` without tripping the predicate. **CONFIRMED first-party by the
controller** at the emission sites: in `funcBody`
([internal/format/decl.go:174](../../../internal/format/decl.go#L174)) the predicate is
consulted BEFORE `p.w.write(" =")`, so at predicate time `p.w.col` does not yet include the
` = ` the inline arm will emit. `pendingPrefix` is therefore a per-site constant naming
exactly those runes, and it is part of the M0 API rather than a caller's responsibility, so
a new site cannot silently omit it:

| Site | `pendingPrefix` | Rationale |
|---|---|---|
| `expr.go:266` (`letIn`) | `0` | the cursor is already at the chain's start; nothing pending |
| `decl.go:174` (`funcBody`, equation form) | `len(" = ")` = 3 | inline arm emits ` = ` then the body on the same line |
| `decl.go:572` (`topLevelLet`) | `len(" = ")` = 3 | same shape |

**AC13 (new).** For each of the three sites, a boundary pair: a body whose rendering lands
the candidate line at EXACTLY `MaxWidth` (must stay inline) and one at `MaxWidth + 1` (must
go multi-line), with the ` = ` included in the arithmetic. A mutant that sets every
`pendingPrefix` to `0` must flip the `MaxWidth`-boundary cases at the two decl sites and
must NOT flip the `letIn` case — that asymmetry is what proves the term is site-specific
rather than a constant fudge.

**What a measurement printer carries, and what it must NOT carry — enumerated, because
the package's one live sub-printer precedent inherits parent mode by default and that is
exactly the failure shape here.** `holeText`
([internal/format/interp.go:160](../../../internal/format/interp.go#L160), V21) builds
`&printer{w: newWriter(p.w.indent), att: p.att}` — it *deliberately* propagates `att` so a
comment rendered inside an interpolation hole is emitted and then caught by its
fail-closed `\n`/`--`/`${` refusal. A measurement printer must do the opposite on every
mode field:

| Field | Measurement printer | Why |
|---|---|---|
| `w` | fresh `newWriter(indent)` with **`depth` left at its zero value**, column 0 | measurement is position-independent; the caller adds `p.w.col`. **The zero `depth` is load-bearing and must never be seeded from the parent** — see the indent note below |
| `att` | **nil** — never the parent's | `hasAnyAttachment` is nil-safe BY DOCUMENTED DESIGN ([format.go:181](../../../internal/format/format.go#L181), V23: "With att:nil … it is always false, so comment-free input takes the exact old inline branch — the byte-identity invariant"), so every attachment-gated site takes the inline arm |
| `measuring` (new bool) | **true** | `exceedsWidth` returns `false` unconditionally when `measuring` is set — the width predicate is itself mode state and must be OFF inside a measurement |
| `maxWidth` | **not consulted** | follows from `measuring`; the measurement's result is invariant to the parent's MaxWidth (AC12 asserts this) |

**No indent double-count — the measurement writer's `depth` MUST stay zero. Added by the
CONTROLLER at quorum round 2 under the narrow-refinement carve-out, in response to
`gemini-3-1-pro`'s objection.** The objection **as stated is REFUTED by measurement**, and
the concern behind it is real, so both halves are recorded. `gemini-3-1-pro` argued that
`newWriter(indent)` starts `atBOL = true` and so prepends indent spaces that `p.w.col`
already accounts for, double-counting. Measured first-party by the controller in
[internal/format/doc.go:22-40](../../../internal/format/doc.go#L22): `newWriter` returns
`&writer{indent: indent, atBOL: true}` — `depth` is **left at its zero value** — and
`write` emits the pending indentation with `for i := 0; i < w.depth; i++`, i.e. **zero
times** on a fresh writer. So a measurement printer prepends NO indent and there is no
double-count. **But the doc did not SAY so**, and the natural implementation instinct — make
the measurement printer resemble its parent by seeding `depth = p.w.depth` — reintroduces
exactly the bug described, silently and in the premature-wrap direction. It is therefore
pinned here as an invariant rather than left to inference.

**AC14 (new).** `inlineWidth` returns the identical value when the PARENT printer is at
`depth` 0 and at `depth` 4 for the same node. The mutation arm — seed the measurement
writer with `depth: p.w.depth` — must make the two values DIFFER by exactly
`4 × len(indent)`. That mutant is the one `gemini-3-1-pro` predicted, so the AC fails
loudly for precisely the reason the objection named.

**Termination as a property, not a promise.** The ONLY construction site of a measurement
printer is inside `exceedsWidth`. In a measurement printer, `exceedsWidth` returns `false`
before constructing anything. Therefore the nesting depth of measurement printers is
**exactly 1** — not bounded-by-argument, but structurally incapable of reaching 2 — and
one predicate consultation costs exactly one O(|subtree|) render. Enforced by a test-hook
depth counter that AC11 asserts never reaches 2, over the full corpus AND a committed
depth-60 pathological fixture (value-position nested chains — verified expressible: V24
shows the depth-60 file type-checks rc=0 and today's `fmt` collapses it to one line).
Total cost per top-level decl: predicate sites are decl bodies plus `letIn` roots, and
`flattenLetChain` makes a k-binding chain ONE site (bindings are siblings, not nested
sites); nested sites arise only through *value-position* chains, giving worst-case
O(n · d) for subtree size n and value-nesting depth d — linear in practice, quadratic only
on fixtures like V24's, which AC11 caps with a runtime ceiling.

**Correctness of the measured value (the silent half of the round-1 objection).** Because
attachments are absent and `exceedsWidth` is pinned false, every layout decision inside a
measurement takes the inline arm — the rendering IS the hypothetical one-line layout,
byte-identical by construction since it runs the very `letInInline`/`expr` code the real
printer would run on that arm. The measurement can never see a width-wrapped multi-line
block, so it cannot return "total rune length of a wrapped rendering" (the non-monotonic,
idempotence-breaking wrong answer). Constructs that are inherently multi-line regardless
of mode (`match` cases, braced blocks — hardline emission at expr.go:402 et al.) emit
their hardlines inside a measurement too; that is why the measurement is defined as
FIRST-LINE width: it measures exactly what would land on the current line, which is what
the predicate is deciding about. This under-triggers when a construct's first line is
short but an interior line is long — those interior lines belong to constructs with their
own predicate sites (nested `letIn`) or to the deferred reflow engine, and stay in the
measured residual.

**Alternatives considered and rejected.**
- *Pure AST-only width function, no printer at all* (precedent for printer-free string
  building exists: `typeString` at [types.go:44](../../../internal/format/types.go#L44)
  and `algebraicType`/`constructor` at
  [decl.go:437](../../../internal/format/decl.go#L437), V22): rejected because it
  duplicates the spelling of every expression emitter — a second definition that can
  silently drift from the real inline arm, which breaks idempotence exactly the way the
  shared-`flattenLetChain` lesson (V8) warns against. One emitter, two modes, beats two
  emitters.
- *Memoized widths keyed on the envelope*: unnecessary given the depth-1 bound and the
  measured corpus shapes; adds an invalidation surface with no measured need.

### M1 — Width-widened chain predicate at all three sites (0.5–1 day)

At each of the three sites, the gate becomes:

```go
if n.Body != nil && (p.hasAnyAttachment(n) || p.exceedsWidth(n)) {
    return p.letChainMultiline(n)   // resp. the continuation layout at the decl sites
}
```

where `exceedsWidth` = `p.w.col + pendingPrefix(site) + inlineWidth(n) > maxWidth` (the
site-specific envelope is defined in M0), with `inlineWidth` the
mode-locked measurement defined in M0 (first-line width of the inline-arm rendering;
depth-1, never wrapping, MaxWidth-invariant). Attachment is checked FIRST
(short-circuit), so no attached case can regress to inline —
comment behavior is untouched by construction. This is the systemic form of the fix: one
predicate meaning, three call sites, changed together (audit-before-patch: V4 enumerates
the sites; there are no others).

Predicted effect (V13): 18 of the 20 collapsed chains fully clear 120; 2 leave exactly one
long binding line each (~126, ~139 — a single binding whose *value* is long; intra-value
reflow is deferred). The 1315/496/485 tail is eliminated. Note the poster child
`list_extremes.ail` drops 1315 → ~139, not to <120 — the ACs are written to that
measurement, not to a wish.

### M2 — Continuation layout for long equation bodies (0.5–1 day)

In `funcBody` (equation form, non-chain body) and `topLevelLet` (non-chain value): when the
one-line rendering `…<signature> = <body>` would exceed MaxWidth, emit the EXISTING
continuation layout instead — ` =` + hardline + body indented one level. Verified to parse
today (V10, arm B) and currently collapsed by `fmt`, so this is again a widening of an
existing layout's trigger, not a new form. Chain bodies are excluded (M1 owns them;
predicate order makes this explicit).

Predicted effect (V14): 72 candidate lines (long one-line func decls + top-level lets,
non-chain), of which 43 land under 120 at indent 2. The remainder (bodies >118 chars wide
on their own — long if-chains, wide literals) stay long on their continuation line, by
design, until an intra-expression engine exists.

### M3 — The second corpus reformat (1 day) — in scope per D-39

`ailang fmt --write` over the corpus under a freshly built, ldflags-stamped binary, with
**iteration 282's evidence discipline reproduced in full**:

1. Comment totals before/after via `lexer.CollectComments`, per-file join, **with a
   poisoned control arm that FIRES** (mutate one file's comment count and show the
   instrument catches it) — total must be unchanged (7865 → 7865 at iteration 282; re-count
   at M3 time since HEAD moves).
2. `ailang check` rc unchanged on every joined pair (iteration 282: 342/342).
3. Two-armed gates against a pristine base: `make verify-examples`, `verify-stdlib`,
   `test-stdlib-ail`, `go test ./internal/format/...` — rc recorded in BOTH arms. (Not
   `go build ./...` — it is rc=1 on pristine dev by design.)
4. The width metrics before/after: >120 count, >100 count, max line, per-construct residual
   classification (the follow-on reflow-engine decision reads THIS table).
5. Attach-refusals and parse-failures fail closed and stay untouched, as in iteration 282.

## Idempotence (first-class acceptance property)

A second `fmt --write` pass must be a byte-for-byte no-op. Why it holds:

- **The decision inputs are AST + column, both derived from the AST-driven rendering.**
  Pass 1: source is one 130-char line → AST → predicate fires → multi-line out. Pass 2:
  source is the multi-line form → **same AST** (whitespace-insensitive parse; V10 shows
  both spellings type-check identically) → same column at the decision point (the signature
  the printer just emitted is a function of the AST) → predicate fires again → identical
  output. No input-layout dependence, no oscillation: expansion never feeds back into the
  measurement, because `inlineWidth` measures the *hypothetical inline* rendering, not the
  emitted one — and the measurement is invariant to BOTH the input's layout (mode-locked
  inline arm, M0) and MaxWidth itself (never consulted while measuring, AC12), so the
  predicate is a monotone function of the AST and the decision cannot flip between
  passes.
- **The attacher and printer keep sharing `flattenLetChain`**
  ([internal/format/letchain.go](../../../internal/format/letchain.go)) — the
  inline-interior doc
  ([m-ailang-fmt-inline-interior.md](../../implemented/v0_30_0/m-ailang-fmt-inline-interior.md))
  documents this shared flattening as the load-bearing idempotence requirement. M1 changes
  *when* `letChainMultiline` runs, not what it emits and not the chain/boundary definition,
  so the agreement is preserved untouched.
- **Enforced, not argued:** the existing idempotence property suites
  (`marker_property_test.go`, `roundtrip_test.go`, `corpus_test.go`,
  `format_test.go:337`) run the full pipeline over the corpus and will exercise every
  width-expanded file automatically; M3's AC4 additionally asserts an empty `git diff`
  after a second corpus-wide pass.

## Comment safety (first-class acceptance property)

Iteration 281 measured the hazard class: **registering an attach list whose owner the
printer renders on one line converts a fail-closed refusal (rc=2, nothing written) into
SILENT COMMENT LOSS (rc=0, comments gone)** — `std/dom.ail` went 54 → 50 comment lines.
This design is in the safe direction on that axis, by construction:

- M1/M2 register **no new attach lists and no new owners** — they only widen the trigger
  for layouts whose attachment behavior already ships. Grep-provable at review: the diff
  touches no `attach.go` registration and adds no `registerList`-class call.
- The predicate ORDER (`hasAnyAttachment || exceedsWidth`) means every currently-multi-line
  attached case stays multi-line; a width-triggered expansion only ADDS line breaks to
  comment-free renderings.
- Import-list wrapping is **deferred precisely because** it needs a new canonical form plus
  a new attach list — the exact iteration-281 hazard shape. It belongs with the typedecl
  work, behind the same design discipline.
- Instrument: `TestCorpusCommentGate` (`corpus_comment_test.go:51`) plus M3's
  `lexer.CollectComments` before/after join with a poisoned control that fires.

## Conflict Surface

This is printer territory (`internal/format/`); enumeration verified by grep/read, not
inferred:

1. **`cmd/ailang/fmt.go:130`** — sole CLI consumer, passes zero-value `format.Options{}`
   (V16). Zero-value → 120 default inside the package is the delivery path; no cmd change.
2. **`scripts/hooks/format_ail.sh:69`** — the Claude Code hook runs `ailang fmt --write` on
   every `.ail` write (V17). The moment M1/M2 merge, every agent session on this repo
   starts emitting the new layout on touched files. Benign but visible as incidental diff
   churn in concurrent sessions until M3 normalizes the corpus — schedule M3 immediately
   after M1/M2 land, same sprint.
3. **Tests that pin current output shapes** (all in `internal/format/`):
   - `inline_interior_test.go` / `inline_interior_shape_test.go` — byte-exact fixtures and
     the 28-target surface-shape census for the attached-chain feature. Fixtures whose
     comment-free chains render ≤120 cols are unaffected; any fixture rendering wider flips
     **deliberately** (testing policy: update the fixture, do not weaken the assert).
   - `TestFmtOutputMatchesTaughtDialect` (`format_test.go:326`) — the dialect guard; its
     snippets must stay green (they are short; a red here means the width change leaked
     into dialect, which is a defect, not a fixture update).
   - `TestCorpusCommentGate` (`corpus_comment_test.go:51`), roundtrip/idempotence suites
     (`roundtrip_test.go`, `roundtrip_soundness_test.go`, `marker_property_test.go`,
     `corpus_test.go`, `totality_test.go`), `dialect_drift_test.go`,
     `node_coverage_test.go` — recompute over the pipeline; must pass unmodified. Baseline
     at HEAD is green (V15).
4. **The corpus itself** (`examples/` + `std/`, 450 files) — M3 rewrites the ~84 canonical
   files carrying over-long lines (V12) plus any file whose layout the new predicate
   changes. Website renders examples via raw-loader → rendering improves; `prompts/` is
   hand-written versioned markdown, NOT generated from `examples/` (measured at the D-39
   session), so `ailang prompt` teaching does not move. AC10 double-checks the prompts
   don't *contradict* the new layout.
5. **The fmt gate** — deliberately NOT wired here (`D-39` sequencing). The dormant gate
   surface (`fmt-check-ail` naming in `internal/cihygiene/gate_wiring_test.go`) is
   untouched; wiring is a follow-on after M3.
6. **What does NOT read `internal/format`**: LSP does not consume it; the only non-test
   importers outside the package are `cmd/ailang/fmt.go` and `internal/testing`
   fixtures (V16 grep). No eval-harness path formats code at runtime.
7. **Parser/attacher** — untouched. `flattenLetChain` and `attach.go` are read, not
   modified; the M-TAINT-TYPES-class risk (new syntax positions) does not arise because no
   syntax is added and both promoted layouts already parse (V10).

## Non-Goals

- **No general reflow engine** (groups/soft-lines, record/list literal wrapping, call-arg
  wrapping, if-chain and match-arm breaking, intra-value wrapping): ~98-line predicted
  residual, decided on M3's measured residual table in a follow-on design. Note for that
  design: `doc.go`'s comment notwithstanding, the primitives do NOT exist (V9).
- **No string literal splitting, ever** — changes program semantics.
- **No comment rewrapping** — `fmt` preserves prose.
- **No import-list wrapping** — new form + new attach list = iteration-281 hazard class;
  goes with `m-fmt-typedecl-printer-needs-multiline-emit` (unblocked by `D-39`, not yet a
  doc — V19).
- **No type-decl body layout** — same queued doc.
- **No function-signature wrapping** — 16 residual lines, separate question.
- **No fmt gate wiring or freeze** — `D-39` sequencing, AC9.
- **No CLI `--width` flag** — canonicity argument above.

## Success Criteria

Every AC is runnable and can fail; V-rows cover each command at the scope the AC reaches.
"Fresh binary" = ldflags-stamped build to a scratch dir, PATH-prepended (never
`make quick-install` on this shared rig).

- **AC1 (predicate, two-armed unit test)**: a comment-free chain whose inline rendering
  exceeds `MaxWidth` at its column emits the multi-line layout; the same chain under a
  large `MaxWidth` stays one-line, byte-identical to today's output. Fails today: V10 arm 1
  shows the 88-char one-liner is emitted regardless.
- **AC2 (tail kill, committed repro)**: `ailang fmt examples/runnable/set_operations.ail`
  (496-char line today, V13-predicted fully fixable) contains no line >120; and
  `examples/runnable/list_extremes.ail` max line drops from 1315 to ≤150 (predicted ~139 —
  NOT <120; one long binding value remains, honestly stated above).
- **AC3 (corpus reduction, after M3)**: over all 450 files, `lines>120` ≤ **105** (from
  159; prediction 98 + estimate margin) and max line ≤ **350** (from 1315; prediction
  ~316). Fails loudly if the predicate is unwired (values would stay 159/1315).
- **AC4 (idempotence)**: after M3's `fmt --write`, a second corpus-wide `fmt --write`
  yields `git diff --stat` = empty. Plus the property suites of AC7.
- **AC5 (comment safety)**: `lexer.CollectComments` totals unchanged across M3
  (before/after per-file join), with a poisoned control arm that FIRES; and
  `go test ./internal/format/ -run TestCorpusCommentGate` green.
- **AC6 (semantics)**: `ailang check` rc unchanged for every joined file pair across M3.
- **AC7 (suites)**: `go test ./internal/format/...` green (baseline green at HEAD, V15);
  two-armed `make verify-examples` / `verify-stdlib` / `test-stdlib-ail` vs pristine base.
  Not `go build ./...` (rc=1 on pristine dev by design).
- **AC8 (residual table)**: M3 publishes the per-construct residual classification (same
  classifier as V11) into the implementation report — the follow-on reflow decision's
  input.
- **AC9 (sequencing, negative)**: no fmt gate wiring added — `git diff` touches no CI
  workflow / `make ci` gate list; `internal/cihygiene/gate_wiring_test.go` unmodified.
- **AC10 (teaching surface)**: `grep` of `prompts/` for collapsed-chain teaching examples
  that would contradict the new canonical layout; any hit updated with
  `ailang check`-verified replacements (expected zero — prompts are hand-written and short).
- **AC11 (bounded measurement — fails precisely on the round-1 defect)**: a test-hook
  counter on measurement-printer construction asserts nesting depth **never reaches 2**
  across (a) the full 450-file corpus format and (b) the committed depth-60
  value-position-nested-chain fixture (shape verified expressible, V24), with the fixture
  run under a wall-clock ceiling (sub-second; exponential blowup at depth 60 cannot hide
  under any test timeout). A deliberately broken measurement printer that inherits
  `measuring=false` (mutation arm) must trip the depth assertion — proving the test can
  fail.
- **AC12 (measurement correctness — fails on the silent wrong-answer half)**: for the V10
  repro chain, `inlineWidth` returns exactly the rune count of its one-line rendering
  (88), and the value is IDENTICAL under parent `MaxWidth` 40 and 120 — if the
  measurement printer wrapped (the gemini-3-1-pro failure mode), the first-line width at
  MaxWidth 40 would differ, and the test fails. A second case measures a chain >120 wide
  and asserts the result equals the un-wrapped inline width, not the length of any
  wrapped block.

## Related Documents

- [m-ailang-fmt-inline-interior.md](../../implemented/v0_30_0/m-ailang-fmt-inline-interior.md)
  (+ sprint plan) — built the multi-line chain emitter and the shared flattening this
  design widens; its idempotence argument is inherited, not re-derived.
- [m-ailang-fmt.md](../../implemented/v0_30_0/m-ailang-fmt.md),
  [m-ailang-fmt-phase2.md](../../implemented/v0_30_0/m-ailang-fmt-phase2.md),
  [m-fmt-properties-printer-roundtrip.md](../../implemented/v0_30_0/m-fmt-properties-printer-roundtrip.md)
  — formatter foundation and property suites.
- [m-fmt-dialect-alignment.md](../../implemented/v0_32_0/m-fmt-dialect-alignment.md) — the
  dialect guard (`TestFmtOutputMatchesTaughtDialect`) this design must keep green.
- `design_docs/v1-mission.md` rows `D-38`, `D-39` — the rulings; iteration 281/282 log
  entries — the comment-loss hazard measurement and the first reformat's evidence
  discipline.
- `m-fmt-typedecl-printer-needs-multiline-emit` — queued (mission ledger; not yet a doc,
  V19): type-decl bodies + `tests […]` lists + (proposed here) import lists.

## Verification Log

All commands run 2026-08-26 in `/Users/voightkampff/dev/sunholo-data/ailang` on branch
`dev` at `f45d4f0fe` (worktree carrying only a concurrent agent's
`docs/static/benchmarks/` edits, untouched). Fresh binary `/tmp/mc_iter_bin/ailang` =
`v0.34.0` dev, commit `f45d4f0fe`(-dirty), built 2026-08-26T10:26Z; PATH-prepended for
every `ailang` invocation (PATH binary is stale v0.33.2 by design on this rig). Outputs
elided to load-bearing lines.

| # | Claim | Command | Observed |
|---|---|---|---|
| V1 | Corpus = 450 `.ail` files; roots `examples/` + `std/` exist (`stdlib/` is NOT a root — recurring trap) | `find examples std -name '*.ail' \| wc -l`; `test -d examples; test -d std; test -d stdlib` | `450`; examples EXISTS, std EXISTS, stdlib ABSENT (controller's independent same-day run agrees: 450 with both `-name` and `-iname`) |
| V2 | Damage: max 1315 @ `list_extremes.ail:24`; >120 = 159; >100 = 282; total 22,378 lines | `find examples std -name '*.ail' -print0 \| xargs -0 awk '{...}'` (max/FILENAME/FNR + >120 + >100 + NR totals) | `max=1315 @ examples/runnable/list_extremes.ail:24 lines>120=159`; `total=22378 >100=282 >120=159` |
| V3 | NO width concept in the printer (negative + same-scope control in one call) | `grep -lE '\b(80\|100\|120\|MaxWidth\|maxWidth\|lineWidth)\b' <non-test format .go>` then control `grep -lE '\bformat\b'` same file set | arm 1: `0` files; arm 2 (control): `14` files — the zero is a measurement |
| V4 | The attachment predicate has exactly THREE non-test call sites | `grep -rn "hasAnyAttachment" internal/format/*.go \| grep -v _test` | `decl.go:174`, `decl.go:572`, `expr.go:266` (+ definition `format.go:181`) — no others |
| V5 | `letIn` gates multiline on the predicate alone | read `internal/format/expr.go:265-270` | `if n.Body != nil && p.hasAnyAttachment(n) { return p.letChainMultiline(n) } return p.letInInline(n)` |
| V6 | `funcBody` already owns the continuation layout (` =` + hardline + indented chain) for attached chain bodies | read `internal/format/decl.go:162-189` | equation-form branch: `p.w.write(" ="); p.w.hardline(); p.w.indented(... letChainMultiline ...)` behind `hasAnyAttachment(let)` |
| V7 | `topLevelLet` mirrors the same continuation layout for attached value chains | read `internal/format/decl.go:557-583` | identical ` =` + hardline + indented branch behind `hasAnyAttachment(val)` |
| V8 | Flattening is SHARED attacher↔printer and documented as the idempotence requirement | read `internal/format/letchain.go:1-45` | header: "one flattening definition guarantees the attacher and the printer agree … the load-bearing requirement for idempotence"; `flattenLetChain` handles single-binding chains too |
| V9 | NEGATIVE: writer tracks NO column; NO soft-line/group primitives exist despite `doc.go`'s comment claiming both (with in-scope positive control) | `wc -l internal/format/doc.go`; `sed -n '68,200p'` (empty); `grep -n "column" doc.go`; `grep -rn "softline\|group\|Group" <non-test format .go>` | `doc.go` = 67 lines total; struct fields `buf, indent, depth, atBOL` only; "column" hits ONLY the line-7 comment; softline/group hits are unrelated `attach.go`/`format.go` comment prose (positive control: the grep instrument fires on those) |
| V10 | Two-armed predicate repro + syntax verification: (arm A) multi-line comment-free chain parses AND `fmt` collapses it to one 88-char line; (arm B) continuation body `func f() -> int =\n  42` parses AND `fmt` collapses it | `ailang check $tmp/a.ail; ailang fmt $tmp/a.ail; ailang check $tmp/b.ail; ailang fmt $tmp/b.ail` (fresh binary) | A: check rc=0, fmt emits `export func main() -> () ! {IO} = let a = 1 in let b = 2 in let c = 3 in println("done")`; B: check rc=0, fmt emits `export func answer() -> int = 42`. Matches controller's independent two-arm repro (comment arm → multi-line, max 33 chars) |
| V11 | Construct breakdown of ALL 159 lines (the scope-deciding table above) | extract each line via `awk length>120 {file:line:len}` → per-line classifier (regex over the literal text; string-literal spans measured) | `FUNC_DECL_ONELINE 62 / LET_SINGLE 23 / LET_CHAIN_2PLUS 20 / IMPORT 10 / STRING_DOM 10 / RECORD_LIST 9 / OTHER 8 / TYPE_DECL 6 / MATCH_ARM 5 / IF_CHAIN 4 / COMMENT 2` — sums to 159; top-10 lengths `1315, 496, 485, 337, 316, 308, 285, 280, 275, 271`; the 3 >400 are ALL chains |
| V12 | 150 of 159 over-long lines are in `fmt --check`-canonical files (printer's own output); 88 distinct files, 4 non-canonical | per-file `ailang fmt --check` (fresh binary) over the 88 files; join back to lines | 84 CANON / 4 NONCANON (`examples/runnable/ai_tool_loop.ail`, `examples/runnable/jwt_decode.ail`, `std/dom.ail`, `std/sem.ail`); 150 lines in CANON files |
| V13 | M1 prediction: 18/20 chains fully clear 120; residuals ~126 (`mcp_tools.ail:22`) and ~139 (`list_extremes.ail:24`) | simulate: split each chain line at top-level ` in let ` boundaries, +2 indent +` in`, max segment vs 120 | `chains=20 fully-fixed-by-M1=18 residual=2` with the two named maxsegs |
| V14 | M2 prediction: 72 candidates, 43 fixed by body-on-continuation-line; 16 func-decl lines have top-level `=` at col ≥100; 10 lines carry a >110-char string literal (unfixable floor) | paren/string-aware top-level `=` scan over the 159; `2+len(body)≤120` test; literal-span measure | `candidates: 72; fixed: 43`; `eq_col>=100: 16`; `literal>110: 10` |
| V15 | Format suite green at HEAD (baseline for AC7; also covers idempotence/roundtrip/comment-gate suites named in Conflict Surface) | `go test ./internal/format/` | `ok … 43.485s`, rc=0 |
| V16 | Sole CLI consumer passes zero-value Options; non-test importers of the package are only `cmd/ailang/fmt.go` + `internal/testing` | `grep -rn "format\.Format\|format\.Options" cmd/ internal/lsp/`; `grep -rln "internal/format" cmd/ internal/ --include='*.go' \| grep -v internal/format/` | `cmd/ailang/fmt.go:130: format.SourceWithComments(prog, src, format.Options{})`; importer list = `cmd/ailang/fmt.go`, `internal/testing/source_strip.go`, `internal/testing/named_test_assert_test.go` — no LSP hit |
| V17 | The write-hook consumer exists (conflict surface #2) | `grep -rn "fmt --check\|fmt --write" Makefile tools/ scripts/ .github/workflows/` | `scripts/hooks/format_ail.sh:69: ailang fmt --write "$file_path" …`; `tools/launchd/nightly-eval.sh:491` (comment) |
| V18 | The `D-39` ruling text relied on above | `grep -n '^\| D-39 \|' design_docs/v1-mission.md` (full row read) | row present at line 106; (a) YES + second reformat + sequencing + D-38 dialect-not-layout scoping, verbatim as summarized |
| V19 | NEGATIVE: no existing planned/implemented doc covers printer width; the typedecl item is a queue row, NOT yet a doc (with in-scope positive control) | `find design_docs -iname '*typedecl*' -o -iname '*width*'`; control `find design_docs -iname '*fmt*'` | width hits are only v0_7_0 `record-width-subtyping` (type-system, unrelated); zero typedecl files; control fires (9+ fmt docs incl. the v0_30_0 family) — duplicate-gate passes |
| V20 | Baseline instrument health: `ailang fmt` on this HEAD emits the collapsed forms (so AC1/AC2 FAIL today) | V10's fmt outputs + V2's corpus measurement | collapse observed on both repro arms; corpus carries the 1315-char line the ACs require the sprint to remove |
| V21 | The package's ONE sub-printer precedent inherits parent mode (`att`) by default — the round-1 failure shape is a live pattern, not hypothetical | VERIFIED BY CONTROLLER: `grep -rnE 'newPrinter\|&printer\{\|scratch' internal/format/*.go` excl. `_test.go` → 3 hits (control `p.w.write` = 160); site read first-party: `sed -n '150,175p' internal/format/interp.go` | Hits: `format.go:94`, `format.go:131` (constructors), `interp.go:160` `sub := &printer{w: newWriter(p.w.indent), att: p.att}` — `holeText` DELIBERATELY propagates `att` (its comment: comment-in-hole is "emitted and then caught" by the `\n`/`--`/`${` refusal). The measurement printer must therefore zero its mode fields EXPLICITLY |
| V22 | Printer-free string-building precedent exists for the rejected AST-only alternative | `sed -n '40,50p' internal/format/types.go; sed -n '437,455p' internal/format/decl.go` | `typeString` (types.go:42-44) returns a canonical source string; `algebraicType`/`constructor` (decl.go:437+) build via `strings.Join` — both are string-returning helpers, no sub-printer |
| V23 | `att: nil` makes `hasAnyAttachment` uniformly false BY DOCUMENTED DESIGN — the measurement printer's inline-arm guarantee is a maintained invariant, not luck | `sed -n '175,195p' internal/format/format.go` | `if p.att == nil { return false }` at the function head; the doc comment names it: "With att:nil (Source) or an empty index … it is always false, so comment-free input takes the exact old inline branch — the byte-identity invariant" |
| V24 | The pathological fixture shape (value-position nested let chains, depth 60) parses, type-checks, and is collapsed to ONE line by today's `fmt` (fresh binary) | generate depth-60 `let v0 = let v1 = … 1 in v59 … in v0` body; `ailang check $tmp/deep.ail; ailang fmt $tmp/deep.ail` | check: `✓ No errors found!` rc=0; fmt emits the entire body on one line (~1.1k chars) — the fixture for AC11 is real, and the naive-measurement hazard would have been reachable |

**Claims deliberately NOT logged as verified**: the M1/M2 fix-count predictions (V13, V14)
are *simulations over line text*, labeled as estimates and given margin in AC3; the
`prompts/`-not-generated-from-`examples/` claim is cited from the attended D-39 session's
measurement recorded in the ledger row (V18), not re-measured here.

## Quorum Round 1 — Revision Record

Round 1 (2026-08-26): **BLOCKED**, both reviewers present (`absent_reviewers: []`), same
defect from both, direction not disputed:

- **gpt5-6-sol**: `inlineWidth`'s scratch printer disabled only attachment handling, not
  width predicates → `exceedsWidth` could recurse into itself unboundedly.
- **gemini-3-1-pro**: the same recursion (geometric blowup), PLUS the silent half — a
  scratch printer defaulting to MaxWidth=120 would wrap nested expressions and return the
  rune length of a multi-line block instead of the hypothetical one-line width, making
  the predicate non-monotonic and idempotence-breaking.

Both objections accepted, neither argued. Root cause in the round-1 text: M0 called the
helper "AST-only" while the measurement subsection implied printer-based rendering — the
ambiguity itself was the defect. Revision (this document): the measurement is now
specified as a **mode-locked measurement printer** with every mode field enumerated
(`att: nil`, `measuring: true`, `maxWidth` unconsulted, fresh writer), a structural
termination property (measurement-printer nesting depth is exactly 1 because the sole
construction site short-circuits in measurement mode), a first-line-width definition that
cannot observe wrapping, two new failure-capable ACs (AC11 bounded-depth with a mutation
arm that trips it; AC12 MaxWidth-invariance of the measured value), and four new
Verification Log rows (V21 sub-printer precedent `interp.go:160` incl. the controller's
grep; V22 string-builder precedent; V23 `att:nil` documented invariant; V24 depth-60
fixture expressibility). The AST-only alternative was considered and rejected for
emitter-drift risk (one emitter in two modes beats two emitters that can disagree).

## Quorum Round 2 — Controller Refinement Record

**Round 2 verdict: BLOCKED**, both external reviewers present (`absent_reviewers: []`),
artifact `.ailang/state/mission-quorum/m-fmt-printer-line-width-limit-2026-08-26T10-54-54Z.json`.
Both objections landed on ONE surface — M0's width arithmetic — and neither disputed the
design DIRECTION (both accept the mode-locked measurement printer and the predicate shape;
they object to the formula's terms). Each carried a concrete, reviewer-authored fix. That
is the mission skill's **narrow-refinement carve-out** exactly, so the CONTROLLER applied
the reviewers' fixes as a bounded second revision and routed onward, rather than spending a
third designer run (the Fable diet's unit is one DOC = authoring + one protocol-mandated
revision, and both were spent at round 1). This is NOT force-passing: standing rule 2 still
forbids proceeding over a contested design direction, and neither objection contests one.

| Reviewer | Objection | Controller disposition |
|---|---|---|
| `gpt5-6-sol` | The predicate omits pending inline syntax (` = `) between `p.w.col` and the node at the two decl sites, so a line can exceed `MaxWidth` without tripping M1/M2; no site-specific envelope is defined and boundary behavior is unverified. | **CONFIRMED first-party**, then applied. `decl.go:174` consults the predicate BEFORE `p.w.write(" =")`, so `p.w.col` genuinely excludes those runes at predicate time. Added `pendingPrefix(site)` to the M0 API (per-site table; `0` for `letIn`, `len(" = ")` for both decl sites) and **AC13**, a per-site `MaxWidth`/`MaxWidth+1` boundary pair whose mutation arm (all prefixes → `0`) must flip the two decl sites and must NOT flip `letIn`. |
| `gemini-3-1-pro` | `inlineWidth` double-counts indentation: the fresh `newWriter(indent)` starts `atBOL = true`, so its first `write` prepends indent spaces already accounted for in `p.w.col`, wrapping lines prematurely. | **REFUTED AS STATED, by measurement — and the spec gap behind it is real, so both are recorded.** `newWriter` leaves `depth` at its zero value (`doc.go:22`), and `write` emits indentation `for i := 0; i < w.depth; i++` — **zero times** on a fresh writer. No double-count exists. But the doc never SAID `depth` must stay zero, and the natural instinct (seed `depth = p.w.depth` so the measurement resembles its parent) reintroduces precisely the described bug, silently, in the premature-wrap direction. Pinned as an invariant in the mode table plus **AC14**, whose mutation arm is literally the implementation `gemini-3-1-pro` predicted: seeding the parent's depth must make the measured value differ by exactly `4 × len(indent)`. |

**Ruled out (rule 3f — a reviewer refuted by measurement is the loop working):** the indent
double-count as an actual defect of the specified mechanism. The instrument was
`internal/format/doc.go:22-40`, read first-party; the same read is what established that
the *unstated* invariant is worth an AC.

**Round-count note (mission skill, iteration 257's rule).** Two rounds, and the objections
have LOCALISED onto one surface (M0's width arithmetic) rather than spreading — round 1 the
measurement printer's mode, round 2 the formula's terms. No reviewer has yet flipped to
pass, so the SPLIT signal is not triggered; if a third round blocks on M0 again while the
other milestones stay clean, the disposition is to split M0 out rather than revise again.

---

## M3 results — the second corpus reformat (iteration 290, 2026-08-27)

Executed by `codex:gpt-5.6-sol` at base `98467151b`; every number below was **re-derived
first-party by the controller outside the sandbox** before it was recorded. Where the two
disagreed, the disagreement is recorded rather than reconciled away.

**Scope.** 450 `.ail` files under `examples/` + `std/` (`test -d examples` YES, `test -d std`
YES, `test -d stdlib` **NO** — the path has never been `stdlib/`; the widened `-iname`
control also returns 450, so the enumerator is not case-blind). 50 files changed:
38 under `examples/`, 12 under `std/`.

### Width — AC3 MET

| Metric | Base (98467151b) | After M3 | AC3 requirement |
|---|---:|---:|---|
| lines > 120 runes | 159 | **100** | ≤ 105 ✅ |
| lines > 100 runes | 281 | 236 | — |
| max line (runes) | 1315 | **316** | ≤ 350 ✅ |
| max line location | `examples/runnable/list_extremes.ail:24` | `examples/runnable/ai_streaming.ail:58` | — |

⚠ **Instrument note, and it corrects this iteration's own controller baseline.** The
controller's directive asserted that a byte counter and a rune counter *agree*. They do at
the acceptance threshold (>120 = 159 base / 100 after, both ways) and on the maximum, and
they do **not** at >100: bytes give 282/237, runes give 281/236. The executor reported the
one-line discrepancy as a deviation rather than coercing it to the controller's stated
agreement, and the executor was right. Root cause, measured: **BSD `awk` on macOS returns
`length()` in BYTES even under `LC_ALL=en_US.UTF-8`** — a 3-byte, 1-rune character measures
**3**, where `python3` measures **1**. The controller's two "independent" arms were therefore
one instrument, so their agreement was an artifact of the shared reader, not evidence.
Re-measured with a genuinely rune-aware counter, the executor's numbers reproduce exactly.
None of this moves AC3, which is decided at >120 where the instruments agree.

### AC8 — per-construct residual classification (the sole input to the follow-on reflow decision)

Classifier reconstructed from V11's described lexical categories and **validated by
reproducing the published V11 base table exactly** (62/23/20/10/10/9/8/6/5/4/2, sum 159)
before being applied to the residual.

| Construct | Base >120 | Residual >120 after M3 |
|---|---:|---:|
| FUNC_DECL_ONELINE | 62 | 31 |
| LET_SINGLE | 23 | 9 |
| LET_CHAIN_2PLUS | 20 | **0** |
| IMPORT | 10 | 10 |
| STRING_DOM | 10 | 10 |
| RECORD_LIST | 9 | 10 |
| OTHER | 8 | 12 |
| TYPE_DECL | 6 | 6 |
| MATCH_ARM | 5 | 5 |
| IF_CHAIN | 4 | 5 |
| COMMENT | 2 | 2 |
| **Total** | **159** | **100** |

Reading for the follow-on: `LET_CHAIN_2PLUS` is **fully closed** (20 → 0) — that is M1
doing exactly what it was designed for. `IMPORT`, `STRING_DOM` and `RECORD_LIST` are
**untouched at 10 each**, which is by design: import-list wrapping needs a new canonical
form plus a new attach list (the iteration-281 comment-loss hazard shape) and is deferred
with the typedecl work. `FUNC_DECL_ONELINE` halved but is still the largest residual class
at 31, so it is where a reflow engine would buy the most.

### AC4 idempotence — MET

A second corpus-wide `fmt --write` produced **zero** content change: sha-256 manifest over
all 450 files, **0 differing rows**, verified independently by the controller.

⚠ **A second instrument note, on the controller's own drill.** The first controller run of
this check reported **16 differing rows** and looked like a genuine idempotence failure. It
was not: `find | xargs shasum` does **not** emit rows in a stable order, so an unsorted
manifest `diff` reports position swaps as content changes. Worse, the poisoned control fired
`rc=1` in exactly the predicted direction and so *agreed with the false finding for the wrong
reason*, and the byte-identical `cp` restore also read as a mismatch. Sorting both manifests
by path before comparing gives 0 rows for the idempotence check and 0 for the restore.
**A manifest is not comparable until it is ordered**, and a control that shares the broken
reader cannot separate the arms.

### AC5 comment safety — MET, with a control that fires

`lexer.CollectComments`, per-file join: **7,865 → 7,865** over **450** joined pairs, 0
mismatches. The poisoned arm **fired**:

```
MISMATCH examples/ai_devtools_workflow/discount_calculator.ail before=7 after=8
POISON_JOIN=450 MISMATCHES=1
```

and after reverting, `REVERTED_JOIN=450 MISMATCHES=0`.

Corroborated by a **second, independent** controller instrument that does not use the lexer
at all — lines containing `--`: **7,225 → 7,225**, with both a known-absent control (0) and a
known-present control (`module`, 517) firing. Different unit, same invariant.

### AC6 semantics — MET

`ailang check` rc per file, joined before/after: **450 pairs, 0 rc changes**, distribution
**374 rc=0 / 76 rc=1** on both sides. Controller-measured on both trees (the pristine base
being a separate checkout at the same commit), anti-vacuity floor asserted at 450, and the
inverted-predicate control returns 450.

The sprint plan's historical `405 / 38 / 7` is **not** an `ailang check` distribution — it is
the `ailang fmt` distribution (405 canonical / 38 attach-refusals / 7 parse-failures), which
also reproduces exactly. The executor caught the mislabel in the controller's directive.

### Fail-closed invariant — HOLDS

The 45 fail-closed files (38 `fmt` rc=2 attach-refusals + 7 rc=3 parse-failures) must be
byte-unchanged. `comm -12` over the sorted changed-file list and the sorted fail-closed list
is **EMPTY**, with the self-intersection control returning 50. This is the sharp invariant
over 45 of 450 that guards against a real regression hiding in a bulk diff.

### AC9 no gate wiring (negative) — MET

`git diff --name-only | grep -cE '^(\.github/|make/)'` = **0**, with a known-positive control
on a matching string returning **1**. `internal/cihygiene/gate_wiring_test.go` sha-256
unchanged. `internal/format/` untouched by this milestone (0 files, control 1).

### AC10 teaching surface — MET

8 `in let ` occurrences across `prompts/` (v0.12.1 and v0.16.0–v0.16.6), **each 112 BYTES / 110 runes**,
**0** exceeding 120 on either instrument, so **0 prompt files changed**. Recording the 8 is what distinguishes
"checked and clear" from "the grep pattern was wrong".

### Gates — all rc=0, base and after, re-run by the controller OUTSIDE the sandbox

| Gate | Base rc | After rc |
|---|---:|---:|
| `make verify-examples` | 0 | 0 |
| `make verify-stdlib` | 0 | 0 |
| `make test-stdlib-ail` | 0 | 0 |
| `go test ./internal/format/...` | 0 | 0 |
| `go build ./cmd/ailang ./internal/...` | 0 | 0 |
| `make check-changelog` | — | 0 |

`go build ./...` is **excluded by design**: it is rc=1 on pristine `dev` (`cmd/wasm` has no
native `main`), so it can neither pass nor attribute. Platform: **darwin/arm64**; the ubuntu
and windows legs were not run locally and are covered by CI.

### Independent evaluation (iteration 290, `sonnet`, own worktree)

**PASS 94/100, zero blocking.** Generator ≠ judge held on both axes: OpenAI wrote the
milestone, Anthropic judged it, and both are distinct from the `opus` controller. The judge
re-derived every named target with instruments it wrote itself rather than reusing the
executor's — its own comment-counting state machine, its own residual classifier, its own
rune counter — and all six named targets held, including a novel check neither the executor
nor the controller ran: **stripping all whitespace from the before/after content of all 50
changed files and diffing gives 0 mismatches**, which proves the diff is a pure layout
transformation with zero token-level change. That is a strictly stronger statement than
AC6's rc-preservation.

**Two non-blocking findings, both against this document, both reproduced first-party by the
controller and both FIXED above:**

1. **The AC8 *Base* column was arithmetically wrong.** As first written it read
   `62,23,20,10,10,10,9,8,6,5,4` — sum **167**, against a stated total of 159 — because the
   controller transcribed V11's ordered list into a labelled table and mis-aligned the tail
   by one position. V11 (Verification Log, above) is authoritative:
   `RECORD_LIST 9 / OTHER 8 / TYPE_DECL 6 / MATCH_ARM 5 / IF_CHAIN 4 / COMMENT 2`. Corrected,
   and the sums are now re-derived **from the file** rather than restated: Base 159,
   Residual 100. This is the transcribed-value defect the mission skill already names —
   a quantity copied out of prose is a claim about that prose, not a measurement.

2. **AC10's "112 runes" was 112 BYTES; the true rune count is 110.** The width came from
   `awk '{print length($0)}'`, i.e. the *same* BSD-awk byte/rune defect the controller had
   diagnosed and corrected for AC3 in this very document — and did not apply to the
   neighbouring criterion. The line carries one em dash (3 bytes, 1 rune), so the delta is
   exactly 2. AC10's verdict is unchanged (110 and 112 are both far below 120), but the
   label was wrong in an audited artifact. *Guard the helper, miss the call site*, inside the
   audit that found the helper.

**And one finding that is not a defect but is the most useful thing here — it is now
measured rather than assumed. All 50 corpus rewrites are pinned by NOTHING.** The judge
reverted `examples/ai_modes.ail` and `std/crypto.ail` to their base spellings (sha-confirmed
mutants) and re-ran every gate — `go test ./internal/format/...`, `verify-examples`,
`verify-stdlib`, `test-stdlib-ail` — and **all stayed rc=0**. The reason is structural:
`TestCorpusCommentFreeRoundTrips` asserts `Format(Format(x)) == Format(x)` and AST
round-trip; it never asserts `data == Format(data)`, so it cannot see a non-canonical
spelling. This is **by design** — AC9 and `D-39` sequence the fmt gate freeze explicitly
*behind* this work — but it means the corpus's new canonical form is **evidence-gated, not
CI-gated**, and any hand edit or concurrent agent can silently de-canonicalise it with every
gate staying green. That is the concrete argument for the gate-freeze follow-on, and it is
now a measurement rather than a plan.

Also noted, and out of scope for M3: the sprint plan's AC9 baseline hash prefix
(`8e805c026…`) does not match `internal/cihygiene/gate_wiring_test.go`'s actual hash
(`045b0336…`, identical on base and after). A stale planning-time snapshot; the comparison
AC9 actually needs — base versus after — holds.
