package ollama

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	ollamaapi "github.com/ollama/ollama/api"
	"github.com/sunholo-data/ailang/internal/ai"
)

// b64 encodes raw bytes to standard base64 for building test fixtures.
func b64(raw string) string {
	return base64.StdEncoding.EncodeToString([]byte(raw))
}

// TestToOllamaImages_TwoImages verifies that two image parts produce two
// ImageData entries carrying the decoded bytes, and that the message marshals
// with an "images" array containing the round-tripped base64 strings.
func TestToOllamaImages_TwoImages(t *testing.T) {
	imgA := b64("PNG-BYTES-A")
	imgB := b64("PNG-BYTES-B")

	imgs := toOllamaImages([]ai.ImagePart{
		{Source: imgA, Mime: "image/png"},
		{Source: imgB, Mime: "image/png"},
	})

	if len(imgs) != 2 {
		t.Fatalf("expected 2 images, got %d", len(imgs))
	}
	// ImageData holds decoded bytes; re-encoding recovers the raw base64 source.
	if got := base64.StdEncoding.EncodeToString(imgs[0]); got != imgA {
		t.Errorf("image[0] = %q, want %q", got, imgA)
	}
	if got := base64.StdEncoding.EncodeToString(imgs[1]); got != imgB {
		t.Errorf("image[1] = %q, want %q", got, imgB)
	}

	om := ollamaapi.Message{Role: "user", Content: "look", Images: imgs}
	data, err := json.Marshal(om)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	js := string(data)
	if !strings.Contains(js, `"images":[`) {
		t.Errorf("marshaled message missing images array: %s", js)
	}
	// The wire base64 must be the source base64, not a double-encoding of it.
	if !strings.Contains(js, imgA) || !strings.Contains(js, imgB) {
		t.Errorf("marshaled images missing expected base64 payloads: %s", js)
	}
}

// TestToOllamaImages_DataURIStripped verifies a data-URI source is reduced to
// its raw base64 payload before decoding.
func TestToOllamaImages_DataURIStripped(t *testing.T) {
	payload := b64("JPEG-BYTES")
	src := "data:image/jpeg;base64," + payload

	imgs := toOllamaImages([]ai.ImagePart{{Source: src, Mime: "image/jpeg"}})
	if len(imgs) != 1 {
		t.Fatalf("expected 1 image, got %d", len(imgs))
	}
	if got := base64.StdEncoding.EncodeToString(imgs[0]); got != payload {
		t.Errorf("decoded image = %q, want %q", got, payload)
	}

	data, err := json.Marshal(ollamaapi.Message{Role: "user", Images: imgs})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if js := string(data); strings.Contains(js, "data:") {
		t.Errorf("data-URI prefix leaked into wire payload: %s", js)
	}
}

// TestToOllamaImages_EmptyOmitted is the regression guard: no images means the
// Images field stays nil and the marshaled message omits the images key
// entirely (byte-for-byte identical to a pre-vision text-only message).
func TestToOllamaImages_EmptyOmitted(t *testing.T) {
	if imgs := toOllamaImages(nil); imgs != nil {
		t.Errorf("expected nil for empty input, got %#v", imgs)
	}

	om := ollamaapi.Message{Role: "user", Content: "hello"}
	// Mirror the step.go guard: only set Images when there are images.
	var m ai.Message // zero value has no images
	if len(m.Images) > 0 {
		om.Images = toOllamaImages(m.Images)
	}
	data, err := json.Marshal(om)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if js := string(data); strings.Contains(js, "images") {
		t.Errorf("empty-images message should omit images key, got: %s", js)
	}
}

// TestRawBase64 covers the source-normalization helper directly.
func TestRawBase64(t *testing.T) {
	cases := []struct {
		name, in, want string
	}{
		{"raw passthrough", "AAAA", "AAAA"},
		{"data-uri png", "data:image/png;base64,AAAA", "AAAA"},
		{"data-uri no base64 marker", "data:image/png,AAAA", "AAAA"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := rawBase64(c.in); got != c.want {
				t.Errorf("rawBase64(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}
