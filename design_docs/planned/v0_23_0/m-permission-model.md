# M-PERMISSION-MODEL: Typed Permission Tiers for the Coordinator via Effect Rows

**Status**: Planned
**Target**: v0.23.0
**Priority**: P1 - Medium
**Estimated**: 1 week
**Dependencies**: Effect row type system (shipped), three-camps architecture (v0.22+)

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Effect → tier mapping is deterministic; same declared effects always yield same tier |
| A2: Replayability | +1 | Audit log enables replay of permission decisions |
| A3: Effect Legibility | +1 | Core feature: formalizes what was previously informal |
| A4: Explicit Authority | +1 | Principal feature — makes all permission escalation explicit and type-checked |
| A5: Bounded Verification | +1 | Type-checker can verify tier compliance at compile time |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | Machine-verifiable permission model; no human guessing required |
| A8: Minimal Syntax | 0 | Uses existing effect row syntax; no new syntax |
| A9: Cost Visibility | +1 | Deployment-tier actions (most expensive/risky) are explicitly gated |
| A10: Composability | +1 | Composes with existing effect system and coordinator |
| A11: Structured Failure | +1 | Typed `PermissionDenied` error with tier info and required escalation |
| A12: System Boundary | +1 | Tier boundaries are explicit system boundaries |

**Net Score: +11** → **Decision: ✅ Proceed**

### Hard Violation Check

- [x] A1 (Determinism): Effect → tier mapping is a pure function
- [x] A3 (Effects): All side effects declared — that's the feature
- [x] A4 (Authority): No ambient access; all authority must be declared in effect row
- [x] A7 (Machines First): Tier model is machine-checkable, not convention-based

## Problem Statement

AILANG's three-camp architecture separates eval-only executors (pure code runners) from Motoko/Claude agentic camps (multi-turn with tool use). But the permission boundary between these camps is informal:

**Current State:**
- Eval-only camp runs arbitrary code — the "read-only" guarantee is convention, not enforcement
- Motoko/Claude agentic camps can call tools with network access, file writes, and shell execution without explicit declaration
- Full-access actions (cloud deployment, credential access, destructive filesystem ops) rely on Claude Code's ad hoc permission prompts — these are per-session and not audited
- No machine-verifiable mapping from "what effects does this task declare?" to "which camp and tier should execute it?"

**Impact:**
- A task that declares `!FS` can be routed to the eval-only camp — but nothing enforces this
- No audit trail: which chains escalated to full-access? Which required HITL approval?
- "Code as Agent Harness" (arXiv:2605.18747) §PEV identifies this as a critical gap: "permissions should depend not only on tool identity but on arguments, environment state, data sensitivity, and expected side effects." AILANG's effect rows already carry this information; the coordinator doesn't use it.

## Goals

**Primary Goal:** Formalize the three-camp permission model using AILANG's existing effect row system, so that a task's declared effects determine its permitted execution tier and any required HITL gates.

**Success Metrics:**
- Coordinator rejects (with typed error) any task routed to a tier below what its declared effects require
- HITL gate fires for any chain that escalates to full-access tier
- Audit log records tier, declared effects, and escalation decisions per chain
- Zero instances of an eval-only camp executing `!Network`-declared tasks

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Static (compile-time) vs. dynamic (runtime) tier enforcement | Static is safer but may block valid dynamic escalation | human | design | high |
| HITL gate UX: blocking prompt vs. async approval queue | Blocking is simpler; async enables parallel chains | human | design | high |
| Rollback scope: single tool call vs. full chain | Single call is simpler; chain-level needs transaction log | agent | compile | med |
| Audit log destination: SQLite chain store vs. separate log | SQLite reuses existing infra; separate log scales better | agent | compile | low |

### Design Freeze

Before implementation begins:

- [ ] Static vs. dynamic tier enforcement strategy confirmed
- [ ] HITL gate UX: blocking prompt vs. async approval queue

## Solution Design

### Overview

Map AILANG effect rows to a three-tier permission model. The coordinator reads declared effects from the task's `.ail` spec (or from the chain's effect signature), looks up the required tier, and either routes to the correct camp or raises a `PermissionDenied` error requiring escalation.

### Architecture

**Tier Model:**

| Tier | Name | Permitted Effects | Camp | HITL Required |
|------|------|------------------|------|---------------|
| T1 | Read-Only | `{}` (pure) or `!Read` only | Eval-only | No |
| T2 | Sandbox-Edit | `!FS`, `!Exec` (sandboxed), `!LLM` | Motoko / Claude agentic | No |
| T3 | Full-Access | `!Network`, `!Deploy`, `!Creds`, `!FS` (unrestricted) | Claude agentic | Yes — HITL gate |

**Effect → Tier Mapping (exhaustive):**

```
!Read           → T1
!LLM            → T2
!FS (read)      → T1
!FS (write)     → T2
!Exec (sandbox) → T2
!Exec (shell)   → T3
!Network        → T3
!Deploy         → T3
!Creds          → T3
```

**Components:**

1. **EffectTierResolver** (`internal/coordinator/permission.go`): Pure function `EffectsToTier(effects EffectRow) (Tier, error)`. Returns the minimum required tier for a given effect set. Deterministic.

2. **TierEnforcer** (`internal/coordinator/enforcer.go`): Middleware in the coordinator task dispatcher. Before routing a chain to a camp, checks `EffectsToTier(task.DeclaredEffects) ≤ camp.Tier`. If not, either escalates (with HITL gate for T3) or returns `PermissionDenied`.

3. **HITLGate** (`internal/coordinator/hitl.go`): For T3 escalation, either blocks with a terminal prompt (v1) or posts to `ailang messages` inbox for async human approval (v2). On approval, records in audit log and proceeds. On denial, terminates chain with `PermissionDenied`.

4. **AuditLogger** (`internal/coordinator/audit.go`): Appends to coordinator's SQLite chain store: chain ID, declared effects, resolved tier, escalation decision (auto/HITL), timestamp, approver.

### Implementation Plan

**Phase 1: EffectTierResolver + Unit Tests** (~1.5 days)
- [ ] `internal/coordinator/permission.go` — pure `EffectsToTier` function
- [ ] `internal/coordinator/permission_test.go` — exhaustive table-driven tests for all effect combinations
- [ ] Verify no existing test regressions (`make test`)

**Phase 2: TierEnforcer + AuditLogger** (~2 days)
- [ ] `internal/coordinator/enforcer.go` — middleware wired into task dispatcher
- [ ] `internal/coordinator/audit.go` — SQLite audit log schema + writer
- [ ] Migration: add `permission_tier` and `declared_effects` columns to chain store
- [ ] `make test` passes

**Phase 3: HITLGate + CLI** (~1.5 days)
- [ ] `internal/coordinator/hitl.go` — blocking HITL prompt for T3 escalation
- [ ] `ailang chains --show-tier` to display tier per chain in `ailang chains` output
- [ ] Documentation: `docs/docs/guides/coordinator.md` — permission tier section

### Files to Modify/Create

**New files:**
- `internal/coordinator/permission.go` — EffectTierResolver (~100 LOC)
- `internal/coordinator/permission_test.go` — exhaustive tests (~150 LOC)
- `internal/coordinator/enforcer.go` — TierEnforcer middleware (~120 LOC)
- `internal/coordinator/audit.go` — AuditLogger (~100 LOC)
- `internal/coordinator/hitl.go` — HITLGate (~80 LOC)

**Modified files:**
- `internal/coordinator/daemon_tasks.go` — wire TierEnforcer into dispatch path (~30 LOC)
- `internal/coordinator/chains.go` — add tier columns to chain schema (~20 LOC)
- `cmd/ailang/chains.go` — display tier in `ailang chains` output (~20 LOC)
- `docs/docs/guides/coordinator.md` — permission tier section (~50 LOC)

## Examples

### Example 1: Task Declaration and Tier Routing

```ailang
-- Task declares only read effects → T1 (eval-only camp)
module task_analyze_types

export func run() : () ! {Read} =
  let src = fs.read("internal/types/checker.go") in
  analyze src

-- Task declares file-write → T2 (agentic camp, no HITL)
module task_fix_bug

export func run() : () ! {FS, LLM} =
  let fix = llm.propose("fix the null dereference") in
  fs.write("internal/types/checker.go", fix)

-- Task declares deploy → T3 (agentic camp, HITL required)
module task_release

export func run() : () ! {Deploy, Network, Creds} =
  gcp.deploy("ailang-multivac", version)
```

### Example 2: HITL Gate in Action

```
$ ailang coordinator run task_release.ail

⚠️  PERMISSION ESCALATION REQUIRED
   Task: task_release.ail
   Declared effects: {Deploy, Network, Creds}
   Required tier: T3 (Full-Access)
   Camp: claude-agentic

   This task will:
   - Write to GCP deployment (gcp.deploy)
   - Access credentials (!Creds)
   - Make network calls (!Network)

   Approve? [y/N]: y

✓ Escalation approved — proceeding with T3 execution
  Audit: chain_id=abc123, tier=T3, approver=human, ts=2026-05-21T09:00:00Z
```

### Example 3: Permission Denied

```
$ ailang coordinator run task_fix_bug.ail --camp eval-only

ERROR: PermissionDenied
  Task declared effects: {FS, LLM}
  Requested camp tier: T1 (Read-Only)
  Required tier: T2 (Sandbox-Edit)

  Resolution: route to motoko or claude-agentic camp,
  or reduce task effects to {Read} for eval-only execution.
```

## Success Criteria

- [ ] `EffectsToTier` is a pure function with 100% test coverage of all effect combinations
- [ ] TierEnforcer rejects (typed error) any task routed below its required tier
- [ ] HITL gate fires for all T3 tasks and records approval in audit log
- [ ] `ailang chains` shows tier column per chain
- [ ] Audit log persists across coordinator restarts
- [ ] Zero `!Network`-declared tasks executed in eval-only camp in CI
- [ ] All tests passing (`make test`)
- [ ] Coordinator guide updated

## Testing Strategy

**Unit tests:**
- Table-driven `TestEffectsToTier` covering all 9 effect types and combinations
- `TestTierEnforcer_Blocks` — verify T1 camp rejects T2/T3 tasks
- `TestHITLGate_Approval` and `TestHITLGate_Denial`

**Integration tests:**
- Run a T3 task in CI with `AILANG_HITL=auto-approve` env var; verify audit log entry
- Run a T1 task in eval-only camp; verify no permission check fires

**Manual testing:**
- Run `task_release.ail` (T3) locally; verify blocking prompt appears
- Inspect `ailang chains --show-tier` output after mixed-tier run

## Deferred Decisions

- Async approval queue via `ailang messages` — agent may add as v2 after blocking gate ships
- Fine-grained argument-level effect analysis (e.g., `fs.write` to `/tmp` vs. `/etc`) — agent may add in a follow-up
- Per-user permission policies (e.g., junior vs. senior agent roles) — deferred to v1_0_0

## Non-Goals

- **Effect inference** — tasks must declare effects explicitly; inference is a separate feature
- **Network-level sandboxing** — this is a coordinator-level policy, not OS-level isolation
- **Revocation of in-flight T3 tasks** — kill switch is out of scope for v1

## Timeline

**Week 1** (~5 days):
- Phase 1: EffectTierResolver + tests (days 1–1.5)
- Phase 2: TierEnforcer + AuditLogger (days 2–3.5)
- Phase 3: HITLGate + CLI + docs (days 4–5)

**Total: ~5 days**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Existing chains have no declared effects — enforcer blocks them all | High | Default undeclared tasks to T2; emit warning, not error, in v1 |
| HITL gate blocks CI pipelines | Med | Add `AILANG_HITL=auto-approve` env var for CI; log as security event |
| Effect declaration is manual and tedious | Med | Add `ailang check --infer-effects` to suggest declarations |

## Related Documents

**Planned (same cluster):**
- [design_docs/planned/v0_23_0/m-harness-dsl.md](design_docs/planned/v0_23_0/m-harness-dsl.md) — Doc 4: workflow DSL declares permissions per stage
- [design_docs/planned/v0_23_0/m-harness-state.md](design_docs/planned/v0_23_0/m-harness-state.md) — Doc 3: shared state tracks tier per active chain

**Implemented (may inform design):**
- [design_docs/implemented/v0_9_0/m-http-hooks-cloud-telemetry-sprint-plan.md](design_docs/implemented/v0_9_0/m-http-hooks-cloud-telemetry-sprint-plan.md) — coordinator hooks patterns

## References

- **Ning et al. (2026).** Code as Agent Harness. arXiv:[2605.18747](https://arxiv.org/abs/2605.18747) — §PEV "multi-tier permission model"; "permissions depend on tool identity, arguments, environment state, data sensitivity, and expected side effects"
- [Design Axioms](/docs/references/axioms)
- [Three-Camps Architecture](../../../docs/docs/guides/three-camps-self-audit.md)
- [Coordinator Guide](../../../docs/docs/guides/coordinator.md)

## Future Work

- **Effect inference**: `ailang check --infer-effects` suggests declarations from function bodies
- **Async HITL via messages**: post T3 escalation to `ailang messages` inbox for human to approve asynchronously
- **Argument-level effect analysis**: distinguish `fs.write("/tmp/...")` (T2) from `fs.write("/etc/...")` (T3) via path analysis

---

**Document created**: 2026-05-21
**Last updated**: 2026-05-21
