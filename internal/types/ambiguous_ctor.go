package types

import (
	"fmt"
	"sort"
	"strings"

	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/typedast"
)

// TC_AMBIGUOUS_CTOR_001 codes a constructor name that two or more imported
// modules offer through the constructor auto-import (M-CTOR-AUTO), that no
// import names explicitly and that nothing in the program decides
// (M-V1-NOMINAL-TYPE-IDENTITY). It used to mean whichever module was imported
// first.
const TC_AMBIGUOUS_CTOR_001 = "TC_AMBIGUOUS_CTOR_001"

// CtorCandidate is one module's constructor a bare constructor name may mean.
type CtorCandidate struct {
	Module   string // the ADT's defining module
	TypeName string // the ADT
	Arity    int    // the constructor's field count
}

// AmbiguousCtorError is TC_AMBIGUOUS_CTOR_001.
type AmbiguousCtorError struct {
	Name       string
	Candidates []CtorCandidate // sorted by module
	Where      string
	Scrutinee  string // non-empty: the scrutinee's type is known and matches none
}

// NewAmbiguousCtorError builds TC_AMBIGUOUS_CTOR_001.
func NewAmbiguousCtorError(name string, cands []CtorCandidate, where string) *AmbiguousCtorError {
	return &AmbiguousCtorError{Name: name, Candidates: cands, Where: where}
}

func (e *AmbiguousCtorError) Error() string {
	parts := make([]string, len(e.Candidates))
	for i, c := range e.Candidates {
		parts[i] = fmt.Sprintf("%s's %s", c.Module, c.TypeName)
	}
	first := ""
	if len(e.Candidates) > 0 {
		first = e.Candidates[0].Module
	}
	if e.Scrutinee != "" {
		return fmt.Sprintf("%s %s: constructor '%s' belongs to %s, but the value matched has type %s.\n"+
			"  Suggestion: match a value of one of those types, or import the constructor you mean by name, e.g. import %s (%s).",
			TC_AMBIGUOUS_CTOR_001, e.Where, e.Name, strings.Join(parts, " and "), e.Scrutinee, first, e.Name)
	}
	return fmt.Sprintf("%s %s: constructor '%s' is ambiguous: it belongs to %s, all imported, and nothing here says which one is meant.\n"+
		"  Suggestion: import the constructor by name from the module you mean, e.g. import %s (%s), or annotate the value matched.",
		TC_AMBIGUOUS_CTOR_001, e.Where, e.Name, strings.Join(parts, " and "), first, e.Name)
}

// SetAmbiguousConstructors installs the auto-imported constructor names that
// two or more modules offer (constructor -> candidates).
func (tc *CoreTypeChecker) SetAmbiguousConstructors(m map[string][]CtorCandidate) {
	tc.ambiguousCtors = m
}

// ctorChoice is a DEFERRED choice between the candidates of an ambiguous
// constructor pattern (M-V1-NOMINAL-TYPE-IDENTITY). Like a class constraint it
// sits in the inference context's constraint list; SolveConstraints resolves
// it once the full set of equalities collected so far fixes the scrutinee's
// type, and adds the chosen constructor's equalities. Until then the
// scrutinee's variables are withheld from let-generalisation (as an integer
// literal's are), so a later use can still decide it. A choice still open when
// its top-level declaration is solved is TC_AMBIGUOUS_CTOR_001. The choice
// never depends on the order constraints were visited: it is made from the
// substitution of all of them, and a head once determined cannot change.
type ctorChoice struct {
	pattern  *core.ConstructorPattern
	scrut    Type
	args     []Type
	cands    []CtorCandidate
	schemes  []*Scheme // factory scheme of each candidate (nil if unavailable)
	path     []string
	resolved bool
}

func (c *ctorChoice) constraint() {}
func (c *ctorChoice) String() string {
	return fmt.Sprintf("choose %s for %s", c.pattern.Name, c.scrut)
}

// checkAmbiguousCtorPattern types a pattern whose constructor two or more
// imported modules offer. The pattern's own arity (syntax, never a candidate's
// position in a list) narrows the candidates first; one left is resolved at
// once, otherwise the choice is deferred.
func (tc *CoreTypeChecker) checkAmbiguousCtorPattern(p *core.ConstructorPattern, cands []CtorCandidate, scrutType Type, ctx *InferenceContext) (map[string]Type, typedast.TypedPattern, error) {
	var fit []CtorCandidate
	for _, c := range cands {
		if c.Arity == len(p.Args) {
			fit = append(fit, c)
		}
	}
	ObserveAmbiguity(AmbiguityEvent{Kind: "ctor-pattern", Name: p.Name, Candidates: candidateRefs(cands)})
	switch len(fit) {
	case 0:
		return nil, nil, fmt.Errorf("%s: constructor pattern %s has %d argument(s), but no imported constructor %s takes %d: %s",
			TC_AMBIGUOUS_CTOR_001, p.Name, len(p.Args), p.Name, len(p.Args), describeCandidates(cands))
	case 1:
		p.TypeName, p.TypeModule = fit[0].TypeName, fit[0].Module
		return tc.checkPattern(p, scrutType, ctx)
	}
	choice := &ctorChoice{pattern: p, scrut: scrutType, cands: fit,
		path: []string{fmt.Sprintf("constructor pattern %s", p.Name)}}
	for _, c := range fit {
		choice.schemes = append(choice.schemes, tc.globalTypes[core.ADTFactoryKey(c.Module, c.TypeName, p.Name)])
	}
	bindings := make(map[string]Type)
	typedArgs := make([]typedast.TypedPattern, len(p.Args))
	for i, argPat := range p.Args {
		argType := ctx.freshTypeVar()
		choice.args = append(choice.args, argType)
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
	ctx.addConstraint(choice)
	return bindings, typedast.TypedConstructorPattern{Name: p.Name, Args: typedArgs}, nil
}

func candidateRefs(cands []CtorCandidate) []TypeRef {
	out := make([]TypeRef, len(cands))
	for i, c := range cands {
		out[i] = TypeRef{Module: c.Module, Name: c.TypeName}
	}
	return out
}

func describeCandidates(cands []CtorCandidate) string {
	parts := make([]string, len(cands))
	for i, c := range cands {
		parts[i] = fmt.Sprintf("%s's %s (%d field(s))", c.Module, c.TypeName, c.Arity)
	}
	return strings.Join(parts, ", ")
}

// resolveCtorChoices resolves every open choice whose scrutinee head the
// substitution fixes, adding the chosen constructor's equalities, until no
// more progress is made. It returns the extended substitution. Resolution is
// monotone: it only ever adds constraints, so the result does not depend on
// the order the choices are tried in.
func (ctx *InferenceContext) resolveCtorChoices(sub Substitution) (Substitution, error) {
	for progress := true; progress; {
		progress = false
		for _, c := range ctx.constraints {
			choice, ok := c.(*ctorChoice)
			if !ok || choice.resolved {
				continue
			}
			head := ctx.choiceHead(ApplySubstitution(sub, choice.scrut))
			if head == nil {
				continue
			}
			idx := -1
			for i, cand := range choice.cands {
				if cand.TypeName == head.Name && cand.Module == head.Module {
					idx = i
				}
			}
			if idx < 0 {
				return nil, &AmbiguousCtorError{Name: choice.pattern.Name, Candidates: choice.cands,
					Where: "in a pattern", Scrutinee: describeTCon(head)}
			}
			choice.resolved = true
			cand := choice.cands[idx]
			choice.pattern.TypeName, choice.pattern.TypeModule = cand.TypeName, cand.Module
			ObserveAmbiguity(AmbiguityEvent{Kind: "ctor-pattern-resolved", Name: choice.pattern.Name, Candidates: candidateRefs(choice.cands)})
			var eqs []TypeConstraint
			if scheme := choice.schemes[idx]; scheme != nil {
				switch t := scheme.Instantiate(ctx.freshType).(type) {
				case *TFunc2:
					eqs = append(eqs, TypeEq{Left: choice.scrut, Right: t.Return, Path: choice.path})
					for i := range choice.args {
						if i < len(t.Params) {
							eqs = append(eqs, TypeEq{Left: choice.args[i], Right: t.Params[i], Path: choice.path})
						}
					}
				default:
					eqs = append(eqs, TypeEq{Left: choice.scrut, Right: t, Path: choice.path})
				}
			}
			for _, eq := range eqs {
				ctx.addConstraint(eq)
				e := eq.(TypeEq)
				var err error
				sub, err = ctx.unifier.Unify(ApplySubstitution(sub, e.Left), ApplySubstitution(sub, e.Right), sub)
				if err != nil {
					return nil, fmt.Errorf("type unification failed at %v: %w", e.Path, err)
				}
			}
			progress = true
		}
	}
	ctx.pendingCtorVars = nil
	for _, c := range ctx.constraints {
		if choice, ok := c.(*ctorChoice); ok && !choice.resolved {
			if ctx.pendingCtorVars == nil {
				ctx.pendingCtorVars = make(map[string]bool)
			}
			collectFreeVars(ApplySubstitution(sub, choice.scrut), ctx.pendingCtorVars)
			for _, a := range choice.args {
				collectFreeVars(ApplySubstitution(sub, a), ctx.pendingCtorVars)
			}
		}
	}
	return sub, nil
}

// choiceHead returns the nominal head of a scrutinee type (aliases expanded
// by their defining module first), or nil while it is not yet known.
func (ctx *InferenceContext) choiceHead(t Type) *TCon {
	if ctx.unifier != nil {
		t = ctx.unifier.expandAliasMode(t, false)
	}
	if app, ok := t.(*TApp); ok {
		t, _ = decomposeApp(app)
	}
	if head, ok := t.(*TCon); ok {
		return head
	}
	return nil
}

// openCtorChoiceError returns TC_AMBIGUOUS_CTOR_001 for the first (in source
// order of the patterns' names, for a stable message) choice still open.
func (ctx *InferenceContext) openCtorChoiceError() error {
	if !nominalAmbiguityIsError {
		return nil
	}
	var open []*ctorChoice
	for _, c := range ctx.constraints {
		if choice, ok := c.(*ctorChoice); ok && !choice.resolved {
			open = append(open, choice)
		}
	}
	if len(open) == 0 {
		return nil
	}
	// Sorted by name, then by candidate list, so neither the message nor the
	// observed event depends on the order the checker visited the patterns.
	sort.SliceStable(open, func(i, j int) bool {
		if open[i].pattern.Name != open[j].pattern.Name {
			return open[i].pattern.Name < open[j].pattern.Name
		}
		return describeCandidates(open[i].cands) < describeCandidates(open[j].cands)
	})
	ObserveAmbiguity(AmbiguityEvent{Kind: "ctor-pattern-unresolved", Name: open[0].pattern.Name, Candidates: candidateRefs(open[0].cands)})
	return NewAmbiguousCtorError(open[0].pattern.Name, open[0].cands, "in a pattern")
}

// solveDeclConstraints is SolveConstraints at the end of a top-level
// declaration: every deferred constructor choice must be decided by now
// (TC_AMBIGUOUS_CTOR_001 otherwise).
func (ctx *InferenceContext) solveDeclConstraints() (Substitution, []ClassConstraint, error) {
	sub, unsolved, err := ctx.SolveConstraints()
	if err != nil {
		return nil, nil, err
	}
	if err := ctx.openCtorChoiceError(); err != nil {
		return nil, nil, err
	}
	return sub, unsolved, nil
}
