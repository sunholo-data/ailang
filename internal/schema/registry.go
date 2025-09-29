// Package schema provides centralized JSON schema versioning and validation
// for AILANG's AI-first features.
package schema

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// Schema version constants
const (
	ErrorV1     = "ailang.error/v1"
	TestV1      = "ailang.test/v1"
	DecisionsV1 = "ailang.decisions/v1"
	PlanV1      = "plan/v1"
	EffectsV1   = "ailang.effects/v1"
)

// Accepts checks if a schema version is compatible with the expected version.
// Supports forward compatibility within major versions (e.g., v1.x accepts v1.0).
func Accepts(got, wantPrefix string) bool {
	if got == wantPrefix {
		return true
	}
	// Check if got is a compatible sub-version
	if strings.HasPrefix(got, wantPrefix+".") {
		return true
	}
	// Check if wantPrefix is requesting a major version and got matches
	if strings.HasSuffix(wantPrefix, "/v1") && strings.HasPrefix(got, strings.TrimSuffix(wantPrefix, "1")+"1.") {
		return true
	}
	return false
}

// MarshalDeterministic marshals a value to JSON with sorted keys for deterministic output.
func MarshalDeterministic(v any) ([]byte, error) {
	// First marshal to get a map (without HTML escaping)
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, fmt.Errorf("initial marshal failed: %w", err)
	}
	data := buf.Bytes()
	// Remove trailing newline added by Encode
	if len(data) > 0 && data[len(data)-1] == '\n' {
		data = data[:len(data)-1]
	}

	// Unmarshal to a generic map to sort keys
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		// Not a map, return as-is
		return data, nil
	}

	// Sort and re-marshal
	return marshalSorted(m)
}

// marshalSorted recursively marshals maps with sorted keys
func marshalSorted(v any) ([]byte, error) {
	switch val := v.(type) {
	case map[string]any:
		// Sort keys
		keys := make([]string, 0, len(val))
		for k := range val {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		// Build ordered map
		result := "{"
		for i, k := range keys {
			if i > 0 {
				result += ","
			}
			// Marshal key without HTML escaping
			var keyBuf bytes.Buffer
			keyEnc := json.NewEncoder(&keyBuf)
			keyEnc.SetEscapeHTML(false)
			if err := keyEnc.Encode(k); err != nil {
				return nil, err
			}
			keyJSON := keyBuf.Bytes()
			// Remove trailing newline
			if len(keyJSON) > 0 && keyJSON[len(keyJSON)-1] == '\n' {
				keyJSON = keyJSON[:len(keyJSON)-1]
			}

			valJSON, err := marshalSorted(val[k])
			if err != nil {
				return nil, err
			}
			result += string(keyJSON) + ":" + string(valJSON)
		}
		result += "}"
		return []byte(result), nil

	case []any:
		// Process arrays recursively
		result := "["
		for i, item := range val {
			if i > 0 {
				result += ","
			}
			itemJSON, err := marshalSorted(item)
			if err != nil {
				return nil, err
			}
			result += string(itemJSON)
		}
		result += "]"
		return []byte(result), nil

	default:
		// Use encoder without HTML escaping for primitives
		var buf bytes.Buffer
		enc := json.NewEncoder(&buf)
		enc.SetEscapeHTML(false)
		if err := enc.Encode(v); err != nil {
			return nil, err
		}
		// Remove trailing newline added by Encode
		result := buf.Bytes()
		if len(result) > 0 && result[len(result)-1] == '\n' {
			result = result[:len(result)-1]
		}
		return result, nil
	}
}

// MustValidate validates a value against a schema.
// Currently a no-op placeholder for future schema validation.
func MustValidate(schemaName string, v any) error {
	// TODO: Implement actual schema validation when needed
	// For now, just check that the schema field matches if present
	if m, ok := v.(map[string]any); ok {
		if schema, ok := m["schema"].(string); ok {
			if !Accepts(schema, schemaName) {
				return fmt.Errorf("schema mismatch: got %q, want %q", schema, schemaName)
			}
		}
	}
	return nil
}

// Compact option for JSON output
var CompactMode = false

// SetCompactMode enables or disables compact JSON output
func SetCompactMode(enabled bool) {
	CompactMode = enabled
}

// FormatJSON formats JSON according to compact mode setting
func FormatJSON(data []byte) ([]byte, error) {
	if CompactMode {
		var buf bytes.Buffer
		err := json.Compact(&buf, data)
		if err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	}
	var prettyBuf bytes.Buffer
	err := json.Indent(&prettyBuf, data, "", "  ")
	if err != nil {
		return nil, err
	}
	return prettyBuf.Bytes(), nil
}
