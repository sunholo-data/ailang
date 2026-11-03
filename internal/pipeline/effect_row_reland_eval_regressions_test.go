package pipeline

import "testing"

// #1958 re-land, v0.57.0 pre-release sweep of 15,447 banked eval programs
// (eval_results `code`, deduplicated): 16 valid programs that v0.56.1 accepts
// were rejected by the re-land. Each case below is one of them or a minimal
// repro of its class. All must type-check.
//
//   - Opened lambda tails generalized into a scheme. A function returning an
//     unannotated lambda unified that lambda's opened tail with its return
//     annotation's tail, and the let/letrec generalization quantified it before
//     the tail was closed. Every use then instantiated a free tail that no
//     scope owns: "internal invariant: missing or malformed ApplicationEffects",
//     "distinct unresolved effect tails", or a spurious "Missing effects".
//   - Self rows two links away. The row unifier's substitution is not
//     idempotent (ρ1 ↦ {...ρ3}, ρ3 ↦ {...ρ6}), so a one-step lookup missed a
//     let-rec's own row from its third recursive call on: "same row variable
//     with different extensions".
func TestEffectRowRelandEvalRegressions(t *testing.T) {
	for _, tt := range []struct{ name, src string }{
		{"function returning a lambda, result called later", `pure func pipe(f: int -> int, g: int -> int) -> int -> int = \x. g(f(x))
pure func sub4(x: int) -> int = x - 4
let d = pipe(sub4, sub4)
export func main() -> () ! {IO} = println(show(d(11)))`},
		{"top-level lambda lets composed through pipe", `pure func pipe(f: int -> int, g: int -> int) -> int -> int = \x. g(f(x))
pure func subtract(x: int, y: int) -> int = x - y
let sub4 = \x. subtract(x, 4)
let double = \x. x * 2
let sub4ThenDouble = pipe(sub4, double)
export func main() -> () ! {IO} {
  let result = sub4ThenDouble(11);
  println("Result: ${show(result)}")
}`},
		{"curried top-level lets", `let multiply = \x. \y. x * y
let pipe = \f. \g. \x. g(f(x))
let sub4 = \x. x - 4
let double = multiply(2)
let sub4ThenDouble = pipe(sub4)(double)
export func main() -> () ! {IO} = println(show(sub4ThenDouble(11)))`},
		{"local curried letrec with lambda arguments", `export func main() -> () ! {IO} {
  let gt = \x. \y. x > y;
  letrec filter = \p. \xs.
    match xs {
      [] => [],
      x :: rest => if p(x) then x :: filter(p)(rest) else filter(p)(rest)
    };
  letrec sortDesc = \xs.
    match xs {
      [] => [],
      x :: rest => {
        let larger = sortDesc(filter(\y. gt(y)(x), rest));
        let smaller = sortDesc(filter(\y. not(gt(y)(x)), rest));
        larger ++ [x] ++ smaller
      }
    };
  println(show(sortDesc([5, 2, 8])))
}`},
		{"IO letrec, three nested ifs", `export func main() -> () ! {IO} = f(3, true, true, true)
func f(q: int, a: bool, b: bool, c: bool) -> () ! {IO} =
  letrec loop = \i. if i > q then () else
    if a then (if b then (if c then { println("x"); loop(i + 1) } else loop(i + 1)) else loop(i + 1)) else loop(i + 1)
  in loop(1)`},
		{"IO letrec, three nested matches", `export func main() -> () ! {IO} = f(3, Some(1))
func f(q: int, a: Option[int]) -> () ! {IO} =
  letrec loop = \i. if i > q then () else
    match a { Some(x) => match a { Some(y) => match a { Some(z) => { println("x"); loop(i + 1) }, None => loop(i + 1) }, None => loop(i + 1) }, None => loop(i + 1) }
  in loop(1)`},
		{"IO letrec, five nested ifs", `export func main() -> () ! {IO} = f(3, true)
func f(q: int, a: bool) -> () ! {IO} =
  letrec loop = \i. if i > q then () else
    if a then (if a then (if a then (if a then (if a then { println("x"); loop(i + 1) } else loop(i + 1)) else loop(i + 1)) else loop(i + 1)) else loop(i + 1)) else loop(i + 1)
  in loop(1)`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			src := "module eval_regress\nimport std/io (println)\nimport std/option (Option, Some, None)\n" + tt.src + "\n"
			if err := checkModuleSource(t, "eval_regress", src); err != nil {
				t.Fatal(err)
			}
		})
	}
}
