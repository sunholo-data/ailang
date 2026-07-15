package gemini

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/sunholo-data/ailang/internal/ai"
)

// Compile-time guarantee: *Client satisfies ai.StreamingProvider so the
// type-assertion in ai.Handler.StepWithStream succeeds.
var _ ai.StreamingProvider = (*Client)(nil)

// StreamStep is the streaming variant of Step, introduced by
// M-AI-STEP-STREAMING (v0.18.7). It targets Gemini's streamGenerateContent
// endpoint with alt=sse — the response is SSE-framed (one JSON object per
// "data: " line, no event-type prefix), each object having the same shape
// as a single generateContent response chunk.
//
// Per-chunk callback semantics:
//   - ai.StreamContentDelta is fired once per non-empty text part in any
//     candidate chunk.
//   - ai.StreamUsage is fired exactly once after the stream closes,
//     carrying the FINAL usageMetadata (Gemini emits a running usage block
//     on every chunk; we only surface the last one to match the
//     Anthropic/OpenAI semantics where Usage means "this is what the call
//     consumed").
//   - functionCall parts are accumulated into resp.ToolCalls (turn-index
//     based deterministic IDs, mirroring the non-streaming Step path).
//     Tool calls are NOT streamed individually as ToolCallDelta — that's
//     deferred to Phase 2.
//
// Implements ai.StreamingProvider so ai.Handler.StepWithStream type-asserts
// successfully and dispatches natively rather than NO-OP-falling-back.
func (c *Client) StreamStep(ctx context.Context, req *ai.Request, onChunk func(ai.StreamChunk)) (*ai.Response, error) {
	if req.Routing != nil && (req.Routing.HasRouting() || req.Routing.PriceCapSet()) {
		return nil, ai.NewAIError(ai.CodeCapabilityNotSupported,
			"gemini: AIRoutingPolicy not supported; use openrouter instead", false)
	}
	if len(req.CacheBreakpoints) > 0 {
		ai.WarnOnceCacheHintIgnored("gemini", "no_explicit_api")
	}

	apiReq, buildErr := buildStepRequest(req)
	if buildErr != nil {
		return nil, buildErr
	}

	url, urlErr := c.buildStreamURL(req.Model)
	if urlErr != nil {
		return nil, ai.ClassifyError(urlErr)
	}

	jsonBody, marshalErr := json.Marshal(apiReq)
	if marshalErr != nil {
		return nil, ai.NewAIError(ai.CodeInternal,
			fmt.Sprintf("gemini: failed to marshal stream request: %v", marshalErr), false)
	}

	httpReq, reqErr := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	if reqErr != nil {
		return nil, ai.NewAIError(ai.CodeInternal,
			fmt.Sprintf("gemini: failed to build stream request: %v", reqErr), false)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	if authErr := c.addAuth(httpReq); authErr != nil {
		return nil, ai.NewAIError(ai.CodeAuthFailed,
			fmt.Sprintf("gemini: %v", authErr), false)
	}

	httpResp, doErr := c.httpClient.Do(httpReq)
	if doErr != nil {
		return nil, ai.ClassifyError(doErr)
	}
	defer func() { _ = httpResp.Body.Close() }()

	if httpResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(httpResp.Body)
		return nil, classifyGeminiHTTP(httpResp.StatusCode, body)
	}

	return parseGeminiSSEStream(req, httpResp.Body, onChunk)
}

// parseGeminiSSEStream consumes the SSE event stream from a Gemini
// streamGenerateContent response and produces a typed *ai.Response. The
// onChunk callback fires per text part and once at end-of-stream with the
// final usage block.
//
// SSE wire format (no event-type lines, just data: <json>):
//
//	data: {"candidates":[{"content":{"parts":[{"text":"Hello"}]}}],"usageMetadata":{...}}
//	data: {"candidates":[{"content":{"parts":[{"text":" world"}]}}],"usageMetadata":{...}}
//	data: {"candidates":[{"finishReason":"STOP","content":{"parts":[]}}],"usageMetadata":{...}}
//
// Note: Gemini does NOT emit a [DONE] sentinel — the stream simply closes.
// Token counts on intermediate chunks are running totals; the LAST chunk's
// usageMetadata is the authoritative final count.
func parseGeminiSSEStream(req *ai.Request, body io.Reader, onChunk func(ai.StreamChunk)) (*ai.Response, error) {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	// turnIndex = count of assistant messages already in req.Messages, used
	// for deterministic tool-call ID generation (matches non-streaming Step).
	turnIndex := 0
	for _, m := range req.Messages {
		if m.Role == "assistant" {
			turnIndex++
		}
	}

	out := &ai.Response{Model: req.Model}
	var textParts []string
	var lastUsage usageMetadata
	lastFinish := ""

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "" {
			continue
		}

		var chunk stepRawResponse
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			return nil, ai.NewAIError(ai.CodeProtocolError,
				fmt.Sprintf("gemini: failed to parse stream chunk: %v", err), false)
		}

		if chunk.ModelVersion != "" {
			out.Model = chunk.ModelVersion
		}

		// Gemini emits running usage on every chunk; keep the LAST one.
		// Zero-valued usageMetadata is identifiable by all-zero counts —
		// don't overwrite a populated one with a zero one.
		if chunk.UsageMetadata.PromptTokenCount > 0 ||
			chunk.UsageMetadata.CandidatesTokenCount > 0 ||
			chunk.UsageMetadata.TotalTokenCount > 0 {
			lastUsage = chunk.UsageMetadata
		}

		if len(chunk.Candidates) == 0 {
			continue
		}
		cand := chunk.Candidates[0]
		if cand.FinishReason != "" {
			lastFinish = cand.FinishReason
		}

		for _, p := range cand.Content.Parts {
			if p.FunctionCall != nil {
				argsJSON, marshalErr := json.Marshal(p.FunctionCall.Args)
				if marshalErr != nil {
					return nil, ai.NewAIError(ai.CodeProtocolError,
						fmt.Sprintf("gemini: failed to re-encode tool args for %q: %v", p.FunctionCall.Name, marshalErr), false)
				}
				out.ToolCalls = append(out.ToolCalls, ai.ToolCall{
					ID:        fmt.Sprintf("%d_%d", turnIndex, len(out.ToolCalls)),
					Name:      p.FunctionCall.Name,
					Arguments: string(argsJSON),
				})
				continue
			}
			// Thought parts (v0.18.8): Gemini 2.5+ thinking models emit
			// reasoning as parts with thought:true. Reasoning text is NOT
			// accumulated into textParts (which becomes Response.Text);
			// it flows ONLY through the StreamThinkingDelta callback so
			// consumers can render it separately or drop it.
			if p.Thought {
				if onChunk != nil && p.Text != "" {
					onChunk(ai.StreamThinkingDelta{Text: p.Text})
				}
				continue
			}
			if p.Text != "" {
				textParts = append(textParts, p.Text)
				if onChunk != nil {
					onChunk(ai.StreamContentDelta{Text: p.Text})
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, ai.ClassifyError(err)
	}

	out.Text = strings.Join(textParts, "")
	out.InputTokens = lastUsage.PromptTokenCount
	out.OutputTokens = lastUsage.CandidatesTokenCount
	out.TotalTokens = lastUsage.TotalTokenCount
	out.ReasonTokens = lastUsage.ThoughtsTokenCount
	out.CacheReadInputTokens = lastUsage.CachedContentTokenCount

	finish := mapFinishReason(lastFinish)
	if len(out.ToolCalls) > 0 {
		finish = "tool_calls"
	}
	out.FinishReason = finish

	if onChunk != nil {
		onChunk(ai.StreamUsage{
			InputTokens:          out.InputTokens,
			OutputTokens:         out.OutputTokens,
			CacheReadInputTokens: out.CacheReadInputTokens,
			// Gemini doesn't surface cache-creation as a separate count
			// (CachedContent API tracks writes via a different control plane).
			CacheCreationInputTokens: 0,
		})
	}

	return out, nil
}
