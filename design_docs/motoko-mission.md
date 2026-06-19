# MISSION: Make Motoko Competitive on Local AILANG

**Type:** Long-running mission (advanced in downtime, e.g. while evals run)
**North star:** the AILANG-native harness (motoko) should match or beat the generic
harnesses (pi 96%, opencode 79%) on local-AILANG synthesis. Today motoko = 26% AILANG.
See [[motoko-strategic-goal]].

## How the mission runs (each cycle)

A cycle picks up **one** item and runs the full record-keeping flow:

1. **Observe** — read the latest rotation/eval numbers (`os/latest.json`,
   `eval_results/rotation/os-rolling`) + the [analysis log](motoko-harness-analysis-log.md).
   Append a new analysis-log entry if there's new failure data.
2. **Pick** — take the top open item from the Backlog below (or a newly-found one).
3. **design-doc → sprint-plan → execute** — same flow as M-EMBED-TASK-PREFIX /
   M-EVAL-OS-LONGITUDINAL, so every change has a design doc + sprint record.
4. **Land** — per the routing rule below. Verify locally before landing.
5. **Record** — update the analysis log (prior-action status), tick the Backlog,
   re-measure on the rig next cycle.

## Routing rule (where changes land)

| change is in… | lands as |
|---|---|
| **AILANG** (`internal/…`, `cmd/…`, eval rig, `tools/…`) | **commit to `dev`** (this repo) |
| **motoko_agent** (`.ail` core, profiles, prompts, TS) | **PR** to `arniwesth/motoko_agent` (via our `sunholo-voight-kampff` fork) — verified working locally first |

## Reference harness

**pi** = `@mariozechner/pi-coding-agent` → `@mariozechner/pi-ai`. It's motoko's
inspiration and the 96% bar. Key learnings (mine the source under
`/opt/homebrew/lib/node_modules/@mariozechner/pi-*` and `internal/executor/pi/`):
- Drives ollama via **OpenAI-compat `/v1/chat/completions`** (no native ollama
  provider) — the reason its tool-calling is reliable on qwen.

## Backlog (prioritized — top = next)

0. **[AILANG dev — TOP, leading candidate] Read qwen's `reasoning` field (generic openai-compat fix).**
   Wire capture (HTTP-wire logger, `c1f87275e`) proved qwen3.6/ollama returns a `reasoning` field
   (10k+ chars) that AILANG's `ParseChatStepResponse` DROPS (reads only content+tool_calls). All 3
   reference harnesses read it (pi, qwen-code, Qwen-Agent); **Qwen-Agent#789 is our exact bug**, fixed
   with a 2-line GENERIC fallback (`reasoning_content` || `reasoning`) — NOT per-model. Fix:
   (a) capture a DISENGAGED benchmark on the wire to confirm the answer/tool-call is stuck in
   `reasoning` (causation not yet proven — a captured graph_bfs run ENGAGED+passed, so the rotation's
   always-disengage list may be stale); (b) add generic reasoning-field read in `internal/ai/openai`
   (+ openrouter); A/B by disengage-rate. Generic, dev-side, keeps core simple.
0b. **[eval rig — user priority] Add `qwen-code` (QwenLM/qwen-code) as an eval-suite harness arm.**
   qwen's own coding agent (OpenAI-compat/ollama, CLI like opencode/pi → fits the executor contract).
   A qwen-tuned reference on the SAME benchmarks — directly measures the motoko↔well-tuned-harness delta.

1. **[THE gap, precisely located 2026-06-18 — DISENGAGEMENT, needs GPU to fix].** Rotation-scale
   failure-mode segmentation (`tools/eval_failure_modes.py`): the motoko↔pi gap (+26pp) is
   **entirely disengagement** — motoko fails with ≤2 tool calls (prose / one inspect call) **29%**
   of runs vs pi's **3%**; grind-wrong (engaged-but-incorrect) is ~1% for both, NOT the gap.
   **7 always-disengage benchmarks** (3/3 fail, 0 tool calls): csv_to_json_converter,
   log_file_analyzer, graph_bfs, polymorphic_ord_defaulting, run_length_encode, symbolic_diff,
   config_file_parser. **Next (GPU):** request-dump the FIRST turn of those 7 to see WHY qwen emits
   0 tool calls (tool schema not seen? task framing? result format?), then a targeted fix A/B'd by
   the **disengage-rate delta** (not just pass rate). Prior fixes touched the symptom not the 29%:
   - *Ruled out — M-MOTOKO-PERSIST-NUDGE* (PR arniwesth/motoko_agent#47): forces continuation AFTER
     disengagement; +3/18 on a biased subset; pi has no such mechanism. Kept default-off; divergent.
   - *Ruled out — M-MOTOKO-AGENT-SYSTEM-PROMPT*: lean agentic system prompt, proper core-tier A/B =
     **+1/52 (null)**; the 6-flaky smoke (+14pp) did NOT generalize. Knob kept, not productionized.
   - *Ruled out — system-role delivery* (teaching → system slot): A/B −2/18.
2. **[AILANG, needs GPU] Temperature A/B** — `AILANG_OLLAMA_TEMPERATURE` knob landed (off by
   default). A/B unset vs 0.2/0.3 on the 6 flaky benchmarks; if it lifts engagement, make a low
   temperature the default; else investigate qwen thinking-mode. Lower priority than #1 (the
   request-diff showed pi runs the same 1.0 default, so temperature isn't the pi gap — but lower
   variance may still reduce motoko's flakiness).
3. **[AILANG] Convergence / robustness**: `finish_reason=tool_calls and no run_summary` +
   `step budget exhausted` tail. Lower priority — the lock + /v1 timeout removed the contention
   trigger; revisit if it recurs un-contended.

## Resolved / ruled out (this investigation arc)
- **System-role delivery** (M-MOTOKO-SYSTEM-ROLE, `7a0caf7a`): A/B off-vs-on (6 flaky ×3, 2026-06-18)
  = 10/18 → 8/18 (net −2, within noise) → **KEEP GATED, not default-on.** Retained as a lever: it
  lifts the FLOOR (config_file_parser/graph_bfs 0→1) and raises iteration depth (turns up across the
  board) — re-test combined with iteration-persistence (#1), not standalone. Clean injection
  mechanism shipped: motoko `--system-prompt` flag PR (arniwesth/motoko_agent#46, GPU-verified).
- **3-way capture** (motoko/pi/opencode, `f92df86b`): threading IDENTICAL across all three (chat
  history ruled out). Differentiator = TOOL SURFACE — opencode 33 tools (23 ailang-docs MCP) vs
  motoko 6 / pi 4 → keep motoko lean. Backs the "why motoko beats opencode" decision.
- **Prompt** is sub-dominant: agent-mode output-delivery override landed (+11pp, `2cbaf85a`,
  motoko 76%→83%); copying pi's full system prompt only added +1/18 (A/B) → NOT the gap.
- **Temperature** ruled out as the pi differentiator (pi runs qwen at the same 1.0 default).
- **Loop guards** (compel-write / definition-of-done) built + reverted (fired 0/18 or regressed).
- **Tolerant Hermes/XML parsing** — moot post-/v1; dropped.

## Done / superseded
- motoko ollama integration enabled + PATH/key rotation fix (this repo, landed).
- First failure analysis → root cause = AILANG-INTEGRATION (tool-calling), not language.
- **#2 ollama tool-calling over `/v1` (M-OLLAMA-V1-TOOLCALLING) — LANDED on `dev`
  (41c52ffe, 2026-06-17).** Root cause of the 26% closed: native `/api/chat` → 0 tool
  calls; now delegates to OpenAI-compat `/v1`. Aggregate confirmed: **AILANG 26%→79%**,
  motoko now BEATS opencode (72%) on AILANG, approaching pi (88%). Zero 0-tool-call runs.
- **`agent_tool_calls` surfaced in result JSON (M-MOTOKO-OBS-TOOLCALLS) — LANDED (e92bc4ba).**
- **/v1 HTTP timeout (63fc63e0) + native rig-lock in eval-suite (8a18e4e7) — LANDED.**
  Fixed a ~2h rig hang: /v1 (non-streaming, no timeout) stalled on a GPU-contention model
  reload; lock prevents concurrent rig jobs (fail-fast). The user's hypothesis — /v1 vs
  native endpoints caused the historical hangs — was correct.
- **Tool-call transcript retained (M-MOTOKO-OBS-TRANSCRIPT) — LANDED.** Parser kept counting
  tool calls but discarding content; now retains tool name + write path/content into
  `agent_transcript`. Unblocks root-causing the stub failures (backlog #1).

## Skill
When the analysis log has ~3+ entries and the cycle is repeatable, codify a
`motoko-analyzer` skill (the log template + lever taxonomy is the spec).
