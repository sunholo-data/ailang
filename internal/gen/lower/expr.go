package lower

import (
	"fmt"

	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/gen/stmt"
	"github.com/sunholo/ailang/internal/types"
)

// LowerExpr converts a Core expression into a Statement IR expression.
// For complex expressions (Let, Match, If-as-statement), it may also
// produce statements that must be prepended to the enclosing block.
// Use LowerBlock for let-chain flattening; this function handles the
// expression-level conversion.
func LowerExpr(e core.CoreExpr, cti types.CoreTypeInfo) stmt.Expr {
	if e == nil {
		return stmt.LitUnit{}
	}
	return lowerExpr(e, cti)
}

func lowerExpr(e core.CoreExpr, cti types.CoreTypeInfo) stmt.Expr {
	switch e := e.(type) {
	case *core.Var:
		return stmt.VarRef{Name: e.Name}

	case *core.VarGlobal:
		return stmt.GlobalRef{Module: e.Ref.Module, Name: e.Ref.Name}

	case *core.Lit:
		return lowerLit(e)

	case *core.Lambda:
		return lowerLambda(e, cti)

	case *core.App:
		return lowerApp(e, cti)

	case *core.BinOp:
		return lowerBinOp(e, cti)

	case *core.UnOp:
		return lowerUnOp(e, cti)

	case *core.Intrinsic:
		return lowerIntrinsic(e, cti)

	case *core.If:
		// If used as an expression (produces a value).
		return stmt.IfExpr{
			Cond: lowerExpr(e.Cond, cti),
			Then: lowerExpr(e.Then, cti),
			Else: lowerExpr(e.Else, cti),
		}

	case *core.Record:
		return lowerRecord(e, cti)

	case *core.RecordAccess:
		return stmt.FieldAccess{
			Record: lowerExpr(e.Record, cti),
			Field:  e.Field,
		}

	case *core.RecordUpdate:
		return lowerRecordUpdate(e, cti)

	case *core.List:
		return lowerList(e, cti)

	case *core.Array:
		return lowerArray(e, cti)

	case *core.Tuple:
		return lowerTuple(e, cti)

	case *core.Let:
		// Let as expression — wrap in an IIFE-like pattern.
		// The block lowerer handles let-chain flattening for statement context.
		// Here we produce the value of the body after binding.
		return lowerLetExpr(e, cti)

	case *core.LetRec:
		return lowerLetRecExpr(e, cti)

	case *core.Match:
		// Match as expression — the match lowerer handles this.
		return LowerMatchExpr(e, cti)

	case *core.DictApp:
		return lowerDictApp(e, cti)

	case *core.DictRef:
		// Dictionary reference — erase to a no-op.
		// DictApp should have already resolved the method call.
		return stmt.LitUnit{}

	case *core.DictAbs:
		// Dictionary abstraction — erase, lower the body.
		return lowerExpr(e.Body, cti)

	case *core.Forall:
		// Forall (M-VERIFY contracts) — erase at codegen.
		return stmt.LitBool{Value: true}

	default:
		// Unknown Core expression type.
		panic(fmt.Sprintf("lower: unhandled Core expression type %T", e))
	}
}

func lowerLit(e *core.Lit) stmt.Expr {
	switch e.Kind {
	case core.IntLit:
		switch v := e.Value.(type) {
		case int64:
			return stmt.LitInt{Value: v}
		case int:
			return stmt.LitInt{Value: int64(v)}
		default:
			return stmt.LitInt{Value: 0}
		}
	case core.FloatLit:
		switch v := e.Value.(type) {
		case float64:
			return stmt.LitFloat{Value: v}
		default:
			return stmt.LitFloat{Value: 0}
		}
	case core.BoolLit:
		if v, ok := e.Value.(bool); ok {
			return stmt.LitBool{Value: v}
		}
		return stmt.LitBool{Value: false}
	case core.StringLit:
		if v, ok := e.Value.(string); ok {
			return stmt.LitString{Value: v}
		}
		return stmt.LitString{Value: ""}
	case core.UnitLit:
		return stmt.LitUnit{}
	default:
		return stmt.LitUnit{}
	}
}

func lowerLambda(e *core.Lambda, cti types.CoreTypeInfo) stmt.Expr {
	params := make([]stmt.Param, len(e.Params))
	for i, name := range e.Params {
		// Try to resolve parameter types from the lambda's type info.
		var paramType stmt.ResolvedType = stmt.InterfaceType{}
		if lambdaType, ok := cti[e.ID()]; ok {
			if ft, ok := lambdaType.(*types.TFunc2); ok && i < len(ft.Params) {
				paramType = ProjectType(ft.Params[i])
			}
		}
		params[i] = stmt.Param{Name: name, Type: paramType}
	}

	// Lower the body. If it's a let-chain, flatten into statements + return expr.
	body, retExpr := FlattenBlock(e.Body, cti)

	return stmt.Lambda{
		Params: params,
		Body:   body,
		Return: retExpr,
	}
}

func lowerApp(e *core.App, cti types.CoreTypeInfo) stmt.Expr {
	// Check for cons operator in both saturated and curried forms.
	// Saturated: App(VarGlobal("::"), [head, tail])
	if vg, ok := e.Func.(*core.VarGlobal); ok && vg.Ref.Name == "::" {
		if len(e.Args) == 2 {
			return stmt.Cons{
				Head: lowerExpr(e.Args[0], cti),
				Tail: lowerExpr(e.Args[1], cti),
			}
		}
	}
	// Curried: App(App(VarGlobal("::"), [head]), [tail])
	if innerApp, ok := e.Func.(*core.App); ok {
		if vg, ok := innerApp.Func.(*core.VarGlobal); ok && vg.Ref.Name == "::" {
			if len(innerApp.Args) == 1 && len(e.Args) == 1 {
				return stmt.Cons{
					Head: lowerExpr(innerApp.Args[0], cti),
					Tail: lowerExpr(e.Args[0], cti),
				}
			}
		}
	}

	args := make([]stmt.Expr, len(e.Args))
	for i, a := range e.Args {
		args[i] = lowerExpr(a, cti)
	}
	return stmt.Call{
		Func: lowerExpr(e.Func, cti),
		Args: args,
	}
}

func lowerBinOp(e *core.BinOp, cti types.CoreTypeInfo) stmt.Expr {
	op := mapBinOpString(e.Op)
	return stmt.BinOp{
		Op:    op,
		Left:  lowerExpr(e.Left, cti),
		Right: lowerExpr(e.Right, cti),
	}
}

func mapBinOpString(op string) stmt.BinOpKind {
	switch op {
	case "+":
		return stmt.OpAdd
	case "-":
		return stmt.OpSub
	case "*":
		return stmt.OpMul
	case "/":
		return stmt.OpDiv
	case "%":
		return stmt.OpMod
	case "==":
		return stmt.OpEq
	case "!=":
		return stmt.OpNeq
	case "<":
		return stmt.OpLt
	case "<=":
		return stmt.OpLte
	case ">":
		return stmt.OpGt
	case ">=":
		return stmt.OpGte
	case "&&":
		return stmt.OpAnd
	case "||":
		return stmt.OpOr
	case "++":
		return stmt.OpConcat
	default:
		return stmt.OpAdd // fallback
	}
}

func lowerUnOp(e *core.UnOp, cti types.CoreTypeInfo) stmt.Expr {
	var op stmt.UnOpKind
	switch e.Op {
	case "-":
		op = stmt.OpNeg
	case "!":
		op = stmt.OpNot
	default:
		op = stmt.OpNeg
	}
	return stmt.UnOp{
		Op:      op,
		Operand: lowerExpr(e.Operand, cti),
	}
}

func lowerIntrinsic(e *core.Intrinsic, cti types.CoreTypeInfo) stmt.Expr {
	args := make([]stmt.Expr, len(e.Args))
	for i, a := range e.Args {
		args[i] = lowerExpr(a, cti)
	}

	op := mapIntrinsicOp(e.Op)

	// Unary intrinsics.
	if len(args) == 1 {
		var uop stmt.UnOpKind
		switch e.Op {
		case core.OpNeg:
			uop = stmt.OpNeg
		case core.OpNot:
			uop = stmt.OpNot
		default:
			uop = stmt.OpNeg
		}
		return stmt.UnOp{Op: uop, Operand: args[0]}
	}

	// Binary intrinsics.
	if len(args) == 2 {
		return stmt.BinOp{Op: op, Left: args[0], Right: args[1]}
	}

	// Shouldn't happen — intrinsics are always unary or binary.
	return stmt.LitUnit{}
}

func mapIntrinsicOp(op core.IntrinsicOp) stmt.BinOpKind {
	switch op {
	case core.OpAdd:
		return stmt.OpAdd
	case core.OpSub:
		return stmt.OpSub
	case core.OpMul:
		return stmt.OpMul
	case core.OpDiv:
		return stmt.OpDiv
	case core.OpMod:
		return stmt.OpMod
	case core.OpEq:
		return stmt.OpEq
	case core.OpNe:
		return stmt.OpNeq
	case core.OpLt:
		return stmt.OpLt
	case core.OpLe:
		return stmt.OpLte
	case core.OpGt:
		return stmt.OpGt
	case core.OpGe:
		return stmt.OpGte
	case core.OpConcat:
		return stmt.OpConcat
	case core.OpAnd:
		return stmt.OpAnd
	case core.OpOr:
		return stmt.OpOr
	default:
		return stmt.OpAdd
	}
}

func lowerRecord(e *core.Record, cti types.CoreTypeInfo) stmt.Expr {
	// Determine record type name from CoreTypeInfo.
	typeName := ""
	if t, ok := cti[e.ID()]; ok {
		if rt, ok := t.(*types.TRecord); ok && rt.TypeName != "" {
			typeName = rt.TypeName
		}
	}

	fields := make([]stmt.FieldInit, 0, len(e.Fields))
	for name, val := range e.Fields {
		fields = append(fields, stmt.FieldInit{
			Name:  name,
			Value: lowerExpr(val, cti),
		})
	}

	return stmt.RecordLit{
		TypeName: typeName,
		Fields:   fields,
	}
}

func lowerRecordUpdate(e *core.RecordUpdate, cti types.CoreTypeInfo) stmt.Expr {
	fields := make([]stmt.FieldInit, 0, len(e.Updates))
	for name, val := range e.Updates {
		fields = append(fields, stmt.FieldInit{
			Name:  name,
			Value: lowerExpr(val, cti),
		})
	}
	return stmt.RecordUpdate{
		Base:   lowerExpr(e.Base, cti),
		Fields: fields,
	}
}

func lowerList(e *core.List, cti types.CoreTypeInfo) stmt.Expr {
	elemType := resolveCollectionElemType(e.ID(), cti)
	elems := make([]stmt.Expr, len(e.Elements))
	for i, el := range e.Elements {
		elems[i] = lowerExpr(el, cti)
	}
	return stmt.ListLit{
		ElemType: elemType,
		Elems:    elems,
	}
}

func lowerArray(e *core.Array, cti types.CoreTypeInfo) stmt.Expr {
	elemType := resolveCollectionElemType(e.ID(), cti)
	elems := make([]stmt.Expr, len(e.Elements))
	for i, el := range e.Elements {
		elems[i] = lowerExpr(el, cti)
	}
	return stmt.ArrayLit{
		ElemType: elemType,
		Elems:    elems,
	}
}

func lowerTuple(e *core.Tuple, cti types.CoreTypeInfo) stmt.Expr {
	elems := make([]stmt.Expr, len(e.Elements))
	for i, el := range e.Elements {
		elems[i] = lowerExpr(el, cti)
	}
	return stmt.TupleLit{Elems: elems}
}

// resolveCollectionElemType extracts the element type from a list/array type.
func resolveCollectionElemType(nodeID uint64, cti types.CoreTypeInfo) stmt.ResolvedType {
	if t, ok := cti[nodeID]; ok {
		switch t := t.(type) {
		case *types.TList:
			return ProjectType(t.Element)
		case *types.TArray:
			return ProjectType(t.Element)
		case *types.TApp:
			if con, ok := t.Constructor.(*types.TCon); ok {
				if (con.Name == "list" || con.Name == "Array") && len(t.Args) == 1 {
					return ProjectType(t.Args[0])
				}
			}
		}
	}
	return stmt.InterfaceType{}
}

// lowerLetExpr handles Let as an expression (not in statement context).
// For statement context, use FlattenBlock instead.
func lowerLetExpr(e *core.Let, cti types.CoreTypeInfo) stmt.Expr {
	// In expression context, we must produce a single Expr.
	// The block lowerer handles this better, but as a fallback
	// we lower the body which should reference the bound variable.
	// The caller (lowerLambda, etc.) should use FlattenBlock instead.
	return lowerExpr(e.Body, cti)
}

func lowerLetRecExpr(e *core.LetRec, cti types.CoreTypeInfo) stmt.Expr {
	return lowerExpr(e.Body, cti)
}

// lowerDictApp handles type class method dispatch.
// DictApp nodes represent calls like `dict.add(x, y)` where `dict` is a
// type class dictionary. We resolve these to concrete operations.
func lowerDictApp(e *core.DictApp, cti types.CoreTypeInfo) stmt.Expr {
	args := make([]stmt.Expr, len(e.Args))
	for i, a := range e.Args {
		args[i] = lowerExpr(a, cti)
	}

	// If the dict is a DictRef, we can resolve the method directly.
	if ref, ok := e.Dict.(*core.DictRef); ok {
		return lowerDictMethod(ref.ClassName, ref.TypeName, e.Method, args)
	}

	// If the dict is a Var (from DictAbs parameter), we can't resolve
	// statically. This happens in polymorphic code that wasn't monomorphized.
	// Fall back to a builtin call.
	return stmt.BuiltinCall{
		Name: fmt.Sprintf("_dict_%s", e.Method),
		Args: args,
	}
}

// lowerDictMethod resolves a known dictionary method to a concrete operation.
func lowerDictMethod(className, typeName, method string, args []stmt.Expr) stmt.Expr {
	// Num methods → arithmetic operators.
	if className == "Num" {
		switch method {
		case "add":
			if len(args) == 2 {
				return stmt.BinOp{Op: stmt.OpAdd, Left: args[0], Right: args[1]}
			}
		case "sub":
			if len(args) == 2 {
				return stmt.BinOp{Op: stmt.OpSub, Left: args[0], Right: args[1]}
			}
		case "mul":
			if len(args) == 2 {
				return stmt.BinOp{Op: stmt.OpMul, Left: args[0], Right: args[1]}
			}
		case "negate":
			if len(args) == 1 {
				return stmt.UnOp{Op: stmt.OpNeg, Operand: args[0]}
			}
		}
	}

	// Eq methods → equality operators.
	if className == "Eq" {
		switch method {
		case "eq":
			if len(args) == 2 {
				return stmt.BinOp{Op: stmt.OpEq, Left: args[0], Right: args[1]}
			}
		case "ne":
			if len(args) == 2 {
				return stmt.BinOp{Op: stmt.OpNeq, Left: args[0], Right: args[1]}
			}
		}
	}

	// Ord methods → comparison operators.
	if className == "Ord" {
		switch method {
		case "lt":
			if len(args) == 2 {
				return stmt.BinOp{Op: stmt.OpLt, Left: args[0], Right: args[1]}
			}
		case "le":
			if len(args) == 2 {
				return stmt.BinOp{Op: stmt.OpLte, Left: args[0], Right: args[1]}
			}
		case "gt":
			if len(args) == 2 {
				return stmt.BinOp{Op: stmt.OpGt, Left: args[0], Right: args[1]}
			}
		case "ge":
			if len(args) == 2 {
				return stmt.BinOp{Op: stmt.OpGte, Left: args[0], Right: args[1]}
			}
		}
	}

	// Show methods → builtin call.
	if className == "Show" && method == "show" {
		return stmt.BuiltinCall{Name: "_show", Args: args}
	}

	// Unresolved — emit as builtin call with descriptive name.
	return stmt.BuiltinCall{
		Name: fmt.Sprintf("_%s_%s_%s", className, typeName, method),
		Args: args,
	}
}
