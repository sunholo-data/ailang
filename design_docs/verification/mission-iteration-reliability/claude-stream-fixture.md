# The claude stream fixture, and the four defects it found

**Date:** 2026-09-14 (attended)
**Closes:** handover step 1 — "capture one claude-code and one pi stream fixture"
**Fixture:** `internal/executor/claude/testdata/claude_stream_partial.ndjson`

## Why a fixture was the right instrument

The per-harness accumulation table in `internal/executor/tokens_processed.go` was written
from inference: no claude-code or pi stream had ever been recorded in this tree, so each
row stated what the code *appeared* to do. The handover called capturing one "the cheapest
high-value item", expecting it to upgrade the table from documented to asserted.

It did not merely confirm the table. **It falsified the claude row**, and the row was wrong
in the direction that disarms a guard.

## What was captured

A real run under the executor's own flags — `--output-format stream-json
--include-partial-messages --verbose` (`claude.go:177-180`) — reading five named files, one
`Read` call each. Six turns. Tool-result payloads are trimmed and `$HOME` is rewritten;
every usage-bearing line is byte-exact. 70KB, in line with pi's existing 56KB fixture.

The first capture attempt **omitted `--include-partial-messages`** and produced no
`stream_event` lines at all — the executor's entire parser is under that flag. Worth
knowing: a fixture captured without the executor's exact argv is a fixture of a different
program.

## What the recording settles

Per-turn usage sums to the result event **exactly**, on all four counters:

| counter | per-turn sum | result event |
|---|---|---|
| `input_tokens` | 10+8+8+8+8+8 = 50 | 50 |
| `cache_creation_input_tokens` | 0+11757+10005+9610+10069+7773 = 49,214 | 49,214 |
| `cache_read_input_tokens` | 315,648 | 315,648 |
| `output_tokens` | 214+104+106+106+122+54 = 706 | 706 |

Two documented beliefs die here:

1. **"claude emits CUMULATIVE usage, so assign the latest value."** The counters are
   cumulative *within* a turn and **reset at every `message_start`**. A run total is the
   SUM of the per-turn finals. Assigning kept only the last turn — 8 input and 54 output
   against a true 50 and 706.
2. **"cache_creation_input_tokens sits outside the usage block the message_delta handler
   reads."** It is inside both `message_start.message.usage` and `message_delta.usage`. The
   in-flight kill was abandoned on a premise that a single recording disproves.

## The four defects

Compounding, all in `internal/executor/claude`:

1. **The in-flight token cap could never fire.** With cache creation pinned at 0 and only
   the last turn assigned, the guard weighed **62 tokens against a cap of 20,000** on a run
   that processed 49,970 — off by ~800x. A cap is a runaway backstop; this one could only
   observe a runaway after it had been paid for in full.
2. **Counters assigned, not summed** — the accumulation bug above, which also made the
   in-flight cost `Budget` see only small inter-turn deltas.
3. **A thrash kill was banked as `FinishReason: "stop"`.** `success` was correctly set to
   false, but the finish reason came from the provider's own subtype, so the
   failure-attribution layer could not distinguish a cap kill from a clean finish.
4. **The kill path under-reported its own tokens.** The non-final return paths carried
   `InputTokens`/`OutputTokens` only — no cache buckets — so the one Result that exists to
   report "this run was too big" omitted the bucket it was killed for.

Defect 4 was latent while defect 1 stood: with no in-flight kill, that path was rarely
reached. Fixing 1 made 4 load-bearing. This is the ordinary shape of a guard that has never
actually fired.

## What changed

`applyTurnUsage` in `claude.go` is now the single place a usage block is folded in: it sums
per-turn finals, charges the cost budget only the increment, and tests the cap — killing
the process on breach. It is called from `message_start` (so a turn large enough to breach
is caught before its output is generated) and from `message_delta`. The result-event test
remains as a backstop rather than the only check. `killFinishReason` makes a guard kill
attributable on every return path.

`maxInt` keeps within-turn counters monotonic, so a duplicate or out-of-order event cannot
lower a running total and refund an over-budget agent.

## Asserted, not documented

- `TestClaudeStreamUsageIsPerTurnAndSums` — reads the fixture directly, stating the
  *provider's* contract independently of how we consume it. If claude-code switches to
  run-cumulative usage, this fails first and names the reason.
- `TestClaudeTokenCapKillsInFlight` — the fake claude holds its result event behind a sleep
  and touches a marker just before emitting it, so "killed mid-stream?" is answered by a
  file's existence rather than by timing. Against the old code the marker was always
  written; the test's runtime fell from 3.25s to 0.29s once the kill worked.
- `TestClaudeUncappedRunReportsFixtureTotals` — the control.
- `TestPiSumsPerTurnUsageIncludingCacheWrites` / `TestPiCapCountsCacheCreation` — pi's half
  of the handover item needed **no new capture**: `tool_use.ndjson` already carries four
  turns and 9,264 tokens of `cacheWrite`. The cap test fires at 3,000, an order of
  magnitude above the run's entire input+output of 235, so it can only pass if cache
  creation is inside the guard.

## Still unasserted

opencode, codex and motoko have no recorded stream behind their rows. **Any row in that
table without a fixture behind it is a belief**, and the claude row is what a belief costs.

## Open, deliberately not fixed here

`claude.go` never calls `Budget.AddCache`, so the claude **cost** budget is blind to cache
tokens — the same class as the pi fix in `b3ed92633` ("the in-flight budget could not see
prompt-cache tokens — 69% of a real bill"). Fixing it changes cost-kill behaviour in evals,
which is a wider blast radius than a token cap, so it is reported rather than bundled.
