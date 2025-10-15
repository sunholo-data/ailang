package link

import (
	"fmt"
	"sort"

	"github.com/sunholo/ailang/internal/ast"
	"github.com/sunholo/ailang/internal/errors"
)

// newIMP010 creates an error report for missing export
// Data fields: symbol, module_id, available_exports[], search_trace[]
func newIMP010(symbol, modID string, available, trace []string, span *ast.Span) *errors.Report {
	// Ensure deterministic ordering
	sortedAvailable := make([]string, len(available))
	copy(sortedAvailable, available)
	sort.Strings(sortedAvailable)

	sortedTrace := make([]string, len(trace))
	copy(sortedTrace, trace)
	sort.Strings(sortedTrace)

	return &errors.Report{
		Schema:  "ailang.error/v1",
		Code:    "IMP010",
		Phase:   "link",
		Message: fmt.Sprintf("symbol '%s' not exported by '%s'", symbol, modID),
		Span:    span,
		Data: map[string]any{
			"available_exports": sortedAvailable,
			"module_id":         modID,
			"search_trace":      sortedTrace,
			"symbol":            symbol,
		},
		Fix: &errors.Fix{
			Suggestion: fmt.Sprintf("Check exports in %s or import an existing symbol", modID),
			Confidence: 0.85,
		},
	}
}

// newIMP011 creates an error report for import conflict
// Data fields: symbol, module_id, providers[]
func newIMP011(symbol, modID string, providers []string, span *ast.Span) *errors.Report {
	// Ensure deterministic ordering
	sortedProviders := make([]string, len(providers))
	copy(sortedProviders, providers)
	sort.Strings(sortedProviders)

	// Build provider rows with canonical structure
	providerRows := make([]map[string]string, len(sortedProviders))
	for i, p := range sortedProviders {
		providerRows[i] = map[string]string{
			"export":    symbol,
			"module_id": p,
		}
	}

	return &errors.Report{
		Schema:  "ailang.error/v1",
		Code:    "IMP011",
		Phase:   "link",
		Message: fmt.Sprintf("import conflict for '%s'", symbol),
		Span:    span,
		Data: map[string]any{
			"module_id": modID,
			"providers": providerRows,
			"symbol":    symbol,
		},
		Fix: &errors.Fix{
			Suggestion: "Import only one provider of the symbol (use selective imports)",
			Confidence: 0.9,
		},
	}
}

// newIMP012 creates an error report for unsupported import form (namespace imports)
// Data fields: module_id, import_syntax
func newIMP012(modID, importSyntax string, span *ast.Span) *errors.Report {
	return &errors.Report{
		Schema:  "ailang.error/v1",
		Code:    "IMP012",
		Phase:   "link",
		Message: "namespace imports not yet supported",
		Span:    span,
		Data: map[string]any{
			"import_syntax": importSyntax,
			"module_id":     modID,
		},
		Fix: &errors.Fix{
			Suggestion: fmt.Sprintf("Use selective import: import %s (symbol1, symbol2)", modID),
			Confidence: 0.9,
		},
	}
}
