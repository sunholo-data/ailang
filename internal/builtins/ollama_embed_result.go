package builtins

import (
	"errors"
	"fmt"
	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/types"
)

func init() {
	for _, media := range []bool{false, true} {
		registerOllamaEmbedResult(media)
	}
}

func registerOllamaEmbedResult(media bool) {
	name, op, count := "_ollama_embed_result", "embedOllama", 2
	params := []ParamDoc{{Name: "model", Description: "Ollama embedding model"}, {Name: "text", Description: "Text to embed"}}
	if media {
		name, op, count = "_ollama_embed_media_result", "embedOllamaMedia", 4
		params = append(params, ParamDoc{Name: "image", Description: "Raw image bytes, or empty"}, ParamDoc{Name: "audio", Description: "Raw audio bytes, or empty"})
	}
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module: "std/ai", Name: name, NumArgs: count, IsPure: false, Effect: "AI",
		Type: func() types.Type {
			T := types.NewBuilder()
			args := []types.Type{T.String(), T.String()}
			if media {
				args = append(args, T.Bytes(), T.Bytes())
			}
			return T.Func(args...).Returns(T.App("Result", T.List(T.Float()), T.String())).Effects("AI")
		},
		Impl: func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
			v, err := effects.Call(ctx, "AI", op, args)
			if err != nil {
				var providerErr *effects.EmbeddingError
				if !errors.As(err, &providerErr) {
					return nil, err
				}
				return &eval.TaggedValue{CtorName: "Err", Fields: []eval.Value{&eval.StringValue{Value: err.Error()}}}, nil
			}
			return &eval.TaggedValue{CtorName: "Ok", Fields: []eval.Value{v}}, nil
		},
		Metadata: &BuiltinMetadata{Description: "Generate an Ollama embedding with recoverable string errors", LongDesc: "Returns Ok(vector) or Err(message). Requires AI capability and consumes one AI budget unit. Supports --ai-stub and embedding fixtures; capability and budget failures remain runtime errors.", Params: params, Returns: "Result[[float], string]", Since: "unreleased", Stability: StabilityExperimental, Tags: []string{"ai", "embedding", "ollama", "result"}, Category: "ai", SeeAlso: []string{"_ollama_embed", "_ollama_embed_media"}},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register %s: %v", name, err))
	}
}
