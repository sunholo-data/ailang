package repl

import (
	"fmt"

	ailerrors "github.com/sunholo-data/ailang/internal/errors"
	"github.com/sunholo-data/ailang/internal/eval"
)

// intDivFn is Num[Int].div for the REPL's own dictionaries (the REPL
// instances and the module registry's prelude). A zero divisor is RT001, the
// error the evaluator and VM raise, not a Go panic that ends the session
// (#1449). It is shared so the three dictionaries cannot drift apart again:
// one panicked and two divided unguarded.
func intDivFn(args []eval.Value) (eval.Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("expected 2 arguments, got %d", len(args))
	}
	x, ok1 := args[0].(*eval.IntValue)
	y, ok2 := args[1].(*eval.IntValue)
	if !ok1 || !ok2 {
		return nil, fmt.Errorf("expected int arguments")
	}
	if err := ailerrors.CheckIntDivisor(ailerrors.OpDivision, int64(y.Value)); err != nil {
		return nil, err
	}
	return &eval.IntValue{Value: x.Value / y.Value}, nil
}
