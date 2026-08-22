package golang

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestGeneratedSliceConverters(t *testing.T) {
	gen := New("runtimefixture")
	gen.RegisterADTConstructor("UserADT", "One", 0)
	runtimeCode, err := gen.GenerateRuntime()
	if err != nil {
		t.Fatalf("GenerateRuntime failed: %v", err)
	}

	testCode := `package runtimefixture

import "testing"

func recoveredPanic(f func()) (got interface{}) {
	defer func() { got = recover() }()
	f()
	return nil
}

func TestLiteralConverterRejectsUnsupportedInput(t *testing.T) {
	got := recoveredPanic(func() { ConvertToInt64Slice(int64(7)) })
	want := "ConvertToInt64Slice: expected list or array slice, got int64"
	if got != want {
		t.Fatalf("recovered panic = %#v, want %q", got, want)
	}
}

// Regression pin: template-loop converters already failed loudly before ArrayVal support.
func TestTemplateConverterRejectsUnsupportedInput(t *testing.T) {
	got := recoveredPanic(func() { ConvertToUserADTSlice(int64(7)) })
	want := "ConvertToUserADTSlice: expected []interface{}, got int64"
	if got != want {
		t.Fatalf("recovered panic = %#v, want %q", got, want)
	}
}

func TestConvertersAcceptArrayVal(t *testing.T) {
	ints := ConvertToInt64Slice(ArrayVal{int64(1), int64(2), int64(3)})
	wantInts := []int64{1, 2, 3}
	if len(ints) != len(wantInts) {
		t.Fatalf("int converter length = %d, want %d", len(ints), len(wantInts))
	}
	for i := range wantInts {
		if ints[i] != wantInts[i] {
			t.Errorf("int converter element %d = %d, want %d", i, ints[i], wantInts[i])
		}
	}

	one := &UserADT{Kind: "One"}
	adts := ConvertToUserADTSlice(ArrayVal{one})
	if len(adts) != 1 || adts[0] != one {
		t.Fatalf("ADT converter result = %#v, want [%p]", adts, one)
	}
}

// Regression pin: nil is a legitimate input, not a type mismatch.
func TestLiteralConverterPreservesNil(t *testing.T) {
	if got := ConvertToInt64Slice(nil); got != nil {
		t.Fatalf("ConvertToInt64Slice(nil) = %#v, want nil", got)
	}
}
`

	tmp := t.TempDir()
	files := map[string][]byte{
		"go.mod":          []byte("module runtimefixture\n\ngo 1.21\n"),
		"runtime.go":      runtimeCode,
		"runtime_test.go": []byte(testCode),
		"user_adt.go":     []byte("package runtimefixture\n\ntype UserADT struct { Kind string }\n"),
	}
	for name, contents := range files {
		if err := os.WriteFile(filepath.Join(tmp, name), contents, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	cmd := exec.Command("go", "test", "-count=1", "./...")
	cmd.Dir = tmp
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("generated runtime tests failed: %v\n%s", err, output)
	}
}

func TestSliceConverterEmittersHaveNoSilentTypeFallback(t *testing.T) {
	source, err := os.ReadFile("codegen_runtime_slices.go")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(source), "if !ok {\n\t\tg.indent++\n\t\tg.writef(\"return nil") != 0 {
		t.Fatal("slice converter emitter retains an if !ok { return nil } fallback")
	}
}
