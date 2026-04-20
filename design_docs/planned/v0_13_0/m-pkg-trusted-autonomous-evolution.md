# M-PKG-TRUSTED-AUTONOMOUS-EVOLUTION: Secure Autonomous Package Evolution

**Status**: Planned
**Target**: v0.11.0
**Priority**: P0 (Foundational — correctness + security + scalability)
**Estimated**: 4 weeks (phased across v0.10.x–v0.11.0)
**Dependencies**: M-PKG-UPGRADE-CHAIN-TOOLING, M-PKG-AUTONOMOUS-UPDATES, M-COORDINATOR, Package messaging system

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | All upgrades are deterministic DAG transformations; no implicit version drift |
| A2: Replayability | +1 | Upgrade plans are structured graphs; full provenance chain enables replay |
| A3: Effect Legibility | +1 | No install-time code execution; all effects explicit in package signatures |
| A4: Explicit Authority | +2 | Signed publishing, admission policy, effect ceilings — no ambient authority at any layer |
| A5: Bounded Verification | +1 | Interface hash + effect delta enable local compatibility verification |
| A6: Safe Concurrency | 0 | No concurrency model changes |
| A7: Machines First | +2 | Structured output everywhere; upgrade graphs, classification, policy — all agent-parseable |
| A8: Minimal Syntax | 0 | No language syntax changes |
| A9: Cost Visibility | +1 | Upgrade plans show exact cost (which packages, how many republishes) |
| A10: Composability | +1 | Composes with existing messaging, coordinator, and registry systems |
| A11: Structured Failure | +1 | Explicit failure states (BLOCKED, FAILED, PARTIAL) — no rollback illusion |
| A12: System Boundary | +1 | Registry publish is explicit boundary; provenance tracks all crossings |

**Net Score: +12** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): Upgrades are explicit graph transformations — no implicit nondeterminism
- [x] A3 (Effects): No install-time execution; all effects declared in package signatures
- [x] A4 (Authority): Signed publishing + admission policy — no ambient access
- [x] A7 (Machines First): All outputs structured for agent consumption

## Problem Statement

### Current State

AILANG now has:
- Deterministic package resolution (exact versions)
- Strict conflict detection (no silent override)
- Upgrade tooling planned (M-PKG-UPGRADE-CHAIN-TOOLING: `upgrade-plan`, `republish-chain`)
- Message-driven autonomous updates planned (M-PKG-AUTONOMOUS-UPDATES)

However:
- Upgrades require manual cascade republishing
- Ecosystem evolution does not scale beyond ~20 packages
- Supply-chain risks (e.g., LiteLLM-style attacks) are not addressed at the package layer
- No formal trust/admission model exists
- No signed provenance for published artifacts

### External Trigger: LiteLLM Incident

Recent supply-chain attack characteristics:
- Compromised publisher credentials
- Malicious package uploaded under trusted name
- Install-time execution (`.pth`) triggered automatically
- Secret exfiltration (env vars, SSH keys, tokens)

**Key lesson:** Resolution correctness is not enough — trust, provenance, and execution constraints must be enforced.

**Impact:**
- Any AILANG package ecosystem user is vulnerable to the same class of attack
- Without signed provenance, a compromised registry or publisher key enables silent substitution
- Without admission policy, consuming projects have no defense-in-depth

## Goals

**Primary Goal:** Enable a package ecosystem that is self-maintaining, deterministic, secure by construction, policy-controlled, and agent-operable.

**Success Metrics:**
- No silent dependency substitution possible at any layer
- All upgrades traceable via structured provenance chain
- Malicious package cannot execute at install time (by construction)
- Untrusted publisher blocked before artifact contents are admitted into the dependency graph
- Upgrade chains computable and automatable by agents
- System converges to consistent state autonomously under policy constraints

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Signing key format and storage | Determines publisher identity infrastructure | human | design | high |
| Policy file format (`ailang-policy.toml`) | Shapes every consumer's trust configuration; hard to change once adopted | human | design | high |
| Classification taxonomy (SAFE/ADDITIVE/BREAKING/AUTHORITY/UNKNOWN) | Determines what auto-upgrades vs blocks; wrong taxonomy = either too permissive or too restrictive | human | design | high |
| No install-time code execution (ever) | Fundamental security guarantee; shapes entire install pipeline | human | design | high |
| Interface-aware pruning strategy | Determines when republishes can be skipped; affects ecosystem velocity | agent | compile | med |
| Failure semantics: no rollback of published artifacts | Shapes recovery UX and partial-state handling | human | design | med |
| Runtime containment model (least privilege, scoped credentials) | Determines what even trusted code can do | human | design | high |

### Design Freeze

Before implementation begins, these must be resolved:

- [ ] Signing key format: Ed25519 vs ECDSA, key storage mechanism
- [ ] Policy file schema: `ailang-policy.toml` fields and semantics
- [ ] Classification taxonomy: confirm 5-class model (SAFE/ADDITIVE/BREAKING/AUTHORITY/UNKNOWN)
- [ ] No install-time execution: confirm this is absolute (no exceptions)
- [ ] Runtime containment scope: what restrictions apply to trusted code
- [ ] Failure semantics: confirm no-rollback model for published artifacts
- [ ] Install flow: confirm metadata-only fetch → policy check → artifact fetch sequence

## Solution Design

### Overview

This design integrates four concerns into a unified package evolution system:

1. **Autonomous upgrade loop** — detect, plan, classify, verify, execute, publish, notify
2. **Security model** — six defense layers from language constraints to runtime containment
3. **Policy engine** — admission control via `ailang-policy.toml`
4. **Trust infrastructure** — signed publishing with provenance verification

### Core Principles

**P1 — Package Identity Is Inviolable.** A published package version means: exact dependency graph, exact interface + effects, immutable content. No root override of transitive dependencies.

**P2 — No Code Runs at Install Time.** Installation follows a strict sequence: (1) fetch metadata only, (2) verify provenance + evaluate policy, (3) fetch artifact, (4) verify artifact hash + signature, (5) extract. Never: execute, evaluate, or mutate environment.

**P3 — Upgrades Are Explicit Graph Transformations.** An upgrade is a deterministic transformation of a dependency DAG, not implicit version drift.

**P4 — Trust Is First-Class.** A package is not trusted because it resolves. It must satisfy: identity, provenance, policy, compatibility.

**P5 — Automation Is Policy-Bounded.** Autonomous actions must respect policy, pass verification gates, and be traceable. Autonomous upgrade mode only operates on packages whose provenance is verifiable under the configured trust policy.

### Architecture

#### A. Autonomous Upgrade Loop

```
Detect → Plan → Classify → Verify → Execute → Publish → Notify
```

**Event Detection** — Triggers:
- Package publish event
- Security advisory
- Manual request (`ailang upgrade`)

```json
{
  "type": "package_published",
  "package": "sunholo/firestore",
  "version": "0.2.0"
}
```

**Impact Graph** — Reverse dependency lookup, build minimal affected DAG.

**Upgrade Plan** — Graph structure with nodes and edges (reuses `UpgradeGraph` from M-PKG-UPGRADE-CHAIN-TOOLING):

```json
{
  "target": "sunholo/firestore@0.2.0",
  "nodes": [
    {
      "package": "sunholo/billing_store",
      "from": "0.5.0",
      "to": "0.6.0",
      "classification": "SAFE",
      "interface_change": false
    }
  ],
  "edges": [
    ["sunholo/firestore", "sunholo/billing_store"]
  ]
}
```

#### B. Compatibility Classification

Based on interface hash, effects, and contracts:

| Class | Meaning | Action |
|-------|---------|--------|
| SAFE | Identical interface + effects | Auto-upgrade |
| ADDITIVE | Backward-compatible additions | Auto (if policy allows) |
| BREAKING | Incompatible interface change | Block |
| AUTHORITY | Effects widened (new capabilities) | Block |
| UNKNOWN | Unverifiable (no interface hash, etc.) | Block |

**Important:** SAFE means *structurally compatible by available interface/effect signals*, not proof of semantic equivalence. A package can keep the same interface hash and effect set while changing runtime behavior. Contracts and future behavioral verification (see Future Work) strengthen this guarantee over time.

**Interface-Aware Pruning:** If a dependency version changes BUT interface hash + effects are unchanged → skip republish (optional, opt-in).

#### C. Supply-Chain Defense Model

**Threat Model:**

| Threat | Example |
|--------|---------|
| Compromised publisher | Stolen API key uploads malicious version |
| Malicious release | Injected code in otherwise-legitimate package |
| Supply chain poisoning | Transitive dependency attack |
| Install-time execution | `.pth` exploit, post-install scripts |
| Silent behavior change | Dependency drift changes semantics |

**Six Defense Layers:**

**Layer 1 — Language Constraints:** Explicit effects, no ambient authority, effect ceilings. Already enforced by AILANG's type system.

**Layer 2 — Package Integrity:** Content hash, interface hash, immutable versions. Already implemented in registry (v0.9.7).

**Layer 3 — Provenance (NEW):** Signed publishing establishes artifact origin. Each package includes:

```json
{
  "content_hash": "sha256:abc...",
  "interface_hash": "sha256:def...",
  "signature": "ed25519:...",
  "publisher": "key-id",
  "timestamp": "2026-04-01T12:00:00Z"
}
```

Registry enforces: signature valid, publisher authorized for namespace.

**Layer 4 — Admission Policy (NEW):** Policy decides whether a given origin is acceptable in a given project or environment. Provenance establishes *who*; policy decides *whether*. Packages must pass consumer-defined policy before admission.

```toml
# ailang-policy.toml

[trust]
allowed_publishers = ["sunholo/*"]
allowed_registries = ["registry.sunholo.com"]

[effects]
max_allowed = ["Net"]

[upgrade]
allow_safe = true
allow_additive = true
allow_breaking = false
allow_authority_widening = false

[verification]
require_contracts = true
```

**Layer 5 — Change Classification (NEW):** Every publish produces:
- Interface delta (hash comparison)
- Effect delta (new/removed effects)
- Dependency delta (version changes)
- Publisher change flag (different signer than previous version)

**Layer 6 — Runtime Containment:** Even trusted code runs with: least privilege, scoped credentials, network restrictions, container isolation. Note: runtime containment is a required deployment practice; the specific enforcement mechanism is out of scope for this doc and depends on the execution environment (Cloud Run, local dev, etc.).

#### D. Coordinator Integration

Per upgrade node:
1. Update manifest
2. Run verification (`ailang check --package`)
3. Enforce admission policy
4. Publish to registry
5. Emit structured message

All changes emit events:

```json
{
  "type": "package_upgraded",
  "package": "sunholo/billing_store",
  "from": "0.5.0",
  "to": "0.6.0",
  "cause": "cascade upgrade of sunholo/firestore@0.1.0 → 0.2.0",
  "classification": "SAFE",
  "provenance": {
    "signature": "ed25519:...",
    "publisher": "key-id"
  }
}
```

### Implementation Plan

**Phase 1: Upgrade Tooling Foundation (v0.10.x)** (~1 week)
- [ ] `ailang upgrade-plan <pkg>@<version>` — compute upgrade graph
- [ ] `ailang republish-chain <pkg>@<version>` — execute cascade
- [ ] `ailang lock --explain` — enhanced conflict diagnostics with first-resolver provenance
- [ ] Reuse `UpgradeGraph` types from M-PKG-UPGRADE-CHAIN-TOOLING
- [ ] Tests for graph construction, topo sort, compatibility check

**Phase 2: Classification + Pruning** (~1 week)
- [ ] Define `ChangeClassification` type (SAFE/ADDITIVE/BREAKING/AUTHORITY/UNKNOWN)
- [ ] Interface hash comparison between old and new dependency versions
- [ ] Effect delta computation
- [ ] Publisher change detection
- [ ] Graph-based plan format with classification per node
- [ ] Interface-aware pruning (opt-in `--skip-compatible`)
- [ ] Tests for classification edge cases

**Phase 3: Signed Publishing + Policy** (~1 week)
- [ ] Key generation: `ailang pkg keygen` (Ed25519)
- [ ] Signing on publish: `ailang publish --sign`
- [ ] Registry signature verification on upload
- [ ] `ailang-policy.toml` parser and schema
- [ ] Policy enforcement on `ailang install` / `ailang lock`
- [ ] `ailang policy check <pkg>@<version>` — dry-run policy evaluation
- [ ] Tests for signature verification, policy enforcement

**Phase 4: Coordinator Integration + Autonomous Mode** (~1 week)
- [ ] `ailang coordinator --auto-upgrade` mode (requires all packages have verified provenance)
- [ ] Event detection → plan → classify → verify → execute → publish pipeline
- [ ] Reject unsigned/unverified packages from autonomous flows
- [ ] Policy-bounded automation (SAFE auto, ADDITIVE configurable, BREAKING/AUTHORITY block)
- [ ] Structured messaging for all upgrade events
- [ ] Circuit breaker for cascade failures
- [ ] Budget limits for autonomous operations
- [ ] End-to-end integration tests

### Files to Modify/Create

**New files:**
- `internal/pkg/classify.go` — Change classification engine (~200 LOC)
- `internal/pkg/classify_test.go` — Classification tests (~200 LOC)
- `internal/pkg/signing.go` — Ed25519 signing/verification (~150 LOC)
- `internal/pkg/signing_test.go` — Signing tests (~100 LOC)
- `internal/pkg/policy.go` — Policy engine + TOML parser (~250 LOC)
- `internal/pkg/policy_test.go` — Policy tests (~200 LOC)
- `cmd/ailang/policy.go` — `ailang policy check` CLI command (~80 LOC)
- `cmd/ailang/keygen.go` — `ailang pkg keygen` CLI command (~60 LOC)

**Modified files:**
- `internal/pkg/upgrade.go` — Add classification to upgrade nodes (~+50 LOC)
- `internal/pkg/resolver.go` — Policy enforcement during resolution (~+30 LOC)
- `internal/pkg/registry.go` — Signature verification on publish/fetch (~+60 LOC)
- `cmd/ailang/publish.go` — Add `--sign` flag (~+20 LOC)
- `cmd/ailang/install.go` — Policy check on install (~+20 LOC)
- `internal/coordinator/daemon_tasks.go` — Auto-upgrade task type (~+80 LOC)

## Examples

### Example 1: Upgrade with Classification

```
$ ailang upgrade sunholo/firestore@0.2.0

upgrade plan for sunholo/firestore@0.2.0

  classification:
    sunholo/billing_store@0.5.0
      dependency change: firestore 0.1.0 → 0.2.0
      interface impact: none (same interface hash)
      class: SAFE
      → auto-republish allowed

    sunholo/access_gate@0.4.0
      dependency change: billing_store 0.5.0 → 0.6.0
      interface impact: new export added
      class: ADDITIVE
      → auto-republish allowed (policy: allow_additive = true)

  republish order (2 packages):
    1. sunholo/billing_store@0.5.0 → 0.6.0 [SAFE]
    2. sunholo/access_gate@0.4.0 → 0.5.0 [ADDITIVE]

  execute? [y/N/dry-run]
```

### Example 2: Policy Blocks Untrusted Publisher

```
$ ailang install sketchy-org/data-tools@1.0.0

error: policy violation
  package: sketchy-org/data-tools@1.0.0
  rule: trust.allowed_publishers = ["sunholo/*"]
  publisher "sketchy-org" is not in allowed list

  to override: add "sketchy-org/*" to ailang-policy.toml [trust] section
```

### Example 3: Authority Widening Blocked

```
$ ailang upgrade sunholo/firestore@0.3.0

upgrade plan for sunholo/firestore@0.3.0

  classification:
    sunholo/firestore@0.2.0 → 0.3.0
      new effects: [FileSystem]
      class: AUTHORITY
      → blocked by policy (allow_authority_widening = false)

  action required: review effect widening before proceeding
  run: ailang upgrade --allow-authority sunholo/firestore@0.3.0
```

### Example 4: Signed Publishing

```
$ ailang publish --sign
  signing with key: ~/.ailang/keys/sunholo.ed25519
  content hash: sha256:abc123...
  interface hash: sha256:def456...
  signature: ed25519:789...

  published: sunholo/billing_store@0.6.0
  provenance: signed by sunholo-voight-kampff
```

## CLI Surface

| Command | Purpose |
|---------|---------|
| `ailang upgrade <pkg>@<version>` | Compute plan, show diff, classify risk, optionally execute |
| `ailang upgrade-plan <pkg>@<version>` | Compute and display plan without executing |
| `ailang republish-chain <pkg>@<version>` | Execute cascade republish |
| `ailang lock --explain` | Show conflict chain with first-resolver provenance |
| `ailang publish --sign` | Sign package on publish |
| `ailang pkg keygen` | Generate Ed25519 signing keypair |
| `ailang policy check <pkg>@<version>` | Dry-run policy evaluation |
| `ailang coordinator --auto-upgrade` | Autonomous upgrade mode (requires verified provenance) |

**Note:** Policy evaluation is automatic on `ailang install` and `ailang lock` when `ailang-policy.toml` exists. `ailang policy check` is for dry-run evaluation without modifying state.

## Failure Semantics

**No rollback of published artifacts.** Once published, a version is permanent.

Instead of rollback:
- Stop execution at failure point
- Mark partial state explicitly
- Emit diagnostics with recovery instructions

| State | Meaning | Recovery |
|-------|---------|----------|
| BLOCKED | Policy violation | Review policy, adjust or override |
| FAILED | Execution error (build/test failure) | Fix code, resume from failed node |
| PARTIAL | Incomplete chain | Resume from last successful node |

## Versioning Rules

| Condition | Default Bump | Rationale |
|-----------|-------------|-----------|
| Dependency change only, no source changes | Patch | Minimal version increment |
| Source code also changed | Minor | Signals functional change |
| Breaking interface change | Explicit bump required (tool refuses implicit default) | Consumer must adapt; silent minor bump would hide breakage |
| Override via CLI | `--patch` / `--minor` / `--major` | Explicit control |

## Success Criteria

- [ ] No silent dependency substitution at any layer
- [ ] All upgrades traceable via structured provenance chain
- [ ] Malicious package cannot execute at install time (by construction)
- [ ] Untrusted publisher blocked before package artifact contents are fetched, extracted, or admitted into the dependency graph
- [ ] Upgrade chains computable as DAGs and automatable by agents
- [ ] Classification correctly categorizes SAFE/ADDITIVE/BREAKING/AUTHORITY/UNKNOWN
- [ ] `ailang-policy.toml` enforced on install and lock
- [ ] Signed packages verified on registry upload and consumer fetch
- [ ] Coordinator auto-upgrade respects policy boundaries
- [ ] System converges to consistent state autonomously
- [ ] All tests passing
- [ ] Documentation updated
- [ ] Examples added

## Testing Strategy

**Unit tests:**
- Classification engine: all 5 classes with edge cases
- Signing: key generation, sign, verify, tampered-content rejection
- Policy: TOML parsing, each rule type, composition of rules
- Upgrade graph: classification per node, pruning decisions

**Integration tests:**
- End-to-end upgrade with classification and policy
- Signed publish → fetch → verify cycle
- Policy rejection on install
- Coordinator auto-upgrade with SAFE/ADDITIVE/BLOCKED scenarios
- Cascade failure and partial state recovery

**Manual testing:**
- LiteLLM-class attack scenario (unsigned package, untrusted publisher, effect widening)
- Multi-package cascade upgrade
- Policy override workflow

## Deferred Decisions

The following are intentionally left open for the implementer:

- Key storage location and format (filesystem vs keyring) — agent may choose, human reviews
- Policy file discovery (project root vs `~/.ailang/`) — agent may choose
- CLI output formatting for classification display — agent may choose
- Internal helper naming and organization — agent may choose
- Test fixture structure — agent may choose
- Exact TOML field names in `ailang-policy.toml` (beyond schema) — agent may choose

## Non-Goals

- **Semver range resolution** — AILANG uses exact versions only
- **Multiple versions of same package** — AILANG stays flat
- **Silent dependency override** — rejected (weakens package identity)
- **Install-time code execution** — rejected (fundamental security guarantee)
- **npm-style dynamic dependency behavior** — rejected (nondeterministic)
- **Workspace-mode relaxed resolution** — deferred to separate design doc
- **Unpublishing packages** — breaks immutability guarantee
- **Key revocation infrastructure** — future work (v0.12+)

## Timeline

**Week 1** (Phase 1: Upgrade Tooling Foundation):
- Upgrade graph + `--explain` diagnostics
- `upgrade-plan` + `republish-chain` commands
- Builds on M-PKG-UPGRADE-CHAIN-TOOLING design

**Week 2** (Phase 2: Classification + Pruning):
- 5-class classification engine
- Interface hash + effect delta comparison
- Graph-based plan with classification per node

**Week 3** (Phase 3: Signed Publishing + Policy):
- Ed25519 key generation and signing
- Registry signature verification
- `ailang-policy.toml` parser and enforcement

**Week 4** (Phase 4: Coordinator Integration):
- Auto-upgrade mode
- Policy-bounded automation
- End-to-end integration testing
- Documentation and examples

**Total: ~4 weeks**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Malicious publish via compromised key | High | Signature verification + policy + publisher allowlist |
| Compromised maintainer changes package behavior | High | Classification detects effect/interface changes; publisher change flag |
| Cascade explosion (one upgrade triggers 100+ republishes) | Med | Interface-aware pruning skips no-op republishes; budget limits |
| Runaway autonomous upgrades | Med | Policy boundaries + cost budgets + circuit breaker |
| Partial upgrade leaves ecosystem inconsistent | Med | Explicit PARTIAL state + resume-from-node recovery |
| Hidden behavior change (same interface, different semantics) | Med | Classification flags as SAFE but provenance chain enables audit; contracts (future) add behavioral verification |
| Key management complexity deters adoption | Low | Simple `ailang pkg keygen` UX; key storage deferred to implementer |

## Related Documents

**Implemented (foundations this builds on):**
- [m-pkg-package-system.md](../../implemented/v0_9_5/m-pkg-package-system.md) — Core package system
- [m-pkg-registry.md](../../implemented/v0_9_7/m-pkg-registry.md) — GCS-backed registry with content/interface hashing
- [m-pkg-msg-package-messaging-graph.md](../../implemented/v0_9_9/m-pkg-msg-package-messaging-graph.md) — Package messaging system
- [m-pkg-resolver-direct-wins.md](../../implemented/v0_9_11/m-pkg-resolver-direct-wins.md) — Strict conflict detection

**Planned (direct dependencies/overlap):**
- [m-pkg-upgrade-chain-tooling.md](../v0_10_0/m-pkg-upgrade-chain-tooling.md) — Upgrade graph model + CLI (Phase 1 prerequisite)
- [m-pkg-autonomous-updates.md](../v0_10_0/m-pkg-autonomous-updates.md) — Message-driven autonomous updates (Phase 4 prerequisite)

## References

- [Design Axioms](/docs/references/axioms) — The 12 non-negotiable principles
- [Philosophical Foundations](/docs/references/philosophical-foundations) — Block-universe determinism
- [Design Lineage](/docs/references/design-lineage) — What we adopted/rejected and why
- LiteLLM supply-chain incident (2025) — Motivating external trigger

## Future Work

- **Key revocation** — CRL or transparency log for compromised keys (v0.12+)
- **Behavioral contracts** — Verify semantic equivalence, not just interface compatibility
- **Workspace-mode relaxation** — More flexible resolution for local path deps being edited together
- **CI/CD integration** — `ailang upgrade` as GitHub Action triggered by leaf package publish
- **Multi-registry federation** — Policy enforcement across multiple registries
- **Transparency log** — Append-only log of all publish events for public auditing

---

**Document created**: 2026-03-25
**Last updated**: 2026-03-25
