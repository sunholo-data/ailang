package pipeline

import (
	"fmt"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
)

// A lowercase primitive name in annotation position must be a concrete type,
// never a type variable. `bytes` was once missing from the parser's list, so
// `f(b: bytes) -> bytes` was silently polymorphic and accepted `f(42)`.
func TestBuiltinTypeAnnotation_IntPassedAsBytesRejected(t *testing.T) {
	src := `module bytes_annot
pure func f(b: bytes) -> bytes = b
export pure func g() -> bytes = f(42)
`
	err := checkSource(t, "bytes_annot", src)
	if err == nil {
		t.Fatal("f(42) against a bytes parameter type-checked; bytes annotation is being treated as a type variable")
	}
	if !strings.Contains(err.Error(), "bytes") {
		t.Fatalf("expected the error to name bytes, got: %v", err)
	}
}

// Every name in the canonical list must reject an int argument. This catches a
// name that the parser accepts but a converter downstream maps back to a
// type variable. int and float are excluded: an int literal is a valid value
// of both.
func TestBuiltinTypeAnnotation_EveryPrimitiveIsConcrete(t *testing.T) {
	for _, name := range ast.BuiltinTypeNames() {
		if name == "int" || name == "float" {
			continue
		}
		t.Run(name, func(t *testing.T) {
			mod := "prim_" + name
			src := fmt.Sprintf(`module %s
pure func f(x: %s) -> %s = x
export pure func g() -> %s = f(42)
`, mod, name, name, name)
			if err := checkSource(t, mod, src); err == nil {
				t.Fatalf("f(42) against a %s parameter type-checked; %s is being treated as a type variable", name, name)
			}
		})
	}
}

// The positive control: a correctly typed bytes function still checks.
func TestBuiltinTypeAnnotation_BytesIdentityChecks(t *testing.T) {
	src := `module bytes_ok
import std/bytes (fromString)
pure func f(b: bytes) -> bytes = b
export pure func g() -> bytes = f(fromString("hi"))
`
	if err := checkSource(t, "bytes_ok", src); err != nil {
		t.Fatalf("well-typed bytes identity failed to check: %v", err)
	}
}
