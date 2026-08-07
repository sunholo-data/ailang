package testing

import (
	"fmt"
	"time"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/eval"
)

// runEnsuresProperty executes an ensures-clause property test.
//
// For each iteration:
//  1. Generate a value for each function parameter (using the existing per-type generators).
//  2. Build a Core harness that calls the function with those values, binds `result`,
//     and evaluates the predicate.
//  3. If the predicate evaluates to false, report a counterexample and stop.
//
// Out of scope here: shrinking the counterexample (existing shrinkCounterexample plumbing
// is wired for forall-binders, not function parameters; follow-up work).
func (r *Runner) runEnsuresProperty(propCase PropertyCase) PropertyResult {
	start := time.Now()

	result := PropertyResult{
		Name:     propCase.Name,
		Location: propCase.Location.String(),
		TestsRun: 0,
		Seed:     r.propertySeed(propCase.Name),
	}

	if propCase.Function == nil {
		result.Status = StatusSkip
		result.SkipKind = SkipKindUnsupported
		result.Error = "ensures property has no function context (top-level ensures not supported)"
		result.Duration = time.Since(start)
		return result
	}

	if r.executor.sourceFile == nil {
		result.Status = StatusFail
		result.Error = "source file not set on executor (call SetSourceFile first)"
		result.Duration = time.Since(start)
		return result
	}

	// Extract the Core function binding (re-uses the inline-tests path).
	// This also caches the elaborated + lowered DeclMeta on the executor.
	binding, err := r.executor.ExtractFunctionBinding(propCase.FunctionCtx, r.executor.sourceFile)
	if err != nil {
		result.Status = StatusFail
		result.Error = fmt.Sprintf("failed to extract function binding for %s: %v", propCase.FunctionCtx, err)
		result.Duration = time.Since(start)
		return result
	}

	// M-DX26 Phase 5.1: Pull the *already-lowered* ensures predicate from Core.Meta
	// instead of converting the surface AST predicate ourselves. This lets arithmetic
	// operators (`+`, `*`) in predicates work — they get rewritten to typed dictionary
	// calls during the standard OpLowering pass that runs over Meta.Contracts.
	loweredPredicate := r.findLoweredContractPredicate(propCase, ast.EnsuresKind, core.EnsuresKind)
	if loweredPredicate == nil {
		result.Status = StatusFail
		result.Error = fmt.Sprintf("could not locate lowered ensures predicate for %s", propCase.FunctionCtx)
		result.Duration = time.Since(start)
		return result
	}

	// Build generators per parameter type. We do NOT use Property.Binders here —
	// ensures has no forall binders; the values flow into the function call.
	params := propCase.Function.Params
	generators := make([]Generator, len(params))
	for i, p := range params {
		gen, _ := r.createGeneratorForType(p.Type)
		if gen == nil {
			result.Status = StatusSkip
			result.SkipKind = SkipKindNoGenerator
			result.Error = fmt.Sprintf("no generator for parameter %s: %v", p.Name, p.Type)
			result.Duration = time.Since(start)
			return result
		}
		generators[i] = gen
	}

	const (
		requiredAccepted = 100
		maxAttempts      = 1000
	)
	rng := newRNG(r.propertySeed(propCase.Name))
	requires := r.findAllLoweredContractPredicates(propCase, core.RequiresKind)

	for result.TestsRun < requiredAccepted && result.GeneratedInputs < maxAttempts {
		result.GeneratedInputs++
		generatedValues := make([]eval.Value, len(generators))
		ensuresParams := make([]EnsuresParam, len(generators))
		for i, gen := range generators {
			v := gen.Generate(rng)
			generatedValues[i] = v
			ensuresParams[i] = EnsuresParam{
				Name:  params[i].Name,
				Value: astExprToCore(r.valueToLiteral(v)),
			}
		}

		requiresHold, err := r.allRequiresHold(ensuresParams, requires)
		if err != nil {
			result.Status = StatusFail
			result.Error = fmt.Sprintf("test %d: %v", result.GeneratedInputs-1, err)
			result.Duration = time.Since(start)
			return result
		}
		if !requiresHold {
			result.DiscardedInputs++
			continue
		}

		result.TestsRun++
		boolValueRaw, err := r.executor.EvaluateEnsuresHarnessFromCore(*binding, ensuresParams, loweredPredicate)
		if err != nil {
			result.Status = StatusFail
			result.Error = fmt.Sprintf("test %d: %v", result.GeneratedInputs-1, err)
			result.Duration = time.Since(start)
			return result
		}

		boolVal, ok := boolValueRaw.(*eval.BoolValue)
		if !ok {
			result.Status = StatusFail
			result.Error = fmt.Sprintf("test %d: ensures predicate must return bool, got %T", result.GeneratedInputs-1, boolValueRaw)
			result.Duration = time.Since(start)
			return result
		}

		if !boolVal.Value {
			result.Status = StatusFail
			result.Error = fmt.Sprintf("ensures violated for input: %s", formatEnsuresInputs(params, generatedValues))
			result.Duration = time.Since(start)
			return result
		}
	}

	if result.TestsRun < requiredAccepted {
		result.Status = StatusSkip
		result.SkipKind = SkipKindOutOfContract
		result.Error = fmt.Sprintf(
			"unverified: requires filter accepted %d of %d generated inputs; need %d (%d discarded)",
			result.TestsRun, result.GeneratedInputs, requiredAccepted, result.DiscardedInputs)
		result.Duration = time.Since(start)
		return result
	}

	result.Status = StatusPass
	result.Duration = time.Since(start)
	return result
}

// findAllLoweredContractPredicates returns every lowered contract predicate of
// coreKind for the property case's function, preserving source order.
func (r *Runner) findAllLoweredContractPredicates(propCase PropertyCase, coreKind core.ContractKind) []core.CoreExpr {
	if propCase.Function == nil {
		return nil
	}
	meta := r.executor.LastDeclMeta(propCase.FunctionCtx)
	if meta == nil {
		return nil
	}

	var predicates []core.CoreExpr
	for _, contract := range meta.Contracts {
		if contract.Kind == coreKind {
			predicates = append(predicates, contract.Expr)
		}
	}
	return predicates
}

// allRequiresHold evaluates requires predicates in source order and stops at
// the first false result. Evaluation failures and non-boolean results fail loud.
func (r *Runner) allRequiresHold(params []EnsuresParam, predicates []core.CoreExpr) (bool, error) {
	for _, predicate := range predicates {
		value, err := r.executor.EvaluateRequiresHarnessFromCore(params, predicate)
		if err != nil {
			return false, fmt.Errorf("requires predicate evaluation failed: %w", err)
		}
		boolValue, ok := value.(*eval.BoolValue)
		if !ok {
			return false, fmt.Errorf("requires predicate must return bool, got %T", value)
		}
		if !boolValue.Value {
			return false, nil
		}
	}
	return true, nil
}
