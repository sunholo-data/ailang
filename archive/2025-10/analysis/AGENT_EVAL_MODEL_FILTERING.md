# Agent Eval Model Filtering - Implementation Summary

**Date**: 2025-10-28
**Issue**: Agent evals were mislabeling results and trying to use unsupported models
**Status**: ✅ **FIXED**

---

## Problem Statement

When running `ailang eval-suite --agent --models gpt5-mini`, the system would:
1. Run agent eval with Claude Haiku (correct behavior)
2. Label the result as `gpt5-mini` (WRONG - confusing!)
3. Not skip unsupported models (OpenAI, Gemini)

**Root cause**: No mechanism to indicate which models support agent CLI evaluation.

---

## Solution: agent_cli Field in models.yml

### 1. Updated models.yml Schema

Added two new fields to each model:
- `agent_cli`: CLI command for agent eval (`"claude"`, `null`, etc.)
- `agent_model_name`: Model name to pass to CLI (`"haiku"`, `"sonnet"`)

**Example:**
```yaml
models:
  claude-haiku-4-5:
    api_name: "claude-haiku-4-5-20251001"
    provider: "anthropic"
    agent_cli: "claude"           # ← NEW
    agent_model_name: "haiku"      # ← NEW
    pricing: {...}

  gpt5-mini:
    api_name: "gpt-5-mini"
    provider: "openai"
    agent_cli: null                # ← Not supported yet
    pricing: {...}
```

### 2. Updated Model Config Structure

**File**: `internal/eval_harness/models.go`

```go
type ModelConfig struct {
    APIName        string  `yaml:"api_name"`
    Provider       string  `yaml:"provider"`
    // ... existing fields ...
    AgentCLI       *string `yaml:"agent_cli"`        // NEW
    AgentModelName *string `yaml:"agent_model_name"` // NEW
    Pricing        Pricing `yaml:"pricing"`
}
```

### 3. Added Helper Functions

**New methods on `ModelsConfig`:**

```go
// Check if model supports agent eval
func (c *ModelsConfig) SupportsAgentEval(name string) bool

// Get CLI command (e.g., "claude")
func (c *ModelsConfig) GetAgentCLI(name string) (string, error)

// Get model name for CLI (e.g., "haiku")
func (c *ModelsConfig) GetAgentModelName(name string) (string, error)

// Filter model list to only supported models
func (c *ModelsConfig) FilterAgentSupportedModels(models []string) []string
```

### 4. Updated eval-suite Command

**File**: `cmd/ailang/eval_suite.go`

**Added automatic filtering in agent mode:**

```go
if *agent {
    // Filter to only models that support agent eval
    originalModels := modelList
    modelList = eval_harness.GlobalModelsConfig.FilterAgentSupportedModels(modelList)

    // Warn about skipped models
    if len(modelList) < len(originalModels) {
        skipped := []string{}
        for _, model := range originalModels {
            if !eval_harness.GlobalModelsConfig.SupportsAgentEval(model) {
                skipped = append(skipped, model)
            }
        }
        fmt.Fprintf(os.Stderr, "⚠️  Agent mode: Skipping %d unsupported model(s): %v\n",
            len(skipped), skipped)
        fmt.Fprintf(os.Stderr, "   These models require CLI integration (not yet implemented)\n")
        fmt.Fprintf(os.Stderr, "   Only Claude models support agent eval currently\n")
    }
}
```

---

## Behavior Changes

### Before

```bash
ailang eval-suite --agent --models gpt5-mini,claude-haiku-4-5 --benchmarks fizzbuzz
# ❌ Runs agent with both models
# ❌ Results labeled as "gpt5-mini" but agent used Claude
# ❌ No warning about unsupported models
```

### After

```bash
ailang eval-suite --agent --models gpt5-mini,claude-haiku-4-5 --benchmarks fizzbuzz
# ✅ Automatically filters to [claude-haiku-4-5]
# ✅ Warning: "Skipping 1 unsupported model(s): [gpt5-mini]"
# ✅ Results correctly labeled with actual agent model used
# ✅ Agent model name pulled from models.yml (no --agent-model flag needed)
```

---

## Current Agent Support Status

| Model | Agent CLI | Status | Notes |
|-------|-----------|--------|-------|
| **claude-sonnet-4-5** | `claude` | ✅ Supported | agent_model_name: "sonnet" |
| **claude-haiku-4-5** | `claude` | ✅ Supported | agent_model_name: "haiku" |
| **gpt5** | `null` | ❌ Not yet | Requires OpenAI Codex CLI |
| **gpt5-mini** | `null` | ❌ Not yet | Requires OpenAI Codex CLI |
| **gemini-2-5-pro** | `null` | ❌ Not yet | Requires Gemini CLI |
| **gemini-2-5-flash** | `null` | ❌ Not yet | Requires Gemini CLI |

---

## Testing

### Test 1: Mixed Models (Supported + Unsupported)

```bash
ailang eval-suite \
  --agent \
  --benchmarks fizzbuzz \
  --models gpt5-mini,claude-haiku-4-5,gemini-2-5-flash \
  --langs ailang
```

**Expected output:**
```
⚠️  Agent mode: Skipping 2 unsupported model(s): [gpt5-mini gemini-2-5-flash]
   These models require CLI integration (not yet implemented)
   Only Claude models support agent eval currently

Models:     [claude-haiku-4-5]
...
```

### Test 2: Only Unsupported Models

```bash
ailang eval-suite \
  --agent \
  --benchmarks fizzbuzz \
  --models gpt5-mini
```

**Expected output:**
```
Error: No models support agent evaluation
Agent mode currently only supports Claude models (claude-sonnet-4-5, claude-haiku-4-5)

Example:
  ailang eval-suite --agent --models claude-haiku-4-5 --benchmarks fizzbuzz
```

### Test 3: Only Supported Models

```bash
ailang eval-suite \
  --agent \
  --benchmarks fizzbuzz \
  --models claude-haiku-4-5 \
  --langs ailang
```

**Expected output:**
```
🤖 Agent mode ENABLED (Claude Code)
  - Models: [claude-haiku-4-5]
  - Agent CLI model: haiku
  ...
```

---

## Files Modified

1. `internal/eval_harness/models.yml` (+6 lines per model)
   - Added `agent_cli` and `agent_model_name` fields

2. `internal/eval_harness/models.go` (+54 lines)
   - Updated `ModelConfig` struct
   - Added 4 new helper methods

3. `cmd/ailang/eval_suite.go` (+40 lines)
   - Added model filtering in agent mode
   - Added warning messages for skipped models
   - Auto-detect agent model name from models.yml

---

## Future Work

### When Adding New Agent CLI Support

**Example: Adding OpenAI Codex CLI support**

1. **Update models.yml:**
```yaml
gpt5:
  agent_cli: "openai"           # CLI command
  agent_model_name: "gpt-5"     # Model name for CLI
```

2. **No code changes needed!** The filtering will automatically work.

3. **(Optional) Add CLI handler** in `internal/agentrunner/llm_cli_handler.go`:
```go
func NewOpenAICLIHandler(model, agentFile, workDir string) *LLMCLIHandler {
    return NewLLMCLIHandler(&LLMCLIConfig{
        CLICommand: "openai",
        Model:      model,
        ArgsTemplate: []string{"--prompt", "{{prompt}}", "--model", "{{model}}"},
        ...
    })
}
```

---

## Migration Notes

### Backward Compatibility

- `--agent-model` flag still works (for manual override)
- If not provided, auto-detects from first model in `--models` list
- Falls back to "haiku" if detection fails

### Breaking Changes

**None!** This is a purely additive change.

---

## Related Documentation

- [M-AGENT-PROTOCOL Audit](design_docs/archive/M-AGENT-PROTOCOL-AUDIT.md)
- [Agent Eval Verification](AGENT_EVAL_VERIFICATION.md)
- [models.yml](internal/eval_harness/models.yml)

---

**Implemented by:** Claude (Sonnet 4.5)
**Date:** 2025-10-28
**Status:** ✅ Complete and tested
