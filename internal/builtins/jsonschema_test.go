package builtins

import (
	"fmt"
	"github.com/sunholo-data/ailang/internal/eval"
	"testing"
)

func TestJSONSchemaPublicAPI(t *testing.T) {
	schema, err := newJSONBuilder(`{"type":"string","minLength":2}`).build()
	if err != nil {
		t.Fatal(err)
	}
	result, err := jsonschemaImpl(true, false)(nil, []eval.Value{schema, makeJString("x")})
	if err != nil {
		t.Fatal(err)
	}
	errors := result.(*eval.ListValue).Elements
	if len(errors) != 1 {
		t.Fatalf("expected one error, got %v", result)
	}
	record := errors[0].(*eval.RecordValue)
	if record.Fields["keyword"].(*eval.StringValue).Value != "minLength" {
		t.Fatal(record)
	}
	if record.Fields["path"].(*eval.StringValue).Value != "" {
		t.Fatal(record)
	}
}

func TestJSONSchemaCompileBeforeValidate(t *testing.T) {
	schema, _ := newJSONBuilder(`{"unknown":true,"type":"integer"}`).build()
	for _, check := range []bool{false, true} {
		result, err := jsonschemaImpl(true, check)(nil, []eval.Value{schema, makeJString("bad")})
		if err != nil {
			t.Fatal(err)
		}
		errors := result.(*eval.ListValue).Elements
		if len(errors) != 1 || errors[0].(*eval.RecordValue).Fields["keyword"].(*eval.StringValue).Value != "unknown" {
			t.Fatal(result)
		}
	}
	result, err := jsonschemaImpl(false, false)(nil, []eval.Value{schema, makeJString("bad")})
	if err != nil {
		t.Fatal(err)
	}
	if result.(*eval.ListValue).Elements[0].(*eval.RecordValue).Fields["keyword"].(*eval.StringValue).Value != "type" {
		t.Fatal(result)
	}
}

// Includes the Json ADT conversion and record-list construction of validate.
func BenchmarkJSONSchemaPublicAPI(b *testing.B) {
	props := map[string]eval.Value{}
	doc := map[string]eval.Value{}
	for i := 0; i < 459; i++ {
		props[fmt.Sprintf("field%d", i)] = &eval.RecordValue{Fields: map[string]eval.Value{
			"type": &eval.StringValue{Value: "string"}, "minLength": &eval.FloatValue{Value: 1},
			"maxLength": &eval.FloatValue{Value: 64}, "pattern": &eval.StringValue{Value: "^[a-z]+$"},
		}}
	}
	for i := 0; i < 1000; i++ {
		doc[fmt.Sprintf("field%d", i)] = &eval.StringValue{Value: "value"}
	}
	schema := valueToJSON(&eval.RecordValue{Fields: map[string]eval.Value{
		"type": &eval.StringValue{Value: "object"}, "properties": &eval.RecordValue{Fields: props}, "additionalProperties": &eval.BoolValue{Value: true},
	}})
	args := []eval.Value{schema, valueToJSON(&eval.RecordValue{Fields: doc})}
	impl := jsonschemaImpl(true, false)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := impl(nil, args)
		if err != nil {
			b.Fatal(err)
		}
		if len(result.(*eval.ListValue).Elements) != 0 {
			b.Fatal(result)
		}
	}
}
