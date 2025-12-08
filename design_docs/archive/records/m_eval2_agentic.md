# M-EVAL2: Agentic AI Evaluation with CLI Integration

**Milestone**: M-EVAL2 (Agentic Evaluation Framework)
**Version**: v0.3.0 (depends on M-EVAL completion)
**Timeline**: 2–3 weeks
**Estimated LOC**: ~800–1,000 (Go + integration + tests)
**Priority**: MEDIUM (validates real-world AI coding workflows)

---

## Executive Summary

M-EVAL2 extends the baseline evaluation system (M-EVAL) with **true agentic evaluation** using production AI coding tools like Claude Code CLI and Gemini CLI.

Instead of measuring single-shot code generation, M-EVAL2 measures:
- **Total effort**: Cumulative tokens across all turns
- **Debugging cost**: Tokens spent fixing errors
- **Iteration count**: How many attempts until success
- **Success rate**: Does the agent eventually solve the task?

**Key Insight from M-EVAL (Phase 1):**
The baseline tests will reveal where LLMs struggle with AILANG syntax/semantics, directly informing:
- Documentation improvements
- Example code needed
- Prompt engineering strategies
- Language design decisions

---

## Problem Statement

**M-EVAL (Phase 1) showed us:**
- Python: 290 tokens, compiles ✅, runs ❌ (logic error)
- AILANG: 180 tokens, compiles ❌ (syntax error), can't run

**But we don't know:**
- Can the AI *fix* the AILANG syntax error if given feedback?
- How many total tokens does it take to get working code?
- Is AILANG's lower token count worth the extra debugging turns?

**Real-world AI coding involves iteration:**
- Write code → test → see error → fix → repeat
- Multiple conversation turns
- Cumulative token cost
- Learning from documentation

**M-EVAL2 measures the *full* cost of AI-assisted development.**

---

## Goals & Non-Goals

### Goals
1. **Multi-turn evaluation**: Max 5 turns per benchmark, feedback loop
2. **CLI agent integration**: Support Claude Code, Gemini CLI, others
3. **Cumulative metrics**: Total tokens, turn count, time to success
4. **Agent comparison**: Our agent loop vs. production agents
5. **Prompt learning**: Discover what context/docs help AILANG succeed

### Non-Goals (v0.3.x)
- ❌ Training LLMs on AILANG (v0.4.0+)
- ❌ Real-time code editing (IDE integration)
- ❌ Complex benchmarks requiring web/database
- ❌ Human-in-the-loop evaluation

---

## Design

### Architecture Overview

```
┌──────────────┐
│ Benchmark    │
│ fizzbuzz.yml │
└───────┬──────┘
        │
        ▼
┌─────────────────┐
│ Eval Orchestrator│  ailang eval --agent claude-code --benchmark fizzbuzz
└───────┬─────────┘
        │
    ┌───▼───┐
    │ Agent │  (Strategy Pattern)
    │ Type? │
    └───┬───┘
        │
   ┌────┴─────────────┬──────────────┐
   │                  │              │
   ▼                  ▼              ▼
┌─────────┐    ┌──────────────┐  ┌──────────┐
│ API Loop│    │ Claude Code  │  │ Gemini   │
│ (built) │    │ CLI (spawn)  │  │ CLI      │
└────┬────┘    └──────┬───────┘  └────┬─────┘
     │                │               │
     │    ┌───────────▼───────────┐   │
     └────► Run Code, Collect     ◄───┘
          │ Tokens, Check Success │
          └───────────┬───────────┘
                      │
                      ▼
              ┌───────────────┐
              │ Metrics Logger│
              │ (multi-turn)  │
              └───────────────┘
```

---

## Key Components

### 1. Agent Interface

**File**: `internal/eval_harness/agent.go`

```go
type Agent interface {
    // Run a benchmark task with multi-turn iteration
    Run(ctx context.Context, task BenchmarkTask) (*AgentResult, error)

    // Name returns the agent type (e.g., "claude-code", "api-loop")
    Name() string
}

type BenchmarkTask struct {
    Prompt       string
    Lang         string
    ExpectedOut  string
    Workspace    string
    Caps         []string
    MaxTurns     int
    Timeout      time.Duration
}

type AgentResult struct {
    Success      bool
    Turns        int
    TotalTokens  int
    TokensPerTurn []int
    DurationMs   int64
    FinalCode    string
    ErrorHistory []string
}
```

### 2. Built-In Agent Loop

**File**: `internal/eval_harness/loop_agent.go`

Simple multi-turn refinement using LLM APIs:

```go
type LoopAgent struct {
    model    string
    maxTurns int
}

func (a *LoopAgent) Run(ctx context.Context, task BenchmarkTask) (*AgentResult, error) {
    result := &AgentResult{}
    conversation := []Message{}

    // Turn 1: Initial attempt
    conversation = append(conversation, Message{
        Role: "user",
        Content: task.Prompt,
    })

    for turn := 1; turn <= task.MaxTurns; turn++ {
        // Generate code
        response, err := callLLM(ctx, a.model, conversation)
        if err != nil {
            return nil, err
        }

        result.TotalTokens += response.Tokens
        result.TokensPerTurn = append(result.TokensPerTurn, response.Tokens)

        code := extractCode(response.Content)

        // Test the code
        runResult := executeCode(task.Lang, code, task.ExpectedOut, task.Caps)

        if runResult.Success {
            result.Success = true
            result.Turns = turn
            result.FinalCode = code
            return result, nil
        }

        // Prepare feedback for next turn
        feedback := fmt.Sprintf(
            "The code failed with error:\n%s\n\nExpected output:\n%s\n\nActual output:\n%s\n\nPlease fix the code.",
            runResult.Stderr,
            task.ExpectedOut,
            runResult.Stdout,
        )

        conversation = append(conversation, Message{
            Role: "assistant",
            Content: response.Content,
        }, Message{
            Role: "user",
            Content: feedback,
        })

        result.ErrorHistory = append(result.ErrorHistory, runResult.Stderr)
    }

    return result, fmt.Errorf("failed after %d turns", task.MaxTurns)
}
```

### 3. Claude Code CLI Agent

**File**: `internal/eval_harness/claude_agent.go`

Integration with Anthropic's Claude Code CLI:

```go
type ClaudeAgent struct {
    binaryPath string
    apiKey     string
}

func (a *ClaudeAgent) Run(ctx context.Context, task BenchmarkTask) (*AgentResult, error) {
    // Create isolated session directory
    sessionDir := filepath.Join(task.Workspace, "session")
    os.MkdirAll(sessionDir, 0755)

    // Spawn Claude Code CLI
    cmd := exec.CommandContext(ctx,
        a.binaryPath,
        "--session-dir", sessionDir,
        "--lang", task.Lang,
        "--task", task.Prompt,
        "--test", task.ExpectedOut,
    )

    // Capture stdout/stderr
    var stdout, stderr bytes.Buffer
    cmd.Stdout = &stdout
    cmd.Stderr = &stderr

    start := time.Now()
    err := cmd.Run()
    duration := time.Since(start)

    // Parse Claude Code output for token usage
    tokens := parseClaudeTokenUsage(stdout.String())
    turns := parseClaudeTurnCount(stdout.String())

    return &AgentResult{
        Success:     err == nil,
        Turns:       turns,
        TotalTokens: tokens,
        DurationMs:  duration.Milliseconds(),
        FinalCode:   readFinalCode(sessionDir),
    }, nil
}
```

### 4. Gemini CLI Agent

**File**: `internal/eval_harness/gemini_agent.go`

Similar pattern for Google's Gemini CLI:

```go
type GeminiAgent struct {
    binaryPath string
    apiKey     string
}

// Similar implementation to ClaudeAgent
```

### 5. Updated Metrics

**File**: `internal/eval_harness/metrics.go` (extend existing)

```go
type MultiTurnMetrics struct {
    RunMetrics              // Embed existing fields
    Turns          int      `json:"turns"`
    TokensPerTurn  []int    `json:"tokens_per_turn"`
    ErrorHistory   []string `json:"error_history"`
    AgentType      string   `json:"agent_type"` // "api-loop", "claude-code", "gemini"
}
```

---

## Implementation Plan

### Phase 1: Built-In Agent Loop (Week 1)

**Day 1-2: Core Loop**
- Implement `Agent` interface
- Build `LoopAgent` with max 5 turns
- Error feedback mechanism
- Token accumulation

**Day 3: Integration**
- Update `eval.go` to support `--agent` flag
- Wire agent selection (api, loop)
- Update metrics logger for multi-turn

**Day 4-5: Testing**
- Unit tests for agent loop
- Integration tests with mock LLM
- Test with real benchmarks (Python first)

### Phase 2: Claude Code CLI Integration (Week 2)

**Day 1-2: CLI Wrapper**
- Implement `ClaudeAgent`
- Session management
- Token usage parsing
- Error detection

**Day 3: Testing**
- Test with Claude Code installed
- Compare loop vs. Claude Code
- Measure token deltas

**Day 4-5: Documentation**
- Installation guide for Claude Code
- Usage examples
- Troubleshooting

### Phase 3: Gemini CLI + Analysis (Week 3)

**Day 1-2: Gemini Integration**
- Implement `GeminiAgent`
- Similar pattern to Claude Code

**Day 3-4: Analysis Tools**
- Enhanced reporting (turn-by-turn breakdown)
- Error pattern analysis
- Prompt improvement suggestions

**Day 5: Documentation**
- Update benchmarking guide
- Add agent comparison section
- Publish baseline results

---

## Updated CLI Commands

```bash
# Use built-in agent loop (3 turns max)
ailang eval --agent loop --benchmark fizzbuzz --model gpt-4 --max-turns 3

# Use Claude Code CLI (external)
ailang eval --agent claude-code --benchmark fizzbuzz --claude-path /usr/local/bin/claude-code

# Use Gemini CLI (external)
ailang eval --agent gemini --benchmark adt_option --gemini-path /usr/local/bin/gemini

# Compare all agents
ailang eval --agent all --benchmark fizzbuzz --seed 42

# Generate multi-turn report
make eval-report-agentic
```

---

## Expected Results Format

### Turn-by-Turn Breakdown

```json
{
  "id": "fizzbuzz",
  "lang": "ailang",
  "model": "gpt-4",
  "agent_type": "loop",
  "seed": 42,
  "turns": 3,
  "total_tokens": 450,
  "tokens_per_turn": [180, 150, 120],
  "success": true,
  "duration_ms": 2400,
  "error_history": [
    "parse error: unexpected token 'fizz' at line 3",
    "runtime error: undefined function 'mod'"
  ],
  "final_code": "...",
  "timestamp": "2025-10-02T12:34:56Z"
}
```

### Markdown Report (Extended)

```markdown
# AILANG vs Python Benchmark Results (Agentic)

**Agent**: loop (max 5 turns) | **Model**: gpt-4 | **Seed**: 42

| Benchmark | Lang | Agent | Turns | Total Tokens | Avg/Turn | Success | Time |
|-----------|------|-------|-------|--------------|----------|---------|------|
| fizzbuzz | python | loop | 2 | 440 | 220 | ✅ | 1.8s |
| fizzbuzz | ailang | loop | 3 | 450 | 150 | ✅ | 2.4s |
| fizzbuzz | python | claude | 1 | 350 | 350 | ✅ | 5.2s |
| fizzbuzz | ailang | claude | 2 | 380 | 190 | ✅ | 6.1s |

## Summary

### Token Efficiency
- **AILANG avg per turn:** 150 tokens
- **Python avg per turn:** 220 tokens
- **Reduction:** 31.8%

### Iteration Cost
- **AILANG avg turns:** 2.5
- **Python avg turns:** 1.5
- **AILANG needs more debugging** (syntax unfamiliarity)

### Cumulative Cost
- **AILANG total:** 415 tokens avg
- **Python total:** 395 tokens avg
- **Python slightly cheaper overall** (fewer iterations)

## Insights

1. **AILANG more concise per attempt** (31% fewer tokens)
2. **But AI needs more iterations** (67% more turns)
3. **Net result:** Similar total cost, slightly favoring Python
4. **Opportunity:** Better AILANG docs/examples could reduce iterations

## Error Patterns (AILANG)

- Turn 1 errors: 80% syntax, 20% logic
- Turn 2 errors: 40% syntax, 60% logic
- Turn 3 errors: 0% syntax, 100% logic

**Conclusion:** AI learns AILANG syntax by turn 3, but needs upfront examples.
```

---

## Baseline Test Strategy (M-EVAL Phase 1 Results)

**Before building M-EVAL2, we need baseline data:**

### Step 1: Run Baseline Tests (This Week)

```bash
# Set up API key
export OPENAI_API_KEY="sk-..."

# Run all 5 benchmarks, both languages
for bench in fizzbuzz json_parse pipeline cli_args adt_option; do
    ailang eval --benchmark $bench --langs python,ailang --model gpt-4 --seed 42
done

# Generate report
make eval-report
```

### Step 2: Analyze Failures

**Look for patterns:**
- What AILANG syntax errors occur?
- Which concepts confuse the AI? (ADTs? Effects? Pattern matching?)
- What Python code patterns does it try to use in AILANG?

**Example findings we expect:**
```
fizzbuzz (AILANG):
  Turn 1 error: "SyntaxError: expected 'in' after let binding"
  → AI tried: `let x = 5 x * 2`
  → Needs: `let x = 5 in x * 2`
  → Fix: Add to prompt "AILANG requires 'in' after let"

adt_option (AILANG):
  Turn 1 error: "TypeError: Option is not defined"
  → AI doesn't know AILANG has Option in stdlib
  → Fix: Add to prompt "Use stdlib/std/option for Option type"
```

### Step 3: Derive Improved Prompts

**Update benchmark prompts with AILANG hints:**

```yaml
# Before (baseline)
prompt: |
  Write a program in <LANG> that implements FizzBuzz.

# After (informed by baseline failures)
prompt: |
  Write a program in <LANG> that implements FizzBuzz.

  <LANG=AILANG> Additional context:
  - Use `let x = value in body` syntax
  - Import Option from stdlib/std/option if needed
  - Effects must be declared with ! syntax
```

### Step 4: Re-run with Improved Prompts

```bash
# Run again with v2 prompts
ailang eval --benchmark fizzbuzz --langs ailang --model gpt-4 --seed 42
```

**Compare:**
- Baseline: 180 tokens, compile error
- Improved: 220 tokens, success
- **Net gain:** 40 extra tokens in prompt, but 100% success rate

---

## Success Criteria

- ✅ Agent interface defined and implemented
- ✅ Built-in agent loop working (3-5 turns)
- ✅ Claude Code CLI integration complete
- ✅ Gemini CLI integration complete
- ✅ Multi-turn metrics collected and logged
- ✅ Turn-by-turn reports generated
- ✅ At least 3 benchmarks run with all agents
- ✅ Baseline comparison: single-shot vs. multi-turn
- ✅ Documentation with CLI installation guides

---

## Risks & Mitigation

| Risk | Impact | Mitigation |
|------|--------|------------|
| CLI tools have breaking changes | High | Version pin, test on CI |
| Token usage hard to parse from CLI output | Medium | Fall back to API logs, estimate |
| Agents timeout before success | Medium | Configurable max turns, partial credit |
| AILANG docs insufficient for AI | High | **Use M-EVAL baseline to guide doc improvements** |

---

## Dependencies

**External Tools:**
- Claude Code CLI (Anthropic) - optional
- Gemini CLI (Google) - optional
- OpenAI API key - required for built-in loop
- Python 3.8+ - required for testing

**Go Packages:**
- No new dependencies (reuse M-EVAL harness)

---

## Future Extensions

**v0.4.0:**
- Add more agents (GitHub Copilot CLI, Codex)
- Tool use support (file search, web search)
- Memory/context window optimization

**v0.5.0:**
- Fine-tuning dataset generation from successful runs
- Prompt optimization via reinforcement learning
- Continuous benchmarking in CI

---

## Key Insight: Feedback Loop

**M-EVAL → Documentation → M-EVAL2**

```
┌─────────────┐
│ M-EVAL      │  Baseline tests reveal AILANG syntax confusion
│ (Phase 1)   │
└──────┬──────┘
       │
       ▼
┌─────────────┐
│ Update Docs │  Add examples, improve guide, refine error messages
│ & Prompts   │
└──────┬──────┘
       │
       ▼
┌─────────────┐
│ M-EVAL2     │  Re-test with better context → higher success rate
│ (Phase 2)   │
└──────┬──────┘
       │
       ▼ (repeat)
```

**This creates a virtuous cycle:**
1. Baseline tests show where AI struggles
2. We improve docs/prompts/examples
3. Multi-turn tests show improvement
4. Repeat until AILANG is as easy as Python for AI

---

## Status

**M-EVAL (Phase 1):** ✅ COMPLETE (baseline single-shot)
**M-EVAL2 (Phase 2):** 📋 DESIGN COMPLETE, READY FOR IMPLEMENTATION

**Next Steps:**
1. Run baseline tests this week (all 5 benchmarks)
2. Analyze failure patterns
3. Update documentation based on findings
4. Begin M-EVAL2 implementation (v0.3.0)

---

**Parallelizable:** Yes, but benefits from M-EVAL baseline data first.
