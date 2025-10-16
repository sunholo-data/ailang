package eval_harness

import (
	"strings"
	"testing"
)

func TestNormalizeProgram_BareExpression(t *testing.T) {
	code := "print 5 % 3"
	caps := []string{"IO"}

	normalized, log := normalizeProgram(code, caps)

	// Check logs
	if !log.Wrapped {
		t.Error("Expected Wrapped=true")
	}
	if !log.AddedModule {
		t.Error("Expected AddedModule=true")
	}
	if log.CallFixes != 1 {
		t.Errorf("Expected CallFixes=1, got %d", log.CallFixes)
	}

	// Check normalized code
	if !strings.Contains(normalized, "module benchmark/solution") {
		t.Error("Missing module declaration")
	}
	if !strings.Contains(normalized, "import std/io") {
		t.Error("Missing std/io import")
	}
	if !strings.Contains(normalized, "print(5 % 3)") {
		t.Errorf("Expected 'print(5 %% 3)', got:\n%s", normalized)
	}
	if !strings.Contains(normalized, "export func main()") {
		t.Error("Missing main function")
	}

	t.Logf("Normalized code:\n%s", normalized)
}

func TestNormalizeProgram_BareExpressionNoFunc(t *testing.T) {
	code := "5 % 3"
	caps := []string{"IO"}

	normalized, log := normalizeProgram(code, caps)

	if !log.Wrapped {
		t.Error("Expected Wrapped=true")
	}

	// Should wrap in println(show(...))
	if !strings.Contains(normalized, "println(show(") {
		t.Errorf("Expected println(show(...)), got:\n%s", normalized)
	}

	t.Logf("Normalized code:\n%s", normalized)
}

func TestNormalizeProgram_CompleteModule(t *testing.T) {
	code := `module benchmark/solution

import std/io

export func main() -> () ! {IO} {
  println("Hello")
}`
	caps := []string{"IO"}

	normalized, log := normalizeProgram(code, caps)

	// Should not wrap (already complete)
	if log.Wrapped {
		t.Error("Should not wrap complete module")
	}

	// Should be mostly unchanged
	if normalized != code {
		t.Logf("Original:\n%s", code)
		t.Logf("Normalized:\n%s", normalized)
	}
}

func TestFixBarePrintCalls(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		count    int
	}{
		{
			name:     "bare print",
			input:    "print 5 % 3",
			expected: "print(5 % 3)",
			count:    1,
		},
		{
			name:     "bare println",
			input:    "println x + y",
			expected: "println(x + y)",
			count:    1,
		},
		{
			name:     "already has parens",
			input:    "print(5 % 3)",
			expected: "print(5 % 3)",
			count:    0,
		},
		{
			name:     "multiline",
			input:    "print 1\nprintln 2",
			expected: "print(1)\nprintln(2)",
			count:    2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, count := fixBarePrintCalls(tt.input)
			if result != tt.expected {
				t.Errorf("Expected:\n%s\nGot:\n%s", tt.expected, result)
			}
			if count != tt.count {
				t.Errorf("Expected count=%d, got %d", tt.count, count)
			}
		})
	}
}
