package display

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// PrettyJSON formats a JSON string with indentation.
// Returns the original string if it's not valid JSON.
func PrettyJSON(data string) string {
	var out bytes.Buffer
	if err := json.Indent(&out, []byte(data), "", "  "); err != nil {
		return data
	}
	return out.String()
}

// CompactJSON removes whitespace from a JSON string.
// Returns the original string if it's not valid JSON.
func CompactJSON(data string) string {
	var out bytes.Buffer
	if err := json.Compact(&out, []byte(data)); err != nil {
		return data
	}
	return out.String()
}

// FormatJSONValue formats a JSON value as a string for display.
// Truncates long values and handles nested objects.
func FormatJSONValue(val interface{}, maxLen int) string {
	switch v := val.(type) {
	case string:
		return Truncate(v, maxLen)
	case float64:
		return Truncate(formatNumber(v), maxLen)
	case bool:
		if v {
			return "true"
		}
		return "false"
	case nil:
		return "null"
	case map[string]interface{}:
		data, _ := json.Marshal(v)
		return Truncate(string(data), maxLen)
	case []interface{}:
		data, _ := json.Marshal(v)
		return Truncate(string(data), maxLen)
	default:
		data, _ := json.Marshal(v)
		return Truncate(string(data), maxLen)
	}
}

// formatNumber formats a float64 for display.
// Shows integers without decimal places.
func formatNumber(v float64) string {
	// Format as integer if no decimal part
	if v == float64(int64(v)) {
		return fmt.Sprintf("%d", int64(v))
	}
	// For floats, show up to 2 decimal places
	return fmt.Sprintf("%.2f", v)
}

// IsJSON checks if a string is valid JSON.
func IsJSON(s string) bool {
	s = strings.TrimSpace(s)
	if len(s) == 0 {
		return false
	}
	return (s[0] == '{' && s[len(s)-1] == '}') ||
		(s[0] == '[' && s[len(s)-1] == ']')
}

// FormatJSONKeyValue formats a JSON object's key-value pairs for display.
// Each key-value pair is formatted on its own line with truncation.
func FormatJSONKeyValue(data string, maxValueLen int) string {
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(data), &parsed); err != nil {
		return Truncate(data, maxValueLen*3) // Fallback for invalid JSON
	}

	var sb strings.Builder
	for key, val := range parsed {
		valStr := FormatJSONValue(val, maxValueLen)
		sb.WriteString(key)
		sb.WriteString(": ")
		sb.WriteString(valStr)
		sb.WriteString("\n")
	}
	return strings.TrimRight(sb.String(), "\n")
}
