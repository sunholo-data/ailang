package types

import (
	"fmt"
	"sort"
	"strings"

	"github.com/sunholo-data/ailang/internal/core"
)

// TC_NEWTYPE_001 codes a newtype-record ADT (`type W = W({x: int})`, the
// M-STREAM-DX/M4 shape) meeting a record type (#2239). W is nominal: it is
// not its record, so neither converts to the other implicitly. Field access
// `w.x` is a projection (fieldAccess below), not a conversion.
const TC_NEWTYPE_001 = "TC_NEWTYPE_001"

// fieldAccess is a DEFERRED field access `e.f` (#2239). Like a constructor
// choice (ambiguous_ctor.go) it sits in the constraint list and is decided by
// the substitution of all the equalities collected so far:
//
//   - the receiver is a newtype-record ADT W: the field is W's record field
//     (a projection). The receiver keeps type W; W never becomes a record.
//   - the receiver is anything else that is known (a record, a record alias,
//     an ADT, ...): `recv ~ {f: α | r}`, exactly the equality inferRecordAccess
//     used to emit eagerly, with the same diagnostics (tagged unions, missing
//     fields).
//   - the receiver is still a type variable: undecided. A let that
//     generalises over the receiver carries the access in its scheme, and
//     every use instantiates a fresh one (as a class constraint is), so
//     `let getN = \r. r.name` stays polymorphic over records and newtypes
//     alike. At the top-level declaration an undecided access is settled as
//     a record — the old eager reading, now only for a receiver nothing else
//     constrains.
//
// The eager equality bound an unknown receiver to a record at the access, so
// a value used as both a W and a record (`\r. { r.x; getW(r) }`, or
// `map(\t. t.name, ws)`) only type-checked because the unifier expanded W into
// its record whenever the two met — which let records and other newtypes pass
// as W. Deciding the access from the receiver's eventual type removes the need
// for that expansion. The decision is monotone (a head once known never
// changes; constraints are only appended), so it does not depend on the order
// constraints were visited, and every solve re-derives it identically.
type fieldAccess struct {
	recv      Type
	field     string
	fieldType Type
	row       Type // the record row of `{f: α | r}`
	path      []string
	// defaulted is set when the access was still undecided where the
	// declaration was settled: from then on it reads as a record access.
	defaulted bool
}

// constraint marks fieldAccess as a TypeConstraint; it has no behaviour.
func (c *fieldAccess) constraint() { /* marker method of TypeConstraint */ }

func (c *fieldAccess) String() string {
	return fmt.Sprintf("%s has field %s : %s", c.recv, c.field, c.fieldType)
}

// equation returns the equality the access stands for under sub, or false
// while the receiver is still unknown.
func (ctx *InferenceContext) fieldAccessEquation(fa *fieldAccess, sub Substitution) (TypeEq, bool) {
	expected := &TRecordOpen{Fields: map[string]Type{fa.field: fa.fieldType}, Row: fa.row}
	if !fa.defaulted {
		head := ApplySubstitution(sub, fa.recv)
		if ctx.unifier != nil {
			head = ctx.unifier.expandAlias(head)
		}
		switch h := head.(type) {
		case *TVar2, *TVar:
			return TypeEq{}, false
		case *TCon:
			if body, ok := ctx.unifier.newtypeBody(h); ok {
				path := append(append([]string(nil), fa.path...),
					fmt.Sprintf("field `%s` of %s, a newtype over %s", fa.field, describeTCon(h), body))
				return TypeEq{Left: body, Right: expected, Path: path}, true
			}
		}
	}
	return TypeEq{Left: fa.recv, Right: expected, Path: fa.path}, true
}

// solveFieldAccess folds fa into sub when its receiver is decided.
func (ctx *InferenceContext) solveFieldAccess(fa *fieldAccess, sub Substitution) (Substitution, bool, error) {
	eq, decided := ctx.fieldAccessEquation(fa, sub)
	if !decided {
		return sub, false, nil
	}
	next, err := ctx.unifier.Unify(ApplySubstitution(sub, eq.Left), ApplySubstitution(sub, eq.Right), sub)
	if err != nil {
		return nil, true, ctx.unifyFailure(eq, sub, err)
	}
	return next, true, nil
}

// retryFieldAccesses folds every access of pending whose receiver sub now
// decides, until no more progress is made, and returns the ones still open.
func (ctx *InferenceContext) retryFieldAccesses(sub Substitution, pending []*fieldAccess) (Substitution, []*fieldAccess, error) {
	for progress := true; progress && len(pending) > 0; {
		progress = false
		var open []*fieldAccess
		for _, fa := range pending {
			next, decided, err := ctx.solveFieldAccess(fa, sub)
			if err != nil {
				return nil, nil, err
			}
			if !decided {
				open = append(open, fa)
				continue
			}
			sub, progress = next, true
		}
		pending = open
	}
	return sub, pending, nil
}

// solveDeferred decides the constructor choices and field accesses the
// equalities leave open, each possibly deciding the other, to a fixpoint.
// When ctx.settleFieldAccesses is set, accesses still undecided at the
// fixpoint are settled as record accesses (permanently) and solving
// continues. The accesses left open are recorded in ctx.pendingAccesses for
// let-generalisation (withFieldAccesses).
func (ctx *InferenceContext) solveDeferred(sub Substitution, pending []*fieldAccess) (Substitution, error) {
	for {
		var err error
		if sub, err = ctx.resolveCtorChoices(sub); err != nil {
			return nil, err
		}
		before := len(pending)
		if sub, pending, err = ctx.retryFieldAccesses(sub, pending); err != nil {
			return nil, err
		}
		if len(pending) < before {
			continue // a decided access may decide a constructor choice
		}
		if len(pending) == 0 || !ctx.settleFieldAccesses {
			break
		}
		for _, fa := range pending {
			fa.defaulted = true
		}
	}
	ctx.pendingAccesses = pending
	return sub, nil
}

// generalizeCarrying is generalizeWithConstraints for a let (or letrec)
// binding that leaves field accesses undecided. The accesses that share a
// variable with the binding's type — closed over: an access sharing one with
// an access already taken joins too (`r.a.b`: the receiver of `.b` is the field
// type of `.a`) — are generalised TOGETHER with the type, so a variable that
// occurs only inside one (the field type of `p.x` in `\p. show(p.x)`) is
// quantified under the same side conditions as the type's own variables, and
// the class constraints on it (`Show`, `Eq`, `Num`) travel with the scheme
// (schemeConstraintsFor). The scheme carries those accesses; every use
// instantiates them afresh (Scheme.instantiate) and decides them. The originals
// stay in the constraint list over the scheme's own variables, so the
// binding's body is typed as before (settled as a record at the declaration
// if nothing decides it). Review of #2284, item 1: quantifying only the type's
// variables left such a field type shared by every use.
func (tc *CoreTypeChecker) generalizeCarrying(ctx *InferenceContext, defaulting Substitution, typ Type, effects *Row, constraints []ClassConstraint, sub Substitution, currentEnv *TypeEnv, baseEnvFreeVars map[string]bool, baseEnv *TypeEnv, literalWithhold map[string]bool) *Scheme {
	carried := ctx.accessesSharingVars(typ, sub, defaulting)
	if len(carried) == 0 {
		return tc.generalizeWithConstraints(typ, effects, constraints, sub, currentEnv, baseEnvFreeVars, baseEnv, literalWithhold)
	}
	parts := []Type{typ}
	for _, fa := range carried {
		parts = append(parts, fa.recv, fa.expected())
	}
	scheme := tc.generalizeWithConstraints(&TTuple{Elements: parts}, effects, constraints, sub, currentEnv, baseEnvFreeVars, baseEnv, literalWithhold)
	scheme.Type = typ
	scheme.fieldAccesses = carried
	return scheme
}

// expected is the record an access requires of a record receiver.
func (fa *fieldAccess) expected() *TRecordOpen {
	return &TRecordOpen{Fields: map[string]Type{fa.field: fa.fieldType}, Row: fa.row}
}

// accessesSharingVars returns the undecided accesses (under sub) connected to
// typ through shared variables, as copies over the substituted types. The
// access row is left out of the connection: on this branch every access shares
// the literal row name "r", which would connect unrelated accesses.
func (ctx *InferenceContext) accessesSharingVars(typ Type, sub, defaulting Substitution) []*fieldAccess {
	if len(ctx.pendingAccesses) == 0 {
		return nil
	}
	vars := allTypeVars(typ)
	cands := make([]*fieldAccess, len(ctx.pendingAccesses))
	candVars := make([]map[string]bool, len(cands))
	for i, fa := range ctx.pendingAccesses {
		cands[i] = &fieldAccess{recv: subThen(sub, defaulting, fa.recv), field: fa.field,
			fieldType: subThen(sub, defaulting, fa.fieldType), row: fa.row, path: fa.path}
		candVars[i] = allTypeVars(&TTuple{Elements: []Type{cands[i].recv, cands[i].fieldType}})
	}
	taken := make([]bool, len(cands))
	for progress := true; progress; {
		progress = false
		for i := range cands {
			if taken[i] || !overlaps(candVars[i], vars) {
				continue
			}
			taken[i], progress = true, true
			for v := range candVars[i] {
				vars[v] = true
			}
		}
	}
	var out []*fieldAccess
	for i, fa := range cands {
		if taken[i] {
			out = append(out, fa)
		}
	}
	return out
}

// subThen applies sub and then the let's defaulting substitution (which binds
// an ambiguous Num variable, e.g. the field type of `\r. show(r.n * 2)`, to
// int): an access must see the binding exactly as its type does, or a variable
// defaulted in the body would be quantified in the access and each use would
// pick a different numeric dictionary at run time (review of #2284, r10–r13).
func subThen(sub, defaulting Substitution, t Type) Type {
	t = ApplySubstitution(sub, t)
	if len(defaulting) > 0 {
		t = ApplySubstitution(defaulting, t)
	}
	return t
}

// allTypeVars is every variable of t (type, record-row and effect-row) by the
// shared free-variable walk (free_vars.go), which descends into open records,
// record rows and effect rows.
func allTypeVars(t Type) map[string]bool {
	out := make(map[string]bool)
	collectVars(t, out, func(varOccurrence) bool { return true }, nil)
	return out
}

// instantiateFieldAccesses returns the scheme's field accesses under subs.
func (s *Scheme) instantiateFieldAccesses(subs map[string]Type, use []string) []*fieldAccess {
	if len(s.fieldAccesses) == 0 {
		return nil
	}
	out := make([]*fieldAccess, len(s.fieldAccesses))
	for i, fa := range s.fieldAccesses {
		row := fa.row
		if row != nil {
			row = row.Substitute(subs)
		}
		out[i] = &fieldAccess{
			recv: fa.recv.Substitute(subs), field: fa.field, fieldType: fa.fieldType.Substitute(subs),
			row: row, path: append(append([]string(nil), fa.path...), use...),
		}
	}
	return out
}

// solveConstraintsAt is SolveConstraints at the generalisation of binder. When
// binder is the declaration being checked, the solve settles every undecided
// access: its scheme is exported as is, so none may stay undecided there.
func (ctx *InferenceContext) solveConstraintsAt(binder core.CoreExpr) (Substitution, []ClassConstraint, error) {
	if ctx.declRoot == nil || binder != ctx.declRoot {
		return ctx.SolveConstraints()
	}
	prev := ctx.settleFieldAccesses
	ctx.settleFieldAccesses = true
	defer func() { ctx.settleFieldAccesses = prev }()
	return ctx.SolveConstraints()
}

// newtypeBody returns the record a newtype-record ADT wraps.
func (u *Unifier) newtypeBody(con *TCon) (Type, bool) {
	if u == nil || con == nil {
		return nil, false
	}
	key := con.KeyOf()
	if !u.newtypes[key] {
		return nil, false
	}
	body, ok := u.aliasEnv[key]
	return body, ok && body != nil
}

// newtypeRecordMismatch is TC_NEWTYPE_001 when one side is a newtype-record
// ADT and the other a record type, else nil.
func (u *Unifier) newtypeRecordMismatch(t1, t2 Type) error {
	con, rec := t1, t2
	if _, ok := con.(*TCon); !ok {
		con, rec = t2, t1
	}
	c, ok := con.(*TCon)
	if !ok || !isRecordish(rec) {
		return nil
	}
	body, ok := u.newtypeBody(c)
	if !ok {
		return nil
	}
	ctors := collectAndSortCtors(c.Name, u.constructorTypes)
	ctor := c.Name
	if len(ctors) == 1 {
		ctor = ctors[0]
	}
	shown, open := diagRecord(rec)
	if open {
		// The record side is a row-polymorphic record: typically the parameter
		// of a top-level function whose type was inferred from `.f` reads
		// (`pure func getx(r) = r.x`). An undecided access is read as a record
		// access at its declaration (it must not reach an exported scheme), so
		// such a function takes records, not a W (review of #2284, item 2).
		return fmt.Errorf("%s: cannot unify %s with the record type %s: a function whose parameter type was inferred from field reads takes a record; "+
			"%s is a type with one constructor wrapping the record %s, not a record.\n"+
			"  Suggestion: annotate that parameter as %s (field access works on a %s), or pass the record inside: match v { %s(r) => … r … }.",
			TC_NEWTYPE_001, describeTCon(c), shown, c.Name, diagRecordString(body), c.Name, c.Name, ctor)
	}
	return fmt.Errorf("%s: cannot unify %s with the record type %s: %s is a type with one constructor wrapping the record %s, "+
		"not that record, and the two do not convert implicitly.\n"+
		"  Suggestion: build a %s with its constructor, %s({...}), or take the record out with match v { %s(r) => r }; "+
		"field access (v.%s) works on a %s directly.",
		TC_NEWTYPE_001, describeTCon(c), shown, c.Name, diagRecordString(body), c.Name, ctor, ctor, firstField(body), c.Name)
}

// diagRecord renders a record type for a diagnostic as a user writes one:
// `{ x: int, y: _ }` — fields sorted, an unresolved field type as `_`, a
// trailing `...` only when the record is open — instead of the internal form
// (`{ x: α4, ...{x: α4} }`, review of #2284 item 5). open reports a row that
// can still take fields.
func diagRecord(t Type) (string, bool) {
	var fields map[string]Type
	open := false
	switch r := t.(type) {
	case *TRecord:
		fields, open = r.Fields, recordRowIsOpen(r.Row)
	case *TRecordOpen:
		fields = r.Fields
		open = r.Row != nil && recordRowIsOpen(r.Row)
	case *TRecord2:
		return diagRecord(TRecord2ToTRecord(r))
	default:
		return fmt.Sprint(t), false
	}
	names := make([]string, 0, len(fields))
	for n := range fields {
		names = append(names, n)
	}
	sort.Strings(names)
	parts := make([]string, 0, len(names)+1)
	for _, n := range names {
		ft := "_"
		switch fields[n].(type) {
		case *TVar2, *TVar:
		default:
			ft = fields[n].String()
		}
		parts = append(parts, n+": "+ft)
	}
	if open {
		parts = append(parts, "...")
	}
	return "{ " + strings.Join(parts, ", ") + " }", open
}

func diagRecordString(t Type) string {
	s, _ := diagRecord(t)
	return s
}

// firstField names a field of a record for a diagnostic example ("field" if
// it has none).
func firstField(t Type) string {
	var names []string
	switch r := t.(type) {
	case *TRecord:
		for n := range r.Fields {
			names = append(names, n)
		}
	case *TRecordOpen:
		for n := range r.Fields {
			names = append(names, n)
		}
	}
	if len(names) == 0 {
		return "field"
	}
	sort.Strings(names)
	return names[0]
}
