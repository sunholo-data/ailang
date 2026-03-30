package pipeline

import (
	"strings"
	"testing"
)

// TestConsExpression_TypeChecks tests that :: expressions in modules
// pass through the full pipeline (parse → elaborate → typecheck).
func TestConsExpression_TypeChecks(t *testing.T) {
	tests := []struct {
		name string
		code string
	}{
		{
			name: "basic prepend",
			code: `module test
export func main() -> [int] {
  1 :: [2, 3]
}`,
		},
		{
			name: "chained cons right-associative",
			code: `module test
export func main() -> [int] {
  1 :: 2 :: 3 :: []
}`,
		},
		{
			name: "cons in let binding",
			code: `module test
export func main() -> [int] {
  let xs = 1 :: [2, 3];
  xs
}`,
		},
		{
			name: "cons in if branch",
			code: `module test
export func main() -> [int] {
  if true then 1 :: [2] else [3]
}`,
		},
		{
			name: "cons in match branch expression",
			code: `module test
export func main() -> [int] {
  match [1] {
    [] => [],
    x :: rest => x :: x :: rest
  }
}`,
		},
		{
			name: "cons with string list",
			code: `module test
export func main() -> [string] {
  "a" :: ["b", "c"]
}`,
		},
		{
			name: "cons onto empty list",
			code: `module test
export func main() -> [int] {
  42 :: []
}`,
		},
		{
			name: "pattern cons still works",
			code: `module test
export func main() -> int {
  match [1, 2, 3] {
    x :: rest => x,
    [] => 0
  }
}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{
				Mode: ModeCheck,
			}

			src := Source{
				Code:     tt.code,
				Filename: "",
				IsREPL:   false,
			}

			_, err := Run(cfg, src)
			if err != nil {
				t.Fatalf("pipeline error: %v", err)
			}
		})
	}
}

// TestConsExpression_TypeError tests that using :: with a non-list second argument
// produces a type error.
func TestConsExpression_TypeError(t *testing.T) {
	code := `module test
export func main() -> [int] {
  1 :: 2
}`

	cfg := Config{
		Mode: ModeCheck,
	}

	src := Source{
		Code:     code,
		Filename: "",
		IsREPL:   false,
	}

	_, err := Run(cfg, src)
	if err == nil {
		t.Fatal("expected type error for `1 :: 2`, but got none")
	}

	errStr := err.Error()
	if !strings.Contains(errStr, "unif") && !strings.Contains(errStr, "type") && !strings.Contains(errStr, "mismatch") && !strings.Contains(errStr, "list") {
		t.Errorf("expected a type-related error, got: %v", err)
	}
}
