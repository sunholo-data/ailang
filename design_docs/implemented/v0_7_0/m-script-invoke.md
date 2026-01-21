# M-SCRIPT-INVOKE: Deterministic Script Execution for Agent Workflows

## Status
**Status**: IMPLEMENTED
**Target Version**: v0.6.4
**Priority**: P1 (Enables deterministic workflows)
**Estimated**: 2-3 days
**Created**: 2026-01-07
**Dependencies**: M-COORD-GENERIC-WORKFLOWS (implemented v0.6.3)

## Related Documents

- [M-COORD-GENERIC-WORKFLOWS](../../implemented/v0_6_3/m-coord-generic-workflows.md) - Established `InvokeConfig` with `skill`, `agent`, `prompt` types
- [M-UNIFIED-AI-CONTROL-PLANE](m-unified-ai-control-plane.md) - Message-based task coordination
- [M-UNIFIED-AI-PROVIDERS](../../implemented/v0_5_10/m-unified-ai-providers.md) - Provider architecture pattern

## Problem Statement

The coordinator currently supports three invoke types for agents:

```yaml
invoke:
  type: skill   # Invoke a Claude Code skill
  type: agent   # Send message to another agent
  type: prompt  # Use custom prompt template
```

**All three ultimately route to AI providers** (Claude Code CLI or Gemini CLI). This works well for creative/adaptive tasks but is:

1. **Non-deterministic** - Same input can produce different outputs
2. **Expensive** - AI tokens cost money, even for simple tasks
3. **Slow** - API latency for tasks that could run in milliseconds
4. **Overkill** - Running eval suites, deploying code, or syncing data doesn't need AI

### Use Cases Blocked

| Task | Current Approach | Problem |
|------|------------------|---------|
| Run eval baseline | AI agent runs `ailang eval-suite` | Wasteful - could be direct script |
| Deploy to staging | AI agent runs deploy script | Non-deterministic, expensive |
| Sync GitHub issues | AI agent runs `ailang messages import-github` | Overkill for cron job |
| Generate reports | AI agent runs `ailang eval-report` | Deterministic task doesn't need AI |
| Database migrations | AI figures out how to run migration | Dangerous - should be exact script |
| CI/CD pipelines | N/A (can't use coordinator) | Missing capability entirely |

### The Gap

The messaging system (`ailang messages`) provides excellent infrastructure:
- Inbox-based routing
- GitHub issue sync
- Approval workflows
- Dashboard visibility
- Task hierarchy tracking

But this infrastructure **only routes to AI providers**. Deterministic scripts cannot benefit from it.

## Goals

### Primary Goal
Add a `script` invoke type that executes shell scripts directly, with JSON payload variables, using the same messaging infrastructure as AI agents.

### Success Metrics
- [ ] Any agent inbox can be configured with `invoke.type: script`
- [ ] JSON message payload automatically becomes environment variables
- [ ] Scripts run with configurable timeout and working directory
- [ ] Output captured and parsed for markers (same as AI output)
- [ ] Approval workflow works identically to AI agents
- [ ] Mixed pipelines work: AI agent → Script agent → AI agent
- [ ] Dashboard shows script execution in real-time

## Solution Design

### Summary

Add `script` as a fourth invoke type in `InvokeConfig`:

```yaml
coordinator:
  agents:
    - id: eval-runner
      inbox: eval-runner
      workspace: /path/to/ailang
      invoke:
        type: script
        command: ./scripts/run-eval.sh    # Script to execute
        shell: /bin/bash                   # Shell to use (default: /bin/sh)
        env_from_payload: true             # Parse JSON payload as env vars
        timeout: 30m                       # Execution timeout
        working_dir: "{{.Workspace}}"      # Template support
      output_markers: ["EVAL_RESULT:", "PASS_RATE:"]
      trigger_on_complete: [report-generator]
```

### InvokeConfig Extensions

```go
// internal/coordinator/agent_registry.go

type InvokeConfig struct {
    Type     string `yaml:"type" json:"type"`         // "skill", "agent", "prompt", or "script"
    Name     string `yaml:"name" json:"name"`         // Skill/agent name (for skill/agent types)
    Template string `yaml:"template" json:"template"` // Custom template (for prompt type)

    // Script-specific fields (v0.6.4+)
    Command        string `yaml:"command" json:"command,omitempty"`                 // Script path or command
    Shell          string `yaml:"shell" json:"shell,omitempty"`                     // Shell to use (default: /bin/sh)
    EnvFromPayload bool   `yaml:"env_from_payload" json:"env_from_payload,omitempty"` // Parse JSON as env vars
    Timeout        string `yaml:"timeout" json:"timeout,omitempty"`                 // Execution timeout (e.g., "30m")
    WorkingDir     string `yaml:"working_dir" json:"working_dir,omitempty"`         // Working directory template
}
```

### Payload to Environment Variable Mapping

When `env_from_payload: true`, the JSON message payload is parsed and converted to environment variables:

```bash
# Message sent:
ailang messages send eval-runner '{"model": "gpt5", "benchmarks": "medium,hard", "parallel": true}' \
  --title "Run eval baseline v0.6.4"

# Script receives:
MODEL=gpt5
BENCHMARKS=medium,hard
PARALLEL=true
AILANG_TASK_ID=task-abc123
AILANG_MESSAGE_ID=msg-xyz789
AILANG_GITHUB_ISSUE=42
AILANG_WORKSPACE=/path/to/ailang
```

**Conversion rules:**
- Keys converted to UPPER_SNAKE_CASE
- Nested objects flattened: `{"db": {"host": "localhost"}}` → `DB_HOST=localhost`
- Arrays converted to comma-separated: `["a", "b"]` → `VALUE=a,b`
- Booleans: `true`/`false` strings
- Numbers: string representation
- Special `AILANG_*` vars injected automatically

### Script Provider Implementation

```go
// internal/coordinator/provider_script.go

type ScriptProvider struct {
    logger *log.Logger
}

func (p *ScriptProvider) Name() string {
    return "script"
}

func (p *ScriptProvider) CanHandle(task *AnalyzedTask) bool {
    // Check if the agent has script invoke type
    // This is determined at routing time, not here
    return false // Script tasks are explicitly routed
}

func (p *ScriptProvider) Execute(ctx context.Context, task *AnalyzedTask, opts *ExecuteOptions) (*ExecuteResult, error) {
    startTime := time.Now()

    // Parse invoke config from task metadata
    invokeConfig := opts.InvokeConfig
    if invokeConfig == nil || invokeConfig.Type != "script" {
        return nil, fmt.Errorf("invalid invoke config for script provider")
    }

    // Build environment from payload
    env := os.Environ()
    if invokeConfig.EnvFromPayload {
        payloadEnv, err := parsePayloadToEnv(task.Task.Content)
        if err != nil {
            return nil, fmt.Errorf("failed to parse payload: %w", err)
        }
        env = append(env, payloadEnv...)
    }

    // Add AILANG_* context variables
    env = append(env,
        fmt.Sprintf("AILANG_TASK_ID=%s", task.Task.ID),
        fmt.Sprintf("AILANG_MESSAGE_ID=%s", task.Task.MessageID),
        fmt.Sprintf("AILANG_WORKSPACE=%s", opts.Workspace),
    )

    // Determine shell and command
    shell := invokeConfig.Shell
    if shell == "" {
        shell = "/bin/sh"
    }

    // Parse timeout
    timeout := 5 * time.Minute
    if invokeConfig.Timeout != "" {
        if parsed, err := time.ParseDuration(invokeConfig.Timeout); err == nil {
            timeout = parsed
        }
    }

    // Create command with timeout context
    ctx, cancel := context.WithTimeout(ctx, timeout)
    defer cancel()

    cmd := exec.CommandContext(ctx, shell, "-c", invokeConfig.Command)
    cmd.Env = env
    cmd.Dir = opts.Workspace

    // Capture output
    var stdout, stderr bytes.Buffer
    cmd.Stdout = &stdout
    cmd.Stderr = &stderr

    // Stream output to event handler if available
    if opts.EventHandler != nil {
        cmd.Stdout = io.MultiWriter(&stdout, &streamWriter{handler: opts.EventHandler, stream: "stdout"})
        cmd.Stderr = io.MultiWriter(&stderr, &streamWriter{handler: opts.EventHandler, stream: "stderr"})
    }

    // Execute
    err := cmd.Run()
    duration := time.Since(startTime)

    result := &ExecuteResult{
        Provider: "script",
        Duration: duration,
        Output:   stdout.String(),
        Cost:     0.0, // Scripts are free!
    }

    if err != nil {
        result.Success = false
        result.Error = fmt.Sprintf("%v\nstderr: %s", err, stderr.String())
    } else {
        result.Success = cmd.ProcessState.ExitCode() == 0
        if !result.Success {
            result.Error = fmt.Sprintf("exit code %d\nstderr: %s", cmd.ProcessState.ExitCode(), stderr.String())
        }
    }

    return result, nil
}
```

### Task Executor Routing

```go
// internal/coordinator/task_executor.go

func (te *TaskExecutor) selectProvider(task *AnalyzedTask, agent *AgentConfig) Provider {
    // Check for script invoke type
    if agent != nil && agent.Invoke != nil && agent.Invoke.Type == "script" {
        return te.scriptProvider
    }

    // Existing AI provider routing...
    return te.selectAIProvider(task)
}
```

### Stage Execution Integration

```go
// internal/coordinator/stage_execution.go

func BuildDirectiveFromConfig(task *TaskRecord, agent *AgentConfig) string {
    if agent == nil {
        return task.Content
    }

    invoke := agent.GetEffectiveInvokeConfig()
    if invoke == nil {
        return task.Content
    }

    switch invoke.Type {
    case "skill":
        return buildSkillDirectiveWithConfig(task, agent, invoke)
    case "agent":
        return buildAgentHandoffDirectiveWithConfig(task, agent, invoke)
    case "prompt":
        return buildTemplateDirective(task, agent)
    case "script":
        // For scripts, the "directive" is the JSON payload
        // The actual command comes from invoke.Command
        return task.Content  // Pass through as-is
    default:
        return task.Content
    }
}
```

## Example Configurations

### 1. Eval Baseline Runner

```yaml
coordinator:
  agents:
    - id: eval-runner
      label: "Eval Baseline Runner"
      inbox: eval-runner
      workspace: /path/to/ailang
      invoke:
        type: script
        command: |
          set -e
          echo "Running eval baseline for $MODEL..."
          ailang eval-suite --models "$MODEL" --benchmarks "$BENCHMARKS"
          echo "EVAL_RESULT: success"
          echo "PASS_RATE: $(cat eval_results/summary.json | jq -r '.pass_rate')"
        env_from_payload: true
        timeout: 2h
      output_markers: ["EVAL_RESULT:", "PASS_RATE:"]
      trigger_on_complete: [dashboard-updater]
```

**Usage:**
```bash
ailang messages send eval-runner '{"model": "gpt5,claude-sonnet-4-5", "benchmarks": "all"}' \
  --title "v0.6.4 Baseline" --github
```

### 2. Staging Deployment

```yaml
coordinator:
  agents:
    - id: deploy-staging
      label: "Staging Deployer"
      inbox: deploy-staging
      workspace: /path/to/app
      invoke:
        type: script
        command: ./scripts/deploy.sh staging
        env_from_payload: true
        timeout: 15m
      approval:
        needs_label: "needs-deploy-approval"
        approved_label: "deploy-approved"
      output_markers: ["DEPLOY_URL:", "VERSION:"]
```

**Usage:**
```bash
ailang messages send deploy-staging '{"version": "v1.2.3", "rollback_on_fail": true}' \
  --title "Deploy v1.2.3 to staging" --github
```

### 3. Mixed AI + Script Pipeline

```yaml
coordinator:
  agents:
    # Stage 1: AI creates design doc
    - id: design-doc-creator
      inbox: design-doc-creator
      invoke:
        type: skill
        name: design-doc-creator
      trigger_on_complete: [lint-and-validate]

    # Stage 2: Script validates design doc format
    - id: lint-and-validate
      inbox: lint-and-validate
      invoke:
        type: script
        command: |
          set -e
          echo "Validating design doc at $DESIGN_DOC_PATH..."
          ./scripts/validate-design-doc.sh "$DESIGN_DOC_PATH"
          echo "VALIDATION_RESULT: passed"
        env_from_payload: true
        timeout: 1m
      trigger_on_complete: [sprint-planner]

    # Stage 3: AI plans sprint
    - id: sprint-planner
      inbox: sprint-planner
      invoke:
        type: skill
        name: sprint-planner
```

### 4. GitHub Issue Sync (Cron-like)

```yaml
coordinator:
  agents:
    - id: github-sync
      inbox: github-sync
      invoke:
        type: script
        command: |
          ailang messages import-github --labels "$LABELS"
          echo "SYNC_RESULT: success"
          echo "ISSUES_IMPORTED: $(ailang messages list --unread --json | jq length)"
        env_from_payload: true
        timeout: 5m
      output_markers: ["SYNC_RESULT:", "ISSUES_IMPORTED:"]
```

**Trigger periodically:**
```bash
# Could be run from cron or scheduler
ailang messages send github-sync '{"labels": "bug,feature,from:external"}' \
  --title "Periodic GitHub sync"
```

## Implementation Plan

### Phase 1: Core Script Provider (~4 hours)

- [ ] Add script fields to `InvokeConfig` struct
- [ ] Implement `ScriptProvider` with basic execution
- [ ] Add payload-to-env conversion
- [ ] Add `AILANG_*` context injection
- [ ] Unit tests for provider

**Files:**
- `internal/coordinator/agent_registry.go` (+30 lines)
- `internal/coordinator/provider_script.go` (~250 lines, new)
- `internal/coordinator/provider_script_test.go` (~200 lines, new)

### Phase 2: Task Executor Integration (~2 hours)

- [ ] Add script provider to `TaskExecutor`
- [ ] Update provider selection logic
- [ ] Handle script-type invoke config in routing
- [ ] Integration tests

**Files:**
- `internal/coordinator/task_executor.go` (+40 lines)
- `internal/coordinator/daemon_tasks.go` (+20 lines)

### Phase 3: Output Streaming (~2 hours)

- [ ] Implement `streamWriter` for real-time output
- [ ] Connect to `EventHandler` for dashboard streaming
- [ ] Add output marker parsing (reuse existing)
- [ ] Test with long-running scripts

**Files:**
- `internal/coordinator/provider_script.go` (+50 lines)
- `internal/coordinator/stream_writer.go` (~80 lines, new)

### Phase 4: Documentation & Examples (~2 hours)

- [ ] Update CLAUDE.md with script invoke type
- [ ] Add example scripts in `scripts/examples/`
- [ ] Update coordinator guide
- [ ] Add to config.yaml sample

**Files:**
- `CLAUDE.md` (+50 lines)
- `docs/docs/guides/coordinator.md` (+100 lines)
- `scripts/examples/` (new directory with examples)

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Scripts are fully deterministic given same input |
| A2: Replayability | +1 | Script execution can be replayed with captured env vars |
| A3: Effect Legibility | +1 | Effects explicit in script, captured in output |
| A4: Explicit Authority | 0 | Scripts inherit ambient authority (existing pattern) |
| A7: Machines First | +1 | JSON payloads, structured output markers |
| A9: Cost Visibility | +1 | Cost = $0.00, clearly visible in dashboard |
| A10: Composability | +1 | Scripts compose with AI agents in pipelines |

**Net Score: +6** (Accept)

## Security Considerations

### Script Source Trust
- Scripts must be in the workspace repository
- No remote script execution
- Path traversal prevented by working directory

### Environment Isolation
- Scripts run in worktree (isolated git workspace)
- No access to main repository directly
- Timeout prevents runaway processes

### Payload Validation
- JSON payload parsed, not executed
- Only string values become env vars
- No shell injection in key names (alphanumeric + underscore only)

```go
// Safe key conversion
func sanitizeEnvKey(key string) string {
    key = strings.ToUpper(key)
    key = regexp.MustCompile(`[^A-Z0-9_]`).ReplaceAllString(key, "_")
    return key
}
```

## Success Criteria

- [ ] `invoke.type: script` works for any agent inbox
- [ ] JSON payload correctly becomes environment variables
- [ ] Script output streams to dashboard in real-time
- [ ] Output markers parsed identically to AI output
- [ ] Approval workflow works (if configured)
- [ ] Mixed AI + script pipelines work end-to-end
- [ ] Timeout properly kills long-running scripts
- [ ] Exit code determines success/failure
- [ ] Cost shows $0.00 in dashboard

## Future Extensions

### Not in Scope (v0.6.4)
- Remote script execution (security concerns)
- Container isolation (complexity)
- Script library/registry (not needed yet)
- Parallel script execution (single execution per message)

### Possible Future Work
- **v0.7.0**: Script result caching (same payload = cached output)
- **v0.7.0**: Script dependency graph (run scripts in order)
- **v0.8.0**: Sandboxed execution (nsjail, firejail)
