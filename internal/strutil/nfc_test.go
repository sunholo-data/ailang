package strutil

import (
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

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

// nfcParagraphs reads testdata/nfc_paragraphs.txt (shared with the fmt test).
func nfcParagraphs(t *testing.T) map[string]string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "nfc_paragraphs.txt"))
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]string{}
	for _, line := range strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n") {
		if name, text, ok := strings.Cut(line, "\t"); ok && !strings.HasPrefix(line, "#") {
			out[name] = text
		}
	}
	if len(out) != 8 {
		t.Fatalf("%d paragraphs, want 8", len(out))
	}
	return out
}

// Ordinary NFC text is written verbatim: ZERO escaped runes in a real
// paragraph of each script (fourth review: Arabic with harakat had 42% of its
// runes escaped, because a haraka after U+0622-0626 was judged a reordering).
func TestQuoteKeepsOrdinaryTextReadable(t *testing.T) {
	for name, para := range nfcParagraphs(t) {
		if norm.NFC.String(para) != para {
			t.Fatalf("%s: test paragraph is not NFC", name)
		}
		if nfcMayChange(para) {
			t.Errorf("%s: NFC paragraph judged changeable", name)
		}
		if q, want := Quote(para), `"`+para+`"`; q != want {
			t.Errorf("%s: %d of %d runes escaped: %s", name, strings.Count(q, `\u{`), utf8.RuneCountInString(para), q)
		}
	}
}

// Exact, not just conservative, on a finite class: every precomposed
// character whose decomposition ends in a mark, followed by every mark.
// nfcMayChange must say "no change" exactly where x/text makes none, so
// over-escaping cannot creep back in for this class.
func TestNFCCompositePlusMarkExact(t *testing.T) {
	var composites, marks []rune
	for r := rune(0xA0); r <= 0x10FFFF; r++ {
		if r >= 0xD800 && r <= 0xDFFF {
			continue
		}
		s := string(r)
		if p := norm.NFC.PropertiesString(s); p.CCC() != 0 {
			marks = append(marks, r)
		} else if d := []rune(norm.NFD.String(s)); len(d) > 1 && norm.NFC.String(s) == s &&
			norm.NFC.PropertiesString(string(d[len(d)-1])).CCC() != 0 {
			composites = append(composites, r)
		}
	}
	pairs := 0
	for _, c := range composites {
		for _, m := range marks {
			s := string(c) + string(m)
			if got, want := nfcMayChange(s), norm.NFC.String(s) != s; got != want {
				t.Fatalf("nfcMayChange(%U %U) = %v, x/text changes it = %v", c, m, got, want)
			}
			pairs++
		}
	}
	if len(composites) < 850 || len(marks) < 850 {
		t.Fatalf("only %d composites and %d marks enumerated", len(composites), len(marks))
	}
	t.Logf("%d composites x %d marks = %d pairs, all exact", len(composites), len(marks), pairs)
}

// Exact on mixed text from the scripts above: starters, composites, marks of
// several classes (equal classes block, lower ones reorder), Hangul jamo and
// syllables, and runs past the stream-safe limit with starters between them.
func TestNFCExactOnScriptText(t *testing.T) {
	var alphabet []string
	for _, para := range nfcParagraphs(t) {
		for _, r := range norm.NFD.String(para) + para {
			alphabet = append(alphabet, string(r))
		}
	}
	alphabet = append(alphabet, "a", "A", "o", "\U000000C1", "\U000001FB", "\U00001EA0", "\U00001FC7",
		"\U00000622", "\U00000623", "\U00000624", "\U00000625", "\U00000626", "\U00000627", "\U00000648", "\U0000064A",
		"\U00000300", "\U00000301", "\U00000302", "\U0000030A", "\U00000316", "\U00000323", "\U00000327", "\U00000328",
		"\U00000315", "\U0000035C", "\U00000653", "\U00000654", "\U00000655", "\U0000064E", "\U00000650", "\U00000651",
		"\U00001100", "\U00001161", "\U000011A8", "\U0000AC00", "\U0000AC01", "\U00003099", "\U0000FF9E", "\U0000304B")
	rnd := rand.New(rand.NewSource(4))
	for i := 0; i < 300000; i++ {
		var b strings.Builder
		for n := 1 + rnd.Intn(10); n > 0; n-- {
			b.WriteString(alphabet[rnd.Intn(len(alphabet))])
		}
		if rnd.Intn(30) == 0 { // a run around the limit, a starter, another run
			m := stressMarks[rnd.Intn(len(stressMarks))]
			b.WriteString(strings.Repeat(m, 10+rnd.Intn(25)) + "a" + strings.Repeat(m, 10+rnd.Intn(25)))
		}
		s := b.String()
		if got, want := nfcMayChange(s), norm.NFC.String(s) != s; got != want {
			t.Fatalf("nfcMayChange(%+q) = %v, x/text changes it = %v", s, got, want)
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
