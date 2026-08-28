package ollama

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ai"
)

func TestNewClient(t *testing.T) {
	// Save and restore OLLAMA_HOST
	origHost := os.Getenv("OLLAMA_HOST")
	defer os.Setenv("OLLAMA_HOST", origHost)

	tests := []struct {
		name     string
		envHost  string
		opts     []ClientOption
		wantHost string
	}{
		{
			name:     "default endpoint",
			envHost:  "",
			opts:     nil,
			wantHost: "http://localhost:11434",
		},
		{
			name:     "env override",
			envHost:  "http://custom:8080",
			opts:     nil,
			wantHost: "http://custom:8080",
		},
		{
			name:     "option override",
			envHost:  "",
			opts:     []ClientOption{WithEndpoint("http://option:9090")},
			wantHost: "http://option:9090",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("OLLAMA_HOST", tt.envHost)

			client, err := NewClient(tt.opts...)
			if err != nil {
				t.Fatalf("NewClient() error = %v", err)
			}

			if client.endpoint != tt.wantHost {
				t.Errorf("endpoint = %v, want %v", client.endpoint, tt.wantHost)
			}
		})
	}
}

func TestClientName(t *testing.T) {
	client, err := NewClient()
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	if got := client.Name(); got != "ollama" {
		t.Errorf("Name() = %v, want ollama", got)
	}
}

func TestGuessProvider(t *testing.T) {
	tests := []struct {
		model string
		want  ai.ProviderType
	}{
		{"ollama:codellama", ai.ProviderOllama},
		{"ollama:llama3.2:3b", ai.ProviderOllama},
		{"OLLAMA:qwen2.5-coder", ai.ProviderOllama},
		{"gpt-5", ai.ProviderOpenAI},
		{"claude-sonnet-4-5", ai.ProviderAnthropic},
		{"gemini-2-5-flash", ai.ProviderGoogle},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			got := ai.GuessProvider(tt.model)
			if got != tt.want {
				t.Errorf("GuessProvider(%q) = %v, want %v", tt.model, got, tt.want)
			}
		})
	}
}

// TestCheckConnection is an integration test that requires Ollama running.
// Skip if Ollama is not available.
func TestCheckConnection(t *testing.T) {
	client, err := NewClient()
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	// This will fail if Ollama isn't running, which is expected in CI
	err = client.CheckConnection(t.Context())
	if err != nil {
		t.Skipf("Ollama not running (expected in CI): %v", err)
	}
}

// TestGenerate_RejectsRoutingPolicy ensures ollama refuses routing policies
// rather than silently ignoring them. Per CLAUDE.md no-silent-fallbacks.
func TestGenerate_RejectsRoutingPolicy(t *testing.T) {
	client, err := NewClient(WithEndpoint("http://localhost:1"))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	_, err = client.Generate(context.Background(), &ai.Request{
		Model:      "codellama:7b",
		UserPrompt: "hi",
		Routing: &ai.AIRoutingPolicy{
			Order:         []string{"local"},
			AllowFallback: true,
		},
	})
	if err == nil {
		t.Fatal("expected error rejecting routing policy, got nil")
	}
	if !errors.Is(err, ai.ErrRoutingNotSupported) {
		t.Errorf("error %v is not ai.ErrRoutingNotSupported", err)
	}
}

// TestClient_Generate_WireShape pins the tool-less path to /api/generate with
// the schema passed through as `format`. fb_2dbfd79dbe2d1d3c: a report claimed
// the schema never reached the wire; the capture showed it did, but on
// /api/chat — which on ollama 0.33.1 costs 15-26s per call vs ~0.6s on
// /api/generate for the identical body. This test fails if the endpoint or the
// format passthrough regresses.
func TestClient_Generate_WireShape(t *testing.T) {
	var path, body string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		path = r.URL.Path
		body = string(b)
		w.Header().Set("Content-Type", "application/x-ndjson")
		_, _ = w.Write([]byte(`{"model":"gemma4:e4b","created_at":"2026-08-28T10:00:00Z","response":"{\"category\":\"fyi\"}","done":true,"prompt_eval_count":33,"eval_count":6}` + "\n"))
	}))
	defer srv.Close()
	// NewClient lets OLLAMA_HOST override WithEndpoint, and other tests in this
	// package leave it set process-wide — pin it or the request leaks to the
	// real server on localhost:11434.
	t.Setenv("OLLAMA_HOST", srv.URL)
	c, err := NewClient(WithEndpoint(srv.URL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	schema := `{"type":"object","properties":{"category":{"type":"string","enum":["a","b"]}},"required":["category"]}`
	resp, err := c.Generate(context.Background(), &ai.Request{
		Model:          "ollama/gemma4:e4b",
		SystemPrompt:   "be brief",
		UserPrompt:     "classify this",
		ResponseFormat: "json",
		ResponseSchema: schema,
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if resp.Text != `{"category":"fyi"}` {
		t.Errorf("Text = %q, want schema-constrained reply", resp.Text)
	}
	if path != "/api/generate" {
		t.Errorf("endpoint = %q, want /api/generate (chat endpoint schedules 15-26s per call on the rig)", path)
	}
	if !strings.Contains(body, `"format":`+schema) {
		t.Errorf("schema must pass through verbatim as format; body: %s", body)
	}
	if strings.Contains(body, `"messages"`) {
		t.Errorf("Generate must not send a messages array; body: %s", body)
	}
	for _, want := range []string{`"prompt":"classify this"`, `"system":"be brief"`, `"model":"gemma4:e4b"`} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing %s; body: %s", want, body)
		}
	}
	if strings.Contains(body, "ollama/") {
		t.Errorf("routing prefix must be stripped from the model name; body: %s", body)
	}
	if resp.InputTokens != 33 || resp.OutputTokens != 6 {
		t.Errorf("token tally = (%d, %d), want (33, 6)", resp.InputTokens, resp.OutputTokens)
	}
}
