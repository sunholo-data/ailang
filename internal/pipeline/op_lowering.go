// Package pipeline provides compilation passes for AILANG
package pipeline

import (
	"fmt"
	"os"

	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/types"
)

// FallbackEvent tracks when CoreTI lookup misses during lowering
type FallbackEvent struct {
	Op       core.IntrinsicOp
	NodeID   uint64
	Fallback string // "CoreTI-hit", "ResolvedConstraints", "Default"
	Location string // Source location if available
}

// OpLowerer performs type-directed lowering of intrinsic operations
type OpLowerer struct {
	typeEnv             *types.TypeEnv
	resolvedConstraints map[uint64]*types.ResolvedConstraint // NodeID → resolved constraint
	bindings            map[string]core.CoreExpr             // Variable name → bound expression
	errors              []error
	CoreTI              types.CoreTypeInfo // Core NodeID → inferred types (for type-guided lowering)

	// M-DX4: Telemetry for tracking CoreTI coverage (gated by --debug-compile)
	telemetry       []FallbackEvent
	enableTelemetry bool
}

// NewOpLowerer creates a new operation lowerer
func NewOpLowerer(typeEnv *types.TypeEnv, coreTI types.CoreTypeInfo) *OpLowerer {
	return &OpLowerer{
		typeEnv:             typeEnv,
		resolvedConstraints: make(map[uint64]*types.ResolvedConstraint),
		bindings:            make(map[string]core.CoreExpr),
		errors:              []error{},
		CoreTI:              coreTI,
		telemetry:           []FallbackEvent{},
		enableTelemetry:     false, // Enable with SetEnableTelemetry(true)
	}
}

// SetEnableTelemetry enables/disables fallback telemetry tracking
func (l *OpLowerer) SetEnableTelemetry(enable bool) {
	l.enableTelemetry = enable
}

// GetTelemetry returns collected telemetry events
func (l *OpLowerer) GetTelemetry() []FallbackEvent {
	return l.telemetry
}

// trackFallback records a fallback event for telemetry
func (l *OpLowerer) trackFallback(op core.IntrinsicOp, nodeID uint64, fallback string, location string) {
	if l.enableTelemetry {
		l.telemetry = append(l.telemetry, FallbackEvent{
			Op:       op,
			NodeID:   nodeID,
			Fallback: fallback,
			Location: location,
		})
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
		Meta:  prog.Meta, // Preserve metadata (overwritten below if contracts exist)
	}

	for i, decl := range prog.Decls {
		loweredDecl := l.lowerExpr(decl)
		if loweredDecl == nil {
			return nil, fmt.Errorf("failed to lower declaration %d", i)
		}
		lowered.Decls[i] = loweredDecl
	}

	// M-CONTRACTS-OPLOWERING: Also lower contract expressions in Meta.
	// Contract predicates (requires/ensures) contain Intrinsic nodes that
	// need the same lowering as regular code to avoid requiring --experimental-binop-shim.
	if prog.Meta != nil {
		lowered.Meta = make(map[string]*core.DeclMeta, len(prog.Meta))
		for name, meta := range prog.Meta {
			if len(meta.Contracts) == 0 {
				lowered.Meta[name] = meta
				continue
			}
			// Deep-copy meta to avoid mutating the original program
			newMeta := *meta
			newMeta.Contracts = make([]*core.Contract, len(meta.Contracts))
			for i, contract := range meta.Contracts {
				newContract := *contract
				if contract.Expr != nil {
					newContract.Expr = l.lowerExpr(contract.Expr)
				}
				newMeta.Contracts[i] = &newContract
			}
			lowered.Meta[name] = &newMeta
		}
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

	// M-DX4: Get location for telemetry (if available)
	location := ""
	pos := intrinsic.OriginalSpan()
	if pos.Line > 0 {
		location = fmt.Sprintf("line %d", pos.Line)
	}

	// First, try to use CoreTI (principal types from type inference)
	// This is the preferred method and eliminates ANF guessing
	if inferredType, ok := l.CoreTI.Get(typeNode); ok {
		head := types.Head(inferredType)

		// M-DX4 DEBUG: Log the type and head we got
		if l.enableTelemetry {
			fmt.Fprintf(os.Stderr, "[DEBUG M-DX4] NodeID %d: type=%v, head=%v\n", typeNode, inferredType, head)
		}

		switch head {
		case types.HeadInt:
			typeSuffix = "Int"
			l.trackFallback(intrinsic.Op, typeNode, "CoreTI-hit", location)
		case types.HeadFloat:
			typeSuffix = "Float"
			l.trackFallback(intrinsic.Op, typeNode, "CoreTI-hit", location)
		case types.HeadString:
			typeSuffix = "String"
			l.trackFallback(intrinsic.Op, typeNode, "CoreTI-hit", location)
		case types.HeadBool:
			typeSuffix = "Bool"
			l.trackFallback(intrinsic.Op, typeNode, "CoreTI-hit", location)
		case types.HeadList:
			typeSuffix = "List"
			l.trackFallback(intrinsic.Op, typeNode, "CoreTI-hit", location)
		default:
			// Unknown head (TVar or unknown) - try resolved constraints as fallback
			if constraint, ok := l.resolvedConstraints[typeNode]; ok {
				typeSuffix = getTypeSuffixFromType(constraint.Type)
				l.trackFallback(intrinsic.Op, typeNode, "ResolvedConstraints", location)
			} else {
				// M-DX4: For polymorphic operands, check if the INTRINSIC itself has a constraint
				// This handles cases where lambda parameters are polymorphic but the comparison
				// at the call site has been resolved
				if intrConstraint, ok := l.resolvedConstraints[intrinsic.ID()]; ok {
					typeSuffix = getTypeSuffixFromType(intrConstraint.Type)
					l.trackFallback(intrinsic.Op, intrinsic.ID(), "ResolvedConstraints-intrinsic", location)
				} else {
					// M-FIX-FLOAT-OP: For arithmetic operators, try operand types before defaulting
					// This fixes float operators in pure functions where the intrinsic node
					// has a type variable but the operands have concrete types
					typeSuffix = l.tryOperandTypes(intrinsic)
					if typeSuffix == "" {
						// M-WASM-DICTIONARY-DISPATCH: For comparison/equality ops with unknown
						// type, leave as Intrinsic for runtime dispatch via binop shim.
						// This avoids incorrectly defaulting to eq_Int for string comparisons
						// inside lambdas passed to polymorphic HOFs like any().
						if isComparisonOrEqualityOp(intrinsic.Op) {
							l.trackFallback(intrinsic.Op, typeNode, "Deferred-to-shim", location)
							return &core.Intrinsic{
								CoreNode: intrinsic.CoreNode,
								Op:       intrinsic.Op,
								Args:     l.lowerExprs(intrinsic.Args),
							}
						}
						// Last resort for non-comparison ops: use default based on operator
						typeSuffix = getDefaultTypeSuffix(intrinsic.Op)
						l.trackFallback(intrinsic.Op, typeNode, "Default", location)
					} else {
						l.trackFallback(intrinsic.Op, typeNode, "OperandType", location)
					}
				}
			}
		}
	} else {
		// CoreTI miss - track this as a gap
		l.trackFallback(intrinsic.Op, typeNode, "CoreTI-miss", location)

		if constraint, ok := l.resolvedConstraints[typeNode]; ok {
			// Fallback to resolved constraints if CoreTI unavailable
			typeSuffix = getTypeSuffixFromType(constraint.Type)
		} else {
			// M-DX4: For polymorphic operands, check if the INTRINSIC itself has a constraint
			if intrConstraint, ok := l.resolvedConstraints[intrinsic.ID()]; ok {
				typeSuffix = getTypeSuffixFromType(intrConstraint.Type)
			} else {
				// M-FIX-FLOAT-OP: For arithmetic operators, try operand types before defaulting
				typeSuffix = l.tryOperandTypes(intrinsic)
				if typeSuffix == "" {
					// M-WASM-DICTIONARY-DISPATCH: Same deferral for CoreTI-miss path
					if isComparisonOrEqualityOp(intrinsic.Op) {
						return &core.Intrinsic{
							CoreNode: intrinsic.CoreNode,
							Op:       intrinsic.Op,
							Args:     l.lowerExprs(intrinsic.Args),
						}
					}
					// Last resort: use default based on operator
					typeSuffix = getDefaultTypeSuffix(intrinsic.Op)
				}
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
		// M-CONCAT-DISAMBIG Phase 2: `++` is lists only.
		return "List"
	default:
		// Most operators default to Int
		return "Int"
	}
}

// tryOperandTypes attempts to determine the type suffix from operand types.
// M-FIX-FLOAT-OP: This fixes float operators in pure functions where the intrinsic
// node has a type variable but the operands have concrete types (e.g., float parameters).
// Returns empty string if operand types cannot be determined.
func (l *OpLowerer) tryOperandTypes(intrinsic *core.Intrinsic) string {
	if len(intrinsic.Args) == 0 {
		return ""
	}

	// Check first operand's type in CoreTI
	firstArg := intrinsic.Args[0]
	if inferredType, ok := l.CoreTI.Get(firstArg.ID()); ok {
		head := types.Head(inferredType)
		switch head {
		case types.HeadFloat:
			return "Float"
		case types.HeadInt:
			return "Int"
		case types.HeadString:
			return "String"
		case types.HeadBool:
			return "Bool"
		case types.HeadList:
			return "List"
		}
	}

	// If first operand is a Var, check if it has a known type in resolved constraints
	if varExpr, ok := firstArg.(*core.Var); ok {
		// Check if there's a binding for this variable with a known type
		if boundExpr, ok := l.bindings[varExpr.Name]; ok {
			if boundType, ok := l.CoreTI.Get(boundExpr.ID()); ok {
				head := types.Head(boundType)
				switch head {
				case types.HeadFloat:
					return "Float"
				case types.HeadInt:
					return "Int"
				case types.HeadString:
					return "String"
				case types.HeadBool:
					return "Bool"
				}
			}
		}
	}

	return ""
}

// getTypeSuffixFromType extracts the type suffix from a resolved type
// Maps TInt → "Int", TFloat → "Float", TBool → "Bool", TString → "String"
//
// IMPORTANT: This function is purely SHALLOW - it only checks top-level type constructors.
// It NEVER calls t.String() or traverses nested type structures, because cyclic types
// (e.g., List[NPCState] where NPCState contains List[NPCState]) would cause infinite loops.
// See M-PERF2 design doc for details.
func getTypeSuffixFromType(t types.Type) string {
	// Direct primitive singletons - O(1) pointer comparison
	switch t {
	case types.TInt:
		return "Int"
	case types.TFloat:
		return "Float"
	case types.TBool:
		return "Bool"
	case types.TString:
		return "String"
	}

	// Check if it's a TCon (type constructor) directly
	if con, ok := t.(*types.TCon); ok {
		switch con.Name {
		case "Int", "int":
			return "Int"
		case "Float", "float":
			return "Float"
		case "Bool", "bool":
			return "Bool"
		case "String", "string":
			return "String"
		case "list": // DX-17: canonical form is lowercase
			return "List"
		}
	}

	// Shallow check for type applications (e.g., list[a])
	// Only inspect the top-level constructor, NEVER traverse arguments
	// DX-17: canonical form is lowercase "list"
	if app, ok := t.(*types.TApp); ok {
		if con, ok := app.Constructor.(*types.TCon); ok {
			if con.Name == "list" {
				return "List"
			}
		}
	}

	// Default to Int for unknown types (backward compatibility)
	// NO t.String() - cannot risk traversing cyclic types
	return "Int"
}

// CreateTypeMismatchError creates a structured type mismatch error for operators
func CreateTypeMismatchError(op core.IntrinsicOp, leftType, rightType types.Type) error {
	opStr := map[core.IntrinsicOp]string{
		core.OpAdd: "+", core.OpSub: "-", core.OpMul: "*", core.OpDiv: "/", core.OpMod: "%",
		core.OpEq: "==", core.OpNe: "!=", core.OpLt: "<", core.OpLe: "<=", core.OpGt: ">", core.OpGe: ">=",
		core.OpConcat: "++", core.OpAnd: "&&", core.OpOr: "||", core.OpNot: "not", core.OpNeg: "-",
		core.OpBitwiseAnd: "&", core.OpBitwiseXor: "^", core.OpBitwiseNot: "~",
		core.OpShiftLeft: "<<", core.OpShiftRight: ">>",
	}[op]

	// For now, return a simple error
	// TODO: Use structured error when error encoder is available
	return fmt.Errorf("ELB_OP001: Operator '%s' has no implementation for types (%s, %s). Suggestion: Use matching types or add explicit conversion",
		opStr, leftType, rightType)
}
