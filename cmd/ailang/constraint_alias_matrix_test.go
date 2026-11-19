package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/testutil"
)

func runInDir(t *testing.T, dir, bin string, args ...string) (string, string, int) {
	t.Helper()
	return testutil.RunBounded(t, dir, 60*time.Second, bin, args...)
}

// #2285 third review: an operator on a value whose type is an alias must run
// the same implementation as on the type the alias stands for. The `+` node
// keeps its written type (Id[b]), and lowering picks an implementation from
// it, so a constraint may be judged through an alias only when that resolves
// it outright (constraint_alias.go). Each function below is written once with
// an alias and once without; both programs must check and print the same on
// the evaluator and on --strict-bytecode.
type aliasMatrixCase struct {
	name, aliased, plain, call string
}

func aliasMatrixCases() []aliasMatrixCase {
	ops := []struct{ name, expr, kind string }{
		{"add", "x + y", "T"}, {"sub", "x - y", "T"}, {"mul", "x * y", "T"}, {"div", "x / y", "T"},
		{"neg", "-x", "T"}, {"lt", "x < y", "bool"}, {"eq", "x == y", "bool"}, {"show", "show(x)", "string"},
	}
	var cs []aliasMatrixCase
	for _, op := range ops {
		ret := func(t string) string {
			switch op.kind {
			case "bool":
				return "bool"
			case "string":
				return "string"
			}
			return t
		}
		add := func(form, aParams, pParams, aRet, pRet, aBody, pBody, call string) {
			n := fmt.Sprintf("%s_%s", form, op.name)
			cs = append(cs, aliasMatrixCase{
				name:    n,
				aliased: fmt.Sprintf("pure func %s(%s) -> %s = %s", n, aParams, aRet, aBody),
				plain:   fmt.Sprintf("pure func %s(%s) -> %s = %s", n, pParams, pRet, pBody),
				call:    fmt.Sprintf("%s(%s)", n, call),
			})
		}
		// Identity alias at a ground type, nested, and imported from another module.
		add("id", "x: Id[float], y: Id[float]", "x: float, y: float", ret("Id[float]"), ret("float"), op.expr, op.expr, "7.5, 2.5")
		add("nested", "x: Id[Id[float]], y: Id[Id[float]]", "x: float, y: float", ret("Id[Id[float]]"), ret("float"), op.expr, op.expr, "7.5, 2.5")
		add("xmod", "x: XId[int], y: XId[int]", "x: int, y: int", ret("XId[int]"), ret("int"), op.expr, op.expr, "7, 3")
		// Phantom parameter, polymorphic in it.
		add("tag", "x: Tag[b], y: Tag[b]", "x: int, y: int", ret("Tag[b]"), ret("int"), op.expr, op.expr, "7, 3")
		// Element use through a tuple alias and a list alias.
		pairBody := "match p { (x, y) => " + op.expr + " }"
		add("pair", "p: Pair[float]", "p: (float, float)", ret("float"), ret("float"), pairBody, pairBody, "(7.5, 2.5)")
		def := map[string]string{"T": "0.5", "bool": "false", "string": `""`}[op.kind]
		listBody := "match xs { [x, y] => " + op.expr + ", _ => d }"
		add("list", "xs: F[float], d: "+ret("float"), "xs: [float], d: "+ret("float"), ret("float"), ret("float"), listBody, listBody, "[7.5, 2.5], "+def)
	}
	// The tag functions are generic in b; mark them so.
	for i := range cs {
		if strings.HasPrefix(cs[i].name, "tag_") {
			cs[i].aliased = strings.Replace(cs[i].aliased, "(", "[b](", 1)
			cs[i].plain = strings.Replace(cs[i].plain, "(", "[b](", 1)
		}
	}
	return cs
}

func TestConstraintAliasOperatorMatrix(t *testing.T) {
	bin := buildAilang(t)
	cs := aliasMatrixCases()
	var aliased, plain strings.Builder
	aliased.WriteString("module main\nimport lib/al (XId)\ntype Id[a] = a\ntype Tag[t] = int\ntype Pair[a] = (a, a)\ntype F[a] = [a]\n")
	plain.WriteString("module main\n")
	for _, c := range cs {
		aliased.WriteString(c.aliased + "\n")
		plain.WriteString(c.plain + "\n")
	}
	// One printing function per 8 cases: a single main this size exceeds the
	// VM's register window and is refused under --strict-bytecode.
	var body, mainCalls strings.Builder
	for i := 0; i < len(cs); i += 8 {
		var lines []string
		for _, c := range cs[i:min(i+8, len(cs))] {
			lines = append(lines, fmt.Sprintf("  println(\"%s\");\n  println(show(%s))", c.name, c.call))
		}
		fmt.Fprintf(&body, "func p%d() -> () ! {IO} = {\n%s\n}\n", i/8, strings.Join(lines, ";\n"))
		fmt.Fprintf(&mainCalls, "  p%d();\n", i/8)
	}
	body.WriteString("export func main() -> () ! {IO} = {\n" + strings.TrimSuffix(mainCalls.String(), ";\n") + "\n}\n")
	outputs := map[string]string{}
	for side, src := range map[string]string{"aliased": aliased.String() + body.String(), "plain": plain.String() + body.String()} {
		dir := t.TempDir()
		if err := os.MkdirAll(filepath.Join(dir, "lib"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "lib", "al.ail"), []byte("module lib/al\nexport type XId[a] = a\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "main.ail"), []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
		run := func(args ...string) (string, string, int) {
			return runInDir(t, dir, bin, args...)
		}
		if stdout, stderr, code := run("check", "main.ail"); code != 0 {
			t.Fatalf("%s program must check:\n%s%s\n%s", side, stdout, stderr, src)
		}
		for _, engine := range [][]string{
			{"run", "--quiet", "--caps", "IO", "main.ail"},
			{"run", "--quiet", "--caps", "IO", "--bytecode", "--strict-bytecode", "main.ail"},
		} {
			stdout, stderr, code := run(engine...)
			if code != 0 {
				t.Fatalf("%s %v: exit %d\n%s", side, engine, code, stderr)
			}
			outputs[side+" "+strings.Join(engine, " ")] = strings.TrimSpace(stdout)
		}
	}
	want := outputs["plain run --quiet --caps IO main.ail"]
	if want == "" || strings.Count(want, "\n")+1 != 2*len(cs) {
		t.Fatalf("plain program printed %d lines for %d cases:\n%s", strings.Count(want, "\n")+1, len(cs), want)
	}
	for k, got := range outputs {
		if got != want {
			t.Errorf("%s differs from the alias-free program:\n got:\n%s\nwant:\n%s", k, got, want)
		}
	}
}

// Rejected, as on dev: an operator on an alias of a type variable. Judging
// Num[Id[b]] as Num[b] would defer it while lowering still picks the Int
// implementation from Id[b] (dbl(1.5) crashed on both engines).
func TestConstraintAliasOfVariableRejected(t *testing.T) {
	bin := buildAilang(t)
	for _, c := range []struct{ name, expr, ret, class string }{
		{"add", "x + y", "Id[b]", "Num"}, {"div", "x / y", "Id[b]", "Num"}, {"neg", "-x", "Id[b]", "Num"},
		{"lt", "x < y", "bool", "Ord"}, {"eq", "x == y", "bool", "Eq"},
	} {
		t.Run(c.name, func(t *testing.T) {
			src := fmt.Sprintf("module main\ntype Id[a] = a\npure func f[b](x: Id[b], y: Id[b]) -> %s = %s\nexport func main() -> () ! {IO} = println(show(f(1.5, 2.5)))\n", c.ret, c.expr)
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, "main.ail"), []byte(src), 0o644); err != nil {
				t.Fatal(err)
			}
			stdout, stderr, code := runInDir(t, dir, bin, "check", "main.ail")
			if code == 0 {
				t.Fatalf("an operator on an alias of a type variable must be rejected:\n%s", stdout)
			}
			if want := "No instance for " + c.class + "[Id[b]]"; !strings.Contains(stdout+stderr, want) {
				t.Errorf("want %q, got:\n%s%s", want, stdout, stderr)
			}
		})
	}
}

// #2285 fourth review: a branching self-referential alias made `check` use
// memory without bound (the expansion re-expanded the alias at every
// occurrence to a fixed depth: 65 GB before it was killed). Each program must
// be decided within a hard time cap; the process is killed at the cap.
func TestConstraintAliasExpansionIsBounded(t *testing.T) {
	bin := buildAilang(t)
	var pairs strings.Builder
	pairs.WriteString("type P1 = (int, int)\n")
	for i := 2; i <= 20; i++ {
		fmt.Fprintf(&pairs, "type P%d = (P%d, P%d)\n", i, i-1, i-1)
	}
	cases := []struct{ name, src, want string }{
		{"branching_self_eq", "type T2[a] = (T2[a], T2[a])\npure func f(x: T2[int]) -> bool = x == x\n", "TC_TYPE_ALIAS_CYCLE_001: type alias 'T2' refers to itself"},
		{"branching_self_num", "type T2[a] = (T2[a], T2[a])\npure func f(x: T2[int]) -> T2[int] = x + x\n", "TC_TYPE_ALIAS_CYCLE_001: type alias 'T2' refers to itself"},
		{"branching_self_option", "type T3[a] = Option[(T3[a], T3[a], T3[a])]\npure func f(x: T3[int]) -> bool = x == x\n", "TC_TYPE_ALIAS_CYCLE_001: type alias 'T3' refers to itself"},
		// 2^20 leaves legitimately: past the expansion budget it is judged as
		// written, as on dev.
		{"pair_nest_20", pairs.String() + "pure func f(x: P20, y: P20) -> bool = x == y\n", "No instance for Eq[P20]"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir := t.TempDir()
			src := "module main\n" + c.src + "export func main() -> () ! {IO} = println(\"hi\")\n"
			if err := os.WriteFile(filepath.Join(dir, "main.ail"), []byte(src), 0o644); err != nil {
				t.Fatal(err)
			}
			start := time.Now()
			stdout, stderr, code := testutil.RunBounded(t, dir, 5*time.Second, bin, "check", "main.ail")
			if code == -1 {
				t.Fatalf("check did not finish within 5s (killed after %s)", time.Since(start))
			}
			if code == 0 {
				t.Fatalf("check must reject:\n%s", stdout)
			}
			if !strings.Contains(stdout+stderr, c.want) {
				t.Errorf("want %q, got:\n%s%s", c.want, stdout, stderr)
			}
		})
	}
}

// A small pair nest is expanded and compared structurally on both engines.
func TestConstraintAliasPairNestRuns(t *testing.T) {
	bin := buildAilang(t)
	src := `module main
type P1 = (int, int)
type P2 = (P1, P1)
type P3 = (P2, P2)
pure func same(x: P3, y: P3) -> bool = x == y
export func main() -> () ! {IO} = {
  let a = (((1, 2), (3, 4)), ((5, 6), (7, 8)));
  let b = (((1, 2), (3, 4)), ((5, 6), (7, 9)));
  println(show(same(a, a)));
  println(show(same(a, b)))
}
`
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.ail"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	run := func(args ...string) (string, string, int) { return runInDir(t, dir, bin, args...) }
	if stdout, stderr, code := run("check", "main.ail"); code != 0 {
		t.Fatalf("pair nest must check:\n%s%s", stdout, stderr)
	}
	assertBothEngines(t, "true\nfalse", "main.ail", run)
}

// Self-referential and mutually recursive RECORD declarations are legal and
// common (`type Node = {label: string, kids: [Node]}`); a record alias is
// never expanded for a class constraint, so each must check as on dev, in
// bounded time, and print the same on both engines (#2285 fourth review).
// Direct `==` on such a record is left out of the run: with `deriving (Eq)`
// it crashes the evaluator on dev too (#2318).
func TestConstraintSelfReferentialRecords(t *testing.T) {
	bin := buildAilang(t)
	shapes := []struct{ name, decl, rec, v1, v2, want string }{
		{"node", "type Node = {n: int, label: string, kids: [Node]} deriving (Eq)", "Node",
			`{n: 1, label: "a", kids: []}`, `{n: 2, label: "b", kids: [{n: 3, label: "c", kids: []}]}`, "true\nfalse\ntrue\ntrue\ntrue\n2\n{kids: [], label: \"a\", n: 1}"},
		{"person", "type Person = {n: int, name: string, friends: [Person]} deriving (Eq)", "Person",
			`{n: 1, name: "p", friends: []}`, `{n: 2, name: "q", friends: []}`, "true\nfalse\ntrue\ntrue\ntrue\n2\n{friends: [], n: 1, name: \"p\"}"},
		{"chain", "type N = {n: int, next: Option[N]} deriving (Eq)", "N",
			`{n: 1, next: None}`, `{n: 2, next: Some({n: 3, next: None})}`, "true\nfalse\ntrue\ntrue\ntrue\n2\n{n: 1, next: None}"},
		{"account", "type AccountState = {n: int, owner: string, snapshots: [AccountState]} deriving (Eq)", "AccountState",
			`{n: 1, owner: "x", snapshots: []}`, `{n: 1, owner: "x", snapshots: [{n: 0, owner: "x", snapshots: []}]}`, "true\nfalse\ntrue\ntrue\ntrue\n0\n{n: 1, owner: \"x\", snapshots: []}"},
		{"mutual", "type RA = {n: int, b: Option[RB]} deriving (Eq)\ntype RB = {m: int, ras: [RA]} deriving (Eq)", "RA",
			`{n: 1, b: None}`, `{n: 2, b: Some({m: 1, ras: []})}`, "true\nfalse\ntrue\ntrue\ntrue\n2\n{b: None, n: 1}"},
	}
	for _, sh := range shapes {
		t.Run(sh.name, func(t *testing.T) {
			r, a, b := sh.rec, sh.v1, sh.v2
			src := fmt.Sprintf(`module main
%s
type Box[a] = (a, a)
pure func eqL(x: [%[2]s], y: [%[2]s]) -> bool = x == y
pure func neO(x: Option[%[2]s], y: Option[%[2]s]) -> bool = x != y
pure func eqT(x: (%[2]s, int), y: (%[2]s, int)) -> bool = x == y
pure func eqB(x: Box[%[2]s], y: Box[%[2]s]) -> bool = x == y
pure func field(x: %[2]s, y: %[2]s) -> int = if x.n < y.n then x.n + 1 else y.n - 1
pure func sameD(x: %[2]s, y: %[2]s) -> bool = x == y
export func main() -> () ! {IO} = {
  println(show(eqL([%[3]s], [%[3]s])));
  println(show(eqL([%[3]s], [%[4]s])));
  println(show(neO(Some(%[3]s), Some(%[4]s))));
  println(show(eqT((%[3]s, 1), (%[3]s, 1))));
  println(show(eqB((%[3]s, %[4]s), (%[3]s, %[4]s))));
  println(show(field(%[3]s, %[4]s)));
  println(show(%[3]s))
}
`, sh.decl, r, a, b)
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, "main.ail"), []byte(src), 0o644); err != nil {
				t.Fatal(err)
			}
			stdout, stderr, code := testutil.RunBounded(t, dir, 5*time.Second, bin, "check", "main.ail")
			if code == -1 {
				t.Fatal("check did not finish within 5s")
			}
			if code != 0 {
				t.Fatalf("must check:\n%s%s", stdout, stderr)
			}
			// Without deriving (Eq) the record has no ==, directly or in a container.
			plain := strings.ReplaceAll(src, " deriving (Eq)", "")
			if err := os.WriteFile(filepath.Join(dir, "plain.ail"), []byte(strings.Replace(plain, "module main", "module plain", 1)), 0o644); err != nil {
				t.Fatal(err)
			}
			stdout, stderr, code = testutil.RunBounded(t, dir, 5*time.Second, bin, "check", "plain.ail")
			if code == -1 {
				t.Fatal("check (no deriving) did not finish within 5s")
			}
			if code == 0 || !strings.Contains(stdout+stderr, "No instance for Eq[") {
				t.Fatalf("without deriving (Eq) == must be rejected, got exit %d:\n%s%s", code, stdout, stderr)
			}
			run := func(args ...string) (string, string, int) {
				return testutil.RunBounded(t, dir, 10*time.Second, bin, args...)
			}
			assertBothEngines(t, sh.want, "main.ail", run)
		})
	}
}
