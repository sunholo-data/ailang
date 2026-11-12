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
// defining module were always rejected.
const exportedConstraintLib = `module lib/boxes
export type Box[a] = Full(a) | Empty
export pure func add[a](x: a, y: a) -> a = x + y
export pure func lt[a](x: a, y: a) -> bool = x < y
export pure func eqq[a](x: a, y: a) -> bool = x == y
export pure func addU(x, y) = x + y
`

func writeExportedConstraintCase(t *testing.T, main string) (string, func(args ...string) (string, string, int)) {
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
	return dir, func(args ...string) (string, string, int) {
		return testutil.RunBounded(t, dir, 60*time.Second, bin, args...)
	}
}

func TestExportedConstraintsCheckedAtImporter(t *testing.T) {
	cases := []struct{ name, call, want string }{
		{"add on an ADT", "add(Full(1), Full(2))", "No instance for Num[Box[int]]"},
		{"lt on lambdas", `lt(\u. u + 1, \u. u + 2)`, "No instance for Ord[int -> int]"},
		{"eqq on lambdas", `eqq(\u. u + 1, \u. u + 2)`, "No instance for Eq[int -> int]"},
		{"unannotated addU on Option", "addU(Some(1), Some(2))", "No instance for Num[Option[int]]"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			main := "module main\nimport lib/boxes (Box, Full, add, lt, eqq, addU)\n" +
				"export func main() -> () ! {IO} = { let r = " + c.call + "; println(\"done\") }\n"
			_, run := writeExportedConstraintCase(t, main)
			stdout, stderr, code := run("check", "main.ail")
			if code == 0 {
				t.Fatalf("check must reject %s across the module boundary, passed:\n%s", c.call, stdout)
			}
			if out := stdout + stderr; !strings.Contains(out, c.want) {
				t.Errorf("want %q, got:\n%s", c.want, out)
			}
		})
	}
}

// Controls: valid cross-module uses still check and run on both engines,
// and `ailang iface` shows the constraint.
func TestExportedConstraintsControls(t *testing.T) {
	main := `module main
import lib/boxes (add, lt, eqq, addU)
export func main() -> () ! {IO} = {
  println(show(add(1, 2)));
  println(show(add(1.5, 2.0)));
  println(show(lt("a", "b")));
  println(show(eqq([1], [1])));
  println(show(eqq(None, None)));
  println(show(addU(2, 3)))
}
`
	_, run := writeExportedConstraintCase(t, main)
	if stdout, stderr, code := run("check", "main.ail"); code != 0 {
		t.Fatalf("controls must check:\n%s%s", stdout, stderr)
	}
	want := "3\n3.5\ntrue\ntrue\ntrue\n5"
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
	if !strings.Contains(stdout, `"Eq[a]"`) || !strings.Contains(stdout, `"Ord[a]"`) {
		t.Errorf("iface must list the exported constraints, got:\n%s", stdout)
	}
}
