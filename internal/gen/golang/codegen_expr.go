// Package golang provides Go code generation from AILANG Core AST.
package golang

import (
	"fmt"
	"strings"

	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/types"
)

// generateExpr generates Go code for a Core expression.
func (g *Generator) generateExpr(expr core.CoreExpr) error {
	switch e := expr.(type) {
	case *core.Lit:
		return g.generateLit(e)

	case *core.Var:
		// Check if this is a reference to a top-level function
		if goName, ok := g.topLevelFuncs[e.Name]; ok {
			// M-DX26: In _impl functions, call other _impl functions
			if g.expectedReturnType == "interface{}" {
				g.write(ToGoVarName(e.Name) + "_impl")
			} else {
				g.write(goName)
			}
		} else {
			g.write(ToGoVarName(e.Name))
		}
		return nil

	case *core.VarGlobal:
		// Check if this is an ADT factory call (from elaboration)
		// Elaborator generates: $adt.make_TypeName_CtorName
		if e.Ref.Module == "$adt" && strings.HasPrefix(e.Ref.Name, "make_") {
			// Parse "make_TypeName_CtorName" to get the constructor info
			parts := strings.SplitN(e.Ref.Name[5:], "_", 2) // Skip "make_"
			if len(parts) == 2 {
				typeName := parts[0]
				ctorName := parts[1]
				// Generate Go constructor name using proper naming to avoid double-prefix
				goFuncName := "New" + ToVariantStructName(typeName, ctorName)

				// Check if this is a nullary constructor (needs to be called immediately)
				if ctorInfo, ok := g.adtConstructors[ctorName]; ok && ctorInfo.FieldCount == 0 {
					g.write(goFuncName + "()")
				} else {
					g.write(goFuncName)
				}
				return nil
			}
		}

		// Check if this is a registered ADT constructor by name
		if ctorInfo, ok := g.adtConstructors[e.Ref.Name]; ok {
			// Generate the proper constructor function call
			if ctorInfo.FieldCount == 0 {
				// Nullary constructor: call with no args
				g.write(ctorInfo.GoFuncName + "()")
			} else {
				// Constructor with fields - just reference the function
				// (it will be called with App)
				g.write(ctorInfo.GoFuncName)
			}
			return nil
		}

		// Check if this is an effect builtin that needs handler qualification
		if handlerCall := mapEffectBuiltinToHandler(e.Ref.Name); handlerCall != "" {
			g.write(handlerCall)
			return nil
		}

		// For other global references, use PascalCase
		g.write(ToPascalCase(e.Ref.Name))
		return nil

	case *core.Lambda:
		return g.generateLambda(e)

	case *core.App:
		return g.generateApp(e)

	case *core.Let:
		return g.generateLet(e)

	case *core.LetRec:
		return g.generateLetRec(e)

	case *core.If:
		return g.generateIf(e)

	case *core.Match:
		return g.generateMatch(e)

	case *core.BinOp:
		return g.generateBinOp(e)

	case *core.UnOp:
		return g.generateUnOp(e)

	case *core.Record:
		return g.generateRecord(e)

	case *core.RecordAccess:
		return g.generateRecordAccess(e)

	case *core.RecordUpdate:
		return g.generateRecordUpdate(e)

	case *core.List:
		return g.generateList(e)

	case *core.Array:
		return g.generateArray(e)

	case *core.Tuple:
		return g.generateTuple(e)

	case *core.Intrinsic:
		return g.generateIntrinsic(e)

	case *core.DictRef:
		// Dictionary references are runtime-resolved
		g.writef("dict_%s_%s", e.ClassName, e.TypeName)
		return nil

	case *core.DictApp:
		return g.generateDictApp(e)

	default:
		return fmt.Errorf("unsupported expression type: %T", expr)
	}
}

// generateLit generates a Go literal.
// M-DX17: Wrap numeric literals in explicit type conversions for interface{} compatibility.
func (g *Generator) generateLit(lit *core.Lit) error {
	switch lit.Kind {
	case core.IntLit:
		// M-DX17: Wrap in int64() for consistent interface{} type assertions
		if v, ok := lit.Value.(int64); ok {
			g.writef("int64(%d)", v)
		} else if v, ok := lit.Value.(int); ok {
			g.writef("int64(%d)", v)
		} else {
			g.writef("int64(%v)", lit.Value)
		}
	case core.FloatLit:
		// M-DX17: Wrap in float64() for consistent interface{} type assertions
		g.writef("float64(%v)", lit.Value)
	case core.BoolLit:
		g.writef("%v", lit.Value)
	case core.StringLit:
		g.writef("%q", lit.Value)
	case core.UnitLit:
		g.write("struct{}{}")
	default:
		return fmt.Errorf("unsupported literal kind: %v", lit.Kind)
	}
	return nil
}

// generateLambda generates a Go anonymous function.
func (g *Generator) generateLambda(lam *core.Lambda) error {
	var params []string
	for _, p := range lam.Params {
		params = append(params, fmt.Sprintf("%s interface{}", ToGoVarName(p)))
	}

	g.writef("func(%s) interface{} {\n", strings.Join(params, ", "))
	g.indent++
	g.writef("return ")
	if err := g.generateExpr(lam.Body); err != nil {
		return err
	}
	g.writef("\n")
	g.indent--
	g.write("}")

	return nil
}

// generateApp generates a Go function application.
func (g *Generator) generateApp(app *core.App) error {
	// M-DX24.2: Check if this is an arithmetic helper that can be emitted as native operator
	if g.canEmitNativeOp(app) {
		return g.generateNativeOp(app)
	}

	// Special handling for cons operator (::)
	if v, ok := app.Func.(*core.Var); ok && v.Name == "::" {
		g.write("Cons(")
		for i, arg := range app.Args {
			if i > 0 {
				g.write(", ")
			}
			if err := g.generateExpr(arg); err != nil {
				return err
			}
		}
		g.write(")")
		return nil
	}
	// Also check for VarGlobal with :: name
	if v, ok := app.Func.(*core.VarGlobal); ok && v.Ref.Name == "::" {
		g.write("Cons(")
		for i, arg := range app.Args {
			if i > 0 {
				g.write(", ")
			}
			if err := g.generateExpr(arg); err != nil {
				return err
			}
		}
		g.write(")")
		return nil
	}

	// Check if this is an ADT constructor call that needs type assertions
	if ctorInfo := g.getADTConstructorForApp(app); ctorInfo != nil && len(ctorInfo.FieldTypes) > 0 {
		// Generate ADT constructor call with type assertions/conversions
		g.write(ctorInfo.GoFuncName + "(")
		for i, arg := range app.Args {
			if i > 0 {
				g.write(", ")
			}
			// Add type assertion/conversion if we have type info for this field
			if i < len(ctorInfo.FieldTypes) {
				goType := ctorInfo.FieldTypes[i]
				if goType != "interface{}" {
					// Check if this is a slice type - needs conversion helper
					if sliceConv := g.getSliceConversion(goType); sliceConv != "" {
						// Slice types need conversion (Go slices are invariant)
						g.writef("%s(", sliceConv)
						if err := g.generateExpr(arg); err != nil {
							return err
						}
						g.write(")")
					} else if lit, isLit := arg.(*core.Lit); isLit {
						// Literals need type conversion, not assertion
						g.writef("%s(", goType)
						if err := g.generateExpr(lit); err != nil {
							return err
						}
						g.write(")")
					} else {
						// Interface values need type assertion
						if err := g.generateExpr(arg); err != nil {
							return err
						}
						g.writef(".(%s)", goType)
					}
				} else {
					if err := g.generateExpr(arg); err != nil {
						return err
					}
				}
			} else {
				if err := g.generateExpr(arg); err != nil {
					return err
				}
			}
		}
		g.write(")")
		return nil
	}

	// Check if function is a variable that needs type assertion
	needsAssertion := false
	if v, ok := app.Func.(*core.Var); ok {
		// Check if it's NOT a known top-level function
		if _, isTopLevel := g.topLevelFuncs[v.Name]; !isTopLevel {
			needsAssertion = true
		}
	}

	if needsAssertion {
		// Lambda stored in variable - needs type assertion
		g.write("CallFunc(")
		if err := g.generateExpr(app.Func); err != nil {
			return err
		}
		for _, arg := range app.Args {
			g.write(", ")
			if err := g.generateExpr(arg); err != nil {
				return err
			}
		}
		g.write(")")
	} else {
		// M-DX26: In _impl functions, don't add type assertions - all params are interface{}
		inImplFunc := g.expectedReturnType == "interface{}"

		// M-DX25.9: Get function's parameter types for call site type assertions
		// M-DX26: Skip in _impl functions since we call _impl versions with interface{} params
		var paramTypes []string
		if !inImplFunc {
			paramTypes = g.getFuncParamTypes(app.Func)
		}

		if err := g.generateExpr(app.Func); err != nil {
			return err
		}
		g.write("(")
		for i, arg := range app.Args {
			if i > 0 {
				g.write(", ")
			}
			// M-DX25.9: Add type assertion if arg is interface{} but param expects concrete
			// M-DX26: Skip in _impl functions
			if !inImplFunc && i < len(paramTypes) && paramTypes[i] != "" && paramTypes[i] != "interface{}" {
				if g.exprProducesInterface(arg) {
					if err := g.generateExpr(arg); err != nil {
						return err
					}
					g.writef(".(%s)", paramTypes[i])
				} else if lit, isLit := arg.(*core.Lit); isLit && isPrimitiveGoType(paramTypes[i]) {
					// Literals need type conversion
					g.writef("%s(", paramTypes[i])
					if err := g.generateExpr(lit); err != nil {
						return err
					}
					g.write(")")
				} else {
					if err := g.generateExpr(arg); err != nil {
						return err
					}
				}
			} else {
				if err := g.generateExpr(arg); err != nil {
					return err
				}
			}
		}
		g.write(")")
	}
	return nil
}

// getADTConstructorForApp checks if an App is calling an ADT constructor and returns its info.
func (g *Generator) getADTConstructorForApp(app *core.App) *ADTConstructorInfo {
	// Check for $adt.make_TypeName_CtorName pattern
	if v, ok := app.Func.(*core.VarGlobal); ok {
		if v.Ref.Module == "$adt" && strings.HasPrefix(v.Ref.Name, "make_") {
			parts := strings.SplitN(v.Ref.Name[5:], "_", 2) // Skip "make_"
			if len(parts) == 2 {
				ctorName := parts[1]
				if info, ok := g.adtConstructors[ctorName]; ok {
					return info
				}
			}
		}
		// Also check direct constructor name
		if info, ok := g.adtConstructors[v.Ref.Name]; ok {
			return info
		}
	}
	return nil
}

// getFuncParamTypes returns the Go parameter types for a function expression.
// M-DX25.9: Used to add type assertions at call sites when args are interface{}.
func (g *Generator) getFuncParamTypes(funcExpr core.CoreExpr) []string {
	// M-DX25.10: For VarGlobal referencing top-level functions, look up stored param types
	if v, ok := funcExpr.(*core.VarGlobal); ok {
		if paramTypes, found := g.funcParamTypes[v.Ref.Name]; found {
			return paramTypes
		}
	}

	// Also try Var (local function reference)
	if v, ok := funcExpr.(*core.Var); ok {
		if paramTypes, found := g.funcParamTypes[v.Name]; found {
			return paramTypes
		}
	}

	// Fallback: look up from CoreTypeInfo
	if g.coreTypeInfo == nil {
		return nil
	}

	// Get the function expression's NodeID and look up its type
	nodeID := g.getExprNodeID(funcExpr)
	if nodeID == 0 {
		return nil
	}

	typ, ok := g.coreTypeInfo[nodeID]
	if !ok {
		return nil
	}

	// Extract parameter types from function type
	return g.extractParamTypes(typ)
}

// extractParamTypes extracts Go parameter types from an AILANG type.
func (g *Generator) extractParamTypes(typ types.Type) []string {
	switch t := typ.(type) {
	case *types.TFunc:
		var params []string
		for _, p := range t.Params {
			if goType, err := g.TypeMapper.MapType(p); err == nil {
				params = append(params, string(goType))
			} else {
				params = append(params, "interface{}")
			}
		}
		return params
	case *types.TFunc2:
		var params []string
		for _, p := range t.Params {
			if goType, err := g.TypeMapper.MapType(p); err == nil {
				params = append(params, string(goType))
			} else {
				params = append(params, "interface{}")
			}
		}
		return params
	default:
		return nil
	}
}

// generateLet generates a Go variable binding.
// M-DX25.2: Use SEPARATE types for variable (value's type) and IIFE return (body's type).
// Bug fix: Originally used let's type for both, but let's type IS body's type, not value's type.
func (g *Generator) generateLet(let *core.Let) error {
	// M-DX26: In _impl functions, everything is interface{}
	inImplFunc := g.expectedReturnType == "interface{}"

	// M-DX25.2 FIX: Variable type comes from VALUE expression, not the let expression
	varType := "interface{}"

	// M-DX26: Skip type inference in _impl functions
	if !inImplFunc {
		// M-DX25.10: Special case for Record expressions - infer type from fields
		// TypeMapper returns "struct{}" for TRecord, but we can get the proper
		// struct name by looking up the record type from its fields.
		if rec, isRec := let.Value.(*core.Record); isRec {
			fieldNames := make(map[string]bool, len(rec.Fields))
			for name := range rec.Fields {
				fieldNames[name] = true
			}
			if recordType := g.GetRecordTypeByFields(fieldNames); recordType != nil {
				varType = "*" + recordType.Name // Records are generated as pointers
			}
		} else if g.coreTypeInfo != nil {
			valueNodeID := g.getExprNodeID(let.Value)
			if valueNodeID != 0 {
				if typ, ok := g.coreTypeInfo[valueNodeID]; ok {
					if goType, err := g.TypeMapper.MapType(typ); err == nil {
						varType = string(goType)
					}
				}
			}
		}
	}

	// M-DX25.2 FIX: Return type comes from LET expression (= body's type)
	returnType := "interface{}"
	// M-DX26: Skip type inference in _impl functions
	if !inImplFunc {
		// M-DX25.10: Special case for Record body - infer type from fields
		if rec, isRec := let.Body.(*core.Record); isRec {
			fieldNames := make(map[string]bool, len(rec.Fields))
			for name := range rec.Fields {
				fieldNames[name] = true
			}
			if recordType := g.GetRecordTypeByFields(fieldNames); recordType != nil {
				returnType = "*" + recordType.Name
			}
		} else if g.coreTypeInfo != nil {
			if typ, ok := g.coreTypeInfo[let.NodeID]; ok {
				if goType, err := g.TypeMapper.MapType(typ); err == nil {
					returnType = string(goType)
				}
			}
		}
	}

	g.writef("func() %s {\n", returnType)
	g.indent++
	g.writef("var %s %s = ", ToGoVarName(let.Name), varType)

	// Add type assertion if value produces interface{} but we need concrete varType
	needsValueAssertion := varType != "interface{}" && g.exprProducesInterface(let.Value)
	if err := g.generateExpr(let.Value); err != nil {
		return err
	}
	if needsValueAssertion {
		g.writef(".(%s)", varType)
	}
	g.writef("\n")

	g.writef("_ = %s // suppress unused\n", ToGoVarName(let.Name))
	g.writef("return ")

	// Add type assertion if body produces interface{} but we need concrete returnType
	// M-DX25.10: Special case - if body is just the variable we declared, we know its type
	needsBodyAssertion := false
	if v, isVar := let.Body.(*core.Var); isVar && v.Name == let.Name {
		// Body is just the variable we declared - its Go type is varType
		needsBodyAssertion = returnType != "interface{}" && varType == "interface{}"
	} else {
		needsBodyAssertion = returnType != "interface{}" && g.exprProducesInterface(let.Body)
	}
	if err := g.generateExpr(let.Body); err != nil {
		return err
	}
	if needsBodyAssertion {
		g.writef(".(%s)", returnType)
	}
	g.writef("\n")

	g.indent--
	g.write("}()")
	return nil
}

// getExprNodeID extracts the NodeID from a CoreExpr.
// M-DX25.2: Used to look up value expression's type separately from let's type.
func (g *Generator) getExprNodeID(expr core.CoreExpr) uint64 {
	if expr == nil {
		return 0
	}
	return expr.ID()
}

// generateLetRec generates recursive function bindings.
func (g *Generator) generateLetRec(letrec *core.LetRec) error {
	g.writef("func() interface{} {\n")
	g.indent++

	// Declare all bindings first
	for _, bind := range letrec.Bindings {
		g.writef("var %s func(...interface{}) interface{}\n", ToGoVarName(bind.Name))
	}

	// Assign values
	for _, bind := range letrec.Bindings {
		g.writef("%s = func(args ...interface{}) interface{} {\n", ToGoVarName(bind.Name))
		g.indent++
		g.writef("return ")
		if err := g.generateExpr(bind.Value); err != nil {
			return err
		}
		g.writef("\n")
		g.indent--
		g.writef("}\n")
	}

	g.writef("return ")
	if err := g.generateExpr(letrec.Body); err != nil {
		return err
	}
	g.writef("\n")
	g.indent--
	g.write("}()")
	return nil
}

// generateIf generates a Go if expression.
// M-DX25.3: Uses typed IIFE return and conditional type assertions.
// M-DX26: In _impl functions, uses interface{} everywhere.
func (g *Generator) generateIf(ifExpr *core.If) error {
	// M-DX26: In _impl functions, everything is interface{}
	inImplFunc := g.expectedReturnType == "interface{}"

	// M-DX25.3: Look up If expression's type for IIFE return type
	returnType := "interface{}"
	// M-DX26: Skip type inference in _impl functions
	if !inImplFunc {
		// M-DX25.10: Special case for Record branches - infer type from fields
		// Check Then branch first (both branches should have same type)
		if rec, isRec := ifExpr.Then.(*core.Record); isRec {
			fieldNames := make(map[string]bool, len(rec.Fields))
			for name := range rec.Fields {
				fieldNames[name] = true
			}
			if recordType := g.GetRecordTypeByFields(fieldNames); recordType != nil {
				returnType = "*" + recordType.Name
			}
		} else if g.coreTypeInfo != nil {
			if typ, ok := g.coreTypeInfo[ifExpr.NodeID]; ok {
				if goType, err := g.TypeMapper.MapType(typ); err == nil {
					returnType = string(goType)
				}
			}
		}
	}

	g.writef("func() %s {\n", returnType)
	g.indent++
	g.writef("if ")
	if err := g.generateExpr(ifExpr.Cond); err != nil {
		return err
	}
	// M-DX25.3: Only add .(bool) if condition produces interface{}
	if g.exprProducesInterface(ifExpr.Cond) {
		g.write(".(bool)")
	}
	g.write(" {\n")
	g.indent++
	g.writef("return ")
	if err := g.generateExpr(ifExpr.Then); err != nil {
		return err
	}
	// M-DX25.3: Add type assertion if Then branch produces interface{} but we need concrete type
	if returnType != "interface{}" && g.exprProducesInterface(ifExpr.Then) {
		g.writef(".(%s)", returnType)
	}
	g.writef("\n")
	g.indent--
	g.writef("}\n")
	g.writef("return ")
	if err := g.generateExpr(ifExpr.Else); err != nil {
		return err
	}
	// M-DX25.3: Add type assertion if Else branch produces interface{} but we need concrete type
	if returnType != "interface{}" && g.exprProducesInterface(ifExpr.Else) {
		g.writef(".(%s)", returnType)
	}
	g.writef("\n")
	g.indent--
	g.write("}()")
	return nil
}

// mapEffectBuiltinToHandler maps AILANG effect builtin names to Go handler method calls.
// Returns empty string if not an effect builtin (pure functions like string/array ops).
// Uses require* guard functions to provide helpful error messages when handlers aren't initialized.
func mapEffectBuiltinToHandler(name string) string {
	// Effect builtins follow pattern: _effect_method
	// Wrapper functions follow pattern: effect_method (no underscore prefix)
	// Map to: requireEffect().Method (guards ensure helpful panic on nil handler)
	effectMappings := map[string]string{
		// Rand effect - builtins
		"_rand_int":   "requireRand().RandInt",
		"_rand_float": "requireRand().RandFloat",
		"_rand_bool":  "requireRand().RandBool",
		"_rand_seed":  "requireRand().SetSeed",
		// Rand effect - stdlib wrappers (std/rand exports these)
		"rand_int":   "requireRand().RandInt",
		"rand_float": "requireRand().RandFloat",
		"rand_bool":  "requireRand().RandBool",
		"rand_seed":  "requireRand().SetSeed",
		// Clock effect - builtins
		"_clock_now":   "requireClock().Now",
		"_clock_sleep": "requireClock().Sleep",
		// Clock effect - stdlib wrappers
		"clock_now":   "requireClock().Now",
		"clock_sleep": "requireClock().Sleep",
		// Game effect - maps to Clock (alternative API for game engines)
		"_game_delta_time":  "requireClock().DeltaTime",
		"_game_total_time":  "requireClock().TotalTime",
		"_game_frame_count": "requireClock().FrameCount",
		// Debug effect - builtins
		"_debug_log":   "requireDebug().Log",
		"_debug_check": "requireDebug().Assert",
		// Debug effect - stdlib wrappers
		"debug_log":   "requireDebug().Log",
		"debug_check": "requireDebug().Assert",
		// IO effect - use inline Log helper (simpler than full handler)
		"_io_print":   "Log",
		"_io_println": "Log",
		"io_print":    "Log",
		"io_println":  "Log",
		// FS effect - builtins
		"_fs_exists":    "requireFS().Exists",
		"_fs_readFile":  "requireFS().ReadFile",
		"_fs_writeFile": "requireFS().WriteFile",
		// FS effect - stdlib wrappers
		"fs_exists":    "requireFS().Exists",
		"fs_readFile":  "requireFS().ReadFile",
		"fs_writeFile": "requireFS().WriteFile",
		// Net effect - builtins
		"_net_httpGet":     "requireNet().HttpGet",
		"_net_httpPost":    "requireNet().HttpPost",
		"_net_httpRequest": "requireNet().HttpRequest",
		// Net effect - stdlib wrappers
		"net_httpGet":     "requireNet().HttpGet",
		"net_httpPost":    "requireNet().HttpPost",
		"net_httpRequest": "requireNet().HttpRequest",
		// Env effect - builtins
		"_env_getArgs": "requireEnv().GetArgs",
		"_env_getEnv":  "requireEnv().GetEnv",
		"_env_hasEnv":  "requireEnv().HasEnv",
		// Env effect - stdlib wrappers
		"env_getArgs": "requireEnv().GetArgs",
		"env_getEnv":  "requireEnv().GetEnv",
		"env_hasEnv":  "requireEnv().HasEnv",
		// AI effect - builtins
		"_ai_call": "requireAI().Call",
		// AI effect - stdlib wrappers
		"ai_call": "requireAI().Call",
	}
	return effectMappings[name]
}

// Note: These builtins are NOT effect builtins and don't need handler mapping:
// - Pure string ops: _str_* (compiled to Go string operations)
// - Pure array ops: _array_* (compiled to Go slice operations)
// - Pure JSON ops: _json_decode, _json_encode (compiled inline)
// - Pure conversions: _stringToInt, _stringToFloat (compiled inline)

// canEmitNativeOp checks if an App can be emitted as a native Go operator.
// M-DX24.2: Returns true for arithmetic/comparison helpers when operands have known types.
// M-DX26: Returns false in _impl functions where all params are interface{}.
func (g *Generator) canEmitNativeOp(app *core.App) bool {
	// M-DX26: In _impl functions (interface{} world), never emit native ops
	// All params are interface{}, so Go operators won't work
	if g.expectedReturnType == "interface{}" {
		return false
	}

	// Must have exactly 2 arguments for binary ops
	if len(app.Args) != 2 {
		return false
	}

	// Check if function is a known arithmetic/comparison helper
	funcName := g.getAppFuncName(app)
	if funcName == "" {
		return false
	}

	// Check if this is an arithmetic/comparison helper
	op := arithmeticHelperToOp(funcName)
	if op == "" {
		return false
	}

	// Check if both operands have known types
	// For now, we check if operands are:
	// 1. Typed parameters (Var with known type)
	// 2. Literals (always typed)
	// 3. Other expressions that produce concrete types
	return g.operandHasKnownType(app.Args[0]) && g.operandHasKnownType(app.Args[1])
}

// generateNativeOp generates a native Go operator expression.
// M-DX24.2: Emits (a + b) instead of AddInt(a, b).
func (g *Generator) generateNativeOp(app *core.App) error {
	funcName := g.getAppFuncName(app)
	op := arithmeticHelperToOp(funcName)

	g.write("(")
	if err := g.generateExpr(app.Args[0]); err != nil {
		return err
	}
	g.writef(" %s ", op)
	if err := g.generateExpr(app.Args[1]); err != nil {
		return err
	}
	g.write(")")
	return nil
}

// getAppFuncName extracts the function name from an App expression.
func (g *Generator) getAppFuncName(app *core.App) string {
	switch f := app.Func.(type) {
	case *core.Var:
		return f.Name
	case *core.VarGlobal:
		return f.Ref.Name
	default:
		return ""
	}
}

// operandHasKnownType checks if an operand has a known concrete type.
// M-DX24.2: Used to determine if we can emit native operators.
func (g *Generator) operandHasKnownType(expr core.CoreExpr) bool {
	switch e := expr.(type) {
	case *core.Lit:
		// Literals always have concrete types
		return true

	case *core.Var:
		// Variables are typed if they're function parameters
		// We can't easily check this at codegen time, so we're conservative
		// and assume local variables are typed (they come from typed function params)
		return true

	case *core.VarGlobal:
		// Global variables might be typed
		return true

	case *core.App:
		// Function calls - check if the function returns a concrete type
		funcName := g.getAppFuncName(e)
		if retType := runtimeHelperReturnType(funcName); retType != "" && retType != "interface{}" {
			return true
		}
		// ADT constructors return concrete types
		if _, isADT := g.adtConstructors[funcName]; isADT {
			return true
		}
		// Top-level functions may return concrete types
		if _, isTopLevel := g.topLevelFuncs[funcName]; isTopLevel {
			return true
		}
		// Arithmetic helpers we're about to emit as native ops
		if arithmeticHelperToOp(funcName) != "" {
			return true
		}
		return false

	default:
		return false
	}
}

// arithmeticHelperToOp maps arithmetic helper function names to Go operators.
// M-DX24.2: Returns empty string if not a known arithmetic helper.
func arithmeticHelperToOp(name string) string {
	switch name {
	// Integer arithmetic
	case "add_Int", "AddInt":
		return "+"
	case "sub_Int", "SubInt":
		return "-"
	case "mul_Int", "MulInt":
		return "*"
	case "div_Int", "DivInt":
		return "/"
	case "mod_Int", "ModInt":
		return "%"

	// Float arithmetic
	case "add_Float", "AddFloat":
		return "+"
	case "sub_Float", "SubFloat":
		return "-"
	case "mul_Float", "MulFloat":
		return "*"
	case "div_Float", "DivFloat":
		return "/"

	// Integer comparisons
	case "eq_Int", "EqInt":
		return "=="
	case "ne_Int", "NeInt":
		return "!="
	case "lt_Int", "LtInt":
		return "<"
	case "le_Int", "LeInt":
		return "<="
	case "gt_Int", "GtInt":
		return ">"
	case "ge_Int", "GeInt":
		return ">="

	// Float comparisons
	case "eq_Float", "EqFloat":
		return "=="
	case "ne_Float", "NeFloat":
		return "!="
	case "lt_Float", "LtFloat":
		return "<"
	case "le_Float", "LeFloat":
		return "<="
	case "gt_Float", "GtFloat":
		return ">"
	case "ge_Float", "GeFloat":
		return ">="

	// Boolean operations
	case "and_Bool", "AndBool":
		return "&&"
	case "or_Bool", "OrBool":
		return "||"

	default:
		return ""
	}
}
