package effects

import (
	"errors"
	"testing"

	"github.com/sunholo-data/ailang/internal/ai"
	"github.com/sunholo-data/ailang/internal/eval"
)

func TestAIContext_NilHandler(t *testing.T) {
	ctx := NewAIContext(nil)

	_, err := ctx.Call("test input")
	if err == nil {
		t.Fatal("expected error for nil handler, got nil")
	}

	if !errors.Is(err, ErrNoAIHandler) {
		t.Errorf("expected ErrNoAIHandler, got %v", err)
	}
}

func TestStubAIHandler_DefaultResponse(t *testing.T) {
	handler := NewStubAIHandler()
	ctx := NewAIContext(handler)

	output, err := ctx.Call("any input")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output != `{"kind":"Wait"}` {
		t.Errorf("expected default response, got %q", output)
	}
}

func TestStubAIHandler_CustomResponse(t *testing.T) {
	handler := NewStubAIHandler()
	handler.SetResponse(`{"health":50}`, `{"kind":"Pickup","0":5}`)

	ctx := NewAIContext(handler)

	output, err := ctx.Call(`{"health":50}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output != `{"kind":"Pickup","0":5}` {
		t.Errorf("expected custom response, got %q", output)
	}
}

func TestStubAIHandler_SetDefaultResponse(t *testing.T) {
	handler := NewStubAIHandler()
	handler.SetDefaultResponse(`{"kind":"Attack"}`)

	ctx := NewAIContext(handler)

	output, err := ctx.Call("unknown input")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output != `{"kind":"Attack"}` {
		t.Errorf("expected custom default response, got %q", output)
	}
}

func TestAICall_EffectOperation(t *testing.T) {
	// Create effect context with AI capability
	ctx := NewEffContext(nil)
	ctx.Grant(NewCapability("AI"))
	ctx.AI = NewAIContext(NewStubAIHandler())

	// Call AI operation
	args := []eval.Value{
		&eval.StringValue{Value: `{"action":"decide"}`},
	}

	result, err := Call(ctx, "AI", "call", args)
	if err != nil {
		t.Fatalf("AI call failed: %v", err)
	}

	strVal, ok := result.(*eval.StringValue)
	if !ok {
		t.Errorf("expected StringValue, got %T", result)
	}

	if strVal.Value != `{"kind":"Wait"}` {
		t.Errorf("expected default response, got %q", strVal.Value)
	}
}

func TestAI_CapabilityRequired(t *testing.T) {
	// Create effect context WITHOUT AI capability
	ctx := NewEffContext(nil)
	ctx.AI = NewAIContext(NewStubAIHandler())

	// Try to call AI - should fail with capability error
	args := []eval.Value{
		&eval.StringValue{Value: "test"},
	}

	_, err := Call(ctx, "AI", "call", args)
	if err == nil {
		t.Fatal("expected capability error, got nil")
	}

	capErr, ok := err.(*CapabilityError)
	if !ok {
		t.Fatalf("expected CapabilityError, got %T: %v", err, err)
	}

	if capErr.Effect != "AI" {
		t.Errorf("expected AI capability error, got %s", capErr.Effect)
	}
}

func TestAI_NoContext(t *testing.T) {
	// Create effect context with AI capability but no AIContext
	ctx := NewEffContext(nil)
	ctx.Grant(NewCapability("AI"))
	// ctx.AI is nil

	args := []eval.Value{
		&eval.StringValue{Value: "test"},
	}

	_, err := Call(ctx, "AI", "call", args)
	if err == nil {
		t.Fatal("expected error for nil AI context, got nil")
	}

	if !errors.Is(err, ErrNoAIHandler) {
		t.Errorf("expected ErrNoAIHandler, got %v", err)
	}
}

// --- ModelResolver (per-call friendly-name resolution) ---------------------

func TestAIContext_ModelResolver_ResolvesFriendlyName(t *testing.T) {
	ctx := NewAIContext(NewStubAIHandler()).WithModelResolver(
		func(model string) (string, error) {
			if model == "gemini-2-5-flash" {
				return "gemini-2.5-flash", nil // friendly -> api_name
			}
			return model, nil
		})
	resp, err := ctx.Step("gemini-2-5-flash", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Model != "gemini-2.5-flash" {
		t.Errorf("expected friendly name resolved to api_name, got %q", resp.Model)
	}
}

func TestAIContext_ModelResolver_UnknownPassesThrough(t *testing.T) {
	// An unknown string (already an api_name) must reach the handler unchanged,
	// preserving the pre-resolver path where step("gemini-2.5-flash") worked.
	ctx := NewAIContext(NewStubAIHandler()).WithModelResolver(
		func(model string) (string, error) { return model, nil })
	resp, err := ctx.Step("gemini-2.5-flash", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Model != "gemini-2.5-flash" {
		t.Errorf("expected api_name unchanged, got %q", resp.Model)
	}
}

func TestAIContext_ModelResolver_EmptyModelSkipsResolver(t *testing.T) {
	called := false
	ctx := NewAIContext(NewStubAIHandler()).WithModelResolver(
		func(model string) (string, error) { called = true; return model, nil })
	if _, err := ctx.Step("", nil, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if called {
		t.Error("resolver should not be called for empty model (handler default)")
	}
}

func TestAIContext_ModelResolver_CrossVendorErrors(t *testing.T) {
	ctx := NewAIContext(NewStubAIHandler()).WithModelResolver(
		func(model string) (string, error) {
			return "", ai.NewAIError(ai.CodeModelNotAllowed, "cross-vendor", false)
		})
	_, err := ctx.Step("claude-haiku-4-5", nil, nil)
	if err == nil {
		t.Fatal("expected cross-vendor error, got nil")
	}
	var aiErr *ai.AIError
	if !errors.As(err, &aiErr) || aiErr.Code != ai.CodeModelNotAllowed {
		t.Errorf("expected ModelNotAllowed AIError, got %v", err)
	}
}

func TestAIContext_ModelResolver_NilResolverPassthrough(t *testing.T) {
	// Default (no resolver) must pass the model straight through.
	ctx := NewAIContext(NewStubAIHandler())
	resp, err := ctx.Step("anything", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Model != "anything" {
		t.Errorf("expected passthrough, got %q", resp.Model)
	}
}
