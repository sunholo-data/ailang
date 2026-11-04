package builtins

import (
	"fmt"
	"testing"
)

// BenchmarkJSONSchemaCensusScale measures the public API's compile-plus-validate
// work: 460 schema nodes, 1000 document properties, and no cache across calls.
func BenchmarkJSONSchemaCensusScale(b *testing.B) {
	props := map[string]any{}
	doc := map[string]any{}
	for i := 0; i < 459; i++ {
		name := fmt.Sprintf("field%d", i)
		props[name] = map[string]any{"type": "string", "minLength": float64(1), "maxLength": float64(64), "pattern": "^[a-z]+$"}
	}
	for i := 0; i < 1000; i++ {
		doc[fmt.Sprintf("field%d", i)] = "value"
	}
	schema := map[string]any{"type": "object", "properties": props, "additionalProperties": true}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p, errs := compileSchema(schema, true)
		if len(errs) > 0 {
			b.Fatal(errs)
		}
		if errs := validateCompiledSchema(p, doc); len(errs) > 0 {
			b.Fatal(errs)
		}
	}
}
