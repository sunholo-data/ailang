package types

import (
	"fmt"
	"strings"

	"github.com/sunholo-data/ailang/internal/core"
)

// M-V1-NOMINAL-TYPE-IDENTITY, review round 3: an ambiguous constructor can
// shape inference without being an error itself —
//
//   - while its choice is open, the variables it depends on are kept
//     monomorphic at a let (literalWithhold), so a let-bound function matching
//     it cannot be used at two types (mp1, mp2, c_poly);
//   - a pattern whose arity fits exactly one candidate is read as that one,
//     and a later conflict with another candidate's type follows (ca2b).
//
// The failure then surfaces as a plain unification error at a use site that
// does not mention the constructor. A ctorNote records each such decision, and
// explainCtorFailure turns a unification failure it explains into
// TC_AMBIGUOUS_CTOR_001 at the pattern, with the failure as its cause. This
// changes no verdict: it only names the cause of an error that happens anyway.
//
// For a monomorphic let the explanation must be TRUE: the failure has to be a
// conflict between two uses of the binding, i.e. one that a generic binding
// would not have. usesConflict establishes that by dry runs: every call of the
// binding on its own (the other calls dropped) must solve — equalities and
// constructor choices — while all of them together fail. One call, or a call
// that is wrong by itself (probe b4: `let r: string = f(mkb())`), keeps the
// plain error. Uses other than calls are not counted, so the rule under-fires
// rather than over-fires.
type ctorNote struct {
	pattern *core.ConstructorPattern
	all     []CtorCandidate
	// A let kept monomorphic while the choice was open:
	binder string
	vars   map[string]bool
	// A pattern read by its arity alone:
	chosen *CtorCandidate
}

// noteMonomorphicLet records that binder's generalisation withheld vars
// because the open choices depend on them.
func (ctx *InferenceContext) noteMonomorphicLet(binder core.CoreExpr) {
	if len(ctx.pendingCtorVars) == 0 {
		return
	}
	open := ctx.openChoices()
	if len(open) == 0 {
		return
	}
	vars := make(map[string]bool, len(ctx.pendingCtorVars))
	for v := range ctx.pendingCtorVars {
		vars[v] = true
	}
	ctx.ctorNotes = append(ctx.ctorNotes, &ctorNote{pattern: open[0].pattern, all: open[0].all, binder: binderName(binder), vars: vars})
}

func binderName(b core.CoreExpr) string {
	switch x := b.(type) {
	case *core.Let:
		return x.Name
	case *core.LetRec:
		if len(x.Bindings) == 1 {
			return x.Bindings[0].Name
		}
	}
	return ""
}

// unifyFailure is the error of a failed equality eq (under sub): the
// unification failure itself, or TC_AMBIGUOUS_CTOR_001 when an ambiguous
// constructor is behind it.
func (ctx *InferenceContext) unifyFailure(eq TypeEq, sub Substitution, err error) error {
	failure := fmt.Errorf("type unification failed at %v: %w", eq.Path, err)
	if cause := ctx.explainCtorFailure(eq, sub, failure); cause != nil {
		return cause
	}
	return failure
}

// explainCtorFailure returns TC_AMBIGUOUS_CTOR_001 explaining failure (the
// unification of eq under sub) when a recorded ambiguous-constructor decision
// is involved in it, else nil.
func (ctx *InferenceContext) explainCtorFailure(eq TypeEq, sub Substitution, failure error) error {
	if len(ctx.ctorNotes) == 0 || !nominalAmbiguityIsError {
		return nil
	}
	raw := make(map[string]bool)
	collectFreeVars(eq.Left, raw)
	collectFreeVars(eq.Right, raw)
	heads := make(map[string]bool)
	collectTConKeys(ApplySubstitution(sub, eq.Left), heads, 0)
	collectTConKeys(ApplySubstitution(sub, eq.Right), heads, 0)
	for _, n := range ctx.ctorNotes {
		if n.vars != nil && overlaps(n.vars, raw) && n.binder != "" && ctx.usesConflict(n.binder) {
			what := "a function"
			if n.binder != "" {
				what = fmt.Sprintf("`%s`", n.binder)
			}
			return &AmbiguousCtorError{Name: n.pattern.Name, Candidates: n.all, Where: patternWhere(n.pattern), Cause: failure,
				Detail: fmt.Sprintf("constructor '%s' belongs to %s, and which one this pattern means is not yet decided where %s is defined, so %s cannot be generic: all its uses must have one type — they do not.",
					n.pattern.Name, candidateList(n.all), what, what)}
		}
		if n.chosen != nil && heads[NominalKey(n.chosen.Module, n.chosen.TypeName)] {
			for _, c := range n.all {
				if c != *n.chosen && heads[NominalKey(c.Module, c.TypeName)] {
					return arityError(n.pattern, c, n.all, failure)
				}
			}
		}
	}
	return nil
}

// usesConflict reports whether the calls of binder conflict with each other:
// at least two of them, each of which solves on its own (the other calls'
// equalities left out), dry-run over the current constraints.
func (ctx *InferenceContext) usesConflict(binder string) bool {
	prefix := "function application of " + binder + " at "
	var calls []int
	for i, c := range ctx.constraints {
		if eq, ok := c.(TypeEq); ok && len(eq.Path) > 0 && strings.HasPrefix(eq.Path[0], prefix) {
			calls = append(calls, i)
		}
	}
	if len(calls) < 2 {
		return false
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
		if !ctx.solvesDry(cs) {
			return false
		}
	}
	return true
}

// solvesDry reports whether cs solves — every equality, then the constructor
// choices — without touching the context (its fresh-name counter is restored).
func (ctx *InferenceContext) solvesDry(cs []TypeConstraint) bool {
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
	_, err := ctx.resolveChoicesIn(sub, cs, map[*ctorChoice]bool{})
	return err == nil
}

func overlaps(a, b map[string]bool) bool {
	for v := range a {
		if b[v] {
			return true
		}
	}
	return false
}

// collectTConKeys adds the NominalKey of every type constructor in t.
// Cycle-safe: types are finite trees after substitution; depth bounds a
// malformed one.
func collectTConKeys(t Type, out map[string]bool, depth int) {
	if t == nil || depth > 64 {
		return
	}
	switch x := t.(type) {
	case *TCon:
		out[NominalKey(x.Module, x.Name)] = true
	case *TApp:
		collectTConKeys(x.Constructor, out, depth+1)
		for _, a := range x.Args {
			collectTConKeys(a, out, depth+1)
		}
	case *TFunc2:
		for _, p := range x.Params {
			collectTConKeys(p, out, depth+1)
		}
		collectTConKeys(x.Return, out, depth+1)
	case *TList:
		collectTConKeys(x.Element, out, depth+1)
	case *TTuple:
		for _, e := range x.Elements {
			collectTConKeys(e, out, depth+1)
		}
	}
}
