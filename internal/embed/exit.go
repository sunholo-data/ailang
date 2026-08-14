package embed

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/eval"
)

// ExitError reports that an embedded AILANG program called exit(code).
//
// The exit() builtin raises a *eval.EvalExitCode sentinel panic
// (internal/effects/io.go). The CLI recovers it and maps it onto os.Exit;
// an embedding host has no process exit to map onto, so the embed layer
// surfaces it as this typed error instead. Callers that want the CLI's
// batch semantics can branch on Code:
//
//	if ee := new(ExitError); errors.As(err, &ee) && ee.Code == 0 {
//	    // program terminated normally
//	}
//
// Note that Code 0 is reported as an ExitError too, not as a nil error:
// the host cannot otherwise distinguish "the function returned unit" from
// "the program called exit(0) and never reached its return". Collapsing
// exit(0) to nil would discard information the caller cannot recover.
type ExitError struct {
	// Code is the argument the program passed to exit().
	Code int
}

func (e *ExitError) Error() string {
	return fmt.Sprintf("program called exit(%d)", e.Code)
}

// recoverProgramExit runs one entrypoint call and maps the exit() sentinel
// panic onto a typed *ExitError, so that an embedded module calling exit()
// cannot take the host process down with it.
//
// Three branches, each independently pinned by internal/embed/exit_test.go:
//   - call returns without panicking: its value and error pass through unchanged
//   - the panic is *eval.EvalExitCode: reported as *ExitError, host survives
//   - any other panic: re-raised unchanged, so a real crash stays loud
//
// Split out from the Call methods so every branch is reachable from a unit
// test without standing up a module runtime.
func recoverProgramExit(call func() (eval.Value, error)) (val eval.Value, err error) {
	defer func() {
		r := recover()
		if r == nil {
			return
		}
		ec, ok := r.(*eval.EvalExitCode)
		if !ok {
			panic(r) // re-panic: not an exit(), so it is a real crash
		}
		val, err = nil, &ExitError{Code: ec.Code}
	}()
	return call()
}
