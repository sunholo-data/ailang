package observatory

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

// OTLP/JSON encodes trace and span IDs as HEX strings. This is a deliberate
// deviation from the proto3 JSON mapping, which encodes `bytes` fields as
// base64 — and protojson, correctly implementing proto3, follows the latter.
//
// The result is that feeding a spec-compliant OTLP/JSON body straight to
// protojson.Unmarshal base64-decodes the hex string: a 32-char hex trace ID
// (16 bytes) becomes 24 bytes of unrelated data, and nothing errors. Measured
// against production on 2026-08-13, every OpenRouter Broadcast span landed with
// a 48-hex-char trace ID instead of 32.
//
// normalizeOTLPJSONIDs closes that gap by rewriting the ID fields from hex into
// the base64 protojson expects, ahead of the decode. Keeping protojson as the
// single decoder confines the spec deviation to this one function.
//
// See https://opentelemetry.io/docs/specs/otlp/#json-protobuf-encoding

// otlpIDFields maps the OTLP/JSON ID field names to their required byte length.
// protojson accepts both lowerCamelCase and snake_case for the same field, so
// both spellings are normalized.
var otlpIDFields = map[string]int{
	"traceId":        16,
	"trace_id":       16,
	"spanId":         8,
	"span_id":        8,
	"parentSpanId":   8,
	"parent_span_id": 8,
}

// InvalidOTLPIDError reports an ID field that is not a valid hex string of the
// length its field requires.
//
// Per the M-OPENROUTER-BROADCAST-INGEST Design Freeze this is REJECTED rather
// than repaired or best-effort stored: trace IDs are the join key for
// `ailang chains`, so a wrong one is a data-integrity fault and CLAUDE.md
// Principle 2 gives it no fallback.
type InvalidOTLPIDError struct {
	Field     string
	Value     string
	WantBytes int
}

func (e *InvalidOTLPIDError) Error() string {
	return fmt.Sprintf("invalid %s: expected %d hex chars, got %d (%q)",
		e.Field, e.WantBytes*2, len(e.Value), e.Value)
}

// normalizeOTLPJSONIDs rewrites hex-encoded trace/span IDs in an OTLP/JSON body
// into the base64 form protojson expects.
//
// Bodies that are not valid JSON are returned unchanged so that protojson
// produces the parse error, keeping error reporting in one place. A body with
// no ID fields is returned unchanged and unallocated.
func normalizeOTLPJSONIDs(body []byte) ([]byte, error) {
	var root any
	if err := json.Unmarshal(body, &root); err != nil {
		return body, nil
	}

	changed, err := normalizeOTLPIDNode(root)
	if err != nil {
		return nil, err
	}
	if !changed {
		return body, nil
	}
	return json.Marshal(root)
}

// normalizeOTLPIDNode walks the decoded JSON tree in place, converting any ID
// field it finds. It reports whether anything was rewritten.
func normalizeOTLPIDNode(node any) (bool, error) {
	switch v := node.(type) {
	case map[string]any:
		changed := false
		for key, val := range v {
			if wantBytes, isIDField := otlpIDFields[key]; isIDField {
				str, isString := val.(string)
				// An absent or empty ID is legal (protojson decodes it to nil);
				// only a populated string is ours to convert.
				if isString && str != "" {
					raw, err := decodeOTLPIDHex(key, str, wantBytes)
					if err != nil {
						return false, err
					}
					v[key] = base64.StdEncoding.EncodeToString(raw)
					changed = true
					continue
				}
			}
			nested, err := normalizeOTLPIDNode(val)
			if err != nil {
				return false, err
			}
			changed = changed || nested
		}
		return changed, nil

	case []any:
		changed := false
		for _, item := range v {
			nested, err := normalizeOTLPIDNode(item)
			if err != nil {
				return false, err
			}
			changed = changed || nested
		}
		return changed, nil
	}

	return false, nil
}

// decodeOTLPIDHex hex-decodes a single ID field, enforcing its exact length.
//
// Hex is tried unconditionally rather than sniffed, which fixes the precedence
// for values that are simultaneously valid hex and valid base64 (every 32-char
// hex string is one): the OTLP/JSON spec says hex, so hex wins.
func decodeOTLPIDHex(field, value string, wantBytes int) ([]byte, error) {
	if len(value) != wantBytes*2 {
		return nil, &InvalidOTLPIDError{Field: field, Value: value, WantBytes: wantBytes}
	}
	raw, err := hex.DecodeString(value)
	if err != nil {
		return nil, &InvalidOTLPIDError{Field: field, Value: value, WantBytes: wantBytes}
	}
	return raw, nil
}
