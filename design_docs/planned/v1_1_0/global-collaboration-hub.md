# Global Collaboration Hub - Cross-Computer Agent Collaboration

**Status**: Planned
**Target**: v0.6.0
**Priority**: P1 — High (downgraded from P0 per v1-mission iteration 0, Mark 2026-07-10: not v1.0-gating)
**Dependencies**: Collaboration Hub v2 (v0.5.0)

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Centralized state ensures consistent cross-machine ordering |
| A2: Replayability | +1 | Message history persisted, traces available for audit |
| A3: Effect Legibility | 0 | Infrastructure feature, not language-level effects |
| A4: Explicit Authority | +1 | IAM-based access control, project-level permissions |
| A5: Bounded Verification | 0 | No verification impact |
| A6: Safe Concurrency | +1 | Pub/Sub prevents races, ordered message delivery |
| A7: Machines First | +1 | API-first design, JSON schemas, no human prose |
| A8: Minimal Syntax | 0 | No syntax changes |
| A9: Cost Visibility | +1 | Detailed cost estimation table, usage tracking |
| A10: Composability | +1 | Modular architecture (Pub/Sub, SQL, Storage) |
| A11: Structured Failure | +1 | Error handling, retry logic, dead letter queues |
| A12: System Boundary | +1 | Clear local/global boundary, hybrid mode |

**Net Score: +9** ✅ Proceed to implementation

### Hard Violation Check

- [x] A1 (Determinism): Ordered message delivery with sequence numbers
- [x] A3 (Effects): Infrastructure feature, does not hide language-level effects
- [x] A4 (Authority): IAM roles enforce explicit access control
- [x] A7 (Machines First): API-first with JSON schemas, no prose-based config

## Problem Statement

The current Collaboration Hub is single-machine only:

**Current State (v0.5.6+):**
- SQLite database at `~/.ailang/state/collaboration.db` - local only
- WebSocket connections to `localhost:1957` - single machine
- **Unified inbox messaging in `collaboration.db`** (v0.5.6) - `inbox_messages` table
- CLI (`ailang messages`) and dashboard share same database
- REST API `/api/inbox` and WebSocket `inbox_message` events for real-time updates
- No way to coordinate agents across multiple computers
- No shared visibility into team-wide agent activity

**v0.5.6 Foundation (Completed):**
The local messaging system is now unified:
- `inbox_messages` table stores all agent-to-agent, agent-to-user, user-to-agent messages
- REST endpoints: `GET/POST /api/inbox`, `PUT /api/inbox/{id}`, `POST /api/inbox/ack-all`
- WebSocket broadcasts new messages to all connected dashboard clients
- CLI commands: `ailang messages list/send/ack/read/watch/cleanup`
- This provides the foundation for global cross-machine messaging

**Use Cases Blocked:**
1. **Team Collaboration**: Multiple developers want to see each other's agent runs
2. **Multi-Machine Agents**: Spawn agents on cloud VMs that coordinate with local IDE
3. **Shared Approval Queues**: Approve/reject requests from any device
4. **Centralized Monitoring**: Dashboard showing all team agent activity
5. **Cross-Project Coordination**: Agents in different repos collaborating

**Impact:**
- Agents are isolated silos per machine
- No team visibility into AI-assisted development
- Can't leverage cloud compute for expensive agent tasks
- Missing opportunity for cross-agent learning and coordination

## Goals

**Primary Goal:** Enable AILANG agents to collaborate across multiple computers with a global dashboard, shared message queue, and centralized state.

**Success Metrics:**
- Agent messages delivered across machines in <2 seconds
- 99.9% message delivery guarantee
- Support for 100+ concurrent agent connections
- <100ms dashboard update latency
- Zero data loss during network partitions (eventual consistency)
- Cost target: <$50/month for typical team usage

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Firestore over Cloud SQL for cloud state | Determines cost model, scaling strategy, and real-time architecture; locks in NoSQL data modeling | human | design | high |
| 100% serverless architecture (no always-on instances) | Eliminates Cloud SQL, shapes all deployment and operational patterns | human | design | high |
| CollaborationStore abstraction (SQLite/Firestore backends) | Core interface contract that all components depend on; wrong shape blocks hybrid mode | human | design | high |
| Cloud Pub/Sub as global message bus | Determines delivery guarantees (at-least-once), ordering model, and retry semantics | human | design | high |
| Hybrid mode with local-first sync strategy | Defines offline behavior, conflict resolution, and consistency model (last-write-wins with vector clocks) | human | design | med |
| Google Cloud IAM + OIDC for auth | Locks into GCP identity; no multi-cloud auth support | human | design | med |
| Firestore real-time listeners for dashboard (not custom WebSocket) | Simplifies architecture but couples React UI to Firestore SDK | human | design | med |
| At-least-once delivery with idempotency via message_id deduplication | Shapes all consumer code; exactly-once would require different infrastructure | human | design | high |

### Design Freeze

Before implementation begins, these must be resolved:

- [ ] Firestore schema finalized (collection hierarchy, indexing strategy, security rules)
- [ ] CollaborationStore interface signature frozen (all methods agreed upon)
- [ ] Pub/Sub topic naming and ordering key strategy locked
- [ ] Hybrid mode conflict resolution algorithm specified (vector clocks vs timestamps)
- [ ] Authentication flow finalized (OIDC provider, token lifetime, refresh strategy)
- [ ] Cost budget enforcement mechanism decided (Firestore triggers vs Cloud Run middleware)

## Solution Design

### Overview

A **100% serverless** Google Cloud architecture that extends the local Collaboration Hub:

1. **Cloud Pub/Sub** - Global message bus for agent communication
2. **Firestore** - Serverless database for all state (replaces Cloud SQL)
3. **Cloud Run** - Serverless API and WebSocket gateway
4. **Cloud Run Jobs** - Serverless task execution (Coordinator workers)
5. **Cloud Storage** - Large file attachments and artifacts
6. **Cloud IAM** - Authentication and authorization

**Key Design Decision: Firestore over Cloud SQL**

| Aspect | Cloud SQL | Firestore (Chosen) |
|--------|-----------|-------------------|
| Serverless | ❌ Always-on instance | ✅ Pay-per-operation |
| Min cost | ~$30/month | $0 (free tier) |
| Scaling | Manual | Automatic |
| Real-time | Requires WebSocket | Built-in listeners |
| Offline | N/A | Built-in support |

**Storage Abstraction Layer**: Both local (SQLite) and cloud (Firestore) backends implement a common `CollaborationStore` interface, enabling seamless switching between modes.

### Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    Global Collaboration Hub (100% Serverless)                │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌─────────────┐      ┌─────────────┐      ┌─────────────┐                 │
│  │  Machine A  │      │  Machine B  │      │  Machine C  │   (Clients)     │
│  │  (Dev IDE)  │      │  (Cloud VM) │      │ (Teammate)  │                 │
│  │  + Coord.   │      │  + Coord.   │      │             │                 │
│  └──────┬──────┘      └──────┬──────┘      └──────┬──────┘                 │
│         │                    │                    │                         │
│         └────────────────────┼────────────────────┘                         │
│                              │                                              │
│                              ▼                                              │
│  ┌───────────────────────────────────────────────────────────────────────┐ │
│  │                        Cloud Run (API Gateway)                         │ │
│  │  ┌─────────────┐  ┌─────────────────┐  ┌────────────────────────────┐ │ │
│  │  │  REST API   │  │ WebSocket Proxy │  │  Authentication Middleware │ │ │
│  │  │  /api/*     │  │  /ws            │  │  (Cloud IAM / OIDC)        │ │ │
│  │  └──────┬──────┘  └────────┬────────┘  └────────────────────────────┘ │ │
│  └─────────┼──────────────────┼──────────────────────────────────────────┘ │
│            │                  │                                             │
│            ▼                  ▼                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                       Google Cloud Pub/Sub                           │   │
│  │  ┌───────────────┐  ┌───────────────┐  ┌───────────────────────────┐ │   │
│  │  │ hub-messages  │  │ hub-approvals │  │ coordinator-tasks         │ │   │
│  │  │ (fanout)      │  │ (req/resp)    │  │ (task queue)              │ │   │
│  │  └───────────────┘  └───────────────┘  └───────────────────────────┘ │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                              │                                              │
│       ┌──────────────────────┼──────────────────────┐                      │
│       ▼                      ▼                      ▼                       │
│  ┌──────────────┐  ┌─────────────────────────┐  ┌─────────────────────┐    │
│  │ Cloud        │  │      Firestore          │  │ Cloud Run Jobs      │    │
│  │ Storage      │  │  (ALL state - unified)  │  │ (Coordinator)       │    │
│  │              │  │                         │  │                     │    │
│  │ • files      │  │ • messages & threads    │  │ • Task workers      │    │
│  │ • logs       │  │ • approvals & budgets   │  │ • Claude/Gemini     │    │
│  │ • artifacts  │  │ • machines & presence   │  │ • Git worktrees     │    │
│  │ • worktrees  │  │ • coordinator tasks     │  │                     │    │
│  └──────────────┘  │ • cost tracking         │  └─────────────────────┘    │
│                    │ • repo locks            │                              │
│                    └─────────────────────────┘                              │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘

Local Mode (SQLite):                     Cloud Mode (Firestore):
┌─────────────────────┐                  ┌─────────────────────┐
│ ~/.ailang/state/    │                  │ Firestore           │
│ collaboration.db    │  ◄──Interface──► │ hub/{project}/...   │
│ (SQLite)            │                  │                     │
└─────────────────────┘                  └─────────────────────┘
```

### Component Details

#### 1. Cloud Pub/Sub - Message Transport

**Topics:**

```
ailang-{project-id}-messages          # General agent messages (fanout)
ailang-{project-id}-commands          # Agent commands (point-to-point)
ailang-{project-id}-approvals         # Approval requests/responses
ailang-{project-id}-metrics           # Telemetry data
ailang-{project-id}-presence          # Agent online/offline status
```

**Subscriptions:**

```
# Per-machine subscriptions (ephemeral)
ailang-{project-id}-messages-{machine-id}     # Each machine gets all messages
ailang-{project-id}-commands-{agent-id}       # Agent-specific commands

# Global subscriptions (persistent)
ailang-{project-id}-messages-persist          # Write to Cloud SQL
ailang-{project-id}-metrics-persist           # Aggregate to Firestore
```

**Message Schema:**

```json
{
  "envelope": {
    "version": "2.0",
    "message_id": "uuid-v4",
    "timestamp": "2024-12-01T10:30:00Z",
    "source": {
      "machine_id": "machine-uuid",
      "agent_id": "sprint-planner",
      "project": "sunholo-data/ailang"
    },
    "destination": {
      "type": "broadcast" | "agent" | "machine",
      "target": "*" | "agent-id" | "machine-id"
    },
    "trace_id": "uuid-for-distributed-tracing",
    "signature": "hmac-sha256-signature"
  },
  "payload": {
    "type": "message" | "command" | "approval_request" | "approval_response",
    "thread_id": "thread-uuid",
    "content": { ... }
  }
}
```

**Delivery Guarantees:**
- At-least-once delivery (Pub/Sub default)
- Idempotency via message_id deduplication
- Message ordering within thread via ordering key
- Dead letter queue for failed deliveries
- 7-day message retention

#### 2. Storage Abstraction Layer

**Design Principle**: Single interface, multiple backends. Local development uses SQLite, cloud uses Firestore.

```go
// internal/hub/store.go

type CollaborationStore interface {
    // Messages
    CreateMessage(ctx context.Context, msg *Message) error
    GetMessage(ctx context.Context, id string) (*Message, error)
    ListMessages(ctx context.Context, opts ListOptions) ([]*Message, error)
    AckMessage(ctx context.Context, id string) error

    // Threads
    CreateThread(ctx context.Context, thread *Thread) error
    GetThread(ctx context.Context, id string) (*Thread, error)
    ListThreads(ctx context.Context, opts ListOptions) ([]*Thread, error)

    // Approvals
    CreateApproval(ctx context.Context, approval *ApprovalRequest) error
    GetApproval(ctx context.Context, id string) (*ApprovalRequest, error)
    UpdateApproval(ctx context.Context, id string, response *ApprovalResponse) error
    ListPendingApprovals(ctx context.Context) ([]*ApprovalRequest, error)

    // Machines & Presence
    RegisterMachine(ctx context.Context, machine *Machine) error
    HeartbeatMachine(ctx context.Context, id string) error
    ListMachines(ctx context.Context, projectID string) ([]*Machine, error)

    // Coordinator Tasks (see M-COORDINATOR design doc)
    CreateTask(ctx context.Context, task *Task) error
    GetTaskByExternalKey(ctx context.Context, key ExternalKey) (*Task, error)
    UpdateTaskStatus(ctx context.Context, id string, status TaskStatus) error
    ListPendingTasks(ctx context.Context, limit int) ([]*Task, error)

    // Cost Tracking
    RecordCost(ctx context.Context, taskID string, cost *CostRecord) error
    GetBudget(ctx context.Context) (*CostBudget, error)
    UpdateBudget(ctx context.Context, budget *CostBudget) error
    GetDailySpending(ctx context.Context, date time.Time) (*DailySpending, error)

    // Repo Locks (for Coordinator concurrency)
    AcquireRepoLock(ctx context.Context, repoURL, taskID string) error
    ReleaseRepoLock(ctx context.Context, repoURL string) error
    HeartbeatRepoLock(ctx context.Context, repoURL string) error
}
```

**Store Factory:**

```go
// internal/hub/store_factory.go

func NewStore(cfg *Config) (CollaborationStore, error) {
    switch cfg.Mode {
    case "local":
        dbPath := filepath.Join(cfg.StateDir, "collaboration.db")
        return NewSQLiteStore(dbPath)
    case "cloud":
        return NewFirestoreStore(ctx, cfg.CloudProjectID)
    default:
        return nil, fmt.Errorf("unknown mode: %s", cfg.Mode)
    }
}
```

#### 3. Firestore - Unified Cloud State

**Why Firestore (not Cloud SQL):**
- 100% serverless (pay-per-operation, no always-on instance)
- Built-in real-time listeners (no custom WebSocket needed)
- Automatic scaling
- Offline support built-in
- Free tier covers development
- ~$0.06/100K reads vs $30+/month Cloud SQL minimum

**Firestore Schema:**

```
hub/{project-id}/
├── config/
│   └── settings
│       ├── budget: {daily_limit_cents, monthly_limit_cents, ...}
│       └── coordinator: {enabled, max_workers, ...}
│
├── machines/{machine-id}
│   ├── name: "dev-laptop"
│   ├── owner: "user@example.com"
│   ├── last_seen: timestamp
│   ├── online: boolean
│   └── agents: [{id, status, ...}]
│
├── messages/{message-id}
│   ├── thread_id: "thread_123"
│   ├── from_agent: "sprint-planner"
│   ├── to_inbox: "user"
│   ├── source_machine: "machine_abc"
│   ├── pubsub_id: "msg_xyz"  (for deduplication)
│   ├── status: "unread"
│   ├── created_at: timestamp
│   └── payload: {...}
│
├── threads/{thread-id}
│   ├── title: "Fix parser bug"
│   ├── participants: ["user", "sprint-executor"]
│   ├── created_at: timestamp
│   └── last_activity: timestamp
│
├── approvals/{approval-id}
│   ├── task_id: "task_123"
│   ├── requesting_machine: "machine_abc"
│   ├── stage: "commit"
│   ├── status: "pending"
│   ├── approval_packet: {...}
│   └── created_at: timestamp
│
├── coordinator/
│   ├── tasks/{task-id}
│   │   ├── external_key: "message:ailang:msg_456"
│   │   ├── status: "running"
│   │   ├── task_type: "bug"
│   │   ├── workflow: "bug-fix"
│   │   └── ...
│   │
│   ├── worktrees/{task-id}
│   │   ├── path: "gs://..."
│   │   ├── branch: "coordinator/task_123"
│   │   ├── heartbeat_at: timestamp
│   │   └── status: "active"
│   │
│   ├── repo_locks/{repo-url-sanitized}
│   │   ├── locked_by: "task_123"
│   │   ├── locked_at: timestamp
│   │   └── heartbeat_at: timestamp
│   │
│   └── costs/{date}
│       ├── daily_spent_cents: 2340
│       └── breakdown: {claude: 1800, gemini: 540}
│
└── metrics/{period}/{timestamp}
    ├── runs: number
    ├── tokens: number
    └── cost: number
```

**Idempotency via Transactions:**

```go
func (s *FirestoreStore) CreateMessage(ctx context.Context, msg *Message) error {
    return s.client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
        // Check for duplicate via pubsub_id
        query := s.client.Collection(s.messagesPath()).
            Where("pubsub_id", "==", msg.PubSubID).
            Limit(1)

        docs, err := tx.Documents(query).GetAll()
        if err != nil {
            return err
        }
        if len(docs) > 0 {
            return nil // Idempotent: already exists
        }

        ref := s.client.Collection(s.messagesPath()).Doc(msg.ID)
        return tx.Set(ref, msg)
    })
}
```

**SQLite Schema (Local Mode):**

```sql
-- Extended for global compatibility
CREATE TABLE machines (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    owner TEXT NOT NULL,
    last_seen INTEGER NOT NULL,
    metadata TEXT DEFAULT '{}',
    created_at INTEGER DEFAULT (strftime('%s', 'now'))
);

-- inbox_messages extended for global
ALTER TABLE inbox_messages ADD COLUMN source_machine TEXT;
ALTER TABLE inbox_messages ADD COLUMN pubsub_id TEXT;

-- Coordinator tables
CREATE TABLE coordinator_tasks (
    id TEXT PRIMARY KEY,
    external_key TEXT UNIQUE NOT NULL,
    source TEXT NOT NULL,
    source_id TEXT NOT NULL,
    task_type TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    workflow TEXT,
    current_stage TEXT,
    created_at INTEGER NOT NULL,
    started_at INTEGER,
    completed_at INTEGER
);

CREATE TABLE repo_locks (
    repo_url TEXT PRIMARY KEY,
    locked_by TEXT NOT NULL,
    locked_at INTEGER NOT NULL,
    heartbeat_at INTEGER NOT NULL
);

CREATE TABLE cost_tracking (
    date TEXT PRIMARY KEY,
    daily_spent_cents INTEGER DEFAULT 0,
    breakdown TEXT DEFAULT '{}'
);
```

#### 4. Cloud Run - Serverless API Gateway

**Services:**

```yaml
# api-gateway - Main REST API
apiVersion: serving.knative.dev/v1
kind: Service
metadata:
  name: ailang-api
spec:
  template:
    spec:
      containers:
        - image: gcr.io/project/ailang-api
          env:
            - name: DB_CONNECTION
              valueFrom:
                secretKeyRef:
                  name: db-credentials
                  key: connection-string
          resources:
            limits:
              memory: 512Mi
              cpu: "1"
      containerConcurrency: 100

# ws-gateway - WebSocket connections
apiVersion: serving.knative.dev/v1
kind: Service
metadata:
  name: ailang-ws
  annotations:
    run.googleapis.com/session-affinity: "true"  # Sticky sessions for WebSocket
spec:
  template:
    spec:
      containers:
        - image: gcr.io/project/ailang-ws
          ports:
            - containerPort: 8080
      timeoutSeconds: 3600  # 1 hour for long-lived connections
```

**API Endpoints (extend existing):**

```
# Project/Org management
POST   /api/projects                    # Create project
GET    /api/projects/:id                # Get project details
PUT    /api/projects/:id/settings       # Update settings

# Machine registration
POST   /api/machines/register           # Register this machine
POST   /api/machines/heartbeat          # Keep-alive
GET    /api/machines                    # List machines in project

# Global views
GET    /api/global/hierarchy            # All agents across machines
GET    /api/global/metrics              # Aggregated metrics
GET    /api/global/activity             # Recent activity feed

# Authentication
POST   /api/auth/token                  # Get access token (OIDC)
POST   /api/auth/refresh                # Refresh token
GET    /api/auth/me                     # Current user info
```

#### 5. Cloud Storage - Large Files

**Bucket Structure:**

```
gs://ailang-{project-id}/
├── artifacts/
│   ├── {thread-id}/
│   │   ├── {message-id}/
│   │   │   ├── file1.txt
│   │   │   └── screenshot.png
├── logs/
│   ├── agents/
│   │   └── {agent-id}/{date}/execution.log
├── exports/
│   └── {export-id}.json
```

**Signed URLs for Client Access:**
```go
// Generate short-lived signed URL for file upload
url, err := storage.SignedURL(bucket, object, &storage.SignedURLOptions{
    Method:  "PUT",
    Expires: time.Now().Add(15 * time.Minute),
})
```

#### 6. Coordinator Daemon (Autonomous Task Execution)

**The Coordinator is an optional component** that enables fully autonomous development workflows. It monitors messages and automatically executes tasks via Claude Code or Gemini.

**Full specification**: See [M-COORDINATOR: Always-On Autonomous Development Daemon](../v0_7_0/m-coordinator-always-on-daemon.md)

**Key Features:**
- Git worktrees for conflict-free concurrent execution
- Human-in-the-loop with `<uncertainty>` tag protocol
- Cost budgets enforced via dashboard controls
- Dual-provider support (Claude Code headless + Gemini API)

**Cloud Mode Architecture:**

```
Pub/Sub (coordinator-tasks) → Cloud Run Jobs (workers) → Firestore (state)
                                      │
                                      ├─► Claude Code (headless)
                                      └─► Gemini API
```

**CLI Commands:**

```bash
# Local daemon mode
ailang coordinator start              # Start daemon
ailang coordinator stop               # Stop daemon
ailang coordinator status             # Show status

# Cloud mode (via Pub/Sub + Cloud Run Jobs)
# Configured via `ailang cloud` commands
```

**Integration with Hub:**
- Uses `CollaborationStore` interface (same as messaging)
- Tasks stored in `coordinator/tasks/` in Firestore
- Cost tracking integrated with Hub's budget system
- Approval requests sent to Hub's approval queue

**Real-time Listener (React):**
```typescript
// Subscribe to all machines' presence
const unsubscribe = onSnapshot(
  collection(db, `hub/${projectId}/machines`),
  (snapshot) => {
    const machines = snapshot.docs.map(doc => ({
      id: doc.id,
      ...doc.data()
    }));
    setConnectedMachines(machines);
  }
);
```

#### 7. Authentication & Authorization

**Strategy: Google Cloud IAM + Project-level Permissions**

```yaml
# IAM roles for AILANG
roles:
  ailang.viewer:
    - pubsub.subscriptions.consume
    - cloudsql.instances.connect (read-only)
    - storage.objects.get

  ailang.agent:
    - pubsub.topics.publish
    - pubsub.subscriptions.consume
    - cloudsql.instances.connect
    - storage.objects.create

  ailang.admin:
    - All of the above
    - projects.settings.update
    - machines.manage
```

**Client Authentication Flow:**

```
┌─────────────┐     ┌─────────────┐     ┌─────────────────┐
│   ailang    │     │  Cloud Run  │     │   Cloud IAM     │
│   CLI       │     │  API        │     │   (OIDC)        │
└──────┬──────┘     └──────┬──────┘     └────────┬────────┘
       │                   │                      │
       │ 1. ailang auth login                     │
       ├──────────────────────────────────────────►
       │                   │                      │
       │ 2. OAuth redirect │                      │
       │◄──────────────────┼──────────────────────┤
       │                   │                      │
       │ 3. Browser login  │                      │
       ├──────────────────────────────────────────►
       │                   │                      │
       │ 4. JWT token      │                      │
       │◄──────────────────┼──────────────────────┤
       │                   │                      │
       │ 5. API calls with Bearer token           │
       ├──────────────────►│                      │
       │                   │ 6. Validate token    │
       │                   ├─────────────────────►│
       │                   │◄─────────────────────┤
       │ 7. Response       │                      │
       │◄──────────────────┤                      │
```

### Hybrid Mode: Local + Global

**Critical Feature:** Support both local-only and global modes seamlessly.

```go
type CollaborationMode string

const (
    ModeLocal  CollaborationMode = "local"   // SQLite + local WebSocket
    ModeGlobal CollaborationMode = "global"  // Cloud SQL + Pub/Sub
    ModeHybrid CollaborationMode = "hybrid"  // Local primary, sync to global
)

// Configuration
type GlobalConfig struct {
    Mode          CollaborationMode
    ProjectID     string
    Region        string
    Credentials   string  // Path to service account JSON
    SyncInterval  time.Duration
    OfflineQueue  bool    // Queue messages when offline
}
```

**Hybrid Mode Sync Strategy:**

```
┌─────────────────────────────────────────────────────────────┐
│                      Hybrid Mode                             │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌─────────────────┐          ┌─────────────────────────┐  │
│  │  Local SQLite   │◄────────►│  Sync Engine            │  │
│  │  (Primary)      │          │  - Conflict resolution  │  │
│  └─────────────────┘          │  - Offline queue        │  │
│                               │  - Delta sync           │  │
│                               └───────────┬─────────────┘  │
│                                           │                 │
│                                           ▼                 │
│                               ┌─────────────────────────┐  │
│                               │  Cloud SQL              │  │
│                               │  (Secondary/Backup)     │  │
│                               └─────────────────────────┘  │
│                                                             │
└─────────────────────────────────────────────────────────────┘

Sync Rules:
1. Local writes are primary (low latency)
2. Async sync to global (eventual consistency)
3. Global writes pulled on subscription
4. Conflict resolution: Last-write-wins with vector clocks
5. Offline: Queue locally, sync when connected
```

### CLI Integration

**New Commands:**

```bash
# Project setup
ailang cloud init                       # Initialize global collaboration
ailang cloud login                      # Authenticate with Google Cloud
ailang cloud status                     # Show connection status

# Project management
ailang cloud project create NAME        # Create new project
ailang cloud project join PROJECT_ID    # Join existing project
ailang cloud project list               # List projects

# Machine management
ailang cloud machine register NAME      # Register this machine
ailang cloud machine list               # List machines in project

# Mode switching
ailang cloud mode local                 # Use local-only mode
ailang cloud mode global                # Use global mode
ailang cloud mode hybrid                # Use hybrid mode (default)

# Sync operations (hybrid mode)
ailang cloud sync                       # Force sync now
ailang cloud sync status                # Show sync status
ailang cloud sync conflicts             # List unresolved conflicts
```

**Environment Variables:**

```bash
AILANG_CLOUD_MODE=hybrid               # local | global | hybrid
AILANG_CLOUD_PROJECT=my-project-id     # Google Cloud project
AILANG_CLOUD_REGION=us-central1        # Preferred region
AILANG_CLOUD_CREDENTIALS=/path/to/sa.json  # Service account
```

### Implementation Plan

**Phase 1: Cloud Infrastructure Setup** (~12 hours)
- [ ] Create Terraform/Pulumi IaC for all GCP resources
- [ ] Set up Cloud SQL PostgreSQL instance
- [ ] Create Pub/Sub topics and subscriptions
- [ ] Configure Cloud Run services
- [ ] Set up Cloud Storage bucket
- [ ] Configure IAM roles and permissions
- [ ] Create development vs production environments

**Phase 2: Database Migration** (~8 hours)
- [ ] Add PostgreSQL driver to Go dependencies
- [ ] Create database abstraction layer (SQLite/PostgreSQL)
- [ ] Write schema migration scripts
- [ ] Implement connection pooling for Cloud SQL
- [ ] Add machine and project tables
- [ ] Test migration from existing SQLite databases
- [ ] Add rollback capability

**Phase 3: Pub/Sub Integration** (~10 hours)
- [ ] Create Pub/Sub client wrapper in Go
- [ ] Implement message publishing
- [ ] Implement subscription management
- [ ] Add message deduplication logic
- [ ] Implement dead letter handling
- [ ] Create ordering key strategy for threads
- [ ] Add retry and backoff logic
- [ ] Test message delivery guarantees

**Phase 4: Cloud Run API Gateway** (~10 hours)
- [ ] Containerize existing API server
- [ ] Add Cloud SQL Proxy sidecar
- [ ] Implement authentication middleware
- [ ] Add project/machine endpoints
- [ ] Create global hierarchy endpoint
- [ ] Deploy to Cloud Run
- [ ] Configure custom domain (optional)
- [ ] Set up health checks and monitoring

**Phase 5: Firestore Real-time** (~8 hours)
- [ ] Set up Firestore database
- [ ] Create presence system
- [ ] Implement real-time metrics sync
- [ ] Add activity feed
- [ ] Update React UI to use Firestore listeners
- [ ] Test offline/online transitions
- [ ] Configure security rules

**Phase 6: CLI Integration** (~6 hours)
- [ ] Add `ailang cloud` command group
- [ ] Implement OAuth flow for authentication
- [ ] Add project management commands
- [ ] Add machine registration
- [ ] Implement mode switching
- [ ] Add sync commands for hybrid mode
- [ ] Update configuration handling

**Phase 7: Hybrid Mode** (~8 hours)
- [ ] Implement sync engine
- [ ] Add offline queue
- [ ] Create conflict resolution logic
- [ ] Test network partition scenarios
- [ ] Add sync status indicators
- [ ] Implement selective sync (per-thread)

**Phase 8: Documentation & Testing** (~6 hours)
- [ ] Write setup guide
- [ ] Document cost estimation
- [ ] Create architecture diagrams
- [ ] Write integration tests
- [ ] Perform load testing
- [ ] Security audit

### Files to Create

**New Go packages:**

```
internal/cloud/
├── config.go           # Cloud configuration (~100 LOC)
├── pubsub.go           # Pub/Sub client wrapper (~300 LOC)
├── cloudsql.go         # Cloud SQL connection (~150 LOC)
├── storage.go          # Cloud Storage operations (~150 LOC)
├── firestore.go        # Firestore real-time sync (~200 LOC)
├── auth.go             # IAM/OIDC authentication (~200 LOC)
├── sync.go             # Hybrid mode sync engine (~400 LOC)
└── commands.go         # CLI commands (~300 LOC)

internal/messaging/
├── store_postgres.go   # PostgreSQL implementation (~400 LOC)
├── store_interface.go  # Common interface (~50 LOC)
└── migration.go        # Schema migration (~200 LOC)
```

**Infrastructure as Code:**

```
infra/
├── main.tf             # Terraform main config (~200 LOC)
├── pubsub.tf           # Pub/Sub resources (~100 LOC)
├── cloudsql.tf         # Cloud SQL instance (~100 LOC)
├── cloudrun.tf         # Cloud Run services (~150 LOC)
├── storage.tf          # Cloud Storage bucket (~50 LOC)
├── iam.tf              # IAM roles and bindings (~100 LOC)
├── firestore.tf        # Firestore setup (~50 LOC)
└── variables.tf        # Configuration variables (~100 LOC)
```

**React UI updates:**

```
ui/src/
├── providers/
│   └── FirestoreProvider.tsx   # Firestore context (~150 LOC)
├── hooks/
│   ├── usePresence.ts          # Real-time presence (~100 LOC)
│   ├── useGlobalMetrics.ts     # Global metrics (~80 LOC)
│   └── useActivityFeed.ts      # Activity stream (~80 LOC)
├── components/
│   ├── GlobalDashboard/        # Cross-machine view (~300 LOC)
│   ├── MachineList/            # Connected machines (~150 LOC)
│   └── ConnectionStatus/       # Online/offline indicator (~100 LOC)
```

### Cost Estimation

**Monthly costs for typical team (5 developers, 1000 agent runs/day):**

| Service | Usage | Estimated Cost |
|---------|-------|----------------|
| Firestore | 2M reads, 1M writes | $3.00 |
| Cloud Run (API) | 100K requests | $2.00 |
| Cloud Run Jobs (Coordinator) | 500 task executions | $2.00 |
| Pub/Sub | 1M messages | $0.40 |
| Cloud Storage | 10 GB | $0.26 |
| **Total** | | **~$8/month** |

**Note:** Firestore has a generous free tier (50K reads/day, 20K writes/day) that covers most development use cases.

**Enterprise scale (50 developers, 50K agent runs/day):**
- Firestore: $30 (10M reads, 5M writes)
- Cloud Run (API + Jobs): $25
- Pub/Sub: $20
- Storage: $10
- **Total: ~$85/month**

**Comparison with Cloud SQL approach:**
| Aspect | Cloud SQL | Firestore (Chosen) |
|--------|-----------|-------------------|
| Base cost | ~$30/month minimum | $0 (free tier) |
| Small team | ~$17/month | ~$8/month |
| Enterprise | ~$115/month | ~$85/month |
| Scaling | Manual (resize instance) | Automatic |

### Security Considerations

**Data Protection:**
- All data encrypted at rest (Firestore, Cloud Storage)
- TLS 1.3 for all connections
- VPC Service Controls for production
- No PII in agent messages (lint check)
- Firestore security rules for access control

**Access Control:**
- Project-level isolation
- IAM for service-to-service auth
- OIDC for user authentication
- Capability tokens for approvals (existing)

**Audit:**
- Cloud Audit Logs enabled
- Message signing with HMAC
- Approval history retention

## Success Criteria

- [ ] Agent messages delivered cross-machine in <2 seconds (p99)
- [ ] Global dashboard shows all machines' agents in real-time
- [ ] Approval requests can be handled from any machine
- [ ] Hybrid mode works seamlessly with local-first UX
- [ ] Cost under $50/month for team of 10
- [ ] Zero data loss during 30-second network partitions
- [ ] Documentation complete with setup guide
- [ ] Load test: 100 concurrent agents, 1000 msg/min

## Deferred Decisions

The following are intentionally left open for the implementer:

- Firestore index configuration and query optimization strategy — [agent may resolve]
- Cloud Run container sizing (memory/CPU limits) and concurrency tuning — [agent may resolve]
- Terraform vs Pulumi for IaC — [human may resolve]
- Signed URL expiration policy for Cloud Storage uploads — [agent may resolve]
- Sync interval tuning for hybrid mode (currently unspecified `SyncInterval`) — [agent may resolve]
- WebSocket gateway session affinity and reconnection strategy — [agent may resolve]
- Dead letter queue retry policy and alerting thresholds — [agent may resolve]

## Non-Goals

**Not in this feature:**
- Multi-tenant SaaS offering (single-project focus)
- Mobile apps (web dashboard only)
- Custom regions/data residency (GCP defaults)
- Agent-to-agent direct messaging (go through hub)
- Video/audio collaboration (text/files only)

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Cloud costs exceed budget | Medium | Implement cost alerts, auto-scaling limits |
| Network latency affects UX | Medium | Local-first hybrid mode, optimistic updates |
| Pub/Sub message ordering | High | Use ordering keys, sequence numbers |
| Data consistency conflicts | Medium | Vector clocks, last-write-wins, manual resolve |
| Authentication complexity | Medium | Use Google identity, simple OAuth flow |
| Cold start latency (Cloud Run) | Low | Keep minimum instances, connection pooling |

## Future Work

- **Multi-region replication** - Global low-latency access
- **Agent marketplace** - Share agents across projects
- **Team features** - User management, roles, permissions
- **Webhooks** - Integrate with external systems
- **Mobile app** - Approve requests from phone
- **Self-hosted option** - Run on own GCP/AWS

---

**Document created**: 2024-12-01
**Last updated**: 2024-12-04 (Updated with v0.5.6 unified messaging foundation)
