package quickfix

import (
	"strings"
	"testing"
)

func TestComputeEdits_EffectStamp(t *testing.T) {
	t.Run("unannotated_return_type_adds_effects", func(t *testing.T) {
		src := `module m
export func greet(name: string) -> () {
  println(name)
}
`
		msg := "Effect checking failed for function 'greet'\n" +
			"  Function uses effects not declared in signature\n\n" +
			"  Missing effects: IO\n\n" +
			"  Current signature: func greet(...) -> T\n" +
			"  Suggested fix:     func greet(...) -> T ! {IO}\n"
		edits := ComputeEdits(Diagnostic{Code: "ERROR", Message: msg}, "m.ail", src)
		if len(edits) != 1 {
			t.Fatalf("got %d edits, want 1", len(edits))
		}
		got, n := ApplyEdits(src, edits)
		want := `module m
export func greet(name: string) -> () ! {IO} {
  println(name)
}
`
		if n != 1 || got != want {
			t.Fatalf("applied=%d\ngot:\n%s\nwant:\n%s", n, got, want)
		}
	})

	t.Run("pure_func_strips_pure_and_stamps_effects", func(t *testing.T) {
		src := `module m
export pure func greet(name: string) -> () {
  println(name)
}
`
		msg := "Effect checking failed for function 'greet'\n" +
			"  Function uses effects not declared in signature\n\n" +
			"  Missing effects: IO\n\n" +
			"  Current signature: func greet(...) -> T\n" +
			"  Suggested fix:     func greet(...) -> T ! {IO}\n"
		edits := ComputeEdits(Diagnostic{Code: "ERROR", Message: msg}, "m.ail", src)
		if len(edits) != 2 {
			t.Fatalf("got %d edits, want 2 (stamp + strip pure)", len(edits))
		}
		got, n := ApplyEdits(src, edits)
		want := `module m
export func greet(name: string) -> () ! {IO} {
  println(name)
}
`
		if n != 2 || got != want {
			t.Fatalf("applied=%d\ngot:\n%s\nwant:\n%s", n, got, want)
		}
	})

	t.Run("merge_existing_effect_row_preserving_limit", func(t *testing.T) {
		src := `module m
export func load(p: string) -> string ! {IO @limit=5} {
  println(p);
  readFile(p)
}
`
		msg := "Effect checking failed for function 'load'\n" +
			"  Function uses effects not declared in signature\n\n" +
			"  Missing effects: FS\n\n" +
			"  Current signature: func load(...) -> T ! {IO @limit=5}\n" +
			"  Suggested fix:     func load(...) -> T ! {FS, IO}\n"
		edits := ComputeEdits(Diagnostic{Code: "ERROR", Message: msg}, "m.ail", src)
		if len(edits) != 1 {
			t.Fatalf("got %d edits, want 1", len(edits))
		}
		got, n := ApplyEdits(src, edits)
		want := `module m
export func load(p: string) -> string ! {FS, IO @limit=5} {
  println(p);
  readFile(p)
}
`
		if n != 1 || got != want {
			t.Fatalf("applied=%d\ngot:\n%s\nwant:\n%s", n, got, want)
		}
	})

	t.Run("record_return_type_not_confused_with_body_brace", func(t *testing.T) {
		src := `module m
export func makeRec(p: string) -> { path: string, ok: bool } {
  println(p);
  { path: p, ok: true }
}
`
		msg := "Effect checking failed for function 'makeRec'\n" +
			"  Missing effects: IO\n" +
			"  Suggested fix:     func makeRec(...) -> T ! {IO}\n"
		edits := ComputeEdits(Diagnostic{Code: "ERROR", Message: msg}, "m.ail", src)
		if len(edits) != 1 {
			t.Fatalf("got %d edits, want 1", len(edits))
		}
		got, n := ApplyEdits(src, edits)
		want := `module m
export func makeRec(p: string) -> { path: string, ok: bool } ! {IO} {
  println(p);
  { path: p, ok: true }
}
`
		if n != 1 || got != want {
			t.Fatalf("applied=%d\ngot:\n%s\nwant:\n%s", n, got, want)
		}
	})

	t.Run("unannotated_return_and_multiline_params", func(t *testing.T) {
		src := `module m
-- func helper(x: int) -> int
func helper(
  a: string,
  b: { x: int }
) {
  println(a)
}
`
		msg := "Effect checking failed for function 'helper'\n" +
			"  Missing effects: IO\n" +
			"  Suggested fix:     func helper(...) -> T ! {IO}\n"
		edits := ComputeEdits(Diagnostic{Code: "ERROR", Message: msg}, "m.ail", src)
		if len(edits) != 1 {
			t.Fatalf("got %d edits, want 1", len(edits))
		}
		got, n := ApplyEdits(src, edits)
		if n != 1 || !strings.Contains(got, "\n) ! {IO} {\n") {
			t.Fatalf("applied=%d got:\n%s", n, got)
		}
	})

	t.Run("generic_and_contracts_preserved", func(t *testing.T) {
		src := `module m
export func logVal[a](x: a) -> Result[{ v: a }, string]
  requires { true }
{
  println("ok");
  Ok({ v: x })
}
`
		msg := "Effect checking failed for function 'logVal'\n" +
			"  Missing effects: IO\n" +
			"  Suggested fix:     func logVal(...) -> T ! {IO}\n"
		edits := ComputeEdits(Diagnostic{Code: "ERROR", Message: msg}, "m.ail", src)
		if len(edits) != 1 {
			t.Fatalf("got %d edits, want 1", len(edits))
		}
		got, n := ApplyEdits(src, edits)
		if n != 1 || !strings.Contains(got, "-> Result[{ v: a }, string] ! {IO}\n  requires { true }") {
			t.Fatalf("applied=%d got:\n%s", n, got)
		}
	})

	t.Run("no_suggested_fix_returns_nil", func(t *testing.T) {
		src := "module m\nexport func f() -> () ! {IO} { println(\"hi\") }\n"
		msg := "Effect checking failed for function 'f'\n  Unresolved effect tail: e\n"
		edits := ComputeEdits(Diagnostic{Code: "ERROR", Message: msg}, "m.ail", src)
		if len(edits) != 0 {
			t.Fatalf("expected 0 edits without Suggested fix, got %d", len(edits))
		}
	})
}

func TestComputeEdits_StringConcat(t *testing.T) {
	t.Run("literal_plus_ident", func(t *testing.T) {
		src := "module m\nexport pure func greet(name: string) -> string = \"Hello, \" ++ name\n"
		col := strings.Index(strings.Split(src, "\n")[1], "++") + 1
		edits := ComputeEdits(Diagnostic{
			Code:    "ERROR",
			Message: "++ operator at m.ail:2:60: `++` is for lists only. For strings use \"${expr}\" interpolation",
			Line:    2,
			Column:  col,
		}, "m.ail", src)
		if len(edits) != 1 {
			t.Fatalf("got %d edits, want 1", len(edits))
		}
		got, n := ApplyEdits(src, edits)
		want := "module m\nexport pure func greet(name: string) -> string = \"Hello, ${name}\"\n"
		if n != 1 || got != want {
			t.Fatalf("applied=%d\ngot:\n%s\nwant:\n%s", n, got, want)
		}
	})

	t.Run("two_idents", func(t *testing.T) {
		src := "module m\nexport pure func join2(a: string, b: string) -> string = a ++ b\n"
		edits := ComputeEdits(Diagnostic{
			Code:    "ERROR",
			Message: "type error in m (decl 0): ++ operator at m.ail:2:60: `++` is for lists only.",
		}, "m.ail", src)
		if len(edits) != 1 {
			t.Fatalf("got %d edits, want 1", len(edits))
		}
		got, n := ApplyEdits(src, edits)
		want := "module m\nexport pure func join2(a: string, b: string) -> string = \"${a}${b}\"\n"
		if n != 1 || got != want {
			t.Fatalf("applied=%d\ngot:\n%s\nwant:\n%s", n, got, want)
		}
	})

	t.Run("chained_three_parts_with_calls_and_literals", func(t *testing.T) {
		src := "module m\nexport pure func fmtMsg(x: int, name: string) -> string = \"id=\" ++ show(x) ++ \": \" ++ name ++ \"!\"\n"
		col := strings.Index(strings.Split(src, "\n")[1], "++") + 1
		edits := ComputeEdits(Diagnostic{
			Code:    "ERROR",
			Message: "++ operator at m.ail:2:65: `++` is for lists only.",
			Line:    2,
			Column:  col,
		}, "m.ail", src)
		if len(edits) != 1 {
			t.Fatalf("got %d edits, want 1", len(edits))
		}
		got, n := ApplyEdits(src, edits)
		want := "module m\nexport pure func fmtMsg(x: int, name: string) -> string = \"id=${show(x)}: ${name}!\"\n"
		if n != 1 || got != want {
			t.Fatalf("applied=%d\ngot:\n%s\nwant:\n%s", n, got, want)
		}
	})

	t.Run("two_literals_folded_preserving_escapes", func(t *testing.T) {
		src := "module m\nexport pure func f() -> string = \"a\\n\" ++ \"\\\"b\\\"\"\n"
		col := strings.Index(strings.Split(src, "\n")[1], "++") + 1
		edits := ComputeEdits(Diagnostic{
			Code:    "ERROR",
			Message: "`++` is for lists only",
			Line:    2,
			Column:  col,
		}, "m.ail", src)
		if len(edits) != 1 {
			t.Fatalf("got %d edits, want 1", len(edits))
		}
		got, n := ApplyEdits(src, edits)
		want := "module m\nexport pure func f() -> string = \"a\\n\\\"b\\\"\"\n"
		if n != 1 || got != want {
			t.Fatalf("applied=%d\ngot:\n%s\nwant:\n%s", n, got, want)
		}
	})

	t.Run("skips_non_literal_operand_containing_quotes", func(t *testing.T) {
		src := "module m\nexport pure func f(s: string) -> string = s ++ replace(s, \"a\", \"b\")\n"
		col := strings.Index(strings.Split(src, "\n")[1], "++") + 1
		edits := ComputeEdits(Diagnostic{
			Code:    "ERROR",
			Message: "`++` is for lists only",
			Line:    2,
			Column:  col,
		}, "m.ail", src)
		if len(edits) != 0 {
			t.Fatalf("expected 0 edits when non-literal operand contains quotes, got %+v", edits)
		}
	})
}
