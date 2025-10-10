// Package repl provides planning commands for the REPL.
package repl

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/sunholo/ailang/internal/planning"
	"github.com/sunholo/ailang/internal/schema"
)

// ProposePlanCommand validates a plan file and prints the result
func ProposePlanCommand(filename string) error {
	// Load plan from JSON file
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read plan file: %w", err)
	}

	plan, err := schema.PlanFromJSON(data)
	if err != nil {
		return fmt.Errorf("failed to parse plan: %w", err)
	}

	// Validate plan
	result, err := planning.ValidatePlan(plan)
	if err != nil {
		return fmt.Errorf("validation error: %w", err)
	}

	// Print validation result
	printValidationResult(result)

	if !result.Valid {
		return fmt.Errorf("plan validation failed with %d error(s)", len(result.Errors))
	}

	return nil
}

// ScaffoldCommand generates code from a plan file
func ScaffoldCommand(planFile, outputDir string, overwrite bool) error {
	// Load plan from JSON file
	data, err := os.ReadFile(planFile)
	if err != nil {
		return fmt.Errorf("failed to read plan file: %w", err)
	}

	plan, err := schema.PlanFromJSON(data)
	if err != nil {
		return fmt.Errorf("failed to parse plan: %w", err)
	}

	// Configure options
	opts := planning.DefaultScaffoldOptions(outputDir)
	opts.OverwriteFiles = overwrite

	// Generate scaffolding
	result, err := planning.ScaffoldFromPlan(plan, opts)
	if err != nil {
		return fmt.Errorf("scaffolding failed: %w", err)
	}

	// Print result
	planning.PrintScaffoldSummary(result)

	if !result.Success {
		return fmt.Errorf("scaffolding failed: %s", result.ErrorMessage)
	}

	return nil
}

// printValidationResult prints a colorized validation result
func printValidationResult(result *planning.ValidationResult) {
	if result.Valid {
		fmt.Printf("✅ Plan is valid!\n\n")
	} else {
		fmt.Printf("❌ Plan validation failed\n\n")
	}

	// Print errors
	if len(result.Errors) > 0 {
		fmt.Printf("Errors (%d):\n", len(result.Errors))
		for i, e := range result.Errors {
			fmt.Printf("  %d. [%s] %s\n", i+1, e.Code, e.Message)
			fmt.Printf("     Location: %s\n", e.Location)
		}
		fmt.Println()
	}

	// Print warnings
	if len(result.Warnings) > 0 {
		fmt.Printf("Warnings (%d):\n", len(result.Warnings))
		for i, w := range result.Warnings {
			fmt.Printf("  %d. [%s] %s\n", i+1, w.Code, w.Message)
			fmt.Printf("     Location: %s\n", w.Location)
		}
		fmt.Println()
	}

	// If valid, print summary
	if result.Valid {
		fmt.Println("✅ Ready to scaffold!")
		fmt.Println("   Run: :scaffold --from-plan <plan.json> --output <dir>")
	}
}

// ParseProposeCommand parses the :propose command arguments
// Format: :propose <filename.json>
func ParseProposeCommand(input string) (string, error) {
	parts := strings.Fields(input)
	if len(parts) < 2 {
		return "", fmt.Errorf("usage: :propose <plan.json>")
	}

	filename := parts[1]
	if !strings.HasSuffix(filename, ".json") {
		return "", fmt.Errorf("plan file must be a .json file")
	}

	return filename, nil
}

// ParseScaffoldCommand parses the :scaffold command arguments
// Format: :scaffold --from-plan <plan.json> [--output <dir>] [--overwrite]
func ParseScaffoldCommand(input string) (planFile, outputDir string, overwrite bool, err error) {
	parts := strings.Fields(input)

	// Default values
	outputDir = "./generated"
	overwrite = false

	// Parse flags
	i := 1 // Skip ":scaffold"
	for i < len(parts) {
		switch parts[i] {
		case "--from-plan":
			if i+1 >= len(parts) {
				return "", "", false, fmt.Errorf("--from-plan requires a filename")
			}
			planFile = parts[i+1]
			i += 2

		case "--output":
			if i+1 >= len(parts) {
				return "", "", false, fmt.Errorf("--output requires a directory")
			}
			outputDir = parts[i+1]
			i += 2

		case "--overwrite":
			overwrite = true
			i++

		default:
			return "", "", false, fmt.Errorf("unknown flag: %s", parts[i])
		}
	}

	if planFile == "" {
		return "", "", false, fmt.Errorf("usage: :scaffold --from-plan <plan.json> [--output <dir>] [--overwrite]")
	}

	if !strings.HasSuffix(planFile, ".json") {
		return "", "", false, fmt.Errorf("plan file must be a .json file")
	}

	return planFile, outputDir, overwrite, nil
}

// PrintPlanHelp prints help text for planning commands
func PrintPlanHelp() {
	fmt.Println("Planning Commands:")
	fmt.Println("  :propose <plan.json>")
	fmt.Println("      Validate an architecture plan before coding")
	fmt.Println()
	fmt.Println("  :scaffold --from-plan <plan.json> [--output <dir>] [--overwrite]")
	fmt.Println("      Generate module stubs from a validated plan")
	fmt.Println("      Options:")
	fmt.Println("        --output <dir>   Output directory (default: ./generated)")
	fmt.Println("        --overwrite      Overwrite existing files")
	fmt.Println()
	fmt.Println("Example plan structure:")
	fmt.Println(`{
  "schema": "ailang.plan/v1",
  "goal": "Build a REST API",
  "modules": [{
    "path": "api/core",
    "exports": ["handleRequest"],
    "imports": ["std/io"]
  }],
  "types": [{
    "name": "Request",
    "kind": "record",
    "definition": "{url: string, method: string}",
    "module": "api/core"
  }],
  "functions": [{
    "name": "handleRequest",
    "type": "(Request) -> () ! {IO}",
    "effects": ["IO"],
    "module": "api/core"
  }],
  "effects": ["IO"]
}`)
	fmt.Println()
}

// CreateExamplePlan creates an example plan and returns it as JSON
func CreateExamplePlan() (string, error) {
	plan := schema.NewPlan("Example REST API")
	plan.AddModule("api/core", []string{"handleRequest"}, []string{"std/io"})
	plan.AddType("Request", "record", "{url: string, method: string}", "api/core")
	plan.AddFunction("handleRequest", "(Request) -> () ! {IO}", "api/core", []string{"IO"})
	plan.AddEffect("IO")

	data, err := plan.ToJSON()
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// SaveExamplePlan saves an example plan to a file
func SaveExamplePlan(filename string) error {
	planJSON, err := CreateExamplePlan()
	if err != nil {
		return fmt.Errorf("failed to create example plan: %w", err)
	}

	if err := os.WriteFile(filename, []byte(planJSON), 0644); err != nil {
		return fmt.Errorf("failed to write plan file: %w", err)
	}

	fmt.Printf("✅ Example plan saved to: %s\n", filename)
	return nil
}

// ValidatePlanJSON validates raw JSON against the plan schema
func ValidatePlanJSON(jsonData []byte) error {
	var raw map[string]interface{}
	if err := json.Unmarshal(jsonData, &raw); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	// Check schema field
	schemaField, ok := raw["schema"].(string)
	if !ok {
		return fmt.Errorf("missing or invalid 'schema' field")
	}

	if schemaField != schema.PlanV1 {
		return fmt.Errorf("invalid schema version: %s (expected %s)", schemaField, schema.PlanV1)
	}

	// Check required fields
	requiredFields := []string{"goal", "modules", "types", "functions"}
	for _, field := range requiredFields {
		if _, ok := raw[field]; !ok {
			return fmt.Errorf("missing required field: %s", field)
		}
	}

	return nil
}
