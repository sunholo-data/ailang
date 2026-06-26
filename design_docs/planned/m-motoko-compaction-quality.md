# M-MOTOKO-COMPACTION-QUALITY — Make AI compaction summaries useful, not just small

**Status:** Planned (2026-06-25)
**Owner:** motoko mission
**Component:** `compaction_ai` extension (`sunholo-data/ailang-packages/packages/motoko-ext-compaction-ai`)
**Related:** [[m-motoko-self-improvement-loop]], motoko-harness-analysis-log.md, task #8 (keep current file verbatim), PR arniwesth/motoko_agent#75 (draft)

## TL;DR

Compaction now *fires* at scale (validated: 26 AI-summarizations on the docx run, context held 237K→6.5K). But the summaries are **too lossy to be useful** — and sometimes actively misleading. The fix is a higher-quality, structured, task-pinned summary that the model can actually resume from. It lives almost entirely in the **`compaction_ai` extension** (a repo we own), not motoko core or AILANG.

## Problem — "compaction fires" ≠ "compaction works"

On the docx re-run (hex fix active), the AI summarizer fired 26 times, each catching usage at ~51–65% and compressing 160–213 old turns back to ~0–3%. Context stayed bounded. **But the run still ground to the step budget without converging** — and the reconstructed summaries show why.

The current prompt asks for **"2–3 sentences."** A real sample, produced by running the actual summarizer (qwen3.6) over 22 turns of this run:

> *"The assistant iteratively corrected syntax errors in `docx_parser.ail`… the file was rewritten to contain a clean, syntactically correct implementation. **The task is now at a finalized state with the parser module ready for compilation**…"*

This is three problems at once:
1. **Misleading** — claims the task is "finalized / ready for compilation" when the run had finished nothing. A model resuming from this thinks it's basically done.
2. **No working set** — loses which of the 13 functions are implemented, which remain, and what the actual current compile errors are.
3. **Decay** — each of the 26 rounds re-summarizes the *previous* summary (prose→prose), so detail erodes every cycle.

### What the model actually keeps after compaction (current format)

```
[system message]      ← AILANG reference. Pinned by host, never elided. OK.
["[CONTEXT SUMMARY] " + 2-3 sentence prose]   ← ALL old turns, incl. the original task
[last keep_recent (=6) messages, verbatim]
```

The **original task is NOT pinned** — it is a turn-0 user message that gets folded into the lossy summary (only the AILANG *reference* survives as the system message). The summary text is **not logged anywhere**, so its quality was invisible until reconstructed by hand.

## Current architecture (where each piece is today)

| Piece | Location |
|---|---|
| Summary prompt, `summary_msg`, `split_msgs`, `compact_with_ai` | `compaction_ai` extension — `ailang-packages/packages/motoko-ext-compaction-ai/compaction_ai.ail` |
| `keep_recent`, `threshold_pct`, summarizer `model` | per-profile `.motoko/config/<profile>/compaction_ai.json` |
| Structural fallback (`compact_step_actual` — drop old tool results) | motoko `src/core/compaction.ail` |
| Host wiring / `dispatch_pre_step` / `ctx.task` + `ctx.context_limit` | motoko `src/core/agent_loop_v2.ail` (passes `ctx.task`) |

The extension already receives `ctx.task`, `ctx.context_limit`, and the full `msgs`, so it has everything needed for the fix.

## Architecture decision — motoko owns *when*, extensions own *how*

The right separation of concerns (and it fixes the trigger-misalignment bug we observed):

Today the decision to compact is **split and duplicated** —
- the `compaction_ai` extension self-decides (`if usage_percent < threshold then PassThrough`), **and**
- the host's `compact_step_actual` independently decides via its own actual-token tiers (60/75/85%).

These two triggers can disagree, which is exactly what happened on the docx runs: the host's structural elision fired first and pre-empted the AI summarizer, so the smart layer never ran (0 AI compactions). **Two deciders, one decision = bug.**

**Target architecture:**
- **Motoko (host) owns the policy — "do I need compaction, and how urgently?"** Single source of truth, computed from the provider's **actual** `input_tokens` vs `context_limit`. It emits a directive: `None | Gentle(target) | Aggressive(target) | Emergency(target)`.
- **Extensions own the mechanism — "how do I compact?"** The host dispatches the directive to the configured compaction-strategy extension(s); each is *told* to compact toward a target and never self-decides. `compaction_ai` becomes a pure strategy.
- **Config selects + orders strategies** per profile (`compaction.strategy = "ai-structured"`, or an ordered pipeline `"structural,ai-structured"`).

This inverts the dependency cleanly — policy in the host, mechanism in pluggable extensions. The structured-summary work below is then simply the first (and best) *strategy* under this framework.

## How other harnesses handle this

| Harness | Summary format | Pinned / never summarized | Decay handling | Trigger | Recent verbatim |
|---|---|---|---|---|---|
| **Claude Code** | **9-section structured** (intent / concepts / files+code / errors+fixes / problem-solving / **all user messages verbatim** / pending tasks / current work / next step) | system+CLAUDE.md, all user messages, recent tool calls; **re-reads up to 5 edited files (≤50k tok) post-compaction** | re-summarizes prior summary; can degrade | effective_window − 13k tok | threshold-bounded |
| **pi** (mariozechner) | **running structured state**, 9 headings (Goal/Constraints/Progress[Done·InProgress·Blocked]/Decisions/Next/Critical) | recent suffix verbatim; **file read/written/edited sets tracked deterministically from tool calls**, chained across compactions | **UPDATE-merge** prompt (feeds `<previous-summary>`, updates in place) — no prose decay | `tokens > window − reserve(16384)` | keepRecent 20k tok; never cuts inside a tool result |
| **Codex CLI** | 4-part handoff | **all user messages verbatim**; excludes prior summaries (no stacking) | replace, don't stack | before user turn, ≤90% | ~20k recent |
| **qwen-code** | single LLM summary, keep last 30% / summarize first 70% | recent ~30% | fresh each time (lossy) | 70% of window | last 30% |
| **Qwen-Agent** | **no LLM summary by default** — hierarchical drop/fold (S1–S5) | query + final response + most recent step | removal/folding | at max_input_tokens | most recent step |
| **OpenCode** | 6-heading summary | **last 40k tok + 2 user turns**; "skill" outputs never pruned | fresh; tool outputs hidden by timestamp-marking | window − output, only if frees >20k | last 40k + 2 turns |
| **Cursor** | LLM summary (sections unpublished) | summary **+ full history exposed as a searchable file** (recovery hatch) | fresh; recovers via history-file search | window fills | unpublished |
| **Aider** | `weak-model` summary past soft limit | `/add`-ed files; repo-map (separate) | re-summarizes | soft chat-history token limit | soft-limit driven |

### Best-practice synthesis (what the strongest converge on)
1. **Cascade — summarize LAST.** Clear bulky **stale tool outputs** → sliding-window trim → LLM summary only as last resort. The LLM call is the most expensive + lossiest step; gate it. (Claude Code's 3 layers, OpenCode prune-then-protect.) **This is exactly the host-owns-trigger / pluggable-strategy split — the urgency directive picks cheap-vs-thorough.**
2. **Structured running state > prose re-summarization.** Fixed-section template you **UPDATE in place** (Done→InProgress, drop resolved blockers), feeding the prior summary back in — avoids the prose→prose decay we measured. **pi** is the cleanest; **Claude Code's** 9 sections the richest.
3. **Pin the load-bearing set verbatim:** original intent, **all user messages** (course-corrections + security), recent N tokens, and **recently-edited file contents**.
4. **Track files/decisions deterministically from tool calls**, out-of-band of the LLM summary — so the summarizer can't "forget" which files are in play ("lost in the middle", Liu 2023).
- Plus: a **recovery hatch** (Cursor keeps full history searchable) and **auditability** (store summary as a typed entry; trigger as a clear `window − reserve` formula).

_Sources: Claude Code summarization prompt (Piebald-AI/claude-code-system-prompts), pi local source (`@mariozechner/pi-coding-agent/dist/core/compaction/`), Codex/OpenCode compaction write-ups (justin3go.com, codex.danielvaughan.com), qwen-code issues #1924/#2817, Qwen-Agent docs, Aider docs, "Lost in the Middle" (Liu 2023). Full list in the research transcript._

## Proposed design

Adopt the patterns the strongest harnesses converge on — **pi's running structured state + Claude Code's pin set + the cascade** — implemented as the `ai-structured` strategy:

1. **Pin the load-bearing set verbatim** (never summarized, always at the top): the **original task** (`ctx.task`); **all prior user messages** (course-corrections, constraints — not just turn-0); and the **most recent `keep_recent` turns** (raised from 6).
2. **Structured running state, UPDATE-merged (not re-summarized prose).** A fixed schema (pi's, adapted):
   ```
   ## Goal           <the task>
   ## Progress
     Done:        <files/functions completed + verified>
     In progress: <current focus>
     Blocked:     <what's stuck + why>
   ## Current errors <exact compile/test errors outstanding>
   ## Key decisions  <syntax facts learned, approaches chosen>
   ## Next action    <the single next step>
   ```
   Each round feeds the **previous summary back in** ("PRESERVE existing; move In-progress→Done when complete; drop resolved blockers") — a running state, not prose→prose decay. Explicit: **never claim completion**; report the real, incomplete state.
3. **Track files deterministically from tool calls** (pi's trick) — extract read/written/edited sets from `tool_calls` in `msgs` **in code, not via the LLM**, and carry them forward; the summarizer can't "forget" which files are in play. Keep the **latest WriteFile/EditFile body of each touched file verbatim** out of the summary (task #8) — the model never re-reads the file it's editing.
4. **Cascade — summarize last.** A cheap structural pass (drop stale tool-result *bodies*, cap huge `cat`/`ailang check` dumps) runs first; the LLM summary fires only if that's not enough. (This is the host-owns-trigger / pluggable-strategy split — see Architecture.)
5. **Log + recover.** Emit the generated summary as a first-class event so `ailang chains` shows it (auditability — it's invisible today). The full pre-compaction history is **already archived in chains** via `import-motoko` — that *is* Cursor's recovery hatch; we/the model can search back into it.

## Strategy space (the optimization surface — pluggable by config)

| Strategy | Mechanism | Best for |
|---|---|---|
| **structural elision** | drop old tool-result bodies; cap huge outputs | cheap first pass (the cascade's step 1) |
| **ai-structured** (this doc) | UPDATE-merged structured state, task+files pinned, current file verbatim | the working set on a long coding loop |
| **semantic dedup** | drop near-duplicate turns (re-reads/re-checks of the same file) | grind loops |
| **retrieval / RAG** | move old turns to a store, retrieve on demand | very long tasks needing specific recall |
| **compression** | `gzip`/token-pack rarely-needed spans, expand on reference | lossless-ish headroom |

Composition is the point: a cheap structural pass usually buys enough that the expensive `ai-structured` pass runs rarely — and the **host's urgency directive picks cheap-vs-thorough.**

## Placement / change surface (answers "motoko or extension or where?")

Per the Architecture — **policy in the host, mechanism in the extension:**

| Layer | Change | File(s) |
|---|---|---|
| **Host = trigger/policy** | make `compact_step_actual` the **single** decider (actual tokens, `window − reserve`); emit an urgency directive + token target; dispatch to the configured strategy instead of eliding inline; remove the extension's self-threshold | motoko `src/core/agent_loop_v2.ail` + `compaction.ail` |
| **Extension = strategy** | `ai-structured`: pin set, UPDATE-merge running state, deterministic file tracking, current-file verbatim, emit the summary event; *takes a directive, never self-decides* | `ailang-packages/packages/motoko-ext-compaction-ai/` (**we own it** → iterate + `ailang publish` freely) |
| **Config = selection** | `compaction.strategy` (+ ordering for the cascade), `keep_recent`, reserve target | `.motoko/config/*/compaction_ai.json` |

**NOT** arniwesth/motoko core beyond the trigger wiring; **NOT** AILANG core.

### Staging
- **Phase 1 (now, low-risk, extension-only):** upgrade the *current* extension — structured UPDATE-merge summary, pin task + user messages, deterministic file tracking, current-file verbatim, **log the summary**. Keeps the existing (imperfect) two-trigger wiring; validates the quality fix in isolation. Highest leverage per effort.
- **Phase 2 (the framework):** move the trigger fully into the host (single decider + directive), make strategies pure + pluggable + composable (the cascade), remove the misalignment.

## Rollout plan

1. Prototype the new prompt + structured summary + task-pin in the extension; **log the summary text** first (cheap, immediately auditable).
2. Cheap-confirm: replay the new summarizer over captured docx turns (like the sample above) and eyeball the structured output — does it preserve the working set + the real (incomplete) state?
3. Re-run docx (survivable / detached) with the new extension; import to `ailang chains`; read the now-logged summaries; check whether the model converges or at least stops losing state.
4. If good: `ailang publish` the bumped extension version; update the profile config.

## Risks / open questions

- **Summarizer capability:** qwen3.6 must reliably fill a structured schema without hallucinating "done." Mitigation: strict schema + the explicit "do not claim completion" instruction; the cheap-confirm replay gates this.
- **Current-file detection heuristic:** identifying "the latest WriteFile per file" from opaque `msgs` — needs the tool-call shape; doable since `msgs` carry `tool_calls`.
- **Token budget of the richer summary:** a structured summary + pinned task + current file is bigger than 2–3 sentences. That's the point, but it must still net a real reduction. The running-state approach keeps it bounded.
- **Cross-harness portability:** the extension is shared; the structured schema should be generic enough for non-coding tasks (or gated by task type).
