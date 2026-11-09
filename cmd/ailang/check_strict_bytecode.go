package main

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/pipeline"
	"github.com/sunholo-data/ailang/internal/runner"
)

// checkStrictBytecode is `ailang check --strict-bytecode`: after a clean
// type/effect check, also compile the bytecode image and fail with one BC001
// per evaluator-only function the module reaches (M-BYTECODE-STRICT-256-REG-
// CEILING Phase A). Set once at command entry, like checkAgentFormat.
var checkStrictBytecode bool

// addStrictBytecodeErrors appends the strict-VM coverage diagnostics to a
// passing check result, so they flow through the same human / --json /
// --format agent rendering and exit code as every other check error. A
// result that already failed is left alone: there is no image to compile.
func addStrictBytecodeErrors(result *pipeline.Result) {
	if !checkStrictBytecode || len(result.Errors) > 0 {
		return
	}
	entries, err := runner.StrictBytecodeReport(*result)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("strict-bytecode: %w", err))
		return
	}
	result.Errors = append(result.Errors, runner.BC001Errors(entries)...)
}
