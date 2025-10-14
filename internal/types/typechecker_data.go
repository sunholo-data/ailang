package types

import (
	"fmt"

	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/typedast"
)

// inferRecord infers type of record construction
func (tc *CoreTypeChecker) inferRecord(ctx *InferenceContext, rec *core.Record) (*typedast.TypedRecord, *TypeEnv, error) {
	fields := make(map[string]typedast.TypedNode)
	fieldTypes := make(map[string]Type)
	var allEffects []*Row

	for name, value := range rec.Fields {
		valueNode, _, err := tc.inferCore(ctx, value)
		if err != nil {
			return nil, ctx.env, err
		}
		fields[name] = valueNode
		fieldTypes[name] = getType(valueNode)
		allEffects = append(allEffects, getEffectRow(valueNode))
	}

	// Create record type - use TRecord2 if flag is set
	var recordType Type
	if tc.useRecordsV2 {
		recordType = &TRecord2{
			Row: &Row{
				Kind:   RecordRow,
				Labels: fieldTypes,
				Tail:   nil, // Closed record literal
			},
		}
	} else {
		recordType = &TRecord{
			Fields: fieldTypes,
			Row: &Row{
				Kind:   RecordRow,
				Labels: fieldTypes,
				Tail:   nil, // Closed record for now
			},
		}
	}

	return &typedast.TypedRecord{
		TypedExpr: typedast.TypedExpr{
			NodeID:    rec.ID(),
			Span:      rec.Span(),
			Type:      recordType,
			EffectRow: combineEffectList(allEffects),
			Core:      rec,
		},
		Fields: fields,
	}, ctx.env, nil
}

// inferRecordAccess infers type of field access
func (tc *CoreTypeChecker) inferRecordAccess(ctx *InferenceContext, acc *core.RecordAccess) (*typedast.TypedRecordAccess, *TypeEnv, error) {
	// Infer record type
	recordNode, _, err := tc.inferCore(ctx, acc.Record)
	if err != nil {
		return nil, ctx.env, err
	}

	// Create fresh type variable for field
	fieldType := ctx.freshTypeVar()
	rowVar := &RowVar{Name: "r", Kind: RecordRow}

	// Expected record type with field - use TRecordOpen for subsumption
	// This allows {id:int} to unify with {id:int, email:string}
	expectedRecord := &TRecordOpen{
		Fields: map[string]Type{acc.Field: fieldType},
		Row:    rowVar, // Mark as open for unknown fields
	}

	ctx.addConstraint(TypeEq{
		Left:  getType(recordNode),
		Right: expectedRecord,
		Path:  []string{"field access at " + acc.Span().String()},
	})

	return &typedast.TypedRecordAccess{
		TypedExpr: typedast.TypedExpr{
			NodeID:    acc.ID(),
			Span:      acc.Span(),
			Type:      fieldType,
			EffectRow: getEffectRow(recordNode),
			Core:      acc,
		},
		Record: recordNode,
		Field:  acc.Field,
	}, ctx.env, nil
}

// inferList infers type of list construction
func (tc *CoreTypeChecker) inferList(ctx *InferenceContext, list *core.List) (*typedast.TypedList, *TypeEnv, error) {
	if len(list.Elements) == 0 {
		// Empty list - polymorphic
		elemType := ctx.freshTypeVar()
		return &typedast.TypedList{
			TypedExpr: typedast.TypedExpr{
				NodeID:    list.ID(),
				Span:      list.Span(),
				Type:      &TList{Element: elemType},
				EffectRow: EmptyEffectRow(),
				Core:      list,
			},
			Elements: nil,
		}, ctx.env, nil
	}

	// Non-empty list - all elements must have same type
	var elements []typedast.TypedNode
	var allEffects []*Row

	firstElem, _, err := tc.inferCore(ctx, list.Elements[0])
	if err != nil {
		return nil, ctx.env, err
	}
	elements = append(elements, firstElem)
	allEffects = append(allEffects, getEffectRow(firstElem))
	elemType := getType(firstElem)

	for i := 1; i < len(list.Elements); i++ {
		elemNode, _, err := tc.inferCore(ctx, list.Elements[i])
		if err != nil {
			return nil, ctx.env, err
		}
		elements = append(elements, elemNode)
		allEffects = append(allEffects, getEffectRow(elemNode))

		// All elements must have same type
		ctx.addConstraint(TypeEq{
			Left:  getType(elemNode),
			Right: elemType,
			Path:  []string{fmt.Sprintf("list element %d at %s", i, list.Span())},
		})
	}

	return &typedast.TypedList{
		TypedExpr: typedast.TypedExpr{
			NodeID:    list.ID(),
			Span:      list.Span(),
			Type:      &TList{Element: elemType},
			EffectRow: combineEffectList(allEffects),
			Core:      list,
		},
		Elements: elements,
	}, ctx.env, nil
}

// inferTuple infers type of tuple construction
func (tc *CoreTypeChecker) inferTuple(ctx *InferenceContext, tuple *core.Tuple) (*typedast.TypedTuple, *TypeEnv, error) {
	// Infer types for all elements
	var elements []typedast.TypedNode
	var elemTypes []Type
	var allEffects []*Row

	for _, elem := range tuple.Elements {
		elemNode, _, err := tc.inferCore(ctx, elem)
		if err != nil {
			return nil, ctx.env, err
		}
		elements = append(elements, elemNode)
		elemTypes = append(elemTypes, getType(elemNode))
		allEffects = append(allEffects, getEffectRow(elemNode))
	}

	return &typedast.TypedTuple{
		TypedExpr: typedast.TypedExpr{
			NodeID:    tuple.ID(),
			Span:      tuple.Span(),
			Type:      &TTuple{Elements: elemTypes},
			EffectRow: combineEffectList(allEffects),
			Core:      tuple,
		},
		Elements: elements,
	}, ctx.env, nil
}

// Helper functions

// isCoreValue checks if expression is a syntactic value (for value restriction)
func isCoreValue(expr core.CoreExpr) bool {
	switch e := expr.(type) {
	case *core.Lit, *core.Lambda:
		return true
	case *core.Var:
		// Variables are values if bound to values
		return true
	case *core.Record:
		// Record is value if all fields are values
		for _, field := range e.Fields {
			if !isCoreValue(field) {
				return false
			}
		}
		return true
	case *core.List:
		// List is value if all elements are values
		for _, elem := range e.Elements {
			if !isCoreValue(elem) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

// combineEffects combines two effect rows
func combineEffects(e1, e2 *Row) *Row {
	if e1 == nil || (len(e1.Labels) == 0 && e1.Tail == nil) {
		return e2
	}
	if e2 == nil || (len(e2.Labels) == 0 && e2.Tail == nil) {
		return e1
	}

	// Combine labels
	combined := make(map[string]Type)
	for k, v := range e1.Labels {
		combined[k] = v
	}
	for k, v := range e2.Labels {
		combined[k] = v
	}

	// For now, ignore tail variables in combination
	// Full implementation would handle row unification
	return &Row{
		Kind:   EffectRow,
		Labels: combined,
		Tail:   nil,
	}
}

// combineEffectList combines multiple effect rows
func combineEffectList(effects []*Row) *Row {
	result := EmptyEffectRow()
	for _, e := range effects {
		result = combineEffects(result, e)
	}
	return result
}

// toInterfaceSlice converts []Type to []interface{}
func toInterfaceSlice(types []Type) []interface{} {
	result := make([]interface{}, len(types))
	for i, t := range types {
		result[i] = t
	}
	return result
}

// getType safely extracts Type from interface{}
func getType(node typedast.TypedNode) Type {
	if t := node.GetType(); t != nil {
		return t.(Type)
	}
	return nil
}

// getEffectRow safely extracts Row from interface{}
func getEffectRow(node typedast.TypedNode) *Row {
	if r := node.GetEffectRow(); r != nil {
		return r.(*Row)
	}
	return EmptyEffectRow()
}
