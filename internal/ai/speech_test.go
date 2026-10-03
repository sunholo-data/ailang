package ai

import (
	"context"
	"errors"
	"testing"
)

type speechMock struct {
	mockProvider
	last *SpeechRequest
}

func (s *speechMock) Speech(_ context.Context, req *SpeechRequest) ([]byte, error) {
	s.last = req
	return []byte{1, 0, 2, 0}, nil
}

// #1495: a provider without SpeechProvider fails loudly with
// CapabilityNotSupported; options are strict; the bound model is the
// default and options.model overrides it.
func TestHandler_CallSpeech(t *testing.T) {
	plain := NewHandler(&mockProvider{name: "anthropic"}, "claude-x")
	_, err := plain.CallSpeech("hi", "Kore", "", "")
	var aiErr *AIError
	if !errors.As(err, &aiErr) || aiErr.Code != CodeCapabilityNotSupported || aiErr.Retryable {
		t.Fatalf("non-speech provider: want CapabilityNotSupported, got %v", err)
	}

	sp := &speechMock{mockProvider: mockProvider{name: "gemini"}}
	h := NewHandler(sp, "gemini-2.5-flash")
	pcm, err := h.CallSpeech("Hello", "Kore", "Say warmly", "")
	if err != nil || len(pcm) != 4 {
		t.Fatalf("speech: %v %v", pcm, err)
	}
	if sp.last.Model != "gemini-2.5-flash" || sp.last.Voice != "Kore" || sp.last.Style != "Say warmly" || sp.last.Text != "Hello" {
		t.Fatalf("request: %+v", sp.last)
	}
	if _, err := h.CallSpeech("Hello", "Kore", "", `{"model":"gemini-2.5-pro-preview-tts","language_code":"en-GB"}`); err != nil {
		t.Fatal(err)
	}
	if sp.last.Model != "gemini-2.5-pro-preview-tts" || sp.last.LanguageCode != "en-GB" {
		t.Fatalf("options not applied: %+v", sp.last)
	}

	for name, opts := range map[string]string{"unknown key": `{"sample_rate":16000}`, "not json": `{`} {
		if _, err := h.CallSpeech("Hello", "Kore", "", opts); !errors.As(err, &aiErr) || aiErr.Code != CodeSchemaValidation {
			t.Errorf("%s: want SchemaValidation, got %v", name, err)
		}
	}
	if _, err := h.CallSpeech("  ", "Kore", "", ""); !errors.As(err, &aiErr) || aiErr.Code != CodeSchemaValidation {
		t.Errorf("empty text: want SchemaValidation, got %v", err)
	}
}
