# Claude Code Headless Setup Guide

**For M-EVAL-AGENT and M-CLAUDE-CODE-HEADLESS implementations**

## Problem: Multiple Claude Installations

You may have:
1. ✅ **VSCode Extension**: Claude Code integrated in VSCode (what you currently have)
2. ❌ **Standalone CLI**: `claude` command for headless execution (what we need)

**The VSCode extension and standalone CLI are different!** The extension can't be invoked headlessly.

---

## Solution: Install Standalone Claude CLI

### Option 1: Global NPM Install (Recommended)

```bash
# Install globally
npm install -g @anthropic-ai/claude-code

# Verify installation
claude --version

# Test headless mode
claude -p "echo hello" --output-format json
```

**If you need specific node version:**
```bash
# Use nvm to switch to node 18+ (required)
nvm use 18
npm install -g @anthropic-ai/claude-code

# Verify it works
claude --version
```

**Add to your shell profile** (to avoid nvm switching):
```bash
# Add to ~/.zshrc or ~/.bashrc:
export PATH="$HOME/.nvm/versions/node/v18.x.x/bin:$PATH"

# Or create alias:
alias claude="nvm exec 18 claude"
```

### Option 2: Project-Local Install

```bash
# Install in ailang project
cd /path/to/ailang
npm install @anthropic-ai/claude-code

# Use with npx
npx claude -p "echo hello" --output-format json
```

### Option 3: Docker (Most Robust)

```bash
# Create Dockerfile for headless Claude
cat > Dockerfile.claude <<'EOF'
FROM node:18-alpine
RUN npm install -g @anthropic-ai/claude-code
ENTRYPOINT ["claude"]
EOF

# Build image
docker build -t ailang-claude -f Dockerfile.claude .

# Use in eval harness
docker run --rm -v /tmp:/tmp ailang-claude -p "task" --output-format json
```

---

## Verification

### Test 1: Basic Headless Command

```bash
claude -p "List files in current directory" \
  --output-format json \
  --allowedTools "Bash"
```

**Expected output:**
```json
{
  "type": "result",
  "subtype": "success",
  "total_cost_usd": 0.001,
  "duration_ms": 1234,
  "num_turns": 2,
  "result": "...",
  "session_id": "abc123"
}
```

### Test 2: Check Available Metrics

```bash
claude -p "echo test" --output-format json | jq 'keys'
```

**Available fields:**
- `type` - Always "result"
- `subtype` - "success" or "error"
- `total_cost_usd` - ✅ Cost tracking
- `duration_ms` - ✅ Total time
- `duration_api_ms` - API time
- `num_turns` - ✅ Conversation turns (proxy for iterations)
- `usage` - ✅ Token usage details
  - `input_tokens` - Input tokens
  - `output_tokens` - Output tokens
  - `cache_creation_input_tokens` - Cache creation tokens
  - `cache_read_input_tokens` - Cache read tokens
- `modelUsage` - ✅ Per-model breakdown with token counts and costs
- `result` - Output text
- `session_id` - Session ID
- `is_error` - Error flag
- `uuid` - Unique run identifier

**What we CAN collect:**
- ✅ Token counts (input/output) - Available in `usage` field!
- ✅ Per-model costs - Available in `modelUsage` field!
- ✅ Cache metrics - Cache hit/creation tokens
- ✅ Iterations - `num_turns` field

---

## Robust Detection in Code

### Bash Script Approach

```bash
#!/bin/bash
# tools/find_claude.sh - Detect and use Claude CLI

detect_claude() {
    # Option 1: Direct command
    if command -v claude &> /dev/null; then
        echo "claude"
        return 0
    fi

    # Option 2: NPX (local install)
    if [ -f "node_modules/.bin/claude" ]; then
        echo "npx claude"
        return 0
    fi

    # Option 3: NVM with node 18
    if command -v nvm &> /dev/null; then
        if nvm exec 18 which claude &> /dev/null; then
            echo "nvm exec 18 claude"
            return 0
        fi
    fi

    # Option 4: Docker
    if command -v docker &> /dev/null; then
        if docker images ailang-claude -q | grep -q .; then
            echo "docker run --rm -v /tmp:/tmp ailang-claude"
            return 0
        fi
    fi

    # Not found
    echo "ERROR: Claude CLI not found. See docs/CLAUDE_CODE_SETUP.md" >&2
    return 1
}

# Usage:
CLAUDE_CMD=$(detect_claude)
if [ $? -ne 0 ]; then
    exit 1
fi

# Run headless
$CLAUDE_CMD -p "task" --output-format json
```

### Go Approach (for eval harness)

```go
// internal/eval_harness/claude_detect.go

func detectClaudeCLI() (string, error) {
    // Option 1: Direct command
    if _, err := exec.LookPath("claude"); err == nil {
        return "claude", nil
    }

    // Option 2: NPX (local install)
    if _, err := os.Stat("node_modules/.bin/claude"); err == nil {
        return "npx claude", nil
    }

    // Option 3: Docker
    cmd := exec.Command("docker", "images", "ailang-claude", "-q")
    if output, err := cmd.Output(); err == nil && len(output) > 0 {
        return "docker run --rm -v /tmp:/tmp ailang-claude", nil
    }

    return "", fmt.Errorf("Claude CLI not found. Install: npm install -g @anthropic-ai/claude-code")
}

// Usage in eval harness:
func RunAgentEvalSuite() error {
    claudeCmd, err := detectClaudeCLI()
    if err != nil {
        return fmt.Errorf("Claude CLI setup required: %w\nSee: docs/CLAUDE_CODE_SETUP.md", err)
    }

    log.Printf("Using Claude CLI: %s", claudeCmd)
    // ... proceed with eval
}
```

---

## Recommended Setup for Development

**Quick start:**
```bash
# 1. Install globally with node 18+
nvm use 18
npm install -g @anthropic-ai/claude-code

# 2. Verify it works
claude --version
claude -p "echo test" --output-format json

# 3. Add to PATH (to avoid nvm switching)
# Add to ~/.zshrc:
export PATH="$(nvm which 18 | xargs dirname):$PATH"

# 4. Test from AILANG project
cd /path/to/ailang
make test-claude-headless  # (we'll create this target)
```

---

## Troubleshooting

### Issue: `claude: command not found`

**Solution 1**: Install globally
```bash
npm install -g @anthropic-ai/claude-code
```

**Solution 2**: Use npx
```bash
npx @anthropic-ai/claude-code -p "test" --output-format json
```

### Issue: Node version mismatch

**Solution**: Claude requires node 18+
```bash
nvm install 18
nvm use 18
npm install -g @anthropic-ai/claude-code
```

### Issue: NPM global install not in PATH

**Solution**: Add npm global bin to PATH
```bash
# Find npm global bin directory
npm config get prefix

# Add to PATH in ~/.zshrc or ~/.bashrc:
export PATH="$(npm config get prefix)/bin:$PATH"
```

### Issue: Multiple node versions installed

**Solution**: Use nvm alias for consistency
```bash
nvm alias claude-node 18
nvm exec claude-node claude -p "test" --output-format json
```

---

## Makefile Target (for easy testing)

```makefile
# Add to Makefile:

.PHONY: setup-claude
setup-claude:
	@echo "Installing Claude CLI globally..."
	npm install -g @anthropic-ai/claude-code
	@echo "✓ Claude CLI installed"
	@claude --version

.PHONY: test-claude-headless
test-claude-headless:
	@echo "Testing Claude headless mode..."
	@claude -p "echo test" --output-format json | jq -r '.subtype'
	@if [ $$? -eq 0 ]; then \
		echo "✓ Claude headless mode working"; \
	else \
		echo "✗ Claude headless mode failed. Run: make setup-claude"; \
		exit 1; \
	fi

.PHONY: check-claude
check-claude:
	@command -v claude >/dev/null 2>&1 || { \
		echo "Claude CLI not found."; \
		echo "Install: npm install -g @anthropic-ai/claude-code"; \
		echo "Or run: make setup-claude"; \
		exit 1; \
	}
	@echo "✓ Claude CLI found: $$(which claude)"
	@echo "✓ Version: $$(claude --version)"
```

**Usage:**
```bash
make setup-claude         # Install Claude CLI
make update-claude        # Update to latest version
make check-claude         # Verify installation
make test-claude-headless # Test headless mode
```

---

## What Metrics We Can Actually Collect

Based on the actual JSON output from Claude Code v2.0.27, here's what we have:

**✅ Available from `claude -p --output-format json`:**
- `total_cost_usd` - Exact cost per session
- `duration_ms` - Total execution time
- `duration_api_ms` - API time (excluding local overhead)
- `num_turns` - Number of conversation turns
- `subtype` - "success" or "error"
- `session_id` - For debugging
- `uuid` - Unique run identifier
- **`usage`** - Token usage breakdown:
  - `input_tokens` - Total input tokens
  - `output_tokens` - Total output tokens
  - `cache_creation_input_tokens` - Cache creation tokens
  - `cache_read_input_tokens` - Cache hit tokens
- **`modelUsage`** - Per-model breakdown (e.g., haiku vs sonnet):
  - `inputTokens`, `outputTokens` per model
  - `cacheReadInputTokens`, `cacheCreationInputTokens` per model
  - `costUSD` per model
  - `contextWindow` per model

**What we CAN track:**
- ✅ Exact token counts (not estimated!)
- ✅ Per-model costs and usage
- ✅ Cache efficiency metrics
- ✅ Iterations (num_turns)
- ✅ API vs total duration

---

## Updated Eval Harness Metrics

Given the ACTUAL available data (better than we thought!), here's what we can collect:

```go
type BenchmarkResult struct {
    BenchmarkID  string
    Success      bool
    Cost         float64  // From total_cost_usd
    Duration     float64  // From duration_ms
    NumTurns     int      // From num_turns
    SessionID    string
    UUID         string
    Error        error

    // Token metrics (ACTUAL counts, not estimated!)
    InputTokens              int     // From usage.input_tokens
    OutputTokens             int     // From usage.output_tokens
    CacheCreationTokens      int     // From usage.cache_creation_input_tokens
    CacheReadTokens          int     // From usage.cache_read_input_tokens

    // Per-model breakdown
    ModelUsage map[string]ModelMetrics
}

type ModelMetrics struct {
    InputTokens         int
    OutputTokens        int
    CacheReadTokens     int
    CacheCreationTokens int
    CostUSD             float64
    ContextWindow       int
}
```

**Dashboard will show:**
- Success rate: ✅
- Avg cost: ✅
- Avg num_turns: ✅ (as "iterations")
- Avg duration: ✅
- Estimated tokens: ⚠️ (rough approximation)

---

## Next Steps

1. **Install Claude CLI**: Run `make setup-claude`
2. **Verify installation**: Run `make test-claude-headless`
3. **Update eval harness**: Use actual available metrics
4. **Document limitations**: Dashboard notes token estimates are approximate

---

**Created**: October 27, 2025
**For**: M-EVAL-AGENT (v0.4.0) and M-CLAUDE-CODE-HEADLESS (v0.3.22)
