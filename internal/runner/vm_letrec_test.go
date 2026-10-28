package runner

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

// runOnEngine runs entry `main` of file (or of source, written to file) on
// one engine and returns its stdout: the evaluator, `--bytecode`
// (fallback allowed) or `--strict-bytecode` (fallback is an error).
func runOnEngine(t *testing.T, file, source, engine string) string {
	t.Helper()
	var out bytes.Buffer
	rt, res := terminalFixtureRuntime(t, file, source, strings.NewReader(""), &out)
	params := ModuleExecParams{
		Filename: file, Iface: res.Interface, Modules: res.Modules, Entry: "main",
		ArgsJSON: "null", NoPrint: true, Quiet: true, PipelineResult: &res,
		BytecodeMode:   engine != "eval",
		StrictBytecode: engine == "strict",
	}
	if err := ExecuteModuleEntrypoint(rt, params); err != nil {
		t.Fatalf("%s: %v", engine, err)
	}
	return out.String()
}

var allEngines = []string{"eval", "bytecode", "strict"}

// TestLetRecLocalShadowExample pins examples/runnable/letrec_local_shadow.ail
// on all three engines. Before M-VM-COVERAGE G1b, both VM engines printed
// 1000 for every line, because the local letrec name bound to the top-level
// decoy of the same name. Strict mode passing also proves the file runs
// natively, with no evaluator fallback.
func TestLetRecLocalShadowExample(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)
	const want = "sumTo(4) = 10\ncountUp(1000000) = 3000000\nwalkAll = [100, 101, 103]\nnestedRec(4) = 10\n"
	for _, engine := range allEngines {
		t.Run(engine, func(t *testing.T) {
			if got := runOnEngine(t, "examples/runnable/letrec_local_shadow.ail", "", engine); got != want {
				t.Fatalf("stdout = %q, want %q", got, want)
			}
		})
	}
}

// TestLetRecShadowPositions covers the positions a local letrec can sit in.
// Each lowers through a different path (block statement, value of a let,
// if branch, match arm, lambda body), and each once resolved the letrec
// name to the top-level decoy on the VM.
func TestLetRecShadowPositions(t *testing.T) {
	source := `module letrec_positions
import std/io (println)

func go(a: int) -> int = 1000

func twice(k: int) -> int = k * 2

func viaArg(n: int) -> int =
  twice({ letrec go = \i. if i <= 0 then 0 else i + go(i - 1); go(n) })

func viaIf(n: int) -> int =
  if n > 0 then { letrec go = \i. if i <= 0 then 0 else i + go(i - 1); go(n) } else 0

func viaMatch(o: Option[int]) -> int =
  match o {
    Some(n) => { letrec go = \i. if i <= 0 then 0 else i + go(i - 1); go(n) },
    None => 0
  }

func viaLambda(n: int) -> int {
  let f = \m. { letrec go = \i. if i <= 0 then 0 else i + go(i - 1); go(m) };
  f(n)
}

func paramShadow(n: int) -> int {
  letrec go = \go. if go <= 0 then 0 else go;
  go(n)
}

export func main() -> () ! {IO} {
  println(show(viaArg(4)));
  println(show(viaIf(4)));
  println(show(viaMatch(Some(4))));
  println(show(viaLambda(4)));
  println(show(paramShadow(3)))
}
`
	file := filepath.Join(t.TempDir(), "letrec_positions.ail")
	const want = "20\n10\n10\n10\n3\n"
	for _, engine := range allEngines {
		t.Run(engine, func(t *testing.T) {
			if got := runOnEngine(t, file, source, engine); got != want {
				t.Fatalf("stdout = %q, want %q", got, want)
			}
		})
	}
}
