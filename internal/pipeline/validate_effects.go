package pipeline

import (
	"fmt"
	"strings"

	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/types"
)

// ValidateEffects validates that functions declare all effects they use
// Returns error if a function uses effects not declared in its signature
func ValidateEffects(coreProg *core.Program, coreTypeInfo types.CoreTypeInfo) error {
	for _, decl := range coreProg.Decls {
		// Get the name from metadata (if available)
		name := getDeclName(decl, coreProg.Meta)

		// Extract declared effects from function signature
		declared := extractDeclaredEffects(decl, coreTypeInfo)

		// Collect required effects from function body
		required := collectRequiredEffects(decl, coreTypeInfo)

		// Check if required effects are subsumed by declared effects
		if !types.SubsumeEffectRows(required, declared) {
			return formatEffectError(name, required, declared)
		}
	}

	return nil
}

// getDeclName extracts the name of a declaration from metadata
func getDeclName(decl core.CoreExpr, meta map[string]*core.DeclMeta) string {
	// Try to find metadata by searching for a matching declaration
	for name, m := range meta {
		// Match by checking if the decl is a lambda with this name
		// (This is a heuristic; in practice, metadata is keyed by name)
		_ = m
		return name // Return first name found (TODO: better matching)
	}
	return "<unknown>"
}

// extractDeclaredEffects extracts the effect row from a function's type signature
// Returns nil for pure functions (no effects declared)
func extractDeclaredEffects(decl core.CoreExpr, typeInfo types.CoreTypeInfo) *types.Row {
	funcType, ok := typeInfo.Get(decl.ID())
	if !ok {
		// No type info - shouldn't happen if CoreTypeInfo validation passed
		return nil
	}

	// Extract effect row from function type
	return extractEffectFromType(funcType)
}

// extractEffectFromType extracts the effect row from a type
// Handles TFunc2 (function types with effects)
func extractEffectFromType(t types.Type) *types.Row {
	switch typ := t.(type) {
	case *types.TFunc2:
		// Function type with effects
		return typ.EffectRow
	case *types.TApp:
		// Type application - might be a partially applied function
		// Check if the constructor is a function
		if fn, ok := typ.Constructor.(*types.TFunc2); ok {
			return fn.EffectRow
		}
		// Also check arguments recursively
		for _, arg := range typ.Args {
			if row := extractEffectFromType(arg); row != nil {
				return row
			}
		}
	}

	return nil // Pure (no effects)
}

// collectRequiredEffects recursively walks the expression to collect all required effects
// Returns the union of all effects used in the expression
func collectRequiredEffects(expr core.CoreExpr, typeInfo types.CoreTypeInfo) *types.Row {
	if expr == nil {
		return nil
	}

	switch e := expr.(type) {
	case *core.Var:
		// Variables don't introduce effects
		return nil

	case *core.VarGlobal:
		// Global variables might have effects (e.g., builtins)
		if t, ok := typeInfo.Get(e.ID()); ok {
			return extractEffectFromType(t)
		}
		return nil

	case *core.Lit:
		// Literals are pure
		return nil

	case *core.Lambda:
		// Lambdas: the effects are in their type, and we need to check the body
		bodyEffects := collectRequiredEffects(e.Body, typeInfo)
		return bodyEffects

	case *core.App:
		// Function application: the key case for effect checking!
		// Get the type of the function being called
		var calleeEffects *types.Row
		if t, ok := typeInfo.Get(e.Func.ID()); ok {
			calleeEffects = extractEffectFromType(t)
		}

		// Also recursively check function and arguments for effects
		funcEffects := collectRequiredEffects(e.Func, typeInfo)
		var argEffects *types.Row
		for _, arg := range e.Args {
			argEff := collectRequiredEffects(arg, typeInfo)
			argEffects = types.UnionEffectRows(argEffects, argEff)
		}

		// Union all effects
		result := types.UnionEffectRows(calleeEffects, funcEffects)
		result = types.UnionEffectRows(result, argEffects)
		return result

	case *core.Let:
		// Let binding: union of RHS and body effects
		rhsEffects := collectRequiredEffects(e.Value, typeInfo)
		bodyEffects := collectRequiredEffects(e.Body, typeInfo)
		return types.UnionEffectRows(rhsEffects, bodyEffects)

	case *core.LetRec:
		// LetRec: union of all binding effects and body
		var effects *types.Row
		for _, binding := range e.Bindings {
			bindingEffects := collectRequiredEffects(binding.Value, typeInfo)
			effects = types.UnionEffectRows(effects, bindingEffects)
		}
		bodyEffects := collectRequiredEffects(e.Body, typeInfo)
		return types.UnionEffectRows(effects, bodyEffects)

	case *core.If:
		// If: union of condition, then, and else effects
		condEffects := collectRequiredEffects(e.Cond, typeInfo)
		thenEffects := collectRequiredEffects(e.Then, typeInfo)
		elseEffects := collectRequiredEffects(e.Else, typeInfo)

		result := types.UnionEffectRows(condEffects, thenEffects)
		result = types.UnionEffectRows(result, elseEffects)
		return result

	case *core.Match:
		// Match: union of scrutinee and all arms
		scrutEffects := collectRequiredEffects(e.Scrutinee, typeInfo)
		result := scrutEffects

		for _, arm := range e.Arms {
			armEffects := collectRequiredEffects(arm.Body, typeInfo)
			result = types.UnionEffectRows(result, armEffects)
		}
		return result

	case *core.BinOp:
		// Binary operators: union of left and right
		leftEffects := collectRequiredEffects(e.Left, typeInfo)
		rightEffects := collectRequiredEffects(e.Right, typeInfo)
		return types.UnionEffectRows(leftEffects, rightEffects)

	case *core.UnOp:
		// Unary operators: effects from operand
		return collectRequiredEffects(e.Operand, typeInfo)

	case *core.Record:
		// Records: union of all field effects
		var effects *types.Row
		for _, fieldVal := range e.Fields {
			fieldEffects := collectRequiredEffects(fieldVal, typeInfo)
			effects = types.UnionEffectRows(effects, fieldEffects)
		}
		return effects

	case *core.RecordAccess:
		// Record access: effects from record expression
		return collectRequiredEffects(e.Record, typeInfo)

	case *core.RecordUpdate:
		// Record update: union of base and updated fields
		baseEffects := collectRequiredEffects(e.Base, typeInfo)
		var updateEffects *types.Row
		for _, updateVal := range e.Updates {
			fieldEffects := collectRequiredEffects(updateVal, typeInfo)
			updateEffects = types.UnionEffectRows(updateEffects, fieldEffects)
		}
		return types.UnionEffectRows(baseEffects, updateEffects)

	case *core.List:
		// Lists: union of all element effects
		var effects *types.Row
		for _, elem := range e.Elements {
			elemEffects := collectRequiredEffects(elem, typeInfo)
			effects = types.UnionEffectRows(effects, elemEffects)
		}
		return effects

	case *core.Tuple:
		// Tuples: union of all element effects
		var effects *types.Row
		for _, elem := range e.Elements {
			elemEffects := collectRequiredEffects(elem, typeInfo)
			effects = types.UnionEffectRows(effects, elemEffects)
		}
		return effects

	case *core.Intrinsic:
		// Intrinsics might have effects (check their type)
		if t, ok := typeInfo.Get(e.ID()); ok {
			return extractEffectFromType(t)
		}
		return nil

	case *core.DictAbs:
		// Dictionary abstraction: check body effects
		return collectRequiredEffects(e.Body, typeInfo)

	case *core.DictApp:
		// Dictionary application: check dict and all argument effects
		dictEffects := collectRequiredEffects(e.Dict, typeInfo)
		var argEffects *types.Row
		for _, arg := range e.Args {
			argEff := collectRequiredEffects(arg, typeInfo)
			argEffects = types.UnionEffectRows(argEffects, argEff)
		}
		return types.UnionEffectRows(dictEffects, argEffects)

	default:
		// Unknown expression type - be conservative and assume no effects
		return nil
	}
}

// formatEffectError creates a helpful error message for effect violations
func formatEffectError(funcName string, required *types.Row, declared *types.Row) error {
	// Find which effects are missing
	missing := types.EffectRowDifference(required, declared)

	if len(missing) == 0 {
		// Shouldn't happen if SubsumeEffectRows returned false
		return fmt.Errorf("effect checking failed for function %s (no specific missing effects identified)", funcName)
	}

	// Build helpful error message
	var msg strings.Builder
	msg.WriteString(fmt.Sprintf("Effect checking failed for function '%s'\n", funcName))
	msg.WriteString(fmt.Sprintf("  Function uses effects not declared in signature\n"))
	msg.WriteString(fmt.Sprintf("\n"))
	msg.WriteString(fmt.Sprintf("  Missing effects: %s\n", strings.Join(missing, ", ")))
	msg.WriteString(fmt.Sprintf("\n"))

	// Show current and suggested signatures
	msg.WriteString(fmt.Sprintf("  Current signature: func %s(...) -> T", funcName))
	if declared != nil && len(declared.Labels) > 0 {
		msg.WriteString(fmt.Sprintf(" %s", types.FormatEffectRow(declared)))
	}
	msg.WriteString("\n")

	// Suggest fix
	suggestedEffects := types.UnionEffectRows(declared, required)
	msg.WriteString(fmt.Sprintf("  Suggested fix:     func %s(...) -> T %s", funcName, types.FormatEffectRow(suggestedEffects)))
	msg.WriteString("\n")

	return fmt.Errorf("%s", msg.String())
}
