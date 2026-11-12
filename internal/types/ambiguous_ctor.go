package types

import (
	"fmt"
	"sort"
	"strings"

	"github.com/sunholo-data/ailang/internal/ast"
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
	Where      string          // "at file:line:col" (the pattern or expression), or a description
	Scrutinee  string          // non-empty: the scrutinee's type is known and matches none
	Expression bool            // an expression use (decided by the elaborator; no annotation helps)
	// Detail, when set, replaces the generic explanation (arity, monomorphism);
	// Cause is the unification failure it explains.
	Detail string
	Cause  error
	// Meant, when set, is the candidate the suggestion should name (the type
	// matched, for an arity error) instead of the first one.
	Meant *CtorCandidate
}

// NewAmbiguousCtorError builds TC_AMBIGUOUS_CTOR_001 for a constructor
// EXPRESSION (the elaborator's case); where is "at file:line:col".
func NewAmbiguousCtorError(name string, cands []CtorCandidate, where string) *AmbiguousCtorError {
	return &AmbiguousCtorError{Name: name, Candidates: cands, Where: where, Expression: true}
}

func (e *AmbiguousCtorError) Unwrap() error { return e.Cause }

func (e *AmbiguousCtorError) Error() string {
	parts := make([]string, len(e.Candidates))
	for i, c := range e.Candidates {
		parts[i] = fmt.Sprintf("%s's %s", c.Module, c.TypeName)
	}
	example := ""
	if len(e.Candidates) > 0 {
		example = fmt.Sprintf("import %s (%s)", e.Candidates[0].Module, e.Name)
	}
	if e.Meant != nil {
		// An arity error: the fix is the pattern's argument count.
		holes := make([]string, e.Meant.Arity)
		for i := range holes {
			holes[i] = "_"
		}
		msg := fmt.Sprintf("%s %s: %s", TC_AMBIGUOUS_CTOR_001, e.Where, e.Detail)
		if e.Cause != nil {
			msg += "\n  Caused: " + e.Cause.Error()
		}
		return msg + fmt.Sprintf("\n  Suggestion: give the pattern %d argument(s), %s(%s), to match %s's %s.",
			e.Meant.Arity, e.Name, strings.Join(holes, ", "), e.Meant.Module, e.Meant.TypeName)
	}
	switch {
	case e.Detail != "":
		msg := fmt.Sprintf("%s %s: %s", TC_AMBIGUOUS_CTOR_001, e.Where, e.Detail)
		if e.Cause != nil {
			msg += "\n  Caused: " + e.Cause.Error()
		}
		return msg + fmt.Sprintf("\n  Suggestion: import the constructor by name from the module you mean, e.g. %s.", example)
	case e.Scrutinee != "":
		return fmt.Sprintf("%s %s: constructor '%s' belongs to %s, but the value matched has type %s.\n"+
			"  Suggestion: match a value of one of those types, or import the constructor you mean by name, e.g. %s.",
			TC_AMBIGUOUS_CTOR_001, e.Where, e.Name, strings.Join(parts, " and "), e.Scrutinee, example)
	case e.Expression:
		// An expression is decided where it is written, by its imports only.
		return fmt.Sprintf("%s %s: constructor '%s' is ambiguous: it belongs to %s, all imported, and nothing here says which one is meant.\n"+
			"  Suggestion: import the constructor by name from the module you mean, e.g. %s (or alias one: import %s (%s as Other%s)).",
			TC_AMBIGUOUS_CTOR_001, e.Where, e.Name, strings.Join(parts, " and "), example, e.Candidates[len(e.Candidates)-1].Module, e.Name, e.Name)
	}
	return fmt.Sprintf("%s %s: constructor '%s' is ambiguous: it belongs to %s, all imported, and nothing in this declaration says which one the value matched is.\n"+
		"  Suggestion: import the constructor by name from the module you mean, e.g. %s, or annotate the value matched.",
		TC_AMBIGUOUS_CTOR_001, e.Where, e.Name, strings.Join(parts, " and "), example)
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
	cands    []CtorCandidate // the candidates the pattern's arity allows
	all      []CtorCandidate // every candidate (for arity diagnostics)
	schemes  []*Scheme       // factory scheme of each candidate (nil if unavailable)
	path     []string
	resolved bool
}

func (c *ctorChoice) constraint() {}
func (c *ctorChoice) String() string {
	return fmt.Sprintf("choose %s for %s", c.pattern.Name, c.scrut)
}

// patternWhere renders a constructor pattern's position for a diagnostic.
func patternWhere(p *core.ConstructorPattern) string {
	if p.Pos == (ast.Pos{}) {
		return "in a pattern"
	}
	if p.Pos.File == "" {
		return fmt.Sprintf("at %d:%d", p.Pos.Line, p.Pos.Column)
	}
	return fmt.Sprintf("at %s:%d:%d", p.Pos.File, p.Pos.Line, p.Pos.Column)
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
	// A scrutinee whose type is already known to be a candidate the arity
	// excluded is an arity error about THAT constructor (round 3, ca2).
	if head := ctx.choiceHead(scrutType); head != nil {
		if c, ok := candidateOf(cands, head); ok && c.Arity != len(p.Args) {
			return nil, nil, arityError(p, c, cands, nil)
		}
	}
	switch len(fit) {
	case 0:
		return nil, nil, &AmbiguousCtorError{Name: p.Name, Candidates: cands, Where: patternWhere(p),
			Detail: fmt.Sprintf("constructor pattern %s has %d argument(s), but no imported constructor %s takes %d: %s.",
				p.Name, len(p.Args), p.Name, len(p.Args), describeCandidates(cands))}
	case 1:
		p.TypeName, p.TypeModule = fit[0].TypeName, fit[0].Module
		if len(cands) > len(fit) {
			// Read by its arity alone: a later conflict with an excluded
			// candidate's type is explained by this pattern (ca2b).
			ctx.ctorNotes = append(ctx.ctorNotes, &ctorNote{pattern: p, all: cands, chosen: &fit[0]})
		}
		return tc.checkPattern(p, scrutType, ctx)
	}
	choice := &ctorChoice{pattern: p, scrut: scrutType, cands: fit, all: cands,
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

// candidateOf returns the candidate whose ADT is head.
func candidateOf(cands []CtorCandidate, head *TCon) (CtorCandidate, bool) {
	for _, c := range cands {
		if c.TypeName == head.Name && c.Module == head.Module {
			return c, true
		}
	}
	return CtorCandidate{}, false
}

// arityError is TC_AMBIGUOUS_CTOR_001 for a pattern whose argument count fits
// some imported constructor of the name, but not the one of the type matched.
func arityError(p *core.ConstructorPattern, meant CtorCandidate, all []CtorCandidate, cause error) error {
	return &AmbiguousCtorError{Name: p.Name, Candidates: all, Where: patternWhere(p), Cause: cause, Meant: &meant,
		Detail: fmt.Sprintf("constructor pattern %s has %d argument(s), but the value matched has type %s, whose %s takes %d. (The imported constructors named %s are %s.)",
			p.Name, len(p.Args), describeTCon(&TCon{Name: meant.TypeName, Module: meant.Module}), p.Name, meant.Arity,
			p.Name, describeCandidates(all))}
}

func candidateRefs(cands []CtorCandidate) []TypeRef {
	out := make([]TypeRef, len(cands))
	for i, c := range cands {
		out[i] = TypeRef{Module: c.Module, Name: c.TypeName}
	}
	return out
}

// candidateList renders candidates as "q/a's T and q/b's U".
func candidateList(cands []CtorCandidate) string {
	parts := make([]string, len(cands))
	for i, c := range cands {
		parts[i] = fmt.Sprintf("%s's %s", c.Module, c.TypeName)
	}
	return strings.Join(parts, " and ")
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
	return ctx.resolveChoicesIn(sub, ctx.constraints, nil)
}

// resolveChoicesIn is resolveCtorChoices over cs. With dry non-nil it is a
// DRY RUN for a diagnostic (ambiguous_ctor_cause.go): choices resolved are
// recorded in dry instead of on the choice, and nothing is mutated, observed
// or added to the context.
func (ctx *InferenceContext) resolveChoicesIn(sub Substitution, cs []TypeConstraint, dry map[*ctorChoice]bool) (Substitution, error) {
	for progress := true; progress; {
		progress = false
		for _, c := range cs {
			choice, ok := c.(*ctorChoice)
			// A dry run re-decides every choice from its own constraints.
			if !ok || (dry == nil && choice.resolved) || dry[choice] {
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
				if c, ok := candidateOf(choice.all, head); ok {
					return nil, arityError(choice.pattern, c, choice.all, nil)
				}
				return nil, &AmbiguousCtorError{Name: choice.pattern.Name, Candidates: choice.cands,
					Where: patternWhere(choice.pattern), Scrutinee: describeTCon(head)}
			}
			if dry != nil {
				dry[choice] = true
			} else {
				choice.resolved = true
				cand := choice.cands[idx]
				choice.pattern.TypeName, choice.pattern.TypeModule = cand.TypeName, cand.Module
				ObserveAmbiguity(AmbiguityEvent{Kind: "ctor-pattern-resolved", Name: choice.pattern.Name, Candidates: candidateRefs(choice.cands)})
			}
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
				e := eq.(TypeEq)
				var err error
				if dry != nil {
					if sub, err = ctx.unifier.Unify(ApplySubstitution(sub, e.Left), ApplySubstitution(sub, e.Right), sub); err != nil {
						return nil, err
					}
					continue
				}
				ctx.derivedEqs = append(ctx.derivedEqs, len(ctx.constraints))
				ctx.addConstraint(eq)
				sub, err = ctx.unifier.Unify(ApplySubstitution(sub, e.Left), ApplySubstitution(sub, e.Right), sub)
				if err != nil {
					return nil, ctx.unifyFailure(e, sub, err)
				}
			}
			progress = true
		}
	}
	if dry != nil {
		return sub, nil
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
	return ctx.unifier.nominalHead(t)
}

// openChoices returns the choices still open, sorted by name, then by
// candidates, then by position, so neither a message nor an observed event
// depends on the order the checker visited the patterns.
func (ctx *InferenceContext) openChoices() []*ctorChoice {
	var open []*ctorChoice
	for _, c := range ctx.constraints {
		if choice, ok := c.(*ctorChoice); ok && !choice.resolved {
			open = append(open, choice)
		}
	}
	sort.SliceStable(open, func(i, j int) bool {
		a, b := open[i], open[j]
		if a.pattern.Name != b.pattern.Name {
			return a.pattern.Name < b.pattern.Name
		}
		if da, db := describeCandidates(a.cands), describeCandidates(b.cands); da != db {
			return da < db
		}
		if a.pattern.Pos.Line != b.pattern.Pos.Line {
			return a.pattern.Pos.Line < b.pattern.Pos.Line
		}
		return a.pattern.Pos.Column < b.pattern.Pos.Column
	})
	return open
}

// openCtorChoiceError returns TC_AMBIGUOUS_CTOR_001 for a choice still open
// at the end of the top-level declaration.
func (ctx *InferenceContext) openCtorChoiceError() error {
	if !nominalAmbiguityIsError {
		return nil
	}
	open := ctx.openChoices()
	if len(open) == 0 {
		return nil
	}
	ObserveAmbiguity(AmbiguityEvent{Kind: "ctor-pattern-unresolved", Name: open[0].pattern.Name, Candidates: candidateRefs(open[0].cands)})
	return &AmbiguousCtorError{Name: open[0].pattern.Name, Candidates: open[0].cands, Where: patternWhere(open[0].pattern)}
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
