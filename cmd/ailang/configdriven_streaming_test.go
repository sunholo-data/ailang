package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/ai"
	"github.com/sunholo-data/ailang/internal/ai/configdriven"
	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/pkg"
)

// streamTestContext returns an EffContext configured for streaming tests.
// Mirrors newSSETestContext from internal/effects/stream_sse_test.go but
// also grants the AI cap (which the streaming dispatch requires).
func streamTestContext(t *testing.T) *effects.EffContext {
	t.Helper()
	ctx := effects.NewEffContext(nil)
	ctx.Grant(effects.NewCapability("Stream"))
	ctx.Grant(effects.NewCapability("AI"))
	ctx.Grant(effects.NewCapability("Net"))
	ctx.Stream = effects.NewStreamContext()
	ctx.Stream.AllowHTTP = true
	ctx.Stream.AllowLocalhost = true
	ctx.Stream.IdleTimeout = 3 * time.Second
	ctx.Stream.MaxDuration = 10 * time.Second
	return ctx
}

// newOpenAISSEServer constructs a mock OpenAI-compatible SSE server. The
// `requestObserver` callback fires on each incoming request so tests can
// assert headers and request body shape.
func newOpenAISSEServer(t *testing.T, observer func(r *http.Request, body []byte)) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if observer != nil {
			observer(r, body)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		flusher, _ := w.(http.Flusher)
		// A realistic OpenAI streaming response: two delta events + [DONE]
		_, _ = fmt.Fprintln(w, `data: {"choices":[{"delta":{"content":"Hello"}}]}`)
		_, _ = fmt.Fprintln(w, "")
		flusher.Flush()
		_, _ = fmt.Fprintln(w, `data: {"choices":[{"delta":{"content":" world"}}]}`)
		_, _ = fmt.Fprintln(w, "")
		flusher.Flush()
		_, _ = fmt.Fprintln(w, "data: [DONE]")
		_, _ = fmt.Fprintln(w, "")
		flusher.Flush()
	}))
}

// registerOpenAITestProvider creates a config-driven provider pointed at the
// given URL. Caller is responsible for resetting the global registry.
func registerOpenAITestProvider(t *testing.T, name, url string, withAuthEnv string) {
	t.Helper()
	spec := &pkg.AIProviderSpec{
		SchemaVersion: 1,
		Name:          name,
		Endpoint:      url,
		RequestShape:  "openai_chat",
		ResponsePath:  "$.choices[0].message.content",
		Capabilities:  pkg.AIProviderCapabilities{Streaming: true},
		Streaming: pkg.AIProviderStreaming{
			Enabled:      true,
			DeltaPath:    "$.choices[0].delta.content",
			DoneSentinel: "[DONE]",
		},
	}
	if withAuthEnv != "" {
		spec.Auth = pkg.AIProviderAuth{Type: "bearer", Env: withAuthEnv}
	} else {
		spec.Auth = pkg.AIProviderAuth{Type: "none"}
	}
	provider := configdriven.New(spec)
	if err := ai.GlobalProviderRegistry.Register(name, provider, "test://"+name); err != nil {
		t.Fatalf("register provider: %v", err)
	}
}

// TestAIStreamCall_OpenAIShape_Success: full happy-path flow.
//
// Verifies:
// - Provider lookup via registry works
// - Body construction injects stream:true at the top level
// - bearer auth header reaches the upstream
// - StreamSSEPost connects + the SSE response flows back through StreamConn
// - Result wrapping is correct
func TestAIStreamCall_OpenAIShape_Success(t *testing.T) {
	ai.GlobalProviderRegistry.Reset()
	defer ai.GlobalProviderRegistry.Reset()
	t.Setenv("OPENAI_TEST_KEY", "sk-test-stream-123")

	var sawAuth string
	var sawBody []byte
	server := newOpenAISSEServer(t, func(r *http.Request, body []byte) {
		sawAuth = r.Header.Get("Authorization")
		sawBody = body
	})
	defer server.Close()

	registerOpenAITestProvider(t, "test-openai-stream", server.URL, "OPENAI_TEST_KEY")

	ctx := streamTestContext(t)
	result, err := aiStreamCall(ctx, []eval.Value{
		&eval.StringValue{Value: "test-openai-stream"},
		&eval.StringValue{Value: "gpt-4o-mini"},
		&eval.StringValue{Value: `[{"role":"user","content":"Say hi"}]`},
	})
	if err != nil {
		t.Fatalf("aiStreamCall returned error: %v", err)
	}

	tagged, ok := result.(*eval.TaggedValue)
	if !ok || tagged.CtorName != "Ok" {
		t.Fatalf("expected Ok(StreamConn), got %+v", result)
	}
	conn, ok := tagged.Fields[0].(*eval.TaggedValue)
	if !ok || conn.CtorName != "StreamConn" {
		t.Fatalf("expected Ok(StreamConn(_)), got %+v", tagged.Fields[0])
	}

	if sawAuth != "Bearer sk-test-stream-123" {
		t.Errorf("upstream Authorization header = %q, want Bearer sk-test-stream-123", sawAuth)
	}
	bodyStr := string(sawBody)
	if !strings.Contains(bodyStr, `"stream":true`) {
		t.Errorf("body should contain stream:true, got: %s", bodyStr)
	}
	if !strings.Contains(bodyStr, `"model":"gpt-4o-mini"`) {
		t.Errorf("body should contain model, got: %s", bodyStr)
	}
}

// Verifies provider name unknown → ProviderNotFound (not panic, not 500).
func TestAIStreamCall_ProviderNotFound(t *testing.T) {
	ai.GlobalProviderRegistry.Reset()
	defer ai.GlobalProviderRegistry.Reset()

	ctx := streamTestContext(t)
	result, err := aiStreamCall(ctx, []eval.Value{
		&eval.StringValue{Value: "nonexistent-provider"},
		&eval.StringValue{Value: "any-model"},
		&eval.StringValue{Value: `[{"role":"user","content":"hi"}]`},
	})
	if err != nil {
		t.Fatalf("expected Result error, got Go error: %v", err)
	}
	tagged := result.(*eval.TaggedValue)
	if tagged.CtorName != "Err" {
		t.Fatalf("expected Err, got %s", tagged.CtorName)
	}
	inner := tagged.Fields[0].(*eval.TaggedValue)
	if inner.CtorName != "ProviderNotFound" {
		t.Errorf("expected ProviderNotFound variant, got %s", inner.CtorName)
	}
}

// Verifies streaming.enabled=false → ConnectionFailed with explanatory message.
func TestAIStreamCall_StreamingNotEnabled(t *testing.T) {
	ai.GlobalProviderRegistry.Reset()
	defer ai.GlobalProviderRegistry.Reset()

	spec := &pkg.AIProviderSpec{
		SchemaVersion: 1,
		Name:          "no-stream",
		Endpoint:      "http://localhost:1",
		RequestShape:  "openai_chat",
		ResponsePath:  "$.choices[0].message.content",
		Auth:          pkg.AIProviderAuth{Type: "none"},
		// Streaming.Enabled = false (default)
		Capabilities: pkg.AIProviderCapabilities{Streaming: false},
	}
	if err := ai.GlobalProviderRegistry.Register("no-stream", configdriven.New(spec), "test://no-stream"); err != nil {
		t.Fatal(err)
	}

	ctx := streamTestContext(t)
	result, _ := aiStreamCall(ctx, []eval.Value{
		&eval.StringValue{Value: "no-stream"},
		&eval.StringValue{Value: "x"},
		&eval.StringValue{Value: `[{"role":"user","content":"hi"}]`},
	})
	tagged := result.(*eval.TaggedValue)
	if tagged.CtorName != "Err" {
		t.Fatalf("expected Err, got %s", tagged.CtorName)
	}
	inner := tagged.Fields[0].(*eval.TaggedValue)
	if inner.CtorName != "ConnectionFailed" {
		t.Errorf("expected ConnectionFailed variant, got %s", inner.CtorName)
	}
	msg := inner.Fields[0].(*eval.StringValue).Value
	if !strings.Contains(msg, "does not enable streaming") {
		t.Errorf("error message should explain streaming is disabled, got: %q", msg)
	}
}

// Verifies missing AI cap → Go error from RequireCapWithBudget path.
// The aiStreamCall op itself doesn't check AI cap (the builtin does),
// but Stream cap is checked early in the op.
func TestAIStreamCall_MissingStreamCap(t *testing.T) {
	ai.GlobalProviderRegistry.Reset()
	defer ai.GlobalProviderRegistry.Reset()

	registerOpenAITestProvider(t, "needs-stream-cap", "http://localhost:1", "")

	ctx := effects.NewEffContext(nil)
	// Note: deliberately NOT setting ctx.Stream.

	_, err := aiStreamCall(ctx, []eval.Value{
		&eval.StringValue{Value: "needs-stream-cap"},
		&eval.StringValue{Value: "any"},
		&eval.StringValue{Value: `[{"role":"user","content":"hi"}]`},
	})
	if err == nil {
		t.Fatal("expected error for missing Stream context")
	}
	if !strings.Contains(err.Error(), "E_STREAM_NO_CONTEXT") {
		t.Errorf("expected E_STREAM_NO_CONTEXT error, got: %v", err)
	}
}

// Verifies env var unset for bearer auth → ConnectionFailed (not silent).
func TestAIStreamCall_AuthEnvUnset(t *testing.T) {
	ai.GlobalProviderRegistry.Reset()
	defer ai.GlobalProviderRegistry.Reset()
	t.Setenv("UNSET_KEY_FOR_TEST", "")

	registerOpenAITestProvider(t, "needs-key", "http://localhost:1", "UNSET_KEY_FOR_TEST")

	ctx := streamTestContext(t)
	result, _ := aiStreamCall(ctx, []eval.Value{
		&eval.StringValue{Value: "needs-key"},
		&eval.StringValue{Value: "any"},
		&eval.StringValue{Value: `[{"role":"user","content":"hi"}]`},
	})
	tagged := result.(*eval.TaggedValue)
	if tagged.CtorName != "Err" {
		t.Fatalf("expected Err, got %s", tagged.CtorName)
	}
	inner := tagged.Fields[0].(*eval.TaggedValue)
	msg := inner.Fields[0].(*eval.StringValue).Value
	if !strings.Contains(msg, "UNSET_KEY_FOR_TEST") {
		t.Errorf("error should name the unset env var, got: %q", msg)
	}
}

// Verifies the messages_json validator rejects empty/malformed input cleanly.
func TestAIStreamCall_InvalidMessagesJSON(t *testing.T) {
	ai.GlobalProviderRegistry.Reset()
	defer ai.GlobalProviderRegistry.Reset()

	registerOpenAITestProvider(t, "messages-test", "http://localhost:1", "")

	ctx := streamTestContext(t)
	cases := []struct {
		name string
		json string
	}{
		{"not_json", "not json at all"},
		{"empty_array", "[]"},
		{"no_user_message", `[{"role":"system","content":"hi"}]`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result, _ := aiStreamCall(ctx, []eval.Value{
				&eval.StringValue{Value: "messages-test"},
				&eval.StringValue{Value: "any"},
				&eval.StringValue{Value: tc.json},
			})
			tagged, ok := result.(*eval.TaggedValue)
			if !ok || tagged.CtorName != "Err" {
				t.Fatalf("expected Err for %s, got %+v", tc.name, result)
			}
		})
	}
}

// Verifies models allow-list enforcement: requesting a model not in the
// allow-list is rejected before any HTTP call.
func TestAIStreamCall_ModelsAllowlist(t *testing.T) {
	ai.GlobalProviderRegistry.Reset()
	defer ai.GlobalProviderRegistry.Reset()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("server unexpectedly hit — model should have been rejected by allow-list")
	}))
	defer server.Close()

	spec := &pkg.AIProviderSpec{
		SchemaVersion: 1,
		Name:          "allowlist-test",
		Endpoint:      server.URL,
		RequestShape:  "openai_chat",
		ResponsePath:  "$.choices[0].message.content",
		Auth:          pkg.AIProviderAuth{Type: "none"},
		Capabilities:  pkg.AIProviderCapabilities{Streaming: true},
		Streaming:     pkg.AIProviderStreaming{Enabled: true, DeltaPath: "$.choices[0].delta.content"},
		Models:        pkg.AIProviderModels{Allowed: []string{"approved-model"}},
	}
	if err := ai.GlobalProviderRegistry.Register("allowlist-test", configdriven.New(spec), "test://allowlist"); err != nil {
		t.Fatal(err)
	}

	ctx := streamTestContext(t)
	result, _ := aiStreamCall(ctx, []eval.Value{
		&eval.StringValue{Value: "allowlist-test"},
		&eval.StringValue{Value: "rejected-model"},
		&eval.StringValue{Value: `[{"role":"user","content":"hi"}]`},
	})
	tagged := result.(*eval.TaggedValue)
	if tagged.CtorName != "Err" {
		t.Fatalf("expected Err for disallowed model, got %s", tagged.CtorName)
	}
}

// BuildStreamRequest unit tests at the configdriven layer — exercises the
// request-construction path independently of the effects-layer dispatch.
func TestBuildStreamRequest_OpenAIShape(t *testing.T) {
	t.Setenv("BSR_KEY", "secret-bsr")

	spec := &pkg.AIProviderSpec{
		SchemaVersion: 1,
		Name:          "bsr-test",
		Endpoint:      "https://api.example.com/v1/chat/completions",
		RequestShape:  "openai_chat",
		ResponsePath:  "$.choices[0].message.content",
		Auth:          pkg.AIProviderAuth{Type: "bearer", Env: "BSR_KEY"},
		Capabilities:  pkg.AIProviderCapabilities{Streaming: true},
		Streaming:     pkg.AIProviderStreaming{Enabled: true, DeltaPath: "$.choices[0].delta.content"},
	}
	req, err := configdriven.BuildStreamRequest(spec, "gpt-4o", `[{"role":"user","content":"hello"}]`)
	if err != nil {
		t.Fatalf("BuildStreamRequest: %v", err.Message)
	}

	if req.URL != "https://api.example.com/v1/chat/completions" {
		t.Errorf("URL = %q", req.URL)
	}
	if !strings.Contains(req.Body, `"stream":true`) {
		t.Errorf("body missing stream:true, got: %s", req.Body)
	}
	if !strings.Contains(req.Body, `"model":"gpt-4o"`) {
		t.Errorf("body missing model, got: %s", req.Body)
	}
	// Auth header
	var foundAuth bool
	for _, h := range req.Headers {
		if h.Name == "Authorization" && h.Value == "Bearer secret-bsr" {
			foundAuth = true
		}
	}
	if !foundAuth {
		t.Errorf("Authorization header missing or wrong, got headers: %+v", req.Headers)
	}
}

// query-param auth should error in v1 (documented limitation).
func TestBuildStreamRequest_QueryParamAuthRejected(t *testing.T) {
	t.Setenv("QP_KEY", "any")

	spec := &pkg.AIProviderSpec{
		SchemaVersion: 1,
		Name:          "qp-test",
		Endpoint:      "https://example.com/v1",
		RequestShape:  "openai_chat",
		ResponsePath:  "$.choices[0].message.content",
		Auth:          pkg.AIProviderAuth{Type: "query-param", Name: "key", Env: "QP_KEY"},
		Capabilities:  pkg.AIProviderCapabilities{Streaming: true},
		Streaming:     pkg.AIProviderStreaming{Enabled: true, DeltaPath: "$.x"},
	}
	_, perr := configdriven.BuildStreamRequest(spec, "x", `[{"role":"user","content":"hi"}]`)
	if perr == nil {
		t.Fatal("expected error for query-param auth in streaming")
	}
	if !strings.Contains(perr.Message, "query-param") {
		t.Errorf("error should mention query-param limitation, got: %q", perr.Message)
	}
}

// Verifies BuildStreamRequest doesn't inject stream:true for Anthropic shape
// (Anthropic uses a different streaming convention via header, not body field).
func TestBuildStreamRequest_AnthropicShapeNoStreamFlag(t *testing.T) {
	t.Setenv("ANT_KEY", "sk-ant-bsr")

	spec := &pkg.AIProviderSpec{
		SchemaVersion: 1,
		Name:          "ant-test",
		Endpoint:      "https://api.anthropic.com/v1/messages",
		RequestShape:  "anthropic_messages",
		ResponsePath:  "$.content[0].text",
		Auth:          pkg.AIProviderAuth{Type: "x-api-key", Env: "ANT_KEY"},
		AuthHeaders:   map[string]string{"anthropic-version": "2023-06-01"},
		Capabilities:  pkg.AIProviderCapabilities{Streaming: true},
		Streaming:     pkg.AIProviderStreaming{Enabled: true, DeltaPath: "$.delta.text"},
	}
	req, perr := configdriven.BuildStreamRequest(spec, "claude-sonnet-4-5", `[{"role":"user","content":"hi"}]`)
	if perr != nil {
		t.Fatalf("BuildStreamRequest: %v", perr.Message)
	}
	if strings.Contains(req.Body, `"stream":true`) {
		t.Errorf("Anthropic shape body should NOT contain stream:true, got: %s", req.Body)
	}
	// Verify both auth shapes are present
	headerMap := map[string]string{}
	for _, h := range req.Headers {
		headerMap[h.Name] = h.Value
	}
	if headerMap["x-api-key"] != "sk-ant-bsr" {
		t.Errorf("x-api-key = %q", headerMap["x-api-key"])
	}
	if headerMap["anthropic-version"] != "2023-06-01" {
		t.Errorf("anthropic-version = %q", headerMap["anthropic-version"])
	}
}
