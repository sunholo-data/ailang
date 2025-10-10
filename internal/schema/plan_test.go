package schema

import (
	"encoding/json"
	"testing"
)

func TestNewPlan(t *testing.T) {
	p := NewPlan("Test application")

	if p.Schema != PlanV1 {
		t.Errorf("expected schema %s, got %s", PlanV1, p.Schema)
	}

	if p.Goal != "Test application" {
		t.Errorf("expected goal 'Test application', got '%s'", p.Goal)
	}

	if len(p.Modules) != 0 || len(p.Types) != 0 || len(p.Functions) != 0 {
		t.Error("expected empty collections for new plan")
	}
}

func TestPlanJSON_RoundTrip(t *testing.T) {
	plan := NewPlan("Build a REST API")
	plan.AddModule("api/core", []string{"handleRequest"}, []string{"std/io"})
	plan.AddType("Request", "record", "{url: string, method: string}", "api/core")
	plan.AddFunction("handleRequest", "(Request) -> () ! {IO}", "api/core", []string{"IO"})
	plan.AddEffect("IO")

	// Marshal to JSON
	data, err := plan.ToJSON()
	if err != nil {
		t.Fatalf("failed to marshal plan: %v", err)
	}

	// Unmarshal back
	loaded, err := PlanFromJSON(data)
	if err != nil {
		t.Fatalf("failed to unmarshal plan: %v", err)
	}

	// Verify fields
	if loaded.Goal != plan.Goal {
		t.Errorf("goal mismatch: expected '%s', got '%s'", plan.Goal, loaded.Goal)
	}

	if len(loaded.Modules) != 1 {
		t.Errorf("expected 1 module, got %d", len(loaded.Modules))
	}

	if len(loaded.Types) != 1 {
		t.Errorf("expected 1 type, got %d", len(loaded.Types))
	}

	if len(loaded.Functions) != 1 {
		t.Errorf("expected 1 function, got %d", len(loaded.Functions))
	}

	if len(loaded.Effects) != 1 || loaded.Effects[0] != "IO" {
		t.Errorf("expected effects [IO], got %v", loaded.Effects)
	}
}

func TestPlanFromJSON_InvalidSchema(t *testing.T) {
	invalidJSON := `{"schema": "unknown.v99", "goal": "test"}`

	_, err := PlanFromJSON([]byte(invalidJSON))
	if err == nil {
		t.Error("expected error for invalid schema version")
	}
}

func TestPlanFromJSON_InvalidJSON(t *testing.T) {
	invalidJSON := `{this is not valid json}`

	_, err := PlanFromJSON([]byte(invalidJSON))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestAddModule(t *testing.T) {
	p := NewPlan("Test")
	p.AddModule("app/core", []string{"main"}, []string{"std/io"})

	if len(p.Modules) != 1 {
		t.Fatalf("expected 1 module, got %d", len(p.Modules))
	}

	m := p.Modules[0]
	if m.Path != "app/core" {
		t.Errorf("expected path 'app/core', got '%s'", m.Path)
	}

	if len(m.Exports) != 1 || m.Exports[0] != "main" {
		t.Errorf("expected exports [main], got %v", m.Exports)
	}

	if len(m.Imports) != 1 || m.Imports[0] != "std/io" {
		t.Errorf("expected imports [std/io], got %v", m.Imports)
	}
}

func TestAddType(t *testing.T) {
	p := NewPlan("Test")
	p.AddType("Option", "adt", "Some(a) | None", "core")

	if len(p.Types) != 1 {
		t.Fatalf("expected 1 type, got %d", len(p.Types))
	}

	tp := p.Types[0]
	if tp.Name != "Option" {
		t.Errorf("expected name 'Option', got '%s'", tp.Name)
	}

	if tp.Kind != "adt" {
		t.Errorf("expected kind 'adt', got '%s'", tp.Kind)
	}

	if tp.Module != "core" {
		t.Errorf("expected module 'core', got '%s'", tp.Module)
	}
}

func TestAddFunction(t *testing.T) {
	p := NewPlan("Test")
	p.AddFunction("process", "(string) -> int ! {IO}", "core", []string{"IO"})

	if len(p.Functions) != 1 {
		t.Fatalf("expected 1 function, got %d", len(p.Functions))
	}

	f := p.Functions[0]
	if f.Name != "process" {
		t.Errorf("expected name 'process', got '%s'", f.Name)
	}

	if f.Type != "(string) -> int ! {IO}" {
		t.Errorf("expected type '(string) -> int ! {IO}', got '%s'", f.Type)
	}

	if f.Module != "core" {
		t.Errorf("expected module 'core', got '%s'", f.Module)
	}

	if len(f.Effects) != 1 || f.Effects[0] != "IO" {
		t.Errorf("expected effects [IO], got %v", f.Effects)
	}
}

func TestAddEffect_NoDuplicates(t *testing.T) {
	p := NewPlan("Test")
	p.AddEffect("IO")
	p.AddEffect("FS")
	p.AddEffect("IO") // Duplicate

	if len(p.Effects) != 2 {
		t.Errorf("expected 2 unique effects, got %d: %v", len(p.Effects), p.Effects)
	}
}

func TestPlanJSONStructure(t *testing.T) {
	p := NewPlan("Simple plan")
	p.AddModule("app", []string{"main"}, []string{})
	p.AddType("Data", "record", "{value: int}", "app")

	data, err := p.ToJSON()
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	// Verify it's valid JSON
	var decoded map[string]interface{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}

	// Check required fields exist
	if _, ok := decoded["schema"]; !ok {
		t.Error("missing 'schema' field")
	}

	if _, ok := decoded["goal"]; !ok {
		t.Error("missing 'goal' field")
	}

	if _, ok := decoded["modules"]; !ok {
		t.Error("missing 'modules' field")
	}

	if _, ok := decoded["types"]; !ok {
		t.Error("missing 'types' field")
	}

	if _, ok := decoded["functions"]; !ok {
		t.Error("missing 'functions' field")
	}
}
