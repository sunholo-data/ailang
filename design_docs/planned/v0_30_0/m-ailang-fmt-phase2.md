# M-AILANG-FMT-PHASE2: Phase 2 — Lossless Comment Preservation for `ailang fmt`

**Status**: ⛔ **PARKED — needs-human-review (design-quorum BLOCKED, 2 rounds)** — see the ⛔ block below
**Target**: v0.30.0
**Priority**: P1 (Phase 1 refuses 372/393 = 94.7% of the live example corpus; the formatter is unusable on normal commented code until this lands)
**Estimated**: 2–3 days (sprint-sized)
**Dependencies**: [M-AILANG-FMT Phase 1](../../implemented/v0_30_0/m-ailang-fmt.md) (IMPLEMENTED 2026-07-18)
**Refined from**: the "Phase 2: Lossless Attachment" section of the implemented Phase 1 doc (its technical decisions are preserved verbatim below and expanded into a standalone plan)
**Companion doc**: [M-AILANG-FMT-ADOPTION](m-ailang-fmt-adoption.md) — discoverability + opt-in hooks, **gated behind this doc**

## ⛔ Quorum Block (iteration 59, 2026-07-19) — needs-human-review

Created + revised this iteration on Mark's #399 directive ("Yep do the fmt design docs next"). Ran
the multi-provider design-quorum twice (the mission's bounded one-revision cap). The design
**converged but did not clear the gate** — R1's central-feasibility objections were fully resolved
(see Verification Log V14–V20: column unit resolved to 1-based NFC rune, AST spans proven unusable,
design pivoted to the token-anchored envelope, corpus-swept). R2 then surfaced **two new, narrower,
fixable defects**:

1. **`gpt5-6-sol` — parse-validity premise unproven.** V4's "393 files parse" claim ran Phase-1
   `ailang fmt`, whose comment preflight *refuses* 372 files **before** parsing — so it cannot
   establish that the refused majority actually parse. The 393-file formatting gate + round-trip
   baseline rest on that unverified premise. **Fix (small):** re-verify parse-validity with a
   parser-level sweep (`ailang check` per file, not `fmt`), quantify any non-parsing files, and
   restate the corpus gate over the *parse-valid* subset.
2. **`gemini-3-1-pro` — left-widening rule over-consumes the parent open-delimiter.** In
   `[ /* C */ x ]` the first child `x` widens left over the parent's `[` (its match `]` lies after
   the min-anchor), trapping comment `C` inside `x` instead of attaching it to the list's boundary 0
   — breaking attacher totality. **Fix (small):** the widening rule must stop at the nearest
   enclosing-list open-delimiter (never cross a boundary the parent owns); add `[ /* C */ x ]` and
   nested-bracket fixtures to the totality test.

**Both are tightly-scoped corrections on an otherwise-sound design, not architecture rejections.**
Per the mission's bounded quorum gate (one revision + one re-quorum), the item is **parked for the
human**: authorize one more short revision round to address the two objections, or amend the design.
Quorum artifacts: `.ailang/state/mission-quorum/m-ailang-fmt-phase2-2026-07-19T07-01-55Z.json` (R1),
`…-07-21-45Z.json` (R2). Metered cost both rounds: ~$0.16.

## Problem Statement

`ailang fmt` shipped in v0.30.0 (iter-56) as a canonical formatter, but Phase 1 is deliberately
comment-non-preserving: any file containing a real comment is refused before parsing (exit 2,
byte-identical, `comments are not yet supported by ailang fmt`). This was the correct fail-closed
choice — the lexer skips comments and the AST has no trivia fields, so a naive reprint would
silently delete user text — but it makes the formatter unusable on essentially all human- and
agent-authored code.

**Current State (all measured live 2026-07-19, see Verification Log):**

- 372 of 393 `examples/**/*.ail` files (94.7%) are refused with the comment error; only 21 format.
  (Iteration 58 measured 344/393 = 87.5% with a top-level-only variant of the sweep; the recursive
  live number is worse.)
- A commented file is *valid input* to every other tool: `ailang check` on the same file exits 0.
- The Phase 1 doc's own constraint: *"Until that phase lands, the Phase 1 refusal is mandatory and
  cannot be weakened to a warning."* Phase 2 is the only sanctioned unblock.

**Impact:**

- Developers and agents cannot format normal code; `--check` cannot be adopted in CI (it would fail
  on 94.7% of files with an *operational* error, not a drift report).
- The companion adoption doc (teaching-prompt line, harness hooks) is fully blocked: teaching agents
  a tool that refuses commented files would be worse than not teaching it.

## Goals

**Primary Goal:** `ailang fmt` formats commented AILANG files losslessly — every input comment
appears in the output exactly once, deterministically placed — unlocking the formatter for the
94.7% of the corpus Phase 1 refuses.

**Success Metrics:**

- **Corpus gate (hard precondition for refusal removal):** all 393 `examples/**/*.ail` files format
  with 0 comment-refusals AND 0 envelope/attachment errors. The two load-bearing envelope premises
  (rune-column conversion exactness, anchor↔token alignment) are already **measured corpus-wide at
  design time** (V14–V19: 81,224 tokens, exact for every token outside string-literal interiors,
  and the interior class is fully characterized and clamped by design). The one remaining corpus
  unknown — attachment-rule coverage — is turned into a measured gate in M3: the refusal path is
  deleted **only when the sweep shows zero residual errors**. So "every parse-valid corpus file
  formats" is an *enforced shipping precondition*, not a hope that coexists with fail-closed errors.
- **Fail-closed posture reconciled with the goal:** on the corpus, envelope/attachment errors are
  gated to exactly zero before release. Off-corpus files may still hit a fail-closed envelope error
  (exit 2, file untouched) — and every such occurrence is classified a **formatter defect** to be
  fixed (tracked like a bug via the envelope-error taxonomy), never an accepted steady-state
  outcome. Fail-closed is the *defect-surfacing mechanism during bring-up*, not a carve-out from
  the goal.
- Marker property test: every input comment carries a unique marker; formatted output contains
  every marker exactly once, and output is idempotent (`fmt(fmt(x)) == fmt(x)` byte-for-byte).
- Structural round-trip `Parse(fmt(x)) ≡ Parse(x)` (positions/spans/trivia ignored) extended to
  comment-preserving inputs over the full corpus.
- The Phase 1 refusal path is removed from `cmd/ailang/fmt.go`, and the exit-code contract is
  split: **exit 3 = input does not parse** (the expected mid-edit state that harness hooks defer
  on) vs **exit 2 = genuine operational error** (usage / read / print / round-trip / envelope /
  write). Required by the companion adoption doc's no-silent-fallback hook contract, which must
  distinguish "file not parseable yet" from "formatter broke" (see
  [M-AILANG-FMT-ADOPTION](m-ailang-fmt-adoption.md), Hook Contract).

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Node ranges come from a **token-anchored envelope** built on the lossless byte scan — NOT from converting/widening AST `Span`s (that option is refuted by measurement, V15–V17) and NOT from a parser span retrofit | Determines whether `internal/parser`/`internal/ast` construction sites change (they do NOT); wrong choice = large core conflict surface, or an architecture resting on spans that demonstrably don't exist for most node kinds | human | design | high |
| The 5 attachment rules and emission order are fixed as specified (verbatim from Phase 1 doc) | Attachment determinism is the correctness contract; any heuristic drift breaks idempotence | human | design | high |
| Envelope inconsistency **fails closed** (exit 2 for that file, no write), never guess-attaches; corpus-gated to zero occurrences before refusal removal | Silent misattachment = silent comment relocation, the exact failure Phase 1 refused to risk | human | design | med |
| Comment trivia lives in `internal/format` structures, NOT as `internal/ast` node fields | Keeps AST semantic nodes and `print.go` golden contract untouched | human | design | med |
| Exit-code split: **3 = input parse error, 2 = operational error** (today both are 2) | The adoption doc's non-silent hook contract cannot distinguish "mid-edit file, defer" from "formatter broke, surface" without it; folding it here keeps the CLI contract change in the doc that already owns `cmd/ailang/fmt.go`'s exit paths | human | design | low |
| Refusal removal happens only after the marker property test is green on the full corpus | Removing the gate early re-opens silent-deletion risk | agent (ordering within sprint) | compile | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] **Column unit RESOLVED (verified, not deferred):** `Token.Column`/`ast.Pos.Column` is a
  **1-based rune index into the NFC-normalized source** — established by reading
  `lexer.readChar()` (`internal/lexer/lexer.go:47-62`: one `column++` per
  `utf8.DecodeRuneInString`, after `Normalize()` at the `New()` boundary) and confirmed
  empirically with a multi-byte fixture and an 81,224-token corpus sweep (V14, V18).
- [x] Ranges via the token-anchored envelope built from the lossless byte scan; `internal/parser`
  node construction unchanged (decided here; evidence and rationale in Solution Design)
- [x] AST `Span`s are **not used at all** — measured as absent on all but 3 node kinds, with
  `End` = start-of-last-token and call-path-dependent `Start` where they do exist (V15–V17)
- [x] Attachment rules 1–5 and emission order as written (carried verbatim from the implemented Phase 1 doc)
- [x] Fail-closed on envelope/attachment inconsistency — exit 2, file untouched, no warning-downgrade
- [x] Exit-code split (3 = parse error, 2 = operational) lands with the refusal removal in M3

## Deferred Decisions

The following are intentionally left open for the implementer:

- Internal representation of the lossless scan result (separate comment slice + region slice vs one interleaved trivia record stream) — agent may choose, provided byte-exact `Start`/`End` per comment AND per literal region
- Whether the envelope (anchor table, bracket matcher, boundary resolution) is built lazily per-node or eagerly per-file — agent may choose
- Exact blank-line normalization *between* consecutive floating comments beyond "preserve relative blank-line grouping" (e.g., collapse 3+ blank lines to 1) — agent may choose, must be idempotent
- Test fixture organization under `internal/format/testdata/` — agent may choose
- Whether `HasComments` survives as an internal helper or is absorbed into the collector — agent may choose (its CLI refusal call-site is removed either way)

## Solution Design

### Overview

Phase 2 adds trivia without changing grammar. The lexer exposes comments and exact byte spans
through a separate lossless scan (extending the existing opt-in `ScanForComment` machinery into a
collector); the parser-visible token stream is byte-for-byte unchanged. A formatter-owned
**token-anchored envelope** supplies the structural boundaries attachment needs — built from the
byte-exact lossless scan plus verified anchor conversion of AST start positions, **never** from
AST `Span`s (measured as unusable, see the Design Decision below). Deterministic rules attach
every comment to an owner boundary; emission interleaves comments into the existing canonical
printer output. A marker property test proves no comment is ever lost, duplicated, or destabilized
under re-formatting.

### Comment and Attachment Model (verbatim from Phase 1 doc)

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

Attachment is deterministic:

1. A comment after a node on the same source line attaches **trailing** to the nearest preceding node.
2. A comment before the next node with no blank line attaches **leading** to that next node.
3. A comment separated by a blank line, between sibling nodes, or immediately before a closing
   delimiter attaches **floating** to the smallest enclosing ordered list at the boundary before the
   next child (or at `len(children)` before the close).
4. Comments before the module attach to the file at boundary zero; comments after the final top-level
   node attach to the file at the final boundary.
5. Consecutive comments preserve source order and relative blank-line grouping.

Emission order is: leading comments, node, same-line trailing comments, then boundary-floating
comments.

### Design Decision: Token-Anchored Envelope (AST spans are NOT the range source)

The Phase 1 doc left one open risk (its final Risks row): *"Existing AST spans are insufficient for
Phase 2 floating comments — do not guess attachment from start positions; require full ranges or a
parallel syntax-envelope index."* An earlier draft of this doc proposed converting AST line/col
span endpoints to byte offsets and "widening" them into full ranges. **Design-quorum review
challenged that premise, and design-time measurement refuted it.** The verified facts, and the
architecture they force, follow.

#### Verified facts (design-time probes + code reads; V14–V19 for full evidence)

1. **Column unit is RESOLVED: a 1-based rune index into NFC-normalized source.**
   `lexer.New()` normalizes input (BOM strip + NFC — `internal/lexer/lexer.go:32-44`), and
   `readChar()` does one `column++` per `utf8.DecodeRuneInString` (`lexer.go:47-62`). Empirical
   fixture: in `let x = "héπ😀" ; 1`, the `;` lands at column 16 (rune count) not 20 (byte count).
   Corollary: **normalization changes bytes** (a decomposed-é fixture: 12 raw bytes → 11
   normalized), so all envelope byte offsets are defined against the *normalized* source, which is
   exactly what the comment scanner already walks (`ScanForComment` calls `Normalize` first).
2. **AST `Span`s cannot be the range source — for most nodes they do not exist.** Exactly 3 node
   kinds carry a `Span` field (`FuncDecl`, `ModuleDecl`, `ImportDecl` — grep of `internal/ast`);
   every expression, block, pattern, and type node carries only a start `Pos`. Where `Span` does
   exist: `End` is `curPos()` under the parser's cursor-AT-last-token convention, i.e. the **start
   position of the last token** (probe: `module probe` has `Span.End` at col 8, the start of
   `probe`, excluding "robe"); and `Start` is **call-path-dependent** (probe: `export pure func` →
   `Span.Start` at `pure`; plain `pure func` → at `func` — so even `astedit`'s doc comment,
   "starts at `func`", is imprecise). `Pos.Offset` is populated nowhere (grep + probe: 0
   everywhere).
3. **A node's `Pos` is not, in general, its textual start.** Probe: `BinaryOp.Position()` is the
   **operator** token (`x + 42` → position of `+`); a paren-wrapped expression loses its `(`
   entirely (no `ast.ParenExpr` — Phase 1 V20). So "widen the node's start" was doubly unsound:
   there is no reliable start to widen.
4. **What DOES hold, by construction and by measurement: every parser-recorded `Pos` is copied
   verbatim from a real token's (line, column).** All `ast.Pos{...}` construction sites in
   `internal/parser` copy `curToken`/`peekToken`/saved token fields (exhaustive grep, V17). And
   the conversion `offset(line, col) = lineStart[line] + byteLen(first col−1 runes)` is **exact**:
   a design-time sweep over all 393 corpus files (81,224 tokens) found the converted offset lands
   precisely on the token's source text for **every token outside string-literal interiors**
   (V18). The only exceptions — 3,510 tokens, fully characterized — are **interpolation-queue
   tokens** whose positions point inside `${...}` string interiors (plus 8 nested-interpolation
   cases; tagged-quasiquote token positions were verified correct at their tag start). Comments
   can never occur inside a literal, and literals are re-emitted from the AST as atomic units, so
   the envelope **clamps any anchor falling inside a scanned literal region to that region's
   start** — the exception class is measured, bounded, and neutralized by construction.

#### Options considered

- **(a) Parser span retrofit** — make every `internal/parser` construction site populate byte
  offsets and full-extent spans. Generally useful (it is `astedit`'s documented "PRODUCTION fix"),
  but it touches every node-construction site in the parser, has a broad regression surface, and
  is a core change this feature does not need.
- **(b) Convert + widen AST spans (REJECTED — refuted by facts 2–3 above)** — the earlier draft's
  proposal. There are no ends to convert for ~all node kinds, existing ends stop at
  start-of-last-token, and starts are modifier- and operator-skewed. Any implementation would have
  been guesswork wearing a deterministic hat.
- **(c) Token-anchored envelope (CHOSEN)** — a formatter-owned structure that derives all ranges
  from the **byte-accurate lossless scan** (which reads normalized source bytes directly, so its
  offsets are exact *by construction*), using the AST only for what it verifiably provides
  (ordered structure + token-start anchors):
  1. **Lossless scan** (extends `comment_scan.go`): yields every comment `[start, end)` byte span
     AND a **literal-region map** (byte ranges of string/char/regex/quasiquote literals, with
     interpolation-aware nesting — fixing the current scanner's naive `skipString`, which ends a
     region at the first unescaped `"` and misclassifies nested interpolation; measured on
     `examples/runnable/directory_ops.ail`, V19).
  2. **Anchor conversion**: each AST node `Pos` → byte offset via the line-start table + rune walk
     (fact 1 makes this exact; fact 4 verifies it corpus-wide). Anchors inside a literal region
     are clamped to the region start. An anchor that maps to no code byte is an envelope error.
  3. **Child-boundary resolution** (replaces "widening"): for each ordered child list the printer
     emits (file top-level decls, block children, list/tuple/record elements, match arms, import
     lists — parser appends children in source order, so lists are source-ordered by
     construction), child *k*'s **head** = the minimum anchor over its subtree (the leftmost
     recorded token — for `x + 42` that is `x`, recovering what the operator-positioned `BinaryOp`
     node obscures), then widened **left** over a CLOSED token class that can only belong to this
     child: opening brackets whose match (byte-level bracket matching over code bytes — proper
     nesting is guaranteed for parse-valid input, and brackets inside comments/literals are
     excluded by the region map) lies at/after the min-anchor, and modifier keywords (`export`,
     `pure`). Sibling min-anchors are strictly increasing (children are disjoint and ordered), so
     boundaries between siblings are unambiguous; nested lists resolve by interval containment
     (subtree intervals nest). An unclassifiable token during widening is an envelope error —
     the class is extended only by explicit doc amendment, never heuristically.
  4. **Fail closed**: any inconsistency (anchor to no token, unclassifiable widening token,
     comment falling inside no boundary) is an **envelope error: exit 2 for that file, no write,
     no partial output** — narrowed to genuinely un-analyzable files instead of all commented
     files, and gated to **zero occurrences over the corpus** before the refusal is removed (M3).

  Note what this design does *not* need: per-node end positions. The attachment rules 1–5 are
  line- and blank-line-based; they consume comment byte spans + line numbers (exact, from the
  scan) and child-boundary token intervals (exact, from anchors + bracket matching). "Full node
  ranges" were never the real requirement — *boundaries between siblings and list edges* are.

This matches the program's default bias (extension over core change), keeps
`internal/parser`/`internal/ast` out of the diff entirely, and leaves option (a) available later as
an independent quality upgrade (see Future Work) — if edit-grade parser spans ever land, the
envelope's anchor-conversion step collapses to a field read with no API change to attachment or
emission.

**Premise-sweep test (M1, permanent):** the design-time corpus sweep (fact 4) is re-implemented as
a checked-in test — for every corpus file, every parse-path token's converted offset must land on
its source text (outside literal interiors) — so the two load-bearing premises are continuously
enforced against parser/lexer drift, not just verified once at design time.

### Invariants

For every parse-valid source `x`, **including commented files** (this extends Phase 1's invariant,
which held only for comment-free input):

```text
Parse(fmt(x)) ≡ Parse(x)              -- structural, ignoring positions/spans/trivia
fmt(fmt(x)) == fmt(x)                 -- byte-for-byte, comments included
markers(fmt(x)) == markers(x)         -- every comment survives exactly once, in source order
```

The idempotence obligation is stronger than it looks: emission must place each comment where the
attachment rules would **re-derive the same attachment** on the formatted output. Each of rules 1–5
needs a paired emission test proving the round-trip is a fixed point (e.g., a trailing comment must
be re-emitted on the same line as its owner, or the second pass would reclassify it as leading).

The constraint from the Phase 1 doc — *"the Phase 1 refusal is mandatory and cannot be weakened to
a warning"* — is **removed by this doc**: once the marker property test and corpus round-trip are
green, the refusal call-site in `cmd/ailang/fmt.go` is deleted. That removal IS the unblock for
[M-AILANG-FMT-ADOPTION](m-ailang-fmt-adoption.md).

### Architecture

**Components:**

1. **Lossless comment collector + literal-region map** (`internal/lexer`, extending
   `comment_scan.go`): opt-in scan returning `[]Comment` with exact byte spans and kinds PLUS the
   byte ranges of every string/char/regex/quasiquote literal; distinguishes real comments from
   `--`/`//` inside literals (the Phase 1 `ScanForComment` state machine already does this
   boolean-ly; Phase 2 makes it yield both comments and regions). **Must upgrade `skipString` to
   track `${...}` interpolation nesting** — the current early-termination at the first unescaped
   `"` is conservative-safe for the Phase 1 boolean (worst case: false positive → refusal) but
   would corrupt the region map (measured: nested interpolation in
   `examples/runnable/directory_ops.ail`, V19). `Lexer.NextToken()` and all parser-visible tokens
   remain byte-for-byte unchanged.
2. **Token-anchored envelope** (`internal/format`): line-start table + verified rune-walk anchor
   conversion + literal-region clamping + byte-level bracket matching + child-boundary resolution
   (min-anchor + closed-class left widening); fails closed on inconsistency. Includes the
   permanent corpus premise-sweep test.
3. **Attacher** (`internal/format`): implements rules 1–5 over (comments × boundaries), producing
   `[]Attachment`; total — every comment gets exactly one attachment or the file errors.
4. **Emission integration** (`internal/format`): the existing document-builder printers gain
   comment interleaving in the fixed order (leading / node / same-line trailing / boundary-floating);
   `internal/ast/print.go` remains untouched. Comment text is emitted verbatim from the
   *normalized* source (NFC-idempotent, so `fmt` stays idempotent; a non-NFC input file will have
   its comment bytes NFC-normalized on first format — consistent with the lexer's existing
   normalization boundary, and documented in `formatter.md`).
5. **CLI refusal removal + exit-code split** (`cmd/ailang/fmt.go`): delete the `HasComments`
   refusal path; wire the collector into the format pipeline; split exit codes — input parse
   errors (`parseForFmt` on the ORIGINAL source) exit **3**, all other operational failures
   (usage, read, print, round-trip re-parse/AST-diff, envelope, write) stay **2**; help text
   updated (Phase 1 limitation paragraph removed, exit-code table gains the 2/3 distinction).

### Files to Modify/Create

**New files:**
- `internal/format/envelope.go` (~300 LOC) — line-start table, rune-walk anchor conversion,
  literal-region clamping, bracket matching, child-boundary resolution, envelope-error taxonomy
- `internal/format/attach.go` (~250 LOC) — attachment rules 1–5, `Attachment` model
- `internal/format/attach_test.go`, `envelope_test.go`, `marker_property_test.go`,
  `premise_sweep_test.go` (~700 LOC total) — incl. the permanent corpus premise sweep (anchor
  conversion exactness over all example files)

**Modified files:**
- `internal/lexer/comment_scan.go` (+150/−30 LOC) — `ScanForComment` → comment collector with
  spans + literal-region map; interpolation-aware `skipString`
- `internal/format/comments.go` (+20/−10 LOC) — expose collector; `HasComments` retained or absorbed (deferred)
- `internal/format/format.go`, `decl.go`, `expr.go` (+150 LOC) — comment interleaving in emission
- `cmd/ailang/fmt.go` (+25/−25 LOC) — remove refusal path; exit-code split (3 = parse error);
  update help text
- `internal/format/corpus_test.go` (+40 LOC) — corpus sweep now includes commented examples
- `docs/docs/reference/formatter.md` (+35/−10 LOC) — Phase 2 behavior + exit-code split + NFC
  normalization note documented

## Examples

Input (verified with `ailang check`, exit 0 — Verification Log V10):

```ailang
-- File header: formatting demo.
module examples/fmt_demo

import std/io (println)

-- Adds two integers.
pure func add(x: int, y: int) -> int = x + y  -- trailing note

export func main() -> () ! {IO} {
  let s = add(1, 2);  -- compute the sum

  -- floating comment before the final expression
  println(show(s))
}
```

**Phase 1 today (verified, V2):** exit 2, `comments are not yet supported by ailang fmt`, file
byte-identical.

**Phase 2 target output** (canonical layout per the shipped Phase 1 rules — the comment-free
skeleton below was verified as `ailang fmt` output today, V11 — with comments re-attached per rules
1/2/3/4):

```ailang
-- File header: formatting demo.
module examples/fmt_demo

import std/io (println)

-- Adds two integers.
pure func add(x: int, y: int) -> int = x + y  -- trailing note

export func main() -> () ! {IO} {
  let s = add(1, 2)  -- compute the sum

  -- floating comment before the final expression
  println(show(s))
}
```

Rule walk-through: file header → rule 4 (file boundary 0); `-- Adds two integers.` → rule 2
(leading, no blank line before `pure func add`); `-- trailing note` and `-- compute the sum` →
rule 1 (same-line trailing; note the sequencing `;` is canonically dropped while the trailing
comment stays with its owner); `-- floating comment…` → rule 3 (blank line above → floating at the
boundary before `println(...)` in the enclosing block's child list).

## Milestones

### M1: Lossless Collector + Token-Anchored Envelope — 1 day

- [ ] **Premise-sweep test first** (`premise_sweep_test.go`): re-implement the design-time sweep
  (V18) as a permanent test — every parse-path token's rune-walk-converted offset lands on its
  source text, all corpus files, literal interiors exempted via the region map. Green before any
  attachment code exists.
- [ ] Extend `internal/lexer/comment_scan.go` into a comment collector + literal-region map:
  `[]Comment{Kind, Text, Start, End}` with byte-exact spans; interpolation-aware `skipString`
  (nested `${...}` fixture from `directory_ops.ail`); literal/quasiquote disambiguation preserved;
  parser token stream provably unchanged (`go test ./internal/lexer ./internal/parser`)
- [ ] Envelope: line-start table + anchor conversion + literal clamping; bracket matching over
  code bytes; child-boundary resolution (min-anchor + closed-class left widening over
  brackets/modifiers); envelope-error taxonomy; fail-closed wiring
- [ ] Unit tests: collector spans (incl. unicode + nested interpolation), boundary resolution over
  `export`/`pure` and paren-wrapped heads, envelope errors on constructed inconsistencies

### M2: Deterministic Attachment + Emission — 1 day

- [ ] Implement rules 1–5; totality check (every comment attached or file errors)
- [ ] Emission interleaving in the document builder: leading / node / same-line trailing / boundary-floating; rule-5 blank-line grouping
- [ ] Fixed-point tests per rule: each attachment class re-derives identically on its own output (idempotence at the rule level)

### M3: Property Gate, Refusal Removal + Exit Split, Corpus + Docs — 0.5–1 day

- [ ] Marker property test: unique marker per input comment; output contains each marker exactly once; `fmt(fmt(x)) == fmt(x)`
- [ ] Full-corpus sweep: all 393 `examples/**/*.ail` — parse-valid files format with **0
  comment-refusals AND 0 envelope/attachment errors** (hard gate: refusal removal blocked until
  zero), structural round-trip green
- [ ] Remove the refusal path in `cmd/ailang/fmt.go`; split exit codes (3 = input parse error,
  2 = operational error — the adoption doc's hook contract depends on this); update `fmt --help`
  and `docs/docs/reference/formatter.md` (exit table + NFC note)
- [ ] `make test`, `make verify-examples`, `make check-file-sizes` green

**Total: 2.5–3 days.**

## Conflict Surface

This feature is lexer/AST-adjacent but remains presentation-only. Same posture as Phase 1: the
semantic invariant is `Parse(fmt(x)) ≡ Parse(x)`, now extended to commented inputs.

### Syntactic positions touched

None. No grammar production, parser entry point, or token position changes. The parser-visible
token stream is byte-for-byte unchanged; comments continue to be skipped on the parse path.

### Areas touched

| Area | Relationship and constraint |
|---|---|
| `internal/lexer` | Extends the existing **opt-in** lossless scan (`comment_scan.go`, added in Phase 1 M4) from boolean to collector + literal-region map, incl. the interpolation-aware `skipString` upgrade. `NextToken()`/`skipComment()` behavior and every parser-visible token remain byte-for-byte unchanged; existing lexer tests (incl. `TestComments`) must pass unmodified. |
| `internal/ast` | **Read only. No node or field changes.** Structural boundaries come from the formatter-owned token-anchored envelope, not from AST trivia fields, `Span` reads (measured unusable, V15–V16), or span retrofits. (`grep -rin "comment\|trivia" internal/ast/*.go` is empty today and stays empty — V5.) |
| `internal/ast/print.go` | **Forbidden modification** (unchanged from Phase 1). Its normalized JSON output is a golden-test contract. |
| `internal/parser` | Used to parse input and reparse output for verification only. No construction-site changes; accepted grammar, AST shapes, spans, and diagnostics unchanged. R1/R2 equivalence and corpus AST-diff tests remain green. |
| `internal/format` | Owns envelope, attachment, emission, and all Phase 2 tests. The Phase 1 canonical layout rules are unchanged — comment interleaving composes with them. |
| `cmd/ailang` | Removes the comment-refusal call-site in `fmt.go`; updates help text. Exit-code table **changes shape deliberately**: exit 3 is introduced for input-parse errors, exit 2 remains for every genuine operational error (comment-preflight ceases to be a cause; envelope errors become one). This split is a REQUIREMENT of the adoption doc's non-silent hook contract (verified there, its V10: today all failure classes share exit 2 and are distinguishable only by fragile stderr-message inspection). |
| evaluator / types / effects / codegen / runtime | **No dependency and no modification.** Formatting never enters semantic compilation paths. |

### Programs that MUST still work

Regression fixtures (all exist; paths verified via the corpus sweep V4, which parsed all 393):

1. The 21 currently-formattable comment-free examples — Phase 2 output must be byte-identical to Phase 1 output for comment-free input (zero-comment regression gate)
2. `examples/hello_world.ail` (verified: exists, comment-refused today with exit 2; file-header comment → rule 4, leading decl comment → rule 2)
3. `examples/datetime_demo.ail` (verified: exists, comment-refused; consecutive header comment group with internal blank line → rule 5, effect-using module)
4. `examples/deriving_eq.ail` (verified: exists, comment-refused; leading comments on ADT/type declarations → rule 2 at declaration position)
5. `internal/format/testdata/` Phase 1 goldens — unchanged output for comment-free fixtures

(The three named examples are verified fixtures — each was confirmed to exist and to be
comment-refused (`ailang fmt` exit 2) on 2026-07-19, see Verification Log V13. They are samples of
attachment classes; the release gate remains M3's full 393-file sweep, so no example is silently
dropped.)

### What deliberately changes

- `ailang fmt` on a commented file: **exit 2 + refusal message → exit 0 + formatted output.** This
  is the entire point and is sanctioned by the Phase 1 doc's own phase plan.
- `ailang fmt` on a file that does not parse: **exit 2 → exit 3** (message unchanged). Exit 2 now
  means *only* genuine operational errors. Consumers of "nonzero = failed" are unaffected;
  consumers that need the distinction (the adoption doc's hooks) are the reason for the change.
- Comment bytes in output are NFC-normalized (matching the lexer's existing input-normalization
  boundary); for NFC input (the entire corpus today) output comments are byte-identical.
- No other behavioral change; `--check`/`--write` semantics, argument validation, and atomicity
  are untouched.

## Testing Strategy

**Unit tests:**
- Collector: spans, kinds, literal/quasiquote non-comments, nested-interpolation regions, unicode column handling
- Envelope: anchor conversion (incl. the permanent corpus premise sweep), literal clamping, bracket matching, boundary resolution, each envelope-error class (fail closed, never guess)
- Attacher: one test per rule 1–5 + boundary cases (comment at EOF with no trailing newline; comment between `;`-separated expressions; consecutive groups with/without blank lines)

**Property tests:**
- Marker preservation: every input comment's unique marker appears exactly once in output
- Idempotence: `fmt(fmt(x)) == fmt(x)` byte-for-byte over generated commented ASTs + corpus
- Structural round-trip: `Parse(fmt(x)) ≡ Parse(x)` (go-cmp, positions/spans ignored — same oracle as Phase 1 M3)

**Regression-surface tests:**
- One per "Programs that MUST still work" entry above; comment-free byte-identity vs Phase 1 goldens is the highest-value gate

**Manual testing:**
- `ailang fmt --check` over `examples/` in a working tree; eyeball a handful of diverse diffs before enabling `--write`

## Non-Goals

- No comment *reflowing*, re-wrapping, or content normalization — comment text is emitted verbatim
- No doc-comment semantics (`---` extraction, doc generation) — comments remain opaque trivia
- No parser span retrofit (option (a)) — deferred to Future Work; not a Phase 2 dependency
- No AST trivia fields — attachment metadata lives in `internal/format`
- No prompt/harness adoption work — that is [M-AILANG-FMT-ADOPTION](m-ailang-fmt-adoption.md), gated behind this doc
- No stdin, directory recursion, config files, or editor/LSP integration (unchanged from Phase 1)

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Emission is not a fixed point of attachment (comment reclassified on second pass → idempotence breaks) | High | Per-rule fixed-point tests in M2; marker property test in M3 runs `fmt` twice |
| Line/col→offset conversion wrong for multi-byte/unicode source | ~~High~~ **Retired at design time** | Column unit VERIFIED (rune index into NFC-normalized source, V14) and conversion exactness MEASURED over all 81,224 corpus tokens (V18); enforced permanently by the M1 premise-sweep test |
| Parser/lexer drift later breaks the anchor premise (a future construction site synthesizes a `Pos` that is not a token start) | Medium | M1 premise-sweep test is permanent and corpus-wide — drift fails CI loudly, and the envelope fails closed per file at runtime |
| Anchors inside string-literal interiors (interpolation-queue tokens; measured class of 3,510+8 on the corpus, V18–V19) | Medium | Clamped to the enclosing literal-region start by design; comments cannot occur inside literals and literals re-emit atomically, so clamping cannot misplace a comment |
| Boundary resolution meets a token outside the closed widening class (unknown prefix shape) | Medium | Envelope inconsistency fails closed (exit 2, no write) — a gap surfaces as a loud per-file error on the M3 corpus sweep (gated to zero), never as silent misattachment; class extended only via explicit doc amendment |
| Corpus contains a comment position the 5 rules don't cover | Medium | Attacher totality check errors the file; M3 sweep gates coverage to zero residual before refusal removal; rules extended only via explicit doc amendment |
| Collector drifts from `skipComment` (two comment grammars) | Medium | Collector tests assert agreement with parse-path skipping on shared fixtures; lexer tests unchanged |
| Nested-interpolation handling in the upgraded `skipString` is wrong | Medium | Dedicated fixtures from the measured cases (`directory_ops.ail`, V19); today's naive version errs conservative (false positive → refusal), so any regression direction is loud, not lossy |
| Scope creep into parser span retrofit | Low | Conflict Surface forbids `internal/parser`/`internal/ast` changes; envelope design makes retrofit unnecessary |

The open risk inherited from Phase 1 ("existing AST spans are insufficient") is now **resolved by
measurement, not assumption**: AST spans were probed, found insufficient (V15–V17), and removed
from the design entirely. The token-anchored envelope rests on two premises that are verified
corpus-wide at design time and re-enforced by a permanent test; any residual insufficiency
manifests as a fail-closed envelope error (gated to zero on the corpus), not a misplaced comment.

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Attachment and emission are fully rule-driven; no heuristics, no source-position guessing; formatter output is a deterministic function of source bytes |
| A2: Replayability | 0 | No trace impact; formatting never enters execution |
| A3: Effect Legibility | 0 | No effect changes |
| A4: Explicit Authority | 0 | No capability changes |
| A5: Bounded Verification | 0 | No verification changes |
| A6: Safe Concurrency | 0 | No concurrency impact |
| A7: Machines First | +1 | Lossless canonical formatting of real (commented) code is what agent pipelines need; unlocks `--check`/`--write` harness use on 94.7% of the corpus |
| A8: Minimal Syntax | 0 | No new syntax; grammar untouched |
| A9: Cost Visibility | 0 | No resource changes |
| A10: Composability | +1 | Composes with the shipped Phase 1 printer unchanged; envelope layer composes with (and is obsoleted gracefully by) a future span retrofit |
| A11: Structured Failure | +1 | Fail-closed envelope/attachment errors with a stable exit-code contract replace a blanket refusal; failure narrows to genuinely un-analyzable files |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +4** → **Decision: Proceed**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Not optimizing for human convenience over machine analysis

## Verification Log

All commands run 2026-07-19 in the main checkout (`dev`), installed binary
`AILANG v0.29.2-421-g81a45f2d8`.

**Revision pass (2026-07-19, same day):** a multi-provider design quorum blocked the original
draft on its central feasibility claim (AST line/col spans → unique tokens → widened full ranges,
with the column unit deferred to implementation). V14–V20 below are the design-time verification
that answered it: throwaway probes (in-repo Go programs importing `internal/lexer`/`internal/parser`,
run then deleted; no repo files modified) plus targeted code reads. Outcome: the objection was
CORRECT — AST spans were measured as unusable — and the design pivoted to the token-anchored
envelope, whose two load-bearing premises are now verified corpus-wide (81,224 tokens) rather than
assumed. The exit-code split (V20) was added for the companion adoption doc's non-silent hook
contract.

| # | Command / inspection | Observed result |
|---|---|---|
| V1 | `ailang fmt --help` | Exit 0. Documents stdout/`--write`/`--check`, exit codes 0/1/2, and the Phase 1 limitation paragraph ("files containing comments are refused (exit 2) and left byte-identical"). |
| V2 | `ailang fmt` on a temp commented file | Exit 2; stderr `<path>: comments are not yet supported by ailang fmt`; `cmp` confirms file byte-identical. |
| V3 | `ailang check` on the same commented file | Exit 0, `✓ No errors found!` — comments are valid input everywhere except `fmt`. |
| V4 | Recursive sweep: `find examples -name '*.ail'`, classify `ailang fmt` result per file | `total=393 ok=21 comment_refused=372 other_fail=0` → 94.7% refused, all with the comment message (no parse-error contamination in the count). |
| V5 | `grep -rin "comment\|trivia" internal/ast/*.go` (negative-existence) | Empty — the AST has no comment/trivia fields. Confirms trivia must live outside `internal/ast` (or the "no AST changes" decision would be moot). |
| V6 | Read `internal/format/comments.go` | `HasComments` delegates to `lexer.ScanForComment(source)`; documented as a lossless lexical preflight (literal/quasiquote-aware), boolean-only. Phase 2 extends this machinery to a collector. |
| V7 | Read `internal/lexer/comment_scan.go`, `lexer.go:118,136,308` | Opt-in `ScanForComment` exists separate from the parse path; `NextToken` calls `skipComment()` — comments never reach parser tokens. |
| V8 | `grep -rn "Offset:" internal/lexer/*.go internal/parser/*.go` (negative-existence) + read `internal/astedit/astedit.go:40-46` | Grep empty — nothing populates `Pos.Offset`. astedit documents: parser populates Span line/col **not byte Offset**; `Span.Start` **excludes** leading `export`/`pure` modifiers; production fix (edit-grade spans) is explicitly backlog (`m-ailang-native-harness.md`). Grounds the envelope-index decision. |
| V9 | Read `internal/ast/ast.go:41-53` | `Pos{Line, Column, File, Offset}` (Offset commented "for SID calculation"), `Span{Start, End Pos}`. The field exists; V8 shows it is unpopulated on the parse path. |
| V10 | `ailang check` on the Examples-section input (leading + trailing + floating comments, `;`-sequenced block) | Exit 0, `✓ No errors found!` — the shown syntax is valid AILANG. |
| V11 | `ailang fmt` on the comment-free variant of the same file | Exit 0; output is exactly the canonical skeleton shown in Examples (blank-line-separated decls, braced newline block, no `;`). |
| V12 | Phase 1 doc V9/V20 (cited, re-confirmed constraints) | `internal/ast/print.go` is a JSON golden-test contract (forbidden to modify); no `ast.ParenExpr` exists — both constraints carried into this doc's Conflict Surface unchanged. |
| V13 | `ls` + `ailang fmt` on the three cited regression fixtures | `examples/hello_world.ail`, `examples/datetime_demo.ail`, `examples/deriving_eq.ail` all exist, all comment-refused today (exit 2), and their head comments match the attachment classes cited for each (file-header / blank-line-grouped header block / leading-on-type-decl). An earlier draft cited three non-existent paths; corrected against `ls`. |
| V14 | **Column-unit probe** (throwaway Go program in-repo, importing `internal/lexer`; deleted after run): lex `let x = "héπ😀" ; 1`, print every token's (line, col) | `;` at **col 16** = 1-based rune count (string occupies rune-cols 9–14); byte counting would give col 20. Cross-read: `lexer.go:47-62` — one `column++` per `utf8.DecodeRuneInString`. **Column = rune index. Design-Freeze item RESOLVED at design time.** |
| V15 | Same probe: NFC check with decomposed é (`e` + U+0301) | `lexer.Normalize` shrinks 12 raw bytes → 11 normalized bytes. Positions are defined against NORMALIZED source; all envelope offsets must be too (and `ScanForComment` already normalizes identically). |
| V16 | Same probe: parse `module probe` + `export pure func add(…) { x + 42 }` + `pure func eq(n) = n + 1`; print `Pos`/`Span`/`Offset` per node | `ModuleDecl.Span.End` = col 8 = START of `probe` (End = start-of-last-token, excludes its text). `add` (export pure): `Pos`/`Span.Start` at **`pure`**; `eq` (pure only): at **`func`** — Span.Start is call-path-dependent, and `astedit`'s "starts at `func`" doc comment is itself imprecise. `BinaryOp.Position()` = the **operator** (`+`), mid-node. `Offset` = 0 on every node. Grep of `internal/ast`: exactly 3 node kinds carry `Span` (`FuncDecl`, `ModuleDecl`, `ImportDecl`); all others carry only start `Pos`. **AST spans are unusable as the range source — the earlier draft's convert-and-widen design is refuted.** |
| V17 | Exhaustive grep of `ast.Pos{` / `Line:` / `Column:` construction sites in `internal/parser` | Every recorded position copies a real token's fields (`curPos()`/`peekToken`/saved `p.curToken.Line` as in `parser_effect.go:61`). No synthesized/interpolated positions exist → "every AST Pos is a token start" holds by construction. Probe D confirmed: all 11 sampled node positions landed exactly on lexed token starts. |
| V18 | **Corpus-wide premise sweep** (throwaway probe #2/#3; deleted after run): for all 393 `examples/**/*.ail`, every `NextToken` token's (line,col) converted via line-start table + rune walk over normalized source; converted offset must land on the token's source text | 393 files, **81,224 tokens**: conversion **exact for every token outside string-literal interiors**. Mismatches: 3,510, ALL inside string-literal regions = interpolation-queue tokens (positions synthesized inside `${…}`), + 8 nested-interpolation tokens (also inside strings; see V19), + 8 tagged-quasiquote tokens (`sql"""`/`json{` in `examples/experimental/web_api.ail`) whose positions were **verified correct** at the tag start (probe's literal-vs-source matcher was too strict, not a position error). The exception class is fully characterized and confined to literal interiors, where comments cannot occur → clamped by design. |
| V19 | Mismatch classification probe: literal-region map (mirroring `comment_scan.go` states) applied to all V18 mismatches | 3,510/3,526 inside mapped regions; the 16 "outside" decompose into the 8 verified-correct quasiquote tags + 8 nested-interpolation tokens (`examples/runnable/directory_ops.ail:21`, `"${f("${base}/…")}"`-shaped) where the probe's region scanner — replicating `comment_scan.go`'s `skipString` — terminated the region at the first unescaped `"`. **Discovered real Phase-2 work item:** the collector's `skipString` must become interpolation-aware; today's naive version errs conservative (possible false-positive refusal, never data loss). |
| V20 | Read `cmd/ailang/fmt.go` error paths end-to-end | All failure classes (usage, read, comment-preflight, parse, print, round-trip, write) exit **2** with distinct stderr text only (`parseForFmt` → `"<path>: parse error: …"`). Exit codes alone cannot distinguish input-parse-refusal from operational errors → grounds the exit-3 split this doc now owns (required by the adoption doc's hook contract). |

## References

- **Refined from**: [M-AILANG-FMT (implemented Phase 1)](../../implemented/v0_30_0/m-ailang-fmt.md) — §"Phase 2: Lossless Attachment", Conflict Surface, and Risks rows referencing Phase 2. Model, rules, emission order, and property test carried verbatim.
- **Gates**: [M-AILANG-FMT-ADOPTION](m-ailang-fmt-adoption.md) — must not execute until this lands.
- **Span premise**: `internal/astedit/astedit.go` (span-handling doc comment — note V16 measured it as slightly imprecise: with `export pure`, Span.Start is at `pure`, not `func`) and [m-ailang-native-harness.md](../m-ailang-native-harness.md) (parser-route span backlog — the deferred option (a)).
- **Superseded stub**: [v0.29.0 formatter stub](../../planned/v0_29_0/m-ailang-fmt.md).
- **Greenlight**: Mark, GitHub issue #399 ("do the fmt design docs next").
- **Axiom reference**: [Design Axioms](/docs/references/axioms)

## Future Work

- **Edit-grade parser spans** (option (a)): byte offsets + full-decl boundaries incl. modifiers/annotations, benefiting `astedit` and LSP as well; when it lands, the envelope's conversion step collapses to a field read with no attachment/emission API change.
- **Doc-comment conventions**: once comments survive formatting, a doc-extraction convention becomes possible (separate design).
- **Directory recursion / stdin for `fmt`**: quality-of-life CLI extensions, unblocked but unscheduled.
