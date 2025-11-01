# Multi-Provider Agent Eval Setup

**For M-EVAL-AGENT Phase 2+ (v0.4.0+)**

This document covers setup requirements for running agent evals across Claude Code, Gemini CLI, and OpenAI Codex.

---

## Provider Comparison (2025)

| Feature | Claude Code | Gemini CLI | OpenAI Codex |
|---------|-------------|------------|--------------|
| **Status** | ✅ Phase 1 (v0.4.0) | 🔜 Phase 2 (v0.4.1+) | 🔜 Phase 2 (v0.4.1+) |
| **CLI Command** | `claude` | `gemini` | `codex` |
| **Headless Mode** | `claude -p` | `gemini -p` or `gemini -b` | `codex exec --quiet` |
| **JSON Output** | `--output-format json` | `--output-format json` | `--quiet` (returns JSON) |
| **Installation** | npm global | npm global | Rust binary |
| **Built-in Tools** | Bash, Read, Write, Edit, Grep | Search, Files, Shell, Web | Read, Write, Execute (sandboxed) |
| **Context Window** | 200k tokens | 1M tokens | ~128k tokens (GPT-5-Codex) |
| **Model** | Claude Sonnet 4.5 | Gemini 2.5 Pro | GPT-5-Codex |
| **Pricing** | ~$3/$15 per M tokens | ~$1.25/$5 per M tokens | ~$5/$15 per M tokens |
| **Open Source** | ❌ | ❌ | ✅ (CLI is open source) |

---

## 1. Claude Code (Phase 1 - CURRENT)

**Status**: ✅ Implemented in M-EVAL-AGENT Phase 1

### Installation

```bash
# Global install
npm install -g @anthropic-ai/claude-code

# Verify
claude --version
```

### Headless Execution

```bash
claude -p "Solve this AILANG benchmark" \
  --output-format json \
  --allowedTools "Bash,Read,Write,Edit,Grep"
```

### JSON Output Format

```json
{
  "type": "result",
  "subtype": "success",
  "total_cost_usd": 0.15,
  "duration_ms": 45000,
  "num_turns": 3,
  "session_id": "abc123",
  "result": "..."
}
```

### Available Metrics

- ✅ Cost (`total_cost_usd`)
- ✅ Duration (`duration_ms`)
- ✅ Turns/iterations (`num_turns`)
- ✅ Success (`subtype`)
- ❌ Token counts (not exposed)
- ❌ Tool usage (not exposed)

**See**: [CLAUDE_CODE_SETUP.md](CLAUDE_CODE_SETUP.md) for full setup guide

---

## 2. Gemini CLI (Phase 2 - PLANNED)

**Status**: 🔜 Planned for v0.4.1+

### Installation

```bash
# Global install
npm install -g @google-gemini/gemini-cli

# Authenticate (one-time)
gemini auth login

# Verify
gemini --version
```

### Headless Execution

```bash
# Pipeline mode (interactive permission prompts)
gemini -p "Solve this AILANG benchmark" \
  --output-format json

# Full headless mode (auto-approve in working directory)
gemini -b "Solve this AILANG benchmark" \
  --output-format json \
  --yolo
```

**Flags**:
- `-p` or `--pipeline`: Pipeline mode (headless with prompts)
- `-b` or `--batch`: Full headless mode
- `--yolo`: Auto-approve actions in working directory
- `--output-format json`: Structured JSON output
- `--output-format stream-json`: Real-time streaming JSON

### JSON Output Format

```json
{
  "type": "result",
  "status": "success",
  "token_usage": {
    "input_tokens": 8500,
    "output_tokens": 9500,
    "total_tokens": 18000
  },
  "tool_calls": [
    {"tool": "shell", "count": 5},
    {"tool": "file_read", "count": 4},
    {"tool": "file_write", "count": 2}
  ],
  "model": "gemini-2.5-pro",
  "result": "..."
}
```

### Available Metrics

- ✅ Token counts (input/output/total) - **BETTER than Claude!**
- ✅ Tool usage details - **BETTER than Claude!**
- ✅ Cost (can be calculated from tokens)
- ✅ Duration (track externally)
- ⚠️  Iterations (may need to infer from tool_calls)

### Authentication Considerations

**Issue**: Gemini CLI requires browser authentication, which doesn't work on headless servers.

**Solutions**:

1. **Pre-authenticate on local machine**:
   ```bash
   gemini auth login
   # Saves credentials to ~/.gemini/credentials.json
   # Copy this file to headless server
   ```

2. **Use service account** (for CI/CD):
   ```bash
   export GOOGLE_APPLICATION_CREDENTIALS=/path/to/service-account.json
   gemini -p "task" --output-format json
   ```

3. **API key authentication** (if supported):
   ```bash
   export GEMINI_API_KEY=your-api-key
   gemini -p "task" --output-format json
   ```

**For eval harness**: Pre-authenticate during setup, credentials persist.

### Built-in Tools

- **Google Search**: Web search with grounding
- **File operations**: Read/write files
- **Shell commands**: Execute bash commands
- **Web fetching**: HTTP requests
- **MCP support**: Model Context Protocol for custom tools

### Context Window

- **1M tokens**: Largest context window of the three providers
- Great for complex benchmarks with large codebases

---

## 3. OpenAI Codex (Phase 2 - PLANNED)

**Status**: 🔜 Planned for v0.4.1+

### Installation

**Codex CLI is written in Rust** - different from npm packages!

```bash
# Option 1: Install from binary release
# Download from: https://github.com/openai/codex/releases
curl -L https://github.com/openai/codex/releases/latest/download/codex-$(uname -s)-$(uname -m) -o codex
chmod +x codex
sudo mv codex /usr/local/bin/

# Option 2: Build from source (requires Rust)
git clone https://github.com/openai/codex
cd codex
cargo build --release
sudo cp target/release/codex /usr/local/bin/

# Verify
codex --version
```

### Authentication

```bash
# Set OpenAI API key
export OPENAI_API_KEY=your-api-key

# Or configure via CLI
codex config set api-key your-api-key
```

### Headless Execution

```bash
# Headless mode for well-scoped tasks
codex exec --quiet "Solve this AILANG benchmark"

# Auto-approval mode (automatically approves actions in working directory)
codex run --auto-approve "Solve this AILANG benchmark"
```

**Flags**:
- `exec`: Execute a task
- `--quiet`: Headless mode (JSON output)
- `--auto-approve`: Skip permission prompts
- `--workspace`: Set working directory

### JSON Output Format

```json
{
  "status": "success",
  "result": "...",
  "execution_time_ms": 45000,
  "commands_executed": 5,
  "files_modified": 2,
  "model": "gpt-5-codex"
}
```

### Available Metrics

- ✅ Success status
- ✅ Execution time
- ✅ Commands executed (proxy for iterations)
- ✅ Files modified
- ⚠️  Cost (need to track API usage separately)
- ⚠️  Token counts (may not be exposed)

### Security: Sandboxed Execution

Codex CLI uses platform-specific sandboxing:
- **macOS**: Seatbelt
- **Linux**: Landlock + seccomp

**Approval modes**:
- **Default**: Prompts for approval for any action
- **Auto-approve**: Automatically approves actions in working directory
- **Network**: Requires explicit approval for network access

**For eval harness**: Use `--auto-approve` in isolated workspace per benchmark.

### Built-in Capabilities

- Read/write files in working directory
- Execute shell commands (sandboxed)
- Code analysis and modification
- **Local execution**: Code never leaves your machine

### Model

- **GPT-5-Codex**: Optimized for agentic coding tasks
- Built on GPT-5, specialized for code generation and execution
- Supports multi-turn iteration and planning

---

## Implementation Strategy

### Phase 1: Claude Code (v0.4.0) ✅

**Current priority**: Get Claude Code working end-to-end

- Eval harness uses `claude -p`
- Collect metrics: cost, duration, num_turns
- Update dashboard with agent tier
- Prove the concept works

### Phase 2: Add Gemini + Codex (v0.4.1+)

**After Claude is proven**, add other providers:

```go
// internal/eval_harness/provider_interface.go

type AgentProvider interface {
    Name() string
    Detect() error // Check if installed
    RunBenchmark(task BenchmarkTask) (*BenchmarkResult, error)
}

// Implementations:
type ClaudeCodeProvider struct { ... }  // ✅ v0.4.0
type GeminiCLIProvider struct { ... }   // 🔜 v0.4.1
type OpenAICodexProvider struct { ... } // 🔜 v0.4.1
```

### CLI Integration

```bash
# Phase 1: Claude only
ailang eval-suite --agent --provider claude-code

# Phase 2: Multi-provider
ailang eval-suite --agent --provider gemini
ailang eval-suite --agent --provider codex

# Run all providers
ailang eval-suite --agent --providers claude-code,gemini,codex

# Dashboard compares all three
ailang eval-report eval_results/agent/v0.4.1 v0.4.1 --compare-providers
```

---

## Provider Detection & Fallback

```go
// internal/eval_harness/provider_detect.go

func DetectAvailableProviders() []AgentProvider {
    providers := []AgentProvider{}

    // Try Claude Code
    if _, err := exec.LookPath("claude"); err == nil {
        providers = append(providers, &ClaudeCodeProvider{})
    }

    // Try Gemini CLI
    if _, err := exec.LookPath("gemini"); err == nil {
        providers = append(providers, &GeminiCLIProvider{})
    }

    // Try OpenAI Codex
    if _, err := exec.LookPath("codex"); err == nil {
        if os.Getenv("OPENAI_API_KEY") != "" {
            providers = append(providers, &OpenAICodexProvider{})
        }
    }

    return providers
}

// Usage:
providers := DetectAvailableProviders()
if len(providers) == 0 {
    return fmt.Errorf("No agent providers found. Install at least one:\n" +
        "  Claude Code: npm install -g @anthropic-ai/claude-code\n" +
        "  Gemini CLI:  npm install -g @google-gemini/gemini-cli\n" +
        "  OpenAI Codex: See docs/MULTI_PROVIDER_AGENT_SETUP.md")
}
```

---

## Metric Normalization

Different providers expose different metrics. Normalize to common format:

```go
type NormalizedBenchmarkResult struct {
    BenchmarkID     string
    Provider        string // "claude-code", "gemini", "codex"
    Success         bool

    // Always available (normalized)
    Cost            float64 // Calculated or provided
    Duration        float64 // Seconds
    Iterations      int     // num_turns, tool_calls, or commands_executed

    // Provider-specific (optional)
    InputTokens     *int    // Gemini provides, Claude doesn't
    OutputTokens    *int
    ToolCalls       map[string]int // Gemini provides, others don't

    // Raw provider response
    RawOutput       json.RawMessage
}
```

### Cost Calculation

```go
func CalculateCost(provider string, tokens *TokenUsage) float64 {
    switch provider {
    case "claude-code":
        // Already provided in JSON
        return result.TotalCostUSD

    case "gemini":
        // Calculate from tokens: $1.25/$5 per M tokens
        inputCost := float64(tokens.Input) * 0.00000125
        outputCost := float64(tokens.Output) * 0.000005
        return inputCost + outputCost

    case "codex":
        // Query OpenAI API for usage
        return queryOpenAICost(sessionID)
    }
}
```

---

## Dashboard: Multi-Provider Comparison

```markdown
## AILANG v0.4.1 Agent Tier Comparison

### Success Rate by Provider

| Provider | 0-Shot | 1-Repair | Agent | Improvement |
|----------|--------|----------|-------|-------------|
| Claude Sonnet 4.5 | 65% | 78% | **92%** | +14% |
| Gemini 2.5 Pro | 68% | 80% | **90%** | +10% |
| GPT-5 Codex | 67% | 79% | **91%** | +12% |

### Cost Efficiency

| Provider | Avg Cost | Success | Cost per Success |
|----------|----------|---------|------------------|
| Gemini 2.5 Pro | $0.08 | 90% | $0.089 | ← **Best value**
| Claude Sonnet 4.5 | $0.15 | 92% | $0.163 |
| GPT-5 Codex | $0.19 | 91% | $0.209 |

### Iteration Efficiency

| Provider | Avg Iterations | Fastest Solve |
|----------|----------------|---------------|
| Claude Sonnet 4.5 | 3.5 | 1.2 min | ← **Fastest**
| GPT-5 Codex | 3.8 | 1.4 min |
| Gemini 2.5 Pro | 4.1 | 1.6 min |

**Key Insight**: All three providers show significant improvement over 1-repair tier, validating AILANG's AI-first design!
```

---

## Setup Checklist (Future)

### Claude Code (Phase 1)
- [x] Install: `npm install -g @anthropic-ai/claude-code`
- [x] Test: `make test-claude-headless`
- [x] Configure: No auth needed (uses Claude API)

### Gemini CLI (Phase 2)
- [ ] Install: `npm install -g @google-gemini/gemini-cli`
- [ ] Authenticate: `gemini auth login`
- [ ] Test: `gemini -p "echo test" --output-format json`
- [ ] Configure: Credentials in `~/.gemini/credentials.json`

### OpenAI Codex (Phase 2)
- [ ] Install: Download from GitHub releases
- [ ] Set API key: `export OPENAI_API_KEY=...`
- [ ] Test: `codex exec --quiet "echo test"`
- [ ] Configure: API key in `~/.codex/config.toml`

---

## References

**Claude Code**:
- Docs: https://docs.claude.com/en/docs/claude-code/headless
- Setup: [CLAUDE_CODE_SETUP.md](CLAUDE_CODE_SETUP.md)

**Gemini CLI**:
- Docs: https://geminicli.com/docs/cli/headless/
- GitHub: https://github.com/google-gemini/gemini-cli
- Code Execution: https://ai.google.dev/gemini-api/docs/code-execution

**OpenAI Codex**:
- Docs: https://developers.openai.com/codex/cli/
- GitHub: https://github.com/openai/codex
- Blog: https://openai.com/index/introducing-codex/

---

**Created**: October 27, 2025
**Target**: M-EVAL-AGENT Phase 2+ (v0.4.1+)
**Priority**: After Claude Code Phase 1 is proven working
