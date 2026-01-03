# M-AI-OLLAMA: Unified Ollama Provider for Local Models

**Status:** Planned
**Target:** v0.6.2
**Priority:** P1 (Medium)
**Estimated:** 2-3 days (includes eval harness migration)
**Dependencies:** M-UNIFIED-AI-PROVIDERS (v0.5.10) - partially implemented (CLI done, eval harness NOT migrated)
**Created:** 2024-12-24
**Updated:** 2024-12-24

## Problem Statement

The unified AI provider system (`internal/ai/`) supports OpenAI, Anthropic, and Google - but **no local models**. This creates gaps:

**⚠️ CRITICAL FINDING:** The eval harness (`internal/eval_harness/`) does NOT use the unified `internal/ai/` package! It has its own duplicated implementations:
- `api_openai.go` - ~120 LOC duplicate
- `api_anthropic.go` - ~100 LOC duplicate
- `api_google.go` - ~150 LOC duplicate

This means adding a new provider requires changes in TWO places, defeating the purpose of unification. **This design doc will fix that** by migrating the eval harness to use `internal/ai/`.

**Current gaps:**

1. **Cost**: Cloud API calls for every eval run ($2-5 per benchmark suite)
2. **Privacy**: Code sent to cloud APIs for processing
3. **Offline**: No AI capabilities without internet
4. **Open models**: Can't use Llama, Mistral, DeepSeek, Qwen, etc.
5. **Reproducibility**: Cloud model versions change silently

**Existing Infrastructure:**
- Unified `internal/ai/` package with `Provider` interface (v0.5.10)
- Ollama integration for embeddings (`internal/messaging/embedder.go`)
- `github.com/ollama/ollama/api` package already imported
- Config pattern in `~/.ailang/config.yaml`

## Goals

**Primary Goal:** Add Ollama as a first-class provider in `internal/ai/`, enabling local models across ALL touchpoints.

**Touchpoints that automatically gain Ollama support:**
- ✅ `ailang eval-suite --models ollama:codellama`
- ✅ `ailang run --caps AI --ai ollama:qwen2.5-coder file.ail`
- ✅ AILANG programs using `std/ai` effect
- ✅ Any future AI-powered features

**Success Metrics:**
- [ ] `internal/ai/ollama/` implements `Provider` interface
- [ ] All three touchpoints work with local models
- [ ] Same JSON output format as cloud providers
- [ ] Clear error when Ollama not running

## Solution Design

### Architecture

```
internal/ai/
├── provider.go          # Provider interface (exists)
├── config.go            # ProviderType enum (add ProviderOllama)
├── handler.go           # Effect handler wrapper (exists)
│
├── openai/              # OpenAI (exists)
├── gemini/              # Google (exists)
├── anthropic/           # Anthropic (exists)
│
└── ollama/              # NEW - Local models via Ollama
    ├── client.go        # HTTP client, connection check
    ├── generate.go      # Chat completion (implements Provider)
    ├── handler.go       # implements effects.AIHandler
    └── client_test.go   # Unit tests
```

### Provider Interface Implementation

```go
// internal/ai/ollama/client.go
package ollama

import (
    "context"
    "fmt"
    "os"
    "strings"

    ollamaapi "github.com/ollama/ollama/api"
    "github.com/sunholo/ailang/internal/ai"
)

// Client implements ai.Provider for local Ollama models
type Client struct {
    client   *ollamaapi.Client
    endpoint string
}

// NewClient creates a new Ollama client
func NewClient() (*Client, error) {
    endpoint := os.Getenv("OLLAMA_HOST")
    if endpoint == "" {
        endpoint = "http://localhost:11434"
    }
    os.Setenv("OLLAMA_HOST", endpoint)

    client, err := ollamaapi.ClientFromEnvironment()
    if err != nil {
        return nil, fmt.Errorf("failed to create Ollama client: %w", err)
    }

    return &Client{client: client, endpoint: endpoint}, nil
}

// Generate implements ai.Provider
func (c *Client) Generate(ctx context.Context, req *ai.Request) (*ai.Response, error) {
    var response strings.Builder

    // Use Chat API for instruction following
    err := c.client.Chat(ctx, &ollamaapi.ChatRequest{
        Model: req.Model,
        Messages: []ollamaapi.Message{
            {Role: "system", Content: req.SystemPrompt},
            {Role: "user", Content: req.UserPrompt},
        },
        Options: map[string]interface{}{
            "temperature": req.Temperature,
            "seed":        42, // Deterministic by default
        },
    }, func(resp ollamaapi.ChatResponse) error {
        response.WriteString(resp.Message.Content)
        return nil
    })

    if err != nil {
        return nil, ai.NewProviderError("ollama", 0, err.Error(), err)
    }

    return &ai.Response{
        Text:         response.String(),
        Model:        req.Model,
        InputTokens:  0,  // Ollama doesn't report tokens the same way
        OutputTokens: 0,
    }, nil
}

// Name implements ai.Provider
func (c *Client) Name() string {
    return "ollama"
}

// CheckConnection verifies Ollama is running
func (c *Client) CheckConnection(ctx context.Context) error {
    _, err := c.client.List(ctx)
    if err != nil {
        return fmt.Errorf("Ollama not running at %s: %w\n\n"+
            "To start Ollama:\n"+
            "  1. Install: https://ollama.ai/download\n"+
            "  2. Start:   ollama serve\n"+
            "  3. Pull:    ollama pull codellama:7b",
            c.endpoint, err)
    }
    return nil
}
```

### Integration Points

**1. Add ProviderOllama to config.go:**

```go
// internal/ai/config.go
const (
    ProviderOpenAI    ProviderType = "openai"
    ProviderAnthropic ProviderType = "anthropic"
    ProviderGoogle    ProviderType = "google"
    ProviderOllama    ProviderType = "ollama"  // NEW
)

func GuessProvider(modelName string) ProviderType {
    // Add at top - highest priority for explicit prefix
    if strings.HasPrefix(modelName, "ollama:") {
        return ProviderOllama
    }
    // ... existing logic
}
```

**2. Update CLI ai_handlers.go:**

```go
// cmd/ailang/ai_handlers.go
import "github.com/sunholo/ailang/internal/ai/ollama"

func setupAIHandler(effCtx *effects.EffContext, model string) error {
    provider := ai.GuessProvider(model)

    var handler effects.AIHandler
    switch provider {
    case ai.ProviderOllama:
        modelName := strings.TrimPrefix(model, "ollama:")
        client, err := ollama.NewClient()
        if err != nil {
            return err
        }
        // Check connection before proceeding
        if err := client.CheckConnection(ctx); err != nil {
            return err
        }
        handler = ai.NewHandler(client, modelName)
    // ... existing providers
    }

    effCtx.AI = effects.NewAIContext(handler)
    return nil
}
```

**3. Update eval harness (minimal change):**

The eval harness already uses the unified providers via `ai.GuessProvider()`. Just need to ensure it routes correctly.

**4. Add to models.yml:**

```yaml
  # === LOCAL MODELS (via Ollama) ===

  ollama-codellama:
    api_name: "codellama:7b"
    provider: "ollama"
    description: "CodeLlama 7B - local code generation"
    env_var: ""  # No API key needed
    pricing:
      input_per_1k: 0.0
      output_per_1k: 0.0
    notes: |
      Local model via Ollama. Requires: ollama pull codellama:7b
      Best for: Quick iteration, offline development

  ollama-deepseek-coder:
    api_name: "deepseek-coder:6.7b"
    provider: "ollama"
    description: "DeepSeek Coder 6.7B - competitive with GPT-3.5"
    env_var: ""
    pricing:
      input_per_1k: 0.0
      output_per_1k: 0.0

  ollama-qwen-coder:
    api_name: "qwen2.5-coder:7b"
    provider: "ollama"
    description: "Qwen 2.5 Coder 7B - top-tier open model"
    env_var: ""
    pricing:
      input_per_1k: 0.0
      output_per_1k: 0.0

# Add to suites
local_models:
  - "ollama-codellama"
  - "ollama-deepseek-coder"
  - "ollama-qwen-coder"
```

### Implementation Plan

#### Phase 1: Migrate Eval Harness to Unified Package (~4 hours)

**Critical prerequisite:** Eval harness must use `internal/ai/` before we can add Ollama.

- [ ] Update `internal/eval_harness/ai_agent.go` to use `ai.Provider`
- [ ] Create thin wrapper: `GenerateResult` → `ai.Response`
- [ ] Delete `internal/eval_harness/api_openai.go` (~120 LOC)
- [ ] Delete `internal/eval_harness/api_anthropic.go` (~100 LOC)
- [ ] Delete `internal/eval_harness/api_google.go` (~150 LOC)
- [ ] Run full eval baseline to verify no regression

```go
// internal/eval_harness/ai_agent.go (after migration)
import (
    "github.com/sunholo/ailang/internal/ai"
    "github.com/sunholo/ailang/internal/ai/openai"
    "github.com/sunholo/ailang/internal/ai/anthropic"
    "github.com/sunholo/ailang/internal/ai/gemini"
)

func (a *AIAgent) GenerateCode(ctx context.Context, prompt string) (*GenerateResult, error) {
    provider := a.getProvider()

    resp, err := provider.Generate(ctx, &ai.Request{
        Model:        a.model,
        SystemPrompt: "You are a programming assistant...",
        UserPrompt:   prompt,
    })
    if err != nil {
        return nil, err
    }

    return &GenerateResult{
        Code:         extractCodeFromMarkdown(resp.Text),
        RawResponse:  resp.Text,
        InputTokens:  resp.InputTokens,
        OutputTokens: resp.OutputTokens,
    }, nil
}
```

#### Phase 2: Add Ollama Provider (~3 hours)

- [ ] Create `internal/ai/ollama/client.go` (~80 LOC)
- [ ] Create `internal/ai/ollama/generate.go` (~60 LOC)
- [ ] Create `internal/ai/ollama/client_test.go` (~100 LOC)
- [ ] Add `ProviderOllama` to `internal/ai/config.go` (~10 LOC)
- [ ] Update `GuessProvider()` for `ollama:` prefix

#### Phase 3: Integration (~2 hours)

- [ ] Update `cmd/ailang/ai_handlers.go` to route to Ollama
- [ ] Add local models to `models.yml`
- [ ] Test all three touchpoints:
  - [ ] `ailang run --ai ollama:codellama`
  - [ ] `ailang eval-suite --models ollama:codellama`
  - [ ] AILANG program with `std/ai` effect

#### Phase 4: Polish (~1 hour)

- [ ] Add connection check with helpful error message
- [ ] Add `--suite local_models` flag
- [ ] Update documentation

### Files to Modify

| File | Change | LOC |
|------|--------|-----|
| **Phase 1: Migrate Eval Harness** | | |
| `internal/eval_harness/ai_agent.go` | Use `ai.Provider` interface | ~50 (refactor) |
| `internal/eval_harness/api_openai.go` | DELETE | -120 |
| `internal/eval_harness/api_anthropic.go` | DELETE | -100 |
| `internal/eval_harness/api_google.go` | DELETE | -150 |
| **Phase 2: Add Ollama Provider** | | |
| `internal/ai/ollama/client.go` | NEW - Ollama client | ~80 |
| `internal/ai/ollama/generate.go` | NEW - Provider impl | ~60 |
| `internal/ai/ollama/client_test.go` | NEW - Tests | ~100 |
| `internal/ai/config.go` | Add ProviderOllama | ~15 |
| **Phase 3: Integration** | | |
| `cmd/ailang/ai_handlers.go` | Add Ollama routing | ~20 |
| `internal/eval_harness/models.yml` | Add local models | ~50 |
| **Net Change** | Delete 370 LOC, Add 375 LOC | **~+5 net** |

**Key win:** By migrating eval harness first, we DELETE ~370 LOC of duplicate code while adding ~375 LOC of new functionality. Net code is roughly the same, but architecture is unified.

## Examples

### CLI with Local Model

```bash
# Run AILANG program with local AI
ailang run --caps IO,AI --ai ollama:qwen2.5-coder:7b program.ail

# Dynamic model - any Ollama model works
ailang run --caps AI --ai ollama:llama3.2:3b program.ail
```

### Eval Suite

```bash
# Run benchmarks against local model
ailang eval-suite --models ollama:codellama

# Mix local and cloud
ailang eval-suite --models ollama:deepseek-coder,claude-haiku-4-5

# Use local suite
ailang eval-suite --suite local_models
```

### AILANG Program

```ailang
-- Works with ANY provider now including Ollama
import std/ai as AI

func ask(question: string) -> string ! {AI} =
    AI.call(question)

-- Run with: ailang run --caps AI --ai ollama:codellama --entry main file.ail
```

## Axiom Compliance

| Axiom | Score | Rationale |
|-------|-------|-----------|
| A1: Determinism | +1 | Local models with seed=fixed are reproducible |
| A3: Effect Legibility | 0 | Uses existing AI effect - no change |
| A7: Machines First | +1 | Same JSON format, machine-readable |
| A9: Cost Visibility | +1 | Explicit $0.00 pricing |
| A10: Composability | +1 | Composes with existing Provider interface |

**Net Score: +4** (Accept)

## Success Criteria

- [ ] `internal/ai/ollama/` package implements `Provider` interface
- [ ] `ailang run --ai ollama:codellama` works (CLI)
- [ ] `ailang eval-suite --models ollama:codellama` works (evals)
- [ ] AILANG programs with `std/ai` work with Ollama
- [ ] Same JSON output schema as cloud providers
- [ ] Clear error when Ollama not running
- [ ] All existing tests pass

## Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Ollama API changes | Low | Medium | Pin ollama/api version |
| Model quality variance | High | Low | Document expected quality |
| No token counting | Medium | Low | Return 0, note in docs |
| Streaming differences | Low | Low | Use callback pattern |

## Related Documents

- [M-UNIFIED-AI-PROVIDERS](../implemented/v0_5_10/m-unified-ai-providers.md) - Provider architecture
- [DX-15: Semantic Caching](../implemented/v0_6_0/DX-15-semantic-caching-MVP.md) - Existing Ollama use
- [M-EVAL-AGENT](./M-EVAL-AGENT.md) - Agent-based evaluation

## Future Extensions

1. **Token counting**: Parse Ollama response metrics for accurate counts
2. **Model preloading**: `ailang ollama warmup codellama` before benchmarks
3. **GPU config**: Document CUDA/Metal setup
4. **Streaming**: Add `GenerateStream()` implementation
5. **Tool calling**: Support Ollama's function calling for agents
