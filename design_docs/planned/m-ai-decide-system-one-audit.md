# Audit: where we make structured-data decisions via AI calls (2026-09-18)

**Asked by**: Mark, after M-AI-DECIDE-SYSTEM-ONE Phase 1 — "where do we make structured data decisions via AI calls now, in packages or ailang or workflows; can the package drop in and replace them?"
**Method**: grep for structured-output requests (`ResponseSchema`/`ResponseFormat: "json"`/`json_schema`/`callJson*`) and for code that parses a model's JSON/fenced verdict, across the ailang repo (Go + `.ail`), `ailang-packages`, `email-parse`, `daneel`, and the skills. Each site was read, not just matched.
**Companion**: [m-ai-decide-system-one.md](../implemented/v0_40_1/m-ai-decide-system-one.md) · [shadow report](../implemented/v0_40_1/m-ai-decide-system-one-shadow-report.md) · package `sunholo/decisions` 0.1.2 (`ailang pkg-docs sunholo/decisions`).

## The rule for reading this table

A site is a **drop-in** only if every field its consumer *branches on* is a category, a yes/no, or an ordered level. A System One model generates no text: any `reasoning`/`objection`/`blockers` string that downstream code *reads* keeps that site on an LLM (or splits it: decisions to `sunholo/decisions`, generation stays on `std/ai`).

## Inventory

| # | Site | Today | Fields branched on | Verdict | Why / blockers |
|---|---|---|---|---|---|
| 1 | **`internal/feedbackgate`** (Go) — stage 3 of `Decide`, wired into the coordinator daemon | `claude-haiku-4-5`, `ResponseFormat: "json"`, strict struct; parse failure → fail closed (file) | `is_genuine_feedback` (bool), `is_prompt_injection` (bool), `best_category` (enum), `estimated_dispatch_value` (level); `reasoning` is audit-only | **Drop-in — first candidate.** Noul / Noul / Choice / Score, exactly; the migration example in AGENT.md is this classifier | Go caller → via `internal/embed` (policy-in-AILANG precedent, `budget_checker.ail`) or Phase 2. New vendor for feedback text (today Anthropic) → a ruling. Already has an audit table (`feedback_gate_audit.go`) to shadow against |
| 2 | **eparse `packages/eparse/triage.ail`** (AILANG, email-parse repo) | `callJson(prompt, categorySchema())` on `gemini-2-5-flash` → one of 5 categories; a **second model pass** writes `triage_2`; `needs_human = triage_2 != triage` | category (enum); disagreement (derived) | **Drop-in — second candidate, and the best fit.** One `Choice` call replaces two LLM calls; the `probabilities` spread *is* the disagreement signal, calibrated instead of a 2-model coin flip | The file carries IFC labels (`Declassify` effect; email body is untrusted) — confirm `std/net.httpRequest` accepts the same declassified string path before switching. Vendor ruling (today Gemini) |
| 3 | **Daneel `tools/daneel_decide.ail`** | Calls **no model**; reads eparse's stored labels as data; Z3-proven decision | — | **Not a site — and a constraint.** Daneel's stated axiom: escalation is never model-supplied confidence ("precisely what an injected model controls"). Jev's `confidence` must NOT become a Daneel escalation input. If #2 switches, Daneel keeps reading argmax labels as data; the distribution is banked, not acted on | Daneel's Decision 13 boundary (words leave for Vertex) would need extending to TypeSafe |
| 4 | **`internal/mission/quorum`** (design-review / design-quorum) | Reviewer returns `{verdict, strongest_objection, catch, proposed_fix}` | `verdict` — but the *objection text* is the product | **Keep on LLM.** Augment only: a calibrated `Noul("this design will survive independent review")` as a cheap pre-screen or tie-breaker, banked not acted | Generation-heavy by design |
| 5 | **`internal/eval_harness/gemini_evaluator_bridge.go`** (sprint-evaluator) | Fenced JSON `{score 0-100, pass, blockers[]}` | `score`, `pass` — and `blockers` (text) | **Hybrid.** `score` is a `Score` question over a diff bundle; `blockers` is generation. Candidate for a **second opinion on `pass`** (cheap consistency check, shadow first), not a replacement | 32k state ceiling vs diff bundles; needs the same size-bounding the bridge already does |
| 6 | **`internal/coordinator/retry_chain.go ClassifyFailure`** | Substring heuristic (`"429"`, `"returned zero bytes"`…), **no model** | transport vs model failure | **Upgrade candidate, low priority.** `Noul("this failure is transport-class")` gated, heuristic as fallback — only worth it if misclassification is measured first (memory: the `"429"` substring once retried into a spent bucket) | Measure before building |
| 7 | **`error_category` `api_error` catch-all** (`internal/eval_harness/metrics.go:214`) | Heuristic; `api_error` = "cause unknown" | category | **Shadow sub-classification** (design doc Future Work): one `Choice` per banked `api_error` row asking the cause; banked, then mined | No ground truth yet → hand labels first |
| 8 | `motoko-ext-compaction-ai`, `tools/motoko/r8_headroom_band.ail`, `ai_compat`, `ailang-docs` prompts | `step`/`call` for **summarisation / probing** | — | Not decisions. Out of scope | — |
| 9 | **Skills that decide in prose** — mission-control §4 routing, `ailang-core-triage` recommendation, `github-issue-triage`, `eval-analyzer`/`eval-gap-finder` priority, `sprint-evaluator` rubric | A frontier turn reads, judges, writes markdown | lane / priority / recommendation, embedded in prose | **Pre-classify, don't replace.** The shadow lane-router is exactly this for §4: 14/20 vs declared lane, and 4 of the 6 misses were our labels. Acting waits on the label fix + re-run | Skills have no structured output contract to swap; the win is a typed pre-decision the skill reads |

**No `[[ai_provider]]`, workflow (`Workflow` tool) or Daneel `ext/` package makes a structured AI decision today** — searched, none found (the Daneel extensions claim by tag/subject match, proven, and hand the rest to eparse's labels).

## Can it drop in? (the docs question)

- **For a new AILANG consumer: yes.** `ailang pkg-docs sunholo/decisions` serves the AGENT.md (protocol, the complete gated example, the four consumer rules, limitations). The site page under `/docs/packages/sunholo/decisions` is **generated from the registry at the next docs deploy** (`docs/scripts/sync-registry.sh`) — no hand-written page is needed; the design doc's Files entry for one was wrong.
- **For migrating an existing `callJson` classifier: yes as of 0.1.2**, which added the migration section: schema-property → `Question` mapping (enum→Choice, bool→Noul, integer range→Score, free text→*not this tool*), what is lost, a complete example that is the feedback gate's shape (compiled against the published package this session), and the two Go routes.
- **For Go callers: not directly.** Route 1 is `internal/embed` (the repo's policy-in-AILANG pattern); route 2 is Phase 2 (`std/ai.decide` + `DecisionProvider`, which also brings budget/trace accounting). AGENT.md says: do not hand-roll the HTTP call in Go meanwhile — that is a second implementation of one seam.
- **What no doc can settle: the data boundary.** Every site above sends content to a vendor under some ruling (Anthropic for feedback, Gemini for email, Vertex for Daneel). Switching a site to TypeSafe-via-OpenRouter extends that ruling; AGENT.md now says so in one line.

## Recommended order

1. **Feedback gate shadow** (#1): run `sunholo/decisions` beside Haiku on live traffic via an embed shim, log both into the existing audit table, compare on the next 200 messages. Real traffic, an audit path already built, and the field mapping is exact.
2. **eparse triage shadow** (#2): one `Choice` beside the two-model quorum on the next batch; compare Jev's distribution spread against `needs_human`. If it predicts the quorum disagreement, one call replaces two.
3. **Evaluator `pass` second opinion** (#5): banked only.
4. Fix the four mislabeled lanes / PROGRAM §4 wording, re-run the lane shadow (from the Phase 1 report) — then choose the first *acting* consumer from 1–3.

Phase 2 (`std/ai.decide`) is justified by items 1 and 5 being Go: they are the sites where "outside the AI budget/trace" stops being tolerable.
