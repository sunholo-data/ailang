package bytecode

import (
	"strings"
	"testing"
)

// TestDisassemble_SmallProgram builds a hand-assembled image with one
// function that loads a constant and returns it, plus a jump, and verifies
// the disassembler renders the expected sections (header, constant pool,
// per-prototype block, jump target annotation, constant annotation).
func TestDisassemble_SmallProgram(t *testing.T) {
	img := NewImage()
	c0 := img.AddConstant(NewInt(42))
	c1 := img.AddConstant(NewString("hello"))

	p := &FuncPrototype{
		Name:      "test",
		NumParams: 0,
		NumRegs:   3,
		Constants: []int{c0, c1},
		Instructions: []Instruction{
			EncodeABx(OpLoadConst, 0, 0), // r0 = const#0 (42)
			EncodeABx(OpLoadConst, 1, 1), // r1 = const#1 ("hello")
			EncodeASBx(OpJump, 0, 1),     // jump +1 → ip 4
			EncodeABC(OpAdd, 2, 0, 0),    // (skipped)
			EncodeABC(OpReturn, 0, 0, 0), // return r0
		},
		LineInfo: []int{10, 10, 11, 11, 12},
	}
	idx := img.AddPrototype(p)
	if err := img.SetEntryPoint(idx); err != nil {
		t.Fatalf("SetEntryPoint: %v", err)
	}

	out := Disassemble(img)
	t.Logf("Disassembly:\n%s", out)

	mustContain := []string{
		"=== BytecodeImage ===",
		"Prototypes: 1",
		"Constants:  2",
		"Entry:      0 (test)",
		"--- Constant Pool ---",
		"#0 = int 42",
		`#1 = string "hello"`,
		"--- Prototype 0: test ---",
		"params=0  regs=3  captures=0  insns=5",
		"local consts: [0→#0] [1→#1]",
		"; line 10",
		"; line 11",
		"; line 12",
		"LOAD_CONST",
		"; int 42",
		`; string "hello"`,
		"JUMP",
		"-> 0004", // jump target annotation
		"RETURN",
	}
	for _, s := range mustContain {
		if !strings.Contains(out, s) {
			t.Errorf("disassembly missing %q\n--- output ---\n%s", s, out)
		}
	}
}

// TestDisassemble_NilImage and TestDisassemble_NoEntry verify the edge
// cases don't panic.
func TestDisassemble_NilImage(t *testing.T) {
	out := Disassemble(nil)
	if !strings.Contains(out, "<nil image>") {
		t.Errorf("expected nil-image message, got %q", out)
	}
}

func TestDisassemble_NoEntry(t *testing.T) {
	img := NewImage()
	p := &FuncPrototype{Name: "lib", NumRegs: 1, Instructions: []Instruction{
		EncodeABC(OpReturn, 0, 0, 0),
	}}
	img.AddPrototype(p)
	out := Disassemble(img)
	if !strings.Contains(out, "Entry:      <none>") {
		t.Errorf("expected no-entry marker, got:\n%s", out)
	}
}
