# M-GEMINI-INTERACTIONS: Google Gemini Interactions API Support

**Status**: Planned
**Target**: v0.6.0 (postponed from v0.5.10)
**Priority**: P1 (Medium)
**Estimated**: 4 days (~16-20 hours)
**Parent**: [M-UNIFIED-AI-PROVIDERS](./m-unified-ai-providers.md)
**Dependencies**: Core infrastructure from parent doc

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | External API calls remain non-deterministic |
| A2: Replayability | +1 | interaction_id enables conversation replay |
| A3: Effect Legibility | 0 | AI effect already explicit in system |
| A4: Explicit Authority | 0 | No capability changes |
| A5: Bounded Verification | 0 | External dependency |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | Unified module simplifies AI provider integration |
| A8: Minimal Syntax | 0 | No syntax changes |
| A9: Cost Visibility | +1 | Server-side context reduces token costs |
| A10: Composability | +1 | Composes with unified internal/ai/ package |
| A11: Structured Failure | +1 | Consistent error handling across APIs |
| A12: System Boundary | +1 | Clear API boundary between generate and interactions |

**Net Score: +6** ✅ Proceed to implementation

### Hard Violation Check

- [x] A1 (Determinism): No additional nondeterminism (external API is inherently non-deterministic)
- [x] A3 (Effects): AI effect already required for calls
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Simplifies programmatic AI access

## Problem Statement

Google's new [Interactions API](https://ai.google.dev/gemini-api/docs/interactions) provides a unified interface for Gemini models with stateful conversation management, tool orchestration, and background task support. AILANG currently uses the older `generateContent` API in two places.

**Current State:**
- **Eval harness** uses `generateContent` via Vertex AI (`internal/eval_harness/api_google.go`)
- **CLI `--ai` flag** uses `generateContent` via AI Studio (`cmd/ailang/ai_handlers.go:182`)
- Both are stateless: full context must be sent with each request
- No support for tool orchestration or background execution
- Works but doesn't leverage new capabilities

**Impact:**
- Missing access to Gemini's new agentic features
- No server-side conversation state (potential token cost increase)
- Cannot benchmark Deep Research agent or tool-using workflows
- AILANG programs using `--ai gemini-*` don't benefit from new API features
- Parity gap with OpenAI Responses API support (also planned for v0.5.10)

## Goals

**Primary Goal:** Create a unified Gemini client module supporting both `generateContent` and Interactions APIs, enabling future agentic eval support.

**Success Metrics:**
- All existing Gemini models (gemini-2.5-pro, gemini-2.5-flash, gemini-3-pro) continue working
- New `internal/gemini/` package with clean API
- Interactions API support for future agent benchmarks
- Test coverage >80% for new package
- No regression in existing eval results

## Solution Design

### Overview

Create `internal/gemini/` package that abstracts Gemini API calls, similar to the `internal/openai/` package for OpenAI. The package will:
1. Support both `generateContent` (current) and Interactions API (new)
2. Auto-detect which API to use based on model configuration
3. Maintain backward compatibility with current eval harness

### Architecture

**Unified `internal/ai/` package** - shared by eval harness, CLI `--ai` flag, and std/ai effect:

```
internal/ai/
├── provider.go              # Common Provider interface
├── gemini/                  # This design doc
│   ├── client.go            # HTTP client with auth
│   ├── generate.go          # generateContent API
│   ├── interactions.go      # Interactions API (new)
│   └── handler.go           # implements effects.AIHandler
├── openai/                  # See M-OPENAI-RESPONSES-API
│   └── ...
└── anthropic/               # Existing, to be migrated
    └── ...
```

**Components:**
1. **Client** (`client.go`): HTTP client with auth handling, API routing
2. **GenerateContent** (`generate.go`): Current `generateContent` API implementation
3. **Interactions** (`interactions.go`): New Interactions API implementation
4. **Handler** (`handler.go`): Implements `effects.AIHandler` for std/ai integration
5. **Auth**: Google auth via ADC or API key (context-dependent)

### API Comparison

| Feature | generateContent | Interactions API |
|---------|-----------------|------------------|
| Endpoint | `/v1/models/{model}:generateContent` | `/v1beta/interactions` |
| State | Stateless | Stateful (server-managed) |
| Input format | `contents[]` array | `input` (string/array) |
| Session tracking | Manual | `previous_interaction_id` |
| Tool support | `tools[]` | `tools[]` + built-in tools |
| Background tasks | No | `background: true` |
| Storage | No | 55 days (paid) / 1 day (free) |

### Implementation Plan

**Phase 1: Core Module & Types** (~4 hours)
- [ ] Create `internal/gemini/types.go` with Request/Response structs
- [ ] Create `internal/gemini/client.go` with Client struct
- [ ] Extract `internal/gemini/auth.go` from api_google.go
- [ ] Add basic unit tests for auth and client construction

**Phase 2: GenerateContent Extraction** (~4 hours)
- [ ] Create `internal/gemini/generate.go` with generateContent logic
- [ ] Extract and refactor from `internal/eval_harness/api_google.go`
- [ ] Ensure backward compatibility with current behavior
- [ ] Add mock HTTP tests

**Phase 3: Interactions API Implementation** (~6 hours)
- [ ] Create `internal/gemini/interactions.go`
- [ ] Implement `/v1beta/interactions` endpoint format
- [ ] Handle `input` formats (string, content objects, multi-turn)
- [ ] Support `previous_interaction_id` for stateful conversations
- [ ] Parse response with `outputs[]` array
- [ ] Add comprehensive mock tests

**Phase 4: Integration (Eval Harness + CLI)** (~4 hours)
- [ ] Update `internal/eval_harness/api_google.go` to use new module
- [ ] Update `cmd/ailang/ai_handlers.go` `GoogleHandler` to use new module
- [ ] Add `api_type` flag to models.yml for API selection
- [ ] Test with existing Gemini models (both eval and `--ai` flag)
- [ ] Verify no regression in eval results or CLI behavior

**Phase 5: Documentation & Testing** (~2 hours)
- [ ] Update models.yml with Interactions API notes
- [ ] Add integration test (manual, requires API access)
- [ ] Move design doc to implemented/
- [ ] Update CHANGELOG.md

### Files to Modify/Create

**New files (unified `internal/ai/` package):**
- `internal/ai/provider.go` - Common Provider interface (~50 LOC)
- `internal/ai/gemini/types.go` - Request/Response types (~100 LOC)
- `internal/ai/gemini/client.go` - Client struct and routing (~120 LOC)
- `internal/ai/gemini/generate.go` - generateContent implementation (~100 LOC)
- `internal/ai/gemini/interactions.go` - Interactions API (~150 LOC)
- `internal/ai/gemini/handler.go` - effects.AIHandler impl (~60 LOC)
- `internal/ai/gemini/client_test.go` - Unit tests (~250 LOC)

**Modified files:**
- `internal/eval_harness/api_google.go` - Use `internal/ai/gemini` (~-80 LOC)
- `internal/eval_harness/models.yml` - Add api_type flags (~+10 LOC)
- `cmd/ailang/ai_handlers.go` - Use `internal/ai/gemini` (~-70 LOC, delete GoogleHandler)

**Total: ~780 LOC new + significant cleanup**

### Benefits to AILANG Programs

After this feature, AILANG programs using `--ai gemini-*` will benefit from:
- **Stateful conversations**: Multi-turn context without manual history management
- **Future tool support**: Enable AI-assisted AILANG programs with web search, code execution
- **Consistent API**: Same module powers both eval harness and CLI

## Examples

### Example 1: Basic generateContent (current behavior)

**Before (api_google.go):**
```go
func (a *AIAgent) callGemini(ctx context.Context, prompt string) (*GenerateResult, error) {
    accessToken, err := getGoogleAccessToken()
    // ... inline auth and HTTP handling ...
    req := geminiRequest{
        Contents: []geminiContent{{Role: "user", Parts: []geminiPart{{Text: fullPrompt}}}},
    }
    // ... send to generateContent endpoint ...
}
```

**After (using internal/gemini):**
```go
func (a *AIAgent) callGemini(ctx context.Context, prompt string) (*GenerateResult, error) {
    client, err := gemini.NewClient(ctx)
    if err != nil {
        return nil, err
    }

    resp, err := client.Generate(ctx, a.model, &gemini.GenerateRequest{
        SystemPrompt: "You are a programming assistant...",
        UserPrompt:   prompt,
    })
    if err != nil {
        return nil, err
    }

    return &GenerateResult{
        Code:         extractCodeFromMarkdown(resp.Text),
        InputTokens:  resp.InputTokens,
        OutputTokens: resp.OutputTokens,
        Model:        a.model,
    }, nil
}
```

### Example 2: Interactions API (future agentic usage)

```go
// Stateful multi-turn conversation with tool use
client, _ := gemini.NewClient(ctx)

// First interaction
resp1, _ := client.CreateInteraction(ctx, &gemini.InteractionRequest{
    Model: "gemini-2.5-flash",
    Input: "What's the AILANG syntax for defining an ADT?",
    Tools: []gemini.Tool{{Type: "google_search"}},
})

// Follow-up (server maintains context)
resp2, _ := client.CreateInteraction(ctx, &gemini.InteractionRequest{
    Model:                 "gemini-2.5-flash",
    Input:                 "Show me an example with pattern matching",
    PreviousInteractionID: resp1.ID, // Server-side context!
})
```

### Example 3: models.yml configuration

```yaml
gemini-2-5-pro:
  api_name: "gemini-2.5-pro"
  provider: "google"
  api_type: "generate"  # Use generateContent (default)

gemini-3-agentic:       # Future agentic model
  api_name: "gemini-3-agentic"
  provider: "google"
  api_type: "interactions"  # Use Interactions API
```

## Success Criteria

- [ ] All existing Gemini models work without changes (backward compatible)
- [ ] New `internal/gemini/` package with >80% test coverage
- [ ] `generateContent` extracted and working via new module
- [ ] Interactions API implementation with mock tests
- [ ] `make test` passes (no regressions)
- [ ] `make lint` clean
- [ ] models.yml updated with api_type documentation
- [ ] Design doc moved to implemented/

## Testing Strategy

**Unit tests:**
- Client construction and auth
- Model detection (generate vs interactions)
- Request/Response marshaling
- Mock HTTP responses for both APIs

**Integration tests:**
- Manual test with real Gemini API (requires ADC setup)
- Run existing benchmark with gemini-2.5-flash
- Verify token counts and costs correct

**Manual testing:**
- `ailang eval-suite --models gemini-2-5-flash` produces same results
- No regression in eval baseline scores

## Non-Goals

**Not in this feature:**
- Tool orchestration in eval harness - Future work, needs framework design
- Background/long-running tasks - Complex, defer to agent eval phase
- Deep Research agent support - Needs tool framework first
- Streaming responses - Not needed for eval harness (batch mode)
- AI Studio API support - Focus on Vertex AI (ADC auth)

## Timeline

**Day 1** (4 hours):
- Phase 1: Core Module & Types
- Basic client structure and auth extraction

**Day 2** (4 hours):
- Phase 2: GenerateContent Extraction
- Backward compatibility verification

**Day 3** (6 hours):
- Phase 3: Interactions API Implementation
- Mock tests for new endpoint

**Day 4** (4-6 hours):
- Phase 4: Eval Harness Integration
- Phase 5: Documentation & Testing

**Total: ~16-20 hours across 4 days**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Interactions API is beta (subject to changes) | Med | Abstract behind interface, easy to update |
| No local testing without API access | Med | Comprehensive mock tests |
| Auth differences between APIs | Low | Extract auth module, share between both |
| Breaking existing eval results | High | Run full baseline before/after, compare |
| Vertex AI vs AI Studio endpoint differences | Low | Focus on Vertex AI only (documented) |

## References

- [Gemini Interactions API Docs](https://ai.google.dev/gemini-api/docs/interactions)
- [M-OPENAI-RESPONSES-API design doc](./m-openai-responses-api-sprint.md) - Parallel feature
- [Current api_google.go](../../internal/eval_harness/api_google.go)
- [models.yml](../../internal/eval_harness/models.yml)

## Future Work

Features that build on this but are out of scope for now:
- **Tool Framework**: Generic tool orchestration for eval benchmarks
- **Agent Eval Mode**: Multi-turn agentic benchmarks using Interactions API
- **Deep Research**: Benchmark Google's Deep Research agent
- **Streaming**: Add streaming support for interactive use cases
- **Session Management**: Persistent interaction chains for long-running evals

---

**Document created**: 2025-12-11
**Last updated**: 2025-12-11
