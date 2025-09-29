// Package pipeline provides compilation passes for AILANG
package pipeline

import (
	"fmt"

	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/types"
)

// OpLowerer performs type-directed lowering of intrinsic operations
type OpLowerer struct {
	typeEnv *types.TypeEnv
	errors  []error
}

// NewOpLowerer creates a new operation lowerer
func NewOpLowerer(typeEnv *types.TypeEnv) *OpLowerer {
	return &OpLowerer{
		typeEnv: typeEnv,
		errors:  []error{},
	}
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

	// For non-short-circuiting operations, recursively lower the arguments
	args := l.lowerExprs(intrinsic.Args)

	// Determine the type suffix based on the operation
	// TODO: Get actual types from typechecker
	// For MVP, use simple heuristics
	var typeSuffix string

	switch intrinsic.Op {
	case core.OpNot:
		typeSuffix = "Bool"
	case core.OpConcat:
		typeSuffix = "String"
	default:
		// For arithmetic and comparison, default to Int
		// A real implementation would inspect types
		typeSuffix = "Int"

		// Check if we have float literals
		if len(args) > 0 {
			if lit, ok := args[0].(*core.Lit); ok && lit.Kind == core.FloatLit {
				typeSuffix = "Float"
			}
		}
	}

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
