package runner

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sunholo-data/ailang/internal/bytecode"
	"github.com/sunholo-data/ailang/internal/pipeline"
	"github.com/sunholo-data/ailang/internal/vm"
)

// VM execution benchmarks (M-VM-QUICK-WINS Phase 0). Each workload is
// compiled once; the timed loop runs only the VM, on a fresh VM per
// iteration as `ailang run --bytecode` would. tools/perf_sweep.sh banks
// their ns/op and B/op as the vm_* metrics.

const vmBenchSource = `module bench/vmbench

import std/list (foldl, map, range)

export func fib(n: int) -> int =
  if n <= 1 then n
  else fib(n - 1) + fib(n - 2)

export func loop(i: int, n: int, acc: int) -> int =
  if i >= n then acc
  else loop(i + 1, n, acc + (i * i) % 7)

export func tailLoop(n: int) -> int = loop(0, n, 0)

export func mapFold(n: int) -> int =
  foldl(\acc x. acc + x, 0, map(\x. x * 2, range(0, n)))
`

type vmBenchWorkload struct {
	img   *bytecode.BytecodeImage
	proto *bytecode.FuncPrototype
	arg   bytecode.Value
	want  int64
}

func compileVMBench(tb testing.TB, entry string, arg int64, want int64) vmBenchWorkload {
	tb.Helper()
	// The file sits at bench/vmbench.ail under the working directory so its
	// module path matches and no MOD010 warning is printed per run.
	dir := tb.TempDir()
	tb.Chdir(dir)
	file := filepath.Join("bench", "vmbench.ail")
	if err := os.MkdirAll(filepath.Join(dir, "bench"), 0700); err != nil {
		tb.Fatal(err)
	}
	if err := os.WriteFile(file, []byte(vmBenchSource), 0600); err != nil {
		tb.Fatal(err)
	}
	res, err := pipeline.Run(pipeline.Config{Mode: pipeline.ModeCheck, TransientRoot: true},
		pipeline.Source{Filename: file, Code: vmBenchSource})
	if err != nil {
		tb.Fatal(err)
	}
	if len(res.Errors) > 0 {
		tb.Fatalf("pipeline errors: %v", res.Errors)
	}
	img, err := CompileBytecodeFromResult(res, "vmbench")
	if err != nil {
		tb.Fatal(err)
	}
	proto := FindEntryProto(img, entry)
	if proto == nil {
		tb.Fatalf("entry %q not in image", entry)
	}
	if proto.EvalOnly {
		tb.Fatalf("entry %q is evaluator-only (%s); the benchmark must measure the VM", entry, proto.EvalReason)
	}
	return vmBenchWorkload{img: img, proto: proto, arg: bytecode.NewInt(arg), want: want}
}

func (w vmBenchWorkload) run(tb testing.TB) {
	got, err := vm.NewVM(w.img).Run(w.proto, []bytecode.Value{w.arg})
	if err != nil {
		tb.Fatal(err)
	}
	if got.Tag != bytecode.TagInt || got.Int != w.want {
		tb.Fatalf("got %v, want %d", got, w.want)
	}
}

func tailLoopWant(n int64) int64 {
	var acc int64
	for i := int64(0); i < n; i++ {
		acc += (i * i) % 7
	}
	return acc
}

func mapFoldWant(n int64) int64 { return n * (n - 1) } // sum of 2x for x in [0, n)

func benchVM(b *testing.B, w vmBenchWorkload) {
	b.ReportAllocs()
	for b.Loop() {
		w.run(b)
	}
}

func BenchmarkVMFib25(b *testing.B) { benchVM(b, compileVMBench(b, "fib", 25, 75025)) }

func BenchmarkVMTailLoop1M(b *testing.B) {
	benchVM(b, compileVMBench(b, "tailLoop", 1_000_000, tailLoopWant(1_000_000)))
}

func BenchmarkVMMapFold200K(b *testing.B) {
	benchVM(b, compileVMBench(b, "mapFold", 200_000, mapFoldWant(200_000)))
}

// TestVMBenchWorkloadsRunNatively keeps the benchmarks honest in the normal
// test run: every workload compiles fully to the VM and returns its answer.
func TestVMBenchWorkloadsRunNatively(t *testing.T) {
	for _, w := range []vmBenchWorkload{
		compileVMBench(t, "fib", 15, 610),
		compileVMBench(t, "tailLoop", 1000, tailLoopWant(1000)),
		compileVMBench(t, "mapFold", 1000, mapFoldWant(1000)),
	} {
		w.run(t)
	}
}
