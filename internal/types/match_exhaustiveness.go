package types

import (
	"errors"
	"fmt"
	"sort"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/types/exhaust"
)

// Match exhaustiveness as a checker verdict (M-V1-MATCH-EXHAUSTIVENESS M2/M3).
//
// VerifyMatchExhaustiveness runs after inference, on Core whose types are
// final, and before dictionary elaboration, specialisation and op-lowering
// (which only copy matches), so it sees exactly the user's matches. For each
// one it asks internal/types/exhaust whether the unguarded arms cover every
// value of the scrutinee's type and which arms can never be reached.
//
// A constructor pattern's ADT is its NOMINAL identity (defining module, name):
// the elaborator's stamp (TypeModule, TypeName), else — for a constructor
// matched by name (#323) — the head of the type the checker matched it
// against, else the only loaded ADT that declares the name. Its universe is
// that ADT's complete constructor list (SetMatchUniverse), never "the
// constructors in scope": a module that imports one constructor of a type
// still has to cover all of them.
//
// Verdicts:
//   - not exhaustive          -> TC_MATCH_002 (error) naming concrete missing patterns
//   - an arm never reached    -> TC_MATCH_003 (warning)
//   - step budget exceeded    -> TC_MATCH_004 (error) unless an unguarded `_` arm closes the match
//   - wrong constructor arity -> TC_MATCH_006 (error): the evaluator never
//     matches such a pattern while the VM binds a prefix of the fields (#2243),
//     so its coverage is not one thing
//
// Anything the check cannot decide (no universe for a type, a pattern that is
// not one of its type's constructors) is an error too, unless an unguarded
// catch-all arm makes the answer obvious: never a silent "exhaustive".

// Diagnostic codes.
const (
	TC_MATCH_002 = "TC_MATCH_002" // non-exhaustive match
	TC_MATCH_003 = "TC_MATCH_003" // unreachable (redundant) arm — warning
	TC_MATCH_004 = "TC_MATCH_004" // step budget exceeded
	TC_MATCH_006 = "TC_MATCH_006" // constructor pattern with the wrong number of arguments
)

// matchVerdictIsError makes the check's errors errors. Only the run-time
// match-failure tests turn it off: RT002 is defence in depth that a checked
// program cannot reach, so its tests need programs the check would reject.
// There is no flag, setting or environment variable that reaches it.
var matchVerdictIsError = true

// SetMatchExhaustivenessErrorsForTest chooses whether match-check errors are
// reported and returns a function restoring the previous setting. Tests only.
func SetMatchExhaustivenessErrorsForTest(asError bool) (restore func()) {
	prev := matchVerdictIsError
	matchVerdictIsError = asError
	return func() { matchVerdictIsError = prev }
}

// matchState is the type checker's exhaustiveness state: every ADT's complete
// constructor list, and the type each constructor pattern the elaborator left
// unstamped was checked against (kept current by the same substitutions as
// CoreTI).
type matchState struct {
	matchUniverse    MatchUniverse
	ctorPatternTypes map[*core.ConstructorPattern]Type
}

// MatchUniverse maps an ADT's NominalKey(module, name) to its complete
// constructor list, sorted by name (interfaces carry no declaration order).
type MatchUniverse map[string][]exhaust.CtorSig

// SetMatchUniverse installs the constructor universe of every ADT the module
// can see: its own declarations and every loaded interface's.
func (tc *CoreTypeChecker) SetMatchUniverse(u MatchUniverse) { tc.matchUniverse = u }

// recordCtorPatternType remembers the type an unstamped constructor pattern
// was checked against; substituteCtorPatternTypes keeps it current.
func (tc *CoreTypeChecker) recordCtorPatternType(p *core.ConstructorPattern, t Type) {
	if p.TypeName != "" {
		return
	}
	if tc.ctorPatternTypes == nil {
		tc.ctorPatternTypes = make(map[*core.ConstructorPattern]Type)
	}
	tc.ctorPatternTypes[p] = t
}

func (tc *CoreTypeChecker) substituteCtorPatternTypes(sub Substitution) {
	if len(sub) == 0 {
		return
	}
	for p, t := range tc.ctorPatternTypes {
		tc.ctorPatternTypes[p] = ApplySubstitution(sub, t)
	}
}

// VerifyMatchExhaustiveness checks every match in exprs. It returns the
// TC_MATCH_003 warnings, and every error as one *MatchErrors (sorted by
// position) so an author can fix them in one pass. It sets each checked
// match's Exhaustive flag.
func (tc *CoreTypeChecker) VerifyMatchExhaustiveness(exprs ...core.CoreExpr) ([]*MatchWarning, error) {
	v := &matchVerifier{tc: tc, byName: tc.matchUniverse.byName()}
	for _, e := range exprs {
		v.walk(e)
	}
	sort.SliceStable(v.warns, func(i, j int) bool {
		if v.warns[i].Pos != v.warns[j].Pos {
			return posLess(v.warns[i].Pos, v.warns[j].Pos)
		}
		return v.warns[i].Arm < v.warns[j].Arm
	})
	if len(v.errs) == 0 || !matchVerdictIsError {
		return v.warns, nil
	}
	sort.SliceStable(v.errs, func(i, j int) bool { return posLess(v.errs[i].Pos, v.errs[j].Pos) })
	return v.warns, MatchErrors(v.errs)
}

type matchVerifier struct {
	tc     *CoreTypeChecker
	byName map[string][]string // constructor name -> universe keys declaring it
	unif   *Unifier
	errs   []*MatchError
	warns  []*MatchWarning
}

func (u MatchUniverse) byName() map[string][]string {
	out := make(map[string][]string)
	for key, sigs := range u {
		for _, s := range sigs {
			out[s.Name] = append(out[s.Name], key)
		}
	}
	return out
}

// walk visits every sub-expression. Exhaustive over Core node kinds: a new
// kind must be added here, or the matches inside it would go unchecked.
func (v *matchVerifier) walk(expr core.CoreExpr) {
	if expr == nil {
		return
	}
	switch e := expr.(type) {
	case *core.Var, *core.VarGlobal, *core.Lit, *core.DictRef:
	case *core.Lambda:
		v.walk(e.Body)
	case *core.Let:
		v.walk(e.Value)
		v.walk(e.Body)
	case *core.LetRec:
		for _, b := range e.Bindings {
			v.walk(b.Value)
		}
		v.walk(e.Body)
	case *core.App:
		v.walk(e.Func)
		v.walkAll(e.Args)
	case *core.If:
		v.walk(e.Cond)
		v.walk(e.Then)
		v.walk(e.Else)
	case *core.Match:
		v.walk(e.Scrutinee)
		for _, arm := range e.Arms {
			v.walk(arm.Guard)
			v.walk(arm.Body)
		}
		v.check(e)
	case *core.BinOp:
		v.walk(e.Left)
		v.walk(e.Right)
	case *core.UnOp:
		v.walk(e.Operand)
	case *core.Intrinsic:
		v.walkAll(e.Args)
	case *core.Record:
		for _, f := range e.Fields {
			v.walk(f)
		}
	case *core.RecordAccess:
		v.walk(e.Record)
	case *core.RecordUpdate:
		v.walk(e.Base)
		for _, f := range e.Updates {
			v.walk(f)
		}
	case *core.List:
		v.walkAll(e.Elements)
	case *core.Array:
		v.walkAll(e.Elements)
	case *core.Tuple:
		v.walkAll(e.Elements)
	case *core.DictAbs:
		v.walk(e.Body)
	case *core.DictApp:
		v.walk(e.Dict)
		v.walkAll(e.Args)
	case *core.Forall:
		v.walk(e.Body)
	default:
		panic(fmt.Sprintf("VerifyMatchExhaustiveness: unhandled Core node %T", expr))
	}
}

func (v *matchVerifier) walkAll(es []core.CoreExpr) {
	for _, e := range es {
		v.walk(e)
	}
}

// errUndecidable marks a pattern the check cannot place in a universe.
var errUndecidable = errors.New("cannot be checked for exhaustiveness")

// check decides one match.
func (v *matchVerifier) check(m *core.Match) {
	pos := matchPosition(m)
	arms := make([]exhaust.Arm, len(m.Arms))
	closed := false // an unguarded `_` / variable arm covers everything
	guarded := false
	for i, arm := range m.Arms {
		p, err := v.pattern(arm.Pattern)
		if err != nil {
			var ae *patternArityError
			if errors.As(err, &ae) {
				v.errs = append(v.errs, newPatternArityError(pos, ae))
				return
			}
			if hasCatchAll(m) {
				m.Exhaustive = true
				return
			}
			v.errs = append(v.errs, newUndecidedError(pos, err))
			return
		}
		arms[i] = exhaust.Arm{Pattern: p, Guarded: arm.Guard != nil}
		if arm.Guard != nil {
			guarded = true
		} else if isCatchAll(arm.Pattern) {
			closed = true
		}
	}
	if err := v.crossCheckRoot(m, arms); err != nil {
		v.errs = append(v.errs, newUndecidedError(pos, err))
		return
	}
	res, err := exhaust.Check(arms, exhaust.StaticUniverse(v.tc.matchUniverse), exhaust.Options{})
	if err != nil {
		if closed {
			m.Exhaustive = true // the answer does not depend on the rest
			return
		}
		if errors.Is(err, exhaust.ErrBudget) {
			v.errs = append(v.errs, newBudgetError(pos, err))
		} else {
			v.errs = append(v.errs, newUndecidedError(pos, err))
		}
		return
	}
	m.Exhaustive = res.Exhaustive
	if !res.Exhaustive {
		v.errs = append(v.errs, newNonExhaustiveError(pos, v.scrutineeName(m), res.Missing, guarded))
	}
	for _, i := range res.Redundant {
		v.warns = append(v.warns, &MatchWarning{Pos: pos, Arm: i + 1, Pattern: exhaust.Render(arms[i].Pattern)})
	}
}

// pattern converts a Core pattern, resolving each constructor's ADT.
func (v *matchVerifier) pattern(p core.CorePattern) (exhaust.Pattern, error) {
	switch p := p.(type) {
	case *core.WildcardPattern, *core.VarPattern:
		return exhaust.Wild{}, nil
	case *core.LitPattern:
		if f, ok := p.Value.(float32); ok {
			return exhaust.Lit{Value: float64(f)}, nil
		}
		return exhaust.Lit{Value: p.Value}, nil
	case *core.ConstructorPattern:
		key, sigs, err := v.ctorUniverse(p)
		if err != nil {
			return nil, err
		}
		for _, s := range sigs {
			if s.Name == p.Name && s.Arity != len(p.Args) {
				return nil, &patternArityError{pattern: p, arity: s.Arity}
			}
		}
		args, err := v.patterns(p.Args)
		if err != nil {
			return nil, err
		}
		return exhaust.Ctor{TypeName: key, Name: p.Name, Args: args}, nil
	case *core.TuplePattern:
		elems, err := v.patterns(p.Elements)
		if err != nil {
			return nil, err
		}
		return exhaust.Tuple{Elems: elems}, nil
	case *core.ListPattern:
		elems, err := v.patterns(p.Elements)
		if err != nil {
			return nil, err
		}
		l := exhaust.List{Elems: elems}
		if p.Tail != nil {
			if l.Rest, err = v.pattern(*p.Tail); err != nil {
				return nil, err
			}
		}
		return l, nil
	case *core.RecordPattern:
		fields := make(map[string]exhaust.Pattern, len(p.Fields))
		for f, fp := range p.Fields {
			q, err := v.pattern(fp)
			if err != nil {
				return nil, err
			}
			fields[f] = q
		}
		return exhaust.Record{Fields: fields}, nil
	}
	return nil, fmt.Errorf("%w: unknown pattern form %T", errUndecidable, p)
}

func (v *matchVerifier) patterns(ps []core.CorePattern) ([]exhaust.Pattern, error) {
	out := make([]exhaust.Pattern, len(ps))
	for i, p := range ps {
		q, err := v.pattern(p)
		if err != nil {
			return nil, err
		}
		out[i] = q
	}
	return out, nil
}

// ctorUniverse returns the NominalKey of the ADT a constructor pattern
// belongs to and its constructors.
//
//   - A pattern the elaborator stamped belongs to the stamped ADT: the type
//     checker unified the value matched with it.
//   - An unstamped pattern (a constructor matched by name, #323) constrains
//     nothing, so its ADT is the head of the type the checker matched it
//     against — and only that: a by-name constructor of ANOTHER type, or one
//     matched against a type nothing fixes (a type variable), would let the
//     check call a match exhaustive that a value of the real type falls
//     through. Both fail closed.
//   - Only a pattern the type checker never saw falls back to the scope
//     registry and then to the one loaded ADT declaring the name.
func (v *matchVerifier) ctorUniverse(p *core.ConstructorPattern) (string, []exhaust.CtorSig, error) {
	var cands []string
	switch t, checked := v.tc.ctorPatternTypes[p]; {
	case p.TypeName != "":
		cands = append(cands, NominalKey(p.TypeModule, p.TypeName))
	case checked:
		head := v.head(t)
		if head == nil {
			return "", nil, fmt.Errorf("%w: constructor %s is matched by name and nothing fixes the type of the value it is matched against (%s); import the constructor's type, or annotate that value", errUndecidable, p.Name, t)
		}
		cands = append(cands, head.KeyOf())
	default:
		if ref, ok := v.tc.patternADT(p); ok {
			cands = append(cands, NominalKey(ref.Module, ref.Name))
		}
		if keys := v.byName[p.Name]; len(keys) == 1 {
			cands = append(cands, keys[0])
		}
	}
	for _, key := range cands {
		sigs := v.tc.matchUniverse[key]
		for _, s := range sigs {
			if s.Name == p.Name {
				return key, sigs, nil
			}
		}
	}
	if len(cands) == 0 {
		return "", nil, fmt.Errorf("%w: the type of constructor %s is unknown", errUndecidable, p.Name)
	}
	_, name := SplitNominalKey(cands[0])
	return "", nil, fmt.Errorf("%w: constructor %s is not a known constructor of %s", errUndecidable, p.Name, name)
}

// crossCheckRoot: the scrutinee's own type and its patterns' constructors
// must name the same ADT; a disagreement is a checker bug, never skipped.
func (v *matchVerifier) crossCheckRoot(m *core.Match, arms []exhaust.Arm) error {
	t, ok := v.tc.CoreTI.Get(m.Scrutinee.ID())
	if !ok {
		return nil
	}
	head := v.head(t)
	if head == nil {
		return nil
	}
	if _, known := v.tc.matchUniverse[head.KeyOf()]; !known {
		return nil
	}
	for _, a := range arms {
		if c, ok := a.Pattern.(exhaust.Ctor); ok && c.TypeName != head.KeyOf() {
			_, name := SplitNominalKey(c.TypeName)
			return fmt.Errorf("%w: constructor %s belongs to %s (%s), but the value matched has type %s (%s); this is a compiler bug, please report it",
				errUndecidable, c.Name, name, c.TypeName, head.Name, head.KeyOf())
		}
	}
	return nil
}

// head returns the nominal head of t after alias expansion, or nil.
func (v *matchVerifier) head(t Type) *TCon {
	if v.unif == nil {
		v.unif = v.tc.aliasUnifier()
	}
	t = v.unif.expandAliasMode(t, false)
	if app, ok := t.(*TApp); ok {
		t, _ = decomposeApp(app)
	}
	if h, ok := t.(*TCon); ok {
		return h
	}
	return nil
}

// aliasUnifier is a unifier that knows this module's aliases, for expansion.
func (tc *CoreTypeChecker) aliasUnifier() *Unifier {
	u := NewUnifierWithAliasesAndParams(tc.aliasEnv, tc.aliasParams)
	u.SetAliasCaptures(tc.aliasCaptures)
	u.SetNewtypes(tc.newtypes)
	return u
}

// scrutineeName names the scrutinee's type for a message, "" when unknown.
func (v *matchVerifier) scrutineeName(m *core.Match) string {
	t, ok := v.tc.CoreTI.Get(m.Scrutinee.ID())
	if !ok {
		return ""
	}
	switch t.(type) {
	case *TVar, *TVar2:
		return ""
	}
	return t.String()
}

func hasCatchAll(m *core.Match) bool {
	for _, a := range m.Arms {
		if a.Guard == nil && isCatchAll(a.Pattern) {
			return true
		}
	}
	return false
}

func isCatchAll(p core.CorePattern) bool {
	switch p.(type) {
	case *core.WildcardPattern, *core.VarPattern:
		return true
	}
	return false
}

// matchPosition is the match's surface position when the elaborator recorded
// one, else its Core span — the position RT002 reports for the same match.
func matchPosition(m *core.Match) ast.Pos {
	if p := m.OriginalSpan(); p.Line > 0 {
		return p
	}
	return m.Span()
}

func posLess(a, b ast.Pos) bool {
	if a.File != b.File {
		return a.File < b.File
	}
	if a.Line != b.Line {
		return a.Line < b.Line
	}
	return a.Column < b.Column
}
