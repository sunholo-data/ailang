package builtins

import "testing"

func TestSchemaReferencesAndPatterns(t *testing.T) {
	for _, ref := range []string{"#", "#/$defs/a~1b~0c", "#named", "#dynamic"} {
		schema := map[string]any{"$ref": ref, "$defs": map[string]any{"a/b~c": map[string]any{"$anchor": "named", "$dynamicAnchor": "dynamic", "pattern": "^[a-z]+$"}}}
		p, errs := compileSchema(schema, true)
		if len(errs) > 0 || p.root.ref == nil {
			t.Fatalf("%s: %#v", ref, errs)
		}
	}
	for _, schema := range []map[string]any{
		{"$ref": "other.json"}, {"$ref": "#/missing"}, {"$ref": "#/$defs/a~2"},
		{"pattern": `(a)\1`}, {"pattern": `(?=a)`},
		{"patternProperties": map[string]any{"(?=a)": true}},
		{"$schema": "http://json-schema.org/draft-07/schema#"},
		{"$id": "https://example.com/schema"},
		{"$vocabulary": map[string]any{"https://example.com/unknown": true}},
	} {
		_, errs := compileSchema(schema, true)
		if len(errs) == 0 {
			t.Fatalf("accepted %#v", schema)
		}
	}
}

func TestSchemaDynamicReferenceTarget(t *testing.T) {
	p, errs := compileSchema(map[string]any{"$dynamicAnchor": "node", "$dynamicRef": "#node"}, true)
	if len(errs) > 0 || p.root.dynamicRef != "#node" || p.root.children["$dynamicRef"] != p.root {
		t.Fatalf("%#v %#v", p, errs)
	}
}
