package configdriven

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ai"
	"github.com/sunholo-data/ailang/internal/pkg"
)

// openaiChatSpec returns a minimal valid AIProviderSpec with bearer auth
// pointed at a caller-supplied URL — typical fixture for httptest tests.
func openaiChatSpec(url string) *pkg.AIProviderSpec {
	return &pkg.AIProviderSpec{
		SchemaVersion: 1,
		Name:          "test-openai",
		Endpoint:      url,
		RequestShape:  "openai_chat",
		ResponsePath:  "$.choices[0].message.content",
		ErrorPath:     "$.error.message",
		Auth:          pkg.AIProviderAuth{Type: "bearer", Env: "TEST_PROVIDER_KEY"},
		Cost: pkg.AIProviderCost{
			InputPer1MUSD:  1.0,
			OutputPer1MUSD: 2.0,
		},
		Capabilities: pkg.AIProviderCapabilities{JSONMode: true},
	}
}

// captureRequest is a helper that records the incoming HTTP request for
// post-call assertions in addition to returning a canned response.
type captureRequest struct {
	headers http.Header
	body    []byte
	url     string
}

func TestGenerate_OpenAIChat_Success(t *testing.T) {
	t.Setenv("TEST_PROVIDER_KEY", "sk-test-secret")

	var captured captureRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		captured = captureRequest{headers: r.Header.Clone(), body: body, url: r.URL.String()}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
            "choices": [{"message": {"role": "assistant", "content": "Hello!"}}],
            "usage": {"prompt_tokens": 12, "completion_tokens": 5, "total_tokens": 17}
        }`))
	}))
	defer server.Close()

	p := New(openaiChatSpec(server.URL))
	resp, err := p.Generate(context.Background(), &ai.Request{
		Model:        "gpt-4o-mini",
		SystemPrompt: "You are concise.",
		UserPrompt:   "Hi",
		MaxTokens:    100,
	})
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	if resp.Text != "Hello!" {
		t.Errorf("Text = %q, want Hello!", resp.Text)
	}
	if resp.InputTokens != 12 || resp.OutputTokens != 5 || resp.TotalTokens != 17 {
		t.Errorf("token counts = (%d,%d,%d), want (12,5,17)",
			resp.InputTokens, resp.OutputTokens, resp.TotalTokens)
	}
	// Cost: 12 * 1/1M + 5 * 2/1M = 0.000022
	if resp.CostUSD != "0.000022" {
		t.Errorf("CostUSD = %q, want 0.000022", resp.CostUSD)
	}

	// Verify outgoing request shape:
	if got := captured.headers.Get("Authorization"); got != "Bearer sk-test-secret" {
		t.Errorf("Authorization = %q, want Bearer sk-test-secret", got)
	}
	if got := captured.headers.Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", got)
	}
	var sentBody openaiChatRequest
	if err := json.Unmarshal(captured.body, &sentBody); err != nil {
		t.Fatalf("could not parse sent body: %v", err)
	}
	if sentBody.Model != "gpt-4o-mini" {
		t.Errorf("sent model = %q, want gpt-4o-mini", sentBody.Model)
	}
	if len(sentBody.Messages) != 2 {
		t.Errorf("messages len = %d, want 2 (system + user)", len(sentBody.Messages))
	}
	if sentBody.Messages[0].Role != "system" || sentBody.Messages[1].Role != "user" {
		t.Errorf("message roles = %q,%q", sentBody.Messages[0].Role, sentBody.Messages[1].Role)
	}
	if sentBody.MaxTokens != 100 {
		t.Errorf("max_tokens = %d, want 100", sentBody.MaxTokens)
	}
}

func TestGenerate_AuthFailed_401(t *testing.T) {
	t.Setenv("TEST_PROVIDER_KEY", "wrong-key")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error": {"message": "Invalid API key", "type": "invalid_api_key"}}`))
	}))
	defer server.Close()

	p := New(openaiChatSpec(server.URL))
	_, err := p.Generate(context.Background(), &ai.Request{Model: "x", UserPrompt: "Hi"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var perr *ai.ProviderError
	if !errors.As(err, &perr) {
		t.Fatalf("error is not *ProviderError: %T", err)
	}
	if perr.StatusCode != 401 {
		t.Errorf("StatusCode = %d, want 401", perr.StatusCode)
	}
	if !strings.Contains(perr.Message, "Invalid API key") {
		t.Errorf("Message = %q, want it to contain Invalid API key", perr.Message)
	}
}

func TestGenerate_5xxRetryable(t *testing.T) {
	t.Setenv("TEST_PROVIDER_KEY", "sk-test")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"error": {"message": "upstream gateway timeout"}}`))
	}))
	defer server.Close()

	p := New(openaiChatSpec(server.URL))
	_, err := p.Generate(context.Background(), &ai.Request{Model: "x", UserPrompt: "Hi"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var perr *ai.ProviderError
	if !errors.As(err, &perr) || perr.StatusCode != 502 {
		t.Errorf("expected ProviderError with status 502, got %v", err)
	}
}

func TestGenerate_RoutingPolicyRejected(t *testing.T) {
	t.Setenv("TEST_PROVIDER_KEY", "sk-test")
	// Server should never be hit; provider rejects before HTTP.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("server unexpectedly called — routing policy should have been rejected before HTTP")
	}))
	defer server.Close()

	p := New(openaiChatSpec(server.URL))
	req := &ai.Request{
		Model:      "x",
		UserPrompt: "Hi",
		Routing: &ai.AIRoutingPolicy{
			Order:         []string{"anthropic", "openai"},
			AllowFallback: true,
		},
	}
	_, err := p.Generate(context.Background(), req)
	if err == nil {
		t.Fatal("expected ErrRoutingNotSupported, got nil")
	}
	if !errors.Is(err, ai.ErrRoutingNotSupported) {
		t.Errorf("expected ErrRoutingNotSupported, got %v", err)
	}
}

func TestGenerate_ImageRequestRejectedWithoutVision(t *testing.T) {
	t.Setenv("TEST_PROVIDER_KEY", "sk-test")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("server unexpectedly called — image request should have been rejected before HTTP")
	}))
	defer server.Close()

	p := New(openaiChatSpec(server.URL))
	_, err := p.Generate(context.Background(), &ai.Request{
		Model:              "x",
		UserPrompt:         "Draw a cat",
		ResponseModalities: []string{"IMAGE"},
	})
	if err == nil {
		t.Fatal("expected error for image request without vision capability")
	}
	if !strings.Contains(err.Error(), "image generation not supported") {
		t.Errorf("error should mention image generation, got %v", err)
	}
}

func TestGenerate_JSONModeRejectedWhenUnsupported(t *testing.T) {
	t.Setenv("TEST_PROVIDER_KEY", "sk-test")
	spec := openaiChatSpec("http://unused")
	spec.Capabilities.JSONMode = false
	spec.Capabilities.StructuredOutputs = false

	p := New(spec)
	_, err := p.Generate(context.Background(), &ai.Request{
		Model:          "x",
		UserPrompt:     "Hi",
		ResponseFormat: "json",
	})
	if err == nil {
		t.Fatal("expected error for JSON mode without capability")
	}
	if !strings.Contains(err.Error(), "JSON mode not supported") {
		t.Errorf("error should mention JSON mode, got %v", err)
	}
}

func TestGenerate_ModelsAllowlistEnforced(t *testing.T) {
	t.Setenv("TEST_PROVIDER_KEY", "sk-test")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("server unexpectedly called — model should have been rejected by allowlist")
	}))
	defer server.Close()

	spec := openaiChatSpec(server.URL)
	spec.Models.Allowed = []string{"llama-3.1-70b", "llama-3.1-8b"}

	p := New(spec)
	_, err := p.Generate(context.Background(), &ai.Request{Model: "qwen-2.5-72b", UserPrompt: "Hi"})
	if err == nil {
		t.Fatal("expected error for disallowed model")
	}
	if !strings.Contains(err.Error(), "is not in the allowed list") {
		t.Errorf("error should mention allowlist, got %v", err)
	}
}

func TestGenerate_EnvVarMissing(t *testing.T) {
	// Deliberately do NOT set TEST_PROVIDER_KEY.
	t.Setenv("TEST_PROVIDER_KEY", "") // Force-clear in case of inherited env

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("server unexpectedly called — should have failed before HTTP on missing env var")
	}))
	defer server.Close()

	p := New(openaiChatSpec(server.URL))
	_, err := p.Generate(context.Background(), &ai.Request{Model: "x", UserPrompt: "Hi"})
	if err == nil {
		t.Fatal("expected error for missing env var")
	}
	if !strings.Contains(err.Error(), "TEST_PROVIDER_KEY") || !strings.Contains(err.Error(), "unset") {
		t.Errorf("error should clearly mention the missing env var, got %v", err)
	}
}

func TestGenerate_AnthropicMessages(t *testing.T) {
	t.Setenv("ANTHROPIC_TEST_KEY", "sk-ant-test")

	var captured captureRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		captured = captureRequest{headers: r.Header.Clone(), body: body}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
            "content": [{"type": "text", "text": "Hi from Claude"}],
            "usage": {"input_tokens": 8, "output_tokens": 4}
        }`))
	}))
	defer server.Close()

	spec := &pkg.AIProviderSpec{
		SchemaVersion: 1,
		Name:          "test-anthropic",
		Endpoint:      server.URL,
		RequestShape:  "anthropic_messages",
		ResponsePath:  "$.content[0].text",
		Auth:          pkg.AIProviderAuth{Type: "x-api-key", Env: "ANTHROPIC_TEST_KEY"},
		AuthHeaders:   map[string]string{"anthropic-version": "2023-06-01"},
	}
	p := New(spec)
	resp, err := p.Generate(context.Background(), &ai.Request{
		Model:        "claude-sonnet-4-5",
		SystemPrompt: "Be brief.",
		UserPrompt:   "Hello",
		MaxTokens:    50,
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if resp.Text != "Hi from Claude" {
		t.Errorf("Text = %q", resp.Text)
	}
	if resp.InputTokens != 8 || resp.OutputTokens != 4 {
		t.Errorf("tokens = (%d,%d), want (8,4)", resp.InputTokens, resp.OutputTokens)
	}
	if got := captured.headers.Get("x-api-key"); got != "sk-ant-test" {
		t.Errorf("x-api-key = %q", got)
	}
	if got := captured.headers.Get("anthropic-version"); got != "2023-06-01" {
		t.Errorf("anthropic-version = %q", got)
	}
	// Verify body shape: messages[0].content[0].text + system
	var body anthropicMessagesRequest
	if err := json.Unmarshal(captured.body, &body); err != nil {
		t.Fatalf("body parse: %v", err)
	}
	if body.System != "Be brief." {
		t.Errorf("system = %q", body.System)
	}
	if body.MaxTokens != 50 {
		t.Errorf("max_tokens = %d", body.MaxTokens)
	}
	if len(body.Messages) != 1 || len(body.Messages[0].Content) != 1 {
		t.Fatalf("messages structure unexpected: %+v", body)
	}
	if body.Messages[0].Content[0].Text != "Hello" {
		t.Errorf("user content = %q", body.Messages[0].Content[0].Text)
	}
}

func TestGenerate_SimpleCompletion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"content": "Pong"}`))
	}))
	defer server.Close()

	spec := &pkg.AIProviderSpec{
		SchemaVersion: 1,
		Name:          "test-llamacpp",
		Endpoint:      server.URL,
		RequestShape:  "simple_completion",
		ResponsePath:  "$.content",
		Auth:          pkg.AIProviderAuth{Type: "none"},
	}
	p := New(spec)
	resp, err := p.Generate(context.Background(), &ai.Request{Model: "llama", UserPrompt: "Ping"})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if resp.Text != "Pong" {
		t.Errorf("Text = %q", resp.Text)
	}
}

func TestGenerate_AuthHeadersInterpolation(t *testing.T) {
	t.Setenv("FOO_TOKEN", "abc123")
	t.Setenv("FOO_ORG", "org-42")

	var captured captureRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = captureRequest{headers: r.Header.Clone()}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"ok"}}]}`))
	}))
	defer server.Close()

	spec := &pkg.AIProviderSpec{
		SchemaVersion: 1,
		Name:          "test-custom",
		Endpoint:      server.URL,
		RequestShape:  "openai_chat",
		ResponsePath:  "$.choices[0].message.content",
		AuthHeaders: map[string]string{
			"X-Token": "Bearer ${FOO_TOKEN}",
			"X-Org":   "${FOO_ORG}",
		},
	}
	p := New(spec)
	_, err := p.Generate(context.Background(), &ai.Request{Model: "x", UserPrompt: "Hi"})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if got := captured.headers.Get("X-Token"); got != "Bearer abc123" {
		t.Errorf("X-Token = %q", got)
	}
	if got := captured.headers.Get("X-Org"); got != "org-42" {
		t.Errorf("X-Org = %q", got)
	}
}

func TestGenerate_QueryParamAuth(t *testing.T) {
	t.Setenv("GEMINI_TEST_KEY", "k-test-123")

	var captured captureRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = captureRequest{url: r.URL.String()}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"ok"}}]}`))
	}))
	defer server.Close()

	spec := &pkg.AIProviderSpec{
		SchemaVersion: 1,
		Name:          "test-querykey",
		Endpoint:      server.URL,
		RequestShape:  "openai_chat",
		ResponsePath:  "$.choices[0].message.content",
		Auth:          pkg.AIProviderAuth{Type: "query-param", Name: "key", Env: "GEMINI_TEST_KEY"},
	}
	p := New(spec)
	_, err := p.Generate(context.Background(), &ai.Request{Model: "x", UserPrompt: "Hi"})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if !strings.Contains(captured.url, "key=k-test-123") {
		t.Errorf("URL did not carry key query param: %q", captured.url)
	}
}

func TestGenerate_MalformedJSONResponse(t *testing.T) {
	t.Setenv("TEST_PROVIDER_KEY", "sk-test")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("this is not JSON"))
	}))
	defer server.Close()

	p := New(openaiChatSpec(server.URL))
	_, err := p.Generate(context.Background(), &ai.Request{Model: "x", UserPrompt: "Hi"})
	if err == nil {
		t.Fatal("expected error for non-JSON response")
	}
	if !strings.Contains(err.Error(), "not valid JSON") {
		t.Errorf("error should mention invalid JSON, got %v", err)
	}
}

func TestGenerate_ResponsePathMisses(t *testing.T) {
	t.Setenv("TEST_PROVIDER_KEY", "sk-test")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"unexpected": {"shape": "without choices field"}}`))
	}))
	defer server.Close()

	p := New(openaiChatSpec(server.URL))
	_, err := p.Generate(context.Background(), &ai.Request{Model: "x", UserPrompt: "Hi"})
	if err == nil {
		t.Fatal("expected error for response_path miss")
	}
	if !strings.Contains(err.Error(), "response_path") {
		t.Errorf("error should mention response_path, got %v", err)
	}
}

func TestGenerate_ErrorPathMissesFallsBackToBody(t *testing.T) {
	t.Setenv("TEST_PROVIDER_KEY", "sk-test")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		// Response shape doesn't match the configured error_path
		_, _ = w.Write([]byte(`{"detail": "permission denied"}`))
	}))
	defer server.Close()

	p := New(openaiChatSpec(server.URL))
	_, err := p.Generate(context.Background(), &ai.Request{Model: "x", UserPrompt: "Hi"})
	if err == nil {
		t.Fatal("expected error")
	}
	var perr *ai.ProviderError
	if !errors.As(err, &perr) || perr.StatusCode != 403 {
		t.Fatalf("expected 403 ProviderError, got %v", err)
	}
	// Should fall back to truncated raw body (no error_path match)
	if !strings.Contains(perr.Message, "permission denied") {
		t.Errorf("Message should fall back to body content, got %q", perr.Message)
	}
}

func TestComputeCost(t *testing.T) {
	cases := []struct {
		name      string
		cost      pkg.AIProviderCost
		inTokens  int
		outTokens int
		want      string
	}{
		{
			name: "no cost data",
			cost: pkg.AIProviderCost{},
			want: "",
		},
		{
			name:      "per-token only",
			cost:      pkg.AIProviderCost{InputPer1MUSD: 3.0, OutputPer1MUSD: 15.0},
			inTokens:  1_000_000,
			outTokens: 1_000_000,
			want:      "18.000000",
		},
		{
			name: "per-call only",
			cost: pkg.AIProviderCost{PerCallUSD: 0.001},
			want: "0.001000",
		},
		{
			name:      "both per-call and per-token",
			cost:      pkg.AIProviderCost{InputPer1MUSD: 1.0, OutputPer1MUSD: 2.0, PerCallUSD: 0.01},
			inTokens:  1_000_000,
			outTokens: 500_000,
			want:      "2.010000", // 1.0 + 1.0 + 0.01
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := computeCost(tc.cost, tc.inTokens, tc.outTokens)
			if got != tc.want {
				t.Errorf("computeCost = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestExtractPath_HappyPaths(t *testing.T) {
	parsed := mustParseJSON(t, `{
        "choices": [
            {"message": {"role": "assistant", "content": "Hello"}}
        ],
        "usage": {"prompt_tokens": 10, "total_tokens": 15}
    }`)

	cases := []struct {
		path string
		want any
	}{
		{"$", parsed},
		{"$.choices[0].message.content", "Hello"},
		{"$.choices[0].message.role", "assistant"},
		{"$.usage.prompt_tokens", float64(10)},
	}
	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			got, err := extractPath(parsed, tc.path)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			// Compare via JSON representation since Go interface{}
			// equality is too strict for nested structures.
			a, _ := json.Marshal(got)
			b, _ := json.Marshal(tc.want)
			if string(a) != string(b) {
				t.Errorf("got %s, want %s", a, b)
			}
		})
	}
}

func TestExtractPath_Errors(t *testing.T) {
	parsed := mustParseJSON(t, `{"choices": [{"message": {"content": "x"}}]}`)
	cases := []struct {
		path    string
		errPart string
	}{
		{"", "empty path"},
		{"choices[0]", "must start with $"},
		{"$.missing", "missing field"},
		{"$.choices[5].message.content", "out of bounds"},
		{"$.choices[abc]", "non-integer index"},
		{"$.choices[0", "unclosed ["},
		{"$..foo", "empty field"},
	}
	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			_, err := extractPath(parsed, tc.path)
			if err == nil {
				t.Fatalf("expected error for %q", tc.path)
			}
			if !strings.Contains(err.Error(), tc.errPart) {
				t.Errorf("error %q does not contain %q", err.Error(), tc.errPart)
			}
		})
	}
}

func TestEnvPattern_Guard(t *testing.T) {
	if !guardEnvPattern() {
		t.Errorf("env pattern guard tripped — likely drift between internal/pkg and configdriven")
	}
}

func mustParseJSON(t *testing.T, s string) any {
	t.Helper()
	var v any
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return v
}
