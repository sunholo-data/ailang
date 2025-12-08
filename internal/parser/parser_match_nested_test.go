package parser

import (
	"testing"

	"github.com/sunholo/ailang/internal/lexer"
)

// TestNestedMatchSimple tests 2-level nesting with pure expressions (currently works)
func TestNestedMatchSimple(t *testing.T) {
	input := `
module test

func test() -> int {
  let x = (1, 2);
  match x {
    (a, b) => {
      let y = (3, 4);
      match y {
        (c, d) => c + d
      }
    }
  }
}`

	l := lexer.New(input, "test.ail")
	p := New(l)
	program := p.Parse()

	if len(p.Errors()) != 0 {
		for _, err := range p.Errors() {
			t.Errorf("  %s", err)
		}
		t.Fatalf("parser had %d errors", len(p.Errors()))
	}

	if program == nil || program.File == nil {
		t.Fatal("Expected file to parse successfully")
	}

	if len(program.File.Funcs) != 1 {
		t.Fatalf("Expected 1 function declaration, got %d", len(program.File.Funcs))
	}

	if program.File.Funcs[0].Name != "test" {
		t.Fatalf("Expected function named 'test', got %s", program.File.Funcs[0].Name)
	}
}

// TestNestedMatchWithIO tests 2-level nesting with IO effects (currently works)
func TestNestedMatchWithIO(t *testing.T) {
	input := `
module test

import std/io (println)

func test() -> () ! {IO} {
  let x = (1, 2);
  match x {
    (a, b) => {
      println("outer");
      let y = (3, 4);
      match y {
        (c, d) => println("inner")
      }
    }
  }
}`

	l := lexer.New(input, "test.ail")
	p := New(l)
	program := p.Parse()

	if len(p.Errors()) != 0 {
		for _, err := range p.Errors() {
			t.Errorf("  %s", err)
		}
		t.Fatalf("parser had %d errors", len(p.Errors()))
	}

	if program == nil || program.File == nil {
		t.Fatal("Expected file to parse successfully")
	}

	if len(program.File.Funcs) != 1 {
		t.Fatalf("Expected 1 function declaration, got %d", len(program.File.Funcs))
	}
}

// TestNestedMatchThreeLevel tests 3-level nesting with pure expressions
// This test is expected to fail until Phase 2 fix is implemented
func TestNestedMatchThreeLevel(t *testing.T) {
	input := `
module test

func test() -> int {
  let x = (1, 2);
  match x {
    (a, b) => {
      let y = (3, 4);
      match y {
        (c, d) => {
          let z = (5, 6);
          match z {
            (e, f) => e + f
          }
        }
      }
    }
  }
}`

	l := lexer.New(input, "test.ail")
	p := New(l)
	program := p.Parse()

	if len(p.Errors()) != 0 {
		for _, err := range p.Errors() {
			t.Errorf("  %s", err)
		}
		t.Fatalf("parser had %d errors", len(p.Errors()))
	}

	if program == nil || program.File == nil {
		t.Fatal("Expected file to parse successfully")
	}

	if len(program.File.Funcs) != 1 {
		t.Fatalf("Expected 1 function declaration, got %d", len(program.File.Funcs))
	}
}

// TestNestedMatchThreeLevelIO tests 3-level nesting with IO effects
// This is the KEY TEST that reproduces the gpt5-mini benchmark failure
// Expected to fail until Phase 2 fix is implemented
func TestNestedMatchThreeLevelIO(t *testing.T) {
	input := `
module test

import std/io (println)

func add(value: int, state: int) -> (int, int) {
  let newState = state + value;
  (newState, newState)
}

func main() -> () ! {IO} {
  let state0 = 0;
  println("Initial: " ++ show(state0));
  let r1 = add(5, state0);
  match r1 {
    (state1, _) => {
      println("After add: " ++ show(state1));
      let r2 = add(10, state1);
      match r2 {
        (state2, _) => {
          println("After second add: " ++ show(state2));
          let r3 = add(3, state2);
          match r3 {
            (state3, _) => println("Final: " ++ show(state3))
          }
        }
      }
    }
  }
}`

	l := lexer.New(input, "test.ail")
	p := New(l)
	program := p.Parse()

	if len(p.Errors()) != 0 {
		for _, err := range p.Errors() {
			t.Errorf("  %s", err)
		}
		t.Fatalf("parser had %d errors", len(p.Errors()))
	}

	if program == nil || program.File == nil {
		t.Fatal("Expected file to parse successfully")
	}

	// Verify we got both function declarations
	if len(program.File.Funcs) != 2 {
		t.Fatalf("Expected 2 function declarations, got %d", len(program.File.Funcs))
	}
}

// TestMatchArmBlockTrailingSemicolon tests block arms with trailing semicolons
func TestMatchArmBlockTrailingSemicolon(t *testing.T) {
	input := `
module test

import std/io (println)

func test() -> () ! {IO} {
  let x = 1;
  match x {
    1 => {
      println("one");
    }
  }
}`

	l := lexer.New(input, "test.ail")
	p := New(l)
	program := p.Parse()

	if len(p.Errors()) != 0 {
		for _, err := range p.Errors() {
			t.Errorf("  %s", err)
		}
		t.Fatalf("parser had %d errors", len(p.Errors()))
	}

	if program == nil || program.File == nil {
		t.Fatal("Expected file to parse successfully")
	}
}

// TestMatchArmBlockEmpty tests empty block in match arm
func TestMatchArmBlockEmpty(t *testing.T) {
	input := `
module test

func test() -> () {
  let x = 1;
  match x {
    1 => {},
    _ => ()
  }
}`

	l := lexer.New(input, "test.ail")
	p := New(l)
	program := p.Parse()

	if len(p.Errors()) != 0 {
		for _, err := range p.Errors() {
			t.Errorf("  %s", err)
		}
		t.Fatalf("parser had %d errors", len(p.Errors()))
	}

	if program == nil || program.File == nil {
		t.Fatal("Expected file to parse successfully")
	}
}

// TestMatchArmBlockCommentAdjacency tests comments near block delimiters
func TestMatchArmBlockCommentAdjacency(t *testing.T) {
	input := `
module test

import std/io (println)

func test() -> () ! {IO} {
  let x = 1;
  match x {
    1 => { // comment after open brace
      println("test");
      let y = 2;
      match y {
        2 => println("nested") // comment before close brace
      } // comment after close brace
    }
  }
}`

	l := lexer.New(input, "test.ail")
	p := New(l)
	program := p.Parse()

	if len(p.Errors()) != 0 {
		for _, err := range p.Errors() {
			t.Errorf("  %s", err)
		}
		t.Fatalf("parser had %d errors", len(p.Errors()))
	}

	if program == nil || program.File == nil {
		t.Fatal("Expected file to parse successfully")
	}
}

// TestMatchArmBlockNestedMatchNoSemicolon tests nested match as final expression
func TestMatchArmBlockNestedMatchNoSemicolon(t *testing.T) {
	input := `
module test

func test() -> int {
  let x = (1, 2);
  match x {
    (a, b) => {
      let y = (3, 4);
      match y {
        (c, d) => c
      }
    }
  }
}`

	l := lexer.New(input, "test.ail")
	p := New(l)
	program := p.Parse()

	if len(p.Errors()) != 0 {
		for _, err := range p.Errors() {
			t.Errorf("  %s", err)
		}
		t.Fatalf("parser had %d errors", len(p.Errors()))
	}

	if program == nil || program.File == nil {
		t.Fatal("Expected file to parse successfully")
	}
}

// TestMatchArmBlockMixedEffects tests mixed pure and effectful expressions in nested matches
func TestMatchArmBlockMixedEffects(t *testing.T) {
	input := `
module test

import std/io (println)

func test() -> int ! {IO} {
  let x = (1, 2);
  match x {
    (a, b) => {
      println("Computing...");
      let y = a + b;
      match y {
        3 => {
          println("Got 3");
          y
        },
        _ => 0
      }
    }
  }
}`

	l := lexer.New(input, "test.ail")
	p := New(l)
	program := p.Parse()

	if len(p.Errors()) != 0 {
		for _, err := range p.Errors() {
			t.Errorf("  %s", err)
		}
		t.Fatalf("parser had %d errors", len(p.Errors()))
	}

	if program == nil || program.File == nil {
		t.Fatal("Expected file to parse successfully")
	}
}

// TestMatchArmBlockWhitespaceStress tests various whitespace patterns
func TestMatchArmBlockWhitespaceStress(t *testing.T) {
	input := `
module test

import std/io (println)

func test() -> () ! {IO} {
  let x = 1;
  match x {
    1 => {

      println("lots of whitespace");

      let y = 2;

      match y {
        2 => println("nested")
      }

    }
  }
}`

	l := lexer.New(input, "test.ail")
	p := New(l)
	program := p.Parse()

	if len(p.Errors()) != 0 {
		for _, err := range p.Errors() {
			t.Errorf("  %s", err)
		}
		t.Fatalf("parser had %d errors", len(p.Errors()))
	}

	if program == nil || program.File == nil {
		t.Fatal("Expected file to parse successfully")
	}
}

// TestMatchArmBlockMalformedRecovery tests parser error recovery with malformed blocks
func TestMatchArmBlockMalformedRecovery(t *testing.T) {
	input := `
module test

func test() -> int {
  let x = 1;
  match x {
    1 => {
      let y = 2
      // Missing close brace intentionally
}`

	l := lexer.New(input, "test.ail")
	p := New(l)
	program := p.Parse()

	if program == nil || program.File == nil {
		t.Fatal("Expected parser to return a file even with errors")
	}

	// We expect parser errors for malformed input
	if len(p.Errors()) == 0 {
		t.Fatal("Expected parser errors for malformed input")
	}
}
