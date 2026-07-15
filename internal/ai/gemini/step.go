package gemini

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/sunholo-data/ailang/internal/ai"
)

// Step is the multi-turn / tool-aware completion entry point introduced by
// M-AI-TOOL-LOOP (v0.17.0). It translates req.Messages + req.Tools into the
// Gemini generateContent API's function-calling shape (functionCall /
// functionResponse parts on contents[].parts[]) and parses functionCall
// parts in the response into resp.ToolCalls.
//
// Tool-call ID generation: Gemini does NOT assign tool-call IDs natively
// (unlike Anthropic and OpenAI). To keep the round-trip stable across loop
// turns, this adapter generates deterministic IDs of the form
// "<turn_index>_<call_index>" where turn_index is the count of assistant
// messages already present in req.Messages (so the first turn's calls get
// "0_0", "0_1", ...; the third turn's calls get "2_0", "2_1", ...) and
// call_index is the position of the functionCall part within the response
// parts. Loop drivers should treat the ID as opaque; only this adapter
// produces or consumes it.
//
// Errors returned are always *ai.AIError (typed) so the AILANG-side
// _ai_call_result / _ai_step builtins can assert on Code/Retryable.
func (c *Client) Step(ctx context.Context, req *ai.Request) (*ai.Response, error) {
	if req.Routing != nil && (req.Routing.HasRouting() || req.Routing.PriceCapSet()) {
		return nil, ai.NewAIError(ai.CodeCapabilityNotSupported,
			"gemini: AIRoutingPolicy not supported; use openrouter instead", false)
	}
	// M-AI-PROMPT-CACHING (v0.18.4): Gemini exposes Context Caching via a
	// separate async API (CachedContent create + reference by ID), which
	// doesn't fit the synchronous CacheBreakpoint hint shape. Warn once
	// per session and proceed without caching. Explicit CachedContent
	// integration deferred to a Phase 2 design.
	if len(req.CacheBreakpoints) > 0 {
		ai.WarnOnceCacheHintIgnored("gemini", "no_explicit_api")
	}

	apiReq, err := buildStepRequest(req)
	if err != nil {
		return nil, err
	}

	url, urlErr := c.buildURL(req.Model)
	if urlErr != nil {
		return nil, ai.ClassifyError(urlErr)
	}

	jsonBody, marshalErr := json.Marshal(apiReq)
	if marshalErr != nil {
		return nil, ai.NewAIError(ai.CodeInternal,
			fmt.Sprintf("gemini: failed to marshal request: %v", marshalErr), false)
	}

	httpReq, reqErr := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	if reqErr != nil {
		return nil, ai.NewAIError(ai.CodeInternal,
			fmt.Sprintf("gemini: failed to build request: %v", reqErr), false)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if authErr := c.addAuth(httpReq); authErr != nil {
		return nil, ai.NewAIError(ai.CodeAuthFailed,
			fmt.Sprintf("gemini: %v", authErr), false)
	}

	httpResp, doErr := c.httpClient.Do(httpReq)
	if doErr != nil {
		return nil, ai.ClassifyError(doErr)
	}
	defer func() { _ = httpResp.Body.Close() }()

	body, readErr := io.ReadAll(httpResp.Body)
	if readErr != nil {
		return nil, ai.NewAIError(ai.CodeConnectionFailed,
			fmt.Sprintf("gemini: failed to read response body: %v", readErr), true)
	}

	if httpResp.StatusCode != http.StatusOK {
		return nil, classifyGeminiHTTP(httpResp.StatusCode, body)
	}

	return parseStepResponse(req, body)
}

// buildStepRequest translates req.Messages + req.Tools into the Gemini
// generateContent body. It also lifts req.SystemPrompt to the top-level
// systemInstruction field (system-role messages in req.Messages are skipped).
func buildStepRequest(req *ai.Request) (*generateRequest, error) {
	out := &generateRequest{
		Contents: make([]content, 0, len(req.Messages)),
	}

	if req.SystemPrompt != "" {
		out.SystemInstruction = &content{
			Parts: []part{{Text: req.SystemPrompt}},
		}
	}

	// Build a map ToolCallID → tool name from prior assistant ToolCalls so
	// tool-result messages (Role="tool" or Role="user" with ToolCallID) can
	// emit a functionResponse with the correct name field. Gemini requires
	// a name on every functionResponse — we recover it from the prior
	// assistant turn that emitted the call.
	idToName := map[string]string{}
	for _, m := range req.Messages {
		if m.Role == "assistant" {
			for _, tc := range m.ToolCalls {
				if tc.ID != "" {
					idToName[tc.ID] = tc.Name
				}
			}
		}
	}

	for _, m := range req.Messages {
		switch m.Role {
		case "system":
			// System messages are surfaced via systemInstruction at the top
			// level; skip in contents.
			continue

		case "user":
			if m.ToolCallID != "" {
				// User-role tool result (rare but supported — same shape as
				// Role="tool"). Build a functionResponse part.
				name := idToName[m.ToolCallID]
				if name == "" {
					// Fall back to raw text if we can't recover the tool name.
					out.Contents = append(out.Contents, content{
						Role:  "user",
						Parts: []part{{Text: m.Content}},
					})
					continue
				}
				out.Contents = append(out.Contents, content{
					Role: "user",
					Parts: []part{{
						FunctionResponse: &functionResponse{
							Name:     name,
							Response: map[string]interface{}{"content": m.Content},
						},
					}},
				})
				continue
			}
			out.Contents = append(out.Contents, content{
				Role:  "user",
				Parts: []part{{Text: m.Content}},
			})

		case "assistant":
			parts := make([]part, 0, 1+len(m.ToolCalls))
			if m.Content != "" {
				parts = append(parts, part{Text: m.Content})
			}
			for _, tc := range m.ToolCalls {
				// Decode the JSON-string Arguments into a map so it serializes
				// as a JSON OBJECT (Gemini requires args: object, not args: string).
				var args map[string]interface{}
				if tc.Arguments != "" {
					if err := json.Unmarshal([]byte(tc.Arguments), &args); err != nil {
						return nil, ai.NewAIError(ai.CodeProtocolError,
							fmt.Sprintf("gemini: tool call %q has invalid JSON arguments: %v", tc.Name, err), false)
					}
				}
				if args == nil {
					args = map[string]interface{}{}
				}
				parts = append(parts, part{
					FunctionCall: &functionCall{
						Name: tc.Name,
						Args: args,
					},
				})
			}
			// Gemini requires at least one part on every content entry.
			if len(parts) == 0 {
				parts = append(parts, part{Text: ""})
			}
			out.Contents = append(out.Contents, content{
				Role:  "model",
				Parts: parts,
			})

		case "tool":
			name := idToName[m.ToolCallID]
			if name == "" {
				// No matching prior tool call — fall back to plain text in
				// a user turn so the model still sees the content.
				out.Contents = append(out.Contents, content{
					Role:  "user",
					Parts: []part{{Text: m.Content}},
				})
				continue
			}
			out.Contents = append(out.Contents, content{
				Role: "user",
				Parts: []part{{
					FunctionResponse: &functionResponse{
						Name:     name,
						Response: map[string]interface{}{"content": m.Content},
					},
				}},
			})

		default:
			// Unknown role — degrade to user text rather than dropping content.
			out.Contents = append(out.Contents, content{
				Role:  "user",
				Parts: []part{{Text: m.Content}},
			})
		}
	}

	// Tools → top-level functionDeclarations.
	if len(req.Tools) > 0 {
		decls := make([]functionDeclaration, 0, len(req.Tools))
		for _, ts := range req.Tools {
			var params map[string]interface{}
			if ts.Parameters != "" {
				if err := json.Unmarshal([]byte(ts.Parameters), &params); err != nil {
					return nil, ai.NewAIError(ai.CodeSchemaValidation,
						fmt.Sprintf("gemini: tool %q has invalid Parameters JSON Schema: %v", ts.Name, err), false)
				}
			}
			decls = append(decls, functionDeclaration{
				Name:        ts.Name,
				Description: ts.Description,
				Parameters:  params,
			})
		}
		out.Tools = []toolBlock{{FunctionDeclarations: decls}}
	}

	// Generation config: same defaults as Generate (omit zero values).
	if req.MaxTokens > 0 || req.Temperature > 0 {
		out.GenerationConfig = &generationConfig{}
		if req.MaxTokens > 0 {
			out.GenerationConfig.MaxOutputTokens = req.MaxTokens
		}
		if req.Temperature > 0 {
			out.GenerationConfig.Temperature = req.Temperature
		}
	}

	return out, nil
}

// parseStepResponse converts a successful Gemini generateContent response
// body into an ai.Response. Tool-call IDs are generated deterministically
// from the turn_index (count of prior assistant messages) and call_index
// (position within this turn's parts).
func parseStepResponse(req *ai.Request, body []byte) (*ai.Response, error) {
	var raw stepRawResponse
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, ai.NewAIError(ai.CodeProtocolError,
			fmt.Sprintf("gemini: failed to parse response: %v", err), false)
	}
	if len(raw.Candidates) == 0 {
		msg := "gemini: response contained no candidates"
		if raw.PromptFeedback.BlockReason != "" {
			msg += " (blocked: " + raw.PromptFeedback.BlockReason + ")"
		}
		return nil, ai.NewAIError(ai.CodeProtocolError, msg, false)
	}

	cand := raw.Candidates[0]

	// Compute turn_index = count of assistant messages already in req.Messages.
	turnIndex := 0
	for _, m := range req.Messages {
		if m.Role == "assistant" {
			turnIndex++
		}
	}

	var textParts []string
	var toolCalls []ai.ToolCall
	for _, p := range cand.Content.Parts {
		if p.FunctionCall != nil {
			argsJSON, marshalErr := json.Marshal(p.FunctionCall.Args)
			if marshalErr != nil {
				return nil, ai.NewAIError(ai.CodeProtocolError,
					fmt.Sprintf("gemini: failed to re-encode tool args for %q: %v", p.FunctionCall.Name, marshalErr), false)
			}
			toolCalls = append(toolCalls, ai.ToolCall{
				ID:        fmt.Sprintf("%d_%d", turnIndex, len(toolCalls)),
				Name:      p.FunctionCall.Name,
				Arguments: string(argsJSON),
			})
			continue
		}
		if p.Text != "" {
			textParts = append(textParts, p.Text)
		}
	}

	finish := mapFinishReason(cand.FinishReason)
	if len(toolCalls) > 0 {
		finish = "tool_calls"
	}

	model := raw.ModelVersion
	if model == "" {
		model = req.Model
	}

	return &ai.Response{
		Text:                 strings.Join(textParts, "\n"),
		InputTokens:          raw.UsageMetadata.PromptTokenCount,
		OutputTokens:         raw.UsageMetadata.CandidatesTokenCount,
		TotalTokens:          raw.UsageMetadata.TotalTokenCount,
		ReasonTokens:         raw.UsageMetadata.ThoughtsTokenCount,
		CacheReadInputTokens: raw.UsageMetadata.CachedContentTokenCount,
		Model:                model,
		ToolCalls:            toolCalls,
		FinishReason:         finish,
	}, nil
}

// mapFinishReason maps Gemini's finishReason vocabulary onto the normalized
// ai.Response.FinishReason values. Caller overrides "stop" with "tool_calls"
// when functionCall parts are present.
func mapFinishReason(geminiReason string) string {
	switch geminiReason {
	case "STOP":
		return "stop"
	case "MAX_TOKENS":
		return "length"
	case "SAFETY", "RECITATION", "BLOCKLIST", "PROHIBITED_CONTENT", "SPII":
		return "error"
	case "":
		// Defensive — empty finishReason is treated as natural stop.
		return "stop"
	default:
		return "error"
	}
}

// classifyGeminiHTTP wraps ai.ClassifyHTTPError with Gemini-specific body
// parsing so the AIError.Message carries the human-readable error.message
// field rather than the raw JSON envelope.
func classifyGeminiHTTP(statusCode int, body []byte) *ai.AIError {
	var er errorResponse
	msg := string(body)
	if json.Unmarshal(body, &er) == nil && er.Error.Message != "" {
		msg = er.Error.Message
	}
	return ai.ClassifyHTTPError("gemini", statusCode, msg)
}
