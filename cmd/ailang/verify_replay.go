package main

import (
	"time"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/smt"
	ailangtesting "github.com/sunholo-data/ailang/internal/testing"
)

// witnessReplayer lets the verifier decide an inconclusive bounded model by
// running the function on the solver's inputs with runtime contract checking
// (smt.WitnessReplayer, internal/testing.WitnessReplayer). The test executor is
// built on first use and reused for every function of the file. Each replay is
// bounded by the per-function solver timeout.
func witnessReplayer(path string, file *ast.File, timeout time.Duration) smt.WitnessReplayer {
	if path == "" || file == nil {
		return nil
	}
	var r *ailangtesting.WitnessReplayer
	return func(funcName string, args []smt.WitnessArg) smt.ReplayOutcome {
		if r == nil {
			r = ailangtesting.NewWitnessReplayer(path, file, timeout)
		}
		in := make([]ailangtesting.ReplayArg, len(args))
		for i, a := range args {
			in[i] = ailangtesting.ReplayArg{Name: a.Name, Literal: a.Literal}
		}
		violated, detail := r.Replay(funcName, in)
		return smt.ReplayOutcome{Violated: violated, Detail: detail}
	}
}
