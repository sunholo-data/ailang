package vm_test

import "testing"

// #2244: matching a value whose type is an alias of an ADT runs on the
// evaluator and on the strict VM, which agree. The VM dispatches on the
// pattern's ADT (Color), never on the alias the value's type is written with
// (Hue), which declares no tags: before, such a function failed to compile
// for the VM ("unknown ADT Hue in switch").
const aliasMatchSource = `module alias_match
import std/option (Option, Some, None)

type Color = Red | Green | Blue
type Hue = Color
type Tint = Hue
type IntOpt = Option[int]
type Opt[a] = Option[a]

pure func tint(t: Tint) -> int = match t { Red => 1, Green => 2, Blue => 3 }

pure func intOpt(o: IntOpt) -> int = match o { Some(n) => n + 1, None => 0 }

pure func opt[a](o: Opt[a], d: a) -> a = match o { Some(x) => x, None => d }

pure func nested(o: Opt[Hue]) -> int = match o {
  Some(Red) => 10,
  Some(Green) => 20,
  Some(Blue) => 30,
  None => 0
}

pure func guarded(h: Hue, n: int) -> int = match h {
  Red if n > 0 => 100,
  Green => 200,
  _ => 300
}

pure func lambdas(h: Hue) -> int {
  let f = func(x: Hue) -> int { match x { Red => 1000, Green => 2000, Blue => 3000 } };
  let g = \x. match x { Red => 1, Green => 2, Blue => 3 };
  f(h) + g(h)
}

export func main() -> string =
  "${show(tint(Blue))} ${show(intOpt(Some(41)))} ${opt(Some("x"), "d")} ${show(opt(None, 7))} ${show(nested(Some(Green)))} ${show(guarded(Red, 0))} ${show(guarded(Red, 1))} ${show(lambdas(Blue))}"
`

func TestMatchOnADTAliasBothEngines(t *testing.T) {
	const want = "3 42 x 7 20 300 100 3003"
	e, eerr, v, verr := runBothEngines(t, "alias_match", aliasMatchSource)
	if eerr != nil || verr != nil {
		t.Fatalf("eval err %v, vm err %v", eerr, verr)
	}
	if e != want {
		t.Fatalf("evaluator: got %q, want %q", e, want)
	}
	if v != e {
		t.Fatalf("engines differ: eval %q, vm %q", e, v)
	}
}
