package openrouter

// M-OPENROUTER-IMAGE-OUTPUT (#1500) + reference-conditioned image generation
// (#1496): wire-shape and decode tests for image requests on the Generate
// path. Every test drives client.Generate against an httptest server and
// inspects the captured request bytes — no network, no paid calls.

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ai"
)

// pngBytes is an arbitrary byte payload standing in for an image; the adapter
// never inspects image content, only the data-URL framing.
var pngBytes = []byte("\x89PNG\r\n\x1a\nfake-image-bytes")

func pngDataURL() string {
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(pngBytes)
}

// imageServer captures the request body and answers with the given JSON.
type imageServer struct {
	body  []byte
	calls int
	reply string
}

func (s *imageServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.calls++
	s.body, _ = io.ReadAll(r.Body)
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(s.reply))
}

func imageReply(content string, urls ...string) string {
	imgs := make([]map[string]any, 0, len(urls))
	for _, u := range urls {
		imgs = append(imgs, map[string]any{"type": "image_url", "image_url": map[string]any{"url": u}})
	}
	b, _ := json.Marshal(map[string]any{
		"id":    "gen-1",
		"model": "google/gemini-3-pro-image",
		"choices": []map[string]any{{
			"index":         0,
			"finish_reason": "stop",
			"message":       map[string]any{"role": "assistant", "content": content, "images": imgs},
		}},
		"usage": map[string]any{"prompt_tokens": 10, "completion_tokens": 20, "total_tokens": 30, "cost": 0.04},
	})
	return string(b)
}

func runImage(t *testing.T, srv *imageServer, req *ai.Request) (*ai.Response, error) {
	t.Helper()
	server := httptest.NewServer(srv)
	defer server.Close()
	client := NewClient("test-key", WithBaseURL(server.URL))
	return client.Generate(context.Background(), req)
}

func imageRequest(model string) *ai.Request {
	return &ai.Request{
		Model:              model,
		UserPrompt:         "portrait of the navigator",
		ResponseModalities: []string{"IMAGE"},
	}
}

// AC-1: an image request reaches the server and carries lowercase modalities.
func TestGenerate_ImageRequest_SendsModalities(t *testing.T) {
	srv := &imageServer{reply: imageReply("", pngDataURL())}
	if _, err := runImage(t, srv, imageRequest("google/gemini-3-pro-image")); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if srv.calls != 1 {
		t.Fatalf("server calls = %d, want 1", srv.calls)
	}
	want := `{"model":"google/gemini-3-pro-image","messages":[{"role":"user","content":"portrait of the navigator"}],"max_tokens":4096,"modalities":["image","text"]}`
	if string(srv.body) != want {
		t.Errorf("wire body mismatch\n got: %s\nwant: %s", srv.body, want)
	}
}

// AC-2: a data-URL image decodes into ImageData/ImageMIME; caption and cost survive.
func TestGenerate_ImageResponse_DecodesDataURL(t *testing.T) {
	srv := &imageServer{reply: imageReply("here is your portrait", pngDataURL())}
	resp, err := runImage(t, srv, imageRequest("google/gemini-3-pro-image"))
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if !bytes.Equal(resp.ImageData, pngBytes) {
		t.Errorf("ImageData = %q, want %q", resp.ImageData, pngBytes)
	}
	if resp.ImageMIME != "image/png" {
		t.Errorf("ImageMIME = %q, want image/png", resp.ImageMIME)
	}
	if resp.Text != "here is your portrait" {
		t.Errorf("Text = %q", resp.Text)
	}
	if resp.CostUSD != "0.04" {
		t.Errorf("CostUSD = %q, want 0.04", resp.CostUSD)
	}
	if resp.RequestedModel != "google/gemini-3-pro-image" {
		t.Errorf("RequestedModel = %q", resp.RequestedModel)
	}
}

// First image wins when the model returns several.
func TestGenerate_ImageResponse_FirstImageWins(t *testing.T) {
	second := "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString([]byte("second"))
	srv := &imageServer{reply: imageReply("", pngDataURL(), second)}
	resp, err := runImage(t, srv, imageRequest("m"))
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if !bytes.Equal(resp.ImageData, pngBytes) || resp.ImageMIME != "image/png" {
		t.Errorf("got %q (%s), want first image", resp.ImageData, resp.ImageMIME)
	}
}

// AC-3: a text-only request keeps the pre-change wire body exactly.
func TestGenerate_TextRequest_NoImageKeys(t *testing.T) {
	srv := &imageServer{reply: imageReply("hi")}
	_, err := runImage(t, srv, &ai.Request{Model: "z-ai/glm-5.3", UserPrompt: "hello"})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	want := `{"model":"z-ai/glm-5.3","messages":[{"role":"user","content":"hello"}],"max_tokens":4096}`
	if string(srv.body) != want {
		t.Errorf("text body changed\n got: %s\nwant: %s", srv.body, want)
	}
}

// AC-4: aspect_ratio and mime_type map onto image_config.
func TestGenerate_ImageOptions_MapToImageConfig(t *testing.T) {
	srv := &imageServer{reply: imageReply("", pngDataURL())}
	req := imageRequest("m")
	req.ImageOptions = &ai.ImageOptions{AspectRatio: "16:9", MIMEType: "image/jpeg"}
	if _, err := runImage(t, srv, req); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if !strings.Contains(string(srv.body), `"image_config":{"aspect_ratio":"16:9","output_format":"jpeg"}`) {
		t.Errorf("image_config missing/wrong: %s", srv.body)
	}
}

func TestGenerate_ImageOptions_UnsupportedMIMEFailsBeforeDispatch(t *testing.T) {
	srv := &imageServer{reply: imageReply("", pngDataURL())}
	req := imageRequest("m")
	req.ImageOptions = &ai.ImageOptions{MIMEType: "image/tiff"}
	_, err := runImage(t, srv, req)
	if err == nil || !strings.Contains(err.Error(), "image/tiff") {
		t.Fatalf("err = %v, want unsupported mime_type error", err)
	}
	if srv.calls != 0 {
		t.Errorf("server called %d times, want 0", srv.calls)
	}
}

// AC-5: decode failures are typed ProviderErrors naming the model.
func TestGenerate_ImageResponse_Errors(t *testing.T) {
	cases := []struct {
		name  string
		reply string
		want  string
	}{
		{"no images", imageReply("sorry, text only"), "returned no image"},
		{"remote url", imageReply("", "https://cdn.example/x.png"), "remote image URLs are not supported"},
		{"bad base64", imageReply("", "data:image/png;base64,!!!notbase64"), "decode"},
		{"not base64 data url", imageReply("", "data:image/png,rawbytes"), "base64"},
		{"no choices", `{"choices":[],"usage":{}}`, "no choices"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := &imageServer{reply: tc.reply}
			_, err := runImage(t, srv, imageRequest("acme/image-model"))
			if err == nil {
				t.Fatal("expected error")
			}
			var pe *ai.ProviderError
			if !errors.As(err, &pe) {
				t.Fatalf("err type = %T (%v), want *ai.ProviderError", err, err)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("err = %q, want contains %q", err.Error(), tc.want)
			}
			if tc.name != "no choices" && !strings.Contains(err.Error(), "acme/image-model") {
				t.Errorf("err = %q, want it to name the model", err.Error())
			}
		})
	}
}

// #1496: reference images ride as image_url content parts after the text part.
func TestGenerate_ImageRequest_ReferenceImagesAsContentParts(t *testing.T) {
	srv := &imageServer{reply: imageReply("", pngDataURL())}
	req := imageRequest("google/gemini-3-pro-image")
	req.InputImages = []ai.ImagePart{
		{Source: base64.StdEncoding.EncodeToString([]byte("ref1")), Mime: "image/png"},
		{Source: "data:image/jpeg;base64,cmVmMg==", Mime: ""},
	}
	if _, err := runImage(t, srv, req); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	var body struct {
		Model      string   `json:"model"`
		Modalities []string `json:"modalities"`
		Messages   []struct {
			Role    string          `json:"role"`
			Content json.RawMessage `json:"content"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(srv.body, &body); err != nil {
		t.Fatalf("decode body: %v (%s)", err, srv.body)
	}
	if len(body.Modalities) != 2 || body.Modalities[0] != "image" {
		t.Errorf("modalities = %v", body.Modalities)
	}
	if len(body.Messages) != 1 || body.Messages[0].Role != "user" {
		t.Fatalf("messages = %s", srv.body)
	}
	var parts []visionContentPart
	if err := json.Unmarshal(body.Messages[0].Content, &parts); err != nil {
		t.Fatalf("user content is not a parts array: %s", body.Messages[0].Content)
	}
	if len(parts) != 3 {
		t.Fatalf("parts = %d, want 3 (text + 2 images): %s", len(parts), body.Messages[0].Content)
	}
	if parts[0].Type != "text" || parts[0].Text != "portrait of the navigator" {
		t.Errorf("parts[0] = %+v", parts[0])
	}
	if parts[1].Type != "image_url" || parts[1].ImageURL == nil || parts[1].ImageURL.URL != "data:image/png;base64,cmVmMQ==" {
		t.Errorf("parts[1] = %+v", parts[1])
	}
	if parts[2].ImageURL == nil || parts[2].ImageURL.URL != "data:image/jpeg;base64,cmVmMg==" {
		t.Errorf("parts[2] = %+v", parts[2])
	}
}

// The system prompt stays a plain string message ahead of the parts message.
func TestGenerate_ImageRequest_ReferenceImagesKeepSystemPrompt(t *testing.T) {
	srv := &imageServer{reply: imageReply("", pngDataURL())}
	req := imageRequest("m")
	req.SystemPrompt = "you paint portraits"
	req.InputImages = []ai.ImagePart{{Source: "cmVm", Mime: "image/png"}}
	if _, err := runImage(t, srv, req); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if !strings.Contains(string(srv.body), `{"role":"system","content":"you paint portraits"}`) {
		t.Errorf("system message missing: %s", srv.body)
	}
}

// Step and StreamStep reject image output loudly before dispatch.
func TestStep_RejectsImageModalities(t *testing.T) {
	srv := &imageServer{reply: imageReply("x")}
	server := httptest.NewServer(srv)
	defer server.Close()
	client := NewClient("test-key", WithBaseURL(server.URL))
	req := &ai.Request{
		Model:              "m",
		Messages:           []ai.Message{{Role: "user", Content: "draw"}},
		ResponseModalities: []string{"IMAGE"},
	}
	if _, err := client.Step(context.Background(), req); err == nil || !strings.Contains(err.Error(), "callImage") {
		t.Errorf("Step err = %v, want image rejection pointing at callImage", err)
	}
	if _, err := client.StreamStep(context.Background(), req, nil); err == nil || !strings.Contains(err.Error(), "callImage") {
		t.Errorf("StreamStep err = %v, want image rejection pointing at callImage", err)
	}
	if srv.calls != 0 {
		t.Errorf("server called %d times, want 0", srv.calls)
	}
}
