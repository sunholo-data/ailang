# M-COORD-CODEX: Codex Executor for Coordinator + Evals

**Status**: Planned (updated post-refactor)
**Target**: v0.7.2
**Priority**: P2
**Estimated**: 2-3 days (reduced from 4-6 after provider factory refactor)
**Dependencies**: None (optional synergy with `m-arch4-executor-stream-processor`)

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

Every feature must align with AILANG's 12 Design Axioms. Score each axiom and verify no hard violations.

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No semantics change; deterministic execution unchanged |
| A2: Replayability | 0 | Trace structure unchanged; only provider diversity |
| A3: Effect Legibility | 0 | No effect system changes |
| A4: Explicit Authority | 0 | No new ambient capabilities |
| A5: Bounded Verification | 0 | Type checking unchanged |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +2 | Improves agent throughput and provider resilience |
| A8: Minimal Syntax | 0 | No syntax changes |
| A9: Cost Visibility | +1 | Adds explicit cost/token metrics per provider |
| A10: Composability | +1 | Uses existing executor abstraction |
| A11: Structured Failure | 0 | Error handling and reporting unchanged |
| A12: System Boundary | 0 | No new boundary crossings |

**Net Score: +4** → **Decision: Proceed**

### Hard Violation Check

**These axioms cannot have −1 scores (automatic rejection):**

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Optimizes for agent execution and evals

### Decision Thresholds

| Net Score | Decision |
|-----------|----------|
| ≥ +2 | ✅ Proceed to implementation |
| 0 to +1 | ⚠️ Needs stronger justification |
| < 0 | ❌ Reject or redesign |
| Any −1 on A1/A3/A4/A7 | ❌ Automatic rejection |

## Problem Statement

The coordinator and eval harness already support multi-provider execution (Claude Code, Gemini CLI), and the docs explicitly describe Codex setup. However, there is no Codex executor implementation, so coordinator routing and agent evals cannot actually run on Codex.

**Current State:**
- Executor abstraction exists but lacks a Codex implementation.
- ~~Coordinator providers only wrap Claude and Gemini.~~ **RESOLVED**: Coordinator now uses unified `ExecutorProvider` that auto-discovers any registered executor from the factory. No per-executor provider files needed.
- Eval harness supports provider flags but lacks Codex wiring and parsing.
- Codex JSON output schema differs from Claude/Gemini (`codex_compat_test.go` documents incompatibility).

**Impact:**
- Users cannot run `ailang eval-suite --agent --provider codex`.
- ~~Coordinator cannot route tasks to Codex even if installed.~~ **RESOLVED**: Coordinator auto-discovers executors via `executor.GlobalFactory().ListAvailable()`.
- Documentation promises multi-provider support that is incomplete.

## Goals

**Primary Goal:** Add Codex as a first-class executor for coordinator tasks and evals with normalized results.

**Success Metrics:**
- `executor.GlobalFactory().ListAvailable()` includes `codex` when CLI present.
- Coordinator can execute a task with provider `codex` (dry run + live).
- `ailang eval-suite --agent --provider codex` runs and produces results.
- Token/cost metrics are recorded in `executor.Result`.

## Solution Design

### Overview

Implement a Codex executor wrapper around the `codex` CLI and integrate it with coordinator and eval harness provider routing. Normalize Codex JSON output into the shared `executor.Result` schema and expose metrics consistently across providers.

### Architecture

Codex is added as a new executor implementation in `internal/executor/codex` and registered in the global factory via `init()`. ~~Coordinator adds a `CodexProvider` adapter mirroring Claude/Gemini behavior.~~ **No coordinator changes needed** — the unified `ExecutorProvider` (from the provider factory refactor) auto-discovers any executor registered in the global factory. Eval harness adds `codex` to provider selection and result aggregation.

**Components:**
1. **Codex Executor**: CLI wrapper, JSON parsing, normalized result mapping.
2. ~~**Coordinator Provider**: `provider_codex.go`, uses executor factory and system prompt.~~ **ELIMINATED** — handled automatically by `ExecutorProvider` in `provider_executor.go`.
3. **Eval Harness Wiring**: provider selection, metrics extraction, CLI flag support.

### Implementation Plan

**Phase 1: Codex Executor Core** (~10-14 hours)
- [ ] Create `internal/executor/codex` with `Executor` implementation
- [ ] Implement Codex JSON parsing and normalize to `executor.Result`
- [ ] Register executor with global factory via `init()` and add config defaults

**~~Phase 2: Coordinator Integration~~ ELIMINATED** (0 hours)
- ~~Add `internal/coordinator/provider_codex.go`~~ — Not needed. The unified `ExecutorProvider` in `provider_executor.go` auto-wraps any executor from the factory. Once Phase 1 registers "codex" via `init()`, the coordinator discovers it automatically via `executor.GlobalFactory().ListAvailable()`.
- ~~Wire provider selection and health checks~~ — Handled by `ExecutorProvider.CanHandle()` and executor's own `HealthCheck()`.
- Question-mode tool limitations are handled generically in `ExecutorProvider.Execute()` (line 121 of `provider_executor.go`).

**Phase 2 (was Phase 3): Eval Harness Integration** (~6-8 hours)
- [ ] Add provider flag support for `codex`
- [ ] Ensure result aggregation includes Codex metrics
- [ ] Add parser unit tests and gated integration test

### Files to Modify/Create

**New files:**
- `internal/executor/codex/` - Codex executor implementation with `init()` registration (~400-600 LOC)

**Modified files:**
- Eval CLI files in `cmd/ailang/` - Add provider selection (~60-120 LOC)
- `internal/executor/codex_compat_test.go` - Expand tests for parsing (~50 LOC)

**No longer needed (post-refactor):**
- ~~`internal/coordinator/provider_codex.go`~~ — `ExecutorProvider` handles this
- ~~`internal/executor/factory.go` changes~~ — Registration via `init()` in the codex package
- ~~`internal/coordinator/agent_registry.go` wiring~~ — Auto-discovered from factory

## Examples

### Example 1: Eval Suite on Codex

**Before:**
```
ailang eval-suite --agent --provider codex
# error: provider "codex" not supported
```

**After:**
```
ailang eval-suite --agent --provider codex
# runs benchmarks via codex CLI
```

### Example 2: Coordinator Task Routing

**Before:**
Coordinator cannot route to Codex even when installed.

**After:**
Coordinator can route tasks to `codex` and collect metrics.

## Success Criteria

- [ ] `codex` appears in executor registry when CLI is present
- [ ] Coordinator can run a task with provider `codex` (dry-run + live)
- [ ] Eval suite supports `--provider codex` with aggregated metrics
- [ ] Token/cost metrics captured in `executor.Result`
- [ ] Docs updated to reflect actual support

## Testing Strategy

**Unit tests:**
- Codex JSON parser normalization into `executor.Result`
- Factory registration and config defaults

**Integration tests:**
- Gated Codex CLI test (skip if `codex` not installed)
- Coordinator provider dry-run path

**Manual testing:**
- `codex exec --quiet "echo test"` produces parsable JSON
- `ailang eval-suite --agent --provider codex` runs end-to-end

## Non-Goals

**Not in this feature:**
- Switching default provider to Codex
- Codex API integration (CLI-only)
- Changes to AILANG language semantics

## Timeline

**Days 1-2** (~10-14 hours):
- Phase 1: Codex executor core + tests

**Day 3** (~6-8 hours):
- Phase 2: Eval harness wiring + docs

**Total: ~16-22 hours across 3 days** (reduced from ~28-36 hours; Phase 2/coordinator work eliminated by provider factory refactor)

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Codex JSON schema drift | Med | Tolerant parsing + store raw `ProviderData` |
| Tool control not supported | Low | Run without tool control and document limitation |
| Missing token/cost fields | Med | Estimate via `CostModel` when possible |

## Related Documents

<!-- Auto-populated by Ollama neural search on "coord codex executor" -->

**Implemented (may inform design):**
- [design_docs/implemented/v0_3_2/design_3_2.md](design_docs/implemented/v0_3_2/design_3_2.md) (1.00)
- [design_docs/implemented/v0_3/20251013_letrec_surface_syntax.md](design_docs/implemented/v0_3/20251013_letrec_surface_syntax.md) (0.95)
- [design_docs/implemented/v0_6_1/m-exec-gemini-sprint-plan.md](design_docs/implemented/v0_6_1/m-exec-gemini-sprint-plan.md) (0.90)

**Planned (check for overlap):**
- [design_docs/planned/v0_7_2/m-dx34-subagent-encouragement.md](design_docs/planned/v0_7_2/m-dx34-subagent-encouragement.md) (1.00)
- [design_docs/planned/v0_8_0/m-arch4-executor-stream-processor.md](design_docs/planned/v0_8_0/m-arch4-executor-stream-processor.md) (0.95)
- [design_docs/planned/v0_8_0/m-eval-unified-exec-codegen.md](design_docs/planned/v0_8_0/m-eval-unified-exec-codegen.md) (0.90)

## References

- [Design Axioms](/docs/references/axioms) - The 12 non-negotiable principles
- [Philosophical Foundations](/docs/references/philosophical-foundations) - Block-universe determinism
- [Design Lineage](/docs/references/design-lineage) - What we adopted/rejected and why
- `.claude/MULTI_PROVIDER_SETUP.md`
- `.claude/EVAL_ARCHITECTURE.md`
- `internal/executor/codex_compat_test.go`

## Future Work

[Features that build on this but are out of scope for now]

---

**Document created**: 2026-02-02
**Last updated**: 2026-02-07 (updated for provider factory refactor — Phase 2 eliminated)
