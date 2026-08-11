package ollama

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/ai"
)

// TestStep_ToolsAdvertisedAndParsed_Native verifies the legacy NATIVE /api/chat
// tool-calling path (opt-in via AILANG_OLLAMA_NATIVE_TOOLS=1): req.Tools are
// translated into Ollama's tool schema in the request, and tool_calls in the
// response are parsed back into ai.Response.ToolCalls with FinishReason="tool_calls".
// (Default tool path is now /v1 — see TestStep_ToolsViaOpenAICompat.)
func TestStep_ToolsAdvertisedAndParsed_Native(t *testing.T) {
	t.Setenv("AILANG_OLLAMA_NATIVE_TOOLS", "1") // force the native /api/chat tool path
	var capturedBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, 16384)
		n, _ := r.Body.Read(buf)
		capturedBody = string(buf[:n])
		w.Header().Set("Content-Type", "application/x-ndjson")
		_, _ = w.Write([]byte(`{"message":{"role":"assistant","content":"","tool_calls":[{"function":{"name":"write_file","arguments":{"path":"x.ail","content":"hi"}}}]},"done":true}` + "\n"))
	}))
	defer srv.Close()

	t.Setenv("OLLAMA_HOST", srv.URL)
	c, err := NewClient(WithEndpoint(srv.URL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	resp, stepErr := c.Step(context.Background(), &ai.Request{
		Model:    "qwen3.5",
		Messages: []ai.Message{{Role: "user", Content: "write x.ail"}},
		Tools: []ai.ToolSchema{{
			Name:        "write_file",
			Description: "write a file",
			Parameters:  `{"type":"object","properties":{"path":{"type":"string"},"content":{"type":"string"}},"required":["path","content"]}`,
		}},
	})
	if stepErr != nil {
		t.Fatalf("Step: %v", stepErr)
	}
	// Tools advertised in the request body.
	if !contains(capturedBody, `"tools"`) || !contains(capturedBody, "write_file") {
		t.Errorf("request missing advertised tools; got: %s", capturedBody)
	}
	// Tool call parsed from the response.
	if resp.FinishReason != "tool_calls" {
		t.Errorf("FinishReason = %q, want tool_calls", resp.FinishReason)
	}
	if len(resp.ToolCalls) != 1 || resp.ToolCalls[0].Name != "write_file" {
		t.Fatalf("ToolCalls = %+v, want one write_file call", resp.ToolCalls)
	}
	if !contains(resp.ToolCalls[0].Arguments, "x.ail") {
		t.Errorf("tool call args missing path; got: %s", resp.ToolCalls[0].Arguments)
	}
}

// TestStep_ToolsViaOpenAICompat verifies the DEFAULT tool-calling path
// (M-OLLAMA-V1-TOOLCALLING): when req.Tools is present and
// AILANG_OLLAMA_NATIVE_TOOLS is unset, Step delegates to Ollama's
// OpenAI-compatible /v1/chat/completions endpoint (the path pi/opencode use,
// where qwen3.x reliably emits tool_calls) rather than native /api/chat. The
// request must hit /v1/chat/completions with tools advertised, the model name
// must be stripped of the ollama: prefix, and OpenAI-format tool_calls in the
// response must be parsed back into ai.Response.ToolCalls.
func TestStep_ToolsViaOpenAICompat(t *testing.T) {
	// Ensure the native opt-out is OFF so we exercise the default /v1 path.
	t.Setenv("AILANG_OLLAMA_NATIVE_TOOLS", "")
	var capturedPath, capturedBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		buf := make([]byte, 16384)
		n, _ := r.Body.Read(buf)
		capturedBody = string(buf[:n])
		w.Header().Set("Content-Type", "application/json")
		// OpenAI-format (non-streaming) chat completion carrying a tool call.
		_, _ = w.Write([]byte(`{"id":"chatcmpl-1","object":"chat.completion","model":"qwen3.6","choices":[{"index":0,"message":{"role":"assistant","content":"","tool_calls":[{"id":"call_1","type":"function","function":{"name":"write_file","arguments":"{\"path\":\"x.ail\",\"content\":\"hi\"}"}}]},"finish_reason":"tool_calls"}],"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`))
	}))
	defer srv.Close()

	t.Setenv("OLLAMA_HOST", srv.URL)
	c, err := NewClient(WithEndpoint(srv.URL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	resp, stepErr := c.Step(context.Background(), &ai.Request{
		Model:    "ollama:qwen3.6",
		Messages: []ai.Message{{Role: "user", Content: "write x.ail"}},
		Tools: []ai.ToolSchema{{
			Name:        "write_file",
			Description: "write a file",
			Parameters:  `{"type":"object","properties":{"path":{"type":"string"},"content":{"type":"string"}},"required":["path","content"]}`,
		}},
	})
	if stepErr != nil {
		t.Fatalf("Step: %v", stepErr)
	}
	// Delegated to the OpenAI-compat endpoint, not native /api/chat.
	if capturedPath != "/v1/chat/completions" {
		t.Errorf("request path = %q, want /v1/chat/completions (native path used?)", capturedPath)
	}
	// Tools advertised in the OpenAI request body.
	if !contains(capturedBody, `"tools"`) || !contains(capturedBody, "write_file") {
		t.Errorf("request missing advertised tools; got: %s", capturedBody)
	}
	// Model prefix stripped for the API (qwen3.6, not ollama:qwen3.6).
	if !contains(capturedBody, `"model":"qwen3.6"`) || contains(capturedBody, "ollama:qwen3.6") {
		t.Errorf("model not de-prefixed for /v1; got: %s", capturedBody)
	}
	// OpenAI-format tool_calls parsed back out.
	if resp.FinishReason != "tool_calls" {
		t.Errorf("FinishReason = %q, want tool_calls", resp.FinishReason)
	}
	if len(resp.ToolCalls) != 1 || resp.ToolCalls[0].Name != "write_file" {
		t.Fatalf("ToolCalls = %+v, want one write_file call", resp.ToolCalls)
	}
	if !contains(resp.ToolCalls[0].Arguments, "x.ail") {
		t.Errorf("tool call args missing path; got: %s", resp.ToolCalls[0].Arguments)
	}
}

// TestStep_ToolsViaOpenAICompat_TimesOut verifies that a STALLED /v1 server
// (model reload under GPU contention stalling an open connection) makes Step
// fail fast instead of hanging forever. The /v1 delegation path is non-streaming,
// so without a client timeout io.ReadAll blocks indefinitely — this regression
// hung a live motoko run for ~2h. AILANG_OLLAMA_HTTP_TIMEOUT_SEC bounds it.
func TestStep_ToolsViaOpenAICompat_TimesOut(t *testing.T) {
	t.Setenv("AILANG_OLLAMA_NATIVE_TOOLS", "")
	t.Setenv("AILANG_OLLAMA_HTTP_TIMEOUT_SEC", "1") // tiny cap for the test

	releaseHang := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if f, ok := w.(http.Flusher); ok {
			f.Flush() // send headers, then stall the body forever (simulates a wedged stream)
		}
		<-releaseHang
	}))
	// Defers run LIFO: close(releaseHang) first (unblocks the handler), THEN
	// srv.Close() (which waits for the now-unblocked handler goroutine). The
	// reverse order would deadlock Close() against the still-stalled handler.
	defer srv.Close()
	defer close(releaseHang)

	t.Setenv("OLLAMA_HOST", srv.URL)
	c, err := NewClient(WithEndpoint(srv.URL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	done := make(chan error, 1)
	go func() {
		_, stepErr := c.Step(context.Background(), &ai.Request{
			Model:    "ollama:qwen3.6",
			Messages: []ai.Message{{Role: "user", Content: "hi"}},
			Tools:    []ai.ToolSchema{{Name: "write_file", Description: "w", Parameters: `{"type":"object"}`}},
		})
		done <- stepErr
	}()

	select {
	case stepErr := <-done:
		if stepErr == nil {
			t.Fatal("expected a timeout error from the stalled /v1 server, got nil")
		}
		// Good: bounded failure rather than an infinite hang.
	case <-time.After(10 * time.Second):
		t.Fatal("Step did not return within 10s — the 1s HTTP timeout is not being applied (infinite-hang regression)")
	}
}

// TestOllamaV1Timeout verifies the env-configurable timeout helper.
func TestOllamaV1Timeout(t *testing.T) {
	t.Setenv("AILANG_OLLAMA_HTTP_TIMEOUT_SEC", "")
	if got := ollamaV1Timeout(); got != defaultOllamaV1TimeoutSec*time.Second {
		t.Errorf("default = %v, want %v", got, defaultOllamaV1TimeoutSec*time.Second)
	}
	t.Setenv("AILANG_OLLAMA_HTTP_TIMEOUT_SEC", "45")
	if got := ollamaV1Timeout(); got != 45*time.Second {
		t.Errorf("override = %v, want 45s", got)
	}
	t.Setenv("AILANG_OLLAMA_HTTP_TIMEOUT_SEC", "0")
	if got := ollamaV1Timeout(); got != 0 {
		t.Errorf("disabled = %v, want 0", got)
	}
}

// TestResolveOllamaTemperature verifies the precedence: req>0 wins, then the
// env, then unset (0). (M-OLLAMA-TEMPERATURE-KNOB)
func TestResolveOllamaTemperature(t *testing.T) {
	t.Setenv("AILANG_OLLAMA_TEMPERATURE", "")
	if got := resolveOllamaTemperature(0); got != 0 {
		t.Errorf("unset env + req 0 = %v, want 0", got)
	}
	if got := resolveOllamaTemperature(0.7); got != 0.7 {
		t.Errorf("req 0.7 = %v, want 0.7 (req wins)", got)
	}
	t.Setenv("AILANG_OLLAMA_TEMPERATURE", "0.2")
	if got := resolveOllamaTemperature(0); got != 0.2 {
		t.Errorf("env 0.2 + req 0 = %v, want 0.2", got)
	}
	if got := resolveOllamaTemperature(0.9); got != 0.9 {
		t.Errorf("req 0.9 must override env = %v, want 0.9", got)
	}
	t.Setenv("AILANG_OLLAMA_TEMPERATURE", "garbage")
	if got := resolveOllamaTemperature(0); got != 0 {
		t.Errorf("unparseable env = %v, want 0 (ignored)", got)
	}
}

// TestStep_ToolsViaOpenAICompat_TemperatureEnv verifies the env knob reaches the
// /v1 request body, and that it's absent by default (today's behaviour).
func TestStep_ToolsViaOpenAICompat_TemperatureEnv(t *testing.T) {
	run := func(envVal string) string {
		t.Setenv("AILANG_OLLAMA_NATIVE_TOOLS", "")
		t.Setenv("AILANG_OLLAMA_TEMPERATURE", envVal)
		var body string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			buf := make([]byte, 16384)
			n, _ := r.Body.Read(buf)
			body = string(buf[:n])
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"x","object":"chat.completion","choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}]}`))
		}))
		defer srv.Close()
		t.Setenv("OLLAMA_HOST", srv.URL)
		c, _ := NewClient(WithEndpoint(srv.URL))
		_, _ = c.Step(context.Background(), &ai.Request{
			Model:    "ollama:qwen3.6",
			Messages: []ai.Message{{Role: "user", Content: "hi"}},
			Tools:    []ai.ToolSchema{{Name: "w", Description: "w", Parameters: `{"type":"object"}`}},
		})
		return body
	}
	if b := run("0.2"); !contains(b, `"temperature":0.2`) {
		t.Errorf("env 0.2 not in /v1 body; got: %s", b)
	}
	if b := run(""); contains(b, `"temperature"`) {
		t.Errorf("temperature must be absent by default; got: %s", b)
	}
}

// TestStep_NativePath_TimesOut verifies that a STALLED native /api/chat stream
// (ollama idle, producing no data and no error) makes Step fail fast via the
// context deadline instead of hanging forever — the native path has no client
// timeout of its own (this is the 7h-motoko-hang class the /v1 timeout missed).
func TestStep_NativePath_TimesOut(t *testing.T) {
	t.Setenv("AILANG_OLLAMA_NATIVE_TOOLS", "1") // force native /api/chat
	t.Setenv("AILANG_OLLAMA_HTTP_TIMEOUT_SEC", "1")

	releaseHang := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-ndjson")
		if f, ok := w.(http.Flusher); ok {
			f.Flush() // headers only, then stall the stream forever
		}
		<-releaseHang
	}))
	defer srv.Close()
	defer close(releaseHang)

	t.Setenv("OLLAMA_HOST", srv.URL)
	c, err := NewClient(WithEndpoint(srv.URL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	done := make(chan error, 1)
	go func() {
		_, stepErr := c.Step(context.Background(), &ai.Request{
			Model:    "qwen3.6",
			Messages: []ai.Message{{Role: "user", Content: "hi"}},
			Tools:    []ai.ToolSchema{{Name: "w", Description: "w", Parameters: `{"type":"object"}`}},
		})
		done <- stepErr
	}()

	select {
	case stepErr := <-done:
		if stepErr == nil {
			t.Fatal("expected a deadline error from the stalled native stream, got nil")
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Step did not return within 10s — the native-path context deadline is not applied (hang regression)")
	}
}

// TestStep_NoMessages_DelegatesToGenerate verifies the legacy
// single-shot path: when req.Messages is empty, Step routes to
// Generate via the same ollama chat endpoint.
func TestStep_NoMessages_DelegatesToGenerate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Ollama Chat API streams NDJSON. Emit a single response chunk
		// with done=true so the streaming loop terminates immediately.
		w.Header().Set("Content-Type", "application/x-ndjson")
		_, _ = w.Write([]byte(`{"model":"llama3.2","created_at":"2026-05-05T13:00:00Z","message":{"role":"assistant","content":"hi from Generate"},"done":true}` + "\n"))
	}))
	defer srv.Close()

	t.Setenv("OLLAMA_HOST", srv.URL)
	c, err := NewClient(WithEndpoint(srv.URL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	resp, stepErr := c.Step(context.Background(), &ai.Request{
		Model:        "llama3.2",
		SystemPrompt: "be brief",
		UserPrompt:   "say hi",
	})
	if stepErr != nil {
		t.Fatalf("Step: %v", stepErr)
	}
	if resp.Text != "hi from Generate" {
		t.Errorf("Text = %q, want \"hi from Generate\"", resp.Text)
	}
	// Generate doesn't set FinishReason — that's expected legacy behaviour
	// preserved by the delegation.
}

// TestStep_MultiTurnNoTools_TranslatesMessages verifies that req.Messages
// (multi-turn conversation, no tools) gets sent through Ollama's chat
// endpoint with the messages list intact.
func TestStep_MultiTurnNoTools_TranslatesMessages(t *testing.T) {
	var capturedBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, 4096)
		n, _ := r.Body.Read(buf)
		capturedBody = string(buf[:n])
		w.Header().Set("Content-Type", "application/x-ndjson")
		_, _ = w.Write([]byte(`{"model":"llama3.2","created_at":"2026-05-05T13:00:00Z","message":{"role":"assistant","content":"yes"},"done":true}` + "\n"))
	}))
	defer srv.Close()

	t.Setenv("OLLAMA_HOST", srv.URL)
	c, err := NewClient(WithEndpoint(srv.URL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	resp, stepErr := c.Step(context.Background(), &ai.Request{
		Model: "llama3.2",
		Messages: []ai.Message{
			{Role: "user", Content: "what is 2+2?"},
			{Role: "assistant", Content: "4"},
			{Role: "user", Content: "are you sure?"},
		},
	})
	if stepErr != nil {
		t.Fatalf("Step: %v", stepErr)
	}
	if resp.Text != "yes" {
		t.Errorf("Text = %q, want \"yes\"", resp.Text)
	}
	if resp.FinishReason != "stop" {
		t.Errorf("FinishReason = %q, want \"stop\"", resp.FinishReason)
	}
	// All three messages should be in the request body, not flattened
	// to a single user prompt.
	for _, want := range []string{"what is 2+2?", "are you sure?"} {
		if !contains(capturedBody, want) {
			t.Errorf("captured body missing %q; got: %s", want, capturedBody)
		}
	}
}

// TestStep_SystemPromptPrepended verifies that req.SystemPrompt is
// prepended as a system-role message when req.Messages doesn't already
// contain a system role.
func TestStep_SystemPromptPrepended(t *testing.T) {
	var capturedBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, 4096)
		n, _ := r.Body.Read(buf)
		capturedBody = string(buf[:n])
		w.Header().Set("Content-Type", "application/x-ndjson")
		_, _ = w.Write([]byte(`{"message":{"role":"assistant","content":"ok"},"done":true}` + "\n"))
	}))
	defer srv.Close()

	t.Setenv("OLLAMA_HOST", srv.URL)
	c, err := NewClient(WithEndpoint(srv.URL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	_, stepErr := c.Step(context.Background(), &ai.Request{
		Model:        "llama3.2",
		SystemPrompt: "you are helpful",
		Messages: []ai.Message{
			{Role: "user", Content: "hi"},
		},
	})
	if stepErr != nil {
		t.Fatalf("Step: %v", stepErr)
	}
	if !contains(capturedBody, "you are helpful") {
		t.Errorf("captured body missing system prompt; got: %s", capturedBody)
	}
	if !contains(capturedBody, `"role":"system"`) {
		t.Errorf("captured body missing system role; got: %s", capturedBody)
	}
}

// TestStep_SystemPromptNotDuplicated verifies that req.SystemPrompt is
// IGNORED when req.Messages already contains a system role (req.Messages
// wins to avoid double system prompts).
func TestStep_SystemPromptNotDuplicated(t *testing.T) {
	var capturedBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, 4096)
		n, _ := r.Body.Read(buf)
		capturedBody = string(buf[:n])
		w.Header().Set("Content-Type", "application/x-ndjson")
		_, _ = w.Write([]byte(`{"message":{"role":"assistant","content":"ok"},"done":true}` + "\n"))
	}))
	defer srv.Close()

	t.Setenv("OLLAMA_HOST", srv.URL)
	c, err := NewClient(WithEndpoint(srv.URL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	_, stepErr := c.Step(context.Background(), &ai.Request{
		Model:        "llama3.2",
		SystemPrompt: "FROM_SYSTEMPROMPT_FIELD",
		Messages: []ai.Message{
			{Role: "system", Content: "FROM_MESSAGES_ARRAY"},
			{Role: "user", Content: "hi"},
		},
	})
	if stepErr != nil {
		t.Fatalf("Step: %v", stepErr)
	}
	// Messages-array system wins; SystemPrompt field is dropped.
	if !contains(capturedBody, "FROM_MESSAGES_ARRAY") {
		t.Errorf("captured body missing messages-array system; got: %s", capturedBody)
	}
	if contains(capturedBody, "FROM_SYSTEMPROMPT_FIELD") {
		t.Errorf("captured body should NOT contain SystemPrompt field (Messages wins); got: %s", capturedBody)
	}
}

// TestStep_ToolRoleMessageNative verifies that a Role="tool" message is passed
// through to Ollama's NATIVE tool role with its tool_call_id intact (Ollama's
// /api/chat supports tool-result messages now), rather than being flattened to
// a user message.
func TestStep_ToolRoleMessageNative(t *testing.T) {
	var capturedBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, 8192)
		n, _ := r.Body.Read(buf)
		capturedBody = string(buf[:n])
		w.Header().Set("Content-Type", "application/x-ndjson")
		_, _ = w.Write([]byte(`{"message":{"role":"assistant","content":"ok"},"done":true}` + "\n"))
	}))
	defer srv.Close()

	t.Setenv("OLLAMA_HOST", srv.URL)
	c, err := NewClient(WithEndpoint(srv.URL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	_, stepErr := c.Step(context.Background(), &ai.Request{
		Model: "llama3.2",
		Messages: []ai.Message{
			{Role: "user", Content: "read foo.txt"},
			{Role: "assistant", Content: "reading..."},
			{Role: "tool", ToolCallID: "call_1", Content: "hello world"},
		},
	})
	if stepErr != nil {
		t.Fatalf("Step: %v", stepErr)
	}
	// Tool-result content goes through with the native tool role + tool_call_id.
	if !contains(capturedBody, "hello world") {
		t.Errorf("captured body missing tool result content; got: %s", capturedBody)
	}
	if !contains(capturedBody, `"role":"tool"`) {
		t.Errorf("captured body should use native role=tool; got: %s", capturedBody)
	}
	if !contains(capturedBody, `"tool_call_id":"call_1"`) {
		t.Errorf("captured body missing tool_call_id; got: %s", capturedBody)
	}
}

// TestStep_ChatErrorClassified verifies that errors from the underlying
// chat call get wrapped into *ai.AIError via ClassifyError, not the
// legacy *ai.ProviderError.
func TestStep_ChatErrorClassified(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return a 500 with a plain text body. The Ollama client surfaces
		// this as a Go error; ClassifyError should produce a typed AIError.
		w.WriteHeader(500)
		_, _ = w.Write([]byte(`upstream blew up`))
	}))
	defer srv.Close()

	t.Setenv("OLLAMA_HOST", srv.URL)
	c, err := NewClient(WithEndpoint(srv.URL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	_, stepErr := c.Step(context.Background(), &ai.Request{
		Model: "llama3.2",
		Messages: []ai.Message{
			{Role: "user", Content: "hi"},
		},
	})
	if stepErr == nil {
		t.Fatal("expected error from 500 response")
	}
	var aiErr *ai.AIError
	if !errors.As(stepErr, &aiErr) {
		t.Fatalf("expected *AIError, got %T %v", stepErr, stepErr)
	}
	// ClassifyError defaults to CodeInternal for unrecognized error
	// strings; ConnectionFailed if the body parses as a connection issue.
	// Either is acceptable — the contract is "not a generic error".
	if aiErr.Code == "" {
		t.Error("AIError.Code is empty")
	}
}

// --- M3 response-parity tests (M-OLLAMA-V1-STREAMING-IDLE-TIMEOUT) -----------
//
// These pin REFUTATION #2: the streamed /v1 path must be RESPONSE-equivalent to
// the buffered one, not merely the same shape. The SSE parser sets no Reasoning
// and runs no Hermes tool-call recovery; stepV1Stream restores both. Each test
// runs the SAME logical response once buffered (flag off) and once as SSE
// (flag on) and compares. The fake-/v1 helpers (newFakeV1, streamEnv, toolReq,
// contentChunk, toolFragChunk, finishChunk, doneChunk, sse*) live in
// streambranch_test.go.

// reasoningChunk emits an SSE chunk carrying a delta.reasoning fragment — the
// field the SSE parser surfaces as ai.StreamThinkingDelta and the exact place
// qwen3 thinking models put a Hermes <tool_call> block.
func reasoningChunk(text string) string {
	return sseData(map[string]any{
		"id": "chatcmpl-1", "object": "chat.completion.chunk", "model": "qwen3.6",
		"choices": []any{map[string]any{"index": 0, "delta": map[string]any{"reasoning": text}}},
	})
}

// fullToolChunk emits a complete native tool_call (id + name + args) in ONE
// chunk — the committed-fixture shape (AC-M3.3 case a).
func fullToolChunk(id, name, args string) string {
	return sseData(map[string]any{
		"id": "chatcmpl-1", "object": "chat.completion.chunk", "model": "qwen3.6",
		"choices": []any{map[string]any{"index": 0, "delta": map[string]any{
			"tool_calls": []any{map[string]any{
				"index": 0, "id": id, "type": "function",
				"function": map[string]any{"name": name, "arguments": args},
			}},
		}}},
	})
}

// nativeTC builds one non-streaming (buffered) tool_call. arguments is the
// OpenAI wire shape: a JSON STRING, so json.Marshal escapes it correctly.
func nativeTC(id, name, args string) map[string]any {
	return map[string]any{
		"id": id, "type": "function",
		"function": map[string]any{"name": name, "arguments": args},
	}
}

// bufferedV1Body renders a non-streaming /v1 chat completion body. reasoning is
// omitted when empty; toolCalls is omitted when nil.
func bufferedV1Body(content, reasoning, finish string, toolCalls []map[string]any) string {
	msg := map[string]any{"role": "assistant", "content": content}
	if reasoning != "" {
		msg["reasoning"] = reasoning
	}
	if toolCalls != nil {
		msg["tool_calls"] = toolCalls
	}
	body := map[string]any{
		"id": "chatcmpl-1", "object": "chat.completion", "model": "qwen3.6",
		"choices": []any{map[string]any{"index": 0, "message": msg, "finish_reason": finish}},
		"usage":   map[string]any{"prompt_tokens": 1, "completion_tokens": 1, "total_tokens": 2},
	}
	b, err := json.Marshal(body)
	if err != nil {
		panic(err) // test-only: the map above is always marshalable
	}
	return string(b)
}

// runOllamaBuffered drives the flag-OFF buffered /v1 path against a server that
// returns jsonBody, and returns the parsed response.
func runOllamaBuffered(t *testing.T, jsonBody string) *ai.Response {
	t.Helper()
	f := newFakeV1(t, func(_ *fakeV1, w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, jsonBody)
	})
	t.Setenv("AILANG_OLLAMA_NATIVE_TOOLS", "")
	t.Setenv("AILANG_OLLAMA_V1_STREAM", "") // flag OFF -> buffered path
	t.Setenv("OLLAMA_HOST", f.srv.URL)
	c := newTestClient(t, f.srv.URL)
	resp, err := c.Step(context.Background(), toolReq())
	if err != nil {
		t.Fatalf("buffered Step: %v", err)
	}
	return resp
}

// runOllamaStreamed drives the flag-ON streaming /v1 path against a server that
// emits chunks then [DONE], and returns the parsed response.
func runOllamaStreamed(t *testing.T, chunks ...string) *ai.Response {
	t.Helper()
	f := newFakeV1(t, func(_ *fakeV1, w http.ResponseWriter, _ *http.Request) {
		sseHeaders(w)
		for _, ch := range chunks {
			if !sseWrite(w, ch) {
				return
			}
		}
		sseWrite(w, doneChunk)
	})
	streamEnv(t, f.srv.URL)
	c := newTestClient(t, f.srv.URL)
	got := stepWithin(t, c, toolReq(), 5*time.Second)
	if got.err != nil {
		t.Fatalf("streamed Step: %v", got.err)
	}
	return got.resp
}

// TestStreamParity_HermesRecoveryFromReasoning is AC-M3.1 (mutation R8). A
// Hermes <tool_call> block split across THREE delta.reasoning chunks, with zero
// native tool_calls, must be recovered into exactly one write_file call on the
// streamed path — and the identical logical response on the buffered path must
// recover it too (the anti-vacuity control).
func TestStreamParity_HermesRecoveryFromReasoning(t *testing.T) {
	assertOneWriteFile := func(t *testing.T, resp *ai.Response) {
		t.Helper()
		if len(resp.ToolCalls) != 1 {
			t.Fatalf("ToolCalls = %+v, want exactly 1", resp.ToolCalls)
		}
		tc := resp.ToolCalls[0]
		if tc.Name != "write_file" {
			t.Errorf("Name = %q, want write_file", tc.Name)
		}
		var args struct {
			Path string `json:"path"`
		}
		if err := json.Unmarshal([]byte(tc.Arguments), &args); err != nil {
			t.Fatalf("arguments %q do not parse as JSON: %v", tc.Arguments, err)
		}
		if args.Path != "a.ail" {
			t.Errorf("arguments.path = %q, want a.ail", args.Path)
		}
		if resp.FinishReason != "tool_calls" {
			t.Errorf("FinishReason = %q, want tool_calls", resp.FinishReason)
		}
	}

	t.Run("streamed_flag_on_recovers_across_reasoning_chunks", func(t *testing.T) {
		resp := runOllamaStreamed(t,
			reasoningChunk(`<tool_call>{"name":"wr`),
			reasoningChunk(`ite_file","arguments":{"path":"a.ail"}}`),
			reasoningChunk(`</tool_call>`),
			finishChunk("stop"),
		)
		assertOneWriteFile(t, resp)
	})

	t.Run("buffered_flag_off_control_also_recovers", func(t *testing.T) {
		body := bufferedV1Body("",
			`<tool_call>{"name":"write_file","arguments":{"path":"a.ail"}}</tool_call>`,
			"stop", nil)
		assertOneWriteFile(t, runOllamaBuffered(t, body))
	})
}

// TestStreamParity_ReasoningByteIdentical is AC-M3.2 (mutation R9). The same
// reasoning served buffered vs streamed (as three fragments) must produce a
// non-empty, byte-identical Response.Reasoning.
func TestStreamParity_ReasoningByteIdentical(t *testing.T) {
	const reasoning = "Step 1: read the spec.\nStep 2: write the file.\nStep 3: run it."
	buffered := runOllamaBuffered(t, bufferedV1Body("done", reasoning, "stop", nil))
	streamed := runOllamaStreamed(t,
		reasoningChunk("Step 1: read the spec.\n"),
		reasoningChunk("Step 2: write the file.\n"),
		reasoningChunk("Step 3: run it."),
		contentChunk("done"),
		finishChunk("stop"),
	)
	if buffered.Reasoning == "" {
		t.Fatal("buffered Reasoning is empty — the control cannot prove parity")
	}
	if streamed.Reasoning == "" {
		t.Error("streamed Reasoning is empty — R9 accumulation did not fire")
	}
	if streamed.Reasoning != buffered.Reasoning {
		t.Errorf("Reasoning mismatch:\n streamed = %q\n buffered = %q", streamed.Reasoning, buffered.Reasoning)
	}
}

// TestStreamParity_StreamedEquivalentToBuffered is AC-M3.3 (doc S6,
// strengthened; mutations R8/R9/R10). Three cases, each served once buffered
// and once as SSE, asserting Text, ToolCalls (order/IDs/assembled args),
// FinishReason AND Reasoning are identical.
func TestStreamParity_StreamedEquivalentToBuffered(t *testing.T) {
	cases := []struct {
		name         string
		bufferedJSON string
		streamChunks []string
	}{
		{
			name: "native_single_toolcall_chunk",
			bufferedJSON: bufferedV1Body("", "reasoned about it.", "tool_calls",
				[]map[string]any{nativeTC("call_1", "write_file", `{"path":"a.ail"}`)}),
			streamChunks: []string{
				reasoningChunk("reasoned about it."),
				fullToolChunk("call_1", "write_file", `{"path":"a.ail"}`),
				finishChunk("tool_calls"),
			},
		},
		{
			name: "toolcall_fragmented_across_3_chunks",
			bufferedJSON: bufferedV1Body("", "fragmented think", "tool_calls",
				[]map[string]any{nativeTC("call_1", "write_file", `{"path":"a.ail"}`)}),
			streamChunks: []string{
				reasoningChunk("fragmented think"),
				toolFragChunk("write_file", ""),
				toolFragChunk("", `{"pa`),
				toolFragChunk("", `th":"a`),
				toolFragChunk("", `.ail"}`),
				finishChunk("tool_calls"),
			},
		},
		{
			name: "hermes_in_reasoning_zero_native",
			bufferedJSON: bufferedV1Body("ok",
				`plan: <tool_call>{"name":"write_file","arguments":{"path":"a.ail"}}</tool_call>`,
				"stop", nil),
			streamChunks: []string{
				contentChunk("ok"),
				reasoningChunk(`plan: <tool_call>{"name":"write_file","arguments":{"path":"a.ail"}}</tool_call>`),
				finishChunk("stop"),
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			buffered := runOllamaBuffered(t, tc.bufferedJSON)
			streamed := runOllamaStreamed(t, tc.streamChunks...)

			if streamed.Text != buffered.Text {
				t.Errorf("Text: streamed=%q buffered=%q", streamed.Text, buffered.Text)
			}
			if streamed.FinishReason != buffered.FinishReason {
				t.Errorf("FinishReason: streamed=%q buffered=%q", streamed.FinishReason, buffered.FinishReason)
			}
			if streamed.Reasoning != buffered.Reasoning {
				t.Errorf("Reasoning: streamed=%q buffered=%q", streamed.Reasoning, buffered.Reasoning)
			}
			if !reflect.DeepEqual(streamed.ToolCalls, buffered.ToolCalls) {
				t.Errorf("ToolCalls mismatch:\n streamed=%+v\n buffered=%+v", streamed.ToolCalls, buffered.ToolCalls)
			}
			// Anti-vacuity: every case carries exactly one tool call on BOTH
			// paths, so an all-empty pass cannot masquerade as parity.
			if len(buffered.ToolCalls) != 1 {
				t.Fatalf("buffered ToolCalls = %+v, want exactly 1 (control)", buffered.ToolCalls)
			}
			if len(streamed.ToolCalls) != 1 {
				t.Fatalf("streamed ToolCalls = %+v, want exactly 1", streamed.ToolCalls)
			}
		})
	}
}

func contains(s, sub string) bool {
	if len(sub) == 0 {
		return true
	}
	if len(s) < len(sub) {
		return false
	}
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
