package strutil

import (
	"strconv"
	"strings"
	"unicode/utf8"
)

// The quoting rule, shared byte for byte with the Go emitter: this file,
// nfc.go and nfc_tables.go use only the standard library, and
// internal/gen/golang embeds a copy of all three in every compiled program
// (emittedQuoteRuntime, kept current by TestEmittedQuoteRuntimeUpToDate).
// Change the rule here only.

// stringWriter is what QuoteTo writes to (strings.Builder, the show renderer's
// bounded writer). Declared here rather than io.StringWriter so the copy in
// compiled programs needs no extra import.
type stringWriter interface {
	WriteString(s string) (int, error)
}

// QuoteTo writes Quote(s) to w in segments: verbatim runs are written as
// sub-slices of s, so a bounded or counting writer never sees more than the
// escaped text itself. Write errors are ignored; every writer this is used
// with (strings.Builder, the show renderer's bounded writer) never fails.
func QuoteTo(w stringWriter, s string) {
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
	// U+212B ANGSTROM SIGN) is escaped where writing it raw could change it:
	// always if NFC might change s (nfcMayChange: not NFC, or a run of more
	// than 30 non-starters, where the loader inserts U+034F), and right after
	// an escape sequence, whose last ASCII letter it could combine with
	// (`\n` + U+0303 -> `\ñ`). Text NFC leaves alone is otherwise written
	// verbatim, so ordinary text stays readable.
	nfc := !nfcMayChange(s)
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
			r >= utf8.RuneSelf && (!nfc || afterEscape) && !nfcStableStarter(r):
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
