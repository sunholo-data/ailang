package types

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
)

// Every primitive the parser emits as a SimpleType must convert to a concrete
// TCon, never a type variable. "unit" and "char" used to fall through to the
// lowercase-means-TVar default here.
func TestAstTypeToType_BuiltinNamesAreConcrete(t *testing.T) {
	tc := NewTypeChecker()
	for _, name := range ast.BuiltinTypeNames() {
		got := tc.astTypeToType(&ast.SimpleType{Name: name})
		if _, ok := got.(*TCon); !ok {
			t.Errorf("astTypeToType(%q) = %T (%v), want *TCon", name, got, got)
		}
	}
}
