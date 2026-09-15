package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// JSONCall is one JSON-over-HTTP round trip to a provider endpoint. Every
// non-streaming client (anthropic, openai, openrouter, gemini, configdriven)
// used to repeat the same fifteen lines around it — build request, set
// headers, Do, ReadAll, status check, hoist the error envelope, Unmarshal —
// with the error shapes drifting between copies (M-V1-SIMPLIFY-S3 M4).
type JSONCall struct {
	// Provider names the client in errors ("anthropic", "openai", a config-driven spec name).
	Provider string
	// Client is the *http.Client to use; nil means http.DefaultClient.
	Client *http.Client
	// URL is the endpoint; the method is always POST.
	URL string
	// Headers are added to the request. Content-Type defaults to application/json.
	Headers http.Header
	// Body is the already-marshalled JSON request body.
	Body []byte
	// ErrorMessage hoists the provider's human-readable message out of a
	// non-2xx body. nil uses ErrorEnvelopeMessage, the {"error":{"message"}}
	// shape every built-in provider speaks; an empty return keeps the raw body.
	ErrorMessage func(body []byte) string
}

// JSONResult is the wire-level outcome. It is returned alongside the error
// whenever a response was received, so callers can log the exact bytes or
// read the status without re-parsing the error.
type JSONResult struct {
	StatusCode int
	Body       []byte
	// WallMS is request-sent → body-fully-read, the per-call latency datum
	// for route A/Bs (M-LYCEUM-PROVIDER M3). Non-streaming, so TTFT is
	// unobservable here and stays unmeasured.
	WallMS int64
}

// DoJSON sends the call, reads the whole body, and decodes a 2xx body into
// out (nil skips decoding). Every failure is a *ProviderError carrying the
// provider name, the status (0 before a response), the hoisted message and
// the wall time — the shape Generate callers return as-is. Its Err is the
// classification: a non-2xx wraps ClassifyHTTPError (so a 429 is
// CodeRateLimit, a spent quota is CodeBudgetExhausted), an undecodable 2xx
// wraps CodeProtocolError, and a transport failure wraps the raw error so
// errors.Is(err, context.DeadlineExceeded) still holds. Step callers hand
// the error to ClassifyError, which unwraps that AIError unchanged.
func DoJSON(ctx context.Context, call JSONCall, out any) (*JSONResult, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, call.URL, bytes.NewReader(call.Body))
	if err != nil {
		return nil, &ProviderError{Provider: call.Provider, Message: "failed to create request", Err: err}
	}
	for k, vs := range call.Headers {
		for _, v := range vs {
			httpReq.Header.Add(k, v)
		}
	}
	if httpReq.Header.Get("Content-Type") == "" {
		httpReq.Header.Set("Content-Type", "application/json")
	}
	client := call.Client
	if client == nil {
		client = http.DefaultClient
	}

	start := time.Now()
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, &ProviderError{Provider: call.Provider, Message: "request failed", Err: err, WallMS: time.Since(start).Milliseconds()}
	}
	defer func() { _ = resp.Body.Close() }()

	res := &JSONResult{StatusCode: resp.StatusCode}
	body, err := io.ReadAll(resp.Body)
	res.WallMS = time.Since(start).Milliseconds()
	if err != nil {
		return res, &ProviderError{Provider: call.Provider, StatusCode: resp.StatusCode, Message: "failed to read response", Err: err, WallMS: res.WallMS}
	}
	res.Body = body

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := string(body)
		hoist := call.ErrorMessage
		if hoist == nil {
			hoist = ErrorEnvelopeMessage
		}
		if m := hoist(body); m != "" {
			msg = m
		}
		return res, &ProviderError{
			Provider:   call.Provider,
			StatusCode: resp.StatusCode,
			Message:    msg,
			Err:        ClassifyHTTPError(call.Provider, resp.StatusCode, msg),
			WallMS:     res.WallMS,
		}
	}

	if out != nil {
		if err := json.Unmarshal(body, out); err != nil {
			msg := fmt.Sprintf("failed to parse response: body is not valid JSON: %v", err)
			return res, &ProviderError{
				Provider: call.Provider,
				Message:  msg,
				Err:      NewAIError(CodeProtocolError, call.Provider+": "+msg, false),
				WallMS:   res.WallMS,
			}
		}
	}
	return res, nil
}

// ErrorEnvelopeMessage returns error.message from the {"error":{"message":
// "..."}} envelope that Anthropic, OpenAI, OpenRouter and Gemini all use for
// non-2xx bodies, or "" when the body is not that shape.
func ErrorEnvelopeMessage(body []byte) string {
	var env struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if json.Unmarshal(body, &env) != nil {
		return ""
	}
	return env.Error.Message
}
