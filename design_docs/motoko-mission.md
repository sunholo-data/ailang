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

1. **[AILANG, BLOCKED ON DATA → unblocks next cycle] Root-cause the stub failures.** 9/10
   AILANG failures submit the seeded `solution.ail` placeholder (1 tool call, no solution
   written, finish=stop). M-MOTOKO-OBS-TRANSCRIPT (landed) now retains the tool-call
   transcript — **next cycle: read the failing runs' transcripts** and determine which:
   (a) `WriteFile` to the wrong path / isolation mismatch → AILANG fix; (b) non-write call
   then quit, or genuine stub → motoko prompt/loop PR. Fix the side the data points to.
2. **[AILANG] Request-param engagement (temperature / thinking)** — the pi-faithful lever.
   Diagnosis (2026-06-17): failures = qwen non-deterministically emits prose / 0 tool calls;
   pi has NO loop magic (ends on no-tool-call too) and sends a vanilla /v1 body — its edge is
   the model *engaging*. qwen3.6's ollama default is **temperature 1.0** (high variance) and
   we don't gate qwen's thinking; pi does. **Next: A/B temperature ~0.2–0.3 on the agentic
   ollama path** (few-line change, commit to dev) on the 6 benchmarks. The "compel-write loop
   guard" (M-MOTOKO-COMPEL-WRITE) was built + A/B'd + **reverted** (guard fired 0/18, one
   regression) — parked in favour of this.
3. **[AILANG] Convergence / robustness**: `finish_reason=tool_calls and no run_summary`
   (seen under ollama contention) + the `step budget exhausted` tail. Lower priority — the
   lock + /v1 timeout removed the contention trigger; revisit if it recurs un-contended.
4. **[AILANG, only if needed] Tolerant tool-call parsing** for qwen Hermes/XML blocks —
   likely moot post-/v1; parked.

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
