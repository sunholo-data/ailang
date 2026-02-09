package effects

import (
	"errors"
	"fmt"

	"github.com/sunholo/ailang/internal/eval"
)

// ErrNoAIHandler is returned when AI.call is invoked without a configured handler
var ErrNoAIHandler = errors.New("AI handler not configured (use StubAIHandler for testing)")

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
}

// AIContext holds the handler for the current execution
//
// Thread-safety: AIContext is designed for single-threaded use
// within one evaluation. Create a new context for each step/tick.
type AIContext struct {
	handler AIHandler
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

// init registers AI effect operations
func init() {
	RegisterOp("AI", "call", aiCall)
	RegisterOp("AI", "callJson", aiCallJson)
	RegisterOp("AI", "callJsonSimple", aiCallJsonSimple)
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
		return nil, fmt.Errorf("E_AI_CALL_ERROR: %w", err)
	}

	return &eval.StringValue{Value: output}, nil
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
		return nil, fmt.Errorf("E_AI_CALL_ERROR: %w", err)
	}

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
		return nil, fmt.Errorf("E_AI_CALL_ERROR: %w", err)
	}

	return &eval.StringValue{Value: output}, nil
}
