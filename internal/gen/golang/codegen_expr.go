// Package golang provides Go code generation from AILANG Core AST.
package golang

import (
	"fmt"
	"strings"

	"github.com/sunholo/ailang/internal/core"
)

// generateExpr generates Go code for a Core expression.
func (g *Generator) generateExpr(expr core.CoreExpr) error {
	switch e := expr.(type) {
	case *core.Lit:
		return g.generateLit(e)

	case *core.Var:
		// Check if this is a reference to a top-level function
		if goName, ok := g.topLevelFuncs[e.Name]; ok {
			g.write(goName)
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
func (g *Generator) generateLit(lit *core.Lit) error {
	switch lit.Kind {
	case core.IntLit:
		if v, ok := lit.Value.(int64); ok {
			g.writef("%d", v)
		} else if v, ok := lit.Value.(int); ok {
			g.writef("%d", v)
		} else {
			g.writef("%v", lit.Value)
		}
	case core.FloatLit:
		g.writef("%v", lit.Value)
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
		if err := g.generateExpr(app.Func); err != nil {
			return err
		}
		g.write("(")
		for i, arg := range app.Args {
			if i > 0 {
				g.write(", ")
			}
			if err := g.generateExpr(arg); err != nil {
				return err
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

// generateLet generates a Go variable binding.
func (g *Generator) generateLet(let *core.Let) error {
	g.writef("func() interface{} {\n")
	g.indent++
	g.writef("%s := ", ToGoVarName(let.Name))
	if err := g.generateExpr(let.Value); err != nil {
		return err
	}
	g.writef("\n")
	g.writef("_ = %s // suppress unused\n", ToGoVarName(let.Name))
	g.writef("return ")
	if err := g.generateExpr(let.Body); err != nil {
		return err
	}
	g.writef("\n")
	g.indent--
	g.write("}()")
	return nil
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
func (g *Generator) generateIf(ifExpr *core.If) error {
	g.write("func() interface{} {\n")
	g.indent++
	g.writef("if ")
	if err := g.generateExpr(ifExpr.Cond); err != nil {
		return err
	}
	g.write(".(bool) {\n")
	g.indent++
	g.writef("return ")
	if err := g.generateExpr(ifExpr.Then); err != nil {
		return err
	}
	g.writef("\n")
	g.indent--
	g.writef("}\n")
	g.writef("return ")
	if err := g.generateExpr(ifExpr.Else); err != nil {
		return err
	}
	g.writef("\n")
	g.indent--
	g.write("}()")
	return nil
}

// mapEffectBuiltinToHandler maps AILANG effect builtin names to Go handler method calls.
// Returns empty string if not an effect builtin.
func mapEffectBuiltinToHandler(name string) string {
	// Effect builtins follow pattern: _effect_method
	// Wrapper functions follow pattern: effect_method (no underscore prefix)
	// Map to: handlers.Effect.Method
	effectMappings := map[string]string{
		// Rand effect - builtins
		"_rand_int":   "handlers.Rand.RandInt",
		"_rand_float": "handlers.Rand.RandFloat",
		"_rand_bool":  "handlers.Rand.RandBool",
		"_rand_seed":  "handlers.Rand.SetSeed",
		// Rand effect - stdlib wrappers (std/rand exports these)
		"rand_int":   "handlers.Rand.RandInt",
		"rand_float": "handlers.Rand.RandFloat",
		"rand_bool":  "handlers.Rand.RandBool",
		"rand_seed":  "handlers.Rand.SetSeed",
		// Clock effect - builtins
		"_clock_now":   "handlers.Clock.Now",
		"_clock_sleep": "handlers.Clock.Sleep",
		// Clock effect - stdlib wrappers
		"clock_now":   "handlers.Clock.Now",
		"clock_sleep": "handlers.Clock.Sleep",
		// Debug effect - builtins
		"_debug_log":   "handlers.Debug.Log",
		"_debug_check": "handlers.Debug.Assert",
		// Debug effect - stdlib wrappers
		"debug_log":   "handlers.Debug.Log",
		"debug_check": "handlers.Debug.Assert",
	}
	return effectMappings[name]
}
