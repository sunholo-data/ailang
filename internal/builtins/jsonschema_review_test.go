package builtins

import "testing"

func TestJSONSchemaDynamicResourceUnvisitedAnchor(t *testing.T) {
	schema := `{"$defs":{"override":{"$dynamicAnchor":"node","type":"integer"},"inner":{"$id":"inner","$defs":{"base":{"$dynamicAnchor":"node","type":"string"}},"$dynamicRef":"#node"}},"$ref":"#/$defs/inner"}`
	p, errs := compileSchema(engineJSON(t, schema), true)
	if len(errs) > 0 {
		t.Fatal(errs)
	}
	if errs := validateCompiledSchema(p, float64(1)); len(errs) > 0 {
		t.Fatalf("outer resource's dynamic anchor ignored: %v", errs)
	}
	if errs := validateCompiledSchema(p, "x"); len(errs) == 0 {
		t.Fatal("inner anchor incorrectly overrode outer resource anchor")
	}
}

func TestJSONSchemaDynamicPointerRemainsStatic(t *testing.T) {
	schema := `{"$dynamicAnchor":"node","type":"object","properties":{"x":{"$id":"child","$defs":{"base":{"$dynamicAnchor":"node","type":"string"}},"$dynamicRef":"#/$defs/base"}}}`
	p, errs := compileSchema(engineJSON(t, schema), true)
	if len(errs) > 0 {
		t.Fatal(errs)
	}
	if errs := validateCompiledSchema(p, map[string]any{"x": "ok"}); len(errs) > 0 {
		t.Fatalf("pointer dynamicRef incorrectly rebound: %v", errs)
	}
}
