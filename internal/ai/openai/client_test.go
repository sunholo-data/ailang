package openai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sunholo/ailang/internal/ai"
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
		if reqBody.Model != "gpt-5-mini" {
			t.Errorf("Model = %s, want gpt-5-mini", reqBody.Model)
		}
		// GPT-5 models use MaxCompletionTokens, not MaxTokens
		if reqBody.MaxCompletionTokens != 2048 {
			t.Errorf("MaxCompletionTokens = %d, want 2048", reqBody.MaxCompletionTokens)
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
			Model:  "gpt-5-mini-20250901",
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

	client := NewClient("test-key", WithBaseURL(server.URL))

	resp, err := client.Generate(context.Background(), &ai.Request{
		Model:        "gpt-5-mini",
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

func TestClient_Generate_WithReasoningTokens(t *testing.T) {
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

	client := NewClient("test-key", WithBaseURL(server.URL))

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

			client := NewClient("test-key", WithBaseURL(server.URL))
			_, err := client.Generate(context.Background(), &ai.Request{
				Model:      "gpt-5",
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

func TestClient_Generate_NoChoices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := chatResponse{
			Choices: []chatChoice{},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	_, err := client.Generate(context.Background(), &ai.Request{
		Model:      "gpt-5",
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
		{"gpt-5", APIChatCompletions},
		{"gpt-5-mini", APIChatCompletions},
		{"gpt-5.1", APIChatCompletions},
		{"gpt-5.1-codex-max", APIResponses},
		{"codex-mini", APIResponses},
		{"o1-preview", APIChatCompletions},
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

func TestClient_Generate_WithSeed(t *testing.T) {
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

	client := NewClient("test-key", WithBaseURL(server.URL))
	client.Generate(context.Background(), &ai.Request{
		Model:      "gpt-5",
		UserPrompt: "test",
		Options:    map[string]any{"seed": int64(42)},
	})

	if receivedSeed == nil || *receivedSeed != 42 {
		t.Errorf("Seed not passed correctly, got %v", receivedSeed)
	}
}

func TestClient_Generate_DefaultMaxTokens(t *testing.T) {
	var receivedMaxCompletionTokens int

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody chatRequest
		json.NewDecoder(r.Body).Decode(&reqBody)
		// GPT-5 uses MaxCompletionTokens, not MaxTokens
		receivedMaxCompletionTokens = reqBody.MaxCompletionTokens

		resp := chatResponse{
			Choices: []chatChoice{{Message: chatMessage{Content: "ok"}}},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	client.Generate(context.Background(), &ai.Request{
		Model:      "gpt-5",
		UserPrompt: "test",
		// MaxTokens not set - should default to 4096
	})

	if receivedMaxCompletionTokens != 4096 {
		t.Errorf("default MaxCompletionTokens = %d, want 4096", receivedMaxCompletionTokens)
	}
}
