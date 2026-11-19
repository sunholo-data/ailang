package strutil

import (
	"strings"
	"testing"

	"golang.org/x/text/unicode/norm"
)

// The escape table of Quote, row by row. The lexer round trip (including the
// loader's NFC normalisation) is internal/lexer/quote_roundtrip_test.go.
func TestQuoteTable(t *testing.T) {
	cases := []struct{ in, want string }{
		{"", `""`},
		{"plain text", `"plain text"`},
		{`"`, `"\""`},
		{`\`, `"\\"`},
		{"\n\t\r", `"\n\t\r"`},
		{"${x}", `"\${x}"`},
		{"$x {", `"$x {"`},
		{"$", `"$"`},
		{"\x00\x1b\x7f", `"\u{0}\u{1B}\u{7F}"`},
		{"\U00000085\U0000200B\U00002028\U0000FEFF", `"\u{85}\u{200B}\u{2028}\u{FEFF}"`},
		{"\xff", `"\xFF"`},
		{"a\xc3", `"a\xC3"`},
		// NFC text, including precomposed and NFC-stable combining text,
		// stays verbatim.
		{"caf\U000000E9 \U00004E2D\U00006587 \U0001F600", "\"caf\U000000E9 \U00004E2D\U00006587 \U0001F600\""},
		{"q\U00000303", "\"q\U00000303\""},
		{"\U00000301x", "\"\U00000301x\""},
		// A string that is not NFC: every non-starter / singleton escaped.
		{"e\U00000301", `"e\u{301}"`},
		{"\U0000212B", `"\u{212B}"`},
		{"\U00001100\U00001161", "\"\U00001100\\u{1161}\""},
		// A non-starter right after an escape is escaped even in NFC text.
		{"\n\U00000303", `"\n\u{303}"`},
		{"\"\U00000301", `"\"\u{301}"`},
	}
	for _, c := range cases {
		if got := Quote(c.in); got != c.want {
			t.Errorf("Quote(%q) = %s, want %s", c.in, got, c.want)
		}
		var b strings.Builder
		QuoteTo(&b, c.in)
		if b.String() != Quote(c.in) {
			t.Errorf("QuoteTo and Quote disagree on %q", c.in)
		}
	}
}

// What Quote writes verbatim never changes under NFC in place: normalising
// the whole literal is a no-op whenever the input was valid UTF-8.
func TestQuoteOutputIsNFCStable(t *testing.T) {
	for _, s := range []string{"e\U00000301", "\n\U00000303\U00000323", "\U0000212B\U00002126", "\U0000AC00\U000011A8", "a\U00000302\U00000323", "plain", "caf\U000000E9"} {
		q := Quote(s)
		if n := norm.NFC.String(q); n != q {
			t.Errorf("Quote(%q) = %q changes under NFC to %q", s, q, n)
		}
	}
}

func TestIsPrintableRune(t *testing.T) {
	for r, want := range map[rune]bool{' ': true, 'a': true, '\U000000E9': true, '\n': false, 0x7f: false, 0x85: false, 0x200b: false} {
		if got := IsPrintableRune(r); got != want {
			t.Errorf("IsPrintableRune(%U) = %v, want %v", r, got, want)
		}
	}
}
