package iface

import (
	"testing"

	"github.com/sunholo/ailang/internal/types"
)

func TestConstructorScheme(t *testing.T) {
	// Create a constructor scheme for Some(a) :: Option[a]
	intType := &types.TCon{Name: "Int"}
	optionType := &types.TCon{Name: "Option"}

	scheme := &ConstructorScheme{
		TypeName:   "Option",
		CtorName:   "Some",
		FieldTypes: []types.Type{intType},
		ResultType: optionType,
		Arity:      1,
	}

	if scheme.TypeName != "Option" {
		t.Errorf("TypeName = %q, want %q", scheme.TypeName, "Option")
	}
	if scheme.CtorName != "Some" {
		t.Errorf("CtorName = %q, want %q", scheme.CtorName, "Some")
	}
	if scheme.Arity != 1 {
		t.Errorf("Arity = %d, want %d", scheme.Arity, 1)
	}
	if len(scheme.FieldTypes) != 1 {
		t.Errorf("len(FieldTypes) = %d, want %d", len(scheme.FieldTypes), 1)
	}
}

func TestIfaceAddConstructor(t *testing.T) {
	iface := NewIface("test/module")

	// Add Some(a) constructor
	intType := &types.TCon{Name: "Int"}
	optionType := &types.TCon{Name: "Option"}

	iface.AddConstructor("Option", "Some", []types.Type{intType}, optionType)

	// Verify it was added
	ctor, ok := iface.GetConstructor("Some")
	if !ok {
		t.Fatal("GetConstructor(\"Some\") returned false, want true")
	}
	if ctor.CtorName != "Some" {
		t.Errorf("CtorName = %q, want %q", ctor.CtorName, "Some")
	}
	if ctor.TypeName != "Option" {
		t.Errorf("TypeName = %q, want %q", ctor.TypeName, "Option")
	}
}

func TestIfaceMultipleConstructors(t *testing.T) {
	iface := NewIface("test/module")

	// Add Option ADT constructors
	intType := &types.TCon{Name: "Int"}
	optionIntType := &types.TCon{Name: "Option"}

	// Some(a) :: Option[a]
	iface.AddConstructor("Option", "Some", []types.Type{intType}, optionIntType)

	// None :: Option[a]
	iface.AddConstructor("Option", "None", []types.Type{}, optionIntType)

	// Verify both were added
	some, ok := iface.GetConstructor("Some")
	if !ok {
		t.Fatal("GetConstructor(\"Some\") returned false")
	}
	if some.Arity != 1 {
		t.Errorf("Some.Arity = %d, want %d", some.Arity, 1)
	}

	none, ok := iface.GetConstructor("None")
	if !ok {
		t.Fatal("GetConstructor(\"None\") returned false")
	}
	if none.Arity != 0 {
		t.Errorf("None.Arity = %d, want %d", none.Arity, 0)
	}

	// Both should have same TypeName
	if some.TypeName != none.TypeName {
		t.Errorf("Some.TypeName = %q, None.TypeName = %q, want them equal",
			some.TypeName, none.TypeName)
	}
}

func TestConstructorNotFound(t *testing.T) {
	iface := NewIface("test/module")

	// Try to get a constructor that doesn't exist
	_, ok := iface.GetConstructor("NonExistent")
	if ok {
		t.Error("GetConstructor(\"NonExistent\") returned true, want false")
	}
}

func TestDigestWithConstructors(t *testing.T) {
	// Create two identical interfaces
	iface1 := NewIface("test/module")
	iface2 := NewIface("test/module")

	// Add same constructors to both
	intType := &types.TCon{Name: "Int"}
	optionIntType := &types.TCon{Name: "Option"}

	iface1.AddConstructor("Option", "None", []types.Type{}, optionIntType)
	iface1.AddConstructor("Option", "Some", []types.Type{intType}, optionIntType)

	iface2.AddConstructor("Option", "None", []types.Type{}, optionIntType)
	iface2.AddConstructor("Option", "Some", []types.Type{intType}, optionIntType)

	// Compute digests
	builder := &Builder{module: "test/module"}
	digest1, err1 := builder.computeDigest(iface1)
	digest2, err2 := builder.computeDigest(iface2)

	if err1 != nil {
		t.Fatalf("computeDigest(iface1) error: %v", err1)
	}
	if err2 != nil {
		t.Fatalf("computeDigest(iface2) error: %v", err2)
	}

	// Digests should be identical (deterministic)
	if digest1 != digest2 {
		t.Errorf("Digests differ:\niface1: %s\niface2: %s", digest1, digest2)
	}
}

func TestDigestDifferentConstructors(t *testing.T) {
	// Create two interfaces with different constructors
	iface1 := NewIface("test/module")
	iface2 := NewIface("test/module")

	intType := &types.TCon{Name: "Int"}
	optionIntType := &types.TCon{Name: "Option"}

	// iface1: Option with Some and None
	iface1.AddConstructor("Option", "Some", []types.Type{intType}, optionIntType)
	iface1.AddConstructor("Option", "None", []types.Type{}, optionIntType)

	// iface2: Option with only Some
	iface2.AddConstructor("Option", "Some", []types.Type{intType}, optionIntType)

	// Compute digests
	builder := &Builder{module: "test/module"}
	digest1, err1 := builder.computeDigest(iface1)
	digest2, err2 := builder.computeDigest(iface2)

	if err1 != nil {
		t.Fatalf("computeDigest(iface1) error: %v", err1)
	}
	if err2 != nil {
		t.Fatalf("computeDigest(iface2) error: %v", err2)
	}

	// Digests should differ
	if digest1 == digest2 {
		t.Errorf("Digests are identical but should differ:\n%s", digest1)
	}
}
