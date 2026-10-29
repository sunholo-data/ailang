package runner

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sunholo-data/ailang/internal/bytecode"
	"github.com/sunholo-data/ailang/internal/pipeline"
	"github.com/sunholo-data/ailang/internal/vm"
)

// M-VM-QUICK-WINS invariants: calling a known function and taking a tail
// call must not allocate per call. Allocation counts are compared across two
// problem sizes, so the assertion holds on any machine (it measures
// allocations, never RSS).

const vmQuickWinsSource = `module bench/quickwins

import std/list (foldl)

export func fib(n: int) -> int =
  if n <= 1 then n
  else fib(n - 1) + fib(n - 2)

export func loop(i: int, n: int, acc: int) -> int =
  if i >= n then acc
  else loop(i + 1, n, acc + (i * i) % 7)

export func tailLoop(n: int) -> int = loop(0, n, 0)

export func inc(x: int) -> int = x + 1

export func sameFn(u: int) -> (int -> int, int -> int) = (inc, inc)

export func mk(k: int) -> (int -> int) = \x. x + k

export func adders(k: int) -> (int -> int, int -> int) = (mk(k), mk(k))

-- A tail call whose arguments come from a HOF callback that itself runs a
-- tail-recursive loop: the VM re-enters run() while computing the args.
export func viaFold(i: int, n: int, acc: int) -> int =
  if i >= n then acc
  else viaFold(i + 1, n, foldl(\a x. a + loop(0, x, 0), acc, [i, i]))

export func viaFoldRun(n: int) -> int = viaFold(0, n, 0)
`

func compileQuickWins(t testing.TB) *bytecode.BytecodeImage {
	t.Helper()
	dir := t.TempDir()
	t.Chdir(dir)
	file := filepath.Join("bench", "quickwins.ail")
	if err := os.MkdirAll(filepath.Join(dir, "bench"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte(vmQuickWinsSource), 0600); err != nil {
		t.Fatal(err)
	}
	res, err := pipeline.Run(pipeline.Config{Mode: pipeline.ModeCheck, TransientRoot: true},
		pipeline.Source{Filename: file, Code: vmQuickWinsSource})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Errors) > 0 {
		t.Fatalf("pipeline errors: %v", res.Errors)
	}
	img, err := CompileBytecodeFromResult(res, "quickwins")
	if err != nil {
		t.Fatal(err)
	}
	return img
}

func runQuickWins(t testing.TB, img *bytecode.BytecodeImage, entry string, arg int64) bytecode.Value {
	t.Helper()
	proto := FindEntryProto(img, entry)
	if proto == nil || proto.EvalOnly {
		t.Fatalf("entry %q missing or evaluator-only", entry)
	}
	got, err := vm.NewVM(img).Run(proto, []bytecode.Value{bytecode.NewInt(arg)})
	if err != nil {
		t.Fatal(err)
	}
	return got
}

// allocGrowth returns allocs(big) - allocs(small) for one entry point.
func allocGrowth(t *testing.T, img *bytecode.BytecodeImage, entry string, small, big int64) float64 {
	t.Helper()
	a := testing.AllocsPerRun(5, func() { runQuickWins(t, img, entry, small) })
	b := testing.AllocsPerRun(5, func() { runQuickWins(t, img, entry, big) })
	return b - a
}

func TestVMKnownCallsDoNotAllocatePerCall(t *testing.T) {
	img := compileQuickWins(t)
	// fib(15) makes 1,973 calls and fib(20) makes 21,891. Before shared
	// zero-capture closures, every call allocated one closure.
	if growth := allocGrowth(t, img, "fib", 15, 20); growth > 64 {
		t.Fatalf("fib(20) allocates %.0f more than fib(15): a call to a known function allocates", growth)
	}
}

func TestVMTailCallsDoNotAllocatePerStep(t *testing.T) {
	img := compileQuickWins(t)
	if growth := allocGrowth(t, img, "tailLoop", 1_000, 100_000); growth > 64 {
		t.Fatalf("tailLoop(100000) allocates %.0f more than tailLoop(1000): a tail call allocates per step", growth)
	}
}

func TestVMZeroCaptureClosuresAreShared(t *testing.T) {
	img := compileQuickWins(t)
	same := runQuickWins(t, img, "sameFn", 0).Obj.(*bytecode.TupleObj).Elems
	if same[0].AsClosure() != same[1].AsClosure() {
		t.Fatal("two loads of the zero-capture function inc gave distinct closures")
	}
	// Two closures built by one capturing lambda must stay distinct objects.
	capt := runQuickWins(t, img, "adders", 5).Obj.(*bytecode.TupleObj).Elems
	a, b := capt[0].AsClosure(), capt[1].AsClosure()
	if a.Proto != b.Proto || len(a.Captures) == 0 {
		t.Fatalf("fixture drifted: want one capturing prototype, got %v / %v", a.Proto, b.Proto)
	}
	if a == b {
		t.Fatal("two capturing closures were shared")
	}
	if !capt[0].Equal(capt[1]) {
		t.Fatal("structurally identical capturing closures compare unequal")
	}
}

func TestVMTailCallWithReentrantArgs(t *testing.T) {
	img := compileQuickWins(t)
	var want int64
	for i := int64(0); i < 200; i++ {
		want += 2 * tailLoopWant(i)
	}
	if got := runQuickWins(t, img, "viaFoldRun", 200); got.Tag != bytecode.TagInt || got.Int != want {
		t.Fatalf("viaFold(200) = %v, want %d", got, want)
	}
}
