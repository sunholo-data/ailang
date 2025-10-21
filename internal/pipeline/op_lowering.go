// Package pipeline provides compilation passes for AILANG
package pipeline

import (
	"fmt"

	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/types"
)

// OpLowerer performs type-directed lowering of intrinsic operations
type OpLowerer struct {
	typeEnv             *types.TypeEnv
	resolvedConstraints map[uint64]*types.ResolvedConstraint // NodeID → resolved constraint
	bindings            map[string]core.CoreExpr             // Variable name → bound expression
	errors              []error
	CoreTI              types.CoreTypeInfo // Core NodeID → inferred types (for type-guided lowering)
}

// NewOpLowerer creates a new operation lowerer
func NewOpLowerer(typeEnv *types.TypeEnv, coreTI types.CoreTypeInfo) *OpLowerer {
	return &OpLowerer{
		typeEnv:             typeEnv,
		resolvedConstraints: make(map[uint64]*types.ResolvedConstraint),
		bindings:            make(map[string]core.CoreExpr),
		errors:              []error{},
		CoreTI:              coreTI,
	}
}

// SetResolvedConstraints sets the resolved constraints from type checking
func (l *OpLowerer) SetResolvedConstraints(constraints map[uint64]*types.ResolvedConstraint) {
	l.resolvedConstraints = constraints
}

// Lower performs type-directed lowering of intrinsic operations
func (l *OpLowerer) Lower(prog *core.Program) (*core.Program, error) {
	// Create new program with lowered expressions
	lowered := &core.Program{
		Decls: make([]core.CoreExpr, len(prog.Decls)),
		Meta:  prog.Meta, // Preserve metadata
	}

	for i, decl := range prog.Decls {
		loweredDecl := l.lowerExpr(decl)
		if loweredDecl == nil {
			return nil, fmt.Errorf("failed to lower declaration %d", i)
		}
		lowered.Decls[i] = loweredDecl
	}

	// Return any collected errors
	if len(l.errors) > 0 {
		return nil, l.errors[0] // TODO: Return all errors
	}

	return lowered, nil
}

// lowerExpr recursively lowers expressions
func (l *OpLowerer) lowerExpr(expr core.CoreExpr) core.CoreExpr {
	if expr == nil {
		return nil
	}

	switch e := expr.(type) {
	case *core.Intrinsic:
		return l.lowerIntrinsic(e)

	case *core.Let:
		// Track bindings before lowering (for type inference of concat)
		l.bindings[e.Name] = e.Value

		return &core.Let{
			CoreNode: e.CoreNode,
			Name:     e.Name,
			Value:    l.lowerExpr(e.Value),
			Body:     l.lowerExpr(e.Body),
		}

	case *core.LetRec:
		var bindings []core.RecBinding
		for _, b := range e.Bindings {
			bindings = append(bindings, core.RecBinding{
				Name:  b.Name,
				Value: l.lowerExpr(b.Value),
			})
		}
		return &core.LetRec{
			CoreNode: e.CoreNode,
			Bindings: bindings,
			Body:     l.lowerExpr(e.Body),
		}

	case *core.Lambda:
		return &core.Lambda{
			CoreNode: e.CoreNode,
			Params:   e.Params,
			Body:     l.lowerExpr(e.Body),
		}

	case *core.App:
		return &core.App{
			CoreNode: e.CoreNode,
			Func:     l.lowerExpr(e.Func),
			Args:     l.lowerExprs(e.Args),
		}

	case *core.If:
		return &core.If{
			CoreNode: e.CoreNode,
			Cond:     l.lowerExpr(e.Cond),
			Then:     l.lowerExpr(e.Then),
			Else:     l.lowerExpr(e.Else),
		}

	case *core.Match:
		var arms []core.MatchArm
		for _, arm := range e.Arms {
			arms = append(arms, core.MatchArm{
				Pattern: arm.Pattern,
				Guard:   l.lowerExpr(arm.Guard),
				Body:    l.lowerExpr(arm.Body),
			})
		}
		return &core.Match{
			CoreNode:   e.CoreNode,
			Scrutinee:  l.lowerExpr(e.Scrutinee),
			Arms:       arms,
			Exhaustive: e.Exhaustive,
		}

	case *core.BinOp:
		// Legacy BinOp nodes - convert to Intrinsic first
		var op core.IntrinsicOp
		switch e.Op {
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
			// Unknown operator, preserve as-is
			return &core.BinOp{
				CoreNode: e.CoreNode,
				Op:       e.Op,
				Left:     l.lowerExpr(e.Left),
				Right:    l.lowerExpr(e.Right),
			}
		}

		// Convert to Intrinsic and lower
		intrinsic := &core.Intrinsic{
			CoreNode: e.CoreNode,
			Op:       op,
			Args:     []core.CoreExpr{e.Left, e.Right},
		}
		return l.lowerIntrinsic(intrinsic)

	case *core.UnOp:
		// Legacy UnOp nodes - convert to Intrinsic first
		var op core.IntrinsicOp
		switch e.Op {
		case "-":
			op = core.OpNeg
		case "not":
			op = core.OpNot
		default:
			// Unknown operator, preserve as-is
			return &core.UnOp{
				CoreNode: e.CoreNode,
				Op:       e.Op,
				Operand:  l.lowerExpr(e.Operand),
			}
		}

		// Convert to Intrinsic and lower
		intrinsic := &core.Intrinsic{
			CoreNode: e.CoreNode,
			Op:       op,
			Args:     []core.CoreExpr{e.Operand},
		}
		return l.lowerIntrinsic(intrinsic)

	case *core.Record:
		fields := make(map[string]core.CoreExpr)
		for k, v := range e.Fields {
			fields[k] = l.lowerExpr(v)
		}
		return &core.Record{
			CoreNode: e.CoreNode,
			Fields:   fields,
		}

	case *core.RecordAccess:
		return &core.RecordAccess{
			CoreNode: e.CoreNode,
			Record:   l.lowerExpr(e.Record),
			Field:    e.Field,
		}

	case *core.List:
		return &core.List{
			CoreNode: e.CoreNode,
			Elements: l.lowerExprs(e.Elements),
		}

	// Atomic expressions and dictionary operations - pass through
	case *core.Var, *core.VarGlobal, *core.Lit, *core.DictRef, *core.DictAbs, *core.DictApp:
		return expr

	default:
		// Unknown type - pass through
		return expr
	}
}

// lowerExprs lowers a slice of expressions
func (l *OpLowerer) lowerExprs(exprs []core.CoreExpr) []core.CoreExpr {
	result := make([]core.CoreExpr, len(exprs))
	for i, e := range exprs {
		result[i] = l.lowerExpr(e)
	}
	return result
}

// isComparisonOrEqualityOp returns true for comparison and equality operators
// These operators return Bool but need to be lowered based on operand types
func isComparisonOrEqualityOp(op core.IntrinsicOp) bool {
	switch op {
	case core.OpLt, core.OpLe, core.OpGt, core.OpGe, core.OpEq, core.OpNe:
		return true
	default:
		return false
	}
}

// lowerIntrinsic performs type-directed lowering of an intrinsic operation
func (l *OpLowerer) lowerIntrinsic(intrinsic *core.Intrinsic) core.CoreExpr {
	// Special handling for short-circuiting boolean operations
	if intrinsic.Op == core.OpAnd {
		// Lower to: if left then right else false
		// This preserves short-circuit semantics
		left := l.lowerExpr(intrinsic.Args[0])
		right := l.lowerExpr(intrinsic.Args[1])
		return &core.If{
			CoreNode: intrinsic.CoreNode,
			Cond:     left,
			Then:     right,
			Else:     &core.Lit{CoreNode: intrinsic.CoreNode, Kind: core.BoolLit, Value: false},
		}
	}

	if intrinsic.Op == core.OpOr {
		// Lower to: if left then true else right
		// This preserves short-circuit semantics
		left := l.lowerExpr(intrinsic.Args[0])
		right := l.lowerExpr(intrinsic.Args[1])
		return &core.If{
			CoreNode: intrinsic.CoreNode,
			Cond:     left,
			Then:     &core.Lit{CoreNode: intrinsic.CoreNode, Kind: core.BoolLit, Value: true},
			Else:     right,
		}
	}

	// Determine the type suffix using type-guided lowering
	var typeSuffix string

	// For comparison and equality operators, use the operand type (not result type Bool)
	// For other operators, use the intrinsic's result type
	var typeNode uint64
	if isComparisonOrEqualityOp(intrinsic.Op) && len(intrinsic.Args) > 0 {
		// Use first operand's type for comparison/equality
		typeNode = intrinsic.Args[0].ID()
	} else {
		// Use intrinsic's own type for arithmetic, boolean, etc.
		typeNode = intrinsic.ID()
	}

	// First, try to use CoreTI (principal types from type inference)
	// This is the preferred method and eliminates ANF guessing
	if inferredType, ok := l.CoreTI.Get(typeNode); ok {
		head := types.Head(inferredType)
		switch head {
		case types.HeadInt:
			typeSuffix = "Int"
		case types.HeadFloat:
			typeSuffix = "Float"
		case types.HeadString:
			typeSuffix = "String"
		case types.HeadBool:
			typeSuffix = "Bool"
		case types.HeadList:
			typeSuffix = "List"
		default:
			// Unknown head - try resolved constraints as fallback
			if constraint, ok := l.resolvedConstraints[typeNode]; ok {
				typeSuffix = getTypeSuffixFromType(constraint.Type)
			} else {
				// M-DX4: For polymorphic operands, check if the INTRINSIC itself has a constraint
				// This handles cases where lambda parameters are polymorphic but the comparison
				// at the call site has been resolved
				if intrConstraint, ok := l.resolvedConstraints[intrinsic.ID()]; ok {
					typeSuffix = getTypeSuffixFromType(intrConstraint.Type)
				} else {
					// Last resort: use default based on operator
					typeSuffix = getDefaultTypeSuffix(intrinsic.Op)
				}
			}
		}
	} else {
		if constraint, ok := l.resolvedConstraints[typeNode]; ok {
			// Fallback to resolved constraints if CoreTI unavailable
			typeSuffix = getTypeSuffixFromType(constraint.Type)
		} else {
			// M-DX4: For polymorphic operands, check if the INTRINSIC itself has a constraint
			if intrConstraint, ok := l.resolvedConstraints[intrinsic.ID()]; ok {
				typeSuffix = getTypeSuffixFromType(intrConstraint.Type)
			} else {
				// Last resort: use default based on operator
				typeSuffix = getDefaultTypeSuffix(intrinsic.Op)
			}
		}
	}

	// For non-short-circuiting operations, recursively lower the arguments
	args := l.lowerExprs(intrinsic.Args)

	// Get the builtin name from the operator table
	builtinName, err := GetBuiltinName(intrinsic.Op, typeSuffix)
	if err != nil {
		// If the operator isn't supported for this type, add error and return unchanged
		l.AddError(err)
		return &core.Intrinsic{
			CoreNode: intrinsic.CoreNode,
			Op:       intrinsic.Op,
			Args:     args,
		}
	}

	// Create a builtin call
	// We use VarGlobal with module "$builtin" to represent builtins
	builtinRef := &core.VarGlobal{
		CoreNode: intrinsic.CoreNode,
		Ref: core.GlobalRef{
			Module: "$builtin",
			Name:   builtinName,
		},
	}

	// Create the application
	return &core.App{
		CoreNode: intrinsic.CoreNode,
		Func:     builtinRef,
		Args:     args,
	}
}

// AddError adds an error to the lowerer
func (l *OpLowerer) AddError(err error) {
	l.errors = append(l.errors, err)
}

// getDefaultTypeSuffix returns a default type suffix based on operator semantics
// This is used as a last resort when type information is unavailable
func getDefaultTypeSuffix(op core.IntrinsicOp) string {
	switch op {
	case core.OpNot:
		return "Bool"
	case core.OpConcat:
		// Default to String for backward compatibility
		// (List concatenation is less common in fallback scenarios)
		return "String"
	default:
		// Most operators default to Int
		return "Int"
	}
}

// getTypeSuffixFromType extracts the type suffix from a resolved type
// Maps TInt → "Int", TFloat → "Float", TBool → "Bool", TString → "String"
func getTypeSuffixFromType(t types.Type) string {
	switch t {
	case types.TInt:
		return "Int"
	case types.TFloat:
		return "Float"
	case types.TBool:
		return "Bool"
	case types.TString:
		return "String"
	default:
		// Check if it's a List type
		if app, ok := t.(*types.TApp); ok {
			if con, ok := app.Constructor.(*types.TCon); ok && con.Name == "List" {
				return "List"
			}
		}

		// For complex types, try to extract from string representation
		typeStr := t.String()
		// Handle common cases like "Int", "Float", "Bool", "String"
		if typeStr == "Int" || typeStr == "int" {
			return "Int"
		}
		if typeStr == "Float" || typeStr == "float" {
			return "Float"
		}
		if typeStr == "Bool" || typeStr == "bool" {
			return "Bool"
		}
		if typeStr == "String" || typeStr == "string" {
			return "String"
		}
		// Check for List in string form
		if len(typeStr) > 5 && typeStr[:5] == "List[" {
			return "List"
		}
		// Default to Int for unknown types (backward compatibility)
		return "Int"
	}
}

// CreateTypeMismatchError creates a structured type mismatch error for operators
func CreateTypeMismatchError(op core.IntrinsicOp, leftType, rightType types.Type) error {
	opStr := map[core.IntrinsicOp]string{
		core.OpAdd: "+", core.OpSub: "-", core.OpMul: "*", core.OpDiv: "/", core.OpMod: "%",
		core.OpEq: "==", core.OpNe: "!=", core.OpLt: "<", core.OpLe: "<=", core.OpGt: ">", core.OpGe: ">=",
		core.OpConcat: "++", core.OpAnd: "&&", core.OpOr: "||", core.OpNot: "not", core.OpNeg: "-",
	}[op]

	// For now, return a simple error
	// TODO: Use structured error when error encoder is available
	return fmt.Errorf("ELB_OP001: Operator '%s' has no implementation for types (%s, %s). Suggestion: Use matching types or add explicit conversion",
		opStr, leftType, rightType)
}
