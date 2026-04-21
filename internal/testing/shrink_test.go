package testing

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/sunholo-data/ailang/internal/eval"
)

// TestIntShrinker_Zero tests that zero cannot be shrunk.
func TestIntShrinker_Zero(t *testing.T) {
	shrinker := NewIntShrinker()
	val := &eval.IntValue{Value: 0}

	shrinks := shrinker.Shrink(val)
	assert.Nil(t, shrinks, "zero should not shrink")
}

// TestIntShrinker_Positive tests shrinking positive integers.
func TestIntShrinker_Positive(t *testing.T) {
	shrinker := NewIntShrinker()
	val := &eval.IntValue{Value: 100}

	shrinks := shrinker.Shrink(val)
	assert.NotNil(t, shrinks, "should have shrinks")
	assert.Greater(t, len(shrinks), 0, "should have at least one shrink")

	// First shrink should be zero
	assert.Equal(t, 0, shrinks[0].(*eval.IntValue).Value, "first shrink should be 0")

	// All shrinks should be smaller than original
	for _, shrink := range shrinks {
		intShrink := shrink.(*eval.IntValue)
		assert.Less(t, intShrink.Value, 100, "shrink %d should be < 100", intShrink.Value)
	}

	// Should include binary search path
	// For 100: [0, 50, 75, 88, 94, 97, 99]
	values := make(map[int]bool)
	for _, shrink := range shrinks {
		values[shrink.(*eval.IntValue).Value] = true
	}
	assert.True(t, values[0], "should include 0")
	assert.True(t, values[50], "should include 50 (100/2)")
	assert.True(t, values[99], "should include 99 (100-1)")
}

// TestIntShrinker_Negative tests shrinking negative integers.
func TestIntShrinker_Negative(t *testing.T) {
	shrinker := NewIntShrinker()
	val := &eval.IntValue{Value: -100}

	shrinks := shrinker.Shrink(val)
	assert.NotNil(t, shrinks, "should have shrinks")

	// First shrink should be zero
	assert.Equal(t, 0, shrinks[0].(*eval.IntValue).Value, "first shrink should be 0")

	// All shrinks should be closer to zero than original
	for _, shrink := range shrinks {
		intShrink := shrink.(*eval.IntValue)
		assert.Greater(t, intShrink.Value, -100, "shrink %d should be > -100", intShrink.Value)
	}

	// Should include -99 (minimal step toward zero)
	values := make(map[int]bool)
	for _, shrink := range shrinks {
		values[shrink.(*eval.IntValue).Value] = true
	}
	assert.True(t, values[-99], "should include -99 (-100+1)")
}

// TestIntShrinker_Small tests shrinking small positive integers.
func TestIntShrinker_Small(t *testing.T) {
	shrinker := NewIntShrinker()
	val := &eval.IntValue{Value: 1}

	shrinks := shrinker.Shrink(val)
	assert.NotNil(t, shrinks, "should have shrinks")
	assert.Equal(t, 1, len(shrinks), "1 should shrink to just [0]")
	assert.Equal(t, 0, shrinks[0].(*eval.IntValue).Value, "should shrink to 0")
}

// TestIntShrinker_WrongType tests that non-int values return nil.
func TestIntShrinker_WrongType(t *testing.T) {
	shrinker := NewIntShrinker()
	val := &eval.StringValue{Value: "not an int"}

	shrinks := shrinker.Shrink(val)
	assert.Nil(t, shrinks, "should return nil for non-int")
}

// TestFloatShrinker_Zero tests that zero cannot be shrunk.
func TestFloatShrinker_Zero(t *testing.T) {
	shrinker := NewFloatShrinker()
	val := &eval.FloatValue{Value: 0.0}

	shrinks := shrinker.Shrink(val)
	assert.Nil(t, shrinks, "zero should not shrink")
}

// TestFloatShrinker_Positive tests shrinking positive floats.
func TestFloatShrinker_Positive(t *testing.T) {
	shrinker := NewFloatShrinker()
	val := &eval.FloatValue{Value: 100.0}

	shrinks := shrinker.Shrink(val)
	assert.NotNil(t, shrinks, "should have shrinks")

	// First shrink should be zero
	assert.Equal(t, 0.0, shrinks[0].(*eval.FloatValue).Value, "first shrink should be 0.0")

	// All shrinks should be smaller than original
	for _, shrink := range shrinks {
		floatShrink := shrink.(*eval.FloatValue)
		assert.Less(t, floatShrink.Value, 100.0, "shrink %f should be < 100.0", floatShrink.Value)
	}

	// Should include halvings: 50.0, 25.0, 12.5, ...
	values := make(map[float64]bool)
	for _, shrink := range shrinks {
		values[shrink.(*eval.FloatValue).Value] = true
	}
	assert.True(t, values[50.0], "should include 50.0 (100/2)")
	assert.True(t, values[25.0], "should include 25.0 (50/2)")
}

// TestFloatShrinker_WrongType tests that non-float values return nil.
func TestFloatShrinker_WrongType(t *testing.T) {
	shrinker := NewFloatShrinker()
	val := &eval.IntValue{Value: 42}

	shrinks := shrinker.Shrink(val)
	assert.Nil(t, shrinks, "should return nil for non-float")
}

// TestStringShrinker_Empty tests that empty string cannot be shrunk.
func TestStringShrinker_Empty(t *testing.T) {
	shrinker := NewStringShrinker()
	val := &eval.StringValue{Value: ""}

	shrinks := shrinker.Shrink(val)
	assert.Nil(t, shrinks, "empty string should not shrink")
}

// TestStringShrinker_SingleChar tests shrinking single character.
func TestStringShrinker_SingleChar(t *testing.T) {
	shrinker := NewStringShrinker()
	val := &eval.StringValue{Value: "a"}

	shrinks := shrinker.Shrink(val)
	assert.NotNil(t, shrinks, "should have shrinks")
	assert.Equal(t, 1, len(shrinks), "single char should shrink to just [\"\"]")
	assert.Equal(t, "", shrinks[0].(*eval.StringValue).Value, "should shrink to empty")
}

// TestStringShrinker_MultiChar tests shrinking multi-character strings.
func TestStringShrinker_MultiChar(t *testing.T) {
	shrinker := NewStringShrinker()
	val := &eval.StringValue{Value: "hello"}

	shrinks := shrinker.Shrink(val)
	assert.NotNil(t, shrinks, "should have shrinks")

	// First shrink should be empty string
	assert.Equal(t, "", shrinks[0].(*eval.StringValue).Value, "first shrink should be empty")

	// Should include halves
	found := false
	for _, shrink := range shrinks {
		if shrink.(*eval.StringValue).Value == "he" {
			found = true
			break
		}
	}
	assert.True(t, found, "should include first half \"he\"")

	// All shrinks should be shorter than original
	for _, shrink := range shrinks {
		strShrink := shrink.(*eval.StringValue)
		assert.Less(t, len(strShrink.Value), len("hello"), "shrink %q should be shorter", strShrink.Value)
	}
}

// TestStringShrinker_Unicode tests shrinking Unicode strings.
func TestStringShrinker_Unicode(t *testing.T) {
	shrinker := NewStringShrinker()
	val := &eval.StringValue{Value: "🎉🎊🎈"}

	shrinks := shrinker.Shrink(val)
	assert.NotNil(t, shrinks, "should have shrinks")

	// First shrink should be empty
	assert.Equal(t, "", shrinks[0].(*eval.StringValue).Value, "first shrink should be empty")

	// Should handle Unicode correctly (count runes, not bytes)
	for _, shrink := range shrinks {
		strShrink := shrink.(*eval.StringValue)
		assert.Less(t, len([]rune(strShrink.Value)), len([]rune("🎉🎊🎈")), "shrink should have fewer runes")
	}
}

// TestStringShrinker_WrongType tests that non-string values return nil.
func TestStringShrinker_WrongType(t *testing.T) {
	shrinker := NewStringShrinker()
	val := &eval.IntValue{Value: 42}

	shrinks := shrinker.Shrink(val)
	assert.Nil(t, shrinks, "should return nil for non-string")
}

// TestListShrinker_Empty tests that empty list cannot be shrunk.
func TestListShrinker_Empty(t *testing.T) {
	shrinker := NewListShrinker(nil)
	val := &eval.ListValue{Elements: []eval.Value{}}

	shrinks := shrinker.Shrink(val)
	assert.Nil(t, shrinks, "empty list should not shrink")
}

// TestListShrinker_SingleElement tests shrinking single-element list.
func TestListShrinker_SingleElement(t *testing.T) {
	shrinker := NewListShrinker(nil)
	val := &eval.ListValue{Elements: []eval.Value{
		&eval.IntValue{Value: 42},
	}}

	shrinks := shrinker.Shrink(val)
	assert.NotNil(t, shrinks, "should have shrinks")

	// First shrink should be empty list
	assert.Equal(t, 0, len(shrinks[0].(*eval.ListValue).Elements), "first shrink should be empty list")
}

// TestListShrinker_MultiElement tests shrinking multi-element list.
func TestListShrinker_MultiElement(t *testing.T) {
	shrinker := NewListShrinker(nil)
	val := &eval.ListValue{Elements: []eval.Value{
		&eval.IntValue{Value: 1},
		&eval.IntValue{Value: 2},
		&eval.IntValue{Value: 3},
		&eval.IntValue{Value: 4},
	}}

	shrinks := shrinker.Shrink(val)
	assert.NotNil(t, shrinks, "should have shrinks")

	// First shrink should be empty list
	assert.Equal(t, 0, len(shrinks[0].(*eval.ListValue).Elements), "first shrink should be empty list")

	// Should include removing halves
	foundHalf := false
	for _, shrink := range shrinks {
		list := shrink.(*eval.ListValue)
		if len(list.Elements) == 2 {
			foundHalf = true
			break
		}
	}
	assert.True(t, foundHalf, "should include list with half elements removed")

	// All shrinks should have fewer elements
	for _, shrink := range shrinks {
		listShrink := shrink.(*eval.ListValue)
		assert.LessOrEqual(t, len(listShrink.Elements), 4, "shrink should have <= 4 elements")
	}
}

// TestListShrinker_WithElementShrinker tests shrinking list elements.
func TestListShrinker_WithElementShrinker(t *testing.T) {
	elemShrinker := NewIntShrinker()
	shrinker := NewListShrinker(elemShrinker)
	val := &eval.ListValue{Elements: []eval.Value{
		&eval.IntValue{Value: 10},
		&eval.IntValue{Value: 20},
	}}

	shrinks := shrinker.Shrink(val)
	assert.NotNil(t, shrinks, "should have shrinks")

	// Should include shrinking first element to 0
	found := false
	for _, shrink := range shrinks {
		list := shrink.(*eval.ListValue)
		if len(list.Elements) == 2 &&
			list.Elements[0].(*eval.IntValue).Value == 0 &&
			list.Elements[1].(*eval.IntValue).Value == 20 {
			found = true
			break
		}
	}
	assert.True(t, found, "should include [0, 20] (first element shrunk)")
}

// TestListShrinker_WrongType tests that non-list values return nil.
func TestListShrinker_WrongType(t *testing.T) {
	shrinker := NewListShrinker(nil)
	val := &eval.IntValue{Value: 42}

	shrinks := shrinker.Shrink(val)
	assert.Nil(t, shrinks, "should return nil for non-list")
}

// TestADTShrinker_Nullary tests that nullary constructors can't be shrunk.
func TestADTShrinker_Nullary(t *testing.T) {
	shrinker := NewADTShrinker([]Shrinker{})
	val := &eval.TaggedValue{
		ModulePath: "test",
		TypeName:   "Option",
		CtorName:   "None",
		Fields:     []eval.Value{},
	}

	shrinks := shrinker.Shrink(val)
	assert.Nil(t, shrinks, "nullary constructor should not shrink")
}

// TestADTShrinker_Unary tests shrinking unary constructor.
func TestADTShrinker_Unary(t *testing.T) {
	fieldShrinker := NewIntShrinker()
	shrinker := NewADTShrinker([]Shrinker{fieldShrinker})
	val := &eval.TaggedValue{
		ModulePath: "test",
		TypeName:   "Option",
		CtorName:   "Some",
		Fields: []eval.Value{
			&eval.IntValue{Value: 100},
		},
	}

	shrinks := shrinker.Shrink(val)
	assert.NotNil(t, shrinks, "should have shrinks")
	assert.Greater(t, len(shrinks), 0, "should have at least one shrink")

	// Should include Some(0)
	found := false
	for _, shrink := range shrinks {
		tagged := shrink.(*eval.TaggedValue)
		if tagged.CtorName == "Some" &&
			len(tagged.Fields) == 1 &&
			tagged.Fields[0].(*eval.IntValue).Value == 0 {
			found = true
			break
		}
	}
	assert.True(t, found, "should include Some(0)")

	// All shrinks should keep the constructor name
	for _, shrink := range shrinks {
		tagged := shrink.(*eval.TaggedValue)
		assert.Equal(t, "Some", tagged.CtorName, "constructor name should be preserved")
	}
}

// TestADTShrinker_Nary tests shrinking n-ary constructor.
func TestADTShrinker_Nary(t *testing.T) {
	fieldShrinkers := []Shrinker{
		NewIntShrinker(),
		NewStringShrinker(),
	}
	shrinker := NewADTShrinker(fieldShrinkers)
	val := &eval.TaggedValue{
		ModulePath: "test",
		TypeName:   "Pair",
		CtorName:   "Pair",
		Fields: []eval.Value{
			&eval.IntValue{Value: 42},
			&eval.StringValue{Value: "hello"},
		},
	}

	shrinks := shrinker.Shrink(val)
	assert.NotNil(t, shrinks, "should have shrinks")

	// Should include Pair(0, "hello") (first field shrunk)
	foundFirst := false
	for _, shrink := range shrinks {
		tagged := shrink.(*eval.TaggedValue)
		if len(tagged.Fields) == 2 &&
			tagged.Fields[0].(*eval.IntValue).Value == 0 &&
			tagged.Fields[1].(*eval.StringValue).Value == "hello" {
			foundFirst = true
			break
		}
	}
	assert.True(t, foundFirst, "should include Pair(0, \"hello\")")

	// Should include Pair(42, "") (second field shrunk)
	foundSecond := false
	for _, shrink := range shrinks {
		tagged := shrink.(*eval.TaggedValue)
		if len(tagged.Fields) == 2 &&
			tagged.Fields[0].(*eval.IntValue).Value == 42 &&
			tagged.Fields[1].(*eval.StringValue).Value == "" {
			foundSecond = true
			break
		}
	}
	assert.True(t, foundSecond, "should include Pair(42, \"\")")
}

// TestADTShrinker_WrongType tests that non-ADT values return nil.
func TestADTShrinker_WrongType(t *testing.T) {
	shrinker := NewADTShrinker([]Shrinker{})
	val := &eval.IntValue{Value: 42}

	shrinks := shrinker.Shrink(val)
	assert.Nil(t, shrinks, "should return nil for non-ADT")
}

// TestNoOpShrinker tests that NoOpShrinker never shrinks.
func TestNoOpShrinker(t *testing.T) {
	shrinker := NewNoOpShrinker()

	// Test various value types
	testCases := []eval.Value{
		&eval.IntValue{Value: 42},
		&eval.BoolValue{Value: true},
		&eval.StringValue{Value: "test"},
		&eval.ListValue{Elements: []eval.Value{}},
	}

	for _, val := range testCases {
		shrinks := shrinker.Shrink(val)
		assert.Nil(t, shrinks, "NoOpShrinker should never return shrinks")
	}
}

// TestShrinkingInterface tests that all shrinkers implement the interface.
func TestShrinkingInterface(t *testing.T) {
	var _ Shrinker = (*IntShrinker)(nil)
	var _ Shrinker = (*FloatShrinker)(nil)
	var _ Shrinker = (*StringShrinker)(nil)
	var _ Shrinker = (*ListShrinker)(nil)
	var _ Shrinker = (*ADTShrinker)(nil)
	var _ Shrinker = (*NoOpShrinker)(nil)
}
