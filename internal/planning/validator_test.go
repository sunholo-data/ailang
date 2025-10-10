package planning

import (
	"testing"

	"github.com/sunholo/ailang/internal/schema"
)

func TestValidatePlan_EmptyPlan(t *testing.T) {
	plan := schema.NewPlan("Empty test")

	result, err := ValidatePlan(plan)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Valid {
		t.Error("expected invalid result for empty plan")
	}

	if len(result.Errors) == 0 {
		t.Error("expected at least one error for empty plan")
	}

	// Check for VAL_G01 error
	found := false
	for _, e := range result.Errors {
		if e.Code == VAL_G01 {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected VAL_G01 error for empty plan")
	}
}

func TestValidatePlan_ValidSimplePlan(t *testing.T) {
	plan := schema.NewPlan("Simple valid plan")
	plan.AddModule("app/core", []string{"main"}, []string{"std/io"})
	plan.AddType("Option", "adt", "Some(a) | None", "app/core")
	plan.AddFunction("main", "() -> () ! {IO}", "app/core", []string{"IO"})
	plan.AddEffect("IO")

	result, err := ValidatePlan(plan)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Valid {
		t.Errorf("expected valid result, got errors: %v", result.Errors)
	}

	if len(result.Errors) > 0 {
		t.Errorf("expected no errors, got: %v", result.Errors)
	}
}

func TestCheckModulePaths_InvalidPath(t *testing.T) {
	modules := []schema.ModulePlan{
		{Path: "Foo/Bar", Exports: []string{"test"}},           // Invalid: uppercase
		{Path: "foo/bar/", Exports: []string{"test"}},          // Invalid: trailing slash
		{Path: "foo-bar", Exports: []string{"test"}},           // Invalid: dash
		{Path: "foo/bar", Exports: []string{"test"}},           // Valid
		{Path: "valid_name", Exports: []string{"test"}},        // Valid
	}

	issues := CheckModulePaths(modules)

	// Should have 3 errors (first 3 modules)
	errorCount := 0
	for _, issue := range issues {
		if issue.Level == ValidationError && issue.Code == VAL_M01 {
			errorCount++
		}
	}

	if errorCount != 3 {
		t.Errorf("expected 3 path errors, got %d", errorCount)
	}
}

func TestCheckModulePaths_DuplicatePaths(t *testing.T) {
	modules := []schema.ModulePlan{
		{Path: "app/core", Exports: []string{"main"}},
		{Path: "app/utils", Exports: []string{"helper"}},
		{Path: "app/core", Exports: []string{"other"}}, // Duplicate
	}

	issues := CheckModulePaths(modules)

	found := false
	for _, issue := range issues {
		if issue.Code == VAL_M03 {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected VAL_M03 error for duplicate module paths")
	}
}

func TestCheckModulePaths_NoExports(t *testing.T) {
	modules := []schema.ModulePlan{
		{Path: "app/core", Exports: []string{}}, // No exports - should warn
	}

	issues := CheckModulePaths(modules)

	found := false
	for _, issue := range issues {
		if issue.Code == VAL_M04 && issue.Level == ValidationWarning {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected VAL_M04 warning for module with no exports")
	}
}

func TestCheckTypeDefinitions_InvalidNames(t *testing.T) {
	types := []schema.TypePlan{
		{Name: "option", Kind: "adt", Definition: "Some | None", Module: "core"},   // Invalid: lowercase
		{Name: "Result", Kind: "adt", Definition: "Ok | Err", Module: "core"},      // Valid
		{Name: "my_type", Kind: "record", Definition: "{x: int}", Module: "core"},  // Invalid: underscore
	}

	modules := []schema.ModulePlan{
		{Path: "core", Exports: []string{}},
	}

	issues := CheckTypeDefinitions(types, modules)

	errorCount := 0
	for _, issue := range issues {
		if issue.Code == VAL_T01 {
			errorCount++
		}
	}

	if errorCount != 2 {
		t.Errorf("expected 2 type name errors, got %d", errorCount)
	}
}

func TestCheckTypeDefinitions_InvalidKind(t *testing.T) {
	types := []schema.TypePlan{
		{Name: "Option", Kind: "union", Definition: "Some | None", Module: "core"}, // Invalid kind
	}

	modules := []schema.ModulePlan{
		{Path: "core", Exports: []string{}},
	}

	issues := CheckTypeDefinitions(types, modules)

	found := false
	for _, issue := range issues {
		if issue.Code == VAL_T02 {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected VAL_T02 error for invalid type kind")
	}
}

func TestCheckTypeDefinitions_ModuleNotFound(t *testing.T) {
	types := []schema.TypePlan{
		{Name: "Option", Kind: "adt", Definition: "Some | None", Module: "nonexistent"},
	}

	modules := []schema.ModulePlan{
		{Path: "core", Exports: []string{}},
	}

	issues := CheckTypeDefinitions(types, modules)

	found := false
	for _, issue := range issues {
		if issue.Code == VAL_T05 {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected VAL_T05 error for undefined module")
	}
}

func TestCheckTypeDefinitions_DuplicateNames(t *testing.T) {
	types := []schema.TypePlan{
		{Name: "Option", Kind: "adt", Definition: "Some | None", Module: "core"},
		{Name: "Option", Kind: "adt", Definition: "Ok | Err", Module: "other"}, // Duplicate
	}

	modules := []schema.ModulePlan{
		{Path: "core", Exports: []string{}},
		{Path: "other", Exports: []string{}},
	}

	issues := CheckTypeDefinitions(types, modules)

	found := false
	for _, issue := range issues {
		if issue.Code == VAL_T04 {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected VAL_T04 error for duplicate type names")
	}
}

func TestCheckFunctionSignatures_InvalidNames(t *testing.T) {
	funcs := []schema.FuncPlan{
		{Name: "Process", Type: "() -> ()", Module: "core"},     // Invalid: uppercase
		{Name: "process", Type: "() -> ()", Module: "core"},     // Valid
		{Name: "do-thing", Type: "() -> ()", Module: "core"},    // Invalid: dash
	}

	modules := []schema.ModulePlan{
		{Path: "core", Exports: []string{"process"}},
	}

	issues := CheckFunctionSignatures(funcs, modules)

	errorCount := 0
	for _, issue := range issues {
		if issue.Code == VAL_F01 {
			errorCount++
		}
	}

	if errorCount != 2 {
		t.Errorf("expected 2 function name errors, got %d", errorCount)
	}
}

func TestCheckFunctionSignatures_NotExported(t *testing.T) {
	funcs := []schema.FuncPlan{
		{Name: "process", Type: "() -> ()", Module: "core"},
	}

	modules := []schema.ModulePlan{
		{Path: "core", Exports: []string{"other"}}, // process not in exports
	}

	issues := CheckFunctionSignatures(funcs, modules)

	found := false
	for _, issue := range issues {
		if issue.Code == VAL_F05 && issue.Level == ValidationWarning {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected VAL_F05 warning for function not in exports")
	}
}

func TestCheckFunctionSignatures_ModuleNotFound(t *testing.T) {
	funcs := []schema.FuncPlan{
		{Name: "process", Type: "() -> ()", Module: "nonexistent"},
	}

	modules := []schema.ModulePlan{
		{Path: "core", Exports: []string{}},
	}

	issues := CheckFunctionSignatures(funcs, modules)

	found := false
	for _, issue := range issues {
		if issue.Code == VAL_F04 {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected VAL_F04 error for undefined module")
	}
}

func TestCheckFunctionSignatures_DuplicateFunctions(t *testing.T) {
	funcs := []schema.FuncPlan{
		{Name: "process", Type: "() -> ()", Module: "core"},
		{Name: "process", Type: "(int) -> int", Module: "core"}, // Duplicate in same module
	}

	modules := []schema.ModulePlan{
		{Path: "core", Exports: []string{"process"}},
	}

	issues := CheckFunctionSignatures(funcs, modules)

	found := false
	for _, issue := range issues {
		if issue.Code == VAL_F03 {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected VAL_F03 error for duplicate function names")
	}
}

func TestCheckEffects_UnknownEffect(t *testing.T) {
	effects := []string{"IO", "CustomEffect"} // CustomEffect is invalid

	issues := CheckEffects(effects, []schema.FuncPlan{})

	found := false
	for _, issue := range issues {
		if issue.Code == VAL_E01 {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected VAL_E01 error for unknown effect")
	}
}

func TestCheckEffects_FunctionEffectNotInPlan(t *testing.T) {
	effects := []string{"IO"} // Only IO in plan

	funcs := []schema.FuncPlan{
		{Name: "read", Type: "() -> string", Module: "core", Effects: []string{"FS"}}, // Uses FS
	}

	issues := CheckEffects(effects, funcs)

	found := false
	for _, issue := range issues {
		if issue.Code == VAL_E02 && issue.Level == ValidationWarning {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected VAL_E02 warning for effect not in plan")
	}
}

func TestCheckDependencyCycles_SimpleCycle(t *testing.T) {
	modules := []schema.ModulePlan{
		{Path: "a", Imports: []string{"b"}},
		{Path: "b", Imports: []string{"a"}}, // Cycle: a -> b -> a
	}

	issues := CheckDependencyCycles(modules)

	found := false
	for _, issue := range issues {
		if issue.Code == VAL_M02 {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected VAL_M02 error for circular dependency")
	}
}

func TestCheckDependencyCycles_NoCycle(t *testing.T) {
	modules := []schema.ModulePlan{
		{Path: "a", Imports: []string{"c"}},
		{Path: "b", Imports: []string{"c"}},
		{Path: "c", Imports: []string{}},
	}

	issues := CheckDependencyCycles(modules)

	for _, issue := range issues {
		if issue.Code == VAL_M02 {
			t.Errorf("unexpected cycle detection: %v", issue)
		}
	}
}

func TestValidationResult_ToJSON(t *testing.T) {
	result := &ValidationResult{
		Schema: schema.PlanV1,
		Valid:  false,
		Errors: []ValidationIssue{
			{Level: ValidationError, Code: VAL_M01, Message: "Test error", Location: "modules[0]"},
		},
		Warnings: []ValidationIssue{
			{Level: ValidationWarning, Code: VAL_M04, Message: "Test warning", Location: "modules[1]"},
		},
	}

	data, err := result.ToJSON()
	if err != nil {
		t.Fatalf("failed to marshal result: %v", err)
	}

	if len(data) == 0 {
		t.Error("expected non-empty JSON output")
	}
}
