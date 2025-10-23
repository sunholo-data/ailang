# Agent Handlers Architecture - Design Decision

**Question**: Should we merge `claude_bridge.go` with `llm_cli_handler.go`?

**Answer**: **NO** - They serve different purposes and should remain separate.

---

## Two Types of LLM Integration

### 1. **Full Agent Execution** (`claude_bridge.go`)

**Purpose**: Execute complete agent files (`.claude/agents/*.md`) with full context, tools, and state.

**Handlers**:
- `ClaudeAgentHandler` - Executes `.claude/agents/*.md` using Anthropic Agent SDK
- `SkillHandler` - Executes `.claude/skills/*` workflows
- `CommandHandler` - Executes shell commands
- `FunctionHandler` - Wraps Go functions

**Example**:
```go
// Full agent with tools, MCP, memory
handler := NewClaudeAgentHandler(
    ".claude/agents/eval-analyzer.md",  // Agent file defines behavior
    ".",
)
```

**Use when**:
- You have a `.claude/agents/*.md` file defining the agent
- Agent needs access to tools (file system, MCP servers, etc.)
- Agent maintains state across messages
- You want agentic behavior (autonomous actions, tool use, multi-step reasoning)

---

### 2. **Simple LLM Chat** (`llm_cli_handler.go`)

**Purpose**: Quick prompt-response via LLM CLI commands.

**Handlers**:
- `LLMCLIHandler` (generic) - Works with any LLM CLI
- `NewClaudeCLIHandler` - Convenience for "claude" CLI
- `NewGeminiCLIHandler` - Convenience for "gemini" CLI
- `NewOpenAICLIHandler` - Convenience for "openai" CLI

**Example**:
```go
// Simple prompt-response
handler := NewClaudeCLIHandler(
    "claude-sonnet-4-5",  // Model name
    "",                    // No agent file needed
    ".",
)
```

**Use when**:
- You just need a quick LLM response
- No agent file required
- Simple prompt → response pattern
- No tools or state needed

---

## Architecture Diagram

```
Message Arrives
    |
    v
Runner.processMessage()
    |
    v
handler.HandleMessage(msg)
    |
    +-- ClaudeAgentHandler (claude_bridge.go)
    |      |
    |      +-- Reads .claude/agents/eval-analyzer.md
    |      +-- Calls Anthropic Agent SDK
    |      +-- Agent has tools, MCP, memory
    |      +-- Returns structured response
    |
    +-- NewClaudeCLIHandler (llm_cli_handler.go)
    |      |
    |      +-- Builds prompt from message
    |      +-- Calls "claude --prompt '...' --model claude-sonnet-4-5"
    |      +-- Returns LLM response
    |
    +-- NewGeminiCLIHandler (llm_cli_handler.go)
    |      |
    |      +-- Calls "gemini --prompt '...' --model gemini-2.5-pro"
    |
    +-- NewOpenAICLIHandler (llm_cli_handler.go)
           |
           +-- Calls "openai api chat.completions.create -m gpt-4 ..."
```

---

## Comparison Table

| Feature | ClaudeAgentHandler | NewClaudeCLIHandler |
|---------|-------------------|---------------------|
| **File** | `claude_bridge.go` | `llm_cli_handler.go` |
| **Purpose** | Full agent execution | Simple LLM chat |
| **Requires agent file** | Yes (`.claude/agents/*.md`) | No (optional) |
| **Tools/MCP** | ✅ Yes | ❌ No |
| **State/Memory** | ✅ Yes | ❌ No |
| **Multi-step reasoning** | ✅ Yes | ❌ No |
| **Execution** | Anthropic Agent SDK | `claude` CLI command |
| **Complexity** | High (full agent) | Low (prompt-response) |
| **Use case** | Autonomous agents | Quick LLM queries |

---

## Real-World Examples

### Example 1: Eval Analyzer (Full Agent)

**Requirement**: Analyze eval failures, create design docs, send to sprint-planner

**Solution**: Use `ClaudeAgentHandler`

```go
// .claude/agents/eval-analyzer.md defines:
// - Tools: file system (read eval results, write design docs)
// - MCP: GitHub integration (create issues)
// - Memory: Track previous analyses
// - Multi-step: Analyze → Design → Notify

handler := NewClaudeAgentHandler(
    ".claude/agents/eval-analyzer.md",
    ".",
)

runner, _ := NewRunner(&AgentConfig{
    AgentID: "eval-analyzer",
    Handler: handler,
})
```

**Why**: Agent needs tools, state, and multi-step reasoning.

---

### Example 2: Quick Translation (Simple Chat)

**Requirement**: Translate error messages to user-friendly text

**Solution**: Use `NewClaudeCLIHandler`

```go
// No agent file needed - just translate text
handler := NewClaudeCLIHandler(
    "claude-haiku-4-5",  // Cheap model for simple task
    "",                   // No agent file
    ".",
)

runner, _ := NewRunner(&AgentConfig{
    AgentID: "translator",
    Handler: handler,
})

// Send message:
// {"action": "translate", "error": "ENOENT: no such file"}
// → "The file you're looking for doesn't exist"
```

**Why**: Simple prompt-response, no tools needed.

---

### Example 3: Multi-Model Routing

**Requirement**: Route to different LLM providers based on task complexity

**Solution**: Use `MultiModelAgentHandler` with CLI handlers

```go
multiHandler := NewMultiModelAgentHandler()

// Simple tasks → Gemini Flash (cheap)
multiHandler.RegisterHandler("simple",
    NewGeminiCLIHandler("gemini-2.5-flash", "", "."))

// Medium tasks → Claude Haiku
multiHandler.RegisterHandler("medium",
    NewClaudeCLIHandler("claude-haiku-4-5", "", "."))

// Complex tasks → Claude Sonnet
multiHandler.RegisterHandler("complex",
    NewClaudeCLIHandler("claude-sonnet-4-5", "", "."))

runner, _ := NewRunner(&AgentConfig{
    AgentID: "smart-router",
    Handler: multiHandler,
})

// Message includes model choice:
// {"model": "simple", "action": "summarize", "text": "..."}
// {"model": "complex", "action": "reason", "problem": "..."}
```

**Why**: Simple routing, no agent files needed for each model.

---

## Why Keep Them Separate?

### 1. **Different Abstractions**

- **ClaudeAgentHandler**: Agent-centric (agent files, tools, state)
- **LLMCLIHandler**: Model-centric (prompt-response, multi-provider)

Merging them would conflate two different mental models.

### 2. **Different Dependencies**

- **ClaudeAgentHandler**: Will use Anthropic Agent SDK (Python/TypeScript)
- **LLMCLIHandler**: Uses generic CLI commands (works with any provider)

### 3. **Different Use Cases**

- **ClaudeAgentHandler**: Autonomous agents, complex workflows
- **LLMCLIHandler**: Quick LLM queries, multi-provider routing

### 4. **Evolution Path**

**Future for ClaudeAgentHandler** (v0.3.20+):
- Integrate Anthropic Agent SDK
- Add streaming responses
- Support agent memory/state
- MCP server integration

**Future for LLMCLIHandler** (v0.4.0+):
- Add more providers (Ollama, LM Studio)
- HTTP endpoint support
- Batch processing
- Cost tracking

These are different feature sets that should evolve independently.

---

## Decision: Keep Both ✅

**Recommendation**: Keep `claude_bridge.go` and `llm_cli_handler.go` separate.

**Rationale**:
1. Different purposes (full agents vs simple LLM chat)
2. Different abstractions (agent-centric vs model-centric)
3. Different dependencies (Agent SDK vs CLI commands)
4. Different evolution paths (agent features vs multi-provider)

**Clarifications Added**:
- ✅ Added comments to `ClaudeAgentHandler` explaining difference
- ✅ Added comments to `NewClaudeCLIHandler` explaining difference
- ✅ Created this decision document

---

## When to Use Which?

### Use `ClaudeAgentHandler` when:

- ✅ You have a `.claude/agents/*.md` file
- ✅ Agent needs tools (file system, HTTP, MCP)
- ✅ Agent needs memory/state across messages
- ✅ You want agentic behavior (autonomous, multi-step)
- ✅ You're building autonomous workflows (eval → design → sprint → release)

### Use `NewClaudeCLIHandler` (or Gemini/OpenAI) when:

- ✅ You just need a quick LLM response
- ✅ Simple prompt-response pattern
- ✅ No tools or state needed
- ✅ You want multi-provider support (route to Claude/Gemini/OpenAI)
- ✅ You're building simple chatbots or translators

### Use `MultiModelAgentHandler` when:

- ✅ You want to route to different models based on task
- ✅ Cost optimization (cheap models for simple tasks)
- ✅ Provider redundancy (fallback chains)
- ✅ Benchmark comparison (run same task on multiple models)

---

## Summary

**Two complementary systems, not duplicates**:

1. **Full Agent System** (`claude_bridge.go`)
   - ClaudeAgentHandler, SkillHandler, CommandHandler, FunctionHandler
   - Purpose: Execute complete agents with tools and state

2. **LLM CLI System** (`llm_cli_handler.go`)
   - LLMCLIHandler (generic), Claude/Gemini/OpenAI convenience constructors
   - Purpose: Quick LLM prompt-response across multiple providers

**Both are needed** for a complete agent system.

---

**Last Updated**: October 23, 2025
**Version**: v0.3.19 (Unreleased)
