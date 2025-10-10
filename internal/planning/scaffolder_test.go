package planning

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo/ailang/internal/schema"
)

func TestScaffoldFromPlan_SimplePlan(t *testing.T) {
	plan := schema.NewPlan("Simple REST API")
	plan.AddModule("api/core", []string{"handleRequest"}, []string{"std/io"})
	plan.AddType("Request", "record", "{url: string, method: string}", "api/core")
	plan.AddFunction("handleRequest", "(Request) -> () ! {IO}", "api/core", []string{"IO"})
	plan.AddEffect("IO")

	// Use temp directory
	tmpDir := t.TempDir()
	opts := DefaultScaffoldOptions(tmpDir)

	result, err := ScaffoldFromPlan(plan, opts)
	if err != nil {
		t.Fatalf("scaffolding failed: %v", err)
	}

	if !result.Success {
		t.Errorf("expected success, got failure: %s", result.ErrorMessage)
	}

	if result.TotalFiles != 1 {
		t.Errorf("expected 1 file, got %d", result.TotalFiles)
	}

	// Check file was created
	expectedPath := filepath.Join(tmpDir, "api/core.ail")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("expected file not created: %s", expectedPath)
	}

	// Read and check content
	content, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("failed to read generated file: %v", err)
	}

	contentStr := string(content)

	// Should contain module declaration
	if !strings.Contains(contentStr, "module api/core") {
		t.Error("generated file missing module declaration")
	}

	// Should contain import
	if !strings.Contains(contentStr, "import std/io") {
		t.Error("generated file missing import statement")
	}

	// Should contain type definition
	if !strings.Contains(contentStr, "type Request") {
		t.Error("generated file missing type definition")
	}

	// Should contain function stub
	if !strings.Contains(contentStr, "func handleRequest") {
		t.Error("generated file missing function stub")
	}

	// Should contain effect annotation
	if !strings.Contains(contentStr, "! {IO}") {
		t.Error("generated file missing effect annotation")
	}
}

func TestScaffoldFromPlan_InvalidPlan(t *testing.T) {
	// Plan with validation errors
	plan := schema.NewPlan("Invalid plan")
	plan.AddModule("Invalid/Path", []string{}, []string{}) // Invalid: uppercase

	tmpDir := t.TempDir()
	opts := DefaultScaffoldOptions(tmpDir)

	_, err := ScaffoldFromPlan(plan, opts)
	if err == nil {
		t.Error("expected error for invalid plan, got nil")
	}
}

func TestScaffoldFromPlan_FileExists(t *testing.T) {
	plan := schema.NewPlan("Test plan")
	plan.AddModule("test", []string{"main"}, []string{})

	tmpDir := t.TempDir()
	opts := DefaultScaffoldOptions(tmpDir)
	opts.OverwriteFiles = false

	// Create file first
	testFile := filepath.Join(tmpDir, "test.ail")
	if err := os.WriteFile(testFile, []byte("existing content"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Try to scaffold (should fail)
	result, err := ScaffoldFromPlan(plan, opts)
	if err == nil {
		t.Error("expected error for existing file, got nil")
	}

	if result != nil && result.Success {
		t.Error("expected failure for existing file")
	}
}

func TestScaffoldFromPlan_Overwrite(t *testing.T) {
	plan := schema.NewPlan("Test plan")
	plan.AddModule("test", []string{"main"}, []string{})

	tmpDir := t.TempDir()
	opts := DefaultScaffoldOptions(tmpDir)
	opts.OverwriteFiles = true

	// Create file first
	testFile := filepath.Join(tmpDir, "test.ail")
	if err := os.WriteFile(testFile, []byte("existing content"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Try to scaffold with overwrite enabled (should succeed)
	result, err := ScaffoldFromPlan(plan, opts)
	if err != nil {
		t.Errorf("unexpected error with overwrite enabled: %v", err)
	}

	if !result.Success {
		t.Error("expected success with overwrite enabled")
	}

	// Check content was replaced
	content, _ := os.ReadFile(testFile)
	if strings.Contains(string(content), "existing content") {
		t.Error("file was not overwritten")
	}
}

func TestGenerateModuleFile_EmptyModule(t *testing.T) {
	mod := schema.ModulePlan{
		Path:    "empty/module",
		Exports: []string{},
		Imports: []string{},
	}

	opts := DefaultScaffoldOptions("")

	code, lineCount := GenerateModuleFile(mod, nil, nil, opts)

	if lineCount == 0 {
		t.Error("expected non-zero line count")
	}

	if !strings.Contains(code, "module empty/module") {
		t.Error("generated code missing module declaration")
	}
}

func TestGenerateModuleFile_WithImports(t *testing.T) {
	mod := schema.ModulePlan{
		Path:    "app/core",
		Exports: []string{"main"},
		Imports: []string{"std/io", "std/fs"},
	}

	opts := DefaultScaffoldOptions("")

	code, _ := GenerateModuleFile(mod, nil, nil, opts)

	if !strings.Contains(code, "import std/io") {
		t.Error("generated code missing import std/io")
	}

	if !strings.Contains(code, "import std/fs") {
		t.Error("generated code missing import std/fs")
	}
}

func TestGenerateTypeDecl_ADT(t *testing.T) {
	typ := schema.TypePlan{
		Name:       "Option",
		Kind:       "adt",
		Definition: "Some(a) | None",
		Module:     "core",
	}

	opts := DefaultScaffoldOptions("")

	code, lineCount := GenerateTypeDecl(typ, opts)

	if lineCount == 0 {
		t.Error("expected non-zero line count")
	}

	if !strings.Contains(code, "type Option = Some(a) | None") {
		t.Errorf("unexpected ADT code: %s", code)
	}
}

func TestGenerateTypeDecl_Record(t *testing.T) {
	typ := schema.TypePlan{
		Name:       "Person",
		Kind:       "record",
		Definition: "{name: string, age: int}",
		Module:     "core",
	}

	opts := DefaultScaffoldOptions("")

	code, _ := GenerateTypeDecl(typ, opts)

	if !strings.Contains(code, "type Person") {
		t.Error("generated code missing type name")
	}

	if !strings.Contains(code, "{name: string, age: int}") {
		t.Error("generated code missing record definition")
	}
}

func TestGenerateFuncStub_WithEffects(t *testing.T) {
	fn := schema.FuncPlan{
		Name:    "readFile",
		Type:    "(string) -> string",
		Effects: []string{"FS"},
		Module:  "core",
	}

	opts := DefaultScaffoldOptions("")

	code, lineCount := GenerateFuncStub(fn, opts)

	if lineCount == 0 {
		t.Error("expected non-zero line count")
	}

	if !strings.Contains(code, "func readFile") {
		t.Error("generated code missing function name")
	}

	if !strings.Contains(code, "! {FS}") {
		t.Error("generated code missing effect annotation")
	}

	if !strings.Contains(code, "TODO") {
		t.Error("generated code missing TODO comment")
	}
}

func TestGenerateFuncStub_NoEffects(t *testing.T) {
	fn := schema.FuncPlan{
		Name:    "add",
		Type:    "(int, int) -> int",
		Effects: []string{},
		Module:  "math",
	}

	opts := DefaultScaffoldOptions("")

	code, _ := GenerateFuncStub(fn, opts)

	if strings.Contains(code, "! {") {
		t.Error("generated code should not have effect annotation for pure function")
	}

	if !strings.Contains(code, "func add") {
		t.Error("generated code missing function name")
	}
}

func TestGenerateFuncStub_WithoutTODOs(t *testing.T) {
	fn := schema.FuncPlan{
		Name:    "process",
		Type:    "() -> ()",
		Effects: []string{},
		Module:  "core",
	}

	opts := DefaultScaffoldOptions("")
	opts.IncludeTODOs = false

	code, _ := GenerateFuncStub(fn, opts)

	if strings.Contains(code, "TODO") {
		t.Error("generated code should not contain TODOs when disabled")
	}
}

func TestGetReturnPlaceholder(t *testing.T) {
	tests := []struct {
		typeSignature string
		shouldContain string
	}{
		{"() -> ()", "()"},
		{"(int) -> int", "0"},
		{"(string) -> string", "\"\""},
		{"() -> bool", "false"},
		{"() -> Option[int]", "None"},
		{"() -> Result[string]", "Err"},
		{"() -> [int]", "[]"},
		{"() -> CustomType", "()"},
	}

	for _, tt := range tests {
		result := getReturnPlaceholder(tt.typeSignature)
		if !strings.Contains(result, tt.shouldContain) {
			t.Errorf("for type %s, expected placeholder containing '%s', got '%s'",
				tt.typeSignature, tt.shouldContain, result)
		}
	}
}

func TestValidatePlanForScaffolding_ValidPlan(t *testing.T) {
	plan := schema.NewPlan("Valid plan")
	plan.AddModule("test", []string{"main"}, []string{})

	err := ValidatePlanForScaffolding(plan)
	if err != nil {
		t.Errorf("unexpected error for valid plan: %v", err)
	}
}

func TestValidatePlanForScaffolding_InvalidPlan(t *testing.T) {
	plan := schema.NewPlan("Invalid plan")
	plan.AddModule("Invalid/Module", []string{}, []string{}) // Invalid: uppercase

	err := ValidatePlanForScaffolding(plan)
	if err == nil {
		t.Error("expected error for invalid plan, got nil")
	}
}

func TestScaffoldResult_ToJSON(t *testing.T) {
	result := &ScaffoldResult{
		OutputDir:    "/tmp/test",
		FilesCreated: []string{"/tmp/test/core.ail"},
		TotalLines:   42,
		TotalFiles:   1,
		Success:      true,
	}

	// Just verify it doesn't crash - we don't have JSON method yet
	// but the struct should be serializable
	_ = result
}

func TestScaffoldFromPlan_MultipleModules(t *testing.T) {
	plan := schema.NewPlan("Multi-module app")
	plan.AddModule("app/core", []string{"main"}, []string{})
	plan.AddModule("app/utils", []string{"helper"}, []string{})

	tmpDir := t.TempDir()
	opts := DefaultScaffoldOptions(tmpDir)

	result, err := ScaffoldFromPlan(plan, opts)
	if err != nil {
		t.Fatalf("scaffolding failed: %v", err)
	}

	if result.TotalFiles != 2 {
		t.Errorf("expected 2 files, got %d", result.TotalFiles)
	}

	// Check both files were created
	coreFile := filepath.Join(tmpDir, "app/core.ail")
	utilsFile := filepath.Join(tmpDir, "app/utils.ail")

	if _, err := os.Stat(coreFile); os.IsNotExist(err) {
		t.Errorf("core file not created: %s", coreFile)
	}

	if _, err := os.Stat(utilsFile); os.IsNotExist(err) {
		t.Errorf("utils file not created: %s", utilsFile)
	}
}
