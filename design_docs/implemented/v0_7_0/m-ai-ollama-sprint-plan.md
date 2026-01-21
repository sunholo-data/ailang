# M-AI-OLLAMA Sprint Plan

**Sprint ID:** M-AI-OLLAMA
**Design Doc:** [m-eval-ollama-local-models.md](m-eval-ollama-local-models.md)
**Target Version:** v0.6.2
**Estimated Duration:** 2 days (12-14 hours)
**Risk Level:** Low

## Summary

Unify AI provider architecture and add Ollama for local model support. This sprint prioritizes **eval harness migration first** (user requirement), eliminating ~420 LOC of duplicate code before adding Ollama support.

**Key Deliverable:** Run `ailang eval-suite --models ollama:codellama` with local models.

## Current Status Analysis

### Existing Architecture

**Unified `internal/ai/` package (v0.5.10):**
- `provider.go` (99 LOC) - Clean `Provider` interface
- `config.go` (104 LOC) - `ProviderType` enum, `GuessProvider()`
- `openai/`, `anthropic/`, `gemini/` - Working implementations

**Eval harness duplicates (to delete):**
| File | LOC | Status |
|------|-----|--------|
| `internal/eval_harness/api_openai.go` | 159 | Duplicate of `internal/ai/openai/` |
| `internal/eval_harness/api_anthropic.go` | 104 | Duplicate of `internal/ai/anthropic/` |
| `internal/eval_harness/api_google.go` | 154 | Duplicate of `internal/ai/gemini/` |
| **Total to delete** | **417** | |

### Velocity Reference

Recent work shows ~150-200 LOC/day for well-scoped tasks. This sprint is straightforward refactoring + new provider addition.

## Milestones

### M1: Migrate Eval Harness to Unified Providers (~4 hours)

**Goal:** Eval harness uses `internal/ai/` instead of its own API implementations.

**Tasks:**
1. Create adapter in `internal/eval_harness/ai_provider.go` (~80 LOC)
   - Wraps `ai.Provider` for eval harness use
   - Converts `ai.Response` → `GenerateResult`
   - Keeps `extractCodeFromMarkdown()` (eval-specific)

2. Update `internal/eval_harness/ai_agent.go` (~50 LOC changes)
   - Replace `callOpenAI/Anthropic/Gemini` with unified provider calls
   - Remove provider-specific branching in `GenerateCode()`
   - Keep retry logic and MockAIAgent

3. Delete duplicate files (-417 LOC)
   - `api_openai.go`
   - `api_anthropic.go`
   - `api_google.go`

4. Run full eval baseline to verify no regression

**Acceptance Criteria:**
- [ ] `ailang eval-suite --models gpt5-mini` works unchanged
- [ ] `ailang eval-suite --models claude-haiku-4-5` works unchanged
- [ ] `ailang eval-suite --models gemini-2-5-flash` works unchanged
- [ ] All existing eval tests pass
- [ ] ~370 net LOC reduction

**Files Modified:**
| File | Change | LOC |
|------|--------|-----|
| `internal/eval_harness/ai_provider.go` | NEW - Adapter | +80 |
| `internal/eval_harness/ai_agent.go` | Refactor | ~50 changes |
| `internal/eval_harness/api_openai.go` | DELETE | -159 |
| `internal/eval_harness/api_anthropic.go` | DELETE | -104 |
| `internal/eval_harness/api_google.go` | DELETE | -154 |

---

### M2: Add Ollama Provider (~3 hours)

**Goal:** `internal/ai/ollama/` implements `Provider` interface.

**Tasks:**
1. Create `internal/ai/ollama/client.go` (~80 LOC)
   - Uses existing `github.com/ollama/ollama/api` package
   - `NewClient()` with endpoint from `OLLAMA_HOST` env
   - `CheckConnection()` with helpful error message

2. Create `internal/ai/ollama/generate.go` (~60 LOC)
   - Implements `Provider.Generate()`
   - Uses Chat API for instruction following
   - Deterministic seed=42 by default

3. Add to `internal/ai/config.go` (~15 LOC)
   - Add `ProviderOllama ProviderType = "ollama"`
   - Update `GuessProvider()` for `ollama:` prefix

4. Create `internal/ai/ollama/client_test.go` (~100 LOC)
   - Unit tests for client creation
   - Mock tests for generate (skip if Ollama not running)

**Acceptance Criteria:**
- [ ] `ollama.NewClient()` works
- [ ] `client.Generate()` returns valid `ai.Response`
- [ ] `ai.GuessProvider("ollama:codellama")` returns `ProviderOllama`
- [ ] Clear error message when Ollama not running
- [ ] All tests pass (mocked, no Ollama dependency in CI)

**Files Created:**
| File | LOC |
|------|-----|
| `internal/ai/ollama/client.go` | ~80 |
| `internal/ai/ollama/generate.go` | ~60 |
| `internal/ai/ollama/client_test.go` | ~100 |
| `internal/ai/config.go` | +15 |

---

### M3: Integration (~2 hours)

**Goal:** All three touchpoints work with Ollama models.

**Tasks:**
1. Update `cmd/ailang/ai_handlers.go` (~20 LOC)
   - Add Ollama routing in `setupAIHandler()`
   - Connection check before proceeding

2. Update eval harness integration (~10 LOC)
   - Eval harness now auto-routes via unified provider

3. Add local models to `internal/eval_harness/models.yml` (~50 LOC)
   - `ollama-codellama`, `ollama-deepseek-coder`, `ollama-qwen-coder`
   - `local_models` suite definition

4. Manual testing of all touchpoints:
   - `ailang run --ai ollama:codellama --caps AI program.ail`
   - `ailang eval-suite --models ollama:codellama`

**Acceptance Criteria:**
- [ ] CLI `--ai ollama:model` works
- [ ] Eval suite `--models ollama:model` works
- [ ] Models show $0.00 cost in reports
- [ ] Same JSON output format as cloud providers

**Files Modified:**
| File | Change | LOC |
|------|--------|-----|
| `cmd/ailang/ai_handlers.go` | Add Ollama | +20 |
| `internal/eval_harness/models.yml` | Add local models | +50 |

---

### M4: Polish (~1 hour)

**Goal:** Documentation and cleanup.

**Tasks:**
1. Add connection check with helpful error message
2. Update documentation (if needed)
3. Final test run

**Acceptance Criteria:**
- [ ] `make test` passes
- [ ] `make lint` passes
- [ ] Error message shows install instructions when Ollama not running

---

## Day-by-Day Breakdown

### Day 1 (6-7 hours)

| Time | Task |
|------|------|
| 1h | M1.1: Create `ai_provider.go` adapter |
| 1h | M1.2: Refactor `ai_agent.go` |
| 0.5h | M1.3: Delete duplicate files |
| 1h | M1.4: Run eval baseline, verify no regression |
| 1.5h | M2.1-2: Create Ollama client and generate |
| 1h | M2.3-4: Add config and tests |

### Day 2 (5-6 hours)

| Time | Task |
|------|------|
| 1h | M3.1: Update CLI ai_handlers |
| 1h | M3.2-3: Eval harness integration + models.yml |
| 2h | M3.4: Manual testing all touchpoints |
| 1h | M4: Polish, docs, final testing |

## Success Metrics

- [ ] **LOC reduction:** ~370 net LOC deleted (417 deleted, ~50 adapter added)
- [ ] **Architecture unified:** One Provider interface for all AI
- [ ] **Local models work:** `ollama:codellama` runs evals
- [ ] **No regression:** All existing eval tests pass
- [ ] **Test coverage:** New Ollama code has >80% coverage

## Dependencies

- `github.com/ollama/ollama/api` - Already imported for embeddings
- Unified `internal/ai/` package - Complete (v0.5.10)

## Risks & Mitigations

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Eval regression | Low | Run full baseline before/after M1 |
| Token counting differences | Medium | Return 0 for Ollama (documented) |
| Ollama not available in CI | Expected | Skip integration tests, mock unit tests |

## Open Questions

None - design doc is comprehensive.

## Estimated Totals

| Metric | Value |
|--------|-------|
| **LOC Added** | ~385 |
| **LOC Deleted** | ~417 |
| **Net Change** | ~-30 LOC |
| **New Files** | 4 |
| **Deleted Files** | 3 |
| **Test Files** | 1 |
