package types

import (
	"errors"
	"fmt"
	"sort"
)

// M-EQ-DERIVE-CONTAINERS: Eq composes through containers.
//
// Eq[T] is synthesized when T is a list, Option, Result or tuple whose parts all
// resolve Eq, or a record whose nominal alias was declared `deriving (Eq)`.
// Anonymous records get nothing, so there is no action at a distance: a record
// has == only because its declaration asked for it.
//
// Every synthesized instance means STRUCTURAL comparison at runtime. That is
// exactly the semantics of composing the parts' dictionaries, because every Eq
// instance that exists is either a primitive or a derived structural one:
// user-written instances cannot be declared yet (parseInstanceDeclaration is a
// stub). When they land, a custom element Eq could differ from structural
// comparison and containers will need child-dictionary threading.

// eqSynthDepthCap bounds recursive synthesis so `ailang check` stays bounded (A5).
// Cycle-safety: synthesizeEq recurses only through eqParts and stops at this cap,
// so it terminates even on a cyclic type.
const eqSynthDepthCap = 8

// StructuralEqTypeName is the dictionary type name every synthesized Eq instance
// resolves to. The registry maps it to the DerivedStructuralEquality marker.
const StructuralEqTypeName = "StructuralEq"

// EqSynthDepthError is returned when Eq synthesis nests deeper than the cap.
// It is deliberately NOT a MissingInstanceError: the type does have ==, the
// checker just refuses to look that deep.
type EqSynthDepthError struct {
	Type Type
}

func (e *EqSynthDepthError) Error() string {
	return fmt.Sprintf("E_EQ_SYNTH_DEPTH: Eq synthesis depth cap (%d) exceeded while resolving Eq[%s]; "+
		"the value nests containers deeper than the cap supports. Fix: compare an inner part directly, "+
		"or wrap an inner level in a type declared with `deriving (Eq)`", eqSynthDepthCap, e.Type)
}

// eqParts returns the component types whose Eq instances decide Eq[typ], and
// whether typ is a shape Eq can be synthesized for.
func eqParts(typ Type) ([]Type, bool) {
	if elem, ok := AsList(typ); ok {
		return []Type{elem}, true
	}
	switch t := typ.(type) {
	case *TTuple:
		return t.Elements, true
	case *TApp:
		head, args := decomposeApp(t)
		if con, ok := head.(*TCon); ok {
			switch {
			case con.Name == "Option" && len(args) == 1:
				return args, true
			case con.Name == "Result" && len(args) == 2:
				return args, true
			}
		}
	}
	return nil, false
}

// synthesizeEq tries to build Eq[typ] from its parts. handled is false when typ
// is not a synthesizable shape, so the caller reports the ordinary missing instance.
func (env *InstanceEnv) synthesizeEq(typ Type, depth int) (inst *ClassInstance, handled bool, err error) {
	// Nominal record: the value of a record alias (or a single-constructor ADT
	// wrapping a record) carries the declared name. Its fields were checked when
	// the declaration derived Eq, so only the declaration is consulted here.
	if rec, ok := typ.(*TRecord); ok {
		if rec.TypeName == "" {
			return nil, false, nil
		}
		if _, found := env.instances[canonicalKey("Eq", &TCon{Name: rec.TypeName})]; !found {
			return nil, false, nil
		}
		return structuralEqInstance(typ), true, nil
	}

	parts, ok := eqParts(typ)
	if !ok {
		return nil, false, nil
	}
	if depth >= eqSynthDepthCap {
		return nil, true, &EqSynthDepthError{Type: typ}
	}
	for _, part := range parts {
		if _, err := env.lookup("Eq", part, depth+1); err != nil {
			return nil, true, wrapPartError(typ, err)
		}
	}
	return structuralEqInstance(typ), true, nil
}

// wrapPartError re-reports a part's Eq failure against the container typ.
// A missing instance names the innermost part that lacks Eq (not an
// intermediate container); a depth error names the outermost type, since
// that is what the user wrote.
func wrapPartError(typ Type, err error) error {
	var missing *MissingInstanceError
	if errors.As(err, &missing) {
		leaf, leafHint := missing.Type, missing.Hint
		if missing.leaf != nil {
			leaf, leafHint = missing.leaf, missing.leafHint
		}
		return &MissingInstanceError{
			Class:    "Eq",
			Type:     typ,
			Hint:     fmt.Sprintf("Equality on %s needs == on %s, which has none. %s", typ, leaf, leafHint),
			leaf:     leaf,
			leafHint: leafHint,
		}
	}
	var deep *EqSynthDepthError
	if errors.As(err, &deep) {
		return &EqSynthDepthError{Type: typ}
	}
	return err
}

func structuralEqInstance(typ Type) *ClassInstance {
	return &ClassInstance{
		ClassName: "Eq",
		TypeHead:  typ,
		Dict: Dict{
			"eq":  "derived_structural_eq",
			"neq": "derived_structural_neq",
		},
		Structural: true,
	}
}

// CheckDerivedEqField reports whether a field of a `deriving (Eq)` declaration
// has Eq. A derived type has Eq iff every field does (M-EQ-DERIVE-CONTAINERS
// R-D5). The declaration opts in every record shape its fields reach, so a
// record reached through a field is checked by its own fields instead of by an
// Eq instance of its own. That covers an anonymous record written inline, like
// the field of `type Boxed = Boxed({a: int})`, and a record alias that is not
// itself `deriving (Eq)`, like `DF` in `type D = D([DF]) deriving (Eq)`: before
// R-D5 no field was checked and both compiled and compared structurally, so
// demanding Eq of the alias broke working code (motoko_ext_abi, 2026-09-26).
// The record still has no == of its own at use sites; only the declaration
// that reaches it compares it.
//
// aliases resolves type names to their definitions (local and imported). A nil
// map disables alias expansion, leaving only the Eq-instance lookup.
// Cycle-safety: aliases already being expanded are assumed Eq (a recursive
// record is Eq iff its other fields are), containers are depth-capped, and
// inline record fields are finite syntax trees.
func (env *InstanceEnv) CheckDerivedEqField(t Type, aliases map[string]Type) error {
	return env.checkDerivedEqField(t, aliases, map[string]bool{}, 0)
}

func (env *InstanceEnv) checkDerivedEqField(t Type, aliases map[string]Type, expanding map[string]bool, depth int) error {
	if depth > eqSynthDepthCap {
		return &EqSynthDepthError{Type: t}
	}
	if _, err := env.Lookup("Eq", t); err == nil {
		return nil
	} else if !isRecordReach(t, aliases) {
		return err
	}
	switch tt := t.(type) {
	case *TRecord:
		if tt.TypeName != "" {
			if expanding[tt.TypeName] {
				return nil
			}
			expanding[tt.TypeName] = true
			defer delete(expanding, tt.TypeName)
		}
		names := make([]string, 0, len(tt.Fields))
		for name := range tt.Fields {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			if err := env.checkDerivedEqField(tt.Fields[name], aliases, expanding, depth); err != nil {
				return err
			}
		}
		return nil
	case *TCon:
		// Keyed apart from the record's TypeName: the alias target usually
		// carries the same name, and sharing the key would skip its fields.
		key := "alias:" + tt.Name
		if expanding[key] {
			return nil
		}
		expanding[key] = true
		defer delete(expanding, key)
		return env.checkDerivedEqField(aliases[tt.Name], aliases, expanding, depth)
	}
	parts, _ := eqParts(t)
	for _, part := range parts {
		if err := env.checkDerivedEqField(part, aliases, expanding, depth+1); err != nil {
			return wrapPartError(t, err)
		}
	}
	return nil
}

// isRecordReach reports whether t is, or contains through list/Option/Result/
// tuple parts, a record shape (inline, or an alias naming one) whose own fields
// a derived declaration may check. Anything else keeps the plain Eq lookup and
// its error.
func isRecordReach(t Type, aliases map[string]Type) bool {
	switch tt := t.(type) {
	case *TRecord:
		return true
	case *TCon:
		_, isRec := aliases[tt.Name].(*TRecord)
		return isRec
	}
	parts, ok := eqParts(t)
	if !ok {
		return false
	}
	for _, p := range parts {
		if isRecordReach(p, aliases) {
			return true
		}
	}
	return false
}

// reduceEq decomposes a NON-ground Eq[typ] by the same rules as synthesizeEq
// (M-EQ-DERIVE-CONTAINERS R-D7). Ground parts must resolve now; bare type
// variables are returned as residuals and stay polymorphic (generalized, or
// phantom as in `None == None`). Anything else fails loudly.
//
// Before R-D7 a non-ground Eq constraint was never checked: Ok(1) == Ok(1)
// compiled only because the Err type stayed a variable, and Err(f) == Err(f)
// handed two closures to the runtime comparator, which answered false.
// Cycle-safety: depth-capped exactly like synthesizeEq.
func (env *InstanceEnv) reduceEq(typ Type, depth int) ([]Type, error) {
	switch typ.(type) {
	case *TVar, *TVar2:
		return []Type{typ}, nil
	}
	if isGroundDeep(typ) {
		_, err := env.lookup("Eq", typ, depth)
		return nil, err
	}
	if rec, ok := typ.(*TRecord); ok && rec.TypeName != "" {
		if _, found := env.instances[canonicalKey("Eq", &TCon{Name: rec.TypeName})]; found {
			return nil, nil
		}
	}
	parts, ok := eqParts(typ)
	if !ok {
		return nil, &MissingInstanceError{Class: "Eq", Type: typ, Hint: actionableInstanceHint("Eq", typ)}
	}
	if depth >= eqSynthDepthCap {
		return nil, &EqSynthDepthError{Type: typ}
	}
	var residual []Type
	for _, part := range parts {
		r, err := env.reduceEq(part, depth+1)
		if err != nil {
			return nil, wrapPartError(typ, err)
		}
		residual = append(residual, r...)
	}
	return residual, nil
}

// ReduceEqConstraints checks every non-ground Eq constraint that is not a bare
// type variable, replacing it with the Eq constraints on its residual variables.
// Other classes pass through unchanged.
func (env *InstanceEnv) ReduceEqConstraints(constraints []ClassConstraint) ([]ClassConstraint, error) {
	out := make([]ClassConstraint, 0, len(constraints))
	for _, c := range constraints {
		if c.Class != "Eq" {
			out = append(out, c)
			continue
		}
		residual, err := env.reduceEq(c.Type, 0)
		if err != nil {
			if len(c.Path) > 0 {
				return nil, fmt.Errorf("at %s: %w", c.Path[0], err)
			}
			return nil, err
		}
		for _, r := range residual {
			rc := c
			rc.Type = r
			out = append(out, rc)
		}
	}
	return out, nil
}
