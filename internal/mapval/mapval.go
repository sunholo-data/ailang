// Package mapval reads typed values out of a map[string]interface{} — a
// decoded JSON object, a Firestore document, an AILANG record crossing the
// embed bridge — without caring which decoder produced it.
//
// The trap it exists for: encoding/json decodes every number as float64,
// Firestore hands back int64, a decoder with UseNumber hands back
// json.Number, and a Go-built map holds int. A reader that type-asserts one
// of those reads the others as 0. cmd/ailang/dashboard.go, the Firestore
// converter and the server's AILANG bridge each carried their own reader
// with a different subset of cases (M-V1-SIMPLIFY-S3 M5 fixed the
// dashboard's; S4 M3B made this the one home). Stdlib leaf.
//
// A missing key, a nil value, or a value of the wrong kind reads as the
// zero value. That is the right answer for optional document fields; a
// caller that must distinguish "absent" from "zero" indexes the map itself.
package mapval

import "encoding/json"

// String returns m[key] when it is a string, else "".
func String(m map[string]interface{}, key string) string {
	s, _ := m[key].(string)
	return s
}

// Bool returns m[key] when it is a bool, else false.
func Bool(m map[string]interface{}, key string) bool {
	b, _ := m[key].(bool)
	return b
}

// Int returns m[key] as an int however the decoder represented the number.
func Int(m map[string]interface{}, key string) int {
	return int(Int64(m, key))
}

// Int64 returns m[key] as an int64 however the decoder represented the
// number. A float is truncated toward zero, as int(f) does.
func Int64(m map[string]interface{}, key string) int64 {
	switch v := m[key].(type) {
	case int:
		return int64(v)
	case int32:
		return int64(v)
	case int64:
		return v
	case float32:
		return int64(v)
	case float64:
		return int64(v)
	case json.Number:
		if i, err := v.Int64(); err == nil {
			return i
		}
		if f, err := v.Float64(); err == nil {
			return int64(f)
		}
	}
	return 0
}

// Float returns m[key] as a float64 however the decoder represented the
// number.
func Float(m map[string]interface{}, key string) float64 {
	switch v := m[key].(type) {
	case int:
		return float64(v)
	case int32:
		return float64(v)
	case int64:
		return float64(v)
	case float32:
		return float64(v)
	case float64:
		return v
	case json.Number:
		if f, err := v.Float64(); err == nil {
			return f
		}
	}
	return 0
}

// Strings returns m[key] as a []string whether it arrived as a real
// []string or as the []interface{} every decoder produces for an array.
// Non-string elements are dropped. A missing key is nil, so omitempty works.
func Strings(m map[string]interface{}, key string) []string {
	switch arr := m[key].(type) {
	case []string:
		return arr
	case []interface{}:
		out := make([]string, 0, len(arr))
		for _, x := range arr {
			if s, ok := x.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}
