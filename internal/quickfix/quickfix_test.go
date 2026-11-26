package quickfix

import (
	"strings"
	"testing"
)

func TestComputeEdits_MechanicalFixes(t *testing.T) {
	t.Run("PAR017_equals_before_brace", func(t *testing.T) {
		src := "module m\nexport pure func f(x: int) -> int = {\n  let y = x + 1;\n  y\n}\n"
		edits := ComputeEdits(Diagnostic{Code: "PAR017", Line: 3, Column: 16}, "m.ail", src)
		if len(edits) != 1 {
			t.Fatalf("got %d edits, want 1", len(edits))
		}
		got, n := ApplyEdits(src, edits)
		if n != 1 || !strings.Contains(got, "func f(x: int) -> int {") {
			t.Fatalf("PAR017 fix failed: n=%d got:\n%s", n, got)
		}
	})

	t.Run("PAR018_let_rec", func(t *testing.T) {
		src := "module m\nexport pure func f(n: int) -> int {\n  let rec go = \\x. x;\n  go(n)\n}\n"
		edits := ComputeEdits(Diagnostic{Code: "PAR018", Line: 3, Column: 7}, "m.ail", src)
		if len(edits) != 1 {
			t.Fatalf("got %d edits, want 1", len(edits))
		}
		got, n := ApplyEdits(src, edits)
		if n != 1 || !strings.Contains(got, "letrec go =") {
			t.Fatalf("PAR018 fix failed: n=%d got:\n%s", n, got)
		}
	})

	t.Run("PAR_HASH_COMMENT", func(t *testing.T) {
		src := "module m\n# Python comment\nexport pure func f() -> int = 1\n"
		edits := ComputeEdits(Diagnostic{Code: "PAR_HASH_COMMENT", Line: 2, Column: 1}, "m.ail", src)
		if len(edits) != 1 {
			t.Fatalf("got %d edits, want 1", len(edits))
		}
		got, n := ApplyEdits(src, edits)
		if n != 1 || !strings.Contains(got, "-- Python comment") {
			t.Fatalf("PAR_HASH_COMMENT fix failed: n=%d got:\n%s", n, got)
		}
	})

	t.Run("IMP010_rename_and_builtin", func(t *testing.T) {
		src := "module m\nimport std/string (nthChar, show)\nexport pure func f(s: string) -> string = s\n"
		e1 := ComputeEdits(Diagnostic{
			Code:    "IMP010",
			Message: "symbol 'nthChar' not exported by 'std/string' — did you mean 'charAt'? (exported by 'std/string')",
		}, "m.ail", src)
		got1, n1 := ApplyEdits(src, e1)
		if n1 != 1 || !strings.Contains(got1, "import std/string (charAt, show)") {
			t.Fatalf("rename failed: n=%d got:\n%s", n1, got1)
		}
		e2 := ComputeEdits(Diagnostic{
			Code:    "IMP010",
			Message: "symbol 'show' not exported by 'std/string' — 'show' is a builtin available in every module; remove it from the import list",
		}, "m.ail", got1)
		got2, n2 := ApplyEdits(got1, e2)
		if n2 != 1 || !strings.Contains(got2, "import std/string (charAt)") {
			t.Fatalf("builtin removal failed: n=%d got:\n%s", n2, got2)
		}
	})

	t.Run("missing_stdlib_import_single_module", func(t *testing.T) {
		src := "module m\nexport pure func f(xs: [int]) -> int = foldl(\\a b. a + b, 0, xs)\n"
		edits := ComputeEdits(Diagnostic{
			Code:    "ERROR",
			Message: "undefined variable: foldl at m.ail:2:40",
		}, "m.ail", src)
		if len(edits) != 1 {
			t.Fatalf("got %d edits, want 1", len(edits))
		}
		got, n := ApplyEdits(src, edits)
		if n != 1 || !strings.Contains(got, "import std/list (foldl)\n") {
			t.Fatalf("missing import failed: n=%d got:\n%s", n, got)
		}
	})

	t.Run("missing_stdlib_import_disambiguated", func(t *testing.T) {
		src := "module m\nimport std/list (filterMap)\nexport pure func f(xs: [int]) -> [int] = map(\\x. x + 1, xs)\n"
		edits := ComputeEdits(Diagnostic{
			Code:    "ERROR",
			Message: "undefined variable: map at m.ail:3:42 — `map` is exported by std/list, std/option, std/result",
		}, "m.ail", src)
		if len(edits) != 1 {
			t.Fatalf("got %d edits, want 1", len(edits))
		}
		got, n := ApplyEdits(src, edits)
		if n != 1 || !strings.Contains(got, "import std/list (filterMap, map)") {
			t.Fatalf("import disambiguation failed: n=%d got:\n%s", n, got)
		}
	})
}

func TestComputeEdits_TCMatch002(t *testing.T) {
	t.Run("multiline_single_missing_arm", func(t *testing.T) {
		src := `module m
type Color = Red | Green | Blue
export pure func code(c: Color) -> int =
  match c {
    Red => 1,
    Green => 2
  }
`
		edits := ComputeEdits(Diagnostic{
			Code:    "TC_MATCH_002",
			Message: "non-exhaustive match on Color — no arm matches: Blue",
			Line:    4,
			Column:  3,
		}, "m.ail", src)
		if len(edits) != 1 {
			t.Fatalf("got %d edits, want 1", len(edits))
		}
		if !strings.Contains(edits[0].Title, "Blue => _?") {
			t.Errorf("unexpected title: %q", edits[0].Title)
		}
		got, n := ApplyEdits(src, edits)
		want := `module m
type Color = Red | Green | Blue
export pure func code(c: Color) -> int =
  match c {
    Red => 1,
    Green => 2,
    Blue => _?
  }
`
		if n != 1 || got != want {
			t.Fatalf("applied=%d\ngot:\n%s\nwant:\n%s", n, got, want)
		}
	})

	t.Run("multiline_trailing_comma_and_multiple_patterns_with_args", func(t *testing.T) {
		src := `module m
type Tree = Empty | Leaf(int) | Node(Tree, Tree)
export pure func size(t: Tree) -> int =
  match t {
    Empty => 0,
  }
`
		edits := ComputeEdits(Diagnostic{
			Code:    "TC_MATCH_002",
			Message: "non-exhaustive match on Tree — no arm matches: Leaf(_), Node(_, _)",
			Line:    4,
			Column:  3,
		}, "m.ail", src)
		if len(edits) != 1 {
			t.Fatalf("got %d edits, want 1", len(edits))
		}
		got, n := ApplyEdits(src, edits)
		want := `module m
type Tree = Empty | Leaf(int) | Node(Tree, Tree)
export pure func size(t: Tree) -> int =
  match t {
    Empty => 0,
    Leaf(_) => _?,
    Node(_, _) => _?
  }
`
		if n != 1 || got != want {
			t.Fatalf("applied=%d\ngot:\n%s\nwant:\n%s", n, got, want)
		}
	})

	t.Run("single_line_match", func(t *testing.T) {
		src := "module m\ntype Color = Red | Blue\nexport pure func f(c: Color) -> int = match c { Red => 1 }\n"
		edits := ComputeEdits(Diagnostic{
			Code:    "TC_MATCH_002",
			Message: "non-exhaustive match on Color — no arm matches: Blue",
			Line:    3,
			Column:  39,
		}, "m.ail", src)
		if len(edits) != 1 {
			t.Fatalf("got %d edits, want 1", len(edits))
		}
		got, n := ApplyEdits(src, edits)
		want := "module m\ntype Color = Red | Blue\nexport pure func f(c: Color) -> int = match c { Red => 1, Blue => _? }\n"
		if n != 1 || got != want {
			t.Fatalf("applied=%d\ngot:\n%s\nwant:\n%s", n, got, want)
		}
	})

	t.Run("infinite_domain_wildcard_stripped", func(t *testing.T) {
		src := `module m
export pure func f(n: int) -> int =
  match n {
    0 => 1
  }
`
		edits := ComputeEdits(Diagnostic{
			Code:    "TC_MATCH_002",
			Message: "non-exhaustive match on int — no arm matches: _ (any int other than 0)",
			Line:    3,
			Column:  3,
		}, "m.ail", src)
		if len(edits) != 1 {
			t.Fatalf("got %d edits, want 1", len(edits))
		}
		got, n := ApplyEdits(src, edits)
		if n != 1 || !strings.Contains(got, "    0 => 1,\n    _ => _?\n  }") {
			t.Fatalf("applied=%d got:\n%s", n, got)
		}
	})

	t.Run("nested_braces_in_arm_body", func(t *testing.T) {
		src := `module m
type Color = Red | Blue
export pure func f(c: Color) -> int =
  match c {
    Red => {
      let x = 1;
      x
    }
  }
`
		edits := ComputeEdits(Diagnostic{
			Code:    "TC_MATCH_002",
			Message: "non-exhaustive match on Color — no arm matches: Blue",
			Line:    4,
			Column:  3,
		}, "m.ail", src)
		if len(edits) != 1 {
			t.Fatalf("got %d edits, want 1", len(edits))
		}
		got, n := ApplyEdits(src, edits)
		if n != 1 || !strings.Contains(got, "    },\n    Blue => _?\n  }") {
			t.Fatalf("applied=%d got:\n%s", n, got)
		}
	})
}
