package vm_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// M-PERSISTENT-COLLECTIONS 1a: structured Map keys, run end to end through
// the evaluator and the strict VM (no bridge fallback), which must agree.

func runBothEngines(t *testing.T, name, source string) (evalOut string, evalErr error, vmOut string, vmErr error) {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)
	filename := filepath.Join(t.TempDir(), name+".ail")
	if err := os.WriteFile(filename, []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	evalOut, evalErr = runMapModule(t, filename, source, false)
	vmOut, vmErr = runMapModule(t, filename, source, true)
	return strings.TrimSpace(evalOut), evalErr, strings.TrimSpace(vmOut), vmErr
}

const structuredKeysSource = `module structured_keys
import std/map (empty, insert, lookup, member, remove, keys, fromList, toList, size)
import std/option (Some, None)

type Dir = North | East | South | West

export func main() -> string {
  let t = fromList([((2, 1), "a"), ((10, 0), "b"), ((1, 2), "c"), ((1, 2), "d")]);
  let r = insert(insert(empty(), {x: 1, y: 2}, 1), {y: 1, x: 2}, 2);
  let d = fromList([(South, 1), (North, 2), (West, 3), (East, 4)]);
  let nested = fromList([((1, North, "s"), true), ((1, East, "s"), false)]);
  let f = fromList([(2.5, "x"), (-1.0, "y"), (0.0, "z"), (-0.0, "w")]);
  let u = insert(empty(), (), 7);
  let o = fromList([(Some(3), "s"), (None, "n")]);
  let lex = keys(fromList([(9, "a"), (10, "b"), (2, "c")]));
  "${show(keys(t))} | ${show(lookup(t, (1, 2)))} | ${show(member(t, (2, 2)))} | ${show(size(remove(t, (10, 0))))} | ${show(lookup(r, {y: 2, x: 1}))} | ${show(keys(d))} | ${show(toList(nested))} | ${show(toList(f))} | ${show(lookup(u, ()))} | ${show(keys(o))} | ${show(lex)}"
}
`

func TestStructuredMapKeysBothEngines(t *testing.T) {
	// Tuple keys sort element-wise with lexical ints (the D2-preserved int
	// order, also shown by the last field: V10's [10, 2, 9]); constructors sort
	// by name; floats numerically, with -0.0 and 0.0 one key.
	want := `[(1, 2), (10, 0), (2, 1)] | Some(d) | false | 2 | Some(1) | [East, North, South, West] | [((1, East, s), false), ((1, North, s), true)] | [(-1.0, y), (-0.0, w), (2.5, x)] | Some(7) | [None, Some(3)] | [10, 2, 9]`
	var first string
	for run := 0; run < 2; run++ {
		e, eerr, v, verr := runBothEngines(t, "structured_keys", structuredKeysSource)
		if eerr != nil || verr != nil {
			t.Fatalf("run %d: eval err %v, vm err %v", run, eerr, verr)
		}
		if e != want {
			t.Fatalf("evaluator:\n got %s\nwant %s", e, want)
		}
		if v != e {
			t.Fatalf("engines differ:\n eval %s\n   vm %s", e, v)
		}
		if run == 0 {
			first = e
		} else if e != first {
			t.Fatalf("non-deterministic across runs:\n%s\n%s", first, e)
		}
	}
}

// V9: before 1a, member on a tuple-keyed map printed false because the key
// could not be encoded. It still prints false, now because the key is absent.
func TestTupleKeyAbsentIsGenuinelyFalse(t *testing.T) {
	src := `module silent
import std/map (empty, member)
func e() -> Map[(int, int), bool] { empty() }
export func main() -> bool { member(e(), (1, 2)) }
`
	e, eerr, v, verr := runBothEngines(t, "silent", src)
	if eerr != nil || verr != nil || e != "false" || v != "false" {
		t.Fatalf("eval %q %v, vm %q %v", e, eerr, v, verr)
	}
}

// V9 inverted: a key that can never be encoded is a loud error on every
// operation (lookup, member, remove, insert), naming the type, in both engines.
func TestUnencodableKeyIsLoudOnEveryOperation(t *testing.T) {
	for _, op := range []struct{ name, call string }{
		{"member", "show(member(e(), \\x. x + 1))"},
		{"lookup", "match lookup(e(), \\x. x + 1) { Some(_) => \"some\", None => \"none\" }"},
		{"remove", "show(size(remove(e(), \\x. x + 1)))"},
		{"insert", "show(size(insert(e(), \\x. x + 1, true)))"},
	} {
		t.Run(op.name, func(t *testing.T) {
			src := fmt.Sprintf(`module fnkey
import std/map (empty, member, lookup, remove, insert, size)
import std/option (Some, None)
func e() -> Map[int -> int, bool] { empty() }
export func main() -> string { %s }
`, op.call)
			e, eerr, v, verr := runBothEngines(t, "fnkey", src)
			for engine, err := range map[string]error{"eval": eerr, "vm": verr} {
				if err == nil || !strings.Contains(err.Error(), "unsupported map key type: function") {
					t.Fatalf("%s: want loud function-key error, got output %q / %q, err %v", engine, e, v, err)
				}
			}
		})
	}
}

func TestNaNKeyIsLoudBothEngines(t *testing.T) {
	src := `module nankey
import std/map (empty, insert, size)
export func main() -> int { let nan = 0.0 / 0.0; size(insert(empty(), (1, nan), true)) }
`
	_, eerr, _, verr := runBothEngines(t, "nankey", src)
	for engine, err := range map[string]error{"eval": eerr, "vm": verr} {
		if err == nil || !strings.Contains(err.Error(), "NaN") {
			t.Fatalf("%s: want NaN key error, got %v", engine, err)
		}
	}
}
