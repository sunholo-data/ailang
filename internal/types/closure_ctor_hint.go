package types

import (
	"fmt"
	"sort"
	"strings"
)

// Since #2371 Some/None/Ok/Err are the prelude's in every module outside
// std/. A library that matched a constructor of its OWN type of that name by
// name (a `type Status = Ok | Err(string) | Pending` it never imported) now
// gets the prelude's constructor, so the pattern is an arity error or a
// foreign-constructor error. The fix is one import; these errors name it
// when a type in the module's import closure declares the constructor. The
// import names the type AND its constructors: importing the type alone does
// not bring them into scope, and every one of them the prelude shadows
// (Ok and Err, say) needs the explicit import to win.

// ImportCtorHint is the one-step fix for meaning c's constructor named ctor:
// import c's type with all its constructors (from u, the import closure).
func ImportCtorHint(u MatchUniverse, ctor string, c CtorCandidate) string {
	names := []string{c.TypeName}
	for _, s := range u[NominalKey(c.Module, c.TypeName)] {
		names = append(names, s.Name)
	}
	return fmt.Sprintf("if you mean %s's %s.%s, import it: import %s (%s)", c.Module, c.TypeName, ctor, c.Module, strings.Join(names, ", "))
}

// ImportCtorHints joins the hints for cands with " — or ".
func ImportCtorHints(u MatchUniverse, ctor string, cands []CtorCandidate) string {
	hints := make([]string, len(cands))
	for i, c := range cands {
		hints[i] = ImportCtorHint(u, ctor, c)
	}
	return strings.Join(hints, " — or ")
}

// ClosureCtorCandidates returns, for every constructor name, the types in u
// (a module's import closure: SetMatchUniverse) that declare it, sorted by
// module then type. Used only for diagnostics.
func ClosureCtorCandidates(u MatchUniverse) map[string][]CtorCandidate {
	out := make(map[string][]CtorCandidate)
	for key, sigs := range u {
		mod, name := SplitNominalKey(key)
		for _, s := range sigs {
			out[s.Name] = append(out[s.Name], CtorCandidate{Module: mod, TypeName: name, Arity: s.Arity})
		}
	}
	for _, cs := range out {
		sort.Slice(cs, func(i, j int) bool {
			if cs[i].Module != cs[j].Module {
				return cs[i].Module < cs[j].Module
			}
			return cs[i].TypeName < cs[j].TypeName
		})
	}
	return out
}

// foreignCtorHint names the import that makes ctor the scrutinee's own
// constructor: the closure types named scrutADT that declare ctor, other
// than ctorADT's (the one in scope). "" when there is none.
func (tc *CoreTypeChecker) foreignCtorHint(ctor, ctorADT, scrutADT string) string {
	var match []CtorCandidate
	for _, c := range ClosureCtorCandidates(tc.matchUniverse)[ctor] {
		if c.TypeName == scrutADT && c.TypeName != ctorADT && c.Module != "" {
			match = append(match, c)
		}
	}
	return ImportCtorHints(tc.matchUniverse, ctor, match)
}
