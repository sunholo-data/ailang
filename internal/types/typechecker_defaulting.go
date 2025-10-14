package types

import (
	"fmt"
)

// defaultAmbiguities applies spec-compliant numeric defaulting at generalization boundaries
// This is the ONLY place where defaulting should happen in the entire system
func (tc *CoreTypeChecker) defaultAmbiguities(
	monotype Type,
	constraints []ClassConstraint,
) (Substitution, Type, []ClassConstraint, error) {

	if !tc.defaultingConfig.Enabled {
		return make(Substitution), monotype, constraints, nil
	}

	// Step 1: Compute ambiguous type variables A = ftv(C) \ ftv(τ)
	constraintVars := make(map[string]bool)
	for _, c := range constraints {
		collectConstraintVars(c.Type, constraintVars)
	}

	monotypeVars := make(map[string]bool)
	collectFreeVars(monotype, monotypeVars)

	ambiguousVars := make(map[string]bool)
	for v := range constraintVars {
		if !monotypeVars[v] {
			ambiguousVars[v] = true
		}
	}

	if tc.debugMode && len(ambiguousVars) > 0 {
		fmt.Printf("[debug] Ambiguous vars: ")
		for v := range ambiguousVars {
			fmt.Printf("%s ", v)
		}
		fmt.Printf("\n[debug] Monotype vars: ")
		for v := range monotypeVars {
			fmt.Printf("%s ", v)
		}
		fmt.Println()
	}

	if len(ambiguousVars) == 0 {
		return make(Substitution), monotype, constraints, nil
	}

	// Step 2: For each ambiguous var α, collect class set Kα
	varClasses := make(map[string]map[string]bool)
	for _, c := range constraints {
		if varName := extractVarName(c.Type); varName != "" && ambiguousVars[varName] {
			if varClasses[varName] == nil {
				varClasses[varName] = make(map[string]bool)
			}
			varClasses[varName][c.Class] = true
		}
	}

	// Step 3: Apply module defaults with conflict detection
	sub := make(Substitution)
	// traces := []DefaultingTrace{} // Not used

	for varName, classes := range varClasses {
		defaultType, err := tc.pickDefault(classes)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("ambiguous type variable %s with classes %v: %w",
				varName, getClassNames(classes), err)
		}

		if defaultType != nil {
			sub[varName] = defaultType

			// Record trace for deterministic output
			trace := DefaultingTrace{
				TypeVar:   varName,
				ClassName: getFirstClassName(classes), // Representative class
				Default:   defaultType,
				Location:  "generalization boundary",
			}
			// traces = append(traces, trace) // Not used after this
			tc.defaultingConfig.Traces = append(tc.defaultingConfig.Traces, trace)

			if tc.debugMode {
				tc.logDefaulting(trace)
			}
		}
	}

	// Step 4: Apply substitution consistently everywhere
	if len(sub) > 0 {
		monotype = ApplySubstitution(sub, monotype)
		constraints = tc.applySubstitutionToConstraints(sub, constraints)

		// SAFETY CHECK: Ensure defaulting only affects Star-kinded types
		for varName, defaultType := range sub {
			if !isStarKinded(defaultType) {
				return nil, nil, nil, fmt.Errorf("INTERNAL ERROR: defaulting variable %s to non-Star type %s", varName, defaultType)
			}
		}
	}

	return sub, monotype, constraints, nil
}

// defaultAmbiguitiesTopLevel applies defaulting at top-level, including non-ambiguous numeric literals
func (tc *CoreTypeChecker) defaultAmbiguitiesTopLevel(
	monotype Type,
	constraints []ClassConstraint,
) (Substitution, Type, []ClassConstraint, error) {

	if !tc.defaultingConfig.Enabled {
		return make(Substitution), monotype, constraints, nil
	}

	// At top-level, we want to default ANY type variable with defaultable constraints
	// not just ambiguous ones (this gives the REPL experience users expect)

	// Collect all type variables in constraints that have defaultable classes
	defaultableVars := make(map[string]map[string]bool)
	for _, c := range constraints {
		if varName := extractVarName(c.Type); varName != "" {
			// Check if this class is defaultable
			if tc.isDefaultableClass(c.Class) {
				if defaultableVars[varName] == nil {
					defaultableVars[varName] = make(map[string]bool)
				}
				defaultableVars[varName][c.Class] = true
			}
		}
	}

	if len(defaultableVars) == 0 {
		return make(Substitution), monotype, constraints, nil
	}

	// Apply defaults to all defaultable variables
	sub := make(Substitution)
	// traces := []DefaultingTrace{} // Not used

	for varName, classes := range defaultableVars {
		defaultType, err := tc.pickDefault(classes)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("ambiguous type variable %s with classes %v: %w",
				varName, getClassNames(classes), err)
		}

		if defaultType != nil {
			sub[varName] = defaultType

			trace := DefaultingTrace{
				TypeVar:   varName,
				ClassName: getFirstClassName(classes),
				Default:   defaultType,
				Location:  "top-level",
			}
			// traces = append(traces, trace) // Not used after this
			tc.defaultingConfig.Traces = append(tc.defaultingConfig.Traces, trace)

			if tc.debugMode {
				tc.logDefaulting(trace)
			}
		}
	}

	// Apply substitution
	if len(sub) > 0 {
		monotype = ApplySubstitution(sub, monotype)
		constraints = tc.applySubstitutionToConstraints(sub, constraints)

		// Safety check
		for varName, defaultType := range sub {
			if !isStarKinded(defaultType) {
				return nil, nil, nil, fmt.Errorf("INTERNAL ERROR: defaulting variable %s to non-Star type %s", varName, defaultType)
			}
		}
	}

	return sub, monotype, constraints, nil
}

// isDefaultableClass checks if a class can be defaulted
func (tc *CoreTypeChecker) isDefaultableClass(className string) bool {
	switch className {
	case "Num", "Fractional":
		return true
	default:
		return false
	}
}

// pickDefault applies module-scoped defaulting rules
func (tc *CoreTypeChecker) pickDefault(classes map[string]bool) (Type, error) {
	// Define neutral classes that don't affect numeric defaulting
	// These classes don't choose a numeric representation
	neutral := map[string]bool{
		"Eq":   true,
		"Ord":  true,
		"Show": true,
	}

	// Filter out neutral classes to find primary numeric constraints
	var primary []string
	for class := range classes {
		if !neutral[class] {
			primary = append(primary, class)
		}
	}

	// Handle defaulting based on remaining primary constraints
	switch {
	case len(primary) == 0:
		// Only neutral constraints present (Eq, Ord, Show)
		// Default to Int for Ord/Eq/Show when no numeric context
		// This handles comparisons like `x > y` where x, y are already Int
		if classes["Ord"] || classes["Eq"] {
			return &TCon{Name: "int"}, nil
		}
		// For Show-only, also default to Int
		if classes["Show"] {
			return &TCon{Name: "int"}, nil
		}
		return nil, fmt.Errorf("ambiguous type requires annotation")

	case len(primary) == 1 && primary[0] == "Num":
		// Pure Num constraint (possibly with neutral constraints like Eq, Ord)
		if def := tc.instanceEnv.DefaultFor("Num"); def != nil {
			return def, nil
		}
		return nil, fmt.Errorf("no default for Num; add type annotation")

	case len(primary) == 1 && primary[0] == "Fractional":
		// Pure Fractional constraint (possibly with neutral constraints)
		if def := tc.instanceEnv.DefaultFor("Fractional"); def != nil {
			return def, nil
		}
		return nil, fmt.Errorf("no default for Fractional; add type annotation")

	case len(primary) == 2 && classes["Fractional"] && classes["Num"]:
		// Fractional implies Num, so this is effectively just Fractional
		if def := tc.instanceEnv.DefaultFor("Fractional"); def != nil {
			return def, nil
		}
		return nil, fmt.Errorf("no default for Fractional; add type annotation")

	default:
		// Mixed non-neutral constraints → require annotation
		// This maintains spec compliance: only default within a single family
		return nil, fmt.Errorf("mixed constraints require type annotation")
	}
}

// Helper functions for defaulting

func collectConstraintVars(t Type, vars map[string]bool) {
	switch typ := t.(type) {
	case *TVar:
		vars[typ.Name] = true
	case *TVar2:
		vars[typ.Name] = true
	case *TApp:
		collectConstraintVars(typ.Constructor, vars)
		for _, arg := range typ.Args {
			collectConstraintVars(arg, vars)
		}
	case *TFunc:
		for _, p := range typ.Params {
			collectConstraintVars(p, vars)
		}
		collectConstraintVars(typ.Return, vars)
	case *TFunc2:
		for _, p := range typ.Params {
			collectConstraintVars(p, vars)
		}
		collectConstraintVars(typ.Return, vars)
	case *TRecord:
		for _, fieldType := range typ.Fields {
			collectConstraintVars(fieldType, vars)
		}
	}
}

func collectFreeVars(t Type, vars map[string]bool) {
	switch typ := t.(type) {
	case *TVar:
		vars[typ.Name] = true
	case *TVar2:
		vars[typ.Name] = true
	case *TApp:
		collectFreeVars(typ.Constructor, vars)
		for _, arg := range typ.Args {
			collectFreeVars(arg, vars)
		}
	case *TFunc:
		for _, p := range typ.Params {
			collectFreeVars(p, vars)
		}
		collectFreeVars(typ.Return, vars)
	case *TFunc2:
		for _, p := range typ.Params {
			collectFreeVars(p, vars)
		}
		collectFreeVars(typ.Return, vars)
	case *TRecord:
		for _, fieldType := range typ.Fields {
			collectFreeVars(fieldType, vars)
		}
	}
}

func extractVarName(t Type) string {
	switch typ := t.(type) {
	case *TVar:
		return typ.Name
	case *TVar2:
		return typ.Name
	default:
		return ""
	}
}

func getClassNames(classes map[string]bool) []string {
	names := make([]string, 0, len(classes))
	for name := range classes {
		names = append(names, name)
	}
	// Sort for deterministic output
	for i := 0; i < len(names)-1; i++ {
		for j := i + 1; j < len(names); j++ {
			if names[i] > names[j] {
				names[i], names[j] = names[j], names[i]
			}
		}
	}
	return names
}

func getFirstClassName(classes map[string]bool) string {
	names := getClassNames(classes)
	if len(names) > 0 {
		return names[0]
	}
	return ""
}

// isStarKinded checks that a type has kind Star (not effect row or record row)
func isStarKinded(t Type) bool {
	switch t.(type) {
	case *TVar:
		return true // Assume Star for TVar (simplified)
	case *TVar2:
		return true // Assume Star for TVar2 (simplified)
	case *Row:
		return false // Rows are not Star-kinded
	case *RowVar:
		return false // Row variables are not Star-kinded
	default:
		return true // TCon, TInt, TFloat, etc. are Star-kinded
	}
}
