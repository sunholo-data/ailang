package elaborate

import (
	"fmt"
	"github.com/sunholo/ailang/internal/ast"
	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/types"
)

// Elaborator transforms surface AST to Core ANF
type Elaborator struct {
	nextID       uint64
	surfaceSpans map[uint64]ast.Pos // Map Core IDs to surface positions
	freshVarNum  int                // For generating fresh variable names
}

// NewElaborator creates a new elaborator
func NewElaborator() *Elaborator {
	return &Elaborator{
		nextID:       1,
		surfaceSpans: make(map[uint64]ast.Pos),
		freshVarNum:  0,
	}
}

// Elaborate transforms a surface program to Core ANF
func (e *Elaborator) Elaborate(prog *ast.Program) (*core.Program, error) {
	if prog.Module == nil {
		return &core.Program{}, nil
	}

	var coreDecls []core.CoreExpr
	for _, decl := range prog.Module.Decls {
		coreExpr, err := e.elaborateNode(decl)
		if err != nil {
			return nil, err
		}
		if coreExpr != nil {
			coreDecls = append(coreDecls, coreExpr)
		}
	}

	return &core.Program{Decls: coreDecls}, nil
}

// elaborateNode handles any AST node
func (e *Elaborator) elaborateNode(node ast.Node) (core.CoreExpr, error) {
	switch n := node.(type) {
	case ast.Expr:
		return e.elaborateExpr(n)
	case *ast.FuncDecl:
		return e.elaborateFuncDecl(n)
	default:
		return nil, fmt.Errorf("elaboration not implemented for %T", node)
	}
}

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
		return &core.Var{
			CoreNode: e.makeNode(ex.Position()),
			Name:     ex.Name,
		}, nil

	case *ast.Lambda:
		return e.normalizeLambda(ex)

	case *ast.BinaryOp:
		return e.normalizeBinaryOp(ex)

	case *ast.UnaryOp:
		return e.normalizeUnaryOp(ex)

	case *ast.If:
		return e.normalizeIf(ex)

	case *ast.Let:
		return e.normalizeLet(ex)

	case *ast.FuncCall:
		return e.normalizeFuncCall(ex)

	case *ast.Record:
		return e.normalizeRecord(ex)

	case *ast.RecordAccess:
		return e.normalizeRecordAccess(ex)

	case *ast.List:
		return e.normalizeList(ex)

	case *ast.Match:
		return e.normalizeMatch(ex)

	default:
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

	return &core.Lambda{
		CoreNode: e.makeNode(lam.Position()),
		Params:   params,
		Body:     body,
	}, nil
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

	// Create the binary operation
	result := &core.BinOp{
		CoreNode: e.makeNode(binop.Position()),
		Op:       binop.Op,
		Left:     left,
		Right:    right,
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

	result := &core.UnOp{
		CoreNode: e.makeNode(unop.Position()),
		Op:       unop.Op,
		Operand:  operand,
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
		// Non-recursive let
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

// normalizeFuncCall handles function application
func (e *Elaborator) normalizeFuncCall(app *ast.FuncCall) (core.CoreExpr, error) {
	// Function can be complex, but args must be atomic
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

// normalizeMatch handles pattern matching
func (e *Elaborator) normalizeMatch(match *ast.Match) (core.CoreExpr, error) {
	// Scrutinee must be atomic
	scrutinee, binds, err := e.normalizeToAtomic(match.Expr)
	if err != nil {
		return nil, err
	}

	// Convert arms
	var arms []core.MatchArm
	for _, caseClause := range match.Cases {
		pattern, err := e.elaboratePattern(caseClause.Pattern)
		if err != nil {
			return nil, err
		}

		body, err := e.normalize(caseClause.Body)
		if err != nil {
			return nil, err
		}

		arms = append(arms, core.MatchArm{
			Pattern: pattern,
			Body:    body,
		})
	}

	result := &core.Match{
		CoreNode:   e.makeNode(match.Position()),
		Scrutinee:  scrutinee,
		Arms:       arms,
		Exhaustive: false, // Will be set by typechecker
	}

	return e.wrapWithBindings(result, binds), nil
}

// elaboratePattern converts surface pattern to core pattern
func (e *Elaborator) elaboratePattern(pat ast.Pattern) (core.CorePattern, error) {
	switch p := pat.(type) {
	case *ast.Identifier:
		return &core.VarPattern{Name: p.Name}, nil
	case *ast.Literal:
		return &core.LitPattern{Value: p.Value}, nil
	case *ast.WildcardPattern:
		return &core.WildcardPattern{}, nil
	default:
		return nil, fmt.Errorf("pattern elaboration not implemented for %T", pat)
	}
}

// elaborateFuncDecl handles function declarations
func (e *Elaborator) elaborateFuncDecl(fn *ast.FuncDecl) (core.CoreExpr, error) {
	// Convert to lambda
	lambda := &ast.Lambda{
		Params: fn.Params,
		Body:   fn.Body,
		Pos:    fn.Pos,
	}

	value, err := e.normalizeLambda(lambda)
	if err != nil {
		return nil, err
	}

	// Wrap in let rec if recursive
	return &core.LetRec{
		CoreNode: e.makeNode(fn.Position()),
		Bindings: []core.RecBinding{{Name: fn.Name, Value: value}},
		Body:     &core.Var{CoreNode: e.makeNode(fn.Position()), Name: fn.Name},
	}, nil
}

// Helper types and functions

type binding struct {
	Name  string
	Value core.CoreExpr
}

// normalizeToAtomic ensures expression is atomic, introducing let bindings if needed
func (e *Elaborator) normalizeToAtomic(expr ast.Expr) (core.CoreExpr, []binding, error) {
	normalized, err := e.normalize(expr)
	if err != nil {
		return nil, nil, err
	}

	if core.IsAtomic(normalized) {
		return normalized, nil, nil
	}

	// Need to bind non-atomic expression
	freshName := e.freshVar()
	bind := binding{Name: freshName, Value: normalized}
	varRef := &core.Var{
		CoreNode: e.makeNode(expr.Position()),
		Name:     freshName,
	}

	return varRef, []binding{bind}, nil
}

// wrapWithBindings wraps expression with let bindings
func (e *Elaborator) wrapWithBindings(expr core.CoreExpr, bindings []binding) core.CoreExpr {
	result := expr
	// Apply bindings in reverse order (innermost first)
	for i := len(bindings) - 1; i >= 0; i-- {
		bind := bindings[i]
		result = &core.Let{
			CoreNode: e.makeNode(bind.Value.Span()),
			Name:     bind.Name,
			Value:    bind.Value,
			Body:     result,
		}
	}
	return result
}

// makeNode creates a new CoreNode with unique ID
func (e *Elaborator) makeNode(pos ast.Pos) core.CoreNode {
	id := e.nextID
	e.nextID++
	e.surfaceSpans[id] = pos
	return core.CoreNode{
		NodeID:   id,
		CoreSpan: pos,
		OrigSpan: pos,
	}
}

// freshVar generates a fresh variable name
func (e *Elaborator) freshVar() string {
	e.freshVarNum++
	return fmt.Sprintf("$tmp%d", e.freshVarNum)
}

// GetSurfaceSpan retrieves the original surface span for a Core node ID
func (e *Elaborator) GetSurfaceSpan(nodeID uint64) (ast.Pos, bool) {
	span, ok := e.surfaceSpans[nodeID]
	return span, ok
}

// ElaborateWithDictionaries transforms operators to dictionary calls
// This is the second pass after type checking
func ElaborateWithDictionaries(prog *core.Program, resolved map[uint64]*types.ResolvedConstraint) (*core.Program, error) {
	elaborator := &DictElaborator{
		resolved:    resolved,
		freshVarNum: 0,
	}

	// Transform each declaration
	var newDecls []core.CoreExpr
	for _, decl := range prog.Decls {
		transformed := elaborator.transformExpr(decl)
		newDecls = append(newDecls, transformed)
	}

	return &core.Program{Decls: newDecls}, nil
}

// DictElaborator handles dictionary transformation
type DictElaborator struct {
	resolved    map[uint64]*types.ResolvedConstraint
	freshVarNum int
}

// transformExpr recursively transforms Core expressions
func (de *DictElaborator) transformExpr(expr core.CoreExpr) core.CoreExpr {
	if expr == nil {
		return nil
	}

	switch e := expr.(type) {
	case *core.BinOp:
		// Check if this operator has a resolved constraint
		if rc, ok := de.resolved[e.ID()]; ok && rc.Method != "" {
			// Guard against nil Type in resolved constraint
			if rc.Type == nil {
				// Skip dictionary transformation if type is nil
				return &core.BinOp{
					CoreNode: e.CoreNode,
					Op:       e.Op,
					Left:     de.transformExpr(e.Left),
					Right:    de.transformExpr(e.Right),
				}
			}

			// Transform to dictionary application
			// First transform the operands
			left := de.transformExpr(e.Left)
			right := de.transformExpr(e.Right)

			// Create dictionary reference
			typeName := types.NormalizeTypeName(rc.Type)
			// fmt.Printf("DEBUG ELABORATE: BinOp NodeID=%d, Class=%s, Type=%v, NormalizedType=%s, Method=%s\n",
			// 	e.ID(), rc.ClassName, rc.Type, typeName, rc.Method)
			dictRef := &core.DictRef{
				CoreNode:  e.CoreNode,
				ClassName: rc.ClassName,
				TypeName:  typeName,
			}

			// Create dictionary application directly

			// Build the ANF structure properly:
			// For now, just use DictApp directly with DictRef as the dictionary
			// This is valid ANF since DictRef is atomic
			return &core.DictApp{
				CoreNode: e.CoreNode,
				Dict:     dictRef,
				Method:   rc.Method,
				Args:     []core.CoreExpr{left, right},
			}
		}

		// No dictionary transformation needed, just recurse
		return &core.BinOp{
			CoreNode: e.CoreNode,
			Op:       e.Op,
			Left:     de.transformExpr(e.Left),
			Right:    de.transformExpr(e.Right),
		}

	case *core.UnOp:
		// Check if this operator has a resolved constraint
		if rc, ok := de.resolved[e.ID()]; ok && rc.Method != "" {
			// Guard against nil Type in resolved constraint
			if rc.Type == nil {
				// Skip dictionary transformation if type is nil
				return &core.UnOp{
					CoreNode: e.CoreNode,
					Op:       e.Op,
					Operand:  de.transformExpr(e.Operand),
				}
			}

			// Transform to dictionary application
			operand := de.transformExpr(e.Operand)

			// Create dictionary reference
			typeName := types.NormalizeTypeName(rc.Type)
			dictRef := &core.DictRef{
				CoreNode:  e.CoreNode,
				ClassName: rc.ClassName,
				TypeName:  typeName,
			}

			// Create dictionary application directly

			// Build ANF structure properly with DictRef directly in DictApp
			return &core.DictApp{
				CoreNode: e.CoreNode,
				Dict:     dictRef,
				Method:   rc.Method,
				Args:     []core.CoreExpr{operand},
			}
		}

		// No transformation needed
		return &core.UnOp{
			CoreNode: e.CoreNode,
			Op:       e.Op,
			Operand:  de.transformExpr(e.Operand),
		}

	case *core.Let:
		return &core.Let{
			CoreNode: e.CoreNode,
			Name:     e.Name,
			Value:    de.transformExpr(e.Value),
			Body:     de.transformExpr(e.Body),
		}

	case *core.LetRec:
		var newBindings []core.RecBinding
		for _, binding := range e.Bindings {
			newBindings = append(newBindings, core.RecBinding{
				Name:  binding.Name,
				Value: de.transformExpr(binding.Value),
			})
		}
		return &core.LetRec{
			CoreNode: e.CoreNode,
			Bindings: newBindings,
			Body:     de.transformExpr(e.Body),
		}

	case *core.Lambda:
		return &core.Lambda{
			CoreNode: e.CoreNode,
			Params:   e.Params,
			Body:     de.transformExpr(e.Body),
		}

	case *core.App:
		var newArgs []core.CoreExpr
		for _, arg := range e.Args {
			newArgs = append(newArgs, de.transformExpr(arg))
		}
		return &core.App{
			CoreNode: e.CoreNode,
			Func:     de.transformExpr(e.Func),
			Args:     newArgs,
		}

	case *core.If:
		return &core.If{
			CoreNode: e.CoreNode,
			Cond:     de.transformExpr(e.Cond),
			Then:     de.transformExpr(e.Then),
			Else:     de.transformExpr(e.Else),
		}

	case *core.Match:
		var newArms []core.MatchArm
		for _, arm := range e.Arms {
			newArms = append(newArms, core.MatchArm{
				Pattern: arm.Pattern,
				Body:    de.transformExpr(arm.Body),
			})
		}
		return &core.Match{
			CoreNode:   e.CoreNode,
			Scrutinee:  de.transformExpr(e.Scrutinee),
			Arms:       newArms,
			Exhaustive: e.Exhaustive,
		}

	case *core.Record:
		newFields := make(map[string]core.CoreExpr)
		for k, v := range e.Fields {
			newFields[k] = de.transformExpr(v)
		}
		return &core.Record{
			CoreNode: e.CoreNode,
			Fields:   newFields,
		}

	case *core.RecordAccess:
		return &core.RecordAccess{
			CoreNode: e.CoreNode,
			Record:   de.transformExpr(e.Record),
			Field:    e.Field,
		}

	case *core.List:
		var newElements []core.CoreExpr
		for _, elem := range e.Elements {
			newElements = append(newElements, de.transformExpr(elem))
		}
		return &core.List{
			CoreNode: e.CoreNode,
			Elements: newElements,
		}

	// Atomic expressions - return as is
	case *core.Var, *core.Lit, *core.DictRef:
		return expr

	// Already dictionary nodes - preserve
	case *core.DictAbs, *core.DictApp:
		return expr

	default:
		// Unknown type - return as is
		return expr
	}
}
