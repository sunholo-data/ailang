package strutil

import (
	"sort"
	"unicode/utf8"
)

// A standard-library-only implementation of the two NFC questions the quoting
// rule asks (quote.go): is this string already in NFC, and is this rune an
// NFC-stable starter (a rune that NFC can neither rewrite on its own nor
// combine with the rune before it). The data comes from nfc_tables.go,
// generated from golang.org/x/text/unicode/norm — the normaliser the loader
// uses — and TestNFCAgreesWithXText checks both answers against x/text.
//
// It has no dependency outside the standard library on purpose: the Go
// emitter copies this file, quote.go's rule and the tables into every
// compiled program (internal/gen/golang, emittedQuoteRuntime), so compiled
// programs quote exactly as the interpreters do.

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

// nfcCCC is r's canonical combining class (0 for a starter or a stray byte).
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

// nfcDecompose appends r's full canonical decomposition (or r) to dst.
func nfcDecompose(dst []rune, r rune) []rune {
	if r >= hangulSBase && r < hangulSBase+hangulSCount {
		s := r - hangulSBase
		dst = append(dst, hangulLBase+s/hangulNCount, hangulVBase+(s%hangulNCount)/hangulTCount)
		if t := s % hangulTCount; t != 0 {
			dst = append(dst, hangulTBase+t)
		}
		return dst
	}
	if r >= 0xC0 {
		i := sort.Search(len(nfcDecompKeys), func(i int) bool { return nfcDecompKeys[i] >= int32(r) })
		if i < len(nfcDecompKeys) && nfcDecompKeys[i] == int32(r) {
			for _, d := range nfcDecompData[nfcDecompOff[i]:nfcDecompOff[i+1]] {
				dst = append(dst, rune(d))
			}
			return dst
		}
	}
	return append(dst, r)
}

// nfcCompose returns the primary composite of a followed by b, if any.
func nfcCompose(a, b rune) (rune, bool) {
	if a >= hangulLBase && a < hangulLBase+hangulLCount && b >= hangulVBase && b < hangulVBase+hangulVCount {
		return hangulSBase + ((a-hangulLBase)*hangulVCount+(b-hangulVBase))*hangulTCount, true
	}
	if a >= hangulSBase && a < hangulSBase+hangulSCount && (a-hangulSBase)%hangulTCount == 0 &&
		b > hangulTBase && b < hangulTBase+hangulTCount {
		return a + (b - hangulTBase), true
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
	if nfcCCC(r) != 0 || nfcHas(nfcBackward, r) ||
		(r >= hangulVBase && r < hangulVBase+hangulVCount) || (r > hangulTBase && r < hangulTBase+hangulTCount) {
		return false
	}
	// A rune with a decomposition survives NFC only if it is a primary
	// composite (singletons and composition exclusions do not).
	if r >= hangulSBase && r < hangulSBase+hangulSCount {
		return true
	}
	if one := nfcDecompose(nil, r); len(one) == 1 && one[0] == r {
		return true
	}
	return nfcHas(nfcComposites, r)
}

// nfcIsNormal reports whether s is already in NFC (invalid bytes pass
// through untouched, as starters that compose with nothing).
func nfcIsNormal(s string) bool {
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		if (r == utf8.RuneError && size == 1) || nfcStableStarter(r) {
			i += size
			continue
		}
		return nfcString(s) == s
	}
	return true
}

// nfcString returns NFC(s): canonical decomposition, canonical ordering,
// canonical composition. A byte that is not valid UTF-8 is carried through as
// a pseudo-rune below zero.
func nfcString(s string) string {
	buf := make([]rune, 0, len(s))
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && size == 1 {
			buf = append(buf, -1-rune(s[i]))
		} else {
			buf = nfcDecompose(buf, r)
		}
		i += size
	}
	ccc := func(r rune) int32 {
		if r < 0 {
			return 0
		}
		return nfcCCC(r)
	}
	// Canonical ordering: stable sort of each run of non-starters by class.
	for i := 1; i < len(buf); i++ {
		for j := i; j > 0; j-- {
			cj, cp := ccc(buf[j]), ccc(buf[j-1])
			if cj == 0 || cp <= cj {
				break
			}
			buf[j], buf[j-1] = buf[j-1], buf[j]
		}
	}
	// Canonical composition.
	out := buf[:0]
	starter, last := -1, int32(0)
	for _, c := range buf {
		cc := ccc(c)
		if starter >= 0 && c >= 0 && out[starter] >= 0 && (last < cc || last == 0 && len(out)-1 == starter) {
			if comp, ok := nfcCompose(out[starter], c); ok {
				out[starter] = comp
				continue
			}
		}
		if cc == 0 {
			starter, last = len(out), 0
		} else {
			last = cc
		}
		out = append(out, c)
	}
	b := make([]byte, 0, len(s))
	for _, r := range out {
		if r < 0 {
			b = append(b, byte(-1-r))
		} else {
			b = utf8.AppendRune(b, r)
		}
	}
	return string(b)
}
