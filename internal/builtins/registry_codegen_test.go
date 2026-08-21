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
	spec := GetCodegenSpecByStdlibName("std/string", "trim")
	if spec == nil {
		t.Fatal("stdlib 'trim' should resolve to a GoCodegenSpec")
	}
	if spec.Inline == "" {
		t.Error("'trim' should have Inline spec")
	}
}

func TestCodegenSpecHelperLookup(t *testing.T) {
	spec := GetCodegenSpecByStdlibName("std/list", "map")
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

func TestQualifiedStdlibResolutionRejectsMeasuredCrossModuleCollisions(t *testing.T) {
	if spec := GetCodegenSpecByStdlibName("std/option", "map"); spec != nil {
		t.Fatalf("std/option.map resolved to codegen spec %+v; must not resolve to _list_map", spec)
	}
	if spec := GetCodegenSpecByStdlibName("std/list", "length"); spec != nil {
		t.Fatalf("std/list.length resolved to codegen spec %+v; must not resolve to _str_len", spec)
	}

	controls := []struct {
		module, name, builtin string
	}{
		{"std/string", "trim", "_str_trim"},
		{"std/list", "map", "_list_map"},
	}
	for _, control := range controls {
		key := StdlibKey{Module: control.module, Name: control.name}
		if spec := GetCodegenSpecByStdlibName(control.module, control.name); spec == nil {
			t.Errorf("positive control %s.%s did not resolve", control.module, control.name)
		} else if got := StdlibIndex[key]; got != control.builtin {
			t.Errorf("positive control %s.%s resolved to %q, want %q", control.module, control.name, got, control.builtin)
		}
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
	required := []StdlibKey{
		{"std/string", "trim"}, {"std/string", "split"}, {"std/string", "substring"}, {"std/string", "toUpper"}, {"std/string", "toLower"},
		{"std/list", "map"}, {"std/list", "filter"}, {"std/list", "foldl"}, {"std/list", "reverse"}, {"std/list", "any"},
		{"std/math", "sin"}, {"std/math", "cos"}, {"std/math", "sqrt"}, {"std/math", "pow"},
		{"std/io", "println"}, {"std/fs", "readFile"}, {"std/fs", "writeFile"},
		{"std/json", "decode"}, {"std/json", "encode"}, {"std/json", "get"}, {"std/json", "getString"},
		{"std/option", "isNone"}, {"std/option", "isSome"}, {"std/option", "getOrElse"},
		{"std/xml", "parse"}, {"std/xml", "findAll"}, {"std/xml", "getText"},
	}
	for _, key := range required {
		if _, ok := StdlibIndex[key]; !ok {
			t.Errorf("StdlibIndex missing required entry: %s.%s", key.Module, key.Name)
		}
	}
}
