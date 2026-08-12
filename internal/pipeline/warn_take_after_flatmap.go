package pipeline

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/elaborate"
)

// TakeAfterFlatMapWarning is a non-blocking warning that strict flatMap
// materializes its complete result before take can bound it.
type TakeAfterFlatMapWarning struct {
	Location ast.Pos
}

var _ elaborate.Warning = (*TakeAfterFlatMapWarning)(nil)

// Position exposes the surface location of the outer take call.
func (w *TakeAfterFlatMapWarning) Position() (line, col int) {
	return w.Location.Line, w.Location.Column
}

// String renders the warning and its bounded-memory replacement.
func (w *TakeAfterFlatMapWarning) String() string {
	return fmt.Sprintf(
		"LIST_TAKE_AFTER_FLATMAP at %s\n"+
			"  take(n, flatMap(f, xs)) materializes the entire flatMap result before take runs.\n"+
			"  Fix: use takeFlatMap(n, f, xs) to stop producing values once the prefix is full.",
		w.Location.String(),
	)
}

// Code returns the stable diagnostic code.
func (w *TakeAfterFlatMapWarning) Code() string { return "LIST_TAKE_AFTER_FLATMAP" }

// DetectTakeAfterFlatMap finds direct take(n, flatMap(f, xs)) calls on resolved
// std/list globals. Calls with an intervening expression intentionally do not
// match.
func DetectTakeAfterFlatMap(prog *core.Program) []elaborate.Warning {
	if prog == nil {
		return nil
	}
	var warnings []elaborate.Warning
	WalkCore(prog, func(expr core.CoreExpr) {
		if outer, ok := directTakeFlatMap(expr); ok {
			warnings = append(warnings, &TakeAfterFlatMapWarning{Location: outer.OriginalSpan()})
		}
	})
	return warnings
}

// directTakeFlatMap recognizes both the source-shaped nested application and
// its ANF representation: let tmp = flatMap(f, xs) in take(n, tmp). Requiring
// the take application to be the immediate let body preserves the direct-call
// scope rule.
//
// MEASURED (V1 iteration 186): elaboration always produces the ANF form, so the
// isTakeOfFlatMapApp arm below is NOT exercised by any test in this repo —
// neutering it alone leaves ./internal/diag and ./internal/pipeline fully green,
// while neutering the ANF arm reds TestTakeAfterFlatMapWarningFixtures/direct_trap
// with "got warnings: []". The nested arm is retained deliberately as a cheap
// guard should elaboration stop ANF-ing applications; it is named here rather
// than left to imply coverage it does not have.
func directTakeFlatMap(expr core.CoreExpr) (*core.App, bool) {
	if outer, ok := expr.(*core.App); ok && isTakeOfFlatMapApp(outer) {
		return outer, true
	}
	binding, ok := expr.(*core.Let)
	if !ok {
		return nil, false
	}
	inner, ok := binding.Value.(*core.App)
	if !ok || len(inner.Args) != 2 || !isListGlobal(inner.Func, "flatMap") {
		return nil, false
	}
	outer, ok := binding.Body.(*core.App)
	if !ok || len(outer.Args) != 2 || !isListGlobal(outer.Func, "take") {
		return nil, false
	}
	arg, ok := outer.Args[1].(*core.Var)
	return outer, ok && arg.Name == binding.Name
}

func isTakeOfFlatMapApp(outer *core.App) bool {
	if len(outer.Args) != 2 || !isListGlobal(outer.Func, "take") {
		return false
	}
	inner, ok := outer.Args[1].(*core.App)
	return ok && len(inner.Args) == 2 && isListGlobal(inner.Func, "flatMap")
}

func isListGlobal(expr core.CoreExpr, name string) bool {
	global, ok := expr.(*core.VarGlobal)
	return ok && global.Ref.Module == "std/list" && global.Ref.Name == name
}
