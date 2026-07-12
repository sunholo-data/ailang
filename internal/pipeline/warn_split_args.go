package pipeline

import (
	"fmt"
	"unicode/utf8"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/elaborate"
)

// M-DX-SPLIT-ARG: Compile-time, non-blocking warning for reversed same-typed
// arguments.
//
// The "same-typed argument swap" bug class: when both arguments of a function
// share a type, the type system provides no guardrail against passing them in
// the wrong order. The canonical case is `split(s, delimiter)` (data-first):
// `split("/", name)` silently returns `["/"]` (it splits the literal "/" by the
// contents of `name`) instead of splitting `name` by "/".
//
// This pass runs over the *elaborated* Core program, BEFORE lowering rewrites
// `split` into the `_str_split` builtin. At that stage a call to an imported
// `std/string.split` is `App{Func: VarGlobal{Module:"std/string", Name:"split"},
// Args:[a, b]}`. A user-defined local `split` elaborates to a plain
// `*core.Var{Name:"split"}` (never a VarGlobal), so matching on VarGlobal with a
// module guard structurally excludes user functions.
//
// The pass is extensible: swapTraps is a table keyed by the imported symbol.
// Only std/string.split is armed today.

// argOrderDetector inspects a matched 2-arg call and returns a warning if the
// heuristic fires, or nil otherwise.
type argOrderDetector func(app *core.App) *ArgOrderWarning

// swapTrap describes a function whose same-typed arguments are prone to being
// swapped, and how to detect a likely swap.
type swapTrap struct {
	module string
	name   string
	detect argOrderDetector
}

// swapTraps is the extensible registry of same-typed-arg swap traps.
// Only std/string.split is armed. Add more entries (find, contains, ...) here.
var swapTraps = []swapTrap{
	{
		module: "std/string",
		name:   "split",
		detect: detectSplitReversed,
	},
}

// swapTrapIndex is a fast lookup from GlobalRef{module,name} to its detector,
// built once from swapTraps.
var swapTrapIndex = buildSwapTrapIndex()

func buildSwapTrapIndex() map[core.GlobalRef]argOrderDetector {
	idx := make(map[core.GlobalRef]argOrderDetector, len(swapTraps))
	for _, t := range swapTraps {
		idx[core.GlobalRef{Module: t.module, Name: t.name}] = t.detect
	}
	return idx
}

// ArgOrderWarning is a non-blocking warning that a same-typed-argument function
// was likely called with its arguments reversed. It satisfies elaborate.Warning.
type ArgOrderWarning struct {
	Location   ast.Pos // Surface location of the offending call
	FuncName   string  // e.g. "split"
	Delimiter  string  // the short literal that was passed as arg0
	Arg1String string  // String() rendering of the (non-literal) arg1
}

// compile-time assertion that ArgOrderWarning satisfies the warning interface.
var _ elaborate.Warning = (*ArgOrderWarning)(nil)

// Position exposes 1-indexed line/col for LSP diagnostic placement.
func (w *ArgOrderWarning) Position() (line, col int) {
	return w.Location.Line, w.Location.Column
}

// String renders the warning in the design-doc format (hint + note).
func (w *ArgOrderWarning) String() string {
	return fmt.Sprintf(
		"warning: %s(s, delimiter) takes the string first, delimiter second.\n"+
			"  --> %s\n"+
			"  = hint: did you mean %s(%s, %q)?\n"+
			"  = note: %s(%q, %s) splits %q by %s, which likely returns [%q]",
		w.FuncName,
		w.Location.String(),
		w.FuncName, w.Arg1String, w.Delimiter,
		w.FuncName, w.Delimiter, w.Arg1String, w.Delimiter, w.Arg1String, w.Delimiter,
	)
}

// detectSplitReversed applies the split heuristic to a 2-arg split call:
// arg0 is a StringLit of 1-3 runes AND arg1 is NOT a StringLit.
func detectSplitReversed(app *core.App) *ArgOrderWarning {
	if len(app.Args) != 2 {
		return nil
	}
	lit, ok := app.Args[0].(*core.Lit)
	if !ok || lit.Kind != core.StringLit {
		return nil
	}
	s, ok := lit.Value.(string)
	if !ok {
		return nil
	}
	// Short literal (1-3 runes) is a likely delimiter, not data.
	if n := utf8.RuneCountInString(s); n < 1 || n > 3 {
		return nil
	}
	// arg1 a string literal too => can't tell (edge case), don't warn.
	if lit1, ok := app.Args[1].(*core.Lit); ok && lit1.Kind == core.StringLit {
		return nil
	}
	return &ArgOrderWarning{
		Location:   app.OriginalSpan(),
		FuncName:   "split",
		Delimiter:  s,
		Arg1String: app.Args[1].String(),
	}
}

// DetectArgOrderWarnings walks an elaborated Core program and returns any
// same-typed-arg swap warnings. It is read-only over Core and must be called
// BEFORE lowering (while split is still an imported VarGlobal).
func DetectArgOrderWarnings(prog *core.Program) []elaborate.Warning {
	if prog == nil {
		return nil
	}
	var out []elaborate.Warning
	for _, decl := range prog.Decls {
		walkArgOrder(decl, &out)
	}
	return out
}

// walkArgOrder recursively visits every Core expression, applying swap-trap
// detectors to matching App nodes.
func walkArgOrder(e core.CoreExpr, out *[]elaborate.Warning) {
	if e == nil {
		return
	}
	switch n := e.(type) {
	case *core.App:
		if vg, ok := n.Func.(*core.VarGlobal); ok {
			if detect, ok := swapTrapIndex[vg.Ref]; ok {
				if w := detect(n); w != nil {
					*out = append(*out, w)
				}
			}
		}
		walkArgOrder(n.Func, out)
		for _, a := range n.Args {
			walkArgOrder(a, out)
		}
	case *core.Lambda:
		walkArgOrder(n.Body, out)
	case *core.Let:
		walkArgOrder(n.Value, out)
		walkArgOrder(n.Body, out)
	case *core.LetRec:
		for _, b := range n.Bindings {
			walkArgOrder(b.Value, out)
		}
		walkArgOrder(n.Body, out)
	case *core.If:
		walkArgOrder(n.Cond, out)
		walkArgOrder(n.Then, out)
		walkArgOrder(n.Else, out)
	case *core.Match:
		walkArgOrder(n.Scrutinee, out)
		for _, arm := range n.Arms {
			walkArgOrder(arm.Guard, out)
			walkArgOrder(arm.Body, out)
		}
	case *core.BinOp:
		walkArgOrder(n.Left, out)
		walkArgOrder(n.Right, out)
	case *core.UnOp:
		walkArgOrder(n.Operand, out)
	case *core.Intrinsic:
		for _, a := range n.Args {
			walkArgOrder(a, out)
		}
	case *core.Record:
		for _, v := range n.Fields {
			walkArgOrder(v, out)
		}
	case *core.RecordAccess:
		walkArgOrder(n.Record, out)
	case *core.RecordUpdate:
		walkArgOrder(n.Base, out)
		for _, v := range n.Updates {
			walkArgOrder(v, out)
		}
	case *core.List:
		for _, el := range n.Elements {
			walkArgOrder(el, out)
		}
	case *core.Array:
		for _, el := range n.Elements {
			walkArgOrder(el, out)
		}
	case *core.Tuple:
		for _, el := range n.Elements {
			walkArgOrder(el, out)
		}
	default:
		// Atomic nodes (Var, VarGlobal, Lit) and any unhandled leaf: nothing to
		// recurse into.
	}
}
