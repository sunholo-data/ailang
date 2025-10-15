package effects

import (
	"testing"

	"github.com/sunholo/ailang/internal/eval"
)

// TestIntegration_EffContextFlow tests the full flow from CLI to effects
func TestIntegration_EffContextFlow(t *testing.T) {
	// Simulate CLI setup
	evaluator := eval.NewCoreEvaluator()

	// Case 1: No caps granted
	effCtx := NewEffContext()
	evaluator.SetEffContext(effCtx)

	// Get context back
	ctx := evaluator.GetEffContext()
	if ctx == nil {
		t.Fatal("expected EffContext to be set")
	}

	effCtxBack, ok := ctx.(*EffContext)
	if !ok {
		t.Fatalf("expected *EffContext, got %T", ctx)
	}

	// Should not have IO cap
	if effCtxBack.HasCap("IO") {
		t.Error("should not have IO capability")
	}

	// Should fail to call IO operation
	_, err := Call(effCtxBack, "IO", "println", []eval.Value{&eval.StringValue{Value: "test"}})
	if err == nil {
		t.Error("expected capability error")
	}

	// Case 2: With IO cap
	effCtx2 := NewEffContext()
	effCtx2.Grant(NewCapability("IO"))
	evaluator.SetEffContext(effCtx2)

	ctx2 := evaluator.GetEffContext().(*EffContext)
	if !ctx2.HasCap("IO") {
		t.Error("should have IO capability")
	}
}
