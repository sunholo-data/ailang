package runner

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/pipeline"
	"github.com/sunholo-data/ailang/internal/runtime"
)

func terminalFixtureRuntime(t *testing.T, filename, source string, input io.Reader, output io.Writer) (*runtime.ModuleRuntime, pipeline.Result) {
	t.Helper()
	if source != "" {
		if err := os.WriteFile(filename, []byte(source), 0600); err != nil {
			t.Fatal(err)
		}
	}
	res, err := pipeline.Run(pipeline.Config{Mode: pipeline.ModeCheck, TransientRoot: source != ""}, pipeline.Source{Filename: filename, Code: source})
	if err != nil || len(res.Errors) > 0 {
		t.Fatalf("compile %s: %v %v", filename, err, res.Errors)
	}
	rt := runtime.NewModuleRuntime(".")
	if res.DictReg != nil {
		rt.GetEvaluator().SetDictionaryRegistry(res.DictReg)
	}
	ctx := effects.NewEffContext(nil)
	ctx.Grant(effects.Capability{Name: "IO"})
	ctx.IOReader = input
	ctx.IOWriter = output
	rt.GetEvaluator().SetEffContext(ctx)
	ctx.FnCaller = rt.GetEvaluator().CallValue
	ctx.FnCallerN = rt.GetEvaluator().CallValueN
	return rt, res
}

func TestTerminalExistingIOFixturesParity(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)
	cases := []struct{ file, entry, args, want string }{
		{"examples/runnable/micro_io_echo.ail", "main", "null", "Hello from AILANG!\n"},
		{"examples/runnable/test_io_builtins.ail", "greet", "null", "Hello from AILANG with builtins!\n"},
		{"examples/runnable/test_io_builtins.ail", "greet_name", `"Ada"`, "Ada\n"},
		{"examples/progress_bar.ail", "render", "50", "\r\x1b[92m[" + strings.Repeat("█", 10) + strings.Repeat("·", 10) + "]\x1b[0m 50%"},
	}
	for _, tc := range cases {
		for _, strict := range []bool{false, true} {
			t.Run(tc.entry+fmt.Sprintf("/strict=%v", strict), func(t *testing.T) {
				var output bytes.Buffer
				rt, res := terminalFixtureRuntime(t, tc.file, "", strings.NewReader(""), &output)
				params := ModuleExecParams{Filename: tc.file, Iface: res.Interface, Modules: res.Modules, Entry: tc.entry, ArgsJSON: tc.args, NoPrint: true, Quiet: true, BytecodeMode: strict, StrictBytecode: strict, PipelineResult: &res}
				if err := ExecuteModuleEntrypoint(rt, params); err != nil {
					t.Fatal(err)
				}
				if output.String() != tc.want {
					t.Fatalf("configured output = %q, want %q", output.String(), tc.want)
				}
			})
		}
	}
}

type failingTerminalReader struct {
	data     *strings.Reader
	failures int
}

func (r *failingTerminalReader) Read(p []byte) (int, error) {
	if r.data.Len() > 0 {
		return r.data.Read(p)
	}
	r.failures++
	return 0, fmt.Errorf("fixture input failed")
}

func TestNativeVMFailureDoesNotReplayIO(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)
	source := `module terminal_failure
import std/io (println, readLineOpt)
import std/option (Some, None)
export func main() -> () ! {IO} {
 println("ONCE");
 let first = readLineOpt();
 match first { Some(line) => println(line), None => println("EOF") };
 let ignored = readLineOpt();
 ()
}`
	var output bytes.Buffer
	reader := &failingTerminalReader{data: strings.NewReader("FIRST\n")}
	rt, res := terminalFixtureRuntime(t, filepath.Join(t.TempDir(), "terminal_failure.ail"), source, reader, &output)
	err = ExecuteModuleEntrypoint(rt, ModuleExecParams{Filename: "terminal_failure.ail", Iface: res.Interface, Modules: res.Modules, Entry: "main", ArgsJSON: "null", NoPrint: true, Quiet: true, BytecodeMode: true, PipelineResult: &res})
	if err == nil || !strings.Contains(err.Error(), "fixture input failed") {
		t.Fatalf("expected input error, got %v", err)
	}
	if reader.failures != 1 {
		t.Fatalf("failed read attempted %d times; input must remain consumed", reader.failures)
	}
	if output.String() != "ONCE\nFIRST\n" {
		t.Fatalf("live VM IO was replayed by fallback: %q", output.String())
	}
}
