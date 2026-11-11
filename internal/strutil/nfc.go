package strutil

import (
	"sort"
	"unicode/utf8"
)

// The two NFC questions the quoting rule asks (quote_rule.go), answered with
// the standard library and the tables in nfc_tables.go (generated from
// golang.org/x/text/unicode/norm, the normaliser the loader uses):
//
//   - nfcStableStarter(r): can NFC neither rewrite r on its own nor combine it
//     with the rune before it? Exact (TestNFCAgreesWithXText checks every rune).
//   - nfcMayChange(s): might NFC change s? CONSERVATIVE: false means NFC
//     leaves s unchanged; true means it might. A wrong "true" only costs an
//     unnecessary \u{…} escape, which still round-trips; a wrong "false"
//     would be a bug, so the test asserts that direction only.
//
// nfcMayChange is a single linear pass (no normalisation, no sorting): the
// UAX #15 quick check (no rune NFC rewrites on its own, canonical order holds,
// no rune that combines backward meets something it combines with) plus the
// Stream-Safe Text Format limit x/text applies (a run of more than 30
// non-starters gets a U+034F inserted).
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

// nfcCompose reports whether a followed by b has a primary composite.
func nfcCompose(a, b rune) bool {
	if a >= hangulLBase && a < hangulLBase+hangulLCount && b >= hangulVBase && b < hangulVBase+hangulVCount {
		return true
	}
	if nfcIsHangulSyllable(a) && (a-hangulSBase)%hangulTCount == 0 && b > hangulTBase && b < hangulTBase+hangulTCount {
		return true
	}
	i := sort.Search(len(nfcCompTable), func(i int) bool {
		p := nfcCompTable[i]
		return p[0] > int32(a) || p[0] == int32(a) && p[1] >= int32(b)
	})
	return i < len(nfcCompTable) && nfcCompTable[i][0] == int32(a) && nfcCompTable[i][1] == int32(b)
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
func nfcMayChange(s string) bool {
	prev := rune(-1)      // the previous rune; -1 = none, or a byte that is not UTF-8
	starter := rune(-1)   // the last rune of class 0 (-1: none, or a stray byte)
	lastTrail := int32(0) // class of the last rune of prev's decomposition
	nonStarters := 0      // x/text's stream-safe count
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		i += size
		if r < 0xA0 || (r == utf8.RuneError && size == 1) {
			// Class 0, no decomposition, combines with nothing before it.
			prev, starter, lastTrail, nonStarters = r, r, 0, 0
			if size == 1 && r >= utf8.RuneSelf {
				prev, starter = -1, -1 // a stray byte composes with nothing
			}
			continue
		}
		ccc := nfcCCC(r)
		lead, trail := ccc, ccc
		if d := nfcDecomp(r); d != nil {
			if !nfcHas(nfcComposites, r) {
				return true // a singleton or composition exclusion: NFC rewrites it
			}
			lead, trail = nfcCCC(rune(d[0])), nfcCCC(rune(d[len(d)-1]))
		}
		if lead != 0 && lastTrail > lead {
			return true // not in canonical order: NFC reorders
		}
		// r composes with the last starter unless something between blocks
		// it: a starter, or (the marks are in canonical order, so the last is
		// the highest) a mark of class >= r's.
		if nfcBackwardRune(r) && starter >= 0 && (prev == starter || ccc != 0 && lastTrail < ccc) && nfcCompose(starter, r) {
			return true
		}
		// The stream-safe limit: x/text inserts U+034F after 30 non-starters.
		if nLead, nTrail := nfcNonStarters(r); nLead == 0 {
			nonStarters = nTrail
		} else if nonStarters += nLead; nonStarters > nfcMaxNonStarters {
			return true
		}
		prev, lastTrail = r, trail
		if ccc == 0 {
			starter = r
		}
	}
	return false
}
