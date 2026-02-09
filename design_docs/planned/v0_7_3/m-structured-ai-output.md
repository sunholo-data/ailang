# M-STRUCTURED-AI-OUTPUT: Structured output in std/ai

**Status:** Planned
**Version:** v0.7.3
**Priority:** P2 — eliminates JSON parsing hacks in AI-heavy applications
**Estimated effort:** 10-14 hours
**Origin:** DocParse feature request + DX Feedback (Feb 2026)
**Dependencies:** None (all provider infrastructure exists; Json type already in std/json)

## Problem

`std/ai` currently provides only `call(prompt: string) -> string ! {AI}` — unstructured text output. When applications need structured data from AI (JSON, typed objects), they must:

1. Prompt the AI to return JSON
2. Parse the free-text response
3. Strip markdown code fences (AI often wraps JSON in ````json ... ````)
4. Handle truncation, malformed JSON, extra text outside JSON
5. Validate the schema manually

DocParse has a hand-rolled `stripCodeFences` utility and multiple `decode` calls that fail silently when AI returns unexpected formats. This is fragile and defeats AILANG's "deterministic, verifiable" philosophy.

## Current Architecture

The AI effect call path is:

```
AILANG: call(input)                  [std/ai.ail:40]
  → _ai_call builtin                [internal/builtins/ai.go:83]
  → effects.Call(ctx, "AI", "call") [internal/effects/ai.go:93-128]
  → AIContext.Call(input string)     [internal/effects/ai.go:48-53]
  → AIHandler.Call(input string)     [AIHandler interface, line 25]
  → Handler.Call()                   [internal/ai/handler.go:60-71]
  → Provider.Generate(Request)       [internal/ai/provider.go:62-65]
  → HTTP POST to provider API
  → Response.Text extracted          [all providers return string only]
```

**Key interfaces:**

```go
// internal/effects/ai.go:24-26
type AIHandler interface {
    Call(input string) (string, error)
}

// internal/ai/provider.go:19-37
type Request struct {
    Model        string
    SystemPrompt string
    UserPrompt   string
    MaxTokens    int
    Temperature  float64
    Options      map[string]any  // EXISTS but unused by any provider
}
```

**Critical finding:** `Request.Options` field exists but is ignored by all four providers. This is the natural extension point for structured output configuration.

**Provider structured output support (in their APIs, not in our code):**

| Provider | Supports | API Field |
|----------|----------|-----------|
| Gemini | Yes | `generationConfig.responseMimeType`, `generationConfig.responseSchema` |
| OpenAI | Yes (GPT-4+) | `response_format: {type: "json_schema", json_schema: {...}}` |
| Claude | Yes (tool use) | Single tool with required `tool_choice` |
| Ollama | Partial | `format: "json"` (no schema enforcement) |

**None of our provider implementations currently use these fields.**

## Axiom Compliance

| Axiom | Score | Rationale |
|-------|-------|-----------|
| A1: Determinism | +1 | Schema enforcement makes AI output more deterministic |
| A3: Effect Legibility | 0 | Same AI effect, just different output format |
| A4: Explicit Authority | 0 | Same capability requirements |
| A7: Machines First | +1 | Structured output is inherently more machine-processable |
| A8: Minimal Syntax | 0 | No new syntax — just new builtin functions |
| A9: Cost Visibility | +1 | Same budget cost per call; schema validation prevents wasted retries |
| A10: Composability | +1 | Json return type composes with std/json accessors directly |
| A11: Structured Failure | +1 | Schema validation failures are typed errors, not silent |

**Net score: +5.** Strong alignment. No axiom violations.

## Proposed API

### Option A: `callJson` — Schema-validated JSON response (Recommended for v0.7.3)

```ailang
import std/ai (callJson, callJsonSimple)
import std/json (getString, getNumber, getArray)

-- With schema: provider enforces response format
let blocks = callJson(
  "Extract content blocks from this document: " ++ data,
  "{\"type\": \"array\", \"items\": {\"type\": \"object\", \"properties\": {\"type\": {\"type\": \"string\"}, \"text\": {\"type\": \"string\"}}}}"
);
-- blocks is a Json value, guaranteed valid JSON matching schema

-- Without schema: just guarantees valid JSON (no code fences, no truncation)
let result = callJsonSimple("Return the top 3 programming languages as a JSON array");
-- result is a Json value, guaranteed valid JSON
```

Signatures:
```ailang
export func callJson(prompt: string, schema: string) -> Json ! {AI}
export func callJsonSimple(prompt: string) -> Json ! {AI}
```

### Why not Option B (records) or Option C (type reflection)

**Option B** (`callWithOptions` with record arg) adds complexity — record-typed optional configuration requires the type checker to handle optional fields, and the ergonomics are worse than positional args for 2 params.

**Option C** (`callTyped[Block](prompt)` with auto-schema from AILANG types) is the ideal long-term solution but requires type reflection (`reflect(typeOf(f))`), which is a v0.9.0+ feature. Schema generation from ADT definitions would be powerful but is currently impossible.

**Decision:** Option A for v0.7.3. Simple, implementable now, addresses the pain point.

## Design Decisions

### Return type: `Json` (not `string`)

Returning `Json` instead of `string` eliminates the entire `decode` → `stripCodeFences` → `decode` again pattern. Users get a typed value they can immediately destructure with `getString`, `getNumber`, etc.

The `Json` ADT is already defined in `std/json.ail:11-17`:
```ailang
type Json = JNull | JBool(bool) | JNumber(float) | JString(string)
           | JArray(List[Json]) | JObject(List[{key: string, value: Json}])
```

The builtin implementation will:
1. Send request with structured output config to provider
2. Receive string response
3. Parse JSON using existing `_json_decode` builtin (Go `encoding/json`)
4. Return parsed `Json` value

If parsing fails → return error (no silent fallback). The provider API should guarantee valid JSON, but we validate defensively.

### Schema format: JSON Schema string

Accept JSON Schema as a raw string. This maps 1:1 to what every provider API accepts. No AILANG-side schema type system needed for v0.7.3.

The schema string is passed through to the provider without modification. Each provider wraps it in their expected format (Gemini: `responseSchema`, OpenAI: `json_schema.schema`, etc.).

### Budget interaction

`callJson` and `callJsonSimple` each consume **exactly 1 budget unit** — same as `call`. One API call = one unit. The structured output configuration doesn't add extra calls.

```ailang
func extractBlocks(data: string) -> Json ! {AI @limit=5} {
  -- Each callJson costs 1 unit
  callJson("Extract blocks: " ++ data, schema)  -- unit 1
}
```

### Error handling

| Scenario | Behavior |
|----------|----------|
| Provider returns valid JSON matching schema | Return `Json` value |
| Provider returns valid JSON NOT matching schema | Provider-dependent (most providers enforce schema) |
| Provider returns invalid JSON | Return error: `"AI returned invalid JSON: ..."` |
| Provider doesn't support structured output | Return error: `"Provider X does not support structured output"` |
| Network/auth error | Same error handling as `call()` |

**No silent fallbacks.** If the provider can't produce valid JSON, the user must know.

### Multimodal input + structured output

These compose naturally. The prompt is still a string (may contain base64 image data). Only the output format changes:

```ailang
let blocks = callJson(
  "Extract content from this image. Image (base64): " ++ imageData,
  blockSchema
);
```

No special handling needed.

## Implementation Plan

### Phase 1: Provider support (4-5 hours)

Add structured output configuration to each provider. This is the bulk of the work.

1. **Request struct** (`internal/ai/provider.go:19-37`): Add fields:
   ```go
   ResponseFormat string // "json" or "" (empty = text)
   ResponseSchema string // JSON Schema string (optional)
   ```

2. **Gemini** (`internal/ai/gemini/types.go:15-29`): Add to `generationConfig`:
   ```go
   ResponseMimeType string          `json:"responseMimeType,omitempty"`
   ResponseSchema   json.RawMessage `json:"responseSchema,omitempty"`
   ```
   Update `internal/ai/gemini/generate.go` to pass through from Request.

3. **OpenAI** (`internal/ai/openai/types.go:5-13`): Add to `chatRequest`:
   ```go
   ResponseFormat *responseFormatObj `json:"response_format,omitempty"`
   ```
   With nested type: `{Type: "json_schema", JsonSchema: {Name: "response", Schema: ...}}`

4. **Anthropic** (`internal/ai/anthropic/client.go:71-103`): Use tool_use pattern:
   - Add single tool with schema as input_schema
   - Set `tool_choice: {type: "tool", name: "respond"}`
   - Extract tool result from response

5. **Ollama** (`internal/ai/ollama/client.go`): Add `format: "json"` to request. No schema validation (Ollama limitation).

6. **Handler** (`internal/ai/handler.go`): Add `CallJson(input, schema string) (string, error)` method that sets ResponseFormat/ResponseSchema on Request.

### Phase 2: Builtin + effect wiring (3-4 hours)

7. **AIHandler interface** (`internal/effects/ai.go:24-26`): Add method:
   ```go
   type AIHandler interface {
       Call(input string) (string, error)
       CallJson(input string, schema string) (string, error)
   }
   ```

8. **AIContext** (`internal/effects/ai.go`): Add `callJson` operation alongside existing `call`

9. **Builtin: `_ai_call_json`** (`internal/builtins/ai.go`): Register with:
   - Module: `std/ai`
   - Arity: 2 (prompt string, schema string)
   - Effect: AI
   - Returns: string (raw JSON — the stdlib wrapper parses to Json)

10. **Builtin: `_ai_call_json_simple`**: Same as above, arity 1, no schema

### Phase 3: stdlib wrapper (1-2 hours)

11. Add to `std/ai.ail`:
    ```ailang
    import std/json (Json, decode)
    import std/result (Result, Ok, Err)

    export func callJson(prompt: string, schema: string) -> Json ! {AI} {
      let raw = _ai_call_json(prompt, schema);
      match decode(raw) {
        Ok(json) => json,
        Err(msg) => _panic("AI returned invalid JSON: " ++ msg)
      }
    }

    export func callJsonSimple(prompt: string) -> Json ! {AI} {
      let raw = _ai_call_json_simple(prompt);
      match decode(raw) {
        Ok(json) => json,
        Err(msg) => _panic("AI returned invalid JSON: " ++ msg)
      }
    }
    ```

12. Register exports in module manifest

### Phase 4: Documentation (2-3 hours)

13. Update teaching prompt with `callJson` examples
14. Create `examples/runnable/structured_ai.ail`
15. Update `std/ai` docstrings
16. Update CHANGELOG.md

## Files to Modify

| File | Change | LOC Est |
|------|--------|---------|
| `internal/ai/provider.go` | Add ResponseFormat, ResponseSchema to Request | +5 |
| `internal/ai/gemini/types.go` | Add responseMimeType, responseSchema to config | +5 |
| `internal/ai/gemini/generate.go` | Pass structured output config | +10 |
| `internal/ai/openai/types.go` | Add response_format struct | +15 |
| `internal/ai/openai/client.go` | Pass response_format in request | +10 |
| `internal/ai/anthropic/client.go` | Tool-use pattern for structured output | +30 |
| `internal/ai/ollama/client.go` | Add format: json | +5 |
| `internal/ai/handler.go` | Add CallJson method | +20 |
| `internal/effects/ai.go` | Add CallJson to AIHandler, add operation | +25 |
| `internal/builtins/ai.go` | Register _ai_call_json, _ai_call_json_simple | +40 |
| `std/ai.ail` | Add callJson, callJsonSimple wrappers | +20 |
| `prompts/v0.7.3.md` | Document structured output | +25 |
| Tests (various) | Provider, builtin, integration | +80 |
| **Total** | | **~290** |

## Risks

1. **Provider variance** — Each provider handles structured output differently (Gemini: native, OpenAI: response_format, Anthropic: tool_use, Ollama: format flag). Mitigation: abstracted behind Handler.CallJson; provider differences hidden.
2. **Anthropic tool_use complexity** — Claude's structured output requires creating a fake "tool" with the schema. Response extraction differs from text responses. Mitigation: well-tested, established pattern in Anthropic's documentation.
3. **Schema validation inconsistency** — Some providers enforce schema strictly (Gemini), others don't (Ollama). Mitigation: document per-provider behavior; our `decode()` call validates at minimum that it's valid JSON.
4. **Json type dependency** — `callJson` returns `Json` from `std/json`. The builtin must know how to construct Json values. Mitigation: builtin returns raw string; stdlib wrapper does the `decode()`.
5. **Budget interaction** — Users must understand callJson = 1 budget unit. Mitigation: document clearly; same behavior as `call()`.

## Acceptance Criteria

- [ ] `callJsonSimple("Return [1,2,3] as JSON")` returns `JArray([JNumber(1), JNumber(2), JNumber(3)])`
- [ ] `callJson(prompt, schema)` sends schema to provider
- [ ] Gemini: responseMimeType and responseSchema set in generationConfig
- [ ] OpenAI: response_format.json_schema set in request
- [ ] Invalid JSON response produces clear error (not silent fallback)
- [ ] Effect budget counts `callJson` as one AI call
- [ ] Works with `--ai-stub` (returns valid JSON stub)
- [ ] Multimodal prompts work with structured output
- [ ] Teaching prompt updated
- [ ] Example file created and verified
- [ ] CHANGELOG.md updated

## Future Work (not v0.7.3)

- **Option C: `callTyped[T](prompt)`** — auto-generate JSON Schema from AILANG ADT type. Requires type reflection (v0.9.0+).
- **Streaming structured output** — incremental JSON parsing for large responses
- **Schema validation at AILANG level** — validate response against schema before returning, with typed schema mismatch errors
- **Schema builder DSL** — AILANG functions to build JSON Schema programmatically instead of raw strings

## Related Documents

- [M-EFFECTFUL-LIST-COMBINATORS](m-effectful-list-combinators.md) — `mapE` with `callJson` for batch document processing
- DocParse feature request (msg f9513d77) — original request
- DocParse DX Feedback (msg 6ff4f02e) — stripCodeFences hack
- `std/ai.ail` (current) — 41 lines, string-only interface
- `std/json.ail` — Json ADT definition and accessors

## References

- Gemini API: `generationConfig.responseMimeType` + `responseSchema`
- OpenAI Structured Outputs: `response_format.json_schema`
- Anthropic Tool Use for structured output: single-tool pattern
- AILANG AI handler: `internal/effects/ai.go` (AIHandler interface)
- AILANG AI builtins: `internal/builtins/ai.go` (_ai_call registration)
- AILANG provider interface: `internal/ai/provider.go` (Request.Options unused)
