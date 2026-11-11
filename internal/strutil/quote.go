package strutil

import (
	"io"
	"strconv"
	"strings"
	"unicode/utf8"
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
//	a byte that is not valid UTF-8          -> \xHH (uppercase)
//
// Every other rune, including printable non-ASCII text, is written verbatim.
// For valid UTF-8 the result re-lexes (lexer readEscape) to exactly s. A
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
		case !IsPrintableRune(r):
			esc = `\u{` + upperHex(uint64(r), 0) + `}`
		}
		if esc != "" {
			flush(i)
			_, _ = w.WriteString(esc)
			start = i + size
		}
		i += size
	}
	flush(len(s))
	_, _ = w.WriteString(`"`)
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
