package builtins

import (
	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/types"
)

// AI image-generation builtins: _ai_call_image / _ai_call_image_base64
// (M-AI-IMAGE, v0.10.0) and the reference-conditioned variants
// _ai_call_image_with_refs / _ai_call_image_base64_with_refs (#1496).

func init() {
	registerAICallImage()
	registerAICallImageBase64()
	registerAICallImageWithRefs()
	registerAICallImageBase64WithRefs()
}

// imageOptionsDoc is the options-JSON contract shared by every image builtin.
const imageOptionsDoc = `Options is a JSON string with optional fields:
  - aspect_ratio: "1:1", "16:9", "9:16", etc.
  - mime_type: "image/png" (default), "image/jpeg" (openrouter also: "image/webp")
  - model: per-call model override (api_name or models.yml friendly name);
    empty/absent = the --ai bound model. A name that resolves to a different
    provider than the bound handler fails with ModelNotAllowed.

Providers: Gemini (e.g. gemini-2.5-flash-image) and OpenRouter image-output
models (e.g. google/gemini-3-pro-image via --ai openrouter/...). Other
providers reject image generation with a clear error.`

// imageRefsDoc documents the refs parameter of the WithRefs variants.
const imageRefsDoc = `refs is a list of ImagePart records {source, mime}: source is a base64
payload or a data-URI; mime is required for raw base64 (e.g. "image/png").
std/fs.readFileBytes returns base64 suitable for source. Gemini sends refs as
inlineData parts, OpenRouter as image_url content parts; other providers fail
loudly rather than ignoring them. An empty list behaves like the non-refs call.`

// _ai_call_image: Generate an image and save to file
func registerAICallImage() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/ai",
		Name:    "_ai_call_image",
		NumArgs: 3, // prompt, output_path, options
		Effect:  "AI",
		Type:    makeAICallImageType,
		Impl:    aiCallImageImpl,
		Metadata: &BuiltinMetadata{
			Description: "Generate an image via AI and save to file",
			LongDesc: `Calls an AI image generation model and writes the resulting image to the
specified output path.

` + imageOptionsDoc + `

Returns the output path on success. Requires both AI and FS capabilities.`,
			Params: []ParamDoc{
				{Name: "prompt", Description: "Image generation prompt"},
				{Name: "output_path", Description: "File path to write the image"},
				{Name: "options", Description: "JSON options string (aspect_ratio, mime_type, model)"},
			},
			Returns: "The output file path",
			Examples: []Example{
				{Code: `AI.callImage("A sunset over mountains", "output.png", "{}")`, Description: "Generate image with defaults"},
				{Code: `AI.callImage("Banner", "banner.png", "{\"aspect_ratio\": \"16:9\"}")`, Description: "Generate with aspect ratio"},
				{Code: `AI.callImage("Portrait", "p.png", "{\"model\": \"google/gemini-3-pro-image\"}")`, Description: "Per-call image model (e.g. under --ai openrouter/...)"},
			},
			SeeAlso:   []string{"std/ai.callImageBase64", "std/ai.callImageWithRefs", "std/ai.call"},
			Since:     "v0.10.0",
			Stability: StabilityStable,
			Tags:      []string{"ai", "image", "generation", "gemini", "openrouter"},
			Category:  "ai",
		},
	})
	if err != nil {
		panic("failed to register _ai_call_image builtin: " + err.Error())
	}
}

func makeAICallImageType() types.Type {
	T := types.NewBuilder()
	// (prompt: string, output_path: string, options: string) -> string ! {AI}
	return T.Func(T.String(), T.String(), T.String()).Returns(T.String()).Effects("AI")
}

func aiCallImageImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	if err := ctx.RequireCapWithBudget("AI", ""); err != nil {
		return nil, err
	}
	return effects.Call(ctx, "AI", "callImage", args)
}

// _ai_call_image_base64: Generate an image and return as base64 JSON
func registerAICallImageBase64() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/ai",
		Name:    "_ai_call_image_base64",
		NumArgs: 2, // prompt, options
		Effect:  "AI",
		Type:    makeAICallImageBase64Type,
		Impl:    aiCallImageBase64Impl,
		Metadata: &BuiltinMetadata{
			Description: "Generate an image via AI and return as base64 JSON",
			LongDesc: `Calls an AI image generation model and returns the image as a JSON string:
{"base64": "<base64-encoded-data>", "mime_type": "image/png"}

` + imageOptionsDoc + `

Use std/bytes.fromBase64 to decode the base64 data to bytes if needed.
Only requires AI capability (no file system access).`,
			Params: []ParamDoc{
				{Name: "prompt", Description: "Image generation prompt"},
				{Name: "options", Description: "JSON options string (aspect_ratio, mime_type, model)"},
			},
			Returns: "JSON string with base64 and mime_type fields",
			Examples: []Example{
				{Code: `AI.callImageBase64("A logo", "{}")`, Description: "Generate image as base64"},
			},
			SeeAlso:   []string{"std/ai.callImage", "std/ai.callImageBase64WithRefs", "std/bytes.fromBase64"},
			Since:     "v0.10.0",
			Stability: StabilityStable,
			Tags:      []string{"ai", "image", "generation", "base64"},
			Category:  "ai",
		},
	})
	if err != nil {
		panic("failed to register _ai_call_image_base64 builtin: " + err.Error())
	}
}

func makeAICallImageBase64Type() types.Type {
	T := types.NewBuilder()
	// (prompt: string, options: string) -> string ! {AI}
	return T.Func(T.String(), T.String()).Returns(T.String()).Effects("AI")
}

func aiCallImageBase64Impl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	if err := ctx.RequireCapWithBudget("AI", ""); err != nil {
		return nil, err
	}
	return effects.Call(ctx, "AI", "callImageBase64", args)
}

// _ai_call_image_with_refs: reference-conditioned image generation to a file
func registerAICallImageWithRefs() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/ai",
		Name:    "_ai_call_image_with_refs",
		NumArgs: 4, // prompt, refs, output_path, options
		Effect:  "AI",
		Type:    makeAICallImageWithRefsType,
		Impl:    aiCallImageWithRefsImpl,
		Metadata: &BuiltinMetadata{
			Description: "Generate an image conditioned on reference images and save to file",
			LongDesc: `Like callImage, but the model also sees reference images (image+text input):
editing, or keeping the same character across variations.

` + imageRefsDoc + `

` + imageOptionsDoc + `

Returns the output path on success. Requires both AI and FS capabilities.`,
			Params: []ParamDoc{
				{Name: "prompt", Description: "Image generation prompt"},
				{Name: "refs", Description: "Reference images ([ImagePart])"},
				{Name: "output_path", Description: "File path to write the image"},
				{Name: "options", Description: "JSON options string (aspect_ratio, mime_type, model)"},
			},
			Returns: "The output file path",
			Examples: []Example{
				{Code: `AI.callImageWithRefs("Same person, smiling", [{source: b64, mime: "image/png"}], "smile.png", "{}")`, Description: "Condition on one identity reference"},
			},
			SeeAlso:   []string{"std/ai.callImage", "std/ai.callImageBase64WithRefs", "std/fs.readFileBytes"},
			Since:     "v0.51.1",
			Stability: StabilityExperimental,
			Tags:      []string{"ai", "image", "generation", "reference", "edit"},
			Category:  "ai",
		},
	})
	if err != nil {
		panic("failed to register _ai_call_image_with_refs builtin: " + err.Error())
	}
}

func makeAICallImageWithRefsType() types.Type {
	T := types.NewBuilder()
	// (prompt: string, refs: [ImagePart], output_path: string, options: string) -> string ! {AI}
	return T.Func(T.String(), T.List(imagePartRecordType(T)), T.String(), T.String()).Returns(T.String()).Effects("AI")
}

func aiCallImageWithRefsImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	if err := ctx.RequireCapWithBudget("AI", ""); err != nil {
		return nil, err
	}
	return effects.Call(ctx, "AI", "callImageWithRefs", args)
}

// _ai_call_image_base64_with_refs: reference-conditioned image generation as base64 JSON
func registerAICallImageBase64WithRefs() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/ai",
		Name:    "_ai_call_image_base64_with_refs",
		NumArgs: 3, // prompt, refs, options
		Effect:  "AI",
		Type:    makeAICallImageBase64WithRefsType,
		Impl:    aiCallImageBase64WithRefsImpl,
		Metadata: &BuiltinMetadata{
			Description: "Generate an image conditioned on reference images and return base64 JSON",
			LongDesc: `Like callImageBase64, but the model also sees reference images (image+text
input). Returns {"base64": "...", "mime_type": "..."}.

` + imageRefsDoc + `

` + imageOptionsDoc + `

Only requires AI capability (no file system access).`,
			Params: []ParamDoc{
				{Name: "prompt", Description: "Image generation prompt"},
				{Name: "refs", Description: "Reference images ([ImagePart])"},
				{Name: "options", Description: "JSON options string (aspect_ratio, mime_type, model)"},
			},
			Returns: "JSON string with base64 and mime_type fields",
			Examples: []Example{
				{Code: `AI.callImageBase64WithRefs("Same person, aged 60", [{source: b64, mime: "image/png"}], "{\"model\": \"google/gemini-3-pro-image\"}")`, Description: "Per-call model plus identity reference"},
			},
			SeeAlso:   []string{"std/ai.callImageBase64", "std/ai.callImageWithRefs", "std/bytes.fromBase64"},
			Since:     "v0.51.1",
			Stability: StabilityExperimental,
			Tags:      []string{"ai", "image", "generation", "reference", "base64"},
			Category:  "ai",
		},
	})
	if err != nil {
		panic("failed to register _ai_call_image_base64_with_refs builtin: " + err.Error())
	}
}

func makeAICallImageBase64WithRefsType() types.Type {
	T := types.NewBuilder()
	// (prompt: string, refs: [ImagePart], options: string) -> string ! {AI}
	return T.Func(T.String(), T.List(imagePartRecordType(T)), T.String()).Returns(T.String()).Effects("AI")
}

func aiCallImageBase64WithRefsImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	if err := ctx.RequireCapWithBudget("AI", ""); err != nil {
		return nil, err
	}
	return effects.Call(ctx, "AI", "callImageBase64WithRefs", args)
}
