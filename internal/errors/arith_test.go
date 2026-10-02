package errors

import (
	"errors"
	"fmt"
	"testing"
)

func TestDivByZeroError_Message(t *testing.T) {
	e := &DivByZeroError{Op: OpDivision}
	if got, want := e.Error(), "RT001: integer division by zero"; got != want {
		t.Errorf("no position: %q, want %q", got, want)
	}
	e = &DivByZeroError{Op: OpModulo, Pos: "m.ail:3:20"}
	if got, want := e.Error(), "RT001: integer modulo by zero at m.ail:3:20"; got != want {
		t.Errorf("with position: %q, want %q", got, want)
	}
}

func TestDivByZeroError_CheckIntDivisor(t *testing.T) {
	for _, d := range []int64{1, -1, 7, -9223372036854775808} {
		if err := CheckIntDivisor(OpDivision, d); err != nil {
			t.Errorf("CheckIntDivisor(%d) = %v, want nil", d, err)
		}
	}
	err := CheckIntDivisor(OpModulo, 0)
	var dz *DivByZeroError
	if !errors.As(fmt.Errorf("wrapped: %w", err), &dz) || dz.Op != OpModulo || dz.Pos != "" {
		t.Fatalf("CheckIntDivisor(0) = %#v, want *DivByZeroError{Op: modulo}", err)
	}
	// Each call returns a fresh value: the evaluator fills Pos in place, so a
	// shared sentinel would leak one call site's position into another.
	if CheckIntDivisor(OpModulo, 0) == err {
		t.Error("CheckIntDivisor returned a shared error value")
	}
}

func TestDivByZeroError_RegistryRow(t *testing.T) {
	info, ok := GetErrorInfo(RT001)
	if !ok || info.Phase != "runtime" || info.Category != "arithmetic" {
		t.Fatalf("RT001 registry row = %+v, %v; want runtime/arithmetic", info, ok)
	}
	if (&DivByZeroError{}).Code() != RT001 {
		t.Error("DivByZeroError.Code() is not RT001")
	}
}
