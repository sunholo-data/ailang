package effects

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/ai"
	"github.com/sunholo-data/ailang/internal/eval"
)

// ============================================================================
// Decoders — AILANG records → Go structs
// ============================================================================

// decodeMessages converts an AILANG list of Message records into []ai.Message.
// Tolerates missing fields (treated as zero values) so callers can omit
// optional fields like ToolCalls / ToolCallID. Rejects non-record entries.
func decodeMessages(list *eval.ListValue) ([]ai.Message, error) {
	out := make([]ai.Message, 0, len(list.Elements))
	for i, elem := range list.Elements {
		rec, ok := elem.(*eval.RecordValue)
		if !ok {
			return nil, fmt.Errorf("messages[%d]: expected record, got %T", i, elem)
		}
		msg := ai.Message{
			Role:       getStringField(rec, "role"),
			Content:    getStringField(rec, "content"),
			ToolCallID: getStringField(rec, "tool_call_id"),
		}
		// tool_calls is an optional list[ToolCall]; absent or nil = no calls.
		if tcList, ok := rec.Fields["tool_calls"].(*eval.ListValue); ok {
			calls, err := decodeToolCalls(tcList)
			if err != nil {
				return nil, fmt.Errorf("messages[%d]: %w", i, err)
			}
			msg.ToolCalls = calls
		}
		// media is a list[MediaPart] (M-STD-AI-GEMINI-FILE-SLICING, v0.60.7).
		// Empty/absent = text-only message.
		if mediaList, ok := rec.Fields["media"].(*eval.ListValue); ok && len(mediaList.Elements) > 0 {
			media, err := decodeMediaParts(mediaList)
			if err != nil {
				return nil, fmt.Errorf("messages[%d]: %w", i, err)
			}
			msg.Media = media
		}
		// Legacy Go-test fallback: tolerate "images" if a Go unit test constructs
		// a pre-v0.60.7 RecordValue directly.
		if len(msg.Media) == 0 {
			if imgList, ok := rec.Fields["images"].(*eval.ListValue); ok && len(imgList.Elements) > 0 {
				images, err := decodeImageParts(imgList)
				if err != nil {
					return nil, fmt.Errorf("messages[%d]: %w", i, err)
				}
				msg.Images = images
			}
		}
		out = append(out, msg)
	}
	return out, nil
}

// decodeToolCalls converts an AILANG list of ToolCall records into []ai.ToolCall.
func decodeToolCalls(list *eval.ListValue) ([]ai.ToolCall, error) {
	out := make([]ai.ToolCall, 0, len(list.Elements))
	for i, elem := range list.Elements {
		rec, ok := elem.(*eval.RecordValue)
		if !ok {
			return nil, fmt.Errorf("tool_calls[%d]: expected record, got %T", i, elem)
		}
		out = append(out, ai.ToolCall{
			ID:        getStringField(rec, "id"),
			Name:      getStringField(rec, "name"),
			Arguments: getStringField(rec, "arguments"),
		})
	}
	return out, nil
}

// decodeMediaParts decodes an AILANG list[MediaPart] into []ai.MediaPart and
// validates every part (resolution enum, required source/mime, per-MIME slicing
// options JSON) at the effect boundary.
func decodeMediaParts(list *eval.ListValue) ([]ai.MediaPart, error) {
	out := make([]ai.MediaPart, 0, len(list.Elements))
	for i, elem := range list.Elements {
		tv, ok := elem.(*eval.TaggedValue)
		if !ok {
			return nil, ai.NewAIError(ai.CodeSchemaValidation,
				fmt.Sprintf("media[%d]: expected MediaPart ADT (Image or File), got %T", i, elem), false)
		}
		if len(tv.Fields) != 1 {
			return nil, ai.NewAIError(ai.CodeSchemaValidation,
				fmt.Sprintf("media[%d]: %s constructor expected 1 record payload, got %d", i, tv.CtorName, len(tv.Fields)), false)
		}
		rec, ok := tv.Fields[0].(*eval.RecordValue)
		if !ok {
			return nil, ai.NewAIError(ai.CodeSchemaValidation,
				fmt.Sprintf("media[%d]: %s payload expected record, got %T", i, tv.CtorName, tv.Fields[0]), false)
		}
		var part ai.MediaPart
		switch tv.CtorName {
		case "Image":
			part = ai.MediaPart{
				Kind:       ai.MediaKindImage,
				Source:     getStringField(rec, "source"),
				Mime:       getStringField(rec, "mime"),
				Resolution: getStringField(rec, "resolution"),
			}
		case "File":
			part = ai.MediaPart{
				Kind:       ai.MediaKindFile,
				Source:     getStringField(rec, "uri"),
				Mime:       getStringField(rec, "mime"),
				Resolution: getStringField(rec, "resolution"),
				Options:    getStringField(rec, "options"),
			}
		default:
			return nil, ai.NewAIError(ai.CodeSchemaValidation,
				fmt.Sprintf("media[%d]: unknown MediaPart constructor %q (want Image or File)", i, tv.CtorName), false)
		}
		if err := ai.ValidateMediaPart(&part, fmt.Sprintf("media[%d]", i)); err != nil {
			return nil, err
		}
		out = append(out, part)
	}
	return out, nil
}

// decodeImageParts decodes an AILANG list[ImagePart] into []ai.ImagePart.
// Each element is a {source, mime} record (M-STD-AI-VISION-INPUT). An empty
// source is rejected — the caller (decodeMessages) surfaces it as a typed
// SchemaValidation AIError rather than silently forwarding a blank image.
func decodeImageParts(list *eval.ListValue) ([]ai.ImagePart, error) {
	out := make([]ai.ImagePart, 0, len(list.Elements))
	for i, elem := range list.Elements {
		rec, ok := elem.(*eval.RecordValue)
		if !ok {
			return nil, fmt.Errorf("images[%d]: expected record, got %T", i, elem)
		}
		source := getStringField(rec, "source")
		if source == "" {
			return nil, fmt.Errorf("images[%d]: empty source (expected base64 or data-URI)", i)
		}
		out = append(out, ai.ImagePart{
			Source: source,
			Mime:   getStringField(rec, "mime"),
		})
	}
	return out, nil
}

// decodeCacheBreakpoints converts an AILANG list of CacheBreakpoint records
// into []ai.CacheBreakpoint. Tolerates missing fields (treated as empty
// strings) — provider-side validation rejects unknown positions with a
// once-per-session warning rather than failing.
func decodeCacheBreakpoints(list *eval.ListValue) ([]ai.CacheBreakpoint, error) {
	if list == nil || len(list.Elements) == 0 {
		return nil, nil
	}
	out := make([]ai.CacheBreakpoint, 0, len(list.Elements))
	for i, elem := range list.Elements {
		rec, ok := elem.(*eval.RecordValue)
		if !ok {
			return nil, fmt.Errorf("cache_breakpoints[%d]: expected record, got %T", i, elem)
		}
		out = append(out, ai.CacheBreakpoint{
			Position: getStringField(rec, "position"),
			TTL:      getStringField(rec, "ttl"),
		})
	}
	return out, nil
}

// decodeToolSchemas converts an AILANG list of ToolSchema records into
// []ai.ToolSchema.
func decodeToolSchemas(list *eval.ListValue) ([]ai.ToolSchema, error) {
	out := make([]ai.ToolSchema, 0, len(list.Elements))
	for i, elem := range list.Elements {
		rec, ok := elem.(*eval.RecordValue)
		if !ok {
			return nil, fmt.Errorf("tools[%d]: expected record, got %T", i, elem)
		}
		out = append(out, ai.ToolSchema{
			Name:        getStringField(rec, "name"),
			Description: getStringField(rec, "description"),
			Parameters:  getStringField(rec, "parameters"),
		})
	}
	return out, nil
}

// getStringField extracts a string field from a record, returning ""
// if the field is missing or not a StringValue. Forgiving by design —
// schema-level rigor is the type checker's job, not this layer.
func getStringField(rec *eval.RecordValue, name string) string {
	v, ok := rec.Fields[name]
	if !ok {
		return ""
	}
	s, ok := v.(*eval.StringValue)
	if !ok {
		return ""
	}
	return s.Value
}

// ============================================================================
// Encoders — Go structs → AILANG Result records
// ============================================================================
