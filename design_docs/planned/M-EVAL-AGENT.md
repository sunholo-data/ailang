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

## Vision: 3-Tier Eval System (Dashboard Dimensions)

### Tier 1: 0-Shot (Baseline)
**Current**: ✅ Implemented
**Method**: LLM generates code directly from problem description
**Dashboard Metrics**:
- Success rate: % benchmarks passing
- Avg tokens: Input + output tokens per benchmark
- Avg cost: $ per benchmark
- Time: Total runtime

**Example**: Claude Sonnet 4.5 → 65% success, 2.5k tokens, $0.02/benchmark, 2min total

---

### Tier 2: 1-Repair (Current Default)
**Current**: ✅ Implemented (default for `ailang eval-suite`)
**Method**: Generate → Test → Fix once based on error message
**Dashboard Metrics**:
- Success rate: % benchmarks passing after 1 repair
- Avg tokens: Input + output tokens (initial + repair)
- Avg cost: $ per benchmark (initial + repair)
- Repair success rate: % that succeeded after repair vs 0-shot
- Time: Total runtime

**Example**: Claude Sonnet 4.5 → 78% success, 5k tokens, $0.04/benchmark, 5min total

---

### Tier 3: Agent (Future) 🎯
**Current**: ❌ Not implemented (this milestone!)
**Method**: Full autonomous agent with tools and iteration
**Tools Available**:
- ✅ Read/write AILANG files
- ✅ Run `ailang check` (type checking)
- ✅ Run tests repeatedly
- ✅ Read error messages and stack traces
- ✅ Inspect code structure (future: reflection API)
- ✅ Plan multi-step solutions

**Dashboard Metrics** (NEW!):
- Success rate: % benchmarks passing after N iterations
- Avg iterations: How many attempts to success
- Avg tokens: Total tokens across all iterations
- Avg cost: $ per benchmark (all iterations)
- Tool usage: Avg Read/Write/Bash/Edit calls per benchmark
- Time: Total runtime + time per iteration
- Iteration distribution: Histogram of iterations to success

**Example**: Claude Sonnet 4.5 → 92% success, 18k tokens, $0.15/benchmark, 3.5 iterations avg, 20min total

**Key Insight**: Agent tier is **3x more expensive but 15-20% more successful** - proves AILANG's AI-first design!

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

## Why Claude Code is the Ideal First Implementation

**Claude Code is the pragmatic way AILANG will be used in practice:**

### 1. Native Tool Integration
- **Built-in tools**: Bash, Read, Write, Edit, Grep - exactly what's needed for AILANG development
- **No custom tooling**: Don't need to implement file I/O or shell access
- **Real-world workflow**: Matches how developers actually use AILANG

### 2. Iteration is Natural
- **Multi-turn by design**: Claude Code sessions are inherently iterative
- **Error feedback loops**: Read error → edit code → run again → repeat
- **Type-check → test → fix**: Natural workflow for AILANG development

### 3. Fresh Sessions = Deterministic Benchmarks
- **Headless mode**: `claude -p` creates isolated session per benchmark
- **No state bleed**: Each benchmark starts from clean slate
- **Reproducible**: Same prompt + tools = consistent behavior (modulo LLM sampling)

### 4. Cost-Effective for Development
- **Pay-per-use**: Only pay for benchmark runs that happen
- **No infrastructure**: Don't need to host agent runtime
- **Transparent pricing**: Headless JSON includes `total_cost_usd`

### 5. Dogfooding AILANG's Design
- **AILANG was built for this**: Deterministic semantics, explicit effects, machine-readable errors
- **Validates vision**: If Claude Code + AILANG works well, it proves the AI-first design
- **Marketing**: "AILANG benchmarks show 15-20% improvement with agent workflows" = powerful message

### 6. Other Providers Later
Once Claude Code proves the concept:
- **Gemini Agents**: Similar headless API (when available)
- **OpenAI o1**: Reasoning models (CLI-based)
- **Custom frameworks**: LangChain, AutoGPT, etc.

**Decision**: Start with Claude Code, expand to others in Phase 2+

---

## Architecture

### Claude Code Integration with Agent Protocol (Queue-Based Architecture)

**Critical Design Principle**: Eval harness uses AILANG agent protocol as a **task queue** to coordinate parallel execution of isolated headless sessions.

**Architecture Overview:**

```
Eval Harness (ailang eval-suite --agent)
    ↓
1. Send all 264 benchmarks to agent inbox
    ↓
2. Poll inbox for tasks
    ↓
3. Spawn headless Claude sessions (10 parallel, rate-limited)
    │
    ├─→ Session 1: list_map      → posts result to inbox → ack task
    ├─→ Session 2: tree_traversal → posts result to inbox → ack task
    ├─→ ...
    └─→ Session 10: factorial    → posts result to inbox → ack task
    ↓
4. Collect all results from inbox
    ↓
5. Generate eval report
```

**Key Insight**: The eval harness **controls everything** - it's both the producer and consumer of tasks. The agent inbox is just a **persistent queue** for robustness.

**Why This Architecture:**
1. ✅ **Persistent queue**: Agent inbox survives crashes, tasks don't get lost
2. ✅ **State isolation**: Each headless session = one benchmark = no context bleed
3. ✅ **Rate-limited parallelism**: Spawn N sessions concurrently, respect API quotas
4. ✅ **Dogfooding**: Uses AILANG's own agent messaging system
5. ✅ **Result collection**: All results land in inbox for easy aggregation
6. ✅ **Crash recovery**: If harness crashes, unacked tasks remain in queue for retry

**Detailed Workflow:**

```go
// internal/eval_harness/agent_runner.go

func RunAgentEvalSuite(benchmarks []Benchmark, config *AgentConfig) (*EvalReport, error) {
    // Step 1: Send all benchmarks to agent inbox (task creation)
    log.Printf("Sending %d benchmarks to eval-agent inbox...", len(benchmarks))
    for _, benchmark := range benchmarks {
        msg := map[string]interface{}{
            "task":         "solve_benchmark",
            "benchmark_id": benchmark.ID,
            "spec":         benchmark.Spec,
            "tests_file":   benchmark.TestsFile,
            "timeout":      300,
        }
        if err := sendAgentMessage("eval-agent", msg); err != nil {
            return nil, err
        }
    }
    log.Printf("✓ All %d tasks queued", len(benchmarks))

    // Step 2: Process tasks with rate-limited parallelism
    maxConcurrent := config.MaxConcurrent // e.g., 10
    sem := make(chan struct{}, maxConcurrent)
    results := make(chan *BenchmarkResult, len(benchmarks))

    // Rate limiter (respect API quotas: e.g., 60 requests/min)
    rateLimiter := time.NewTicker(time.Second / time.Duration(config.RequestsPerSecond))
    defer rateLimiter.Stop()

    processed := 0
    for processed < len(benchmarks) {
        // Get next batch of unread tasks
        tasks, err := getUnreadTasks("eval-agent")
        if err != nil {
            return nil, err
        }

        for _, task := range tasks {
            <-rateLimiter.C  // Wait for rate limiter
            sem <- struct{}{}  // Acquire semaphore slot

            go func(task AgentTask) {
                defer func() { <-sem }()  // Release semaphore

                // Spawn fresh headless Claude Code session
                result := runHeadlessSession(task)

                // Post result to results inbox
                resultMsg := map[string]interface{}{
                    "benchmark_id": task.BenchmarkID,
                    "success":      result.Success,
                    "cost":         result.Cost,
                    "iterations":   result.Iterations,
                    "duration":     result.Duration,
                }
                sendAgentMessage("eval-results", resultMsg)

                // Acknowledge task (removes from queue)
                ackTask(task.MessageID)

                results <- result
                processed++
            }(task)
        }

        time.Sleep(1 * time.Second)  // Poll interval
    }

    // Wait for all sessions to complete
    for i := 0; i < cap(sem); i++ {
        sem <- struct{}{}
    }
    close(results)

    // Step 3: Collect all results from results inbox
    log.Println("Collecting results from eval-results inbox...")
    allResults := collectResultsFromInbox("eval-results")

    // Step 4: Generate eval report
    report := generateEvalReport(allResults)
    return report, nil
}

func runHeadlessSession(task AgentTask) *BenchmarkResult {
    workspace := fmt.Sprintf("/tmp/ailang_eval_%s_%d", task.BenchmarkID, os.Getpid())
    os.MkdirAll(workspace, 0755)
    defer os.RemoveAll(workspace)

    // Generate prompt
    prompt := generateBenchmarkPrompt(task)

    startTime := time.Now()

    // Run headless Claude Code (ONE benchmark, then exit)
    cmd := exec.Command("claude", "-p", prompt,
        "--output-format", "json",
        "--workspace", workspace,
        "--allowedTools", "Bash,Read,Write,Edit,Grep",
    )
    output, err := cmd.Output()

    duration := time.Since(startTime).Seconds()

    // Parse JSON result (captures ALL metrics from Claude Code)
    var result struct {
        Subtype      string  `json:"subtype"`        // "success" or "error"
        TotalCostUSD float64 `json:"total_cost_usd"` // Total cost for session
        SessionID    string  `json:"session_id"`     // Unique session ID
        Result       string  `json:"result"`         // Output text

        // Token counts (if available from Claude API)
        InputTokens  int `json:"input_tokens"`
        OutputTokens int `json:"output_tokens"`

        // Tool usage stats (NEW - would need claude -p to expose this)
        ToolCalls []struct {
            Tool  string `json:"tool"`  // "Bash", "Read", "Write", "Edit"
            Count int    `json:"count"`
        } `json:"tool_calls"`
    }
    json.Unmarshal(output, &result)

    // Count iterations by parsing tool calls (heuristic)
    iterations := estimateIterations(result.ToolCalls)

    return &BenchmarkResult{
        BenchmarkID:  task.BenchmarkID,
        Success:      result.Subtype == "success",
        Cost:         result.TotalCostUSD,
        InputTokens:  result.InputTokens,
        OutputTokens: result.OutputTokens,
        TotalTokens:  result.InputTokens + result.OutputTokens,
        Iterations:   iterations,
        Duration:     duration,
        ToolCalls:    result.ToolCalls,
        SessionID:    result.SessionID,
        Error:        err,
    }
}

func estimateIterations(toolCalls []ToolCall) int {
    // Heuristic: Each "run tests" is likely an iteration
    testRuns := 0
    for _, call := range toolCalls {
        if strings.Contains(call.Tool, "Bash") && call.Count > 0 {
            // Approximate: ailang run/check calls = iterations
            testRuns += call.Count
        }
    }
    return max(1, testRuns) // At least 1 iteration
}
```

**CLI Usage:**

```bash
# Run agent eval suite with rate limiting
ailang eval-suite --agent \
  --models claude-sonnet-4-5 \
  --max-concurrent 10 \
  --requests-per-second 1

# Output:
# Sending 264 benchmarks to eval-agent inbox...
# ✓ All 264 tasks queued
# Processing: [=====>    ] 132/264 (50%) | ETA: 5m 12s
# Parallel sessions: 10 active
# Avg iterations: 3.2 | Avg cost: $0.14 | Total tokens: 17.8k avg
# ✓ All tasks completed
# Collecting results from eval-results inbox...
# ✓ 243/264 passed (92% success rate)
# Report: eval_results/agent/claude-sonnet-4-5/v0.4.0/
```

---

## Dashboard Integration (NEW Dimension)

**Current dashboard** (`docs/BENCHMARK_COMPARISON.md`):
- Shows 0-shot vs 1-repair for all models
- Metrics: success rate, cost, tokens

**Updated dashboard** (after M-EVAL-AGENT):
- Shows **3 tiers**: 0-shot, 1-repair, **agent** (NEW!)
- New metrics: iterations, tool usage, time per iteration

### Example Dashboard Table:

```markdown
## Claude Sonnet 4.5 Performance (v0.4.0)

| Tier      | Success | Avg Tokens | Avg Cost | Avg Iterations | Avg Time | Tool Calls |
|-----------|---------|------------|----------|----------------|----------|------------|
| 0-Shot    | 65%     | 2,500      | $0.02    | 1              | 45s      | -          |
| 1-Repair  | 78%     | 5,000      | $0.04    | 2 (max)        | 90s      | -          |
| **Agent** | **92%** | **18,000** | **$0.15**| **3.5**        | **180s** | **Read: 4.2, Write: 3.1, Bash: 5.3** |

**Key Insights**:
- Agent tier achieves **+14% higher success** than 1-repair
- Cost increase: **3.75x** (but 14% more problems solved)
- Iteration distribution: 1 iter (20%), 2-4 iters (60%), 5+ iters (20%)
- Most common failure mode: Timeout at 10 iterations (5% of benchmarks)
```

### Multi-Model Comparison (Agent Tier):

```markdown
## Agent Tier Performance Comparison (v0.4.0)

| Model               | Success | Avg Tokens | Avg Cost | Avg Iterations |
|---------------------|---------|------------|----------|----------------|
| Claude Sonnet 4.5   | 92%     | 18,000     | $0.15    | 3.5            |
| Gemini 2.5 Pro      | 88%     | 16,500     | $0.08    | 4.1            |
| GPT-5               | 89%     | 19,200     | $0.19    | 3.8            |
| Claude Haiku 4.5    | 82%     | 14,000     | $0.05    | 4.5            |
| Gemini 2.5 Flash    | 79%     | 13,500     | $0.03    | 5.2            |

**Best Overall**: Claude Sonnet 4.5 (92% success, reasonable cost)
**Best Value**: Gemini 2.5 Flash (79% success, $0.03/benchmark)
**Fastest**: Claude Sonnet 4.5 (3.5 iterations avg)
```

### Visualization:

```
Success Rate by Tier (Claude Sonnet 4.5)

0-Shot:   ████████████████          65%
1-Repair: ███████████████████       78%  (+13% improvement)
Agent:    ███████████████████████   92%  (+14% improvement)
          ├────────┼────────┼────────┼────────┼────────┤
          0%      20%      40%      60%      80%     100%

Cost by Tier (Claude Sonnet 4.5)

0-Shot:   █                         $0.02
1-Repair: ██                        $0.04  (2x)
Agent:    ████████                  $0.15  (7.5x)
          ├────────┼────────┼────────┼────────┤
          $0.00   $0.05   $0.10    $0.15   $0.20
```

**Key Insight**: **Each `claude -p` invocation is a fresh session that takes ONE task, solves it, and exits**. No long-running agent session that processes multiple benchmarks.

**Summary of Data Flow:**

1. **Eval harness** → Sends 264 tasks → **eval-agent inbox**
2. **Eval harness** → Polls **eval-agent inbox** → Gets unread tasks
3. **Eval harness** → Spawns **10 parallel headless sessions** (rate-limited)
4. **Each headless session** → Solves ONE benchmark → Posts result to **eval-results inbox**
5. **Eval harness** → Acks task in **eval-agent inbox** (removes from queue)
6. **Eval harness** → Collects all results from **eval-results inbox**
7. **Eval harness** → Generates report

**Benefits of this architecture:**
- ✅ **Eval harness controls everything**: Single process orchestrates the whole pipeline
- ✅ **Agent protocol provides robustness**: Tasks persist in inbox, crash recovery built-in
- ✅ **Fresh sessions ensure isolation**: Each benchmark starts with zero context
- ✅ **Rate limiting respects API quotas**: Won't hit Claude API limits
- ✅ **Parallel execution speeds up runs**: 10 concurrent sessions = 10x faster
- ✅ **Result collection is simple**: All results land in one inbox for aggregation

**Integration with M-CLAUDE-CODE-HEADLESS:**

The headless wrapper script handles session management:

```bash
# tools/run_headless_claude.sh wraps claude -p with:
# - Workspace isolation
# - JSON output capture
# - Cost tracking
# - Error handling
# - Timeout enforcement

./tools/run_headless_claude.sh \
  --task-from-inbox eval-agent \
  --workspace /tmp/eval_${benchmark_id} \
  --timeout 300
```

### Agent Benchmark Runner

```go
// internal/eval_harness/agent_runner.go

type AgentBenchmarkConfig struct {
    Provider     string  // "claude-code", "claude-cli", "gemini", "openai"
    Model        string  // "claude-sonnet-4-5", "gemini-2.5-pro", etc.

    // Claude Code specific
    HeadlessMode bool    // true = use headless wrapper, false = SDK (not recommended for evals)
    WorkspaceDir string  // Isolated workspace per benchmark

    MaxIterations int    // e.g., 10
    TimeoutSeconds int   // e.g., 300 (5 minutes)

    // Tools available to agent
    Tools []string  // ["Bash", "Read", "Write", "Edit", "Grep"]
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

### Example Agent Session (Claude Code - Headless)

**Benchmark**: `list_map` - Implement map function over lists

**Workspace**: `/tmp/ailang_eval_list_map_20251027_143022/`

**Prompt** (sent to headless Claude Code):
```
You are solving an AILANG benchmark. Your task is to implement the `map` function that applies a function to each element of a list.

Benchmark ID: list_map
Language: AILANG
Tools Available: Bash, Read, Write, Edit, Grep

Files in workspace:
- README.md: Problem description
- tests.ail: Test cases you must pass
- solution.ail: Your implementation goes here (currently empty)

Instructions:
1. Read README.md to understand the problem
2. Write your implementation in solution.ail
3. Use `ailang check solution.ail` to type-check
4. Use `ailang run --entry test --caps IO tests.ail` to run tests
5. Iterate until all tests pass
6. You have 10 iterations max and 5 minutes timeout

IMPORTANT: AILANG syntax reference is in /workspace/syntax_reference.md
```

**Session Flow**:

```
[Iteration 1]
Claude Code: Let me read the problem description first.
  Tool: Read(README.md)
  Output: "Implement map :: (a -> b) -> [a] -> [b]"

Claude Code: I'll write an initial implementation.
  Tool: Write(solution.ail, "...")
  Tool: Bash("ailang check solution.ail")
  Output: ✓ Type check passed

  Tool: Bash("ailang run --entry test --caps IO tests.ail")
  Output: ✗ Test 2/5 failed: Empty list case returns error

[Iteration 2]
Claude Code: I need to handle the empty list case in the pattern match.
  Tool: Edit(solution.ail, old="...", new="...")
  Tool: Bash("ailang run --entry test --caps IO tests.ail")
  Output: ✗ Test 4/5 failed: Type mismatch in result

[Iteration 3]
Claude Code: The issue is in how I'm constructing the result list. Let me fix that.
  Tool: Edit(solution.ail, old="...", new="...")
  Tool: Bash("ailang check solution.ail")
  Output: ✓ Type check passed
  Tool: Bash("ailang run --entry test --caps IO tests.ail")
  Output: ✓ All 5 tests passed!

Final Result: SUCCESS (3 iterations, 47 seconds, $0.12)
```

**JSON Output** (captured by headless wrapper):
```json
{
  "type": "result",
  "subtype": "success",
  "session_id": "abc123",
  "total_cost_usd": 0.12,
  "result": "All tests passed! Implementation complete.",
  "iterations": 3,
  "duration_seconds": 47,
  "tool_calls": [
    {"tool": "Read", "count": 2},
    {"tool": "Write", "count": 1},
    {"tool": "Edit", "count": 2},
    {"tool": "Bash", "count": 5}
  ]
}
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

### Phase 0: Prerequisites (v0.3.21) [REQUIRED FIRST]

**Dependency**: M-CLAUDE-CODE-HEADLESS must be implemented first!

1. ✅ Agent protocol system (v0.3.19 - done!)
2. ✅ Claude Code hooks integration (v0.3.20 - done!)
3. ⏳ **Headless wrapper scripts** (v0.3.21 - BLOCKING)
   - `tools/run_headless_claude.sh` - Wrapper for `claude -p`
   - JSON output capture with session_id, cost, result
   - Workspace isolation per invocation
   - Error handling and timeouts

**Why this blocks**: We can't run agent evals without headless mode. Each benchmark needs a fresh Claude Code session, which requires the headless wrapper.

**Estimated**: 2-3 days (see M-CLAUDE-CODE-HEADLESS design doc)

**Status**: 🚧 Blocked until M-CLAUDE-CODE-HEADLESS ships

---

### Phase 1: Claude Code Integration (~1 week)

**Once M-CLAUDE-CODE-HEADLESS is done:**

1. ⏳ Create `ClaudeCodeBenchmarkRunner` (wraps headless wrapper)
   - Workspace creation/cleanup per benchmark
   - Prompt construction (benchmark spec → Claude Code task)
   - JSON result parsing (extract success, iterations, cost)
   - Integration with existing `AgentBenchmarkResult` struct

2. ⏳ Benchmark prompt engineering
   - System prompt: "You are solving AILANG benchmarks. Use `ailang check`, `ailang run`, iterate until tests pass."
   - Include AILANG syntax reference (from prompts/v0.3.X.md)
   - Examples of using ailang CLI tools
   - Iteration strategies

3. ⏳ CLI integration
   - `ailang eval-suite --agent --provider claude-code`
   - `ailang eval-benchmark <id> --agent --provider claude-code`
   - Automatic workspace management (create/cleanup)

4. ⏳ Run pilot with 10 benchmarks
   - Select: 2 simple, 5 medium, 3 complex benchmarks
   - Compare: 0-shot vs 1-repair vs Claude Code agent
   - Measure: success rate, iterations, cost, time

**Deliverable**: Proof-of-concept showing Claude Code agent outperforms 1-repair

**Estimated**: 1 week (assuming Phase 0 complete)

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
**Updated**: October 27, 2025 (Claude Code integration specifics added)
**Target Version**: v0.4.0
**Dependencies**:
- ✅ v0.3.19 (Agent protocol system) - Complete
- ✅ v0.3.20 (Claude Code hooks integration) - Complete
- 🚧 **v0.3.21 (M-CLAUDE-CODE-HEADLESS) - BLOCKING** - Required for headless wrapper
- ⏳ Reflection API (future, v0.5.0+) - For advanced tools

**Critical Path**:
1. M-CLAUDE-CODE-HEADLESS (v0.3.21) must ship first
2. Then M-EVAL-AGENT Phase 1 can begin
3. Phase 1 deliverable: Claude Code agent benchmarks working

**Related**:
- M-EVAL: Original eval harness
- M-EVAL-LOOP: Go-based eval tools
- M-AGENT-PROTOCOL: Agent communication system (v0.3.19)
- M-CLAUDE-CODE-INTEGRATION-HOOKS: Interactive Claude Code integration (v0.3.20)
- **M-CLAUDE-CODE-HEADLESS**: Headless mode wrapper (v0.3.21) - REQUIRED FIRST
