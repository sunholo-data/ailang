package embed

import (
	"encoding/json"
	"testing"

	"github.com/sunholo/ailang/internal/eval"
)

func TestFromGo(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		wantType string
	}{
		{"nil", nil, "unit"},
		{"bool true", true, "bool"},
		{"bool false", false, "bool"},
		{"int", 42, "int"},
		{"int8", int8(8), "int"},
		{"int64", int64(64), "int"},
		{"uint", uint(100), "int"},
		{"float32", float32(3.14), "float"},
		{"float64", float64(2.71), "float"},
		{"string", "hello", "string"},
		{"[]byte", []byte{1, 2, 3}, "bytes"},
		{"[]int", []int{1, 2, 3}, "list"},
		{"[]string", []string{"a", "b"}, "list"},
		{"map[string]int", map[string]int{"a": 1}, "record"},
		{"struct", struct{ Name string }{"Alice"}, "record"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FromGo(tt.input)
			if err != nil {
				t.Fatalf("FromGo(%v) error: %v", tt.input, err)
			}
			if got.Type() != tt.wantType {
				t.Errorf("FromGo(%v).Type() = %q, want %q", tt.input, got.Type(), tt.wantType)
			}
		})
	}
}

func TestFromGoErrors(t *testing.T) {
	// Map with non-string keys should fail
	_, err := FromGo(map[int]string{1: "a"})
	if err == nil {
		t.Error("FromGo(map[int]string) should error")
	}
}

func TestFromGo_MapWithEvalValues(t *testing.T) {
	// Regression test: map[string]interface{} containing *eval.TaggedValue
	// must preserve the TaggedValue, not convert it to a plain RecordValue.
	// This is the exact pattern used by @raw route handlers.
	jobj := &eval.TaggedValue{
		ModulePath: "std/json", TypeName: "Json", CtorName: "JObject",
		Fields: []eval.Value{&eval.ListValue{Elements: []eval.Value{
			&eval.RecordValue{
				Fields: map[string]eval.Value{
					"key": &eval.StringValue{Value: "X-Test"},
					"value": &eval.TaggedValue{
						ModulePath: "std/json", TypeName: "Json", CtorName: "JString",
						Fields: []eval.Value{&eval.StringValue{Value: "hello"}},
					},
				},
			},
		}}},
	}

	m := map[string]interface{}{
		"body":    "test body",
		"headers": jobj,
		"method":  "POST",
	}

	result, err := FromGo(m)
	if err != nil {
		t.Fatalf("FromGo error: %v", err)
	}

	rec, ok := result.(*eval.RecordValue)
	if !ok {
		t.Fatalf("expected RecordValue, got %T", result)
	}

	// headers must be the original TaggedValue, not a converted struct
	headers, ok := rec.Fields["headers"].(*eval.TaggedValue)
	if !ok {
		t.Fatalf("headers should be *eval.TaggedValue, got %T", rec.Fields["headers"])
	}
	if headers.CtorName != "JObject" {
		t.Errorf("headers should be JObject, got %s", headers.CtorName)
	}
}

func TestToGo(t *testing.T) {
	tests := []struct {
		name  string
		input eval.Value
		want  interface{}
	}{
		{"unit", &eval.UnitValue{}, nil},
		{"bool", &eval.BoolValue{Value: true}, true},
		{"int", &eval.IntValue{Value: 42}, 42},
		{"float", &eval.FloatValue{Value: 3.14}, 3.14},
		{"string", &eval.StringValue{Value: "hello"}, "hello"},
		{"bytes", &eval.BytesValue{Value: []byte{1, 2}}, []byte{1, 2}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ToGo(tt.input)
			if err != nil {
				t.Fatalf("ToGo(%v) error: %v", tt.input, err)
			}
			// For bytes, compare length
			if b, ok := got.([]byte); ok {
				wantB := tt.want.([]byte)
				if len(b) != len(wantB) {
					t.Errorf("ToGo() bytes len = %d, want %d", len(b), len(wantB))
				}
				return
			}
			if got != tt.want {
				t.Errorf("ToGo(%v) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestToGoList(t *testing.T) {
	list := &eval.ListValue{
		Elements: []eval.Value{
			&eval.IntValue{Value: 1},
			&eval.IntValue{Value: 2},
			&eval.IntValue{Value: 3},
		},
	}
	got, err := ToGo(list)
	if err != nil {
		t.Fatalf("ToGo(list) error: %v", err)
	}
	slice, ok := got.([]interface{})
	if !ok {
		t.Fatalf("ToGo(list) type = %T, want []interface{}", got)
	}
	if len(slice) != 3 {
		t.Errorf("len(slice) = %d, want 3", len(slice))
	}
}

func TestToGoRecord(t *testing.T) {
	record := &eval.RecordValue{
		Fields: map[string]eval.Value{
			"name": &eval.StringValue{Value: "Alice"},
			"age":  &eval.IntValue{Value: 30},
		},
	}
	got, err := ToGo(record)
	if err != nil {
		t.Fatalf("ToGo(record) error: %v", err)
	}
	m, ok := got.(map[string]interface{})
	if !ok {
		t.Fatalf("ToGo(record) type = %T, want map[string]interface{}", got)
	}
	if m["name"] != "Alice" {
		t.Errorf("m[name] = %v, want Alice", m["name"])
	}
	if m["age"] != 30 {
		t.Errorf("m[age] = %v, want 30", m["age"])
	}
}

func TestTypeSafeExtractors(t *testing.T) {
	t.Run("ToInt", func(t *testing.T) {
		v, err := ToInt(&eval.IntValue{Value: 42})
		if err != nil || v != 42 {
			t.Errorf("ToInt() = %d, %v; want 42, nil", v, err)
		}
		_, err = ToInt(&eval.StringValue{Value: "nope"})
		if err == nil {
			t.Error("ToInt(string) should error")
		}
	})

	t.Run("ToFloat", func(t *testing.T) {
		v, err := ToFloat(&eval.FloatValue{Value: 3.14})
		if err != nil || v != 3.14 {
			t.Errorf("ToFloat() = %f, %v; want 3.14, nil", v, err)
		}
		// Int should coerce to float
		v, err = ToFloat(&eval.IntValue{Value: 5})
		if err != nil || v != 5.0 {
			t.Errorf("ToFloat(int) = %f, %v; want 5.0, nil", v, err)
		}
	})

	t.Run("ToString", func(t *testing.T) {
		v, err := ToString(&eval.StringValue{Value: "hello"})
		if err != nil || v != "hello" {
			t.Errorf("ToString() = %q, %v; want hello, nil", v, err)
		}
	})

	t.Run("ToBool", func(t *testing.T) {
		v, err := ToBool(&eval.BoolValue{Value: true})
		if err != nil || !v {
			t.Errorf("ToBool() = %v, %v; want true, nil", v, err)
		}
	})

	t.Run("ToBytes", func(t *testing.T) {
		v, err := ToBytes(&eval.BytesValue{Value: []byte{1, 2, 3}})
		if err != nil || len(v) != 3 {
			t.Errorf("ToBytes() len = %d, %v; want 3, nil", len(v), err)
		}
	})

	t.Run("ToList", func(t *testing.T) {
		v, err := ToList(&eval.ListValue{Elements: []eval.Value{&eval.IntValue{Value: 1}}})
		if err != nil || len(v) != 1 {
			t.Errorf("ToList() len = %d, %v; want 1, nil", len(v), err)
		}
	})

	t.Run("ToRecord", func(t *testing.T) {
		v, err := ToRecord(&eval.RecordValue{Fields: map[string]eval.Value{"a": &eval.IntValue{Value: 1}}})
		if err != nil || len(v) != 1 {
			t.Errorf("ToRecord() len = %d, %v; want 1, nil", len(v), err)
		}
	})
}

func TestIsUnit(t *testing.T) {
	if !IsUnit(&eval.UnitValue{}) {
		t.Error("IsUnit(UnitValue) = false, want true")
	}
	if IsUnit(&eval.IntValue{Value: 0}) {
		t.Error("IsUnit(IntValue) = true, want false")
	}
}

func TestRoundTrip(t *testing.T) {
	// Test that values survive Go → AILANG → Go conversion
	original := map[string]interface{}{
		"name":   "Alice",
		"age":    42,
		"active": true,
		"scores": []interface{}{90, 85, 92},
	}

	ailangVal, err := FromGo(original)
	if err != nil {
		t.Fatalf("FromGo() error: %v", err)
	}

	roundTrip, err := ToGo(ailangVal)
	if err != nil {
		t.Fatalf("ToGo() error: %v", err)
	}

	// Check the map came back correctly
	m, ok := roundTrip.(map[string]interface{})
	if !ok {
		t.Fatalf("roundTrip type = %T, want map", roundTrip)
	}
	if m["name"] != "Alice" {
		t.Errorf("name = %v, want Alice", m["name"])
	}
	if m["age"] != 42 {
		t.Errorf("age = %v, want 42", m["age"])
	}
	if m["active"] != true {
		t.Errorf("active = %v, want true", m["active"])
	}
}

func TestJSONConversion(t *testing.T) {
	// Test JSON round-trip
	inputJSON := []byte(`{"name": "Bob", "count": 5}`)
	ailangVal, err := FromJSON(inputJSON)
	if err != nil {
		t.Fatalf("FromJSON() error: %v", err)
	}

	outputJSON, err := ToJSON(ailangVal)
	if err != nil {
		t.Fatalf("ToJSON() error: %v", err)
	}

	// Parse back to verify
	var result map[string]interface{}
	if err := json.Unmarshal(outputJSON, &result); err != nil {
		t.Fatalf("json.Unmarshal() error: %v", err)
	}
	if result["name"] != "Bob" {
		t.Errorf("name = %v, want Bob", result["name"])
	}
}

func TestEngineEval(t *testing.T) {
	// GAP-5: Standalone expressions still don't work in Eval()
	// The pipeline returns "empty program" for expressions without module context
	// See: internal/pipeline/pipeline_single.go:297-300
	t.Skip("Eval() for standalone expressions requires pipeline fix (GAP-5)")

	engine := New(".")
	defer engine.Close()

	// Test simple evaluation
	result, err := engine.Eval("1 + 2")
	if err != nil {
		t.Fatalf("Eval() error: %v", err)
	}

	intVal, err := ToInt(result)
	if err != nil {
		t.Fatalf("ToInt() error: %v", err)
	}
	if intVal != 3 {
		t.Errorf("Eval(1 + 2) = %d, want 3", intVal)
	}
}

func TestEngineEvalString(t *testing.T) {
	t.Skip("Eval() for standalone expressions requires pipeline fix (GAP-5)")

	engine := New(".")
	defer engine.Close()

	result, err := engine.Eval(`"hello" ++ " world"`)
	if err != nil {
		t.Fatalf("Eval() error: %v", err)
	}

	strVal, err := ToString(result)
	if err != nil {
		t.Fatalf("ToString() error: %v", err)
	}
	if strVal != "hello world" {
		t.Errorf("Eval() = %q, want %q", strVal, "hello world")
	}
}

func TestEngineEvalList(t *testing.T) {
	t.Skip("Eval() for standalone expressions requires pipeline fix (GAP-5)")

	engine := New(".")
	defer engine.Close()

	result, err := engine.Eval("[1, 2, 3]")
	if err != nil {
		t.Fatalf("Eval() error: %v", err)
	}

	list, err := ToList(result)
	if err != nil {
		t.Fatalf("ToList() error: %v", err)
	}
	if len(list) != 3 {
		t.Errorf("len(list) = %d, want 3", len(list))
	}
}

func TestEngineClosedError(t *testing.T) {
	engine := New(".")
	engine.Close()

	_, err := engine.Eval("1 + 1")
	if err == nil {
		t.Error("Eval on closed engine should error")
	}
}

// TestFromGoWholeFloatBecomesInt verifies that FromGo converts whole-number
// float64 values (as produced by JSON decoding) to IntValue, not FloatValue.
// This is critical for serve-api route handlers where HTTP JSON args like
// {"count": 100} arrive as float64(100) from Go's json.Unmarshal.
// Regression: CallPreserveFloats kept them as FloatValue, breaking int-typed
// record fields in cross-package function calls.
func TestFromGoWholeFloatBecomesInt(t *testing.T) {
	tests := []struct {
		name     string
		input    float64
		wantType string
	}{
		{"whole number 100.0", 100.0, "int"},
		{"whole number 0.0", 0.0, "int"},
		{"whole number -42.0", -42.0, "int"},
		{"fractional 3.14", 3.14, "float"},
		{"fractional 0.5", 0.5, "float"},
		{"large whole 1e10", 1e10, "int"},
		// Beyond safe range, should stay float
		{"beyond safe 1e16", 1e16, "float"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FromGo(tt.input)
			if err != nil {
				t.Fatalf("FromGo(%v) error: %v", tt.input, err)
			}
			if got.Type() != tt.wantType {
				t.Errorf("FromGo(%v).Type() = %q, want %q (value: %s)",
					tt.input, got.Type(), tt.wantType, got.String())
			}
		})
	}
}

// TestFromGoPreserveFloatsKeepsFloat verifies that FromGoPreserveFloats
// retains FloatValue for all float64 inputs, even whole numbers.
func TestFromGoPreserveFloatsKeepsFloat(t *testing.T) {
	tests := []struct {
		name  string
		input float64
	}{
		{"whole number 100.0", 100.0},
		{"zero 0.0", 0.0},
		{"fractional 3.14", 3.14},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FromGoPreserveFloats(tt.input)
			if err != nil {
				t.Fatalf("FromGoPreserveFloats(%v) error: %v", tt.input, err)
			}
			if got.Type() != "float" {
				t.Errorf("FromGoPreserveFloats(%v).Type() = %q, want %q",
					tt.input, got.Type(), "float")
			}
		})
	}
}
