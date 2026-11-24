package gemini

import (
	"fmt"
	"strings"

	"github.com/sunholo-data/ailang/internal/ai"
)

// requestNeedsVertexBeta reports whether a Gemini request uses v1beta1-only
// Part fields on Vertex AI (documentMetadata, spreadsheetMetadata,
// audioMetadata, videoMetadata.contentLayers, per-part mediaResolution, or
// multimodal functionResponse.parts). Vertex v1 rejects those fields with
// HTTP 400; Vertex v1beta1 on the same regional/EU host exposes them.
func requestNeedsVertexBeta(req *ai.Request) bool {
	if req == nil {
		return false
	}
	for _, p := range req.InputMedia {
		if mediaPartNeedsVertexBeta(p) {
			return true
		}
	}
	for _, m := range req.Messages {
		media := m.EffectiveMedia()
		if len(media) == 0 {
			continue
		}
		if m.Role == "tool" || m.ToolCallID != "" {
			return true
		}
		for _, p := range media {
			if mediaPartNeedsVertexBeta(p) {
				return true
			}
		}
	}
	return false
}

func mediaPartNeedsVertexBeta(p ai.MediaPart) bool {
	return p.Resolution != "" || p.HasSlicingOptions()
}

// userMediaParts builds the ordered Gemini parts slice for a user message
// carrying typed media attachments. A leading text part is emitted only when
// m.Content is non-empty, followed by one part per attachment in order.
func userMediaParts(m ai.Message, authType AuthType) ([]part, error) {
	media := m.EffectiveMedia()
	parts := make([]part, 0, 1+len(media))
	if m.Content != "" {
		parts = append(parts, part{Text: m.Content})
	}
	for i, mp := range media {
		p, err := buildMediaPart(mp, authType, fmt.Sprintf("media[%d]", i))
		if err != nil {
			return nil, err
		}
		parts = append(parts, p)
	}
	return parts, nil
}

// toolMediaParts builds the Gemini parts slice for multimodal tool-result
// attachments (attached to functionResponse.parts[]).
func toolMediaParts(m ai.Message, authType AuthType) ([]part, error) {
	media := m.EffectiveMedia()
	parts := make([]part, 0, len(media))
	for i, mp := range media {
		p, err := buildMediaPart(mp, authType, fmt.Sprintf("tool.media[%d]", i))
		if err != nil {
			return nil, err
		}
		parts = append(parts, p)
	}
	return parts, nil
}

// buildMediaPart translates one validated ai.MediaPart into a Gemini wire part,
// enforcing transport capability gates before any network dispatch.
func buildMediaPart(mp ai.MediaPart, authType AuthType, label string) (part, error) {
	if mp.Parsed == nil && mp.HasSlicingOptions() {
		if err := ai.ValidateMediaPart(&mp, label); err != nil {
			return part{}, err
		}
	}

	var out part
	switch {
	case strings.HasPrefix(mp.Source, "gs://"):
		if authType == AuthAPIKey {
			return part{}, ai.NewAIError(ai.CodeCapabilityNotSupported,
				fmt.Sprintf("gemini: %s uses gs:// URI which requires Vertex AI (ADC), not AI Studio API key", label), false)
		}
		out.FileData = &fileData{
			MimeType: mp.Mime,
			FileUri:  mp.Source,
		}
	case strings.HasPrefix(mp.Source, "https://") || strings.HasPrefix(mp.Source, "http://"):
		out.FileData = &fileData{
			MimeType: mp.Mime,
			FileUri:  mp.Source,
		}
	default:
		mimeType, data := splitDataURI(mp.Source, mp.Mime)
		out.InlineData = &inlineData{
			MimeType: mimeType,
			Data:     data,
		}
	}

	if mp.Resolution != "" {
		res, err := mapMediaResolution(mp.Resolution, label)
		if err != nil {
			return part{}, err
		}
		out.MediaResolution = res
	}

	if mp.Parsed != nil {
		if err := applySlicingMetadata(&out, mp, label); err != nil {
			return part{}, err
		}
	}

	return out, nil
}

func mapMediaResolution(res, label string) (*partMediaResolution, error) {
	switch res {
	case "low":
		return &partMediaResolution{Level: "MEDIA_RESOLUTION_LOW"}, nil
	case "medium":
		return &partMediaResolution{Level: "MEDIA_RESOLUTION_MEDIUM"}, nil
	case "high":
		return &partMediaResolution{Level: "MEDIA_RESOLUTION_HIGH"}, nil
	default:
		return nil, ai.NewAIError(ai.CodeSchemaValidation,
			fmt.Sprintf("gemini: %s has unsupported resolution %q", label, res), false)
	}
}

func applySlicingMetadata(out *part, mp ai.MediaPart, label string) error {
	opts := mp.Parsed
	class := ai.ClassifyMediaMIME(mp.Mime)

	switch class {
	case ai.MediaClassDocument:
		dm := &documentMetadata{}
		if opts.PageOptions != nil {
			dm.StartPage = opts.PageOptions.StartPage
			dm.EndPage = opts.PageOptions.EndPage
		}
		for _, layer := range opts.ContentLayers {
			switch layer {
			case "page_text":
				dm.ContentLayers = append(dm.ContentLayers, "PAGE_TEXT")
			case "page_image":
				dm.ContentLayers = append(dm.ContentLayers, "PAGE_IMAGE")
			}
		}
		if dm.StartPage > 0 || dm.EndPage > 0 || len(dm.ContentLayers) > 0 {
			out.DocumentMetadata = dm
		}
		return nil

	case ai.MediaClassSpreadsheet:
		sm := &spreadsheetMetadata{
			VisibleCells: opts.VisibleCells,
		}
		if opts.SheetIndex != nil {
			// AILANG sheet_index is 0-based; Gemini wire spreadsheetMetadata.sheetIndex is 1-based.
			sm.SheetIndex = *opts.SheetIndex + 1
		}
		for _, layer := range opts.ContentLayers {
			switch layer {
			case "data":
				sm.ContentLayers = append(sm.ContentLayers, "DATA")
			case "formulas":
				sm.ContentLayers = append(sm.ContentLayers, "FORMULAS")
			case "visuals":
				sm.ContentLayers = append(sm.ContentLayers, "VISUAL")
			}
		}
		if sm.SheetIndex > 0 || sm.VisibleCells != "" || len(sm.ContentLayers) > 0 {
			out.SpreadsheetMetadata = sm
		}
		return nil

	case ai.MediaClassAudio:
		if len(opts.ContentLayers) > 0 {
			return ai.NewAIError(ai.CodeCapabilityNotSupported,
				fmt.Sprintf("gemini: %s: audioMetadata does not support content_layers (only start_offset and end_offset)", label), false)
		}
		if opts.StartOffset != "" || opts.EndOffset != "" {
			out.AudioMetadata = &audioMetadata{
				StartOffset: opts.StartOffset,
				EndOffset:   opts.EndOffset,
			}
		}
		return nil

	case ai.MediaClassVideo:
		vm := &videoMetadata{
			StartOffset: opts.StartOffset,
			EndOffset:   opts.EndOffset,
		}
		if opts.FPS != nil {
			vm.FPS = *opts.FPS
		}
		for _, layer := range opts.ContentLayers {
			switch layer {
			case "transcript":
				vm.ContentLayers = append(vm.ContentLayers, "TRANSCRIPT")
			case "frames":
				vm.ContentLayers = append(vm.ContentLayers, "FRAMES")
			case "audio":
				vm.ContentLayers = append(vm.ContentLayers, "AUDIO")
			}
		}
		if vm.StartOffset != "" || vm.EndOffset != "" || vm.FPS > 0 || len(vm.ContentLayers) > 0 {
			out.VideoMetadata = vm
		}
		return nil

	case ai.MediaClassImage:
		if len(opts.ContentLayers) > 0 {
			return ai.NewAIError(ai.CodeCapabilityNotSupported,
				fmt.Sprintf("gemini: %s: image parts do not support content_layers (use resolution instead)", label), false)
		}
		return nil

	default:
		return ai.NewAIError(ai.CodeCapabilityNotSupported,
			fmt.Sprintf("gemini: %s: slicing metadata is not supported for MIME type %q", label, mp.Mime), false)
	}
}
