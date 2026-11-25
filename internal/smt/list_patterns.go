package smt

import (
	"fmt"
	"strings"

	"github.com/sunholo-data/ailang/internal/core"
)

// List patterns (M-VERIFY-SOUNDNESS-COVERAGE): `[]`, `[a, b]`, `[x, ...rest]`
// and `x :: rest` (elaborated to the same ListPattern) have no SMT match form,
// since a list is a Seq, not a datatype. A match using one goes through the
// nested-pattern encoder (encodeNestedMatch): each arm is a test on the length
// and on the elements, and binds its variables to seq.nth and seq.extract
// terms. The arms form a first-match ite chain whose last arm is taken
// untested, so the chain is emitted only when listCoverage proves the
// unguarded arms cover every length; otherwise the function is skipped.
//
// The bindings are well defined where they are used: an arm's body is reached
// only when its length test holds (the last arm by coverage), so seq.nth is
// never read out of range there.

// needsNestedMatch reports whether a match with an arm of pattern p must go
// through encodeNestedMatch: SMT match has neither nested constructor
// patterns nor list patterns.
func needsNestedMatch(p core.CorePattern) bool {
	return hasNestedConstructorPattern(p) || hasListPattern(p)
}

// hasListPattern reports whether p is, or contains, a list pattern.
func hasListPattern(p core.CorePattern) bool {
	switch x := p.(type) {
	case *core.ListPattern:
		return true
	case *core.ConstructorPattern:
		for _, arg := range x.Args {
			if hasListPattern(arg) {
				return true
			}
		}
	}
	return false
}

// seqElementSort is T for the sort (Seq T), and "" for any other sort.
func seqElementSort(sort string) string {
	if !strings.HasPrefix(sort, "(Seq ") {
		return ""
	}
	parts := smtListParts(sort)
	if len(parts) != 2 || parts[0] != "Seq" {
		return ""
	}
	return parts[1]
}

// listPatternTest is nestedPatternTest's case for a list pattern on value,
// whose sort is (Seq elem).
func listPatternTest(p *core.ListPattern, value, sort string) (string, []string, error) {
	elem := seqElementSort(sort)
	if elem == "" {
		return "", nil, fmt.Errorf("%w: list pattern on a value of sort %s", ErrUnresolvableTypes, sort)
	}
	n := len(p.Elements)
	length := fmt.Sprintf("(seq.len %s)", value)
	tests := []string{fmt.Sprintf("(= %s %d)", length, n)}
	if p.Tail != nil {
		tests[0] = fmt.Sprintf("(>= %s %d)", length, n)
	}
	var bindings []string
	add := func(pat core.CorePattern, v, s string) error {
		test, b, err := nestedPatternTest(pat, v, s)
		if err != nil {
			return err
		}
		if test != "true" {
			tests = append(tests, test)
		}
		bindings = append(bindings, b...)
		return nil
	}
	for i, el := range p.Elements {
		if err := add(el, fmt.Sprintf("(seq.nth %s %d)", value, i), elem); err != nil {
			return "", nil, err
		}
	}
	if p.Tail != nil {
		rest := fmt.Sprintf("(seq.extract %s %d (- %s %d))", value, n, length, n)
		if err := add(*p.Tail, rest, sort); err != nil {
			return "", nil, err
		}
	}
	if len(tests) == 1 {
		return tests[0], bindings, nil
	}
	return "(and " + strings.Join(tests, " ") + ")", bindings, nil
}

// listCoverage is patternCoverage's case for a first column of sort
// (Seq elem). With m the most elements any list pattern in the column names,
// the lengths 0…m are checked one by one, each as that many element columns,
// and every length above m as m element columns. A pattern with a tail covers
// a length when it names no more elements than that, and only if the tail
// pattern itself is irrefutable; a refutable tail never counts as covering
// (conservative: coverage may fail, never succeed wrongly).
func listCoverage(rows [][]core.CorePattern, sorts []string, elem string, depth int) bool {
	maxN := 0
	for _, r := range rows {
		switch p := r[0].(type) {
		case *core.ListPattern:
			maxN = max(maxN, len(p.Elements))
		default:
			if !irrefutablePattern(p) {
				return false
			}
		}
	}
	for k := 0; k <= maxN+1; k++ {
		longer := k == maxN+1
		cols := k
		if longer {
			cols = maxN
		}
		var next [][]core.CorePattern
		for _, r := range rows {
			args, ok := listRowAt(r[0], k, cols, longer)
			if !ok {
				continue
			}
			next = append(next, append(args, r[1:]...))
		}
		nextSorts := make([]string, 0, cols+len(sorts)-1)
		for range cols {
			nextSorts = append(nextSorts, elem)
		}
		nextSorts = append(nextSorts, sorts[1:]...)
		if !patternCoverage(next, nextSorts, depth+1) {
			return false
		}
	}
	return true
}

// listRowAt specialises one first-column pattern to lists of length k (every
// length above the longest pattern when longer), as cols element patterns.
// ok is false when the pattern does not cover that length.
func listRowAt(p core.CorePattern, k, cols int, longer bool) ([]core.CorePattern, bool) {
	wild := func(n int) []core.CorePattern {
		out := make([]core.CorePattern, n)
		for i := range out {
			out[i] = &core.WildcardPattern{}
		}
		return out
	}
	if irrefutablePattern(p) {
		return wild(cols), true
	}
	lp, ok := p.(*core.ListPattern)
	if !ok {
		return nil, false
	}
	n := len(lp.Elements)
	if lp.Tail == nil {
		if longer || n != k {
			return nil, false
		}
		return append([]core.CorePattern(nil), lp.Elements...), true
	}
	if n > k || !irrefutablePattern(*lp.Tail) {
		return nil, false
	}
	return append(append([]core.CorePattern(nil), lp.Elements...), wild(cols-n)...), true
}
