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

// TestNormalizeProgram_MToolingMotivatingFragment is the regression guard for the
// SUPERSEDED design doc M-TOOLING-DETERMINISTIC (planned/v0_29_0). That doc proposed a
// standalone `ailang normalize` / `suggest-imports` / `apply` CLI trio to deterministically
// repair AI fragments without LLM inference. Reality-check (mission iteration, 2026-07-18):
// the core deterministic-normalize capability the doc wanted ALREADY EXISTS here as
// normalizeProgram — this test pins that it covers the doc's exact json_parse motivating
// fragment, and documents the one boundary (general symbol→import resolution, e.g. std/json)
// that normalizeProgram does NOT do: that need is now met by agentic-mode `ailang check`
// feedback + implicit prelude imports + `ailang docs` discovery, not a suggest-imports command.
// If this capability regresses, the supersession rationale no longer holds — reopen the doc.
func TestNormalizeProgram_MToolingMotivatingFragment(t *testing.T) {
	// The doc's exact "Example Failure (from json_parse benchmark)" fragment:
	// a bare func main with no module, no imports, no effect annotation.
	code := `func main() {
  let data = decode("[{\"name\":\"Alice\"}]")
  println(show(data))
}`
	caps := []string{"IO"}

	normalized, log := normalizeProgram(code, caps)

	// Deterministic: same input → identical output (doc Goal 4 "byte-stable").
	normalized2, _ := normalizeProgram(code, caps)
	if normalized != normalized2 {
		t.Errorf("normalizeProgram is non-deterministic:\nfirst:\n%s\nsecond:\n%s", normalized, normalized2)
	}

	// Goal 1 (normalize/wrap fragment): module scaffold added.
	if !log.Wrapped {
		t.Error("Expected Wrapped=true for a module-less fragment")
	}
	if !log.AddedModule {
		t.Error("Expected AddedModule=true (fragment had no module declaration)")
	}
	if !strings.Contains(normalized, "module benchmark/solution") {
		t.Errorf("Missing synthesized module declaration:\n%s", normalized)
	}

	// Goal 2 (imports), std/io portion: println → std/io injected automatically.
	if !strings.Contains(normalized, "import std/io") {
		t.Errorf("Missing auto-injected std/io import:\n%s", normalized)
	}

	// Original body preserved verbatim (decode + println(show ...)).
	if !strings.Contains(normalized, "decode(") || !strings.Contains(normalized, "println(show(data))") {
		t.Errorf("Fragment body not preserved:\n%s", normalized)
	}

	// BOUNDARY (documented supersession): normalizeProgram resolves std/io ONLY, not
	// arbitrary symbols — `decode` is from std/json and is intentionally NOT auto-imported
	// here. The doc's general `suggest-imports` was never built; the need is now absorbed by
	// the agentic compiler-feedback loop. Pin the boundary so a future change that DOES add
	// general import resolution forces a conscious update of the supersession record.
	if strings.Contains(normalized, "import std/json") {
		t.Errorf("Unexpected std/json auto-import — general symbol resolution is NOT part of "+
			"normalizeProgram's contract; update the M-TOOLING-DETERMINISTIC supersession record:\n%s", normalized)
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
