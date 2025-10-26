package testing

import (
	"github.com/sunholo/ailang/internal/eval"
)

// Shrinker defines the interface for shrinking values.
// Shrinking is the process of finding simpler counterexamples when a property fails.
// For example, if a property fails on input 100, we might shrink to 50, 25, ..., 0
// to find the minimal failing case.
type Shrinker interface {
	// Shrink returns a list of simpler values to try.
	// The list should be ordered from "most simplified" to "least simplified".
	// For example, IntShrinker.Shrink(100) might return [0, 50, 75, 88, 94, 97, 99]
	Shrink(val eval.Value) []eval.Value
}

// IntShrinker shrinks integers toward zero.
type IntShrinker struct{}

// NewIntShrinker creates a new integer shrinker.
func NewIntShrinker() *IntShrinker {
	return &IntShrinker{}
}

// Shrink generates shrunk integer values.
// Strategy: Binary search toward zero, generating intermediate values.
// For positive N: [0, N/2, N/2 + N/4, N/2 + N/4 + N/8, ..., N-1]
// For negative N: [0, N/2, ..., N+1]
func (s *IntShrinker) Shrink(val eval.Value) []eval.Value {
	intVal, ok := val.(*eval.IntValue)
	if !ok {
		return nil // Can't shrink non-int
	}

	n := intVal.Value
	if n == 0 {
		return nil // Already minimal
	}

	var shrinks []eval.Value

	// Always try zero first (most simplified)
	shrinks = append(shrinks, &eval.IntValue{Value: 0})

	// Generate binary search path toward zero
	if n > 0 {
		// Positive: 0, n/2, n/2 + n/4, n/2 + n/4 + n/8, ..., n-1
		step := n
		current := 0
		for step > 1 {
			step = step / 2
			current += step
			if current < n {
				shrinks = append(shrinks, &eval.IntValue{Value: current})
			}
		}
		// Also try n-1 (minimal step)
		if n > 1 {
			shrinks = append(shrinks, &eval.IntValue{Value: n - 1})
		}
	} else {
		// Negative: 0, n/2, n/2 + n/4, ..., n+1
		step := -n // Make positive for easier math
		current := 0
		for step > 1 {
			step = step / 2
			current -= step
			if current > n {
				shrinks = append(shrinks, &eval.IntValue{Value: current})
			}
		}
		// Also try n+1 (minimal step toward zero)
		if n < -1 {
			shrinks = append(shrinks, &eval.IntValue{Value: n + 1})
		}
	}

	return shrinks
}

// FloatShrinker shrinks floats toward zero.
type FloatShrinker struct{}

// NewFloatShrinker creates a new float shrinker.
func NewFloatShrinker() *FloatShrinker {
	return &FloatShrinker{}
}

// Shrink generates shrunk float values.
// Strategy: Similar to int, but also try truncating decimals.
func (s *FloatShrinker) Shrink(val eval.Value) []eval.Value {
	floatVal, ok := val.(*eval.FloatValue)
	if !ok {
		return nil
	}

	f := floatVal.Value
	if f == 0.0 {
		return nil
	}

	var shrinks []eval.Value

	// Always try zero first
	shrinks = append(shrinks, &eval.FloatValue{Value: 0.0})

	// Try halving repeatedly
	current := f
	for i := 0; i < 10; i++ { // Limit iterations to prevent infinite loops
		current = current / 2.0
		if current == 0.0 {
			break
		}
		shrinks = append(shrinks, &eval.FloatValue{Value: current})
	}

	return shrinks
}

// StringShrinker shrinks strings toward empty string.
type StringShrinker struct{}

// NewStringShrinker creates a new string shrinker.
func NewStringShrinker() *StringShrinker {
	return &StringShrinker{}
}

// Shrink generates shrunk string values.
// Strategy:
// 1. Try empty string
// 2. Try removing half the string
// 3. Try removing individual characters
func (s *StringShrinker) Shrink(val eval.Value) []eval.Value {
	strVal, ok := val.(*eval.StringValue)
	if !ok {
		return nil
	}

	str := strVal.Value
	if len(str) == 0 {
		return nil
	}

	var shrinks []eval.Value

	// Try empty string
	shrinks = append(shrinks, &eval.StringValue{Value: ""})

	// Try removing halves
	runes := []rune(str)
	if len(runes) > 1 {
		// First half
		shrinks = append(shrinks, &eval.StringValue{Value: string(runes[:len(runes)/2])})
		// Second half
		shrinks = append(shrinks, &eval.StringValue{Value: string(runes[len(runes)/2:])})
	}

	// Try removing each character (limit to first 10 to avoid explosion)
	// Skip this for single character strings (already handled by empty string above)
	if len(runes) > 1 {
		limit := len(runes)
		if limit > 10 {
			limit = 10
		}
		for i := 0; i < limit; i++ {
			removed := make([]rune, 0, len(runes)-1)
			removed = append(removed, runes[:i]...)
			removed = append(removed, runes[i+1:]...)
			shrinks = append(shrinks, &eval.StringValue{Value: string(removed)})
		}
	}

	return shrinks
}

// ListShrinker shrinks lists by removing elements or shrinking elements.
type ListShrinker struct {
	elemShrinker Shrinker // Optional: shrinker for list elements
}

// NewListShrinker creates a new list shrinker.
func NewListShrinker(elemShrinker Shrinker) *ListShrinker {
	return &ListShrinker{elemShrinker: elemShrinker}
}

// Shrink generates shrunk list values.
// Strategy:
// 1. Try empty list
// 2. Try removing halves/chunks
// 3. Try removing individual elements
// 4. If elemShrinker provided, try shrinking each element
func (s *ListShrinker) Shrink(val eval.Value) []eval.Value {
	listVal, ok := val.(*eval.ListValue)
	if !ok {
		return nil
	}

	elems := listVal.Elements
	if len(elems) == 0 {
		return nil
	}

	var shrinks []eval.Value

	// Try empty list
	shrinks = append(shrinks, &eval.ListValue{Elements: []eval.Value{}})

	// Try removing chunks (halves, quarters, etc.)
	if len(elems) > 1 {
		// Remove first half
		shrinks = append(shrinks, &eval.ListValue{
			Elements: elems[len(elems)/2:],
		})
		// Remove second half
		shrinks = append(shrinks, &eval.ListValue{
			Elements: elems[:len(elems)/2],
		})
	}

	// Try removing each element (limit to first 10)
	limit := len(elems)
	if limit > 10 {
		limit = 10
	}
	for i := 0; i < limit; i++ {
		removed := make([]eval.Value, 0, len(elems)-1)
		removed = append(removed, elems[:i]...)
		removed = append(removed, elems[i+1:]...)
		shrinks = append(shrinks, &eval.ListValue{Elements: removed})
	}

	// If we have an element shrinker, try shrinking each element
	if s.elemShrinker != nil {
		for i := 0; i < len(elems) && i < 5; i++ { // Limit to first 5 elements
			elemShrinks := s.elemShrinker.Shrink(elems[i])
			for _, shrunkElem := range elemShrinks {
				newElems := make([]eval.Value, len(elems))
				copy(newElems, elems)
				newElems[i] = shrunkElem
				shrinks = append(shrinks, &eval.ListValue{Elements: newElems})
			}
		}
	}

	return shrinks
}

// ADTShrinker shrinks ADT values by shrinking their fields.
type ADTShrinker struct {
	fieldShrinkers []Shrinker // Shrinkers for each field
}

// NewADTShrinker creates a new ADT shrinker.
func NewADTShrinker(fieldShrinkers []Shrinker) *ADTShrinker {
	return &ADTShrinker{fieldShrinkers: fieldShrinkers}
}

// Shrink generates shrunk ADT values.
// Strategy: Try shrinking each field independently.
// For nullary constructors (no fields), return nil (can't shrink).
func (s *ADTShrinker) Shrink(val eval.Value) []eval.Value {
	taggedVal, ok := val.(*eval.TaggedValue)
	if !ok {
		return nil
	}

	fields := taggedVal.Fields
	if len(fields) == 0 {
		return nil // Can't shrink nullary constructor
	}

	var shrinks []eval.Value

	// Try shrinking each field
	for i, field := range fields {
		if i < len(s.fieldShrinkers) && s.fieldShrinkers[i] != nil {
			fieldShrinks := s.fieldShrinkers[i].Shrink(field)
			for _, shrunkField := range fieldShrinks {
				newFields := make([]eval.Value, len(fields))
				copy(newFields, fields)
				newFields[i] = shrunkField

				shrinks = append(shrinks, &eval.TaggedValue{
					ModulePath: taggedVal.ModulePath,
					TypeName:   taggedVal.TypeName,
					CtorName:   taggedVal.CtorName,
					Fields:     newFields,
				})
			}
		}
	}

	return shrinks
}

// NoOpShrinker is a shrinker that doesn't shrink (returns no shrinks).
// Useful for values that can't be simplified (e.g., booleans, unit).
type NoOpShrinker struct{}

// NewNoOpShrinker creates a new no-op shrinker.
func NewNoOpShrinker() *NoOpShrinker {
	return &NoOpShrinker{}
}

// Shrink returns no shrinks.
func (s *NoOpShrinker) Shrink(val eval.Value) []eval.Value {
	return nil
}
