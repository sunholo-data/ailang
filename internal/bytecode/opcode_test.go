package bytecode

import (
	"strings"
	"testing"
)

func TestOpCodeString_AllDefined(t *testing.T) {
	for op := OpCode(0); int(op) < OpCount(); op++ {
		s := op.String()
		if strings.HasPrefix(s, "UNKNOWN") {
			t.Errorf("opcode %d has no String() mapping", op)
		}
	}
}

func TestOpCodeString_Sentinel(t *testing.T) {
	// Anything past opCount must report UNKNOWN.
	got := OpCode(opCount).String()
	if !strings.HasPrefix(got, "UNKNOWN") {
		t.Errorf("expected UNKNOWN for sentinel, got %q", got)
	}
}

func TestEncodeABC_RoundTrip(t *testing.T) {
	cases := []struct {
		op      OpCode
		a, b, c uint8
	}{
		{OpAdd, 1, 2, 3},
		{OpSub, 0, 0, 0},
		{OpMul, 255, 255, 255},
		{OpEq, 7, 8, 9},
		{OpMakeList, 4, 5, 6},
		{OpMakeRecord, 0, 1, 2},
		{OpCall, 10, 20, 30},
	}
	for _, tc := range cases {
		i := EncodeABC(tc.op, tc.a, tc.b, tc.c)
		if i.Op() != tc.op {
			t.Errorf("Op: got %v, want %v", i.Op(), tc.op)
		}
		if i.A() != tc.a {
			t.Errorf("A: got %d, want %d", i.A(), tc.a)
		}
		if i.B() != tc.b {
			t.Errorf("B: got %d, want %d", i.B(), tc.b)
		}
		if i.C() != tc.c {
			t.Errorf("C: got %d, want %d", i.C(), tc.c)
		}
	}
}

func TestEncodeABx_RoundTrip(t *testing.T) {
	cases := []struct {
		op OpCode
		a  uint8
		bx uint16
	}{
		{OpLoadConst, 0, 0},
		{OpLoadConst, 5, 1234},
		{OpLoadGlobal, 255, 65535},
		{OpClosure, 1, 42},
		{OpBuiltinCall, 3, 7},
	}
	for _, tc := range cases {
		i := EncodeABx(tc.op, tc.a, tc.bx)
		if i.Op() != tc.op {
			t.Errorf("Op: got %v, want %v", i.Op(), tc.op)
		}
		if i.A() != tc.a {
			t.Errorf("A: got %d, want %d", i.A(), tc.a)
		}
		if i.Bx() != tc.bx {
			t.Errorf("Bx: got %d, want %d", i.Bx(), tc.bx)
		}
	}
}

func TestEncodeASBx_SignedRoundTrip(t *testing.T) {
	cases := []struct {
		op  OpCode
		a   uint8
		sbx int
	}{
		{OpJump, 0, 0},
		{OpJump, 0, 1},
		{OpJump, 0, -1},
		{OpJump, 0, 32767},
		{OpJump, 0, -32768},
		{OpJumpIfFalse, 5, -100},
		{OpJumpIfFalse, 5, 100},
	}
	for _, tc := range cases {
		i := EncodeASBx(tc.op, tc.a, tc.sbx)
		if i.Op() != tc.op {
			t.Errorf("Op: got %v, want %v", i.Op(), tc.op)
		}
		if i.A() != tc.a {
			t.Errorf("A: got %d, want %d", i.A(), tc.a)
		}
		if i.SBx() != tc.sbx {
			t.Errorf("SBx: got %d, want %d", i.SBx(), tc.sbx)
		}
	}
}

func TestEncodeASBx_OutOfRangePanics(t *testing.T) {
	cases := []int{32768, -32769, 1 << 20, -(1 << 20)}
	for _, sbx := range cases {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("expected panic for SBx=%d", sbx)
				}
			}()
			_ = EncodeASBx(OpJump, 0, sbx)
		}()
	}
}

func TestInstruction_StringFormat(t *testing.T) {
	// Spot check a few formats to ensure no panics and reasonable output.
	cases := []Instruction{
		EncodeABC(OpAdd, 1, 2, 3),
		EncodeABx(OpLoadConst, 0, 5),
		EncodeASBx(OpJump, 0, -10),
		EncodeASBx(OpJumpIfFalse, 4, 20),
		EncodeABC(OpMove, 1, 2, 0),
		EncodeABC(OpReturn, 7, 0, 0),
	}
	for _, i := range cases {
		s := i.String()
		if s == "" {
			t.Errorf("empty string for %v", i.Op())
		}
	}
}

func TestOpCount_Stable(t *testing.T) {
	// If you add a new opcode, update this number deliberately. This guards
	// against accidental opcode space changes that would invalidate any
	// serialized bytecode in the wild (none yet, but the discipline matters).
	const expected = 32
	if OpCount() != expected {
		t.Errorf("OpCount() = %d, expected %d — if you added an opcode, update this test and bump bytecode format version", OpCount(), expected)
	}
}
