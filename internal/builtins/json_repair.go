package builtins

import (
	"fmt"
	"strings"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/types"
)

// JSON repair builtin - attempts to fix truncated or malformed JSON strings.
//
// Common issues handled:
//   - Trailing whitespace/padding (Gemini structured output)
//   - Unclosed strings (truncated mid-value)
//   - Unclosed arrays/objects (truncated mid-structure)
//   - Trailing commas before closing brackets
//   - Truncated keywords (tru, fals, nul)

func init() {
	registerJSONRepair()
}

func registerJSONRepair() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/json",
		Name:    "_json_repair",
		NumArgs: 1,
		IsPure:  true,
		Type:    makeJSONRepairType,
		Impl:    jsonRepairImpl,
		Metadata: &BuiltinMetadata{
			Description: "Attempt to repair truncated or malformed JSON",
			LongDesc: `Repairs common JSON truncation issues that occur when AI model output
is cut off by max_tokens limits. Trims whitespace padding, closes unclosed strings,
arrays, and objects, removes trailing commas, and completes truncated keywords.
Returns the repaired string (not parsed Json) so callers can inspect or re-decode.
Returns Result[string, string] - Ok(repaired) on success, Err(message) if unrecoverable.`,
			Params: []ParamDoc{
				{Name: "input", Description: "Potentially truncated JSON string to repair"},
			},
			Returns: "Result[string, string] - Ok(repaired JSON string) on success, Err(error message) if unrecoverable",
			Examples: []Example{
				{Code: `_json_repair("{\"name\": \"Alice\"")`, Description: "Closes unclosed object → Ok(\"{\\\"name\\\": \\\"Alice\\\"}\")"},
				{Code: `_json_repair("[1, 2, 3")`, Description: "Closes unclosed array → Ok(\"[1, 2, 3]\")"},
				{Code: `_json_repair("{\"key\": \"val")`, Description: "Closes unclosed string and object → Ok(\"{\\\"key\\\": \\\"val\\\"}\")"},
				{Code: `_json_repair("{\"a\": 1,}")`, Description: "Removes trailing comma → Ok(\"{\\\"a\\\": 1}\")"},
				{Code: `_json_repair("{\"valid\": true}")`, Description: "Valid JSON passes through → Ok(\"{\\\"valid\\\": true}\")"},
			},
			SeeAlso:   []string{"std/json.decode", "std/json.encode"},
			Since:     "v0.7.3",
			Stability: StabilityStable,
			Tags:      []string{"json", "repair", "parsing", "truncation", "error-recovery"},
			Category:  "json",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _json_repair: %v", err))
	}
}

func makeJSONRepairType() types.Type {
	T := types.NewBuilder()
	// Type signature: string -> Result[string, string]
	resultType := T.App("Result", T.String(), T.String())
	return T.Func(T.String()).Returns(resultType).Build()
}

func jsonRepairImpl(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
	strVal, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("_json_repair: expected string, got %T", args[0])
	}

	repaired, err := repairJSON(strVal.Value)
	if err != nil {
		return &eval.TaggedValue{
			ModulePath: "std/result",
			TypeName:   "Result",
			CtorName:   "Err",
			Fields:     []eval.Value{&eval.StringValue{Value: err.Error()}},
		}, nil
	}

	return &eval.TaggedValue{
		ModulePath: "std/result",
		TypeName:   "Result",
		CtorName:   "Ok",
		Fields:     []eval.Value{&eval.StringValue{Value: repaired}},
	}, nil
}

// repairJSON attempts to fix common JSON truncation issues.
// Returns the repaired JSON string or an error if the input is unrecoverable.
func repairJSON(input string) (string, error) {
	// Step 1: Trim whitespace padding (Gemini structured output pads with spaces)
	s := strings.TrimSpace(input)

	if s == "" {
		return "", fmt.Errorf("empty input")
	}

	// Step 2: Try parsing as-is first. If valid, return immediately.
	if isValidJSON(s) {
		return s, nil
	}

	// Step 3: Scan the string tracking structural state
	repaired, err := scanAndRepair(s)
	if err != nil {
		return "", err
	}

	return repaired, nil
}

// scanAndRepair walks through the JSON string character by character,
// tracking nesting depth and string state, then closes any unclosed structures.
func scanAndRepair(s string) (string, error) {
	var (
		buf       strings.Builder
		stack     []byte // tracks nesting: '{' or '['
		inString  bool
		escaped   bool
		lastNonWS byte // last non-whitespace char written
	)

	buf.Grow(len(s) + 16) // pre-allocate with room for closing brackets

	for i := 0; i < len(s); i++ {
		ch := s[i]

		if inString {
			if escaped {
				buf.WriteByte(ch)
				escaped = false
				continue
			}
			if ch == '\\' {
				buf.WriteByte(ch)
				escaped = true
				continue
			}
			if ch == '"' {
				buf.WriteByte(ch)
				inString = false
				lastNonWS = ch
				continue
			}
			// Check for invalid control characters that would break JSON
			if ch < 0x20 && ch != '\t' && ch != '\n' && ch != '\r' {
				// Skip invalid control chars
				continue
			}
			buf.WriteByte(ch)
			continue
		}

		// Outside string
		switch ch {
		case '"':
			buf.WriteByte(ch)
			inString = true
			lastNonWS = ch
		case '{':
			buf.WriteByte(ch)
			stack = append(stack, '{')
			lastNonWS = ch
		case '}':
			// Remove trailing comma before closing brace
			trimTrailingComma(&buf, &lastNonWS)
			buf.WriteByte(ch)
			if len(stack) > 0 && stack[len(stack)-1] == '{' {
				stack = stack[:len(stack)-1]
			}
			lastNonWS = ch
		case '[':
			buf.WriteByte(ch)
			stack = append(stack, '[')
			lastNonWS = ch
		case ']':
			// Remove trailing comma before closing bracket
			trimTrailingComma(&buf, &lastNonWS)
			buf.WriteByte(ch)
			if len(stack) > 0 && stack[len(stack)-1] == '[' {
				stack = stack[:len(stack)-1]
			}
			lastNonWS = ch
		case ' ', '\t', '\n', '\r':
			buf.WriteByte(ch)
		default:
			buf.WriteByte(ch)
			lastNonWS = ch
		}
	}

	result := buf.String()

	// Step 4: Close any unclosed structures
	if inString {
		// Truncated inside a string — close the string
		// Check if the last byte was a backslash (incomplete escape)
		if escaped {
			// Remove the dangling backslash
			result = result[:len(result)-1]
		}
		result += `"`
		lastNonWS = '"'
	}

	// Complete truncated keywords at the end
	result = completeTruncatedKeyword(result)

	// Remove trailing commas before we close brackets
	result = removeTrailingCommaFromEnd(result)

	// Close unclosed brackets/braces in reverse order
	for i := len(stack) - 1; i >= 0; i-- {
		switch stack[i] {
		case '{':
			// If the last meaningful content is a key with colon but no value,
			// add null before closing
			trimmed := strings.TrimRight(result, " \t\n\r")
			if len(trimmed) > 0 && trimmed[len(trimmed)-1] == ':' {
				result = trimmed + "null"
			}
			result = strings.TrimRight(result, " \t\n\r,") + "}"
		case '[':
			result = strings.TrimRight(result, " \t\n\r,") + "]"
		}
	}

	// Final validation: does it parse now?
	if !isValidJSON(result) {
		return result, fmt.Errorf("repair incomplete: structural damage too severe")
	}

	return result, nil
}

// isValidJSON checks if a string is valid JSON without allocating a full parse result.
func isValidJSON(s string) bool {
	v := &quickValidator{data: []byte(s)}
	return v.validate()
}

// quickValidator does a fast structural JSON validation.
type quickValidator struct {
	data []byte
	pos  int
}

func (v *quickValidator) validate() bool {
	v.skipWS()
	if v.pos >= len(v.data) {
		return false
	}
	if !v.parseValue() {
		return false
	}
	v.skipWS()
	return v.pos == len(v.data) // must consume all input
}

func (v *quickValidator) parseValue() bool {
	if v.pos >= len(v.data) {
		return false
	}
	switch v.data[v.pos] {
	case '"':
		return v.parseString()
	case '{':
		return v.parseObject()
	case '[':
		return v.parseArray()
	case 't':
		return v.parseLiteral("true")
	case 'f':
		return v.parseLiteral("false")
	case 'n':
		return v.parseLiteral("null")
	default:
		return v.parseNumber()
	}
}

func (v *quickValidator) parseString() bool {
	if v.pos >= len(v.data) || v.data[v.pos] != '"' {
		return false
	}
	v.pos++
	for v.pos < len(v.data) {
		ch := v.data[v.pos]
		if ch == '\\' {
			v.pos++
			if v.pos >= len(v.data) {
				return false
			}
			v.pos++
			continue
		}
		if ch == '"' {
			v.pos++
			return true
		}
		if ch < 0x20 {
			return false // unescaped control character
		}
		v.pos++
	}
	return false // unterminated string
}

func (v *quickValidator) parseObject() bool {
	if v.pos >= len(v.data) || v.data[v.pos] != '{' {
		return false
	}
	v.pos++
	v.skipWS()
	if v.pos < len(v.data) && v.data[v.pos] == '}' {
		v.pos++
		return true
	}
	for {
		v.skipWS()
		if !v.parseString() {
			return false
		}
		v.skipWS()
		if v.pos >= len(v.data) || v.data[v.pos] != ':' {
			return false
		}
		v.pos++
		v.skipWS()
		if !v.parseValue() {
			return false
		}
		v.skipWS()
		if v.pos >= len(v.data) {
			return false
		}
		if v.data[v.pos] == '}' {
			v.pos++
			return true
		}
		if v.data[v.pos] != ',' {
			return false
		}
		v.pos++
	}
}

func (v *quickValidator) parseArray() bool {
	if v.pos >= len(v.data) || v.data[v.pos] != '[' {
		return false
	}
	v.pos++
	v.skipWS()
	if v.pos < len(v.data) && v.data[v.pos] == ']' {
		v.pos++
		return true
	}
	for {
		v.skipWS()
		if !v.parseValue() {
			return false
		}
		v.skipWS()
		if v.pos >= len(v.data) {
			return false
		}
		if v.data[v.pos] == ']' {
			v.pos++
			return true
		}
		if v.data[v.pos] != ',' {
			return false
		}
		v.pos++
	}
}

func (v *quickValidator) parseLiteral(lit string) bool {
	if v.pos+len(lit) > len(v.data) {
		return false
	}
	if string(v.data[v.pos:v.pos+len(lit)]) == lit {
		v.pos += len(lit)
		return true
	}
	return false
}

func (v *quickValidator) parseNumber() bool {
	start := v.pos
	if v.pos < len(v.data) && v.data[v.pos] == '-' {
		v.pos++
	}
	if v.pos >= len(v.data) || v.data[v.pos] < '0' || v.data[v.pos] > '9' {
		return false
	}
	for v.pos < len(v.data) && v.data[v.pos] >= '0' && v.data[v.pos] <= '9' {
		v.pos++
	}
	if v.pos < len(v.data) && v.data[v.pos] == '.' {
		v.pos++
		if v.pos >= len(v.data) || v.data[v.pos] < '0' || v.data[v.pos] > '9' {
			return false
		}
		for v.pos < len(v.data) && v.data[v.pos] >= '0' && v.data[v.pos] <= '9' {
			v.pos++
		}
	}
	if v.pos < len(v.data) && (v.data[v.pos] == 'e' || v.data[v.pos] == 'E') {
		v.pos++
		if v.pos < len(v.data) && (v.data[v.pos] == '+' || v.data[v.pos] == '-') {
			v.pos++
		}
		if v.pos >= len(v.data) || v.data[v.pos] < '0' || v.data[v.pos] > '9' {
			return false
		}
		for v.pos < len(v.data) && v.data[v.pos] >= '0' && v.data[v.pos] <= '9' {
			v.pos++
		}
	}
	return v.pos > start
}

func (v *quickValidator) skipWS() {
	for v.pos < len(v.data) && (v.data[v.pos] == ' ' || v.data[v.pos] == '\t' || v.data[v.pos] == '\n' || v.data[v.pos] == '\r') {
		v.pos++
	}
}

// trimTrailingComma removes a trailing comma from the buffer.
func trimTrailingComma(buf *strings.Builder, lastNonWS *byte) {
	if *lastNonWS == ',' {
		s := buf.String()
		// Find and remove the last comma
		idx := strings.LastIndex(s, ",")
		if idx >= 0 {
			buf.Reset()
			buf.WriteString(s[:idx])
			buf.WriteString(s[idx+1:])
			// Update lastNonWS
			trimmed := strings.TrimRight(buf.String(), " \t\n\r")
			if len(trimmed) > 0 {
				*lastNonWS = trimmed[len(trimmed)-1]
			}
		}
	}
}

// removeTrailingCommaFromEnd removes a trailing comma at the end of the string.
func removeTrailingCommaFromEnd(s string) string {
	trimmed := strings.TrimRight(s, " \t\n\r")
	if len(trimmed) > 0 && trimmed[len(trimmed)-1] == ',' {
		return trimmed[:len(trimmed)-1]
	}
	return s
}

// completeTruncatedKeyword fixes truncated JSON keywords at the end of input.
// e.g., "tru" → "true", "fals" → "false", "nul" → "null"
func completeTruncatedKeyword(s string) string {
	trimmed := strings.TrimRight(s, " \t\n\r")
	if len(trimmed) == 0 {
		return s
	}

	// Check each keyword prefix from longest to shortest
	keywords := []string{"true", "false", "null"}
	for _, kw := range keywords {
		for prefixLen := len(kw) - 1; prefixLen >= 2; prefixLen-- {
			prefix := kw[:prefixLen]
			if strings.HasSuffix(trimmed, prefix) {
				// Verify it's a keyword context (preceded by a structural char or start)
				beforeIdx := len(trimmed) - prefixLen
				if beforeIdx == 0 || isStructuralBefore(trimmed[beforeIdx-1]) {
					return trimmed[:len(trimmed)-prefixLen] + kw
				}
			}
		}
	}

	return s
}

// isStructuralBefore returns true if the byte could precede a JSON value.
func isStructuralBefore(ch byte) bool {
	return ch == ':' || ch == ',' || ch == '[' || ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r'
}
