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

1. **[AILANG] Observability: retain motoko session JSONL on eval failure.** Failing
   runs drop the raw model response (turns/finish null, code empty). Capture it so we
   can see qwen's actual tool-call output. *Unblocks everything below.*
2. **[motoko PR] Prompt: compel tool use** on small local models (SYSTEM.md / a
   `motoko_ext_*`), if the model still under-uses tools after the /v1 fix (#2 below).
3. **[AILANG] Convergence**: the `step budget exhausted` tail.
4. **[AILANG, only if needed] Tolerant tool-call parsing** in `internal/ai/ollama/step.go`
   for qwen's Hermes/XML `<function>` blocks — likely unnecessary now that /v1 normalizes
   tool calls; keep parked unless a rotation shows residual 0-tool-call runs.

## Done / superseded
- motoko ollama integration enabled + PATH/key rotation fix (this repo, landed).
- First failure analysis → root cause = AILANG-INTEGRATION (tool-calling), not language.
- **#2 ollama tool-calling over `/v1` (M-OLLAMA-V1-TOOLCALLING) — LANDED on `dev`
  (41c52ffe, 2026-06-17).** Root cause of the 26% closed: native `/api/chat` → 0 tool
  calls; now delegates to OpenAI-compat `/v1`. Live fizzbuzz: 4 turns / 3 tool calls /
  pass vs 0-tool-calls baseline. Re-measure aggregate on next OS rotation.

## Skill
When the analysis log has ~3+ entries and the cycle is repeatable, codify a
`motoko-analyzer` skill (the log template + lever taxonomy is the spec).
