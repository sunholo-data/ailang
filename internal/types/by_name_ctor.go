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
// candidates are the loaded types declaring the name: once the equalities
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

// checkByNameCtorPattern types a constructor pattern matched by name: at
// once when one loaded type declares it, else as a deferred choice between
// the types that do. ok is false when no loaded type declares the name: the
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
	// One loaded type declares the name: nothing is undecided, so it is
	// typed as that type's constructor at once — its equalities at fresh
	// type arguments, which a let generalises like any other (a helper over
	// Some is usable at Option[int] and Option[string]). A wrong argument
	// count is the match check's TC_MATCH_006 against that type. Only a
	// choice between two or more types is deferred, and only that keeps a
	// let-bound helper monomorphic until a use decides it.
	if len(all) == 1 {
		if scheme := tc.globalTypes[core.ADTFactoryKey(all[0].Module, all[0].TypeName, p.Name)]; scheme != nil {
			p.TypeName, p.TypeModule = all[0].TypeName, all[0].Module
			bindings, typed, err := tc.checkCtorPatternAt(p, scheme, scrutType, ctx)
			return bindings, typed, true, err
		}
	}
	choice := &ctorChoice{pattern: p, scrut: scrutType, cands: fit, all: all, byName: true,
		path: []string{fmt.Sprintf("constructor pattern %s", p.Name)}}
	bindings, typed, err := tc.deferCtorChoice(choice, ctx)
	return bindings, typed, true, err
}

// checkCtorPatternAt types p as the constructor whose factory scheme is
// scheme, instantiated at fresh type arguments: the value matched has the
// factory's result type and each argument pattern the matching field type.
func (tc *CoreTypeChecker) checkCtorPatternAt(p *core.ConstructorPattern, scheme *Scheme, scrutType Type, ctx *InferenceContext) (map[string]Type, typedast.TypedPattern, error) {
	path := []string{fmt.Sprintf("constructor pattern %s", p.Name)}
	var fields []Type
	switch t := scheme.Instantiate(ctx.freshType).(type) {
	case *TFunc2:
		fields = t.Params
		ctx.addConstraint(TypeEq{Left: scrutType, Right: t.Return, Path: path})
	default:
		ctx.addConstraint(TypeEq{Left: scrutType, Right: t, Path: path})
	}
	bindings := make(map[string]Type)
	typedArgs := make([]typedast.TypedPattern, len(p.Args))
	for i, argPat := range p.Args {
		argType := ctx.freshTypeVar()
		if i < len(fields) {
			argType = fields[i]
		}
		argBindings, typedArg, err := tc.checkPattern(argPat, argType, ctx)
		if err != nil {
			return nil, nil, err
		}
		for name, typ := range argBindings {
			if existing, ok := bindings[name]; ok {
				ctx.addConstraint(TypeEq{Left: existing, Right: typ, Path: []string{fmt.Sprintf("pattern variable %s", name)}})
			} else {
				bindings[name] = typ
			}
		}
		typedArgs[i] = typedArg
	}
	return bindings, typedast.TypedConstructorPattern{Name: p.Name, Args: typedArgs}, nil
}
