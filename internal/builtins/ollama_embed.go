package builtins

import (
	"context"
	"fmt"
	"time"

	"github.com/ollama/ollama/api"
	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/types"
)

// Ollama embedding builtin for AILANG
// Part of DX-18 (Neural Embeddings via Ollama)

func init() {
	registerOllamaEmbed()
}

// registerOllamaEmbed registers the _ollama_embed builtin
func registerOllamaEmbed() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/ai",
		Name:    "_ollama_embed",
		NumArgs: 2,
		IsPure:  false, // Has IO effect (network call)
		Effect:  "IO",
		Type:    makeOllamaEmbedType,
		Impl:    ollamaEmbedImpl,

		Metadata: &BuiltinMetadata{
			Description: "Generate embeddings using Ollama",
			LongDesc:    "Calls Ollama's embedding API to generate vector embeddings for text. Requires Ollama to be running locally with an embedding model (e.g., embeddinggemma).",
			Params: []ParamDoc{
				{Name: "model", Description: "The Ollama model name (e.g., 'embeddinggemma')"},
				{Name: "text", Description: "The text to embed"},
			},
			Returns: "list[float]: The embedding vector",
			Examples: []Example{
				{Code: `_ollama_embed("embeddinggemma", "Hello world")`, Description: "Generate embedding for text"},
			},
			SeeAlso:   []string{"_simhash", "_hamming_distance"},
			Since:     "v0.5.12",
			Stability: StabilityExperimental,
			Tags:      []string{"ai", "embedding", "ollama", "neural"},
			Category:  "ai",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _ollama_embed: %v", err))
	}
}

// makeOllamaEmbedType builds the type signature for _ollama_embed
// Type: (string, string) -> list[float] ! {IO}
func makeOllamaEmbedType() types.Type {
	T := types.NewBuilder()
	return T.Func(
		T.String(), // model
		T.String(), // text
	).Returns(T.List(T.Float())).Effects("IO")
}

// ollamaEmbedImpl is the implementation for _ollama_embed
func ollamaEmbedImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	modelVal, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("_ollama_embed: expected String model, got %T", args[0])
	}

	textVal, ok := args[1].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("_ollama_embed: expected String text, got %T", args[1])
	}

	// Create Ollama client
	client, err := api.ClientFromEnvironment()
	if err != nil {
		return nil, fmt.Errorf("_ollama_embed: failed to create Ollama client: %w", err)
	}

	// Call embed API
	goCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := client.Embed(goCtx, &api.EmbedRequest{
		Model: modelVal.Value,
		Input: textVal.Value,
	})
	if err != nil {
		return nil, fmt.Errorf("_ollama_embed: Ollama embed failed: %w", err)
	}

	// Check we got embeddings
	if len(resp.Embeddings) == 0 {
		return nil, fmt.Errorf("_ollama_embed: no embeddings returned")
	}

	// Convert float32 to float64 and wrap in ListValue
	embedding := resp.Embeddings[0]
	elements := make([]eval.Value, len(embedding))
	for i, v := range embedding {
		elements[i] = &eval.FloatValue{Value: float64(v)}
	}

	return &eval.ListValue{Elements: elements}, nil
}
