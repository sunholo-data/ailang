package effects

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
)

// fixtureVector rejects null coordinates: encoding/json otherwise silently
// turns a JSON null into float64 zero, hiding malformed fixture data.
type fixtureVector []float64

func (v *fixtureVector) UnmarshalJSON(data []byte) error {
	var raw []json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	values := make([]float64, len(raw))
	for i, coordinate := range raw {
		if bytes.Equal(bytes.TrimSpace(coordinate), []byte("null")) {
			return fmt.Errorf("embed vector coordinate %d must be numeric", i)
		}
		if err := json.Unmarshal(coordinate, &values[i]); err != nil {
			return err
		}
	}
	*v = values
	return nil
}

// EmbedFixtureCanonical keys text and media separately. Media includes explicit
// presence flags, never raw image/audio data. Compact sorted JSON, no HTML escaping.
func EmbedFixtureCanonical(model, text string, media, image, audio bool) string {
	request := map[string]any{"model": model, "text": text}
	if media {
		request["image"] = image
		request["audio"] = audio
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(request)
	return strings.TrimSuffix(buf.String(), "\n")
}

func (e *fixtureEntry) embedKey() (string, error) {
	if e.Prompt != nil {
		return "", fmt.Errorf("embed fixture needs request or sha256, not prompt")
	}
	if e.Request == nil {
		return "", fmt.Errorf("embed fixture needs request or sha256")
	}
	if e.SHA256 != "" {
		return "", fmt.Errorf("give exactly one of request or sha256")
	}
	r := e.Request
	if r.Model == nil {
		return "", fmt.Errorf("embed request needs model")
	}
	if r.Voice != "" || r.Style != "" {
		return "", fmt.Errorf("embed request cannot carry voice or style")
	}
	if (r.Image == nil) != (r.Audio == nil) {
		return "", fmt.Errorf("media request needs both image and audio booleans")
	}
	media := r.Image != nil
	image, audio := false, false
	if media {
		image, audio = *r.Image, *r.Audio
	}
	return FixtureKey(EmbedFixtureCanonical(*r.Model, r.Text, media, image, audio)), nil
}

func (e *fixtureEntry) validateEmbed() error {
	if e.Response != nil || len(e.ToolCalls) > 0 || e.Base64 != "" || e.MimeType != "" || e.PCMBase64 != "" {
		return fmt.Errorf("embed fixture carries only vector or error")
	}
	if e.Error != nil {
		if e.Vector != nil || strings.TrimSpace(e.Error.Message) == "" {
			return fmt.Errorf("embed error needs a non-empty message and no vector")
		}
		return nil
	}
	if len(e.Vector) == 0 {
		return fmt.Errorf("embed fixture needs non-empty numeric vector")
	}
	for _, v := range e.Vector {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return fmt.Errorf("embed vector must contain finite numbers")
		}
	}
	return nil
}

func (h *FixtureAIHandler) embed(canonical string) ([]float64, error) {
	// Preserve fixture error messages verbatim rather than classifying as AIError.
	key := FixtureKey(canonical)
	if e := h.fixtures.entries[fixtureKindEmbed][key]; e != nil {
		if e.Error != nil {
			return nil, errors.New(e.Error.Message)
		}
		return append([]float64(nil), e.Vector...), nil
	}
	_, err := h.fixtures.lookup(fixtureKindEmbed, canonical)
	return nil, err
}

func (h *FixtureAIHandler) embedOllama(model, text string, media bool, image, audio []byte) ([]float64, error) {
	return h.embed(EmbedFixtureCanonical(model, text, media, len(image) > 0, len(audio) > 0))
}
