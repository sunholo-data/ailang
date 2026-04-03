package builtins

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/sunholo/ailang/internal/effects"
	"github.com/sunholo/ailang/internal/eval"
	"github.com/sunholo/ailang/internal/types"
)

// ============================================================================
// String Encoding Builtins (M-STD-STRING-PERF)
// ============================================================================

func init() {
	registerDecodeQuotedPrintable()
}

// registerDecodeQuotedPrintable registers the _str_decodeQP builtin
func registerDecodeQuotedPrintable() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/string",
		Name:    "_str_decodeQP",
		NumArgs: 1,
		IsPure:  true,
		Effect:  "",
		Type:    makeDecodeQPType,
		Impl:    decodeQPImpl,

		Metadata: &BuiltinMetadata{
			Description: "Decode quoted-printable encoded string (RFC 2045 §6.7)",
			LongDesc:    "Decodes a quoted-printable encoded string in a single O(n) pass. Handles =XX hex escapes, soft line breaks (=\\r\\n and =\\n), and passes through invalid sequences unchanged. Uses strings.Builder for zero-copy accumulation.",
			Params: []ParamDoc{
				{Name: "s", Description: "Quoted-printable encoded string to decode"},
			},
			Returns: "Decoded string with =XX sequences replaced by their byte values",
			Examples: []Example{
				{Code: `_str_decodeQP("hello=20world")`, Description: `Returns "hello world"`},
				{Code: `_str_decodeQP("line1=\r\nline2")`, Description: `Returns "line1line2" (soft line break removed)`},
				{Code: `_str_decodeQP("=C3=A9")`, Description: `Returns "é" (UTF-8 multi-byte)`},
				{Code: `_str_decodeQP("=GG")`, Description: `Returns "=GG" (invalid hex passed through)`},
			},
			SeeAlso:   []string{"_str_replace", "_str_replaceMany"},
			Since:     "v0.11.0",
			Stability: StabilityStable,
			Tags:      []string{"string", "encoding", "quoted-printable", "email", "mime", "rfc2045"},
			Category:  "string",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _str_decodeQP: %v", err))
	}
}

func makeDecodeQPType() types.Type {
	T := types.NewBuilder()
	return T.Func(T.String()).Returns(T.String()).Build()
}

// decodeQPImpl implements RFC 2045 §6.7 quoted-printable decoding.
//
// Rules:
//   - =XX (where XX are hex digits) → the byte with that hex value
//   - = followed by \r\n or \n (soft line break) → removed entirely
//   - = at end of string → passed through as "="
//   - = followed by non-hex chars → passed through unchanged (e.g., "=GG" → "=GG")
//   - All other bytes → passed through unchanged
func decodeQPImpl(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
	s, err := SafeAsString(args[0])
	if err != nil {
		return nil, fmt.Errorf("_str_decodeQP: arg 0 - %w", err)
	}

	var buf strings.Builder
	buf.Grow(len(s)) // decoded is always <= encoded length

	i := 0
	for i < len(s) {
		if s[i] != '=' {
			buf.WriteByte(s[i])
			i++
			continue
		}

		// We have '=' — check what follows
		remaining := len(s) - i - 1

		if remaining < 1 {
			// '=' at end of string — pass through
			buf.WriteByte('=')
			i++
			continue
		}

		// Check for soft line break: =\r\n or =\n
		if s[i+1] == '\n' {
			// =\n — soft line break, skip both
			i += 2
			continue
		}
		if s[i+1] == '\r' {
			if remaining >= 2 && s[i+2] == '\n' {
				// =\r\n — soft line break, skip all three
				i += 3
			} else {
				// =\r without \n — still treat as soft break
				i += 2
			}
			continue
		}

		// Check for =XX hex escape
		if remaining >= 2 {
			hexStr := s[i+1 : i+3]
			decoded, hexErr := hex.DecodeString(hexStr)
			if hexErr == nil {
				buf.Write(decoded)
				i += 3
				continue
			}
		}

		// Invalid sequence — pass through the '=' and continue
		buf.WriteByte('=')
		i++
	}

	return &eval.StringValue{Value: buf.String()}, nil
}
