# Sprint Plan: M-OPENAI-RESPONSES-API

## Summary
Create a unified OpenAI client module supporting both Chat Completions and Responses APIs, enabling:
1. **Eval harness** to benchmark `gpt-5.1-codex-max` (77.9% SWE-Bench)
2. **CLI `--ai` flag** for AILANG programs to use new API features

**Duration:** 4 days (~16-20 hours)
**Parent:** [M-UNIFIED-AI-PROVIDERS](./m-unified-ai-providers.md)
**Dependencies:** Core infrastructure from parent doc
**Risk Level:** Medium (new API integration, external dependency on OpenAI)

### Current OpenAI Usage
- **Eval harness**: `internal/eval_harness/api_openai.go` - Chat Completions API
- **CLI `--ai` flag**: `cmd/ailang/ai_handlers.go:125` - Chat Completions API
- Both need updating to support Responses API for codex models

## Current Status Analysis

### Completed Recently
- M-DX11-TYPE-REPORT: Type debugging API (~300 LOC) in 1 day
- M-DX11-CYCLES: Cycle detection for recursive ADTs (~660 LOC) in 2 days
- Math builtins: 17 new functions (~250 LOC) in 1 day

### Velocity
- Recent average: ~300-400 LOC/day
- Estimated capacity: ~600-800 LOC for this sprint (conservative)

### Remaining from Design Doc
- Phase 1: Core Module - ~330 LOC (client.go, chat.go, types.go)
- Phase 2: Responses API - ~200 LOC (responses.go)
- Phase 3: Eval Integration - ~50 LOC (refactor api_openai.go)
- Phase 4: Tests - ~300 LOC (client_test.go)

## Proposed Milestones

### Milestone 1: Core Module & Types
**Goal:** Create `internal/ai/openai/` package with shared types and client structure
**Estimated:** 250 LOC implementation + 50 LOC tests = 300 LOC
**Duration:** 0.5 days

**Architecture:** Unified `internal/ai/` package shared by eval harness, CLI `--ai`, and std/ai:

```
internal/ai/
├── provider.go              # Common Provider interface (shared with gemini)
├── openai/                  # This design doc
│   ├── client.go            # HTTP client
│   ├── chat.go              # Chat Completions API
│   ├── responses.go         # Responses API (new)
│   └── handler.go           # implements effects.AIHandler
├── gemini/                  # See M-GEMINI-INTERACTIONS-API
│   └── ...
└── anthropic/               # To be migrated from ai_handlers.go
    └── ...
```

**Tasks:**
- Create `internal/ai/provider.go` with common Provider interface
- Create `internal/ai/openai/types.go` with Request/Response structs
- Create `internal/ai/openai/client.go` with Client struct and routing logic
- Implement model detection (Chat vs Responses API)
- Add basic unit tests for model detection

**Files:**
- `internal/ai/provider.go` (~50 LOC) - shared
- `internal/ai/openai/types.go` (~80 LOC)
- `internal/ai/openai/client.go` (~100 LOC)
- `internal/ai/openai/client_test.go` (~70 LOC)

**Acceptance Criteria:**
- [ ] Request/Response types defined with all fields from design doc
- [ ] Client struct with constructor and API routing logic
- [ ] Model detection correctly identifies codex models
- [ ] `make test` passes
- [ ] `make lint` clean

**Risks:**
- None - straightforward type definitions

### Milestone 2: Chat Completions Extraction
**Goal:** Extract existing Chat Completions logic into reusable module
**Estimated:** 150 LOC implementation + 50 LOC tests = 200 LOC
**Duration:** 0.5 days

**Tasks:**
- Create `internal/ai/openai/chat.go` with Chat Completions implementation
- Create `internal/ai/openai/handler.go` implementing `effects.AIHandler`
- Extract logic from `internal/eval_harness/api_openai.go`
- Add system/user message formatting
- Add unit tests with mocked HTTP responses

**Files:**
- `internal/ai/openai/chat.go` (~100 LOC)
- `internal/ai/openai/handler.go` (~50 LOC) - for std/ai integration
- `internal/ai/openai/client_test.go` (+50 LOC)

**Acceptance Criteria:**
- [ ] Chat Completions implementation matches current api_openai.go behavior
- [ ] Token counting correct (including reasoning tokens for GPT-5)
- [ ] Mock HTTP tests passing
- [ ] `make test` passes
- [ ] `make lint` clean

**Risks:**
- Breaking existing functionality - Mitigation: Keep old api_openai.go until verified

### Milestone 3: Responses API Implementation
**Goal:** Implement OpenAI Responses API support for codex models
**Estimated:** 200 LOC implementation + 100 LOC tests = 300 LOC
**Duration:** 1 day

**Tasks:**
- Create `internal/ai/openai/responses.go` with Responses API implementation
- Implement `/v1/responses` endpoint format (input array, reasoning.effort)
- Handle response parsing (output array with message/output_text)
- Support reasoning tokens in usage stats
- Add comprehensive mock tests

**Files:**
- `internal/ai/openai/responses.go` (~200 LOC)
- `internal/ai/openai/client_test.go` (+100 LOC)

**Acceptance Criteria:**
- [ ] Responses API request format matches OpenAI docs
- [ ] Response parsing handles all output types
- [ ] reasoning.effort parameter supported
- [ ] Reasoning tokens tracked separately
- [ ] Mock HTTP tests for success and error cases
- [ ] `make test` passes
- [ ] `make lint` clean

**Risks:**
- API format changes - Mitigation: Pin to documented format, add version checks
- No local testing without API key - Mitigation: Comprehensive mock tests

### Milestone 4: Integration (Eval Harness + CLI)
**Goal:** Wire unified `internal/ai/openai/` into eval harness and CLI `--ai` flag
**Estimated:** 80 LOC changes + integration testing
**Duration:** 0.5 days

**Tasks:**
- Update `internal/eval_harness/api_openai.go` to use `internal/ai/openai`
- Update `cmd/ailang/ai_handlers.go` to use `internal/ai/openai.NewHandler()`
- Delete `OpenAIHandler` struct from ai_handlers.go (replaced by unified handler)
- Update `internal/eval_harness/ai_agent.go` if needed
- Keep extractCodeFromMarkdown helper (shared utility)
- Run integration test with real API (if key available)

**Files:**
- `internal/eval_harness/api_openai.go` (refactor, net -50 LOC)
- `cmd/ailang/ai_handlers.go` (delete OpenAIHandler, net -60 LOC)
- `internal/eval_harness/ai_agent.go` (minor updates if needed)

**Acceptance Criteria:**
- [ ] Eval harness uses new openai module
- [ ] CLI `--ai gpt*` uses new openai module
- [ ] Existing GPT-5/GPT-5.1 models still work (both eval and CLI)
- [ ] `gpt5-1-codex-max` detects Responses API automatically
- [ ] All eval harness tests passing
- [ ] `make test` passes
- [ ] `make lint` clean

**Risks:**
- Regression in existing models - Mitigation: Test with gpt5-mini before merging

### Milestone 5: Integration Testing & Documentation
**Goal:** Verify end-to-end functionality and update docs
**Estimated:** 50 LOC docs + manual testing
**Duration:** 0.5 days

**Tasks:**
- Test with real gpt5-1-codex-max on fizzbuzz benchmark (if API access)
- Verify token counting and cost calculations
- Update models.yml notes
- Update design doc with implementation notes
- Move design doc to implemented/

**Acceptance Criteria:**
- [ ] `gpt5-1-codex-max` successfully generates code (manual test)
- [ ] Cost calculations correct in eval report
- [ ] models.yml updated with api_type notes
- [ ] Design doc moved to implemented/v0_5_10/

**Risks:**
- No API access to gpt5-1-codex-max - Mitigation: Verify with mock tests, test with real API when available

## Success Metrics
- Test coverage: >80% for new `internal/openai/` package
- All existing eval tests passing
- All new unit tests passing (>15 test cases)
- Documentation: models.yml updated, design doc complete
- All tests passing: `make test`
- All linting passing: `make lint`

## Dependencies
- None - models.yml already has gpt5-1-codex-max defined
- OPENAI_API_KEY for integration testing (optional for unit tests)

## Open Questions
- Do we have API access to gpt-5.1-codex-max for integration testing?
- Should we add streaming support in this sprint or defer?

## Notes
- Responses API uses `input` instead of `messages` and `developer` role instead of `system`
- gpt5-1-codex-max supports up to 400K context and 24+ hour autonomous operation
- Existing models (gpt5, gpt5-mini, gpt5-1, gpt5-1-instant) continue using Chat Completions
- Only codex models use Responses API for now

## LOC Summary
| Component | Implementation | Tests | Total |
|-----------|---------------|-------|-------|
| `internal/ai/provider.go` | 50 | - | 50 |
| `internal/ai/openai/types.go` | 80 | - | 80 |
| `internal/ai/openai/client.go` | 100 | 70 | 170 |
| `internal/ai/openai/chat.go` | 100 | 50 | 150 |
| `internal/ai/openai/handler.go` | 50 | - | 50 |
| `internal/ai/openai/responses.go` | 200 | 100 | 300 |
| `api_openai.go` refactor | -50 | - | -50 |
| `ai_handlers.go` (delete OpenAIHandler) | -60 | - | -60 |
| **Total** | **470** | **220** | **690** |

## Benefits to AILANG Programs

After this feature, AILANG programs using `--ai gpt*` will benefit from:
- **Codex models**: Access to `gpt5-1-codex-max` for autonomous tasks
- **Reasoning tokens**: Better cost tracking for reasoning-heavy models
- **Consistent API**: Same module powers both eval harness and CLI
- **Future features**: Easier to add streaming, tool use, etc.
