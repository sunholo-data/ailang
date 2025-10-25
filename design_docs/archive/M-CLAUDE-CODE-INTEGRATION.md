# M-CLAUDE-CODE-INTEGRATION: Interactive ↔ Autonomous Agent Bridge

**Status**: Planned (v0.3.20)
**Created**: October 23, 2025
**Motivation**: Enable seamless handoff between interactive Claude Code sessions and autonomous AILANG agents

---

## Problem Statement

**Current limitation**: Interactive sessions and autonomous agents operate in isolation.

**What we have**:
- ✅ Agent protocol system (file-based messaging + SQLite coordination)
- ✅ Multi-model support (Claude, Gemini, OpenAI)
- ✅ Claude CLI integration for agent execution
- ✅ Polling-based autonomous agents
- ✅ Claude Code hooks documentation

**What's missing**:
- ❌ No bridge between interactive Claude Code sessions and autonomous agents
- ❌ Interactive sessions can't delegate work to autonomous agents
- ❌ Autonomous agents can't report back to interactive sessions
- ❌ No fully autonomous mode (headless Claude Code)

**Why this matters**:
The most powerful workflows combine **human-guided exploration** (interactive) with **automated execution** (autonomous):

```
[Interactive Session]
User: "Analyze eval failures and create design doc"
Claude Code: *analyzes, creates design_docs/planned/M-FIX-123.md*
Claude Code: *hooks trigger message to sprint-planner agent*

[Autonomous Agent]
sprint-planner: *reads design doc, creates sprint plan*
sprint-planner: *sends message to sprint-executor agent*

[Autonomous Agent]
sprint-executor: *implements fix with TDD*
sprint-executor: *runs tests, commits code*
sprint-executor: *sends completion message back to user*

[Interactive Session]
Claude Code: *receives notification, shows results to user*
```

---

## Background: What We've Built (v0.3.19)

### 1. Agent Protocol System

**File-based messaging** (`.ailang/state/messages/`):
- Atomic writes (temp → fsync → rename)
- Human-readable JSON for observability
- Cross-process safe (no network required)

**SQLite coordination** (`.ailang/state/agent_state.db`):
- 11 tables: agents, messages, locks, metrics, dx_reports
- WAL mode for concurrent access
- Lease-based processing (crash recovery)
- Cross-process deduplication

**Implementation**: `internal/agentprotocol/` (~1,000 LOC, 50+ tests, 82.5% coverage)

### 2. Agent Runner System

**Autonomous polling loop** (`internal/agentrunner/runner.go`):
```go
type Runner struct {
    config *AgentConfig
    db     *agentprotocol.DB
    handler MessageHandler
}

func (r *Runner) Run() error {
    ticker := time.NewTicker(r.config.PollInterval)
    for {
        select {
        case <-ticker.C:
            r.poll()  // Check for messages
        }
    }
}
```

**Handler types**:
- `ClaudeAgentHandler` - Execute `.claude/agents/*.md` files
- `LLMCLIHandler` - Generic multi-model prompt-response
- `SkillHandler` - Execute `.claude/skills/*` workflows
- `CommandHandler` - Execute shell commands
- `FunctionHandler` - Wrap Go functions

**Implementation**: `internal/agentrunner/` (~600 LOC)

### 3. Multi-Model Support

**Generic LLM CLI Handler** (`llm_cli_handler.go`):
- Works with Claude, Gemini, OpenAI, Ollama
- Template-based arguments: `{{prompt}}`, `{{model}}`, `{{format}}`
- 75% code reduction from initial approach (800 LOC → 300 LOC)

**Multi-model routing** (`MultiModelAgentHandler`):
- Route messages to different providers based on task
- Cost optimization (cheap models for simple tasks)
- Provider redundancy (fallback chains)

### 4. Claude Code Hooks (Documented, Not Yet Integrated)

**Hook events** (from https://docs.claude.com/en/docs/claude-code/hooks-guide):
- `SessionStart` - New session begins
- `UserPromptSubmit` - User sends message
- `PreToolUse` - Before tool execution
- `PostToolUse` - After tool execution
- `Notification` - Agent sends notification
- `Stop` - Session pauses/ends
- `TaskComplete` - Task finished
- `SessionEnd` - Session closes

**Configuration** (`.claude/hooks.json`):
```json
{
  "hooks": {
    "Stop": {
      "command": "bash",
      "args": ["-c", "./scripts/agent_handoff.sh"]
    },
    "PostToolUse": {
      "command": "python3",
      "args": ["./scripts/track_tool_usage.py"]
    }
  }
}
```

**Hook script receives**:
- Event type
- Session ID
- User ID
- Timestamp
- Tool use details (if applicable)
- Context from session

### 5. Headless Mode (Not Yet Used)

**From**: https://docs.claude.com/en/docs/claude-code/headless

**What it is**: Run Claude Code programmatically without interactive UI using the `claude` CLI with the `--print` (or `-p`) flag.

**Use cases**:
- CI/CD pipelines
- Scheduled tasks
- Long-running automation
- Batch processing
- SRE incident response
- Security audits

**Basic usage**:
```bash
# Non-interactive mode - prints final result
claude -p "Stage my changes and write a set of commits for them" \
  --allowedTools "Bash,Read" \
  --permission-mode acceptEdits

# With JSON output for programmatic parsing
claude -p "Analyze eval failures" --output-format json

# Resume conversation by session ID
claude --resume 550e8400-e29b-41d4-a716-446655440000 "Fix linting issues" --no-interactive

# Continue most recent conversation
claude --continue "Now refactor for better performance"
```

**Key flags for automation**:
- `--print`, `-p` - Non-interactive mode
- `--output-format` - `text`, `json`, or `stream-json`
- `--resume` - Resume conversation by session ID
- `--continue` - Continue most recent conversation
- `--allowedTools` - Whitelist specific tools
- `--disallowedTools` - Blacklist specific tools
- `--append-system-prompt` - Add custom instructions
- `--mcp-config` - Load MCP servers from JSON

**Output formats**:
```bash
# Text output (default)
claude -p "Explain this code"
# Output: This is a React component showing...

# JSON output (structured data with metadata)
claude -p "How does the data layer work?" --output-format json
# Output: {"type":"result","subtype":"success","total_cost_usd":0.003,"result":"...","session_id":"abc123"}

# Streaming JSON (each message as separate JSON object)
claude -p "Build an application" --output-format stream-json
# Output: Stream of JSON objects (init → user/assistant messages → result)
```

---

## Architecture

### Two Operating Modes

#### Mode 1: Interactive → Autonomous Handoff

**Flow**:
```
[Interactive Claude Code Session]
    User types: "Analyze eval failures"
    ↓
    Claude Code analyzes, creates design_docs/planned/M-FIX-123.md
    ↓
    Session stops (user says "looks good")
    ↓
    Stop hook fires
    ↓
    scripts/agent_handoff.sh runs:
        ./bin/send-message sprint-planner '{
          "task": "implement_design_doc",
          "design_doc": "design_docs/planned/M-FIX-123.md"
        }'
    ↓
[Autonomous sprint-planner Agent]
    Receives message
    ↓
    Reads design doc, creates sprint plan
    ↓
    Sends message to sprint-executor agent
    ↓
[Autonomous sprint-executor Agent]
    Receives message
    ↓
    Implements fix with TDD
    ↓
    Runs tests, commits code
    ↓
    Sends completion message back to user inbox
    ↓
[Interactive Session - Next Time User Opens]
    Claude Code: "While you were away, sprint-executor completed M-FIX-123"
```

#### Mode 2: Fully Autonomous (Headless)

**Flow**:
```
[Scheduled Task - No Human]
    Cron job runs:
    claude -p "Analyze eval failures in eval_results/baselines/v0.3.19 and create design docs" \
      --output-format json \
      --allowedTools "Bash,Read,Write,Grep" > /tmp/claude_result.json
    ↓
[Headless Claude Code Execution]
    Claude analyzes (no UI, non-interactive)
    ↓
    Creates design docs in design_docs/planned/
    ↓
    Exits with JSON result (status, cost, session_id)
    ↓
    Cron script checks exit code
    ↓
    scripts/auto_handoff.sh runs:
        for doc in design_docs/planned/*.md; do
            ./bin/send-message sprint-planner '{
              "task": "implement_design_doc",
              "design_doc": "$doc"
            }'
        done
    ↓
[Autonomous Agents Take Over]
    sprint-planner → sprint-executor → post-release
    ↓
[Final Notification]
    Sends message to user's inbox:
    "v0.3.20 implemented and released. See CHANGELOG.md"
```

### Key Components

#### 1. Hook Scripts (New)

**Location**: `scripts/hooks/`

**Purpose**: Bridge Claude Code events to agent messages

**Scripts**:
- `agent_handoff.sh` - Send design doc to sprint-planner on Stop
- `track_tool_usage.sh` - Log tool calls to database
- `error_notification.sh` - Send errors to monitoring agent
- `task_complete.sh` - Record metrics on TaskComplete

**Example** (`scripts/hooks/agent_handoff.sh`):
```bash
#!/bin/bash
# Hook: Stop
# Triggers when interactive session pauses

set -euo pipefail

# Find most recent design doc
DESIGN_DOC=$(ls -t design_docs/planned/*.md 2>/dev/null | head -1)

if [[ -z "$DESIGN_DOC" ]]; then
    echo "No design docs found, skipping handoff"
    exit 0
fi

echo "Handing off $DESIGN_DOC to sprint-planner agent"

# Send message to sprint-planner
./bin/send-message sprint-planner '{
  "task": "implement_design_doc",
  "design_doc": "'"$DESIGN_DOC"'",
  "priority": "high",
  "context": {
    "session_id": "'"${CLAUDE_SESSION_ID:-unknown}"'",
    "user": "'"${CLAUDE_USER_ID:-unknown}"'",
    "timestamp": "'"$(date -u +"%Y-%m-%dT%H:%M:%SZ")"'"
  }
}'

echo "Message sent successfully"
```

#### 2. User Inbox (New)

**Purpose**: Agents send messages back to users

**Location**: `.ailang/state/messages/inbox/user/`

**Message format**:
```json
{
  "to_agent": "user",
  "from_agent": "sprint-executor",
  "message_type": "task_complete",
  "correlation_id": "M-FIX-123",
  "payload": {
    "status": "completed",
    "task": "implement_design_doc",
    "design_doc": "design_docs/planned/M-FIX-123.md",
    "implemented_at": "2025-10-23T14:32:00Z",
    "commits": ["abc123", "def456"],
    "tests_passed": true,
    "summary": "Implemented M-FIX-123: Fix type inference for recursive functions"
  }
}
```

**Reading inbox** (interactive sessions):
```bash
# Check for new messages
./bin/check-inbox user

# Output:
# [2025-10-23 14:32:00] sprint-executor: Task completed - M-FIX-123
# [2025-10-23 13:15:00] eval-analyzer: Found 3 failures in v0.3.19 baseline
```

**Integration with Claude Code**:
- SessionStart hook: Check inbox, show messages to user
- Notification hook: Real-time updates from agents

#### 3. Headless Runner (New)

**Purpose**: Run Claude CLI in non-interactive mode with agent-like prompts

**Implementation**: Wrapper around `claude -p` with JSON output

**File**: `tools/run_headless_claude.sh`

```bash
#!/bin/bash
# Run Claude in headless (non-interactive) mode

set -euo pipefail

PROMPT="${1:?Usage: $0 <prompt> [output_file] [allowed_tools]}"
OUTPUT_FILE="${2:-.ailang/state/headless_output/$(date +%Y%m%d_%H%M%S).json}"
ALLOWED_TOOLS="${3:-Bash,Read,Write,Grep,Glob,Edit}"

mkdir -p "$(dirname "$OUTPUT_FILE")"

echo "Running Claude in headless mode..."
echo "Prompt: ${PROMPT:0:80}..."
echo "Output: $OUTPUT_FILE"
echo "Allowed tools: $ALLOWED_TOOLS"

# Run Claude in non-interactive mode with JSON output
claude -p "$PROMPT" \
  --output-format json \
  --allowedTools "$ALLOWED_TOOLS" \
  > "$OUTPUT_FILE" 2>&1

EXIT_CODE=$?

if [[ $EXIT_CODE -eq 0 ]]; then
    echo "✅ Claude completed successfully"

    # Extract session ID and cost from JSON
    SESSION_ID=$(jq -r '.session_id' "$OUTPUT_FILE" 2>/dev/null || echo "unknown")
    COST=$(jq -r '.total_cost_usd' "$OUTPUT_FILE" 2>/dev/null || echo "unknown")
    RESULT=$(jq -r '.result' "$OUTPUT_FILE" 2>/dev/null || echo "<no result>")

    echo "Session ID: $SESSION_ID"
    echo "Cost: \$$COST"
    echo "Result preview: ${RESULT:0:200}..."
else
    echo "❌ Claude failed with exit code $EXIT_CODE"

    # Send error notification
    ./bin/send-message user '{
      "type": "error",
      "source": "headless_claude",
      "exit_code": '"$EXIT_CODE"',
      "log_file": "'"$OUTPUT_FILE"'"
    }'
fi

exit $EXIT_CODE
```

**Alternative: Agent-style wrapper** (`tools/run_claude_agent.sh`):

```bash
#!/bin/bash
# Run Claude with agent-like behavior (read agent file, execute task)

set -euo pipefail

AGENT_FILE="${1:?Usage: $0 <agent_file> [task_description]}"
TASK="${2:-Execute the agent task}"

if [[ ! -f "$AGENT_FILE" ]]; then
    echo "Error: Agent file not found: $AGENT_FILE"
    exit 1
fi

# Read agent instructions
AGENT_INSTRUCTIONS=$(cat "$AGENT_FILE")

# Combine agent instructions with task
FULL_PROMPT="$AGENT_INSTRUCTIONS

Task: $TASK"

# Run via headless wrapper
./tools/run_headless_claude.sh "$FULL_PROMPT"
```

#### 4. Enhanced send-message CLI (Updated)

**Current** (`examples/agents/send_message.go`):
- Sends message to specific agent
- Returns message ID
- No waiting for response

**Enhanced version**:
```go
// Add flags:
//   --wait: Wait for response (poll for reply message)
//   --timeout: How long to wait (default: 60s)
//   --to-user: Send to user inbox instead of agent

func main() {
    wait := flag.Bool("wait", false, "Wait for response")
    timeout := flag.Duration("timeout", 60*time.Second, "Wait timeout")
    toUser := flag.Bool("to-user", false, "Send to user inbox")

    // ... existing code ...

    if *wait {
        // Poll for response with correlation_id
        response := pollForResponse(msgID, *timeout)
        fmt.Println("Response:", response)
    }
}
```

---

## Implementation Plan

### Phase 1: Hook Integration (Week 1)

**Goal**: Interactive sessions can trigger autonomous agents

**Tasks**:
1. Create `.claude/hooks.json` configuration
2. Implement hook scripts in `scripts/hooks/`
   - `agent_handoff.sh` (Stop hook)
   - `track_tool_usage.sh` (PostToolUse hook)
   - `session_start.sh` (SessionStart hook - check inbox)
3. Test with local Claude Code installation
4. Document setup in `docs/CLAUDE_CODE_SETUP.md`

**Success criteria**:
- ✅ Stop hook triggers message to sprint-planner
- ✅ SessionStart hook shows inbox messages to user
- ✅ PostToolUse hook logs to database

**Estimated time**: 3 days

### Phase 2: User Inbox (Week 1-2)

**Goal**: Agents can send messages back to users

**Tasks**:
1. Implement user inbox directory structure
2. Update `check-inbox` CLI to support user inbox
3. Add inbox check to SessionStart hook
4. Create notification formatting (user-friendly)
5. Test agent → user message flow

**Success criteria**:
- ✅ Agents can send messages to `user` inbox
- ✅ `check-inbox user` shows pending messages
- ✅ SessionStart hook displays new messages
- ✅ Messages marked as read after display

**Estimated time**: 2 days

### Phase 3: Headless Mode (Week 2)

**Goal**: Run Claude CLI in non-interactive mode for automation

**Tasks**:
1. Create `tools/run_headless_claude.sh` wrapper (basic)
2. Create `tools/run_claude_agent.sh` wrapper (agent-style)
3. Test with agent files (.claude/agents/*.md)
4. Add error handling and notification
5. Document headless usage with examples
6. Create example cron jobs

**Success criteria**:
- ✅ Can run `claude -p` with prompts or agent files
- ✅ JSON output captured with session_id, cost, result
- ✅ Errors sent to user inbox
- ✅ Output logged to `.ailang/state/headless_output/`
- ✅ Cron jobs work reliably

**Estimated time**: 2 days

### Phase 4: Enhanced CLI (Week 2)

**Goal**: Better tools for agent communication

**Tasks**:
1. Add `--wait` flag to `send-message`
2. Implement response polling with timeout
3. Add `--to-user` flag for user messages
4. Improve `check-inbox` formatting
5. Add `clear-inbox` command

**Success criteria**:
- ✅ `send-message --wait` blocks until response
- ✅ `send-message --to-user` sends to user inbox
- ✅ `check-inbox` shows formatted messages
- ✅ `clear-inbox` marks all as read

**Estimated time**: 2 days

### Phase 5: Integration Testing (Week 3)

**Goal**: Prove end-to-end workflows work

**Tasks**:
1. Test interactive → autonomous handoff
2. Test fully autonomous (headless + agents)
3. Test agent → user notifications
4. Test error handling and recovery
5. Measure latency and reliability

**Test scenarios**:
1. **Eval → Design → Implement**
   - Interactive: Analyze eval failures
   - Stop hook → sprint-planner
   - sprint-planner → sprint-executor
   - sprint-executor → user notification

2. **Scheduled Baseline**
   - Cron: Run eval-analyzer headless
   - Agent creates design docs
   - SessionEnd hook → sprint-planner
   - Full autonomous implementation

3. **Error Recovery**
   - Agent fails during implementation
   - Error message to user inbox
   - User reviews in next session
   - User manually fixes and restarts

**Success criteria**:
- ✅ All 3 test scenarios complete successfully
- ✅ No message loss
- ✅ Latency < 5 seconds for handoff
- ✅ Errors properly reported

**Estimated time**: 3 days

---

## Use Cases

### Use Case 1: Development Workflow Automation

**Scenario**: User analyzes eval failures, agents implement fixes

**Steps**:
1. **[Interactive]** User: "Analyze eval_results/baselines/v0.3.19"
2. **[Interactive]** Claude Code analyzes, creates `design_docs/planned/M-FIX-*.md` (3 docs)
3. **[Interactive]** User: "Looks good" (session stops)
4. **[Hook]** Stop hook triggers `agent_handoff.sh`
5. **[Autonomous]** sprint-planner receives 3 messages, prioritizes
6. **[Autonomous]** sprint-executor implements fixes with TDD
7. **[Autonomous]** post-release runs tests, updates dashboard
8. **[Notification]** User inbox: "3 fixes implemented, all tests passing"
9. **[Interactive]** Next session: User reviews commits, approves PR

**Benefits**:
- Human guides direction (what to fix)
- Agents handle execution (how to fix)
- User stays in the loop (notifications)
- Faster iteration (agents work while user sleeps)

### Use Case 2: Continuous Improvement Pipeline

**Scenario**: Scheduled eval baselines trigger autonomous improvements

**Steps**:
1. **[Cron]** Daily 3am: `./tools/run_claude_agent.sh .claude/agents/eval-analyzer.md "Analyze v0.3.$(date +%Y%m%d)"`
2. **[Headless]** Claude runs in non-interactive mode (no UI)
3. **[Headless]** Analyzes eval failures, creates design docs in `design_docs/planned/`
4. **[Headless]** Exits with JSON result (status, cost, session_id)
5. **[Cron Script]** Checks exit code, runs `scripts/auto_handoff.sh` on success
6. **[Autonomous]** sprint-planner receives design docs via messages
7. **[Autonomous]** sprint-executor implements fixes
8. **[Autonomous]** release-manager creates PR
9. **[Notification]** User inbox: "Daily baseline complete. 2 fixes proposed. Review PR #123"
10. **[Interactive]** Morning: User reviews PR, merges or adjusts

**Benefits**:
- Continuous monitoring (no manual baseline runs)
- Proactive fixes (failures addressed immediately)
- Human oversight (user approves before merge)
- Time-efficient (uses off-hours compute)
- Cost tracking (JSON output includes `total_cost_usd`)

### Use Case 3: Real-Time Tool Monitoring

**Scenario**: Track tool usage for cost optimization

**Steps**:
1. **[Interactive]** User works on implementing new feature
2. **[Hook]** PostToolUse hook fires after each tool call
3. **[Script]** `track_tool_usage.sh` logs to database:
   - Tool name (Read, Write, Bash, WebFetch)
   - Duration
   - Input/output size
   - Success/failure
4. **[Database]** SQLite `tool_usage` table aggregates data
5. **[Weekly]** Scheduled report agent analyzes patterns:
   - Most expensive tools
   - Optimization opportunities
   - Failure patterns
6. **[Notification]** User inbox: "Weekly tool usage report: 234 tool calls, $0.42 total cost"

**Benefits**:
- Visibility into costs
- Identify optimization opportunities
- Detect tool failures
- Track development patterns

### Use Case 4: Automated PR Security Review

**Scenario**: Security bot reviews PRs before merge (inspired by Claude headless docs)

**Steps**:
1. **[GitHub Action]** PR created → webhook triggers review script
2. **[Script]** Fetches PR diff: `gh pr diff 123`
3. **[Headless]** Runs security review:
   ```bash
   gh pr diff 123 | claude -p \
     --append-system-prompt "You are a security engineer. Review this PR for vulnerabilities, insecure patterns, and compliance issues." \
     --output-format json \
     --allowedTools "Read,Grep,WebSearch" \
     > security-report.json
   ```
4. **[Script]** Parses JSON result, extracts findings
5. **[Script]** Posts findings as PR comment
6. **[Notification]** User inbox: "PR #123 security review complete. 2 issues found."
7. **[Interactive]** Developer reviews issues, fixes code
8. **[Automated]** Re-run review on push, verify fixes

**Benefits**:
- Automated security checks (no manual review for every PR)
- Consistent standards (same prompt, same model)
- Fast feedback (seconds, not hours)
- Cost tracking (JSON includes `total_cost_usd`)
- Audit trail (JSON logs all findings)

### Use Case 5: Error Recovery with Human-in-Loop

**Scenario**: Agent fails during implementation, user intervenes

**Steps**:
1. **[Autonomous]** sprint-executor implementing M-FIX-123
2. **[Error]** Tests fail after 3 retry attempts
3. **[Notification]** Agent sends to user inbox:
   ```json
   {
     "type": "error",
     "task": "M-FIX-123",
     "attempts": 3,
     "error": "Type mismatch in recursive call",
     "suggestion": "Manual review needed - complex type inference issue"
   }
   ```
4. **[Interactive]** Next session: User sees error message
5. **[Interactive]** User reviews code, sees the issue
6. **[Interactive]** User: "Add type annotation to recursive call"
7. **[Interactive]** Claude Code fixes, commits
8. **[Hook]** Stop hook sends updated context to sprint-executor
9. **[Autonomous]** sprint-executor retries with new context, succeeds
10. **[Notification]** User inbox: "M-FIX-123 completed successfully"

**Benefits**:
- Graceful degradation (agent tries, user helps)
- Context preservation (agent learns from user fix)
- Hybrid workflow (automated + manual)
- Knowledge capture (fix becomes training data)

---

## Technical Decisions

### Decision 1: Hooks vs Polling for Interactive → Autonomous

**Options**:
- A) Hooks trigger messages (chosen)
- B) Agents poll Claude Code state
- C) Webhook server

**Chosen**: A) Hooks trigger messages

**Rationale**:
- ✅ Native Claude Code integration
- ✅ No additional infrastructure (no webhook server)
- ✅ Low latency (immediate trigger)
- ✅ Simple configuration (.claude/hooks.json)
- ❌ Requires Claude Code installation (acceptable - dev environment)

### Decision 2: User Inbox vs Database Notifications

**Options**:
- A) User inbox directory (chosen)
- B) Database table
- C) Email/SMS

**Chosen**: A) User inbox directory

**Rationale**:
- ✅ Consistent with agent message format
- ✅ Observable (can inspect files)
- ✅ Persistent (survives restarts)
- ✅ Simple (no email config)
- ✅ Can add database index later for fast queries

### Decision 3: Headless Wrapper vs Direct Integration

**Options**:
- A) Shell wrapper (chosen)
- B) Go binary that calls claude-code
- C) Integrate Claude Code SDK

**Chosen**: A) Shell wrapper

**Rationale**:
- ✅ Simple (bash script)
- ✅ Flexible (easy to customize)
- ✅ Logging built-in (tee to file)
- ✅ No Go dependencies
- ❌ Less portable (requires bash) - acceptable for dev environment

### Decision 4: Synchronous vs Asynchronous Handoff

**Options**:
- A) Asynchronous (hook sends message, returns immediately) - chosen
- B) Synchronous (hook waits for agent response)

**Chosen**: A) Asynchronous

**Rationale**:
- ✅ Interactive session not blocked
- ✅ User can continue working
- ✅ Multiple agents can work in parallel
- ✅ Notifications when complete
- ❌ Can't show immediate results - acceptable, agents take time anyway

---

## Configuration

### .claude/hooks.json

```json
{
  "version": "1.0",
  "hooks": {
    "SessionStart": {
      "command": "bash",
      "args": ["-c", "./scripts/hooks/session_start.sh"],
      "description": "Check inbox on session start"
    },
    "Stop": {
      "command": "bash",
      "args": ["-c", "./scripts/hooks/agent_handoff.sh"],
      "description": "Hand off design docs to sprint-planner"
    },
    "PostToolUse": {
      "command": "bash",
      "args": ["-c", "./scripts/hooks/track_tool_usage.sh"],
      "description": "Log tool usage to database"
    },
    "TaskComplete": {
      "command": "bash",
      "args": ["-c", "./scripts/hooks/task_complete.sh"],
      "description": "Record task metrics"
    },
    "Notification": {
      "command": "bash",
      "args": ["-c", "./scripts/hooks/notification_handler.sh"],
      "description": "Handle agent notifications"
    }
  },
  "settings": {
    "timeout_seconds": 30,
    "retry_on_failure": true,
    "log_dir": ".ailang/state/hook_logs"
  }
}
```

### Crontab for Scheduled Agents

```bash
# Daily eval baseline at 3am
0 3 * * * cd /path/to/ailang && ./tools/run_claude_agent.sh .claude/agents/eval-analyzer.md "Analyze eval_results/baselines/v0.3.$(date +\%Y\%m\%d)"

# Weekly performance report on Mondays at 9am
0 9 * * 1 cd /path/to/ailang && ./tools/run_headless_claude.sh "Generate weekly performance report for the past 7 days"

# Hourly DX friction checks (detect pain points)
0 * * * * cd /path/to/ailang && ./tools/run_headless_claude.sh "Check for DX friction points in recent commits and tool usage"
```

### Environment Variables

```bash
# .envrc (for direnv)
export AILANG_STATE_DIR=".ailang/state"
export AILANG_AGENT_POLL_INTERVAL="2s"
export AILANG_HOOK_TIMEOUT="30s"
export CLAUDE_SESSION_ID="${CLAUDE_SESSION_ID:-unknown}"
export CLAUDE_USER_ID="${CLAUDE_USER_ID:-$(whoami)}"
```

---

## Success Metrics

### Primary Metrics

**1. Handoff Latency**
- **Target**: < 5 seconds from Stop hook to message received
- **Measure**: Time from hook trigger to agent poll
- **Baseline**: N/A (new feature)

**2. Message Reliability**
- **Target**: 100% message delivery (no loss)
- **Measure**: Messages sent vs messages received
- **Baseline**: 100% (proven in integration tests)

**3. End-to-End Workflow Time**
- **Target**: < 30 minutes for simple fix (eval failure → design → implement → test)
- **Measure**: Timestamp from design doc creation to PR creation
- **Baseline**: N/A (currently manual, ~2-4 hours)

**4. User Satisfaction**
- **Target**: 80% of workflows complete without human intervention
- **Measure**: Tasks completed autonomously vs requiring human help
- **Baseline**: 0% (no autonomous workflows yet)

### Secondary Metrics

**5. Hook Success Rate**
- **Target**: > 95% hooks execute successfully
- **Measure**: Successful hook executions / total triggers
- **Monitor**: Hook logs in `.ailang/state/hook_logs/`

**6. Headless Agent Stability**
- **Target**: > 90% headless runs complete without crash
- **Measure**: Successful runs / total runs
- **Monitor**: Exit codes, error logs

**7. Inbox Check Rate**
- **Target**: Users check inbox at least once per session
- **Measure**: SessionStart hook executions
- **Monitor**: Hook logs

---

## Security & Safety

### Concerns

**1. Hook Script Execution**
- Hooks run arbitrary shell commands
- Potential for malicious scripts

**Mitigation**:
- ✅ Hooks in version control (`.claude/hooks.json`)
- ✅ Script signing (optional, future)
- ✅ Timeout enforcement (30s default)
- ✅ Sandboxing (run in restricted directory)

**2. Headless Agent Access**
- Headless agents have full CLI access
- Could make unintended changes

**Mitigation**:
- ✅ Agent files reviewed and version controlled
- ✅ Logs all actions to `.ailang/state/headless_output/`
- ✅ Dry-run mode for testing
- ✅ Capability restrictions (agents declare required capabilities)

**3. User Inbox Spoofing**
- Malicious agents could send fake notifications

**Mitigation**:
- ✅ Message signing (HMAC, Phase 2 of agent protocol)
- ✅ Agent registration (only known agents can send)
- ✅ User inbox separate from agent inboxes

### Best Practices

**For Hook Scripts**:
1. Always use `set -euo pipefail` (fail fast)
2. Validate inputs (check file existence, etc.)
3. Log all actions for audit trail
4. Handle errors gracefully (don't crash Claude Code)
5. Use absolute paths (avoid path confusion)

**For Headless Agents**:
1. Test in dry-run mode first
2. Log all actions with timestamps
3. Send summary to user inbox on completion
4. Use timeouts (prevent runaway processes)
5. Monitor resource usage (CPU, memory)

**For User Inbox**:
1. Mark messages as read after display
2. Archive old messages (keep last 30 days)
3. Rate limit notifications (max 10/hour)
4. Format messages for readability
5. Include correlation IDs for debugging

---

## Migration Path

### For Existing Users (v0.3.19 → v0.3.20)

**No breaking changes**:
- Agent protocol system works as-is
- Hooks are optional (opt-in)
- Headless mode is optional

**To enable hooks**:
1. Create `.claude/hooks.json` (copy template)
2. Create `scripts/hooks/` directory
3. Copy hook scripts from `examples/hooks/`
4. Test with `claude-code --test-hooks`

**To enable headless**:
1. Install Claude CLI if not already installed (`npm install -g @anthropic-ai/claude-code`)
2. Test: `claude -p "Test message" --output-format json`
3. Test with agent: `./tools/run_claude_agent.sh .claude/agents/echo-agent.md "Test task"`
4. Create cron jobs if desired

**To enable user inbox**:
1. No config needed (automatically created)
2. Check inbox: `./bin/check-inbox user`
3. SessionStart hook will display messages

### For New Users (v0.3.20+)

**Recommended setup**:
1. Install AILANG: `make install`
2. Install Claude Code: `npm install -g @anthropic-ai/claude-code`
3. Configure hooks: `cp examples/hooks/hooks.json .claude/`
4. Start first session: Claude Code will set up everything

---

## Future Work (v0.4.0+)

### Enhancement 1: Agent Marketplace

**Vision**: Agents discover and invoke other agents dynamically

**Example**:
```ailang
// agent registry
let agents = discovery.list_agents()
// → [{id: "sprint-planner", capabilities: ["planning", "estimation"]}]

let agent = discovery.find_agent("planning")
let result = agent.invoke({task: "plan_sprint", doc: "M-FIX-123"})
```

### Enhancement 2: Multi-Agent Collaboration

**Vision**: Agents coordinate on complex tasks

**Example**:
```
[eval-analyzer] → [design-doc-creator] → [sprint-planner] → [sprint-executor]
                                                ↓
                                          [code-reviewer]
                                                ↓
                                          [test-generator]
```

### Enhancement 3: Agent Learning

**Vision**: Agents learn from failures and user corrections

**Example**:
```
[sprint-executor] fails to implement fix
[user] provides corrected implementation
[learning-agent] extracts pattern: "recursive functions need type annotations"
[knowledge-base] stores pattern for future use
[sprint-executor] applies pattern in next attempt
```

### Enhancement 4: Cross-Repository Agents

**Vision**: Agents work across multiple AILANG projects

**Example**:
```
# Global agent registry in ~/.ailang/agents/
# Agents can send messages to any project's inbox
./bin/send-message --project ~/dev/other-project sprint-planner '{...}'
```

---

## Real-World Headless Examples

### Example 1: SRE Incident Response Bot

**From Claude headless docs** - Automated incident investigation:

```bash
#!/bin/bash
# scripts/sre_incident_bot.sh

investigate_incident() {
    local incident_description="$1"
    local severity="${2:-medium}"

    claude -p "Incident: $incident_description (Severity: $severity)" \
      --append-system-prompt "You are an SRE expert. Diagnose the issue, assess impact, and provide immediate action items." \
      --output-format json \
      --allowedTools "Bash,Read,WebSearch,mcp__datadog" \
      --mcp-config monitoring-tools.json \
      > "/tmp/incident_$(date +%s).json"

    # Parse and alert
    jq -r '.result' "/tmp/incident_$(date +%s).json"
}

# Usage:
investigate_incident "Payment API returning 500 errors" "high"
```

### Example 2: Multi-Turn Legal Document Review

**From Claude headless docs** - Session persistence for complex tasks:

```bash
#!/bin/bash
# scripts/legal_review.sh

# Start session
session_id=$(claude -p "Start legal review session" --output-format json | jq -r '.session_id')

# Review in multiple steps (maintains context)
claude --resume "$session_id" -p "Review contract.pdf for liability clauses"
claude --resume "$session_id" -p "Check compliance with GDPR requirements"
claude --resume "$session_id" -p "Generate executive summary of risks"

echo "Review complete. Session ID: $session_id"
```

### Example 3: Streaming JSON for Real-Time Updates

**Use streaming JSON when you need to process output as it arrives**:

```bash
#!/bin/bash
# scripts/stream_analysis.sh

# Stream analysis results
claude -p "Analyze this codebase for performance issues" \
  --output-format stream-json \
  --allowedTools "Bash,Read,Grep" | \
while IFS= read -r line; do
    # Process each message as it arrives
    msg_type=$(echo "$line" | jq -r '.type')

    case "$msg_type" in
        "init")
            echo "Starting analysis..."
            ;;
        "assistant")
            content=$(echo "$line" | jq -r '.message.content[0].text')
            echo "Progress: $content"
            ;;
        "result")
            echo "Analysis complete!"
            cost=$(echo "$line" | jq -r '.total_cost_usd')
            echo "Total cost: \$$cost"
            ;;
    esac
done
```

### Example 4: AILANG Eval Baseline with Cost Tracking

**AILANG-specific** - Run eval baseline, track costs, send results to agents:

```bash
#!/bin/bash
# scripts/cron/nightly_eval_baseline.sh

set -euo pipefail

VERSION="v0.3.$(date +%Y%m%d)"
OUTPUT_DIR="eval_results/baselines/$VERSION"

# Run eval baseline via Claude
claude -p "Run full eval baseline for $VERSION and analyze failures" \
  --output-format json \
  --allowedTools "Bash,Read,Write,Grep,Glob" \
  --append-system-prompt "You are an AILANG eval expert. Run: make eval-baseline EVAL_VERSION=$VERSION FULL=true" \
  > "/tmp/eval_baseline_$VERSION.json"

EXIT_CODE=$?

if [[ $EXIT_CODE -eq 0 ]]; then
    # Extract cost and results
    COST=$(jq -r '.total_cost_usd' "/tmp/eval_baseline_$VERSION.json")
    DURATION=$(jq -r '.duration_ms' "/tmp/eval_baseline_$VERSION.json")
    SESSION_ID=$(jq -r '.session_id' "/tmp/eval_baseline_$VERSION.json")

    echo "✅ Eval baseline complete"
    echo "Version: $VERSION"
    echo "Cost: \$$COST"
    echo "Duration: ${DURATION}ms"
    echo "Session: $SESSION_ID"

    # Send success message to eval-analyzer agent
    ./bin/send-message eval-analyzer '{
      "type": "baseline_complete",
      "version": "'"$VERSION"'",
      "output_dir": "'"$OUTPUT_DIR"'",
      "cost_usd": '"$COST"',
      "duration_ms": '"$DURATION"',
      "session_id": "'"$SESSION_ID"'"
    }'
else
    echo "❌ Eval baseline failed with exit code $EXIT_CODE"

    # Send error to user inbox
    ./bin/send-message user '{
      "type": "error",
      "source": "nightly_eval_baseline",
      "version": "'"$VERSION"'",
      "exit_code": '"$EXIT_CODE"',
      "log_file": "/tmp/eval_baseline_'"$VERSION"'.json"
    }'
fi
```

**Why these examples matter**:
- Show real-world patterns from Claude docs
- Demonstrate JSON output parsing
- Show session persistence for complex tasks
- Show streaming for real-time feedback
- AILANG-specific integration with agent protocol

---

## Appendix: File Changes

### New Files

```
.claude/hooks.json                          # Hook configuration
scripts/hooks/session_start.sh              # Check inbox on start
scripts/hooks/agent_handoff.sh              # Hand off to agents on stop
scripts/hooks/track_tool_usage.sh           # Log tool usage
scripts/hooks/task_complete.sh              # Record task metrics
scripts/hooks/notification_handler.sh       # Handle notifications
scripts/auto_handoff.sh                     # Auto handoff after headless runs
tools/run_headless_claude.sh                # Basic headless wrapper (prompt → JSON)
tools/run_claude_agent.sh                   # Agent-style wrapper (agent file + task)
docs/CLAUDE_CODE_SETUP.md                   # Setup guide
docs/HEADLESS_AGENTS.md                     # Headless usage guide + examples
examples/hooks/                             # Example hook scripts
examples/cron/                              # Example cron jobs
```

### Modified Files

```
examples/agents/send_message.go             # Add --wait, --to-user flags
examples/agents/check_inbox.go              # Support user inbox
internal/agentprotocol/protocol.go          # Add user inbox logic
CHANGELOG.md                                # Add v0.3.20 entry
CLAUDE.md                                   # Update with hooks workflow
```

### Lines of Code Estimate

```
Hook scripts:        ~500 LOC (5 scripts × 100 lines)
Headless wrapper:    ~150 LOC
CLI enhancements:    ~200 LOC
Documentation:       ~1,500 LOC
Tests:               ~300 LOC
Configuration:       ~100 LOC
-----------------------------------------------------
Total:               ~2,750 LOC
```

---

## Summary

**What**: Bridge between interactive Claude Code sessions and autonomous AILANG agents

**Why**: Enable hybrid workflows (human guidance + autonomous execution)

**How**:
- Claude Code hooks trigger messages to agents
- Agents work autonomously and notify users
- Headless mode for fully autonomous operation

**When**: v0.3.20 (estimated 2 weeks)

**Success**:
- Interactive → Autonomous handoff works
- Agents complete tasks without human intervention
- Users receive notifications of completed work
- End-to-end latency < 30 minutes for simple fixes

**Risk**: Low - builds on proven agent protocol, hooks are well-documented, headless mode exists

**Dependencies**:
- ✅ Agent protocol system (complete, v0.3.19)
- ✅ Claude Code installation (developer environment)
- ✅ Hooks API (documented by Anthropic)
- ✅ Headless mode (exists in Claude Code)

---

**Created**: October 23, 2025
**Target Version**: v0.3.20
**Estimated Time**: 2 weeks (10 days development)
**Dependencies**: M-AGENT-PROTOCOL (complete, v0.3.19)
**Related**: M-EVAL-AGENT (v0.4.0), M-DX series

**Next Steps**:
1. User review and approval of design doc
2. Phase 1 implementation (hook integration)
3. Phase 2 implementation (user inbox)
4. Testing and validation
5. Documentation and examples
