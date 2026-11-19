package vm

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/bytecode"
)

// The VM's __show_preview rejects malformed calls loudly (a compiled program
// never makes one; these guard the table entry itself).
func TestBuiltinShowPreviewRejectsBadArgs(t *testing.T) {
	if _, err := builtinShowPreview([]bytecode.Value{bytecode.NewInt(1)}); err == nil || !strings.Contains(err.Error(), "expected 2 args") {
		t.Fatalf("one arg: %v", err)
	}
	if _, err := builtinShowPreview([]bytecode.Value{bytecode.NewInt(1), bytecode.NewString("3")}); err == nil || !strings.Contains(err.Error(), "maxChars") {
		t.Fatalf("string maxChars: %v", err)
	}
	if _, err := builtinShow(nil); err == nil {
		t.Fatal("_show with no argument must fail")
	}
}

// Shapes only the VM can hold: an ADT built without a constructor name (a
// construction-site bug) is shown as the VM's debug form rather than guessed,
// and vm.Show is the same renderer as the builtin.
func TestVMShowOddShapes(t *testing.T) {
	unnamed := bytecode.NewADT(3, "", nil)
	if got := showValue(unnamed); got != unnamed.String() {
		t.Fatalf("unnamed ADT: got %q, want its debug form %q", got, unnamed.String())
	}
	v := bytecode.NewList([]bytecode.Value{bytecode.NewString("a"), bytecode.NewInt(1)})
	if Show(v) != `["a", 1]` || Show(v) != showValue(v) {
		t.Fatalf("vm.Show = %q", Show(v))
	}
	out, err := builtinShow([]bytecode.Value{v})
	if err != nil || out.AsString() != `["a", 1]` {
		t.Fatalf("builtinShow = %q, %v", out.AsString(), err)
	}
}
