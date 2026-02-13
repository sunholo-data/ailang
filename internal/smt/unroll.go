package smt

import (
	"fmt"
	"strings"

	"github.com/sunholo/ailang/internal/core"
)

// UnrollConfig holds parameters for bounded recursive function unrolling.
type UnrollConfig struct {
	FuncName   string          // Name of the recursive function
	Params     []FunctionParam // Function parameters
	Body       core.CoreExpr   // Original function body
	ReturnSort string          // SMT-LIB return sort
	Depth      int             // Unrolling depth (1-10)
}

// UnrollResult holds the generated SMT-LIB declarations for a bounded unrolling.
type UnrollResult struct {
	// Declarations contains the declare-fun (level 0) and define-fun (levels 1..N).
	Declarations []string
	// TopLevelName is the name of the deepest unrolling (funcName_N).
	TopLevelName string
}

// UnrollRecursiveFunction generates a chain of define-fun declarations for
// bounded recursive function verification (Dafny-style unrolling).
//
// For depth N, it produces N+1 declarations:
//   - Level 0: (declare-fun funcName_0 ...) — uninterpreted function
//   - Levels 1..N: (define-fun funcName_k ...) where self-calls are replaced
//     with funcName_(k-1)
//
// The top-level name funcName_N is used in verification conditions.
func UnrollRecursiveFunction(cfg UnrollConfig) (*UnrollResult, error) {
	if cfg.Depth < 1 || cfg.Depth > 10 {
		return nil, fmt.Errorf("unroll depth must be 1-10, got %d", cfg.Depth)
	}

	result := &UnrollResult{}

	// Level 0: uninterpreted function (declare-fun)
	level0Name := fmt.Sprintf("%s_0", cfg.FuncName)
	var paramSorts []string
	for _, p := range cfg.Params {
		sort, err := MapType(p.Type)
		if err != nil {
			return nil, fmt.Errorf("cannot map parameter %q type: %w", p.Name, err)
		}
		paramSorts = append(paramSorts, sort)
	}
	declareFun := fmt.Sprintf("(declare-fun %s (%s) %s)",
		level0Name, strings.Join(paramSorts, " "), cfg.ReturnSort)
	result.Declarations = append(result.Declarations, declareFun)

	// Levels 1..N: define-fun with self-calls replaced
	for k := 1; k <= cfg.Depth; k++ {
		levelName := fmt.Sprintf("%s_%d", cfg.FuncName, k)
		prevName := fmt.Sprintf("%s_%d", cfg.FuncName, k-1)

		// Rewrite body: replace self-calls with previous level
		rewrittenBody := ReplaceSelfCalls(cfg.Body, cfg.FuncName, prevName)

		// Encode the rewritten body
		bodyExpr, err := EncodeExpr(rewrittenBody)
		if err != nil {
			return nil, fmt.Errorf("encoding unrolled body at depth %d: %w", k, err)
		}

		// Build define-fun
		defineFun := buildDefineFun(levelName, cfg.Params, cfg.ReturnSort, bodyExpr)
		result.Declarations = append(result.Declarations, defineFun)
	}

	result.TopLevelName = fmt.Sprintf("%s_%d", cfg.FuncName, cfg.Depth)
	return result, nil
}

// ReplaceSelfCalls walks the Core AST and replaces references to funcName
// with replacement. Returns a new AST (does not mutate the original).
// Respects Lambda/Let/LetRec shadowing.
func ReplaceSelfCalls(expr core.CoreExpr, funcName, replacement string) core.CoreExpr {
	if expr == nil {
		return nil
	}
	switch e := expr.(type) {
	case *core.Var:
		if e.Name == funcName {
			return &core.Var{Name: replacement}
		}
		return e
	case *core.VarGlobal:
		if e.Ref.Name == funcName {
			return &core.VarGlobal{Ref: core.GlobalRef{
				Module: e.Ref.Module,
				Name:   replacement,
			}}
		}
		return e
	case *core.Lit:
		return e
	case *core.Lambda:
		// If the lambda re-binds the name, it shadows — stop replacing
		for _, p := range e.Params {
			if p == funcName {
				return e
			}
		}
		newBody := ReplaceSelfCalls(e.Body, funcName, replacement)
		if newBody == e.Body {
			return e
		}
		return &core.Lambda{Params: e.Params, Body: newBody}
	case *core.App:
		newFunc := ReplaceSelfCalls(e.Func, funcName, replacement)
		newArgs := replaceSelfCallsSlice(e.Args, funcName, replacement)
		if newFunc == e.Func && sliceUnchanged(newArgs, e.Args) {
			return e
		}
		return &core.App{Func: newFunc, Args: newArgs}
	case *core.If:
		newCond := ReplaceSelfCalls(e.Cond, funcName, replacement)
		newThen := ReplaceSelfCalls(e.Then, funcName, replacement)
		newElse := ReplaceSelfCalls(e.Else, funcName, replacement)
		if newCond == e.Cond && newThen == e.Then && newElse == e.Else {
			return e
		}
		return &core.If{Cond: newCond, Then: newThen, Else: newElse}
	case *core.Let:
		newValue := ReplaceSelfCalls(e.Value, funcName, replacement)
		// If let re-binds the name, stop replacing in body
		if e.Name == funcName {
			if newValue == e.Value {
				return e
			}
			return &core.Let{Name: e.Name, Value: newValue, Body: e.Body}
		}
		newBody := ReplaceSelfCalls(e.Body, funcName, replacement)
		if newValue == e.Value && newBody == e.Body {
			return e
		}
		return &core.Let{Name: e.Name, Value: newValue, Body: newBody}
	case *core.LetRec:
		// Check if any binding shadows the function name
		shadowed := false
		for _, b := range e.Bindings {
			if b.Name == funcName {
				shadowed = true
				break
			}
		}
		if shadowed {
			return e
		}
		newBindings := make([]core.RecBinding, len(e.Bindings))
		changed := false
		for i, b := range e.Bindings {
			newVal := ReplaceSelfCalls(b.Value, funcName, replacement)
			newBindings[i] = core.RecBinding{Name: b.Name, Value: newVal}
			if newVal != b.Value {
				changed = true
			}
		}
		newBody := ReplaceSelfCalls(e.Body, funcName, replacement)
		if !changed && newBody == e.Body {
			return e
		}
		return &core.LetRec{Bindings: newBindings, Body: newBody}
	case *core.Match:
		newScrutinee := ReplaceSelfCalls(e.Scrutinee, funcName, replacement)
		newArms := make([]core.MatchArm, len(e.Arms))
		changed := newScrutinee != e.Scrutinee
		for i, arm := range e.Arms {
			newArmBody := ReplaceSelfCalls(arm.Body, funcName, replacement)
			var newGuard core.CoreExpr
			if arm.Guard != nil {
				newGuard = ReplaceSelfCalls(arm.Guard, funcName, replacement)
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
		newLeft := ReplaceSelfCalls(e.Left, funcName, replacement)
		newRight := ReplaceSelfCalls(e.Right, funcName, replacement)
		if newLeft == e.Left && newRight == e.Right {
			return e
		}
		return &core.BinOp{Op: e.Op, Left: newLeft, Right: newRight}
	case *core.UnOp:
		newOperand := ReplaceSelfCalls(e.Operand, funcName, replacement)
		if newOperand == e.Operand {
			return e
		}
		return &core.UnOp{Op: e.Op, Operand: newOperand}
	case *core.Intrinsic:
		newArgs := replaceSelfCallsSlice(e.Args, funcName, replacement)
		if sliceUnchanged(newArgs, e.Args) {
			return e
		}
		return &core.Intrinsic{Op: e.Op, Args: newArgs}
	case *core.Record:
		newFields := make(map[string]core.CoreExpr, len(e.Fields))
		changed := false
		for k, v := range e.Fields {
			newV := ReplaceSelfCalls(v, funcName, replacement)
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
		newRecord := ReplaceSelfCalls(e.Record, funcName, replacement)
		if newRecord == e.Record {
			return e
		}
		return &core.RecordAccess{Record: newRecord, Field: e.Field}
	case *core.RecordUpdate:
		newBase := ReplaceSelfCalls(e.Base, funcName, replacement)
		newUpdates := make(map[string]core.CoreExpr, len(e.Updates))
		changed := newBase != e.Base
		for k, v := range e.Updates {
			newV := ReplaceSelfCalls(v, funcName, replacement)
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
		newElems := replaceSelfCallsSlice(e.Elements, funcName, replacement)
		if sliceUnchanged(newElems, e.Elements) {
			return e
		}
		return &core.List{Elements: newElems}
	case *core.Tuple:
		newElems := replaceSelfCallsSlice(e.Elements, funcName, replacement)
		if sliceUnchanged(newElems, e.Elements) {
			return e
		}
		return &core.Tuple{Elements: newElems}
	case *core.DictApp:
		newDict := ReplaceSelfCalls(e.Dict, funcName, replacement)
		newArgs := replaceSelfCallsSlice(e.Args, funcName, replacement)
		if newDict == e.Dict && sliceUnchanged(newArgs, e.Args) {
			return e
		}
		return &core.DictApp{Dict: newDict, Args: newArgs}
	case *core.DictAbs:
		newBody := ReplaceSelfCalls(e.Body, funcName, replacement)
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

func replaceSelfCallsSlice(exprs []core.CoreExpr, funcName, replacement string) []core.CoreExpr {
	result := make([]core.CoreExpr, len(exprs))
	for i, e := range exprs {
		result[i] = ReplaceSelfCalls(e, funcName, replacement)
	}
	return result
}

func sliceUnchanged(newSlice, oldSlice []core.CoreExpr) bool {
	if len(newSlice) != len(oldSlice) {
		return false
	}
	for i := range newSlice {
		if newSlice[i] != oldSlice[i] {
			return false
		}
	}
	return true
}
