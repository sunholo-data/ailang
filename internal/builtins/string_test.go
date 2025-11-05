package builtins

import (
	"testing"

	"github.com/sunholo/ailang/internal/effects/testctx"
	"github.com/sunholo/ailang/internal/eval"
)

// TestStringToInt tests the _stringToInt builtin
func TestStringToInt(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantSome bool
		wantVal  int
	}{
		// Valid integers
		{"positive integer", "42", true, 42},
		{"negative integer", "-123", true, -123},
		{"zero", "0", true, 0},
		{"large positive", "2147483647", true, 2147483647},
		{"large negative", "-2147483648", true, -2147483648},
		{"positive with plus sign", "+456", true, 456},

		// Invalid inputs - should return None
		{"empty string", "", false, 0},
		{"letters only", "abc", false, 0},
		{"float notation", "3.14", false, 0},
		{"mixed alphanumeric", "123abc", false, 0},
		{"whitespace", "  ", false, 0},
		{"whitespace with number", " 42 ", false, 0}, // strconv.ParseInt doesn't trim
		{"multiple signs", "++42", false, 0},
		{"scientific notation", "1e10", false, 0},
	}

	ctx := testctx.NewMockEffContext()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Call the builtin
			args := []eval.Value{&eval.StringValue{Value: tt.input}}
			result, err := stringToIntImpl(ctx.EffContext, args)

			if err != nil {
				t.Fatalf("stringToIntImpl() returned error: %v", err)
			}

			// Check that result is a TaggedValue (Option)
			tagged, ok := result.(*eval.TaggedValue)
			if !ok {
				t.Fatalf("stringToIntImpl() returned %T, expected *eval.TaggedValue", result)
			}

			if tagged.TypeName != "Option" {
				t.Errorf("TypeName = %q, want %q", tagged.TypeName, "Option")
			}

			if tt.wantSome {
				// Should return Some(n)
				if tagged.CtorName != "Some" {
					t.Errorf("CtorName = %q, want %q", tagged.CtorName, "Some")
				}
				if len(tagged.Fields) != 1 {
					t.Fatalf("len(Fields) = %d, want 1", len(tagged.Fields))
				}

				intVal, ok := tagged.Fields[0].(*eval.IntValue)
				if !ok {
					t.Fatalf("Field type = %T, expected *eval.IntValue", tagged.Fields[0])
				}

				if intVal.Value != tt.wantVal {
					t.Errorf("IntValue = %d, want %d", intVal.Value, tt.wantVal)
				}
			} else {
				// Should return None
				if tagged.CtorName != "None" {
					t.Errorf("CtorName = %q, want %q for invalid input", tagged.CtorName, "None")
				}
				if len(tagged.Fields) != 0 {
					t.Errorf("None should have 0 fields, got %d", len(tagged.Fields))
				}
			}
		})
	}
}

// TestStringToFloat tests the _stringToFloat builtin
func TestStringToFloat(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantSome bool
		wantVal  float64
	}{
		// Valid floats
		{"positive float", "3.14", true, 3.14},
		{"negative float", "-2.5", true, -2.5},
		{"zero", "0.0", true, 0.0},
		{"integer notation", "42", true, 42.0},
		{"scientific notation positive exp", "1e10", true, 1e10},
		{"scientific notation negative exp", "1e-10", true, 1e-10},
		{"scientific notation with decimal", "3.14e2", true, 314.0},
		{"large float", "1.7976931348623157e+308", true, 1.7976931348623157e+308},
		{"small positive", "2.2250738585072014e-308", true, 2.2250738585072014e-308},
		{"positive with plus sign", "+3.14", true, 3.14},

		// Invalid inputs - should return None
		{"empty string", "", false, 0},
		{"letters only", "abc", false, 0},
		{"mixed alphanumeric", "3.14abc", false, 0},
		{"whitespace", "  ", false, 0},
		{"whitespace with number", " 3.14 ", false, 0}, // strconv.ParseFloat doesn't trim
		{"multiple dots", "3.14.15", false, 0},
		{"invalid scientific", "1e", false, 0},
		{"text mixed with number", "abc123.45", false, 0},
	}

	ctx := testctx.NewMockEffContext()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Call the builtin
			args := []eval.Value{&eval.StringValue{Value: tt.input}}
			result, err := stringToFloatImpl(ctx.EffContext, args)

			if err != nil {
				t.Fatalf("stringToFloatImpl() returned error: %v", err)
			}

			// Check that result is a TaggedValue (Option)
			tagged, ok := result.(*eval.TaggedValue)
			if !ok {
				t.Fatalf("stringToFloatImpl() returned %T, expected *eval.TaggedValue", result)
			}

			if tagged.TypeName != "Option" {
				t.Errorf("TypeName = %q, want %q", tagged.TypeName, "Option")
			}

			if tt.wantSome {
				// Should return Some(f)
				if tagged.CtorName != "Some" {
					t.Errorf("CtorName = %q, want %q", tagged.CtorName, "Some")
				}
				if len(tagged.Fields) != 1 {
					t.Fatalf("len(Fields) = %d, want 1", len(tagged.Fields))
				}

				floatVal, ok := tagged.Fields[0].(*eval.FloatValue)
				if !ok {
					t.Fatalf("Field type = %T, expected *eval.FloatValue", tagged.Fields[0])
				}

				if floatVal.Value != tt.wantVal {
					t.Errorf("FloatValue = %f, want %f", floatVal.Value, tt.wantVal)
				}
			} else {
				// Should return None
				if tagged.CtorName != "None" {
					t.Errorf("CtorName = %q, want %q for invalid input", tagged.CtorName, "None")
				}
				if len(tagged.Fields) != 0 {
					t.Errorf("None should have 0 fields, got %d", len(tagged.Fields))
				}
			}
		})
	}
}

// TestStringToIntTypeCheck tests that the type signature is correct
func TestStringToIntTypeCheck(t *testing.T) {
	typ := makeStringToIntType()
	if typ == nil {
		t.Fatal("makeStringToIntType() returned nil")
	}
	// Type should be: String -> Option[Int]
	// We can't easily inspect the type structure here, but we verify it builds
}

// TestStringToFloatTypeCheck tests that the type signature is correct
func TestStringToFloatTypeCheck(t *testing.T) {
	typ := makeStringToFloatType()
	if typ == nil {
		t.Fatal("makeStringToFloatType() returned nil")
	}
	// Type should be: String -> Option[Float]
	// We can't easily inspect the type structure here, but we verify it builds
}

// TestStringToIntWrongArgType tests error handling for wrong argument type
func TestStringToIntWrongArgType(t *testing.T) {
	ctx := testctx.NewMockEffContext()

	// Pass an IntValue instead of StringValue
	args := []eval.Value{&eval.IntValue{Value: 42}}
	_, err := stringToIntImpl(ctx.EffContext, args)

	if err == nil {
		t.Error("stringToIntImpl() should return error for wrong arg type, got nil")
	}
}

// TestStringToFloatWrongArgType tests error handling for wrong argument type
func TestStringToFloatWrongArgType(t *testing.T) {
	ctx := testctx.NewMockEffContext()

	// Pass an IntValue instead of StringValue
	args := []eval.Value{&eval.IntValue{Value: 42}}
	_, err := stringToFloatImpl(ctx.EffContext, args)

	if err == nil {
		t.Error("stringToFloatImpl() should return error for wrong arg type, got nil")
	}
}
