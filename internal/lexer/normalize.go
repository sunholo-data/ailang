package lexer

import (
	"bytes"

	"golang.org/x/text/unicode/norm"
)

// bomUTF8 is the UTF-8 Byte Order Mark
var bomUTF8 = []byte{0xEF, 0xBB, 0xBF}

// Normalize performs input normalization at the lexer boundary:
// 1. Strips UTF-8 BOM if present
// 2. Applies Unicode NFC normalization
//
// This ensures that lexically equivalent source code produces identical
// token streams regardless of encoding variations.
//
// Examples:
//   - "café" in NFC vs NFD → identical tokens
//   - "\uFEFF let x = 5" → "let x = 5" (BOM stripped)
//
// Normalization is performed once at input to avoid repeated processing.
func Normalize(src []byte) []byte {
	// Strip BOM if present
	src = bytes.TrimPrefix(src, bomUTF8)

	// Apply NFC normalization
	// IsNormal() is fast and avoids allocation if already normalized
	if !norm.NFC.IsNormal(src) {
		src = norm.NFC.Bytes(src)
	}

	return src
}
