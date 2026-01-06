# Sprint Plan: M-OPENAI-RESPONSES-API Completion

## Summary
Complete the OpenAI Responses API implementation for codex models (gpt5-1-codex-max).
The infrastructure exists but `generateResponses()` is a stub that falls back to Chat API.

**Duration:** 2 hours (~200 LOC)
**Risk Level:** Low (well-understood API, existing patterns to follow)

## API Differences: Chat Completions vs Responses

| Aspect | Chat Completions | Responses API |
|--------|------------------|---------------|
| Endpoint | `/v1/chat/completions` | `/v1/responses` |
| Messages | `messages` array | `input` array |
| System role | `system` | `developer` (or `system`) |
| Response | `choices[0].message.content` | `output[]` polymorphic items |
| Reasoning | N/A | `reasoning.effort` (low/medium/high/none) |

## Milestones

### M1: Add Responses API Types (~50 LOC)
**Duration:** 15 min

Add to `internal/ai/openai/types.go`:
- `responsesRequest` struct (input array, model, reasoning)
- `responsesInput` struct (role, content)
- `responsesReasoning` struct (effort parameter)
- `responsesResponse` struct (id, output array, usage)
- `responsesOutputItem` struct (polymorphic: message, reasoning, function_call)
- `responsesContent` struct (type, text)
- `responsesUsage` struct (input_tokens, output_tokens, reasoning_tokens)

**Acceptance Criteria:**
- [ ] All types match OpenAI Responses API docs
- [ ] Types support polymorphic output items
- [ ] `make lint` clean

### M2: Implement generateResponses() (~100 LOC)
**Duration:** 45 min

Replace stub in `internal/ai/openai/responses.go`:
- Build `input` array with developer/user roles
- Set `reasoning.effort` from request options (default: medium)
- POST to `/v1/responses` endpoint
- Parse polymorphic `output` array
- Extract text from message-type items
- Map usage stats to ai.Response

**Acceptance Criteria:**
- [ ] Correct request format for Responses API
- [ ] Handles developer role (maps from SystemPrompt)
- [ ] Parses polymorphic output correctly
- [ ] Reasoning tokens tracked separately
- [ ] Error handling consistent with chat.go
- [ ] `make test` passes

### M3: Add Tests (~50 LOC)
**Duration:** 30 min

Add to `internal/ai/openai/client_test.go`:
- `TestResponsesAPI_BasicRequest` - verify request format
- `TestResponsesAPI_ParseOutput` - polymorphic output parsing
- `TestResponsesAPI_ReasoningTokens` - token counting
- `TestResponsesAPI_Error` - error handling

**Acceptance Criteria:**
- [ ] Mock HTTP tests for success/error cases
- [ ] Verifies request structure matches API spec
- [ ] Tests polymorphic output parsing
- [ ] `make test` passes
- [ ] Coverage >80% for responses.go

### M4: Integration & Docs (~15 min)
**Duration:** 15 min

- Test with `codex` model detection (already implemented in client.go)
- Move design doc to `implemented/v0_6_3/`
- Update CHANGELOG.md

**Acceptance Criteria:**
- [ ] `detectAPIType("gpt5-1-codex-max")` returns `APIResponses`
- [ ] Design doc moved to implemented/
- [ ] CHANGELOG updated

## LOC Summary

| Component | LOC |
|-----------|-----|
| types.go additions | 50 |
| responses.go implementation | 100 |
| client_test.go additions | 50 |
| **Total** | **200** |

## Sources

- [OpenAI Responses API Reference](https://platform.openai.com/docs/api-reference/responses)
- [Why we built the Responses API](https://developers.openai.com/blog/responses-api/)
- [Migration Guide](https://platform.openai.com/docs/guides/migrate-to-responses)
