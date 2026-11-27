package repl

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/eval"
)

// #2410: ModuleRegistry.LoadModule (WASM playground loadModule, REPL :import)
// must run the same soundness checks as the CLI pipeline: effect validation,
// match exhaustiveness (TC_MATCH_002), contract type-checking (TC_CONTRACT_001)
// and deriving (Eq) instance registration.

// TST-RC1-09
func TestLoadModuleParityRejectsPureFuncCallingPrintln(t *testing.T) {
	reg := NewModuleRegistry()
	code := `module parity_effects
export pure func shout(s: string) -> () = println(s)
`
	_, err := reg.LoadModule("parity_effects", code)
	if err == nil {
		t.Fatal("LoadModule accepted a pure func that calls println; the CLI rejects it (effect checking)")
	}
	if !strings.Contains(err.Error(), "effect") {
		t.Errorf("expected an effect-checking error, got: %v", err)
	}
	if _, getErr := reg.GetExport("parity_effects", "shout"); getErr == nil {
		t.Error("a module that failed effect checking must not publish its exports")
	}
}

// TST-RC1-09
func TestLoadModuleParityRejectsNonExhaustiveMatch(t *testing.T) {
	reg := NewModuleRegistry()
	code := `module parity_match
type Color = Red | Green | Blue
export pure func name(c: Color) -> string = match c {
  Red => "red",
  Green => "green"
}
`
	_, err := reg.LoadModule("parity_match", code)
	if err == nil {
		t.Fatal("LoadModule accepted a non-exhaustive match; the CLI rejects it with TC_MATCH_002")
	}
	if !strings.Contains(err.Error(), "TC_MATCH_002") && !strings.Contains(err.Error(), "Blue") {
		t.Errorf("expected TC_MATCH_002 naming the missing Blue, got: %v", err)
	}
}

// TST-RC1-09
func TestLoadModuleParityRejectsIllTypedContract(t *testing.T) {
	reg := NewModuleRegistry()
	code := `module parity_contract
export pure func half(x: int) -> int
requires { x ++ "a" }
{
  x / 2
}
`
	_, err := reg.LoadModule("parity_contract", code)
	if err == nil {
		t.Fatal("LoadModule accepted an ill-typed requires clause; the CLI rejects it with TC_CONTRACT_001")
	}
	if !strings.Contains(err.Error(), "TC_CONTRACT_001") && !strings.Contains(err.Error(), "requires") {
		t.Errorf("expected a contract type error, got: %v", err)
	}
}

// TST-RC1-09
func TestLoadModuleParityAcceptsWellFormedModule(t *testing.T) {
	reg := NewModuleRegistry()
	code := `module parity_ok
type Color = Red | Green | Blue
export pure func name(c: Color) -> string = match c {
  Red => "red",
  Green => "green",
  Blue => "blue"
}
export pure func half(x: int) -> int
requires { x >= 0 }
{
  x / 2
}
export func shout(s: string) -> () ! {IO} = println(s)
`
	if _, err := reg.LoadModule("parity_ok", code); err != nil {
		t.Fatalf("well-formed module rejected: %v", err)
	}
}

// TST-RC1-09
func TestLoadModuleParityDerivedEqAcrossModules(t *testing.T) {
	reg := NewModuleRegistry()
	shapes := `module parity_shapes
export type Shape = Circle(int) | Square(int) deriving (Eq)
export pure func circle(r: int) -> Shape = Circle(r)
`
	if _, err := reg.LoadModule("parity_shapes", shapes); err != nil {
		t.Fatalf("loading deriving (Eq) module: %v", err)
	}
	user := `module parity_shapes_user
import parity_shapes (Shape, circle)
export pure func same(a: int, b: int) -> bool = circle(a) == circle(b)
`
	if _, err := reg.LoadModule("parity_shapes_user", user); err != nil {
		t.Fatalf("loading module comparing an imported deriving (Eq) type: %v", err)
	}
	for _, tc := range []struct {
		a, b int
		want bool
	}{{3, 3, true}, {3, 4, false}} {
		got, err := reg.InvokeExport("parity_shapes_user", "same", []eval.Value{&eval.IntValue{Value: tc.a}, &eval.IntValue{Value: tc.b}})
		if err != nil {
			t.Fatalf("same(%d, %d): %v", tc.a, tc.b, err)
		}
		b, ok := got.(*eval.BoolValue)
		if !ok || b.Value != tc.want {
			t.Errorf("same(%d, %d) = %v, want %v", tc.a, tc.b, got, tc.want)
		}
	}
}
