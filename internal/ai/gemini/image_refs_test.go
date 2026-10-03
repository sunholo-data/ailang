package gemini

// Reference-conditioned image generation (#1496): InputImages become
// inlineData parts after the text part of the user turn, and the request URL
// targets the per-call model. httptest only — no network.

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ai"
)

func TestGenerate_ImageRequest_ReferenceImagesAsInlineData(t *testing.T) {
	var body []byte
	var path string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ = io.ReadAll(r.Body)
		path = r.URL.Path
		_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{"inlineData":{"mimeType":"image/png","data":"b3V0"}}]},"finishReason":"STOP"}]}`))
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	resp, err := client.Generate(context.Background(), &ai.Request{
		Model:              "gemini-2.5-flash-image",
		UserPrompt:         "same person, smiling",
		ResponseModalities: []string{"IMAGE"},
		InputImages: []ai.ImagePart{
			{Source: "cmVmMQ==", Mime: "image/png"},
			{Source: "data:image/jpeg;base64,cmVmMg==", Mime: ""},
		},
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if string(resp.ImageData) != "out" {
		t.Errorf("ImageData = %q, want out", resp.ImageData)
	}
	if !strings.Contains(path, "gemini-2.5-flash-image") {
		t.Errorf("path = %s, want the request model", path)
	}

	var req generateRequest
	if err := json.Unmarshal(body, &req); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if len(req.Contents) != 1 {
		t.Fatalf("contents = %d, want 1", len(req.Contents))
	}
	parts := req.Contents[0].Parts
	if len(parts) != 3 {
		t.Fatalf("parts = %d, want 3 (text + 2 refs): %s", len(parts), body)
	}
	if parts[0].Text != "same person, smiling" {
		t.Errorf("parts[0] = %+v, want the prompt text", parts[0])
	}
	if parts[1].InlineData == nil || parts[1].InlineData.Data != "cmVmMQ==" || parts[1].InlineData.MimeType != "image/png" {
		t.Errorf("parts[1] = %+v", parts[1].InlineData)
	}
	// data-URI sources are split: raw base64 payload, declared media type.
	if parts[2].InlineData == nil || parts[2].InlineData.Data != "cmVmMg==" || parts[2].InlineData.MimeType != "image/jpeg" {
		t.Errorf("parts[2] = %+v", parts[2].InlineData)
	}
	if req.GenerationConfig == nil || len(req.GenerationConfig.ResponseModalities) != 1 {
		t.Errorf("generationConfig.responseModalities missing: %s", body)
	}
}

// Without references the user turn is the single text part it always was.
func TestGenerate_ImageRequest_NoReferencesUnchanged(t *testing.T) {
	var body []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ = io.ReadAll(r.Body)
		_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{"inlineData":{"mimeType":"image/png","data":"b3V0"}}]}}]}`))
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	if _, err := client.Generate(context.Background(), &ai.Request{
		Model:              "gemini-2.5-flash-image",
		UserPrompt:         "a lighthouse",
		ResponseModalities: []string{"IMAGE"},
	}); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	want := `{"contents":[{"role":"user","parts":[{"text":"a lighthouse"}]}],"generationConfig":{"responseModalities":["IMAGE"]}}`
	if string(body) != want {
		t.Errorf("body changed\n got: %s\nwant: %s", body, want)
	}
}
