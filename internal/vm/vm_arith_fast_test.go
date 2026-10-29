package vm

import (
	"math"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/bytecode"
)

// runBinop executes `r2 = op r0 r1; return r2` on the VM.
func runBinop(t *testing.T, op bytecode.OpCode, a, b bytecode.Value) (bytecode.Value, error) {
	t.Helper()
	img := bytecode.NewImage()
	p := &bytecode.FuncPrototype{Name: "binop", NumRegs: 3, NumParams: 2}
	p.Instructions = []bytecode.Instruction{
		bytecode.EncodeABC(op, 2, 0, 1),
		bytecode.EncodeABC(bytecode.OpReturn, 2, 0, 0),
	}
	img.AddPrototype(p)
	return NewVM(img).Run(p, []bytecode.Value{a, b})
}

// TestBinopDispatchMatchesHelpers: through the real dispatch loop, int x int
// (the inline fast path) gives the helpers' answer and every mixed or non-int pair still fails or
// answers exactly as the helpers do, with the same error text.
func TestBinopDispatchMatchesHelpers(t *testing.T) {
	// Int edges exercise the inline int fast path (M-VM-QUICK-WINS W3),
	// including wrap-around; the rest exercise the fall-through to the helpers.
	vals := []bytecode.Value{bytecode.NewFloat(2.5), bytecode.NewString("s"), bytecode.NewBool(true)}
	for _, n := range []int64{0, 1, -1, 6, -4, 1 << 31, math.MaxInt64, math.MinInt64, math.MinInt64 + 1} {
		vals = append(vals, bytecode.NewInt(n))
	}
	ops := []bytecode.OpCode{bytecode.OpAdd, bytecode.OpSub, bytecode.OpMul, bytecode.OpDiv, bytecode.OpMod, bytecode.OpLt, bytecode.OpLe, bytecode.OpEq}
	for _, op := range ops {
		for _, a := range vals {
			for _, b := range vals {
				got, err := runBinop(t, op, a, b)
				var want bytecode.Value
				var wantErr error
				switch op {
				case bytecode.OpEq:
					want = bytecode.NewBool(runtimeEq(a, b))
				case bytecode.OpLt, bytecode.OpLe:
					var r bool
					r, wantErr = compare(op, a, b)
					want = bytecode.NewBool(r)
				default:
					want, wantErr = arith(op, a, b)
				}
				if (err != nil) != (wantErr != nil) {
					t.Fatalf("%s(%v, %v): err %v, helper err %v", op, a, b, err, wantErr)
				}
				if wantErr != nil {
					if !strings.Contains(err.Error(), wantErr.Error()) {
						t.Fatalf("%s(%v, %v): error %q does not carry the helper's %q", op, a, b, err, wantErr)
					}
					continue
				}
				if got != want {
					t.Fatalf("%s(%v, %v) = %v, helper %v", op, a, b, got, want)
				}
			}
		}
	}
	// Division and modulo by zero keep their RT001 error on the VM.
	for _, op := range []bytecode.OpCode{bytecode.OpDiv, bytecode.OpMod} {
		if _, err := runBinop(t, op, bytecode.NewInt(1), bytecode.NewInt(0)); err == nil {
			t.Fatalf("%s by zero succeeded", op)
		}
	}
}
