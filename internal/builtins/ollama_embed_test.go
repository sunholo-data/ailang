package builtins

import (
	"testing"

	"github.com/sunholo/ailang/internal/effects"
	"github.com/sunholo/ailang/internal/eval"
)

func TestOllamaEmbed(t *testing.T) {
	// Skip if Ollama not available
	ctx := &effects.EffContext{}

	args := []eval.Value{
		&eval.StringValue{Value: "embeddinggemma"},
		&eval.StringValue{Value: "The sky is blue"},
	}

	result, err := ollamaEmbedImpl(ctx, args)
	if err != nil {
		t.Fatalf("ollamaEmbedImpl failed: %v", err)
	}

	listVal, ok := result.(*eval.ListValue)
	if !ok {
		t.Fatalf("expected ListValue, got %T", result)
	}

	// EmbeddingGemma returns 768 dimensions
	if len(listVal.Elements) != 768 {
		t.Errorf("expected 768 dimensions, got %d", len(listVal.Elements))
	}

	// Check first element is a float
	if _, ok := listVal.Elements[0].(*eval.FloatValue); !ok {
		t.Errorf("expected FloatValue elements, got %T", listVal.Elements[0])
	}

	t.Logf("Got %d-dimensional embedding", len(listVal.Elements))
}
