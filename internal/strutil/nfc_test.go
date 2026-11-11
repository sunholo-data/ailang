package strutil

import (
	"math/rand"
	"strings"
	"testing"
	"time"

	"golang.org/x/text/unicode/norm"
)

// stressPieces: NFD and NFC text, combining marks of several classes,
// Hangul jamo and syllables, singletons, compatibility characters, ZWJ,
// variation selectors, separators, BOM, escape-triggering characters and
// invalid bytes. Written with \U escapes so the source file is ASCII.
var stressPieces = []string{
	"a", "e", "q", "n", "A", "\"", "\\", "\n", "$", "{", "${", " ",
	"\U000000E9", "e\U00000301", "\U00000301", "\U00000303", "\U00000308", "\U00000316", "\U00000323", "\U00000302",
	"\U00000345", "\U00000344", "\U00000340", "\U00000F73", "\U0000034F", "\U000000C0", "\U00001EA1",
	"\U0000212B", "\U00002126", "\U00000958", "\U0000FB01", "\U000000BD", "\U0000FF21",
	"\U00001100", "\U00001161", "\U000011A8", "\U0000AC00", "\U0000AC01",
	"\U00000928", "\U0000093C", "\U00000915", "\U0000094D", "\U00000BC6", "\U00000BBE", "\U000009BE",
	"\U000005D0", "\U000005B8", "\U00000627", "\U0000064E", "\U00000651", "\U00000654",
	"\U0000200D", "\U0000FE0F", "\U000E0100", "\U0001F468", "\U00002028", "\U0000FEFF", "\U00003099", "\U0000304B",
	"\xff", "\xc3",
}

// Non-starters as x/text counts them: marks, and starters that combine
// backward (Hangul vowels and trailing jamo, U+0BBE) or whose compatibility
// decomposition ends in a mark (U+00A8, U+0385).
var stressMarks = []string{"\U00000301", "\U00000316", "\U00000323", "\U00000302", "\U0000064E", "\U000005B8", "\U00003099",
	"\U00001161", "\U000011A8", "\U00000BBE", "\U000000A8", "\U00000385"}

// stressString: mostly short mixes, sometimes long strings and long runs of
// marks (up to a few thousand runes).
func stressString(r *rand.Rand) string {
	var b strings.Builder
	n := 1 + r.Intn(8)
	if r.Intn(10) == 0 {
		n = 50 + r.Intn(800)
	}
	for ; n > 0; n-- {
		if r.Intn(25) == 0 {
			run := 1 + r.Intn(60)
			if r.Intn(20) == 0 {
				run = 100 + r.Intn(3000)
			}
			for k := 0; k < run; k++ {
				b.WriteString(stressMarks[r.Intn(len(stressMarks))])
			}
			continue
		}
		b.WriteString(stressPieces[r.Intn(len(stressPieces))])
	}
	return b.String()
}

// nfcStableStarter is exact on every rune; nfcMayChange is conservative:
// whenever it says NFC leaves a string alone, x/text must leave it alone.
func TestNFCAgreesWithXText(t *testing.T) {
	var interesting []rune
	for r := rune(0); r <= 0x10FFFF; r++ {
		if r >= 0xD800 && r <= 0xDFFF {
			continue
		}
		s := string(r)
		want := norm.NFC.PropertiesString(s).BoundaryBefore() && norm.NFC.IsNormalString(s)
		if got := nfcStableStarter(r); got != want {
			t.Fatalf("nfcStableStarter(%U) = %v, x/text %v", r, got, want)
		}
		if !nfcMayChange(s) && norm.NFC.String(s) != s {
			t.Fatalf("nfcMayChange(%U) = false, but NFC changes it", r)
		}
		if !want {
			interesting = append(interesting, r)
		}
	}
	// Every pair of runes that are not stable starters, after a starter.
	pairs := 0
	for _, a := range interesting {
		for _, b := range interesting {
			if (a+b)%3 != 0 { // a third of the pairs keeps the test under a few seconds
				continue
			}
			s := "a" + string(a) + string(b)
			if !nfcMayChange(s) && norm.NFC.String(s) != s {
				t.Fatalf("nfcMayChange(%+q) = false, but NFC changes it to %+q", s, norm.NFC.String(s))
			}
			pairs++
		}
	}
	rnd := rand.New(rand.NewSource(20261111))
	safe, long := 0, 0
	for i := 0; i < 200000; i++ {
		s := stressString(rnd)
		if len(s) > 1000 {
			long++
		}
		if !nfcMayChange(s) {
			safe++
			if norm.NFC.String(s) != s {
				t.Fatalf("nfcMayChange(%+q) = false, but NFC changes it", s)
			}
		}
	}
	if safe < 20000 || long < 5000 {
		t.Fatalf("generator too weak: %d safe, %d long strings of 200000", safe, long)
	}
	t.Logf("%d runes, %d pairs, 200000 strings (%d long, %d judged unchanged)", len(interesting), pairs, long, safe)
}

// The stream-safe limit is counted as x/text counts it: a run of up to 30
// non-starters is left alone, a 31st gets U+034F inserted.
func TestNFCStreamSafeBoundary(t *testing.T) {
	for _, start := range []string{"q", "\U000000C0", "\U00001EA1", "", "\U00000301", "\U000000A8", "\U00001161", "\U0000AC01"} {
		for n := 0; n <= 40; n++ {
			for _, mark := range []string{"\U00000301", "\U00000316"} {
				s := start + strings.Repeat(mark, n) + "x"
				changes := norm.NFC.String(s) != s
				got := nfcMayChange(s)
				if !got && changes {
					t.Fatalf("%+q + %d x %U: nfcMayChange = false, but NFC changes it", start, n, []rune(mark)[0])
				}
				// Unless the start composes with the mark, a repeated mark can
				// only change through the stream-safe limit, so the answer
				// must be exact there: the count matches x/text's.
				if start != "\U000000C0" && start != "\U00001EA1" && got != changes {
					t.Fatalf("%+q + %d x %U: nfcMayChange = %v, NFC changes it = %v", start, n, []rune(mark)[0], got, changes)
				}
			}
		}
	}
}

// Ordinary NFC text in several scripts is written verbatim.
func TestQuoteKeepsOrdinaryTextReadable(t *testing.T) {
	for name, para := range map[string]string{
		"Korean":     "\U0000D55C\U0000AD6D\U0000C5B4\U0000B294 \U0000C544\U0000B984\U0000B2E4\U0000C6B4 \U0000C5B8\U0000C5B4\U0000C785\U0000B2C8\U0000B2E4.",
		"Vietnamese": "Ti\U00001EBFng Vi\U00001EC7t c\U000000F3 nhi\U00001EC1u d\U000000E2\U00001EA5u thanh \U00001EDF \U00000111\U000000E2y.",
		"Devanagari": "\U00000939\U0000093F\U00000928\U0000094D\U00000926\U00000940 \U00000915\U00000940 \U0000091C\U0000093C\U00000930\U0000093E \U00000926\U00000947\U00000916\U00000947\U00000902\U00000964",
		"Arabic":     "\U00000627\U00000644\U00000644\U0000064F\U0000063A\U0000064E\U00000629\U0000064F \U00000627\U00000644\U0000064A\U0000064E\U00000648\U0000064E\U00000645\U0000064E",
		"Thai":       "\U00000E20\U00000E32\U00000E29\U00000E32\U00000E44\U00000E17\U00000E22 \U00000E2A\U00000E27\U00000E31\U00000E2A\U00000E14\U00000E35",
		"emoji":      "\U0001F468\U0001F469\U0001F467 \U0001F44D\U0001F3FD \U00002764\U0000FE0F \U0001F389", // ZWJ is a format character: escaped by the non-printing rule, not by NFC
	} {
		para = norm.NFC.String(para)
		if nfcMayChange(para) {
			t.Errorf("%s: NFC paragraph judged changeable: %+q", name, para)
		}
		if q := Quote(para); strings.Contains(q, `\u{`) {
			t.Errorf("%s: escaped: %s", name, q)
		}
	}
}

// Quoting is linear: no sort over a run, no normalisation. 1M marks quote in
// well under the bound, and 10x the input costs about 10x, not 100x.
func TestQuoteLinearTime(t *testing.T) {
	marks := func(n int) string {
		var b strings.Builder
		b.WriteString("q")
		for i := 0; i < n; i++ {
			if i%2 == 0 {
				b.WriteString("\U00000301")
			} else {
				b.WriteString("\U00000316")
			}
		}
		return b.String()
	}
	timeIt := func(s string) time.Duration {
		start := time.Now()
		q := Quote(s)
		d := time.Since(start)
		if !strings.Contains(q, `\u{301}`) {
			t.Fatal("a long run of marks must be escaped")
		}
		return d
	}
	small, big := marks(100000), marks(1000000)
	timeIt(small) // warm up
	t100k, t1m := timeIt(small), timeIt(big)
	if t1m > 5*time.Second {
		t.Fatalf("quoting 1M marks took %v", t1m)
	}
	if t100k > 0 && t1m > 40*t100k {
		t.Fatalf("not linear: 100k marks %v, 1M marks %v", t100k, t1m)
	}
	t.Logf("100k marks %v, 1M marks %v", t100k, t1m)
}

// nfcNonStarters counts exactly as x/text does, measured from its behaviour,
// not from the rule the generator applies: for every rune, the number of
// U+0316 marks before (after) it at which x/text starts inserting U+034F.
func TestNFCNonStartersMatchXText(t *testing.T) {
	const mark = "\U00000316"
	overflows := func(s string) bool {
		return strings.Count(norm.NFC.String(s), "\U0000034F") > strings.Count(s, "\U0000034F")
	}
	measure := func(build func(k int) string) int {
		n := 0
		for k := 30; k >= 27 && overflows(build(k)); k-- {
			n = 31 - k
		}
		return n
	}
	for r := rune(0); r <= 0x10FFFF; r++ {
		if (r >= 0xD800 && r <= 0xDFFF) || r == 0x34F { // U+034F is the probe's own marker
			continue
		}
		lead := measure(func(k int) string { return "a" + strings.Repeat(mark, k) + string(r) })
		trail := measure(func(k int) string { return string(r) + strings.Repeat(mark, k) + "x" })
		if gl, gt := nfcNonStarters(r); gl != lead || gt != trail {
			t.Fatalf("nfcNonStarters(%U) = %d/%d, x/text counts %d/%d", r, gl, gt, lead, trail)
		}
	}
}
