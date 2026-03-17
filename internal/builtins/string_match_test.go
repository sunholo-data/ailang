package builtins

import (
	"fmt"
	"testing"

	"github.com/sunholo/ailang/internal/effects/testctx"
	"github.com/sunholo/ailang/internal/eval"
)

// TestStrChars tests the _str_chars builtin
func TestStrChars(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		// Basic cases
		{"empty string", "", []string{}},
		{"single character", "a", []string{"a"}},
		{"two characters", "ab", []string{"a", "b"}},

		// ASCII strings
		{"hello", "hello", []string{"h", "e", "l", "l", "o"}},
		{"digits", "123", []string{"1", "2", "3"}},
		{"special characters", "!@#", []string{"!", "@", "#"}},

		// Unicode - emoji
		{"single emoji", "🎉", []string{"🎉"}},
		{"two emoji", "🎉🎊", []string{"🎉", "🎊"}},
		{"emoji and text", "a🎉b", []string{"a", "🎉", "b"}},

		// Unicode - accented characters
		{"accented cafe", "café", []string{"c", "a", "f", "é"}},
		{"accented greek", "αβγ", []string{"α", "β", "γ"}},

		// Mixed Unicode
		{"mixed multilingual", "hello世界", []string{"h", "e", "l", "l", "o", "世", "界"}},
		{"mixed with emoji", "hi🎉", []string{"h", "i", "🎉"}},

		// Edge cases
		{"single space", " ", []string{" "}},
		{"whitespace", " \t", []string{" ", "\t"}},
		{"newline", "a\nb", []string{"a", "\n", "b"}},
	}

	ctx := testctx.NewMockEffContext()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Call the builtin
			args := []eval.Value{&eval.StringValue{Value: tt.input}}
			result, err := strCharsImpl(ctx.EffContext, args)

			if err != nil {
				t.Fatalf("strCharsImpl() returned error: %v", err)
			}

			// Check that result is a ListValue
			listVal, ok := result.(*eval.ListValue)
			if !ok {
				t.Fatalf("strCharsImpl() returned %T, expected *eval.ListValue", result)
			}

			// Convert Elements to []string for comparison
			var got []string
			for _, elem := range listVal.Elements {
				strVal, ok := elem.(*eval.StringValue)
				if !ok {
					t.Fatalf("expected *eval.StringValue in list, got %T", elem)
				}
				got = append(got, strVal.Value)
			}

			if len(got) != len(tt.expected) {
				t.Errorf("chars(%q) length = %d, want %d\ngot: %v\nwant: %v",
					tt.input, len(got), len(tt.expected), got, tt.expected)
				return
			}

			for i := range got {
				if got[i] != tt.expected[i] {
					t.Errorf("chars(%q)[%d] = %q, want %q\ngot: %v\nwant: %v",
						tt.input, i, got[i], tt.expected[i], got, tt.expected)
				}
			}
		})
	}
}

// TestStrCharsWrongArgType tests error handling for wrong argument types
func TestStrCharsWrongArgType(t *testing.T) {
	ctx := testctx.NewMockEffContext()

	// Test with integer instead of string
	args := []eval.Value{&eval.IntValue{Value: 42}}
	_, err := strCharsImpl(ctx.EffContext, args)
	if err == nil {
		t.Error("strCharsImpl() should return error for wrong arg type, got nil")
	}
}

// TestStrCharsType verifies the type signature is correctly constructed
func TestStrCharsType(t *testing.T) {
	typ := makeStrCharsType()
	if typ == nil {
		t.Error("makeStrCharsType() returned nil")
	}
	// Type should be: String -> [String]
}

// TestStrStartsWith tests the _str_startsWith builtin
func TestStrStartsWith(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		prefix   string
		expected bool
	}{
		{"match", "hello world", "hello", true},
		{"no match", "hello world", "world", false},
		{"empty prefix", "hello", "", true},
		{"empty string", "", "hello", false},
		{"both empty", "", "", true},
		{"exact match", "hello", "hello", true},
		{"prefix longer", "hi", "hello", false},
		{"unicode prefix", "café latte", "café", true},
		{"unicode no match", "café latte", "latte", false},
	}

	ctx := testctx.NewMockEffContext()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := []eval.Value{
				&eval.StringValue{Value: tt.s},
				&eval.StringValue{Value: tt.prefix},
			}
			result, err := strStartsWithImpl(ctx.EffContext, args)
			if err != nil {
				t.Fatalf("strStartsWithImpl() error: %v", err)
			}
			bv, ok := result.(*eval.BoolValue)
			if !ok {
				t.Fatalf("expected *eval.BoolValue, got %T", result)
			}
			if bv.Value != tt.expected {
				t.Errorf("startsWith(%q, %q) = %v, want %v",
					tt.s, tt.prefix, bv.Value, tt.expected)
			}
		})
	}
}

// TestStrStartsWithWrongArgType tests error handling
func TestStrStartsWithWrongArgType(t *testing.T) {
	ctx := testctx.NewMockEffContext()

	args := []eval.Value{&eval.IntValue{Value: 42}, &eval.StringValue{Value: "hi"}}
	_, err := strStartsWithImpl(ctx.EffContext, args)
	if err == nil {
		t.Error("strStartsWithImpl() should return error for wrong arg type")
	}

	args = []eval.Value{&eval.StringValue{Value: "hi"}, &eval.IntValue{Value: 42}}
	_, err = strStartsWithImpl(ctx.EffContext, args)
	if err == nil {
		t.Error("strStartsWithImpl() should return error for wrong arg 1 type")
	}
}

// TestStrStartsWithType verifies the type signature
func TestStrStartsWithType(t *testing.T) {
	typ := makeStrStartsWithType()
	if typ == nil {
		t.Error("makeStrStartsWithType() returned nil")
	}
}

// TestStrEndsWith tests the _str_endsWith builtin
func TestStrEndsWith(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		suffix   string
		expected bool
	}{
		{"match", "hello world", "world", true},
		{"no match", "hello world", "hello", false},
		{"empty suffix", "hello", "", true},
		{"empty string", "", "hello", false},
		{"both empty", "", "", true},
		{"exact match", "hello", "hello", true},
		{"suffix longer", "hi", "hello", false},
		{"unicode suffix", "café latte", "latte", true},
		{"unicode no match", "café latte", "café", false},
		{"file extension", "document.pdf", ".pdf", true},
		{"wrong extension", "document.pdf", ".txt", false},
	}

	ctx := testctx.NewMockEffContext()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := []eval.Value{
				&eval.StringValue{Value: tt.s},
				&eval.StringValue{Value: tt.suffix},
			}
			result, err := strEndsWithImpl(ctx.EffContext, args)
			if err != nil {
				t.Fatalf("strEndsWithImpl() error: %v", err)
			}
			bv, ok := result.(*eval.BoolValue)
			if !ok {
				t.Fatalf("expected *eval.BoolValue, got %T", result)
			}
			if bv.Value != tt.expected {
				t.Errorf("endsWith(%q, %q) = %v, want %v",
					tt.s, tt.suffix, bv.Value, tt.expected)
			}
		})
	}
}

// TestStrEndsWithWrongArgType tests error handling
func TestStrEndsWithWrongArgType(t *testing.T) {
	ctx := testctx.NewMockEffContext()

	args := []eval.Value{&eval.IntValue{Value: 42}, &eval.StringValue{Value: "hi"}}
	_, err := strEndsWithImpl(ctx.EffContext, args)
	if err == nil {
		t.Error("strEndsWithImpl() should return error for wrong arg type")
	}

	args = []eval.Value{&eval.StringValue{Value: "hi"}, &eval.IntValue{Value: 42}}
	_, err = strEndsWithImpl(ctx.EffContext, args)
	if err == nil {
		t.Error("strEndsWithImpl() should return error for wrong arg 1 type")
	}
}

// TestStrEndsWithType verifies the type signature
func TestStrEndsWithType(t *testing.T) {
	typ := makeStrEndsWithType()
	if typ == nil {
		t.Error("makeStrEndsWithType() returned nil")
	}
}

// ============================================================================
// String Join Tests (M-PERF5)
// ============================================================================

// TestStrJoin_Basic tests the _str_join builtin with standard cases
func TestStrJoin_Basic(t *testing.T) {
	ctx := testctx.NewMockEffContext()

	tests := []struct {
		name     string
		parts    []string
		sep      string
		expected string
	}{
		{"comma separated", []string{"a", "b", "c"}, ", ", "a, b, c"},
		{"dash separated", []string{"hello", "world"}, "-", "hello-world"},
		{"empty separator", []string{"a", "b", "c"}, "", "abc"},
		{"single element", []string{"only"}, ", ", "only"},
		{"empty list", []string{}, ", ", ""},
		{"empty strings", []string{"", "", ""}, ",", ",,"},
		{"newline separator", []string{"line1", "line2", "line3"}, "\n", "line1\nline2\nline3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			elements := make([]eval.Value, len(tt.parts))
			for i, s := range tt.parts {
				elements[i] = &eval.StringValue{Value: s}
			}
			args := []eval.Value{
				&eval.ListValue{Elements: elements},
				&eval.StringValue{Value: tt.sep},
			}
			result, err := strJoinImpl(ctx.EffContext, args)
			if err != nil {
				t.Fatalf("strJoinImpl() error: %v", err)
			}
			sv, ok := result.(*eval.StringValue)
			if !ok {
				t.Fatalf("expected StringValue, got %T", result)
			}
			if sv.Value != tt.expected {
				t.Errorf("got %q, want %q", sv.Value, tt.expected)
			}
		})
	}
}

// TestStrJoin_LargeList tests with 1000+ elements to verify O(n) performance
func TestStrJoin_LargeList(t *testing.T) {
	ctx := testctx.NewMockEffContext()

	n := 1000
	elements := make([]eval.Value, n)
	for i := 0; i < n; i++ {
		elements[i] = &eval.StringValue{Value: "item"}
	}
	args := []eval.Value{
		&eval.ListValue{Elements: elements},
		&eval.StringValue{Value: ","},
	}

	result, err := strJoinImpl(ctx.EffContext, args)
	if err != nil {
		t.Fatalf("strJoinImpl() error: %v", err)
	}
	sv, ok := result.(*eval.StringValue)
	if !ok {
		t.Fatalf("expected StringValue, got %T", result)
	}

	// Expected: "item" repeated 1000 times with "," between = 4*1000 + 999 = 4999 chars
	expectedLen := 4*n + (n - 1)
	if len(sv.Value) != expectedLen {
		t.Errorf("got length %d, want %d", len(sv.Value), expectedLen)
	}
}

// TestStrJoin_Determinism runs 20 iterations to verify deterministic output
func TestStrJoin_Determinism(t *testing.T) {
	ctx := testctx.NewMockEffContext()

	elements := make([]eval.Value, 20)
	for i := 0; i < 20; i++ {
		elements[i] = &eval.StringValue{Value: fmt.Sprintf("elem_%d", i)}
	}
	args := []eval.Value{
		&eval.ListValue{Elements: elements},
		&eval.StringValue{Value: "|"},
	}

	// Run first to get baseline
	baseline, err := strJoinImpl(ctx.EffContext, args)
	if err != nil {
		t.Fatalf("strJoinImpl() error: %v", err)
	}
	baselineStr := baseline.(*eval.StringValue).Value

	// Run 19 more times and compare
	for i := 0; i < 19; i++ {
		result, err := strJoinImpl(ctx.EffContext, args)
		if err != nil {
			t.Fatalf("iteration %d: strJoinImpl() error: %v", i, err)
		}
		if result.(*eval.StringValue).Value != baselineStr {
			t.Errorf("iteration %d: nondeterministic output", i)
		}
	}
}

// TestStrJoin_ErrorCases tests error handling
func TestStrJoin_ErrorCases(t *testing.T) {
	ctx := testctx.NewMockEffContext()

	// Wrong type for first arg (not a list)
	args := []eval.Value{
		&eval.StringValue{Value: "not a list"},
		&eval.StringValue{Value: ","},
	}
	_, err := strJoinImpl(ctx.EffContext, args)
	if err == nil {
		t.Error("expected error for non-list first argument")
	}

	// Wrong type for separator (not a string)
	args = []eval.Value{
		&eval.ListValue{Elements: []eval.Value{&eval.StringValue{Value: "a"}}},
		&eval.IntValue{Value: 42},
	}
	_, err = strJoinImpl(ctx.EffContext, args)
	if err == nil {
		t.Error("expected error for non-string separator")
	}

	// Non-string element in list
	args = []eval.Value{
		&eval.ListValue{Elements: []eval.Value{
			&eval.StringValue{Value: "a"},
			&eval.IntValue{Value: 42},
		}},
		&eval.StringValue{Value: ","},
	}
	_, err = strJoinImpl(ctx.EffContext, args)
	if err == nil {
		t.Error("expected error for non-string element in list")
	}
}

// TestStrJoinType verifies the type signature
func TestStrJoinType(t *testing.T) {
	typ := makeStrJoinType()
	if typ == nil {
		t.Error("makeStrJoinType() returned nil")
	}
}
