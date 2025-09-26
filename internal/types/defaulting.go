package types

import (
	"fmt"
	"sort"
	"strings"
)

// DefaultingTrace records when numeric defaulting occurs
type DefaultingTrace struct {
	TypeVar   string   // The type variable being defaulted
	ClassName string   // The class constraint (Num, Fractional, etc.)
	Default   Type     // The chosen default type
	Location  string   // Source location
}

// DefaultingConfig controls numeric literal defaulting
type DefaultingConfig struct {
	Enabled    bool              // Whether defaulting is enabled
	Defaults   map[string]Type   // Class -> default type mapping
	Traces     []DefaultingTrace // Record of defaulting decisions
}

// NewDefaultingConfig creates a standard defaulting configuration
func NewDefaultingConfig() *DefaultingConfig {
	return &DefaultingConfig{
		Enabled: true,
		Defaults: map[string]Type{
			"Num":        TInt,   // Ambiguous numeric literals default to Int
			"Fractional": TFloat, // Fractional literals default to Float
		},
		Traces: []DefaultingTrace{},
	}
}

// applyNumericDefaulting applies defaulting to ambiguous numeric type variables
// This happens AFTER unification, BEFORE constraint partitioning
func (tc *CoreTypeChecker) applyNumericDefaulting(
	sub Substitution,
	constraints []ClassConstraint,
	config *DefaultingConfig,
) (Substitution, []DefaultingTrace) {
	
	if !config.Enabled {
		return sub, nil
	}

	traces := []DefaultingTrace{}
	
	// Group constraints by type variable
	varConstraints := make(map[string][]ClassConstraint)
	for _, c := range constraints {
		// Handle both TVar and TVar2 types
		var varName string
		switch tv := c.Type.(type) {
		case *TVar:
			varName = tv.Name
		case *TVar2:
			varName = tv.Name
		default:
			continue // Not a type variable
		}
		
		// Check if already resolved by substitution
		if _, resolved := sub[varName]; !resolved {
			varConstraints[varName] = append(varConstraints[varName], c)
		}
	}
	
	// Apply defaulting for each ambiguous type variable
	for varName, constrs := range varConstraints {
		// Check if this is a numeric constraint that can be defaulted
		for _, c := range constrs {
			if defaultType, ok := config.Defaults[c.Class]; ok {
				// Apply the default
				sub[varName] = defaultType
				
				// Record the trace
				trace := DefaultingTrace{
					TypeVar:   varName,
					ClassName: c.Class,
					Default:   defaultType,
					Location:  fmt.Sprintf("%v", c.Path),
				}
				traces = append(traces, trace)
				config.Traces = append(config.Traces, trace)
				
				// Log the defaulting decision
				tc.logDefaulting(trace)
				break // Only default once per variable
			}
		}
	}
	
	return sub, traces
}

// logDefaulting logs a defaulting decision for reproducibility
func (tc *CoreTypeChecker) logDefaulting(trace DefaultingTrace) {
	if tc.debugMode {
		fmt.Printf("[default] %s under %s → %s at %s\n",
			trace.TypeVar,
			trace.ClassName,
			trace.Default.String(),
			trace.Location,
		)
	}
}

// FormatDefaultingTraces creates a human-readable summary of defaulting decisions
func FormatDefaultingTraces(traces []DefaultingTrace) string {
	if len(traces) == 0 {
		return ""
	}
	
	var lines []string
	lines = append(lines, "Numeric defaulting applied:")
	
	// Sort for deterministic output
	sort.Slice(traces, func(i, j int) bool {
		if traces[i].Location != traces[j].Location {
			return traces[i].Location < traces[j].Location
		}
		return traces[i].TypeVar < traces[j].TypeVar
	})
	
	for _, trace := range traces {
		lines = append(lines, fmt.Sprintf("  • %s: %s[%s] defaulted to %s",
			trace.Location,
			trace.ClassName,
			trace.TypeVar,
			trace.Default.String(),
		))
	}
	
	return strings.Join(lines, "\n")
}

// isAmbiguousNumeric checks if a type variable is ambiguous and numeric
func isAmbiguousNumeric(tv *TVar, constraints []ClassConstraint) bool {
	for _, c := range constraints {
		if ctv, ok := c.Type.(*TVar); ok && ctv.Name == tv.Name {
			switch c.Class {
			case "Num", "Fractional", "Integral", "RealFrac":
				return true
			}
		}
	}
	return false
}

// DisableDefaulting creates a config with defaulting disabled
func DisableDefaulting() *DefaultingConfig {
	return &DefaultingConfig{
		Enabled:  false,
		Defaults: map[string]Type{},
		Traces:   []DefaultingTrace{},
	}
}

// ModuleScopedDefaults allows per-module defaulting configuration
type ModuleScopedDefaults struct {
	ModuleName string
	Config     *DefaultingConfig
}

// GetModuleDefaults retrieves defaulting config for a specific module
func GetModuleDefaults(moduleName string) *DefaultingConfig {
	// For now, all modules use the same defaults
	// Later this can be extended to per-module configuration
	return NewDefaultingConfig()
}