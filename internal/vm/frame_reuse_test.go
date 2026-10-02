package vm

import (
	"errors"
	"sync"
	"testing"

	"github.com/sunholo-data/ailang/internal/bytecode"
	ailerrors "github.com/sunholo-data/ailang/internal/errors"
)

// M-FOLDL-CONS-COST-MODEL Phase 3 (#1501): HOF builtins re-enter the VM
// through CallClosure once per element. These tests pin that the per-element
// callback allocates nothing (frames are recycled), and that recycling is
// safe under nesting, mid-call errors, stack overflow and concurrent VMs.

// hofFixture is a hand-assembled image with the closures the tests need.
type hofFixture struct {
	img    *bytecode.BytecodeImage
	add    bytecode.Value // \acc x. acc + x
	div    bytecode.Value // \acc x. acc + 100 / x   (faults on x == 0)
	nested bytecode.Value // \acc xs. acc + foldl(add, 0, xs)  -- re-enters a HOF
	tail   bytecode.Value // \acc x. add(acc, x) as a TAIL_CALL from an 8-register frame
	call   bytecode.Value // \acc x. add(acc, x) as a plain CALL (callee frame has a Caller)
	rec    *bytecode.FuncPrototype
}

const foldlHOFIndex = 2 // position of __list_foldl in HOFBuiltinTable

func newHOFFixture(t testing.TB) *hofFixture {
	t.Helper()
	if bytecode.HOFBuiltinNames[foldlHOFIndex] != "__list_foldl" {
		t.Fatalf("HOF table moved: index %d is %q", foldlHOFIndex, bytecode.HOFBuiltinNames[foldlHOFIndex])
	}
	img := bytecode.NewImage()

	add := &bytecode.FuncPrototype{Name: "add", NumParams: 2, NumRegs: 3}
	add.Instructions = []bytecode.Instruction{
		bytecode.EncodeABC(bytecode.OpAdd, 2, 0, 1),
		bytecode.EncodeABC(bytecode.OpReturn, 2, 0, 0),
	}
	addIdx := img.AddPrototype(add)

	div := &bytecode.FuncPrototype{Name: "div", NumParams: 2, NumRegs: 5}
	addConstants(img, div, bytecode.NewInt(100))
	div.Instructions = []bytecode.Instruction{
		bytecode.EncodeABx(bytecode.OpLoadConst, 2, 0),
		bytecode.EncodeABC(bytecode.OpDiv, 3, 2, 1),
		bytecode.EncodeABC(bytecode.OpAdd, 4, 0, 3),
		bytecode.EncodeABC(bytecode.OpReturn, 4, 0, 0),
	}
	img.AddPrototype(div)

	// r0=acc r1=xs r2=captured add; r3 = foldl(r4=add, r5=0, r6=xs); r7 = acc + r3
	nested := &bytecode.FuncPrototype{Name: "nested", NumParams: 2, NumCaptures: 1, NumRegs: 8}
	addConstants(img, nested, bytecode.NewInt(0))
	nested.Instructions = []bytecode.Instruction{
		bytecode.EncodeABC(bytecode.OpMove, 4, 2, 0),
		bytecode.EncodeABx(bytecode.OpLoadConst, 5, 0),
		bytecode.EncodeABC(bytecode.OpMove, 6, 1, 0),
		bytecode.EncodeABC(bytecode.OpBuiltinCallHOF, 3, foldlHOFIndex, 3),
		bytecode.EncodeABC(bytecode.OpAdd, 7, 0, 3),
		bytecode.EncodeABC(bytecode.OpReturn, 7, 0, 0),
	}
	img.AddPrototype(nested)

	// Fills r5..r7, then tail-calls the 3-register add: reuseFor shrinks the
	// slab to 3 and leaves r5..r7 past len, which releaseFrame must clear.
	tail := &bytecode.FuncPrototype{Name: "tail", NumParams: 2, NumRegs: 8, NestedProtos: []int{addIdx}}
	addConstants(img, tail, bytecode.NewString("stale"))
	tail.Instructions = []bytecode.Instruction{
		bytecode.EncodeABx(bytecode.OpLoadConst, 5, 0),
		bytecode.EncodeABx(bytecode.OpLoadConst, 6, 0),
		bytecode.EncodeABx(bytecode.OpLoadConst, 7, 0),
		bytecode.EncodeABx(bytecode.OpClosure, 2, 0),
		bytecode.EncodeABC(bytecode.OpMove, 3, 0, 0),
		bytecode.EncodeABC(bytecode.OpMove, 4, 1, 0),
		bytecode.EncodeABC(bytecode.OpTailCall, 2, 2, 0),
	}
	img.AddPrototype(tail)

	call := &bytecode.FuncPrototype{Name: "call", NumParams: 2, NumRegs: 5, NestedProtos: []int{addIdx}}
	call.Instructions = []bytecode.Instruction{
		bytecode.EncodeABx(bytecode.OpClosure, 2, 0),
		bytecode.EncodeABC(bytecode.OpMove, 3, 0, 0),
		bytecode.EncodeABC(bytecode.OpMove, 4, 1, 0),
		bytecode.EncodeABC(bytecode.OpCall, 2, 2, 1),
		bytecode.EncodeABC(bytecode.OpReturn, 2, 0, 0),
	}
	img.AddPrototype(call)

	// rec() = rec() -- unbounded non-tail recursion.
	rec := &bytecode.FuncPrototype{Name: "rec", NumRegs: 2}
	recIdx := img.AddPrototype(rec)
	rec.NestedProtos = []int{recIdx}
	rec.Instructions = []bytecode.Instruction{
		bytecode.EncodeABx(bytecode.OpClosure, 0, 0),
		bytecode.EncodeABC(bytecode.OpCall, 0, 0, 1),
		bytecode.EncodeABC(bytecode.OpReturn, 0, 0, 0),
	}

	if err := img.SetEntryPoint(0); err != nil {
		t.Fatalf("SetEntryPoint: %v", err)
	}
	if err := img.Validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}
	addC := bytecode.NewClosure(add, nil)
	return &hofFixture{
		img:    img,
		add:    addC,
		div:    bytecode.NewClosure(div, nil),
		nested: bytecode.NewClosure(nested, []bytecode.Value{addC}),
		tail:   bytecode.NewClosure(tail, nil),
		call:   bytecode.NewClosure(call, nil),
		rec:    rec,
	}
}

func intList(from, to int) bytecode.Value {
	elems := make([]bytecode.Value, 0, to-from)
	for i := from; i < to; i++ {
		elems = append(elems, bytecode.NewInt(int64(i)))
	}
	return bytecode.NewList(elems)
}

func foldl(t testing.TB, vm *VM, fn, init, list bytecode.Value) (bytecode.Value, error) {
	t.Helper()
	return hofBuiltinListFoldl(vm, []bytecode.Value{fn, init, list})
}

// TestHOFCallbackAllocatesNothingPerElement: an O(1)-step fold must cost O(1)
// allocations per *call*, not per element. Before frame recycling, each
// element allocated a Frame, its register slab and the argument slice.
func TestHOFCallbackAllocatesNothingPerElement(t *testing.T) {
	fx := newHOFFixture(t)
	vm := NewVM(fx.img)
	const n = 1000
	list := intList(0, n)
	args := []bytecode.Value{fx.add, bytecode.NewInt(0), list}

	got, err := hofBuiltinListFoldl(vm, args)
	if err != nil || got.Int != n*(n-1)/2 {
		t.Fatalf("foldl = %v, %v; want %d", got, err, n*(n-1)/2)
	}
	allocs := testing.AllocsPerRun(20, func() {
		if _, err := hofBuiltinListFoldl(vm, args); err != nil {
			t.Fatal(err)
		}
	})
	if perElem := allocs / n; perElem > 0.01 {
		t.Errorf("foldl callback allocates %.2f objects per element (%.0f per call of %d); want ~0", perElem, allocs, n)
	}

	mapArgs := []bytecode.Value{bytecode.NewClosure(&bytecode.FuncPrototype{
		Name: "inc", NumParams: 1, NumRegs: 1,
		Instructions: []bytecode.Instruction{bytecode.EncodeABC(bytecode.OpReturn, 0, 0, 0)},
	}, nil), list}
	allocs = testing.AllocsPerRun(20, func() {
		if _, err := hofBuiltinListMap(vm, mapArgs); err != nil {
			t.Fatal(err)
		}
	})
	if perElem := allocs / n; perElem > 0.01 {
		t.Errorf("map callback allocates %.2f objects per element (%.0f per call of %d); want ~0", perElem, allocs, n)
	}
}

// TestFrameReuseNestedHOF: a callback that itself runs a HOF builtin holds
// its own frame live while the inner fold acquires and releases frames.
func TestFrameReuseNestedHOF(t *testing.T) {
	fx := newHOFFixture(t)
	vm := NewVM(fx.img)
	rows := []bytecode.Value{intList(0, 10), intList(10, 20), intList(0, 0), intList(5, 105)}
	want := int64(45 + 145 + 0 + 5450)
	for i := 0; i < 3; i++ {
		got, err := foldl(t, vm, fx.nested, bytecode.NewInt(0), bytecode.NewList(rows))
		if err != nil || got.Int != want {
			t.Fatalf("round %d: nested foldl = %v, %v; want %d", i, got, err, want)
		}
		if len(vm.Stack) != 0 {
			t.Fatalf("round %d: %d frames left on the stack", i, len(vm.Stack))
		}
	}
	assertPoolClean(t, vm)
}

// TestFrameReuseAfterCallAndTailCall: frames pushed by CALL inside a callback
// (which link to a caller) and frames shortened by a TAIL_CALL to a smaller
// callee are recycled too. Neither may carry state into the pool: a kept
// Caller link pins a dead frame chain, and registers left past a shortened
// slab would be visible to the next, larger callee.
func TestFrameReuseAfterCallAndTailCall(t *testing.T) {
	fx := newHOFFixture(t)
	for _, tc := range []struct {
		name string
		fn   bytecode.Value
	}{{"call", fx.call}, {"tail_call", fx.tail}} {
		t.Run(tc.name, func(t *testing.T) {
			vm := NewVM(fx.img)
			got, err := foldl(t, vm, tc.fn, bytecode.NewInt(0), intList(0, 10))
			if err != nil || got.Int != 45 {
				t.Fatalf("foldl = %v, %v; want 45", got, err)
			}
			if len(vm.framePool) == 0 {
				t.Fatal("no frame was recycled")
			}
			assertPoolClean(t, vm)
		})
	}
}

// TestFrameReuseAfterCallbackError: a fault inside a callback must not leave
// the VM in a state that corrupts later calls (stale frames, stack depth).
func TestFrameReuseAfterCallbackError(t *testing.T) {
	fx := newHOFFixture(t)
	vm := NewVM(fx.img)
	vm.MaxStack = 8
	bad := bytecode.NewList([]bytecode.Value{bytecode.NewInt(1), bytecode.NewInt(0), bytecode.NewInt(4)})
	good := bytecode.NewList([]bytecode.Value{bytecode.NewInt(1), bytecode.NewInt(2), bytecode.NewInt(4)})
	// More failures than MaxStack: a frame leaked per error would overflow.
	for i := 0; i < 20; i++ {
		_, err := foldl(t, vm, fx.div, bytecode.NewInt(0), bad)
		var dz *ailerrors.DivByZeroError
		if !errors.As(err, &dz) {
			t.Fatalf("round %d: want DivByZeroError, got %v", i, err)
		}
		if len(vm.Stack) != 0 {
			t.Fatalf("round %d: %d frames left on the stack after a callback error", i, len(vm.Stack))
		}
	}
	got, err := foldl(t, vm, fx.div, bytecode.NewInt(0), good)
	if err != nil || got.Int != 100+50+25 {
		t.Fatalf("after errors: foldl = %v, %v; want 175", got, err)
	}
	assertPoolClean(t, vm)
}

// TestFrameReuseAfterStackOverflow: overflow unwinds through Run; the same VM
// must then run deep-but-legal recursion and HOF callbacks correctly.
func TestFrameReuseAfterStackOverflow(t *testing.T) {
	fx := newHOFFixture(t)
	vm := NewVM(fx.img)
	vm.MaxStack = 50
	for i := 0; i < 3; i++ {
		if _, err := vm.Run(fx.rec, nil); !errors.Is(err, ErrStackOverflow) {
			t.Fatalf("round %d: want ErrStackOverflow, got %v", i, err)
		}
		got, err := foldl(t, vm, fx.nested, bytecode.NewInt(1), bytecode.NewList([]bytecode.Value{intList(0, 4)}))
		if err != nil || got.Int != 7 {
			t.Fatalf("round %d: foldl after overflow = %v, %v; want 7", i, got, err)
		}
	}
	if len(vm.framePool) > vm.MaxStack {
		t.Errorf("frame pool holds %d frames, more than MaxStack %d", len(vm.framePool), vm.MaxStack)
	}
	assertPoolClean(t, vm)
}

// TestFrameReuseConcurrentVMs: VMs sharing one image on separate goroutines
// must not share frames. Run under -race.
func TestFrameReuseConcurrentVMs(t *testing.T) {
	fx := newHOFFixture(t)
	rows := bytecode.NewList([]bytecode.Value{intList(0, 50), intList(50, 100)})
	var wg sync.WaitGroup
	errs := make(chan error, 8)
	for g := 0; g < 8; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			vm := NewVM(fx.img)
			for i := 0; i < 200; i++ {
				got, err := hofBuiltinListFoldl(vm, []bytecode.Value{fx.nested, bytecode.NewInt(0), rows})
				if err != nil {
					errs <- err
					return
				}
				if got.Int != 4950 {
					errs <- errors.New("wrong nested-fold result under concurrency")
					return
				}
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}
}

// assertPoolClean: recycled frames must not pin values (a pooled register
// holding a large list would keep it alive) or link to a caller.
func assertPoolClean(t *testing.T, vm *VM) {
	t.Helper()
	for i, f := range vm.framePool {
		if f.Caller != nil || f.Proto != nil {
			t.Errorf("pooled frame %d still links proto/caller", i)
		}
		for r, v := range f.Regs[:cap(f.Regs)] {
			if v != (bytecode.Value{}) {
				t.Errorf("pooled frame %d register %d not cleared: %v", i, r, v)
			}
		}
	}
}

// BenchmarkHOFFoldlCallback drives __list_foldl with an O(1) closure over
// 1,000 elements: the per-callback cost of CallClosure in isolation.
func BenchmarkHOFFoldlCallback(b *testing.B) {
	fx := newHOFFixture(b)
	vm := NewVM(fx.img)
	args := []bytecode.Value{fx.add, bytecode.NewInt(0), intList(0, 1000)}
	b.ReportAllocs()
	for b.Loop() {
		if _, err := hofBuiltinListFoldl(vm, args); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkHOFFoldlNested: callbacks that re-enter a HOF (100 rows of 10).
func BenchmarkHOFFoldlNested(b *testing.B) {
	fx := newHOFFixture(b)
	vm := NewVM(fx.img)
	rows := make([]bytecode.Value, 100)
	for i := range rows {
		rows[i] = intList(0, 10)
	}
	args := []bytecode.Value{fx.nested, bytecode.NewInt(0), bytecode.NewList(rows)}
	b.ReportAllocs()
	for b.Loop() {
		if _, err := hofBuiltinListFoldl(vm, args); err != nil {
			b.Fatal(err)
		}
	}
}
