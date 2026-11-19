package types

import (
	"fmt"
	"sort"

	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/typedast"
)

// A constructor matched by name (#323) — neither it nor its type is declared
// or imported here, its module is only loaded — constrains nothing by its
// name alone. Before #2353 its arguments were typed with fresh variables tied
// to nothing, so `match nth([1, 2], 0) { Some(v) => v ++ ["a"], … }` passed
// `check` with v an int used as a list, and a constructor nested in one
// (`Some(Red)`, `MkBox(Red)`) could never have its own type fixed (#2303's
// TC_MATCH_007 rejected it although the value's type was annotated).
//
// It is now a deferred constructor choice (ambiguous_ctor.go) whose
// candidates are the types declaring the name in the module's own import
// closure (pipeline: resolveModuleImports reads only `reach`, never the
// siblings a program happens to load): once the equalities
// fix the value's type, the choice picks that type's constructor and adds its
// equalities — the value's type is the constructor's result, each argument
// pattern's type the matching field at the value's type arguments — so the
// variables it binds get their real types and a nested by-name pattern is
// decided from the outer one. A choice the declaration never decides is left
// to the match check, which reports TC_MATCH_007 (match_exhaustiveness.go).

// byNameCandidates returns every loaded type that declares a constructor
// named name, with that constructor's arity, sorted by module then type.
func (tc *CoreTypeChecker) byNameCandidates(name string) []CtorCandidate {
	var out []CtorCandidate
	for key, sigs := range tc.matchUniverse {
		for _, s := range sigs {
			if s.Name == name {
				mod, typ := SplitNominalKey(key)
				out = append(out, CtorCandidate{Module: mod, TypeName: typ, Arity: s.Arity})
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Module != out[j].Module {
			return out[i].Module < out[j].Module
		}
		return out[i].TypeName < out[j].TypeName
	})
	return out
}

// checkByNameCtorPattern types a constructor pattern matched by name as a
// deferred choice. ok is false when no loaded type declares the name: the
// pattern is then typed as before (its arguments against fresh variables).
// A pattern whose arity no candidate has is still a choice, with no fitting
// candidate: the value's type, once fixed, is stamped on it and the match
// check reports the arity (TC_MATCH_006) against that type.
func (tc *CoreTypeChecker) checkByNameCtorPattern(p *core.ConstructorPattern, scrutType Type, ctx *InferenceContext) (map[string]Type, typedast.TypedPattern, bool, error) {
	all := tc.byNameCandidates(p.Name)
	if len(all) == 0 {
		return nil, nil, false, nil
	}
	var fit []CtorCandidate
	for _, c := range all {
		if c.Arity == len(p.Args) {
			fit = append(fit, c)
		}
	}
	choice := &ctorChoice{pattern: p, scrut: scrutType, cands: fit, all: all, byName: true,
		path: []string{fmt.Sprintf("constructor pattern %s", p.Name)}}
	bindings, typed, err := tc.deferCtorChoice(choice, ctx)
	return bindings, typed, true, err
}
