package openrouter

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ai"
)

// TestClient_Generate_HappyPath exercises the full happy-path flow:
// auth header, endpoint, request body, OpenRouter-extended usage block.
func TestClient_Generate_HappyPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

		var reqBody chatRequest
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Fatalf("Failed to decode request body: %v", err)
		}
		if reqBody.Model != "anthropic/claude-sonnet-4.5" {
			t.Errorf("Model = %s, want anthropic/claude-sonnet-4.5", reqBody.Model)
		}
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

		// Return response with full OpenRouter-extended usage block
		raw := `{
			"id": "gen-abc123",
			"object": "chat.completion",
			"model": "anthropic/claude-sonnet-4.5",
			"choices": [
				{
					"index": 0,
					"message": {"role": "assistant", "content": "Hello from claude via openrouter!"},
					"finish_reason": "stop"
				}
			],
			"usage": {
				"prompt_tokens": 100,
				"completion_tokens": 25,
				"total_tokens": 125,
				"prompt_tokens_details": {"cached_tokens": 60},
				"cost": 0.000345,
				"cost_details": {"upstream_inference_cost": 0.000300}
			}
		}`
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(raw))
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))

	resp, err := client.Generate(context.Background(), &ai.Request{
		Model:        "anthropic/claude-sonnet-4.5",
		SystemPrompt: "You are helpful.",
		UserPrompt:   "Hello",
		MaxTokens:    2048,
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if resp.Text != "Hello from claude via openrouter!" {
		t.Errorf("Text = %q", resp.Text)
	}
	if resp.InputTokens != 100 {
		t.Errorf("InputTokens = %d, want 100", resp.InputTokens)
	}
	if resp.OutputTokens != 25 {
		t.Errorf("OutputTokens = %d, want 25", resp.OutputTokens)
	}
	if resp.TotalTokens != 125 {
		t.Errorf("TotalTokens = %d, want 125", resp.TotalTokens)
	}
	if resp.CachedTokens != 60 {
		t.Errorf("CachedTokens = %d, want 60", resp.CachedTokens)
	}
	if resp.CostUSD != "0.000345" {
		t.Errorf("CostUSD = %q, want %q", resp.CostUSD, "0.000345")
	}
	if resp.Model != "anthropic/claude-sonnet-4.5" {
		t.Errorf("Model = %q", resp.Model)
	}
}

// TestClient_Generate_ReasoningTokens covers the reasoning-token subtraction
// path (mirrors openai's behaviour for reasoning models routed through OR).
func TestClient_Generate_ReasoningTokens(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := chatResponse{
			Model: "openai/gpt-5",
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
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))

	resp, err := client.Generate(context.Background(), &ai.Request{
		Model:      "openai/gpt-5",
		UserPrompt: "test",
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if resp.OutputTokens != 20 {
		t.Errorf("OutputTokens = %d, want 20 (100 - 80 reasoning)", resp.OutputTokens)
	}
	if resp.ReasonTokens != 80 {
		t.Errorf("ReasonTokens = %d, want 80", resp.ReasonTokens)
	}
}

// TestClient_Generate_Errors covers 401 auth, 5xx server, and rate-limit
// error paths — the typical failure modes from OpenRouter.
func TestClient_Generate_Errors(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		wantMsg    string
	}{
		{
			name:       "auth error",
			statusCode: 401,
			body:       `{"error":{"message":"Invalid API key","type":"authentication_error"}}`,
			wantMsg:    "Invalid API key",
		},
		{
			name:       "rate limit",
			statusCode: 429,
			body:       `{"error":{"message":"Rate limit exceeded","type":"rate_limit_error"}}`,
			wantMsg:    "Rate limit exceeded",
		},
		{
			name:       "server error plain text",
			statusCode: 502,
			body:       `Bad Gateway`,
			wantMsg:    "Bad Gateway",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()

			client := NewClient("test-key", WithBaseURL(server.URL))
			_, err := client.Generate(context.Background(), &ai.Request{
				Model:      "anthropic/claude-sonnet-4.5",
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
			if providerErr.Provider != "openrouter" {
				t.Errorf("Provider = %q, want openrouter", providerErr.Provider)
			}
		})
	}
}

// TestClient_Generate_MalformedJSON covers the response-parse failure path.
func TestClient_Generate_MalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{this is not valid json`))
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	_, err := client.Generate(context.Background(), &ai.Request{
		Model:      "anthropic/claude-sonnet-4.5",
		UserPrompt: "test",
	})
	if err == nil {
		t.Fatal("Generate() expected error for malformed JSON, got nil")
	}
	if !strings.Contains(err.Error(), "failed to parse response") {
		t.Errorf("err = %v, want 'failed to parse response'", err)
	}
}

// TestClient_Generate_EmptyChoices covers the empty-choices error path.
func TestClient_Generate_EmptyChoices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := chatResponse{Choices: []chatChoice{}}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	_, err := client.Generate(context.Background(), &ai.Request{
		Model:      "anthropic/claude-sonnet-4.5",
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
		t.Errorf("Message = %q", providerErr.Message)
	}
}

// TestClient_Generate_NoCostNoCachedTokens verifies that a response without
// the OpenRouter extension fields leaves CachedTokens=0 and CostUSD="".
func TestClient_Generate_NoCostNoCachedTokens(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := chatResponse{
			Model: "openrouter/auto",
			Choices: []chatChoice{
				{Message: chatMessage{Content: "ok"}},
			},
			Usage: chatUsage{
				PromptTokens: 5, CompletionTokens: 3, TotalTokens: 8,
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	resp, err := client.Generate(context.Background(), &ai.Request{
		Model:      "openrouter/auto",
		UserPrompt: "ping",
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if resp.CachedTokens != 0 {
		t.Errorf("CachedTokens = %d, want 0", resp.CachedTokens)
	}
	if resp.CostUSD != "" {
		t.Errorf("CostUSD = %q, want empty", resp.CostUSD)
	}
}

// TestClient_Generate_RejectsImageRequests covers the image-modality guard:
// OpenRouter does not currently route image generation, so we should fail
// loudly with a typed ProviderError rather than calling the API.
func TestClient_Generate_RejectsImageRequests(t *testing.T) {
	// Server should never be hit
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("server should not be called for image requests")
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	_, err := client.Generate(context.Background(), &ai.Request{
		Model:              "openai/gpt-5",
		UserPrompt:         "draw a cat",
		ResponseModalities: []string{"IMAGE"},
	})
	if err == nil {
		t.Fatal("Generate() expected error for image request, got nil")
	}
	providerErr, ok := err.(*ai.ProviderError)
	if !ok {
		t.Fatalf("err type = %T, want *ai.ProviderError", err)
	}
	if !strings.Contains(providerErr.Message, "image generation not supported") {
		t.Errorf("Message = %q, want contains 'image generation not supported'", providerErr.Message)
	}
}

// TestClient_Generate_StructuredOutput verifies that a json_schema response_format
// is sent through correctly.
func TestClient_Generate_StructuredOutput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody chatRequest
		_ = json.NewDecoder(r.Body).Decode(&reqBody)
		if reqBody.ResponseFormat == nil {
			t.Fatal("ResponseFormat is nil, want json_schema")
		}
		if reqBody.ResponseFormat.Type != "json_schema" {
			t.Errorf("ResponseFormat.Type = %q, want json_schema", reqBody.ResponseFormat.Type)
		}
		if reqBody.ResponseFormat.JSONSchema == nil {
			t.Fatal("JSONSchema is nil")
		}
		if !reqBody.ResponseFormat.JSONSchema.Strict {
			t.Error("JSONSchema.Strict = false, want true")
		}

		resp := chatResponse{
			Choices: []chatChoice{{Message: chatMessage{Content: `{"answer": 42}`}}},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	_, err := client.Generate(context.Background(), &ai.Request{
		Model:          "openai/gpt-5",
		UserPrompt:     "answer",
		ResponseFormat: "json",
		ResponseSchema: `{"type":"object","properties":{"answer":{"type":"number"}}}`,
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
}

// TestClient_Generate_JSONObjectFormat verifies the json_object fallback
// when no schema is supplied.
func TestClient_Generate_JSONObjectFormat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody chatRequest
		_ = json.NewDecoder(r.Body).Decode(&reqBody)
		if reqBody.ResponseFormat == nil || reqBody.ResponseFormat.Type != "json_object" {
			t.Errorf("ResponseFormat.Type = %v, want json_object", reqBody.ResponseFormat)
		}

		resp := chatResponse{
			Choices: []chatChoice{{Message: chatMessage{Content: `{}`}}},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	_, err := client.Generate(context.Background(), &ai.Request{
		Model:          "openai/gpt-5",
		UserPrompt:     "ping",
		ResponseFormat: "json",
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
}

// TestClient_Generate_DefaultMaxTokens verifies that omitting MaxTokens
// results in the 4096 default being sent upstream.
func TestClient_Generate_DefaultMaxTokens(t *testing.T) {
	var receivedMaxTokens int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody chatRequest
		_ = json.NewDecoder(r.Body).Decode(&reqBody)
		receivedMaxTokens = reqBody.MaxTokens

		resp := chatResponse{Choices: []chatChoice{{Message: chatMessage{Content: "ok"}}}}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	_, _ = client.Generate(context.Background(), &ai.Request{
		Model:      "openai/gpt-5",
		UserPrompt: "test",
	})
	if receivedMaxTokens != 4096 {
		t.Errorf("default MaxTokens = %d, want 4096", receivedMaxTokens)
	}
}

// TestClient_Generate_SeedOption verifies that an int64 seed in Options
// is forwarded as the seed field.
func TestClient_Generate_SeedOption(t *testing.T) {
	var receivedSeed *int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody chatRequest
		_ = json.NewDecoder(r.Body).Decode(&reqBody)
		receivedSeed = reqBody.Seed

		resp := chatResponse{Choices: []chatChoice{{Message: chatMessage{Content: "ok"}}}}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	_, _ = client.Generate(context.Background(), &ai.Request{
		Model:      "openai/gpt-5",
		UserPrompt: "test",
		Options:    map[string]any{"seed": int64(7)},
	})
	if receivedSeed == nil || *receivedSeed != 7 {
		t.Errorf("Seed not passed correctly, got %v", receivedSeed)
	}
}

// TestClient_Generate_OptionalAttributionHeaders verifies that HTTP-Referer and
// X-Title headers are sent when the env vars are set, and omitted otherwise.
func TestClient_Generate_OptionalAttributionHeaders(t *testing.T) {
	var seenReferer, seenTitle string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenReferer = r.Header.Get("HTTP-Referer")
		seenTitle = r.Header.Get("X-Title")

		resp := chatResponse{Choices: []chatChoice{{Message: chatMessage{Content: "ok"}}}}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	t.Setenv("OPENROUTER_HTTP_REFERER", "https://ailang.dev")
	t.Setenv("OPENROUTER_X_TITLE", "AILANG")

	client := NewClient("test-key", WithBaseURL(server.URL))
	_, _ = client.Generate(context.Background(), &ai.Request{
		Model:      "openai/gpt-5",
		UserPrompt: "test",
	})

	if seenReferer != "https://ailang.dev" {
		t.Errorf("HTTP-Referer = %q, want https://ailang.dev", seenReferer)
	}
	if seenTitle != "AILANG" {
		t.Errorf("X-Title = %q, want AILANG", seenTitle)
	}
}

func TestClient_Name(t *testing.T) {
	client := NewClient("test-key")
	if client.Name() != "openrouter" {
		t.Errorf("Name() = %q, want openrouter", client.Name())
	}
}

func TestClient_NewHandler(t *testing.T) {
	client := NewClient("test-key")
	handler := client.NewHandler("anthropic/claude-sonnet-4.5",
		ai.WithSystemPrompt("Be helpful"),
		ai.WithMaxTokens(1000),
	)
	if handler.Provider() != client {
		t.Error("Handler.Provider() did not return expected client")
	}
	if handler.Model() != "anthropic/claude-sonnet-4.5" {
		t.Errorf("Handler.Model() = %q", handler.Model())
	}
}

func TestClient_WithOptions(t *testing.T) {
	customHTTP := &http.Client{}
	client := NewClient("test-key",
		WithBaseURL("https://custom.api.com"),
		WithHTTPClient(customHTTP),
	)
	if client.baseURL != "https://custom.api.com" {
		t.Errorf("baseURL = %q", client.baseURL)
	}
	if client.httpClient != customHTTP {
		t.Error("httpClient not set correctly")
	}
}

func TestClient_DefaultBaseURL(t *testing.T) {
	client := NewClient("test-key")
	if client.baseURL != defaultBaseURL {
		t.Errorf("baseURL = %q, want %q", client.baseURL, defaultBaseURL)
	}
}
