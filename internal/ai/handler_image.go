package ai

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Image generation on the Handler: callImage / callImageBase64 (v0.10.0) and
// their reference-conditioned variants (#1496). Every entry point honours a
// per-call "model" key in the options JSON (#1496/#1500); absent or empty, the
// handler's bound model is used exactly as before.

// CallImage generates an image and writes it to outputPath.
// Options is a JSON string: {"aspect_ratio": "16:9", "mime_type": "image/png", "model": "..."}.
func (h *Handler) CallImage(prompt, outputPath, options string) (string, error) {
	return h.CallImageWithRefs(prompt, nil, outputPath, options)
}

// CallImageBase64 generates an image and returns JSON with base64 data.
// Returns: {"base64": "...", "mime_type": "image/png"}
func (h *Handler) CallImageBase64(prompt, options string) (string, error) {
	return h.CallImageBase64WithRefs(prompt, nil, options)
}

// CallImageWithRefs generates an image conditioned on reference images and
// writes it to outputPath. Empty refs behaves exactly like CallImage.
func (h *Handler) CallImageWithRefs(prompt string, refs []ImagePart, outputPath, options string) (string, error) {
	resp, err := h.generateImage(prompt, refs, options)
	if err != nil {
		return "", err
	}
	// Ensure parent directory exists
	if dir := filepath.Dir(outputPath); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return "", fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}
	if err := os.WriteFile(outputPath, resp.ImageData, 0o644); err != nil {
		return "", fmt.Errorf("failed to write image to %s: %w", outputPath, err)
	}
	return outputPath, nil
}

// CallImageBase64WithRefs generates an image conditioned on reference images
// and returns {"base64": "...", "mime_type": "..."}. Empty refs behaves
// exactly like CallImageBase64.
func (h *Handler) CallImageBase64WithRefs(prompt string, refs []ImagePart, options string) (string, error) {
	resp, err := h.generateImage(prompt, refs, options)
	if err != nil {
		return "", err
	}
	b64 := base64.StdEncoding.EncodeToString(resp.ImageData)
	mime := resp.ImageMIME
	if mime == "" {
		mime = "image/png"
	}
	return fmt.Sprintf(`{"base64":"%s","mime_type":"%s"}`, b64, mime), nil
}

// generateImage is the one dispatch path behind every image entry point.
func (h *Handler) generateImage(prompt string, refs []ImagePart, options string) (*Response, error) {
	if err := ValidateInputImages(refs); err != nil {
		return nil, err
	}
	opts := ParseImageOptions(options)
	model := h.model
	if opts != nil && opts.Model != "" {
		model = opts.Model
	}
	resp, err := h.provider.Generate(context.Background(), &Request{
		Model:              model,
		SystemPrompt:       h.systemPrompt,
		UserPrompt:         prompt,
		ResponseModalities: []string{"IMAGE"},
		ImageOptions:       opts,
		InputImages:        refs,
		Routing:            h.routingPolicy,
	})
	if err != nil {
		return nil, err
	}
	if resp.ImageData == nil {
		return nil, fmt.Errorf("provider returned no image data for prompt: %s", prompt)
	}
	return resp, nil
}

// ValidateInputImages checks reference images before dispatch: each needs a
// non-empty source, and a raw base64 source needs a mime type (a data-URI
// carries its own). Returns a typed SchemaValidation AIError on failure.
func ValidateInputImages(refs []ImagePart) error {
	for i, r := range refs {
		if r.Source == "" {
			return NewAIError(CodeSchemaValidation,
				fmt.Sprintf("refs[%d]: empty source (expected base64 or data-URI)", i), false)
		}
		if !strings.HasPrefix(r.Source, "data:") && r.Mime == "" {
			return NewAIError(CodeSchemaValidation,
				fmt.Sprintf("refs[%d]: raw base64 source needs a mime type (e.g. \"image/png\")", i), false)
		}
	}
	return nil
}

// ParseImageOptions parses a JSON options string into ImageOptions. Empty,
// "{}", malformed JSON, or JSON with none of the known keys (aspect_ratio,
// mime_type, model) returns nil; unknown keys are ignored.
func ParseImageOptions(optionsJSON string) *ImageOptions {
	if optionsJSON == "" || optionsJSON == "{}" {
		return nil
	}
	var raw struct {
		AspectRatio string `json:"aspect_ratio"`
		MIMEType    string `json:"mime_type"`
		Model       string `json:"model"`
	}
	if err := json.Unmarshal([]byte(optionsJSON), &raw); err != nil {
		return nil
	}
	if raw.AspectRatio == "" && raw.MIMEType == "" && raw.Model == "" {
		return nil
	}
	return &ImageOptions{
		AspectRatio: raw.AspectRatio,
		MIMEType:    raw.MIMEType,
		Model:       raw.Model,
	}
}

// SetImageOptionsModel returns optionsJSON with its "model" key set to model,
// preserving every other key. Used by the effects layer to substitute a
// resolved api_name for a friendly per-call model name.
func SetImageOptionsModel(optionsJSON, model string) (string, error) {
	fields := map[string]json.RawMessage{}
	if strings.TrimSpace(optionsJSON) != "" {
		if err := json.Unmarshal([]byte(optionsJSON), &fields); err != nil {
			return "", fmt.Errorf("image options: invalid JSON: %w", err)
		}
	}
	m, err := json.Marshal(model)
	if err != nil {
		return "", err
	}
	fields["model"] = m
	out, err := json.Marshal(fields)
	if err != nil {
		return "", err
	}
	return string(out), nil
}
