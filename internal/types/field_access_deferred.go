package types

import (
	"fmt"
	"sort"

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

func (c *fieldAccess) constraint() {}
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
// let-generalisation (attachFieldAccesses).
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

// attachFieldAccesses records in scheme (a let's generalised type, solved
// under sub) every access still undecided whose receiver or field type
// mentions a variable the scheme quantifies: each use of the binding
// instantiates it afresh (Scheme.instantiate), and the use decides it. The
// original stays in the constraint list over the scheme's own variables, so
// the binding's body is typed as before (settled as a record at the
// declaration if nothing decides it).
func (ctx *InferenceContext) attachFieldAccesses(scheme *Scheme, sub Substitution) {
	if scheme == nil || len(ctx.pendingAccesses) == 0 || len(scheme.TypeVars)+len(scheme.RowVars) == 0 {
		return
	}
	quantified := make(map[string]bool, len(scheme.TypeVars)+len(scheme.RowVars))
	for _, v := range scheme.TypeVars {
		quantified[v] = true
	}
	for _, v := range scheme.RowVars {
		quantified[v] = true
	}
	for _, fa := range ctx.pendingAccesses {
		recv, ft := ApplySubstitution(sub, fa.recv), ApplySubstitution(sub, fa.fieldType)
		free := make(map[string]bool)
		collectFreeVars(recv, free)
		collectFreeVars(ft, free)
		if overlaps(free, quantified) {
			scheme.fieldAccesses = append(scheme.fieldAccesses, &fieldAccess{
				recv: recv, field: fa.field, fieldType: ft, row: fa.row, path: fa.path,
			})
		}
	}
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

// settleFieldAccessesAt marks binder's solve as a settling one when binder is
// the declaration being checked: its scheme is exported as is, so no access
// may stay undecided there.
func (ctx *InferenceContext) settleFieldAccessesAt(binder core.CoreExpr) func() {
	if ctx.declRoot == nil || binder != ctx.declRoot {
		return func() {}
	}
	prev := ctx.settleFieldAccesses
	ctx.settleFieldAccesses = true
	return func() { ctx.settleFieldAccesses = prev }
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
	return fmt.Errorf("%s: cannot unify %s with the record type %s: %s is a type with one constructor wrapping the record %s, "+
		"not that record, and the two do not convert implicitly.\n"+
		"  Suggestion: build a %s with its constructor, %s({...}), or take the record out with match v { %s(r) => r }; "+
		"field access (v.%s) works on a %s directly.",
		TC_NEWTYPE_001, describeTCon(c), rec, c.Name, body, c.Name, ctor, ctor, firstField(body), c.Name)
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
