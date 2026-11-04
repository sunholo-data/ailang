package builtins

import (
	"encoding/json"
	"testing"
)

func engineJSON(t *testing.T, s string) any {
	t.Helper()
	var v any
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		t.Fatal(err)
	}
	return v
}
func TestJSONSchemaEngine(t *testing.T) {
	tests := []struct{ name, schema, good, bad string }{
		{"integer", `{"type":"integer"}`, `2.0`, `2.5`},
		{"union", `{"type":["string","null"]}`, `null`, `3`},
		{"enum", `{"enum":[{"a":1},2]}`, `{"a":1}`, `{"a":2}`},
		{"const", `{"const":[1,2]}`, `[1,2]`, `[2,1]`},
		{"numeric", `{"minimum":1,"maximum":3,"exclusiveMinimum":0,"exclusiveMaximum":4,"multipleOf":0.1}`, `2.0`, `3.1`},
		{"unicode", `{"minLength":2,"maxLength":3,"pattern":"^é"}`, `"é水"`, `"é"`},
		{"required", `{"required":["a"],"properties":{"a":{"type":"number"}},"additionalProperties":false}`, `{"a":1}`, `{"a":1,"b":2}`},
		{"patterns", `{"patternProperties":{"^a":{"type":"string"}},"additionalProperties":{"type":"number"}}`, `{"a":"x","b":1}`, `{"a":1}`},
		{"names", `{"propertyNames":{"pattern":"^[a-z]+$"},"minProperties":1,"maxProperties":2}`, `{"abc":1}`, `{"A":1}`},
		{"dependent", `{"dependentRequired":{"a":["b"]}}`, `{"a":1,"b":2}`, `{"a":1}`},
		{"tuple", `{"prefixItems":[{"type":"string"}],"items":{"type":"number"},"minItems":2,"maxItems":3}`, `["x",2]`, `["x","y"]`},
		{"unique", `{"uniqueItems":true}`, `[{"a":1},{"a":2}]`, `[{"a":1},{"a":1}]`},
		{"contains", `{"contains":{"type":"number"},"minContains":2,"maxContains":2}`, `[1,"x",2]`, `[1]`},
		{"allOf", `{"allOf":[{"minimum":1},{"maximum":3}]}`, `2`, `4`},
		{"anyOf", `{"anyOf":[{"type":"number"},{"type":"string"}]}`, `"x"`, `true`},
		{"oneOf", `{"oneOf":[{"type":"number"},{"minimum":0}]}`, `-1`, `2`},
		{"not", `{"not":{"const":1}}`, `2`, `1`},
		{"conditional", `{"if":{"type":"number"},"then":{"minimum":2},"else":{"type":"string"}}`, `"x"`, `1`},
		{"dependentSchema", `{"dependentSchemas":{"a":{"required":["b"]}}}`, `{"a":1,"b":1}`, `{"a":1}`},
		{"refSibling", `{"$defs":{"n":{"type":"number"}},"$ref":"#/$defs/n","minimum":2}`, `3`, `1`},
		{"unevaluatedObject", `{"anyOf":[{"properties":{"a":true}},{"properties":{"b":true}}],"unevaluatedProperties":false}`, `{"a":1,"b":2}`, `{"c":1}`},
		{"unevaluatedArray", `{"contains":{"type":"number"},"unevaluatedItems":false}`, `[1,2]`, `[1,"x"]`},
		{"failedAnnotations", `{"anyOf":[{"properties":{"a":true},"required":["z"]},{"properties":{"b":true}}],"unevaluatedProperties":false}`, `{"b":1}`, `{"a":1}`},
		{"conditionalAnnotations", `{"if":{"properties":{"a":true},"required":["z"]},"else":{"properties":{"b":true}},"unevaluatedProperties":false}`, `{"b":1}`, `{"a":1}`},
		{"recursive", `{"$defs":{"node":{"type":"object","properties":{"next":{"$ref":"#/$defs/node"}}}},"$ref":"#/$defs/node"}`, `{"next":{"next":{}}}`, `{"next":1}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, errs := compileSchema(engineJSON(t, tt.schema), true)
			if len(errs) > 0 {
				t.Fatalf("compile: %v", errs)
			}
			if errs := validateCompiledSchema(p, engineJSON(t, tt.good)); len(errs) > 0 {
				t.Fatalf("good: %v", errs)
			}
			if errs := validateCompiledSchema(p, engineJSON(t, tt.bad)); len(errs) == 0 {
				t.Fatal("invalid document accepted")
			}
		})
	}
}
func TestJSONSchemaCycleAndPointer(t *testing.T) {
	p, errs := compileSchema(engineJSON(t, `{"$ref":"#"}`), true)
	if len(errs) > 0 {
		t.Fatal(errs)
	}
	if len(validateCompiledSchema(p, nil)) == 0 {
		t.Fatal("cycle accepted")
	}
	p, errs = compileSchema(engineJSON(t, `{"properties":{"a/b~c":{"type":"number"}}}`), true)
	if len(errs) > 0 {
		t.Fatal(errs)
	}
	result := validateCompiledSchema(p, engineJSON(t, `{"a/b~c":"x"}`))
	if len(result) != 1 || result[0].Path != "/a~1b~0c" {
		t.Fatalf("pointer: %v", result)
	}
}

func TestJSONSchemaDynamicAndReferenceAnnotations(t *testing.T) {
	tests := []struct{ name, schema, good, bad string }{
		{"referenceAnnotations", `{"$defs":{"base":{"properties":{"a":true}}},"$ref":"#/$defs/base","unevaluatedProperties":false}`, `{"a":1}`, `{"b":1}`},
		{"allOfAnnotations", `{"allOf":[{"properties":{"a":true}},{"properties":{"b":true}}],"unevaluatedProperties":false}`, `{"a":1,"b":2}`, `{"c":3}`},
		{"oneOfAnnotations", `{"oneOf":[{"properties":{"a":true},"required":["a"]},{"properties":{"b":true},"required":["b"]}],"unevaluatedProperties":false}`, `{"a":1}`, `{"a":1,"c":2}`},
		{"dynamicOverride", `{"$dynamicAnchor":"node","$ref":"#/$defs/tree","required":["tag"],"$defs":{"tree":{"$id":"tree","$dynamicAnchor":"node","type":"object","properties":{"next":{"$dynamicRef":"#node"}}}}}`, `{"tag":true,"next":{"tag":true}}`, `{"tag":true,"next":{}}`},
		{"bothRefs", `{"$defs":{"number":{"type":"number"},"positive":{"minimum":1}},"$ref":"#/$defs/number","$dynamicRef":"#/$defs/positive"}`, `2`, `0`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, errs := compileSchema(engineJSON(t, tt.schema), true)
			if len(errs) > 0 {
				t.Fatalf("compile: %v", errs)
			}
			if errs := validateCompiledSchema(p, engineJSON(t, tt.good)); len(errs) > 0 {
				t.Fatalf("good: %v", errs)
			}
			if errs := validateCompiledSchema(p, engineJSON(t, tt.bad)); len(errs) == 0 {
				t.Fatal("invalid document accepted")
			}
		})
	}
}

func TestJSONSchemaHardCycleAndMultipleOf(t *testing.T) {
	for _, schema := range []string{`{"not":{"$ref":"#"}}`, `{"anyOf":[true,{"$ref":"#"}]}`, `{"if":{"$ref":"#"},"then":true,"else":true}`} {
		p, errs := compileSchema(engineJSON(t, schema), true)
		if len(errs) > 0 {
			t.Fatal(errs)
		}
		if len(validateCompiledSchema(p, nil)) == 0 {
			t.Fatalf("cycle masked by %s", schema)
		}
	}
	p, errs := compileSchema(engineJSON(t, `{"multipleOf":1}`), true)
	if len(errs) > 0 {
		t.Fatal(errs)
	}
	if len(validateCompiledSchema(p, float64(1000000000000.5))) == 0 {
		t.Fatal("large fractional value accepted as integral multiple")
	}
}

// Isolate each boundary: a neighboring assertion must not mask a missing check.
func TestJSONSchemaAssertionBoundaries(t *testing.T) {
	tests := []struct{ schema, good, bad string }{
		{`{"minimum":2}`, `2`, `1`}, {`{"maximum":2}`, `2`, `3`},
		{`{"exclusiveMinimum":2}`, `3`, `2`}, {`{"exclusiveMaximum":2}`, `1`, `2`},
		{`{"multipleOf":2}`, `4`, `3`},
		{`{"minLength":2}`, `"é水"`, `"é"`}, {`{"maxLength":2}`, `"é水"`, `"é水a"`},
		{`{"pattern":"^a+$"}`, `"aaa"`, `"b"`},
		{`{"minProperties":2}`, `{"a":1,"b":1}`, `{"a":1}`},
		{`{"maxProperties":1}`, `{"a":1}`, `{"a":1,"b":1}`},
		{`{"required":["a"]}`, `{"a":null}`, `{}`},
		{`{"properties":{"a":{"type":"number"}}}`, `{"a":1}`, `{"a":"x"}`},
		{`{"patternProperties":{"^a":{"type":"number"}}}`, `{"a":1}`, `{"a":"x"}`},
		{`{"additionalProperties":{"type":"number"}}`, `{"a":1}`, `{"a":"x"}`},
		{`{"propertyNames":{"pattern":"^a$"}}`, `{"a":1}`, `{"b":1}`},
		{`{"minItems":2}`, `[1,2]`, `[1]`}, {`{"maxItems":1}`, `[1]`, `[1,2]`},
		{`{"prefixItems":[{"type":"number"}]}`, `[1,"x"]`, `["x",1]`},
		{`{"items":{"type":"number"}}`, `[1,2]`, `[1,"x"]`},
		{`{"contains":{"type":"number"}}`, `["x",1]`, `["x"]`},
		{`{"contains":{"type":"number"},"minContains":2}`, `[1,2]`, `[1]`},
		{`{"contains":{"type":"number"},"maxContains":2}`, `[1,2]`, `[1,2,3]`},
		{`{"if":{"type":"number"},"then":{"minimum":2}}`, `2`, `1`},
		{`{"if":{"type":"number"},"else":{"type":"string"}}`, `"x"`, `true`},
		{`{"unevaluatedProperties":{"type":"number"}}`, `{"a":1}`, `{"a":"x"}`},
		{`{"unevaluatedItems":{"type":"number"}}`, `[1]`, `["x"]`},
	}
	for _, tt := range tests {
		t.Run(tt.schema, func(t *testing.T) {
			p, errs := compileSchema(engineJSON(t, tt.schema), true)
			if len(errs) > 0 {
				t.Fatal(errs)
			}
			if errs := validateCompiledSchema(p, engineJSON(t, tt.good)); len(errs) > 0 {
				t.Fatalf("good rejected: %v", errs)
			}
			if len(validateCompiledSchema(p, engineJSON(t, tt.bad))) == 0 {
				t.Fatal("bad accepted")
			}
		})
	}
}
