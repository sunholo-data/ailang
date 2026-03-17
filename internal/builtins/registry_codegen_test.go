package builtins

import "testing"

func TestCodegenSpecDirectLookup(t *testing.T) {
	spec := GetCodegenSpec("_str_trim")
	if spec == nil {
		t.Fatal("_str_trim should have GoCodegenSpec")
	}
	if spec.Inline == "" {
		t.Error("_str_trim should have Inline spec")
	}
	if spec.StdlibName != "trim" {
		t.Errorf("_str_trim StdlibName = %q, want 'trim'", spec.StdlibName)
	}
}

func TestCodegenSpecStdlibLookup(t *testing.T) {
	spec := GetCodegenSpecByStdlibName("trim")
	if spec == nil {
		t.Fatal("stdlib 'trim' should resolve to a GoCodegenSpec")
	}
	if spec.Inline == "" {
		t.Error("'trim' should have Inline spec")
	}
}

func TestCodegenSpecHelperLookup(t *testing.T) {
	spec := GetCodegenSpecByStdlibName("map")
	if spec == nil {
		t.Fatal("stdlib 'map' should resolve to a GoCodegenSpec")
	}
	if spec.Helper == nil {
		t.Fatal("'map' should have Helper spec")
	}
	if spec.Helper.FuncName != "Map" {
		t.Errorf("'map' Helper.FuncName = %q, want 'Map'", spec.Helper.FuncName)
	}
}

func TestCodegenSpecCoverage(t *testing.T) {
	withCodegen := 0
	for _, meta := range Registry {
		if meta.GoCodegen != nil {
			withCodegen++
		}
	}
	if withCodegen < 50 {
		t.Errorf("Expected at least 50 builtins with GoCodegenSpec, got %d", withCodegen)
	}

	if len(StdlibIndex) < 40 {
		t.Errorf("Expected at least 40 StdlibIndex entries, got %d", len(StdlibIndex))
	}
}

func TestStdlibIndexCompleteness(t *testing.T) {
	// Verify key stdlib functions are indexed
	required := []string{
		"trim", "split", "substring", "toUpper", "toLower",
		"map", "filter", "foldl", "reverse", "any",
		"sin", "cos", "sqrt", "pow",
		"println", "readFile", "writeFile",
		"decode", "encode", "get", "getString",
		"isNone", "isSome", "getOrElse",
		"parse", "findAll", "getText",
	}
	for _, name := range required {
		if _, ok := StdlibIndex[name]; !ok {
			t.Errorf("StdlibIndex missing required entry: %q", name)
		}
	}
}
