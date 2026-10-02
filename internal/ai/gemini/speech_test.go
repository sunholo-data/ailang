package gemini

import (
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

func ttsServer(t *testing.T, mime string, pcm []byte, seen *map[string]any, path *string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*path = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(body, seen); err != nil {
			t.Errorf("request body is not JSON: %v", err)
		}
		if r.Header.Get("x-goog-api-key") != "k" {
			t.Errorf("missing api key header")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{"inlineData":{"mimeType":"` + mime +
			`","data":"` + base64.StdEncoding.EncodeToString(pcm) + `"}}]},"finishReason":"STOP"}]}`))
	}))
}

// #1495: the Gemini TTS request carries AUDIO modality, the voice and the
// style folded into the prompt; the PCM comes back byte-for-byte.
func TestClient_Speech(t *testing.T) {
	pcm := []byte{0x01, 0x00, 0xff, 0x7f, 0x00, 0x80}
	var seen map[string]any
	var path string
	srv := ttsServer(t, "audio/L16;codec=pcm;rate=24000", pcm, &seen, &path)
	defer srv.Close()

	c := NewClient("k", WithBaseURL(srv.URL))
	got, err := c.Speech(context.Background(), &ai.SpeechRequest{
		Model: "gemini-2.5-flash", Text: "Well met.", Voice: "Kore", Style: "Say gruffly", LanguageCode: "en-GB",
	})
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(pcm) {
		t.Fatalf("pcm = %v, want %v", got, pcm)
	}
	// A text model is not a TTS model: the documented default is used.
	if !strings.Contains(path, ai.DefaultGeminiTTSModel+":generateContent") {
		t.Errorf("path = %s, want the default TTS model", path)
	}
	raw, _ := json.Marshal(seen)
	for _, want := range []string{`"responseModalities":["AUDIO"]`, `"voiceName":"Kore"`, `"text":"Say gruffly: Well met."`, `"languageCode":"en-GB"`} {
		if !strings.Contains(string(raw), want) {
			t.Errorf("request missing %s: %s", want, raw)
		}
	}

	// A TTS model in the request is used as-is.
	if _, err := c.Speech(context.Background(), &ai.SpeechRequest{Model: "gemini-2.5-pro-preview-tts", Text: "x"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(path, "gemini-2.5-pro-preview-tts:generateContent") {
		t.Errorf("path = %s", path)
	}
	raw, _ = json.Marshal(seen)
	if strings.Contains(string(raw), "voiceConfig") || strings.Contains(string(raw), "Say") {
		t.Errorf("empty voice/style must not be sent: %s", raw)
	}
}

// Anything other than 24 kHz 16-bit PCM is refused as a ProtocolError.
func TestClient_Speech_RejectsWrongFormat(t *testing.T) {
	for name, tc := range map[string]struct {
		mime string
		pcm  []byte
	}{
		"wrong rate":  {"audio/L16;codec=pcm;rate=16000", []byte{0, 0}},
		"not pcm":     {"audio/mpeg", []byte{0, 0}},
		"odd length":  {"audio/L16;rate=24000", []byte{0, 0, 0}},
		"empty audio": {"audio/L16;rate=24000", nil},
	} {
		var seen map[string]any
		var path string
		srv := ttsServer(t, tc.mime, tc.pcm, &seen, &path)
		_, err := NewClient("k", WithBaseURL(srv.URL)).Speech(context.Background(), &ai.SpeechRequest{Text: "x"})
		srv.Close()
		var aiErr *ai.AIError
		if !errors.As(err, &aiErr) || aiErr.Code != ai.CodeProtocolError {
			t.Errorf("%s: want ProtocolError, got %v", name, err)
		}
	}
}

// The gemini client satisfies the optional speech capability.
var _ ai.SpeechProvider = (*Client)(nil)
