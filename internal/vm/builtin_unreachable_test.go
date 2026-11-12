package vm

import (
	"errors"
	"testing"

	"github.com/sunholo-data/ailang/internal/bytecode"
	ailerrors "github.com/sunholo-data/ailang/internal/errors"
)

// unreachable(msg) on the VM raises the evaluator's RT010 error type and text.
func TestBuiltinUnreachable(t *testing.T) {
	_, err := builtinUnreachable([]bytecode.Value{bytecode.NewString("no blue here")})
	var ue *ailerrors.UnreachableError
	if !errors.As(err, &ue) || ue.Msg != "no blue here" || err.Error() != "RT010: unreachable code reached: no blue here" {
		t.Fatalf("got %v", err)
	}
	if _, err := builtinUnreachable([]bytecode.Value{bytecode.NewInt(1)}); err == nil || errors.As(err, &ue) {
		t.Fatalf("a non-string argument is an internal error, not RT010: %v", err)
	}
}
