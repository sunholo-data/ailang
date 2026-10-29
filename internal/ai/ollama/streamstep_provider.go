package ollama

import (
	"context"
	"sync/atomic"

	"github.com/sunholo-data/ailang/internal/ai"
)

// StreamStep makes the ollama client an ai.StreamingProvider, so a caller of
// std/ai.stepWithStream (motoko) receives the streamed /v1 deltas as they
// arrive (2026-10-16).
//
// Before it, the handler's fallback ran Step and fired one synthetic chunk at
// the end, so with AILANG_OLLAMA_V1_STREAM=1 the HTTP call streamed but the
// caller still saw nothing until the turn finished. motoko journals each delta
// to its session log; with no deltas a long local generation looked exactly
// like a hang to the executor's idle watchdog, which killed 9 of 39 v0.54.9
// qwen3.8 runs mid-generation.
//
// Same request, same response: the callback reaches stepV1Stream through the
// context, so Step's code is untouched. When the call does not take the
// streaming path (flag off, no tools, native tools), it falls back to exactly
// what the handler did: one ContentDelta with the text, then Usage.
func (c *Client) StreamStep(ctx context.Context, req *ai.Request, onChunk func(ai.StreamChunk)) (*ai.Response, error) {
	var fired atomic.Bool
	sink := func(chunk ai.StreamChunk) {
		fired.Store(true)
		if onChunk != nil {
			onChunk(chunk)
		}
	}
	resp, err := c.Step(context.WithValue(ctx, chunkSinkKey{}, sink), req)
	if err != nil || onChunk == nil || fired.Load() {
		return resp, err
	}
	if resp.Text != "" {
		onChunk(ai.StreamContentDelta{Text: resp.Text})
	}
	onChunk(ai.StreamUsage{
		InputTokens:              resp.InputTokens,
		OutputTokens:             resp.OutputTokens,
		CacheReadInputTokens:     resp.CacheReadInputTokens,
		CacheCreationInputTokens: resp.CacheCreationInputTokens,
	})
	return resp, nil
}

var _ ai.StreamingProvider = (*Client)(nil)

type chunkSinkKey struct{}

// chunkSinkFrom returns the StreamStep callback carried by ctx, nil from Step.
func chunkSinkFrom(ctx context.Context) func(ai.StreamChunk) {
	f, _ := ctx.Value(chunkSinkKey{}).(func(ai.StreamChunk))
	return f
}
