package openai

import (
	"encoding/json"
	"testing"

	"github.com/sunholo-data/ailang/internal/ai"
)

// visionContentPart mirrors the multimodal content part wire shape so tests can
// inspect the array OpenAI receives for a user message carrying images.
type visionContentPart struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	ImageURL *struct {
		URL string `json:"url"`
	} `json:"image_url,omitempty"`
}

// buildAndMarshalUserContent runs BuildChatStepRequest and returns the raw
// JSON bytes of the FIRST message's content field, marshaling the whole body
// first so the assertion reflects real wire bytes.
func buildAndMarshalUserContent(t *testing.T, req *ai.Request) json.RawMessage {
	t.Helper()
	apiReq, aiErr := BuildChatStepRequest(req, ai.ReasoningDecision{})
	if aiErr != nil {
		t.Fatalf("BuildChatStepRequest returned error: %v", aiErr)
	}
	body, err := json.Marshal(apiReq)
	if err != nil {
		t.Fatalf("marshal request body: %v", err)
	}
	var probe struct {
		Messages []struct {
			Content json.RawMessage `json:"content"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(body, &probe); err != nil {
		t.Fatalf("unmarshal request body: %v", err)
	}
	if len(probe.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(probe.Messages))
	}
	return probe.Messages[0].Content
}

// TestBuildChatStepRequest_UserImageAndText verifies a user message with text
// plus one raw-base64 image becomes a content ARRAY: a text part first, then an
// image_url part whose url is a properly assembled data-URI.
func TestBuildChatStepRequest_UserImageAndText(t *testing.T) {
	req := &ai.Request{
		Model: "gpt-4o",
		Messages: []ai.Message{
			{
				Role:    "user",
				Content: "what is in this image?",
				Images:  []ai.ImagePart{{Source: "iVBORw0KGgo=", Mime: "image/png"}},
			},
		},
	}
	content := buildAndMarshalUserContent(t, req)

	var parts []visionContentPart
	if err := json.Unmarshal(content, &parts); err != nil {
		t.Fatalf("content is not a JSON array: %v (raw=%s)", err, content)
	}
	if len(parts) != 2 {
		t.Fatalf("expected 2 parts, got %d (raw=%s)", len(parts), content)
	}
	if parts[0].Type != "text" || parts[0].Text != "what is in this image?" {
		t.Errorf("part[0] wrong: %+v", parts[0])
	}
	if parts[1].Type != "image_url" || parts[1].ImageURL == nil {
		t.Fatalf("part[1] wrong: %+v", parts[1])
	}
	if got, want := parts[1].ImageURL.URL, "data:image/png;base64,iVBORw0KGgo="; got != want {
		t.Errorf("image url = %q, want %q", got, want)
	}
}

// TestBuildChatStepRequest_UserImageDataURIPassthrough verifies a source that is
// already a data-URI is passed through unchanged in the image_url url.
func TestBuildChatStepRequest_UserImageDataURIPassthrough(t *testing.T) {
	const dataURI = "data:image/jpeg;base64,/9j/4AAQSkZJRg=="
	req := &ai.Request{
		Model: "gpt-4o",
		Messages: []ai.Message{
			{
				Role:   "user",
				Images: []ai.ImagePart{{Source: dataURI, Mime: "image/jpeg"}},
			},
		},
	}
	content := buildAndMarshalUserContent(t, req)

	var parts []visionContentPart
	if err := json.Unmarshal(content, &parts); err != nil {
		t.Fatalf("content is not a JSON array: %v (raw=%s)", err, content)
	}
	// No text part (Content == ""), so only the single image part.
	if len(parts) != 1 {
		t.Fatalf("expected 1 part, got %d (raw=%s)", len(parts), content)
	}
	if parts[0].Type != "image_url" || parts[0].ImageURL == nil {
		t.Fatalf("part[0] wrong: %+v", parts[0])
	}
	if got := parts[0].ImageURL.URL; got != dataURI {
		t.Errorf("image url = %q, want passthrough %q", got, dataURI)
	}
}

// TestBuildChatStepRequest_NoImagesPlainString is the regression guard: a user
// message with no images must marshal content as a plain JSON STRING, identical
// to pre-vision behaviour.
func TestBuildChatStepRequest_NoImagesPlainString(t *testing.T) {
	req := &ai.Request{
		Model: "gpt-4o",
		Messages: []ai.Message{
			{Role: "user", Content: "hello world"},
		},
	}
	content := buildAndMarshalUserContent(t, req)

	if got, want := string(content), `"hello world"`; got != want {
		t.Fatalf("content = %s, want plain string %s", got, want)
	}
	var s string
	if err := json.Unmarshal(content, &s); err != nil {
		t.Fatalf("content did not decode as a JSON string: %v", err)
	}
	if s != "hello world" {
		t.Errorf("decoded string = %q, want %q", s, "hello world")
	}
}
