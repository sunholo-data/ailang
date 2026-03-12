package smt

import (
	"fmt"

	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/types"
)

// HOFKind identifies the type of higher-order function.
type HOFKind int

const (
	HOFMap    HOFKind = iota // map(lambda, list)
	HOFFilter                // filter(lambda, list)
	HOFFoldl                 // foldl(lambda, init, list)
)

// InlinableHOFCall describes a detected higher-order function call
// that can be specialized (lambda argument is a literal, not a variable).
type InlinableHOFCall struct {
	Kind    HOFKind
	Lambda  *core.Lambda  // The literal lambda argument
	ListArg core.CoreExpr // The list argument
	InitVal core.CoreExpr // Only for foldl: the initial accumulator value
	// Original is the original App node for replacement
	Original *core.App
}

// InlineResult holds the result of HOF inlining for a function body.
type InlineResult struct {
	// NewBody is the modified function body with HOF calls replaced
	// by calls to specialized recursive functions.
	NewBody core.CoreExpr
	// Specializations are the unrolled recursive function declarations.
	Specializations []SpecializationResult
}

// SpecializationResult holds the unrolled declarations for one specialization.
type SpecializationResult struct {
	// Declarations are the SMT-LIB declare-fun/define-fun chain.
	Declarations []string
	// TopLevelName is the top-level unrolled function name.
	TopLevelName string
	// Kind is which HOF was specialized.
	Kind HOFKind
}

// knownHOFBuiltins maps builtin names to their HOF kind.
var knownHOFBuiltins = map[string]HOFKind{
	"map_List":    HOFMap,
	"filter_List": HOFFilter,
	"foldl_List":  HOFFoldl,
}

// knownHOFStdlib maps std/list function names to their HOF kind.
var knownHOFStdlib = map[string]HOFKind{
	"map":    HOFMap,
	"filter": HOFFilter,
	"foldl":  HOFFoldl,
}

// IsInlinableHOF walks the Core AST looking for HOF calls (map, filter, foldl)
// where the function argument is a literal lambda (not a variable).
// Returns info about each inlinable call found.
func IsInlinableHOF(body core.CoreExpr) []InlinableHOFCall {
	var calls []InlinableHOFCall
	walkForInlinableHOF(body, &calls)
	return calls
}

func walkForInlinableHOF(expr core.CoreExpr, calls *[]InlinableHOFCall) {
	if expr == nil {
		return
	}
	switch e := expr.(type) {
	case *core.App:
		if call, ok := matchHOFCall(e); ok {
			*calls = append(*calls, call)
			return
		}
		// Walk function and args for nested HOF calls
		walkForInlinableHOF(e.Func, calls)
		for _, arg := range e.Args {
			walkForInlinableHOF(arg, calls)
		}
	case *core.If:
		walkForInlinableHOF(e.Cond, calls)
		walkForInlinableHOF(e.Then, calls)
		walkForInlinableHOF(e.Else, calls)
	case *core.Let:
		walkForInlinableHOF(e.Value, calls)
		walkForInlinableHOF(e.Body, calls)
	case *core.LetRec:
		for _, b := range e.Bindings {
			walkForInlinableHOF(b.Value, calls)
		}
		walkForInlinableHOF(e.Body, calls)
	case *core.Match:
		walkForInlinableHOF(e.Scrutinee, calls)
		for _, arm := range e.Arms {
			walkForInlinableHOF(arm.Body, calls)
			if arm.Guard != nil {
				walkForInlinableHOF(arm.Guard, calls)
			}
		}
	case *core.BinOp:
		walkForInlinableHOF(e.Left, calls)
		walkForInlinableHOF(e.Right, calls)
	case *core.UnOp:
		walkForInlinableHOF(e.Operand, calls)
	case *core.Lambda:
		walkForInlinableHOF(e.Body, calls)
	case *core.Intrinsic:
		for _, arg := range e.Args {
			walkForInlinableHOF(arg, calls)
		}
	case *core.Record:
		for _, v := range e.Fields {
			walkForInlinableHOF(v, calls)
		}
	case *core.RecordAccess:
		walkForInlinableHOF(e.Record, calls)
	case *core.List:
		for _, elem := range e.Elements {
			walkForInlinableHOF(elem, calls)
		}
	}
}

// matchHOFCall checks if an App node is a known HOF call with a literal lambda.
func matchHOFCall(app *core.App) (InlinableHOFCall, bool) {
	vg, ok := app.Func.(*core.VarGlobal)
	if !ok {
		return InlinableHOFCall{}, false
	}

	var hofKind HOFKind
	found := false

	// Check $builtin module
	if vg.Ref.Module == "$builtin" {
		if kind, ok := knownHOFBuiltins[vg.Ref.Name]; ok {
			hofKind = kind
			found = true
		}
	}

	// Check std/list module
	if vg.Ref.Module == "std/list" {
		if kind, ok := knownHOFStdlib[vg.Ref.Name]; ok {
			hofKind = kind
			found = true
		}
	}

	if !found {
		return InlinableHOFCall{}, false
	}

	// Now check that the function argument (first arg) is a literal Lambda
	switch hofKind {
	case HOFMap, HOFFilter:
		// Expected: App(hof, [lambda, list])
		if len(app.Args) < 2 {
			return InlinableHOFCall{}, false
		}
		lambda, ok := app.Args[0].(*core.Lambda)
		if !ok {
			return InlinableHOFCall{}, false // Function arg is not a literal lambda
		}
		return InlinableHOFCall{
			Kind:     hofKind,
			Lambda:   lambda,
			ListArg:  app.Args[1],
			Original: app,
		}, true

	case HOFFoldl:
		// Expected: App(foldl, [lambda, init, list])
		if len(app.Args) < 3 {
			return InlinableHOFCall{}, false
		}
		lambda, ok := app.Args[0].(*core.Lambda)
		if !ok {
			return InlinableHOFCall{}, false
		}
		return InlinableHOFCall{
			Kind:     hofKind,
			Lambda:   lambda,
			ListArg:  app.Args[2],
			InitVal:  app.Args[1],
			Original: app,
		}, true
	}

	return InlinableHOFCall{}, false
}

// AllHigherOrderIsInlinable returns true if every lambda-in-arg-position
// in the body is part of a known HOF call (map_List, filter_List, foldl_List
// with a literal lambda). Returns true when there is no higher-order at all.
func AllHigherOrderIsInlinable(body core.CoreExpr) bool {
	if body == nil {
		return true
	}
	return walkAllHOInlinable(body, false)
}

// walkAllHOInlinable returns false if it finds any lambda-in-arg-position
// that is NOT part of a known inlinable HOF call.
func walkAllHOInlinable(expr core.CoreExpr, inArgPosition bool) bool {
	if expr == nil {
		return true
	}
	switch e := expr.(type) {
	case *core.Lambda:
		if inArgPosition {
			// This lambda is in arg position but we don't know
			// if its parent is a known HOF — return false
			// (The parent App handler sets inArgPosition=true only for
			// non-HOF calls; HOF calls are handled specially.)
			return false
		}
		return walkAllHOInlinable(e.Body, false)
	case *core.App:
		// Check if this App is a known HOF call with literal lambda
		if _, ok := matchHOFCall(e); ok {
			// This entire App is a known inlinable HOF.
			// We still need to check the lambda body and list arg
			// for any deeper HOF that might not be inlinable.
			// Walk args but NOT in arg position (they are inlinable).
			for _, arg := range e.Args {
				if _, isLambda := arg.(*core.Lambda); isLambda {
					// Walk inside the lambda body (not in arg position)
					lam := arg.(*core.Lambda)
					if !walkAllHOInlinable(lam.Body, false) {
						return false
					}
				} else {
					if !walkAllHOInlinable(arg, false) {
						return false
					}
				}
			}
			return true
		}
		// Not a known HOF call — walk func normally, args in arg position
		if !walkAllHOInlinable(e.Func, false) {
			return false
		}
		for _, arg := range e.Args {
			if !walkAllHOInlinable(arg, true) {
				return false
			}
		}
		return true
	case *core.If:
		return walkAllHOInlinable(e.Cond, false) &&
			walkAllHOInlinable(e.Then, false) &&
			walkAllHOInlinable(e.Else, false)
	case *core.Let:
		return walkAllHOInlinable(e.Value, false) &&
			walkAllHOInlinable(e.Body, false)
	case *core.LetRec:
		for _, b := range e.Bindings {
			if !walkAllHOInlinable(b.Value, false) {
				return false
			}
		}
		return walkAllHOInlinable(e.Body, false)
	case *core.Match:
		if !walkAllHOInlinable(e.Scrutinee, false) {
			return false
		}
		for _, arm := range e.Arms {
			if !walkAllHOInlinable(arm.Body, false) {
				return false
			}
		}
		return true
	case *core.BinOp:
		return walkAllHOInlinable(e.Left, false) &&
			walkAllHOInlinable(e.Right, false)
	case *core.UnOp:
		return walkAllHOInlinable(e.Operand, false)
	case *core.Intrinsic:
		for _, arg := range e.Args {
			if !walkAllHOInlinable(arg, false) {
				return false
			}
		}
		return true
	default:
		return true
	}
}

// InlineHOFCalls detects inlinable HOF calls in the body, generates specialized
// recursive functions, replaces the HOF calls with calls to the specialized
// functions, and returns the unrolled declarations. Returns nil if no HOF calls
// are found.
func InlineHOFCalls(funcName string, body core.CoreExpr, depth int) *InlineResult {
	calls := IsInlinableHOF(body)
	if len(calls) == 0 {
		return nil
	}

	if depth < 1 {
		depth = 3 // default unroll depth for HOF specializations
	}

	result := &InlineResult{
		NewBody: body,
	}

	specCounter := 0
	for _, call := range calls {
		specCounter++
		var specBody core.CoreExpr
		var specName string
		var params []FunctionParam
		var returnSort string

		// Get the list param name (or generate one)
		listParamName := fmt.Sprintf("_hof_xs_%d", specCounter)

		switch call.Kind {
		case HOFMap:
			specBody, specName = SpecializeMap(funcName, call.Lambda, listParamName)
			params = []FunctionParam{
				{Name: listParamName, Type: &types.TList{Element: &types.TCon{Name: "int"}}},
			}
			returnSort = "(Seq Int)"

		case HOFFilter:
			specBody, specName = SpecializeFilter(funcName, call.Lambda, listParamName)
			params = []FunctionParam{
				{Name: listParamName, Type: &types.TList{Element: &types.TCon{Name: "int"}}},
			}
			returnSort = "(Seq Int)"

		case HOFFoldl:
			specBody, specName = SpecializeFoldl(funcName, call.Lambda, listParamName)
			params = []FunctionParam{
				{Name: "_hof_acc", Type: &types.TCon{Name: "int"}},
				{Name: listParamName, Type: &types.TList{Element: &types.TCon{Name: "int"}}},
			}
			returnSort = "Int"
		}

		if specBody == nil {
			continue
		}

		// Unroll the specialized recursive function
		unrollResult, err := UnrollRecursiveFunction(UnrollConfig{
			FuncName:   specName,
			Params:     params,
			Body:       specBody,
			ReturnSort: returnSort,
			Depth:      depth,
		})
		if err != nil {
			// If unrolling fails, skip this specialization
			continue
		}

		result.Specializations = append(result.Specializations, SpecializationResult{
			Declarations: unrollResult.Declarations,
			TopLevelName: unrollResult.TopLevelName,
			Kind:         call.Kind,
		})

		// Replace the HOF call in the body with a call to the top-level unrolled function
		var replacementCall core.CoreExpr
		switch call.Kind {
		case HOFMap, HOFFilter:
			replacementCall = &core.App{
				Func: &core.Var{Name: unrollResult.TopLevelName},
				Args: []core.CoreExpr{call.ListArg},
			}
		case HOFFoldl:
			replacementCall = &core.App{
				Func: &core.Var{Name: unrollResult.TopLevelName},
				Args: []core.CoreExpr{call.InitVal, call.ListArg},
			}
		}

		result.NewBody = replaceHOFCall(result.NewBody, call.Original, replacementCall)
	}

	return result
}

// replaceHOFCall replaces a specific App node in the AST with a replacement.
// Uses pointer identity to find the right node.
func replaceHOFCall(expr core.CoreExpr, target *core.App, replacement core.CoreExpr) core.CoreExpr {
	if expr == nil {
		return nil
	}
	switch e := expr.(type) {
	case *core.App:
		if e == target {
			return replacement
		}
		newFunc := replaceHOFCall(e.Func, target, replacement)
		newArgs := make([]core.CoreExpr, len(e.Args))
		changed := newFunc != e.Func
		for i, arg := range e.Args {
			newArgs[i] = replaceHOFCall(arg, target, replacement)
			if newArgs[i] != arg {
				changed = true
			}
		}
		if !changed {
			return e
		}
		return &core.App{Func: newFunc, Args: newArgs}
	case *core.If:
		newCond := replaceHOFCall(e.Cond, target, replacement)
		newThen := replaceHOFCall(e.Then, target, replacement)
		newElse := replaceHOFCall(e.Else, target, replacement)
		if newCond == e.Cond && newThen == e.Then && newElse == e.Else {
			return e
		}
		return &core.If{Cond: newCond, Then: newThen, Else: newElse}
	case *core.Let:
		newValue := replaceHOFCall(e.Value, target, replacement)
		newBody := replaceHOFCall(e.Body, target, replacement)
		if newValue == e.Value && newBody == e.Body {
			return e
		}
		return &core.Let{Name: e.Name, Value: newValue, Body: newBody}
	case *core.LetRec:
		newBindings := make([]core.RecBinding, len(e.Bindings))
		changed := false
		for i, b := range e.Bindings {
			newVal := replaceHOFCall(b.Value, target, replacement)
			newBindings[i] = core.RecBinding{Name: b.Name, Value: newVal}
			if newVal != b.Value {
				changed = true
			}
		}
		newBody := replaceHOFCall(e.Body, target, replacement)
		if !changed && newBody == e.Body {
			return e
		}
		return &core.LetRec{Bindings: newBindings, Body: newBody}
	case *core.Match:
		newScrutinee := replaceHOFCall(e.Scrutinee, target, replacement)
		newArms := make([]core.MatchArm, len(e.Arms))
		changed := newScrutinee != e.Scrutinee
		for i, arm := range e.Arms {
			newArmBody := replaceHOFCall(arm.Body, target, replacement)
			var newGuard core.CoreExpr
			if arm.Guard != nil {
				newGuard = replaceHOFCall(arm.Guard, target, replacement)
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
		newLeft := replaceHOFCall(e.Left, target, replacement)
		newRight := replaceHOFCall(e.Right, target, replacement)
		if newLeft == e.Left && newRight == e.Right {
			return e
		}
		return &core.BinOp{Op: e.Op, Left: newLeft, Right: newRight}
	case *core.UnOp:
		newOperand := replaceHOFCall(e.Operand, target, replacement)
		if newOperand == e.Operand {
			return e
		}
		return &core.UnOp{Op: e.Op, Operand: newOperand}
	case *core.Lambda:
		newBody := replaceHOFCall(e.Body, target, replacement)
		if newBody == e.Body {
			return e
		}
		return &core.Lambda{Params: e.Params, Body: newBody}
	case *core.Intrinsic:
		newArgs := make([]core.CoreExpr, len(e.Args))
		changed := false
		for i, arg := range e.Args {
			newArgs[i] = replaceHOFCall(arg, target, replacement)
			if newArgs[i] != arg {
				changed = true
			}
		}
		if !changed {
			return e
		}
		return &core.Intrinsic{Op: e.Op, Args: newArgs}
	case *core.Record:
		newFields := make(map[string]core.CoreExpr, len(e.Fields))
		changed := false
		for k, v := range e.Fields {
			newV := replaceHOFCall(v, target, replacement)
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
		newRecord := replaceHOFCall(e.Record, target, replacement)
		if newRecord == e.Record {
			return e
		}
		return &core.RecordAccess{Record: newRecord, Field: e.Field}
	case *core.List:
		newElems := make([]core.CoreExpr, len(e.Elements))
		changed := false
		for i, elem := range e.Elements {
			newElems[i] = replaceHOFCall(elem, target, replacement)
			if newElems[i] != elem {
				changed = true
			}
		}
		if !changed {
			return e
		}
		return &core.List{Elements: newElems}
	default:
		return expr
	}
}
