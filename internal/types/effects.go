package types

import (
	"fmt"
	"sort"
	"strings"

	"github.com/sunholo-data/ailang/internal/ast"
)

// isEffectRowVar returns true if name is a lowercase identifier (effect row variable)
func isEffectRowVar(name string) bool {
	return len(name) > 0 && name[0] >= 'a' && name[0] <= 'z'
}

// defaultEffectModes is the per-effect default-mode lookup table.
// When a bare effect (no params) is elaborated, if its name has an entry
// here, the elaborator desugars to the parameterised form.
//
// Phase 1 (v0.15.0) ships only the Rand entry. Other effects (Clock,
// Net, FS, AI) intentionally have no entry — their bare forms continue
// to type-check unchanged (back-compat). Their port sprints add rows
// here in their respective milestones.
//
// This is intentional, not a fallback (per CLAUDE.md no-silent-fallbacks).
// Effects without entries stay bare; they don't silently get a default.
var defaultEffectModes = map[string]struct{ Key, Value string }{
	"Rand": {Key: "mode", Value: "os"},
	// Future:
	// "Clock": {Key: "mode", Value: "wall"},
	// "Net":   {Key: "mode", Value: "live"},
	// "FS":    {Key: "mode", Value: "real"},
	// "AI":    {Key: "mode", Value: "fixed"},
}

// DefaultModeFor returns the default mode key=value for an effect, if one is registered.
// Returns ("", "", false) for effects without a registered default.
//
// Used during effect-row elaboration: bare !{Rand} desugars to !{Rand[mode=os]}
// because Rand has a registered default. Bare !{IO} stays bare because IO has none.
func DefaultModeFor(effectName string) (key, value string, ok bool) {
	if e, found := defaultEffectModes[effectName]; found {
		return e.Key, e.Value, true
	}
	return "", "", false
}

// paramsOf returns the param map for an effect in a row, or nil if none.
// Helper for invariant unification of parameterised effects.
func paramsOf(r *Row, effectName string) map[string]string {
	if r == nil || r.Params == nil {
		return nil
	}
	return r.Params[effectName]
}

// effectiveParamsOf returns the effective param map for an effect: explicitly
// stored params if present, otherwise the registered default for that effect
// (via DefaultModeFor), otherwise nil.
//
// This is the comparison-time normalisation used by paramsEqualForEffect and
// effectParamsCompatible. Two rows that only differ by "explicit Rand[mode=os]
// vs nil Rand" are considered identical because both desugar to the same
// effective params.
//
// Without this, rows built outside the elaborator (e.g. via
// stringSliceToEffectRow in pipeline/validate_effects.go) would be incompatible
// with rows built inside the elaborator (which applies defaults). Phase 1 ships
// only the Rand default; effects without entries return nil unchanged.
func effectiveParamsOf(r *Row, effectName string) map[string]string {
	if p := paramsOf(r, effectName); len(p) > 0 {
		return p
	}
	if k, v, ok := DefaultModeFor(effectName); ok {
		return map[string]string{k: v}
	}
	return nil
}

// paramMapsEqual compares two effect-parameter maps for invariant unification.
// nil and empty maps are treated as equivalent.
//
// This is a low-level comparison used by tests and directly when you have two
// raw param maps. For comparing effect rows during unification or subsumption,
// use effectParamsCompatible (which applies the default-mode normalisation).
func paramMapsEqual(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if bv, ok := b[k]; !ok || v != bv {
			return false
		}
	}
	return true
}

// effectParamsCompatible compares per-effect params between two rows, normalising
// each side via effectiveParamsOf so that "explicit default" and "implicit
// default (nil)" are treated as equal. Two non-default values still fail
// (invariant), as does default vs explicit non-default.
func effectParamsCompatible(a, b *Row, effectName string) bool {
	return paramMapsEqual(effectiveParamsOf(a, effectName), effectiveParamsOf(b, effectName))
}

// applyDefaultParam populates params[effectName] from DefaultModeFor if no
// explicit params were supplied for that effect. No-op for effects without
// a registered default. Mutates the params map in place.
func applyDefaultParam(params map[string]map[string]string, effectName string) {
	if params == nil {
		return
	}
	if _, alreadySet := params[effectName]; alreadySet {
		return
	}
	if k, v, ok := DefaultModeFor(effectName); ok {
		params[effectName] = map[string]string{k: v}
	}
}

// ElaborateEffectRow converts AST effect names to a normalized effect row
// Returns nil for empty effect sets (purity sentinel)
// Labels are sorted alphabetically for determinism
// Supports effect row variables (lowercase identifiers like 'e') for polymorphism
func ElaborateEffectRow(effectNames []string) (*Row, error) {
	if len(effectNames) == 0 {
		return nil, nil // Purity sentinel
	}

	// Validate and collect effects, separating row variables from concrete effects
	validatedEffects := make(map[string]bool)
	var rowVarName string
	for _, name := range effectNames {
		if isEffectRowVar(name) {
			if rowVarName != "" && rowVarName != name {
				return nil, fmt.Errorf("multiple effect row variables: %s, %s", rowVarName, name)
			}
			rowVarName = name
			continue
		}
		// Validate against known effects
		if !IsKnownEffect(name) {
			return nil, fmt.Errorf("unknown effect: %s", name)
		}
		validatedEffects[name] = true
	}

	// Convert to sorted slice for determinism
	sortedNames := make([]string, 0, len(validatedEffects))
	for name := range validatedEffects {
		sortedNames = append(sortedNames, name)
	}
	sort.Strings(sortedNames)

	// Build effect row with sorted labels
	labels := make(map[string]Type)
	for _, name := range sortedNames {
		// Each effect maps to Unit type (effects are just labels)
		labels[name] = Unit()
	}

	// Apply default-mode desugar for bare effects with registered defaults
	// (M-EFFECT-REFINEMENT Phase 1: Rand→{mode=os}). Effects without an
	// entry in defaultEffectModes get no entry here (back-compat).
	params := make(map[string]map[string]string)
	for _, name := range sortedNames {
		applyDefaultParam(params, name)
	}
	if len(params) == 0 {
		params = nil
	}

	// Set row tail for effect polymorphism
	var tail *RowVar
	if rowVarName != "" {
		tail = &RowVar{Name: rowVarName, Kind: EffectRow}
	}

	return &Row{
		Kind:   EffectRow,
		Labels: labels,
		Tail:   tail,
		Params: params,
	}, nil
}

// ElaborateEffectRowWithBudgets converts AST effect annotations (with optional budgets)
// to a normalized effect row. Returns nil for empty effect sets (purity sentinel).
// Labels are sorted alphabetically for determinism.
// Supports @limit=N (max) and @min=N (minimum) annotations (M-DX25 M4).
// Supports effect row variables (lowercase identifiers like 'e') for polymorphism.
func ElaborateEffectRowWithBudgets(effects []ast.EffectAnnotation) (*Row, error) {
	if len(effects) == 0 {
		return nil, nil // Purity sentinel
	}

	// Validate and collect effects with budgets, separating row variables
	validatedEffects := make(map[string]bool)
	budgets := make(map[string]*int)
	minBudgets := make(map[string]*int)
	// Per-effect parameter map (M-EFFECT-REFINEMENT Phase 1).
	// Populated explicitly from eff.Params; defaults applied below for
	// bare effects with registered DefaultModeFor entries.
	params := make(map[string]map[string]string)
	var rowVarName string

	for _, eff := range effects {
		if eff.IsRowVar || isEffectRowVar(eff.Name) {
			if rowVarName != "" && rowVarName != eff.Name {
				return nil, fmt.Errorf("multiple effect row variables: %s, %s", rowVarName, eff.Name)
			}
			rowVarName = eff.Name
			continue
		}
		// Validate against known effects
		if !IsKnownEffect(eff.Name) {
			return nil, fmt.Errorf("unknown effect: %s", eff.Name)
		}
		validatedEffects[eff.Name] = true
		if eff.Budget != nil {
			// Copy budget value to avoid pointer aliasing
			val := *eff.Budget
			budgets[eff.Name] = &val
		}
		if eff.Min != nil {
			// Copy min value to avoid pointer aliasing
			val := *eff.Min
			minBudgets[eff.Name] = &val
		}
		// Capture explicit user-supplied params; user wins over default.
		if len(eff.Params) > 0 {
			pmap := make(map[string]string, len(eff.Params))
			for _, p := range eff.Params {
				pmap[p.Key] = p.Value
			}
			params[eff.Name] = pmap
		}
	}

	// Convert to sorted slice for determinism
	sortedNames := make([]string, 0, len(validatedEffects))
	for name := range validatedEffects {
		sortedNames = append(sortedNames, name)
	}
	sort.Strings(sortedNames)

	// Build effect row with sorted labels
	labels := make(map[string]Type)
	for _, name := range sortedNames {
		// Each effect maps to Unit type (effects are just labels)
		labels[name] = Unit()
	}

	// Only include budgets/minBudgets maps if there are any values
	var budgetsMap map[string]*int
	if len(budgets) > 0 {
		budgetsMap = budgets
	}
	var minBudgetsMap map[string]*int
	if len(minBudgets) > 0 {
		minBudgetsMap = minBudgets
	}

	// Apply default-mode desugar for bare effects with registered defaults.
	// User-supplied params (already in params map) win over defaults.
	for _, name := range sortedNames {
		applyDefaultParam(params, name)
	}
	var paramsMap map[string]map[string]string
	if len(params) > 0 {
		paramsMap = params
	}

	// Set row tail for effect polymorphism
	var tail *RowVar
	if rowVarName != "" {
		tail = &RowVar{Name: rowVarName, Kind: EffectRow}
	}

	return &Row{
		Kind:       EffectRow,
		Labels:     labels,
		Tail:       tail,
		Budgets:    budgetsMap,
		MinBudgets: minBudgetsMap,
		Params:     paramsMap,
	}, nil
}

// IsKnownEffect checks if an effect name is one of the canonical effects
func IsKnownEffect(name string) bool {
	knownEffects := map[string]bool{
		"IO":          true,
		"FS":          true,
		"Net":         true,
		"Clock":       true,
		"Rand":        true,
		"DB":          true,
		"Trace":       true,
		"Async":       true,
		"Env":         true, // v0.4.0: Environment variable access
		"Debug":       true, // v0.4.10: Structured tracing/assertions (ghost effect)
		"AI":          true, // v0.5.1: General-purpose AI oracle
		"SharedMem":   true, // v0.5.11: Shared memory cache effect (M-DX15)
		"SharedIndex": true, // v0.5.11: Similarity index for semantic retrieval (M-DX16)
		"Stream":      true, // v0.8.1: Bidirectional WebSocket streaming (M-STREAM-BIDI)
		"Process":     true, // v0.8.0: External command execution (M-PROCESS)
		"Declassify":  true, // v0.16.0: IFC declassification capability (M-TAINT-TYPES)
	}
	return knownEffects[name]
}

// Unit returns the Unit type
func Unit() Type {
	return &TCon{Name: "()"}
}

// UnionEffectRows creates the union of two effect rows
// nil is treated as the identity (pure)
// Result is always sorted for determinism
// For budgets: when both rows have a budget for the same effect, budgets are summed
// (nested scopes compose by addition for total allowed invocations)
func UnionEffectRows(a, b *Row) *Row {
	if a == nil {
		return b
	}
	if b == nil {
		return a
	}

	// Merge labels
	merged := make(map[string]struct{})
	for k := range a.Labels {
		merged[k] = struct{}{}
	}
	for k := range b.Labels {
		merged[k] = struct{}{}
	}

	// Convert to sorted slice
	labels := make([]string, 0, len(merged))
	for k := range merged {
		labels = append(labels, k)
	}
	sort.Strings(labels)

	// Build effect row
	effectLabels := make(map[string]Type)
	for _, name := range labels {
		effectLabels[name] = Unit()
	}

	// Merge budgets - sum budgets when both have them
	var budgets map[string]*int
	if a.Budgets != nil || b.Budgets != nil {
		budgets = make(map[string]*int)
		for _, name := range labels {
			var aBudget, bBudget *int
			if a.Budgets != nil {
				aBudget = a.Budgets[name]
			}
			if b.Budgets != nil {
				bBudget = b.Budgets[name]
			}

			// Compose budgets: sum if both present, take available otherwise
			if aBudget != nil && bBudget != nil {
				sum := *aBudget + *bBudget
				budgets[name] = &sum
			} else if aBudget != nil {
				val := *aBudget
				budgets[name] = &val
			} else if bBudget != nil {
				val := *bBudget
				budgets[name] = &val
			}
			// If neither has budget, the effect is unlimited (no entry in budgets map)
		}
		// Clean up empty budgets map
		if len(budgets) == 0 {
			budgets = nil
		}
	}

	// Merge per-effect params (M-EFFECT-REFINEMENT Phase 1).
	// When both rows specify params for the same effect, they should already
	// be equal (the unifier enforces invariance upstream). If they disagree
	// here we conservatively prefer 'a' and continue; this matches the
	// pragmatic union semantics used by the rest of the row algebra and
	// keeps the function total. A mismatch should have been caught by
	// unifyRows before reaching union.
	var params map[string]map[string]string
	if a.Params != nil || b.Params != nil {
		params = make(map[string]map[string]string)
		for _, name := range labels {
			ap := paramsOf(a, name)
			bp := paramsOf(b, name)
			switch {
			case len(ap) > 0:
				cp := make(map[string]string, len(ap))
				for k, v := range ap {
					cp[k] = v
				}
				params[name] = cp
			case len(bp) > 0:
				cp := make(map[string]string, len(bp))
				for k, v := range bp {
					cp[k] = v
				}
				params[name] = cp
			}
		}
		if len(params) == 0 {
			params = nil
		}
	}

	// If no effects after merging, return nil (pure)
	if len(effectLabels) == 0 {
		return nil
	}

	return &Row{
		Kind:    EffectRow,
		Labels:  effectLabels,
		Tail:    nil,
		Budgets: budgets,
		Params:  params,
	}
}

// SubsumeEffectRows checks if effect row 'a' is subsumed by effect row 'b'
// Returns true if all effects in 'a' are present in 'b'
// Pure (nil) is subsumed by anything
//
// Phase 1 (M-EFFECT-REFINEMENT): for parameterised effects the param map
// is invariant — !{Rand[mode=os]} is NOT subsumed by !{Rand[mode=seeded]}.
// This is required so the routeable→fixed example becomes a typecheck
// rejection in Phase 5.
func SubsumeEffectRows(a, b *Row) bool {
	if a == nil {
		return true // Pure is subsumed by anything
	}
	if b == nil {
		return a == nil // Only pure is subsumed by pure
	}

	// All labels in 'a' must be in 'b' with matching params (invariant).
	// effectParamsCompatible normalises each side via DefaultModeFor so a
	// row with explicit Rand[mode=os] is compatible with a row whose Rand
	// has nil params (both desugar to the same effective {mode: os}).
	for k := range a.Labels {
		if _, ok := b.Labels[k]; !ok {
			return false
		}
		if !effectParamsCompatible(a, b, k) {
			return false
		}
	}
	return true
}

// EffectRowDifference returns the effects in 'a' that are not in 'b'
// Result is sorted alphabetically
func EffectRowDifference(a, b *Row) []string {
	if a == nil {
		return nil
	}

	var diff []string
	for k := range a.Labels {
		if b == nil || b.Labels[k] == nil {
			diff = append(diff, k)
		}
	}

	sort.Strings(diff)
	return diff
}

// FormatEffectRow formats an effect row for display
// Returns "! {IO, FS}" for non-empty rows, "" for pure (nil)
// Includes budget annotations when present: "! {IO @limit=5, FS}"
func FormatEffectRow(row *Row) string {
	if row == nil || len(row.Labels) == 0 {
		return "" // Pure function, omit effect annotation
	}

	// Sort labels for deterministic output
	labels := make([]string, 0, len(row.Labels))
	for k := range row.Labels {
		labels = append(labels, k)
	}
	sort.Strings(labels)

	// Format as ! {Effect1, Effect2, ...} with optional params and budgets
	result := "! {"
	for i, label := range labels {
		if i > 0 {
			result += ", "
		}
		head := label
		// Include params block if present (alphabetical by key)
		if row.Params != nil {
			if pmap, ok := row.Params[label]; ok && len(pmap) > 0 {
				pkeys := make([]string, 0, len(pmap))
				for pk := range pmap {
					pkeys = append(pkeys, pk)
				}
				sort.Strings(pkeys)
				paramParts := make([]string, len(pkeys))
				for j, pk := range pkeys {
					paramParts[j] = fmt.Sprintf("%s=%s", pk, pmap[pk])
				}
				head = fmt.Sprintf("%s[%s]", label, strings.Join(paramParts, ", "))
			}
		}
		// Include budget annotation if present
		if row.Budgets != nil {
			if budget, ok := row.Budgets[label]; ok && budget != nil {
				result += fmt.Sprintf("%s @limit=%d", head, *budget)
			} else {
				result += head
			}
		} else {
			result += head
		}
	}
	result += "}"

	return result
}
