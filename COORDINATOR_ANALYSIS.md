# AILANG Coordinator Codebase Exploration Summary

## Overview
The coordinator is a daemon that orchestrates autonomous task execution using AI agents (Claude Code CLI and Gemini CLI). It provides GitHub integration, approval workflows, and multi-agent task chains.

**Codebase Size**: 
- 12,523 total lines of Go code
- 42 Go files in coordinator package
- 136 test functions across 14 test files
- 4,729 lines of test code (38% test-to-code ratio)

---

## Key Coordinator Packages and Responsibilities

### 1. **Core Task Processing** (`internal/coordinator/`)

#### Package Roles:
- **daemon.go** (config + lifecycle) - Daemon configuration, initialization, status tracking
- **daemon_tasks.go** - Task processing initialization, inbox adapters, analyzer setup
- **daemon_github.go** - GitHub sync (periodic issue import via `ailang messages import-github`)
- **daemon_lifecycle.go** - Daemon start/stop, context management
- **daemon_approval.go** - Approval request handling and dashboard updates

#### Key Data Models:
```
Message (messaging system)
├─ ID, From, Title, Content, Type, Kind
├─ GithubIssue (linked issue number)
└─ CreatedAt

Task (from Message)
├─ ID, Title, Content, Kind ("directive" or "question")
├─ Priority (inferred)
└─ MessageID (source)

AnalyzedTask (classification)
├─ Task (embedded)
├─ Type (bug-fix, feature, docs, research, refactor, test, unknown)
├─ Keywords (extracted)
├─ Fingerprint (SimHash for dedup)
└─ DuplicateOf (if detected)

TaskRecord (stored)
├─ Status (pending, queued, running, pending_approval, completed, failed, rejected, cancelled, duplicate)
├─ Stage (design, sprint, implementation, merge) - for GitHub pipeline
├─ GithubIssue (linked issue number)
├─ Provider (executor used)
├─ WorktreeID, SessionID (for resumption)
├─ Cost, TokensUsed, Duration
└─ Timestamps (created, started, completed)
```

#### Message Flow:
1. **GitHub Issues → Messages**: `runGitHubSync()` runs `ailang messages import-github` periodically
   - Imports GitHub issues as messages into `inbox_messages` table
   - Filters by WatchLabels (`coordinator:bug`, `coordinator:feature`, etc.)
   - Imports to configurable inbox (default: "coordinator")

2. **Messages → Tasks**: `MessageWatcher.poll()` converts unread messages to tasks
   - Polls messaging store via `ListUnread()`
   - Converts `Message` to `Task` with priority classification
   - Sends to task channel (non-blocking, 100-capacity buffer)

3. **Tasks → Execution**: Daemon's main loop processes tasks
   - Reads from task channel
   - Passes to `TaskAnalyzer` for classification + deduplication
   - Routes to appropriate executor/provider

---

### 2. **Message Adapter** (`message_adapter.go`)

**InboxMessageAdapter**: Bridges messaging.Store (collaboration.db) to coordinator's MessageStore interface

```go
// Implementation of MessageStore interface
ListUnread()    // Query inbox_messages table with UnreadOnly=true, Collapsed=true
MarkAsRead()    // Mark message as read in messaging.Store
```

**Inbox Types**:
- Each agent watches a specific inbox (e.g., "design-doc-creator", "sprint-planner")
- Creates per-agent inbox adapters in `initTaskProcessing()`
- Shared messaging.Store (single database connection)

---

### 3. **Task Analysis & Classification** (`analyzer.go`, `analyzer_test.go`)

**TaskAnalyzer**: Uses SimHash fingerprinting for duplicate detection + task type classification

#### Classification Logic (Priority Order):
1. **Bug/Fix** keywords: "bug", "fix", "error", "crash", "broken", "issue", "problem", "fail", "wrong"
2. **Test** keywords: "test", "coverage", "spec", "unittest", "integration test"
3. **Docs** keywords: "document", "docs", "readme", "comment", "example", "guide", "tutorial"
4. **Research** keywords: "research", "investigate", "explore", "analyze", "compare", "evaluate", "benchmark"
5. **Refactor** keywords: "refactor", "cleanup", "reorganize", "restructure", "simplify", "optimize"
6. **Feature** keywords: "add", "implement", "create", "new", "feature", "support", "enable"
7. **Unknown** (default)

#### Duplicate Detection:
- SimHash fingerprinting of content
- Hamming distance similarity check (default threshold: 0.8 = 80%)
- Registered fingerprints prevent re-processing

**Test Pattern Example**:
```go
func TestAnalyze_BugDetection(t *testing.T) {
    analyzer := NewTaskAnalyzer(0.8)
    task := &Task{
        ID:      "t1",
        Content: "There's a bug in the parser when it encounters...",
    }
    analyzed := analyzer.Analyze(task)
    if analyzed.Type != TaskTypeBugFix {
        t.Errorf("expected bug-fix, got %v", analyzed.Type)
    }
}
```

---

### 4. **GitHub Integration** (M-COORD-GITHUB-AUTO-ROUTING)

#### Components:

**GitHubPoster** (`github_poster.go`):
- Wraps `messaging.GitHubClient` for coordinator operations
- Methods: `PostComment()`, `AddLabel()`, `RemoveLabel()`, `CloseIssue()`, `GetLabels()`
- Auto-creates labels with predefined colors and descriptions
- Label Colors:
  - Routing: `coordinator:bug` (red), `coordinator:feature` (cyan), `coordinator:docs` (green)
  - Status: `needs-design-approval` (dark red), `needs-sprint-approval`, `needs-merge-approval`
  - Approval: `design-approved` (green), `sprint-approved`, `merge-approved`

**ApprovalWatcher** (`approval_watcher.go`):
- Polls GitHub issues for approval labels (default interval: 60s)
- Watches issues linked to tasks via `WatchIssue(issueNum, taskID)`
- Detects approval labels and triggers handlers
- Removes approval labels after processing (prevents re-triggering)
- Event Types: `ApprovalEventDesign`, `ApprovalEventSprint`, `ApprovalEventMerge`, `ApprovalEventRevision`

**TaskChain** (`task_chain.go`):
- Manages stage transitions: Design → Sprint → Implementation → Merge
- Registers approval handlers with ApprovalWatcher
- Posts stage-specific comments to GitHub issues
- Calls `StartTask()` to initialize GitHub-linked task
- Methods:
  - `OnDesignDocComplete()` → adds `needs-design-approval` label
  - `OnDesignApproved()` → transitions to sprint stage
  - `OnSprintPlanComplete()` → adds `needs-sprint-approval` label
  - `OnSprintApproved()` → transitions to implementation stage
  - `OnImplementationComplete()` → adds `needs-merge-approval` label
  - `OnMergeApproved()` → closes GitHub issue, clears stage
  - `OnNeedsRevision()` → pauses pipeline, posts revision comment
  - `OnError()` → posts error comment to GitHub

#### Approval Workflow:
```
GitHub Issue Created
    ↓
GitHub Sync: import-github → creates Message
    ↓
Daemon: Message → Task → AnalyzedTask
    ↓
Executor: AnalyzedTask → Design Document (adds needs-design-approval label)
    ↓
Human: Reviews on GitHub, adds design-approved label
    ↓
ApprovalWatcher: Detects label change
    ↓
TaskChain.OnDesignApproved(): Transitions to sprint stage
    ↓
[Repeat for sprint and implementation stages]
    ↓
Human: Approves merge
    ↓
TaskChain.OnMergeApproved(): Closes issue
```

---

### 5. **Task Storage** (`store.go`, `store_sqlite.go`, `store_sqlite_test.go`)

**Store Interface** (neutral, backend-agnostic):
- **CRUD**: CreateTask, GetTask, UpdateTask, DeleteTask
- **Queries**: ListTasks, GetTaskStats, GetTasksByGithubIssue, GetTasksByStage
- **State Transitions**: MarkTaskQueued, MarkTaskRunning, MarkTaskCompleted, MarkTaskFailed, etc.
- **GitHub Integration**: SetTaskGithubIssue, SetTaskStage, GetTasksByStage
- **Deduplication**: FindDuplicateTask, SetTaskFingerprint
- **Approvals**: CreateApprovalRequest, ListPendingApprovals, ResolveApprovalRequest
- **Events**: StoreTaskEvent, GetTaskEvents

**SQLiteStore Implementation** (~800 lines):
- Uses `sqlite3` driver
- Auto-creates schema on first use
- Normalized tables:
  - `tasks` - TaskRecord data
  - `task_events` - TaskEventRecord (streaming execution logs)
  - `approval_requests` - Approval workflow (future: GitHub label sync)

**Test Pattern**:
```go
func createTestStore(t *testing.T) Store {
    tmpDir := t.TempDir()
    store, err := NewSQLiteStore(filepath.Join(tmpDir, "test.db"))
    if err != nil {
        t.Fatalf("failed to create store: %v", err)
    }
    return store
}

func TestSQLiteStoreCreateAndGetTask(t *testing.T) {
    store := createTestStore(t)
    defer store.Close()
    ctx := context.Background()
    
    task := &TaskRecord{ /* ... */ }
    store.CreateTask(ctx, task)
    retrieved, _ := store.GetTask(ctx, task.ID)
    // assertions...
}
```

---

### 6. **Agent Configuration** (`agent_registry.go`, `agent_config.go`)

**AgentRegistry**: Manages configured agents and their inboxes

**Agent Configuration** (from `~/.ailang/config.yaml`):
```yaml
coordinator:
  agents:
    - id: design-doc-creator
      inbox: design-doc-creator          # Message inbox to watch
      workspace: /path/to/project        # Git workspace
      trigger_on_complete: [sprint-planner]  # Next agents to trigger
      auto_approve_handoffs: false       # Skip approval for agent-to-agent
      session_continuity: true           # Resume with --resume / --conversation-id
```

**Agent Responsibilities**:
- `design-doc-creator`: Reads GitHub issues, creates design docs (design stage)
- `sprint-planner`: Creates sprint plans from design (sprint stage)
- `sprint-executor`: Implements approved sprints (implementation stage)

Each agent gets:
- Dedicated `InboxMessageAdapter` for its inbox
- `WorktreeManager` for isolated git worktrees
- Task executor with provider routing

---

### 7. **Task Executors** (`provider.go`, `provider_claude.go`, `provider_gemini.go`)

**Provider Interface**:
```go
type Provider interface {
    Name() string
    CanHandle(task *AnalyzedTask) bool
    Execute(ctx context.Context, task *AnalyzedTask, opts *ExecuteOptions) (*ExecuteResult, error)
}
```

**Two Executor Types**:

1. **Executor-based Providers** (for code changes):
   - `ClaudeProvider` - Uses Claude Code CLI (`claude -p --output-format json`)
   - `GeminiProvider` - Uses Gemini CLI (`gemini --output-format json`)
   - Can edit files, run commands, implement features
   - Support session continuity (resume with `--resume` / `--conversation-id`)

2. **API-based Providers** (for text generation):
   - Uses `internal/ai/` API clients (Anthropic, OpenAI, Gemini, Ollama)
   - Direct text generation, no file editing
   - Fallback for specialized tasks

**ExecuteResult**:
```go
type ExecuteResult struct {
    Success    bool
    Output     string
    Error      string
    Provider   string
    Duration   time.Duration
    Cost       float64
    TokensUsed, InputTokens, OutputTokens int
    FilesCreated, FilesModified []string
    SessionID  string  // For resumption
}
```

---

### 8. **Approval Workflow** (`approval_checkpoint.go`, `approval_checkpoint_test.go`)

**ApprovalCheckpoint**: Synchronous approval requests (blocking wait for human decision)

```go
type ApprovalRequest struct {
    ID          string
    TaskID      string
    Type        ApprovalType // design, sprint, merge
    Title       string
    Description string
    AutoReject  bool          // Auto-reject if timeout reached
}

type ApprovalStatus string
const (
    ApprovalStatusPending   = "pending"
    ApprovalStatusApproved  = "approved"
    ApprovalStatusRejected  = "rejected"
    ApprovalStatusTimeout   = "timeout"
)
```

**Workflow**:
1. Agent creates approval request via `RequestApproval()`
2. Blocks until human decides or timeout
3. Returns `ApprovalStatus` to caller
4. If rejected: human can request revision via `Reject()`
5. If auto-reject enabled and timeout: auto-rejects

**Test Pattern**:
```go
func TestApprovalCheckpoint_Basic(t *testing.T) {
    ac := NewApprovalCheckpoint(1 * time.Hour)
    
    var status ApprovalStatus
    go func() {
        status, _ = ac.RequestApproval(ctx, &ApprovalRequest{/*...*/})
    }()
    
    time.Sleep(50 * time.Millisecond)
    ac.Approve("req-id", "user")
    
    if status != ApprovalStatusApproved { /* ... */ }
}
```

---

### 9. **Event Broadcasting** (`http_broadcaster.go`, `event_handler.go`)

**EventBroadcaster**: Real-time task execution updates to dashboard

- Connects to Collaboration Hub dashboard via POST/WebSocket
- Broadcasts task creation, execution start/completion
- Streams intermediate execution events (tool use, output, errors)
- Dashboard at `http://localhost:1957` displays live task progress

**Event Types**:
- `task:created` - New task added
- `task:started` - Task execution started
- `task:completed` - Task finished
- `task:failed` - Task errored
- `stream:text` - Text output
- `stream:tool_use` - Tool invocation
- `stream:tool_result` - Tool result
- `stream:error` - Error event

---

### 10. **Resource Tracking** (`resource_tracker.go`, `resource_tracker_test.go`)

**ResourceTrackerRegistry**: Monitors CPU, memory, tokens for running tasks

- Tracks metrics per task (peak CPU, peak memory)
- Updates `TaskRecord` after completion
- Useful for cost analysis and performance tuning
- Can limit resources (future: enforce limits)

---

## Current Test Coverage Analysis

### Test Files (14 total, 136 tests):

| File | Lines | Purpose |
|------|-------|---------|
| `approval_checkpoint_test.go` | 855 | Comprehensive approval workflow (sync, timeout, auto-reject) |
| `store_sqlite_test.go` | 410 | Task CRUD, state transitions, GitHub linking, statistics |
| `analyzer_test.go` | 352 | Task type classification, duplicate detection |
| `agent_registry_test.go` | 341 | Agent config loading, validation, inbox management |
| `worktree_test.go` | 323 | Git worktree creation, branch management, cleanup |
| `event_handler_test.go` | 283 | Event broadcast, WebSocket streaming |
| `watcher_test.go` | 309 | Message polling, task conversion, seen-message tracking |
| `provider_test.go` | 226 | Mock provider, execution options |
| `resource_tracker_test.go` | 226 | CPU/memory metrics, peak tracking |
| `integration_test.go` | 850 | Full task lifecycle: create → run → complete |
| `daemon_test.go` | 243 | Config, PID file, status reporting |
| `github_sync_test.go` | 81 | GitHub issue import interval handling |
| `merge_test.go` | 84 | Git merge operations |
| `handoff_test.go` | 146 | Agent-to-agent task handoff |

### Coverage Highlights:
- ✅ **Excellent**: Approval workflow, approval checkpoint, task storage, task analysis
- ✅ **Good**: Integration tests (full lifecycle), provider interface, event handling
- ⚠️ **Moderate**: GitHub sync (only interval tests, no mock GitHub client)
- ⚠️ **Gaps**:
  - ApprovalWatcher polling logic (no tests for label detection)
  - TaskChain stage transitions (no tests for comment posting, label management)
  - GitHubPoster (relies on real GitHub API, no mocks)
  - Multi-agent agent-to-agent triggering
  - Error recovery and task retries
  - Resource limits enforcement
  - Approval request persistence (store integration)

---

## GitHub API Client Usage

**Location**: `internal/messaging/github.go` (GitHubClient)

**Methods Used by Coordinator**:
- `AddComment(repo, issueNum, body)` - Post comment to issue
- `AddLabelToIssue(repo, issueNum, label)` - Add label
- `RemoveLabelFromIssue(repo, issueNum, label)` - Remove label
- `GetIssueLabels(repo, issueNum)` - List issue labels
- `CloseIssue(repo, issueNum, comment)` - Close issue with comment
- `EnsureLabel(repo, label, description, color)` - Create label if missing

**Authentication**: Via `gh` CLI (GitHub token in `~/.config/gh/hosts.yml`)

**Error Handling**: Mostly ignored with logging (potential improvement area)

---

## Message Sending & Task Triggering Flow

```
1. GitHub Issue Created
   ↓
2. Coordinator.runGitHubSync() [periodic, default: 5 min]
   → exec `ailang messages import-github --labels coordinator:bug`
   → Creates Message in collaboration.db (inbox_messages table)
   ↓
3. MessageWatcher.poll() [periodic, 30 sec default]
   → Queries ListUnread() from InboxMessageAdapter
   → Converts Message → Task
   → Sends to tasksChan (non-blocking)
   ↓
4. Daemon main loop
   → Receives Task from tasksChan
   → TaskAnalyzer.Analyze() classifies type, detects dupes
   → Routes to Provider (Claude or Gemini)
   → Creates TaskRecord in coordinator.db
   ↓
5. Provider.Execute()
   → Runs `claude` or `gemini` CLI with task content
   → Returns ExecuteResult (files changed, cost, tokens)
   ↓
6. TaskChain.OnDesignDocComplete()
   → Posts comment to GitHub issue
   → Adds needs-design-approval label
   ↓
7. ApprovalWatcher.pollOnce() [default: 60 sec]
   → Checks watched issues for approval labels
   → Detects design-approved label
   → Calls TaskChain.OnDesignApproved()
   ↓
8. TaskChain.OnDesignApproved()
   → Updates task stage: design → sprint
   → Posts "working on sprint plan" comment
   → [Process repeats for sprint/implementation]
```

---

## Test Patterns Used in Codebase

### 1. **Setup/Teardown Pattern** (most common):
```go
func TestSomething(t *testing.T) {
    // Setup
    tmpDir := t.TempDir()
    store, _ := NewSQLiteStore(filepath.Join(tmpDir, "test.db"))
    defer store.Close()  // Cleanup
    
    // Test
    // assertions...
}
```

### 2. **Helper Functions**:
```go
func createTestStore(t *testing.T) Store {
    tmpDir := t.TempDir()
    store, err := NewSQLiteStore(filepath.Join(tmpDir, "test.db"))
    if err != nil {
        t.Fatalf("failed to create store: %v", err)
    }
    return store
}

func TestSQLiteStoreCreateAndGetTask(t *testing.T) {
    store := createTestStore(t)
    defer store.Close()
    // ... test ...
}
```

### 3. **Mock Provider Pattern** (for testing executor flow):
```go
type IntegrationMockProvider struct {
    name       string
    handleFunc func(task *AnalyzedTask) bool
    execFunc   func(ctx context.Context, task *AnalyzedTask, opts *ExecuteOptions) (*ExecuteResult, error)
    execCount  int
}

func (m *IntegrationMockProvider) Execute(ctx context.Context, task *AnalyzedTask, opts *ExecuteOptions) (*ExecuteResult, error) {
    m.execCount++
    return m.execFunc(ctx, task, opts)
}

func TestIntegration_TaskLifecycle(t *testing.T) {
    provider := NewIntegrationMockProvider("mock")
    provider.SetExecuteFunc(func(...) (*ExecuteResult, error) {
        return &ExecuteResult{Success: true, ...}, nil
    })
    // ... assertions on provider.execCount, result ...
}
```

### 4. **Context + Goroutine Pattern** (for async operations):
```go
func TestApprovalCheckpoint_Basic(t *testing.T) {
    ac := NewApprovalCheckpoint(1 * time.Hour)
    
    var wg sync.WaitGroup
    var status ApprovalStatus
    
    wg.Add(1)
    go func() {
        defer wg.Done()
        status, _ = ac.RequestApproval(context.Background(), &ApprovalRequest{/*...*/})
    }()
    
    time.Sleep(50 * time.Millisecond)  // Let goroutine start
    ac.Approve("req-id", "user")        // Unblock
    
    wg.Wait()  // Wait for completion
    if status != ApprovalStatusApproved { /* ... */ }
}
```

### 5. **Table-Driven Tests** (not as common, but used):
```go
func TestClassifyTaskType(t *testing.T) {
    tests := []struct {
        name    string
        content string
        want    TaskType
    }{
        {"bug keyword", "There's a bug in...", TaskTypeBugFix},
        {"feature keyword", "Add support for...", TaskTypeFeature},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            if got := classifyTaskType(tt.content); got != tt.want {
                t.Errorf("got %v, want %v", got, tt.want)
            }
        })
    }
}
```

---

## Key Architectural Observations

### 1. **Separations of Concern**:
- **Messaging**: Isolated in `internal/messaging/` (SQLite, GitHub sync)
- **Task Processing**: Isolated in `internal/coordinator/` (execution, approval)
- **Executors**: Abstracted via `Provider` interface (supports Claude, Gemini, APIs)
- **Storage**: Neutral `Store` interface (SQLite now, cloud-ready design)

### 2. **GitHub Integration Strategy**:
- **Issue Import**: Delegated to `ailang messages import-github` CLI (decoupled)
- **Issue Management**: GitHubPoster wraps messaging.GitHubClient (reuses existing client)
- **Approval Workflow**: ApprovalWatcher polls labels (event-driven without webhooks)

### 3. **Agent-to-Agent Workflow**:
- Uses message system for handoff (Thread-based)
- Agents watch dedicated inboxes
- Task completion triggers next agent via `trigger_on_complete` config
- Session continuity via `--resume` / `--conversation-id` flags

### 4. **Error Handling**:
- ⚠️ **Issue**: GitHub API errors mostly logged, not propagated
- Many operations continue despite errors (graceful degradation)
- Could benefit from retry logic with exponential backoff

### 5. **Testing Strategy**:
- Unit tests: Individual component behavior (store, analyzer, provider)
- Integration tests: Full task lifecycle with mock providers
- No external service mocking: GitHub tests require real auth (limitation)
- Temp directories for all file I/O (good practice)

---

## Coverage Gaps for GitHub Integration Testing

### 1. **ApprovalWatcher Not Tested**:
- Label detection (checkIssueLabels)
- Event handler invocation
- Label removal after processing
- Watched issue loading from store
- Multi-watcher scenarios

### 2. **TaskChain Not Tested**:
- Stage transitions (design → sprint → implementation → merge)
- Comment posting flow
- Label addition/removal sequence
- GitHub issue error handling
- Multi-stage concurrent tasks

### 3. **GitHubPoster Not Tested**:
- Real API calls skipped (auth required)
- Label color/description handling
- Error recovery (rate limits, auth failures)
- Comment markdown formatting

### 4. **GitHub Sync Integration**:
- Only interval logic tested, not issue import
- No mock GitHub API for testing filtering by labels
- No verification of message creation from issues

---

## Recommendations for Test Improvement

1. **Add Mock GitHub Client**: Wrap `messaging.GitHubClient` in test doubles
   ```go
   type MockGitHubClient struct {
       comments     map[int][]string  // issue -> comments
       labels       map[int][]string  // issue -> labels
       closedIssues map[int]bool
   }
   ```

2. **Test ApprovalWatcher Label Detection**:
   ```go
   func TestApprovalWatcher_DetectsApprovalLabel(t *testing.T) {
       mockClient := &MockGitHubClient{ /*...*/ }
       poster := &GitHubPoster{client: mockClient}
       watcher := NewApprovalWatcher(poster, store, 100*time.Millisecond)
       
       // Add approval label
       mockClient.labels[123] = append(mockClient.labels[123], LabelDesignApproved)
       
       // Verify handler called
       watcher.RegisterHandler(ApprovalEventDesign, ...)
       watcher.Start(ctx)
       // assertions...
   }
   ```

3. **Test TaskChain Stage Transitions**:
   ```go
   func TestTaskChain_DesignToSprintTransition(t *testing.T) {
       // Create task in design stage
       // Complete design doc (OnDesignDocComplete)
       // Verify needs-design-approval label added
       // Simulate approval
       // Verify stage updated to sprint
   }
   ```

4. **Integration Test with Mock GitHub**:
   ```go
   func TestIntegration_GitHubIssueToMergedPR(t *testing.T) {
       // 1. Create GitHub issue (mock)
       // 2. Sync issues (import-github)
       // 3. Execute design doc
       // 4. Verify comment posted, label added
       // 5. Simulate approval
       // 6. Execute sprint plan
       // 7. Verify transition and next comment
       // ... continue through merge
   }
   ```

---

## Summary Table

| Aspect | Status | Notes |
|--------|--------|-------|
| **Core Architecture** | ✅ Solid | Clear separation, neutral interfaces |
| **GitHub Integration** | ⚠️ Functional | Works but lacks test coverage |
| **Approval Workflow** | ✅ Well-tested | Good checkpoint tests |
| **Task Storage** | ✅ Excellent | Comprehensive SQL tests |
| **Message Flow** | ⚠️ Basic tests | Only interval/config, not full flow |
| **Error Handling** | ⚠️ Weak | Mostly log-and-continue |
| **Provider Abstraction** | ✅ Good | Interface-based, testable |
| **Agent-to-Agent** | ⚠️ Untested | Config works, triggering untested |
| **Multi-Agent Routing** | ⚠️ Basic | Registry exists, no routing tests |
| **Resource Tracking** | ✅ Tested | Good metric tests |

