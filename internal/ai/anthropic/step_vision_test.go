package anthropic

import (
	"encoding/json"
	"testing"

	"github.com/sunholo-data/ailang/internal/ai"
)

// visionBlock is the decoded shape of a single content block for vision tests.
type visionBlock struct {
	Type   string `json:"type"`
	Text   string `json:"text,omitempty"`
	Source *struct {
		Type      string `json:"type"`
		MediaType string `json:"media_type"`
		Data      string `json:"data"`
	} `json:"source,omitempty"`
}

// marshalUserContent runs buildStepRequest for a single user message and
// returns the raw JSON of its content field.
func marshalUserContent(t *testing.T, m ai.Message) json.RawMessage {
	t.Helper()
	req := &ai.Request{
		Model:    "claude-test",
		Messages: []ai.Message{m},
	}
	apiReq, aiErr := buildStepRequest(req, ai.ReasoningDecision{})
	if aiErr != nil {
		t.Fatalf("buildStepRequest returned error: %v", aiErr)
	}
	if len(apiReq.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(apiReq.Messages))
	}
	return apiReq.Messages[0].Content
}

// TestBuildStepRequest_ImageWithText verifies a user message with text + one
// raw-base64 image emits a content array: text block first, then image block.
func TestBuildStepRequest_ImageWithText(t *testing.T) {
	content := marshalUserContent(t, ai.Message{
		Role:    "user",
		Content: "What is in this image?",
		Images:  []ai.ImagePart{{Source: "RAWBASE64DATA", Mime: "image/jpeg"}},
	})

	var blocks []visionBlock
	if err := json.Unmarshal(content, &blocks); err != nil {
		t.Fatalf("content is not a block array: %v (raw=%s)", err, content)
	}
	if len(blocks) != 2 {
		t.Fatalf("expected 2 blocks (text+image), got %d: %s", len(blocks), content)
	}

	if blocks[0].Type != "text" || blocks[0].Text != "What is in this image?" {
		t.Errorf("block[0] = %+v, want text block with prompt", blocks[0])
	}

	img := blocks[1]
	if img.Type != "image" {
		t.Fatalf("block[1].Type = %q, want image", img.Type)
	}
	if img.Source == nil {
		t.Fatalf("block[1].Source is nil")
	}
	if img.Source.Type != "base64" {
		t.Errorf("source.type = %q, want base64", img.Source.Type)
	}
	if img.Source.MediaType != "image/jpeg" {
		t.Errorf("source.media_type = %q, want image/jpeg", img.Source.MediaType)
	}
	if img.Source.Data != "RAWBASE64DATA" {
		t.Errorf("source.data = %q, want RAWBASE64DATA", img.Source.Data)
	}
}

// TestBuildStepRequest_ImageOnly verifies an image-only message (empty Content)
// omits the text block.
func TestBuildStepRequest_ImageOnly(t *testing.T) {
	content := marshalUserContent(t, ai.Message{
		Role:   "user",
		Images: []ai.ImagePart{{Source: "ABC", Mime: "image/png"}},
	})
	var blocks []visionBlock
	if err := json.Unmarshal(content, &blocks); err != nil {
		t.Fatalf("content is not a block array: %v (raw=%s)", err, content)
	}
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block (image only), got %d: %s", len(blocks), content)
	}
	if blocks[0].Type != "image" {
		t.Errorf("block[0].Type = %q, want image", blocks[0].Type)
	}
}

// TestBuildStepRequest_DataURISplit verifies a data-URI source is split into a
// raw-base64 data field and a derived media_type.
func TestBuildStepRequest_DataURISplit(t *testing.T) {
	content := marshalUserContent(t, ai.Message{
		Role:    "user",
		Content: "describe",
		// Note: no Mime supplied — media_type must come from the data URI.
		Images: []ai.ImagePart{{Source: "data:image/png;base64,ABC123"}},
	})
	var blocks []visionBlock
	if err := json.Unmarshal(content, &blocks); err != nil {
		t.Fatalf("content is not a block array: %v (raw=%s)", err, content)
	}
	if len(blocks) != 2 {
		t.Fatalf("expected 2 blocks, got %d: %s", len(blocks), content)
	}
	img := blocks[1]
	if img.Source == nil {
		t.Fatalf("image block has no source")
	}
	if img.Source.MediaType != "image/png" {
		t.Errorf("source.media_type = %q, want image/png", img.Source.MediaType)
	}
	if img.Source.Data != "ABC123" {
		t.Errorf("source.data = %q, want ABC123 (prefix stripped)", img.Source.Data)
	}
}

// TestBuildStepRequest_NoImagesPlainString is the critical no-regression check:
// a user message with no images must marshal content as a plain JSON string,
// byte-for-byte identical to the pre-vision behavior.
func TestBuildStepRequest_NoImagesPlainString(t *testing.T) {
	content := marshalUserContent(t, ai.Message{
		Role:    "user",
		Content: "hello world",
	})
	// A plain string decodes into a Go string; a block array does not.
	var s string
	if err := json.Unmarshal(content, &s); err != nil {
		t.Fatalf("expected plain JSON string content, got: %s (err=%v)", content, err)
	}
	if s != "hello world" {
		t.Errorf("content = %q, want hello world", s)
	}
	// Byte-for-byte: json.Marshal of the string must equal the wire content.
	want, _ := json.Marshal("hello world")
	if string(content) != string(want) {
		t.Errorf("content bytes = %s, want %s", content, want)
	}
}

// TestSplitDataURI covers the helper directly.
func TestSplitDataURI(t *testing.T) {
	tests := []struct {
		name      string
		source    string
		mime      string
		wantMedia string
		wantData  string
	}{
		{"raw base64 uses fallback mime", "ABC", "image/png", "image/png", "ABC"},
		{"data uri prefers declared type", "data:image/png;base64,ABC123", "image/jpeg", "image/png", "ABC123"},
		{"data uri falls back to mime when type empty", "data:;base64,XYZ", "image/webp", "image/webp", "XYZ"},
		{"data uri with charset param", "data:image/gif;charset=utf-8;base64,GGG", "", "image/gif", "GGG"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			media, data := splitDataURI(tt.source, tt.mime)
			if media != tt.wantMedia {
				t.Errorf("media = %q, want %q", media, tt.wantMedia)
			}
			if data != tt.wantData {
				t.Errorf("data = %q, want %q", data, tt.wantData)
			}
		})
	}
}
