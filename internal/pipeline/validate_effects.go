package pipeline

import (
	"fmt"
	"os"
	"strings"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/types"
)

var debugEffects = os.Getenv("DEBUG_EFFECTS") == "1"

func debugLog(format string, args ...interface{}) {
	if debugEffects || os.Getenv("DEBUG_EFFECTS") == "1" {
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
// declaredLambdaLookup returns a lambda's original source-declared effect row by
// NodeID (before inference overwrote the CoreTI row), or (nil,false).
type declaredLambdaLookup func(nodeID uint64) (*types.Row, bool)

func ValidateEffects(surfaceAST *ast.File, coreProg *core.Program, coreTypeInfo types.CoreTypeInfo, lambdaLookup ...declaredLambdaLookup) error {
	return validateEffects(surfaceAST, coreProg, coreTypeInfo, nil, nil, lambdaLookup...)
}

func ValidateEffectsWithCalls(surfaceAST *ast.File, coreProg *core.Program, coreTypeInfo types.CoreTypeInfo, maskLookup latentMaskLookup, lambdaLookup declaredLambdaLookup, valueLookup ...effectValueLookup) error {
	if maskLookup == nil {
		return fmt.Errorf("internal invariant: missing LatentParamMask lookup")
	}
	return validateEffects(surfaceAST, coreProg, coreTypeInfo, maskLookup, valueLookup, lambdaLookup)
}

func validateEffects(surfaceAST *ast.File, coreProg *core.Program, coreTypeInfo types.CoreTypeInfo, maskLookup latentMaskLookup, valueLookup []effectValueLookup, lambdaLookup ...declaredLambdaLookup) error {
	// Early return for empty programs
	if len(coreProg.Decls) == 0 {
		return nil
	}

	// Build map of function names to their declared effects from Surface AST
	declaredEffects := make(map[string]*types.Row)
	if surfaceAST != nil {
		for _, funcDecl := range surfaceAST.Funcs {
			row, err := types.ElaborateEffectRowWithBudgets(funcDecl.Effects)
			if err != nil {
				return fmt.Errorf("elaborate effects for function %q: %w", funcDecl.Name, err)
			}
			declaredEffects[funcDecl.Name] = row
		}
	}

	collector := &effectCollector{typeInfo: coreTypeInfo, declared: declaredEffects, maskLookup: maskLookup, invariant: new(error)}
	if len(valueLookup) > 0 {
		collector.valueLookup = valueLookup[0]
	}
	// Walk all top-level declarations
	// Note: Top-level declarations are often wrapped in Let/LetRec nodes
	for _, decl := range coreProg.Decls {
		if err := validateDecl(decl, declaredEffects, coreTypeInfo, collector); err != nil {
			return err
		}
	}

	// M-EFFECT-ROW-SHOW-INTERP (#386): validate INLINE lambda effect annotations.
	// A lambda with an explicit CLOSED effect annotation (e.g. a combinator
	// argument `func(x) -> int ! {} { println(show(x)); x*2 }`) is never reached
	// by validateDecl (it is not a top-level binding), so its declared-vs-actual
	// effects were never checked and an effectful body annotated `! {}` was
	// wrongly accepted. Walk every lambda and reject any whose closed declared
	// effect row omits an effect its body actually requires. Requires the
	// checker's declared-lambda-effect lookup (inference overwrites the CoreTI
	// row); when unavailable (legacy callers) this pass is skipped.
	if len(lambdaLookup) > 0 && lambdaLookup[0] != nil {
		for _, decl := range coreProg.Decls {
			if err := validateLambdaAnnotations(decl, declaredEffects, coreTypeInfo, lambdaLookup[0], collector); err != nil {
				return err
			}
		}
	}

	return *collector.invariant
}

// validateLambdaAnnotations recursively finds inline lambdas with an explicit
// CLOSED effect annotation and rejects any whose body requires an effect the
// annotation does not declare. M-EFFECT-ROW-SHOW-INTERP (#386).
func validateLambdaAnnotations(expr core.CoreExpr, declaredEffects map[string]*types.Row, typeInfo types.CoreTypeInfo, lookup declaredLambdaLookup, collector *effectCollector) error {
	if expr == nil {
		return nil
	}
	switch e := expr.(type) {
	case *core.Lambda:
		// A lambda's declared effect row is the SOURCE-declared row captured by
		// the type checker (inference overwrites the CoreTI row with resolved
		// effects). Only enforce when the source annotation is a CLOSED row (no
		// tail): open/row-polymorphic callbacks subsume extra effects legitimately.
		if declared, ok := lookup(e.ID()); ok && declared != nil && declared.Tail == nil {
			required := eraseGhostEffects(collector.shadow(e.Params...).collect(e.Body))
			if !types.SubsumeEffectRows(required, declared) {
				return formatLambdaEffectError(e.Span().String(), required, declared)
			}
		}
		return validateLambdaAnnotations(e.Body, declaredEffects, typeInfo, lookup, collector.shadow(e.Params...))
	case *core.App:
		if err := validateLambdaAnnotations(e.Func, declaredEffects, typeInfo, lookup, collector); err != nil {
			return err
		}
		for _, a := range e.Args {
			if err := validateLambdaAnnotations(a, declaredEffects, typeInfo, lookup, collector); err != nil {
				return err
			}
		}
	case *core.Let:
		if err := validateLambdaAnnotations(e.Value, declaredEffects, typeInfo, lookup, collector); err != nil {
			return err
		}
		return validateLambdaAnnotations(e.Body, declaredEffects, typeInfo, lookup, collector.shadow(e.Name))
	case *core.LetRec:
		collector = collector.shadow(recBindingNames(e.Bindings)...)
		for _, b := range e.Bindings {
			if err := validateLambdaAnnotations(b.Value, declaredEffects, typeInfo, lookup, collector); err != nil {
				return err
			}
		}
		return validateLambdaAnnotations(e.Body, declaredEffects, typeInfo, lookup, collector)
	case *core.If:
		if err := validateLambdaAnnotations(e.Cond, declaredEffects, typeInfo, lookup, collector); err != nil {
			return err
		}
		if err := validateLambdaAnnotations(e.Then, declaredEffects, typeInfo, lookup, collector); err != nil {
			return err
		}
		return validateLambdaAnnotations(e.Else, declaredEffects, typeInfo, lookup, collector)
	}
	return nil
}

// validateDecl validates a single declaration, handling Let/LetRec specially
func validateDecl(decl core.CoreExpr, declaredEffects map[string]*types.Row, typeInfo types.CoreTypeInfo, collector *effectCollector) error {
	// If this is a LetRec, validate each binding as a separate function
	if letRec, ok := decl.(*core.LetRec); ok {
		for _, binding := range letRec.Bindings {
			debugLog("=== Validating LetRec binding: %s ===", binding.Name)

			// Get declared effects from Surface AST
			declared := declaredEffects[binding.Name]
			debugLog("  Declared effects: %s", formatRow(declared))

			// Collect required effects from Core AST
			required := collector.collect(binding.Value)
			if *collector.invariant != nil {
				return fmt.Errorf("function %s: %w", binding.Name, *collector.invariant)
			}
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
		declared := declaredEffects[let.Name]
		debugLog("  Declared effects: %s", formatRow(declared))

		// Collect required effects from Core AST
		required := collector.collect(let.Value)
		if *collector.invariant != nil {
			return fmt.Errorf("function %s: %w", let.Name, *collector.invariant)
		}
		debugLog("  Required effects: %s", formatRow(required))

		// Ghost effects (Debug) don't need to be declared by callers
		required = eraseGhostEffects(required)
		debugLog("  Required effects (after ghost erasure): %s", formatRow(required))

		if !types.SubsumeEffectRows(required, declared) {
			return formatEffectError(let.Name, required, declared)
		}

		// Also validate the body
		return validateDecl(let.Body, declaredEffects, typeInfo, collector)
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

// extractEffectFromType extracts the effect row from a type
// Handles TFunc2 (function types with effects)
// Normalizes empty effect rows to nil (pure functions)
func extractEffectFromType(t types.Type) *types.Row {
	switch typ := t.(type) {
	case *types.TFunc2:
		// Function type with effects
		row := typ.EffectRow
		// Normalize: empty effect row = nil (pure)
		if row != nil && len(row.Labels) == 0 && row.Tail == nil {
			return nil
		}
		return row
	case *types.TApp:
		// Type application - might be a partially applied function
		// Check if the constructor is a function
		if fn, ok := typ.Constructor.(*types.TFunc2); ok {
			row := fn.EffectRow
			// Normalize: empty effect row = nil (pure)
			if row != nil && len(row.Labels) == 0 && row.Tail == nil {
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
func (collector *effectCollector) collect(expr core.CoreExpr) *types.Row {
	typeInfo, declaredEffects := collector.typeInfo, collector.declared
	if expr == nil {
		return nil
	}

	switch e := expr.(type) {
	case *core.Var:
		// Variables don't introduce effects
		debugLog("    Var(%s) -> []", e.Name)
		return nil

	case *core.VarGlobal:
		// Referring to a function value does not invoke its latent effects.
		return nil

	case *core.Lit:
		// Literals are pure
		return nil

	case *core.Lambda:
		// Lambdas: the effects are in their type, and we need to check the body
		debugLog("    Lambda -> checking body")
		bodyEffects := collector.shadow(e.Params...).collect(e.Body)
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
				calleeEffects = cloneEffectRow(declaredEff)
				usedDeclared = true
				debugLog("      Callee declared effects (from signature): %s", formatRow(calleeEffects))
			}
		}

		// Fall back to CoreTypeInfo only if NOT found in declared effects
		if !usedDeclared {
			if t, ok := collector.effectType(e.Func); ok {
				calleeEffects = extractEffectFromType(t)
				debugLog("      Callee type effects (from CoreTypeInfo): %s", formatRow(calleeEffects))
			}
		}

		// Also recursively check function and arguments for effects
		funcEffects := collector.collect(e.Func)
		debugLog("      Func expr effects: %s", formatRow(funcEffects))

		publication := collector.applicationEffects(e)
		debugLog("      Published App %d call row: %v", e.ID(), publication.CallRow)
		mask := publication.LatentParamMask
		if collector.maskLookup != nil && (!usedDeclared || calleeEffects != nil && calleeEffects.Tail != nil) {
			calleeEffects = cloneEffectRow(publication.CallRow)
			if publication.ImplicitOwnedTail && calleeEffects != nil {
				calleeEffects.Tail = nil
			}
		}
		var argEffects *types.Row
		for i, arg := range e.Args {

			argEff := collector.collect(arg)
			if i < len(mask) && mask[i] {
				var latent *types.Row
				if collector.maskLookup == nil {
					latent = collector.latentEffects(arg)
				} else if i >= len(publication.CallbackRows) || publication.CallbackRows[i] == nil {
					if *collector.invariant == nil {
						*collector.invariant = fmt.Errorf("internal invariant: missing callback row for typed application %d at %s (argument %d)", e.ID(), e.Span(), i)
					}
				} else {
					latent = cloneEffectRow(publication.CallbackRows[i])
					if i < len(publication.ImplicitCallbacks) && publication.ImplicitCallbacks[i] && latent != nil {
						latent.Tail = nil
					}
				}
				argEff = collector.union(argEff, latent)
			}
			debugLog("      Arg[%d] effects: %s", i, formatRow(argEff))
			argEffects = collector.union(argEffects, argEff)
		}
		debugLog("      Combined arg effects: %s", formatRow(argEffects))

		// Union all effects
		result := collector.union(calleeEffects, funcEffects)
		result = collector.union(result, argEffects)
		debugLog("      App total effects: %s", formatRow(result))
		return result

	case *core.Let:
		// Let binding: collect effects from BOTH the RHS value and the body.
		//
		// M-EFFECT-ROW-SHOW-INTERP (#386): block expressions and nested pure calls
		// desugar via ANF into inner lets, e.g. `{ println(show(x)); x*2 }` becomes
		// `let _block_0 = (let $tmp1 = show(x) in println($tmp1)) in x*2`. The IO
		// effect lives in the BODY (`println($tmp1)`) of an inner let, not its
		// value. Previously this case collected only e.Value, so `println`'s IO was
		// silently dropped and the function was wrongly accepted as pure — the
		// exact #386 soundness hole for `println(show(x))`.
		//
		// The M-PERF1 O(m²) concern was about validateDecl walking a chain of
		// TOP-LEVEL decl-lets (it recurses the body itself and passes only e.Value
		// here). That path is unaffected: the value of a top-level function binding
		// is a Lambda, whose body traversal here does not re-enter the outer decl
		// chain. Inner block/ANF lets — which validateDecl never recurses into —
		// require their body to be walked here for soundness.
		debugLog("    Let(%s) -> checking value and body", e.Name)
		valueEffects := collector.collect(e.Value)
		bodyEffects := collector.shadow(e.Name).collect(e.Body)
		effects := collector.union(valueEffects, bodyEffects)
		debugLog("    Let(%s) effects (value ∪ body): %s", e.Name, formatRow(effects))
		return effects

	case *core.LetRec:
		// LetRec: collect from binding values AND the body (see *core.Let above for
		// the #386 rationale on why the body must be traversed).
		debugLog("    LetRec with %d bindings", len(e.Bindings))
		var effects *types.Row
		collector = collector.shadow(recBindingNames(e.Bindings)...)
		for _, binding := range e.Bindings {
			debugLog("      LetRec binding: %s", binding.Name)
			bindingEffects := collector.collect(binding.Value)
			debugLog("      LetRec binding %s effects: %s", binding.Name, formatRow(bindingEffects))
			effects = collector.union(effects, bindingEffects)
		}
		effects = collector.union(effects, collector.collect(e.Body))
		debugLog("    LetRec total effects: %s", formatRow(effects))
		return effects

	case *core.If:
		// If: union of condition, then, and else effects
		condEffects := collector.collect(e.Cond)
		thenEffects := collector.collect(e.Then)
		elseEffects := collector.collect(e.Else)

		result := collector.union(condEffects, thenEffects)
		result = collector.union(result, elseEffects)
		return result

	case *core.Match:
		// Match: union of scrutinee and all arms
		scrutEffects := collector.collect(e.Scrutinee)
		result := scrutEffects

		for _, arm := range e.Arms {
			armEffects := collector.shadow(patternBindingNames(arm.Pattern)...).collect(arm.Body)
			result = collector.union(result, armEffects)
		}
		return result

	case *core.BinOp:
		// Binary operators: union of left and right
		leftEffects := collector.collect(e.Left)
		rightEffects := collector.collect(e.Right)
		return collector.union(leftEffects, rightEffects)

	case *core.UnOp:
		// Unary operators: effects from operand
		return collector.collect(e.Operand)

	case *core.Record:
		// Records: union of all field effects
		var effects *types.Row
		for _, fieldVal := range e.Fields {
			fieldEffects := collector.collect(fieldVal)
			effects = collector.union(effects, fieldEffects)
		}
		return effects

	case *core.RecordAccess:
		// Record access: effects from record expression
		return collector.collect(e.Record)

	case *core.RecordUpdate:
		// Record update: union of base and updated fields
		baseEffects := collector.collect(e.Base)
		var updateEffects *types.Row
		for _, updateVal := range e.Updates {
			fieldEffects := collector.collect(updateVal)
			updateEffects = collector.union(updateEffects, fieldEffects)
		}
		return collector.union(baseEffects, updateEffects)

	case *core.List:
		// Lists: union of all element effects
		var effects *types.Row
		for _, elem := range e.Elements {
			elemEffects := collector.collect(elem)
			effects = collector.union(effects, elemEffects)
		}
		return effects

	case *core.Tuple:
		// Tuples: union of all element effects
		var effects *types.Row
		for _, elem := range e.Elements {
			elemEffects := collector.collect(elem)
			effects = collector.union(effects, elemEffects)
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
		return collector.collect(e.Body)

	case *core.DictApp:
		// Dictionary application: check dict and all argument effects
		dictEffects := collector.collect(e.Dict)
		var argEffects *types.Row
		for _, arg := range e.Args {
			argEff := collector.collect(arg)
			argEffects = collector.union(argEffects, argEff)
		}
		return collector.union(dictEffects, argEffects)

	default:
		// Unknown expression type - be conservative and assume no effects
		return nil
	}
}

// formatEffectError creates a helpful error message for effect violations
func formatEffectError(funcName string, required *types.Row, declared *types.Row) error {
	diff := types.DiffEffectRows(required, declared)
	if len(diff.Missing) == 0 && len(diff.ParamMismatches) == 0 && diff.UnresolvedTail == "" {
		return fmt.Errorf("internal invariant: effect failure for function %s has an empty difference", funcName)
	}

	// Build helpful error message
	var msg strings.Builder
	msg.WriteString(fmt.Sprintf("Effect checking failed for function '%s'\n", funcName))
	msg.WriteString("  Function uses effects not declared in signature\n")
	msg.WriteString("\n")
	writeEffectDiff(&msg, diff)
	msg.WriteString("\n")

	// Show current and suggested signatures
	msg.WriteString(fmt.Sprintf("  Current signature: func %s(...) -> T", funcName))
	if declared != nil && (len(declared.Labels) > 0 || declared.Tail != nil) {
		msg.WriteString(fmt.Sprintf(" %s", types.FormatEffectRow(declared)))
	}
	msg.WriteString("\n")

	// A union can retain the wrong declared mode or render a conflict set,
	// neither of which is a valid fix. Only suggest the label-union case.
	if len(diff.ParamMismatches) == 0 && diff.UnresolvedTail == "" {
		suggestedEffects := types.UnionEffectRows(declared, required)
		msg.WriteString(fmt.Sprintf("  Suggested fix:     func %s(...) -> T %s\n", funcName, types.FormatEffectRow(suggestedEffects)))
	}

	return fmt.Errorf("%s", msg.String())
}

func formatLambdaEffectError(span string, required, declared *types.Row) error {
	var msg strings.Builder
	fmt.Fprintf(&msg, "effect checking failed: lambda at %s uses effects not declared in its %s annotation\n",
		span, types.FormatEffectRow(declared))
	writeEffectDiff(&msg, types.DiffEffectRows(required, declared))
	return fmt.Errorf("%s", msg.String())
}

func writeEffectDiff(msg *strings.Builder, diff types.EffectRowDiff) {
	if diff.UnresolvedTail != "" {
		fmt.Fprintf(msg, "  Unresolved effect tail: %s; propagate the shared row in the function signature\n", diff.UnresolvedTail)
	}
	if len(diff.Missing) > 0 {
		fmt.Fprintf(msg, "  Missing effects: %s\n", strings.Join(diff.Missing, ", "))
	}
	for _, mismatch := range diff.ParamMismatches {
		fmt.Fprintf(msg, "  Effect %s mismatch: %s requires %s; declaration provides %s\n",
			mismatch.Key, mismatch.Effect,
			formatEffectParamValue(mismatch.Effect, mismatch.Key, mismatch.RequiredValue),
			formatEffectParamValue(mismatch.Effect, mismatch.Key, mismatch.DeclaredValue))
	}
}

// formatEffectParamValue renders key=value, or "unscoped <Effect>" when the
// side carries no parameter (e.g. a bare Declassify callee under a scoped caller).
func formatEffectParamValue(effect, key, value string) string {
	if value == "" {
		return "unscoped " + effect
	}
	return key + "=" + value
}
