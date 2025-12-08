# M-BACKGROUND-AGENT-DAEMON: Background Agent Service

**Status:** PLANNED
**Version:** v0.5.0
**Priority:** HIGH
**Estimated Effort:** 5-7 days
**Prerequisites:** v0.4.5 (Agent Execution Integration - DONE)

## Summary

Enable `ailang-agent` to run as a background service that:
1. Polls for messages every 10 minutes (configurable)
2. Executes directives in a project directory (loads `.claude/` settings)
3. Reports status to the existing `ailang serve` Collaboration Hub UI
4. Runs automatically via launchd (Mac) / systemd (Linux)

## What Already Works

| Component | Status | Location |
|-----------|--------|----------|
| Agent polling loop | ✅ v0.4.5 | `cmd/ailang-agent/agent.go` |
| Message claiming | ✅ v0.4.5 | `messaging.Client.ClaimMessage()` |
| Claude Code execution | ✅ v0.4.5 | `internal/agent/executor.go` |
| Approval flow | ✅ v0.4.5 | `internal/agent/capability.go` |
| Collaboration Hub UI | ✅ v0.4.4 | `ailang serve` → `localhost:1956` |
| Messages/Approvals tabs | ✅ v0.4.4 | `ui/src/components/` |

**Both share:** `~/.ailang/state/collaboration.db`

## What's Missing

| Feature | Why Needed |
|---------|------------|
| Config file | Persistent settings without CLI flags |
| 10-minute poll | 2s is too aggressive for background |
| Project-dir execution | Load `.claude/` settings (skills, commands) |
| Agent Status tab | See agent activity in UI |
| launchd service | Run unattended on Mac |

## Design

### Architecture

```
                      ┌─────────────────────────────────────┐
                      │     ailang serve (port 1956)        │
                      │  ┌─────────────────────────────────┐│
                      │  │         React UI                ││
                      │  │ [Messages] [Approvals] [Agents] ││
                      │  └─────────────────────────────────┘│
                      └──────────────────┬──────────────────┘
                                         │
                                         │ WebSocket + REST
                                         │
                      ┌──────────────────┴──────────────────┐
                      │        collaboration.db             │
                      │   (threads, messages, approvals,    │
                      │    agent_status)  ← NEW table       │
                      └──────────────────┬──────────────────┘
                                         │
                      ┌──────────────────┴──────────────────┐
                      │      ailang-agent (daemon)          │
                      │                                     │
                      │  - Polls every 10 minutes           │
                      │  - Executes in project directory    │
                      │  - Reports status to DB             │
                      │  - Runs via launchd                 │
                      └─────────────────────────────────────┘
```

### Config File (`~/.ailang/agent.yml`)

```yaml
# ~/.ailang/agent.yml
instance_id: background-agent
poll_interval: 10m

# Workspace configuration
workspace:
  mode: isolated            # 'in-place' or 'isolated'
  source_dir: /Users/mark/dev/sunholo/ailang  # Project to work on

  # For 'isolated' mode: copy these into fresh workspace
  copy_configs:
    - .claude/              # Claude Code settings, skills, commands
    - CLAUDE.md             # Project instructions
    - .gitignore            # Optional: git config

executor:
  type: claude-code         # 'claude-code', 'gemini', 'codex'
  model: claude-sonnet-4-5
  max_turns: 50
  timeout: 5m

cost:
  daily_cap_usd: 10.0
```

### Workspace Modes

The agent supports two workspace modes for flexibility:

#### Mode 1: `in-place` - Execute Directly in Project

```yaml
workspace:
  mode: in-place
  source_dir: /Users/mark/dev/sunholo/ailang
```

```
source_dir/
├── .claude/           ← Claude Code reads settings from here
├── CLAUDE.md          ← Project instructions loaded
├── src/               ← Agent modifies files directly
└── ...
```

**Pros:**
- Full project context available
- Changes persist immediately
- Git history preserved

**Cons:**
- Agent modifies real project files
- Could break things if directive fails mid-way
- Must trust the directive

**Use case:** Trusted directives, working on your own project

#### Mode 2: `isolated` - Fresh Workspace with Config Copy

```yaml
workspace:
  mode: isolated
  source_dir: /Users/mark/dev/sunholo/ailang
  copy_configs:
    - .claude/
    - CLAUDE.md
```

```
source_dir/                    temp_workspace/
├── .claude/        ──copy──►  ├── .claude/
├── CLAUDE.md       ──copy──►  ├── CLAUDE.md
├── src/                       ├── .git/  (fake, for isolation)
└── ...                        └── (agent creates files here)
```

**Pros:**
- Safe - doesn't modify real project
- Can review changes before applying
- Good for untrusted or experimental directives

**Cons:**
- Less project context (only copied files)
- Must manually copy results back
- Workspace deleted after execution (unless `DEBUG_AGENT=1`)

**Use case:** Untrusted directives, experiments, code generation tasks

#### Executor-Agnostic Pattern

This workspace pattern applies to all executors:

| Executor | Config Location | Settings Loaded |
|----------|-----------------|-----------------|
| `claude-code` | `.claude/` | `settings.json`, skills, commands, `CLAUDE.md` |
| `gemini` | `.gemini/` | TBD (future) |
| `codex` | `.codex/` | TBD (future) |

The executor interface handles mode selection:

```go
type Executor interface {
    Name() string
    ConfigDir() string  // ".claude/", ".gemini/", etc.
    Execute(ctx context.Context, req *ExecutionRequest) (*ExecutionResult, error)
}

type ExecutionRequest struct {
    Directive    string
    WorkspaceMode string           // "in-place" or "isolated"
    SourceDir    string            // Project directory
    CopyConfigs  []string          // Files to copy in isolated mode
}
```

### CLI Usage

```bash
# Start agent (foreground, for testing)
ailang-agent --config ~/.ailang/agent.yml

# Install as launchd service
ailang-agent install

# Manage service
ailang-agent start
ailang-agent stop
ailang-agent status

# Open Collaboration Hub UI
ailang serve
```

### Agent Status Tab (New in UI)

Add third tab to existing Collaboration Hub:

```
┌─────────────────────────────────────────────────────────────────────┐
│  🤖 AILANG Collaboration Hub                                        │
│  [💬 Messages] [🔒 Approvals] [📊 Agents]  ← NEW                    │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  Agent: background-agent                                            │
│  Status: ● Running                                                  │
│  Project: /Users/mark/dev/sunholo/ailang                           │
│  Last Poll: 2 minutes ago                                          │
│  Next Poll: in 8 minutes                                           │
│                                                                     │
│  Today's Activity                                                   │
│  ────────────────                                                   │
│  Processed: 5 messages                                              │
│  Cost: $1.23 / $10.00 cap                                          │
│                                                                     │
│  Recent Executions                                                  │
│  ─────────────────                                                  │
│  14:30 ✓ "Implement type alias"     $0.42   18 turns               │
│  14:15 ✓ "Fix parser bug"           $0.28   12 turns               │
│  13:45 ✗ "Run tests" (timeout)      $0.55   50 turns               │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

## Implementation

### Phase 1: Config & Workspace Modes (2-3 days)

**New file:** `cmd/ailang-agent/config.go`

```go
type Config struct {
    InstanceID   string        `yaml:"instance_id"`
    PollInterval time.Duration `yaml:"poll_interval"`

    Workspace struct {
        Mode        string   `yaml:"mode"`         // "in-place" or "isolated"
        SourceDir   string   `yaml:"source_dir"`   // Project directory
        CopyConfigs []string `yaml:"copy_configs"` // Files to copy in isolated mode
    } `yaml:"workspace"`

    Executor struct {
        Type     string        `yaml:"type"`     // "claude-code", "gemini", "codex"
        Model    string        `yaml:"model"`
        MaxTurns int           `yaml:"max_turns"`
        Timeout  time.Duration `yaml:"timeout"`
    } `yaml:"executor"`

    Cost struct {
        DailyCapUSD float64 `yaml:"daily_cap_usd"`
    } `yaml:"cost"`
}

func LoadConfig(path string) (*Config, error)
```

**New file:** `internal/agent/workspace.go`

```go
// WorkspaceManager handles workspace setup for both modes
type WorkspaceManager struct {
    baseDir string  // Where to create temp workspaces
}

// PrepareWorkspace sets up the execution workspace based on mode
func (w *WorkspaceManager) PrepareWorkspace(mode, sourceDir string, copyConfigs []string) (workDir string, cleanup func(), err error) {
    switch mode {
    case "in-place":
        // Execute directly in source directory
        return sourceDir, func() {}, nil

    case "isolated":
        // Create temp workspace and copy configs
        workDir = filepath.Join(w.baseDir, fmt.Sprintf("workspace_%s", uuid.New().String()[:8]))
        os.MkdirAll(workDir, 0755)

        // Copy specified config files/dirs from source
        for _, config := range copyConfigs {
            src := filepath.Join(sourceDir, config)
            dst := filepath.Join(workDir, config)
            if err := copyPath(src, dst); err != nil {
                // Warn but don't fail - config may not exist
                log.Printf("Warning: could not copy %s: %v", config, err)
            }
        }

        // Create fake .git for isolation
        os.MkdirAll(filepath.Join(workDir, ".git"), 0755)

        cleanup = func() {
            if os.Getenv("DEBUG_AGENT") == "" {
                os.RemoveAll(workDir)
            }
        }
        return workDir, cleanup, nil

    default:
        return "", nil, fmt.Errorf("unknown workspace mode: %s", mode)
    }
}

// copyPath copies a file or directory
func copyPath(src, dst string) error
```

**Modify:** `internal/agent/executor.go`

```go
// ExecuteWithWorkspace executes directive with workspace mode support
func (e *DirectiveExecutor) ExecuteWithWorkspace(directive string, workDir string) (*DirectiveResult, error) {
    spec := &eval_harness.BenchmarkSpec{
        ID:      "directive_" + uuid.New().String()[:8],
        Timeout: e.config.TimeoutSeconds,
    }

    result, err := eval_harness.RunHeadlessSessionStreaming(
        spec, "", directive, workDir, e.config,
    )
    if err != nil {
        return nil, err
    }

    return convertResult(result, workDir), nil
}
```

**Usage in agent:**

```go
// In agent.go processMessage()
workDir, cleanup, err := a.workspaceManager.PrepareWorkspace(
    a.config.Workspace.Mode,
    a.config.Workspace.SourceDir,
    a.config.Workspace.CopyConfigs,
)
if err != nil {
    return err
}
defer cleanup()

result, err := a.executor.ExecuteWithWorkspace(directive, workDir)
```

### Phase 2: Agent Status DB + API (1 day)

**New table in collaboration.db:**

```sql
CREATE TABLE agent_status (
    agent_id TEXT PRIMARY KEY,
    status TEXT,              -- 'running', 'idle', 'stopped'
    project_dir TEXT,
    poll_interval_sec INTEGER,
    last_poll INTEGER,        -- Unix timestamp
    next_poll INTEGER,
    daily_cost_usd REAL,
    daily_cap_usd REAL,
    messages_today INTEGER,
    updated_at INTEGER
);

CREATE TABLE execution_log (
    id INTEGER PRIMARY KEY,
    agent_id TEXT,
    message_id TEXT,
    started_at INTEGER,
    completed_at INTEGER,
    status TEXT,              -- 'success', 'failed', 'timeout'
    cost_usd REAL,
    turns INTEGER,
    summary TEXT
);
```

**New API endpoints:**

```go
// GET /api/agents - List agents with status
// GET /api/agents/{id}/executions - Get execution history
```

### Phase 3: Agent Status UI Tab (1 day)

**New file:** `ui/src/components/AgentStatus/AgentStatus.tsx`

- Fetch from `/api/agents`
- Show status, cost, recent executions
- Poll every 30 seconds for updates

### Phase 4: launchd Service (1 day)

**New file:** `internal/agent/platform_darwin.go`

```go
const launchdPlist = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.ailang.agent</string>
    <key>ProgramArguments</key>
    <array>
        <string>{{.BinaryPath}}</string>
        <string>--config</string>
        <string>{{.ConfigPath}}</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>{{.LogDir}}/agent.log</string>
    <key>StandardErrorPath</key>
    <string>{{.LogDir}}/agent.log</string>
</dict>
</plist>`

func InstallService() error {
    // Write plist to ~/Library/LaunchAgents/com.ailang.agent.plist
    // Run: launchctl load ...
}

func StartService() error  { /* launchctl start */ }
func StopService() error   { /* launchctl stop */ }
func StatusService() error { /* launchctl list | grep ailang */ }
```

### Phase 5: Integration Testing (1 day)

- Test config loading
- Test project-dir execution (verify .claude/ loaded)
- Test agent status reporting
- Test launchd install/start/stop
- Test UI shows agent status

## File Summary

**New files:**
```
cmd/ailang-agent/
├── config.go              # Config loading (~100 LOC)
├── service.go             # install/start/stop commands (~150 LOC)

internal/agent/
├── workspace.go           # Workspace mode management (~150 LOC)
├── platform_darwin.go     # launchd integration (~100 LOC)
├── status.go              # Agent status reporting (~150 LOC)

internal/server/
├── agents.go              # New API endpoints (~100 LOC)

ui/src/components/AgentStatus/
├── AgentStatus.tsx        # New UI tab (~200 LOC)
```

**Modified files:**
```
cmd/ailang-agent/main.go   # Add config loading, subcommands
cmd/ailang-agent/agent.go  # Use WorkspaceManager, status reporting
internal/agent/executor.go # Add ExecuteWithWorkspace()
internal/server/server.go  # Register /api/agents routes
ui/src/App.tsx             # Add Agents tab
```

**Total:** ~950 LOC new code

## Success Criteria

- [ ] `ailang-agent --config ~/.ailang/agent.yml` runs with config
- [ ] `in-place` mode: Agent executes directly in source_dir
- [ ] `in-place` mode: `.claude/` settings loaded (skills, commands work)
- [ ] `isolated` mode: Agent creates temp workspace with copied configs
- [ ] `isolated` mode: `.claude/` settings copied and loaded
- [ ] `isolated` mode: Workspace cleaned up after execution
- [ ] `ailang serve` shows Agent Status tab
- [ ] `ailang-agent install` creates launchd service
- [ ] `ailang-agent start/stop/status` work
- [ ] Agent polls every 10 minutes (not 2 seconds)
- [ ] Daily cost tracking enforces cap

## Related Docs

- [Agent Execution Integration (v0.4.5)](../implemented/v0_4_5/agent-execution-integration.md) - Base implementation
- [Agent Execution Enhancements (v0.4.6)](agent-execution-enhancements.md) - Future: progress streaming, budgets

---

**Created:** 2025-11-29
