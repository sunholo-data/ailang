package types

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"

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
// against, which the check then stamps on the pattern; a pattern with
// neither is TC_MATCH_007 (#2303), never a guess. Its universe is
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
//   - a constructor pattern whose type cannot be named -> TC_MATCH_007
//     (error), a catch-all notwithstanding (#2303)
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
	TC_MATCH_007 = "TC_MATCH_007" // constructor pattern whose type cannot be named (#2303)
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
	// Sorted: every message and choice built from these lists must not
	// depend on map iteration order.
	for _, keys := range out {
		sort.Strings(keys)
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
		v.walk(e.Lo)
		v.walk(e.Hi)
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

// unresolvedCtorError (TC_MATCH_007, #2303): a constructor matched by name
// (#323) — neither its type nor the constructor is imported or declared
// here — whose type the check cannot name. Either nothing fixes the type of
// the value matched (scrutinee == ""), or that type, scrutinee, has no
// constructor of the name. cands lists the loaded ADTs that declare the
// name (NominalKeys, sorted), for the suggestion only.
type unresolvedCtorError struct {
	pattern   *core.ConstructorPattern
	cands     []string
	scrutinee string   // NominalKey of the value's type when it is fixed
	known     bool     // scrutinee has a constructor universe
	ctors     []string // scrutinee's constructors (known only)
	rigid     string   // the signature type variable the value has, as written
	fixedType string   // the value's fixed type when it has no nominal head (a record, a tuple, ...)
}

func (e *unresolvedCtorError) Error() string {
	switch {
	case e.fixedType != "":
		return fmt.Sprintf("constructor %s is matched by name against a value of type %s, which is not built from constructors here, so this arm can never match", e.pattern.Name, e.fixedType)
	case e.scrutinee != "":
		_, name := SplitNominalKey(e.scrutinee)
		if !e.known {
			return fmt.Sprintf("constructor %s is matched by name against a value of type %s, which is not built from constructors here, so this arm can never match", e.pattern.Name, name)
		}
		return fmt.Sprintf("constructor %s is matched by name against a value of type %s, which has no constructor %s, so this arm can never match", e.pattern.Name, name, e.pattern.Name)
	case e.rigid != "":
		return fmt.Sprintf("constructor %s cannot be matched against a value of type %s: %s is a type variable of the signature, so the value can have any type", e.pattern.Name, e.rigid, e.rigid)
	}
	return fmt.Sprintf("cannot tell which type %s belongs to here: it is matched by name (neither its type nor the constructor is imported here), and nothing fixes the type of the value matched", e.pattern.Name)
}

// isTypeVar reports whether t is a type variable (nothing fixes it).
func isTypeVar(t Type) bool {
	switch t.(type) {
	case *TVar, *TVar2:
		return true
	}
	return false
}

// describeType renders a fixed type for a message, with signature variables
// as written and an alias with what it expands to: "P (= {a: int})".
func (v *matchVerifier) describeType(t Type) string {
	if v.unif == nil {
		v.unif = v.tc.aliasUnifier()
	}
	s, exp := writtenType(t), writtenType(v.unif.expandAlias(t))
	if exp != s {
		return s + " (= " + exp + ")"
	}
	return s
}

// openRowOnly is a function type's effect row that is only an open tail
// (" ! {...ε3}"): inference bookkeeping, not something the author wrote.
var openRowOnly = regexp.MustCompile(` ! \{\.\.\.[^{}\s,]*\}`)

// writtenType renders t for a message as the author would write it:
// signature variables as written, a list as [T], no bare open effect rows.
// The rendering itself is displayType's, bounded (#2363).
func writtenType(t Type) string {
	switch l := t.(type) {
	case *TList:
		return "[" + writtenType(l.Element) + "]"
	case *TApp:
		if c, ok := l.Constructor.(*TCon); ok && strings.EqualFold(c.Name, "list") && len(l.Args) == 1 {
			return "[" + writtenType(l.Args[0]) + "]"
		}
	}
	return openRowOnly.ReplaceAllString(WrittenTypeVarNames(displayType(t)), "")
}

// rigidVarName is t's name as written when t is a signature type variable
// (#2273: rigid, named "a@{f}"), else "".
func rigidVarName(t Type) string {
	var name string
	switch tv := t.(type) {
	case *TVar:
		name = tv.Name
	case *TVar2:
		name = tv.Name
	}
	if !strings.Contains(name, "@{") {
		return ""
	}
	return WrittenTypeVarNames(name)
}

// check decides one match.
func (v *matchVerifier) check(m *core.Match) {
	pos := matchPosition(m)
	// Every constructor pattern names ONE type before anything else is
	// decided, a catch-all notwithstanding (#2303): the back ends test a
	// value's tag against the ordinal of the pattern's own type, and a
	// pattern no type was fixed for has no ordinal to test.
	for _, arm := range m.Arms {
		if err := v.resolvePatterns(arm.Pattern); err != nil {
			v.errs = append(v.errs, newUnresolvedCtorError(pos, err))
			return
		}
	}
	// A constructor pattern with the wrong number of arguments is an error
	// whatever else the match says, a catch-all included: the evaluator
	// never matches it and the VM binds a prefix of the fields (#2243).
	for _, arm := range m.Arms {
		var ae *patternArityError
		if err := v.arityOf(arm.Pattern); errors.As(err, &ae) {
			v.errs = append(v.errs, newPatternArityError(pos, ae))
			return
		}
	}
	arms, err := v.arms(m)
	if err == nil {
		// A disagreement between the scrutinee's type and its patterns is a
		// checker bug: reported even under a catch-all.
		if cerr := v.crossCheckRoot(m, arms); cerr != nil {
			v.errs = append(v.errs, newUndecidedError(pos, cerr))
			return
		}
	}
	var res *exhaust.Result
	if err == nil {
		res, err = exhaust.Check(arms, exhaust.StaticUniverse(v.tc.matchUniverse), exhaust.Options{MaxWitnesses: shownWitnesses + 1})
	}
	switch {
	case err != nil && hasCatchAll(m):
		m.Exhaustive = true // an unguarded `_` covers everything, whatever the rest says
		return
	case errors.Is(err, exhaust.ErrBudget):
		v.errs = append(v.errs, newBudgetError(pos, err))
		return
	case err != nil:
		v.errs = append(v.errs, newUndecidedError(pos, err))
		return
	}
	m.Exhaustive = res.Exhaustive
	if !res.Exhaustive {
		v.errs = append(v.errs, newNonExhaustiveError(pos, v.scrutineeName(m), res.Missing, hasGuard(m)))
	}
	for _, i := range res.Redundant {
		v.warns = append(v.warns, &MatchWarning{Pos: pos, Arm: i + 1, Pattern: exhaust.Render(arms[i].Pattern)})
	}
}

// arityOf finds a constructor pattern, at any depth, whose argument count
// differs from its ADT's declaration of the constructor. It runs after
// resolvePatterns, so every pattern names its ADT, and before, independently
// of, the coverage check.
func (v *matchVerifier) arityOf(p core.CorePattern) error {
	switch p := p.(type) {
	case *core.ConstructorPattern:
		if err := v.ctorArity(p); err != nil {
			return err
		}
		return v.arityOfAll(p.Args)
	case *core.TuplePattern:
		return v.arityOfAll(p.Elements)
	case *core.ListPattern:
		if err := v.arityOfAll(p.Elements); err != nil || p.Tail == nil {
			return err
		}
		return v.arityOf(*p.Tail)
	case *core.RecordPattern:
		for _, f := range sortedPatternFields(p.Fields) {
			if err := v.arityOf(p.Fields[f]); err != nil {
				return err
			}
		}
	}
	return nil
}

// ctorArity checks one constructor pattern's argument count against its
// ADT's declaration. A pattern whose ADT has no universe (one the elaborator
// stamped with a type no loaded declaration lists) has nothing to check
// against.
func (v *matchVerifier) ctorArity(p *core.ConstructorPattern) error {
	_, sigs, err := v.ctorUniverse(p)
	if err != nil {
		return nil
	}
	for _, a := range sigArities(sigs, p.Name) {
		if a != len(p.Args) {
			return &patternArityError{pattern: p, arity: a}
		}
	}
	return nil
}

func (v *matchVerifier) arityOfAll(ps []core.CorePattern) error {
	for _, p := range ps {
		if err := v.arityOf(p); err != nil {
			return err
		}
	}
	return nil
}

func sigArities(sigs []exhaust.CtorSig, name string) []int {
	var out []int
	for _, s := range sigs {
		if s.Name == name {
			out = append(out, s.Arity)
		}
	}
	return out
}

func sortedPatternFields(m map[string]core.CorePattern) []string {
	out := make([]string, 0, len(m))
	for f := range m {
		out = append(out, f)
	}
	sort.Strings(out)
	return out
}

// arms converts a match's arms; the first pattern that cannot be converted
// stops it.
func (v *matchVerifier) arms(m *core.Match) ([]exhaust.Arm, error) {
	arms := make([]exhaust.Arm, len(m.Arms))
	for i, arm := range m.Arms {
		p, err := v.pattern(arm.Pattern)
		if err != nil {
			return nil, err
		}
		arms[i] = exhaust.Arm{Pattern: p, Guarded: arm.Guard != nil}
	}
	return arms, nil
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
		key, _, err := v.ctorUniverse(p)
		if err != nil {
			return nil, err
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
		for _, f := range sortedPatternFields(p.Fields) {
			fp := p.Fields[f]
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
//     matched against a type nothing fixes (a type variable), is an
//     *unresolvedCtorError (#2303). Which loaded types happen to declare the
//     name never decides it: under separate compilation a caller can pass a
//     value of a type this module never loaded.
//   - A pattern the type checker never saw takes the ADT its constructor
//     names in this module's scope (local declarations and imports), or none.
func (v *matchVerifier) ctorUniverse(p *core.ConstructorPattern) (string, []exhaust.CtorSig, error) {
	var key string
	switch t, checked := v.tc.ctorPatternTypes[p]; {
	case p.TypeName != "":
		key = NominalKey(p.TypeModule, p.TypeName)
	case checked:
		head := v.head(t)
		if head == nil || v.tc.matchUniverse[head.KeyOf()] == nil && !isTypeVar(t) {
			e := &unresolvedCtorError{pattern: p, cands: v.byName[p.Name], rigid: rigidVarName(t)}
			if !isTypeVar(t) {
				// Fixed, and not a type built from constructors (a record,
				// tuple, list, function, a primitive, or an alias of one).
				e.fixedType = v.describeType(t)
			}
			return "", nil, e
		}
		key = head.KeyOf()
	default:
		ref, ok := v.tc.patternADT(p)
		if !ok {
			return "", nil, &unresolvedCtorError{pattern: p, cands: v.byName[p.Name]}
		}
		key = NominalKey(ref.Module, ref.Name)
	}
	sigs := v.tc.matchUniverse[key]
	for _, s := range sigs {
		if s.Name == p.Name {
			return key, sigs, nil
		}
	}
	if p.TypeName == "" {
		e := &unresolvedCtorError{pattern: p, cands: v.byName[p.Name], scrutinee: key, known: sigs != nil}
		for _, s := range sigs {
			e.ctors = append(e.ctors, s.Name)
		}
		return "", nil, e
	}
	_, name := SplitNominalKey(key)
	return "", nil, fmt.Errorf("%w: constructor %s is not a known constructor of %s", errUndecidable, p.Name, name)
}

// resolvePatterns stamps every constructor pattern in p, at any depth, with
// the ADT ctorUniverse resolves it to — for a constructor matched by name
// (#323), the nominal head of the type the checker matched it against, after
// alias expansion (#2244). The back ends dispatch on the stamp alone, so none
// of them resolves a type name, an alias above all, on its own. A pattern
// that cannot be resolved is an *unresolvedCtorError, whatever the other
// arms are: before #2303 a catch-all let such a pattern through unstamped,
// and the VM (and the since-removed Go emitter) tested it against the first
// loaded ADT that declared the name — a different ordinal for a different type.
func (v *matchVerifier) resolvePatterns(p core.CorePattern) error {
	switch p := p.(type) {
	case *core.ConstructorPattern:
		if p.TypeName == "" {
			key, _, err := v.ctorUniverse(p)
			if err != nil {
				return err
			}
			p.TypeModule, p.TypeName = SplitNominalKey(key)
		}
		return v.resolveAll(p.Args)
	case *core.TuplePattern:
		return v.resolveAll(p.Elements)
	case *core.ListPattern:
		if err := v.resolveAll(p.Elements); err != nil || p.Tail == nil {
			return err
		}
		return v.resolvePatterns(*p.Tail)
	case *core.RecordPattern:
		for _, f := range sortedPatternFields(p.Fields) {
			if err := v.resolvePatterns(p.Fields[f]); err != nil {
				return err
			}
		}
	}
	return nil
}

func (v *matchVerifier) resolveAll(ps []core.CorePattern) error {
	for _, p := range ps {
		if err := v.resolvePatterns(p); err != nil {
			return err
		}
	}
	return nil
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

// head returns the nominal head of t after alias expansion, or nil
// (nominal_head.go).
func (v *matchVerifier) head(t Type) *TCon {
	if v.unif == nil {
		v.unif = v.tc.aliasUnifier()
	}
	return v.unif.nominalHead(t)
}

// aliasUnifier is a unifier that knows this module's aliases, for expansion.
func (tc *CoreTypeChecker) aliasUnifier() *Unifier {
	u := NewUnifierWithAliasesAndParams(tc.aliasEnv, tc.aliasParams)
	u.SetAliasCaptures(tc.aliasCaptures)
	u.SetNewtypes(tc.newtypes)
	return u
}

// scrutineeName names the scrutinee's type for a message, "" when unknown.
// A type written as an alias is named with what it expands to, "Hue (=
// Color)", because the missing patterns are the expansion's constructors.
func (v *matchVerifier) scrutineeName(m *core.Match) string {
	t, ok := v.tc.CoreTI.Get(m.Scrutinee.ID())
	if !ok {
		return ""
	}
	switch t.(type) {
	case *TVar, *TVar2:
		return ""
	}
	if v.unif == nil {
		v.unif = v.tc.aliasUnifier()
	}
	name := displayType(t) // bounded: the expansion can be exponential (#2363)
	if exp := displayType(v.unif.expandAlias(t)); exp != name {
		return name + " (= " + exp + ")"
	}
	return name
}

func hasCatchAll(m *core.Match) bool {
	for _, a := range m.Arms {
		if a.Guard == nil && isCatchAll(a.Pattern) {
			return true
		}
	}
	return false
}

func hasGuard(m *core.Match) bool {
	for _, a := range m.Arms {
		if a.Guard != nil {
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
