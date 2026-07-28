# Motoko Analyzer — the mission diagnostic loop

Run ONE motoko-mission improvement cycle with the deterministic diagnostic gates that this
mission's history proved necessary. The gates run FIRST and are not skippable: you may not
propose or build a fix until the cycle report and wire diff exist. This encodes the hard-won
ordering — **observe → diff → cheap-confirm → build → validate** — so we stop spending GPU/hours
on assumptions that a 2-minute capture refutes.

Read [`design_docs/motoko-mission.md`](../../../design_docs/motoko-mission.md) and
[`design_docs/motoko-harness-analysis-log.md`](../../../design_docs/motoko-harness-analysis-log.md)
for state. Append a log entry every cycle.

## The five gates (run in order — earlier gates are cheap, later ones expensive)

### Gate 1 — OBSERVE (deterministic, no GPU). Where is the gap, by failure MODE?
```
scripts/segment.sh           # failure-mode segmentation of the latest rotation
```
Outputs pass / **disengage** (fail, ≤2 tool calls) / **grind-wrong** (fail, >2) per harness, and the
motoko↔pi gap. **Do not propose a fix before pasting this.** If the gap is disengagement, continue;
if grind-wrong, the lever is correctness (different playbook).

> Note: the rotation result JSON's `finish_reason` is SUMMARY-level and HIDES per-turn truncation.
> Disengage stats alone are not enough — you must run Gate 2.

### Gate 2 — DIFF (small GPU). What does motoko actually put on the wire, and why 0 tool calls?
```
scripts/wire_diag.sh <benchmark> [<benchmark> ...]   # default: the top always-disengage benches
```
Captures the EXACT HTTP request+response bytes (via the `ai-http-log` sentinel + the openai-provider
wire logger) for failing benchmarks, then classifies each disengaged turn's **per-call** finish_reason
(**`length` = truncation**, `stop` = genuine "done") and prints motoko's request fields. This is the
step that surfaces truncation, dropped fields, parsing gaps — the things invisible to result-JSON stats.
**The cause is necessarily something that DIFFERS between motoko and pi for the same input.** Capture
pi too (`scripts/wire_diag.sh --pi <benchmark>`) when you need the comparison.

### Gate 3 — CHEAP-CONFIRM (tiny GPU, ~1 min). Does the proposed change actually fix it?
```
scripts/cheap_confirm.py <captured-request.json> --set max_tokens=16384   # raw replay, N samples
```
Replays a captured request with ONE field changed and reports engage/disengage. **Run this before any
build or A/B.** A param hypothesis confirmed here in 1 minute saves an hour of plumbing + a multi-hour A/B.

### Gate 4 — BUILD (routing rule). One well-scoped, verified change.
- AILANG (`internal/`, `cmd/`, `tools/`, eval rig) → commit to `dev` (gh = `sunholo-voight-kampff`, rebase).
- motoko_agent (`.ail`, profiles, prompts, TS) → DRAFT PR to `arniwesth/motoko_agent` via the fork.
- Add a unit test. Gate behind an env/flag if it changes default behavior.

### Gate 5 — VALIDATE + RECORD. Measure by the RIGHT metric; write the ledger.
- Validate on the BROAD set (core tier), not the biased flaky subset — smokes over-promise.
- Measure by the failure-mode delta (e.g. disengage-rate), not just pass rate.
- Append an analysis-log entry with: finding, the **Ruled-out ledger** (what was tested and refuted —
  this is the most valuable artifact; it stops re-chasing), lever class, prior-action status, next.

## Standing rules (the discipline that earned its keep)
1. **Never blame the controlled variable.** The eval holds the teaching prompt + benchmark constant to
   eliminate them — so neither can explain a harness gap. The cause DIFFERS between harnesses. (See
   memory `motoko-investigation-discipline`.)
2. **Source/capture before GPU.** Confirm from a reference harness's source (pi:
   `/opt/homebrew/lib/node_modules/@mariozechner/pi-*`; qwen-code/Qwen-Agent on GitHub + their issues)
   or a capture, before building. Reference harnesses + their issue trackers carry the answers
   (Qwen-Agent#789 was literally our bug).
3. **Don't generalize from a biased subset.** Flaky-6 smokes ≠ core-tier reality. Validate broad.
4. **Record negatives.** A refuted hypothesis is progress; the ruled-out ledger is the map.
5. **One verified change per cycle.** Conservative, lock-respecting (`scripts/segment.sh` reads only;
   GPU gates acquire the rig lock, `--parallel 1`, never force-push, never touch unrelated work).

## Failure-mode taxonomy (use these exact terms in the log)
```
fail
├── disengage  (≤2 tool calls — no real solution attempt)
│   ├── truncation     (per-call finish_reason=length — budget too small for the model's reasoning)
│   └── genuine-stop   (finish_reason=stop — model decided it was done)
└── grind-wrong (>2 tool calls — engaged but incorrect → correctness lever)
```

## When the cycle is mechanical enough, the launchd job can fire this skill unattended
A `dev.ailang.motoko-analyzer` launchd plist (StartInterval, rig-lock-respecting, blackout-aware —
mirror `tools/launchd/os-rotation-filler.*`) can run Gate 1 every idle window and open a chip/notify
when the dominant failure mode shifts. Judgment gates (3b hypothesis, 4 build) still need a human/agent.
