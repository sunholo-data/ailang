package main

import (
	"testing"
)

func TestParseMemorySize(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
		wantErr  bool
	}{
		{"256MB", 256 * 1024 * 1024, false},
		{"512mb", 512 * 1024 * 1024, false},
		{"1GB", 1024 * 1024 * 1024, false},
		{"2gb", 2 * 1024 * 1024 * 1024, false},
		{"64KB", 64 * 1024, false},
		{"1073741824", 1073741824, false}, // 1GB in bytes
		{"1.5GB", int64(1.5 * 1024 * 1024 * 1024), false},

		// Error cases
		{"", 0, true},
		{"abc", 0, true},
		{"MB", 0, true},
		{"-1GB", -1073741824, false}, // negative is valid parse; applyMemoryLimit rejects it
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseMemorySize(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error for input %q, got %d", tt.input, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for input %q: %v", tt.input, err)
			}
			if got != tt.expected {
				t.Errorf("parseMemorySize(%q) = %d, want %d", tt.input, got, tt.expected)
			}
		})
	}
}
