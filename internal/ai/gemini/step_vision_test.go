package gemini

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ai"
)

// findUserContent returns the first content entry with role "user" from a
// built generateRequest, failing the test if none exists.
func findUserContent(t *testing.T, req *generateRequest) content {
	t.Helper()
	for _, c := range req.Contents {
		if c.Role == "user" {
			return c
		}
	}
	t.Fatalf("no user content in request: %+v", req.Contents)
	return content{}
}

// TestBuildStepRequest_UserImage_RawBase64: a user message with text plus one
// raw-base64 image produces a Text part followed by an InlineData part, and the
// marshaled JSON carries inlineData with mimeType and data.
func TestBuildStepRequest_UserImage_RawBase64(t *testing.T) {
	req, err := buildStepRequest(&ai.Request{
		Model: "gemini-3-flash",
		Messages: []ai.Message{
			{
				Role:    "user",
				Content: "What is in this image?",
				Images: []ai.ImagePart{
					{Source: "iVBORw0KGgoAAAA", Mime: "image/png"},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("buildStepRequest error = %v", err)
	}

	uc := findUserContent(t, req)
	if len(uc.Parts) != 2 {
		t.Fatalf("Parts len = %d, want 2 (text + image); parts = %+v", len(uc.Parts), uc.Parts)
	}
	if uc.Parts[0].Text != "What is in this image?" {
		t.Errorf("Parts[0].Text = %q, want the user text", uc.Parts[0].Text)
	}
	if uc.Parts[0].InlineData != nil {
		t.Errorf("Parts[0].InlineData = %+v, want nil (text part)", uc.Parts[0].InlineData)
	}
	if uc.Parts[1].InlineData == nil {
		t.Fatal("Parts[1].InlineData is nil, want the image part")
	}
	if uc.Parts[1].InlineData.MimeType != "image/png" {
		t.Errorf("InlineData.MimeType = %q, want image/png", uc.Parts[1].InlineData.MimeType)
	}
	if uc.Parts[1].InlineData.Data != "iVBORw0KGgoAAAA" {
		t.Errorf("InlineData.Data = %q, want raw base64", uc.Parts[1].InlineData.Data)
	}

	// Marshal and confirm the wire shape carries the inline-data object with a
	// mime type and data payload (RAW base64, no data-URI prefix).
	raw, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal error = %v", err)
	}
	js := string(raw)
	if !strings.Contains(js, `"inlineData"`) {
		t.Errorf("JSON missing inlineData object: %s", js)
	}
	if !strings.Contains(js, `"mimeType":"image/png"`) {
		t.Errorf("JSON missing mimeType: %s", js)
	}
	if !strings.Contains(js, `"data":"iVBORw0KGgoAAAA"`) {
		t.Errorf("JSON missing data payload: %s", js)
	}
	if strings.Contains(js, "data:image/png;base64,") {
		t.Errorf("JSON leaked a data-URI prefix; Gemini wants raw base64: %s", js)
	}
}

// TestBuildStepRequest_UserImage_DataURI: a data-URI source is split into raw
// base64 with the mime type derived from the URI prefix (not img.Mime).
func TestBuildStepRequest_UserImage_DataURI(t *testing.T) {
	req, err := buildStepRequest(&ai.Request{
		Model: "gemini-3-flash",
		Messages: []ai.Message{
			{
				Role:    "user",
				Content: "", // no text — image only
				Images: []ai.ImagePart{
					{Source: "data:image/jpeg;base64,/9j/4AAQSkZJRg==", Mime: "image/png"},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("buildStepRequest error = %v", err)
	}

	uc := findUserContent(t, req)
	if len(uc.Parts) != 1 {
		t.Fatalf("Parts len = %d, want 1 (image only, empty text elided); parts = %+v", len(uc.Parts), uc.Parts)
	}
	id := uc.Parts[0].InlineData
	if id == nil {
		t.Fatal("Parts[0].InlineData is nil, want the image part")
	}
	// Mime derived from the data-URI prefix, overriding img.Mime ("image/png").
	if id.MimeType != "image/jpeg" {
		t.Errorf("InlineData.MimeType = %q, want image/jpeg (from data-URI prefix)", id.MimeType)
	}
	if id.Data != "/9j/4AAQSkZJRg==" {
		t.Errorf("InlineData.Data = %q, want raw base64 with prefix stripped", id.Data)
	}
}

// TestBuildStepRequest_UserNoImages_Unchanged: a text-only user message (no
// Images) yields exactly one Text part with no InlineData — wire-identical to
// pre-vision behavior (regression guard).
func TestBuildStepRequest_UserNoImages_Unchanged(t *testing.T) {
	req, err := buildStepRequest(&ai.Request{
		Model: "gemini-3-flash",
		Messages: []ai.Message{
			{Role: "user", Content: "hello"},
		},
	})
	if err != nil {
		t.Fatalf("buildStepRequest error = %v", err)
	}

	uc := findUserContent(t, req)
	if len(uc.Parts) != 1 {
		t.Fatalf("Parts len = %d, want exactly 1 text part; parts = %+v", len(uc.Parts), uc.Parts)
	}
	if uc.Parts[0].Text != "hello" {
		t.Errorf("Parts[0].Text = %q, want hello", uc.Parts[0].Text)
	}
	if uc.Parts[0].InlineData != nil {
		t.Errorf("Parts[0].InlineData = %+v, want nil for text-only message", uc.Parts[0].InlineData)
	}
}

// TestSplitDataURI covers the mime/data derivation branches directly.
func TestSplitDataURI(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		mime     string
		wantMime string
		wantData string
	}{
		{
			name:     "raw base64 uses provided mime",
			source:   "AAAABBBB",
			mime:     "image/png",
			wantMime: "image/png",
			wantData: "AAAABBBB",
		},
		{
			name:     "data-URI mime overrides provided mime",
			source:   "data:image/jpeg;base64,/9j/AAAA",
			mime:     "image/png",
			wantMime: "image/jpeg",
			wantData: "/9j/AAAA",
		},
		{
			name:     "data-URI with no declared mime falls back to provided",
			source:   "data:;base64,ZZZZ",
			mime:     "image/webp",
			wantMime: "image/webp",
			wantData: "ZZZZ",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotMime, gotData := splitDataURI(tt.source, tt.mime)
			if gotMime != tt.wantMime {
				t.Errorf("mimeType = %q, want %q", gotMime, tt.wantMime)
			}
			if gotData != tt.wantData {
				t.Errorf("data = %q, want %q", gotData, tt.wantData)
			}
		})
	}
}
