# M-PKG-AUTONOMOUS-UPDATES: Message-Driven Autonomous Package Updates

**Status**: Planned
**Target**: v0.10.0
**Priority**: P1 (High)
**Estimated**: 2 weeks (Phase 1-2: 1 week, Phase 3-4: 1 week)
**Dependencies**: M-PKG-REGISTRY (implemented v0.9.7), M-PKG-MSG (implemented v0.8.0), Cloud execution (implemented v0.9.0+)

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Package updates are reproducible: same message + same code = same result. Provenance chain makes every update traceable |
| A2: Replayability | +1 | Full message chain recorded in registry metadata; cascade can be replayed from any trigger point |
| A3: Effect Legibility | +1 | All side effects (publish, git push, message send) are explicit in agent configs and message schemas |
| A4: Explicit Authority | +1 | Graduated autonomy: change class determines approval gates. No ambient access — agents operate within declared capabilities |
| A5: Bounded Verification | 0 | No change to verification — existing `ailang check --package` and registry validator unchanged |
| A6: Safe Concurrency | +1 | `max_concurrent_tasks: 1` per package agent + worktree isolation prevents conflicts. DAG scheduling prevents cascade races |
| A7: Machines First | +1 | Entire system is machine-driven: messages trigger agents, agents follow structured workflows, provenance is machine-queryable |
| A8: Minimal Syntax | 0 | No language syntax changes |
| A9: Cost Visibility | +1 | Per-task cost budgets already exist in coordinator. Provenance records which agent spent what |
| A10: Composability | +1 | Composes with existing message kinds, agent configs, and approval workflows |
| A11: Structured Failure | 0 | Uses existing error handling; circuit breaker adds structured cascade failure |
| A12: System Boundary | +1 | Registry publish is an explicit system boundary crossing. Provenance tracks crossing metadata |

**Net Score: +9** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): Message-driven pipeline is fully deterministic given same inputs
- [x] A3 (Effects): All effects declared in agent capabilities and message schemas
- [x] A4 (Authority): Graduated autonomy with explicit change-class-to-approval mapping
- [x] A7 (Machines First): System designed for autonomous agents, not human convenience

## Problem Statement

AILANG has 14 published packages in a monorepo (`ailang-packages`), with three mature systems that operate independently:

1. **Messaging**: Package inboxes (`pkg:sunholo/auth`), 11 message kinds, auto-emit on publish
2. **Registry**: GCS-backed, validator on Cloud Run, content/interface/tarball hashing
3. **Cloud execution**: Coordinator daemon, Cloud Run Jobs, full design→sprint→execute pipeline

**Current State:**
- When core AILANG publishes a new version, **nothing happens** to packages — a human must manually check each one
- When a package publishes, `emitPublishMessages()` sends to the *package's own inbox* but **not to its dependents**
- No agent watches `pkg:*` inboxes — messages arrive but nobody processes them
- No provenance trail records *why* a package was updated or *what triggered it*
- Core ailang already has the full 3-phase chain configured in `config.cloud.yaml` with human approval gates, but packages have no equivalent

**Impact:**
- Package staleness: 14 packages fall behind as AILANG evolves
- Manual overhead: developer must check each package after every core change
- No audit trail: impossible to trace the chain of events leading to a package version
- Wasted infrastructure: messaging and cloud execution systems exist but aren't connected for packages

## Goals

**Primary Goal:** Connect the messaging, registry, and cloud execution systems so package updates flow autonomously from trigger message to published version with full provenance.

**Success Metrics:**
- Package agent picks up `pkg:sunholo/auth` message and executes within 60s of arrival
- Class A (patch) updates complete end-to-end without human intervention
- Provenance chain queryable: `ailang pkg provenance sunholo/auth@0.2.0` shows trigger→approval→publish
- Cascade update of 3+ packages completes in topological order without duplicates
- Core ailang updates visible in same system with stricter approval gates (already configured)

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| One agent per package vs template agent | Determines config scale and inbox routing architecture | human | design | high |
| `subdirectory` field on AgentConfig | Enables monorepo package targeting; touches agent dispatch path | human | design | high |
| Change-class-to-autonomy mapping | Determines what runs unattended vs needs human approval | human | design | med |
| ProvenanceInfo schema in registry metadata | Extends published artifact format — must be backward compatible | human | design | high |
| Dependent notification on publish | Determines cascade trigger mechanism | agent | compile | med |
| AGENT.md as first-class registry artifact | Changes registry storage layout | agent | compile | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] One agent per package (not template) — 14 agents is manageable, each gets its own inbox
- [x] Add `subdirectory` field to AgentConfig for monorepo support
- [x] Class A = auto, Class B = semi-auto, Class C = human gated
- [ ] ProvenanceInfo schema finalized (proposed below, needs review)

## Solution Design

### Overview

Package agents are registered in `config.cloud.yaml` following the exact same pattern as existing workspace agents (ailang, stapledons_voyage, TwilightGame). Each package gets a dedicated agent watching its `pkg:vendor/name` inbox. When a message arrives (typically `upgrade-available` from a dependency), the coordinator triages it by change class and dispatches the appropriate workflow depth.

### Architecture

```
ailang publish (core or package)
       │
       ▼
emitPublishMessages()
  + NEW: emitDependentNotifications()
       │
       ▼
┌──────────────────────────────────────────────────┐
│  pkg:sunholo/auth   pkg:sunholo/gcp-auth   ...  │  Package inboxes
└──────────────┬───────────────────────────────────┘
               │
               ▼
┌──────────────────────────────────────┐
│  Coordinator Daemon                  │
│  InboxMessageAdapter polls pkg:*     │
│  TriagePackageMessage() → class A/B/C│
│  DynamicAutonomyRouter adjusts config│
└──────────────┬───────────────────────┘
               │
        ┌──────┴──────┐
        ▼             ▼
   Class A         Class B/C
   Single-step     Sprint chain
        │             │
        ▼             ▼
┌──────────────┐  ┌──────────────────────────────┐
│ Cloud Run Job│  │ design-doc → sprint-planner → │
│ Update deps  │  │ sprint-executor               │
│ Run tests    │  │ (same chain as core ailang)    │
│ Publish      │  └──────────────────────────────┘
└──────┬───────┘
       │
       ▼
Registry publish with ProvenanceInfo
       │
       ▼
emitDependentNotifications() → cascade to downstream
```

### Component 1: Package Agent Configuration

Add one agent per package to `config.cloud.yaml`. Example for `sunholo/auth`:

```yaml
# ═══════════════════════════════════════════════════════════
# AILANG PACKAGES (ailang-packages monorepo)
# ═══════════════════════════════════════════════════════════

- id: pkg-sunholo-auth
  label: "Package: sunholo/auth"
  inbox: "pkg:sunholo/auth"
  workspace: sunholo-data/ailang-packages
  merge_branch: main
  subdirectory: packages/auth          # NEW FIELD
  capabilities: [code, test, docs]
  provider: claude
  model: sonnet
  timeout: "30m"
  idle_timeout: "3m"
  auto_approve_handoffs: true          # Dynamic: overridden for class C
  auto_merge: false
  session_continuity: true
  max_concurrent_tasks: 1
  invoke:
    type: prompt
    template_file: /etc/ailang/templates/pkg-update.md
  output_markers:
    - "PUBLISH_RESULT:"
    - "VERSION:"
  artifact_patterns:
    - "packages/auth/**/*"
  git_mode: guardrails
```

**New `subdirectory` field** on `AgentConfig`:
```go
// AgentConfig in internal/coordinator/agent_registry.go
type AgentConfig struct {
    // ... existing fields ...

    // Subdirectory within the workspace for monorepo support (v0.10.0).
    // Agent's working context is scoped to this directory within the worktree.
    // Used by: worktree setup (cd), invoke template ({{.Subdirectory}}), artifact patterns.
    Subdirectory string `yaml:"subdirectory" json:"subdirectory,omitempty"`
}
```

**Agent invoke template** (`/etc/ailang/templates/pkg-update.md`):
```markdown
You are an autonomous package maintainer for the AILANG package `{{.PackageName}}`.

## Working Context
- Repository: ailang-packages (monorepo)
- Package directory: {{.Subdirectory}}/
- Read AGENT.md in the package directory for package-specific instructions

## Incoming Message
{{.Content}}

## Change Class: {{.ChangeClass}}

## Instructions by Change Class

### Class A (patch/internal)
1. Update `ailang.toml` version constraint if needed
2. Run `ailang check --package .` and `ailang test --package .`
3. Bump patch version in ailang.toml
4. Update CHANGELOG.md
5. Run `ailang publish`

### Class B (minor/additive)
1. Read the interface change notice carefully
2. Update imports or API usage as needed
3. Add/update tests for changed interfaces
4. Bump minor version
5. Update CHANGELOG.md
6. Run `ailang publish`

### Class C (breaking/major)
1. Create a design doc in {{.Subdirectory}}/design_docs/
2. Plan migration strategy
3. Implement changes with full test coverage
4. Bump major version
5. Update CHANGELOG.md and AGENT.md
6. Run `ailang publish`
```

### Component 2: Dynamic Autonomy Router

A pre-execution hook in the coordinator that adjusts agent config based on change class:

```go
// internal/coordinator/autonomy_router.go

// AdjustAutonomyForChangeClass modifies the effective agent config
// based on the incoming package message's change class.
func AdjustAutonomyForChangeClass(agent *AgentConfig, msg *InboxMessage) *AgentConfig {
    env, err := messaging.ExtractPackageEnvelope(msg)
    if err != nil || env == nil {
        return agent // Not a package message, use defaults
    }

    effective := *agent // Copy
    switch classifyChange(env) {
    case ChangeClassA:
        // Patch: fully autonomous
        effective.SkipApproval = true
        effective.AutoMerge = true
        effective.AutoApproveHandoffs = true
    case ChangeClassB:
        // Minor: auto-approve intermediate steps, human reviews publish
        effective.SkipApproval = false
        effective.AutoMerge = false
        effective.AutoApproveHandoffs = true
    case ChangeClassC:
        // Breaking: full human approval at every gate
        effective.SkipApproval = false
        effective.AutoMerge = false
        effective.AutoApproveHandoffs = false
        // Trigger full 3-phase chain
        effective.TriggerOnComplete = []string{
            fmt.Sprintf("pkg-%s-sprint-planner", sanitize(env.Package.Name)),
        }
    }
    return &effective
}

func classifyChange(env *PackageMessageEnvelope) ChangeClass {
    if env.Package.Breaking {
        return ChangeClassC
    }
    switch env.Kind {
    case PkgMsgEffectWidening:
        return ChangeClassC // Effect ceiling changes always need review
    case PkgMsgInterfaceChange:
        if env.Package.ChangeClass == "major" {
            return ChangeClassC
        }
        return ChangeClassB
    case PkgMsgUpgradeAvailable:
        if env.Package.ChangeClass == "minor" {
            return ChangeClassB
        }
        return ChangeClassA
    default:
        return ChangeClassB // Conservative default
    }
}
```

### Component 3: Dependent Notification

Extend `emitPublishMessages()` in `cmd/ailang/pkg_publish.go`:

```go
// emitDependentNotifications queries the registry index for packages
// that depend on the just-published package and sends them upgrade-available.
func emitDependentNotifications(store *messaging.Store, registry *pkg.RegistryClient,
    publishedPkg string, newVersion string, interfaceHashChanged bool) error {

    index, err := registry.FetchIndex()
    if err != nil {
        return fmt.Errorf("failed to fetch registry index: %w", err)
    }

    for _, entry := range index.Packages {
        if entry.Name == publishedPkg {
            continue // Skip self
        }
        // Check if this package depends on the published package
        meta, err := registry.FetchMetadata(entry.Name, entry.Latest)
        if err != nil {
            continue // Best effort
        }
        if !dependsOn(meta.Manifest, publishedPkg) {
            continue
        }

        changeClass := "patch"
        if interfaceHashChanged {
            changeClass = "minor"
        }

        env := &messaging.PackageMessageEnvelope{
            Schema: messaging.PackageMessageSchema,
            Kind:   messaging.PkgMsgUpgradeAvailable,
            From:   "registry",
            To:     []string{messaging.FormatPackageInbox(entry.Name)},
            Package: messaging.PackageRef{
                Name:        publishedPkg,
                ToVersion:   newVersion,
                ChangeClass: changeClass,
            },
            Summary: fmt.Sprintf("Dependency %s updated to %s", publishedPkg, newVersion),
        }
        // Send to dependent's inbox
        messaging.EmitPackageMessage(store, env)
    }
    return nil
}
```

### Component 4: Registry Provenance Extension

Add to `internal/pkg/registry_types.go`:

```go
type PackageMetadata struct {
    // ... existing fields ...
    Provenance *ProvenanceInfo `json:"provenance,omitempty"`
}

type ProvenanceInfo struct {
    TriggerMessageID string   `json:"trigger_message_id"`          // Message that started this update
    CorrelationIDs   []string `json:"correlation_ids,omitempty"`   // Full message chain
    AgentTraceID     string   `json:"agent_trace_id,omitempty"`    // OTEL trace ID
    ChainID          string   `json:"chain_id,omitempty"`          // Observatory chain ID
    ApprovedBy       string   `json:"approved_by,omitempty"`       // Human approver GitHub handle
    ApprovedAt       string   `json:"approved_at,omitempty"`       // ISO 8601 timestamp
    AutoApproved     bool     `json:"auto_approved"`               // true for class A
    ChangeClass      string   `json:"change_class"`                // "A", "B", or "C"
    PreviousVersion  string   `json:"previous_version,omitempty"`  // Version before this update
}
```

**AGENT.md as first-class artifact**: Modify the registry validator to upload `AGENT.md` as a separate blob at `packages/vendor/name/version/AGENT.md` alongside `package.tar.gz` and `metadata.json`. This makes it directly fetchable for AI discovery without downloading the tarball.

### Component 4b: Version History Log

Each package version gets a `history.json` file uploaded alongside `metadata.json`. This records the message chain, actions taken, and outcomes — making every update auditable and surfaceable on the registry website.

**Registry storage layout (extended):**
```
gs://ailang-registry/packages/sunholo/auth/0.2.0/
  package.tar.gz      # Source tarball
  metadata.json       # Validation results + hashes + provenance
  AGENT.md            # AI discovery guide (new)
  history.json        # Message + action log for this version (new)
```

**History schema:**
```go
// internal/pkg/registry_types.go

// VersionHistory records the full message and action trail for a published version.
// Stored as history.json in the registry, surfaced on the website for package discovery.
type VersionHistory struct {
    Schema    string         `json:"schema"`     // "ailang.version-history/v1"
    Package   string         `json:"package"`    // "sunholo/auth"
    Version   string         `json:"version"`    // "0.2.0"
    Previous  string         `json:"previous"`   // "0.1.0"
    CreatedAt string         `json:"created_at"` // ISO 8601
    Messages  []HistoryEntry `json:"messages"`   // Ordered message trail
    Summary   string         `json:"summary"`    // AI-generated 1-line summary of what changed
}

// HistoryEntry is a single event in the version's message/action trail.
type HistoryEntry struct {
    Timestamp string `json:"timestamp"`
    Kind      string `json:"kind"`      // Message kind or action type
    From      string `json:"from"`      // Who sent/did it
    Title     string `json:"title"`     // Brief description
    Detail    string `json:"detail"`    // Full content (message payload or action result)
    Status    string `json:"status"`    // "received", "acknowledged", "completed", "failed"
}
```

**Example `history.json`:**
```json
{
  "schema": "ailang.version-history/v1",
  "package": "sunholo/gcp-auth",
  "version": "0.2.0",
  "previous": "0.1.0",
  "created_at": "2026-04-01T14:30:00Z",
  "messages": [
    {
      "timestamp": "2026-04-01T12:00:00Z",
      "kind": "upgrade-available",
      "from": "registry",
      "title": "Dependency sunholo/auth updated to 0.2.0",
      "detail": "Interface hash changed: abc123 → def456. Breaking: true.",
      "status": "received"
    },
    {
      "timestamp": "2026-04-01T12:01:00Z",
      "kind": "triage",
      "from": "coordinator",
      "title": "Change class: C (breaking) → full 3-phase workflow",
      "detail": "Dynamic autonomy: auto_approve_handoffs=false",
      "status": "completed"
    },
    {
      "timestamp": "2026-04-01T12:05:00Z",
      "kind": "design-doc-created",
      "from": "pkg-sunholo-gcp-auth",
      "title": "Migration design doc for auth 0.2.0",
      "detail": "packages/gcp-auth/design_docs/migrate-auth-0.2.md",
      "status": "completed"
    },
    {
      "timestamp": "2026-04-01T13:00:00Z",
      "kind": "approval",
      "from": "MarkEdmondson1234",
      "title": "Design approved",
      "detail": "GitHub label: design-approved on issue #42",
      "status": "completed"
    },
    {
      "timestamp": "2026-04-01T13:30:00Z",
      "kind": "sprint-executed",
      "from": "pkg-sunholo-gcp-auth",
      "title": "Migration implemented: updated imports, added tests",
      "detail": "Files changed: gcp_oauth.ail (+12 -5), gcp_oauth_test.ail (+30)",
      "status": "completed"
    },
    {
      "timestamp": "2026-04-01T14:15:00Z",
      "kind": "approval",
      "from": "MarkEdmondson1234",
      "title": "Implementation approved",
      "detail": "GitHub label: implementation-approved on issue #42",
      "status": "completed"
    },
    {
      "timestamp": "2026-04-01T14:30:00Z",
      "kind": "published",
      "from": "pkg-sunholo-gcp-auth",
      "title": "Published sunholo/gcp-auth@0.2.0",
      "detail": "Content hash: sha256:abc..., interface hash: sha256:def...",
      "status": "completed"
    }
  ],
  "summary": "Breaking migration: updated gcp-auth to use auth 0.2.0 new key validation API"
}
```

**Website/discovery use cases:**
- **Package detail page**: Show timeline of how each version was created (human-approved? auto-updated? what triggered it?)
- **Package health signal**: Packages with recent `history.json` entries are actively maintained
- **Dependency impact view**: "When auth@0.2.0 published, these 3 packages auto-updated within 2 hours"
- **Trust indicator**: Provenance chain visible — users can see if a version was human-reviewed or fully autonomous
- **Search enrichment**: `summary` field feeds into `ai_summary` for better package discovery

**How it's built**: The coordinator's completion handler collects `HistoryEntry` events throughout the task lifecycle (message received, triage decision, design doc created, approvals, publish result). On successful publish, these are serialized to `history.json` and uploaded to the registry alongside the package.

**Index extension** — add to `IndexEntry` for website listing:
```go
type IndexEntry struct {
    // ... existing fields ...
    LastUpdated    string `json:"last_updated,omitempty"`     // When the latest version was published
    UpdatedBy      string `json:"updated_by,omitempty"`       // "human", "agent", or agent ID
    LatestSummary  string `json:"latest_summary,omitempty"`   // From history.json summary
}
```

### Component 5: Cascade Management

```go
// internal/coordinator/cascade_scheduler.go

// ScheduleCascadeUpdate determines the execution order for a dependency graph update.
// Returns packages in topological order (leaf deps first).
func ScheduleCascadeUpdate(index *pkg.RegistryIndex, triggerPkg string) []string {
    // Build dependency graph from registry index
    // Topological sort with the trigger package as root
    // Return ordered list of packages to update
}

// CascadeCircuitBreaker tracks failures during a cascade update.
// If 3+ packages fail, pauses the cascade and alerts the human.
type CascadeCircuitBreaker struct {
    MaxFailures    int
    FailureCount   int
    CorrelationID  string
}
```

**Dedup**: Use correlation IDs to group all messages in a cascade. `SupersedeOlderMessages()` already handles within a single package. For cross-package cascades, the correlation ID prevents re-triggering: if a package already has an unread message with the same correlation ID, skip it.

### Component 6: Skills Distribution

Package agents receive skills through the existing plugin system:

1. **`ailang_bootstrap`** (pre-baked in Docker image at `/plugins/ailang_bootstrap`): Provides design-doc-creator, sprint-planner, sprint-executor, ailang-inbox — sufficient for the full 3-phase workflow
2. **AGENT.md** per package: Provides package-specific context as system prompt material
3. **Optional `plugin_dirs`**: If a package needs the `ailang-packages` skill for publishing context, add it as a second plugin directory in the agent config

### Implementation Plan

**Phase 1: MVP — Single Package, Local** (~3 days)
- [ ] Add `Subdirectory` field to `AgentConfig` struct
- [ ] Wire `Subdirectory` through worktree setup (agent's working dir = `worktree/subdirectory`)
- [ ] Add one package agent config to local `~/.ailang/config.yaml` (sunholo/auth)
- [ ] Verify InboxMessageAdapter works with `pkg:sunholo/auth` inbox format (colon in name)
- [ ] Manual test: `ailang messages send pkg:sunholo/auth "test update" --title "Test"` → verify daemon picks it up
- [ ] Agent executes in correct subdirectory, runs tests, reports result

**Phase 2: Cloud + Multi-Package** (~2 days)
- [ ] Add all 14 package agent configs to `config.cloud.yaml`
- [ ] Add workspace mapping for `sunholo-data/ailang-packages` in workspaces config
- [ ] Create `pkg-update.md` invoke template (uploaded to GCS config or baked into plugin)
- [ ] Deploy to dev environment, test with one package via Pub/Sub message
- [ ] Verify worktree isolation — two package agents don't conflict in same monorepo

**Phase 3: Dependent Notification + Dynamic Autonomy** (~3 days)
- [ ] Implement `emitDependentNotifications()` in `cmd/ailang/pkg_publish.go`
- [ ] Add `dependsOn()` helper that checks registry manifests
- [ ] Implement `AdjustAutonomyForChangeClass()` pre-execution hook
- [ ] Wire hook into `executeTask()` in `daemon_tasks_exec.go`
- [ ] Test cascade: publish `sunholo/auth` → verify `sunholo/gcp-auth` gets notified

**Phase 4: Provenance + History + AGENT.md + Cascade** (~3 days)
- [ ] Add `ProvenanceInfo` to `PackageMetadata` (backward compatible — optional field)
- [ ] Modify publish flow to embed provenance from triggering message context
- [ ] Implement `HistoryCollector` — captures events during task lifecycle into `VersionHistory`
- [ ] Upload `history.json` alongside package on publish (message trail + actions + approvals)
- [ ] Extend `IndexEntry` with `last_updated`, `updated_by`, `latest_summary` for website
- [ ] Upload AGENT.md as separate blob in registry validator
- [ ] Implement `ScheduleCascadeUpdate()` with topological sort
- [ ] Add `CascadeCircuitBreaker` (3 failures = pause + alert)
- [ ] Add `ailang pkg provenance vendor/name@version` CLI command
- [ ] Add `ailang pkg history vendor/name@version` CLI command (fetches + displays history.json)

### Files to Modify/Create

**New files:**
- `internal/coordinator/autonomy_router.go` — Change-class-to-autonomy mapping (~80 LOC)
- `internal/coordinator/cascade_scheduler.go` — Topological cascade scheduling (~120 LOC)
- `internal/coordinator/history_collector.go` — Collect HistoryEntry events during task lifecycle (~100 LOC)

**Modified files:**
- `internal/coordinator/agent_registry.go` — Add `Subdirectory` field (~5 LOC)
- `internal/coordinator/daemon_tasks_exec.go` — Wire autonomy router hook + history collection (~30 LOC)
- `internal/coordinator/worktree.go` — Use `Subdirectory` for agent working dir (~10 LOC)
- `cmd/ailang/pkg_publish.go` — Add `emitDependentNotifications()`, upload `history.json` (~80 LOC)
- `internal/pkg/registry_types.go` — Add `ProvenanceInfo`, `VersionHistory`, `HistoryEntry` structs, `IndexEntry` extensions (~60 LOC)
- `cmd/registry-validator/main.go` — Upload AGENT.md + history.json as separate blobs (~25 LOC)

**Config files:**
- `config/config.cloud.yaml` (ailang-multivac) — Add 14 package agent entries
- GCS templates — `pkg-update.md` invoke template

## Examples

### Example 1: Class A — Patch Update (Fully Autonomous)

**Trigger:** Core AILANG publishes v0.10.0 with no interface changes.

```bash
# Automatically emitted by ailang publish:
ailang messages send pkg:sunholo/auth \
  '{"schema":"ailang.package-message/v1","kind":"upgrade-available","package":{"name":"ailang","to_version":"0.10.0","change_class":"patch","breaking":false}}' \
  --title "AILANG 0.10.0 available" --from registry

# Coordinator picks up message within 60s
# Agent runs in ailang-packages/packages/auth/:
#   1. Updates ailang version constraint in ailang.toml
#   2. Runs ailang check --package . && ailang test --package .
#   3. Bumps version 0.1.0 → 0.1.1
#   4. Runs ailang publish
#   5. Registry records provenance: auto_approved=true, change_class=A
```

### Example 2: Class C — Breaking Change (Human Gated)

**Trigger:** Package `sunholo/auth` publishes v0.2.0 with changed export signatures.

```bash
# Auto-emitted to dependents:
# → pkg:sunholo/gcp-auth gets interface-change-notice (breaking=true)

# Coordinator triages: Class C → full 3-phase chain
# 1. Agent creates design_docs/ in packages/gcp-auth/design_docs/migration-auth-0.2.md
# 2. GitHub issue created with needs-design-approval label
# 3. Human reviews, approves design
# 4. Sprint planner runs, creates plan
# 5. Human reviews, approves sprint
# 6. Sprint executor implements migration
# 7. Human reviews, approves implementation
# 8. ailang publish with provenance: approved_by="MarkEdmondson1234"
```

### Example 3: Querying Provenance

```bash
$ ailang pkg provenance sunholo/gcp-auth@0.2.0

Version: 0.2.0
Published: 2026-04-01T14:30:00Z
Published By: pkg-sunholo-gcp-auth (agent)

Provenance:
  Trigger: msg_20260401_120000_abc123 (upgrade-available from sunholo/auth@0.2.0)
  Correlation: [msg_20260401_120000_abc123, msg_20260401_130000_def456]
  Change Class: C (breaking)
  Approved By: MarkEdmondson1234
  Approved At: 2026-04-01T14:15:00Z
  Agent Trace: 4bf92f3577b34da6a3ce929d0e0e4736
  Chain ID: chain-20260401-gcp-auth-migration
  Previous Version: 0.1.0
```

## Success Criteria

- [ ] Package agent picks up message from `pkg:sunholo/auth` inbox and executes
- [ ] `Subdirectory` field correctly scopes agent to `packages/auth/` within worktree
- [ ] Class A update completes without human intervention (auto-approve + auto-publish)
- [ ] Class C update triggers full 3-phase chain with approval gates
- [ ] `emitDependentNotifications()` sends to all packages depending on published package
- [ ] `ProvenanceInfo` recorded in registry `metadata.json` for agent-driven publishes
- [ ] AGENT.md accessible as separate blob at `packages/vendor/name/version/AGENT.md`
- [ ] `history.json` uploaded per version with full message trail and action log
- [ ] `IndexEntry` includes `last_updated`, `updated_by`, `latest_summary` for website listing
- [ ] Cascade of 3+ packages completes in dependency order without duplicates
- [ ] Circuit breaker pauses cascade after 3 failures
- [ ] All existing tests passing
- [ ] Documentation updated (CHANGELOG.md, this design doc moved to implemented)

## Testing Strategy

**Unit tests:**
- `autonomy_router_test.go` — verify class A/B/C mapping for all 11 message kinds
- `cascade_scheduler_test.go` — topological sort with diamond deps, cycles (should error)
- `registry_types_test.go` — ProvenanceInfo JSON marshal/unmarshal, backward compat

**Integration tests:**
- Local coordinator with one package agent: send message → verify task creation → verify worktree subdirectory
- Publish with dependent notification: mock registry index → verify messages sent to dependents
- Cascade dedup: send duplicate messages with same correlation ID → verify no re-trigger

**Manual testing:**
- Full local flow: `ailang messages send pkg:sunholo/auth "test" → agent runs → ailang publish`
- Cloud deployment to dev: end-to-end via Pub/Sub
- Provenance query: `ailang pkg provenance sunholo/auth@X.Y.Z` after agent publish

## Deferred Decisions

The following are intentionally left open for the implementer:

- **Which 14 packages get agents first vs phased rollout** — agent may choose based on dependency order (leaf packages first)
- **Exact invoke template wording** — agent may refine based on initial test runs
- **Circuit breaker threshold** — proposed 3 failures, agent may adjust based on testing
- **Provenance display format** — `ailang pkg provenance` output format is flexible

## Non-Goals

**Not attempted in this feature:**
- **Package yanking or deprecation automation** — registry is immutable; deprecation is manual
- **Cross-registry federation** — single registry only
- **Version range resolution** — AILANG uses exact versions only
- **Automatic semantic versioning** — agent determines bump level, not computed from code diffs
- **External package maintainer support** — this is for first-party `sunholo/*` packages in the monorepo

## Graduated Autonomy Summary

| Project | Change Class | Workflow | Approval Pattern |
|---------|-------------|----------|-----------------|
| Packages | A (patch) | Single-step: update, test, publish | `skip_approval: true, auto_merge: true` |
| Packages | B (minor) | Sprint plan → execute | `auto_approve_handoffs: true, auto_merge: false` |
| Packages | C (breaking) | Design doc → sprint → execute | All approval gates active |
| Core ailang | All classes | Design doc → sprint → execute | `auto_approve_handoffs: false` at every stage (already configured in `config.cloud.yaml`) |

## Timeline

**Week 1** (~5 days):
- Phase 1: `Subdirectory` field, single package agent, local MVP
- Phase 2: All 14 agents in cloud config, workspace mapping, deploy to dev

**Week 2** (~5 days):
- Phase 3: Dependent notification, dynamic autonomy router
- Phase 4: Provenance in registry, AGENT.md as first-class artifact, cascade scheduling

**Total: ~10 days across 2 weeks**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Cascade storms (A updates → B,C,D update → re-trigger A) | High | Correlation ID dedup + SupersedeOlderMessages + circuit breaker |
| Monorepo worktree conflicts (14 agents, 1 repo) | Med | `max_concurrent_tasks: 1` per agent + sequential scheduling for same-repo agents |
| `pkg:` format breaks inbox adapter registration | Med | Verify in Phase 1 MVP; colon is valid in SQLite column values |
| Agent publishes broken package (tests pass locally, validator rejects) | Med | Validator rejection → message back to package inbox → agent retries with error |
| Cost runaway (14 agents each running 30-min sessions) | Low | Per-task cost budgets in coordinator config; Class A uses sonnet (cheaper) |

## Related Documents

<!-- Auto-populated by Ollama neural search on "pkg autonomous updates" -->

**Implemented (may inform design):**
- [m-coord-auto-revision.md](../../implemented/v0_7_0/m-coord-auto-revision.md) — Automatic agent revision on GitHub feedback (same approval pattern)
- [m-coordinator-always-on-daemon.md](../../implemented/v0_7_0/m-coordinator-always-on-daemon.md) — Daemon polling architecture (inbox adapter pattern)
- [m-coord-github-auto-routing.md](../../implemented/v0_6_2/m-coord-github-auto-routing.md) — GitHub issue → agent routing
- [m-pkg-registry.md](../../implemented/v0_9_7/m-pkg-registry.md) — Registry architecture (metadata schema, validator)
- [m-pkg-package-system.md](../../implemented/v0_9_5/m-pkg-package-system.md) — Package system design

**Planned (check for overlap):**
- [m-pkg-transitive-lock-fix.md](m-pkg-transitive-lock-fix.md) — Transitive dependency locking
- [m-dx-package-check.md](m-dx-package-check.md) — Package check DX improvements
- [m-pkg-ecosystem-status.md](m-pkg-ecosystem-status.md) — Current ecosystem status audit

## References

- [Design Axioms](/docs/references/axioms) - The 12 non-negotiable principles
- [Agent Messaging Guide](../../../docs/docs/guides/agent-messaging.md) - Messaging system documentation
- [Coordinator Guide](../../../docs/docs/guides/coordinator.md) - Coordinator architecture
- [config.cloud.yaml](https://github.com/sunholo-data/ailang-multivac/blob/dev/config/config.cloud.yaml) - Existing agent configurations
- [internal/messaging/pkg_schema.go](../../../internal/messaging/pkg_schema.go) - 11 package message kinds

## Future Work

- **Package agent skill marketplace**: Publish package-maintenance skills to ailang_bootstrap for community packages
- **Cross-registry cascades**: When multiple registries exist, coordinate updates across them
- **AI-driven version bump**: Analyze code diff to automatically determine patch/minor/major
- **Registry website**: Package detail pages powered by `history.json` — show update timelines, dependency impact graphs, trust indicators (human-reviewed vs autonomous), and AI-generated changelogs
- **Package health dashboard**: Observatory view showing package freshness, update history, provenance chains
- **External maintainer onboarding**: Allow third-party package authors to register their own package agents

---

**Document created**: 2026-03-24
**Last updated**: 2026-03-24
