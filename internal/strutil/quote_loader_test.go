package strutil_test

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/lexer"
	"github.com/sunholo-data/ailang/internal/strutil"
)

// loadString reads src the way the loader reads source (lexer.Normalize:
// BOM strip + NFC, then the lexer) and returns its one string literal.
func loadString(src string) (string, bool) {
	l := lexer.New(string(lexer.Normalize([]byte(src))), "quote.ail")
	tok := l.NextToken()
	if tok.Type != lexer.STRING || l.NextToken().Type != lexer.EOF {
		return "", false
	}
	return tok.Literal, true
}

func checkLoads(t *testing.T, s string) {
	t.Helper()
	q := strutil.Quote(s)
	if got, ok := loadString(q); !ok || got != s {
		t.Fatalf("Quote(%+q) = %s does not load back (got %+q, ok %v)", s, q, got, ok)
	}
}

// Every rune x/text counts as a non-starter (nfcNonStarters, from the
// generated tables, not a hand list), in runs at the stream-safe limit, alone
// and mixed with ordinary marks, survives the loader (fourth review of
// #2229: U+FF9E and the Hangul compatibility vowels are stable starters with
// a leading count, and were written raw past the limit).
func TestQuoteNonStarterRunsSurviveTheLoader(t *testing.T) {
	leading, trailingOnly := 0, 0
	for r := rune(0xA0); r <= 0x10FFFF; r++ {
		if r >= 0xD800 && r <= 0xDFFF {
			continue
		}
		lead, trail := strutil.NFCNonStarters(r)
		if lead == 0 && trail == 0 {
			continue
		}
		c := string(r)
		if lead == 0 {
			trailingOnly++ // e.g. a Hangul syllable or U+00A8: the run follows it
			for _, n := range []int{29, 30, 31} {
				checkLoads(t, "x"+c+strings.Repeat("\U00000301", n)+"y")
			}
			continue
		}
		leading++
		for _, n := range []int{30, 31, 32} {
			checkLoads(t, "x"+strings.Repeat(c, n)+"y")
			checkLoads(t, strings.Repeat(c, n))
			checkLoads(t, "x"+strings.Repeat(c+"\U00000316", (n+1)/2)+"y")
			checkLoads(t, "\n"+strings.Repeat("\U00000301"+c, (n+1)/2))
		}
	}
	if leading < 1000 || trailingOnly < 11172 {
		t.Fatalf("enumerated %d leading and %d trailing-only non-starters: the table walk is broken", leading, trailingOnly)
	}
	t.Logf("%d runes with a leading count, %d trailing-only", leading, trailingOnly)
}

// The after-escape rule: a rune that could combine with the last ASCII
// letter of an escape (`\n` + U+0303 would load as `\` + n-tilde, an invalid escape)
// is escaped, for every escape form the quoter writes.
func TestQuoteAfterEscape(t *testing.T) {
	escapes := map[string]string{
		"\n": `\n`, "\t": `\t`, "\r": `\r`, "\\": `\\`, "\"": `\"`,
		"\x01": `\u{1}`, "\U0000200B": `\u{200B}`, "\xff": `\xFF`,
	}
	followers := map[string]string{
		"\U00000303": `\u{303}`, "\U00000301": `\u{301}`, "\U00001161": `\u{1161}`,
		"\U000011A8": `\u{11A8}`, "\U0000FF9E": `\u{FF9E}`, "\U00003099": `\u{3099}`,
	}
	for in, esc := range escapes {
		for f, fesc := range followers {
			s := "a" + in + f + "b"
			q := strutil.Quote(s)
			if !strings.Contains(q, esc+fesc) {
				t.Errorf("Quote(%+q) = %s, want %s%s", s, q, esc, fesc)
			}
			if in != "\xff" { // an invalid byte loads as U+00FF by design
				checkLoads(t, s)
			}
		}
	}
	// `${` is written `\$` + `{`: the brace is not an escape, and a mark after
	// it is NFC-stable, so it stays raw and still loads back.
	checkLoads(t, "${\U00000303}")
	checkLoads(t, "$\U00000303{")
}
