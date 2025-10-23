# Multi-Model Agent Support

**AILANG agents are model-agnostic!** The agent protocol system works with any LLM provider.

---

## Supported Providers

### 1. Anthropic Claude ✅

**Models**: Claude Sonnet 4.5, Claude Haiku 4.5, etc.

**Handler**: `AnthropicAgentHandler` / `ClaudeAgentHandler`

**CLI**: Anthropic SDK (Python/TypeScript)

**Example**:
```go
handler := agentrunner.NewAnthropicAgentHandler(
    "claude-sonnet-4-5",
    ".claude/agents/eval-analyzer.md",
    ".",
)
```

**Status**: ✅ Handler implemented (mock execution for now, full SDK integration in v0.3.20)

---

### 2. Google Gemini ✅

**Models**: Gemini 2.5 Pro, Gemini 2.5 Flash, etc.

**Handler**: `GeminiAgentHandler`

**CLI**: `gcloud ai generative-models generate-content`

**Example**:
```go
handler := agentrunner.NewGeminiAgentHandler(
    "gemini-2.5-pro",
    ".claude/agents/eval-analyzer.md",
    ".",
)
```

**Status**: ✅ Handler implemented (requires gcloud CLI)

---

### 3. OpenAI GPT ✅

**Models**: GPT-4, GPT-4 Turbo, GPT-3.5, o1-preview, o1-mini, etc.

**Handler**: `OpenAIAgentHandler`

**CLI**: `openai` CLI or `curl` (fallback)

**Example**:
```go
handler := agentrunner.NewOpenAIAgentHandler(
    "gpt-4",
    ".claude/agents/eval-analyzer.md",
    ".",
)

// Or reasoning models
handler := agentrunner.NewOpenAIAgentHandler(
    "o1-preview",
    ".claude/agents/reasoning-agent.md",
    ".",
)
```

**Status**: ✅ Handler implemented (requires openai CLI or OPENAI_API_KEY)

---

## Multi-Model Router

Use **one agent** that routes to **different models** based on the message:

```go
// Create multi-model handler
multiHandler := agentrunner.NewMultiModelAgentHandler()

// Register Claude
multiHandler.RegisterHandler("claude",
    agentrunner.NewAnthropicAgentHandler("claude-sonnet-4-5", "agent.md", "."))

// Register Gemini
multiHandler.RegisterHandler("gemini",
    agentrunner.NewGeminiAgentHandler("gemini-2.5-pro", "agent.md", "."))

// Register GPT
multiHandler.RegisterHandler("gpt",
    agentrunner.NewOpenAIAgentHandler("gpt-4", "agent.md", "."))

// Set default
multiHandler.RegisterHandler("default",
    agentrunner.NewAnthropicAgentHandler("claude-sonnet-4-5", "agent.md", "."))

// Use in runner
runner, _ := agentrunner.NewRunner(&agentrunner.AgentConfig{
    AgentID: "multi-model-agent",
    Handler: multiHandler,
})
runner.Run()
```

**Send messages with model choice**:
```bash
# Use default (Claude)
./bin/send-message multi-model-agent '{"action": "analyze"}'

# Use Gemini
./bin/send-message multi-model-agent '{"model": "gemini", "action": "analyze"}'

# Use GPT-4
./bin/send-message multi-model-agent '{"model": "gpt", "action": "analyze"}'
```

---

## Use Cases

### 1. Cost Optimization

Use cheap models for simple tasks, expensive models for complex tasks:

```go
multiHandler := agentrunner.NewMultiModelAgentHandler()

// Cheap model for simple tasks
multiHandler.RegisterHandler("simple",
    agentrunner.NewAnthropicAgentHandler("claude-haiku-4-5", "agent.md", "."))

// Expensive model for complex tasks
multiHandler.RegisterHandler("complex",
    agentrunner.NewAnthropicAgentHandler("claude-sonnet-4-5", "agent.md", "."))

// Use in agent logic
func routeByComplexity(msg *agentprotocol.Envelope) string {
    if isComplexTask(msg.Payload) {
        return "complex"
    }
    return "simple"
}
```

---

### 2. Provider Redundancy

Fallback to other providers if one fails:

```go
type RedundantHandler struct {
    Handlers []MessageHandler
}

func (h *RedundantHandler) HandleMessage(msg *agentprotocol.Envelope) (map[string]interface{}, error) {
    for i, handler := range h.Handlers {
        result, err := handler.HandleMessage(msg)
        if err == nil {
            return result, nil
        }
        log.Printf("Handler %d failed: %v, trying next...", i, err)
    }
    return nil, fmt.Errorf("all handlers failed")
}

// Usage
redundant := &RedundantHandler{
    Handlers: []MessageHandler{
        agentrunner.NewAnthropicAgentHandler("claude-sonnet-4-5", "agent.md", "."),
        agentrunner.NewGeminiAgentHandler("gemini-2.5-pro", "agent.md", "."),
        agentrunner.NewOpenAIAgentHandler("gpt-4", "agent.md", "."),
    },
}
```

---

### 3. Benchmark Comparison

Run same task on multiple models and compare:

```go
type BenchmarkHandler struct {
    Handlers map[string]MessageHandler
}

func (h *BenchmarkHandler) HandleMessage(msg *agentprotocol.Envelope) (map[string]interface{}, error) {
    results := make(map[string]interface{})

    for model, handler := range h.Handlers {
        start := time.Now()
        result, err := handler.HandleMessage(msg)
        latency := time.Since(start)

        results[model] = map[string]interface{}{
            "result":  result,
            "error":   err,
            "latency": latency.Milliseconds(),
        }
    }

    return results, nil
}
```

---

### 4. Specialized Models

Route based on task type:

```go
multiHandler := agentrunner.NewMultiModelAgentHandler()

// Code generation: Use OpenAI Codex
multiHandler.RegisterHandler("code",
    agentrunner.NewOpenAIAgentHandler("gpt-4", "code-agent.md", "."))

// Reasoning: Use o1
multiHandler.RegisterHandler("reasoning",
    agentrunner.NewOpenAIAgentHandler("o1-preview", "reasoning-agent.md", "."))

// General: Use Claude
multiHandler.RegisterHandler("general",
    agentrunner.NewAnthropicAgentHandler("claude-sonnet-4-5", "general-agent.md", "."))

// Send with task type
./bin/send-message agent '{"model": "code", "action": "generate", "spec": "..."}'
./bin/send-message agent '{"model": "reasoning", "action": "solve", "problem": "..."}'
```

---

## Setup Requirements

### Anthropic Claude

**Option 1: Anthropic SDK** (recommended, coming in v0.3.20)
```bash
pip install anthropic
export ANTHROPIC_API_KEY=your_key
```

**Option 2: Claude CLI** (current mock)
```bash
# Handler will be updated to use SDK
```

---

### Google Gemini

**Requires gcloud CLI**:
```bash
# Install gcloud
curl https://sdk.cloud.google.com | bash

# Initialize
gcloud init

# Enable AI API
gcloud services enable aiplatform.googleapis.com

# Test
gcloud ai generative-models generate-content \
  --model gemini-2.5-pro \
  --prompt "Hello, Gemini!"
```

---

### OpenAI GPT

**Option 1: OpenAI CLI**:
```bash
pip install openai
export OPENAI_API_KEY=your_key

# Test
openai api chat.completions.create -m gpt-4 -g user "Hello, GPT!"
```

**Option 2: curl** (automatic fallback):
```bash
export OPENAI_API_KEY=your_key
# Handler will automatically use curl if openai CLI not found
```

---

## Performance Comparison

Based on AILANG eval benchmarks (hypothetical):

| Model | Latency (p50) | Cost (per 1M tokens) | Success Rate |
|-------|---------------|----------------------|--------------|
| Claude Sonnet 4.5 | 1.2s | $3.00 | 94% |
| Claude Haiku 4.5 | 0.6s | $0.25 | 87% |
| Gemini 2.5 Pro | 1.5s | $7.00 | 91% |
| Gemini 2.5 Flash | 0.4s | $0.15 | 82% |
| GPT-4 | 2.1s | $30.00 | 89% |
| GPT-4 Turbo | 1.3s | $10.00 | 88% |
| o1-preview | 8.5s | $15.00 | 96% |

**Recommendation**: Use Claude Sonnet 4.5 for best balance of speed/cost/quality.

---

## Example: Eval Analyzer with Multi-Model

```go
package main

import (
    "log"
    "github.com/yourusername/ailang/internal/agentprotocol"
    "github.com/yourusername/ailang/internal/agentrunner"
)

func main() {
    // Create handler that chooses model based on failure complexity
    handler := agentrunner.NewFunctionHandler(func(msg *agentprotocol.Envelope) (map[string]interface{}, error) {
        failures := msg.Payload["failures"].([]interface{})
        complexity := estimateComplexity(failures)

        var model string
        if complexity > 0.8 {
            model = "o1-preview" // Hard problems: use reasoning model
        } else if complexity > 0.5 {
            model = "claude-sonnet-4-5" // Medium: use Claude
        } else {
            model = "gemini-2.5-flash" // Easy: use cheap model
        }

        log.Printf("Complexity: %.2f, using model: %s", complexity, model)

        // Route to appropriate model
        multiHandler := createMultiModelHandler()
        return multiHandler.HandleMessage(&agentprotocol.Envelope{
            Payload: map[string]interface{}{
                "model": model,
                "failures": failures,
            },
        })
    })

    runner, _ := agentrunner.NewRunner(&agentrunner.AgentConfig{
        AgentID: "smart-eval-analyzer",
        Handler: handler,
    })
    runner.Run()
}

func estimateComplexity(failures []interface{}) float64 {
    // Analyze failure patterns, return complexity score 0.0-1.0
    return 0.7
}
```

---

## Future: Local Models

**Planned for v0.4.0**: Support for local models (no API calls)

### Ollama Support

```go
handler := agentrunner.NewOllamaAgentHandler(
    "llama3.1:70b",
    "agent.md",
    ".",
)
```

### LM Studio Support

```go
handler := agentrunner.NewLMStudioAgentHandler(
    "local-model",
    "agent.md",
    ".",
)
```

### Custom HTTP Endpoints

```go
handler := agentrunner.NewHTTPAgentHandler(
    "http://localhost:8080/v1/chat",
    "agent.md",
    ".",
)
```

---

## Extending with Your Own Model

**Implement the `MessageHandler` interface**:

```go
type MyCustomHandler struct {
    ModelEndpoint string
    AgentFile     string
}

func (h *MyCustomHandler) HandleMessage(msg *agentprotocol.Envelope) (map[string]interface{}, error) {
    // 1. Build prompt from agent file + message
    prompt := buildPrompt(h.AgentFile, msg)

    // 2. Call your model API
    response := callMyModel(h.ModelEndpoint, prompt)

    // 3. Return structured response
    return map[string]interface{}{
        "status": "completed",
        "output": response,
    }, nil
}

// Use it
runner, _ := agentrunner.NewRunner(&agentrunner.AgentConfig{
    AgentID: "custom-agent",
    Handler: &MyCustomHandler{
        ModelEndpoint: "https://my-model.com/api",
        AgentFile:     "agent.md",
    },
})
```

---

## Demo

See [examples/agents/multi_model_agent.go](../examples/agents/multi_model_agent.go) for complete working example.

**Run it**:
```bash
# Build
go build -o bin/multi-model-agent examples/agents/multi_model_agent.go

# Start
./bin/multi-model-agent

# Send messages with different models
./bin/send-message multi-model-agent '{"action": "analyze"}'
./bin/send-message multi-model-agent '{"model": "gemini", "action": "analyze"}'
./bin/send-message multi-model-agent '{"model": "gpt-4", "action": "analyze"}'
./bin/send-message multi-model-agent '{"model": "o1", "action": "reason", "problem": "..."}'
```

---

**Last Updated**: October 23, 2025
**Version**: v0.3.19 (Unreleased)
