package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/testutil"
)

// #2290: an exported function's class constraints were dropped at the module
// boundary, so a caller in another module could instantiate them at a type
// with no instance. `check` passed and the call crashed at run time; on the
// VM `eqq(f, g)` on two lambdas printed false. The same calls inside the
// defining module were always rejected. The type-level cases are in
// internal/pipeline/exported_constraints_test.go; these run the binary.
const exportedConstraintLib = `module lib/boxes
export type Box[a] = Full(a) | Empty
export pure func add[a](x: a, y: a) -> a = x + y
export pure func lt[a](x: a, y: a) -> bool = x < y
export pure func eqq[a](x: a, y: a) -> bool = x == y
export pure func addU(x, y) = x + y
export pure func addv(p, q) = {p | v: p.v + q.v}
`

func writeExportedConstraintCase(t *testing.T, main string) func(args ...string) (string, string, int) {
	t.Helper()
	bin := buildAilang(t)
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "lib"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "lib", "boxes.ail"), []byte(exportedConstraintLib), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.ail"), []byte(main), 0o644); err != nil {
		t.Fatal(err)
	}
	return func(args ...string) (string, string, int) {
		return testutil.RunBounded(t, dir, 60*time.Second, bin, args...)
	}
}

// The rejection reaches `check --json` with its code.
func TestExportedConstraintsCheckJSONCode(t *testing.T) {
	run := writeExportedConstraintCase(t, "module main\nimport lib/boxes (eqq)\n"+
		"export func main() -> () ! {IO} = println(show(eqq(\\u. u + 1, \\u. u + 2)))\n")
	stdout, stderr, code := run("check", "--json", "main.ail")
	if code == 0 {
		t.Fatalf("check must reject eqq on lambdas across the module boundary:\n%s", stdout)
	}
	out := stdout + stderr
	if !strings.Contains(out, `"code": "TC_CONSTRAINT_001"`) || !strings.Contains(out, "`eqq` from lib/boxes requires Eq") {
		t.Errorf("want TC_CONSTRAINT_001 naming eqq and lib/boxes, got:\n%s", out)
	}
}

// Controls: valid cross-module uses, the #2312 programs and std/list's
// Eq-constrained functions (#2291) check and run the same on both engines,
// and `ailang iface` shows the constraints.
func TestExportedConstraintsControlsRun(t *testing.T) {
	main := `module main
import lib/boxes (add, lt, eqq, addU, addv)
import std/list (contains, dedup)
export func main() -> () ! {IO} = {
  println(show(add(1, 2)));
  println(show(add(1.5, 2.0)));
  println(show(lt("a", "b")));
  println(show(eqq([1], [1])));
  println(show(eqq(None, None)));
  println(show(addU(2, 3)));
  println(show(addv({v: 1.5, z: 1}, {v: 2.0}).v));
  let f = \p. {p | x: p.x + 1};
  println(show(f({x: 1.5, z: 1}).x));
  println(show(contains([(1, "a")], (1, "a"))));
  println(show(dedup(["a", "b", "a"])))
}
`
	run := writeExportedConstraintCase(t, main)
	if stdout, stderr, code := run("check", "main.ail"); code != 0 {
		t.Fatalf("controls must check:\n%s%s", stdout, stderr)
	}
	want := "3\n3.5\ntrue\ntrue\ntrue\n5\n3.5\n2.5\ntrue\n[\"a\", \"b\"]"
	for _, engine := range [][]string{
		{"run", "--quiet", "--caps", "IO", "main.ail"},
		{"run", "--quiet", "--caps", "IO", "--bytecode", "--strict-bytecode", "main.ail"},
	} {
		stdout, stderr, code := run(engine...)
		if code != 0 {
			t.Errorf("%v: exit %d\n%s", engine, code, stderr)
			continue
		}
		if got := strings.TrimSpace(stdout); got != want {
			t.Errorf("%v:\ngot\n%s\nwant\n%s", engine, got, want)
		}
	}
	stdout, stderr, code := run("iface", "lib/boxes.ail")
	if code != 0 {
		t.Fatalf("iface: %s", stderr)
	}
	for _, c := range []string{`"Eq[a]"`, `"Ord[a]"`, `"Num[a]"`, `"Num[b]"`} {
		if !strings.Contains(stdout, c) {
			t.Errorf("iface must list %s, got:\n%s", c, stdout)
		}
	}
}
