# M-AILANG-SEMANTIC-CONTEXT: Typed, Semantic Context Minimization for the Agent Loop

**Status**: Planned (near-term = match pi, motoko-side; routes = AILANG-native, this repo)
**Target**: v0.26.x (near-term) / v1.x (AILANG-native routes)
**Mission item**: motoko-mission residual — *convergence efficiency* (pi ~5 turns vs motoko 15–50 on the same model)
**Estimated**: near-term ~1 day (motoko config/TS, fork PR + A/B); routes = multi-cycle, scoped per route below

## Progress (2026-06-20)

- **Observability LANDED.** Compaction telemetry instrument complete: A1 (motoko emits
  `compaction_structural`, fork `6bd9fa1`, type-checks; dist rebuild + PR pending rig), A2 (harness
  captures `compaction_count`/`first_compaction_step`/`compaction_level_max` into result JSON, this
  repo), A3 (`compaction_rate.sh` / `eval_compaction_rate.py` fire-rate report). Compaction is no
  longer invisible.
- **R1 LANDED (renderer).** `ailang check --format=agent` ships the compact one-line diagnostic
  format; `--json` unchanged. **R1b follow-up:** type-checker errors are not yet structured
  `ailerrors.Report` values, so the renderer distills the embedded `at file:line:col:` location
  heuristically — Report-ifying them gives native Code/Span/Fix.Suggestion (real fix hints, no regex).
- **Pending rig window:** A1 dist rebuild + draft PR; near-term pi-match (#1–#3); first
  compaction-fire-rate measurement on a fresh run.

## Problem (evidence-backed, 2026-06-19/20)

On local qwen3.6, after the truncation fix closed the disengagement gap, motoko reaches **statistical parity with pi on pass rate** (88.9% vs 90.4%) but is **3–10× less convergence-efficient**: pi solves in a **median of 5 turns** (562 runs; only 2 exceed 50), while motoko takes **15–50** — passing runs at 11–42 turns, failing runs hitting the 50-step `step budget exhausted`.

Reading both loops' source + a real motoko transcript (`csv_to_json_converter`, 42 turns) shows the mechanism is **context hygiene, not a verifier or step-cap**:

- Motoko's tool results are **full stdout/stderr verbatim, no line cap** (`tool_dispatch_adapter.ail:107`); pi **truncates to last ~30 lines / 128KB** (`bash.js:247`).
- Motoko runs **automatic compaction at 70% / 85% / 95%** context usage, eliding old `tool_result` content to ~80 chars + `[elided N chars]` (`compaction.ail:127`); pi does **no compaction by default**.
- Motoko's `WriteFile` result returns **sha256 + diff, no content echo** (`tool_dispatch_adapter.ail:85`).

**Vicious cycle:** verbose tool results fill context fast → 70% compaction elides the model's *own* earlier writes/reads → the model loses its working memory → re-reads files it just wrote and rewrites whole files → more turns → more context → more compaction. The transcript shows exactly this: `WriteFile(full) → Bash → WriteFile(full again) → EditFile(tiny) → ReadFile(its own write) → …` ×41. pi keeps a lean, fully-retained context and **builds forward**.

> **Pending cheap-confirm (Gate 2):** a `wire_diag.sh` capture on a thrash benchmark to verify `[elided …]` markers appear in the per-turn request *just before* each re-read. The source + transcript evidence is strong; this nails causation before the threshold A/B.

## Observability first: compaction telemetry (the leading indicator)

We have been *inferring* compaction because it is currently **invisible to our metrics**, and the durable fix is to measure it. Two gaps:
- The structural elision that drives the thrash — `compact_step` (`compaction.ail:127`) — is a `pure` function that **emits no event**; it fires silently at 70/85/95% usage.
- The events motoko *does* emit (`compaction_extension`, `compaction_exhausted`, `agent_loop_v2.ail:925/942`) are **dropped by the harness parser** — `internal/executor/motoko/parser.go:302` has no `compaction_*` case.

**Three-part instrument (do this before any threshold change):**
1. **Emit structural compaction (motoko, fork).** Have `compact_step` *return* its compaction metadata (level, bytes elided, turns affected) and let the caller — `agent_loop_v2` (which holds `session_id`) — emit a `compaction_structural` event. Keeps `compact_step` pure; the effect stays in the loop.
2. **Capture it (AILANG, this repo).** Add `compaction_*` cases to `parser.go` and surface per-run fields in the result JSON: `compaction_count`, `first_compaction_step`, `compaction_level_max`. (Same pattern as the existing `agent_turns`/`finish_reason` capture.)
3. **Aggregate (this repo).** A `segment.sh`-style **compaction-fire-rate** report per (harness, benchmark) — expect it to correlate with turns-to-success and inversely with pass rate.

**Why this is the right metric:** it is a *leading indicator* (compaction fires before the thrash inflates turns and before pass-rate degrades), and it is the **A/B success criterion** for the near-term fixes — pi's effective structural-compaction rate is ~0, so "match pi" is operationally "drive motoko's compaction-fire-rate toward 0 on the core tier." Per mission discipline (observe → diff → build), this telemetry is the cheap, continuous observe step that the one-off wire capture only samples.

## Near-term: match pi (motoko-side, draft PR to the fork)

Three small, independently A/B-able changes. Each "surfaces only what's needed" the way pi does — *syntactic* token reduction:

1. **Truncate tool-result output** to last ~30 lines / N KB, with a `[full output: <path>]` pointer (mirror `bash.js` `formatOutput`). Slows context fill → delays/avoids compaction. *Lowest risk, highest leverage.*
2. **Raise or disable compaction for short tasks.** pi proves these tasks never need it (5 turns). Lift the 70% trigger substantially (e.g. 90%), or gate compaction off below a turn/size floor. *Validate it doesn't regress genuinely long tasks.*
3. **Echo a concise write confirmation** in the `WriteFile` result (path + final line count + a one-line structural summary) so the model trusts its write and stops re-reading. *Cheap, removes a whole re-read class.*

**Routing:** motoko_agent config (`.motoko/config/ollama/config.json`) + `tool_dispatch_adapter.ail` — draft PR to `arniwesth/motoko_agent` via the `sunholo-voight-kampff` fork, verified locally, A/B by **turns-to-success** (not just pass rate) on the core tier.

## The AILANG-native opportunity (where we go *further* than pi)

pi and tools like [rtk](https://github.com/rtk-ai/rtk) do **syntactic** context reduction — truncate token-heavy text outputs. AILANG can do **semantic** context *minimization*, because it is a typed, effect-tracked language that emits structured diagnostics and deterministic execution traces. The agent never needs the raw bytes — it needs the *meaning*. This is AILANG's core design principle ("structured execution traces", "semantic transparency", "designed for AI code synthesis") applied to the agent loop itself.

The strategic framing: build these as **`std/ai` + compiler/runtime capabilities in this repo**, so *any* AILANG agent — motoko, the `std/ai.runTools` loop (`_ai_step`), future harnesses — gets minimal typed context for free. That makes "the best AILANG coding harness" a property of the *language*, not one harness's prompt tuning, and is a differentiator no generic harness can copy.

### Routes (a menu — pick per cycle, each independently shippable)

**R1 — Distilled diagnostics.** `ailang check`/`run` already produce typed errors. Surface a compact *structured* error instead of the full compiler dump: code + location + the specific type/instance + a one-line fix hint. E.g. replace a multi-line stderr with
`No instance for Num[string] at solution.ail:25:33 — arithmetic on a string; use stringToInt or ++ for concat.`
Smaller **and** more actionable. This alone likely closes the `Num[string]` residual class. *Routing: AILANG (`internal/types` error rendering + an `--format=agent` mode), this repo.*

**R2 — Type-directed context surfacing.** When the model edits a function, surface only the relevant signatures and effect rows it touches (from `internal/iface`), not whole files. AILANG's type environment *is* the relevant context; "surface only what's needed" becomes a typed projection, not a heuristic. *Routing: AILANG (`iface` projection) + a motoko tool that calls it.*

**R3 — Semantic / AST diffs.** `WriteFile`/`EditFile` results return a *semantic* diff (which decls/signatures/effects changed) instead of a text diff — smaller and more meaningful, and immune to compaction eliding "what I wrote". *Routing: AILANG (AST diff over `internal/ast`), consumed by motoko's tool result.*

**R4 — Effect-row deltas.** Effects are explicit in signatures, so a change's effect impact (`main now requires {IO, FS}`) is a compact, high-signal token. The model reasons about effect correctness without re-reading the file. *Routing: AILANG (effect-row diff), this repo.*

**R5 — Execution-trace distillation.** AILANG's deterministic structured traces (the `AILANG_TRACE` tiers) can be distilled to *the step that diverged from expected*, not the full trace — pairing a failing run with the minimal causal slice. Leverages determinism + semantic transparency. *Routing: AILANG (`internal` trace projection), this repo.*

**R6 — Typed tool results + a context-budget projection layer.** Represent tool results as structured typed values the loop can *project* (surface only needed fields) rather than JSON-stringifying everything — `_ai_step` already takes structured messages. Then: given a context budget + structured session state, compute the *minimal sufficient projection* (relevant sigs + distilled errors + last-N semantic diffs). Deterministic and typed → unit-testable, unlike heuristic compaction. The projection's "what's relevant" ranking can be **embedding-powered (R7)** rather than purely type-directed. *Routing: AILANG (`std/ai` context library), this repo — the capstone.*

**R7 — Brain-powered semantic context engine (reuse the existing SEM-CACHE).** AILANG already ships an embedding/SimHash semantic cache — "The Brain" (M-BRAIN, `internal/effects/brain.go`, `std/sem`, `SharedMem`), explicitly designed as a *"Loop Accelerator"* for "avoiding redundant work in agent loops," with a local-capable embedder (EmbeddingGemma/nomic via ollama). microRAG already points this machinery at *external* knowledge (syntax/builtins) and moved the needle a little. R7 points the *same* engine at the agent's *own* conversation. Three sub-applications, cheapest first:
- **(a) Semantic dedup of re-reads (the thrash killer).** The observed thrash is literally re-reading the same file / rewriting whole files. SimHash dedup is purpose-built for "is this ≈ something I've already seen" — collapse near-duplicate tool results to one. Fast, local, approximate ("useful, sometimes wrong but fast" — report-only by default, inspectable). Highest signal-per-effort; directly drops compaction-fire-rate.
- **(b) Relevance-ranked context selection (the principled compaction).** Today's compaction elides by *recency* (keep last N), which erases the model's own still-relevant writes. Instead: embed each message + the current frontier (task + current error), keep the most *relevant* regardless of age. Conversational RAG over the loop's own history. Hybrid with always-keep-last-N for safety. This is the embedding upgrade to R6's ranking.
- **(c) Memoize tool results / sub-answers** (the cache's designed purpose): if the model re-issues a semantically similar query/tool call, serve the cached result instead of re-running and re-growing context.

*Why low-risk:* this is **proven, cross-cutting infra, not a science project** — the same machinery is already in production across the message inbox (SimHash dedup), coordinator triage (cosine clustering @0.75), the Brain/microRAG, and **`docsearch`, which already implements the exact two-stage pipeline R7 needs: "SimHash shortlist → lazy embedding → cosine similarity"** (`internal/docsearch/search.go`). `_simhash` is a first-class builtin "for near-duplicate detection". So R7a is *reusing a battle-tested pipeline*, and the re-read thrash is precisely the near-duplicate problem it's built for.

*Fit / caveats:* reuses existing infra (no new dependency), embeddings run locally — but per-message embedding adds latency (do the cheap SimHash dedup first; make neural selection opt-in), and it's approximate so pair with recency fallback + inspection. Calibrate expectations: microRAG was a *small* lift, so R7 is a refinement of compaction, not a silver bullet — but (a) is targeted enough to be a clean, measurable win once the telemetry shows where compaction fires. *Routing: AILANG (`std/sem`/`SharedMem` context layer) this repo; motoko's loop calls it (fork) — a language-level capability any AILANG agent inherits.*

### Why this is the right shape for AILANG

- **Deterministic & typed** → context projection is testable and reproducible (heuristic text-compaction is neither).
- **Semantic, not syntactic** → fewer tokens *and* higher signal; the agent gets meaning, not bytes.
- **Language-level, not harness-level** → benefits motoko, `std/ai.runTools`, and any future agent; compounds with every model.
- **Differentiator** → a generic harness (pi/opencode) cannot do R1–R6 because it doesn't have AILANG's type/effect/trace structure to project from.

## Recommended sequencing

1. **Observe first:** compaction telemetry (3-part instrument above) + a one-off `wire_diag` elision capture to confirm causation. The telemetry is the durable leading indicator and the A/B success metric.
2. **Near-term pi-match** (#1–#3) — fork PR, A/B by **turns-to-success and compaction-fire-rate** (target → ~0, pi's rate). Gets motoko to pi-level efficiency.
3. **R1** (distilled diagnostics) — smallest AILANG-native win, closes a concrete residual, sets the `--format=agent` precedent.
4. **R3/R4** (semantic + effect diffs) and **R6** (the typed projection layer) as the capstone, once R1's pattern is proven.

## Risks / open questions

- **Compaction threshold A/B** must validate on genuinely long tasks, not just the 5-turn core benches — raising it could hurt context-limited cases. Gate behind a turn/size floor.
- **Truncation hiding signal:** pi-style last-30-lines can drop a relevant earlier error; pair with R1 so the *distilled* error is always retained even when raw output is truncated.
- **Upstream coordination:** R2/R3/R6 need an AILANG `--format=agent` / projection API that motoko consumes — defines a stable contract between this repo and the fork.
- **Measurement:** the right metric is **turns-to-success** (convergence efficiency), not pass rate — parity on pass rate already holds.

## References

- Analysis: `design_docs/motoko-harness-analysis-log.md` (2026-06-19/20 convergence-efficiency entries) + `design_docs/motoko-mission.md`.
- Source: pi `bash.js:247` (truncation), `agent-loop.js` (no default compaction); motoko `compaction.ail:127`, `tool_dispatch_adapter.ail:85,107`.
- Prior art (syntactic token reduction): [rtk-ai/rtk](https://github.com/rtk-ai/rtk).
- AILANG agent primitives: `std/ai` `_ai_step` / `runTools` (`examples/runnable/ai_tool_loop.ail`).
