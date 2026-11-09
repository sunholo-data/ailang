package core

import (
	"fmt"
	"sort"

	"github.com/sunholo-data/ailang/internal/ast"
)

// MarkTraceRedactions stamps App.TraceRedact on every App whose surface origin
// is a labelled call site reported by the static IFC pass
// (types.CheckModuleIFCWithTraceSites, M-TRACE-LABEL-AWARE). An App is matched
// by OrigSpan, which the elaborator copies from the surface FuncCall and every
// later Core rewrite preserves, so this runs once on the final Core.
//
// It returns the sites that matched no App, sorted by position (for
// --debug-compile). Such a site was elaborated to something else — an
// operator, a type-class method (DictApp), an erased debug call — which makes
// no function call and so emits no function trace event.
//
// The switch is exhaustive over the CoreExpr kinds registered in gob.go and
// panics on an unknown one: a skipped subtree would be a silent leak.
func MarkTraceRedactions(prog *Program, sites map[ast.Pos]string) (unmatched []ast.Pos) {
	if prog == nil || len(sites) == 0 {
		return nil
	}
	m := traceMarker{sites: sites, matched: map[ast.Pos]bool{}}
	m.walkProgram(prog)
	for pos := range sites {
		if !m.matched[pos] {
			unmatched = append(unmatched, pos)
		}
	}
	sort.Slice(unmatched, func(i, j int) bool {
		a, b := unmatched[i], unmatched[j]
		if a.File != b.File {
			return a.File < b.File
		}
		return a.Offset < b.Offset
	})
	return unmatched
}

// walkProgram visits every declaration and every contract expression.
func (m *traceMarker) walkProgram(prog *Program) {
	for _, d := range prog.Decls {
		m.walk(d)
	}
	for _, meta := range prog.Meta {
		if meta == nil {
			continue
		}
		for _, c := range meta.Contracts {
			if c != nil {
				m.walk(c.Expr)
			}
		}
	}
}

// HasTraceRedactions reports whether any App in prog (including contract
// expressions) carries a TraceRedact label. An engine that cannot honour the
// label (the bytecode VM) uses it to refuse a traced run.
func HasTraceRedactions(prog *Program) bool {
	if prog == nil {
		return false
	}
	m := traceMarker{}
	m.walkProgram(prog)
	return m.found
}

// traceMarker stamps labels from sites (nil when only scanning) and records
// whether it saw any labelled App.
type traceMarker struct {
	sites   map[ast.Pos]string
	matched map[ast.Pos]bool
	found   bool
}

func (m *traceMarker) walkAll(es []CoreExpr) {
	for _, e := range es {
		m.walk(e)
	}
}

func (m *traceMarker) walk(e CoreExpr) {
	switch x := e.(type) {
	case nil:
	case *Var, *VarGlobal, *Lit, *DictRef:
	case *App:
		if label, ok := m.sites[x.OrigSpan]; ok {
			x.TraceRedact = label
			m.matched[x.OrigSpan] = true
		}
		if x.TraceRedact != "" {
			m.found = true
		}
		m.walk(x.Func)
		m.walkAll(x.Args)
	case *Lambda:
		m.walk(x.Body)
	case *Let:
		m.walk(x.Value)
		m.walk(x.Body)
	case *LetRec:
		for _, b := range x.Bindings {
			m.walk(b.Value)
		}
		m.walk(x.Body)
	case *If:
		m.walk(x.Cond)
		m.walk(x.Then)
		m.walk(x.Else)
	case *Match:
		m.walk(x.Scrutinee)
		for _, arm := range x.Arms {
			m.walk(arm.Guard)
			m.walk(arm.Body)
		}
	case *BinOp:
		m.walk(x.Left)
		m.walk(x.Right)
	case *UnOp:
		m.walk(x.Operand)
	case *Record:
		for _, f := range x.Fields {
			m.walk(f)
		}
	case *RecordAccess:
		m.walk(x.Record)
	case *RecordUpdate:
		m.walk(x.Base)
		for _, f := range x.Updates {
			m.walk(f)
		}
	case *List:
		m.walkAll(x.Elements)
	case *Array:
		m.walkAll(x.Elements)
	case *Tuple:
		m.walkAll(x.Elements)
	case *Intrinsic:
		m.walkAll(x.Args)
	case *DictAbs:
		m.walk(x.Body)
	case *DictApp:
		m.walk(x.Dict)
		m.walkAll(x.Args)
	case *Forall:
		m.walk(x.Lo)
		m.walk(x.Hi)
		m.walk(x.Body)
	default:
		panic(fmt.Sprintf("core.MarkTraceRedactions: unhandled CoreExpr %T (a skipped subtree would leak labelled values into traces)", e))
	}
}
