package ollama

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
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
