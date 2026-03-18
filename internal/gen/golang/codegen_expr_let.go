// Package golang provides Go code generation from AILANG Core AST.
package golang

import (
	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/types"
)

// isLetIfChain checks if a Let's body is an If that starts an if-else chain.
// M-CODEGEN-FLAT-IF-ELSE: Used to detect chains at the Let level.
func isLetIfChain(let *core.Let) bool {
	ifExpr, ok := let.Body.(*core.If)
	if !ok {
		return false
	}
	return isIfElseChain(ifExpr)
}

// isLetChain checks if a Let's body is another Let (forming a chain).
// M-CODEGEN-LIST: Used to detect Let chains that should be flattened.
func isLetChain(let *core.Let) bool {
	_, ok := let.Body.(*core.Let)
	return ok
}

// isLetBoolMatchChain checks if a Let starts a bool match chain pattern.
// M-CODEGEN-LIST: Pattern: Let $cmp = <comparison> in Match $cmp { true => A, false => ... }
// where the false arm continues the chain.
func isLetBoolMatchChain(let *core.Let) bool {
	// Body must be a Match
	match, ok := let.Body.(*core.Match)
	if !ok {
		return false
	}

	// Must have exactly 2 arms
	if len(match.Arms) != 2 {
		return false
	}

	// Both arms must be bool literal patterns
	var trueArm, falseArm *core.MatchArm
	for i := range match.Arms {
		arm := &match.Arms[i]
		if lp, ok := arm.Pattern.(*core.LitPattern); ok {
			// LitPattern.Value is the Go value directly (bool), not *core.Lit
			if val, ok := lp.Value.(bool); ok {
				if val {
					trueArm = arm
				} else {
					falseArm = arm
				}
			}
		}
	}

	if trueArm == nil || falseArm == nil {
		return false
	}

	// False arm must continue the chain (Let or Match)
	switch falseArm.Body.(type) {
	case *core.Let, *core.Match:
		return true
	default:
		return false
	}
}

// LetBoolMatchEntry represents one condition-result pair from a Let-Bool-Match chain.
type LetBoolMatchEntry struct {
	Condition core.CoreExpr // The actual comparison expression (e.g., LtFloat(roll, 0.76))
	TrueBody  core.CoreExpr // What to return when condition is true
}

// collectLetBoolMatchChain extracts a chain from Let-Bool-Match pattern.
// Returns entries for each condition and the final else body.
func collectLetBoolMatchChain(let *core.Let) ([]LetBoolMatchEntry, core.CoreExpr) {
	var entries []LetBoolMatchEntry
	currentExpr := core.CoreExpr(let)

	for {
		// Current must be a Let with Match body
		currentLet, ok := currentExpr.(*core.Let)
		if !ok {
			// Not a Let - check if it's a direct Match (continuation)
			if match, ok := currentExpr.(*core.Match); ok {
				// Direct Match - use scrutinee as condition (already computed)
				if len(match.Arms) == 2 {
					var trueArm, falseArm *core.MatchArm
					for i := range match.Arms {
						arm := &match.Arms[i]
						if lp, ok := arm.Pattern.(*core.LitPattern); ok {
							// LitPattern.Value is the Go value directly (bool)
							if val, ok := lp.Value.(bool); ok {
								if val {
									trueArm = arm
								} else {
									falseArm = arm
								}
							}
						}
					}
					if trueArm != nil && falseArm != nil {
						entries = append(entries, LetBoolMatchEntry{
							Condition: match.Scrutinee,
							TrueBody:  trueArm.Body,
						})
						// Continue with false arm
						switch fb := falseArm.Body.(type) {
						case *core.Let, *core.Match:
							currentExpr = fb
							continue
						default:
							return entries, falseArm.Body
						}
					}
				}
			}
			return entries, currentExpr
		}

		match, ok := currentLet.Body.(*core.Match)
		if !ok {
			return entries, currentExpr
		}

		// Extract true and false arms
		if len(match.Arms) != 2 {
			return entries, currentExpr
		}

		var trueArm, falseArm *core.MatchArm
		for i := range match.Arms {
			arm := &match.Arms[i]
			if lp, ok := arm.Pattern.(*core.LitPattern); ok {
				// LitPattern.Value is the Go value directly (bool)
				if val, ok := lp.Value.(bool); ok {
					if val {
						trueArm = arm
					} else {
						falseArm = arm
					}
				}
			}
		}

		if trueArm == nil || falseArm == nil {
			return entries, currentExpr
		}

		// Add entry with the Let's Value as the condition
		entries = append(entries, LetBoolMatchEntry{
			Condition: currentLet.Value, // The actual comparison (LtFloat, etc.)
			TrueBody:  trueArm.Body,
		})

		// Continue with false arm
		switch fb := falseArm.Body.(type) {
		case *core.Let, *core.Match:
			currentExpr = fb
		default:
			// End of chain - falseBody is the final else
			return entries, falseArm.Body
		}
	}
}

// generateLetBoolMatchChain generates a flat if-else chain from Let-Bool-Match pattern.
// M-CODEGEN-LIST: Eliminates O(n) nested IIFEs for bool match chains.
func (g *Generator) generateLetBoolMatchChain(let *core.Let) error {
	entries, elseBody := collectLetBoolMatchChain(let)

	// Determine return type
	returnType := "interface{}"
	inImplFunc := g.expectedReturnType == "interface{}"
	if !inImplFunc {
		if g.coreTypeInfo != nil {
			if typ, ok := g.coreTypeInfo[let.NodeID]; ok {
				if goType, err := g.TypeMapper.MapType(typ); err == nil {
					returnType = string(goType)
				}
			}
		}
	}

	// Generate single IIFE with flat if-else
	g.writef("func() %s {\n", returnType)
	g.indent++

	for i, entry := range entries {
		if i == 0 {
			g.writef("if ")
		} else {
			g.writef("} else if ")
		}
		if err := g.generateExpr(entry.Condition); err != nil {
			return err
		}
		if g.exprProducesInterface(entry.Condition) {
			g.write(".(bool)")
		}
		g.write(" {\n")
		g.indent++
		g.writef("return ")
		if err := g.generateExpr(entry.TrueBody); err != nil {
			return err
		}
		g.writef("\n")
		g.indent--
	}

	// Final else
	g.writef("} else {\n")
	g.indent++
	g.writef("return ")
	if err := g.generateExpr(elseBody); err != nil {
		return err
	}
	g.writef("\n")
	g.indent--
	g.writef("}\n")

	g.indent--
	g.write("}()")
	return nil
}

// generateFlatLetChain generates a chain of Let bindings as flat code in a single IIFE.
// M-CODEGEN-LIST: Instead of nesting IIFEs for each Let, emit all bindings sequentially.
func (g *Generator) generateFlatLetChain(let *core.Let) error {
	// Collect all Let bindings in the chain
	var bindings []*core.Let
	current := let
	for {
		bindings = append(bindings, current)
		if nextLet, ok := current.Body.(*core.Let); ok {
			current = nextLet
		} else {
			break
		}
	}
	// current.Body is the final body (not a Let)
	finalBody := current.Body

	// Determine return type from the outermost let (its type = final body's type)
	returnType := "interface{}"
	inImplFunc := g.expectedReturnType == "interface{}"
	if !inImplFunc {
		if g.coreTypeInfo != nil {
			if typ, ok := g.coreTypeInfo[let.NodeID]; ok {
				if goType, err := g.TypeMapper.MapType(typ); err == nil {
					returnType = string(goType)
				}
			}
		}
	}

	// Generate single IIFE wrapper
	g.writef("func() %s {\n", returnType)
	g.indent++

	// Set flag to prevent nested chains from re-wrapping
	oldInFlatChain := g.inFlatChain
	g.inFlatChain = true

	// Generate all Let bindings sequentially
	for _, binding := range bindings {
		g.writef("var %s interface{} = ", ToGoVarName(binding.Name))
		if err := g.generateExpr(binding.Value); err != nil {
			g.inFlatChain = oldInFlatChain
			return err
		}
		g.writef("\n")
		g.writeSuppressUnused(ToGoVarName(binding.Name))
	}

	// Generate the final body
	g.writef("return ")
	if err := g.generateExpr(finalBody); err != nil {
		g.inFlatChain = oldInFlatChain
		return err
	}
	if returnType != "interface{}" && g.exprProducesInterface(finalBody) {
		g.writef(".(%s)", returnType)
	}
	g.writef("\n")

	g.inFlatChain = oldInFlatChain
	g.indent--
	g.write("}()")
	return nil
}

// generateLet generates a Go variable binding.
// M-DX25.2: Use SEPARATE types for variable (value's type) and IIFE return (body's type).
// Bug fix: Originally used let's type for both, but let's type IS body's type, not value's type.
// M-CODEGEN-FLAT-IF-ELSE: Detects if-else chains and generates flat code.
// M-CODEGEN-LIST: Detects Let chains and flattens them.
func (g *Generator) generateLet(let *core.Let) error {
	// M-CODEGEN-LIST: Detect if this Let starts a chain of Lets
	// If so, flatten them into a single IIFE
	if !g.inFlatChain && isLetChain(let) {
		return g.generateFlatLetChain(let)
	}

	// M-CODEGEN-LIST: Detect if this Let starts a bool match chain
	// Pattern: Let $cmp = <comparison> in Match $cmp { true => A, false => ... }
	if !g.inFlatChain && isLetBoolMatchChain(let) {
		return g.generateLetBoolMatchChain(let)
	}

	// M-CODEGEN-FLAT-IF-ELSE: Detect if this Let starts an if-else chain
	// If so, delegate to the chain generator
	if !g.inFlatChain && isLetIfChain(let) {
		return g.generateLetIfChain(let)
	}

	// M-DX26: In _impl functions, everything is interface{}
	inImplFunc := g.expectedReturnType == "interface{}"

	// M-DX25.2 FIX: Variable type comes from VALUE expression, not the let expression
	varType := "interface{}"

	// M-DX26: Skip type inference in _impl functions
	if !inImplFunc {
		// M-DX25.10: Special case for Record expressions - infer type from fields
		// TypeMapper returns "struct{}" for TRecord, but we can get the proper
		// struct name by looking up the record type from its fields.
		// M-TYPENAME-NESTED-PROPAGATION: First check coreTypeInfo for TypeName
		if rec, isRec := let.Value.(*core.Record); isRec {
			// First try coreTypeInfo for TypeName (set during unification)
			foundFromTypeInfo := false
			if g.coreTypeInfo != nil {
				if typ, ok := g.coreTypeInfo[rec.NodeID]; ok {
					if tRec, ok := typ.(*types.TRecord); ok && tRec.TypeName != "" {
						// Use TypeName from type checking
						varType = "*" + ToGoTypeName(tRec.TypeName)
						foundFromTypeInfo = true
					}
				}
			}
			// Fall back to field matching if no TypeName
			if !foundFromTypeInfo {
				fieldNames := make(map[string]bool, len(rec.Fields))
				for name := range rec.Fields {
					fieldNames[name] = true
				}
				if recordType := g.GetRecordTypeByFields(fieldNames); recordType != nil {
					varType = "*" + recordType.Name // Records are generated as pointers
				}
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

	g.writeSuppressUnused(ToGoVarName(let.Name))
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

// generateLetIfChain generates a Let that starts an if-else chain as flat code.
// M-CODEGEN-FLAT-IF-ELSE: Collects the initial Let binding and all chain branches,
// then generates a single IIFE with all bindings and flat if statements.
func (g *Generator) generateLetIfChain(let *core.Let) error {
	// The body must be an If (verified by isLetIfChain)
	ifExpr := let.Body.(*core.If)

	// Determine return type for the IIFE wrapper
	returnType := "interface{}"
	inImplFunc := g.expectedReturnType == "interface{}"
	if !inImplFunc {
		if g.coreTypeInfo != nil {
			if typ, ok := g.coreTypeInfo[let.NodeID]; ok {
				if goType, err := g.TypeMapper.MapType(typ); err == nil {
					returnType = string(goType)
				}
			}
		}
	}

	// Collect all branches and Let bindings from the chain
	branches, chainLets := collectIfChain(ifExpr)

	// Generate single IIFE wrapper
	g.writef("func() %s {\n", returnType)
	g.indent++

	// Set flag to prevent nested chains from re-wrapping
	oldInFlatChain := g.inFlatChain
	g.inFlatChain = true

	// Generate the first Let binding (from the initial Let)
	g.writef("var %s interface{} = ", ToGoVarName(let.Name))
	if err := g.generateExpr(let.Value); err != nil {
		g.inFlatChain = oldInFlatChain
		return err
	}
	g.writef("\n")
	g.writeSuppressUnused(ToGoVarName(let.Name))

	// Generate all chain Let bindings
	for _, chainLet := range chainLets {
		g.writef("var %s interface{} = ", ToGoVarName(chainLet.Name))
		if err := g.generateExpr(chainLet.Value); err != nil {
			g.inFlatChain = oldInFlatChain
			return err
		}
		g.writef("\n")
		g.writeSuppressUnused(ToGoVarName(chainLet.Name))
	}

	// Generate flat if statements for each branch
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

// getExprNodeID extracts the NodeID from a CoreExpr.
// M-DX25.2: Used to look up value expression's type separately from let's type.
func (g *Generator) getExprNodeID(expr core.CoreExpr) uint64 {
	if expr == nil {
		return 0
	}
	return expr.ID()
}
