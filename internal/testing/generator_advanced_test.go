package testing

import (
	"math/rand"
	"testing"

	"github.com/sunholo/ailang/internal/eval"
)

// TestMapGenerator tests value transformation.
func TestMapGenerator(t *testing.T) {
	// Generate ints, then double them
	intGen := NewIntGenerator(1, 10)
	doubleGen := NewMapGenerator(intGen, func(v eval.Value) eval.Value {
		intVal := v.(*eval.IntValue)
		return &eval.IntValue{Value: intVal.Value * 2}
	})

	rng := rand.New(rand.NewSource(42))

	for i := 0; i < 10; i++ {
		val := doubleGen.Generate(rng)
		intVal := val.(*eval.IntValue)

		// Should be even and in range [2, 20]
		if intVal.Value < 2 || intVal.Value > 20 {
			t.Errorf("value %d out of expected range [2, 20]", intVal.Value)
		}

		if intVal.Value%2 != 0 {
			t.Errorf("value %d should be even (doubled)", intVal.Value)
		}
	}
}

// TestFilterGenerator tests conditional generation.
func TestFilterGenerator(t *testing.T) {
	// Generate only even ints
	intGen := NewIntGenerator(1, 100)
	evenGen := NewFilterGenerator(intGen, func(v eval.Value) bool {
		intVal := v.(*eval.IntValue)
		return intVal.Value%2 == 0
	}, 100)

	rng := rand.New(rand.NewSource(42))

	for i := 0; i < 20; i++ {
		val := evenGen.Generate(rng)
		intVal := val.(*eval.IntValue)

		if intVal.Value%2 != 0 {
			t.Errorf("value %d should be even (filtered)", intVal.Value)
		}
	}
}

// TestFilterGenerator_MaxRetries tests retry limit.
func TestFilterGenerator_MaxRetries(t *testing.T) {
	// Impossible filter (always false)
	intGen := NewIntGenerator(1, 10)
	impossibleGen := NewFilterGenerator(intGen, func(v eval.Value) bool {
		return false // Never satisfied
	}, 5)

	rng := rand.New(rand.NewSource(42))

	// Should still return a value (fallback)
	val := impossibleGen.Generate(rng)
	if val == nil {
		t.Error("expected fallback value, got nil")
	}
}

// TestOneOfGenerator tests random choice.
func TestOneOfGenerator(t *testing.T) {
	// Choose between generating 1, 2, or 3
	gen := NewOneOfGenerator(
		NewConstantGenerator(&eval.IntValue{Value: 1}),
		NewConstantGenerator(&eval.IntValue{Value: 2}),
		NewConstantGenerator(&eval.IntValue{Value: 3}),
	)

	rng := rand.New(rand.NewSource(42))

	seen := make(map[int]bool)
	for i := 0; i < 100; i++ {
		val := gen.Generate(rng)
		intVal := val.(*eval.IntValue)
		seen[intVal.Value] = true
	}

	// Should see all three values
	if !seen[1] || !seen[2] || !seen[3] {
		t.Errorf("expected to see all three values, got %v", seen)
	}
}

// TestFrequencyGenerator tests weighted choice.
func TestFrequencyGenerator(t *testing.T) {
	// 90% weight for 1, 10% for 2
	gen := NewFrequencyGenerator(
		[]int{90, 10},
		[]Generator{
			NewConstantGenerator(&eval.IntValue{Value: 1}),
			NewConstantGenerator(&eval.IntValue{Value: 2}),
		},
	)

	rng := rand.New(rand.NewSource(42))

	count1 := 0
	count2 := 0
	total := 1000

	for i := 0; i < total; i++ {
		val := gen.Generate(rng)
		intVal := val.(*eval.IntValue)

		if intVal.Value == 1 {
			count1++
		} else if intVal.Value == 2 {
			count2++
		}
	}

	// Should be roughly 90/10 split (with tolerance)
	ratio1 := float64(count1) / float64(total)
	if ratio1 < 0.85 || ratio1 > 0.95 {
		t.Errorf("expected ~90%% for value 1, got %.2f%%", ratio1*100)
	}
}

// TestSizedGenerator tests size-aware generation.
func TestSizedGenerator(t *testing.T) {
	// Generate lists with size-dependent length
	gen := NewSizedGenerator(5, func(size int) Generator {
		return NewListGenerator(NewIntGenerator(0, 10), size, size)
	})

	rng := rand.New(rand.NewSource(42))

	val := gen.Generate(rng)
	listVal := val.(*eval.ListValue)

	if len(listVal.Elements) != 5 {
		t.Errorf("expected list of length 5, got %d", len(listVal.Elements))
	}

	// Test with different size
	gen2 := gen.WithSize(10)
	val2 := gen2.Generate(rng)
	listVal2 := val2.(*eval.ListValue)

	if len(listVal2.Elements) != 10 {
		t.Errorf("expected list of length 10, got %d", len(listVal2.Elements))
	}
}

// TestADTGenerator_Nullary tests ADT with no fields.
func TestADTGenerator_Nullary(t *testing.T) {
	gen := NewADTGenerator("None", []Generator{}, true)
	rng := rand.New(rand.NewSource(42))

	val := gen.Generate(rng)
	tagged, ok := val.(*eval.TaggedValue)
	if !ok {
		t.Fatalf("expected TaggedValue, got %T", val)
	}

	if tagged.CtorName != "None" {
		t.Errorf("expected constructor 'None', got %q", tagged.CtorName)
	}

	if len(tagged.Fields) != 0 {
		t.Errorf("expected 0 fields for None, got %d", len(tagged.Fields))
	}
}

// TestADTGenerator_Unary tests ADT with single field.
func TestADTGenerator_Unary(t *testing.T) {
	gen := NewADTGenerator("Some", []Generator{NewIntGenerator(1, 10)}, true)
	rng := rand.New(rand.NewSource(42))

	val := gen.Generate(rng)
	tagged, ok := val.(*eval.TaggedValue)
	if !ok {
		t.Fatalf("expected TaggedValue, got %T", val)
	}

	if tagged.CtorName != "Some" {
		t.Errorf("expected constructor 'Some', got %q", tagged.CtorName)
	}

	// Should have one field
	if len(tagged.Fields) != 1 {
		t.Fatalf("expected 1 field for Some, got %d", len(tagged.Fields))
	}

	intVal, ok := tagged.Fields[0].(*eval.IntValue)
	if !ok {
		t.Errorf("expected IntValue field, got %T", tagged.Fields[0])
	} else if intVal.Value < 1 || intVal.Value > 10 {
		t.Errorf("field %d out of range [1, 10]", intVal.Value)
	}
}

// TestADTGenerator_Nary tests ADT with multiple fields.
func TestADTGenerator_Nary(t *testing.T) {
	gen := NewADTGenerator("Pair", []Generator{
		NewIntGenerator(1, 5),
		NewStringGenerator(3, 3, "abc"),
	}, true)

	rng := rand.New(rand.NewSource(42))

	val := gen.Generate(rng)
	tagged, ok := val.(*eval.TaggedValue)
	if !ok {
		t.Fatalf("expected TaggedValue, got %T", val)
	}

	if tagged.CtorName != "Pair" {
		t.Errorf("expected constructor 'Pair', got %q", tagged.CtorName)
	}

	// Should have 2 fields
	if len(tagged.Fields) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(tagged.Fields))
	}

	if _, ok := tagged.Fields[0].(*eval.IntValue); !ok {
		t.Errorf("field 0: expected IntValue, got %T", tagged.Fields[0])
	}

	if _, ok := tagged.Fields[1].(*eval.StringValue); !ok {
		t.Errorf("field 1: expected StringValue, got %T", tagged.Fields[1])
	}
}

// TestRecordGenerator tests record generation.
func TestRecordGenerator(t *testing.T) {
	gen := NewRecordGenerator(map[string]Generator{
		"age":  NewIntGenerator(1, 100),
		"name": NewStringGenerator(5, 10, ""),
	})

	rng := rand.New(rand.NewSource(42))

	val := gen.Generate(rng)
	record, ok := val.(*eval.RecordValue)
	if !ok {
		t.Fatalf("expected RecordValue, got %T", val)
	}

	if len(record.Fields) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(record.Fields))
	}

	age, ok := record.Fields["age"]
	if !ok {
		t.Error("missing 'age' field")
	} else if _, ok := age.(*eval.IntValue); !ok {
		t.Errorf("age: expected IntValue, got %T", age)
	}

	name, ok := record.Fields["name"]
	if !ok {
		t.Error("missing 'name' field")
	} else if _, ok := name.(*eval.StringValue); !ok {
		t.Errorf("name: expected StringValue, got %T", name)
	}
}

// TestTupleGenerator tests tuple generation.
func TestTupleGenerator(t *testing.T) {
	gen := NewTupleGenerator([]Generator{
		NewIntGenerator(1, 10),
		NewBoolGenerator(),
		NewStringGenerator(0, 5, ""),
	})

	rng := rand.New(rand.NewSource(42))

	val := gen.Generate(rng)
	tuple, ok := val.(*eval.TupleValue)
	if !ok {
		t.Fatalf("expected TupleValue, got %T", val)
	}

	if len(tuple.Elements) != 3 {
		t.Fatalf("expected 3 elements, got %d", len(tuple.Elements))
	}

	if _, ok := tuple.Elements[0].(*eval.IntValue); !ok {
		t.Errorf("element 0: expected IntValue, got %T", tuple.Elements[0])
	}

	if _, ok := tuple.Elements[1].(*eval.BoolValue); !ok {
		t.Errorf("element 1: expected BoolValue, got %T", tuple.Elements[1])
	}

	if _, ok := tuple.Elements[2].(*eval.StringValue); !ok {
		t.Errorf("element 2: expected StringValue, got %T", tuple.Elements[2])
	}
}

// TestOptionGenerator tests Option ADT generation.
func TestOptionGenerator(t *testing.T) {
	gen := OptionGenerator(NewIntGenerator(1, 100))
	rng := rand.New(rand.NewSource(42))

	seenNone := false
	seenSome := false

	for i := 0; i < 100; i++ {
		val := gen.Generate(rng)
		tagged := val.(*eval.TaggedValue)

		if tagged.CtorName == "None" {
			seenNone = true
		} else if tagged.CtorName == "Some" {
			seenSome = true
		}
	}

	if !seenNone {
		t.Error("expected to see None variant")
	}

	if !seenSome {
		t.Error("expected to see Some variant")
	}
}

// TestResultGenerator tests Result ADT generation.
func TestResultGenerator(t *testing.T) {
	gen := ResultGenerator(
		NewIntGenerator(1, 100),
		NewStringGenerator(5, 10, ""),
	)
	rng := rand.New(rand.NewSource(42))

	seenOk := false
	seenErr := false

	for i := 0; i < 100; i++ {
		val := gen.Generate(rng)
		tagged := val.(*eval.TaggedValue)

		if tagged.CtorName == "Ok" {
			seenOk = true
			if len(tagged.Fields) != 1 {
				t.Errorf("Ok: expected 1 field, got %d", len(tagged.Fields))
			} else if _, ok := tagged.Fields[0].(*eval.IntValue); !ok {
				t.Errorf("Ok field: expected IntValue, got %T", tagged.Fields[0])
			}
		} else if tagged.CtorName == "Err" {
			seenErr = true
			if len(tagged.Fields) != 1 {
				t.Errorf("Err: expected 1 field, got %d", len(tagged.Fields))
			} else if _, ok := tagged.Fields[0].(*eval.StringValue); !ok {
				t.Errorf("Err field: expected StringValue, got %T", tagged.Fields[0])
			}
		}
	}

	if !seenOk {
		t.Error("expected to see Ok variant")
	}

	if !seenErr {
		t.Error("expected to see Err variant")
	}
}

// TestConstantGenerator tests constant value generation.
func TestConstantGenerator(t *testing.T) {
	constant := &eval.IntValue{Value: 42}
	gen := NewConstantGenerator(constant)
	rng := rand.New(rand.NewSource(42))

	for i := 0; i < 10; i++ {
		val := gen.Generate(rng)
		intVal := val.(*eval.IntValue)

		if intVal.Value != 42 {
			t.Errorf("expected constant 42, got %d", intVal.Value)
		}
	}
}

// TestMapGenerator_Composition tests chaining map generators.
func TestMapGenerator_Composition(t *testing.T) {
	// Start with int, double it, then convert to string length
	intGen := NewIntGenerator(1, 5)
	doubleGen := NewMapGenerator(intGen, func(v eval.Value) eval.Value {
		intVal := v.(*eval.IntValue)
		return &eval.IntValue{Value: intVal.Value * 2}
	})
	strGen := NewMapGenerator(doubleGen, func(v eval.Value) eval.Value {
		intVal := v.(*eval.IntValue)
		str := ""
		for i := 0; i < intVal.Value; i++ {
			str += "x"
		}
		return &eval.StringValue{Value: str}
	})

	rng := rand.New(rand.NewSource(42))

	val := strGen.Generate(rng)
	strVal := val.(*eval.StringValue)

	// Length should be even and in range [2, 10]
	length := len(strVal.Value)
	if length < 2 || length > 10 {
		t.Errorf("string length %d out of expected range [2, 10]", length)
	}

	if length%2 != 0 {
		t.Errorf("string length %d should be even", length)
	}
}

// TestNestedADT tests generating nested ADT structures.
func TestNestedADT(t *testing.T) {
	// Result<Option<int>, string>
	innerGen := OptionGenerator(NewIntGenerator(1, 10))
	gen := ResultGenerator(innerGen, NewStringGenerator(3, 5, ""))

	rng := rand.New(rand.NewSource(42))

	for i := 0; i < 10; i++ {
		val := gen.Generate(rng)
		result := val.(*eval.TaggedValue)

		if result.CtorName == "Ok" {
			// Field should be Option (TaggedValue)
			if len(result.Fields) != 1 {
				t.Errorf("Ok: expected 1 field, got %d", len(result.Fields))
				continue
			}

			option, ok := result.Fields[0].(*eval.TaggedValue)
			if !ok {
				t.Errorf("Ok field: expected TaggedValue (Option), got %T", result.Fields[0])
				continue
			}

			if option.CtorName != "None" && option.CtorName != "Some" {
				t.Errorf("expected Option constructor (None/Some), got %q", option.CtorName)
			}
		}
	}
}

// TestRecordGenerator_EmptyRecord tests empty record generation.
func TestRecordGenerator_EmptyRecord(t *testing.T) {
	gen := NewRecordGenerator(map[string]Generator{})
	rng := rand.New(rand.NewSource(42))

	val := gen.Generate(rng)
	record := val.(*eval.RecordValue)

	if len(record.Fields) != 0 {
		t.Errorf("expected empty record, got %d fields", len(record.Fields))
	}
}

// TestGeneratorInterface_Advanced tests that advanced generators implement Generator.
func TestGeneratorInterface_Advanced(t *testing.T) {
	rng := rand.New(rand.NewSource(42))

	tests := []struct {
		name string
		gen  Generator
	}{
		{"MapGenerator", NewMapGenerator(NewIntGenerator(1, 10), func(v eval.Value) eval.Value { return v })},
		{"FilterGenerator", NewFilterGenerator(NewIntGenerator(1, 10), func(v eval.Value) bool { return true }, 10)},
		{"OneOfGenerator", NewOneOfGenerator(NewIntGenerator(1, 10), NewBoolGenerator())},
		{"FrequencyGenerator", NewFrequencyGenerator([]int{1, 1}, []Generator{NewIntGenerator(1, 10), NewBoolGenerator()})},
		{"SizedGenerator", NewSizedGenerator(5, func(size int) Generator { return NewIntGenerator(1, size) })},
		{"ADTGenerator", NewADTGenerator("Tag", []Generator{}, true)},
		{"RecordGenerator", NewRecordGenerator(map[string]Generator{"x": NewIntGenerator(1, 10)})},
		{"TupleGenerator", NewTupleGenerator([]Generator{NewIntGenerator(1, 10)})},
		{"ConstantGenerator", NewConstantGenerator(&eval.IntValue{Value: 42})},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gen Generator = tt.gen
			val := gen.Generate(rng)
			if val == nil {
				t.Error("expected non-nil value")
			}
		})
	}
}

// TestFrequencyGenerator_SingleWeight tests frequency with only one option.
func TestFrequencyGenerator_SingleWeight(t *testing.T) {
	gen := NewFrequencyGenerator(
		[]int{100},
		[]Generator{NewConstantGenerator(&eval.IntValue{Value: 42})},
	)

	rng := rand.New(rand.NewSource(42))

	for i := 0; i < 10; i++ {
		val := gen.Generate(rng)
		intVal := val.(*eval.IntValue)

		if intVal.Value != 42 {
			t.Errorf("expected 42, got %d", intVal.Value)
		}
	}
}

// TestADTGenerator_WithoutTag tests generating payload without wrapping in TaggedValue.
func TestADTGenerator_WithoutTag(t *testing.T) {
	gen := NewADTGenerator("Ignored", []Generator{NewIntGenerator(1, 10)}, false)
	rng := rand.New(rand.NewSource(42))

	val := gen.Generate(rng)

	// Should be IntValue directly, not TaggedValue
	intVal, ok := val.(*eval.IntValue)
	if !ok {
		t.Fatalf("expected IntValue (no tag), got %T", val)
	}

	if intVal.Value < 1 || intVal.Value > 10 {
		t.Errorf("value %d out of range [1, 10]", intVal.Value)
	}
}
