package runner

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

// M-VM-COVERAGE G3: a module-level value binding read on the VM is its
// value. The VM used to read it as a closure and fail mid-run with
// `arith ADD: type mismatch Closure vs Int`, after "start" was printed.
// `scale` is the shape the soundness agent reported: a float constant read
// inside a non-main function (`arith MUL: type mismatch Float vs Closure`).
func TestModuleLevelValueOnVM(t *testing.T) {
	source := `module caf_values
import std/io (println)

let base = 40 + 1
let derived = base * 2
let names = ["a", "b"]
let quad = \y. y * 4
let halfF = 2.0

func scale(x: float) -> float = x * halfF

export func main() -> () ! {IO} = {
  println("start");
  println(show(base + 1));
  println(show(derived));
  println(show(names));
  println(show(quad(base)));
  println(show(scale(3.0)))
}
`
	file := filepath.Join(t.TempDir(), "caf_values.ail")
	const want = "start\n42\n82\n[a, b]\n164\n6.0\n"
	for _, engine := range allEngines {
		t.Run(engine, func(t *testing.T) {
			if got := runOnEngine(t, file, source, engine); got != want {
				t.Fatalf("stdout = %q, want %q", got, want)
			}
		})
	}
}

// M-VM-COVERAGE G4 safety net (design doc V14): native `main` prints, then
// passes a user ADT to an evaluator-only function. The bridge cannot carry
// the ADT, and the failure used to arrive after "start" had been printed, so
// the run could not be replayed and exited 1. Now the compiler sees the edge:
// --bytecode runs the whole program on the evaluator, and --strict-bytecode
// refuses before any effect.
func TestUnbridgeableEdgeFallsBackBeforeEffects(t *testing.T) {
	source := `module adt_bridge
import std/io (println)
import std/debug (log)

type Dir = U | D

func step(d: Dir, n: int) -> int {
  log("step");
  match d {
    U => n,
    D => 0 - n
  }
}

export func main() -> () ! {IO} {
  println("start");
  println(show(step(U, 5)))
}
`
	file := filepath.Join(t.TempDir(), "adt_bridge.ail")
	const want = "start\n5\n"
	for _, engine := range []string{"eval", "bytecode"} {
		if got := runOnEngine(t, file, source, engine); got != want {
			t.Errorf("%s: stdout = %q, want %q", engine, got, want)
		}
	}

	var out bytes.Buffer
	rt, res := terminalFixtureRuntime(t, file, source, strings.NewReader(""), &out)
	err := ExecuteModuleEntrypoint(rt, ModuleExecParams{
		Filename: file, Iface: res.Interface, Modules: res.Modules, Entry: "main", ArgsJSON: "null",
		NoPrint: true, Quiet: true, PipelineResult: &res, BytecodeMode: true, StrictBytecode: true,
	})
	if err == nil || !strings.Contains(err.Error(), "cannot cross the VM/evaluator bridge") {
		t.Fatalf("strict: err = %v, want a refusal naming the bridge", err)
	}
	if out.Len() != 0 {
		t.Fatalf("strict: %q was printed before the refusal; it must refuse before any effect", out.String())
	}
}
