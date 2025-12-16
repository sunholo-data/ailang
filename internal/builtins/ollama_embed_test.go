package builtins

import (
	"net/http"
	"testing"
	"time"

	"github.com/sunholo/ailang/internal/effects"
	"github.com/sunholo/ailang/internal/eval"
)

func TestOllamaEmbed(t *testing.T) {
	// Skip if Ollama not available
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get("http://localhost:11434/api/tags")
	if err != nil {
		t.Skip("Ollama not available: ", err)
	}
	resp.Body.Close()

	ctx := &effects.EffContext{}

	args := []eval.Value{
		&eval.StringValue{Value: "embeddinggemma"},
		&eval.StringValue{Value: "The sky is blue"},
	}

	result, err := ollamaEmbedImpl(ctx, args)
	if err != nil {
		t.Skipf("Ollama embed failed (model may not be available): %v", err)
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
