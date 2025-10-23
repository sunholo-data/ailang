# M-EVAL-AGENT: Multi-Agent Eval Benchmark Suite

**Status**: Planned (v0.4.0)
**Created**: October 23, 2025
**Motivation**: Benchmark autonomous agents across multiple providers using AILANG's deterministic design

---

## Problem

**Current M-EVAL limitations**:
1. **0-shot**: LLM generates code once, no feedback loop
2. **1-repair**: LLM can fix code once based on test failure
3. **No agentic workflows**: Can't leverage multi-step reasoning, tool use, iteration

**Why this matters**:
AILANG is designed for **AI-first development**:
- Deterministic semantics (predictable evaluation)
- Effect system (explicit side effects)
- Reflection API (future: inspect code structure)
- Canonical normalization (semantic comparison)

These features make AILANG **ideal for autonomous agents**, but we're not benchmarking that!

---

## Vision: 3-Tier Eval System

### Tier 1: 0-Shot (Baseline)
**Current**: ✅ Implemented
**Method**: LLM generates code directly from problem description
**Benchmark**: Pass/fail on test suite
**Strengths**: Fast, cheap
**Weaknesses**: No learning from failures

### Tier 2: 1-Repair (Current Best)
**Current**: ✅ Implemented
**Method**: Generate → Test → Fix once based on error message
**Benchmark**: Pass/fail after single repair attempt
**Strengths**: Simple feedback loop
**Weaknesses**: Only one repair attempt, no tool use

### Tier 3: Agent (Future) 🎯
**Current**: ❌ Not implemented
**Method**: Full autonomous agent with tools and iteration
**Tools Available**:
- ✅ Read/write AILANG files
- ✅ Run `ailang check` (type checking)
- ✅ Run tests repeatedly
- ✅ Read error messages and stack traces
- ✅ Inspect code structure (future: reflection API)
- ✅ Plan multi-step solutions

**Benchmark**: Pass/fail after agent completes (with iteration limit)
**Strengths**: Multi-step reasoning, tool use, iteration
**Weaknesses**: Slower, more expensive

---

## Expected Performance Gains

Based on AILANG's design advantages for agents:

| Benchmark | 0-Shot | 1-Repair | Agent (Predicted) |
|-----------|--------|----------|-------------------|
| **Simple** (e.g., hello_world) | 95% | 98% | 99% |
| **Medium** (e.g., list_ops) | 60% | 75% | **90%** ← Big gain |
| **Complex** (e.g., type_inference) | 30% | 45% | **80%** ← Huge gain |
| **Very Complex** (e.g., effect_propagation) | 10% | 20% | **60%** ← Dramatic gain |

**Why agents should excel**:
1. **Iteration**: Can fix code multiple times based on test failures
2. **Tool use**: Can run tests, type checker, read documentation
3. **Planning**: Can break complex problems into steps
4. **AILANG's determinism**: Predictable evaluation aids reasoning

---

## Architecture

### Agent Benchmark Runner

```go
// internal/eval_harness/agent_runner.go

type AgentBenchmarkConfig struct {
    Provider     string  // "claude", "gemini", "openai"
    Model        string  // "claude-sonnet-4-5", "gemini-2.5-pro", etc.
    AgentFile    string  // Path to .claude/agents/*.md or nil for CLI
    MaxIterations int    // e.g., 10
    TimeoutSeconds int   // e.g., 300 (5 minutes)

    // Tools available to agent
    Tools []string  // ["read_file", "write_file", "run_tests", "type_check"]
}

type AgentBenchmarkResult struct {
    Success       bool
    Iterations    int      // How many attempts before success/failure
    Duration      float64  // Seconds
    TokensUsed    int
    Cost          float64
    ToolCalls     []ToolCall
    FinalCode     string
    ErrorTrace    []string  // History of errors encountered
}

func RunAgentBenchmark(benchmark *Benchmark, config *AgentBenchmarkConfig) (*AgentBenchmarkResult, error) {
    // 1. Start agent with problem description
    // 2. Agent can:
    //    - Read problem spec
    //    - Generate code
    //    - Run `ailang check`
    //    - Run tests
    //    - Read error messages
    //    - Iterate until passing or max iterations
    // 3. Return result with metrics
}
```

### Tools for Agents

```go
// Tools agents can use during benchmarks

type AgentTool interface {
    Name() string
    Description() string
    Execute(args map[string]interface{}) (map[string]interface{}, error)
}

// ReadFileTool - Read AILANG source code
type ReadFileTool struct {}

func (t *ReadFileTool) Execute(args map[string]interface{}) (map[string]interface{}, error) {
    filePath := args["path"].(string)
    content, err := os.ReadFile(filePath)
    return map[string]interface{}{"content": string(content)}, err
}

// WriteFileTool - Write AILANG source code
type WriteFileTool struct {}

// TypeCheckTool - Run `ailang check`
type TypeCheckTool struct {}

func (t *TypeCheckTool) Execute(args map[string]interface{}) (map[string]interface{}, error) {
    filePath := args["path"].(string)
    cmd := exec.Command("ailang", "check", filePath)
    output, err := cmd.CombinedOutput()

    return map[string]interface{}{
        "success": err == nil,
        "output":  string(output),
    }, nil
}

// RunTestsTool - Run benchmark tests
type RunTestsTool struct {
    Benchmark *Benchmark
}

func (t *RunTestsTool) Execute(args map[string]interface{}) (map[string]interface{}, error) {
    // Run benchmark test suite
    result := runBenchmarkTests(t.Benchmark)

    return map[string]interface{}{
        "passed":  result.Passed,
        "failed":  result.Failed,
        "errors":  result.Errors,
    }, nil
}

// InspectCodeTool - Use reflection API (future)
type InspectCodeTool struct {}
```

### Multi-Provider Support

```go
// Support for Claude, Gemini, OpenAI agents

type AgentProvider interface {
    RunBenchmark(benchmark *Benchmark, tools []AgentTool, maxIterations int) (*AgentBenchmarkResult, error)
}

// ClaudeAgentProvider - Uses .claude/agents/*.md or Claude CLI
type ClaudeAgentProvider struct {
    Handler *agentrunner.ClaudeAgentHandler  // Full agent
    // OR
    Handler *agentrunner.LLMCLIHandler       // CLI-based
}

// GeminiAgentProvider - Uses Gemini CLI
type GeminiAgentProvider struct {
    Handler *agentrunner.LLMCLIHandler
}

// OpenAIAgentProvider - Uses OpenAI CLI or o1 reasoning
type OpenAIAgentProvider struct {
    Handler *agentrunner.LLMCLIHandler
}

// Usage:
providers := []AgentProvider{
    &ClaudeAgentProvider{Model: "claude-sonnet-4-5"},
    &GeminiAgentProvider{Model: "gemini-2.5-pro"},
    &OpenAIAgentProvider{Model: "o1-preview"},  // Reasoning model!
}

for _, provider := range providers {
    result := provider.RunBenchmark(benchmark, tools, 10)
    // Compare results across providers
}
```

---

## Benchmark Workflow

### Agent Iteration Loop

```
1. Agent receives problem description
   ↓
2. Agent generates initial AILANG code
   ↓
3. Agent writes code to file
   ↓
4. Agent runs `ailang check` (type checking)
   ├─ Success → Continue to tests
   └─ Failure → Read error, revise code, goto 3
   ↓
5. Agent runs tests
   ├─ All pass → SUCCESS (exit)
   └─ Some fail → Read failures, analyze, goto 3
   ↓
6. Repeat until:
   - All tests pass (success)
   - Max iterations reached (failure)
   - Timeout (failure)
```

### Example Agent Session (Claude)

```
[Iteration 1]
Agent: I'll implement the list_map function.
  Tool: write_file(path="solution.ail", code="...")
  Tool: type_check(path="solution.ail")
  Result: ✓ Type check passed
  Tool: run_tests(path="solution.ail")
  Result: ✗ 2/5 tests failed
    - Empty list handling incorrect
    - Type mismatch in result

[Iteration 2]
Agent: I see the issue - need to handle empty list case.
  Tool: write_file(path="solution.ail", code="...improved...")
  Tool: run_tests(path="solution.ail")
  Result: ✓ All tests passed!

Final Result: SUCCESS (2 iterations, 45 seconds)
```

---

## CLI Integration

### New Commands

```bash
# Run agent benchmarks (default: Claude only)
ailang eval-suite --agent

# Run with specific models
ailang eval-suite --agent --models claude-sonnet-4-5,gemini-2.5-pro,o1-preview

# Compare 0-shot vs 1-repair vs agent
ailang eval-suite --full --include-agent

# Run single benchmark with agent
ailang eval-benchmark list_map --agent --provider claude --model claude-sonnet-4-5

# Agent-specific options
ailang eval-suite --agent --max-iterations 20 --timeout 600
```

### Report Format

```bash
ailang eval-report eval_results/agents/v0.4.0 v0.4.0 --format=markdown

# Output:
## AILANG v0.4.0 Eval Results (Agent Benchmarks)

### Summary by Tier

| Tier | Claude Sonnet 4.5 | Gemini 2.5 Pro | OpenAI o1 |
|------|-------------------|----------------|-----------|
| 0-Shot | 65% | 62% | 68% |
| 1-Repair | 78% | 75% | 80% |
| **Agent** | **92%** | **88%** | **95%** |

### Agent Performance Breakdown

| Benchmark | Complexity | 0-Shot | 1-Repair | Agent | Iterations (Avg) |
|-----------|------------|--------|----------|-------|------------------|
| hello_world | Simple | 98% | 99% | 99% | 1.2 |
| list_map | Medium | 70% | 82% | **95%** | 3.5 |
| type_inference | Complex | 35% | 48% | **85%** | 6.8 |
| effect_propagation | Very Complex | 12% | 22% | **65%** | 9.2 |

### Tool Usage Stats

Average tool calls per successful benchmark:
- write_file: 4.2
- type_check: 3.8
- run_tests: 5.1
- read_file: 2.3

### Cost Analysis

| Tier | Avg Tokens | Avg Cost | Total Cost (264 benchmarks) |
|------|-----------|----------|----------------------------|
| 0-Shot | 2,500 | $0.02 | $5.28 |
| 1-Repair | 5,000 | $0.04 | $10.56 |
| **Agent** | **18,000** | **$0.15** | **$39.60** |

Note: Agent tier is more expensive but achieves 92% vs 78% success rate.
```

---

## Expected Challenges

### 1. Iteration Management

**Problem**: Agents might loop indefinitely or make same mistake repeatedly.

**Solution**:
- Max iteration limit (configurable, default: 10)
- Timeout (configurable, default: 5 minutes)
- Track error history to detect loops

### 2. Cost Control

**Problem**: Agent benchmarks are expensive (18k tokens vs 2.5k for 0-shot).

**Solution**:
- Use cheap models for development (Gemini Flash, Claude Haiku)
- Use expensive models only for baseline releases
- Cache successful solutions to avoid re-running

### 3. Tool Safety

**Problem**: Agents could write malicious code or access unintended files.

**Solution**:
- Sandbox: Agent can only read/write to benchmark-specific directory
- No network access during benchmarks
- No system commands except `ailang check` and test runner

### 4. Reproducibility

**Problem**: Agents are non-deterministic (different runs = different results).

**Solution**:
- Set temperature=0 for models that support it
- Use seeded randomness where possible
- Run each benchmark 3 times, report average

---

## Implementation Phases

### Phase 1: Single Provider (Claude) (~1 week)

1. ✅ Agent protocol system (already done in v0.3.19!)
2. ⏳ Create AgentBenchmarkRunner
3. ⏳ Implement 4 basic tools (read/write/typecheck/test)
4. ⏳ Integrate with existing eval_harness
5. ⏳ Run pilot with 10 benchmarks

**Deliverable**: Proof-of-concept showing agent outperforms 1-repair

### Phase 2: Multi-Provider (~3 days)

6. ⏳ Add GeminiAgentProvider
7. ⏳ Add OpenAIAgentProvider (with o1 support!)
8. ⏳ Comparison dashboard

**Deliverable**: Head-to-head comparison of Claude vs Gemini vs OpenAI agents

### Phase 3: Production Polish (~1 week)

9. ⏳ Cost tracking and optimization
10. ⏳ Caching successful solutions
11. ⏳ Parallel execution (run multiple benchmarks concurrently)
12. ⏳ Dashboard integration

**Deliverable**: Production-ready agent eval suite

### Phase 4: Advanced Tools (~2 weeks, future)

13. ⏳ Reflection API integration (inspect code structure)
14. ⏳ Planning tools (break complex problems into steps)
15. ⏳ Multi-agent collaboration (agents can message each other)

**Deliverable**: State-of-the-art agent benchmarking for deterministic languages

---

## Success Metrics

### Primary Metric: Agent Improvement Over 1-Repair

**Target**: Agent tier achieves **15-20% higher success rate** than 1-repair tier.

| Complexity | 1-Repair Success | Agent Target | Improvement |
|------------|------------------|--------------|-------------|
| Simple | 95% | 98% | +3% |
| Medium | 75% | **90%** | **+15%** |
| Complex | 45% | **65%** | **+20%** |
| Very Complex | 20% | **40%** | **+20%** |

### Secondary Metrics

- **Iteration efficiency**: Avg iterations to success < 6
- **Tool usage**: Avg tool calls < 15 per benchmark
- **Cost efficiency**: Cost per successful benchmark < $0.20
- **Provider parity**: Claude, Gemini, OpenAI within 5% of each other

---

## Why This Matters for AILANG

### 1. Validates AI-First Design

AILANG was designed for autonomous AI development. Agent benchmarks prove this works.

### 2. Differentiator

No other language has:
- Deterministic semantics for predictable AI reasoning
- Effect system for explicit side effects
- Reflection API for code inspection
- Agent benchmarks measuring agentic workflows

### 3. Research Platform

Enables research questions like:
- Do reasoning models (o1) outperform chat models on code generation?
- How much does iteration help vs better prompts?
- What's the optimal tool set for code generation agents?

### 4. Dogfooding

Use agents to build AILANG itself:
- eval-analyzer agent finds failures
- design-doc-creator agent proposes fixes
- sprint-executor agent implements fixes
- Agent benchmarks measure if fixes work

**Full circle**: Agents building a language for agents! 🤯

---

## Next Steps

**Immediate (v0.3.20)**:
1. None - focus on SDK integration for ClaudeAgentHandler

**Short-term (v0.4.0)**:
2. Implement AgentBenchmarkRunner (Phase 1)
3. Run pilot with 10 benchmarks
4. Compare Claude agent vs 1-repair

**Long-term (v0.5.0+)**:
5. Multi-provider support (Phase 2)
6. Production polish (Phase 3)
7. Advanced tools with reflection API (Phase 4)

---

## Open Questions

1. **Iteration limits**: 10 iterations enough? Too many?
2. **Tool set**: Which tools are most valuable for agents?
3. **Model selection**: Should we use reasoning models (o1) or chat models (Claude)?
4. **Cost-benefit**: Is 3x cost for 15% improvement worth it?
5. **Caching strategy**: How to avoid re-running expensive agent benchmarks?

---

**Created**: October 23, 2025
**Target Version**: v0.4.0
**Dependencies**:
- v0.3.19 (Agent protocol system) ✅
- v0.3.20 (SDK integration for ClaudeAgentHandler)
- Reflection API (future, v0.5.0+)

**Related**:
- M-EVAL: Original eval harness
- M-EVAL-LOOP: Go-based eval tools
- M-AGENT-PROTOCOL: Agent communication system
