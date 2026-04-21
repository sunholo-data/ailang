package types

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
)

func TestTypeInfo_MustReturnsTypeWhenExists(t *testing.T) {
	ti := NewTypeInfo()
	intType := &TCon{Name: "int"}
	expr := &ast.Identifier{Name: "x"}
	ti.Set(expr, intType)

	result := ti.Must(expr)
	if result != intType {
		t.Errorf("Must() returned wrong type: got %v, want %v", result, intType)
	}
}

func TestTypeInfo_MustPanicsWhenMissing(t *testing.T) {
	ti := NewTypeInfo()
	expr := &ast.Identifier{Name: "missing"}

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Must() did not panic for missing expression")
		} else {
			// Check that panic message is helpful
			msg := r.(string)
			if msg == "" {
				t.Errorf("Must() panic message is empty")
			}
			t.Logf("Panic message: %s", msg)
		}
	}()

	ti.Must(expr)
}

func TestTypeInfo_GetReturnsTypeAndTrue(t *testing.T) {
	ti := NewTypeInfo()
	strType := &TCon{Name: "string"}
	expr := &ast.Literal{Kind: ast.StringLit, Value: "hello"}
	ti.Set(expr, strType)

	result, ok := ti.Get(expr)
	if !ok {
		t.Errorf("Get() returned false for existing expression")
	}
	if result != strType {
		t.Errorf("Get() returned wrong type: got %v, want %v", result, strType)
	}
}

func TestTypeInfo_GetReturnsFalseWhenMissing(t *testing.T) {
	ti := NewTypeInfo()
	expr := &ast.Identifier{Name: "missing"}

	result, ok := ti.Get(expr)
	if ok {
		t.Errorf("Get() returned true for missing expression")
	}
	if result != nil {
		t.Errorf("Get() returned non-nil type for missing expression: %v", result)
	}
}

func TestTypeInfo_Has(t *testing.T) {
	ti := NewTypeInfo()
	expr1 := &ast.Identifier{Name: "x"}
	expr2 := &ast.Identifier{Name: "y"}
	ti.Set(expr1, &TCon{Name: "bool"})

	if !ti.Has(expr1) {
		t.Errorf("Has() returned false for existing expression")
	}
	if ti.Has(expr2) {
		t.Errorf("Has() returned true for missing expression")
	}
}

func TestTypeInfo_Set(t *testing.T) {
	ti := NewTypeInfo()
	floatType := &TCon{Name: "float"}
	expr := &ast.Literal{Kind: ast.FloatLit, Value: 3.14}

	ti.Set(expr, floatType)

	if !ti.Has(expr) {
		t.Errorf("Set() did not store type")
	}

	result := ti.Must(expr)
	if result != floatType {
		t.Errorf("Set() stored wrong type: got %v, want %v", result, floatType)
	}
}

func TestTypeInfo_NewTypeInfo(t *testing.T) {
	ti := NewTypeInfo()

	if ti == nil {
		t.Errorf("NewTypeInfo() returned nil")
	}
	if len(ti) != 0 {
		t.Errorf("NewTypeInfo() returned non-empty map: %d entries", len(ti))
	}
}

func TestTypeInfo_PointerIdentity(t *testing.T) {
	ti := NewTypeInfo()
	expr1 := &ast.Identifier{Name: "x"}
	expr2 := &ast.Identifier{Name: "x"} // Same name, different pointer

	ti.Set(expr1, &TCon{Name: "int"})

	if ti.Has(expr2) {
		t.Errorf("TypeInfo should use pointer identity, not value equality")
	}
	if !ti.Has(expr1) {
		t.Errorf("TypeInfo should find expression by pointer identity")
	}
}

// CoreTypeInfo tests

func TestCoreTypeInfo_MustReturnsTypeWhenExists(t *testing.T) {
	cti := NewCoreTypeInfo()
	intType := &TCon{Name: "int"}
	cti.Set(42, intType)

	result := cti.Must(42)
	if result != intType {
		t.Errorf("Must() returned wrong type: got %v, want %v", result, intType)
	}
}

func TestCoreTypeInfo_MustPanicsWhenMissing(t *testing.T) {
	cti := NewCoreTypeInfo()

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Must() did not panic for missing NodeID")
		} else {
			msg := r.(string)
			if msg == "" {
				t.Errorf("Must() panic message is empty")
			}
			t.Logf("Panic message: %s", msg)
		}
	}()

	cti.Must(99)
}

func TestCoreTypeInfo_GetReturnsTypeAndTrue(t *testing.T) {
	cti := NewCoreTypeInfo()
	strType := &TCon{Name: "string"}
	cti.Set(10, strType)

	result, ok := cti.Get(10)
	if !ok {
		t.Errorf("Get() returned false for existing NodeID")
	}
	if result != strType {
		t.Errorf("Get() returned wrong type: got %v, want %v", result, strType)
	}
}

func TestCoreTypeInfo_GetReturnsFalseWhenMissing(t *testing.T) {
	cti := NewCoreTypeInfo()

	result, ok := cti.Get(99)
	if ok {
		t.Errorf("Get() returned true for missing NodeID")
	}
	if result != nil {
		t.Errorf("Get() returned non-nil type for missing NodeID: %v", result)
	}
}

func TestCoreTypeInfo_Has(t *testing.T) {
	cti := NewCoreTypeInfo()
	cti.Set(5, &TCon{Name: "bool"})

	if !cti.Has(5) {
		t.Errorf("Has() returned false for existing NodeID")
	}
	if cti.Has(99) {
		t.Errorf("Has() returned true for missing NodeID")
	}
}

func TestCoreTypeInfo_Set(t *testing.T) {
	cti := NewCoreTypeInfo()
	floatType := &TCon{Name: "float"}

	cti.Set(20, floatType)

	if !cti.Has(20) {
		t.Errorf("Set() did not store type")
	}

	result := cti.Must(20)
	if result != floatType {
		t.Errorf("Set() stored wrong type: got %v, want %v", result, floatType)
	}
}

func TestCoreTypeInfo_NewCoreTypeInfo(t *testing.T) {
	cti := NewCoreTypeInfo()

	if cti == nil {
		t.Errorf("NewCoreTypeInfo() returned nil")
	}
	if len(cti) != 0 {
		t.Errorf("NewCoreTypeInfo() returned non-empty map: %d entries", len(cti))
	}
}

func TestCoreTypeInfo_ApplySubstitution(t *testing.T) {
	// Create type variables
	tv1 := &TVar2{Name: "t1", Kind: Star}
	tv2 := &TVar2{Name: "t2", Kind: Star}

	// Create CoreTypeInfo with type variables
	cti := NewCoreTypeInfo()
	cti.Set(1, tv1)                  // Node 1 has type variable t1
	cti.Set(2, tv2)                  // Node 2 has type variable t2
	cti.Set(3, TInt)                 // Node 3 has concrete type Int
	cti.Set(4, &TList{Element: tv1}) // Node 4 has List[t1]
	cti.Set(5, &TFunc2{              // Node 5 has function t1 -> t2
		Params:    []Type{tv1},
		Return:    tv2,
		EffectRow: EmptyEffectRow(),
	})

	// Create substitution: {t1 → Float, t2 → String}
	sub := Substitution{
		"t1": TFloat,
		"t2": TString,
	}

	// Apply substitution
	cti.ApplySubstitution(sub)

	// Verify t1 → Float
	result1, ok1 := cti.Get(1)
	if !ok1 {
		t.Errorf("Node 1 missing after substitution")
	}
	if result1 != TFloat {
		t.Errorf("Node 1: expected TFloat, got %v", result1)
	}

	// Verify t2 → String
	result2, ok2 := cti.Get(2)
	if !ok2 {
		t.Errorf("Node 2 missing after substitution")
	}
	if result2 != TString {
		t.Errorf("Node 2: expected TString, got %v", result2)
	}

	// Verify Int unchanged
	result3, ok3 := cti.Get(3)
	if !ok3 {
		t.Errorf("Node 3 missing after substitution")
	}
	if result3 != TInt {
		t.Errorf("Node 3: expected TInt, got %v", result3)
	}

	// Verify List[t1] → List[Float]
	result4, ok4 := cti.Get(4)
	if !ok4 {
		t.Errorf("Node 4 missing after substitution")
	}
	list4, ok := result4.(*TList)
	if !ok {
		t.Errorf("Node 4: expected TList, got %T", result4)
	} else if list4.Element != TFloat {
		t.Errorf("Node 4: expected List[Float], got List[%v]", list4.Element)
	}

	// Verify t1 -> t2 became Float -> String
	result5, ok5 := cti.Get(5)
	if !ok5 {
		t.Errorf("Node 5 missing after substitution")
	}
	func5, ok := result5.(*TFunc2)
	if !ok {
		t.Errorf("Node 5: expected TFunc2, got %T", result5)
	} else {
		if len(func5.Params) != 1 || func5.Params[0] != TFloat {
			t.Errorf("Node 5: expected param Float, got %v", func5.Params)
		}
		if func5.Return != TString {
			t.Errorf("Node 5: expected return String, got %v", func5.Return)
		}
	}
}
