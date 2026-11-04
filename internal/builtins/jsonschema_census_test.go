package builtins

import (
	"bytes"
	"encoding/json"
	"os"
	"reflect"
	"sort"
	"testing"
)

func loadSyntheticCensus(t testing.TB) any {
	t.Helper()
	b, err := os.ReadFile("testdata/jsonschema/synthetic-census.json")
	if err != nil {
		t.Fatal(err)
	}
	var schema any
	if err := json.Unmarshal(b, &schema); err != nil {
		t.Fatal(err)
	}
	return schema
}

func TestJSONSchemaSyntheticCensus(t *testing.T) {
	schema := loadSyntheticCensus(t)
	want := map[string]int{"type": 193, "required": 57, "properties": 48, "$ref": 40, "enum": 35, "items": 34, "additionalProperties": 33, "minLength": 19, "pattern": 17, "if": 12, "then": 12, "const": 11, "minItems": 8, "minimum": 8, "allOf": 6, "exclusiveMinimum": 3, "anyOf": 2, "not": 2, "maxLength": 2, "dependentRequired": 1, "oneOf": 1, "maximum": 1}
	got := map[string]int{}
	var walk func(any)
	walk = func(v any) {
		switch x := v.(type) {
		case map[string]any:
			for k, v := range x {
				if _, ok := want[k]; ok {
					got[k]++
				}
				walk(v)
			}
		case []any:
			for _, v := range x {
				walk(v)
			}
		}
	}
	walk(schema)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("synthetic census got %v; want %v", got, want)
	}
	program, errs := compileSchema(schema, true)
	if len(errs) > 0 {
		t.Fatal(errs)
	}
	doc := map[string]any{"name": "", "age": float64(-1)}
	baseline := validateCompiledSchema(program, doc)
	if len(baseline) != 2 {
		t.Fatalf("want two document errors: %v", baseline)
	}
	seen := map[string]string{}
	for _, e := range baseline {
		seen[e.Path] = e.Keyword
	}
	if !reflect.DeepEqual(seen, map[string]string{"/name": "minLength", "/age": "minimum"}) {
		t.Fatal(baseline)
	}
	for i := 0; i < 100; i++ {
		// Reverse serialized object key order, then recreate maps each run.
		b, marshalErr := reversedSchemaJSON(schema)
		if marshalErr != nil {
			t.Fatal(marshalErr)
		}
		var permuted any
		if err := json.Unmarshal(b, &permuted); err != nil {
			t.Fatal(err)
		}
		p, err := compileSchema(permuted, true)
		if len(err) > 0 {
			t.Fatal(err)
		}
		if got := validateCompiledSchema(p, doc); !reflect.DeepEqual(got, baseline) {
			t.Fatalf("run %d: %v != %v", i, got, baseline)
		}
	}
}

// reversedSchemaJSON deliberately reverses every object key; arrays keep order.
func reversedSchemaJSON(value any) ([]byte, error) {
	switch v := value.(type) {
	case map[string]any:
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sort.Sort(sort.Reverse(sort.StringSlice(keys)))
		parts := make([][]byte, 0, len(keys))
		for _, k := range keys {
			key, err := json.Marshal(k)
			if err != nil {
				return nil, err
			}
			child, err := reversedSchemaJSON(v[k])
			if err != nil {
				return nil, err
			}
			parts = append(parts, append(append(key, ':'), child...))
		}
		return append(append([]byte("{"), bytes.Join(parts, []byte(","))...), '}'), nil
	case []any:
		parts := make([][]byte, 0, len(v))
		for _, child := range v {
			b, err := reversedSchemaJSON(child)
			if err != nil {
				return nil, err
			}
			parts = append(parts, b)
		}
		return append(append([]byte("["), bytes.Join(parts, []byte(","))...), ']'), nil
	default:
		return json.Marshal(value)
	}
}
