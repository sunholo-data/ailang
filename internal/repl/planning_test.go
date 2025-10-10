package repl

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseProposeCommand_Valid(t *testing.T) {
	filename, err := ParseProposeCommand(":propose myplan.json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if filename != "myplan.json" {
		t.Errorf("expected 'myplan.json', got '%s'", filename)
	}
}

func TestParseProposeCommand_MissingFile(t *testing.T) {
	_, err := ParseProposeCommand(":propose")
	if err == nil {
		t.Error("expected error for missing filename")
	}
}

func TestParseProposeCommand_NotJSON(t *testing.T) {
	_, err := ParseProposeCommand(":propose plan.txt")
	if err == nil {
		t.Error("expected error for non-JSON file")
	}
}

func TestParseScaffoldCommand_Valid(t *testing.T) {
	planFile, outputDir, overwrite, err := ParseScaffoldCommand(":scaffold --from-plan test.json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if planFile != "test.json" {
		t.Errorf("expected 'test.json', got '%s'", planFile)
	}

	if outputDir != "./generated" {
		t.Errorf("expected './generated', got '%s'", outputDir)
	}

	if overwrite {
		t.Error("expected overwrite to be false by default")
	}
}

func TestParseScaffoldCommand_WithOutput(t *testing.T) {
	planFile, outputDir, overwrite, err := ParseScaffoldCommand(":scaffold --from-plan test.json --output /tmp/output")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if planFile != "test.json" {
		t.Errorf("expected 'test.json', got '%s'", planFile)
	}

	if outputDir != "/tmp/output" {
		t.Errorf("expected '/tmp/output', got '%s'", outputDir)
	}

	if overwrite {
		t.Error("expected overwrite to be false")
	}
}

func TestParseScaffoldCommand_WithOverwrite(t *testing.T) {
	_, _, overwrite, err := ParseScaffoldCommand(":scaffold --from-plan test.json --overwrite")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !overwrite {
		t.Error("expected overwrite to be true")
	}
}

func TestParseScaffoldCommand_MissingPlanFile(t *testing.T) {
	_, _, _, err := ParseScaffoldCommand(":scaffold")
	if err == nil {
		t.Error("expected error for missing --from-plan")
	}
}

func TestParseScaffoldCommand_UnknownFlag(t *testing.T) {
	_, _, _, err := ParseScaffoldCommand(":scaffold --from-plan test.json --unknown-flag")
	if err == nil {
		t.Error("expected error for unknown flag")
	}
}

func TestCreateExamplePlan(t *testing.T) {
	planJSON, err := CreateExamplePlan()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(planJSON) == 0 {
		t.Error("expected non-empty plan JSON")
	}

	// Should be valid JSON
	if planJSON[0] != '{' {
		t.Error("expected JSON to start with '{'")
	}
}

func TestSaveExamplePlan(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "example_plan.json")

	err := SaveExamplePlan(tmpFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check file was created
	if _, err := os.Stat(tmpFile); os.IsNotExist(err) {
		t.Error("example plan file was not created")
	}

	// Read and verify content
	content, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	if len(content) == 0 {
		t.Error("expected non-empty file content")
	}
}

func TestValidatePlanJSON_Valid(t *testing.T) {
	validJSON := []byte(`{
		"schema": "ailang.plan/v1",
		"goal": "Test",
		"modules": [],
		"types": [],
		"functions": []
	}`)

	err := ValidatePlanJSON(validJSON)
	if err != nil {
		t.Errorf("unexpected error for valid JSON: %v", err)
	}
}

func TestValidatePlanJSON_InvalidJSON(t *testing.T) {
	invalidJSON := []byte(`{not valid json}`)

	err := ValidatePlanJSON(invalidJSON)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestValidatePlanJSON_MissingSchema(t *testing.T) {
	missingSchema := []byte(`{
		"goal": "Test",
		"modules": []
	}`)

	err := ValidatePlanJSON(missingSchema)
	if err == nil {
		t.Error("expected error for missing schema field")
	}
}

func TestValidatePlanJSON_WrongSchema(t *testing.T) {
	wrongSchema := []byte(`{
		"schema": "wrong.schema/v1",
		"goal": "Test",
		"modules": [],
		"types": [],
		"functions": []
	}`)

	err := ValidatePlanJSON(wrongSchema)
	if err == nil {
		t.Error("expected error for wrong schema version")
	}
}

func TestValidatePlanJSON_MissingRequiredFields(t *testing.T) {
	missingFields := []byte(`{
		"schema": "ailang.plan/v1",
		"goal": "Test"
	}`)

	err := ValidatePlanJSON(missingFields)
	if err == nil {
		t.Error("expected error for missing required fields")
	}
}
