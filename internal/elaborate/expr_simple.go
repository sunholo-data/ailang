package elaborate

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/errors"
	"github.com/sunholo-data/ailang/internal/types"
)

// Simple expression normalization: literals, lambdas, operators

// normalizeLiteral handles literals
func (e *Elaborator) normalizeLiteral(lit *ast.Literal) (core.CoreExpr, error) {
	var kind core.LitKind
	switch lit.Kind {
	case ast.IntLit:
		kind = core.IntLit
	case ast.FloatLit:
		kind = core.FloatLit
	case ast.StringLit:
		kind = core.StringLit
	case ast.BoolLit:
		kind = core.BoolLit
	case ast.UnitLit:
		kind = core.UnitLit
	default:
		return nil, fmt.Errorf("unknown literal kind: %v", lit.Kind)
	}

	return &core.Lit{
		CoreNode: e.makeNode(lit.Position()),
		Kind:     kind,
		Value:    lit.Value,
	}, nil
}

// recordSignatureAnnots hands a function's parameter and return annotations to
// the type checker (inferLambda reads them by Lambda NodeID). It is the ONE
// path for every function form — top-level `func`, module/local `func(...)`
// literals, `func(x: T) => e` lambdas — so an annotation is a constraint
// wherever it is written (#2238: inline literals used to drop theirs).
// A nil ast.Type (an unannotated parameter, or no return type) records nothing
// for that slot; inferLambda then uses a fresh variable.
//
// literal marks a function literal: the checker enforces its parameter
// annotations as equalities (as a let annotation is) instead of seeding the
// parameter's type. Seeding resolves a callback parameter's annotated effect
// row inside the literal's own application solves, where the row's open
// compatibility tail then reads as an undeclared effect of the enclosing
// function — a pre-existing effect-row defect that top-level `func`s share.
// Top-level `func`s keep seeding (M-FIX-FLOAT-OP relies on it).
func (e *Elaborator) recordSignatureAnnots(lam *core.Lambda, params []*ast.Param, ret ast.Type, literal bool) {
	// Keep lam.Params ("_") intact for zero-arg unit injection in eval.CallFunction,
	// while rejecting any unconsumed bare '_' expression (*core.Var{Name: "_"})
	// inside the elaborated body (valid `|>` `_` placeholders were already
	// rewritten to `$pipe_arg_<id>` in normalizePipe).
	if e.typeAnnotationErr == nil && lam != nil {
		if wildPos, found := findBareWildcardVar(lam.Body); found {
			e.typeAnnotationErr = fmt.Errorf("wildcard '_' cannot be used as an expression at %s (use '_' in patterns or as a direct call argument on the right-hand side of '|>')", wildPos)
		}
	}
	paramTypes := make([]types.Type, len(params))
	hasAnnotations := false
	for i, param := range params {
		if param.Type != nil {
			paramTypes[i] = e.astTypeToInternalType(param.Type)
			hasAnnotations = hasAnnotations || paramTypes[i] != nil
		}
	}
	if hasAnnotations {
		e.paramTypeAnnots[lam.ID()] = paramTypes
	}
	if literal {
		if e.literalLambdas == nil {
			e.literalLambdas = make(map[uint64]bool)
		}
		e.literalLambdas[lam.ID()] = true
	} else {
		delete(e.literalLambdas, lam.ID())
	}
	if ret != nil {
		if returnType := e.astTypeToInternalType(ret); returnType != nil {
			e.returnTypeAnnots[lam.ID()] = returnType
		}
	}
}

// normalizeLambda handles lambda expressions
func (e *Elaborator) normalizeLambda(lam *ast.Lambda) (core.CoreExpr, error) {
	// Extract parameter names; for destructuring pattern params (e.g. \(a, b). body),
	// mint a synthetic parameter name and desugar to a single-arm exhaustive core.Match
	// wrapping the lambda body (M-SYNTAX-ERGONOMICS-V2 E4).
	type patBinding struct {
		synName string
		pat     core.CorePattern
		pos     ast.Pos
	}
	params := make([]string, len(lam.Params))
	scopeNames := make([]string, 0, len(lam.Params))
	var patBindings []patBinding
	for i, p := range lam.Params {
		if p.Pattern != nil {
			synName := e.freshVar()
			params[i] = synName
			corePat, err := e.elaboratePattern(p.Pattern)
			if err != nil {
				return nil, fmt.Errorf("failed to elaborate lambda parameter pattern: %w", err)
			}
			patBindings = append(patBindings, patBinding{
				synName: synName,
				pat:     corePat,
				pos:     p.Pos,
			})
			scopeNames = append(scopeNames, astPatternBinders(p.Pattern)...)
		} else {
			params[i] = p.Name
			scopeNames = append(scopeNames, p.Name)
		}
	}

	// Normalize body with the parameters (and pattern binders) in scope (#1467)
	popTry := e.enterTryLambda(lam)
	e.pushScope(scopeNames...)
	body, err := e.normalize(lam.Body)
	e.popScope()
	popTry()
	if err != nil {
		return nil, err
	}

	// Wrap body with single-arm exhaustive matches for any pattern parameters
	// (right-to-left so earlier parameter bindings sit in outer scope).
	for i := len(patBindings) - 1; i >= 0; i-- {
		pb := patBindings[i]
		body = &core.Match{
			CoreNode: e.makeNode(pb.pos),
			Scrutinee: &core.Var{
				CoreNode: e.makeNode(pb.pos),
				Name:     pb.synName,
			},
			Arms: []core.MatchArm{
				{
					Pattern: pb.pat,
					Body:    body,
				},
			},
			Exhaustive: true,
		}
	}

	// Create Core Lambda node
	coreLam := &core.Lambda{
		CoreNode: e.makeNode(lam.Position()),
		Params:   params,
		Body:     body,
	}
	// `func(x: int) => e` carries parameter types (no return type).
	e.recordSignatureAnnots(coreLam, lam.Params, nil, true)

	// Store effect annotations if present
	if len(lam.Effects) > 0 {
		// Extract effect names for backward compatibility
		effectNames := ast.EffectNames(lam.Effects)
		// Validate and normalize effect names
		_, err := types.ElaborateEffectRow(effectNames)
		if err != nil {
			return nil, fmt.Errorf("invalid effect annotation: %w", err)
		}
		e.effectAnnots[coreLam.ID()] = effectNames
		// M-CAPABILITY-BUDGETS: Also store full annotations with budgets
		e.effectAnnotsFull[coreLam.ID()] = lam.Effects
	}

	return coreLam, nil
}

// normalizeFuncLit handles function literal expressions (func(x) -> T { body })
// Desugars to Lambda: func(x: int) -> int { x + 1 } ≡ \x. x + 1
func (e *Elaborator) normalizeFuncLit(funcLit *ast.FuncLit) (core.CoreExpr, error) {
	params := make([]string, len(funcLit.Params))
	for i, p := range funcLit.Params {
		params[i] = p.Name
	}

	// Normalize body with the parameters in scope (#1467)
	popTry := e.enterTryFuncLit(funcLit)
	e.pushScope(params...)
	body, err := e.normalize(funcLit.Body)
	e.popScope()
	popTry()
	if err != nil {
		return nil, err
	}

	// Create Core Lambda node
	coreLam := &core.Lambda{
		CoreNode: e.makeNode(funcLit.Position()),
		Params:   params,
		Body:     body,
	}
	e.recordSignatureAnnots(coreLam, funcLit.Params, funcLit.ReturnType, true)

	// An explicit `! {}` (non-nil, empty — the parser distinguishes it from no
	// annotation) is an annotation: the literal is declared pure, and the
	// checker enforces it like any other declared row (#2238 review). Recorded
	// as a present, empty entry.
	if funcLit.Effects != nil && len(funcLit.Effects) == 0 {
		e.effectAnnotsFull[coreLam.ID()] = []ast.EffectAnnotation{}
	}

	// Store effect annotations if present
	if len(funcLit.Effects) > 0 {
		// Extract effect names for backward compatibility
		effectNames := ast.EffectNames(funcLit.Effects)
		// Validate and normalize effect names
		_, err := types.ElaborateEffectRow(effectNames)
		if err != nil {
			return nil, fmt.Errorf("invalid effect annotation: %w", err)
		}
		e.effectAnnots[coreLam.ID()] = effectNames
		// M-CAPABILITY-BUDGETS: Also store full annotations with budgets
		e.effectAnnotsFull[coreLam.ID()] = funcLit.Effects
	}

	return coreLam, nil
}

// normalizeBinaryOp handles binary operations with ANF transformation
func (e *Elaborator) normalizeBinaryOp(binop *ast.BinaryOp) (core.CoreExpr, error) {
	// Short-circuit operators desugar to core.If so downstream phases
	// (evaluator, bytecode VM, SMT) inherit laziness from If lowering.
	// See design_docs/planned/v0_11_3/m-eval-short-circuit-bool.md.
	if binop.Op == "&&" || binop.Op == "||" {
		return e.normalizeShortCircuit(binop)
	}

	// Pipe operator `x |> f` desugars to LHS-first Let + App (M-PIPE-OPERATOR).
	// D2=a (M-V1-RC1-READINESS-AND-DX-SWARM): when the RHS is a FuncCall with
	// direct `_` placeholder argument(s):
	//   - k == 0: unchanged unary/curried App(elaborate(rhs), [elaborate(lhs)])
	//   - k == 1: Let($pipe_arg_<id>, elaborate(lhs), elaborate(f(..., $pipe_arg_<id>, ...)))
	//   - k > 1:  structured error TC_PIPE_001
	if binop.Op == "|>" {
		return e.normalizePipe(binop)
	}

	// Normalize operands to atomic values
	left, leftBinds, err := e.normalizeToAtomic(binop.Left)
	if err != nil {
		return nil, err
	}

	right, rightBinds, err := e.normalizeToAtomic(binop.Right)
	if err != nil {
		return nil, err
	}

	// Map operator to intrinsic
	var op core.IntrinsicOp
	switch binop.Op {
	case "+":
		op = core.OpAdd
	case "-":
		op = core.OpSub
	case "*":
		op = core.OpMul
	case "/":
		op = core.OpDiv
	case "%":
		op = core.OpMod
	case "==":
		op = core.OpEq
	case "!=":
		op = core.OpNe
	case "<":
		op = core.OpLt
	case "<=":
		op = core.OpLe
	case ">":
		op = core.OpGt
	case ">=":
		op = core.OpGe
	case "++":
		op = core.OpConcat
	case "&":
		op = core.OpBitwiseAnd
	case "^":
		op = core.OpBitwiseXor
	case "<<":
		op = core.OpShiftLeft
	case ">>":
		op = core.OpShiftRight
	default:
		// For compatibility, still create BinOp for unknown operators
		result := &core.BinOp{
			CoreNode: e.makeNode(binop.Position()),
			Op:       binop.Op,
			Left:     left,
			Right:    right,
		}
		return e.wrapWithBindings(result, append(leftBinds, rightBinds...)), nil
	}

	// Create intrinsic operation
	result := &core.Intrinsic{
		CoreNode: e.makeNode(binop.Position()),
		Op:       op,
		Args:     []core.CoreExpr{left, right},
	}

	// Wrap with let bindings from normalization
	return e.wrapWithBindings(result, append(leftBinds, rightBinds...)), nil
}

// isPipePlaceholder reports whether expr is a bare `_` identifier argument
// (`_?` parses as *ast.Hole and nested lambdas/calls are distinct AST nodes).
func isPipePlaceholder(expr ast.Expr) bool {
	ident, ok := expr.(*ast.Identifier)
	return ok && ident.Name == "_" && !ident.ResolveAsBuiltin
}

// normalizePipe elaborates `lhs |> rhs`, supporting a single explicit `_`
// direct call-argument placeholder on a FuncCall RHS (`D2=a`).
func (e *Elaborator) normalizePipe(binop *ast.BinaryOp) (core.CoreExpr, error) {
	if call, ok := binop.Right.(*ast.FuncCall); ok {
		placeholderIdx := -1
		placeholderCount := 0
		var dupPos ast.Pos
		for i, arg := range call.Args {
			if isPipePlaceholder(arg) {
				placeholderCount++
				if placeholderCount == 1 {
					placeholderIdx = i
				} else if placeholderCount == 2 {
					dupPos = arg.Position()
				}
			}
		}
		if placeholderCount > 1 {
			if dupPos.Line == 0 {
				dupPos = call.Position()
			}
			return nil, errors.WrapReport(&errors.Report{
				Schema:  "ailang.error/v1",
				Code:    "TC_PIPE_001",
				Phase:   "elaborate",
				Message: "pipe stage has multiple '_' placeholders; use a lambda (\\x. ...) instead",
				Span:    &ast.Span{Start: dupPos, End: dupPos},
				Fix: &errors.Fix{
					Suggestion: "Bind the piped value once with a lambda, e.g. x |> (\\v. f(v, v))",
					Confidence: 0.95,
				},
			})
		}
		if placeholderCount == 1 {
			e.freshVarNum++
			pipeTmp := fmt.Sprintf("$pipe_arg_%d", e.freshVarNum)
			elabLHS, err := e.normalize(binop.Left)
			if err != nil {
				return nil, err
			}
			rewrittenArgs := make([]ast.Expr, len(call.Args))
			copy(rewrittenArgs, call.Args)
			rewrittenArgs[placeholderIdx] = &ast.Identifier{
				Name: pipeTmp,
				Pos:  call.Args[placeholderIdx].Position(),
			}
			rewrittenCall := &ast.FuncCall{
				Func: call.Func,
				Args: rewrittenArgs,
				Pos:  call.Pos,
			}
			e.pushScope(pipeTmp)
			elabCall, err := e.normalizeFuncCall(rewrittenCall)
			e.popScope()
			if err != nil {
				return nil, err
			}
			return &core.Let{
				CoreNode: e.makeNode(binop.Position()),
				Name:     pipeTmp,
				Value:    elabLHS,
				Body:     elabCall,
			}, nil
		}
	}

	// k == 0 (unary function, lambda, or curried call such as `5 |> makeAdder(1)`):
	// normalizeToAtomic(Left) runs before normalizeToAtomic(Right), pinning
	// left-to-right evaluation order for effectful stages.
	left, leftBinds, err := e.normalizeToAtomic(binop.Left)
	if err != nil {
		return nil, err
	}
	right, rightBinds, err := e.normalizeToAtomic(binop.Right)
	if err != nil {
		return nil, err
	}
	app := &core.App{
		CoreNode: e.makeNode(binop.Position()),
		Func:     right,
		Args:     []core.CoreExpr{left},
	}
	return e.wrapWithBindings(app, append(leftBinds, rightBinds...)), nil
}

// findBareWildcardVar reports whether expr contains an unconsumed *core.Var{Name: "_"}.
func findBareWildcardVar(expr core.CoreExpr) (ast.Pos, bool) {
	if expr == nil {
		return ast.Pos{}, false
	}
	switch n := expr.(type) {
	case *core.Var:
		if n.Name == "_" {
			return n.OriginalSpan(), true
		}
	case *core.Lambda:
		return findBareWildcardVar(n.Body)
	case *core.Let:
		if p, ok := findBareWildcardVar(n.Value); ok {
			return p, true
		}
		return findBareWildcardVar(n.Body)
	case *core.LetRec:
		for _, b := range n.Bindings {
			if p, ok := findBareWildcardVar(b.Value); ok {
				return p, true
			}
		}
		return findBareWildcardVar(n.Body)
	case *core.App:
		if p, ok := findBareWildcardVar(n.Func); ok {
			return p, true
		}
		for _, a := range n.Args {
			if p, ok := findBareWildcardVar(a); ok {
				return p, true
			}
		}
	case *core.If:
		if p, ok := findBareWildcardVar(n.Cond); ok {
			return p, true
		}
		if p, ok := findBareWildcardVar(n.Then); ok {
			return p, true
		}
		return findBareWildcardVar(n.Else)
	case *core.Match:
		if p, ok := findBareWildcardVar(n.Scrutinee); ok {
			return p, true
		}
		for _, arm := range n.Arms {
			if p, ok := findBareWildcardVar(arm.Guard); ok {
				return p, true
			}
			if p, ok := findBareWildcardVar(arm.Body); ok {
				return p, true
			}
		}
	case *core.Intrinsic:
		for _, a := range n.Args {
			if p, ok := findBareWildcardVar(a); ok {
				return p, true
			}
		}
	case *core.BinOp:
		if p, ok := findBareWildcardVar(n.Left); ok {
			return p, true
		}
		return findBareWildcardVar(n.Right)
	case *core.UnOp:
		return findBareWildcardVar(n.Operand)
	case *core.Record:
		for _, v := range n.Fields {
			if p, ok := findBareWildcardVar(v); ok {
				return p, true
			}
		}
	case *core.RecordAccess:
		return findBareWildcardVar(n.Record)
	case *core.RecordUpdate:
		if p, ok := findBareWildcardVar(n.Base); ok {
			return p, true
		}
		for _, v := range n.Updates {
			if p, ok := findBareWildcardVar(v); ok {
				return p, true
			}
		}
	case *core.List:
		for _, el := range n.Elements {
			if p, ok := findBareWildcardVar(el); ok {
				return p, true
			}
		}
	case *core.Array:
		for _, el := range n.Elements {
			if p, ok := findBareWildcardVar(el); ok {
				return p, true
			}
		}
	case *core.Tuple:
		for _, el := range n.Elements {
			if p, ok := findBareWildcardVar(el); ok {
				return p, true
			}
		}
	case *core.Forall:
		if p, ok := findBareWildcardVar(n.Lo); ok {
			return p, true
		}
		if p, ok := findBareWildcardVar(n.Hi); ok {
			return p, true
		}
		return findBareWildcardVar(n.Body)
	}
	return ast.Pos{}, false
}

// normalizeShortCircuit desugars `a && b` and `a || b` into core.If so that
// the right operand is only evaluated when needed:
//
//	a && b  =>  if a then b else false
//	a || b  =>  if a then true else b
//
// The LHS is normalized to atomic (ANF) so any complex subexpressions run
// exactly once before the branch. The RHS is normalized *without* forcing it
// to atomic — any let-bindings it would emit stay inside the lazy branch.
func (e *Elaborator) normalizeShortCircuit(binop *ast.BinaryOp) (core.CoreExpr, error) {
	left, leftBinds, err := e.normalizeToAtomic(binop.Left)
	if err != nil {
		return nil, err
	}

	right, err := e.normalize(binop.Right)
	if err != nil {
		return nil, err
	}

	node := e.makeNode(binop.Position())
	boolLit := func(v bool) core.CoreExpr {
		return &core.Lit{
			CoreNode: e.makeNode(binop.Position()),
			Kind:     core.BoolLit,
			Value:    v,
		}
	}

	var thenBranch, elseBranch core.CoreExpr
	switch binop.Op {
	case "&&":
		thenBranch = right
		elseBranch = boolLit(false)
	case "||":
		thenBranch = boolLit(true)
		elseBranch = right
	default:
		return nil, fmt.Errorf("normalizeShortCircuit called with non-short-circuit op %q", binop.Op)
	}

	result := &core.If{
		CoreNode: node,
		Cond:     left,
		Then:     thenBranch,
		Else:     elseBranch,
	}
	return e.wrapWithBindings(result, leftBinds), nil
}

// normalizeUnaryOp handles unary operations
func (e *Elaborator) normalizeUnaryOp(unop *ast.UnaryOp) (core.CoreExpr, error) {
	operand, binds, err := e.normalizeToAtomic(unop.Expr)
	if err != nil {
		return nil, err
	}

	// Map operator to intrinsic
	var op core.IntrinsicOp
	switch unop.Op {
	case "-":
		op = core.OpNeg
	case "not", "!":
		// `!e` is the same boolean negation as `not e` (the teaching prompt promises
		// both). Without this arm `!` fell through to a generic UnOp that no later
		// stage knows, and died in the type checker as "unknown unary operator: !"
		// (claude-sonnet-5, mlfq_scheduler_hidden, 2026-09-22). Mapped here rather than
		// in the parser so the surface AST — and `ailang fmt` — keep what the user wrote.
		op = core.OpNot
	case "~":
		op = core.OpBitwiseNot
	default:
		// For compatibility, still create UnOp for unknown operators
		result := &core.UnOp{
			CoreNode: e.makeNode(unop.Position()),
			Op:       unop.Op,
			Operand:  operand,
		}
		return e.wrapWithBindings(result, binds), nil
	}

	// Create intrinsic operation
	result := &core.Intrinsic{
		CoreNode: e.makeNode(unop.Position()),
		Op:       op,
		Args:     []core.CoreExpr{operand},
	}

	return e.wrapWithBindings(result, binds), nil
}
