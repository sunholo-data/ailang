package openrouter

// Image output (M-OPENROUTER-IMAGE-OUTPUT, #1500) and reference-conditioned
// image input (#1496) on the Generate path.
//
// OpenRouter serves image-output models through the same Chat Completions
// endpoint as text: the request carries "modalities": ["image","text"] and the
// response returns generated images inline in choices[].message.images[] as
// base64 data URLs. Reference images ride on the user turn as OpenAI-style
// image_url content parts.

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sunholo-data/ailang/internal/ai"
)

// imageOutputFormats maps the options mime_type to image_config.output_format.
// An explicit mime_type outside this set fails before dispatch rather than
// silently producing a different format.
var imageOutputFormats = map[string]string{
	"image/png":  "png",
	"image/jpeg": "jpeg",
	"image/jpg":  "jpeg",
	"image/webp": "webp",
}

// rejectImageOnStep refuses image output on the Step/StreamStep paths, which
// build from Messages/Tools and would otherwise silently return text.
func rejectImageOnStep(req *ai.Request, path string) *ai.AIError {
	if !ai.RequestsImage(req) {
		return nil
	}
	return ai.NewAIError(ai.CodeCapabilityNotSupported,
		fmt.Sprintf("openrouter: image output is not supported on the %s path; use callImage/callImageBase64", path), false)
}

// applyImageRequest sets the image-output extension fields on apiReq when req
// asks for an image. Text requests are left untouched (byte-identical bodies).
func applyImageRequest(apiReq *chatRequest, req *ai.Request) error {
	if !ai.RequestsImage(req) {
		return nil
	}
	// ai.Request uses Gemini-style uppercase modality names; OpenRouter's enum
	// is lowercase. Ask for text too: image models routinely caption.
	apiReq.Modalities = []string{"image", "text"}

	opts := req.ImageOptions
	if opts == nil || (opts.AspectRatio == "" && opts.MIMEType == "") {
		return nil
	}
	cfg := &imageConfig{AspectRatio: opts.AspectRatio}
	if opts.MIMEType != "" {
		format, ok := imageOutputFormats[strings.ToLower(opts.MIMEType)]
		if !ok {
			return ai.NewProviderError("openrouter", 0,
				fmt.Sprintf("unsupported image mime_type %q for model %q (supported: image/png, image/jpeg, image/webp)",
					opts.MIMEType, req.Model), nil)
		}
		cfg.OutputFormat = format
	}
	apiReq.ImageConfig = cfg
	return nil
}

// marshalChatRequest marshals apiReq, switching the messages array to
// multimodal content parts when the request carries reference images.
func marshalChatRequest(apiReq chatRequest, req *ai.Request) ([]byte, error) {
	if len(req.InputImages) == 0 {
		return json.Marshal(apiReq)
	}
	msgs := make([]partsMessage, 0, len(apiReq.Messages))
	for _, m := range apiReq.Messages {
		if m.Role != "user" {
			msgs = append(msgs, partsMessage{Role: m.Role, Content: m.Content})
			continue
		}
		parts := make([]contentPart, 0, 1+len(req.InputImages))
		if m.Content != "" {
			parts = append(parts, contentPart{Type: "text", Text: m.Content})
		}
		for _, img := range req.InputImages {
			parts = append(parts, contentPart{
				Type:     "image_url",
				ImageURL: &imageURL{URL: toDataURL(img)},
			})
		}
		msgs = append(msgs, partsMessage{Role: "user", Content: parts})
	}
	return json.Marshal(chatRequestWithParts{chatRequest: apiReq, Messages: msgs})
}

// toDataURL passes a data-URI source through and wraps a raw base64 payload.
func toDataURL(img ai.ImagePart) string {
	if strings.HasPrefix(img.Source, "data:") {
		return img.Source
	}
	return "data:" + img.Mime + ";base64," + img.Source
}

// harvestImage extracts the first generated image from the response. It
// returns the decoded bytes, the MIME type and the number of images returned.
// Every failure is a typed ProviderError naming the model.
func harvestImage(msg assistantMessage, model string) ([]byte, string, int, error) {
	if len(msg.Images) == 0 {
		return nil, "", 0, ai.NewProviderError("openrouter", 0,
			fmt.Sprintf("model %q returned no image data for an image request (is it an image-output model?)", model), nil)
	}
	data, mime, err := decodeImageDataURL(msg.Images[0].ImageURL.URL)
	if err != nil {
		return nil, "", len(msg.Images), ai.NewProviderError("openrouter", 0,
			fmt.Sprintf("model %q: %v", model, err), nil)
	}
	return data, mime, len(msg.Images), nil
}

// decodeImageDataURL decodes "data:<mime>;base64,<payload>". Remote URLs are
// refused: fetching them would be an unrequested second network hop.
func decodeImageDataURL(url string) ([]byte, string, error) {
	if !strings.HasPrefix(url, "data:") {
		return nil, "", fmt.Errorf("remote image URLs are not supported by this adapter (got %q)", truncateURL(url))
	}
	comma := strings.IndexByte(url, ',')
	if comma < 0 {
		return nil, "", fmt.Errorf("malformed image data URL (no ',' separator)")
	}
	meta := url[len("data:"):comma]
	if !strings.HasSuffix(meta, ";base64") {
		return nil, "", fmt.Errorf("image data URL is not base64-encoded (%q)", meta)
	}
	mime := strings.TrimSuffix(meta, ";base64")
	if semi := strings.IndexByte(mime, ';'); semi >= 0 {
		mime = mime[:semi]
	}
	if mime == "" {
		mime = "image/png"
	}
	data, err := base64.StdEncoding.DecodeString(url[comma+1:])
	if err != nil {
		return nil, "", fmt.Errorf("failed to decode image data URL: %v", err)
	}
	return data, mime, nil
}

func truncateURL(u string) string {
	if len(u) > 80 {
		return u[:80] + "..."
	}
	return u
}
