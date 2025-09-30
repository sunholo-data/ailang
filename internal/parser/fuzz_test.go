package parser

import (
	"testing"

	"github.com/sunholo/ailang/internal/lexer"
)

// FuzzParseExpr fuzzes the expression parser
func FuzzParseExpr(f *testing.F) {
	// Seed corpus with valid expressions
	seeds := []string{
		"1 + 2",
		"let x = 5 in x",
		"[1, 2, 3]",
		`\x. x + 1`,
		"{x: 1, y: 2}",
		"if true then 1 else 0",
		"match x { Some(y) => y, None => 0 }",
		"foo(bar, baz)",
		"1 + 2 * 3 - 4 / 5",
		"x && y || z",
		"[1, [2, 3], 4]",
		"{a: {b: {c: 1}}}",
		`\x. \y. x + y`,
		`let f = \x. x * 2 in f(21)`,
		"(1, 2, 3)",
		"true",
		"false",
		`"hello world"`,
		"42",
		"3.14",
		"foo.bar.baz",
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		// Must not panic
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("parser panicked on input %q: %v", input, r)
			}
		}()

		l := lexer.New(input, "fuzz")
		p := New(l)
		prog := p.Parse()

		// Either succeeds or returns structured errors
		// Both are acceptable - just must not panic
		_ = prog
		_ = p.Errors()
	})
}

// FuzzParseModule fuzzes module-level declarations
func FuzzParseModule(f *testing.F) {
	// Seed corpus with valid module code
	seeds := []string{
		"module Foo",
		"module Foo/Bar/Baz",
		"import Foo (bar, baz)",
		"import Foo/Bar (baz)",
		`func add(x, y) { x + y }`,
		`func factorial(n) { if n <= 1 then 1 else n * factorial(n - 1) }`,
		`pure func double(x) { x * 2 }`,
		`export func public() { 42 }`,
		`let x = 5`,
		`module Test
		import Std (map, filter)
		func process(x) { x + 1 }`,
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		// Must not panic
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("parser panicked on module input %q: %v", input, r)
			}
		}()

		l := lexer.New(input, "fuzz")
		p := New(l)
		prog := p.Parse()

		// Either succeeds or returns structured errors
		_ = prog
		_ = p.Errors()
	})
}

// FuzzParseMalformed fuzzes with intentionally malformed input
func FuzzParseMalformed(f *testing.F) {
	// Seed corpus with malformed inputs
	seeds := []string{
		"[1, 2, 3",
		"{x: 1, y:",
		"let x =",
		"if true then",
		`\x.`,
		"match x {",
		"func foo(",
		"1 + + 2",
		"* 1 + 2",
		"[[[[[",
		"}}}}}",
		")))))",
		"import",
		"module",
		"let let = let",
		"func func func",
		"1 + 2 * 3 /",
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		// Must not panic even on malformed input
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("parser panicked on malformed input %q: %v", input, r)
			}
		}()

		l := lexer.New(input, "fuzz")
		p := New(l)
		prog := p.Parse()

		// Should return errors for malformed input
		// But must not panic
		_ = prog
		_ = p.Errors()
	})
}

// FuzzParseUnicode fuzzes with various Unicode inputs
func FuzzParseUnicode(f *testing.F) {
	// Seed corpus with Unicode
	seeds := []string{
		"let π = 3.14",
		`"hello 世界"`,
		"let café = true",
		"let emoji = \"🚀\"",
		"λx. x + 1", // Lambda character
		"let résumé = {}",
		"\xEF\xBB\xBF42", // UTF-8 BOM
		"let x = 1\r\n",  // CRLF
		"let y = 2\n",    // LF
		"let z = 3\r",    // CR
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		// Must handle Unicode gracefully
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("parser panicked on Unicode input %q: %v", input, r)
			}
		}()

		l := lexer.New(input, "fuzz")
		p := New(l)
		prog := p.Parse()

		_ = prog
		_ = p.Errors()
	})
}
