package builtins

import (
	"errors"
	"fmt"

	"github.com/sunholo-data/ailang/internal/eval"
)

// callbackErr wraps an error returned by an AILANG callback with the builtin's
// context, except RT_REC_003, which passes through unchanged. Deep recursion
// through a callback fails at the innermost level, and a per-level prefix turned
// that one line into megabytes of "callback error at index 0:" (#1317).
func callbackErr(err error, format string, args ...interface{}) error {
	var rle *eval.RecursionLimitError
	if errors.As(err, &rle) {
		return err
	}
	return fmt.Errorf(format+": %w", append(args, err)...)
}
