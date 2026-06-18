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

1. **[motoko PR + GPU] Iteration persistence — THE gap.** Quantified 2026-06-18: on FAILURES
   motoko gives up at median **2** turns (max 9) while pi grinds median **33** (max 164) on the
   SAME model; motoko's model stops emitting tool calls after ~2 turns and motoko finalizes.
   Not a budget cap (passes reach 49 turns), not prompt (sub-dominant, A/B'd), not temperature
   (equal). Two GPU-bound steps:
   - **(a) Root cause** — capture a fresh failing motoko run's FULL turn-by-turn via the
     request-dump (`AILANG_OLLAMA_LOG_REQUESTS`/sentinel, built) + transcript, to see *why* the
     model disengages at turn 2 (gives up in prose? hits an unhelpful tool error? thinks it's
     done?). One lock-respecting GPU run.
   - **(b) Fix** — a motoko-side loop-persistence change: don't finalize on an early apparent-stop
     while the solution clearly isn't working; re-prompt / keep iterating (bounded). Draft PR to
     `arniwesth/motoko_agent`, A/B-validated on the rig. (The earlier "write a file" guard
     M-MOTOKO-COMPEL-WRITE was the wrong shape — this is about *persistence*, not just *writing*.)
2. **[AILANG, needs GPU] Temperature A/B** — `AILANG_OLLAMA_TEMPERATURE` knob landed (off by
   default). A/B unset vs 0.2/0.3 on the 6 flaky benchmarks; if it lifts engagement, make a low
   temperature the default; else investigate qwen thinking-mode. Lower priority than #1 (the
   request-diff showed pi runs the same 1.0 default, so temperature isn't the pi gap — but lower
   variance may still reduce motoko's flakiness).
3. **[AILANG] Convergence / robustness**: `finish_reason=tool_calls and no run_summary` +
   `step budget exhausted` tail. Lower priority — the lock + /v1 timeout removed the contention
   trigger; revisit if it recurs un-contended.

## Resolved / ruled out (this investigation arc)
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
