package elaborate

import (
	"fmt"

	"github.com/sunholo/ailang/internal/ast"
	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/types"
)

// elaborateExpr transforms surface expression to Core ANF
func (e *Elaborator) elaborateExpr(expr ast.Expr) (core.CoreExpr, error) {
	// First pass: desugar surface constructs
	desugared := e.desugar(expr)

	// Second pass: normalize to ANF
	return e.normalize(desugared)
}

// desugar handles surface syntax sugar
func (e *Elaborator) desugar(expr ast.Expr) ast.Expr {
	// For now, pass through - will add ? operator desugaring etc
	return expr
}

// normalize converts expression to A-Normal Form
func (e *Elaborator) normalize(expr ast.Expr) (core.CoreExpr, error) {
	switch ex := expr.(type) {
	case *ast.Literal:
		return e.normalizeLiteral(ex)

	case *ast.Identifier:
		// Check if this is a nullary constructor (e.g., None, True, False)
		if ctorInfo, isConstructor := e.constructors[ex.Name]; isConstructor && ctorInfo.Arity == 0 {
			// Nullary constructor: transform None → $adt.make_Option_None
			// Note: No App node needed - factory is a value, not a function call
			factoryName := fmt.Sprintf("make_%s_%s", ctorInfo.TypeName, ctorInfo.CtorName)
			return &core.VarGlobal{
				CoreNode: e.makeNode(ex.Position()),
				Ref: core.GlobalRef{
					Module: "$adt",
					Name:   factoryName,
				},
			}, nil
		}
		// Check if this is an imported symbol
		if ref, ok := e.globalEnv[ex.Name]; ok {
			return &core.VarGlobal{
				CoreNode: e.makeNode(ex.Position()),
				Ref:      ref,
			}, nil
		}
		// Otherwise it's a local variable
		return &core.Var{
			CoreNode: e.makeNode(ex.Position()),
			Name:     ex.Name,
		}, nil

	case *ast.Lambda:
		return e.normalizeLambda(ex)

	case *ast.FuncLit:
		return e.normalizeFuncLit(ex)

	case *ast.BinaryOp:
		return e.normalizeBinaryOp(ex)

	case *ast.UnaryOp:
		return e.normalizeUnaryOp(ex)

	case *ast.If:
		return e.normalizeIf(ex)

	case *ast.Let:
		return e.normalizeLet(ex)

	case *ast.LetRec:
		return e.normalizeLetRec(ex)

	case *ast.Block:
		return e.normalizeBlock(ex)

	case *ast.FuncCall:
		return e.normalizeFuncCall(ex)

	case *ast.Record:
		return e.normalizeRecord(ex)

	case *ast.RecordAccess:
		return e.normalizeRecordAccess(ex)

	case *ast.List:
		return e.normalizeList(ex)

	case *ast.Tuple:
		return e.normalizeTuple(ex)

	case *ast.Match:
		return e.normalizeMatch(ex)

	default:
		if expr == nil {
			return nil, fmt.Errorf("normalization received nil expression")
		}
		return nil, fmt.Errorf("normalization not implemented for %T", expr)
	}
}

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

// normalizeLambda handles lambda expressions
func (e *Elaborator) normalizeLambda(lam *ast.Lambda) (core.CoreExpr, error) {
	// Extract parameter names
	params := make([]string, len(lam.Params))
	for i, p := range lam.Params {
		params[i] = p.Name
	}

	// Normalize body
	body, err := e.normalize(lam.Body)
	if err != nil {
		return nil, err
	}

	// Create Core Lambda node
	coreLam := &core.Lambda{
		CoreNode: e.makeNode(lam.Position()),
		Params:   params,
		Body:     body,
	}

	// Store effect annotations if present
	if len(lam.Effects) > 0 {
		// Validate and normalize effect names
		_, err := types.ElaborateEffectRow(lam.Effects)
		if err != nil {
			return nil, fmt.Errorf("invalid effect annotation: %w", err)
		}
		e.effectAnnots[coreLam.ID()] = lam.Effects
	}

	return coreLam, nil
}

// normalizeFuncLit handles function literal expressions (func(x) -> T { body })
// Desugars to Lambda: func(x: int) -> int { x + 1 } ≡ \x. x + 1
func (e *Elaborator) normalizeFuncLit(funcLit *ast.FuncLit) (core.CoreExpr, error) {
	// Extract parameter names (type annotations are handled by type checker)
	params := make([]string, len(funcLit.Params))
	for i, p := range funcLit.Params {
		params[i] = p.Name
	}

	// Normalize body
	body, err := e.normalize(funcLit.Body)
	if err != nil {
		return nil, err
	}

	// Create Core Lambda node
	coreLam := &core.Lambda{
		CoreNode: e.makeNode(funcLit.Position()),
		Params:   params,
		Body:     body,
	}

	// Store effect annotations if present
	if len(funcLit.Effects) > 0 {
		// Validate and normalize effect names
		_, err := types.ElaborateEffectRow(funcLit.Effects)
		if err != nil {
			return nil, fmt.Errorf("invalid effect annotation: %w", err)
		}
		e.effectAnnots[coreLam.ID()] = funcLit.Effects
	}

	return coreLam, nil
}

// normalizeBinaryOp handles binary operations with ANF transformation
func (e *Elaborator) normalizeBinaryOp(binop *ast.BinaryOp) (core.CoreExpr, error) {
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
	case "&&":
		op = core.OpAnd
	case "||":
		op = core.OpOr
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
	case "not":
		op = core.OpNot
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

// normalizeIf handles conditionals
func (e *Elaborator) normalizeIf(ifExpr *ast.If) (core.CoreExpr, error) {
	// Condition must be atomic
	cond, condBinds, err := e.normalizeToAtomic(ifExpr.Condition)
	if err != nil {
		return nil, err
	}

	// Branches can be complex
	thenBranch, err := e.normalize(ifExpr.Then)
	if err != nil {
		return nil, err
	}

	elseBranch, err := e.normalize(ifExpr.Else)
	if err != nil {
		return nil, err
	}

	result := &core.If{
		CoreNode: e.makeNode(ifExpr.Position()),
		Cond:     cond,
		Then:     thenBranch,
		Else:     elseBranch,
	}

	return e.wrapWithBindings(result, condBinds), nil
}

// normalizeLet handles let bindings
func (e *Elaborator) normalizeLet(let *ast.Let) (core.CoreExpr, error) {
	// Handle let statements without body (e.g., "let x = 1;" in a block)
	// These are created by the parser for semicolon-terminated lets
	// They should only appear as part of a Block, which will sequence them properly
	if let.Body == nil {
		// Just normalize the value and return it wrapped in a Let that binds to Unit
		// The Block normalization will thread this through properly
		value, err := e.normalize(let.Value)
		if err != nil {
			return nil, err
		}

		// Return a Let that binds the value but returns Unit
		// This allows the value to be computed (for side effects) and the binding to be visible
		// in subsequent expressions in the block
		return &core.Let{
			CoreNode: e.makeNode(let.Position()),
			Name:     let.Name,
			Value:    value,
			Body: &core.Lit{
				CoreNode: e.makeNode(let.Position()),
				Kind:     core.UnitLit,
				Value:    "()",
			},
		}, nil
	}

	// Check if it's recursive (let rec)
	isRec := false
	// For now, detect recursion by checking if the value references the name
	// This is simplified - full implementation would analyze the value expression

	if isRec {
		// Handle recursive binding
		value, err := e.normalize(let.Value)
		if err != nil {
			return nil, err
		}

		body, err := e.normalize(let.Body)
		if err != nil {
			return nil, err
		}

		return &core.LetRec{
			CoreNode: e.makeNode(let.Position()),
			Bindings: []core.RecBinding{{Name: let.Name, Value: value}},
			Body:     body,
		}, nil
	} else {
		// Non-recursive let with body (let x = 1 in x + 1)
		value, err := e.normalize(let.Value)
		if err != nil {
			return nil, err
		}

		body, err := e.normalize(let.Body)
		if err != nil {
			return nil, err
		}

		return &core.Let{
			CoreNode: e.makeNode(let.Position()),
			Name:     let.Name,
			Value:    value,
			Body:     body,
		}, nil
	}
}

// normalizeLetRec handles recursive let bindings
// Syntax: letrec name = value in body
func (e *Elaborator) normalizeLetRec(letrec *ast.LetRec) (core.CoreExpr, error) {
	// Normalize value (which can reference the name being bound)
	value, err := e.normalize(letrec.Value)
	if err != nil {
		return nil, err
	}

	// Handle missing body (REPL case)
	if letrec.Body == nil {
		// Return a LetRec that binds the value but returns Unit
		return &core.LetRec{
			CoreNode: e.makeNode(letrec.Position()),
			Bindings: []core.RecBinding{{Name: letrec.Name, Value: value}},
			Body: &core.Lit{
				CoreNode: e.makeNode(letrec.Position()),
				Kind:     core.UnitLit,
				Value:    "()",
			},
		}, nil
	}

	// Normal case: letrec with body
	body, err := e.normalize(letrec.Body)
	if err != nil {
		return nil, err
	}

	return &core.LetRec{
		CoreNode: e.makeNode(letrec.Position()),
		Bindings: []core.RecBinding{{Name: letrec.Name, Value: value}},
		Body:     body,
	}, nil
}

// normalizeBlock converts a block of semicolon-separated expressions
// into nested Let expressions: { e1; e2; e3 } => let _ = e1 in let _ = e2 in e3
func (e *Elaborator) normalizeBlock(block *ast.Block) (core.CoreExpr, error) {
	// Empty block: should not happen but handle gracefully
	if len(block.Exprs) == 0 {
		// Return unit literal
		return &core.Lit{
			CoreNode: e.makeNode(block.Position()),
			Kind:     core.UnitLit,
			Value:    "()",
		}, nil
	}

	// Single expression: just normalize it directly
	if len(block.Exprs) == 1 {
		return e.normalize(block.Exprs[0])
	}

	// Multiple expressions: convert to nested Lets
	// Start with the last expression (the return value)
	result, err := e.normalize(block.Exprs[len(block.Exprs)-1])
	if err != nil {
		return nil, err
	}

	// Work backwards through the expressions, wrapping each in a Let
	for i := len(block.Exprs) - 2; i >= 0; i-- {
		expr := block.Exprs[i]

		// Special case: if this is a Let with nil body (statement form),
		// normalize its value and use the let's name directly
		if letExpr, ok := expr.(*ast.Let); ok && letExpr.Body == nil {
			value, err := e.normalize(letExpr.Value)
			if err != nil {
				return nil, err
			}

			result = &core.Let{
				CoreNode: e.makeNode(letExpr.Position()),
				Name:     letExpr.Name, // Use the actual let name, not _block_N
				Value:    value,
				Body:     result, // Thread through to next expression
			}
		} else {
			// Regular expression: normalize and bind to a wildcard
			value, err := e.normalize(expr)
			if err != nil {
				return nil, err
			}

			// Use a wildcard name for the binding since we're discarding the result
			// Generate unique names to avoid conflicts
			bindingName := fmt.Sprintf("_block_%d", i)

			result = &core.Let{
				CoreNode: e.makeNode(block.Position()),
				Name:     bindingName,
				Value:    value,
				Body:     result,
			}
		}
	}

	return result, nil
}

// normalizeFuncCall handles function application
func (e *Elaborator) normalizeFuncCall(app *ast.FuncCall) (core.CoreExpr, error) {
	// Check if this is a constructor call
	if ident, ok := app.Func.(*ast.Identifier); ok {
		if ctorInfo, isConstructor := e.constructors[ident.Name]; isConstructor {
			// This is a constructor! Emit $adt factory call
			// Transform Some(x) → $adt.make_Option_Some(x)
			factoryName := fmt.Sprintf("make_%s_%s", ctorInfo.TypeName, ctorInfo.CtorName)

			// Normalize arguments to atomic values
			var allBindings []binding
			var atomicArgs []core.CoreExpr

			for _, arg := range app.Args {
				atomic, binds, err := e.normalizeToAtomic(arg)
				if err != nil {
					return nil, err
				}
				atomicArgs = append(atomicArgs, atomic)
				allBindings = append(allBindings, binds...)
			}

			// Create factory function reference
			factoryRef := &core.VarGlobal{
				CoreNode: e.makeNode(app.Position()),
				Ref: core.GlobalRef{
					Module: "$adt",
					Name:   factoryName,
				},
			}

			result := &core.App{
				CoreNode: e.makeNode(app.Position()),
				Func:     factoryRef,
				Args:     atomicArgs,
			}

			return e.wrapWithBindings(result, allBindings), nil
		}
	}

	// Not a constructor - handle as normal function call
	fun, err := e.normalize(app.Func)
	if err != nil {
		return nil, err
	}

	var allBindings []binding
	var atomicArgs []core.CoreExpr

	for _, arg := range app.Args {
		atomic, binds, err := e.normalizeToAtomic(arg)
		if err != nil {
			return nil, err
		}
		atomicArgs = append(atomicArgs, atomic)
		allBindings = append(allBindings, binds...)
	}

	// If function is not atomic, bind it too
	if !core.IsAtomic(fun) {
		freshVar := e.freshVar()
		allBindings = append(allBindings, binding{Name: freshVar, Value: fun})
		fun = &core.Var{
			CoreNode: e.makeNode(app.Position()),
			Name:     freshVar,
		}
	}

	result := &core.App{
		CoreNode: e.makeNode(app.Position()),
		Func:     fun,
		Args:     atomicArgs,
	}

	return e.wrapWithBindings(result, allBindings), nil
}

// normalizeRecord handles record construction
func (e *Elaborator) normalizeRecord(rec *ast.Record) (core.CoreExpr, error) {
	fields := make(map[string]core.CoreExpr)
	var allBindings []binding

	for i, field := range rec.Fields {
		value := field.Value
		name := field.Name
		atomic, binds, err := e.normalizeToAtomic(value)
		if err != nil {
			return nil, err
		}
		fields[name] = atomic
		allBindings = append(allBindings, binds...)
		_ = i // use i to avoid warning
	}

	result := &core.Record{
		CoreNode: e.makeNode(rec.Position()),
		Fields:   fields,
	}

	return e.wrapWithBindings(result, allBindings), nil
}

// normalizeRecordAccess handles field access
func (e *Elaborator) normalizeRecordAccess(acc *ast.RecordAccess) (core.CoreExpr, error) {
	record, binds, err := e.normalizeToAtomic(acc.Record)
	if err != nil {
		return nil, err
	}

	result := &core.RecordAccess{
		CoreNode: e.makeNode(acc.Position()),
		Record:   record,
		Field:    acc.Field,
	}

	return e.wrapWithBindings(result, binds), nil
}

// normalizeList handles list construction
func (e *Elaborator) normalizeList(list *ast.List) (core.CoreExpr, error) {
	var elements []core.CoreExpr
	var allBindings []binding

	for _, elem := range list.Elements {
		atomic, binds, err := e.normalizeToAtomic(elem)
		if err != nil {
			return nil, err
		}
		elements = append(elements, atomic)
		allBindings = append(allBindings, binds...)
	}

	result := &core.List{
		CoreNode: e.makeNode(list.Position()),
		Elements: elements,
	}

	return e.wrapWithBindings(result, allBindings), nil
}

// normalizeTuple handles tuple construction
func (e *Elaborator) normalizeTuple(tuple *ast.Tuple) (core.CoreExpr, error) {
	var elements []core.CoreExpr
	var allBindings []binding

	for _, elem := range tuple.Elements {
		atomic, binds, err := e.normalizeToAtomic(elem)
		if err != nil {
			return nil, err
		}
		elements = append(elements, atomic)
		allBindings = append(allBindings, binds...)
	}

	result := &core.Tuple{
		CoreNode: e.makeNode(tuple.Position()),
		Elements: elements,
	}

	return e.wrapWithBindings(result, allBindings), nil
}
