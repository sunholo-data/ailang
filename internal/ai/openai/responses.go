package openai

import (
	"context"

	"github.com/sunholo/ailang/internal/ai"
)

// generateResponses uses the Responses API (/v1/responses).
// This is used for codex models that support autonomous operation.
//
// NOTE: This is a stub implementation for M3 (OpenAI Chat Completions).
// Full implementation will be added in M4 (OpenAI Responses API).
func (c *Client) generateResponses(ctx context.Context, req *ai.Request) (*ai.Response, error) {
	// For now, fall back to Chat Completions
	// M4 will implement the full Responses API
	return c.generateChat(ctx, req)
}
