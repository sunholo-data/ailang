# M-PKG-MSG: Package Messaging Graph for Multi-Agent Coordination

**Status**: Planned
**Target**: v1.1.0+
**Priority**: P0 — required for package-scale parallel AI coordination
**Estimated**: 2–3 weeks (core graph model + message schema + coordinator hooks)
**Dependencies**:
- Package system (implemented — manifests, lock files, path/git/registry sources)
- Cross-project messaging system (implemented — package-scoped inboxes, semantic search, GitHub sync)
- Coordinator daemon (planned / partial)
- Package hashes (content + interface)
- Effect ceilings (implemented)

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Package identity remains hash-based; message payloads reference exact versions/hashes |
| A2: Replayability | +1 | Coordination state is stored as structured messages with explicit package refs |
| A3: Effect Legibility | +1 | Effect deltas become first-class message fields for upgrade and risk triage |
| A4: Explicit Authority | +1 | Authority widening is promoted from silent metadata change to explicit coordination event |
| A5: Bounded Verification | +1 | Verification remains package-local; messages trigger only impacted dependents |
| A6: Safe Concurrency | +2 | Package nodes + typed coordination edges provide schedulable parallel work units |
| A7: Machines First | +2 | Messages use structured schemas, not free-form prose, for coordinator and agent consumption |
| A8: Minimal Syntax | 0 | No new language syntax required; this is tooling / protocol level |
| A9: Cost Visibility | +1 | Impacted dependency cones and re-verification scope become explicit in messages |
| A10: Composability | +1 | Static package graph and dynamic coordination graph compose cleanly |
| A11: Structured Failure | +2 | Upgrade failures, contract regressions, and migration blocks become typed workflow states |
| A12: System Boundary | +1 | Package inboxes are explicit system boundaries for operational coordination |

**Net Score: +13** → **Decision: Strong fit**

### Hard Violation Check

- [x] A1 (Determinism): Messages reference immutable package identities, not mutable labels
- [x] A3 (Effects): Effect changes are explicit fields, not inferred from prose
- [x] A4 (Authority): Authority widening requires explicit message flow
- [x] A7 (Machines First): JSON-first coordination schema

---

## Problem Statement

The package system solves the static side of scale:
- package identity
- exported interfaces
- effect ceilings
- dependency graphs
- lockfile reproducibility
- interface/content hash change detection

But large-scale multi-agent development also needs a dynamic coordination layer:
- package owners must announce changes
- downstream consumers must report breakage or validate compatibility
- coordinators must triage and assign upgrade work
- duplicate reports should be clustered
- migration state should be visible without rereading code or issue threads

### Current Gap

Without a structured coordination layer, package changes degrade into:
- ad hoc GitHub issues
- free-form chat messages
- manual repo inspection
- duplicated compatibility reports
- no canonical upgrade negotiation flow

This becomes a scaling bottleneck when many agents or teams operate on many packages in parallel.

### Core Insight

The package system defines the **nodes** of the ecosystem.
The messaging system defines the **edges** of the coordination graph.

Together they form a **package messaging graph**:
- **Nodes** = packages, versions, interfaces, maintainers, workspaces
- **Edges** = upgrade notices, compatibility reports, migration requests, acknowledgements, deprecations

This allows AILANG to move from "many agents editing code" to "many agents operating on a structured coordination graph."

---

## Design Goals

| Goal | Axiom | Description |
|------|-------|-------------|
| G1 — Preserve Static Truth | A1 | Package metadata remains the source of semantic truth |
| G2 — Add Dynamic Coordination | A6 | Messaging carries workflow around semantic changes |
| G3 — Typed Package Workflows | A7 | Package-related messages are structured and machine-readable |
| G4 — Impact-Aware Triage | A9 | Coordinators can identify who is affected and what must be re-verified |
| G5 — Support Parallel Upgrades | A6 | Independent package upgrade tasks can be scheduled concurrently |
| G6 — Minimize Human Arbitration | A11 | Common upgrade and compatibility flows should be automatable |
| G7 — Preserve Bounded Reasoning | A5 | Messages should scope work to affected package cones, not whole repos |

---

## Non-Goals (v1)

- Messaging is **not** the source of semantic truth about compatibility
- Free-form chat is **not** sufficient for package coordination
- No distributed consensus protocol or CRDT-style merge system
- No package graph mutation via messages alone
- No "approval by prose" replacing interface/effect/hash checks
- No requirement that every coordination event sync to GitHub

---

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Message schema version (`ailang.package-message/v1`) | Baked into all coordination tooling; changing later requires migration | agent | design | high |
| Inbox addressing scheme (`pkg:vendor/name`) | Affects routing, CLI, and all downstream tooling | human | design | high |
| Where package message types live (`internal/messaging/` vs new `internal/coordpkg/`) | Determines coupling between messaging and package systems | agent | design | med |
| Which package events auto-emit messages | Determines noise level and whether messaging is opt-in or default | human | design | med |
| Compatibility report format (structured JSON vs freeform + metadata) | Affects whether reports are machine-actionable or human-readable-only | agent | design | med |

### Design Freeze

Before implementation begins, these must be resolved:

- [ ] Inbox addressing: `pkg:<vendor/name>` for packages, `workspace:<name>` for workspaces
- [ ] Message schema version: `ailang.package-message/v1`
- [ ] Package message types extend existing `internal/messaging/` (not separate package)
- [ ] Auto-emission on publish events is opt-in initially, default-on in Phase 2
- [ ] Compatibility reports use structured JSON with required evidence fields

---

## Core Model

### Static Graph: Package Dependency Graph

Defined by package manifests and lockfiles.

**Nodes:**
- package
- package version
- content hash
- interface hash
- effect ceiling

**Edges:**
- declared dependency
- dependent relationship

This graph answers:
- what exists
- what depends on what
- what authority each package has
- whether a change is internal or externally visible

### Dynamic Graph: Package Messaging Graph

Defined by structured package-scoped messages.

**Nodes:**
- package inbox
- workspace inbox
- maintainer team inbox
- upgrade task
- migration state

**Edges:**
- change announcement
- compatibility request
- validation response
- regression report
- deprecation notice
- migration completion

This graph answers:
- who needs to act
- who has acknowledged
- who is blocked
- what the current operational state is

### Design Principle

> Package metadata determines whether change is semantically valid.
> Messaging coordinates how affected actors respond to that change.

---

## Package Coordination Topology

### Node Types

#### 1. Package Node

Represents a package identity such as `sunholo/auth`.

Fields:
- name
- current version
- current content hash
- current interface hash
- effect ceiling
- stability
- maintainers / inbox

#### 2. Package Version Node

Represents a specific released artifact.

Fields:
- version
- source kind (path, git, registry)
- content hash
- interface hash
- published timestamp
- change class (optional derived metadata)

#### 3. Workspace Node

Represents a consuming project or monorepo.

Fields:
- workspace name
- lockfile snapshot
- dependency set
- package inbox
- owner team

#### 4. Upgrade Task Node

Represents a scoped action to move one workspace/package from one dependency state to another.

Fields:
- target package
- current version/hash
- desired version/hash
- change class
- status
- assigned agent / team

#### 5. Coordination Inbox Node

Operational endpoint for messages.

Examples:
- `pkg:sunholo/auth`
- `pkg:sunholo/http-helpers`
- `workspace:docparse`
- `team:registry-admin`

---

## Edge Types (Message Kinds)

### 1. `upgrade-available`
A package maintainer announces a new version or release candidate.

### 2. `interface-change-notice`
A package interface hash changed and downstreams may need action.

### 3. `effect-widening-warning`
A package widened authority and downstream policy/verification may block adoption.

### 4. `compatibility-request`
A maintainer requests validation from downstream consumers for a specific update.

### 5. `compatibility-report`
A downstream consumer reports pass/fail/partial compatibility with structured evidence.

### 6. `contract-regression`
A downstream or validator reports that a previously assumed contract no longer holds.

### 7. `migration-request`
A downstream asks for guidance or support in moving to a new version.

### 8. `deprecation-notice`
A maintainer announces intent to remove or replace an API.

### 9. `upgrade-complete`
A downstream signals successful adoption.

### 10. `blocked`
A migration or upgrade is blocked on a specific issue.

### 11. `superseded`
A previous coordination thread/message is obsolete due to a newer release or resolution.

---

## Message Schema

Package coordination messages must be structured.

### Canonical Envelope

```json
{
  "schema": "ailang.package-message/v1",
  "message_id": "msg_123",
  "kind": "upgrade-available",
  "from": "pkg:sunholo/auth",
  "to": ["workspace:docparse", "workspace:web-api-demo"],
  "timestamp": "2026-03-21T12:00:00Z",
  "package": {
    "name": "sunholo/auth",
    "from_version": "0.1.0",
    "to_version": "0.2.0",
    "from_interface_hash": "sha256:aaa",
    "to_interface_hash": "sha256:bbb",
    "from_content_hash": "sha256:ccc",
    "to_content_hash": "sha256:ddd",
    "change_class": "C",
    "effect_delta": [],
    "breaking": false
  },
  "summary": "Bearer extraction normalized; key validation contract tightened",
  "recommended_action": "Run auth package compatibility checks before upgrade",
  "refs": {
    "package_url": "pkg:sunholo/auth@0.2.0",
    "release_notes": "optional",
    "lockfile_ref": "optional"
  },
  "status": "open"
}
```

### Required Fields

- `schema`
- `kind`
- `from`
- `to`
- `timestamp`
- `package.name`
- at least one of: version delta, interface hash delta, content hash delta

### Optional Fields

- `change_class`
- `effect_delta`
- `breaking`
- `summary`
- `recommended_action`
- `refs`
- `status`
- `supersedes`
- `related_messages`

---

## Message Kinds — Required Payloads

### `upgrade-available`

**Required:** package name, from/to version, from/to interface hash, change class
**Optional:** release notes summary, recommended upgrade strategy

### `compatibility-report`

**Required:** package name, tested from/to versions, target workspace, result (`pass` | `fail` | `partial`), evidence summary
**Optional:** failing exports, contract violations, lockfile snapshot ref

### `effect-widening-warning`

**Required:** package name, previous effect ceiling, new effect ceiling, affected exports or package summary

### `contract-regression`

**Required:** package name, affected export(s), previous contract summary, new contract summary or failure mode

### `migration-request`

**Required:** source workspace/package, target package, current version, desired version, block reason

---

## Coordinator Responsibilities

The coordinator consumes the package messaging graph and turns it into work.

### Required Behaviors

1. **Watch package inboxes** — Observe package-scoped channels for structured events
2. **Correlate static and dynamic state** — Link messages to package metadata, lockfile state, dependency cone, interface hash deltas
3. **Classify actionability** — Determine whether a message implies: no action, local verification only, downstream migration task, escalation to maintainer, policy block
4. **De-duplicate reports** — Use semantic search + package refs to cluster duplicate compatibility failures
5. **Spawn upgrade tasks** — Create package-scoped or workspace-scoped tasks for agents
6. **Track lifecycle** — Update state: open → acknowledged → in-progress → blocked → completed → superseded

### Non-Responsibilities

The coordinator does **not** decide semantic truth by reading message prose. It must always confirm against package metadata, hashes, and lockfile state.

---

## Interaction with Package System

### Package System Remains Source of Truth For

- exported APIs
- effect ceilings
- dependency graph
- version identity
- content hash
- interface hash
- registry metadata

### Messaging Adds

- awareness
- negotiation
- triage
- acknowledgement
- migration workflow
- social/operational state

### Invariant

> A message may announce or report a change, but only package metadata and lockfile state determine the actual semantic meaning of that change.

---

## Change Propagation Workflows

### Case A — Internal-Only Release

1. Maintainer publishes new version
2. Content hash changes, interface hash unchanged
3. Coordinator emits optional `upgrade-available`
4. Downstreams may auto-upgrade if policy allows
5. No compatibility request required

### Case B — Additive API Change

1. New export or module added
2. Interface hash changes
3. `interface-change-notice` sent
4. Existing downstreams not blocked
5. Optional adoption tasks created

### Case C — Contract Change

1. Exported contract changes
2. Interface hash changes
3. `upgrade-available` + `compatibility-request` sent
4. Downstreams run targeted verification/tests
5. `compatibility-report` replies collected
6. Coordinator marks complete or blocked per workspace

### Case D — Effect Widening

1. Package effect ceiling widens
2. `effect-widening-warning` sent
3. Coordinator checks dependency policies
4. Impacted downstreams either: reject automatically, queue review/migration, approve with policy override

---

## Message Lifecycle

### States

- `open`
- `acknowledged`
- `in_progress`
- `blocked`
- `completed`
- `rejected`
- `superseded`

### Rules

- Only one active `upgrade-available` per package/version pair
- Newer release messages may supersede older unresolved notices
- `compatibility-report` messages are append-only records
- Completion is scoped to recipient workspace/package, not global by default

---

## Package-Scoped Inboxes

Package-scoped inboxes already exist (via `to_inbox` field in messaging system) and should be formalized as first-class coordination endpoints.

### Addressing Scheme

| Form | Example |
|------|---------|
| `pkg:<vendor/name>` | `pkg:sunholo/auth` |
| `workspace:<name>` | `workspace:docparse` |
| `team:<name>` | `team:package-admin` |

### Routing Rules

- Release-originated notices originate from `pkg:<name>`
- Downstream adoption results originate from `workspace:<name>`
- Policy or review decisions may originate from `team:<name>`

---

## Automation Triggers

The messaging graph should be driven by package events where possible.

### Trigger Sources

| Trigger | Emits |
|---------|-------|
| Publish event | `upgrade-available`, `interface-change-notice` (if interface hash changed), `effect-widening-warning` (if authority widened) |
| Lockfile diff | Local upgrade task, `migration-request` (if update blocked) |
| Verification failure | `compatibility-report`, optionally `contract-regression` |
| Deprecation declaration | `deprecation-notice` |

**Design Principle:** Messages should be generated from concrete package events, not manual prose, whenever possible.

---

## CLI Surface

### Proposed Commands

```bash
ailang msg send --to pkg:sunholo/auth --type compatibility-report --json payload.json
ailang msg inbox --channel pkg:sunholo/auth
ailang msg search --channel pkg:sunholo/auth "contract regression"
ailang pkg notify-upgrade sunholo/auth@0.2.0
ailang pkg affected-by sunholo/auth@0.2.0
ailang pkg upgrade-plan workspace:docparse
```

### Future Commands

```bash
ailang coord watch --packages
ailang coord triage --channel pkg:sunholo/auth
ailang coord graph --workspace docparse
ailang coord resolve --message msg_123
```

---

## Integration with Registry

When a package is published via the registry, the registry may emit package messages automatically.

### Registry-to-Messaging Hooks

| Registry Event | Message Kind |
|----------------|-------------|
| Publish success | `upgrade-available` |
| Interface hash delta | `interface-change-notice` |
| Effect delta | `effect-widening-warning` |
| Deprecation flag | `deprecation-notice` |

This makes the registry the source of package release events, while messaging becomes the carrier of operational coordination.

---

## Integration with Git Dependencies

For git-based package sources, package messages should still use **package identity**, not raw git URLs, as the primary key.

Required refs may include: git URL, tag, resolved rev, package path within repo.

But message routing should remain package-centric (`pkg:sunholo/auth`, not `git:https://github.com/...`). This preserves stable coordination endpoints even if hosting changes.

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Messaging becomes a second source of truth | High | Enforce invariant: package metadata + lockfile remain authoritative |
| Too much free-form prose in messages | High | Require structured JSON schema for package workflow events |
| Message volume becomes noisy | Medium | Package-scoped inboxes + semantic dedupe + typed kinds |
| Coordinators act on stale package state | Medium | Require hash/version refs in messages and re-check metadata before acting |
| Humans bypass typed message flows | Medium | CLI helpers and auto-generated package events |
| Workflow complexity overwhelms small projects | Low | Make automation optional; manual use remains possible |

---

## Success Criteria

### Phase 1

- [ ] Canonical package message schema defined
- [ ] Package message kinds standardized
- [ ] Package-scoped inbox routing formalized
- [ ] Publish/upgrade events can emit structured messages
- [ ] Coordinator can correlate message → package → impacted workspaces

### Phase 2

- [ ] Compatibility reports deduplicated via semantic search + package refs
- [ ] Upgrade tasks automatically spawned from package events
- [ ] Status lifecycle visible per package/workspace pair
- [ ] Lockfile diff can produce upgrade plan messages

### Phase 3

- [ ] Fleet-wide package adoption dashboard
- [ ] Automatic migration campaigns across workspaces
- [ ] Policy-aware blocking of unsafe upgrades
- [ ] Package trust overlays from compatibility data

---

## Implementation Plan

### Files to Create

| File | Purpose |
|------|---------|
| `internal/messaging/pkg_schema.go` | Typed package message schema and validation |
| `internal/messaging/pkg_events.go` | Maps package system events to message emission |
| `internal/messaging/pkg_triage.go` | Correlates incoming messages with package graph and lockfile state |
| `internal/messaging/pkg_status.go` | Tracks per-recipient workflow state |

### Files to Modify

| File | Change |
|------|--------|
| `cmd/ailang/messages_send.go` | Support `--type` flag for structured package message helpers |
| `cmd/ailang/pkg_*.go` | Add `notify-upgrade`, `affected-by`, `upgrade-plan` subcommands |
| `internal/packages/publish.go` | Emit coordination events on publish |
| `internal/packages/install.go` | Emit events on lockfile changes |
| `internal/coordinator/` | Watch package channels and spawn upgrade tasks |

### Deferred Decisions

- Whether `pkg:` prefix inboxes need separate SQLite table or reuse existing `inbox_messages`
- Whether `compatibility-report` evidence should include full test output or just summary
- Whether Phase 3 dashboard is part of Collaboration Hub or standalone

---

## Example Flow: Upgrade Notice

`sunholo/auth` publishes 0.2.0 with a contract-tightening change.

1. Registry/package tool computes: old 0.1.0 → new 0.2.0, interface hash changed, change class C
2. Tool emits `upgrade-available` message to `workspace:docparse`
3. Coordinator inspects `workspace:docparse`
4. Finds dependency cone includes `sunholo/auth`
5. Creates upgrade task
6. Agent runs compatibility checks
7. Agent sends `compatibility-report` (pass/fail/partial)
8. State becomes `completed` or `blocked`

---

## Related Documents

- [M-PKG: AILANG Package System & Multi-Agent Coordination](m-pkg-package-system.md)
- [M-PKG Registry](m-pkg-registry.md)
- [Cross-project messaging guide](/docs/docs/guides/agent-messaging.md)
- [Coordinator daemon design](m-agent-orchestration.md)

---

## Key Principle

> **Packages are the static units of composition. Messages are the dynamic units of coordination.**
>
> Packages answer: what exists, what changed, what authority it has, what can compose.
> Messages answer: who is affected, who has acknowledged, what is blocked, what should happen next.
>
> That separation is what makes large-scale parallel AI coding tractable.
