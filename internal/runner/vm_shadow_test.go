package runner

import (
	"path/filepath"
	"strings"
	"testing"
)

// shadowRegressionSource holds one function per shadowing shape. The first
// group printed a wrong answer on both VM engines before #2375 (the inner
// binding overwrote the outer one, or captured a later read of a top-level
// name); the controls lowered correctly already and must stay that way.
const shadowRegressionSource = `module shadow_regress

import std/io (println)

func r() -> int = 7

-- #2375 repro: a block let shadows an outer let.
func blockLet() -> string {
  let r = 1;
  let s = { let r = 2; r * 10 };
  "${show(r)} ${show(s)}"
}

-- A block statement whose let shadows: the later read sees the outer.
func blockStmt() -> int {
  let r = 1;
  let s = { let r = 2; r };
  r * 10 + s
}

-- A closure built after the inner block captures the outer binding.
func captureAfter() -> int {
  let r = 1;
  let s = { let r = 2; r * 10 };
  let f = \x. x + r;
  f(0) * 100 + s
}

-- A closure built inside the block escapes it with the inner binding.
func captureEscape() -> int {
  let r = 1;
  let f = { let r = 2; \x. x + r };
  f(0) * 10 + r
}

-- A lambda body shadows a name it also captures from outside.
func lambdaBodyCapture() -> int {
  let r = 1;
  let f = \x. { let y = { let r = x; r * 2 }; y + r };
  f(5) * 10 + r
}

-- A parameter shadowed by a block let.
func paramShadow(r: int) -> int {
  let s = { let r = r + 100; r * 2 };
  r * 1000 + s
}

-- A tail-recursive loop whose parameter is shadowed in the loop body.
func tailLoop(i: int, acc: int) -> int {
  if i <= 0 then acc else {
    let acc2 = { let i = i * 2; i + acc };
    tailLoop(i - 1, acc2)
  }
}

-- A local letrec shadows a local value of the same name.
func letrecShadow() -> string {
  let f = 5;
  let s = { letrec f = \n. if n <= 0 then 0 else n + f(n - 1); f(4) };
  "${show(f)} ${show(s)}"
}

-- A block let shadows the top-level function r; the later call is the
-- function (the VM called the int: "CALL on non-closure").
func globalShadow() -> int {
  let s = { let r = 2; r * 10 };
  r() * 100 + s
}

-- Three nested blocks, each shadowing the one outside.
func deep3() -> int {
  let r = 1;
  let s = { let r = 2; let t = { let r = 3; let u = { let r = 4; r }; r * 10 + u }; r * 100 + t };
  r * 1000 + s
}

-- Records and strings, not just ints.
func recordShadow() -> int {
  let p = {a: 1, b: 2};
  let s = { let p = {a: 10, b: 20}; p.a + p.b };
  p.a * 100 + s
}

func stringShadow() -> string {
  let r = "outer";
  let s = { let r = "inner"; "${r}!" };
  "${r} ${s}"
}

-- A match scrutinee computed by a shadowing block.
func scrutineeBlock() -> int {
  let r = 1;
  let s = match { let r = Some(9); r } { Some(v) => v, None => 0 };
  s * 10 + r
}

-- Controls: shapes that open a VM scope already.
func seqRebind() -> int {
  let x = 1;
  let x = x + 1;
  let x = x * 10;
  x
}

func patternVar() -> int {
  let r = 1;
  let s = match Some(3) { Some(r) => r * 2, None => 0 };
  r * 10 + s
}

func ifArm(c: bool) -> int {
  let r = 1;
  let s = if c then { let r = 5; r + 1 } else { let r = 7; r + 2 };
  r * 10 + s
}

func lambdaParam() -> int {
  let r = 1;
  let f = \r. r * 3;
  let g = \x. x + r;
  f(5) * 100 + g(0) * 10 + r
}

func tuplePattern() -> int {
  let a = 1;
  let s = match (5, 6) { (a, b) => a + b };
  a * 100 + s
}

func recordPattern() -> int {
  let a = 1;
  let s = match {a: 5, b: 6} { {a, b} => a * b };
  a * 100 + s
}

export func main() -> () ! {IO} {
  println(blockLet());
  println(show(blockStmt()));
  println(show(captureAfter()));
  println(show(captureEscape()));
  println(show(lambdaBodyCapture()));
  println(show(paramShadow(1)));
  println(show(tailLoop(5, 0)));
  println(letrecShadow());
  println(show(globalShadow()));
  println(show(deep3()));
  println(show(recordShadow()));
  println(stringShadow());
  println(show(scrutineeBlock()));
  println(show(seqRebind()));
  println(show(patternVar()));
  println(show(ifArm(true)));
  println(show(lambdaParam()));
  println(show(tuplePattern()));
  println(show(recordPattern()))
}
`

// shadowRegressionWant is what lexical scope prints; the evaluator printed it
// before #2375 too. The comment after each line is what --strict-bytecode
// printed on dev before the fix (measured one function at a time).
var shadowRegressionWant = strings.Join([]string{
	"1 20",         // blockLet: 2 20
	"12",           // blockStmt: 22
	"120",          // captureAfter: 220
	"21",           // captureEscape: 22
	"111",          // lambdaBodyCapture: 151
	"1202",         // paramShadow: 101202
	"30",           // tailLoop: 114
	"5 10",         // letrecShadow: <function> 10
	"720",          // globalShadow: runtime error "CALL on non-closure (Int)"
	"1234",         // deep3: 4444
	"130",          // recordShadow: 1030
	"outer inner!", // stringShadow: inner inner!
	"91",           // scrutineeBlock: runtime error "arith ADD: type mismatch Int vs ADT"
	"20",           // seqRebind (control)
	"16",           // patternVar (control)
	"16",           // ifArm (control)
	"1511",         // lambdaParam (control)
	"111",          // tuplePattern (control)
	"130",          // recordPattern (control)
}, "\n") + "\n"

// TestShadowingRegressionAllEngines pins every shadowing shape that gave a
// wrong answer on the VM before #2375, plus controls, on all three engines.
func TestShadowingRegressionAllEngines(t *testing.T) {
	file := filepath.Join(t.TempDir(), "shadow_regress.ail")
	for _, engine := range allEngines {
		t.Run(engine, func(t *testing.T) {
			if got := runOnEngine(t, file, shadowRegressionSource, engine); got != shadowRegressionWant {
				t.Fatalf("stdout:\n%s\nwant:\n%s", got, shadowRegressionWant)
			}
		})
	}
}
