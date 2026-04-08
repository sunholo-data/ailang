package bytecode

import (
	"strings"
	"testing"
)

func TestImage_NewImage(t *testing.T) {
	img := NewImage()
	if img.EntryPoint != -1 {
		t.Errorf("new image entry point: got %d, want -1", img.EntryPoint)
	}
	if len(img.Constants) != 0 || len(img.Prototypes) != 0 {
		t.Error("new image should have empty constants and prototypes")
	}
}

func TestImage_AddConstant_Dedupes(t *testing.T) {
	img := NewImage()
	a := img.AddConstant(NewInt(42))
	b := img.AddConstant(NewInt(42))
	c := img.AddConstant(NewInt(43))
	if a != b {
		t.Errorf("equal constants should dedupe: a=%d, b=%d", a, b)
	}
	if a == c {
		t.Error("different constants should not dedupe")
	}
	if len(img.Constants) != 2 {
		t.Errorf("got %d constants, want 2", len(img.Constants))
	}
}

func TestImage_AddConstant_StructuralDedup(t *testing.T) {
	img := NewImage()
	a := img.AddConstant(NewList([]Value{NewInt(1), NewInt(2)}))
	b := img.AddConstant(NewList([]Value{NewInt(1), NewInt(2)}))
	if a != b {
		t.Error("structurally equal lists should dedupe")
	}
}

func TestImage_AddPrototype(t *testing.T) {
	img := NewImage()
	p1 := &FuncPrototype{Name: "f", NumRegs: 1}
	p2 := &FuncPrototype{Name: "g", NumRegs: 2}
	i1 := img.AddPrototype(p1)
	i2 := img.AddPrototype(p2)
	if i1 != 0 || i2 != 1 {
		t.Errorf("indices: got %d %d, want 0 1", i1, i2)
	}
	if img.Prototypes[i1] != p1 || img.Prototypes[i2] != p2 {
		t.Error("prototype lookup mismatch")
	}
}

func TestImage_SetEntryPoint(t *testing.T) {
	img := NewImage()
	img.AddPrototype(&FuncPrototype{Name: "main", NumRegs: 1})
	if err := img.SetEntryPoint(0); err != nil {
		t.Fatalf("SetEntryPoint: %v", err)
	}
	if img.EntryPoint != 0 {
		t.Errorf("got %d, want 0", img.EntryPoint)
	}
	if err := img.SetEntryPoint(5); err == nil {
		t.Error("expected error for out-of-range index")
	}
}

func TestFuncPrototype_LookupConstant(t *testing.T) {
	img := NewImage()
	idx := img.AddConstant(NewInt(99))
	p := &FuncPrototype{
		Name:      "f",
		NumRegs:   1,
		Constants: []int{idx},
	}
	v, ok := p.LookupConstant(0, img)
	if !ok {
		t.Fatal("lookup failed")
	}
	if v.Tag != TagInt || v.Int != 99 {
		t.Errorf("got %v, want Int(99)", v)
	}

	// Out of range local index.
	if _, ok := p.LookupConstant(5, img); ok {
		t.Error("expected lookup failure for out-of-range local index")
	}

	// Local maps to a bad pool index.
	pBad := &FuncPrototype{Constants: []int{42}}
	if _, ok := pBad.LookupConstant(0, img); ok {
		t.Error("expected lookup failure for out-of-range pool index")
	}
}

func TestValidate_HappyPath(t *testing.T) {
	img := NewImage()
	cIdx := img.AddConstant(NewInt(7))
	p := &FuncPrototype{
		Name:      "id",
		NumRegs:   2,
		NumParams: 0,
		Constants: []int{cIdx},
		Instructions: []Instruction{
			EncodeABx(OpLoadConst, 0, 0),
			EncodeABC(OpReturn, 0, 0, 0),
		},
	}
	img.AddPrototype(p)
	_ = img.SetEntryPoint(0)
	if err := img.Validate(); err != nil {
		t.Errorf("Validate: %v", err)
	}
}

func TestValidate_RegisterOverflow(t *testing.T) {
	img := NewImage()
	cIdx := img.AddConstant(NewInt(0))
	p := &FuncPrototype{
		Name:      "bad",
		NumRegs:   2,
		Constants: []int{cIdx},
		Instructions: []Instruction{
			EncodeABx(OpLoadConst, 5 /* exceeds NumRegs=2 */, 0),
		},
	}
	img.AddPrototype(p)
	err := img.Validate()
	if err == nil || !strings.Contains(err.Error(), "exceeds NumRegs") {
		t.Errorf("expected register overflow error, got %v", err)
	}
}

func TestValidate_BadConstantIndex(t *testing.T) {
	img := NewImage()
	p := &FuncPrototype{
		Name:    "bad",
		NumRegs: 1,
		// Constants is empty but instruction references local index 0.
		Instructions: []Instruction{
			EncodeABx(OpLoadConst, 0, 0),
		},
	}
	img.AddPrototype(p)
	err := img.Validate()
	if err == nil || !strings.Contains(err.Error(), "constant index") {
		t.Errorf("expected constant index error, got %v", err)
	}
}

func TestValidate_JumpOutOfRange(t *testing.T) {
	img := NewImage()
	p := &FuncPrototype{
		Name:    "bad",
		NumRegs: 1,
		Instructions: []Instruction{
			EncodeASBx(OpJump, 0, 100),
		},
	}
	img.AddPrototype(p)
	err := img.Validate()
	if err == nil || !strings.Contains(err.Error(), "jump target") {
		t.Errorf("expected jump target error, got %v", err)
	}
}

func TestValidate_NumParamsExceedsNumRegs(t *testing.T) {
	img := NewImage()
	p := &FuncPrototype{
		Name:      "bad",
		NumRegs:   2,
		NumParams: 5,
	}
	img.AddPrototype(p)
	err := img.Validate()
	if err == nil || !strings.Contains(err.Error(), "NumParams") {
		t.Errorf("expected NumParams error, got %v", err)
	}
}

func TestValidate_LineInfoLengthMismatch(t *testing.T) {
	img := NewImage()
	cIdx := img.AddConstant(NewInt(0))
	p := &FuncPrototype{
		Name:      "bad",
		NumRegs:   1,
		Constants: []int{cIdx},
		Instructions: []Instruction{
			EncodeABx(OpLoadConst, 0, 0),
			EncodeABC(OpReturn, 0, 0, 0),
		},
		LineInfo: []int{1}, // mismatched length
	}
	img.AddPrototype(p)
	err := img.Validate()
	if err == nil || !strings.Contains(err.Error(), "LineInfo") {
		t.Errorf("expected LineInfo error, got %v", err)
	}
}

func TestFuncPrototype_AsRefInterface(t *testing.T) {
	// Compile-time check that *FuncPrototype implements FuncPrototypeRef.
	var _ FuncPrototypeRef = (*FuncPrototype)(nil)
	p := &FuncPrototype{Name: "f", NumRegs: 3, NumParams: 1}
	var ref FuncPrototypeRef = p
	if ref.ProtoName() != "f" {
		t.Errorf("ProtoName: got %q, want f", ref.ProtoName())
	}
	if ref.NumRegisters() != 3 || ref.NumParameters() != 1 {
		t.Errorf("regs/params via interface: got %d/%d, want 3/1", ref.NumRegisters(), ref.NumParameters())
	}
}

func TestValidate_NestedProtoOutOfRange(t *testing.T) {
	img := NewImage()
	p := &FuncPrototype{
		Name:    "bad",
		NumRegs: 1,
		// NestedProtos[0] = 99 → bad image table index
		NestedProtos: []int{99},
		Instructions: []Instruction{
			EncodeABx(OpClosure, 0, 0),
		},
	}
	img.AddPrototype(p)
	err := img.Validate()
	if err == nil || !strings.Contains(err.Error(), "prototype table index") {
		t.Errorf("expected nested proto error, got %v", err)
	}
}
