package strutil

import (
	"sort"
	"unicode/utf8"
)

// The two NFC questions the quoting rule asks (quote_rule.go), answered with
// the standard library and the tables in nfc_tables.go (generated from
// golang.org/x/text/unicode/norm, the normaliser the loader uses):
//
//   - nfcRawSafe(r): may r be written raw in a string NFC might change? A
//     stable starter (NFC can neither rewrite it alone nor combine it with
//     the rune before it; exact on every rune) that x/text does not count as
//     a leading non-starter.
//   - nfcMayChange(s): might NFC change s? Safe in one direction by design:
//     false means NFC leaves s unchanged. A wrong "true" would only cost an
//     unnecessary \u{…} escape, which still round-trips; the tests also pin
//     it exact on the classes where over-escaping showed up (composite +
//     mark, script text), so readable text stays readable.
//
// nfcMayChange is a single linear pass (no normalisation, no sorting): the
// UAX #15 quick check (no rune NFC rewrites on its own, canonical order holds,
// no rune that combines backward meets an unblocked base it composes with)
// plus the Stream-Safe Text Format limit x/text applies (a run of more than
// 30 non-starters gets a U+034F inserted).
//
// Standard library only on purpose: the Go emitter copies this file,
// quote_rule.go and the tables into every compiled program that uses show
// (internal/gen/golang, emittedQuoteRuntime).

const (
	hangulSBase  = 0xAC00
	hangulLBase  = 0x1100
	hangulVBase  = 0x1161
	hangulTBase  = 0x11A7
	hangulLCount = 19
	hangulVCount = 21
	hangulTCount = 28
	hangulNCount = hangulVCount * hangulTCount
	hangulSCount = hangulLCount * hangulNCount

	// nfcMaxNonStarters is x/text's maxNonStarters (Stream-Safe Text Format).
	nfcMaxNonStarters = 30
)

// nfcComposites and nfcBackward are the sorted composite runes and the sorted
// runes that can combine with a preceding rune, derived from nfcCompTable.
var nfcComposites, nfcBackward = nfcDeriveSets()

func nfcDeriveSets() ([]int32, []int32) {
	comps := make([]int32, 0, len(nfcCompTable))
	seen := map[int32]bool{}
	back := []int32{}
	for _, p := range nfcCompTable {
		comps = append(comps, p[2])
		if !seen[p[1]] {
			seen[p[1]] = true
			back = append(back, p[1])
		}
	}
	sort.Slice(comps, func(i, j int) bool { return comps[i] < comps[j] })
	sort.Slice(back, func(i, j int) bool { return back[i] < back[j] })
	return comps, back
}

func nfcHas(set []int32, r rune) bool {
	i := sort.Search(len(set), func(i int) bool { return set[i] >= int32(r) })
	return i < len(set) && set[i] == int32(r)
}

// nfcCCC is r's canonical combining class.
func nfcCCC(r rune) int32 {
	if r < 0x300 {
		return 0
	}
	i := sort.Search(len(nfcCCCTable), func(i int) bool { return nfcCCCTable[i][1] >= int32(r) })
	if i < len(nfcCCCTable) && nfcCCCTable[i][0] <= int32(r) {
		return nfcCCCTable[i][2]
	}
	return 0
}

func nfcIsHangulSyllable(r rune) bool { return r >= hangulSBase && r < hangulSBase+hangulSCount }

// nfcBackwardRune: r can combine with the rune before it (a Hangul vowel or
// trailing jamo, or the second rune of a primary composite).
func nfcBackwardRune(r rune) bool {
	return (r >= hangulVBase && r < hangulVBase+hangulVCount) ||
		(r > hangulTBase && r < hangulTBase+hangulTCount) || nfcHas(nfcBackward, r)
}

// nfcDecomp returns r's full canonical decomposition from the table, or nil
// (no decomposition, or a Hangul syllable: all starters).
func nfcDecomp(r rune) []int32 {
	if r < 0xC0 || nfcIsHangulSyllable(r) {
		return nil
	}
	i := sort.Search(len(nfcDecompKeys), func(i int) bool { return nfcDecompKeys[i] >= int32(r) })
	if i < len(nfcDecompKeys) && nfcDecompKeys[i] == int32(r) {
		return nfcDecompData[nfcDecompOff[i]:nfcDecompOff[i+1]]
	}
	return nil
}

// nfcCompose returns the primary composite of a followed by b, if any.
func nfcCompose(a, b rune) (rune, bool) {
	if a >= hangulLBase && a < hangulLBase+hangulLCount && b >= hangulVBase && b < hangulVBase+hangulVCount {
		return hangulSBase + ((a-hangulLBase)*hangulVCount+b-hangulVBase)*hangulTCount, true
	}
	if nfcIsHangulSyllable(a) && (a-hangulSBase)%hangulTCount == 0 && b > hangulTBase && b < hangulTBase+hangulTCount {
		return a + b - hangulTBase, true
	}
	i := sort.Search(len(nfcCompTable), func(i int) bool {
		p := nfcCompTable[i]
		return p[0] > int32(a) || p[0] == int32(a) && p[1] >= int32(b)
	})
	if i < len(nfcCompTable) && nfcCompTable[i][0] == int32(a) && nfcCompTable[i][1] == int32(b) {
		return rune(nfcCompTable[i][2]), true
	}
	return 0, false
}

// nfcStableStarter reports whether r is a starter that NFC can neither
// rewrite on its own nor combine with the rune before it.
func nfcStableStarter(r rune) bool {
	if r < 0x300 {
		return true
	}
	if nfcCCC(r) != 0 || nfcBackwardRune(r) {
		return false
	}
	// A rune with a decomposition survives NFC only if it is a primary
	// composite (singletons and composition exclusions do not).
	return nfcDecomp(r) == nil || nfcHas(nfcComposites, r)
}

// nfcRawSafe reports whether r can be written raw in a string NFC might
// change: a stable starter that x/text does not count as a leading
// non-starter (U+FF9E and the Hangul compatibility vowels are stable
// starters with a leading count, so a run of them overflows the stream-safe
// limit like a run of marks).
func nfcRawSafe(r rune) bool {
	lead, _ := nfcNonStarters(r)
	return lead == 0 && nfcStableStarter(r)
}

// nfcNonStarters is r's leading and trailing non-starter count as x/text's
// Stream-Safe Text Format counts them: runes of class != 0 or that combine
// backward, at the start and end of r's decomposition (x/text uses the
// compatibility decomposition; nfcNonStarterExtra holds where that differs).
func nfcNonStarters(r rune) (lead, trail int) {
	i := sort.Search(len(nfcNonStarterExtra), func(i int) bool { return nfcNonStarterExtra[i][0] >= int32(r) })
	if i < len(nfcNonStarterExtra) && nfcNonStarterExtra[i][0] == int32(r) {
		return int(nfcNonStarterExtra[i][1]), int(nfcNonStarterExtra[i][2])
	}
	if nfcIsHangulSyllable(r) {
		if (r-hangulSBase)%hangulTCount == 0 {
			return 0, 1 // LV: the vowel
		}
		return 0, 2 // LVT: vowel and trailing jamo
	}
	ns := func(x int32) bool { return nfcCCC(rune(x)) != 0 || nfcBackwardRune(rune(x)) }
	d := nfcDecomp(r)
	if d == nil {
		d = []int32{int32(r)}
	}
	for lead < len(d) && ns(d[lead]) {
		lead++
	}
	for trail < len(d) && ns(d[len(d)-1-trail]) {
		trail++
	}
	return lead, trail
}

// nfcMayChange reports whether NFC might change s (see the file comment:
// false is a guarantee, true is conservative). One pass, linear in len(s).
//
// It follows each starter and the marks after it. NFC decomposes the
// starter, sorts its trailing marks together with the marks that follow, and
// recomposes. A following mark of class c therefore meets, as its base, the
// starter's decomposition composed only up to its marks of class <= c
// (nfcPartialBase): U+0623 is U+0627 U+0654, so a fatha (class 30) after it
// meets U+0627, does not combine with it, and the text is unchanged.
func nfcMayChange(s string) bool {
	starter := rune(-1)  // the last rune of class 0 (-1: none, or a stray byte)
	starterTrail := 0    // class of the last rune of the starter's decomposition
	lastMark := int32(0) // class of the last mark since the starter (0: none)
	nonStarters := 0     // x/text's stream-safe count
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		i += size
		if r < 0xA0 || (r == utf8.RuneError && size == 1) {
			// Class 0, no decomposition, combines with nothing before it.
			starter, starterTrail, lastMark, nonStarters = r, 0, 0, 0
			if size == 1 && r >= utf8.RuneSelf {
				starter = -1 // a stray byte composes with nothing
			}
			continue
		}
		ccc := nfcCCC(r)
		d := nfcDecomp(r)
		if d != nil && (!nfcHas(nfcComposites, r) || ccc != 0 || nfcCCC(rune(d[0])) != 0) {
			return true // NFC rewrites it (a singleton or exclusion), or a shape this check does not model
		}
		if ccc != 0 {
			if lastMark > ccc {
				return true // not in canonical order: NFC reorders
			}
			// Blocked by a mark of the same class between the base and r.
			if starter >= 0 && lastMark < ccc && nfcBackwardRune(r) {
				base, ok := nfcPartialBase(starter, starterTrail, ccc)
				if !ok {
					return true
				}
				if _, ok := nfcCompose(base, r); ok {
					return true
				}
			}
			lastMark = ccc
		} else {
			// A starter combines with the one before it only when adjacent.
			if starter >= 0 && lastMark == 0 && nfcBackwardRune(r) {
				if _, ok := nfcCompose(starter, r); ok {
					return true
				}
			}
			starter, starterTrail, lastMark = r, 0, 0
			if d != nil {
				starterTrail = int(nfcCCC(rune(d[len(d)-1])))
			}
		}
		// The stream-safe limit: x/text inserts U+034F after 30 non-starters.
		if nLead, nTrail := nfcNonStarters(r); nLead == 0 {
			nonStarters = nTrail
		} else if nonStarters += nLead; nonStarters > nfcMaxNonStarters {
			return true
		}
	}
	return false
}

// nfcPartialBase is what a mark of class c meets as its base after starter
// s once NFC has sorted it in among s's own trailing marks: s's
// decomposition recomposed up to its marks of class <= c. ok is false when
// that prefix does not recompose to one rune (the caller then answers
// conservatively).
func nfcPartialBase(s rune, trail int, c int32) (rune, bool) {
	if int32(trail) <= c {
		return s, true
	}
	d := nfcDecomp(s)
	base := rune(d[0])
	for _, x := range d[1:] {
		if k := nfcCCC(rune(x)); k == 0 || k > c {
			break
		}
		next, ok := nfcCompose(base, rune(x))
		if !ok {
			return 0, false
		}
		base = next
	}
	return base, true
}
