package strutil

import (
	"strings"
	"unicode/utf8"
)

// Quote renders s as a double-quoted AILANG string literal. It is the one
// escaper behind both `ailang fmt` (canonical literals, internal/format) and
// `show` of a string nested inside a value (internal/eval/show.go), so the two
// can never spell the same string differently (M-V1-SHOW-TOTAL). The rule
// itself is QuoteTo (quote_rule.go), which the Go emitter copies verbatim.
//
// Escapes, and nothing else is escaped:
//
//	"  \            -> \"  \\
//	LF TAB CR       -> \n  \t  \r
//	$ before {      -> \$   (so the lexer does not read an interpolation)
//	other C0, DEL, C1 and non-graphic runes -> \u{X} (uppercase hex, no padding)
//	a rune NFC could change in place        -> \u{X}: a non-starter or
//	    singleton when NFC might change s (not NFC, or a run of more than
//	    30 non-starters), or one that directly follows an escape
//	a byte that is not valid UTF-8          -> \xHH (uppercase)
//
// Every other rune, including printable non-ASCII text, is written verbatim.
// For valid UTF-8 the result, read back the way the loader reads source
// (BOM strip + NFC, lexer.Normalize, then readEscape), is exactly s. A
// printable rune is never written as \xHH, so the invalid-byte spelling keeps
// the mapping injective even though \xHH re-lexes to the rune U+00HH rather
// than to the raw byte.
func Quote(s string) string {
	var b strings.Builder
	b.Grow(len(s) + 2)
	QuoteTo(&b, s)
	return b.String()
}

// RuneSafeCut returns the largest k <= max at which s can be cut without
// splitting a UTF-8 sequence (it backs off at most 3 bytes; a stray byte
// counts as its own rune). max >= len(s) returns len(s).
func RuneSafeCut(s string, max int) int {
	if max >= len(s) {
		return len(s)
	}
	if max <= 0 {
		return 0
	}
	k := max
	for k > 0 && max-k < utf8.UTFMax-1 && !utf8.RuneStart(s[k]) {
		k--
	}
	if !utf8.RuneStart(s[k]) {
		return max // not UTF-8 here; a byte cut is all there is
	}
	return k
}
