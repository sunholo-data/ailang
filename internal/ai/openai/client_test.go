package openai

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sunholo-data/ailang/internal/ai"
)

func TestClient_Generate_ChatCompletions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != "POST" {
			t.Errorf("Method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/chat/completions" {
			t.Errorf("Path = %s, want /chat/completions", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("Authorization = %s, want Bearer test-key", r.Header.Get("Authorization"))
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Content-Type = %s, want application/json", r.Header.Get("Content-Type"))
		}

		// Verify request body
		var reqBody chatRequest
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Fatalf("Failed to decode request body: %v", err)
		}
		if reqBody.Model != "gpt-4-turbo" {
			t.Errorf("Model = %s, want gpt-4-turbo", reqBody.Model)
		}
		// GPT-4 uses MaxTokens, not MaxCompletionTokens
		if reqBody.MaxTokens != 2048 {
			t.Errorf("MaxTokens = %d, want 2048", reqBody.MaxTokens)
		}
		if len(reqBody.Messages) != 2 {
			t.Errorf("len(Messages) = %d, want 2", len(reqBody.Messages))
		}
		if reqBody.Messages[0].Role != "system" {
			t.Errorf("Messages[0].Role = %s, want system", reqBody.Messages[0].Role)
		}
		if reqBody.Messages[1].Role != "user" {
			t.Errorf("Messages[1].Role = %s, want user", reqBody.Messages[1].Role)
		}

		// Return response
		resp := chatResponse{
			ID:     "chatcmpl-123",
			Object: "chat.completion",
			Model:  "gpt-4-turbo-20240901",
			Choices: []chatChoice{
				{
					Index: 0,
					Message: chatMessage{
						Role:    "assistant",
						Content: "Hello! How can I help you today?",
					},
					FinishReason: "stop",
				},
			},
			Usage: chatUsage{
				PromptTokens:     20,
				CompletionTokens: 15,
				TotalTokens:      35,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Use GPT-4 (legacy model) to test Chat Completions path
	client := NewClient("test-key", WithBaseURL(server.URL))

	resp, err := client.Generate(context.Background(), &ai.Request{
		Model:        "gpt-4-turbo",
		SystemPrompt: "You are helpful.",
		UserPrompt:   "Hello",
		MaxTokens:    2048,
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if resp.Text != "Hello! How can I help you today?" {
		t.Errorf("Text = %q, want %q", resp.Text, "Hello! How can I help you today?")
	}
	if resp.InputTokens != 20 {
		t.Errorf("InputTokens = %d, want 20", resp.InputTokens)
	}
	if resp.OutputTokens != 15 {
		t.Errorf("OutputTokens = %d, want 15", resp.OutputTokens)
	}
	if resp.TotalTokens != 35 {
		t.Errorf("TotalTokens = %d, want 35", resp.TotalTokens)
	}
}

func TestClient_Generate_ChatCompletions_WithReasoningTokens(t *testing.T) {
	// Test Chat Completions reasoning tokens with explicit API type override
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify max_completion_tokens is used for GPT-5.1
		var reqBody chatRequest
		json.NewDecoder(r.Body).Decode(&reqBody)
		if reqBody.MaxCompletionTokens != 4096 {
			t.Errorf("MaxCompletionTokens = %d, want 4096", reqBody.MaxCompletionTokens)
		}
		if reqBody.MaxTokens != 0 {
			t.Errorf("MaxTokens = %d, want 0 (should use MaxCompletionTokens)", reqBody.MaxTokens)
		}

		resp := chatResponse{
			Model: "gpt-5.1",
			Choices: []chatChoice{
				{Message: chatMessage{Content: "Thinking..."}},
			},
			Usage: chatUsage{
				PromptTokens:     10,
				CompletionTokens: 100,
				TotalTokens:      110,
				CompletionTokensDetails: struct {
					ReasoningTokens int `json:"reasoning_tokens"`
				}{ReasoningTokens: 80},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Force Chat Completions API to test that path
	client := NewClient("test-key", WithBaseURL(server.URL), WithAPIType(APIChatCompletions))

	resp, err := client.Generate(context.Background(), &ai.Request{
		Model:      "gpt-5.1",
		UserPrompt: "test",
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	// OutputTokens should be completion - reasoning
	if resp.OutputTokens != 20 {
		t.Errorf("OutputTokens = %d, want 20 (100 - 80 reasoning)", resp.OutputTokens)
	}
	if resp.ReasonTokens != 80 {
		t.Errorf("ReasonTokens = %d, want 80", resp.ReasonTokens)
	}
}

func TestClient_Generate_ChatCompletions_Error(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		wantMsg    string
	}{
		{
			name:       "rate limit",
			statusCode: 429,
			body:       `{"error":{"message":"Rate limit exceeded","type":"rate_limit_error"}}`,
			wantMsg:    "Rate limit exceeded",
		},
		{
			name:       "invalid api key",
			statusCode: 401,
			body:       `{"error":{"message":"Invalid API key","type":"authentication_error"}}`,
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

			// Use GPT-4 (legacy model) to test Chat Completions error handling
			client := NewClient("test-key", WithBaseURL(server.URL))
			_, err := client.Generate(context.Background(), &ai.Request{
				Model:      "gpt-4",
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

func TestClient_Generate_ChatCompletions_NoChoices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := chatResponse{
			Choices: []chatChoice{},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Use GPT-4 (legacy model) to test Chat Completions path
	client := NewClient("test-key", WithBaseURL(server.URL))
	_, err := client.Generate(context.Background(), &ai.Request{
		Model:      "gpt-4",
		UserPrompt: "test",
	})

	if err == nil {
		t.Fatal("Generate() expected error for empty choices, got nil")
	}

	providerErr, ok := err.(*ai.ProviderError)
	if !ok {
		t.Fatalf("err type = %T, want *ai.ProviderError", err)
	}
	if providerErr.Message != "no choices in response" {
		t.Errorf("Message = %q, want %q", providerErr.Message, "no choices in response")
	}
}

func TestClient_DetectAPIType(t *testing.T) {
	client := NewClient("test-key")

	tests := []struct {
		model string
		want  APIType
	}{
		// Modern models use Responses API
		{"gpt-5", APIResponses},
		{"gpt-5-mini", APIResponses},
		{"gpt-5.1", APIResponses},
		{"gpt-5.2", APIResponses},
		{"gpt-5.1-codex-max", APIResponses},
		{"codex-mini", APIResponses},
		{"o1-preview", APIResponses},
		{"o3-mini", APIResponses},
		// Legacy models use Chat Completions
		{"gpt-4", APIChatCompletions},
		{"gpt-4-turbo", APIChatCompletions},
		{"gpt-3.5-turbo", APIChatCompletions},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			got := client.detectAPIType(tt.model)
			if got != tt.want {
				t.Errorf("detectAPIType(%q) = %q, want %q", tt.model, got, tt.want)
			}
		})
	}
}

func TestClient_WithAPITypeOverride(t *testing.T) {
	client := NewClient("test-key", WithAPIType(APIResponses))

	// Even though gpt-5 normally uses Chat, it should use Responses due to override
	if client.detectAPIType("gpt-5") != APIResponses {
		t.Errorf("WithAPIType override not working")
	}
}

func TestClient_Name(t *testing.T) {
	client := NewClient("test-key")
	if client.Name() != "openai" {
		t.Errorf("Name() = %q, want %q", client.Name(), "openai")
	}
}

func TestClient_NewHandler(t *testing.T) {
	client := NewClient("test-key")
	handler := client.NewHandler("gpt-5",
		ai.WithSystemPrompt("Be helpful"),
		ai.WithMaxTokens(1000),
	)

	if handler.Provider() != client {
		t.Error("Handler.Provider() did not return expected client")
	}
	if handler.Model() != "gpt-5" {
		t.Errorf("Handler.Model() = %q, want %q", handler.Model(), "gpt-5")
	}
}

func TestClient_WithOptions(t *testing.T) {
	customHTTP := &http.Client{}
	client := NewClient("test-key",
		WithBaseURL("https://custom.api.com"),
		WithHTTPClient(customHTTP),
	)

	if client.baseURL != "https://custom.api.com" {
		t.Errorf("baseURL = %q, want %q", client.baseURL, "https://custom.api.com")
	}
	if client.httpClient != customHTTP {
		t.Error("httpClient not set correctly")
	}
}

func TestUsesMaxCompletionTokens(t *testing.T) {
	tests := []struct {
		model string
		want  bool
	}{
		{"gpt-5", true},      // All GPT-5 models use max_completion_tokens
		{"gpt-5-mini", true}, // All GPT-5 models use max_completion_tokens
		{"gpt-5.1", true},
		{"gpt-5.2", true},
		{"gpt-5-1", true},
		{"gpt-5-2", true},
		{"o1-preview", true},
		{"o3-mini", true},
		{"gpt-4", false}, // GPT-4 uses max_tokens
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			got := usesMaxCompletionTokens(tt.model)
			if got != tt.want {
				t.Errorf("usesMaxCompletionTokens(%q) = %v, want %v", tt.model, got, tt.want)
			}
		})
	}
}

func TestClient_Generate_ChatCompletions_WithSeed(t *testing.T) {
	var receivedSeed *int64

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody chatRequest
		json.NewDecoder(r.Body).Decode(&reqBody)
		receivedSeed = reqBody.Seed

		resp := chatResponse{
			Choices: []chatChoice{{Message: chatMessage{Content: "ok"}}},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Use GPT-4 (legacy model) to test Chat Completions path
	client := NewClient("test-key", WithBaseURL(server.URL))
	client.Generate(context.Background(), &ai.Request{
		Model:      "gpt-4",
		UserPrompt: "test",
		Options:    map[string]any{"seed": int64(42)},
	})

	if receivedSeed == nil || *receivedSeed != 42 {
		t.Errorf("Seed not passed correctly, got %v", receivedSeed)
	}
}

func TestClient_Generate_ChatCompletions_DefaultMaxTokens(t *testing.T) {
	var receivedMaxTokens int

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody chatRequest
		json.NewDecoder(r.Body).Decode(&reqBody)
		// GPT-4 uses MaxTokens
		receivedMaxTokens = reqBody.MaxTokens

		resp := chatResponse{
			Choices: []chatChoice{{Message: chatMessage{Content: "ok"}}},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Use GPT-4 (legacy model) to test Chat Completions path
	client := NewClient("test-key", WithBaseURL(server.URL))
	client.Generate(context.Background(), &ai.Request{
		Model:      "gpt-4",
		UserPrompt: "test",
		// MaxTokens not set - should default to 4096
	})

	if receivedMaxTokens != 4096 {
		t.Errorf("default MaxTokens = %d, want 4096", receivedMaxTokens)
	}
}

// ============================================================================
// Responses API Tests
// ============================================================================

func TestClient_Generate_ResponsesAPI(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request goes to /responses endpoint
		if r.URL.Path != "/responses" {
			t.Errorf("Path = %s, want /responses", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("Authorization = %s, want Bearer test-key", r.Header.Get("Authorization"))
		}

		// Verify request body format
		var reqBody responsesRequest
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Fatalf("Failed to decode request body: %v", err)
		}
		if reqBody.Model != "gpt5-1-codex-max" {
			t.Errorf("Model = %s, want gpt5-1-codex-max", reqBody.Model)
		}
		if len(reqBody.Input) != 2 {
			t.Errorf("len(Input) = %d, want 2", len(reqBody.Input))
		}
		// System prompt maps to "developer" role
		if reqBody.Input[0].Role != "developer" {
			t.Errorf("Input[0].Role = %s, want developer", reqBody.Input[0].Role)
		}
		if reqBody.Input[0].Content != "You are a coding assistant." {
			t.Errorf("Input[0].Content = %q", reqBody.Input[0].Content)
		}
		if reqBody.Input[1].Role != "user" {
			t.Errorf("Input[1].Role = %s, want user", reqBody.Input[1].Role)
		}
		// Check reasoning effort
		if reqBody.Reasoning == nil || reqBody.Reasoning.Effort != "medium" {
			t.Errorf("Reasoning.Effort = %v, want medium", reqBody.Reasoning)
		}

		// Return Responses API format response
		resp := responsesResponse{
			ID:    "resp_123",
			Model: "gpt5-1-codex-max",
			Output: []responsesOutputItem{
				{
					Type:   "message",
					Status: "completed",
					Role:   "assistant",
					Content: []responsesContent{
						{Type: "output_text", Text: "Here is your code:"},
					},
				},
			},
			Usage: responsesUsage{
				InputTokens:  50,
				OutputTokens: 100,
				TotalTokens:  150,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))

	resp, err := client.Generate(context.Background(), &ai.Request{
		Model:        "gpt5-1-codex-max", // Codex model triggers Responses API
		SystemPrompt: "You are a coding assistant.",
		UserPrompt:   "Write fizzbuzz in Go",
		MaxTokens:    8192,
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if resp.Text != "Here is your code:" {
		t.Errorf("Text = %q, want %q", resp.Text, "Here is your code:")
	}
	if resp.InputTokens != 50 {
		t.Errorf("InputTokens = %d, want 50", resp.InputTokens)
	}
	if resp.OutputTokens != 100 {
		t.Errorf("OutputTokens = %d, want 100", resp.OutputTokens)
	}
}

func TestClient_Generate_ResponsesAPI_WithReasoningTokens(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := responsesResponse{
			Model: "gpt5-1-codex-max",
			Output: []responsesOutputItem{
				{
					Type:   "message",
					Status: "completed",
					Role:   "assistant",
					Content: []responsesContent{
						{Type: "output_text", Text: "Thought about it..."},
					},
				},
			},
			Usage: responsesUsage{
				InputTokens:  20,
				OutputTokens: 200,
				TotalTokens:  220,
				OutputDetails: struct {
					ReasoningTokens int `json:"reasoning_tokens"`
				}{ReasoningTokens: 150},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL), WithAPIType(APIResponses))

	resp, err := client.Generate(context.Background(), &ai.Request{
		Model:      "test-model",
		UserPrompt: "test",
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	// OutputTokens should be output - reasoning
	if resp.OutputTokens != 50 {
		t.Errorf("OutputTokens = %d, want 50 (200 - 150 reasoning)", resp.OutputTokens)
	}
	if resp.ReasonTokens != 150 {
		t.Errorf("ReasonTokens = %d, want 150", resp.ReasonTokens)
	}
}

// TestClient_Generate_ResponsesAPI_ReasoningEffort_FailLoud captures the
// M-AI-REASONING-EFFORT (v0.31.0) behavior change: the legacy untyped
// Options["reasoning_effort"] is now validated + capability-gated by the shared
// resolver BEFORE dispatch. An unregistered model with an explicit effort is
// rejected (ErrUnsupportedReasoningEffort) rather than silently passed through
// to the wire — no HTTP request is made. (The old test asserted the pre-v0.31.0
// pass-through and is intentionally replaced per no-backward-compat policy.)
func TestClient_Generate_ResponsesAPI_ReasoningEffort_FailLoud(t *testing.T) {
	hit := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hit = true
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL), WithAPIType(APIResponses))
	_, err := client.Generate(context.Background(), &ai.Request{
		Model:      "test", // NOT capability-registered
		UserPrompt: "test",
		Options:    map[string]any{"reasoning_effort": "high"},
	})
	if err == nil {
		t.Fatalf("expected fail-loud error for explicit effort on unregistered model, got nil")
	}
	if !errors.Is(err, ai.ErrUnsupportedReasoningEffort) {
		t.Fatalf("error = %v, want ErrUnsupportedReasoningEffort", err)
	}
	if hit {
		t.Fatalf("HTTP request was dispatched; validation must occur before dispatch")
	}
}

func TestClient_Generate_ResponsesAPI_PolymorphicOutput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return response with multiple output types
		resp := responsesResponse{
			Model: "gpt5-1-codex-max",
			Output: []responsesOutputItem{
				{
					Type:    "reasoning",
					Summary: []responsesSummary{{Type: "summary_text", Text: "Thinking..."}},
				},
				{
					Type: "message",
					Role: "assistant",
					Content: []responsesContent{
						{Type: "output_text", Text: "First part."},
					},
				},
				{
					Type:      "function_call",
					Name:      "run_code",
					Arguments: `{"code": "print('hello')"}`,
					CallID:    "call_123",
				},
				{
					Type: "message",
					Role: "assistant",
					Content: []responsesContent{
						{Type: "output_text", Text: "Second part."},
					},
				},
			},
			Usage: responsesUsage{InputTokens: 10, OutputTokens: 20, TotalTokens: 30},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL), WithAPIType(APIResponses))

	resp, err := client.Generate(context.Background(), &ai.Request{
		Model:      "test",
		UserPrompt: "test",
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	// Should concatenate text from all message outputs
	expected := "First part.\nSecond part."
	if resp.Text != expected {
		t.Errorf("Text = %q, want %q", resp.Text, expected)
	}
}

func TestClient_Generate_ResponsesAPI_NoTextOutput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return response with only reasoning/function_call (no message)
		resp := responsesResponse{
			Output: []responsesOutputItem{
				{Type: "reasoning", Summary: []responsesSummary{{Type: "summary_text", Text: "thinking"}}},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL), WithAPIType(APIResponses))
	_, err := client.Generate(context.Background(), &ai.Request{
		Model:      "test",
		UserPrompt: "test",
	})

	if err == nil {
		t.Fatal("Generate() expected error for no text output, got nil")
	}

	providerErr, ok := err.(*ai.ProviderError)
	if !ok {
		t.Fatalf("err type = %T, want *ai.ProviderError", err)
	}
	if providerErr.Message != "no text output in response" {
		t.Errorf("Message = %q, want %q", providerErr.Message, "no text output in response")
	}
}

func TestClient_Generate_ResponsesAPI_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
		w.Write([]byte(`{"error":{"message":"Invalid model","type":"invalid_request_error"}}`))
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL), WithAPIType(APIResponses))
	_, err := client.Generate(context.Background(), &ai.Request{
		Model:      "invalid",
		UserPrompt: "test",
	})

	if err == nil {
		t.Fatal("Generate() expected error, got nil")
	}

	providerErr, ok := err.(*ai.ProviderError)
	if !ok {
		t.Fatalf("err type = %T, want *ai.ProviderError", err)
	}
	if providerErr.StatusCode != 400 {
		t.Errorf("StatusCode = %d, want 400", providerErr.StatusCode)
	}
	if providerErr.Message != "Invalid model" {
		t.Errorf("Message = %q, want %q", providerErr.Message, "Invalid model")
	}
}

// TestGenerate_RejectsRoutingPolicy ensures openai refuses routing policies
// rather than silently ignoring them. Per CLAUDE.md no-silent-fallbacks.
func TestGenerate_RejectsRoutingPolicy(t *testing.T) {
	client := NewClient("test-key")
	_, err := client.Generate(context.Background(), &ai.Request{
		Model:      "gpt-4",
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
// rather than silently ignored. OpenAI cannot enforce a per-token price cap, so
// a policy carrying one (even with no Order/AllowFallback) must fail loud.
func TestGenerate_RejectsPriceCapOnlyPolicy(t *testing.T) {
	client := NewClient("test-key")
	_, err := client.Generate(context.Background(), &ai.Request{
		Model:      "gpt-4",
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
