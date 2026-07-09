# M-FABLE-STRATEGY-REVIEW: What AILANG Should Do To Be the Best AI Programming Language

**Status**: Planned (strategy review — routes into other docs, does not ship code itself)
**Target**: Direction for v0.29 – v0.34
**Priority**: P0 (strategic direction)
**Estimated**: Review itself: done. Routed work: sized in each follow-up doc.
**Dependencies**: [PROGRAM.md](../PROGRAM.md) (routing rule + invariants)
**Author**: Claude Fable 5 (frontier-model perspective audit, requested by Mark, 2026-07-08)

---

## Why this document is unusual

AILANG's stated customer is the AI model, not the human ("Machines First", A7). This review
was written by one of those customers. I am the model that currently scores highest on
AILANG's own benchmark suite, and this doc is my answer to the question: *"What could AILANG
do to help you?"* Every claim below is either verified against a live `ailang check`/CLI run
(see Verification Log) or cited to a dated artifact in this repo.

## The headline finding

**The strongest models already write AILANG better than Python. The gap is not capability —
it is cost, and accessibility for mid-tier models.**

From the v0.25.0 baseline ([docs/static/benchmarks/latest.json](../../docs/static/benchmarks/latest.json), 2026-06-21, 814 runs):

| Signal | AILANG | Python | Read |
|---|---|---|---|
| claude-fable-5 zero-shot | **97%** | 91% | Frontier models *prefer* AILANG's explicitness |
| gemini-3-1-pro zero-shot | **91%** | 89% | Same effect, second frontier model |
| claude-sonnet-4-6 zero-shot | 75% | **91%** | Mid-tier models lean on Python priors |
| All models zero-shot | 83.3% | 87.5% | Aggregate gap is a mid-tier gap |
| Agent mode (all) | 94.1% | ~96.4% | Feedback loop nearly erases the prior advantage |
| Zero-shot → agent uplift | **+18.2 pts** | smaller | The loop is worth more than the prior |
| Cost per success (zero-shot) | $0.052 | $0.005 | **10× — the real deficit** |
| Cost per success (agent) | $0.094 | $0.033 | 2.8× — better, still losing |
| Avg tokens per solution | 466 | 366 | 27% verbosity tax |

Three strategic conclusions fall out of this table:

1. **Time is on AILANG's side — but only on the capability axis.** As models improve,
   AILANG's relative pass-rate position improves *for free* (Fable and Gemini 3.1 Pro already
   invert the gap). Nothing about the cost axis improves for free.
2. **Feedback beats priors.** Agent mode recovers almost the entire Python gap. Python's
   billions of lines of training data are worth less than one good compiler iteration.
   Therefore the highest-leverage investments make **each loop iteration cheaper and more
   informative**, not the first shot more likely.
3. **The headline KPI should change.** Pass rate is nearly saturated at the frontier.
   The metric that decides whether anyone *uses* AILANG is **cost-per-success ratio vs
   Python**, tracked per tier. Everything below is in service of that number and of the
   mid-tier pass gap.

## Axiom Compliance

This is a strategy review; the scores reflect the aggregate direction of the recommendations.

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | R8 (trace flywheel) monetizes determinism as training data; R7 mandates a linear-time regex engine |
| A2: Replayability | +1 | R8 depends on and strengthens replayable traces |
| A3: Effect Legibility | +1 | R6 makes the effect system the marketed differentiator |
| A4: Explicit Authority | +1 | R6 showcases capability-gated `std/ai`; no ambient authority added anywhere |
| A5: Bounded Verification | +1 | R2 puts check/verify inside the default agent loop |
| A6: Safe Concurrency | 0 | No change |
| A7: Machines First | +1 | The entire doc optimizes for the machine customer (token cost, structured feedback) |
| A8: Minimal Syntax | +1 | R4 removes footguns; explicitly rejects Python-lookalike sugar (Non-Goals) |
| A9: Cost Visibility | +1 | Cost-per-success becomes the headline KPI; per-run cost already traced |
| A10: Composability | 0 | No change |
| A11: Structured Failure | +1 | R1/R5 extend structured, suggestion-carrying diagnostics |
| A12: System Boundary | 0 | No change |

**Net Score: +9** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): This doc *is* the machine's requirements list

## Problem Statement

AILANG has won the argument it set out to win: with explicit effects, HM types, and a
compiler in the loop, a frontier model writes it more reliably than Python. But three
problems block it from being the *best* AI programming language rather than a promising one:

**Current State:**
- **Cost:** 10× cost-per-success zero-shot, 2.8× in agent mode (v0.25.0 baseline). Part of
  this is a 2,518-line teaching prompt ([prompts/v0.16.2.md](../../prompts/v0.16.2.md)) paid
  on every single run, part is a 27% solution-token verbosity tax, part is extra agent turns
  (6.2 vs 4.4 on opencode).
- **Mid-tier accessibility:** sonnet-class models score 75% AILANG vs 91% Python. Their
  failures cluster on a *finite, documented* footgun list (block-body vs expression-body,
  curried vs multi-arg arity, match-in-lambda parser limitation, polymorphic arithmetic
  panic) — see [docs/LIMITATIONS.md](../../docs/LIMITATIONS.md) and the prompt's Common
  Mistakes table.
- **Under-leveraged moats:** the things no mainstream language can copy — Z3 contracts,
  capability-gated `std/ai`, deterministic replay — are not in the default agent loop, not
  in the default benchmark rotation, and not the marketing lead. M-THREE-CAMPS
  ([v0_22_0/m-three-camps-language-survey.md](v0_22_0/m-three-camps-language-survey.md))
  found AILANG is the only full verification-camp member with strong orchestration-camp
  membership, and that the orchestration strength is "dramatically under-benchmarked and
  under-surfaced."

**Impact:**
- Every eval run, every motoko session, every downstream consumer pays the cost gap today.
- The mid-tier gap caps adoption: most real-world agent fleets run sonnet/flash-class
  models for cost reasons, exactly the tier where AILANG currently loses.

## Goals

**Primary Goal:** Make AILANG the language with the lowest *cost per verified-correct
program* for AI authors — not the highest zero-shot pass rate.

**Success Metrics:**
- Cost-per-success ratio (AILANG/Python): zero-shot 10× → **≤ 3×**; agent 2.8× → **≤ 1.5×** by v0.32 baseline.
- Mid-tier gap: sonnet-class AILANG−Python delta −16 pts → **≥ −5 pts**.
- Teaching prompt: every Common Mistakes entry covered by a self-explaining diagnostic, and
  the corresponding prompt lines **deleted** — target ≤ 1,500 lines with no pass-rate loss.
- `ai_effect_*` / orchestration benchmarks in the default rotation with published results.
- First external release of a trace-derived (error → fix) dataset.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Headline KPI becomes cost-per-success ratio vs Python (not pass rate) | Redirects all eval/prompt/harness prioritization | human | design | high |
| Teaching moves from prompt-time to error-time; prompt shrinks as diagnostics improve | Converts a per-run token tax into a one-time compiler investment | human | design | med |
| `ailang check` pre-flight becomes mandatory in the motoko loop (extension lane) | Cheapest possible iteration: fail in ms before burning model tokens | human | design | low |
| Target vertical = verified AI-orchestration programs, not general-purpose Python parity | Focuses stdlib/benchmark/marketing effort where AILANG is unmatched | human | design | high |
| Publish trace-derived training data externally | Attacks training-data gravity; future models arrive with AILANG priors | human | runtime | med |

### Design Freeze

- [ ] Mark confirms cost-per-success as the headline dashboard KPI (affects eval dashboard + post-release reporting)
- [ ] Mark confirms the orchestration vertical as the marketing/benchmark lead (affects docs site + README positioning)
- [ ] Mark decides whether trace-derived datasets may be published externally (licensing/privacy call — R8 blocked until then; internal use may proceed)

## The Recommendations

Each is routed per PROGRAM.md §4: **AILANG fix** | **motoko extension** | **harness/eval** —
never motoko core. Overlapping planned docs are cross-linked, not duplicated.

### R1 (P0, AILANG fix): Error-time teaching — make the compiler the prompt

The single best diagnostic in the language today (verified live, v0.28.0):

```
`++` is for lists only. For strings use "${expr}" interpolation, concat([parts]), or join(sep, parts).
```

That error *is* a teaching prompt, delivered in ~25 tokens, exactly when needed, only when
needed. Compare: the same rule occupies permanent real estate in a 2,518-line prompt paid on
every run. As the model on the receiving end: **I do not need to be warned about mistakes I
haven't made. I need the error I actually hit to tell me what to write instead.**

**Proposal:**
1. Build a **footgun coverage table**: every entry in the prompt's Common Mistakes section
   maps to an error code whose message carries a concrete fix (like `++` above), with the
   `suggestion` field populated in `check --json` output.
2. CI-enforce it: a test fixture per footgun asserting the diagnostic text includes the fix.
3. **Then delete the prompt lines.** Track "prompt tokens deleted per release" as a KPI.
   A prompt line that survives a release must justify why it can't be a diagnostic instead.

Known current gaps to close first (from the audit): no "did you mean?" for typos'd
identifiers; cryptic parse errors on complex expressions; arity mismatch (curried vs
multi-arg) reports a type error without naming the *call-style* mistake.

*Overlaps:* [m-agent-ergonomics.md](m-agent-ergonomics.md) (dialect-slip loop) supplies the
*data* for which footguns fire most; R1 is the *mechanism* that retires them.

### R2 (P0, motoko extension): Verification in the default loop — the oracle advantage

Python gives a model priors. AILANG can give a model an **oracle**: state the contract
first, then let `check`/`verify`/Z3 report when the implementation satisfies it. That
converts "generate and hope" into "search with a termination condition" — the mode where
LLMs are strongest. The pieces exist (`requires`/`ensures` + Z3, `astedit.InjectContract`,
`ailang check --format=agent`) but none are in the default motoko loop: the audit found
motoko runs autonomously with **no `ailang check` pre-flight**, so a type error is
discovered by running, after the tokens are spent.

**Proposal (in priority order):**
1. **Check pre-flight extension:** after every file write, run `ailang check --format=agent`
   and inject the one-line diagnostics into the next step. Milliseconds of compute replacing
   a full model turn. This is the cheapest cost-per-success lever in the whole program.
2. **Contract-first task framing:** benchmarks/tasks deliver the `requires`/`ensures` spec;
   grading is contract verification, not stdout diffing (reference-free, un-gameable).
3. `ailang verify` surfaced in the agent-prompt as the "am I done?" command.

*Overlaps:* [motoko-mission.md](../motoko-mission.md) R2 lever ("AILANG-native verifier
power") — this recommendation says: sequence it **first** among the levers, because it
compounds with every other loop improvement.

### R3 (P0, harness/eval): The cost-per-success program

27% more solution tokens, more turns, and a 2,518-line prompt multiply into the 10× gap.
Attack all three, with measurement first:

1. **Pass-rate-per-prompt-token curve:** A/B the teaching prompt at several sizes
   (agent-prompt ~180 lines → full 2,518) per model tier. We have never measured where the
   curve flattens. My prediction as the consumer: for frontier models it flattens *very*
   early once R1 diagnostics exist; mid-tier models need the middle sizes.
2. **Prompt-caching-friendly structure:** stable prefix ordering so the system prompt is a
   cache hit across a session's steps (motoko already tracks cache-read tokens — report the
   cache-hit rate per session as a rig metric).
3. **Turn-count reduction** falls out of R2's pre-flight (fewer discover-the-error turns).

*Overlaps:* [m-ailang-semantic-context.md](m-ailang-semantic-context.md) (motoko's 3–10×
verbosity vs pi) is the same program from the context side; adopt its truncated-tool-results
work as part of this.

### R4 (P1, AILANG fix): Burn down the mid-tier footgun list

The sonnet-class 75%-vs-91% gap is not diffuse — it concentrates on documented, *finite*
friction. Several entries are parser/type limitations, not design choices:

- `match` inside block-body lambdas in HOF arguments fails to parse (M-DX-MATCH-HOF,
  [docs/LIMITATIONS.md](../../docs/LIMITATIONS.md)) — workaround is "extract a helper".
- Polymorphic **arithmetic** in lambdas panics while polymorphic *comparison* works
  (LIMITATIONS.md, v0.4.0 partial fix) — an inconsistency a model cannot predict.
- Arity style mismatch (curried vs multi-arg) produces a generic type error (see R1).

Every fix here does triple duty: removes a mid-tier failure mode, deletes prompt lines (R1
KPI), and removes a LIMITATIONS.md entry. Each needs its own design doc with the mandatory
Conflict Surface section (parser changes).

**Explicitly not in this bucket:** things that look like footguns but are load-bearing
design (explicit `!{IO}` effects, selective imports, `show()` for stringification). Models
learn those once; they are the product.

### R5 (P1, AILANG CLI): One cheap answer for every question I'd otherwise grep for

As an agent, my expensive moments are discovery: *what does module M export? what is the
type of f? what changed when I edited this file?* Today: `iface --compact` (good, recently
ADT-aware), `check --json`/`--format=agent` (good), `run -json` (structured **errors** only
— verified v0.28.0). Missing:

1. A structured **run result envelope** (outcome, exit reason, contract results, budget
   report in one JSON object) — today success-vs-failure parsing of `run` output is ad hoc.
2. An **`ailang ast-query`** surface (find-refs / type-of / exports-of) — the LSP already
   computes all of this but speaks only editor protocol; expose the same answers as CLI JSON.

*Overlaps:* [m-motoko-editdecl-astedit.md](m-motoko-editdecl-astedit.md) (AST-anchored
edits) is the write side of this; R5 is the read side. Same infrastructure.

### R6 (P1, eval + docs): Own the orchestration vertical

The programs the world will most need in 2026–2028 are **programs that call AIs**: pipelines,
agent fleets, tool servers. AILANG is the only language where an LLM call is a typed,
capability-gated, budgeted, replayable effect (`std/ai`, `!{AI}`, effect budgets, IFC
labels for the secrets agents notoriously leak). M-THREE-CAMPS built 14 gap benchmarks
including `ai_effect_summarize`, `ai_effect_json_schema`, `multi_agent_handoff` — and they
sit outside the default rotation.

**Proposal:** promote the orchestration benchmarks into the core rotation; make the flagship
docs example a *verified multi-step AI pipeline* (typed LLM calls + budget + secret-flow
enforcement + replay); lead the README/site positioning with it. "General-purpose language
that happens to be AI-friendly" is a losing pitch against Python; "the language where AI
orchestration is type-checked" has no competitor per the THREE-CAMPS survey.

### R7 (P2, AILANG fix): Stdlib workaround-killers — verified list only

The research sweep claimed four stdlib gaps; live verification killed two (base64 exists in
`std/bytes`; datetime *parsing* exists: `_dt_parseISODate`, `_dt_parseRFC3339`). Confirmed:

- **Regex: absent** (0 builtins, no std module). Every string-extraction task becomes
  30–50 lines of manual parsing — pure logic-error surface (logic errors are the dominant
  failure class for strong models: gpt5-5 had 1 compile vs 9 logic errors in v0.20.0).
  Mandate a **linear-time engine (RE2-style)**: no catastrophic backtracking preserves A1
  determinism and A9 cost-visibility.
- **URL parsing: absent** (`_net_url_encode` exists; no parse/split into scheme/host/path/query).

Route through the builtin-developer skill, one doc each.

### R8 (P2, extension/observatory): The trace flywheel — determinism as a training-data moat

The strategic answer to training-data gravity is not fighting Python's corpus — it is
**manufacturing a better corpus**. Every motoko session that converges error → green is a
labeled repair trajectory; AILANG's determinism (A1/A2) makes each one *verifiable by
replay* — a property no mainstream language's training data has. The observatory already
captures everything needed (session JSONL, spans, costs, diagnostics).

**Proposal:** an observatory exporter emitting (diagnostic, failing code, fix diff,
verification result) tuples; use internally for prompt regression testing immediately;
publish externally once the Design Freeze licensing decision clears. Success = future model
generations arriving with native AILANG priors, shrinking the teaching prompt toward zero —
the logical endpoint of R1.

## Solution Design

### Overview

This review ships no code. Its implementation is: (1) agree the Design Freeze items,
(2) update PROGRAM.md §7 to reference this review and adopt the KPI, (3) open the follow-up
design docs below in priority order, each through the normal doc → sprint → verify cycle.

### Routing table (implementation plan)

**Phase 1 — measure & cheap wins (v0.29):**
- [ ] R2.1 check pre-flight extension (new doc: `m-motoko-ext-check-preflight.md`) — smallest, highest-leverage
- [ ] R3.1 prompt-size A/B on the rig (extends [m-eval-rig-reliability.md](m-eval-rig-reliability.md) tooling)
- [ ] R1.1 footgun coverage table + CI fixtures (new doc: `m-diagnostic-coverage.md`)

**Phase 2 — structural (v0.30–v0.31):**
- [ ] R1.2 prompt-line deletion pass gated on diagnostics landing
- [ ] R4 parser footgun docs (M-DX-MATCH-HOF first; Conflict Surface mandatory)
- [ ] R5 run result envelope + ast-query (pairs with [m-motoko-editdecl-astedit.md](m-motoko-editdecl-astedit.md))
- [ ] R6 orchestration benchmarks into core rotation

**Phase 3 — moats (v0.32+):**
- [ ] R2.2 contract-first grading tier
- [ ] R7 regex + URL parse builtins
- [ ] R8 trace exporter (internal first)

### Files to Modify/Create

None directly. Each routed doc carries its own file plan. PROGRAM.md §7 gets a two-line
pointer to this review after the Design Freeze items are decided.

## Examples

### Example: the same footgun, three eras

**Era 1 — prompt-time teaching (today, paid every run):**
> 2,518-line prompt including: *"Do not use ++ on strings; use ${} interpolation…"* (+ ~20 more warnings)

**Era 2 — error-time teaching (R1, paid only on mistake):**
```
$ ailang check solution.ail --format=agent
solution.ail:2:41 TC_XXX: `++` is for lists only → use "${a}${b}", concat([...]), or join(sep, ...)
```
(prompt line deleted; mid-tier model self-corrects in one cheap turn — R2 pre-flight makes the turn milliseconds)

**Era 3 — prior-time teaching (R8):**
The (error → fix) pair above, harvested from thousands of real sessions and verified by
replay, ships in training corpora. The next model generation doesn't make the mistake at all.

## Success Criteria

- [ ] Design Freeze items resolved by Mark; PROGRAM.md §7 updated
- [ ] Phase 1 docs created and sprint-planned
- [ ] v0.30 baseline reports cost-per-success ratio on the dashboard headline
- [ ] First footgun class retired end-to-end (diagnostic shipped + CI fixture + prompt lines deleted + rig re-measure shows the friction entry vanished)

## Testing Strategy

Not applicable at review level; each routed doc defines its own. The program-level test is
the PROGRAM.md loop itself: rig re-measurement after each routed fix (targeted friction
entry vanishes, pass/cost metrics move as predicted).

## Deferred Decisions

- Prompt compression method (hand-curated vs generated-from-diagnostics) — agent may choose during R3.1, data decides
- Regex engine (RE2 port vs Go `regexp` binding) — agent may choose in the R7 doc, subject to the linear-time mandate
- `ast-query` CLI shape (subcommand vs LSP-over-stdio JSON mode) — agent may choose in the R5 doc
- Which orchestration benchmark becomes the flagship docs example — agent proposes, Mark picks

## Non-Goals

- **Python-lookalike syntax sugar** — the explicitness *is* why frontier models score higher on AILANG; sanding it off destroys the differentiator (and violates the spirit of A7/A8)
- **Ecosystem breadth parity** (package-count race with Python/npm) — unwinnable, and irrelevant to the orchestration vertical
- **Motoko core changes** — everything here routes as extension/AILANG-fix per PROGRAM.md; the core stays frozen
- **Chasing Python's cost floor on trivial scripts** — a 3-line Python script will always be cheaper; the target is lowest cost per *verified-correct* program, where verification is included
- **Zero-shot pass-rate maximalism** — saturating at the frontier already; optimizing it further buys nothing the loop doesn't buy cheaper

## Timeline

Strategy reviews don't get week plans; the routed docs do. Sequencing intent: Phase 1 inside
v0.29's cycle (R2.1 is days, not weeks), Phase 2 across v0.30–v0.31, Phase 3 opportunistic.

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Prompt deletion regresses mid-tier pass rate | High | R1 gate: delete lines only after the replacement diagnostic ships + rig A/B confirms no loss |
| Cost KPI incentivizes gaming (shorter but wrong programs) | Med | Denominator is *verified* success (contracts/goldens), not self-reported completion |
| Orchestration-vertical pivot narrows perceived audience | Med | It's positioning + benchmark weight, not feature removal; general-purpose capability keeps shipping |
| Trace dataset publication leaks user/project data | High | Design Freeze gate; internal use first; export only benchmark-rig sessions initially |
| R4 parser fixes regress existing programs | High | Conflict Surface section mandatory per skill rules; fixtures before code |
| This doc rots like all strategy docs | Med | Single owner metric (cost-per-success on dashboard) keeps it falsifiable; revisit at each baseline |

## Verification Log (per design-doc hard gate)

| Claim | Method | Result |
|---|---|---|
| `++` on strings rejected with fix-suggestion | `ailang check` live repro, v0.28.0-9-g21e1251 | Confirmed; error text quoted in R1 |
| `ailang run -json` exists but covers errors only | `ailang run --help` | Confirmed: "Output errors in structured JSON format"; no success envelope |
| `ailang check --json` / `--format=agent` exist | `ailang check --help` | Confirmed |
| No regex support | `ailang builtins list` grep: 0 hits; no `std/regex` | Confirmed absent |
| Base64 support exists | builtins list: `_bytes_from_base64`, `_bytes_to_base64`, `_bytes_from_base64url` | Claim of gap REJECTED — exists |
| Datetime string parsing exists | builtins list: `_dt_parseISODate`, `_dt_parseRFC3339` | Claim of gap REJECTED — exists |
| URL encode exists, parse absent | builtins list: `_net_url_encode`, `_net_url_encode_form` only | Confirmed: no parse/split |
| Eval numbers | [docs/static/benchmarks/latest.json](../../docs/static/benchmarks/latest.json) (2026-06-21), `eval_results/baselines/v0.20.0/` | Cited with dates in table |
| Footgun list | [docs/LIMITATIONS.md](../../docs/LIMITATIONS.md), [prompts/v0.16.2.md](../../prompts/v0.16.2.md) | Cited; not re-verified individually — each R4 doc must verify its own target |

## Related Documents

**This review routes into / cross-references (checked for overlap, none duplicated):**
- [PROGRAM.md](../PROGRAM.md) — north star; this review adopts its routing rule wholesale
- [motoko-mission.md](../motoko-mission.md) — R2 sequences its verifier lever first
- [m-agent-ergonomics.md](m-agent-ergonomics.md) — supplies friction data consumed by R1
- [m-ailang-semantic-context.md](m-ailang-semantic-context.md) — context-side half of R3
- [m-motoko-editdecl-astedit.md](m-motoko-editdecl-astedit.md) — write-side half of R5
- [v0_22_0/m-three-camps-language-survey.md](v0_22_0/m-three-camps-language-survey.md) — evidentiary basis of R6
- [m-eval-frontier-tier.md](m-eval-frontier-tier.md) — benchmark-tier machinery R6 rides on

**Neural search results at creation (all < 0.45 — no duplicate-gate hits):**
- design_docs/implemented/v0_5_0/M-GAME-E2-ai-effect.md (0.31)
- design_docs/planned/m-eval-frontier-tier.md (0.34)

## References

- [Design Axioms](/docs/references/axioms) — the 12 principles scored above
- v0.25.0 eval baseline, 2026-06-21 — the empirical backbone of this review
- v0.20.0 baseline error-category breakdown — `eval_results/baselines/v0.20.0/`
- Nov 2025 gap analysis (66% of failures were teaching gaps) — `design_docs/archive/v0_4_6_implementation-gaps-analysis.md`

## Future Work

- Re-run this review at each major baseline (v0.30, v0.32) against the two headline metrics; if cost-per-success isn't converging toward ≤3×/≤1.5×, the strategy — not just the execution — is wrong and should be revisited.
- R8's endpoint suggests an eventual "prompt-zero" experiment: a model fine-tuned on the trace corpus attempting benchmarks with *no* teaching prompt.

---

**Document created**: 2026-07-08
**Last updated**: 2026-07-08
