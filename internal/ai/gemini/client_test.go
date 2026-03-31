package gemini

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sunholo/ailang/internal/ai"
)

func TestClient_Generate_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != "POST" {
			t.Errorf("Method = %s, want POST", r.Method)
		}
		if !strings.Contains(r.URL.Path, "generateContent") {
			t.Errorf("Path = %s, want path containing generateContent", r.URL.Path)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Content-Type = %s, want application/json", r.Header.Get("Content-Type"))
		}

		// Verify request body
		var reqBody generateRequest
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Fatalf("Failed to decode request body: %v", err)
		}
		if len(reqBody.Contents) != 1 {
			t.Errorf("len(Contents) = %d, want 1", len(reqBody.Contents))
		}
		if reqBody.Contents[0].Role != "user" {
			t.Errorf("Contents[0].Role = %s, want user", reqBody.Contents[0].Role)
		}
		if reqBody.SystemInstruction == nil {
			t.Error("SystemInstruction is nil, want non-nil")
		}

		// Return response
		resp := generateResponse{
			Candidates: []candidate{
				{
					Content: content{
						Role: "model",
						Parts: []part{
							{Text: "Hello! How can I help you today?"},
						},
					},
					FinishReason: "STOP",
				},
			},
			UsageMetadata: usageMetadata{
				PromptTokenCount:     15,
				CandidatesTokenCount: 10,
				TotalTokenCount:      25,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))

	resp, err := client.Generate(context.Background(), &ai.Request{
		Model:        "gemini-2.5-flash",
		SystemPrompt: "You are helpful.",
		UserPrompt:   "Hello",
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if resp.Text != "Hello! How can I help you today?" {
		t.Errorf("Text = %q, want %q", resp.Text, "Hello! How can I help you today?")
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
}

func TestClient_Generate_WithGenerationConfig(t *testing.T) {
	var receivedConfig *generationConfig

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody generateRequest
		json.NewDecoder(r.Body).Decode(&reqBody)
		receivedConfig = reqBody.GenerationConfig

		resp := generateResponse{
			Candidates: []candidate{
				{Content: content{Parts: []part{{Text: "ok"}}}},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	client.Generate(context.Background(), &ai.Request{
		Model:       "gemini-2.5-flash",
		UserPrompt:  "test",
		MaxTokens:   2048,
		Temperature: 0.7,
	})

	if receivedConfig == nil {
		t.Fatal("GenerationConfig is nil")
	}
	if receivedConfig.MaxOutputTokens != 2048 {
		t.Errorf("MaxOutputTokens = %d, want 2048", receivedConfig.MaxOutputTokens)
	}
	if receivedConfig.Temperature != 0.7 {
		t.Errorf("Temperature = %f, want 0.7", receivedConfig.Temperature)
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
			body:       `{"error":{"code":429,"message":"Rate limit exceeded","status":"RESOURCE_EXHAUSTED"}}`,
			wantMsg:    "Rate limit exceeded",
		},
		{
			name:       "invalid api key",
			statusCode: 401,
			body:       `{"error":{"code":401,"message":"Invalid API key","status":"UNAUTHENTICATED"}}`,
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
				Model:      "gemini-2.5-flash",
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

func TestClient_Generate_NoContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := generateResponse{
			Candidates: []candidate{},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	_, err := client.Generate(context.Background(), &ai.Request{
		Model:      "gemini-2.5-flash",
		UserPrompt: "test",
	})

	if err == nil {
		t.Fatal("Generate() expected error for empty candidates, got nil")
	}

	providerErr, ok := err.(*ai.ProviderError)
	if !ok {
		t.Fatalf("err type = %T, want *ai.ProviderError", err)
	}
	if providerErr.Message != "no content in response" {
		t.Errorf("Message = %q, want %q", providerErr.Message, "no content in response")
	}
}

func TestClient_Generate_WithReasoningTokens(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := generateResponse{
			Candidates: []candidate{
				{Content: content{Parts: []part{{Text: "thinking..."}}}},
			},
			UsageMetadata: usageMetadata{
				PromptTokenCount:     10,
				CandidatesTokenCount: 100,
				TotalTokenCount:      110,
				ThoughtsTokenCount:   80,
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	resp, err := client.Generate(context.Background(), &ai.Request{
		Model:      "gemini-3-pro",
		UserPrompt: "test",
	})

	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if resp.ReasonTokens != 80 {
		t.Errorf("ReasonTokens = %d, want 80", resp.ReasonTokens)
	}
}

func TestClient_Generate_MultipleParts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := generateResponse{
			Candidates: []candidate{
				{
					Content: content{
						Parts: []part{
							{Text: "Hello "},
							{Text: "World!"},
						},
					},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	resp, err := client.Generate(context.Background(), &ai.Request{
		Model:      "gemini-2.5-flash",
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
	if client.Name() != "gemini" {
		t.Errorf("Name() = %q, want %q", client.Name(), "gemini")
	}
}

func TestClient_NewHandler(t *testing.T) {
	client := NewClient("test-key")
	handler := client.NewHandler("gemini-2.5-flash",
		ai.WithSystemPrompt("Be helpful"),
		ai.WithMaxTokens(1000),
	)

	if handler.Provider() != client {
		t.Error("Handler.Provider() did not return expected client")
	}
	if handler.Model() != "gemini-2.5-flash" {
		t.Errorf("Handler.Model() = %q, want %q", handler.Model(), "gemini-2.5-flash")
	}
}

func TestClient_BuildURL_APIKey(t *testing.T) {
	client := NewClient("test-key")
	url, err := client.buildURL("gemini-2.5-flash")
	if err != nil {
		t.Fatalf("buildURL() error = %v", err)
	}

	expected := "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-flash:generateContent?key=test-key"
	if url != expected {
		t.Errorf("buildURL() = %q, want %q", url, expected)
	}
}

func TestClient_BuildURL_CustomBase(t *testing.T) {
	client := NewClient("test-key", WithBaseURL("http://localhost:8080"))
	url, err := client.buildURL("gemini-2.5-flash")
	if err != nil {
		t.Fatalf("buildURL() error = %v", err)
	}

	expected := "http://localhost:8080/models/gemini-2.5-flash:generateContent"
	if url != expected {
		t.Errorf("buildURL() = %q, want %q", url, expected)
	}
}

func TestClient_WithOptions(t *testing.T) {
	customHTTP := &http.Client{}
	client := NewClient("test-key",
		WithHTTPClient(customHTTP),
		WithLocation("us-central1"),
	)

	if client.httpClient != customHTTP {
		t.Error("httpClient not set correctly")
	}
	if client.location != "us-central1" {
		t.Errorf("location = %q, want %q", client.location, "us-central1")
	}
}

func TestBuildParts_FileUri(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantLen    int
		wantFile   bool   // expect fileData part
		wantInline bool   // expect inlineData part
		wantURI    string // expected fileUri
		wantData   string // expected inline data
		wantText   string // expected text part
	}{
		{
			name:     "fileUri only",
			input:    `{"mode":"multimodal","mimeType":"application/pdf","fileUri":"gs://bucket/doc.pdf","prompt":"Summarize"}`,
			wantLen:  2,
			wantFile: true,
			wantURI:  "gs://bucket/doc.pdf",
			wantText: "Summarize",
		},
		{
			name:       "data only (existing behavior)",
			input:      `{"mode":"multimodal","mimeType":"image/png","data":"iVBOR...","prompt":"Describe"}`,
			wantLen:    2,
			wantInline: true,
			wantData:   "iVBOR...",
			wantText:   "Describe",
		},
		{
			name:     "both fileUri and data — fileUri wins",
			input:    `{"mode":"multimodal","mimeType":"application/pdf","fileUri":"gs://bucket/doc.pdf","data":"base64stuff","prompt":"Read"}`,
			wantLen:  2,
			wantFile: true,
			wantURI:  "gs://bucket/doc.pdf",
			wantText: "Read",
		},
		{
			name:     "fileUri with Files API URI",
			input:    `{"mode":"multimodal","mimeType":"video/mp4","fileUri":"https://generativelanguage.googleapis.com/v1beta/files/abc123","prompt":"Analyze"}`,
			wantLen:  2,
			wantFile: true,
			wantURI:  "https://generativelanguage.googleapis.com/v1beta/files/abc123",
			wantText: "Analyze",
		},
		{
			name:    "neither fileUri nor data — falls through to text",
			input:   `{"mode":"multimodal","mimeType":"image/png"}`,
			wantLen: 1,
		},
		{
			name:     "fileUri without prompt — no text part",
			input:    `{"mode":"multimodal","mimeType":"application/pdf","fileUri":"gs://bucket/doc.pdf"}`,
			wantLen:  1,
			wantFile: true,
			wantURI:  "gs://bucket/doc.pdf",
		},
		{
			name:     "fileUri with fileName fallback",
			input:    `{"mode":"multimodal","mimeType":"application/pdf","fileUri":"gs://bucket/doc.pdf","fileName":"report.pdf"}`,
			wantLen:  2,
			wantFile: true,
			wantURI:  "gs://bucket/doc.pdf",
			wantText: "report.pdf",
		},
		{
			name:    "plain text (no multimodal)",
			input:   "Hello world",
			wantLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parts := buildParts(tt.input)
			if len(parts) != tt.wantLen {
				t.Fatalf("len(parts) = %d, want %d", len(parts), tt.wantLen)
			}

			if tt.wantFile {
				if parts[0].FileData == nil {
					t.Fatal("parts[0].FileData is nil, want non-nil")
				}
				if parts[0].FileData.FileUri != tt.wantURI {
					t.Errorf("FileUri = %q, want %q", parts[0].FileData.FileUri, tt.wantURI)
				}
				if parts[0].InlineData != nil {
					t.Error("parts[0].InlineData should be nil when FileData is set")
				}
			}

			if tt.wantInline {
				if parts[0].InlineData == nil {
					t.Fatal("parts[0].InlineData is nil, want non-nil")
				}
				if parts[0].InlineData.Data != tt.wantData {
					t.Errorf("Data = %q, want %q", parts[0].InlineData.Data, tt.wantData)
				}
				if parts[0].FileData != nil {
					t.Error("parts[0].FileData should be nil when InlineData is set")
				}
			}

			if tt.wantText != "" && len(parts) > 1 {
				if parts[1].Text != tt.wantText {
					t.Errorf("Text = %q, want %q", parts[1].Text, tt.wantText)
				}
			}
		})
	}
}

func TestBuildParts_FileData_JSONMarshal(t *testing.T) {
	parts := buildParts(`{"mode":"multimodal","mimeType":"application/pdf","fileUri":"gs://bucket/doc.pdf","prompt":"Summarize"}`)

	// Marshal the part to verify correct JSON field names for Gemini API
	data, err := json.Marshal(parts[0])
	if err != nil {
		t.Fatalf("json.Marshal error = %v", err)
	}

	var raw map[string]interface{}
	json.Unmarshal(data, &raw)

	fd, ok := raw["fileData"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected fileData key in JSON, got keys: %v", raw)
	}
	if fd["mimeType"] != "application/pdf" {
		t.Errorf("fileData.mimeType = %v, want application/pdf", fd["mimeType"])
	}
	if fd["fileUri"] != "gs://bucket/doc.pdf" {
		t.Errorf("fileData.fileUri = %v, want gs://bucket/doc.pdf", fd["fileUri"])
	}
	if _, hasInline := raw["inlineData"]; hasInline {
		t.Error("expected no inlineData key in JSON when fileData is set")
	}
}
