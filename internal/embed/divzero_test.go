package embed

import (
	"errors"
	"math"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/effects"
	ailerrors "github.com/sunholo-data/ailang/internal/errors"
	"github.com/sunholo-data/ailang/internal/eval"
)

// M-INT-DIV-ZERO-ERROR (#1449): integer / and % by zero fail with a typed
// RT001 error that carries the position of the dividing expression. Before
// the fix `/` panicked out of the Num[int] dictionary method and `%` returned
// an unpositioned "[RT_DIV0] Modulo by zero".

const divZeroModule = "internal/embed/testdata/divzero"

func newDivZeroEngine(t *testing.T, contracts bool) *Engine {
	t.Helper()
	engine := newExitTestEngine(t)
	if contracts {
		effCtx := effects.NewEffContext(nil)
		effCtx.Contracts = effects.NewContractContextWithMode(effects.ContractModePanic)
		engine.runtime.GetEvaluator().SetEffContext(effCtx)
	}
	return engine
}

// callDivZero calls fn and returns the *DivByZeroError it must fail with.
func callDivZero(t *testing.T, engine *Engine, fn string, args ...interface{}) *ailerrors.DivByZeroError {
	t.Helper()
	val, err := engine.Call(divZeroModule, fn, args...)
	if err == nil {
		t.Fatalf("%s%v = %v, want an RT001 error", fn, args, val)
	}
	var dz *ailerrors.DivByZeroError
	if !errors.As(err, &dz) {
		t.Fatalf("%s%v error = %T %q, want *errors.DivByZeroError", fn, args, err, err)
	}
	if !strings.Contains(err.Error(), "RT001: integer "+dz.Op+" by zero") {
		t.Errorf("%s: message %q does not carry the RT001 text", fn, err)
	}
	return dz
}

func TestIntDivZero_Positions(t *testing.T) {
	engine := newDivZeroEngine(t, false)
	cases := []struct {
		fn   string
		args []interface{}
		op   string
		line string // ":<line>:" of the dividing expression in divzero.ail
	}{
		{"divBy", []interface{}{0}, "division", ":7:"},
		{"modBy", []interface{}{0}, "modulo", ":9:"},
		{"outer", []interface{}{1}, "division", ":7:"}, // the inner function's line, not the caller's
		{"literal", nil, "division", ":13:"},
		{"inCallback", []interface{}{0}, "division", ":15:"},
	}
	for _, c := range cases {
		t.Run(c.fn, func(t *testing.T) {
			dz := callDivZero(t, engine, c.fn, c.args...)
			if dz.Op != c.op {
				t.Errorf("Op = %q, want %q", dz.Op, c.op)
			}
			if !strings.Contains(dz.Pos, "divzero.ail"+c.line) {
				t.Errorf("Pos = %q, want divzero.ail%s<col>", dz.Pos, c.line)
			}
		})
	}
}

func TestIntDivZero_InEnsuresContract(t *testing.T) {
	engine := newDivZeroEngine(t, true)
	dz := callDivZero(t, engine, "ensuresDivides", 0)
	if !strings.Contains(dz.Pos, ":18:") {
		t.Errorf("Pos = %q, want the ensures line :18:", dz.Pos)
	}
}

func TestIntDivZero_NonZeroAndFloatUnchanged(t *testing.T) {
	engine := newDivZeroEngine(t, false)
	if v, err := engine.Call(divZeroModule, "divBy", 5); err != nil || v.(*eval.IntValue).Value != 2 {
		t.Errorf("divBy(5) = %v, %v; want 2", v, err)
	}
	if v, err := engine.CallPreserveFloats(divZeroModule, "fdiv", 0.0); err != nil || !math.IsInf(v.(*eval.FloatValue).Value, 1) {
		t.Errorf("fdiv(0.0) = %v, %v; want +Inf (IEEE 754)", v, err)
	}
	if v, err := engine.CallPreserveFloats(divZeroModule, "fmod", 0.0); err != nil || !math.IsNaN(v.(*eval.FloatValue).Value) {
		t.Errorf("fmod(0.0) = %v, %v; want NaN (IEEE 754)", v, err)
	}
}

// Go defines MinInt64 / -1 == MinInt64 and MinInt64 % -1 == 0 (two's-complement
// wrap, no panic), so division by zero is the only integer-arithmetic panic.
func TestIntDivZero_MinIntOverflowDoesNotPanic(t *testing.T) {
	engine := newDivZeroEngine(t, false)
	if v, err := engine.Call(divZeroModule, "minDiv", -1); err != nil || v.(*eval.IntValue).Value != math.MinInt64 {
		t.Errorf("minDiv(-1) = %v, %v; want %d", v, err, int64(math.MinInt64))
	}
	if v, err := engine.Call(divZeroModule, "minMod", -1); err != nil || v.(*eval.IntValue).Value != 0 {
		t.Errorf("minMod(-1) = %v, %v; want 0", v, err)
	}
}
