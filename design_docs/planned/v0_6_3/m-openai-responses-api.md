# M-OPENAI-RESPONSES-API: OpenAI Responses API & Agent Module

**Status**: Planned
**Target**: v0.6.0 (postponed from v0.5.10)
**Priority**: P1 - Medium (enables new model support)
**Estimated**: 3-4 days
**Dependencies**: None (models.yml updated with gpt5-1-codex-max)

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | External API calls remain non-deterministic |
| A2: Replayability | +1 | Unified client enables consistent request/response logging |
| A3: Effect Legibility | 0 | AI calls already explicit in effect system |
| A4: Explicit Authority | 0 | No capability changes |
| A5: Bounded Verification | 0 | External dependency |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | Unified module simplifies AI provider integration |
| A8: Minimal Syntax | 0 | No syntax changes |
| A9: Cost Visibility | +1 | Reasoning token tracking improves cost accuracy |
| A10: Composability | +1 | Unified internal/openai/ package for all OpenAI models |
| A11: Structured Failure | +1 | Consistent error handling across APIs |
| A12: System Boundary | +1 | Clear API boundary between Chat and Responses |

**Net Score: +6** ✅ Proceed to implementation

### Hard Violation Check

- [x] A1 (Determinism): No additional nondeterminism (external API is inherently non-deterministic)
- [x] A3 (Effects): AI effect already required for calls
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Simplifies programmatic AI access

## Problem Statement

OpenAI has released `gpt-5.1-codex-max`, their most powerful coding model (77.9% SWE-Bench). However, this model uses the **Responses API** (`/v1/responses`) instead of Chat Completions (`/v1/chat/completions`).

**Current State:**
- Eval harness only supports Chat Completions API
- `gpt-5.1-codex-max` returns error: "This is not a chat model"
- No unified OpenAI agent module for broader use beyond evals
- Each new OpenAI API requires changes to `api_openai.go`

**Impact:**
- Cannot benchmark the best coding model available
- No path to support future Responses API models
- Code duplication if we add Responses API only to eval harness

## Goals

**Primary Goal:** Create a reusable OpenAI agent module supporting both Chat Completions and Responses APIs.

**Success Metrics:**
- `gpt-5.1-codex-max` works in eval suite
- Zero code duplication between API implementations
- < 5% overhead vs direct API calls
- Module usable outside eval harness (agent workflows, REPL integration)

## Solution Design

### Overview

Create a new `internal/openai` package providing a unified interface for OpenAI API interactions. This module will:
1. Support both Chat Completions and Responses APIs
2. Auto-detect which API to use based on model name
3. Handle reasoning tokens, tool use, and model-specific parameters
4. Be consumed by eval harness and future agent integrations

### Architecture

```
internal/openai/
├── client.go         # Main client with provider selection
├── chat.go           # Chat Completions API implementation
├── responses.go      # Responses API implementation
├── types.go          # Shared request/response types
└── client_test.go    # Unit tests with mocked HTTP
```

**Components:**

1. **Client** (`client.go`): Unified interface that routes to appropriate API
   ```go
   type Client struct {
       apiKey string
       http   *http.Client
   }

   func (c *Client) Generate(ctx context.Context, req *Request) (*Response, error)
   ```

2. **Chat Completions** (`chat.go`): Existing functionality extracted
   - `/v1/chat/completions` endpoint
   - System/user message format
   - Standard token counting

3. **Responses API** (`responses.go`): New implementation
   - `/v1/responses` endpoint
   - `reasoning.effort` parameter support
   - Extended `max_completion_tokens` handling
   - Tool/function calling support

4. **Types** (`types.go`): Shared structures
   ```go
   type Request struct {
       Model            string
       Prompt           string
       SystemPrompt     string
       Seed             *int64
       ReasoningEffort  string // "low", "medium", "high" - Responses API only
       MaxTokens        int
       Tools            []Tool // For agentic use
   }

   type Response struct {
       Content         string
       InputTokens     int
       OutputTokens    int
       ReasoningTokens int
       Model           string
       FinishReason    string
   }
   ```

### Model Detection

Auto-detect which API to use based on model name patterns:

```go
// Models requiring Responses API
var responsesAPIModels = map[string]bool{
    "gpt-5.1-codex":     true,
    "gpt-5.1-codex-max": true,
    // Future models can be added here
}

func (c *Client) Generate(ctx context.Context, req *Request) (*Response, error) {
    if responsesAPIModels[req.Model] || strings.Contains(req.Model, "codex") {
        return c.callResponsesAPI(ctx, req)
    }
    return c.callChatCompletions(ctx, req)
}
```

### Responses API Format

Based on OpenAI documentation, the Responses API uses:

```json
{
  "model": "gpt-5.1-codex-max",
  "input": [
    {"role": "developer", "content": "You are a coding assistant..."},
    {"role": "user", "content": "Write a function..."}
  ],
  "reasoning": {
    "effort": "high"
  },
  "max_output_tokens": 16384
}
```

Response format:
```json
{
  "id": "resp_...",
  "output": [
    {"type": "message", "content": [{"type": "output_text", "text": "..."}]}
  ],
  "usage": {
    "input_tokens": 123,
    "output_tokens": 456,
    "reasoning_tokens": 100
  }
}
```

### Implementation Plan

**Phase 1: Core Module** (~6 hours)
- [ ] Create `internal/openai/` package structure
- [ ] Implement `types.go` with shared structures
- [ ] Implement `client.go` with routing logic
- [ ] Extract existing Chat Completions to `chat.go`
- [ ] Write unit tests for Chat Completions

**Phase 2: Responses API** (~6 hours)
- [ ] Implement `responses.go` with full API support
- [ ] Add `reasoning.effort` parameter handling
- [ ] Handle tool/function call responses
- [ ] Add response parsing for different output types
- [ ] Write unit tests with mocked HTTP responses

**Phase 3: Eval Integration** (~4 hours)
- [ ] Update `eval_harness/api_openai.go` to use new module
- [ ] Update models.yml with API type hints (optional)
- [ ] Test with `gpt-5.1-codex-max` on fizzbuzz benchmark
- [ ] Update documentation

**Phase 4: Extended Features** (~4 hours)
- [ ] Add streaming support (optional)
- [ ] Add retry logic with exponential backoff
- [ ] Add rate limiting support
- [ ] Document module for external use

### Files to Modify/Create

**New files:**
- `internal/openai/client.go` - Main client, ~150 LOC
- `internal/openai/chat.go` - Chat Completions, ~100 LOC (extracted)
- `internal/openai/responses.go` - Responses API, ~200 LOC
- `internal/openai/types.go` - Shared types, ~80 LOC
- `internal/openai/client_test.go` - Unit tests, ~300 LOC

**Modified files:**
- `internal/eval_harness/api_openai.go` - Use new module, -100 LOC
- `internal/eval_harness/ai_agent.go` - Minor updates, ~10 LOC
- `internal/eval_harness/models.yml` - Add api_type field (optional)

## Examples

### Example 1: Eval Harness Usage

**Before (current):**
```go
// api_openai.go - hardcoded to chat completions
func (a *AIAgent) callOpenAI(ctx context.Context, prompt string) (*GenerateResult, error) {
    url := "https://api.openai.com/v1/chat/completions"
    // ... 80 lines of Chat Completions specific code
}
```

**After:**
```go
// api_openai.go - uses unified module
import "github.com/sunholo/ailang/internal/openai"

func (a *AIAgent) callOpenAI(ctx context.Context, prompt string) (*GenerateResult, error) {
    client := openai.NewClient(a.apiKey)
    resp, err := client.Generate(ctx, &openai.Request{
        Model:        a.model,
        Prompt:       prompt,
        SystemPrompt: "You are a programming assistant...",
        Seed:         a.seed,
    })
    if err != nil {
        return nil, err
    }
    return &GenerateResult{
        Code:         extractCodeFromMarkdown(resp.Content),
        InputTokens:  resp.InputTokens,
        OutputTokens: resp.OutputTokens,
        Model:        resp.Model,
    }, nil
}
```

### Example 2: Standalone Agent Usage

```go
// Future: agent workflows outside eval harness
package main

import (
    "context"
    "github.com/sunholo/ailang/internal/openai"
)

func main() {
    client := openai.NewClient(os.Getenv("OPENAI_API_KEY"))

    // Use Codex-Max for agentic coding task
    resp, err := client.Generate(context.Background(), &openai.Request{
        Model:           "gpt-5.1-codex-max",
        Prompt:          "Implement a binary search tree in Go",
        ReasoningEffort: "high",  // Tell model to think deeply
        MaxTokens:       16384,
    })
    // ...
}
```

### Example 3: Running Eval with Codex-Max

```bash
# After implementation, this will work:
ailang eval-suite --models gpt5-1-codex-max --benchmarks fizzbuzz,recursion_factorial

# Expected output:
# Running benchmark: fizzbuzz
# Model: gpt5-1-codex-max (Responses API)
# ✓ Benchmark completed
# Result: PASS (100%)
```

## Success Criteria

- [ ] `gpt-5.1-codex-max` successfully completes fizzbuzz benchmark
- [ ] Eval suite auto-detects correct API for each model
- [ ] No regressions in existing Chat Completions models (gpt5, gpt5-mini, gpt5-1)
- [ ] Token counting correct for both APIs (including reasoning tokens)
- [ ] Module importable and usable outside eval harness
- [ ] All tests passing (>80% coverage for new module)
- [ ] Documentation updated (models.yml notes, eval guide)
- [ ] **Update eval suites**: Replace `gpt5-2` with `gpt5-1-codex-max` in `benchmark_suite` and `extended_suite` (Codex-Max is OpenAI's flagship coding model with 77.9% SWE-Bench)

## Testing Strategy

**Unit tests:**
- Mock HTTP responses for both APIs
- Test model detection logic
- Test request/response marshaling
- Test error handling (rate limits, auth errors)

**Integration tests:**
- Test with real API (gated by API key presence)
- Verify token counts match OpenAI dashboard
- Test timeout/cancellation behavior

**Manual testing:**
- Run `ailang eval-suite --models gpt5-1-codex-max --benchmarks fizzbuzz`
- Compare results with gpt5-1 on same benchmark
- Verify cost calculations in eval report

## Non-Goals

**Not in this feature:**
- Streaming responses - Deferred to v0.5.11 if needed
- Function/tool calling in evals - Eval prompts are code-only
- Batch API support - Different use case (async)
- Fine-tuning API - Different use case

## Timeline

**Day 1** (6 hours):
- Phase 1: Core module structure and Chat Completions extraction

**Day 2** (6 hours):
- Phase 2: Responses API implementation and testing

**Day 3** (4 hours):
- Phase 3: Eval integration and real-world testing

**Day 4** (4 hours):
- Phase 4: Extended features, documentation, cleanup

**Total: ~20 hours across 4 days**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Responses API format changes | High | Pin to documented version, add version header |
| Model detection fails for new models | Medium | Make detection configurable via models.yml |
| Performance regression | Low | Benchmark before/after, keep hot paths efficient |
| Breaking eval harness | High | Comprehensive test suite before migration |

## References

- [OpenAI Responses API Docs](https://platform.openai.com/docs/api-reference/responses)
- [GPT-5.1-Codex-Max Announcement](https://openai.com/index/gpt-5-1-codex-max/)
- [models.yml](../../../internal/eval_harness/models.yml) - Model configuration
- [api_openai.go](../../../internal/eval_harness/api_openai.go) - Current implementation

## Future Work

- **Streaming support**: Add streaming for long-running agent tasks
- **Tool/function calling**: Enable agentic workflows with tool use
- **Gemini agent module**: Similar refactor for Google APIs
- **Anthropic agent module**: Similar refactor (already cleaner but could unify)
- **Agent orchestration**: Higher-level agent framework using these modules

---

**Document created**: 2025-12-10
**Last updated**: 2025-12-10
