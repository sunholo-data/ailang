# M-MOTOKO-HYBRID-TOOL-CORRELATION: Fix tool_use_id correlation in motoko's hybrid-bash mode

**Status**: Planned
**Target**: v0.18.3 (patch on top of v0.18.2)
**Priority**: P2 (Medium — degrades but doesn't block parallel motoko eval runs; ~7% sporadic failure on numeric_modulo and similar benchmarks)
**Estimated**: 2–4 hours (~50 LOC + tests)
**Dependencies**: ✅ M-MOTOKO-EVAL-HARNESS-HARDENING (v0.18.1) + M-MOTOKO-PARALLEL-EXECUTION-ISOLATION (v0.18.2)

**Author**: Claude Sonnet 4.6 + Mark
**Created**: 2026-05-08

**Source event**: Surfaced during the v0.18.2 parallel-execution sprint (2026-05-08). The 30-run aggregate (parallel-4 × 15-bench × 2 iterations) saw `numeric_modulo` fail with:

```
Provider returned error (Bedrock): messages.18.content.0:
unexpected `tool_use_id` found in `tool_result` blocks: hybrid-step-8.
Each `tool_result` block must have a corresponding `tool_use` block in the previous message.
```

---

## Problem Statement

motoko's hybrid mode (in `agent_loop_v2.ail`) lets a model emit a fenced bash block in prose instead of a structured `tool_call`. When this fires, motoko synthesizes a fake `BashExec` `ToolCall` with id `hybrid-step-N` (where N is the step index), runs it through the same dispatcher as native tool calls, and feeds the result back as a `tool_result` message.

The Anthropic Messages API enforces a strict invariant: every `tool_result` must reference a `tool_use_id` from a `tool_use` block in the immediately preceding assistant message. Anthropic Bedrock validates this strictly (other Anthropic backends are more permissive — same pattern as the v0.18.1 dotted-tool-name bug).

Currently motoko's synthesized `tool_use_id="hybrid-step-N"` is included in the `tool_result` envelope sent to Bedrock, but **the corresponding `tool_use` block is NOT injected into the prior assistant message** (since the prose was just text, not a tool call). Bedrock rejects with the error above.

**Frequency**: ~3-7% of motoko parallel runs (sporadic — depends on whether the model emits a fenced bash block in a given task). Did not surface in v0.18.1 because the dotted-tool-name issue masked it.

---

## Goals

**Primary Goal:** Hybrid bash extraction works end-to-end with Anthropic Bedrock without `tool_use_id` correlation errors.

**Success Metrics:**
- `numeric_modulo` benchmark passes consistently in motoko parallel runs (currently ~50% pass rate due to this bug)
- No new `Bedrock 400` errors on tool-result correlation in 30-run parallel-4 smoke aggregate
- Single new fixture test in motoko_agent verifies a hybrid-bash flow against the Anthropic Messages API contract

---

## Solution Approaches (TBD which lands)

### Option A: Inject synthetic `tool_use` block into prior assistant message
When hybrid mode synthesizes a `BashExec` ToolCall, also retroactively patch the prior assistant message to include a synthetic `tool_use` block with the same id. The model didn't generate it, but Anthropic doesn't care about authorship — only correlation.

**Pros**: Preserves the existing `tool_result` flow; minimal change to dispatch logic.
**Cons**: Mutating prior messages is unusual; requires careful handling to not break the conversation history.

### Option B: Skip the hybrid-result envelope; insert tool result inline as user message
Instead of sending a `tool_result` (which requires correlation), send the bash output as plain text in a `user` message. Loses the structured `tool_result` semantics but sidesteps the correlation requirement entirely.

**Pros**: Cleanest fix; doesn't depend on retroactive message mutation.
**Cons**: Loses telemetry semantics (`exit_code` etc.) and the model can't distinguish bash output from user messages.

### Option C: Disable hybrid mode for Anthropic Bedrock backends
Detect when the provider is Bedrock and skip hybrid mode (force native tool calls only). 

**Pros**: Trivially correct.
**Cons**: Hybrid mode exists because some prompts elicit fenced bash from cheap models that won't otherwise emit structured tool calls. Disabling it for Bedrock means motoko-claude-haiku-4-5 loses a meaningful capability.

**Recommendation**: Option A — Phase 1 should verify that retroactive message mutation works against Bedrock + Anthropic-direct + Vertex (all 3 Anthropic backends), then commit.

---

## Out of Scope (deferred)

- Replacing motoko's hybrid mode with a structured equivalent (would require a model-prompting change that's beyond this sprint)
- Adding a generic `provider_quirks` table in motoko config to encode strict-vs-permissive validation per provider (interesting but premature)

---

## Related Documents

- [v0.18.2 design doc](../implemented/v0_18_2/m-motoko-parallel-execution-isolation.md) — surfaced this issue during parallel-4 acceptance testing
- [v0.18.1 design doc](../implemented/v0_18_1/m-motoko-eval-harness-hardening.md) — same Bedrock-vs-Anthropic-direct validation asymmetry, different surface (tool name pattern)
- motoko `agent_loop_v2.ail` lines around `synthesize_hybrid_bash_call` and `dispatch_calls` — the change site

---

**Document created**: 2026-05-08
**Last updated**: 2026-05-08
