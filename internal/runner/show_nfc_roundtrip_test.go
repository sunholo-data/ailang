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
		"e\U00000301",           // NFD e-acute
		"\U000000E9",            // NFC e-acute (stays readable)
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
		"plain, \"quoted\" \\ ${x} $y \U00004E2D\U00006587 \U0001F600", // the ordinary escapes still apply
	}
	// Long runs of marks (review 3): the loader's normaliser inserts U+034F
	// after 30 non-starters, so a run that long must not be written raw.
	acute, below := "\U00000301", "\U00000316"
	for _, n := range []int{1, 29, 30, 31, 32, 100, 100000} {
		strs = append(strs,
			"q"+strings.Repeat(acute, n)+"x",              // starter before and after
			strings.Repeat(acute, n),                      // at the start of the string
			"\n"+strings.Repeat(acute+below, (n+1)/2)+"x", // after an escape, mixed classes
		)
	}
	strs = append(strs,
		strings.Repeat("\U00001100\U00001161", 40), // Hangul jamo run (L V pairs)
		strings.Repeat("\U000011A8", 40),           // trailing jamo run
		"\U00001EA1"+strings.Repeat(acute, 30)+"x", // composite with a trailing mark, then 30
		"\U000000C0"+strings.Repeat(below, 29)+"x", // composite with a mark of a lower class
	)
	// Fourth review: stable starters that x/text counts as leading
	// non-starters (U+FF9E, Hangul compatibility vowels), in runs past the
	// limit, alone and mixed with marks; and every escape form followed by a
	// mark, a jamo and one of those runes.
	for _, z := range []string{"\U0000FF9E", "\U00003133", "\U0000314F", "\U0000FFA3", "\U0000FFDC"} {
		strs = append(strs, "x"+strings.Repeat(z, 31), strings.Repeat(z+acute, 16)+"y", "x"+strings.Repeat(z, 30)+"y")
	}
	for _, esc := range []string{"\n", "\t", "\r", "\\", "\"", "\x01", "\U0000200B"} {
		for _, f := range []string{"\U00000303", "\U00001161", "\U000011A8", "\U0000FF9E"} {
			strs = append(strs, "a"+esc+f+"b")
		}
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
