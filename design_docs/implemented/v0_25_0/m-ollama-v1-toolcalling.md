# M-OLLAMA-V1-TOOLCALLING: Route Ollama Tool-Calling via OpenAI-Compat /v1

**Status**: Planned
**Target**: v0.25.x
**Priority**: P0 for the motoko mission — root cause of motoko AILANG 26% (vs pi 96%)
**Estimated**: ~60 LOC + test; <0.5 day
**Mission item**: #2 in [motoko-mission.md](motoko-mission.md)

## Problem

motoko's agent loop on local qwen3.6 produces **0 tool calls** in ~71% of runs
(see [analysis log](../motoko-harness-analysis-log.md)) → no solution written → 26%
AILANG (and low across Python/JS/Go too — language-agnostic). Root cause, confirmed
from pi's source: AILANG's ollama provider (`internal/ai/ollama`) drives tool-calling
through ollama's **native `/api/chat`** Tools API (`github.com/ollama/ollama/api`),
and qwen3.6 does not reliably emit *native* tool calls over that path. pi (the 96%
reference harness) and opencode drive ollama through the **OpenAI-compatible
`/v1/chat/completions`** endpoint, where ollama's compat layer normalizes the model's
tool-call output and qwen emits `tool_calls` reliably.

## Approach

AILANG already has a complete OpenAI provider (`internal/ai/openai`) that does
`/v1/chat/completions` with tool-calling and supports a custom base URL
(`WithBaseURL`). Ollama's compat endpoint is `<ollama-host>/v1`. So: in
`ollama.Step`, when tools are present, **delegate to an `openai.Client` pointed at
the ollama host's `/v1`** (dummy API key `"ollama"`; ollama ignores auth). Reuses
the battle-tested OpenAI-style tool path — same mechanism as pi.

No import cycle: `openai` does not import `ollama`.

```go
// internal/ai/ollama/step.go (sketch)
if len(req.Tools) > 0 && os.Getenv("AILANG_OLLAMA_NATIVE_TOOLS") != "1" {
    v1 := openai.NewClient("ollama", openai.WithBaseURL(strings.TrimRight(c.endpoint,"/")+"/v1"))
    r2 := *req
    r2.Model = bareModel(req.Model) // strip ollama:/ollama/ prefix for the API
    return v1.Step(ctx, &r2)
}
// else: existing native /api/chat tool path (retained as opt-in fallback)
```

Gate behind `AILANG_OLLAMA_NATIVE_TOOLS=1` so the old native path stays available
for A/B / regression, but `/v1` is the default.

## Acceptance criteria
- [x] `ollama.Step` with `req.Tools` routes to `<host>/v1/chat/completions` (default).
      (`step.go:60-65`; `TestStep_ToolsViaOpenAICompat` asserts path + de-prefix + parse.)
- [x] `internal/ai/...` builds; existing ollama tests pass (native path still works under
      `AILANG_OLLAMA_NATIVE_TOOLS=1` — `TestStep_ToolsAdvertisedAndParsed_Native`).
- [x] Live check: `motoko-local-qwen3-6` on fizzbuzz (AILANG) now makes tool calls —
      **4 turns, 3 tool calls**, compile/runtime/stdout ✓, 1/1 pass (chain `c6409fd7`,
      2026-06-17) — vs the 0-tool-calls baseline.
- [x] No regression for opencode/pi (unaffected — they drive their own /v1, not this path).

## Status: Implemented (v0.25.x, 2026-06-17)

## Out of scope
- Switching the single-shot `Generate` path (non-agentic) — leave native.
- Tolerant Hermes/XML parsing (mission item #3) — only if `/v1` doesn't fully fix it.

## References
- pi root cause: [analysis log](../motoko-harness-analysis-log.md) (2026-06-17 entry)
- openai provider: `internal/ai/openai/{client,step,chat}.go`
