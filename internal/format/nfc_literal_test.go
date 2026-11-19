package format

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/lexer"
)

// stringTokens returns the payloads of the STRING tokens of src as the loader
// reads it (lexer.Normalize first).
func stringTokens(src string) []string {
	l := lexer.New(string(lexer.Normalize([]byte(src))), "test://nfc")
	var out []string
	for tok := l.NextToken(); tok.Type != lexer.EOF; tok = l.NextToken() {
		if tok.Type == lexer.STRING {
			out = append(out, tok.Literal)
		}
	}
	return out
}

// fmt shares strutil.Quote with show, so the NFC rule reaches string literals
// (M-V1-SHOW-TOTAL review item 3). fmt must stay value-preserving and
// idempotent through the loader's normalisation, and must not escape ordinary
// NFC text. Before the rule, fmt wrote `"e\u{301}"` back with a raw combining
// mark, which the next load composed into a different string, and turned
// `"\n\u{303}"` into `"\n` + U+0303, which re-reads as the invalid escape `\ñ`.
func TestFormatStringLiteralsSurviveNFC(t *testing.T) {
	cases := []struct {
		name, lit   string
		mustContain string // verbatim text fmt must keep (readability), or ""
	}{
		{"explicit NFD escape", `"e\u{301}"`, `\u{301}`},
		{"ordinary NFC text stays raw", "\"café 中文\"", "café 中文"},
		{"NFC-stable combining mark stays raw", "\"q\U00000303\"", "q\U00000303"},
		{"combining mark after an escape", `"\n\u{303}"`, `\n\u{303}`},
		{"singleton", `"\u{212B}"`, `\u{212B}`},
		{"jamo pair", `"\u{1100}\u{1161}"`, `\u{1161}`},
		{"lone mark at the start", `"\u{301}x"`, ""},
		{"line separator", `"a\u{2028}b"`, `\u{2028}`},
		// Runs at the stream-safe limit (third review): the loader inserts
		// U+034F after 30 non-starters, so 31 escaped marks must stay escaped
		// and 31 raw ones (which load with a U+034F in them) must survive.
		{"30 escaped marks", `"q` + strings.Repeat(`\u{301}`, 30) + `"`, ""},
		{"31 escaped marks", `"q` + strings.Repeat(`\u{301}`, 31) + `"`, `\u{301}`},
		{"100 escaped mixed marks", `"q` + strings.Repeat(`\u{316}\u{301}`, 50) + `"`, `\u{301}`},
		{"31 raw marks", "\"q" + strings.Repeat("\U00000301", 31) + "\"", ""},
		{"31 trailing jamo", `"` + strings.Repeat(`\u{11A8}`, 31) + `"`, `\u{11A8}`},
		// Stable starters x/text counts as leading non-starters (fourth review).
		{"31 halfwidth voiced marks", `"x` + strings.Repeat(`\u{FF9E}`, 31) + `"`, `\u{FF9E}`},
		{"31 Hangul compatibility vowels", `"x` + strings.Repeat(`\u{314F}`, 31) + `"`, `\u{314F}`},
		// The after-escape rule, for each escape form.
		{"mark after tab", `"\t\u{301}"`, `\t\u{301}`},
		{"mark after CR", `"\r\u{303}"`, `\r\u{303}`},
		{"jamo after backslash", `"\\\u{1161}"`, `\\\u{1161}`},
		{"mark after quote", `"\"\u{303}"`, `\"\u{303}`},
		{"mark after a hex escape", `"\u{1}\u{303}"`, `\u{1}\u{303}`},
	}
	for _, c := range cases {
		src := "module m\n\nexport func f() -> string = " + c.lit + "\n"
		want := stringTokens(src)
		if len(want) != 1 {
			t.Fatalf("%s: source has %d string tokens", c.name, len(want))
		}
		once := formatSrc(t, string(lexer.Normalize([]byte(src))))
		twice := formatSrc(t, string(lexer.Normalize([]byte(once))))
		if once != twice {
			t.Errorf("%s: fmt is not idempotent:\n%s\n%s", c.name, once, twice)
		}
		if got := stringTokens(once); len(got) != 1 || got[0] != want[0] {
			t.Errorf("%s: fmt changed the value: %q -> %q (output %s)", c.name, want[0], got, once)
		}
		if c.mustContain != "" && !strings.Contains(once, c.mustContain) {
			t.Errorf("%s: fmt output %q does not keep %q", c.name, once, c.mustContain)
		}
	}
}

// NFC text in a user's file is left byte-identical by fmt (fourth review:
// fmt rewrote the Arabic literal alef-hamza + fatha into an escape). The
// paragraphs are shared with strutil's readability test.
func TestFormatLeavesNFCParagraphsAlone(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "strutil", "testdata", "nfc_paragraphs.txt"))
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, line := range strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n") {
		name, text, ok := strings.Cut(line, "\t")
		if !ok || strings.HasPrefix(line, "#") {
			continue
		}
		n++
		lit := "\"" + text + "\""
		src := "module m\n\nexport func f() -> string = " + lit + "\n"
		if once := formatSrc(t, string(lexer.Normalize([]byte(src)))); !strings.Contains(once, lit) {
			t.Errorf("%s: fmt rewrote an NFC literal:\n%s", name, once)
		}
	}
	if n != 8 {
		t.Fatalf("%d paragraphs, want 8", n)
	}
}
