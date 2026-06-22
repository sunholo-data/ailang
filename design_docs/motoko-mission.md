# MISSION: Make Motoko Competitive on Local AILANG

**Type:** Long-running mission (advanced in downtime, e.g. while evals run)
**North star:** the AILANG-native harness (motoko) should match or beat the generic
harnesses (pi 96%, opencode 79%) on local-AILANG synthesis.

**STATUS 2026-06-21 — CORE GOAL MET (standard set).** Fresh broad post-fix baseline
(`eval_results/rotation/postfix-broad-20260621`, 49 smoke+core benches × 2 trials, clean dir):
motoko **pass@1 = 96.9% / best-of-N EXACT = 100% / 0 hard-fails**. **AIRTIGHT head-to-head confirmed
2026-06-21** (fresh pi on the SAME 49 benches × 2 trials, `postfix-broad-pi-20260621`): **pi = 96.9%
pass@1 / 98% best-of-N** → **motoko = pi at pass@1 (PARITY), motoko slightly ahead on best-of-N (100% vs
98%, within noise = 1 benchmark)**. Trajectory: ~26% (mission start) → **96.9% (parity with pi)**, driven
by the truncation fix; best-of-N shipped as a first-class rotation metric. **Core goal "match or beat pi"
= MET.**

**The standard set is now SATURATED** (both harnesses 100% best-of-N ceiling, 0 hard-fails) — it can no
longer discriminate "best vs equal." **Remaining (optional, "best not just equal"):** the large-context
frontier (P0, ailang-parse repo-source reimplement instrument) where pi and motoko actually diverge — a
deliberate focused-session build, not a cron task. See [[motoko-strategic-goal]] +
the [analysis log](motoko-harness-analysis-log.md).

## Roadmap: BEYOND parity (2026-06-21) — exploit what pi structurally can't do

The standard set is saturated (motoko = pi). Beating pi requires AILANG-native structural advantages a
generic harness has no access to. Priorities (mine):

- **R1 (TOP — build now) — Contract-aware best-of-N selector.** Extend the best-of-N selector (select-best
  / the rotation rollup) to reject runs-but-WRONG candidates via AILANG **contracts** (`ailang run
  --verify-contracts`, or Z3 `ailang verify`). pi has no typed verifier → it keeps selector-misses (the
  head-to-head `pipeline` case: pi picked runs-but-wrong; motoko 0 misses). The `contract_*` stretch
  benchmarks are the proving ground. **Lever class: AILANG-native MOAT.** Small build (Go/eval-harness).

- **R2 — Real-codebase "evolving codebase" eval (design-doc → sprint → motoko-execute → compare-reference).**
  The realistic instrument that replaces saturated synthetic benchmarks (the user's idea). **TARGET = real
  codebases WRITTEN IN AILANG, NOT the ailang-core repo's design_docs/ (those are the Go COMPILER — wrong
  substrate).** Use: `ailang-parse` (docx/office parsing), `ailang-demos` (ecommerce/BigQuery/budgets),
  `docparse`, `motoko_agent` (`src/core/*.ail`). Pipeline: take a feature/design-doc from one of THESE
  AILANG projects (reference AILANG implementation exists) → sprint-plan → motoko executes on the pre-feature
  copy → grade vs the reference (its AILANG tests + diff: "did motoko match or BEAT the human solution?").
  docx-reimplement (ailang-parse) is the first instance — already AILANG-source. Tests ALL differentiators
  on real evolving-AILANG-codebase work.

- **R3 — Cross-model + cross-language generality study.** Once motoko is optimal on-device: motoko vs pi/
  opencode on BIG openrouter models (gpt5, opus, gemini-3) + across langs (ailang/python/js/go). Q: (a) do
  motoko's gains hold with strong models (model-independent harness win)? (b) AILANG-specific or general?
  **Generality split:** best-of-N (check+run) is LANGUAGE-GENERAL (any compiler+runtime) = portable edge;
  contracts (R1) are AILANG-SPECIFIC = the moat. This positions motoko: portable advantage vs substrate moat.

Sequence: R1 (lever, now) → R2 (instrument, measures R1 + context_mode on real work) → R3 (positioning).
docx-reimplement = R2's first instance + the context_mode/large-context proving ground.

**Large-context infra prerequisites (surfaced by docx-reimplement, 2026-06-22) — clear these for a clean P0/R2 grade:**
1. ailang parser PANIC on `s[0]` index access → **FIXED on dev** (dbc8bf391; nil-guard + regression test).
2. ollama `/v1` TOTAL timeout aborts legitimately-long requests ("context deadline exceeded", killed docx at steps 14 & 27). Band-aid: raise total to 1800s (motoko_agent#65). **Robust fix designed:** [`planned/m-ollama-v1-streaming-idle-timeout.md`](planned/m-ollama-v1-streaming-idle-timeout.md) — stream `/v1` + idle/inter-chunk deadline (tolerates long generation, still catches hangs). P1, v0.26.0.
3. Root lever for prefill size = **P2 context-compression** (a bigger timeout can't shrink the prompt). Sequence: #65 (unblock now) → streaming-idle-timeout (robust) → P2 (root).

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

**[NEXT — 2026-06-20] CONVERGENCE EFFICIENCY (context hygiene).** Truncation + disengagement are
fixed; motoko is at pass-rate parity with pi (88.9% vs 90.4%) but **3–10× less efficient** (pi median
5 turns vs motoko 15–50). Source+transcript root cause: motoko's **verbose tool results + 70%
auto-compaction erase the model's own writes**, forcing re-reads/rewrites (vicious cycle). Plan in
[`planned/m-ailang-semantic-context.md`](planned/m-ailang-semantic-context.md): near-term = match pi
(truncate tool results, raise compaction floor, echo writes; fork PR, A/B by turns-to-success), then
AILANG-native semantic-context routes (distilled diagnostics, type/effect-directed surfacing, AST
diffs, trace distillation, typed projection layer). Pending cheap-confirm: `wire_diag` elision capture.

0. **[LANDED 2026-06-19 — TRUNCATION fixed.]** max_tokens floor 16384 (fac848054): disengaging
   benchmarks **21%→79%**, finish=length 11→0. Per-model precision plumbed (006a679a6 + motoko PR #48:
   registry max_output_tokens→motoko via AILANG_OLLAMA_MAX_TOKENS). NEXT (blocked on fresh rotation
   data): re-run Gate 1 to measure the full gap drop. Original finding below:
   **[ROOT CAUSE: TRUNCATION.]** Wire-proven: motoko's
   disengagement = qwen3.6 thinks ~4k+ tokens (median 13.9k chars reasoning) and **`finish_reason=length`
   truncates at `max_tokens=4096` BEFORE the tool call** (11/14 disengaged turns = length). pi sends
   `max_completion_tokens=16384`. **Fix: raise the agent/ollama max_tokens to ≥16384** (pi-faithful;
   AILANG default is `internal/ai/handler.go:95`). Also: motoko's `enable_thinking:false` is DROPPED
   before /v1 (never forwarded) — optional second fix (forward a real thinking-disable param). Reasoning-
   parsing/hermes hypothesis REFUTED (0 tool calls found in reasoning); reasoning-CAPTURE done anyway
   (79714e3d5, c1f87275e) — it's the observability that found this. Next: raw-replay confirm
   (max_tokens=16384 → qwen finishes?), then raise + A/B by disengage-rate. Routing: AILANG default vs
   motoko per-request max_tokens (motoko has no max_tokens config knob today).
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
2. **[RULED OUT 2026-06-19 — SAMPLING is already optimal].** Wire-verified motoko sends only
   `{model, max_tokens}` (no temp/top_p/top_k); ollama then applies qwen3.6's OWN modelfile vector
   (`temperature 1, top_p 0.95, top_k 20, presence_penalty 1.5`). So motoko already runs the model's
   recommended sampling — there is NO sampling lever (the "forcing greedy temp 0" premise was false;
   `resolveOllamaTemperature`→0 is omitempty → unsent). The `AILANG_OLLAMA_TEMPERATURE` knob stays as
   an opt-in override only. See analysis log 2026-06-19 SAMPLING-RULED-OUT.
3. **[THE residual, data-elevated 2026-06-19 — CONVERGENCE/correctness, partly SHARED with pi].**
   Post-truncation-fix the gap is no longer disengagement. Partial h2h (n=18 ea): motoko 88% vs pi
   94%; motoko's only net loss = `explicit_dataflow_ssa` (grind, 32 tool calls fighting AILANG
   `expected float arguments` numeric-type friction), and `csv_to_json_converter` fails for BOTH
   harnesses (shared qwen-on-AILANG limit). "Step budget too low" ruled out (motoko runs ~50 steps
   via `rpc.ail` clamp). **Next (disciplined, NO pre-committed lever): complete the full head-to-head
   as a clean MEASUREMENT, read EVERY residual transcript, then classify each as (i) motoko-specific
   & fixable harness lever, or (ii) shared qwen-on-AILANG capability friction → AILANG eval-gap item,
   not a motoko harness lever.** See analysis log 2026-06-19 SAMPLING-RULED-OUT / residual entry.

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
