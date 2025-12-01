# Global Collaboration Hub - Cross-Computer Agent Collaboration

**Status**: Planned
**Target**: v0.6.0
**Priority**: P0 - High
**Dependencies**: Collaboration Hub v2 (v0.5.0)

## AI-First Alignment Check

**Score this feature against AILANG's core principles:**

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | 0 | 0 | Infrastructure feature, not language syntax |
| Preserve Semantic Clarity | ++ | +2 | Global visibility into all agent executions across machines |
| Increase Determinism | + | +1 | Centralized state, consistent cross-machine ordering |
| Lower Token Cost | ++ | +2 | Share context/results between agents, avoid duplicate work |
| **Net Score** | | **+5** | **Decision: Move forward** |

**Reference:** See [AI-first DX philosophy](../v0_3_15/example-parity-vision-alignment.md#-design-principle-ai-first-dx)

## Problem Statement

The current Collaboration Hub is single-machine only:

**Current State:**
- SQLite database at `~/.ailang/state/collaboration.db` - local only
- WebSocket connections to `localhost:1957` - single machine
- File-based agent protocol at `~/.ailang/state/messages/` - not synced
- No way to coordinate agents across multiple computers
- No shared visibility into team-wide agent activity

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

## Solution Design

### Overview

A serverless Google Cloud architecture that extends the local Collaboration Hub:

1. **Cloud Pub/Sub** - Global message bus for agent communication
2. **Cloud SQL (PostgreSQL)** - Shared database for state
3. **Cloud Run** - Serverless API and WebSocket gateway
4. **Cloud Storage** - Large file attachments and artifacts
5. **Firebase/Firestore** - Real-time dashboard sync (alternative to custom WebSocket)
6. **Cloud IAM** - Authentication and authorization

### Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         Global Collaboration Hub                             │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌─────────────┐      ┌─────────────┐      ┌─────────────┐                 │
│  │  Machine A  │      │  Machine B  │      │  Machine C  │   (Clients)     │
│  │  (Dev IDE)  │      │  (Cloud VM) │      │ (Teammate)  │                 │
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
│  │  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────────┐  │   │
│  │  │  agent-messages │  │  agent-commands │  │  agent-approvals    │  │   │
│  │  │  (fanout topic) │  │  (point-to-point)│ │  (request/response) │  │   │
│  │  └─────────────────┘  └─────────────────┘  └─────────────────────┘  │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                              │                                              │
│            ┌─────────────────┼─────────────────┐                           │
│            ▼                 ▼                 ▼                            │
│  ┌─────────────────┐  ┌─────────────┐  ┌─────────────────────────────────┐ │
│  │   Cloud SQL     │  │   Cloud     │  │       Firestore                 │ │
│  │  (PostgreSQL)   │  │   Storage   │  │  (Real-time Dashboard Sync)     │ │
│  │  - threads      │  │  - files    │  │  - agent status                 │ │
│  │  - messages     │  │  - logs     │  │  - live metrics                 │ │
│  │  - approvals    │  │  - artifacts│  │  - connection presence          │ │
│  │  - metrics      │  │             │  │                                 │ │
│  └─────────────────┘  └─────────────┘  └─────────────────────────────────┘ │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
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

#### 2. Cloud SQL (PostgreSQL) - Shared State

**Why PostgreSQL over SQLite:**
- Multi-connection support (agents across machines)
- ACID transactions for approval workflows
- Connection pooling via Cloud SQL Proxy
- Automatic backups and point-in-time recovery
- Easy migration from SQLite (compatible schema)

**Schema Changes:**

```sql
-- Add machine tracking
CREATE TABLE machines (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    owner TEXT NOT NULL,                 -- User/team identifier
    last_seen TIMESTAMPTZ NOT NULL,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Add project/org hierarchy
CREATE TABLE projects (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    org_id TEXT,                         -- Optional organization
    settings JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Extend agents table
ALTER TABLE agents ADD COLUMN machine_id TEXT REFERENCES machines(id);
ALTER TABLE agents ADD COLUMN project_id TEXT REFERENCES projects(id);
ALTER TABLE agents ADD COLUMN is_global BOOLEAN DEFAULT FALSE;

-- Extend messages for global delivery
ALTER TABLE messages ADD COLUMN source_machine TEXT;
ALTER TABLE messages ADD COLUMN delivered_to JSONB DEFAULT '[]';
ALTER TABLE messages ADD COLUMN pubsub_id TEXT;  -- For deduplication

-- Global message ordering
CREATE INDEX idx_messages_global_order ON messages(thread_id, created_at, seq);

-- Cross-machine approval tracking
ALTER TABLE approvals ADD COLUMN requesting_machine TEXT;
ALTER TABLE approvals ADD COLUMN approved_by_machine TEXT;
```

**Connection Strategy:**
```go
// Cloud SQL Proxy connection (from Cloud Run)
db, err := sql.Open("pgx", "host=/cloudsql/project:region:instance user=ailang dbname=collab")

// Local development fallback
if os.Getenv("AILANG_USE_LOCAL_DB") == "true" {
    db, err := sql.Open("sqlite3", "~/.ailang/state/collaboration.db")
}
```

#### 3. Cloud Run - Serverless API Gateway

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

#### 4. Cloud Storage - Large Files

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

#### 5. Firestore - Real-time Dashboard

**Why Firestore:**
- Built-in real-time listeners (no custom WebSocket needed)
- Scales automatically
- Offline support built-in
- Easy security rules

**Collections:**

```
/projects/{projectId}/
├── presence/
│   └── {machineId}
│       ├── online: boolean
│       ├── lastSeen: timestamp
│       └── agents: [{id, status}]
├── metrics/
│   └── {scope}/
│       └── {period}/
│           └── {timestamp}
│               ├── runs: number
│               ├── tokens: number
│               └── cost: number
├── activity/
│   └── {activityId}
│       ├── type: string
│       ├── agent: string
│       ├── machine: string
│       └── timestamp: timestamp
```

**Real-time Listener (React):**
```typescript
// Subscribe to all machines' presence
const unsubscribe = onSnapshot(
  collection(db, `projects/${projectId}/presence`),
  (snapshot) => {
    const machines = snapshot.docs.map(doc => ({
      id: doc.id,
      ...doc.data()
    }));
    setConnectedMachines(machines);
  }
);
```

#### 6. Authentication & Authorization

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
| Cloud SQL (db-f1-micro) | 730 hours | $8 |
| Cloud Run (API) | 100K requests | $2 |
| Cloud Run (WebSocket) | 500 instance hours | $5 |
| Pub/Sub | 1M messages | $0.40 |
| Cloud Storage | 10 GB | $0.26 |
| Firestore | 1M reads, 500K writes | $1.50 |
| **Total** | | **~$17/month** |

**Enterprise scale (50 developers, 50K agent runs/day):**
- Cloud SQL (db-g1-small): $50
- Cloud Run: $20
- Pub/Sub: $20
- Storage: $10
- Firestore: $15
- **Total: ~$115/month**

### Security Considerations

**Data Protection:**
- All data encrypted at rest (Cloud SQL, Storage, Firestore)
- TLS 1.3 for all connections
- VPC Service Controls for production
- No PII in agent messages (lint check)

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
**Last updated**: 2024-12-01
