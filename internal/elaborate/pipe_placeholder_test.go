package elaborate_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/bytecode"
	"github.com/sunholo-data/ailang/internal/effects"
	ailerrors "github.com/sunholo-data/ailang/internal/errors"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/pipeline"
	"github.com/sunholo-data/ailang/internal/runner"
	"github.com/sunholo-data/ailang/internal/runtime"
	"github.com/sunholo-data/ailang/internal/vm"
)

func evalAndStrictBytecode(t *testing.T, dir, name, src string) (string, string) {
	t.Helper()
	path := filepath.Join(dir, name+".ail")
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := pipeline.Run(pipeline.Config{
		Mode:         pipeline.ModeEval,
		RelaxModules: true,
		NoCache:      true,
	}, pipeline.Source{
		Code:     src,
		Filename: path,
	})
	if err != nil {
		t.Fatalf("pipeline.Run(%s) failed: %v", filepath.ToSlash(path), err)
	}
	if len(res.Errors) > 0 {
		t.Fatalf("pipeline.Run(%s) errors: %v", filepath.ToSlash(path), res.Errors)
	}

	// 1. Run via tree-walking evaluator
	rt := runtime.NewModuleRuntime(dir)
	rt.GetEvaluator().SetEffContext(effects.NewEffContext(nil))
	for modID, loaded := range res.Modules {
		rt.PreloadModule(modID, loaded)
	}
	inst, err := rt.LoadAndEvaluate(res.Interface.Module)
	if err != nil {
		t.Fatalf("LoadAndEvaluate(%s): %v", name, err)
	}
	evalVal, err := runtime.CallEntrypoint(rt, inst, "main", nil)
	if err != nil {
		t.Fatalf("evaluator CallEntrypoint(%s): %v", name, err)
	}

	// 2. Run via strict bytecode VM
	img, err := runner.CompileBytecodeFromResult(res, res.Interface.Module)
	if err != nil {
		t.Fatalf("CompileBytecodeFromResult(%s): %v", name, err)
	}
	if err := img.Validate(); err != nil {
		t.Fatalf("img.Validate(%s): %v", name, err)
	}
	entryMod := runner.EntryModuleID(res)
	proto := runner.FindEntryProto(img, entryMod, "main")
	if proto == nil {
		t.Fatalf("FindEntryProto(%s) returned nil", name)
	}
	if evalOnly := img.EvalOnlyReachable([]int{img.ProtoIndex(proto)}); len(evalOnly) > 0 {
		t.Fatalf("strict-bytecode EvalOnlyReachable(%s): %v", name, evalOnly)
	}
	machine := vm.NewVM(img)
	machine.Effects = effects.NewEffContext(nil)
	var bcArgs []bytecode.Value
	if proto.NumParams == 1 {
		bcArgs = []bytecode.Value{bytecode.Unit()}
	}
	vmVal, err := machine.Run(proto, bcArgs)
	if err != nil {
		t.Fatalf("VM Run(%s): %v", name, err)
	}
	vmAsEval, err := vm.BytecodeToEval(vmVal)
	if err != nil {
		t.Fatalf("BytecodeToEval(%s): %v", name, err)
	}
	return eval.Show(evalVal), eval.Show(vmAsEval)
}

// TST-RC1-11
func TestPipePlaceholderElaborationAndParity(t *testing.T) {
	dir := t.TempDir()

	t.Run("chained_data_last_and_data_first_parity", func(t *testing.T) {
		src := `module pipe_ok
import std/list (filter, map, foldl)
import std/string (split, join)

pure func makeAdder(a: int) -> int -> int = \b. a + b

export pure func main() -> string {
  let nums = [-2, 1, 3, -1, 4] |> filter(\x. x > 0, _) |> map(\x. x * 10, _) |> foldl(\acc x. acc + x, 0, _)
  let curried = 5 |> makeAdder(1)
  let nestedWild = [1, 2, 3] |> map(\_. 9, _) |> foldl(\acc x. acc + x, 0, _)
  let parts = "a,b,c" |> split(_, ",") |> join("-", _)
  "${show(nums)}:${show(curried)}:${show(nestedWild)}:${parts}"
}
`
		evalOut, vmOut := evalAndStrictBytecode(t, dir, "pipe_ok", src)
		want := "80:6:27:a-b-c"
		if evalOut != want {
			t.Errorf("evaluator got %s, want %s", evalOut, want)
		}
		if vmOut != want {
			t.Errorf("strict-bytecode VM got %s, want %s", vmOut, want)
		}
	})

	t.Run("parenthesized_rhs_call_with_placeholder", func(t *testing.T) {
		src := `module pipe_parens
pure func sub(a: int, b: int) -> int = a - b
export pure func main() -> int = 3 |> (sub(10, _))
`
		evalOut, vmOut := evalAndStrictBytecode(t, dir, "pipe_parens", src)
		if evalOut != "7" || vmOut != "7" {
			t.Fatalf("got eval=%s vm=%s, want 7", evalOut, vmOut)
		}
	})

	t.Run("multiple_placeholders_rejected_with_TC_PIPE_001", func(t *testing.T) {
		src := `module pipe_multi
pure func add(a: int, b: int) -> int = a + b
export pure func main() -> int = 5 |> add(_, _)
`
		err := runCheck(t, dir, "pipe_multi", src)
		if err == nil {
			t.Fatal("expected TC_PIPE_001 error for x |> add(_, _), got nil")
		}
		rep, ok := ailerrors.AsReport(err)
		if !ok || rep.Code != "TC_PIPE_001" {
			t.Fatalf("expected structured TC_PIPE_001 report, got ok=%v err=%v", ok, err)
		}
		if !strings.Contains(err.Error(), "multiple '_' placeholders") {
			t.Errorf("unexpected TC_PIPE_001 message: %v", err)
		}
	})

	t.Run("unnamed_hole_remains_typed_hole", func(t *testing.T) {
		src := `module pipe_hole
pure func add(a: int, b: int) -> int = a + b
export pure func main() -> int = add(1, _?)
`
		err := runCheck(t, dir, "pipe_hole", src)
		if err == nil || !strings.Contains(err.Error(), "TC_HOLE") {
			t.Fatalf("expected TC_HOLE for f(1, _?), got: %v", err)
		}
	})

	t.Run("bare_wildcard_outside_pipe_still_errors_including_zero_arg_func", func(t *testing.T) {
		cases := []struct {
			name string
			src  string
		}{
			{
				name: "int_arg",
				src: `module bare_wild_int
pure func add(a: int, b: int) -> int = a + b
export pure func f(x: int) -> int = add(x, _)
`,
			},
			{
				name: "unit_arg_inside_zero_arg_main",
				src: `module bare_wild_unit
pure func takeUnit(a: int, u: ()) -> int = a
export pure func main() -> int = takeUnit(42, _)
`,
			},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				if err := runCheck(t, dir, tc.name, tc.src); err == nil {
					t.Fatalf("%s: expected bare '_' outside |> to error, got nil", tc.name)
				}
			})
		}
	})
}
