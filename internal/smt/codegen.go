package smt

import (
	"fmt"
	"sort"
	"strings"

	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/types"
)

// FunctionParam describes a function parameter for SMT encoding.
type FunctionParam struct {
	Name string
	Type types.Type
}

// activeResolvedCallees holds the set of callee function names that have been
// resolved as define-fun during the current EncodeFunction call.
// This is a package-level variable set during encoding (sequential per function).
var activeResolvedCallees map[string]bool

// RecordTypeInfo describes a record type for SMT encoding.
type RecordTypeInfo struct {
	SortName   string            // SMT-LIB sort name (e.g., "Point")
	CtorName   string            // Constructor name (e.g., "mk_Point")
	FieldNames []string          // Sorted field names
	FieldSorts map[string]string // Field name → SMT-LIB sort
}

// activeRecordTypes maps record sort names to their type info.
// Populated during EncodeFunction, used by encodeRecord/encodeRecordUpdate.
var activeRecordTypes map[string]*RecordTypeInfo

// activeFieldSetToSort maps a canonical field-set key to sort name,
// enabling record construction lookup by field names.
var activeFieldSetToSort map[string]string

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

// EncodeFunctionOpts holds optional parameters for cross-function call resolution.
type EncodeFunctionOpts struct {
	// Program is the full Core program (needed for resolving cross-function calls).
	Program *core.Program
	// SurfaceParams maps function names to their parameter info.
	SurfaceParams map[string][]FunctionParam
	// SurfaceReturnSorts maps function names to their return sort strings.
	SurfaceReturnSorts map[string]string
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
	opts ...EncodeFunctionOpts,
) (*EncodeResult, error) {
	ctx := NewSMTContext()
	result := &EncodeResult{}

	// Resolve cross-function call dependencies
	var calleeDefs []CalleeDef
	if len(opts) > 0 && opts[0].Program != nil {
		var err error
		calleeDefs, err = ResolveCallees(
			funcName, body, opts[0].Program,
			opts[0].SurfaceParams, opts[0].SurfaceReturnSorts, adtTypes,
		)
		if err != nil {
			return nil, fmt.Errorf("resolving cross-function calls: %w", err)
		}
		// Register resolved callees so encodeApp can recognize them
		for _, def := range calleeDefs {
			ctx.ResolvedCallees[def.Name] = true
		}
	}

	// Set active resolved callees for encodeApp to use
	activeResolvedCallees = ctx.ResolvedCallees
	defer func() { activeResolvedCallees = nil }()

	// Collect and declare record types from function parameters and return type
	activeRecordTypes = make(map[string]*RecordTypeInfo)
	activeFieldSetToSort = make(map[string]string)
	defer func() { activeRecordTypes = nil; activeFieldSetToSort = nil }()

	collectAndDeclareRecordTypes(params, returnSort, ctx, result)

	// Step 1: Declare ADT types
	for typeName, variants := range adtTypes {
		if !ctx.DeclaredTypes[typeName] {
			decl := DeclareDatatype(typeName, variants)
			result.Declarations = append(result.Declarations, decl)
			ctx.DeclaredTypes[typeName] = true
		}
	}

	// Step 1.5: Add callee define-fun declarations (for cross-function calls)
	for _, def := range calleeDefs {
		result.Declarations = append(result.Declarations, def.SMTLib)
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
		return encodeRecord(e)

	case *core.RecordAccess:
		return encodeRecordAccess(e)

	case *core.RecordUpdate:
		return encodeRecordUpdate(e)

	case *core.List:
		return encodeList(e)

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
		v, ok := lit.Value.(string)
		if !ok {
			return "", fmt.Errorf("StringLit with non-string value: %T", lit.Value)
		}
		// SMT-LIB string literals are enclosed in double quotes with escaping
		return encodeStringLiteral(v), nil
	default:
		return "", fmt.Errorf("unknown literal kind: %d", lit.Kind)
	}
}

// encodeStringLiteral converts a Go string to an SMT-LIB string literal.
// SMT-LIB strings use "" for quotes and standard escapes.
func encodeStringLiteral(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, ch := range s {
		switch ch {
		case '"':
			b.WriteString(`""`) // SMT-LIB escapes " as ""
		case '\\':
			b.WriteString(`\\`)
		default:
			b.WriteRune(ch)
		}
	}
	b.WriteByte('"')
	return b.String()
}

// encodeApp handles function application.
// After op_lowering, most operators appear as App(VarGlobal($builtin.XXX), args).
func encodeApp(app *core.App) (string, error) {
	// Check for builtin operator pattern: App(VarGlobal($builtin.XXX), args)
	if vg, ok := app.Func.(*core.VarGlobal); ok && vg.Ref.Module == "$builtin" {
		// Standard builtins (direct op mapping)
		smtOp, isBuiltin := BuiltinToSMTOp[vg.Ref.Name]
		if isBuiltin {
			return encodeBuiltinOp(smtOp, app.Args)
		}
		// String builtins with special encoding
		if spec, ok := StringBuiltinSpecial[vg.Ref.Name]; ok {
			return encodeStringBuiltin(spec, app.Args)
		}
		// List builtins with special encoding
		if spec, ok := ListBuiltinSpecial[vg.Ref.Name]; ok {
			return encodeListBuiltin(spec, app.Args)
		}
	}

	// Check for curried builtin: App(App(VarGlobal($builtin.XXX), [arg1]), [arg2])
	if innerApp, ok := app.Func.(*core.App); ok {
		if vg, ok := innerApp.Func.(*core.VarGlobal); ok && vg.Ref.Module == "$builtin" {
			// Combine args: inner args + outer args
			allArgs := make([]core.CoreExpr, 0, len(innerApp.Args)+len(app.Args))
			allArgs = append(allArgs, innerApp.Args...)
			allArgs = append(allArgs, app.Args...)

			smtOp, isBuiltin := BuiltinToSMTOp[vg.Ref.Name]
			if isBuiltin {
				return encodeBuiltinOp(smtOp, allArgs)
			}
			// String builtins with special encoding (curried)
			if spec, ok := StringBuiltinSpecial[vg.Ref.Name]; ok {
				return encodeStringBuiltin(spec, allArgs)
			}
			// List builtins with special encoding (curried)
			if spec, ok := ListBuiltinSpecial[vg.Ref.Name]; ok {
				return encodeListBuiltin(spec, allArgs)
			}
		}
	}

	// Check for std/list builtins (:: and concat_List registered under std/list module)
	if vg, ok := app.Func.(*core.VarGlobal); ok && vg.Ref.Module == "std/list" {
		if spec, ok := ListBuiltinSpecial[vg.Ref.Name]; ok {
			return encodeListBuiltin(spec, app.Args)
		}
	}

	// Cross-function call: App(VarGlobal(module.funcName), args)
	// where funcName has been resolved as a define-fun callee
	if vg, ok := app.Func.(*core.VarGlobal); ok && vg.Ref.Module != "$builtin" && vg.Ref.Module != "std/list" {
		if activeResolvedCallees != nil && activeResolvedCallees[vg.Ref.Name] {
			return encodeUserFunctionCall(vg.Ref.Name, app.Args)
		}
		// ADT constructor application
		name := stripConstructorPrefix(vg.Ref.Name)
		return encodeConstructorApp(name, app.Args)
	}

	// Plain variable application — check if it's a resolved callee (same-module function call)
	if v, ok := app.Func.(*core.Var); ok {
		if activeResolvedCallees != nil && activeResolvedCallees[v.Name] {
			return encodeUserFunctionCall(v.Name, app.Args)
		}
		// Otherwise treat as constructor reference
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

// encodeStringBuiltin encodes a string builtin with special handling.
func encodeStringBuiltin(spec StringBuiltinSpec, args []core.CoreExpr) (string, error) {
	if spec.Unary {
		if len(args) != 1 {
			return "", fmt.Errorf("string builtin %q expects 1 arg, got %d", spec.Op, len(args))
		}
		arg, err := EncodeExpr(args[0])
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("(%s %s)", spec.Op, arg), nil
	}

	// SubstrMode: _str_slice(s, start, end) → (str.substr s start (- end start))
	if spec.SubstrMode {
		if len(args) != 3 {
			return "", fmt.Errorf("string builtin %q expects 3 args, got %d", spec.Op, len(args))
		}
		s, err := EncodeExpr(args[0])
		if err != nil {
			return "", err
		}
		start, err := EncodeExpr(args[1])
		if err != nil {
			return "", err
		}
		end, err := EncodeExpr(args[2])
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("(%s %s %s (- %s %s))", spec.Op, s, start, end, start), nil
	}

	if len(args) < 2 {
		return "", fmt.Errorf("string builtin %q expects at least 2 args, got %d", spec.Op, len(args))
	}

	left, err := EncodeExpr(args[0])
	if err != nil {
		return "", err
	}
	right, err := EncodeExpr(args[1])
	if err != nil {
		return "", err
	}

	if spec.FlipArgs {
		left, right = right, left
	}

	if spec.AppendZero {
		return fmt.Sprintf("(%s %s %s 0)", spec.Op, left, right), nil
	}

	return fmt.Sprintf("(%s %s %s)", spec.Op, left, right), nil
}

// encodeUserFunctionCall encodes a call to a user-defined function
// that has been resolved as a define-fun in the SMT-LIB context.
func encodeUserFunctionCall(funcName string, args []core.CoreExpr) (string, error) {
	if len(args) == 0 {
		return fmt.Sprintf("(%s)", funcName), nil
	}
	encodedArgs := make([]string, len(args))
	for i, arg := range args {
		encoded, err := EncodeExpr(arg)
		if err != nil {
			return "", fmt.Errorf("function call %s arg %d: %w", funcName, i, err)
		}
		encodedArgs[i] = encoded
	}
	return fmt.Sprintf("(%s %s)", funcName, strings.Join(encodedArgs, " ")), nil
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
		core.OpNeg:    "-",
		core.OpConcat: "str.++",
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
		case core.StringLit:
			return "String"
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
	case *core.Record:
		// Record construction returns the record sort
		if info := lookupRecordByFields(b.Fields); info != nil {
			return info.SortName
		}
	case *core.RecordAccess:
		// Field access on a record — need to look at the record's type
		// and the field's sort
		return "Int" // conservative fallback
	case *core.List:
		// List expressions return a Seq sort
		// Try to determine element sort from first element
		if len(b.Elements) > 0 {
			elemSort := inferResultSortInner(b.Elements[0], ctx, ctorToType)
			return fmt.Sprintf("(Seq %s)", elemSort)
		}
		return "(Seq Int)" // default
	}
	return "Int"
}

// --- Record encoding ---

// collectAndDeclareRecordTypes collects record types from function parameters
// and populates activeRecordTypes/activeFieldSetToSort for use during encoding.
func collectAndDeclareRecordTypes(params []FunctionParam, returnSort string, ctx *SMTContext, result *EncodeResult) {
	for _, p := range params {
		collectRecordType(p.Type, ctx, result)
	}
}

// collectRecordType recursively extracts record types from an AILANG type
// and emits declare-datatype declarations.
func collectRecordType(t types.Type, ctx *SMTContext, result *EncodeResult) {
	if t == nil {
		return
	}
	rec, ok := t.(*types.TRecord)
	if !ok {
		return
	}

	sortName := MapRecordSortName(rec)
	if ctx.DeclaredTypes[sortName] {
		return // already declared
	}

	// Map all field types (may recursively discover nested record types)
	fieldSorts, err := MapRecordFields(rec)
	if err != nil {
		return // skip records with unencodable field types
	}

	// Recursively collect nested record types first
	for _, fieldType := range rec.Fields {
		collectRecordType(fieldType, ctx, result)
	}

	// Build record type info
	fieldNames := SortedFieldNamesStr(fieldSorts)
	info := &RecordTypeInfo{
		SortName:   sortName,
		CtorName:   RecordConstructorName(sortName),
		FieldNames: fieldNames,
		FieldSorts: fieldSorts,
	}
	activeRecordTypes[sortName] = info

	// Build field-set key for lookup during encoding
	key := strings.Join(fieldNames, ",")
	activeFieldSetToSort[key] = sortName

	// Emit declaration
	decl := DeclareRecordDatatype(sortName, fieldSorts)
	result.Declarations = append(result.Declarations, decl)
	ctx.DeclaredTypes[sortName] = true
}

// encodeRecord encodes a record construction expression.
// Record{Fields: {x: 5, y: 10}} → (mk_Point 5 10)
func encodeRecord(rec *core.Record) (string, error) {
	info := lookupRecordByFields(rec.Fields)
	if info == nil {
		return "", fmt.Errorf("record construction: unknown record type with fields %v (not declared in function signature)", fieldNamesFromExprMap(rec.Fields))
	}

	// Encode field values in sorted order
	var args []string
	for _, fieldName := range info.FieldNames {
		fieldExpr, ok := rec.Fields[fieldName]
		if !ok {
			return "", fmt.Errorf("record construction: missing field %q", fieldName)
		}
		encoded, err := EncodeExpr(fieldExpr)
		if err != nil {
			return "", fmt.Errorf("record field %q: %w", fieldName, err)
		}
		args = append(args, encoded)
	}

	return fmt.Sprintf("(%s %s)", info.CtorName, strings.Join(args, " ")), nil
}

// encodeRecordAccess encodes a record field access.
// RecordAccess{Record: p, Field: "x"} → (x p)
func encodeRecordAccess(ra *core.RecordAccess) (string, error) {
	record, err := EncodeExpr(ra.Record)
	if err != nil {
		return "", fmt.Errorf("record access: %w", err)
	}
	return fmt.Sprintf("(%s %s)", ra.Field, record), nil
}

// encodeRecordUpdate encodes a functional record update.
// RecordUpdate{Base: p, Updates: {x: 20}} → (mk_Point 20 (y p))
func encodeRecordUpdate(ru *core.RecordUpdate) (string, error) {
	info := lookupRecordByFields(ru.Updates)
	if info == nil {
		// Try to find by looking at base expression type (if it's a known variable)
		info = lookupRecordForUpdate(ru)
	}
	if info == nil {
		return "", fmt.Errorf("record update: unknown record type")
	}

	base, err := EncodeExpr(ru.Base)
	if err != nil {
		return "", fmt.Errorf("record update base: %w", err)
	}

	// Build constructor args: updated fields use new values, others use accessor on base
	var args []string
	for _, fieldName := range info.FieldNames {
		if updateExpr, ok := ru.Updates[fieldName]; ok {
			encoded, err := EncodeExpr(updateExpr)
			if err != nil {
				return "", fmt.Errorf("record update field %q: %w", fieldName, err)
			}
			args = append(args, encoded)
		} else {
			// Use accessor on base: (fieldName base)
			args = append(args, fmt.Sprintf("(%s %s)", fieldName, base))
		}
	}

	return fmt.Sprintf("(%s %s)", info.CtorName, strings.Join(args, " ")), nil
}

// lookupRecordByFields finds a record type info that contains ALL the given field names.
// For construction, the fields must match exactly; for updates, they must be a subset.
func lookupRecordByFields(fields interface{}) *RecordTypeInfo {
	if activeRecordTypes == nil {
		return nil
	}

	var names []string
	switch f := fields.(type) {
	case map[string]core.CoreExpr:
		for name := range f {
			names = append(names, name)
		}
	case map[string]string:
		for name := range f {
			names = append(names, name)
		}
	default:
		return nil
	}
	sort.Strings(names)
	key := strings.Join(names, ",")

	if sortName, ok := activeFieldSetToSort[key]; ok {
		return activeRecordTypes[sortName]
	}
	return nil
}

// lookupRecordForUpdate finds the record type for an update expression.
// Tries all known record types and checks if the update fields are a subset.
func lookupRecordForUpdate(ru *core.RecordUpdate) *RecordTypeInfo {
	if activeRecordTypes == nil {
		return nil
	}
	updateFields := make(map[string]bool, len(ru.Updates))
	for name := range ru.Updates {
		updateFields[name] = true
	}
	for _, info := range activeRecordTypes {
		// Check if all update fields exist in this record type
		allPresent := true
		for name := range updateFields {
			if _, ok := info.FieldSorts[name]; !ok {
				allPresent = false
				break
			}
		}
		if allPresent {
			return info
		}
	}
	return nil
}

// fieldNamesFromExprMap extracts sorted field names from a map of expressions.
func fieldNamesFromExprMap(fields map[string]core.CoreExpr) []string {
	names := make([]string, 0, len(fields))
	for name := range fields {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// --- List encoding ---

// encodeList encodes a list literal to SMT-LIB using Z3 sequence theory.
// [1, 2, 3] → (seq.++ (seq.unit 1) (seq.++ (seq.unit 2) (seq.unit 3)))
// [] → (as seq.empty (Seq Int))  (needs type from context)
func encodeList(list *core.List) (string, error) {
	if len(list.Elements) == 0 {
		// Empty list — need an element sort for the typed empty sequence.
		// Default to Int; the SMT solver will unify sorts as needed.
		return "(as seq.empty (Seq Int))", nil
	}

	// Single element: (seq.unit elem)
	if len(list.Elements) == 1 {
		elem, err := EncodeExpr(list.Elements[0])
		if err != nil {
			return "", fmt.Errorf("list element 0: %w", err)
		}
		return fmt.Sprintf("(seq.unit %s)", elem), nil
	}

	// Multiple elements: chain of (seq.++ (seq.unit e1) (seq.++ (seq.unit e2) ...))
	// Build right-to-left
	encoded := make([]string, len(list.Elements))
	for i, elem := range list.Elements {
		e, err := EncodeExpr(elem)
		if err != nil {
			return "", fmt.Errorf("list element %d: %w", i, err)
		}
		encoded[i] = fmt.Sprintf("(seq.unit %s)", e)
	}

	// Z3's seq.++ is variadic, so we can pass all at once
	return fmt.Sprintf("(seq.++ %s)", strings.Join(encoded, " ")), nil
}

// encodeListBuiltin encodes a list builtin with special handling.
func encodeListBuiltin(spec ListBuiltinSpec, args []core.CoreExpr) (string, error) {
	if spec.ConsMode {
		// :: (cons): (seq.++ (seq.unit head) tail)
		if len(args) != 2 {
			return "", fmt.Errorf("cons (::) expects 2 args, got %d", len(args))
		}
		head, err := EncodeExpr(args[0])
		if err != nil {
			return "", err
		}
		tail, err := EncodeExpr(args[1])
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("(seq.++ (seq.unit %s) %s)", head, tail), nil
	}

	if spec.Unary {
		if len(args) != 1 {
			return "", fmt.Errorf("list builtin %q expects 1 arg, got %d", spec.Op, len(args))
		}
		arg, err := EncodeExpr(args[0])
		if err != nil {
			return "", err
		}
		if spec.AppendZero {
			// _list_head: (seq.nth xs 0)
			return fmt.Sprintf("(%s %s 0)", spec.Op, arg), nil
		}
		return fmt.Sprintf("(%s %s)", spec.Op, arg), nil
	}

	// Binary: concat_List(xs, ys) → (seq.++ xs ys), _list_nth(xs, i) → (seq.nth xs i)
	if len(args) < 2 {
		return "", fmt.Errorf("list builtin %q expects at least 2 args, got %d", spec.Op, len(args))
	}

	left, err := EncodeExpr(args[0])
	if err != nil {
		return "", err
	}
	right, err := EncodeExpr(args[1])
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("(%s %s %s)", spec.Op, left, right), nil
}
