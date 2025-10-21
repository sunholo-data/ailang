package types

import (
	"github.com/sunholo/ailang/internal/ast"
)

// Helper functions for type inference

func getParamNames(params []*ast.Param) []string {
	names := make([]string, len(params))
	for i, param := range params {
		names[i] = param.Name
	}
	return names
}

// hasLinearCapabilities checks if a type contains linear capabilities
func hasLinearCapabilities(typ interface{}) bool {
	switch t := typ.(type) {
	case *TFunc2:
		// Check if function effects contain linear capabilities
		return hasLinearEffects(t.EffectRow)
	case *Scheme:
		// Check the underlying type
		if funcType, ok := t.Type.(*TFunc2); ok {
			return hasLinearEffects(funcType.EffectRow)
		}
	}
	return false
}

// hasLinearEffects checks if an effect row contains linear capabilities
func hasLinearEffects(effectRow *Row) bool {
	if effectRow == nil {
		return false
	}

	// Check for known linear capabilities in effect labels
	// In a real implementation, this would be configurable
	linearCapabilities := []string{"FS", "Net", "Time", "Rand", "Console"}

	for _, capName := range linearCapabilities {
		if _, exists := effectRow.Labels[capName]; exists {
			return true
		}
	}

	return false
}

// getLinearCapabilities returns the names of linear capabilities in a type
func getLinearCapabilities(typ interface{}) []string {
	var capabilities []string

	switch t := typ.(type) {
	case *TFunc2:
		capabilities = append(capabilities, getLinearEffectNames(t.EffectRow)...)
	case *Scheme:
		if funcType, ok := t.Type.(*TFunc2); ok {
			capabilities = append(capabilities, getLinearEffectNames(funcType.EffectRow)...)
		}
	}

	return capabilities
}

// getLinearEffectNames extracts linear capability names from an effect row
func getLinearEffectNames(effectRow *Row) []string {
	if effectRow == nil {
		return nil
	}

	var linearCaps []string
	linearCapabilities := []string{"FS", "Net", "Time", "Rand", "Console"}

	for _, capName := range linearCapabilities {
		if _, exists := effectRow.Labels[capName]; exists {
			linearCaps = append(linearCaps, capName)
		}
	}

	return linearCaps
}
