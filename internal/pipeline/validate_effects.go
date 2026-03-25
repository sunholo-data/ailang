package pipeline

import (
	"fmt"
	"os"
	"strings"

	"github.com/sunholo/ailang/internal/ast"
	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/types"
)

var debugEffects = os.Getenv("DEBUG_EFFECTS") == "1"

func debugLog(format string, args ...interface{}) {
	if debugEffects {
		fmt.Fprintf(os.Stderr, "[DEBUG_EFFECTS] "+format+"\n", args...)
	}
}

// ghostEffects lists effects that are transparent to callers.
// Ghost effects don't propagate through function calls — callers never need
// to declare them. The effect is still enforced at runtime (capability check)
// and can be erased entirely in release mode (--release).
var ghostEffects = map[string]bool{
	"Debug": true,
}

// eraseGhostEffects removes ghost effects (e.g., Debug) from a required effect row.
// Ghost effects don't propagate to callers — they're transparent at the type level.
func eraseGhostEffects(row *types.Row) *types.Row {
	if row == nil {
		return nil
	}
	result := row
	for effect := range ghostEffects {
		result = EraseEffectFromRow(result, effect)
	}
	return result
}

// EraseEffectFromRow removes a named effect from an effect row.
// Returns nil if the row becomes empty (pure).
func EraseEffectFromRow(row *types.Row, effect string) *types.Row {
	if row == nil {
		return nil
	}
	if _, has := row.Labels[effect]; !has {
		return row
	}
	newLabels := make(map[string]types.Type, len(row.Labels)-1)
	for k, v := range row.Labels {
		if k != effect {
			newLabels[k] = v
		}
	}
	// Copy budgets without the erased effect
	var newBudgets map[string]*int
	if row.Budgets != nil {
		newBudgets = make(map[string]*int, len(row.Budgets))
		for k, v := range row.Budgets {
			if k != effect {
				newBudgets[k] = v
			}
		}
		if len(newBudgets) == 0 {
			newBudgets = nil
		}
	}
	var newMinBudgets map[string]*int
	if row.MinBudgets != nil {
		newMinBudgets = make(map[string]*int, len(row.MinBudgets))
		for k, v := range row.MinBudgets {
			if k != effect {
				newMinBudgets[k] = v
			}
		}
		if len(newMinBudgets) == 0 {
			newMinBudgets = nil
		}
	}
	if len(newLabels) == 0 && row.Tail == nil {
		return nil
	}
	return &types.Row{
		Kind:       row.Kind,
		Labels:     newLabels,
		Tail:       row.Tail,
		Budgets:    newBudgets,
		MinBudgets: newMinBudgets,
	}
}

// ValidateEffects validates that functions declare all effects they use
// Compares declared effects from Surface AST with required effects from Core AST
// Returns error if a function uses effects not declared in its signature
// Ghost effects (e.g., Debug) are filtered from required effects — callers
// never need to declare them.
func ValidateEffects(surfaceAST *ast.File, coreProg *core.Program, coreTypeInfo types.CoreTypeInfo) error {
	// Early return for empty programs
	if len(coreProg.Decls) == 0 {
		return nil
	}

	// Build map of function names to their declared effects from Surface AST
	declaredEffects := make(map[string][]string)
	if surfaceAST != nil {
		for _, funcDecl := range surfaceAST.Funcs {
			declaredEffects[funcDecl.Name] = ast.EffectNames(funcDecl.Effects)
		}
	}

	// Walk all top-level declarations
	// Note: Top-level declarations are often wrapped in Let/LetRec nodes
	for _, decl := range coreProg.Decls {
		if err := validateDecl(decl, declaredEffects, coreTypeInfo); err != nil {
			return err
		}
	}

	return nil
}

// validateDecl validates a single declaration, handling Let/LetRec specially
func validateDecl(decl core.CoreExpr, declaredEffects map[string][]string, typeInfo types.CoreTypeInfo) error {
	// If this is a LetRec, validate each binding as a separate function
	if letRec, ok := decl.(*core.LetRec); ok {
		for _, binding := range letRec.Bindings {
			debugLog("=== Validating LetRec binding: %s ===", binding.Name)

			// Get declared effects from Surface AST
			surfaceDeclaredEffects := declaredEffects[binding.Name]
			declared := stringSliceToEffectRow(surfaceDeclaredEffects)
			debugLog("  Declared effects: %v", surfaceDeclaredEffects)

			// Collect required effects from Core AST
			required := collectRequiredEffects(binding.Value, typeInfo, declaredEffects)
			debugLog("  Required effects: %s", formatRow(required))

			// Ghost effects (Debug) don't need to be declared by callers
			required = eraseGhostEffects(required)
			debugLog("  Required effects (after ghost erasure): %s", formatRow(required))

			if !types.SubsumeEffectRows(required, declared) {
				return formatEffectError(binding.Name, required, declared)
			}
		}
		return nil
	}

	// If this is a Let, validate the binding and recurse on body
	if let, ok := decl.(*core.Let); ok {
		debugLog("=== Validating Let binding: %s ===", let.Name)

		// Get declared effects from Surface AST
		surfaceDeclaredEffects := declaredEffects[let.Name]
		declared := stringSliceToEffectRow(surfaceDeclaredEffects)
		debugLog("  Declared effects: %v", surfaceDeclaredEffects)

		// Collect required effects from Core AST
		required := collectRequiredEffects(let.Value, typeInfo, declaredEffects)
		debugLog("  Required effects: %s", formatRow(required))

		// Ghost effects (Debug) don't need to be declared by callers
		required = eraseGhostEffects(required)
		debugLog("  Required effects (after ghost erasure): %s", formatRow(required))

		if !types.SubsumeEffectRows(required, declared) {
			return formatEffectError(let.Name, required, declared)
		}

		// Also validate the body
		return validateDecl(let.Body, declaredEffects, typeInfo)
	}

	// For other declarations (shouldn't happen in normal flow)
	return nil
}

func formatRow(r *types.Row) string {
	if r == nil {
		return "[]"
	}
	labels := make([]string, 0, len(r.Labels))
	for k := range r.Labels {
		labels = append(labels, k)
	}
	return fmt.Sprintf("[%s]", strings.Join(labels, ", "))
}

// stringSliceToEffectRow converts a slice of effect label strings to an effect row
// Returns nil for empty slice (pure function)
func stringSliceToEffectRow(effects []string) *types.Row {
	if len(effects) == 0 {
		return nil // Pure (no effects)
	}

	labels := make(map[string]types.Type)
	for _, effect := range effects {
		labels[effect] = &types.TCon{Name: effect} // Effect labels map to their names as types
	}

	return &types.Row{
		Kind:   types.KRow{ElemKind: types.KEffect{}},
		Labels: labels,
		Tail:   nil, // Closed row (no extension)
	}
}

// extractEffectFromType extracts the effect row from a type
// Handles TFunc2 (function types with effects)
// Normalizes empty effect rows to nil (pure functions)
func extractEffectFromType(t types.Type) *types.Row {
	switch typ := t.(type) {
	case *types.TFunc2:
		// Function type with effects
		row := typ.EffectRow
		// Normalize: empty effect row = nil (pure)
		if row != nil && len(row.Labels) == 0 {
			return nil
		}
		return row
	case *types.TApp:
		// Type application - might be a partially applied function
		// Check if the constructor is a function
		if fn, ok := typ.Constructor.(*types.TFunc2); ok {
			row := fn.EffectRow
			// Normalize: empty effect row = nil (pure)
			if row != nil && len(row.Labels) == 0 {
				return nil
			}
			return row
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
// declaredEffects maps function names to their declared effect signatures from Surface AST
func collectRequiredEffects(expr core.CoreExpr, typeInfo types.CoreTypeInfo, declaredEffects map[string][]string) *types.Row {
	if expr == nil {
		return nil
	}

	switch e := expr.(type) {
	case *core.Var:
		// Variables don't introduce effects
		debugLog("    Var(%s) -> []", e.Name)
		return nil

	case *core.VarGlobal:
		// Global variables might have effects (e.g., builtins)
		if t, ok := typeInfo.Get(e.ID()); ok {
			eff := extractEffectFromType(t)
			debugLog("    VarGlobal(%s.%s) -> %s", e.Ref.Module, e.Ref.Name, formatRow(eff))
			return eff
		}
		debugLog("    VarGlobal(%s.%s) -> [] (no type info)", e.Ref.Module, e.Ref.Name)
		return nil

	case *core.Lit:
		// Literals are pure
		return nil

	case *core.Lambda:
		// Lambdas: the effects are in their type, and we need to check the body
		debugLog("    Lambda -> checking body")
		bodyEffects := collectRequiredEffects(e.Body, typeInfo, declaredEffects)
		debugLog("    Lambda body effects: %s", formatRow(bodyEffects))
		return bodyEffects

	case *core.App:
		// Function application: the key case for effect checking!
		debugLog("    App (function application)")

		// Get the type of the function being called
		// FIX: First check if this is a call to a known function with declared effects
		// Use declared effects instead of CoreTypeInfo to avoid contamination
		var calleeEffects *types.Row
		usedDeclared := false
		if funcVar, ok := e.Func.(*core.Var); ok {
			// This is a call to a locally-bound function - use declared effects if available
			if declaredEff, found := declaredEffects[funcVar.Name]; found {
				calleeEffects = stringSliceToEffectRow(declaredEff)
				usedDeclared = true
				debugLog("      Callee declared effects (from signature): %s", formatRow(calleeEffects))
			}
		}

		// Fall back to CoreTypeInfo only if NOT found in declared effects
		if !usedDeclared {
			if t, ok := typeInfo.Get(e.Func.ID()); ok {
				calleeEffects = extractEffectFromType(t)
				debugLog("      Callee type effects (from CoreTypeInfo): %s", formatRow(calleeEffects))
			}
		}

		// Also recursively check function and arguments for effects
		funcEffects := collectRequiredEffects(e.Func, typeInfo, declaredEffects)
		debugLog("      Func expr effects: %s", formatRow(funcEffects))

		var argEffects *types.Row
		for i, arg := range e.Args {
			argEff := collectRequiredEffects(arg, typeInfo, declaredEffects)
			debugLog("      Arg[%d] effects: %s", i, formatRow(argEff))
			argEffects = types.UnionEffectRows(argEffects, argEff)
		}
		debugLog("      Combined arg effects: %s", formatRow(argEffects))

		// Union all effects
		result := types.UnionEffectRows(calleeEffects, funcEffects)
		result = types.UnionEffectRows(result, argEffects)
		debugLog("      App total effects: %s", formatRow(result))
		return result

	case *core.Let:
		// Let binding: only collect from RHS value
		// IMPORTANT: Do NOT traverse body here - validateDecl already handles body recursion
		// Traversing body here caused O(m²) complexity where m = number of Let bindings
		// (M-PERF1 fix for effect checker hang on large arrays)
		debugLog("    Let(%s) -> checking value only", e.Name)
		valueEffects := collectRequiredEffects(e.Value, typeInfo, declaredEffects)
		debugLog("    Let(%s) value effects: %s", e.Name, formatRow(valueEffects))
		return valueEffects

	case *core.LetRec:
		// LetRec: only collect from binding values
		// IMPORTANT: Do NOT traverse body here - validateDecl already handles body recursion
		// (M-PERF1 fix for effect checker hang on large arrays)
		debugLog("    LetRec with %d bindings", len(e.Bindings))
		var effects *types.Row
		for _, binding := range e.Bindings {
			debugLog("      LetRec binding: %s", binding.Name)
			bindingEffects := collectRequiredEffects(binding.Value, typeInfo, declaredEffects)
			debugLog("      LetRec binding %s effects: %s", binding.Name, formatRow(bindingEffects))
			effects = types.UnionEffectRows(effects, bindingEffects)
		}
		debugLog("    LetRec total effects: %s", formatRow(effects))
		return effects

	case *core.If:
		// If: union of condition, then, and else effects
		condEffects := collectRequiredEffects(e.Cond, typeInfo, declaredEffects)
		thenEffects := collectRequiredEffects(e.Then, typeInfo, declaredEffects)
		elseEffects := collectRequiredEffects(e.Else, typeInfo, declaredEffects)

		result := types.UnionEffectRows(condEffects, thenEffects)
		result = types.UnionEffectRows(result, elseEffects)
		return result

	case *core.Match:
		// Match: union of scrutinee and all arms
		scrutEffects := collectRequiredEffects(e.Scrutinee, typeInfo, declaredEffects)
		result := scrutEffects

		for _, arm := range e.Arms {
			armEffects := collectRequiredEffects(arm.Body, typeInfo, declaredEffects)
			result = types.UnionEffectRows(result, armEffects)
		}
		return result

	case *core.BinOp:
		// Binary operators: union of left and right
		leftEffects := collectRequiredEffects(e.Left, typeInfo, declaredEffects)
		rightEffects := collectRequiredEffects(e.Right, typeInfo, declaredEffects)
		return types.UnionEffectRows(leftEffects, rightEffects)

	case *core.UnOp:
		// Unary operators: effects from operand
		return collectRequiredEffects(e.Operand, typeInfo, declaredEffects)

	case *core.Record:
		// Records: union of all field effects
		var effects *types.Row
		for _, fieldVal := range e.Fields {
			fieldEffects := collectRequiredEffects(fieldVal, typeInfo, declaredEffects)
			effects = types.UnionEffectRows(effects, fieldEffects)
		}
		return effects

	case *core.RecordAccess:
		// Record access: effects from record expression
		return collectRequiredEffects(e.Record, typeInfo, declaredEffects)

	case *core.RecordUpdate:
		// Record update: union of base and updated fields
		baseEffects := collectRequiredEffects(e.Base, typeInfo, declaredEffects)
		var updateEffects *types.Row
		for _, updateVal := range e.Updates {
			fieldEffects := collectRequiredEffects(updateVal, typeInfo, declaredEffects)
			updateEffects = types.UnionEffectRows(updateEffects, fieldEffects)
		}
		return types.UnionEffectRows(baseEffects, updateEffects)

	case *core.List:
		// Lists: union of all element effects
		var effects *types.Row
		for _, elem := range e.Elements {
			elemEffects := collectRequiredEffects(elem, typeInfo, declaredEffects)
			effects = types.UnionEffectRows(effects, elemEffects)
		}
		return effects

	case *core.Tuple:
		// Tuples: union of all element effects
		var effects *types.Row
		for _, elem := range e.Elements {
			elemEffects := collectRequiredEffects(elem, typeInfo, declaredEffects)
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
		return collectRequiredEffects(e.Body, typeInfo, declaredEffects)

	case *core.DictApp:
		// Dictionary application: check dict and all argument effects
		dictEffects := collectRequiredEffects(e.Dict, typeInfo, declaredEffects)
		var argEffects *types.Row
		for _, arg := range e.Args {
			argEff := collectRequiredEffects(arg, typeInfo, declaredEffects)
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
	msg.WriteString("  Function uses effects not declared in signature\n")
	msg.WriteString("\n")
	msg.WriteString(fmt.Sprintf("  Missing effects: %s\n", strings.Join(missing, ", ")))
	msg.WriteString("\n")

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
