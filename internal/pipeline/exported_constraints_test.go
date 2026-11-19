package pipeline

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/iface"
	"github.com/sunholo-data/ailang/internal/types"
)

// #2290: an exported scheme's class constraints were dropped at the module
// boundary (iface builder) and not re-emitted at a cross-module use
// (inferVarGlobal), so a caller in another module could instantiate them at a
// type with no instance: `check` passed and the call failed at run time, or for
// `==` on two functions on the VM answered false. The same calls inside the
// defining module were always rejected.
const xconLib = `module q/lib
import std/option (Option, Some, None)
export type Box[a] = Full(a) | Empty
export type Color = Red | Green deriving (Eq)
export pure func add[a](x: a, y: a) -> a = x + y
export pure func lt[a](x: a, y: a) -> bool = x < y
export pure func eqq[a](x: a, y: a) -> bool = x == y
export pure func addU(x, y) = x + y
export pure func eql[a](xs: [a], ys: [a]) -> bool = xs == ys
export pure func addx(p) = p.x + 1
export pure func addxy(p, q) = p.x + q.y
export pure func getx(p) = p.x
export pure func sumx(p, q) = { let gx = \r. r.x; gx(p) + gx(q) }
export pure func eqx(p, q) = { let gx = \r. r.x; gx(p) == gx(q) }
export pure func addv(p, q) = {p | v: p.v + q.v}
`

func checkXcon(t *testing.T, imports, body string) error {
	t.Helper()
	_, err := writeAndCheck(t, map[string]string{
		"q/lib.ail": xconLib,
		"q/m.ail": "module q/m\nimport q/lib (" + imports + ")\n" +
			"export func main() -> () ! {IO} = " + body + "\n",
	}, "q/m.ail")
	return err
}

func TestExportedConstraints_RejectedAtImporter(t *testing.T) {
	cases := []struct{ name, imports, body, fn, want string }{
		{"add on strings", "add", `println(add("s", "t"))`, "add", "No instance for Num[string]"},
		{"add on an ADT", "Box, Full, add", "println(show(match add(Full(1), Full(2)) { Full(n) => n, _ => 0 }))", "add", "No instance for Num[Box[int]]"},
		{"add on lambdas", "add", `println(show(add(\u. u + 1, \u. u + 2)(1)))`, "add", "No instance for Num[int -> int]"},
		{"lt on lambdas", "lt", `println(show(lt(\u. u + 1, \u. u + 2)))`, "lt", "No instance for Ord[int -> int]"},
		{"lt on Option", "lt", `println(show(lt(None, None)))`, "lt", "No instance for Ord[Option["},
		{"eqq on lambdas", "eqq", `println(show(eqq(\u. u + 1, \u. u + 2)))`, "eqq", "No instance for Eq[int -> int]"},
		{"eql on lists of lambdas", "eql", `println(show(eql([\u. u + 1], [\u. u + 1])))`, "eql", "needs == on int -> int"},
		{"unannotated addU on Option", "addU", "println(show(addU(Some(1), Some(2))))", "addU", "No instance for Num[Option[int]]"},
		// record rows (#2288): the field variable carries the constraint
		{"addx on a string field", "addx", `println(addx({x: "s"}))`, "addx", "No instance for Num[string]"},
		{"addxy on string fields", "addxy", `println(addxy({x: "s"}, {y: "t"}))`, "addxy", "No instance for Num[string]"},
		// let-bound field readers (#2284): decided before export
		{"sumx on strings", "sumx", `println(sumx({x: "a"}, {x: "b"}))`, "sumx", "No instance for Num[string]"},
		{"eqx on lambdas", "eqx", `println(show(eqx({x: \u. u}, {x: \u. u})))`, "eqx", "Functions have no =="},
		// shared defaulting walk (#2312): addv is Num-polymorphic in v
		{"addv on strings", "addv", `println(addv({v: "a"}, {v: "b"}).v)`, "addv", "No instance for Num[string]"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := checkXcon(t, c.imports, c.body)
			wantCode(t, err, types.TC_CONSTRAINT_001, c.want)
			if !strings.Contains(err.Error(), "`"+c.fn+"` from q/lib requires") {
				t.Errorf("the message must name the imported function and its module: %v", err)
			}
		})
	}
}

// Controls: correct cross-module uses keep checking, including an alias of a
// primitive and an alias of an ADT that derives Eq (#2285 sees through them).
func TestExportedConstraints_ControlsCheck(t *testing.T) {
	cases := []struct{ name, imports, body string }{
		{"numeric", "add, addU", "{ println(show(add(1, 2))); println(show(add(1.5, 2.0))); println(show(addU(2, 3))) }"},
		{"ordered", "lt", `{ println(show(lt("a", "b"))); println(show(lt(1.5, 2.5))) }`},
		{"equality", "eqq, eql", `{ println(show(eqq([1], [1]))); println(show(eqq(None, None))); println(show(eqq(Some("a"), None))); println(show(eql([(1, "a")], [(1, "a")]))) }`},
		{"derived Eq", "Color, Red, Green, eqq", "println(show(eqq(Red, Green)))"},
		{"record rows", "addx, addxy, getx, sumx, eqx, addv", `{ println(show(addx({x: 1}))); println(show(addxy({x: 1}, {y: 2}))); println(getx({x: "s"})); println(show(sumx({x: 1.5}, {x: 2.0}))); println(show(eqx({x: "a"}, {x: "a"}))); println(show(addv({v: 1.5, z: 1}, {v: 2.0}).v)) }`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if err := checkXcon(t, c.imports, c.body); err != nil {
				t.Fatalf("must check: %v", err)
			}
		})
	}
	aliases := map[string]string{
		"q/lib.ail": xconLib,
		"q/m.ail": `module q/m
import q/lib (Color, Red, Green, add, eqq, lt)
type Score = int
type C2 = Color
export func main() -> () ! {IO} = {
  let a: Score = 1;
  let b: Score = 2;
  let c: C2 = Red;
  println(show(add(a, b)));
  println(show(lt(a, b)));
  println(show(eqq(c, Green)))
}
`,
	}
	if _, err := writeAndCheck(t, aliases, "q/m.ail"); err != nil {
		t.Fatalf("aliases of int and of an Eq ADT must check: %v", err)
	}
}

// The interface carries each export's constraints, with the type's canonical
// variable names, and only decided ones: a let-bound field reader's (#2284)
// constraint is on the exported field variable.
func TestExportedConstraints_Interface(t *testing.T) {
	res, err := writeAndCheck(t, map[string]string{"q/lib.ail": xconLib}, "q/lib.ail")
	if err != nil {
		t.Fatal(err)
	}
	data, err := res.Interface.ToNormalizedJSON()
	if err != nil {
		t.Fatal(err)
	}
	var rendered struct {
		Funcs []struct {
			Name, Type  string
			Constraints []string
		} `json:"funcs"`
	}
	if err := json.Unmarshal(data, &rendered); err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"add":   "(a,a)->a Num[a]",
		"lt":    "(a,a)->bool Ord[a]",
		"eqq":   "(a,a)->bool Eq[a]",
		"addU":  "(a,a)->a Num[a]",
		"addx":  "({x: b, ...a})->b Num[b]",
		"addxy": "({x: b, ...a},{y: b, ...c})->b Num[b]",
		"getx":  "({x: b, ...a})->b",
		"sumx":  "({x: b, ...a},{x: b, ...c})->b Num[b]",
		"eqx":   "({x: b, ...a},{x: b, ...c})->bool Eq[b]",
		"addv":  "({v: b, ...a},{v: b, ...c})->{v: b, ...a} Num[b]",
	}
	for _, f := range rendered.Funcs {
		w, ok := want[f.Name]
		if !ok {
			continue
		}
		got := strings.TrimSpace(f.Type + " " + strings.Join(f.Constraints, " "))
		if got != w {
			t.Errorf("%s: got %q, want %q", f.Name, got, w)
		}
		delete(want, f.Name)
	}
	if len(want) > 0 {
		t.Errorf("exports missing from the interface: %v", want)
	}
}

// A dependency that gains a constraint must move its importer's cache key:
// the constraint is in the interface digest. Without it the importer's warm
// verdict was the stale "passes" while a cold check failed.
func TestExportedConstraints_CacheKeyMovesWithConstraint(t *testing.T) {
	cacheDir := t.TempDir()
	t.Setenv("AILANG_CACHE_DIR", cacheDir)
	t.Setenv("AILANG_NO_CACHE", "")
	dir := t.TempDir()
	write := func(rel, src string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(filepath.Join(dir, rel)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, rel), []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	const noEq = "module q/c\nexport pure func eqq[a](x: a, y: a) -> bool = true\n"
	const withEq = "module q/c\nexport pure func eqq[a](x: a, y: a) -> bool = x == y\n"
	write("q/m.ail", "module q/m\nimport q/c (eqq)\nexport func main() -> () ! {IO} = println(show(eqq(\\u. u + 1, \\u. u + 2)))\n")
	t.Chdir(dir)
	check := func() error {
		_, err := Run(Config{Mode: ModeCheck}, Source{Filename: "q/m.ail"})
		return err
	}
	write("q/c.ail", noEq)
	for i := 0; i < 2; i++ { // cold, then warm
		if err := check(); err != nil {
			t.Fatalf("no constraint: must check (run %d): %v", i, err)
		}
	}
	write("q/c.ail", withEq)
	if err := check(); err == nil || !strings.Contains(err.Error(), types.TC_CONSTRAINT_001) {
		t.Fatalf("the dependency gained Eq[a]: a warm check must fail with %s, got %v", types.TC_CONSTRAINT_001, err)
	}
	write("q/c.ail", noEq)
	if err := check(); err != nil {
		t.Fatalf("the dependency lost Eq[a]: a warm check must pass again: %v", err)
	}
}

// #2312 (#2221 part 4): defaulting found ambiguous variables with its own
// free-variable walk, which skips open records, so a constrained field under
// an open row was defaulted to int and a float use was "No instance for
// Fractional[int]". It now uses the shared walk; that waited on #2290, since
// the field's Num constraint must reach an importer.
func TestDefaulting_SharedWalk_2312(t *testing.T) {
	for name, src := range map[string]string{
		"let-bound update": `module m
export func main() -> () ! {IO} {
  let f = \p. {p | x: p.x + 1};
  println(show(f({x: 1.5, z: 1}).x))
}
`,
		"issue program": `module m
export func main() -> () ! {IO} {
  let f = \p. { let y = {p | x: p.x + 1}; y.x };
  println(show(f({x: 1.5, z: 1})))
}
`,
		"top-level, no literal": `module m
func addv(p, q) = {p | v: p.v + q.v}
export func main() -> () ! {IO} = println(show(addv({v: 1.5, z: 1}, {v: 2.0}).v))
`,
		"let-polymorphic, two types": `module m
export func main() -> () ! {IO} {
  let g = \p. \q. {p | v: p.v + q.v};
  println(show(g({v: 1.5, z: 1})({v: 2.0}).v));
  println(show(g({v: 1, z: 1})({v: 2}).v))
}
`,
	} {
		t.Run(name, func(t *testing.T) {
			if err := checkModule(t, src); err != nil {
				t.Fatalf("must check: %v", err)
			}
		})
	}
	// #2378: a module declaration's integer literal under an open record is
	// generalised like any other (the declaration has a float variant), so
	// used at float it checks; #2364 kept it defaulted, a Fractional[int] error.
	if err := checkModule(t, `module m
func g(p) = {p | x: p.x + 1}
export func main() -> () ! {IO} = println(show(g({x: 1.5, z: 1}).x))
`); err != nil {
		t.Fatalf("a top-level literal field used at float must check: %v", err)
	}
}

// #2291: std/list's contains, member, dedup, union, intersect and difference
// compare elements with ==, through builtins whose schemes had no Eq, so a
// list of functions passed check and the answer was silently false. Their
// builtins now carry Eq (BuiltinSpec.EqVars), which the std functions export.
func TestListEqConstraint_2291(t *testing.T) {
	calls := map[string]string{
		"contains":   `show(contains([\x. x + 1], \x. x + 1))`,
		"member":     `show(member(\x. x + 1, [\x. x + 1]))`,
		"dedup":      `show(length(dedup([\x. x + 1])))`,
		"union":      `show(length(union([\x. x + 1], [\x. x + 2])))`,
		"intersect":  `show(length(intersect([\x. x + 1], [\x. x + 2])))`,
		"difference": `show(length(difference([\x. x + 1], [\x. x + 2])))`,
	}
	for fn, call := range calls {
		t.Run(fn, func(t *testing.T) {
			err := checkModule(t, "module m\nimport std/list ("+fn+", length)\nexport func main() -> () ! {IO} = println("+call+")\n")
			wantCode(t, err, types.TC_CONSTRAINT_001, "`"+fn+"` from std/list requires Eq")
		})
	}
	ok := `module m
import std/list (contains, dedup, union, intersect, difference)
type Color = Red | Green deriving (Eq)
export func main() -> () ! {IO} = {
  println(show(contains([1, 2], 2)));
  println(show(dedup(["a", "a"])));
  println(show(contains([Red], Green)));
  println(show(union([(1, "x")], [(2, "y")])));
  println(show(intersect([Some(1)], [None])));
  println(show(difference([[1]], [[2]])))
}
`
	if err := checkModule(t, ok); err != nil {
		t.Fatalf("element types with Eq must check: %v", err)
	}
}

// ifaceJSONOf checks one library module and returns its normalized interface.
func ifaceJSONOf(t *testing.T, src string) *iface.InterfaceJSON {
	t.Helper()
	res, err := writeAndCheck(t, map[string]string{"q/c.ail": src}, "q/c.ail")
	if err != nil {
		t.Fatal(err)
	}
	data, err := res.Interface.ToNormalizedJSON()
	if err != nil {
		t.Fatal(err)
	}
	var j iface.InterfaceJSON
	if err := json.Unmarshal(data, &j); err != nil {
		t.Fatal(err)
	}
	return &j
}

// The registry's v2 identity (HashProjection) of a real module moves when an
// export gains a constraint with its type unchanged, and stays put for a body
// change that keeps the constraints (#2290).
func TestExportedConstraints_HashProjectionOfCheckedModule(t *testing.T) {
	project := func(src string) string {
		t.Helper()
		b, err := iface.HashProjection(ifaceJSONOf(t, src))
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	noEq := project("module q/c\nexport pure func eqq[a](x: a, y: a) -> bool = true\n")
	withEq := project("module q/c\nexport pure func eqq[a](x: a, y: a) -> bool = x == y\n")
	swapped := project("module q/c\nexport pure func eqq[a](x: a, y: a) -> bool = y == x\n")
	if noEq == withEq {
		t.Fatalf("gaining Eq[a] must change the hash projection: %s", withEq)
	}
	if withEq != swapped {
		t.Errorf("a body change that keeps the constraints must not change it:\n%s\n%s", withEq, swapped)
	}
	if !strings.Contains(withEq, `"constraints":["Eq[a]"]`) || strings.Contains(noEq, "constraints") {
		t.Errorf("projection must carry exactly the export's constraints:\n%s\n%s", noEq, withEq)
	}
}

// #2378: a module declaration's integer literal under an open record no longer
// pins the field to int (#2364's literalAtDecl): the export is generic in the
// field and carries its constraint, exactly as the same shape without a
// literal.
func TestDefaulting_LiteralUnderOpenRowExportShape(t *testing.T) {
	j := ifaceJSONOf(t, "module q/c\nexport pure func bump(p) = {p | x: p.x + 1}\nexport pure func twice(p) = {p | x: p.x + p.x}\n")
	got := map[string]string{}
	for _, f := range j.Funcs {
		got[f.Name] = strings.TrimSpace(f.Type + " " + strings.Join(f.Constraints, " "))
	}
	if want := "({x: b, ...a})->{x: b, ...a} Num[b]"; got["bump"] != want {
		t.Errorf("bump (literal under an open row) must be generic with Num[b]: got %q, want %q", got["bump"], want)
	}
	if want := "({x: b, ...a})->{x: b, ...a} Num[b]"; got["twice"] != want {
		t.Errorf("twice (no literal) must be generic with Num[b]: got %q, want %q", got["twice"], want)
	}
}
