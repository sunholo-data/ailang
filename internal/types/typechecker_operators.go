package types

import (
	"fmt"

	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/typedast"
)

// OperatorMethod returns the method name for an operator.
// Exported for use by the elaborator during dictionary-passing transformation.
// Binary operators map to their corresponding type class methods.
// Unary minus is handled as "neg" (negate) method in the Num class.
func OperatorMethod(op string, isUnary bool) string {
	// Handle unary operators
	if isUnary {
		switch op {
		case "-":
			return "neg" // Unary minus uses Num.neg method
		case "!":
			return "not" // Boolean not (if we have a Bool class)
		default:
			return ""
		}
	}

	// Binary operators
	switch op {
	case "+":
		return "add"
	case "-":
		return "sub"
	case "*":
		return "mul"
	case "/":
		return "div"
	case "==":
		return "eq"
	case "!=":
		return "neq"
	case "<":
		return "lt"
	case "<=":
		return "lte"
	case ">":
		return "gt"
	case ">=":
		return "gte"
	default:
		return ""
	}
}

// inferBinOp infers type of binary operation
func (tc *CoreTypeChecker) inferBinOp(ctx *InferenceContext, binop *core.BinOp) (*typedast.TypedBinOp, *TypeEnv, error) {
	// Infer operand types
	leftNode, _, err := tc.inferCore(ctx, binop.Left)
	if err != nil {
		return nil, ctx.env, err
	}

	rightNode, _, err := tc.inferCore(ctx, binop.Right)
	if err != nil {
		return nil, ctx.env, err
	}

	// Determine result type based on operator
	var resultType Type

	switch binop.Op {
	case "+", "-", "*", "/", "%":
		// Arithmetic operators - unify operand types first
		ctx.addConstraint(TypeEq{
			Left:  getType(leftNode),
			Right: getType(rightNode),
			Path:  []string{"arithmetic at " + binop.Span().String()},
		})

		// The result type is the same as the operand types
		resultType = getType(leftNode)

		// IMPORTANT: use the unified type to decide the most specific numeric class
		// This looks at constraints on the unified type, not individual nodes
		cls := tc.mostSpecificNumericClass(ctx, resultType)

		// Attach ONE class constraint to the unified type
		ctx.addConstraint(ClassConstraint{
			Class:  cls, // "Fractional" or "Num"
			Type:   resultType,
			Path:   []string{binop.Span().String()},
			NodeID: binop.ID(), // keep this for operator→method linking
		})

	case "++":
		// Concatenation: works for both strings and lists
		leftType := getType(leftNode)
		rightType := getType(rightNode)

		// DEBUG output (commented out - pollutes output)
		//fmt.Printf("DEBUG ++ operator: left=%T(%v), right=%T(%v)\n", leftType, leftType, rightType, rightType)

		// Check type patterns
		_, leftIsList := leftType.(*TList)
		_, rightIsList := rightType.(*TList)
		_, leftIsVar := leftType.(*TVar2)
		_, rightIsVar := rightType.(*TVar2)

		// Check if both are strings (TCon "String"/"string" or TString)
		leftIsString := false
		rightIsString := false

		if leftType == TString {
			leftIsString = true
		} else if leftCon, ok := leftType.(*TCon); ok && (leftCon.Name == "String" || leftCon.Name == "string") {
			leftIsString = true
		}

		if rightType == TString {
			rightIsString = true
		} else if rightCon, ok := rightType.(*TCon); ok && (rightCon.Name == "String" || rightCon.Name == "string") {
			rightIsString = true
		}

		// Decision tree:
		// 1. If at least one is a concrete list → list concat
		// 2. If at least one is a concrete string → string concat
		// 3. If both are type variables → list concat (more polymorphic)
		// 4. Otherwise → string concat (fallback)

		if leftIsList || rightIsList {
			// At least one is definitely a list → list concat
			elemType := ctx.freshTypeVar()

			ctx.addConstraint(TypeEq{
				Left:  leftType,
				Right: &TList{Element: elemType},
				Path:  []string{"list concat left at " + binop.Span().String()},
			})
			ctx.addConstraint(TypeEq{
				Left:  rightType,
				Right: &TList{Element: elemType},
				Path:  []string{"list concat right at " + binop.Span().String()},
			})

			resultType = &TList{Element: elemType}
		} else if leftIsString || rightIsString {
			// At least one is a concrete string → string concat
			// The type variable (if any) will be unified with String
			ctx.addConstraint(TypeEq{
				Left:  leftType,
				Right: TString,
				Path:  []string{"string concat left at " + binop.Span().String()},
			})
			ctx.addConstraint(TypeEq{
				Left:  rightType,
				Right: TString,
				Path:  []string{"string concat right at " + binop.Span().String()},
			})
			resultType = TString
		} else if leftIsVar && rightIsVar {
			// Both are type variables - default to list concat (more polymorphic)
			elemType := ctx.freshTypeVar()

			ctx.addConstraint(TypeEq{
				Left:  leftType,
				Right: &TList{Element: elemType},
				Path:  []string{"list concat left at " + binop.Span().String()},
			})
			ctx.addConstraint(TypeEq{
				Left:  rightType,
				Right: &TList{Element: elemType},
				Path:  []string{"list concat right at " + binop.Span().String()},
			})

			resultType = &TList{Element: elemType}
		} else {
			// Fallback: assume string concat
			ctx.addConstraint(TypeEq{
				Left:  leftType,
				Right: TString,
				Path:  []string{"string concat at " + binop.Span().String()},
			})
			ctx.addConstraint(TypeEq{
				Left:  rightType,
				Right: TString,
				Path:  []string{"string concat at " + binop.Span().String()},
			})
			resultType = TString
		}

	case "<", ">", "<=", ">=":
		// Comparison operators - require Ord constraint
		ctx.addConstraint(ClassConstraint{
			Class:  "Ord",
			Type:   getType(leftNode),
			Path:   []string{binop.Span().String()},
			NodeID: binop.ID(),
		})
		ctx.addConstraint(ClassConstraint{
			Class:  "Ord",
			Type:   getType(rightNode),
			Path:   []string{binop.Span().String()},
			NodeID: binop.ID(),
		})
		ctx.addConstraint(TypeEq{
			Left:  getType(leftNode),
			Right: getType(rightNode),
			Path:  []string{"comparison at " + binop.Span().String()},
		})
		resultType = TBool

	case "==", "!=":
		// Equality - require Eq constraint
		ctx.addConstraint(ClassConstraint{
			Class:  "Eq",
			Type:   getType(leftNode),
			Path:   []string{binop.Span().String()},
			NodeID: binop.ID(),
		})
		ctx.addConstraint(ClassConstraint{
			Class:  "Eq",
			Type:   getType(rightNode),
			Path:   []string{binop.Span().String()},
			NodeID: binop.ID(),
		})
		ctx.addConstraint(TypeEq{
			Left:  getType(leftNode),
			Right: getType(rightNode),
			Path:  []string{"equality at " + binop.Span().String()},
		})
		resultType = TBool

	case "&&", "||":
		// Boolean operators
		ctx.addConstraint(TypeEq{
			Left:  getType(leftNode),
			Right: TBool,
			Path:  []string{"boolean op at " + binop.Span().String()},
		})
		ctx.addConstraint(TypeEq{
			Left:  getType(rightNode),
			Right: TBool,
			Path:  []string{"boolean op at " + binop.Span().String()},
		})
		resultType = TBool

	default:
		return nil, ctx.env, fmt.Errorf("unknown binary operator: %s", binop.Op)
	}

	// Combine effects
	effects := combineEffects(getEffectRow(leftNode), getEffectRow(rightNode))

	return &typedast.TypedBinOp{
		TypedExpr: typedast.TypedExpr{
			NodeID:    binop.ID(),
			Span:      binop.Span(),
			Type:      resultType,
			EffectRow: effects,
			Core:      binop,
		},
		Op:    binop.Op,
		Left:  leftNode,
		Right: rightNode,
	}, ctx.env, nil
}

// inferUnOp infers type of unary operation
func (tc *CoreTypeChecker) inferUnOp(ctx *InferenceContext, unop *core.UnOp) (*typedast.TypedUnOp, *TypeEnv, error) {
	// Infer operand type
	operandNode, _, err := tc.inferCore(ctx, unop.Operand)
	if err != nil {
		return nil, ctx.env, err
	}

	var resultType Type

	switch unop.Op {
	case "-":
		// Negation - requires Num constraint
		ctx.addConstraint(ClassConstraint{
			Class:  "Num",
			Type:   getType(operandNode),
			Path:   []string{unop.Span().String()},
			NodeID: unop.ID(),
		})
		resultType = getType(operandNode)

	case "not":
		// Boolean negation
		ctx.addConstraint(TypeEq{
			Left:  getType(operandNode),
			Right: TBool,
			Path:  []string{"not at " + unop.Span().String()},
		})
		resultType = TBool

	default:
		return nil, ctx.env, fmt.Errorf("unknown unary operator: %s", unop.Op)
	}

	return &typedast.TypedUnOp{
		TypedExpr: typedast.TypedExpr{
			NodeID:    unop.ID(),
			Span:      unop.Span(),
			Type:      resultType,
			EffectRow: getEffectRow(operandNode),
			Core:      unop,
		},
		Op:      unop.Op,
		Operand: operandNode,
	}, ctx.env, nil
}

// mostSpecificNumericClass returns "Fractional" if any ClassConstraint on tUnified is Fractional,
// otherwise "Num". It ignores neutral classes (Eq/Ord/Show).
func (tc *CoreTypeChecker) mostSpecificNumericClass(ctx *InferenceContext, t Type) string {
	anyFractional := false

	// Walk all constraints currently in context
	for _, c := range ctx.qualifiedConstraints {
		if isNeutralClass(c.Class) { // Eq, Ord, Show
			continue
		}
		// Compare the *unified* types, not raw pointers
		if typesEqual(c.Type, t) {
			if c.Class == "Fractional" {
				anyFractional = true
			}
		}
	}
	if anyFractional {
		return "Fractional"
	}
	return "Num"
}

// isNeutralClass returns true for classes that don't influence numeric defaulting
func isNeutralClass(class string) bool {
	switch class {
	case "Eq", "Ord", "Show":
		return true
	default:
		return false
	}
}

// typesEqual compares types for equality (used for constraint matching)
func typesEqual(t1, t2 Type) bool {
	if t1 == nil || t2 == nil {
		return t1 == t2
	}

	switch typ1 := t1.(type) {
	case *TCon:
		if typ2, ok := t2.(*TCon); ok {
			return typ1.Name == typ2.Name
		}
	case *TVar:
		if typ2, ok := t2.(*TVar); ok {
			return typ1.Name == typ2.Name
		}
	case *TVar2:
		if typ2, ok := t2.(*TVar2); ok {
			return typ1.Name == typ2.Name
		}
	}

	// For more complex types, use string representation as fallback
	return t1.String() == t2.String()
}

// FillOperatorMethods fills in the Method field for resolved constraints
// by traversing the Core AST and matching NodeIDs (exported for REPL)
func (tc *CoreTypeChecker) FillOperatorMethods(expr core.CoreExpr) {
	// fmt.Printf("DEBUG FillOperatorMethods called with %T\n", expr)
	tc.walkCore(expr)
}

// walkCore recursively walks the Core AST to fill operator methods
func (tc *CoreTypeChecker) walkCore(expr core.CoreExpr) {
	if expr == nil {
		return
	}

	switch e := expr.(type) {
	case *core.BinOp:
		// If we have a resolved constraint for this node, fill in the method
		if rc, ok := tc.resolvedConstraints[e.ID()]; ok {
			method := OperatorMethod(e.Op, false)
			// fmt.Printf("DEBUG BinOp: node=%d, op='%s' -> method='%s'\n", e.ID(), e.Op, method)
			rc.Method = method
		}
		// else {
		// 	// fmt.Printf("DEBUG BinOp: node=%d, op='%s' (NO CONSTRAINT)\n", e.ID(), e.Op)
		// }
		// Recurse on operands
		tc.walkCore(e.Left)
		tc.walkCore(e.Right)

	case *core.UnOp:
		// Fill in the method name for unary operators
		if rc, ok := tc.resolvedConstraints[e.ID()]; ok {
			rc.Method = OperatorMethod(e.Op, true)
		}
		tc.walkCore(e.Operand)

	case *core.Intrinsic:
		// Fill in the method name for intrinsic operators
		if rc, ok := tc.resolvedConstraints[e.ID()]; ok {
			// Map IntrinsicOp to operator string for OperatorMethod lookup
			opStr := intrinsicOpToString(e.Op)
			if opStr != "" {
				method := OperatorMethod(opStr, len(e.Args) == 1)
				rc.Method = method
				// DEBUG
				if tc.debugMode {
					fmt.Printf("[DEBUG] Intrinsic %s (node %d): class=%s, type=%s, method=%s\n",
						opStr, e.ID(), rc.ClassName, rc.Type, method)
				}
			}
		} else if tc.debugMode {
			fmt.Printf("[DEBUG] Intrinsic node %d has no resolved constraint\n", e.ID())
		}
		// Recurse on arguments
		for _, arg := range e.Args {
			tc.walkCore(arg)
		}

	case *core.Let:
		tc.walkCore(e.Value)
		tc.walkCore(e.Body)

	case *core.LetRec:
		for _, binding := range e.Bindings {
			tc.walkCore(binding.Value)
		}
		tc.walkCore(e.Body)

	case *core.Lambda:
		tc.walkCore(e.Body)

	case *core.App:
		tc.walkCore(e.Func)
		for _, arg := range e.Args {
			tc.walkCore(arg)
		}

	case *core.If:
		tc.walkCore(e.Cond)
		tc.walkCore(e.Then)
		tc.walkCore(e.Else)

	case *core.Match:
		tc.walkCore(e.Scrutinee)
		for _, arm := range e.Arms {
			tc.walkCore(arm.Body)
		}

	case *core.Record:
		for _, field := range e.Fields {
			tc.walkCore(field)
		}

	case *core.RecordAccess:
		tc.walkCore(e.Record)

	case *core.List:
		for _, elem := range e.Elements {
			tc.walkCore(elem)
		}

	// Atomic expressions don't need recursion
	case *core.Var, *core.Lit, *core.DictRef:
		return
	}
}

// intrinsicOpToString converts IntrinsicOp to operator string for method lookup
func intrinsicOpToString(op core.IntrinsicOp) string {
	switch op {
	case core.OpAdd:
		return "+"
	case core.OpSub:
		return "-"
	case core.OpMul:
		return "*"
	case core.OpDiv:
		return "/"
	case core.OpMod:
		return "%"
	case core.OpEq:
		return "=="
	case core.OpNe:
		return "!="
	case core.OpLt:
		return "<"
	case core.OpLe:
		return "<="
	case core.OpGt:
		return ">"
	case core.OpGe:
		return ">="
	case core.OpConcat:
		return "++"
	case core.OpAnd:
		return "&&"
	case core.OpOr:
		return "||"
	case core.OpNot:
		return "!"
	case core.OpNeg:
		return "-"
	default:
		return ""
	}
}
