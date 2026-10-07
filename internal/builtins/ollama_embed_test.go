package builtins

import (
	"encoding/base64"
	"net/http"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
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

func TestOllamaMediaInput(t *testing.T) {
	in := ollamaMediaInput("", []byte{0x89, 'P', 'N', 'G'}, nil)
	if len(in) != 1 || in["image"] != "iVBORw==" {
		t.Errorf("image-only input = %v, want only image base64", in)
	}

	in = ollamaMediaInput("caption", []byte{1}, []byte{2})
	if in["text"] != "caption" || in["image"] != "AQ==" || in["audio"] != "Ag==" {
		t.Errorf("all-parts input = %v", in)
	}

	if in := ollamaMediaInput("", nil, []byte{}); len(in) != 0 {
		t.Errorf("empty parts should be omitted, got %v", in)
	}
}

func TestOllamaEmbedMediaRejectsEmpty(t *testing.T) {
	_, err := ollamaEmbedMediaImpl(&effects.EffContext{}, []eval.Value{
		&eval.StringValue{Value: "embeddinggemma-2"},
		&eval.StringValue{Value: ""},
		&eval.BytesValue{},
		&eval.BytesValue{},
	})
	if err == nil {
		t.Fatal("expected an error when text, image and audio are all empty")
	}
}

func TestOllamaEmbedMedia(t *testing.T) {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get("http://localhost:11434/api/tags")
	if err != nil {
		t.Skip("Ollama not available: ", err)
	}
	resp.Body.Close()

	// A 1x1 PNG
	png, _ := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8DwHwAFBQIAX8jx0gAAAABJRU5ErkJggg==")

	result, err := ollamaEmbedMediaImpl(&effects.EffContext{}, []eval.Value{
		&eval.StringValue{Value: "embeddinggemma-2"},
		&eval.StringValue{Value: ""},
		&eval.BytesValue{Value: png},
		&eval.BytesValue{},
	})
	if err != nil {
		t.Skipf("Ollama media embed failed (model or Ollama >= 0.36 may not be available): %v", err)
	}

	listVal, ok := result.(*eval.ListValue)
	if !ok {
		t.Fatalf("expected ListValue, got %T", result)
	}
	if len(listVal.Elements) == 0 {
		t.Fatal("expected a non-empty embedding")
	}
	t.Logf("Got %d-dimensional image embedding", len(listVal.Elements))
}
