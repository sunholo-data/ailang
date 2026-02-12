package smt

import (
	"fmt"
	"strings"

	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/types"
)

// FunctionParam describes a function parameter for SMT encoding.
type FunctionParam struct {
	Name string
	Type types.Type
}

// EncodeResult holds the generated SMT-LIB program for a function.
type EncodeResult struct {
	// SMTLib is the complete SMT-LIB program text.
	SMTLib string
	// Declarations are the type/constant declarations.
	Declarations []string
	// Assertions are the assert statements.
	Assertions []string
	// BodyExpr is the encoded function body expression.
	BodyExpr string
}

// EncodeFunction generates a complete SMT-LIB program for verifying a function's contracts.
//
// The encoding follows the standard pattern for bounded verification:
//  1. Declare types (datatypes for ADTs)
//  2. Declare symbolic variables (function parameters)
//  3. Assert preconditions (requires clauses)
//  4. Define function body as "result"
//  5. Assert negation of postconditions (ensures clauses)
//  6. Check satisfiability
//
// If the result is "unsat", all postconditions hold for all inputs satisfying preconditions.
// If "sat", the model provides a counterexample.
func EncodeFunction(
	funcName string,
	params []FunctionParam,
	body core.CoreExpr,
	returnSort string,
	meta *core.DeclMeta,
	adtTypes map[string][]ADTVariant,
) (*EncodeResult, error) {
	ctx := NewSMTContext()
	result := &EncodeResult{}

	// Step 1: Declare ADT types
	for typeName, variants := range adtTypes {
		if !ctx.DeclaredTypes[typeName] {
			decl := DeclareDatatype(typeName, variants)
			result.Declarations = append(result.Declarations, decl)
			ctx.DeclaredTypes[typeName] = true
		}
	}

	// Step 2: Declare symbolic variables for function parameters
	for _, p := range params {
		sort, err := MapType(p.Type)
		if err != nil {
			return nil, fmt.Errorf("cannot encode parameter %q: %w", p.Name, err)
		}
		decl := DeclareConst(p.Name, sort)
		result.Declarations = append(result.Declarations, decl)
		ctx.Variables[p.Name] = sort
	}

	// Step 3: Assert preconditions
	for _, contract := range meta.Contracts {
		if contract.Kind != core.RequiresKind {
			continue
		}
		if contract.Expr == nil {
			continue
		}
		encoded, err := EncodeExpr(contract.Expr)
		if err != nil {
			return nil, fmt.Errorf("cannot encode requires clause: %w", err)
		}
		result.Assertions = append(result.Assertions, Assert(encoded))
	}

	// Step 4: Encode function body and bind to "result"
	bodyExpr, err := EncodeExpr(body)
	if err != nil {
		return nil, fmt.Errorf("cannot encode function body: %w", err)
	}
	result.BodyExpr = bodyExpr

	// Determine result type
	resultSort := returnSort
	if resultSort == "" {
		resultSort = inferResultSort(params, body, ctx, adtTypes)
	}
	resultDecl := fmt.Sprintf("(define-const result %s %s)", resultSort, bodyExpr)
	result.Declarations = append(result.Declarations, resultDecl)

	// Step 5: Assert negation of postconditions (ensures clauses)
	for _, contract := range meta.Contracts {
		if contract.Kind != core.EnsuresKind {
			continue
		}
		if contract.Expr == nil {
			continue
		}
		encoded, err := EncodeExpr(contract.Expr)
		if err != nil {
			return nil, fmt.Errorf("cannot encode ensures clause: %w", err)
		}
		result.Assertions = append(result.Assertions, AssertNot(encoded))
	}

	// Step 6: Build complete SMT-LIB program
	var lines []string
	lines = append(lines, fmt.Sprintf("; Verification of %s", funcName))
	lines = append(lines, "(set-logic ALL)")
	lines = append(lines, "")
	lines = append(lines, result.Declarations...)
	lines = append(lines, "")
	lines = append(lines, result.Assertions...)
	lines = append(lines, "")
	lines = append(lines, CheckSat())
	lines = append(lines, GetModel())

	result.SMTLib = strings.Join(lines, "\n")
	return result, nil
}

// EncodeExpr translates a Core AST expression to an SMT-LIB expression string.
func EncodeExpr(expr core.CoreExpr) (string, error) {
	if expr == nil {
		return "", fmt.Errorf("nil expression")
	}

	switch e := expr.(type) {
	case *core.Lit:
		return encodeLit(e)

	case *core.Var:
		return e.Name, nil

	case *core.VarGlobal:
		// For builtins, return the builtin name (will be used as function in App)
		if e.Ref.Module == "$builtin" {
			return e.Ref.Name, nil
		}
		// For ADT constructors, strip make_Type_ prefix
		return stripConstructorPrefix(e.Ref.Name), nil

	case *core.App:
		return encodeApp(e)

	case *core.If:
		cond, err := EncodeExpr(e.Cond)
		if err != nil {
			return "", fmt.Errorf("if condition: %w", err)
		}
		then, err := EncodeExpr(e.Then)
		if err != nil {
			return "", fmt.Errorf("if then: %w", err)
		}
		els, err := EncodeExpr(e.Else)
		if err != nil {
			return "", fmt.Errorf("if else: %w", err)
		}
		return fmt.Sprintf("(ite %s %s %s)", cond, then, els), nil

	case *core.Let:
		return encodeLet(e)

	case *core.Match:
		return encodeMatch(e)

	case *core.Intrinsic:
		// Pre-lowered intrinsic (shouldn't appear after op_lowering, but handle gracefully)
		return encodeIntrinsic(e)

	case *core.BinOp:
		return encodeBinOp(e)

	case *core.UnOp:
		return encodeUnOp(e)

	case *core.DictApp:
		return encodeDictApp(e)

	case *core.DictAbs:
		// Dictionary abstraction: transparently encode the body
		return EncodeExpr(e.Body)

	case *core.DictRef:
		// Dictionary reference: these are type class instances, skip in SMT
		return "", fmt.Errorf("dictionary reference cannot be encoded directly in SMT-LIB")

	case *core.Lambda:
		return "", fmt.Errorf("lambda expressions cannot be encoded in SMT-LIB (higher-order)")

	case *core.LetRec:
		return "", fmt.Errorf("recursive let bindings cannot be encoded in SMT-LIB")

	case *core.Tuple:
		return "", fmt.Errorf("tuple expressions cannot be encoded in SMT-LIB")

	case *core.Record:
		return "", fmt.Errorf("record expressions cannot be encoded in SMT-LIB")

	case *core.RecordAccess:
		return "", fmt.Errorf("record access cannot be encoded in SMT-LIB")

	case *core.List:
		return "", fmt.Errorf("list expressions cannot be encoded in SMT-LIB")

	default:
		return "", fmt.Errorf("unsupported Core expression type %T", expr)
	}
}

// encodeLit encodes a literal value.
func encodeLit(lit *core.Lit) (string, error) {
	switch lit.Kind {
	case core.IntLit:
		v, ok := lit.Value.(int64)
		if !ok {
			return "", fmt.Errorf("IntLit with non-int64 value: %T", lit.Value)
		}
		if v < 0 {
			return fmt.Sprintf("(- %d)", -v), nil
		}
		return fmt.Sprintf("%d", v), nil
	case core.FloatLit:
		v, ok := lit.Value.(float64)
		if !ok {
			return "", fmt.Errorf("FloatLit with non-float64 value: %T", lit.Value)
		}
		if v < 0 {
			return fmt.Sprintf("(- %g)", -v), nil
		}
		return fmt.Sprintf("%g", v), nil
	case core.BoolLit:
		v, ok := lit.Value.(bool)
		if !ok {
			return "", fmt.Errorf("BoolLit with non-bool value: %T", lit.Value)
		}
		if v {
			return "true", nil
		}
		return "false", nil
	case core.UnitLit:
		return "", fmt.Errorf("unit literals cannot be encoded in SMT-LIB")
	case core.StringLit:
		return "", fmt.Errorf("string literals cannot be encoded in SMT-LIB")
	default:
		return "", fmt.Errorf("unknown literal kind: %d", lit.Kind)
	}
}

// encodeApp handles function application.
// After op_lowering, most operators appear as App(VarGlobal($builtin.XXX), args).
func encodeApp(app *core.App) (string, error) {
	// Check for builtin operator pattern: App(VarGlobal($builtin.XXX), args)
	if vg, ok := app.Func.(*core.VarGlobal); ok && vg.Ref.Module == "$builtin" {
		smtOp, isBuiltin := BuiltinToSMTOp[vg.Ref.Name]
		if isBuiltin {
			return encodeBuiltinOp(smtOp, app.Args)
		}
	}

	// Check for curried builtin: App(App(VarGlobal($builtin.XXX), [arg1]), [arg2])
	if innerApp, ok := app.Func.(*core.App); ok {
		if vg, ok := innerApp.Func.(*core.VarGlobal); ok && vg.Ref.Module == "$builtin" {
			smtOp, isBuiltin := BuiltinToSMTOp[vg.Ref.Name]
			if isBuiltin {
				// Combine args: inner args + outer args
				allArgs := make([]core.CoreExpr, 0, len(innerApp.Args)+len(app.Args))
				allArgs = append(allArgs, innerApp.Args...)
				allArgs = append(allArgs, app.Args...)
				return encodeBuiltinOp(smtOp, allArgs)
			}
		}
	}

	// ADT constructor application: App(VarGlobal(module.Ctor), args)
	if vg, ok := app.Func.(*core.VarGlobal); ok && vg.Ref.Module != "$builtin" {
		name := stripConstructorPrefix(vg.Ref.Name)
		return encodeConstructorApp(name, app.Args)
	}

	// Plain variable application (could be a local constructor reference)
	if v, ok := app.Func.(*core.Var); ok {
		name := stripConstructorPrefix(v.Name)
		if len(app.Args) == 0 {
			return name, nil
		}
		return encodeConstructorApp(name, app.Args)
	}

	return "", fmt.Errorf("unsupported application: %s", app.String())
}

// encodeBuiltinOp encodes a builtin operator application.
func encodeBuiltinOp(smtOp string, args []core.CoreExpr) (string, error) {
	if len(args) == 1 {
		// Unary: (- x) or (not x)
		arg, err := EncodeExpr(args[0])
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("(%s %s)", smtOp, arg), nil
	}
	if len(args) == 2 {
		// Binary: (>= x 0)
		left, err := EncodeExpr(args[0])
		if err != nil {
			return "", err
		}
		right, err := EncodeExpr(args[1])
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("(%s %s %s)", smtOp, left, right), nil
	}
	return "", fmt.Errorf("builtin %q with unexpected arity %d", smtOp, len(args))
}

// encodeConstructorApp encodes an ADT constructor application.
func encodeConstructorApp(ctorName string, args []core.CoreExpr) (string, error) {
	if len(args) == 0 {
		return ctorName, nil
	}
	encodedArgs := make([]string, len(args))
	for i, arg := range args {
		encoded, err := EncodeExpr(arg)
		if err != nil {
			return "", fmt.Errorf("constructor arg %d: %w", i, err)
		}
		encodedArgs[i] = encoded
	}
	return fmt.Sprintf("(%s %s)", ctorName, strings.Join(encodedArgs, " ")), nil
}

// encodeLet encodes a let binding as SMT-LIB let expression.
func encodeLet(let *core.Let) (string, error) {
	value, err := EncodeExpr(let.Value)
	if err != nil {
		return "", fmt.Errorf("let value: %w", err)
	}
	body, err := EncodeExpr(let.Body)
	if err != nil {
		return "", fmt.Errorf("let body: %w", err)
	}
	return fmt.Sprintf("(let ((%s %s)) %s)", let.Name, value, body), nil
}

// encodeMatch encodes a match expression.
// For enum ADTs: (match var ((Variant1 body1) (Variant2 body2)))
// For ADTs with fields: (match var (((Ctor field1 field2) body)))
func encodeMatch(m *core.Match) (string, error) {
	scrutinee, err := EncodeExpr(m.Scrutinee)
	if err != nil {
		return "", fmt.Errorf("match scrutinee: %w", err)
	}

	var arms []string
	for _, arm := range m.Arms {
		pattern, err := encodePattern(arm.Pattern)
		if err != nil {
			return "", fmt.Errorf("match pattern: %w", err)
		}
		body, err := EncodeExpr(arm.Body)
		if err != nil {
			return "", fmt.Errorf("match body: %w", err)
		}
		arms = append(arms, fmt.Sprintf("(%s %s)", pattern, body))
	}

	return fmt.Sprintf("(match %s (%s))", scrutinee, strings.Join(arms, " ")), nil
}

// encodePattern encodes a Core pattern for SMT-LIB match.
func encodePattern(pat core.CorePattern) (string, error) {
	if pat == nil {
		return "", fmt.Errorf("nil pattern")
	}
	switch p := pat.(type) {
	case *core.ConstructorPattern:
		if len(p.Args) == 0 {
			return p.Name, nil
		}
		var argParts []string
		for _, arg := range p.Args {
			encoded, err := encodePattern(arg)
			if err != nil {
				return "", err
			}
			argParts = append(argParts, encoded)
		}
		return fmt.Sprintf("(%s %s)", p.Name, strings.Join(argParts, " ")), nil
	case *core.VarPattern:
		return p.Name, nil
	case *core.WildcardPattern:
		// SMT-LIB uses _ as wildcard in some dialects; use a fresh variable
		return "_", nil
	case *core.LitPattern:
		return fmt.Sprintf("%v", p.Value), nil
	default:
		return "", fmt.Errorf("unsupported pattern type %T in SMT encoding", pat)
	}
}

// encodeIntrinsic handles pre-lowered Intrinsic nodes.
// After op_lowering these should not appear, but handle them for robustness.
func encodeIntrinsic(intr *core.Intrinsic) (string, error) {
	opMap := map[core.IntrinsicOp]string{
		core.OpAdd: "+", core.OpSub: "-", core.OpMul: "*",
		core.OpDiv: "div", core.OpMod: "mod",
		core.OpEq: "=", core.OpNe: "distinct",
		core.OpLt: "<", core.OpLe: "<=", core.OpGt: ">", core.OpGe: ">=",
		core.OpAnd: "and", core.OpOr: "or", core.OpNot: "not",
		core.OpNeg: "-",
	}
	smtOp, ok := opMap[intr.Op]
	if !ok {
		return "", fmt.Errorf("unsupported intrinsic op: %v", intr.Op)
	}
	return encodeBuiltinOp(smtOp, intr.Args)
}

// encodeBinOp handles pre-lowered BinOp nodes.
func encodeBinOp(bop *core.BinOp) (string, error) {
	opMap := map[string]string{
		"+": "+", "-": "-", "*": "*", "/": "div", "%": "mod",
		"==": "=", "!=": "distinct",
		"<": "<", "<=": "<=", ">": ">", ">=": ">=",
		"&&": "and", "||": "or",
	}
	smtOp, ok := opMap[bop.Op]
	if !ok {
		return "", fmt.Errorf("unsupported binary op: %s", bop.Op)
	}
	left, err := EncodeExpr(bop.Left)
	if err != nil {
		return "", err
	}
	right, err := EncodeExpr(bop.Right)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("(%s %s %s)", smtOp, left, right), nil
}

// encodeUnOp handles pre-lowered UnOp nodes.
func encodeUnOp(uop *core.UnOp) (string, error) {
	opMap := map[string]string{
		"not": "not", "-": "-",
	}
	smtOp, ok := opMap[uop.Op]
	if !ok {
		return "", fmt.Errorf("unsupported unary op: %s", uop.Op)
	}
	operand, err := EncodeExpr(uop.Operand)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("(%s %s)", smtOp, operand), nil
}

// encodeDictApp handles DictApp nodes — type class method applications.
// DictApp appears when dictionary passing is used instead of op_lowering.
// It maps type class method names (e.g., "ge", "add") to SMT-LIB operators.
func encodeDictApp(da *core.DictApp) (string, error) {
	// Map type class method names to SMT-LIB operators
	methodToSMT := map[string]string{
		// Num class
		"add": "+", "sub": "-", "mul": "*", "div": "div", "neg": "-",
		// Eq class
		"eq": "=", "neq": "distinct", "ne": "distinct",
		// Ord class
		"lt": "<", "lte": "<=", "le": "<=",
		"gt": ">", "gte": ">=", "ge": ">=",
		// Bool (if accessed via type class)
		"and": "and", "or": "or", "not": "not",
	}

	smtOp, ok := methodToSMT[da.Method]
	if !ok {
		return "", fmt.Errorf("unsupported type class method %q in SMT encoding", da.Method)
	}

	return encodeBuiltinOp(smtOp, da.Args)
}

// stripConstructorPrefix removes the Core "make_TypeName_" prefix from constructor names.
// Core represents ADT constructors as "make_Season_LOW_SEASON" but SMT-LIB uses "LOW_SEASON".
func stripConstructorPrefix(name string) string {
	if strings.HasPrefix(name, "make_") {
		// Format: make_TypeName_ConstructorName
		// Find second underscore to extract constructor name
		rest := name[5:] // Remove "make_"
		if idx := strings.Index(rest, "_"); idx >= 0 {
			return rest[idx+1:]
		}
	}
	return name
}

// inferResultSort tries to determine the result sort from the function parameters/body.
// Falls back to "Int" if unable to determine.
func inferResultSort(params []FunctionParam, body core.CoreExpr, ctx *SMTContext, adtTypes map[string][]ADTVariant) string {
	// Build reverse lookup: constructor name → type name
	ctorToType := make(map[string]string)
	for typeName, variants := range adtTypes {
		for _, v := range variants {
			ctorToType[v.Name] = typeName
		}
	}

	return inferResultSortInner(body, ctx, ctorToType)
}

func inferResultSortInner(body core.CoreExpr, ctx *SMTContext, ctorToType map[string]string) string {
	if body == nil {
		return "Int"
	}
	switch b := body.(type) {
	case *core.Lit:
		switch b.Kind {
		case core.IntLit:
			return "Int"
		case core.FloatLit:
			return "Real"
		case core.BoolLit:
			return "Bool"
		}
	case *core.Var:
		if sort, ok := ctx.Variables[b.Name]; ok {
			return sort
		}
	case *core.VarGlobal:
		// Check if it's a constructor reference
		name := stripConstructorPrefix(b.Ref.Name)
		if typeName, ok := ctorToType[name]; ok {
			return typeName
		}
	case *core.If:
		return inferResultSortInner(b.Then, ctx, ctorToType)
	case *core.Let:
		return inferResultSortInner(b.Body, ctx, ctorToType)
	case *core.Match:
		if len(b.Arms) > 0 {
			return inferResultSortInner(b.Arms[0].Body, ctx, ctorToType)
		}
	case *core.App:
		// Constructor application — check if func is a constructor
		if vg, ok := b.Func.(*core.VarGlobal); ok {
			name := stripConstructorPrefix(vg.Ref.Name)
			if typeName, ok := ctorToType[name]; ok {
				return typeName
			}
		}
	}
	return "Int"
}
