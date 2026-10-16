package builtins

import (
	"fmt"

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
		IsPure:  false, // Has AI effect (network call)
		Effect:  "AI",
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
// Type: (string, string) -> list[float] ! {AI}
func makeOllamaEmbedType() types.Type {
	T := types.NewBuilder()
	return T.Func(
		T.String(), // model
		T.String(), // text
	).Returns(T.List(T.Float())).Effects("AI")
}

// ollamaEmbedImpl is the implementation for _ollama_embed
func ollamaEmbedImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	return effects.Call(ctx, "AI", "embedOllama", args)
}

// registerOllamaEmbedMedia registers the _ollama_embed_media builtin
func registerOllamaEmbedMedia() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/ai",
		Name:    "_ollama_embed_media",
		NumArgs: 4,
		IsPure:  false, // Has AI effect (network call)
		Effect:  "AI",
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
// Type: (string, string, bytes, bytes) -> list[float] ! {AI}
func makeOllamaEmbedMediaType() types.Type {
	T := types.NewBuilder()
	return T.Func(
		T.String(), // model
		T.String(), // text
		T.Bytes(),  // image
		T.Bytes(),  // audio
	).Returns(T.List(T.Float())).Effects("AI")
}

// ollamaEmbedMediaImpl is the implementation for _ollama_embed_media
func ollamaEmbedMediaImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	return effects.Call(ctx, "AI", "embedOllamaMedia", args)
}
