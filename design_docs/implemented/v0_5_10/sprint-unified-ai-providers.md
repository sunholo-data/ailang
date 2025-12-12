# Sprint Plan: M-UNIFIED-AI-PROVIDERS

## Summary
Create unified `internal/ai/` package consolidating all AI provider implementations, eliminating code duplication across eval harness, CLI `--ai` flag, and std/ai effect. Implements new Gemini Interactions and OpenAI Responses APIs.

**Duration:** 6 days (~30 hours)
**Dependencies:** None
**Risk Level:** Medium (external API dependencies, must maintain backward compatibility)

## Current Status Analysis

### Completed Recently
- M-DX11-TYPE-REPORT: Type debugging API (~300 LOC) in 1 day
- M-DX11-CYCLES: Cycle detection for recursive ADTs (~660 LOC) in 2 days
- Math builtins: 17 new functions (~250 LOC) in 1 day
- String conversion builtins (~240 LOC) in 0.5 day

### Velocity
- Recent average: ~200-250 LOC/day
- Estimated capacity: ~1,400 LOC for this 6-day sprint

### Current AI Implementation (to be unified)
| Location | Provider | API | LOC |
|----------|----------|-----|-----|
| `internal/eval_harness/api_google.go` | Gemini | generateContent | ~150 |
| `internal/eval_harness/api_openai.go` | OpenAI | Chat Completions | ~120 |
| `cmd/ailang/ai_handlers.go` | All 3 | Various | ~345 |
| **Total duplicated** | | | **~615** |

## Proposed Milestones

### Milestone 1: Core Infrastructure
**Goal:** Create `internal/ai/` package with common Provider interface and shared types
**Estimated:** 150 LOC implementation + 50 LOC tests = 200 LOC
**Duration:** 0.5 days

**Tasks:**
- Create `internal/ai/provider.go` with Provider interface, Request/Response types
- Create `internal/ai/config.go` for model config loading from models.yml
- Create `internal/ai/handler.go` with effects.AIHandler wrapper
- Add unit tests for common types

**Files:**
- `internal/ai/provider.go` (~100 LOC)
- `internal/ai/config.go` (~50 LOC)
- `internal/ai/handler.go` (~40 LOC)
- `internal/ai/provider_test.go` (~50 LOC)

**Acceptance Criteria:**
- [ ] Provider interface defined with Generate() method
- [ ] Request/Response types match design doc
- [ ] Handler wrapper implements effects.AIHandler
- [ ] `make test` passes
- [ ] `make lint` clean

**Risks:**
- None - straightforward type definitions

---

### Milestone 2: Anthropic Provider
**Goal:** Migrate AnthropicHandler from ai_handlers.go to unified package
**Estimated:** 100 LOC implementation + 50 LOC tests = 150 LOC
**Duration:** 0.5 days

**Tasks:**
- Create `internal/ai/anthropic/client.go` with Messages API
- Create `internal/ai/anthropic/handler.go` implementing effects.AIHandler
- Add mock HTTP tests
- Verify `--ai claude-haiku-4-5` works

**Files:**
- `internal/ai/anthropic/client.go` (~80 LOC)
- `internal/ai/anthropic/handler.go` (~30 LOC)
- `internal/ai/anthropic/client_test.go` (~50 LOC)

**Acceptance Criteria:**
- [ ] Client implements Provider interface
- [ ] Handler implements effects.AIHandler
- [ ] Mock tests pass
- [ ] `ailang run --ai claude-haiku-4-5` works (manual test)
- [ ] `make test` passes

**Risks:**
- None - simple extraction of existing working code

---

### Milestone 3: OpenAI Chat Completions
**Goal:** Extract OpenAI Chat Completions to unified package
**Estimated:** 150 LOC implementation + 70 LOC tests = 220 LOC
**Duration:** 0.5 days

**Tasks:**
- Create `internal/ai/openai/types.go` with request/response structs
- Create `internal/ai/openai/client.go` with Client and routing logic
- Create `internal/ai/openai/chat.go` with Chat Completions implementation
- Create `internal/ai/openai/handler.go` implementing effects.AIHandler
- Add mock HTTP tests

**Files:**
- `internal/ai/openai/types.go` (~80 LOC)
- `internal/ai/openai/client.go` (~100 LOC)
- `internal/ai/openai/chat.go` (~100 LOC)
- `internal/ai/openai/handler.go` (~50 LOC)
- `internal/ai/openai/client_test.go` (~70 LOC)

**Acceptance Criteria:**
- [ ] Chat Completions matches current api_openai.go behavior
- [ ] Token counting correct (including reasoning tokens)
- [ ] Model detection routes to Chat API
- [ ] `ailang run --ai gpt5-mini` works (manual test)
- [ ] `make test` passes

**Risks:**
- Breaking existing functionality - Mitigation: Keep old api_openai.go until verified

---

### Milestone 4: OpenAI Responses API
**Goal:** Add Responses API support for codex models
**Estimated:** 200 LOC implementation + 100 LOC tests = 300 LOC
**Duration:** 1 day

**Tasks:**
- Create `internal/ai/openai/responses.go` with Responses API
- Implement `/v1/responses` endpoint format
- Handle input array and developer role
- Support reasoning.effort parameter
- Track reasoning tokens separately
- Add comprehensive mock tests

**Files:**
- `internal/ai/openai/responses.go` (~200 LOC)
- `internal/ai/openai/client_test.go` (+100 LOC)

**Acceptance Criteria:**
- [ ] Responses API request format matches OpenAI docs
- [ ] Response parsing handles output array
- [ ] reasoning.effort parameter supported
- [ ] Model detection routes codex models to Responses API
- [ ] Mock tests pass for success and error cases
- [ ] `make test` passes

**Risks:**
- API format changes - Mitigation: Pin to documented format
- No local testing - Mitigation: Comprehensive mock tests

---

### Milestone 5: Gemini generateContent
**Goal:** Extract Gemini generateContent to unified package
**Estimated:** 150 LOC implementation + 80 LOC tests = 230 LOC
**Duration:** 0.5 days

**Tasks:**
- Create `internal/ai/gemini/types.go` with request/response structs
- Create `internal/ai/gemini/client.go` with Client and auth handling
- Create `internal/ai/gemini/generate.go` with generateContent implementation
- Create `internal/ai/gemini/handler.go` implementing effects.AIHandler
- Support both ADC auth (Vertex AI) and API key auth (AI Studio)
- Add mock HTTP tests

**Files:**
- `internal/ai/gemini/types.go` (~100 LOC)
- `internal/ai/gemini/client.go` (~120 LOC)
- `internal/ai/gemini/generate.go` (~100 LOC)
- `internal/ai/gemini/handler.go` (~60 LOC)
- `internal/ai/gemini/client_test.go` (~80 LOC)

**Acceptance Criteria:**
- [ ] generateContent matches current api_google.go behavior
- [ ] Both ADC and API key auth supported
- [ ] Token counting correct
- [ ] `ailang run --ai gemini-2-5-flash` works (manual test)
- [ ] `make test` passes

**Risks:**
- Auth complexity - Mitigation: Support both auth methods

---

### Milestone 6: Gemini Interactions API
**Goal:** Add Interactions API support
**Estimated:** 150 LOC implementation + 100 LOC tests = 250 LOC
**Duration:** 1 day

**Tasks:**
- Create `internal/ai/gemini/interactions.go`
- Implement `/v1beta/interactions` endpoint format
- Handle input formats (string, content objects)
- Support `previous_interaction_id` for stateful conversations
- Parse response with outputs array
- Add comprehensive mock tests

**Files:**
- `internal/ai/gemini/interactions.go` (~150 LOC)
- `internal/ai/gemini/client_test.go` (+100 LOC)

**Acceptance Criteria:**
- [ ] Interactions API request format matches Google docs
- [ ] Response parsing handles outputs array
- [ ] previous_interaction_id supported (for future use)
- [ ] Model detection can route to Interactions API
- [ ] Mock tests pass
- [ ] `make test` passes

**Risks:**
- API is beta - Mitigation: Abstract behind interface

---

### Milestone 7: Integration & Cleanup
**Goal:** Wire unified package into all consumers, delete duplicated code
**Estimated:** 50 LOC changes + extensive testing
**Duration:** 1 day

**Tasks:**
- Update `internal/eval_harness/api_openai.go` to use `internal/ai/openai`
- Update `internal/eval_harness/api_google.go` to use `internal/ai/gemini`
- Update `cmd/ailang/ai_handlers.go` to use unified providers
- Delete `OpenAIHandler`, `GoogleHandler`, `AnthropicHandler` structs
- Add api_type support to models.yml
- Run full eval baseline comparison

**Files:**
- `internal/eval_harness/api_openai.go` (refactor, net -60 LOC)
- `internal/eval_harness/api_google.go` (refactor, net -100 LOC)
- `cmd/ailang/ai_handlers.go` (refactor, net -180 LOC)
- `internal/eval_harness/models.yml` (+10 LOC)

**Acceptance Criteria:**
- [ ] All existing `--ai` models work (claude, gpt, gemini)
- [ ] Eval harness produces same results
- [ ] AnthropicHandler, OpenAIHandler, GoogleHandler deleted
- [ ] No code duplication between eval harness and CLI
- [ ] `make test` passes
- [ ] `make lint` clean

**Risks:**
- Breaking changes - Mitigation: Test each provider before/after

---

### Milestone 8: Documentation & Release
**Goal:** Update docs, move design docs, prepare for release
**Estimated:** 50 LOC docs + verification
**Duration:** 0.5 days

**Tasks:**
- Update CLAUDE.md with new `internal/ai/` architecture
- Update models.yml with api_type documentation
- Move design docs to `implemented/v0_5_10/`
- Update CHANGELOG.md
- Create examples of `--ai` usage

**Files:**
- `CLAUDE.md` (update ~30 lines)
- `CHANGELOG.md` (add release notes)
- `design_docs/implemented/v0_5_10/` (move 3 docs)

**Acceptance Criteria:**
- [ ] CLAUDE.md documents new architecture
- [ ] Design docs moved to implemented/
- [ ] CHANGELOG.md updated
- [ ] All manual tests pass

**Risks:**
- None

---

## Day-by-Day Schedule

| Day | Milestones | LOC Target | Cumulative |
|-----|------------|------------|------------|
| 1 | M1 (Core) + M2 (Anthropic) | 350 | 350 |
| 2 | M3 (OpenAI Chat) + M4 start | 300 | 650 |
| 3 | M4 (OpenAI Responses) complete | 220 | 870 |
| 4 | M5 (Gemini generate) + M6 start | 280 | 1150 |
| 5 | M6 (Gemini Interactions) complete | 200 | 1350 |
| 6 | M7 (Integration) + M8 (Docs) | 100 | 1450 |

## Success Metrics

- [ ] Test coverage: >80% for new `internal/ai/` package
- [ ] All 3 providers working: anthropic, openai, gemini
- [ ] CLI `--ai` working for all models
- [ ] Eval harness producing same results
- [ ] ~340 LOC deleted from duplicated handlers
- [ ] All tests passing: `make test`
- [ ] All linting passing: `make lint`
- [ ] Documentation updated

## LOC Summary

| Component | Implementation | Tests | Net Change |
|-----------|---------------|-------|------------|
| `internal/ai/` core | 190 | 50 | +240 |
| `internal/ai/anthropic/` | 110 | 50 | +160 |
| `internal/ai/openai/` | 530 | 270 | +800 |
| `internal/ai/gemini/` | 530 | 280 | +810 |
| Refactor eval harness | -160 | - | -160 |
| Refactor ai_handlers.go | -180 | - | -180 |
| **Total** | **1020** | **650** | **+1670** |

## Dependencies

- ANTHROPIC_API_KEY for Anthropic testing (optional - mock tests cover most cases)
- OPENAI_API_KEY for OpenAI testing (optional)
- GOOGLE_API_KEY or ADC for Gemini testing (optional)

## Open Questions

1. Should we add Responses API for gpt5-1-codex-max now or defer until we have API access?
2. Should Interactions API be default for Gemini, or opt-in via models.yml flag?
3. Do we need a feature flag during migration, or can we switch atomically?

## Related Design Docs

- [M-UNIFIED-AI-PROVIDERS](./m-unified-ai-providers.md) - Architecture overview
- [M-GEMINI-INTERACTIONS-API](./m-gemini-interactions-api.md) - Gemini details
- [M-OPENAI-RESPONSES-API](./m-openai-responses-api-sprint.md) - OpenAI details

---

**Document created**: 2025-12-11
**Last updated**: 2025-12-11
