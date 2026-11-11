package runner

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/eval"
	"golang.org/x/text/unicode/norm"
)

// asciiLiteral spells s as an AILANG string literal using only ASCII, so NFC
// normalisation of the source cannot touch it: every non-ASCII or special
// rune is a \u{X} escape.
func asciiLiteral(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		if r >= 0x20 && r < 0x7f && r != '"' && r != '\\' && r != '$' {
			b.WriteRune(r)
			continue
		}
		fmt.Fprintf(&b, `\u{%X}`, r)
	}
	b.WriteByte('"')
	return b.String()
}

// Shown text parses back to the same value THROUGH THE REAL LOADER (review
// item 3): the loader NFC-normalises the whole source file, so a nested string
// written raw that NFC changes (NFD text, a combining mark after an escape, a
// singleton like U+212B) used to come back as a different string. The
// originals are built from ASCII-only escapes; the rendering of them is pasted
// into the same program as source; `==` is evaluated by AILANG on every
// engine.
func TestShowOutputSurvivesTheLoaderAllEngines(t *testing.T) {
	strs := []string{
		"e\U00000301",           // NFD é
		"\U000000E9",            // NFC é (stays readable)
		"\U00000301x",           // lone combining mark at the start
		"\n\U00000303",          // combining mark right after an escape
		"q\U00000303",           // NFC-stable, no precomposed form
		"a\U00000302\U00000323", // non-canonical mark order
		"\U0000212B",            // ANGSTROM SIGN -> U+00C5 under NFC
		"\U00002126",            // OHM SIGN -> U+03A9
		"\U00000344",            // decomposes under NFC
		"\U00001100\U00001161",  // Hangul jamo L+V -> syllable
		"\U0000AC00\U000011A8",  // LV syllable + T jamo -> LVT
		"\U0000FB01",            // NFKC-sensitive ligature, NFC-stable
		"\U00002028\U00002029",  // line / paragraph separators
		"\U0000FEFFbom",         // BOM inside a string
		"plain, \"quoted\" \\ ${x} $y 中文 😀", // the ordinary escapes still apply
	}
	vals := make([]eval.Value, len(strs))
	lits := make([]string, len(strs))
	nonNFC := 0
	for i, s := range strs {
		vals[i] = &eval.StringValue{Value: s}
		lits[i] = asciiLiteral(s)
		if !norm.NFC.IsNormalString(s) {
			nonNFC++
		}
	}
	if nonNFC < 5 {
		t.Fatalf("only %d inputs are non-NFC; the test would not exercise normalisation", nonNFC)
	}
	shown := eval.Show(&eval.ListValue{Elements: vals})
	src := "module show_nfc_roundtrip\nimport std/io (println)\n\nexport func main() -> () ! {IO} {\n" +
		"  let orig: [string] = [" + strings.Join(lits, ", ") + "];\n" +
		"  let pasted: [string] = " + shown + ";\n" +
		"  println(show(pasted == orig));\n" +
		"  println(show(pasted))\n}\n"
	want := "true\n" + shown + "\n"
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)
	file := filepath.Join(t.TempDir(), "show_nfc_roundtrip.ail")
	for _, engine := range allEngines {
		t.Run(engine, func(t *testing.T) {
			if got := runOnEngine(t, file, src, engine); got != want {
				t.Fatalf("%s: shown text did not parse back to the same value:\n%s\nsource:\n%s", engine, firstLineDiff(got, want), src)
			}
		})
	}
}
