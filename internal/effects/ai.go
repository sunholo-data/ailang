package effects

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sunholo-data/ailang/internal/ai"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/trace"
)

// ErrNoAIHandler is returned when AI.call is invoked without a configured handler
var ErrNoAIHandler = errors.New("no AI model configured — add --ai <model> flag (e.g. --ai gemini-2-5-flash), or --ai-stub for testing")

// AIHandler interface for pluggable AI implementation
//
// The AI effect is AILANG's general-purpose AI oracle - an opaque, host-provided
// effect for calling external AI/ML systems. Use cases include:
//   - Game NPC decision-making (via typed wrappers)
//   - CLI tools with AI assistance
//   - Agents written in AILANG calling LLMs
//   - Data analysis pipelines with AI steps
//
// The interface is intentionally simple: string → string
// By convention, JSON is used for input/output, but this is not enforced.
type AIHandler interface {
	Call(input string) (string, error)
	// CallJson sends a request configured for structured JSON output.
	// If schema is non-empty, providers enforce the schema.
	// Returns raw JSON string (caller parses to Json ADT).
	CallJson(input string, schema string) (string, error)
	// CallImage generates an image and writes it to outputPath.
	// Options is a JSON string with optional fields: aspect_ratio, mime_type.
	// Returns the output path on success.
	CallImage(prompt string, outputPath string, options string) (string, error)
	// CallImageBase64 generates an image and returns it as a JSON string
	// containing base64-encoded data: {"base64": "...", "mime_type": "image/png"}.
	CallImageBase64(prompt string, options string) (string, error)

	// Step is the multi-turn / tool-aware completion entry point introduced
	// by M-AI-TOOL-LOOP (v0.17.0). Unlike Call/CallJson, it takes a
	// conversation (messages) plus an optional tool catalog (tools), and
	// returns an *ai.Response carrying assistant Text + ToolCalls +
	// FinishReason. On failure, returns a plain error which the caller
	// (the aiStep effect op) classifies via ai.ClassifyError into a typed
	// *ai.AIError before returning Err(AIError record) to AILANG.
	//
	// Model is passed explicitly because Step is per-call routable (unlike
	// Call/CallJson which use the handler's bound model).
	//
	// Reuses ai.Message / ai.ToolSchema / ai.Response (defined in
	// internal/ai during M1) to avoid wire-type duplication.
	Step(model string, messages []ai.Message, tools []ai.ToolSchema) (*ai.Response, error)

	// StepWithCache is the cache-aware variant introduced by
	// M-AI-PROMPT-CACHING (v0.18.4). Same contract as Step but with an
	// extra slice of opt-in CacheBreakpoint hints that providers interpret
	// per their own caching contract — empty slice = identical behavior to
	// Step, bit-for-bit identical wire shape.
	//
	// Stubs and test handlers may delegate to Step (cache hints are pure
	// telemetry/cost optimization, not behaviorally observable). The real
	// ai.Handler propagates breakpoints into ai.Request.CacheBreakpoints
	// so the per-provider step.go can act on them.
	StepWithCache(model string, messages []ai.Message, tools []ai.ToolSchema, cacheBreakpoints []ai.CacheBreakpoint) (*ai.Response, error)
}

// AIHandlerWithRouting is an optional capability — handlers that implement
// it can report routing metadata about their most recent Call/CallJson.
//
// The unified ai.Handler in internal/ai implements this and surfaces
// OpenRouter resolution data (requested vs resolved model, fallback
// chain, cost, cached tokens) so the AI effect ops can attach it to
// trace events.
//
// Thread-safety: callers must invoke LastRoutingMetadata() immediately
// after the matching Call/CallJson returns, before any other handler
// operation. Single-threaded use only — matches the AIHandler contract.
type AIHandlerWithRouting interface {
	AIHandler
	// LastRoutingMetadata returns routing info for the most recent
	// Call/CallJson, or nil if the call did not engage routing.
	LastRoutingMetadata() *trace.ResolvedRoute
}

// errAICallFmt is the format string used by all AI op error wrappers.
// Centralized so the literal isn't duplicated across each op.
const errAICallFmt = "E_AI_CALL_ERROR: %w"

// AIContext holds the handler for the current execution
//
// Thread-safety: AIContext is designed for single-threaded use
// within one evaluation. Create a new context for each step/tick.
type AIContext struct {
	handler AIHandler
}

// LastRoutingMetadata returns routing info from the underlying handler
// if it implements AIHandlerWithRouting. Returns nil otherwise (or when
// the handler is nil, or the most recent call didn't engage routing).
func (c *AIContext) LastRoutingMetadata() *trace.ResolvedRoute {
	if c == nil || c.handler == nil {
		return nil
	}
	if rh, ok := c.handler.(AIHandlerWithRouting); ok {
		return rh.LastRoutingMetadata()
	}
	return nil
}

// NewAIContext creates a context with the given handler
//
// IMPORTANT: Pass nil only in tests to verify error handling.
// Production code should always have a real handler or explicit stub.
func NewAIContext(handler AIHandler) *AIContext {
	return &AIContext{handler: handler}
}

// Call invokes the AI handler with the given input
//
// Returns ErrNoAIHandler if no handler is configured.
// This is intentional - no silent fallbacks for AI calls.
func (c *AIContext) Call(input string) (string, error) {
	if c.handler == nil {
		return "", ErrNoAIHandler
	}
	return c.handler.Call(input)
}

// CallJson invokes the AI handler requesting structured JSON output.
// If schema is non-empty, providers enforce the schema on the response.
func (c *AIContext) CallJson(input string, schema string) (string, error) {
	if c.handler == nil {
		return "", ErrNoAIHandler
	}
	return c.handler.CallJson(input, schema)
}

// CallImage generates an image and writes it to outputPath.
func (c *AIContext) CallImage(prompt, outputPath, options string) (string, error) {
	if c.handler == nil {
		return "", ErrNoAIHandler
	}
	return c.handler.CallImage(prompt, outputPath, options)
}

// CallImageBase64 generates an image and returns it as base64 JSON.
func (c *AIContext) CallImageBase64(prompt, options string) (string, error) {
	if c.handler == nil {
		return "", ErrNoAIHandler
	}
	return c.handler.CallImageBase64(prompt, options)
}

// Step is the multi-turn / tool-aware completion entry point — passes
// through to the underlying handler. Introduced by M-AI-TOOL-LOOP (v0.17.0).
// Sentinel error path (no handler) returns ErrNoAIHandler so the calling
// effect op can wrap it as AIError{ProviderNotFound} for the AILANG side.
func (c *AIContext) Step(model string, messages []ai.Message, tools []ai.ToolSchema) (*ai.Response, error) {
	if c.handler == nil {
		return nil, ErrNoAIHandler
	}
	return c.handler.Step(model, messages, tools)
}

// StepWithCache is the cache-aware variant — passes through to the handler.
// Empty cacheBreakpoints behaves bit-for-bit identically to Step.
func (c *AIContext) StepWithCache(model string, messages []ai.Message, tools []ai.ToolSchema, cacheBreakpoints []ai.CacheBreakpoint) (*ai.Response, error) {
	if c.handler == nil {
		return nil, ErrNoAIHandler
	}
	return c.handler.StepWithCache(model, messages, tools, cacheBreakpoints)
}

// StubAIHandler returns deterministic placeholder responses
//
// Use for testing and development. Supports:
//   - Default response for all inputs
//   - Per-input canned responses
type StubAIHandler struct {
	defaultResponse string
	responses       map[string]string // exact match input → response
}

// NewStubAIHandler creates a stub handler with a sensible default
func NewStubAIHandler() *StubAIHandler {
	return &StubAIHandler{
		defaultResponse: `{"kind":"Wait"}`,
		responses:       make(map[string]string),
	}
}

// Call returns the configured response for the input
func (h *StubAIHandler) Call(input string) (string, error) {
	if resp, ok := h.responses[input]; ok {
		return resp, nil
	}
	return h.defaultResponse, nil
}

// CallJson returns valid JSON for structured output requests.
// The stub returns the default response (which is valid JSON).
func (h *StubAIHandler) CallJson(input string, schema string) (string, error) {
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

// CallImage returns a stub image path (writes a minimal 1x1 PNG to disk).
func (h *StubAIHandler) CallImage(prompt, outputPath, options string) (string, error) {
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return "", fmt.Errorf("stub: failed to create directory: %w", err)
	}
	if err := os.WriteFile(outputPath, stubPNG, 0o644); err != nil {
		return "", fmt.Errorf("stub: failed to write image: %w", err)
	}
	return outputPath, nil
}

// CallImageBase64 returns a stub base64 JSON response.
func (h *StubAIHandler) CallImageBase64(prompt, options string) (string, error) {
	b64 := base64.StdEncoding.EncodeToString(stubPNG)
	return fmt.Sprintf(`{"base64":"%s","mime_type":"image/png"}`, b64), nil
}

// Step is the stub multi-turn / tool-aware completion. Mirrors the Call
// stub's deterministic-response behaviour: returns the configured default
// response as Text with FinishReason="stop" and no tool calls. Useful for
// runTools tests where the loop should terminate after one turn.
func (h *StubAIHandler) Step(model string, messages []ai.Message, tools []ai.ToolSchema) (*ai.Response, error) {
	// Synthesize a single text turn from the default response. Tests that
	// need richer behaviour (forced tool_calls, simulated errors) should
	// use a custom handler instead of the stub.
	return &ai.Response{
		Text:         h.defaultResponse,
		FinishReason: "stop",
		Model:        model,
		// Tokens: stubs don't account.
	}, nil
}

// StepWithCache delegates to Step — cache hints are non-behavioral and the
// stub doesn't account tokens, so they have nothing to act on. The slice
// is accepted but ignored.
func (h *StubAIHandler) StepWithCache(model string, messages []ai.Message, tools []ai.ToolSchema, _ []ai.CacheBreakpoint) (*ai.Response, error) {
	return h.Step(model, messages, tools)
}

// stubPNG is a minimal valid 1x1 transparent PNG (67 bytes).
var stubPNG = func() []byte {
	// Minimal 1x1 RGBA PNG
	b, _ := base64.StdEncoding.DecodeString(
		"iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==",
	)
	return b
}()

// init registers AI effect operations
func init() {
	RegisterOp("AI", "call", aiCall)
	RegisterOp("AI", "callJson", aiCallJson)
	RegisterOp("AI", "callJsonSimple", aiCallJsonSimple)
	RegisterOp("AI", "callImage", aiCallImage)
	RegisterOp("AI", "callImageBase64", aiCallImageBase64)
}

// aiCall implements AI.call(input: string) -> string
//
// Invokes the configured AI handler with the input string.
// Returns the handler's response or an error if no handler is configured.
//
// Parameters:
//   - ctx: Effect context
//   - args: [StringValue (input)]
//
// Returns:
//   - StringValue with handler response
func aiCall(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("E_AI_TYPE_ERROR: call: expected 1 argument, got %d", len(args))
	}

	input, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("E_AI_TYPE_ERROR: call: expected string input, got %T", args[0])
	}

	// Get AI context - must be configured
	if ctx.AI == nil {
		return nil, ErrNoAIHandler
	}

	output, err := ctx.AI.Call(input.Value)
	if err != nil {
		return nil, fmt.Errorf(errAICallFmt, err)
	}

	// Record trace event with optional routing metadata. Truncate args/result
	// to keep the trace stream compact (long prompts/responses dominate size).
	ctx.RecordAIEffect("call",
		[]string{truncateForTrace(input.Value)},
		truncateForTrace(output),
		ctx.AI.LastRoutingMetadata(),
	)

	return &eval.StringValue{Value: output}, nil
}

// traceArgMaxLen caps the length of a single trace arg/result string so a
// 100k-token prompt doesn't bloat the trace stream. Matches the rough
// envelope used elsewhere in the trace pipeline.
const traceArgMaxLen = 256

// truncateForTrace shortens long strings for inclusion in a trace event,
// appending an ellipsis marker when the input was clipped.
func truncateForTrace(s string) string {
	if len(s) <= traceArgMaxLen {
		return s
	}
	return s[:traceArgMaxLen] + "...[truncated]"
}

// aiCallJson implements AI.callJson(input: string, schema: string) -> string
func aiCallJson(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("E_AI_TYPE_ERROR: callJson: expected 2 arguments, got %d", len(args))
	}

	input, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("E_AI_TYPE_ERROR: callJson: expected string input, got %T", args[0])
	}

	schema, ok := args[1].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("E_AI_TYPE_ERROR: callJson: expected string schema, got %T", args[1])
	}

	if ctx.AI == nil {
		return nil, ErrNoAIHandler
	}

	output, err := ctx.AI.CallJson(input.Value, schema.Value)
	if err != nil {
		return nil, fmt.Errorf(errAICallFmt, err)
	}

	ctx.RecordAIEffect("callJson",
		[]string{truncateForTrace(input.Value), truncateForTrace(schema.Value)},
		truncateForTrace(output),
		ctx.AI.LastRoutingMetadata(),
	)

	return &eval.StringValue{Value: output}, nil
}

// aiCallJsonSimple implements AI.callJsonSimple(input: string) -> string
func aiCallJsonSimple(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("E_AI_TYPE_ERROR: callJsonSimple: expected 1 argument, got %d", len(args))
	}

	input, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("E_AI_TYPE_ERROR: callJsonSimple: expected string input, got %T", args[0])
	}

	if ctx.AI == nil {
		return nil, ErrNoAIHandler
	}

	output, err := ctx.AI.CallJson(input.Value, "")
	if err != nil {
		return nil, fmt.Errorf(errAICallFmt, err)
	}

	ctx.RecordAIEffect("callJsonSimple",
		[]string{truncateForTrace(input.Value)},
		truncateForTrace(output),
		ctx.AI.LastRoutingMetadata(),
	)

	return &eval.StringValue{Value: output}, nil
}

// aiCallImage implements AI.callImage(prompt: string, output_path: string, options: string) -> string
func aiCallImage(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	if len(args) < 3 {
		return nil, fmt.Errorf("E_AI_TYPE_ERROR: callImage: expected 3 arguments, got %d", len(args))
	}

	prompt, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("E_AI_TYPE_ERROR: callImage: expected string prompt, got %T", args[0])
	}

	outputPath, ok := args[1].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("E_AI_TYPE_ERROR: callImage: expected string output_path, got %T", args[1])
	}

	options, ok := args[2].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("E_AI_TYPE_ERROR: callImage: expected string options, got %T", args[2])
	}

	if ctx.AI == nil {
		return nil, ErrNoAIHandler
	}

	result, err := ctx.AI.CallImage(prompt.Value, outputPath.Value, options.Value)
	if err != nil {
		return nil, fmt.Errorf(errAICallFmt, err)
	}

	return &eval.StringValue{Value: result}, nil
}

// aiCallImageBase64 implements AI.callImageBase64(prompt: string, options: string) -> string
func aiCallImageBase64(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("E_AI_TYPE_ERROR: callImageBase64: expected 2 arguments, got %d", len(args))
	}

	prompt, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("E_AI_TYPE_ERROR: callImageBase64: expected string prompt, got %T", args[0])
	}

	options, ok := args[1].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("E_AI_TYPE_ERROR: callImageBase64: expected string options, got %T", args[1])
	}

	if ctx.AI == nil {
		return nil, ErrNoAIHandler
	}

	result, err := ctx.AI.CallImageBase64(prompt.Value, options.Value)
	if err != nil {
		return nil, fmt.Errorf(errAICallFmt, err)
	}

	return &eval.StringValue{Value: result}, nil
}
