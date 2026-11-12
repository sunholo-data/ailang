package repl

import (
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
