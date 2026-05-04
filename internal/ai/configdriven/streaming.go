package configdriven

import (
	"encoding/json"
	"fmt"

	"github.com/sunholo-data/ailang/internal/ai"
	"github.com/sunholo-data/ailang/internal/pkg"
)

// HeaderPair is one (name, value) entry for the StreamSSEPost headers list.
// Mirrors the AILANG-side `{name, value}` record shape that StreamSSEPost
// expects in its config arg.
type HeaderPair struct {
	Name  string
	Value string
}

// StreamRequest is the prepared input to effects.StreamSSEPost: the URL to
// POST to, the JSON body, and the resolved auth + content headers.
type StreamRequest struct {
	URL     string
	Body    string
	Headers []HeaderPair
}

// BuildStreamRequest prepares an SSE-POST request from an [[ai_provider]] spec
// plus a model + serialised messages. Resolves env-var references at call
// time so credentials don't leak into start-time globals. Returns an
// AI-shaped error on capability mismatch or auth failure so the caller can
// short-circuit without hitting the network.
//
// Callers (effects/ai_streaming.go): do NOT bypass this — Stream-effect-only
// code paths break the AI cap (D1) and budget tracking (D11). All AI
// streaming dispatch goes through here so the budget/cap/span machinery
// applies uniformly.
func BuildStreamRequest(spec *pkg.AIProviderSpec, model string, messagesJSON string) (*StreamRequest, *ai.ProviderError) {
	if spec == nil {
		return nil, ai.NewProviderError("", 0, "config-driven streaming: nil spec", nil)
	}
	if !spec.Streaming.Enabled {
		return nil, ai.NewProviderError(spec.Name, 0,
			fmt.Sprintf("provider %q does not enable streaming (set [ai_provider.streaming] enabled = true)", spec.Name), nil)
	}
	if !spec.Capabilities.Streaming {
		return nil, ai.NewProviderError(spec.Name, 0,
			fmt.Sprintf("provider %q advertises capabilities.streaming = false", spec.Name), nil)
	}

	// Models allow-list enforcement (mirrors Provider.Generate).
	if len(spec.Models.Allowed) > 0 {
		ok := false
		for _, m := range spec.Models.Allowed {
			if m == model {
				ok = true
				break
			}
		}
		if !ok {
			return nil, ai.NewProviderError(spec.Name, 0,
				fmt.Sprintf("model %q is not in the allowed list for provider %q", model, spec.Name), nil)
		}
	}

	// Construct the AI request body.
	// We accept messagesJSON as a pre-serialised list so AILANG-side callers
	// can pass typed [Message] records without us reaching across the FFI to
	// reconstruct them. The body builder reads back from that JSON shape.
	req := &ai.Request{
		Model:        model,
		UserPrompt:   "", // overridden below by message reconstruction
		SystemPrompt: "",
	}
	if err := populateRequestFromMessagesJSON(req, messagesJSON); err != nil {
		return nil, ai.NewProviderError(spec.Name, 0,
			fmt.Sprintf("invalid messages_json: %s", err.Error()), err)
	}

	bodyBytes, err := buildRequestBody(spec.RequestShape, req)
	if err != nil {
		return nil, ai.NewProviderError(spec.Name, 0, err.Error(), err)
	}

	// Inject `stream: true` for OpenAI-shaped requests so the upstream
	// returns SSE rather than a single JSON envelope. Most OpenAI-compat
	// servers (vLLM, OpenRouter, Together, Groq) follow this convention.
	// Anthropic doesn't need a `stream` field — the /messages endpoint
	// switches to SSE based on Accept header / a separate streaming flag,
	// which we let the provider config encode via auth_headers if needed.
	if spec.RequestShape == "openai_chat" || spec.RequestShape == "simple_completion" {
		bodyBytes, err = injectStreamFlag(bodyBytes)
		if err != nil {
			return nil, ai.NewProviderError(spec.Name, 0, err.Error(), err)
		}
	}

	// Build headers from the auth shape and any custom auth_headers.
	headers, hdrErr := buildStreamHeaders(spec)
	if hdrErr != nil {
		return nil, ai.NewProviderError(spec.Name, 0, hdrErr.Error(), hdrErr)
	}

	url := spec.EffectiveStreamingEndpoint()
	return &StreamRequest{
		URL:     url,
		Body:    string(bodyBytes),
		Headers: headers,
	}, nil
}

// populateRequestFromMessagesJSON reads a JSON array of {role, content}
// records and threads them into the ai.Request fields used by the existing
// shape templates (SystemPrompt + UserPrompt for OpenAI/Ollama; the
// Anthropic shape reads UserPrompt + SystemPrompt the same way).
//
// Multi-turn conversations: only the last user message is placed in
// UserPrompt; earlier user/assistant turns would need a richer ai.Request
// representation than v1 provides. For v1 we accept a single round-trip;
// motoko_agent's loop is conversation-as-context-window so this is fine.
func populateRequestFromMessagesJSON(req *ai.Request, messagesJSON string) error {
	type msg struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	var msgs []msg
	if err := json.Unmarshal([]byte(messagesJSON), &msgs); err != nil {
		return fmt.Errorf("messages must be JSON array of {role, content}: %w", err)
	}
	if len(msgs) == 0 {
		return fmt.Errorf("messages array is empty")
	}

	// Take the last system + last user. (v1 is single-shot.)
	for _, m := range msgs {
		switch m.Role {
		case "system":
			req.SystemPrompt = m.Content
		case "user":
			req.UserPrompt = m.Content
		case "assistant":
			// Ignored in v1; v2 multi-turn would extend ai.Request shape.
		}
	}
	if req.UserPrompt == "" {
		return fmt.Errorf("messages must include at least one user message")
	}
	return nil
}

// injectStreamFlag rewrites a JSON object body to add `"stream": true` at
// the top level. Used for OpenAI-compat endpoints.
func injectStreamFlag(body []byte) ([]byte, error) {
	var obj map[string]any
	if err := json.Unmarshal(body, &obj); err != nil {
		return nil, fmt.Errorf("inject stream flag: body is not a JSON object: %w", err)
	}
	obj["stream"] = true
	return json.Marshal(obj)
}

// buildStreamHeaders produces the headers list that StreamSSEPost expects.
// Mirrors applyAuth's logic but emits a slice of HeaderPair rather than
// mutating an *http.Request. Resolves env-var references at call time.
func buildStreamHeaders(spec *pkg.AIProviderSpec) ([]HeaderPair, error) {
	out := []HeaderPair{}

	if spec.HasNamedAuth() {
		switch spec.Auth.Type {
		case "none":
			// no-op
		case "bearer":
			secret, err := requireEnv(spec.Auth.Env)
			if err != nil {
				return nil, err
			}
			out = append(out, HeaderPair{Name: "Authorization", Value: "Bearer " + secret})
		case "x-api-key":
			secret, err := requireEnv(spec.Auth.Env)
			if err != nil {
				return nil, err
			}
			out = append(out, HeaderPair{Name: "x-api-key", Value: secret})
		case "query-param":
			// Streaming variant: query-param auth is not threaded through here
			// because StreamSSEPost takes a URL string and we'd have to mutate
			// it. For v1 we error — most LLM providers use bearer or x-api-key
			// for streaming anyway. If demand emerges, extend StreamRequest.URL
			// to include the param via url.Values.Encode at construction time.
			return nil, fmt.Errorf("auth type %q is not supported for streaming in v1; use bearer, x-api-key, or auth_headers", spec.Auth.Type)
		default:
			return nil, fmt.Errorf("unknown auth.type %q", spec.Auth.Type)
		}
	}

	for headerName, headerVal := range spec.AuthHeaders {
		resolved, err := interpolateEnv(headerVal)
		if err != nil {
			return nil, fmt.Errorf("auth_headers[%q]: %w", headerName, err)
		}
		out = append(out, HeaderPair{Name: headerName, Value: resolved})
	}

	return out, nil
}
