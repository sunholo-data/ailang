package parser

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/lexer"
)

// TestListConsPatternSimple tests basic :: (cons) pattern matching
// This is the minimal reproduction of the bug from M-DX10
func TestListConsPatternSimple(t *testing.T) {
	input := `module test

func sum(xs: List[int]) -> int {
  match xs {
    [] => 0,
    ::(x, rest) => x + sum(rest)
  }
}`

	l := lexer.New(input, "test.ail")
	p := New(l)
	program := p.Parse()

	// This test is expected to FAIL until the bug is fixed
	// The parser currently fails with PAR_UNEXPECTED_TOKEN at ::(x, rest)
	if len(p.Errors()) != 0 {
		t.Log("Parser errors (bug reproduction):")
		for _, err := range p.Errors() {
			t.Logf("  %s", err)
		}
		t.Fatal("BUG REPRODUCED: Parser failed to accept :: pattern after [] pattern")
	}

	if program == nil || program.File == nil {
		t.Fatal("Expected file to parse successfully")
	}

	if len(program.File.Funcs) != 1 {
		t.Fatalf("Expected 1 function declaration, got %d", len(program.File.Funcs))
	}
}

// TestListConsPatternFromBenchmark tests the exact failing code from json_parse benchmark
func TestListConsPatternFromBenchmark(t *testing.T) {
	input := `module test

type JSON =
  | JNull
  | JBool(bool)
  | JNum(float)
  | JStr(string)
  | JArr(List[JSON])
  | JObj(List[(string, JSON)])

func findKey(kvs: List[(string, JSON)], key: string) -> JSON {
  match kvs {
    [] => JNull,
    ::((k, v), rest) => {
      if k == key then v else findKey(rest, key)
    }
  }
}`

	l := lexer.New(input, "test.ail")
	p := New(l)
	program := p.Parse()

	// This reproduces the exact failure from claude-haiku-4-5 on json_parse benchmark
	if len(p.Errors()) != 0 {
		t.Log("Parser errors (benchmark reproduction):")
		for _, err := range p.Errors() {
			t.Logf("  %s", err)
		}
		t.Fatal("BUG REPRODUCED: Parser failed on json_parse benchmark pattern")
	}

	if program == nil || program.File == nil {
		t.Fatal("Expected file to parse successfully")
	}
}

// TestListConsPatternMultipleArms tests :: patterns with 3+ match arms
func TestListConsPatternMultipleArms(t *testing.T) {
	input := `module test

func describe(xs: List[int]) -> string {
  match xs {
    [] => "empty",
    ::(x, []) => "singleton",
    ::(x, ::(y, [])) => "pair",
    ::(x, rest) => "many"
  }
}`

	l := lexer.New(input, "test.ail")
	p := New(l)
	program := p.Parse()

	if len(p.Errors()) != 0 {
		t.Log("Parser errors:")
		for _, err := range p.Errors() {
			t.Logf("  %s", err)
		}
		t.Fatal("Failed to parse multiple :: pattern arms")
	}

	if program == nil || program.File == nil {
		t.Fatal("Expected file to parse successfully")
	}
}

// TestListConsPatternNested tests nested :: patterns like ::(h, ::(h2, t2))
func TestListConsPatternNested(t *testing.T) {
	input := `module test

func secondElement(xs: List[int]) -> int {
  match xs {
    ::(_, ::(x, _)) => x,
    _ => 0
  }
}`

	l := lexer.New(input, "test.ail")
	p := New(l)
	program := p.Parse()

	if len(p.Errors()) != 0 {
		t.Log("Parser errors:")
		for _, err := range p.Errors() {
			t.Logf("  %s", err)
		}
		t.Fatal("Failed to parse nested :: patterns")
	}

	if program == nil || program.File == nil {
		t.Fatal("Expected file to parse successfully")
	}
}

// TestListConsPatternWithTuple tests :: pattern with tuple elements
func TestListConsPatternWithTuple(t *testing.T) {
	input := `module test

func firstKey(kvs: List[(string, int)]) -> string {
  match kvs {
    [] => "",
    ::((k, v), rest) => k
  }
}`

	l := lexer.New(input, "test.ail")
	p := New(l)
	program := p.Parse()

	if len(p.Errors()) != 0 {
		t.Log("Parser errors:")
		for _, err := range p.Errors() {
			t.Logf("  %s", err)
		}
		t.Fatal("Failed to parse :: pattern with tuple")
	}

	if program == nil || program.File == nil {
		t.Fatal("Expected file to parse successfully")
	}
}

// TestListConsPatternWithConstructor tests :: pattern with ADT constructor
func TestListConsPatternWithConstructor(t *testing.T) {
	input := `module test

type Option = | None | Some(int)

func firstSome(xs: List[Option]) -> int {
  match xs {
    [] => 0,
    ::(None, rest) => firstSome(rest),
    ::(Some(x), rest) => x
  }
}`

	l := lexer.New(input, "test.ail")
	p := New(l)
	program := p.Parse()

	if len(p.Errors()) != 0 {
		t.Log("Parser errors:")
		for _, err := range p.Errors() {
			t.Logf("  %s", err)
		}
		t.Fatal("Failed to parse :: pattern with ADT constructor")
	}

	if program == nil || program.File == nil {
		t.Fatal("Expected file to parse successfully")
	}
}

// TestListConsPatternWithRecord is the regression guard for M-DX-RECORD-CONS.
// A record-literal pattern as the head of an infix :: cons pattern used to fail
// with PAT_INVALID_CONS (a parser cursor-position mismatch after the record
// pattern). Verified fixed and closed as a ghost in mission iteration 18
// (2026-07-13); this test locks it in — both the infix and canonical forms.
func TestListConsPatternWithRecord(t *testing.T) {
	cases := []struct {
		name  string
		input string
	}{
		{
			name: "infix record head",
			input: `module test

func firstText(rows: List[{text: string, bold: bool}]) -> string {
  match rows {
    {text: s, bold: b} :: rest => s,
    [] => ""
  }
}`,
		},
		{
			name: "canonical record head",
			input: `module test

func firstText(rows: List[{text: string, bold: bool}]) -> string {
  match rows {
    ::({text: s, bold: b}, rest) => s,
    [] => ""
  }
}`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			l := lexer.New(tc.input, "test.ail")
			p := New(l)
			program := p.Parse()

			if len(p.Errors()) != 0 {
				t.Log("Parser errors:")
				for _, err := range p.Errors() {
					t.Logf("  %s", err)
				}
				t.Fatal("REGRESSION: record-literal head in :: cons pattern rejected (M-DX-RECORD-CONS)")
			}

			if program == nil || program.File == nil {
				t.Fatal("Expected file to parse successfully")
			}
		})
	}
}

// TestListConsPatternInvalidNoArgs tests error when :: has no arguments
func TestListConsPatternInvalidNoArgs(t *testing.T) {
	input := `module test

func f(xs: List[int]) -> int {
  match xs {
    [] => 0,
    :: => 1
  }
}`

	l := lexer.New(input, "test.ail")
	p := New(l)
	_ = p.Parse()

	// Should have error: :: without arguments
	if len(p.Errors()) == 0 {
		t.Fatal("Expected parser error for :: without arguments")
	}

	found := false
	for _, err := range p.Errors() {
		errStr := err.Error()
		// Check if error contains "PAT_INVALID_CONS"
		if len(errStr) >= 16 && errStr[:16] == "PAT_INVALID_CONS" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected PAT_INVALID_CONS error, got: %v", p.Errors())
	}
}
