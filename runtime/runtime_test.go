package runtime

import (
	"testing"
)

func TestShow(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected string
	}{
		{"string", "hello", "hello"},
		{"int64", int64(42), "42"},
		{"int", 42, "42"},
		{"float64", 3.14, "3.14"},
		{"bool true", true, "true"},
		{"bool false", false, "false"},
		{"nil", nil, "()"},
		{"slice", []int{1, 2, 3}, "[1 2 3]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Show(tt.input)
			if result != tt.expected {
				t.Errorf("Show(%v) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestConcatString(t *testing.T) {
	tests := []struct {
		a, b     any
		expected string
	}{
		{"hello", " world", "hello world"},
		{"count: ", 42, "count: 42"},
		{3.14, " is pi", "3.14 is pi"},
		{true, " story", "true story"},
	}

	for _, tt := range tests {
		result := ConcatString(tt.a, tt.b)
		if result != tt.expected {
			t.Errorf("ConcatString(%v, %v) = %q, want %q", tt.a, tt.b, result, tt.expected)
		}
	}
}

func TestConvertToInt64Slice(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected []int64
	}{
		{"nil", nil, nil},
		{"empty", []any{}, []int64{}},
		{"int64s", []any{int64(1), int64(2), int64(3)}, []int64{1, 2, 3}},
		{"ints", []any{1, 2, 3}, []int64{1, 2, 3}},
		{"interface slice", []interface{}{int64(1), int64(2)}, []int64{1, 2}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertToInt64Slice(tt.input)
			if tt.expected == nil {
				if result != nil {
					t.Errorf("ConvertToInt64Slice(%v) = %v, want nil", tt.input, result)
				}
				return
			}
			if len(result) != len(tt.expected) {
				t.Errorf("ConvertToInt64Slice(%v) length = %d, want %d", tt.input, len(result), len(tt.expected))
				return
			}
			for i, v := range result {
				if v != tt.expected[i] {
					t.Errorf("ConvertToInt64Slice(%v)[%d] = %d, want %d", tt.input, i, v, tt.expected[i])
				}
			}
		})
	}
}

func TestConvertToStringSlice(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected []string
	}{
		{"nil", nil, nil},
		{"empty", []any{}, []string{}},
		{"strings", []any{"a", "b", "c"}, []string{"a", "b", "c"}},
		{"interface slice", []interface{}{"x", "y"}, []string{"x", "y"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertToStringSlice(tt.input)
			if tt.expected == nil {
				if result != nil {
					t.Errorf("ConvertToStringSlice(%v) = %v, want nil", tt.input, result)
				}
				return
			}
			if len(result) != len(tt.expected) {
				t.Errorf("ConvertToStringSlice(%v) length = %d, want %d", tt.input, len(result), len(tt.expected))
				return
			}
			for i, v := range result {
				if v != tt.expected[i] {
					t.Errorf("ConvertToStringSlice(%v)[%d] = %q, want %q", tt.input, i, v, tt.expected[i])
				}
			}
		})
	}
}

func TestConvertToRecordSlice(t *testing.T) {
	input := []any{
		map[string]any{"x": int64(1), "y": int64(2)},
		map[string]any{"x": int64(3), "y": int64(4)},
	}

	result := ConvertToRecordSlice(input)
	if len(result) != 2 {
		t.Fatalf("ConvertToRecordSlice length = %d, want 2", len(result))
	}

	if result[0]["x"] != int64(1) || result[0]["y"] != int64(2) {
		t.Errorf("ConvertToRecordSlice[0] = %v, want {x:1, y:2}", result[0])
	}
	if result[1]["x"] != int64(3) || result[1]["y"] != int64(4) {
		t.Errorf("ConvertToRecordSlice[1] = %v, want {x:3, y:4}", result[1])
	}
}

func TestConvertToRecordSlice_InterfaceInput(t *testing.T) {
	input := []interface{}{
		map[string]interface{}{"a": "hello"},
		map[string]interface{}{"b": "world"},
	}

	result := ConvertToRecordSlice(input)
	if len(result) != 2 {
		t.Fatalf("ConvertToRecordSlice length = %d, want 2", len(result))
	}

	if result[0]["a"] != "hello" {
		t.Errorf("ConvertToRecordSlice[0][a] = %v, want hello", result[0]["a"])
	}
}
