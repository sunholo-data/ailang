package ai

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Per-call image model (#1496/#1500) and reference images (#1496) at the
// Handler layer, against stubProvider (handler_routing_test.go).

func imageStub() *stubProvider {
	return &stubProvider{resp: &Response{ImageData: []byte("img"), ImageMIME: "image/png"}}
}

func TestParseImageOptions_Model(t *testing.T) {
	opts := ParseImageOptions(`{"model":"google/gemini-3-pro-image","aspect_ratio":"1:1"}`)
	if opts == nil || opts.Model != "google/gemini-3-pro-image" || opts.AspectRatio != "1:1" {
		t.Fatalf("ParseImageOptions = %+v", opts)
	}
	if got := ParseImageOptions(`{"model":"m"}`); got == nil || got.Model != "m" {
		t.Errorf("model-only options = %+v, want Model m", got)
	}
	if got := ParseImageOptions(`{}`); got != nil {
		t.Errorf("{} = %+v, want nil", got)
	}
}

func TestHandler_CallImage_PerCallModelOverride(t *testing.T) {
	p := imageStub()
	h := NewHandler(p, "z-ai/glm-5.3")
	out := filepath.Join(t.TempDir(), "a.png")
	if _, err := h.CallImage("portrait", out, `{"model":"google/gemini-3-pro-image"}`); err != nil {
		t.Fatalf("CallImage: %v", err)
	}
	if p.lastReq.Model != "google/gemini-3-pro-image" {
		t.Errorf("Model = %q, want the per-call override", p.lastReq.Model)
	}
	if b, _ := os.ReadFile(out); string(b) != "img" {
		t.Errorf("file = %q", b)
	}

	// The override is call-local: the next call without one uses the bound model.
	if _, err := h.CallImageBase64("portrait", `{}`); err != nil {
		t.Fatalf("CallImageBase64: %v", err)
	}
	if p.lastReq.Model != "z-ai/glm-5.3" {
		t.Errorf("Model = %q, want bound model after override call", p.lastReq.Model)
	}
}

func TestHandler_CallImageBase64_PerCallModelOverride(t *testing.T) {
	p := imageStub()
	h := NewHandler(p, "bound")
	if _, err := h.CallImageBase64("x", `{"model":"other","mime_type":"image/png"}`); err != nil {
		t.Fatalf("CallImageBase64: %v", err)
	}
	if p.lastReq.Model != "other" {
		t.Errorf("Model = %q, want other", p.lastReq.Model)
	}
	if p.lastReq.ImageOptions == nil || p.lastReq.ImageOptions.MIMEType != "image/png" {
		t.Errorf("ImageOptions = %+v", p.lastReq.ImageOptions)
	}
}

func TestHandler_CallImageWithRefs_ForwardsReferences(t *testing.T) {
	p := imageStub()
	h := NewHandler(p, "gemini-2.5-flash-image")
	refs := []ImagePart{{Source: "cmVm", Mime: "image/png"}, {Source: "data:image/jpeg;base64,eA==", Mime: ""}}
	got, err := h.CallImageBase64WithRefs("same person", refs, `{}`)
	if err != nil {
		t.Fatalf("CallImageBase64WithRefs: %v", err)
	}
	if !strings.Contains(got, `"mime_type":"image/png"`) {
		t.Errorf("result = %s", got)
	}
	if len(p.lastReq.InputImages) != 2 || p.lastReq.InputImages[0].Source != "cmVm" {
		t.Errorf("InputImages = %+v", p.lastReq.InputImages)
	}
	if !RequestsImage(p.lastReq) {
		t.Error("request does not ask for an image")
	}

	out := filepath.Join(t.TempDir(), "sub", "b.png")
	if _, err := h.CallImageWithRefs("same person", refs, out, `{"model":"gemini-3-pro-image"}`); err != nil {
		t.Fatalf("CallImageWithRefs: %v", err)
	}
	if p.lastReq.Model != "gemini-3-pro-image" || len(p.lastReq.InputImages) != 2 {
		t.Errorf("req = model %q, %d refs", p.lastReq.Model, len(p.lastReq.InputImages))
	}
}

func TestHandler_CallImageWithRefs_InvalidReferenceFailsBeforeDispatch(t *testing.T) {
	cases := []struct {
		name string
		ref  ImagePart
		want string
	}{
		{"empty source", ImagePart{Source: "", Mime: "image/png"}, "empty source"},
		{"raw base64 without mime", ImagePart{Source: "cmVm"}, "mime"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := imageStub()
			h := NewHandler(p, "m")
			_, err := h.CallImageBase64WithRefs("x", []ImagePart{tc.ref}, `{}`)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err = %v, want contains %q", err, tc.want)
			}
			if p.lastReq != nil {
				t.Error("provider was called despite an invalid reference")
			}
		})
	}
}

func TestSetImageOptionsModel_PreservesOtherKeys(t *testing.T) {
	got, err := SetImageOptionsModel(`{"aspect_ratio":"16:9","model":"friendly","x_custom":{"a":1}}`, "api-name")
	if err != nil {
		t.Fatalf("SetImageOptionsModel: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(got), &m); err != nil {
		t.Fatalf("result not JSON: %s", got)
	}
	if m["model"] != "api-name" || m["aspect_ratio"] != "16:9" || m["x_custom"] == nil {
		t.Errorf("rewritten options = %s", got)
	}
	if _, err := SetImageOptionsModel(`not json`, "m"); err == nil {
		t.Error("malformed options: want error")
	}
}
