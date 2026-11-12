package types_test

import (
	"strings"
	"testing"
)

// #2238: the parameter and return annotations of a function literal constrain
// it exactly as a top-level `func`'s do. Every form that can carry an
// annotation goes through elaborate.recordSignatureAnnots; each row below pins
// one form (rejects) or one shape that must keep checking (accepts). Seen
// through the pipeline (like ambiguous_ctor_pipeline_test.go), so the
// checker side (literalParamType) is exercised in this package.
func TestFuncLitAnnotations_Enforced(t *testing.T) {
	cases := []struct {
		name    string
		src     string
		wantErr string // "" = must check
	}{
		// --- rejected: the annotation is the only thing that can catch these ---
		{
			name: "issue_repro_param", // body ignores x, so only the annotation constrains it
			src: `module m/main
export func main() -> () ! {IO} {
  let f = func(x: int) -> int { 0 };
  println(show(f("str")))
}`,
			wantErr: "int vs string",
		},
		{
			name: "return_annotation",
			src: `module m/main
export func main() -> () ! {IO} {
  let f = func(x: int) -> int { "s" };
  println(f(1))
}`,
			wantErr: "return type annotation",
		},
		{
			name: "fat_arrow_lambda_param", // func(x: T) => e
			src: `module m/main
export func main() -> () ! {IO} {
  let f = func(x: int) => 0;
  println(show(f("str")))
}`,
			wantErr: "int vs string",
		},
		{
			name: "module_level_let_literal",
			src: `module m/main
let g = func(x: int) -> int { 0 }
export func main() -> () ! {IO} = println(show(g("str")))`,
			wantErr: "int vs string",
		},
		{
			name: "letrec_literal",
			src: `module m/main
export func main() -> () ! {IO} {
  letrec f = func(x: int) -> int { 0 };
  println(show(f("str")))
}`,
			wantErr: "int vs string",
		},
		{
			name: "literal_as_callback_argument",
			src: `module m/main
import std/list (map)
export func main() -> () ! {IO} =
  println(show(map(func(x: int) -> int { 0 }, ["a", "b"])))`,
			wantErr: "function application of map",
		},
		// --- controls: top-level and let-annotation paths were already enforced ---
		{
			name: "control_top_level_func",
			src: `module m/main
func top(x: int) -> int = 0
export func main() -> () ! {IO} = println(show(top("str")))`,
			wantErr: "int vs string",
		},
		{
			name: "control_let_type_annotation",
			src: `module m/main
export func main() -> () ! {IO} {
  let f: int -> int = \x. 0;
  println(show(f("str")))
}`,
			wantErr: "int vs string",
		},
		// --- accepted: correct programs that must keep checking ---
		// --- an explicit `! {}` is an annotation (review of #2289: d02, d04) ---
		{
			name: "explicit_pure_literal_that_prints",
			src: `module m/main
export func main() -> () ! {IO} {
  let f = func(x: int) -> int ! {} { println("side"); x };
  println(show(f(1)))
}`,
			wantErr: "uses effects not declared",
		},
		{
			name: "explicit_pure_literal_calling_an_IO_callback",
			src: `module m/main
export func main() -> () ! {IO} {
  let run = func(g: int -> () ! {IO}) -> () ! {} { g(1) };
  run(\n. println(show(n)))
}`,
			wantErr: "uses effects not declared",
		},
		{
			// motoko-fork localannot shape: a lambda is atomic, so ANF leaves it
			// directly in the record; the annotation check must still reach it.
			name: "explicit_pure_literal_stored_in_a_record",
			src: `module m/main
type W = { f: (int) -> int }
export func main() -> () ! {IO} {
  let w: W = { f: func(n: int) -> int ! {} { println("side"); n } };
  println(show(w.f(1)))
}`,
			wantErr: "uses effects not declared",
		},
		{
			name: "explicit_pure_literal_that_is_pure",
			src: `module m/main
import std/list (map)
export func main() -> () ! {IO} = println(show(map(func(x: int) -> int ! {} { x + 1 }, [1, 2])))`,
		},
		{
			name: "foldl_callback",
			src: `module m/main
import std/list (foldl)
export func main() -> () ! {IO} = println(show(foldl(func(acc: int, x: int) -> int { acc + x }, 0, [1, 2, 3])))`,
		},
		{
			name: "let_polymorphic_annotated_literal",
			src: `module m/main
export func main() -> () ! {IO} {
  let id = func(x: a) -> a { x };
  println("${show(id(1))}${id("s")}")
}`,
		},
		{
			name: "enclosing_type_param",
			src: `module m/main
import std/list (map)
pure func keep[a](xs: [a]) -> [a] = map(func(x: a) -> a { x }, xs)
export func main() -> () ! {IO} = println(show(keep([1, 2])))`,
		},
		{
			name: "effectful_callback_param",
			src: `module m/main
export func main() -> () ! {IO} {
  let run = func(g: (int) -> () ! {IO}) -> () ! {IO} { g(1) };
  run(\n. println(show(n)))
}`,
		},
		{
			name: "adt_record_list_and_open_record_params",
			src: `module m/main
type P = { name: string, age: int }
export func main() -> () ! {IO} {
  let f = func(o: Option[int], p: P, xs: [int]) -> int { match o { Some(n) => n + p.age, None => 0 } };
  let nm = func(r: { name: string | rest }) -> string { r.name };
  println(show(f(Some(1), { name: "a", age: 2 }, [1])));
  println(nm({ name: "z", k: 1 }))
}`,
		},
		{
			// motoko-fork stub_step.live_ports shape: an annotated callback
			// parameter passed on to an effect-polymorphic callee. Seeding the
			// annotation leaked its open effect tail into the enclosing function
			// (60 downstream files); literal annotations are equalities instead.
			name: "annotated_callback_passed_to_effect_polymorphic_callee",
			src: `module m/main
type Ex = { v: int }
type Ports = Ports({ step: (int, int -> () ! {IO}) -> Ex ! {IO, Trace} })
func poly(n: int, cb: int -> () ! {e}) -> int ! {e} { cb(n); n }
export func live() -> Ports ! {IO} {
  let step = func(n: int, cb: int -> () ! {IO}) -> Ex ! {IO, Trace} { let m = poly(n, cb); { v: m } };
  Ports({ step: step })
}`,
		},
		{
			name: "recursive_literal",
			src: `module m/main
export func main() -> () ! {IO} {
  letrec fact = func(n: int) -> int { if n <= 1 then 1 else n * fact(n - 1) };
  println(show(fact(5)))
}`,
		},
		{
			name: "unannotated_param_stays_inferred",
			src: `module m/main
export func main() -> () ! {IO} {
  let f = func(x) -> int { 0 };
  println(show(f("str")))
}`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := checkModules(t, map[string]string{"m/main.ail": tc.src})
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("expected the program to check, got: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected a type error containing %q, program checked", tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected error containing %q, got: %v", tc.wantErr, err)
			}
		})
	}
}
