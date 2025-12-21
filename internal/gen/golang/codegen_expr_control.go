// Package golang provides Go code generation from AILANG Core AST.
package golang

import (
	"github.com/sunholo/ailang/internal/core"
)

// chainBranch represents a single branch in an if-else chain.
// M-CODEGEN-FLAT-IF-ELSE: Used to flatten nested if-else into linear if statements.
type chainBranch struct {
	Cond core.CoreExpr // nil for the final else branch
	Then core.CoreExpr
}

// unwrapToIf unwraps Let expressions to find an inner If.
// M-CODEGEN-FLAT-IF-ELSE: The elaborator wraps conditions in Lets, so we need to look through them.
// Returns the If expression and any Let bindings that wrap it.
func unwrapToIf(expr core.CoreExpr) (*core.If, []*core.Let) {
	var lets []*core.Let
	current := expr
	for {
		switch e := current.(type) {
		case *core.If:
			return e, lets
		case *core.Let:
			lets = append(lets, e)
			current = e.Body
		default:
			return nil, nil
		}
	}
}

// isIfElseChain checks if an if expression is part of a chain.
// M-CODEGEN-FLAT-IF-ELSE: Identifies chains worth flattening.
// Looks through Let wrappers since the elaborator creates Let bindings for conditions.
func isIfElseChain(ifExpr *core.If) bool {
	// Check if else branch is an If (directly or wrapped in Lets)
	nextIf, _ := unwrapToIf(ifExpr.Else)
	return nextIf != nil
}

// collectIfChain extracts all branches from an if-else chain.
// M-CODEGEN-FLAT-IF-ELSE: Walks the chain and collects conditions, bodies, and Let bindings.
// Returns branches and all Let bindings needed before the flat if-else.
func collectIfChain(ifExpr *core.If) ([]chainBranch, []*core.Let) {
	var branches []chainBranch
	var allLets []*core.Let
	current := ifExpr
	for {
		branches = append(branches, chainBranch{
			Cond: current.Cond,
			Then: current.Then,
		})
		// Try to unwrap else branch to find next If
		nextIf, lets := unwrapToIf(current.Else)
		if nextIf != nil {
			allLets = append(allLets, lets...)
			current = nextIf
		} else {
			// Final else branch (no condition)
			branches = append(branches, chainBranch{
				Cond: nil,
				Then: current.Else,
			})
			break
		}
	}
	return branches, allLets
}

// generateIf generates a Go if expression.
// M-DX25.3: Uses typed IIFE return and conditional type assertions.
// M-DX26: In _impl functions, uses interface{} everywhere.
// M-CODEGEN-FLAT-IF-ELSE: Detects if-else chains and generates flat code.
func (g *Generator) generateIf(ifExpr *core.If) error {
	// M-CODEGEN-FLAT-IF-ELSE: Detect and flatten if-else chains
	// Only flatten at the chain root (not when already inside a chain)
	if !g.inFlatChain && isIfElseChain(ifExpr) {
		return g.generateIfChain(ifExpr)
	}

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

// generateIfChain generates flat Go if statements for an if-else chain.
// M-CODEGEN-FLAT-IF-ELSE: Produces linear code instead of nested closures.
// Example: if c1 { return v1 } if c2 { return v2 } return vN
func (g *Generator) generateIfChain(ifExpr *core.If) error {
	// M-DX26: In _impl functions, everything is interface{}
	inImplFunc := g.expectedReturnType == "interface{}"

	// Determine return type for the IIFE wrapper
	returnType := "interface{}"
	if !inImplFunc {
		// Try to infer type from the first Then branch
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

	// Collect all branches and Let bindings
	branches, lets := collectIfChain(ifExpr)

	// Generate single IIFE wrapper
	g.writef("func() %s {\n", returnType)
	g.indent++

	// Set flag to prevent nested chains from re-wrapping
	oldInFlatChain := g.inFlatChain
	g.inFlatChain = true

	// Generate all Let bindings upfront (conditions evaluated before if-else)
	for _, let := range lets {
		g.writef("var %s interface{} = ", ToGoVarName(let.Name))
		if err := g.generateExpr(let.Value); err != nil {
			g.inFlatChain = oldInFlatChain
			return err
		}
		g.writef("\n")
		g.writef("_ = %s // suppress unused\n", ToGoVarName(let.Name))
	}

	// Generate flat if statements for each branch (except the last)
	for i, branch := range branches {
		if branch.Cond != nil {
			// Conditional branch: if cond { return value }
			g.writef("if ")
			if err := g.generateExpr(branch.Cond); err != nil {
				g.inFlatChain = oldInFlatChain
				return err
			}
			if g.exprProducesInterface(branch.Cond) {
				g.write(".(bool)")
			}
			g.write(" {\n")
			g.indent++
			g.writef("return ")
			if err := g.generateExpr(branch.Then); err != nil {
				g.inFlatChain = oldInFlatChain
				return err
			}
			if returnType != "interface{}" && g.exprProducesInterface(branch.Then) {
				g.writef(".(%s)", returnType)
			}
			g.writef("\n")
			g.indent--
			g.writef("}\n")
		} else {
			// Final else branch (no condition): return value
			if i != len(branches)-1 {
				// Safety check: nil Cond should only be on last branch
				g.inFlatChain = oldInFlatChain
				return g.generateExpr(branch.Then) // fallback
			}
			g.writef("return ")
			if err := g.generateExpr(branch.Then); err != nil {
				g.inFlatChain = oldInFlatChain
				return err
			}
			if returnType != "interface{}" && g.exprProducesInterface(branch.Then) {
				g.writef(".(%s)", returnType)
			}
			g.writef("\n")
		}
	}

	g.inFlatChain = oldInFlatChain
	g.indent--
	g.write("}()")
	return nil
}

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
		// M-DX22: Use LookupADTConstructor for backwards-compatible lookup
		if _, ok := g.LookupADTConstructor("", funcName); ok {
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
