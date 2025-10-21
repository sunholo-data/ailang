package eval

import (
	"strings"
	"testing"
)

func TestArgTypeMismatch_Basic(t *testing.T) {
	err := ArgTypeMismatch("_str_len", "string", "int")

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	msg := err.Error()
	if !strings.Contains(msg, "_str_len") {
		t.Errorf("Error message should contain builtin name: %s", msg)
	}
	if !strings.Contains(msg, "expected string") {
		t.Errorf("Error message should contain expected type: %s", msg)
	}
	if !strings.Contains(msg, "got int") {
		t.Errorf("Error message should contain actual type: %s", msg)
	}
}

func TestArgTypeMismatch_ConcatConfusion(t *testing.T) {
	tests := []struct {
		name     string
		builtin  string
		expected string
		got      string
		wantHint string
	}{
		{
			name:     "list concat got string",
			builtin:  "_list_concat",
			expected: "list",
			got:      "string",
			wantHint: "Use ++ for list concatenation",
		},
		{
			name:     "string concat got list",
			builtin:  "_str_concat",
			expected: "string",
			got:      "list",
			wantHint: "Use ++ for string concatenation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ArgTypeMismatch(tt.builtin, tt.expected, tt.got)
			msg := err.Error()

			if !strings.Contains(msg, tt.wantHint) {
				t.Errorf("Expected hint %q in error message:\n%s", tt.wantHint, msg)
			}
		})
	}
}

func TestArgTypeMismatch_StringOperations(t *testing.T) {
	tests := []struct {
		builtin  string
		got      string
		wantHint string
	}{
		{"_str_len", "int", "Use _str_len only on strings"},
		{"_str_slice", "list", "Use _str_slice only on strings"},
	}

	for _, tt := range tests {
		t.Run(tt.builtin, func(t *testing.T) {
			err := ArgTypeMismatch(tt.builtin, "string", tt.got)
			msg := err.Error()

			if !strings.Contains(msg, tt.wantHint) {
				t.Errorf("Expected hint containing %q in error message:\n%s", tt.wantHint, msg)
			}
		})
	}
}

func TestArgTypeMismatch_ListOperations(t *testing.T) {
	tests := []struct {
		builtin  string
		wantHint string
	}{
		{"_list_length", "Use _list_length only on lists"},
		{"_list_head", "Use _list_head only on lists"},
		{"_list_tail", "Use _list_tail only on lists"},
	}

	for _, tt := range tests {
		t.Run(tt.builtin, func(t *testing.T) {
			err := ArgTypeMismatch(tt.builtin, "list", "string")
			msg := err.Error()

			if !strings.Contains(msg, tt.wantHint) {
				t.Errorf("Expected hint containing %q in error message:\n%s", tt.wantHint, msg)
			}
		})
	}
}

func TestArgTypeMismatch_MathOperations(t *testing.T) {
	builtins := []string{"_add", "_sub", "_mul", "_div"}

	for _, builtin := range builtins {
		t.Run(builtin+"_string", func(t *testing.T) {
			err := ArgTypeMismatch(builtin, "number", "string")
			msg := err.Error()

			if !strings.Contains(msg, "arithmetic on strings") {
				t.Errorf("Expected hint about arithmetic on strings in:\n%s", msg)
			}
			if !strings.Contains(msg, "Use ++ for string concatenation") {
				t.Errorf("Expected concat hint in:\n%s", msg)
			}
		})

		t.Run(builtin+"_list", func(t *testing.T) {
			err := ArgTypeMismatch(builtin, "number", "list")
			msg := err.Error()

			if !strings.Contains(msg, "arithmetic on lists") {
				t.Errorf("Expected hint about arithmetic on lists in:\n%s", msg)
			}
		})
	}
}

func TestArgTypeMismatch_Comparisons(t *testing.T) {
	builtins := []string{"_lt", "_le", "_gt", "_ge"}

	for _, builtin := range builtins {
		t.Run(builtin, func(t *testing.T) {
			err := ArgTypeMismatch(builtin, "number", "string")
			msg := err.Error()

			if !strings.Contains(msg, "_str_compare") {
				t.Errorf("Expected hint about _str_compare in:\n%s", msg)
			}
		})
	}
}

func TestArgTypeMismatch_GenericHints(t *testing.T) {
	tests := []struct {
		name     string
		expected string
		got      string
		wantHint string
	}{
		{
			name:     "int to string",
			expected: "string",
			got:      "int",
			wantHint: "convert the integer to a string",
		},
		{
			name:     "string to int",
			expected: "int",
			got:      "string",
			wantHint: "parse the string as an integer",
		},
		{
			name:     "string to list",
			expected: "list",
			got:      "string",
			wantHint: "split the string into a list",
		},
		{
			name:     "list to string",
			expected: "string",
			got:      "list",
			wantHint: "join the list into a string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ArgTypeMismatch("_some_builtin", tt.expected, tt.got)
			msg := err.Error()

			if !strings.Contains(msg, tt.wantHint) {
				t.Errorf("Expected hint containing %q in error message:\n%s", tt.wantHint, msg)
			}
		})
	}
}

func TestIndexOutOfBounds_Negative(t *testing.T) {
	err := IndexOutOfBounds("_list_at", -1, 10)
	msg := err.Error()

	if !strings.Contains(msg, "Negative indices") {
		t.Errorf("Expected hint about negative indices in:\n%s", msg)
	}
}

func TestIndexOutOfBounds_TooLarge(t *testing.T) {
	err := IndexOutOfBounds("_list_at", 10, 5)
	msg := err.Error()

	if !strings.Contains(msg, "index 10") {
		t.Errorf("Expected index value in error message:\n%s", msg)
	}
	if !strings.Contains(msg, "0 to 4") {
		t.Errorf("Expected valid range hint in error message:\n%s", msg)
	}
}

func TestInvalidOperation(t *testing.T) {
	err := InvalidOperation("_some_op", "Division by zero is not allowed")
	msg := err.Error()

	if !strings.Contains(msg, "_some_op") {
		t.Errorf("Expected builtin name in error message:\n%s", msg)
	}
	if !strings.Contains(msg, "Division by zero") {
		t.Errorf("Expected reason in error message:\n%s", msg)
	}
}

func TestEmptyListError_Head(t *testing.T) {
	err := EmptyListError("_list_head")
	msg := err.Error()

	if !strings.Contains(msg, "empty list") {
		t.Errorf("Expected 'empty list' in error message:\n%s", msg)
	}
	if !strings.Contains(msg, "Cannot get the head") {
		t.Errorf("Expected hint about head in error message:\n%s", msg)
	}
}

func TestEmptyListError_Tail(t *testing.T) {
	err := EmptyListError("_list_tail")
	msg := err.Error()

	if !strings.Contains(msg, "empty list") {
		t.Errorf("Expected 'empty list' in error message:\n%s", msg)
	}
	if !strings.Contains(msg, "Cannot get the tail") {
		t.Errorf("Expected hint about tail in error message:\n%s", msg)
	}
}

func TestBuiltinError_String(t *testing.T) {
	err := &BuiltinError{
		Builtin:  "_test_func",
		Expected: "string",
		Got:      "int",
		Hint:     "This is a helpful hint",
	}

	msg := err.Error()

	// Check all components are present
	if !strings.Contains(msg, "_test_func") {
		t.Errorf("Missing builtin name in: %s", msg)
	}
	if !strings.Contains(msg, "expected string") {
		t.Errorf("Missing expected type in: %s", msg)
	}
	if !strings.Contains(msg, "got int") {
		t.Errorf("Missing actual type in: %s", msg)
	}
	if !strings.Contains(msg, "Hint: This is a helpful hint") {
		t.Errorf("Missing hint in: %s", msg)
	}
}

func TestBuiltinError_NoHint(t *testing.T) {
	err := &BuiltinError{
		Builtin:  "_test_func",
		Expected: "string",
		Got:      "int",
		Hint:     "",
	}

	msg := err.Error()

	// Should not contain "Hint:" when hint is empty
	if strings.Contains(msg, "Hint:") {
		t.Errorf("Should not contain 'Hint:' when hint is empty: %s", msg)
	}
}
