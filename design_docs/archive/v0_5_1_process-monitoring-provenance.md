# Process Monitoring & Provenance - Unified Hierarchy View

**Status**: Planned
**Target**: v0.5.0
**Priority**: P0 - High
**Dependencies**: Collaboration Hub v2 (partial)

## AI-First Alignment Check

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | 0 | 0 | Infrastructure feature |
| Preserve Semantic Clarity | ++ | +2 | Clear provenance of WHO started each process |
| Increase Determinism | + | +1 | Reproducible process tracing |
| Lower Token Cost | + | +1 | Better debugging reduces iterations |
| **Net Score** | | **+4** | **Decision: Move forward** |

## Problem Statement

The Collaboration Hub monitors two fundamentally different types of processes:

1. **Claude Code agents** - IDE-driven AI assistants (e.g., claude-code sessions)
2. **AILANG runs** - Language interpreter executions (`ailang run`, eval suite, etc.)

**Current Problems:**

1. **No Provenance Tracking**: Can't see WHO initiated a process
   - Was this `ailang run` started by a user in terminal?
   - Was it started by Claude Code as part of a task?
   - Was it started by the eval suite?
   - Was it spawned by another AILANG process?

2. **Flat Process List**: No hierarchy relationship between processes
   - Claude Code session spawns `ailang run` → not shown as child
   - Parent eval process spawns 50 benchmark runs → all appear as siblings
   - Can't trace execution lineage

3. **Two Separate Views**: Claude agents and AILANG runs shown differently
   - Monitor tab shows processes (CPU, RAM)
   - Hierarchy shows agents and threads
   - No unified view connecting them

4. **Resource Usage Not in Hierarchy**: CPU/RAM metrics exist but not integrated
   - `/api/monitor` returns process stats
   - Hierarchy doesn't show resource usage at each level

**User's Exact Words:**
> "we are monitoring two different things - claude code etc. but also ailang runs, and sometimes claude code executes ailang runs as well - we want that clear in the UI showing if ailang run is by user, the eval suite, via claude code etc."

## Goals

**Primary Goal:** Create a unified hierarchy that shows:
- WHO started each process (provenance)
- WHAT processes are children of others (hierarchy)
- HOW much resources each process uses (monitoring)

**Success Metrics:**
- 100% of processes show their initiator type
- Parent-child relationships visible in tree view
- CPU/RAM visible at process level in hierarchy
- <2s latency for process tree updates

## Solution Design

### Provenance Model

Every process records its **initiator**:

```go
type ProcessInitiator struct {
    Type       string `json:"type"`        // "user", "claude-code", "eval-suite", "ailang-process"
    ID         string `json:"id"`          // Session ID, instance ID, or empty for user
    ParentPID  int    `json:"parent_pid"`  // PID of parent process (0 = top-level)
}
```

**Initiator Types:**

| Type | Description | Example |
|------|-------------|---------|
| `user` | Human ran from terminal | `ailang run hello.ail` in shell |
| `claude-code` | Claude Code spawned it | Sprint executor running commands |
| `eval-suite` | Eval harness spawned it | `ailang eval-suite` benchmark run |
| `ailang-process` | Another AILANG process spawned it | Parent script calling `ailang run` |

### Hierarchy Model

Extend the existing hierarchy to include processes:

```
┌─────────────────────────────────────────────────────────────┐
│  Unified Hierarchy View                                      │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  🖥️ claude-code (session_abc123)        CPU: 12%  RAM: 120MB │
│  ├── 📁 Thread: "Fix parser bug"                             │
│  │   └── 🔄 ailang run parser_test.ail  CPU: 5%   RAM: 45MB │
│  └── 📁 Thread: "Run eval suite"                             │
│      └── 🔄 eval-suite fizzbuzz        CPU: 180%  RAM: 320MB │
│          ├── 🔄 benchmark gpt5         CPU: 45%   RAM: 80MB │
│          ├── 🔄 benchmark claude       CPU: 42%   RAM: 78MB │
│          └── 🔄 benchmark gemini       CPU: 48%   RAM: 82MB │
│                                                              │
│  👤 user (terminal)                                          │
│  └── 🔄 ailang run hello.ail           CPU: 2%   RAM: 30MB  │
│                                                              │
│  🧪 eval-suite (headless)               CPU: 5%   RAM: 50MB │
│  └── 🔄 full baseline v0.5.0           CPU: 250%  RAM: 400MB │
│      └── (264 benchmarks running...)                         │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### Database Schema

```sql
-- Extend processes table with provenance
CREATE TABLE IF NOT EXISTS processes (
    id TEXT PRIMARY KEY,
    pid INTEGER NOT NULL,
    instance_id TEXT,

    -- Provenance fields
    initiator_type TEXT NOT NULL,        -- 'user', 'claude-code', 'eval-suite', 'ailang-process'
    initiator_id TEXT,                   -- Session/instance ID of initiator
    parent_process_id TEXT,              -- FK to parent process (NULL = top-level)

    -- Process info
    command TEXT NOT NULL,               -- e.g., "ailang run hello.ail"
    working_dir TEXT,
    started_at INTEGER NOT NULL,
    ended_at INTEGER,
    exit_code INTEGER,
    status TEXT NOT NULL DEFAULT 'running',  -- 'running', 'completed', 'failed'

    -- Resource usage (updated periodically)
    cpu_percent REAL DEFAULT 0,
    memory_mb REAL DEFAULT 0,

    -- Metrics (final values)
    total_tokens INTEGER DEFAULT 0,
    total_cost_cents INTEGER DEFAULT 0,
    duration_ms INTEGER DEFAULT 0,

    FOREIGN KEY (parent_process_id) REFERENCES processes(id)
);

CREATE INDEX idx_processes_parent ON processes(parent_process_id);
CREATE INDEX idx_processes_initiator ON processes(initiator_type, initiator_id);
CREATE INDEX idx_processes_status ON processes(status, started_at);
```

### API Endpoints

**Unified Hierarchy with Processes:**
```
GET /api/hierarchy
```

Response now includes processes nested under their initiators:

```json
{
  "agents": [
    {
      "id": "claude-code",
      "name": "Claude Code Session",
      "session_id": "session_abc123",
      "cpu_percent": 12,
      "memory_mb": 120,
      "threads": [
        {
          "id": "thread_123",
          "title": "Fix parser bug",
          "processes": [
            {
              "id": "proc_456",
              "command": "ailang run parser_test.ail",
              "status": "running",
              "cpu_percent": 5,
              "memory_mb": 45,
              "initiator_type": "claude-code",
              "children": []
            }
          ]
        }
      ]
    }
  ],
  "user_processes": [
    {
      "id": "proc_789",
      "command": "ailang run hello.ail",
      "status": "completed",
      "cpu_percent": 0,
      "memory_mb": 0,
      "initiator_type": "user",
      "children": []
    }
  ],
  "eval_processes": [
    {
      "id": "proc_eval_001",
      "command": "eval-suite --full",
      "status": "running",
      "cpu_percent": 250,
      "memory_mb": 400,
      "initiator_type": "eval-suite",
      "children": [
        {"id": "proc_b1", "command": "benchmark fizzbuzz gpt5", ...},
        {"id": "proc_b2", "command": "benchmark fizzbuzz claude", ...}
      ]
    }
  ]
}
```

**Process Tree Endpoint:**
```
GET /api/processes/tree
GET /api/processes/{id}/children
GET /api/processes/{id}/ancestors
```

### Process Registration

Processes register themselves with the hub:

**1. AILANG Run Auto-Registration:**
```go
// In cmd/ailang/run.go - when starting execution
func registerWithHub(cmd string, parentPID int) {
    initiator := detectInitiator()  // Detect who started us

    req := ProcessRegistration{
        PID:           os.Getpid(),
        Command:       cmd,
        InitiatorType: initiator.Type,
        InitiatorID:   initiator.ID,
        ParentPID:     parentPID,
    }

    // POST to hub if AILANG_HUB_URL is set
    if hubURL := os.Getenv("AILANG_HUB_URL"); hubURL != "" {
        http.Post(hubURL+"/api/processes/register", "application/json", ...)
    }
}

func detectInitiator() ProcessInitiator {
    // Check environment variables set by parent processes
    if sessionID := os.Getenv("CLAUDE_SESSION_ID"); sessionID != "" {
        return ProcessInitiator{Type: "claude-code", ID: sessionID}
    }
    if evalID := os.Getenv("AILANG_EVAL_ID"); evalID != "" {
        return ProcessInitiator{Type: "eval-suite", ID: evalID}
    }
    if parentID := os.Getenv("AILANG_PARENT_PROCESS"); parentID != "" {
        return ProcessInitiator{Type: "ailang-process", ID: parentID}
    }
    return ProcessInitiator{Type: "user", ID: ""}
}
```

**2. Eval Suite Process Marking:**
```go
// In eval harness when spawning benchmark processes
cmd := exec.Command("ailang", "run", benchmarkFile)
cmd.Env = append(os.Environ(),
    "AILANG_EVAL_ID="+evalRunID,
    "AILANG_PARENT_PROCESS="+currentProcessID,
)
```

**3. Claude Code Session Marking:**

Claude Code hooks can set environment variables when spawning processes:
```json
// .claude/hooks/pre-exec.json
{
  "env": {
    "CLAUDE_SESSION_ID": "${CLAUDE_SESSION_ID}",
    "AILANG_HUB_URL": "http://localhost:1957"
  }
}
```

### UI Components

**1. Unified Hierarchy Tree:**
```tsx
// ui/src/components/UnifiedHierarchy/UnifiedHierarchy.tsx

interface ProcessNode {
  id: string;
  command: string;
  status: 'running' | 'completed' | 'failed';
  cpuPercent: number;
  memoryMb: number;
  initiatorType: 'user' | 'claude-code' | 'eval-suite' | 'ailang-process';
  children: ProcessNode[];
}

const ProcessNodeView: React.FC<{node: ProcessNode}> = ({node}) => (
  <div className={`process-node ${node.status}`}>
    <span className="process-icon">{getInitiatorIcon(node.initiatorType)}</span>
    <span className="process-command">{node.command}</span>
    <span className="process-resources">
      CPU: {node.cpuPercent}% | RAM: {node.memoryMb}MB
    </span>
    {node.children.length > 0 && (
      <div className="process-children">
        {node.children.map(child => (
          <ProcessNodeView key={child.id} node={child} />
        ))}
      </div>
    )}
  </div>
);

function getInitiatorIcon(type: string): string {
  switch (type) {
    case 'user': return '👤';
    case 'claude-code': return '🖥️';
    case 'eval-suite': return '🧪';
    case 'ailang-process': return '🔄';
    default: return '❓';
  }
}
```

**2. Resource Badges:**
```tsx
// Resource usage badges with color coding
<ResourceBadge
  cpu={process.cpuPercent}
  memory={process.memoryMb}
  warnings={{cpu: 80, memory: 500}}  // Thresholds
/>
```

**3. Provenance Badge:**
```tsx
// Show who started this process
<ProvenanceBadge
  type={process.initiatorType}
  id={process.initiatorId}
  onClick={() => navigateToInitiator(process)}
/>
```

## Implementation Plan

### M1: Process Registry & Provenance (~10 hours)

**Goal:** Track all processes with their initiators

**Tasks:**
- [ ] Add `processes` table with provenance fields
- [ ] Create `/api/processes/register` endpoint
- [ ] Add auto-registration to `ailang run` command
- [ ] Implement initiator detection logic
- [ ] Add AILANG_EVAL_ID and CLAUDE_SESSION_ID env var support
- [ ] Update `/api/monitor` to include provenance
- [ ] Unit tests for process registry
- [ ] Integration test: spawn process → verify registration

**Files:**
- `internal/messaging/processes.go` (new, ~250 LOC)
- `internal/server/process_handlers.go` (new, ~150 LOC)
- `cmd/ailang/run.go` (modify, +50 LOC)
- `internal/eval_harness/runner.go` (modify, +30 LOC)

### M2: Process Hierarchy & Parent-Child (~8 hours)

**Goal:** Track parent-child relationships between processes

**Tasks:**
- [ ] Add `parent_process_id` to process registration
- [ ] Implement `/api/processes/tree` endpoint
- [ ] Add ancestor/descendant queries
- [ ] Update eval harness to set parent env vars
- [ ] Handle orphaned processes (parent exits)
- [ ] Add hierarchy depth limits (max 10 levels)
- [ ] Tests for process tree queries

**Files:**
- `internal/messaging/processes.go` (extend, +100 LOC)
- `internal/server/process_handlers.go` (extend, +100 LOC)

### M3: Resource Monitoring Integration (~6 hours)

**Goal:** Show CPU/RAM in hierarchy view

**Tasks:**
- [ ] Add background worker to poll process resources
- [ ] Store resource snapshots in process records
- [ ] Include resources in hierarchy API response
- [ ] Add resource history for trends (last 5 minutes)
- [ ] Handle process termination gracefully
- [ ] Tests for resource polling

**Files:**
- `internal/server/resource_monitor.go` (new, ~200 LOC)
- `internal/messaging/processes.go` (extend, +50 LOC)

### M4: Unified Hierarchy UI (~10 hours)

**Goal:** Display processes nested in hierarchy with resources

**Tasks:**
- [ ] Create UnifiedHierarchy component
- [ ] Add ProcessNode with resource badges
- [ ] Add ProvenanceBadge component
- [ ] Integrate into Monitor tab
- [ ] Add expand/collapse for process trees
- [ ] Real-time updates via WebSocket
- [ ] Color-coded warnings for high resource usage
- [ ] "Kill process" action in UI
- [ ] E2E test: spawn nested processes → verify UI

**Files:**
- `ui/src/components/UnifiedHierarchy/` (new, ~400 LOC)
- `ui/src/components/ProcessNode/` (new, ~150 LOC)
- `ui/src/components/ResourceBadge/` (new, ~80 LOC)
- `ui/src/components/ProvenanceBadge/` (new, ~60 LOC)

### M5: Claude Code Integration (~6 hours)

**Goal:** Link Claude Code sessions to spawned processes

**Tasks:**
- [ ] Document hook configuration for Claude Code
- [ ] Create example hooks for session tracking
- [ ] Test with actual Claude Code sessions
- [ ] Show Claude Code sessions as top-level nodes
- [ ] Link threads to sessions
- [ ] Documentation for setup

**Files:**
- `.claude/hooks/process-tracking.md` (new)
- `docs/guides/claude-code-integration.md` (new)

## Success Criteria

- [ ] Every process shows initiator type (user/claude-code/eval-suite/ailang-process)
- [ ] Parent-child relationships visible in tree view
- [ ] CPU/RAM visible at process level
- [ ] Unified hierarchy shows agents, threads, AND processes
- [ ] Resource warnings (>80% CPU, >500MB RAM) highlighted
- [ ] Process kill action works from UI
- [ ] Claude Code sessions appear as top-level nodes
- [ ] All tests passing
- [ ] Documentation complete

## Testing Strategy

**Unit Tests:**
- Provenance detection logic
- Process tree queries
- Resource polling

**Integration Tests:**
- Process registration flow
- Hierarchy API with nested processes
- WebSocket updates for process changes

**E2E Tests:**
- Spawn eval suite → verify process tree in UI
- Claude Code spawns ailang run → verify parent-child

**Manual Tests:**
- Start eval baseline → watch processes appear with correct initiators
- Kill runaway process from UI → verify termination

## Non-Goals

**Not in this sprint:**
- Prometheus metrics export
- Historical process analytics
- Process resource limits/quotas
- Multi-user process isolation

## References

- [Collaboration Hub v2](collaboration-hub-v2.md) - Base infrastructure
- [M-AGENT-MONITOR](m-agent-monitor.md) - Resource monitoring design
- [Process Telemetry](internal/eval_harness/telemetry_reporter.go) - Existing telemetry

---

**Document created**: 2025-11-30
**Last updated**: 2025-11-30
