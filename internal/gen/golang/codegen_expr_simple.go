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
	// Check if this is a reference to a top-level function
	if goName, ok := g.topLevelFuncs[v.Name]; ok {
		// M-DX26: In _impl functions, call other _impl functions
		if g.expectedReturnType == "interface{}" {
			g.write(ToGoVarName(v.Name) + "_impl")
		} else {
			g.write(goName)
		}
	} else {
		g.write(ToGoVarName(v.Name))
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

	// For other global references
	// M-CODEGEN-TYPE-ASSERTIONS: In _impl functions, call other _impl functions
	// to avoid type mismatches (typed exports expect concrete types, not interface{})
	if g.expectedReturnType == "interface{}" {
		// Check if this is a known top-level function (has _impl version)
		if _, isTopLevel := g.topLevelFuncs[e.Ref.Name]; isTopLevel {
			g.write(ToGoVarName(e.Ref.Name) + "_impl")
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
