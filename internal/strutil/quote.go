package strutil

import (
	"io"
	"strconv"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

// Quote renders s as a double-quoted AILANG string literal. It is the one
// escaper behind both `ailang fmt` (canonical literals, internal/format) and
// `show` of a string nested inside a value (internal/eval/show.go), so the two
// can never spell the same string differently (M-V1-SHOW-TOTAL).
//
// Escapes, and nothing else is escaped:
//
//	"  \            -> \"  \\
//	LF TAB CR       -> \n  \t  \r
//	$ before {      -> \$   (so the lexer does not read an interpolation)
//	other C0, DEL, C1 and non-graphic runes -> \u{X} (uppercase hex, no padding)
//	a rune NFC could change in place        -> \u{X}: a non-starter or
//	    singleton when s is not NFC, or one that directly follows an escape
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

// QuoteTo writes Quote(s) to w in segments: verbatim runs are written as
// sub-slices of s, so a bounded or counting writer never sees more than the
// escaped text itself. Write errors are ignored; every writer this is used
// with (strings.Builder, the show renderer's bounded writer) never fails.
func QuoteTo(w io.StringWriter, s string) {
	_, _ = w.WriteString(`"`)
	start := 0 // first byte of the pending verbatim run
	flush := func(end int) {
		if end > start {
			_, _ = w.WriteString(s[start:end])
		}
	}
	// The loader NFC-normalises source before lexing (lexer.Normalize), so a
	// literal must survive NFC. A rune that is not an NFC-stable starter
	// (a combining mark, a Hangul vowel/trailing jamo, a singleton such as
	// U+212B ANGSTROM SIGN) is escaped when written raw it could change:
	// always if s itself is not NFC, and right after an escape sequence,
	// whose last ASCII letter it could combine with (`\n` + U+0303 -> `\ñ`).
	// NFC text is otherwise written verbatim, so ordinary text stays readable.
	nfc := norm.NFC.IsNormalString(s)
	afterEscape := false
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		esc := ""
		switch {
		case r == utf8.RuneError && size == 1:
			esc = `\x` + upperHex(uint64(s[i]), 2)
		case r == '"':
			esc = `\"`
		case r == '\\':
			esc = `\\`
		case r == '\n':
			esc = `\n`
		case r == '\t':
			esc = `\t`
		case r == '\r':
			esc = `\r`
		case r == '$':
			if i+1 < len(s) && s[i+1] == '{' {
				esc = `\$`
			}
		case !IsPrintableRune(r),
			r >= utf8.RuneSelf && (!nfc || afterEscape) && !nfcStableStarter(s[i:i+size]):
			esc = `\u{` + upperHex(uint64(r), 0) + `}`
		}
		if esc != "" {
			flush(i)
			_, _ = w.WriteString(esc)
			start = i + size
		}
		afterEscape = esc != ""
		i += size
	}
	flush(len(s))
	_, _ = w.WriteString(`"`)
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

// nfcStableStarter reports whether the rune in r (one UTF-8 sequence) can
// neither combine with a rune before it nor be rewritten by NFC on its own.
func nfcStableStarter(r string) bool {
	return norm.NFC.PropertiesString(r).BoundaryBefore() && norm.NFC.IsNormalString(r)
}

// IsPrintableRune reports whether r can be written verbatim inside a string
// literal. Space is printable; other ASCII control characters, DEL and
// non-graphic Unicode runes (C1 controls, unassigned, format characters) are
// not.
func IsPrintableRune(r rune) bool {
	if r == ' ' {
		return true
	}
	if r < 0x20 || r == 0x7f {
		return false
	}
	return strconv.IsPrint(r)
}

// upperHex spells v in uppercase hex, left-padded with zeros to at least
// width digits.
func upperHex(v uint64, width int) string {
	h := strings.ToUpper(strconv.FormatUint(v, 16))
	for len(h) < width {
		h = "0" + h
	}
	return h
}
