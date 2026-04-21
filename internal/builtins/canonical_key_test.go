package builtins

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
)

// TestCanonicalKey_Determinism verifies that canonicalKey produces identical
// output for the same value across multiple runs. Run with -count=20.
func TestCanonicalKey_Determinism(t *testing.T) {
	values := map[string]eval.Value{
		"int":    &eval.IntValue{Value: 42},
		"float":  &eval.FloatValue{Value: 3.14},
		"string": &eval.StringValue{Value: "hello"},
		"bool":   &eval.BoolValue{Value: true},
		"unit":   &eval.UnitValue{},
		"list": &eval.ListValue{Elements: []eval.Value{
			&eval.IntValue{Value: 1},
			&eval.StringValue{Value: "two"},
			&eval.BoolValue{Value: false},
		}},
		"record": &eval.RecordValue{Fields: map[string]eval.Value{
			"z": &eval.IntValue{Value: 3},
			"a": &eval.IntValue{Value: 1},
			"m": &eval.IntValue{Value: 2},
		}},
		"tagged": &eval.TaggedValue{
			CtorName: "Some",
			Fields:   []eval.Value{&eval.IntValue{Value: 42}},
		},
		"nested_record": &eval.RecordValue{Fields: map[string]eval.Value{
			"inner": &eval.RecordValue{Fields: map[string]eval.Value{
				"b": &eval.StringValue{Value: "deep"},
				"a": &eval.IntValue{Value: 1},
			}},
			"list": &eval.ListValue{Elements: []eval.Value{
				&eval.IntValue{Value: 10},
			}},
		}},
	}

	// Store first-run keys
	expected := make(map[string]string)
	for name, v := range values {
		expected[name] = canonicalKey(v)
	}

	// Verify determinism (meaningful with -count=20)
	for name, v := range values {
		got := canonicalKey(v)
		assert.Equal(t, expected[name], got, "canonicalKey not deterministic for %s", name)
	}
}

// TestCanonicalKey_TypeTagging verifies that different-typed values with
// the same "content" produce different canonical keys.
func TestCanonicalKey_TypeTagging(t *testing.T) {
	tests := []struct {
		name string
		a, b eval.Value
	}{
		{"int vs string", &eval.IntValue{Value: 1}, &eval.StringValue{Value: "1"}},
		{"int vs float", &eval.IntValue{Value: 1}, &eval.FloatValue{Value: 1.0}},
		{"int vs bool", &eval.IntValue{Value: 1}, &eval.BoolValue{Value: true}},
		{"string vs bool", &eval.StringValue{Value: "true"}, &eval.BoolValue{Value: true}},
		{"list vs array", &eval.ListValue{Elements: []eval.Value{&eval.IntValue{Value: 1}}},
			&eval.ArrayValue{Elements: []eval.Value{&eval.IntValue{Value: 1}}}},
		{"list vs tuple", &eval.ListValue{Elements: []eval.Value{&eval.IntValue{Value: 1}}},
			&eval.TupleValue{Elements: []eval.Value{&eval.IntValue{Value: 1}}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keyA := canonicalKey(tt.a)
			keyB := canonicalKey(tt.b)
			assert.NotEqual(t, keyA, keyB, "different types should produce different keys")
		})
	}
}

// TestCanonicalKey_RecordKeyOrder verifies records are sorted by key name,
// producing the same canonical key regardless of Go map iteration order.
func TestCanonicalKey_RecordKeyOrder(t *testing.T) {
	// Create two records with same fields but different insertion order
	r1 := &eval.RecordValue{Fields: map[string]eval.Value{
		"z": &eval.IntValue{Value: 3},
		"a": &eval.IntValue{Value: 1},
		"m": &eval.IntValue{Value: 2},
	}}
	r2 := &eval.RecordValue{Fields: map[string]eval.Value{
		"a": &eval.IntValue{Value: 1},
		"m": &eval.IntValue{Value: 2},
		"z": &eval.IntValue{Value: 3},
	}}

	key1 := canonicalKey(r1)
	key2 := canonicalKey(r2)
	assert.Equal(t, key1, key2, "records with same fields should have same canonical key")
	// Verify sorted order in key
	assert.Contains(t, key1, "a:")
	assert.Contains(t, key1, "r:{")
}

// TestCanonicalKey_NestedStructures verifies correct encoding of deeply nested values.
func TestCanonicalKey_NestedStructures(t *testing.T) {
	// 3 levels deep: record -> list -> tagged -> int
	v := &eval.RecordValue{Fields: map[string]eval.Value{
		"items": &eval.ListValue{Elements: []eval.Value{
			&eval.TaggedValue{CtorName: "Ok", Fields: []eval.Value{&eval.IntValue{Value: 42}}},
			&eval.TaggedValue{CtorName: "Err", Fields: []eval.Value{&eval.StringValue{Value: "fail"}}},
		}},
	}}

	key := canonicalKey(v)
	assert.NotEmpty(t, key)
	assert.Contains(t, key, "r:{")
	assert.Contains(t, key, "l:[")
	assert.Contains(t, key, "t:Ok(")
	assert.Contains(t, key, "t:Err(")
}

// TestCanonicalKey_EdgeCases verifies behavior for edge cases.
func TestCanonicalKey_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		value    eval.Value
		expected string
	}{
		{"empty list", &eval.ListValue{Elements: nil}, "l:[]"},
		{"empty record", &eval.RecordValue{Fields: map[string]eval.Value{}}, "r:{}"},
		{"empty tuple", &eval.TupleValue{Elements: nil}, "tp:()"},
		{"zero int", &eval.IntValue{Value: 0}, "i:0"},
		{"negative int", &eval.IntValue{Value: -1}, "i:-1"},
		{"empty string", &eval.StringValue{Value: ""}, "s:0:"},
		{"unit", &eval.UnitValue{}, "u"},
		{"tagged no fields", &eval.TaggedValue{CtorName: "None", Fields: nil}, "t:None()"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := canonicalKey(tt.value)
			assert.Equal(t, tt.expected, got)
		})
	}
}

// TestCanonicalKey_StringLengthPrefix verifies that strings with similar
// prefixes don't collide due to length-prefixing.
func TestCanonicalKey_StringLengthPrefix(t *testing.T) {
	a := &eval.StringValue{Value: "ab"}
	b := &eval.StringValue{Value: "a"}
	keyA := canonicalKey(a)
	keyB := canonicalKey(b)
	assert.NotEqual(t, keyA, keyB)
	assert.Equal(t, "s:2:ab", keyA)
	assert.Equal(t, "s:1:a", keyB)
}

// TestValuesEqual_StructuralComparison verifies the new structural valuesEqual
// handles all value types correctly (replacing reflect.DeepEqual).
func TestValuesEqual_StructuralComparison(t *testing.T) {
	tests := []struct {
		name     string
		a, b     eval.Value
		expected bool
	}{
		{"int equal", &eval.IntValue{Value: 42}, &eval.IntValue{Value: 42}, true},
		{"int not equal", &eval.IntValue{Value: 1}, &eval.IntValue{Value: 2}, false},
		{"float equal", &eval.FloatValue{Value: 3.14}, &eval.FloatValue{Value: 3.14}, true},
		{"string equal", &eval.StringValue{Value: "hi"}, &eval.StringValue{Value: "hi"}, true},
		{"string not equal", &eval.StringValue{Value: "a"}, &eval.StringValue{Value: "b"}, false},
		{"bool equal", &eval.BoolValue{Value: true}, &eval.BoolValue{Value: true}, true},
		{"bool not equal", &eval.BoolValue{Value: true}, &eval.BoolValue{Value: false}, false},
		{"unit equal", &eval.UnitValue{}, &eval.UnitValue{}, true},
		{"cross-type", &eval.IntValue{Value: 1}, &eval.StringValue{Value: "1"}, false},

		// Records
		{"record equal", &eval.RecordValue{Fields: map[string]eval.Value{
			"x": &eval.IntValue{Value: 1},
			"y": &eval.IntValue{Value: 2},
		}}, &eval.RecordValue{Fields: map[string]eval.Value{
			"x": &eval.IntValue{Value: 1},
			"y": &eval.IntValue{Value: 2},
		}}, true},
		{"record not equal", &eval.RecordValue{Fields: map[string]eval.Value{
			"x": &eval.IntValue{Value: 1},
		}}, &eval.RecordValue{Fields: map[string]eval.Value{
			"x": &eval.IntValue{Value: 2},
		}}, false},
		{"record different keys", &eval.RecordValue{Fields: map[string]eval.Value{
			"x": &eval.IntValue{Value: 1},
		}}, &eval.RecordValue{Fields: map[string]eval.Value{
			"y": &eval.IntValue{Value: 1},
		}}, false},

		// Lists
		{"list equal", &eval.ListValue{Elements: []eval.Value{
			&eval.IntValue{Value: 1}, &eval.IntValue{Value: 2},
		}}, &eval.ListValue{Elements: []eval.Value{
			&eval.IntValue{Value: 1}, &eval.IntValue{Value: 2},
		}}, true},
		{"list different length", &eval.ListValue{Elements: []eval.Value{
			&eval.IntValue{Value: 1},
		}}, &eval.ListValue{Elements: []eval.Value{
			&eval.IntValue{Value: 1}, &eval.IntValue{Value: 2},
		}}, false},

		// Tagged values (ADTs)
		{"tagged equal", &eval.TaggedValue{CtorName: "Some", Fields: []eval.Value{
			&eval.IntValue{Value: 42},
		}}, &eval.TaggedValue{CtorName: "Some", Fields: []eval.Value{
			&eval.IntValue{Value: 42},
		}}, true},
		{"tagged different ctor", &eval.TaggedValue{CtorName: "Some", Fields: []eval.Value{
			&eval.IntValue{Value: 42},
		}}, &eval.TaggedValue{CtorName: "None", Fields: nil}, false},

		// Tuples
		{"tuple equal", &eval.TupleValue{Elements: []eval.Value{
			&eval.IntValue{Value: 1}, &eval.StringValue{Value: "a"},
		}}, &eval.TupleValue{Elements: []eval.Value{
			&eval.IntValue{Value: 1}, &eval.StringValue{Value: "a"},
		}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := valuesEqual(tt.a, tt.b)
			assert.Equal(t, tt.expected, got)
		})
	}
}

// generateNStrings creates a list of N unique string values for benchmarking.
func generateNStrings(n int) *eval.ListValue {
	elems := make([]eval.Value, n)
	for i := 0; i < n; i++ {
		elems[i] = &eval.StringValue{Value: fmt.Sprintf("word_%d", i)}
	}
	return &eval.ListValue{Elements: elems}
}

// generateNStringsWithDuplicates creates a list of N strings where ~half are duplicates.
func generateNStringsWithDuplicates(n int) *eval.ListValue {
	elems := make([]eval.Value, n)
	for i := 0; i < n; i++ {
		elems[i] = &eval.StringValue{Value: fmt.Sprintf("word_%d", i%(n/2))}
	}
	return &eval.ListValue{Elements: elems}
}

func benchmarkDedup(b *testing.B, n int) {
	list := generateNStringsWithDuplicates(n)
	args := []eval.Value{list}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = listDedupImpl(nil, args)
	}
}

func BenchmarkDedup100(b *testing.B)   { benchmarkDedup(b, 100) }
func BenchmarkDedup1000(b *testing.B)  { benchmarkDedup(b, 1000) }
func BenchmarkDedup5000(b *testing.B)  { benchmarkDedup(b, 5000) }
func BenchmarkDedup10000(b *testing.B) { benchmarkDedup(b, 10000) }

func benchmarkIntersect(b *testing.B, n int) {
	list1 := generateNStrings(n)
	list2 := generateNStrings(n) // same strings — worst case for intersection
	args := []eval.Value{list1, list2}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = listIntersectImpl(nil, args)
	}
}

func BenchmarkIntersect1000(b *testing.B) { benchmarkIntersect(b, 1000) }
func BenchmarkIntersect5000(b *testing.B) { benchmarkIntersect(b, 5000) }

func benchmarkUnion(b *testing.B, n int) {
	list1 := generateNStrings(n)
	// Create overlapping but not identical list
	elems := make([]eval.Value, n)
	for i := 0; i < n; i++ {
		elems[i] = &eval.StringValue{Value: fmt.Sprintf("word_%d", i+n/2)}
	}
	list2 := &eval.ListValue{Elements: elems}
	args := []eval.Value{list1, list2}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = listUnionImpl(nil, args)
	}
}

func BenchmarkUnion5000(b *testing.B) { benchmarkUnion(b, 5000) }

func BenchmarkCanonicalKeyNestedRecord(b *testing.B) {
	// 3-level nested record
	v := &eval.RecordValue{Fields: map[string]eval.Value{
		"level1": &eval.RecordValue{Fields: map[string]eval.Value{
			"level2": &eval.RecordValue{Fields: map[string]eval.Value{
				"a": &eval.IntValue{Value: 1},
				"b": &eval.StringValue{Value: "hello"},
				"c": &eval.ListValue{Elements: []eval.Value{
					&eval.IntValue{Value: 10},
					&eval.IntValue{Value: 20},
				}},
			}},
		}},
	}}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = canonicalKey(v)
	}
}

// TestSetOps_WithRecords verifies hash-accelerated set ops work correctly
// with record values (the main type that previously used reflect.DeepEqual).
func TestSetOps_WithRecords(t *testing.T) {
	r1 := &eval.RecordValue{Fields: map[string]eval.Value{
		"x": &eval.IntValue{Value: 1},
		"y": &eval.IntValue{Value: 2},
	}}
	r2 := &eval.RecordValue{Fields: map[string]eval.Value{
		"x": &eval.IntValue{Value: 1},
		"y": &eval.IntValue{Value: 2},
	}}
	r3 := &eval.RecordValue{Fields: map[string]eval.Value{
		"x": &eval.IntValue{Value: 3},
		"y": &eval.IntValue{Value: 4},
	}}

	ctx := &effects.EffContext{}

	// dedup with duplicate records
	list := &eval.ListValue{Elements: []eval.Value{r1, r2, r3}}
	result, err := listDedupImpl(ctx, []eval.Value{list})
	assert.NoError(t, err)
	deduped := result.(*eval.ListValue)
	assert.Len(t, deduped.Elements, 2, "should deduplicate identical records")

	// intersect with records
	list1 := &eval.ListValue{Elements: []eval.Value{r1, r3}}
	list2 := &eval.ListValue{Elements: []eval.Value{r2}} // r2 == r1 structurally
	result, err = listIntersectImpl(ctx, []eval.Value{list1, list2})
	assert.NoError(t, err)
	inter := result.(*eval.ListValue)
	assert.Len(t, inter.Elements, 1, "should find record intersection")

	// union with records
	result, err = listUnionImpl(ctx, []eval.Value{list1, list2})
	assert.NoError(t, err)
	uni := result.(*eval.ListValue)
	assert.Len(t, uni.Elements, 2, "union should have 2 unique records")

	// difference with records
	result, err = listDifferenceImpl(ctx, []eval.Value{list1, list2})
	assert.NoError(t, err)
	diff := result.(*eval.ListValue)
	assert.Len(t, diff.Elements, 1, "difference should have 1 record")
}
