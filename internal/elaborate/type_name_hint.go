package elaborate

import (
	"strings"

	"github.com/sunholo-data/ailang/internal/importhint"
)

// typeSynonyms maps a type name from another language to the AILANG built-in
// a newcomer means by it. Only unambiguous ones: `number` is the
// floating-point number of JavaScript and JSON.
var typeSynonyms = map[string]string{
	"str": "string", "text": "string",
	"integer": "int", "long": "int", "int64": "int",
	"number": "float", "double": "float", "real": "float", "float64": "float",
	"boolean": "bool",
}

// closestTypeName picks the type a misspelt or foreign type name most likely
// meant, or "" when nothing is close — a wrong hint is worse than none. In
// order: a synonym from another language (str → string, integer → int);
// the same name up to case (point → Point, option → Option); then, for names
// of four letters or more, a candidate one is a prefix of (integer ↔ int,
// both at least three letters), else one within edit distance 1 (2 for names
// of six letters or more: on a short name, two edits reach unrelated words —
// point → int). The declaration's own name and the unit type are never
// suggested.
func closestTypeName(name, declName string, cands []string) string {
	keep := make([]string, 0, len(cands))
	for _, c := range cands {
		if c != declName && c != "()" && c != name {
			keep = append(keep, c)
		}
	}
	if s, ok := typeSynonyms[strings.ToLower(name)]; ok && s != declName {
		return s
	}
	for _, c := range keep {
		if strings.EqualFold(c, name) {
			return c
		}
	}
	if len(name) < 4 {
		return ""
	}
	lower := strings.ToLower(name)
	best, bestDiff := "", 0
	for _, c := range keep {
		lc := strings.ToLower(c)
		short, long := lc, lower
		if len(short) > len(long) {
			short, long = long, short
		}
		if len(short) >= 3 && strings.HasPrefix(long, short) {
			if d := len(long) - len(short); best == "" || d < bestDiff {
				best, bestDiff = c, d
			}
		}
	}
	if best != "" {
		return best
	}
	limit := 1
	if len(name) >= 6 {
		limit = 2
	}
	bestDist := limit + 1
	for _, c := range keep {
		if d := importhint.EditDistance(lower, strings.ToLower(c)); d < bestDist {
			best, bestDist = c, d
		}
	}
	return best
}
