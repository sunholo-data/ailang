package effects

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// FormatReport formats a budget report as human-readable text
//
// M-DX25: Shows both semantic (charged to caller) and physical (actual calls) counts.
// When semantic == physical, shows simplified format.
//
// Example output:
//
//	Budget Report:
//	  main:               IO 20        (when semantic == physical)
//	  getDefaultProject:  FS semantic 5/5 physical 12
//
//	Total: IO=20, FS semantic=5 physical=12
func FormatReport(br *BudgetReport) string {
	if br == nil || !br.HasUsage() {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("\nBudget Report:\n")

	// Collect all function names from both semantic and physical maps
	funcSet := make(map[string]bool)
	for name := range br.FunctionUsage {
		funcSet[name] = true
	}
	for name := range br.FunctionPhysicalUsage {
		funcSet[name] = true
	}

	funcNames := make([]string, 0, len(funcSet))
	for name := range funcSet {
		funcNames = append(funcNames, name)
	}
	sort.Strings(funcNames)

	// Find max function name length for alignment
	maxLen := 0
	for _, name := range funcNames {
		if len(name) > maxLen {
			maxLen = len(name)
		}
	}

	// Print per-function usage
	for _, funcName := range funcNames {
		semanticUsage := br.FunctionUsage[funcName]
		physicalUsage := br.FunctionPhysicalUsage[funcName]
		limits := br.FunctionLimits[funcName]

		sb.WriteString(fmt.Sprintf("  %-*s  ", maxLen, funcName+":"))

		// Collect all effects for this function
		effectSet := make(map[string]bool)
		for effect := range semanticUsage {
			effectSet[effect] = true
		}
		for effect := range physicalUsage {
			effectSet[effect] = true
		}

		effects := make([]string, 0, len(effectSet))
		for effect := range effectSet {
			effects = append(effects, effect)
		}
		sort.Strings(effects)

		parts := make([]string, 0, len(effects))
		for _, effect := range effects {
			semantic := 0
			if semanticUsage != nil {
				semantic = semanticUsage[effect]
			}
			physical := 0
			if physicalUsage != nil {
				physical = physicalUsage[effect]
			}

			// Format: "IO 20" if same, "IO semantic 3/5 physical 19" if different
			if semantic == physical {
				// Same counts - simpler format
				if limits != nil {
					if limit, ok := limits[effect]; ok && limit != nil {
						parts = append(parts, fmt.Sprintf("%s %d/%d", effect, physical, *limit))
					} else {
						parts = append(parts, fmt.Sprintf("%s %d", effect, physical))
					}
				} else {
					parts = append(parts, fmt.Sprintf("%s %d", effect, physical))
				}
			} else {
				// Different counts - show both
				if limits != nil {
					if limit, ok := limits[effect]; ok && limit != nil {
						parts = append(parts, fmt.Sprintf("%s semantic %d/%d physical %d", effect, semantic, *limit, physical))
					} else {
						parts = append(parts, fmt.Sprintf("%s semantic %d physical %d", effect, semantic, physical))
					}
				} else {
					parts = append(parts, fmt.Sprintf("%s semantic %d physical %d", effect, semantic, physical))
				}
			}
		}
		sb.WriteString(strings.Join(parts, "  "))
		sb.WriteString("\n")
	}

	// Print totals
	sb.WriteString("\nTotal: ")

	// Collect all effects
	effectSet := make(map[string]bool)
	for effect := range br.TotalUsage {
		effectSet[effect] = true
	}
	for effect := range br.TotalPhysicalUsage {
		effectSet[effect] = true
	}

	effects := make([]string, 0, len(effectSet))
	for effect := range effectSet {
		effects = append(effects, effect)
	}
	sort.Strings(effects)

	parts := make([]string, 0, len(effects))
	for _, effect := range effects {
		semantic := br.TotalUsage[effect]
		physical := br.TotalPhysicalUsage[effect]
		if semantic == physical {
			parts = append(parts, fmt.Sprintf("%s=%d", effect, physical))
		} else {
			parts = append(parts, fmt.Sprintf("%s semantic=%d physical=%d", effect, semantic, physical))
		}
	}
	sb.WriteString(strings.Join(parts, ", "))
	sb.WriteString("\n")

	return sb.String()
}

// BudgetReportJSON is the JSON-serializable format for budget reports
type BudgetReportJSON struct {
	Functions     map[string]FunctionBudgetJSON `json:"functions"`
	Total         map[string]int                `json:"total"`
	TotalPhysical map[string]int                `json:"total_physical,omitempty"`
}

// FunctionBudgetJSON represents budget usage for a single function
type FunctionBudgetJSON struct {
	Effects map[string]EffectBudgetJSON `json:"effects"`
}

// EffectBudgetJSON represents budget usage for a single effect
type EffectBudgetJSON struct {
	Semantic int  `json:"semantic"`        // Semantic usage (charged to caller)
	Physical int  `json:"physical"`        // Physical usage (actual calls)
	Limit    *int `json:"limit,omitempty"` // Semantic limit (nil = unlimited)
}

// FormatReportJSON formats a budget report as JSON
func FormatReportJSON(br *BudgetReport) ([]byte, error) {
	if br == nil || !br.HasUsage() {
		return []byte("{}"), nil
	}

	report := BudgetReportJSON{
		Functions:     make(map[string]FunctionBudgetJSON),
		Total:         br.TotalUsage,
		TotalPhysical: br.TotalPhysicalUsage,
	}

	// Collect all function names
	funcSet := make(map[string]bool)
	for name := range br.FunctionUsage {
		funcSet[name] = true
	}
	for name := range br.FunctionPhysicalUsage {
		funcSet[name] = true
	}

	for funcName := range funcSet {
		funcBudget := FunctionBudgetJSON{
			Effects: make(map[string]EffectBudgetJSON),
		}

		semanticUsage := br.FunctionUsage[funcName]
		physicalUsage := br.FunctionPhysicalUsage[funcName]
		limits := br.FunctionLimits[funcName]

		// Collect all effects
		effectSet := make(map[string]bool)
		for effect := range semanticUsage {
			effectSet[effect] = true
		}
		for effect := range physicalUsage {
			effectSet[effect] = true
		}

		for effect := range effectSet {
			semantic := 0
			if semanticUsage != nil {
				semantic = semanticUsage[effect]
			}
			physical := 0
			if physicalUsage != nil {
				physical = physicalUsage[effect]
			}

			eb := EffectBudgetJSON{
				Semantic: semantic,
				Physical: physical,
			}
			if limits != nil {
				if limit, ok := limits[effect]; ok {
					eb.Limit = limit
				}
			}
			funcBudget.Effects[effect] = eb
		}

		report.Functions[funcName] = funcBudget
	}

	return json.MarshalIndent(report, "", "  ")
}

// FormatReportForError formats a budget report summary for inclusion in error messages
func FormatReportForError(br *BudgetReport) string {
	if br == nil || !br.HasUsage() {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("\n\nBudget at failure:\n")

	// Collect all function names
	funcSet := make(map[string]bool)
	for name := range br.FunctionUsage {
		funcSet[name] = true
	}
	for name := range br.FunctionPhysicalUsage {
		funcSet[name] = true
	}

	funcNames := make([]string, 0, len(funcSet))
	for name := range funcSet {
		funcNames = append(funcNames, name)
	}
	sort.Strings(funcNames)

	for _, funcName := range funcNames {
		semanticUsage := br.FunctionUsage[funcName]
		physicalUsage := br.FunctionPhysicalUsage[funcName]
		limits := br.FunctionLimits[funcName]

		// Collect all effects
		effectSet := make(map[string]bool)
		for effect := range semanticUsage {
			effectSet[effect] = true
		}
		for effect := range physicalUsage {
			effectSet[effect] = true
		}

		effects := make([]string, 0, len(effectSet))
		for effect := range effectSet {
			effects = append(effects, effect)
		}
		sort.Strings(effects)

		for _, effect := range effects {
			semantic := 0
			if semanticUsage != nil {
				semantic = semanticUsage[effect]
			}
			physical := 0
			if physicalUsage != nil {
				physical = physicalUsage[effect]
			}

			if limits != nil {
				if limit, ok := limits[effect]; ok && limit != nil {
					sb.WriteString(fmt.Sprintf("  %s: %s semantic %d/%d physical %d\n", funcName, effect, semantic, *limit, physical))
				} else {
					sb.WriteString(fmt.Sprintf("  %s: %s semantic %d physical %d\n", funcName, effect, semantic, physical))
				}
			} else {
				if semantic == physical {
					sb.WriteString(fmt.Sprintf("  %s: %s %d\n", funcName, effect, physical))
				} else {
					sb.WriteString(fmt.Sprintf("  %s: %s semantic %d physical %d\n", funcName, effect, semantic, physical))
				}
			}
		}
	}

	return sb.String()
}
