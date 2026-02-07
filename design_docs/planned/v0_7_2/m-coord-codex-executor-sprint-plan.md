# Sprint Plan: M-COORD-CODEX (Codex Executor for Coordinator + Evals)

## Summary
Enable OpenAI Codex as a first-class executor across coordinator and eval harness, with normalized metrics and robust parsing.

**Duration:** 3 days (reduced from 5 — coordinator integration eliminated by provider factory refactor)
**Dependencies:** Codex CLI installed for integration tests (optional/gated)
**Risk Level:** Low (reduced from Medium — less code to write)

## Current Status Analysis

### Completed Recently
- ✅ v0.7.1.x releases with WASM and telemetry changes (mixed LOC, ~630 LOC noted in recent CHANGELOG)
- ✅ E2E agent handoff verification (design docs + sprint infra)

### Velocity
- Recent average: ~100-150 LOC/day (based on recent CHANGELOG entry and commit activity; LOC metrics are sparse)
- Estimated capacity: ~700-900 LOC for this sprint

### Remaining from Design Doc
- ⏳ Codex executor implementation and registration (~400-600 LOC)
- ~~⏳ Coordinator provider wiring (~200 LOC)~~ **ELIMINATED** — unified `ExecutorProvider` auto-discovers executors from factory
- ⏳ Eval harness provider support + tests (~150-250 LOC)

## Proposed Milestones

### Milestone 1: Codex Executor Core
**Goal:** Implement Codex CLI executor and normalize JSON output into `executor.Result`.
**Estimated:** 350 LOC implementation + 120 LOC tests = 470 LOC
**Duration:** 2 days

**Tasks:**
- Day 1: Implement `internal/executor/codex` with CLI invocation, config defaults, and factory registration.
- Day 2: Add JSON parsing, metrics mapping, and unit tests (including schema drift tolerance).

**Acceptance Criteria:**
- [ ] `executor.GlobalFactory().ListAvailable()` includes `codex` when installed
- [ ] Codex JSON parsing produces `executor.Result` with tokens and cost (if available)
- [ ] Unit tests cover typical JSON output and missing fields
- [ ] All tests passing
- [ ] Linting clean

**Risks:**
- Codex JSON schema changes - Mitigation: tolerant parsing + store raw data

### ~~Milestone 2: Coordinator Provider Integration~~ ELIMINATED
~~**Goal:** Add `CodexProvider` to coordinator and enable routing.~~
**Status:** Not needed. The provider factory refactor (Feb 2026) introduced `ExecutorProvider` in `provider_executor.go` which auto-wraps any executor registered in `executor.GlobalFactory()`. Once Milestone 1 registers "codex" via `init()`, the coordinator discovers and uses it automatically. Zero coordinator code changes required.

### Milestone 2 (was 3): Eval Harness Integration + Docs
**Goal:** Enable `--provider codex` in eval suite and update docs.
**Estimated:** 150 LOC implementation + 60 LOC tests + 40 LOC docs = 250 LOC
**Duration:** 1 day

**Tasks:**
- Day 3: Add provider flag support, result aggregation, CLI plumbing, tests, and docs.

**Acceptance Criteria:**
- [ ] `ailang eval-suite --agent --provider codex` runs (gated if CLI missing)
- [ ] Eval results include Codex metrics (tokens, cost when available)
- [ ] Docs updated (multi-provider + eval architecture)
- [ ] All tests passing
- [ ] Linting clean

**Risks:**
- Missing token/cost fields - Mitigation: estimate via CostModel when needed

## Success Metrics
- Test coverage: maintain current baseline
- Examples passing: unchanged or improved
- Documentation: `.claude/MULTI_PROVIDER_SETUP.md`, `.claude/EVAL_ARCHITECTURE.md`
- All tests passing: ✅
- All linting passing: ✅

## Dependencies
- Codex CLI availability for live integration tests
- OpenAI API key configured for Codex CLI (for live runs)

## Open Questions
- Which Codex CLI flags are required for model selection and tool control?
- Is streaming JSON required for coordinator event handlers in v0.7.2, or batch only?

## Notes
- Integration tests should be gated behind a Codex availability check.
- If Codex CLI does not support tool allowlists, document limitation and proceed.
