package main

import (
	"errors"
	"fmt"
)

// errGoEmitterRemoved is what `ailang compile` (and so `compile --emit-go`
// and `compile --emit-go-v2`) answers. The command existed only to emit Go;
// the emitter was removed so that v1.0.0 has two engines that agree, the
// evaluator and the bytecode VM. The row stays, hidden, so an old script gets
// this message and exit 1 instead of "unknown command".
var errGoEmitterRemoved = errors.New("`ailang compile` and the Go emitter (--emit-go, --emit-go-v2) were removed after v0.60.5. " +
	"AILANG runs programs with `ailang run` (add --bytecode for the VM). " +
	"See https://ailang.sunholo.com/docs/reference/stability")

// compileRemoved fails every invocation with errGoEmitterRemoved, whatever
// its flags. A help flag anywhere in the arguments (`compile --emit-go
// --help` included) is answered with the same text on stdout and exit 0, as
// --help is everywhere (TestCLI_HelpExitsZeroEverywhere).
func compileRemoved(args []string) error {
	for _, a := range args {
		switch a {
		case "-h", "--h", "-help", "--help":
			fmt.Println(errGoEmitterRemoved)
			return nil
		}
	}
	return errGoEmitterRemoved
}
