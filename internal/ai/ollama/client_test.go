package ollama

import (
	"context"
	"errors"
	"os"
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
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

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
