package effects

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/ai"
	"github.com/sunholo-data/ailang/internal/eval"
)

// WithEmbedder attaches the host-selected embedding provider.
func (c *AIContext) WithEmbedder(e Embedder) *AIContext {
	c.embedder = e
	c.embedderConfigured = true
	return c
}

// Embed supports embedding-only contexts and lazy completion contexts alike.
func (c *AIContext) Embed(text string) ([]float32, error) {
	if c == nil {
		return nil, ai.NewAIError(ai.CodeProviderNotFound, "no embedding provider configured", false)
	}
	if c.embedder != nil {
		return c.embedder.Embed(text)
	}
	if c.embedderConfigured {
		return nil, ai.NewAIError(ai.CodeProviderNotFound, "no embedding provider configured; configure embeddings.provider", false)
	}
	if c.lazy != nil {
		resolved, err := c.ready()
		if err != nil {
			return nil, err
		}
		return resolved.Embed(text)
	}
	return nil, ai.NewAIError(ai.CodeProviderNotFound, "no embedding provider configured; configure embeddings.provider", false)
}
func init() { RegisterOp("AI", "embed", aiEmbed) }
func aiEmbed(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("E_AI_TYPE_ERROR: embed: expected 1 argument, got %d", len(args))
	}
	text, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("E_AI_TYPE_ERROR: embed: expected string, got %T", args[0])
	}
	vector, err := ctx.AI.Embed(text.Value)
	if err != nil {
		typed := classifyOpError(err)
		ctx.RecordAIEffect("embed", []string{truncateForTrace(text.Value)}, fmt.Sprintf(errResultPrefix, typed.Code), nil)
		return makeAIErrorResultRecord(typed), nil
	}
	values := make([]eval.Value, len(vector))
	for i, v := range vector {
		values[i] = &eval.FloatValue{Value: float64(v)}
	}
	ctx.RecordAIEffect("embed", []string{truncateForTrace(text.Value)}, fmt.Sprintf("%d dimensions", len(vector)), nil)
	return &eval.TaggedValue{CtorName: "Ok", Fields: []eval.Value{&eval.ListValue{Elements: values}}}, nil
}

// WithEmbeddingCacheResource transfers cache ownership to this context.
func (c *AIContext) WithEmbeddingCacheResource(store EmbeddingCache) *AIContext {
	c.embedCacheResource = store
	return c
}

// Close releases only host-owned embedding resources; it never initializes a lazy provider.
func (c *AIContext) Close() error {
	if c == nil || c.embedCacheResource == nil {
		return nil
	}
	c.embedCacheClose.Do(func() { c.embedCacheCloseErr = c.embedCacheResource.Close() })
	return c.embedCacheCloseErr
}
