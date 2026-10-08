package builtins

import (
	"context"
	"encoding/base64"
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
	registerOllamaEmbedMedia()
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
			SeeAlso:   []string{"_ollama_embed_media", "_simhash", "_hamming_distance"},
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

	return ollamaEmbedRequest("_ollama_embed", modelVal.Value, textVal.Value, 30*time.Second)
}

// registerOllamaEmbedMedia registers the _ollama_embed_media builtin
func registerOllamaEmbedMedia() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/ai",
		Name:    "_ollama_embed_media",
		NumArgs: 4,
		IsPure:  false, // Has IO effect (network call)
		Effect:  "IO",
		Type:    makeOllamaEmbedMediaType,
		Impl:    ollamaEmbedMediaImpl,

		Metadata: &BuiltinMetadata{
			Description: "Generate a multimodal embedding (text, image, audio) using Ollama",
			LongDesc: "Calls Ollama's embedding API with a single input that may carry text, an image and audio together, " +
				"for models with the vision or audio capability (e.g., embeddinggemma-2, Ollama >= 0.36). " +
				"An empty string or empty bytes means that part is absent; at least one part must be present. " +
				"Image and audio are raw bytes (PNG/JPEG, WAV); the builtin base64-encodes them. " +
				"The vector shares the model's text space, so it compares directly with _ollama_embed output from the same model.",
			Params: []ParamDoc{
				{Name: "model", Description: "The Ollama model name (e.g., 'embeddinggemma-2')"},
				{Name: "text", Description: "Text to embed alongside the media, or \"\" for none"},
				{Name: "image", Description: "Image bytes, or empty bytes for none"},
				{Name: "audio", Description: "Audio bytes, or empty bytes for none"},
			},
			Returns: "list[float]: The embedding vector",
			Examples: []Example{
				{Code: `_ollama_embed_media("embeddinggemma-2", "", imgBytes, _bytes_from_string(""))`, Description: "Embed an image on its own"},
				{Code: `_ollama_embed_media("embeddinggemma-2", "a cat on a windowsill", imgBytes, _bytes_from_string(""))`, Description: "Embed an image together with a caption"},
			},
			SeeAlso:   []string{"_ollama_embed"},
			Since:     "v0.53.0",
			Stability: StabilityExperimental,
			Tags:      []string{"ai", "embedding", "ollama", "neural", "multimodal", "image", "audio"},
			Category:  "ai",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _ollama_embed_media: %v", err))
	}
}

// makeOllamaEmbedMediaType builds the type signature for _ollama_embed_media
// Type: (string, string, bytes, bytes) -> list[float] ! {IO}
func makeOllamaEmbedMediaType() types.Type {
	T := types.NewBuilder()
	return T.Func(
		T.String(), // model
		T.String(), // text
		T.Bytes(),  // image
		T.Bytes(),  // audio
	).Returns(T.List(T.Float())).Effects("IO")
}

// ollamaEmbedMediaImpl is the implementation for _ollama_embed_media
func ollamaEmbedMediaImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	modelVal, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("_ollama_embed_media: expected String model, got %T", args[0])
	}
	textVal, ok := args[1].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("_ollama_embed_media: expected String text, got %T", args[1])
	}
	imageVal, ok := args[2].(*eval.BytesValue)
	if !ok {
		return nil, fmt.Errorf("_ollama_embed_media: expected Bytes image, got %T", args[2])
	}
	audioVal, ok := args[3].(*eval.BytesValue)
	if !ok {
		return nil, fmt.Errorf("_ollama_embed_media: expected Bytes audio, got %T", args[3])
	}

	input := ollamaMediaInput(textVal.Value, imageVal.Value, audioVal.Value)
	if len(input) == 0 {
		return nil, fmt.Errorf("_ollama_embed_media: text, image and audio are all empty")
	}

	// Media inputs are larger and vision/audio models load slower than text-only ones
	return ollamaEmbedRequest("_ollama_embed_media", modelVal.Value, input, 120*time.Second)
}

// ollamaMediaInput builds Ollama's multimodal embed input object, omitting empty parts
func ollamaMediaInput(text string, image, audio []byte) map[string]string {
	input := map[string]string{}
	if text != "" {
		input["text"] = text
	}
	if len(image) > 0 {
		input["image"] = base64.StdEncoding.EncodeToString(image)
	}
	if len(audio) > 0 {
		input["audio"] = base64.StdEncoding.EncodeToString(audio)
	}
	return input
}

// ollamaEmbedRequest calls Ollama's embed API with one input and returns its vector
func ollamaEmbedRequest(name, model string, input any, timeout time.Duration) (eval.Value, error) {
	client, err := api.ClientFromEnvironment()
	if err != nil {
		return nil, fmt.Errorf("%s: failed to create Ollama client: %w", name, err)
	}

	goCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	resp, err := client.Embed(goCtx, &api.EmbedRequest{
		Model: model,
		Input: input,
	})
	if err != nil {
		return nil, fmt.Errorf("%s: Ollama embed failed: %w", name, err)
	}

	if len(resp.Embeddings) == 0 {
		return nil, fmt.Errorf("%s: no embeddings returned", name)
	}

	// Convert float32 to float64 and wrap in ListValue
	embedding := resp.Embeddings[0]
	elements := make([]eval.Value, len(embedding))
	for i, v := range embedding {
		elements[i] = &eval.FloatValue{Value: float64(v)}
	}

	return &eval.ListValue{Elements: elements}, nil
}
