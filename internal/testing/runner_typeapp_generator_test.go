package testing

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
)

func TestCreateGeneratorForType_TypeAppList(t *testing.T) {
	runner := NewRunner("test.ail")

	gen, shrink := runner.createGeneratorForType(&ast.TypeApp{
		Constructor: "list",
		Args:        []ast.Type{&ast.SimpleType{Name: "int"}},
	})
	if gen == nil {
		t.Fatal("expected generator for list[int] TypeApp")
	}
	if shrink == nil {
		t.Fatal("expected shrinker for list[int] TypeApp")
	}
}

func TestCreateGeneratorForType_TypeAppListUnsupportedElement(t *testing.T) {
	runner := NewRunner("test.ail")

	gen, shrink := runner.createGeneratorForType(&ast.TypeApp{
		Constructor: "list",
		Args:        []ast.Type{&ast.SimpleType{Name: "Tree"}},
	})
	if gen != nil || shrink != nil {
		t.Fatalf("expected no generator fallback for list[Tree], got generator=%T shrinker=%T", gen, shrink)
	}
}
