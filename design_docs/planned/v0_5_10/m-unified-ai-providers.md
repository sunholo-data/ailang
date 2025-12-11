# M-UNIFIED-AI-PROVIDERS: Unified AI Provider Architecture

**Status**: Planned
**Target**: v0.5.10
**Priority**: P1 (Medium)
**Estimated**: 6 days (~30 hours total across all providers)
**Dependencies**: None

## AI-First Alignment Check

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | + | +1 | Single API for all providers |
| Preserve Semantic Clarity | + | +1 | Clear provider abstraction |
| Increase Determinism | + | +1 | Consistent behavior across entry points |
| Lower Token Cost | + | +1 | Stateful APIs reduce context repetition |
| **Net Score** | | **+4** | **Decision: Move forward** |

## Problem Statement

AILANG has **three entry points** for AI API calls, each with duplicated implementations:

| Entry Point | Current Location | Purpose |
|-------------|------------------|---------|
| Eval harness | `internal/eval_harness/api_*.go` | Benchmark AI code generation |
| CLI `--ai` flag | `cmd/ailang/ai_handlers.go` | Run AILANG programs with AI |
| std/ai effect | `internal/effects/ai.go` | AILANG programs call `AI.call()` |

**Current problems:**
- **Code duplication**: Same HTTP calls implemented 2-3 times per provider
- **Inconsistent features**: Eval harness may have features CLI lacks
- **Maintenance burden**: Bug fixes needed in multiple places
- **API drift**: New APIs (Gemini Interactions, OpenAI Responses) need 3x integration

## Goals

**Primary Goal:** Create unified `internal/ai/` package that all entry points share.

**Success Metrics:**
- Single implementation per provider, used by all entry points
- Support for both legacy and new APIs (generateContent/Interactions, Chat/Responses)
- Delete ~200 LOC of duplicated code from `ai_handlers.go`
- No regression in eval harness or CLI `--ai` behavior
- Clear extension path for future providers

## Solution Design

### Architecture

```
internal/ai/
├── provider.go                  # Common Provider interface
├── config.go                    # Model config loading (from models.yml)
│
├── openai/                      # OpenAI provider
│   ├── client.go                # HTTP client, routing
│   ├── chat.go                  # Chat Completions API
│   ├── responses.go             # Responses API (new)
│   ├── handler.go               # implements effects.AIHandler
│   └── client_test.go
│
├── gemini/                      # Google Gemini provider
│   ├── client.go                # HTTP client, auth
│   ├── generate.go              # generateContent API
│   ├── interactions.go          # Interactions API (new)
│   ├── handler.go               # implements effects.AIHandler
│   └── client_test.go
│
└── anthropic/                   # Anthropic provider
    ├── client.go                # Messages API
    ├── handler.go               # implements effects.AIHandler
    └── client_test.go
```

### Common Interface

```go
// internal/ai/provider.go

package ai

import "context"

// Request represents a generic AI request (v0.5.10: text only)
type Request struct {
    Model        string            // Model name (e.g., "gemini-2.5-flash")
    SystemPrompt string            // System/developer instructions
    UserPrompt   string            // User message
    MaxTokens    int               // Max response tokens
    Options      map[string]any    // Provider-specific options
}

// Response represents a generic AI response (v0.5.10: text only)
type Response struct {
    Text          string  // Generated text
    InputTokens   int     // Prompt tokens
    OutputTokens  int     // Completion tokens
    TotalTokens   int     // Total tokens
    ReasonTokens  int     // Reasoning tokens (if applicable)
    Model         string  // Model used
}

// Provider interface for AI providers
type Provider interface {
    // Generate makes a single completion request
    Generate(ctx context.Context, req *Request) (*Response, error)

    // Name returns the provider name
    Name() string
}

// Handler wraps a Provider for use with effects.AIHandler
type Handler struct {
    provider Provider
    model    string
}

func NewHandler(provider Provider, model string) *Handler {
    return &Handler{provider: provider, model: model}
}

// Call implements effects.AIHandler
func (h *Handler) Call(input string) (string, error) {
    resp, err := h.provider.Generate(context.Background(), &Request{
        Model:      h.model,
        UserPrompt: input,
        MaxTokens:  4096,
    })
    if err != nil {
        return "", err
    }
    return resp.Text, nil
}
```

### Consumer Integration

**1. Eval Harness:**
```go
// internal/eval_harness/api_google.go
import "github.com/sunholo/ailang/internal/ai/gemini"

func (a *AIAgent) callGemini(ctx context.Context, prompt string) (*GenerateResult, error) {
    client, err := gemini.NewClient(ctx)  // Handles auth via ADC
    if err != nil {
        return nil, err
    }

    resp, err := client.Generate(ctx, &ai.Request{
        Model:        a.model,
        SystemPrompt: "You are a programming assistant...",
        UserPrompt:   prompt,
    })
    if err != nil {
        return nil, err
    }

    return &GenerateResult{
        Code:         extractCodeFromMarkdown(resp.Text),
        InputTokens:  resp.InputTokens,
        OutputTokens: resp.OutputTokens,
        Model:        resp.Model,
    }, nil
}
```

**2. CLI `--ai` flag:**
```go
// cmd/ailang/ai_handlers.go
import (
    "github.com/sunholo/ailang/internal/ai"
    "github.com/sunholo/ailang/internal/ai/gemini"
    "github.com/sunholo/ailang/internal/ai/openai"
    "github.com/sunholo/ailang/internal/ai/anthropic"
)

func setupAIHandler(effCtx *effects.EffContext, model string) error {
    provider := guessProvider(model)

    var handler effects.AIHandler
    switch provider {
    case "google":
        client, _ := gemini.NewClientWithAPIKey(os.Getenv("GOOGLE_API_KEY"))
        handler = ai.NewHandler(client, model)
    case "openai":
        client := openai.NewClient(os.Getenv("OPENAI_API_KEY"))
        handler = ai.NewHandler(client, model)
    case "anthropic":
        client := anthropic.NewClient(os.Getenv("ANTHROPIC_API_KEY"))
        handler = ai.NewHandler(client, model)
    }

    effCtx.AI = effects.NewAIContext(handler)
    return nil
}
```

**3. std/ai effect (already works):**
```ailang
-- AILANG program using AI effect
import std/ai as AI

func ask(question: string) -> string ! {AI} =
    AI.call(question)

-- Run with: ailang run --caps IO,AI --ai gemini-2-5-flash --entry main
```

## Implementation Plan

This design doc coordinates three provider-specific implementations:

### Phase 1: Core Infrastructure (~4 hours)
- [ ] Create `internal/ai/provider.go` with common interfaces
- [ ] Create `internal/ai/config.go` for model config loading
- [ ] Add `internal/ai/handler.go` wrapper for effects.AIHandler
- [ ] Unit tests for common types

### Phase 2: Provider Implementations (parallel work)

| Provider | Design Doc | Estimated |
|----------|------------|-----------|
| Gemini | [M-GEMINI-INTERACTIONS-API](./m-gemini-interactions-api.md) | 4 days |
| OpenAI | [M-OPENAI-RESPONSES-API](./m-openai-responses-api-sprint.md) | 4 days |
| Anthropic | Migrate from ai_handlers.go | 0.5 days |

### Phase 3: Integration & Cleanup (~4 hours)
- [ ] Update `cmd/ailang/ai_handlers.go` to use unified providers
- [ ] Delete `OpenAIHandler`, `GoogleHandler`, `AnthropicHandler` structs
- [ ] Verify eval harness works with all providers
- [ ] Verify CLI `--ai` works with all models

### Phase 4: Documentation (~2 hours)
- [ ] Update CLAUDE.md with new architecture
- [ ] Update models.yml documentation
- [ ] Move design docs to implemented/

## Files to Create/Modify

**New files:**
- `internal/ai/provider.go` (~100 LOC)
- `internal/ai/config.go` (~50 LOC)
- `internal/ai/handler.go` (~40 LOC)
- `internal/ai/openai/` - See [M-OPENAI-RESPONSES-API](./m-openai-responses-api-sprint.md)
- `internal/ai/gemini/` - See [M-GEMINI-INTERACTIONS-API](./m-gemini-interactions-api.md)
- `internal/ai/anthropic/client.go` (~80 LOC)
- `internal/ai/anthropic/handler.go` (~30 LOC)

**Modified files:**
- `cmd/ailang/ai_handlers.go` - Delete handlers, use unified (~-180 LOC)
- `internal/eval_harness/api_openai.go` - Use unified (~-60 LOC)
- `internal/eval_harness/api_google.go` - Use unified (~-100 LOC)

**Net change:** ~1200 LOC new, ~340 LOC deleted = ~860 LOC net gain (but much cleaner)

## Success Criteria

- [ ] Single `internal/ai/` package with all providers
- [ ] Eval harness uses unified package (no duplicated API calls)
- [ ] CLI `--ai` uses unified package (handlers deleted from ai_handlers.go)
- [ ] All existing tests pass
- [ ] No regression in eval baseline results
- [ ] `make test && make lint` clean

## Testing Strategy

**Unit tests:**
- Common interface compliance for each provider
- Mock HTTP responses for each API
- Handler wrapper behavior

**Integration tests:**
- `ailang run --ai gemini-2-5-flash` works
- `ailang run --ai gpt5-mini` works
- `ailang run --ai claude-haiku-4-5` works
- `ailang eval-suite --models gemini-2-5-flash` matches baseline

## Non-Goals (v0.5.10)

- **Streaming support** - Defer to v0.5.11
- **Tool/function calling** - Defer to agent eval phase
- **Multi-modal inputs/outputs** - See below for roadmap
- **Rate limiting/retry** - Use provider defaults

## Multi-Modal Capabilities (Roadmap)

The underlying APIs support rich multimodal I/O. We'll implement in phases:

**v0.5.10 - Text Only (this release):**
- Text input → Text output
- Foundation for future multimodal

**v0.5.11 - Input Multimodal:**
- Image input (base64, URLs, Files API)
- PDF input (for document analysis)
- Audio input (for transcription)

**v0.5.12 - Output Multimodal:**
- Image generation (Imagen, DALL-E)
- Audio generation (TTS)
- Structured JSON output (`response_format`)

**v0.6.0 - Full Multimodal:**
- Video input (Gemini Files API)
- Streaming multimodal responses
- Artifact management (files, images, audio)

**API Support Matrix:**

| Capability | Gemini | OpenAI | Anthropic | AILANG Target |
|------------|--------|--------|-----------|---------------|
| Text I/O | ✅ | ✅ | ✅ | v0.5.10 |
| Image input | ✅ | ✅ | ✅ | v0.5.11 |
| PDF input | ✅ | ✅ | ✅ | v0.5.11 |
| Audio input | ✅ | ✅ | ❌ | v0.5.11 |
| Video input | ✅ | ❌ | ❌ | v0.6.0 |
| Image output | ✅ | ✅ | ❌ | v0.5.12 |
| Audio output | ✅ | ✅ | ❌ | v0.5.12 |
| JSON schema | ✅ | ✅ | ✅ | v0.5.11 |

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Breaking eval harness | High | Run full baseline before/after |
| Breaking CLI `--ai` | High | Test all providers manually |
| API incompatibilities | Med | Abstract differences in provider impl |
| Auth differences (ADC vs API key) | Low | Support both in each provider |

## References

- [M-GEMINI-INTERACTIONS-API](./m-gemini-interactions-api.md) - Gemini provider details
- [M-OPENAI-RESPONSES-API](./m-openai-responses-api-sprint.md) - OpenAI provider details
- [Gemini Interactions API](https://ai.google.dev/gemini-api/docs/interactions)
- [OpenAI Responses API](https://platform.openai.com/docs/api-reference/responses)
- [Current ai_handlers.go](../../../cmd/ailang/ai_handlers.go)

## Future Work

- **Streaming**: Add `GenerateStream()` method to Provider interface
- **Tools**: Add tool/function calling support for agentic workflows
- **Multi-modal**: Support image/audio inputs
- **Caching**: Response caching for repeated prompts
- **Observability**: Structured logging, metrics, tracing

---

**Document created**: 2025-12-11
**Last updated**: 2025-12-11
