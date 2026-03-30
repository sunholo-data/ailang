# M-PKG-UPGRADE-CHAIN-TOOLING: Package Upgrade Chain Diagnostics and Automation

**Status**: Planned
**Target**: v0.10.0
**Priority**: P1 (DX pain — cascade republishing is correct but tedious)
**Estimated**: 3 days
**Dependencies**: M-PKG-RESOLVER-DIRECT-WINS (conflict detection — already implemented)
**Origin**: docparse cascade republish experience (2026-03-24)

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Conflict semantics unchanged |
| A2: Replayability | 0 | Lock files still fully reproducible |
| A3: Effect Legibility | 0 | No effect changes |
| A4: Explicit Authority | +1 | Makes upgrade authority chain visible |
| A5: Bounded Verification | +1 | Upgrade plan is verifiable before execution |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +2 | Structured conflict chains are agent-parseable; automation enables agent-driven upgrades |
| A8: Minimal Syntax | 0 | No language changes |
| A9: Cost Visibility | +1 | Shows exact cost of an upgrade (which packages, how many) |
| A10: Composability | +1 | Treats package ecosystems as coordinated upgrade graphs |
| A11: Structured Failure | +1 | Richer conflict diagnostics than bare error |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +7** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No resolution semantics changed
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access — all upgrades explicit
- [x] A7 (Machines First): Structured output for agent consumption

## Design Philosophy

### Why Not "Direct Deps Override Transitive"

The original design doc (M-PKG-RESOLVER-DIRECT-WINS) proposed pre-seeding `resolvedSet` with root direct deps so transitive conflicts are silently overridden. **This was rejected.**

**Rationale:** If `billing_store@0.5.0` was published against `firestore@0.1.0`, forcing the graph to use `firestore@0.2.0` because the root asked for it changes the dependency contract of a published package without republishing it. That weakens package identity.

A published package version means:
- These are the dependencies it was resolved/tested/published against
- These hashes/interfaces are what it composes with
- If you want different dependency versions, publish a new package version

If the root can override at lock time, published packages are no longer closed, reproducible units. They become "published code plus consumer-selected dependency substitutions" — a much looser model that cuts against AILANG's explicitness and reproducibility.

### The Right Fix

The cascade republishing is **semantically correct**. The system is forcing compatibility work to happen explicitly. The pain is real, but override is the wrong fix.

The right response is not more magical resolution. The right response is **better upgrade workflow tooling** that makes the deliberate, visible republishing easy and automatable.

### Two Modes

| Context | Policy | Rationale |
|---------|--------|-----------|
| Published registry packages | **Strict conflict** — no override, no substitution | Package identity is inviolable |
| Local workspace/path deps | **Consider root-biased unification** (future work) | Packages are being edited together, not consuming immutable artifacts |

This design doc covers tooling for the strict registry case. Workspace-mode flexibility is deferred.

## Problem Statement

After implementing conflict detection (commit `1cd08456`), the resolver correctly fails when transitive deps disagree with root direct deps. But the user must manually:

1. Read the error to understand the conflict chain
2. Figure out which packages need republishing
3. Determine the correct republish order (leaves first)
4. Bump versions, republish each package
5. Update root manifest with new versions
6. Re-lock

For deep dependency chains (docparse → access_gate → billing_store → firestore → gcp_auth), bumping a leaf dep requires 4+ manual republish steps.

## Goals

**Primary:** Make cascade republishing deliberate, visible, and easy — without changing resolution semantics.

**Success Metrics:**
- `ailang upgrade <pkg>@<version>` is the primary UX for adopting new dependency versions
- `ailang lock --explain` shows full conflict chain with first-resolver provenance
- Upgrade plan computed as a graph (nodes + edges), linearized for CLI, structured for agents
- Interface hash comparison prunes unnecessary republishes
- All commands emit structured, agent-parseable output
- Resolution semantics unchanged (fail on conflict, no override)

## Solution Design

### Primary UX: `ailang upgrade <pkg>@<version>`

The main entry point. Internally computes an upgrade plan, shows the diff, optionally executes the chain.

```
$ ailang upgrade sunholo/firestore@0.2.0

upgrade plan for sunholo/firestore@0.2.0

  compatibility check:
    sunholo/billing_store@0.5.0
      dependency change: firestore 0.1.0 → 0.2.0
      interface impact: none (same interface hash)
      → republish required (dependency version changed)

    sunholo/docparse_access_gate@0.4.0
      dependency change: billing_store 0.5.0 → 0.6.0
      interface impact: none
      → republish required (dependency version changed)

  republish order (2 packages):
    1. sunholo/billing_store@0.5.0 → 0.6.0
       bump firestore dep to 0.2.0

    2. sunholo/docparse_access_gate@0.4.0 → 0.5.0
       bump billing_store dep to 0.6.0

  after chain:
    update root ailang.toml:
      "sunholo/billing_store" = "0.6.0"
      "sunholo/docparse_access_gate" = "0.5.0"
      "sunholo/firestore" = "0.2.0"

  execute? [y/N/dry-run]
```

**Flags:**
- `--dry-run` — show plan, don't execute
- `--yes` — skip confirmation prompts
- `--json` — emit graph as JSON (for agents)
- `--patch` / `--minor` / `--major` — version bump policy (default: `--patch`)
- `--skip-compatible` — skip republish for packages where interface hash is unchanged (future, once interface hash comparison is reliable)

### Internal Model: Upgrade Graph

The upgrade plan is a **directed acyclic graph**, not a flat list. The CLI linearizes it; agents consume the graph.

**Internal representation:**

```go
type UpgradeGraph struct {
    Target  string            // "sunholo/firestore@0.2.0"
    Nodes   []UpgradeNode
    Edges   []UpgradeEdge     // [from, to] — republish order
}

type UpgradeNode struct {
    Package         string    // "sunholo/billing_store"
    FromVersion     string    // "0.5.0"
    ToVersion       string    // "0.6.0"
    DependsOn       []string  // packages this node depends on (within the upgrade)
    InterfaceChange bool      // true if interface hash differs
    DepChanges      []DepChange // which deps are being bumped
}

type UpgradeEdge struct {
    From string // package that changed
    To   string // package that must be republished because of it
}

type DepChange struct {
    Package     string
    FromVersion string
    ToVersion   string
}
```

**JSON output (`--json`):**

```json
{
  "target": "sunholo/firestore@0.2.0",
  "nodes": [
    {
      "package": "sunholo/billing_store",
      "from": "0.5.0",
      "to": "0.6.0",
      "depends_on": ["sunholo/firestore"],
      "interface_change": false,
      "dep_changes": [
        {"package": "sunholo/firestore", "from": "0.1.0", "to": "0.2.0"}
      ]
    },
    {
      "package": "sunholo/docparse_access_gate",
      "from": "0.4.0",
      "to": "0.5.0",
      "depends_on": ["sunholo/billing_store"],
      "interface_change": false,
      "dep_changes": [
        {"package": "sunholo/billing_store", "from": "0.5.0", "to": "0.6.0"}
      ]
    }
  ],
  "edges": [
    ["sunholo/firestore", "sunholo/billing_store"],
    ["sunholo/billing_store", "sunholo/docparse_access_gate"]
  ]
}
```

### Version Bump Policy

Explicit rules to prevent inconsistent bumps across tools/agents:

| Condition | Default Bump | Rationale |
|-----------|-------------|-----------|
| Only dependency versions changed, no source changes | Patch | Minimal version increment |
| Source code also changed | Minor | Signals functional change |
| Breaking interface change (different interface hash) | Minor (warn) | Consumer must adapt |
| User override | `--patch`, `--minor`, `--major` flags | Explicit control |

**Rules:**
- Default is `--patch` (bump patch version of each affected package)
- `--minor` bumps minor version instead
- `--major` bumps major version (for breaking changes)
- If interface hash changes, warn even with `--patch` — the user may want `--minor`

### Compatibility Check (No-Op Detection)

Before republishing each package, check whether the upgrade actually requires a new version:

```
Step 0: compatibility check

sunholo/billing_store
  dependency change: firestore 0.1.0 → 0.2.0
  interface impact: none (same interface hash)
  → republish required (dependency version changed in manifest)

sunholo/http_helpers
  dependency change: gcp_auth 0.7.0 → 0.8.0
  interface impact: none
  source impact: none (no code uses changed APIs)
  → republish optional (skip with --skip-compatible)
```

**How it works:**
1. Download both versions of the changed dependency
2. Compare `InterfaceHash()` between old and new
3. If interface hash is identical, the upgrade is API-compatible
4. Still republish by default (dependency contract changed), but flag as skippable

**Note:** `--skip-compatible` is opt-in and should be used cautiously. Even if the interface hash is the same, behavior may have changed. This is an optimization for experienced users, not the default.

### Enhanced Conflict Diagnostics: `ailang lock --explain`

Shows the full dependency traversal that led to a conflict, including **first-resolver provenance** — which path resolved the package first.

**Enhanced output:**

```
version conflict: sunholo/firestore
  root requires: 0.2.0
  already resolved: 0.1.0

  first resolved by:
    sunholo/billing_store@0.5.0
      → sunholo/firestore@0.1.0

  conflict chain:
    sunholo/docparse_access_gate@0.4.0
      → sunholo/billing_store@0.5.0
        → sunholo/firestore@0.1.0  ← pins old version

  root direct dep:
    sunholo/firestore@0.2.0

  to resolve: republish the chain from leaf to root
    1. sunholo/billing_store (currently pins firestore@0.1.0)
    2. sunholo/docparse_access_gate (currently pins billing_store@0.5.0)

  run: ailang upgrade sunholo/firestore@0.2.0
```

**Implementation:**
- `internal/pkg/resolver.go`: Add `resolvedBy map[string]string` (name → "resolved by package") to track which path first resolved each package
- `VersionConflictError`: Add `Chain []ChainEntry` and `FirstResolvedBy string` fields
- `cmd/ailang/lock.go`: Add `--explain` flag, format chain output with provenance

### Execution: `ailang upgrade --execute`

When the user confirms the plan (or passes `--yes`), execute the cascade:

**Requirements:**
- All affected packages must be available locally (path or workspace). **Fail fast** if any package is not locally available — do not attempt partial automation.
- Interactive confirmation before each publish step (unless `--yes`)
- Dry-run mode shows plan without side effects

**Failure semantics:**

| Mode | Behavior |
|------|----------|
| **Safe (default)** | Stop on first publish failure. Emit recovery instructions listing completed steps and remaining steps. Do NOT attempt to unpublish — that breaks immutability. |
| **Continue** (`--continue-on-error`) | Log failure, continue to next independent package (if no dependency on failed step). Mark graph as incomplete. |

**No rollback of published packages.** Once published, a version is permanent. If step 2 fails after step 1 published, the instructions tell the user: "step 1 complete (billing_store@0.6.0 published), resume from step 2."

### Package Messaging Integration

After a successful upgrade chain, send structured messages to affected package inboxes:

```json
{
  "type": "dependency_upgraded",
  "package": "sunholo/billing_store",
  "old_version": "0.5.0",
  "new_version": "0.6.0",
  "reason": "cascade upgrade of sunholo/firestore@0.1.0 → 0.2.0",
  "upgraded_by": "sunholo/docparse",
  "timestamp": "2026-03-24T17:00:00Z"
}
```

This enables:
- Audit trail from root upgrade → cascade → individual publishes
- Agent-driven upgrade workflows (detect upgrade, plan, execute, verify)
- Notifications to downstream consumers
- Foundation for coordinator-driven autonomous upgrades

## Coordinator Integration (Future)

This design gives us: dependency graph + upgrade plan + execution steps + messaging. That is one step from **coordinator-driven upgrades**:

1. New version published (e.g., `gcp_auth@0.8.0`)
2. Coordinator computes `ailang upgrade --json` for affected consumers
3. Tasks distributed to package owners/agents
4. Each agent executes its step in the upgrade graph
5. Verification gates applied (tests, interface hash)
6. Chain completion tracked via messaging

This doc is therefore an M-COORDINATOR dependency, not just CLI DX.

## Files to Modify / Create

| File | Change | LOC |
|------|--------|-----|
| `internal/pkg/upgrade.go` | New: `UpgradeGraph`, `ComputeUpgradePlan()`, graph construction, topo sort | ~200 |
| `internal/pkg/resolver.go` | Add `resolvedBy` tracking, extend `VersionConflictError` with chain + provenance | ~50 |
| `cmd/ailang/upgrade.go` | New: `ailang upgrade` CLI command, plan display, execution orchestration | ~200 |
| `cmd/ailang/lock.go` | Add `--explain` flag, format enhanced conflict output | ~40 |
| `internal/pkg/upgrade_test.go` | New: tests for graph construction, topo sort, compatibility check, version bump | ~150 |
| `cmd/ailang/main.go` | Wire `upgrade` subcommand | ~5 |

## Implementation Plan

### Phase 1: Upgrade graph model + `--explain` — ~3 hours
- [ ] Define `UpgradeGraph`, `UpgradeNode`, `UpgradeEdge` types
- [ ] Add `resolvedBy` tracking to resolver
- [ ] Extend `VersionConflictError` with `Chain` and `FirstResolvedBy`
- [ ] Add `--explain` flag to `ailang lock`
- [ ] Tests for chain reconstruction and first-resolver provenance

### Phase 2: `ailang upgrade` plan computation — ~3 hours
- [ ] `ComputeUpgradePlan()`: build reverse dep graph from lock file
- [ ] Topological sort of affected packages (leaves first)
- [ ] Interface hash comparison for no-op detection
- [ ] Version bump logic with `--patch`/`--minor`/`--major`
- [ ] CLI command with human-readable and `--json` output
- [ ] Tests for upgrade plan computation

### Phase 3: `ailang upgrade` execution — ~3 hours
- [ ] Fail-fast check: all affected packages available locally
- [ ] Manifest modification (version bump + dep bump) via existing write
- [ ] Orchestrated publish in topological order
- [ ] Safe-mode failure handling (stop + recovery instructions)
- [ ] Package messaging integration
- [ ] Tests for execution + failure recovery

### Phase 4: Edge cases + docs — ~1.5 hours
- [ ] Multiple conflicting packages in one upgrade
- [ ] Packages not available locally (clear error, fail fast)
- [ ] `--json` output for agent consumption
- [ ] CHANGELOG update
- [ ] CLI help text

## Effort Estimate

| Phase | LOC | Time |
|-------|-----|------|
| P1: Graph model + `--explain` | 100 | 3 hours |
| P2: Upgrade plan computation | 250 | 3 hours |
| P3: Upgrade execution | 200 | 3 hours |
| P4: Edge cases + docs | 100 | 1.5 hours |
| **Total** | **~650** | **~10.5 hours (3 days)** |

**Risk level:** Low-Medium — no resolution semantics changed; new commands are additive.

## Success Criteria

- [ ] `ailang upgrade <pkg>@<version>` computes correct upgrade graph
- [ ] `ailang upgrade --json` emits graph with nodes + edges for agents
- [ ] `ailang upgrade --yes` executes cascade with correct topological order
- [ ] `ailang lock --explain` shows first-resolver provenance + full conflict chain
- [ ] Interface hash comparison flags no-op republishes
- [ ] Version bump policy is explicit and consistent (`--patch`/`--minor`/`--major`)
- [ ] Failure stops cleanly with recovery instructions (no rollback illusion)
- [ ] All packages must be locally available (fail fast, no partial automation)
- [ ] Resolution semantics unchanged — conflicts still fail, no override
- [ ] Existing resolver tests unaffected

## Non-Goals

- Root deps overriding transitive deps (rejected — weakens package identity)
- `ailang lock --force` to bypass conflicts (rejected — "override with a scarier flag")
- Semver range resolution (AILANG uses exact versions)
- Multiple versions of same package (AILANG stays flat)
- Workspace-mode root-biased unification (future work, separate design doc)
- Unpublishing packages on failure (breaks immutability)
- Automatic conflict resolution without user confirmation

## Future Work

- **Workspace mode**: For local path deps being edited together, consider more flexible root-biased resolution. Separate design doc.
- **CI integration**: `ailang upgrade` as a GitHub Action triggered by leaf package publish
- **`--skip-compatible`**: Skip republish when interface hash unchanged (opt-in, for experienced users)
- **Coordinator-driven upgrades**: Coordinator detects version bumps, computes upgrade graphs, distributes tasks to agents, tracks completion via messaging
- **Multi-package upgrades**: `ailang upgrade sunholo/firestore@0.2.0 sunholo/gcp_auth@0.8.0` — plan for multiple leaf upgrades simultaneously

## Related Documents

- [m-pkg-resolver-direct-wins.md](../../implemented/v0_9_11/m-pkg-resolver-direct-wins.md) — Conflict detection (implemented v0.9.11)
- [m-pkg-transitive-lock-fix.md](m-pkg-transitive-lock-fix.md) — Transitive registry dep resolution
- [m-pkg-autonomous-updates.md](m-pkg-autonomous-updates.md) — Autonomous package update workflows

---

**Document created**: 2026-03-24
**Last updated**: 2026-03-24
