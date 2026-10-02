package repl

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	ailerrors "github.com/sunholo-data/ailang/internal/errors"
	"github.com/sunholo-data/ailang/internal/eval"
)

// #1449: `10 / 0` at the prompt used to panic out of the Num[int] dictionary
// and end the REPL session. It now prints RT001 and the session goes on.
func TestREPLIntDivZero(t *testing.T) {
	r := New()
	// What StartWithContext does before the first prompt.
	r.initBuiltins()
	r.importModule("std/prelude", io.Discard)
	var out bytes.Buffer
	r.ProcessExpression("10 / 0", &out)
	if !strings.Contains(out.String(), "RT001: integer division by zero") {
		t.Fatalf("10 / 0 printed %q, want RT001", out.String())
	}
	out.Reset()
	r.ProcessExpression("1 + 1", &out)
	if !strings.HasPrefix(out.String(), "2 ::") {
		t.Fatalf("after the error, 1 + 1 printed %q", out.String())
	}
}

// The module registry (the WASM/playground path) divides through its own
// prelude dictionary, which had no zero guard at all.
func TestREPLIntDivZero_ModuleRegistry(t *testing.T) {
	reg := NewModuleRegistry()
	code := "module test/divzero\n\nexport func divBy(n: int) -> int = 10 / n\n"
	if _, err := reg.LoadModule("test/divzero", code); err != nil {
		t.Fatalf("LoadModule: %v", err)
	}
	_, err := reg.InvokeExport("test/divzero", "divBy", []eval.Value{&eval.IntValue{Value: 0}})
	var dz *ailerrors.DivByZeroError
	if !errors.As(err, &dz) {
		t.Fatalf("divBy(0) error = %T %v, want *DivByZeroError", err, err)
	}
	v, err := reg.InvokeExport("test/divzero", "divBy", []eval.Value{&eval.IntValue{Value: 5}})
	if err != nil || v.(*eval.IntValue).Value != 2 {
		t.Fatalf("divBy(5) = %v, %v; want 2", v, err)
	}
}
