package eval

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
)

// FormatJSONNumber is the single source of truth for the JSON text of a
// float64 (M-JSON-NUMBER-ROUNDTRIP). The registry encoder, the legacy
// evaluator encoder and the bytecode VM encoder all delegate here so the
// three backends emit byte-identical text.
//
// Rules (Go encoding/json / ECMAScript window, plus signed zero):
//   - NaN and ±Inf → "null" (RFC 8259 has no non-finite tokens)
//   - ±0           → "0", or "-0.0" when the sign bit is set (keeps the
//     sign and keeps the token non-integer for syntax-discriminating readers)
//   - 1e-6 <= |f| < 1e21 → shortest round-trip digits, fixed notation
//   - otherwise    → shortest round-trip digits, exponent notation with a
//     two-digit exponent's leading zero stripped (1e-07 → 1e-7)
//
// No float→int conversion is involved, so the output cannot depend on the
// host CPU's out-of-range conversion behaviour.
func FormatJSONNumber(f float64) string {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return "null"
	}
	if f == 0 {
		if math.Signbit(f) {
			return "-0.0"
		}
		return "0"
	}
	abs := math.Abs(f)
	if abs >= 1e-6 && abs < 1e21 {
		return strconv.FormatFloat(f, 'f', -1, 64)
	}
	s := strconv.FormatFloat(f, 'e', -1, 64)
	// Clean up e-09 to e-9 (same cleanup as encoding/json).
	if n := len(s); n >= 4 && s[n-4] == 'e' && s[n-3] == '-' && s[n-2] == '0' {
		s = s[:n-2] + s[n-1:]
	}
	return s
}

// ParseJSONNumber converts a grammar-validated JSON number token to float64.
// Every in-range literal decodes to its nearest float64 (including integer
// literals beyond ±2^63 and "-0", which keeps its sign). A literal outside
// float64 range is an error rather than a silent ±Inf or 0.
func ParseJSONNumber(n json.Number) (float64, error) {
	f, err := strconv.ParseFloat(string(n), 64)
	if err != nil {
		return 0, fmt.Errorf("json: number %s out of float64 range", string(n))
	}
	return f, nil
}
