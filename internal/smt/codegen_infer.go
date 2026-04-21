package smt

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/core"
)

// inferResultSort tries to determine the result sort from the function parameters/body.
// Falls back to "Int" if unable to determine.
func inferResultSort(params []FunctionParam, body core.CoreExpr, ctx *SMTContext, adtTypes map[string][]ADTVariant) string {
	// Build reverse lookup: constructor name → type name
	ctorToType := make(map[string]string)
	for typeName, variants := range adtTypes {
		for _, v := range variants {
			ctorToType[v.Name] = typeName
		}
	}

	return inferResultSortInner(body, ctx, ctorToType)
}

func inferResultSortInner(body core.CoreExpr, ctx *SMTContext, ctorToType map[string]string) string {
	if body == nil {
		return "Int"
	}
	switch b := body.(type) {
	case *core.Lit:
		switch b.Kind {
		case core.IntLit:
			return "Int"
		case core.FloatLit:
			return "Real"
		case core.BoolLit:
			return "Bool"
		case core.StringLit:
			return "String"
		}
	case *core.Var:
		if sort, ok := ctx.Variables[b.Name]; ok {
			return sort
		}
	case *core.VarGlobal:
		// Check if it's a constructor reference
		name := stripConstructorPrefix(b.Ref.Name)
		if typeName, ok := ctorToType[name]; ok {
			return typeName
		}
	case *core.If:
		return inferResultSortInner(b.Then, ctx, ctorToType)
	case *core.Let:
		return inferResultSortInner(b.Body, ctx, ctorToType)
	case *core.Match:
		if len(b.Arms) > 0 {
			return inferResultSortInner(b.Arms[0].Body, ctx, ctorToType)
		}
	case *core.App:
		// Constructor application — check if func is a constructor
		if vg, ok := b.Func.(*core.VarGlobal); ok {
			name := stripConstructorPrefix(vg.Ref.Name)
			if typeName, ok := ctorToType[name]; ok {
				return typeName
			}
		}
	case *core.Record:
		// Record construction returns the record sort
		if info := lookupRecordByFields(b.Fields); info != nil {
			return info.SortName
		}
	case *core.RecordAccess:
		// Field access on a record — need to look at the record's type
		// and the field's sort
		return "Int" // conservative fallback
	case *core.List:
		// List expressions return a Seq sort
		// Try to determine element sort from first element
		if len(b.Elements) > 0 {
			elemSort := inferResultSortInner(b.Elements[0], ctx, ctorToType)
			return fmt.Sprintf("(Seq %s)", elemSort)
		}
		return "(Seq Int)" // default
	}
	return "Int"
}
