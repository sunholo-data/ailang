package builtins

import (
	"reflect"
	"strings"
	"testing"
)

func TestSchemaCompileAudit(t *testing.T) {
	cases := []struct {
		name    string
		schema  any
		strict  bool
		keyword string
	}{
		{"boolean", true, true, ""},
		{"unknown", map[string]any{"vendor": true}, true, "vendor"},
		{"lenient", map[string]any{"vendor": true}, false, ""},
		{"lenient malformed", map[string]any{"type": 42.0}, false, "type"},
		{"unused defs", map[string]any{"$defs": map[string]any{"unused": map[string]any{"oops": true}}}, true, "oops"},
		{"annotation data", map[string]any{"default": map[string]any{"oops": true}}, true, ""},
		{"items migration", map[string]any{"items": []any{true}}, true, "items"},
		{"non schema", nil, true, "schema"},
		{"duplicate required", map[string]any{"required": []any{"x", "x"}}, true, "required"},
		{"bad length", map[string]any{"minLength": -1.0}, true, "minLength"},
		{"bad multiple", map[string]any{"multipleOf": 0.0}, true, "multipleOf"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, errs := compileSchema(tc.schema, tc.strict)
			if tc.keyword == "" {
				if len(errs) > 0 {
					t.Fatal(errs)
				}
				return
			}
			if len(errs) == 0 || errs[0].Keyword != tc.keyword || !strings.Contains(errs[0].Message, "schema") {
				t.Fatalf("got %#v", errs)
			}
		})
	}
}

func TestSchemaCompileDeterministic(t *testing.T) {
	schema := map[string]any{"z": true, "a": true, "properties": map[string]any{"z": map[string]any{"bad": true}, "a": map[string]any{"bad": true}}}
	_, want := compileSchema(schema, true)
	for i := 0; i < 100; i++ {
		_, got := compileSchema(schema, true)
		if !reflect.DeepEqual(got, want) {
			t.Fatal(got, want)
		}
	}
}
