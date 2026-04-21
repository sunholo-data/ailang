package testing

import (
	"math/rand"
	"testing"

	"github.com/sunholo-data/ailang/internal/eval"
)

// TestIntGenerator_Range tests that generated ints are within range.
func TestIntGenerator_Range(t *testing.T) {
	gen := NewIntGenerator(-10, 10)
	rng := rand.New(rand.NewSource(42))

	for i := 0; i < 100; i++ {
		val := gen.Generate(rng)
		intVal, ok := val.(*eval.IntValue)
		if !ok {
			t.Fatalf("expected IntValue, got %T", val)
		}

		if intVal.Value < -10 || intVal.Value > 10 {
			t.Errorf("value %d out of range [-10, 10]", intVal.Value)
		}
	}
}

// TestIntGenerator_SingleValue tests generator with min==max.
func TestIntGenerator_SingleValue(t *testing.T) {
	gen := NewIntGenerator(5, 5)
	rng := rand.New(rand.NewSource(42))

	for i := 0; i < 10; i++ {
		val := gen.Generate(rng)
		intVal := val.(*eval.IntValue)

		if intVal.Value != 5 {
			t.Errorf("expected 5, got %d", intVal.Value)
		}
	}
}

// TestIntGenerator_LargeRange tests generator with large range.
func TestIntGenerator_LargeRange(t *testing.T) {
	gen := NewIntGenerator(-1000, 1000)
	rng := rand.New(rand.NewSource(42))

	seen := make(map[int]bool)
	for i := 0; i < 100; i++ {
		val := gen.Generate(rng)
		intVal := val.(*eval.IntValue)
		seen[intVal.Value] = true
	}

	// Should see variety of values
	if len(seen) < 50 {
		t.Errorf("expected diverse values, got only %d unique values", len(seen))
	}
}

// TestFloatGenerator_Range tests that generated floats are within range.
func TestFloatGenerator_Range(t *testing.T) {
	gen := NewFloatGenerator(-1.0, 1.0)
	rng := rand.New(rand.NewSource(42))

	for i := 0; i < 100; i++ {
		val := gen.Generate(rng)
		floatVal, ok := val.(*eval.FloatValue)
		if !ok {
			t.Fatalf("expected FloatValue, got %T", val)
		}

		if floatVal.Value < -1.0 || floatVal.Value > 1.0 {
			t.Errorf("value %f out of range [-1.0, 1.0]", floatVal.Value)
		}
	}
}

// TestFloatGenerator_Distribution tests that floats are reasonably distributed.
func TestFloatGenerator_Distribution(t *testing.T) {
	gen := NewFloatGenerator(0.0, 10.0)
	rng := rand.New(rand.NewSource(42))

	var sum float64
	count := 1000

	for i := 0; i < count; i++ {
		val := gen.Generate(rng)
		floatVal := val.(*eval.FloatValue)
		sum += floatVal.Value
	}

	mean := sum / float64(count)
	expectedMean := 5.0 // Middle of [0, 10]

	// Mean should be roughly 5.0 (within 1.0 tolerance)
	if mean < expectedMean-1.0 || mean > expectedMean+1.0 {
		t.Errorf("mean %f not close to expected %f", mean, expectedMean)
	}
}

// TestBoolGenerator_Distribution tests that bools are roughly 50/50.
func TestBoolGenerator_Distribution(t *testing.T) {
	gen := NewBoolGenerator()
	rng := rand.New(rand.NewSource(42))

	trueCount := 0
	totalCount := 1000

	for i := 0; i < totalCount; i++ {
		val := gen.Generate(rng)
		boolVal, ok := val.(*eval.BoolValue)
		if !ok {
			t.Fatalf("expected BoolValue, got %T", val)
		}

		if boolVal.Value {
			trueCount++
		}
	}

	// Should be roughly 50% true (within 10% tolerance)
	ratio := float64(trueCount) / float64(totalCount)
	if ratio < 0.4 || ratio > 0.6 {
		t.Errorf("true ratio %f not close to 0.5", ratio)
	}
}

// TestStringGenerator_Length tests that generated strings respect length bounds.
func TestStringGenerator_Length(t *testing.T) {
	gen := NewStringGenerator(5, 10, "")
	rng := rand.New(rand.NewSource(42))

	for i := 0; i < 100; i++ {
		val := gen.Generate(rng)
		strVal, ok := val.(*eval.StringValue)
		if !ok {
			t.Fatalf("expected StringValue, got %T", val)
		}

		length := len(strVal.Value)
		if length < 5 || length > 10 {
			t.Errorf("string length %d out of range [5, 10]", length)
		}
	}
}

// TestStringGenerator_EmptyString tests generating empty strings.
func TestStringGenerator_EmptyString(t *testing.T) {
	gen := NewStringGenerator(0, 0, "")
	rng := rand.New(rand.NewSource(42))

	val := gen.Generate(rng)
	strVal := val.(*eval.StringValue)

	if strVal.Value != "" {
		t.Errorf("expected empty string, got %q", strVal.Value)
	}
}

// TestStringGenerator_CustomCharset tests custom character sets.
func TestStringGenerator_CustomCharset(t *testing.T) {
	gen := NewStringGenerator(10, 10, "abc")
	rng := rand.New(rand.NewSource(42))

	for i := 0; i < 10; i++ {
		val := gen.Generate(rng)
		strVal := val.(*eval.StringValue)

		// All characters should be a, b, or c
		for _, ch := range strVal.Value {
			if ch != 'a' && ch != 'b' && ch != 'c' {
				t.Errorf("unexpected character %c in string %q", ch, strVal.Value)
			}
		}
	}
}

// TestListGenerator_Length tests that generated lists respect length bounds.
func TestListGenerator_Length(t *testing.T) {
	elemGen := NewIntGenerator(0, 100)
	listGen := NewListGenerator(elemGen, 3, 7)
	rng := rand.New(rand.NewSource(42))

	for i := 0; i < 100; i++ {
		val := listGen.Generate(rng)
		listVal, ok := val.(*eval.ListValue)
		if !ok {
			t.Fatalf("expected ListValue, got %T", val)
		}

		length := len(listVal.Elements)
		if length < 3 || length > 7 {
			t.Errorf("list length %d out of range [3, 7]", length)
		}
	}
}

// TestListGenerator_EmptyList tests generating empty lists.
func TestListGenerator_EmptyList(t *testing.T) {
	elemGen := NewIntGenerator(0, 100)
	listGen := NewListGenerator(elemGen, 0, 0)
	rng := rand.New(rand.NewSource(42))

	val := listGen.Generate(rng)
	listVal := val.(*eval.ListValue)

	if len(listVal.Elements) != 0 {
		t.Errorf("expected empty list, got %d elements", len(listVal.Elements))
	}
}

// TestListGenerator_ElementTypes tests that list elements have correct type.
func TestListGenerator_ElementTypes(t *testing.T) {
	elemGen := NewBoolGenerator()
	listGen := NewListGenerator(elemGen, 5, 10)
	rng := rand.New(rand.NewSource(42))

	val := listGen.Generate(rng)
	listVal := val.(*eval.ListValue)

	for i, elem := range listVal.Elements {
		if _, ok := elem.(*eval.BoolValue); !ok {
			t.Errorf("element %d: expected BoolValue, got %T", i, elem)
		}
	}
}

// TestListGenerator_NestedLists tests generating lists of lists.
func TestListGenerator_NestedLists(t *testing.T) {
	innerGen := NewListGenerator(NewIntGenerator(0, 10), 1, 3)
	outerGen := NewListGenerator(innerGen, 2, 4)
	rng := rand.New(rand.NewSource(42))

	val := outerGen.Generate(rng)
	listVal := val.(*eval.ListValue)

	// Verify outer list has correct length
	if len(listVal.Elements) < 2 || len(listVal.Elements) > 4 {
		t.Fatalf("outer list length %d out of range [2, 4]", len(listVal.Elements))
	}

	// Verify each element is itself a list
	for i, elem := range listVal.Elements {
		innerList, ok := elem.(*eval.ListValue)
		if !ok {
			t.Errorf("element %d: expected ListValue, got %T", i, elem)
			continue
		}

		// Verify inner list has correct length
		if len(innerList.Elements) < 1 || len(innerList.Elements) > 3 {
			t.Errorf("inner list %d length %d out of range [1, 3]", i, len(innerList.Elements))
		}
	}
}

// TestPropertyRunner_DeterministicSeed tests that same seed produces same values.
func TestPropertyRunner_DeterministicSeed(t *testing.T) {
	config1 := DefaultConfig()
	config1.Seed = 12345

	config2 := DefaultConfig()
	config2.Seed = 12345

	runner1 := NewPropertyRunner(config1, 100)
	runner2 := NewPropertyRunner(config2, 100)

	// Generate 10 ints from each runner
	for i := 0; i < 10; i++ {
		val1 := runner1.GenerateIntInRange()
		val2 := runner2.GenerateIntInRange()

		if val1 != val2 {
			t.Errorf("iteration %d: values differ (seed not deterministic): %d vs %d", i, val1, val2)
		}
	}
}

// TestPropertyRunner_DifferentSeeds tests that different seeds produce different values.
func TestPropertyRunner_DifferentSeeds(t *testing.T) {
	config1 := DefaultConfig()
	config1.Seed = 11111

	config2 := DefaultConfig()
	config2.Seed = 22222

	runner1 := NewPropertyRunner(config1, 100)
	runner2 := NewPropertyRunner(config2, 100)

	// Generate values and check they differ
	same := 0
	total := 100

	for i := 0; i < total; i++ {
		val1 := runner1.GenerateIntInRange()
		val2 := runner2.GenerateIntInRange()

		if val1 == val2 {
			same++
		}
	}

	// Most values should differ (allow <20% same by chance)
	if same > total/5 {
		t.Errorf("too many same values (%d/%d) for different seeds", same, total)
	}
}

// TestPropertyRunner_GenerateBool tests bool generation via runner.
func TestPropertyRunner_GenerateBool(t *testing.T) {
	config := DefaultConfig()
	config.Seed = 42
	runner := NewPropertyRunner(config, 100)

	trueCount := 0
	for i := 0; i < 1000; i++ {
		if runner.GenerateBool() {
			trueCount++
		}
	}

	// Should be roughly 50%
	ratio := float64(trueCount) / 1000.0
	if ratio < 0.4 || ratio > 0.6 {
		t.Errorf("true ratio %f not close to 0.5", ratio)
	}
}

// TestPropertyRunner_GenerateString tests string generation via runner.
func TestPropertyRunner_GenerateString(t *testing.T) {
	config := DefaultConfig()
	config.Seed = 42
	config.MaxSize = 20
	runner := NewPropertyRunner(config, 100)

	for i := 0; i < 10; i++ {
		str := runner.GenerateString()

		if len(str) > 20 {
			t.Errorf("string length %d exceeds MaxSize %d", len(str), config.MaxSize)
		}
	}
}

// TestPropertyRunner_GenerateList tests list generation via runner.
func TestPropertyRunner_GenerateList(t *testing.T) {
	config := DefaultConfig()
	config.Seed = 42
	runner := NewPropertyRunner(config, 100)

	elemGen := NewIntGenerator(0, 100)
	list := runner.GenerateList(elemGen, 10)

	if len(list) > 10 {
		t.Errorf("list length %d exceeds max %d", len(list), 10)
	}

	// Verify all elements are ints
	for i, elem := range list {
		if _, ok := elem.(*eval.IntValue); !ok {
			t.Errorf("element %d: expected IntValue, got %T", i, elem)
		}
	}
}

// TestPropertyRunner_GetConfig tests config retrieval.
func TestPropertyRunner_GetConfig(t *testing.T) {
	config := DefaultConfig()
	config.Seed = 999
	config.MaxSize = 50

	runner := NewPropertyRunner(config, 100)
	retrieved := runner.GetConfig()

	if retrieved.Seed != 999 {
		t.Errorf("expected seed 999, got %d", retrieved.Seed)
	}

	if retrieved.MaxSize != 50 {
		t.Errorf("expected MaxSize 50, got %d", retrieved.MaxSize)
	}
}

// TestDefaultConfig tests default configuration values.
func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.MaxSize != 100 {
		t.Errorf("expected MaxSize 100, got %d", config.MaxSize)
	}

	if config.MinInt != -1000 {
		t.Errorf("expected MinInt -1000, got %d", config.MinInt)
	}

	if config.MaxInt != 1000 {
		t.Errorf("expected MaxInt 1000, got %d", config.MaxInt)
	}
}

// TestPropertyRunner_FloatGeneration tests float generation.
func TestPropertyRunner_FloatGeneration(t *testing.T) {
	config := DefaultConfig()
	config.Seed = 42
	config.MinFloat = -10.0
	config.MaxFloat = 10.0

	runner := NewPropertyRunner(config, 100)

	for i := 0; i < 100; i++ {
		val := runner.GenerateFloatInRange()

		if val < -10.0 || val > 10.0 {
			t.Errorf("float %f out of range [-10.0, 10.0]", val)
		}
	}
}

// TestGeneratorInterface_IntGenerator tests IntGenerator implements Generator.
func TestGeneratorInterface_IntGenerator(t *testing.T) {
	var gen Generator = NewIntGenerator(0, 10)
	rng := rand.New(rand.NewSource(42))

	val := gen.Generate(rng)
	if _, ok := val.(*eval.IntValue); !ok {
		t.Errorf("expected IntValue, got %T", val)
	}
}

// TestGeneratorInterface_FloatGenerator tests FloatGenerator implements Generator.
func TestGeneratorInterface_FloatGenerator(t *testing.T) {
	var gen Generator = NewFloatGenerator(0.0, 1.0)
	rng := rand.New(rand.NewSource(42))

	val := gen.Generate(rng)
	if _, ok := val.(*eval.FloatValue); !ok {
		t.Errorf("expected FloatValue, got %T", val)
	}
}

// TestGeneratorInterface_BoolGenerator tests BoolGenerator implements Generator.
func TestGeneratorInterface_BoolGenerator(t *testing.T) {
	var gen Generator = NewBoolGenerator()
	rng := rand.New(rand.NewSource(42))

	val := gen.Generate(rng)
	if _, ok := val.(*eval.BoolValue); !ok {
		t.Errorf("expected BoolValue, got %T", val)
	}
}

// TestGeneratorInterface_StringGenerator tests StringGenerator implements Generator.
func TestGeneratorInterface_StringGenerator(t *testing.T) {
	var gen Generator = NewStringGenerator(0, 10, "")
	rng := rand.New(rand.NewSource(42))

	val := gen.Generate(rng)
	if _, ok := val.(*eval.StringValue); !ok {
		t.Errorf("expected StringValue, got %T", val)
	}
}

// TestGeneratorInterface_ListGenerator tests ListGenerator implements Generator.
func TestGeneratorInterface_ListGenerator(t *testing.T) {
	elemGen := NewIntGenerator(0, 10)
	var gen Generator = NewListGenerator(elemGen, 0, 5)
	rng := rand.New(rand.NewSource(42))

	val := gen.Generate(rng)
	if _, ok := val.(*eval.ListValue); !ok {
		t.Errorf("expected ListValue, got %T", val)
	}
}

// TestPropertyRunner_ShrinkInt tests shrinking integers to minimal counterexample.
func TestPropertyRunner_ShrinkInt(t *testing.T) {
	config := DefaultConfig()
	runner := NewPropertyRunner(config, 100)
	shrinker := NewIntShrinker()

	// Property: all integers must be < 50
	// This will fail for values >= 50
	original := &eval.IntValue{Value: 100}

	// Predicate: returns true if property holds, false if fails
	predicate := func(v eval.Value) bool {
		intVal := v.(*eval.IntValue)
		return intVal.Value < 50 // Property holds if < 50
	}

	// Shrink to minimal failure
	shrunk := runner.ShrinkValue(original, shrinker, predicate)
	shrunkInt := shrunk.(*eval.IntValue)

	// Should shrink 100 down to 50 (minimal value that fails the property)
	if shrunkInt.Value != 50 {
		t.Errorf("expected minimal failing value 50, got %d", shrunkInt.Value)
	}
}

// TestPropertyRunner_ShrinkList tests shrinking lists to minimal counterexample.
func TestPropertyRunner_ShrinkList(t *testing.T) {
	config := DefaultConfig()
	runner := NewPropertyRunner(config, 100)
	elemShrinker := NewIntShrinker()
	shrinker := NewListShrinker(elemShrinker)

	// Start with a large list containing a failing value
	original := &eval.ListValue{Elements: []eval.Value{
		&eval.IntValue{Value: 1},
		&eval.IntValue{Value: 2},
		&eval.IntValue{Value: 100}, // This value fails
		&eval.IntValue{Value: 4},
		&eval.IntValue{Value: 5},
	}}

	// Property: all elements must be < 50
	predicate := func(v eval.Value) bool {
		listVal := v.(*eval.ListValue)
		for _, elem := range listVal.Elements {
			if elem.(*eval.IntValue).Value >= 50 {
				return false // Property fails if any element >= 50
			}
		}
		return true // Property holds if all elements < 50
	}

	// Shrink to minimal failure
	shrunk := runner.ShrinkValue(original, shrinker, predicate)
	shrunkList := shrunk.(*eval.ListValue)

	// Should shrink to minimal failing case
	if len(shrunkList.Elements) >= len(original.Elements) {
		t.Errorf("shrunk list should be smaller than original: got %d, original %d",
			len(shrunkList.Elements), len(original.Elements))
	}

	// At least one element should still fail the property
	foundFailing := false
	for _, elem := range shrunkList.Elements {
		if elem.(*eval.IntValue).Value >= 50 {
			foundFailing = true
			break
		}
	}
	if !foundFailing {
		t.Error("shrunk list should still contain failing element")
	}
}

// TestPropertyRunner_ShrinkString tests shrinking strings to minimal counterexample.
func TestPropertyRunner_ShrinkString(t *testing.T) {
	config := DefaultConfig()
	runner := NewPropertyRunner(config, 100)
	shrinker := NewStringShrinker()

	// Property: string length must be < 5
	original := &eval.StringValue{Value: "hello world"} // 11 chars, fails

	predicate := func(v eval.Value) bool {
		strVal := v.(*eval.StringValue)
		return len(strVal.Value) < 5 // Property holds if < 5 chars
	}

	// Shrink to minimal failure
	shrunk := runner.ShrinkValue(original, shrinker, predicate)
	shrunkStr := shrunk.(*eval.StringValue)

	// Should shrink to a 5-character string (minimal failing length)
	if len(shrunkStr.Value) != 5 {
		t.Errorf("expected minimal failing length 5, got %d (%q)",
			len(shrunkStr.Value), shrunkStr.Value)
	}
}

// TestPropertyRunner_ShrinkNoSmallerFailure tests when no smaller failure exists.
func TestPropertyRunner_ShrinkNoSmallerFailure(t *testing.T) {
	config := DefaultConfig()
	runner := NewPropertyRunner(config, 100)
	shrinker := NewIntShrinker()

	// Property: value must be != 42
	original := &eval.IntValue{Value: 42}

	predicate := func(v eval.Value) bool {
		intVal := v.(*eval.IntValue)
		return intVal.Value != 42 // Only 42 fails
	}

	// Try to shrink
	shrunk := runner.ShrinkValue(original, shrinker, predicate)
	shrunkInt := shrunk.(*eval.IntValue)

	// Should stay at 42 (0 passes, so we can't shrink to 0)
	if shrunkInt.Value != 42 {
		t.Errorf("expected value to stay at 42, got %d", shrunkInt.Value)
	}
}

// TestPropertyRunner_ShrinkNilShrinker tests behavior with nil shrinker.
func TestPropertyRunner_ShrinkNilShrinker(t *testing.T) {
	config := DefaultConfig()
	runner := NewPropertyRunner(config, 100)

	original := &eval.IntValue{Value: 100}

	predicate := func(v eval.Value) bool {
		return v.(*eval.IntValue).Value < 50
	}

	// Shrink with nil shrinker should return original
	shrunk := runner.ShrinkValue(original, nil, predicate)

	if shrunk != original {
		t.Error("should return original when shrinker is nil")
	}
}
