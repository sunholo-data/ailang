// Package golang provides Go code generation from AILANG Core AST.
package golang

import (
	"strings"

	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/types"
)

// generateApp generates a Go function application.
func (g *Generator) generateApp(app *core.App) error {
	// M-DX24.2: Check if this is an arithmetic helper that can be emitted as native operator
	if g.canEmitNativeOp(app) {
		return g.generateNativeOp(app)
	}

	// M-CODEGEN-STDLIB-MATH Bug #27: Handle math constants and functions
	if mathConst := g.getMathConstant(app.Func); mathConst != "" {
		// Math constants (PI, E) - emit without () even though AILANG calls them as PI()
		g.write(mathConst)
		return nil
	}
	if mathFunc := g.getMathFunction(app.Func); mathFunc != "" {
		// Math functions (sin, cos, etc.) - emit with float64 type assertions on args
		g.write(mathFunc + "(")
		for i, arg := range app.Args {
			if i > 0 {
				g.write(", ")
			}
			// M-CODEGEN-STDLIB-MATH: Wrap args in .(float64) for interface{} values
			if g.exprProducesInterface(arg) {
				if err := g.generateExpr(arg); err != nil {
					return err
				}
				g.write(".(float64)")
			} else {
				if err := g.generateExpr(arg); err != nil {
					return err
				}
			}
		}
		g.write(")")
		return nil
	}

	// M-CODEGEN-STDLIB-STRING: Handle string conversion functions
	if strConv := g.getStringConvFunction(app.Func); strConv != StringConvNone {
		switch strConv {
		case StringConvFloatToStr:
			// floatToStr(f) → strconv.FormatFloat(f.(float64), 'g', -1, 64)
			g.write("strconv.FormatFloat(")
			if len(app.Args) > 0 {
				if g.exprProducesInterface(app.Args[0]) {
					if err := g.generateExpr(app.Args[0]); err != nil {
						return err
					}
					g.write(".(float64)")
				} else {
					if err := g.generateExpr(app.Args[0]); err != nil {
						return err
					}
				}
			}
			g.write(", 'g', -1, 64)")
			return nil
		case StringConvIntToStr:
			// intToStr(n) → strconv.Itoa(int(n.(int64)))
			g.write("strconv.Itoa(int(")
			if len(app.Args) > 0 {
				if g.exprProducesInterface(app.Args[0]) {
					if err := g.generateExpr(app.Args[0]); err != nil {
						return err
					}
					g.write(".(int64)")
				} else {
					if err := g.generateExpr(app.Args[0]); err != nil {
						return err
					}
				}
			}
			g.write("))")
			return nil
		}
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
						// M-CODEGEN-V2.M4: Only wrap literals if their native type differs
						// from the expected type. generateLit already adds int64()/float64().
						litType := g.getLitGoType(lit)
						if litType != goType && litType != "" {
							// Type conversion needed (e.g., int to float)
							g.writef("%s(", goType)
							if err := g.generateExpr(lit); err != nil {
								return err
							}
							g.write(")")
						} else {
							// Same type - just generate the literal (already typed)
							if err := g.generateExpr(lit); err != nil {
								return err
							}
						}
					} else {
						// M-CODEGEN-ADT-TYPE-ASSERT: Only add type assertion if arg produces interface{}
						// ADT constructor calls (like NewSpectralClassG()) return typed values, not interface{}
						if err := g.generateExpr(arg); err != nil {
							return err
						}
						if g.exprProducesInterface(arg) {
							g.writef(".(%s)", goType)
						}
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
