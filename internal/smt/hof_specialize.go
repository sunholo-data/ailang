package smt

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/core"
)

// SubstituteLambdaVar replaces all free occurrences of paramName in body
// with the replacement expression. Respects variable shadowing by Lambda/Let.
func SubstituteLambdaVar(body core.CoreExpr, paramName string, replacement core.CoreExpr) core.CoreExpr {
	if body == nil {
		return nil
	}
	switch e := body.(type) {
	case *core.Var:
		if e.Name == paramName {
			return replacement
		}
		return e
	case *core.VarGlobal:
		return e
	case *core.Lit:
		return e
	case *core.Lambda:
		for _, p := range e.Params {
			if p == paramName {
				return e // shadowed
			}
		}
		newBody := SubstituteLambdaVar(e.Body, paramName, replacement)
		if newBody == e.Body {
			return e
		}
		return &core.Lambda{Params: e.Params, Body: newBody}
	case *core.App:
		newFunc := SubstituteLambdaVar(e.Func, paramName, replacement)
		newArgs := substituteLambdaVarSlice(e.Args, paramName, replacement)
		if newFunc == e.Func && sliceUnchanged(newArgs, e.Args) {
			return e
		}
		return &core.App{Func: newFunc, Args: newArgs}
	case *core.If:
		newCond := SubstituteLambdaVar(e.Cond, paramName, replacement)
		newThen := SubstituteLambdaVar(e.Then, paramName, replacement)
		newElse := SubstituteLambdaVar(e.Else, paramName, replacement)
		if newCond == e.Cond && newThen == e.Then && newElse == e.Else {
			return e
		}
		return &core.If{Cond: newCond, Then: newThen, Else: newElse}
	case *core.Let:
		newValue := SubstituteLambdaVar(e.Value, paramName, replacement)
		if e.Name == paramName {
			// Shadowed in body
			if newValue == e.Value {
				return e
			}
			return &core.Let{Name: e.Name, Value: newValue, Body: e.Body}
		}
		newBody := SubstituteLambdaVar(e.Body, paramName, replacement)
		if newValue == e.Value && newBody == e.Body {
			return e
		}
		return &core.Let{Name: e.Name, Value: newValue, Body: newBody}
	case *core.LetRec:
		// Check if any binding shadows the param
		for _, b := range e.Bindings {
			if b.Name == paramName {
				return e
			}
		}
		newBindings := make([]core.RecBinding, len(e.Bindings))
		changed := false
		for i, b := range e.Bindings {
			newVal := SubstituteLambdaVar(b.Value, paramName, replacement)
			newBindings[i] = core.RecBinding{Name: b.Name, Value: newVal}
			if newVal != b.Value {
				changed = true
			}
		}
		newBody := SubstituteLambdaVar(e.Body, paramName, replacement)
		if !changed && newBody == e.Body {
			return e
		}
		return &core.LetRec{Bindings: newBindings, Body: newBody}
	case *core.Match:
		newScrutinee := SubstituteLambdaVar(e.Scrutinee, paramName, replacement)
		newArms := make([]core.MatchArm, len(e.Arms))
		changed := newScrutinee != e.Scrutinee
		for i, arm := range e.Arms {
			newArmBody := SubstituteLambdaVar(arm.Body, paramName, replacement)
			var newGuard core.CoreExpr
			if arm.Guard != nil {
				newGuard = SubstituteLambdaVar(arm.Guard, paramName, replacement)
			}
			newArms[i] = core.MatchArm{
				Pattern: arm.Pattern,
				Guard:   newGuard,
				Body:    newArmBody,
			}
			if newArmBody != arm.Body || newGuard != arm.Guard {
				changed = true
			}
		}
		if !changed {
			return e
		}
		return &core.Match{Scrutinee: newScrutinee, Arms: newArms}
	case *core.BinOp:
		newLeft := SubstituteLambdaVar(e.Left, paramName, replacement)
		newRight := SubstituteLambdaVar(e.Right, paramName, replacement)
		if newLeft == e.Left && newRight == e.Right {
			return e
		}
		return &core.BinOp{Op: e.Op, Left: newLeft, Right: newRight}
	case *core.UnOp:
		newOperand := SubstituteLambdaVar(e.Operand, paramName, replacement)
		if newOperand == e.Operand {
			return e
		}
		return &core.UnOp{Op: e.Op, Operand: newOperand}
	case *core.Intrinsic:
		newArgs := substituteLambdaVarSlice(e.Args, paramName, replacement)
		if sliceUnchanged(newArgs, e.Args) {
			return e
		}
		return &core.Intrinsic{Op: e.Op, Args: newArgs}
	case *core.Record:
		newFields := make(map[string]core.CoreExpr, len(e.Fields))
		changed := false
		for k, v := range e.Fields {
			newV := SubstituteLambdaVar(v, paramName, replacement)
			newFields[k] = newV
			if newV != v {
				changed = true
			}
		}
		if !changed {
			return e
		}
		return &core.Record{Fields: newFields}
	case *core.RecordAccess:
		newRecord := SubstituteLambdaVar(e.Record, paramName, replacement)
		if newRecord == e.Record {
			return e
		}
		return &core.RecordAccess{Record: newRecord, Field: e.Field}
	case *core.RecordUpdate:
		newBase := SubstituteLambdaVar(e.Base, paramName, replacement)
		newUpdates := make(map[string]core.CoreExpr, len(e.Updates))
		changed := newBase != e.Base
		for k, v := range e.Updates {
			newV := SubstituteLambdaVar(v, paramName, replacement)
			newUpdates[k] = newV
			if newV != v {
				changed = true
			}
		}
		if !changed {
			return e
		}
		return &core.RecordUpdate{Base: newBase, Updates: newUpdates}
	case *core.List:
		newElems := substituteLambdaVarSlice(e.Elements, paramName, replacement)
		if sliceUnchanged(newElems, e.Elements) {
			return e
		}
		return &core.List{Elements: newElems}
	case *core.Tuple:
		newElems := substituteLambdaVarSlice(e.Elements, paramName, replacement)
		if sliceUnchanged(newElems, e.Elements) {
			return e
		}
		return &core.Tuple{Elements: newElems}
	case *core.DictApp:
		newDict := SubstituteLambdaVar(e.Dict, paramName, replacement)
		newArgs := substituteLambdaVarSlice(e.Args, paramName, replacement)
		if newDict == e.Dict && sliceUnchanged(newArgs, e.Args) {
			return e
		}
		return &core.DictApp{Dict: newDict, Args: newArgs}
	case *core.DictAbs:
		newBody := SubstituteLambdaVar(e.Body, paramName, replacement)
		if newBody == e.Body {
			return e
		}
		return &core.DictAbs{Params: e.Params, Body: newBody}
	case *core.DictRef:
		return e
	default:
		return e
	}
}

func substituteLambdaVarSlice(exprs []core.CoreExpr, paramName string, replacement core.CoreExpr) []core.CoreExpr {
	result := make([]core.CoreExpr, len(exprs))
	for i, e := range exprs {
		result[i] = SubstituteLambdaVar(e, paramName, replacement)
	}
	return result
}

// SpecializeMap transforms map(\x -> body, xs) into a recursive function:
//
//	map_spec(xs) =
//	  if seq.len(xs) == 0 then (as seq.empty (Seq Int))
//	  else let h = seq.nth(xs, 0) in
//	       let t = seq.extract(xs, 1, seq.len(xs) - 1) in
//	       (seq.++ (seq.unit body[x -> h]) map_spec(t))
//
// Returns the specialized body and the specialized function name.
func SpecializeMap(funcName string, lambda *core.Lambda, listParam string) (core.CoreExpr, string) {
	specName := fmt.Sprintf("_map_spec_%s", funcName)

	if len(lambda.Params) == 0 {
		return nil, ""
	}
	paramName := lambda.Params[0]

	xsVar := &core.Var{Name: listParam}
	hVar := &core.Var{Name: "_hof_h"}
	tVar := &core.Var{Name: "_hof_t"}

	// Substitute x -> h in the lambda body
	bodyWithH := SubstituteLambdaVar(lambda.Body, paramName, hVar)

	specBody := &core.If{
		// Condition: _list_length(xs) == 0
		Cond: &core.App{
			Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "eq_Int"}},
			Args: []core.CoreExpr{
				&core.App{
					Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "_list_length"}},
					Args: []core.CoreExpr{xsVar},
				},
				&core.Lit{Kind: core.IntLit, Value: int64(0)},
			},
		},
		// Then: empty list
		Then: &core.List{Elements: nil},
		// Else: let h = head(xs) in let t = tail(xs) in body[x->h] :: recurse(t)
		Else: &core.Let{
			Name: "_hof_h",
			Value: &core.App{
				Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "_list_head"}},
				Args: []core.CoreExpr{xsVar},
			},
			Body: &core.Let{
				Name:  "_hof_t",
				Value: buildListTail(xsVar),
				Body: &core.App{
					Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "::"}},
					Args: []core.CoreExpr{
						bodyWithH,
						&core.App{
							Func: &core.Var{Name: specName},
							Args: []core.CoreExpr{tVar},
						},
					},
				},
			},
		},
	}

	return specBody, specName
}

// SpecializeFilter transforms filter(\x -> pred, xs) into a recursive function:
//
//	filter_spec(xs) =
//	  if seq.len(xs) == 0 then []
//	  else let h = head(xs) in
//	       let t = tail(xs) in
//	       if pred[x -> h] then h :: filter_spec(t)
//	       else filter_spec(t)
func SpecializeFilter(funcName string, lambda *core.Lambda, listParam string) (core.CoreExpr, string) {
	specName := fmt.Sprintf("_filter_spec_%s", funcName)

	if len(lambda.Params) == 0 {
		return nil, ""
	}
	paramName := lambda.Params[0]

	xsVar := &core.Var{Name: listParam}
	hVar := &core.Var{Name: "_hof_h"}
	tVar := &core.Var{Name: "_hof_t"}

	predWithH := SubstituteLambdaVar(lambda.Body, paramName, hVar)

	specBody := &core.If{
		Cond: &core.App{
			Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "eq_Int"}},
			Args: []core.CoreExpr{
				&core.App{
					Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "_list_length"}},
					Args: []core.CoreExpr{xsVar},
				},
				&core.Lit{Kind: core.IntLit, Value: int64(0)},
			},
		},
		Then: &core.List{Elements: nil},
		Else: &core.Let{
			Name: "_hof_h",
			Value: &core.App{
				Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "_list_head"}},
				Args: []core.CoreExpr{xsVar},
			},
			Body: &core.Let{
				Name:  "_hof_t",
				Value: buildListTail(xsVar),
				Body: &core.If{
					Cond: predWithH,
					Then: &core.App{
						Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "::"}},
						Args: []core.CoreExpr{
							hVar,
							&core.App{
								Func: &core.Var{Name: specName},
								Args: []core.CoreExpr{tVar},
							},
						},
					},
					Else: &core.App{
						Func: &core.Var{Name: specName},
						Args: []core.CoreExpr{tVar},
					},
				},
			},
		},
	}

	return specBody, specName
}

// SpecializeFoldl transforms foldl(\acc x -> body, init, xs) into a recursive function:
//
//	fold_spec(acc, xs) =
//	  if seq.len(xs) == 0 then acc
//	  else let h = head(xs) in
//	       let t = tail(xs) in
//	       fold_spec(body[acc -> acc, x -> h], t)
//
// Note: The specialized function takes (acc, xs) as parameters.
// The initial value is passed as the acc argument at the call site.
func SpecializeFoldl(funcName string, lambda *core.Lambda, listParam string) (core.CoreExpr, string) {
	specName := fmt.Sprintf("_foldl_spec_%s", funcName)

	if len(lambda.Params) < 2 {
		return nil, ""
	}
	accParam := lambda.Params[0]
	elemParam := lambda.Params[1]

	xsVar := &core.Var{Name: listParam}
	accVar := &core.Var{Name: "_hof_acc"}
	hVar := &core.Var{Name: "_hof_h"}
	tVar := &core.Var{Name: "_hof_t"}

	// Substitute both acc and x in the lambda body
	bodyWithAcc := SubstituteLambdaVar(lambda.Body, accParam, accVar)
	bodyWithBoth := SubstituteLambdaVar(bodyWithAcc, elemParam, hVar)

	specBody := &core.If{
		Cond: &core.App{
			Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "eq_Int"}},
			Args: []core.CoreExpr{
				&core.App{
					Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "_list_length"}},
					Args: []core.CoreExpr{xsVar},
				},
				&core.Lit{Kind: core.IntLit, Value: int64(0)},
			},
		},
		Then: accVar,
		Else: &core.Let{
			Name: "_hof_h",
			Value: &core.App{
				Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "_list_head"}},
				Args: []core.CoreExpr{xsVar},
			},
			Body: &core.Let{
				Name:  "_hof_t",
				Value: buildListTail(xsVar),
				Body: &core.App{
					Func: &core.Var{Name: specName},
					Args: []core.CoreExpr{
						bodyWithBoth,
						tVar,
					},
				},
			},
		},
	}

	return specBody, specName
}

// buildListTail builds the Core AST for extracting the tail of a list.
// Generates: App(VarGlobal($builtin._list_tail), [listExpr])
// which the SMT encoder maps to: (seq.extract xs 1 (- (seq.len xs) 1))
func buildListTail(listExpr core.CoreExpr) core.CoreExpr {
	return &core.App{
		Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "_list_tail"}},
		Args: []core.CoreExpr{listExpr},
	}
}
