# M-GAME-E2: AI Effect (General-Purpose AI Oracle)

**Status**: Planned
**Target**: v0.5.1
**Priority**: P1 - High (Contract requirement)
**Estimated**: 3-4 days (~350 LOC)
**Dependencies**: M-GAME-C (Go codegen infrastructure)

## AI-First Alignment Check

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | + | +1 | Generic string→string, domain types in wrappers |
| Preserve Semantic Clarity | + | +2 | Clear boundary: language ↔ host ↔ AI |
| Increase Determinism | + | +2 | Explicit stub mode, record/replay possible |
| Lower Token Cost | + | +1 | Pluggable handler, no hard-coded logic |
| **Net Score** | | **+6** | **Decision: High priority** |

## Design Philosophy

**This is NOT "just a game NPC brain".**

The AI effect is AILANG's **general-purpose AI oracle** - an opaque, host-provided effect for calling external AI/ML systems. Use cases include:
- Game NPC decision-making (via typed wrappers)
- CLI tools with AI assistance
- Agents written in AILANG calling LLMs
- DSL interpreters with structured prompting
- Data analysis pipelines with AI steps

### What You're Really Introducing

Conceptually, this effect is:

> A non-deterministic string→string oracle, whose concrete semantics are provided by the host and are usually "call an LLM with some JSON and get JSON back".

This fits the existing effect story:
- External, non-pure (like RNG, IO)
- Host-defined (like Net, Env)
- DI-friendly (handlers pluggable at runtime)

### Key Design Decisions

1. **Neutral operation name** - `call`, not `decide` (not game-flavored)
2. **JSON as convention, not law** - Type is `string→string`, JSON is recommended pattern
3. **Explicit stub mode** - Nil handler in prod = error, not silent fallback
4. **Error handling in host** - Keep HTTP/API complexity out of AILANG code
5. **No inspection from AILANG** - Telemetry via Debug effect or host-side
6. **Aligned threading** - Same pattern as RNGContext/DebugContext

## Solution Design

### Effect Definition (Core)

```ailang
-- std/ai.ail
-- AI is an opaque, host-provided effect

effect AI {
    call(input: string) -> string
    -- NOTE: "call" is neutral; domain wrappers add semantics
}
```

**Core semantics:**
- `call` takes an arbitrary UTF-8 string, returns an arbitrary UTF-8 string
- By **convention**, libraries use JSON encoding (via `std/json`)
- Language-level semantics do NOT enforce JSON validity or schema
- Host provides the actual implementation (LLM, mock, replay, etc.)

### Host Contract (All Backends)

Every backend must implement:

```
AIHandler Interface:
  - Call(input string) -> (output string, error)

AIContext:
  - handler: AIHandler (nil = error in prod, use stub explicitly)
  - Call(input string) -> (output string, error)
```

**Lifecycle:**
1. Host creates AIContext with explicit handler choice
2. AILANG code calls `AI.call(input)`
3. Host handler executes (LLM API, stub, replay, etc.)
4. Result returned to AILANG code

**Determinism model:**
- AI is a **non-deterministic effect** (like RNG)
- Same inputs may produce different outputs
- For reproducibility, use:
  - `StubAIHandler` (completely deterministic)
  - Record/replay handler (replays captured AI I/O)

### Usage in AILANG

```ailang
import std/ai (AI)
import std/json

-- Game-specific types (NOT part of std/ai)
type NPCContext = {
    position: Vec2,
    health: int,
    nearby_enemies: [Entity],
    nearby_items: [Item]
}

type NPCAction =
  | Move(Vec2)
  | Attack(int)
  | Pickup(int)
  | Wait

-- Domain wrapper: typed interface over generic AI.call
func choose_action(ctx: NPCContext) -> NPCAction ! {AI} {
    let input = json.encode(ctx)
    let output = AI.call(input)
    match json.decode[NPCAction](output) {
        Ok(action) => action,
        Err(_)     => Wait  -- Safe fallback on parse failure
    }
}

-- Usage in game loop
func update_npc(npc: Entity, world: World) -> Entity ! {AI, RNG} {
    let ctx = build_context(npc, world)
    let action = choose_action(ctx)
    apply_action(npc, action)
}
```

**The pattern:**
- `AI.call` is the primitive (string→string)
- Domain code wraps with typed encode/decode
- Decode failures handled in AILANG (safe fallback)

### Non-Game Example: CLI Tool

```ailang
import std/ai (AI)
import std/json

type CodeReviewRequest = {
    filename: string,
    diff: string,
    context: string
}

type CodeReviewResponse = {
    issues: [Issue],
    suggestions: [string]
}

func review_code(req: CodeReviewRequest) -> CodeReviewResponse ! {AI} {
    let input = json.encode(req)
    let output = AI.call(input)
    match json.decode[CodeReviewResponse](output) {
        Ok(resp) => resp,
        Err(_)   => { issues = [], suggestions = ["Parse error in AI response"] }
    }
}
```

### Go Codegen Output

```go
// ai_handler.go (generated)
package game

import "errors"

// AIHandler interface for pluggable AI implementation
type AIHandler interface {
    Call(input string) (string, error)
}

// ErrNoAIHandler is returned when AI.call is invoked without a configured handler
var ErrNoAIHandler = errors.New("AI handler not configured (use StubAIHandler for testing)")

// StubAIHandler returns deterministic placeholder responses
type StubAIHandler struct {
    defaultResponse string
    responses       map[string]string // exact match input → response
}

func NewStubAIHandler() *StubAIHandler {
    return &StubAIHandler{
        defaultResponse: `{"kind":"Wait"}`,
        responses:       make(map[string]string),
    }
}

func (h *StubAIHandler) Call(input string) (string, error) {
    if resp, ok := h.responses[input]; ok {
        return resp, nil
    }
    return h.defaultResponse, nil
}

// SetResponse sets a canned response for an exact input match
func (h *StubAIHandler) SetResponse(input, response string) {
    h.responses[input] = response
}

// SetDefaultResponse sets the fallback for unmatched inputs
func (h *StubAIHandler) SetDefaultResponse(response string) {
    h.defaultResponse = response
}

// AIContext holds the handler for the current execution
type AIContext struct {
    handler AIHandler
}

// NewAIContext creates a context with the given handler
// IMPORTANT: Pass nil only in tests; prod should always have a real handler
func NewAIContext(handler AIHandler) *AIContext {
    return &AIContext{handler: handler}
}

func (c *AIContext) Call(input string) (string, error) {
    if c.handler == nil {
        return "", ErrNoAIHandler
    }
    return c.handler.Call(input)
}
```

### Handler Implementations (Examples)

**1. Stub Handler (explicit opt-in for tests):**
```go
// In test code
handler := game.NewStubAIHandler()
handler.SetResponse(`{"health":50}`, `{"kind":"Pickup","0":5}`)
aiCtx := game.NewAIContext(handler)
```

**2. OpenAI Handler (user implements):**
```go
// real_ai_handler.go (user code, not generated)
type OpenAIHandler struct {
    client *openai.Client
    model  string
    prompt string
}

func (h *OpenAIHandler) Call(input string) (string, error) {
    resp, err := h.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
        Model: h.model,
        Messages: []openai.ChatCompletionMessage{
            {Role: "system", Content: h.prompt},
            {Role: "user", Content: input},
        },
    })
    if err != nil {
        return "", err  // Host decides how to handle
    }
    return resp.Choices[0].Message.Content, nil
}
```

**3. Record/Replay Handler (for reproducible debugging):**
```go
// ReplayAIHandler replays captured AI I/O for reproducibility
type ReplayAIHandler struct {
    log     []AILogEntry
    current int
}

type AILogEntry struct {
    Input  string
    Output string
}

func NewReplayAIHandler(log []AILogEntry) *ReplayAIHandler {
    return &ReplayAIHandler{log: log}
}

func (h *ReplayAIHandler) Call(input string) (string, error) {
    if h.current >= len(h.log) {
        return "", errors.New("replay log exhausted")
    }
    entry := h.log[h.current]
    if entry.Input != input {
        return "", fmt.Errorf("replay mismatch: expected %q, got %q", entry.Input, input)
    }
    h.current++
    return entry.Output, nil
}
```

### Error Handling Strategy

**Two failure domains:**
1. **Handler fails** (network, rate limit, 500, etc.)
2. **Decode fails** (model produced invalid JSON or wrong schema)

**Design decision:** Keep complexity in the host, not AILANG code.

**Recommended patterns:**

**For handler errors:**
```go
// Option A: Handler returns safe fallback + logs error
func (h *ResilientHandler) Call(input string) (string, error) {
    output, err := h.inner.Call(input)
    if err != nil {
        h.debugCtx.Log(fmt.Sprintf("AI call failed: %v", err), "ai_handler.go:42")
        return `{"kind":"Wait"}`, nil  // Safe fallback
    }
    return output, nil
}

// Option B: Let error propagate, handle at top level
// (Requires AILANG to handle Result types in future)
```

**For decode errors (in AILANG):**
```ailang
match json.decode[NPCAction](output) {
    Ok(action) => action,
    Err(_)     => Wait  -- Safe fallback
}
```

### Integration with Game Loop

```go
// cmd/game/main.go
func main() {
    // EXPLICIT handler choice - no silent fallback
    var aiHandler game.AIHandler
    switch os.Getenv("GAME_AI_MODE") {
    case "openai":
        aiHandler = NewOpenAIHandler(os.Getenv("OPENAI_API_KEY"))
    case "local":
        aiHandler = NewOllamaHandler("llama3")
    case "stub":
        aiHandler = game.NewStubAIHandler()
    default:
        log.Fatal("GAME_AI_MODE must be set to: openai, local, or stub")
    }

    // Create contexts (aligned threading pattern)
    aiCtx := game.NewAIContext(aiHandler)
    debugCtx := game.NewDebugContext()
    rngCtx := game.NewRNGContext(42)

    // Run game loop
    world := game.InitWorld(42, rngCtx)
    for tick := 0; tick < 1000; tick++ {
        debugCtx.Reset()
        debugCtx.SetTimestamp(int64(tick))

        input := captureInput()
        world, output, err := game.Step(world, input, aiCtx, debugCtx, rngCtx)
        if err != nil {
            log.Fatal(err)
        }

        // Optional: Log AI calls to Debug for replay
        debugData := debugCtx.Collect()
        render(output)
    }
}
```

### Cross-Backend Contract

**Core contract (all backends):**
- `effect AI { call : string -> string }` exists at language level
- Any backend may support AI by providing a handler

**Interpreter:**
- Simple stub implementation (configurable response)
- Optional CLI flag for real handler (e.g., HTTP to sidecar)
- `ailang run --ai-handler=stub|http://localhost:8080`

**Go codegen:**
- `AIHandler` interface + `AIContext` (as shown above)
- Threaded alongside `RNGContext` and `DebugContext`

**Future WASM/JS backend:**
- Expose JS callback hook; same semantics

### Integration with Debug Effect

AI calls can be logged via Debug for telemetry/replay:

```go
// In a logging wrapper handler
type LoggingAIHandler struct {
    inner    game.AIHandler
    debugCtx *game.DebugContext
}

func (h *LoggingAIHandler) Call(input string) (string, error) {
    h.debugCtx.Log(fmt.Sprintf("AI.call input: %s", input), "ai")
    output, err := h.inner.Call(input)
    if err != nil {
        h.debugCtx.Log(fmt.Sprintf("AI.call error: %v", err), "ai")
        return "", err
    }
    h.debugCtx.Log(fmt.Sprintf("AI.call output: %s", output), "ai")
    return output, nil
}
```

This enables:
- Log all AI inputs/outputs into DebugOutput
- Build replay handler from captured logs
- Debug "the AI chose something weird" scenarios

### Implementation Plan

**Milestone E2.1: Effect System (1 day)**
- [ ] Define `AI` effect in `internal/effects/ai.go`
- [ ] Register builtin: `_ai_call`
- [ ] Implement interpreter support with configurable stub
- [ ] Add `std/ai.ail` module

**Milestone E2.2: Go Codegen (1.5 days)**
- [ ] Generate `ai_handler.go` with interface
- [ ] Generate `StubAIHandler` with SetResponse
- [ ] Generate `AIContext` with nil check
- [ ] Thread `AIContext` through generated functions (same pattern as Debug/RNG)

**Milestone E2.3: Testing & Docs (1 day)**
- [ ] Add tests for stub handler
- [ ] Add tests for nil handler error
- [ ] Document handler interface
- [ ] Add example to sim_stub
- [ ] Update consumer contract

### Files to Modify/Create

**New files:**
- `internal/effects/ai.go` - Effect definition (~80 LOC)
- `internal/gen/golang/ai.go` - Go codegen for AI (~200 LOC)
- `std/ai.ail` - Standard library module (~15 LOC)

**Modified files:**
- `internal/builtins/spec.go` - Register AI builtin
- `examples/sim_stub/world.ail` - Add AI usage example
- `examples/sim_stub/impl.go` - Add stub handler usage

### Testing Strategy

**Unit tests:**
- Stub handler returns configured response
- Custom responses can be set per input
- Nil handler returns ErrNoAIHandler
- Default response works for unmatched inputs

**Integration tests:**
- AI effect works in interpreter
- Go codegen produces valid code
- Handler can be swapped at runtime
- sim_stub example demonstrates usage

**Replay tests:**
- Captured log can be replayed exactly
- Mismatch detection works

### Success Criteria

- [ ] `AI.call(input)` works in interpreter
- [ ] Operation named `call`, not `decide`
- [ ] Stub handler is explicit opt-in
- [ ] Nil handler = error (no silent fallback)
- [ ] Generated Go code compiles
- [ ] Handler interface is pluggable
- [ ] Threading matches Debug/RNG pattern
- [ ] Documentation complete
- [ ] Example in sim_stub
- [ ] Consumer contract updated to ✅

## Future Enhancements

### Higher-Level std/ai Library (v0.6.0+)

Built on top of primitive `call`:

```ailang
-- std/ai/chat.ail (future)
type ChatMessage = { role: string, content: string }
type ChatResponse = { content: string, finish_reason: string }

func callChat(messages: [ChatMessage]) -> ChatResponse ! {AI} {
    let input = json.encode({ messages = messages })
    let output = AI.call(input)
    json.decode[ChatResponse](output)
}

-- std/ai/embed.ail (future)
func embed(text: string) -> [float] ! {AI} {
    let input = json.encode({ text = text, mode = "embed" })
    let output = AI.call(input)
    json.decode[[float]](output)
}
```

### Other Future Work

- Batched calls for multiple NPCs
- Caching layer for repeated contexts
- Async/streaming responses
- Cost tracking (via Debug telemetry)
- Model selection at runtime
- Fine-tuning data collection (JSONL export from Debug logs)

## Security Considerations

- AI handler runs outside AILANG sandbox
- Input sanitization is caller's responsibility
- Rate limiting should be in handler
- API keys managed externally (env vars, not in AILANG code)
- No prompt injection protection at language level (handler responsibility)

---

**Document created**: 2025-12-02
**Last updated**: 2025-12-02 (incorporated architectural feedback)

## Appendix: Why `call` Not `decide`?

**"decide" is game-flavored:**
```ailang
AI.decide(input)  -- Implies NPC decision-making
```

**"call" is neutral:**
```ailang
AI.call(input)  -- Just "call the AI oracle"

-- Game code wraps it:
func choose_action(ctx) = AI.call(json.encode(ctx))

-- CLI tool wraps it differently:
func review_code(diff) = AI.call(json.encode({diff = diff}))

-- Agent wraps it:
func plan_next_step(state) = AI.call(json.encode(state))
```

The primitive stays generic; domain code adds semantics.

## Appendix: Stub vs Nil Handler

**Why not default to stub?**

```go
// ❌ DANGEROUS: Silent fallback hides misconfiguration
func NewAIContext(handler AIHandler) *AIContext {
    if handler == nil {
        handler = NewStubAIHandler()  // Silent!
    }
    return &AIContext{handler: handler}
}
```

**Problems:**
1. Prod runs with stub because someone forgot to set GAME_AI_MODE
2. NPCs all just "Wait" - hard to debug
3. Violates "No Silent Fallbacks" principle

**Correct approach:**
```go
// ✅ SAFE: Explicit failure, explicit stub
func (c *AIContext) Call(input string) (string, error) {
    if c.handler == nil {
        return "", ErrNoAIHandler  // Loud!
    }
    return c.handler.Call(input)
}

// In test code: explicit stub
aiCtx := game.NewAIContext(game.NewStubAIHandler())

// In prod code: must configure
aiCtx := game.NewAIContext(NewOpenAIHandler(apiKey))
```
