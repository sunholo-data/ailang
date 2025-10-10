// Package planning provides code scaffolding from validated plans.
package planning

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sunholo/ailang/internal/schema"
)

// ScaffoldResult contains information about the scaffolding operation
type ScaffoldResult struct {
	OutputDir    string   `json:"output_dir"`
	FilesCreated []string `json:"files_created"`
	TotalLines   int      `json:"total_lines"`
	TotalFiles   int      `json:"total_files"`
	Success      bool     `json:"success"`
	ErrorMessage string   `json:"error_message,omitempty"`
}

// ScaffoldOptions configures the scaffolding behavior
type ScaffoldOptions struct {
	OutputDir       string
	OverwriteFiles  bool
	IncludeTODOs    bool
	IncludeComments bool
}

// DefaultScaffoldOptions returns sensible defaults
func DefaultScaffoldOptions(outputDir string) *ScaffoldOptions {
	return &ScaffoldOptions{
		OutputDir:       outputDir,
		OverwriteFiles:  false,
		IncludeTODOs:    true,
		IncludeComments: true,
	}
}

// ScaffoldFromPlan generates AILANG module files from a validated plan
func ScaffoldFromPlan(plan *schema.Plan, opts *ScaffoldOptions) (*ScaffoldResult, error) {
	if plan == nil {
		return nil, fmt.Errorf("plan is nil")
	}

	if opts == nil {
		opts = DefaultScaffoldOptions("./generated")
	}

	result := &ScaffoldResult{
		OutputDir:    opts.OutputDir,
		FilesCreated: []string{},
		Success:      false,
	}

	// Validate plan first
	validation, err := ValidatePlan(plan)
	if err != nil {
		return nil, fmt.Errorf("validation check failed: %w", err)
	}

	if !validation.Valid {
		return nil, fmt.Errorf("plan validation failed with %d errors", len(validation.Errors))
	}

	// Create output directory
	if err := os.MkdirAll(opts.OutputDir, 0755); err != nil {
		result.ErrorMessage = fmt.Sprintf("failed to create output directory: %v", err)
		return result, err
	}

	// Group types and functions by module
	moduleTypes := make(map[string][]schema.TypePlan)
	moduleFuncs := make(map[string][]schema.FuncPlan)

	for _, typ := range plan.Types {
		moduleTypes[typ.Module] = append(moduleTypes[typ.Module], typ)
	}

	for _, fn := range plan.Functions {
		moduleFuncs[fn.Module] = append(moduleFuncs[fn.Module], fn)
	}

	// Generate a file for each module
	for _, mod := range plan.Modules {
		types := moduleTypes[mod.Path]
		funcs := moduleFuncs[mod.Path]

		code, lineCount := GenerateModuleFile(mod, types, funcs, opts)

		// Determine output file path
		filePath := filepath.Join(opts.OutputDir, mod.Path+".ail")
		fileDir := filepath.Dir(filePath)

		// Create subdirectories if needed
		if err := os.MkdirAll(fileDir, 0755); err != nil {
			result.ErrorMessage = fmt.Sprintf("failed to create directory %s: %v", fileDir, err)
			return result, err
		}

		// Check if file exists
		if !opts.OverwriteFiles {
			if _, err := os.Stat(filePath); err == nil {
				result.ErrorMessage = fmt.Sprintf("file already exists: %s (use --overwrite to replace)", filePath)
				return result, fmt.Errorf("file exists: %s", filePath)
			}
		}

		// Write file
		if err := os.WriteFile(filePath, []byte(code), 0644); err != nil {
			result.ErrorMessage = fmt.Sprintf("failed to write file %s: %v", filePath, err)
			return result, err
		}

		result.FilesCreated = append(result.FilesCreated, filePath)
		result.TotalLines += lineCount
		result.TotalFiles++
	}

	result.Success = true
	return result, nil
}

// GenerateModuleFile generates complete AILANG code for a module
func GenerateModuleFile(mod schema.ModulePlan, types []schema.TypePlan, funcs []schema.FuncPlan, opts *ScaffoldOptions) (string, int) {
	var sb strings.Builder
	lineCount := 0

	// Module declaration
	sb.WriteString(fmt.Sprintf("module %s\n\n", mod.Path))
	lineCount += 2

	// Imports
	if len(mod.Imports) > 0 {
		for _, imp := range mod.Imports {
			sb.WriteString(fmt.Sprintf("import %s\n", imp))
			lineCount++
		}
		sb.WriteString("\n")
		lineCount++
	}

	// Comment about scaffolded code
	if opts.IncludeComments {
		sb.WriteString("-- This module was scaffolded from a plan\n")
		sb.WriteString("-- TODO: Implement the functions below\n\n")
		lineCount += 3
	}

	// Type definitions
	if len(types) > 0 {
		if opts.IncludeComments {
			sb.WriteString("-- Type Definitions\n\n")
			lineCount += 2
		}

		for _, typ := range types {
			code, lines := GenerateTypeDecl(typ, opts)
			sb.WriteString(code)
			sb.WriteString("\n")
			lineCount += lines + 1
		}
	}

	// Function stubs
	if len(funcs) > 0 {
		if opts.IncludeComments {
			sb.WriteString("-- Functions\n\n")
			lineCount += 2
		}

		for _, fn := range funcs {
			code, lines := GenerateFuncStub(fn, opts)
			sb.WriteString(code)
			sb.WriteString("\n")
			lineCount += lines + 1
		}
	}

	return sb.String(), lineCount
}

// GenerateTypeDecl generates an AILANG type declaration
func GenerateTypeDecl(typ schema.TypePlan, opts *ScaffoldOptions) (string, int) {
	var sb strings.Builder
	lineCount := 0

	if opts.IncludeComments {
		sb.WriteString(fmt.Sprintf("-- Type: %s (%s)\n", typ.Name, typ.Kind))
		lineCount++
	}

	switch typ.Kind {
	case "adt":
		// ADT: type Name = Ctor1 | Ctor2
		sb.WriteString(fmt.Sprintf("type %s = %s\n", typ.Name, typ.Definition))
		lineCount++

	case "record":
		// Record: type Name = {field1: Type1, field2: Type2}
		sb.WriteString(fmt.Sprintf("type %s = %s\n", typ.Name, typ.Definition))
		lineCount++

	case "alias":
		// Type alias: type Name = ExistingType
		sb.WriteString(fmt.Sprintf("type %s = %s\n", typ.Name, typ.Definition))
		lineCount++

	default:
		// Fallback
		sb.WriteString(fmt.Sprintf("type %s = %s\n", typ.Name, typ.Definition))
		lineCount++
	}

	return sb.String(), lineCount
}

// GenerateFuncStub generates a function stub with signature and TODO body
func GenerateFuncStub(fn schema.FuncPlan, opts *ScaffoldOptions) (string, int) {
	var sb strings.Builder
	lineCount := 0

	if opts.IncludeComments {
		sb.WriteString(fmt.Sprintf("-- Function: %s\n", fn.Name))
		lineCount++
		if len(fn.Effects) > 0 {
			sb.WriteString(fmt.Sprintf("-- Effects: %s\n", strings.Join(fn.Effects, ", ")))
			lineCount++
		}
	}

	// Parse type signature to extract args and return type
	// For simplicity, we'll generate a basic function stub
	// More sophisticated parsing would extract argument names and types

	effectAnnotation := ""
	if len(fn.Effects) > 0 {
		effectAnnotation = fmt.Sprintf(" ! {%s}", strings.Join(fn.Effects, ", "))
	}

	// Check if type signature is a simple function arrow
	// E.g., "(int, string) -> Result[int]"
	// For scaffolding, we'll generate a simple body

	// Determine if function should be exported
	// (In practice, check against module exports, but for now assume yes)
	exportKeyword := "export "

	// Generate function with placeholder body
	sb.WriteString(fmt.Sprintf("%sfunc %s: %s%s {\n", exportKeyword, fn.Name, fn.Type, effectAnnotation))
	lineCount++

	if opts.IncludeTODOs {
		sb.WriteString(fmt.Sprintf("  -- TODO: Implement %s\n", fn.Name))
		lineCount++
		sb.WriteString(fmt.Sprintf("  -- Signature: %s%s\n", fn.Type, effectAnnotation))
		lineCount++
	}

	// Add placeholder return value based on type
	returnPlaceholder := getReturnPlaceholder(fn.Type)
	sb.WriteString(fmt.Sprintf("  %s\n", returnPlaceholder))
	lineCount++

	sb.WriteString("}\n")
	lineCount++

	return sb.String(), lineCount
}

// getReturnPlaceholder generates a placeholder return value based on type signature
func getReturnPlaceholder(typeSignature string) string {
	// Very basic heuristic - look for return type
	if strings.Contains(typeSignature, "-> ()") {
		return "()  -- Unit return"
	}
	if strings.Contains(typeSignature, "-> int") {
		return "0  -- Placeholder int"
	}
	if strings.Contains(typeSignature, "-> string") {
		return "\"\"  -- Placeholder string"
	}
	if strings.Contains(typeSignature, "-> bool") {
		return "false  -- Placeholder bool"
	}
	if strings.Contains(typeSignature, "-> Option") {
		return "None  -- Placeholder Option"
	}
	if strings.Contains(typeSignature, "-> Result") {
		return "Err(\"not implemented\")  -- Placeholder Result"
	}
	if strings.Contains(typeSignature, "-> [") {
		return "[]  -- Empty list"
	}

	// Default fallback
	return "()  -- TODO: Return appropriate value"
}

// Helper: Check if function is in module exports (currently unused, reserved for future use)
// func isFunctionExported(fnName string, module schema.ModulePlan) bool {
// 	for _, exp := range module.Exports {
// 		if exp == fnName {
// 			return true
// 		}
// 	}
// 	return false
// }

// ValidatePlanForScaffolding performs additional checks before scaffolding
func ValidatePlanForScaffolding(plan *schema.Plan) error {
	validation, err := ValidatePlan(plan)
	if err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	if !validation.Valid {
		var errMsgs []string
		for _, e := range validation.Errors {
			errMsgs = append(errMsgs, fmt.Sprintf("[%s] %s (at %s)", e.Code, e.Message, e.Location))
		}
		return fmt.Errorf("plan has validation errors:\n%s", strings.Join(errMsgs, "\n"))
	}

	return nil
}

// PrintScaffoldSummary prints a human-readable summary of scaffolding
func PrintScaffoldSummary(result *ScaffoldResult) {
	if result.Success {
		fmt.Printf("✅ Scaffolding successful!\n\n")
		fmt.Printf("Output directory: %s\n", result.OutputDir)
		fmt.Printf("Files created: %d\n", result.TotalFiles)
		fmt.Printf("Total lines: %d\n\n", result.TotalLines)

		if len(result.FilesCreated) > 0 {
			fmt.Println("Generated files:")
			for _, file := range result.FilesCreated {
				fmt.Printf("  - %s\n", file)
			}
		}
	} else {
		fmt.Printf("❌ Scaffolding failed: %s\n", result.ErrorMessage)
	}
}
