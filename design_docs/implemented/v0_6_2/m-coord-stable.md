# M-COORD-STABLE: Coordinator Daemon Stability & Agent Collaboration Foundation

| Field | Value |
|-------|-------|
| Status | Planned |
| Target | v0.6.2 |
| Priority | P0 (Critical) |
| Estimated | 3 days |
| Dependencies | M-COORD-HUMAN-LOOP (supersedes) |
| Created | 2025-12-31 |
| Related | [m-coord-human-loop.md](m-coord-human-loop.md) |

## Problem Statement

The coordinator daemon has critical issues preventing it from being a stable foundation for agent collaboration:

### Critical Bugs Found (Audit Dec 31, 2025)

| Issue | Location | Impact |
|-------|----------|--------|
| **Worktrees deleted immediately** | `daemon.go:509-514` | All agent work is lost before approval |
| **ApprovalCheckpoint not wired** | `daemon.go` | Approval mechanism exists but never used |
| **HTTP 404 errors** | `http_broadcaster.go` | Dashboard never receives streaming events |
| **Metrics race condition** | `event_handler.go:204` | Events stored before metrics available (all show 0) |
| **Silent event storage errors** | `event_handler.go:221` | Events silently lost on DB errors |
| **Only hardcoded coordinator inbox** | `daemon.go:218` | Can't route to other agents |

### Missing Agent Collaboration Features

1. **No agent configuration** - No way to register agents with workspaces, inboxes, capabilities
2. **No message routing** - Messages to "sprint-planner" don't reach sprint-planner agent
3. **No agent-to-agent messaging** - Agents can't delegate work to each other
4. **No GitHub sync trigger** - Issues imported but don't auto-trigger tasks
5. **No worktree preservation** - Work deleted before human can review/approve
6. **No merge workflow** - Even if approved, no path to merge changes to main

### Evidence of Lost Work

```
Task task-593740ab:
- Completed heatmap implementation (11.6KB React, Go backend)
- Worktree created at ~/.ailang/state/worktrees/coordinator/task-593740ab/
- Worktree DELETED at daemon.go:511 immediately after completion
- Work lost forever
- Cost: $2+ in AI execution with zero output
```

## Goals

**Primary Goal:** Make the coordinator daemon a rock-solid foundation for autonomous agent collaboration.

**Success Metrics:**
1. Zero work lost - all agent output preserved until explicit approval/rejection
2. Real-time streaming visible in dashboard
3. Agents can be configured with workspaces and capabilities
4. Messages routed to correct agents based on inbox
5. Agent-to-agent messaging triggers new tasks
6. GitHub issues auto-trigger tasks via periodic sync
7. Human approval required before merge OR delete

## Solution Design

### Architecture Overview

```
┌──────────────────────────────────────────────────────────────────────────┐
│                        ~/.ailang/config.yaml                              │
│  agents:                                                                  │
│    - id: coordinator                                                      │
│      workspace: /Users/mark/dev/sunholo/ailang                           │
│      capabilities: [claude-code, gemini-cli]                             │
│    - id: sprint-planner                                                   │
│      workspace: /Users/mark/dev/sunholo/ailang                           │
│      capabilities: [claude-code]                                          │
│    - id: stapledon-agent                                                  │
│      workspace: /Users/mark/dev/sunholo/stapledons_voyage                │
│      capabilities: [claude-code]                                          │
└──────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌──────────────────────────────────────────────────────────────────────────┐
│                      Multi-Agent Coordinator Daemon                       │
├──────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  ┌─────────────┐    ┌──────────────┐    ┌─────────────────────────────┐ │
│  │ Message     │    │ Agent Router │    │ Task Executor               │ │
│  │ Watcher     │───▶│              │───▶│ (per-agent worktree)        │ │
│  │             │    │ inbox→agent  │    │                             │ │
│  └─────────────┘    └──────────────┘    └─────────────────────────────┘ │
│         │                                            │                   │
│         │                                            ▼                   │
│         │                               ┌─────────────────────────────┐ │
│         │                               │ Approval Checkpoint         │ │
│         │                               │ (blocks until approved)     │ │
│         │                               └─────────────────────────────┘ │
│         │                                            │                   │
│         ▼                                            ▼                   │
│  ┌─────────────┐                        ┌─────────────────────────────┐ │
│  │ GitHub Sync │                        │ Worktree Manager            │ │
│  │ (periodic)  │                        │ (PRESERVED until approval)  │ │
│  └─────────────┘                        └─────────────────────────────┘ │
│                                                      │                   │
└──────────────────────────────────────────────────────┼───────────────────┘
                                                       │
                          HTTP POST /api/coordinator/events
                                                       ▼
┌──────────────────────────────────────────────────────────────────────────┐
│                      Collaboration Hub Server                             │
├──────────────────────────────────────────────────────────────────────────┤
│  /api/coordinator/events  → WebSocket broadcast → Dashboard              │
│  /api/coordinator/pending → Pending approval list                        │
│  /api/coordinator/approve/{id} → Approve & merge                         │
│  /api/coordinator/reject/{id} → Reject & cleanup                         │
│  /api/coordinator/tasks/{id}/diff → View worktree diff                   │
└──────────────────────────────────────────────────────────────────────────┘
```

### Phase 1: Fix Critical Bugs (4 hours)

#### 1.1 Stop Premature Worktree Deletion

**Current (WRONG):**
```go
// daemon.go:509-514
// Cleanup worktree
if worktree != nil && d.worktreeMgr != nil {
    if rmErr := d.worktreeMgr.RemoveWorktree(task.ID); rmErr != nil {
        d.logger.Printf("Warning: Failed to remove worktree: %v", rmErr)
    }
}
```

**Fixed:**
```go
// daemon.go - REMOVE the cleanup block entirely
// Worktrees are now preserved until approval

// After task completion, mark as pending_approval
if result.Success {
    // Don't mark completed - mark pending approval
    if err := d.taskStore.MarkTaskPendingApproval(d.ctx, task.ID, worktree.Path); err != nil {
        d.logger.Printf("Warning: Failed to mark task pending approval: %v", err)
    }

    // Request human approval
    if d.approvalCheckpoint != nil {
        go d.requestApproval(task, worktree)
    }
}
```

**Files:**
- `internal/coordinator/daemon.go` - Remove worktree cleanup, add pending_approval
- `internal/coordinator/store.go` - Add `MarkTaskPendingApproval`, `WorktreePath` field
- `internal/coordinator/store_sqlite.go` - Implement new methods

#### 1.2 Wire ApprovalCheckpoint

**Current:** `ApprovalCheckpoint` exists but daemon never instantiates it.

**Fixed:**
```go
// daemon.go - Add to Daemon struct
type Daemon struct {
    // ... existing fields ...
    approvalCheckpoint *StoreBackedApprovalCheckpoint
}

// initTaskProcessing - Add initialization
func (d *Daemon) initTaskProcessing() error {
    // ... existing code ...

    // Initialize approval checkpoint with store backing
    d.approvalCheckpoint = NewStoreBackedApprovalCheckpoint(
        d.taskStore.(ApprovalStore),
        1*time.Hour, // Default timeout
    )
}
```

#### 1.3 Fix HTTP 404 Errors

**Problem:** Events POST to `/api/coordinator/events` but get 404.

**Investigation needed:**
```go
// handlers_coordinator.go:219 - Handler exists
func (s *Server) handleCoordinatorTaskEvents(w http.ResponseWriter, r *http.Request) {

// Check if route is registered in server.go
```

**Files:**
- `internal/server/server.go` - Verify route registration
- `internal/coordinator/http_broadcaster.go` - Add retry logic, better logging

#### 1.4 Fix Metrics Race Condition

**Problem:** Events stored in goroutine before metrics available.

**Fixed:**
```go
// event_handler.go - Store metrics with final status event only
func (h *CoordinatorEventHandler) EmitStatus(status string) {
    // For status events, include current metrics
    h.mu.Lock()
    record := &TaskEventRecord{
        // ... existing fields ...
        TokensIn:    h.tokensIn,
        TokensOut:   h.tokensOut,
        Cost:        h.cost,
        DurationSec: int(time.Since(h.startTime).Seconds()),
    }
    h.mu.Unlock()

    // Store synchronously for status events (important)
    if h.store != nil {
        if err := h.store(record); err != nil {
            h.logger.Printf("ERROR: Failed to store status event: %v", err)
        }
    }
}
```

### Phase 2: Agent Configuration System (4 hours)

#### 2.1 Config Schema

Add to `~/.ailang/config.yaml`:

```yaml
# Agent configuration
agents:
  # Coordinator - the orchestration daemon
  - id: coordinator
    label: "Coordinator Daemon"
    inbox: coordinator           # Messages to this inbox trigger this agent
    workspace: /Users/mark/dev/sunholo/ailang
    capabilities:
      - claude-code
      - gemini-cli
    auto_merge: false            # Require approval before merge
    max_concurrent: 3

  # Sprint Planner - creates sprint plans from design docs
  - id: sprint-planner
    label: "Sprint Planner"
    inbox: sprint-planner
    workspace: /Users/mark/dev/sunholo/ailang
    capabilities:
      - claude-code
    trigger_on_complete:         # After this agent completes...
      - sprint-executor          # ...trigger sprint-executor with output
    auto_approve_handoffs: false # Require human approval before triggering downstream agent
    auto_merge: false

  # Sprint Executor - implements sprint plans
  - id: sprint-executor
    label: "Sprint Executor"
    inbox: sprint-executor
    workspace: /Users/mark/dev/sunholo/ailang
    capabilities:
      - claude-code
    auto_merge: false

  # External project agent
  - id: stapledon-agent
    label: "Stapledon's Voyage"
    inbox: stapledons_voyage
    workspace: /Users/mark/dev/sunholo/stapledons_voyage
    capabilities:
      - claude-code

# GitHub sync configuration
github:
  sync_interval: 5m              # Check for new issues every 5 minutes
  auto_import: true              # Auto-import issues as messages
  watch_labels:
    - from:stapledon
    - ailang-task
  default_inbox: coordinator     # Where to route imported issues
```

#### 2.2 Agent Registry

```go
// internal/coordinator/agent_registry.go

type AgentConfig struct {
    ID                  string   `yaml:"id"`
    Label               string   `yaml:"label"`
    Inbox               string   `yaml:"inbox"`
    Workspace           string   `yaml:"workspace"`
    Capabilities        []string `yaml:"capabilities"`
    TriggerOnComplete   []string `yaml:"trigger_on_complete"`
    AutoApproveHandoffs bool     `yaml:"auto_approve_handoffs"` // If false (default), require human approval for agent-to-agent handoffs
    AutoMerge           bool     `yaml:"auto_merge"`            // If false (default), require human approval before merging to main
    MaxConcurrent       int      `yaml:"max_concurrent"`
}

type AgentRegistry struct {
    agents map[string]*AgentConfig // keyed by inbox
}

func (r *AgentRegistry) GetAgentForInbox(inbox string) *AgentConfig {
    return r.agents[inbox]
}

func (r *AgentRegistry) GetAgentByID(id string) *AgentConfig {
    for _, agent := range r.agents {
        if agent.ID == id {
            return agent
        }
    }
    return nil
}
```

**Files:**
- `internal/coordinator/agent_registry.go` - New file (~150 LOC)
- `internal/coordinator/config.go` - Load agent config from YAML

### Phase 3: Multi-Inbox Message Routing (3 hours)

#### 3.1 Watch Multiple Inboxes

**Current:** Hardcoded to watch only "coordinator" inbox.

**Fixed:**
```go
// daemon.go - Watch all configured agent inboxes
func (d *Daemon) initTaskProcessing() error {
    // Load agent registry from config
    d.agentRegistry = LoadAgentRegistry()

    // Create message watchers for each agent's inbox
    d.inboxWatchers = make(map[string]*InboxWatcher)
    for inbox, agent := range d.agentRegistry.agents {
        watcher, err := NewInboxWatcher(inbox, agent)
        if err != nil {
            d.logger.Printf("Warning: Failed to create watcher for %s: %v", inbox, err)
            continue
        }
        d.inboxWatchers[inbox] = watcher
    }
}

// Poll all inboxes
func (d *Daemon) pollAndProcessTasks() error {
    for inbox, watcher := range d.inboxWatchers {
        messages, err := watcher.ListUnread()
        if err != nil {
            d.logger.Printf("Error polling inbox %s: %v", inbox, err)
            continue
        }

        for _, msg := range messages {
            agent := d.agentRegistry.GetAgentForInbox(inbox)
            if err := d.createTaskForAgent(msg, agent); err != nil {
                d.logger.Printf("Failed to create task: %v", err)
            }
        }
    }
    return nil
}
```

#### 3.2 Agent-Specific Worktrees

```go
// Use agent's configured workspace for worktree
func (d *Daemon) createWorktreeForAgent(taskID string, agent *AgentConfig) (*Worktree, error) {
    // Get or create worktree manager for agent's workspace
    wtMgr, ok := d.worktreeManagers[agent.ID]
    if !ok {
        var err error
        wtMgr, err = NewWorktreeManager(
            agent.Workspace,
            filepath.Join(d.config.StateDir, "worktrees", agent.ID),
            agent.MaxConcurrent,
        )
        if err != nil {
            return nil, err
        }
        d.worktreeManagers[agent.ID] = wtMgr
    }

    return wtMgr.CreateWorktree(taskID)
}
```

### Phase 4: Agent-to-Agent Messaging (3 hours)

#### 4.1 Trigger On Complete (With Approval Gate)

When an agent completes, check if it should trigger another agent. **By default, human approval is required for handoffs** (configurable via `auto_approve_handoffs`):

```go
// daemon.go
func (d *Daemon) onTaskCompleted(task *TaskRecord, result *ExecuteResult, agent *AgentConfig) {
    // ... existing completion logic ...

    // Check for downstream triggers
    for _, targetAgentID := range agent.TriggerOnComplete {
        targetAgent := d.agentRegistry.GetAgentByID(targetAgentID)
        if targetAgent == nil {
            d.logger.Printf("Warning: Unknown trigger target: %s", targetAgentID)
            continue
        }

        // Check if human approval is required for handoff
        if !agent.AutoApproveHandoffs {
            // Request approval before triggering downstream agent
            request := &ApprovalRequest{
                TaskID:      task.ID,
                Type:        ApprovalTypeHandoff,
                Title:       fmt.Sprintf("Handoff: %s → %s", agent.Label, targetAgent.Label),
                Description: fmt.Sprintf("Agent %s completed. Approve to trigger %s with output?",
                    agent.Label, targetAgent.Label),
                Timeout:     1 * time.Hour,
                AutoReject:  false,
            }

            status, _ := d.approvalCheckpoint.RequestApproval(d.ctx, request)
            if status != ApprovalStatusApproved {
                d.logger.Printf("Handoff to %s not approved", targetAgentID)
                continue
            }
        }

        // Send message to target agent's inbox
        msg := &messaging.Message{
            Inbox:   targetAgent.Inbox,
            Title:   fmt.Sprintf("Follow-up from %s: %s", agent.Label, task.Title),
            Content: result.Output, // Pass output as input to next agent
            From:    agent.ID,
            Kind:    "directive",
            Type:    "handoff",
            // Include session ID for conversation continuity
            Metadata: map[string]string{
                "parent_task_id":  task.ID,
                "parent_session":  result.SessionID, // Claude Code/Gemini CLI session
            },
        }

        if err := d.msgStore.CreateInboxMessage(msg); err != nil {
            d.logger.Printf("Failed to trigger %s: %v", targetAgentID, err)
        } else {
            d.logger.Printf("Triggered %s with handoff from %s", targetAgentID, agent.ID)
        }
    }
}
```

#### 4.2 Leverage Claude Code / Gemini CLI Session IDs

**Key principle: Don't reimplement features that Claude Code and Gemini CLI already provide.**

Both tools support session/conversation IDs that maintain chat history:

```bash
# Claude Code - use --resume to continue a session
claude -p "continue the task" --resume SESSION_ID --output-format json

# Gemini CLI - use conversation ID
gemini chat --conversation-id CONV_ID "continue the task"
```

**Integration:**

```go
// executor/claude/claude.go
type ClaudeResult struct {
    SessionID string `json:"session_id"` // From Claude Code output
    // ... other fields
}

// executor/gemini/gemini.go
type GeminiResult struct {
    ConversationID string `json:"conversation_id"` // From Gemini CLI output
    // ... other fields
}

// When resuming work on a task, pass the session ID
func (p *ClaudeCodeProvider) ExecuteWithSession(ctx context.Context, task *Task, sessionID string) (*Result, error) {
    args := []string{"-p", task.Content, "--output-format", "json"}
    if sessionID != "" {
        args = append(args, "--resume", sessionID)
    }
    // ... execute
}
```

**Benefits:**
- Full conversation history maintained by Claude Code/Gemini CLI
- No need to store/replay messages ourselves
- Context window managed by the tools
- Can resume interrupted tasks seamlessly

**Store session IDs:**

```go
// store.go - TaskRecord
type TaskRecord struct {
    // ... existing fields ...
    SessionID string `json:"session_id,omitempty"` // Claude/Gemini session for resumption
}
```

#### 4.3 Explicit Agent Messaging

Agents can send messages to other agents via AILANG code or executor output:

```go
// Parse executor output for @agent directives
func (d *Daemon) parseAgentDirectives(output string) []AgentDirective {
    // Look for patterns like:
    // @sprint-executor: Implement the following plan...
    // @coordinator: Need human review for this change

    var directives []AgentDirective
    re := regexp.MustCompile(`@([a-zA-Z0-9-]+):\s*(.+?)(?:\n\n|$)`)
    matches := re.FindAllStringSubmatch(output, -1)

    for _, match := range matches {
        directives = append(directives, AgentDirective{
            TargetAgent: match[1],
            Content:     match[2],
        })
    }
    return directives
}
```

### Phase 5: GitHub Sync Auto-Trigger (2 hours)

#### 5.1 Periodic GitHub Import

```go
// daemon.go - Add GitHub sync to main loop
func (d *Daemon) Run() error {
    pollTicker := time.NewTicker(d.config.PollInterval)
    githubTicker := time.NewTicker(d.config.GitHubSyncInterval)
    defer pollTicker.Stop()
    defer githubTicker.Stop()

    for {
        select {
        case <-d.ctx.Done():
            return nil

        case <-pollTicker.C:
            d.pollAndProcessTasks()
            d.executeTaskQueue()

        case <-githubTicker.C:
            if d.config.GitHubAutoImport {
                if err := d.syncGitHubIssues(); err != nil {
                    d.logger.Printf("GitHub sync error: %v", err)
                }
            }
        }
    }
}

func (d *Daemon) syncGitHubIssues() error {
    // Run ailang messages import-github
    cmd := exec.Command("ailang", "messages", "import-github",
        "--labels", strings.Join(d.config.GitHubWatchLabels, ","),
    )
    output, err := cmd.CombinedOutput()
    if err != nil {
        return fmt.Errorf("import failed: %w\nOutput: %s", err, output)
    }

    d.logger.Printf("GitHub sync complete: %s", strings.TrimSpace(string(output)))
    return nil
}
```

### Phase 6: Approval Workflow & Merge (4 hours)

#### 6.1 Task Status: pending_approval

Add new task status:

```go
// store.go
const (
    TaskStatusPending         TaskStatus = "pending"
    TaskStatusRunning         TaskStatus = "running"
    TaskStatusPendingApproval TaskStatus = "pending_approval" // NEW
    TaskStatusCompleted       TaskStatus = "completed"
    TaskStatusFailed          TaskStatus = "failed"
    TaskStatusRejected        TaskStatus = "rejected"         // NEW
)

// TaskRecord - add worktree path
type TaskRecord struct {
    // ... existing fields ...
    WorktreePath string `json:"worktree_path,omitempty"`
}
```

#### 6.2 Request Approval After Task Completion

```go
// daemon.go
func (d *Daemon) requestApproval(task *TaskRecord, worktree *Worktree) {
    // Get change summary
    changes, err := d.worktreeMgr.GetChangeSummary(task.ID)
    if err != nil {
        d.logger.Printf("Failed to get change summary: %v", err)
        return
    }

    request := &ApprovalRequest{
        TaskID:       task.ID,
        ThreadID:     task.ThreadID,
        Type:         ApprovalTypeMerge,
        Title:        fmt.Sprintf("Merge changes from: %s", task.Title),
        Description:  fmt.Sprintf("Agent completed work. Review and approve to merge.\n\nFiles changed: %d\nCommits: %s",
            len(changes.FilesChanged), changes.CommitsAhead),
        DiffSummary:  changes.DiffSummary,
        FilesChanged: changes.FilesChanged,
        Timeout:      24 * time.Hour,
        AutoReject:   false, // Don't auto-reject - human must decide
    }

    // This blocks until human approves/rejects (or timeout)
    status, err := d.approvalCheckpoint.RequestApproval(d.ctx, request)
    if err != nil {
        d.logger.Printf("Approval request failed: %v", err)
        return
    }

    switch status {
    case ApprovalStatusApproved:
        d.handleApproval(task, worktree)
    case ApprovalStatusRejected:
        d.handleRejection(task, worktree)
    case ApprovalStatusTimeout:
        d.logger.Printf("Approval timeout for task %s - keeping worktree for later", task.ID)
    }
}
```

#### 6.3 Merge Handler

```go
// daemon.go
func (d *Daemon) handleApproval(task *TaskRecord, worktree *Worktree) {
    d.logger.Printf("Task %s approved - merging to main", task.ID)

    // Commit any uncommitted changes in worktree
    if hasChanges, _ := d.worktreeMgr.HasChanges(task.ID); hasChanges {
        cmd := exec.Command("git", "add", "-A")
        cmd.Dir = worktree.Path
        _ = cmd.Run()

        cmd = exec.Command("git", "commit", "-m",
            fmt.Sprintf("Auto-commit from task %s: %s", task.ID, task.Title))
        cmd.Dir = worktree.Path
        _ = cmd.Run()
    }

    // Merge worktree branch to main/dev
    agent := d.agentRegistry.GetAgentForInbox(task.Workspace)
    mainBranch := "dev" // or read from git config

    cmd := exec.Command("git", "merge", worktree.Branch, "--no-ff",
        "-m", fmt.Sprintf("Merge %s: %s", task.ID, task.Title))
    cmd.Dir = agent.Workspace
    output, err := cmd.CombinedOutput()
    if err != nil {
        d.logger.Printf("Merge failed: %v\nOutput: %s", err, output)
        // Keep worktree for manual resolution
        d.taskStore.MarkTaskFailed(d.ctx, task.ID,
            fmt.Errorf("merge conflict - manual resolution required"))
        return
    }

    // Success - clean up worktree
    if err := d.worktreeMgr.RemoveWorktree(task.ID); err != nil {
        d.logger.Printf("Warning: Failed to clean up worktree: %v", err)
    }

    d.taskStore.MarkTaskCompleted(d.ctx, task.ID, nil)
    d.logger.Printf("Task %s merged successfully", task.ID)

    // Notify user
    d.postTaskStatus(task, "merged", "Changes merged to "+mainBranch)
}
```

#### 6.4 Rejection Handler

```go
func (d *Daemon) handleRejection(task *TaskRecord, worktree *Worktree) {
    d.logger.Printf("Task %s rejected", task.ID)

    // Mark as rejected but KEEP worktree for potential retry
    d.taskStore.MarkTaskRejected(d.ctx, task.ID)

    // Notify user
    d.postTaskStatus(task, "rejected",
        "Changes rejected. Worktree preserved at: "+worktree.Path)
}
```

### Phase 7: Dashboard Integration (2 hours)

#### 7.1 Pending Approvals Endpoint

Already exists at `/api/coordinator/pending` - verify it returns worktree info:

```go
// handlers_coordinator.go - Enhance pending approvals
func (s *Server) handleCoordinatorPendingApprovals(w http.ResponseWriter, r *http.Request) {
    // ... existing code ...

    // Include worktree path and diff summary
    for _, req := range pending {
        result = append(result, map[string]interface{}{
            "id":            req.ID,
            "task_id":       req.TaskID,
            "title":         req.Title,
            "description":   req.Description,
            "worktree_path": req.WorktreePath,
            "diff_summary":  req.DiffSummary,
            "files_changed": req.FilesChanged,
            "created_at":    req.CreatedAt,
        })
    }
}
```

#### 7.2 Diff Viewer Endpoint

```go
// handlers_coordinator.go - New endpoint
// GET /api/coordinator/tasks/{id}/diff
func (s *Server) handleCoordinatorTaskDiff(w http.ResponseWriter, r *http.Request) {
    taskID := extractTaskID(r.URL.Path)

    task, err := s.taskStore.GetTask(r.Context(), taskID)
    if err != nil {
        http.Error(w, "Task not found", http.StatusNotFound)
        return
    }

    if task.WorktreePath == "" {
        http.Error(w, "No worktree for this task", http.StatusBadRequest)
        return
    }

    // Get full diff
    cmd := exec.Command("git", "diff", "HEAD")
    cmd.Dir = task.WorktreePath
    diffOutput, err := cmd.Output()
    if err != nil {
        http.Error(w, "Failed to get diff", http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "text/plain")
    w.Write(diffOutput)
}
```

## Implementation Plan

### Day 1: Critical Bug Fixes

| Task | Hours | LOC |
|------|-------|-----|
| 1.1 Stop premature worktree deletion | 1 | 30 |
| 1.2 Wire ApprovalCheckpoint | 1 | 50 |
| 1.3 Fix HTTP 404 errors | 1 | 30 |
| 1.4 Fix metrics race condition | 1 | 40 |
| **Day 1 Total** | **4** | **150** |

### Day 2: Agent Configuration & Routing

| Task | Hours | LOC |
|------|-------|-----|
| 2.1 Config schema | 1 | 50 |
| 2.2 Agent registry | 2 | 150 |
| 3.1 Multi-inbox watching | 2 | 100 |
| 3.2 Agent-specific worktrees | 1 | 50 |
| **Day 2 Total** | **6** | **350** |

### Day 3: Agent Messaging & Approval

| Task | Hours | LOC |
|------|-------|-----|
| 4.1 Trigger on complete | 2 | 100 |
| 4.2 Explicit agent messaging | 1 | 50 |
| 5.1 GitHub sync auto-trigger | 2 | 80 |
| 6.1-6.4 Approval workflow | 3 | 200 |
| 7.1-7.2 Dashboard integration | 2 | 100 |
| **Day 3 Total** | **10** | **530** |

**Total: ~20 hours, ~1030 LOC**

## Success Criteria

- [ ] **Zero work lost**: Agent work preserved in worktrees until explicit decision
- [ ] **Streaming works**: Dashboard shows real-time agent activity
- [ ] **Agent routing**: Messages to `sprint-planner` inbox reach sprint-planner agent
- [ ] **Agent-to-agent**: Agent A completion triggers Agent B
- [ ] **GitHub sync**: New GitHub issues auto-create tasks
- [ ] **Approval gate**: Human must approve/reject before merge
- [ ] **Merge works**: Approved changes merge to main branch
- [ ] **Rejection preserves**: Rejected work stays in worktree for retry

## Testing Plan

1. **Unit tests**: Agent registry, config loading, worktree manager
2. **Integration tests**:
   - Send message to `sprint-planner` inbox → sprint-planner agent picks it up
   - Agent completes → triggers downstream agent
   - Task completes → appears in pending approval
   - Approve → changes merged
   - Reject → worktree preserved
3. **End-to-end test**:
   - Create GitHub issue with `ailang-task` label
   - Issue auto-imported as message
   - Message triggers coordinator
   - Coordinator completes work
   - Dashboard shows pending approval
   - Approve via dashboard
   - Changes appear in git log

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Merge conflicts | Medium | Medium | Detect early, show to user, keep worktree |
| Worktree accumulation | Medium | Low | Add periodic cleanup for old rejected tasks |
| Agent infinite loop | Low | High | Add max-depth for agent-to-agent chains |
| Config errors | Medium | Low | Validate config at startup with clear errors |

## Relationship to M-COORD-HUMAN-LOOP

This design doc **supersedes** M-COORD-HUMAN-LOOP by:
1. Including all of its streaming/approval features
2. Adding agent configuration system
3. Adding multi-agent routing
4. Adding agent-to-agent messaging
5. Adding GitHub sync auto-trigger
6. Fixing the critical bugs not identified in the original

The M-COORD-HUMAN-LOOP sprint plan can be incorporated as phases 1 and 6-7 of this plan.

## Notes

This is the foundation for autonomous agent collaboration. Once stable:
- Dashboard becomes a monitoring/approval interface
- Agents can be added via config without code changes
- External projects can send tasks via messages
- Human stays in the loop for merge decisions
- No work is ever lost

### Additional Approval Types

Add to `approval_checkpoint.go`:

```go
const (
    ApprovalTypeMerge     ApprovalType = "merge"      // Request to merge worktree changes
    ApprovalTypeDestroy   ApprovalType = "destroy"    // Request to destroy worktree with changes
    ApprovalTypeExecute   ApprovalType = "execute"    // Request to execute a destructive operation
    ApprovalTypeCost      ApprovalType = "cost"       // Cost threshold exceeded
    ApprovalTypeHandoff   ApprovalType = "handoff"    // Agent-to-agent handoff (NEW)
)
```

### Future Consideration: Semantic Search

The current `ailang docs search` uses SimHash which isn't always good at finding conceptually related docs. Future work should add semantic search using embeddings (already partially implemented with `--neural` flag but slow).

This would help agents find relevant design docs and context before executing tasks.
