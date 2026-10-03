package vm

import (
	"errors"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/bytecode"
	ailerrors "github.com/sunholo-data/ailang/internal/errors"
)

// #1449: the VM reports integer / and % by zero as the same RT001
// *errors.DivByZeroError the evaluator raises, reachable with errors.As
// through the VMError, so hosts can match on the code for either engine.

func runArith(t *testing.T, op bytecode.OpCode, a, b int64) error {
	t.Helper()
	img := bytecode.NewImage()
	p := &bytecode.FuncPrototype{Name: "arith", NumRegs: 3}
	addConstants(img, p, bytecode.NewInt(a), bytecode.NewInt(b))
	p.Instructions = []bytecode.Instruction{
		bytecode.EncodeABx(bytecode.OpLoadConst, 0, 0),
		bytecode.EncodeABx(bytecode.OpLoadConst, 1, 1),
		bytecode.EncodeABC(op, 2, 0, 1),
		bytecode.EncodeABC(bytecode.OpReturn, 2, 0, 0),
	}
	img.AddPrototype(p)
	_ = img.SetEntryPoint(0)
	_, err := NewVM(img).Run(p, nil)
	return err
}

func TestVMIntDivZero_Opcodes(t *testing.T) {
	for op, name := range map[bytecode.OpCode]string{bytecode.OpDiv: ailerrors.OpDivision, bytecode.OpMod: ailerrors.OpModulo} {
		err := runArith(t, op, 10, 0)
		var dz *ailerrors.DivByZeroError
		if !errors.As(err, &dz) || dz.Op != name {
			t.Fatalf("%s by zero: error %T %v, want *DivByZeroError{Op: %s}", op, err, err, name)
		}
		if !strings.Contains(err.Error(), "RT001: integer "+name+" by zero") {
			t.Errorf("%s by zero: message %q lacks the RT001 text", op, err)
		}
	}
}

func TestVMIntDivZero_ModBuiltin(t *testing.T) {
	_, err := builtinModInt([]bytecode.Value{bytecode.NewInt(10), bytecode.NewInt(0)})
	var dz *ailerrors.DivByZeroError
	if !errors.As(err, &dz) || dz.Op != ailerrors.OpModulo {
		t.Fatalf("_mod_Int(10, 0) error %T %v, want *DivByZeroError{Op: modulo}", err, err)
	}
}

// MinInt64 / -1 wraps and MinInt64 % -1 is 0 in Go: no panic, matching the evaluator.
func TestVMIntDivZero_MinIntOverflowDoesNotPanic(t *testing.T) {
	const minInt = -1 << 63
	if err := runArith(t, bytecode.OpDiv, minInt, -1); err != nil {
		t.Errorf("MinInt / -1: %v", err)
	}
	if err := runArith(t, bytecode.OpMod, minInt, -1); err != nil {
		t.Errorf("MinInt %% -1: %v", err)
	}
}
