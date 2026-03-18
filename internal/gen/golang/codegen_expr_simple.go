// Package golang provides Go code generation from AILANG Core AST.
package golang

import (
	"fmt"
	"strings"

	"github.com/sunholo/ailang/internal/core"
)

// getLitGoType returns the Go type that generateLit will produce for a literal.
// M-CODEGEN-V2.M4: Used to avoid redundant type conversions like int64(int64(1)).
func (g *Generator) getLitGoType(lit *core.Lit) string {
	switch lit.Kind {
	case core.IntLit:
		return "int64"
	case core.FloatLit:
		return "float64"
	case core.BoolLit:
		return "bool"
	case core.StringLit:
		return "string"
	case core.UnitLit:
		return "struct{}"
	default:
		return ""
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

// generateVar generates code for a Var expression (local variable reference).
func (g *Generator) generateVar(v *core.Var) error {
	// M-CODEGEN-LETBIND-FIX: Check top-level variables FIRST (non-function lets).
	// These must NOT get _impl suffix — they are plain Go variables.
	if goName, ok := g.topLevelVars[v.Name]; ok {
		g.write(goName)
		return nil
	}

	// Check if this is a reference to a top-level function
	if goName, ok := g.topLevelFuncs[v.Name]; ok {
		// M-DX26: In _impl functions, call other _impl functions
		// M-DX18-FIX: Use topLevelImplFuncs which has the correct _impl name
		// (ToGoVarName for _impl differs from ToGoFuncName for wrapper on exported funcs)
		if g.expectedReturnType == "interface{}" {
			if implName, ok := g.topLevelImplFuncs[v.Name]; ok {
				g.write(implName)
			} else {
				// Fallback for backwards compatibility
				g.write(goName + "_impl")
			}
		} else {
			g.write(goName)
		}
	} else {
		// M-CODEGEN-COMPILE-GATE: Check if this local var name matches a builtin.
		// This handles cases like `println` being passed as a first-class function
		// argument — the Core IR binds it as a Var, but we need to resolve it
		// to our helper function name (e.g., Println) to avoid shadowing Go built-ins.
		if resolved := g.resolveBuiltinViaRegistry(v.Name); resolved != "" {
			g.write(resolved)
		} else {
			g.write(ToGoVarName(v.Name))
		}
	}
	return nil
}

// generateVarGlobal generates code for a VarGlobal expression (module-qualified reference).
func (g *Generator) generateVarGlobal(e *core.VarGlobal) error {
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

			// M-DX22: Use qualified lookup with typeName for disambiguation
			if ctorInfo, ok := g.LookupADTConstructor(typeName, ctorName); ok && ctorInfo.FieldCount == 0 {
				g.write(goFuncName + "()")
			} else {
				g.write(goFuncName)
			}
			return nil
		}
	}

	// Check if this is a registered ADT constructor by name
	// M-DX22: Use LookupADTConstructor for backwards-compatible fallback search
	if ctorInfo, ok := g.LookupADTConstructor("", e.Ref.Name); ok {
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

	// M-CODEGEN-SUSTAINABILITY: Query the builtin registry for Go codegen specs.
	// This replaces mapPureMathBuiltin, mapPureListBuiltin, and mapStdlibBuiltin.
	if resolved := g.resolveBuiltinViaRegistry(e.Ref.Name); resolved != "" {
		g.write(resolved)
		return nil
	}

	// For other global references
	// M-CODEGEN-TYPE-ASSERTIONS: In _impl functions, call other _impl functions
	// to avoid type mismatches (typed exports expect concrete types, not interface{})
	if g.expectedReturnType == "interface{}" {
		// M-CODEGEN-CROSS-MODULE: Check if this is from another user-defined module
		// (not a pseudo-module like $adt, $builtin, and not stdlib like std/*)
		// Cross-module function references need _impl versions to stay in interface{} land
		// BUT stdlib modules (std/*) don't generate _impl - they use typed wrappers that call runtime helpers
		if e.Ref.Module != "" && !strings.HasPrefix(e.Ref.Module, "$") && !strings.HasPrefix(e.Ref.Module, "std/") {
			// M-CODEGEN-MULTIMOD: Cross-module references need the target module's prefix
			parts := strings.Split(e.Ref.Module, "/")
			targetModule := parts[len(parts)-1]
			// Sanitize module name same as compile.go does
			targetModule = strings.Map(func(r rune) rune {
				if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
					return r
				}
				if r == '-' {
					return '_'
				}
				return '_'
			}, targetModule)
			g.write(targetModule + "__" + ToGoVarName(e.Ref.Name) + "_impl")
			return nil
		}
		// Check if this is a known top-level function (has _impl version)
		// M-DX18-FIX: Use topLevelImplFuncs which has the correct _impl name
		if implName, isTopLevel := g.topLevelImplFuncs[e.Ref.Name]; isTopLevel {
			g.write(implName)
			return nil
		}
	}
	g.write(ToPascalCase(e.Ref.Name))
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

// M-CODEGEN-SUSTAINABILITY: mapPureMathBuiltin, mapPureListBuiltin, and mapStdlibBuiltin
// have been replaced by resolveBuiltinViaRegistry() in codegen_registry.go.
// The registry queries BuiltinMeta.GoCodegenSpec instead of hardcoded mapping tables.

// mathConstants lists the math builtins that are constants (not functions).
// M-CODEGEN-STDLIB-MATH Bug #27: These should not be called with ().
var mathConstants = map[string]string{
	"_math_PI": "math.Pi",
	"_math_E":  "math.E",
	"PI":       "math.Pi",
	"E":        "math.E",
}

// mathFunctions lists the math builtins that are functions (need float64 args).
// M-CODEGEN-STDLIB-MATH Bug #27: These need .(float64) type assertions on args.
var mathFunctions = map[string]string{
	// Trig functions
	"_math_sin": "math.Sin",
	"_math_cos": "math.Cos",
	"_math_tan": "math.Tan",
	"sin":       "math.Sin",
	"cos":       "math.Cos",
	"tan":       "math.Tan",
	// Inverse trig
	"_math_asin":  "math.Asin",
	"_math_acos":  "math.Acos",
	"_math_atan":  "math.Atan",
	"_math_atan2": "math.Atan2",
	"asin":        "math.Asin",
	"acos":        "math.Acos",
	"atan":        "math.Atan",
	"atan2":       "math.Atan2",
	// Exponential/logarithmic
	"_math_exp":   "math.Exp",
	"_math_log":   "math.Log",
	"_math_log10": "math.Log10",
	"_math_pow":   "math.Pow",
	"_math_sqrt":  "math.Sqrt",
	"exp":         "math.Exp",
	"log":         "math.Log",
	"log10":       "math.Log10",
	"pow":         "math.Pow",
	"sqrt":        "math.Sqrt",
	// Rounding
	"_math_ceil":  "math.Ceil",
	"_math_floor": "math.Floor",
	"_math_round": "math.Round",
	"ceil":        "math.Ceil",
	"floor":       "math.Floor",
	"round":       "math.Round",
	// Utility
	"_math_abs_Float": "math.Abs",
	"abs_Float":       "math.Abs",
}

// getMathConstant checks if a function expression refers to a math constant (PI, E).
// Returns the Go expression (e.g., "math.Pi") or empty string if not a constant.
// M-CODEGEN-STDLIB-MATH Bug #27: Used to emit constants without () in App.
func (g *Generator) getMathConstant(funcExpr core.CoreExpr) string {
	name := ""
	if v, ok := funcExpr.(*core.VarGlobal); ok {
		name = v.Ref.Name
	} else if v, ok := funcExpr.(*core.Var); ok {
		name = v.Name
	}
	if name == "" {
		return ""
	}
	if goExpr, ok := mathConstants[name]; ok {
		g.needsMathImport = true
		return goExpr
	}
	return ""
}

// getMathFunction checks if a function expression refers to a math function (sin, cos, etc).
// Returns the Go function name (e.g., "math.Sin") or empty string if not a math function.
// M-CODEGEN-STDLIB-MATH Bug #27: Used to emit functions with float64 type assertions.
func (g *Generator) getMathFunction(funcExpr core.CoreExpr) string {
	name := ""
	if v, ok := funcExpr.(*core.VarGlobal); ok {
		name = v.Ref.Name
	} else if v, ok := funcExpr.(*core.Var); ok {
		name = v.Name
	}
	if name == "" {
		return ""
	}
	if goFunc, ok := mathFunctions[name]; ok {
		g.needsMathImport = true
		return goFunc
	}
	return ""
}

// StringConvKind represents the type of string conversion function.
type StringConvKind int

const (
	StringConvNone StringConvKind = iota
	StringConvFloatToStr
	StringConvIntToStr
)

// stringConvFunctions maps AILANG string conversion builtins to their kind.
// M-CODEGEN-STDLIB-STRING: Used to emit strconv calls with proper arguments.
var stringConvFunctions = map[string]StringConvKind{
	// Builtins (underscore prefix)
	"_string_floatToStr": StringConvFloatToStr,
	"_string_intToStr":   StringConvIntToStr,
	// stdlib wrappers
	"floatToStr": StringConvFloatToStr,
	"intToStr":   StringConvIntToStr,
}

// getStringConvFunction checks if a function expression refers to a string conversion function.
// Returns the conversion kind or StringConvNone if not a string conversion.
// M-CODEGEN-STDLIB-STRING: Used to emit strconv calls with proper arguments.
func (g *Generator) getStringConvFunction(funcExpr core.CoreExpr) StringConvKind {
	name := ""
	if v, ok := funcExpr.(*core.VarGlobal); ok {
		name = v.Ref.Name
	} else if v, ok := funcExpr.(*core.Var); ok {
		name = v.Name
	}
	if name == "" {
		return StringConvNone
	}
	if kind, ok := stringConvFunctions[name]; ok {
		g.needsStrconvImport = true
		return kind
	}
	return StringConvNone
}
