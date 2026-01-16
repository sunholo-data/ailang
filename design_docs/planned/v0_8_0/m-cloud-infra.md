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
| `ailang-agent-claude` | Claude Code CLI execution | Eventarc (Pub/Sub) | 2 vCPU, 4GB, 30min timeout |
| `ailang-agent-gemini` | Gemini CLI execution | Eventarc (Pub/Sub) | 2 vCPU, 4GB, 30min timeout |
| `ailang-agent-script` | Deterministic script execution | Eventarc (Pub/Sub) | 1 vCPU, 2GB, 2hr timeout |

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

### Phase 2: Docker Containers (Week 1-2)

**M5: Dockerfiles** (~200 LOC)
- [ ] `infra/docker/Dockerfile.coordinator` - Go binary + git
- [ ] `infra/docker/Dockerfile.dashboard` - Go binary + embedded React UI
- [ ] `infra/docker/Dockerfile.agent-executor` - Go + Claude CLI + Gemini CLI + git
- [ ] Multi-stage builds for minimal image size

**M6: Docker Compose** (~100 LOC)
- [ ] `infra/docker/docker-compose.yml` with emulators
- [ ] Firestore emulator container
- [ ] Pub/Sub emulator container
- [ ] Local development workflow

**M7: Cloud Execution Command** (~200 LOC)
- [ ] `cmd/ailang/coordinator_cloud.go` - New `execute-job` subcommand
- [ ] Fetch task from Firestore via storage backends
- [ ] Clone repo and create branch (reuse WorktreeManager patterns)
- [ ] Execute via existing TaskExecutor → Provider → Executor chain
- [ ] Commit/push changes (reuse existing git helpers)
- [ ] Report completion via Pub/Sub

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
| `infra/docker/Dockerfile.coordinator` | ~40 | Coordinator container |
| `infra/docker/Dockerfile.dashboard` | ~50 | Dashboard container |
| `infra/docker/Dockerfile.agent-executor` | ~60 | Agent job container |
| `infra/docker/docker-compose.yml` | ~100 | Local dev stack |
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
| `Makefile` | Add docker/cloud targets |

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
// - GOOGLE_CLOUD_PROJECT
```

### Coordinator Providers (`internal/coordinator/provider_*.go`)

Providers wrap executors with coordinator-specific logic:

```go
// ClaudeCodeProvider - Uses internal/executor/claude
// GeminiCLIProvider - Uses internal/executor/gemini
// ScriptProvider - Direct shell execution

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
| Cloud Run (Dashboard) | 1 instance, 24/7 | ~$25 |
| Cloud Run (Coordinator) | 1 instance, 24/7 | ~$25 |
| Cloud Run Jobs | 100 jobs/day, 10min avg | ~$15 |
| Firestore | 100K reads, 50K writes/day | ~$10 |
| BigQuery | 50GB storage, 1TB queries | ~$15 |
| Pub/Sub | 1M messages/month | ~$5 |
| Cloud Storage | 10GB artifacts | ~$1 |
| Secret Manager | 5 secrets, 10K access | ~$1 |
| Networking | Internal only | ~$5 |
| **Total** | | **~$105/month** |

### Cost Optimization Tips

1. Use `min_instance_count: 0` for non-critical services (saves ~$50/mo)
2. BigQuery slots reservation for predictable query costs
3. Firestore cache to reduce read operations
4. Cloud Run CPU allocation: "always" for coordinator (faster cold starts)

---

## Security Considerations

### IAM Roles (Principle of Least Privilege)

| Service Account | Roles |
|-----------------|-------|
| `ailang-dashboard` | `roles/datastore.user`, `roles/secretmanager.secretAccessor` |
| `ailang-coordinator` | `roles/datastore.user`, `roles/pubsub.editor`, `roles/secretmanager.secretAccessor` |
| `ailang-agent-executor` | `roles/secretmanager.secretAccessor`, `roles/pubsub.publisher` |

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

### Dockerfiles

#### Dashboard Server (`infra/docker/Dockerfile.dashboard`)

```dockerfile
# Multi-stage build for minimal image size
# Stage 1: Build React UI
FROM node:20-alpine AS ui-builder

WORKDIR /ui
COPY ui/package*.json ./
RUN npm ci --production=false
COPY ui/ .
RUN npm run build

# Stage 2: Build Go binary
FROM golang:1.22-alpine AS go-builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
# Copy built UI into embed location
COPY --from=ui-builder /ui/dist /app/internal/server/dist

# Build with embedded UI
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X main.Version=${VERSION:-dev}" \
    -o /ailang ./cmd/ailang

# Stage 3: Runtime image
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=go-builder /ailang /usr/local/bin/ailang
COPY --from=go-builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=go-builder /usr/share/zoneinfo /usr/share/zoneinfo

USER nonroot:nonroot

ENV AILANG_STORAGE=gcp
ENV PORT=8080

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/ailang"]
CMD ["serve", "--port", "8080"]
```

#### Coordinator Daemon (`infra/docker/Dockerfile.coordinator`)

```dockerfile
# Multi-stage build
FROM golang:1.22-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X main.Version=${VERSION:-dev}" \
    -o /ailang ./cmd/ailang

# Runtime image with git (needed for worktree operations)
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata git openssh-client

# Create non-root user
RUN adduser -D -u 1000 ailang
USER ailang

COPY --from=builder /ailang /usr/local/bin/ailang

# Create state directories
RUN mkdir -p /home/ailang/.ailang/state /home/ailang/.ailang/logs

ENV AILANG_STORAGE=gcp
ENV AILANG_STATE_DIR=/home/ailang/.ailang/state

ENTRYPOINT ["/usr/local/bin/ailang"]
CMD ["coordinator", "start", "--foreground"]
```

#### Agent Executor (`infra/docker/Dockerfile.agent-executor`)

```dockerfile
# Agent executor with Claude CLI, Gemini CLI, and ailang binary
# This container runs in Cloud Run Jobs, triggered by Eventarc from Pub/Sub
#
# The entrypoint is `ailang coordinator execute-job` which:
# 1. Fetches task from Firestore
# 2. Clones repo and creates branch
# 3. Executes via internal/executor (ClaudeCodeProvider or GeminiCLIProvider)
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

# Stage 2: Runtime with CLIs
FROM ubuntu:22.04

# Prevent interactive prompts
ENV DEBIAN_FRONTEND=noninteractive

# Install base dependencies
RUN apt-get update && apt-get install -y \
    curl \
    git \
    ca-certificates \
    jq \
    openssh-client \
    gnupg \
    && rm -rf /var/lib/apt/lists/*

# Install Node.js 20 (for Claude Code CLI and Gemini CLI)
RUN curl -fsSL https://deb.nodesource.com/setup_20.x | bash - \
    && apt-get install -y nodejs \
    && rm -rf /var/lib/apt/lists/*

# Install Claude Code CLI (npm package)
# The claude binary is the headless CLI for programmatic use
RUN npm install -g @anthropic-ai/claude-code@latest

# Install Gemini CLI (Google's AI coding assistant)
RUN npm install -g @google/generative-ai-cli@latest || echo "Gemini CLI optional"

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
ENV PATH="/usr/local/bin:${PATH}"

# Health check (verifies claude CLI is available)
HEALTHCHECK --interval=30s --timeout=5s --retries=3 \
    CMD claude --version || exit 1

# Entrypoint: ailang coordinator execute-job
# Receives AILANG_TASK_ID from Eventarc and executes the full task lifecycle
ENTRYPOINT ["/usr/local/bin/ailang", "coordinator", "execute-job"]
```

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
│     └──► ClaudeCodeProvider.Execute()                                        │
│          │                                                                   │
│          └──► internal/executor/claude/claude.go                             │
│               │                                                              │
│               └──► exec.Command("claude",                                    │
│                       "-p", directive,                                       │
│                       "--output-format", "stream-json",                      │
│                       "--permission-mode", "bypassPermissions",              │
│                       "--model", model,                                      │
│                       "--session-id", sessionID,                             │
│                       "--add-dir", workspace)                                │
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

**IMPORTANT: The existing AILANG implementation uses the `internal/executor/` package directly from Go code, NOT a shell script. The coordinator daemon orchestrates task execution via providers (`ClaudeCodeProvider`, `GeminiCLIProvider`, `ScriptProvider`).**

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
        Timeout:   30 * time.Minute,
        Workspace: workspace,
    }
    if agentConfig != nil && agentConfig.Invoke != nil {
        opts.InvokeConfig = agentConfig.Invoke
    }

    // 9. Execute task (uses ClaudeCodeProvider or GeminiCLIProvider)
    // The provider runs: `claude -p <directive> --output-format stream-json ...`
    // See internal/executor/claude/claude.go for the actual CLI invocation
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
    projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
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

```yaml
# Cloud Build configuration for AILANG services
# Triggers on push to main/dev branches
# Builds and deploys all three services

substitutions:
  _REGION: us-central1
  _ARTIFACT_REGISTRY: us-central1-docker.pkg.dev/${PROJECT_ID}/ailang

options:
  logging: CLOUD_LOGGING_ONLY
  machineType: E2_HIGHCPU_8

steps:
  # ═══════════════════════════════════════════════════════════════════════
  # STEP 1: Build Dashboard Image
  # ═══════════════════════════════════════════════════════════════════════
  - id: build-dashboard
    name: gcr.io/cloud-builders/docker
    args:
      - build
      - --file=infra/docker/Dockerfile.dashboard
      - --tag=${_ARTIFACT_REGISTRY}/dashboard:${SHORT_SHA}
      - --tag=${_ARTIFACT_REGISTRY}/dashboard:latest
      - --cache-from=${_ARTIFACT_REGISTRY}/dashboard:latest
      - --build-arg=VERSION=${TAG_NAME:-${SHORT_SHA}}
      - .

  # ═══════════════════════════════════════════════════════════════════════
  # STEP 2: Build Coordinator Image
  # ═══════════════════════════════════════════════════════════════════════
  - id: build-coordinator
    name: gcr.io/cloud-builders/docker
    args:
      - build
      - --file=infra/docker/Dockerfile.coordinator
      - --tag=${_ARTIFACT_REGISTRY}/coordinator:${SHORT_SHA}
      - --tag=${_ARTIFACT_REGISTRY}/coordinator:latest
      - --cache-from=${_ARTIFACT_REGISTRY}/coordinator:latest
      - --build-arg=VERSION=${TAG_NAME:-${SHORT_SHA}}
      - .

  # ═══════════════════════════════════════════════════════════════════════
  # STEP 3: Build Agent Executor Image
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
  # STEP 4: Push Images to Artifact Registry (in parallel)
  # ═══════════════════════════════════════════════════════════════════════
  - id: push-dashboard
    name: gcr.io/cloud-builders/docker
    args: [push, --all-tags, '${_ARTIFACT_REGISTRY}/dashboard']
    waitFor: [build-dashboard]

  - id: push-coordinator
    name: gcr.io/cloud-builders/docker
    args: [push, --all-tags, '${_ARTIFACT_REGISTRY}/coordinator']
    waitFor: [build-coordinator]

  - id: push-agent-executor
    name: gcr.io/cloud-builders/docker
    args: [push, --all-tags, '${_ARTIFACT_REGISTRY}/agent-executor']
    waitFor: [build-agent-executor]

  # ═══════════════════════════════════════════════════════════════════════
  # STEP 5: Deploy Dashboard to Cloud Run
  # ═══════════════════════════════════════════════════════════════════════
  - id: deploy-dashboard
    name: gcr.io/google.com/cloudsdktool/cloud-sdk
    entrypoint: gcloud
    args:
      - run
      - deploy
      - ailang-dashboard
      - --image=${_ARTIFACT_REGISTRY}/dashboard:${SHORT_SHA}
      - --region=${_REGION}
      - --platform=managed
      - --allow-unauthenticated=false  # IAP handles auth
      - --service-account=ailang-dashboard@${PROJECT_ID}.iam.gserviceaccount.com
      - --set-env-vars=AILANG_STORAGE=gcp,GOOGLE_CLOUD_PROJECT=${PROJECT_ID}
      - --set-secrets=ANTHROPIC_API_KEY=anthropic-api-key:latest,GITHUB_TOKEN=github-token:latest
      - --min-instances=1
      - --max-instances=10
      - --memory=512Mi
      - --cpu=1
      - --port=8080
      - --vpc-connector=ailang-vpc-connector
      - --vpc-egress=all-traffic
    waitFor: [push-dashboard]

  # ═══════════════════════════════════════════════════════════════════════
  # STEP 6: Deploy Coordinator to Cloud Run
  # ═══════════════════════════════════════════════════════════════════════
  - id: deploy-coordinator
    name: gcr.io/google.com/cloudsdktool/cloud-sdk
    entrypoint: gcloud
    args:
      - run
      - deploy
      - ailang-coordinator
      - --image=${_ARTIFACT_REGISTRY}/coordinator:${SHORT_SHA}
      - --region=${_REGION}
      - --platform=managed
      - --no-allow-unauthenticated
      - --service-account=ailang-coordinator@${PROJECT_ID}.iam.gserviceaccount.com
      - --set-env-vars=AILANG_STORAGE=gcp,GOOGLE_CLOUD_PROJECT=${PROJECT_ID},COORDINATOR_MODE=cloud
      - --set-secrets=ANTHROPIC_API_KEY=anthropic-api-key:latest,GITHUB_TOKEN=github-token:latest,GOOGLE_API_KEY=google-api-key:latest
      - --min-instances=1
      - --max-instances=3
      - --memory=1Gi
      - --cpu=1
      - --timeout=3600
      - --vpc-connector=ailang-vpc-connector
      - --vpc-egress=all-traffic
    waitFor: [push-coordinator]

  # ═══════════════════════════════════════════════════════════════════════
  # STEP 7: Update Cloud Run Job for Agent Executor
  # ═══════════════════════════════════════════════════════════════════════
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
      - --set-env-vars=AILANG_STORAGE=gcp,GOOGLE_CLOUD_PROJECT=${PROJECT_ID}
      - --set-secrets=ANTHROPIC_API_KEY=anthropic-api-key:latest,GITHUB_TOKEN=github-token:latest,GOOGLE_API_KEY=google-api-key:latest
      - --memory=4Gi
      - --cpu=2
      - --task-timeout=1800
      - --max-retries=1
      - --vpc-connector=ailang-vpc-connector
      - --vpc-egress=all-traffic
    waitFor: [push-agent-executor]

images:
  - ${_ARTIFACT_REGISTRY}/dashboard:${SHORT_SHA}
  - ${_ARTIFACT_REGISTRY}/dashboard:latest
  - ${_ARTIFACT_REGISTRY}/coordinator:${SHORT_SHA}
  - ${_ARTIFACT_REGISTRY}/coordinator:latest
  - ${_ARTIFACT_REGISTRY}/agent-executor:${SHORT_SHA}
  - ${_ARTIFACT_REGISTRY}/agent-executor:latest

timeout: 1800s  # 30 minutes
```

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
}

func NewPubSubBroker(ctx context.Context) (*PubSubBroker, error) {
    projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
    if projectID == "" {
        return nil, fmt.Errorf("GOOGLE_CLOUD_PROJECT not set")
    }

    client, err := pubsub.NewClient(ctx, projectID)
    if err != nil {
        return nil, fmt.Errorf("failed to create pubsub client: %w", err)
    }

    return &PubSubBroker{
        client:    client,
        projectID: projectID,
        topics:    make(map[string]*pubsub.Topic),
        subs:      make(map[string]*pubsub.Subscription),
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

            // Store Pub/Sub message ID for ack/nack
            msg.pubsubMsgID = m.ID
            msg.pubsubAckID = m.AckID

            select {
            case msgChan <- &msg:
            case <-ctx.Done():
                m.Nack()
                return
            }
        })

        if err != nil && err != context.Canceled {
            // Log error
        }
    }()

    return msgChan, nil
}

func (b *PubSubBroker) Ack(ctx context.Context, inbox string, msgID string) error {
    // Note: In Pub/Sub, acks happen automatically when Receive callback returns
    // This is for explicit ack scenarios
    return nil
}

func (b *PubSubBroker) Nack(ctx context.Context, inbox string, msgID string) error {
    // Nacks are handled in the Receive callback
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

#### IAP Configuration (`infra/terraform/iap.tf`)

```hcl
# ═══════════════════════════════════════════════════════════════════════════
# IDENTITY-AWARE PROXY (IAP) FOR DASHBOARD
# ═══════════════════════════════════════════════════════════════════════════

# Enable IAP API
resource "google_project_service" "iap" {
  service            = "iap.googleapis.com"
  disable_on_destroy = false
}

# OAuth consent screen (required for IAP)
resource "google_iap_brand" "ailang" {
  support_email     = var.support_email
  application_title = "AILANG Control Plane"
  project           = var.project_id
}

# OAuth client for IAP
resource "google_iap_client" "dashboard" {
  display_name = "AILANG Dashboard"
  brand        = google_iap_brand.ailang.name
}

# IAP settings for Cloud Run
resource "google_iap_web_backend_service_iam_binding" "dashboard" {
  project             = var.project_id
  web_backend_service = google_compute_backend_service.dashboard.name
  role                = "roles/iap.httpsResourceAccessor"
  members             = var.dashboard_users  # List of users/groups
}

# ═══════════════════════════════════════════════════════════════════════════
# LOAD BALANCER FOR IAP (Required for Cloud Run + IAP)
# ═══════════════════════════════════════════════════════════════════════════

# External IP
resource "google_compute_global_address" "dashboard" {
  name = "ailang-dashboard-ip"
}

# SSL certificate (managed)
resource "google_compute_managed_ssl_certificate" "dashboard" {
  name = "ailang-dashboard-cert"
  managed {
    domains = [var.dashboard_domain]  # e.g., "ailang.sunholo.com"
  }
}

# Network Endpoint Group for Cloud Run
resource "google_compute_region_network_endpoint_group" "dashboard" {
  name                  = "ailang-dashboard-neg"
  region                = var.region
  network_endpoint_type = "SERVERLESS"

  cloud_run {
    service = google_cloud_run_v2_service.dashboard.name
  }
}

# Backend service
resource "google_compute_backend_service" "dashboard" {
  name                  = "ailang-dashboard-backend"
  protocol              = "HTTPS"
  port_name             = "http"
  timeout_sec           = 30
  enable_cdn            = false

  iap {
    oauth2_client_id     = google_iap_client.dashboard.client_id
    oauth2_client_secret = google_iap_client.dashboard.secret
  }

  backend {
    group = google_compute_region_network_endpoint_group.dashboard.id
  }
}

# URL map
resource "google_compute_url_map" "dashboard" {
  name            = "ailang-dashboard-urlmap"
  default_service = google_compute_backend_service.dashboard.id
}

# HTTPS proxy
resource "google_compute_target_https_proxy" "dashboard" {
  name             = "ailang-dashboard-proxy"
  url_map          = google_compute_url_map.dashboard.id
  ssl_certificates = [google_compute_managed_ssl_certificate.dashboard.id]
}

# Forwarding rule
resource "google_compute_global_forwarding_rule" "dashboard" {
  name                  = "ailang-dashboard-forwarding"
  target                = google_compute_target_https_proxy.dashboard.id
  port_range            = "443"
  ip_address            = google_compute_global_address.dashboard.address
  load_balancing_scheme = "EXTERNAL_MANAGED"
}

# DNS record (if using Cloud DNS)
resource "google_dns_record_set" "dashboard" {
  count        = var.create_dns_record ? 1 : 0
  name         = "${var.dashboard_domain}."
  managed_zone = var.dns_zone_name
  type         = "A"
  ttl          = 300
  rrdatas      = [google_compute_global_address.dashboard.address]
}

# ═══════════════════════════════════════════════════════════════════════════
# VARIABLES FOR IAP
# ═══════════════════════════════════════════════════════════════════════════

variable "support_email" {
  description = "Support email for OAuth consent screen"
  type        = string
  default     = "support@sunholo.com"
}

variable "dashboard_domain" {
  description = "Domain for dashboard (e.g., ailang.sunholo.com)"
  type        = string
}

variable "dashboard_users" {
  description = "List of users/groups allowed to access dashboard"
  type        = list(string)
  default     = []
  # Example: ["user:mark@sunholo.com", "group:ailang-team@sunholo.com"]
}

variable "create_dns_record" {
  description = "Whether to create a DNS record for the dashboard"
  type        = bool
  default     = false
}

variable "dns_zone_name" {
  description = "Cloud DNS zone name for dashboard domain"
  type        = string
  default     = ""
}
```

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

1. **Auth**: Should we add Google Cloud IAM auth to dashboard before cloud deploy?
2. **Multi-region**: Is single-region (us-central1) sufficient for initial deployment?
3. **Monitoring**: Should we integrate Cloud Monitoring/Alerting from day 1?
4. **CI/CD**: Should we add Cloud Build triggers for automated deployment?

---

## Related Documents

- [M-GENERIC-PIPELINE](m-generic-pipeline.md) - **Prerequisite**: Config-driven agent pipelines (removes hardcoded TaskStage)
- [M-CLOUD-STORAGE](../v0_7_0/m-cloud-storage.md) - Original cloud storage design (planned)
- [M-OTEL-DASHBOARD](../../implemented/v0_6_3/m-otel-dashboard.md) - Observatory foundation
- [M-COORD-STABLE](../../implemented/v0_6_2/m-coord-stable.md) - Coordinator architecture
- [M-CONTROL-PLANE-V4](../../implemented/v0_6_4/m-control-plane-v4.md) - Dashboard design
