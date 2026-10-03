package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// Text-to-speech (#1495): std/ai.callSpeech. The output format is FIXED, not
// negotiated: raw PCM, signed 16-bit little-endian, mono, 24 kHz — what
// Gemini TTS emits and what std/audio (wavFromPcm, encode) consumes. A
// provider that returns anything else fails with CodeProtocolError rather
// than handing the program bytes it would misinterpret.
const (
	SpeechSampleRate    = 24000
	SpeechChannels      = 1
	SpeechBitsPerSample = 16

	// DefaultGeminiTTSModel is the TTS model used when neither the call's
	// options nor the bound --ai model name a TTS model (a name containing
	// "tts"). The bound model is usually a text model, which cannot speak.
	DefaultGeminiTTSModel = "gemini-2.5-flash-preview-tts"
)

// SpeechRequest is one text-to-speech call.
type SpeechRequest struct {
	Model        string // provider model; "" lets the provider pick its TTS default
	Text         string // what to say
	Voice        string // provider voice name (Gemini prebuilt voice, e.g. "Kore"); "" = provider default
	Style        string // natural-language delivery instruction, e.g. "Say cheerfully"; "" = none
	LanguageCode string // BCP-47, optional
}

// SpeechProvider is implemented by providers that can synthesise speech.
// Speech returns s16le mono PCM at SpeechSampleRate.
type SpeechProvider interface {
	Speech(ctx context.Context, req *SpeechRequest) ([]byte, error)
}

// SpeechOptions is the parsed callSpeech options JSON. Unknown keys are an
// error so a misspelt option cannot be silently ignored.
type SpeechOptions struct {
	Model        string `json:"model,omitempty"`
	LanguageCode string `json:"language_code,omitempty"`
}

// ParseSpeechOptions parses the options string ("" and "{}" mean none).
func ParseSpeechOptions(s string) (SpeechOptions, error) {
	var o SpeechOptions
	if strings.TrimSpace(s) == "" {
		return o, nil
	}
	dec := json.NewDecoder(bytes.NewReader([]byte(s)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&o); err != nil {
		return o, NewAIError(CodeSchemaValidation,
			fmt.Sprintf("callSpeech options: %v (allowed keys: model, language_code)", err), false)
	}
	return o, nil
}

// CallSpeech implements the effects speech handler: s16le mono 24 kHz PCM.
// A provider without SpeechProvider fails with CodeCapabilityNotSupported.
func (h *Handler) CallSpeech(text, voice, style, options string) ([]byte, error) {
	opts, err := ParseSpeechOptions(options)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(text) == "" {
		return nil, NewAIError(CodeSchemaValidation, "callSpeech: text is empty", false)
	}
	sp, ok := h.provider.(SpeechProvider)
	if !ok {
		return nil, NewAIError(CodeCapabilityNotSupported, fmt.Sprintf(
			"callSpeech: text-to-speech is not supported by provider %q (supported: gemini)", h.provider.Name()), false)
	}
	model := opts.Model
	if model == "" {
		model = h.model
	}
	return sp.Speech(context.Background(), &SpeechRequest{
		Model:        model,
		Text:         text,
		Voice:        voice,
		Style:        style,
		LanguageCode: opts.LanguageCode,
	})
}
