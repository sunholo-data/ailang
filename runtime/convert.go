package runtime

// ConvertToInt64Slice converts []any to []int64.
// Used when ADT constructors expect [int] fields.
func ConvertToInt64Slice(v any) []int64 {
	if v == nil {
		return nil
	}
	src, ok := v.([]any)
	if !ok {
		// Try []interface{} for compatibility
		if srcIface, ok := v.([]interface{}); ok {
			dst := make([]int64, len(srcIface))
			for i, elem := range srcIface {
				dst[i] = toInt64(elem)
			}
			return dst
		}
		return nil
	}
	dst := make([]int64, len(src))
	for i, elem := range src {
		dst[i] = toInt64(elem)
	}
	return dst
}

// ConvertToStringSlice converts []any to []string.
// Used when ADT constructors expect [string] fields.
func ConvertToStringSlice(v any) []string {
	if v == nil {
		return nil
	}
	src, ok := v.([]any)
	if !ok {
		if srcIface, ok := v.([]interface{}); ok {
			dst := make([]string, len(srcIface))
			for i, elem := range srcIface {
				dst[i] = toString(elem)
			}
			return dst
		}
		return nil
	}
	dst := make([]string, len(src))
	for i, elem := range src {
		dst[i] = toString(elem)
	}
	return dst
}

// ConvertToRecordSlice converts []any to []map[string]any.
// Used when ADT constructors expect [{...}] record fields.
func ConvertToRecordSlice(v any) []map[string]any {
	if v == nil {
		return nil
	}
	src, ok := v.([]any)
	if !ok {
		if srcIface, ok := v.([]interface{}); ok {
			dst := make([]map[string]any, len(srcIface))
			for i, elem := range srcIface {
				dst[i] = toRecord(elem)
			}
			return dst
		}
		return nil
	}
	dst := make([]map[string]any, len(src))
	for i, elem := range src {
		dst[i] = toRecord(elem)
	}
	return dst
}

// toInt64 converts any value to int64.
func toInt64(v any) int64 {
	switch x := v.(type) {
	case int64:
		return x
	case int:
		return int64(x)
	case float64:
		return int64(x)
	default:
		return 0
	}
}

// toString converts any value to string.
func toString(v any) string {
	switch x := v.(type) {
	case string:
		return x
	default:
		return Show(v)
	}
}

// toRecord converts any value to map[string]any.
// Note: map[string]any and map[string]interface{} are the same type in Go 1.18+.
func toRecord(v any) map[string]any {
	if x, ok := v.(map[string]any); ok {
		return x
	}
	return nil
}
