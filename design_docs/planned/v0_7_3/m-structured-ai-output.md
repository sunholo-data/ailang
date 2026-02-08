# M-STRUCTURED-AI-OUTPUT: Structured output in std/ai

**Status:** Planned
**Version:** v0.7.3
**Priority:** P2 — eliminates JSON parsing hacks in AI-heavy applications
**Estimated effort:** 8-12 hours
**Origin:** DocParse feature request + DX Feedback (Feb 2026)

## Problem

`std/ai` currently provides only `call(prompt: string) -> string ! {AI}` — unstructured text output. When applications need structured data from AI (JSON, typed objects), they must:

1. Prompt the AI to return JSON
2. Parse the free-text response
3. Strip markdown code fences (AI often wraps JSON in ````json ... ````)
4. Handle truncation, malformed JSON, extra text outside JSON
5. Validate the schema manually

DocParse has a `stripCodeFences` utility and multiple `decode` calls that fail silently when AI returns unexpected formats. This is fragile and defeats AILANG's "deterministic, verifiable" philosophy.

## Current State

```ailang
import std/ai (call)
import std/json (decode, getString)

-- What users do today (fragile)
let response = call("Extract content as JSON: " ++ data);
let cleaned = stripCodeFences(response);  -- hand-rolled utility
let parsed = decode(cleaned);             -- may fail on malformed JSON
let title = getString(parsed, "title");   -- may be None
```

## Proposed API

### Option A: `callJson` — Schema-validated JSON response

```ailang
import std/ai (callJson)

-- callJson enforces response_mime_type: application/json
-- and optionally validates against a schema
let blocks = callJson(
  "Extract content blocks from this document",
  "{\"type\": \"array\", \"items\": {\"type\": \"object\", \"properties\": {\"type\": {\"type\": \"string\"}, \"text\": {\"type\": \"string\"}}}}"
);
-- blocks is a Json value (not a string), guaranteed valid JSON matching schema
```

Signature:
```ailang
-- Schema-validated JSON response
export func callJson(prompt: string, schema: string) -> Json ! {AI}

-- Simple JSON response (no schema, just guarantees valid JSON)
export func callJsonSimple(prompt: string) -> Json ! {AI}
```

### Option B: `call` with options record

```ailang
import std/ai (callWithOptions)

let blocks = callWithOptions(
  "Extract content blocks",
  {response_format: "json", schema: schemaStr, temperature: 0.1}
);
```

This is more extensible but requires record-typed options which adds complexity.

### Option C: `callTyped` — AILANG type as schema

```ailang
import std/ai (callTyped)

type Block = TextBlock({text: string}) | HeadingBlock({text: string, level: int})

-- Automatically generates JSON schema from AILANG type definition
let blocks: [Block] = callTyped[Block]("Extract content blocks", data)
```

This is the most ergonomic but requires type reflection (v0.4.0+ feature, not yet available).

**Recommendation:** Option A for v0.7.3. It's implementable now with the existing type system. Option C is the long-term goal for v0.9.0+ when reflection is available.

## Design Decisions

### Return type: `Json` vs `string`

| Return | Pros | Cons |
|--------|------|------|
| `Json` | Type-safe, use `getString`/`getNumber` directly | Requires Json type in scope |
| `string` | Simple, no new types | User still needs to `decode()` |

**Recommendation:** Return `Json`. The whole point is avoiding string parsing. Users already import `std/json` for `decode` — returning `Json` directly saves that step.

### Schema format: JSON Schema string vs AILANG type

For v0.7.3, accept JSON Schema as a string. This maps directly to what Gemini/OpenAI/Claude APIs accept. Type-to-schema conversion is a future enhancement.

### Provider support

| Provider | Structured Output | API Field |
|----------|-------------------|-----------|
| Gemini | Yes | `response_mime_type: "application/json"`, `response_schema` |
| OpenAI | Yes (GPT-4+) | `response_format: {type: "json_schema", json_schema: {...}}` |
| Claude | Yes (tool use) | `tool_choice: {type: "tool"}` with single tool |
| Ollama | Partial | `format: "json"` (no schema validation) |

The builtin implementation should:
1. Set the appropriate provider-specific field
2. Parse the response as JSON before returning
3. Fail with a clear error if the response isn't valid JSON (no silent fallbacks)

### Multimodal + structured output

DocParse sends base64 images to AI for description. The combination of multimodal input + structured output should work:

```ailang
let blocks = callJson(
  "Extract content from this image as structured blocks. Image (base64): " ++ imageData,
  blockSchema
);
```

This requires no special handling — the prompt is still a string, only the output format changes.

## Implementation Plan

### Phase 1: Builtin registration (3-4 hours)
1. Add `_ai_call_json` builtin in `internal/builtins/` with signature `(string, string) -> Json ! {AI}`
2. Add `_ai_call_json_simple` builtin: `(string) -> Json ! {AI}`
3. Register in builtin spec with AI effect requirement
4. Implement in `internal/effects/ai.go` (or similar)

### Phase 2: Provider support (3-4 hours)
5. Update Gemini provider (`internal/ai/gemini/`) — add `response_mime_type` and `response_schema`
6. Update OpenAI provider (`internal/ai/openai/`) — add `response_format`
7. Update Anthropic provider (`internal/ai/anthropic/`) — use tool_use pattern for structured output
8. Update Ollama provider (`internal/ai/ollama/`) — add `format: "json"`
9. Parse JSON response in each provider, return parsed `Json` value

### Phase 3: stdlib wrapper (1-2 hours)
10. Add `callJson` and `callJsonSimple` to `std/ai.ail`
11. Wire to builtins
12. Register exports

### Phase 4: Documentation (2-3 hours)
13. Update teaching prompt with `callJson` examples
14. Create `examples/runnable/structured_ai.ail`
15. Update CHANGELOG.md
16. Add to `std/ai` docs

## Risks

1. **Provider variance** — Each provider handles structured output differently. The builtin must abstract over these differences. Mitigation: provider-specific handling in `internal/ai/*/`.
2. **Schema validation errors** — What happens when the AI returns JSON that doesn't match the schema? Options: retry, return error, return raw JSON with warning. Recommendation: return error (no silent fallbacks).
3. **Json type availability** — The `Json` type must be available from `std/json`. Verify it's exported and usable as a return type from builtins.
4. **Effect budget interaction** — `callJson` should consume the same budget as `call`. One API call = one budget unit.

## Acceptance Criteria

- [ ] `callJsonSimple("Return a JSON array of 3 numbers")` returns a valid `Json` value
- [ ] `callJson(prompt, schema)` validates response against schema
- [ ] Invalid JSON response produces clear error (not silent fallback)
- [ ] Works with Gemini provider
- [ ] Works with OpenAI provider
- [ ] Effect budget counts `callJson` as one AI call
- [ ] Multimodal prompts work with structured output
- [ ] Teaching prompt updated
- [ ] Example file created and verified
- [ ] CHANGELOG.md updated

## References

- DocParse feature request (Feb 2026) — `callJson` for document parsing
- DocParse DX Feedback — `stripCodeFences` hack
- Gemini API structured output: `generationConfig.response_mime_type`
- OpenAI structured outputs: `response_format.json_schema`
- Anthropic tool use for structured output
