# M-MOTOKO-OBS-TRANSCRIPT: Retain Motoko Tool-Call Transcript

**Status**: Implemented (v0.25.x, 2026-06-17)
**Target**: v0.25.x
**Priority**: P1 — mission item #1 (observability), unblocks diagnosing the stub-failure class
**Estimated**: ~50 LOC + test; <0.5 day

## Problem

9 of 10 current motoko AILANG failures submit the **byte-identical 112-char
placeholder** (`module benchmark/solution … // TODO: Add your solution code below`)
— the eval reads the unmodified `solution.ail` because motoko made **1 tool call,
then stopped (`finish=stop`) without writing a valid solution.** Passing runs
capture real code, so the capture path works; these specific runs genuinely
produced no solution.

We could not diagnose *why* (wrong write path? a non-write tool call then quit?
silent write failure?) because the motoko executor **parses the session JSONL to
count tool calls but discards their content** (`parser.go` decoded
`native_tool_calls` as `{id, tool, arguments}`, did `numToolCalls += len(...)`,
and threw the rest away), and the workspace JSONL is `os.RemoveAll`'d after each
run. Result: `agent_transcript=None`, no `session_*.jsonl` retained.

## Approach

Have the parser **accumulate a compact, bounded transcript** of each tool call
(tool name + truncated args — crucially the write **path** and content preview)
and the final `done` output, into `res.Transcript`. That already flows:
`executor.Result.Transcript` → `AgentBenchmarkResult.SessionLog` →
`RunMetrics.AgentTranscript` → retained in the per-run result JSON. No new
plumbing — the field was simply never populated for motoko.

`internal/executor/motoko/parser.go`:
- In the `native_tool_calls` case, append `tool_call: <tool> <args…>` per call
  (`summarizeToolCall`, args truncated to 400 chars, newlines collapsed).
- In the `done` case, append `done: <output…>` (300 chars).
- After the event loop: `res.Transcript = transcript.String()`.

Bounded per-call (400 chars) and by the existing 1 MB `truncateField` cap; a
typical run is a few hundred bytes.

## What it buys us

Next rotation cycle, a failing run's transcript will read e.g.
`tool_call: WriteFile {"path":"solution.ail",…}` — immediately telling us:
- **no `WriteFile` at all** → motoko made a non-write call and quit (loop/prompt), or
- **`WriteFile` to the wrong path** (e.g. `solution.ail` vs `benchmark/solution.ail`)
  → a path/isolation bug (AILANG-side fix), or
- **`WriteFile` of a stub** → genuine model behaviour.

Then we fix the *actual* root cause instead of guessing.

## Acceptance criteria
- [x] `res.Transcript` captures tool name + write path/content per call + done output.
      Verified: `tool_call: WriteFile {"path":"f.ail","content":"export func answer()…"}`.
- [x] Flows to the result JSON `agent_transcript` (existing plumbing).
- [x] Bounded (per-call 400-char arg cap; 1 MB field cap).
- [x] `go test ./internal/executor/motoko/...` green (extended
      `TestParseSessionJSONL_Success` asserts the transcript content).

## Out of scope
- Acting on what the transcript reveals — that's the *next* cycle, once the
  rotation has produced transcripts for the failing benchmarks.
- Retaining the full raw JSONL file (the compact transcript is enough to
  diagnose; revisit only if a case needs per-token detail).

## References
- Failure analysis: [motoko-harness-analysis-log.md](../../motoko-harness-analysis-log.md)
- Placeholder seeding / read-back: `internal/eval_harness/agent_runner_multi.go:138-168,330`
- Tool-call decode: `internal/executor/motoko/parser.go` (`native_tool_calls`).
