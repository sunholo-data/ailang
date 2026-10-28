package vm_test

import (
	"strings"
	"testing"
)

// M-PERSISTENT-COLLECTIONS 1b: every std/set operation under the evaluator
// and the strict VM, which must agree, run twice for determinism.

const setOpsSource = `module set_ops
import std/set as S
import std/set (Set)

type Dir = North | East | South | West

export func main() -> string {
  let a = S.fromList([3, 1, 2, 3, 10]);
  let b = S.insert(S.insert(S.empty(), 2), 99);
  let v = S.insert(S.insert(S.insert(S.empty(), (1, 2, North)), (1, 2, East)), (1, 2, North));
  let r = S.fromList([{x: 1, y: 2}, {y: 2, x: 1}, {x: 0, y: 5}]);
  let e: Set[int] = S.empty();
  "${show(S.toList(a))} ${show(S.size(a))} ${show(S.member(a, 10))} ${show(S.member(a, 4))} ${show(S.toList(S.remove(a, 3)))} ${show(S.size(S.remove(a, 42)))} ${show(S.toList(S.union(a, b)))} ${show(S.toList(S.intersect(a, b)))} ${show(S.toList(S.difference(a, b)))} ${show(S.toList(v))} ${show(S.size(v))} ${show(S.member(v, (1, 2, South)))} ${show(S.size(r))} ${show(S.member(r, {y: 5, x: 0}))} ${show(S.size(e))} ${show(S.toList(e))}"
}
`

func TestSetOperationsBothEngines(t *testing.T) {
	// toList is canonical key order: lexical for top-level ints (D2), and
	// element-wise for tuples with constructors by name.
	want := `[1, 10, 2, 3] 4 true false [1, 10, 2] 4 [1, 10, 2, 3, 99] [2] [1, 10, 3] [(1, 2, East), (1, 2, North)] 2 false 2 true 0 []`
	for run := 0; run < 2; run++ {
		e, eerr, v, verr := runBothEngines(t, "set_ops", setOpsSource)
		if eerr != nil || verr != nil {
			t.Fatalf("run %d: eval err %v, vm err %v", run, eerr, verr)
		}
		if e != want {
			t.Fatalf("evaluator:\n got %s\nwant %s", e, want)
		}
		if v != e {
			t.Fatalf("engines differ:\n eval %s\n   vm %s", e, v)
		}
	}
}

// A set operation on an element that can never be a key is loud in both
// engines, member included.
func TestSetUnencodableElementIsLoud(t *testing.T) {
	src := `module set_fn
import std/set as S
import std/set (Set)
func e() -> Set[int -> int] { S.empty() }
export func main() -> bool { S.member(e(), \x. x) }
`
	_, eerr, _, verr := runBothEngines(t, "set_fn", src)
	for engine, err := range map[string]error{"eval": eerr, "vm": verr} {
		if err == nil || !strings.Contains(err.Error(), "unsupported map key type: function") {
			t.Fatalf("%s: want loud function-element error, got %v", engine, err)
		}
	}
}
