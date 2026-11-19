package bytecode

import (
	"strings"
	"testing"
)

// matchFailImage is a one-instruction prototype: MATCH_FAIL r<reg>, pos#0
// over a constant pool whose entry 0 is pos.
func matchFailImage(reg uint8, pos Value) *BytecodeImage {
	img := NewImage()
	cIdx := img.AddConstant(pos)
	img.AddPrototype(&FuncPrototype{
		Name:         "f",
		NumRegs:      2,
		NumParams:    1,
		Constants:    []int{cIdx},
		Instructions: []Instruction{EncodeABx(OpMatchFail, reg, 0)},
	})
	_ = img.SetEntryPoint(0)
	return img
}

// MATCH_FAIL's operands are checked like any other: the scrutinee register
// must exist and the position must be a String constant
// (M-V1-MATCH-EXHAUSTIVENESS M0).
func TestValidate_MatchFail(t *testing.T) {
	if err := matchFailImage(0, NewString("m.ail:3:5")).Validate(); err != nil {
		t.Errorf("valid MATCH_FAIL rejected: %v", err)
	}
	if err := matchFailImage(9, NewString("m.ail:3:5")).Validate(); err == nil || !strings.Contains(err.Error(), "scrutinee") {
		t.Errorf("out-of-range scrutinee register: got %v", err)
	}
	if err := matchFailImage(0, NewInt(3)).Validate(); err == nil || !strings.Contains(err.Error(), "match position constant") {
		t.Errorf("non-String position constant: got %v", err)
	}
	img := matchFailImage(0, NewString("m.ail:3:5"))
	img.Prototypes[0].Instructions[0] = EncodeABx(OpMatchFail, 0, 7)
	if err := img.Validate(); err == nil {
		t.Error("out-of-range position constant accepted")
	}
}

func TestDisassemble_MatchFail(t *testing.T) {
	dis := Disassemble(matchFailImage(1, NewString("m.ail:3:5")))
	if !strings.Contains(dis, "MATCH_FAIL   r1, pos#0") || !strings.Contains(dis, "m.ail:3:5") {
		t.Errorf("disassembly:\n%s", dis)
	}
	if got := EncodeABx(OpMatchFail, 1, 0).String(); got != "MATCH_FAIL r1, #0" {
		t.Errorf("Instruction.String() = %q", got)
	}
}
