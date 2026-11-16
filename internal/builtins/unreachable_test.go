package builtins

import (
	"errors"
	"strings"
	"testing"

	ailerrors "github.com/sunholo-data/ailang/internal/errors"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/types"
)

// unreachable : ∀α. string -> α, pure; RT010 on the evaluator; generated Go
// panics with the same text.
func TestUnreachableBuiltin(t *testing.T) {
	fn, ok := makeUnreachableType().(*types.TFunc2)
	if !ok || len(fn.Params) != 1 || fn.Params[0] != types.TString {
		t.Fatalf("type = %v", makeUnreachableType())
	}
	if _, isVar := fn.Return.(*types.TVar2); !isVar {
		t.Errorf("result must be a type variable, got %v", fn.Return)
	}
	_, err := unreachableImpl(nil, []eval.Value{&eval.StringValue{Value: "why"}})
	var ue *ailerrors.UnreachableError
	if !errors.As(err, &ue) || ue.Code() != ailerrors.RT010 || err.Error() != "RT010: unreachable code reached: why" {
		t.Fatalf("got %v", err)
	}
	if _, err := unreachableImpl(nil, []eval.Value{&eval.IntValue{Value: 1}}); err == nil || errors.As(err, &ue) {
		t.Fatalf("a non-string argument is an internal error, not RT010: %v", err)
	}
	spec := GetCodegenSpec("unreachable")
	if spec == nil || spec.Helper == nil || !strings.Contains(spec.Helper.Body, `panic("RT010: unreachable code reached: "`) {
		t.Fatalf("Go codegen spec = %+v", spec)
	}
}
