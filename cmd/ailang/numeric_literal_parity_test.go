package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// M-SOUND-NUMERIC-LITERALS: an integer literal resolved at Float must be a
// float on BOTH engines. Every entry below passed `ailang check` before the
// fix and then failed at runtime ("expected float arguments", "missing
// dictionary method: prelude::Fractional::Int::add", the VM's "type mismatch
// Int vs Float", "scatterAdd: arg 0 element 0 must be float") or returned an
// Int where the type said float (retZero printed 0, tuple printed (1, 1.5)).
// Strict bytecode proves the VM ran it rather than bridging to the evaluator.
const numericLiteralSource = `module numeric_literal
import std/array (make, scatterAdd)

export pure func cmpGe(f: float) -> bool = f >= 0
export pure func litLeft(f: float) -> bool = 0 < f
export pure func addOne(f: float) -> float = f + 1
export pure func mulTwo(f: float) -> float = 2 * f
export pure func retZero(f: float) -> float = 0
export pure func tuple(f: float) -> (float, float) = (1, f)
export pure func list(f: float) -> [float] = [1, f, 3]
export pure func letLit(h: float) -> float { let w = 70; w / (h * h) }
export pure func letInc(f: float) -> float { let inc = \x. x + 1; inc(f) }
export pure func scatter(f: float) -> Array[float] = scatterAdd(make(3, 0), [0, 0, 2], [1, 2, 5])
export pure func intsStayInts(f: float) -> int { let n = 7; n / 2 }
`

func TestNumericLiteralParityVMvsInterpreter(t *testing.T) {
	bin := buildAilang(t)
	file := filepath.Join(t.TempDir(), "numeric_literal.ail")
	if err := os.WriteFile(file, []byte(numericLiteralSource), 0o644); err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"cmpGe":        "true",
		"litLeft":      "true",
		"addOne":       "3.0",
		"mulTwo":       "4.0",
		"retZero":      "0.0",
		"tuple":        "(1.0, 2.0)",
		"list":         "[1.0, 2.0, 3.0]",
		"letLit":       "17.5",
		"letInc":       "3.0",
		"scatter":      "#[3.0, 0.0, 5.0]",
		"intsStayInts": "3",
	}
	for entry, w := range want {
		interp, ierr, icode := runAilangBin(t, bin, "run", "--quiet", "--entry", entry, "--args-json", "2.0", file)
		vm, verr, vcode := runAilangBin(t, bin, "run", "--quiet", "--bytecode", "--strict-bytecode", "--entry", entry, "--args-json", "2.0", file)
		if icode != 0 || vcode != 0 {
			t.Errorf("%s: exit interp=%d vm=%d\ninterp stderr: %s\nvm stderr: %s", entry, icode, vcode, ierr, verr)
			continue
		}
		if got := strings.TrimSpace(interp); got != w {
			t.Errorf("%s: interpreter %q, want %q", entry, got, w)
		}
		if got := strings.TrimSpace(vm); got != w {
			t.Errorf("%s: VM %q, want %q", entry, got, w)
		}
	}
}

// The residue the restriction does not reach: an integer literal inside a
// generic TOP-LEVEL function instantiated at float. It still fails at
// runtime, but with a message naming the cause and the fix instead of
// "expected float arguments".
func TestNumericLiteralGenericResidueFailsLoudly(t *testing.T) {
	bin := buildAilang(t)
	file := filepath.Join(t.TempDir(), "numeric_residue.ail")
	src := `module numeric_residue
func incG[a](x: a) -> a { x + 1 }
export pure func run(f: float) -> float = incG(f)
`
	if err := os.WriteFile(file, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	_, stderr, code := runAilangBin(t, bin, "run", "--quiet", "--entry", "run", "--args-json", "2.5", file)
	if code == 0 {
		t.Fatalf("expected a runtime error for an int literal at a generic float instantiation")
	}
	if !strings.Contains(stderr, "received an int and a float") {
		t.Errorf("error should name the int-literal-at-generic-type cause, got: %s", stderr)
	}
}
