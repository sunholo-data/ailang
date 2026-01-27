package builtins

import (
	"testing"

	"github.com/sunholo/ailang/internal/eval"
)

// TestSafeAsString tests the SafeAsString function
func TestSafeAsString(t *testing.T) {
	tests := []struct {
		name      string
		value     eval.Value
		wantVal   string
		wantError bool
	}{
		// Success cases
		{
			name:    "valid string",
			value:   &eval.StringValue{Value: "hello"},
			wantVal: "hello",
		},
		{
			name:    "empty string",
			value:   &eval.StringValue{Value: ""},
			wantVal: "",
		},
		{
			name:    "string with spaces",
			value:   &eval.StringValue{Value: "hello world"},
			wantVal: "hello world",
		},

		// Error cases
		{
			name:      "int value instead of string",
			value:     &eval.IntValue{Value: 42},
			wantError: true,
		},
		{
			name:      "float value instead of string",
			value:     &eval.FloatValue{Value: 3.14},
			wantError: true,
		},
		{
			name:      "bool value instead of string",
			value:     &eval.BoolValue{Value: true},
			wantError: true,
		},
		{
			name:      "list value instead of string",
			value:     &eval.ListValue{Elements: []eval.Value{}},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SafeAsString(tt.value)

			if tt.wantError {
				if err == nil {
					t.Errorf("SafeAsString() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("SafeAsString() unexpected error: %v", err)
				}
				if got != tt.wantVal {
					t.Errorf("SafeAsString() = %q, want %q", got, tt.wantVal)
				}
			}
		})
	}
}

// TestSafeAsInt tests the SafeAsInt function
func TestSafeAsInt(t *testing.T) {
	tests := []struct {
		name      string
		value     eval.Value
		wantVal   int
		wantError bool
	}{
		// Success cases
		{
			name:    "positive integer",
			value:   &eval.IntValue{Value: 42},
			wantVal: 42,
		},
		{
			name:    "negative integer",
			value:   &eval.IntValue{Value: -42},
			wantVal: -42,
		},
		{
			name:    "zero",
			value:   &eval.IntValue{Value: 0},
			wantVal: 0,
		},

		// Error cases
		{
			name:      "string value instead of int",
			value:     &eval.StringValue{Value: "42"},
			wantError: true,
		},
		{
			name:      "float value instead of int",
			value:     &eval.FloatValue{Value: 42.5},
			wantError: true,
		},
		{
			name:      "bool value instead of int",
			value:     &eval.BoolValue{Value: true},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SafeAsInt(tt.value)

			if tt.wantError {
				if err == nil {
					t.Errorf("SafeAsInt() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("SafeAsInt() unexpected error: %v", err)
				}
				if got != tt.wantVal {
					t.Errorf("SafeAsInt() = %d, want %d", got, tt.wantVal)
				}
			}
		})
	}
}

// TestSafeAsFloat tests the SafeAsFloat function
func TestSafeAsFloat(t *testing.T) {
	tests := []struct {
		name      string
		value     eval.Value
		wantVal   float64
		wantError bool
	}{
		// Success cases
		{
			name:    "positive float",
			value:   &eval.FloatValue{Value: 3.14},
			wantVal: 3.14,
		},
		{
			name:    "negative float",
			value:   &eval.FloatValue{Value: -3.14},
			wantVal: -3.14,
		},
		{
			name:    "zero",
			value:   &eval.FloatValue{Value: 0.0},
			wantVal: 0.0,
		},

		// Error cases
		{
			name:      "string value instead of float",
			value:     &eval.StringValue{Value: "3.14"},
			wantError: true,
		},
		{
			name:      "int value instead of float",
			value:     &eval.IntValue{Value: 3},
			wantError: true,
		},
		{
			name:      "bool value instead of float",
			value:     &eval.BoolValue{Value: true},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SafeAsFloat(tt.value)

			if tt.wantError {
				if err == nil {
					t.Errorf("SafeAsFloat() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("SafeAsFloat() unexpected error: %v", err)
				}
				if got != tt.wantVal {
					t.Errorf("SafeAsFloat() = %f, want %f", got, tt.wantVal)
				}
			}
		})
	}
}

// TestSafeAsBool tests the SafeAsBool function
func TestSafeAsBool(t *testing.T) {
	tests := []struct {
		name      string
		value     eval.Value
		wantVal   bool
		wantError bool
	}{
		// Success cases
		{
			name:    "true",
			value:   &eval.BoolValue{Value: true},
			wantVal: true,
		},
		{
			name:    "false",
			value:   &eval.BoolValue{Value: false},
			wantVal: false,
		},

		// Error cases
		{
			name:      "string value instead of bool",
			value:     &eval.StringValue{Value: "true"},
			wantError: true,
		},
		{
			name:      "int value instead of bool",
			value:     &eval.IntValue{Value: 1},
			wantError: true,
		},
		{
			name:      "float value instead of bool",
			value:     &eval.FloatValue{Value: 1.0},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SafeAsBool(tt.value)

			if tt.wantError {
				if err == nil {
					t.Errorf("SafeAsBool() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("SafeAsBool() unexpected error: %v", err)
				}
				if got != tt.wantVal {
					t.Errorf("SafeAsBool() = %v, want %v", got, tt.wantVal)
				}
			}
		})
	}
}

// TestSafeAsList tests the SafeAsList function
func TestSafeAsList(t *testing.T) {
	emptyList := []eval.Value{}
	simpleList := []eval.Value{
		&eval.IntValue{Value: 1},
		&eval.IntValue{Value: 2},
	}

	tests := []struct {
		name      string
		value     eval.Value
		wantLen   int
		wantError bool
	}{
		{
			name:    "empty list",
			value:   &eval.ListValue{Elements: emptyList},
			wantLen: 0,
		},
		{
			name:    "list with elements",
			value:   &eval.ListValue{Elements: simpleList},
			wantLen: 2,
		},
		{
			name:      "string value instead of list",
			value:     &eval.StringValue{Value: "[]"},
			wantError: true,
		},
		{
			name:      "int value instead of list",
			value:     &eval.IntValue{Value: 42},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SafeAsList(tt.value)

			if tt.wantError {
				if err == nil {
					t.Errorf("SafeAsList() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("SafeAsList() unexpected error: %v", err)
				}
				if len(got) != tt.wantLen {
					t.Errorf("SafeAsList() len = %d, want %d", len(got), tt.wantLen)
				}
			}
		})
	}
}
