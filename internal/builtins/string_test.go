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

// TestStrSplit tests the _str_split builtin comprehensively
func TestStrSplit(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		delimiter string
		expected  []string
	}{
		{"basic comma", "a,b,c", ",", []string{"a", "b", "c"}},
		{"no delimiter found", "hello", ",", []string{"hello"}},
		{"empty fields", "a,,c", ",", []string{"a", "", "c"}},
		{"leading delimiter", ",b,c", ",", []string{"", "b", "c"}},
		{"trailing delimiter", "a,b,", ",", []string{"a", "b", ""}},
		{"empty string with delimiter", "", ",", []string{""}},
		{"empty string empty delimiter", "", "", []string{}}, // Special case!
		{"multi-char delimiter", "a::b::c", "::", []string{"a", "b", "c"}},
		{"empty delimiter", "abc", "", []string{"a", "b", "c"}},
		{"newlines", "line1\nline2\nline3", "\n", []string{"line1", "line2", "line3"}},
		{"tabs", "col1\tcol2\tcol3", "\t", []string{"col1", "col2", "col3"}},
		{"unicode", "café", "", []string{"c", "a", "f", "é"}},
		{"emoji", "hi🎉bye", "", []string{"h", "i", "🎉", "b", "y", "e"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := testctx.NewMockEffContext()
			args := []eval.Value{
				&eval.StringValue{Value: tt.input},
				&eval.StringValue{Value: tt.delimiter},
			}

			result, err := strSplitImpl(ctx.EffContext, args)
			if err != nil {
				t.Fatalf("strSplitImpl() failed: %v", err)
			}

			// Result should be a ListValue
			listVal, ok := result.(*eval.ListValue)
			if !ok {
				t.Fatalf("expected *eval.ListValue, got %T", result)
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
				t.Errorf("split(%q, %q) length = %d, want %d\ngot: %v\nwant: %v",
					tt.input, tt.delimiter, len(got), len(tt.expected), got, tt.expected)
				return
			}

			for i := range got {
				if got[i] != tt.expected[i] {
					t.Errorf("split(%q, %q)[%d] = %q, want %q\ngot: %v\nwant: %v",
						tt.input, tt.delimiter, i, got[i], tt.expected[i], got, tt.expected)
				}
			}
		})
	}
}

// TestStrSplitWrongArgType tests error handling for wrong argument types
func TestStrSplitWrongArgType(t *testing.T) {
	ctx := testctx.NewMockEffContext()

	// Test first arg wrong type
	args := []eval.Value{
		&eval.IntValue{Value: 42},
		&eval.StringValue{Value: ","},
	}
	_, err := strSplitImpl(ctx.EffContext, args)
	if err == nil {
		t.Error("strSplitImpl() should return error for wrong first arg type, got nil")
	}

	// Test second arg wrong type
	args = []eval.Value{
		&eval.StringValue{Value: "hello"},
		&eval.IntValue{Value: 42},
	}
	_, err = strSplitImpl(ctx.EffContext, args)
	if err == nil {
		t.Error("strSplitImpl() should return error for wrong second arg type, got nil")
	}
}

// TestStrSplitType verifies the type signature is correctly constructed
func TestStrSplitType(t *testing.T) {
	typ := makeStrSplitType()
	if typ == nil {
		t.Error("makeStrSplitType() returned nil")
	}
	// Type should be: String -> String -> [String]
	// We can't easily inspect the curried function structure here,
	// but we verify it builds without panicking
}

// TestStringReverse tests the _string_reverse builtin
func TestStringReverse(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// Basic cases
		{"empty string", "", ""},
		{"single character", "a", "a"},
		{"two characters", "ab", "ba"},

		// ASCII strings
		{"hello world", "hello", "olleh"},
		{"digits", "12345", "54321"},
		{"special characters", "!@#$%", "%$#@!"},
		{"mixed alphanumeric", "abc123", "321cba"},

		// Unicode - emoji
		{"single emoji", "🎉", "🎉"},
		{"two emoji", "🎉🎊", "🎊🎉"},
		{"emoji and text", "a🎉b", "b🎉a"},
		{"multiple emoji", "🎈🎊🎉", "🎉🎊🎈"},

		// Unicode - accented characters
		{"accented cafe", "café", "éfac"},
		{"accented greek", "αβγ", "γβα"},
		{"mixed accents", "é à ñ", "ñ à é"},

		// Complex Unicode
		{"emoji with spaces", "🎉 🎊", "🎊 🎉"},
		{"mixed multilingual", "hello🎉世界", "界世🎉olleh"},
		{"newline character", "hello\nworld", "dlrow\nolleh"},
		{"tab character", "hello\tworld", "dlrow\tolleh"},

		// Edge cases
		{"single space", " ", " "},
		{"multiple spaces", "   ", "   "},
		{"whitespace only", "\t\n ", " \n\t"},
		{"long string", "abcdefghijklmnopqrstuvwxyz", "zyxwvutsrqponmlkjihgfedcba"},
	}

	ctx := testctx.NewMockEffContext()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Call the builtin
			args := []eval.Value{&eval.StringValue{Value: tt.input}}
			result, err := stringReverseImpl(ctx.EffContext, args)

			if err != nil {
				t.Fatalf("stringReverseImpl() returned error: %v", err)
			}

			// Check that result is a StringValue
			strVal, ok := result.(*eval.StringValue)
			if !ok {
				t.Fatalf("stringReverseImpl() returned %T, expected *eval.StringValue", result)
			}

			if strVal.Value != tt.expected {
				t.Errorf("reverse(%q) = %q, want %q", tt.input, strVal.Value, tt.expected)
			}
		})
	}
}

// TestStringReverseWrongArgType tests error handling for wrong argument types
func TestStringReverseWrongArgType(t *testing.T) {
	ctx := testctx.NewMockEffContext()

	// Test with integer instead of string
	args := []eval.Value{&eval.IntValue{Value: 42}}
	_, err := stringReverseImpl(ctx.EffContext, args)
	if err == nil {
		t.Error("stringReverseImpl() should return error for wrong arg type, got nil")
	}

	// Test with bool
	args = []eval.Value{&eval.BoolValue{Value: true}}
	_, err = stringReverseImpl(ctx.EffContext, args)
	if err == nil {
		t.Error("stringReverseImpl() should return error for bool arg type, got nil")
	}
}

// TestStringReverseType verifies the type signature is correctly constructed
func TestStringReverseType(t *testing.T) {
	typ := makeStringReverseType()
	if typ == nil {
		t.Error("makeStringReverseType() returned nil")
	}
	// Type should be: String -> String
	// We can't easily inspect the function structure here,
	// but we verify it builds without panicking
}

// TestStringReverseIdempotent tests that reversing twice returns original
func TestStringReverseIdempotent(t *testing.T) {
	tests := []string{
		"hello",
		"🎉🎊",
		"café",
		"123",
		"",
		"a",
	}

	ctx := testctx.NewMockEffContext()

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			// First reverse
			args1 := []eval.Value{&eval.StringValue{Value: input}}
			result1, err := stringReverseImpl(ctx.EffContext, args1)
			if err != nil {
				t.Fatalf("first reverse failed: %v", err)
			}

			reversed, ok := result1.(*eval.StringValue)
			if !ok {
				t.Fatalf("expected *eval.StringValue, got %T", result1)
			}

			// Second reverse
			args2 := []eval.Value{reversed}
			result2, err := stringReverseImpl(ctx.EffContext, args2)
			if err != nil {
				t.Fatalf("second reverse failed: %v", err)
			}

			doubleReversed, ok := result2.(*eval.StringValue)
			if !ok {
				t.Fatalf("expected *eval.StringValue, got %T", result2)
			}

			if doubleReversed.Value != input {
				t.Errorf("reverse(reverse(%q)) = %q, want %q",
					input, doubleReversed.Value, input)
			}
		})
	}
}

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
