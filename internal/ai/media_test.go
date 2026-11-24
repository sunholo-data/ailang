package ai

import (
	"context"
	"strings"
	"testing"
)

func TestValidateMediaPart_ValidClasses(t *testing.T) {
	cases := []struct {
		name string
		part MediaPart
		want func(t *testing.T, p MediaPart)
	}{
		{
			name: "image with resolution",
			part: MediaPart{Kind: MediaKindImage, Source: "aGVsbG8=", Mime: "image/png", Resolution: "high"},
			want: func(t *testing.T, p MediaPart) {
				if p.Parsed != nil {
					t.Errorf("expected nil Parsed for Image, got %+v", p.Parsed)
				}
			},
		},
		{
			name: "document page_options and content_layers",
			part: MediaPart{
				Kind:       MediaKindFile,
				Source:     "gs://eu-bucket/report.pdf",
				Mime:       "application/pdf",
				Resolution: "low",
				Options:    `{"page_options":{"start_page":3,"end_page":5},"content_layers":["page_text","page_image"]}`,
			},
			want: func(t *testing.T, p MediaPart) {
				if p.Parsed == nil || p.Parsed.PageOptions == nil {
					t.Fatalf("expected PageOptions")
				}
				if p.Parsed.PageOptions.StartPage != 3 || p.Parsed.PageOptions.EndPage != 5 {
					t.Errorf("pages = %+v, want 3..5", p.Parsed.PageOptions)
				}
				if len(p.Parsed.ContentLayers) != 2 {
					t.Errorf("layers = %v, want 2", p.Parsed.ContentLayers)
				}
			},
		},
		{
			name: "spreadsheet sheet_index visible_cells layers",
			part: MediaPart{
				Kind:    MediaKindFile,
				Source:  "gs://eu-bucket/q3.xlsx",
				Mime:    "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
				Options: `{"sheet_index":0,"visible_cells":"A1:G42","content_layers":["data","formulas","visuals"]}`,
			},
			want: func(t *testing.T, p MediaPart) {
				if p.Parsed == nil || p.Parsed.SheetIndex == nil || *p.Parsed.SheetIndex != 0 {
					t.Fatalf("SheetIndex = %v, want 0", p.Parsed)
				}
				if p.Parsed.VisibleCells != "A1:G42" {
					t.Errorf("VisibleCells = %q, want A1:G42", p.Parsed.VisibleCells)
				}
			},
		},
		{
			name: "video offsets fps layers",
			part: MediaPart{
				Kind:    MediaKindFile,
				Source:  "gs://eu-bucket/clip.mp4",
				Mime:    "video/mp4",
				Options: `{"start_offset":"PT1M30S","end_offset":"150","fps":2,"content_layers":["transcript","frames","audio"]}`,
			},
			want: func(t *testing.T, p MediaPart) {
				if p.Parsed.StartOffset != "90s" || p.Parsed.EndOffset != "150s" {
					t.Errorf("offsets = %q..%q, want 90s..150s", p.Parsed.StartOffset, p.Parsed.EndOffset)
				}
				if p.Parsed.FPS == nil || *p.Parsed.FPS != 2 {
					t.Errorf("FPS = %v, want 2", p.Parsed.FPS)
				}
			},
		},
		{
			name: "audio offsets",
			part: MediaPart{
				Kind:    MediaKindFile,
				Source:  "gs://eu-bucket/call.mp3",
				Mime:    "audio/mpeg",
				Options: `{"start_offset":"10s","end_offset":"PT45S"}`,
			},
			want: func(t *testing.T, p MediaPart) {
				if p.Parsed.StartOffset != "10s" || p.Parsed.EndOffset != "45s" {
					t.Errorf("offsets = %q..%q, want 10s..45s", p.Parsed.StartOffset, p.Parsed.EndOffset)
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := tc.part
			if err := ValidateMediaPart(&p, "media[0]"); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			tc.want(t, p)
		})
	}
}

func TestValidateMediaPart_RejectsInvalid(t *testing.T) {
	cases := []struct {
		name    string
		part    MediaPart
		wantSub string
	}{
		{
			name:    "bad resolution",
			part:    MediaPart{Kind: MediaKindImage, Source: "abc", Mime: "image/png", Resolution: "ultra"},
			wantSub: "invalid resolution",
		},
		{
			name:    "empty image source",
			part:    MediaPart{Kind: MediaKindImage, Source: "", Mime: "image/png"},
			wantSub: "empty source",
		},
		{
			name:    "raw base64 without mime",
			part:    MediaPart{Kind: MediaKindFile, Source: "abc", Mime: ""},
			wantSub: "requires a mime type",
		},
		{
			name:    "malformed options JSON",
			part:    MediaPart{Kind: MediaKindFile, Source: "gs://b/a.pdf", Mime: "application/pdf", Options: "{bad"},
			wantSub: "invalid options JSON",
		},
		{
			name:    "unknown option key on pdf",
			part:    MediaPart{Kind: MediaKindFile, Source: "gs://b/a.pdf", Mime: "application/pdf", Options: `{"sheet_index":0}`},
			wantSub: "unknown option",
		},
		{
			name:    "start_page < 1",
			part:    MediaPart{Kind: MediaKindFile, Source: "gs://b/a.pdf", Mime: "application/pdf", Options: `{"page_options":{"start_page":0,"end_page":1}}`},
			wantSub: "invalid page range",
		},
		{
			name:    "end_page < start_page",
			part:    MediaPart{Kind: MediaKindFile, Source: "gs://b/a.pdf", Mime: "application/pdf", Options: `{"page_options":{"start_page":5,"end_page":3}}`},
			wantSub: "invalid page range",
		},
		{
			name:    "wrong content_layers for spreadsheet",
			part:    MediaPart{Kind: MediaKindFile, Source: "gs://b/s.xlsx", Mime: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", Options: `{"content_layers":["page_text"]}`},
			wantSub: "invalid content_layers",
		},
		{
			name:    "invalid A1 range",
			part:    MediaPart{Kind: MediaKindFile, Source: "gs://b/s.xlsx", Mime: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", Options: `{"visible_cells":"not-a-range"}`},
			wantSub: "invalid visible_cells",
		},
		{
			name:    "negative sheet_index",
			part:    MediaPart{Kind: MediaKindFile, Source: "gs://b/s.xlsx", Mime: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", Options: `{"sheet_index":-1}`},
			wantSub: "sheet_index must be >= 0",
		},
		{
			name:    "fps on audio",
			part:    MediaPart{Kind: MediaKindFile, Source: "gs://b/a.mp3", Mime: "audio/mpeg", Options: `{"fps":1}`},
			wantSub: "only valid for video",
		},
		{
			name:    "end_offset before start_offset",
			part:    MediaPart{Kind: MediaKindFile, Source: "gs://b/v.mp4", Mime: "video/mp4", Options: `{"start_offset":"60","end_offset":"30"}`},
			wantSub: "end_offset",
		},
		{
			name:    "non-positive fps",
			part:    MediaPart{Kind: MediaKindFile, Source: "gs://b/v.mp4", Mime: "video/mp4", Options: `{"fps":0}`},
			wantSub: "fps must be > 0",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := tc.part
			err := ValidateMediaPart(&p, "media[0]")
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tc.wantSub)
			}
			aiErr, ok := err.(*AIError)
			if !ok {
				t.Fatalf("expected *AIError, got %T: %v", err, err)
			}
			if aiErr.Code != CodeSchemaValidation || aiErr.Retryable {
				t.Errorf("got code=%q retryable=%v, want SchemaValidation/false", aiErr.Code, aiErr.Retryable)
			}
			if !strings.Contains(aiErr.Message, tc.wantSub) {
				t.Errorf("message %q does not contain %q", aiErr.Message, tc.wantSub)
			}
		})
	}
}

func TestParseMediaCallOptions(t *testing.T) {
	if opts, err := ParseMediaCallOptions(""); err != nil || opts.Model != "" {
		t.Fatalf("empty options: got %+v, %v", opts, err)
	}
	if opts, err := ParseMediaCallOptions(`{"model":"gemini-3.5-flash"}`); err != nil || opts.Model != "gemini-3.5-flash" {
		t.Fatalf("model override: got %+v, %v", opts, err)
	}
	if _, err := ParseMediaCallOptions(`{"page_options":{"start_page":1,"end_page":1}}`); err == nil {
		t.Fatal("expected SchemaValidation for unknown call-level option")
	}
}

type mockMediaProvider struct {
	gotReq *Request
	resp   *Response
}

func (m *mockMediaProvider) Generate(_ context.Context, req *Request) (*Response, error) {
	m.gotReq = req
	return m.resp, nil
}
func (m *mockMediaProvider) Step(_ context.Context, req *Request) (*Response, error) {
	m.gotReq = req
	return m.resp, nil
}
func (m *mockMediaProvider) Name() string { return "mock" }

func TestHandlerCallMediaResult(t *testing.T) {
	prov := &mockMediaProvider{
		resp: &Response{Text: "totals: 42", InputTokens: 210, OutputTokens: 12},
	}
	h := NewHandler(prov, "default-model")
	resp, err := h.CallMediaResult(
		"extract totals",
		[]MediaPart{{
			Kind:       MediaKindFile,
			Source:     "gs://eu-bucket/inv.pdf",
			Mime:       "application/pdf",
			Resolution: "low",
			Options:    `{"page_options":{"start_page":1,"end_page":2}}`,
		}},
		`{"model":"override-model"}`,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if prov.gotReq.Model != "override-model" {
		t.Errorf("model = %q, want override-model", prov.gotReq.Model)
	}
	if len(prov.gotReq.InputMedia) != 1 || prov.gotReq.InputMedia[0].Parsed == nil {
		t.Fatalf("expected validated InputMedia, got %+v", prov.gotReq.InputMedia)
	}
	if resp.Text != "totals: 42" || resp.InputTokens != 210 || resp.OutputTokens != 12 {
		t.Errorf("resp = %+v", resp)
	}
}

func TestExtractLegacyImages_FailClosed(t *testing.T) {
	// Default-resolution image passes.
	imgs, err := ExtractLegacyImages("openai", Message{
		Role: "user",
		Media: []MediaPart{{
			Kind:   MediaKindImage,
			Source: "aGVsbG8=",
			Mime:   "image/png",
		}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(imgs) != 1 || imgs[0].Source != "aGVsbG8=" {
		t.Fatalf("imgs = %+v", imgs)
	}

	// File part fails closed with CapabilityNotSupported.
	if _, err := ExtractLegacyImages("anthropic", Message{
		Role: "user",
		Media: []MediaPart{{
			Kind:   MediaKindFile,
			Source: "gs://b/a.pdf",
			Mime:   "application/pdf",
		}},
	}); err == nil || err.Code != CodeCapabilityNotSupported {
		t.Errorf("File part: got %#v, want CapabilityNotSupported", err)
	}

	// Explicit resolution fails closed.
	if _, err := ExtractLegacyImages("ollama", Message{
		Role: "user",
		Media: []MediaPart{{
			Kind:       MediaKindImage,
			Source:     "aGVsbG8=",
			Mime:       "image/png",
			Resolution: "low",
		}},
	}); err == nil || err.Code != CodeCapabilityNotSupported {
		t.Errorf("Resolution: got %#v, want CapabilityNotSupported", err)
	}

	// Tool-role media fails closed.
	if _, err := ExtractLegacyImages("openai", Message{
		Role: "tool",
		Media: []MediaPart{{
			Kind:   MediaKindImage,
			Source: "aGVsbG8=",
			Mime:   "image/png",
		}},
	}); err == nil || err.Code != CodeCapabilityNotSupported {
		t.Errorf("Tool-role media: got %#v, want CapabilityNotSupported", err)
	}

	// RejectUnsupportedInputMedia fails closed.
	if err := RejectUnsupportedInputMedia("anthropic", &Request{
		InputMedia: []MediaPart{{Kind: MediaKindImage, Source: "a", Mime: "image/png"}},
	}); err == nil {
		t.Error("expected error from RejectUnsupportedInputMedia")
	}
}
