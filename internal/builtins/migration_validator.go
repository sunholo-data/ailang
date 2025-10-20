package builtins

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// LegacyBuiltinLocation represents a location where builtins used to be registered
type LegacyBuiltinLocation struct {
	FilePath    string   // Path to the file
	Description string   // What used to be there
	Patterns    []string // Code patterns to search for
}

// MigrationReport contains the results of a migration validation
type MigrationReport struct {
	LegacyLocations      []LegacyLocationReport
	CurrentRegistryCount int
	OrphanedBuiltins     []OrphanedBuiltin
	Warnings             []string
	IsClean              bool
}

// LegacyLocationReport reports on one legacy location
type LegacyLocationReport struct {
	Location      LegacyBuiltinLocation
	FileExists    bool
	FoundBuiltins []string
	Notes         string
}

// OrphanedBuiltin represents a builtin found in old locations but not in new registry
type OrphanedBuiltin struct {
	Name     string
	Location string
	Hint     string
}

// GetLegacyLocations returns all locations where builtins used to be registered
func GetLegacyLocations(projectRoot string) []LegacyBuiltinLocation {
	return []LegacyBuiltinLocation{
		{
			FilePath:    filepath.Join(projectRoot, "internal/eval/builtins.go"),
			Description: "Old eval package builtin registry (pre-v0.3.10)",
			Patterns: []string{
				"registry.Register(",
				"registerArithmeticBuiltins",
				"registerComparisonBuiltins",
			},
		},
		{
			FilePath:    filepath.Join(projectRoot, "internal/types/env.go"),
			Description: "Old type environment with show() (pre-v0.3.10)",
			Patterns: []string{
				`"show"`,
				"builtinTypes[",
			},
		},
		{
			FilePath:    filepath.Join(projectRoot, "internal/runtime/builtins.go"),
			Description: "Old runtime builtin registry (pre-v0.3.10)",
			Patterns: []string{
				"RegisterPure(",
				"RegisterEffect(",
				"br.RegisterPure",
				"br.RegisterEffect",
			},
		},
		{
			FilePath:    filepath.Join(projectRoot, "internal/link/builtin_module.go"),
			Description: "Old link interface builtin declarations (pre-v0.3.10)",
			Patterns: []string{
				"iface.Decls[",
				"&iface.FuncDecl{",
			},
		},
	}
}

// ValidateMigration checks if all builtins have been migrated to the new registry
func ValidateMigration(projectRoot string) (*MigrationReport, error) {
	report := &MigrationReport{
		CurrentRegistryCount: len(AllSpecs()),
		IsClean:              true,
	}

	// Check each legacy location
	locations := GetLegacyLocations(projectRoot)
	for _, loc := range locations {
		locReport := scanLegacyLocation(loc)
		report.LegacyLocations = append(report.LegacyLocations, locReport)

		// If we found builtins in old locations, mark as not clean
		if len(locReport.FoundBuiltins) > 0 {
			report.IsClean = false
			for _, builtin := range locReport.FoundBuiltins {
				// Check if it's in the new registry
				if _, exists := GetSpec(builtin); !exists {
					report.OrphanedBuiltins = append(report.OrphanedBuiltins, OrphanedBuiltin{
						Name:     builtin,
						Location: loc.FilePath,
						Hint:     fmt.Sprintf("Add to internal/builtins/ (string.go, math.go, io.go, or net.go)"),
					})
				}
			}
		}
	}

	// Add warnings if we found issues
	if len(report.OrphanedBuiltins) > 0 {
		report.Warnings = append(report.Warnings,
			fmt.Sprintf("Found %d orphaned builtins in legacy locations!", len(report.OrphanedBuiltins)))
	}

	return report, nil
}

// scanLegacyLocation scans a single legacy location for builtin registrations
func scanLegacyLocation(loc LegacyBuiltinLocation) LegacyLocationReport {
	report := LegacyLocationReport{
		Location:      loc,
		FileExists:    false,
		FoundBuiltins: []string{},
	}

	// Check if file exists
	if _, err := os.Stat(loc.FilePath); os.IsNotExist(err) {
		report.Notes = "File does not exist (expected after migration)"
		return report
	}

	report.FileExists = true

	// Read file content
	content, err := os.ReadFile(loc.FilePath)
	if err != nil {
		report.Notes = fmt.Sprintf("Error reading file: %v", err)
		return report
	}

	// Parse the Go source file
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, loc.FilePath, content, parser.AllErrors)
	if err != nil {
		// If we can't parse, fall back to string matching
		report.FoundBuiltins = scanWithStringMatching(string(content), loc.Patterns)
		report.Notes = fmt.Sprintf("Parsed with string matching (parse error: %v)", err)
		return report
	}

	// Use AST-based scanning for more accurate results
	report.FoundBuiltins = scanWithAST(node, loc.Patterns)

	if len(report.FoundBuiltins) == 0 {
		report.Notes = "No builtins found (clean)"
	} else {
		report.Notes = fmt.Sprintf("Found %d potential builtin registrations", len(report.FoundBuiltins))
	}

	return report
}

// scanWithAST scans the AST for builtin registrations
func scanWithAST(node *ast.File, patterns []string) []string {
	var found []string
	foundMap := make(map[string]bool) // Deduplicate

	// Visit all call expressions
	ast.Inspect(node, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		// Check if this is a registration call
		if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
			funcName := sel.Sel.Name
			// Check against patterns
			for _, pattern := range patterns {
				if strings.Contains(pattern, funcName) {
					// Extract builtin name from first argument if it's a string
					if len(call.Args) > 0 {
						if lit, ok := call.Args[0].(*ast.BasicLit); ok && lit.Kind == token.STRING {
							name := strings.Trim(lit.Value, `"`)
							if !foundMap[name] {
								found = append(found, name)
								foundMap[name] = true
							}
						}
					}
				}
			}
		}

		return true
	})

	sort.Strings(found)
	return found
}

// scanWithStringMatching is a fallback when AST parsing fails
func scanWithStringMatching(content string, patterns []string) []string {
	var found []string
	foundMap := make(map[string]bool)

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		// Skip comments
		if strings.Contains(line, "//") {
			line = strings.Split(line, "//")[0]
		}

		// Check if line contains any pattern
		for _, pattern := range patterns {
			if strings.Contains(line, pattern) {
				// Try to extract quoted strings (potential builtin names)
				if strings.Contains(line, `"`) {
					parts := strings.Split(line, `"`)
					for i := 1; i < len(parts); i += 2 {
						name := parts[i]
						if isValidBuiltinName(name) && !foundMap[name] {
							found = append(found, name)
							foundMap[name] = true
						}
					}
				}
			}
		}
	}

	sort.Strings(found)
	return found
}

// isValidBuiltinName checks if a string looks like a builtin name
func isValidBuiltinName(name string) bool {
	// Builtins typically:
	// - Start with _ or are lowercase with _
	// - Contain letters, numbers, underscores
	// - Are not empty or very long
	if len(name) == 0 || len(name) > 50 {
		return false
	}

	// Common false positives to skip
	falsePositives := []string{
		"", "pure", "effect", "builtin", "name", "value", "type",
		"Int", "Float", "String", "Bool", "List", "Record",
	}
	for _, fp := range falsePositives {
		if name == fp {
			return false
		}
	}

	// Should contain underscore or start with lowercase
	return strings.Contains(name, "_") || (name[0] >= 'a' && name[0] <= 'z')
}

// FormatReport formats a migration report for display
func FormatReport(report *MigrationReport) string {
	var sb strings.Builder

	sb.WriteString("=== Builtin Migration Validation ===\n\n")

	// Summary
	sb.WriteString("Current Registry:\n")
	sb.WriteString(fmt.Sprintf("  Total builtins: %d\n\n", report.CurrentRegistryCount))

	// Legacy locations
	sb.WriteString("Legacy Locations:\n")
	for _, loc := range report.LegacyLocations {
		status := "✓"
		if len(loc.FoundBuiltins) > 0 {
			status = "⚠"
		}
		sb.WriteString(fmt.Sprintf("  %s %s\n", status, filepath.Base(loc.Location.FilePath)))
		if loc.FileExists {
			sb.WriteString(fmt.Sprintf("     %s\n", loc.Notes))
			if len(loc.FoundBuiltins) > 0 {
				sb.WriteString("     Found: ")
				sb.WriteString(strings.Join(loc.FoundBuiltins, ", "))
				sb.WriteString("\n")
			}
		} else {
			sb.WriteString("     File does not exist (expected)\n")
		}
	}
	sb.WriteString("\n")

	// Orphaned builtins
	if len(report.OrphanedBuiltins) > 0 {
		sb.WriteString("⚠️  ORPHANED BUILTINS DETECTED:\n")
		for _, orphan := range report.OrphanedBuiltins {
			sb.WriteString(fmt.Sprintf("\n  • %s\n", orphan.Name))
			sb.WriteString(fmt.Sprintf("    Location: %s\n", filepath.Base(orphan.Location)))
			sb.WriteString(fmt.Sprintf("    Action: %s\n", orphan.Hint))
		}
		sb.WriteString("\n")
	}

	// Warnings
	if len(report.Warnings) > 0 {
		sb.WriteString("Warnings:\n")
		for _, warning := range report.Warnings {
			sb.WriteString(fmt.Sprintf("  • %s\n", warning))
		}
		sb.WriteString("\n")
	}

	// Final status
	if report.IsClean {
		sb.WriteString("✅ Migration Status: COMPLETE\n")
		sb.WriteString("   No orphaned builtins detected.\n")
	} else {
		sb.WriteString("❌ Migration Status: INCOMPLETE\n")
		sb.WriteString("   Some builtins may not have been migrated.\n")
		sb.WriteString("   Review the orphaned builtins above and migrate them.\n")
	}

	return sb.String()
}
