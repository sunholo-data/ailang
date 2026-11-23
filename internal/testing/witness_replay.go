package testing

import (
	"fmt"
	"strings"
	"time"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/eval"
)

// WitnessReplayer runs a function on concrete inputs and checks its lowered
// requires/ensures at runtime — the ground truth `ailang verify` consults to
// decide an inconclusive bounded model (M-VERIFY-SOUNDNESS-COVERAGE M6). It
// reuses the property-test harness, so the contracts are evaluated exactly as
// `ailang test`'s generated ensures properties evaluate them.
type WitnessReplayer struct {
	exec    *Executor
	file    *ast.File
	timeout time.Duration
}

// ReplayArg is one input: the parameter name and its value as a literal.
type ReplayArg struct {
	Name    string
	Literal ast.Expr
}

// NewWitnessReplayer prepares replay for the module at modulePath (its parsed
// surface file). Each replay is bounded by timeout; a replay that does not
// finish in time decides nothing.
func NewWitnessReplayer(modulePath string, file *ast.File, timeout time.Duration) *WitnessReplayer {
	exec := NewExecutor(modulePath)
	exec.SetSourceFile(file)
	return &WitnessReplayer{exec: exec, file: file, timeout: timeout}
}

// Replay reports violated=true only when every requires clause holds on args
// and some ensures clause is false at runtime. detail says which clause failed,
// or why the replay could not decide (requires not met, a runtime error, an
// unconvertible value, a timeout) — all of which leave the verdict unchanged.
func (w *WitnessReplayer) Replay(funcName string, args []ReplayArg) (violated bool, detail string) {
	type outcome struct {
		violated bool
		detail   string
	}
	done := make(chan outcome, 1)
	go func() {
		v, d := w.replay(funcName, args)
		done <- outcome{v, d}
	}()
	select {
	case o := <-done:
		return o.violated, o.detail
	case <-time.After(w.timeout):
		return false, fmt.Sprintf("did not finish within %s", w.timeout)
	}
}

func (w *WitnessReplayer) replay(funcName string, args []ReplayArg) (bool, string) {
	binding, err := w.exec.ExtractFunctionBinding(funcName, w.file)
	if err != nil {
		return false, fmt.Sprintf("could not load %s: %v", funcName, err)
	}
	meta := w.exec.LastDeclMeta(funcName)
	if meta == nil {
		return false, "no contracts found at runtime"
	}
	params := make([]EnsuresParam, len(args))
	shown := make([]string, len(args))
	for i, a := range args {
		v, err := astExprToCore(a.Literal)
		if err != nil {
			return false, fmt.Sprintf("input %s: %v", a.Name, err)
		}
		params[i] = EnsuresParam{Name: a.Name, Value: v}
		shown[i] = fmt.Sprintf("%s = %s", a.Name, a.Literal.String())
	}
	input := strings.Join(shown, ", ")

	for _, c := range meta.Contracts {
		if c == nil || c.Kind != core.RequiresKind {
			continue
		}
		v, err := w.exec.EvaluateRequiresHarnessFromCore(params, c.Expr)
		if err != nil {
			return false, fmt.Sprintf("requires could not be evaluated on %s: %v", input, err)
		}
		if b, ok := v.(*eval.BoolValue); !ok || !b.Value {
			return false, fmt.Sprintf("the solver's input %s does not satisfy requires at runtime", input)
		}
	}
	clause := 0
	for _, c := range meta.Contracts {
		if c == nil || c.Kind != core.EnsuresKind {
			continue
		}
		clause++
		v, err := w.exec.EvaluateEnsuresHarnessFromCore(*binding, params, c.Expr)
		if err != nil {
			// A runtime error is not a postcondition failure in the verifier's
			// partial-correctness model; it decides nothing here.
			return false, fmt.Sprintf("running %s on %s raised: %v", funcName, input, err)
		}
		b, ok := v.(*eval.BoolValue)
		if !ok {
			return false, fmt.Sprintf("ensures returned %T", v)
		}
		if !b.Value {
			return true, fmt.Sprintf("ensures clause %d is false for %s", clause, input)
		}
	}
	return false, fmt.Sprintf("contracts hold at runtime for %s", input)
}
