package vm_test

import (
	"strings"
	"testing"
)

// M-PERSISTENT-COLLECTIONS 1c: std/array.accumulate under the evaluator and
// the strict VM. Writes fold in list order with one copy; an out-of-bounds
// index is loud in both engines.

const accumulateSource = `module acc
import std/array as A

export func main() -> string {
  let d = A.accumulate(A.make(4, 0), [(0, 1), (2, 5), (0, 2)], \a b. a + b);
  let s = A.accumulate(A.make(3, ""), [(1, "a"), (1, "b")], \a b. "${a}${b}");
  let f = A.accumulate(A.make(2, 0.5), [(1, 2.0)], \a b. a * b);
  let m = A.accumulate(A.make(2, 0), [(0, 3), (0, 1), (1, 7)], \a b. if a > b then a else b);
  let order = A.accumulate(A.make(1, 0), [(0, 1), (0, 2), (0, 3)], \a b. a * 10 + b);
  let none = A.accumulate(A.make(2, 9), [], \a b. a + b);
  let src = A.make(2, 1);
  let changed = A.accumulate(src, [(0, 5)], \a b. a + b);
  let diff = A.accumulate(A.make(6, 0), [(1, 1), (4, -1), (2, 1), (5, -1)], \a b. a + b);
  let prefix = A.toList(A.fromList(diff1(A.toList(diff), 0)));
  "${show(d)} ${show(s)} ${show(f)} ${show(m)} ${show(order)} ${show(none)} ${show(src)} ${show(changed)} ${show(prefix)}"
}

func diff1(xs: [int], run: int) -> [int] =
  match xs {
    [] => [],
    x :: rest => (run + x) :: diff1(rest, run + x)
  }
`

func TestArrayAccumulateBothEngines(t *testing.T) {
	want := `#[3, 0, 5, 0] #[, ab, ] #[0.5, 1.0] #[3, 7] #[123] #[9, 9] #[1, 1] #[6, 1] [0, 1, 2, 2, 1, 0]`
	for run := 0; run < 2; run++ {
		e, eerr, v, verr := runBothEngines(t, "acc", accumulateSource)
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

func TestArrayAccumulateOutOfBoundsIsLoud(t *testing.T) {
	for _, idx := range []string{"3", "-1"} {
		src := `module acc_oob
import std/array as A
export func main() -> int { A.length(A.accumulate(A.make(3, 0), [(0, 1), (` + idx + `, 1)], \a b. a + b)) }
`
		_, eerr, _, verr := runBothEngines(t, "acc_oob", src)
		for engine, err := range map[string]error{"eval": eerr, "vm": verr} {
			if err == nil || !strings.Contains(err.Error(), "accumulate: index "+idx+" out of bounds (array length: 3)") {
				t.Fatalf("index %s, %s: want out-of-bounds error, got %v", idx, engine, err)
			}
		}
	}
}
