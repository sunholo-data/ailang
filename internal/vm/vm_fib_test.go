package vm

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/bytecode"
)

// TestVM_Fib25 is the Phase 2B acceptance gate from M-BYTECODE-VM §10.
//
// Hand-assembled bytecode for the recursive fibonacci:
//
//	fib(n) = if n < 2 then n else fib(n-1) + fib(n-2)
//
// fib(25) = 75025. The test exercises CALL/RETURN, recursive self-reference
// via CLOSURE, conditional branching with JUMP_IF_FALSE, integer arithmetic,
// and frame-based register isolation.
//
// Register layout for fib:
//
//	r0  = n           (parameter)
//	r1  = constant 2  (lives through the function)
//	r2  = cond (n < 2)
//	r3  = constant 1
//	r4  = n - 1
//	r5  = closure(fib) for first call → result of fib(n-1)
//	r6  = arg slot for first call (set to r4 via MOVE)
//	     reused for n-2 in the second branch
//	r7  = closure(fib) for second call → result of fib(n-2)
//	r8  = arg slot for second call (set to r6)
//	r9  = sum
func TestVM_Fib25(t *testing.T) {
	img := bytecode.NewImage()

	fib := &bytecode.FuncPrototype{
		Name:      "fib",
		NumRegs:   10,
		NumParams: 1,
	}
	addConstants(img, fib, bytecode.NewInt(2), bytecode.NewInt(1))
	fibIdx := img.AddPrototype(fib)
	fib.NestedProtos = []int{fibIdx}

	fib.Instructions = []bytecode.Instruction{
		bytecode.EncodeABx(bytecode.OpLoadConst, 1, 0),    // 0:  r1 = 2
		bytecode.EncodeABC(bytecode.OpLt, 2, 0, 1),        // 1:  r2 = n < 2
		bytecode.EncodeASBx(bytecode.OpJumpIfFalse, 2, 1), // 2:  if !cond jump +1 → inst 4
		bytecode.EncodeABC(bytecode.OpReturn, 0, 0, 0),    // 3:  return n   (base case)
		bytecode.EncodeABx(bytecode.OpLoadConst, 3, 1),    // 4:  r3 = 1
		bytecode.EncodeABC(bytecode.OpSub, 4, 0, 3),       // 5:  r4 = n - 1
		bytecode.EncodeABx(bytecode.OpClosure, 5, 0),      // 6:  r5 = closure(fib)
		bytecode.EncodeABC(bytecode.OpMove, 6, 4, 0),      // 7:  r6 = r4 (arg for first call)
		bytecode.EncodeABC(bytecode.OpCall, 5, 1, 1),      // 8:  r5 = fib(r6) = fib(n-1)
		bytecode.EncodeABC(bytecode.OpSub, 6, 0, 1),       // 9:  r6 = n - 2 (reuse r1 = 2)
		bytecode.EncodeABx(bytecode.OpClosure, 7, 0),      // 10: r7 = closure(fib)
		bytecode.EncodeABC(bytecode.OpMove, 8, 6, 0),      // 11: r8 = r6 (arg for second call)
		bytecode.EncodeABC(bytecode.OpCall, 7, 1, 1),      // 12: r7 = fib(r8) = fib(n-2)
		bytecode.EncodeABC(bytecode.OpAdd, 9, 5, 7),       // 13: r9 = fib(n-1) + fib(n-2)
		bytecode.EncodeABC(bytecode.OpReturn, 9, 0, 0),    // 14: return r9
	}

	if err := img.SetEntryPoint(fibIdx); err != nil {
		t.Fatalf("SetEntryPoint: %v", err)
	}
	if err := img.Validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}

	// Document the bytecode in test output for the design doc audit trail.
	t.Log("fib bytecode listing:")
	for i, inst := range fib.Instructions {
		t.Logf("  %2d: %s", i, inst)
	}

	vm := NewVM(img)
	got, err := vm.Run(fib, []bytecode.Value{bytecode.NewInt(25)})
	if err != nil {
		t.Fatalf("Run fib(25): %v", err)
	}
	if got.Tag != bytecode.TagInt {
		t.Fatalf("got tag %v, want Int", got.Tag)
	}
	const want = int64(75025)
	if got.Int != want {
		t.Errorf("fib(25) = %d, want %d", got.Int, want)
	}
}

// TestVM_Fib25_SmallerInputs spot-checks the base-case path and a few small
// inputs to ensure the recursive path isn't accidentally returning a constant.
func TestVM_Fib25_SmallerInputs(t *testing.T) {
	img := bytecode.NewImage()
	fib := &bytecode.FuncPrototype{
		Name:      "fib",
		NumRegs:   10,
		NumParams: 1,
	}
	addConstants(img, fib, bytecode.NewInt(2), bytecode.NewInt(1))
	fibIdx := img.AddPrototype(fib)
	fib.NestedProtos = []int{fibIdx}
	fib.Instructions = []bytecode.Instruction{
		bytecode.EncodeABx(bytecode.OpLoadConst, 1, 0),
		bytecode.EncodeABC(bytecode.OpLt, 2, 0, 1),
		bytecode.EncodeASBx(bytecode.OpJumpIfFalse, 2, 1),
		bytecode.EncodeABC(bytecode.OpReturn, 0, 0, 0),
		bytecode.EncodeABx(bytecode.OpLoadConst, 3, 1),
		bytecode.EncodeABC(bytecode.OpSub, 4, 0, 3),
		bytecode.EncodeABx(bytecode.OpClosure, 5, 0),
		bytecode.EncodeABC(bytecode.OpMove, 6, 4, 0),
		bytecode.EncodeABC(bytecode.OpCall, 5, 1, 1),
		bytecode.EncodeABC(bytecode.OpSub, 6, 0, 1),
		bytecode.EncodeABx(bytecode.OpClosure, 7, 0),
		bytecode.EncodeABC(bytecode.OpMove, 8, 6, 0),
		bytecode.EncodeABC(bytecode.OpCall, 7, 1, 1),
		bytecode.EncodeABC(bytecode.OpAdd, 9, 5, 7),
		bytecode.EncodeABC(bytecode.OpReturn, 9, 0, 0),
	}
	_ = img.SetEntryPoint(fibIdx)
	if err := img.Validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}

	cases := []struct {
		n, want int64
	}{
		{0, 0}, {1, 1}, {2, 1}, {3, 2}, {4, 3}, {5, 5},
		{10, 55}, {15, 610}, {20, 6765},
	}
	for _, tc := range cases {
		vm := NewVM(img)
		got, err := vm.Run(fib, []bytecode.Value{bytecode.NewInt(tc.n)})
		if err != nil {
			t.Errorf("fib(%d): %v", tc.n, err)
			continue
		}
		if got.Int != tc.want {
			t.Errorf("fib(%d) = %d, want %d", tc.n, got.Int, tc.want)
		}
	}
}

// TestVM_FibTailCallAccumulator exercises TAIL_CALL with the linear accumulator
// formulation:
//
//	fibTail(n, a, b) = if n == 0 then a else fibTail(n-1, b, a+b)
//
// fibTail(25, 0, 1) = 75025. The bounded-stack assertion verifies TCO is
// actually working — without it, fibTail(25, ...) would push 25 frames and
// fibTail(10000, ...) would explode. We use a tiny MaxStack ceiling to make
// any frame growth fatal.
//
// Register layout for fibTail:
//
//	r0  = n           (parameter)
//	r1  = a           (parameter)
//	r2  = b           (parameter)
//	r3  = constant 0
//	r4  = cond (n == 0)
//	r5  = constant 1
//	r6  = n - 1
//	r7  = a + b
//	r8  = closure(fibTail) for tail call
//	r9  = arg slot 1: n - 1
//	r10 = arg slot 2: b (becomes new a)
//	r11 = arg slot 3: a + b (becomes new b)
func TestVM_FibTailCallAccumulator(t *testing.T) {
	img := bytecode.NewImage()
	fibTail := &bytecode.FuncPrototype{
		Name:      "fibTail",
		NumRegs:   12,
		NumParams: 3,
	}
	addConstants(img, fibTail, bytecode.NewInt(0), bytecode.NewInt(1))
	fibIdx := img.AddPrototype(fibTail)
	fibTail.NestedProtos = []int{fibIdx}

	fibTail.Instructions = []bytecode.Instruction{
		bytecode.EncodeABx(bytecode.OpLoadConst, 3, 0),    // 0:  r3 = 0
		bytecode.EncodeABC(bytecode.OpEq, 4, 0, 3),        // 1:  r4 = (n == 0)
		bytecode.EncodeASBx(bytecode.OpJumpIfFalse, 4, 1), // 2:  if !cond jump +1 → inst 4
		bytecode.EncodeABC(bytecode.OpReturn, 1, 0, 0),    // 3:  return a
		bytecode.EncodeABx(bytecode.OpLoadConst, 5, 1),    // 4:  r5 = 1
		bytecode.EncodeABC(bytecode.OpSub, 6, 0, 5),       // 5:  r6 = n - 1
		bytecode.EncodeABC(bytecode.OpAdd, 7, 1, 2),       // 6:  r7 = a + b
		bytecode.EncodeABx(bytecode.OpClosure, 8, 0),      // 7:  r8 = closure(self)
		bytecode.EncodeABC(bytecode.OpMove, 9, 6, 0),      // 8:  r9  = r6 (n-1)
		bytecode.EncodeABC(bytecode.OpMove, 10, 2, 0),     // 9:  r10 = b
		bytecode.EncodeABC(bytecode.OpMove, 11, 7, 0),     // 10: r11 = a+b
		bytecode.EncodeABC(bytecode.OpTailCall, 8, 3, 0),  // 11: tail-call r8(r9, r10, r11)
	}

	_ = img.SetEntryPoint(fibIdx)
	if err := img.Validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}

	// Tiny stack ceiling. If TCO is broken, even fibTail(25, ...) will exceed.
	vm := NewVM(img)
	vm.MaxStack = 5

	args := []bytecode.Value{
		bytecode.NewInt(25),
		bytecode.NewInt(0),
		bytecode.NewInt(1),
	}
	got, err := vm.Run(fibTail, args)
	if err != nil {
		t.Fatalf("fibTail(25, 0, 1): %v (final stack depth: %d)", err, len(vm.Stack))
	}
	if got.Int != 75025 {
		t.Errorf("fibTail(25, 0, 1) = %d, want 75025", got.Int)
	}

	// Additional stress: 10_000 iterations to be unambiguous about TCO.
	got2, err := vm.Run(fibTail, []bytecode.Value{
		bytecode.NewInt(10000),
		bytecode.NewInt(0),
		bytecode.NewInt(1),
	})
	if err != nil {
		t.Fatalf("fibTail(10000, ...): %v", err)
	}
	// We don't check the value (it overflows int64 long before n=10000) — we
	// only care that the call returned without stack overflow. The result is
	// a defined int64 wrap-around.
	_ = got2
}
