package openai

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/ai"
)

// M-LYCEUM-PROVIDER M3: the transport is non-streaming, so WallMS must be set
// (the whole round trip) and TTFTMS must stay 0 (unmeasured — the body arrives
// at once; any number would be wall time, not TTFT).
func TestClient_Generate_ChatCompletions_Timing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(20 * time.Millisecond) // measurable but fast
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(chatResponse{
			ID:     "chatcmpl-timing",
			Object: "chat.completion",
			Model:  "test-model",
			Choices: []chatChoice{{
				Index:        0,
				FinishReason: "stop",
				Message:      chatMessage{Role: "assistant", Content: "ok"},
			}},
			Usage: chatUsage{PromptTokens: 5, CompletionTokens: 2, TotalTokens: 7},
		})
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	resp, err := client.Generate(context.Background(), &ai.Request{
		Model:      "test-model",
		UserPrompt: "hi",
		MaxTokens:  10,
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if resp.WallMS <= 0 {
		t.Errorf("WallMS = %d, want > 0 (client-observed round trip)", resp.WallMS)
	}
	if resp.TTFTMS != 0 {
		t.Errorf("TTFTMS = %d, want 0 (non-streaming transport cannot measure TTFT)", resp.TTFTMS)
	}
}

// Failed call must carry the wall time up to the error — that is the datum
// distinguishing "gateway 504 after 30s" from "after 6 minutes".
func TestClient_Generate_ErrorCarriesWallMS(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(20 * time.Millisecond)
		w.WriteHeader(http.StatusGatewayTimeout)
		w.Write([]byte(`{"error":{"message":"upstream request timeout"}}`))
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	_, err := client.Generate(context.Background(), &ai.Request{
		Model:      "test-model",
		UserPrompt: "hi",
		MaxTokens:  10,
	})
	if err == nil {
		t.Fatal("Generate should fail on 504")
	}
	var pe *ai.ProviderError
	if !errors.As(err, &pe) {
		t.Fatalf("error is %T, want *ai.ProviderError", err)
	}
	if pe.StatusCode != http.StatusGatewayTimeout {
		t.Errorf("StatusCode = %d, want 504", pe.StatusCode)
	}
	if pe.WallMS <= 0 {
		t.Errorf("WallMS = %d, want > 0 on the failed call", pe.WallMS)
	}
}
