# M-CLOUD-INFRA: GCP Cloud Infrastructure for AILANG Services

**Status**: Planned
**Target**: v0.8.0
**Priority**: P0 (High)
**Estimated**: 3-4 weeks
**Dependencies**: [M-GENERIC-PIPELINE](m-generic-pipeline.md) (config-driven agent pipelines)
**Author**: Claude + Mark
**Created**: 2026-01-16

---

## Executive Summary

Deploy AILANG services (coordinator daemon, dashboard server, agent executors) to GCP as always-on cloud infrastructure. This enables 24/7 autonomous agent execution, multi-user access, and production-scale operations (100+ tasks/day). The architecture uses Cloud Run for services, Cloud Run Jobs for ephemeral agent execution, Pub/Sub as message broker, Firestore for OLTP, and BigQuery for OLAP.

---

## Problem Statement

### Current State: Local-Only Architecture

The current AILANG infrastructure runs entirely on developer machines:

```
┌─────────────────────────────────────────────────────────────────┐
│ Current Architecture - Local Machine Only                        │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │              Developer Machine (macOS)                   │    │
│  │                                                          │    │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐   │    │
│  │  │ coordinator  │  │   server     │  │    ailang    │   │    │
│  │  │   daemon     │  │ (dashboard)  │  │     CLI      │   │    │
│  │  └──────┬───────┘  └──────┬───────┘  └──────────────┘   │    │
│  │         │                 │                              │    │
│  │         ▼                 ▼                              │    │
│  │  ┌────────────────────────────────────────────────────┐ │    │
│  │  │              SQLite Databases                       │ │    │
│  │  │  • coordinator.db  • collaboration.db  • observatory│ │    │
│  │  └────────────────────────────────────────────────────┘ │    │
│  │                                                          │    │
│  │  ┌────────────────────────────────────────────────────┐ │    │
│  │  │              Local Filesystem                       │ │    │
│  │  │  • ~/.ailang/state/worktrees/<agent>/              │ │    │
│  │  │  • ~/.ailang/logs/                                 │ │    │
│  │  │  • ~/.ailang/config.yaml                           │ │    │
│  │  └────────────────────────────────────────────────────┘ │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                  │
│  Pain Points:                                                    │
│  • Single user only (no collaboration)                          │
│  • Agents stop when laptop sleeps                               │
│  • No multi-region or scaling                                   │
│  • Manual SSH for remote access                                 │
│  • Data locked on single machine                                │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

**Pain Points:**
1. **No always-on execution** - Agents stop when developer machine sleeps
2. **Single user** - No multi-user collaboration or team dashboard
3. **No scalability** - Cannot scale beyond single machine resources
4. **Manual deployment** - Requires SSH/VNC for remote access
5. **Data isolation** - Trace data, tasks, and messages locked on local disk
6. **No redundancy** - Single point of failure

### Target User Persona

**Platform/DevOps Engineer** deploying AILANG for organization-wide autonomous coding:
- Needs 24/7 agent availability for CI/CD integration
- Manages multiple projects/workspaces simultaneously
- Requires cost tracking and budget controls
- Wants centralized dashboard for monitoring
- Needs auth integration (Google Cloud IAM)

---

## Goals

### Primary Goal

Deploy AILANG services to GCP with a **hybrid architecture** that maintains local development experience while enabling cloud-scale production operations.

### Success Metrics

| Metric | Current | Target |
|--------|---------|--------|
| Agent uptime | ~8h/day (laptop hours) | 24/7 (99.9%) |
| Concurrent users | 1 | 10+ |
| Task throughput | ~20/day (manual) | 100+/day (automated) |
| Data redundancy | None | Multi-zone |
| Deployment | Manual SSH | `terraform apply` |
| Cost visibility | None | Per-task, per-model |

---

## Solution Design

### Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                      AILANG GCP CLOUD ARCHITECTURE                          │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌──────────────────┐       ┌──────────────────────────────────────────┐   │
│  │ LOCAL DEV        │       │              GCP (us-central1)           │   │
│  │                  │       │                                          │   │
│  │ AILANG_STORAGE=  │       │  ┌────────────────────────────────────┐ │   │
│  │   local          │       │  │      Cloud Run Services            │ │   │
│  │                  │       │  │  ┌────────────┐ ┌────────────────┐ │ │   │
│  │ ┌──────────────┐ │  HTTPS│  │  │ Dashboard  │ │  Coordinator   │ │ │   │
│  │ │ SQLite DBs   │ │ ◄────►│  │  │ (always-on)│ │  (always-on)   │ │ │   │
│  │ │ • coord.db   │ │       │  │  └─────┬──────┘ └───────┬────────┘ │ │   │
│  │ │ • collab.db  │ │       │  └────────┼────────────────┼──────────┘ │   │
│  │ │ • observ.db  │ │       │           │                │            │   │
│  │ └──────────────┘ │       │           ▼                ▼            │   │
│  └──────────────────┘       │  ┌────────────────────────────────────┐ │   │
│                             │  │          Cloud Pub/Sub             │ │   │
│                             │  │  • ailang-inbox-{agent}            │ │   │
│                             │  │  • ailang-tasks                    │ │   │
│                             │  │  • ailang-events                   │ │   │
│                             │  └──────────────┬─────────────────────┘ │   │
│                             │                 │                       │   │
│                             │                 ▼                       │   │
│                             │  ┌────────────────────────────────────┐ │   │
│                             │  │     Cloud Run Jobs (Ephemeral)     │ │   │
│                             │  │  • Clone repo from GitHub          │ │   │
│                             │  │  • Run Claude/Gemini/Script        │ │   │
│                             │  │  • Push branch, report completion  │ │   │
│                             │  └────────────────────────────────────┘ │   │
│                             │                                         │   │
│                             │  ┌────────────────────────────────────┐ │   │
│                             │  │            Storage                 │ │   │
│                             │  │  ┌──────────┐    ┌──────────────┐ │ │   │
│                             │  │  │Firestore │    │  BigQuery    │ │ │   │
│                             │  │  │(OLTP)    │    │  (OLAP)      │ │ │   │
│                             │  │  │• tasks   │    │• spans       │ │ │   │
│                             │  │  │• messages│    │• analytics   │ │ │   │
│                             │  │  │• approvals│   │              │ │ │   │
│                             │  │  └──────────┘    └──────────────┘ │ │   │
│                             │  │                                   │ │   │
│                             │  │  ┌──────────────┐ ┌────────────┐ │ │   │
│                             │  │  │Secret Manager│ │Cloud Storage│ │ │   │
│                             │  │  │• API keys    │ │• artifacts │ │ │   │
│                             │  │  │• GitHub PAT  │ │• logs      │ │ │   │
│                             │  │  └──────────────┘ └────────────┘ │ │   │
│                             │  └────────────────────────────────────┘ │   │
│                             └──────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Component Breakdown

#### 1. Cloud Run Services (Always-On)

| Service | Responsibility | Resources | Cost Model |
|---------|----------------|-----------|------------|
| `ailang-dashboard` | HTTP server, WebSocket, REST API, React UI | 0.5 vCPU, 512MB | ~$25/mo |
| `ailang-coordinator` | Message broker: GitHub sync, Pub/Sub routing, approval workflow | 0.5 vCPU, 512MB | ~$20/mo |

**Coordinator as Message Broker (NOT executor):**

The coordinator daemon in cloud mode becomes a lightweight **message broker** that:
1. **Imports from external sources** - GitHub issues (via `runGitHubSync()`), future: email, webhooks
2. **Routes messages to Pub/Sub topics** - Each agent has a topic: `ailang-inbox-{agent-id}`
3. **Watches GitHub for approval labels** - ApprovalWatcher polls for `design-approved`, `sprint-approved`, etc.
4. **Triggers handoffs** - On approval, publishes to next agent's Pub/Sub topic

**Key distinction from local mode:**
| | Local Mode | Cloud Mode |
|---|------------|------------|
| Message source | SQLite polling | Pub/Sub subscription |
| Task execution | Direct (daemon calls provider) | Pub/Sub → Eventarc → Cloud Run Job |
| Event streaming | HTTP POST to localhost | Pub/Sub to dashboard |

```
┌────────────────────────────────────────────────────────────────────────────────┐
│                    COORDINATOR AS MESSAGE BROKER                                │
├────────────────────────────────────────────────────────────────────────────────┤
│                                                                                 │
│  EXTERNAL SOURCES                    COORDINATOR (Cloud Run Service)            │
│  ┌─────────────┐                    ┌──────────────────────────────────────┐   │
│  │ GitHub      │──(import-issues)──►│ • runGitHubSync() - polls issues      │   │
│  │ Issues      │                    │ • runLabelResync() - label routing    │   │
│  └─────────────┘                    │ • ApprovalWatcher - detects labels    │   │
│  ┌─────────────┐                    │ • TaskChain - stage transitions       │   │
│  │ Email       │──(future)────────►│                                        │   │
│  │ Webhooks    │                    │       ▼ (on new message)              │   │
│  └─────────────┘                    │ ┌────────────────────────────────┐   │   │
│                                     │ │ Publish to Pub/Sub topic:       │   │   │
│                                     │ │ ailang-inbox-{agent-id}         │   │   │
│                                     │ └─────────────┬──────────────────┘   │   │
│                                     └───────────────┼──────────────────────┘   │
│                                                     │                          │
│  AGENT EXECUTION                                    ▼                          │
│  ┌─────────────────────────────────────────────────────────────────────────┐  │
│  │                         Pub/Sub Topic                                    │  │
│  │                   ailang-inbox-design-doc-creator                        │  │
│  └─────────────────────────────────┬───────────────────────────────────────┘  │
│                                    │ Eventarc trigger                         │
│                                    ▼                                          │
│  ┌─────────────────────────────────────────────────────────────────────────┐  │
│  │                     Cloud Run Job                                        │  │
│  │  ailang coordinator execute-job                                          │  │
│  │  • Receives message payload from Pub/Sub                                 │  │
│  │  • Creates TaskRecord in Firestore                                       │  │
│  │  • Clones repo, creates branch                                           │  │
│  │  • Executes claude/gemini CLI (internal/executor/)                       │  │
│  │  • Commits, pushes branch                                                │  │
│  │  • Publishes completion to ailang-task-completions                       │  │
│  └─────────────────────────────────────────────────────────────────────────┘  │
│                                    │                                          │
│  COMPLETION HANDLING               ▼                                          │
│  ┌─────────────────────────────────────────────────────────────────────────┐  │
│  │ Coordinator receives completion event:                                   │  │
│  │ • Updates task status in Firestore                                       │  │
│  │ • Posts to GitHub (design doc comment, adds needs-approval label)        │  │
│  │ • If agent has trigger_on_complete: publishes to next agent's inbox      │  │
│  │ • ApprovalWatcher continues watching GitHub for approval labels          │  │
│  └─────────────────────────────────────────────────────────────────────────┘  │
│                                                                                │
└────────────────────────────────────────────────────────────────────────────────┘
```

#### Agent Pipeline Configuration: Generic vs Hardcoded

**IMPORTANT**: The codebase currently has TWO parallel pipeline systems. For cloud deployment, use the **generic config-driven approach** exclusively.

**See [M-GENERIC-PIPELINE](m-generic-pipeline.md) for the refactoring plan to unify these systems.**

**1. Hardcoded Stage Pipeline (DEPRECATED - see M-GENERIC-PIPELINE)**

Located in `internal/coordinator/stage_execution.go`, this system maps fixed `TaskStage` values to agent IDs:

```go
// stage_execution.go - hardcoded pipeline (local development only)
func stageToAgentIDForDirective(stage TaskStage) string {
    switch stage {
    case TaskStageDesign:
        return "design-doc-creator"
    case TaskStageSprint:
        return "sprint-planner"
    case TaskStageImplementation:
        return "sprint-executor"
    }
}

// Fixed stages: design → sprint → implementation → merge
```

**Problems with hardcoded stages:**
- Cannot add new agents without code changes
- Cannot reorder or skip pipeline stages
- Requires redeployment for any workflow change
- Tightly coupled to specific agent naming

**2. Generic Config-Driven Pipeline (RECOMMENDED)**

Located in `internal/coordinator/agent_registry.go`, agents define their handoff targets via `trigger_on_complete`:

```yaml
# ~/.ailang/config.yaml - flexible pipeline configuration
coordinator:
  agents:
    - id: design-doc-creator
      inbox: design-doc-creator
      trigger_on_complete: [sprint-planner]  # ← next agent(s)
      auto_approve_handoffs: false

    - id: sprint-planner
      inbox: sprint-planner
      trigger_on_complete: [sprint-executor]  # ← next agent(s)

    - id: sprint-executor
      inbox: sprint-executor
      trigger_on_complete: []  # ← end of pipeline

    # Add new agents without code changes!
    - id: code-reviewer
      inbox: code-reviewer
      trigger_on_complete: [sprint-executor]  # ← insert into pipeline
```

**Benefits of config-driven approach:**
- Add/remove agents without redeployment
- Reorder pipeline stages via config
- Support parallel handoffs (multiple agents in `trigger_on_complete`)
- Per-agent `auto_approve_handoffs` for workflow control
- Cloud-native: no hardcoded paths in containers

**Cloud Implementation:**

For cloud mode, the coordinator publishes to the next agent's Pub/Sub topic based on `trigger_on_complete`:

```go
// On task completion, check agent config for handoff targets
agent := registry.GetAgent(task.AgentID)
for _, nextAgentID := range agent.TriggerOnComplete {
    // Publish to ailang-inbox-{nextAgentID} topic
    broker.Publish(ctx, nextAgentID, handoffMessage)
}
```

**Migration path:**
1. Define all agent pipelines in `~/.ailang/config.yaml`
2. Set `COORDINATOR_MODE=cloud` to disable hardcoded stage execution
3. Hardcoded stages remain for backwards compatibility in local mode only

**Existing code to adapt for cloud broker mode:**
| File | Current Behavior | Cloud Change |
|------|------------------|--------------|
| `daemon_github.go:runGitHubSync()` | Imports via `ailang messages import-github` to SQLite | Publish to Pub/Sub topic instead |
| `daemon.go:pollAndProcessTasks()` | Polls SQLite → executes directly | Subscribe to Pub/Sub → publish to job trigger topic |
| `task_chain.go:OnDesignApproved()` | Requeues task in SQLite | Publish to `ailang-inbox-sprint-planner` |
| `approval_watcher.go` | Polls GitHub for labels | No change (GitHub polling still needed) |
| `http_broadcaster.go` | POSTs to localhost:1957 | Publish to `ailang-events` topic |

#### 2. Cloud Run Jobs (Ephemeral Agents)

| Job | Responsibility | Trigger | Resources |
|-----|----------------|---------|-----------|
| `ailang-agent-claude` | Claude Code CLI execution | Eventarc (Pub/Sub) | 2 vCPU, 4GB, 3hr timeout |
| `ailang-agent-gemini` | Gemini CLI execution | Eventarc (Pub/Sub) | 2 vCPU, 4GB, 3hr timeout |
| `ailang-agent-script` | Deterministic script execution | Eventarc (Pub/Sub) | 1 vCPU, 2GB, 3hr timeout |

#### 3. Storage Services

| Service | Data Type | Access Pattern | Current Interface |
|---------|-----------|----------------|-------------------|
| **Firestore** | Tasks, messages, approvals | OLTP (transactional) | `coordinator.Store` |
| **BigQuery** | Spans, traces, analytics | OLAP (analytical) | `observatory.Backend` |
| **Cloud Storage** | Worktree archives, logs | Blob storage | New |
| **Secret Manager** | API keys, GitHub PAT | Secrets | Env injection |

#### 4. Message Broker (Pub/Sub)

| Topic | Purpose | Subscribers |
|-------|---------|-------------|
| `ailang-inbox-{agent}` | Per-agent message queue | Coordinator daemon |
| `ailang-tasks` | Task dispatch to jobs | Cloud Run Jobs (Eventarc) |
| `ailang-events` | Real-time event streaming | Dashboard WebSocket |
| `ailang-approvals` | Approval workflow events | Dashboard + CLI |

**Pub/Sub Subscription Retry & Dead Letter Policy:**

All subscriptions use exponential backoff with dead letter topics to prevent message loss:

```hcl
# Example: ailang-tasks subscription (Terraform)
resource "google_pubsub_subscription" "tasks_coordinator" {
  name  = "ailang-tasks-coordinator"
  topic = google_pubsub_topic.tasks.name

  # Exponential backoff: 10s min, 600s max
  retry_policy {
    minimum_backoff = "10s"
    maximum_backoff = "600s"
  }

  # Dead letter after 5 failed attempts
  dead_letter_policy {
    dead_letter_topic     = google_pubsub_topic.dead_letter.id
    max_delivery_attempts = 5
  }

  # Ack deadline: 600s (10 min) — agent jobs can take hours,
  # but ack should happen once the job is *dispatched*, not completed.
  ack_deadline_seconds = 600

  # Retain unacked messages for 24 hours
  message_retention_duration = "86400s"
}
```

**Idempotency requirement (at-least-once delivery):**

Pub/Sub provides at-least-once delivery — duplicate messages are expected. All event handlers
MUST be idempotent:

- `execute-job`: Check Firestore task status before executing (skip if `running` or `completed`)
- Use Firestore transactions for status transitions (`pending` → `running`) to prevent races
- Store Pub/Sub message ID in task record for deduplication
- Completion handler: Use `task_id` as idempotency key — re-completing an already-completed task is a no-op

### Data Flow: Task Execution

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        TASK EXECUTION DATA FLOW                              │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  1. TASK CREATION                                                            │
│  ┌────────────┐                                                              │
│  │   User     │──── ailang messages send design-doc-creator "..." ────►     │
│  │   CLI      │                                                              │
│  └────────────┘                                                              │
│        │                                                                     │
│        ▼                                                                     │
│  ┌────────────────┐     ┌────────────────┐     ┌────────────────┐           │
│  │ Pub/Sub Topic  │────►│ Firestore      │────►│ Coordinator    │           │
│  │ ailang-inbox-  │     │ messages/      │     │ Daemon (polls) │           │
│  │ design-doc-    │     │ {msg-id}       │     │                │           │
│  │ creator        │     │                │     │                │           │
│  └────────────────┘     └────────────────┘     └────────────────┘           │
│                                                       │                      │
│  2. TASK DISPATCH                                     │                      │
│        ┌──────────────────────────────────────────────┘                      │
│        │                                                                     │
│        ▼                                                                     │
│  ┌────────────────┐     ┌────────────────┐     ┌────────────────┐           │
│  │ Task Analyzer  │────►│ Pub/Sub Topic  │────►│ Eventarc       │           │
│  │ • Classify     │     │ ailang-tasks   │     │ Trigger        │           │
│  │ • Deduplicate  │     │                │     │                │           │
│  │ • Route        │     │                │     │                │           │
│  └────────────────┘     └────────────────┘     └────────────────┘           │
│                                                       │                      │
│  3. AGENT EXECUTION (Cloud Run Job)                   │                      │
│        ┌──────────────────────────────────────────────┘                      │
│        │                                                                     │
│        ▼                                                                     │
│  ┌──────────────────────────────────────────────────────────┐               │
│  │                Cloud Run Job Instance                     │               │
│  │  ┌────────────────────────────────────────────────────┐  │               │
│  │  │ 1. Pull task payload from Pub/Sub message          │  │               │
│  │  │ 2. Clone repo (GitHub PAT from Secret Manager)     │  │               │
│  │  │ 3. Create branch: coordinator/{task-id}            │  │               │
│  │  │ 4. Run Claude Code CLI / Gemini CLI                │  │               │
│  │  │ 5. Commit changes                                  │  │               │
│  │  │ 6. Push branch to GitHub                           │  │               │
│  │  │ 7. Report completion via Pub/Sub                   │  │               │
│  │  └────────────────────────────────────────────────────┘  │               │
│  └──────────────────────────────────────────────────────────┘               │
│        │                                                                     │
│  4. APPROVAL WORKFLOW                                                        │
│        │                                                                     │
│        ▼                                                                     │
│  ┌────────────────┐     ┌────────────────┐     ┌────────────────┐           │
│  │ Firestore      │────►│ Dashboard      │────►│ Human Review   │           │
│  │ approvals/     │     │ WebSocket      │     │ • View diff    │           │
│  │ {task-id}      │     │ broadcast      │     │ • Approve/     │           │
│  │                │     │                │     │   Reject       │           │
│  └────────────────┘     └────────────────┘     └────────────────┘           │
│        │                                                                     │
│  5. MERGE & HANDOFF                                                          │
│        │                                                                     │
│        ▼                                                                     │
│  ┌────────────────┐     ┌────────────────┐     ┌────────────────┐           │
│  │ On Approve:    │────►│ Merge to dev   │────►│ Trigger next   │           │
│  │ gh pr merge    │     │ branch         │     │ agent (if      │           │
│  │                │     │                │     │ configured)    │           │
│  └────────────────┘     └────────────────┘     └────────────────┘           │
│                                                                              │
└──────────────────────────────────────────────────────────────────────────────┘
```

### Backend Selection Pattern

```go
// internal/storage/backend.go

type Backends struct {
    Coordinator coordinator.Store
    Messaging   messaging.Store
    Observatory observatory.Backend
}

func NewBackends(ctx context.Context) (*Backends, error) {
    mode := os.Getenv("AILANG_STORAGE") // local | gcp | hybrid
    switch mode {
    case "local", "":
        return newSQLiteBackends()
    case "gcp":
        return newGCPBackends(ctx)
    case "hybrid":
        // SQLite for coordinator (fast writes), BigQuery for observatory (analytics)
        return newHybridBackends(ctx)
    default:
        return nil, fmt.Errorf("unknown storage mode: %s", mode)
    }
}
```

### Firestore Collections

```
/tasks/{task_id}
  ├── id: string
  ├── status: "pending" | "queued" | "running" | "completed" | "failed"
  ├── agent_id: string
  ├── message_id: string
  ├── parent_task_id: string (for hierarchy)
  ├── iteration: number
  ├── worktree_path: string (Cloud Storage path)
  ├── session_id: string (Claude/Gemini session)
  ├── created_at: timestamp
  ├── started_at: timestamp
  ├── completed_at: timestamp
  ├── cost: number
  ├── tokens_in: number
  ├── tokens_out: number
  └── result: object

/messages/{message_id}
  ├── id: string
  ├── message_id: string (formatted: msg_DATE_SHORTID)
  ├── to_inbox: string
  ├── from_agent: string
  ├── title: string
  ├── payload: string
  ├── status: "unread" | "read" | "archived"
  ├── category: string
  ├── github_issue: number
  ├── simhash: number (for dedup)
  ├── embedding: array<number>
  ├── created_at: timestamp
  └── read_at: timestamp

/approvals/{approval_id}
  ├── id: string
  ├── task_id: string
  ├── type: "merge" | "merge_handoff" | "human"
  ├── status: "pending" | "approved" | "rejected"
  ├── context_json: string
  ├── resolved_by: string
  ├── created_at: timestamp
  └── resolved_at: timestamp

/agents/{agent_id}
  ├── id: string
  ├── label: string
  ├── inbox: string
  ├── workspace: string
  ├── capabilities: array<string>
  ├── provider: string
  ├── trust_scores: map<string, number>
  └── config_json: string

/workspaces/{workspace_id}
  # NOTE: Document ID encoding - "/" replaced with "__"
  # Example: "sunholo-data/ailang" → document ID "sunholo-data__ailang"
  # This is required because Firestore doesn't allow "/" in document IDs
  ├── id: string              # Original workspace ID with "/" (e.g., "sunholo-data/ailang")
  ├── name: string            # Display name (e.g., "AILANG Project")
  ├── github_repo: string     # GitHub repo identifier (same as id)
  ├── is_public: boolean      # If true, visible to all users
  ├── created_at: timestamp
  └── created_by: string      # "cli" or user email

/workspace_access/{workspace_id}/users/{email}
  # Subcollection for per-user access grants
  # workspace_id is encoded (same as parent collection)
  # email is used as document ID (allows @ and .)
  ├── email: string
  ├── workspace_id: string    # Original workspace ID with "/"
  ├── role: "Viewer" | "Approver"
  ├── granted_at: timestamp
  └── granted_by: string      # "cli" or admin email

# REQUIRED FIRESTORE INDEXES:
# The workspace_access subcollection uses collection group queries.
# Create this index in Firebase Console → Firestore → Indexes:
#   - Collection group: "users"
#   - Field: "email" (Ascending)
#   - Query scope: Collection group
```

### BigQuery Schema (Observatory)

```sql
-- observatory.spans table (time-partitioned)
CREATE TABLE observatory.spans (
  id STRING NOT NULL,
  trace_id STRING NOT NULL,
  parent_span_id STRING,
  task_id STRING,
  agent_assignment_id STRING,
  name STRING NOT NULL,
  kind STRING NOT NULL,  -- "internal" | "client" | "server"
  status STRING NOT NULL,  -- "ok" | "error" | "unset"
  start_time TIMESTAMP NOT NULL,
  end_time TIMESTAMP,
  duration_ms INT64,
  tokens_in INT64,
  tokens_out INT64,
  cost_usd FLOAT64,
  model STRING,
  provider STRING,
  attributes JSON,
  created_at TIMESTAMP NOT NULL
)
PARTITION BY DATE(start_time)
CLUSTER BY trace_id, task_id;
```

---

## Implementation Plan

### Phase 1: Backend Abstractions (Week 1)

**M1: Storage Interface Layer** (~300 LOC)
- [ ] Create `internal/storage/backend.go` with `Backends` struct
- [ ] Extract interface from `internal/messaging/store.go`
- [ ] Add `AILANG_STORAGE` environment variable parsing
- [ ] Factory functions for local vs GCP backends

**M2: Firestore Coordinator Store** (~400 LOC)
- [ ] Implement `coordinator.Store` interface with Firestore
- [ ] Task CRUD operations
- [ ] Approval workflow methods
- [ ] Task events streaming

**M3: Firestore Messaging Store** (~300 LOC)
- [ ] Implement messaging store with Firestore
- [ ] Inbox message CRUD
- [ ] SimHash/embedding storage
- [ ] GitHub sync metadata

**M4: BigQuery Observatory Backend** (~250 LOC)
- [ ] Implement `observatory.Backend` interface
- [ ] Streaming inserts for spans
- [ ] Query translation for filters
- [ ] Aggregation queries

### Phase 2: Cloud Run Source Deploy (Week 1-2)

**Simplified approach**: Use `gcloud run deploy --source .` instead of Dockerfiles.
Cloud Run's buildpacks auto-detect Go via `go.mod`, build the binary, and deploy —
no Dockerfile maintenance required.

See: [Cloud Run source deployment docs](https://cloud.google.com/run/docs/deploying-source-code)
See: [Google Cloud buildpacks for Go](https://cloud.google.com/docs/buildpacks/go)

**M5: Source Deploy Configuration** (~50 LOC)
- [ ] `.gcloudignore` to exclude `ui/node_modules/`, `.git/`, test fixtures
- [ ] Pre-build UI assets: `cd ui && npm run build` before deploy (go:embed picks them up)
- [ ] `GOOGLE_BUILDABLE=./cmd/ailang` build env var to target the correct Go package
- [ ] Validate `go:embed` works with buildpacks (embedded UI, stdlib, prompts)
- [ ] Use `--command`/`--args` flags on `gcloud run deploy` (NOT Procfile — avoids conflicts)

**M6: Deploy Scripts** (~100 LOC)
- [ ] `scripts/cloud-deploy-dashboard.sh` - Source deploy with env vars
- [ ] `scripts/cloud-deploy-coordinator.sh` - Source deploy with env vars
- [ ] `Makefile` targets: `make cloud-deploy-dashboard`, `make cloud-deploy-coordinator`

**M7: Cloud Execution Command** (~200 LOC)
- [ ] `cmd/ailang/coordinator_cloud.go` - New `execute-job` subcommand
- [ ] Fetch task from Firestore via storage backends
- [ ] Clone repo and create branch (reuse WorktreeManager patterns)
- [ ] Execute via existing TaskExecutor → Provider → Executor chain
- [ ] Commit/push changes (reuse existing git helpers)
- [ ] Report completion via Pub/Sub
- [ ] Agent executor still needs Dockerfile (requires Claude CLI + Gemini CLI + Node.js)

### Phase 3: Terraform Infrastructure (Week 2-3)

**M8: Core Infrastructure** (~300 LOC)
- [ ] `infra/terraform/main.tf` - Provider, APIs, VPC
- [ ] `infra/terraform/variables.tf` - Configuration
- [ ] `infra/terraform/outputs.tf` - Service URLs

**M9: Storage Resources** (~200 LOC)
- [ ] `infra/terraform/firestore.tf` - Database + indexes
- [ ] `infra/terraform/bigquery.tf` - Dataset + tables
- [ ] `infra/terraform/secrets.tf` - Secret Manager

**M10: Pub/Sub Resources** (~150 LOC)
- [ ] `infra/terraform/pubsub.tf` - Topics + subscriptions
- [ ] Dead letter topics
- [ ] Retry policies

**M11: Cloud Run Resources** (~200 LOC)
- [ ] `infra/terraform/cloud_run.tf` - Services
- [ ] `infra/terraform/cloud_run_jobs.tf` - Jobs + Eventarc

### Phase 4: CLI Integration (Week 3)

**M12: Cloud Commands** (~150 LOC)
- [ ] `ailang cloud status` - Show deployment status
- [ ] `ailang cloud migrate` - Export local → import cloud
- [ ] `ailang cloud logs` - Stream Cloud Logging

**M13: Makefile Targets** (~50 LOC)
- [ ] `make docker-build` - Build all containers
- [ ] `make docker-push` - Push to Artifact Registry
- [ ] `make cloud-deploy` - Run terraform apply
- [ ] `make cloud-destroy` - Teardown

### Phase 5: Testing & Documentation (Week 4)

**M14: Integration Tests** (~200 LOC)
- [ ] Backend switching tests
- [ ] Firestore store tests (with emulator)
- [ ] Pub/Sub integration tests

**M15: Documentation**
- [ ] Deployment guide
- [ ] Architecture diagram
- [ ] Cost optimization tips

---

## Files to Create/Modify

### New Files

| File | LOC | Purpose |
|------|-----|---------|
| `internal/storage/backend.go` | ~100 | Backend selector |
| `internal/storage/firestore/coordinator.go` | ~400 | Firestore coordinator store |
| `internal/storage/firestore/messaging.go` | ~300 | Firestore messaging store |
| `internal/storage/bigquery/observatory.go` | ~250 | BigQuery observatory |
| `.gcloudignore` | ~15 | Exclude files from source upload |
| `scripts/cloud-deploy-dashboard.sh` | ~30 | Dashboard source deploy script |
| `scripts/cloud-deploy-coordinator.sh` | ~30 | Coordinator source deploy script |
| `infra/docker/Dockerfile.agent-executor` | ~60 | Agent job container (needs CLIs) |
| `cmd/ailang/coordinator_cloud.go` | ~200 | Cloud Run Job execution command |
| `infra/terraform/main.tf` | ~150 | Terraform provider |
| `infra/terraform/firestore.tf` | ~80 | Firestore config |
| `infra/terraform/pubsub.tf` | ~100 | Pub/Sub config |
| `infra/terraform/cloud_run.tf` | ~120 | Cloud Run services |
| `infra/terraform/cloud_run_jobs.tf` | ~100 | Cloud Run jobs |
| `infra/terraform/bigquery.tf` | ~60 | BigQuery config |
| `infra/terraform/secrets.tf` | ~50 | Secrets config |
| `infra/terraform/variables.tf` | ~40 | Variables |
| `infra/terraform/outputs.tf` | ~30 | Outputs |
| **Total** | **~2,180** | |

### Modified Files

| File | Changes |
|------|---------|
| `internal/coordinator/store.go` | Add backend factory |
| `internal/messaging/store.go` | Extract interface |
| `internal/server/server.go` | Use backend selector |
| `internal/observatory/backend.go` | Add BigQuery backend |
| `cmd/ailang/main.go` | Add cloud subcommands |
| `Makefile` | Add cloud-deploy targets |

### New Files (Planned)

| File | LOC | Purpose |
|------|-----|---------|
| `internal/executor/codex/codex.go` | ~250 | Codex CLI executor (mirrors claude/gemini pattern) |

---

## Implementation Notes: Existing Code Patterns to Reuse

**CRITICAL: The cloud implementation MUST reuse existing infrastructure, NOT reinvent it.**

### Executor Package (`internal/executor/`)

The executor package provides the CLI invocation abstraction:

```go
// internal/executor/executor.go - Interface definition
type Executor interface {
    Execute(ctx context.Context, task *Task) (*Result, error)
    ExecuteStreaming(ctx context.Context, task *Task, events chan<- Event) (*Result, error)
}

// internal/executor/claude/claude.go - Claude CLI wrapper
// Key function: ExecuteStreaming() builds and runs claude command
args := []string{
    "-p", task.Directive,
    "--output-format", "stream-json",
    "--permission-mode", e.permissionMode,
    "--model", e.getModel(task),
    "--session-id", sessionID,
}
cmd := exec.CommandContext(ctx, e.claudePath, args...)
```

### Environment Setup (`internal/executor/environment.go`)

All environment variable setup is centralized:

```go
// BuildEnvironment() creates common env vars for AI executor processes
// - AILANG_STDLIB_PATH
// - TRACEPARENT (W3C trace context)
// - AILANG_TASK_ID, AILANG_SESSION_ID
// - OTEL_RESOURCE_ATTRIBUTES
// - AILANG_CLOUD_PROJECT
```

### Coordinator Providers (`internal/coordinator/provider_*.go`)

Providers wrap executors with coordinator-specific logic:

```go
// ClaudeCodeProvider - Uses internal/executor/claude
// GeminiCLIProvider - Uses internal/executor/gemini
// CodexProvider     - Uses internal/executor/codex (planned)
// ScriptProvider    - Direct shell execution

// provider.go interface
type Provider interface {
    Execute(ctx context.Context, task *AnalyzedTask, opts *ExecuteOptions) (*ExecuteResult, error)
    Name() string
}
```

### Task Execution Flow (`internal/coordinator/daemon_tasks.go`)

Current daemon execution flow to mirror:

```go
func (d *Daemon) executeTask(ctx context.Context, task *TaskRecord) error {
    // 1. Load agent config
    agent := d.agentRegistry.GetAgent(task.AgentID)

    // 2. Build directive from config + skill SKILL.md files
    directive := BuildDirectiveFromConfig(task, agent)

    // 3. Create analyzed task
    analyzed := &AnalyzedTask{...}

    // 4. Execute via provider
    result, err := d.executor.Execute(ctx, analyzed, opts)

    // 5. Handle completion (approval, handoffs)
}
```

### Worktree Management (`internal/coordinator/worktree.go`)

Patterns for git operations:

```go
// WorktreeManager handles isolated git worktrees
// - CreateWorktree(taskID, agentID) - creates branch and worktree
// - CleanupWorktree(taskID) - removes worktree
// - Uses git worktree add/remove commands
```

### Key Files Reference

| Existing File | Purpose | Cloud Equivalent |
|--------------|---------|------------------|
| `internal/executor/claude/claude.go:100-150` | Claude CLI invocation | Same code, different trigger |
| `internal/executor/environment.go:52-143` | Environment setup | Reuse directly |
| `internal/coordinator/provider_claude.go` | Provider wrapper | Same provider, Firestore storage |
| `internal/coordinator/daemon_tasks.go:150-250` | Execution orchestration | Cloud execute-job command |
| `internal/coordinator/store.go` | Store interface | Implement with Firestore |

---

## Cost Estimate (100+ tasks/day)

| Service | Usage | Monthly Cost |
|---------|-------|-------------|
| Cloud Run (Dashboard) | 1 instance, 24/7, request-based billing | ~$25 |
| Cloud Run (Coordinator) | 1 instance, 24/7, instance-based billing (`--no-cpu-throttling`) | ~$35 |
| Cloud Run Jobs | 100 jobs/day, 30min avg (3hr max), 2 vCPU | ~$45 |
| Firestore | 100K reads, 50K writes/day | ~$10 |
| BigQuery | 50GB storage, 1TB queries | ~$15 |
| Pub/Sub | 1M messages/month | ~$5 |
| Cloud Storage | 10GB artifacts | ~$1 |
| Secret Manager | 5 secrets, 10K access | ~$1 |
| Networking | Internal only (no LB — IAP direct on Cloud Run) | ~$2 |
| **Total** | | **~$140/month** |

**Note:** AI provider costs are NOT included above. With Claude Code Max subscription(s):
- 1x Max 20x: +$200/mo (covers most workloads)
- 2x Max 20x: +$400/mo (for parallel high-volume agents)
- Total with 1x Max: **~$340/month** all-in
- See the Agent Executor Authentication Options section for full comparison.

### Cost Optimization Tips

1. Use `min_instance_count: 0` for the dashboard if 24/7 uptime isn't needed (saves ~$25/mo)
2. BigQuery slots reservation for predictable query costs
3. Firestore cache to reduce read operations
4. Coordinator MUST use instance-based billing (`--no-cpu-throttling`) — it polls GitHub/Pub/Sub in background
5. Cloud Run Jobs only charge while running — 3hr timeout doesn't cost more if jobs finish early
6. Direct IAP on Cloud Run eliminates load balancer costs (~$18/mo saved vs LB approach)

---

## Security Considerations

### IAM Roles (Principle of Least Privilege)

| Service Account | Roles |
|-----------------|-------|
| `ailang-dashboard` | `roles/datastore.user`, `roles/bigquery.dataViewer`, `roles/pubsub.subscriber`, `roles/secretmanager.secretAccessor` |
| `ailang-coordinator` | `roles/datastore.user`, `roles/bigquery.dataEditor`, `roles/pubsub.editor`, `roles/run.invoker`, `roles/eventarc.eventReceiver`, `roles/secretmanager.secretAccessor` |
| `ailang-agent-executor` | `roles/datastore.user`, `roles/pubsub.publisher`, `roles/storage.objectCreator`, `roles/secretmanager.secretAccessor` |

### Network Security

- **VPC Connector**: All Cloud Run services use private networking
- **No public IPs**: Agent executors cannot be accessed from internet
- **Firewall rules**: Only allow internal traffic between services
- **Cloud Armor** (optional): WAF for dashboard if public-facing

### Secret Management

- All API keys in Secret Manager (never env vars in source)
- Secret rotation with versioned secrets
- Audit logging for secret access
- GitHub PAT with minimum scope (`repo` only)

### Data Protection

- Encryption at rest (Firestore and BigQuery default)
- Encryption in transit (TLS 1.3)
- Data residency: Single region deployment
- Backup: Firestore point-in-time recovery, BigQuery snapshots

---

## Deployment Artifacts

### Cloud Run Source Deploy (Dashboard + Coordinator)

**No Dockerfiles needed** for the dashboard and coordinator. Cloud Run builds from source
using [Google Cloud buildpacks](https://cloud.google.com/docs/buildpacks/go), which
auto-detect Go via `go.mod` and build the binary.

**How it works:**
1. `gcloud run deploy --source .` uploads source to Cloud Storage
2. Cloud Build triggers with Google Cloud buildpacks
3. Buildpacks detect `go.mod`, run `go build`, produce container image
4. Image stored in Artifact Registry (`cloud-run-source-deploy` repo)
5. Cloud Run deploys the new revision

**Key build env var:** `GOOGLE_BUILDABLE=./cmd/ailang` — tells buildpacks which
Go package to build (required for multi-package repos like AILANG).

**Important:** `go:embed` works with buildpacks — embedded assets (UI, stdlib, prompts)
are compiled into the binary at build time. Do NOT set `GOOGLE_CLEAR_SOURCE=true`.

#### Dashboard Server

```bash
# Pre-build React UI (go:embed picks up dist/ at build time)
cd ui && npm ci && npm run build && cd ..

# Deploy from source
gcloud run deploy ailang-dashboard \
  --source . \
  --region us-central1 \
  --set-build-env-vars GOOGLE_BUILDABLE=./cmd/ailang \
  --set-env-vars AILANG_STORAGE=gcp,AILANG_CLOUD_PROJECT=${AILANG_CLOUD_PROJECT} \
  --set-secrets ANTHROPIC_API_KEY=anthropic-api-key:latest,GITHUB_TOKEN=github-token:latest \
  --service-account ailang-dashboard@${PROJECT_ID}.iam.gserviceaccount.com \
  --allow-unauthenticated=false \
  --min-instances 1 \
  --max-instances 10 \
  --memory 512Mi \
  --cpu 1 \
  --port 8080 \
  --command ailang \
  --args "serve,--port,8080"
```

#### Coordinator Daemon

```bash
gcloud run deploy ailang-coordinator \
  --source . \
  --region us-central1 \
  --set-build-env-vars GOOGLE_BUILDABLE=./cmd/ailang \
  --set-env-vars AILANG_STORAGE=gcp,AILANG_CLOUD_PROJECT=${AILANG_CLOUD_PROJECT},COORDINATOR_MODE=cloud \
  --set-secrets ANTHROPIC_API_KEY=anthropic-api-key:latest,GITHUB_TOKEN=github-token:latest,GOOGLE_API_KEY=google-api-key:latest \
  --service-account ailang-coordinator@${PROJECT_ID}.iam.gserviceaccount.com \
  --no-allow-unauthenticated \
  --no-cpu-throttling \
  --min-instances 1 \
  --max-instances 3 \
  --memory 1Gi \
  --cpu 1 \
  --timeout 3600 \
  --command ailang \
  --args "coordinator,start,--foreground"
```

#### `.gcloudignore`

```
# VCS
.git/

# Frontend build dependencies (dist/ is kept — go:embed'd into binary)
ui/node_modules/

# Development/test files not needed at runtime
examples/
tests/
benchmarks/
design_docs/
docs/
*.test
*_test.go

# NOTE: Do NOT ignore these — they are go:embed'd into the binary:
# - stdlib/       (AILANG standard library)
# - prompts/      (AI teaching prompts)
# - ui/dist/      (React dashboard build output)
# - internal/     (Go source — buildpacks compile from source)
```

#### Agent Executor (`infra/docker/Dockerfile.agent-executor`)

**Note:** The agent executor still needs a Dockerfile because it requires CLI coding
agents (Claude Code, Gemini CLI, Codex CLI) that buildpacks can't auto-detect from `go.mod`.

**Official references:**
- Claude Code: [Official devcontainer Dockerfile](https://github.com/anthropics/claude-code/blob/main/.devcontainer/Dockerfile) (base: `node:20`, install via npm)
- Gemini CLI: [google-gemini/gemini-cli](https://github.com/google-gemini/gemini-cli) (npm package, no official Docker image)
- Codex CLI: [openai/codex-universal](https://github.com/openai/codex-universal) (sandbox base image at `ghcr.io/openai/codex-universal:latest`, CLI is a separate Rust binary)

| Agent CLI | Install Method | Binary |
|-----------|---------------|--------|
| Claude Code | `npm install -g @anthropic-ai/claude-code` | `claude` |
| Gemini CLI | `npm install -g @google/gemini-cli` | `gemini` |
| Codex CLI | `npm install -g @openai/codex` | `codex` |

Since all three are npm packages, we use `node:20` as the base (matching
Anthropic's [official devcontainer](https://code.claude.com/docs/en/devcontainer)).

```dockerfile
# Agent executor with Claude Code, Gemini CLI, Codex CLI, and ailang binary
# Runs as Cloud Run Job, triggered by Eventarc from Pub/Sub
#
# The entrypoint is `ailang coordinator execute-job` which:
# 1. Fetches task from Firestore
# 2. Clones repo and creates branch
# 3. Executes via internal/executor (ClaudeCodeProvider, GeminiCLIProvider, or CodexProvider)
# 4. Commits/pushes changes
# 5. Reports completion via Pub/Sub

# Stage 1: Build ailang binary
FROM golang:1.22-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X main.Version=${VERSION:-dev}" \
    -o /ailang ./cmd/ailang

# Stage 2: Runtime — node:20 base (matches Anthropic's official devcontainer)
# All three CLI agents are npm packages, so node:20 is the natural base.
FROM node:20

RUN apt-get update && apt-get install -y --no-install-recommends \
    git \
    ca-certificates \
    jq \
    openssh-client \
    && rm -rf /var/lib/apt/lists/*

# Install CLI coding agents (all npm packages)
RUN npm install -g @anthropic-ai/claude-code@latest
RUN npm install -g @google/gemini-cli@latest
RUN npm install -g @openai/codex@latest

# Copy ailang binary from builder
COPY --from=builder /ailang /usr/local/bin/ailang

# Copy standard library (needed for AILANG scripts)
COPY std/ /opt/ailang/std/

# Create working directory
WORKDIR /workspace

# Create non-root user
RUN useradd -m -u 1000 agent
USER agent

# Environment configuration
ENV AILANG_STORAGE=gcp
ENV AILANG_STDLIB_PATH=/opt/ailang/std
ENV HOME=/home/agent

# Entrypoint: ailang coordinator execute-job
# Receives AILANG_TASK_ID from Eventarc and executes the full task lifecycle
ENTRYPOINT ["/usr/local/bin/ailang", "coordinator", "execute-job"]
```

#### Agent Executor: Claude Code Authentication Options

The agent executor runs `claude -p --output-format stream-json` inside Cloud Run Jobs.
There are **two authentication models** for Claude Code in headless/cloud mode.

**Cost reality check:** A typical agent coding session uses 100K-500K tokens. At API rates,
100 tasks/day with Sonnet costs ~$3,000-15,000/month. A Max 20x subscription at $200/month
is **10-50x cheaper**. Max is the clear winner for cost — the only challenge is auth.

##### Option A: Claude Code Max Subscription (PREFERRED — $200/month flat)

| Aspect | Max 5x ($100/mo) | Max 20x ($200/mo) |
|--------|-------------------|-------------------|
| Usage | 5x Pro throughput | 20x Pro throughput |
| Rate limit | ~225 msgs/5hr rolling window | ~900 msgs/5hr rolling window |
| Weekly cap | Rolling 7-day reset | Rolling 7-day reset |
| Models | Sonnet + Opus | Sonnet + Opus |
| Overflow | Extra usage billed at API rates | Extra usage billed at API rates |

**Limits are throughput-based** (5-hour rolling windows + weekly caps), NOT hard session
counts. Usage is shared across claude.ai + Claude Code. See
[Max plan docs](https://support.claude.com/en/articles/11049741-what-is-the-max-plan).

**The auth challenge: No official M2M auth (yet)**

Claude Code Max uses OAuth (interactive browser login). There is no API key equivalent
for Max subscriptions. See [anthropics/claude-code#1454](https://github.com/anthropics/claude-code/issues/1454).

**Workaround: OAuth credential injection via Secret Manager**

```bash
# Step 1: Log in locally (one-time, interactive)
claude login

# Step 2: Store OAuth credentials in Secret Manager
gcloud secrets create claude-oauth-credentials \
  --data-file=$HOME/.claude/.credentials.json

# Step 3: Mount as a file volume in Cloud Run Job
gcloud run jobs update ailang-agent-executor \
  --set-secrets=/home/agent/.claude/.credentials.json=claude-oauth-credentials:latest

# Step 4: Claude Code picks up credentials from $HOME/.claude/.credentials.json
# The executor runs as USER agent with HOME=/home/agent
```

**Mitigations for known issues:**

| Issue | Mitigation |
|-------|------------|
| **Token expiry** | Enable "extra usage" on the Max plan — auto-falls back to API rates when OAuth token expires, so agents never hard-fail. Refresh token periodically via Cloud Scheduler. |
| **Concurrent instances** | Coordinator already serializes per agent ID — only one job per agent runs at a time. Different agents (design-doc-creator, sprint-executor) can run in parallel safely since each Cloud Run Job instance has its own filesystem. |
| **Token refresh** | Schedule a Cloud Scheduler job (e.g., weekly) that triggers a lightweight Cloud Run service to re-authenticate and update the Secret Manager version. Or manually refresh when rate-limited. |

**Multiple Max subscriptions for higher throughput:**

For workloads exceeding a single Max plan's throughput, use separate subscriptions per
agent role. Each subscription's OAuth credentials are stored as a separate secret:

```yaml
# Secret Manager secrets (one per Max subscription):
#   claude-oauth-design     → design-doc-creator agent
#   claude-oauth-executor   → sprint-executor agent
#   claude-oauth-shared     → sprint-planner + other low-volume agents
```

| Agent | Subscription | Monthly Cost |
|-------|-------------|-------------|
| `design-doc-creator` | Max account A | $200/mo |
| `sprint-executor` | Max account B | $200/mo |
| `sprint-planner` | Shared with A | (included) |
| **Total** | | **$400/mo** |

This gives 2x the throughput and full isolation between agent roles.

##### Per-User / Per-Workspace Credential Management

In multi-tenant deployments, different users or teams bring their own credentials
(Max subscriptions or API keys). The system must isolate credentials so that:
- User A's Max subscription is not consumed by User B's tasks
- Teams can independently manage their own billing
- Revoking one user's access doesn't affect others

**Credential storage in Secret Manager (per-user):**

```bash
# Naming convention: claude-creds-{workspace_id} or claude-creds-{user_email_hash}
gcloud secrets create claude-creds-ws-acme-team \
  --data-file=/tmp/acme-credentials.json

gcloud secrets create claude-creds-ws-research-team \
  --data-file=/tmp/research-credentials.json

# Grant the agent executor service account access to ALL credential secrets
gcloud secrets add-iam-policy-binding claude-creds-ws-acme-team \
  --member="serviceAccount:ailang-agent-executor@${PROJECT}.iam.gserviceaccount.com" \
  --role="roles/secretmanager.secretAccessor"
```

**Firestore schema for credential mapping:**

```
/workspaces/{workspace_id}/
  credential_type: "max" | "api_key"
  secret_name: "claude-creds-ws-acme-team"    # Secret Manager reference
  provider: "claude" | "gemini"
  max_plan: "5x" | "20x"                       # For Max subscriptions
  registered_by: "user@example.com"
  registered_at: timestamp
  last_refreshed: timestamp                     # OAuth token refresh tracking
  status: "active" | "expired" | "revoked"
```

**How `execute-job` selects credentials:**

```go
// In the Cloud Run Job entrypoint (execute-job command):
func selectCredentials(ctx context.Context, task *coordinator.Task) (string, error) {
    // 1. Look up workspace from task metadata
    workspaceID := task.Metadata["workspace_id"]

    // 2. Fetch credential config from Firestore
    credDoc, err := firestoreClient.Collection("workspaces").
        Doc(workspaceID).Get(ctx)
    if err != nil {
        return "", fmt.Errorf("no credentials for workspace %s: %w", workspaceID, err)
    }

    // 3. Fetch the actual secret from Secret Manager
    secretName := credDoc.Data()["secret_name"].(string)
    secret, err := secretClient.AccessSecretVersion(ctx,
        &secretmanagerpb.AccessSecretVersionRequest{
            Name: fmt.Sprintf("projects/%s/secrets/%s/versions/latest", projectID, secretName),
        })
    if err != nil {
        return "", fmt.Errorf("failed to access secret %s: %w", secretName, err)
    }

    // 4. Write credentials to expected location based on type
    credType := credDoc.Data()["credential_type"].(string)
    switch credType {
    case "max":
        // Write OAuth credentials file
        os.MkdirAll("/home/agent/.claude", 0700)
        os.WriteFile("/home/agent/.claude/.credentials.json", secret.Payload.Data, 0600)
    case "api_key":
        // Set environment variable
        os.Setenv("ANTHROPIC_API_KEY", string(secret.Payload.Data))
    }
    return credType, nil
}
```

**Dashboard UI: Credential registration flow:**

Users register their credentials through the Collaboration Hub:

1. **Max subscription**: User authenticates via `claude login` locally, uploads
   `.credentials.json` through the dashboard. Backend stores in Secret Manager
   and records the mapping in Firestore.

2. **API key**: User enters their Anthropic API key in the dashboard settings.
   Backend stores in Secret Manager (never in Firestore directly).

3. **Status monitoring**: Dashboard shows credential health per workspace —
   active, expired (needs refresh), or revoked.

```
┌──────────────────────────────────────────────────────┐
│  Settings > Workspace Credentials                     │
├──────────────────────────────────────────────────────┤
│                                                       │
│  Workspace: acme-team                                │
│  ┌──────────────────────────────────────────────┐    │
│  │ Provider: Claude                              │    │
│  │ Auth Type: Max 20x (OAuth)                    │    │
│  │ Status: ● Active                              │    │
│  │ Last Refreshed: 2026-02-18 14:30 UTC          │    │
│  │ [Refresh Credentials] [Revoke]                │    │
│  └──────────────────────────────────────────────┘    │
│                                                       │
│  [+ Add Credential]                                  │
│                                                       │
└──────────────────────────────────────────────────────┘
```

**Security considerations:**

| Concern | Mitigation |
|---------|------------|
| **Credential isolation** | Each workspace's secret has a unique Secret Manager entry; no cross-workspace access possible |
| **Least privilege** | Agent executor SA can only read secrets, not create or modify them |
| **Rotation** | Dashboard shows expiry warnings; Cloud Scheduler can auto-refresh per workspace |
| **Revocation** | Setting `status: "revoked"` in Firestore immediately blocks task execution for that workspace |
| **Audit trail** | All credential access logged via Cloud Audit Logs on Secret Manager |

##### Option B: API Key (Simple auth, expensive at scale)

```bash
# Set ANTHROPIC_API_KEY — Claude Code uses pay-per-token API
export ANTHROPIC_API_KEY="sk-ant-..."
claude -p "Fix the bug" --output-format stream-json
```

| Aspect | Detail |
|--------|--------|
| Auth | `ANTHROPIC_API_KEY` env var (injected via Secret Manager) |
| Billing | Pay-per-token (see cost comparison below) |
| Concurrency | Unlimited parallel Cloud Run Jobs |
| Reliability | Official, production-supported |
| Drawback | **Extremely expensive at scale** |

**Cost comparison (100 tasks/day, ~200K tokens avg per agent session):**

| Method | Monthly Cost | Notes |
|--------|-------------|-------|
| **Max 20x** | **$200 flat** | ~900 msgs/5hr window, overflow at API rates |
| **Max 5x** | **$100 flat** | ~225 msgs/5hr window, good for <50 tasks/day |
| **2x Max 20x** | **$400 flat** | For parallel high-volume agents |
| API Key (Haiku) | ~$150/mo | Cheapest API, but Haiku may be too weak for coding |
| API Key (Sonnet) | ~$6,000/mo | 100 tasks × 200K tokens × $0.003/1K × 30 days |
| API Key (Opus) | ~$30,000/mo | 100 tasks × 200K tokens × $0.015/1K × 30 days |

##### Recommended approach

| Phase | Auth | Why |
|-------|------|-----|
| **Phase 1** | Max 20x + OAuth injection + "extra usage" fallback | Cost-effective, works today |
| **Phase 1 scale-up** | Multiple Max subscriptions (1 per high-volume agent) | Scales throughput linearly |
| **Fallback** | API Key (`ANTHROPIC_API_KEY`) | If Max auth breaks, instant switch |
| **Future** | Max M2M auth ([#1454](https://github.com/anthropics/claude-code/issues/1454)) | Eliminates OAuth workaround |

**How the execution flow works:**

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    CLOUD RUN JOB EXECUTION FLOW                              │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  1. Eventarc triggers Cloud Run Job with AILANG_TASK_ID                     │
│                                                                              │
│  2. Container starts: ailang coordinator execute-job                         │
│     │                                                                        │
│     ▼                                                                        │
│  3. Fetch task from Firestore (internal/storage/firestore/coordinator.go)   │
│     │                                                                        │
│     ▼                                                                        │
│  4. Clone repository: git clone https://oauth2:$GITHUB_TOKEN@...            │
│     │                                                                        │
│     ▼                                                                        │
│  5. Create branch: git checkout -b coordinator/{task-id}                    │
│     │                                                                        │
│     ▼                                                                        │
│  6. Execute task via TaskExecutor.Execute()                                  │
│     │                                                                        │
│     ├──► ClaudeCodeProvider  → internal/executor/claude/claude.go            │
│     │    └──► exec.Command("claude", "-p", directive, ...)                   │
│     │                                                                        │
│     ├──► GeminiCLIProvider   → internal/executor/gemini/gemini.go            │
│     │    └──► exec.Command("gemini", "--output-format", "json", ...)         │
│     │                                                                        │
│     └──► CodexProvider       → internal/executor/codex/codex.go (planned)    │
│          └──► exec.Command("codex", ...)                                     │
│     │                                                                        │
│     ▼                                                                        │
│  7. Auto-commit changes: git add -A && git commit                           │
│     │                                                                        │
│     ▼                                                                        │
│  8. Push branch: git push origin coordinator/{task-id}                       │
│     │                                                                        │
│     ▼                                                                        │
│  9. Report completion via Pub/Sub (ailang-task-completions topic)           │
│                                                                              │
└──────────────────────────────────────────────────────────────────────────────┘
```

#### Cloud Execution via `ailang coordinator execute-job` (NEW COMMAND)

**IMPORTANT: The existing AILANG implementation uses the `internal/executor/` package directly from Go code, NOT a shell script. The coordinator daemon orchestrates task execution via providers (`ClaudeCodeProvider`, `GeminiCLIProvider`, `CodexProvider` (planned), `ScriptProvider`).**

For Cloud Run Jobs, we add a new CLI command that wraps the existing executor infrastructure:

```go
// cmd/ailang/coordinator_cloud.go (NEW)
//
// `ailang coordinator execute-job` is designed for Cloud Run Jobs.
// It fetches task details from Firestore and executes using the existing
// TaskExecutor → Provider → Executor chain, then reports completion via Pub/Sub.

func coordinatorExecuteJob(args []string) error {
    // 1. Get task ID from environment (injected by Eventarc)
    taskID := os.Getenv("AILANG_TASK_ID")
    if taskID == "" {
        return fmt.Errorf("AILANG_TASK_ID environment variable required")
    }

    // 2. Initialize Firestore backend (AILANG_STORAGE=gcp)
    ctx := context.Background()
    backends, err := storage.NewBackends(ctx)
    if err != nil {
        return fmt.Errorf("failed to initialize backends: %w", err)
    }

    // 3. Fetch task from Firestore
    task, err := backends.Coordinator.GetTask(ctx, taskID)
    if err != nil {
        return fmt.Errorf("failed to get task %s: %w", taskID, err)
    }

    // 4. Clone repository to /workspace
    repoURL := os.Getenv("AILANG_REPO_URL")
    if repoURL == "" {
        repoURL = "sunholo-data/ailang" // default
    }
    workspace := "/workspace/repo"
    if err := cloneRepository(repoURL, workspace, task.ID); err != nil {
        return fmt.Errorf("failed to clone repo: %w", err)
    }

    // 5. Load agent config
    agentConfig := getAgentConfig(task.AgentID)

    // 6. Create analyzed task (reuses existing coordinator logic)
    directive := coordinator.BuildDirectiveFromConfig(task, agentConfig)
    analyzed := &coordinator.AnalyzedTask{
        Task: &coordinator.Task{
            ID:        task.ID,
            Title:     task.Title,
            Content:   directive,
            Kind:      task.Kind,
            MessageID: task.MessageID,
            Iteration: task.Iteration,
            SessionID: task.SessionID,
        },
        Type: task.Type,
    }

    // 7. Create executor with configured providers
    // This reuses internal/executor/claude and internal/executor/gemini
    executor, err := coordinator.DefaultTaskExecutor()
    if err != nil {
        return fmt.Errorf("failed to create executor: %w", err)
    }

    // 8. Configure execution options
    opts := &coordinator.ExecuteOptions{
        Timeout:   3 * time.Hour,
        Workspace: workspace,
    }
    if agentConfig != nil && agentConfig.Invoke != nil {
        opts.InvokeConfig = agentConfig.Invoke
    }

    // 9. Execute task (uses ClaudeCodeProvider, GeminiCLIProvider, or CodexProvider)
    // The provider runs the configured CLI agent (claude, gemini, or codex)
    // See internal/executor/{claude,gemini,codex}/ for CLI invocations
    result, err := executor.Execute(ctx, analyzed, opts)
    if err != nil {
        return reportCompletion(ctx, task.ID, nil, err)
    }

    // 10. Commit and push changes
    if err := commitAndPush(workspace, task.ID, task.Title); err != nil {
        log.Printf("Warning: commit/push failed: %v", err)
    }

    // 11. Report completion via Pub/Sub
    return reportCompletion(ctx, task.ID, result, nil)
}

// cloneRepository clones the repo and creates the task branch
func cloneRepository(repoURL, workspace, taskID string) error {
    githubToken := os.Getenv("GITHUB_TOKEN")

    // Configure git
    exec.Command("git", "config", "--global", "user.name", "AILANG Coordinator").Run()
    exec.Command("git", "config", "--global", "user.email", "coordinator@ailang.dev").Run()

    // Clone
    cloneURL := fmt.Sprintf("https://oauth2:%s@github.com/%s.git", githubToken, repoURL)
    cmd := exec.Command("git", "clone", cloneURL, workspace)
    if err := cmd.Run(); err != nil {
        return err
    }

    // Create branch
    branchName := fmt.Sprintf("coordinator/%s", taskID)
    cmd = exec.Command("git", "checkout", "-b", branchName)
    cmd.Dir = workspace
    return cmd.Run()
}

// reportCompletion publishes completion status to Pub/Sub
func reportCompletion(ctx context.Context, taskID string, result *coordinator.ExecuteResult, execErr error) error {
    projectID := os.Getenv("AILANG_CLOUD_PROJECT")
    client, _ := pubsub.NewClient(ctx, projectID)
    defer client.Close()

    status := "completed"
    if execErr != nil || (result != nil && !result.Success) {
        status = "failed"
    }

    payload := map[string]interface{}{
        "task_id":    taskID,
        "status":     status,
        "session_id": "",
    }
    if result != nil {
        payload["session_id"] = result.SessionID
        payload["cost"] = result.Cost
        payload["tokens"] = result.TokensUsed
    }
    if execErr != nil {
        payload["error"] = execErr.Error()
    }

    data, _ := json.Marshal(payload)
    topic := client.Topic("ailang-task-completions")
    _, err := topic.Publish(ctx, &pubsub.Message{
        Data: data,
        Attributes: map[string]string{"task_id": taskID},
    }).Get(ctx)

    return err
}
```

**Key Points:**

1. **Reuses existing executor infrastructure** - No duplication of Claude/Gemini invocation logic
2. **Provider selection happens automatically** via `TaskExecutor.selectProvider()`
3. **The actual CLI invocation** is in `internal/executor/claude/claude.go`:
   ```go
   // From claude.go ExecuteStreaming():
   args := []string{
       "-p", task.Directive,
       "--output-format", "stream-json",
       "--permission-mode", e.permissionMode,
       "--model", e.getModel(task),
       "--session-id", sessionID,
   }
   cmd := exec.CommandContext(ctx, e.claudePath, args...)
   ```
4. **Environment setup** uses `executor.BuildEnvironment()` for trace context propagation
5. **Session continuity** - Uses `--resume <sessionId>` for Claude when `Iteration > 1`

---

### Cloud Build CI/CD (`infra/cloudbuild.yaml`)

**Simplified with source deploy**: Dashboard and coordinator use `gcloud run deploy --source .`
which handles build+push+deploy in one step. Only the agent executor needs a Dockerfile.

```yaml
# Cloud Build configuration for AILANG services
# Triggers on push to main/dev branches
# Dashboard + coordinator: source deploy (buildpacks, no Dockerfile)
# Agent executor: Docker build (needs Claude CLI + Node.js)

substitutions:
  _REGION: us-central1
  _ARTIFACT_REGISTRY: us-central1-docker.pkg.dev/${PROJECT_ID}/ailang

options:
  logging: CLOUD_LOGGING_ONLY
  machineType: E2_HIGHCPU_8

steps:
  # ═══════════════════════════════════════════════════════════════════════
  # STEP 1: Build React UI (needed for go:embed before source deploy)
  # ═══════════════════════════════════════════════════════════════════════
  - id: build-ui
    name: node:20-alpine
    entrypoint: sh
    args:
      - -c
      - cd ui && npm ci && npm run build

  # ═══════════════════════════════════════════════════════════════════════
  # STEP 2: Deploy Dashboard from Source (buildpacks, no Dockerfile)
  # ═══════════════════════════════════════════════════════════════════════
  - id: deploy-dashboard
    name: gcr.io/google.com/cloudsdktool/cloud-sdk
    entrypoint: gcloud
    args:
      - run
      - deploy
      - ailang-dashboard
      - --source=.
      - --region=${_REGION}
      - --set-build-env-vars=GOOGLE_BUILDABLE=./cmd/ailang
      - --allow-unauthenticated=false
      - --service-account=ailang-dashboard@${PROJECT_ID}.iam.gserviceaccount.com
      - --set-env-vars=AILANG_STORAGE=gcp,AILANG_CLOUD_PROJECT=${PROJECT_ID}
      - --set-secrets=ANTHROPIC_API_KEY=anthropic-api-key:latest,GITHUB_TOKEN=github-token:latest
      - --min-instances=1
      - --max-instances=10
      - --memory=512Mi
      - --cpu=1
      - --port=8080
      - --command=ailang
      - --args=serve,--port,8080
    waitFor: [build-ui]

  # ═══════════════════════════════════════════════════════════════════════
  # STEP 3: Deploy Coordinator from Source (buildpacks, no Dockerfile)
  # ═══════════════════════════════════════════════════════════════════════
  - id: deploy-coordinator
    name: gcr.io/google.com/cloudsdktool/cloud-sdk
    entrypoint: gcloud
    args:
      - run
      - deploy
      - ailang-coordinator
      - --source=.
      - --region=${_REGION}
      - --set-build-env-vars=GOOGLE_BUILDABLE=./cmd/ailang
      - --no-allow-unauthenticated
      - --no-cpu-throttling
      - --service-account=ailang-coordinator@${PROJECT_ID}.iam.gserviceaccount.com
      - --set-env-vars=AILANG_STORAGE=gcp,AILANG_CLOUD_PROJECT=${PROJECT_ID},COORDINATOR_MODE=cloud
      - --set-secrets=ANTHROPIC_API_KEY=anthropic-api-key:latest,GITHUB_TOKEN=github-token:latest,GOOGLE_API_KEY=google-api-key:latest
      - --min-instances=1
      - --max-instances=3
      - --memory=1Gi
      - --cpu=1
      - --timeout=3600
      - --command=ailang
      - --args=coordinator,start,--foreground
    waitFor: [build-ui]

  # ═══════════════════════════════════════════════════════════════════════
  # STEP 4: Build Agent Executor Image (Dockerfile — needs CLIs)
  # ═══════════════════════════════════════════════════════════════════════
  - id: build-agent-executor
    name: gcr.io/cloud-builders/docker
    args:
      - build
      - --file=infra/docker/Dockerfile.agent-executor
      - --tag=${_ARTIFACT_REGISTRY}/agent-executor:${SHORT_SHA}
      - --tag=${_ARTIFACT_REGISTRY}/agent-executor:latest
      - --cache-from=${_ARTIFACT_REGISTRY}/agent-executor:latest
      - --build-arg=VERSION=${TAG_NAME:-${SHORT_SHA}}
      - .

  # ═══════════════════════════════════════════════════════════════════════
  # STEP 5: Push Agent Executor + Deploy as Cloud Run Job
  # ═══════════════════════════════════════════════════════════════════════
  - id: push-agent-executor
    name: gcr.io/cloud-builders/docker
    args: [push, --all-tags, '${_ARTIFACT_REGISTRY}/agent-executor']
    waitFor: [build-agent-executor]

  - id: deploy-agent-executor
    name: gcr.io/google.com/cloudsdktool/cloud-sdk
    entrypoint: gcloud
    args:
      - run
      - jobs
      - update
      - ailang-agent-executor
      - --image=${_ARTIFACT_REGISTRY}/agent-executor:${SHORT_SHA}
      - --region=${_REGION}
      - --service-account=ailang-agent-executor@${PROJECT_ID}.iam.gserviceaccount.com
      - --set-env-vars=AILANG_STORAGE=gcp,AILANG_CLOUD_PROJECT=${PROJECT_ID}
      - --set-secrets=ANTHROPIC_API_KEY=anthropic-api-key:latest,GITHUB_TOKEN=github-token:latest,GOOGLE_API_KEY=google-api-key:latest
      - --memory=4Gi
      - --cpu=2
      - --task-timeout=10800
      - --max-retries=1
    waitFor: [push-agent-executor]

images:
  - ${_ARTIFACT_REGISTRY}/agent-executor:${SHORT_SHA}
  - ${_ARTIFACT_REGISTRY}/agent-executor:latest

timeout: 1800s  # 30 minutes
```

**Key simplification**: Dashboard and coordinator go from 7 steps (build 3 images + push 3 + deploy 3)
to 4 steps (build UI + source deploy 2 + Docker build/push/deploy agent). Source deploy handles
build + push + deploy in a single `gcloud run deploy --source .` command.

---

### Pub/Sub Message Broker

The broker bridges `ailang messages` CLI commands to GCP Pub/Sub for cloud-native messaging.

#### Broker Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                      PUB/SUB MESSAGE BROKER ARCHITECTURE                     │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  LOCAL CLI                          CLOUD                                   │
│  ┌────────────────┐                 ┌────────────────────────────────────┐  │
│  │ ailang messages│                 │              Pub/Sub                │  │
│  │ send inbox ... │                 │                                    │  │
│  └───────┬────────┘                 │  ┌────────────────────────────┐   │  │
│          │                          │  │ ailang-inbox-{agent}       │   │  │
│          │ AILANG_STORAGE=gcp       │  │  ↓                         │   │  │
│          ▼                          │  │ Subscription: coordinator  │   │  │
│  ┌────────────────┐                 │  └────────────────────────────┘   │  │
│  │ MessageBroker  │ ──Publish───►   │                                    │  │
│  │ (pubsub impl)  │                 │  ┌────────────────────────────┐   │  │
│  └────────────────┘                 │  │ ailang-tasks               │   │  │
│                                     │  │  ↓                         │   │  │
│  CLOUD RUN                          │  │ Eventarc → Cloud Run Job   │   │  │
│  ┌────────────────┐                 │  └────────────────────────────┘   │  │
│  │ Coordinator    │ ◄─Subscribe──   │                                    │  │
│  │ Daemon         │                 │  ┌────────────────────────────┐   │  │
│  └───────┬────────┘                 │  │ ailang-events              │   │  │
│          │                          │  │  ↓                         │   │  │
│          │ Poll inbox-{agent}       │  │ Dashboard WebSocket push   │   │  │
│          ▼                          │  └────────────────────────────┘   │  │
│  ┌────────────────┐                 │                                    │  │
│  │ Task Analyzer  │ ──Publish───►   │  ┌────────────────────────────┐   │  │
│  │ & Dispatcher   │                 │  │ ailang-task-completions    │   │  │
│  └────────────────┘                 │  │  ↓                         │   │  │
│                                     │  │ Coordinator (job results)  │   │  │
│                                     │  └────────────────────────────┘   │  │
│                                     └────────────────────────────────────┘  │
│                                                                              │
└──────────────────────────────────────────────────────────────────────────────┘
```

#### Message Broker Interface (`internal/messaging/broker.go`)

```go
// MessageBroker provides an abstraction over message transport
// Local mode uses SQLite; GCP mode uses Pub/Sub
type MessageBroker interface {
    // Publish sends a message to a named inbox
    Publish(ctx context.Context, inbox string, msg *InboxMessage) error

    // Subscribe creates a subscription to an inbox
    // Returns a channel that receives messages
    Subscribe(ctx context.Context, inbox string) (<-chan *InboxMessage, error)

    // Ack acknowledges a message was processed
    Ack(ctx context.Context, inbox string, msgID string) error

    // Nack negative-acknowledges a message (will be redelivered)
    Nack(ctx context.Context, inbox string, msgID string) error

    // Close cleans up resources
    Close() error
}

// NewMessageBroker creates a broker based on AILANG_STORAGE env var
func NewMessageBroker(ctx context.Context) (MessageBroker, error) {
    mode := os.Getenv("AILANG_STORAGE")
    switch mode {
    case "local", "":
        return NewSQLiteBroker()
    case "gcp":
        return NewPubSubBroker(ctx)
    default:
        return nil, fmt.Errorf("unknown storage mode: %s", mode)
    }
}
```

#### Pub/Sub Broker Implementation (`internal/messaging/pubsub_broker.go`)

```go
package messaging

import (
    "context"
    "encoding/json"
    "fmt"
    "os"

    "cloud.google.com/go/pubsub"
)

type PubSubBroker struct {
    client    *pubsub.Client
    projectID string
    topics    map[string]*pubsub.Topic
    subs      map[string]*pubsub.Subscription
    // pendingAcks tracks ack/nack callbacks for messages awaiting processing.
    // Key is message ID, value is the Pub/Sub ack function.
    pendingAcks map[string]func()
    pendingMu   sync.Mutex
}

func NewPubSubBroker(ctx context.Context) (*PubSubBroker, error) {
    projectID := os.Getenv("AILANG_CLOUD_PROJECT")
    if projectID == "" {
        return nil, fmt.Errorf("AILANG_CLOUD_PROJECT not set")
    }

    client, err := pubsub.NewClient(ctx, projectID)
    if err != nil {
        return nil, fmt.Errorf("failed to create pubsub client: %w", err)
    }

    return &PubSubBroker{
        client:      client,
        projectID:   projectID,
        topics:      make(map[string]*pubsub.Topic),
        subs:        make(map[string]*pubsub.Subscription),
        pendingAcks: make(map[string]func()),
    }, nil
}

func (b *PubSubBroker) topicName(inbox string) string {
    return fmt.Sprintf("ailang-inbox-%s", inbox)
}

func (b *PubSubBroker) Publish(ctx context.Context, inbox string, msg *InboxMessage) error {
    topicName := b.topicName(inbox)

    // Get or create topic reference
    topic, ok := b.topics[topicName]
    if !ok {
        topic = b.client.Topic(topicName)
        b.topics[topicName] = topic
    }

    // Serialize message
    data, err := json.Marshal(msg)
    if err != nil {
        return fmt.Errorf("failed to marshal message: %w", err)
    }

    // Publish with attributes for filtering
    result := topic.Publish(ctx, &pubsub.Message{
        Data: data,
        Attributes: map[string]string{
            "message_id": msg.MessageID,
            "from_agent": msg.FromAgent,
            "to_inbox":   msg.ToInbox,
            "category":   msg.Category,
        },
    })

    // Wait for publish to complete
    _, err = result.Get(ctx)
    if err != nil {
        return fmt.Errorf("failed to publish message: %w", err)
    }

    return nil
}

func (b *PubSubBroker) Subscribe(ctx context.Context, inbox string) (<-chan *InboxMessage, error) {
    topicName := b.topicName(inbox)
    subName := fmt.Sprintf("%s-coordinator", topicName)

    sub := b.client.Subscription(subName)
    b.subs[inbox] = sub

    msgChan := make(chan *InboxMessage, 100)

    go func() {
        defer close(msgChan)

        err := sub.Receive(ctx, func(ctx context.Context, m *pubsub.Message) {
            var msg InboxMessage
            if err := json.Unmarshal(m.Data, &msg); err != nil {
                m.Nack()
                return
            }

            // Store Pub/Sub message ID for later ack/nack
            msg.pubsubMsgID = m.ID

            // IMPORTANT: Store the ack callback so the coordinator can ack
            // after successful processing. Do NOT ack here — if the coordinator
            // crashes before processing, the message must be redelivered.
            b.pendingMu.Lock()
            b.pendingAcks[m.ID] = m.Ack
            b.pendingMu.Unlock()

            select {
            case msgChan <- &msg:
                // Message sent to coordinator for processing.
                // Ack will be called by coordinator via broker.Ack().
            case <-ctx.Done():
                m.Nack()
                return
            }
        })

        if err != nil && err != context.Canceled {
            log.Printf("PubSub Subscribe error for %s: %v", inbox, err)
        }
    }()

    return msgChan, nil
}

func (b *PubSubBroker) Ack(ctx context.Context, inbox string, msgID string) error {
    b.pendingMu.Lock()
    ackFn, ok := b.pendingAcks[msgID]
    if ok {
        delete(b.pendingAcks, msgID)
    }
    b.pendingMu.Unlock()

    if !ok {
        return fmt.Errorf("no pending ack for message %s", msgID)
    }
    ackFn()
    return nil
}

func (b *PubSubBroker) Nack(ctx context.Context, inbox string, msgID string) error {
    b.pendingMu.Lock()
    _, ok := b.pendingAcks[msgID]
    if ok {
        delete(b.pendingAcks, msgID)
    }
    b.pendingMu.Unlock()

    // Nack is implicit — if we don't ack before the ack deadline expires,
    // Pub/Sub will redeliver the message automatically.
    // For explicit nack, we simply remove from pending and let deadline expire.
    return nil
}

func (b *PubSubBroker) Close() error {
    // Stop all topic publishing
    for _, topic := range b.topics {
        topic.Stop()
    }
    return b.client.Close()
}
```

---

### Service Accounts & IAM

#### Service Account Configuration (`infra/terraform/iam.tf`)

```hcl
# ═══════════════════════════════════════════════════════════════════════════
# SERVICE ACCOUNTS
# ═══════════════════════════════════════════════════════════════════════════

# Dashboard Service Account
resource "google_service_account" "dashboard" {
  account_id   = "ailang-dashboard"
  display_name = "AILANG Dashboard Service Account"
  description  = "Service account for the AILANG dashboard web server"
}

# Coordinator Service Account
resource "google_service_account" "coordinator" {
  account_id   = "ailang-coordinator"
  display_name = "AILANG Coordinator Service Account"
  description  = "Service account for the AILANG coordinator daemon"
}

# Agent Executor Service Account
resource "google_service_account" "agent_executor" {
  account_id   = "ailang-agent-executor"
  display_name = "AILANG Agent Executor Service Account"
  description  = "Service account for Cloud Run Jobs executing agents"
}

# Cloud Build Service Account (for CI/CD)
resource "google_service_account" "cloud_build" {
  account_id   = "ailang-cloud-build"
  display_name = "AILANG Cloud Build Service Account"
  description  = "Service account for Cloud Build deployments"
}

# ═══════════════════════════════════════════════════════════════════════════
# IAM BINDINGS - DASHBOARD
# ═══════════════════════════════════════════════════════════════════════════

# Firestore access (read/write messages, tasks, approvals)
resource "google_project_iam_member" "dashboard_firestore" {
  project = var.project_id
  role    = "roles/datastore.user"
  member  = "serviceAccount:${google_service_account.dashboard.email}"
}

# BigQuery access (read observatory data)
resource "google_project_iam_member" "dashboard_bigquery" {
  project = var.project_id
  role    = "roles/bigquery.dataViewer"
  member  = "serviceAccount:${google_service_account.dashboard.email}"
}

# Pub/Sub subscriber (receive events for WebSocket)
resource "google_project_iam_member" "dashboard_pubsub" {
  project = var.project_id
  role    = "roles/pubsub.subscriber"
  member  = "serviceAccount:${google_service_account.dashboard.email}"
}

# Secret Manager access
resource "google_secret_manager_secret_iam_member" "dashboard_anthropic" {
  secret_id = google_secret_manager_secret.anthropic_key.secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.dashboard.email}"
}

# ═══════════════════════════════════════════════════════════════════════════
# IAM BINDINGS - COORDINATOR
# ═══════════════════════════════════════════════════════════════════════════

# Firestore access (full CRUD for tasks, messages)
resource "google_project_iam_member" "coordinator_firestore" {
  project = var.project_id
  role    = "roles/datastore.user"
  member  = "serviceAccount:${google_service_account.coordinator.email}"
}

# BigQuery write access (insert spans)
resource "google_project_iam_member" "coordinator_bigquery" {
  project = var.project_id
  role    = "roles/bigquery.dataEditor"
  member  = "serviceAccount:${google_service_account.coordinator.email}"
}

# Pub/Sub editor (publish tasks, subscribe to inboxes)
resource "google_project_iam_member" "coordinator_pubsub" {
  project = var.project_id
  role    = "roles/pubsub.editor"
  member  = "serviceAccount:${google_service_account.coordinator.email}"
}

# Cloud Run Jobs invoker (trigger agent jobs)
resource "google_project_iam_member" "coordinator_run_invoker" {
  project = var.project_id
  role    = "roles/run.invoker"
  member  = "serviceAccount:${google_service_account.coordinator.email}"
}

# Eventarc event receiver (required for Pub/Sub → Cloud Run Job triggers)
resource "google_project_iam_member" "coordinator_eventarc" {
  project = var.project_id
  role    = "roles/eventarc.eventReceiver"
  member  = "serviceAccount:${google_service_account.coordinator.email}"
}

# Secret Manager access
resource "google_secret_manager_secret_iam_member" "coordinator_anthropic" {
  secret_id = google_secret_manager_secret.anthropic_key.secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.coordinator.email}"
}

resource "google_secret_manager_secret_iam_member" "coordinator_github" {
  secret_id = google_secret_manager_secret.github_token.secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.coordinator.email}"
}

resource "google_secret_manager_secret_iam_member" "coordinator_google_api" {
  secret_id = google_secret_manager_secret.google_api_key.secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.coordinator.email}"
}

# ═══════════════════════════════════════════════════════════════════════════
# IAM BINDINGS - AGENT EXECUTOR
# ═══════════════════════════════════════════════════════════════════════════

# Firestore access (read task details, update task status)
resource "google_project_iam_member" "agent_firestore" {
  project = var.project_id
  role    = "roles/datastore.user"
  member  = "serviceAccount:${google_service_account.agent_executor.email}"
}

# Pub/Sub publisher only (report completions)
resource "google_project_iam_member" "agent_pubsub" {
  project = var.project_id
  role    = "roles/pubsub.publisher"
  member  = "serviceAccount:${google_service_account.agent_executor.email}"
}

# Cloud Storage writer (upload artifacts)
resource "google_storage_bucket_iam_member" "agent_storage" {
  bucket = google_storage_bucket.artifacts.name
  role   = "roles/storage.objectCreator"
  member = "serviceAccount:${google_service_account.agent_executor.email}"
}

# Secret Manager access (API keys)
resource "google_secret_manager_secret_iam_member" "agent_anthropic" {
  secret_id = google_secret_manager_secret.anthropic_key.secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.agent_executor.email}"
}

resource "google_secret_manager_secret_iam_member" "agent_github" {
  secret_id = google_secret_manager_secret.github_token.secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.agent_executor.email}"
}

resource "google_secret_manager_secret_iam_member" "agent_google_api" {
  secret_id = google_secret_manager_secret.google_api_key.secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.agent_executor.email}"
}

# ═══════════════════════════════════════════════════════════════════════════
# IAM BINDINGS - CLOUD BUILD
# ═══════════════════════════════════════════════════════════════════════════

# Cloud Run admin (deploy services and jobs)
resource "google_project_iam_member" "build_run_admin" {
  project = var.project_id
  role    = "roles/run.admin"
  member  = "serviceAccount:${google_service_account.cloud_build.email}"
}

# Artifact Registry writer (push images)
resource "google_project_iam_member" "build_artifact" {
  project = var.project_id
  role    = "roles/artifactregistry.writer"
  member  = "serviceAccount:${google_service_account.cloud_build.email}"
}

# Service account user (deploy with other SAs)
resource "google_project_iam_member" "build_sa_user" {
  project = var.project_id
  role    = "roles/iam.serviceAccountUser"
  member  = "serviceAccount:${google_service_account.cloud_build.email}"
}
```

---

### Dashboard IAM Authentication

Using **Identity-Aware Proxy (IAP)** for Google Cloud identity-based access control.

**Approach: Direct IAP on Cloud Run (GA Feb 2026)**

As of Feb 2026, Cloud Run supports [IAP directly on the service](https://cloud.google.com/run/docs/securing/identity-aware-proxy-cloud-run)
without needing a load balancer. This protects the `run.app` URL directly, eliminates LB costs
(~$18/mo), and dramatically simplifies the Terraform.

**Known limitations of direct IAP on Cloud Run:**
- The project must be within a Google Cloud organization
- Identities must be from within the same organization
- Some integrations (e.g., Pub/Sub push) might not authenticate correctly if IAP is enabled
- **IMPORTANT**: Do NOT enable IAP on the coordinator service — it subscribes to Pub/Sub

#### Enable IAP via gcloud

```bash
# Enable IAP on the dashboard service (not the coordinator!)
gcloud beta run services update ailang-dashboard \
  --region us-central1 \
  --iap

# Grant access to specific users/groups
gcloud beta iap web add-iam-policy-binding \
  --member=user:mark@sunholo.com \
  --role=roles/iap.httpsResourceAccessor \
  --region=us-central1 \
  --resource-type=cloud-run \
  --service=ailang-dashboard

# Grant access to a group
gcloud beta iap web add-iam-policy-binding \
  --member=group:ailang-team@sunholo.com \
  --role=roles/iap.httpsResourceAccessor \
  --region=us-central1 \
  --resource-type=cloud-run \
  --service=ailang-dashboard
```

#### IAP Configuration (`infra/terraform/iap.tf`)

```hcl
# ═══════════════════════════════════════════════════════════════════════════
# IDENTITY-AWARE PROXY (IAP) FOR DASHBOARD
# Direct IAP on Cloud Run — no load balancer needed (GA Feb 2026)
# See: https://cloud.google.com/run/docs/securing/identity-aware-proxy-cloud-run
# ═══════════════════════════════════════════════════════════════════════════

# Enable IAP API
resource "google_project_service" "iap" {
  service            = "iap.googleapis.com"
  disable_on_destroy = false
}

# IAP is enabled directly on the Cloud Run service via the
# google_cloud_run_v2_service resource's iap block (or gcloud CLI).
# No load balancer, NEG, backend service, or forwarding rule needed.

# Grant dashboard access to users/groups
resource "google_iap_web_iam_binding" "dashboard_access" {
  project = var.project_id
  role    = "roles/iap.httpsResourceAccessor"
  members = var.dashboard_users
}

# ═══════════════════════════════════════════════════════════════════════════
# VARIABLES FOR IAP
# ═══════════════════════════════════════════════════════════════════════════

variable "support_email" {
  description = "Support email for OAuth consent screen"
  type        = string
  default     = "support@sunholo.com"
}

variable "dashboard_users" {
  description = "List of users/groups allowed to access dashboard"
  type        = list(string)
  default     = []
  # Example: ["user:mark@sunholo.com", "group:ailang-team@sunholo.com"]
}
```

**Optional: Load balancer for custom domain**

If you need a custom domain (e.g., `ailang.sunholo.com`) instead of the default `run.app` URL,
you can add a load balancer later. IAP will protect both the `run.app` URL and the LB endpoint.
See [Enabling IAP for Cloud Run via LB](https://cloud.google.com/iap/docs/enabling-cloud-run)
for the full load balancer setup.

#### Server-Side IAP Validation (`internal/server/middleware_iap.go`)

```go
package server

import (
    "context"
    "fmt"
    "net/http"
    "os"
    "strings"

    "google.golang.org/api/idtoken"
)

// IAPMiddleware validates IAP JWT tokens for authenticated requests
func IAPMiddleware(audience string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // Skip IAP validation in local mode
            if os.Getenv("AILANG_STORAGE") != "gcp" {
                next.ServeHTTP(w, r)
                return
            }

            // Skip health checks
            if r.URL.Path == "/health" {
                next.ServeHTTP(w, r)
                return
            }

            // Get IAP JWT from header
            iapJWT := r.Header.Get("X-Goog-IAP-JWT-Assertion")
            if iapJWT == "" {
                http.Error(w, "Missing IAP JWT", http.StatusUnauthorized)
                return
            }

            // Validate token
            payload, err := idtoken.Validate(r.Context(), iapJWT, audience)
            if err != nil {
                http.Error(w, fmt.Sprintf("Invalid IAP JWT: %v", err), http.StatusUnauthorized)
                return
            }

            // Extract user email
            email, ok := payload.Claims["email"].(string)
            if !ok {
                http.Error(w, "Missing email in IAP JWT", http.StatusUnauthorized)
                return
            }

            // Add user info to context
            ctx := context.WithValue(r.Context(), "user_email", email)
            ctx = context.WithValue(ctx, "iap_payload", payload)

            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}

// GetUserEmail extracts the authenticated user's email from context
func GetUserEmail(ctx context.Context) string {
    email, _ := ctx.Value("user_email").(string)
    return email
}

// RequireRole middleware checks if user has required role
func RequireRole(role string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            email := GetUserEmail(r.Context())

            // Check role from Firestore or config
            // For now, allow all authenticated users
            if email == "" {
                http.Error(w, "Authentication required", http.StatusUnauthorized)
                return
            }

            next.ServeHTTP(w, r)
        })
    }
}
```

---

## Axiom Compliance

| Axiom | Score | Notes |
|-------|-------|-------|
| A1: Determinism | +1 | Storage backends are deterministic; same input → same output |
| A2: Replayability | +1 | All task events persisted in Firestore for replay |
| A3: Effect Legibility | +1 | Cloud operations are explicit (env var switches mode) |
| A4: Explicit Authority | +1 | IAM roles are explicit per service; no ambient access |
| A5: Bounded Verification | 0 | N/A (infrastructure layer) |
| A6: Safe Concurrency | 0 | N/A (Firestore handles concurrency) |
| A7: Machines First | +1 | API-first design; Terraform for declarative infrastructure |
| A8: Minimal Syntax | 0 | N/A (infrastructure layer) |
| A9: Cost Visibility | +1 | Per-task cost tracking; budget controls in cloud |
| A10: Composability | +1 | Backend selection composes with existing code |
| A11: Structured Failure | +1 | Pub/Sub dead letters; explicit error handling |
| A12: System Boundary | +1 | Clear boundary: local dev ↔ cloud prod via env var |
| **Net Score** | **+8** | Strongly aligned |

---

## Success Criteria

- [ ] `AILANG_STORAGE=gcp ailang serve` starts dashboard with Firestore backend
- [ ] `AILANG_STORAGE=gcp ailang coordinator start` connects to Pub/Sub
- [ ] Agent tasks execute in Cloud Run Jobs, push branches to GitHub
- [ ] Dashboard WebSocket receives real-time events via Pub/Sub
- [ ] `make docker-compose-up` runs full stack locally with emulators
- [ ] `terraform apply` provisions all GCP infrastructure in <10 minutes
- [ ] Cost stays within $150/month for 100 tasks/day workload
- [ ] All existing tests pass with both local and GCP backends

---

## Open Questions

1. ~~**Auth**: Should we add Google Cloud IAM auth to dashboard before cloud deploy?~~ **RESOLVED**: Using IAP directly on Cloud Run (GA Feb 2026). No load balancer needed.
2. **Multi-region**: Is single-region (us-central1) sufficient for initial deployment?
3. **Monitoring**: Should we integrate Cloud Monitoring/Alerting from day 1?
4. **CI/CD**: Should we add Cloud Build triggers for automated deployment?
5. **Claude Code Max M2M auth**: When [#1454](https://github.com/anthropics/claude-code/issues/1454) ships, the OAuth credential injection workaround can be replaced with proper M2M auth. Monitor this issue.
6. **Agent SDK alternative**: Should cloud agents use the [Anthropic Agent SDK](https://platform.claude.com/docs/en/agent-sdk/overview) (Python/TypeScript) instead of Claude Code CLI? This would remove the Node.js dependency from the Dockerfile but requires rewriting the executor.

---

## Related Documents

- [M-GENERIC-PIPELINE](m-generic-pipeline.md) - **Prerequisite**: Config-driven agent pipelines (removes hardcoded TaskStage)
- [M-CLOUD-STORAGE](../v0_7_0/m-cloud-storage.md) - Original cloud storage design (planned)
- [M-OTEL-DASHBOARD](../../implemented/v0_6_3/m-otel-dashboard.md) - Observatory foundation
- [M-COORD-STABLE](../../implemented/v0_6_2/m-coord-stable.md) - Coordinator architecture
- [M-CONTROL-PLANE-V4](../../implemented/v0_6_4/m-control-plane-v4.md) - Dashboard design
