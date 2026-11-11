package lexer

import (
	"math/rand"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/sunholo-data/ailang/internal/strutil"
)

// strutil.Quote is the escaper behind `ailang fmt` and behind `show` of a
// nested string (M-V1-SHOW-TOTAL). Its contract: for valid UTF-8, the quoted
// form lexes back to one STRING token whose payload is exactly the input. The
// fragments are the characters every escape rule exists for.
var quoteFragments = []string{
	"", "a", "Z", " ", `"`, `\`, `\"`, `\\`, ", ", "${", "$", "{", "}", "$${", `\${`,
	"\n", "\t", "\r", "\x00", "\x01", "\x1b", "\x7f", "\u0085", "\u00a0", "\u200b", "\u2028", "\ufeff",
	"é", "中文", "😀", "\U0010FFFF", "\ufffd", "[", "]", "(", ")", ": ", "Some(", "x", "\\x41", "\\u{41}",
}

func randomQuoteInput(r *rand.Rand) string {
	var b strings.Builder
	for n := r.Intn(6); n > 0; n-- {
		b.WriteString(quoteFragments[r.Intn(len(quoteFragments))])
	}
	return b.String()
}

func lexOneString(t *testing.T, src string) (string, bool) {
	t.Helper()
	l := New(src, "quote.ail")
	tok := l.NextToken()
	if tok.Type != STRING {
		return "", false
	}
	if next := l.NextToken(); next.Type != EOF {
		return "", false
	}
	return tok.Literal, true
}

func TestQuoteRoundTripsThroughTheLexer(t *testing.T) {
	r := rand.New(rand.NewSource(20261111))
	inputs := append([]string{}, quoteFragments...)
	for i := 0; i < 20000; i++ {
		inputs = append(inputs, randomQuoteInput(r))
	}
	for _, s := range inputs {
		if !utf8.ValidString(s) {
			continue
		}
		q := strutil.Quote(s)
		got, ok := lexOneString(t, q)
		if !ok {
			t.Fatalf("Quote(%q) = %s does not lex as one STRING token", s, q)
		}
		if got != s {
			t.Fatalf("Quote(%q) = %s lexes back to %q", s, q, got)
		}
	}
}

// Invalid UTF-8 is the one documented exception: each bad byte is spelled
// \xHH (still valid syntax), which lexes to the rune U+00HH, not the byte.
func TestQuoteInvalidUTF8IsEscapedAndLexes(t *testing.T) {
	for s, want := range map[string]string{
		"\xff":       `"\xFF"`,
		"a\xc3":      `"a\xC3"`,
		"\xc3\xa9":   `"é"`,
		"\xe2\x82":   `"\xE2\x82"`,
		"ok\x80\x80": `"ok\x80\x80"`,
	} {
		q := strutil.Quote(s)
		if q != want {
			t.Errorf("Quote(%q) = %s, want %s", s, q, want)
		}
		if !utf8.ValidString(q) {
			t.Errorf("Quote(%q) = %q is not valid UTF-8", s, q)
		}
		if _, ok := lexOneString(t, q); !ok {
			t.Errorf("Quote(%q) = %s does not lex", s, q)
		}
	}
}

// The escape table, row by row (design doc "Nested string escaping").
func TestQuoteEscapeTable(t *testing.T) {
	for s, want := range map[string]string{
		`"`:            `"\""`,
		`\`:            `"\\"`,
		"\n\t\r":       `"\n\t\r"`,
		"${x}":         `"\${x}"`,
		"$x {":         `"$x {"`,
		"$":            `"$"`,
		"\x00\x1b\x7f": `"\u{0}\u{1B}\u{7F}"`,
		"\u0085":       `"\u{85}"`,
		"\u200b":       `"\u{200B}"`,
		"é 😀 中":        `"é 😀 中"`,
		"":             `""`,
	} {
		if got := strutil.Quote(s); got != want {
			t.Errorf("Quote(%q) = %s, want %s", s, got, want)
		}
	}
}
