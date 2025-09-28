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