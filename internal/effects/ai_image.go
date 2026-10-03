package effects

// AI image effect ops: callImage / callImageBase64 (v0.10.0) and their
// reference-conditioned variants callImageWithRefs / callImageBase64WithRefs
// (#1496). Every op honours a per-call "model" key in the options JSON,
// resolved through the same ModelResolver step() uses (#1496/#1500).

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sunholo-data/ailang/internal/ai"
	"github.com/sunholo-data/ailang/internal/eval"
)

// AIHandlerWithImageRefs is an optional capability: handlers that implement
// it can condition image generation on reference images. The unified
// ai.Handler and StubAIHandler implement it; a handler that does not makes
// callImage*WithRefs fail loudly instead of silently dropping the references.
type AIHandlerWithImageRefs interface {
	CallImageWithRefs(prompt string, refs []ai.ImagePart, outputPath string, options string) (string, error)
	CallImageBase64WithRefs(prompt string, refs []ai.ImagePart, options string) (string, error)
}

func init() {
	RegisterOp("AI", "callImage", aiCallImage)
	RegisterOp("AI", "callImageBase64", aiCallImageBase64)
	RegisterOp("AI", "callImageWithRefs", aiCallImageWithRefs)
	RegisterOp("AI", "callImageBase64WithRefs", aiCallImageBase64WithRefs)
}

// resolveImageOptions runs the options "model" key through the injected
// ModelResolver and substitutes the resolved api_name. No model key, no
// resolver, or an unchanged name returns options untouched; a cross-provider
// name returns the resolver's typed CodeModelNotAllowed error.
func (c *AIContext) resolveImageOptions(options string) (string, error) {
	if c.modelResolver == nil {
		return options, nil
	}
	opts := ai.ParseImageOptions(options)
	if opts == nil || opts.Model == "" {
		return options, nil
	}
	resolved, err := c.modelResolver(opts.Model)
	if err != nil {
		return "", err
	}
	if resolved == opts.Model {
		return options, nil
	}
	return ai.SetImageOptionsModel(options, resolved)
}

// CallImage generates an image and writes it to outputPath.
func (c *AIContext) CallImage(prompt, outputPath, options string) (string, error) {
	if c.handler == nil {
		return "", ErrNoAIHandler
	}
	options, err := c.resolveImageOptions(options)
	if err != nil {
		return "", err
	}
	return c.handler.CallImage(prompt, outputPath, options)
}

// CallImageBase64 generates an image and returns it as base64 JSON.
func (c *AIContext) CallImageBase64(prompt, options string) (string, error) {
	if c.handler == nil {
		return "", ErrNoAIHandler
	}
	options, err := c.resolveImageOptions(options)
	if err != nil {
		return "", err
	}
	return c.handler.CallImageBase64(prompt, options)
}

// imageRefsHandler returns the handler's reference-image capability or a
// loud not-supported error.
func (c *AIContext) imageRefsHandler() (AIHandlerWithImageRefs, error) {
	if c.handler == nil {
		return nil, ErrNoAIHandler
	}
	rh, ok := c.handler.(AIHandlerWithImageRefs)
	if !ok {
		return nil, fmt.Errorf("reference-conditioned image generation is not supported by AI handler %T", c.handler)
	}
	return rh, nil
}

// CallImageWithRefs generates an image conditioned on refs and writes it to outputPath.
func (c *AIContext) CallImageWithRefs(prompt string, refs []ai.ImagePart, outputPath, options string) (string, error) {
	rh, err := c.imageRefsHandler()
	if err != nil {
		return "", err
	}
	options, err = c.resolveImageOptions(options)
	if err != nil {
		return "", err
	}
	return rh.CallImageWithRefs(prompt, refs, outputPath, options)
}

// CallImageBase64WithRefs generates an image conditioned on refs and returns base64 JSON.
func (c *AIContext) CallImageBase64WithRefs(prompt string, refs []ai.ImagePart, options string) (string, error) {
	rh, err := c.imageRefsHandler()
	if err != nil {
		return "", err
	}
	options, err = c.resolveImageOptions(options)
	if err != nil {
		return "", err
	}
	return rh.CallImageBase64WithRefs(prompt, refs, options)
}

// CallImageWithRefs (stub) ignores the references and writes the same 1x1 PNG
// as CallImage — deterministic, no network.
func (h *StubAIHandler) CallImageWithRefs(prompt string, _ []ai.ImagePart, outputPath, options string) (string, error) {
	return h.CallImage(prompt, outputPath, options)
}

// CallImageBase64WithRefs (stub) returns the same payload as CallImageBase64.
func (h *StubAIHandler) CallImageBase64WithRefs(prompt string, _ []ai.ImagePart, options string) (string, error) {
	return h.CallImageBase64(prompt, options)
}

// CallImage returns a stub image path (writes a minimal 1x1 PNG to disk).
func (h *StubAIHandler) CallImage(prompt, outputPath, options string) (string, error) {
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return "", fmt.Errorf("stub: failed to create directory: %w", err)
	}
	if err := os.WriteFile(outputPath, stubPNG, 0o644); err != nil {
		return "", fmt.Errorf("stub: failed to write image: %w", err)
	}
	return outputPath, nil
}

// CallImageBase64 returns a stub base64 JSON response.
func (h *StubAIHandler) CallImageBase64(prompt, options string) (string, error) {
	b64 := base64.StdEncoding.EncodeToString(stubPNG)
	return fmt.Sprintf(`{"base64":"%s","mime_type":"image/png"}`, b64), nil
}

// stringArgs type-checks the leading string arguments of an image op.
func stringArgs(op string, args []eval.Value, names ...string) ([]string, error) {
	out := make([]string, len(names))
	for i, name := range names {
		s, ok := args[i].(*eval.StringValue)
		if !ok {
			return nil, fmt.Errorf("E_AI_TYPE_ERROR: %s: expected string %s, got %T", op, name, args[i])
		}
		out[i] = s.Value
	}
	return out, nil
}

// refsArg decodes the [ImagePart] argument of a WithRefs op.
func refsArg(op string, v eval.Value) ([]ai.ImagePart, error) {
	list, ok := v.(*eval.ListValue)
	if !ok {
		return nil, fmt.Errorf("E_AI_TYPE_ERROR: %s: expected [ImagePart] refs, got %T", op, v)
	}
	refs, err := decodeImageParts(list)
	if err != nil {
		return nil, fmt.Errorf("E_AI_TYPE_ERROR: %s: refs: %w", op, err)
	}
	return refs, nil
}

// aiCallImage implements AI.callImage(prompt, output_path, options) -> string
func aiCallImage(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	if len(args) < 3 {
		return nil, fmt.Errorf("E_AI_TYPE_ERROR: callImage: expected 3 arguments, got %d", len(args))
	}
	s, err := stringArgs("callImage", args, "prompt", "output_path", "options")
	if err != nil {
		return nil, err
	}
	if ctx.AI == nil {
		return nil, ErrNoAIHandler
	}
	result, err := ctx.AI.CallImage(s[0], s[1], s[2])
	if err != nil {
		return nil, fmt.Errorf(errAICallFmt, err)
	}
	return &eval.StringValue{Value: result}, nil
}

// aiCallImageBase64 implements AI.callImageBase64(prompt, options) -> string
func aiCallImageBase64(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("E_AI_TYPE_ERROR: callImageBase64: expected 2 arguments, got %d", len(args))
	}
	s, err := stringArgs("callImageBase64", args, "prompt", "options")
	if err != nil {
		return nil, err
	}
	if ctx.AI == nil {
		return nil, ErrNoAIHandler
	}
	result, err := ctx.AI.CallImageBase64(s[0], s[1])
	if err != nil {
		return nil, fmt.Errorf(errAICallFmt, err)
	}
	return &eval.StringValue{Value: result}, nil
}

// aiCallImageWithRefs implements
// AI.callImageWithRefs(prompt, refs: [ImagePart], output_path, options) -> string
func aiCallImageWithRefs(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	const op = "callImageWithRefs"
	if len(args) < 4 {
		return nil, fmt.Errorf("E_AI_TYPE_ERROR: %s: expected 4 arguments, got %d", op, len(args))
	}
	prompt, err := stringArgs(op, args[:1], "prompt")
	if err != nil {
		return nil, err
	}
	refs, err := refsArg(op, args[1])
	if err != nil {
		return nil, err
	}
	rest, err := stringArgs(op, args[2:], "output_path", "options")
	if err != nil {
		return nil, err
	}
	if ctx.AI == nil {
		return nil, ErrNoAIHandler
	}
	result, err := ctx.AI.CallImageWithRefs(prompt[0], refs, rest[0], rest[1])
	if err != nil {
		return nil, fmt.Errorf(errAICallFmt, err)
	}
	return &eval.StringValue{Value: result}, nil
}

// aiCallImageBase64WithRefs implements
// AI.callImageBase64WithRefs(prompt, refs: [ImagePart], options) -> string
func aiCallImageBase64WithRefs(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	const op = "callImageBase64WithRefs"
	if len(args) < 3 {
		return nil, fmt.Errorf("E_AI_TYPE_ERROR: %s: expected 3 arguments, got %d", op, len(args))
	}
	prompt, err := stringArgs(op, args[:1], "prompt")
	if err != nil {
		return nil, err
	}
	refs, err := refsArg(op, args[1])
	if err != nil {
		return nil, err
	}
	options, err := stringArgs(op, args[2:], "options")
	if err != nil {
		return nil, err
	}
	if ctx.AI == nil {
		return nil, ErrNoAIHandler
	}
	result, err := ctx.AI.CallImageBase64WithRefs(prompt[0], refs, options[0])
	if err != nil {
		return nil, fmt.Errorf(errAICallFmt, err)
	}
	return &eval.StringValue{Value: result}, nil
}
