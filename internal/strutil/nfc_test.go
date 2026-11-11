package strutil

import (
	"math/rand"
	"strings"
	"testing"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

// nfcStressPieces are the reviewer's string classes: NFD and NFC text,
// combining marks (in and out of canonical order), Hangul jamo and syllables,
// singletons, compatibility characters, ZWJ sequences, variation selectors,
// line separators, BOM, escape-triggering characters and invalid bytes.
var nfcStressPieces = []string{
	"a", "e", "q", "n", "A", "K", "\"", "\\", "\n", "\t", "$", "{", "${", " ",
	"\U000000E9", "e\U00000301", "\U00000301", "\U00000303", "\U00000308", "\U00000323", "\U00000302", "\U00000345", "\U00000344", "\U00000340", "\U00000F73",
	"\U0000212B", "\U00002126", "\U0000212A", "\U000000C5", "\U00000958", "\U0000FB01", "\U00002160", "\U000000BD", "\U0000FF21", "\U00001E9B",
	"\U00001100", "\U00001161", "\U000011A8", "\U0000AC00", "\U0000AC01", "\U00003131", "\U00001100\U00001161\U000011A8",
	"\U00000928", "\U0000093C", "\U00000929", "\U00000B47", "\U00000B56", "\U00000BC6", "\U00000BBE", "\U000009C7", "\U000009BE", "\U00000D46", "\U00000D3E",
	"\U000005D0", "\U000005B8", "\U000005BC", "\U00000627", "\U0000064E", "\U00000653", "\U00000654",
	"\U0000200D", "\U0000200C", "\U0000FE0F", "\U000e0100", "\U0001f468", "\U00002028", "\U00002029", "\U0000FEFF", "\U000000AD",
	"\U00003099", "\U0000304B", "\U0000304C", "\U000110ba", "\U00011099", "\U0001133e", "\U00011347",
	"\xff", "\xc3", "\xe2\x82",
}

func nfcStressString(r *rand.Rand) string {
	var b strings.Builder
	for n := 1 + r.Intn(7); n > 0; n-- {
		b.WriteString(nfcStressPieces[r.Intn(len(nfcStressPieces))])
	}
	return b.String()
}

// The standard-library implementation answers both NFC questions exactly as
// golang.org/x/text does: for every rune, and for 200 000 random strings.
func TestNFCAgreesWithXText(t *testing.T) {
	for r := rune(0); r <= 0x10FFFF; r++ {
		if r >= 0xD800 && r <= 0xDFFF {
			continue
		}
		s := string(r)
		want := norm.NFC.PropertiesString(s).BoundaryBefore() && norm.NFC.IsNormalString(s)
		if got := nfcStableStarter(r); got != want {
			t.Fatalf("nfcStableStarter(%U) = %v, x/text %v", r, got, want)
		}
		if got, want := nfcString(s), norm.NFC.String(s); got != want {
			t.Fatalf("nfcString(%U) = %+q, x/text %+q", r, got, want)
		}
	}
	rnd := rand.New(rand.NewSource(20261111))
	nonNormal := 0
	for i := 0; i < 200000; i++ {
		s := nfcStressString(rnd)
		want := norm.NFC.IsNormalString(s)
		if got := nfcIsNormal(s); got != want {
			t.Fatalf("nfcIsNormal(%+q) = %v, x/text %v", s, got, want)
		}
		if !want {
			nonNormal++
		}
		if utf8.ValidString(s) {
			if got, w := nfcString(s), norm.NFC.String(s); got != w {
				t.Fatalf("nfcString(%+q) = %+q, x/text %+q", s, got, w)
			}
		}
	}
	if nonNormal < 10000 {
		t.Fatalf("only %d of 200000 strings were non-NFC; the generator is not exercising normalisation", nonNormal)
	}
}
