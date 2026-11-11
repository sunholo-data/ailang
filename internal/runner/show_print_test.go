package runner

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/eval"
)

// captureStdout runs f with os.Stdout redirected and returns what it printed.
func captureStdout(t *testing.T, f func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stdout
	os.Stdout = w
	done := make(chan string)
	go func() {
		b, _ := io.ReadAll(r)
		done <- string(b)
	}()
	f()
	os.Stdout = old
	w.Close()
	return <-done
}

// `ailang run` prints a non-unit entry result with show, on every engine
// (M-V1-SHOW-TOTAL M4): nested strings quoted, floats in show's spelling.
func TestRunPrintsEntryResultWithShow(t *testing.T) {
	const src = `module print_entry
import std/option (Option, Some, None)

export func main() -> [(Option[string], float)] {
  [(Some("a, b"), 1.0e21), (None, -0.0)]
}
`
	const want = `[(Some("a, b"), 1000000000000000000000.0), (None, -0.0)]` + "\n"
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)
	file := filepath.Join(t.TempDir(), "print_entry.ail")
	for _, engine := range allEngines {
		t.Run(engine, func(t *testing.T) {
			rt, res := terminalFixtureRuntime(t, file, src, strings.NewReader(""), io.Discard)
			params := ModuleExecParams{
				Filename: file, Iface: res.Interface, Modules: res.Modules, Entry: "main",
				ArgsJSON: "null", Print: true, Quiet: true, PipelineResult: &res,
				BytecodeMode: engine != "eval", StrictBytecode: engine == "strict",
			}
			var runErr error
			got := captureStdout(t, func() { runErr = ExecuteModuleEntrypoint(rt, params) })
			if runErr != nil {
				t.Fatal(runErr)
			}
			if got != want {
				t.Fatalf("%s printed %q, want %q", engine, got, want)
			}
		})
	}
}

func TestPrintNonModuleResultUsesShow(t *testing.T) {
	v := &eval.ListValue{Elements: []eval.Value{&eval.StringValue{Value: "x"}, &eval.FloatValue{Value: 2}}}
	if got := captureStdout(t, func() { PrintNonModuleResult(v, true, false) }); got != "[\"x\", 2.0]\n" {
		t.Fatalf("printed %q", got)
	}
	if got := captureStdout(t, func() { PrintNonModuleResult(v, true, true) }); got != "" {
		t.Fatalf("--no-print printed %q", got)
	}
	if got := captureStdout(t, func() { PrintNonModuleResult(&eval.UnitValue{}, true, false) }); got != "" {
		t.Fatalf("unit printed %q", got)
	}
}
