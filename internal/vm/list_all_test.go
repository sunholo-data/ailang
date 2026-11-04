package vm_test

import "testing"

func TestListAllBothEngines(t *testing.T) {
	source := `module list_all
import std/list (all, range)
export func main() -> string {
 let empty = all(\x. x > 0, []);
 let yes = all(\x. x > 0, [1, 2, 3]);
 let no = all(\x. x > 0, [1, 0, 3]);
 let short = all(\x. 10 / x > 0, [-1, 0]);
 let large = all(\x. x >= 0, range(0, 10000));
 "${show(empty)} ${show(yes)} ${show(no)} ${show(short)} ${show(large)}"
}`
	e, ee, v, ve := runBothEngines(t, "list_all", source)
	if ee != nil || ve != nil {
		t.Fatalf("eval %v; VM %v", ee, ve)
	}
	if e != "true true false false true" || v != e {
		t.Fatalf("eval %q VM %q", e, v)
	}
}
