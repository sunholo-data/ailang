package gemini

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ai"
)

func TestBuildURLForRequest_VertexEUV1Beta1ForSlicing(t *testing.T) {
	c, err := NewVertexAIClient("test-proj", WithLocation("eu"))
	if err != nil {
		t.Fatalf("NewVertexAIClient: %v", err)
	}

	plainURL, err := c.buildURLForRequest(&ai.Request{Model: "gemini-3.5-flash", UserPrompt: "hi"})
	if err != nil {
		t.Fatalf("plain buildURLForRequest: %v", err)
	}
	wantPlain := "https://aiplatform.eu.rep.googleapis.com/v1/projects/test-proj/locations/eu/publishers/google/models/gemini-3.5-flash:generateContent"
	if plainURL != wantPlain {
		t.Errorf("plain URL = %q, want %q", plainURL, wantPlain)
	}

	slicedURL, err := c.buildURLForRequest(&ai.Request{
		Model:      "gemini-3.5-flash",
		UserPrompt: "extract",
		InputMedia: []ai.MediaPart{{
			Kind:       ai.MediaKindFile,
			Source:     "gs://eu-bucket/report.pdf",
			Mime:       "application/pdf",
			Resolution: "low",
			Options:    `{"page_options":{"start_page":3,"end_page":5},"content_layers":["page_text"]}`,
		}},
	})
	if err != nil {
		t.Fatalf("sliced buildURLForRequest: %v", err)
	}
	wantBeta := "https://aiplatform.eu.rep.googleapis.com/v1beta1/projects/test-proj/locations/eu/publishers/google/models/gemini-3.5-flash:generateContent"
	if slicedURL != wantBeta {
		t.Errorf("sliced URL = %q, want %q", slicedURL, wantBeta)
	}

	streamBetaURL, err := c.buildStreamURLForRequest(&ai.Request{
		Model: "gemini-3.5-flash",
		Messages: []ai.Message{{
			Role:    "user",
			Content: "hi",
			Media: []ai.MediaPart{{
				Kind:       ai.MediaKindImage,
				Source:     "aGVsbG8=",
				Mime:       "image/png",
				Resolution: "high",
			}},
		}},
	})
	if err != nil {
		t.Fatalf("stream buildStreamURLForRequest: %v", err)
	}
	wantStream := "https://aiplatform.eu.rep.googleapis.com/v1beta1/projects/test-proj/locations/eu/publishers/google/models/gemini-3.5-flash:streamGenerateContent?alt=sse"
	if streamBetaURL != wantStream {
		t.Errorf("stream URL = %q, want %q", streamBetaURL, wantStream)
	}
}

func TestUserMediaParts_WireShapesAndGating(t *testing.T) {
	parts, err := userMediaParts(ai.Message{
		Role:    "user",
		Content: "Analyze all three",
		Media: []ai.MediaPart{
			{
				Kind:       ai.MediaKindFile,
				Source:     "gs://eu-bucket/report.pdf",
				Mime:       "application/pdf",
				Resolution: "high",
				Options:    `{"page_options":{"start_page":3,"end_page":5},"content_layers":["page_text","page_image"]}`,
			},
			{
				Kind:    ai.MediaKindFile,
				Source:  "gs://eu-bucket/q3.xlsx",
				Mime:    "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
				Options: `{"sheet_index":0,"visible_cells":"A1:G42","content_layers":["data","formulas","visuals"]}`,
			},
			{
				Kind:    ai.MediaKindFile,
				Source:  "https://generativelanguage.googleapis.com/v1beta/files/clip1",
				Mime:    "video/mp4",
				Options: `{"start_offset":"PT10S","end_offset":"40","fps":2,"content_layers":["transcript","frames","audio"]}`,
			},
			{
				Kind:       ai.MediaKindImage,
				Source:     "data:image/png;base64,iVBORw0KGgo=",
				Mime:       "image/png",
				Resolution: "low",
			},
		},
	}, AuthADC)
	if err != nil {
		t.Fatalf("userMediaParts: %v", err)
	}
	if len(parts) != 5 {
		t.Fatalf("got %d parts, want 5", len(parts))
	}
	if parts[0].Text != "Analyze all three" {
		t.Errorf("parts[0].Text = %q", parts[0].Text)
	}

	// PDF: gs:// -> fileData (never inlineData), documentMetadata, mediaResolution
	pdf := parts[1]
	if pdf.FileData == nil || pdf.FileData.FileUri != "gs://eu-bucket/report.pdf" || pdf.InlineData != nil {
		t.Fatalf("pdf FileData/InlineData mismatch: %+v", pdf)
	}
	if pdf.MediaResolution == nil || pdf.MediaResolution.Level != "MEDIA_RESOLUTION_HIGH" {
		t.Errorf("pdf MediaResolution = %+v", pdf.MediaResolution)
	}
	if pdf.DocumentMetadata == nil || pdf.DocumentMetadata.StartPage != 3 || pdf.DocumentMetadata.EndPage != 5 {
		t.Errorf("pdf DocumentMetadata = %+v", pdf.DocumentMetadata)
	}
	if strings.Join(pdf.DocumentMetadata.ContentLayers, ",") != "PAGE_TEXT,PAGE_IMAGE" {
		t.Errorf("pdf layers = %v", pdf.DocumentMetadata.ContentLayers)
	}

	// Spreadsheet: 0-based sheet_index -> 1-based sheetIndex, visuals -> VISUAL
	sheet := parts[2]
	if sheet.SpreadsheetMetadata == nil || sheet.SpreadsheetMetadata.SheetIndex != 1 || sheet.SpreadsheetMetadata.VisibleCells != "A1:G42" {
		t.Errorf("sheet SpreadsheetMetadata = %+v", sheet.SpreadsheetMetadata)
	}
	if strings.Join(sheet.SpreadsheetMetadata.ContentLayers, ",") != "DATA,FORMULAS,VISUAL" {
		t.Errorf("sheet layers = %v", sheet.SpreadsheetMetadata.ContentLayers)
	}

	// Video: offsets, fps, layers
	vid := parts[3]
	if vid.VideoMetadata == nil || vid.VideoMetadata.StartOffset != "10s" || vid.VideoMetadata.EndOffset != "40s" || vid.VideoMetadata.FPS != 2 {
		t.Errorf("vid VideoMetadata = %+v", vid.VideoMetadata)
	}
	if strings.Join(vid.VideoMetadata.ContentLayers, ",") != "TRANSCRIPT,FRAMES,AUDIO" {
		t.Errorf("vid layers = %v", vid.VideoMetadata.ContentLayers)
	}

	// Image: data-URI stripped to raw base64 + MEDIA_RESOLUTION_LOW
	img := parts[4]
	if img.InlineData == nil || img.InlineData.Data != "iVBORw0KGgo=" || img.InlineData.MimeType != "image/png" {
		t.Errorf("img InlineData = %+v", img.InlineData)
	}
	if img.MediaResolution == nil || img.MediaResolution.Level != "MEDIA_RESOLUTION_LOW" {
		t.Errorf("img MediaResolution = %+v", img.MediaResolution)
	}
}

func TestBuildMediaPart_RejectsUnsupportedCapabilities(t *testing.T) {
	// gs:// on AI Studio API key -> CapabilityNotSupported
	_, err := buildMediaPart(ai.MediaPart{
		Kind:   ai.MediaKindFile,
		Source: "gs://eu-bucket/report.pdf",
		Mime:   "application/pdf",
	}, AuthAPIKey, "media[0]")
	if err == nil {
		t.Fatal("expected error for gs:// on AuthAPIKey")
	}
	if aiErr, ok := err.(*ai.AIError); !ok || aiErr.Code != ai.CodeCapabilityNotSupported {
		t.Errorf("got %#v, want CapabilityNotSupported", err)
	}

	// Audio content_layers -> CapabilityNotSupported (audioMetadata only has offsets)
	_, err = buildMediaPart(ai.MediaPart{
		Kind:    ai.MediaKindFile,
		Source:  "gs://eu-bucket/call.mp3",
		Mime:    "audio/mpeg",
		Options: `{"content_layers":["transcript"]}`,
	}, AuthADC, "media[0]")
	if err == nil {
		t.Fatal("expected error for audio content_layers")
	}
	if aiErr, ok := err.(*ai.AIError); !ok || aiErr.Code != ai.CodeCapabilityNotSupported {
		t.Errorf("got %#v, want CapabilityNotSupported", err)
	}
}

func TestBuildStepRequest_ToolResultMediaAttachedToFunctionResponse(t *testing.T) {
	req := &ai.Request{
		Model: "gemini-3.5-flash",
		Messages: []ai.Message{
			{Role: "user", Content: "Fetch page 2"},
			{
				Role: "assistant",
				ToolCalls: []ai.ToolCall{{
					ID:        "0_0",
					Name:      "fetch_pdf",
					Arguments: `{"doc":"inv"}`,
				}},
			},
			{
				Role:       "tool",
				ToolCallID: "0_0",
				Content:    "fetched",
				Media: []ai.MediaPart{{
					Kind:       ai.MediaKindFile,
					Source:     "gs://eu-bucket/inv.pdf",
					Mime:       "application/pdf",
					Resolution: "medium",
					Options:    `{"page_options":{"start_page":2,"end_page":2}}`,
				}},
			},
		},
	}

	apiReq, err := buildStepRequestWithAuth(req, ai.ReasoningDecision{}, AuthADC)
	if err != nil {
		t.Fatalf("buildStepRequest: %v", err)
	}
	if len(apiReq.Contents) != 3 {
		t.Fatalf("contents len = %d, want 3", len(apiReq.Contents))
	}
	fr := apiReq.Contents[2].Parts[0].FunctionResponse
	if fr == nil || fr.Name != "fetch_pdf" {
		t.Fatalf("expected functionResponse fetch_pdf, got %+v", apiReq.Contents[2].Parts[0])
	}
	if len(fr.Parts) != 1 || fr.Parts[0].FileData == nil || fr.Parts[0].DocumentMetadata == nil {
		t.Fatalf("expected sliced fileData inside functionResponse.parts, got %+v", fr.Parts)
	}
	raw, err := json.Marshal(fr)
	if err != nil {
		t.Fatalf("marshal functionResponse: %v", err)
	}
	if !strings.Contains(string(raw), `"documentMetadata":{"startPage":2,"endPage":2}`) {
		t.Errorf("unexpected functionResponse JSON: %s", string(raw))
	}
}
