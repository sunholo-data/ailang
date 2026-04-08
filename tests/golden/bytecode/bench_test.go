// Package bytecode_golden_test: M-BYTECODE-2D M5 benchmark harness.
//
// This file runs paired Go benchmarks comparing the bytecode VM path
// (pipeline → lower → bytecode compiler → vm.VM) against the tree-walking
// evaluator path (pipeline → runtime.ModuleRuntime → eval.CallFunction) on
// the same golden .ail files. The goal is to report a speedup ratio for
// the M-BYTECODE-VM Phase 2D sprint commit message.
//
// Compile cost is amortized outside b.N — we measure steady-state call-site
// throughput, not cold-start compile time. The VM is reused across
// iterations inside one b.Run (its Stack is reset at the end of each Run).
// Similarly, the evaluator is forked once per case (matching what
// runtime.CallEntrypoint does) and reused across iterations.
//
// Run with:
//
//	go test -bench=. -benchmem -run=^$ ./tests/golden/bytecode/
//
// To focus on the throughput cases (fib):
//
//	go test -bench=BenchmarkFib -benchmem -run=^$ ./tests/golden/bytecode/
package bytecode_golden_test

import (
	"path/filepath"
	"testing"

	"github.com/sunholo/ailang/internal/ast"
	"github.com/sunholo/ailang/internal/bytecode"
	"github.com/sunholo/ailang/internal/bytecode/compiler"
	"github.com/sunholo/ailang/internal/eval"
	"github.com/sunholo/ailang/internal/gen/lower"
	"github.com/sunholo/ailang/internal/gen/stmt"
	"github.com/sunholo/ailang/internal/pipeline"
	"github.com/sunholo/ailang/internal/runtime"
	"github.com/sunholo/ailang/internal/vm"
)

// benchCase describes one benchmarkable function call paired across both
// backends. args and evalArgs must be semantically equivalent — the harness
// does NOT auto-convert between the two value worlds so the benchmark
// itself is not charged with bridge cost.
type benchCase struct {
	name     string
	file     string // .ail file under tests/golden/codegen/
	fn       string
	args     []bytecode.Value
	evalArgs []eval.Value
}

func benchCases() []benchCase {
	return []benchCase{
		{
			name:     "Identity",
			file:     "functions.ail",
			fn:       "identity",
			args:     []bytecode.Value{bytecode.NewInt(42)},
			evalArgs: []eval.Value{&eval.IntValue{Value: 42}},
		},
		{
			name:     "AddInts",
			file:     "arithmetic.ail",
			fn:       "addInts",
			args:     []bytecode.Value{bytecode.NewInt(2), bytecode.NewInt(3)},
			evalArgs: []eval.Value{&eval.IntValue{Value: 2}, &eval.IntValue{Value: 3}},
		},
		{
			name:     "Factorial10",
			file:     "functions.ail",
			fn:       "factorial",
			args:     []bytecode.Value{bytecode.NewInt(10)},
			evalArgs: []eval.Value{&eval.IntValue{Value: 10}},
		},
		{
			name:     "Classify",
			file:     "if_else.ail",
			fn:       "classify",
			args:     []bytecode.Value{bytecode.NewInt(-1)},
			evalArgs: []eval.Value{&eval.IntValue{Value: -1}},
		},
		{
			name:     "Clamp",
			file:     "if_else.ail",
			fn:       "clamp",
			args:     []bytecode.Value{bytecode.NewInt(5), bytecode.NewInt(0), bytecode.NewInt(10)},
			evalArgs: []eval.Value{&eval.IntValue{Value: 5}, &eval.IntValue{Value: 0}, &eval.IntValue{Value: 10}},
		},
		{
			name:     "Fib20",
			file:     "fib.ail",
			fn:       "fib",
			args:     []bytecode.Value{bytecode.NewInt(20)},
			evalArgs: []eval.Value{&eval.IntValue{Value: 20}},
		},
		{
			name:     "Fib25",
			file:     "fib.ail",
			fn:       "fib",
			args:     []bytecode.Value{bytecode.NewInt(25)},
			evalArgs: []eval.Value{&eval.IntValue{Value: 25}},
		},
		{
			name:     "Fib30",
			file:     "fib.ail",
			fn:       "fib",
			args:     []bytecode.Value{bytecode.NewInt(30)},
			evalArgs: []eval.Value{&eval.IntValue{Value: 30}},
		},
	}
}

// benchFixture holds the compiled artifacts for one .ail file, shared
// across all cases that target it. This avoids re-running the pipeline
// and lower/compile passes inside the b.N loop.
type benchFixture struct {
	vmImage    *bytecode.BytecodeImage
	moduleInst *runtime.ModuleInstance
	moduleRt   *runtime.ModuleRuntime
}

// loadFixture runs the pipeline on the given file and returns both a
// compiled BytecodeImage (for the VM path) and a fully evaluated
// ModuleInstance (for the evaluator path). Fatal on error.
func loadFixture(b *testing.B, filename string) *benchFixture {
	b.Helper()
	abs, err := filepath.Abs(filepath.Join("..", "codegen", filename))
	if err != nil {
		b.Fatalf("abs %s: %v", filename, err)
	}
	data, err := readFileImpl(abs)
	if err != nil {
		b.Fatalf("read %s: %v", filename, err)
	}

	res, err := pipeline.Run(pipeline.Config{
		Mode:         pipeline.ModeCheck,
		RelaxModules: true,
	}, pipeline.Source{Filename: abs, Code: string(data)})
	if err != nil {
		b.Fatalf("pipeline %s: %v", filename, err)
	}
	if len(res.Errors) > 0 {
		b.Fatalf("pipeline errors %s: %v", filename, res.Errors)
	}

	// --- Bytecode side: lower + compile ------------------------------------
	prog := &stmt.Program{Package: "bench"}
	seenTypes := map[string]bool{}
	if res.Artifacts.AST != nil {
		for _, decl := range res.Artifacts.AST.Decls {
			td, ok := decl.(*ast.TypeDecl)
			if !ok || seenTypes[td.Name] {
				continue
			}
			seenTypes[td.Name] = true
			prog.TypeDecls = append(prog.TypeDecls, lower.LowerTypeDecl(td))
		}
	}
	fileProg, err := lower.LowerProgram(res.Artifacts.Core, res.Artifacts.CoreTI, res.Artifacts.AST, "bench")
	if err != nil {
		b.Fatalf("lower %s: %v", filename, err)
	}
	prog.FuncDecls = append(prog.FuncDecls, fileProg.FuncDecls...)
	img, err := compiler.Compile(prog)
	if err != nil {
		b.Fatalf("compile %s: %v", filename, err)
	}

	// --- Evaluator side: runtime + LoadAndEvaluate -------------------------
	rt := runtime.NewModuleRuntime(".")
	if res.DictReg != nil {
		rt.GetEvaluator().SetDictionaryRegistry(res.DictReg)
	}
	for path, loaded := range res.Modules {
		rt.PreloadModule(path, loaded)
	}
	inst, err := rt.LoadAndEvaluate(res.Interface.Module)
	if err != nil {
		b.Fatalf("LoadAndEvaluate %s: %v", filename, err)
	}

	return &benchFixture{
		vmImage:    img,
		moduleInst: inst,
		moduleRt:   rt,
	}
}

// resolveEvalFn looks up a function by name in the instance's export table.
// The benchmark dispatches directly through evaluator.CallValueN without
// forking or resolver reconfiguration — the goal is to measure call-site
// throughput of the existing fn value, not CallEntrypoint setup cost.
func resolveEvalFn(b *testing.B, inst *runtime.ModuleInstance, name string) eval.Value {
	b.Helper()
	fn, err := inst.GetExport(name)
	if err != nil {
		b.Fatalf("GetExport %s: %v", name, err)
	}
	return fn
}

// BenchmarkBytecodeVsEvaluator runs every bench case on both backends so
// the output is `benchstat`-friendly: each case produces a Bytecode/<case>
// line and an Evaluator/<case> line with identical units, letting the
// caller compute the ratio directly.
func BenchmarkBytecodeVsEvaluator(b *testing.B) {
	// Cache fixtures per file so we don't recompile for every case.
	fixtures := map[string]*benchFixture{}
	getFixture := func(filename string) *benchFixture {
		if fx, ok := fixtures[filename]; ok {
			return fx
		}
		fx := loadFixture(b, filename)
		fixtures[filename] = fx
		return fx
	}

	cases := benchCases()

	b.Run("Bytecode", func(b *testing.B) {
		for _, c := range cases {
			c := c
			fx := getFixture(c.file)
			proto := findProto(fx.vmImage, c.fn)
			if proto == nil {
				b.Fatalf("bytecode proto %q not found in %s", c.fn, c.file)
			}

			// The lower pass adds a synthetic Unit to nullary fns. We don't
			// have any nullary cases here but preserve the pattern for
			// future additions.
			args := c.args
			if int(proto.NumParams) == len(args)+1 {
				padded := make([]bytecode.Value, 0, len(args)+1)
				padded = append(padded, bytecode.Unit())
				padded = append(padded, args...)
				args = padded
			}
			if int(proto.NumParams) != len(args) {
				b.Fatalf("case %q arity mismatch: proto=%d args=%d", c.name, proto.NumParams, len(args))
			}

			machine := vm.NewVM(fx.vmImage)
			b.Run(c.name, func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					_, err := machine.Run(proto, args)
					if err != nil {
						b.Fatalf("vm run %s: %v", c.name, err)
					}
				}
			})
		}
	})

	b.Run("Evaluator", func(b *testing.B) {
		for _, c := range cases {
			c := c
			fx := getFixture(c.file)
			fnVal := resolveEvalFn(b, fx.moduleInst, c.fn)
			evaluator := fx.moduleRt.GetEvaluator()

			b.Run(c.name, func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					_, err := evaluator.CallValueN(fnVal, c.evalArgs)
					if err != nil {
						b.Fatalf("eval call %s: %v", c.name, err)
					}
				}
			})
		}
	})
}

// BenchmarkFib is a focused throughput benchmark for the two canonical
// recursion cases. Separated from BenchmarkBytecodeVsEvaluator so the
// sprint-level speedup ratio can be read off a targeted run.
func BenchmarkFib(b *testing.B) {
	fx := loadFixture(b, "fib.ail")
	proto := findProto(fx.vmImage, "fib")
	if proto == nil {
		b.Fatal("fib proto missing")
	}
	fnVal := resolveEvalFn(b, fx.moduleInst, "fib")
	evaluator := fx.moduleRt.GetEvaluator()
	machine := vm.NewVM(fx.vmImage)

	for _, n := range []int{20, 25, 30} {
		n := n
		vmArgs := []bytecode.Value{bytecode.NewInt(int64(n))}
		evalArgs := []eval.Value{&eval.IntValue{Value: n}}

		b.Run("Bytecode/Fib"+itoa(n), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				if _, err := machine.Run(proto, vmArgs); err != nil {
					b.Fatal(err)
				}
			}
		})
		b.Run("Evaluator/Fib"+itoa(n), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				if _, err := evaluator.CallValueN(fnVal, evalArgs); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// itoa is a tiny local helper to keep fib benchmark names free of imports.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
