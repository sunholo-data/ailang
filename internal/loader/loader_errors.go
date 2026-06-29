package loader

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/errors"
	"github.com/sunholo-data/ailang/internal/importhint"
)

// newIMP010Loader creates an IMP010 error report (symbol not exported)
// Similar to link.newIMP010 but for the loader context
func newIMP010Loader(symbol, modID string, available []string, span *ast.Span) *errors.Report {
	sortedAvailable := make([]string, len(available))
	copy(sortedAvailable, available)
	sort.Strings(sortedAvailable)

	// The CLI renders only "CODE: Message", so the actionable hint must live in Message to reach
	// an agent reading `ailang check` output. Same hint as the linker path. M-AGENT-STUCK-FIXES M2.
	hint := importhint.IMP010(symbol, modID)
	suggestion := fmt.Sprintf("Check exports in %s. Available: %s",
		modID, strings.Join(sortedAvailable[:min(3, len(sortedAvailable))], ", "))
	confidence := 0.85
	if hint != "" {
		suggestion = strings.TrimPrefix(hint, " — ")
		confidence = 0.95
	}

	return &errors.Report{
		Schema:  "ailang.error/v1",
		Code:    "IMP010",
		Phase:   "loader",
		Message: fmt.Sprintf("symbol '%s' not exported by '%s'%s", symbol, modID, hint),
		Span:    span,
		Data: map[string]any{
			"available_exports": sortedAvailable,
			"module_id":         modID,
			"symbol":            symbol,
		},
		Fix: &errors.Fix{
			Suggestion: suggestion,
			Confidence: confidence,
		},
	}
}

// newLDR001 creates an error report for module not found
// Data fields: module_id, search_trace[], similar[] (optional)
func newLDR001(modID string, searchTrace, similar []string, span *ast.Span) *errors.Report {
	// Ensure deterministic ordering
	sortedTrace := make([]string, len(searchTrace))
	copy(sortedTrace, searchTrace)
	sort.Strings(sortedTrace)

	sortedSimilar := make([]string, len(similar))
	copy(sortedSimilar, similar)
	sort.Strings(sortedSimilar)

	data := map[string]any{
		"module_id":    modID,
		"search_trace": sortedTrace,
	}

	// Only add similar if non-empty
	if len(sortedSimilar) > 0 {
		data["similar"] = sortedSimilar
	}

	suggestion := fmt.Sprintf("Check module path '%s' exists", modID)
	if len(sortedSimilar) > 0 {
		suggestion = fmt.Sprintf("Module not found. Similar modules: %s", strings.Join(sortedSimilar[:min(3, len(sortedSimilar))], ", "))
	}

	return &errors.Report{
		Schema:  "ailang.error/v1",
		Code:    "LDR001",
		Phase:   "loader",
		Message: fmt.Sprintf("module not found: %s", modID),
		Span:    span,
		Data:    data,
		Fix: &errors.Fix{
			Suggestion: suggestion,
			Confidence: 0.85,
		},
	}
}

// suggestSimilar finds similar module names based on simple heuristic
func (ml *ModuleLoader) suggestSimilar(want string) []string {
	// Collect all cached module paths
	var all []string
	for cached := range ml.cache {
		all = append(all, cached)
	}

	// Find modules containing any part of the wanted path
	var hits []string
	base := filepath.Base(want)

	for _, s := range all {
		// Check if the cached path contains the base name
		if strings.Contains(s, base) {
			hits = append(hits, s)
			continue
		}
		// Check if any path component matches
		wantParts := strings.Split(want, "/")
		sParts := strings.Split(s, "/")
		for _, wp := range wantParts {
			for _, sp := range sParts {
				if wp == sp && wp != "" {
					hits = append(hits, s)
					break
				}
			}
		}
	}

	// Remove duplicates and sort
	seen := make(map[string]bool)
	var unique []string
	for _, h := range hits {
		if !seen[h] {
			seen[h] = true
			unique = append(unique, h)
		}
	}

	sort.Strings(unique)

	// Return top 5
	if len(unique) > 5 {
		return unique[:5]
	}
	return unique
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
