package anthropic

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sunholo-data/ailang/internal/ai"
)

func TestClient_Generate_Success(t *testing.T) {
	// Mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != "POST" {
			t.Errorf("Method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/messages" {
			t.Errorf("Path = %s, want /messages", r.URL.Path)
		}
		if r.Header.Get("x-api-key") != "test-key" {
			t.Errorf("x-api-key = %s, want test-key", r.Header.Get("x-api-key"))
		}
		if r.Header.Get("anthropic-version") != "2023-06-01" {
			t.Errorf("anthropic-version = %s, want 2023-06-01", r.Header.Get("anthropic-version"))
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Content-Type = %s, want application/json", r.Header.Get("Content-Type"))
		}

		// Verify request body
		var reqBody messagesRequest
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Fatalf("Failed to decode request body: %v", err)
		}
		if reqBody.Model != "claude-sonnet-4-5" {
			t.Errorf("Model = %s, want claude-sonnet-4-5", reqBody.Model)
		}
		if reqBody.MaxTokens != 2048 {
			t.Errorf("MaxTokens = %d, want 2048", reqBody.MaxTokens)
		}
		if reqBody.System != "You are helpful." {
			t.Errorf("System = %s, want 'You are helpful.'", reqBody.System)
		}
		if len(reqBody.Messages) != 1 || reqBody.Messages[0].Content != "Hello" {
			t.Errorf("Messages = %v, want [{user Hello}]", reqBody.Messages)
		}

		// Return response
		resp := messagesResponse{
			ID:    "msg_123",
			Type:  "message",
			Role:  "assistant",
			Model: "claude-sonnet-4-5-20251001",
			Content: []contentBlock{
				{Type: "text", Text: "Hello, how can I help you today?"},
			},
			StopReason: "end_turn",
			Usage: anthropicUsage{
				InputTokens:  15,
				OutputTokens: 10,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Create client
	client := NewClient("test-key", WithBaseURL(server.URL))

	// Make request
	resp, err := client.Generate(context.Background(), &ai.Request{
		Model:        "claude-sonnet-4-5",
		SystemPrompt: "You are helpful.",
		UserPrompt:   "Hello",
		MaxTokens:    2048,
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	// Verify response
	if resp.Text != "Hello, how can I help you today?" {
		t.Errorf("Text = %q, want %q", resp.Text, "Hello, how can I help you today?")
	}
	if resp.InputTokens != 15 {
		t.Errorf("InputTokens = %d, want 15", resp.InputTokens)
	}
	if resp.OutputTokens != 10 {
		t.Errorf("OutputTokens = %d, want 10", resp.OutputTokens)
	}
	if resp.TotalTokens != 25 {
		t.Errorf("TotalTokens = %d, want 25", resp.TotalTokens)
	}
	if resp.Model != "claude-sonnet-4-5-20251001" {
		t.Errorf("Model = %s, want claude-sonnet-4-5-20251001", resp.Model)
	}
}

func TestClient_Generate_Error(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		wantMsg    string
	}{
		{
			name:       "rate limit",
			statusCode: 429,
			body:       `{"type":"error","error":{"type":"rate_limit_error","message":"Rate limit exceeded"}}`,
			wantMsg:    "Rate limit exceeded",
		},
		{
			name:       "invalid api key",
			statusCode: 401,
			body:       `{"type":"error","error":{"type":"authentication_error","message":"Invalid API key"}}`,
			wantMsg:    "Invalid API key",
		},
		{
			name:       "server error",
			statusCode: 500,
			body:       `Internal Server Error`,
			wantMsg:    "Internal Server Error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.body))
			}))
			defer server.Close()

			client := NewClient("test-key", WithBaseURL(server.URL))
			_, err := client.Generate(context.Background(), &ai.Request{
				Model:      "claude-sonnet-4-5",
				UserPrompt: "test",
			})

			if err == nil {
				t.Fatal("Generate() expected error, got nil")
			}

			providerErr, ok := err.(*ai.ProviderError)
			if !ok {
				t.Fatalf("err type = %T, want *ai.ProviderError", err)
			}

			if providerErr.StatusCode != tt.statusCode {
				t.Errorf("StatusCode = %d, want %d", providerErr.StatusCode, tt.statusCode)
			}
			if providerErr.Message != tt.wantMsg {
				t.Errorf("Message = %q, want %q", providerErr.Message, tt.wantMsg)
			}
		})
	}
}

func TestClient_Generate_EmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := messagesResponse{
			ID:      "msg_123",
			Type:    "message",
			Content: []contentBlock{}, // Empty content
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	_, err := client.Generate(context.Background(), &ai.Request{
		Model:      "claude-sonnet-4-5",
		UserPrompt: "test",
	})

	if err == nil {
		t.Fatal("Generate() expected error for empty response, got nil")
	}

	providerErr, ok := err.(*ai.ProviderError)
	if !ok {
		t.Fatalf("err type = %T, want *ai.ProviderError", err)
	}
	if providerErr.Message != "empty response from Claude" {
		t.Errorf("Message = %q, want %q", providerErr.Message, "empty response from Claude")
	}
}

func TestClient_Generate_DefaultMaxTokens(t *testing.T) {
	var receivedMaxTokens int

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody messagesRequest
		json.NewDecoder(r.Body).Decode(&reqBody)
		receivedMaxTokens = reqBody.MaxTokens

		resp := messagesResponse{
			Content: []contentBlock{{Type: "text", Text: "ok"}},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	client.Generate(context.Background(), &ai.Request{
		Model:      "claude-sonnet-4-5",
		UserPrompt: "test",
		// MaxTokens not set
	})

	if receivedMaxTokens != 4096 {
		t.Errorf("default MaxTokens = %d, want 4096", receivedMaxTokens)
	}
}

func TestClient_Generate_MultipleContentBlocks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := messagesResponse{
			Content: []contentBlock{
				{Type: "text", Text: "Hello "},
				{Type: "text", Text: "World!"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	resp, err := client.Generate(context.Background(), &ai.Request{
		Model:      "claude-sonnet-4-5",
		UserPrompt: "test",
	})

	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if resp.Text != "Hello World!" {
		t.Errorf("Text = %q, want %q", resp.Text, "Hello World!")
	}
}

func TestClient_Name(t *testing.T) {
	client := NewClient("test-key")
	if client.Name() != "anthropic" {
		t.Errorf("Name() = %q, want %q", client.Name(), "anthropic")
	}
}

func TestClient_NewHandler(t *testing.T) {
	client := NewClient("test-key")
	handler := client.NewHandler("claude-sonnet-4-5",
		ai.WithSystemPrompt("Be helpful"),
		ai.WithMaxTokens(1000),
	)

	if handler.Provider() != client {
		t.Error("Handler.Provider() did not return expected client")
	}
	if handler.Model() != "claude-sonnet-4-5" {
		t.Errorf("Handler.Model() = %q, want %q", handler.Model(), "claude-sonnet-4-5")
	}
}

func TestClient_WithOptions(t *testing.T) {
	customHTTP := &http.Client{}
	client := NewClient("test-key",
		WithBaseURL("https://custom.api.com"),
		WithHTTPClient(customHTTP),
		WithAPIVersion("2024-01-01"),
	)

	if client.baseURL != "https://custom.api.com" {
		t.Errorf("baseURL = %q, want %q", client.baseURL, "https://custom.api.com")
	}
	if client.httpClient != customHTTP {
		t.Error("httpClient not set correctly")
	}
	if client.apiVersion != "2024-01-01" {
		t.Errorf("apiVersion = %q, want %q", client.apiVersion, "2024-01-01")
	}
}

// TestGenerate_RejectsRoutingPolicy ensures anthropic refuses routing policies
// rather than silently ignoring them. Per CLAUDE.md no-silent-fallbacks.
func TestGenerate_RejectsRoutingPolicy(t *testing.T) {
	client := NewClient("test-key")
	_, err := client.Generate(context.Background(), &ai.Request{
		Model:      "claude-sonnet-4-5",
		UserPrompt: "hi",
		Routing: &ai.AIRoutingPolicy{
			Order:         []string{"anthropic"},
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

// TestGenerate_RejectsPriceCapOnlyPolicy ensures a max-price cost cap is refused
// rather than silently ignored. Anthropic cannot enforce a per-token price cap,
// so a policy carrying one (even with no Order/AllowFallback) must fail loud —
// a cost guard the caller believes is active but isn't is worse than none.
func TestGenerate_RejectsPriceCapOnlyPolicy(t *testing.T) {
	client := NewClient("test-key")
	_, err := client.Generate(context.Background(), &ai.Request{
		Model:      "claude-sonnet-4-5",
		UserPrompt: "hi",
		Routing:    &ai.AIRoutingPolicy{MaxPricePerMTok: "0.005"},
	})
	if err == nil {
		t.Fatal("expected error rejecting price-cap policy, got nil")
	}
	if !errors.Is(err, ai.ErrRoutingNotSupported) {
		t.Errorf("error %v is not ai.ErrRoutingNotSupported", err)
	}
}
