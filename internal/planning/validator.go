// Package planning provides plan validation and code scaffolding for proactive architecture.
package planning

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/sunholo/ailang/internal/schema"
)

// ValidationLevel represents the severity of a validation issue
type ValidationLevel string

const (
	ValidationError   ValidationLevel = "error"
	ValidationWarning ValidationLevel = "warning"
)

// ValidationIssue represents a single validation error or warning
type ValidationIssue struct {
	Level    ValidationLevel `json:"level"`
	Code     string          `json:"code"`
	Message  string          `json:"message"`
	Location string          `json:"location"` // e.g., "modules[0].path"
}

// ValidationResult is the result of validating a plan
type ValidationResult struct {
	Schema   string            `json:"schema"`
	Valid    bool              `json:"valid"`
	Errors   []ValidationIssue `json:"errors"`
	Warnings []ValidationIssue `json:"warnings"`
}

// Validation error codes
const (
	// Module validation (VAL_M##)
	VAL_M01 = "VAL_M01" // Invalid module path
	VAL_M02 = "VAL_M02" // Circular dependency
	VAL_M03 = "VAL_M03" // Duplicate module path
	VAL_M04 = "VAL_M04" // Empty exports list

	// Type validation (VAL_T##)
	VAL_T01 = "VAL_T01" // Invalid type name
	VAL_T02 = "VAL_T02" // Unsupported type kind
	VAL_T03 = "VAL_T03" // Invalid type syntax
	VAL_T04 = "VAL_T04" // Duplicate type name
	VAL_T05 = "VAL_T05" // Module not found for type

	// Function validation (VAL_F##)
	VAL_F01 = "VAL_F01" // Invalid function name
	VAL_F02 = "VAL_F02" // Invalid function signature
	VAL_F03 = "VAL_F03" // Duplicate function name
	VAL_F04 = "VAL_F04" // Module not found for function
	VAL_F05 = "VAL_F05" // Function not exported

	// Effect validation (VAL_E##)
	VAL_E01 = "VAL_E01" // Unknown effect
	VAL_E02 = "VAL_E02" // Effect mismatch (function uses effect not in plan)

	// General validation (VAL_G##)
	VAL_G01 = "VAL_G01" // Empty plan
	VAL_G02 = "VAL_G02" // Missing required field
)

// Canonical AILANG effects
var canonicalEffects = map[string]bool{
	"IO":    true,
	"FS":    true,
	"Net":   true,
	"Clock": true,
	"Rand":  true,
	"DB":    true,
	"Trace": true,
	"Async": true,
}

// ValidatePlan validates a complete plan and returns all errors and warnings
func ValidatePlan(plan *schema.Plan) (*ValidationResult, error) {
	if plan == nil {
		return nil, fmt.Errorf("plan is nil")
	}

	result := &ValidationResult{
		Schema:   schema.PlanV1,
		Valid:    true,
		Errors:   []ValidationIssue{},
		Warnings: []ValidationIssue{},
	}

	// Check for empty plan
	if len(plan.Modules) == 0 && len(plan.Types) == 0 && len(plan.Functions) == 0 {
		result.addError(VAL_G01, "Plan is empty (no modules, types, or functions)", "plan")
		result.Valid = false
		return result, nil
	}

	// Validate modules
	moduleIssues := CheckModulePaths(plan.Modules)
	for _, issue := range moduleIssues {
		result.add(issue)
	}

	// Validate types
	typeIssues := CheckTypeDefinitions(plan.Types, plan.Modules)
	for _, issue := range typeIssues {
		result.add(issue)
	}

	// Validate functions
	funcIssues := CheckFunctionSignatures(plan.Functions, plan.Modules)
	for _, issue := range funcIssues {
		result.add(issue)
	}

	// Validate effects
	effectIssues := CheckEffects(plan.Effects, plan.Functions)
	for _, issue := range effectIssues {
		result.add(issue)
	}

	// Check for circular dependencies
	cycleIssues := CheckDependencyCycles(plan.Modules)
	for _, issue := range cycleIssues {
		result.add(issue)
	}

	// Mark as invalid if any errors were found
	result.Valid = len(result.Errors) == 0

	return result, nil
}

// CheckModulePaths validates module path syntax and uniqueness
func CheckModulePaths(modules []schema.ModulePlan) []ValidationIssue {
	var issues []ValidationIssue
	seen := make(map[string]int)

	// Valid module path pattern: lowercase/underscore, no trailing slashes
	pathPattern := regexp.MustCompile(`^[a-z][a-z0-9_]*(/[a-z][a-z0-9_]*)*$`)

	for i, mod := range modules {
		loc := fmt.Sprintf("modules[%d].path", i)

		// Check for empty path
		if mod.Path == "" {
			issues = append(issues, ValidationIssue{
				Level:    ValidationError,
				Code:     VAL_M01,
				Message:  "Module path cannot be empty",
				Location: loc,
			})
			continue
		}

		// Check path syntax
		if !pathPattern.MatchString(mod.Path) {
			issues = append(issues, ValidationIssue{
				Level:    ValidationError,
				Code:     VAL_M01,
				Message:  fmt.Sprintf("Invalid module path '%s': must be lowercase alphanumeric with underscores, separated by /", mod.Path),
				Location: loc,
			})
		}

		// Check for duplicates
		if prevIdx, exists := seen[mod.Path]; exists {
			issues = append(issues, ValidationIssue{
				Level:    ValidationError,
				Code:     VAL_M03,
				Message:  fmt.Sprintf("Duplicate module path '%s' (also defined at modules[%d])", mod.Path, prevIdx),
				Location: loc,
			})
		} else {
			seen[mod.Path] = i
		}

		// Warn if no exports
		if len(mod.Exports) == 0 {
			issues = append(issues, ValidationIssue{
				Level:    ValidationWarning,
				Code:     VAL_M04,
				Message:  fmt.Sprintf("Module '%s' has no exports", mod.Path),
				Location: loc,
			})
		}
	}

	return issues
}

// CheckTypeDefinitions validates type definitions
func CheckTypeDefinitions(types []schema.TypePlan, modules []schema.ModulePlan) []ValidationIssue {
	var issues []ValidationIssue
	seen := make(map[string]int)
	modulePaths := make(map[string]bool)

	// Build module path set
	for _, mod := range modules {
		modulePaths[mod.Path] = true
	}

	// Valid type name pattern: CamelCase
	namePattern := regexp.MustCompile(`^[A-Z][a-zA-Z0-9]*$`)

	for i, typ := range types {
		loc := fmt.Sprintf("types[%d]", i)

		// Check name syntax
		if !namePattern.MatchString(typ.Name) {
			issues = append(issues, ValidationIssue{
				Level:    ValidationError,
				Code:     VAL_T01,
				Message:  fmt.Sprintf("Invalid type name '%s': must be CamelCase (start with uppercase)", typ.Name),
				Location: loc + ".name",
			})
		}

		// Check kind
		if typ.Kind != "adt" && typ.Kind != "record" && typ.Kind != "alias" {
			issues = append(issues, ValidationIssue{
				Level:    ValidationError,
				Code:     VAL_T02,
				Message:  fmt.Sprintf("Unsupported type kind '%s': must be 'adt', 'record', or 'alias'", typ.Kind),
				Location: loc + ".kind",
			})
		}

		// Check for duplicate names
		if prevIdx, exists := seen[typ.Name]; exists {
			issues = append(issues, ValidationIssue{
				Level:    ValidationError,
				Code:     VAL_T04,
				Message:  fmt.Sprintf("Duplicate type name '%s' (also defined at types[%d])", typ.Name, prevIdx),
				Location: loc + ".name",
			})
		} else {
			seen[typ.Name] = i
		}

		// Check module exists
		if typ.Module != "" && !modulePaths[typ.Module] {
			issues = append(issues, ValidationIssue{
				Level:    ValidationError,
				Code:     VAL_T05,
				Message:  fmt.Sprintf("Type '%s' references undefined module '%s'", typ.Name, typ.Module),
				Location: loc + ".module",
			})
		}

		// Basic syntax check for definition
		if typ.Definition == "" {
			issues = append(issues, ValidationIssue{
				Level:    ValidationError,
				Code:     VAL_T03,
				Message:  fmt.Sprintf("Type '%s' has empty definition", typ.Name),
				Location: loc + ".definition",
			})
		}
	}

	return issues
}

// CheckFunctionSignatures validates function signatures
func CheckFunctionSignatures(funcs []schema.FuncPlan, modules []schema.ModulePlan) []ValidationIssue {
	var issues []ValidationIssue
	seen := make(map[string]int)
	modulePaths := make(map[string]bool)

	// Build module path set and exports
	moduleExports := make(map[string]map[string]bool)
	for _, mod := range modules {
		modulePaths[mod.Path] = true
		exports := make(map[string]bool)
		for _, exp := range mod.Exports {
			exports[exp] = true
		}
		moduleExports[mod.Path] = exports
	}

	// Valid function name pattern: camelCase
	namePattern := regexp.MustCompile(`^[a-z][a-zA-Z0-9_]*$`)

	for i, fn := range funcs {
		loc := fmt.Sprintf("functions[%d]", i)

		// Check name syntax
		if !namePattern.MatchString(fn.Name) {
			issues = append(issues, ValidationIssue{
				Level:    ValidationError,
				Code:     VAL_F01,
				Message:  fmt.Sprintf("Invalid function name '%s': must be camelCase (start with lowercase)", fn.Name),
				Location: loc + ".name",
			})
		}

		// Check module exists
		if fn.Module != "" && !modulePaths[fn.Module] {
			issues = append(issues, ValidationIssue{
				Level:    ValidationError,
				Code:     VAL_F04,
				Message:  fmt.Sprintf("Function '%s' references undefined module '%s'", fn.Name, fn.Module),
				Location: loc + ".module",
			})
		}

		// Check if function is exported (warning only)
		if fn.Module != "" && modulePaths[fn.Module] {
			if exports, ok := moduleExports[fn.Module]; ok {
				if !exports[fn.Name] {
					issues = append(issues, ValidationIssue{
						Level:    ValidationWarning,
						Code:     VAL_F05,
						Message:  fmt.Sprintf("Function '%s' not listed in module '%s' exports", fn.Name, fn.Module),
						Location: loc + ".name",
					})
				}
			}
		}

		// Check for duplicate function names in same module
		key := fn.Module + "::" + fn.Name
		if prevIdx, exists := seen[key]; exists {
			issues = append(issues, ValidationIssue{
				Level:    ValidationError,
				Code:     VAL_F03,
				Message:  fmt.Sprintf("Duplicate function '%s' in module '%s' (also defined at functions[%d])", fn.Name, fn.Module, prevIdx),
				Location: loc + ".name",
			})
		} else {
			seen[key] = i
		}

		// Basic signature check
		if fn.Type == "" {
			issues = append(issues, ValidationIssue{
				Level:    ValidationError,
				Code:     VAL_F02,
				Message:  fmt.Sprintf("Function '%s' has empty type signature", fn.Name),
				Location: loc + ".type",
			})
		}

		// Check effects are valid
		for j, eff := range fn.Effects {
			if !canonicalEffects[eff] {
				issues = append(issues, ValidationIssue{
					Level:    ValidationError,
					Code:     VAL_E01,
					Message:  fmt.Sprintf("Function '%s' uses unknown effect '%s' (valid: IO, FS, Net, Clock, Rand, DB, Trace, Async)", fn.Name, eff),
					Location: fmt.Sprintf("%s.effects[%d]", loc, j),
				})
			}
		}
	}

	return issues
}

// CheckEffects validates effect usage across the plan
func CheckEffects(effects []string, funcs []schema.FuncPlan) []ValidationIssue {
	var issues []ValidationIssue

	// Check each plan-level effect is valid
	for i, eff := range effects {
		if !canonicalEffects[eff] {
			issues = append(issues, ValidationIssue{
				Level:    ValidationError,
				Code:     VAL_E01,
				Message:  fmt.Sprintf("Unknown effect '%s' (valid: IO, FS, Net, Clock, Rand, DB, Trace, Async)", eff),
				Location: fmt.Sprintf("effects[%d]", i),
			})
		}
	}

	// Collect all effects used by functions
	usedEffects := make(map[string]bool)
	for _, fn := range funcs {
		for _, eff := range fn.Effects {
			usedEffects[eff] = true
		}
	}

	// Warn if a function uses an effect not in the plan-level effects list
	planEffects := make(map[string]bool)
	for _, eff := range effects {
		planEffects[eff] = true
	}

	for i, fn := range funcs {
		for j, eff := range fn.Effects {
			if !planEffects[eff] && canonicalEffects[eff] {
				issues = append(issues, ValidationIssue{
					Level:    ValidationWarning,
					Code:     VAL_E02,
					Message:  fmt.Sprintf("Function '%s' uses effect '%s' not listed in plan.effects", fn.Name, eff),
					Location: fmt.Sprintf("functions[%d].effects[%d]", i, j),
				})
			}
		}
	}

	return issues
}

// CheckDependencyCycles detects circular dependencies between modules
func CheckDependencyCycles(modules []schema.ModulePlan) []ValidationIssue {
	var issues []ValidationIssue

	// Build dependency graph
	deps := make(map[string][]string)
	for _, mod := range modules {
		deps[mod.Path] = mod.Imports
	}

	// Check each module for cycles
	for _, mod := range modules {
		if cycle := findCycle(mod.Path, deps, make(map[string]bool), []string{}); cycle != nil {
			issues = append(issues, ValidationIssue{
				Level:    ValidationError,
				Code:     VAL_M02,
				Message:  fmt.Sprintf("Circular dependency detected: %s", strings.Join(cycle, " -> ")),
				Location: fmt.Sprintf("modules[%s]", mod.Path),
			})
		}
	}

	return issues
}

// findCycle performs DFS to detect cycles in module dependencies
func findCycle(node string, graph map[string][]string, visited map[string]bool, path []string) []string {
	if visited[node] {
		// Found a cycle - extract it from path
		for i, p := range path {
			if p == node {
				return append(path[i:], node)
			}
		}
		return nil
	}

	visited[node] = true
	path = append(path, node)

	for _, dep := range graph[node] {
		// Only check dependencies that are internal modules (in the graph)
		if _, exists := graph[dep]; exists {
			if cycle := findCycle(dep, graph, visited, path); cycle != nil {
				return cycle
			}
		}
	}

	visited[node] = false
	return nil
}

// Helper methods for ValidationResult

func (v *ValidationResult) add(issue ValidationIssue) {
	if issue.Level == ValidationError {
		v.Errors = append(v.Errors, issue)
	} else {
		v.Warnings = append(v.Warnings, issue)
	}
}

func (v *ValidationResult) addError(code, message, location string) {
	v.Errors = append(v.Errors, ValidationIssue{
		Level:    ValidationError,
		Code:     code,
		Message:  message,
		Location: location,
	})
}

// addWarning is currently unused but reserved for future validation rules
// func (v *ValidationResult) addWarning(code, message, location string) {
// 	v.Warnings = append(v.Warnings, ValidationIssue{
// 		Level:    ValidationWarning,
// 		Code:     code,
// 		Message:  message,
// 		Location: location,
// 	})
// }

// ToJSON converts validation result to JSON
func (v *ValidationResult) ToJSON() ([]byte, error) {
	return schema.MarshalDeterministic(v)
}

// cleanModulePath is currently unused but reserved for future path cleaning
// func cleanModulePath(path string) string {
// 	return filepath.Clean(strings.TrimSpace(path))
// }
