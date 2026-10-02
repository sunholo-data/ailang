package gemini

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"mime"
	"strconv"
	"strings"

	"github.com/sunholo-data/ailang/internal/ai"
)

// Text-to-speech through the Gemini TTS models (#1495). Same generateContent
// endpoint and auth as text generation, with responseModalities ["AUDIO"]
// and a speechConfig; the reply is one inlineData part of raw PCM
// (audio/L16, 24 kHz, mono, 16-bit little-endian).
//
// The request types live here rather than in types.go so the speech wire
// shape cannot change the text/image request bodies.

type ttsRequest struct {
	Contents         []content           `json:"contents"`
	GenerationConfig ttsGenerationConfig `json:"generationConfig"`
}

type ttsGenerationConfig struct {
	ResponseModalities []string        `json:"responseModalities"`
	SpeechConfig       ttsSpeechConfig `json:"speechConfig"`
}

type ttsSpeechConfig struct {
	VoiceConfig  *ttsVoiceConfig `json:"voiceConfig,omitempty"`
	LanguageCode string          `json:"languageCode,omitempty"`
}

type ttsVoiceConfig struct {
	PrebuiltVoiceConfig ttsPrebuiltVoice `json:"prebuiltVoiceConfig"`
}

type ttsPrebuiltVoice struct {
	VoiceName string `json:"voiceName"`
}

// speechModel picks the TTS model: the request's, when it names a TTS model
// (the bound --ai model usually is a text model), else the documented default.
func speechModel(model string) string {
	if strings.Contains(strings.ToLower(model), "tts") {
		return model
	}
	return ai.DefaultGeminiTTSModel
}

// ttsPrompt folds the style into the prompt the way Gemini TTS is steered:
// a natural-language instruction before the line ("Say cheerfully: Hi").
func ttsPrompt(text, style string) string {
	if strings.TrimSpace(style) == "" {
		return text
	}
	return strings.TrimSpace(style) + ": " + text
}

// Speech implements ai.SpeechProvider.
func (c *Client) Speech(ctx context.Context, req *ai.SpeechRequest) ([]byte, error) {
	body := ttsRequest{
		Contents: []content{{Role: "user", Parts: []part{{Text: ttsPrompt(req.Text, req.Style)}}}},
		GenerationConfig: ttsGenerationConfig{
			ResponseModalities: []string{"AUDIO"},
			SpeechConfig:       ttsSpeechConfig{LanguageCode: req.LanguageCode},
		},
	}
	if req.Voice != "" {
		body.GenerationConfig.SpeechConfig.VoiceConfig = &ttsVoiceConfig{PrebuiltVoiceConfig: ttsPrebuiltVoice{VoiceName: req.Voice}}
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, ai.NewProviderError("gemini", 0, "failed to marshal speech request", err)
	}
	url, err := c.buildURL(speechModel(req.Model))
	if err != nil {
		return nil, err
	}
	headers, err := c.authHeaders()
	if err != nil {
		return nil, err
	}
	var result generateResponse
	if _, err := ai.DoJSON(ctx, ai.JSONCall{
		Provider: "gemini",
		Client:   c.httpClient,
		URL:      url,
		Headers:  headers,
		Body:     jsonBody,
	}, &result); err != nil {
		return nil, err
	}
	return extractPCM(&result)
}

// extractPCM returns the concatenated audio parts, refusing anything that is
// not the documented 24 kHz 16-bit PCM — a resampled or encoded payload
// passed off as s16le would play as noise.
func extractPCM(result *generateResponse) ([]byte, error) {
	protocol := func(msg string) error {
		return ai.NewAIError(ai.CodeProtocolError, "gemini speech: "+msg, false)
	}
	if len(result.Candidates) == 0 {
		return nil, protocol("no candidates in response")
	}
	var pcm []byte
	for _, p := range result.Candidates[0].Content.Parts {
		if p.InlineData == nil || p.InlineData.Data == "" {
			continue
		}
		if err := checkPCMMime(p.InlineData.MimeType); err != nil {
			return nil, protocol(err.Error())
		}
		b, err := base64.StdEncoding.DecodeString(p.InlineData.Data)
		if err != nil {
			return nil, protocol("audio data does not decode: " + err.Error())
		}
		pcm = append(pcm, b...)
	}
	if len(pcm) == 0 {
		return nil, protocol(fmt.Sprintf("no audio in response (finishReason %q)", result.Candidates[0].FinishReason))
	}
	if len(pcm)%2 != 0 {
		return nil, protocol(fmt.Sprintf("odd PCM length %d is not whole 16-bit samples", len(pcm)))
	}
	return pcm, nil
}

// checkPCMMime accepts audio/L16 or audio/pcm; a rate parameter must be the
// documented ai.SpeechSampleRate.
func checkPCMMime(mt string) error {
	media, params, err := mime.ParseMediaType(mt)
	if err != nil {
		return fmt.Errorf("unparseable audio mime type %q", mt)
	}
	if media != "audio/l16" && media != "audio/pcm" {
		return fmt.Errorf("audio mime type %q is not raw 16-bit PCM", mt)
	}
	if r, ok := params["rate"]; ok {
		if n, err := strconv.Atoi(r); err != nil || n != ai.SpeechSampleRate {
			return fmt.Errorf("audio rate %q, want %d", r, ai.SpeechSampleRate)
		}
	}
	return nil
}
