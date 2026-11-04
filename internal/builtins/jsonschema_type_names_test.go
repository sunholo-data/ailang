package builtins

import "testing"

func TestSchemaRejectsCompositeTypeNames(t *testing.T) {
	for _, strict := range []bool{true, false} {
		for _, name := range []string{"null|boolean", "object|array", "null|boolean|object|array|number|string|integer"} {
			for _, typ := range []any{name, []any{name}} {
				_, errs := compileSchema(map[string]any{"type": typ}, strict)
				if len(errs) == 0 {
					t.Fatalf("strict=%v accepted type %#v", strict, typ)
				}
			}
		}
	}
}
