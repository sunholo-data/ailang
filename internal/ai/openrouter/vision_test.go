package openrouter

// Vision-input (M-STD-AI-VISION-INPUT, v0.30.0) wire-format regression tests
// for the OpenRouter Step path.
//
// OpenRouter speaks OpenAI Chat Completions on the wire and builds its
// messages array via the shared openai.BuildChatStepRequest helper (see
// step.go / streamstep.go). That helper already maps ai.Message.Images to
// the OpenAI multimodal content array (text part first when non-empty, then
// one image_url part per image, each carrying a data-URI). These tests drive
// the FULL OpenRouter Step path end-to-end (BuildChatStepRequest ->
// applyCacheHintsForRoute -> marshalStepBodyWithProvider -> HTTP body) and
// inspect the captured wire bytes, so a future fork of the builder that broke
// image passthrough — or a regression in the string-only fast path — would be
// caught here rather than only in the openai package's own tests.

import (
	"context"
	"encoding/json"
	"testing"

	"net/http/httptest"

	"github.com/sunholo-data/ailang/internal/ai"
)

// visionReqBody is the minimal slice of the OpenRouter wire body these tests
// care about: the messages array with Content left as RawMessage so we can
// tell a plain JSON string from a multimodal parts array.
type visionReqBody struct {
	Messages []visionReqMessage `json:"messages"`
}

type visionReqMessage struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

// visionContentPart mirrors the OpenAI multimodal content part shape that
// OpenRouter forwards: {"type":"text","text":...} or
// {"type":"image_url","image_url":{"url":<data-uri>}}.
type visionContentPart struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	ImageURL *struct {
		URL string `json:"url"`
	} `json:"image_url,omitempty"`
}

// captureStep drives client.Step against a capture server and returns the
// decoded outbound wire body. The canned response is a trivial success so
// Step returns without error.
func captureStep(t *testing.T, req *ai.Request) visionReqBody {
	t.Helper()
	h := &captureHandler{}
	server := httptest.NewServer(h)
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	if _, err := client.Step(context.Background(), req); err != nil {
		t.Fatalf("Step() error = %v", err)
	}
	if len(h.captured) == 0 {
		t.Fatal("no request body captured")
	}
	var body visionReqBody
	if err := json.Unmarshal(h.captured, &body); err != nil {
		t.Fatalf("unmarshal captured body: %v\nbody: %s", err, h.captured)
	}
	return body
}

// lastUserMessage returns the final user-role message from the captured body.
func lastUserMessage(t *testing.T, body visionReqBody) visionReqMessage {
	t.Helper()
	for i := len(body.Messages) - 1; i >= 0; i-- {
		if body.Messages[i].Role == "user" {
			return body.Messages[i]
		}
	}
	t.Fatal("no user message in captured body")
	return visionReqMessage{}
}

// TestStep_Vision_UserImageAndText_BecomesPartsArray verifies that a user
// message carrying text + one raw-base64 image serializes its content as a
// parts array: text part first, then an image_url part whose url is a
// data-URI built from the raw base64 source and mime.
func TestStep_Vision_UserImageAndText_BecomesPartsArray(t *testing.T) {
	body := captureStep(t, &ai.Request{
		Model: "anthropic/claude-sonnet-4.5",
		Messages: []ai.Message{{
			Role:    "user",
			Content: "What is in this image?",
			Images:  []ai.ImagePart{{Source: "AAAABBBB", Mime: "image/png"}},
		}},
	})

	um := lastUserMessage(t, body)
	if len(um.Content) == 0 || um.Content[0] != '[' {
		t.Fatalf("user content should be a JSON array, got: %s", um.Content)
	}

	var parts []visionContentPart
	if err := json.Unmarshal(um.Content, &parts); err != nil {
		t.Fatalf("unmarshal content parts: %v", err)
	}
	if len(parts) != 2 {
		t.Fatalf("want 2 content parts (text + image), got %d: %s", len(parts), um.Content)
	}
	if parts[0].Type != "text" || parts[0].Text != "What is in this image?" {
		t.Errorf("part[0] = %+v, want text part %q", parts[0], "What is in this image?")
	}
	if parts[1].Type != "image_url" {
		t.Fatalf("part[1].Type = %q, want image_url", parts[1].Type)
	}
	if parts[1].ImageURL == nil {
		t.Fatal("part[1].image_url missing")
	}
	const wantURL = "data:image/png;base64,AAAABBBB"
	if parts[1].ImageURL.URL != wantURL {
		t.Errorf("image url = %q, want %q", parts[1].ImageURL.URL, wantURL)
	}
}

// TestStep_Vision_DataURISource_PassedThrough verifies that when the image
// Source is already a data-URI it is forwarded verbatim (not re-wrapped).
func TestStep_Vision_DataURISource_PassedThrough(t *testing.T) {
	const dataURI = "data:image/jpeg;base64,ZZZZ9999"
	body := captureStep(t, &ai.Request{
		Model: "openai/gpt-4o",
		Messages: []ai.Message{{
			Role:   "user",
			Images: []ai.ImagePart{{Source: dataURI, Mime: "image/jpeg"}},
		}},
	})

	um := lastUserMessage(t, body)
	var parts []visionContentPart
	if err := json.Unmarshal(um.Content, &parts); err != nil {
		t.Fatalf("unmarshal content parts: %v\ncontent: %s", err, um.Content)
	}
	// No text part expected (Content was empty), only the image part.
	if len(parts) != 1 {
		t.Fatalf("want 1 content part (image only), got %d: %s", len(parts), um.Content)
	}
	if parts[0].Type != "image_url" || parts[0].ImageURL == nil {
		t.Fatalf("part[0] = %+v, want image_url part", parts[0])
	}
	if parts[0].ImageURL.URL != dataURI {
		t.Errorf("image url = %q, want passthrough %q", parts[0].ImageURL.URL, dataURI)
	}
}

// TestStep_Vision_NoImages_ContentStaysString is the no-regression guard:
// with Images empty the user content MUST serialize as a plain JSON string,
// byte-for-byte identical to the pre-vision wire shape.
func TestStep_Vision_NoImages_ContentStaysString(t *testing.T) {
	body := captureStep(t, &ai.Request{
		Model:    "anthropic/claude-sonnet-4.5",
		Messages: []ai.Message{{Role: "user", Content: "plain text only"}},
	})

	um := lastUserMessage(t, body)
	if len(um.Content) == 0 || um.Content[0] != '"' {
		t.Fatalf("user content should be a JSON string, got: %s", um.Content)
	}
	var s string
	if err := json.Unmarshal(um.Content, &s); err != nil {
		t.Fatalf("content is not a JSON string: %v", err)
	}
	if s != "plain text only" {
		t.Errorf("content = %q, want %q", s, "plain text only")
	}
}
