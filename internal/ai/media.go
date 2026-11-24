package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

// MediaKind identifies the constructor variant of a std/ai.MediaPart.
type MediaKind string

const (
	// MediaKindImage corresponds to std/ai.Image({source, mime, resolution}).
	MediaKindImage MediaKind = "image"
	// MediaKindFile corresponds to std/ai.File({uri, mime, resolution, options}).
	MediaKindFile MediaKind = "file"
)

// MediaClass categorizes a File part's MIME type for slicing option validation.
type MediaClass string

const (
	MediaClassDocument    MediaClass = "document"
	MediaClassSpreadsheet MediaClass = "spreadsheet"
	MediaClassAudio       MediaClass = "audio"
	MediaClassVideo       MediaClass = "video"
	MediaClassImage       MediaClass = "image"
	MediaClassOther       MediaClass = "other"
)

// MediaPart is one typed media INPUT attachment (generalizes ImagePart,
// M-STD-AI-GEMINI-FILE-SLICING, #2458).
//
//	Kind       : MediaKindImage ("image") or MediaKindFile ("file")
//	Source     : Image.source (base64 or data-URI) or File.uri (gs://, https://,
//	             data:, or raw base64)
//	Mime       : IANA media type (e.g., "image/png", "application/pdf")
//	Resolution : "" | "low" | "medium" | "high" ("" = provider default)
//	Options    : File-only JSON slicing options string ("" = whole artifact)
//	Parsed     : Validated slicing options (populated by ValidateMediaPart)
type MediaPart struct {
	Kind       MediaKind
	Source     string
	Mime       string
	Resolution string
	Options    string
	Parsed     *MediaSlicingOptions
}

// HasSlicingOptions reports whether the part carries any non-empty slicing
// controls beyond whole-file attachment.
func (p MediaPart) HasSlicingOptions() bool {
	if p.Parsed != nil {
		return p.Parsed.PageOptions != nil ||
			p.Parsed.SheetIndex != nil ||
			p.Parsed.VisibleCells != "" ||
			p.Parsed.StartOffset != "" ||
			p.Parsed.EndOffset != "" ||
			p.Parsed.FPS != nil ||
			len(p.Parsed.ContentLayers) > 0
	}
	s := strings.TrimSpace(p.Options)
	return s != "" && s != "{}"
}

// PageOptions specifies a 1-indexed inclusive page range for documents/slides.
type PageOptions struct {
	StartPage int `json:"start_page"`
	EndPage   int `json:"end_page"`
}

// MediaSlicingOptions holds validated per-part server-side slicing controls.
type MediaSlicingOptions struct {
	// Documents / slides
	PageOptions *PageOptions

	// Spreadsheets (0-indexed in AILANG API; translated to 1-indexed on Gemini wire)
	SheetIndex   *int
	VisibleCells string

	// Audio / video (normalized to protobuf duration string "<seconds>s" for wire)
	StartOffset string
	EndOffset   string
	FPS         *float64

	// ContentLayers holds validated lowercase layer names for this media class.
	ContentLayers []string
}

// MediaCallOptions holds call-level options for callMediaResult.
type MediaCallOptions struct {
	Model string `json:"model"`
}

// ParseMediaCallOptions strictly decodes callMediaResult's options JSON.
// Empty string or "{}" returns a zero MediaCallOptions. Unknown keys fail with
// non-retryable SchemaValidation.
func ParseMediaCallOptions(optionsJSON string) (MediaCallOptions, error) {
	trimmed := strings.TrimSpace(optionsJSON)
	if trimmed == "" || trimmed == "{}" {
		return MediaCallOptions{}, nil
	}
	var opts MediaCallOptions
	dec := json.NewDecoder(bytes.NewReader([]byte(trimmed)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&opts); err != nil {
		return MediaCallOptions{}, NewAIError(CodeSchemaValidation,
			fmt.Sprintf("callMediaResult: invalid options JSON: %v", err), false)
	}
	var trailing any
	if err := dec.Decode(&trailing); err == nil {
		return MediaCallOptions{}, NewAIError(CodeSchemaValidation,
			"callMediaResult: trailing data in options JSON", false)
	}
	return opts, nil
}

// ClassifyMediaMIME maps a MIME type (case-insensitive, parameters stripped) to
// its slicing media class.
func ClassifyMediaMIME(mime string) MediaClass {
	m := strings.ToLower(strings.TrimSpace(mime))
	if semi := strings.IndexByte(m, ';'); semi >= 0 {
		m = strings.TrimSpace(m[:semi])
	}
	switch {
	case m == "application/pdf",
		strings.HasPrefix(m, "application/vnd.openxmlformats-officedocument.presentationml."),
		strings.HasPrefix(m, "application/vnd.openxmlformats-officedocument.wordprocessingml."),
		m == "application/vnd.google-apps.presentation",
		m == "application/vnd.google-apps.document",
		m == "application/vnd.oasis.opendocument.presentation",
		m == "application/vnd.oasis.opendocument.text":
		return MediaClassDocument
	case strings.HasPrefix(m, "application/vnd.openxmlformats-officedocument.spreadsheetml."),
		m == "application/vnd.google-apps.spreadsheet",
		m == "application/vnd.oasis.opendocument.spreadsheet",
		m == "text/csv":
		return MediaClassSpreadsheet
	case strings.HasPrefix(m, "audio/"):
		return MediaClassAudio
	case strings.HasPrefix(m, "video/"):
		return MediaClassVideo
	case strings.HasPrefix(m, "image/"):
		return MediaClassImage
	default:
		return MediaClassOther
	}
}

var (
	a1RangePattern  = regexp.MustCompile(`^[A-Za-z]{1,3}[1-9][0-9]*(:[A-Za-z]{1,3}[1-9][0-9]*)?$`)
	isoDurPattern   = regexp.MustCompile(`^PT(?:(\d+(?:\.\d+)?)H)?(?:(\d+(?:\.\d+)?)M)?(?:(\d+(?:\.\d+)?)S)?$`)
	plainSecPattern = regexp.MustCompile(`^(\d+(?:\.\d+)?)s?$`)
)

// NormalizeOffsetDuration validates an audio/video offset ("PT90S", "PT1M30S",
// "90", "90s", "12.5s") and normalizes it to protobuf duration syntax ("90s").
func NormalizeOffsetDuration(raw string) (string, float64, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", 0, fmt.Errorf("empty duration")
	}
	if m := isoDurPattern.FindStringSubmatch(s); m != nil && (m[1] != "" || m[2] != "" || m[3] != "") {
		var total float64
		if m[1] != "" {
			v, _ := strconv.ParseFloat(m[1], 64)
			total += v * 3600
		}
		if m[2] != "" {
			v, _ := strconv.ParseFloat(m[2], 64)
			total += v * 60
		}
		if m[3] != "" {
			v, _ := strconv.ParseFloat(m[3], 64)
			total += v
		}
		return formatProtoSeconds(total), total, nil
	}
	if m := plainSecPattern.FindStringSubmatch(s); m != nil {
		v, err := strconv.ParseFloat(m[1], 64)
		if err != nil || math.IsNaN(v) || math.IsInf(v, 0) || v < 0 {
			return "", 0, fmt.Errorf("invalid seconds duration %q", raw)
		}
		return formatProtoSeconds(v), v, nil
	}
	return "", 0, fmt.Errorf("invalid duration %q (expected ISO-8601 like \"PT90S\" or seconds like \"90\" / \"90s\")", raw)
}

func formatProtoSeconds(sec float64) string {
	return strconv.FormatFloat(sec, 'f', -1, 64) + "s"
}

// ValidateMediaParts validates every MediaPart in place, populating Parsed
// slicing options. Returns a non-retryable SchemaValidation AIError on failure.
func ValidateMediaParts(parts []MediaPart) error {
	for i := range parts {
		if err := ValidateMediaPart(&parts[i], fmt.Sprintf("media[%d]", i)); err != nil {
			return err
		}
	}
	return nil
}

// ValidateMediaPart validates one MediaPart in place.
func ValidateMediaPart(p *MediaPart, label string) error {
	switch p.Resolution {
	case "", "low", "medium", "high":
	default:
		return NewAIError(CodeSchemaValidation,
			fmt.Sprintf("%s: invalid resolution %q (want \"\", \"low\", \"medium\", or \"high\")", label, p.Resolution), false)
	}

	switch p.Kind {
	case MediaKindImage:
		if strings.TrimSpace(p.Source) == "" {
			return NewAIError(CodeSchemaValidation,
				fmt.Sprintf("%s: Image has empty source (expected base64 or data-URI)", label), false)
		}
		if !strings.HasPrefix(p.Source, "data:") && strings.TrimSpace(p.Mime) == "" {
			return NewAIError(CodeSchemaValidation,
				fmt.Sprintf("%s: Image raw base64 source requires a mime type (e.g. \"image/png\")", label), false)
		}
		if strings.TrimSpace(p.Options) != "" {
			return NewAIError(CodeSchemaValidation,
				fmt.Sprintf("%s: Image does not accept slicing options", label), false)
		}
		p.Parsed = nil
		return nil

	case MediaKindFile:
		if strings.TrimSpace(p.Source) == "" {
			return NewAIError(CodeSchemaValidation,
				fmt.Sprintf("%s: File has empty uri (expected gs://, https://, data:, or base64)", label), false)
		}
		if !strings.HasPrefix(p.Source, "data:") && strings.TrimSpace(p.Mime) == "" {
			return NewAIError(CodeSchemaValidation,
				fmt.Sprintf("%s: File requires a mime type (e.g. \"application/pdf\")", label), false)
		}
		opts, err := ParseMediaOptions(p.Mime, p.Options, label)
		if err != nil {
			return err
		}
		p.Parsed = opts
		return nil

	default:
		return NewAIError(CodeSchemaValidation,
			fmt.Sprintf("%s: unknown MediaPart kind %q", label, p.Kind), false)
	}
}

// ParseMediaOptions validates a File part's JSON options string against the
// allowed keys and value ranges for its MIME class.
func ParseMediaOptions(mime, optionsJSON, label string) (*MediaSlicingOptions, error) {
	trimmed := strings.TrimSpace(optionsJSON)
	if trimmed == "" || trimmed == "{}" {
		return nil, nil
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal([]byte(trimmed), &raw); err != nil {
		return nil, NewAIError(CodeSchemaValidation,
			fmt.Sprintf("%s: invalid options JSON: %v", label, err), false)
	}
	if len(raw) == 0 {
		return nil, nil
	}

	class := ClassifyMediaMIME(mime)
	switch class {
	case MediaClassDocument:
		return parseDocumentOptions(raw, label)
	case MediaClassSpreadsheet:
		return parseSpreadsheetOptions(raw, label)
	case MediaClassAudio:
		return parseAudioVideoOptions(raw, label, false)
	case MediaClassVideo:
		return parseAudioVideoOptions(raw, label, true)
	case MediaClassImage:
		return parseImageFileOptions(raw, label)
	default:
		return nil, NewAIError(CodeSchemaValidation,
			fmt.Sprintf("%s: slicing options are not defined for MIME type %q", label, mime), false)
	}
}

func parseDocumentOptions(raw map[string]json.RawMessage, label string) (*MediaSlicingOptions, error) {
	out := &MediaSlicingOptions{}
	for k, v := range raw {
		switch k {
		case "page_options":
			var po struct {
				StartPage *int `json:"start_page"`
				EndPage   *int `json:"end_page"`
			}
			dec := json.NewDecoder(bytes.NewReader(v))
			dec.DisallowUnknownFields()
			if err := dec.Decode(&po); err != nil {
				return nil, NewAIError(CodeSchemaValidation,
					fmt.Sprintf("%s: invalid page_options: %v", label, err), false)
			}
			if po.StartPage == nil || po.EndPage == nil {
				return nil, NewAIError(CodeSchemaValidation,
					fmt.Sprintf("%s: page_options requires both start_page and end_page", label), false)
			}
			if *po.StartPage < 1 || *po.EndPage < *po.StartPage {
				return nil, NewAIError(CodeSchemaValidation,
					fmt.Sprintf("%s: invalid page range [%d..%d] (1-indexed inclusive, start_page >= 1, end_page >= start_page)", label, *po.StartPage, *po.EndPage), false)
			}
			out.PageOptions = &PageOptions{StartPage: *po.StartPage, EndPage: *po.EndPage}

		case "content_layers":
			layers, err := parseStringLayers(v, label, "document", map[string]bool{
				"page_text":  true,
				"page_image": true,
			})
			if err != nil {
				return nil, err
			}
			out.ContentLayers = layers

		default:
			return nil, NewAIError(CodeSchemaValidation,
				fmt.Sprintf("%s: unknown option %q for document media (allowed: page_options, content_layers)", label, k), false)
		}
	}
	return out, nil
}

func parseSpreadsheetOptions(raw map[string]json.RawMessage, label string) (*MediaSlicingOptions, error) {
	out := &MediaSlicingOptions{}
	for k, v := range raw {
		switch k {
		case "sheet_index":
			var idx int
			if err := json.Unmarshal(v, &idx); err != nil {
				return nil, NewAIError(CodeSchemaValidation,
					fmt.Sprintf("%s: sheet_index must be an integer: %v", label, err), false)
			}
			if idx < 0 {
				return nil, NewAIError(CodeSchemaValidation,
					fmt.Sprintf("%s: sheet_index must be >= 0, got %d", label, idx), false)
			}
			out.SheetIndex = &idx

		case "visible_cells":
			var cells string
			if err := json.Unmarshal(v, &cells); err != nil {
				return nil, NewAIError(CodeSchemaValidation,
					fmt.Sprintf("%s: visible_cells must be a string: %v", label, err), false)
			}
			cells = strings.TrimSpace(cells)
			if !a1RangePattern.MatchString(cells) {
				return nil, NewAIError(CodeSchemaValidation,
					fmt.Sprintf("%s: invalid visible_cells A1 range %q (expected e.g. \"A1:G42\")", label, cells), false)
			}
			out.VisibleCells = cells

		case "content_layers":
			layers, err := parseStringLayers(v, label, "spreadsheet", map[string]bool{
				"data":     true,
				"formulas": true,
				"visuals":  true,
			})
			if err != nil {
				return nil, err
			}
			out.ContentLayers = layers

		default:
			return nil, NewAIError(CodeSchemaValidation,
				fmt.Sprintf("%s: unknown option %q for spreadsheet media (allowed: sheet_index, visible_cells, content_layers)", label, k), false)
		}
	}
	return out, nil
}

func parseAudioVideoOptions(raw map[string]json.RawMessage, label string, isVideo bool) (*MediaSlicingOptions, error) {
	out := &MediaSlicingOptions{}
	var startSec, endSec float64
	var hasStart, hasEnd bool
	className := "audio"
	allowedLayers := map[string]bool{"transcript": true, "audio": true}
	if isVideo {
		className = "video"
		allowedLayers["frames"] = true
	}

	for k, v := range raw {
		switch k {
		case "start_offset":
			var s string
			if err := json.Unmarshal(v, &s); err != nil {
				return nil, NewAIError(CodeSchemaValidation,
					fmt.Sprintf("%s: start_offset must be a string: %v", label, err), false)
			}
			norm, sec, err := NormalizeOffsetDuration(s)
			if err != nil {
				return nil, NewAIError(CodeSchemaValidation, fmt.Sprintf("%s: start_offset: %v", label, err), false)
			}
			out.StartOffset = norm
			startSec, hasStart = sec, true

		case "end_offset":
			var s string
			if err := json.Unmarshal(v, &s); err != nil {
				return nil, NewAIError(CodeSchemaValidation,
					fmt.Sprintf("%s: end_offset must be a string: %v", label, err), false)
			}
			norm, sec, err := NormalizeOffsetDuration(s)
			if err != nil {
				return nil, NewAIError(CodeSchemaValidation, fmt.Sprintf("%s: end_offset: %v", label, err), false)
			}
			out.EndOffset = norm
			endSec, hasEnd = sec, true

		case "fps":
			if !isVideo {
				return nil, NewAIError(CodeSchemaValidation,
					fmt.Sprintf("%s: option \"fps\" is only valid for video media, not audio", label), false)
			}
			var fps float64
			if err := json.Unmarshal(v, &fps); err != nil {
				return nil, NewAIError(CodeSchemaValidation,
					fmt.Sprintf("%s: fps must be a number: %v", label, err), false)
			}
			if fps <= 0 || math.IsNaN(fps) || math.IsInf(fps, 0) {
				return nil, NewAIError(CodeSchemaValidation,
					fmt.Sprintf("%s: fps must be > 0, got %v", label, fps), false)
			}
			out.FPS = &fps

		case "content_layers":
			layers, err := parseStringLayers(v, label, className, allowedLayers)
			if err != nil {
				return nil, err
			}
			out.ContentLayers = layers

		default:
			return nil, NewAIError(CodeSchemaValidation,
				fmt.Sprintf("%s: unknown option %q for %s media", label, k, className), false)
		}
	}
	if hasStart && hasEnd && endSec < startSec {
		return nil, NewAIError(CodeSchemaValidation,
			fmt.Sprintf("%s: end_offset (%s) must be >= start_offset (%s)", label, out.EndOffset, out.StartOffset), false)
	}
	return out, nil
}

func parseImageFileOptions(raw map[string]json.RawMessage, label string) (*MediaSlicingOptions, error) {
	out := &MediaSlicingOptions{}
	for k, v := range raw {
		switch k {
		case "content_layers":
			layers, err := parseStringLayers(v, label, "image", map[string]bool{
				"image_text": true,
				"image":      true,
			})
			if err != nil {
				return nil, err
			}
			out.ContentLayers = layers
		default:
			return nil, NewAIError(CodeSchemaValidation,
				fmt.Sprintf("%s: unknown option %q for image media (allowed: content_layers)", label, k), false)
		}
	}
	return out, nil
}

func parseStringLayers(raw json.RawMessage, label, className string, allowed map[string]bool) ([]string, error) {
	var layers []string
	if err := json.Unmarshal(raw, &layers); err != nil {
		return nil, NewAIError(CodeSchemaValidation,
			fmt.Sprintf("%s: content_layers must be a string array: %v", label, err), false)
	}
	if len(layers) == 0 {
		return nil, NewAIError(CodeSchemaValidation,
			fmt.Sprintf("%s: content_layers must not be empty when specified", label), false)
	}
	seen := make(map[string]bool, len(layers))
	out := make([]string, 0, len(layers))
	for _, layer := range layers {
		if !allowed[layer] {
			return nil, NewAIError(CodeSchemaValidation,
				fmt.Sprintf("%s: invalid content_layers value %q for %s media", label, layer, className), false)
		}
		if !seen[layer] {
			seen[layer] = true
			out = append(out, layer)
		}
	}
	return out, nil
}

// ExtractLegacyImages extracts default-resolution ImageParts from a Message's
// EffectiveMedia() for providers that support image input only (Anthropic,
// OpenAI, OpenRouter, Ollama). Any File part, explicit Resolution, slicing
// options, or media attached to a non-user message fails closed with a
// non-retryable CapabilityNotSupported AIError rather than silently dropping
// the attachment or control.
func ExtractLegacyImages(provider string, m Message) ([]ImagePart, *AIError) {
	media := m.EffectiveMedia()
	if len(media) == 0 {
		return nil, nil
	}
	if m.Role != "user" {
		return nil, NewAIError(CodeCapabilityNotSupported,
			fmt.Sprintf("%s: media attachments on %q messages are not supported (use Gemini)", provider, m.Role), false)
	}
	out := make([]ImagePart, 0, len(media))
	for i, mp := range media {
		if mp.Kind == MediaKindFile {
			return nil, NewAIError(CodeCapabilityNotSupported,
				fmt.Sprintf("%s: media[%d]: File media parts are not supported (use Gemini)", provider, i), false)
		}
		if mp.Resolution != "" {
			return nil, NewAIError(CodeCapabilityNotSupported,
				fmt.Sprintf("%s: media[%d]: per-part resolution %q is not supported (use Gemini)", provider, i, mp.Resolution), false)
		}
		if mp.HasSlicingOptions() {
			return nil, NewAIError(CodeCapabilityNotSupported,
				fmt.Sprintf("%s: media[%d]: slicing options are not supported (use Gemini)", provider, i), false)
		}
		out = append(out, ImagePart{Source: mp.Source, Mime: mp.Mime})
	}
	return out, nil
}

// RejectUnsupportedInputMedia returns a non-retryable CapabilityNotSupported
// AIError if req.InputMedia is non-empty. Used by providers that do not
// implement single-shot typed media/slicing calls.
func RejectUnsupportedInputMedia(provider string, req *Request) error {
	if req != nil && len(req.InputMedia) > 0 {
		return NewAIError(CodeCapabilityNotSupported,
			fmt.Sprintf("%s: callMediaResult / typed media input is not supported (use Gemini)", provider), false)
	}
	return nil
}

// CallMediaResult executes a single-shot prompt with typed media attachments
// and returns the full ai.Response (text + InputTokens / OutputTokens).
func (h *Handler) CallMediaResult(prompt string, media []MediaPart, options string) (*Response, error) {
	parts := make([]MediaPart, len(media))
	copy(parts, media)
	if err := ValidateMediaParts(parts); err != nil {
		return nil, err
	}
	opts, err := ParseMediaCallOptions(options)
	if err != nil {
		return nil, err
	}
	model := h.model
	if opts.Model != "" {
		model = opts.Model
	}
	resp, err := h.provider.Generate(context.Background(), &Request{
		Model:        model,
		SystemPrompt: h.systemPrompt,
		UserPrompt:   prompt,
		InputMedia:   parts,
		Routing:      h.routingPolicy,
	})
	if err != nil {
		return nil, err
	}
	return resp, nil
}
