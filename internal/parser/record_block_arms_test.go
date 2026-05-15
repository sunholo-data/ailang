package parser

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/lexer"
)

// M-PARSER-BLOCK-TR — record literal as the trailing expression in a `;`-separated block.
//
// Bug (inbox c8813647): `{ stmt; ...; {field: value} }` fails to parse because
// `parseBlockOrExpression`'s `;`-separated loop dispatches the inner LBRACE through
// the generic expression path which treats `{` as opening a new nested block, not
// a record literal. Same architectural bug class as M-DX16 (which fixed it for
// match arms by applying the IDENT-COLON / IDENT-PIPE lookahead at the LBRACE
// dispatch point), but in a different parser location.
//
// These five tests cover all five shapes from the design doc's success metrics.
// Until the fix in M1, every test in this file should FAIL with PAR_UNEXPECTED_TOKEN.

// parseSourceForTest parses a complete module source and returns parser errors.
// Returns the joined error string (empty if clean) for assertion convenience.
func parseSourceForTest(t *testing.T, src string) string {
	t.Helper()

	l := lexer.New(src, "test.ail")
	p := New(l)
	_ = p.Parse()

	if errs := p.Errors(); len(errs) > 0 {
		parts := make([]string, len(errs))
		for i, e := range errs {
			parts[i] = e.Error()
		}
		return strings.Join(parts, "\n")
	}
	return ""
}

// TestBlockTrailingRecord_LetIfRhs — the original repro from inbox c8813647.
// `let parsed = if cond then { stmt; record } else { stmt; record } in parsed.f`.
func TestBlockTrailingRecord_LetIfRhs(t *testing.T) {
	src := `module test_let_if
import std/io (println)

export func main() -> int ! {IO} {
  let parsed = if true
    then {
      println("a");
      {a: 1, b: 0}
    }
    else {
      println("b");
      {a: 2, b: 1}
    }
  in parsed.a
}
`
	if errs := parseSourceForTest(t, src); errs != "" {
		t.Errorf("Expected clean parse, got errors:\n%s", errs)
	}
}

// TestBlockTrailingRecord_TopLevel — same shape but as a top-level statement
// in a function body, not as the RHS of a let.
func TestBlockTrailingRecord_TopLevel(t *testing.T) {
	src := `module test_top
import std/io (println)

export func main() -> int ! {IO} {
  if true
    then {
      println("a");
      {a: 1, b: 0}
    }
    else {
      println("b");
      {a: 2, b: 1}
    };
  0
}
`
	if errs := parseSourceForTest(t, src); errs != "" {
		t.Errorf("Expected clean parse, got errors:\n%s", errs)
	}
}

// TestBlockTrailingRecord_Nested — nested blocks with trailing records:
// `{ s1; { s2; rec } }`. The inner block also ends with a record literal,
// so both the outer and inner positions exercise the dispatch.
func TestBlockTrailingRecord_Nested(t *testing.T) {
	src := `module test_nested
import std/io (println)

export func main() -> int ! {IO} {
  let outer = {
    println("outer");
    let inner = {
      println("inner");
      {a: 1, b: 0}
    } in
    inner
  } in
  outer.a
}
`
	if errs := parseSourceForTest(t, src); errs != "" {
		t.Errorf("Expected clean parse, got errors:\n%s", errs)
	}
}

// TestBlockTrailingRecord_FuncBody — function body block ending in a record literal.
// `func f() { stmt; rec }`.
func TestBlockTrailingRecord_FuncBody(t *testing.T) {
	src := `module test_funcbody
import std/io (println)

export func makeRecord() -> {a: int, b: int} ! {IO} {
  println("building");
  {a: 1, b: 0}
}

export func main() -> int ! {IO} {
  let r = makeRecord() in
  r.a
}
`
	if errs := parseSourceForTest(t, src); errs != "" {
		t.Errorf("Expected clean parse, got errors:\n%s", errs)
	}
}

// TestBlockTrailingRecord_MatchArm — regression check for M-DX16. The same
// `{ stmt; record }` shape inside a match-arm body must continue to parse
// (M-DX16 fixed match arms via parseRecordLiteral; our M1 fix in
// parseBlockOrExpression must not break that).
func TestBlockTrailingRecord_MatchArm(t *testing.T) {
	src := `module test_matcharm
import std/io (println)

type Tag = A | B

export func dispatch(t: Tag) -> {a: int, b: int} ! {IO} {
  match t {
    A => {
      println("got A");
      {a: 1, b: 0}
    },
    B => {
      println("got B");
      {a: 2, b: 1}
    }
  }
}

export func main() -> int ! {IO} {
  let r = dispatch(A) in
  r.a
}
`
	if errs := parseSourceForTest(t, src); errs != "" {
		t.Errorf("Expected clean parse, got errors:\n%s", errs)
	}
}
