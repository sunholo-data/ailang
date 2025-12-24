# M-EVAL-OLLAMA: Local Model Evaluation via Ollama

**Status:** Planned
**Target:** v0.6.2
**Priority:** P1 (Medium)
**Estimated:** 2 days
**Dependencies:** None (Ollama already integrated for embeddings)
**Created:** 2024-12-24

## Problem Statement

Currently, the AILANG eval harness only supports cloud-based LLMs (OpenAI, Anthropic, Google). This has several limitations:

1. **Cost**: Running benchmarks across all 46 AILANG tasks costs $2-5 per run, limiting iteration speed
2. **Privacy**: Some users want to evaluate code generation without sending code to cloud APIs
3. **Offline development**: Can't run evals without internet connectivity
4. **New model testing**: Can't quickly test open-weight models (Llama, Mistral, CodeLlama, DeepSeek, etc.)
5. **Reproducibility**: Cloud model versions change; local models are fully deterministic

**Existing Infrastructure:**
- Ollama already integrated for semantic caching (`internal/messaging/embedder.go`)
- `_ollama_embed` builtin works (`internal/builtins/ollama_embed.go`)
- Uses official `github.com/ollama/ollama/api` package
- Configuration pattern exists in `~/.ailang/config.yaml`

## Goals

**Primary Goal:** Enable running AILANG eval benchmarks against local Ollama models with zero additional dependencies.

**Success Metrics:**
- [ ] `ailang eval-suite --models ollama:codellama` works end-to-end
- [ ] 100% parity with cloud model output format (same JSON schema)
- [ ] <5% overhead vs direct Ollama API calls
- [ ] Clear error messages when Ollama isn't running

## Solution Design

### Overview

Add `ollama` as a new provider in the eval harness, reusing the existing Ollama client infrastructure from embeddings.

### Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    eval_harness/                            │
├─────────────────────────────────────────────────────────────┤
│  ai_agent.go         - Add callOllama() method              │
│  api_ollama.go (NEW) - Ollama API implementation            │
│  models.go           - Add "ollama" provider support        │
│  models.yml          - Add local model definitions          │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│  github.com/ollama/ollama/api (already imported)            │
│  - api.ClientFromEnvironment()                              │
│  - client.Chat() / client.Generate()                        │
└─────────────────────────────────────────────────────────────┘
```

### Implementation Plan

#### Phase 1: Core Ollama Provider (~4 hours)

**1.1 Create `internal/eval_harness/api_ollama.go`**

```go
package eval_harness

import (
    "context"
    "fmt"
    "os"
    "strings"

    "github.com/ollama/ollama/api"
)

// callOllama makes a request to a local Ollama model
func (a *AIAgent) callOllama(ctx context.Context, prompt string) (*GenerateResult, error) {
    // Use OLLAMA_HOST from env or default
    if endpoint := os.Getenv("OLLAMA_HOST"); endpoint == "" {
        os.Setenv("OLLAMA_HOST", "http://localhost:11434")
    }

    client, err := api.ClientFromEnvironment()
    if err != nil {
        return nil, fmt.Errorf("failed to create Ollama client: %w", err)
    }

    // Extract model name from "ollama:modelname" format
    modelName := strings.TrimPrefix(a.model, "ollama:")

    // Use Chat API for better instruction following
    var response strings.Builder
    err = client.Chat(ctx, &api.ChatRequest{
        Model: modelName,
        Messages: []api.Message{
            {
                Role:    "system",
                Content: "You are a programming assistant. Generate ONLY code without explanations or markdown formatting.",
            },
            {
                Role:    "user",
                Content: prompt,
            },
        },
        Options: map[string]interface{}{
            "temperature": 0.0,  // Deterministic for reproducibility
            "seed":        a.seed,
        },
    }, func(resp api.ChatResponse) error {
        response.WriteString(resp.Message.Content)
        return nil
    })

    if err != nil {
        return nil, fmt.Errorf("Ollama chat failed: %w", err)
    }

    // Extract code from response
    code := extractCode(response.String())

    return &GenerateResult{
        Code:         code,
        RawResponse:  response.String(),
        InputTokens:  0,  // Ollama doesn't report tokens in same way
        OutputTokens: 0,
    }, nil
}
```

**1.2 Update `ai_agent.go` to route to Ollama**

```go
// In Generate() method, add:
case "ollama":
    return a.callOllama(ctx, prompt)
```

**1.3 Update `models.go` to handle dynamic Ollama models**

```go
// Add to guessProvider():
case "oll":
    return "ollama"

// Add to ResolveModelName():
if strings.HasPrefix(name, "ollama:") {
    modelName := strings.TrimPrefix(name, "ollama:")
    return modelName, "ollama", nil
}
```

#### Phase 2: Model Configuration (~2 hours)

**2.1 Add local models to `models.yml`**

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
      Local model via Ollama. Requires: ollama run codellama:7b
      Best for: Quick iteration, offline development

  ollama-deepseek-coder:
    api_name: "deepseek-coder:6.7b"
    provider: "ollama"
    description: "DeepSeek Coder 6.7B - strong code model"
    env_var: ""
    pricing:
      input_per_1k: 0.0
      output_per_1k: 0.0
    notes: |
      Local model via Ollama. Requires: ollama run deepseek-coder:6.7b
      Competitive with GPT-3.5 on coding benchmarks.

  ollama-qwen-coder:
    api_name: "qwen2.5-coder:7b"
    provider: "ollama"
    description: "Qwen 2.5 Coder 7B - excellent for code"
    env_var: ""
    pricing:
      input_per_1k: 0.0
      output_per_1k: 0.0
    notes: |
      Local model via Ollama. Requires: ollama run qwen2.5-coder:7b
      Top-tier open model for code generation.

# Add local model suite
local_models:
  - "ollama-codellama"
  - "ollama-deepseek-coder"
  - "ollama-qwen-coder"
```

**2.2 Support dynamic `ollama:modelname` syntax**

Allow any Ollama model without pre-configuration:

```bash
# Pre-configured (in models.yml)
ailang eval-suite --models ollama-codellama

# Dynamic (any model Ollama has pulled)
ailang eval-suite --models ollama:mistral:7b
ailang eval-suite --models ollama:llama3.2:3b
ailang eval-suite --models ollama:phi3:mini
```

#### Phase 3: Testing & Polish (~2 hours)

**3.1 Add connection check**

```go
// CheckOllamaConnection verifies Ollama is running
func CheckOllamaConnection() error {
    client, err := api.ClientFromEnvironment()
    if err != nil {
        return fmt.Errorf("cannot create Ollama client: %w", err)
    }

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    _, err = client.List(ctx)
    if err != nil {
        return fmt.Errorf("Ollama not running at %s: %w",
            os.Getenv("OLLAMA_HOST"), err)
    }
    return nil
}
```

**3.2 Add helpful error messages**

```
Error: Ollama not running at http://localhost:11434

To start Ollama:
  1. Install: https://ollama.ai/download
  2. Start:   ollama serve
  3. Pull model: ollama pull codellama:7b
  4. Retry:   ailang eval-suite --models ollama:codellama
```

**3.3 Unit tests**

- `api_ollama_test.go` - Mock Ollama responses
- Integration test (skipped if Ollama not running)

### Files to Modify

| File | Change | LOC |
|------|--------|-----|
| `internal/eval_harness/api_ollama.go` | NEW - Ollama provider | ~120 |
| `internal/eval_harness/ai_agent.go` | Add callOllama routing | ~10 |
| `internal/eval_harness/models.go` | Add ollama: prefix handling | ~15 |
| `internal/eval_harness/models.yml` | Add local model definitions | ~60 |
| `internal/eval_harness/api_ollama_test.go` | NEW - Unit tests | ~100 |
| **Total** | | **~305** |

## Examples

### Running Local Eval

```bash
# Ensure Ollama is running
ollama serve &

# Pull a code model
ollama pull qwen2.5-coder:7b

# Run eval against local model
ailang eval-suite --models ollama:qwen2.5-coder:7b --output eval_results/local

# Compare local vs cloud
ailang eval-compare eval_results/local eval_results/baselines/v0.6.1
```

### Mixed Local + Cloud

```bash
# Compare local model against cloud baselines
ailang eval-suite --models ollama:codellama,claude-haiku-4-5
```

### Offline Development

```bash
# No internet needed after model is pulled
export OLLAMA_HOST=http://localhost:11434
ailang eval-suite --models ollama:deepseek-coder:6.7b
```

## Axiom Compliance

| Axiom | Score | Rationale |
|-------|-------|-----------|
| A1: Determinism | +1 | Local models with seed=fixed are fully reproducible |
| A2: Replayability | +1 | Traces work identically for local models |
| A3: Effect Legibility | 0 | No change to effect system |
| A7: Machines First | +1 | Same JSON output format, machine-readable |
| A9: Cost Visibility | +1 | Explicit $0.00 pricing for local models |

**Net Score: +4** (Accept)

## Success Criteria

- [ ] `ailang eval-suite --models ollama:codellama` completes successfully
- [ ] Results JSON has same schema as cloud model results
- [ ] Clear error when Ollama not running
- [ ] Can mix local and cloud models in same run
- [ ] Documentation updated with local model setup guide
- [ ] All existing tests still pass

## Timeline

| Day | Tasks |
|-----|-------|
| 1 | Phase 1 (api_ollama.go, routing) + Phase 2 (models.yml) |
| 2 | Phase 3 (testing, error messages, docs) |

## Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Ollama API changes | Low | Medium | Pin to specific ollama/api version in go.mod |
| Model quality variance | High | Low | Document expected quality; users choose models |
| Response format differences | Medium | Medium | Robust code extraction with fallbacks |

## Related Documents

- [DX-15: Semantic Caching MVP](../implemented/v0_6_0/DX-15-semantic-caching-MVP.md) - Ollama integration for embeddings
- [M-EVAL-AGENT](./M-EVAL-AGENT.md) - Agent-based evaluation (different approach)
- [models.yml](../../internal/eval_harness/models.yml) - Current model configuration

## Future Extensions

1. **GPU acceleration**: Document CUDA/Metal setup for faster inference
2. **Model comparison dashboard**: Add local models to benchmark dashboard
3. **Quantization options**: Support different quantization levels (Q4, Q8, FP16)
4. **Batch inference**: Use Ollama's batch API for faster multi-benchmark runs
