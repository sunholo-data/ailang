package chatgpt

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/sunholo-data/ailang/internal/ai"
)

// DefaultBaseURL is the ChatGPT codex backend. It speaks only the streaming
// Responses API.
const DefaultBaseURL = "https://chatgpt.com/backend-api/codex"

// DefaultTimeout bounds the entire HTTP response, including continuous SSE output.
const DefaultTimeout = 10 * time.Minute

// ModelPrefix routes a model to this provider ("chatgpt/gpt-6.1-sol"); it is
// stripped before the request.
const ModelPrefix = "chatgpt/"

var (
	_ ai.Provider          = (*Client)(nil)
	_ ai.StreamingProvider = (*Client)(nil)
)

// Client calls the ChatGPT codex backend with the subscription credential.
type Client struct {
	baseURL    string
	httpClient *http.Client
	load       func() (Credential, error)
}

// Option configures a Client.
type Option func(*Client)

// WithBaseURL points the client at another endpoint (tests).
func WithBaseURL(u string) Option { return func(c *Client) { c.baseURL = strings.TrimRight(u, "/") } }

// WithTimeout overrides the total HTTP deadline (zero disables it).
// This absorbs the interim client deadline portion of #1259 without a new flag.
func WithTimeout(timeout time.Duration) Option {
	return func(c *Client) { c.httpClient.Timeout = timeout }
}

// WithCredential pins the credential instead of reading auth.json (tests).
func WithCredential(cred Credential) Option {
	return func(c *Client) { c.load = func() (Credential, error) { return cred, nil } }
}

// NewClient builds a client. The credential is read per request, so a token
// codex refreshed mid-run is picked up.
func NewClient(opts ...Option) *Client {
	c := &Client{baseURL: DefaultBaseURL, httpClient: &http.Client{Timeout: DefaultTimeout}, load: LoadCredential}
	for _, o := range opts {
		o(c)
	}
	return c
}

// Name implements ai.Provider.
func (c *Client) Name() string { return "chatgpt" }

// Generate implements ai.Provider: one turn, no tools.
func (c *Client) Generate(ctx context.Context, req *ai.Request) (*ai.Response, error) {
	return c.StreamStep(ctx, req, nil)
}

// Step implements ai.Provider: the tool-aware turn, without a chunk callback.
func (c *Client) Step(ctx context.Context, req *ai.Request) (*ai.Response, error) {
	return c.StreamStep(ctx, req, nil)
}

// StreamStep implements ai.StreamingProvider. The backend only streams, so
// Generate and Step go through here too.
func (c *Client) StreamStep(ctx context.Context, req *ai.Request, onChunk func(ai.StreamChunk)) (*ai.Response, error) {
	if len(req.InputImages) > 0 {
		return nil, ai.NewAIError(ai.CodeCapabilityNotSupported, "chatgpt: image input is not supported on this lane", false)
	}
	cred, err := c.load()
	if err != nil {
		return nil, ai.NewAIError(ai.CodeAuthFailed, err.Error(), false)
	}
	body, err := json.Marshal(buildRequest(req))
	if err != nil {
		return nil, ai.NewAIError(ai.CodeInternal, fmt.Sprintf("chatgpt: marshal request: %v", err), false)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/responses", bytes.NewReader(body))
	if err != nil {
		return nil, ai.ClassifyError(err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+cred.AccessToken)
	httpReq.Header.Set("chatgpt-account-id", cred.AccountID)
	httpReq.Header.Set("OpenAI-Beta", "responses=experimental")
	httpReq.Header.Set("originator", "ailang") // who is calling; never another client's name
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, ai.ClassifyError(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(resp.Body)
		msg := strings.TrimSpace(string(raw))
		if m := ai.ErrorEnvelopeMessage(raw); m != "" {
			msg = m
		}
		return nil, ai.ClassifyHTTPError("chatgpt", resp.StatusCode, msg)
	}
	return parseStream(resp.Body, apiModel(req.Model), onChunk)
}

// apiModel strips the routing prefix.
func apiModel(m string) string {
	if strings.HasPrefix(strings.ToLower(m), ModelPrefix) {
		return m[len(ModelPrefix):]
	}
	return m
}

type request struct {
	Model             string           `json:"model"`
	Instructions      string           `json:"instructions"`
	Input             []map[string]any `json:"input"`
	Tools             []map[string]any `json:"tools,omitempty"`
	ToolChoice        string           `json:"tool_choice,omitempty"`
	ParallelToolCalls bool             `json:"parallel_tool_calls"`
	Reasoning         map[string]any   `json:"reasoning,omitempty"`
	Stream            bool             `json:"stream"`
	Store             bool             `json:"store"`
}

// buildRequest maps an ai.Request onto Responses input items. System content
// becomes `instructions`; an assistant tool call becomes a function_call item
// and its result a function_call_output item, paired by call_id.
func buildRequest(req *ai.Request) request {
	out := request{Model: apiModel(req.Model), Stream: true, Store: false, ParallelToolCalls: true}
	var sys []string
	if req.SystemPrompt != "" {
		sys = append(sys, req.SystemPrompt)
	}
	msgs := req.Messages
	if len(msgs) == 0 && req.UserPrompt != "" {
		msgs = []ai.Message{{Role: "user", Content: req.UserPrompt}}
	}
	for _, m := range msgs {
		switch m.Role {
		case "system":
			if m.Content != "" {
				sys = append(sys, m.Content)
			}
		case "assistant":
			if m.Content != "" {
				out.Input = append(out.Input, map[string]any{"type": "message", "role": "assistant",
					"content": []map[string]any{{"type": "output_text", "text": m.Content}}})
			}
			for _, tc := range m.ToolCalls {
				out.Input = append(out.Input, map[string]any{"type": "function_call",
					"call_id": tc.ID, "name": tc.Name, "arguments": tc.Arguments})
			}
		case "tool":
			out.Input = append(out.Input, map[string]any{"type": "function_call_output",
				"call_id": m.ToolCallID, "output": m.Content})
		default:
			out.Input = append(out.Input, map[string]any{"type": "message", "role": "user",
				"content": []map[string]any{{"type": "input_text", "text": m.Content}}})
		}
	}
	out.Instructions = strings.Join(sys, "\n\n")
	for _, t := range req.Tools {
		var params any = map[string]any{"type": "object", "properties": map[string]any{}}
		if strings.TrimSpace(t.Parameters) != "" {
			var p any
			if json.Unmarshal([]byte(t.Parameters), &p) == nil {
				params = p
			}
		}
		out.Tools = append(out.Tools, map[string]any{"type": "function", "name": t.Name,
			"description": t.Description, "parameters": params, "strict": false})
	}
	if len(out.Tools) > 0 {
		out.ToolChoice = "auto"
	}
	if req.ReasoningEffort != "" {
		out.Reasoning = map[string]any{"effort": req.ReasoningEffort}
	}
	return out
}

// event is the union of the Responses stream events this client reads.
type event struct {
	Type     string `json:"type"`
	Delta    string `json:"delta"`
	Code     string `json:"code"`
	Message  string `json:"message"`
	Item     *item  `json:"item"`
	Response *struct {
		Model             string `json:"model"`
		Status            string `json:"status"`
		IncompleteDetails *struct {
			Reason string `json:"reason"`
		} `json:"incomplete_details"`
		Error *struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
		Usage *struct {
			InputTokens        int `json:"input_tokens"`
			OutputTokens       int `json:"output_tokens"`
			TotalTokens        int `json:"total_tokens"`
			InputTokensDetails struct {
				CachedTokens int `json:"cached_tokens"`
			} `json:"input_tokens_details"`
			OutputTokensDetails struct {
				ReasoningTokens int `json:"reasoning_tokens"`
			} `json:"output_tokens_details"`
		} `json:"usage"`
	} `json:"response"`
}

type item struct {
	Type      string `json:"type"`
	CallID    string `json:"call_id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// parseStream reads the SSE body into an ai.Response. A stream that ends
// without a terminal event is a protocol error, never a silent empty answer.
func parseStream(r io.Reader, model string, onChunk func(ai.StreamChunk)) (*ai.Response, error) {
	out := &ai.Response{Model: model, RequestedModel: model, ResolvedProvider: "chatgpt"}
	var text, reasoning strings.Builder
	terminal := false
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	for sc.Scan() {
		line := sc.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		payload := strings.TrimSpace(line[len("data: "):])
		if payload == "" || payload == "[DONE]" {
			continue
		}
		var ev event
		if err := json.Unmarshal([]byte(payload), &ev); err != nil {
			return nil, ai.NewAIError(ai.CodeProtocolError, fmt.Sprintf("chatgpt: bad stream event: %v", err), true)
		}
		switch ev.Type {
		case "response.output_text.delta":
			text.WriteString(ev.Delta)
			if onChunk != nil && ev.Delta != "" {
				onChunk(ai.StreamContentDelta{Text: ev.Delta})
			}
		case "response.reasoning_summary_text.delta", "response.reasoning_text.delta":
			reasoning.WriteString(ev.Delta)
			if onChunk != nil && ev.Delta != "" {
				onChunk(ai.StreamThinkingDelta{Text: ev.Delta})
			}
		case "response.output_item.done":
			if ev.Item != nil && ev.Item.Type == "function_call" {
				out.ToolCalls = append(out.ToolCalls, ai.ToolCall{ID: ev.Item.CallID, Name: ev.Item.Name, Arguments: ev.Item.Arguments})
			}
		case "error":
			return nil, ai.NewAIError(ai.CodeProtocolError, fmt.Sprintf("chatgpt: stream error %s: %s", ev.Code, ev.Message), true)
		case "response.failed":
			msg := "response failed"
			if ev.Response != nil && ev.Response.Error != nil {
				msg = ev.Response.Error.Code + ": " + ev.Response.Error.Message
			}
			return nil, ai.NewAIError(ai.CodeProtocolError, "chatgpt: "+msg, true)
		case "response.completed", "response.incomplete":
			terminal = true
			out.FinishReason = "stop"
			if len(out.ToolCalls) > 0 {
				out.FinishReason = "tool_calls"
			}
			if ev.Response != nil {
				if ev.Response.Model != "" {
					out.Model = ev.Response.Model
				}
				if ev.Type == "response.incomplete" || ev.Response.Status == "incomplete" {
					out.FinishReason = "length"
					if d := ev.Response.IncompleteDetails; d != nil && d.Reason != "" && d.Reason != "max_output_tokens" {
						out.FinishReason = d.Reason
					}
				}
				if u := ev.Response.Usage; u != nil {
					out.InputTokens = u.InputTokens
					out.OutputTokens = u.OutputTokens
					out.TotalTokens = u.TotalTokens
					out.ReasonTokens = u.OutputTokensDetails.ReasoningTokens
					out.CachedTokens = u.InputTokensDetails.CachedTokens
					out.CacheReadInputTokens = u.InputTokensDetails.CachedTokens
					if onChunk != nil {
						onChunk(ai.StreamUsage{InputTokens: u.InputTokens, OutputTokens: u.OutputTokens,
							CacheReadInputTokens: u.InputTokensDetails.CachedTokens})
					}
				}
			}
		}
	}
	if err := sc.Err(); err != nil {
		return nil, ai.ClassifyError(err)
	}
	if !terminal {
		return nil, ai.NewAIError(ai.CodeProtocolError, "chatgpt: stream ended without response.completed", true)
	}
	out.Text = text.String()
	out.Reasoning = reasoning.String()
	return out, nil
}
