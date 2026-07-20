# M-COMMENTS-FOR-AI-AUTHORS: measured comment semantics for an AI-first language — style guidance, first-class doc-comments, contracts-as-documentation

**Status**: Planned — **direction RATIFIED by Mark 2026-07-20**: "guide a little in the prompt for
the right AILANG code style for comments, but also have first-class doc-comments, and encourage
use of the contract/test system as self-documentation as much as is reasonable" + "put it in for
our evals". Milestone order below is the ratified shape; quorum-at-pick reviews the details.
**Target**: M1–M2 v0.30.x road (prompt + evals, no language surface) · M3 v0.31.x (parser/AST)
**Priority**: P2 (accessibility research with a v1.x language payoff; M2's evidence also feeds the
cost-per-success KPI — comments are tokens on every agentic re-read)
**Estimated**: M1 ~0.5d · M2 ~1d + eval time · M3 ~2–3d · M4 ~0.5d (rolling)
**Dependencies**: fmt + adoption (LANDED — canonicalization is the enforcement arm of style
guidance); fmt polish pair (M3 interacts with the attacher; sequence M3 after it lands);
m-eval-fmt-weakmodel-ab (M2 SHARES its corpus-variant eval machinery — build once, run both)
**Author**: interactive session with Mark, 2026-07-20

---

## Problem statement

AILANG's authors are AIs, but its comment story is inherited wholesale from human languages:
free-text trivia, invisible to the AST (the root cause of fmt's entire attachment problem), never
checked, and costing tokens on every agentic re-read. 372/393 of the live example corpus is
commented — models comment because a billion human files trained them to — yet we have ZERO
measured evidence whether those comments help or harm the AI that must later READ and MODIFY the
code. Three distinct things travel in comments today:

1. **Truth-claims** ("x must be non-negative") — unchecked prose duplicating what `requires`/
   `ensures`/types/effect rows express checkably. Worst container for the content.
2. **Reasoning/intent** ("recursion over fold: depth blew up") — plausibly LOAD-BEARING for AI
   authors: conversation context dies at compaction, but the file survives. A comment is the only
   reasoning trace that persists inside the artifact (the mission's ruled-out ledger at micro
   scale; our compile-stuck data shows models re-chase dead ends they once escaped).
3. **Ritual noise** ("-- increment the counter") — in-distribution habit, pure token cost.

## Design — three ratified prongs + the experiment that grounds them

### M1 — prompt style guidance (cheap, immediate; prompt-manager lane)
Add a compact "Comments in AILANG" section to the teaching prompt (respecting the ≤1500-line diet
— this REPLACES prose elsewhere, net ≤15 lines): (a) truth-claims go in contracts/types, never
comments; (b) one intent comment where a non-obvious choice was made (the WHY, not the WHAT);
(c) no narration of self-evident code; (d) doc position = the line(s) directly above a
declaration (the future `---` home, so today's habit becomes tomorrow's syntax); (e) run
`ailang fmt` (canonicalization as enforcement). Verify against `ailang check` per the
prompt-accuracy rule.

### M2 — the evidence: comment-variant eval A/B (the "put it in for our evals" piece)
Modification-task benchmarks (the agent must READ + CHANGE existing code — where comments would
pay, unlike write-fresh tasks), one weak model (haiku; optionally one local model as replication),
N-run aggregates (noisy-agentic-metrics rule), same harness variants as m-eval-fmt-weakmodel-ab
(build the corpus-variant machinery ONCE):
- **V-strip**: corpus with all comments removed.
- **V-keep**: corpus as-is.
- **V-migrate**: truth-claim comments converted to `requires`/`ensures`/type ascriptions; intent
  comments kept; noise dropped (a one-time curated transform of the benchmark corpus).
Metrics: pass rate · convergence (compile-stuck/green-stability) · tokens-per-solve (comments are
context weight — the cost side of the ledger). Hypotheses registered UP FRONT: H1 V-keep ≥
V-strip on modification tasks (intent comments pay); H2 V-migrate ≥ V-keep (checked semantics
beat prose); H3 V-strip wins tokens-per-solve but loses convergence. Any outcome is publishable
signal; routing/prompt decisions cite it.

### M3 — first-class doc-comments (`---`, attached in the AST; v0.31 language work)
A `---` doc-comment binds to the FOLLOWING declaration as a real AST node (a `Doc` field on
decls — the trivia field the AST deliberately never had, added ONLY for this position):
- **Dissolves the fmt attachment problem at its root** for the doc position — a tree node needs
  no re-anchoring heuristics; ordinary `--` comments elsewhere keep the Phase-2 envelope.
- Machine-addressable docs: μRAG/`ailang docs`/LSP hover get per-declaration anchors for free.
- In-distribution: models already put doc-blocks above declarations; this makes the habit syntax.
Conflict Surface: lexer (new token) + parser (decl attachment) + `internal/format` (print `Doc`
verbatim-canonical) + `ast/print.go` golden JSON (additive field). NOT a general trivia system —
`---` in any non-decl position is a parse error with a teaching diagnostic (fail-loud, no silent
reinterpretation as `--`).

### M4 — contracts/tests as self-documentation (rolling, "as much as is reasonable" per Mark)
Teaching prompt + examples lead with the pattern: the signature + effect row + `requires`/
`ensures` + named test blocks ARE the documentation; prose only for what they can't say. Add a
"self-documenting function" exemplar to examples/. Explicitly NOT mandatory-contracts — the
"reasonable" boundary is: migrate a comment when a checker exists for its content, never invent
ceremony (`@decision` annotations stay OUT unless M2's evidence someday earns them).

## Non-goals
- Banning or auto-stripping comments (fights the training prior; weak models drift off-distribution).
- Mandatory doc-comments or contracts (ceremony without evidence).
- A general trivia/comment-preservation AST system (only the decl-doc position is first-classed).
- Structured `@decision`/intent annotations — parked pending M2 evidence.

## Verification log
| Claim | Method | Result |
|---|---|---|
| 372/393 corpus files commented | fmt Phase-1 preflight census | Confirmed (2026-07-18) |
| Comments are the fmt refusal root cause; contracts-can't-format is the non-comment residue | m-ailang-fmt-phase2 corpus gate (59/386 refusals) + m-fmt-properties repro | Confirmed |
| File content survives compaction; conversation doesn't | motoko compaction incidents (memory: teaching-in-system-role); mission file-based memory design | Confirmed pattern |
| Comment utility to AI readers | **UNMEASURED — M2 is the experiment** | open |
| AST has no trivia field today; spans unusable for attachment | fmt-phase2 design verification (V-series) | Confirmed |

## Related
- [m-ailang-fmt-phase2](../v0_30_0/m-ailang-fmt-phase2.md) / polish pair — canonicalization + the attachment machinery M3 partially retires
- m-eval-fmt-weakmodel-ab (charter queue) — shared eval-variant machinery; run as siblings
- m-contracts-as-code-vertical (clause 4) — M4's exemplar home
- Lilian Weng, harness engineering — file-system-as-memory: comments are the file-level instance

---
**Document created**: 2026-07-20 (interactive; expect quorum-at-pick)
