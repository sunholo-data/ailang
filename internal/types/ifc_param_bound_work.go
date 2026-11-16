package types

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/ast"
)

// The IFC parameter-bound walk's work budget (#2363 review). Sharing makes the
// common alias shapes linear, but some shapes have exponentially many
// distinct arguments (an alias whose two uses pass differently built
// arguments at every level, `P1[a, b] = (P0[(a, b), b], P0[a, (b, a)])`),
// which no memo can share. The walk then stops at the same size-relative
// budget as unification (workBudget, type_work.go), where n is the written
// size of the parameter type and of the module's type declarations, and
// reports the existing information-flow analysis limit instead of running
// for minutes.

// step counts one addBound call against the walk's budget.
func (w *ifcBoundWalk) step() {
	if w.steps++; w.limit > 0 && w.steps > w.limit {
		panic(ifcAnalysisLimit{reason: fmt.Sprintf("parameter type work: more than %s steps for a written size of %s",
			groupDigits(w.limit), groupDigits(int64(w.size)))})
	}
}

// boundWork is the size and the work budget of one paramBound walk over t.
func (ts *ifcTypes) boundWork(t ast.Type) (size int, limit int64) {
	if ts.written == 0 {
		ts.written = 1
		for _, td := range ts.decls {
			for _, body := range declBodyTypes(td) {
				ts.written += astTypeSize(body)
			}
		}
	}
	size = ts.written + astTypeSize(t)
	return size, workBudget(size)
}

// astTypeSize counts the nodes of a written type.
func astTypeSize(t ast.Type) int {
	n := 1
	switch x := t.(type) {
	case *ast.ListType:
		n += astTypeSize(x.Element)
	case *ast.ArrayType:
		n += astTypeSize(x.Element)
	case *ast.TupleType:
		for _, e := range x.Elements {
			n += astTypeSize(e)
		}
	case *ast.TypeApp:
		for _, e := range x.Args {
			n += astTypeSize(e)
		}
	case *ast.FuncType:
		for _, e := range x.Params {
			n += astTypeSize(e)
		}
		n += astTypeSize(x.Return)
	case *ast.RecordType:
		for _, f := range x.Fields {
			if f != nil {
				n += astTypeSize(f.Type)
			}
		}
	case *ast.LabelledType:
		n += astTypeSize(x.Base)
	}
	return n
}
