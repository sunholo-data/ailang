package runner

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/pipeline"
)

// A local `letrec` must not leak into closures made earlier in the same
// block. evalCoreLetRec used to copy every letrec binding into the parent
// environment ("Phase 2.5"), which those closures had captured, so isEven's
// free `isOdd` was rebound to the LATER local letrec. Lexically, and for the
// type checker and the bytecode VM, it is the top-level `isOdd`.
const letrecLeakSource = `module letrec_leak
import std/io (println)

func isOdd(n: int) -> bool = true

func parity(n: int) -> bool {
  letrec isEven = \k. if k == 0 then true else isOdd(k - 1);
  letrec isOdd = \k. if k == 0 then false else isEven(k - 1);
  isEven(n)
}

export func main() -> () ! {IO} {
  println(show(parity(7)))
}
`

func TestLocalLetrecDoesNotLeakIntoEarlierClosures(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)
	file := filepath.Join(t.TempDir(), "letrec_leak.ail")

	// isEven(7) = top-level isOdd(6) = true.
	const want = "true\n"
	for _, engine := range []string{"eval", "bytecode", "strict"} {
		t.Run(engine, func(t *testing.T) {
			var out bytes.Buffer
			rt, res := terminalFixtureRuntime(t, file, letrecLeakSource, strings.NewReader(""), &out)
			err := ExecuteModuleEntrypoint(rt, ModuleExecParams{
				Filename: file, Iface: res.Interface, Modules: res.Modules, Entry: "main", ArgsJSON: "null",
				NoPrint: true, Quiet: true, PipelineResult: &res,
				BytecodeMode: engine != "eval", StrictBytecode: engine == "strict",
			})
			if err != nil {
				t.Fatalf("%s: %v", engine, err)
			}
			if out.String() != want {
				t.Fatalf("%s: stdout = %q, want %q (the lexical answer)", engine, out.String(), want)
			}
		})
	}

	// The type checker agrees the first letrec cannot see the second: with
	// no top-level isOdd, the name is unbound there.
	noOuter := strings.Replace(letrecLeakSource, "func isOdd(n: int) -> bool = true\n", "", 1)
	checkFile := filepath.Join(t.TempDir(), "letrec_leak_check.ail")
	if err := os.WriteFile(checkFile, []byte(noOuter), 0600); err != nil {
		t.Fatal(err)
	}
	res, err := pipeline.Run(pipeline.Config{Mode: pipeline.ModeCheck, TransientRoot: true}, pipeline.Source{Filename: checkFile, Code: noOuter})
	if err == nil && len(res.Errors) == 0 {
		t.Fatal("without a top-level isOdd the program type-checked; the first letrec must not see the second")
	}
}
