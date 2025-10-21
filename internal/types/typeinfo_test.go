package types

import (
	"testing"

	"github.com/sunholo/ailang/internal/ast"
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
