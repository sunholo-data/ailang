# M-GIT-GUARDRAILS: Per-Agent Git Mode for Cloud Coordinator

**Status**: Planned
**Target**: v0.9.2
**Priority**: P1 (Medium — Phase 1 works via Terraform, this enables per-agent config)
**Estimated**: 3 hours (Go changes + tests + config updates)
**Dependencies**: Phase 1 deployed in `ailang_bootstrap` + `ailang-multivac`

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Enforces deterministic git behavior — agents can't create random branches |
| A2: Replayability | 0 | No impact on AILANG traces |
| A3: Effect Legibility | +1 | Makes git side effects explicit and configurable per agent |
| A4: Explicit Authority | +1 | Git capabilities are declared in agent config, not ambient |
| A5: Bounded Verification | 0 | No verification changes |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | Machine-readable config (`git_mode: guardrails`) replaces implicit behavior |
| A8: Minimal Syntax | 0 | No new AILANG syntax — this is infra/config only |
| A9: Cost Visibility | +1 | Prevents wasted turns ($0.90+) on merge conflicts from rogue branches |
| A10: Composability | 0 | Orthogonal to other features |
| A11: Structured Failure | +1 | Deny messages give structured, self-correcting feedback to agents |
| A12: System Boundary | +1 | Git operations are explicit system boundary crossings, now guarded |

**Net Score: +7** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism — strengthens determinism
- [x] A3 (Effects): Makes git side effects more visible, not less
- [x] A4 (Authority): Enforces capability constraints, no ambient access
- [x] A7 (Machines First): Machine-readable YAML config for git policy

## Problem Statement

AI agents running in Cloud Run Jobs have full git access via `--dangerously-skip-permissions`. In production, agents have:

1. **Created rogue branches** instead of committing to the expected branch (e.g., `feature/redesign` instead of `main`), causing GitHub Pages deploys to miss changes entirely.
2. **Wasted turns on merge conflicts** — spending 10+ turns ($0.90+) fighting rebase issues on branches from previous runs.
3. **Pushed to wrong remotes/branches** — the agent interprets "push your changes" differently each run.

**Current State:**
- Phase 1 deployed: `git_guard.sh` PreToolUse hook in `ailang_bootstrap` intercepts Bash git commands
- `AILANG_GIT_MODE=guardrails` set globally in Terraform `cloud_run_jobs.tf` for both job templates
- Three modes supported by the hook: `guardrails` (default), `strict`, `permissive`
- **All agents get the same mode** — no per-agent differentiation

**Impact:**
- `website-builder` should be `strict` (read-only git — the executor handles all git ops)
- `sprint-executor` should be `guardrails` (can commit locally, push only to expected branch)
- `design-doc-creator` should be `guardrails` (needs to commit design docs)
- Future research agents may need `permissive` (experimental repos)

## Goals

**Primary Goal:** Enable per-agent git mode configuration via `AgentConfig` YAML, passed through the dispatcher to Cloud Run Jobs.

**Success Metrics:**
- `git_mode` field available in `config.cloud.yaml` agent definitions
- Per-agent `AILANG_GIT_MODE` env var reaches Cloud Run Job containers
- Default fallback to `"guardrails"` when unset
- Zero changes to the existing hook (`git_guard.sh`) — it already reads `AILANG_GIT_MODE`

## Solution Design

### Overview

Add `git_mode` to the existing `AgentConfig` → `DispatchParams` → Cloud Run Job env var pipeline. This follows the exact same pattern as `Model`, `Timeout`, and `AuthMode` fields added in v0.8.0–v0.9.2.

### Architecture

The plumbing already exists for similar fields. This is a 4-file change following established patterns:

```
config.cloud.yaml          # Agent declares: git_mode: strict
    ↓
AgentConfig.GitMode        # Parsed from YAML
    ↓
DispatchParams.GitMode     # Copied during task dispatch
    ↓
AILANG_GIT_MODE env var    # Injected into Cloud Run Job
    ↓
git_guard.sh reads it      # Already implemented — no changes needed
```

**Components:**
1. **AgentConfig field**: New `GitMode string` in `agent_registry.go`
2. **DispatchParams field**: New `GitMode string` in `cloud_dispatcher.go`
3. **Dispatcher wiring**: Pass env var in `dispatcher.go`
4. **Coordinator wiring**: Copy from AgentConfig to DispatchParams in `daemon_tasks_exec.go`
5. **Fallback**: Set default in `coordinator_cloud.go` before executor runs

### Implementation Plan

**Phase 1: Go Plumbing** (~2 hours)

- [ ] Add `GitMode string` to `AgentConfig` in `internal/coordinator/agent_registry.go`
- [ ] Add `GitMode string` to `DispatchParams` in `internal/coordinator/cloud_dispatcher.go`
- [ ] Wire `AgentConfig.GitMode` → `DispatchParams.GitMode` in `internal/coordinator/daemon_tasks_exec.go`
- [ ] Pass `AILANG_GIT_MODE` env var in `internal/dispatch/cloudrun/dispatcher.go`
- [ ] Add fallback `os.Setenv("AILANG_GIT_MODE", "guardrails")` in `cmd/ailang/coordinator_cloud.go`

**Phase 2: Config + Testing** (~1 hour)

- [ ] Add `git_mode` to agent definitions in `ailang-multivac/config/config.cloud.yaml`
- [ ] Add unit test for DispatchParams with GitMode
- [ ] Add integration test verifying env var reaches container config
- [ ] Verify `make test` passes

### Files to Modify/Create

**Modified files (ailang repo):**
- `internal/coordinator/agent_registry.go` — Add `GitMode` field to `AgentConfig` (~2 LOC)
- `internal/coordinator/cloud_dispatcher.go` — Add `GitMode` field to `DispatchParams` (~2 LOC)
- `internal/coordinator/daemon_tasks_exec.go` — Wire AgentConfig → DispatchParams (~4 LOC)
- `internal/dispatch/cloudrun/dispatcher.go` — Pass `AILANG_GIT_MODE` env override (~5 LOC)
- `cmd/ailang/coordinator_cloud.go` — Default fallback before `runExecutor()` (~3 LOC)

**Modified files (ailang-multivac repo):**
- `config/config.cloud.yaml` — Add `git_mode` to agent definitions (~6 LOC)

**No new files needed.**

## Examples

### Example 1: Per-Agent Config in YAML

**Before (all agents get same mode from Terraform):**
```yaml
# config.cloud.yaml
- id: website-builder
  provider: claude
  model: sonnet
  # No git_mode — all agents use Terraform default
```

**After (per-agent git mode):**
```yaml
# config.cloud.yaml
- id: website-builder
  provider: claude
  model: sonnet
  git_mode: strict        # Read-only git — executor handles all ops

- id: sprint-executor
  provider: claude
  model: opus
  git_mode: guardrails    # Can commit, push only to expected branch

- id: design-doc-creator
  provider: claude
  model: haiku
  git_mode: guardrails    # Can commit design docs
```

### Example 2: Dispatcher Env Var Injection

Follows existing pattern for Model/Timeout/AuthMode:

```go
// internal/dispatch/cloudrun/dispatcher.go
// M-GIT-GUARDRAILS: Pass per-agent git mode for PreToolUse hook enforcement.
if params.GitMode != "" {
    envOverrides = append(envOverrides, &runpb.EnvVar{
        Name: "AILANG_GIT_MODE", Values: &runpb.EnvVar_Value{Value: params.GitMode},
    })
}
```

### Example 3: Coordinator Wiring

Follows existing pattern at `daemon_tasks_exec.go:147-156`:

```go
// After existing Model/Timeout/AuthMode wiring:
if agent.GitMode != "" {
    params.GitMode = agent.GitMode
}
```

### Example 4: Cloud Executor Fallback

```go
// cmd/ailang/coordinator_cloud.go — before runExecutor()
// M-GIT-GUARDRAILS: Default to guardrails if not set per-agent.
// The Terraform env var also sets this, but this provides a code-level default
// so local coordinator runs and test environments also get guardrails.
if os.Getenv("AILANG_GIT_MODE") == "" {
    os.Setenv("AILANG_GIT_MODE", "guardrails")
}
```

## Success Criteria

- [ ] `git_mode: strict` on website-builder prevents git write ops in cloud
- [ ] `git_mode: guardrails` on sprint-executor allows commits but blocks rogue pushes
- [ ] Default `guardrails` applied when `git_mode` is omitted from config
- [ ] Terraform-level `AILANG_GIT_MODE` overridden by per-agent config when set
- [ ] `make test` passes
- [ ] No changes needed to `git_guard.sh` hook (it already reads `AILANG_GIT_MODE`)

## Testing Strategy

**Unit tests:**
- Test `DispatchParams` serialization with `GitMode` field
- Test `AgentConfig` YAML parsing with `git_mode` field
- Test default fallback logic in coordinator

**Integration tests:**
- Verify dispatcher includes `AILANG_GIT_MODE` in Cloud Run Job env overrides
- Verify per-agent config overrides Terraform default

**Manual testing:**
- Deploy with `website-builder: strict`, verify git writes blocked
- Deploy with `sprint-executor: guardrails`, verify push to expected branch works
- Verify local development (no `AILANG_GIT_MODE`) is unaffected

## Non-Goals

**Not in this feature:**
- Modifying `git_guard.sh` hook logic — already complete in Phase 1
- Adding new git modes beyond `guardrails`/`strict`/`permissive` — three modes sufficient
- Gemini CLI git guardrails — Gemini has no hook system; out of scope
- Observatory logging of blocked commands — deferred to Phase 3
- Lifecycle pre/post scripts — deferred to Phase 3
- Commit message validation — executor's commit is authoritative

## Timeline

**Day 1** (3 hours):
- Phase 1: All Go changes (5 files, ~16 LOC total)
- Phase 2: Config + tests
- PR and deploy

**Total: ~3 hours, single session**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Per-agent env var overrides Terraform default unexpectedly | Low | Dispatcher only sends env var when `GitMode != ""` — empty preserves Terraform default |
| Agent uses `permissive` and does destructive git ops | Med | `permissive` still blocks `--force` and `reset --hard` (hardcoded in hook) |
| Config typo (e.g., `git_mode: gaurdails`) silently ignored | Low | Hook treats unknown modes as permissive with warning — could add validation in AgentConfig |

## Related Documents

<!-- Auto-populated by Ollama neural search on "git guardrails" -->

**Implemented (informs design):**
- [design_docs/implemented/v0_8_1/m-process-subcmd-allowlist.md](design_docs/implemented/v0_8_1/m-process-subcmd-allowlist.md) — Similar pattern: constraining what commands agents can execute
- [design_docs/implemented/v0_3_14/BULLETPROOF_SUMMARY.md](design_docs/implemented/v0_3_14/BULLETPROOF_SUMMARY.md) — Robustness patterns for agent execution

**Planned (check for overlap):**
- [design_docs/planned/v0_10_0/m-arch-boundaries.md](design_docs/planned/v0_10_0/m-arch-boundaries.md) — System boundary architecture (overlaps with git as system boundary)

**External:**
- [ailang-multivac/docs/design/git-guardrails.md](../../../ailang-multivac/docs/design/git-guardrails.md) — Full Phase 1 design doc (hook logic, decision matrix, edge cases)

## References

- [Design Axioms](/docs/references/axioms) - The 12 non-negotiable principles
- [ailang-multivac Phase 1 design doc](../../../ailang-multivac/docs/design/git-guardrails.md) — Hook implementation, decision matrix, edge cases
- Claude Code hooks documentation — PreToolUse hook payload format
- [Agent message: Git Guardrails Phase 2](msg_20260316_085930_b3613f1a) — Original request from design-doc-creator agent

## Future Work

- **Phase 3: Lifecycle scripts** — Pre/post scripts for validation, normalization, deterministic commit messages
- **Phase 3: Observatory logging** — Log blocked git commands to telemetry for visibility
- **Phase 3: Manifest files** — Declarative git policy files alongside agent config
- **Config validation** — Reject unknown `git_mode` values at config load time (not silently)
- **Gemini agent guardrails** — Wrapper script approach since Gemini CLI lacks hooks

---

**Document created**: 2026-03-16
**Last updated**: 2026-03-16
