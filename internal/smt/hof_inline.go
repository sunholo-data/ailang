package smt

import (
	"fmt"
	"strings"

	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/types"
)

// HOFKind identifies the type of higher-order function.
type HOFKind int

const (
	HOFMap     HOFKind = iota // map(lambda, list)
	HOFFilter                 // filter(lambda, list)
	HOFFoldl                  // foldl(lambda, init, list)
	HOFAny                    // any(lambda, list)
	HOFFlatMap                // flatMap(lambda, list)
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
	// Unsupported is non-empty when a literal-lambda HOF call was found but
	// could not be specialised (its list or result type is unknown, or the
	// specialised chain does not encode). The call is then left in NewBody, so
	// the encoder must refuse the function rather than encode it partially.
	Unsupported string
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
	"_list_any":   HOFAny,
}

// knownHOFStdlib maps std/list function names to their HOF kind.
var knownHOFStdlib = map[string]HOFKind{
	"map":     HOFMap,
	"filter":  HOFFilter,
	"foldl":   HOFFoldl,
	"any":     HOFAny,
	"flatMap": HOFFlatMap,
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
	case HOFMap, HOFFilter, HOFAny, HOFFlatMap:
		// Expected: App(hof, [lambda, list])
		if len(app.Args) < 2 {
			return InlinableHOFCall{}, false
		}
		lambda, ok := hofFunctionArg(app.Args[0], 1)
		if !ok {
			return InlinableHOFCall{}, false // neither a literal lambda nor a named function
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
		lambda, ok := hofFunctionArg(app.Args[0], 2)
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

		// The specialised chain's signature comes from the call's own types:
		// the list argument's type, and the call's result type (a list for
		// map/filter, the accumulator for foldl). Never assumed: hard-coding
		// (Seq Int) sent every non-int list to Z3 as a sort error (#215).
		listType := hofExprType(call.ListArg)
		resultType := hofExprType(call.Original)
		if listType == nil || resultType == nil {
			result.Unsupported = fmt.Sprintf("the list or result type of a %s call is unknown", call.Kind)
			return result
		}
		sort, err := MapType(resultType)
		if err != nil {
			result.Unsupported = fmt.Sprintf("the result type of a %s call has no SMT sort: %v", call.Kind, err)
			return result
		}
		returnSort = sort

		// One specialisation per call site: two map calls in one function
		// must not share a define-fun name.
		specOwner := funcName
		if specCounter > 1 {
			specOwner = fmt.Sprintf("%s_%d", funcName, specCounter)
		}
		switch call.Kind {
		case HOFMap:
			specBody, specName = SpecializeMap(specOwner, call.Lambda, listParamName)
			params = []FunctionParam{{Name: listParamName, Type: listType}}

		case HOFFilter:
			specBody, specName = SpecializeFilter(specOwner, call.Lambda, listParamName)
			params = []FunctionParam{{Name: listParamName, Type: listType}}

		case HOFAny:
			specBody, specName = SpecializeAny(specOwner, call.Lambda, listParamName)
			params = []FunctionParam{{Name: listParamName, Type: listType}}

		case HOFFlatMap:
			specBody, specName = SpecializeFlatMap(specOwner, call.Lambda, listParamName)
			params = []FunctionParam{{Name: listParamName, Type: listType}}

		case HOFFoldl:
			specBody, specName = SpecializeFoldl(specOwner, call.Lambda, listParamName)
			params = []FunctionParam{
				{Name: "_hof_acc", Type: resultType},
				{Name: listParamName, Type: listType},
			}
		}

		if specBody == nil {
			result.Unsupported = fmt.Sprintf("a %s call's lambda takes no parameter", call.Kind)
			return result
		}

		// Free variables of the lambda travel as extra parameters (hof_captures.go),
		// bound under a $c_ prefix: a capture named like a record accessor
		// (`filter(\p. p.actor == actor, rs)`) would otherwise shadow it.
		caps, err := lambdaCaptures(call.Lambda)
		if err != nil {
			result.Unsupported = fmt.Sprintf("%s call: %v", call.Kind, err)
			return result
		}
		for _, c := range caps {
			params = append(params, FunctionParam{Name: captureSymbol(c.Name), Type: c.Type})
			specBody = SubstituteLambdaVar(specBody, c.Name, &core.Var{Name: captureSymbol(c.Name)})
		}
		specBody = appendCaptureArgs(specBody, specName, caps)

		// Unroll the specialized recursive function
		unrollResult, err := UnrollRecursiveFunction(UnrollConfig{
			FuncName:   specName,
			Params:     params,
			Body:       specBody,
			ReturnSort: returnSort,
			Depth:      depth,
			Frontier:   hofFrontier(call.Kind),
		})
		if err != nil {
			result.Unsupported = fmt.Sprintf("the specialised %s chain does not encode: %v", call.Kind, err)
			return result
		}

		result.Specializations = append(result.Specializations, SpecializationResult{
			Declarations: unrollResult.Declarations,
			TopLevelName: unrollResult.TopLevelName,
			Kind:         call.Kind,
		})

		// Replace the HOF call in the body with a call to the top-level unrolled function
		var replacementCall core.CoreExpr
		captureArgs := make([]core.CoreExpr, len(caps))
		for i, c := range caps {
			captureArgs[i] = c.Node // encoded in the caller's scope, with its renames
		}
		switch call.Kind {
		case HOFMap, HOFFilter, HOFAny, HOFFlatMap:
			replacementCall = &core.App{
				Func: &core.Var{Name: unrollResult.TopLevelName},
				Args: append([]core.CoreExpr{call.ListArg}, captureArgs...),
			}
		case HOFFoldl:
			replacementCall = &core.App{
				Func: &core.Var{Name: unrollResult.TopLevelName},
				Args: append([]core.CoreExpr{call.InitVal, call.ListArg}, captureArgs...),
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
		n := *e
		n.Func, n.Args = newFunc, newArgs
		return &n
	case *core.If:
		newCond := replaceHOFCall(e.Cond, target, replacement)
		newThen := replaceHOFCall(e.Then, target, replacement)
		newElse := replaceHOFCall(e.Else, target, replacement)
		if newCond == e.Cond && newThen == e.Then && newElse == e.Else {
			return e
		}
		n := *e
		n.Cond, n.Then, n.Else = newCond, newThen, newElse
		return &n
	case *core.Let:
		newValue := replaceHOFCall(e.Value, target, replacement)
		newBody := replaceHOFCall(e.Body, target, replacement)
		if newValue == e.Value && newBody == e.Body {
			return e
		}
		n := *e
		n.Value, n.Body = newValue, newBody
		return &n
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
		n := *e
		n.Bindings, n.Body = newBindings, newBody
		return &n
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
		n := *e
		n.Scrutinee, n.Arms = newScrutinee, newArms
		return &n
	case *core.BinOp:
		newLeft := replaceHOFCall(e.Left, target, replacement)
		newRight := replaceHOFCall(e.Right, target, replacement)
		if newLeft == e.Left && newRight == e.Right {
			return e
		}
		n := *e
		n.Left, n.Right = newLeft, newRight
		return &n
	case *core.UnOp:
		newOperand := replaceHOFCall(e.Operand, target, replacement)
		if newOperand == e.Operand {
			return e
		}
		n := *e
		n.Operand = newOperand
		return &n
	case *core.Lambda:
		newBody := replaceHOFCall(e.Body, target, replacement)
		if newBody == e.Body {
			return e
		}
		n := *e
		n.Body = newBody
		return &n
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
		n := *e
		n.Args = newArgs
		return &n
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
		n := *e
		n.Fields = newFields
		return &n
	case *core.RecordAccess:
		newRecord := replaceHOFCall(e.Record, target, replacement)
		if newRecord == e.Record {
			return e
		}
		n := *e
		n.Record = newRecord
		return &n
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
		n := *e
		n.Elements = newElems
		return &n
	default:
		return expr
	}
}

// String names the HOF kind for diagnostics.
func (k HOFKind) String() string {
	switch k {
	case HOFMap:
		return "map"
	case HOFFilter:
		return "filter"
	case HOFFoldl:
		return "foldl"
	case HOFAny:
		return "any"
	case HOFFlatMap:
		return "flatMap"
	}
	return fmt.Sprintf("HOFKind(%d)", int(k))
}

// hofExprType returns the elaborated type of a Core node from the active
// encoding's type info, or nil when it is unknown. Nodes synthesised by the
// verifier carry NodeID 0 and are never looked up.
func hofExprType(expr core.CoreExpr) types.Type {
	c := activeADTContext
	if c == nil || c.ti == nil || expr == nil || expr.ID() == 0 {
		return nil
	}
	t, ok := c.ti.Get(expr.ID())
	if !ok {
		return nil
	}
	return t
}

// hofFunctionArg returns the function argument of a HOF call as a lambda: a
// literal lambda as written, or a named function (`map(f, xs)`, `foldl(step, 0, xs)`)
// eta-expanded to `\$eta0 … . f($eta0 …)` so it is specialised the same way.
// The named function is then an ordinary call head inside the lambda body,
// resolved and admitted like any other callee.
func hofFunctionArg(arg core.CoreExpr, arity int) (*core.Lambda, bool) {
	switch f := arg.(type) {
	case *core.Lambda:
		return f, true
	case *core.Var, *core.VarGlobal:
		// Same-module top-level functions are plain Vars in Core, so both forms
		// expand. A local function value (a parameter of function type) has no
		// body: its application is refused at encoding (ErrUnsupportedConstruct,
		// a skip), as before — only later than the fragment check.
		params := make([]string, arity)
		args := make([]core.CoreExpr, arity)
		for i := range params {
			params[i] = fmt.Sprintf("$eta%d", i)
			args[i] = &core.Var{Name: params[i]}
		}
		return &core.Lambda{Params: params, Body: &core.App{Func: f, Args: args}}, true
	}
	return nil, false
}

// hofCallParts returns the expressions a specialised HOF call still evaluates:
// the (possibly eta-expanded) lambda body, the list, and foldl's initial value.
// Callee collection walks these instead of the raw arguments, so a named
// function argument is seen in call position.
func hofCallParts(call InlinableHOFCall) []core.CoreExpr {
	parts := []core.CoreExpr{call.Lambda.Body, call.ListArg}
	if call.InitVal != nil {
		parts = append(parts, call.InitVal)
	}
	return parts
}

// hofFrontier shapes a specialisation's frontier stub (the unexplored tail) so
// it has the length behaviour every real call has, without quantifiers:
//
//	map:    f_0(xs, …) = (seq.map g xs)                          ; len preserved
//	filter: f_0(xs, …) = (seq.extract (u xs …) 0 (seq.len xs))    ; len never grows
//
// g and u are unconstrained, so the real map (g := the lambda) and the real
// filter (u := the filtered list) are instances: sound. With it, contracts such
// as `length(result) == length(xs)` (map) or `<=` (filter) are proved for lists
// of any length, not only within the unrolling depth. The list is the first
// parameter of a map/filter specialisation; captures follow it.
func hofFrontier(kind HOFKind) func(string, []string, string) []string {
	if kind != HOFMap && kind != HOFFilter {
		return nil
	}
	return func(level0 string, sorts []string, returnSort string) []string {
		binders := make([]string, len(sorts))
		vars := make([]string, len(sorts))
		for i, s := range sorts {
			vars[i] = fmt.Sprintf("$fx%d", i)
			binders[i] = fmt.Sprintf("(%s %s)", vars[i], s)
		}
		head := fmt.Sprintf("(define-fun %s (%s) %s", level0, strings.Join(binders, " "), returnSort)
		if kind == HOFMap {
			in, ok1 := seqElemSort(sorts[0])
			out, ok2 := seqElemSort(returnSort)
			if !ok1 || !ok2 {
				return []string{fmt.Sprintf("(declare-fun %s (%s) %s)", level0, strings.Join(sorts, " "), returnSort)}
			}
			g := level0 + "$g"
			return []string{
				fmt.Sprintf("(declare-const %s (Array %s %s))", g, in, out),
				fmt.Sprintf("%s (seq.map %s %s))", head, g, vars[0]),
			}
		}
		u := level0 + "$u"
		return []string{
			fmt.Sprintf("(declare-fun %s (%s) %s)", u, strings.Join(sorts, " "), returnSort),
			fmt.Sprintf("%s (seq.extract (%s %s) 0 (seq.len %s)))", head, u, strings.Join(vars, " "), vars[0]),
		}
	}
}
