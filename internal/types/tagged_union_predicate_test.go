package types

import "testing"

// TestIsTaggedUnion exercises the predicate that gates
// inferRecordAccess against silent auto-unwrap of multi-constructor
// ADT values during record-field access. See
// M-TYPECHECK-NO-AUTO-UNWRAP-RESULT design doc.

// resultRegistry mirrors the constructorTypes the elaborator would
// build for `Result[T, E] = Ok(T) | Err(E)`.
var resultRegistry = map[string]string{
	"Ok":  "Result",
	"Err": "Result",
}

// optionRegistry mirrors the constructorTypes for `Option[T] = Some(T) | None`.
var optionRegistry = map[string]string{
	"Some": "Option",
	"None": "Option",
}

// singleCtorRegistry: a single-constructor ADT (commonly used as struct
// emulation, e.g. `type Wrap = Wrap({x:int})`). Must NOT be flagged.
var singleCtorRegistry = map[string]string{
	"Wrap": "Wrap",
}

// multiCtorRegistry: a 3-variant user ADT.
var multiCtorRegistry = map[string]string{
	"Up":    "Direction",
	"Down":  "Direction",
	"Left":  "Direction",
	"Right": "Direction",
}

func TestIsTaggedUnion_RejectsResult(t *testing.T) {
	// Result[Int, String] must be flagged.
	resultIntString := &TApp{
		Constructor: &TCon{Name: "Result"},
		Args:        []Type{&TCon{Name: "int"}, &TCon{Name: "string"}},
	}
	if !isTaggedUnion(resultIntString, resultRegistry) {
		t.Errorf("isTaggedUnion(Result[Int, String]) = false; want true")
	}
}

func TestIsTaggedUnion_RejectsOption(t *testing.T) {
	// Option[Record{a:int}] must be flagged.
	optionRecord := &TApp{
		Constructor: &TCon{Name: "Option"},
		Args:        []Type{&TCon{Name: "int"}},
	}
	if !isTaggedUnion(optionRecord, optionRegistry) {
		t.Errorf("isTaggedUnion(Option[Int]) = false; want true")
	}
}

func TestIsTaggedUnion_RejectsMultiVariantUserADT(t *testing.T) {
	dir := &TCon{Name: "Direction"}
	if !isTaggedUnion(dir, multiCtorRegistry) {
		t.Errorf("isTaggedUnion(Direction) = false; want true")
	}
}

func TestIsTaggedUnion_AllowsSingleConstructorADT(t *testing.T) {
	// `type Wrap = Wrap({x:int})` — single-ctor ADTs are commonly used as
	// struct emulation. They MUST continue to permit .field access.
	wrap := &TCon{Name: "Wrap"}
	if isTaggedUnion(wrap, singleCtorRegistry) {
		t.Errorf("isTaggedUnion(Wrap) = true; want false (single-ctor ADTs are not tagged unions)")
	}
}

func TestIsTaggedUnion_AllowsPlainRecord(t *testing.T) {
	rec := &TRecord{
		Row: &Row{Labels: map[string]Type{"a": &TCon{Name: "int"}}},
	}
	if isTaggedUnion(rec, resultRegistry) {
		t.Errorf("isTaggedUnion(Record{a:int}) = true; want false")
	}
}

func TestIsTaggedUnion_AllowsPrimitives(t *testing.T) {
	for _, name := range []string{"int", "float", "string", "bool", "unit"} {
		prim := &TCon{Name: name}
		if isTaggedUnion(prim, resultRegistry) {
			t.Errorf("isTaggedUnion(%s) = true; want false (primitive)", name)
		}
	}
}

func TestIsTaggedUnion_AllowsTypeVar(t *testing.T) {
	// TVar should not be flagged — by definition we don't yet know what it'll
	// resolve to; the gate fires AFTER substitution.
	tvar := &TVar{Name: "alpha"}
	if isTaggedUnion(tvar, resultRegistry) {
		t.Errorf("isTaggedUnion(TVar alpha) = true; want false")
	}
	tvar2 := &TVar2{Name: "beta", Kind: KStar{}}
	if isTaggedUnion(tvar2, resultRegistry) {
		t.Errorf("isTaggedUnion(TVar2 beta) = true; want false")
	}
}

func TestIsTaggedUnion_AllowsList(t *testing.T) {
	// Lists are tagged-union-shaped at the runtime level (Cons / Nil) but
	// the predicate explicitly excludes them — list field access is a
	// separate (different) error class anyway.
	list := &TApp{
		Constructor: &TCon{Name: "list"},
		Args:        []Type{&TCon{Name: "int"}},
	}
	if isTaggedUnion(list, resultRegistry) {
		t.Errorf("isTaggedUnion(list[int]) = true; want false")
	}
}

func TestIsTaggedUnion_NilRegistryShortCircuits(t *testing.T) {
	// Nil registry means we have no typechecker context; gate must not
	// fire and produce false-positive errors.
	resultIntString := &TApp{
		Constructor: &TCon{Name: "Result"},
		Args:        []Type{&TCon{Name: "int"}, &TCon{Name: "string"}},
	}
	if isTaggedUnion(resultIntString, nil) {
		t.Errorf("isTaggedUnion(Result, nil) = true; want false (nil registry should short-circuit)")
	}
}

func TestIsTaggedUnion_UnregisteredNameReturnsFalse(t *testing.T) {
	// Type that's neither a primitive nor a registered ADT (e.g. a user
	// type that wasn't elaborated). Gate must not fire.
	unknown := &TCon{Name: "MysteryType"}
	if isTaggedUnion(unknown, resultRegistry) {
		t.Errorf("isTaggedUnion(MysteryType) = true; want false (unregistered name)")
	}
}
