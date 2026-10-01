package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// #1453: `show` must render identically on the bytecode VM and in the
// interpreter. The VM used to print ADTs as "<adt#0 6>" (it kept only the
// constructor ordinal), and it skipped the evaluator's depth limit and
// 80-column elision. Each entry runs on both engines; strict bytecode proves
// the VM rendered it rather than bridging back to the evaluator.
const showParitySource = `module show_parity
import std/option (Option, Some, None)
import std/result (Result, Ok, Err)
import std/list (range)

type Shape = Circle(float) | Rect(float, float) | Dot

export pure func someInt(n: int) -> string = show(Some(n))
export pure func none(n: int) -> string = show(if n > 100 then Some(n) else None)
export pure func nested(n: int) -> string = show(Some(Ok(n)))
export pure func err(n: int) -> string = show(Err("bad"))
export pure func user(n: int) -> string = show([Circle(1.5), Rect(2.0, 3.0), Dot])
export pure func inList(n: int) -> string = show([Some(n), None])
export pure func longList(n: int) -> string = show(range(0, n))
export pure func deep(n: int) -> string = show([[[[[n]]]]])
export pure func longRecord(n: int) -> string = show({alpha: range(0, 10), beta: "a long string value here", gamma: n})
export pure func tuple(n: int) -> string = show((Some(n), "x", 1.0))
export pure func retSome(n: int) -> Option[int] = Some(n)
export pure func retList(n: int) -> [Option[float]] = [Some(2.0), None]
export pure func retShape(n: int) -> Shape = Rect(1.0, 2.5)
`

func TestShowParityVMvsInterpreter(t *testing.T) {
	bin := buildAilang(t)
	file := filepath.Join(t.TempDir(), "show_parity.ail")
	if err := os.WriteFile(file, []byte(showParitySource), 0o644); err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"someInt": "Some(40)",
		"none":    "None",
		"nested":  "Some(Ok(40))",
		"err":     "Err(bad)",
		"user":    "[Circle(1.5), Rect(2.0, 3.0), Dot]",
		"retSome": "Some(40)",
		"retList": "[Some(2.0), None]",
	}
	for _, entry := range []string{"someInt", "none", "nested", "err", "user", "inList", "longList", "deep", "longRecord", "tuple", "retSome", "retList", "retShape"} {
		interp, ierr, icode := runAilangBin(t, bin, "run", "--quiet", "--entry", entry, "--args-json", "40", file)
		vm, verr, vcode := runAilangBin(t, bin, "run", "--quiet", "--bytecode", "--strict-bytecode", "--entry", entry, "--args-json", "40", file)
		if icode != 0 || vcode != 0 {
			t.Errorf("%s: exit interp=%d vm=%d\ninterp stderr: %s\nvm stderr: %s", entry, icode, vcode, ierr, verr)
			continue
		}
		if interp != vm {
			t.Errorf("%s: interpreter %q, VM %q", entry, strings.TrimSpace(interp), strings.TrimSpace(vm))
		}
		if w, ok := want[entry]; ok && strings.TrimSpace(interp) != w {
			t.Errorf("%s: interpreter %q, want %q", entry, strings.TrimSpace(interp), w)
		}
	}
}
