package runner

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/bytecode"
	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/pipeline"
	"github.com/sunholo-data/ailang/internal/runtime"
	"github.com/sunholo-data/ailang/internal/types"
	"github.com/sunholo-data/ailang/internal/vm"
)

func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stderr = w
	defer func() { os.Stderr = orig }()

	done := make(chan string, 1)
	go func() {
		data, _ := io.ReadAll(r)
		done <- string(data)
	}()

	fn()
	_ = w.Close()
	out := <-done
	_ = r.Close()
	return filepath.ToSlash(out)
}

// TST-RC1-01
func TestSound1_MultiModuleEqCheckAndEngineParity(t *testing.T) {
	// 1. Cross-module collision (`s1/a.Regex` record with closure field alongside `import std/regex`)
	// must fail type checking (`No instance for Eq`).
	t.Run("cross-module Regex record without deriving Eq fails check", func(t *testing.T) {
		root := t.TempDir()
		t.Chdir(root)
		if err := os.MkdirAll("s1", 0o755); err != nil {
			t.Fatal(err)
		}
		modA := `module s1/a

export type Regex = { pattern: string, matcher: string -> bool }

export pure func mk(p: string) -> Regex =
  { pattern: p, matcher: \s. s == p }
`
		modB := `module s1/b
import std/io (println)
import std/regex (Regex as StdRegex, compile)
import s1/a (Regex, mk)

export func main() -> () ! {IO} {
  let _ = compile("abc");
  let r1: Regex = { pattern: "a", matcher: \s. s == "a" };
  let r2: Regex = mk("a");
  println(show(r1 == r2))
}
`
		if err := os.WriteFile(filepath.Join("s1", "a.ail"), []byte(modA), 0o644); err != nil {
			t.Fatal(err)
		}
		entry := filepath.Join(root, "s1", "b.ail")
		if err := os.WriteFile(entry, []byte(modB), 0o644); err != nil {
			t.Fatal(err)
		}
		res, err := pipeline.Run(pipeline.Config{
			Mode:    pipeline.ModeCheck,
			NoCache: true,
			InstEnv: types.LoadBuiltinInstances(),
			DictReg: types.NewDictionaryRegistry(),
		}, pipeline.Source{Filename: entry})
		combined := ""
		if err != nil {
			combined += err.Error()
		}
		for _, e := range res.Errors {
			combined += "\n" + e.Error()
		}
		if combined == "" || !strings.Contains(combined, "No instance for Eq") {
			t.Fatalf("expected check failure containing 'No instance for Eq', got err=%v errors=%v", err, res.Errors)
		}
	})

	// 2. Intra-module and imported `deriving (Eq)` records must pass on BOTH evaluator and --strict-bytecode.
	t.Run("valid intra-module and imported deriving Eq records pass both engines", func(t *testing.T) {
		root := t.TempDir()
		t.Chdir(root)
		if err := os.MkdirAll("geom", 0o755); err != nil {
			t.Fatal(err)
		}
		modPt := `module geom/pt

export type Pt = { x: int, y: int } deriving (Eq)

export pure func mkPt(x: int, y: int) -> Pt = { x: x, y: y }
`
		modMain := `module geom/main
import std/io (println)
import geom/pt (Pt, mkPt)

type LocalBox = { label: string, pt: Pt } deriving (Eq)

export func main() -> () ! {IO} {
  let a = mkPt(3, 4);
  let b = mkPt(3, 4);
  let c = mkPt(3, 5);
  let b1: LocalBox = { label: "ok", pt: a };
  let b2: LocalBox = { label: "ok", pt: b };
  println("${show(a == b)} ${show(a == c)} ${show(b1 == b2)}")
}
`
		if err := os.WriteFile(filepath.Join("geom", "pt.ail"), []byte(modPt), 0o644); err != nil {
			t.Fatal(err)
		}
		entry := filepath.Join(root, "geom", "main.ail")
		if err := os.WriteFile(entry, []byte(modMain), 0o644); err != nil {
			t.Fatal(err)
		}
		runOnEngines(t, entry, "true false true\n", []string{"eval", "strict-bytecode"})
	})
}

// TST-RC1-02
func TestVMOpEqTrapsOnClosuresAndPreservesConstantPoolDedup(t *testing.T) {
	leafProto := &bytecode.FuncPrototype{
		Name:      "leaf",
		NumRegs:   2,
		NumParams: 1,
		Instructions: []bytecode.Instruction{
			bytecode.EncodeABC(bytecode.OpReturn, 0, 0, 0),
		},
	}
	closA := bytecode.NewClosure(leafProto, nil)
	closB := bytecode.NewClosure(leafProto, []bytecode.Value{bytecode.NewInt(1)})

	// Constant-pool dedup via bytecode.Value.Equal MUST remain intact.
	if !closA.Equal(bytecode.NewClosure(leafProto, nil)) {
		t.Fatal("bytecode.Value.Equal must still compare identical closures true for constant-pool dedup")
	}
	if closA.Equal(closB) {
		t.Fatal("bytecode.Value.Equal must distinguish closures with different captures")
	}

	runEqProg := func(t *testing.T, negate bool, lhs, rhs bytecode.Value) error {
		t.Helper()
		img := bytecode.NewImage()
		img.AddPrototype(leafProto)
		entry := &bytecode.FuncPrototype{
			Name:      "cmp",
			File:      "cmp.ail",
			NumRegs:   3,
			NumParams: 2,
			Instructions: []bytecode.Instruction{
				bytecode.EncodeABC(bytecode.OpEq, 2, 0, 1),
				bytecode.EncodeABC(bytecode.OpReturn, 2, 0, 0),
			},
			LineInfo: []int{5, 5},
		}
		if negate {
			entry.Instructions = []bytecode.Instruction{
				bytecode.EncodeABC(bytecode.OpEq, 2, 0, 1),
				bytecode.EncodeABC(bytecode.OpNot, 2, 2, 0),
				bytecode.EncodeABC(bytecode.OpReturn, 2, 0, 0),
			}
			entry.LineInfo = []int{5, 5, 5}
		}
		img.AddPrototype(entry)
		_, err := vm.NewVM(img).Run(entry, []bytecode.Value{lhs, rhs})
		return err
	}

	cases := []struct {
		name string
		lhs  bytecode.Value
		rhs  bytecode.Value
	}{
		{"direct closure", closA, closA},
		{"distinct closures", closA, closB},
		{"record-nested closure",
			bytecode.NewRecord([]bytecode.RecordField{{Name: "f", Value: closA}, {Name: "x", Value: bytecode.NewInt(1)}}),
			bytecode.NewRecord([]bytecode.RecordField{{Name: "f", Value: closB}, {Name: "x", Value: bytecode.NewInt(1)}}),
		},
		{"tuple-nested closure",
			bytecode.NewTuple([]bytecode.Value{bytecode.NewInt(1), closA}),
			bytecode.NewTuple([]bytecode.Value{bytecode.NewInt(1), closA}),
		},
		{"list-nested closure",
			bytecode.NewList([]bytecode.Value{closA}),
			bytecode.NewList([]bytecode.Value{closB}),
		},
		{"array-nested closure",
			bytecode.NewArray([]bytecode.Value{closA}),
			bytecode.NewArray([]bytecode.Value{closA}),
		},
		{"ADT-nested closure same tag",
			bytecode.NewADT(0, "Some", []bytecode.Value{closA}),
			bytecode.NewADT(0, "Some", []bytecode.Value{closB}),
		},
		{"ADT-nested closure different tag (Some(f) == None)",
			bytecode.NewADT(0, "Some", []bytecode.Value{closA}),
			bytecode.NewADT(1, "None", nil),
		},
	}

	for _, tc := range cases {
		for _, neg := range []bool{false, true} {
			opName := "=="
			if neg {
				opName = "!="
			}
			t.Run(tc.name+"_"+opName, func(t *testing.T) {
				err := runEqProg(t, neg, tc.lhs, tc.rhs)
				if err == nil {
					t.Fatalf("expected RT_EQ_001 trap for %s (%s), got nil", tc.name, opName)
				}
				msg := filepath.ToSlash(err.Error())
				if !strings.Contains(msg, "RT_EQ_001") {
					t.Fatalf("expected RT_EQ_001 in error, got: %s", msg)
				}
				if strings.Contains(msg, "vm: vm:") {
					t.Fatalf("unexpected doubled 'vm: vm:' prefix in error: %s", msg)
				}
			})
		}
	}
}

// TST-RC1-03
func TestBytecodeRuntimeTrapDoesNotReplayOnEvaluator(t *testing.T) {
	t.Run("effectful div-by-zero runs side effects once with single vm prefix", func(t *testing.T) {
		root := t.TempDir()
		t.Chdir(root)
		src := `module trap_div
import std/io (println)

export func main() -> int ! {IO} {
  println("SIDE_EFFECT");
  10 / 0
}
`
		entry := filepath.Join(root, "trap_div.ail")
		if err := os.WriteFile(entry, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
		res, err := pipeline.Run(pipeline.Config{
			Mode:    pipeline.ModeCheck,
			NoCache: true,
			InstEnv: types.LoadBuiltinInstances(),
			DictReg: types.NewDictionaryRegistry(),
		}, pipeline.Source{Filename: entry})
		if err != nil || len(res.Errors) > 0 {
			t.Fatalf("compile: %v %v", err, res.Errors)
		}

		var out bytes.Buffer
		rt := runtime.NewModuleRuntime(".")
		if res.DictReg != nil {
			rt.GetEvaluator().SetDictionaryRegistry(res.DictReg)
		}
		ctx := effects.NewEffContext(nil)
		ctx.Grant(effects.Capability{Name: "IO"})
		ctx.IOWriter = &out
		rt.GetEvaluator().SetEffContext(ctx)
		ctx.FnCaller = rt.GetEvaluator().CallValue
		ctx.FnCallerN = rt.GetEvaluator().CallValueN

		var runErr error
		stderr := captureStderr(t, func() {
			runErr = ExecuteModuleEntrypoint(rt, ModuleExecParams{
				Filename: entry, Iface: res.Interface, Modules: res.Modules, Entry: "main", ArgsJSON: "null",
				NoPrint: true, Quiet: true, PipelineResult: &res,
				BytecodeMode: true, StrictBytecode: false,
			})
		})
		if runErr == nil {
			t.Fatal("expected division-by-zero error, got nil")
		}
		errMsg := filepath.ToSlash(runErr.Error())
		if !strings.Contains(errMsg, "RT001") {
			t.Fatalf("expected RT001 in error, got: %s", errMsg)
		}
		if strings.Contains(errMsg, "vm: vm:") {
			t.Fatalf("expected single 'vm: ' prefix, got doubled prefix: %s", errMsg)
		}
		if !strings.HasPrefix(errMsg, "vm: ") {
			t.Fatalf("expected error to start with 'vm: ', got: %s", errMsg)
		}
		if strings.Contains(stderr, "bytecode path unavailable") {
			t.Fatalf("post-dispatch VM trap must NOT emit fallback warning, got stderr: %s", stderr)
		}
		if gotCount := strings.Count(out.String(), "SIDE_EFFECT"); gotCount != 1 {
			t.Fatalf("expected SIDE_EFFECT printed exactly once, got %d (%q)", gotCount, out.String())
		}
	})

	t.Run("pure div-by-zero before any effect call also does not fall back or double vm prefix", func(t *testing.T) {
		root := t.TempDir()
		t.Chdir(root)
		src := `module pure_div

export pure func main() -> int = 10 / 0
`
		entry := filepath.Join(root, "pure_div.ail")
		if err := os.WriteFile(entry, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
		res, err := pipeline.Run(pipeline.Config{
			Mode:    pipeline.ModeCheck,
			NoCache: true,
			InstEnv: types.LoadBuiltinInstances(),
			DictReg: types.NewDictionaryRegistry(),
		}, pipeline.Source{Filename: entry})
		if err != nil || len(res.Errors) > 0 {
			t.Fatalf("compile: %v %v", err, res.Errors)
		}

		rt := runtime.NewModuleRuntime(".")
		if res.DictReg != nil {
			rt.GetEvaluator().SetDictionaryRegistry(res.DictReg)
		}
		ctx := effects.NewEffContext(nil)
		rt.GetEvaluator().SetEffContext(ctx)

		var runErr error
		stderr := captureStderr(t, func() {
			runErr = ExecuteModuleEntrypoint(rt, ModuleExecParams{
				Filename: entry, Iface: res.Interface, Modules: res.Modules, Entry: "main", ArgsJSON: "null",
				NoPrint: true, Quiet: true, PipelineResult: &res,
				BytecodeMode: true, StrictBytecode: false,
			})
		})
		if runErr == nil {
			t.Fatal("expected RT001 error, got nil")
		}
		errMsg := filepath.ToSlash(runErr.Error())
		if strings.Contains(errMsg, "vm: vm:") {
			t.Fatalf("expected single 'vm: ' prefix, got: %s", errMsg)
		}
		if !strings.HasPrefix(errMsg, "vm: ") || !strings.Contains(errMsg, "RT001") {
			t.Fatalf("expected 'vm: RT001...', got: %s", errMsg)
		}
		if strings.Contains(stderr, "bytecode path unavailable") {
			t.Fatalf("post-dispatch pure VM trap must NOT fall back to evaluator, got stderr: %s", stderr)
		}
	})

	t.Run("stack overflow under --bytecode does not fall back or double vm prefix", func(t *testing.T) {
		root := t.TempDir()
		t.Chdir(root)
		src := `module trap_stack
import std/io (println)

func dive(n: int) -> int = 1 + dive(n + 1)

export func main() -> int ! {IO} {
  println("SIDE_EFFECT");
  dive(0)
}
`
		entry := filepath.Join(root, "trap_stack.ail")
		if err := os.WriteFile(entry, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
		res, err := pipeline.Run(pipeline.Config{
			Mode:    pipeline.ModeCheck,
			NoCache: true,
			InstEnv: types.LoadBuiltinInstances(),
			DictReg: types.NewDictionaryRegistry(),
		}, pipeline.Source{Filename: entry})
		if err != nil || len(res.Errors) > 0 {
			t.Fatalf("compile: %v %v", err, res.Errors)
		}

		var out bytes.Buffer
		rt := runtime.NewModuleRuntime(".")
		if res.DictReg != nil {
			rt.GetEvaluator().SetDictionaryRegistry(res.DictReg)
		}
		ctx := effects.NewEffContext(nil)
		ctx.Grant(effects.Capability{Name: "IO"})
		ctx.IOWriter = &out
		rt.GetEvaluator().SetEffContext(ctx)

		var runErr error
		stderr := captureStderr(t, func() {
			runErr = ExecuteModuleEntrypoint(rt, ModuleExecParams{
				Filename: entry, Iface: res.Interface, Modules: res.Modules, Entry: "main", ArgsJSON: "null",
				NoPrint: true, Quiet: true, MaxRecursionDepth: 32, PipelineResult: &res,
				BytecodeMode: true, StrictBytecode: false,
			})
		})
		if runErr == nil {
			t.Fatal("expected stack overflow error, got nil")
		}
		errMsg := filepath.ToSlash(runErr.Error())
		if strings.Contains(errMsg, "vm: vm:") {
			t.Fatalf("expected single 'vm: ' prefix on stack overflow, got: %s", errMsg)
		}
		if strings.Contains(stderr, "bytecode path unavailable") {
			t.Fatalf("stack overflow must NOT fall back to evaluator, got stderr: %s", stderr)
		}
		if gotCount := strings.Count(out.String(), "SIDE_EFFECT"); gotCount != 1 {
			t.Fatalf("expected SIDE_EFFECT printed once, got %d", gotCount)
		}
	})

	t.Run("pre-dispatch BC001 compile-ineligible module still falls back cleanly under --bytecode", func(t *testing.T) {
		root := t.TempDir()
		t.Chdir(root)
		if err := os.MkdirAll("s1", 0o755); err != nil {
			t.Fatal(err)
		}
		modOpt := `module s1/opt

export type Option = Algo(int)

export pure func unwrapAlgo(o: Option) -> int = match o {
  Algo(n) => n
}
`
		modMain := `module s1/main
import std/io (println)
import s1/opt (Option, Algo, unwrapAlgo)

export func main() -> () ! {IO} {
  println(show(unwrapAlgo(Algo(42))))
}
`
		if err := os.WriteFile(filepath.Join("s1", "opt.ail"), []byte(modOpt), 0o644); err != nil {
			t.Fatal(err)
		}
		entry := filepath.Join(root, "s1", "main.ail")
		if err := os.WriteFile(entry, []byte(modMain), 0o644); err != nil {
			t.Fatal(err)
		}
		res, err := pipeline.Run(pipeline.Config{
			Mode:    pipeline.ModeCheck,
			NoCache: true,
			InstEnv: types.LoadBuiltinInstances(),
			DictReg: types.NewDictionaryRegistry(),
		}, pipeline.Source{Filename: entry})
		if err != nil || len(res.Errors) > 0 {
			t.Fatalf("compile: %v %v", err, res.Errors)
		}

		var out bytes.Buffer
		rt := runtime.NewModuleRuntime(".")
		if res.DictReg != nil {
			rt.GetEvaluator().SetDictionaryRegistry(res.DictReg)
		}
		ctx := effects.NewEffContext(nil)
		ctx.Grant(effects.Capability{Name: "IO"})
		ctx.IOWriter = &out
		rt.GetEvaluator().SetEffContext(ctx)

		var runErr error
		stderr := captureStderr(t, func() {
			runErr = ExecuteModuleEntrypoint(rt, ModuleExecParams{
				Filename: entry, Iface: res.Interface, Modules: res.Modules, Entry: "main", ArgsJSON: "null",
				NoPrint: true, Quiet: true, PipelineResult: &res,
				BytecodeMode: true, StrictBytecode: false,
			})
		})
		if runErr != nil {
			t.Fatalf("expected clean evaluator fallback under --bytecode, got err: %v", runErr)
		}
		// The compile-ineligible entry is detected before VM dispatch and
		// routed to the evaluator with a BC001 coverage warning.
		if !strings.Contains(stderr, "BC001") {
			t.Fatalf("expected pre-dispatch BC001 fallback warning on stderr, got: %s", stderr)
		}
		if out.String() != "42\n" {
			t.Fatalf("expected fallback output \"42\\n\", got %q", out.String())
		}
	})
}

// TST-RC1-04
func TestContainerLambdaOperatorsParityOnBothEngines(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	src := `module container_ops
import std/io (println)
import std/array as A

pure func guardedInc(n: int) -> int
requires { n >= 0 }
ensures { result > n }
{
  let tup = (1, \x. x + 1);
  let arr = #[\x. x * 2];
  let base = { tag: "init", fn: \x. x };
  let upd = { base | fn: \x. x + 10 };
  let fromTuple = match tup { (_, f) => f(n) };
  let fArr = A.get(arr, 0);
  let fromArr = fArr(fromTuple);
  let guarded = match fromArr {
    v if v > 0 => upd.fn(v),
    _ => 0
  };
  guarded
}

export func main() -> () ! {IO} {
  println(show(guardedInc(20)))
}
`
	// (20 + 1) * 2 + 10 = 52
	// Full fixture (incl. `#[...]` array literal) on the evaluator and the
	// bridged VM: the bytecode compiler has no ArrayLit lowering yet, so the
	// array-bearing function runs evaluator-only via the bridge.
	full := filepath.Join(root, "container_ops.ail")
	for _, engine := range []string{"eval", "bytecode"} {
		if got := runOnEngine(t, full, src, engine); got != "52\n" {
			t.Fatalf("%s: stdout = %q, want %q", engine, got, "52\n")
		}
	}

	// Array-free variant (tuple, record update, match guard, contracts) must
	// also run natively under --strict-bytecode.
	strictSrc := strings.NewReplacer(
		"module container_ops\n", "module container_ops_strict\n",
		"import std/array as A\n", "",
		"  let arr = #[\\x. x * 2];\n", "",
		"  let fArr = A.get(arr, 0);\n", "  let fArr = \\x. x * 2;\n",
	).Replace(src)
	if strings.Contains(strictSrc, "#[") {
		t.Fatal("strict fixture still contains an array literal")
	}
	strictEntry := filepath.Join(root, "container_ops_strict.ail")
	if err := os.WriteFile(strictEntry, []byte(strictSrc), 0o644); err != nil {
		t.Fatal(err)
	}
	runOnEngines(t, strictEntry, "52\n", []string{"eval", "strict-bytecode"})
}
