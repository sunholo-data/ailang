package vm

import (
	"errors"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/bytecode"
)

// --- Test helpers -----------------------------------------------------------

// runProto builds a single-prototype image, validates it, and runs it with
// the given args.
func runProto(t *testing.T, p *bytecode.FuncPrototype, args []bytecode.Value) (bytecode.Value, error) {
	t.Helper()
	img := bytecode.NewImage()
	img.AddPrototype(p)
	if err := img.SetEntryPoint(0); err != nil {
		t.Fatalf("SetEntryPoint: %v", err)
	}
	if err := img.Validate(); err != nil {
		t.Fatalf("image validate: %v", err)
	}
	vm := NewVM(img)
	return vm.Run(p, args)
}

// addConstants adds the given values to the image and returns local indices
// pointing to them. Convenience for tests that build a single prototype.
func addConstants(img *bytecode.BytecodeImage, p *bytecode.FuncPrototype, vs ...bytecode.Value) {
	for _, v := range vs {
		idx := img.AddConstant(v)
		p.Constants = append(p.Constants, idx)
	}
}

// --- Loads & moves ----------------------------------------------------------

func TestVM_LoadConstAndReturn(t *testing.T) {
	img := bytecode.NewImage()
	p := &bytecode.FuncPrototype{Name: "main", NumRegs: 1}
	addConstants(img, p, bytecode.NewInt(42))
	p.Instructions = []bytecode.Instruction{
		bytecode.EncodeABx(bytecode.OpLoadConst, 0, 0),
		bytecode.EncodeABC(bytecode.OpReturn, 0, 0, 0),
	}
	img.AddPrototype(p)
	_ = img.SetEntryPoint(0)
	if err := img.Validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}
	vm := NewVM(img)
	got, err := vm.Run(p, nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got.Tag != bytecode.TagInt || got.Int != 42 {
		t.Errorf("got %v, want Int(42)", got)
	}
}

func TestVM_LoadNil(t *testing.T) {
	p := &bytecode.FuncPrototype{
		Name:    "main",
		NumRegs: 1,
		Instructions: []bytecode.Instruction{
			bytecode.EncodeABC(bytecode.OpLoadNil, 0, 0, 0),
			bytecode.EncodeABC(bytecode.OpReturn, 0, 0, 0),
		},
	}
	got, err := runProto(t, p, nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got.Tag != bytecode.TagUnit {
		t.Errorf("got %v, want Unit", got)
	}
}

func TestVM_Move(t *testing.T) {
	img := bytecode.NewImage()
	p := &bytecode.FuncPrototype{Name: "main", NumRegs: 2}
	addConstants(img, p, bytecode.NewInt(7))
	p.Instructions = []bytecode.Instruction{
		bytecode.EncodeABx(bytecode.OpLoadConst, 0, 0),
		bytecode.EncodeABC(bytecode.OpMove, 1, 0, 0),
		bytecode.EncodeABC(bytecode.OpReturn, 1, 0, 0),
	}
	img.AddPrototype(p)
	_ = img.SetEntryPoint(0)
	got, err := NewVM(img).Run(p, nil)
	if err != nil || got.Int != 7 {
		t.Errorf("got %v, err=%v", got, err)
	}
}

func TestVM_LoadGlobal(t *testing.T) {
	img := bytecode.NewImage()
	img.Globals = []bytecode.Value{bytecode.NewInt(99)}
	p := &bytecode.FuncPrototype{
		Name:    "main",
		NumRegs: 1,
		Instructions: []bytecode.Instruction{
			bytecode.EncodeABx(bytecode.OpLoadGlobal, 0, 0),
			bytecode.EncodeABC(bytecode.OpReturn, 0, 0, 0),
		},
	}
	img.AddPrototype(p)
	_ = img.SetEntryPoint(0)
	got, err := NewVM(img).Run(p, nil)
	if err != nil || got.Int != 99 {
		t.Errorf("got %v, err=%v", got, err)
	}
}

// --- Arithmetic -------------------------------------------------------------

func TestVM_Arith_Int(t *testing.T) {
	cases := []struct {
		op   bytecode.OpCode
		a, b int64
		want int64
	}{
		{bytecode.OpAdd, 3, 4, 7},
		{bytecode.OpSub, 10, 3, 7},
		{bytecode.OpMul, 6, 7, 42},
		{bytecode.OpDiv, 20, 4, 5},
		{bytecode.OpMod, 23, 5, 3},
	}
	for _, tc := range cases {
		img := bytecode.NewImage()
		p := &bytecode.FuncPrototype{Name: "arith", NumRegs: 3}
		addConstants(img, p, bytecode.NewInt(tc.a), bytecode.NewInt(tc.b))
		p.Instructions = []bytecode.Instruction{
			bytecode.EncodeABx(bytecode.OpLoadConst, 0, 0),
			bytecode.EncodeABx(bytecode.OpLoadConst, 1, 1),
			bytecode.EncodeABC(tc.op, 2, 0, 1),
			bytecode.EncodeABC(bytecode.OpReturn, 2, 0, 0),
		}
		img.AddPrototype(p)
		_ = img.SetEntryPoint(0)
		got, err := NewVM(img).Run(p, nil)
		if err != nil {
			t.Fatalf("%v: %v", tc.op, err)
		}
		if got.Int != tc.want {
			t.Errorf("%v(%d, %d) = %d, want %d", tc.op, tc.a, tc.b, got.Int, tc.want)
		}
	}
}

func TestVM_Neg(t *testing.T) {
	img := bytecode.NewImage()
	p := &bytecode.FuncPrototype{Name: "neg", NumRegs: 2}
	addConstants(img, p, bytecode.NewInt(5))
	p.Instructions = []bytecode.Instruction{
		bytecode.EncodeABx(bytecode.OpLoadConst, 0, 0),
		bytecode.EncodeABC(bytecode.OpNeg, 1, 0, 0),
		bytecode.EncodeABC(bytecode.OpReturn, 1, 0, 0),
	}
	img.AddPrototype(p)
	_ = img.SetEntryPoint(0)
	got, err := NewVM(img).Run(p, nil)
	if err != nil || got.Int != -5 {
		t.Errorf("got %v, err=%v", got, err)
	}
}

func TestVM_DivByZero(t *testing.T) {
	img := bytecode.NewImage()
	p := &bytecode.FuncPrototype{Name: "div", NumRegs: 3}
	addConstants(img, p, bytecode.NewInt(10), bytecode.NewInt(0))
	p.Instructions = []bytecode.Instruction{
		bytecode.EncodeABx(bytecode.OpLoadConst, 0, 0),
		bytecode.EncodeABx(bytecode.OpLoadConst, 1, 1),
		bytecode.EncodeABC(bytecode.OpDiv, 2, 0, 1),
		bytecode.EncodeABC(bytecode.OpReturn, 2, 0, 0),
	}
	img.AddPrototype(p)
	_ = img.SetEntryPoint(0)
	_, err := NewVM(img).Run(p, nil)
	if err == nil || !strings.Contains(err.Error(), "division by zero") {
		t.Errorf("expected division by zero error, got %v", err)
	}
}

// --- Comparison & jumps -----------------------------------------------------

func TestVM_Compare(t *testing.T) {
	cases := []struct {
		op   bytecode.OpCode
		a, b int64
		want bool
	}{
		{bytecode.OpEq, 3, 3, true},
		{bytecode.OpEq, 3, 4, false},
		{bytecode.OpLt, 3, 4, true},
		{bytecode.OpLt, 4, 3, false},
		{bytecode.OpLt, 3, 3, false},
		{bytecode.OpLe, 3, 3, true},
		{bytecode.OpLe, 4, 3, false},
	}
	for _, tc := range cases {
		img := bytecode.NewImage()
		p := &bytecode.FuncPrototype{Name: "cmp", NumRegs: 3}
		addConstants(img, p, bytecode.NewInt(tc.a), bytecode.NewInt(tc.b))
		p.Instructions = []bytecode.Instruction{
			bytecode.EncodeABx(bytecode.OpLoadConst, 0, 0),
			bytecode.EncodeABx(bytecode.OpLoadConst, 1, 1),
			bytecode.EncodeABC(tc.op, 2, 0, 1),
			bytecode.EncodeABC(bytecode.OpReturn, 2, 0, 0),
		}
		img.AddPrototype(p)
		_ = img.SetEntryPoint(0)
		got, err := NewVM(img).Run(p, nil)
		if err != nil {
			t.Fatalf("%v: %v", tc.op, err)
		}
		if got.Bool != tc.want {
			t.Errorf("%v(%d, %d) = %v, want %v", tc.op, tc.a, tc.b, got.Bool, tc.want)
		}
	}
}

func TestVM_Not(t *testing.T) {
	img := bytecode.NewImage()
	p := &bytecode.FuncPrototype{Name: "not", NumRegs: 2}
	addConstants(img, p, bytecode.NewBool(true))
	p.Instructions = []bytecode.Instruction{
		bytecode.EncodeABx(bytecode.OpLoadConst, 0, 0),
		bytecode.EncodeABC(bytecode.OpNot, 1, 0, 0),
		bytecode.EncodeABC(bytecode.OpReturn, 1, 0, 0),
	}
	img.AddPrototype(p)
	_ = img.SetEntryPoint(0)
	got, err := NewVM(img).Run(p, nil)
	if err != nil || got.Bool != false {
		t.Errorf("got %v, err=%v", got, err)
	}
}

func TestVM_Jump_ForwardAndBackward(t *testing.T) {
	// Compute: x = 0; for i in 0..5 { x += 2 }; return x
	img := bytecode.NewImage()
	p := &bytecode.FuncPrototype{Name: "loop", NumRegs: 5}
	// r0 = i, r1 = x, r2 = limit (5), r3 = step (2), r4 = cond
	addConstants(img, p, bytecode.NewInt(0), bytecode.NewInt(5), bytecode.NewInt(2), bytecode.NewInt(1))
	p.Instructions = []bytecode.Instruction{
		bytecode.EncodeABx(bytecode.OpLoadConst, 0, 0), // i = 0
		bytecode.EncodeABx(bytecode.OpLoadConst, 1, 0), // x = 0
		bytecode.EncodeABx(bytecode.OpLoadConst, 2, 1), // limit = 5
		bytecode.EncodeABx(bytecode.OpLoadConst, 3, 2), // step = 2
		// loop: 4
		bytecode.EncodeABC(bytecode.OpLt, 4, 0, 2),        // 4: cond = i < limit
		bytecode.EncodeASBx(bytecode.OpJumpIfFalse, 4, 4), // 5: if !cond jump +4 → end (10)
		bytecode.EncodeABC(bytecode.OpAdd, 1, 1, 3),       // 6: x += 2
		bytecode.EncodeABx(bytecode.OpLoadConst, 4, 3),    // 7: r4 = 1
		bytecode.EncodeABC(bytecode.OpAdd, 0, 0, 4),       // 8: i += 1
		bytecode.EncodeASBx(bytecode.OpJump, 0, -6),       // 9: jump back to 4
		// end: 10
		bytecode.EncodeABC(bytecode.OpReturn, 1, 0, 0), // 10
	}
	img.AddPrototype(p)
	_ = img.SetEntryPoint(0)
	if err := img.Validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}
	got, err := NewVM(img).Run(p, nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got.Int != 10 {
		t.Errorf("got %d, want 10", got.Int)
	}
}

// --- String -----------------------------------------------------------------

func TestVM_Concat(t *testing.T) {
	img := bytecode.NewImage()
	p := &bytecode.FuncPrototype{Name: "cat", NumRegs: 3}
	addConstants(img, p, bytecode.NewString("hello "), bytecode.NewString("world"))
	p.Instructions = []bytecode.Instruction{
		bytecode.EncodeABx(bytecode.OpLoadConst, 0, 0),
		bytecode.EncodeABx(bytecode.OpLoadConst, 1, 1),
		bytecode.EncodeABC(bytecode.OpConcat, 2, 0, 1),
		bytecode.EncodeABC(bytecode.OpReturn, 2, 0, 0),
	}
	img.AddPrototype(p)
	_ = img.SetEntryPoint(0)
	got, err := NewVM(img).Run(p, nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got.AsString() != "hello world" {
		t.Errorf("got %q", got.AsString())
	}
}

// --- Calls ------------------------------------------------------------------

// buildAddCall creates an image with two protos: an `add(a, b) = a + b`
// callee, and a `main` that calls add(3, 4).
func buildAddCall(t *testing.T) (*bytecode.BytecodeImage, *bytecode.FuncPrototype) {
	t.Helper()
	img := bytecode.NewImage()

	// callee: add(a, b)
	add := &bytecode.FuncPrototype{Name: "add", NumRegs: 3, NumParams: 2}
	add.Instructions = []bytecode.Instruction{
		bytecode.EncodeABC(bytecode.OpAdd, 2, 0, 1),
		bytecode.EncodeABC(bytecode.OpReturn, 2, 0, 0),
	}
	addIdx := img.AddPrototype(add)

	// main: load consts, build closure to add, call it
	main := &bytecode.FuncPrototype{Name: "main", NumRegs: 4, NumParams: 0}
	addConstants(img, main, bytecode.NewInt(3), bytecode.NewInt(4))
	main.NestedProtos = []int{addIdx}
	main.Instructions = []bytecode.Instruction{
		bytecode.EncodeABx(bytecode.OpClosure, 0, 0),   // r0 = closure(add)
		bytecode.EncodeABx(bytecode.OpLoadConst, 1, 0), // r1 = 3
		bytecode.EncodeABx(bytecode.OpLoadConst, 2, 1), // r2 = 4
		bytecode.EncodeABC(bytecode.OpCall, 0, 2, 1),   // r0 = r0(r1, r2)
		bytecode.EncodeABC(bytecode.OpReturn, 0, 0, 0),
	}
	img.AddPrototype(main)
	_ = img.SetEntryPoint(1)
	if err := img.Validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}
	return img, main
}

func TestVM_Call(t *testing.T) {
	img, main := buildAddCall(t)
	got, err := NewVM(img).Run(main, nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got.Int != 7 {
		t.Errorf("got %d, want 7", got.Int)
	}
}

func TestVM_TailCall_Bounded(t *testing.T) {
	// countdown(n) = if n == 0 then 0 else countdown(n-1)
	// Tail-call form. Run with n=10000 — should not blow the stack.
	img := bytecode.NewImage()
	cd := &bytecode.FuncPrototype{Name: "countdown", NumRegs: 5, NumParams: 1}
	addConstants(img, cd, bytecode.NewInt(0), bytecode.NewInt(1))
	cdIdx := img.AddPrototype(cd)
	cd.NestedProtos = []int{cdIdx}
	cd.Instructions = []bytecode.Instruction{
		bytecode.EncodeABx(bytecode.OpLoadConst, 1, 0),    // 0: r1 = 0
		bytecode.EncodeABC(bytecode.OpEq, 2, 0, 1),        // 1: r2 = (n == 0)
		bytecode.EncodeASBx(bytecode.OpJumpIfFalse, 2, 2), // 2: if !r2 jump +2 → 5
		bytecode.EncodeABx(bytecode.OpLoadConst, 3, 0),    // 3: r3 = 0
		bytecode.EncodeABC(bytecode.OpReturn, 3, 0, 0),    // 4: return r3
		// 5: tail-call branch
		bytecode.EncodeABx(bytecode.OpLoadConst, 3, 1),   // 5: r3 = 1
		bytecode.EncodeABC(bytecode.OpSub, 4, 0, 3),      // 6: r4 = n - 1
		bytecode.EncodeABx(bytecode.OpClosure, 1, 0),     // 7: r1 = closure(countdown)  -- callee
		bytecode.EncodeABC(bytecode.OpMove, 2, 4, 0),     // 8: r2 = r4 (arg)
		bytecode.EncodeABC(bytecode.OpTailCall, 1, 1, 0), // 9: tailcall r1(r2)
	}
	if err := img.SetEntryPoint(0); err != nil {
		t.Fatalf("SetEntryPoint: %v", err)
	}
	if err := img.Validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}

	vm := NewVM(img)
	vm.MaxStack = 50 // small ceiling — TCO must not push frames
	got, err := vm.Run(cd, []bytecode.Value{bytecode.NewInt(10000)})
	if err != nil {
		t.Fatalf("Run: %v (final stack depth: %d)", err, len(vm.Stack))
	}
	if got.Int != 0 {
		t.Errorf("got %d, want 0", got.Int)
	}
}

func TestVM_StackOverflow(t *testing.T) {
	// Recursive (non-tail) infinite call must hit ErrStackOverflow.
	img := bytecode.NewImage()
	rec := &bytecode.FuncPrototype{Name: "rec", NumRegs: 2, NumParams: 0}
	recIdx := img.AddPrototype(rec)
	rec.NestedProtos = []int{recIdx}
	rec.Instructions = []bytecode.Instruction{
		bytecode.EncodeABx(bytecode.OpClosure, 0, 0),
		bytecode.EncodeABC(bytecode.OpCall, 0, 0, 1),
		bytecode.EncodeABC(bytecode.OpReturn, 0, 0, 0),
	}
	_ = img.SetEntryPoint(0)
	if err := img.Validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}
	vm := NewVM(img)
	vm.MaxStack = 50
	_, err := vm.Run(rec, nil)
	if !errors.Is(err, ErrStackOverflow) {
		t.Errorf("expected ErrStackOverflow, got %v", err)
	}
}

// --- Closures with captures -------------------------------------------------

func TestVM_Closure_WithCapture(t *testing.T) {
	// inner(x) = x + captured  -- "captured" is captured from caller's r1.
	// main: cap = 10; mk = closure(inner, capture r1); return mk(5) → 15
	img := bytecode.NewImage()

	inner := &bytecode.FuncPrototype{
		Name:        "inner",
		NumRegs:     3,
		NumParams:   1, // x in r0
		NumCaptures: 1, // captured in r1
	}
	// r2 = r0 + r1; return r2
	inner.Instructions = []bytecode.Instruction{
		bytecode.EncodeABC(bytecode.OpAdd, 2, 0, 1),
		bytecode.EncodeABC(bytecode.OpReturn, 2, 0, 0),
	}
	innerIdx := img.AddPrototype(inner)

	// Layout: r0 holds the value to capture (10), r1 holds the closure/result,
	// r2 holds the call argument (5). CALL A=1 reads its arg from A+1 = r2,
	// so the arg slot must be one above the closure register.
	main := &bytecode.FuncPrototype{Name: "main", NumRegs: 4}
	addConstants(img, main, bytecode.NewInt(10), bytecode.NewInt(5))
	main.NestedProtos = []int{innerIdx}
	main.Instructions = []bytecode.Instruction{
		bytecode.EncodeABx(bytecode.OpLoadConst, 0, 0), // 0: r0 = 10 (capture source)
		bytecode.EncodeABx(bytecode.OpClosure, 1, 0),   // 1: r1 = closure(inner, [...])
		bytecode.EncodeABC(bytecode.OpMove, 0, 0, 0),   // 2: pseudo-MOVE: capture src = r0 (B=0)
		bytecode.EncodeABx(bytecode.OpLoadConst, 2, 1), // 3: r2 = 5 (arg)
		bytecode.EncodeABC(bytecode.OpCall, 1, 1, 1),   // 4: r1 = r1(r2)
		bytecode.EncodeABC(bytecode.OpReturn, 1, 0, 0), // 5
	}
	img.AddPrototype(main)
	_ = img.SetEntryPoint(1)
	// NOTE: Validate would flag the pseudo-MOVE as a real instruction. Since
	// the dispatcher knows to skip it via NumCaptures, we deliberately skip
	// validation here. In Phase 2C the compiler will guarantee well-formed
	// images, and Validate can be taught to recognize the pseudo-MOVE pattern.

	got, err := NewVM(img).Run(main, nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got.Int != 15 {
		t.Errorf("got %d, want 15", got.Int)
	}
}

// --- Collections ------------------------------------------------------------

func TestVM_MakeListAndIndex(t *testing.T) {
	img := bytecode.NewImage()
	p := &bytecode.FuncPrototype{Name: "list", NumRegs: 6}
	addConstants(img, p,
		bytecode.NewInt(10),
		bytecode.NewInt(20),
		bytecode.NewInt(30),
		bytecode.NewInt(1), // index
	)
	p.Instructions = []bytecode.Instruction{
		bytecode.EncodeABx(bytecode.OpLoadConst, 0, 0), // 10
		bytecode.EncodeABx(bytecode.OpLoadConst, 1, 1), // 20
		bytecode.EncodeABx(bytecode.OpLoadConst, 2, 2), // 30
		bytecode.EncodeABC(bytecode.OpMakeList, 3, 0, 3),
		bytecode.EncodeABx(bytecode.OpLoadConst, 4, 3), // index 1
		bytecode.EncodeABC(bytecode.OpGetIndex, 5, 3, 4),
		bytecode.EncodeABC(bytecode.OpReturn, 5, 0, 0),
	}
	img.AddPrototype(p)
	_ = img.SetEntryPoint(0)
	got, err := NewVM(img).Run(p, nil)
	if err != nil || got.Int != 20 {
		t.Errorf("got %v err=%v", got, err)
	}
}

func TestVM_Cons(t *testing.T) {
	img := bytecode.NewImage()
	p := &bytecode.FuncPrototype{Name: "cons", NumRegs: 5}
	addConstants(img, p, bytecode.NewInt(0), bytecode.NewInt(1), bytecode.NewInt(2))
	p.Instructions = []bytecode.Instruction{
		bytecode.EncodeABx(bytecode.OpLoadConst, 0, 0),
		bytecode.EncodeABx(bytecode.OpLoadConst, 1, 1),
		bytecode.EncodeABx(bytecode.OpLoadConst, 2, 2),
		bytecode.EncodeABC(bytecode.OpMakeList, 3, 1, 2), // [1, 2]
		bytecode.EncodeABC(bytecode.OpCons, 4, 0, 3),     // 0 :: [1, 2] = [0, 1, 2]
		bytecode.EncodeABC(bytecode.OpReturn, 4, 0, 0),
	}
	img.AddPrototype(p)
	_ = img.SetEntryPoint(0)
	got, err := NewVM(img).Run(p, nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got.Tag != bytecode.TagList || len(got.AsList()) != 3 || got.AsList()[0].Int != 0 {
		t.Errorf("got %v", got)
	}
}

func TestVM_MakeRecordAndGetField(t *testing.T) {
	// Record { x: 1, y: 2 }, get field index 1 (y, since fields sort alphabetically)
	img := bytecode.NewImage()
	p := &bytecode.FuncPrototype{Name: "rec", NumRegs: 4}
	// Local constants: 0="x", 1="y", 2=Int(1), 3=Int(2). Field names are
	// referenced via pseudo-LOAD_CONST instructions following MAKE_RECORD.
	addConstants(img, p,
		bytecode.NewString("x"),
		bytecode.NewString("y"),
		bytecode.NewInt(1),
		bytecode.NewInt(2),
	)
	p.Instructions = []bytecode.Instruction{
		bytecode.EncodeABx(bytecode.OpLoadConst, 0, 2),     // r0 = 1
		bytecode.EncodeABx(bytecode.OpLoadConst, 1, 3),     // r1 = 2
		bytecode.EncodeABC(bytecode.OpMakeRecord, 2, 0, 2), // r2 = {x:r0, y:r1}
		bytecode.EncodeABx(bytecode.OpLoadConst, 0, 0),     // pseudo: name "x"
		bytecode.EncodeABx(bytecode.OpLoadConst, 0, 1),     // pseudo: name "y"
		// GET_FIELD reads C as the sorted field index. "x" < "y", so x is 0, y is 1.
		bytecode.EncodeABC(bytecode.OpGetField, 3, 2, 1), // r3 = rec.y = 2
		bytecode.EncodeABC(bytecode.OpReturn, 3, 0, 0),
	}
	img.AddPrototype(p)
	_ = img.SetEntryPoint(0)
	if err := img.Validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}
	got, err := NewVM(img).Run(p, nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got.Int != 2 {
		t.Errorf("got %d, want 2 (field y)", got.Int)
	}
}

// --- ADT --------------------------------------------------------------------

func TestVM_MakeADTAndGetTag(t *testing.T) {
	// Build Some(42), check tag is 0, return field 0 via GET_INDEX-on-ADT?
	// Actually GET_INDEX is list-only. We just verify GET_TAG.
	img := bytecode.NewImage()
	p := &bytecode.FuncPrototype{Name: "adt", NumRegs: 4}
	addConstants(img, p, bytecode.NewInt(42))
	p.Instructions = []bytecode.Instruction{
		bytecode.EncodeABx(bytecode.OpLoadConst, 1, 0),  // r1 = 42 (will be field 0 of ADT, sitting at A+1)
		bytecode.EncodeABC(bytecode.OpMakeADT, 0, 0, 1), // r0 = ADT{tag=0, fields=[r1]}
		bytecode.EncodeABC(bytecode.OpGetTag, 2, 0, 0),  // r2 = r0.tag
		bytecode.EncodeABC(bytecode.OpReturn, 2, 0, 0),
	}
	img.AddPrototype(p)
	_ = img.SetEntryPoint(0)
	got, err := NewVM(img).Run(p, nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got.Int != 0 {
		t.Errorf("got tag %d, want 0", got.Int)
	}
}

// --- Stub opcodes -----------------------------------------------------------

func TestVM_BuiltinCall_UnknownIndex(t *testing.T) {
	// Index past end of BuiltinTable should produce a clear error.
	img := bytecode.NewImage()
	p := &bytecode.FuncPrototype{Name: "stub", NumRegs: 1}
	p.Instructions = []bytecode.Instruction{
		bytecode.EncodeABC(bytecode.OpBuiltinCall, 0, 250, 0),
		bytecode.EncodeABC(bytecode.OpReturn, 0, 0, 0),
	}
	img.AddPrototype(p)
	_ = img.SetEntryPoint(0)
	_, err := NewVM(img).Run(p, nil)
	if err == nil || !strings.Contains(err.Error(), "unknown builtin index") {
		t.Errorf("got %v", err)
	}
}

// --- Error source-location --------------------------------------------------

func TestVM_VMError_LineInfo(t *testing.T) {
	img := bytecode.NewImage()
	p := &bytecode.FuncPrototype{Name: "boom", NumRegs: 3}
	addConstants(img, p, bytecode.NewInt(10), bytecode.NewInt(0))
	p.Instructions = []bytecode.Instruction{
		bytecode.EncodeABx(bytecode.OpLoadConst, 0, 0),
		bytecode.EncodeABx(bytecode.OpLoadConst, 1, 1),
		bytecode.EncodeABC(bytecode.OpDiv, 2, 0, 1),
		bytecode.EncodeABC(bytecode.OpReturn, 2, 0, 0),
	}
	p.LineInfo = []int{10, 11, 12, 13}
	img.AddPrototype(p)
	_ = img.SetEntryPoint(0)
	_, err := NewVM(img).Run(p, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	verr, ok := err.(*VMError)
	if !ok {
		t.Fatalf("expected *VMError, got %T", err)
	}
	if verr.Line != 12 {
		t.Errorf("got line %d, want 12", verr.Line)
	}
	if verr.Func != "boom" {
		t.Errorf("got func %q, want boom", verr.Func)
	}
}
