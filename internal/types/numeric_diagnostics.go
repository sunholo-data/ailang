package types

import (
	"fmt"
	"sort"
	"strings"

	"github.com/sunholo-data/ailang/internal/core"
)

// #2378 diagnostics: when an int/float conflict comes from one of the numeric
// rules, the error says which rule and where, instead of a generic hint.
//
//   - A module declaration's literal-carrying variable that no parameter
//     carries is an int (declResultOnlyVars). A declaration that uses such a
//     binding and fails on Fractional[int] is told which binding is an int and
//     where its literal is (noteForcedLiteral, explainNumericFailure). The note
//     is only ever about a binding that has such a literal.
//   - A let-bound binding that is one run-time value (an expansive value, or a
//     literal-carrying constant) has one type for all its uses. When two of its
//     calls conflict — each call type-checks on its own, all of them together
//     do not (a dry run, as usesConflict does for constructors) — the error
//     says that the binding is a single value used at two numeric types.

// forcedLiteral is a module declaration whose literal-carrying variable was
// defaulted to int, and where that literal (or the literal-carrying use) is.
type forcedLiteral struct {
	name, where string
}

// singleValueNote is a let-bound binding kept at one type although its
// numeric variables carry literals or numeric constraints.
type singleValueNote struct {
	name, where string
}

// noteForcedLiteral records that binder's declaration had a literal-carrying
// variable defaulted; where is the literal's constraint path.
func (tc *CoreTypeChecker) noteForcedLiteral(binder core.CoreExpr, where string) {
	if tc.forcedPending == nil {
		tc.forcedPending = make(map[string]string)
	}
	for _, name := range binderNames(binder) {
		if _, ok := tc.forcedPending[name]; !ok {
			tc.forcedPending[name] = strings.TrimPrefix(strings.TrimPrefix(where, "literal at "), "use of ")
		}
	}
}

// registerForced ties a declaration's scheme to its forced literal, so a use
// of the binding can be recognised (recordForcedUse).
func (tc *CoreTypeChecker) registerForced(name string, binding interface{}) {
	where, ok := tc.forcedPending[name]
	if !ok {
		return
	}
	delete(tc.forcedPending, name)
	if s, ok := binding.(*Scheme); ok && s != nil {
		if tc.forcedSchemes == nil {
			tc.forcedSchemes = make(map[*Scheme]forcedLiteral)
		}
		tc.forcedSchemes[s] = forcedLiteral{name: name, where: where}
	}
}

// recordForcedUse notes that the declaration being checked uses a binding
// whose literal was defaulted to int.
func (tc *CoreTypeChecker) recordForcedUse(s *Scheme) {
	f, ok := tc.forcedSchemes[s]
	if !ok {
		return
	}
	for _, g := range tc.declForcedUses {
		if g.name == f.name {
			return
		}
	}
	tc.declForcedUses = append(tc.declForcedUses, f)
}

// noteSingleValue records that binder keeps the numeric variables vars at one
// type for all its uses.
func (tc *CoreTypeChecker) noteSingleValue(ctx *InferenceContext, binder core.CoreExpr, vars map[string]bool) {
	if ctx == nil || binder == nil || len(vars) == 0 {
		return
	}
	for _, name := range binderNames(binder) {
		ctx.singleValues = append(ctx.singleValues, singleValueNote{name: name, where: binder.Span().String()})
	}
}

// singleValue records an expansive let's numeric variables as one value
// (noteSingleValue) and returns its type, the let's binding.
func (tc *CoreTypeChecker) singleValue(ctx *InferenceContext, let *core.Let, sub Substitution, t Type) Type {
	tc.noteSingleValue(ctx, let, numericVarsOf(ctx, sub, t))
	return t
}

// numericVarsOf returns the variables of t that a Num-class constraint of ctx
// ranges over (an expansive let's numeric variables).
func numericVarsOf(ctx *InferenceContext, sub Substitution, t Type) map[string]bool {
	inType := freeTypeVars(t)
	out := map[string]bool{}
	for _, c := range ctx.qualifiedConstraints {
		if c.Class != "Num" && c.Class != "Fractional" {
			continue
		}
		for v := range freeTypeVars(ApplySubstitution(sub, c.Type)) {
			if inType[v] {
				out[v] = true
			}
		}
	}
	return out
}

func binderNames(b core.CoreExpr) []string {
	switch x := b.(type) {
	case *core.Let:
		return []string{x.Name}
	case *core.LetRec:
		out := make([]string, len(x.Bindings))
		for i, rb := range x.Bindings {
			out[i] = rb.Name
		}
		return out
	}
	return nil
}

// isNumericConflict reports whether err is an int/float conflict: a missing
// Fractional[int], or an int and a float that do not unify.
func isNumericConflict(err error) bool {
	msg := err.Error()
	if strings.Contains(msg, "No instance for Fractional[int]") {
		return true
	}
	return strings.Contains(msg, "unif") && (strings.Contains(msg, "int") && strings.Contains(msg, "float"))
}

// explainNumericFailure returns err, or a clearer error when a numeric rule
// of #2378 is behind it (see the file comment).
func (tc *CoreTypeChecker) explainNumericFailure(ctx *InferenceContext, err error) error {
	if err == nil || ctx == nil || !isNumericConflict(err) {
		return err
	}
	for _, n := range ctx.singleValues {
		if calls := ctx.numericUsesConflict(tc, n.name); len(calls) > 0 {
			return fmt.Errorf("%s: `%s` is a single value, not a function: what it is bound to is evaluated once, so all its uses have one numeric type, and its uses at %s need different ones (int and float). "+
				"Bind it to a function (a lambda, or a block ending in one) so that each use gets its own type, or give it a type annotation: %w",
				n.where, n.name, strings.Join(calls, " and "), err)
		}
	}
	if len(tc.declForcedUses) > 0 {
		var notes []string
		for _, f := range tc.declForcedUses {
			notes = append(notes, fmt.Sprintf("`%s` is an int: its integer literal (at %s) occurs in no parameter of its type, so it is not generic (#2378); write that literal as a float (3.0), annotate `%s` as float, or convert with intToFloat", f.name, f.where, f.name))
		}
		return fmt.Errorf("%w %s.", err, strings.Join(notes, "; "))
	}
	return err
}

// numericUsesConflict returns the positions of name's calls when they
// conflict with each other numerically: there are at least two, and each one
// on its own (the others' equalities left out) solves with every ground
// Num/Fractional constraint met. nil otherwise.
func (ctx *InferenceContext) numericUsesConflict(tc *CoreTypeChecker, name string) []string {
	prefix := "function application of " + name + " at "
	var calls []int
	var where []string
	for i, c := range ctx.constraints {
		if eq, ok := c.(TypeEq); ok && len(eq.Path) > 0 && strings.HasPrefix(eq.Path[0], prefix) {
			calls = append(calls, i)
			where = append(where, strings.TrimPrefix(eq.Path[0], prefix))
		}
	}
	if len(calls) < 2 {
		return nil
	}
	skip := make(map[int]bool, len(ctx.derivedEqs))
	for _, i := range ctx.derivedEqs {
		skip[i] = true
	}
	for _, keep := range calls {
		var cs []TypeConstraint
		for i, c := range ctx.constraints {
			drop := skip[i]
			for _, call := range calls {
				if i == call && call != keep {
					drop = true
				}
			}
			if !drop {
				cs = append(cs, c)
			}
		}
		if !ctx.solvesNumericDry(tc, cs) {
			return nil
		}
	}
	sort.Strings(where)
	return where
}

// solvesNumericDry is solvesDry that also requires every Num and Fractional
// constraint that the solution makes ground to have an instance.
func (ctx *InferenceContext) solvesNumericDry(tc *CoreTypeChecker, cs []TypeConstraint) bool {
	saved := ctx.freshCounter
	defer func() { ctx.freshCounter = saved }()
	sub := make(Substitution)
	for _, c := range cs {
		var err error
		switch x := c.(type) {
		case TypeEq:
			sub, err = ctx.unifier.Unify(ApplySubstitution(sub, x.Left), ApplySubstitution(sub, x.Right), sub)
		case RowEq:
			sub, err = ctx.unifier.rowUnifier.UnifyRows(x.Left, x.Right, sub)
		}
		if err != nil {
			return false
		}
	}
	sub, err := ctx.resolveChoicesIn(sub, cs, map[*ctorChoice]bool{})
	if err != nil {
		return false
	}
	for _, c := range cs {
		cc, ok := c.(ClassConstraint)
		if !ok || (cc.Class != "Num" && cc.Class != "Fractional") {
			continue
		}
		t := ApplySubstitution(sub, cc.Type)
		if !isGround(t) {
			continue
		}
		if _, err := tc.instanceEnv.Lookup(cc.Class, t); err != nil {
			return false
		}
	}
	return true
}
